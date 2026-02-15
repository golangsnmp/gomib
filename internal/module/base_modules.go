package module

import (
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

var (
	baseModuleMu    sync.RWMutex
	baseModuleCache = make(map[string]*Module)
)

// GetBaseModule returns the Module for the named base module, or nil.
// Modules are created on first access and cached.
func GetBaseModule(name string) *Module {
	if !IsBaseModule(name) {
		return nil
	}

	// Fast path: read lock
	baseModuleMu.RLock()
	if mod, ok := baseModuleCache[name]; ok {
		baseModuleMu.RUnlock()
		return mod
	}
	baseModuleMu.RUnlock()

	// Slow path: write lock, create and cache all base modules
	baseModuleMu.Lock()
	defer baseModuleMu.Unlock()

	if mod, ok := baseModuleCache[name]; ok {
		return mod
	}

	for _, mod := range CreateBaseModules() {
		baseModuleCache[mod.Name] = mod
	}

	return baseModuleCache[name]
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
	return baseModuleNames[:]
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
		NewRangeUnsigned(min, max),
	})
}

func constrainedUintRange(max uint64) TypeSyntax {
	return constrainedIntRange(
		&RangeValueUnsigned{Value: 0},
		&RangeValueUnsigned{Value: max},
	)
}

func makeOidValue(name string, components []OidComponent) Definition {
	return &ValueAssignment{
		Name: name,
		Oid:  NewOidAssignment(components, types.Synthetic),
		Span: types.Synthetic,
	}
}

func basePtr(b types.BaseType) *types.BaseType { return &b }

func makeTypeDef(name string, syntax TypeSyntax, base *types.BaseType, status types.Status) Definition {
	return &TypeDef{
		Name:                name,
		Syntax:              syntax,
		BaseType:            base,
		DisplayHint:         "",
		Status:              status,
		Description:         "",
		Reference:           "",
		IsTextualConvention: false,
		Span:                types.Synthetic,
	}
}

func makeTC(name, displayHint string, syntax TypeSyntax, status types.Status) Definition {
	return &TypeDef{
		Name:                name,
		Syntax:              syntax,
		BaseType:            nil,
		DisplayHint:         displayHint,
		Status:              status,
		Description:         "",
		Reference:           "",
		IsTextualConvention: true,
		Span:                types.Synthetic,
	}
}

func createOidDefinitions() []Definition {
	return []Definition{
		// ccitt OBJECT IDENTIFIER ::= { 0 }
		makeOidValue("ccitt", []OidComponent{&OidComponentNumber{Value: 0}}),
		// iso OBJECT IDENTIFIER ::= { 1 }
		makeOidValue("iso", []OidComponent{&OidComponentNumber{Value: 1}}),
		// joint-iso-ccitt OBJECT IDENTIFIER ::= { 2 }
		makeOidValue("joint-iso-ccitt", []OidComponent{&OidComponentNumber{Value: 2}}),
		// org OBJECT IDENTIFIER ::= { iso 3 }
		makeOidValue("org", []OidComponent{
			&OidComponentName{NameValue: "iso"},
			&OidComponentNumber{Value: 3},
		}),
		// dod OBJECT IDENTIFIER ::= { org 6 }
		makeOidValue("dod", []OidComponent{
			&OidComponentName{NameValue: "org"},
			&OidComponentNumber{Value: 6},
		}),
		// internet OBJECT IDENTIFIER ::= { dod 1 }
		makeOidValue("internet", []OidComponent{
			&OidComponentName{NameValue: "dod"},
			&OidComponentNumber{Value: 1},
		}),
		// directory OBJECT IDENTIFIER ::= { internet 1 }
		makeOidValue("directory", []OidComponent{
			&OidComponentName{NameValue: "internet"},
			&OidComponentNumber{Value: 1},
		}),
		// mgmt OBJECT IDENTIFIER ::= { internet 2 }
		makeOidValue("mgmt", []OidComponent{
			&OidComponentName{NameValue: "internet"},
			&OidComponentNumber{Value: 2},
		}),
		// mib-2 OBJECT IDENTIFIER ::= { mgmt 1 }
		makeOidValue("mib-2", []OidComponent{
			&OidComponentName{NameValue: "mgmt"},
			&OidComponentNumber{Value: 1},
		}),
		// transmission OBJECT IDENTIFIER ::= { mib-2 10 }
		makeOidValue("transmission", []OidComponent{
			&OidComponentName{NameValue: "mib-2"},
			&OidComponentNumber{Value: 10},
		}),
		// experimental OBJECT IDENTIFIER ::= { internet 3 }
		makeOidValue("experimental", []OidComponent{
			&OidComponentName{NameValue: "internet"},
			&OidComponentNumber{Value: 3},
		}),
		// private OBJECT IDENTIFIER ::= { internet 4 }
		makeOidValue("private", []OidComponent{
			&OidComponentName{NameValue: "internet"},
			&OidComponentNumber{Value: 4},
		}),
		// enterprises OBJECT IDENTIFIER ::= { private 1 }
		makeOidValue("enterprises", []OidComponent{
			&OidComponentName{NameValue: "private"},
			&OidComponentNumber{Value: 1},
		}),
		// security OBJECT IDENTIFIER ::= { internet 5 }
		makeOidValue("security", []OidComponent{
			&OidComponentName{NameValue: "internet"},
			&OidComponentNumber{Value: 5},
		}),
		// snmpV2 OBJECT IDENTIFIER ::= { internet 6 }
		makeOidValue("snmpV2", []OidComponent{
			&OidComponentName{NameValue: "internet"},
			&OidComponentNumber{Value: 6},
		}),
		// snmpDomains OBJECT IDENTIFIER ::= { snmpV2 1 }
		makeOidValue("snmpDomains", []OidComponent{
			&OidComponentName{NameValue: "snmpV2"},
			&OidComponentNumber{Value: 1},
		}),
		// snmpProxys OBJECT IDENTIFIER ::= { snmpV2 2 }
		makeOidValue("snmpProxys", []OidComponent{
			&OidComponentName{NameValue: "snmpV2"},
			&OidComponentNumber{Value: 2},
		}),
		// snmpModules OBJECT IDENTIFIER ::= { snmpV2 3 }
		makeOidValue("snmpModules", []OidComponent{
			&OidComponentName{NameValue: "snmpV2"},
			&OidComponentNumber{Value: 3},
		}),
		// zeroDotZero OBJECT IDENTIFIER ::= { 0 0 }
		makeOidValue("zeroDotZero", []OidComponent{
			&OidComponentNumber{Value: 0},
			&OidComponentNumber{Value: 0},
		}),
		// snmp OBJECT IDENTIFIER ::= { mib-2 11 }
		makeOidValue("snmp", []OidComponent{
			&OidComponentName{NameValue: "mib-2"},
			&OidComponentNumber{Value: 11},
		}),
	}
}

func createBaseTypeDefinitions() []Definition {
	int32Min := int64(-2147483648)
	int32Max := int64(2147483647)
	uint32Max := uint64(4294967295)
	uint64Max := uint64(18446744073709551615)

	return []Definition{
		// Integer32 ::= INTEGER (-2147483648..2147483647)
		makeTypeDef("Integer32",
			constrainedIntRange(
				&RangeValueSigned{Value: int32Min},
				&RangeValueSigned{Value: int32Max},
			),
			basePtr(types.BaseInteger32), types.StatusCurrent,
		),
		// Counter32 ::= [APPLICATION 1] IMPLICIT INTEGER (0..4294967295)
		makeTypeDef("Counter32", constrainedUintRange(uint32Max), basePtr(types.BaseCounter32), types.StatusCurrent),
		// Counter64 ::= [APPLICATION 6] IMPLICIT INTEGER (0..18446744073709551615)
		makeTypeDef("Counter64", constrainedUintRange(uint64Max), basePtr(types.BaseCounter64), types.StatusCurrent),
		// Gauge32 ::= [APPLICATION 2] IMPLICIT INTEGER (0..4294967295)
		makeTypeDef("Gauge32", constrainedUintRange(uint32Max), basePtr(types.BaseGauge32), types.StatusCurrent),
		// Unsigned32 ::= [APPLICATION 2] IMPLICIT INTEGER (0..4294967295)
		makeTypeDef("Unsigned32", constrainedUintRange(uint32Max), basePtr(types.BaseUnsigned32), types.StatusCurrent),
		// TimeTicks ::= [APPLICATION 3] IMPLICIT INTEGER (0..4294967295)
		makeTypeDef("TimeTicks", constrainedUintRange(uint32Max), basePtr(types.BaseTimeTicks), types.StatusCurrent),
		// IpAddress ::= [APPLICATION 0] IMPLICIT OCTET STRING (SIZE (4))
		makeTypeDef("IpAddress", constrainedOctetFixed(4), basePtr(types.BaseIpAddress), types.StatusCurrent),
		// Opaque ::= [APPLICATION 4] IMPLICIT OCTET STRING
		makeTypeDef("Opaque", &TypeSyntaxOctetString{}, basePtr(types.BaseOpaque), types.StatusCurrent),
		// ObjectName ::= OBJECT IDENTIFIER
		makeTypeDef("ObjectName", &TypeSyntaxObjectIdentifier{}, nil, types.StatusCurrent),
		// NotificationName ::= OBJECT IDENTIFIER
		makeTypeDef("NotificationName", &TypeSyntaxObjectIdentifier{}, nil, types.StatusCurrent),
		// ExtUTCTime ::= OCTET STRING (SIZE (11 | 13)) - obsolete
		makeTypeDef("ExtUTCTime",
			constrainedOctetSize([]Range{
				{Min: &RangeValueUnsigned{Value: 11}, Max: nil},
				{Min: &RangeValueUnsigned{Value: 13}, Max: nil},
			}),
			nil, types.StatusObsolete,
		),
		// ObjectSyntax, SimpleSyntax, ApplicationSyntax - protocol meta-types
		makeTypeDef("ObjectSyntax", &TypeSyntaxTypeRef{Name: "SimpleSyntax"}, nil, types.StatusCurrent),
		makeTypeDef("SimpleSyntax", &TypeSyntaxTypeRef{Name: "INTEGER"}, nil, types.StatusCurrent),
		makeTypeDef("ApplicationSyntax", &TypeSyntaxTypeRef{Name: "IpAddress"}, nil, types.StatusCurrent),
	}
}

func createSMIv1TypeDefinitions() []Definition {
	uint32Max := uint64(4294967295)

	return []Definition{
		// Counter ::= [APPLICATION 1] IMPLICIT INTEGER (0..4294967295)
		makeTypeDef("Counter", constrainedUintRange(uint32Max), basePtr(types.BaseCounter32), types.StatusCurrent),
		// Gauge ::= [APPLICATION 2] IMPLICIT INTEGER (0..4294967295)
		makeTypeDef("Gauge", constrainedUintRange(uint32Max), basePtr(types.BaseGauge32), types.StatusCurrent),
		// IpAddress ::= [APPLICATION 0] IMPLICIT OCTET STRING (SIZE (4))
		makeTypeDef("IpAddress", constrainedOctetFixed(4), basePtr(types.BaseIpAddress), types.StatusCurrent),
		// NetworkAddress ::= CHOICE { internet IpAddress }
		makeTypeDef("NetworkAddress", &TypeSyntaxTypeRef{Name: "IpAddress"}, basePtr(types.BaseIpAddress), types.StatusCurrent),
		// TimeTicks ::= [APPLICATION 3] IMPLICIT INTEGER (0..4294967295)
		makeTypeDef("TimeTicks", constrainedUintRange(uint32Max), basePtr(types.BaseTimeTicks), types.StatusCurrent),
		// Opaque ::= [APPLICATION 4] IMPLICIT OCTET STRING
		makeTypeDef("Opaque", &TypeSyntaxOctetString{}, basePtr(types.BaseOpaque), types.StatusCurrent),
		// ObjectName ::= OBJECT IDENTIFIER
		makeTypeDef("ObjectName", &TypeSyntaxObjectIdentifier{}, nil, types.StatusCurrent),
	}
}

func createSMIv1OidDefinitions() []Definition {
	return []Definition{
		// iso OBJECT IDENTIFIER ::= { 1 }
		makeOidValue("iso", []OidComponent{&OidComponentNumber{Value: 1}}),
		// org OBJECT IDENTIFIER ::= { iso 3 }
		makeOidValue("org", []OidComponent{
			&OidComponentName{NameValue: "iso"},
			&OidComponentNumber{Value: 3},
		}),
		// dod OBJECT IDENTIFIER ::= { org 6 }
		makeOidValue("dod", []OidComponent{
			&OidComponentName{NameValue: "org"},
			&OidComponentNumber{Value: 6},
		}),
		// internet OBJECT IDENTIFIER ::= { dod 1 }
		makeOidValue("internet", []OidComponent{
			&OidComponentName{NameValue: "dod"},
			&OidComponentNumber{Value: 1},
		}),
		// directory OBJECT IDENTIFIER ::= { internet 1 }
		makeOidValue("directory", []OidComponent{
			&OidComponentName{NameValue: "internet"},
			&OidComponentNumber{Value: 1},
		}),
		// mgmt OBJECT IDENTIFIER ::= { internet 2 }
		makeOidValue("mgmt", []OidComponent{
			&OidComponentName{NameValue: "internet"},
			&OidComponentNumber{Value: 2},
		}),
		// experimental OBJECT IDENTIFIER ::= { internet 3 }
		makeOidValue("experimental", []OidComponent{
			&OidComponentName{NameValue: "internet"},
			&OidComponentNumber{Value: 3},
		}),
		// private OBJECT IDENTIFIER ::= { internet 4 }
		makeOidValue("private", []OidComponent{
			&OidComponentName{NameValue: "internet"},
			&OidComponentNumber{Value: 4},
		}),
		// enterprises OBJECT IDENTIFIER ::= { private 1 }
		makeOidValue("enterprises", []OidComponent{
			&OidComponentName{NameValue: "private"},
			&OidComponentNumber{Value: 1},
		}),
	}
}

func createTCDefinitions() []Definition {
	int32Max := int64(2147483647)

	return []Definition{
		// DisplayString ::= TEXTUAL-CONVENTION DISPLAY-HINT "255a" SYNTAX OCTET STRING (SIZE (0..255))
		makeTC("DisplayString", "255a", constrainedOctetRange(0, 255), types.StatusCurrent),
		// PhysAddress ::= TEXTUAL-CONVENTION DISPLAY-HINT "1x:" SYNTAX OCTET STRING
		makeTC("PhysAddress", "1x:", &TypeSyntaxOctetString{}, types.StatusCurrent),
		// MacAddress ::= TEXTUAL-CONVENTION DISPLAY-HINT "1x:" SYNTAX OCTET STRING (SIZE (6))
		makeTC("MacAddress", "1x:", constrainedOctetFixed(6), types.StatusCurrent),
		// TruthValue ::= TEXTUAL-CONVENTION SYNTAX INTEGER { true(1), false(2) }
		makeTC("TruthValue", "", &TypeSyntaxIntegerEnum{NamedNumbers: []NamedNumber{
			{Name: "true", Value: 1},
			{Name: "false", Value: 2},
		}}, types.StatusCurrent),
		// RowStatus ::= TEXTUAL-CONVENTION SYNTAX INTEGER { active(1), ... }
		makeTC("RowStatus", "", &TypeSyntaxIntegerEnum{NamedNumbers: []NamedNumber{
			{Name: "active", Value: 1},
			{Name: "notInService", Value: 2},
			{Name: "notReady", Value: 3},
			{Name: "createAndGo", Value: 4},
			{Name: "createAndWait", Value: 5},
			{Name: "destroy", Value: 6},
		}}, types.StatusCurrent),
		// StorageType ::= TEXTUAL-CONVENTION SYNTAX INTEGER { other(1), ... }
		makeTC("StorageType", "", &TypeSyntaxIntegerEnum{NamedNumbers: []NamedNumber{
			{Name: "other", Value: 1},
			{Name: "volatile", Value: 2},
			{Name: "nonVolatile", Value: 3},
			{Name: "permanent", Value: 4},
			{Name: "readOnly", Value: 5},
		}}, types.StatusCurrent),
		// TimeStamp ::= TEXTUAL-CONVENTION SYNTAX TimeTicks
		makeTC("TimeStamp", "", &TypeSyntaxTypeRef{Name: "TimeTicks"}, types.StatusCurrent),
		// TimeInterval ::= TEXTUAL-CONVENTION SYNTAX INTEGER (0..2147483647)
		makeTC("TimeInterval", "",
			constrainedIntRange(
				&RangeValueUnsigned{Value: 0},
				&RangeValueSigned{Value: int32Max},
			), types.StatusCurrent,
		),
		// DateAndTime ::= TEXTUAL-CONVENTION DISPLAY-HINT "2d-1d-1d,1d:1d:1d.1d,1a1d:1d" SYNTAX OCTET STRING (SIZE (8 | 11))
		makeTC("DateAndTime", "2d-1d-1d,1d:1d:1d.1d,1a1d:1d",
			constrainedOctetSize([]Range{
				{Min: &RangeValueUnsigned{Value: 8}, Max: nil},
				{Min: &RangeValueUnsigned{Value: 11}, Max: nil},
			}), types.StatusCurrent,
		),
		// TestAndIncr ::= TEXTUAL-CONVENTION SYNTAX INTEGER (0..2147483647)
		makeTC("TestAndIncr", "",
			constrainedIntRange(
				&RangeValueUnsigned{Value: 0},
				&RangeValueSigned{Value: int32Max},
			), types.StatusCurrent,
		),
		// AutonomousType ::= TEXTUAL-CONVENTION SYNTAX OBJECT IDENTIFIER
		makeTC("AutonomousType", "", &TypeSyntaxObjectIdentifier{}, types.StatusCurrent),
		// InstancePointer ::= TEXTUAL-CONVENTION (obsolete) SYNTAX OBJECT IDENTIFIER
		makeTC("InstancePointer", "", &TypeSyntaxObjectIdentifier{}, types.StatusObsolete),
		// VariablePointer ::= TEXTUAL-CONVENTION SYNTAX OBJECT IDENTIFIER
		makeTC("VariablePointer", "", &TypeSyntaxObjectIdentifier{}, types.StatusCurrent),
		// RowPointer ::= TEXTUAL-CONVENTION SYNTAX OBJECT IDENTIFIER
		makeTC("RowPointer", "", &TypeSyntaxObjectIdentifier{}, types.StatusCurrent),
		// TDomain ::= TEXTUAL-CONVENTION SYNTAX OBJECT IDENTIFIER
		makeTC("TDomain", "", &TypeSyntaxObjectIdentifier{}, types.StatusCurrent),
		// TAddress ::= TEXTUAL-CONVENTION SYNTAX OCTET STRING (SIZE (1..255))
		makeTC("TAddress", "", constrainedOctetRange(1, 255), types.StatusCurrent),
	}
}
