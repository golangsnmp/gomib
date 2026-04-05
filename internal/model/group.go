package model

import "slices"

// Group is an OBJECT-GROUP or NOTIFICATION-GROUP definition.
type Group struct {
	entity
	members             []*Node
	isNotificationGroup bool
}

// Members returns the OID tree nodes listed in the OBJECTS or NOTIFICATIONS clause.
func (g *Group) Members() []*Node { return slices.Clone(g.members) }

// IsNotificationGroup reports whether this is a NOTIFICATION-GROUP (vs OBJECT-GROUP).
func (g *Group) IsNotificationGroup() bool { return g.isNotificationGroup }

func (g *Group) addMember(nd *Node)            { g.members = append(g.members, nd) }
func (g *Group) setIsNotificationGroup(v bool) { g.isNotificationGroup = v }
