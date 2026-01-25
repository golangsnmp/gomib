package ast

import (
	"github.com/golangsnmp/gomib/internal/types"
)

// Definition is a definition in a MIB module body.
// Use type switches to handle specific definition types.
type Definition interface {
	// DefinitionName returns the name of this definition, if it has one.
	DefinitionName() *Ident
	// DefinitionSpan returns the source location.
	DefinitionSpan() types.Span
	// definition marker
	definition()
}

// ObjectTypeDef is an OBJECT-TYPE definition.
//
// The most common definition type in MIBs.
//
// Example:
//
//	ifIndex OBJECT-TYPE
//	    SYNTAX      InterfaceIndex
//	    MAX-ACCESS  read-only
//	    STATUS      current
//	    DESCRIPTION "..."
//	    ::= { ifEntry 1 }
type ObjectTypeDef struct {
	// Name is the object name.
	Name Ident
	// Syntax is the SYNTAX clause.
	Syntax SyntaxClause
	// Units is the UNITS clause (optional).
	Units *QuotedString
	// Access is the MAX-ACCESS or ACCESS clause.
	Access AccessClause
	// Status is the STATUS clause (optional in some vendor MIBs).
	Status *StatusClause
	// Description is the DESCRIPTION clause.
	Description *QuotedString
	// Reference is the REFERENCE clause.
	Reference *QuotedString
	// Index is the INDEX clause.
	Index IndexClause
	// Augments is the AUGMENTS clause.
	Augments *AugmentsClause
	// DefVal is the DEFVAL clause.
	DefVal *DefValClause
	// OidAssignment is the OID assignment.
	OidAssignment OidAssignment
	// Span is the source location.
	Span types.Span
}

func (d *ObjectTypeDef) DefinitionName() *Ident     { return &d.Name }
func (d *ObjectTypeDef) DefinitionSpan() types.Span { return d.Span }
func (*ObjectTypeDef) definition()                  {}

// ModuleIdentityDef is a MODULE-IDENTITY definition.
//
// Provides module-level metadata. Must be the first definition in SMIv2 modules.
type ModuleIdentityDef struct {
	// Name is the identity name.
	Name Ident
	// LastUpdated is the LAST-UPDATED value.
	LastUpdated QuotedString
	// Organization is the ORGANIZATION value.
	Organization QuotedString
	// ContactInfo is the CONTACT-INFO value.
	ContactInfo QuotedString
	// Description is the DESCRIPTION value.
	Description QuotedString
	// Revisions are the REVISION clauses.
	Revisions []RevisionClause
	// OidAssignment is the OID assignment.
	OidAssignment OidAssignment
	// Span is the source location.
	Span types.Span
}

func (d *ModuleIdentityDef) DefinitionName() *Ident     { return &d.Name }
func (d *ModuleIdentityDef) DefinitionSpan() types.Span { return d.Span }
func (*ModuleIdentityDef) definition()                  {}

// ObjectIdentityDef is an OBJECT-IDENTITY definition.
//
// Defines an OID without a value, used for documentation and organization.
type ObjectIdentityDef struct {
	// Name is the identity name.
	Name Ident
	// Status is the STATUS clause.
	Status StatusClause
	// Description is the DESCRIPTION value.
	Description QuotedString
	// Reference is the REFERENCE clause.
	Reference *QuotedString
	// OidAssignment is the OID assignment.
	OidAssignment OidAssignment
	// Span is the source location.
	Span types.Span
}

func (d *ObjectIdentityDef) DefinitionName() *Ident     { return &d.Name }
func (d *ObjectIdentityDef) DefinitionSpan() types.Span { return d.Span }
func (*ObjectIdentityDef) definition()                  {}

// NotificationTypeDef is a NOTIFICATION-TYPE definition (SMIv2).
type NotificationTypeDef struct {
	// Name is the notification name.
	Name Ident
	// Objects are the OBJECTS clause (varbind list).
	Objects []Ident
	// Status is the STATUS clause.
	Status StatusClause
	// Description is the DESCRIPTION value.
	Description QuotedString
	// Reference is the REFERENCE clause.
	Reference *QuotedString
	// OidAssignment is the OID assignment.
	OidAssignment OidAssignment
	// Span is the source location.
	Span types.Span
}

func (d *NotificationTypeDef) DefinitionName() *Ident     { return &d.Name }
func (d *NotificationTypeDef) DefinitionSpan() types.Span { return d.Span }
func (*NotificationTypeDef) definition()                  {}

// TrapTypeDef is a TRAP-TYPE definition (SMIv1).
type TrapTypeDef struct {
	// Name is the trap name.
	Name Ident
	// Enterprise is the ENTERPRISE OID.
	Enterprise Ident
	// Variables are the VARIABLES clause.
	Variables []Ident
	// Description is the DESCRIPTION value.
	Description *QuotedString
	// Reference is the REFERENCE clause.
	Reference *QuotedString
	// TrapNumber is the trap number (::= number).
	TrapNumber uint32
	// Span is the source location.
	Span types.Span
}

func (d *TrapTypeDef) DefinitionName() *Ident     { return &d.Name }
func (d *TrapTypeDef) DefinitionSpan() types.Span { return d.Span }
func (*TrapTypeDef) definition()                  {}

// TextualConventionDef is a TEXTUAL-CONVENTION definition.
type TextualConventionDef struct {
	// Name is the TC name.
	Name Ident
	// DisplayHint is the DISPLAY-HINT value.
	DisplayHint *QuotedString
	// Status is the STATUS clause.
	Status StatusClause
	// Description is the DESCRIPTION value.
	Description QuotedString
	// Reference is the REFERENCE clause.
	Reference *QuotedString
	// Syntax is the SYNTAX clause.
	Syntax SyntaxClause
	// Span is the source location.
	Span types.Span
}

func (d *TextualConventionDef) DefinitionName() *Ident     { return &d.Name }
func (d *TextualConventionDef) DefinitionSpan() types.Span { return d.Span }
func (*TextualConventionDef) definition()                  {}

// TypeAssignmentDef is a type assignment definition.
//
// Examples:
//   - InterfaceIndex ::= Integer32 (simple alias)
//   - IfEntry ::= SEQUENCE { ifIndex INTEGER, ... } (row definition)
type TypeAssignmentDef struct {
	// Name is the type name.
	Name Ident
	// Syntax is the type syntax.
	Syntax TypeSyntax
	// Span is the source location.
	Span types.Span
}

func (d *TypeAssignmentDef) DefinitionName() *Ident     { return &d.Name }
func (d *TypeAssignmentDef) DefinitionSpan() types.Span { return d.Span }
func (*TypeAssignmentDef) definition()                  {}

// ValueAssignmentDef is a value assignment definition (OID definition).
//
// Example: internet OBJECT IDENTIFIER ::= { iso org(3) dod(6) 1 }
type ValueAssignmentDef struct {
	// Name is the value name.
	Name Ident
	// OidAssignment is the OID assignment.
	OidAssignment OidAssignment
	// Span is the source location.
	Span types.Span
}

func (d *ValueAssignmentDef) DefinitionName() *Ident     { return &d.Name }
func (d *ValueAssignmentDef) DefinitionSpan() types.Span { return d.Span }
func (*ValueAssignmentDef) definition()                  {}

// ObjectGroupDef is an OBJECT-GROUP definition.
type ObjectGroupDef struct {
	// Name is the group name.
	Name Ident
	// Objects are the OBJECTS in this group.
	Objects []Ident
	// Status is the STATUS clause.
	Status StatusClause
	// Description is the DESCRIPTION value.
	Description QuotedString
	// Reference is the REFERENCE clause.
	Reference *QuotedString
	// OidAssignment is the OID assignment.
	OidAssignment OidAssignment
	// Span is the source location.
	Span types.Span
}

func (d *ObjectGroupDef) DefinitionName() *Ident     { return &d.Name }
func (d *ObjectGroupDef) DefinitionSpan() types.Span { return d.Span }
func (*ObjectGroupDef) definition()                  {}

// NotificationGroupDef is a NOTIFICATION-GROUP definition.
type NotificationGroupDef struct {
	// Name is the group name.
	Name Ident
	// Notifications are the NOTIFICATIONS in this group.
	Notifications []Ident
	// Status is the STATUS clause.
	Status StatusClause
	// Description is the DESCRIPTION value.
	Description QuotedString
	// Reference is the REFERENCE clause.
	Reference *QuotedString
	// OidAssignment is the OID assignment.
	OidAssignment OidAssignment
	// Span is the source location.
	Span types.Span
}

func (d *NotificationGroupDef) DefinitionName() *Ident     { return &d.Name }
func (d *NotificationGroupDef) DefinitionSpan() types.Span { return d.Span }
func (*NotificationGroupDef) definition()                  {}

// ModuleComplianceDef is a MODULE-COMPLIANCE definition.
type ModuleComplianceDef struct {
	// Name is the compliance name.
	Name Ident
	// Status is the STATUS clause.
	Status StatusClause
	// Description is the DESCRIPTION value.
	Description QuotedString
	// Reference is the REFERENCE clause.
	Reference *QuotedString
	// Modules are the MODULE clauses.
	Modules []ComplianceModule
	// OidAssignment is the OID assignment.
	OidAssignment OidAssignment
	// Span is the source location.
	Span types.Span
}

func (d *ModuleComplianceDef) DefinitionName() *Ident     { return &d.Name }
func (d *ModuleComplianceDef) DefinitionSpan() types.Span { return d.Span }
func (*ModuleComplianceDef) definition()                  {}

// ComplianceModule is a MODULE clause in MODULE-COMPLIANCE.
type ComplianceModule struct {
	// ModuleName is the module name (nil = current module).
	ModuleName *Ident
	// ModuleOid is the module OID (optional, rare).
	ModuleOid *OidAssignment
	// MandatoryGroups is the MANDATORY-GROUPS list.
	MandatoryGroups []Ident
	// Compliances are the GROUP and OBJECT refinements.
	Compliances []Compliance
	// Span is the source location.
	Span types.Span
}

// Compliance is a compliance item (GROUP or OBJECT refinement).
type Compliance interface {
	// compliance marker
	compliance()
}

// ComplianceGroup is a GROUP clause in MODULE-COMPLIANCE.
type ComplianceGroup struct {
	// Group is the group reference.
	Group Ident
	// Description is the DESCRIPTION.
	Description QuotedString
	// Span is the source location.
	Span types.Span
}

func (*ComplianceGroup) compliance() {}

// ComplianceObject is an OBJECT refinement in MODULE-COMPLIANCE.
type ComplianceObject struct {
	// Object is the object reference.
	Object Ident
	// Syntax is the SYNTAX restriction (optional).
	Syntax *SyntaxClause
	// WriteSyntax is the WRITE-SYNTAX restriction (optional).
	WriteSyntax *SyntaxClause
	// MinAccess is the MIN-ACCESS restriction (optional).
	MinAccess *AccessClause
	// Description is the DESCRIPTION (required per RFC 2580).
	Description QuotedString
	// Span is the source location.
	Span types.Span
}

func (*ComplianceObject) compliance() {}

// AgentCapabilitiesDef is an AGENT-CAPABILITIES definition.
type AgentCapabilitiesDef struct {
	// Name is the capabilities name.
	Name Ident
	// ProductRelease is the PRODUCT-RELEASE value.
	ProductRelease QuotedString
	// Status is the STATUS clause.
	Status StatusClause
	// Description is the DESCRIPTION value.
	Description QuotedString
	// Reference is the REFERENCE clause.
	Reference *QuotedString
	// Supports are the SUPPORTS clauses.
	Supports []SupportsModule
	// OidAssignment is the OID assignment.
	OidAssignment OidAssignment
	// Span is the source location.
	Span types.Span
}

func (d *AgentCapabilitiesDef) DefinitionName() *Ident     { return &d.Name }
func (d *AgentCapabilitiesDef) DefinitionSpan() types.Span { return d.Span }
func (*AgentCapabilitiesDef) definition()                  {}

// SupportsModule is a SUPPORTS clause in AGENT-CAPABILITIES.
type SupportsModule struct {
	// ModuleName is the module name.
	ModuleName Ident
	// ModuleOid is the module OID (optional).
	ModuleOid *OidAssignment
	// Includes is the INCLUDES list of groups.
	Includes []Ident
	// Variations are the VARIATION clauses.
	Variations []Variation
	// Span is the source location.
	Span types.Span
}

// Variation is a VARIATION clause in AGENT-CAPABILITIES.
type Variation interface {
	// variation marker
	variation()
}

// ObjectVariation is an object VARIATION in AGENT-CAPABILITIES.
type ObjectVariation struct {
	// Object is the object reference.
	Object Ident
	// Syntax is the SYNTAX restriction (optional).
	Syntax *SyntaxClause
	// WriteSyntax is the WRITE-SYNTAX restriction (optional).
	WriteSyntax *SyntaxClause
	// Access is the ACCESS restriction (optional).
	Access *AccessClause
	// CreationRequires is the CREATION-REQUIRES list (optional).
	CreationRequires []Ident
	// DefVal is the DEFVAL override (optional).
	DefVal *DefValClause
	// Description is the DESCRIPTION (required).
	Description QuotedString
	// Span is the source location.
	Span types.Span
}

func (*ObjectVariation) variation() {}

// NotificationVariation is a notification VARIATION in AGENT-CAPABILITIES.
type NotificationVariation struct {
	// Notification is the notification reference.
	Notification Ident
	// Access is the ACCESS restriction (optional, only "not-implemented" is valid).
	Access *AccessClause
	// Description is the DESCRIPTION (required).
	Description QuotedString
	// Span is the source location.
	Span types.Span
}

func (*NotificationVariation) variation() {}

// MacroDefinitionDef is a MACRO definition (skipped content).
//
// MACRO definitions only appear in base SMI modules (SNMPv2-SMI, etc.).
// We record them but don't parse their content.
type MacroDefinitionDef struct {
	// Name is the MACRO name (e.g., OBJECT-TYPE).
	Name Ident
	// Span is the source location.
	Span types.Span
}

func (d *MacroDefinitionDef) DefinitionName() *Ident     { return &d.Name }
func (d *MacroDefinitionDef) DefinitionSpan() types.Span { return d.Span }
func (*MacroDefinitionDef) definition()                  {}

// ErrorDef is a parse error with recovery.
//
// When the parser encounters an error, it records the location and
// attempts to recover to continue parsing.
type ErrorDef struct {
	// Message is the error message.
	Message string
	// Span is the source location where error occurred.
	Span types.Span
}

func (d *ErrorDef) DefinitionName() *Ident     { return nil }
func (d *ErrorDef) DefinitionSpan() types.Span { return d.Span }
func (*ErrorDef) definition()                  {}
