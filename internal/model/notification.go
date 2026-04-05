package model

import "slices"

// Notification is a NOTIFICATION-TYPE or TRAP-TYPE definition.
type Notification struct {
	entity
	objects  []*Object
	trapInfo *TrapInfo
}

// Objects returns the OBJECTS clause entries (the varbinds sent with this notification).
func (n *Notification) Objects() []*Object { return slices.Clone(n.objects) }

// TrapInfo returns SMIv1 TRAP-TYPE fields, or nil for SMIv2 NOTIFICATION-TYPE definitions.
func (n *Notification) TrapInfo() *TrapInfo {
	if n.trapInfo == nil {
		return nil
	}
	clone := *n.trapInfo
	return &clone
}

func (n *Notification) addObject(obj *Object)   { n.objects = append(n.objects, obj) }
func (n *Notification) setTrapInfo(t *TrapInfo) { n.trapInfo = t }
