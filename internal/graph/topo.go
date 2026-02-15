package graph

import "slices"

// TopologicalOrder returns symbols ordered so that dependents come before
// their dependencies (Kahn's algorithm). Symbols involved in cycles are
// returned separately in the second slice.
func (g *Graph) TopologicalOrder() (order []Symbol, cyclic []Symbol) {
	inDegree := make(map[Symbol]int)
	for _, deps := range g.edges {
		for _, dep := range deps {
			inDegree[dep]++
		}
	}

	var queue []Symbol
	for sym := range g.nodes {
		if inDegree[sym] == 0 {
			queue = append(queue, sym)
		}
	}

	for len(queue) > 0 {
		sym := queue[0]
		queue = queue[1:]
		order = append(order, sym)

		for _, dep := range g.edges[sym] {
			inDegree[dep]--
			if inDegree[dep] == 0 {
				queue = append(queue, dep)
			}
		}
	}

	for sym, degree := range inDegree {
		if degree > 0 {
			cyclic = append(cyclic, sym)
		}
	}

	return order, cyclic
}

// ResolutionOrder returns symbols with dependencies before dependents,
// the reverse of TopologicalOrder.
func (g *Graph) ResolutionOrder() (order []Symbol, cyclic []Symbol) {
	order, cyclic = g.TopologicalOrder()
	slices.Reverse(order)
	return order, cyclic
}
