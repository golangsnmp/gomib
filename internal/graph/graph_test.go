package graph

import (
	"slices"
	"testing"
)

func TestGraphBasic(t *testing.T) {
	g := New(0)

	a := Symbol{Module: "M", Name: "a"}
	b := Symbol{Module: "M", Name: "b"}

	g.AddNode(a)
	g.AddNode(b)
	g.AddEdge(a, b)

	if !g.HasNode(a) {
		t.Error("graph should have node a")
	}
	if !g.HasNode(b) {
		t.Error("graph should have node b")
	}
	if len(g.Dependencies(a)) != 1 {
		t.Errorf("a dependencies = %d, want 1", len(g.Dependencies(a)))
	}
	if g.Dependencies(a)[0] != b {
		t.Errorf("a depends on %v, want %v", g.Dependencies(a)[0], b)
	}
}

func TestAddEdgeCreatesNodes(t *testing.T) {
	g := New(0)

	a := Symbol{Module: "M", Name: "a"}
	b := Symbol{Module: "M", Name: "b"}

	// No AddNode calls, only AddEdge.
	g.AddEdge(a, b)

	if !g.HasNode(a) {
		t.Error("AddEdge should create 'from' node")
	}
	if !g.HasNode(b) {
		t.Error("AddEdge should create 'to' node")
	}
	if len(g.Dependencies(a)) != 1 {
		t.Errorf("a dependencies = %d, want 1", len(g.Dependencies(a)))
	}
}

func TestHasNode(t *testing.T) {
	g := New(0)
	a := Symbol{Module: "M", Name: "a"}
	if g.HasNode(a) {
		t.Error("empty graph should not have node")
	}
	g.AddNode(a)
	if !g.HasNode(a) {
		t.Error("should have node after AddNode")
	}
}

func TestDuplicateEdges(t *testing.T) {
	g := New(0)

	a := Symbol{Module: "M", Name: "a"}
	b := Symbol{Module: "M", Name: "b"}

	g.AddEdge(a, b)
	g.AddEdge(a, b)
	g.AddEdge(a, b)

	if len(g.Dependencies(a)) != 1 {
		t.Errorf("dependencies = %d, want 1 (duplicate edges deduplicated)", len(g.Dependencies(a)))
	}

	order, cycles := g.ResolutionOrder()
	if len(cycles) != 0 {
		t.Errorf("cycles = %d, want 0", len(cycles))
	}
	if len(order) != 2 {
		t.Errorf("order = %d, want 2", len(order))
	}
}

func TestResolutionOrderEmpty(t *testing.T) {
	g := New(0)
	order, cycles := g.ResolutionOrder()
	if len(order) != 0 {
		t.Errorf("order = %d, want 0", len(order))
	}
	if len(cycles) != 0 {
		t.Errorf("cycles = %d, want 0", len(cycles))
	}
}

func TestResolutionOrderIsolatedNode(t *testing.T) {
	g := New(0)
	a := Symbol{Module: "M", Name: "a"}
	g.AddNode(a)

	order, cycles := g.ResolutionOrder()
	if len(cycles) != 0 {
		t.Errorf("cycles = %d, want 0", len(cycles))
	}
	if len(order) != 1 {
		t.Fatalf("order = %d, want 1", len(order))
	}
	if order[0] != a {
		t.Errorf("order[0] = %v, want %v", order[0], a)
	}
}

func TestResolutionOrderChain(t *testing.T) {
	g := New(0)

	a := Symbol{Module: "M", Name: "a"}
	b := Symbol{Module: "M", Name: "b"}
	c := Symbol{Module: "M", Name: "c"}

	g.AddEdge(a, b)
	g.AddEdge(b, c)

	order, cycles := g.ResolutionOrder()
	if len(cycles) != 0 {
		t.Errorf("cycles = %d, want 0", len(cycles))
	}

	// Deterministic: c, b, a.
	want := []Symbol{c, b, a}
	if !slices.Equal(order, want) {
		t.Errorf("order = %v, want %v", order, want)
	}
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
	if len(cycles) != 0 {
		t.Errorf("cycles = %d, want 0", len(cycles))
	}
	if len(order) != 4 {
		t.Fatalf("order = %d, want 4", len(order))
	}

	indexOf := func(s Symbol) int {
		for i, sym := range order {
			if sym == s {
				return i
			}
		}
		return -1
	}

	// d must come before b and c, both before a.
	if indexOf(d) >= indexOf(b) {
		t.Error("d should come before b")
	}
	if indexOf(d) >= indexOf(c) {
		t.Error("d should come before c")
	}
	if indexOf(b) >= indexOf(a) {
		t.Error("b should come before a")
	}
	if indexOf(c) >= indexOf(a) {
		t.Error("c should come before a")
	}
}

func TestResolutionOrderSimpleCycle(t *testing.T) {
	g := New(0)

	a := Symbol{Module: "M", Name: "a"}
	b := Symbol{Module: "M", Name: "b"}

	g.AddEdge(a, b)
	g.AddEdge(b, a)

	order, cycles := g.ResolutionOrder()
	if len(order) != 0 {
		t.Errorf("order = %d, want 0 (all nodes in cycle)", len(order))
	}
	if len(cycles) != 1 {
		t.Fatalf("cycles = %d, want 1", len(cycles))
	}
	if len(cycles[0]) != 2 {
		t.Errorf("cycle length = %d, want 2", len(cycles[0]))
	}
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
	if len(cycles) != 1 {
		t.Fatalf("cycles = %d, want 1", len(cycles))
	}
	if len(cycles[0]) != 3 {
		t.Errorf("cycle length = %d, want 3", len(cycles[0]))
	}
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
	if len(cycles) != 1 {
		t.Fatalf("cycles = %d, want 1", len(cycles))
	}
	if len(cycles[0]) != 2 {
		t.Errorf("cycle length = %d, want 2", len(cycles[0]))
	}

	// c should still appear in the order despite depending on a cycle member.
	if len(order) != 1 {
		t.Fatalf("order = %d, want 1", len(order))
	}
	if order[0] != c {
		t.Errorf("order[0] = %v, want %v", order[0], c)
	}
}

func TestSelfLoop(t *testing.T) {
	g := New(0)

	a := Symbol{Module: "M", Name: "a"}
	b := Symbol{Module: "M", Name: "b"}

	g.AddEdge(a, a)
	g.AddEdge(b, a)

	order, cycles := g.ResolutionOrder()
	if len(cycles) != 1 {
		t.Fatalf("cycles = %d, want 1", len(cycles))
	}
	if len(cycles[0]) != 1 {
		t.Errorf("self-loop cycle length = %d, want 1", len(cycles[0]))
	}
	if cycles[0][0] != a {
		t.Errorf("self-loop node = %v, want %v", cycles[0][0], a)
	}

	// b should still appear in the order.
	if len(order) != 1 {
		t.Fatalf("order = %d, want 1", len(order))
	}
	if order[0] != b {
		t.Errorf("order[0] = %v, want %v", order[0], b)
	}
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
	if len(cycles) != 0 {
		t.Errorf("cycles = %d, want 0", len(cycles))
	}

	// Deterministic: A:x first (leaf), then A:y and B:x.
	// A:y before B:x because "A" < "B" in module sort.
	want := []Symbol{a1, a2, b1}
	if !slices.Equal(order, want) {
		t.Errorf("order = %v, want %v", order, want)
	}
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
	if len(cycles) != 0 {
		t.Errorf("cycles = %d, want 0", len(cycles))
	}

	// Deterministic: sorted by name since module is the same.
	want := []Symbol{a, b, c}
	if !slices.Equal(order, want) {
		t.Errorf("order = %v, want %v", order, want)
	}
}
