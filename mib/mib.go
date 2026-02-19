// Package mib provides the in-memory representation of loaded MIB data.
package mib

import (
	"iter"
	"slices"
	"strconv"
	"strings"
)

// Mib is the top-level container for loaded MIB data.
// It is intended to be built once and then used as a read-only structure.
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

// Root returns the unnamed root node of the OID tree.
func (m *Mib) Root() *Node { return m.root }

// Nodes returns an iterator over all nodes in the tree, depth-first by arc.
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

// NodeByOID returns the node at the exact OID, or nil if not found.
func (m *Mib) NodeByOID(oid OID) *Node {
	nd, ok := m.root.walkOID(oid)
	if !ok {
		return nil
	}
	return nd
}

// LongestPrefixByOID returns the deepest node matching a prefix of the OID.
func (m *Mib) LongestPrefixByOID(oid OID) *Node {
	nd, _ := m.root.walkOID(oid)
	return nd
}

// FormatOID translates a numeric OID into a human-readable string using
// the longest matching prefix in the OID tree. The result uses the form
// "MODULE::name.suffix" where suffix contains any unmatched trailing arcs.
// If no named node matches, returns the numeric OID string unchanged.
//
// Examples:
//
//	FormatOID({1,3,6,1,2,1,2,2,1,1,5}) => "IF-MIB::ifIndex.5"
//	FormatOID({1,3,6,1,2,1,2,2,1,1})   => "IF-MIB::ifIndex"
//	FormatOID({1,3,999})                => "1.3.999"
func (m *Mib) FormatOID(oid OID) string {
	if len(oid) == 0 {
		return ""
	}
	node := m.LongestPrefixByOID(oid)
	if node == nil || node.Name() == "" {
		return oid.String()
	}

	nodeOID := node.OID()
	suffix := oid[len(nodeOID):]

	var b strings.Builder
	if mod := node.Module(); mod != nil {
		b.WriteString(mod.Name())
		b.WriteString("::")
	}
	b.WriteString(node.Name())
	for _, arc := range suffix {
		b.WriteByte('.')
		b.WriteString(strconv.FormatUint(uint64(arc), 10))
	}
	return b.String()
}

// Module returns the module with the given name, or nil if not found.
func (m *Mib) Module(name string) *Module {
	return m.moduleByName[name]
}

// Modules returns a copy of all loaded modules.
func (m *Mib) Modules() []*Module { return slices.Clone(m.modules) }

// Objects returns a copy of all object definitions.
func (m *Mib) Objects() []*Object { return slices.Clone(m.objects) }

// Types returns a copy of all type definitions.
func (m *Mib) Types() []*Type { return slices.Clone(m.types) }

// Notifications returns a copy of all notification definitions.
func (m *Mib) Notifications() []*Notification { return slices.Clone(m.notifications) }

// Groups returns a copy of all group definitions.
func (m *Mib) Groups() []*Group { return slices.Clone(m.groups) }

// Compliances returns a copy of all compliance definitions.
func (m *Mib) Compliances() []*Compliance { return slices.Clone(m.compliances) }

// Capabilities returns a copy of all capability definitions.
func (m *Mib) Capabilities() []*Capability { return slices.Clone(m.capabilities) }

// Tables returns all objects classified as tables.
func (m *Mib) Tables() []*Object { return objectsByKind(m.objects, KindTable) }

// Scalars returns all objects classified as scalars.
func (m *Mib) Scalars() []*Object { return objectsByKind(m.objects, KindScalar) }

// Columns returns all objects classified as table columns.
func (m *Mib) Columns() []*Object { return objectsByKind(m.objects, KindColumn) }

// Rows returns all objects classified as table rows.
func (m *Mib) Rows() []*Object { return objectsByKind(m.objects, KindRow) }

// ObjectsByType returns all objects whose resolved type has the given name.
func (m *Mib) ObjectsByType(typeName string) []*Object {
	var result []*Object
	for _, obj := range m.objects {
		if obj.typ != nil && obj.typ.name == typeName {
			result = append(result, obj)
		}
	}
	return result
}

// ObjectsByBaseType returns all objects whose effective base type matches.
func (m *Mib) ObjectsByBaseType(base BaseType) []*Object {
	var result []*Object
	for _, obj := range m.objects {
		if obj.typ != nil && obj.typ.Base() == base {
			result = append(result, obj)
		}
	}
	return result
}

// NodeCount returns the total number of nodes in the OID tree.
func (m *Mib) NodeCount() int { return m.nodeCount }

// Unresolved returns a copy of all unresolved references from loading.
func (m *Mib) Unresolved() []UnresolvedRef { return slices.Clone(m.unresolved) }

// Diagnostics returns a copy of all diagnostics collected during loading.
func (m *Mib) Diagnostics() []Diagnostic { return slices.Clone(m.diagnostics) }

// HasErrors reports whether any diagnostic has error severity or above.
func (m *Mib) HasErrors() bool {
	return slices.ContainsFunc(m.diagnostics, func(d Diagnostic) bool {
		return d.Severity.AtLeast(SeverityError)
	})
}

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
