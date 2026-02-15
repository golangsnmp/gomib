// Package lexer provides tokenization for SMIv1/SMIv2 MIB source text.
package lexer

import (
	"github.com/golangsnmp/gomib/internal/types"
)

// Token represents a lexed unit with its classification and source location.
type Token struct {
	Kind TokenKind
	Span types.Span
}

// NewToken creates a Token from a kind and source span.
func NewToken(kind TokenKind, span types.Span) Token {
	return Token{Kind: kind, Span: span}
}

// TokenKind classifies a token (punctuation, keyword, literal, etc.).
type TokenKind int

const (
	// Special

	TokError TokenKind = iota
	TokEOF
	TokForbiddenKeyword

	// Identifiers

	TokUppercaseIdent
	TokLowercaseIdent

	// Literals

	TokNumber
	TokNegativeNumber
	TokQuotedString
	TokHexString
	TokBinString

	// Single-character punctuation

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

	// Multi-character operators

	TokDotDot
	TokColonColonEqual

	// Structural keywords

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

	// Clause keywords

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

	// MACRO invocation keywords

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

	// Type keywords

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

	// SMIv1 type aliases

	TokKwCounter
	TokKwGauge
	TokKwNetworkAddress

	// ASN.1 tag keywords

	TokKwApplication
	TokKwImplicit
	TokKwUniversal

	// Status/Access value keywords

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

	// Sentinels for keyword range. New keywords must be added above this line.
	tokKeywordEnd // must remain last
)

// IsKeyword reports whether k is any keyword token.
func (k TokenKind) IsKeyword() bool {
	return k >= TokKwDefinitions && k < tokKeywordEnd
}

// IsIdentifier reports whether k is an identifier token (uppercase or lowercase).
func (k TokenKind) IsIdentifier() bool {
	return k == TokUppercaseIdent || k == TokLowercaseIdent
}

// IsTypeKeyword reports whether k is a built-in type keyword
// (INTEGER, Counter32, OCTET STRING components, etc.).
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

// IsMacroKeyword reports whether k is a macro invocation keyword
// (OBJECT-TYPE, MODULE-IDENTITY, etc.).
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

// LibsmiName returns the libsmi-compatible name for this token kind.
func (k TokenKind) LibsmiName() string {
	switch k {
	case TokError:
		return "ERROR"
	case TokEOF:
		return "EOF"
	case TokForbiddenKeyword:
		return "FORBIDDEN_KEYWORD"
	case TokUppercaseIdent:
		return "UPPERCASE_IDENTIFIER"
	case TokLowercaseIdent:
		return "LOWERCASE_IDENTIFIER"
	case TokNumber:
		return "NUMBER"
	case TokNegativeNumber:
		return "NEGATIVENUMBER"
	case TokQuotedString:
		return "QUOTED_STRING"
	case TokHexString:
		return "HEX_STRING"
	case TokBinString:
		return "BIN_STRING"
	case TokLBracket:
		return "LBRACKET"
	case TokRBracket:
		return "RBRACKET"
	case TokLBrace:
		return "LBRACE"
	case TokRBrace:
		return "RBRACE"
	case TokLParen:
		return "LPAREN"
	case TokRParen:
		return "RPAREN"
	case TokColon:
		return "COLON"
	case TokSemicolon:
		return "SEMICOLON"
	case TokComma:
		return "COMMA"
	case TokDot:
		return "DOT"
	case TokPipe:
		return "PIPE"
	case TokMinus:
		return "MINUS"
	case TokDotDot:
		return "DOT_DOT"
	case TokColonColonEqual:
		return "COLON_COLON_EQUAL"
	case TokKwDefinitions:
		return "DEFINITIONS"
	case TokKwBegin:
		return "BEGIN"
	case TokKwEnd:
		return "END"
	case TokKwImports:
		return "IMPORTS"
	case TokKwExports:
		return "EXPORTS"
	case TokKwFrom:
		return "FROM"
	case TokKwObject:
		return "OBJECT"
	case TokKwIdentifier:
		return "IDENTIFIER"
	case TokKwSequence:
		return "SEQUENCE"
	case TokKwOf:
		return "OF"
	case TokKwChoice:
		return "CHOICE"
	case TokKwMacro:
		return "MACRO"
	case TokKwSyntax:
		return "SYNTAX"
	case TokKwMaxAccess:
		return "MAX_ACCESS"
	case TokKwMinAccess:
		return "MIN_ACCESS"
	case TokKwAccess:
		return "ACCESS"
	case TokKwStatus:
		return "STATUS"
	case TokKwDescription:
		return "DESCRIPTION"
	case TokKwReference:
		return "REFERENCE"
	case TokKwIndex:
		return "INDEX"
	case TokKwDefval:
		return "DEFVAL"
	case TokKwAugments:
		return "AUGMENTS"
	case TokKwUnits:
		return "UNITS"
	case TokKwDisplayHint:
		return "DISPLAY_HINT"
	case TokKwObjects:
		return "OBJECTS"
	case TokKwNotifications:
		return "NOTIFICATIONS"
	case TokKwModule:
		return "MODULE"
	case TokKwMandatoryGroups:
		return "MANDATORY_GROUPS"
	case TokKwGroup:
		return "GROUP"
	case TokKwWriteSyntax:
		return "WRITE_SYNTAX"
	case TokKwProductRelease:
		return "PRODUCT_RELEASE"
	case TokKwSupports:
		return "SUPPORTS"
	case TokKwIncludes:
		return "INCLUDES"
	case TokKwVariation:
		return "VARIATION"
	case TokKwCreationRequires:
		return "CREATION_REQUIRES"
	case TokKwRevision:
		return "REVISION"
	case TokKwLastUpdated:
		return "LAST_UPDATED"
	case TokKwOrganization:
		return "ORGANIZATION"
	case TokKwContactInfo:
		return "CONTACT_INFO"
	case TokKwImplied:
		return "IMPLIED"
	case TokKwSize:
		return "SIZE"
	case TokKwEnterprise:
		return "ENTERPRISE"
	case TokKwVariables:
		return "VARIABLES"
	case TokKwModuleIdentity:
		return "MODULE_IDENTITY"
	case TokKwModuleCompliance:
		return "MODULE_COMPLIANCE"
	case TokKwObjectGroup:
		return "OBJECT_GROUP"
	case TokKwNotificationGroup:
		return "NOTIFICATION_GROUP"
	case TokKwAgentCapabilities:
		return "AGENT_CAPABILITIES"
	case TokKwObjectType:
		return "OBJECT_TYPE"
	case TokKwObjectIdentity:
		return "OBJECT_IDENTITY"
	case TokKwNotificationType:
		return "NOTIFICATION_TYPE"
	case TokKwTextualConvention:
		return "TEXTUAL_CONVENTION"
	case TokKwTrapType:
		return "TRAP_TYPE"
	case TokKwInteger:
		return "INTEGER"
	case TokKwInteger32:
		return "INTEGER32"
	case TokKwUnsigned32:
		return "UNSIGNED32"
	case TokKwCounter32:
		return "COUNTER32"
	case TokKwCounter64:
		return "COUNTER64"
	case TokKwGauge32:
		return "GAUGE32"
	case TokKwIpAddress:
		return "IPADDRESS"
	case TokKwOpaque:
		return "OPAQUE"
	case TokKwTimeTicks:
		return "TIMETICKS"
	case TokKwBits:
		return "BITS"
	case TokKwOctet:
		return "OCTET"
	case TokKwString:
		return "STRING"
	case TokKwCounter:
		return "COUNTER"
	case TokKwGauge:
		return "GAUGE"
	case TokKwNetworkAddress:
		return "NETWORKADDRESS"
	case TokKwApplication:
		return "APPLICATION"
	case TokKwImplicit:
		return "IMPLICIT"
	case TokKwUniversal:
		return "UNIVERSAL"
	case TokKwCurrent:
		return "CURRENT"
	case TokKwDeprecated:
		return "DEPRECATED"
	case TokKwObsolete:
		return "OBSOLETE"
	case TokKwMandatory:
		return "MANDATORY"
	case TokKwOptional:
		return "OPTIONAL"
	case TokKwReadOnly:
		return "READ_ONLY"
	case TokKwReadWrite:
		return "READ_WRITE"
	case TokKwReadCreate:
		return "READ_CREATE"
	case TokKwWriteOnly:
		return "WRITE_ONLY"
	case TokKwNotAccessible:
		return "NOT_ACCESSIBLE"
	case TokKwAccessibleForNotify:
		return "ACCESSIBLE_FOR_NOTIFY"
	case TokKwNotImplemented:
		return "NOT_IMPLEMENTED"
	default:
		return "UNKNOWN"
	}
}
