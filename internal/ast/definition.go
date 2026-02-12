package ast

import (
	"github.com/golangsnmp/gomib/internal/types"
)

// Definition is a top-level construct in a MIB module body.
type Definition interface {
	DefinitionName() *Ident
	DefinitionSpan() types.Span
	definition()
}

// ObjectTypeDef represents an OBJECT-TYPE macro invocation (SMIv1/v2).
type ObjectTypeDef struct {
	Name          Ident
	Syntax        SyntaxClause
	Units         *QuotedString
	Access        AccessClause
	Status        *StatusClause
	Description   *QuotedString
	Reference     *QuotedString
	Index         IndexClause
	Augments      *AugmentsClause
	DefVal        *DefValClause
	OidAssignment OidAssignment
	Span          types.Span
}

func (d *ObjectTypeDef) DefinitionName() *Ident     { return &d.Name }
func (d *ObjectTypeDef) DefinitionSpan() types.Span { return d.Span }
func (*ObjectTypeDef) definition()                  {}

// ModuleIdentityDef represents a MODULE-IDENTITY macro invocation (SMIv2).
type ModuleIdentityDef struct {
	Name          Ident
	LastUpdated   QuotedString
	Organization  QuotedString
	ContactInfo   QuotedString
	Description   QuotedString
	Revisions     []RevisionClause
	OidAssignment OidAssignment
	Span          types.Span
}

func (d *ModuleIdentityDef) DefinitionName() *Ident     { return &d.Name }
func (d *ModuleIdentityDef) DefinitionSpan() types.Span { return d.Span }
func (*ModuleIdentityDef) definition()                  {}

// ObjectIdentityDef represents an OBJECT-IDENTITY macro invocation (SMIv2).
type ObjectIdentityDef struct {
	Name          Ident
	Status        StatusClause
	Description   QuotedString
	Reference     *QuotedString
	OidAssignment OidAssignment
	Span          types.Span
}

func (d *ObjectIdentityDef) DefinitionName() *Ident     { return &d.Name }
func (d *ObjectIdentityDef) DefinitionSpan() types.Span { return d.Span }
func (*ObjectIdentityDef) definition()                  {}

// NotificationTypeDef represents a NOTIFICATION-TYPE macro invocation (SMIv2).
type NotificationTypeDef struct {
	Name          Ident
	Objects       []Ident
	Status        StatusClause
	Description   QuotedString
	Reference     *QuotedString
	OidAssignment OidAssignment
	Span          types.Span
}

func (d *NotificationTypeDef) DefinitionName() *Ident     { return &d.Name }
func (d *NotificationTypeDef) DefinitionSpan() types.Span { return d.Span }
func (*NotificationTypeDef) definition()                  {}

// TrapTypeDef represents a TRAP-TYPE macro invocation (SMIv1).
type TrapTypeDef struct {
	Name        Ident
	Enterprise  Ident
	Variables   []Ident
	Description *QuotedString
	Reference   *QuotedString
	TrapNumber  uint32
	Span        types.Span
}

func (d *TrapTypeDef) DefinitionName() *Ident     { return &d.Name }
func (d *TrapTypeDef) DefinitionSpan() types.Span { return d.Span }
func (*TrapTypeDef) definition()                  {}

// TextualConventionDef represents a TEXTUAL-CONVENTION definition (SMIv2).
type TextualConventionDef struct {
	Name        Ident
	DisplayHint *QuotedString
	Status      StatusClause
	Description QuotedString
	Reference   *QuotedString
	Syntax      SyntaxClause
	Span        types.Span
}

func (d *TextualConventionDef) DefinitionName() *Ident     { return &d.Name }
func (d *TextualConventionDef) DefinitionSpan() types.Span { return d.Span }
func (*TextualConventionDef) definition()                  {}

// TypeAssignmentDef represents a type assignment (TypeName ::= TypeSyntax).
type TypeAssignmentDef struct {
	Name   Ident
	Syntax TypeSyntax
	Span   types.Span
}

func (d *TypeAssignmentDef) DefinitionName() *Ident     { return &d.Name }
func (d *TypeAssignmentDef) DefinitionSpan() types.Span { return d.Span }
func (*TypeAssignmentDef) definition()                  {}

// ValueAssignmentDef represents an OID value assignment
// (name OBJECT IDENTIFIER ::= { ... }).
type ValueAssignmentDef struct {
	Name          Ident
	OidAssignment OidAssignment
	Span          types.Span
}

func (d *ValueAssignmentDef) DefinitionName() *Ident     { return &d.Name }
func (d *ValueAssignmentDef) DefinitionSpan() types.Span { return d.Span }
func (*ValueAssignmentDef) definition()                  {}

// ObjectGroupDef represents an OBJECT-GROUP macro invocation (SMIv2).
type ObjectGroupDef struct {
	Name          Ident
	Objects       []Ident
	Status        StatusClause
	Description   QuotedString
	Reference     *QuotedString
	OidAssignment OidAssignment
	Span          types.Span
}

func (d *ObjectGroupDef) DefinitionName() *Ident     { return &d.Name }
func (d *ObjectGroupDef) DefinitionSpan() types.Span { return d.Span }
func (*ObjectGroupDef) definition()                  {}

// NotificationGroupDef represents a NOTIFICATION-GROUP macro invocation (SMIv2).
type NotificationGroupDef struct {
	Name          Ident
	Notifications []Ident
	Status        StatusClause
	Description   QuotedString
	Reference     *QuotedString
	OidAssignment OidAssignment
	Span          types.Span
}

func (d *NotificationGroupDef) DefinitionName() *Ident     { return &d.Name }
func (d *NotificationGroupDef) DefinitionSpan() types.Span { return d.Span }
func (*NotificationGroupDef) definition()                  {}

// ModuleComplianceDef represents a MODULE-COMPLIANCE macro invocation (SMIv2).
type ModuleComplianceDef struct {
	Name          Ident
	Status        StatusClause
	Description   QuotedString
	Reference     *QuotedString
	Modules       []ComplianceModule
	OidAssignment OidAssignment
	Span          types.Span
}

func (d *ModuleComplianceDef) DefinitionName() *Ident     { return &d.Name }
func (d *ModuleComplianceDef) DefinitionSpan() types.Span { return d.Span }
func (*ModuleComplianceDef) definition()                  {}

// ComplianceModule represents a MODULE clause within MODULE-COMPLIANCE.
type ComplianceModule struct {
	ModuleName      *Ident
	ModuleOid       *OidAssignment
	MandatoryGroups []Ident
	Compliances     []Compliance
	Span            types.Span
}

// Compliance represents a GROUP or OBJECT refinement in MODULE-COMPLIANCE.
type Compliance interface {
	compliance()
}

// ComplianceGroup represents a GROUP clause within MODULE-COMPLIANCE.
type ComplianceGroup struct {
	Group       Ident
	Description QuotedString
	Span        types.Span
}

func (*ComplianceGroup) compliance() {}

// ComplianceObject represents an OBJECT refinement within MODULE-COMPLIANCE.
type ComplianceObject struct {
	Object      Ident
	Syntax      *SyntaxClause
	WriteSyntax *SyntaxClause
	MinAccess   *AccessClause
	Description QuotedString
	Span        types.Span
}

func (*ComplianceObject) compliance() {}

// AgentCapabilitiesDef represents an AGENT-CAPABILITIES macro invocation (SMIv2).
type AgentCapabilitiesDef struct {
	Name           Ident
	ProductRelease QuotedString
	Status         StatusClause
	Description    QuotedString
	Reference      *QuotedString
	Supports       []SupportsModule
	OidAssignment  OidAssignment
	Span           types.Span
}

func (d *AgentCapabilitiesDef) DefinitionName() *Ident     { return &d.Name }
func (d *AgentCapabilitiesDef) DefinitionSpan() types.Span { return d.Span }
func (*AgentCapabilitiesDef) definition()                  {}

// SupportsModule represents a SUPPORTS clause within AGENT-CAPABILITIES.
type SupportsModule struct {
	ModuleName Ident
	ModuleOid  *OidAssignment
	Includes   []Ident
	Variations []Variation
	Span       types.Span
}

// Variation represents a VARIATION clause within AGENT-CAPABILITIES.
type Variation interface {
	variation()
}

// ObjectVariation represents an object VARIATION within AGENT-CAPABILITIES.
type ObjectVariation struct {
	Object           Ident
	Syntax           *SyntaxClause
	WriteSyntax      *SyntaxClause
	Access           *AccessClause
	CreationRequires []Ident
	DefVal           *DefValClause
	Description      QuotedString
	Span             types.Span
}

func (*ObjectVariation) variation() {}

// NotificationVariation represents a notification VARIATION within
// AGENT-CAPABILITIES.
type NotificationVariation struct {
	Notification Ident
	Access       *AccessClause
	Description  QuotedString
	Span         types.Span
}

func (*NotificationVariation) variation() {}

// MacroDefinitionDef represents a MACRO definition whose body is skipped.
type MacroDefinitionDef struct {
	Name Ident
	Span types.Span
}

func (d *MacroDefinitionDef) DefinitionName() *Ident     { return &d.Name }
func (d *MacroDefinitionDef) DefinitionSpan() types.Span { return d.Span }
func (*MacroDefinitionDef) definition()                  {}

// ErrorDef records a parse error from which the parser recovered.
type ErrorDef struct {
	Message string
	Span    types.Span
}

func (d *ErrorDef) DefinitionName() *Ident     { return nil }
func (d *ErrorDef) DefinitionSpan() types.Span { return d.Span }
func (*ErrorDef) definition()                  {}
