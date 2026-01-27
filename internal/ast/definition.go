package ast

import (
	"github.com/golangsnmp/gomib/internal/types"
)

// Definition is a definition in a MIB module body.
type Definition interface {
	DefinitionName() *Ident
	DefinitionSpan() types.Span
	definition()
}

// ObjectTypeDef is an OBJECT-TYPE definition.
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

// ModuleIdentityDef is a MODULE-IDENTITY definition.
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

// ObjectIdentityDef is an OBJECT-IDENTITY definition.
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

// NotificationTypeDef is a NOTIFICATION-TYPE definition (SMIv2).
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

// TrapTypeDef is a TRAP-TYPE definition (SMIv1).
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

// TextualConventionDef is a TEXTUAL-CONVENTION definition.
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

// TypeAssignmentDef is a type assignment definition.
type TypeAssignmentDef struct {
	Name   Ident
	Syntax TypeSyntax
	Span   types.Span
}

func (d *TypeAssignmentDef) DefinitionName() *Ident     { return &d.Name }
func (d *TypeAssignmentDef) DefinitionSpan() types.Span { return d.Span }
func (*TypeAssignmentDef) definition()                  {}

// ValueAssignmentDef is a value assignment definition (OID definition).
type ValueAssignmentDef struct {
	Name          Ident
	OidAssignment OidAssignment
	Span          types.Span
}

func (d *ValueAssignmentDef) DefinitionName() *Ident     { return &d.Name }
func (d *ValueAssignmentDef) DefinitionSpan() types.Span { return d.Span }
func (*ValueAssignmentDef) definition()                  {}

// ObjectGroupDef is an OBJECT-GROUP definition.
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

// NotificationGroupDef is a NOTIFICATION-GROUP definition.
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

// ModuleComplianceDef is a MODULE-COMPLIANCE definition.
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

// ComplianceModule is a MODULE clause in MODULE-COMPLIANCE.
type ComplianceModule struct {
	ModuleName      *Ident
	ModuleOid       *OidAssignment
	MandatoryGroups []Ident
	Compliances     []Compliance
	Span            types.Span
}

// Compliance is a compliance item (GROUP or OBJECT refinement).
type Compliance interface {
	compliance()
}

// ComplianceGroup is a GROUP clause in MODULE-COMPLIANCE.
type ComplianceGroup struct {
	Group       Ident
	Description QuotedString
	Span        types.Span
}

func (*ComplianceGroup) compliance() {}

// ComplianceObject is an OBJECT refinement in MODULE-COMPLIANCE.
type ComplianceObject struct {
	Object      Ident
	Syntax      *SyntaxClause
	WriteSyntax *SyntaxClause
	MinAccess   *AccessClause
	Description QuotedString
	Span        types.Span
}

func (*ComplianceObject) compliance() {}

// AgentCapabilitiesDef is an AGENT-CAPABILITIES definition.
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

// SupportsModule is a SUPPORTS clause in AGENT-CAPABILITIES.
type SupportsModule struct {
	ModuleName Ident
	ModuleOid  *OidAssignment
	Includes   []Ident
	Variations []Variation
	Span       types.Span
}

// Variation is a VARIATION clause in AGENT-CAPABILITIES.
type Variation interface {
	variation()
}

// ObjectVariation is an object VARIATION in AGENT-CAPABILITIES.
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

// NotificationVariation is a notification VARIATION in AGENT-CAPABILITIES.
type NotificationVariation struct {
	Notification Ident
	Access       *AccessClause
	Description  QuotedString
	Span         types.Span
}

func (*NotificationVariation) variation() {}

// MacroDefinitionDef is a MACRO definition (skipped content).
type MacroDefinitionDef struct {
	Name Ident
	Span types.Span
}

func (d *MacroDefinitionDef) DefinitionName() *Ident     { return &d.Name }
func (d *MacroDefinitionDef) DefinitionSpan() types.Span { return d.Span }
func (*MacroDefinitionDef) definition()                  {}

// ErrorDef is a parse error with recovery.
type ErrorDef struct {
	Message string
	Span    types.Span
}

func (d *ErrorDef) DefinitionName() *Ident     { return nil }
func (d *ErrorDef) DefinitionSpan() types.Span { return d.Span }
func (*ErrorDef) definition()                  {}
