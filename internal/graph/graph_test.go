package graph

import (
	"testing"

	"github.com/golangsnmp/gomib/internal/testutil"
)

func TestGraphBasic(t *testing.T) {
	g := New()

	a := Symbol{Module: "M", Name: "a"}
	b := Symbol{Module: "M", Name: "b"}

	g.AddNode(a, NodeKindType)
	g.AddNode(b, NodeKindType)
	g.AddEdge(a, b) // a depends on b

	testutil.Equal(t, 2, len(g.Nodes()), "node count")
	testutil.Len(t, g.Dependencies(a), 1, "a dependencies")
	testutil.Len(t, g.Dependents(b), 1, "b dependents")
	testutil.Equal(t, b, g.Dependencies(a)[0], "a depends on b")
	testutil.Equal(t, a, g.Dependents(b)[0], "b is depended on by a")
}

func TestFindCyclesNoCycle(t *testing.T) {
	g := New()

	a := Symbol{Module: "M", Name: "a"}
	b := Symbol{Module: "M", Name: "b"}
	c := Symbol{Module: "M", Name: "c"}

	g.AddNode(a, NodeKindType)
	g.AddNode(b, NodeKindType)
	g.AddNode(c, NodeKindType)
	g.AddEdge(a, b) // a -> b
	g.AddEdge(b, c) // b -> c

	cycles := g.FindCycles()
	testutil.Len(t, cycles, 0, "no cycles expected")
}

func TestFindCyclesSimple(t *testing.T) {
	g := New()

	a := Symbol{Module: "M", Name: "a"}
	b := Symbol{Module: "M", Name: "b"}

	g.AddNode(a, NodeKindType)
	g.AddNode(b, NodeKindType)
	g.AddEdge(a, b) // a -> b
	g.AddEdge(b, a) // b -> a (cycle!)

	cycles := g.FindCycles()
	testutil.Len(t, cycles, 1, "one cycle expected")
	testutil.Len(t, cycles[0], 2, "cycle should have 2 nodes")
}

func TestFindCyclesTriangle(t *testing.T) {
	g := New()

	a := Symbol{Module: "M", Name: "a"}
	b := Symbol{Module: "M", Name: "b"}
	c := Symbol{Module: "M", Name: "c"}

	g.AddNode(a, NodeKindType)
	g.AddNode(b, NodeKindType)
	g.AddNode(c, NodeKindType)
	g.AddEdge(a, b) // a -> b
	g.AddEdge(b, c) // b -> c
	g.AddEdge(c, a) // c -> a (cycle!)

	cycles := g.FindCycles()
	testutil.Len(t, cycles, 1, "one cycle expected")
	testutil.Len(t, cycles[0], 3, "cycle should have 3 nodes")
}

func TestTopologicalOrderSimple(t *testing.T) {
	g := New()

	a := Symbol{Module: "M", Name: "a"}
	b := Symbol{Module: "M", Name: "b"}
	c := Symbol{Module: "M", Name: "c"}

	g.AddNode(a, NodeKindType)
	g.AddNode(b, NodeKindType)
	g.AddNode(c, NodeKindType)
	g.AddEdge(a, b) // a depends on b
	g.AddEdge(b, c) // b depends on c

	order, cyclic := g.ResolutionOrder()
	testutil.Len(t, cyclic, 0, "no cyclic nodes")
	testutil.Len(t, order, 3, "all nodes in order")

	// c should come before b, b should come before a
	indexOf := func(s Symbol) int {
		for i, sym := range order {
			if sym == s {
				return i
			}
		}
		return -1
	}

	testutil.True(t, indexOf(c) < indexOf(b), "c should come before b")
	testutil.True(t, indexOf(b) < indexOf(a), "b should come before a")
}

func TestTopologicalOrderWithCycle(t *testing.T) {
	g := New()

	a := Symbol{Module: "M", Name: "a"}
	b := Symbol{Module: "M", Name: "b"}
	c := Symbol{Module: "M", Name: "c"}

	g.AddNode(a, NodeKindType)
	g.AddNode(b, NodeKindType)
	g.AddNode(c, NodeKindType)
	g.AddEdge(a, b)
	g.AddEdge(b, a) // cycle between a and b
	g.AddEdge(c, a)

	_, cyclic := g.ResolutionOrder()
	testutil.Greater(t, len(cyclic), 0, "should have cyclic nodes")
}

func TestMarkResolved(t *testing.T) {
	g := New()

	a := Symbol{Module: "M", Name: "a"}
	g.AddNode(a, NodeKindType)

	testutil.False(t, g.IsResolved(a), "initially not resolved")

	g.MarkResolved(a)
	testutil.True(t, g.IsResolved(a), "now resolved")
}
