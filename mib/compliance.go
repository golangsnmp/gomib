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

// newCompliance returns a Compliance initialized with the given name.
func newCompliance(name string) *Compliance {
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
	if c == nil || c.node == nil {
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

func (c *Compliance) setNode(nd *Node)                      { c.node = nd }
func (c *Compliance) setModule(m *Module)                   { c.module = m }
func (c *Compliance) setStatus(s Status)                    { c.status = s }
func (c *Compliance) setDescription(d string)               { c.desc = d }
func (c *Compliance) setReference(r string)                 { c.ref = r }
func (c *Compliance) setModules(modules []ComplianceModule) { c.modules = modules }
