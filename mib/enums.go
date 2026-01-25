// Package gomib provides MIB parsing and querying for SNMP management.
package mib

import "fmt"

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
type Access int

const (
	AccessNotAccessible Access = iota
	AccessAccessibleForNotify
	AccessReadOnly
	AccessReadWrite
	AccessReadCreate
	AccessWriteOnly
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
	default:
		return fmt.Sprintf("Access(%d)", a)
	}
}

// Status values for MIB definitions.
type Status int

const (
	StatusCurrent Status = iota
	StatusDeprecated
	StatusObsolete
)

func (s Status) String() string {
	switch s {
	case StatusCurrent:
		return "current"
	case StatusDeprecated:
		return "deprecated"
	case StatusObsolete:
		return "obsolete"
	default:
		return fmt.Sprintf("Status(%d)", s)
	}
}

// Language identifies the SMI version of a module.
type Language int

const (
	LanguageSMIv1 Language = iota
	LanguageSMIv2
)

func (l Language) String() string {
	switch l {
	case LanguageSMIv1:
		return "SMIv1"
	case LanguageSMIv2:
		return "SMIv2"
	default:
		return fmt.Sprintf("Language(%d)", l)
	}
}

// BaseType identifies the fundamental SMI type.
type BaseType int

const (
	BaseInteger32 BaseType = iota
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
)

func (b BaseType) String() string {
	switch b {
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
	default:
		return fmt.Sprintf("BaseType(%d)", b)
	}
}

// Severity for diagnostics.
type Severity int

const (
	SeverityInfo Severity = iota
	SeverityWarning
	SeverityError
)

func (s Severity) String() string {
	switch s {
	case SeverityInfo:
		return "info"
	case SeverityWarning:
		return "warning"
	case SeverityError:
		return "error"
	default:
		return fmt.Sprintf("Severity(%d)", s)
	}
}
