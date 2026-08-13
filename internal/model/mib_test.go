package model

import (
	"testing"

	"github.com/golangsnmp/gomib/internal/testutil"
)

func TestGlobalEntityLookupsUseDefinitionNames(t *testing.T) {
	m := NewMib()
	legacyMod := NewModule("LEGACY-MIB")
	preferredMod := NewModule("PREFERRED-MIB")
	m.addModule(legacyMod)
	m.addModule(preferredMod)

	sharedNode := m.root.getOrCreateChild(1).getOrCreateChild(3).getOrCreateChild(6)
	sharedNode.setName("preferredObject")
	sharedNode.setModule(preferredMod)

	legacyObject := NewObject("legacyObject")
	legacyObject.setNode(sharedNode)
	legacyObject.setModule(legacyMod)
	legacyObject.setDescription("legacy object metadata")
	preferredObject := NewObject("preferredObject")
	preferredObject.setNode(sharedNode)
	preferredObject.setModule(preferredMod)
	preferredObject.setDescription("preferred object metadata")
	sharedNode.setObject(preferredObject)
	m.addObject(legacyObject)
	m.addObject(preferredObject)
	legacyMod.addObject(legacyObject)
	preferredMod.addObject(preferredObject)

	legacyNotification := NewNotification("legacyNotification")
	legacyNotification.setNode(sharedNode)
	legacyNotification.setModule(legacyMod)
	legacyNotification.setDescription("legacy notification metadata")
	preferredNotification := NewNotification("preferredNotification")
	preferredNotification.setNode(sharedNode)
	preferredNotification.setModule(preferredMod)
	preferredNotification.setDescription("preferred notification metadata")
	sharedNode.setNotification(preferredNotification)
	m.addNotification(legacyNotification)
	m.addNotification(preferredNotification)
	legacyMod.addNotification(legacyNotification)
	preferredMod.addNotification(preferredNotification)

	legacyGroup := NewGroup("legacyGroup")
	legacyGroup.setNode(sharedNode)
	legacyGroup.setModule(legacyMod)
	legacyGroup.setDescription("legacy group metadata")
	preferredGroup := NewGroup("preferredGroup")
	preferredGroup.setNode(sharedNode)
	preferredGroup.setModule(preferredMod)
	preferredGroup.setDescription("preferred group metadata")
	sharedNode.setGroup(preferredGroup)
	m.addGroup(legacyGroup)
	m.addGroup(preferredGroup)
	legacyMod.addGroup(legacyGroup)
	preferredMod.addGroup(preferredGroup)

	legacyCompliance := NewCompliance("legacyCompliance")
	legacyCompliance.setNode(sharedNode)
	legacyCompliance.setModule(legacyMod)
	legacyCompliance.setDescription("legacy compliance metadata")
	preferredCompliance := NewCompliance("preferredCompliance")
	preferredCompliance.setNode(sharedNode)
	preferredCompliance.setModule(preferredMod)
	preferredCompliance.setDescription("preferred compliance metadata")
	sharedNode.setCompliance(preferredCompliance)
	m.addCompliance(legacyCompliance)
	m.addCompliance(preferredCompliance)
	legacyMod.addCompliance(legacyCompliance)
	preferredMod.addCompliance(preferredCompliance)

	legacyCapability := NewCapability("legacyCapability")
	legacyCapability.setNode(sharedNode)
	legacyCapability.setModule(legacyMod)
	legacyCapability.setDescription("legacy capability metadata")
	preferredCapability := NewCapability("preferredCapability")
	preferredCapability.setNode(sharedNode)
	preferredCapability.setModule(preferredMod)
	preferredCapability.setDescription("preferred capability metadata")
	sharedNode.setCapability(preferredCapability)
	m.addCapability(legacyCapability)
	m.addCapability(preferredCapability)
	legacyMod.addCapability(legacyCapability)
	preferredMod.addCapability(preferredCapability)
	legacyMod.addNodeNamed("legacyObject", sharedNode)
	preferredMod.addNodeNamed("preferredObject", sharedNode)

	t.Run("global exact-name lookup", func(t *testing.T) {
		testutil.Equal(t, legacyObject, m.Object("legacyObject"), "legacy object")
		testutil.Equal(t, preferredObject, m.Object("preferredObject"), "preferred object")
		testutil.Equal(t, legacyNotification, m.Notification("legacyNotification"), "legacy notification")
		testutil.Equal(t, preferredNotification, m.Notification("preferredNotification"), "preferred notification")
		testutil.Equal(t, legacyGroup, m.Group("legacyGroup"), "legacy group")
		testutil.Equal(t, preferredGroup, m.Group("preferredGroup"), "preferred group")
		testutil.Equal(t, legacyCompliance, m.Compliance("legacyCompliance"), "legacy compliance")
		testutil.Equal(t, preferredCompliance, m.Compliance("preferredCompliance"), "preferred compliance")
		testutil.Equal(t, legacyCapability, m.Capability("legacyCapability"), "legacy capability")
		testutil.Equal(t, preferredCapability, m.Capability("preferredCapability"), "preferred capability")
	})

	t.Run("global symbol lookup", func(t *testing.T) {
		testutil.Equal(t, legacyObject, m.Symbol("legacyObject").Object(), "legacy object symbol")
		testutil.Equal(t, legacyNotification, m.Symbol("legacyNotification").Notification(), "legacy notification symbol")
		testutil.Equal(t, legacyGroup, m.Symbol("legacyGroup").Group(), "legacy group symbol")
		testutil.Equal(t, legacyCompliance, m.Symbol("legacyCompliance").Compliance(), "legacy compliance symbol")
		testutil.Equal(t, legacyCapability, m.Symbol("legacyCapability").Capability(), "legacy capability symbol")
	})

	t.Run("module-scoped lookup", func(t *testing.T) {
		testutil.Equal(t, legacyObject, legacyMod.Object("legacyObject"), "legacy module object")
		testutil.Equal(t, preferredObject, preferredMod.Object("preferredObject"), "preferred module object")
		testutil.Equal(t, legacyNotification, legacyMod.Notification("legacyNotification"), "legacy module notification")
		testutil.Equal(t, legacyGroup, legacyMod.Group("legacyGroup"), "legacy module group")
		testutil.Equal(t, legacyCompliance, legacyMod.Compliance("legacyCompliance"), "legacy module compliance")
		testutil.Equal(t, legacyCapability, legacyMod.Capability("legacyCapability"), "legacy module capability")
		testutil.Equal(t, sharedNode, legacyMod.Node("legacyObject"), "legacy qualified node")
		testutil.Equal(t, sharedNode, preferredMod.Node("preferredObject"), "preferred qualified node")
		testutil.Nil(t, legacyMod.Node("preferredObject"), "legacy cross-module node")
		testutil.True(t, legacyMod.Symbol("preferredObject").IsZero(), "legacy cross-module symbol")
		testutil.Nil(t, preferredMod.Node("legacyObject"), "preferred cross-module node")
		testutil.True(t, preferredMod.Symbol("legacyObject").IsZero(), "preferred cross-module symbol")
	})

	t.Run("shared node keeps preferred metadata", func(t *testing.T) {
		testutil.Equal(t, sharedNode, legacyObject.Node(), "legacy object node")
		testutil.Equal(t, sharedNode, preferredObject.Node(), "preferred object node")
		testutil.Equal(t, preferredObject, sharedNode.Object(), "node object")
		testutil.Equal(t, preferredNotification, sharedNode.Notification(), "node notification")
		testutil.Equal(t, preferredGroup, sharedNode.Group(), "node group")
		testutil.Equal(t, preferredCompliance, sharedNode.Compliance(), "node compliance")
		testutil.Equal(t, preferredCapability, sharedNode.Capability(), "node capability")
	})
}

func TestGlobalEntitySameNameUsesNodePreference(t *testing.T) {
	for _, tc := range []struct {
		name     string
		register func(*Mib, *Node, *Object, *Object)
	}{
		{
			name: "preferred registered last",
			register: func(m *Mib, node *Node, legacy, preferred *Object) {
				m.addObject(legacy)
				node.setObject(preferred)
				m.addObject(preferred)
			},
		},
		{
			name: "preferred registered first",
			register: func(m *Mib, node *Node, legacy, preferred *Object) {
				node.setObject(preferred)
				m.addObject(preferred)
				m.addObject(legacy)
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := NewMib()
			sharedNode := &Node{}
			legacy := NewObject("duplicateName")
			legacy.setNode(sharedNode)
			preferred := NewObject("duplicateName")
			preferred.setNode(sharedNode)

			tc.register(m, sharedNode, legacy, preferred)

			testutil.Equal(t, preferred, m.Object("duplicateName"), "preferred same-name object")
			testutil.Equal(t, preferred, m.Symbol("duplicateName").Object(), "preferred same-name symbol")
		})
	}
}

func TestNodeLookupPriority(t *testing.T) {
	m := NewMib()

	bare := &Node{name: "sysDescr"}
	withNotif := &Node{name: "sysDescr", notif: &Notification{entity: entity{name: "sysDescr"}}}
	withObj := &Node{name: "sysDescr", obj: &Object{entity: entity{name: "sysDescr"}}}

	// Register in worst-to-best order so priority can't rely on position.
	m.registerNode("sysDescr", bare)
	m.registerNode("sysDescr", withNotif)
	m.registerNode("sysDescr", withObj)

	t.Run("prefers object over notification and bare", func(t *testing.T) {
		got := m.Node("sysDescr")
		testutil.Equal(t, withObj, got, "got")
	})

	t.Run("falls back to notification when no object", func(t *testing.T) {
		m2 := NewMib()
		m2.registerNode("trap", bare)
		m2.registerNode("trap", withNotif)

		got := m2.Node("trap")
		testutil.Equal(t, withNotif, got, "got")
	})

	t.Run("falls back to bare node", func(t *testing.T) {
		m2 := NewMib()
		m2.registerNode("other", bare)

		got := m2.Node("other")
		testutil.Equal(t, bare, got, "got")
	})

	t.Run("returns nil for unknown name", func(t *testing.T) {
		testutil.Nil(t, m.Node("nonexistent"), "expected nil")
	})
}

// buildResolveTestMib creates a small Mib with IF-MIB containing
// ifDescr (1.3.6.1.2.1.2.2.1.2) and ifIndex (1.3.6.1.2.1.2.2.1.1).
func buildResolveTestMib() *Mib {
	m := NewMib()

	mod := NewModule("IF-MIB")
	m.addModule(mod)

	addTestNode := func(arcs []uint32, name string) {
		nd := m.root
		for _, arc := range arcs {
			nd = nd.getOrCreateChild(arc)
		}
		nd.setName(name)
		nd.setModule(mod)
		nd.setObject(&Object{entity: entity{name: name, module: mod}})
		m.registerNode(name, nd)
		mod.addNode(nd)
	}

	addTestNode([]uint32{1, 3, 6, 1, 2, 1, 2, 2, 1, 2}, "ifDescr")
	addTestNode([]uint32{1, 3, 6, 1, 2, 1, 2, 2, 1, 1}, "ifIndex")

	return m
}

func TestResolve(t *testing.T) {
	m := buildResolveTestMib()

	tests := []struct {
		name  string
		query string
		want  string // expected node name, "" for nil
	}{
		{"plain name", "ifDescr", "ifDescr"},
		{"plain name other", "ifIndex", "ifIndex"},
		{"qualified name", "IF-MIB::ifDescr", "ifDescr"},
		{"numeric OID", "1.3.6.1.2.1.2.2.1.2", "ifDescr"},
		{"numeric OID leading dot", ".1.3.6.1.2.1.2.2.1.2", "ifDescr"},
		{"unknown name", "nonexistent", ""},
		{"unknown module", "FAKE-MIB::ifDescr", ""},
		{"unknown name in module", "IF-MIB::nonexistent", ""},
		{"no node at OID", "999.999", ""},
		{"empty string", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := m.Resolve(tt.query)
			if tt.want == "" {
				testutil.Nil(t, got, "Resolve(%q)", tt.query)
			} else {
				testutil.NotNil(t, got, "Resolve(%q)", tt.query)
				testutil.Equal(t, tt.want, got.Name(), "Resolve(%q) name", tt.query)
			}
		})
	}
}

func TestResolveOID(t *testing.T) {
	m := buildResolveTestMib()

	tests := []struct {
		name    string
		query   string
		want    string // expected OID string
		wantErr bool
	}{
		// Name lookups
		{"plain name", "ifDescr", "1.3.6.1.2.1.2.2.1.2", false},
		{"qualified name", "IF-MIB::ifDescr", "1.3.6.1.2.1.2.2.1.2", false},

		// Instance suffixes
		{"plain name with suffix", "ifDescr.5", "1.3.6.1.2.1.2.2.1.2.5", false},
		{"qualified with suffix", "IF-MIB::ifDescr.5", "1.3.6.1.2.1.2.2.1.2.5", false},
		{"multi-arc suffix", "IF-MIB::ifDescr.5.3", "1.3.6.1.2.1.2.2.1.2.5.3", false},

		// Numeric OIDs pass through
		{"numeric OID", "1.3.6.1.2.1.2.2.1.2", "1.3.6.1.2.1.2.2.1.2", false},
		{"numeric OID leading dot", ".1.3.6.1.2.1.2.2.1.2", "1.3.6.1.2.1.2.2.1.2", false},
		{"numeric OID unknown", "999.999.999", "999.999.999", false},

		// Errors
		{"empty", "", "", true},
		{"unknown name", "nonexistent", "", true},
		{"unknown module", "FAKE-MIB::ifDescr", "", true},
		{"unknown name in module", "IF-MIB::nonexistent", "", true},
		{"bad suffix", "ifDescr.abc", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := m.ResolveOID(tt.query)
			if tt.wantErr {
				testutil.Error(t, err, "ResolveOID(%q)", tt.query)
				return
			}
			testutil.NoError(t, err, "ResolveOID(%q)", tt.query)
			testutil.Equal(t, tt.want, got.String(), "ResolveOID(%q)", tt.query)
		})
	}
}

func TestResolveOIDFormatRoundTrip(t *testing.T) {
	m := buildResolveTestMib()

	oid := OID{1, 3, 6, 1, 2, 1, 2, 2, 1, 2, 5}
	formatted := m.FormatOID(oid)
	testutil.Equal(t, "IF-MIB::ifDescr.5", formatted, "FormatOID")

	back, err := m.ResolveOID(formatted)
	testutil.NoError(t, err, "ResolveOID")
	testutil.True(t, oid.Equal(back), "round-trip: got %s, want %s", back, oid)
}

func TestLookupInstance(t *testing.T) {
	m := buildResolveTestMib()

	t.Run("with suffix", func(t *testing.T) {
		lookup := m.LookupInstance(OID{1, 3, 6, 1, 2, 1, 2, 2, 1, 1, 5})
		testutil.NotNil(t, lookup.Node(), "node")
		testutil.Equal(t, "ifIndex", lookup.Node().Name(), "node name")
		testutil.Equal(t, "5", lookup.Suffix().String(), "suffix")
	})

	t.Run("exact match", func(t *testing.T) {
		lookup := m.LookupInstance(OID{1, 3, 6, 1, 2, 1, 2, 2, 1, 2})
		testutil.NotNil(t, lookup.Node(), "node")
		testutil.Equal(t, "ifDescr", lookup.Node().Name(), "node name")
		testutil.Len(t, lookup.Suffix(), 0, "suffix should be empty")
	})

	t.Run("multi-arc suffix", func(t *testing.T) {
		lookup := m.LookupInstance(OID{1, 3, 6, 1, 2, 1, 2, 2, 1, 2, 5, 3})
		testutil.NotNil(t, lookup.Node(), "node")
		testutil.Equal(t, "ifDescr", lookup.Node().Name(), "node name")
		testutil.Equal(t, "5.3", lookup.Suffix().String(), "suffix")
	})

	t.Run("no match", func(t *testing.T) {
		lookup := m.LookupInstance(OID{9, 9, 9})
		testutil.NotNil(t, lookup.Node(), "root node returned")
		testutil.Equal(t, "", lookup.Node().Name(), "unnamed root")
	})

	t.Run("empty OID", func(t *testing.T) {
		lookup := m.LookupInstance(OID{})
		testutil.NotNil(t, lookup.Node(), "root node returned")
		testutil.Len(t, lookup.Suffix(), 0, "suffix should be empty")
	})
}

func TestNodesByName(t *testing.T) {
	m := NewMib()

	bare := &Node{name: "ifDescr"}
	withObj := &Node{name: "ifDescr", obj: &Object{entity: entity{name: "ifDescr"}}}
	m.registerNode("ifDescr", bare)
	m.registerNode("ifDescr", withObj)

	t.Run("returns all nodes", func(t *testing.T) {
		got := m.NodesByName("ifDescr")
		testutil.Len(t, got, 2, "NodesByName count")
	})

	t.Run("returns nil for unknown", func(t *testing.T) {
		testutil.Nil(t, m.NodesByName("nonexistent"), "expected nil")
	})

	t.Run("returns clone", func(t *testing.T) {
		got := m.NodesByName("ifDescr")
		got[0] = nil
		testutil.NotNil(t, m.NodesByName("ifDescr")[0], "original should be unmodified")
	})
}

func TestAllSymbols(t *testing.T) {
	m := NewMib()

	mod1 := NewModule("MOD-A")
	mod1.addObject(&Object{entity: entity{name: "objA"}})
	mod1.addType(&Type{name: "TypeA"})
	m.addModule(mod1)

	mod2 := NewModule("MOD-B")
	mod2.addObject(&Object{entity: entity{name: "objB"}})
	m.addModule(mod2)

	var names []string
	for sym := range m.AllSymbols() {
		names = append(names, sym.Name())
	}

	testutil.Len(t, names, 3, "AllSymbols count")
	testutil.Equal(t, "objA", names[0], "first symbol")
	testutil.Equal(t, "TypeA", names[1], "second symbol")
	testutil.Equal(t, "objB", names[2], "third symbol")
}

func TestAllSymbols_Empty(t *testing.T) {
	m := NewMib()

	count := 0
	for range m.AllSymbols() {
		count++
	}
	testutil.Equal(t, 0, count, "AllSymbols on empty Mib")
}

func TestAddTypeFirstWriteWins(t *testing.T) {
	m := NewMib()

	first := &Type{name: "DisplayString"}
	second := &Type{name: "DisplayString"}

	m.addType(first)
	m.addType(second)

	testutil.Equal(t, first, m.Type("DisplayString"), "got")

	// Both are still in the full list.
	testutil.Len(t, m.types, 2, "types")
}
