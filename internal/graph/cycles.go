package graph

// FindCycles returns all strongly connected components with more than one
// node, found via Tarjan's algorithm.
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
				lowlinks[sym] = min(lowlinks[sym], lowlinks[dep])
			} else if onStack[dep] {
				lowlinks[sym] = min(lowlinks[sym], indices[dep])
			}
		}

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
			if len(scc) > 1 {
				sccs = append(sccs, scc)
			} else if len(scc) == 1 {
				// Check for self-loop.
				for _, dep := range g.edges[scc[0]] {
					if dep == scc[0] {
						sccs = append(sccs, scc)
						break
					}
				}
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

// HasCycles reports whether the graph contains any cycles.
func (g *Graph) HasCycles() bool {
	return len(g.FindCycles()) > 0
}
