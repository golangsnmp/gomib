package cst

import "github.com/golangsnmp/gomib/internal/types"

// SyntaxClauseNode is a SYNTAX clause containing a type expression.
type SyntaxClauseNode struct {
	Keyword SyntaxToken // SYNTAX keyword
	Syntax  TypeSyntaxNode
	Span    types.Span
}

// AccessClauseNode is a MAX-ACCESS, ACCESS, or MIN-ACCESS clause.
type AccessClauseNode struct {
	Keyword SyntaxToken // MAX-ACCESS, ACCESS, or MIN-ACCESS keyword
	Value   SyntaxToken // access value (read-only, read-write, etc.)
	Span    types.Span
}

// StatusClauseNode is a STATUS clause.
type StatusClauseNode struct {
	Keyword SyntaxToken // STATUS keyword
	Value   SyntaxToken // status value (current, deprecated, obsolete, mandatory, optional)
	Span    types.Span
}

// DescriptionClauseNode is a DESCRIPTION clause with a quoted string value.
type DescriptionClauseNode struct {
	Keyword SyntaxToken // DESCRIPTION keyword
	Value   SyntaxToken // quoted string
	Span    types.Span
}

// ReferenceClauseNode is a REFERENCE clause with a quoted string value.
type ReferenceClauseNode struct {
	Keyword SyntaxToken // REFERENCE keyword
	Value   SyntaxToken // quoted string
	Span    types.Span
}

// UnitsClauseNode is a UNITS clause with a quoted string value.
type UnitsClauseNode struct {
	Keyword SyntaxToken // UNITS keyword
	Value   SyntaxToken // quoted string
	Span    types.Span
}

// DisplayHintClauseNode is a DISPLAY-HINT clause with a quoted string value.
type DisplayHintClauseNode struct {
	Keyword SyntaxToken // DISPLAY-HINT keyword
	Value   SyntaxToken // quoted string
	Span    types.Span
}

// IndexClauseNode is an INDEX clause with a braced list of index items.
type IndexClauseNode struct {
	Keyword SyntaxToken // INDEX keyword
	LBrace  SyntaxToken
	Items   []IndexItemNode
	Commas  []SyntaxToken
	RBrace  SyntaxToken
	Span    types.Span
}

// IndexItemNode is a single entry in an INDEX clause, with an optional IMPLIED prefix.
type IndexItemNode struct {
	Implied       SyntaxToken // IMPLIED keyword (zero if absent)
	Object        SyntaxToken // object name (OCTET for OCTET STRING)
	StringKeyword SyntaxToken // STRING keyword (zero unless Object is OCTET)
	Span          types.Span
}

// AugmentsClauseNode is an AUGMENTS clause referencing a table row.
type AugmentsClauseNode struct {
	Keyword SyntaxToken // AUGMENTS keyword
	LBrace  SyntaxToken
	Target  SyntaxToken // target table entry name
	RBrace  SyntaxToken
	Span    types.Span
}

// DefValClauseNode is a DEFVAL clause. The content between braces is stored
// as opaque tokens since DEFVAL values have varied forms.
type DefValClauseNode struct {
	Keyword SyntaxToken // DEFVAL keyword
	LBrace  SyntaxToken
	Content []SyntaxToken
	RBrace  SyntaxToken
	Span    types.Span
}

// ObjectsClauseNode is an OBJECTS clause with a braced list of object names.
type ObjectsClauseNode struct {
	Keyword SyntaxToken // OBJECTS keyword
	LBrace  SyntaxToken
	Names   []SyntaxToken
	Commas  []SyntaxToken
	RBrace  SyntaxToken
	Span    types.Span
}

// NotificationsClauseNode is a NOTIFICATIONS clause with a braced list of names.
type NotificationsClauseNode struct {
	Keyword SyntaxToken // NOTIFICATIONS keyword
	LBrace  SyntaxToken
	Names   []SyntaxToken
	Commas  []SyntaxToken
	RBrace  SyntaxToken
	Span    types.Span
}

// RevisionNode is a REVISION clause within a MODULE-IDENTITY.
type RevisionNode struct {
	Keyword     SyntaxToken // REVISION keyword
	Date        SyntaxToken // quoted date string
	Description *DescriptionClauseNode
	Span        types.Span
}

// LastUpdatedClauseNode is a LAST-UPDATED clause.
type LastUpdatedClauseNode struct {
	Keyword SyntaxToken // LAST-UPDATED keyword
	Value   SyntaxToken // quoted date string
	Span    types.Span
}

// OrganizationClauseNode is an ORGANIZATION clause.
type OrganizationClauseNode struct {
	Keyword SyntaxToken // ORGANIZATION keyword
	Value   SyntaxToken // quoted string
	Span    types.Span
}

// ContactInfoClauseNode is a CONTACT-INFO clause.
type ContactInfoClauseNode struct {
	Keyword SyntaxToken // CONTACT-INFO keyword
	Value   SyntaxToken // quoted string
	Span    types.Span
}

// EnterpriseClauseNode is an ENTERPRISE clause in TRAP-TYPE.
type EnterpriseClauseNode struct {
	Keyword SyntaxToken // ENTERPRISE keyword
	Name    SyntaxToken // enterprise name
	Span    types.Span
}

// VariablesClauseNode is a VARIABLES clause in TRAP-TYPE with a braced list of names.
type VariablesClauseNode struct {
	Keyword SyntaxToken // VARIABLES keyword
	LBrace  SyntaxToken
	Names   []SyntaxToken
	Commas  []SyntaxToken
	RBrace  SyntaxToken
	Span    types.Span
}

// ProductReleaseClauseNode is a PRODUCT-RELEASE clause.
type ProductReleaseClauseNode struct {
	Keyword SyntaxToken // PRODUCT-RELEASE keyword
	Value   SyntaxToken // quoted string
	Span    types.Span
}
