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

// OidComponent is a single element in an OID value assignment.
// Use type switches to distinguish concrete types (Name, Number,
// NamedNumber, QualifiedName, QualifiedNamedNumber).
type OidComponent interface {
	ComponentSpan() types.Span
	oidComponent()
}

// OidComponentName is a named reference, e.g. internet, ifEntry.
type OidComponentName struct {
	Name Ident
}

func (c *OidComponentName) ComponentSpan() types.Span { return c.Name.Span }
func (c *OidComponentName) oidComponent()             {}

// OidComponentNumber is a numeric sub-identifier, e.g. 1, 31.
type OidComponentNumber struct {
	Value uint32
	Span  types.Span
}

func (c *OidComponentNumber) ComponentSpan() types.Span { return c.Span }
func (c *OidComponentNumber) oidComponent()             {}

// OidComponentNamedNumber is a name with number, e.g. iso(1), org(3).
type OidComponentNamedNumber struct {
	Name Ident
	Num  uint32
	Span types.Span
}

func (c *OidComponentNamedNumber) ComponentSpan() types.Span { return c.Span }
func (c *OidComponentNamedNumber) oidComponent()             {}

// OidComponentQualifiedName is a module-qualified reference,
// e.g. SNMPv2-SMI.enterprises.
type OidComponentQualifiedName struct {
	ModuleName Ident
	Name       Ident
	Span       types.Span
}

func (c *OidComponentQualifiedName) ComponentSpan() types.Span { return c.Span }
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
func (c *OidComponentQualifiedNamedNumber) oidComponent()             {}
