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
}

func (m *Module) Name() string {
	return m.name
}

func (m *Module) Language() mib.Language {
	return m.language
}

func (m *Module) OID() mib.Oid {
	return m.oid
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
	for _, n := range m.nodes {
		if n.name == name {
			return n
		}
	}
	return nil
}

func (m *Module) Object(name string) mib.Object {
	for _, obj := range m.objects {
		if obj.name == name {
			return obj
		}
	}
	return nil
}

func (m *Module) Type(name string) mib.Type {
	for _, t := range m.types {
		if t.name == name {
			return t
		}
	}
	return nil
}

func (m *Module) Notification(name string) mib.Notification {
	for _, n := range m.notifications {
		if n.name == name {
			return n
		}
	}
	return nil
}

func (m *Module) Groups() []mib.Group {
	return mapSlice(m.groups, func(v *Group) mib.Group { return v })
}

func (m *Module) Group(name string) mib.Group {
	for _, g := range m.groups {
		if g.name == name {
			return g
		}
	}
	return nil
}

func (m *Module) Compliances() []mib.Compliance {
	return mapSlice(m.compliances, func(v *Compliance) mib.Compliance { return v })
}

func (m *Module) ComplianceByName(name string) mib.Compliance {
	for _, c := range m.compliances {
		if c.name == name {
			return c
		}
	}
	return nil
}

func (m *Module) Capabilities() []mib.Capabilities {
	return mapSlice(m.capabilities, func(v *Capabilities) mib.Capabilities { return v })
}

func (m *Module) CapabilitiesByName(name string) mib.Capabilities {
	for _, c := range m.capabilities {
		if c.name == name {
			return c
		}
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
}

func (m *Module) AddType(t *Type) {
	m.types = append(m.types, t)
}

func (m *Module) AddNotification(n *Notification) {
	m.notifications = append(m.notifications, n)
}

func (m *Module) AddGroup(g *Group) {
	m.groups = append(m.groups, g)
}

func (m *Module) AddCompliance(c *Compliance) {
	m.compliances = append(m.compliances, c)
}

func (m *Module) AddCapabilities(c *Capabilities) {
	m.capabilities = append(m.capabilities, c)
}

func (m *Module) AddNode(n *Node) {
	m.nodes = append(m.nodes, n)
}

// InternalObject looks up a concrete object by name.
func (m *Module) InternalObject(name string) *Object {
	for _, obj := range m.objects {
		if obj.name == name {
			return obj
		}
	}
	return nil
}

// InternalType looks up a concrete type by name.
func (m *Module) InternalType(name string) *Type {
	for _, t := range m.types {
		if t.name == name {
			return t
		}
	}
	return nil
}

// InternalObjects returns the concrete objects slice.
func (m *Module) InternalObjects() []*Object {
	return m.objects
}

// InternalTypes returns the concrete types slice.
func (m *Module) InternalTypes() []*Type {
	return m.types
}

// InternalNotifications returns the concrete notifications slice.
func (m *Module) InternalNotifications() []*Notification {
	return m.notifications
}

// InternalGroups returns the concrete groups slice.
func (m *Module) InternalGroups() []*Group {
	return m.groups
}

// InternalCompliances returns the concrete compliances slice.
func (m *Module) InternalCompliances() []*Compliance {
	return m.compliances
}

// InternalCapabilities returns the concrete capabilities slice.
func (m *Module) InternalCapabilities() []*Capabilities {
	return m.capabilities
}
