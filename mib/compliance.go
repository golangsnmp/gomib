package mib

import "slices"

// Compliance is a MODULE-COMPLIANCE definition.
type Compliance struct {
	name    string
	node    *Node
	module  *Module
	status  Status
	desc    string
	ref     string
	modules []ComplianceModule
}

// NewCompliance returns a Compliance initialized with the given name.
func NewCompliance(name string) *Compliance {
	return &Compliance{name: name}
}

func (c *Compliance) Name() string                { return c.name }
func (c *Compliance) Node() *Node                 { return c.node }
func (c *Compliance) Module() *Module             { return c.module }
func (c *Compliance) Status() Status              { return c.status }
func (c *Compliance) Description() string         { return c.desc }
func (c *Compliance) Reference() string           { return c.ref }
func (c *Compliance) Modules() []ComplianceModule { return slices.Clone(c.modules) }

func (c *Compliance) OID() OID {
	if c.node == nil {
		return nil
	}
	return c.node.OID()
}

// String returns a brief summary: "name (oid)".
func (c *Compliance) String() string {
	if c == nil {
		return "<nil>"
	}
	return c.name + " (" + c.OID().String() + ")"
}

func (c *Compliance) SetNode(nd *Node)                      { c.node = nd }
func (c *Compliance) SetModule(m *Module)                   { c.module = m }
func (c *Compliance) SetStatus(s Status)                    { c.status = s }
func (c *Compliance) SetDescription(d string)               { c.desc = d }
func (c *Compliance) SetReference(r string)                 { c.ref = r }
func (c *Compliance) SetModules(modules []ComplianceModule) { c.modules = modules }
