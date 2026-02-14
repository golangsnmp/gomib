package mib

import (
	"iter"
	"slices"
	"strings"
)

// Mib is the top-level container for loaded MIB data.
// It is immutable after construction and safe for concurrent reads.
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

// NewMib returns an empty, initialized Mib.
func NewMib() *Mib {
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

// resolveQuery parses a query string and returns matching nodes.
func (m *Mib) resolveQuery(query string) (nodes []*Node, moduleName string) {
	if modName, itemName, ok := strings.Cut(query, "::"); ok {
		if m.moduleByName[modName] == nil {
			return nil, modName
		}
		return m.nameToNodes[itemName], modName
	}

	q := query
	if len(q) > 0 && q[0] == '.' {
		q = q[1:]
	}
	if len(q) > 0 && q[0] >= '0' && q[0] <= '9' {
		oid, err := ParseOID(q)
		if err != nil || len(oid) == 0 {
			return nil, ""
		}
		if nd := m.nodeByOID(oid); nd != nil {
			return []*Node{nd}, ""
		}
		return nil, ""
	}

	return m.nameToNodes[query], ""
}

func (m *Mib) Node(query string) *Node {
	nodes, moduleName := m.resolveQuery(query)
	if moduleName != "" {
		for _, nd := range nodes {
			if mod := nd.Module(); mod != nil && mod.Name() == moduleName {
				return nd
			}
		}
		return nil
	}
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

func (m *Mib) Object(query string) *Object {
	return findEntity(m, query, func(nd *Node) *Object { return nd.obj })
}

func (m *Mib) Type(query string) *Type {
	if moduleName, typeName, ok := strings.Cut(query, "::"); ok {
		mod := m.moduleByName[moduleName]
		if mod == nil {
			return nil
		}
		for _, t := range mod.types {
			if t.name == typeName {
				return t
			}
		}
		return nil
	}
	return m.typeByName[query]
}

func (m *Mib) Notification(query string) *Notification {
	return findEntity(m, query, func(nd *Node) *Notification { return nd.notif })
}

func (m *Mib) Group(query string) *Group {
	return findEntity(m, query, func(nd *Node) *Group { return nd.group })
}

func (m *Mib) Compliance(query string) *Compliance {
	return findEntity(m, query, func(nd *Node) *Compliance { return nd.compliance })
}

func (m *Mib) Capability(query string) *Capability {
	return findEntity(m, query, func(nd *Node) *Capability { return nd.capabilities })
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

func (m *Mib) SetRoot(r *Node)    { m.root = r }
func (m *Mib) SetNodeCount(n int) { m.nodeCount = n }

func (m *Mib) AddModule(mod *Module) {
	m.modules = append(m.modules, mod)
	if mod.name != "" {
		m.moduleByName[mod.name] = mod
	}
}

func (m *Mib) AddObject(obj *Object) {
	m.objects = append(m.objects, obj)
}

func (m *Mib) AddType(t *Type) {
	m.types = append(m.types, t)
	if t.name != "" && m.typeByName[t.name] == nil {
		m.typeByName[t.name] = t
	}
}

func (m *Mib) AddNotification(n *Notification) {
	m.notifications = append(m.notifications, n)
}

func (m *Mib) AddGroup(g *Group) {
	m.groups = append(m.groups, g)
}

func (m *Mib) AddCompliance(c *Compliance) {
	m.compliances = append(m.compliances, c)
}

func (m *Mib) AddCapability(c *Capability) {
	m.capabilities = append(m.capabilities, c)
}

func (m *Mib) RegisterNode(name string, n *Node) {
	if name != "" {
		m.nameToNodes[name] = append(m.nameToNodes[name], n)
	}
}

func (m *Mib) AddDiagnostic(d Diagnostic) {
	m.diagnostics = append(m.diagnostics, d)
}

func (m *Mib) AddUnresolved(ref UnresolvedRef) {
	m.unresolved = append(m.unresolved, ref)
}

// nodeEntity constrains the entity types that can be attached to a Node.
type nodeEntity interface {
	comparable
	*Object | *Notification | *Group | *Compliance | *Capability
}

// findEntity resolves a query to a node-attached entity using resolveQuery.
func findEntity[T nodeEntity](m *Mib, query string, fromNode func(*Node) T) T {
	var zero T
	nodes, moduleName := m.resolveQuery(query)
	for _, nd := range nodes {
		if v := fromNode(nd); v != zero {
			if moduleName != "" {
				if ndMod := nd.Module(); ndMod == nil || ndMod.Name() != moduleName {
					continue
				}
			}
			return v
		}
	}
	return zero
}
