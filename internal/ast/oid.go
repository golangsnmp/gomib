package ast

import (
	"github.com/golangsnmp/gomib/internal/types"
)

// OidAssignment holds the parsed components of an OBJECT IDENTIFIER
// value, e.g. { iso org(3) dod(6) 1 }.
type OidAssignment struct {
	Components []OidComponent
	Span       types.Span
}

// NewOidAssignment creates an OidAssignment from its components.
func NewOidAssignment(components []OidComponent, span types.Span) OidAssignment {
	return OidAssignment{Components: components, Span: span}
}

// OidComponent is a single element in an OID value assignment.
type OidComponent interface {
	ComponentSpan() types.Span
	Number() (uint32, bool)
	ComponentName() *Ident
	Module() *Ident
	oidComponent()
}

// OidComponentName is a named reference, e.g. internet, ifEntry.
type OidComponentName struct {
	Name Ident
}

func (c *OidComponentName) ComponentSpan() types.Span { return c.Name.Span }
func (c *OidComponentName) Number() (uint32, bool)    { return 0, false }
func (c *OidComponentName) ComponentName() *Ident     { return &c.Name }
func (c *OidComponentName) Module() *Ident            { return nil }
func (c *OidComponentName) oidComponent()             {}

// OidComponentNumber is a numeric sub-identifier, e.g. 1, 31.
type OidComponentNumber struct {
	Value uint32
	Span  types.Span
}

func (c *OidComponentNumber) ComponentSpan() types.Span { return c.Span }
func (c *OidComponentNumber) Number() (uint32, bool)    { return c.Value, true }
func (c *OidComponentNumber) ComponentName() *Ident     { return nil }
func (c *OidComponentNumber) Module() *Ident            { return nil }
func (c *OidComponentNumber) oidComponent()             {}

// OidComponentNamedNumber is a name with number, e.g. iso(1), org(3).
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

// OidComponentQualifiedName is a module-qualified reference,
// e.g. SNMPv2-SMI.enterprises.
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

// OidComponentQualifiedNamedNumber is a module-qualified name with
// number, e.g. SNMPv2-SMI.enterprises(1).
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
