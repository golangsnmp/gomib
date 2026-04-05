package model

import "slices"

// OidRef is a named symbolic reference from an OID assignment with its
// source span, e.g. "system" from ::= { system 1 }.
type OidRef struct {
	Name string
	Span Span
}

// entity holds the fields and methods shared by all resolved SMI entity
// types: Object, Notification, Group, Compliance, and Capability.
type entity struct {
	name       string
	span       Span
	node       *Node
	module     *Module
	status     Status
	desc       string
	ref        string
	statusSpan Span
	descSpan   Span
	refSpan    Span
	oidRefs    []OidRef
}

// Span returns the source byte range of this entity's definition.
// Returns a zero-value Span for synthetic (base module) definitions.
func (e *entity) Span() Span { return e.span }

// Name returns the entity's descriptor name.
func (e *entity) Name() string { return e.name }

// Node returns the OID tree node this entity is attached to.
func (e *entity) Node() *Node { return e.node }

// Module returns the module that defines this entity.
func (e *entity) Module() *Module { return e.module }

// Status returns the STATUS clause value.
func (e *entity) Status() Status { return e.status }

// Description returns the DESCRIPTION clause text.
func (e *entity) Description() string { return e.desc }

// Reference returns the REFERENCE clause text, or "".
func (e *entity) Reference() string { return e.ref }

// StatusSpan returns the source byte range of this entity's STATUS clause.
func (e *entity) StatusSpan() Span { return e.statusSpan }

// DescriptionSpan returns the source byte range of this entity's DESCRIPTION clause.
func (e *entity) DescriptionSpan() Span { return e.descSpan }

// ReferenceSpan returns the source byte range of this entity's REFERENCE clause.
func (e *entity) ReferenceSpan() Span { return e.refSpan }

// OidRefs returns the symbolic OID references from this entity's OID assignment.
func (e *entity) OidRefs() []OidRef { return slices.Clone(e.oidRefs) }

// OID returns the entity's position in the OID tree, or nil if unresolved.
func (e *entity) OID() OID {
	if e.node == nil {
		return nil
	}
	return e.node.OID()
}

// String returns a brief summary: "name (oid)", or just "name" if unresolved.
func (e *entity) String() string {
	if e.node == nil {
		return e.name
	}
	return e.name + " (" + e.OID().String() + ")"
}

func (e *entity) setSpan(s Span)          { e.span = s }
func (e *entity) setNode(nd *Node)        { e.node = nd }
func (e *entity) setModule(m *Module)     { e.module = m }
func (e *entity) setStatus(s Status)      { e.status = s }
func (e *entity) setDescription(d string) { e.desc = d }
func (e *entity) setReference(r string)   { e.ref = r }

func (e *entity) setStatusSpan(s Span)      { e.statusSpan = s }
func (e *entity) setDescriptionSpan(s Span) { e.descSpan = s }
func (e *entity) setReferenceSpan(s Span)   { e.refSpan = s }
func (e *entity) setOidRefs(refs []OidRef)  { e.oidRefs = refs }
