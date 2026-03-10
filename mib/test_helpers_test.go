package mib

import (
	"fmt"
	"strings"
	"testing"

	"github.com/golangsnmp/gomib/internal/module"
	"github.com/golangsnmp/gomib/internal/types"
)

func newTestContext() *resolverContext {
	return newResolverContext(nil, nil, ResolverNormal, DefaultConfig())
}

func newTestContextWithConfig(config DiagnosticConfig) *resolverContext {
	return newResolverContext(nil, nil, ResolverNormal, config)
}

func newTestContextWithPolicy(strictness ResolverStrictness, config DiagnosticConfig) *resolverContext {
	return newResolverContext(nil, nil, strictness, config)
}

func newTestContextForModules(config DiagnosticConfig, mods ...*module.Module) *resolverContext {
	ctx := newResolverContext(mods, nil, ResolverNormal, config)
	for _, mod := range mods {
		ctx.moduleIndex[mod.Name] = append(ctx.moduleIndex[mod.Name], mod)
		resolvedMod := newModule(mod.Name)
		ctx.moduleToResolved[mod] = resolvedMod
		ctx.resolvedToModule[resolvedMod] = mod
	}
	return ctx
}

func newTestContextForModulesWithPolicy(strictness ResolverStrictness, config DiagnosticConfig, mods ...*module.Module) *resolverContext {
	ctx := newResolverContext(mods, nil, strictness, config)
	for _, mod := range mods {
		ctx.moduleIndex[mod.Name] = append(ctx.moduleIndex[mod.Name], mod)
		resolvedMod := newModule(mod.Name)
		ctx.moduleToResolved[mod] = resolvedMod
		ctx.resolvedToModule[resolvedMod] = mod
	}
	return ctx
}

// buildOIDPath chains getOrCreateChild calls to build a node path from root.
// For example, buildOIDPath(root, 1, 3, 6, 1) builds iso.org.dod.internet.
func buildOIDPath(root *Node, arcs ...uint32) *Node {
	n := root
	for _, arc := range arcs {
		n = n.getOrCreateChild(arc)
	}
	return n
}

func resolveStrict(mods ...*module.Module) *Mib {
	cfg := VerboseConfig()
	strictness := ResolverStrict
	return Resolve(mods, nil, &strictness, &cfg)
}

// testOid builds an OID assignment as { parentName subid }.
func testOid(parentName string, subid uint32) module.OidAssignment {
	return module.NewOidAssignment([]module.OidComponent{
		&module.OidComponentName{NameValue: parentName},
		&module.OidComponentNumber{Value: subid},
	}, types.Span{})
}

// newComplianceTestContext creates a resolverContext with strict config,
// registering each module in index and resolved maps.
func newComplianceTestContext(mods ...*module.Module) *resolverContext {
	return newTestContextForModules(VerboseConfig(), mods...)
}

type testModuleConfig struct {
	name         string
	language     types.Language
	baseImport   module.Import
	extraImports []module.Import
	includeRoot  bool
	rootName     string
	rootParent   string
	rootArc      uint32
}

func buildTestModule(cfg testModuleConfig, defs ...module.Definition) *module.Module {
	name := cfg.name
	if name == "" {
		name = "TEST-MIB"
	}
	rootName := cfg.rootName
	if rootName == "" {
		rootName = "testRoot"
	}
	rootParent := cfg.rootParent
	if rootParent == "" {
		rootParent = "enterprises"
	}
	rootArc := cfg.rootArc
	if rootArc == 0 {
		rootArc = 99999
	}

	imports := make([]module.Import, 0, 1+len(cfg.extraImports))
	if cfg.baseImport.Module != "" && cfg.baseImport.Symbol != "" {
		imports = append(imports, cfg.baseImport)
	}
	imports = append(imports, cfg.extraImports...)

	allDefs := make([]module.Definition, 0, len(defs)+1)
	if cfg.includeRoot {
		allDefs = append(allDefs, &module.ValueAssignment{
			DefBase: module.DefBase{Name: rootName},
			Oid:     testOid(rootParent, rootArc),
		})
	}
	allDefs = append(allDefs, defs...)

	return &module.Module{
		Name:        name,
		Language:    cfg.language,
		Imports:     imports,
		Definitions: allDefs,
	}
}

// testSMIv2Module builds a TEST-MIB with SMIv2 language, base enterprises import,
// and testRoot at { enterprises 99999 }. extraImports and defs are appended.
func testSMIv2Module(extraImports []module.Import, defs ...module.Definition) *module.Module {
	return buildTestModule(testModuleConfig{
		language:     types.LanguageSMIv2,
		baseImport:   module.NewImport("SNMPv2-SMI", "enterprises", types.Span{}),
		extraImports: extraImports,
		includeRoot:  true,
	}, defs...)
}

// testSMIv1Module builds a TEST-MIB with SMIv1 language, base enterprises import,
// and testRoot at { enterprises 99999 }. extraImports and defs are appended.
func testSMIv1Module(extraImports []module.Import, defs ...module.Definition) *module.Module {
	return buildTestModule(testModuleConfig{
		language:     types.LanguageSMIv1,
		baseImport:   module.NewImport("RFC1155-SMI", "enterprises", types.Span{}),
		extraImports: extraImports,
		includeRoot:  true,
	}, defs...)
}

// testTableModule builds a TEST-MIB with a standard table/entry/index skeleton.
// extraImports are appended to the base SNMPv2-SMI imports.
// columns are appended after the index definition (callers assign OIDs under testEntry).
func testTableModule(extraImports []module.Import, columns ...module.Definition) *module.Module {
	imports := []module.Import{
		module.NewImport("SNMPv2-SMI", "enterprises", types.Span{}),
		module.NewImport("SNMPv2-SMI", "OBJECT-TYPE", types.Span{}),
		module.NewImport("SNMPv2-SMI", "Integer32", types.Span{}),
	}
	imports = append(imports, extraImports...)

	defs := []module.Definition{
		&module.ValueAssignment{
			DefBase: module.DefBase{Name: "testRoot"},
			Oid:     testOid("enterprises", 99999),
		},
		&module.ObjectType{
			DefBase: module.DefBase{Name: "testTable"},
			Syntax:  &module.TypeSyntaxSequenceOf{EntryType: "TestEntry"},
			Status:  types.StatusCurrent,
			Oid:     testOid("testRoot", 1),
		},
		&module.ObjectType{
			DefBase: module.DefBase{Name: "testEntry"},
			Syntax:  &module.TypeSyntaxTypeRef{Name: "TestEntry"},
			Status:  types.StatusCurrent,
			Index:   []module.IndexItem{{Object: "testIndex"}},
			Oid:     testOid("testTable", 1),
		},
		&module.ObjectType{
			DefBase: module.DefBase{Name: "testIndex"},
			Syntax:  &module.TypeSyntaxTypeRef{Name: "Integer32"},
			Access:  types.AccessNotAccessible,
			Status:  types.StatusCurrent,
			Oid:     testOid("testEntry", 1),
		},
	}
	defs = append(defs, columns...)

	return &module.Module{
		Name:        "TEST-MIB",
		Language:    types.LanguageSMIv2,
		Imports:     imports,
		Definitions: defs,
	}
}

func registerNodeWithSymbol(ctx *resolverContext, mod *module.Module, root *Node, symbol string, arcs ...uint32) *Node {
	node := buildOIDPath(root, arcs...)
	node.setName(symbol)
	ctx.registerModuleNodeSymbol(mod, symbol, node)
	return node
}

func hasDiag(t testing.TB, diags []Diagnostic, code string) Diagnostic {
	t.Helper()
	for _, d := range diags {
		if d.Code == code {
			return d
		}
	}
	t.Fatalf("expected diagnostic with code %q; got: %s", code, formatDiagnostics(diags))
	return Diagnostic{}
}

func noDiag(t testing.TB, diags []Diagnostic, codes ...string) {
	t.Helper()
	for _, code := range codes {
		if countDiag(diags, code) != 0 {
			t.Fatalf("unexpected diagnostic with code %q; got: %s", code, formatDiagnostics(diags))
		}
	}
}

func countDiag(diags []Diagnostic, code string) int {
	count := 0
	for _, d := range diags {
		if d.Code == code {
			count++
		}
	}
	return count
}

func diagsByCode(diags []Diagnostic, code string) []Diagnostic {
	matches := make([]Diagnostic, 0)
	for _, d := range diags {
		if d.Code == code {
			matches = append(matches, d)
		}
	}
	return matches
}

func requireDiagCount(t testing.TB, diags []Diagnostic, code string, want int) {
	t.Helper()
	got := countDiag(diags, code)
	if got != want {
		t.Fatalf("diagnostic %q count mismatch: got=%d want=%d all=%s", code, got, want, formatDiagnostics(diags))
	}
}

func formatDiagnostics(diags []Diagnostic) string {
	if len(diags) == 0 {
		return "<none>"
	}
	parts := make([]string, 0, len(diags))
	for _, d := range diags {
		parts = append(parts, fmt.Sprintf("[%s] %s: %s", d.Module, d.Code, d.Message))
	}
	return strings.Join(parts, "; ")
}

func requireTypeDescriptionAndReference(t testing.TB, m *Mib, name string) *Type {
	t.Helper()
	typ := m.Type(name)
	if typ == nil {
		t.Fatalf("type %q not found", name)
	}
	if typ.Description() == "" {
		t.Fatalf("type %q has empty description", name)
	}
	if typ.Reference() == "" {
		t.Fatalf("type %q has empty reference", name)
	}
	return typ
}

func requireNodeDescription(t testing.TB, m *Mib, name string) *Node {
	t.Helper()
	node := m.Node(name)
	if node == nil {
		t.Fatalf("node %q not found", name)
	}
	if node.Description() == "" {
		t.Fatalf("node %q has empty description", name)
	}
	return node
}

func requireNodeReference(t testing.TB, m *Mib, name string) *Node {
	t.Helper()
	node := m.Node(name)
	if node == nil {
		t.Fatalf("node %q not found", name)
	}
	if node.Reference() == "" {
		t.Fatalf("node %q has empty reference", name)
	}
	return node
}
