package resolver

import (
	"testing"

	"github.com/golangsnmp/gomib/internal/model"
	"github.com/golangsnmp/gomib/internal/module"
	"github.com/golangsnmp/gomib/internal/testutil"
	"github.com/golangsnmp/gomib/internal/types"
)

func TestIsMacroSymbol(t *testing.T) {
	macros := []string{
		"MODULE-IDENTITY",
		"OBJECT-IDENTITY",
		"OBJECT-TYPE",
		"NOTIFICATION-TYPE",
		"TEXTUAL-CONVENTION",
		"OBJECT-GROUP",
		"NOTIFICATION-GROUP",
		"MODULE-COMPLIANCE",
		"AGENT-CAPABILITIES",
		"TRAP-TYPE",
	}
	for _, name := range macros {
		testutil.True(t, isMacroSymbol(name), "isMacroSymbol() = false, want true")
	}

	nonMacros := []string{
		"sysDescr",
		"Counter32",
		"DisplayString",
		"enterprises",
		"",
		"OBJECT-TYPE-EXTRA",
		"module-identity",
	}
	for _, name := range nonMacros {
		testutil.False(t, isMacroSymbol(name), "isMacroSymbol(%q) = true, want false", name)
	}
}

func TestBaseModuleImportAlias(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"SNMPv2-SMI-v1", "SNMPv2-SMI"},
		{"SNMPv2-TC-v1", "SNMPv2-TC"},
		{"RFC1315-MIB", "FRAME-RELAY-DTE-MIB"},
		{"RFC-1213", "RFC1213-MIB"},
		{"SNMPv2-SMI", ""},
		{"UNKNOWN-MIB", ""},
		{"", ""},
	}
	for _, tt := range tests {
		got := baseModuleImportAlias(tt.input)
		testutil.Equal(t, tt.want, got, "baseModuleImportAlias()")
	}
}

func TestNormalizeTimestamp(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		// 10-digit with century < 70 -> 20xx
		{"0210180000Z", "200210180000Z"},
		{"6912310000Z", "206912310000Z"},
		// 10-digit with century >= 70 -> 19xx
		{"7001010000Z", "197001010000Z"},
		{"9905270000Z", "199905270000Z"},
		// Already 12-digit, returned as-is
		{"200210180000Z", "200210180000Z"},
		{"199905270000Z", "199905270000Z"},
		// Invalid values do not enter timestamp ordering
		{"", ""},
		{"Z", ""},
		{"200210180000", ""},
		{"0210180000", ""},
		{"200213180000Z", ""},
		{"200202300000Z", ""},
		{"é12345678Z", ""},
	}
	for _, tt := range tests {
		got := normalizeTimestamp(tt.input)
		testutil.Equal(t, tt.want, got, "normalizeTimestamp()")
	}
}

func TestExtractLastUpdated(t *testing.T) {
	t.Run("module with ModuleIdentity", func(t *testing.T) {
		mod := &module.Module{
			Name:        "TEST-MIB",
			LastUpdated: "0210180000Z",
		}
		addDefs(mod, []module.Definition{
			&module.ModuleIdentity{
				DefBase:     module.DefBase{Name: "testMIB"},
				LastUpdated: "0210180000Z",
			},
		})
		got := extractLastUpdated(mod)
		testutil.Equal(t, "200210180000Z", got, "extractLastUpdated()")
	})

	t.Run("module without ModuleIdentity", func(t *testing.T) {
		mod := &module.Module{
			Name: "TEST-MIB",
		}
		addDefs(mod, []module.Definition{
			&module.ObjectType{DefBase: module.DefBase{Name: "testObject"}},
		})
		got := extractLastUpdated(mod)
		testutil.Equal(t, "", got, "extractLastUpdated()")
	})

	t.Run("module with empty LastUpdated", func(t *testing.T) {
		mod := &module.Module{
			Name: "TEST-MIB",
		}
		addDefs(mod, []module.Definition{
			&module.ModuleIdentity{
				DefBase:     module.DefBase{Name: "testMIB"},
				LastUpdated: "   ",
			},
		})
		got := extractLastUpdated(mod)
		testutil.Equal(t, "", got, "extractLastUpdated()")
	})

	t.Run("no definitions", func(t *testing.T) {
		mod := &module.Module{Name: "EMPTY-MIB"}
		got := extractLastUpdated(mod)
		testutil.Equal(t, "", got, "extractLastUpdated()")
	})
}

// makeTestModule creates a module with the given definitions registered in
// the context's ModuleDefNames index.
func makeTestModule(ctx *resolverContext, name string, defNames []string) *module.Module {
	mod := &module.Module{Name: name}
	defs := make(map[string]struct{}, len(defNames))
	for _, n := range defNames {
		defs[n] = struct{}{}
	}
	ctx.defNames[mod] = defs
	return mod
}

func TestFindCandidateWithAllSymbols(t *testing.T) {
	syms := func(names ...string) []importSymbol {
		out := make([]importSymbol, len(names))
		for i, n := range names {
			out[i] = importSymbol{name: n}
		}
		return out
	}

	t.Run("no candidates", func(t *testing.T) {
		ctx := newTestContext()
		_, ok := findCandidateWithAllSymbols(ctx, nil, syms("foo"))
		testutil.False(t, ok, "expected no match with empty candidates")
	})

	t.Run("single candidate with all symbols", func(t *testing.T) {
		ctx := newTestContext()
		mod := makeTestModule(ctx, "MOD-A", []string{"foo", "bar"})
		got, ok := findCandidateWithAllSymbols(ctx, []*module.Module{mod}, syms("foo", "bar"))
		testutil.True(t, ok, "expected match")
		testutil.Equal(t, mod, got, "expected MOD-A")
	})

	t.Run("single candidate missing symbols", func(t *testing.T) {
		ctx := newTestContext()
		mod := makeTestModule(ctx, "MOD-A", []string{"foo"})
		_, ok := findCandidateWithAllSymbols(ctx, []*module.Module{mod}, syms("foo", "bar"))
		testutil.False(t, ok, "expected no match when candidate is missing symbols")
	})

	t.Run("multiple candidates, pick one with all symbols", func(t *testing.T) {
		ctx := newTestContext()
		modA := makeTestModule(ctx, "MOD-A", []string{"foo"})
		modB := makeTestModule(ctx, "MOD-B", []string{"foo", "bar", "baz"})
		got, ok := findCandidateWithAllSymbols(ctx, []*module.Module{modA, modB}, syms("foo", "bar"))
		testutil.True(t, ok, "expected match")
		testutil.Equal(t, modB, got, "expected MOD-B, got")
	})

	t.Run("tiebreak by LAST-UPDATED, prefer newer", func(t *testing.T) {
		ctx := newTestContext()

		modOld := &module.Module{
			Name:        "MOD-OLD",
			LastUpdated: "9901010000Z",
		}
		addDefs(modOld, []module.Definition{
			&module.ModuleIdentity{
				DefBase:     module.DefBase{Name: "modOld"},
				LastUpdated: "9901010000Z",
			},
		})
		ctx.defNames[modOld] = map[string]struct{}{
			"foo": {},
			"bar": {},
		}

		modNew := &module.Module{
			Name:        "MOD-NEW",
			LastUpdated: "200501010000Z",
		}
		addDefs(modNew, []module.Definition{
			&module.ModuleIdentity{
				DefBase:     module.DefBase{Name: "modNew"},
				LastUpdated: "200501010000Z",
			},
		})
		ctx.defNames[modNew] = map[string]struct{}{
			"foo": {},
			"bar": {},
		}

		got, ok := findCandidateWithAllSymbols(ctx,
			[]*module.Module{modOld, modNew},
			syms("foo", "bar"))
		testutil.True(t, ok, "expected match")
		testutil.Equal(t, modNew, got, "expected MOD-NEW (newer), got")
	})

	t.Run("malformed multibyte timestamps preserve candidate order", func(t *testing.T) {
		ctx := newTestContext()
		first := makeTestModule(ctx, "FIRST-MIB", []string{"foo"})
		first.LastUpdated = "12345678éZ"
		second := makeTestModule(ctx, "SECOND-MIB", []string{"foo"})
		second.LastUpdated = "é12345678Z"

		got, ok := findCandidateWithAllSymbols(ctx, []*module.Module{first, second}, syms("foo"))
		testutil.True(t, ok, "expected match")
		testutil.Equal(t, first, got, "invalid timestamps should use candidate order")
	})

	t.Run("candidate with nil defNames is skipped", func(t *testing.T) {
		ctx := newTestContext()
		modNil := &module.Module{Name: "MOD-NIL"}
		// no defNames registered for modNil
		modGood := makeTestModule(ctx, "MOD-GOOD", []string{"x"})
		got, ok := findCandidateWithAllSymbols(ctx, []*module.Module{modNil, modGood}, syms("x"))
		testutil.True(t, ok, "expected match")
		testutil.Equal(t, modGood, got, "expected MOD-GOOD, got")
	})

	t.Run("no candidate has all symbols", func(t *testing.T) {
		ctx := newTestContext()
		modA := makeTestModule(ctx, "A", []string{"x"})
		modB := makeTestModule(ctx, "B", []string{"y"})
		_, ok := findCandidateWithAllSymbols(ctx, []*module.Module{modA, modB}, syms("x", "y"))
		testutil.False(t, ok, "expected no match when no single candidate has all symbols")
	})
}

func TestResolveImportsFromModule(t *testing.T) {
	syms := func(names ...string) []importSymbol {
		out := make([]importSymbol, len(names))
		for i, n := range names {
			out[i] = importSymbol{name: n}
		}
		return out
	}

	t.Run("macro-only imports are skipped", func(t *testing.T) {
		ctx := newTestContext()
		importing := &module.Module{Name: "IMPORTER"}
		resolveImportsFromModule(ctx, importing, "SNMPv2-SMI",
			syms("MODULE-IDENTITY", "OBJECT-TYPE"))
		// No imports registered, no unresolved recorded
		testutil.Equal(t, 0, len(ctx.importSources[importing]), "expected no imports for macro-only symbols")
		testutil.Len(t, unresolvedByKind(ctx, model.UnresolvedImport), 0, "expected no unresolved imports for macro-only symbols")
	})

	t.Run("direct resolution", func(t *testing.T) {
		ctx := newTestContext()
		source := makeTestModule(ctx, "SOURCE-MIB", []string{"sysDescr", "sysName"})
		ctx.moduleIndex["SOURCE-MIB"] = []*module.Module{source}

		importing := &module.Module{Name: "IMPORTER"}
		resolveImportsFromModule(ctx, importing, "SOURCE-MIB",
			syms("sysDescr", "sysName"))

		imports := ctx.importSources[importing]
		testutil.Equal(t, 2, len(imports), "import count")
		testutil.Equal(t, source, imports["sysDescr"], "sysDescr should resolve to SOURCE-MIB")
		testutil.Equal(t, source, imports["sysName"], "sysName should resolve to SOURCE-MIB")
	})

	t.Run("macros filtered, non-macros resolved", func(t *testing.T) {
		ctx := newTestContext()
		source := makeTestModule(ctx, "SOURCE-MIB", []string{"sysDescr"})
		ctx.moduleIndex["SOURCE-MIB"] = []*module.Module{source}

		importing := &module.Module{Name: "IMPORTER"}
		resolveImportsFromModule(ctx, importing, "SOURCE-MIB",
			syms("OBJECT-TYPE", "sysDescr"))

		imports := ctx.importSources[importing]
		testutil.Equal(t, 1, len(imports), "import count")
		testutil.Equal(t, source, imports["sysDescr"], "sysDescr should resolve to SOURCE-MIB")
	})

	t.Run("alias resolution", func(t *testing.T) {
		ctx := newTestContextWithConfig(model.DefaultConfig())
		// Source module is under the canonical name
		source := makeTestModule(ctx, "SNMPv2-SMI", []string{"enterprises", "Counter32"})
		ctx.moduleIndex["SNMPv2-SMI"] = []*module.Module{source}
		// Import uses the alias name (no candidates under alias)

		importing := &module.Module{Name: "IMPORTER"}
		resolveImportsFromModule(ctx, importing, "SNMPv2-SMI-v1",
			syms("enterprises", "Counter32"))

		imports := ctx.importSources[importing]
		testutil.Equal(t, 2, len(imports), "import count")
		testutil.Equal(t, source, imports["enterprises"], "enterprises should resolve via alias to SNMPv2-SMI")
	})

	t.Run("alias disabled in strict mode", func(t *testing.T) {
		ctx := newTestContextWithPolicy(model.ResolverStrict, model.VerboseConfig())
		source := makeTestModule(ctx, "SNMPv2-SMI", []string{"enterprises"})
		ctx.moduleIndex["SNMPv2-SMI"] = []*module.Module{source}

		importing := &module.Module{Name: "IMPORTER"}
		resolveImportsFromModule(ctx, importing, "SNMPv2-SMI-v1",
			syms("enterprises"))

		testutil.Equal(t, 0, len(ctx.importSources[importing]), "alias should not resolve in strict mode")
		testutil.Equal(t, 1, len(unresolvedByKind(ctx, model.UnresolvedImport)), "strict should record unresolved aliased import")
	})

	t.Run("forwarding resolution", func(t *testing.T) {
		ctx := newTestContextWithConfig(model.DefaultConfig())

		// Ultimate source
		realSource := makeTestModule(ctx, "REAL-SOURCE", []string{"remoteSym"})
		ctx.moduleIndex["REAL-SOURCE"] = []*module.Module{realSource}

		// Intermediate module that defines "localDef" and re-exports "remoteSym"
		intermediate := &module.Module{
			Name: "INTERMEDIATE",
			Imports: []module.Import{
				{Module: "REAL-SOURCE", Symbol: "remoteSym"},
			},
		}
		ctx.defNames[intermediate] = map[string]struct{}{
			"localDef": {},
		}
		ctx.moduleIndex["INTERMEDIATE"] = []*module.Module{intermediate}

		importing := &module.Module{Name: "IMPORTER"}
		resolveImportsFromModule(ctx, importing, "INTERMEDIATE",
			syms("localDef", "remoteSym"))

		imports := ctx.importSources[importing]
		testutil.Equal(t, 2, len(imports), "import count")
		testutil.Equal(t, intermediate, imports["localDef"], "localDef should come from INTERMEDIATE")
		testutil.Equal(t, realSource, imports["remoteSym"], "remoteSym should be forwarded from REAL-SOURCE")
	})

	t.Run("partial resolution", func(t *testing.T) {
		ctx := newTestContextWithConfig(model.DefaultConfig())

		// Candidate module only has some of the symbols
		source := makeTestModule(ctx, "PARTIAL-MIB", []string{"found1", "found2"})
		ctx.moduleIndex["PARTIAL-MIB"] = []*module.Module{source}

		importing := &module.Module{Name: "IMPORTER"}
		resolveImportsFromModule(ctx, importing, "PARTIAL-MIB",
			syms("found1", "found2", "missing1"))

		imports := ctx.importSources[importing]
		testutil.Equal(t, 2, len(imports), "import count")
		testutil.Equal(t, source, imports["found1"], "found1 should resolve to PARTIAL-MIB")
		testutil.Equal(t, source, imports["found2"], "found2 should resolve to PARTIAL-MIB")
		unresImports := unresolvedByKind(ctx, model.UnresolvedImport)
		testutil.Len(t, unresImports, 1, "unresolved count")
		testutil.Equal(t, "missing1", unresImports[0].Symbol, "unresolved symbol")
		testutil.Equal(t, reasonSymbolNotExported, unresImports[0].Reason, "unresolved reason")
	})

	t.Run("partial resolution keeps forwarded symbols", func(t *testing.T) {
		ctx := newTestContextWithConfig(model.DefaultConfig())

		realSource := makeTestModule(ctx, "REAL-SOURCE", []string{"forwarded"})
		ctx.moduleIndex["REAL-SOURCE"] = []*module.Module{realSource}

		source := &module.Module{
			Name: "PARTIAL-FWD-MIB",
			Imports: []module.Import{
				{Module: "REAL-SOURCE", Symbol: "forwarded"},
			},
		}
		ctx.defNames[source] = map[string]struct{}{
			"local": {},
		}
		ctx.moduleIndex["PARTIAL-FWD-MIB"] = []*module.Module{source}

		importing := &module.Module{Name: "IMPORTER"}
		resolveImportsFromModule(ctx, importing, "PARTIAL-FWD-MIB",
			syms("local", "forwarded", "missing"))

		imports := ctx.importSources[importing]
		testutil.Equal(t, 2, len(imports), "import count")
		testutil.Equal(t, source, imports["local"], "local should resolve to PARTIAL-FWD-MIB")
		testutil.Equal(t, realSource, imports["forwarded"], "forwarded should resolve to REAL-SOURCE")
		fwdImports := unresolvedByKind(ctx, model.UnresolvedImport)
		testutil.Len(t, fwdImports, 1, "unresolved count")
		testutil.Equal(t, "missing", fwdImports[0].Symbol, "unresolved symbol")
		testutil.Equal(t, reasonSymbolNotExported, fwdImports[0].Reason, "unresolved reason")
	})

	t.Run("module not found", func(t *testing.T) {
		ctx := newTestContext()
		importing := &module.Module{Name: "IMPORTER"}
		resolveImportsFromModule(ctx, importing, "NONEXISTENT-MIB",
			syms("something"))

		testutil.Equal(t, 0, len(ctx.importSources[importing]), "expected no imports when module is not found")
		notFoundImports := unresolvedByKind(ctx, model.UnresolvedImport)
		testutil.Len(t, notFoundImports, 1, "unresolved count")
		testutil.Equal(t, reasonModuleNotFound, notFoundImports[0].Reason, "reason")

		diags := ctx.Diagnostics()
		found := false
		for _, d := range diags {
			if d.Code == "import-module-not-found" && d.Module == "IMPORTER" {
				found = true
				break
			}
		}
		testutil.True(t, found, "expected diagnostic code %q for module IMPORTER in %v", "import-module-not-found", diags)
	})

	t.Run("missing symbol in strict mode uses partial resolution", func(t *testing.T) {
		ctx := newTestContextWithConfig(model.VerboseConfig())
		// Candidate exists but doesn't have the symbol
		source := makeTestModule(ctx, "SRC", []string{"other"})
		ctx.moduleIndex["SRC"] = []*module.Module{source}

		importing := &module.Module{Name: "IMPORTER"}
		resolveImportsFromModule(ctx, importing, "SRC",
			syms("missing"))

		// Partial resolution is active at all levels, so missing symbols
		// are reported as symbol_not_exported, not module_not_found.
		testutil.Equal(t, 0, len(ctx.importSources[importing]), "expected no imports for missing symbol")
		missingImports := unresolvedByKind(ctx, model.UnresolvedImport)
		testutil.Len(t, missingImports, 1, "unresolved count")
		testutil.Equal(t, reasonSymbolNotExported, missingImports[0].Reason, "reason")
	})

	t.Run("forwarding works in strict mode", func(t *testing.T) {
		ctx := newTestContextWithConfig(model.VerboseConfig())

		realSource := makeTestModule(ctx, "REAL", []string{"sym"})
		ctx.moduleIndex["REAL"] = []*module.Module{realSource}

		intermediate := &module.Module{
			Name: "INTER",
			Imports: []module.Import{
				{Module: "REAL", Symbol: "sym"},
			},
		}
		ctx.defNames[intermediate] = map[string]struct{}{}
		ctx.moduleIndex["INTER"] = []*module.Module{intermediate}

		importing := &module.Module{Name: "IMPORTER"}
		resolveImportsFromModule(ctx, importing, "INTER", syms("sym"))

		imports := ctx.importSources[importing]
		testutil.Equal(t, 1, len(imports), "forwarding should resolve in strict mode")
		testutil.Equal(t, realSource, imports["sym"], "sym should be forwarded from REAL")
	})

	t.Run("dead-end forwarding is rejected on original importer", func(t *testing.T) {
		ctx := newTestContextWithConfig(model.DefaultConfig())
		deadEnd := makeTestModule(ctx, "DEAD-END", nil)
		ctx.moduleIndex["DEAD-END"] = []*module.Module{deadEnd}
		intermediate := makeTestModule(ctx, "INTERMEDIATE", nil)
		intermediate.Imports = []module.Import{{Module: "DEAD-END", Symbol: "missing"}}
		ctx.moduleIndex["INTERMEDIATE"] = []*module.Module{intermediate}

		importing := &module.Module{
			Name:      "IMPORTER",
			Imports:   []module.Import{{Module: "INTERMEDIATE", Symbol: "missing", Span: types.Span{Start: 7, End: 14}}},
			LineTable: []int{0, 5},
		}
		resolveImportsFromModule(ctx, importing, "INTERMEDIATE", []importSymbol{{name: "missing", span: importing.Imports[0].Span}})

		testutil.Equal(t, 0, len(ctx.importSources[importing]), "dead-end forwarding must not register an import")
		unresolved := unresolvedByKind(ctx, model.UnresolvedImport)
		testutil.Len(t, unresolved, 1, "unresolved count")
		testutil.Equal(t, "IMPORTER", unresolved[0].Module, "failure should belong to original importer")
		testutil.Equal(t, "missing", unresolved[0].Symbol, "unresolved symbol")
		testutil.Equal(t, reasonSymbolNotExported, unresolved[0].Reason, "unresolved reason")
		diag := hasDiag(t, ctx.Diagnostics(), types.DiagImportNotFound)
		testutil.Equal(t, "IMPORTER", diag.Module, "diagnostic should belong to original importer")
		testutil.Equal(t, 2, diag.Line, "diagnostic line should come from original import span")
		testutil.Equal(t, 3, diag.Column, "diagnostic column should come from original import span")
		testutil.Contains(t, diag.Message, `from "INTERMEDIATE"`, "diagnostic should name original source module")
	})

	t.Run("forwarding cycle is rejected on original importer", func(t *testing.T) {
		ctx := newTestContextWithConfig(model.DefaultConfig())
		modB := makeTestModule(ctx, "B", nil)
		modC := makeTestModule(ctx, "C", nil)
		modB.Imports = []module.Import{{Module: "C", Symbol: "loop"}}
		modC.Imports = []module.Import{{Module: "B", Symbol: "loop"}}
		ctx.moduleIndex["B"] = []*module.Module{modB}
		ctx.moduleIndex["C"] = []*module.Module{modC}

		span := types.Span{Start: 12, End: 16}
		importing := &module.Module{
			Name:      "IMPORTER",
			Imports:   []module.Import{{Module: "B", Symbol: "loop", Span: span}},
			LineTable: []int{0, 10},
		}
		resolveImportsFromModule(ctx, importing, "B", []importSymbol{{name: "loop", span: span}})

		testutil.Equal(t, 0, len(ctx.importSources[importing]), "cyclic forwarding must not register an import")
		unresolved := unresolvedByKind(ctx, model.UnresolvedImport)
		testutil.Len(t, unresolved, 1, "unresolved count")
		testutil.Equal(t, "IMPORTER", unresolved[0].Module, "failure should belong to original importer")
		testutil.Equal(t, reasonSymbolNotExported, unresolved[0].Reason, "unresolved reason")
		diag := hasDiag(t, ctx.Diagnostics(), types.DiagImportNotFound)
		testutil.Equal(t, "IMPORTER", diag.Module, "diagnostic should belong to original importer")
		testutil.Equal(t, 2, diag.Line, "diagnostic line should come from original import span")
		testutil.Equal(t, 3, diag.Column, "diagnostic column should come from original import span")
		testutil.Contains(t, diag.Message, `from "B"`, "diagnostic should name original source module")
	})

	t.Run("recursive candidates skip newer cycle and use valid sibling", func(t *testing.T) {
		ctx := newTestContextWithConfig(model.DefaultConfig())
		validOlder := makeTestModule(ctx, "VERSIONED", []string{"ForwardedType"})
		validOlder.LastUpdated = "202001010000Z"
		cyclicNewer := makeTestModule(ctx, "VERSIONED", nil)
		cyclicNewer.LastUpdated = "202501010000Z"
		cycleEntry := makeTestModule(ctx, "CYCLE-ENTRY", nil)
		cyclicNewer.Imports = []module.Import{{Module: "CYCLE-ENTRY", Symbol: "ForwardedType"}}
		cycleEntry.Imports = []module.Import{{Module: "CYCLE-BACK", Symbol: "ForwardedType"}}

		// Put the valid candidate first so recursive resolution must reorder by
		// LAST-UPDATED, reject the newer cyclic path, then try the valid sibling.
		candidates := []*module.Module{validOlder, cyclicNewer}
		ctx.moduleIndex["CYCLE-ENTRY"] = []*module.Module{cycleEntry}
		ctx.moduleIndex["CYCLE-BACK"] = []*module.Module{cyclicNewer}

		visited := make(map[*module.Module]struct{})
		source, ok := resolveImportedSymbolWithVisited(ctx, candidates, "ForwardedType", visited)
		testutil.True(t, ok, "valid sibling should resolve after cyclic candidate")
		testutil.Equal(t, validOlder, source, "ultimate mapping should use valid sibling")
		testutil.Equal(t, 0, len(visited), "recursive candidate paths should clean up visited state")
	})

	t.Run("valid multi-hop forwarding resolves definer and tracks importer usage", func(t *testing.T) {
		ctx := newTestContextWithConfig(model.DefaultConfig())
		definer := makeTestModule(ctx, "DEFINER", []string{"ForwardedType"})
		intermediateC := makeTestModule(ctx, "C", nil)
		intermediateB := makeTestModule(ctx, "B", nil)
		intermediateC.Imports = []module.Import{{Module: "DEFINER", Symbol: "ForwardedType"}}
		intermediateB.Imports = []module.Import{{Module: "C", Symbol: "ForwardedType"}}
		ctx.moduleIndex["DEFINER"] = []*module.Module{definer}
		ctx.moduleIndex["C"] = []*module.Module{intermediateC}
		ctx.moduleIndex["B"] = []*module.Module{intermediateB}

		importing := &module.Module{Name: "IMPORTER"}
		resolveImportsFromModule(ctx, importing, "B", syms("ForwardedType"))

		testutil.Equal(t, definer, ctx.importSources[importing]["ForwardedType"], "mapping should point to ultimate definer")
		wantType := model.NewType("ForwardedType")
		ctx.typeSymbols.set(definer, "ForwardedType", wantType)
		gotType, ok := ctx.lookupType(importing, "ForwardedType")
		testutil.True(t, ok, "forwarded type should resolve")
		testutil.Equal(t, wantType, gotType, "resolved forwarded type")
		testutil.True(t, ctx.usedImports.has(importing, "ForwardedType"), "usage should be tracked on original importer")
	})
}

func TestResolveTransitiveImports(t *testing.T) {
	t.Run("direct definer unchanged", func(t *testing.T) {
		ctx := newTestContext()
		modA := &module.Module{Name: "A"}
		modB := &module.Module{Name: "B"}

		ctx.defNames[modB] = map[string]struct{}{"x": {}}
		ctx.registerImport(modA, "x", modB)

		resolveTransitiveImports(ctx)

		testutil.Equal(t, modB, ctx.importSources[modA]["x"], "direct definer should remain unchanged")
	})

	t.Run("one-hop re-export resolved", func(t *testing.T) {
		ctx := newTestContext()
		modA := &module.Module{Name: "A"}
		modB := &module.Module{Name: "B"}
		modC := &module.Module{Name: "C"}

		// B re-exports x from C; C defines x.
		ctx.defNames[modB] = map[string]struct{}{}
		ctx.defNames[modC] = map[string]struct{}{"x": {}}
		ctx.registerImport(modA, "x", modB)
		ctx.registerImport(modB, "x", modC)

		resolveTransitiveImports(ctx)

		testutil.Equal(t, modC, ctx.importSources[modA]["x"], "expected A's import of x to resolve transitively to C")
	})

	t.Run("multi-hop re-export resolved", func(t *testing.T) {
		ctx := newTestContext()
		modA := &module.Module{Name: "A"}
		modB := &module.Module{Name: "B"}
		modC := &module.Module{Name: "C"}
		modD := &module.Module{Name: "D"}

		ctx.defNames[modB] = map[string]struct{}{}
		ctx.defNames[modC] = map[string]struct{}{}
		ctx.defNames[modD] = map[string]struct{}{"x": {}}
		ctx.registerImport(modA, "x", modB)
		ctx.registerImport(modB, "x", modC)
		ctx.registerImport(modC, "x", modD)

		resolveTransitiveImports(ctx)

		testutil.Equal(t, modD, ctx.importSources[modA]["x"], "expected A->D, got A->")
		testutil.Equal(t, modD, ctx.importSources[modB]["x"], "expected B->D, got B->")
		testutil.Equal(t, modD, ctx.importSources[modC]["x"], "expected C->D, got C->")
	})

	t.Run("cycle removed and reported", func(t *testing.T) {
		ctx := newTestContext()
		modA := &module.Module{Name: "A", Imports: []module.Import{{Module: "B", Symbol: "x"}}}
		modB := &module.Module{Name: "B", Imports: []module.Import{{Module: "A", Symbol: "x"}}}

		ctx.defNames[modA] = map[string]struct{}{}
		ctx.defNames[modB] = map[string]struct{}{}
		ctx.registerImport(modA, "x", modB)
		ctx.registerImport(modB, "x", modA)

		resolveTransitiveImports(ctx)

		testutil.False(t, ctx.importSources.has(modA, "x"), "A's cyclic mapping should be removed")
		testutil.False(t, ctx.importSources.has(modB, "x"), "B's cyclic mapping should be removed")
		unresolved := unresolvedByKind(ctx, model.UnresolvedImport)
		testutil.Len(t, unresolved, 2, "each original importer should report failure")
		for _, ref := range unresolved {
			testutil.Equal(t, reasonSymbolNotExported, ref.Reason, "cycle unresolved reason")
		}
	})

	t.Run("dead end removed and reported", func(t *testing.T) {
		ctx := newTestContext()
		modA := &module.Module{Name: "A", Imports: []module.Import{{Module: "B", Symbol: "x"}}}
		modB := &module.Module{Name: "B", Imports: []module.Import{{Module: "C", Symbol: "x"}}}
		modC := &module.Module{Name: "C"}

		ctx.defNames[modB] = map[string]struct{}{}
		ctx.defNames[modC] = map[string]struct{}{}
		ctx.registerImport(modA, "x", modB)
		ctx.registerImport(modB, "x", modC)

		resolveTransitiveImports(ctx)

		testutil.False(t, ctx.importSources.has(modA, "x"), "A's dead-end mapping should be removed")
		testutil.False(t, ctx.importSources.has(modB, "x"), "B's dead-end mapping should be removed")
		unresolved := unresolvedByKind(ctx, model.UnresolvedImport)
		testutil.Len(t, unresolved, 2, "each original importer should report failure")
		for _, ref := range unresolved {
			testutil.Equal(t, reasonSymbolNotExported, ref.Reason, "dead-end unresolved reason")
		}
	})

	t.Run("different symbols resolve independently", func(t *testing.T) {
		ctx := newTestContext()
		modA := &module.Module{Name: "A"}
		modB := &module.Module{Name: "B"}
		modC := &module.Module{Name: "C"}

		// B defines "y" but re-exports "x" from C.
		ctx.defNames[modB] = map[string]struct{}{"y": {}}
		ctx.defNames[modC] = map[string]struct{}{"x": {}}
		ctx.registerImport(modA, "x", modB)
		ctx.registerImport(modA, "y", modB)
		ctx.registerImport(modB, "x", modC)

		resolveTransitiveImports(ctx)

		testutil.Equal(t, modC, ctx.importSources[modA]["x"], "x should resolve transitively to C")
		testutil.Equal(t, modB, ctx.importSources[modA]["y"], "y should stay at B (direct definer)")
	})
}

func TestResolveImports(t *testing.T) {
	t.Run("full integration through resolveImports", func(t *testing.T) {
		source := &module.Module{Name: "SOURCE-MIB"}
		importing := &module.Module{
			Name: "IMPORTER",
			Imports: []module.Import{
				{Module: "SOURCE-MIB", Symbol: "sysDescr", Span: types.Span{}},
				{Module: "SOURCE-MIB", Symbol: "OBJECT-TYPE", Span: types.Span{}},
				{Module: "SOURCE-MIB", Symbol: "sysName", Span: types.Span{}},
			},
		}

		ctx := newResolverContext(nil, model.ResolverNormal, model.DefaultConfig())
		ctx.modules = []*module.Module{importing}
		ctx.moduleIndex["SOURCE-MIB"] = []*module.Module{source}
		ctx.defNames[source] = map[string]struct{}{
			"sysDescr": {},
			"sysName":  {},
		}

		resolveImports(ctx)

		imports := ctx.importSources[importing]
		testutil.Equal(t, 2, len(imports), "import count")
		testutil.Equal(t, source, imports["sysDescr"], "sysDescr should resolve to SOURCE-MIB")
		testutil.Equal(t, source, imports["sysName"], "sysName should resolve to SOURCE-MIB")
	})

	t.Run("multiple source modules", func(t *testing.T) {
		sourceA := &module.Module{Name: "MOD-A"}
		sourceB := &module.Module{Name: "MOD-B"}
		importing := &module.Module{
			Name: "IMPORTER",
			Imports: []module.Import{
				{Module: "MOD-A", Symbol: "alpha"},
				{Module: "MOD-B", Symbol: "beta"},
			},
		}

		ctx := newResolverContext(nil, model.ResolverNormal, model.DefaultConfig())
		ctx.modules = []*module.Module{importing}
		ctx.moduleIndex["MOD-A"] = []*module.Module{sourceA}
		ctx.moduleIndex["MOD-B"] = []*module.Module{sourceB}
		ctx.defNames[sourceA] = map[string]struct{}{"alpha": {}}
		ctx.defNames[sourceB] = map[string]struct{}{"beta": {}}

		resolveImports(ctx)

		imports := ctx.importSources[importing]
		testutil.Equal(t, sourceA, imports["alpha"], "alpha should resolve to MOD-A")
		testutil.Equal(t, sourceB, imports["beta"], "beta should resolve to MOD-B")
	})

	t.Run("duplicate import from different modules emits diagnostic", func(t *testing.T) {
		sourceA := &module.Module{Name: "MOD-A"}
		sourceB := &module.Module{Name: "MOD-B"}
		importing := &module.Module{
			Name: "IMPORTER",
			Imports: []module.Import{
				{Module: "MOD-A", Symbol: "DisplayString"},
				{Module: "MOD-B", Symbol: "DisplayString"},
			},
		}

		ctx := newResolverContext(nil, model.ResolverNormal, model.DefaultConfig())
		ctx.modules = []*module.Module{importing}
		ctx.moduleIndex["MOD-A"] = []*module.Module{sourceA}
		ctx.moduleIndex["MOD-B"] = []*module.Module{sourceB}
		ctx.defNames[sourceA] = map[string]struct{}{"DisplayString": {}}
		ctx.defNames[sourceB] = map[string]struct{}{"DisplayString": {}}

		resolveImports(ctx)

		// First import wins.
		imports := ctx.importSources[importing]
		testutil.Equal(t, sourceA, imports["DisplayString"], "first import should win")

		// Diagnostic emitted for the duplicate.
		diag := hasDiag(t, ctx.Diagnostics(), types.DiagImportDuplicate)
		testutil.Contains(t, diag.Message, "MOD-A", "diagnostic should mention first module")
		testutil.Contains(t, diag.Message, "MOD-B", "diagnostic should mention second module")
	})

	t.Run("duplicate import from same module no diagnostic", func(t *testing.T) {
		sourceA := &module.Module{Name: "MOD-A"}
		importing := &module.Module{
			Name: "IMPORTER",
			Imports: []module.Import{
				{Module: "MOD-A", Symbol: "foo"},
				{Module: "MOD-A", Symbol: "foo"},
			},
		}

		ctx := newResolverContext(nil, model.ResolverNormal, model.DefaultConfig())
		ctx.modules = []*module.Module{importing}
		ctx.moduleIndex["MOD-A"] = []*module.Module{sourceA}
		ctx.defNames[sourceA] = map[string]struct{}{"foo": {}}

		resolveImports(ctx)

		imports := ctx.importSources[importing]
		testutil.Equal(t, sourceA, imports["foo"], "import should resolve")

		// No diagnostic for same-module duplicate.
		for _, d := range ctx.Diagnostics() {
			if d.Code == types.DiagImportDuplicate {
				t.Fatalf("unexpected duplicate import diagnostic: %s", d.Message)
			}
		}
	})
}

func TestCheckUnusedImports(t *testing.T) {
	t.Run("unused import emits diagnostic", func(t *testing.T) {
		mod := testSMIv2Module(
			[]module.Import{
				module.NewImport("SNMPv2-SMI", "Integer32", types.Span{}),
				module.NewImport("SNMPv2-SMI", "OBJECT-TYPE", types.Span{}),
				module.NewImport("SNMPv2-TC", "DisplayString", types.Span{}),
			},
			&module.ObjectType{
				DefBase: module.DefBase{Name: "testObj"},
				Syntax:  &module.TypeSyntaxTypeRef{Name: "Integer32"},
				Status:  types.StatusCurrent,
				Oid:     testOid("testRoot", 1),
			},
		)
		m := resolveStrict(mod)
		d := hasDiag(t, m.Diagnostics(), types.DiagImportUnused)
		testutil.True(t, d.Message != "", "expected non-empty message")
	})

	t.Run("all used no diagnostic", func(t *testing.T) {
		mod := testSMIv2Module(
			[]module.Import{
				module.NewImport("SNMPv2-SMI", "Integer32", types.Span{}),
				module.NewImport("SNMPv2-SMI", "OBJECT-TYPE", types.Span{}),
			},
			&module.ObjectType{
				DefBase: module.DefBase{Name: "testObj"},
				Syntax:  &module.TypeSyntaxTypeRef{Name: "Integer32"},
				Status:  types.StatusCurrent,
				Oid:     testOid("testRoot", 1),
			},
		)
		m := resolveStrict(mod)
		noDiag(t, m.Diagnostics(), types.DiagImportUnused)
	})
}

func TestCheckObsoleteImports_RFC1155InSMIv2(t *testing.T) {
	// An SMIv2 module importing from RFC1155-SMI should emit obsolete-import.
	mod := buildTestModule(testModuleConfig{
		language:    types.LanguageSMIv2,
		baseImport:  module.NewImport("RFC1155-SMI", "enterprises", types.Span{}),
		includeRoot: true,
	})

	m := resolveStrict(mod)
	hasDiag(t, m.Diagnostics(), types.DiagObsoleteImport)
}

func TestCheckObsoleteImports_RFC1213DisplayStringInSMIv2(t *testing.T) {
	// SMIv2 module importing DisplayString from RFC1213-MIB should emit obsolete-import.
	mod := buildTestModule(testModuleConfig{
		language:   types.LanguageSMIv2,
		baseImport: module.NewImport("SNMPv2-SMI", "enterprises", types.Span{}),
		extraImports: []module.Import{
			module.NewImport("RFC1213-MIB", "DisplayString", types.Span{}),
		},
		includeRoot: true,
	})

	m := resolveStrict(mod)
	hasDiag(t, m.Diagnostics(), types.DiagObsoleteImport)
}

func TestCheckObsoleteImports_SMIv1ModuleSkipped(t *testing.T) {
	// An SMIv1 module importing from RFC1155-SMI should NOT emit obsolete-import.
	mod := buildTestModule(testModuleConfig{
		language:    types.LanguageSMIv1,
		baseImport:  module.NewImport("RFC1155-SMI", "enterprises", types.Span{}),
		includeRoot: true,
	})

	m := resolveStrict(mod)
	noDiag(t, m.Diagnostics(), types.DiagObsoleteImport)
}

func TestCheckObsoleteImports_RFC1065InSMIv2(t *testing.T) {
	// SMIv2 module importing from RFC1065-SMI should emit obsolete-import.
	mod := buildTestModule(testModuleConfig{
		language:    types.LanguageSMIv2,
		baseImport:  module.NewImport("RFC1065-SMI", "enterprises", types.Span{}),
		includeRoot: true,
	})

	m := resolveStrict(mod)
	hasDiag(t, m.Diagnostics(), types.DiagObsoleteImport)
}

// --- checkIntegerMisuse ---
