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
// Derived from libsmi's scanner-smi.l lexer.
//
//go:generate stringer -type=TokenKind
type TokenKind int

const (
	// === Special ===

	// TokError is a lexical error.
	TokError TokenKind = iota
	// TokEOF is end of input.
	TokEOF
	// TokForbiddenKeyword is a forbidden ASN.1 keyword (FALSE, TRUE, NULL, etc.).
	TokForbiddenKeyword

	// === Identifiers ===

	// TokUppercaseIdent is an uppercase identifier (module names, type names).
	TokUppercaseIdent
	// TokLowercaseIdent is a lowercase identifier (object names, enum labels).
	TokLowercaseIdent

	// === Literals ===

	// TokNumber is an unsigned decimal number.
	TokNumber
	// TokNegativeNumber is a signed decimal number (negative).
	TokNegativeNumber
	// TokQuotedString is a quoted string literal.
	TokQuotedString
	// TokHexString is a hex string literal ('...'H).
	TokHexString
	// TokBinString is a binary string literal ('...'B).
	TokBinString

	// === Single-character punctuation ===

	// TokLBracket is '['.
	TokLBracket
	// TokRBracket is ']'.
	TokRBracket
	// TokLBrace is '{'.
	TokLBrace
	// TokRBrace is '}'.
	TokRBrace
	// TokLParen is '('.
	TokLParen
	// TokRParen is ')'.
	TokRParen
	// TokColon is ':'.
	TokColon
	// TokSemicolon is ';'.
	TokSemicolon
	// TokComma is ','.
	TokComma
	// TokDot is '.'.
	TokDot
	// TokPipe is '|'.
	TokPipe
	// TokMinus is '-'.
	TokMinus

	// === Multi-character operators ===

	// TokDotDot is '..'.
	TokDotDot
	// TokColonColonEqual is '::='.
	TokColonColonEqual

	// === Structural keywords ===

	// TokKwDefinitions is 'DEFINITIONS'.
	TokKwDefinitions
	// TokKwBegin is 'BEGIN'.
	TokKwBegin
	// TokKwEnd is 'END'.
	TokKwEnd
	// TokKwImports is 'IMPORTS'.
	TokKwImports
	// TokKwExports is 'EXPORTS'.
	TokKwExports
	// TokKwFrom is 'FROM'.
	TokKwFrom
	// TokKwObject is 'OBJECT'.
	TokKwObject
	// TokKwIdentifier is 'IDENTIFIER'.
	TokKwIdentifier
	// TokKwSequence is 'SEQUENCE'.
	TokKwSequence
	// TokKwOf is 'OF'.
	TokKwOf
	// TokKwChoice is 'CHOICE'.
	TokKwChoice
	// TokKwMacro is 'MACRO'.
	TokKwMacro

	// === Clause keywords ===

	// TokKwSyntax is 'SYNTAX'.
	TokKwSyntax
	// TokKwMaxAccess is 'MAX-ACCESS'.
	TokKwMaxAccess
	// TokKwMinAccess is 'MIN-ACCESS'.
	TokKwMinAccess
	// TokKwAccess is 'ACCESS'.
	TokKwAccess
	// TokKwStatus is 'STATUS'.
	TokKwStatus
	// TokKwDescription is 'DESCRIPTION'.
	TokKwDescription
	// TokKwReference is 'REFERENCE'.
	TokKwReference
	// TokKwIndex is 'INDEX'.
	TokKwIndex
	// TokKwDefval is 'DEFVAL'.
	TokKwDefval
	// TokKwAugments is 'AUGMENTS'.
	TokKwAugments
	// TokKwUnits is 'UNITS'.
	TokKwUnits
	// TokKwDisplayHint is 'DISPLAY-HINT'.
	TokKwDisplayHint
	// TokKwObjects is 'OBJECTS'.
	TokKwObjects
	// TokKwNotifications is 'NOTIFICATIONS'.
	TokKwNotifications
	// TokKwModule is 'MODULE'.
	TokKwModule
	// TokKwMandatoryGroups is 'MANDATORY-GROUPS'.
	TokKwMandatoryGroups
	// TokKwGroup is 'GROUP'.
	TokKwGroup
	// TokKwWriteSyntax is 'WRITE-SYNTAX'.
	TokKwWriteSyntax
	// TokKwProductRelease is 'PRODUCT-RELEASE'.
	TokKwProductRelease
	// TokKwSupports is 'SUPPORTS'.
	TokKwSupports
	// TokKwIncludes is 'INCLUDES'.
	TokKwIncludes
	// TokKwVariation is 'VARIATION'.
	TokKwVariation
	// TokKwCreationRequires is 'CREATION-REQUIRES'.
	TokKwCreationRequires
	// TokKwRevision is 'REVISION'.
	TokKwRevision
	// TokKwLastUpdated is 'LAST-UPDATED'.
	TokKwLastUpdated
	// TokKwOrganization is 'ORGANIZATION'.
	TokKwOrganization
	// TokKwContactInfo is 'CONTACT-INFO'.
	TokKwContactInfo
	// TokKwImplied is 'IMPLIED'.
	TokKwImplied
	// TokKwSize is 'SIZE'.
	TokKwSize
	// TokKwEnterprise is 'ENTERPRISE'.
	TokKwEnterprise
	// TokKwVariables is 'VARIABLES'.
	TokKwVariables

	// === MACRO invocation keywords ===

	// TokKwModuleIdentity is 'MODULE-IDENTITY'.
	TokKwModuleIdentity
	// TokKwModuleCompliance is 'MODULE-COMPLIANCE'.
	TokKwModuleCompliance
	// TokKwObjectGroup is 'OBJECT-GROUP'.
	TokKwObjectGroup
	// TokKwNotificationGroup is 'NOTIFICATION-GROUP'.
	TokKwNotificationGroup
	// TokKwAgentCapabilities is 'AGENT-CAPABILITIES'.
	TokKwAgentCapabilities
	// TokKwObjectType is 'OBJECT-TYPE'.
	TokKwObjectType
	// TokKwObjectIdentity is 'OBJECT-IDENTITY'.
	TokKwObjectIdentity
	// TokKwNotificationType is 'NOTIFICATION-TYPE'.
	TokKwNotificationType
	// TokKwTextualConvention is 'TEXTUAL-CONVENTION'.
	TokKwTextualConvention
	// TokKwTrapType is 'TRAP-TYPE'.
	TokKwTrapType

	// === Type keywords ===

	// TokKwInteger is 'INTEGER'.
	TokKwInteger
	// TokKwInteger32 is 'Integer32'.
	TokKwInteger32
	// TokKwUnsigned32 is 'Unsigned32'.
	TokKwUnsigned32
	// TokKwCounter32 is 'Counter32'.
	TokKwCounter32
	// TokKwCounter64 is 'Counter64'.
	TokKwCounter64
	// TokKwGauge32 is 'Gauge32'.
	TokKwGauge32
	// TokKwIpAddress is 'IpAddress'.
	TokKwIpAddress
	// TokKwOpaque is 'Opaque'.
	TokKwOpaque
	// TokKwTimeTicks is 'TimeTicks'.
	TokKwTimeTicks
	// TokKwBits is 'BITS'.
	TokKwBits
	// TokKwOctet is 'OCTET'.
	TokKwOctet
	// TokKwString is 'STRING'.
	TokKwString

	// === SMIv1 type aliases ===

	// TokKwCounter is 'Counter' (normalized to Counter32).
	TokKwCounter
	// TokKwGauge is 'Gauge' (normalized to Gauge32).
	TokKwGauge
	// TokKwNetworkAddress is 'NetworkAddress' (normalized to IpAddress).
	TokKwNetworkAddress

	// === ASN.1 tag keywords ===

	// TokKwApplication is 'APPLICATION'.
	TokKwApplication
	// TokKwImplicit is 'IMPLICIT'.
	TokKwImplicit
	// TokKwUniversal is 'UNIVERSAL'.
	TokKwUniversal

	// === Status/Access value keywords ===

	// TokKwCurrent is 'current'.
	TokKwCurrent
	// TokKwDeprecated is 'deprecated'.
	TokKwDeprecated
	// TokKwObsolete is 'obsolete'.
	TokKwObsolete
	// TokKwMandatory is 'mandatory' (v1 status).
	TokKwMandatory
	// TokKwOptional is 'optional' (v1 status).
	TokKwOptional
	// TokKwReadOnly is 'read-only'.
	TokKwReadOnly
	// TokKwReadWrite is 'read-write'.
	TokKwReadWrite
	// TokKwReadCreate is 'read-create'.
	TokKwReadCreate
	// TokKwWriteOnly is 'write-only' (deprecated).
	TokKwWriteOnly
	// TokKwNotAccessible is 'not-accessible'.
	TokKwNotAccessible
	// TokKwAccessibleForNotify is 'accessible-for-notify'.
	TokKwAccessibleForNotify
	// TokKwNotImplemented is 'not-implemented' (AGENT-CAPABILITIES).
	TokKwNotImplemented
)

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

// IsMacroKeyword returns true if this token is a macro keyword (OBJECT-TYPE, etc.).
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
