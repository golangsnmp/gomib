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

// NewGroup returns a Group initialized with the given name.
func NewGroup(name string) *Group {
	return &Group{name: name}
}

func (g *Group) Name() string        { return g.name }
func (g *Group) Node() *Node         { return g.node }
func (g *Group) Module() *Module     { return g.module }
func (g *Group) Status() Status      { return g.status }
func (g *Group) Description() string { return g.desc }
func (g *Group) Reference() string   { return g.ref }
func (g *Group) Members() []*Node    { return slices.Clone(g.members) }

func (g *Group) IsNotificationGroup() bool { return g.isNotificationGroup }

func (g *Group) OID() OID {
	if g.node == nil {
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

func (g *Group) SetName(name string)           { g.name = name }
func (g *Group) SetNode(nd *Node)              { g.node = nd }
func (g *Group) SetModule(m *Module)           { g.module = m }
func (g *Group) SetMembers(members []*Node)    { g.members = members }
func (g *Group) AddMember(nd *Node)            { g.members = append(g.members, nd) }
func (g *Group) SetStatus(s Status)            { g.status = s }
func (g *Group) SetDescription(d string)       { g.desc = d }
func (g *Group) SetReference(r string)         { g.ref = r }
func (g *Group) SetIsNotificationGroup(v bool) { g.isNotificationGroup = v }
