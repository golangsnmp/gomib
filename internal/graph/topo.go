package graph

// TopologicalOrder returns symbols in an order where all dependencies come
// before the symbols that depend on them. Uses Kahn's algorithm.
//
// If there are cycles, the returned order will include all non-cyclic nodes
// in valid order, followed by the cyclic nodes. The second return value
// contains any symbols that couldn't be ordered due to cycles.
func (g *Graph) TopologicalOrder() (order []Symbol, cyclic []Symbol) {
	// Count incoming edges for each node
	inDegree := make(map[Symbol]int)
	for sym := range g.nodes {
		inDegree[sym] = 0
	}
	for _, deps := range g.edges {
		for _, dep := range deps {
			inDegree[dep]++
		}
	}

	// Start with nodes that have no incoming edges (no dependents)
	var queue []Symbol
	for sym, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, sym)
		}
	}

	// Process queue
	for len(queue) > 0 {
		sym := queue[0]
		queue = queue[1:]
		order = append(order, sym)

		// Reduce in-degree for all nodes this one depends on
		for _, dep := range g.edges[sym] {
			inDegree[dep]--
			if inDegree[dep] == 0 {
				queue = append(queue, dep)
			}
		}
	}

	// Any remaining nodes with in-degree > 0 are part of cycles
	for sym, degree := range inDegree {
		if degree > 0 {
			cyclic = append(cyclic, sym)
		}
	}

	return order, cyclic
}

// ResolutionOrder returns symbols in the order they should be resolved.
// Dependencies are returned before the symbols that depend on them.
// This is the reverse of TopologicalOrder (which orders dependents first).
func (g *Graph) ResolutionOrder() (order []Symbol, cyclic []Symbol) {
	topo, cyc := g.TopologicalOrder()
	// Reverse the order so dependencies come first
	for i := len(topo) - 1; i >= 0; i-- {
		order = append(order, topo[i])
	}
	return order, cyc
}
