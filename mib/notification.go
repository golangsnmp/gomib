package mib

import "slices"

// Notification is a NOTIFICATION-TYPE or TRAP-TYPE definition.
type Notification struct {
	name     string
	node     *Node
	module   *Module
	objects  []*Object
	status   Status
	desc     string
	ref      string
	trapInfo *TrapInfo
}

// newNotification returns a Notification initialized with the given name.
func newNotification(name string) *Notification {
	return &Notification{name: name}
}

// Name returns the notification's descriptor.
func (n *Notification) Name() string { return n.name }

// Node returns the OID tree node this notification is attached to.
func (n *Notification) Node() *Node { return n.node }

// Module returns the module that defines this notification.
func (n *Notification) Module() *Module { return n.module }

// Status returns the STATUS clause value.
func (n *Notification) Status() Status { return n.status }

// Description returns the DESCRIPTION clause text.
func (n *Notification) Description() string { return n.desc }

// Reference returns the REFERENCE clause text, or "".
func (n *Notification) Reference() string { return n.ref }

// Objects returns the OBJECTS clause entries (the varbinds sent with this notification).
func (n *Notification) Objects() []*Object { return slices.Clone(n.objects) }

// OID returns the notification's position in the OID tree, or nil if unresolved.
func (n *Notification) OID() OID {
	if n == nil || n.node == nil {
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

// TrapInfo returns SMIv1 TRAP-TYPE fields, or nil for SMIv2 NOTIFICATION-TYPE definitions.
func (n *Notification) TrapInfo() *TrapInfo { return n.trapInfo }

func (n *Notification) setNode(nd *Node)        { n.node = nd }
func (n *Notification) setModule(m *Module)     { n.module = m }
func (n *Notification) addObject(obj *Object)   { n.objects = append(n.objects, obj) }
func (n *Notification) setStatus(s Status)      { n.status = s }
func (n *Notification) setDescription(d string) { n.desc = d }
func (n *Notification) setReference(r string)   { n.ref = r }
func (n *Notification) setTrapInfo(t *TrapInfo) { n.trapInfo = t }
