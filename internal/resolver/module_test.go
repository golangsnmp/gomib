package resolver

import (
	"testing"

	"github.com/golangsnmp/gomib/internal/model"
	"github.com/golangsnmp/gomib/internal/module"
	"github.com/golangsnmp/gomib/internal/testutil"
	"github.com/golangsnmp/gomib/internal/types"
)

func TestResolvePreservesImportMap(t *testing.T) {
	// Build two modules: SOURCE-MIB defines "srcObj" at {enterprises 99998 1},
	// and TEST-MIB imports "srcObj" from SOURCE-MIB.
	sourceMod := buildTestModule(testModuleConfig{
		name:        "SOURCE-MIB",
		language:    types.LanguageSMIv2,
		baseImport:  module.NewImport("SNMPv2-SMI", "enterprises", types.Span{}),
		includeRoot: true,
		rootName:    "srcRoot",
		rootParent:  "enterprises",
		rootArc:     99998,
	}, &module.ObjectType{
		DefBase: module.DefBase{Name: "srcObj"},
		Oid:     testOid("srcRoot", 1),
		Syntax:  &module.TypeSyntaxTypeRef{Name: "Integer32"},
		Access:  types.AccessReadOnly,
		Status:  types.StatusCurrent,
	})

	testMod := buildTestModule(testModuleConfig{
		name:       "TEST-MIB",
		language:   types.LanguageSMIv2,
		baseImport: module.NewImport("SNMPv2-SMI", "enterprises", types.Span{}),
		extraImports: []module.Import{
			module.NewImport("SOURCE-MIB", "srcObj", types.Span{}),
		},
		includeRoot: true,
		rootName:    "testRoot",
		rootParent:  "enterprises",
		rootArc:     99999,
	}, &module.ObjectType{
		DefBase: module.DefBase{Name: "testObj"},
		Oid:     testOid("testRoot", 1),
		Syntax:  &module.TypeSyntaxTypeRef{Name: "Integer32"},
		Access:  types.AccessReadOnly,
		Status:  types.StatusCurrent,
	})

	m := Resolve([]*module.Module{sourceMod, testMod}, nil, nil, nil)

	resolvedTest := m.Module("TEST-MIB")
	testutil.NotNil(t, resolvedTest, "TEST-MIB not found")

	resolvedSource := m.Module("SOURCE-MIB")
	testutil.NotNil(t, resolvedSource, "SOURCE-MIB not found")

	t.Run("ImportSource returns correct module", func(t *testing.T) {
		src := resolvedTest.ImportSource("srcObj")
		testutil.NotNil(t, src, "ImportSource(srcObj) should not be nil")
		testutil.Equal(t, "SOURCE-MIB", src.Name(), "imported srcObj should come from SOURCE-MIB")
	})

	t.Run("ImportSource returns nil for own def", func(t *testing.T) {
		src := resolvedTest.ImportSource("testObj")
		testutil.Nil(t, src, "ImportSource(testObj) should be nil for own definition")
	})

	t.Run("AvailableSymbols includes own and imported", func(t *testing.T) {
		found := make(map[string]bool)
		for sym := range resolvedTest.AvailableSymbols() {
			found[sym.Name()] = true
		}
		testutil.True(t, found["testObj"], "AvailableSymbols should include own def testObj")
		testutil.True(t, found["srcObj"], "AvailableSymbols should include imported srcObj")
		testutil.True(t, found["testRoot"], "AvailableSymbols should include own node testRoot")
	})

	t.Run("Symbol lookup works for own defs", func(t *testing.T) {
		sym := resolvedTest.Symbol("testObj")
		testutil.False(t, sym.IsZero(), "Symbol(testObj) should not be zero")
		testutil.Equal(t, "testObj", sym.Name(), "name")
	})

	t.Run("ExportedSymbols matches Definitions", func(t *testing.T) {
		var exported, defs []string
		for sym := range resolvedTest.ExportedSymbols() {
			exported = append(exported, sym.Name())
		}
		for sym := range resolvedTest.Definitions() {
			defs = append(defs, sym.Name())
		}
		testutil.Equal(t, len(defs), len(exported), "length mismatch")
	})
}

func TestModulesDefiningCrossModule(t *testing.T) {
	// Two modules define the same name "sharedName".
	mod1 := buildTestModule(testModuleConfig{
		name:        "MOD-A",
		language:    types.LanguageSMIv2,
		baseImport:  module.NewImport("SNMPv2-SMI", "enterprises", types.Span{}),
		includeRoot: true,
		rootName:    "rootA",
		rootParent:  "enterprises",
		rootArc:     99990,
	}, &module.ObjectType{
		DefBase: module.DefBase{Name: "sharedName"},
		Oid:     testOid("rootA", 1),
		Syntax:  &module.TypeSyntaxTypeRef{Name: "Integer32"},
		Access:  types.AccessReadOnly,
		Status:  types.StatusCurrent,
	})
	mod2 := buildTestModule(testModuleConfig{
		name:        "MOD-B",
		language:    types.LanguageSMIv2,
		baseImport:  module.NewImport("SNMPv2-SMI", "enterprises", types.Span{}),
		includeRoot: true,
		rootName:    "rootB",
		rootParent:  "enterprises",
		rootArc:     99991,
	}, &module.ObjectType{
		DefBase: module.DefBase{Name: "sharedName"},
		Oid:     testOid("rootB", 1),
		Syntax:  &module.TypeSyntaxTypeRef{Name: "Integer32"},
		Access:  types.AccessReadOnly,
		Status:  types.StatusCurrent,
	})

	cfg := model.VerboseConfig()
	m := Resolve([]*module.Module{mod1, mod2}, nil, nil, &cfg)

	mods := m.ModulesDefining("sharedName")
	testutil.Equal(t, 2, len(mods), "ModulesDefining count")
}
