package mibimpl

import "github.com/golangsnmp/gomib/mib"

// Capabilities is the concrete implementation of mib.Capabilities.
type Capabilities struct {
	name           string
	node           *Node
	module         *Module
	status         mib.Status
	desc           string
	ref            string
	productRelease string
	supports       []mib.CapabilitiesModule
}

// Interface methods (mib.Capabilities)

func (c *Capabilities) Name() string {
	return c.name
}

func (c *Capabilities) Node() mib.Node {
	if c.node == nil {
		return nil
	}
	return c.node
}

func (c *Capabilities) Module() mib.Module {
	if c.module == nil {
		return nil
	}
	return c.module
}

func (c *Capabilities) OID() mib.Oid {
	if c.node == nil {
		return nil
	}
	return c.node.OID()
}

func (c *Capabilities) Status() mib.Status {
	return c.status
}

func (c *Capabilities) Description() string {
	return c.desc
}

func (c *Capabilities) Reference() string {
	return c.ref
}

func (c *Capabilities) ProductRelease() string {
	return c.productRelease
}

func (c *Capabilities) Supports() []mib.CapabilitiesModule {
	return c.supports
}

// String returns a brief summary: "name (oid)".
func (c *Capabilities) String() string {
	if c == nil {
		return "<nil>"
	}
	return c.name + " (" + c.OID().String() + ")"
}

// Mutation methods (for resolver use)

func (c *Capabilities) SetNode(nd *Node) {
	c.node = nd
}

func (c *Capabilities) SetModule(m *Module) {
	c.module = m
}

func (c *Capabilities) SetStatus(s mib.Status) {
	c.status = s
}

func (c *Capabilities) SetDescription(d string) {
	c.desc = d
}

func (c *Capabilities) SetReference(r string) {
	c.ref = r
}

func (c *Capabilities) SetProductRelease(r string) {
	c.productRelease = r
}

func (c *Capabilities) SetSupports(supports []mib.CapabilitiesModule) {
	c.supports = supports
}

// InternalNode returns the concrete node for resolver use.
func (c *Capabilities) InternalNode() *Node {
	return c.node
}

// InternalModule returns the concrete module for resolver use.
func (c *Capabilities) InternalModule() *Module {
	return c.module
}
