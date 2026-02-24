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
		if isMacroSymbol(name) {
			t.Errorf("isMacroSymbol(%q) = true, want false", name)
		}
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
			Name: "TEST-MIB",
			Definitions: []module.Definition{
				&module.ModuleIdentity{
					Name:        "testMIB",
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
				&module.ObjectType{Name: "testObject"},
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
					Name:        "testMIB",
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
	ctx.ModuleDefNames[mod] = defs
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
			Name: "MOD-OLD",
			Definitions: []module.Definition{
				&module.ModuleIdentity{
					Name:        "modOld",
					LastUpdated: "9901010000Z",
				},
			},
		}
		ctx.ModuleDefNames[modOld] = map[string]struct{}{
			"foo": {},
			"bar": {},
		}

		modNew := &module.Module{
			Name: "MOD-NEW",
			Definitions: []module.Definition{
				&module.ModuleIdentity{
					Name:        "modNew",
					LastUpdated: "200501010000Z",
				},
			},
		}
		ctx.ModuleDefNames[modNew] = map[string]struct{}{
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

func TestTryPartialResolution(t *testing.T) {
	syms := func(names ...string) []importSymbol {
		out := make([]importSymbol, len(names))
		for i, n := range names {
			out[i] = importSymbol{name: n}
		}
		return out
	}

	t.Run("all symbols resolved", func(t *testing.T) {
		ctx := newTestContext()
		mod := makeTestModule(ctx, "SRC", []string{"a", "b", "c"})
		resolved, unresolved := tryPartialResolution(ctx, []*module.Module{mod}, syms("a", "b"))
		testutil.Len(t, resolved, 2, "resolved count")
		testutil.Len(t, unresolved, 0, "unresolved count")
	})

	t.Run("partial resolution", func(t *testing.T) {
		ctx := newTestContext()
		mod := makeTestModule(ctx, "SRC", []string{"a", "c"})
		resolved, unresolved := tryPartialResolution(ctx, []*module.Module{mod}, syms("a", "b", "c"))
		testutil.Len(t, resolved, 2, "resolved count")
		testutil.Len(t, unresolved, 1, "unresolved count")
		if len(unresolved) > 0 && unresolved[0].name != "b" {
			t.Errorf("unresolved symbol = %q, want %q", unresolved[0].name, "b")
		}
	})

	t.Run("no symbols resolved", func(t *testing.T) {
		ctx := newTestContext()
		mod := makeTestModule(ctx, "SRC", []string{"x", "y"})
		resolved, unresolved := tryPartialResolution(ctx, []*module.Module{mod}, syms("a", "b"))
		testutil.Len(t, resolved, 0, "resolved count")
		testutil.Len(t, unresolved, 2, "unresolved count")
	})

	t.Run("multiple candidates, first match wins", func(t *testing.T) {
		ctx := newTestContext()
		mod1 := makeTestModule(ctx, "SRC-1", []string{"a"})
		mod2 := makeTestModule(ctx, "SRC-2", []string{"a", "b"})
		resolved, _ := tryPartialResolution(ctx, []*module.Module{mod1, mod2}, syms("a"))
		testutil.Len(t, resolved, 1, "resolved count")
		testutil.Equal(t, mod1, resolved[0].source, "expected symbol to resolve from SRC-1, got")
	})

	t.Run("candidate with nil defNames", func(t *testing.T) {
		ctx := newTestContext()
		modNil := &module.Module{Name: "NIL-MOD"}
		modGood := makeTestModule(ctx, "GOOD", []string{"a"})
		resolved, unresolved := tryPartialResolution(ctx, []*module.Module{modNil, modGood}, syms("a"))
		testutil.Len(t, resolved, 1, "resolved count")
		testutil.Len(t, unresolved, 0, "unresolved count")
	})
}

func TestTryImportForwarding(t *testing.T) {
	syms := func(names ...string) []importSymbol {
		out := make([]importSymbol, len(names))
		for i, n := range names {
			out[i] = importSymbol{name: n}
		}
		return out
	}

	t.Run("direct definition in candidate", func(t *testing.T) {
		ctx := newTestContext()
		candidate := makeTestModule(ctx, "BASE", []string{"foo", "bar"})
		result := tryImportForwarding(ctx, []*module.Module{candidate}, syms("foo", "bar"))
		testutil.Len(t, result, 2, "forwarded count")
		for _, fwd := range result {
			testutil.Equal(t, candidate, fwd.source, "expected source BASE, got")
		}
	})

	t.Run("re-exported via imports", func(t *testing.T) {
		ctx := newTestContext()
		// The ultimate source module
		sourceMod := &module.Module{Name: "REAL-SOURCE"}

		// Intermediate candidate that re-exports "bar" from REAL-SOURCE
		candidate := &module.Module{
			Name: "INTERMEDIATE",
			Imports: []module.Import{
				{Module: "REAL-SOURCE", Symbol: "bar"},
			},
		}
		ctx.ModuleDefNames[candidate] = map[string]struct{}{
			"foo": {},
		}

		// Register REAL-SOURCE in the module index so forwarding can find it
		ctx.ModuleIndex["REAL-SOURCE"] = []*module.Module{sourceMod}

		result := tryImportForwarding(ctx, []*module.Module{candidate}, syms("foo", "bar"))
		testutil.Len(t, result, 2, "forwarded count")

		// foo should come from candidate (direct), bar from sourceMod (forwarded)
		symbolSources := make(map[string]*module.Module)
		for _, fwd := range result {
			symbolSources[fwd.symbol] = fwd.source
		}
		testutil.Equal(t, candidate, symbolSources["foo"], "foo should come from INTERMEDIATE")
		testutil.Equal(t, sourceMod, symbolSources["bar"], "bar should come from REAL-SOURCE")
	})

	t.Run("forwarded symbol source module not found", func(t *testing.T) {
		ctx := newTestContext()
		candidate := &module.Module{
			Name: "INTERMEDIATE",
			Imports: []module.Import{
				{Module: "MISSING-MODULE", Symbol: "bar"},
			},
		}
		ctx.ModuleDefNames[candidate] = map[string]struct{}{
			"foo": {},
		}
		// MISSING-MODULE is not in the module index
		result := tryImportForwarding(ctx, []*module.Module{candidate}, syms("foo", "bar"))
		testutil.Nil(t, result, "expected nil when forwarded module is missing, got  results")
	})

	t.Run("symbol not found anywhere", func(t *testing.T) {
		ctx := newTestContext()
		candidate := makeTestModule(ctx, "BASE", []string{"foo"})
		candidate.Imports = nil
		result := tryImportForwarding(ctx, []*module.Module{candidate}, syms("foo", "missing"))
		testutil.Nil(t, result, "expected nil when symbol is not found, got  results")
	})

	t.Run("no candidates", func(t *testing.T) {
		ctx := newTestContext()
		result := tryImportForwarding(ctx, nil, syms("foo"))
		testutil.Nil(t, result, "expected nil with no candidates, got  results")
	})

	t.Run("multiple candidates, second succeeds", func(t *testing.T) {
		ctx := newTestContext()
		// First candidate has no definitions or imports for the symbol
		cand1 := &module.Module{Name: "CAND-1"}
		ctx.ModuleDefNames[cand1] = map[string]struct{}{}

		sourceMod := &module.Module{Name: "SOURCE"}
		ctx.ModuleIndex["SOURCE"] = []*module.Module{sourceMod}

		cand2 := &module.Module{
			Name: "CAND-2",
			Imports: []module.Import{
				{Module: "SOURCE", Symbol: "alpha"},
			},
		}
		ctx.ModuleDefNames[cand2] = map[string]struct{}{}

		result := tryImportForwarding(ctx, []*module.Module{cand1, cand2}, syms("alpha"))
		testutil.Len(t, result, 1, "forwarded count")
		testutil.Equal(t, sourceMod, result[0].source, "expected source SOURCE, got")
	})
}

// newTestContextWithConfig creates a resolverContext with a specific DiagnosticConfig.
func newTestContextWithConfig(config DiagnosticConfig) *resolverContext {
	return newResolverContext(nil, nil, config)
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
		testutil.Equal(t, 0, len(ctx.ModuleImports[importing]), "expected no imports for macro-only symbols")
		testutil.Len(t, ctx.unresolvedImports, 0, "expected no unresolved imports for macro-only symbols")
	})

	t.Run("direct resolution", func(t *testing.T) {
		ctx := newTestContext()
		source := makeTestModule(ctx, "SOURCE-MIB", []string{"sysDescr", "sysName"})
		ctx.ModuleIndex["SOURCE-MIB"] = []*module.Module{source}

		importing := &module.Module{Name: "IMPORTER"}
		resolveImportsFromModule(ctx, importing, "SOURCE-MIB",
			syms("sysDescr", "sysName"))

		imports := ctx.ModuleImports[importing]
		testutil.Equal(t, 2, len(imports), "import count")
		testutil.Equal(t, source, imports["sysDescr"], "sysDescr should resolve to SOURCE-MIB")
		testutil.Equal(t, source, imports["sysName"], "sysName should resolve to SOURCE-MIB")
	})

	t.Run("macros filtered, non-macros resolved", func(t *testing.T) {
		ctx := newTestContext()
		source := makeTestModule(ctx, "SOURCE-MIB", []string{"sysDescr"})
		ctx.ModuleIndex["SOURCE-MIB"] = []*module.Module{source}

		importing := &module.Module{Name: "IMPORTER"}
		resolveImportsFromModule(ctx, importing, "SOURCE-MIB",
			syms("OBJECT-TYPE", "sysDescr"))

		imports := ctx.ModuleImports[importing]
		testutil.Equal(t, 1, len(imports), "import count")
		testutil.Equal(t, source, imports["sysDescr"], "sysDescr should resolve to SOURCE-MIB")
	})

	t.Run("alias resolution", func(t *testing.T) {
		ctx := newTestContextWithConfig(DefaultConfig())
		// Source module is under the canonical name
		source := makeTestModule(ctx, "SNMPv2-SMI", []string{"enterprises", "Counter32"})
		ctx.ModuleIndex["SNMPv2-SMI"] = []*module.Module{source}
		// Import uses the alias name (no candidates under alias)

		importing := &module.Module{Name: "IMPORTER"}
		resolveImportsFromModule(ctx, importing, "SNMPv2-SMI-v1",
			syms("enterprises", "Counter32"))

		imports := ctx.ModuleImports[importing]
		testutil.Equal(t, 2, len(imports), "import count")
		testutil.Equal(t, source, imports["enterprises"], "enterprises should resolve via alias to SNMPv2-SMI")
	})

	t.Run("alias disabled in strict mode", func(t *testing.T) {
		ctx := newTestContextWithConfig(StrictConfig())
		source := makeTestModule(ctx, "SNMPv2-SMI", []string{"enterprises"})
		ctx.ModuleIndex["SNMPv2-SMI"] = []*module.Module{source}

		importing := &module.Module{Name: "IMPORTER"}
		resolveImportsFromModule(ctx, importing, "SNMPv2-SMI-v1",
			syms("enterprises"))

		testutil.Equal(t, 0, len(ctx.ModuleImports[importing]), "alias should not be used in strict mode")
		testutil.Len(t, ctx.unresolvedImports, 1, "expected 1 unresolved, got")
	})

	t.Run("forwarding resolution", func(t *testing.T) {
		ctx := newTestContextWithConfig(DefaultConfig())

		// Ultimate source
		realSource := &module.Module{Name: "REAL-SOURCE"}
		ctx.ModuleIndex["REAL-SOURCE"] = []*module.Module{realSource}

		// Intermediate module that defines "localDef" and re-exports "remoteSym"
		intermediate := &module.Module{
			Name: "INTERMEDIATE",
			Imports: []module.Import{
				{Module: "REAL-SOURCE", Symbol: "remoteSym"},
			},
		}
		ctx.ModuleDefNames[intermediate] = map[string]struct{}{
			"localDef": {},
		}
		ctx.ModuleIndex["INTERMEDIATE"] = []*module.Module{intermediate}

		importing := &module.Module{Name: "IMPORTER"}
		resolveImportsFromModule(ctx, importing, "INTERMEDIATE",
			syms("localDef", "remoteSym"))

		imports := ctx.ModuleImports[importing]
		testutil.Equal(t, 2, len(imports), "import count")
		testutil.Equal(t, intermediate, imports["localDef"], "localDef should come from INTERMEDIATE")
		testutil.Equal(t, realSource, imports["remoteSym"], "remoteSym should be forwarded from REAL-SOURCE")
	})

	t.Run("partial resolution", func(t *testing.T) {
		ctx := newTestContextWithConfig(DefaultConfig())

		// Candidate module only has some of the symbols
		source := makeTestModule(ctx, "PARTIAL-MIB", []string{"found1", "found2"})
		ctx.ModuleIndex["PARTIAL-MIB"] = []*module.Module{source}

		importing := &module.Module{Name: "IMPORTER"}
		resolveImportsFromModule(ctx, importing, "PARTIAL-MIB",
			syms("found1", "found2", "missing1"))

		imports := ctx.ModuleImports[importing]
		testutil.Equal(t, 2, len(imports), "import count")
		if imports["found1"] != source || imports["found2"] != source {
			t.Error("found symbols should resolve to PARTIAL-MIB")
		}
		testutil.Len(t, ctx.unresolvedImports, 1, "unresolved count")
		testutil.Equal(t, "missing1", ctx.unresolvedImports[0].symbol, "unresolved symbol")
		testutil.Equal(t, reasonSymbolNotExported, ctx.unresolvedImports[0].reason, "unresolved reason")
	})

	t.Run("module not found", func(t *testing.T) {
		ctx := newTestContext()
		importing := &module.Module{Name: "IMPORTER"}
		resolveImportsFromModule(ctx, importing, "NONEXISTENT-MIB",
			syms("something"))

		testutil.Equal(t, 0, len(ctx.ModuleImports[importing]), "expected no imports when module is not found")
		testutil.Len(t, ctx.unresolvedImports, 1, "unresolved count")
		testutil.Equal(t, reasonModuleNotFound, ctx.unresolvedImports[0].reason, "reason")
	})

	t.Run("module not found emits correct diagnostic code", func(t *testing.T) {
		ctx := newTestContext()
		importing := &module.Module{Name: "IMPORTER"}
		resolveImportsFromModule(ctx, importing, "NONEXISTENT-MIB",
			syms("something"))

		diags := ctx.Diagnostics()
		found := false
		for _, d := range diags {
			if d.Code == "import-module-not-found" && d.Module == "IMPORTER" {
				found = true
				break
			}
		}
		if !found {
			codes := make([]string, len(diags))
			for i, d := range diags {
				codes[i] = d.Code
			}
			t.Errorf("expected diagnostic code %q, got codes: %v", "import-module-not-found", codes)
		}
	})

	t.Run("module not found in strict mode", func(t *testing.T) {
		ctx := newTestContextWithConfig(StrictConfig())
		// Candidate exists but doesn't have the symbol
		source := makeTestModule(ctx, "SRC", []string{"other"})
		ctx.ModuleIndex["SRC"] = []*module.Module{source}

		importing := &module.Module{Name: "IMPORTER"}
		resolveImportsFromModule(ctx, importing, "SRC",
			syms("missing"))

		// Strict mode disallows fallbacks, so it falls through to module_not_found
		testutil.Equal(t, 0, len(ctx.ModuleImports[importing]), "expected no imports in strict mode with missing symbol")
		testutil.Len(t, ctx.unresolvedImports, 1, "unresolved count")
		testutil.Equal(t, reasonModuleNotFound, ctx.unresolvedImports[0].reason, "reason")
	})

	t.Run("forwarding disabled in strict mode", func(t *testing.T) {
		ctx := newTestContextWithConfig(StrictConfig())

		realSource := &module.Module{Name: "REAL"}
		ctx.ModuleIndex["REAL"] = []*module.Module{realSource}

		intermediate := &module.Module{
			Name: "INTER",
			Imports: []module.Import{
				{Module: "REAL", Symbol: "sym"},
			},
		}
		ctx.ModuleDefNames[intermediate] = map[string]struct{}{}
		ctx.ModuleIndex["INTER"] = []*module.Module{intermediate}

		importing := &module.Module{Name: "IMPORTER"}
		resolveImportsFromModule(ctx, importing, "INTER", syms("sym"))

		testutil.Equal(t, 0, len(ctx.ModuleImports[importing]), "forwarding should not be used in strict mode")
	})
}

func TestResolveTransitiveImports(t *testing.T) {
	t.Run("direct definer unchanged", func(t *testing.T) {
		ctx := newTestContext()
		modA := &module.Module{Name: "A"}
		modB := &module.Module{Name: "B"}

		ctx.ModuleDefNames[modB] = map[string]struct{}{"x": {}}
		ctx.registerImport(modA, "x", modB)

		resolveTransitiveImports(ctx)

		testutil.Equal(t, modB, ctx.ModuleImports[modA]["x"], "direct definer should remain unchanged")
	})

	t.Run("one-hop re-export resolved", func(t *testing.T) {
		ctx := newTestContext()
		modA := &module.Module{Name: "A"}
		modB := &module.Module{Name: "B"}
		modC := &module.Module{Name: "C"}

		// B re-exports x from C; C defines x.
		ctx.ModuleDefNames[modB] = map[string]struct{}{}
		ctx.ModuleDefNames[modC] = map[string]struct{}{"x": {}}
		ctx.registerImport(modA, "x", modB)
		ctx.registerImport(modB, "x", modC)

		resolveTransitiveImports(ctx)

		testutil.Equal(t, modC, ctx.ModuleImports[modA]["x"], "expected A's import of x to resolve transitively to C")
	})

	t.Run("multi-hop re-export resolved", func(t *testing.T) {
		ctx := newTestContext()
		modA := &module.Module{Name: "A"}
		modB := &module.Module{Name: "B"}
		modC := &module.Module{Name: "C"}
		modD := &module.Module{Name: "D"}

		ctx.ModuleDefNames[modB] = map[string]struct{}{}
		ctx.ModuleDefNames[modC] = map[string]struct{}{}
		ctx.ModuleDefNames[modD] = map[string]struct{}{"x": {}}
		ctx.registerImport(modA, "x", modB)
		ctx.registerImport(modB, "x", modC)
		ctx.registerImport(modC, "x", modD)

		resolveTransitiveImports(ctx)

		testutil.Equal(t, modD, ctx.ModuleImports[modA]["x"], "expected A->D, got A->")
		testutil.Equal(t, modD, ctx.ModuleImports[modB]["x"], "expected B->D, got B->")
		testutil.Equal(t, modD, ctx.ModuleImports[modC]["x"], "expected C->D, got C->")
	})

	t.Run("cycle does not panic", func(t *testing.T) {
		ctx := newTestContext()
		modA := &module.Module{Name: "A"}
		modB := &module.Module{Name: "B"}

		ctx.ModuleDefNames[modA] = map[string]struct{}{}
		ctx.ModuleDefNames[modB] = map[string]struct{}{}
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
		ctx.ModuleDefNames[modB] = map[string]struct{}{}
		ctx.registerImport(modA, "x", modB)

		resolveTransitiveImports(ctx)

		testutil.Equal(t, modB, ctx.ModuleImports[modA]["x"], "dead end should keep the original target")
	})

	t.Run("different symbols resolve independently", func(t *testing.T) {
		ctx := newTestContext()
		modA := &module.Module{Name: "A"}
		modB := &module.Module{Name: "B"}
		modC := &module.Module{Name: "C"}

		// B defines "y" but re-exports "x" from C.
		ctx.ModuleDefNames[modB] = map[string]struct{}{"y": {}}
		ctx.ModuleDefNames[modC] = map[string]struct{}{"x": {}}
		ctx.registerImport(modA, "x", modB)
		ctx.registerImport(modA, "y", modB)
		ctx.registerImport(modB, "x", modC)

		resolveTransitiveImports(ctx)

		testutil.Equal(t, modC, ctx.ModuleImports[modA]["x"], "x should resolve transitively to C")
		testutil.Equal(t, modB, ctx.ModuleImports[modA]["y"], "y should stay at B (direct definer)")
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

		ctx := newResolverContext([]*module.Module{importing}, nil, DefaultConfig())
		ctx.ModuleIndex["SOURCE-MIB"] = []*module.Module{source}
		ctx.ModuleDefNames[source] = map[string]struct{}{
			"sysDescr": {},
			"sysName":  {},
		}

		resolveImports(ctx)

		imports := ctx.ModuleImports[importing]
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

		ctx := newResolverContext([]*module.Module{importing}, nil, DefaultConfig())
		ctx.ModuleIndex["MOD-A"] = []*module.Module{sourceA}
		ctx.ModuleIndex["MOD-B"] = []*module.Module{sourceB}
		ctx.ModuleDefNames[sourceA] = map[string]struct{}{"alpha": {}}
		ctx.ModuleDefNames[sourceB] = map[string]struct{}{"beta": {}}

		resolveImports(ctx)

		imports := ctx.ModuleImports[importing]
		testutil.Equal(t, sourceA, imports["alpha"], "alpha should resolve to MOD-A")
		testutil.Equal(t, sourceB, imports["beta"], "beta should resolve to MOD-B")
	})
}
