package mibimpl

import (
	"iter"
	"strings"

	"github.com/golangsnmp/gomib/mib"
)

// Data is the concrete implementation of mib.Mib.
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

	diagnostics []mib.Diagnostic
	unresolved  []mib.UnresolvedRef
}

// Interface methods (mib.Mib)

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
	// Try qualified name first (MODULE::name)
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

	// Try numeric OID (starts with digit)
	if len(query) > 0 && query[0] >= '0' && query[0] <= '9' {
		oid, err := mib.ParseOID(query)
		if err != nil || len(oid) == 0 {
			return nil
		}
		return m.NodeByOID(oid)
	}

	// Try partial OID (starts with .)
	if len(query) > 0 && query[0] == '.' {
		oid, err := mib.ParseOID(query[1:])
		if err != nil || len(oid) == 0 {
			return nil
		}
		return m.NodeByOID(oid)
	}

	// Try name lookup - prefer object, then notification, then any node
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
	// Try qualified name first (MODULE::name)
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

	// Try name lookup
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
	result := make([]mib.Module, len(m.modules))
	for i, mod := range m.modules {
		result[i] = mod
	}
	return result
}

func (m *Data) Objects() []mib.Object {
	result := make([]mib.Object, len(m.objects))
	for i, obj := range m.objects {
		result[i] = obj
	}
	return result
}

func (m *Data) Types() []mib.Type {
	result := make([]mib.Type, len(m.types))
	for i, t := range m.types {
		result[i] = t
	}
	return result
}

func (m *Data) Notifications() []mib.Notification {
	result := make([]mib.Notification, len(m.notifications))
	for i, n := range m.notifications {
		result[i] = n
	}
	return result
}

func (m *Data) Tables() []mib.Object {
	var result []mib.Object
	for _, obj := range m.objects {
		if obj.IsTable() {
			result = append(result, obj)
		}
	}
	return result
}

func (m *Data) Scalars() []mib.Object {
	var result []mib.Object
	for _, obj := range m.objects {
		if obj.IsScalar() {
			result = append(result, obj)
		}
	}
	return result
}

func (m *Data) Columns() []mib.Object {
	var result []mib.Object
	for _, obj := range m.objects {
		if obj.IsColumn() {
			result = append(result, obj)
		}
	}
	return result
}

func (m *Data) Rows() []mib.Object {
	var result []mib.Object
	for _, obj := range m.objects {
		if obj.IsRow() {
			result = append(result, obj)
		}
	}
	return result
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
	result := make([]mib.Group, len(m.groups))
	for i, g := range m.groups {
		result[i] = g
	}
	return result
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
	result := make([]mib.Compliance, len(m.compliances))
	for i, c := range m.compliances {
		result[i] = c
	}
	return result
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
	result := make([]mib.Capabilities, len(m.capabilities))
	for i, c := range m.capabilities {
		result[i] = c
	}
	return result
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
	count := 0
	for range m.Nodes() {
		count++
	}
	return count
}

func (m *Data) Unresolved() []mib.UnresolvedRef {
	return m.unresolved
}

func (m *Data) Diagnostics() []mib.Diagnostic {
	return m.diagnostics
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

// Internal helpers

// nodeEntity is the constraint for types that live on a Node.
type nodeEntity interface {
	comparable
	*Object | *Notification | *Group | *Compliance | *Capabilities
}

// findEntity dispatches a query string to find a node-attached entity.
// It handles qualified names (MODULE::name), numeric OIDs, partial OIDs (.1.3...),
// and plain name lookups.
func findEntity[T nodeEntity](m *Data, query string, fromNode func(*Node) T) T {
	var zero T

	// Try qualified name (MODULE::name)
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

	// Try numeric OID (starts with digit)
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

	// Try partial OID (starts with .)
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

	// Try name lookup
	for _, nd := range m.nameToNodes[query] {
		if v := fromNode(nd); v != zero {
			return v
		}
	}
	return zero
}
