package token

import (
	"github.com/golangsnmp/gomib/internal/lexer"
	"github.com/golangsnmp/gomib/internal/types"
)

// Token represents a lexed unit with its classification and source location.
type Token = lexer.Token

// TokenKind classifies a token (punctuation, keyword, literal, etc.).
type TokenKind = lexer.TokenKind

// Span represents a byte range in source text.
type Span = types.Span

// ByteOffset is a byte position in source text.
type ByteOffset = types.ByteOffset

// Token kinds - special.
const (
	Error            = lexer.TokError
	EOF              = lexer.TokEOF
	ForbiddenKeyword = lexer.TokForbiddenKeyword
	Comment          = lexer.TokComment
)

// Token kinds - identifiers.
const (
	UppercaseIdent = lexer.TokUppercaseIdent
	LowercaseIdent = lexer.TokLowercaseIdent
)

// Token kinds - literals.
const (
	Number         = lexer.TokNumber
	NegativeNumber = lexer.TokNegativeNumber
	QuotedString   = lexer.TokQuotedString
	HexString      = lexer.TokHexString
	BinString      = lexer.TokBinString
)

// Token kinds - punctuation.
const (
	LBracket  = lexer.TokLBracket
	RBracket  = lexer.TokRBracket
	LBrace    = lexer.TokLBrace
	RBrace    = lexer.TokRBrace
	LParen    = lexer.TokLParen
	RParen    = lexer.TokRParen
	Colon     = lexer.TokColon
	Semicolon = lexer.TokSemicolon
	Comma     = lexer.TokComma
	Dot       = lexer.TokDot
	Pipe      = lexer.TokPipe
	Minus     = lexer.TokMinus
)

// Token kinds - multi-character operators.
const (
	DotDot          = lexer.TokDotDot
	ColonColonEqual = lexer.TokColonColonEqual
)

// Token kinds - structural keywords.
const (
	KwDefinitions = lexer.TokKwDefinitions
	KwBegin       = lexer.TokKwBegin
	KwEnd         = lexer.TokKwEnd
	KwImports     = lexer.TokKwImports
	KwExports     = lexer.TokKwExports
	KwFrom        = lexer.TokKwFrom
	KwObject      = lexer.TokKwObject
	KwIdentifier  = lexer.TokKwIdentifier
	KwSequence    = lexer.TokKwSequence
	KwOf          = lexer.TokKwOf
	KwChoice      = lexer.TokKwChoice
	KwMacro       = lexer.TokKwMacro
)

// Token kinds - clause keywords.
const (
	KwSyntax           = lexer.TokKwSyntax
	KwMaxAccess        = lexer.TokKwMaxAccess
	KwMinAccess        = lexer.TokKwMinAccess
	KwAccess           = lexer.TokKwAccess
	KwStatus           = lexer.TokKwStatus
	KwDescription      = lexer.TokKwDescription
	KwReference        = lexer.TokKwReference
	KwIndex            = lexer.TokKwIndex
	KwDefval           = lexer.TokKwDefval
	KwAugments         = lexer.TokKwAugments
	KwUnits            = lexer.TokKwUnits
	KwDisplayHint      = lexer.TokKwDisplayHint
	KwObjects          = lexer.TokKwObjects
	KwNotifications    = lexer.TokKwNotifications
	KwModule           = lexer.TokKwModule
	KwMandatoryGroups  = lexer.TokKwMandatoryGroups
	KwGroup            = lexer.TokKwGroup
	KwWriteSyntax      = lexer.TokKwWriteSyntax
	KwProductRelease   = lexer.TokKwProductRelease
	KwSupports         = lexer.TokKwSupports
	KwIncludes         = lexer.TokKwIncludes
	KwVariation        = lexer.TokKwVariation
	KwCreationRequires = lexer.TokKwCreationRequires
	KwRevision         = lexer.TokKwRevision
	KwLastUpdated      = lexer.TokKwLastUpdated
	KwOrganization     = lexer.TokKwOrganization
	KwContactInfo      = lexer.TokKwContactInfo
	KwImplied          = lexer.TokKwImplied
	KwSize             = lexer.TokKwSize
	KwEnterprise       = lexer.TokKwEnterprise
	KwVariables        = lexer.TokKwVariables
)

// Token kinds - macro invocation keywords.
const (
	KwModuleIdentity    = lexer.TokKwModuleIdentity
	KwModuleCompliance  = lexer.TokKwModuleCompliance
	KwObjectGroup       = lexer.TokKwObjectGroup
	KwNotificationGroup = lexer.TokKwNotificationGroup
	KwAgentCapabilities = lexer.TokKwAgentCapabilities
	KwObjectType        = lexer.TokKwObjectType
	KwObjectIdentity    = lexer.TokKwObjectIdentity
	KwNotificationType  = lexer.TokKwNotificationType
	KwTextualConvention = lexer.TokKwTextualConvention
	KwTrapType          = lexer.TokKwTrapType
)

// Token kinds - type keywords.
const (
	KwInteger        = lexer.TokKwInteger
	KwUnsigned32     = lexer.TokKwUnsigned32
	KwCounter32      = lexer.TokKwCounter32
	KwCounter64      = lexer.TokKwCounter64
	KwGauge32        = lexer.TokKwGauge32
	KwIpAddress      = lexer.TokKwIpAddress
	KwOpaque         = lexer.TokKwOpaque
	KwTimeTicks      = lexer.TokKwTimeTicks
	KwBits           = lexer.TokKwBits
	KwOctet          = lexer.TokKwOctet
	KwString         = lexer.TokKwString
	KwCounter        = lexer.TokKwCounter
	KwGauge          = lexer.TokKwGauge
	KwNetworkAddress = lexer.TokKwNetworkAddress
)

// Token kinds - ASN.1 tag keywords.
const (
	KwApplication = lexer.TokKwApplication
	KwImplicit    = lexer.TokKwImplicit
	KwUniversal   = lexer.TokKwUniversal
)

// Token kinds - status/access value keywords.
const (
	KwCurrent             = lexer.TokKwCurrent
	KwDeprecated          = lexer.TokKwDeprecated
	KwObsolete            = lexer.TokKwObsolete
	KwMandatory           = lexer.TokKwMandatory
	KwOptional            = lexer.TokKwOptional
	KwReadOnly            = lexer.TokKwReadOnly
	KwReadWrite           = lexer.TokKwReadWrite
	KwReadCreate          = lexer.TokKwReadCreate
	KwWriteOnly           = lexer.TokKwWriteOnly
	KwNotAccessible       = lexer.TokKwNotAccessible
	KwAccessibleForNotify = lexer.TokKwAccessibleForNotify
	KwNotImplemented      = lexer.TokKwNotImplemented
)

// Tokenize lexes source bytes into a token stream. The returned slice
// includes a final EOF token. Diagnostics are discarded; use the
// internal lexer directly if diagnostics are needed.
func Tokenize(source []byte) []Token {
	l := lexer.New(source, nil, types.DiagnosticConfig{})
	tokens, _ := l.Tokenize()
	return tokens
}
