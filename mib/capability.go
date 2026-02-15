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

func (c *Capability) Name() string                   { return c.name }
func (c *Capability) Node() *Node                    { return c.node }
func (c *Capability) Module() *Module                { return c.module }
func (c *Capability) Status() Status                 { return c.status }
func (c *Capability) Description() string            { return c.desc }
func (c *Capability) Reference() string              { return c.ref }
func (c *Capability) ProductRelease() string         { return c.productRelease }
func (c *Capability) Supports() []CapabilitiesModule { return slices.Clone(c.supports) }

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
