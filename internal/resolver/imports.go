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

// resolveImports resolves all imports across all modules.
func resolveImports(ctx *ResolverContext) {
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

func resolveImportsFromModule(ctx *ResolverContext, importingModule *module.Module, fromModuleName string, symbols []importSymbol) {
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

	if len(candidates) > 0 {
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

	// Try partial resolution - resolve symbols that are found, record unresolved for others.
	// This handles real-world MIBs that import from the "wrong" module.
	if len(candidates) > 0 {
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
			ctx.RecordUnresolvedImport(importingModule, fromModuleName, sym.name, "symbol_not_exported", sym.span)
		}
		return
	}

	// Module not found at all
	if ctx.TraceEnabled() {
		ctx.Trace("imports unresolved",
			slog.String("from", fromModuleName),
			slog.Int("symbols", len(userSymbols)),
			slog.String("reason", "module_not_found"))
	}

	for _, sym := range userSymbols {
		ctx.RecordUnresolvedImport(importingModule, fromModuleName, sym.name, "module_not_found", sym.span)
	}
}

type forwardedSymbol struct {
	symbol string
	source *module.Module
}

// tryPartialResolution tries to resolve as many symbols as possible from the candidates.
// Returns resolved symbols and unresolved symbols separately.
func tryPartialResolution(ctx *ResolverContext, candidates []*module.Module, symbols []importSymbol) ([]forwardedSymbol, []importSymbol) {
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

func tryImportForwarding(ctx *ResolverContext, candidates []*module.Module, symbols []importSymbol) []forwardedSymbol {
	for _, candidate := range candidates {
		defNames := ctx.ModuleDefNames[candidate]
		importMap := make(map[string]string)
		for _, imp := range candidate.Imports {
			importMap[imp.Symbol] = imp.Module
		}

		forwarded := make([]forwardedSymbol, 0, len(symbols))
		allFound := true
		for _, sym := range symbols {
			// First check if directly defined in candidate
			if defNames != nil {
				if _, isDirect := defNames[sym.name]; isDirect {
					forwarded = append(forwarded, forwardedSymbol{
						symbol: sym.name,
						source: candidate,
					})
					continue
				}
			}
			// Otherwise check if imported (re-exported)
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
				source: sourceCandidates[0],
			})
		}
		if allFound && len(forwarded) > 0 {
			return forwarded
		}
	}
	return nil
}

func findCandidateWithAllSymbols(ctx *ResolverContext, candidates []*module.Module, symbols []importSymbol) (*module.Module, bool) {
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
		// Sort by symbol count descending, then by lastUpdated descending
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

func normalizeTimestamp(ts string) string {
	trimmed := strings.TrimSuffix(ts, "Z")
	if len(trimmed) == 10 {
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
