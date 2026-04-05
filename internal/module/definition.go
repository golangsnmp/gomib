package module

import (
	"github.com/golangsnmp/gomib/internal/types"
)

// Definition is a normalized MIB definition. SMIv1 and SMIv2 forms are
// unified where appropriate. Use type switches to dispatch on concrete types.
type Definition interface {
	DefinitionName() string
	DefinitionSpan() types.Span
	// DefinitionOid returns the OID assignment, or nil if none.
	DefinitionOid() *OidAssignment
}

// DefBase provides the Name and Span fields common to all Definition types.
type DefBase struct {
	Name string
	Span types.Span
}

func (d *DefBase) DefinitionName() string     { return d.Name }
func (d *DefBase) DefinitionSpan() types.Span { return d.Span }

// ObjectTypeSpans holds source byte ranges for each clause in an OBJECT-TYPE.
type ObjectTypeSpans struct {
	Syntax      types.Span
	Access      types.Span
	Status      types.Span
	Description types.Span
	Units       types.Span
	Reference   types.Span
	Index       types.Span
	Augments    types.Span
	DefVal      types.Span
}

// ObjectType is an OBJECT-TYPE definition.
type ObjectType struct {
	DefBase
	Syntax         TypeSyntax
	Units          string
	Access         types.Access
	AccessKeyword  types.AccessKeyword
	Status         types.Status
	Description    string
	HasDescription bool // true when DESCRIPTION clause was present in source
	Reference      string
	Index          []IndexItem
	Augments       string
	DefVal         DefVal
	Oid            OidAssignment
	Spans          ObjectTypeSpans
}

func (d *ObjectType) DefinitionOid() *OidAssignment { return &d.Oid }

// NameRef is a reference to a named symbol with its source span.
// Used for member lists where individual diagnostic positioning is useful
// (e.g., notification OBJECTS, group members, mandatory groups).
type NameRef struct {
	Name string
	Span types.Span
}

// IndexItem is an entry in an OBJECT-TYPE INDEX clause.
type IndexItem struct {
	Implied bool
	Object  string
	Span    types.Span
}

// ModuleIdentity is a MODULE-IDENTITY definition.
type ModuleIdentity struct {
	DefBase
	LastUpdated  string
	Organization string
	ContactInfo  string
	Description  string
	Revisions    []Revision
	Oid          OidAssignment
}

func (d *ModuleIdentity) DefinitionOid() *OidAssignment { return &d.Oid }

// Revision is a REVISION clause within a MODULE-IDENTITY.
type Revision struct {
	Date        string
	Description string
	Span        types.Span
}

// ObjectIdentity is an OBJECT-IDENTITY definition.
type ObjectIdentity struct {
	DefBase
	Status      types.Status
	Description string
	Reference   string
	Oid         OidAssignment
}

func (d *ObjectIdentity) DefinitionOid() *OidAssignment { return &d.Oid }

// Notification represents both SMIv1 TRAP-TYPE and SMIv2 NOTIFICATION-TYPE.
type Notification struct {
	DefBase
	Objects        []NameRef
	Status         types.Status
	Description    string
	HasDescription bool // true when DESCRIPTION clause was present in source
	Reference      string
	// TrapInfo holds SMIv1 TRAP-TYPE fields. Nil for NOTIFICATION-TYPE.
	TrapInfo *TrapInfo
	// Oid is nil for TRAP-TYPE; its OID is derived from enterprise + trap number.
	Oid *OidAssignment
}

func (d *Notification) DefinitionOid() *OidAssignment { return d.Oid }

// IsTrap reports whether this is an SMIv1 TRAP-TYPE.
func (d *Notification) IsTrap() bool { return d.TrapInfo != nil }

// TrapInfo holds fields specific to SMIv1 TRAP-TYPE definitions.
type TrapInfo struct {
	Enterprise string
	TrapNumber uint32
}

// TypeDefSpans holds source byte ranges for each clause in a type definition.
type TypeDefSpans struct {
	Syntax      types.Span
	Status      types.Span
	Description types.Span
	Reference   types.Span
	DisplayHint types.Span
}

// TypeDef represents both TEXTUAL-CONVENTION and simple type assignments.
type TypeDef struct {
	DefBase
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
	Spans               TypeDefSpans
}

func (d *TypeDef) DefinitionOid() *OidAssignment { return nil }

// ValueAssignment is a plain OID value assignment.
type ValueAssignment struct {
	DefBase
	Oid         OidAssignment
	Description string
	Reference   string
}

func (d *ValueAssignment) DefinitionOid() *OidAssignment { return &d.Oid }

// ObjectGroup is an OBJECT-GROUP definition.
type ObjectGroup struct {
	DefBase
	Objects     []NameRef
	Status      types.Status
	Description string
	Reference   string
	Oid         OidAssignment
}

func (d *ObjectGroup) DefinitionOid() *OidAssignment { return &d.Oid }

// NotificationGroup is a NOTIFICATION-GROUP definition.
type NotificationGroup struct {
	DefBase
	Notifications []NameRef
	Status        types.Status
	Description   string
	Reference     string
	Oid           OidAssignment
}

func (d *NotificationGroup) DefinitionOid() *OidAssignment { return &d.Oid }

// ModuleCompliance is a MODULE-COMPLIANCE definition.
type ModuleCompliance struct {
	DefBase
	Status      types.Status
	Description string
	Reference   string
	Modules     []ComplianceModule
	Oid         OidAssignment
}

func (d *ModuleCompliance) DefinitionOid() *OidAssignment { return &d.Oid }

// ComplianceModule is a MODULE clause in MODULE-COMPLIANCE.
type ComplianceModule struct {
	// ModuleName is empty when referring to the current module.
	ModuleName      string
	MandatoryGroups []NameRef
	Groups          []ComplianceGroup
	Objects         []ComplianceObject
	Span            types.Span
}

// ComplianceGroup is a GROUP clause within MODULE-COMPLIANCE.
type ComplianceGroup struct {
	Group       string
	Description string
	Span        types.Span
}

// ComplianceObject is an OBJECT refinement within MODULE-COMPLIANCE.
type ComplianceObject struct {
	Object      string
	Syntax      TypeSyntax
	WriteSyntax TypeSyntax
	MinAccess   *types.Access
	Description string
	Span        types.Span
}

// AgentCapabilities is an AGENT-CAPABILITIES definition.
type AgentCapabilities struct {
	DefBase
	ProductRelease string
	Status         types.Status
	Description    string
	Reference      string
	Supports       []SupportsModule
	Oid            OidAssignment
}

func (d *AgentCapabilities) DefinitionOid() *OidAssignment { return &d.Oid }

// SupportsModule is a SUPPORTS clause in AGENT-CAPABILITIES.
type SupportsModule struct {
	ModuleName string
	Includes   []NameRef
	Variations []Variation
	Span       types.Span
}

// Variation is a VARIATION clause in AGENT-CAPABILITIES. At the IR level,
// object vs notification classification is deferred to the resolver, which
// can look up the referenced name's node kind.
type Variation struct {
	Name             string
	Syntax           TypeSyntax
	WriteSyntax      TypeSyntax
	Access           *types.Access
	CreationRequires []NameRef
	DefVal           DefVal
	Description      string
	Span             types.Span
}
