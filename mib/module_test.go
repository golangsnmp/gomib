package mib

import (
	"testing"

	"github.com/golangsnmp/gomib/internal/module"
	"github.com/golangsnmp/gomib/internal/testutil"
	"github.com/golangsnmp/gomib/internal/types"
)

func TestModuleSymbol(t *testing.T) {
	mod := newModule("TEST-MIB")

	obj := &Object{entity: entity{name: "testObj", module: mod}}
	mod.addObject(obj)
	nd := &Node{name: "testObj", obj: obj}
	mod.addNode(nd)

	typ := &Type{name: "TestType", module: mod}
	mod.addType(typ)

	notif := &Notification{entity: entity{name: "testNotif", module: mod}}
	mod.addNotification(notif)
	nNode := &Node{name: "testNotif", notif: notif}
	mod.addNode(nNode)

	plainNode := &Node{name: "plainNode"}
	mod.addNode(plainNode)

	t.Run("object", func(t *testing.T) {
		sym := mod.Symbol("testObj")
		testutil.False(t, sym.IsZero(), "expected non-zero Symbol for testObj")
		testutil.Equal(t, "testObj", sym.Name(), "name")
		testutil.NotNil(t, sym.Object(), "expected Object")
	})

	t.Run("type", func(t *testing.T) {
		sym := mod.Symbol("TestType")
		testutil.False(t, sym.IsZero(), "expected non-zero Symbol for TestType")
		testutil.Equal(t, "TestType", sym.Name(), "name")
		testutil.NotNil(t, sym.Type(), "expected Type")
	})

	t.Run("notification", func(t *testing.T) {
		sym := mod.Symbol("testNotif")
		testutil.False(t, sym.IsZero(), "expected non-zero Symbol for testNotif")
		testutil.Equal(t, "testNotif", sym.Name(), "name")
		testutil.NotNil(t, sym.Notification(), "expected Notification")
	})

	t.Run("plain node", func(t *testing.T) {
		sym := mod.Symbol("plainNode")
		testutil.False(t, sym.IsZero(), "expected non-zero Symbol for plainNode")
		testutil.Equal(t, "plainNode", sym.Name(), "name")
	})

	t.Run("unknown name", func(t *testing.T) {
		sym := mod.Symbol("nonexistent")
		testutil.True(t, sym.IsZero(), "expected zero Symbol for unknown name")
	})

	t.Run("object takes priority over node", func(t *testing.T) {
		// testObj is registered as both object and node; Symbol should return the object.
		sym := mod.Symbol("testObj")
		testutil.NotNil(t, sym.Object(), "expected Object, not plain Node")
	})
}

func TestModuleImportSource(t *testing.T) {
	mod := newModule("TEST-MIB")
	sourceMod := newModule("SOURCE-MIB")

	mod.setResolvedImports(map[string]*Module{
		"importedSym": sourceMod,
	})

	t.Run("imported symbol", func(t *testing.T) {
		src := mod.ImportSource("importedSym")
		testutil.NotNil(t, src, "expected non-nil source for imported symbol")
		testutil.Equal(t, "SOURCE-MIB", src.Name(), "source module name")
	})

	t.Run("own definition", func(t *testing.T) {
		src := mod.ImportSource("ownDef")
		testutil.Nil(t, src, "expected nil for non-imported name")
	})

	t.Run("unknown name", func(t *testing.T) {
		src := mod.ImportSource("nonexistent")
		testutil.Nil(t, src, "expected nil for unknown name")
	})

	t.Run("nil resolvedImports", func(t *testing.T) {
		m := newModule("EMPTY")
		src := m.ImportSource("anything")
		testutil.Nil(t, src, "expected nil when resolvedImports is nil")
	})
}

func TestModuleAvailableSymbols(t *testing.T) {
	// Set up source module with some definitions.
	sourceMod := newModule("SOURCE-MIB")
	srcObj := &Object{entity: entity{name: "srcObj", module: sourceMod}}
	sourceMod.addObject(srcObj)
	srcNode := &Node{name: "srcObj", obj: srcObj}
	sourceMod.addNode(srcNode)

	srcType := &Type{name: "SrcType", module: sourceMod}
	sourceMod.addType(srcType)

	// Set up the importing module.
	mod := newModule("TEST-MIB")
	ownObj := &Object{entity: entity{name: "ownObj", module: mod}}
	mod.addObject(ownObj)
	ownNode := &Node{name: "ownObj", obj: ownObj}
	mod.addNode(ownNode)

	mod.setImports([]Import{
		{
			Module: "SOURCE-MIB",
			Symbols: []ImportSymbol{
				{Name: "srcObj"},
				{Name: "SrcType"},
			},
		},
	})
	mod.setResolvedImports(map[string]*Module{
		"srcObj":  sourceMod,
		"SrcType": sourceMod,
	})

	var names []string
	for sym := range mod.AvailableSymbols() {
		names = append(names, sym.Name())
	}

	// Own definitions come first, then imports.
	testutil.True(t, len(names) >= 3, "expected at least 3 symbols, got %d", len(names))

	// ownObj should be first (own definition).
	testutil.Equal(t, "ownObj", names[0], "first symbol should be own definition")

	// Check that imported symbols are present.
	found := make(map[string]bool)
	for _, n := range names {
		found[n] = true
	}
	testutil.True(t, found["srcObj"], "expected srcObj in available symbols")
	testutil.True(t, found["SrcType"], "expected SrcType in available symbols")
}

func TestModuleAvailableSymbols_OwnDefPriority(t *testing.T) {
	// Source module defines "sharedName".
	sourceMod := newModule("SOURCE-MIB")
	srcType := &Type{name: "sharedName", module: sourceMod}
	sourceMod.addType(srcType)

	// Importing module also defines "sharedName".
	mod := newModule("TEST-MIB")
	ownType := &Type{name: "sharedName", module: mod}
	mod.addType(ownType)

	mod.setImports([]Import{
		{
			Module:  "SOURCE-MIB",
			Symbols: []ImportSymbol{{Name: "sharedName"}},
		},
	})
	mod.setResolvedImports(map[string]*Module{
		"sharedName": sourceMod,
	})

	var names []string
	for sym := range mod.AvailableSymbols() {
		names = append(names, sym.Name())
	}

	// "sharedName" should appear exactly once (as own definition).
	count := 0
	for _, n := range names {
		if n == "sharedName" {
			count++
		}
	}
	testutil.Equal(t, 1, count, "sharedName should appear exactly once")

	// And it should be the own module's definition.
	sym := mod.Symbol("sharedName")
	testutil.Equal(t, mod, sym.Module(), "sharedName should be from own module")
}

func TestModuleExportedSymbols(t *testing.T) {
	mod := newModule("TEST-MIB")
	obj := &Object{entity: entity{name: "testObj", module: mod}}
	mod.addObject(obj)
	nd := &Node{name: "testObj", obj: obj}
	mod.addNode(nd)

	typ := &Type{name: "TestType", module: mod}
	mod.addType(typ)

	// Collect from both ExportedSymbols and Definitions, verify they match.
	var exported []string
	for sym := range mod.ExportedSymbols() {
		exported = append(exported, sym.Name())
	}

	var defs []string
	for sym := range mod.Definitions() {
		defs = append(defs, sym.Name())
	}

	testutil.Equal(t, len(defs), len(exported), "ExportedSymbols and Definitions should have same length")
	for i := range defs {
		testutil.Equal(t, defs[i], exported[i], "symbol at index %d", i)
	}
}

func TestModuleAvailableSymbols_NoImports(t *testing.T) {
	mod := newModule("TEST-MIB")
	obj := &Object{entity: entity{name: "onlyOwn", module: mod}}
	mod.addObject(obj)

	var names []string
	for sym := range mod.AvailableSymbols() {
		names = append(names, sym.Name())
	}

	testutil.Equal(t, 1, len(names), "expected 1 symbol")
	testutil.Equal(t, "onlyOwn", names[0], "symbol name")
}

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
