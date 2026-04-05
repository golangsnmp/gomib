package model

import "slices"

// Capability is an AGENT-CAPABILITIES definition.
type Capability struct {
	entity
	productRelease string
	supports       []CapabilitiesModule
}

// ProductRelease returns the PRODUCT-RELEASE clause text.
func (c *Capability) ProductRelease() string { return c.productRelease }

// Supports returns the SUPPORTS clauses listing the modules this agent implements.
func (c *Capability) Supports() []CapabilitiesModule {
	result := slices.Clone(c.supports)
	for i := range result {
		result[i] = result[i].clone()
	}
	return result
}

func (c *Capability) setProductRelease(r string)                { c.productRelease = r }
func (c *Capability) setSupports(supports []CapabilitiesModule) { c.supports = supports }
