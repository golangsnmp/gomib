package mib

import (
	"iter"
	"slices"
)

// Mib is the top-level container for loaded MIB data.
type Mib struct {
	root          *Node
	modules       []*Module
	objects       []*Object
	types         []*Type
	notifications []*Notification
	groups        []*Group
	compliances   []*Compliance
	capabilities  []*Capability

	moduleByName map[string]*Module
	nameToNodes  map[string][]*Node
	typeByName   map[string]*Type

	nodeCount   int
	diagnostics []Diagnostic
	unresolved  []UnresolvedRef
}

// newMib returns an empty, initialized Mib.
func newMib() *Mib {
	return &Mib{
		root:         &Node{kind: KindInternal},
		moduleByName: make(map[string]*Module),
		nameToNodes:  make(map[string][]*Node),
		typeByName:   make(map[string]*Type),
	}
}

func (m *Mib) Root() *Node { return m.root }

func (m *Mib) Nodes() iter.Seq[*Node] {
	return func(yield func(*Node) bool) {
		for _, child := range m.root.sortedChildren() {
			if !child.yieldAll(yield) {
				return
			}
		}
	}
}

// Node returns the node with the given name, or nil if not found.
// Prefers nodes with object definitions, then notifications, then any.
func (m *Mib) Node(name string) *Node {
	nodes := m.nameToNodes[name]
	for _, nd := range nodes {
		if nd.obj != nil {
			return nd
		}
	}
	for _, nd := range nodes {
		if nd.notif != nil {
			return nd
		}
	}
	if len(nodes) > 0 {
		return nodes[0]
	}
	return nil
}

// Object returns the object with the given name, or nil if not found.
func (m *Mib) Object(name string) *Object {
	return findEntity(m, name, func(nd *Node) *Object { return nd.obj })
}

// Type returns the type with the given name, or nil if not found.
func (m *Mib) Type(name string) *Type {
	return m.typeByName[name]
}

// Notification returns the notification with the given name, or nil if not found.
func (m *Mib) Notification(name string) *Notification {
	return findEntity(m, name, func(nd *Node) *Notification { return nd.notif })
}

// Group returns the group with the given name, or nil if not found.
func (m *Mib) Group(name string) *Group {
	return findEntity(m, name, func(nd *Node) *Group { return nd.group })
}

// Compliance returns the compliance with the given name, or nil if not found.
func (m *Mib) Compliance(name string) *Compliance {
	return findEntity(m, name, func(nd *Node) *Compliance { return nd.compliance })
}

// Capability returns the capability with the given name, or nil if not found.
func (m *Mib) Capability(name string) *Capability {
	return findEntity(m, name, func(nd *Node) *Capability { return nd.capability })
}

func (m *Mib) NodeByOID(oid OID) *Node {
	return m.nodeByOID(oid)
}

func (m *Mib) nodeByOID(oid OID) *Node {
	if len(oid) == 0 {
		return nil
	}
	nd := m.root
	for _, arc := range oid {
		if nd.children == nil {
			return nil
		}
		child := nd.children[arc]
		if child == nil {
			return nil
		}
		nd = child
	}
	return nd
}

func (m *Mib) LongestPrefixByOID(oid OID) *Node {
	if len(oid) == 0 {
		return nil
	}
	var deepest *Node
	nd := m.root
	for _, arc := range oid {
		if nd.children == nil {
			break
		}
		child := nd.children[arc]
		if child == nil {
			break
		}
		nd = child
		deepest = nd
	}
	return deepest
}

func (m *Mib) Module(name string) *Module {
	return m.moduleByName[name]
}

func (m *Mib) Modules() []*Module             { return slices.Clone(m.modules) }
func (m *Mib) Objects() []*Object             { return slices.Clone(m.objects) }
func (m *Mib) Types() []*Type                 { return slices.Clone(m.types) }
func (m *Mib) Notifications() []*Notification { return slices.Clone(m.notifications) }
func (m *Mib) Groups() []*Group               { return slices.Clone(m.groups) }
func (m *Mib) Compliances() []*Compliance     { return slices.Clone(m.compliances) }
func (m *Mib) Capabilities() []*Capability    { return slices.Clone(m.capabilities) }

func (m *Mib) Tables() []*Object  { return objectsByKind(m.objects, KindTable) }
func (m *Mib) Scalars() []*Object { return objectsByKind(m.objects, KindScalar) }
func (m *Mib) Columns() []*Object { return objectsByKind(m.objects, KindColumn) }
func (m *Mib) Rows() []*Object    { return objectsByKind(m.objects, KindRow) }

func (m *Mib) NodeCount() int              { return m.nodeCount }
func (m *Mib) Unresolved() []UnresolvedRef { return slices.Clone(m.unresolved) }
func (m *Mib) Diagnostics() []Diagnostic   { return slices.Clone(m.diagnostics) }

func (m *Mib) HasErrors() bool {
	for _, d := range m.diagnostics {
		if d.Severity <= SeverityError {
			return true
		}
	}
	return false
}

// Construction methods used by the builder/resolver.

func (m *Mib) setNodeCount(n int) { m.nodeCount = n }

func (m *Mib) addModule(mod *Module) {
	m.modules = append(m.modules, mod)
	if mod.name != "" {
		m.moduleByName[mod.name] = mod
	}
}

func (m *Mib) addObject(obj *Object) {
	m.objects = append(m.objects, obj)
}

func (m *Mib) addType(t *Type) {
	m.types = append(m.types, t)
	if t.name != "" && m.typeByName[t.name] == nil {
		m.typeByName[t.name] = t
	}
}

func (m *Mib) addNotification(n *Notification) {
	m.notifications = append(m.notifications, n)
}

func (m *Mib) addGroup(g *Group) {
	m.groups = append(m.groups, g)
}

func (m *Mib) addCompliance(c *Compliance) {
	m.compliances = append(m.compliances, c)
}

func (m *Mib) addCapability(c *Capability) {
	m.capabilities = append(m.capabilities, c)
}

func (m *Mib) registerNode(name string, n *Node) {
	if name != "" {
		m.nameToNodes[name] = append(m.nameToNodes[name], n)
	}
}

func (m *Mib) addDiagnostic(d Diagnostic) {
	m.diagnostics = append(m.diagnostics, d)
}

func (m *Mib) addUnresolved(ref UnresolvedRef) {
	m.unresolved = append(m.unresolved, ref)
}

// nodeEntity constrains the entity types that can be attached to a Node.
type nodeEntity interface {
	comparable
	*Object | *Notification | *Group | *Compliance | *Capability
}

// findEntity looks up a node-attached entity by name.
func findEntity[T nodeEntity](m *Mib, name string, fromNode func(*Node) T) T {
	var zero T
	for _, nd := range m.nameToNodes[name] {
		if v := fromNode(nd); v != zero {
			return v
		}
	}
	return zero
}
