package mibimpl

import (
	"slices"

	"github.com/golangsnmp/gomib/mib"
)

// Module implements mib.Module, holding all definitions from a single
// MIB module.
type Module struct {
	name         string
	language     mib.Language
	oid          mib.Oid
	organization string
	contactInfo  string
	description  string
	revisions    []mib.Revision

	objects       []*Object
	types         []*Type
	notifications []*Notification
	groups        []*Group
	compliances   []*Compliance
	capabilities  []*Capabilities
	nodes         []*Node

	// Name-indexed maps for O(1) lookups, populated by Add*() methods.
	objectsByName       map[string]*Object
	typesByName         map[string]*Type
	notificationsByName map[string]*Notification
	groupsByName        map[string]*Group
	compliancesByName   map[string]*Compliance
	capabilitiesByName  map[string]*Capabilities
	nodesByName         map[string]*Node
}

func (m *Module) Name() string {
	return m.name
}

func (m *Module) Language() mib.Language {
	return m.language
}

func (m *Module) OID() mib.Oid {
	return slices.Clone(m.oid)
}

func (m *Module) Organization() string {
	return m.organization
}

func (m *Module) ContactInfo() string {
	return m.contactInfo
}

func (m *Module) Description() string {
	return m.description
}

func (m *Module) Revisions() []mib.Revision {
	return slices.Clone(m.revisions)
}

func (m *Module) Objects() []mib.Object {
	return mapSlice(m.objects, func(v *Object) mib.Object { return v })
}

func (m *Module) Types() []mib.Type {
	return mapSlice(m.types, func(v *Type) mib.Type { return v })
}

func (m *Module) Notifications() []mib.Notification {
	return mapSlice(m.notifications, func(v *Notification) mib.Notification { return v })
}

func (m *Module) Tables() []mib.Object {
	return objectsByKind(m.objects, mib.KindTable)
}

func (m *Module) Scalars() []mib.Object {
	return objectsByKind(m.objects, mib.KindScalar)
}

func (m *Module) Columns() []mib.Object {
	return objectsByKind(m.objects, mib.KindColumn)
}

func (m *Module) Rows() []mib.Object {
	return objectsByKind(m.objects, mib.KindRow)
}

func (m *Module) Node(name string) mib.Node {
	if n := m.nodesByName[name]; n != nil {
		return n
	}
	return nil
}

func (m *Module) Object(name string) mib.Object {
	if obj := m.objectsByName[name]; obj != nil {
		return obj
	}
	return nil
}

func (m *Module) Type(name string) mib.Type {
	if t := m.typesByName[name]; t != nil {
		return t
	}
	return nil
}

func (m *Module) Notification(name string) mib.Notification {
	if n := m.notificationsByName[name]; n != nil {
		return n
	}
	return nil
}

func (m *Module) Groups() []mib.Group {
	return mapSlice(m.groups, func(v *Group) mib.Group { return v })
}

func (m *Module) Group(name string) mib.Group {
	if g := m.groupsByName[name]; g != nil {
		return g
	}
	return nil
}

func (m *Module) Compliances() []mib.Compliance {
	return mapSlice(m.compliances, func(v *Compliance) mib.Compliance { return v })
}

func (m *Module) ComplianceByName(name string) mib.Compliance {
	if c := m.compliancesByName[name]; c != nil {
		return c
	}
	return nil
}

func (m *Module) Capabilities() []mib.Capabilities {
	return mapSlice(m.capabilities, func(v *Capabilities) mib.Capabilities { return v })
}

func (m *Module) CapabilitiesByName(name string) mib.Capabilities {
	if c := m.capabilitiesByName[name]; c != nil {
		return c
	}
	return nil
}

func (m *Module) SetLanguage(l mib.Language) {
	m.language = l
}

func (m *Module) SetOID(oid mib.Oid) {
	m.oid = oid
}

func (m *Module) SetOrganization(org string) {
	m.organization = org
}

func (m *Module) SetContactInfo(info string) {
	m.contactInfo = info
}

func (m *Module) SetDescription(desc string) {
	m.description = desc
}

func (m *Module) SetRevisions(revs []mib.Revision) {
	m.revisions = revs
}

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

func (m *Module) AddCapabilities(c *Capabilities) {
	m.capabilities = append(m.capabilities, c)
	if m.capabilitiesByName == nil {
		m.capabilitiesByName = make(map[string]*Capabilities)
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

// InternalObject looks up a concrete object by name.
func (m *Module) InternalObject(name string) *Object {
	return m.objectsByName[name]
}

// InternalType looks up a concrete type by name.
func (m *Module) InternalType(name string) *Type {
	return m.typesByName[name]
}
