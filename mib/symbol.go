package mib

// Symbol wraps any resolved MIB definition. Exactly one of the internal
// fields is non-nil. Common accessors (Name, Span, Module) work on all
// symbol kinds without requiring callers to type-switch.
type Symbol struct {
	object       *Object
	notification *Notification
	group        *Group
	compliance   *Compliance
	capability   *Capability
	typ          *Type
	node         *Node // plain node (no entity attachment)
}

func symbolFromObject(o *Object) Symbol             { return Symbol{object: o} }
func symbolFromNotification(n *Notification) Symbol { return Symbol{notification: n} }
func symbolFromGroup(g *Group) Symbol               { return Symbol{group: g} }
func symbolFromCompliance(c *Compliance) Symbol     { return Symbol{compliance: c} }
func symbolFromCapability(c *Capability) Symbol     { return Symbol{capability: c} }
func symbolFromType(t *Type) Symbol                 { return Symbol{typ: t} }
func symbolFromNode(n *Node) Symbol                 { return Symbol{node: n} }

// IsZero reports whether this is an empty Symbol (nothing found).
func (s Symbol) IsZero() bool {
	return s.object == nil && s.notification == nil && s.group == nil &&
		s.compliance == nil && s.capability == nil && s.typ == nil && s.node == nil
}

// Name returns the definition's name, or "" for a zero Symbol.
func (s Symbol) Name() string {
	switch {
	case s.object != nil:
		return s.object.Name()
	case s.notification != nil:
		return s.notification.Name()
	case s.group != nil:
		return s.group.Name()
	case s.compliance != nil:
		return s.compliance.Name()
	case s.capability != nil:
		return s.capability.Name()
	case s.typ != nil:
		return s.typ.Name()
	case s.node != nil:
		return s.node.Name()
	default:
		return ""
	}
}

// Span returns the source byte range of this definition.
func (s Symbol) Span() Span {
	switch {
	case s.object != nil:
		return s.object.Span()
	case s.notification != nil:
		return s.notification.Span()
	case s.group != nil:
		return s.group.Span()
	case s.compliance != nil:
		return s.compliance.Span()
	case s.capability != nil:
		return s.capability.Span()
	case s.typ != nil:
		return s.typ.Span()
	case s.node != nil:
		return s.node.Span()
	default:
		return Span{}
	}
}

// Module returns the module that defines this symbol, or nil.
func (s Symbol) Module() *Module {
	switch {
	case s.object != nil:
		return s.object.Module()
	case s.notification != nil:
		return s.notification.Module()
	case s.group != nil:
		return s.group.Module()
	case s.compliance != nil:
		return s.compliance.Module()
	case s.capability != nil:
		return s.capability.Module()
	case s.typ != nil:
		return s.typ.Module()
	case s.node != nil:
		return s.node.Module()
	default:
		return nil
	}
}

// Node returns the OID tree node for this symbol, or nil for Type symbols.
func (s Symbol) Node() *Node {
	switch {
	case s.object != nil:
		return s.object.Node()
	case s.notification != nil:
		return s.notification.Node()
	case s.group != nil:
		return s.group.Node()
	case s.compliance != nil:
		return s.compliance.Node()
	case s.capability != nil:
		return s.capability.Node()
	case s.node != nil:
		return s.node
	default:
		return nil
	}
}

// OID returns the numeric OID for this symbol, or nil for Type symbols
// and unresolved nodes.
func (s Symbol) OID() OID {
	if nd := s.Node(); nd != nil {
		return nd.OID()
	}
	return nil
}

// Object returns the Object if this symbol is an OBJECT-TYPE, or nil.
func (s Symbol) Object() *Object { return s.object }

// Notification returns the Notification if this symbol is a NOTIFICATION-TYPE
// or TRAP-TYPE, or nil.
func (s Symbol) Notification() *Notification { return s.notification }

// Group returns the Group if this symbol is an OBJECT-GROUP or
// NOTIFICATION-GROUP, or nil.
func (s Symbol) Group() *Group { return s.group }

// Compliance returns the Compliance if this symbol is a MODULE-COMPLIANCE, or nil.
func (s Symbol) Compliance() *Compliance { return s.compliance }

// Capability returns the Capability if this symbol is an AGENT-CAPABILITIES, or nil.
func (s Symbol) Capability() *Capability { return s.capability }

// Type returns the Type if this symbol is a type definition, or nil.
func (s Symbol) Type() *Type { return s.typ }

// Status returns the status of this symbol's definition (current,
// deprecated, obsolete, etc.), or 0 for plain nodes and zero symbols.
func (s Symbol) Status() Status {
	switch {
	case s.object != nil:
		return s.object.Status()
	case s.notification != nil:
		return s.notification.Status()
	case s.group != nil:
		return s.group.Status()
	case s.compliance != nil:
		return s.compliance.Status()
	case s.capability != nil:
		return s.capability.Status()
	case s.typ != nil:
		return s.typ.Status()
	default:
		return 0
	}
}
