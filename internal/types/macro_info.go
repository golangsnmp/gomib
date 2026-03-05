package types

// MacroInfo describes an SMI macro keyword.
type MacroInfo struct {
	Name        string // e.g. "OBJECT-TYPE"
	Module      string // defining module, e.g. "SNMPv2-SMI"
	RFC         string // e.g. "RFC 2578"
	Description string // brief description
}

var macroInfoTable = map[string]MacroInfo{
	"OBJECT-TYPE": {
		Name:        "OBJECT-TYPE",
		Module:      "SNMPv2-SMI",
		RFC:         "RFC 2578",
		Description: "Defines a managed object: its syntax, access level, status, and position in the OID tree.",
	},
	"MODULE-IDENTITY": {
		Name:        "MODULE-IDENTITY",
		Module:      "SNMPv2-SMI",
		RFC:         "RFC 2578",
		Description: "Provides contact, revision history, and description metadata for a MIB module. Also assigns the module's root OID.",
	},
	"OBJECT-IDENTITY": {
		Name:        "OBJECT-IDENTITY",
		Module:      "SNMPv2-SMI",
		RFC:         "RFC 2578",
		Description: "Assigns a name and description to an OID without defining a managed object. Used for administrative OID registrations.",
	},
	"NOTIFICATION-TYPE": {
		Name:        "NOTIFICATION-TYPE",
		Module:      "SNMPv2-SMI",
		RFC:         "RFC 2578",
		Description: "Defines an SNMPv2 notification (trap) with a list of associated objects.",
	},
	"TEXTUAL-CONVENTION": {
		Name:        "TEXTUAL-CONVENTION",
		Module:      "SNMPv2-TC",
		RFC:         "RFC 2579",
		Description: "Defines a named type with a display hint, status, and description. Used to give semantic meaning to base types.",
	},
	"TRAP-TYPE": {
		Name:        "TRAP-TYPE",
		Module:      "RFC-1215",
		RFC:         "RFC 1215",
		Description: "Defines an SNMPv1 trap with an enterprise OID and trap number. Superseded by NOTIFICATION-TYPE in SMIv2.",
	},
	"OBJECT-GROUP": {
		Name:        "OBJECT-GROUP",
		Module:      "SNMPv2-CONF",
		RFC:         "RFC 2580",
		Description: "Defines a collection of related OBJECT-TYPE definitions for conformance purposes.",
	},
	"NOTIFICATION-GROUP": {
		Name:        "NOTIFICATION-GROUP",
		Module:      "SNMPv2-CONF",
		RFC:         "RFC 2580",
		Description: "Defines a collection of related NOTIFICATION-TYPE definitions for conformance purposes.",
	},
	"MODULE-COMPLIANCE": {
		Name:        "MODULE-COMPLIANCE",
		Module:      "SNMPv2-CONF",
		RFC:         "RFC 2580",
		Description: "Specifies minimum conformance requirements for implementing a MIB module, including mandatory groups and optional refinements.",
	},
	"AGENT-CAPABILITIES": {
		Name:        "AGENT-CAPABILITIES",
		Module:      "SNMPv2-CONF",
		RFC:         "RFC 2580",
		Description: "Documents the exact MIB support provided by an SNMP agent, including supported modules and any variations from full compliance.",
	},
}

// MacroDescription returns info about an SMI macro keyword, if known.
func MacroDescription(name string) (MacroInfo, bool) {
	info, ok := macroInfoTable[name]
	return info, ok
}
