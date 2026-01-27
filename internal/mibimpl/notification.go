package mibimpl

import "github.com/golangsnmp/gomib/mib"

// Notification is the concrete implementation of mib.Notification.
type Notification struct {
	name    string
	node    *Node
	module  *Module
	objects []*Object
	status  mib.Status
	desc    string
	ref     string
}

// Interface methods (mib.Notification)

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
	result := make([]mib.Object, len(n.objects))
	for i, obj := range n.objects {
		result[i] = obj
	}
	return result
}

// String returns a brief summary: "name (oid)".
func (n *Notification) String() string {
	if n == nil {
		return "<nil>"
	}
	return n.name + " (" + n.OID().String() + ")"
}

// Mutation methods (for resolver use)

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

// InternalNode returns the concrete node for resolver use.
func (n *Notification) InternalNode() *Node {
	return n.node
}

// InternalObjects returns the concrete objects for resolver use.
func (n *Notification) InternalObjects() []*Object {
	return n.objects
}
