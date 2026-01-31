package mibimpl

import "github.com/golangsnmp/gomib/mib"

// Compliance is the concrete implementation of mib.Compliance.
type Compliance struct {
	name    string
	node    *Node
	module  *Module
	status  mib.Status
	desc    string
	ref     string
	modules []mib.ComplianceModule
}

// Interface methods (mib.Compliance)

func (c *Compliance) Name() string {
	return c.name
}

func (c *Compliance) Node() mib.Node {
	if c.node == nil {
		return nil
	}
	return c.node
}

func (c *Compliance) Module() mib.Module {
	if c.module == nil {
		return nil
	}
	return c.module
}

func (c *Compliance) OID() mib.Oid {
	if c.node == nil {
		return nil
	}
	return c.node.OID()
}

func (c *Compliance) Status() mib.Status {
	return c.status
}

func (c *Compliance) Description() string {
	return c.desc
}

func (c *Compliance) Reference() string {
	return c.ref
}

func (c *Compliance) Modules() []mib.ComplianceModule {
	return c.modules
}

// String returns a brief summary: "name (oid)".
func (c *Compliance) String() string {
	if c == nil {
		return "<nil>"
	}
	return c.name + " (" + c.OID().String() + ")"
}

// Mutation methods (for resolver use)

func (c *Compliance) SetNode(nd *Node) {
	c.node = nd
}

func (c *Compliance) SetModule(m *Module) {
	c.module = m
}

func (c *Compliance) SetStatus(s mib.Status) {
	c.status = s
}

func (c *Compliance) SetDescription(d string) {
	c.desc = d
}

func (c *Compliance) SetReference(r string) {
	c.ref = r
}

func (c *Compliance) SetModules(modules []mib.ComplianceModule) {
	c.modules = modules
}

// InternalNode returns the concrete node for resolver use.
func (c *Compliance) InternalNode() *Node {
	return c.node
}

// InternalModule returns the concrete module for resolver use.
func (c *Compliance) InternalModule() *Module {
	return c.module
}
