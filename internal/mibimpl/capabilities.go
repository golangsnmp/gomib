package mibimpl

import (
	"slices"

	"github.com/golangsnmp/gomib/mib"
)

// Capability implements mib.Capability for AGENT-CAPABILITIES
// definitions.
type Capability struct {
	name           string
	node           *Node
	module         *Module
	status         mib.Status
	desc           string
	ref            string
	productRelease string
	supports       []mib.CapabilitiesModule
}

func (c *Capability) Name() string {
	return c.name
}

func (c *Capability) Node() mib.Node {
	if c.node == nil {
		return nil
	}
	return c.node
}

func (c *Capability) Module() mib.Module {
	if c.module == nil {
		return nil
	}
	return c.module
}

func (c *Capability) OID() mib.OID {
	if c.node == nil {
		return nil
	}
	return c.node.OID()
}

func (c *Capability) Status() mib.Status {
	return c.status
}

func (c *Capability) Description() string {
	return c.desc
}

func (c *Capability) Reference() string {
	return c.ref
}

func (c *Capability) ProductRelease() string {
	return c.productRelease
}

func (c *Capability) Supports() []mib.CapabilitiesModule {
	return slices.Clone(c.supports)
}

// String returns a brief summary: "name (oid)".
func (c *Capability) String() string {
	if c == nil {
		return "<nil>"
	}
	return c.name + " (" + c.OID().String() + ")"
}

func (c *Capability) SetNode(nd *Node) {
	c.node = nd
}

func (c *Capability) SetModule(m *Module) {
	c.module = m
}

func (c *Capability) SetStatus(s mib.Status) {
	c.status = s
}

func (c *Capability) SetDescription(d string) {
	c.desc = d
}

func (c *Capability) SetReference(r string) {
	c.ref = r
}

func (c *Capability) SetProductRelease(r string) {
	c.productRelease = r
}

func (c *Capability) SetSupports(supports []mib.CapabilitiesModule) {
	c.supports = supports
}

// InternalNode returns the concrete node.
func (c *Capability) InternalNode() *Node {
	return c.node
}

// InternalModule returns the concrete module.
func (c *Capability) InternalModule() *Module {
	return c.module
}
