package mib

// entity holds the fields and methods shared by all resolved SMI entity
// types: Object, Notification, Group, Compliance, and Capability.
type entity struct {
	name   string
	node   *Node
	module *Module
	status Status
	desc   string
	ref    string
}

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

// OID returns the entity's position in the OID tree, or nil if unresolved.
func (e *entity) OID() OID {
	if e.node == nil {
		return nil
	}
	return e.node.OID()
}

// String returns a brief summary: "name (oid)".
func (e *entity) String() string {
	return e.name + " (" + e.OID().String() + ")"
}

func (e *entity) setNode(nd *Node)        { e.node = nd }
func (e *entity) setModule(m *Module)     { e.module = m }
func (e *entity) setStatus(s Status)      { e.status = s }
func (e *entity) setDescription(d string) { e.desc = d }
func (e *entity) setReference(r string)   { e.ref = r }
