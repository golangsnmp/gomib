package types

import (
	"fmt"
	"slices"
)

// enumString returns names[v] if in range, or "TypeName(v)" otherwise.
func enumString[T ~int](v T, names []string, typeName string) string {
	i := int(v)
	if i >= 0 && i < len(names) {
		return names[i]
	}
	return fmt.Sprintf("%s(%d)", typeName, i)
}

// Severity indicates how serious a diagnostic issue is (libsmi-compatible).
// Lower values are more severe.
type Severity int

const (
	SeverityFatal   Severity = 0 // Cannot continue parsing
	SeveritySevere  Severity = 1 // Semantics changed to continue, must correct
	SeverityError   Severity = 2 // Able to continue, should correct
	SeverityMinor   Severity = 3 // Minor issue, should correct
	SeverityStyle   Severity = 4 // Style recommendation
	SeverityWarning Severity = 5 // Might be correct under some circumstances
	SeverityInfo    Severity = 6 // Informational notice
)

var severityNames = [...]string{"fatal", "severe", "error", "minor", "style", "warning", "info"}

func (s Severity) String() string { return enumString(s, severityNames[:], "Severity") }

// SeverityNames returns the severity level names in order (fatal..info).
func SeverityNames() []string { return slices.Clone(severityNames[:]) }

// AtLeast reports whether s is at least as severe as other.
// Lower numeric values are more severe (Fatal=0, Info=6).
func (s Severity) AtLeast(other Severity) bool {
	return s <= other
}

// ResolverStrictness controls which heuristic fallbacks the resolver uses
// when resolving ambiguous or missing references.
//
// This only affects resolution strategy. It does not control which
// diagnostics cause failure - that is [DiagnosticConfig.FailAt].
type ResolverStrictness int

const (
	// ResolverStrict uses tier-1 only: no fallbacks.
	ResolverStrict ResolverStrictness = iota
	// ResolverNormal uses tier-1 + tier-2 constrained fallbacks (default).
	ResolverNormal
	// ResolverPermissive uses all tiers including global fallbacks.
	// Best effort for vendor MIBs with spec violations.
	ResolverPermissive
)

func (s ResolverStrictness) String() string {
	switch s {
	case ResolverStrict:
		return "strict"
	case ResolverNormal:
		return "normal"
	case ResolverPermissive:
		return "permissive"
	default:
		return fmt.Sprintf("ResolverStrictness(%d)", s)
	}
}

// AllowConstrainedFallbacks reports whether tier-2 constrained fallbacks are enabled.
func (s ResolverStrictness) AllowConstrainedFallbacks() bool {
	return s != ResolverStrict
}

// AllowGlobalFallbacks reports whether tier-3 global fallbacks are enabled.
func (s ResolverStrictness) AllowGlobalFallbacks() bool {
	return s == ResolverPermissive
}

// ReportingLevel controls diagnostic reporting verbosity.
type ReportingLevel int

const (
	ReportingSilent ReportingLevel = iota
	ReportingQuiet
	ReportingDefault
	ReportingVerbose
)

func (l ReportingLevel) String() string {
	switch l {
	case ReportingVerbose:
		return "verbose"
	case ReportingDefault:
		return "default"
	case ReportingQuiet:
		return "quiet"
	case ReportingSilent:
		return "silent"
	default:
		return fmt.Sprintf("ReportingLevel(%d)", l)
	}
}

// Kind identifies what an OID node represents.
type Kind int

const (
	KindUnknown      Kind = iota
	KindInternal          // internal node without a definition
	KindNode              // OBJECT-IDENTITY, MODULE-IDENTITY, value assignment
	KindScalar            // scalar OBJECT-TYPE
	KindTable             // table (SEQUENCE OF)
	KindRow               // row (has INDEX or AUGMENTS)
	KindColumn            // column (child of row)
	KindNotification      // NOTIFICATION-TYPE or TRAP-TYPE
	KindGroup             // OBJECT-GROUP or NOTIFICATION-GROUP
	KindCompliance        // MODULE-COMPLIANCE
	KindCapability        // AGENT-CAPABILITIES
)

var kindNames = [...]string{
	"unknown", "internal", "node", "scalar", "table", "row", "column",
	"notification", "group", "compliance", "capabilities",
}

func (k Kind) String() string { return enumString(k, kindNames[:], "Kind") }

// IsObjectType reports whether this is a scalar/table/row/column.
func (k Kind) IsObjectType() bool {
	switch k {
	case KindScalar, KindTable, KindRow, KindColumn:
		return true
	default:
		return false
	}
}

// IsConformance reports whether this is a group/compliance/capabilities node.
func (k Kind) IsConformance() bool {
	switch k {
	case KindGroup, KindCompliance, KindCapability:
		return true
	default:
		return false
	}
}

// Access represents the access level for OBJECT-TYPE definitions.
// Includes SMIv1, SMIv2, and AGENT-CAPABILITIES values.
type Access int

const (
	AccessNotAccessible       Access = iota // both: not directly accessible
	AccessAccessibleForNotify               // SMIv2: only in notifications
	AccessReadOnly                          // both: GET only
	AccessReadWrite                         // both: GET and SET
	AccessReadCreate                        // SMIv2: GET, SET, row creation
	AccessWriteOnly                         // SMIv1: SET only (obsolete)
	// AGENT-CAPABILITIES specific
	AccessNotImplemented // variation: not supported
)

var accessNames = [...]string{
	"not-accessible", "accessible-for-notify", "read-only", "read-write",
	"read-create", "write-only", "not-implemented",
}

func (a Access) String() string { return enumString(a, accessNames[:], "Access") }

// AccessNames returns the access value names in order (not-accessible..not-implemented).
func AccessNames() []string { return slices.Clone(accessNames[:]) }

// Status represents the lifecycle state of a MIB definition.
// Preserves SMIv1-specific values (mandatory, optional) without normalizing.
type Status int

const (
	StatusCurrent    Status = iota // SMIv2: active definition
	StatusDeprecated               // SMIv2: being phased out
	StatusObsolete                 // both: no longer used
	StatusMandatory                // SMIv1: agent MUST implement
	StatusOptional                 // SMIv1: agent MAY implement
)

var statusNames = [...]string{"current", "deprecated", "obsolete", "mandatory", "optional"}

func (s Status) String() string { return enumString(s, statusNames[:], "Status") }

// StatusNames returns the status value names in order (current..optional).
func StatusNames() []string { return slices.Clone(statusNames[:]) }

// IsSMIv1 reports whether this is an SMIv1-specific status value.
func (s Status) IsSMIv1() bool {
	return s == StatusMandatory || s == StatusOptional
}

// Language identifies the SMI version of a module.
type Language int

const (
	LanguageUnknown Language = iota
	LanguageSMIv1
	LanguageSMIv2
)

var languageNames = [...]string{"unknown", "SMIv1", "SMIv2"}

func (l Language) String() string { return enumString(l, languageNames[:], "Language") }

// BaseType identifies the fundamental SMI type.
type BaseType int

const (
	BaseUnknown BaseType = iota
	BaseInteger32
	BaseUnsigned32
	BaseCounter32
	BaseCounter64
	BaseGauge32
	BaseTimeTicks
	BaseIpAddress
	BaseOctetString
	BaseObjectIdentifier
	BaseBits
	BaseOpaque
	BaseSequence // For table row SEQUENCE types
)

var baseTypeNames = [...]string{
	"unknown", "Integer32", "Unsigned32", "Counter32", "Counter64",
	"Gauge32", "TimeTicks", "IpAddress", "OCTET STRING", "OBJECT IDENTIFIER",
	"BITS", "Opaque", "SEQUENCE",
}

func (b BaseType) String() string { return enumString(b, baseTypeNames[:], "BaseType") }

// IndexEncoding classifies how an INDEX component maps to instance-identifier
// sub-identifiers per RFC 2578 s7.7.
type IndexEncoding int

const (
	IndexEncodingUnknown        IndexEncoding = iota // cannot determine (bare type, unresolved, etc.)
	IndexEncodingInteger                             // 1 sub-identifier (integer-valued types)
	IndexEncodingFixedString                         // n sub-identifiers, n = fixed SIZE
	IndexEncodingLengthPrefixed                      // n+1 sub-identifiers, length prefix
	IndexEncodingImplied                             // n sub-identifiers, no prefix (IMPLIED)
	IndexEncodingIpAddress                           // 4 sub-identifiers
)

var indexEncodingNames = [...]string{
	"unknown", "integer", "fixed-string", "length-prefixed", "implied", "ip-address",
}

func (e IndexEncoding) String() string { return enumString(e, indexEncodingNames[:], "IndexEncoding") }

// AccessKeyword records which keyword was used (ACCESS, MAX-ACCESS, etc.).
type AccessKeyword int

const (
	AccessKeywordAccess    AccessKeyword = iota // SMIv1: ACCESS
	AccessKeywordMaxAccess                      // SMIv2: MAX-ACCESS
	AccessKeywordMinAccess                      // SMIv2: MIN-ACCESS (compliance)
)

var accessKeywordNames = [...]string{
	"ACCESS", "MAX-ACCESS", "MIN-ACCESS",
}

func (k AccessKeyword) String() string { return enumString(k, accessKeywordNames[:], "AccessKeyword") }
