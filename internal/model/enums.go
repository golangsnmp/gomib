package model

import "github.com/golangsnmp/gomib/internal/types"

// Severity indicates how serious a diagnostic issue is (libsmi-compatible).
// Lower values are more severe.
type Severity = types.Severity

// SeverityNames returns the severity level names in order (fatal..info).
func SeverityNames() []string { return types.SeverityNames() }

// StatusNames returns the status value names in order (current..optional).
func StatusNames() []string { return types.StatusNames() }

// AccessNames returns the access value names in order (not-accessible..not-implemented).
func AccessNames() []string { return types.AccessNames() }

const (
	SeverityFatal   = types.SeverityFatal
	SeveritySevere  = types.SeveritySevere
	SeverityError   = types.SeverityError
	SeverityMinor   = types.SeverityMinor
	SeverityStyle   = types.SeverityStyle
	SeverityWarning = types.SeverityWarning
	SeverityInfo    = types.SeverityInfo
)

// ResolverStrictness controls resolver fallback behavior.
type ResolverStrictness = types.ResolverStrictness

const (
	ResolverStrict     = types.ResolverStrict
	ResolverNormal     = types.ResolverNormal
	ResolverPermissive = types.ResolverPermissive
)

// ReportingLevel controls diagnostic reporting verbosity.
type ReportingLevel = types.ReportingLevel

const (
	ReportingSilent  = types.ReportingSilent
	ReportingQuiet   = types.ReportingQuiet
	ReportingDefault = types.ReportingDefault
	ReportingVerbose = types.ReportingVerbose
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
// Includes SMIv1, SMIv2, and AGENT-CAPABILITIES values.
type Access = types.Access

const (
	AccessNotAccessible       = types.AccessNotAccessible
	AccessAccessibleForNotify = types.AccessAccessibleForNotify
	AccessReadOnly            = types.AccessReadOnly
	AccessReadWrite           = types.AccessReadWrite
	AccessReadCreate          = types.AccessReadCreate
	AccessWriteOnly           = types.AccessWriteOnly
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

// MacroInfo describes an SMI macro keyword.
type MacroInfo = types.MacroInfo

// MacroDescription returns info about an SMI macro keyword, if known.
func MacroDescription(name string) (MacroInfo, bool) { return types.MacroDescription(name) }

// ClauseInfo describes an SMI clause keyword used within a macro definition.
type ClauseInfo = types.ClauseInfo

// ClauseDescription returns info about an SMI clause keyword, if known.
func ClauseDescription(name string) (ClauseInfo, bool) { return types.ClauseDescription(name) }

// ByteOffset is a byte position in source text.
type ByteOffset = types.ByteOffset

// Span is a half-open byte range [Start, End) in source text.
type Span = types.Span

// IndexEncoding classifies how an INDEX component maps to instance-identifier
// sub-identifiers per RFC 2578 s7.7.
type IndexEncoding = types.IndexEncoding

const (
	IndexEncodingUnknown        = types.IndexEncodingUnknown
	IndexEncodingInteger        = types.IndexEncodingInteger
	IndexEncodingFixedString    = types.IndexEncodingFixedString
	IndexEncodingLengthPrefixed = types.IndexEncodingLengthPrefixed
	IndexEncodingImplied        = types.IndexEncodingImplied
	IndexEncodingIpAddress      = types.IndexEncodingIpAddress
)
