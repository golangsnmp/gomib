package module

import (
	"math"
	"slices"
	"sync"

	"github.com/golangsnmp/gomib/internal/types"
)

// BaseModule identifies a well-known SMI base module (types and MACROs).
type BaseModule int

const (
	// BaseModuleSNMPv2SMI is SNMPv2-SMI (RFC 2578) - SMIv2 base types, OIDs, MACROs.
	BaseModuleSNMPv2SMI BaseModule = iota
	// BaseModuleSNMPv2TC is SNMPv2-TC (RFC 2579) - Textual conventions.
	BaseModuleSNMPv2TC
	// BaseModuleSNMPv2CONF is SNMPv2-CONF (RFC 2580) - Conformance MACROs.
	BaseModuleSNMPv2CONF
	// BaseModuleRFC1155SMI is RFC1155-SMI - SMIv1 base types, OIDs.
	BaseModuleRFC1155SMI
	// BaseModuleRFC1065SMI is RFC1065-SMI - Original SMIv1 base (predates RFC 1155).
	BaseModuleRFC1065SMI
	// BaseModuleRFC1212 is RFC-1212 - SMIv1 OBJECT-TYPE MACRO.
	BaseModuleRFC1212
	// BaseModuleRFC1215 is RFC-1215 - SMIv1 TRAP-TYPE MACRO.
	BaseModuleRFC1215
)

// Order matches the BaseModule iota constants.
var baseModuleNames = [...]string{
	"SNMPv2-SMI",
	"SNMPv2-TC",
	"SNMPv2-CONF",
	"RFC1155-SMI",
	"RFC1065-SMI",
	"RFC-1212",
	"RFC-1215",
}

// Name returns the canonical module name.
func (m BaseModule) Name() string {
	if int(m) < len(baseModuleNames) {
		return baseModuleNames[m]
	}
	return ""
}

// IsSMIv2 reports whether this is an SMIv2 base module.
func (m BaseModule) IsSMIv2() bool {
	switch m {
	case BaseModuleSNMPv2SMI, BaseModuleSNMPv2TC, BaseModuleSNMPv2CONF:
		return true
	default:
		return false
	}
}

// IsSMIv1 reports whether this is an SMIv1 base module.
func (m BaseModule) IsSMIv1() bool {
	switch m {
	case BaseModuleRFC1155SMI, BaseModuleRFC1065SMI, BaseModuleRFC1212, BaseModuleRFC1215:
		return true
	default:
		return false
	}
}

var baseModuleByName = func() map[string]BaseModule {
	m := make(map[string]BaseModule, len(baseModuleNames))
	for i, name := range baseModuleNames {
		m[name] = BaseModule(i)
	}
	return m
}()

// BaseModuleFromName returns the BaseModule for the given name, if any.
func BaseModuleFromName(name string) (BaseModule, bool) {
	m, ok := baseModuleByName[name]
	return m, ok
}

// IsBaseModule reports whether name is a recognized base module.
func IsBaseModule(name string) bool {
	_, ok := BaseModuleFromName(name)
	return ok
}

var cachedBaseModules = sync.OnceValue(func() map[string]*Module {
	m := make(map[string]*Module, len(baseModuleNames))
	for _, mod := range CreateBaseModules() {
		m[mod.Name] = mod
	}
	return m
})

// GetBaseModule returns the Module for the named base module, or nil.
func GetBaseModule(name string) *Module {
	return cachedBaseModules()[name]
}

// AllBaseModules returns every BaseModule constant.
func AllBaseModules() []BaseModule {
	result := make([]BaseModule, len(baseModuleNames))
	for i := range baseModuleNames {
		result[i] = BaseModule(i)
	}
	return result
}

// BaseModuleNames returns the canonical names of all base modules.
func BaseModuleNames() []string {
	return slices.Clone(baseModuleNames[:])
}

// CreateBaseModules returns synthetic Module values for all base modules.
// These should be prepended to the user module list before resolution.
func CreateBaseModules() []*Module {
	return []*Module{
		createSNMPv2SMI(),
		createSNMPv2TC(),
		createSNMPv2CONF(),
		createSMIv1Base("RFC1155-SMI"),
		createSMIv1Base("RFC1065-SMI"),
		createRFC1212(),
		createRFC1215(),
	}
}

func createSNMPv2SMI() *Module {
	module := NewModule("SNMPv2-SMI", types.Synthetic)
	module.Language = types.LanguageSMIv2

	module.Definitions = append(module.Definitions, createOidDefinitions()...)
	module.Definitions = append(module.Definitions, createBaseTypeDefinitions()...)

	return module
}

func createSNMPv2TC() *Module {
	module := NewModule("SNMPv2-TC", types.Synthetic)
	module.Language = types.LanguageSMIv2

	module.Imports = []Import{
		NewImport("SNMPv2-SMI", "TimeTicks", types.Synthetic),
	}

	module.Definitions = append(module.Definitions, createTCDefinitions()...)

	return module
}

func createSNMPv2CONF() *Module {
	module := NewModule("SNMPv2-CONF", types.Synthetic)
	module.Language = types.LanguageSMIv2
	// No definitions - MACROs only
	return module
}

func createSMIv1Base(name string) *Module {
	module := NewModule(name, types.Synthetic)
	module.Language = types.LanguageSMIv1

	module.Definitions = append(module.Definitions, createSMIv1TypeDefinitions()...)
	module.Definitions = append(module.Definitions, createSMIv1OidDefinitions()...)

	return module
}

func createRFC1212() *Module {
	module := NewModule("RFC-1212", types.Synthetic)
	module.Language = types.LanguageSMIv1
	// No definitions - MACRO only
	return module
}

func createRFC1215() *Module {
	module := NewModule("RFC-1215", types.Synthetic)
	module.Language = types.LanguageSMIv1
	// No definitions - MACRO only
	return module
}

func constrainedIntRange(min, max RangeValue) TypeSyntax {
	return &TypeSyntaxConstrained{
		Base:       &TypeSyntaxTypeRef{Name: "INTEGER"},
		Constraint: &ConstraintRange{Ranges: []Range{{Min: min, Max: max}}},
	}
}

func constrainedOctetSize(ranges []Range) TypeSyntax {
	return &TypeSyntaxConstrained{
		Base:       &TypeSyntaxOctetString{},
		Constraint: &ConstraintSize{Ranges: ranges},
	}
}

func constrainedOctetFixed(size uint64) TypeSyntax {
	return constrainedOctetSize([]Range{
		{Min: &RangeValueUnsigned{Value: size}, Max: nil},
	})
}

func constrainedOctetRange(min, max uint64) TypeSyntax {
	return constrainedOctetSize([]Range{
		NewRangeUnsigned(min, max, types.Synthetic),
	})
}

func constrainedUintRange(max uint64) TypeSyntax {
	return constrainedIntRange(
		&RangeValueUnsigned{Value: 0},
		&RangeValueUnsigned{Value: max},
	)
}

func makeOidValue(name string, components []OidComponent, desc, ref string) Definition {
	return &ValueAssignment{
		DefBase:     DefBase{Name: name, Span: types.Synthetic},
		Oid:         NewOidAssignment(components, types.Synthetic),
		Description: desc,
		Reference:   ref,
	}
}

func basePtr(b types.BaseType) *types.BaseType { return &b }

func makeTypeDef(name string, syntax TypeSyntax, base *types.BaseType, desc, ref string) Definition {
	return &TypeDef{
		DefBase:     DefBase{Name: name, Span: types.Synthetic},
		Syntax:      syntax,
		BaseType:    base,
		Status:      types.StatusCurrent,
		Description: desc,
		Reference:   ref,
	}
}

func makeTC(name, displayHint string, syntax TypeSyntax, status types.Status, desc string) Definition {
	return &TypeDef{
		DefBase:             DefBase{Name: name, Span: types.Synthetic},
		Syntax:              syntax,
		DisplayHint:         displayHint,
		Status:              status,
		Description:         desc,
		Reference:           "RFC 2579",
		IsTextualConvention: true,
	}
}

// coreOidDefinitions returns the OID tree shared by both SMIv2 and SMIv1.
func coreOidDefinitions() []Definition {
	return []Definition{
		// iso OBJECT IDENTIFIER ::= { 1 }
		makeOidValue("iso", []OidComponent{&OidComponentNumber{Value: 1}},
			"ISO assigned OIDs.", ""),
		// org OBJECT IDENTIFIER ::= { iso 3 }
		makeOidValue("org", []OidComponent{
			&OidComponentName{NameValue: "iso"},
			&OidComponentNumber{Value: 3},
		}, "ISO identified organizations.", ""),
		// dod OBJECT IDENTIFIER ::= { org 6 }
		makeOidValue("dod", []OidComponent{
			&OidComponentName{NameValue: "org"},
			&OidComponentNumber{Value: 6},
		}, "US Department of Defense.", ""),
		// internet OBJECT IDENTIFIER ::= { dod 1 }
		makeOidValue("internet", []OidComponent{
			&OidComponentName{NameValue: "dod"},
			&OidComponentNumber{Value: 1},
		}, "The root of the Internet OID subtree.", "RFC 1155"),
		// directory OBJECT IDENTIFIER ::= { internet 1 }
		makeOidValue("directory", []OidComponent{
			&OidComponentName{NameValue: "internet"},
			&OidComponentNumber{Value: 1},
		}, "Reserved for future use by the OSI directory (X.500).", "RFC 1155"),
		// mgmt OBJECT IDENTIFIER ::= { internet 2 }
		makeOidValue("mgmt", []OidComponent{
			&OidComponentName{NameValue: "internet"},
			&OidComponentNumber{Value: 2},
		}, "Management subtree. Contains mib-2 and other IANA-managed branches.", "RFC 1155"),
		// experimental OBJECT IDENTIFIER ::= { internet 3 }
		makeOidValue("experimental", []OidComponent{
			&OidComponentName{NameValue: "internet"},
			&OidComponentNumber{Value: 3},
		}, "Experimental subtree for Internet experiments.", "RFC 1155"),
		// private OBJECT IDENTIFIER ::= { internet 4 }
		makeOidValue("private", []OidComponent{
			&OidComponentName{NameValue: "internet"},
			&OidComponentNumber{Value: 4},
		}, "Private subtree. Contains enterprises.", "RFC 1155"),
		// enterprises OBJECT IDENTIFIER ::= { private 1 }
		makeOidValue("enterprises", []OidComponent{
			&OidComponentName{NameValue: "private"},
			&OidComponentNumber{Value: 1},
		}, "Vendor-specific OID subtree. Each vendor gets a unique enterprise number from IANA.", "RFC 1155"),
	}
}

func createOidDefinitions() []Definition {
	const rfc2578 = "RFC 2578"

	defs := []Definition{
		// ccitt OBJECT IDENTIFIER ::= { 0 }
		makeOidValue("ccitt", []OidComponent{&OidComponentNumber{Value: 0}},
			"ITU-T (formerly CCITT) assigned OIDs.", ""),
		// joint-iso-ccitt OBJECT IDENTIFIER ::= { 2 }
		makeOidValue("joint-iso-ccitt", []OidComponent{&OidComponentNumber{Value: 2}},
			"Jointly assigned ISO/ITU-T OIDs.", ""),
	}
	defs = append(defs, coreOidDefinitions()...)
	defs = append(defs,
		// mib-2 OBJECT IDENTIFIER ::= { mgmt 1 }
		makeOidValue("mib-2", []OidComponent{
			&OidComponentName{NameValue: "mgmt"},
			&OidComponentNumber{Value: 1},
		}, "The MIB-II subtree. Contains standard MIB modules (system, interfaces, IP, etc.).", rfc2578),
		// transmission OBJECT IDENTIFIER ::= { mib-2 10 }
		makeOidValue("transmission", []OidComponent{
			&OidComponentName{NameValue: "mib-2"},
			&OidComponentNumber{Value: 10},
		}, "Transmission media specific MIBs.", rfc2578),
		// security OBJECT IDENTIFIER ::= { internet 5 }
		makeOidValue("security", []OidComponent{
			&OidComponentName{NameValue: "internet"},
			&OidComponentNumber{Value: 5},
		}, "Security subtree for authentication and privacy.", rfc2578),
		// snmpV2 OBJECT IDENTIFIER ::= { internet 6 }
		makeOidValue("snmpV2", []OidComponent{
			&OidComponentName{NameValue: "internet"},
			&OidComponentNumber{Value: 6},
		}, "SNMPv2 subtree. Contains transport domains, proxies, and module registrations.", rfc2578),
		// snmpDomains OBJECT IDENTIFIER ::= { snmpV2 1 }
		makeOidValue("snmpDomains", []OidComponent{
			&OidComponentName{NameValue: "snmpV2"},
			&OidComponentNumber{Value: 1},
		}, "Registration point for SNMP transport domains.", rfc2578),
		// snmpProxys OBJECT IDENTIFIER ::= { snmpV2 2 }
		makeOidValue("snmpProxys", []OidComponent{
			&OidComponentName{NameValue: "snmpV2"},
			&OidComponentNumber{Value: 2},
		}, "Registration point for SNMP proxy types.", rfc2578),
		// snmpModules OBJECT IDENTIFIER ::= { snmpV2 3 }
		makeOidValue("snmpModules", []OidComponent{
			&OidComponentName{NameValue: "snmpV2"},
			&OidComponentNumber{Value: 3},
		}, "Registration point for SNMP MIB module identities.", rfc2578),
		// zeroDotZero OBJECT IDENTIFIER ::= { 0 0 }
		makeOidValue("zeroDotZero", []OidComponent{
			&OidComponentNumber{Value: 0},
			&OidComponentNumber{Value: 0},
		}, "A value used for null identifiers. Used as a default value when no valid OID is applicable.", rfc2578),
	)
	return defs
}

func createBaseTypeDefinitions() []Definition {
	int32Min := int64(math.MinInt32)
	int32Max := int64(math.MaxInt32)
	uint32Max := uint64(math.MaxUint32)
	uint64Max := uint64(math.MaxUint64)

	const rfc2578 = "RFC 2578, Section 7.1"

	return []Definition{
		// Integer32 ::= INTEGER (-2147483648..2147483647)
		makeTypeDef("Integer32",
			constrainedIntRange(
				&RangeValueSigned{Value: int32Min},
				&RangeValueSigned{Value: int32Max},
			),
			basePtr(types.BaseInteger32),
			"A 32-bit signed integer. The range is -2147483648 to 2147483647.",
			rfc2578+".1",
		),
		// Counter32 ::= [APPLICATION 1] IMPLICIT INTEGER (0..4294967295)
		makeTypeDef("Counter32", constrainedUintRange(uint32Max), basePtr(types.BaseCounter32),
			"A non-negative 32-bit counter that monotonically increases until it wraps at 2^32-1. A Counter32 has no defined initial value and must not be used for a MIB object that has a MAX-ACCESS of read-write or read-create.",
			rfc2578+".6",
		),
		// Counter64 ::= [APPLICATION 6] IMPLICIT INTEGER (0..18446744073709551615)
		makeTypeDef("Counter64", constrainedUintRange(uint64Max), basePtr(types.BaseCounter64),
			"A non-negative 64-bit counter for high-speed interfaces where Counter32 would wrap too frequently. Counter64 should only be used when Counter32 wrapping is a problem.",
			rfc2578+".8",
		),
		// Gauge32 ::= [APPLICATION 2] IMPLICIT INTEGER (0..4294967295)
		makeTypeDef("Gauge32", constrainedUintRange(uint32Max), basePtr(types.BaseGauge32),
			"A non-negative 32-bit integer that may increase or decrease but latches at a maximum value of 2^32-1.",
			rfc2578+".7",
		),
		// Unsigned32 ::= [APPLICATION 2] IMPLICIT INTEGER (0..4294967295)
		makeTypeDef("Unsigned32", constrainedUintRange(uint32Max), basePtr(types.BaseUnsigned32),
			"A 32-bit unsigned integer with range 0 to 4294967295. Shares the same APPLICATION tag as Gauge32.",
			rfc2578+".1",
		),
		// TimeTicks ::= [APPLICATION 3] IMPLICIT INTEGER (0..4294967295)
		makeTypeDef("TimeTicks", constrainedUintRange(uint32Max), basePtr(types.BaseTimeTicks),
			"A non-negative 32-bit integer representing time in hundredths of a second since some reference epoch.",
			rfc2578+".2",
		),
		// IpAddress ::= [APPLICATION 0] IMPLICIT OCTET STRING (SIZE (4))
		makeTypeDef("IpAddress", constrainedOctetFixed(4), basePtr(types.BaseIpAddress),
			"An IPv4 address represented as a 4-byte OCTET STRING in network byte order. New MIBs should use InetAddress from INET-ADDRESS-MIB instead.",
			rfc2578+".3",
		),
		// Opaque ::= [APPLICATION 4] IMPLICIT OCTET STRING
		makeTypeDef("Opaque", &TypeSyntaxOctetString{}, basePtr(types.BaseOpaque),
			"An arbitrary ASN.1 value encoded as an OCTET STRING for transparent transport. Use of Opaque is discouraged in new MIB definitions.",
			rfc2578+".4",
		),
		// ObjectName ::= OBJECT IDENTIFIER
		makeTypeDef("ObjectName", &TypeSyntaxObjectIdentifier{}, nil,
			"An OBJECT IDENTIFIER value that names a managed object.",
			rfc2578,
		),
		// NotificationName ::= OBJECT IDENTIFIER
		makeTypeDef("NotificationName", &TypeSyntaxObjectIdentifier{}, nil,
			"An OBJECT IDENTIFIER value that names a notification.",
			rfc2578,
		),
		// ExtUTCTime ::= OCTET STRING (SIZE (11 | 13))
		makeTypeDef("ExtUTCTime",
			constrainedOctetSize([]Range{
				{Min: &RangeValueUnsigned{Value: 11}, Max: nil},
				{Min: &RangeValueUnsigned{Value: 13}, Max: nil},
			}),
			nil,
			"",
			rfc2578,
		),
		// ObjectSyntax, SimpleSyntax, ApplicationSyntax - protocol meta-types
		makeTypeDef("ObjectSyntax", &TypeSyntaxTypeRef{Name: "SimpleSyntax"}, nil,
			"The union of all SMIv2 data types that may be used in OBJECT-TYPE definitions.",
			rfc2578,
		),
		makeTypeDef("SimpleSyntax", &TypeSyntaxTypeRef{Name: "INTEGER"}, nil,
			"The union of primitive ASN.1 types: INTEGER, OCTET STRING, and OBJECT IDENTIFIER.",
			rfc2578,
		),
		makeTypeDef("ApplicationSyntax", &TypeSyntaxTypeRef{Name: "IpAddress"}, nil,
			"The union of application-wide types: IpAddress, Counter32, Gauge32, Unsigned32, TimeTicks, Opaque, and Counter64.",
			rfc2578,
		),
	}
}

func createSMIv1TypeDefinitions() []Definition {
	uint32Max := uint64(math.MaxUint32)

	const rfc1155 = "RFC 1155, Section 3.2.3"

	return []Definition{
		// Counter ::= [APPLICATION 1] IMPLICIT INTEGER (0..4294967295)
		makeTypeDef("Counter", constrainedUintRange(uint32Max), basePtr(types.BaseCounter32),
			"A non-negative 32-bit counter that monotonically increases until it wraps at 2^32-1. The SMIv1 equivalent of Counter32.",
			rfc1155+".3",
		),
		// Gauge ::= [APPLICATION 2] IMPLICIT INTEGER (0..4294967295)
		makeTypeDef("Gauge", constrainedUintRange(uint32Max), basePtr(types.BaseGauge32),
			"A non-negative 32-bit integer that may increase or decrease but latches at a maximum value. The SMIv1 equivalent of Gauge32.",
			rfc1155+".4",
		),
		// IpAddress ::= [APPLICATION 0] IMPLICIT OCTET STRING (SIZE (4))
		makeTypeDef("IpAddress", constrainedOctetFixed(4), basePtr(types.BaseIpAddress),
			"An IPv4 address represented as a 4-byte OCTET STRING in network byte order.",
			rfc1155+".1",
		),
		// NetworkAddress ::= CHOICE { internet IpAddress }
		makeTypeDef("NetworkAddress", &TypeSyntaxTypeRef{Name: "IpAddress"}, basePtr(types.BaseIpAddress),
			"A network address from one of possibly several protocol families. In practice, only the internet family (IpAddress) is used.",
			rfc1155+".1",
		),
		// TimeTicks ::= [APPLICATION 3] IMPLICIT INTEGER (0..4294967295)
		makeTypeDef("TimeTicks", constrainedUintRange(uint32Max), basePtr(types.BaseTimeTicks),
			"A non-negative 32-bit integer representing time in hundredths of a second since some reference epoch.",
			rfc1155+".5",
		),
		// Opaque ::= [APPLICATION 4] IMPLICIT OCTET STRING
		makeTypeDef("Opaque", &TypeSyntaxOctetString{}, basePtr(types.BaseOpaque),
			"An arbitrary ASN.1 value encoded as an OCTET STRING for transparent transport.",
			rfc1155+".6",
		),
		// ObjectName ::= OBJECT IDENTIFIER
		makeTypeDef("ObjectName", &TypeSyntaxObjectIdentifier{}, nil,
			"An OBJECT IDENTIFIER value that names a managed object.",
			"RFC 1155, Section 3.2",
		),
	}
}

func createSMIv1OidDefinitions() []Definition {
	return coreOidDefinitions()
}

func createTCDefinitions() []Definition {
	int32Max := int64(math.MaxInt32)

	return []Definition{
		// DisplayString ::= TEXTUAL-CONVENTION DISPLAY-HINT "255a" SYNTAX OCTET STRING (SIZE (0..255))
		makeTC("DisplayString", "255a", constrainedOctetRange(0, 255), types.StatusCurrent,
			"An NVT ASCII string of 0 to 255 characters. Preferred over the deprecated RFC 1213 DisplayString.",
		),
		// PhysAddress ::= TEXTUAL-CONVENTION DISPLAY-HINT "1x:" SYNTAX OCTET STRING
		makeTC("PhysAddress", "1x:", &TypeSyntaxOctetString{}, types.StatusCurrent,
			"A media- or physical-level address, represented as an OCTET STRING.",
		),
		// MacAddress ::= TEXTUAL-CONVENTION DISPLAY-HINT "1x:" SYNTAX OCTET STRING (SIZE (6))
		makeTC("MacAddress", "1x:", constrainedOctetFixed(6), types.StatusCurrent,
			"An IEEE 802 MAC address represented as a 6-byte OCTET STRING in canonical order.",
		),
		// TruthValue ::= TEXTUAL-CONVENTION SYNTAX INTEGER { true(1), false(2) }
		makeTC("TruthValue", "", &TypeSyntaxIntegerEnum{NamedNumbers: []NamedNumber{
			{Name: "true", Value: 1},
			{Name: "false", Value: 2},
		}}, types.StatusCurrent,
			"A boolean value: true(1) or false(2).",
		),
		// RowStatus ::= TEXTUAL-CONVENTION SYNTAX INTEGER { active(1), ... }
		makeTC("RowStatus", "", &TypeSyntaxIntegerEnum{NamedNumbers: []NamedNumber{
			{Name: "active", Value: 1},
			{Name: "notInService", Value: 2},
			{Name: "notReady", Value: 3},
			{Name: "createAndGo", Value: 4},
			{Name: "createAndWait", Value: 5},
			{Name: "destroy", Value: 6},
		}}, types.StatusCurrent,
			"Controls creation, deletion, and activation of conceptual rows. Values: active(1), notInService(2), notReady(3), createAndGo(4), createAndWait(5), destroy(6).",
		),
		// StorageType ::= TEXTUAL-CONVENTION SYNTAX INTEGER { other(1), ... }
		makeTC("StorageType", "", &TypeSyntaxIntegerEnum{NamedNumbers: []NamedNumber{
			{Name: "other", Value: 1},
			{Name: "volatile", Value: 2},
			{Name: "nonVolatile", Value: 3},
			{Name: "permanent", Value: 4},
			{Name: "readOnly", Value: 5},
		}}, types.StatusCurrent,
			"Describes the storage persistence of a conceptual row. Values: other(1), volatile(2), nonVolatile(3), permanent(4), readOnly(5).",
		),
		// TimeStamp ::= TEXTUAL-CONVENTION SYNTAX TimeTicks
		makeTC("TimeStamp", "", &TypeSyntaxTypeRef{Name: "TimeTicks"}, types.StatusCurrent,
			"The value of sysUpTime at which a specific occurrence happened. Used to facilitate delta calculations.",
		),
		// TimeInterval ::= TEXTUAL-CONVENTION SYNTAX INTEGER (0..2147483647)
		makeTC("TimeInterval", "",
			constrainedIntRange(
				&RangeValueSigned{Value: 0},
				&RangeValueSigned{Value: int32Max},
			), types.StatusCurrent,
			"A period of time measured in hundredths of a second, ranging from 0 to 2147483647.",
		),
		// DateAndTime ::= TEXTUAL-CONVENTION DISPLAY-HINT "2d-1d-1d,1d:1d:1d.1d,1a1d:1d" SYNTAX OCTET STRING (SIZE (8 | 11))
		makeTC("DateAndTime", "2d-1d-1d,1d:1d:1d.1d,1a1d:1d",
			constrainedOctetSize([]Range{
				{Min: &RangeValueUnsigned{Value: 8}, Max: nil},
				{Min: &RangeValueUnsigned{Value: 11}, Max: nil},
			}), types.StatusCurrent,
			"A date-time specification. 8 bytes for local time, 11 bytes when UTC offset is included.",
		),
		// TestAndIncr ::= TEXTUAL-CONVENTION SYNTAX INTEGER (0..2147483647)
		makeTC("TestAndIncr", "",
			constrainedIntRange(
				&RangeValueSigned{Value: 0},
				&RangeValueSigned{Value: int32Max},
			), types.StatusCurrent,
			"A test-and-increment spinlock. Used for cooperating command generator applications to coordinate SET operations.",
		),
		// AutonomousType ::= TEXTUAL-CONVENTION SYNTAX OBJECT IDENTIFIER
		makeTC("AutonomousType", "", &TypeSyntaxObjectIdentifier{}, types.StatusCurrent,
			"An OID that identifies a type or protocol independently of the place where it is defined.",
		),
		// InstancePointer ::= TEXTUAL-CONVENTION (obsolete) SYNTAX OBJECT IDENTIFIER
		makeTC("InstancePointer", "", &TypeSyntaxObjectIdentifier{}, types.StatusObsolete,
			"Obsolete. A pointer to a specific row in a conceptual table. Use VariablePointer instead.",
		),
		// VariablePointer ::= TEXTUAL-CONVENTION SYNTAX OBJECT IDENTIFIER
		makeTC("VariablePointer", "", &TypeSyntaxObjectIdentifier{}, types.StatusCurrent,
			"A pointer to a specific object instance, including its index. The OID must have an instance suffix.",
		),
		// RowPointer ::= TEXTUAL-CONVENTION SYNTAX OBJECT IDENTIFIER
		makeTC("RowPointer", "", &TypeSyntaxObjectIdentifier{}, types.StatusCurrent,
			"A pointer to a conceptual row. The OID is the first accessible columnar object followed by the row's index values.",
		),
		// TDomain ::= TEXTUAL-CONVENTION SYNTAX OBJECT IDENTIFIER
		makeTC("TDomain", "", &TypeSyntaxObjectIdentifier{}, types.StatusCurrent,
			"An OID identifying a transport domain (e.g., snmpUDPDomain for SNMP over UDP/IPv4).",
		),
		// TAddress ::= TEXTUAL-CONVENTION SYNTAX OCTET STRING (SIZE (1..255))
		makeTC("TAddress", "", constrainedOctetRange(1, 255), types.StatusCurrent,
			"A transport address, whose format is defined by the associated TDomain value.",
		),
	}
}
