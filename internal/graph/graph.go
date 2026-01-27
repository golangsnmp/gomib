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

// Graph is a dependency graph of symbols.
type Graph struct {
	// nodes maps symbols to their metadata
	nodes map[Symbol]*Node
	// edges maps each symbol to symbols it depends on
	edges map[Symbol][]Symbol
	// reverse maps each symbol to symbols that depend on it
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

// AddNode adds a node to the graph.
func (g *Graph) AddNode(sym Symbol, kind NodeKind) {
	if _, exists := g.nodes[sym]; !exists {
		g.nodes[sym] = &Node{
			Symbol:   sym,
			Kind:     kind,
			Resolved: false,
		}
	}
}

// AddEdge adds a dependency edge from -> to.
// This means 'from' depends on 'to' (to must be resolved before from).
func (g *Graph) AddEdge(from, to Symbol) {
	// Ensure nodes exist
	if _, ok := g.nodes[from]; !ok {
		g.nodes[from] = &Node{Symbol: from}
	}
	if _, ok := g.nodes[to]; !ok {
		g.nodes[to] = &Node{Symbol: to}
	}

	// Add forward edge
	g.edges[from] = append(g.edges[from], to)
	// Add reverse edge
	g.reverse[to] = append(g.reverse[to], from)
}

// Node returns the node for a symbol, or nil if not found.
func (g *Graph) Node(sym Symbol) *Node {
	return g.nodes[sym]
}

// Dependencies returns the symbols that sym depends on.
func (g *Graph) Dependencies(sym Symbol) []Symbol {
	return g.edges[sym]
}

// Dependents returns the symbols that depend on sym.
func (g *Graph) Dependents(sym Symbol) []Symbol {
	return g.reverse[sym]
}

// Nodes returns all nodes in the graph.
func (g *Graph) Nodes() []*Node {
	result := make([]*Node, 0, len(g.nodes))
	for _, n := range g.nodes {
		result = append(result, n)
	}
	return result
}

// MarkResolved marks a symbol as resolved.
func (g *Graph) MarkResolved(sym Symbol) {
	if n := g.nodes[sym]; n != nil {
		n.Resolved = true
	}
}

// IsResolved returns true if the symbol is marked as resolved.
func (g *Graph) IsResolved(sym Symbol) bool {
	if n := g.nodes[sym]; n != nil {
		return n.Resolved
	}
	return false
}
