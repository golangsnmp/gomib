package mib

import (
	"cmp"
	"fmt"
	"log/slog"
	"slices"
	"strings"

	"github.com/golangsnmp/gomib/internal/module"
	"github.com/golangsnmp/gomib/internal/types"
)

type importSymbol struct {
	name string
	span types.Span
}

// resolveImports is the import resolution phase entry point.
func resolveImports(ctx *resolverContext) {
	for _, mod := range ctx.modules {
		// Group imports by source module, preserving the order in which
		// source modules first appear. Symbols that appear in multiple
		// groups (e.g., DisplayString imported from both RFC1213-MIB and
		// SNMPv2-TC) are kept only in the first group, so that iteration
		// order is deterministic and matches the MIB's import ordering.
		type importGroup struct {
			module  string
			symbols []importSymbol
		}
		var groups []importGroup
		groupIndex := make(map[string]int)
		seenSymbols := make(map[string]string) // symbol -> first source module

		for _, imp := range mod.Imports {
			if firstMod, dup := seenSymbols[imp.Symbol]; dup {
				if imp.Module != firstMod {
					ctx.EmitDiagnostic(types.DiagImportDuplicate, mod, imp.Span,
						fmt.Sprintf("duplicate import: %q already imported from %q, ignoring import from %q",
							imp.Symbol, firstMod, imp.Module))
				}
				continue
			}
			seenSymbols[imp.Symbol] = imp.Module

			idx, exists := groupIndex[imp.Module]
			if !exists {
				idx = len(groups)
				groupIndex[imp.Module] = idx
				groups = append(groups, importGroup{module: imp.Module})
			}
			groups[idx].symbols = append(groups[idx].symbols, importSymbol{
				name: imp.Symbol,
				span: imp.Span,
			})
		}

		if ctx.TraceEnabled() {
			ctx.Trace("resolving imports for module",
				slog.String("module", mod.Name),
				slog.Int("sources", len(groups)))
		}

		for _, g := range groups {
			resolveImportsFromModule(ctx, mod, g.module, g.symbols)
		}
	}
}

func resolveImportsFromModule(ctx *resolverContext, importingModule *module.Module, fromModuleName string, symbols []importSymbol) {
	var userSymbols []importSymbol
	for _, sym := range symbols {
		if isMacroSymbol(sym.name) {
			continue
		}
		userSymbols = append(userSymbols, sym)
	}
	if len(userSymbols) == 0 {
		return
	}

	candidates := ctx.moduleIndex[fromModuleName]
	if chosen, ok := findCandidateWithAllSymbols(ctx, candidates, userSymbols); ok {
		if ctx.TraceEnabled() {
			ctx.Trace("imports resolved directly",
				slog.String("from", fromModuleName),
				slog.Int("symbols", len(userSymbols)))
		}
		for _, sym := range userSymbols {
			ctx.registerImport(importingModule, sym.name, chosen)
		}
		return
	}

	// Module import aliases are constrained fallbacks (Normal+).
	if ctx.ResolverStrictness().AllowConstrainedFallbacks() {
		if aliased := baseModuleImportAlias(fromModuleName); aliased != "" {
			aliasCandidates := ctx.moduleIndex[aliased]
			if chosen, ok := findCandidateWithAllSymbols(ctx, aliasCandidates, userSymbols); ok {
				if ctx.TraceEnabled() {
					ctx.Trace("imports resolved via alias",
						slog.String("from", fromModuleName),
						slog.String("alias", aliased),
						slog.Int("symbols", len(userSymbols)))
				}
				for _, sym := range userSymbols {
					ctx.registerImport(importingModule, sym.name, chosen)
				}
				return
			}
		}
	}

	// Deterministic: symbols re-exported through explicit intermediate imports.
	if len(candidates) > 0 {
		if forwarded := tryImportForwarding(ctx, candidates, userSymbols); len(forwarded) > 0 {
			if ctx.TraceEnabled() {
				ctx.Trace("imports resolved via forwarding",
					slog.String("from", fromModuleName),
					slog.Int("forwarded", len(forwarded)))
			}
			for _, fwd := range forwarded {
				ctx.registerImport(importingModule, fwd.symbol, fwd.source)
			}
			return
		}
	}

	// Deterministic per symbol: resolve symbols that are reachable from the
	// declared source module and leave the rest unresolved.
	if len(candidates) > 0 {
		resolved, unresolved := tryPartialResolution(ctx, candidates, userSymbols)
		for _, res := range resolved {
			ctx.registerImport(importingModule, res.symbol, res.source)
		}
		if ctx.TraceEnabled() && len(resolved) > 0 {
			ctx.Trace("imports partially resolved",
				slog.String("from", fromModuleName),
				slog.Int("resolved", len(resolved)),
				slog.Int("unresolved", len(unresolved)))
		}
		for _, sym := range unresolved {
			ctx.RecordUnresolvedImport(importingModule, fromModuleName, sym.name, reasonSymbolNotExported, sym.span)
		}
		return
	}

	if ctx.TraceEnabled() {
		ctx.Trace("imports unresolved",
			slog.String("from", fromModuleName),
			slog.Int("symbols", len(userSymbols)),
			slog.String("reason", reasonModuleNotFound))
	}

	for _, sym := range userSymbols {
		ctx.RecordUnresolvedImport(importingModule, fromModuleName, sym.name, reasonModuleNotFound, sym.span)
	}
}

type forwardedSymbol struct {
	symbol string
	source *module.Module
}

// tryPartialResolution resolves as many symbols as possible from the
// candidates, returning resolved and unresolved symbols separately.
func tryPartialResolution(ctx *resolverContext, candidates []*module.Module, symbols []importSymbol) ([]forwardedSymbol, []importSymbol) {
	var resolved []forwardedSymbol
	var unresolved []importSymbol

	for _, sym := range symbols {
		if source, ok := resolveImportedSymbol(ctx, candidates, sym.name); ok {
			resolved = append(resolved, forwardedSymbol{
				symbol: sym.name,
				source: source,
			})
		} else {
			unresolved = append(unresolved, sym)
		}
	}

	return resolved, unresolved
}

func resolveImportedSymbol(ctx *resolverContext, candidates []*module.Module, symbol string) (*module.Module, bool) {
	for _, candidate := range candidates {
		defNames := ctx.moduleDefNames[candidate]
		if defNames != nil {
			if _, isDirect := defNames[symbol]; isDirect {
				return candidate, true
			}
		}

		for _, imp := range candidate.Imports {
			if imp.Symbol != symbol {
				continue
			}
			sourceCandidates := ctx.moduleIndex[imp.Module]
			if len(sourceCandidates) > 0 {
				return bestCandidate(sourceCandidates), true
			}
			break
		}
	}

	return nil, false
}

func tryImportForwarding(ctx *resolverContext, candidates []*module.Module, symbols []importSymbol) []forwardedSymbol {
	for _, candidate := range candidates {
		defNames := ctx.moduleDefNames[candidate]
		importMap := make(map[string]string)
		for _, imp := range candidate.Imports {
			importMap[imp.Symbol] = imp.Module
		}

		forwarded := make([]forwardedSymbol, 0, len(symbols))
		allFound := true
		for _, sym := range symbols {
			if defNames != nil {
				if _, isDirect := defNames[sym.name]; isDirect {
					forwarded = append(forwarded, forwardedSymbol{
						symbol: sym.name,
						source: candidate,
					})
					continue
				}
			}
			// Not directly defined, check if re-exported via imports
			sourceModuleName, ok := importMap[sym.name]
			if !ok {
				allFound = false
				break
			}
			sourceCandidates := ctx.moduleIndex[sourceModuleName]
			if len(sourceCandidates) == 0 {
				allFound = false
				break
			}
			forwarded = append(forwarded, forwardedSymbol{
				symbol: sym.name,
				source: bestCandidate(sourceCandidates),
			})
		}
		if allFound && len(forwarded) > 0 {
			return forwarded
		}
	}
	return nil
}

func findCandidateWithAllSymbols(ctx *resolverContext, candidates []*module.Module, symbols []importSymbol) (*module.Module, bool) {
	if len(candidates) == 0 {
		return nil, false
	}

	type scored struct {
		mod         *module.Module
		symbolCount int
		lastUpdated string
	}

	scoredCandidates := make([]scored, 0, len(candidates))
	totalSymbols := len(symbols)

	for _, candidate := range candidates {
		defNames := ctx.moduleDefNames[candidate]
		if defNames == nil {
			continue
		}

		count := 0
		for _, sym := range symbols {
			if _, ok := defNames[sym.name]; ok {
				count++
			}
		}

		scoredCandidates = append(scoredCandidates, scored{
			mod:         candidate,
			symbolCount: count,
			lastUpdated: extractLastUpdated(candidate),
		})
	}

	slices.SortFunc(scoredCandidates, func(a, b scored) int {
		// Prefer more matching symbols, then newer LAST-UPDATED
		if c := cmp.Compare(b.symbolCount, a.symbolCount); c != 0 {
			return c
		}
		return cmp.Compare(b.lastUpdated, a.lastUpdated)
	})

	if len(scoredCandidates) > 0 && scoredCandidates[0].symbolCount == totalSymbols {
		return scoredCandidates[0].mod, true
	}

	return nil, false
}

// bestCandidate picks the module with the newest LAST-UPDATED timestamp.
// Falls back to the first candidate when no timestamps are present.
func bestCandidate(candidates []*module.Module) *module.Module {
	best := candidates[0]
	bestTS := extractLastUpdated(best)
	for _, c := range candidates[1:] {
		ts := extractLastUpdated(c)
		if ts > bestTS {
			best = c
			bestTS = ts
		}
	}
	return best
}

func extractLastUpdated(mod *module.Module) string {
	for _, def := range mod.Definitions {
		if mi, ok := def.(*module.ModuleIdentity); ok {
			if strings.TrimSpace(mi.LastUpdated) != "" {
				return normalizeTimestamp(mi.LastUpdated)
			}
		}
	}
	return ""
}

// normalizeTimestamp converts SMI LAST-UPDATED timestamps to a sortable form.
// SMIv1 uses 10-digit format "YYMMDDHHmmZ" (2-digit year), SMIv2 uses
// 12-digit "YYYYMMDDHHmmZ" (4-digit year). This expands 10-digit timestamps
// to 12-digit by prepending "19" for years >= 70, "20" otherwise.
// Both formats are normalized to include a trailing "Z".
func normalizeTimestamp(ts string) string {
	const smiv1TimestampLen = 10
	trimmed := strings.TrimSuffix(ts, "Z")
	if len(trimmed) == smiv1TimestampLen {
		yy := trimmed[:2]
		century := "20"
		if yy >= "70" {
			century = "19"
		}
		trimmed = century + trimmed
	}
	if trimmed == "" {
		return ts
	}
	return trimmed + "Z"
}

func isMacroSymbol(name string) bool {
	switch name {
	case "MODULE-IDENTITY", "OBJECT-IDENTITY", "OBJECT-TYPE",
		"NOTIFICATION-TYPE", "TEXTUAL-CONVENTION", "OBJECT-GROUP",
		"NOTIFICATION-GROUP", "MODULE-COMPLIANCE", "AGENT-CAPABILITIES",
		"TRAP-TYPE":
		return true
	default:
		return false
	}
}

// resolveTransitiveImports follows each import entry to the module that
// actually defines the symbol, collapsing re-export chains. After this,
// ModuleImports[mod][symbol] points directly to the defining module.
func resolveTransitiveImports(ctx *resolverContext) {
	for _, imports := range ctx.moduleImports {
		type update struct {
			symbol  string
			definer *module.Module
		}
		var updates []update
		for symbol, sourceMod := range imports {
			ultimate := resolveUltimateDefiner(ctx, sourceMod, symbol)
			if ultimate != sourceMod {
				updates = append(updates, update{symbol, ultimate})
			}
		}
		for _, u := range updates {
			imports[u.symbol] = u.definer
		}
	}
}

// resolveUltimateDefiner follows import chains from mod to find the module
// that actually defines symbol (has it in ModuleDefNames).
func resolveUltimateDefiner(ctx *resolverContext, mod *module.Module, symbol string) *module.Module {
	visited := make(map[*module.Module]struct{}, 4)
	current := mod
	for {
		if _, seen := visited[current]; seen {
			return current
		}
		visited[current] = struct{}{}

		if defNames := ctx.moduleDefNames[current]; defNames != nil {
			if _, ok := defNames[symbol]; ok {
				return current
			}
		}

		if nextImports := ctx.moduleImports[current]; nextImports != nil {
			if next, ok := nextImports[symbol]; ok {
				current = next
				continue
			}
		}

		return current
	}
}

// checkUnusedImports emits diagnostics for imported symbols that were never
// referenced during resolution. Must be called after all resolution phases.
func checkUnusedImports(ctx *resolverContext) {
	for _, mod := range ctx.modules {
		if module.IsBaseModule(mod.Name) {
			continue
		}
		used := ctx.usedImports[mod]
		imports := ctx.moduleImports[mod]
		for _, imp := range mod.Imports {
			if isMacroSymbol(imp.Symbol) {
				continue
			}
			// Skip imports that failed to resolve; already reported as
			// import-not-found or import-module-not-found.
			if imports == nil {
				continue
			}
			if _, resolved := imports[imp.Symbol]; !resolved {
				continue
			}
			if used != nil {
				if _, ok := used[imp.Symbol]; ok {
					continue
				}
			}
			ctx.EmitDiagnostic(types.DiagImportUnused, mod, imp.Span,
				fmt.Sprintf("unused import: %q from %q", imp.Symbol, imp.Module))
		}
	}
}

// copyUsedImportsToModules transfers the resolver's usedImports tracking data
// to each resolved Module so it's available via Module.IsImportUsed().
func copyUsedImportsToModules(ctx *resolverContext) {
	for mod, used := range ctx.usedImports {
		if resolved := ctx.moduleToResolved[mod]; resolved != nil && len(used) > 0 {
			resolved.setUsedImportNames(used)
		}
	}
}

// copyResolvedImportsToModules transfers the resolver's moduleImports data
// to each resolved Module so it's available via Module.ImportSource() and
// Module.AvailableSymbols().
func copyResolvedImportsToModules(ctx *resolverContext) {
	for mod, imports := range ctx.moduleImports {
		resolved := ctx.moduleToResolved[mod]
		if resolved == nil || len(imports) == 0 {
			continue
		}
		resolvedMap := make(map[string]*Module, len(imports))
		for symbol, sourceMod := range imports {
			if sourceResolved := ctx.moduleToResolved[sourceMod]; sourceResolved != nil {
				resolvedMap[symbol] = sourceResolved
			}
		}
		if len(resolvedMap) > 0 {
			resolved.setResolvedImports(resolvedMap)
		}
	}
}

type moduleSymbol struct{ module, symbol string }

// obsoleteImportTarget describes the preferred SMIv2 module and symbol name
// for an import that should no longer come from an obsolete SMIv1 module.
// Derived from libsmi's convertImportv2 table.
type obsoleteImportTarget struct {
	newModule, newSymbol string
}

// obsoleteImports maps (old module, old symbol) pairs to preferred targets.
// RFC1155-SMI and RFC1065-SMI share the same symbol set.
var obsoleteImports = func() map[moduleSymbol]obsoleteImportTarget {
	// Symbols shared by RFC1155-SMI and RFC1065-SMI.
	smiSymbols := []struct{ old, new string }{
		{"internet", "internet"},
		{"directory", "directory"},
		{"mgmt", "mgmt"},
		{"experimental", "experimental"},
		{"private", "private"},
		{"enterprises", "enterprises"},
		{"IpAddress", "IpAddress"},
		{"Counter", "Counter32"},
		{"Gauge", "Gauge32"},
		{"TimeTicks", "TimeTicks"},
		{"Opaque", "Opaque"},
	}
	m := make(map[moduleSymbol]obsoleteImportTarget, len(smiSymbols)*2+2)
	for _, mod := range []string{"RFC1155-SMI", "RFC1065-SMI"} {
		for _, sym := range smiSymbols {
			m[moduleSymbol{mod, sym.old}] = obsoleteImportTarget{moduleSNMPv2SMI, sym.new}
		}
	}
	m[moduleSymbol{"RFC1213-MIB", "mib-2"}] = obsoleteImportTarget{moduleSNMPv2SMI, "mib-2"}
	m[moduleSymbol{"RFC1213-MIB", "DisplayString"}] = obsoleteImportTarget{moduleSNMPv2TC, "DisplayString"}
	return m
}()

// checkObsoleteImports flags SMIv2 modules importing symbols from obsolete
// SMIv1 modules when newer SMIv2 equivalents exist.
func checkObsoleteImports(ctx *resolverContext) {
	for _, mod := range ctx.modules {
		if module.IsBaseModule(mod.Name) {
			continue
		}
		if mod.Language != types.LanguageSMIv2 {
			continue
		}
		for _, imp := range mod.Imports {
			target, ok := obsoleteImports[moduleSymbol{imp.Module, imp.Symbol}]
			if !ok {
				continue
			}
			msg := fmt.Sprintf("%q should be imported from %s instead of %s",
				imp.Symbol, target.newModule, imp.Module)
			if imp.Symbol != target.newSymbol {
				msg = fmt.Sprintf("%q should be imported as %q from %s instead of %s",
					imp.Symbol, target.newSymbol, target.newModule, imp.Module)
			}
			ctx.EmitDiagnostic(types.DiagObsoleteImport, mod, imp.Span, msg)
		}
	}
}

func baseModuleImportAlias(name string) string {
	switch name {
	case "SNMPv2-SMI-v1":
		return moduleSNMPv2SMI
	case "SNMPv2-TC-v1":
		return moduleSNMPv2TC
	case "RFC1315-MIB":
		return "FRAME-RELAY-DTE-MIB"
	case "RFC-1213":
		return "RFC1213-MIB"
	default:
		return ""
	}
}
