package mib

import "testing"

func TestNodeLookupPriority(t *testing.T) {
	m := newMib()

	bare := &Node{name: "sysDescr"}
	withNotif := &Node{name: "sysDescr", notif: &Notification{name: "sysDescr"}}
	withObj := &Node{name: "sysDescr", obj: &Object{name: "sysDescr"}}

	// Register in worst-to-best order so priority can't rely on position.
	m.registerNode("sysDescr", bare)
	m.registerNode("sysDescr", withNotif)
	m.registerNode("sysDescr", withObj)

	t.Run("prefers object over notification and bare", func(t *testing.T) {
		got := m.Node("sysDescr")
		if got != withObj {
			t.Errorf("got %p, want node with object (%p)", got, withObj)
		}
	})

	t.Run("falls back to notification when no object", func(t *testing.T) {
		m2 := newMib()
		m2.registerNode("trap", bare)
		m2.registerNode("trap", withNotif)

		got := m2.Node("trap")
		if got != withNotif {
			t.Errorf("got %p, want node with notification (%p)", got, withNotif)
		}
	})

	t.Run("falls back to bare node", func(t *testing.T) {
		m2 := newMib()
		m2.registerNode("other", bare)

		got := m2.Node("other")
		if got != bare {
			t.Errorf("got %p, want bare node (%p)", got, bare)
		}
	})

	t.Run("returns nil for unknown name", func(t *testing.T) {
		if got := m.Node("nonexistent"); got != nil {
			t.Errorf("got %p, want nil", got)
		}
	})
}

func TestAddTypeFirstWriteWins(t *testing.T) {
	m := newMib()

	first := &Type{name: "DisplayString"}
	second := &Type{name: "DisplayString"}

	m.addType(first)
	m.addType(second)

	if got := m.Type("DisplayString"); got != first {
		t.Errorf("got %p, want first registered type (%p)", got, first)
	}

	// Both are still in the full list.
	if len(m.types) != 2 {
		t.Errorf("got %d types, want 2", len(m.types))
	}
}
