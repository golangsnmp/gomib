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

// NewModule returns a Module initialized with the given name.
func NewModule(name string) *Module {
	return &Module{name: name}
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

func (m *Module) SetLanguage(l Language)       { m.language = l }
func (m *Module) SetOID(oid OID)               { m.oid = oid }
func (m *Module) SetOrganization(org string)   { m.organization = org }
func (m *Module) SetContactInfo(info string)   { m.contactInfo = info }
func (m *Module) SetDescription(desc string)   { m.description = desc }
func (m *Module) SetRevisions(revs []Revision) { m.revisions = revs }

func (m *Module) AddObject(obj *Object) {
	m.objects = append(m.objects, obj)
	if m.objectsByName == nil {
		m.objectsByName = make(map[string]*Object)
	}
	m.objectsByName[obj.name] = obj
}

func (m *Module) AddType(t *Type) {
	m.types = append(m.types, t)
	if m.typesByName == nil {
		m.typesByName = make(map[string]*Type)
	}
	m.typesByName[t.name] = t
}

func (m *Module) AddNotification(n *Notification) {
	m.notifications = append(m.notifications, n)
	if m.notificationsByName == nil {
		m.notificationsByName = make(map[string]*Notification)
	}
	m.notificationsByName[n.name] = n
}

func (m *Module) AddGroup(g *Group) {
	m.groups = append(m.groups, g)
	if m.groupsByName == nil {
		m.groupsByName = make(map[string]*Group)
	}
	m.groupsByName[g.name] = g
}

func (m *Module) AddCompliance(c *Compliance) {
	m.compliances = append(m.compliances, c)
	if m.compliancesByName == nil {
		m.compliancesByName = make(map[string]*Compliance)
	}
	m.compliancesByName[c.name] = c
}

func (m *Module) AddCapability(c *Capability) {
	m.capabilities = append(m.capabilities, c)
	if m.capabilitiesByName == nil {
		m.capabilitiesByName = make(map[string]*Capability)
	}
	m.capabilitiesByName[c.name] = c
}

func (m *Module) AddNode(n *Node) {
	m.nodes = append(m.nodes, n)
	if m.nodesByName == nil {
		m.nodesByName = make(map[string]*Node)
	}
	m.nodesByName[n.name] = n
}
