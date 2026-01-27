// Package lexer provides tokenization for SMIv1/SMIv2 MIB source text.
package lexer

import (
	"github.com/golangsnmp/gomib/internal/types"
)

// Token is a token with kind and source span.
type Token struct {
	Kind TokenKind
	Span types.Span
}

// NewToken creates a new token.
func NewToken(kind TokenKind, span types.Span) Token {
	return Token{Kind: kind, Span: span}
}

// TokenKind identifies a token type.
type TokenKind int

const (
	// === Special ===

	TokError TokenKind = iota
	TokEOF
	TokForbiddenKeyword

	// === Identifiers ===

	TokUppercaseIdent
	TokLowercaseIdent

	// === Literals ===

	TokNumber
	TokNegativeNumber
	TokQuotedString
	TokHexString
	TokBinString

	// === Single-character punctuation ===

	TokLBracket
	TokRBracket
	TokLBrace
	TokRBrace
	TokLParen
	TokRParen
	TokColon
	TokSemicolon
	TokComma
	TokDot
	TokPipe
	TokMinus

	// === Multi-character operators ===

	TokDotDot
	TokColonColonEqual

	// === Structural keywords ===

	TokKwDefinitions
	TokKwBegin
	TokKwEnd
	TokKwImports
	TokKwExports
	TokKwFrom
	TokKwObject
	TokKwIdentifier
	TokKwSequence
	TokKwOf
	TokKwChoice
	TokKwMacro

	// === Clause keywords ===

	TokKwSyntax
	TokKwMaxAccess
	TokKwMinAccess
	TokKwAccess
	TokKwStatus
	TokKwDescription
	TokKwReference
	TokKwIndex
	TokKwDefval
	TokKwAugments
	TokKwUnits
	TokKwDisplayHint
	TokKwObjects
	TokKwNotifications
	TokKwModule
	TokKwMandatoryGroups
	TokKwGroup
	TokKwWriteSyntax
	TokKwProductRelease
	TokKwSupports
	TokKwIncludes
	TokKwVariation
	TokKwCreationRequires
	TokKwRevision
	TokKwLastUpdated
	TokKwOrganization
	TokKwContactInfo
	TokKwImplied
	TokKwSize
	TokKwEnterprise
	TokKwVariables

	// === MACRO invocation keywords ===

	TokKwModuleIdentity
	TokKwModuleCompliance
	TokKwObjectGroup
	TokKwNotificationGroup
	TokKwAgentCapabilities
	TokKwObjectType
	TokKwObjectIdentity
	TokKwNotificationType
	TokKwTextualConvention
	TokKwTrapType

	// === Type keywords ===

	TokKwInteger
	TokKwInteger32
	TokKwUnsigned32
	TokKwCounter32
	TokKwCounter64
	TokKwGauge32
	TokKwIpAddress
	TokKwOpaque
	TokKwTimeTicks
	TokKwBits
	TokKwOctet
	TokKwString

	// === SMIv1 type aliases ===

	TokKwCounter
	TokKwGauge
	TokKwNetworkAddress

	// === ASN.1 tag keywords ===

	TokKwApplication
	TokKwImplicit
	TokKwUniversal

	// === Status/Access value keywords ===

	TokKwCurrent
	TokKwDeprecated
	TokKwObsolete
	TokKwMandatory
	TokKwOptional
	TokKwReadOnly
	TokKwReadWrite
	TokKwReadCreate
	TokKwWriteOnly
	TokKwNotAccessible
	TokKwAccessibleForNotify
	TokKwNotImplemented
)

// IsKeyword returns true if this token is a keyword.
func (k TokenKind) IsKeyword() bool {
	return k >= TokKwDefinitions && k <= TokKwNotImplemented
}

// IsTypeKeyword returns true if this token is a type keyword.
func (k TokenKind) IsTypeKeyword() bool {
	switch k {
	case TokKwInteger, TokKwInteger32, TokKwUnsigned32, TokKwCounter32,
		TokKwCounter64, TokKwGauge32, TokKwIpAddress, TokKwOpaque,
		TokKwTimeTicks, TokKwBits, TokKwOctet, TokKwString,
		TokKwCounter, TokKwGauge, TokKwNetworkAddress:
		return true
	default:
		return false
	}
}

// IsMacroKeyword returns true if this token is a macro keyword.
func (k TokenKind) IsMacroKeyword() bool {
	switch k {
	case TokKwModuleIdentity, TokKwModuleCompliance, TokKwObjectGroup,
		TokKwNotificationGroup, TokKwAgentCapabilities, TokKwObjectType,
		TokKwObjectIdentity, TokKwNotificationType, TokKwTextualConvention,
		TokKwTrapType:
		return true
	default:
		return false
	}
}
