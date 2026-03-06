package types

// ClauseInfo describes an SMI clause keyword used within a macro definition.
type ClauseInfo struct {
	Name        string   // e.g. "SYNTAX"
	Macros      []string // macro(s) that use this clause, e.g. ["OBJECT-TYPE", "TEXTUAL-CONVENTION"]
	RFC         string   // e.g. "RFC 2578"
	Description string   // brief description
}

var clauseInfoTable = map[string]ClauseInfo{
	"SYNTAX": {
		Name:        "SYNTAX",
		Macros:      []string{"OBJECT-TYPE", "TEXTUAL-CONVENTION", "MODULE-COMPLIANCE", "AGENT-CAPABILITIES"},
		RFC:         "RFC 2578",
		Description: "Specifies the ASN.1 type of a managed object or textual convention.",
	},
	"MAX-ACCESS": {
		Name:        "MAX-ACCESS",
		Macros:      []string{"OBJECT-TYPE"},
		RFC:         "RFC 2578",
		Description: "Specifies the maximum access level for a managed object.",
	},
	"MIN-ACCESS": {
		Name:        "MIN-ACCESS",
		Macros:      []string{"MODULE-COMPLIANCE"},
		RFC:         "RFC 2580",
		Description: "Specifies the minimum required access level when granting a compliance exception.",
	},
	"ACCESS": {
		Name:        "ACCESS",
		Macros:      []string{"OBJECT-TYPE", "AGENT-CAPABILITIES"},
		RFC:         "RFC 1155",
		Description: "Specifies the access level for an SMIv1 managed object or an agent capability variation.",
	},
	"STATUS": {
		Name:        "STATUS",
		Macros:      []string{"OBJECT-TYPE", "MODULE-IDENTITY", "OBJECT-IDENTITY", "NOTIFICATION-TYPE", "TEXTUAL-CONVENTION", "OBJECT-GROUP", "NOTIFICATION-GROUP", "MODULE-COMPLIANCE", "AGENT-CAPABILITIES"},
		RFC:         "RFC 2578",
		Description: "Indicates the lifecycle state of a definition: current, deprecated, or obsolete.",
	},
	"DESCRIPTION": {
		Name:        "DESCRIPTION",
		Macros:      []string{"OBJECT-TYPE", "MODULE-IDENTITY", "OBJECT-IDENTITY", "NOTIFICATION-TYPE", "TEXTUAL-CONVENTION", "OBJECT-GROUP", "NOTIFICATION-GROUP", "MODULE-COMPLIANCE", "AGENT-CAPABILITIES", "TRAP-TYPE"},
		RFC:         "RFC 2578",
		Description: "Provides a human-readable text description of the definition.",
	},
	"REFERENCE": {
		Name:        "REFERENCE",
		Macros:      []string{"OBJECT-TYPE", "OBJECT-IDENTITY", "NOTIFICATION-TYPE", "TEXTUAL-CONVENTION", "OBJECT-GROUP", "NOTIFICATION-GROUP", "MODULE-COMPLIANCE", "AGENT-CAPABILITIES", "TRAP-TYPE"},
		RFC:         "RFC 2578",
		Description: "Provides a cross-reference to another document or definition.",
	},
	"DEFVAL": {
		Name:        "DEFVAL",
		Macros:      []string{"OBJECT-TYPE", "AGENT-CAPABILITIES"},
		RFC:         "RFC 2578",
		Description: "Specifies an acceptable default value for a managed object when creating a new row.",
	},
	"UNITS": {
		Name:        "UNITS",
		Macros:      []string{"OBJECT-TYPE"},
		RFC:         "RFC 2578",
		Description: "Specifies a textual description of the units (e.g. \"seconds\") associated with the object.",
	},
	"INDEX": {
		Name:        "INDEX",
		Macros:      []string{"OBJECT-TYPE"},
		RFC:         "RFC 2578",
		Description: "Specifies the set of objects whose values uniquely identify a conceptual row in a table.",
	},
	"IMPLIED": {
		Name:        "IMPLIED",
		Macros:      []string{"OBJECT-TYPE"},
		RFC:         "RFC 2578",
		Description: "Indicates the last INDEX object has an implied length, omitting the length prefix from the instance identifier.",
	},
	"AUGMENTS": {
		Name:        "AUGMENTS",
		Macros:      []string{"OBJECT-TYPE"},
		RFC:         "RFC 2578",
		Description: "Specifies that this row extends (augments) another table's row, sharing its INDEX.",
	},
	"DISPLAY-HINT": {
		Name:        "DISPLAY-HINT",
		Macros:      []string{"TEXTUAL-CONVENTION"},
		RFC:         "RFC 2579",
		Description: "Provides a hint for how the value should be displayed to a human operator.",
	},
	"OBJECTS": {
		Name:        "OBJECTS",
		Macros:      []string{"NOTIFICATION-TYPE", "OBJECT-GROUP"},
		RFC:         "RFC 2578",
		Description: "Lists the objects associated with a notification or the members of a group.",
	},
	"NOTIFICATIONS": {
		Name:        "NOTIFICATIONS",
		Macros:      []string{"NOTIFICATION-GROUP"},
		RFC:         "RFC 2580",
		Description: "Lists the notifications that are members of this group.",
	},
	"MODULE": {
		Name:        "MODULE",
		Macros:      []string{"MODULE-COMPLIANCE"},
		RFC:         "RFC 2580",
		Description: "Identifies a module and its mandatory groups within a compliance statement.",
	},
	"MANDATORY-GROUPS": {
		Name:        "MANDATORY-GROUPS",
		Macros:      []string{"MODULE-COMPLIANCE"},
		RFC:         "RFC 2580",
		Description: "Lists the groups that must be implemented for compliance with this module.",
	},
	"GROUP": {
		Name:        "GROUP",
		Macros:      []string{"MODULE-COMPLIANCE"},
		RFC:         "RFC 2580",
		Description: "Specifies a conditionally mandatory group within a compliance statement.",
	},
	"OBJECT": {
		Name:        "OBJECT",
		Macros:      []string{"MODULE-COMPLIANCE"},
		RFC:         "RFC 2580",
		Description: "Names an object for which compliance refinements (syntax, access, write-syntax) are specified.",
	},
	"WRITE-SYNTAX": {
		Name:        "WRITE-SYNTAX",
		Macros:      []string{"MODULE-COMPLIANCE", "AGENT-CAPABILITIES"},
		RFC:         "RFC 2580",
		Description: "Specifies a restricted syntax that applies when writing (setting) this object.",
	},
	"PRODUCT-RELEASE": {
		Name:        "PRODUCT-RELEASE",
		Macros:      []string{"AGENT-CAPABILITIES"},
		RFC:         "RFC 2580",
		Description: "Identifies the product release associated with the agent capabilities statement.",
	},
	"SUPPORTS": {
		Name:        "SUPPORTS",
		Macros:      []string{"AGENT-CAPABILITIES"},
		RFC:         "RFC 2580",
		Description: "Identifies a MIB module supported by the agent.",
	},
	"INCLUDES": {
		Name:        "INCLUDES",
		Macros:      []string{"AGENT-CAPABILITIES"},
		RFC:         "RFC 2580",
		Description: "Lists the groups from a supported module that the agent implements.",
	},
	"VARIATION": {
		Name:        "VARIATION",
		Macros:      []string{"AGENT-CAPABILITIES"},
		RFC:         "RFC 2580",
		Description: "Describes a deviation from the standard definition of an object.",
	},
	"CREATION-REQUIRES": {
		Name:        "CREATION-REQUIRES",
		Macros:      []string{"AGENT-CAPABILITIES"},
		RFC:         "RFC 2580",
		Description: "Lists the objects that must be set to create a new conceptual row.",
	},
	"REVISION": {
		Name:        "REVISION",
		Macros:      []string{"MODULE-IDENTITY"},
		RFC:         "RFC 2578",
		Description: "Documents a revision of the MIB module with a date and description.",
	},
	"LAST-UPDATED": {
		Name:        "LAST-UPDATED",
		Macros:      []string{"MODULE-IDENTITY"},
		RFC:         "RFC 2578",
		Description: "Specifies the date and time the module was last modified.",
	},
	"ORGANIZATION": {
		Name:        "ORGANIZATION",
		Macros:      []string{"MODULE-IDENTITY"},
		RFC:         "RFC 2578",
		Description: "Identifies the organization responsible for the MIB module.",
	},
	"CONTACT-INFO": {
		Name:        "CONTACT-INFO",
		Macros:      []string{"MODULE-IDENTITY"},
		RFC:         "RFC 2578",
		Description: "Provides contact information for the module's maintainer.",
	},
	"ENTERPRISE": {
		Name:        "ENTERPRISE",
		Macros:      []string{"TRAP-TYPE"},
		RFC:         "RFC 1215",
		Description: "Specifies the enterprise OID under which the SMIv1 trap is defined.",
	},
	"VARIABLES": {
		Name:        "VARIABLES",
		Macros:      []string{"TRAP-TYPE"},
		RFC:         "RFC 1215",
		Description: "Lists the objects included in an SMIv1 trap.",
	},
	"SIZE": {
		Name:        "SIZE",
		Macros:      []string{"OBJECT-TYPE", "TEXTUAL-CONVENTION"},
		RFC:         "RFC 2578",
		Description: "Constrains the length of an OCTET STRING or the number of elements in a BITS value.",
	},
}

// ClauseDescription returns info about an SMI clause keyword, if known.
func ClauseDescription(name string) (ClauseInfo, bool) {
	info, ok := clauseInfoTable[name]
	return info, ok
}
