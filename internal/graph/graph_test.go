package graph

import (
	"testing"

	"github.com/golangsnmp/gomib/internal/testutil"
)

func TestGraphBasic(t *testing.T) {
	g := New(0)

	a := Symbol{Module: "M", Name: "a"}
	b := Symbol{Module: "M", Name: "b"}

	g.AddNode(a)
	g.AddNode(b)
	g.AddEdge(a, b)

	testutil.True(t, g.HasNode(a), "graph should have node a")
	testutil.True(t, g.HasNode(b), "graph should have node b")
	testutil.Len(t, g.Dependencies(a), 1, "a dependencies")
	testutil.Equal(t, b, g.Dependencies(a)[0], "a depends on")
}

func TestAddEdgeCreatesNodes(t *testing.T) {
	g := New(0)

	a := Symbol{Module: "M", Name: "a"}
	b := Symbol{Module: "M", Name: "b"}

	// No AddNode calls, only AddEdge.
	g.AddEdge(a, b)

	testutil.True(t, g.HasNode(a), "AddEdge should create 'from' node")
	testutil.True(t, g.HasNode(b), "AddEdge should create 'to' node")
	testutil.Len(t, g.Dependencies(a), 1, "a dependencies")
}

func TestHasNode(t *testing.T) {
	g := New(0)
	a := Symbol{Module: "M", Name: "a"}
	testutil.False(t, g.HasNode(a), "empty graph should not have node")
	g.AddNode(a)
	testutil.True(t, g.HasNode(a), "should have node after AddNode")
}

func TestDuplicateEdges(t *testing.T) {
	g := New(0)

	a := Symbol{Module: "M", Name: "a"}
	b := Symbol{Module: "M", Name: "b"}

	g.AddEdge(a, b)
	g.AddEdge(a, b)
	g.AddEdge(a, b)

	testutil.Len(t, g.Dependencies(a), 1, "duplicate edges deduplicated")

	order, cycles := g.ResolutionOrder()
	testutil.Len(t, cycles, 0, "cycles")
	testutil.Len(t, order, 2, "order")
}

func TestResolutionOrderEmpty(t *testing.T) {
	g := New(0)
	order, cycles := g.ResolutionOrder()
	testutil.Len(t, order, 0, "order")
	testutil.Len(t, cycles, 0, "cycles")
}

func TestResolutionOrderIsolatedNode(t *testing.T) {
	g := New(0)
	a := Symbol{Module: "M", Name: "a"}
	g.AddNode(a)

	order, cycles := g.ResolutionOrder()
	testutil.Len(t, cycles, 0, "cycles")
	testutil.Len(t, order, 1, "order")
	testutil.Equal(t, a, order[0], "order[0]")
}

func TestResolutionOrderChain(t *testing.T) {
	g := New(0)

	a := Symbol{Module: "M", Name: "a"}
	b := Symbol{Module: "M", Name: "b"}
	c := Symbol{Module: "M", Name: "c"}

	g.AddEdge(a, b)
	g.AddEdge(b, c)

	order, cycles := g.ResolutionOrder()
	testutil.Len(t, cycles, 0, "cycles")

	// Deterministic: c, b, a.
	want := []Symbol{c, b, a}
	testutil.SliceEqual(t, want, order, "order")
}

func TestResolutionOrderDiamond(t *testing.T) {
	g := New(0)

	a := Symbol{Module: "M", Name: "a"}
	b := Symbol{Module: "M", Name: "b"}
	c := Symbol{Module: "M", Name: "c"}
	d := Symbol{Module: "M", Name: "d"}

	// a depends on b and c, both depend on d.
	g.AddEdge(a, b)
	g.AddEdge(a, c)
	g.AddEdge(b, d)
	g.AddEdge(c, d)

	order, cycles := g.ResolutionOrder()
	testutil.Len(t, cycles, 0, "cycles")
	testutil.Len(t, order, 4, "order")

	indexOf := func(s Symbol) int {
		for i, sym := range order {
			if sym == s {
				return i
			}
		}
		return -1
	}

	// d must come before b and c, both before a.
	testutil.True(t, indexOf(d) < indexOf(b), "d should come before b")
	testutil.True(t, indexOf(d) < indexOf(c), "d should come before c")
	testutil.True(t, indexOf(b) < indexOf(a), "b should come before a")
	testutil.True(t, indexOf(c) < indexOf(a), "c should come before a")
}

func TestResolutionOrderSimpleCycle(t *testing.T) {
	g := New(0)

	a := Symbol{Module: "M", Name: "a"}
	b := Symbol{Module: "M", Name: "b"}

	g.AddEdge(a, b)
	g.AddEdge(b, a)

	order, cycles := g.ResolutionOrder()
	testutil.Len(t, order, 0, "all nodes in cycle")
	testutil.Len(t, cycles, 1, "cycles")
	testutil.Len(t, cycles[0], 2, "cycle length")
}

func TestResolutionOrderTriangleCycle(t *testing.T) {
	g := New(0)

	a := Symbol{Module: "M", Name: "a"}
	b := Symbol{Module: "M", Name: "b"}
	c := Symbol{Module: "M", Name: "c"}

	g.AddEdge(a, b)
	g.AddEdge(b, c)
	g.AddEdge(c, a)

	_, cycles := g.ResolutionOrder()
	testutil.Len(t, cycles, 1, "cycles")
	testutil.Len(t, cycles[0], 3, "cycle length")
}

func TestResolutionOrderCycleDependents(t *testing.T) {
	g := New(0)

	a := Symbol{Module: "M", Name: "a"}
	b := Symbol{Module: "M", Name: "b"}
	c := Symbol{Module: "M", Name: "c"}

	g.AddEdge(a, b)
	g.AddEdge(b, a)
	g.AddEdge(c, a)

	order, cycles := g.ResolutionOrder()
	testutil.Len(t, cycles, 1, "cycles")
	testutil.Len(t, cycles[0], 2, "cycle length")

	// c should still appear in the order despite depending on a cycle member.
	testutil.Len(t, order, 1, "order")
	testutil.Equal(t, c, order[0], "order[0]")
}

func TestSelfLoop(t *testing.T) {
	g := New(0)

	a := Symbol{Module: "M", Name: "a"}
	b := Symbol{Module: "M", Name: "b"}

	g.AddEdge(a, a)
	g.AddEdge(b, a)

	order, cycles := g.ResolutionOrder()
	testutil.Len(t, cycles, 1, "cycles")
	testutil.Len(t, cycles[0], 1, "self-loop cycle length")
	testutil.Equal(t, a, cycles[0][0], "self-loop node")

	// b should still appear in the order.
	testutil.Len(t, order, 1, "order")
	testutil.Equal(t, b, order[0], "order[0]")
}

func TestResolutionOrderMultipleSCCs(t *testing.T) {
	// Adapted from the Wikipedia Tarjan's example.
	// Three cycles ({a,b,c}, {d,e}, {f,g}), one self-loop (h).
	g := New(0)

	sym := func(name string) Symbol { return Symbol{Module: "M", Name: name} }
	a, b, c := sym("a"), sym("b"), sym("c")
	d, e := sym("d"), sym("e")
	f, gg := sym("f"), sym("g")
	h := sym("h")

	g.AddEdge(a, b)
	g.AddEdge(b, c)
	g.AddEdge(c, a) // cycle: a, b, c
	g.AddEdge(d, b)
	g.AddEdge(d, c)
	g.AddEdge(d, e)
	g.AddEdge(e, d) // cycle: d, e (but d also depends on the a-b-c cycle)
	g.AddEdge(f, c)
	g.AddEdge(f, gg)
	g.AddEdge(gg, f) // cycle: f, g
	g.AddEdge(h, e)
	g.AddEdge(h, gg)
	g.AddEdge(h, h) // self-loop

	order, cycles := g.ResolutionOrder()

	// Three cycles: {a,b,c}, {d,e}, {f,g}. Plus self-loop {h}.
	if len(cycles) != 4 {
		t.Errorf("cycles = %d, want 4", len(cycles))
		for i, cyc := range cycles {
			t.Logf("  cycle %d: %v", i, cyc)
		}
	}

	// No acyclic nodes in this graph, so order should be empty.
	if len(order) != 0 {
		t.Errorf("order = %d, want 0 (all nodes are in cycles)", len(order))
		t.Logf("  order: %v", order)
	}
}

func TestResolutionOrderCrossModule(t *testing.T) {
	g := New(0)

	// Symbols across two modules. Tests that Module-then-Name sort
	// produces deterministic output.
	a1 := Symbol{Module: "A", Name: "x"}
	a2 := Symbol{Module: "A", Name: "y"}
	b1 := Symbol{Module: "B", Name: "x"}

	g.AddEdge(a2, a1)
	g.AddEdge(b1, a1)

	order, cycles := g.ResolutionOrder()
	testutil.Len(t, cycles, 0, "cycles")

	// Deterministic: A:x first (leaf), then A:y and B:x.
	// A:y before B:x because "A" < "B" in module sort.
	want := []Symbol{a1, a2, b1}
	testutil.SliceEqual(t, want, order, "order")
}

func TestResolutionOrderDisconnected(t *testing.T) {
	g := New(0)

	a := Symbol{Module: "M", Name: "a"}
	b := Symbol{Module: "M", Name: "b"}
	c := Symbol{Module: "M", Name: "c"}

	g.AddNode(a)
	g.AddNode(b)
	g.AddNode(c)

	order, cycles := g.ResolutionOrder()
	testutil.Len(t, cycles, 0, "cycles")

	// Deterministic: sorted by name since module is the same.
	want := []Symbol{a, b, c}
	testutil.SliceEqual(t, want, order, "order")
}
