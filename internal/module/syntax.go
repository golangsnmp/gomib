package module

import (
	"github.com/golangsnmp/gomib/internal/types"
)

// NamedNumber is a named value in an INTEGER enumeration,
// e.g. up(1) in INTEGER { up(1), down(2) }.
type NamedNumber struct {
	Name  string
	Value int64
}

// NewNamedNumber returns a NamedNumber with the given name and value.
func NewNamedNumber(name string, value int64) NamedNumber {
	return NamedNumber{Name: name, Value: value}
}

// NamedBit is a named bit position in a BITS type definition,
// e.g. flag1(0) in BITS { flag1(0), flag2(1) }.
type NamedBit struct {
	Name     string
	Position uint32
}

// NewNamedBit returns a NamedBit with the given name and position.
func NewNamedBit(name string, position uint32) NamedBit {
	return NamedBit{Name: name, Position: position}
}

// SequenceField is a field in a SEQUENCE type used for table row entries.
type SequenceField struct {
	Name   string
	Syntax TypeSyntax
}

// NewSequenceField returns a SequenceField with the given name and syntax.
func NewSequenceField(name string, syntax TypeSyntax) SequenceField {
	return SequenceField{Name: name, Syntax: syntax}
}

// OidAssignment is an unresolved OID assignment. Components remain as
// symbolic references until the resolver phase.
type OidAssignment struct {
	Components []OidComponent
	Span       types.Span
}

// NewOidAssignment returns an OidAssignment with the given components.
func NewOidAssignment(components []OidComponent, span types.Span) OidAssignment {
	return OidAssignment{Components: components, Span: span}
}

// OidComponent is one element of an OID assignment. Use type switches to
// distinguish concrete types (Name, Number, NamedNumber, QualifiedName,
// QualifiedNamedNumber).
type OidComponent interface {
	oidComponent()
}

// OidComponentName is a symbolic name reference, e.g. internet.
type OidComponentName struct {
	NameValue string
}

func (*OidComponentName) oidComponent() {}

// OidComponentNumber is a numeric arc, e.g. 1 or 31.
type OidComponentNumber struct {
	Value uint32
}

func (*OidComponentNumber) oidComponent() {}

// OidComponentNamedNumber is a name with number, e.g. org(3).
type OidComponentNamedNumber struct {
	NameValue   string
	NumberValue uint32
}

func (*OidComponentNamedNumber) oidComponent() {}

// OidComponentQualifiedName is a module-qualified name, e.g. SNMPv2-SMI.enterprises.
type OidComponentQualifiedName struct {
	ModuleValue string
	NameValue   string
}

func (*OidComponentQualifiedName) oidComponent() {}

// OidComponentQualifiedNamedNumber is a module-qualified name with number,
// e.g. SNMPv2-SMI.enterprises(1).
type OidComponentQualifiedNamedNumber struct {
	ModuleValue string
	NameValue   string
	NumberValue uint32
}

func (*OidComponentQualifiedNamedNumber) oidComponent() {}

// TypeSyntax is an unresolved type representation. Use type switches to
// dispatch on concrete types (TypeRef, IntegerEnum, Bits, Constrained,
// SequenceOf, Sequence, OctetString, ObjectIdentifier).
type TypeSyntax interface {
	typeSyntax()
}

// TypeSyntaxTypeRef is a reference to a named type, e.g. Integer32.
type TypeSyntaxTypeRef struct {
	Name string
}

func (*TypeSyntaxTypeRef) typeSyntax() {}

// TypeSyntaxIntegerEnum is an INTEGER with named values, e.g.
// INTEGER { up(1), down(2) }. Base is non-empty when the enum restricts
// a named type rather than bare INTEGER.
type TypeSyntaxIntegerEnum struct {
	Base         string
	NamedNumbers []NamedNumber
}

func (*TypeSyntaxIntegerEnum) typeSyntax() {}

// TypeSyntaxBits is a BITS type with named bit positions.
type TypeSyntaxBits struct {
	NamedBits []NamedBit
}

func (*TypeSyntaxBits) typeSyntax() {}

// TypeSyntaxConstrained is a type with a subtype constraint applied.
type TypeSyntaxConstrained struct {
	Base       TypeSyntax
	Constraint Constraint
}

func (*TypeSyntaxConstrained) typeSyntax() {}

// TypeSyntaxSequenceOf is SEQUENCE OF, used for table types.
type TypeSyntaxSequenceOf struct {
	EntryType string
}

func (*TypeSyntaxSequenceOf) typeSyntax() {}

// TypeSyntaxSequence is a SEQUENCE with named fields, used for row types.
type TypeSyntaxSequence struct {
	Fields []SequenceField
}

func (*TypeSyntaxSequence) typeSyntax() {}

// TypeSyntaxOctetString is an explicit OCTET STRING reference.
type TypeSyntaxOctetString struct{}

func (*TypeSyntaxOctetString) typeSyntax() {}

// TypeSyntaxObjectIdentifier is an explicit OBJECT IDENTIFIER reference.
type TypeSyntaxObjectIdentifier struct{}

func (*TypeSyntaxObjectIdentifier) typeSyntax() {}

// Constraint is a subtype constraint (SIZE or value range).
type Constraint interface {
	constraint()
}

// ConstraintSize is a SIZE constraint, e.g. (SIZE (0..255)).
type ConstraintSize struct {
	Ranges []Range
}

func (*ConstraintSize) constraint() {}

// ConstraintRange is a value range constraint, e.g. (0..65535).
type ConstraintRange struct {
	Ranges []Range
}

func (*ConstraintRange) constraint() {}

// Range is a single range or value within a constraint. Max is nil for
// single-value constraints.
type Range struct {
	Min RangeValue
	Max RangeValue
}

// NewRangeSingleSigned returns a single-value Range with a signed value.
func NewRangeSingleSigned(value int64) Range {
	return Range{Min: &RangeValueSigned{Value: value}, Max: nil}
}

// NewRangeSingleUnsigned returns a single-value Range with an unsigned value.
func NewRangeSingleUnsigned(value uint64) Range {
	return Range{Min: &RangeValueUnsigned{Value: value}, Max: nil}
}

// NewRangeSigned returns a Range from min to max with signed values.
func NewRangeSigned(min, max int64) Range {
	return Range{
		Min: &RangeValueSigned{Value: min},
		Max: &RangeValueSigned{Value: max},
	}
}

// NewRangeUnsigned returns a Range from min to max with unsigned values.
func NewRangeUnsigned(min, max uint64) Range {
	return Range{
		Min: &RangeValueUnsigned{Value: min},
		Max: &RangeValueUnsigned{Value: max},
	}
}

// RangeValue is one endpoint of a Range (signed, unsigned, MIN, or MAX).
type RangeValue interface {
	rangeValue()
}

// RangeValueSigned is a signed range endpoint, used for Integer32 ranges.
type RangeValueSigned struct {
	Value int64
}

func (*RangeValueSigned) rangeValue() {}

// RangeValueUnsigned is an unsigned range endpoint, used for Counter64 ranges.
type RangeValueUnsigned struct {
	Value uint64
}

func (*RangeValueUnsigned) rangeValue() {}

// RangeValueMin represents the MIN keyword in a range constraint.
type RangeValueMin struct{}

func (*RangeValueMin) rangeValue() {}

// RangeValueMax represents the MAX keyword in a range constraint.
type RangeValueMax struct{}

func (*RangeValueMax) rangeValue() {}

// DefVal is an unresolved DEFVAL clause value. Symbol references remain
// unresolved until the semantic phase.
type DefVal interface {
	defVal()
}

// DefValInteger is a signed integer DEFVAL, e.g. DEFVAL { 0 }.
type DefValInteger struct {
	Value int64
}

func (*DefValInteger) defVal() {}

// DefValUnsigned is an unsigned integer DEFVAL for Counter64 and similar.
type DefValUnsigned struct {
	Value uint64
}

func (*DefValUnsigned) defVal() {}

// DefValString is a quoted string DEFVAL, e.g. DEFVAL { "public" }.
type DefValString struct {
	Value string
}

func (*DefValString) defVal() {}

// DefValHexString is a hex string DEFVAL, e.g. DEFVAL { 'FF00'H }.
// Value contains raw uppercase hex digits.
type DefValHexString struct {
	Value string
}

func (*DefValHexString) defVal() {}

// DefValBinaryString is a binary string DEFVAL, e.g. DEFVAL { '1010'B }.
// Value contains raw binary digits.
type DefValBinaryString struct {
	Value string
}

func (*DefValBinaryString) defVal() {}

// DefValEnum is an enumeration label DEFVAL, e.g. DEFVAL { enabled }.
// The name refers to a value defined in the object's INTEGER enum type.
type DefValEnum struct {
	Name string
}

func (*DefValEnum) defVal() {}

// DefValBits is a BITS value DEFVAL, e.g. DEFVAL { { flag1, flag2 } }.
// Each label refers to a bit name in the object's BITS type.
type DefValBits struct {
	Labels []string
}

func (*DefValBits) defVal() {}

// DefValOidRef is an OID name reference DEFVAL, e.g. DEFVAL { sysName }.
type DefValOidRef struct {
	Name string
}

func (*DefValOidRef) defVal() {}

// DefValOidValue is an OID value with explicit components,
// e.g. DEFVAL { { iso 3 6 1 } }.
type DefValOidValue struct {
	Components []OidComponent
}

func (*DefValOidValue) defVal() {}

// DefValUnparsed represents a DEFVAL whose content could not be parsed.
type DefValUnparsed struct{}

func (*DefValUnparsed) defVal() {}
