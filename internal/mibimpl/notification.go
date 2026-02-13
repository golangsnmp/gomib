package mibimpl

import "github.com/golangsnmp/gomib/mib"

// Notification implements mib.Notification for NOTIFICATION-TYPE and
// TRAP-TYPE definitions.
type Notification struct {
	name    string
	node    *Node
	module  *Module
	objects []*Object
	status  mib.Status
	desc    string
	ref     string
}

func (n *Notification) Name() string {
	return n.name
}

func (n *Notification) Node() mib.Node {
	if n.node == nil {
		return nil
	}
	return n.node
}

func (n *Notification) Module() mib.Module {
	if n.module == nil {
		return nil
	}
	return n.module
}

func (n *Notification) OID() mib.Oid {
	if n.node == nil {
		return nil
	}
	return n.node.OID()
}

func (n *Notification) Status() mib.Status {
	return n.status
}

func (n *Notification) Description() string {
	return n.desc
}

func (n *Notification) Reference() string {
	return n.ref
}

func (n *Notification) Objects() []mib.Object {
	return mapSlice(n.objects, func(v *Object) mib.Object { return v })
}

// String returns a brief summary: "name (oid)".
func (n *Notification) String() string {
	if n == nil {
		return "<nil>"
	}
	return n.name + " (" + n.OID().String() + ")"
}

func (n *Notification) SetName(name string) {
	n.name = name
}

func (n *Notification) SetNode(nd *Node) {
	n.node = nd
}

func (n *Notification) SetModule(m *Module) {
	n.module = m
}

func (n *Notification) SetObjects(objs []*Object) {
	n.objects = objs
}

func (n *Notification) AddObject(obj *Object) {
	n.objects = append(n.objects, obj)
}

func (n *Notification) SetStatus(s mib.Status) {
	n.status = s
}

func (n *Notification) SetDescription(d string) {
	n.desc = d
}

func (n *Notification) SetReference(r string) {
	n.ref = r
}

// InternalNode returns the concrete node.
func (n *Notification) InternalNode() *Node {
	return n.node
}

// InternalObjects returns the concrete OBJECTS list.
func (n *Notification) InternalObjects() []*Object {
	return n.objects
}
