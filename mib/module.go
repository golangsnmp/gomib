package mib

import "slices"

// Module represents a loaded and resolved MIB module.
type Module struct {
	name         string
	language     Language
	sourcePath   string
	oid          OID
	organization string
	contactInfo  string
	description  string
	lastUpdated  string
	revisions    []Revision
	imports      []Import

	objects       []*Object
	types         []*Type
	notifications []*Notification
	groups        []*Group
	compliances   []*Compliance
	capabilities  []*Capability
	nodes         []*Node

	// Name-indexed maps for O(1) lookups, populated by Add*() methods.
	objectsByName       map[string]*Object
	typesByName         map[string]*Type
	notificationsByName map[string]*Notification
	groupsByName        map[string]*Group
	compliancesByName   map[string]*Compliance
	capabilitiesByName  map[string]*Capability
	nodesByName         map[string]*Node
}

// newModule returns a Module initialized with the given name.
func newModule(name string) *Module {
	return &Module{
		name:                name,
		objectsByName:       make(map[string]*Object),
		typesByName:         make(map[string]*Type),
		notificationsByName: make(map[string]*Notification),
		groupsByName:        make(map[string]*Group),
		compliancesByName:   make(map[string]*Compliance),
		capabilitiesByName:  make(map[string]*Capability),
		nodesByName:         make(map[string]*Node),
	}
}

// Name returns the module name (e.g. "IF-MIB").
func (m *Module) Name() string { return m.name }

// Language returns the SMI language version of this module.
func (m *Module) Language() Language { return m.language }

// SourcePath returns the file path this module was loaded from, or "" for synthetic modules.
func (m *Module) SourcePath() string { return m.sourcePath }

// OID returns the MODULE-IDENTITY OID, or nil if not declared.
func (m *Module) OID() OID { return slices.Clone(m.oid) }

// Organization returns the ORGANIZATION clause text, or "".
func (m *Module) Organization() string { return m.organization }

// ContactInfo returns the CONTACT-INFO clause text, or "".
func (m *Module) ContactInfo() string { return m.contactInfo }

// Description returns the DESCRIPTION clause text.
func (m *Module) Description() string { return m.description }

// LastUpdated returns the LAST-UPDATED clause value, or "".
func (m *Module) LastUpdated() string { return m.lastUpdated }

// Revisions returns the REVISION clauses in declaration order.
func (m *Module) Revisions() []Revision { return slices.Clone(m.revisions) }

// Imports returns the IMPORTS declarations for this module.
func (m *Module) Imports() []Import { return slices.Clone(m.imports) }

// Objects returns all OBJECT-TYPE definitions in this module.
func (m *Module) Objects() []*Object { return slices.Clone(m.objects) }

// Types returns all type definitions in this module.
func (m *Module) Types() []*Type { return slices.Clone(m.types) }

// Notifications returns all NOTIFICATION-TYPE and TRAP-TYPE definitions in this module.
func (m *Module) Notifications() []*Notification { return slices.Clone(m.notifications) }

// Groups returns all OBJECT-GROUP and NOTIFICATION-GROUP definitions in this module.
func (m *Module) Groups() []*Group { return slices.Clone(m.groups) }

// Compliances returns all MODULE-COMPLIANCE definitions in this module.
func (m *Module) Compliances() []*Compliance { return slices.Clone(m.compliances) }

// Capabilities returns all AGENT-CAPABILITIES definitions in this module.
func (m *Module) Capabilities() []*Capability { return slices.Clone(m.capabilities) }

// Nodes returns all OID tree nodes registered by this module.
func (m *Module) Nodes() []*Node { return slices.Clone(m.nodes) }

// Tables returns the OBJECT-TYPE definitions classified as tables.
func (m *Module) Tables() []*Object { return objectsByKind(m.objects, KindTable) }

// Scalars returns the OBJECT-TYPE definitions classified as scalars.
func (m *Module) Scalars() []*Object { return objectsByKind(m.objects, KindScalar) }

// Columns returns the OBJECT-TYPE definitions classified as table columns.
func (m *Module) Columns() []*Object { return objectsByKind(m.objects, KindColumn) }

// Rows returns the OBJECT-TYPE definitions classified as table rows.
func (m *Module) Rows() []*Object { return objectsByKind(m.objects, KindRow) }

// Node returns the node with the given name in this module, or nil if not found.
func (m *Module) Node(name string) *Node {
	return m.nodesByName[name]
}

// Object returns the object with the given name in this module, or nil if not found.
func (m *Module) Object(name string) *Object {
	return m.objectsByName[name]
}

// Type returns the type with the given name in this module, or nil if not found.
func (m *Module) Type(name string) *Type {
	return m.typesByName[name]
}

// Notification returns the notification with the given name in this module, or nil if not found.
func (m *Module) Notification(name string) *Notification {
	return m.notificationsByName[name]
}

// Group returns the group with the given name in this module, or nil if not found.
func (m *Module) Group(name string) *Group {
	return m.groupsByName[name]
}

// Compliance returns the compliance with the given name in this module, or nil if not found.
func (m *Module) Compliance(name string) *Compliance {
	return m.compliancesByName[name]
}

// Capability returns the capability with the given name in this module, or nil if not found.
func (m *Module) Capability(name string) *Capability {
	return m.capabilitiesByName[name]
}

func (m *Module) setSourcePath(path string)    { m.sourcePath = path }
func (m *Module) setLanguage(l Language)       { m.language = l }
func (m *Module) setOID(oid OID)               { m.oid = oid }
func (m *Module) setOrganization(org string)   { m.organization = org }
func (m *Module) setContactInfo(info string)   { m.contactInfo = info }
func (m *Module) setDescription(desc string)   { m.description = desc }
func (m *Module) setLastUpdated(s string)      { m.lastUpdated = s }
func (m *Module) setRevisions(revs []Revision) { m.revisions = revs }

func (m *Module) addObject(obj *Object) {
	m.objects = append(m.objects, obj)
	m.objectsByName[obj.name] = obj
}

func (m *Module) addType(t *Type) {
	m.types = append(m.types, t)
	if m.typesByName[t.name] == nil {
		m.typesByName[t.name] = t
	}
}

func (m *Module) addNotification(n *Notification) {
	m.notifications = append(m.notifications, n)
	m.notificationsByName[n.name] = n
}

func (m *Module) addGroup(g *Group) {
	m.groups = append(m.groups, g)
	m.groupsByName[g.name] = g
}

func (m *Module) addCompliance(c *Compliance) {
	m.compliances = append(m.compliances, c)
	m.compliancesByName[c.name] = c
}

func (m *Module) addCapability(c *Capability) {
	m.capabilities = append(m.capabilities, c)
	m.capabilitiesByName[c.name] = c
}

func (m *Module) addNode(n *Node) {
	m.nodes = append(m.nodes, n)
	m.nodesByName[n.name] = n
}
