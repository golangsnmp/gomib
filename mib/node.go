package mib

import (
	"cmp"
	"iter"
	"maps"
	"slices"
)

// Node is a point in the OID tree. Each node has a numeric arc relative to
// its parent and an optional name. Nodes form a trie rooted at an unnamed
// root; the path from root to a node determines its OID. Entity definitions
// (Object, Notification, Group, Compliance, Capability) are attached to the
// node at their registered OID.
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

// Arc returns the numeric arc of this node relative to its parent.
func (n *Node) Arc() uint32 { return n.arc }

// Name returns the node's symbolic name, or "" if unnamed.
func (n *Node) Name() string { return n.name }

// Kind returns the structural classification of this node.
func (n *Node) Kind() Kind { return n.kind }

// IsRoot reports whether this is the unnamed root of the OID tree.
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

// OID returns the full numeric OID from the root to this node, or nil for the root.
func (n *Node) OID() OID {
	if n == nil || n.parent == nil {
		return nil
	}
	var arcs OID
	for nd := n; nd.parent != nil; nd = nd.parent {
		arcs = append(arcs, nd.arc)
	}
	slices.Reverse(arcs)
	return arcs
}

// Object returns the OBJECT-TYPE attached to this node, or nil.
func (n *Node) Object() *Object { return n.obj }

// Notification returns the NOTIFICATION-TYPE or TRAP-TYPE attached to this node, or nil.
func (n *Node) Notification() *Notification { return n.notif }

// Group returns the OBJECT-GROUP or NOTIFICATION-GROUP attached to this node, or nil.
func (n *Node) Group() *Group { return n.group }

// Compliance returns the MODULE-COMPLIANCE attached to this node, or nil.
func (n *Node) Compliance() *Compliance { return n.compliance }

// Capability returns the AGENT-CAPABILITIES attached to this node, or nil.
func (n *Node) Capability() *Capability { return n.capability }

// Parent returns the parent node, or nil for the root.
func (n *Node) Parent() *Node { return n.parent }

// Child returns the child node at the given arc, or nil if no such child exists.
func (n *Node) Child(arc uint32) *Node {
	if n.children == nil {
		return nil
	}
	return n.children[arc]
}

// Children returns the direct children of this node, sorted by arc.
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
	n.sortedCache = slices.SortedFunc(maps.Values(n.children), func(a, b *Node) int {
		return cmp.Compare(a.arc, b.arc)
	})
	return n.sortedCache
}

// Subtree returns an iterator over this node and all its descendants, depth-first.
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

// LongestPrefix returns the deepest descendant of this node matching a prefix of the OID.
func (n *Node) LongestPrefix(oid OID) *Node {
	nd, _ := n.walkOID(oid)
	return nd
}

// walkOID walks the OID tree from n, returning the last matched node
// and whether the full OID was matched.
func (n *Node) walkOID(oid OID) (matched *Node, exact bool) {
	current := n
	for _, arc := range oid {
		child := current.children[arc] // nil map yields nil
		if child == nil {
			return current, false
		}
		current = child
	}
	return current, true
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
