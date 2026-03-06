package mib

import (
	"testing"

	"github.com/golangsnmp/gomib/internal/testutil"
)

// buildSymbolTestMib creates a Mib with multiple entity types for testing.
// Module "TEST-MIB" (non-base) contains:
//   - ifDescr: Object at 1.3.6.1.2.1.2.2.1.2
//   - linkDown: Notification at 1.3.6.1.6.3.1.1.5.3
//   - ifGroup: Group at 1.3.6.1.2.1.31.1.2.1
//   - ifCompliance: Compliance at 1.3.6.1.2.1.31.2.1.1
//   - ifCap: Capability at 1.3.6.1.2.1.31.3.1
//   - system: plain Node at 1.3.6.1.2.1.1 (no entity)
//   - DisplayString: Type
func buildSymbolTestMib() *Mib {
	m := newMib()

	mod := newModule("TEST-MIB")
	m.addModule(mod)

	makeNode := func(arcs []uint32, name string) *Node {
		nd := m.root
		for _, arc := range arcs {
			nd = nd.getOrCreateChild(arc)
		}
		nd.setName(name)
		nd.setModule(mod)
		nd.setSpan(Span{Start: 10, End: 20})
		m.registerNode(name, nd)
		mod.addNode(nd)
		return nd
	}

	// Object
	objNode := makeNode([]uint32{1, 3, 6, 1, 2, 1, 2, 2, 1, 2}, "ifDescr")
	obj := newObject("ifDescr")
	obj.setModule(mod)
	obj.setNode(objNode)
	obj.setSpan(Span{Start: 100, End: 200})
	objNode.setObject(obj)
	mod.addObject(obj)
	m.addObject(obj)

	// Notification
	notifNode := makeNode([]uint32{1, 3, 6, 1, 6, 3, 1, 1, 5, 3}, "linkDown")
	notif := &Notification{entity: entity{name: "linkDown", module: mod, node: notifNode, span: Span{Start: 300, End: 400}}}
	notifNode.setNotification(notif)
	mod.addNotification(notif)
	m.addNotification(notif)

	// Group
	grpNode := makeNode([]uint32{1, 3, 6, 1, 2, 1, 31, 1, 2, 1}, "ifGroup")
	grp := &Group{entity: entity{name: "ifGroup", module: mod, node: grpNode, span: Span{Start: 500, End: 600}}}
	grpNode.setGroup(grp)
	mod.addGroup(grp)
	m.addGroup(grp)

	// Compliance
	compNode := makeNode([]uint32{1, 3, 6, 1, 2, 1, 31, 2, 1, 1}, "ifCompliance")
	comp := &Compliance{entity: entity{name: "ifCompliance", module: mod, node: compNode, span: Span{Start: 700, End: 800}}}
	compNode.setCompliance(comp)
	mod.addCompliance(comp)
	m.addCompliance(comp)

	// Capability
	capNode := makeNode([]uint32{1, 3, 6, 1, 2, 1, 31, 3, 1}, "ifCap")
	cap := &Capability{entity: entity{name: "ifCap", module: mod, node: capNode, span: Span{Start: 900, End: 1000}}}
	capNode.setCapability(cap)
	mod.addCapability(cap)
	m.addCapability(cap)

	// Plain node (no entity)
	makeNode([]uint32{1, 3, 6, 1, 2, 1, 1}, "system")

	// Type
	typ := newType("DisplayString")
	typ.setModule(mod)
	typ.setSpan(Span{Start: 1100, End: 1200})
	mod.addType(typ)
	m.addType(typ)

	return m
}

func TestSymbolLookupObject(t *testing.T) {
	m := buildSymbolTestMib()
	sym := m.Symbol("ifDescr")
	testutil.False(t, sym.IsZero(), "should not be zero")
	testutil.Equal(t, "ifDescr", sym.Name(), "Name")
	testutil.NotNil(t, sym.Object(), "Object")
	testutil.Nil(t, sym.Notification(), "Notification")
	testutil.Nil(t, sym.Type(), "Type")
	testutil.NotNil(t, sym.Node(), "Node")
	testutil.NotNil(t, sym.OID(), "OID")
	testutil.NotNil(t, sym.Module(), "Module")
	testutil.Equal(t, "TEST-MIB", sym.Module().Name(), "Module name")
	testutil.Equal(t, Span{Start: 100, End: 200}, sym.Span(), "Span")
}

func TestSymbolLookupNotification(t *testing.T) {
	m := buildSymbolTestMib()
	sym := m.Symbol("linkDown")
	testutil.False(t, sym.IsZero(), "should not be zero")
	testutil.Equal(t, "linkDown", sym.Name(), "Name")
	testutil.NotNil(t, sym.Notification(), "Notification")
	testutil.Nil(t, sym.Object(), "Object")
	testutil.NotNil(t, sym.Node(), "Node")
}

func TestSymbolLookupGroup(t *testing.T) {
	m := buildSymbolTestMib()
	sym := m.Symbol("ifGroup")
	testutil.False(t, sym.IsZero(), "should not be zero")
	testutil.Equal(t, "ifGroup", sym.Name(), "Name")
	testutil.NotNil(t, sym.Group(), "Group")
	testutil.Nil(t, sym.Object(), "Object")
}

func TestSymbolLookupCompliance(t *testing.T) {
	m := buildSymbolTestMib()
	sym := m.Symbol("ifCompliance")
	testutil.False(t, sym.IsZero(), "should not be zero")
	testutil.Equal(t, "ifCompliance", sym.Name(), "Name")
	testutil.NotNil(t, sym.Compliance(), "Compliance")
	testutil.Nil(t, sym.Object(), "Object")
}

func TestSymbolLookupCapability(t *testing.T) {
	m := buildSymbolTestMib()
	sym := m.Symbol("ifCap")
	testutil.False(t, sym.IsZero(), "should not be zero")
	testutil.Equal(t, "ifCap", sym.Name(), "Name")
	testutil.NotNil(t, sym.Capability(), "Capability")
	testutil.Nil(t, sym.Object(), "Object")
}

func TestSymbolLookupPlainNode(t *testing.T) {
	m := buildSymbolTestMib()
	sym := m.Symbol("system")
	testutil.False(t, sym.IsZero(), "should not be zero")
	testutil.Equal(t, "system", sym.Name(), "Name")
	testutil.Nil(t, sym.Object(), "Object")
	testutil.Nil(t, sym.Notification(), "Notification")
	testutil.Nil(t, sym.Group(), "Group")
	testutil.Nil(t, sym.Compliance(), "Compliance")
	testutil.Nil(t, sym.Capability(), "Capability")
	testutil.Nil(t, sym.Type(), "Type")
	testutil.NotNil(t, sym.Node(), "Node")
	testutil.NotNil(t, sym.OID(), "OID")
}

func TestSymbolLookupType(t *testing.T) {
	m := buildSymbolTestMib()
	sym := m.Symbol("DisplayString")
	testutil.False(t, sym.IsZero(), "should not be zero")
	testutil.Equal(t, "DisplayString", sym.Name(), "Name")
	testutil.NotNil(t, sym.Type(), "Type")
	testutil.Nil(t, sym.Object(), "Object")
	testutil.Nil(t, sym.Node(), "Node for Type symbol")
	testutil.Nil(t, sym.OID(), "OID for Type symbol")
	testutil.Equal(t, Span{Start: 1100, End: 1200}, sym.Span(), "Span")
}

func TestSymbolLookupNotFound(t *testing.T) {
	m := buildSymbolTestMib()
	sym := m.Symbol("nonexistent")
	testutil.True(t, sym.IsZero(), "should be zero")
	testutil.Equal(t, "", sym.Name(), "Name")
	testutil.Equal(t, Span{}, sym.Span(), "Span")
	testutil.Nil(t, sym.Module(), "Module")
	testutil.Nil(t, sym.Node(), "Node")
	testutil.Nil(t, sym.OID(), "OID")
}

func TestSymbolPriorityObjectOverNode(t *testing.T) {
	// When a name has both an object and a plain node, Symbol should return the object.
	m := newMib()
	mod := newModule("X-MIB")
	m.addModule(mod)

	bare := &Node{name: "foo"}
	withObj := &Node{name: "foo", obj: &Object{entity: entity{name: "foo", module: mod}}}
	m.registerNode("foo", bare)
	m.registerNode("foo", withObj)

	sym := m.Symbol("foo")
	testutil.NotNil(t, sym.Object(), "should prefer Object")
}

func TestModuleDefinitions(t *testing.T) {
	m := buildSymbolTestMib()
	mod := m.Module("TEST-MIB")
	testutil.NotNil(t, mod, "module")

	var names []string
	seen := make(map[string]bool)
	for sym := range mod.Definitions() {
		name := sym.Name()
		if seen[name] {
			t.Fatalf("duplicate definition: %s", name)
		}
		seen[name] = true
		names = append(names, name)
	}

	// Check all entity types are represented.
	testutil.True(t, seen["ifDescr"], "should contain Object ifDescr")
	testutil.True(t, seen["DisplayString"], "should contain Type DisplayString")
	testutil.True(t, seen["linkDown"], "should contain Notification linkDown")
	testutil.True(t, seen["ifGroup"], "should contain Group ifGroup")
	testutil.True(t, seen["ifCompliance"], "should contain Compliance ifCompliance")
	testutil.True(t, seen["ifCap"], "should contain Capability ifCap")
	testutil.True(t, seen["system"], "should contain plain Node system")

	// Nodes with entity attachments should NOT appear as plain nodes.
	// Count how many times ifDescr appears.
	count := 0
	for _, n := range names {
		if n == "ifDescr" {
			count++
		}
	}
	testutil.Equal(t, 1, count, "ifDescr should appear exactly once")
}

func TestModuleDefinitionsEarlyBreak(t *testing.T) {
	m := buildSymbolTestMib()
	mod := m.Module("TEST-MIB")

	// Verify the iterator can be broken early without panic.
	count := 0
	for range mod.Definitions() {
		count++
		if count >= 2 {
			break
		}
	}
	testutil.True(t, count >= 2, "should have iterated at least 2")
}

func TestModuleDefinesSymbol(t *testing.T) {
	m := buildSymbolTestMib()
	mod := m.Module("TEST-MIB")

	testutil.True(t, mod.DefinesSymbol("ifDescr"), "object")
	testutil.True(t, mod.DefinesSymbol("DisplayString"), "type")
	testutil.True(t, mod.DefinesSymbol("linkDown"), "notification")
	testutil.True(t, mod.DefinesSymbol("ifGroup"), "group")
	testutil.True(t, mod.DefinesSymbol("ifCompliance"), "compliance")
	testutil.True(t, mod.DefinesSymbol("ifCap"), "capability")
	testutil.True(t, mod.DefinesSymbol("system"), "plain node")
	testutil.False(t, mod.DefinesSymbol("nonexistent"), "nonexistent")
}

func TestModulesDefining(t *testing.T) {
	m := newMib()

	mod1 := newModule("MOD-A")
	mod1.addObject(newObject("sharedName"))
	m.addModule(mod1)

	mod2 := newModule("MOD-B")
	mod2.addObject(newObject("sharedName"))
	m.addModule(mod2)

	// Base module should be excluded.
	baseMod := newModule("BASE-MOD")
	baseMod.setBase(true)
	baseMod.addObject(newObject("sharedName"))
	m.addModule(baseMod)

	result := m.ModulesDefining("sharedName")
	testutil.Len(t, result, 2, "should find 2 non-base modules")

	names := make(map[string]bool)
	for _, mod := range result {
		names[mod.Name()] = true
	}
	testutil.True(t, names["MOD-A"], "should include MOD-A")
	testutil.True(t, names["MOD-B"], "should include MOD-B")
	testutil.False(t, names["BASE-MOD"], "should exclude base module")
}

func TestModulesDefiningNotFound(t *testing.T) {
	m := buildSymbolTestMib()
	result := m.ModulesDefining("nonexistent")
	testutil.Len(t, result, 0, "should find no modules")
}

func TestModuleImportsSymbol(t *testing.T) {
	mod := newModule("TEST-MIB")
	mod.setImports([]Import{
		{Module: "SNMPv2-SMI", Symbols: []ImportSymbol{{Name: "enterprises"}, {Name: "Counter32"}}},
		{Module: "SNMPv2-TC", Symbols: []ImportSymbol{{Name: "DisplayString"}}},
	})

	testutil.True(t, mod.ImportsSymbol("enterprises"), "enterprises")
	testutil.True(t, mod.ImportsSymbol("Counter32"), "Counter32")
	testutil.True(t, mod.ImportsSymbol("DisplayString"), "DisplayString")
	testutil.False(t, mod.ImportsSymbol("nonexistent"), "nonexistent")

	empty := newModule("EMPTY-MIB")
	testutil.False(t, empty.ImportsSymbol("anything"), "no imports")
}

func TestModulesImporting(t *testing.T) {
	m := newMib()

	mod1 := newModule("MOD-A")
	mod1.setImports([]Import{
		{Module: "SNMPv2-SMI", Symbols: []ImportSymbol{{Name: "enterprises"}, {Name: "Counter32"}}},
	})
	m.addModule(mod1)

	mod2 := newModule("MOD-B")
	mod2.setImports([]Import{
		{Module: "SNMPv2-SMI", Symbols: []ImportSymbol{{Name: "enterprises"}}},
		{Module: "SNMPv2-TC", Symbols: []ImportSymbol{{Name: "DisplayString"}}},
	})
	m.addModule(mod2)

	// Base module should be excluded.
	baseMod := newModule("BASE-MOD")
	baseMod.setBase(true)
	baseMod.setImports([]Import{
		{Module: "SNMPv2-SMI", Symbols: []ImportSymbol{{Name: "enterprises"}}},
	})
	m.addModule(baseMod)

	result := m.ModulesImporting("enterprises")
	testutil.Len(t, result, 2, "should find 2 non-base modules importing enterprises")
	names := make(map[string]bool)
	for _, mod := range result {
		names[mod.Name()] = true
	}
	testutil.True(t, names["MOD-A"], "should include MOD-A")
	testutil.True(t, names["MOD-B"], "should include MOD-B")
	testutil.False(t, names["BASE-MOD"], "should exclude base module")

	result = m.ModulesImporting("DisplayString")
	testutil.Len(t, result, 1, "only MOD-B imports DisplayString")
	testutil.Equal(t, result[0].Name(), "MOD-B", "module name")

	result = m.ModulesImporting("nonexistent")
	testutil.Len(t, result, 0, "should find no modules")
}

func TestModuleIsImportUsed(t *testing.T) {
	mod := newModule("TEST-MIB")
	mod.setUsedImportNames(map[string]struct{}{
		"ifIndex":   {},
		"Counter32": {},
	})

	testutil.True(t, mod.IsImportUsed("ifIndex"), "used import")
	testutil.True(t, mod.IsImportUsed("Counter32"), "used import")
	testutil.False(t, mod.IsImportUsed("DisplayString"), "unused import")
	testutil.False(t, mod.IsImportUsed("nonexistent"), "unknown name")
}

func TestModuleIsImportUsedNilMap(t *testing.T) {
	mod := newModule("TEST-MIB")
	// No usedImportNames set.
	testutil.False(t, mod.IsImportUsed("anything"), "nil map")
}
