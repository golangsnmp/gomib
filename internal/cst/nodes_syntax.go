package cst

import "github.com/golangsnmp/gomib/internal/types"

// TypeSyntaxNode is implemented by all CST type expression nodes.
type TypeSyntaxNode interface {
	typeSyntaxNode()
	TypeSyntaxSpan() types.Span
}

// TypeRefSyntaxNode is a reference to a named type, e.g. Integer32 or DisplayString.
type TypeRefSyntaxNode struct {
	Name SyntaxToken
	Span types.Span
}

func (*TypeRefSyntaxNode) typeSyntaxNode()              {}
func (n *TypeRefSyntaxNode) TypeSyntaxSpan() types.Span { return n.Span }

// IntegerEnumSyntaxNode is an INTEGER (or named type) with named number values,
// e.g. INTEGER { up(1), down(2) }.
type IntegerEnumSyntaxNode struct {
	Base   SyntaxToken // INTEGER keyword or named type (e.g. a TC name)
	LBrace SyntaxToken
	Values []EnumValueNode
	Commas []SyntaxToken
	RBrace SyntaxToken
	Span   types.Span
}

func (*IntegerEnumSyntaxNode) typeSyntaxNode()              {}
func (n *IntegerEnumSyntaxNode) TypeSyntaxSpan() types.Span { return n.Span }

// EnumValueNode is a single named value: label(number). Used by both INTEGER
// enumerations and BITS definitions.
type EnumValueNode struct {
	Label  SyntaxToken // name
	LParen SyntaxToken
	Value  SyntaxToken // number
	RParen SyntaxToken
	Span   types.Span
}

// BitsSyntaxNode is a BITS type with named bit positions,
// e.g. BITS { flag1(0), flag2(1) }.
type BitsSyntaxNode struct {
	Keyword SyntaxToken // BITS keyword
	LBrace  SyntaxToken
	Values  []EnumValueNode
	Commas  []SyntaxToken
	RBrace  SyntaxToken
	Span    types.Span
}

func (*BitsSyntaxNode) typeSyntaxNode()              {}
func (n *BitsSyntaxNode) TypeSyntaxSpan() types.Span { return n.Span }

// ConstrainedSyntaxNode is a type with a subtype constraint,
// e.g. INTEGER (0..255) or OCTET STRING (SIZE (0..255)).
type ConstrainedSyntaxNode struct {
	Base       TypeSyntaxNode
	Constraint *ConstraintNode
	Span       types.Span
}

func (*ConstrainedSyntaxNode) typeSyntaxNode()              {}
func (n *ConstrainedSyntaxNode) TypeSyntaxSpan() types.Span { return n.Span }

// ConstraintNode represents a parenthesized constraint expression.
// Forms: (range | range ...), (SIZE (range | range ...))
type ConstraintNode struct {
	LParen      SyntaxToken // outer (
	Size        SyntaxToken // SIZE keyword (zero if value constraint)
	InnerLParen SyntaxToken // inner ( after SIZE (zero if no SIZE)
	Ranges      []RangeNode
	Pipes       []SyntaxToken // | separators between ranges
	InnerRParen SyntaxToken   // inner ) matching InnerLParen (zero if no SIZE)
	RParen      SyntaxToken   // outer )
	Span        types.Span
}

// RangeNode is a single range or value within a constraint.
// Single value: just Min. Range: Min .. Max.
type RangeNode struct {
	Min    SyntaxToken // number, negative number, or MIN/MAX keyword identifier
	DotDot SyntaxToken // .. (zero if single value)
	Max    SyntaxToken // upper bound (zero if single value)
	Span   types.Span
}

// SequenceOfSyntaxNode is a SEQUENCE OF type, e.g. SEQUENCE OF FooEntry.
type SequenceOfSyntaxNode struct {
	Sequence SyntaxToken // SEQUENCE keyword
	Of       SyntaxToken // OF keyword
	Entry    SyntaxToken // entry type name
	Span     types.Span
}

func (*SequenceOfSyntaxNode) typeSyntaxNode()              {}
func (n *SequenceOfSyntaxNode) TypeSyntaxSpan() types.Span { return n.Span }

// SequenceSyntaxNode is a SEQUENCE with named fields, used for row types.
type SequenceSyntaxNode struct {
	Keyword SyntaxToken // SEQUENCE keyword
	LBrace  SyntaxToken
	Fields  []SequenceFieldNode
	Commas  []SyntaxToken
	RBrace  SyntaxToken
	Span    types.Span
}

func (*SequenceSyntaxNode) typeSyntaxNode()              {}
func (n *SequenceSyntaxNode) TypeSyntaxSpan() types.Span { return n.Span }

// SequenceFieldNode is a single field in a SEQUENCE or CHOICE body.
type SequenceFieldNode struct {
	Name   SyntaxToken
	Syntax TypeSyntaxNode
	Span   types.Span
}

// OctetStringSyntaxNode is an explicit OCTET STRING type reference.
type OctetStringSyntaxNode struct {
	Octet  SyntaxToken // OCTET keyword
	String SyntaxToken // STRING keyword
	Span   types.Span
}

func (*OctetStringSyntaxNode) typeSyntaxNode()              {}
func (n *OctetStringSyntaxNode) TypeSyntaxSpan() types.Span { return n.Span }

// ObjectIdentifierSyntaxNode is an explicit OBJECT IDENTIFIER type reference.
type ObjectIdentifierSyntaxNode struct {
	Object     SyntaxToken // OBJECT keyword
	Identifier SyntaxToken // IDENTIFIER keyword
	Span       types.Span
}

func (*ObjectIdentifierSyntaxNode) typeSyntaxNode()              {}
func (n *ObjectIdentifierSyntaxNode) TypeSyntaxSpan() types.Span { return n.Span }

// TaggedSyntaxNode is an ASN.1 tagged type, e.g. [APPLICATION 4] IMPLICIT OCTET STRING.
type TaggedSyntaxNode struct {
	LBracket SyntaxToken // [
	TagClass SyntaxToken // APPLICATION, UNIVERSAL, etc.
	TagNum   SyntaxToken // tag number
	RBracket SyntaxToken // ]
	Implicit SyntaxToken // IMPLICIT keyword (zero if absent)
	Inner    TypeSyntaxNode
	Span     types.Span
}

func (*TaggedSyntaxNode) typeSyntaxNode()              {}
func (n *TaggedSyntaxNode) TypeSyntaxSpan() types.Span { return n.Span }

// ChoiceSyntaxNode is a CHOICE type with named alternatives.
type ChoiceSyntaxNode struct {
	Keyword SyntaxToken // CHOICE keyword
	LBrace  SyntaxToken
	Fields  []SequenceFieldNode
	Commas  []SyntaxToken
	RBrace  SyntaxToken
	Span    types.Span
}

func (*ChoiceSyntaxNode) typeSyntaxNode()              {}
func (n *ChoiceSyntaxNode) TypeSyntaxSpan() types.Span { return n.Span }
