package mib

import "slices"

// Group is an OBJECT-GROUP or NOTIFICATION-GROUP definition.
type Group struct {
	name                string
	node                *Node
	module              *Module
	members             []*Node
	status              Status
	desc                string
	ref                 string
	isNotificationGroup bool
}

// newGroup returns a Group initialized with the given name.
func newGroup(name string) *Group {
	return &Group{name: name}
}

// Name returns the group's descriptor.
func (g *Group) Name() string { return g.name }

// Node returns the OID tree node this group is attached to.
func (g *Group) Node() *Node { return g.node }

// Module returns the module that defines this group.
func (g *Group) Module() *Module { return g.module }

// Status returns the STATUS clause value.
func (g *Group) Status() Status { return g.status }

// Description returns the DESCRIPTION clause text.
func (g *Group) Description() string { return g.desc }

// Reference returns the REFERENCE clause text, or "".
func (g *Group) Reference() string { return g.ref }

// Members returns the OID tree nodes listed in the OBJECTS or NOTIFICATIONS clause.
func (g *Group) Members() []*Node { return slices.Clone(g.members) }

// IsNotificationGroup reports whether this is a NOTIFICATION-GROUP (vs OBJECT-GROUP).
func (g *Group) IsNotificationGroup() bool { return g.isNotificationGroup }

// OID returns the group's position in the OID tree, or nil if unresolved.
func (g *Group) OID() OID {
	if g == nil || g.node == nil {
		return nil
	}
	return g.node.OID()
}

// String returns a brief summary: "name (oid)".
func (g *Group) String() string {
	if g == nil {
		return "<nil>"
	}
	return g.name + " (" + g.OID().String() + ")"
}

func (g *Group) setNode(nd *Node)              { g.node = nd }
func (g *Group) setModule(m *Module)           { g.module = m }
func (g *Group) addMember(nd *Node)            { g.members = append(g.members, nd) }
func (g *Group) setStatus(s Status)            { g.status = s }
func (g *Group) setDescription(d string)       { g.desc = d }
func (g *Group) setReference(r string)         { g.ref = r }
func (g *Group) setIsNotificationGroup(v bool) { g.isNotificationGroup = v }
