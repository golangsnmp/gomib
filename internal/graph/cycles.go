package graph

// FindCycles uses Tarjan's algorithm to find all strongly connected components
// with more than one node (cycles).
func (g *Graph) FindCycles() [][]Symbol {
	var (
		index    int
		stack    []Symbol
		onStack  = make(map[Symbol]bool)
		indices  = make(map[Symbol]int)
		lowlinks = make(map[Symbol]int)
		sccs     [][]Symbol
	)

	var strongConnect func(sym Symbol)
	strongConnect = func(sym Symbol) {
		indices[sym] = index
		lowlinks[sym] = index
		index++
		stack = append(stack, sym)
		onStack[sym] = true

		for _, dep := range g.edges[sym] {
			if _, visited := indices[dep]; !visited {
				strongConnect(dep)
				if lowlinks[dep] < lowlinks[sym] {
					lowlinks[sym] = lowlinks[dep]
				}
			} else if onStack[dep] {
				if indices[dep] < lowlinks[sym] {
					lowlinks[sym] = indices[dep]
				}
			}
		}

		// If sym is a root node, pop the stack and generate an SCC
		if lowlinks[sym] == indices[sym] {
			var scc []Symbol
			for {
				w := stack[len(stack)-1]
				stack = stack[:len(stack)-1]
				onStack[w] = false
				scc = append(scc, w)
				if w == sym {
					break
				}
			}
			// Only report SCCs with more than one node (actual cycles)
			if len(scc) > 1 {
				sccs = append(sccs, scc)
			}
		}
	}

	for sym := range g.nodes {
		if _, visited := indices[sym]; !visited {
			strongConnect(sym)
		}
	}

	return sccs
}

// HasCycles returns true if the graph contains any cycles.
func (g *Graph) HasCycles() bool {
	return len(g.FindCycles()) > 0
}
