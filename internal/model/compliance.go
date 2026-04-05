package model

import "slices"

// Compliance is a MODULE-COMPLIANCE definition.
type Compliance struct {
	entity
	modules []ComplianceModule
}

// Modules returns the MODULE clauses within this compliance statement.
func (c *Compliance) Modules() []ComplianceModule {
	result := slices.Clone(c.modules)
	for i := range result {
		result[i] = result[i].clone()
	}
	return result
}

func (c *Compliance) setModules(modules []ComplianceModule) { c.modules = modules }
