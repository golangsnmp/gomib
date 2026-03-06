package mib

import (
	"testing"

	"github.com/golangsnmp/gomib/internal/testutil"
)

func TestNodeLookupPriority(t *testing.T) {
	m := newMib()

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
		m2 := newMib()
		m2.registerNode("trap", bare)
		m2.registerNode("trap", withNotif)

		got := m2.Node("trap")
		testutil.Equal(t, withNotif, got, "got")
	})

	t.Run("falls back to bare node", func(t *testing.T) {
		m2 := newMib()
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
	m := newMib()

	mod := newModule("IF-MIB")
	m.addModule(mod)

	addTestNode := func(arcs []uint32, name string) {
		nd := m.root
		for _, arc := range arcs {
			nd = nd.getOrCreateChild(arc)
		}
		nd.setName(name)
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

func TestAllSymbols(t *testing.T) {
	m := newMib()

	mod1 := newModule("MOD-A")
	mod1.addObject(&Object{entity: entity{name: "objA"}})
	mod1.addType(&Type{name: "TypeA"})
	m.addModule(mod1)

	mod2 := newModule("MOD-B")
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
	m := newMib()

	count := 0
	for range m.AllSymbols() {
		count++
	}
	testutil.Equal(t, 0, count, "AllSymbols on empty Mib")
}

func TestAddTypeFirstWriteWins(t *testing.T) {
	m := newMib()

	first := &Type{name: "DisplayString"}
	second := &Type{name: "DisplayString"}

	m.addType(first)
	m.addType(second)

	testutil.Equal(t, first, m.Type("DisplayString"), "got")

	// Both are still in the full list.
	testutil.Len(t, m.types, 2, "types")
}
