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
