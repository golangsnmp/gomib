package ast

import (
	"github.com/golangsnmp/gomib/internal/types"
)

// SyntaxClause wraps a TypeSyntax with its source span.
type SyntaxClause struct {
	Syntax TypeSyntax
	Span   types.Span
}

// NewSyntaxClause creates a SyntaxClause from a type syntax and span.
func NewSyntaxClause(syntax TypeSyntax, span types.Span) SyntaxClause {
	return SyntaxClause{Syntax: syntax, Span: span}
}

// TypeSyntax represents a type expression in a SYNTAX clause or
// type assignment.
type TypeSyntax interface {
	SyntaxSpan() types.Span
	typeSyntax()
}

// TypeSyntaxTypeRef is an unqualified type name reference.
type TypeSyntaxTypeRef struct {
	Name Ident
}

func (t *TypeSyntaxTypeRef) SyntaxSpan() types.Span { return t.Name.Span }
func (*TypeSyntaxTypeRef) typeSyntax()              {}

// TypeSyntaxIntegerEnum is an INTEGER type with enumerated named values.
type TypeSyntaxIntegerEnum struct {
	Base         *Ident
	NamedNumbers []NamedNumber
	Span         types.Span
}

func (t *TypeSyntaxIntegerEnum) SyntaxSpan() types.Span { return t.Span }
func (*TypeSyntaxIntegerEnum) typeSyntax()              {}

// TypeSyntaxBits is a BITS type with named bit positions.
type TypeSyntaxBits struct {
	NamedBits []NamedNumber
	Span      types.Span
}

func (t *TypeSyntaxBits) SyntaxSpan() types.Span { return t.Span }
func (*TypeSyntaxBits) typeSyntax()              {}

// TypeSyntaxConstrained is a type with a SIZE or range constraint.
type TypeSyntaxConstrained struct {
	Base       TypeSyntax
	Constraint Constraint
	Span       types.Span
}

func (t *TypeSyntaxConstrained) SyntaxSpan() types.Span { return t.Span }
func (*TypeSyntaxConstrained) typeSyntax()              {}

// TypeSyntaxSequenceOf is a SEQUENCE OF entry-type reference.
type TypeSyntaxSequenceOf struct {
	EntryType Ident
	Span      types.Span
}

func (t *TypeSyntaxSequenceOf) SyntaxSpan() types.Span { return t.Span }
func (*TypeSyntaxSequenceOf) typeSyntax()              {}

// TypeSyntaxSequence is a SEQUENCE with named fields (table row definition).
type TypeSyntaxSequence struct {
	Fields []SequenceField
	Span   types.Span
}

func (t *TypeSyntaxSequence) SyntaxSpan() types.Span { return t.Span }
func (*TypeSyntaxSequence) typeSyntax()              {}

// TypeSyntaxChoice is a CHOICE type with named alternatives.
type TypeSyntaxChoice struct {
	Alternatives []ChoiceAlternative
	Span         types.Span
}

func (t *TypeSyntaxChoice) SyntaxSpan() types.Span { return t.Span }
func (*TypeSyntaxChoice) typeSyntax()              {}

// TypeSyntaxOctetString is the explicit OCTET STRING type.
type TypeSyntaxOctetString struct {
	Span types.Span
}

func (t *TypeSyntaxOctetString) SyntaxSpan() types.Span { return t.Span }
func (*TypeSyntaxOctetString) typeSyntax()              {}

// TypeSyntaxObjectIdentifier is the OBJECT IDENTIFIER type.
type TypeSyntaxObjectIdentifier struct {
	Span types.Span
}

func (t *TypeSyntaxObjectIdentifier) SyntaxSpan() types.Span { return t.Span }
func (*TypeSyntaxObjectIdentifier) typeSyntax()              {}

// SequenceField is a named field within a SEQUENCE definition.
type SequenceField struct {
	Name   Ident
	Syntax TypeSyntax
	Span   types.Span
}

// ChoiceAlternative is a named alternative within a CHOICE type.
type ChoiceAlternative struct {
	Name   Ident
	Syntax TypeSyntax
	Span   types.Span
}

// Constraint represents a type sub-typing constraint (SIZE or range).
type Constraint interface {
	ConstraintSpan() types.Span
	constraint()
}

// ConstraintSize is a SIZE(...) constraint on length.
type ConstraintSize struct {
	Ranges []Range
	Span   types.Span
}

func (c *ConstraintSize) ConstraintSpan() types.Span { return c.Span }
func (*ConstraintSize) constraint()                  {}

// ConstraintRange is a value range constraint, e.g. (0..65535).
type ConstraintRange struct {
	Ranges []Range
	Span   types.Span
}

func (c *ConstraintRange) ConstraintSpan() types.Span { return c.Span }
func (*ConstraintRange) constraint()                  {}

// Range is a single range element within a constraint (min..max).
type Range struct {
	Min  RangeValue
	Max  RangeValue
	Span types.Span
}

// RangeValue is an endpoint in a range (numeric literal or MIN/MAX).
type RangeValue interface {
	rangeValue()
}

// RangeValueSigned is a signed integer range endpoint.
type RangeValueSigned struct {
	Value int64
}

func (*RangeValueSigned) rangeValue() {}

// RangeValueUnsigned is an unsigned integer range endpoint.
type RangeValueUnsigned struct {
	Value uint64
}

func (*RangeValueUnsigned) rangeValue() {}

// RangeValueIdent is a symbolic range endpoint (MIN or MAX).
type RangeValueIdent struct {
	Name Ident
}

func (*RangeValueIdent) rangeValue() {}

// AccessClause holds a parsed ACCESS, MAX-ACCESS, or MIN-ACCESS clause.
type AccessClause struct {
	Keyword AccessKeyword
	Value   AccessValue
	Span    types.Span
}

// AccessKeyword distinguishes ACCESS, MAX-ACCESS, and MIN-ACCESS.
type AccessKeyword int

const (
	AccessKeywordAccess AccessKeyword = iota
	AccessKeywordMaxAccess
	AccessKeywordMinAccess
	AccessKeywordPibAccess
)

// AccessValue enumerates the possible access levels for MIB objects.
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

// StatusClause holds a parsed STATUS clause value and span.
type StatusClause struct {
	Value StatusValue
	Span  types.Span
}

// StatusValue enumerates the possible status values for MIB objects.
type StatusValue int

const (
	StatusValueCurrent StatusValue = iota
	StatusValueDeprecated
	StatusValueObsolete
	StatusValueMandatory
	StatusValueOptional
)

// IndexClause represents an INDEX or PIB-INDEX clause in OBJECT-TYPE.
type IndexClause interface {
	IndexClauseSpan() types.Span
	Indexes() []IndexItem
	indexClause()
}

// IndexClauseIndex is an INDEX { ... } clause.
type IndexClauseIndex struct {
	Items []IndexItem
	Span  types.Span
}

func (c *IndexClauseIndex) IndexClauseSpan() types.Span { return c.Span }
func (c *IndexClauseIndex) Indexes() []IndexItem        { return c.Items }
func (*IndexClauseIndex) indexClause()                  {}

// IndexClausePibIndex is a PIB-INDEX { ... } clause (SPPI).
type IndexClausePibIndex struct {
	Items []IndexItem
	Span  types.Span
}

func (c *IndexClausePibIndex) IndexClauseSpan() types.Span { return c.Span }
func (c *IndexClausePibIndex) Indexes() []IndexItem        { return c.Items }
func (*IndexClausePibIndex) indexClause()                  {}

// IndexItem is a single entry in an INDEX clause, possibly IMPLIED.
type IndexItem struct {
	Implied bool
	Object  Ident
	Span    types.Span
}

// AugmentsClause holds the target row referenced by AUGMENTS.
type AugmentsClause struct {
	Target Ident
	Span   types.Span
}

// DefValClause holds the default value for an OBJECT-TYPE.
type DefValClause struct {
	Value DefValContent
	Span  types.Span
}

// DefValContent represents the typed content within a DEFVAL { ... } clause.
type DefValContent interface {
	defValContent()
}

// DefValContentInteger is a signed integer default value.
type DefValContentInteger struct {
	Value int64
}

func (*DefValContentInteger) defValContent() {}

// DefValContentUnsigned is an unsigned integer default value.
type DefValContentUnsigned struct {
	Value uint64
}

func (*DefValContentUnsigned) defValContent() {}

// DefValContentString is a quoted string default value.
type DefValContentString struct {
	Value QuotedString
}

func (*DefValContentString) defValContent() {}

// DefValContentIdentifier is a named default (enum label or OID reference).
type DefValContentIdentifier struct {
	Name Ident
}

func (*DefValContentIdentifier) defValContent() {}

// DefValContentBits is a BITS default value with named bit labels.
type DefValContentBits struct {
	Labels []Ident
	Span   types.Span
}

func (*DefValContentBits) defValContent() {}

// DefValContentHexString is a hex string default value ('...'H).
type DefValContentHexString struct {
	Content string
	Span    types.Span
}

func (*DefValContentHexString) defValContent() {}

// DefValContentBinaryString is a binary string default value ('...'B).
type DefValContentBinaryString struct {
	Content string
	Span    types.Span
}

func (*DefValContentBinaryString) defValContent() {}

// DefValContentObjectIdentifier is an OID default value.
type DefValContentObjectIdentifier struct {
	Components []OidComponent
	Span       types.Span
}

func (*DefValContentObjectIdentifier) defValContent() {}

// DefValContentUnparsed represents DEFVAL content that could not be parsed.
// Used by error recovery when the parser skips over unrecognized content.
type DefValContentUnparsed struct {
	Span types.Span
}

func (*DefValContentUnparsed) defValContent() {}

// RevisionClause represents a REVISION clause within MODULE-IDENTITY.
type RevisionClause struct {
	Date        QuotedString
	Description QuotedString
	Span        types.Span
}
