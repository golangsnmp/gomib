package cst

import "github.com/golangsnmp/gomib/internal/types"

// ComplianceModuleNode is a MODULE clause within MODULE-COMPLIANCE.
type ComplianceModuleNode struct {
	Keyword         SyntaxToken // MODULE keyword
	ModuleName      SyntaxToken // module name (zero if referring to current module)
	Oid             *OidValueNode
	MandatoryGroups *MandatoryGroupsNode
	Groups          []ComplianceGroupNode
	Objects         []ComplianceObjectNode
	Span            types.Span
}

// MandatoryGroupsNode is a MANDATORY-GROUPS clause with a braced list of group names.
type MandatoryGroupsNode struct {
	Keyword SyntaxToken // MANDATORY-GROUPS keyword
	LBrace  SyntaxToken
	Names   []SyntaxToken
	Commas  []SyntaxToken
	RBrace  SyntaxToken
	Span    types.Span
}

// ComplianceGroupNode is a GROUP clause within MODULE-COMPLIANCE.
type ComplianceGroupNode struct {
	Keyword     SyntaxToken // GROUP keyword
	Name        SyntaxToken // group name
	Description *DescriptionClauseNode
	Span        types.Span
}

// ComplianceObjectNode is an OBJECT refinement within MODULE-COMPLIANCE.
type ComplianceObjectNode struct {
	Keyword     SyntaxToken // OBJECT keyword
	Name        SyntaxToken // object name
	Syntax      *SyntaxClauseNode
	WriteSyntax *WriteSyntaxClauseNode
	Access      *AccessClauseNode
	Description *DescriptionClauseNode
	Span        types.Span
}

// WriteSyntaxClauseNode is a WRITE-SYNTAX clause in MODULE-COMPLIANCE or
// AGENT-CAPABILITIES.
type WriteSyntaxClauseNode struct {
	Keyword SyntaxToken // WRITE-SYNTAX keyword
	Syntax  TypeSyntaxNode
	Span    types.Span
}

// CapabilityModuleNode is a SUPPORTS clause within AGENT-CAPABILITIES.
type CapabilityModuleNode struct {
	Keyword    SyntaxToken // SUPPORTS keyword
	ModuleName SyntaxToken // module name
	Oid        *OidValueNode
	Includes   *IncludesNode
	Variations []CapabilityVariationNode
	Span       types.Span
}

// IncludesNode is an INCLUDES clause with a braced list of group names.
type IncludesNode struct {
	Keyword SyntaxToken // INCLUDES keyword
	LBrace  SyntaxToken
	Names   []SyntaxToken
	Commas  []SyntaxToken
	RBrace  SyntaxToken
	Span    types.Span
}

// CapabilityVariationNode is a VARIATION clause within AGENT-CAPABILITIES.
type CapabilityVariationNode struct {
	Keyword          SyntaxToken // VARIATION keyword
	Name             SyntaxToken // object or notification name
	Syntax           *SyntaxClauseNode
	WriteSyntax      *WriteSyntaxClauseNode
	Access           *AccessClauseNode
	CreationRequires *CreationRequiresNode
	DefVal           *DefValClauseNode
	Description      *DescriptionClauseNode
	Span             types.Span
}

// CreationRequiresNode is a CREATION-REQUIRES clause with a braced list of names.
type CreationRequiresNode struct {
	Keyword SyntaxToken // CREATION-REQUIRES keyword
	LBrace  SyntaxToken
	Names   []SyntaxToken
	Commas  []SyntaxToken
	RBrace  SyntaxToken
	Span    types.Span
}
