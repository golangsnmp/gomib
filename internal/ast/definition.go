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

// DefBase provides the Name and Span fields common to most Definition types.
type DefBase struct {
	Name Ident
	Span types.Span
}

func (d *DefBase) DefinitionName() *Ident     { return &d.Name }
func (d *DefBase) DefinitionSpan() types.Span { return d.Span }
func (*DefBase) definition()                  {}

// ObjectTypeDef represents an OBJECT-TYPE macro invocation (SMIv1/v2).
type ObjectTypeDef struct {
	DefBase
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
}

// ModuleIdentityDef represents a MODULE-IDENTITY macro invocation (SMIv2).
type ModuleIdentityDef struct {
	DefBase
	LastUpdated   QuotedString
	Organization  QuotedString
	ContactInfo   QuotedString
	Description   QuotedString
	Revisions     []RevisionClause
	OidAssignment OidAssignment
}

// ObjectIdentityDef represents an OBJECT-IDENTITY macro invocation (SMIv2).
type ObjectIdentityDef struct {
	DefBase
	Status        StatusClause
	Description   QuotedString
	Reference     *QuotedString
	OidAssignment OidAssignment
}

// NotificationTypeDef represents a NOTIFICATION-TYPE macro invocation (SMIv2).
type NotificationTypeDef struct {
	DefBase
	Objects       []Ident
	Status        StatusClause
	Description   QuotedString
	Reference     *QuotedString
	OidAssignment OidAssignment
}

// TrapTypeDef represents a TRAP-TYPE macro invocation (SMIv1).
type TrapTypeDef struct {
	DefBase
	Enterprise  Ident
	Variables   []Ident
	Description *QuotedString
	Reference   *QuotedString
	TrapNumber  uint32
}

// TextualConventionDef represents a TEXTUAL-CONVENTION definition (SMIv2).
type TextualConventionDef struct {
	DefBase
	DisplayHint *QuotedString
	Status      StatusClause
	Description QuotedString
	Reference   *QuotedString
	Syntax      SyntaxClause
}

// TypeAssignmentDef represents a type assignment (TypeName ::= TypeSyntax).
type TypeAssignmentDef struct {
	DefBase
	Syntax TypeSyntax
}

// ValueAssignmentDef represents an OID value assignment
// (name OBJECT IDENTIFIER ::= { ... }).
type ValueAssignmentDef struct {
	DefBase
	OidAssignment OidAssignment
}

// ObjectGroupDef represents an OBJECT-GROUP macro invocation (SMIv2).
type ObjectGroupDef struct {
	DefBase
	Objects       []Ident
	Status        StatusClause
	Description   QuotedString
	Reference     *QuotedString
	OidAssignment OidAssignment
}

// NotificationGroupDef represents a NOTIFICATION-GROUP macro invocation (SMIv2).
type NotificationGroupDef struct {
	DefBase
	Notifications []Ident
	Status        StatusClause
	Description   QuotedString
	Reference     *QuotedString
	OidAssignment OidAssignment
}

// ModuleComplianceDef represents a MODULE-COMPLIANCE macro invocation (SMIv2).
type ModuleComplianceDef struct {
	DefBase
	Status        StatusClause
	Description   QuotedString
	Reference     *QuotedString
	Modules       []ComplianceModule
	OidAssignment OidAssignment
}

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
	DefBase
	ProductRelease QuotedString
	Status         StatusClause
	Description    QuotedString
	Reference      *QuotedString
	Supports       []SupportsModule
	OidAssignment  OidAssignment
}

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
	DefBase
}

// ErrorDef records a parse error from which the parser recovered.
type ErrorDef struct {
	DefBase
	Message string
}

func (d *ErrorDef) DefinitionName() *Ident { return nil }
