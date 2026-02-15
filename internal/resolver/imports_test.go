package resolver

import (
	"testing"

	"github.com/golangsnmp/gomib/internal/module"
	"github.com/golangsnmp/gomib/internal/types"
	"github.com/golangsnmp/gomib/mib"
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
		if !isMacroSymbol(name) {
			t.Errorf("isMacroSymbol(%q) = false, want true", name)
		}
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
		if got != tt.want {
			t.Errorf("baseModuleImportAlias(%q) = %q, want %q", tt.input, got, tt.want)
		}
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
		if got != tt.want {
			t.Errorf("normalizeTimestamp(%q) = %q, want %q", tt.input, got, tt.want)
		}
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
		if got != "200210180000Z" {
			t.Errorf("extractLastUpdated() = %q, want %q", got, "200210180000Z")
		}
	})

	t.Run("module without ModuleIdentity", func(t *testing.T) {
		mod := &module.Module{
			Name: "TEST-MIB",
			Definitions: []module.Definition{
				&module.ObjectType{Name: "testObject"},
			},
		}
		got := extractLastUpdated(mod)
		if got != "" {
			t.Errorf("extractLastUpdated() = %q, want empty", got)
		}
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
		if got != "" {
			t.Errorf("extractLastUpdated() = %q, want empty", got)
		}
	})

	t.Run("no definitions", func(t *testing.T) {
		mod := &module.Module{Name: "EMPTY-MIB"}
		got := extractLastUpdated(mod)
		if got != "" {
			t.Errorf("extractLastUpdated() = %q, want empty", got)
		}
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
		if ok {
			t.Error("expected no match with empty candidates")
		}
	})

	t.Run("single candidate with all symbols", func(t *testing.T) {
		ctx := newTestContext()
		mod := makeTestModule(ctx, "MOD-A", []string{"foo", "bar"})
		got, ok := findCandidateWithAllSymbols(ctx, []*module.Module{mod}, syms("foo", "bar"))
		if !ok {
			t.Fatal("expected match")
		}
		if got != mod {
			t.Error("expected MOD-A")
		}
	})

	t.Run("single candidate missing symbols", func(t *testing.T) {
		ctx := newTestContext()
		mod := makeTestModule(ctx, "MOD-A", []string{"foo"})
		_, ok := findCandidateWithAllSymbols(ctx, []*module.Module{mod}, syms("foo", "bar"))
		if ok {
			t.Error("expected no match when candidate is missing symbols")
		}
	})

	t.Run("multiple candidates, pick one with all symbols", func(t *testing.T) {
		ctx := newTestContext()
		modA := makeTestModule(ctx, "MOD-A", []string{"foo"})
		modB := makeTestModule(ctx, "MOD-B", []string{"foo", "bar", "baz"})
		got, ok := findCandidateWithAllSymbols(ctx, []*module.Module{modA, modB}, syms("foo", "bar"))
		if !ok {
			t.Fatal("expected match")
		}
		if got != modB {
			t.Errorf("expected MOD-B, got %s", got.Name)
		}
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
		if !ok {
			t.Fatal("expected match")
		}
		if got != modNew {
			t.Errorf("expected MOD-NEW (newer), got %s", got.Name)
		}
	})

	t.Run("candidate with nil defNames is skipped", func(t *testing.T) {
		ctx := newTestContext()
		modNil := &module.Module{Name: "MOD-NIL"}
		// no defNames registered for modNil
		modGood := makeTestModule(ctx, "MOD-GOOD", []string{"x"})
		got, ok := findCandidateWithAllSymbols(ctx, []*module.Module{modNil, modGood}, syms("x"))
		if !ok {
			t.Fatal("expected match")
		}
		if got != modGood {
			t.Errorf("expected MOD-GOOD, got %s", got.Name)
		}
	})

	t.Run("no candidate has all symbols", func(t *testing.T) {
		ctx := newTestContext()
		modA := makeTestModule(ctx, "A", []string{"x"})
		modB := makeTestModule(ctx, "B", []string{"y"})
		_, ok := findCandidateWithAllSymbols(ctx, []*module.Module{modA, modB}, syms("x", "y"))
		if ok {
			t.Error("expected no match when no single candidate has all symbols")
		}
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
		if len(resolved) != 2 {
			t.Errorf("resolved count = %d, want 2", len(resolved))
		}
		if len(unresolved) != 0 {
			t.Errorf("unresolved count = %d, want 0", len(unresolved))
		}
	})

	t.Run("partial resolution", func(t *testing.T) {
		ctx := newTestContext()
		mod := makeTestModule(ctx, "SRC", []string{"a", "c"})
		resolved, unresolved := tryPartialResolution(ctx, []*module.Module{mod}, syms("a", "b", "c"))
		if len(resolved) != 2 {
			t.Errorf("resolved count = %d, want 2", len(resolved))
		}
		if len(unresolved) != 1 {
			t.Errorf("unresolved count = %d, want 1", len(unresolved))
		}
		if len(unresolved) > 0 && unresolved[0].name != "b" {
			t.Errorf("unresolved symbol = %q, want %q", unresolved[0].name, "b")
		}
	})

	t.Run("no symbols resolved", func(t *testing.T) {
		ctx := newTestContext()
		mod := makeTestModule(ctx, "SRC", []string{"x", "y"})
		resolved, unresolved := tryPartialResolution(ctx, []*module.Module{mod}, syms("a", "b"))
		if len(resolved) != 0 {
			t.Errorf("resolved count = %d, want 0", len(resolved))
		}
		if len(unresolved) != 2 {
			t.Errorf("unresolved count = %d, want 2", len(unresolved))
		}
	})

	t.Run("multiple candidates, first match wins", func(t *testing.T) {
		ctx := newTestContext()
		mod1 := makeTestModule(ctx, "SRC-1", []string{"a"})
		mod2 := makeTestModule(ctx, "SRC-2", []string{"a", "b"})
		resolved, _ := tryPartialResolution(ctx, []*module.Module{mod1, mod2}, syms("a"))
		if len(resolved) != 1 {
			t.Fatalf("resolved count = %d, want 1", len(resolved))
		}
		if resolved[0].source != mod1 {
			t.Errorf("expected symbol to resolve from SRC-1, got %s", resolved[0].source.Name)
		}
	})

	t.Run("candidate with nil defNames", func(t *testing.T) {
		ctx := newTestContext()
		modNil := &module.Module{Name: "NIL-MOD"}
		modGood := makeTestModule(ctx, "GOOD", []string{"a"})
		resolved, unresolved := tryPartialResolution(ctx, []*module.Module{modNil, modGood}, syms("a"))
		if len(resolved) != 1 {
			t.Errorf("resolved count = %d, want 1", len(resolved))
		}
		if len(unresolved) != 0 {
			t.Errorf("unresolved count = %d, want 0", len(unresolved))
		}
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
		if len(result) != 2 {
			t.Fatalf("forwarded count = %d, want 2", len(result))
		}
		for _, fwd := range result {
			if fwd.source != candidate {
				t.Errorf("expected source BASE, got %s", fwd.source.Name)
			}
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
		if len(result) != 2 {
			t.Fatalf("forwarded count = %d, want 2", len(result))
		}

		// foo should come from candidate (direct), bar from sourceMod (forwarded)
		symbolSources := make(map[string]*module.Module)
		for _, fwd := range result {
			symbolSources[fwd.symbol] = fwd.source
		}
		if symbolSources["foo"] != candidate {
			t.Error("foo should come from INTERMEDIATE")
		}
		if symbolSources["bar"] != sourceMod {
			t.Error("bar should come from REAL-SOURCE")
		}
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
		if result != nil {
			t.Errorf("expected nil when forwarded module is missing, got %d results", len(result))
		}
	})

	t.Run("symbol not found anywhere", func(t *testing.T) {
		ctx := newTestContext()
		candidate := makeTestModule(ctx, "BASE", []string{"foo"})
		candidate.Imports = nil
		result := tryImportForwarding(ctx, []*module.Module{candidate}, syms("foo", "missing"))
		if result != nil {
			t.Errorf("expected nil when symbol is not found, got %d results", len(result))
		}
	})

	t.Run("no candidates", func(t *testing.T) {
		ctx := newTestContext()
		result := tryImportForwarding(ctx, nil, syms("foo"))
		if result != nil {
			t.Errorf("expected nil with no candidates, got %d results", len(result))
		}
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
		if len(result) != 1 {
			t.Fatalf("forwarded count = %d, want 1", len(result))
		}
		if result[0].source != sourceMod {
			t.Errorf("expected source SOURCE, got %s", result[0].source.Name)
		}
	})
}

// newTestContextWithConfig creates a resolverContext with a specific DiagnosticConfig.
func newTestContextWithConfig(config mib.DiagnosticConfig) *resolverContext {
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
		if len(ctx.ModuleImports[importing]) != 0 {
			t.Error("expected no imports for macro-only symbols")
		}
		if len(ctx.unresolvedImports) != 0 {
			t.Error("expected no unresolved imports for macro-only symbols")
		}
	})

	t.Run("direct resolution", func(t *testing.T) {
		ctx := newTestContext()
		source := makeTestModule(ctx, "SOURCE-MIB", []string{"sysDescr", "sysName"})
		ctx.ModuleIndex["SOURCE-MIB"] = []*module.Module{source}

		importing := &module.Module{Name: "IMPORTER"}
		resolveImportsFromModule(ctx, importing, "SOURCE-MIB",
			syms("sysDescr", "sysName"))

		imports := ctx.ModuleImports[importing]
		if len(imports) != 2 {
			t.Fatalf("import count = %d, want 2", len(imports))
		}
		if imports["sysDescr"] != source {
			t.Error("sysDescr should resolve to SOURCE-MIB")
		}
		if imports["sysName"] != source {
			t.Error("sysName should resolve to SOURCE-MIB")
		}
	})

	t.Run("macros filtered, non-macros resolved", func(t *testing.T) {
		ctx := newTestContext()
		source := makeTestModule(ctx, "SOURCE-MIB", []string{"sysDescr"})
		ctx.ModuleIndex["SOURCE-MIB"] = []*module.Module{source}

		importing := &module.Module{Name: "IMPORTER"}
		resolveImportsFromModule(ctx, importing, "SOURCE-MIB",
			syms("OBJECT-TYPE", "sysDescr"))

		imports := ctx.ModuleImports[importing]
		if len(imports) != 1 {
			t.Fatalf("import count = %d, want 1", len(imports))
		}
		if imports["sysDescr"] != source {
			t.Error("sysDescr should resolve to SOURCE-MIB")
		}
	})

	t.Run("alias resolution", func(t *testing.T) {
		ctx := newTestContextWithConfig(mib.DefaultConfig())
		// Source module is under the canonical name
		source := makeTestModule(ctx, "SNMPv2-SMI", []string{"enterprises", "Counter32"})
		ctx.ModuleIndex["SNMPv2-SMI"] = []*module.Module{source}
		// Import uses the alias name (no candidates under alias)

		importing := &module.Module{Name: "IMPORTER"}
		resolveImportsFromModule(ctx, importing, "SNMPv2-SMI-v1",
			syms("enterprises", "Counter32"))

		imports := ctx.ModuleImports[importing]
		if len(imports) != 2 {
			t.Fatalf("import count = %d, want 2", len(imports))
		}
		if imports["enterprises"] != source {
			t.Error("enterprises should resolve via alias to SNMPv2-SMI")
		}
	})

	t.Run("alias disabled in strict mode", func(t *testing.T) {
		ctx := newTestContextWithConfig(mib.StrictConfig())
		source := makeTestModule(ctx, "SNMPv2-SMI", []string{"enterprises"})
		ctx.ModuleIndex["SNMPv2-SMI"] = []*module.Module{source}

		importing := &module.Module{Name: "IMPORTER"}
		resolveImportsFromModule(ctx, importing, "SNMPv2-SMI-v1",
			syms("enterprises"))

		if len(ctx.ModuleImports[importing]) != 0 {
			t.Error("alias should not be used in strict mode")
		}
		if len(ctx.unresolvedImports) != 1 {
			t.Errorf("expected 1 unresolved, got %d", len(ctx.unresolvedImports))
		}
	})

	t.Run("forwarding resolution", func(t *testing.T) {
		ctx := newTestContextWithConfig(mib.DefaultConfig())

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
		if len(imports) != 2 {
			t.Fatalf("import count = %d, want 2", len(imports))
		}
		if imports["localDef"] != intermediate {
			t.Error("localDef should come from INTERMEDIATE")
		}
		if imports["remoteSym"] != realSource {
			t.Error("remoteSym should be forwarded from REAL-SOURCE")
		}
	})

	t.Run("partial resolution", func(t *testing.T) {
		ctx := newTestContextWithConfig(mib.DefaultConfig())

		// Candidate module only has some of the symbols
		source := makeTestModule(ctx, "PARTIAL-MIB", []string{"found1", "found2"})
		ctx.ModuleIndex["PARTIAL-MIB"] = []*module.Module{source}

		importing := &module.Module{Name: "IMPORTER"}
		resolveImportsFromModule(ctx, importing, "PARTIAL-MIB",
			syms("found1", "found2", "missing1"))

		imports := ctx.ModuleImports[importing]
		if len(imports) != 2 {
			t.Fatalf("import count = %d, want 2", len(imports))
		}
		if imports["found1"] != source || imports["found2"] != source {
			t.Error("found symbols should resolve to PARTIAL-MIB")
		}
		if len(ctx.unresolvedImports) != 1 {
			t.Fatalf("unresolved count = %d, want 1", len(ctx.unresolvedImports))
		}
		if ctx.unresolvedImports[0].symbol != "missing1" {
			t.Errorf("unresolved symbol = %q, want %q", ctx.unresolvedImports[0].symbol, "missing1")
		}
		if ctx.unresolvedImports[0].reason != reasonSymbolNotExported {
			t.Errorf("unresolved reason = %q, want %q", ctx.unresolvedImports[0].reason, reasonSymbolNotExported)
		}
	})

	t.Run("module not found", func(t *testing.T) {
		ctx := newTestContext()
		importing := &module.Module{Name: "IMPORTER"}
		resolveImportsFromModule(ctx, importing, "NONEXISTENT-MIB",
			syms("something"))

		if len(ctx.ModuleImports[importing]) != 0 {
			t.Error("expected no imports when module is not found")
		}
		if len(ctx.unresolvedImports) != 1 {
			t.Fatalf("unresolved count = %d, want 1", len(ctx.unresolvedImports))
		}
		if ctx.unresolvedImports[0].reason != reasonModuleNotFound {
			t.Errorf("reason = %q, want %q", ctx.unresolvedImports[0].reason, reasonModuleNotFound)
		}
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
		ctx := newTestContextWithConfig(mib.StrictConfig())
		// Candidate exists but doesn't have the symbol
		source := makeTestModule(ctx, "SRC", []string{"other"})
		ctx.ModuleIndex["SRC"] = []*module.Module{source}

		importing := &module.Module{Name: "IMPORTER"}
		resolveImportsFromModule(ctx, importing, "SRC",
			syms("missing"))

		// Strict mode disallows fallbacks, so it falls through to module_not_found
		if len(ctx.ModuleImports[importing]) != 0 {
			t.Error("expected no imports in strict mode with missing symbol")
		}
		if len(ctx.unresolvedImports) != 1 {
			t.Fatalf("unresolved count = %d, want 1", len(ctx.unresolvedImports))
		}
		if ctx.unresolvedImports[0].reason != reasonModuleNotFound {
			t.Errorf("reason = %q, want %q", ctx.unresolvedImports[0].reason, reasonModuleNotFound)
		}
	})

	t.Run("forwarding disabled in strict mode", func(t *testing.T) {
		ctx := newTestContextWithConfig(mib.StrictConfig())

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

		if len(ctx.ModuleImports[importing]) != 0 {
			t.Error("forwarding should not be used in strict mode")
		}
	})
}

func TestResolveTransitiveImports(t *testing.T) {
	t.Run("direct definer unchanged", func(t *testing.T) {
		ctx := newTestContext()
		modA := &module.Module{Name: "A"}
		modB := &module.Module{Name: "B"}

		ctx.ModuleDefNames[modB] = map[string]struct{}{"x": {}}
		ctx.RegisterImport(modA, "x", modB)

		resolveTransitiveImports(ctx)

		if ctx.ModuleImports[modA]["x"] != modB {
			t.Error("direct definer should remain unchanged")
		}
	})

	t.Run("one-hop re-export resolved", func(t *testing.T) {
		ctx := newTestContext()
		modA := &module.Module{Name: "A"}
		modB := &module.Module{Name: "B"}
		modC := &module.Module{Name: "C"}

		// B re-exports x from C; C defines x.
		ctx.ModuleDefNames[modB] = map[string]struct{}{}
		ctx.ModuleDefNames[modC] = map[string]struct{}{"x": {}}
		ctx.RegisterImport(modA, "x", modB)
		ctx.RegisterImport(modB, "x", modC)

		resolveTransitiveImports(ctx)

		if ctx.ModuleImports[modA]["x"] != modC {
			t.Error("expected A's import of x to resolve transitively to C")
		}
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
		ctx.RegisterImport(modA, "x", modB)
		ctx.RegisterImport(modB, "x", modC)
		ctx.RegisterImport(modC, "x", modD)

		resolveTransitiveImports(ctx)

		if ctx.ModuleImports[modA]["x"] != modD {
			t.Errorf("expected A->D, got A->%s", ctx.ModuleImports[modA]["x"].Name)
		}
		if ctx.ModuleImports[modB]["x"] != modD {
			t.Errorf("expected B->D, got B->%s", ctx.ModuleImports[modB]["x"].Name)
		}
		if ctx.ModuleImports[modC]["x"] != modD {
			t.Errorf("expected C->D, got C->%s", ctx.ModuleImports[modC]["x"].Name)
		}
	})

	t.Run("cycle does not panic", func(t *testing.T) {
		ctx := newTestContext()
		modA := &module.Module{Name: "A"}
		modB := &module.Module{Name: "B"}

		ctx.ModuleDefNames[modA] = map[string]struct{}{}
		ctx.ModuleDefNames[modB] = map[string]struct{}{}
		ctx.RegisterImport(modA, "x", modB)
		ctx.RegisterImport(modB, "x", modA)

		// Should not panic or infinite loop.
		resolveTransitiveImports(ctx)
	})

	t.Run("dead end preserved", func(t *testing.T) {
		ctx := newTestContext()
		modA := &module.Module{Name: "A"}
		modB := &module.Module{Name: "B"}

		// B neither defines x nor imports it.
		ctx.ModuleDefNames[modB] = map[string]struct{}{}
		ctx.RegisterImport(modA, "x", modB)

		resolveTransitiveImports(ctx)

		if ctx.ModuleImports[modA]["x"] != modB {
			t.Error("dead end should keep the original target")
		}
	})

	t.Run("different symbols resolve independently", func(t *testing.T) {
		ctx := newTestContext()
		modA := &module.Module{Name: "A"}
		modB := &module.Module{Name: "B"}
		modC := &module.Module{Name: "C"}

		// B defines "y" but re-exports "x" from C.
		ctx.ModuleDefNames[modB] = map[string]struct{}{"y": {}}
		ctx.ModuleDefNames[modC] = map[string]struct{}{"x": {}}
		ctx.RegisterImport(modA, "x", modB)
		ctx.RegisterImport(modA, "y", modB)
		ctx.RegisterImport(modB, "x", modC)

		resolveTransitiveImports(ctx)

		if ctx.ModuleImports[modA]["x"] != modC {
			t.Error("x should resolve transitively to C")
		}
		if ctx.ModuleImports[modA]["y"] != modB {
			t.Error("y should stay at B (direct definer)")
		}
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

		ctx := newResolverContext([]*module.Module{importing}, nil, mib.DefaultConfig())
		ctx.ModuleIndex["SOURCE-MIB"] = []*module.Module{source}
		ctx.ModuleDefNames[source] = map[string]struct{}{
			"sysDescr": {},
			"sysName":  {},
		}

		resolveImports(ctx)

		imports := ctx.ModuleImports[importing]
		if len(imports) != 2 {
			t.Fatalf("import count = %d, want 2 (macros filtered)", len(imports))
		}
		if imports["sysDescr"] != source {
			t.Error("sysDescr should resolve to SOURCE-MIB")
		}
		if imports["sysName"] != source {
			t.Error("sysName should resolve to SOURCE-MIB")
		}
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

		ctx := newResolverContext([]*module.Module{importing}, nil, mib.DefaultConfig())
		ctx.ModuleIndex["MOD-A"] = []*module.Module{sourceA}
		ctx.ModuleIndex["MOD-B"] = []*module.Module{sourceB}
		ctx.ModuleDefNames[sourceA] = map[string]struct{}{"alpha": {}}
		ctx.ModuleDefNames[sourceB] = map[string]struct{}{"beta": {}}

		resolveImports(ctx)

		imports := ctx.ModuleImports[importing]
		if imports["alpha"] != sourceA {
			t.Error("alpha should resolve to MOD-A")
		}
		if imports["beta"] != sourceB {
			t.Error("beta should resolve to MOD-B")
		}
	})
}
