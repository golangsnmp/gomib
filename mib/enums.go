// Package mib provides MIB parsing and querying for SNMP management.
package mib

import "fmt"

// Severity levels for diagnostics (libsmi-compatible).
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

func (s Severity) String() string {
	switch s {
	case SeverityFatal:
		return "fatal"
	case SeveritySevere:
		return "severe"
	case SeverityError:
		return "error"
	case SeverityMinor:
		return "minor"
	case SeverityStyle:
		return "style"
	case SeverityWarning:
		return "warning"
	case SeverityInfo:
		return "info"
	default:
		return fmt.Sprintf("Severity(%d)", s)
	}
}

// StrictnessLevel defines preset strictness configurations.
type StrictnessLevel int

const (
	StrictnessStrict     StrictnessLevel = 0 // RFC-only, reject non-compliant
	StrictnessNormal     StrictnessLevel = 3 // Default, warn on issues
	StrictnessPermissive StrictnessLevel = 5 // Accept most real-world MIBs
	StrictnessSilent     StrictnessLevel = 6 // Accept everything, minimal output
)

func (l StrictnessLevel) String() string {
	switch l {
	case StrictnessStrict:
		return "strict"
	case StrictnessNormal:
		return "normal"
	case StrictnessPermissive:
		return "permissive"
	case StrictnessSilent:
		return "silent"
	default:
		return fmt.Sprintf("StrictnessLevel(%d)", l)
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
	KindCapabilities      // AGENT-CAPABILITIES
)

func (k Kind) String() string {
	switch k {
	case KindUnknown:
		return "unknown"
	case KindInternal:
		return "internal"
	case KindNode:
		return "node"
	case KindScalar:
		return "scalar"
	case KindTable:
		return "table"
	case KindRow:
		return "row"
	case KindColumn:
		return "column"
	case KindNotification:
		return "notification"
	case KindGroup:
		return "group"
	case KindCompliance:
		return "compliance"
	case KindCapabilities:
		return "capabilities"
	default:
		return fmt.Sprintf("Kind(%d)", k)
	}
}

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
	case KindGroup, KindCompliance, KindCapabilities:
		return true
	default:
		return false
	}
}

// Access levels for OBJECT-TYPE definitions.
// Includes SMIv1, SMIv2, SPPI (RFC 3159), and AGENT-CAPABILITIES values.
type Access int

const (
	AccessNotAccessible       Access = iota // both: not directly accessible
	AccessAccessibleForNotify               // SMIv2: only in notifications
	AccessReadOnly                          // both: GET only
	AccessReadWrite                         // both: GET and SET
	AccessReadCreate                        // SMIv2: GET, SET, row creation
	AccessWriteOnly                         // SMIv1: SET only (obsolete)
	// SPPI-specific (RFC 3159)
	AccessInstall       // SPPI: can be installed
	AccessInstallNotify // SPPI: install + notify
	AccessReportOnly    // SPPI: reporting only
	// AGENT-CAPABILITIES specific
	AccessNotImplemented // variation: not supported
)

func (a Access) String() string {
	switch a {
	case AccessNotAccessible:
		return "not-accessible"
	case AccessAccessibleForNotify:
		return "accessible-for-notify"
	case AccessReadOnly:
		return "read-only"
	case AccessReadWrite:
		return "read-write"
	case AccessReadCreate:
		return "read-create"
	case AccessWriteOnly:
		return "write-only"
	case AccessInstall:
		return "install"
	case AccessInstallNotify:
		return "install-notify"
	case AccessReportOnly:
		return "report-only"
	case AccessNotImplemented:
		return "not-implemented"
	default:
		return fmt.Sprintf("Access(%d)", a)
	}
}

// AccessKeyword indicates which keyword was used in the source MIB.
// Preserved for accurate MIB generation.
type AccessKeyword int

const (
	AccessKeywordAccess    AccessKeyword = iota // SMIv1: ACCESS
	AccessKeywordMaxAccess                      // SMIv2: MAX-ACCESS
	AccessKeywordMinAccess                      // SMIv2: MIN-ACCESS (compliance)
	AccessKeywordPibAccess                      // SPPI: PIB-ACCESS
)

func (k AccessKeyword) String() string {
	switch k {
	case AccessKeywordAccess:
		return "ACCESS"
	case AccessKeywordMaxAccess:
		return "MAX-ACCESS"
	case AccessKeywordMinAccess:
		return "MIN-ACCESS"
	case AccessKeywordPibAccess:
		return "PIB-ACCESS"
	default:
		return fmt.Sprintf("AccessKeyword(%d)", k)
	}
}

// Status values for MIB definitions.
// Preserves SMIv1-specific values (mandatory, optional) without normalizing.
type Status int

const (
	StatusCurrent    Status = iota // SMIv2: active definition
	StatusDeprecated               // SMIv2: being phased out
	StatusObsolete                 // both: no longer used
	StatusMandatory                // SMIv1: agent MUST implement
	StatusOptional                 // SMIv1: agent MAY implement
)

func (s Status) String() string {
	switch s {
	case StatusCurrent:
		return "current"
	case StatusDeprecated:
		return "deprecated"
	case StatusObsolete:
		return "obsolete"
	case StatusMandatory:
		return "mandatory"
	case StatusOptional:
		return "optional"
	default:
		return fmt.Sprintf("Status(%d)", s)
	}
}

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
	LanguageSPPI // Policy Information Base (RFC 3159)
)

func (l Language) String() string {
	switch l {
	case LanguageUnknown:
		return "unknown"
	case LanguageSMIv1:
		return "SMIv1"
	case LanguageSMIv2:
		return "SMIv2"
	case LanguageSPPI:
		return "SPPI"
	default:
		return fmt.Sprintf("Language(%d)", l)
	}
}

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

func (b BaseType) String() string {
	switch b {
	case BaseUnknown:
		return "unknown"
	case BaseInteger32:
		return "Integer32"
	case BaseUnsigned32:
		return "Unsigned32"
	case BaseCounter32:
		return "Counter32"
	case BaseCounter64:
		return "Counter64"
	case BaseGauge32:
		return "Gauge32"
	case BaseTimeTicks:
		return "TimeTicks"
	case BaseIpAddress:
		return "IpAddress"
	case BaseOctetString:
		return "OCTET STRING"
	case BaseObjectIdentifier:
		return "OBJECT IDENTIFIER"
	case BaseBits:
		return "BITS"
	case BaseOpaque:
		return "Opaque"
	case BaseSequence:
		return "SEQUENCE"
	default:
		return fmt.Sprintf("BaseType(%d)", b)
	}
}
