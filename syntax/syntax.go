// Package syntax provides public access to the SMI MIB concrete syntax tree
// (CST), token types, and token kind constants. It re-exports types from
// internal packages so callers never need to import internal paths.
//
// Use [Parse] to obtain a lossless CST from source bytes. Use [Tokenize] for
// token-only access. Use [ReconstructText] to verify round-trip fidelity.
// Use [BuildLineTable] and [LineTable.LineCol] to map byte offsets to line and
// column numbers.
package syntax

import (
	"github.com/golangsnmp/gomib/internal/cst"
	cstparser "github.com/golangsnmp/gomib/internal/cst/parser"
	"github.com/golangsnmp/gomib/internal/lexer"
	"github.com/golangsnmp/gomib/internal/types"
)

// --- Lexer types ---

type (
	// Token is a lexed unit with its classification and source location.
	Token = lexer.Token
	// TokenKind classifies a token (punctuation, keyword, literal, etc.).
	TokenKind = lexer.TokenKind
	// Span represents a byte range in source text.
	Span = types.Span
	// ByteOffset is a byte position in source text.
	ByteOffset = types.ByteOffset
	// SpanDiagnostic is a diagnostic from the lexer or parser with a source span.
	SpanDiagnostic = types.SpanDiagnostic
	// Severity is the severity level of a diagnostic.
	Severity = types.Severity
)

// --- CST token types ---

type (
	// SyntaxToken is a meaningful token with attached leading and trailing trivia.
	SyntaxToken = cst.SyntaxToken
	// Trivia is a whitespace or comment fragment attached to a token.
	Trivia = cst.Trivia
	// TriviaKind classifies a trivia element (whitespace or comment).
	TriviaKind = cst.TriviaKind
)

const (
	// TriviaWhitespace is trivia carrying whitespace characters.
	TriviaWhitespace = cst.TriviaWhitespace
	// TriviaComment is trivia carrying a comment.
	TriviaComment = cst.TriviaComment
)

// --- CST module nodes ---

type (
	// ModuleFile is the root CST node for a parsed file.
	ModuleFile = cst.ModuleFile
	// ModuleNode is a single DEFINITIONS ::= BEGIN...END block.
	ModuleNode = cst.ModuleNode
	// ImportsNode is the IMPORTS...; section of a module.
	ImportsNode = cst.ImportsNode
	// ImportGroupNode is one "symbol, symbol FROM ModuleName" group.
	ImportGroupNode = cst.ImportGroupNode
)

// --- CST definition nodes ---

type (
	// ObjectTypeNode is an OBJECT-TYPE definition.
	ObjectTypeNode = cst.ObjectTypeNode
	// ModuleIdentityNode is a MODULE-IDENTITY definition.
	ModuleIdentityNode = cst.ModuleIdentityNode
	// ObjectIdentityNode is an OBJECT-IDENTITY definition.
	ObjectIdentityNode = cst.ObjectIdentityNode
	// NotificationTypeNode is a NOTIFICATION-TYPE definition.
	NotificationTypeNode = cst.NotificationTypeNode
	// TrapTypeNode is a TRAP-TYPE definition.
	TrapTypeNode = cst.TrapTypeNode
	// TextualConventionNode is a TEXTUAL-CONVENTION definition.
	TextualConventionNode = cst.TextualConventionNode
	// TypeAssignmentNode is an ASN.1 type assignment.
	TypeAssignmentNode = cst.TypeAssignmentNode
	// ValueAssignmentNode is an OBJECT IDENTIFIER value assignment.
	ValueAssignmentNode = cst.ValueAssignmentNode
	// ObjectGroupNode is an OBJECT-GROUP definition.
	ObjectGroupNode = cst.ObjectGroupNode
	// NotificationGroupNode is a NOTIFICATION-GROUP definition.
	NotificationGroupNode = cst.NotificationGroupNode
	// ModuleComplianceNode is a MODULE-COMPLIANCE definition.
	ModuleComplianceNode = cst.ModuleComplianceNode
	// AgentCapabilitiesNode is an AGENT-CAPABILITIES definition.
	AgentCapabilitiesNode = cst.AgentCapabilitiesNode
	// MacroDefinitionNode is an ASN.1 MACRO definition.
	MacroDefinitionNode = cst.MacroDefinitionNode
	// ErrorNode holds tokens from a parse error recovery region.
	ErrorNode = cst.ErrorNode
	// DefinitionNode is implemented by all definition CST node types.
	DefinitionNode = cst.DefinitionNode
)

// --- CST clause nodes ---

type (
	// SyntaxClauseNode is a SYNTAX clause.
	SyntaxClauseNode = cst.SyntaxClauseNode
	// AccessClauseNode is a MAX-ACCESS or ACCESS clause.
	AccessClauseNode = cst.AccessClauseNode
	// StatusClauseNode is a STATUS clause.
	StatusClauseNode = cst.StatusClauseNode
	// DescriptionClauseNode is a DESCRIPTION clause.
	DescriptionClauseNode = cst.DescriptionClauseNode
	// ReferenceClauseNode is a REFERENCE clause.
	ReferenceClauseNode = cst.ReferenceClauseNode
	// UnitsClauseNode is a UNITS clause.
	UnitsClauseNode = cst.UnitsClauseNode
	// DisplayHintClauseNode is a DISPLAY-HINT clause.
	DisplayHintClauseNode = cst.DisplayHintClauseNode
	// IndexClauseNode is an INDEX clause.
	IndexClauseNode = cst.IndexClauseNode
	// IndexItemNode is one entry in an INDEX clause.
	IndexItemNode = cst.IndexItemNode
	// AugmentsClauseNode is an AUGMENTS clause.
	AugmentsClauseNode = cst.AugmentsClauseNode
	// DefValClauseNode is a DEFVAL clause.
	DefValClauseNode = cst.DefValClauseNode
	// ObjectsClauseNode is an OBJECTS clause.
	ObjectsClauseNode = cst.ObjectsClauseNode
	// NotificationsClauseNode is a NOTIFICATIONS clause.
	NotificationsClauseNode = cst.NotificationsClauseNode
	// RevisionNode is a REVISION clause.
	RevisionNode = cst.RevisionNode
	// LastUpdatedClauseNode is a LAST-UPDATED clause.
	LastUpdatedClauseNode = cst.LastUpdatedClauseNode
	// OrganizationClauseNode is an ORGANIZATION clause.
	OrganizationClauseNode = cst.OrganizationClauseNode
	// ContactInfoClauseNode is a CONTACT-INFO clause.
	ContactInfoClauseNode = cst.ContactInfoClauseNode
	// EnterpriseClauseNode is an ENTERPRISE clause.
	EnterpriseClauseNode = cst.EnterpriseClauseNode
	// VariablesClauseNode is a VARIABLES clause.
	VariablesClauseNode = cst.VariablesClauseNode
	// ProductReleaseClauseNode is a PRODUCT-RELEASE clause.
	ProductReleaseClauseNode = cst.ProductReleaseClauseNode
	// WriteSyntaxClauseNode is a WRITE-SYNTAX clause.
	WriteSyntaxClauseNode = cst.WriteSyntaxClauseNode
)

// --- CST type syntax nodes ---

type (
	// TypeRefSyntaxNode is a type reference (e.g. "Integer32").
	TypeRefSyntaxNode = cst.TypeRefSyntaxNode
	// IntegerEnumSyntaxNode is an INTEGER with named enumeration values.
	IntegerEnumSyntaxNode = cst.IntegerEnumSyntaxNode
	// EnumValueNode is one named value in an INTEGER enumeration.
	EnumValueNode = cst.EnumValueNode
	// BitsSyntaxNode is a BITS type with named bit positions.
	BitsSyntaxNode = cst.BitsSyntaxNode
	// ConstrainedSyntaxNode is a type with a SIZE or value constraint.
	ConstrainedSyntaxNode = cst.ConstrainedSyntaxNode
	// ConstraintNode is the constraint expression (parenthesised ranges).
	ConstraintNode = cst.ConstraintNode
	// RangeNode is one value or range within a constraint.
	RangeNode = cst.RangeNode
	// SequenceOfSyntaxNode is a SEQUENCE OF type.
	SequenceOfSyntaxNode = cst.SequenceOfSyntaxNode
	// SequenceSyntaxNode is an inline SEQUENCE type definition.
	SequenceSyntaxNode = cst.SequenceSyntaxNode
	// SequenceFieldNode is one field in a SEQUENCE definition.
	SequenceFieldNode = cst.SequenceFieldNode
	// OctetStringSyntaxNode is an OCTET STRING type reference.
	OctetStringSyntaxNode = cst.OctetStringSyntaxNode
	// ObjectIdentifierSyntaxNode is an OBJECT IDENTIFIER type reference.
	ObjectIdentifierSyntaxNode = cst.ObjectIdentifierSyntaxNode
	// TaggedSyntaxNode is an ASN.1 tagged type (e.g. [APPLICATION 0] IMPLICIT ...).
	TaggedSyntaxNode = cst.TaggedSyntaxNode
	// ChoiceSyntaxNode is an ASN.1 CHOICE type.
	ChoiceSyntaxNode = cst.ChoiceSyntaxNode
	// TypeSyntaxNode is implemented by all type syntax CST node types.
	TypeSyntaxNode = cst.TypeSyntaxNode
)

// --- CST OID nodes ---

type (
	// OidValueNode is an OID literal value in braces.
	OidValueNode = cst.OidValueNode
	// OidComponentNode is one component in an OID value.
	OidComponentNode = cst.OidComponentNode
)

// --- CST conformance nodes ---

type (
	// ComplianceModuleNode is a MODULE section within MODULE-COMPLIANCE.
	ComplianceModuleNode = cst.ComplianceModuleNode
	// MandatoryGroupsNode is a MANDATORY-GROUPS clause.
	MandatoryGroupsNode = cst.MandatoryGroupsNode
	// ComplianceGroupNode is a GROUP clause within a compliance module.
	ComplianceGroupNode = cst.ComplianceGroupNode
	// ComplianceObjectNode is an OBJECT clause within a compliance module.
	ComplianceObjectNode = cst.ComplianceObjectNode
	// CapabilityModuleNode is a SUPPORTS section within AGENT-CAPABILITIES.
	CapabilityModuleNode = cst.CapabilityModuleNode
	// IncludesNode is an INCLUDES clause within a capability module.
	IncludesNode = cst.IncludesNode
	// CapabilityVariationNode is a VARIATION clause within a capability module.
	CapabilityVariationNode = cst.CapabilityVariationNode
	// CreationRequiresNode is a CREATION-REQUIRES clause.
	CreationRequiresNode = cst.CreationRequiresNode
)

// --- Token kind constants ---

// Special token kinds.
const (
	TokError            = lexer.TokError
	TokEOF              = lexer.TokEOF
	TokForbiddenKeyword = lexer.TokForbiddenKeyword
	TokComment          = lexer.TokComment
)

// Identifier token kinds.
const (
	TokUppercaseIdent = lexer.TokUppercaseIdent
	TokLowercaseIdent = lexer.TokLowercaseIdent
)

// Literal token kinds.
const (
	TokNumber         = lexer.TokNumber
	TokNegativeNumber = lexer.TokNegativeNumber
	TokQuotedString   = lexer.TokQuotedString
	TokHexString      = lexer.TokHexString
	TokBinString      = lexer.TokBinString
)

// Punctuation token kinds.
const (
	TokLBracket  = lexer.TokLBracket
	TokRBracket  = lexer.TokRBracket
	TokLBrace    = lexer.TokLBrace
	TokRBrace    = lexer.TokRBrace
	TokLParen    = lexer.TokLParen
	TokRParen    = lexer.TokRParen
	TokColon     = lexer.TokColon
	TokSemicolon = lexer.TokSemicolon
	TokComma     = lexer.TokComma
	TokDot       = lexer.TokDot
	TokPipe      = lexer.TokPipe
	TokMinus     = lexer.TokMinus
)

// Operator token kinds.
const (
	TokDotDot          = lexer.TokDotDot
	TokColonColonEqual = lexer.TokColonColonEqual
)

// Structural keyword token kinds.
const (
	TokKwDefinitions = lexer.TokKwDefinitions
	TokKwBegin       = lexer.TokKwBegin
	TokKwEnd         = lexer.TokKwEnd
	TokKwImports     = lexer.TokKwImports
	TokKwExports     = lexer.TokKwExports
	TokKwFrom        = lexer.TokKwFrom
	TokKwObject      = lexer.TokKwObject
	TokKwIdentifier  = lexer.TokKwIdentifier
	TokKwSequence    = lexer.TokKwSequence
	TokKwOf          = lexer.TokKwOf
	TokKwChoice      = lexer.TokKwChoice
	TokKwMacro       = lexer.TokKwMacro
)

// Clause keyword token kinds.
const (
	TokKwSyntax           = lexer.TokKwSyntax
	TokKwMaxAccess        = lexer.TokKwMaxAccess
	TokKwMinAccess        = lexer.TokKwMinAccess
	TokKwAccess           = lexer.TokKwAccess
	TokKwStatus           = lexer.TokKwStatus
	TokKwDescription      = lexer.TokKwDescription
	TokKwReference        = lexer.TokKwReference
	TokKwIndex            = lexer.TokKwIndex
	TokKwDefval           = lexer.TokKwDefval
	TokKwAugments         = lexer.TokKwAugments
	TokKwUnits            = lexer.TokKwUnits
	TokKwDisplayHint      = lexer.TokKwDisplayHint
	TokKwObjects          = lexer.TokKwObjects
	TokKwNotifications    = lexer.TokKwNotifications
	TokKwModule           = lexer.TokKwModule
	TokKwMandatoryGroups  = lexer.TokKwMandatoryGroups
	TokKwGroup            = lexer.TokKwGroup
	TokKwWriteSyntax      = lexer.TokKwWriteSyntax
	TokKwProductRelease   = lexer.TokKwProductRelease
	TokKwSupports         = lexer.TokKwSupports
	TokKwIncludes         = lexer.TokKwIncludes
	TokKwVariation        = lexer.TokKwVariation
	TokKwCreationRequires = lexer.TokKwCreationRequires
	TokKwRevision         = lexer.TokKwRevision
	TokKwLastUpdated      = lexer.TokKwLastUpdated
	TokKwOrganization     = lexer.TokKwOrganization
	TokKwContactInfo      = lexer.TokKwContactInfo
	TokKwImplied          = lexer.TokKwImplied
	TokKwSize             = lexer.TokKwSize
	TokKwEnterprise       = lexer.TokKwEnterprise
	TokKwVariables        = lexer.TokKwVariables
)

// Macro keyword token kinds.
const (
	TokKwModuleIdentity    = lexer.TokKwModuleIdentity
	TokKwModuleCompliance  = lexer.TokKwModuleCompliance
	TokKwObjectGroup       = lexer.TokKwObjectGroup
	TokKwNotificationGroup = lexer.TokKwNotificationGroup
	TokKwAgentCapabilities = lexer.TokKwAgentCapabilities
	TokKwObjectType        = lexer.TokKwObjectType
	TokKwObjectIdentity    = lexer.TokKwObjectIdentity
	TokKwNotificationType  = lexer.TokKwNotificationType
	TokKwTextualConvention = lexer.TokKwTextualConvention
	TokKwTrapType          = lexer.TokKwTrapType
)

// Type keyword token kinds.
const (
	TokKwInteger        = lexer.TokKwInteger
	TokKwUnsigned32     = lexer.TokKwUnsigned32
	TokKwCounter32      = lexer.TokKwCounter32
	TokKwCounter64      = lexer.TokKwCounter64
	TokKwGauge32        = lexer.TokKwGauge32
	TokKwIpAddress      = lexer.TokKwIpAddress
	TokKwOpaque         = lexer.TokKwOpaque
	TokKwTimeTicks      = lexer.TokKwTimeTicks
	TokKwBits           = lexer.TokKwBits
	TokKwOctet          = lexer.TokKwOctet
	TokKwString         = lexer.TokKwString
	TokKwCounter        = lexer.TokKwCounter
	TokKwGauge          = lexer.TokKwGauge
	TokKwNetworkAddress = lexer.TokKwNetworkAddress
)

// Tag keyword token kinds.
const (
	TokKwApplication = lexer.TokKwApplication
	TokKwImplicit    = lexer.TokKwImplicit
	TokKwUniversal   = lexer.TokKwUniversal
)

// Status and access keyword token kinds.
const (
	TokKwCurrent             = lexer.TokKwCurrent
	TokKwDeprecated          = lexer.TokKwDeprecated
	TokKwObsolete            = lexer.TokKwObsolete
	TokKwMandatory           = lexer.TokKwMandatory
	TokKwOptional            = lexer.TokKwOptional
	TokKwReadOnly            = lexer.TokKwReadOnly
	TokKwReadWrite           = lexer.TokKwReadWrite
	TokKwReadCreate          = lexer.TokKwReadCreate
	TokKwWriteOnly           = lexer.TokKwWriteOnly
	TokKwNotAccessible       = lexer.TokKwNotAccessible
	TokKwAccessibleForNotify = lexer.TokKwAccessibleForNotify
	TokKwNotImplemented      = lexer.TokKwNotImplemented
)

// --- Functions ---

// Tokenize lexes source bytes into a token stream. The returned slice
// includes a final EOF token. Diagnostics are discarded; use [Parse]
// if diagnostics are needed.
func Tokenize(source []byte) []Token {
	l := lexer.New(source, nil, types.DiagnosticConfig{})
	tokens, _ := l.Tokenize()
	return tokens
}

// Parse parses source bytes into a lossless concrete syntax tree.
// Returns the CST root and any lexer/parser diagnostics as span-based
// diagnostics (byte offsets). Use [LineTable.LineCol] to convert offsets
// to line/column numbers.
func Parse(source []byte) (file *ModuleFile, diags []SpanDiagnostic) {
	p := cstparser.New(source, nil, types.DiagnosticConfig{})
	return p.ParseModule(), p.Diagnostics()
}

// ReconstructText rebuilds the source text by walking all CST tokens
// in order and concatenating their trivia and token text. A correct
// CST round-trips to the original source.
var ReconstructText = cst.ReconstructText

// IdentifierAt returns the SMI identifier at the given byte offset in source,
// or "" if the offset is not on an identifier character. SMI identifiers
// contain letters, digits, hyphens, and underscores.
func IdentifierAt(source []byte, offset ByteOffset) string {
	off := int(offset)
	if off >= len(source) {
		return ""
	}
	if !IsIdentByte(source[off]) {
		return ""
	}
	start := off
	for start > 0 && IsIdentByte(source[start-1]) {
		start--
	}
	end := off
	for end < len(source) && IsIdentByte(source[end]) {
		end++
	}
	return string(source[start:end])
}

// IsIdentByte reports whether b is valid in an SMI identifier.
// SMI identifiers contain ASCII letters, digits, hyphens, and underscores.
func IsIdentByte(b byte) bool {
	return (b >= 'A' && b <= 'Z') || (b >= 'a' && b <= 'z') || (b >= '0' && b <= '9') || b == '-' || b == '_'
}
