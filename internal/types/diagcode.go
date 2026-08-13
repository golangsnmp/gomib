package types

// Diagnostic codes emitted by the lexer, parser, validate, and resolver phases.
// Each code has a fixed severity defined here. Centralizing codes and severities
// prevents silent breakage from typos and ensures 1:1 code-to-severity mapping.

// Lexer diagnostic codes.
const (
	DiagUnexpectedCharacter   = "unexpected-character"
	DiagUnterminatedString    = "unterminated-string"
	DiagUnterminatedHexBinStr = "unterminated-hex-bin-string"
	DiagMissingHexBinSuffix   = "missing-hex-bin-suffix"
	DiagHexStringInvalidChar  = "hex-string-invalid-char"
	DiagBinStringInvalidChar  = "bin-string-invalid-char"
	DiagHexStringMul2         = "hex-string-mul2"
	DiagBinStringMul8         = "bin-string-mul8"
)

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
	DiagNumberLeadingZero    = "number-leading-zero"
	DiagMissingComma         = "missing-comma"
)

// Validation diagnostic codes.
const (
	DiagMissingModuleIdentity  = "missing-module-identity"
	DiagRevisionLastUpdated    = "revision-last-updated"
	DiagRevisionNotDescending  = "revision-not-descending"
	DiagRevisionAfterUpdate    = "revision-after-update"
	DiagDateCharacter          = "date-character"
	DiagDateLength             = "date-length"
	DiagDateMonth              = "date-month"
	DiagDateDay                = "date-day"
	DiagDateHour               = "date-hour"
	DiagDateMinutes            = "date-minutes"
	DiagDateValue              = "date-value"
	DiagDateYear2Digits        = "date-year-2digits"
	DiagDateInFuture           = "date-in-future"
	DiagDateInPast             = "date-in-past"
	DiagUnknownDefinitionType  = "unknown-definition-type"
	DiagUnknownTypeSyntax      = "unknown-type-syntax"
	DiagUnknownConstraintType  = "unknown-constraint-type"
	DiagUnknownRangeValue      = "unknown-range-value"
	DiagUnknownOidComponent    = "unknown-oid-component-type"
	DiagUnknownDefvalType      = "unknown-defval-type"
	DiagBitsNumberNegative     = "bits-number-negative"
	DiagBitsNumberTooLarge     = "bits-number-too-large"
	DiagBitsNumberLarge        = "bits-number-large"
	DiagEnumZero               = "enum-zero"
	DiagEnumNameRedefinition   = "enum-name-redefinition"
	DiagEnumValueRedefinition  = "enum-value-redefinition"
	DiagBitsNameRedefinition   = "bits-name-redefinition"
	DiagBitsValueRedefinition  = "bits-value-redefinition"
	DiagModuleIdentityNotFirst = "module-identity-not-first"
	DiagModuleIdentityMultiple = "module-identity-multiple"
	DiagMacroNotImported       = "macro-not-imported"
	DiagEmptyDescription       = "empty-description"
	DiagEmptyReference         = "empty-reference"
	DiagEmptyOrganization      = "empty-organization"
	DiagEmptyContact           = "empty-contact"
	DiagEmptyUnits             = "empty-units"
	DiagEmptyFormat            = "empty-format"
	DiagModuleNameSuffix       = "module-name-suffix"
	DiagMacroNotAllowed        = "macro-not-allowed"
	DiagChoiceNotAllowed       = "choice-not-allowed"
	DiagTaggedTypeNotAllowed   = "tagged-type-not-allowed"
)

// Resolver diagnostic codes.
const (
	DiagImportNotFound               = "import-not-found"
	DiagImportModuleNotFound         = "import-module-not-found"
	DiagTypeUnknown                  = "type-unknown"
	DiagTypeCycle                    = "type-cycle"
	DiagOidOrphan                    = "oid-orphan"
	DiagTrapNumberOverflow           = "trap-number-overflow"
	DiagIndexUnresolved              = "index-unresolved"
	DiagObjectsUnresolved            = "objects-unresolved"
	DiagIdentifierHyphenSMI          = "identifier-hyphen-smiv2"
	DiagGroupNotAccessible           = "group-not-accessible"
	DiagNotifObjectNotObject         = "notification-object-not-object"
	DiagNotifObjectAccess            = "notification-object-access"
	DiagNotifNotReversible           = "notification-not-reversible"
	DiagNotifIdTooLarge              = "notification-id-too-large"
	DiagMalformedHexDefval           = "malformed-hex-defval"
	DiagMalformedBinDefval           = "malformed-bin-defval"
	DiagDefvalUnresolved             = "defval-unresolved"
	DiagVariationAccessNotifOnly     = "variation-access-notification-only"
	DiagGroupMemberUnresolved        = "group-member-unresolved"
	DiagIndexNotObject               = "index-not-object"
	DiagAugmentsNotObject            = "augments-not-object"
	DiagAugmentNested                = "augment-nested"
	DiagNotifNoOid                   = "notification-no-oid"
	DiagPrimitiveTypeMissing         = "primitive-type-missing"
	DiagIntegerInSMIv2               = "integer-in-smiv2"
	DiagIndexIntegerNoRange          = "index-integer-no-range"
	DiagIndexNegativeRange           = "index-negative-range"
	DiagDefvalBasetype               = "defval-basetype"
	DiagDefvalRange                  = "defval-range"
	DiagDefvalEnum                   = "defval-enum"
	DiagDefvalBits                   = "defval-bits"
	DiagCounterDefvalIllegal         = "counter-defval-illegal"
	DiagIndexCounterIllegal          = "index-counter-illegal"
	DiagRangeBounds                  = "range-bounds"
	DiagRangeExchanged               = "range-exchanged"
	DiagRangeOverlap                 = "range-overlap"
	DiagRangeAscending               = "range-ascending"
	DiagSizeIllegal                  = "size-illegal"
	DiagRangeIllegal                 = "range-illegal"
	DiagCounterRangeIllegal          = "counter-range-illegal"
	DiagSubtypeEnumIllegal           = "subtype-enumeration-illegal"
	DiagSubtypeBitsIllegal           = "subtype-bits-illegal"
	DiagParentTable                  = "parent-table"
	DiagParentRow                    = "parent-row"
	DiagParentColumn                 = "parent-column"
	DiagParentScalar                 = "parent-scalar"
	DiagParentNode                   = "parent-node"
	DiagParentNotification           = "parent-notification"
	DiagParentGroup                  = "parent-group"
	DiagParentCompliance             = "parent-compliance"
	DiagParentCapabilities           = "parent-capabilities"
	DiagRowSubidentifierOne          = "row-node-subidentifier-one"
	DiagIndexElementNoSize           = "index-element-no-size"
	DiagIndexIllegalBasetype         = "index-illegal-basetype"
	DiagLastSubidZero                = "last-subid-zero"
	DiagOidRecursive                 = "oid-recursive"
	DiagOidRegistered                = "oid-registered"
	DiagOidReuse                     = "oid-reuse"
	DiagSequenceNoColumn             = "sequence-no-column"
	DiagSequenceMissingColumn        = "sequence-missing-column"
	DiagSequenceOrder                = "sequence-order"
	DiagSequenceTypeMismatch         = "sequence-type-mismatch"
	DiagIndexExceedsTooLarge         = "index-exceeds-too-large"
	DiagAccessInvalidSMIv1           = "access-invalid-smiv1"
	DiagAccessWriteOnlySMIv2         = "access-write-only-smiv2"
	DiagAccessTableIllegal           = "access-table-illegal"
	DiagAccessRowIllegal             = "access-row-illegal"
	DiagAccessCounterIllegal         = "access-counter-illegal"
	DiagScalarNotCreatable           = "scalar-not-creatable"
	DiagMaxAccessInSMIv1             = "maxaccess-in-smiv1"
	DiagAccessInSMIv2                = "access-in-smiv2"
	DiagStatusInvalidSMIv1           = "status-invalid-smiv1"
	DiagStatusInvalidSMIv2           = "status-invalid-smiv2"
	DiagTypeStatusDeprecated         = "type-status-deprecated"
	DiagTypeStatusObsolete           = "type-status-obsolete"
	DiagGroupMembership              = "group-membership"
	DiagGroupMemberMixed             = "group-member-mixed"
	DiagGroupObjectsNotification     = "group-objects-notification"
	DiagGroupNotificationsObject     = "group-notifications-object"
	DiagGroupObjectStatus            = "group-object-status"
	DiagComplianceGroupStatus        = "compliance-group-status"
	DiagComplianceObjectStatus       = "compliance-object-status"
	DiagComplianceGroupInvalid       = "compliance-group-invalid"
	DiagRefinementExists             = "refinement-exists"
	DiagOptionalGroupExists          = "optional-group-exists"
	DiagRefinementNotListed          = "refinement-not-listed"
	DiagComplianceMemberNotLocal     = "compliance-member-not-local"
	DiagTimeticksRangeIllegal        = "timeticks-range-illegal"
	DiagStatusInvalidCapabilities    = "status-invalid-capabilities"
	DiagImportUnused                 = "import-unused"
	DiagBasetypeNotImported          = "basetype-not-imported"
	DiagDescriptionMissing           = "description-missing"
	DiagTCNested                     = "textual-convention-nested"
	DiagTypeAssignmentSMIv2          = "type-assignment-smiv2"
	DiagTableNameTable               = "table-name-table"
	DiagRowNameEntry                 = "row-name-entry"
	DiagRowNameTableName             = "row-name-table-name"
	DiagNamedNumbersAscending        = "named-numbers-ascending"
	DiagHyphenInLabel                = "hyphen-in-label"
	DiagOpaqueSMIv2                  = "opaque-smiv2"
	DiagInvalidFormat                = "invalid-format"
	DiagTypeWithoutFormat            = "type-without-format"
	DiagTypeUnreferenced             = "type-unref"
	DiagGroupUnreferenced            = "group-unref"
	DiagImportDuplicate              = "import-duplicate"
	DiagObsoleteImport               = "obsolete-import"
	DiagIdentifierCaseMatch          = "identifier-case-match"
	DiagTrapInSMIv2                  = "trap-in-smiv2"
	DiagNodeImplicit                 = "node-implicit"
	DiagModuleIdentityReg            = "module-identity-registration"
	DiagRowStatusDefault             = "rowstatus-default"
	DiagRowStatusAccess              = "rowstatus-access"
	DiagStorageTypeDefault           = "storagetype-default"
	DiagTAddressTDomain              = "taddress-tdomain"
	DiagIndexAccessible              = "index-accessible"
	DiagIndexNotAccessible           = "index-not-accessible"
	DiagIndexDefval                  = "index-defval"
	DiagAccessWriteOnlySMIv1         = "access-write-only-smiv1"
	DiagIpAddressInSyntax            = "ipaddress-in-syntax"
	DiagInetAddressPairing           = "inetaddress-inetaddresstype"
	DiagInetAddressTypeSubtyped      = "inetaddresstype-subtyped"
	DiagInetAddressSpecific          = "inetaddress-specific"
	DiagTransportAddressPairing      = "transportaddress-transportaddresstype"
	DiagTransportAddressTypeSubtyped = "transportaddresstype-subtyped"
	DiagTransportAddressSpecific     = "transportaddress-specific"
	DiagIncludesUnresolved           = "includes-unresolved"
	DiagIncludesDuplicate            = "includes-duplicate"
	DiagCreationRequiresUnresolved   = "creation-requires-unresolved"
	DiagCreationRequiresDuplicate    = "creation-requires-duplicate"
)

// codeEntry pairs a diagnostic code with its severity.
type codeEntry struct {
	code     string
	severity Severity
}

var lexerDiagCodes = []codeEntry{
	{DiagUnexpectedCharacter, SeverityError},
	{DiagUnterminatedString, SeverityError},
	{DiagUnterminatedHexBinStr, SeverityError},
	{DiagMissingHexBinSuffix, SeverityError},
	{DiagHexStringInvalidChar, SeverityWarning},
	{DiagBinStringInvalidChar, SeverityWarning},
	{DiagHexStringMul2, SeverityWarning},
	{DiagBinStringMul8, SeverityWarning},
}

var parserDiagCodes = []codeEntry{
	{DiagIdentifierUnderscore, SeverityStyle},
	{DiagIdentifierHyphenEnd, SeverityError},
	{DiagIdentifierLength64, SeverityError},
	{DiagIdentifierLength32, SeverityWarning},
	{DiagBadIdentifierCase, SeverityError},
	{DiagParseError, SeverityError},
	{DiagInvalidU32, SeverityError},
	{DiagInvalidI64, SeverityError},
	{DiagKeywordReserved, SeveritySevere},
	{DiagInvalidHexRange, SeverityError},
	{DiagNumberLeadingZero, SeverityMinor},
	{DiagMissingComma, SeverityMinor},
}

var validateDiagCodes = []codeEntry{
	{DiagMissingModuleIdentity, SeverityWarning},
	{DiagRevisionLastUpdated, SeverityMinor},
	{DiagRevisionNotDescending, SeverityMinor},
	{DiagRevisionAfterUpdate, SeverityMinor},
	{DiagDateCharacter, SeverityError},
	{DiagDateLength, SeverityError},
	{DiagDateMonth, SeverityError},
	{DiagDateDay, SeverityError},
	{DiagDateHour, SeverityError},
	{DiagDateMinutes, SeverityError},
	{DiagDateValue, SeverityError},
	{DiagDateYear2Digits, SeverityWarning},
	{DiagDateInFuture, SeverityStyle},
	{DiagDateInPast, SeverityStyle},
	{DiagUnknownDefinitionType, SeverityWarning},
	{DiagUnknownTypeSyntax, SeverityWarning},
	{DiagUnknownConstraintType, SeverityWarning},
	{DiagUnknownRangeValue, SeverityWarning},
	{DiagUnknownOidComponent, SeverityWarning},
	{DiagUnknownDefvalType, SeverityWarning},
	{DiagBitsNumberNegative, SeverityError},
	{DiagBitsNumberTooLarge, SeverityError},
	{DiagBitsNumberLarge, SeverityStyle},
	{DiagEnumZero, SeverityError},
	{DiagEnumNameRedefinition, SeverityError},
	{DiagEnumValueRedefinition, SeverityError},
	{DiagBitsNameRedefinition, SeverityError},
	{DiagBitsValueRedefinition, SeverityError},
	{DiagModuleIdentityNotFirst, SeverityWarning},
	{DiagModuleIdentityMultiple, SeverityError},
	{DiagMacroNotImported, SeverityMinor},
	{DiagEmptyDescription, SeverityStyle},
	{DiagEmptyReference, SeverityStyle},
	{DiagEmptyOrganization, SeverityStyle},
	{DiagEmptyContact, SeverityStyle},
	{DiagEmptyUnits, SeverityStyle},
	{DiagEmptyFormat, SeverityStyle},
	{DiagModuleNameSuffix, SeverityStyle},
	{DiagMacroNotAllowed, SeverityWarning},
	{DiagChoiceNotAllowed, SeverityWarning},
	{DiagTaggedTypeNotAllowed, SeverityWarning},
}

var resolverDiagCodes = []codeEntry{
	{DiagImportNotFound, SeverityError},
	{DiagImportModuleNotFound, SeverityError},
	{DiagTypeUnknown, SeverityError},
	{DiagTypeCycle, SeverityError},
	{DiagOidOrphan, SeverityError},
	{DiagTrapNumberOverflow, SeverityError},
	{DiagIndexUnresolved, SeverityError},
	{DiagObjectsUnresolved, SeverityError},
	{DiagIdentifierHyphenSMI, SeverityWarning},
	{DiagGroupNotAccessible, SeverityMinor},
	{DiagNotifObjectNotObject, SeverityMinor},
	{DiagNotifObjectAccess, SeverityMinor},
	{DiagNotifNotReversible, SeverityWarning},
	{DiagNotifIdTooLarge, SeverityWarning},
	{DiagMalformedHexDefval, SeverityWarning},
	{DiagMalformedBinDefval, SeverityWarning},
	{DiagDefvalUnresolved, SeverityWarning},
	{DiagVariationAccessNotifOnly, SeverityMinor},
	{DiagGroupMemberUnresolved, SeverityMinor},
	{DiagIndexNotObject, SeverityMinor},
	{DiagAugmentsNotObject, SeverityMinor},
	{DiagAugmentNested, SeverityError},
	{DiagNotifNoOid, SeverityMinor},
	{DiagPrimitiveTypeMissing, SeverityError},
	{DiagIntegerInSMIv2, SeverityWarning},
	{DiagIndexIntegerNoRange, SeverityError},
	{DiagIndexNegativeRange, SeverityError},
	{DiagDefvalBasetype, SeverityWarning},
	{DiagDefvalRange, SeverityWarning},
	{DiagDefvalEnum, SeverityWarning},
	{DiagDefvalBits, SeverityWarning},
	{DiagCounterDefvalIllegal, SeverityWarning},
	{DiagIndexCounterIllegal, SeverityWarning},
	{DiagRangeBounds, SeverityError},
	{DiagRangeExchanged, SeverityError},
	{DiagRangeOverlap, SeverityError},
	{DiagRangeAscending, SeverityWarning},
	{DiagSizeIllegal, SeverityError},
	{DiagRangeIllegal, SeverityError},
	{DiagCounterRangeIllegal, SeverityError},
	{DiagSubtypeEnumIllegal, SeverityError},
	{DiagSubtypeBitsIllegal, SeverityError},
	{DiagParentTable, SeverityError},
	{DiagParentRow, SeverityError},
	{DiagParentColumn, SeverityError},
	{DiagParentScalar, SeverityError},
	{DiagParentNode, SeverityError},
	{DiagParentNotification, SeverityError},
	{DiagParentGroup, SeverityError},
	{DiagParentCompliance, SeverityError},
	{DiagParentCapabilities, SeverityError},
	{DiagRowSubidentifierOne, SeverityError},
	{DiagIndexElementNoSize, SeverityMinor},
	{DiagIndexIllegalBasetype, SeveritySevere},
	{DiagLastSubidZero, SeveritySevere},
	{DiagOidRecursive, SeverityError},
	{DiagOidRegistered, SeveritySevere},
	{DiagOidReuse, SeverityWarning},
	{DiagSequenceNoColumn, SeverityMinor},
	{DiagSequenceMissingColumn, SeverityMinor},
	{DiagSequenceOrder, SeverityWarning},
	{DiagSequenceTypeMismatch, SeverityError},
	{DiagIndexExceedsTooLarge, SeverityWarning},
	{DiagAccessInvalidSMIv1, SeverityError},
	{DiagAccessWriteOnlySMIv2, SeverityError},
	{DiagAccessTableIllegal, SeverityMinor},
	{DiagAccessRowIllegal, SeverityMinor},
	{DiagAccessCounterIllegal, SeverityStyle},
	{DiagScalarNotCreatable, SeverityMinor},
	{DiagMaxAccessInSMIv1, SeverityError},
	{DiagAccessInSMIv2, SeverityError},
	{DiagStatusInvalidSMIv1, SeverityError},
	{DiagStatusInvalidSMIv2, SeverityError},
	{DiagTypeStatusDeprecated, SeverityWarning},
	{DiagTypeStatusObsolete, SeverityWarning},
	{DiagGroupMembership, SeverityMinor},
	{DiagGroupMemberMixed, SeverityMinor},
	{DiagGroupObjectsNotification, SeverityMinor},
	{DiagGroupNotificationsObject, SeverityMinor},
	{DiagGroupObjectStatus, SeverityWarning},
	{DiagComplianceGroupStatus, SeverityWarning},
	{DiagComplianceObjectStatus, SeverityWarning},
	{DiagComplianceGroupInvalid, SeverityWarning},
	{DiagRefinementExists, SeverityWarning},
	{DiagOptionalGroupExists, SeverityWarning},
	{DiagRefinementNotListed, SeverityWarning},
	{DiagComplianceMemberNotLocal, SeverityWarning},
	{DiagTimeticksRangeIllegal, SeverityError},
	{DiagStatusInvalidCapabilities, SeverityError},
	{DiagImportDuplicate, SeverityMinor},
	{DiagImportUnused, SeverityStyle},
	{DiagBasetypeNotImported, SeverityMinor},
	{DiagDescriptionMissing, SeverityMinor},
	{DiagTCNested, SeverityStyle},
	{DiagTypeAssignmentSMIv2, SeverityStyle},
	{DiagTableNameTable, SeverityStyle},
	{DiagRowNameEntry, SeverityStyle},
	{DiagRowNameTableName, SeverityStyle},
	{DiagNamedNumbersAscending, SeverityStyle},
	{DiagHyphenInLabel, SeverityStyle},
	{DiagOpaqueSMIv2, SeverityWarning},
	{DiagInvalidFormat, SeverityError},
	{DiagTypeWithoutFormat, SeverityStyle},
	{DiagTypeUnreferenced, SeverityStyle},
	{DiagGroupUnreferenced, SeverityStyle},
	{DiagObsoleteImport, SeverityWarning},
	{DiagIdentifierCaseMatch, SeverityStyle},
	{DiagTrapInSMIv2, SeverityWarning},
	{DiagNodeImplicit, SeverityStyle},
	{DiagModuleIdentityReg, SeverityWarning},
	{DiagRowStatusDefault, SeverityStyle},
	{DiagRowStatusAccess, SeverityStyle},
	{DiagStorageTypeDefault, SeverityStyle},
	{DiagTAddressTDomain, SeverityWarning},
	{DiagIndexAccessible, SeverityMinor},
	{DiagIndexNotAccessible, SeverityMinor},
	{DiagIndexDefval, SeverityWarning},
	{DiagAccessWriteOnlySMIv1, SeverityStyle},
	{DiagIpAddressInSyntax, SeverityStyle},
	{DiagInetAddressPairing, SeverityWarning},
	{DiagInetAddressTypeSubtyped, SeverityWarning},
	{DiagInetAddressSpecific, SeverityStyle},
	{DiagTransportAddressPairing, SeverityWarning},
	{DiagTransportAddressTypeSubtyped, SeverityWarning},
	{DiagTransportAddressSpecific, SeverityStyle},
	{DiagIncludesUnresolved, SeverityWarning},
	{DiagIncludesDuplicate, SeverityWarning},
	{DiagCreationRequiresUnresolved, SeverityWarning},
	{DiagCreationRequiresDuplicate, SeverityWarning},
}

// diagPhases maps phase names to their diagnostic code entries.
var diagPhases = []struct {
	name  string
	codes []codeEntry
}{
	{"lexer", lexerDiagCodes},
	{"parser", parserDiagCodes},
	{"validate", validateDiagCodes},
	{"resolver", resolverDiagCodes},
}

// codeSeverity maps diagnostic codes to their fixed severity.
var codeSeverity map[string]Severity

func init() {
	codeSeverity = make(map[string]Severity)
	for _, phase := range diagPhases {
		for _, entry := range phase.codes {
			if _, exists := codeSeverity[entry.code]; exists {
				panic("duplicate diagnostic code: " + entry.code)
			}
			codeSeverity[entry.code] = entry.severity
		}
	}
}

// SeverityForCode returns the fixed severity for a diagnostic code.
// Panics if the code is not registered - this is a programmer error.
func SeverityForCode(code string) Severity {
	sev, ok := codeSeverity[code]
	if !ok {
		panic("unknown diagnostic code: " + code)
	}
	return sev
}

// AllDiagnosticCodes returns all known diagnostic codes grouped by phase.
func AllDiagnosticCodes() []DiagCodeInfo {
	var result []DiagCodeInfo
	for _, phase := range diagPhases {
		for _, entry := range phase.codes {
			result = append(result, DiagCodeInfo{
				Code:     entry.code,
				Phase:    phase.name,
				Severity: entry.severity,
			})
		}
	}
	return result
}

// DiagCodeInfo describes a diagnostic code, its phase, and its severity.
type DiagCodeInfo struct {
	Code     string
	Phase    string
	Severity Severity
}
