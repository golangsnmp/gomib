package mibimpl

import (
	"cmp"
	"iter"
	"slices"

	"github.com/golangsnmp/gomib/mib"
)

// Node implements mib.Node, representing a single arc in the OID tree.
type Node struct {
	arc          uint32
	name         string
	kind         mib.Kind
	module       *Module
	obj          *Object
	notif        *Notification
	group        *Group
	compliance   *Compliance
	capabilities *Capabilities
	parent       *Node
	children     map[uint32]*Node
}

func (n *Node) Arc() uint32 {
	return n.arc
}

func (n *Node) Name() string {
	return n.name
}

func (n *Node) Kind() mib.Kind {
	return n.kind
}

func (n *Node) IsRoot() bool {
	return n.parent == nil
}

func (n *Node) Module() mib.Module {
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
	if n.capabilities != nil {
		return n.capabilities.module
	}
	if n.module != nil {
		return n.module
	}
	return nil
}

func (n *Node) OID() mib.Oid {
	if n.parent == nil {
		return nil
	}
	depth := 0
	for nd := n; nd.parent != nil; nd = nd.parent {
		depth++
	}
	oid := make(mib.Oid, depth)
	i := depth - 1
	for nd := n; nd.parent != nil; nd = nd.parent {
		oid[i] = nd.arc
		i--
	}
	return oid
}

func (n *Node) Object() mib.Object {
	if n.obj == nil {
		return nil
	}
	return n.obj
}

func (n *Node) Notification() mib.Notification {
	if n.notif == nil {
		return nil
	}
	return n.notif
}

func (n *Node) Group() mib.Group {
	if n.group == nil {
		return nil
	}
	return n.group
}

func (n *Node) Compliance() mib.Compliance {
	if n.compliance == nil {
		return nil
	}
	return n.compliance
}

func (n *Node) Capabilities() mib.Capabilities {
	if n.capabilities == nil {
		return nil
	}
	return n.capabilities
}

func (n *Node) Parent() mib.Node {
	if n.parent == nil {
		return nil
	}
	return n.parent
}

func (n *Node) Child(arc uint32) mib.Node {
	if n.children == nil {
		return nil
	}
	if c := n.children[arc]; c != nil {
		return c
	}
	return nil
}

func (n *Node) Children() []mib.Node {
	if len(n.children) == 0 {
		return nil
	}
	return mapSlice(n.sortedChildren(), func(v *Node) mib.Node { return v })
}

func (n *Node) sortedChildren() []*Node {
	if len(n.children) == 0 {
		return nil
	}
	result := make([]*Node, 0, len(n.children))
	for _, child := range n.children {
		result = append(result, child)
	}
	slices.SortFunc(result, func(a, b *Node) int {
		return cmp.Compare(a.arc, b.arc)
	})
	return result
}

func (n *Node) Descendants() iter.Seq[mib.Node] {
	return func(yield func(mib.Node) bool) {
		n.yieldAll(yield)
	}
}

func (n *Node) yieldAll(yield func(mib.Node) bool) bool {
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

func (n *Node) LongestPrefix(oid mib.Oid) mib.Node {
	if len(oid) == 0 {
		return nil
	}
	var deepest mib.Node
	current := n
	for _, arc := range oid {
		if current.children == nil {
			break
		}
		child := current.children[arc]
		if child == nil {
			break
		}
		current = child
		deepest = current
	}
	return deepest
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

// GetOrCreateChild returns the child at arc, creating it if absent.
func (n *Node) GetOrCreateChild(arc uint32) *Node {
	if n.children == nil {
		n.children = make(map[uint32]*Node)
	}
	if child, ok := n.children[arc]; ok {
		return child
	}
	child := &Node{
		arc:    arc,
		parent: n,
		kind:   mib.KindInternal,
	}
	n.children[arc] = child
	return child
}

func (n *Node) SetName(name string) {
	n.name = name
}

func (n *Node) SetKind(k mib.Kind) {
	n.kind = k
}

func (n *Node) SetModule(m *Module) {
	n.module = m
}

func (n *Node) SetObject(obj *Object) {
	n.obj = obj
}

func (n *Node) SetNotification(notif *Notification) {
	n.notif = notif
}

func (n *Node) SetGroup(g *Group) {
	n.group = g
}

func (n *Node) SetCompliance(c *Compliance) {
	n.compliance = c
}

func (n *Node) SetCapabilities(c *Capabilities) {
	n.capabilities = c
}

// InternalObject returns the concrete object.
func (n *Node) InternalObject() *Object {
	return n.obj
}

// InternalNotification returns the concrete notification.
func (n *Node) InternalNotification() *Notification {
	return n.notif
}

// InternalGroup returns the concrete group.
func (n *Node) InternalGroup() *Group {
	return n.group
}

// InternalCompliance returns the concrete compliance.
func (n *Node) InternalCompliance() *Compliance {
	return n.compliance
}

// InternalCapabilities returns the concrete capabilities.
func (n *Node) InternalCapabilities() *Capabilities {
	return n.capabilities
}

// InternalParent returns the concrete parent node.
func (n *Node) InternalParent() *Node {
	return n.parent
}

// InternalModule returns the concrete module.
func (n *Node) InternalModule() *Module {
	return n.module
}
