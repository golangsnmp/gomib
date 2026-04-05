package model

import (
	"testing"

	"github.com/golangsnmp/gomib/internal/module"
	"github.com/golangsnmp/gomib/internal/testutil"
)

func TestModuleSymbol(t *testing.T) {
	mod := NewModule("TEST-MIB")

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
	mod := NewModule("TEST-MIB")
	sourceMod := NewModule("SOURCE-MIB")

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
		m := NewModule("EMPTY")
		src := m.ImportSource("anything")
		testutil.Nil(t, src, "expected nil when resolvedImports is nil")
	})
}

func TestModuleAvailableSymbols(t *testing.T) {
	// Set up source module with some definitions.
	sourceMod := NewModule("SOURCE-MIB")
	srcObj := &Object{entity: entity{name: "srcObj", module: sourceMod}}
	sourceMod.addObject(srcObj)
	srcNode := &Node{name: "srcObj", obj: srcObj}
	sourceMod.addNode(srcNode)

	srcType := &Type{name: "SrcType", module: sourceMod}
	sourceMod.addType(srcType)

	// Set up the importing module.
	mod := NewModule("TEST-MIB")
	ownObj := &Object{entity: entity{name: "ownObj", module: mod}}
	mod.addObject(ownObj)
	ownNode := &Node{name: "ownObj", obj: ownObj}
	mod.addNode(ownNode)

	mod.setRawImports([]module.Import{
		module.NewImport("SOURCE-MIB", "srcObj", Span{}),
		module.NewImport("SOURCE-MIB", "SrcType", Span{}),
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
	sourceMod := NewModule("SOURCE-MIB")
	srcType := &Type{name: "sharedName", module: sourceMod}
	sourceMod.addType(srcType)

	// Importing module also defines "sharedName".
	mod := NewModule("TEST-MIB")
	ownType := &Type{name: "sharedName", module: mod}
	mod.addType(ownType)

	mod.setRawImports([]module.Import{
		module.NewImport("SOURCE-MIB", "sharedName", Span{}),
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
	mod := NewModule("TEST-MIB")
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
	mod := NewModule("TEST-MIB")
	obj := &Object{entity: entity{name: "onlyOwn", module: mod}}
	mod.addObject(obj)

	var names []string
	for sym := range mod.AvailableSymbols() {
		names = append(names, sym.Name())
	}

	testutil.Equal(t, 1, len(names), "expected 1 symbol")
	testutil.Equal(t, "onlyOwn", names[0], "symbol name")
}

// TestResolvePreservesImportMap moved to internal/resolver/module_test.go
