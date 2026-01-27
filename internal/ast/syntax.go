package ast

import (
	"github.com/golangsnmp/gomib/internal/types"
)

// SyntaxClause is a SYNTAX clause specifying the type of an object.
type SyntaxClause struct {
	Syntax TypeSyntax
	Span   types.Span
}

// NewSyntaxClause creates a new syntax clause.
func NewSyntaxClause(syntax TypeSyntax, span types.Span) SyntaxClause {
	return SyntaxClause{Syntax: syntax, Span: span}
}

// TypeSyntax is type syntax in a SYNTAX clause or type assignment.
type TypeSyntax interface {
	SyntaxSpan() types.Span
	typeSyntax()
}

// TypeSyntaxTypeRef is a simple type reference.
type TypeSyntaxTypeRef struct {
	Name Ident
}

func (t *TypeSyntaxTypeRef) SyntaxSpan() types.Span { return t.Name.Span }
func (*TypeSyntaxTypeRef) typeSyntax()              {}

// TypeSyntaxIntegerEnum is INTEGER with named numbers.
type TypeSyntaxIntegerEnum struct {
	Base         *Ident
	NamedNumbers []NamedNumber
	Span         types.Span
}

func (t *TypeSyntaxIntegerEnum) SyntaxSpan() types.Span { return t.Span }
func (*TypeSyntaxIntegerEnum) typeSyntax()              {}

// TypeSyntaxBits is BITS with named bits.
type TypeSyntaxBits struct {
	NamedBits []NamedNumber
	Span      types.Span
}

func (t *TypeSyntaxBits) SyntaxSpan() types.Span { return t.Span }
func (*TypeSyntaxBits) typeSyntax()              {}

// TypeSyntaxConstrained is a constrained type.
type TypeSyntaxConstrained struct {
	Base       TypeSyntax
	Constraint Constraint
	Span       types.Span
}

func (t *TypeSyntaxConstrained) SyntaxSpan() types.Span { return t.Span }
func (*TypeSyntaxConstrained) typeSyntax()              {}

// TypeSyntaxSequenceOf is SEQUENCE OF.
type TypeSyntaxSequenceOf struct {
	EntryType Ident
	Span      types.Span
}

func (t *TypeSyntaxSequenceOf) SyntaxSpan() types.Span { return t.Span }
func (*TypeSyntaxSequenceOf) typeSyntax()              {}

// TypeSyntaxSequence is SEQUENCE (row definition).
type TypeSyntaxSequence struct {
	Fields []SequenceField
	Span   types.Span
}

func (t *TypeSyntaxSequence) SyntaxSpan() types.Span { return t.Span }
func (*TypeSyntaxSequence) typeSyntax()              {}

// TypeSyntaxChoice is a CHOICE type.
type TypeSyntaxChoice struct {
	Alternatives []ChoiceAlternative
	Span         types.Span
}

func (t *TypeSyntaxChoice) SyntaxSpan() types.Span { return t.Span }
func (*TypeSyntaxChoice) typeSyntax()              {}

// TypeSyntaxOctetString is OCTET STRING (explicit form).
type TypeSyntaxOctetString struct {
	Span types.Span
}

func (t *TypeSyntaxOctetString) SyntaxSpan() types.Span { return t.Span }
func (*TypeSyntaxOctetString) typeSyntax()              {}

// TypeSyntaxObjectIdentifier is OBJECT IDENTIFIER type.
type TypeSyntaxObjectIdentifier struct {
	Span types.Span
}

func (t *TypeSyntaxObjectIdentifier) SyntaxSpan() types.Span { return t.Span }
func (*TypeSyntaxObjectIdentifier) typeSyntax()              {}

// SequenceField is a field in a SEQUENCE definition.
type SequenceField struct {
	Name   Ident
	Syntax TypeSyntax
	Span   types.Span
}

// ChoiceAlternative is an alternative in a CHOICE type.
type ChoiceAlternative struct {
	Name   Ident
	Syntax TypeSyntax
	Span   types.Span
}

// Constraint is a type constraint (SIZE or range).
type Constraint interface {
	ConstraintSpan() types.Span
	constraint()
}

// ConstraintSize is a SIZE constraint.
type ConstraintSize struct {
	Ranges []Range
	Span   types.Span
}

func (c *ConstraintSize) ConstraintSpan() types.Span { return c.Span }
func (*ConstraintSize) constraint()                  {}

// ConstraintRange is a value range constraint.
type ConstraintRange struct {
	Ranges []Range
	Span   types.Span
}

func (c *ConstraintRange) ConstraintSpan() types.Span { return c.Span }
func (*ConstraintRange) constraint()                  {}

// Range is a range in a constraint.
type Range struct {
	Min  RangeValue
	Max  RangeValue
	Span types.Span
}

// RangeValue is a value in a range constraint.
type RangeValue interface {
	rangeValue()
}

// RangeValueSigned is a signed numeric value.
type RangeValueSigned struct {
	Value int64
}

func (*RangeValueSigned) rangeValue() {}

// RangeValueUnsigned is an unsigned numeric value.
type RangeValueUnsigned struct {
	Value uint64
}

func (*RangeValueUnsigned) rangeValue() {}

// RangeValueIdent is a named value (MIN, MAX).
type RangeValueIdent struct {
	Name Ident
}

func (*RangeValueIdent) rangeValue() {}

// AccessClause is an access clause (MAX-ACCESS or ACCESS).
type AccessClause struct {
	Keyword AccessKeyword
	Value   AccessValue
	Span    types.Span
}

// AccessKeyword is the access keyword type.
type AccessKeyword int

const (
	AccessKeywordAccess AccessKeyword = iota
	AccessKeywordMaxAccess
	AccessKeywordMinAccess
	AccessKeywordPibAccess
)

// AccessValue is an access value.
type AccessValue int

const (
	AccessValueReadOnly AccessValue = iota
	AccessValueReadWrite
	AccessValueReadCreate
	AccessValueNotAccessible
	AccessValueAccessibleForNotify
	AccessValueWriteOnly
	AccessValueNotImplemented
	AccessValueInstall
	AccessValueInstallNotify
	AccessValueReportOnly
)

// StatusClause is a status clause.
type StatusClause struct {
	Value StatusValue
	Span  types.Span
}

// StatusValue is a status value.
type StatusValue int

const (
	StatusValueCurrent StatusValue = iota
	StatusValueDeprecated
	StatusValueObsolete
	StatusValueMandatory
	StatusValueOptional
)

// IndexClause is an index clause (INDEX or PIB-INDEX).
type IndexClause interface {
	IndexClauseSpan() types.Span
	Indexes() []IndexItem
	indexClause()
}

// IndexClauseIndex is INDEX { ... }
type IndexClauseIndex struct {
	Items []IndexItem
	Span  types.Span
}

func (c *IndexClauseIndex) IndexClauseSpan() types.Span { return c.Span }
func (c *IndexClauseIndex) Indexes() []IndexItem        { return c.Items }
func (*IndexClauseIndex) indexClause()                  {}

// IndexClausePibIndex is PIB-INDEX { ... } (SPPI)
type IndexClausePibIndex struct {
	Items []IndexItem
	Span  types.Span
}

func (c *IndexClausePibIndex) IndexClauseSpan() types.Span { return c.Span }
func (c *IndexClausePibIndex) Indexes() []IndexItem        { return c.Items }
func (*IndexClausePibIndex) indexClause()                  {}

// IndexItem is an item in an INDEX clause.
type IndexItem struct {
	Implied bool
	Object  Ident
	Span    types.Span
}

// AugmentsClause is an AUGMENTS clause.
type AugmentsClause struct {
	Target Ident
	Span   types.Span
}

// DefValClause is a DEFVAL clause.
type DefValClause struct {
	Value DefValContent
	Span  types.Span
}

// DefValContent is the content of a DEFVAL clause.
type DefValContent interface {
	defValContent()
}

// DefValContentInteger is an integer value.
type DefValContentInteger struct {
	Value int64
}

func (*DefValContentInteger) defValContent() {}

// DefValContentUnsigned is an unsigned integer.
type DefValContentUnsigned struct {
	Value uint64
}

func (*DefValContentUnsigned) defValContent() {}

// DefValContentString is a quoted string.
type DefValContentString struct {
	Value QuotedString
}

func (*DefValContentString) defValContent() {}

// DefValContentIdentifier is an identifier (enum label or OID reference).
type DefValContentIdentifier struct {
	Name Ident
}

func (*DefValContentIdentifier) defValContent() {}

// DefValContentBits is a BITS value.
type DefValContentBits struct {
	Labels []Ident
	Span   types.Span
}

func (*DefValContentBits) defValContent() {}

// DefValContentHexString is a hex string.
type DefValContentHexString struct {
	Content string
	Span    types.Span
}

func (*DefValContentHexString) defValContent() {}

// DefValContentBinaryString is a binary string.
type DefValContentBinaryString struct {
	Content string
	Span    types.Span
}

func (*DefValContentBinaryString) defValContent() {}

// DefValContentObjectIdentifier is an object identifier value.
type DefValContentObjectIdentifier struct {
	Components []OidComponent
	Span       types.Span
}

func (*DefValContentObjectIdentifier) defValContent() {}

// RevisionClause is a REVISION clause in MODULE-IDENTITY.
type RevisionClause struct {
	Date        QuotedString
	Description QuotedString
	Span        types.Span
}
