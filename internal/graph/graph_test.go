package graph

import (
	"testing"
)

func TestGraphBasic(t *testing.T) {
	g := New()

	a := Symbol{Module: "M", Name: "a"}
	b := Symbol{Module: "M", Name: "b"}

	g.AddNode(a, NodeKindType)
	g.AddNode(b, NodeKindType)
	g.AddEdge(a, b) // a depends on b

	if len(g.Nodes()) != 2 {
		t.Errorf("node count = %d, want 2", len(g.Nodes()))
	}
	if len(g.Dependencies(a)) != 1 {
		t.Errorf("a dependencies = %d, want 1", len(g.Dependencies(a)))
	}
	if g.Dependencies(a)[0] != b {
		t.Errorf("a depends on %v, want %v", g.Dependencies(a)[0], b)
	}
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
	if len(cycles) != 0 {
		t.Errorf("cycles = %d, want 0", len(cycles))
	}
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
	if len(cycles) != 1 {
		t.Fatalf("cycles = %d, want 1", len(cycles))
	}
	if len(cycles[0]) != 2 {
		t.Errorf("cycle length = %d, want 2", len(cycles[0]))
	}
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
	if len(cycles) != 1 {
		t.Fatalf("cycles = %d, want 1", len(cycles))
	}
	if len(cycles[0]) != 3 {
		t.Errorf("cycle length = %d, want 3", len(cycles[0]))
	}
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
	if len(cyclic) != 0 {
		t.Errorf("cyclic = %d, want 0", len(cyclic))
	}
	if len(order) != 3 {
		t.Fatalf("order = %d, want 3", len(order))
	}

	// c should come before b, b should come before a
	indexOf := func(s Symbol) int {
		for i, sym := range order {
			if sym == s {
				return i
			}
		}
		return -1
	}

	if indexOf(c) >= indexOf(b) {
		t.Error("c should come before b")
	}
	if indexOf(b) >= indexOf(a) {
		t.Error("b should come before a")
	}
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
	if len(cyclic) == 0 {
		t.Error("should have cyclic nodes")
	}
}

func TestMarkResolved(t *testing.T) {
	g := New()

	a := Symbol{Module: "M", Name: "a"}
	g.AddNode(a, NodeKindType)

	if g.IsResolved(a) {
		t.Error("initially should not be resolved")
	}

	if !g.MarkResolved(a) {
		t.Error("MarkResolved should return true for existing node")
	}
	if !g.IsResolved(a) {
		t.Error("should be resolved after MarkResolved")
	}

	missing := Symbol{Module: "M", Name: "missing"}
	if g.MarkResolved(missing) {
		t.Error("MarkResolved should return false for missing node")
	}
}

func TestHasNode(t *testing.T) {
	g := New()
	a := Symbol{Module: "M", Name: "a"}
	if g.HasNode(a) {
		t.Error("empty graph should not have node")
	}
	g.AddNode(a, NodeKindType)
	if !g.HasNode(a) {
		t.Error("should have node after AddNode")
	}
}

func TestSelfLoop(t *testing.T) {
	g := New()

	a := Symbol{Module: "M", Name: "a"}
	b := Symbol{Module: "M", Name: "b"}

	g.AddNode(a, NodeKindType)
	g.AddNode(b, NodeKindType)
	g.AddEdge(a, a) // self-loop
	g.AddEdge(b, a)

	// FindCycles should detect the self-loop.
	cycles := g.FindCycles()
	if len(cycles) != 1 {
		t.Fatalf("cycles = %d, want 1", len(cycles))
	}
	if len(cycles[0]) != 1 {
		t.Errorf("self-loop cycle length = %d, want 1", len(cycles[0]))
	}
	if cycles[0][0] != a {
		t.Errorf("self-loop node = %v, want %v", cycles[0][0], a)
	}

	// TopologicalOrder should also report it as cyclic.
	_, cyclic := g.TopologicalOrder()
	if len(cyclic) == 0 {
		t.Error("self-loop should be cyclic in topo sort")
	}
}

func TestDuplicateEdges(t *testing.T) {
	g := New()

	a := Symbol{Module: "M", Name: "a"}
	b := Symbol{Module: "M", Name: "b"}

	g.AddNode(a, NodeKindType)
	g.AddNode(b, NodeKindType)
	g.AddEdge(a, b)
	g.AddEdge(a, b) // duplicate
	g.AddEdge(a, b) // duplicate

	// Should only have one edge.
	if len(g.Dependencies(a)) != 1 {
		t.Errorf("dependencies = %d, want 1 (duplicate edges deduplicated)", len(g.Dependencies(a)))
	}

	// Topo sort should work correctly.
	order, cyclic := g.TopologicalOrder()
	if len(cyclic) != 0 {
		t.Errorf("cyclic = %d, want 0", len(cyclic))
	}
	if len(order) != 2 {
		t.Errorf("order = %d, want 2", len(order))
	}
}

func TestImplicitNodeKindUpdate(t *testing.T) {
	g := New()

	a := Symbol{Module: "M", Name: "a"}
	b := Symbol{Module: "M", Name: "b"}

	// b is implicitly created by AddEdge with zero-value kind.
	g.AddEdge(a, b)
	if g.Node(b).Kind != NodeKind(0) {
		t.Errorf("implicit node kind = %v, want 0", g.Node(b).Kind)
	}

	// Explicit AddNode should update the kind.
	g.AddNode(b, NodeKindOID)
	if g.Node(b).Kind != NodeKindOID {
		t.Errorf("kind = %v, want NodeKindOID", g.Node(b).Kind)
	}
}
