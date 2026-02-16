// Package graph provides dependency graph construction and analysis for MIB resolution.
package graph

import (
	"cmp"
	"slices"
)

// Symbol uniquely identifies a definition in a module.
type Symbol struct {
	Module string
	Name   string
}

// Graph is a dependency graph of symbols with forward edges.
// Internally, nodes are assigned integer IDs to avoid repeated string
// hashing during graph algorithms.
type Graph struct {
	nodeToID map[Symbol]int
	idToNode []Symbol
	edges    [][]int
}

// New returns a graph pre-sized for the expected number of nodes.
// Pass 0 if the size is unknown.
func New(sizeHint int) *Graph {
	return &Graph{
		nodeToID: make(map[Symbol]int, sizeHint),
		idToNode: make([]Symbol, 0, sizeHint),
		edges:    make([][]int, 0, sizeHint),
	}
}

// addSym returns the integer ID for sym, creating it if needed.
func (g *Graph) addSym(sym Symbol) int {
	if id, ok := g.nodeToID[sym]; ok {
		return id
	}
	id := len(g.idToNode)
	g.nodeToID[sym] = id
	g.idToNode = append(g.idToNode, sym)
	g.edges = append(g.edges, nil)
	return id
}

// AddNode registers a symbol. Duplicate calls are no-ops.
func (g *Graph) AddNode(sym Symbol) {
	g.addSym(sym)
}

// AddEdge records that "from" depends on "to", meaning "to" must be
// resolved before "from". Missing nodes are created implicitly.
// Duplicate edges are ignored.
func (g *Graph) AddEdge(from, to Symbol) {
	fromID := g.addSym(from)
	toID := g.addSym(to)

	if slices.Contains(g.edges[fromID], toID) {
		return
	}
	g.edges[fromID] = append(g.edges[fromID], toID)
}

// Dependencies returns the symbols that sym depends on (forward edges).
func (g *Graph) Dependencies(sym Symbol) []Symbol {
	id, ok := g.nodeToID[sym]
	if !ok {
		return nil
	}
	deps := g.edges[id]
	result := make([]Symbol, len(deps))
	for i, depID := range deps {
		result[i] = g.idToNode[depID]
	}
	return result
}

// HasNode reports whether the symbol exists in the graph.
func (g *Graph) HasNode(sym Symbol) bool {
	_, ok := g.nodeToID[sym]
	return ok
}

// ResolutionOrder returns symbols ordered so that dependencies come before
// dependents, using Tarjan's algorithm. Strongly connected components with
// more than one node (or a single node with a self-loop) are reported as
// cycles and excluded from the resolution order.
func (g *Graph) ResolutionOrder() (order []Symbol, cycles [][]Symbol) {
	n := len(g.idToNode)
	if n == 0 {
		return nil, nil
	}

	const unvisited = -1
	var (
		index    int
		stack    = make([]int, 0, n)
		onStack  = make([]bool, n)
		indices  = make([]int, n)
		lowlinks = make([]int, n)
	)

	for i := range indices {
		indices[i] = unvisited
	}

	var strongConnect func(id int)
	strongConnect = func(id int) {
		indices[id] = index
		lowlinks[id] = index
		index++
		stack = append(stack, id)
		onStack[id] = true

		for _, dep := range g.edges[id] {
			if indices[dep] == unvisited {
				strongConnect(dep)
				lowlinks[id] = min(lowlinks[id], lowlinks[dep])
			} else if onStack[dep] {
				lowlinks[id] = min(lowlinks[id], indices[dep])
			}
		}

		if lowlinks[id] == indices[id] {
			var scc []int
			for {
				w := stack[len(stack)-1]
				stack = stack[:len(stack)-1]
				onStack[w] = false
				scc = append(scc, w)
				if w == id {
					break
				}
			}
			if len(scc) > 1 {
				syms := make([]Symbol, len(scc))
				for i, sid := range scc {
					syms[i] = g.idToNode[sid]
				}
				cycles = append(cycles, syms)
			} else if slices.Contains(g.edges[scc[0]], scc[0]) {
				cycles = append(cycles, []Symbol{g.idToNode[scc[0]]})
			} else {
				order = append(order, g.idToNode[scc[0]])
			}
		}
	}

	// Sort node IDs for deterministic iteration order.
	sorted := make([]int, n)
	for i := range sorted {
		sorted[i] = i
	}
	slices.SortFunc(sorted, func(a, b int) int {
		sa, sb := g.idToNode[a], g.idToNode[b]
		if c := cmp.Compare(sa.Module, sb.Module); c != 0 {
			return c
		}
		return cmp.Compare(sa.Name, sb.Name)
	})

	for _, id := range sorted {
		if indices[id] == unvisited {
			strongConnect(id)
		}
	}

	return order, cycles
}
