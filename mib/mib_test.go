package mib

import (
	"testing"

	"github.com/golangsnmp/gomib/internal/testutil"
)

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
