package resolver

import (
	"cmp"
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
	for _, mod := range ctx.Modules {
		importsBySource := make(map[string][]importSymbol)
		for _, imp := range mod.Imports {
			importsBySource[imp.Module] = append(importsBySource[imp.Module], importSymbol{
				name: imp.Symbol,
				span: imp.Span,
			})
		}

		if ctx.TraceEnabled() {
			ctx.Trace("resolving imports for module",
				slog.String("module", mod.Name),
				slog.Int("sources", len(importsBySource)))
		}

		for fromModuleName, symbols := range importsBySource {
			resolveImportsFromModule(ctx, mod, fromModuleName, symbols)
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

	candidates := ctx.ModuleIndex[fromModuleName]
	if chosen, ok := findCandidateWithAllSymbols(ctx, candidates, userSymbols); ok {
		if ctx.TraceEnabled() {
			ctx.Trace("imports resolved directly",
				slog.String("from", fromModuleName),
				slog.Int("symbols", len(userSymbols)))
		}
		for _, sym := range userSymbols {
			ctx.RegisterImport(importingModule, sym.name, chosen)
		}
		return
	}

	// Module import aliases handle renamed modules (e.g., SNMPv2-SMI-v1 -> SNMPv2-SMI)
	if ctx.DiagnosticConfig().AllowSafeFallbacks() {
		if aliased := baseModuleImportAlias(fromModuleName); aliased != "" {
			aliasCandidates := ctx.ModuleIndex[aliased]
			if chosen, ok := findCandidateWithAllSymbols(ctx, aliasCandidates, userSymbols); ok {
				if ctx.TraceEnabled() {
					ctx.Trace("imports resolved via alias",
						slog.String("from", fromModuleName),
						slog.String("alias", aliased),
						slog.Int("symbols", len(userSymbols)))
				}
				for _, sym := range userSymbols {
					ctx.RegisterImport(importingModule, sym.name, chosen)
				}
				return
			}
		}
	}

	// Import forwarding: symbols re-exported through intermediate modules
	if ctx.DiagnosticConfig().AllowSafeFallbacks() && len(candidates) > 0 {
		if forwarded := tryImportForwarding(ctx, candidates, userSymbols); len(forwarded) > 0 {
			if ctx.TraceEnabled() {
				ctx.Trace("imports resolved via forwarding",
					slog.String("from", fromModuleName),
					slog.Int("forwarded", len(forwarded)))
			}
			for _, fwd := range forwarded {
				ctx.RegisterImport(importingModule, fwd.symbol, fwd.source)
			}
			return
		}
	}

	// Partial resolution: resolve symbols that are found and record
	// unresolved for the rest. Handles MIBs that import from the wrong module.
	if ctx.DiagnosticConfig().AllowSafeFallbacks() && len(candidates) > 0 {
		resolved, unresolved := tryPartialResolution(ctx, candidates, userSymbols)
		for _, res := range resolved {
			ctx.RegisterImport(importingModule, res.symbol, res.source)
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
		found := false
		for _, candidate := range candidates {
			defNames := ctx.ModuleDefNames[candidate]
			if defNames != nil {
				if _, isDirect := defNames[sym.name]; isDirect {
					resolved = append(resolved, forwardedSymbol{
						symbol: sym.name,
						source: candidate,
					})
					found = true
					break
				}
			}
		}
		if !found {
			unresolved = append(unresolved, sym)
		}
	}

	return resolved, unresolved
}

func tryImportForwarding(ctx *resolverContext, candidates []*module.Module, symbols []importSymbol) []forwardedSymbol {
	for _, candidate := range candidates {
		defNames := ctx.ModuleDefNames[candidate]
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
			sourceCandidates := ctx.ModuleIndex[sourceModuleName]
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
		defNames := ctx.ModuleDefNames[candidate]
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

	for _, cand := range scoredCandidates {
		if cand.symbolCount == totalSymbols {
			return cand.mod, true
		}
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
func normalizeTimestamp(ts string) string {
	const smiv1TimestampLen = 10
	trimmed := strings.TrimSuffix(ts, "Z")
	if len(trimmed) == smiv1TimestampLen {
		yy := trimmed[:2]
		century := "20"
		if yy >= "70" {
			century = "19"
		}
		return century + trimmed + "Z"
	}
	return ts
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
	for _, imports := range ctx.ModuleImports {
		for symbol, sourceMod := range imports {
			ultimate := resolveUltimateDefiner(ctx, sourceMod, symbol)
			if ultimate != sourceMod {
				imports[symbol] = ultimate
			}
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

		if defNames := ctx.ModuleDefNames[current]; defNames != nil {
			if _, ok := defNames[symbol]; ok {
				return current
			}
		}

		if nextImports := ctx.ModuleImports[current]; nextImports != nil {
			if next, ok := nextImports[symbol]; ok {
				current = next
				continue
			}
		}

		return current
	}
}

func baseModuleImportAlias(name string) string {
	switch name {
	case "SNMPv2-SMI-v1":
		return "SNMPv2-SMI"
	case "SNMPv2-TC-v1":
		return "SNMPv2-TC"
	case "RFC1315-MIB":
		return "FRAME-RELAY-DTE-MIB"
	case "RFC-1213":
		return "RFC1213-MIB"
	default:
		return ""
	}
}
