package lexer

import "sort"

// keywords is the sorted keyword table for binary search.
// IMPORTANT: This slice MUST remain sorted alphabetically by text.
// ASCII byte order: uppercase letters (A-Z: 65-90) come before
// lowercase letters (a-z: 97-122). Hyphen (45) comes before digits (48-57)
// and letters.
var keywords = []struct {
	text string
	kind TokenKind
}{
	{"ACCESS", TokKwAccess},
	{"AGENT-CAPABILITIES", TokKwAgentCapabilities},
	{"APPLICATION", TokKwApplication},
	{"AUGMENTS", TokKwAugments},
	{"BEGIN", TokKwBegin},
	{"BITS", TokKwBits},
	{"CHOICE", TokKwChoice},
	{"CONTACT-INFO", TokKwContactInfo},
	{"CREATION-REQUIRES", TokKwCreationRequires},
	{"Counter", TokKwCounter},
	{"Counter32", TokKwCounter32},
	{"Counter64", TokKwCounter64},
	{"DEFINITIONS", TokKwDefinitions},
	{"DEFVAL", TokKwDefval},
	{"DESCRIPTION", TokKwDescription},
	{"DISPLAY-HINT", TokKwDisplayHint},
	{"END", TokKwEnd},
	{"ENTERPRISE", TokKwEnterprise},
	{"EXPORTS", TokKwExports},
	{"FROM", TokKwFrom},
	{"GROUP", TokKwGroup},
	{"Gauge", TokKwGauge},
	{"Gauge32", TokKwGauge32},
	{"IDENTIFIER", TokKwIdentifier},
	{"IMPLICIT", TokKwImplicit},
	{"IMPLIED", TokKwImplied},
	{"IMPORTS", TokKwImports},
	{"INCLUDES", TokKwIncludes},
	{"INDEX", TokKwIndex},
	{"INTEGER", TokKwInteger},
	{"Integer32", TokKwInteger32},
	{"IpAddress", TokKwIpAddress},
	{"LAST-UPDATED", TokKwLastUpdated},
	{"MACRO", TokKwMacro},
	{"MANDATORY-GROUPS", TokKwMandatoryGroups},
	{"MAX-ACCESS", TokKwMaxAccess},
	{"MIN-ACCESS", TokKwMinAccess},
	{"MODULE", TokKwModule},
	{"MODULE-COMPLIANCE", TokKwModuleCompliance},
	{"MODULE-IDENTITY", TokKwModuleIdentity},
	{"NOTIFICATION-GROUP", TokKwNotificationGroup},
	{"NOTIFICATION-TYPE", TokKwNotificationType},
	{"NOTIFICATIONS", TokKwNotifications},
	{"NetworkAddress", TokKwNetworkAddress},
	{"OBJECT", TokKwObject},
	{"OBJECT-GROUP", TokKwObjectGroup},
	{"OBJECT-IDENTITY", TokKwObjectIdentity},
	{"OBJECT-TYPE", TokKwObjectType},
	{"OBJECTS", TokKwObjects},
	{"OCTET", TokKwOctet},
	{"OF", TokKwOf},
	{"ORGANIZATION", TokKwOrganization},
	{"Opaque", TokKwOpaque},
	{"PRODUCT-RELEASE", TokKwProductRelease},
	{"REFERENCE", TokKwReference},
	{"REVISION", TokKwRevision},
	{"SEQUENCE", TokKwSequence},
	{"SIZE", TokKwSize},
	{"STATUS", TokKwStatus},
	{"STRING", TokKwString},
	{"SUPPORTS", TokKwSupports},
	{"SYNTAX", TokKwSyntax},
	{"TEXTUAL-CONVENTION", TokKwTextualConvention},
	{"TRAP-TYPE", TokKwTrapType},
	{"TimeTicks", TokKwTimeTicks},
	{"UNITS", TokKwUnits},
	{"UNIVERSAL", TokKwUniversal},
	{"Unsigned32", TokKwUnsigned32},
	{"VARIABLES", TokKwVariables},
	{"VARIATION", TokKwVariation},
	{"WRITE-SYNTAX", TokKwWriteSyntax},
	{"accessible-for-notify", TokKwAccessibleForNotify},
	{"current", TokKwCurrent},
	{"deprecated", TokKwDeprecated},
	{"mandatory", TokKwMandatory},
	{"not-accessible", TokKwNotAccessible},
	{"not-implemented", TokKwNotImplemented},
	{"obsolete", TokKwObsolete},
	{"optional", TokKwOptional},
	{"read-create", TokKwReadCreate},
	{"read-only", TokKwReadOnly},
	{"read-write", TokKwReadWrite},
	{"write-only", TokKwWriteOnly},
}

// LookupKeyword returns the TokenKind for a keyword, or (TokError, false) if not found.
func LookupKeyword(text string) (TokenKind, bool) {
	idx := sort.Search(len(keywords), func(i int) bool {
		return keywords[i].text >= text
	})
	if idx < len(keywords) && keywords[idx].text == text {
		return keywords[idx].kind, true
	}
	return TokError, false
}

// forbiddenKeywords is the sorted list of forbidden ASN.1 keywords.
// Per libsmi scanner-smi.l:699-705, these keywords emit errors.
// They are ASN.1 reserved words that have no meaning in SMI context.
// IMPORTANT: This slice MUST remain sorted alphabetically for binary search.
var forbiddenKeywords = []string{
	"ABSENT",
	"ANY",
	"BIT",
	"BOOLEAN",
	"BY",
	"COMPONENT",
	"COMPONENTS",
	"DEFAULT",
	"DEFINED",
	"ENUMERATED",
	"EXPLICIT",
	"EXTERNAL",
	"FALSE",
	"MAX",
	"MIN",
	"MINUS-INFINITY",
	"NULL",
	"OPTIONAL",
	"PLUS-INFINITY",
	"PRESENT",
	"PRIVATE",
	"REAL",
	"SET",
	"TAGS",
	"TRUE",
	"WITH",
}

// IsForbiddenKeyword returns true if the text is a forbidden ASN.1 keyword.
func IsForbiddenKeyword(text string) bool {
	idx := sort.SearchStrings(forbiddenKeywords, text)
	return idx < len(forbiddenKeywords) && forbiddenKeywords[idx] == text
}
