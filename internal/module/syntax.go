package module

import (
	"github.com/golangsnmp/gomib/internal/types"
)

// NamedNumber is a named number in an INTEGER enumeration.
//
// Used in `INTEGER { up(1), down(2) }` syntax.
type NamedNumber struct {
	// Name is the name of the enumeration value (e.g., "up", "down").
	Name string
	// Value is the numeric value assigned to this name.
	Value int64
}

// NewNamedNumber creates a new named number.
func NewNamedNumber(name string, value int64) NamedNumber {
	return NamedNumber{Name: name, Value: value}
}

// NamedBit is a named bit in a BITS type definition.
//
// Used in `BITS { flag1(0), flag2(1) }` syntax.
type NamedBit struct {
	// Name is the name of the bit (e.g., "flag1", "flag2").
	Name string
	// Position is the bit position (0-indexed from the left).
	Position uint32
}

// NewNamedBit creates a new named bit.
func NewNamedBit(name string, position uint32) NamedBit {
	return NamedBit{Name: name, Position: position}
}

// SequenceField is a field in a SEQUENCE type (used for row entry types).
//
// Used in `SEQUENCE { ifIndex InterfaceIndex, ifDescr DisplayString }` syntax.
type SequenceField struct {
	// Name is the name of the field (e.g., "ifIndex", "ifDescr").
	Name string
	// Syntax is the type of the field.
	Syntax TypeSyntax
}

// NewSequenceField creates a new sequence field.
func NewSequenceField(name string, syntax TypeSyntax) SequenceField {
	return SequenceField{Name: name, Syntax: syntax}
}

// OidAssignment is an unresolved OID assignment.
//
// Keeps OID components as symbols; resolution happens in the resolver.
type OidAssignment struct {
	// Components are the OID components.
	Components []OidComponent
	// Span is the source span for diagnostics.
	Span types.Span
}

// NewOidAssignment creates a new OID assignment.
func NewOidAssignment(components []OidComponent, span types.Span) OidAssignment {
	return OidAssignment{Components: components, Span: span}
}

// OidComponent is a component of an OID assignment.
// Use type switches to distinguish concrete types:
//   - *OidComponentName: just a name reference
//   - *OidComponentNumber: just a number
//   - *OidComponentNamedNumber: name with number like org(3)
//   - *OidComponentQualifiedName: qualified reference like SNMPv2-SMI.enterprises
//   - *OidComponentQualifiedNamedNumber: qualified with number
type OidComponent interface {
	// oidComponent marker
	oidComponent()
}

// OidComponentName is just a name reference: `internet`, `ifEntry`
type OidComponentName struct {
	NameValue string
}

func (*OidComponentName) oidComponent() {}

// OidComponentNumber is just a number: `1`, `31`
type OidComponentNumber struct {
	Value uint32
}

func (*OidComponentNumber) oidComponent() {}

// OidComponentNamedNumber is a name with number: `org(3)` - common in well-known roots
type OidComponentNamedNumber struct {
	NameValue   string
	NumberValue uint32
}

func (*OidComponentNamedNumber) oidComponent() {}

// OidComponentQualifiedName is a qualified name: `SNMPv2-SMI.enterprises`
type OidComponentQualifiedName struct {
	ModuleValue string
	NameValue   string
}

func (*OidComponentQualifiedName) oidComponent() {}

// OidComponentQualifiedNamedNumber is a qualified name with number: `SNMPv2-SMI.enterprises(1)`
type OidComponentQualifiedNamedNumber struct {
	ModuleValue string
	NameValue   string
	NumberValue uint32
}

func (*OidComponentQualifiedNamedNumber) oidComponent() {}

// TypeSyntax is a type representation with symbol references (not resolved).
// Use type switches to distinguish concrete types:
//   - *TypeSyntaxSequenceOf for SEQUENCE OF (table type)
//   - *TypeSyntaxSequence for SEQUENCE (row type)
//   - *TypeSyntaxTypeRef, *TypeSyntaxIntegerEnum, etc. for other forms
type TypeSyntax interface {
	// typeSyntax marker
	typeSyntax()
}

// TypeSyntaxTypeRef is a reference to another type: `Integer32`, `DisplayString`
type TypeSyntaxTypeRef struct {
	Name string
}

func (*TypeSyntaxTypeRef) typeSyntax() {}

// TypeSyntaxIntegerEnum is INTEGER with enum values: `INTEGER { up(1), down(2) }`
type TypeSyntaxIntegerEnum struct {
	NamedNumbers []NamedNumber
}

func (*TypeSyntaxIntegerEnum) typeSyntax() {}

// TypeSyntaxBits is BITS with named bits: `BITS { flag1(0), flag2(1) }`
type TypeSyntaxBits struct {
	NamedBits []NamedBit
}

func (*TypeSyntaxBits) typeSyntax() {}

// TypeSyntaxConstrained is a constrained type: `OCTET STRING (SIZE (0..255))`
type TypeSyntaxConstrained struct {
	// Base is the base type.
	Base TypeSyntax
	// Constraint is the constraint.
	Constraint Constraint
}

func (*TypeSyntaxConstrained) typeSyntax() {}

// TypeSyntaxSequenceOf is SEQUENCE OF: `SEQUENCE OF IfEntry`
type TypeSyntaxSequenceOf struct {
	EntryType string
}

func (*TypeSyntaxSequenceOf) typeSyntax() {}

// TypeSyntaxSequence is SEQUENCE with fields (for row types).
type TypeSyntaxSequence struct {
	Fields []SequenceField
}

func (*TypeSyntaxSequence) typeSyntax() {}

// TypeSyntaxOctetString is OCTET STRING (explicit).
type TypeSyntaxOctetString struct{}

func (*TypeSyntaxOctetString) typeSyntax() {}

// TypeSyntaxObjectIdentifier is OBJECT IDENTIFIER (explicit).
type TypeSyntaxObjectIdentifier struct{}

func (*TypeSyntaxObjectIdentifier) typeSyntax() {}

// Constraint is a type constraint.
type Constraint interface {
	// constraint marker
	constraint()
}

// ConstraintSize is a SIZE constraint: `(SIZE (0..255))`
type ConstraintSize struct {
	Ranges []Range
}

func (*ConstraintSize) constraint() {}

// ConstraintRange is a value range constraint: `(0..65535)`
type ConstraintRange struct {
	Ranges []Range
}

func (*ConstraintRange) constraint() {}

// Range is a range in a constraint.
type Range struct {
	// Min is the minimum value.
	Min RangeValue
	// Max is the maximum value (nil for single value).
	Max RangeValue
}

// NewRangeSingleSigned creates a single-value range with a signed value.
func NewRangeSingleSigned(value int64) Range {
	return Range{Min: &RangeValueSigned{Value: value}, Max: nil}
}

// NewRangeSingleUnsigned creates a single-value range with an unsigned value.
func NewRangeSingleUnsigned(value uint64) Range {
	return Range{Min: &RangeValueUnsigned{Value: value}, Max: nil}
}

// NewRangeSigned creates a range from min to max with signed values.
func NewRangeSigned(min, max int64) Range {
	return Range{
		Min: &RangeValueSigned{Value: min},
		Max: &RangeValueSigned{Value: max},
	}
}

// NewRangeUnsigned creates a range from min to max with unsigned values.
func NewRangeUnsigned(min, max uint64) Range {
	return Range{
		Min: &RangeValueUnsigned{Value: min},
		Max: &RangeValueUnsigned{Value: max},
	}
}

// RangeValue is a value in a range constraint.
type RangeValue interface {
	// rangeValue marker
	rangeValue()
}

// RangeValueSigned is a signed numeric value (for Integer32 ranges, can be negative).
type RangeValueSigned struct {
	Value int64
}

func (*RangeValueSigned) rangeValue() {}

// RangeValueUnsigned is an unsigned numeric value (for Counter64 ranges, large positive values).
type RangeValueUnsigned struct {
	Value uint64
}

func (*RangeValueUnsigned) rangeValue() {}

// RangeValueMin is the MIN keyword.
type RangeValueMin struct{}

func (*RangeValueMin) rangeValue() {}

// RangeValueMax is the MAX keyword.
type RangeValueMax struct{}

func (*RangeValueMax) rangeValue() {}

// DefVal is a default value for an OBJECT-TYPE.
//
// This is the normalized representation of DEFVAL clause content.
// Symbol references are kept unresolved; resolution happens in the semantic phase.
type DefVal interface {
	// defVal marker
	defVal()
}

// DefValInteger is an integer value: `DEFVAL { 0 }`, `DEFVAL { -1 }`
type DefValInteger struct {
	Value int64
}

func (*DefValInteger) defVal() {}

// DefValUnsigned is an unsigned integer (for Counter64 etc): `DEFVAL { 4294967296 }`
type DefValUnsigned struct {
	Value uint64
}

func (*DefValUnsigned) defVal() {}

// DefValString is a string value: `DEFVAL { "public" }`, `DEFVAL { "" }`
type DefValString struct {
	Value string
}

func (*DefValString) defVal() {}

// DefValHexString is a hex string: `DEFVAL { 'FF00'H }`
// Stored as raw hex digits (uppercase).
type DefValHexString struct {
	Value string
}

func (*DefValHexString) defVal() {}

// DefValBinaryString is a binary string: `DEFVAL { '1010'B }`
// Stored as raw binary digits.
type DefValBinaryString struct {
	Value string
}

func (*DefValBinaryString) defVal() {}

// DefValEnum is an enum label reference: `DEFVAL { enabled }`, `DEFVAL { true }`
// The symbol refers to an enumeration value defined in the object's type.
type DefValEnum struct {
	Name string
}

func (*DefValEnum) defVal() {}

// DefValBits is a BITS value (set of bit labels): `DEFVAL { { flag1, flag2 } }`, `DEFVAL { {} }`
// Each symbol refers to a bit name defined in the object's BITS type.
type DefValBits struct {
	Labels []string
}

func (*DefValBits) defVal() {}

// DefValOidRef is an OID reference: `DEFVAL { sysName }`
// The symbol refers to another OID in the MIB.
type DefValOidRef struct {
	Name string
}

func (*DefValOidRef) defVal() {}

// DefValOidValue is an OID value (explicit components): `DEFVAL { { iso 3 6 1 } }`
// Kept as OID components for resolution.
type DefValOidValue struct {
	Components []OidComponent
}

func (*DefValOidValue) defVal() {}
