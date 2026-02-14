package mib

import "slices"

// Notification is a NOTIFICATION-TYPE or TRAP-TYPE definition.
type Notification struct {
	name    string
	node    *Node
	module  *Module
	objects []*Object
	status  Status
	desc    string
	ref     string
}

// NewNotification returns a Notification initialized with the given name.
func NewNotification(name string) *Notification {
	return &Notification{name: name}
}

func (n *Notification) Name() string        { return n.name }
func (n *Notification) Node() *Node         { return n.node }
func (n *Notification) Module() *Module     { return n.module }
func (n *Notification) Status() Status      { return n.status }
func (n *Notification) Description() string { return n.desc }
func (n *Notification) Reference() string   { return n.ref }
func (n *Notification) Objects() []*Object  { return slices.Clone(n.objects) }

func (n *Notification) OID() OID {
	if n.node == nil {
		return nil
	}
	return n.node.OID()
}

// String returns a brief summary: "name (oid)".
func (n *Notification) String() string {
	if n == nil {
		return "<nil>"
	}
	return n.name + " (" + n.OID().String() + ")"
}

func (n *Notification) SetName(name string)       { n.name = name }
func (n *Notification) SetNode(nd *Node)          { n.node = nd }
func (n *Notification) SetModule(m *Module)       { n.module = m }
func (n *Notification) SetObjects(objs []*Object) { n.objects = objs }
func (n *Notification) AddObject(obj *Object)     { n.objects = append(n.objects, obj) }
func (n *Notification) SetStatus(s Status)        { n.status = s }
func (n *Notification) SetDescription(d string)   { n.desc = d }
func (n *Notification) SetReference(r string)     { n.ref = r }
