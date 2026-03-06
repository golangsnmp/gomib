// Package mib provides the resolved, read-only model produced by loading
// SNMP MIB modules. The central type is [Mib], which contains the OID tree,
// object definitions, type definitions, notifications, groups, compliance
// statements, and agent capabilities extracted from one or more MIB files.
//
// The main model types ([Mib], [Node], [Object], [Type], [Module],
// [Notification], [Group], [Compliance], [Capability]) use unexported fields
// with exported accessor methods. Small value structs ([Range], [NamedValue],
// [Revision], [IndexEntry], [TrapInfo], etc.) use exported fields directly.
// Slice-returning accessors return cloned copies, so callers may freely
// modify the returned slices without affecting the model.
package mib

import (
	"errors"
	"fmt"
	"iter"
	"slices"
	"strconv"
	"strings"
)

// Mib is the top-level container returned by [gomib.Load]. It holds the
// merged OID tree, all resolved definitions (objects, types, notifications,
// groups, compliances, capabilities), and any diagnostics produced during
// loading. A Mib is built once during resolution and is safe for concurrent
// reads from multiple goroutines.
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

// AllSymbols returns an iterator over all definitions across all loaded modules.
// Yields each module's definitions (via [Module.Definitions]) in module-list order.
func (m *Mib) AllSymbols() iter.Seq[Symbol] {
	return func(yield func(Symbol) bool) {
		for _, mod := range m.modules {
			for sym := range mod.Definitions() {
				if !yield(sym) {
					return
				}
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

// Symbol returns the resolved definition for name. Lookup priority:
// node entities (object > notification > group > compliance > capability > plain node),
// then Type. Returns a zero Symbol if not found.
func (m *Mib) Symbol(name string) Symbol {
	// Check node-attached entities first.
	for _, nd := range m.nameToNodes[name] {
		if nd.obj != nil {
			return symbolFromObject(nd.obj)
		}
	}
	for _, nd := range m.nameToNodes[name] {
		if nd.notif != nil {
			return symbolFromNotification(nd.notif)
		}
	}
	for _, nd := range m.nameToNodes[name] {
		if nd.group != nil {
			return symbolFromGroup(nd.group)
		}
	}
	for _, nd := range m.nameToNodes[name] {
		if nd.compliance != nil {
			return symbolFromCompliance(nd.compliance)
		}
	}
	for _, nd := range m.nameToNodes[name] {
		if nd.capability != nil {
			return symbolFromCapability(nd.capability)
		}
	}
	// Plain node (no entity attachment).
	if nodes := m.nameToNodes[name]; len(nodes) > 0 {
		return symbolFromNode(nodes[0])
	}
	// Type lookup.
	if t := m.typeByName[name]; t != nil {
		return symbolFromType(t)
	}
	return Symbol{}
}

// ModulesDefining returns all modules that define a symbol with the given name.
// Excludes synthetic base modules.
func (m *Mib) ModulesDefining(name string) []*Module {
	var result []*Module
	for _, mod := range m.modules {
		if mod.base {
			continue
		}
		if mod.DefinesSymbol(name) {
			result = append(result, mod)
		}
	}
	return result
}

// ModulesImporting returns all modules that import a symbol with the given name.
// Excludes synthetic base modules.
func (m *Mib) ModulesImporting(name string) []*Module {
	var result []*Module
	for _, mod := range m.modules {
		if mod.base {
			continue
		}
		if mod.ImportsSymbol(name) {
			result = append(result, mod)
		}
	}
	return result
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

// Resolve looks up a node by name, qualified name, or numeric OID string.
//
// Accepted formats:
//   - "ifDescr"                     plain name
//   - "IF-MIB::ifDescr"            qualified (MODULE::name)
//   - "1.3.6.1.2.1.2.2.1.2"       numeric OID
//   - ".1.3.6.1.2.1.2.2.1.2"      numeric OID with leading dot
//
// Returns nil if no matching node is found.
func (m *Mib) Resolve(query string) *Node {
	// Qualified name: MODULE::name
	if modName, itemName, ok := strings.Cut(query, "::"); ok {
		mod := m.moduleByName[modName]
		if mod == nil {
			return nil
		}
		return mod.Node(itemName)
	}

	// Numeric OID
	q := query
	if q != "" && q[0] == '.' {
		q = q[1:]
	}
	if q != "" && q[0] >= '0' && q[0] <= '9' {
		oid, err := ParseOID(q)
		if err != nil {
			return nil
		}
		return m.NodeByOID(oid)
	}

	// Plain name
	return m.Node(query)
}

// ResolveOID converts a symbolic or numeric OID string to a numeric OID.
// Unlike Resolve, this handles instance-suffixed forms by resolving the
// name portion and appending trailing numeric arcs.
//
// Accepted formats (all formats accepted by Resolve, plus):
//   - "ifDescr.5"                   name with instance suffix
//   - "IF-MIB::ifDescr.5"          qualified name with instance suffix
//   - "IF-MIB::ifDescr.5.3"        qualified name with multi-arc suffix
//
// ResolveOID is the inverse of FormatOID:
//
//	m.FormatOID(OID{1,3,6,1,2,1,2,2,1,2,5}) => "IF-MIB::ifDescr.5"
//	m.ResolveOID("IF-MIB::ifDescr.5")        => OID{1,3,6,1,2,1,2,2,1,2,5}
//
// Returns an error if the input cannot be parsed or resolved.
func (m *Mib) ResolveOID(query string) (OID, error) {
	if query == "" {
		return nil, errors.New("empty query")
	}

	// Numeric OID: delegate to ParseOID directly (no tree lookup needed).
	q := query
	if q[0] == '.' {
		q = q[1:]
	}
	if q != "" && q[0] >= '0' && q[0] <= '9' {
		return ParseOID(q)
	}

	// Qualified name: MODULE::name[.suffix]
	if modName, rest, ok := strings.Cut(query, "::"); ok {
		name, suffix := splitNameSuffix(rest)
		mod := m.moduleByName[modName]
		if mod == nil {
			return nil, fmt.Errorf("module not found: %s", modName)
		}
		nd := mod.Node(name)
		if nd == nil {
			return nil, fmt.Errorf("node not found: %s::%s", modName, name)
		}
		return appendSuffix(nd.OID(), suffix)
	}

	// Plain name[.suffix]
	name, suffix := splitNameSuffix(query)
	nd := m.Node(name)
	if nd == nil {
		return nil, fmt.Errorf("node not found: %s", name)
	}
	return appendSuffix(nd.OID(), suffix)
}

// splitNameSuffix splits "ifDescr.5.3" into ("ifDescr", ".5.3").
// Returns (s, "") if no dot is present.
func splitNameSuffix(s string) (name, suffix string) {
	if i := strings.IndexByte(s, '.'); i >= 0 {
		return s[:i], s[i:]
	}
	return s, ""
}

// appendSuffix parses dotted numeric arcs from suffix and appends them to base.
// Returns base unchanged if suffix is empty.
func appendSuffix(base OID, suffix string) (OID, error) {
	if suffix == "" {
		return base, nil
	}
	extra, err := ParseOID(suffix)
	if err != nil {
		return nil, fmt.Errorf("invalid instance suffix %q: %w", suffix, err)
	}
	return append(base, extra...), nil
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
	return filterObjects(m.objects, func(obj *Object) bool {
		return obj.typ != nil && obj.typ.name == typeName
	})
}

// ObjectsByBaseType returns all objects whose effective base type matches.
func (m *Mib) ObjectsByBaseType(base BaseType) []*Object {
	return filterObjects(m.objects, func(obj *Object) bool {
		return obj.typ != nil && obj.typ.EffectiveBase() == base
	})
}

func filterObjects(objects []*Object, pred func(*Object) bool) []*Object {
	var result []*Object
	for _, obj := range objects {
		if pred(obj) {
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

// addType uses first-write-wins for the name index. Types are seeded in
// priority order (primitives, then base module types, then user types),
// so the first definition wins cross-module name collisions.
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
