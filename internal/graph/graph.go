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
type Graph struct {
	nodes map[Symbol]struct{}
	edges map[Symbol][]Symbol
}

// New returns a graph with no nodes or edges.
func New() *Graph {
	return &Graph{
		nodes: make(map[Symbol]struct{}),
		edges: make(map[Symbol][]Symbol),
	}
}

// AddNode registers a symbol. Duplicate calls are no-ops.
func (g *Graph) AddNode(sym Symbol) {
	g.nodes[sym] = struct{}{}
}

// AddEdge records that "from" depends on "to", meaning "to" must be
// resolved before "from". Missing nodes are created implicitly.
// Duplicate edges are ignored.
func (g *Graph) AddEdge(from, to Symbol) {
	g.nodes[from] = struct{}{}
	g.nodes[to] = struct{}{}

	if slices.Contains(g.edges[from], to) {
		return
	}
	g.edges[from] = append(g.edges[from], to)
}

// Dependencies returns the symbols that sym depends on (forward edges).
func (g *Graph) Dependencies(sym Symbol) []Symbol {
	return g.edges[sym]
}

// HasNode reports whether the symbol exists in the graph.
func (g *Graph) HasNode(sym Symbol) bool {
	_, ok := g.nodes[sym]
	return ok
}

// ResolutionOrder returns symbols ordered so that dependencies come before
// dependents, using Tarjan's algorithm. Strongly connected components with
// more than one node (or a single node with a self-loop) are reported as
// cycles and excluded from the resolution order.
func (g *Graph) ResolutionOrder() (order []Symbol, cycles [][]Symbol) {
	var (
		index    int
		stack    []Symbol
		onStack  = make(map[Symbol]bool)
		indices  = make(map[Symbol]int)
		lowlinks = make(map[Symbol]int)
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
				cycles = append(cycles, scc)
			} else if slices.Contains(g.edges[scc[0]], scc[0]) {
				cycles = append(cycles, scc)
			} else {
				order = append(order, scc[0])
			}
		}
	}

	sorted := make([]Symbol, 0, len(g.nodes))
	for sym := range g.nodes {
		sorted = append(sorted, sym)
	}
	slices.SortFunc(sorted, func(a, b Symbol) int {
		if c := cmp.Compare(a.Module, b.Module); c != 0 {
			return c
		}
		return cmp.Compare(a.Name, b.Name)
	})

	for _, sym := range sorted {
		if _, visited := indices[sym]; !visited {
			strongConnect(sym)
		}
	}

	return order, cycles
}
