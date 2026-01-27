// Package gomib provides MIB parsing and querying for SNMP management.
package gomib

import "github.com/golangsnmp/gomib/mib"

// Type aliases for public API - all types come from mib subpackage.

// Mib is the top-level container for loaded MIB data.
type Mib = mib.Mib

// Node is a point in the OID tree.
type Node = mib.Node

// Object is an OBJECT-TYPE definition.
type Object = mib.Object

// Type is a type definition (textual convention or type reference).
type Type = mib.Type

// Notification is a NOTIFICATION-TYPE or TRAP-TYPE.
type Notification = mib.Notification

// Module is a MIB module.
type Module = mib.Module

// Oid is a sequence of arc values representing an SNMP Object Identifier.
type Oid = mib.Oid

// Kind identifies what an OID node represents.
type Kind = mib.Kind

// Access levels for OBJECT-TYPE definitions.
type Access = mib.Access

// Status values for MIB definitions.
type Status = mib.Status

// Language identifies the SMI version of a module.
type Language = mib.Language

// BaseType identifies the fundamental SMI type.
type BaseType = mib.BaseType

// Severity for diagnostics.
type Severity = mib.Severity

// Range for size/value constraints.
type Range = mib.Range

// NamedValue represents a labeled integer from an enum or BITS definition.
type NamedValue = mib.NamedValue

// IndexEntry describes an index component for a table row.
type IndexEntry = mib.IndexEntry

// Revision describes a module revision.
type Revision = mib.Revision

// Diagnostic represents a parse or resolution issue.
type Diagnostic = mib.Diagnostic

// UnresolvedRef describes a symbol that could not be resolved.
type UnresolvedRef = mib.UnresolvedRef

// DefVal represents a default value with both interpreted value and raw MIB syntax.
type DefVal = mib.DefVal

// DefValKind identifies the type of default value.
type DefValKind = mib.DefValKind

// DefVal kind constants.
const (
	DefValKindInt    = mib.DefValKindInt
	DefValKindUint   = mib.DefValKindUint
	DefValKindString = mib.DefValKindString
	DefValKindBytes  = mib.DefValKindBytes
	DefValKindEnum   = mib.DefValKindEnum
	DefValKindBits   = mib.DefValKindBits
	DefValKindOID    = mib.DefValKindOID
)

// DefVal constructors.
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

// Kind constants.
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
	KindCapabilities = mib.KindCapabilities
)

// Access constants.
const (
	AccessNotAccessible       = mib.AccessNotAccessible
	AccessAccessibleForNotify = mib.AccessAccessibleForNotify
	AccessReadOnly            = mib.AccessReadOnly
	AccessReadWrite           = mib.AccessReadWrite
	AccessReadCreate          = mib.AccessReadCreate
	AccessWriteOnly           = mib.AccessWriteOnly
)

// Status constants.
const (
	StatusCurrent    = mib.StatusCurrent
	StatusDeprecated = mib.StatusDeprecated
	StatusObsolete   = mib.StatusObsolete
)

// Language constants.
const (
	LanguageSMIv1 = mib.LanguageSMIv1
	LanguageSMIv2 = mib.LanguageSMIv2
)

// BaseType constants.
const (
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

// StrictnessLevel constants.
const (
	StrictnessStrict     = mib.StrictnessStrict
	StrictnessNormal     = mib.StrictnessNormal
	StrictnessPermissive = mib.StrictnessPermissive
	StrictnessSilent     = mib.StrictnessSilent
)

// DiagnosticConfig controls strictness and diagnostic filtering.
type DiagnosticConfig = mib.DiagnosticConfig

// Config constructors.
var (
	DefaultConfig     = mib.DefaultConfig
	StrictConfig      = mib.StrictConfig
	PermissiveConfig  = mib.PermissiveConfig
)

// ParseOID parses an OID from a dotted string (e.g., "1.3.6.1.2.1").
var ParseOID = mib.ParseOID
