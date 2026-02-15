package module

import (
	"github.com/golangsnmp/gomib/internal/types"
)

// AccessKeyword records which keyword was used (ACCESS, MAX-ACCESS, etc.).
type AccessKeyword int

const (
	AccessKeywordAccess    AccessKeyword = iota // SMIv1: ACCESS
	AccessKeywordMaxAccess                      // SMIv2: MAX-ACCESS
	AccessKeywordMinAccess                      // SMIv2: MIN-ACCESS (compliance)
	AccessKeywordPibAccess                      // SPPI: PIB-ACCESS
)

// Definition is a normalized MIB definition. SMIv1 and SMIv2 forms are
// unified where appropriate. Use type switches to dispatch on concrete types.
type Definition interface {
	DefinitionName() string
	DefinitionSpan() types.Span
	// DefinitionOid returns the OID assignment, or nil if none.
	DefinitionOid() *OidAssignment
}

// ObjectType is an OBJECT-TYPE definition.
type ObjectType struct {
	Name          string
	Syntax        TypeSyntax
	Units         string
	Access        types.Access
	AccessKeyword AccessKeyword
	Status        types.Status
	Description   string
	Reference     string
	Index         []IndexItem
	Augments      string
	DefVal        DefVal
	Oid           OidAssignment
	Span          types.Span
}

func (d *ObjectType) DefinitionName() string        { return d.Name }
func (d *ObjectType) DefinitionSpan() types.Span    { return d.Span }
func (d *ObjectType) DefinitionOid() *OidAssignment { return &d.Oid }

// IndexItem is an entry in an OBJECT-TYPE INDEX clause.
type IndexItem struct {
	Implied bool
	Object  string
}

// ModuleIdentity is a MODULE-IDENTITY definition.
type ModuleIdentity struct {
	Name         string
	LastUpdated  string
	Organization string
	ContactInfo  string
	Description  string
	Revisions    []Revision
	Oid          OidAssignment
	Span         types.Span
}

func (d *ModuleIdentity) DefinitionName() string        { return d.Name }
func (d *ModuleIdentity) DefinitionSpan() types.Span    { return d.Span }
func (d *ModuleIdentity) DefinitionOid() *OidAssignment { return &d.Oid }

// Revision is a REVISION clause within a MODULE-IDENTITY.
type Revision struct {
	Date        string
	Description string
}

// ObjectIdentity is an OBJECT-IDENTITY definition.
type ObjectIdentity struct {
	Name        string
	Status      types.Status
	Description string
	Reference   string
	Oid         OidAssignment
	Span        types.Span
}

func (d *ObjectIdentity) DefinitionName() string        { return d.Name }
func (d *ObjectIdentity) DefinitionSpan() types.Span    { return d.Span }
func (d *ObjectIdentity) DefinitionOid() *OidAssignment { return &d.Oid }

// Notification represents both SMIv1 TRAP-TYPE and SMIv2 NOTIFICATION-TYPE.
type Notification struct {
	Name        string
	Objects     []string
	Status      types.Status
	Description string
	Reference   string
	// TrapInfo holds SMIv1 TRAP-TYPE fields. Nil for NOTIFICATION-TYPE.
	TrapInfo *TrapInfo
	// Oid is nil for TRAP-TYPE; its OID is derived from enterprise + trap number.
	Oid  *OidAssignment
	Span types.Span
}

func (d *Notification) DefinitionName() string        { return d.Name }
func (d *Notification) DefinitionSpan() types.Span    { return d.Span }
func (d *Notification) DefinitionOid() *OidAssignment { return d.Oid }

// IsTrap reports whether this is an SMIv1 TRAP-TYPE.
func (d *Notification) IsTrap() bool { return d.TrapInfo != nil }

// TrapInfo holds fields specific to SMIv1 TRAP-TYPE definitions.
type TrapInfo struct {
	Enterprise string
	TrapNumber uint32
}

// TypeDef represents both TEXTUAL-CONVENTION and simple type assignments.
type TypeDef struct {
	Name   string
	Syntax TypeSyntax
	// BaseType overrides the base type derived from Syntax. Some SMI base
	// types like IpAddress are syntactically OCTET STRING (SIZE 4) but have
	// distinct semantic base types (for index encoding, etc.). This field
	// allows synthetic base modules to specify the correct base type.
	BaseType            *types.BaseType
	DisplayHint         string
	Status              types.Status
	Description         string
	Reference           string
	IsTextualConvention bool
	Span                types.Span
}

func (d *TypeDef) DefinitionName() string        { return d.Name }
func (d *TypeDef) DefinitionSpan() types.Span    { return d.Span }
func (d *TypeDef) DefinitionOid() *OidAssignment { return nil }

// ValueAssignment is a plain OID value assignment.
type ValueAssignment struct {
	Name string
	Oid  OidAssignment
	Span types.Span
}

func (d *ValueAssignment) DefinitionName() string        { return d.Name }
func (d *ValueAssignment) DefinitionSpan() types.Span    { return d.Span }
func (d *ValueAssignment) DefinitionOid() *OidAssignment { return &d.Oid }

// ObjectGroup is an OBJECT-GROUP definition.
type ObjectGroup struct {
	Name        string
	Objects     []string
	Status      types.Status
	Description string
	Reference   string
	Oid         OidAssignment
	Span        types.Span
}

func (d *ObjectGroup) DefinitionName() string        { return d.Name }
func (d *ObjectGroup) DefinitionSpan() types.Span    { return d.Span }
func (d *ObjectGroup) DefinitionOid() *OidAssignment { return &d.Oid }

// NotificationGroup is a NOTIFICATION-GROUP definition.
type NotificationGroup struct {
	Name          string
	Notifications []string
	Status        types.Status
	Description   string
	Reference     string
	Oid           OidAssignment
	Span          types.Span
}

func (d *NotificationGroup) DefinitionName() string        { return d.Name }
func (d *NotificationGroup) DefinitionSpan() types.Span    { return d.Span }
func (d *NotificationGroup) DefinitionOid() *OidAssignment { return &d.Oid }

// ModuleCompliance is a MODULE-COMPLIANCE definition.
type ModuleCompliance struct {
	Name        string
	Status      types.Status
	Description string
	Reference   string
	Modules     []ComplianceModule
	Oid         OidAssignment
	Span        types.Span
}

func (d *ModuleCompliance) DefinitionName() string        { return d.Name }
func (d *ModuleCompliance) DefinitionSpan() types.Span    { return d.Span }
func (d *ModuleCompliance) DefinitionOid() *OidAssignment { return &d.Oid }

// ComplianceModule is a MODULE clause in MODULE-COMPLIANCE.
type ComplianceModule struct {
	// ModuleName is empty when referring to the current module.
	ModuleName      string
	MandatoryGroups []string
	Groups          []ComplianceGroup
	Objects         []ComplianceObject
}

// ComplianceGroup is a GROUP clause within MODULE-COMPLIANCE.
type ComplianceGroup struct {
	Group       string
	Description string
}

// ComplianceObject is an OBJECT refinement within MODULE-COMPLIANCE.
type ComplianceObject struct {
	Object      string
	Syntax      TypeSyntax
	WriteSyntax TypeSyntax
	MinAccess   *types.Access
	Description string
}

// AgentCapabilities is an AGENT-CAPABILITIES definition.
type AgentCapabilities struct {
	Name           string
	ProductRelease string
	Status         types.Status
	Description    string
	Reference      string
	Supports       []SupportsModule
	Oid            OidAssignment
	Span           types.Span
}

func (d *AgentCapabilities) DefinitionName() string        { return d.Name }
func (d *AgentCapabilities) DefinitionSpan() types.Span    { return d.Span }
func (d *AgentCapabilities) DefinitionOid() *OidAssignment { return &d.Oid }

// SupportsModule is a SUPPORTS clause in AGENT-CAPABILITIES.
type SupportsModule struct {
	ModuleName             string
	Includes               []string
	ObjectVariations       []ObjectVariation
	NotificationVariations []NotificationVariation
}

// ObjectVariation is an object VARIATION in AGENT-CAPABILITIES.
type ObjectVariation struct {
	Object           string
	Syntax           TypeSyntax
	WriteSyntax      TypeSyntax
	Access           *types.Access
	CreationRequires []string
	DefVal           DefVal
	Description      string
}

// NotificationVariation is a notification VARIATION in AGENT-CAPABILITIES.
type NotificationVariation struct {
	Notification string
	// Access is only "not-implemented" per RFC 2580.
	Access      *types.Access
	Description string
}
