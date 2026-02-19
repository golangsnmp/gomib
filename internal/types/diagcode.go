package types

// Diagnostic codes emitted by the parser, lowering, and resolver phases.
// Centralizing these prevents silent breakage from typos in string literals.

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
)

// Resolver diagnostic codes.
const (
	DiagImportNotFound       = "import-not-found"
	DiagImportModuleNotFound = "import-module-not-found"
	DiagTypeUnknown          = "type-unknown"
	DiagOidOrphan            = "oid-orphan"
	DiagIndexUnresolved      = "index-unresolved"
	DiagObjectsUnresolved    = "objects-unresolved"
	DiagIdentifierHyphenSMI  = "identifier-hyphen-smiv2"
	DiagGroupNotAccessible   = "group-not-accessible"
	DiagNotifObjectNotObject = "notification-object-not-object"
	DiagMalformedHexDefval   = "malformed-hex-defval"
	DiagDefvalUnresolved     = "defval-unresolved"
)

// AllDiagnosticCodes returns all known diagnostic codes grouped by phase.
func AllDiagnosticCodes() []DiagCodeInfo {
	return []DiagCodeInfo{
		// Parser
		{Code: DiagIdentifierUnderscore, Phase: "parser"},
		{Code: DiagIdentifierHyphenEnd, Phase: "parser"},
		{Code: DiagIdentifierLength64, Phase: "parser"},
		{Code: DiagIdentifierLength32, Phase: "parser"},
		{Code: DiagBadIdentifierCase, Phase: "parser"},
		{Code: DiagParseError, Phase: "parser"},
		{Code: DiagInvalidU32, Phase: "parser"},
		{Code: DiagInvalidI64, Phase: "parser"},
		{Code: DiagKeywordReserved, Phase: "parser"},
		{Code: DiagInvalidHexRange, Phase: "parser"},
		// Lowering
		{Code: DiagMissingModuleIdentity, Phase: "lowering"},
		{Code: DiagRevisionLastUpdated, Phase: "lowering"},
		{Code: DiagUnknownDefinitionType, Phase: "lowering"},
		{Code: DiagUnknownTypeSyntax, Phase: "lowering"},
		{Code: DiagUnknownConstraintType, Phase: "lowering"},
		{Code: DiagUnknownRangeValue, Phase: "lowering"},
		{Code: DiagUnknownOidComponent, Phase: "lowering"},
		{Code: DiagUnknownDefvalType, Phase: "lowering"},
		// Resolver
		{Code: DiagImportNotFound, Phase: "resolver"},
		{Code: DiagImportModuleNotFound, Phase: "resolver"},
		{Code: DiagTypeUnknown, Phase: "resolver"},
		{Code: DiagOidOrphan, Phase: "resolver"},
		{Code: DiagIndexUnresolved, Phase: "resolver"},
		{Code: DiagObjectsUnresolved, Phase: "resolver"},
		{Code: DiagIdentifierHyphenSMI, Phase: "resolver"},
		{Code: DiagGroupNotAccessible, Phase: "resolver"},
		{Code: DiagNotifObjectNotObject, Phase: "resolver"},
		{Code: DiagMalformedHexDefval, Phase: "resolver"},
		{Code: DiagDefvalUnresolved, Phase: "resolver"},
	}
}

// DiagCodeInfo describes a diagnostic code and the phase that emits it.
type DiagCodeInfo struct {
	Code  string
	Phase string
}
