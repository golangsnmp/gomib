package mibimpl

import "github.com/golangsnmp/gomib/mib"

// Module is the concrete implementation of mib.Module.
type Module struct {
	name         string
	language     mib.Language
	oid          mib.Oid
	organization string
	contactInfo  string
	description  string
	revisions    []mib.Revision

	// Internal collections
	objects       []*Object
	types         []*Type
	notifications []*Notification
	nodes         []*Node // for OBJECT IDENTIFIER assignments
}

// Interface methods (mib.Module)

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
	return m.revisions
}

func (m *Module) Objects() []mib.Object {
	result := make([]mib.Object, len(m.objects))
	for i, obj := range m.objects {
		result[i] = obj
	}
	return result
}

func (m *Module) Types() []mib.Type {
	result := make([]mib.Type, len(m.types))
	for i, t := range m.types {
		result[i] = t
	}
	return result
}

func (m *Module) Notifications() []mib.Notification {
	result := make([]mib.Notification, len(m.notifications))
	for i, n := range m.notifications {
		result[i] = n
	}
	return result
}

func (m *Module) Tables() []mib.Object {
	var result []mib.Object
	for _, obj := range m.objects {
		if obj.IsTable() {
			result = append(result, obj)
		}
	}
	return result
}

func (m *Module) Scalars() []mib.Object {
	var result []mib.Object
	for _, obj := range m.objects {
		if obj.IsScalar() {
			result = append(result, obj)
		}
	}
	return result
}

func (m *Module) Columns() []mib.Object {
	var result []mib.Object
	for _, obj := range m.objects {
		if obj.IsColumn() {
			result = append(result, obj)
		}
	}
	return result
}

func (m *Module) Rows() []mib.Object {
	var result []mib.Object
	for _, obj := range m.objects {
		if obj.IsRow() {
			result = append(result, obj)
		}
	}
	return result
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

// Mutation methods (for resolver use)

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

func (m *Module) AddNode(n *Node) {
	m.nodes = append(m.nodes, n)
}

// InternalObject returns the concrete object for resolver use.
func (m *Module) InternalObject(name string) *Object {
	for _, obj := range m.objects {
		if obj.name == name {
			return obj
		}
	}
	return nil
}

// InternalType returns the concrete type for resolver use.
func (m *Module) InternalType(name string) *Type {
	for _, t := range m.types {
		if t.name == name {
			return t
		}
	}
	return nil
}

// InternalObjects returns the concrete objects slice for resolver use.
func (m *Module) InternalObjects() []*Object {
	return m.objects
}

// InternalTypes returns the concrete types slice for resolver use.
func (m *Module) InternalTypes() []*Type {
	return m.types
}

// InternalNotifications returns the concrete notifications slice for resolver use.
func (m *Module) InternalNotifications() []*Notification {
	return m.notifications
}
