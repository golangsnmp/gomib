// Package graph provides dependency graph construction and analysis for MIB resolution.
package graph

// Symbol uniquely identifies a definition in a module.
type Symbol struct {
	Module string
	Name   string
}

// NodeKind classifies what kind of definition a node represents.
type NodeKind int

const (
	NodeKindType NodeKind = iota
	NodeKindOID
	NodeKindObject
	NodeKindNotification
)

// Graph is a dependency graph of symbols with forward and reverse edges.
type Graph struct {
	nodes   map[Symbol]*Node
	edges   map[Symbol][]Symbol
	reverse map[Symbol][]Symbol
}

// Node holds metadata about a symbol in the graph.
type Node struct {
	Symbol   Symbol
	Kind     NodeKind
	Resolved bool
}

// New creates an empty dependency graph.
func New() *Graph {
	return &Graph{
		nodes:   make(map[Symbol]*Node),
		edges:   make(map[Symbol][]Symbol),
		reverse: make(map[Symbol][]Symbol),
	}
}

// AddNode registers a symbol with its kind, if not already present.
func (g *Graph) AddNode(sym Symbol, kind NodeKind) {
	if _, exists := g.nodes[sym]; !exists {
		g.nodes[sym] = &Node{
			Symbol:   sym,
			Kind:     kind,
			Resolved: false,
		}
	}
}

// AddEdge records that "from" depends on "to", meaning "to" must be
// resolved before "from". Missing nodes are created implicitly.
func (g *Graph) AddEdge(from, to Symbol) {
	if _, ok := g.nodes[from]; !ok {
		g.nodes[from] = &Node{Symbol: from}
	}
	if _, ok := g.nodes[to]; !ok {
		g.nodes[to] = &Node{Symbol: to}
	}

	g.edges[from] = append(g.edges[from], to)
	g.reverse[to] = append(g.reverse[to], from)
}

// Node returns the metadata for a symbol, or nil if not present.
func (g *Graph) Node(sym Symbol) *Node {
	return g.nodes[sym]
}

// Dependencies returns the symbols that sym depends on (forward edges).
func (g *Graph) Dependencies(sym Symbol) []Symbol {
	return g.edges[sym]
}

// Dependents returns the symbols that depend on sym (reverse edges).
func (g *Graph) Dependents(sym Symbol) []Symbol {
	return g.reverse[sym]
}

// Nodes returns all registered nodes.
func (g *Graph) Nodes() []*Node {
	result := make([]*Node, 0, len(g.nodes))
	for _, n := range g.nodes {
		result = append(result, n)
	}
	return result
}

// MarkResolved flags a symbol as fully resolved.
func (g *Graph) MarkResolved(sym Symbol) {
	if n := g.nodes[sym]; n != nil {
		n.Resolved = true
	}
}

// IsResolved reports whether the symbol has been resolved.
func (g *Graph) IsResolved(sym Symbol) bool {
	if n := g.nodes[sym]; n != nil {
		return n.Resolved
	}
	return false
}
