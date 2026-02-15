package mib

import (
	"cmp"
	"iter"
	"slices"
)

// Node is a point in the OID tree.
type Node struct {
	arc         uint32
	name        string
	kind        Kind
	module      *Module
	obj         *Object
	notif       *Notification
	group       *Group
	compliance  *Compliance
	capability  *Capability
	parent      *Node
	children    map[uint32]*Node
	sortedCache []*Node // lazily computed sorted children; nil = invalidated
}

func (n *Node) Arc() uint32  { return n.arc }
func (n *Node) Name() string { return n.name }
func (n *Node) Kind() Kind   { return n.kind }
func (n *Node) IsRoot() bool { return n.parent == nil }

// Module returns the module that defines this node's primary entity.
// Priority: object > notification > group > compliance > capability > base module.
func (n *Node) Module() *Module {
	if n.obj != nil {
		return n.obj.module
	}
	if n.notif != nil {
		return n.notif.module
	}
	if n.group != nil {
		return n.group.module
	}
	if n.compliance != nil {
		return n.compliance.module
	}
	if n.capability != nil {
		return n.capability.module
	}
	return n.module
}

func (n *Node) OID() OID {
	if n.parent == nil {
		return nil
	}
	var arcs OID
	for nd := n; nd.parent != nil; nd = nd.parent {
		arcs = append(arcs, nd.arc)
	}
	slices.Reverse(arcs)
	return arcs
}

func (n *Node) Object() *Object             { return n.obj }
func (n *Node) Notification() *Notification { return n.notif }
func (n *Node) Group() *Group               { return n.group }
func (n *Node) Compliance() *Compliance     { return n.compliance }
func (n *Node) Capability() *Capability     { return n.capability }
func (n *Node) Parent() *Node               { return n.parent }

func (n *Node) Child(arc uint32) *Node {
	if n.children == nil {
		return nil
	}
	return n.children[arc]
}

func (n *Node) Children() []*Node {
	if len(n.children) == 0 {
		return nil
	}
	return slices.Clone(n.sortedChildren())
}

func (n *Node) sortedChildren() []*Node {
	if len(n.children) == 0 {
		return nil
	}
	if n.sortedCache != nil {
		return n.sortedCache
	}
	result := make([]*Node, 0, len(n.children))
	for _, child := range n.children {
		result = append(result, child)
	}
	slices.SortFunc(result, func(a, b *Node) int {
		return cmp.Compare(a.arc, b.arc)
	})
	n.sortedCache = result
	return result
}

func (n *Node) Subtree() iter.Seq[*Node] {
	return func(yield func(*Node) bool) {
		n.yieldAll(yield)
	}
}

func (n *Node) yieldAll(yield func(*Node) bool) bool {
	if !yield(n) {
		return false
	}
	for _, child := range n.sortedChildren() {
		if !child.yieldAll(yield) {
			return false
		}
	}
	return true
}

func (n *Node) LongestPrefix(oid OID) *Node {
	nd, _ := n.walkOID(oid)
	return nd
}

// walkOID walks the OID tree from n, returning the deepest node reached
// and whether the walk matched all arcs (exact match).
func (n *Node) walkOID(oid OID) (deepest *Node, exact bool) {
	current := n
	for _, arc := range oid {
		if current.children == nil {
			return deepest, false
		}
		child := current.children[arc]
		if child == nil {
			return deepest, false
		}
		current = child
		deepest = current
	}
	return deepest, true
}

// String returns a brief summary: "name (oid)" or just "(oid)" for
// unnamed nodes.
func (n *Node) String() string {
	if n == nil {
		return "<nil>"
	}
	if n.parent == nil {
		return "(root)"
	}
	if n.name == "" {
		return "(" + n.OID().String() + ")"
	}
	return n.name + " (" + n.OID().String() + ")"
}

// getOrCreateChild returns the child at arc, creating it if absent.
func (n *Node) getOrCreateChild(arc uint32) *Node {
	if n.children == nil {
		n.children = make(map[uint32]*Node)
	}
	if child, ok := n.children[arc]; ok {
		return child
	}
	child := &Node{
		arc:    arc,
		parent: n,
		kind:   KindInternal,
	}
	n.children[arc] = child
	n.sortedCache = nil
	return child
}

func (n *Node) setName(name string)                 { n.name = name }
func (n *Node) setKind(k Kind)                      { n.kind = k }
func (n *Node) setModule(m *Module)                 { n.module = m }
func (n *Node) setObject(obj *Object)               { n.obj = obj }
func (n *Node) setNotification(notif *Notification) { n.notif = notif }
func (n *Node) setGroup(g *Group)                   { n.group = g }
func (n *Node) setCompliance(c *Compliance)         { n.compliance = c }
func (n *Node) setCapability(c *Capability)         { n.capability = c }
