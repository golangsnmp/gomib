package mib

import (
	"testing"

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
		// 12-digit without Z suffix, Z is added
		{"200210180000", "200210180000Z"},
		// Edge cases
		{"", ""},
		{"Z", "Z"},
		// 10-digit without Z suffix, still gets century + Z appended
		{"0210180000", "200210180000Z"},
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
			Definitions: []module.Definition{
				&module.ModuleIdentity{
					DefBase:     module.DefBase{Name: "testMIB"},
					LastUpdated: "0210180000Z",
				},
			},
		}
		got := extractLastUpdated(mod)
		testutil.Equal(t, "200210180000Z", got, "extractLastUpdated()")
	})

	t.Run("module without ModuleIdentity", func(t *testing.T) {
		mod := &module.Module{
			Name: "TEST-MIB",
			Definitions: []module.Definition{
				&module.ObjectType{DefBase: module.DefBase{Name: "testObject"}},
			},
		}
		got := extractLastUpdated(mod)
		testutil.Equal(t, "", got, "extractLastUpdated()")
	})

	t.Run("module with empty LastUpdated", func(t *testing.T) {
		mod := &module.Module{
			Name: "TEST-MIB",
			Definitions: []module.Definition{
				&module.ModuleIdentity{
					DefBase:     module.DefBase{Name: "testMIB"},
					LastUpdated: "   ",
				},
			},
		}
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
	ctx.moduleDefNames[mod] = defs
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
			Definitions: []module.Definition{
				&module.ModuleIdentity{
					DefBase:     module.DefBase{Name: "modOld"},
					LastUpdated: "9901010000Z",
				},
			},
		}
		ctx.moduleDefNames[modOld] = map[string]struct{}{
			"foo": {},
			"bar": {},
		}

		modNew := &module.Module{
			Name:        "MOD-NEW",
			LastUpdated: "200501010000Z",
			Definitions: []module.Definition{
				&module.ModuleIdentity{
					DefBase:     module.DefBase{Name: "modNew"},
					LastUpdated: "200501010000Z",
				},
			},
		}
		ctx.moduleDefNames[modNew] = map[string]struct{}{
			"foo": {},
			"bar": {},
		}

		got, ok := findCandidateWithAllSymbols(ctx,
			[]*module.Module{modOld, modNew},
			syms("foo", "bar"))
		testutil.True(t, ok, "expected match")
		testutil.Equal(t, modNew, got, "expected MOD-NEW (newer), got")
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
		testutil.Equal(t, 0, len(ctx.moduleImports[importing]), "expected no imports for macro-only symbols")
		testutil.Len(t, ctx.unresolvedImports, 0, "expected no unresolved imports for macro-only symbols")
	})

	t.Run("direct resolution", func(t *testing.T) {
		ctx := newTestContext()
		source := makeTestModule(ctx, "SOURCE-MIB", []string{"sysDescr", "sysName"})
		ctx.moduleIndex["SOURCE-MIB"] = []*module.Module{source}

		importing := &module.Module{Name: "IMPORTER"}
		resolveImportsFromModule(ctx, importing, "SOURCE-MIB",
			syms("sysDescr", "sysName"))

		imports := ctx.moduleImports[importing]
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

		imports := ctx.moduleImports[importing]
		testutil.Equal(t, 1, len(imports), "import count")
		testutil.Equal(t, source, imports["sysDescr"], "sysDescr should resolve to SOURCE-MIB")
	})

	t.Run("alias resolution", func(t *testing.T) {
		ctx := newTestContextWithConfig(DefaultConfig())
		// Source module is under the canonical name
		source := makeTestModule(ctx, "SNMPv2-SMI", []string{"enterprises", "Counter32"})
		ctx.moduleIndex["SNMPv2-SMI"] = []*module.Module{source}
		// Import uses the alias name (no candidates under alias)

		importing := &module.Module{Name: "IMPORTER"}
		resolveImportsFromModule(ctx, importing, "SNMPv2-SMI-v1",
			syms("enterprises", "Counter32"))

		imports := ctx.moduleImports[importing]
		testutil.Equal(t, 2, len(imports), "import count")
		testutil.Equal(t, source, imports["enterprises"], "enterprises should resolve via alias to SNMPv2-SMI")
	})

	t.Run("alias disabled in strict mode", func(t *testing.T) {
		ctx := newTestContextWithPolicy(ResolverStrict, VerboseConfig())
		source := makeTestModule(ctx, "SNMPv2-SMI", []string{"enterprises"})
		ctx.moduleIndex["SNMPv2-SMI"] = []*module.Module{source}

		importing := &module.Module{Name: "IMPORTER"}
		resolveImportsFromModule(ctx, importing, "SNMPv2-SMI-v1",
			syms("enterprises"))

		testutil.Equal(t, 0, len(ctx.moduleImports[importing]), "alias should not resolve in strict mode")
		testutil.Equal(t, 1, len(ctx.unresolvedImports), "strict should record unresolved aliased import")
	})

	t.Run("forwarding resolution", func(t *testing.T) {
		ctx := newTestContextWithConfig(DefaultConfig())

		// Ultimate source
		realSource := &module.Module{Name: "REAL-SOURCE"}
		ctx.moduleIndex["REAL-SOURCE"] = []*module.Module{realSource}

		// Intermediate module that defines "localDef" and re-exports "remoteSym"
		intermediate := &module.Module{
			Name: "INTERMEDIATE",
			Imports: []module.Import{
				{Module: "REAL-SOURCE", Symbol: "remoteSym"},
			},
		}
		ctx.moduleDefNames[intermediate] = map[string]struct{}{
			"localDef": {},
		}
		ctx.moduleIndex["INTERMEDIATE"] = []*module.Module{intermediate}

		importing := &module.Module{Name: "IMPORTER"}
		resolveImportsFromModule(ctx, importing, "INTERMEDIATE",
			syms("localDef", "remoteSym"))

		imports := ctx.moduleImports[importing]
		testutil.Equal(t, 2, len(imports), "import count")
		testutil.Equal(t, intermediate, imports["localDef"], "localDef should come from INTERMEDIATE")
		testutil.Equal(t, realSource, imports["remoteSym"], "remoteSym should be forwarded from REAL-SOURCE")
	})

	t.Run("partial resolution", func(t *testing.T) {
		ctx := newTestContextWithConfig(DefaultConfig())

		// Candidate module only has some of the symbols
		source := makeTestModule(ctx, "PARTIAL-MIB", []string{"found1", "found2"})
		ctx.moduleIndex["PARTIAL-MIB"] = []*module.Module{source}

		importing := &module.Module{Name: "IMPORTER"}
		resolveImportsFromModule(ctx, importing, "PARTIAL-MIB",
			syms("found1", "found2", "missing1"))

		imports := ctx.moduleImports[importing]
		testutil.Equal(t, 2, len(imports), "import count")
		testutil.Equal(t, source, imports["found1"], "found1 should resolve to PARTIAL-MIB")
		testutil.Equal(t, source, imports["found2"], "found2 should resolve to PARTIAL-MIB")
		testutil.Len(t, ctx.unresolvedImports, 1, "unresolved count")
		testutil.Equal(t, "missing1", ctx.unresolvedImports[0].symbol, "unresolved symbol")
		testutil.Equal(t, reasonSymbolNotExported, ctx.unresolvedImports[0].reason, "unresolved reason")
	})

	t.Run("partial resolution keeps forwarded symbols", func(t *testing.T) {
		ctx := newTestContextWithConfig(DefaultConfig())

		realSource := &module.Module{Name: "REAL-SOURCE"}
		ctx.moduleIndex["REAL-SOURCE"] = []*module.Module{realSource}

		source := &module.Module{
			Name: "PARTIAL-FWD-MIB",
			Imports: []module.Import{
				{Module: "REAL-SOURCE", Symbol: "forwarded"},
			},
		}
		ctx.moduleDefNames[source] = map[string]struct{}{
			"local": {},
		}
		ctx.moduleIndex["PARTIAL-FWD-MIB"] = []*module.Module{source}

		importing := &module.Module{Name: "IMPORTER"}
		resolveImportsFromModule(ctx, importing, "PARTIAL-FWD-MIB",
			syms("local", "forwarded", "missing"))

		imports := ctx.moduleImports[importing]
		testutil.Equal(t, 2, len(imports), "import count")
		testutil.Equal(t, source, imports["local"], "local should resolve to PARTIAL-FWD-MIB")
		testutil.Equal(t, realSource, imports["forwarded"], "forwarded should resolve to REAL-SOURCE")
		testutil.Len(t, ctx.unresolvedImports, 1, "unresolved count")
		testutil.Equal(t, "missing", ctx.unresolvedImports[0].symbol, "unresolved symbol")
		testutil.Equal(t, reasonSymbolNotExported, ctx.unresolvedImports[0].reason, "unresolved reason")
	})

	t.Run("module not found", func(t *testing.T) {
		ctx := newTestContext()
		importing := &module.Module{Name: "IMPORTER"}
		resolveImportsFromModule(ctx, importing, "NONEXISTENT-MIB",
			syms("something"))

		testutil.Equal(t, 0, len(ctx.moduleImports[importing]), "expected no imports when module is not found")
		testutil.Len(t, ctx.unresolvedImports, 1, "unresolved count")
		testutil.Equal(t, reasonModuleNotFound, ctx.unresolvedImports[0].reason, "reason")

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
		ctx := newTestContextWithConfig(VerboseConfig())
		// Candidate exists but doesn't have the symbol
		source := makeTestModule(ctx, "SRC", []string{"other"})
		ctx.moduleIndex["SRC"] = []*module.Module{source}

		importing := &module.Module{Name: "IMPORTER"}
		resolveImportsFromModule(ctx, importing, "SRC",
			syms("missing"))

		// Partial resolution is active at all levels, so missing symbols
		// are reported as symbol_not_exported, not module_not_found.
		testutil.Equal(t, 0, len(ctx.moduleImports[importing]), "expected no imports for missing symbol")
		testutil.Len(t, ctx.unresolvedImports, 1, "unresolved count")
		testutil.Equal(t, reasonSymbolNotExported, ctx.unresolvedImports[0].reason, "reason")
	})

	t.Run("forwarding works in strict mode", func(t *testing.T) {
		ctx := newTestContextWithConfig(VerboseConfig())

		realSource := &module.Module{Name: "REAL"}
		ctx.moduleIndex["REAL"] = []*module.Module{realSource}

		intermediate := &module.Module{
			Name: "INTER",
			Imports: []module.Import{
				{Module: "REAL", Symbol: "sym"},
			},
		}
		ctx.moduleDefNames[intermediate] = map[string]struct{}{}
		ctx.moduleIndex["INTER"] = []*module.Module{intermediate}

		importing := &module.Module{Name: "IMPORTER"}
		resolveImportsFromModule(ctx, importing, "INTER", syms("sym"))

		imports := ctx.moduleImports[importing]
		testutil.Equal(t, 1, len(imports), "forwarding should resolve in strict mode")
		testutil.Equal(t, realSource, imports["sym"], "sym should be forwarded from REAL")
	})
}

func TestResolveTransitiveImports(t *testing.T) {
	t.Run("direct definer unchanged", func(t *testing.T) {
		ctx := newTestContext()
		modA := &module.Module{Name: "A"}
		modB := &module.Module{Name: "B"}

		ctx.moduleDefNames[modB] = map[string]struct{}{"x": {}}
		ctx.registerImport(modA, "x", modB)

		resolveTransitiveImports(ctx)

		testutil.Equal(t, modB, ctx.moduleImports[modA]["x"], "direct definer should remain unchanged")
	})

	t.Run("one-hop re-export resolved", func(t *testing.T) {
		ctx := newTestContext()
		modA := &module.Module{Name: "A"}
		modB := &module.Module{Name: "B"}
		modC := &module.Module{Name: "C"}

		// B re-exports x from C; C defines x.
		ctx.moduleDefNames[modB] = map[string]struct{}{}
		ctx.moduleDefNames[modC] = map[string]struct{}{"x": {}}
		ctx.registerImport(modA, "x", modB)
		ctx.registerImport(modB, "x", modC)

		resolveTransitiveImports(ctx)

		testutil.Equal(t, modC, ctx.moduleImports[modA]["x"], "expected A's import of x to resolve transitively to C")
	})

	t.Run("multi-hop re-export resolved", func(t *testing.T) {
		ctx := newTestContext()
		modA := &module.Module{Name: "A"}
		modB := &module.Module{Name: "B"}
		modC := &module.Module{Name: "C"}
		modD := &module.Module{Name: "D"}

		ctx.moduleDefNames[modB] = map[string]struct{}{}
		ctx.moduleDefNames[modC] = map[string]struct{}{}
		ctx.moduleDefNames[modD] = map[string]struct{}{"x": {}}
		ctx.registerImport(modA, "x", modB)
		ctx.registerImport(modB, "x", modC)
		ctx.registerImport(modC, "x", modD)

		resolveTransitiveImports(ctx)

		testutil.Equal(t, modD, ctx.moduleImports[modA]["x"], "expected A->D, got A->")
		testutil.Equal(t, modD, ctx.moduleImports[modB]["x"], "expected B->D, got B->")
		testutil.Equal(t, modD, ctx.moduleImports[modC]["x"], "expected C->D, got C->")
	})

	t.Run("cycle does not panic", func(_ *testing.T) {
		ctx := newTestContext()
		modA := &module.Module{Name: "A"}
		modB := &module.Module{Name: "B"}

		ctx.moduleDefNames[modA] = map[string]struct{}{}
		ctx.moduleDefNames[modB] = map[string]struct{}{}
		ctx.registerImport(modA, "x", modB)
		ctx.registerImport(modB, "x", modA)

		// Should not panic or infinite loop.
		resolveTransitiveImports(ctx)
	})

	t.Run("dead end preserved", func(t *testing.T) {
		ctx := newTestContext()
		modA := &module.Module{Name: "A"}
		modB := &module.Module{Name: "B"}

		// B neither defines x nor imports it.
		ctx.moduleDefNames[modB] = map[string]struct{}{}
		ctx.registerImport(modA, "x", modB)

		resolveTransitiveImports(ctx)

		testutil.Equal(t, modB, ctx.moduleImports[modA]["x"], "dead end should keep the original target")
	})

	t.Run("different symbols resolve independently", func(t *testing.T) {
		ctx := newTestContext()
		modA := &module.Module{Name: "A"}
		modB := &module.Module{Name: "B"}
		modC := &module.Module{Name: "C"}

		// B defines "y" but re-exports "x" from C.
		ctx.moduleDefNames[modB] = map[string]struct{}{"y": {}}
		ctx.moduleDefNames[modC] = map[string]struct{}{"x": {}}
		ctx.registerImport(modA, "x", modB)
		ctx.registerImport(modA, "y", modB)
		ctx.registerImport(modB, "x", modC)

		resolveTransitiveImports(ctx)

		testutil.Equal(t, modC, ctx.moduleImports[modA]["x"], "x should resolve transitively to C")
		testutil.Equal(t, modB, ctx.moduleImports[modA]["y"], "y should stay at B (direct definer)")
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

		ctx := newResolverContext(nil, ResolverNormal, DefaultConfig())
		ctx.modules = []*module.Module{importing}
		ctx.moduleIndex["SOURCE-MIB"] = []*module.Module{source}
		ctx.moduleDefNames[source] = map[string]struct{}{
			"sysDescr": {},
			"sysName":  {},
		}

		resolveImports(ctx)

		imports := ctx.moduleImports[importing]
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

		ctx := newResolverContext(nil, ResolverNormal, DefaultConfig())
		ctx.modules = []*module.Module{importing}
		ctx.moduleIndex["MOD-A"] = []*module.Module{sourceA}
		ctx.moduleIndex["MOD-B"] = []*module.Module{sourceB}
		ctx.moduleDefNames[sourceA] = map[string]struct{}{"alpha": {}}
		ctx.moduleDefNames[sourceB] = map[string]struct{}{"beta": {}}

		resolveImports(ctx)

		imports := ctx.moduleImports[importing]
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

		ctx := newResolverContext(nil, ResolverNormal, DefaultConfig())
		ctx.modules = []*module.Module{importing}
		ctx.moduleIndex["MOD-A"] = []*module.Module{sourceA}
		ctx.moduleIndex["MOD-B"] = []*module.Module{sourceB}
		ctx.moduleDefNames[sourceA] = map[string]struct{}{"DisplayString": {}}
		ctx.moduleDefNames[sourceB] = map[string]struct{}{"DisplayString": {}}

		resolveImports(ctx)

		// First import wins.
		imports := ctx.moduleImports[importing]
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

		ctx := newResolverContext(nil, ResolverNormal, DefaultConfig())
		ctx.modules = []*module.Module{importing}
		ctx.moduleIndex["MOD-A"] = []*module.Module{sourceA}
		ctx.moduleDefNames[sourceA] = map[string]struct{}{"foo": {}}

		resolveImports(ctx)

		imports := ctx.moduleImports[importing]
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
