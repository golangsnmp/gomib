package mibimpl

import "github.com/golangsnmp/gomib/mib"

// Compliance implements mib.Compliance for MODULE-COMPLIANCE definitions.
type Compliance struct {
	name    string
	node    *Node
	module  *Module
	status  mib.Status
	desc    string
	ref     string
	modules []mib.ComplianceModule
}

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

// InternalNode returns the concrete node.
func (c *Compliance) InternalNode() *Node {
	return c.node
}

// InternalModule returns the concrete module.
func (c *Compliance) InternalModule() *Module {
	return c.module
}
