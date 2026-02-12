package mibimpl

import "github.com/golangsnmp/gomib/mib"

// Group implements mib.Group for OBJECT-GROUP and NOTIFICATION-GROUP
// definitions.
type Group struct {
	name                string
	node                *Node
	module              *Module
	members             []*Node
	status              mib.Status
	desc                string
	ref                 string
	isNotificationGroup bool
}

func (g *Group) Name() string {
	return g.name
}

func (g *Group) Node() mib.Node {
	if g.node == nil {
		return nil
	}
	return g.node
}

func (g *Group) Module() mib.Module {
	if g.module == nil {
		return nil
	}
	return g.module
}

func (g *Group) OID() mib.Oid {
	if g.node == nil {
		return nil
	}
	return g.node.OID()
}

func (g *Group) Status() mib.Status {
	return g.status
}

func (g *Group) Description() string {
	return g.desc
}

func (g *Group) Reference() string {
	return g.ref
}

func (g *Group) Members() []mib.Node {
	result := make([]mib.Node, len(g.members))
	for i, n := range g.members {
		result[i] = n
	}
	return result
}

func (g *Group) IsNotificationGroup() bool {
	return g.isNotificationGroup
}

// String returns a brief summary: "name (oid)".
func (g *Group) String() string {
	if g == nil {
		return "<nil>"
	}
	return g.name + " (" + g.OID().String() + ")"
}

func (g *Group) SetName(name string) {
	g.name = name
}

func (g *Group) SetNode(nd *Node) {
	g.node = nd
}

func (g *Group) SetModule(m *Module) {
	g.module = m
}

func (g *Group) SetMembers(members []*Node) {
	g.members = members
}

func (g *Group) AddMember(nd *Node) {
	g.members = append(g.members, nd)
}

func (g *Group) SetStatus(s mib.Status) {
	g.status = s
}

func (g *Group) SetDescription(d string) {
	g.desc = d
}

func (g *Group) SetReference(r string) {
	g.ref = r
}

func (g *Group) SetIsNotificationGroup(v bool) {
	g.isNotificationGroup = v
}

// InternalNode returns the concrete node.
func (g *Group) InternalNode() *Node {
	return g.node
}

// InternalMembers returns the concrete member nodes.
func (g *Group) InternalMembers() []*Node {
	return g.members
}
