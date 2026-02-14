// Package gomib provides MIB parsing and querying for SNMP management.
package gomib

import "github.com/golangsnmp/gomib/mib"

// Mib is the top-level container for loaded MIB data.
type Mib = mib.Mib

// Node is a point in the OID tree.
type Node = mib.Node

// Object is an OBJECT-TYPE definition.
type Object = mib.Object

// Type is a type definition (textual convention or type reference).
type Type = mib.Type

// Notification is a NOTIFICATION-TYPE or TRAP-TYPE definition.
type Notification = mib.Notification

// Group is an OBJECT-GROUP or NOTIFICATION-GROUP definition.
type Group = mib.Group

// Compliance is a MODULE-COMPLIANCE definition.
type Compliance = mib.Compliance

// Capability is an AGENT-CAPABILITIES definition.
type Capability = mib.Capability

// Module is a parsed MIB module with its definitions.
type Module = mib.Module

// OID is a sequence of arc values representing an SNMP Object Identifier.
type OID = mib.OID

// Kind identifies what an OID node represents.
type Kind = mib.Kind

// Access represents the access level of an OBJECT-TYPE definition.
type Access = mib.Access

// Status represents the lifecycle status of a MIB definition.
type Status = mib.Status

// Language identifies the SMI version of a module.
type Language = mib.Language

// BaseType identifies the fundamental SMI type.
type BaseType = mib.BaseType

// Severity represents how critical a diagnostic is.
type Severity = mib.Severity

// Range represents a size or value constraint bound.
type Range = mib.Range

// NamedValue represents a labeled integer from an enum or BITS definition.
type NamedValue = mib.NamedValue

// IndexEntry describes an index component for a table row.
type IndexEntry = mib.IndexEntry

// Revision describes a module revision entry.
type Revision = mib.Revision

// Diagnostic represents an issue found during parsing or resolution.
type Diagnostic = mib.Diagnostic

// UnresolvedRef describes a symbol that could not be resolved.
type UnresolvedRef = mib.UnresolvedRef

// UnresolvedKind identifies the category of an unresolved reference.
type UnresolvedKind = mib.UnresolvedKind

const (
	UnresolvedImport             = mib.UnresolvedImport
	UnresolvedType               = mib.UnresolvedType
	UnresolvedOID                = mib.UnresolvedOID
	UnresolvedIndex              = mib.UnresolvedIndex
	UnresolvedNotificationObject = mib.UnresolvedNotificationObject
)

// ComplianceModule is a MODULE clause within a MODULE-COMPLIANCE definition.
type ComplianceModule = mib.ComplianceModule

// ComplianceGroup is a GROUP clause within MODULE-COMPLIANCE.
type ComplianceGroup = mib.ComplianceGroup

// ComplianceObject is an OBJECT refinement within MODULE-COMPLIANCE.
type ComplianceObject = mib.ComplianceObject

// CapabilitiesModule is a SUPPORTS clause within AGENT-CAPABILITIES.
type CapabilitiesModule = mib.CapabilitiesModule

// ObjectVariation is an object VARIATION within AGENT-CAPABILITIES.
type ObjectVariation = mib.ObjectVariation

// NotificationVariation is a notification VARIATION within AGENT-CAPABILITIES.
type NotificationVariation = mib.NotificationVariation

// DefVal holds a default value with both interpreted value and raw MIB syntax.
type DefVal = mib.DefVal

// DefValKind identifies the type of default value.
type DefValKind = mib.DefValKind

const (
	DefValKindInt    = mib.DefValKindInt
	DefValKindUint   = mib.DefValKindUint
	DefValKindString = mib.DefValKindString
	DefValKindBytes  = mib.DefValKindBytes
	DefValKindEnum   = mib.DefValKindEnum
	DefValKindBits   = mib.DefValKindBits
	DefValKindOID    = mib.DefValKindOID
)

// DefVal constructors for each value kind.
var (
	NewDefValInt    = mib.NewDefValInt
	NewDefValUint   = mib.NewDefValUint
	NewDefValString = mib.NewDefValString
	NewDefValBytes  = mib.NewDefValBytes
	NewDefValEnum   = mib.NewDefValEnum
	NewDefValBits   = mib.NewDefValBits
	NewDefValOID    = mib.NewDefValOID
)

// DefValAs returns the value as type T if compatible.
// Usage: value, ok := gomib.DefValAs[int64](defval)
func DefValAs[T any](d DefVal) (T, bool) {
	return mib.DefValAs[T](d)
}

const (
	KindUnknown      = mib.KindUnknown
	KindInternal     = mib.KindInternal
	KindNode         = mib.KindNode
	KindScalar       = mib.KindScalar
	KindTable        = mib.KindTable
	KindRow          = mib.KindRow
	KindColumn       = mib.KindColumn
	KindNotification = mib.KindNotification
	KindGroup        = mib.KindGroup
	KindCompliance   = mib.KindCompliance
	KindCapability   = mib.KindCapability
)

const (
	AccessNotAccessible       = mib.AccessNotAccessible
	AccessAccessibleForNotify = mib.AccessAccessibleForNotify
	AccessReadOnly            = mib.AccessReadOnly
	AccessReadWrite           = mib.AccessReadWrite
	AccessReadCreate          = mib.AccessReadCreate
	AccessWriteOnly           = mib.AccessWriteOnly
	AccessInstall             = mib.AccessInstall
	AccessInstallNotify       = mib.AccessInstallNotify
	AccessReportOnly          = mib.AccessReportOnly
	AccessNotImplemented      = mib.AccessNotImplemented
)

const (
	StatusCurrent    = mib.StatusCurrent
	StatusDeprecated = mib.StatusDeprecated
	StatusObsolete   = mib.StatusObsolete
	StatusMandatory  = mib.StatusMandatory
	StatusOptional   = mib.StatusOptional
)

const (
	LanguageUnknown = mib.LanguageUnknown
	LanguageSMIv1   = mib.LanguageSMIv1
	LanguageSMIv2   = mib.LanguageSMIv2
	LanguageSPPI    = mib.LanguageSPPI
)

const (
	BaseUnknown          = mib.BaseUnknown
	BaseInteger32        = mib.BaseInteger32
	BaseUnsigned32       = mib.BaseUnsigned32
	BaseCounter32        = mib.BaseCounter32
	BaseCounter64        = mib.BaseCounter64
	BaseGauge32          = mib.BaseGauge32
	BaseTimeTicks        = mib.BaseTimeTicks
	BaseIpAddress        = mib.BaseIpAddress
	BaseOctetString      = mib.BaseOctetString
	BaseObjectIdentifier = mib.BaseObjectIdentifier
	BaseBits             = mib.BaseBits
	BaseOpaque           = mib.BaseOpaque
	BaseSequence         = mib.BaseSequence
)

// Severity constants (libsmi-compatible, lower = more severe).
const (
	SeverityFatal   = mib.SeverityFatal   // 0: Cannot continue parsing
	SeveritySevere  = mib.SeveritySevere  // 1: Semantics changed to continue
	SeverityError   = mib.SeverityError   // 2: Should correct
	SeverityMinor   = mib.SeverityMinor   // 3: Minor issue
	SeverityStyle   = mib.SeverityStyle   // 4: Style recommendation
	SeverityWarning = mib.SeverityWarning // 5: Might be correct
	SeverityInfo    = mib.SeverityInfo    // 6: Informational
)

// StrictnessLevel defines preset strictness configurations.
type StrictnessLevel = mib.StrictnessLevel

const (
	StrictnessStrict     = mib.StrictnessStrict
	StrictnessNormal     = mib.StrictnessNormal
	StrictnessPermissive = mib.StrictnessPermissive
	StrictnessSilent     = mib.StrictnessSilent
)

// DiagnosticConfig controls strictness and diagnostic filtering.
type DiagnosticConfig = mib.DiagnosticConfig

// Preset diagnostic configuration constructors.
var (
	DefaultConfig    = mib.DefaultConfig
	StrictConfig     = mib.StrictConfig
	PermissiveConfig = mib.PermissiveConfig
)

// ParseOID parses an OID from a dotted string (e.g., "1.3.6.1.2.1").
var ParseOID = mib.ParseOID
