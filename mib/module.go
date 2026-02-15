package mib

import "slices"

// Module represents a loaded and resolved MIB module.
type Module struct {
	name         string
	language     Language
	oid          OID
	organization string
	contactInfo  string
	description  string
	revisions    []Revision

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

func (m *Module) Name() string          { return m.name }
func (m *Module) Language() Language    { return m.language }
func (m *Module) OID() OID              { return slices.Clone(m.oid) }
func (m *Module) Organization() string  { return m.organization }
func (m *Module) ContactInfo() string   { return m.contactInfo }
func (m *Module) Description() string   { return m.description }
func (m *Module) Revisions() []Revision { return slices.Clone(m.revisions) }

func (m *Module) Objects() []*Object             { return slices.Clone(m.objects) }
func (m *Module) Types() []*Type                 { return slices.Clone(m.types) }
func (m *Module) Notifications() []*Notification { return slices.Clone(m.notifications) }
func (m *Module) Groups() []*Group               { return slices.Clone(m.groups) }
func (m *Module) Compliances() []*Compliance     { return slices.Clone(m.compliances) }
func (m *Module) Capabilities() []*Capability    { return slices.Clone(m.capabilities) }

func (m *Module) Tables() []*Object  { return objectsByKind(m.objects, KindTable) }
func (m *Module) Scalars() []*Object { return objectsByKind(m.objects, KindScalar) }
func (m *Module) Columns() []*Object { return objectsByKind(m.objects, KindColumn) }
func (m *Module) Rows() []*Object    { return objectsByKind(m.objects, KindRow) }

func (m *Module) Node(name string) *Node {
	return m.nodesByName[name]
}

func (m *Module) Object(name string) *Object {
	return m.objectsByName[name]
}

func (m *Module) Type(name string) *Type {
	return m.typesByName[name]
}

func (m *Module) Notification(name string) *Notification {
	return m.notificationsByName[name]
}

func (m *Module) Group(name string) *Group {
	return m.groupsByName[name]
}

func (m *Module) Compliance(name string) *Compliance {
	return m.compliancesByName[name]
}

func (m *Module) Capability(name string) *Capability {
	return m.capabilitiesByName[name]
}

func (m *Module) setLanguage(l Language)       { m.language = l }
func (m *Module) setOID(oid OID)               { m.oid = oid }
func (m *Module) setOrganization(org string)   { m.organization = org }
func (m *Module) setContactInfo(info string)   { m.contactInfo = info }
func (m *Module) setDescription(desc string)   { m.description = desc }
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
