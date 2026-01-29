package mibimpl

import (
	"iter"
	"strings"

	"github.com/golangsnmp/gomib/mib"
)

// MibData is the concrete implementation of mib.Mib.
type MibData struct {
	root          *Node
	modules       []*Module
	objects       []*Object
	types         []*Type
	notifications []*Notification

	moduleByName map[string]*Module
	nameToNodes  map[string][]*Node
	typeByName   map[string]*Type

	diagnostics []mib.Diagnostic
	unresolved  []mib.UnresolvedRef
}

// Interface methods (mib.Mib)

func (m *MibData) Root() mib.Node {
	if m.root == nil {
		return nil
	}
	return m.root
}

func (m *MibData) Nodes() iter.Seq[mib.Node] {
	return func(yield func(mib.Node) bool) {
		for _, child := range m.root.sortedChildren() {
			if !child.yieldAll(yield) {
				return
			}
		}
	}
}

func (m *MibData) FindNode(query string) mib.Node {
	// Try qualified name first (MODULE::name)
	if idx := strings.Index(query, "::"); idx >= 0 {
		moduleName := query[:idx]
		nodeName := query[idx+2:]
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

func (m *MibData) FindObject(query string) mib.Object {
	// Try qualified name first (MODULE::name)
	if idx := strings.Index(query, "::"); idx >= 0 {
		obj := m.findObjectByQualified(query)
		if obj == nil {
			return nil
		}
		return obj
	}

	// Try numeric OID (starts with digit)
	if len(query) > 0 && query[0] >= '0' && query[0] <= '9' {
		oid, err := mib.ParseOID(query)
		if err != nil || len(oid) == 0 {
			return nil
		}
		nd := m.nodeByOID(oid)
		if nd == nil || nd.obj == nil {
			return nil
		}
		return nd.obj
	}

	// Try partial OID (starts with .)
	if len(query) > 0 && query[0] == '.' {
		oid, err := mib.ParseOID(query[1:])
		if err != nil || len(oid) == 0 {
			return nil
		}
		nd := m.nodeByOID(oid)
		if nd == nil || nd.obj == nil {
			return nil
		}
		return nd.obj
	}

	// Try name lookup
	nodes := m.nameToNodes[query]
	for _, nd := range nodes {
		if nd.obj != nil {
			return nd.obj
		}
	}
	return nil
}

func (m *MibData) FindType(query string) mib.Type {
	// Try qualified name first (MODULE::name)
	if idx := strings.Index(query, "::"); idx >= 0 {
		moduleName := query[:idx]
		typeName := query[idx+2:]
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

func (m *MibData) FindNotification(query string) mib.Notification {
	// Try qualified name first (MODULE::name)
	if idx := strings.Index(query, "::"); idx >= 0 {
		notif := m.findNotificationByQualified(query)
		if notif == nil {
			return nil
		}
		return notif
	}

	// Try numeric OID (starts with digit)
	if len(query) > 0 && query[0] >= '0' && query[0] <= '9' {
		oid, err := mib.ParseOID(query)
		if err != nil || len(oid) == 0 {
			return nil
		}
		nd := m.nodeByOID(oid)
		if nd == nil || nd.notif == nil {
			return nil
		}
		return nd.notif
	}

	// Try partial OID (starts with .)
	if len(query) > 0 && query[0] == '.' {
		oid, err := mib.ParseOID(query[1:])
		if err != nil || len(oid) == 0 {
			return nil
		}
		nd := m.nodeByOID(oid)
		if nd == nil || nd.notif == nil {
			return nil
		}
		return nd.notif
	}

	// Try name lookup
	nodes := m.nameToNodes[query]
	for _, nd := range nodes {
		if nd.notif != nil {
			return nd.notif
		}
	}
	return nil
}

func (m *MibData) NodeByOID(oid mib.Oid) mib.Node {
	nd := m.nodeByOID(oid)
	if nd == nil {
		return nil
	}
	return nd
}

func (m *MibData) nodeByOID(oid mib.Oid) *Node {
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

func (m *MibData) LongestPrefixByOID(oid mib.Oid) mib.Node {
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

func (m *MibData) Module(name string) mib.Module {
	mod := m.moduleByName[name]
	if mod == nil {
		return nil
	}
	return mod
}

func (m *MibData) Modules() []mib.Module {
	result := make([]mib.Module, len(m.modules))
	for i, mod := range m.modules {
		result[i] = mod
	}
	return result
}

func (m *MibData) Objects() []mib.Object {
	result := make([]mib.Object, len(m.objects))
	for i, obj := range m.objects {
		result[i] = obj
	}
	return result
}

func (m *MibData) Types() []mib.Type {
	result := make([]mib.Type, len(m.types))
	for i, t := range m.types {
		result[i] = t
	}
	return result
}

func (m *MibData) Notifications() []mib.Notification {
	result := make([]mib.Notification, len(m.notifications))
	for i, n := range m.notifications {
		result[i] = n
	}
	return result
}

func (m *MibData) Tables() []mib.Object {
	var result []mib.Object
	for _, obj := range m.objects {
		if obj.IsTable() {
			result = append(result, obj)
		}
	}
	return result
}

func (m *MibData) Scalars() []mib.Object {
	var result []mib.Object
	for _, obj := range m.objects {
		if obj.IsScalar() {
			result = append(result, obj)
		}
	}
	return result
}

func (m *MibData) Columns() []mib.Object {
	var result []mib.Object
	for _, obj := range m.objects {
		if obj.IsColumn() {
			result = append(result, obj)
		}
	}
	return result
}

func (m *MibData) Rows() []mib.Object {
	var result []mib.Object
	for _, obj := range m.objects {
		if obj.IsRow() {
			result = append(result, obj)
		}
	}
	return result
}

func (m *MibData) ModuleCount() int {
	return len(m.modules)
}

func (m *MibData) ObjectCount() int {
	return len(m.objects)
}

func (m *MibData) TypeCount() int {
	return len(m.types)
}

func (m *MibData) NotificationCount() int {
	return len(m.notifications)
}

func (m *MibData) NodeCount() int {
	count := 0
	for range m.Nodes() {
		count++
	}
	return count
}

func (m *MibData) Unresolved() []mib.UnresolvedRef {
	return m.unresolved
}

func (m *MibData) Diagnostics() []mib.Diagnostic {
	return m.diagnostics
}

func (m *MibData) HasErrors() bool {
	for _, d := range m.diagnostics {
		if d.Severity <= mib.SeverityError {
			return true
		}
	}
	return false
}

func (m *MibData) IsComplete() bool {
	return len(m.unresolved) == 0
}

// Internal helpers

func (m *MibData) findObjectByQualified(qname string) *Object {
	moduleName, objName, ok := parseQualifiedName(qname)
	if !ok {
		return nil
	}
	mod := m.moduleByName[moduleName]
	if mod == nil {
		return nil
	}
	for _, obj := range mod.objects {
		if obj.name == objName {
			return obj
		}
	}
	return nil
}

func (m *MibData) findNotificationByQualified(qname string) *Notification {
	moduleName, notifName, ok := parseQualifiedName(qname)
	if !ok {
		return nil
	}
	mod := m.moduleByName[moduleName]
	if mod == nil {
		return nil
	}
	for _, notif := range mod.notifications {
		if notif.name == notifName {
			return notif
		}
	}
	return nil
}

func parseQualifiedName(qname string) (module, name string, ok bool) {
	idx := strings.Index(qname, "::")
	if idx < 0 {
		return "", "", false
	}
	return qname[:idx], qname[idx+2:], true
}
