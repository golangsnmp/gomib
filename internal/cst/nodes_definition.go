package cst

import "github.com/golangsnmp/gomib/internal/types"

// ObjectTypeNode is an OBJECT-TYPE definition.
type ObjectTypeNode struct {
	Name        SyntaxToken
	Keyword     SyntaxToken // OBJECT-TYPE
	Syntax      *SyntaxClauseNode
	Units       *UnitsClauseNode
	Access      *AccessClauseNode
	Status      *StatusClauseNode
	Description *DescriptionClauseNode
	Reference   *ReferenceClauseNode
	Index       *IndexClauseNode
	Augments    *AugmentsClauseNode
	DefVal      *DefValClauseNode
	Assign      SyntaxToken // ::=
	Oid         *OidValueNode
	Span        types.Span
}

func (*ObjectTypeNode) definitionNode()              {}
func (n *ObjectTypeNode) DefinitionSpan() types.Span { return n.Span }

// ModuleIdentityNode is a MODULE-IDENTITY definition.
type ModuleIdentityNode struct {
	Name         SyntaxToken
	Keyword      SyntaxToken // MODULE-IDENTITY
	LastUpdated  *LastUpdatedClauseNode
	Organization *OrganizationClauseNode
	ContactInfo  *ContactInfoClauseNode
	Description  *DescriptionClauseNode
	Revisions    []RevisionNode
	Assign       SyntaxToken // ::=
	Oid          *OidValueNode
	Span         types.Span
}

func (*ModuleIdentityNode) definitionNode()              {}
func (n *ModuleIdentityNode) DefinitionSpan() types.Span { return n.Span }

// ObjectIdentityNode is an OBJECT-IDENTITY definition.
type ObjectIdentityNode struct {
	Name        SyntaxToken
	Keyword     SyntaxToken // OBJECT-IDENTITY
	Status      *StatusClauseNode
	Description *DescriptionClauseNode
	Reference   *ReferenceClauseNode
	Assign      SyntaxToken // ::=
	Oid         *OidValueNode
	Span        types.Span
}

func (*ObjectIdentityNode) definitionNode()              {}
func (n *ObjectIdentityNode) DefinitionSpan() types.Span { return n.Span }

// NotificationTypeNode is a NOTIFICATION-TYPE definition.
type NotificationTypeNode struct {
	Name        SyntaxToken
	Keyword     SyntaxToken // NOTIFICATION-TYPE
	Objects     *ObjectsClauseNode
	Status      *StatusClauseNode
	Description *DescriptionClauseNode
	Reference   *ReferenceClauseNode
	Assign      SyntaxToken // ::=
	Oid         *OidValueNode
	Span        types.Span
}

func (*NotificationTypeNode) definitionNode()              {}
func (n *NotificationTypeNode) DefinitionSpan() types.Span { return n.Span }

// TrapTypeNode is an SMIv1 TRAP-TYPE definition.
type TrapTypeNode struct {
	Name        SyntaxToken
	Keyword     SyntaxToken // TRAP-TYPE
	Enterprise  *EnterpriseClauseNode
	Variables   *VariablesClauseNode
	Description *DescriptionClauseNode
	Reference   *ReferenceClauseNode
	Assign      SyntaxToken // ::=
	Value       SyntaxToken // trap number
	Span        types.Span
}

func (*TrapTypeNode) definitionNode()              {}
func (n *TrapTypeNode) DefinitionSpan() types.Span { return n.Span }

// TextualConventionNode is a TEXTUAL-CONVENTION definition.
type TextualConventionNode struct {
	Name        SyntaxToken
	Assign      SyntaxToken // ::= (optional, may be zero)
	Keyword     SyntaxToken // TEXTUAL-CONVENTION
	DisplayHint *DisplayHintClauseNode
	Status      *StatusClauseNode
	Description *DescriptionClauseNode
	Reference   *ReferenceClauseNode
	Syntax      *SyntaxClauseNode
	Span        types.Span
}

func (*TextualConventionNode) definitionNode()              {}
func (n *TextualConventionNode) DefinitionSpan() types.Span { return n.Span }

// TypeAssignmentNode is a plain type assignment (Name ::= Type).
type TypeAssignmentNode struct {
	Name   SyntaxToken
	Assign SyntaxToken // ::=
	Syntax TypeSyntaxNode
	Span   types.Span
}

func (*TypeAssignmentNode) definitionNode()              {}
func (n *TypeAssignmentNode) DefinitionSpan() types.Span { return n.Span }

// ValueAssignmentNode is an OID value assignment (name OBJECT IDENTIFIER ::= { ... }).
type ValueAssignmentNode struct {
	Name       SyntaxToken
	Object     SyntaxToken // OBJECT keyword
	Identifier SyntaxToken // IDENTIFIER keyword
	Assign     SyntaxToken // ::=
	Oid        *OidValueNode
	Span       types.Span
}

func (*ValueAssignmentNode) definitionNode()              {}
func (n *ValueAssignmentNode) DefinitionSpan() types.Span { return n.Span }

// ObjectGroupNode is an OBJECT-GROUP definition.
type ObjectGroupNode struct {
	Name        SyntaxToken
	Keyword     SyntaxToken // OBJECT-GROUP
	Objects     *ObjectsClauseNode
	Status      *StatusClauseNode
	Description *DescriptionClauseNode
	Reference   *ReferenceClauseNode
	Assign      SyntaxToken // ::=
	Oid         *OidValueNode
	Span        types.Span
}

func (*ObjectGroupNode) definitionNode()              {}
func (n *ObjectGroupNode) DefinitionSpan() types.Span { return n.Span }

// NotificationGroupNode is a NOTIFICATION-GROUP definition.
type NotificationGroupNode struct {
	Name          SyntaxToken
	Keyword       SyntaxToken // NOTIFICATION-GROUP
	Notifications *NotificationsClauseNode
	Status        *StatusClauseNode
	Description   *DescriptionClauseNode
	Reference     *ReferenceClauseNode
	Assign        SyntaxToken // ::=
	Oid           *OidValueNode
	Span          types.Span
}

func (*NotificationGroupNode) definitionNode()              {}
func (n *NotificationGroupNode) DefinitionSpan() types.Span { return n.Span }

// ModuleComplianceNode is a MODULE-COMPLIANCE definition.
type ModuleComplianceNode struct {
	Name        SyntaxToken
	Keyword     SyntaxToken // MODULE-COMPLIANCE
	Status      *StatusClauseNode
	Description *DescriptionClauseNode
	Reference   *ReferenceClauseNode
	Modules     []ComplianceModuleNode
	Assign      SyntaxToken // ::=
	Oid         *OidValueNode
	Span        types.Span
}

func (*ModuleComplianceNode) definitionNode()              {}
func (n *ModuleComplianceNode) DefinitionSpan() types.Span { return n.Span }

// AgentCapabilitiesNode is an AGENT-CAPABILITIES definition.
type AgentCapabilitiesNode struct {
	Name           SyntaxToken
	Keyword        SyntaxToken // AGENT-CAPABILITIES
	ProductRelease *ProductReleaseClauseNode
	Status         *StatusClauseNode
	Description    *DescriptionClauseNode
	Reference      *ReferenceClauseNode
	Modules        []CapabilityModuleNode
	Assign         SyntaxToken // ::=
	Oid            *OidValueNode
	Span           types.Span
}

func (*AgentCapabilitiesNode) definitionNode()              {}
func (n *AgentCapabilitiesNode) DefinitionSpan() types.Span { return n.Span }

// MacroDefinitionNode is a MACRO definition. The body between BEGIN and END
// is treated as opaque tokens since macro bodies are not interpreted.
type MacroDefinitionNode struct {
	Name   SyntaxToken
	Macro  SyntaxToken // MACRO keyword
	Assign SyntaxToken // ::=
	Begin  SyntaxToken // BEGIN keyword
	Body   []SyntaxToken
	End    SyntaxToken // END keyword
	Span   types.Span
}

func (*MacroDefinitionNode) definitionNode()              {}
func (n *MacroDefinitionNode) DefinitionSpan() types.Span { return n.Span }
