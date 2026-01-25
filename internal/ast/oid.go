package ast

import (
	"github.com/golangsnmp/gomib/internal/types"
)

// OidAssignment represents an OID value assignment.
//
// This is the { parent subid ... } portion of an OID definition.
//
// Examples:
//   - { ifEntry 1 } - simple parent + subid
//   - { iso org(3) dod(6) internet(1) } - full path with named numbers
//   - { mib-2 31 } - parent reference + number
type OidAssignment struct {
	// Components are the OID components.
	Components []OidComponent
	// Span is the source location (includes braces).
	Span types.Span
}

// NewOidAssignment creates a new OID assignment.
func NewOidAssignment(components []OidComponent, span types.Span) OidAssignment {
	return OidAssignment{Components: components, Span: span}
}

// OidComponent is a component of an OID value.
//
// OID components can be:
//   - Just a name: internet
//   - Just a number: 1
//   - Name with number: org(3)
//   - Qualified name: SNMPv2-SMI.enterprises
//   - Qualified name with number: SNMPv2-SMI.enterprises(1)
type OidComponent interface {
	// ComponentSpan returns the source location of this component.
	ComponentSpan() types.Span
	// Number returns the numeric value if this component has one.
	Number() (uint32, bool)
	// ComponentName returns the name if this component has one.
	ComponentName() *Ident
	// Module returns the module name if this is a qualified reference.
	Module() *Ident
	// ensure only this package can implement
	oidComponent()
}

// OidComponentName is a named reference: internet, ifEntry
type OidComponentName struct {
	Name Ident
}

func (c *OidComponentName) ComponentSpan() types.Span { return c.Name.Span }
func (c *OidComponentName) Number() (uint32, bool)    { return 0, false }
func (c *OidComponentName) ComponentName() *Ident     { return &c.Name }
func (c *OidComponentName) Module() *Ident            { return nil }
func (c *OidComponentName) oidComponent()             {}

// OidComponentNumber is a numeric subid: 1, 31
type OidComponentNumber struct {
	Value uint32
	Span  types.Span
}

func (c *OidComponentNumber) ComponentSpan() types.Span { return c.Span }
func (c *OidComponentNumber) Number() (uint32, bool)    { return c.Value, true }
func (c *OidComponentNumber) ComponentName() *Ident     { return nil }
func (c *OidComponentNumber) Module() *Ident            { return nil }
func (c *OidComponentNumber) oidComponent()             {}

// OidComponentNamedNumber is name with number: iso(1), org(3)
type OidComponentNamedNumber struct {
	Name Ident
	Num  uint32
	Span types.Span
}

func (c *OidComponentNamedNumber) ComponentSpan() types.Span { return c.Span }
func (c *OidComponentNamedNumber) Number() (uint32, bool)    { return c.Num, true }
func (c *OidComponentNamedNumber) ComponentName() *Ident     { return &c.Name }
func (c *OidComponentNamedNumber) Module() *Ident            { return nil }
func (c *OidComponentNamedNumber) oidComponent()             {}

// OidComponentQualifiedName is a qualified name reference: SNMPv2-SMI.enterprises
type OidComponentQualifiedName struct {
	ModuleName Ident
	Name       Ident
	Span       types.Span
}

func (c *OidComponentQualifiedName) ComponentSpan() types.Span { return c.Span }
func (c *OidComponentQualifiedName) Number() (uint32, bool)    { return 0, false }
func (c *OidComponentQualifiedName) ComponentName() *Ident     { return &c.Name }
func (c *OidComponentQualifiedName) Module() *Ident            { return &c.ModuleName }
func (c *OidComponentQualifiedName) oidComponent()             {}

// OidComponentQualifiedNamedNumber is a qualified name with number: SNMPv2-SMI.enterprises(1)
type OidComponentQualifiedNamedNumber struct {
	ModuleName Ident
	Name       Ident
	Num        uint32
	Span       types.Span
}

func (c *OidComponentQualifiedNamedNumber) ComponentSpan() types.Span { return c.Span }
func (c *OidComponentQualifiedNamedNumber) Number() (uint32, bool)    { return c.Num, true }
func (c *OidComponentQualifiedNamedNumber) ComponentName() *Ident     { return &c.Name }
func (c *OidComponentQualifiedNamedNumber) Module() *Ident            { return &c.ModuleName }
func (c *OidComponentQualifiedNamedNumber) oidComponent()             {}
