package mib

import "slices"

// Capability is an AGENT-CAPABILITIES definition.
type Capability struct {
	name           string
	node           *Node
	module         *Module
	status         Status
	desc           string
	ref            string
	productRelease string
	supports       []CapabilitiesModule
}

// newCapability returns a Capability initialized with the given name.
func newCapability(name string) *Capability {
	return &Capability{name: name}
}

// Name returns the capability statement's descriptor.
func (c *Capability) Name() string { return c.name }

// Node returns the OID tree node this capability statement is attached to.
func (c *Capability) Node() *Node { return c.node }

// Module returns the module that defines this capability statement.
func (c *Capability) Module() *Module { return c.module }

// Status returns the STATUS clause value.
func (c *Capability) Status() Status { return c.status }

// Description returns the DESCRIPTION clause text.
func (c *Capability) Description() string { return c.desc }

// Reference returns the REFERENCE clause text, or "".
func (c *Capability) Reference() string { return c.ref }

// ProductRelease returns the PRODUCT-RELEASE clause text.
func (c *Capability) ProductRelease() string { return c.productRelease }

// Supports returns the SUPPORTS clauses listing the modules this agent implements.
func (c *Capability) Supports() []CapabilitiesModule { return slices.Clone(c.supports) }

// OID returns the capability statement's position in the OID tree, or nil if unresolved.
func (c *Capability) OID() OID {
	if c == nil || c.node == nil {
		return nil
	}
	return c.node.OID()
}

// String returns a brief summary: "name (oid)".
func (c *Capability) String() string {
	if c == nil {
		return "<nil>"
	}
	return c.name + " (" + c.OID().String() + ")"
}

func (c *Capability) setNode(nd *Node)                          { c.node = nd }
func (c *Capability) setModule(m *Module)                       { c.module = m }
func (c *Capability) setStatus(s Status)                        { c.status = s }
func (c *Capability) setDescription(d string)                   { c.desc = d }
func (c *Capability) setReference(r string)                     { c.ref = r }
func (c *Capability) setProductRelease(r string)                { c.productRelease = r }
func (c *Capability) setSupports(supports []CapabilitiesModule) { c.supports = supports }
