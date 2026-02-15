// Package mib defines the public types and interfaces for MIB data.
package mib

import "github.com/golangsnmp/gomib/internal/types"

// Severity indicates how serious a diagnostic issue is (libsmi-compatible).
// Lower values are more severe.
type Severity = types.Severity

const (
	SeverityFatal   = types.SeverityFatal
	SeveritySevere  = types.SeveritySevere
	SeverityError   = types.SeverityError
	SeverityMinor   = types.SeverityMinor
	SeverityStyle   = types.SeverityStyle
	SeverityWarning = types.SeverityWarning
	SeverityInfo    = types.SeverityInfo
)

// StrictnessLevel defines preset strictness configurations.
type StrictnessLevel = types.StrictnessLevel

const (
	StrictnessStrict     = types.StrictnessStrict
	StrictnessNormal     = types.StrictnessNormal
	StrictnessPermissive = types.StrictnessPermissive
	StrictnessSilent     = types.StrictnessSilent
)

// Kind identifies what an OID node represents.
type Kind = types.Kind

const (
	KindUnknown      = types.KindUnknown
	KindInternal     = types.KindInternal
	KindNode         = types.KindNode
	KindScalar       = types.KindScalar
	KindTable        = types.KindTable
	KindRow          = types.KindRow
	KindColumn       = types.KindColumn
	KindNotification = types.KindNotification
	KindGroup        = types.KindGroup
	KindCompliance   = types.KindCompliance
	KindCapability   = types.KindCapability
)

// Access represents the access level for OBJECT-TYPE definitions.
// Includes SMIv1, SMIv2, SPPI (RFC 3159), and AGENT-CAPABILITIES values.
type Access = types.Access

const (
	AccessNotAccessible       = types.AccessNotAccessible
	AccessAccessibleForNotify = types.AccessAccessibleForNotify
	AccessReadOnly            = types.AccessReadOnly
	AccessReadWrite           = types.AccessReadWrite
	AccessReadCreate          = types.AccessReadCreate
	AccessWriteOnly           = types.AccessWriteOnly
	AccessInstall             = types.AccessInstall
	AccessInstallNotify       = types.AccessInstallNotify
	AccessReportOnly          = types.AccessReportOnly
	AccessNotImplemented      = types.AccessNotImplemented
)

// Status represents the lifecycle state of a MIB definition.
// Preserves SMIv1-specific values (mandatory, optional) without normalizing.
type Status = types.Status

const (
	StatusCurrent    = types.StatusCurrent
	StatusDeprecated = types.StatusDeprecated
	StatusObsolete   = types.StatusObsolete
	StatusMandatory  = types.StatusMandatory
	StatusOptional   = types.StatusOptional
)

// Language identifies the SMI version of a module.
type Language = types.Language

const (
	LanguageUnknown = types.LanguageUnknown
	LanguageSMIv1   = types.LanguageSMIv1
	LanguageSMIv2   = types.LanguageSMIv2
	LanguageSPPI    = types.LanguageSPPI
)

// BaseType identifies the fundamental SMI type.
type BaseType = types.BaseType

const (
	BaseUnknown          = types.BaseUnknown
	BaseInteger32        = types.BaseInteger32
	BaseUnsigned32       = types.BaseUnsigned32
	BaseCounter32        = types.BaseCounter32
	BaseCounter64        = types.BaseCounter64
	BaseGauge32          = types.BaseGauge32
	BaseTimeTicks        = types.BaseTimeTicks
	BaseIpAddress        = types.BaseIpAddress
	BaseOctetString      = types.BaseOctetString
	BaseObjectIdentifier = types.BaseObjectIdentifier
	BaseBits             = types.BaseBits
	BaseOpaque           = types.BaseOpaque
	BaseSequence         = types.BaseSequence
)
