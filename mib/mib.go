package mib

import (
	"strings"
)

// Mib is the top-level container for loaded MIB data.
// It is immutable after construction and safe for concurrent reads.
type Mib struct {
	root          *Node // pseudo-root; parent of iso(1), ccitt(0), joint(2)
	modules       []*Module
	objects       []*Object
	types         []*Type
	notifications []*Notification

	moduleByName map[string]*Module
	nameToNodes  map[string][]*Node
	typeByName   map[string]*Type

	diagnostics []Diagnostic
	unresolved  []UnresolvedRef
}

// NewMib creates a new empty Mib with a pseudo-root.
// This is used by the resolver; most users should call Load() instead.
func NewMib() *Mib {
	return &Mib{
		root:         &Node{Kind: KindInternal},
		moduleByName: make(map[string]*Module),
		nameToNodes:  make(map[string][]*Node),
		typeByName:   make(map[string]*Type),
	}
}

// Root returns the pseudo-root of the OID tree.
// The root has no OID, no name, and its children are the top-level arcs (0, 1, 2).
func (m *Mib) Root() *Node {
	return m.root
}

// Node returns the node at the given OID string, or nil if not found.
func (m *Mib) Node(oidStr string) *Node {
	oid, err := ParseOID(oidStr)
	if err != nil || len(oid) == 0 {
		return nil
	}
	return m.NodeByOID(oid)
}

// NodeByOID returns the node at the given OID, or nil if not found.
func (m *Mib) NodeByOID(oid Oid) *Node {
	if len(oid) == 0 {
		return nil
	}
	node := m.root
	for _, arc := range oid {
		node = node.Child(arc)
		if node == nil {
			return nil
		}
	}
	return node
}

// Object returns the first object with the given name, or nil if not found.
func (m *Mib) Object(name string) *Object {
	nodes := m.nameToNodes[name]
	for _, node := range nodes {
		if node.Object != nil {
			return node.Object
		}
	}
	return nil
}

// ObjectsByName returns all objects with the given name.
func (m *Mib) ObjectsByName(name string) []*Object {
	nodes := m.nameToNodes[name]
	var result []*Object
	for _, node := range nodes {
		if node.Object != nil {
			result = append(result, node.Object)
		}
	}
	return result
}

// ObjectByOID returns the object at the given OID string, or nil if not found.
func (m *Mib) ObjectByOID(oidStr string) *Object {
	node := m.Node(oidStr)
	if node == nil {
		return nil
	}
	return node.Object
}

// ObjectByQualified returns the object with the given qualified name (MODULE::name).
func (m *Mib) ObjectByQualified(qname string) *Object {
	moduleName, objName, ok := parseQualifiedName(qname)
	if !ok {
		return nil
	}
	mod := m.moduleByName[moduleName]
	if mod == nil {
		return nil
	}
	for _, obj := range mod.objects {
		if obj.Name == objName {
			return obj
		}
	}
	return nil
}

// Type returns the first type with the given name, or nil if not found.
func (m *Mib) Type(name string) *Type {
	return m.typeByName[name]
}

// Notification returns the first notification with the given name, or nil if not found.
func (m *Mib) Notification(name string) *Notification {
	nodes := m.nameToNodes[name]
	for _, node := range nodes {
		if node.Notif != nil {
			return node.Notif
		}
	}
	return nil
}

// NotificationByQualified returns the notification with the given qualified name (MODULE::name).
func (m *Mib) NotificationByQualified(qname string) *Notification {
	moduleName, notifName, ok := parseQualifiedName(qname)
	if !ok {
		return nil
	}
	mod := m.moduleByName[moduleName]
	if mod == nil {
		return nil
	}
	for _, notif := range mod.notifications {
		if notif.Name == notifName {
			return notif
		}
	}
	return nil
}

// FindNode looks up a node by OID, name, or qualified name.
// It tries multiple query formats in order:
//   - Qualified name: "MODULE::name" (e.g., "IF-MIB::ifIndex")
//   - Numeric OID: "1.3.6.1.2.1.2.2.1.1"
//   - Partial OID: ".1.2.1.2" (leading dot stripped)
//   - Simple name: "ifIndex" (searches objects then notifications)
//
// Returns nil if no matching node is found.
func (m *Mib) FindNode(query string) *Node {
	// Try qualified name first (MODULE::name)
	if idx := strings.Index(query, "::"); idx >= 0 {
		obj := m.ObjectByQualified(query)
		if obj != nil {
			return obj.Node
		}
		notif := m.NotificationByQualified(query)
		if notif != nil {
			return notif.Node
		}
		return nil
	}

	// Try numeric OID (starts with digit)
	if len(query) > 0 && query[0] >= '0' && query[0] <= '9' {
		return m.Node(query)
	}

	// Try partial OID (starts with .)
	if len(query) > 0 && query[0] == '.' {
		return m.Node(query[1:])
	}

	// Try name lookup - object first
	obj := m.Object(query)
	if obj != nil {
		return obj.Node
	}

	// Try notification
	notif := m.Notification(query)
	if notif != nil {
		return notif.Node
	}

	return nil
}

// Module returns the module with the given name, or nil if not found.
func (m *Mib) Module(name string) *Module {
	return m.moduleByName[name]
}

// Walk traverses the entire OID tree in pre-order, starting from the pseudo-root's children.
// Return false from the callback to stop walking.
func (m *Mib) Walk(fn func(*Node) bool) {
	for _, child := range m.root.Children() {
		if !child.walk(fn) {
			return
		}
	}
}

// Objects returns all objects.
func (m *Mib) Objects() []*Object {
	return m.objects
}

// Types returns all types.
func (m *Mib) Types() []*Type {
	return m.types
}

// Notifications returns all notifications.
func (m *Mib) Notifications() []*Notification {
	return m.notifications
}

// Modules returns all modules.
func (m *Mib) Modules() []*Module {
	return m.modules
}

// Diagnostics returns all parse/resolve diagnostics.
func (m *Mib) Diagnostics() []Diagnostic {
	return m.diagnostics
}

// IsComplete returns true if resolution completed without unresolved references.
func (m *Mib) IsComplete() bool {
	return len(m.unresolved) == 0
}

// Unresolved returns all unresolved references.
func (m *Mib) Unresolved() []UnresolvedRef {
	return m.unresolved
}

// --- Internal helpers ---

// parseQualifiedName splits "MODULE::name" into (module, name, ok).
func parseQualifiedName(qname string) (module, name string, ok bool) {
	idx := strings.Index(qname, "::")
	if idx < 0 {
		return "", "", false
	}
	return qname[:idx], qname[idx+2:], true
}

// RegisterNode adds a node to the name index.
func (m *Mib) RegisterNode(name string, node *Node) {
	if name != "" {
		m.nameToNodes[name] = append(m.nameToNodes[name], node)
	}
}

// GetOrCreateNode returns the node at the given OID, creating nodes along the path as needed.
func (m *Mib) GetOrCreateNode(oid Oid) *Node {
	node := m.root
	for _, arc := range oid {
		node = node.GetOrCreateChild(arc)
	}
	return node
}

// GetOrCreateRoot returns the root node with the given arc (0, 1, or 2), creating if needed.
func (m *Mib) GetOrCreateRoot(arc uint32) *Node {
	return m.root.GetOrCreateChild(arc)
}

// AddModule adds a module to the Mib.
func (m *Mib) AddModule(mod *Module) {
	m.modules = append(m.modules, mod)
	if mod.Name != "" {
		m.moduleByName[mod.Name] = mod
	}
}

// AddType adds a type to the Mib.
func (m *Mib) AddType(t *Type) {
	m.types = append(m.types, t)
	if t.Name != "" && m.typeByName[t.Name] == nil {
		m.typeByName[t.Name] = t
	}
}

// AddObject adds an object to the Mib.
func (m *Mib) AddObject(obj *Object) {
	m.objects = append(m.objects, obj)
}

// AddNotification adds a notification to the Mib.
func (m *Mib) AddNotification(n *Notification) {
	m.notifications = append(m.notifications, n)
}

// AddUnresolved adds an unresolved reference.
func (m *Mib) AddUnresolved(ref UnresolvedRef) {
	m.unresolved = append(m.unresolved, ref)
}

// AddDiagnostic adds a diagnostic message.
func (m *Mib) AddDiagnostic(d Diagnostic) {
	m.diagnostics = append(m.diagnostics, d)
}

// ModuleCount returns the number of modules.
func (m *Mib) ModuleCount() int {
	return len(m.modules)
}

// TypeCount returns the number of types.
func (m *Mib) TypeCount() int {
	return len(m.types)
}

// ObjectCount returns the number of objects.
func (m *Mib) ObjectCount() int {
	return len(m.objects)
}

// NotificationCount returns the number of notifications.
func (m *Mib) NotificationCount() int {
	return len(m.notifications)
}

// NodeCount returns the total number of nodes in the tree.
func (m *Mib) NodeCount() int {
	count := 0
	m.root.Walk(func(*Node) bool {
		count++
		return true
	})
	return count
}
