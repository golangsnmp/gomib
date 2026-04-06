package resolver

import (
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/golangsnmp/gomib/internal/cst/lower"
	cstparser "github.com/golangsnmp/gomib/internal/cst/parser"
	"github.com/golangsnmp/gomib/internal/model"
	"github.com/golangsnmp/gomib/internal/module"
	"github.com/golangsnmp/gomib/internal/types"
)

// testBaseModules returns parsed base modules for use in resolver tests.
// Parsed once and cached.
var testBaseModules = sync.OnceValue(func() []*module.Module {
	var result []*module.Module
	cfg := types.DefaultConfig()
	for _, name := range module.BaseModuleNames() {
		content := module.EmbeddedContent(name)
		if content == nil {
			continue
		}
		p := cstparser.New(content, nil, cfg)
		file := p.ParseModule()
		mods := lower.Lower(file, content, p.Diagnostics(), cfg)
		for _, mod := range mods {
			if mod.Name == name {
				result = append(result, mod)
				break
			}
		}
	}
	return result
})

func newTestContext() *resolverContext {
	return newResolverContext(nil, model.ResolverNormal, model.DefaultConfig())
}

func newTestContextWithConfig(config model.DiagnosticConfig) *resolverContext {
	return newResolverContext(nil, model.ResolverNormal, config)
}

func newTestContextWithPolicy(strictness model.ResolverStrictness, config model.DiagnosticConfig) *resolverContext {
	return newResolverContext(nil, strictness, config)
}

func newTestContextForModules(config model.DiagnosticConfig, mods ...*module.Module) *resolverContext {
	ctx := newResolverContext(nil, model.ResolverNormal, config)
	setTestModules(ctx, mods)
	for _, mod := range mods {
		ctx.moduleIndex[mod.Name] = append(ctx.moduleIndex[mod.Name], mod)
		resolvedMod := model.NewModule(mod.Name)
		ctx.moduleToResolved[mod] = resolvedMod
		ctx.resolvedToModule[resolvedMod] = mod
	}
	return ctx
}

func newTestContextForModulesWithPolicy(strictness model.ResolverStrictness, config model.DiagnosticConfig, mods ...*module.Module) *resolverContext {
	ctx := newResolverContext(nil, strictness, config)
	setTestModules(ctx, mods)
	for _, mod := range mods {
		ctx.moduleIndex[mod.Name] = append(ctx.moduleIndex[mod.Name], mod)
		resolvedMod := model.NewModule(mod.Name)
		ctx.moduleToResolved[mod] = resolvedMod
		ctx.resolvedToModule[resolvedMod] = mod
	}
	return ctx
}

// setTestModules sets ctx.modules and builds the oidDefNames cache
// for tests that don't go through registerModules.
func setTestModules(ctx *resolverContext, mods []*module.Module) {
	ctx.modules = mods
	for _, mod := range mods {
		if _, exists := ctx.oidDefNames[mod]; exists {
			continue
		}
		oidDefs := make(map[string]struct{})
		for def := range mod.AllDefinitions() {
			if def.DefinitionOid() != nil {
				oidDefs[def.DefinitionName()] = struct{}{}
			}
		}
		ctx.oidDefNames[mod] = oidDefs
	}
}

// buildOIDPath chains getOrCreateChild calls to build a node path from root.
// For example, buildOIDPath(root, 1, 3, 6, 1) builds iso.org.dod.internet.
func buildOIDPath(root *model.Node, arcs ...uint32) *model.Node {
	n := root
	for _, arc := range arcs {
		n = model.GetOrCreateChild(n, arc)
	}
	return n
}

// resolveWithBase prepends embedded base modules and resolves with
// the given config. Mirrors what the load pipeline does.
func resolveWithBase(mods []*module.Module, strictness *model.ResolverStrictness, diagConfig *model.DiagnosticConfig) *model.Mib {
	all := append(testBaseModules(), mods...)
	return Resolve(all, nil, strictness, diagConfig)
}

func resolveStrict(mods ...*module.Module) *model.Mib {
	cfg := model.VerboseConfig()
	strictness := model.ResolverStrict
	return resolveWithBase(mods, &strictness, &cfg)
}

// testOid builds an OID assignment as { parentName subid }.
func testOid(parentName string, subid uint32) module.OidAssignment {
	return module.NewOidAssignment([]module.OidComponent{
		{Name: parentName},
		{Number: subid, HasNumber: true},
	}, types.Span{})
}

// newComplianceTestContext creates a resolverContext with strict config,
// registering each module in index and resolved maps.
func newComplianceTestContext(mods ...*module.Module) *resolverContext {
	return newTestContextForModules(model.VerboseConfig(), mods...)
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

// addDefs distributes a flat Definition slice into the typed slices on mod.
func addDefs(mod *module.Module, defs []module.Definition) {
	for _, def := range defs {
		switch d := def.(type) {
		case *module.ObjectType:
			mod.ObjectTypes = append(mod.ObjectTypes, d)
		case *module.TypeDef:
			mod.TypeDefs = append(mod.TypeDefs, d)
		case *module.Notification:
			mod.Notifications = append(mod.Notifications, d)
		case *module.ModuleIdentity:
			mod.ModuleIdentities = append(mod.ModuleIdentities, d)
		case *module.ObjectIdentity:
			mod.ObjectIdentities = append(mod.ObjectIdentities, d)
		case *module.ValueAssignment:
			mod.ValueAssignments = append(mod.ValueAssignments, d)
		case *module.ObjectGroup:
			mod.ObjectGroups = append(mod.ObjectGroups, d)
		case *module.NotificationGroup:
			mod.NotificationGroups = append(mod.NotificationGroups, d)
		case *module.ModuleCompliance:
			mod.Compliances = append(mod.Compliances, d)
		case *module.AgentCapabilities:
			mod.Capabilities = append(mod.Capabilities, d)
		}
	}
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

	mod := &module.Module{
		Name:     name,
		Language: cfg.language,
		Imports:  imports,
	}
	addDefs(mod, allDefs)
	return mod
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

	mod := &module.Module{
		Name:     "TEST-MIB",
		Language: types.LanguageSMIv2,
		Imports:  imports,
	}
	addDefs(mod, defs)
	return mod
}

// nameRefs converts string names to NameRef values with zero spans, for tests.
func nameRefs(names ...string) []module.NameRef {
	refs := make([]module.NameRef, len(names))
	for i, n := range names {
		refs[i] = module.NameRef{Name: n}
	}
	return refs
}

func unresolvedByKind(ctx *resolverContext, kind model.UnresolvedKind) []model.UnresolvedRef {
	var result []model.UnresolvedRef
	for _, u := range ctx.mib.Unresolved() {
		if u.Kind == kind {
			result = append(result, u)
		}
	}
	return result
}

func registerNodeWithSymbol(ctx *resolverContext, mod *module.Module, root *model.Node, symbol string, arcs ...uint32) *model.Node {
	node := buildOIDPath(root, arcs...)
	model.SetNodeName(node, symbol)
	ctx.registerModuleNodeSymbol(mod, symbol, node)
	return node
}

func hasDiag(t testing.TB, diags []model.Diagnostic, code string) model.Diagnostic {
	t.Helper()
	for _, d := range diags {
		if d.Code == code {
			return d
		}
	}
	t.Fatalf("expected diagnostic with code %q; got: %s", code, formatDiagnostics(diags))
	return model.Diagnostic{}
}

func noDiag(t testing.TB, diags []model.Diagnostic, codes ...string) {
	t.Helper()
	for _, code := range codes {
		if countDiagnostics(diags, code) != 0 {
			t.Fatalf("unexpected diagnostic with code %q; got: %s", code, formatDiagnostics(diags))
		}
	}
}

func countDiagnostics(diags []model.Diagnostic, code string) int {
	count := 0
	for _, d := range diags {
		if d.Code == code {
			count++
		}
	}
	return count
}

func diagsByCode(diags []model.Diagnostic, code string) []model.Diagnostic {
	matches := make([]model.Diagnostic, 0)
	for _, d := range diags {
		if d.Code == code {
			matches = append(matches, d)
		}
	}
	return matches
}

func requireDiagCount(t testing.TB, diags []model.Diagnostic, code string, want int) {
	t.Helper()
	got := countDiagnostics(diags, code)
	if got != want {
		t.Fatalf("diagnostic %q count mismatch: got=%d want=%d all=%s", code, got, want, formatDiagnostics(diags))
	}
}

func formatDiagnostics(diags []model.Diagnostic) string {
	if len(diags) == 0 {
		return "<none>"
	}
	parts := make([]string, 0, len(diags))
	for _, d := range diags {
		parts = append(parts, fmt.Sprintf("[%s] %s: %s", d.Module, d.Code, d.Message))
	}
	return strings.Join(parts, "; ")
}
