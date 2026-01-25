package module

import (
	"github.com/golangsnmp/gomib/internal/types"
)

// Definition is a definition in a MIB module.
//
// Each definition type is normalized from its AST counterpart.
// SMIv1 and SMIv2 forms are unified where appropriate.
//
// Use type switches to dispatch on specific definition types.
type Definition interface {
	// DefinitionName returns the name of this definition.
	DefinitionName() string
	// DefinitionSpan returns the source location.
	DefinitionSpan() types.Span
	// DefinitionOid returns the OID assignment if this definition has one.
	DefinitionOid() *OidAssignment
}

// ObjectType is an OBJECT-TYPE definition.
type ObjectType struct {
	// Name is the object name.
	Name string
	// Syntax is the SYNTAX clause.
	Syntax TypeSyntax
	// Units is the UNITS clause.
	Units string
	// Access is the MAX-ACCESS (normalized from ACCESS if SMIv1).
	Access types.Access
	// Status is the STATUS (normalized from SMIv1 if needed).
	Status types.Status
	// Description is the DESCRIPTION (optional: many vendor MIBs omit this despite RFC requirement).
	Description string
	// Reference is the REFERENCE clause.
	Reference string
	// Index is the INDEX items (object references).
	Index []IndexItem
	// Augments is the AUGMENTS target.
	Augments string
	// DefVal is the DEFVAL clause (default value).
	DefVal DefVal
	// Oid is the OID assignment.
	Oid OidAssignment
	// Span is the source span.
	Span types.Span
}

func (d *ObjectType) DefinitionName() string        { return d.Name }
func (d *ObjectType) DefinitionSpan() types.Span    { return d.Span }
func (d *ObjectType) DefinitionOid() *OidAssignment { return &d.Oid }

// IndexItem is an item in an INDEX clause.
type IndexItem struct {
	// Implied indicates whether this index is IMPLIED.
	Implied bool
	// Object is the object reference.
	Object string
}

// ModuleIdentity is a MODULE-IDENTITY definition.
type ModuleIdentity struct {
	// Name is the identity name.
	Name string
	// LastUpdated is the LAST-UPDATED value.
	LastUpdated string
	// Organization is the ORGANIZATION value.
	Organization string
	// ContactInfo is the CONTACT-INFO value.
	ContactInfo string
	// Description is the DESCRIPTION value.
	Description string
	// Revisions are the REVISION clauses.
	Revisions []Revision
	// Oid is the OID assignment.
	Oid OidAssignment
	// Span is the source span.
	Span types.Span
}

func (d *ModuleIdentity) DefinitionName() string        { return d.Name }
func (d *ModuleIdentity) DefinitionSpan() types.Span    { return d.Span }
func (d *ModuleIdentity) DefinitionOid() *OidAssignment { return &d.Oid }

// Revision is a REVISION clause.
type Revision struct {
	// Date is the revision date.
	Date string
	// Description is the revision description.
	Description string
}

// ObjectIdentity is an OBJECT-IDENTITY definition.
type ObjectIdentity struct {
	// Name is the identity name.
	Name string
	// Status is the STATUS.
	Status types.Status
	// Description is the DESCRIPTION.
	Description string
	// Reference is the REFERENCE.
	Reference string
	// Oid is the OID assignment.
	Oid OidAssignment
	// Span is the source span.
	Span types.Span
}

func (d *ObjectIdentity) DefinitionName() string        { return d.Name }
func (d *ObjectIdentity) DefinitionSpan() types.Span    { return d.Span }
func (d *ObjectIdentity) DefinitionOid() *OidAssignment { return &d.Oid }

// Notification is a unified notification definition.
//
// Represents both SMIv1 TRAP-TYPE and SMIv2 NOTIFICATION-TYPE.
type Notification struct {
	// Name is the notification name.
	Name string
	// Objects is the OBJECTS/VARIABLES list.
	Objects []string
	// Status is the STATUS.
	Status types.Status
	// Description is the DESCRIPTION (optional: SMIv1 TRAP-TYPE has no DESCRIPTION clause).
	Description string
	// Reference is the REFERENCE.
	Reference string
	// TrapInfo is SMIv1 TRAP-TYPE specific information (nil for NOTIFICATION-TYPE).
	TrapInfo *TrapInfo
	// Oid is the OID assignment (nil for TRAP-TYPE; OID derived from enterprise + trap number).
	Oid *OidAssignment
	// Span is the source span.
	Span types.Span
}

func (d *Notification) DefinitionName() string        { return d.Name }
func (d *Notification) DefinitionSpan() types.Span    { return d.Span }
func (d *Notification) DefinitionOid() *OidAssignment { return d.Oid }

// TrapInfo is SMIv1 TRAP-TYPE specific information.
type TrapInfo struct {
	// Enterprise is the ENTERPRISE OID reference.
	Enterprise string
	// TrapNumber is the trap number.
	TrapNumber uint32
}

// TypeDef is a type definition.
//
// Represents both TEXTUAL-CONVENTION and simple type assignments.
type TypeDef struct {
	// Name is the type name.
	Name string
	// Syntax is the base syntax.
	Syntax TypeSyntax
	// BaseType is an explicit base type override.
	//
	// For most types, the base type is derived from Syntax. However, some
	// SMI base types like IpAddress are syntactically `OCTET STRING (SIZE 4)`
	// but have distinct semantic base types (for index encoding, etc.).
	// This field allows synthetic base modules to specify the correct base type.
	BaseType *types.BaseType
	// DisplayHint is the DISPLAY-HINT.
	DisplayHint string
	// Status is the STATUS.
	Status types.Status
	// Description is the DESCRIPTION (optional: simple type assignments have no DESCRIPTION clause).
	Description string
	// Reference is the REFERENCE.
	Reference string
	// IsTextualConvention is true if this was a TEXTUAL-CONVENTION (vs simple type assignment).
	IsTextualConvention bool
	// Span is the source span.
	Span types.Span
}

func (d *TypeDef) DefinitionName() string        { return d.Name }
func (d *TypeDef) DefinitionSpan() types.Span    { return d.Span }
func (d *TypeDef) DefinitionOid() *OidAssignment { return nil }

// ValueAssignment is a value assignment (OID definition).
type ValueAssignment struct {
	// Name is the value name.
	Name string
	// Oid is the OID assignment.
	Oid OidAssignment
	// Span is the source span.
	Span types.Span
}

func (d *ValueAssignment) DefinitionName() string        { return d.Name }
func (d *ValueAssignment) DefinitionSpan() types.Span    { return d.Span }
func (d *ValueAssignment) DefinitionOid() *OidAssignment { return &d.Oid }

// ObjectGroup is an OBJECT-GROUP definition.
type ObjectGroup struct {
	// Name is the group name.
	Name string
	// Objects is the OBJECTS in this group.
	Objects []string
	// Status is the STATUS.
	Status types.Status
	// Description is the DESCRIPTION.
	Description string
	// Reference is the REFERENCE.
	Reference string
	// Oid is the OID assignment.
	Oid OidAssignment
	// Span is the source span.
	Span types.Span
}

func (d *ObjectGroup) DefinitionName() string        { return d.Name }
func (d *ObjectGroup) DefinitionSpan() types.Span    { return d.Span }
func (d *ObjectGroup) DefinitionOid() *OidAssignment { return &d.Oid }

// NotificationGroup is a NOTIFICATION-GROUP definition.
type NotificationGroup struct {
	// Name is the group name.
	Name string
	// Notifications is the NOTIFICATIONS in this group.
	Notifications []string
	// Status is the STATUS.
	Status types.Status
	// Description is the DESCRIPTION.
	Description string
	// Reference is the REFERENCE.
	Reference string
	// Oid is the OID assignment.
	Oid OidAssignment
	// Span is the source span.
	Span types.Span
}

func (d *NotificationGroup) DefinitionName() string        { return d.Name }
func (d *NotificationGroup) DefinitionSpan() types.Span    { return d.Span }
func (d *NotificationGroup) DefinitionOid() *OidAssignment { return &d.Oid }

// ModuleCompliance is a MODULE-COMPLIANCE definition.
type ModuleCompliance struct {
	// Name is the compliance name.
	Name string
	// Status is the STATUS.
	Status types.Status
	// Description is the DESCRIPTION.
	Description string
	// Reference is the REFERENCE.
	Reference string
	// Modules are the MODULE clauses.
	Modules []ComplianceModule
	// Oid is the OID assignment.
	Oid OidAssignment
	// Span is the source span.
	Span types.Span
}

func (d *ModuleCompliance) DefinitionName() string        { return d.Name }
func (d *ModuleCompliance) DefinitionSpan() types.Span    { return d.Span }
func (d *ModuleCompliance) DefinitionOid() *OidAssignment { return &d.Oid }

// ComplianceModule is a MODULE clause in MODULE-COMPLIANCE.
type ComplianceModule struct {
	// ModuleName is the module name (empty = current module).
	ModuleName string
	// MandatoryGroups is the MANDATORY-GROUPS.
	MandatoryGroups []string
	// Groups are the GROUP refinements.
	Groups []ComplianceGroup
	// Objects are the OBJECT refinements.
	Objects []ComplianceObject
}

// ComplianceGroup is a GROUP clause.
type ComplianceGroup struct {
	// Group is the group reference.
	Group string
	// Description is the description.
	Description string
}

// ComplianceObject is an OBJECT refinement.
type ComplianceObject struct {
	// Object is the object reference.
	Object string
	// Syntax is the SYNTAX restriction.
	Syntax TypeSyntax
	// WriteSyntax is the WRITE-SYNTAX restriction.
	WriteSyntax TypeSyntax
	// MinAccess is the MIN-ACCESS restriction.
	MinAccess *types.Access
	// Description is the description.
	Description string
}

// AgentCapabilities is an AGENT-CAPABILITIES definition.
type AgentCapabilities struct {
	// Name is the capabilities name.
	Name string
	// ProductRelease is the PRODUCT-RELEASE value.
	ProductRelease string
	// Status is the STATUS.
	Status types.Status
	// Description is the DESCRIPTION.
	Description string
	// Reference is the REFERENCE.
	Reference string
	// Supports are the SUPPORTS clauses.
	Supports []SupportsModule
	// Oid is the OID assignment.
	Oid OidAssignment
	// Span is the source span.
	Span types.Span
}

func (d *AgentCapabilities) DefinitionName() string        { return d.Name }
func (d *AgentCapabilities) DefinitionSpan() types.Span    { return d.Span }
func (d *AgentCapabilities) DefinitionOid() *OidAssignment { return &d.Oid }

// SupportsModule is a SUPPORTS clause in AGENT-CAPABILITIES.
type SupportsModule struct {
	// ModuleName is the module name.
	ModuleName string
	// Includes is the INCLUDES list of group references.
	Includes []string
	// ObjectVariations are the object variations.
	ObjectVariations []ObjectVariation
	// NotificationVariations are the notification variations.
	NotificationVariations []NotificationVariation
}

// ObjectVariation is an object VARIATION.
type ObjectVariation struct {
	// Object is the object reference.
	Object string
	// Syntax is the SYNTAX restriction.
	Syntax TypeSyntax
	// WriteSyntax is the WRITE-SYNTAX restriction.
	WriteSyntax TypeSyntax
	// Access is the ACCESS restriction.
	Access *types.Access
	// CreationRequires is the CREATION-REQUIRES list.
	CreationRequires []string
	// DefVal is the DEFVAL override.
	DefVal DefVal
	// Description is the description.
	Description string
}

// NotificationVariation is a notification VARIATION.
type NotificationVariation struct {
	// Notification is the notification reference.
	Notification string
	// Access is the ACCESS restriction (only "not-implemented" is valid per RFC 2580).
	Access *types.Access
	// Description is the description.
	Description string
}
