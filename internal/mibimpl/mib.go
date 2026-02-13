package mibimpl

import (
	"iter"
	"slices"
	"strings"

	"github.com/golangsnmp/gomib/mib"
)

// Data implements mib.Mib, holding all resolved modules, objects, types,
// and the OID tree.
type Data struct {
	root          *Node
	modules       []*Module
	objects       []*Object
	types         []*Type
	notifications []*Notification
	groups        []*Group
	compliances   []*Compliance
	capabilities  []*Capabilities

	moduleByName map[string]*Module
	nameToNodes  map[string][]*Node
	typeByName   map[string]*Type

	nodeCount   int
	diagnostics []mib.Diagnostic
	unresolved  []mib.UnresolvedRef
}

func (m *Data) Root() mib.Node {
	if m.root == nil {
		return nil
	}
	return m.root
}

func (m *Data) Nodes() iter.Seq[mib.Node] {
	return func(yield func(mib.Node) bool) {
		for _, child := range m.root.sortedChildren() {
			if !child.yieldAll(yield) {
				return
			}
		}
	}
}

func (m *Data) FindNode(query string) mib.Node {
	if moduleName, nodeName, ok := strings.Cut(query, "::"); ok {
		if m.moduleByName[moduleName] == nil {
			return nil
		}
		for _, nd := range m.nameToNodes[nodeName] {
			if nd.Module() != nil && nd.Module().Name() == moduleName {
				return nd
			}
		}
		return nil
	}

	if len(query) > 0 && query[0] >= '0' && query[0] <= '9' {
		oid, err := mib.ParseOID(query)
		if err != nil || len(oid) == 0 {
			return nil
		}
		return m.NodeByOID(oid)
	}

	if len(query) > 0 && query[0] == '.' {
		oid, err := mib.ParseOID(query[1:])
		if err != nil || len(oid) == 0 {
			return nil
		}
		return m.NodeByOID(oid)
	}

	// Prefer nodes with objects, then notifications, then any match.
	nodes := m.nameToNodes[query]
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

func (m *Data) FindObject(query string) mib.Object {
	if v := findEntity(m, query, func(nd *Node) *Object { return nd.obj }); v != nil {
		return v
	}
	return nil
}

func (m *Data) FindType(query string) mib.Type {
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

	t := m.typeByName[query]
	if t == nil {
		return nil
	}
	return t
}

func (m *Data) FindNotification(query string) mib.Notification {
	if v := findEntity(m, query, func(nd *Node) *Notification { return nd.notif }); v != nil {
		return v
	}
	return nil
}

func (m *Data) NodeByOID(oid mib.Oid) mib.Node {
	nd := m.nodeByOID(oid)
	if nd == nil {
		return nil
	}
	return nd
}

func (m *Data) nodeByOID(oid mib.Oid) *Node {
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

func (m *Data) LongestPrefixByOID(oid mib.Oid) mib.Node {
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
	if deepest == nil {
		return nil
	}
	return deepest
}

func (m *Data) Module(name string) mib.Module {
	mod := m.moduleByName[name]
	if mod == nil {
		return nil
	}
	return mod
}

func (m *Data) Modules() []mib.Module {
	return mapSlice(m.modules, func(v *Module) mib.Module { return v })
}

func (m *Data) Objects() []mib.Object {
	return mapSlice(m.objects, func(v *Object) mib.Object { return v })
}

func (m *Data) Types() []mib.Type {
	return mapSlice(m.types, func(v *Type) mib.Type { return v })
}

func (m *Data) Notifications() []mib.Notification {
	return mapSlice(m.notifications, func(v *Notification) mib.Notification { return v })
}

func (m *Data) Tables() []mib.Object {
	return objectsByKind(m.objects, mib.KindTable)
}

func (m *Data) Scalars() []mib.Object {
	return objectsByKind(m.objects, mib.KindScalar)
}

func (m *Data) Columns() []mib.Object {
	return objectsByKind(m.objects, mib.KindColumn)
}

func (m *Data) Rows() []mib.Object {
	return objectsByKind(m.objects, mib.KindRow)
}

func (m *Data) ModuleCount() int {
	return len(m.modules)
}

func (m *Data) ObjectCount() int {
	return len(m.objects)
}

func (m *Data) TypeCount() int {
	return len(m.types)
}

func (m *Data) NotificationCount() int {
	return len(m.notifications)
}

func (m *Data) Groups() []mib.Group {
	return mapSlice(m.groups, func(v *Group) mib.Group { return v })
}

func (m *Data) FindGroup(query string) mib.Group {
	if v := findEntity(m, query, func(nd *Node) *Group { return nd.group }); v != nil {
		return v
	}
	return nil
}

func (m *Data) GroupCount() int {
	return len(m.groups)
}

func (m *Data) Compliances() []mib.Compliance {
	return mapSlice(m.compliances, func(v *Compliance) mib.Compliance { return v })
}

func (m *Data) FindCompliance(query string) mib.Compliance {
	if v := findEntity(m, query, func(nd *Node) *Compliance { return nd.compliance }); v != nil {
		return v
	}
	return nil
}

func (m *Data) ComplianceCount() int {
	return len(m.compliances)
}

func (m *Data) Capabilities() []mib.Capabilities {
	return mapSlice(m.capabilities, func(v *Capabilities) mib.Capabilities { return v })
}

func (m *Data) FindCapabilities(query string) mib.Capabilities {
	if v := findEntity(m, query, func(nd *Node) *Capabilities { return nd.capabilities }); v != nil {
		return v
	}
	return nil
}

func (m *Data) CapabilitiesCount() int {
	return len(m.capabilities)
}

func (m *Data) NodeCount() int {
	return m.nodeCount
}

func (m *Data) Unresolved() []mib.UnresolvedRef {
	return slices.Clone(m.unresolved)
}

func (m *Data) Diagnostics() []mib.Diagnostic {
	return slices.Clone(m.diagnostics)
}

func (m *Data) HasErrors() bool {
	for _, d := range m.diagnostics {
		if d.Severity <= mib.SeverityError {
			return true
		}
	}
	return false
}

func (m *Data) IsComplete() bool {
	return len(m.unresolved) == 0
}

// nodeEntity constrains the entity types that can be attached to a Node.
type nodeEntity interface {
	comparable
	*Object | *Notification | *Group | *Compliance | *Capabilities
}

// findEntity resolves a query to a node-attached entity. It accepts
// qualified names (MODULE::name), numeric OIDs, dot-prefixed OIDs,
// and plain names.
func findEntity[T nodeEntity](m *Data, query string, fromNode func(*Node) T) T {
	var zero T

	if moduleName, itemName, ok := strings.Cut(query, "::"); ok {
		if m.moduleByName[moduleName] == nil {
			return zero
		}
		for _, nd := range m.nameToNodes[itemName] {
			if v := fromNode(nd); v != zero {
				if ndMod := nd.Module(); ndMod != nil && ndMod.Name() == moduleName {
					return v
				}
			}
		}
		return zero
	}

	if len(query) > 0 && query[0] >= '0' && query[0] <= '9' {
		oid, err := mib.ParseOID(query)
		if err != nil || len(oid) == 0 {
			return zero
		}
		if nd := m.nodeByOID(oid); nd != nil {
			return fromNode(nd)
		}
		return zero
	}

	if len(query) > 0 && query[0] == '.' {
		oid, err := mib.ParseOID(query[1:])
		if err != nil || len(oid) == 0 {
			return zero
		}
		if nd := m.nodeByOID(oid); nd != nil {
			return fromNode(nd)
		}
		return zero
	}

	for _, nd := range m.nameToNodes[query] {
		if v := fromNode(nd); v != zero {
			return v
		}
	}
	return zero
}
