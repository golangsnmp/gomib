package graph

import (
	"fmt"
	"testing"
)

// FuzzResolutionOrder builds arbitrary dependency graphs from fuzz data
// and validates invariants of the topological sort and cycle detection.
func FuzzResolutionOrder(f *testing.F) {
	// Linear chain: a -> b -> c
	f.Add(uint8(3), []byte{0, 1, 1, 2})
	// Diamond: a -> b, a -> c, b -> d, c -> d
	f.Add(uint8(4), []byte{0, 1, 0, 2, 1, 3, 2, 3})
	// Simple cycle: a -> b, b -> a
	f.Add(uint8(2), []byte{0, 1, 1, 0})
	// Self-loop
	f.Add(uint8(1), []byte{0, 0})
	// Disconnected nodes
	f.Add(uint8(5), []byte{})
	// Triangle cycle with dependent
	f.Add(uint8(4), []byte{0, 1, 1, 2, 2, 0, 3, 0})
	// Empty graph
	f.Add(uint8(0), []byte{})

	f.Fuzz(func(t *testing.T, nodeCount uint8, edgeData []byte) {
		if nodeCount == 0 {
			return
		}
		// Cap node count to avoid spending too long on huge graphs.
		n := min(int(nodeCount), 64)

		g := New(n)

		// Create all nodes.
		syms := make([]Symbol, n)
		for i := range n {
			syms[i] = Symbol{Module: "M", Name: fmt.Sprintf("n%d", i)}
			g.AddNode(syms[i])
		}

		// Add edges from pairs of bytes (from, to), using modular indexing.
		edges := make(map[[2]int]bool)
		for i := 0; i+1 < len(edgeData); i += 2 {
			from := int(edgeData[i]) % n
			to := int(edgeData[i+1]) % n
			g.AddEdge(syms[from], syms[to])
			edges[[2]int{from, to}] = true
		}

		order, cycles := g.ResolutionOrder()

		// Build lookup sets.
		inOrder := make(map[Symbol]int)
		for i, sym := range order {
			inOrder[sym] = i
		}
		inCycle := make(map[Symbol]bool)
		for _, cyc := range cycles {
			for _, sym := range cyc {
				inCycle[sym] = true
			}
		}

		// Invariant 1: every node appears in either order or cycles, not both.
		for _, sym := range syms {
			_, isOrdered := inOrder[sym]
			isCyclic := inCycle[sym]
			if !isOrdered && !isCyclic {
				t.Fatalf("node %v appears in neither order nor cycles", sym)
			}
			if isOrdered && isCyclic {
				t.Fatalf("node %v appears in both order and cycles", sym)
			}
		}

		// Invariant 2: no duplicates in order.
		if len(inOrder) != len(order) {
			t.Fatalf("order has duplicates: %d unique, %d total", len(inOrder), len(order))
		}

		// Invariant 3: dependencies come before dependents in order.
		for edge := range edges {
			from, to := edge[0], edge[1]
			fromIdx, fromOk := inOrder[syms[from]]
			toIdx, toOk := inOrder[syms[to]]
			if fromOk && toOk {
				if toIdx > fromIdx {
					t.Fatalf("dependency ordering violated: %v (idx %d) depends on %v (idx %d)",
						syms[from], fromIdx, syms[to], toIdx)
				}
			}
		}

		// Invariant 4: each cycle has at least 1 node.
		for i, cyc := range cycles {
			if len(cyc) == 0 {
				t.Fatalf("cycle %d is empty", i)
			}
		}
	})
}
