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

// Name returns the compliance statement's descriptor.
func (c *Compliance) Name() string { return c.name }

// Node returns the OID tree node this compliance statement is attached to.
func (c *Compliance) Node() *Node { return c.node }

// Module returns the module that defines this compliance statement.
func (c *Compliance) Module() *Module { return c.module }

// Status returns the STATUS clause value.
func (c *Compliance) Status() Status { return c.status }

// Description returns the DESCRIPTION clause text.
func (c *Compliance) Description() string { return c.desc }

// Reference returns the REFERENCE clause text, or "".
func (c *Compliance) Reference() string { return c.ref }

// Modules returns the MODULE clauses within this compliance statement.
func (c *Compliance) Modules() []ComplianceModule { return slices.Clone(c.modules) }

// OID returns the compliance statement's position in the OID tree, or nil if unresolved.
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
