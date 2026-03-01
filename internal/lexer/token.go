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
	case TokKwInteger, TokKwUnsigned32, TokKwCounter32,
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

var libsmiNames = [...]string{
	TokError:                 "ERROR",
	TokEOF:                   "EOF",
	TokForbiddenKeyword:      "FORBIDDEN_KEYWORD",
	TokUppercaseIdent:        "UPPERCASE_IDENTIFIER",
	TokLowercaseIdent:        "LOWERCASE_IDENTIFIER",
	TokNumber:                "NUMBER",
	TokNegativeNumber:        "NEGATIVENUMBER",
	TokQuotedString:          "QUOTED_STRING",
	TokHexString:             "HEX_STRING",
	TokBinString:             "BIN_STRING",
	TokLBracket:              "LBRACKET",
	TokRBracket:              "RBRACKET",
	TokLBrace:                "LBRACE",
	TokRBrace:                "RBRACE",
	TokLParen:                "LPAREN",
	TokRParen:                "RPAREN",
	TokColon:                 "COLON",
	TokSemicolon:             "SEMICOLON",
	TokComma:                 "COMMA",
	TokDot:                   "DOT",
	TokPipe:                  "PIPE",
	TokMinus:                 "MINUS",
	TokDotDot:                "DOT_DOT",
	TokColonColonEqual:       "COLON_COLON_EQUAL",
	TokKwDefinitions:         "DEFINITIONS",
	TokKwBegin:               "BEGIN",
	TokKwEnd:                 "END",
	TokKwImports:             "IMPORTS",
	TokKwExports:             "EXPORTS",
	TokKwFrom:                "FROM",
	TokKwObject:              "OBJECT",
	TokKwIdentifier:          "IDENTIFIER",
	TokKwSequence:            "SEQUENCE",
	TokKwOf:                  "OF",
	TokKwChoice:              "CHOICE",
	TokKwMacro:               "MACRO",
	TokKwSyntax:              "SYNTAX",
	TokKwMaxAccess:           "MAX_ACCESS",
	TokKwMinAccess:           "MIN_ACCESS",
	TokKwAccess:              "ACCESS",
	TokKwStatus:              "STATUS",
	TokKwDescription:         "DESCRIPTION",
	TokKwReference:           "REFERENCE",
	TokKwIndex:               "INDEX",
	TokKwDefval:              "DEFVAL",
	TokKwAugments:            "AUGMENTS",
	TokKwUnits:               "UNITS",
	TokKwDisplayHint:         "DISPLAY_HINT",
	TokKwObjects:             "OBJECTS",
	TokKwNotifications:       "NOTIFICATIONS",
	TokKwModule:              "MODULE",
	TokKwMandatoryGroups:     "MANDATORY_GROUPS",
	TokKwGroup:               "GROUP",
	TokKwWriteSyntax:         "WRITE_SYNTAX",
	TokKwProductRelease:      "PRODUCT_RELEASE",
	TokKwSupports:            "SUPPORTS",
	TokKwIncludes:            "INCLUDES",
	TokKwVariation:           "VARIATION",
	TokKwCreationRequires:    "CREATION_REQUIRES",
	TokKwRevision:            "REVISION",
	TokKwLastUpdated:         "LAST_UPDATED",
	TokKwOrganization:        "ORGANIZATION",
	TokKwContactInfo:         "CONTACT_INFO",
	TokKwImplied:             "IMPLIED",
	TokKwSize:                "SIZE",
	TokKwEnterprise:          "ENTERPRISE",
	TokKwVariables:           "VARIABLES",
	TokKwModuleIdentity:      "MODULE_IDENTITY",
	TokKwModuleCompliance:    "MODULE_COMPLIANCE",
	TokKwObjectGroup:         "OBJECT_GROUP",
	TokKwNotificationGroup:   "NOTIFICATION_GROUP",
	TokKwAgentCapabilities:   "AGENT_CAPABILITIES",
	TokKwObjectType:          "OBJECT_TYPE",
	TokKwObjectIdentity:      "OBJECT_IDENTITY",
	TokKwNotificationType:    "NOTIFICATION_TYPE",
	TokKwTextualConvention:   "TEXTUAL_CONVENTION",
	TokKwTrapType:            "TRAP_TYPE",
	TokKwInteger:             "INTEGER",
	TokKwUnsigned32:          "UNSIGNED32",
	TokKwCounter32:           "COUNTER32",
	TokKwCounter64:           "COUNTER64",
	TokKwGauge32:             "GAUGE32",
	TokKwIpAddress:           "IPADDRESS",
	TokKwOpaque:              "OPAQUE",
	TokKwTimeTicks:           "TIMETICKS",
	TokKwBits:                "BITS",
	TokKwOctet:               "OCTET",
	TokKwString:              "STRING",
	TokKwCounter:             "COUNTER",
	TokKwGauge:               "GAUGE",
	TokKwNetworkAddress:      "NETWORKADDRESS",
	TokKwApplication:         "APPLICATION",
	TokKwImplicit:            "IMPLICIT",
	TokKwUniversal:           "UNIVERSAL",
	TokKwCurrent:             "CURRENT",
	TokKwDeprecated:          "DEPRECATED",
	TokKwObsolete:            "OBSOLETE",
	TokKwMandatory:           "MANDATORY",
	TokKwOptional:            "OPTIONAL",
	TokKwReadOnly:            "READ_ONLY",
	TokKwReadWrite:           "READ_WRITE",
	TokKwReadCreate:          "READ_CREATE",
	TokKwWriteOnly:           "WRITE_ONLY",
	TokKwNotAccessible:       "NOT_ACCESSIBLE",
	TokKwAccessibleForNotify: "ACCESSIBLE_FOR_NOTIFY",
	TokKwNotImplemented:      "NOT_IMPLEMENTED",
}

// LibsmiName returns the libsmi-compatible name for this token kind.
func (k TokenKind) LibsmiName() string {
	i := int(k)
	if i >= 0 && i < len(libsmiNames) && libsmiNames[i] != "" {
		return libsmiNames[i]
	}
	return "UNKNOWN"
}
