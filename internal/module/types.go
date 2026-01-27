// Package module provides a normalized representation of MIB modules.
//
// This package transforms AST structures into a simplified module representation
// independent of whether the source was SMIv1 or SMIv2. Key transformations:
//
//   - Language detection from imports
//   - Import flattening (one symbol per import)
//   - Unified notification type (TRAP-TYPE and NOTIFICATION-TYPE)
//
// # What Lowering Does NOT Do (per V2 design)
//
// Status and Access values are preserved without normalization:
//   - STATUS: mandatory, optional preserved (not mapped to current/deprecated)
//   - ACCESS: SPPI values preserved (install, install-notify, report-only)
//   - ACCESS keyword preserved (ACCESS vs MAX-ACCESS vs PIB-ACCESS)
//
// These are resolver responsibilities:
//   - OID resolution (keeps OID components as symbols)
//   - Type resolution (keeps type references as symbols)
//   - Nodekind inference (requires resolved OID tree)
//   - Import resolution (just normalize; actual lookup is resolver's job)
//   - Built-in type injection
package module

import "github.com/golangsnmp/gomib/mib"

// Language is an alias to the mib type.
type Language = mib.Language

// Re-export constants for convenience.
const (
	LanguageUnknown = mib.LanguageUnknown
	LanguageSMIv1   = mib.LanguageSMIv1
	LanguageSMIv2   = mib.LanguageSMIv2
	LanguageSPPI    = mib.LanguageSPPI
)

// Status is an alias to the mib type.
type Status = mib.Status

// Re-export status constants.
const (
	StatusCurrent    = mib.StatusCurrent
	StatusDeprecated = mib.StatusDeprecated
	StatusObsolete   = mib.StatusObsolete
	StatusMandatory  = mib.StatusMandatory
	StatusOptional   = mib.StatusOptional
)

// Access is an alias to the mib type.
type Access = mib.Access

// Re-export access constants.
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

// AccessKeyword is an alias to the mib type.
type AccessKeyword = mib.AccessKeyword

// Re-export access keyword constants.
const (
	AccessKeywordAccess    = mib.AccessKeywordAccess
	AccessKeywordMaxAccess = mib.AccessKeywordMaxAccess
	AccessKeywordMinAccess = mib.AccessKeywordMinAccess
	AccessKeywordPibAccess = mib.AccessKeywordPibAccess
)

// BaseType is an alias to the mib type.
type BaseType = mib.BaseType

// Re-export base type constants.
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
