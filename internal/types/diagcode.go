package types

// Diagnostic codes emitted by the parser, lowering, and resolver phases.
// Centralizing these prevents silent breakage from typos in string literals.

// Lexer diagnostic codes.
const (
	DiagUnexpectedCharacter   = "unexpected-character"
	DiagUnterminatedString    = "unterminated-string"
	DiagUnterminatedHexBinStr = "unterminated-hex-bin-string"
	DiagMissingHexBinSuffix   = "missing-hex-bin-suffix"
)

var lexerDiagCodes = []string{
	DiagUnexpectedCharacter,
	DiagUnterminatedString,
	DiagUnterminatedHexBinStr,
	DiagMissingHexBinSuffix,
}

// Parser diagnostic codes.
const (
	DiagIdentifierUnderscore = "identifier-underscore"
	DiagIdentifierHyphenEnd  = "identifier-hyphen-end"
	DiagIdentifierLength64   = "identifier-length-64"
	DiagIdentifierLength32   = "identifier-length-32"
	DiagBadIdentifierCase    = "bad-identifier-case"
	DiagParseError           = "parse-error"
	DiagInvalidU32           = "invalid-u32"
	DiagInvalidI64           = "invalid-i64"
	DiagKeywordReserved      = "keyword-reserved"
	DiagInvalidHexRange      = "invalid-hex-range"
)

var parserDiagCodes = []string{
	DiagIdentifierUnderscore,
	DiagIdentifierHyphenEnd,
	DiagIdentifierLength64,
	DiagIdentifierLength32,
	DiagBadIdentifierCase,
	DiagParseError,
	DiagInvalidU32,
	DiagInvalidI64,
	DiagKeywordReserved,
	DiagInvalidHexRange,
}

// Lowering diagnostic codes.
const (
	DiagMissingModuleIdentity = "missing-module-identity"
	DiagRevisionLastUpdated   = "revision-last-updated"
	DiagUnknownDefinitionType = "unknown-definition-type"
	DiagUnknownTypeSyntax     = "unknown-type-syntax"
	DiagUnknownConstraintType = "unknown-constraint-type"
	DiagUnknownRangeValue     = "unknown-range-value"
	DiagUnknownOidComponent   = "unknown-oid-component-type"
	DiagUnknownDefvalType     = "unknown-defval-type"
	DiagBitsNumberNegative    = "bits-number-negative"
	DiagBitsNumberTooLarge    = "bits-number-too-large"
	DiagBitsNumberLarge       = "bits-number-large"
	DiagEnumZero              = "enum-zero"
	DiagEnumNameRedefinition  = "enum-name-redefinition"
	DiagEnumValueRedefinition = "enum-value-redefinition"
	DiagBitsNameRedefinition  = "bits-name-redefinition"
	DiagBitsValueRedefinition = "bits-value-redefinition"
)

var loweringDiagCodes = []string{
	DiagMissingModuleIdentity,
	DiagRevisionLastUpdated,
	DiagUnknownDefinitionType,
	DiagUnknownTypeSyntax,
	DiagUnknownConstraintType,
	DiagUnknownRangeValue,
	DiagUnknownOidComponent,
	DiagUnknownDefvalType,
	DiagBitsNumberNegative,
	DiagBitsNumberTooLarge,
	DiagBitsNumberLarge,
	DiagEnumZero,
	DiagEnumNameRedefinition,
	DiagEnumValueRedefinition,
	DiagBitsNameRedefinition,
	DiagBitsValueRedefinition,
}

// Resolver diagnostic codes.
const (
	DiagImportNotFound           = "import-not-found"
	DiagImportModuleNotFound     = "import-module-not-found"
	DiagTypeUnknown              = "type-unknown"
	DiagOidOrphan                = "oid-orphan"
	DiagIndexUnresolved          = "index-unresolved"
	DiagObjectsUnresolved        = "objects-unresolved"
	DiagIdentifierHyphenSMI      = "identifier-hyphen-smiv2"
	DiagGroupNotAccessible       = "group-not-accessible"
	DiagNotifObjectNotObject     = "notification-object-not-object"
	DiagMalformedHexDefval       = "malformed-hex-defval"
	DiagMalformedBinDefval       = "malformed-bin-defval"
	DiagDefvalUnresolved         = "defval-unresolved"
	DiagVariationAccessNotifOnly = "variation-access-notification-only"
	DiagGroupMemberUnresolved    = "group-member-unresolved"
	DiagIndexNotObject           = "index-not-object"
	DiagAugmentsNotObject        = "augments-not-object"
	DiagNotificationNoOid        = "notification-no-oid"
	DiagPrimitiveTypeMissing     = "primitive-type-missing"
	DiagIntegerInSMIv2           = "integer-in-smiv2"
	DiagIndexIntegerNoRange      = "index-integer-no-range"
	DiagIndexNegativeRange       = "index-negative-range"
	DiagDefvalBasetype           = "defval-basetype"
	DiagDefvalRange              = "defval-range"
	DiagDefvalEnum               = "defval-enum"
	DiagRangeBounds              = "range-bounds"
	DiagRangeExchanged           = "range-exchanged"
	DiagRangeOverlap             = "range-overlap"
	DiagRangeAscending           = "range-ascending"
	DiagSizeIllegal              = "size-illegal"
	DiagRangeIllegal             = "range-illegal"
	DiagCounterRangeIllegal      = "counter-range-illegal"
	DiagSubtypeEnumIllegal       = "subtype-enumeration-illegal"
	DiagSubtypeBitsIllegal       = "subtype-bits-illegal"
	DiagParentTable              = "parent-table"
	DiagParentRow                = "parent-row"
	DiagParentColumn             = "parent-column"
	DiagParentScalar             = "parent-scalar"
	DiagParentNode               = "parent-node"
	DiagParentNotification       = "parent-notification"
	DiagParentGroup              = "parent-group"
	DiagParentCompliance         = "parent-compliance"
	DiagParentCapabilities       = "parent-capabilities"
)

var resolverDiagCodes = []string{
	DiagImportNotFound,
	DiagImportModuleNotFound,
	DiagTypeUnknown,
	DiagOidOrphan,
	DiagIndexUnresolved,
	DiagObjectsUnresolved,
	DiagIdentifierHyphenSMI,
	DiagGroupNotAccessible,
	DiagNotifObjectNotObject,
	DiagMalformedHexDefval,
	DiagMalformedBinDefval,
	DiagDefvalUnresolved,
	DiagVariationAccessNotifOnly,
	DiagGroupMemberUnresolved,
	DiagIndexNotObject,
	DiagAugmentsNotObject,
	DiagNotificationNoOid,
	DiagPrimitiveTypeMissing,
	DiagIntegerInSMIv2,
	DiagIndexIntegerNoRange,
	DiagIndexNegativeRange,
	DiagDefvalBasetype,
	DiagDefvalRange,
	DiagDefvalEnum,
	DiagRangeBounds,
	DiagRangeExchanged,
	DiagRangeOverlap,
	DiagRangeAscending,
	DiagSizeIllegal,
	DiagRangeIllegal,
	DiagCounterRangeIllegal,
	DiagSubtypeEnumIllegal,
	DiagSubtypeBitsIllegal,
	DiagParentTable,
	DiagParentRow,
	DiagParentColumn,
	DiagParentScalar,
	DiagParentNode,
	DiagParentNotification,
	DiagParentGroup,
	DiagParentCompliance,
	DiagParentCapabilities,
}

// diagPhases maps phase names to their diagnostic code slices.
var diagPhases = []struct {
	name  string
	codes []string
}{
	{"lexer", lexerDiagCodes},
	{"parser", parserDiagCodes},
	{"lowering", loweringDiagCodes},
	{"resolver", resolverDiagCodes},
}

// AllDiagnosticCodes returns all known diagnostic codes grouped by phase.
func AllDiagnosticCodes() []DiagCodeInfo {
	var result []DiagCodeInfo
	for _, phase := range diagPhases {
		for _, code := range phase.codes {
			result = append(result, DiagCodeInfo{Code: code, Phase: phase.name})
		}
	}
	return result
}

// DiagCodeInfo describes a diagnostic code and the phase that emits it.
type DiagCodeInfo struct {
	Code  string
	Phase string
}
