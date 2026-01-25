package ast

import (
	"github.com/golangsnmp/gomib/internal/types"
)

// SyntaxClause is a SYNTAX clause specifying the type of an object.
type SyntaxClause struct {
	// Syntax is the type syntax.
	Syntax TypeSyntax
	// Span is the source location.
	Span types.Span
}

// NewSyntaxClause creates a new syntax clause.
func NewSyntaxClause(syntax TypeSyntax, span types.Span) SyntaxClause {
	return SyntaxClause{Syntax: syntax, Span: span}
}

// TypeSyntax is type syntax in a SYNTAX clause or type assignment.
type TypeSyntax interface {
	// SyntaxSpan returns the source location of this type syntax.
	SyntaxSpan() types.Span
	// ensure only this package can implement
	typeSyntax()
}

// TypeSyntaxTypeRef is a simple type reference: Integer32, DisplayString, IpAddress
type TypeSyntaxTypeRef struct {
	Name Ident
}

func (t *TypeSyntaxTypeRef) SyntaxSpan() types.Span { return t.Name.Span }
func (*TypeSyntaxTypeRef) typeSyntax()              {}

// TypeSyntaxIntegerEnum is INTEGER with named numbers: INTEGER { up(1), down(2) }
type TypeSyntaxIntegerEnum struct {
	// Base is the base type (usually nil, inferred as INTEGER).
	Base *Ident
	// NamedNumbers are the named number values.
	NamedNumbers []NamedNumber
	// Span is the source location.
	Span types.Span
}

func (t *TypeSyntaxIntegerEnum) SyntaxSpan() types.Span { return t.Span }
func (*TypeSyntaxIntegerEnum) typeSyntax()              {}

// TypeSyntaxBits is BITS with named bits: BITS { flag1(0), flag2(1) }
type TypeSyntaxBits struct {
	// NamedBits are the named bit positions.
	NamedBits []NamedNumber
	// Span is the source location.
	Span types.Span
}

func (t *TypeSyntaxBits) SyntaxSpan() types.Span { return t.Span }
func (*TypeSyntaxBits) typeSyntax()              {}

// TypeSyntaxConstrained is a constrained type: OCTET STRING (SIZE (0..255))
type TypeSyntaxConstrained struct {
	// Base is the base type.
	Base TypeSyntax
	// Constraint is the constraint.
	Constraint Constraint
	// Span is the source location.
	Span types.Span
}

func (t *TypeSyntaxConstrained) SyntaxSpan() types.Span { return t.Span }
func (*TypeSyntaxConstrained) typeSyntax()              {}

// TypeSyntaxSequenceOf is SEQUENCE OF: SEQUENCE OF IfEntry
type TypeSyntaxSequenceOf struct {
	// EntryType is the entry type name.
	EntryType Ident
	// Span is the source location.
	Span types.Span
}

func (t *TypeSyntaxSequenceOf) SyntaxSpan() types.Span { return t.Span }
func (*TypeSyntaxSequenceOf) typeSyntax()              {}

// TypeSyntaxSequence is SEQUENCE (row definition): SEQUENCE { ifIndex INTEGER, ... }
type TypeSyntaxSequence struct {
	// Fields are the sequence fields.
	Fields []SequenceField
	// Span is the source location.
	Span types.Span
}

func (t *TypeSyntaxSequence) SyntaxSpan() types.Span { return t.Span }
func (*TypeSyntaxSequence) typeSyntax()              {}

// TypeSyntaxChoice is a CHOICE type: CHOICE { simple SimpleSyntax, application ApplicationSyntax }
type TypeSyntaxChoice struct {
	// Alternatives are the choice alternatives.
	Alternatives []ChoiceAlternative
	// Span is the source location.
	Span types.Span
}

func (t *TypeSyntaxChoice) SyntaxSpan() types.Span { return t.Span }
func (*TypeSyntaxChoice) typeSyntax()              {}

// TypeSyntaxOctetString is OCTET STRING (explicit form).
type TypeSyntaxOctetString struct {
	// Span is the source location.
	Span types.Span
}

func (t *TypeSyntaxOctetString) SyntaxSpan() types.Span { return t.Span }
func (*TypeSyntaxOctetString) typeSyntax()              {}

// TypeSyntaxObjectIdentifier is OBJECT IDENTIFIER type.
type TypeSyntaxObjectIdentifier struct {
	// Span is the source location.
	Span types.Span
}

func (t *TypeSyntaxObjectIdentifier) SyntaxSpan() types.Span { return t.Span }
func (*TypeSyntaxObjectIdentifier) typeSyntax()              {}

// SequenceField is a field in a SEQUENCE definition.
type SequenceField struct {
	// Name is the field name.
	Name Ident
	// Syntax is the field type.
	Syntax TypeSyntax
	// Span is the source location.
	Span types.Span
}

// ChoiceAlternative is an alternative in a CHOICE type.
type ChoiceAlternative struct {
	// Name is the alternative name.
	Name Ident
	// Syntax is the alternative type.
	Syntax TypeSyntax
	// Span is the source location.
	Span types.Span
}

// Constraint is a type constraint (SIZE or range).
type Constraint interface {
	// ConstraintSpan returns the source location.
	ConstraintSpan() types.Span
	// constraint marker
	constraint()
}

// ConstraintSize is a SIZE constraint: (SIZE (0..255))
type ConstraintSize struct {
	// Ranges are the allowed ranges.
	Ranges []Range
	// Span is the source location.
	Span types.Span
}

func (c *ConstraintSize) ConstraintSpan() types.Span { return c.Span }
func (*ConstraintSize) constraint()                  {}

// ConstraintRange is a value range constraint: (0..65535)
type ConstraintRange struct {
	// Ranges are the allowed ranges.
	Ranges []Range
	// Span is the source location.
	Span types.Span
}

func (c *ConstraintRange) ConstraintSpan() types.Span { return c.Span }
func (*ConstraintRange) constraint()                  {}

// Range is a range in a constraint.
type Range struct {
	// Min is the minimum value.
	Min RangeValue
	// Max is the maximum value (nil for single value).
	Max RangeValue
	// Span is the source location.
	Span types.Span
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

// RangeValueIdent is a named value (MIN, MAX).
type RangeValueIdent struct {
	Name Ident
}

func (*RangeValueIdent) rangeValue() {}

// === Access clause ===

// AccessClause is an access clause (MAX-ACCESS or ACCESS).
type AccessClause struct {
	// Keyword is the keyword used (MAX-ACCESS vs ACCESS).
	Keyword AccessKeyword
	// Value is the access value.
	Value AccessValue
	// Span is the source location.
	Span types.Span
}

// AccessKeyword is the access keyword type.
type AccessKeyword int

const (
	// AccessKeywordAccess is SMIv1: ACCESS
	AccessKeywordAccess AccessKeyword = iota
	// AccessKeywordMaxAccess is SMIv2: MAX-ACCESS
	AccessKeywordMaxAccess
	// AccessKeywordMinAccess is SMIv2: MIN-ACCESS (in MODULE-COMPLIANCE)
	AccessKeywordMinAccess
	// AccessKeywordPibAccess is SPPI: PIB-ACCESS
	AccessKeywordPibAccess
)

// AccessValue is an access value.
type AccessValue int

const (
	// AccessValueReadOnly is read-only
	AccessValueReadOnly AccessValue = iota
	// AccessValueReadWrite is read-write
	AccessValueReadWrite
	// AccessValueReadCreate is read-create
	AccessValueReadCreate
	// AccessValueNotAccessible is not-accessible
	AccessValueNotAccessible
	// AccessValueAccessibleForNotify is accessible-for-notify
	AccessValueAccessibleForNotify
	// AccessValueWriteOnly is write-only (deprecated)
	AccessValueWriteOnly
	// AccessValueNotImplemented is not-implemented (AGENT-CAPABILITIES)
	AccessValueNotImplemented
	// SPPI-specific
	// AccessValueInstall is install
	AccessValueInstall
	// AccessValueInstallNotify is install-notify
	AccessValueInstallNotify
	// AccessValueReportOnly is report-only
	AccessValueReportOnly
)

// === Status clause ===

// StatusClause is a status clause.
type StatusClause struct {
	// Value is the status value.
	Value StatusValue
	// Span is the source location.
	Span types.Span
}

// StatusValue is a status value.
type StatusValue int

const (
	// StatusValueCurrent is current
	StatusValueCurrent StatusValue = iota
	// StatusValueDeprecated is deprecated
	StatusValueDeprecated
	// StatusValueObsolete is obsolete
	StatusValueObsolete
	// StatusValueMandatory is mandatory (SMIv1)
	StatusValueMandatory
	// StatusValueOptional is optional (SMIv1)
	StatusValueOptional
)

// === Index clause ===

// IndexClause is an index clause (INDEX or PIB-INDEX).
type IndexClause interface {
	// IndexClauseSpan returns the source location.
	IndexClauseSpan() types.Span
	// Indexes returns the index items.
	Indexes() []IndexItem
	// indexClause marker
	indexClause()
}

// IndexClauseIndex is INDEX { ifIndex, ipAddr IMPLIED }
type IndexClauseIndex struct {
	// Items are the index items.
	Items []IndexItem
	// Span is the source location.
	Span types.Span
}

func (c *IndexClauseIndex) IndexClauseSpan() types.Span { return c.Span }
func (c *IndexClauseIndex) Indexes() []IndexItem        { return c.Items }
func (*IndexClauseIndex) indexClause()                  {}

// IndexClausePibIndex is PIB-INDEX { ... } (SPPI)
type IndexClausePibIndex struct {
	// Items are the index items.
	Items []IndexItem
	// Span is the source location.
	Span types.Span
}

func (c *IndexClausePibIndex) IndexClauseSpan() types.Span { return c.Span }
func (c *IndexClausePibIndex) Indexes() []IndexItem        { return c.Items }
func (*IndexClausePibIndex) indexClause()                  {}

// IndexItem is an item in an INDEX clause.
type IndexItem struct {
	// Implied indicates whether this index is IMPLIED.
	Implied bool
	// Object is the object reference.
	Object Ident
	// Span is the source location.
	Span types.Span
}

// AugmentsClause is an AUGMENTS clause.
type AugmentsClause struct {
	// Target is the target row to augment.
	Target Ident
	// Span is the source location.
	Span types.Span
}

// === DEFVAL clause ===

// DefValClause is a DEFVAL clause.
type DefValClause struct {
	// Value is the default value content.
	Value DefValContent
	// Span is the source location (includes DEFVAL keyword and braces).
	Span types.Span
}

// DefValContent is the content of a DEFVAL clause.
//
// Per RFC 2578, DEFVAL can contain:
//   - Integer literals
//   - String literals (quoted)
//   - Enumeration labels (identifiers)
//   - BITS values (set of bit labels in braces)
//   - OID references (identifiers)
//   - Hex or binary strings
type DefValContent interface {
	// defValContent marker
	defValContent()
}

// DefValContentInteger is an integer value: DEFVAL { 0 }, DEFVAL { -1 }
type DefValContentInteger struct {
	Value int64
}

func (*DefValContentInteger) defValContent() {}

// DefValContentUnsigned is an unsigned integer (for Counter64 etc): DEFVAL { 4294967296 }
type DefValContentUnsigned struct {
	Value uint64
}

func (*DefValContentUnsigned) defValContent() {}

// DefValContentString is a quoted string: DEFVAL { "public" }, DEFVAL { "" }
type DefValContentString struct {
	Value QuotedString
}

func (*DefValContentString) defValContent() {}

// DefValContentIdentifier is an identifier (enum label or OID reference): DEFVAL { enabled }, DEFVAL { sysName }
type DefValContentIdentifier struct {
	Name Ident
}

func (*DefValContentIdentifier) defValContent() {}

// DefValContentBits is a BITS value (set of bit labels): DEFVAL { { flag1, flag2 } }, DEFVAL { {} }
type DefValContentBits struct {
	// Labels are the bit labels (identifiers).
	Labels []Ident
	// Span is the source location.
	Span types.Span
}

func (*DefValContentBits) defValContent() {}

// DefValContentHexString is a hex string: DEFVAL { 'FF00'H }
type DefValContentHexString struct {
	// Content is the hex content (without quotes and 'H' suffix).
	Content string
	// Span is the source location.
	Span types.Span
}

func (*DefValContentHexString) defValContent() {}

// DefValContentBinaryString is a binary string: DEFVAL { '1010'B }
type DefValContentBinaryString struct {
	// Content is the binary content (without quotes and 'B' suffix).
	Content string
	// Span is the source location.
	Span types.Span
}

func (*DefValContentBinaryString) defValContent() {}

// DefValContentObjectIdentifier is an object identifier value: DEFVAL { { iso 3 6 1 } }
type DefValContentObjectIdentifier struct {
	// Components are the OID components.
	Components []OidComponent
	// Span is the source location.
	Span types.Span
}

func (*DefValContentObjectIdentifier) defValContent() {}

// === REVISION clause ===

// RevisionClause is a REVISION clause in MODULE-IDENTITY.
type RevisionClause struct {
	// Date is the revision date.
	Date QuotedString
	// Description is the revision description.
	Description QuotedString
	// Span is the source location.
	Span types.Span
}
