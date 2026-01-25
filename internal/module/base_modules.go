package module

import (
	"sync"

	"github.com/golangsnmp/gomib/internal/types"
)

// BaseModule represents SMI base modules (types and MACROs, not regular MIBs).
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

// baseModuleNames is the single source of truth for base module names.
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

// IsSMIv2 returns true if this is an SMIv2 module.
func (m BaseModule) IsSMIv2() bool {
	switch m {
	case BaseModuleSNMPv2SMI, BaseModuleSNMPv2TC, BaseModuleSNMPv2CONF:
		return true
	default:
		return false
	}
}

// IsSMIv1 returns true if this is an SMIv1 module.
func (m BaseModule) IsSMIv1() bool {
	switch m {
	case BaseModuleRFC1155SMI, BaseModuleRFC1065SMI, BaseModuleRFC1212, BaseModuleRFC1215:
		return true
	default:
		return false
	}
}

// baseModuleByName maps module names to BaseModule values.
var baseModuleByName = func() map[string]BaseModule {
	m := make(map[string]BaseModule, len(baseModuleNames))
	for i, name := range baseModuleNames {
		m[name] = BaseModule(i)
	}
	return m
}()

// BaseModuleFromName looks up a base module by name.
func BaseModuleFromName(name string) (BaseModule, bool) {
	m, ok := baseModuleByName[name]
	return m, ok
}

// IsBaseModule returns true if the module name is a recognized base module.
func IsBaseModule(name string) bool {
	_, ok := BaseModuleFromName(name)
	return ok
}

// baseModuleCache holds lazily-created base modules.
var (
	baseModuleMu    sync.RWMutex
	baseModuleCache = make(map[string]*Module)
)

// GetBaseModule returns the base module with the given name, or nil if not a base module.
// Base modules are created on first access and cached.
func GetBaseModule(name string) *Module {
	if !IsBaseModule(name) {
		return nil
	}

	// Fast path: check if already cached
	baseModuleMu.RLock()
	if mod, ok := baseModuleCache[name]; ok {
		baseModuleMu.RUnlock()
		return mod
	}
	baseModuleMu.RUnlock()

	// Slow path: create all base modules and cache them
	baseModuleMu.Lock()
	defer baseModuleMu.Unlock()

	// Double-check after acquiring write lock
	if mod, ok := baseModuleCache[name]; ok {
		return mod
	}

	for _, mod := range CreateBaseModules() {
		baseModuleCache[mod.Name] = mod
	}

	return baseModuleCache[name]
}

// AllBaseModules returns all base modules.
func AllBaseModules() []BaseModule {
	result := make([]BaseModule, len(baseModuleNames))
	for i := range baseModuleNames {
		result[i] = BaseModule(i)
	}
	return result
}

// BaseModuleNames returns the names of all base modules.
func BaseModuleNames() []string {
	return baseModuleNames[:]
}

// CreateBaseModules creates synthetic modules for all base modules.
//
// Returns modules in order: SNMPv2-SMI, SNMPv2-TC, SNMPv2-CONF,
// RFC1155-SMI, RFC1065-SMI, RFC-1212, RFC-1215.
// These should be prepended to the user module list before resolution.
func CreateBaseModules() []*Module {
	return []*Module{
		createSNMPv2SMI(),
		createSNMPv2TC(),
		createSNMPv2CONF(),
		createRFC1155SMI(),
		createRFC1065SMI(),
		createRFC1212(),
		createRFC1215(),
	}
}

// createSNMPv2SMI creates the synthetic SNMPv2-SMI module.
func createSNMPv2SMI() *Module {
	module := NewModule("SNMPv2-SMI", types.Synthetic)
	module.Language = SmiLanguageSMIv2

	// Add OID root definitions
	module.Definitions = append(module.Definitions, createOidDefinitions()...)

	// Add base type definitions
	module.Definitions = append(module.Definitions, createBaseTypeDefinitions()...)

	return module
}

// createSNMPv2TC creates the synthetic SNMPv2-TC module.
func createSNMPv2TC() *Module {
	module := NewModule("SNMPv2-TC", types.Synthetic)
	module.Language = SmiLanguageSMIv2

	// Add imports from SNMPv2-SMI (for base types used by TCs)
	module.Imports = []Import{
		NewImport("SNMPv2-SMI", "TimeTicks", types.Synthetic),
	}

	// Add textual convention definitions
	module.Definitions = append(module.Definitions, createTCDefinitions()...)

	return module
}

// createSNMPv2CONF creates the synthetic SNMPv2-CONF module.
func createSNMPv2CONF() *Module {
	module := NewModule("SNMPv2-CONF", types.Synthetic)
	module.Language = SmiLanguageSMIv2
	// No definitions - MACROs only
	return module
}

// createRFC1155SMI creates the synthetic RFC1155-SMI module.
func createRFC1155SMI() *Module {
	module := NewModule("RFC1155-SMI", types.Synthetic)
	module.Language = SmiLanguageSMIv1

	// Add SMIv1 type definitions
	module.Definitions = append(module.Definitions, createSMIv1TypeDefinitions()...)

	// Add OID root definitions (subset relevant to SMIv1)
	module.Definitions = append(module.Definitions, createSMIv1OidDefinitions()...)

	return module
}

// createRFC1065SMI creates the synthetic RFC1065-SMI module.
func createRFC1065SMI() *Module {
	module := NewModule("RFC1065-SMI", types.Synthetic)
	module.Language = SmiLanguageSMIv1

	// Same content as RFC1155-SMI
	module.Definitions = append(module.Definitions, createSMIv1TypeDefinitions()...)
	module.Definitions = append(module.Definitions, createSMIv1OidDefinitions()...)

	return module
}

// createRFC1212 creates the synthetic RFC-1212 module.
func createRFC1212() *Module {
	module := NewModule("RFC-1212", types.Synthetic)
	module.Language = SmiLanguageSMIv1
	// No definitions - MACRO only
	return module
}

// createRFC1215 creates the synthetic RFC-1215 module.
func createRFC1215() *Module {
	module := NewModule("RFC-1215", types.Synthetic)
	module.Language = SmiLanguageSMIv1
	// No definitions - MACRO only
	return module
}

// === Helper functions for creating types ===

// constrainedIntRange creates a constrained INTEGER type with a value range.
func constrainedIntRange(min, max RangeValue) TypeSyntax {
	return &TypeSyntaxConstrained{
		Base:       &TypeSyntaxTypeRef{Name: "INTEGER"},
		Constraint: &ConstraintRange{Ranges: []Range{{Min: min, Max: max}}},
	}
}

// constrainedOctetSize creates a constrained OCTET STRING type with size constraints.
func constrainedOctetSize(ranges []Range) TypeSyntax {
	return &TypeSyntaxConstrained{
		Base:       &TypeSyntaxOctetString{},
		Constraint: &ConstraintSize{Ranges: ranges},
	}
}

// constrainedOctetFixed creates a constrained OCTET STRING with a single fixed size.
func constrainedOctetFixed(size uint64) TypeSyntax {
	return constrainedOctetSize([]Range{
		{Min: &RangeValueUnsigned{Value: size}, Max: nil},
	})
}

// constrainedOctetRange creates a constrained OCTET STRING with a size range.
func constrainedOctetRange(min, max uint64) TypeSyntax {
	return constrainedOctetSize([]Range{
		NewRangeUnsigned(min, max),
	})
}

// constrainedUintRange creates a constrained INTEGER with unsigned range (0..max).
func constrainedUintRange(max uint64) TypeSyntax {
	return constrainedIntRange(
		&RangeValueUnsigned{Value: 0},
		&RangeValueUnsigned{Value: max},
	)
}

// makeOidValue creates a ValueAssignment for an OID definition.
func makeOidValue(name string, components []OidComponent) Definition {
	return &ValueAssignment{
		Name: name,
		Oid:  NewOidAssignment(components, types.Synthetic),
		Span: types.Synthetic,
	}
}

// makeTypeDef creates a TypeDef for a type definition without explicit base type.
func makeTypeDef(name string, syntax TypeSyntax) Definition {
	return &TypeDef{
		Name:                name,
		Syntax:              syntax,
		BaseType:            nil,
		DisplayHint:         "",
		Status:              types.StatusCurrent,
		Description:         "",
		Reference:           "",
		IsTextualConvention: false,
		Span:                types.Synthetic,
	}
}

// makeTypeDefWithBase creates a TypeDef for a base type definition with explicit base type.
func makeTypeDefWithBase(name string, syntax TypeSyntax, base types.BaseType) Definition {
	return &TypeDef{
		Name:                name,
		Syntax:              syntax,
		BaseType:            &base,
		DisplayHint:         "",
		Status:              types.StatusCurrent,
		Description:         "",
		Reference:           "",
		IsTextualConvention: false,
		Span:                types.Synthetic,
	}
}

// makeTypeDefObsolete creates a TypeDef for an obsolete type definition.
func makeTypeDefObsolete(name string, syntax TypeSyntax) Definition {
	return &TypeDef{
		Name:                name,
		Syntax:              syntax,
		BaseType:            nil,
		DisplayHint:         "",
		Status:              types.StatusObsolete,
		Description:         "",
		Reference:           "",
		IsTextualConvention: false,
		Span:                types.Synthetic,
	}
}

// makeTC creates a TypeDef for a textual convention.
func makeTC(name, displayHint string, syntax TypeSyntax) Definition {
	return &TypeDef{
		Name:                name,
		Syntax:              syntax,
		BaseType:            nil,
		DisplayHint:         displayHint,
		Status:              types.StatusCurrent,
		Description:         "",
		Reference:           "",
		IsTextualConvention: true,
		Span:                types.Synthetic,
	}
}

// makeTCObsolete creates a TypeDef for an obsolete textual convention.
func makeTCObsolete(name, displayHint string, syntax TypeSyntax) Definition {
	return &TypeDef{
		Name:                name,
		Syntax:              syntax,
		BaseType:            nil,
		DisplayHint:         displayHint,
		Status:              types.StatusObsolete,
		Description:         "",
		Reference:           "",
		IsTextualConvention: true,
		Span:                types.Synthetic,
	}
}

// makeTCWithEnum creates a TypeDef for a textual convention with enumerated values.
func makeTCWithEnum(name string, values []NamedNumber) Definition {
	return &TypeDef{
		Name:                name,
		Syntax:              &TypeSyntaxIntegerEnum{NamedNumbers: values},
		BaseType:            nil,
		DisplayHint:         "",
		Status:              types.StatusCurrent,
		Description:         "",
		Reference:           "",
		IsTextualConvention: true,
		Span:                types.Synthetic,
	}
}

// === OID definitions ===

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

// === SMIv2 base type definitions ===

func createBaseTypeDefinitions() []Definition {
	int32Min := int64(-2147483648)
	int32Max := int64(2147483647)
	uint32Max := uint64(4294967295)
	uint64Max := uint64(18446744073709551615)

	return []Definition{
		// Integer32 ::= INTEGER (-2147483648..2147483647)
		makeTypeDefWithBase("Integer32",
			constrainedIntRange(
				&RangeValueSigned{Value: int32Min},
				&RangeValueSigned{Value: int32Max},
			),
			types.BaseInteger32,
		),
		// Counter32 ::= [APPLICATION 1] IMPLICIT INTEGER (0..4294967295)
		makeTypeDefWithBase("Counter32",
			constrainedUintRange(uint32Max),
			types.BaseCounter32,
		),
		// Counter64 ::= [APPLICATION 6] IMPLICIT INTEGER (0..18446744073709551615)
		makeTypeDefWithBase("Counter64",
			constrainedUintRange(uint64Max),
			types.BaseCounter64,
		),
		// Gauge32 ::= [APPLICATION 2] IMPLICIT INTEGER (0..4294967295)
		makeTypeDefWithBase("Gauge32",
			constrainedUintRange(uint32Max),
			types.BaseGauge32,
		),
		// Unsigned32 ::= [APPLICATION 2] IMPLICIT INTEGER (0..4294967295)
		makeTypeDefWithBase("Unsigned32",
			constrainedUintRange(uint32Max),
			types.BaseUnsigned32,
		),
		// TimeTicks ::= [APPLICATION 3] IMPLICIT INTEGER (0..4294967295)
		makeTypeDefWithBase("TimeTicks",
			constrainedUintRange(uint32Max),
			types.BaseTimeTicks,
		),
		// IpAddress ::= [APPLICATION 0] IMPLICIT OCTET STRING (SIZE (4))
		makeTypeDefWithBase("IpAddress",
			constrainedOctetFixed(4),
			types.BaseIpAddress,
		),
		// Opaque ::= [APPLICATION 4] IMPLICIT OCTET STRING
		makeTypeDefWithBase("Opaque",
			&TypeSyntaxOctetString{},
			types.BaseOpaque,
		),
		// ObjectName ::= OBJECT IDENTIFIER
		makeTypeDef("ObjectName", &TypeSyntaxObjectIdentifier{}),
		// NotificationName ::= OBJECT IDENTIFIER
		makeTypeDef("NotificationName", &TypeSyntaxObjectIdentifier{}),
		// ExtUTCTime ::= OCTET STRING (SIZE (11 | 13)) - obsolete
		makeTypeDefObsolete("ExtUTCTime",
			constrainedOctetSize([]Range{
				{Min: &RangeValueUnsigned{Value: 11}, Max: nil},
				{Min: &RangeValueUnsigned{Value: 13}, Max: nil},
			}),
		),
		// ObjectSyntax, SimpleSyntax, ApplicationSyntax - protocol meta-types
		makeTypeDef("ObjectSyntax", &TypeSyntaxTypeRef{Name: "SimpleSyntax"}),
		makeTypeDef("SimpleSyntax", &TypeSyntaxTypeRef{Name: "INTEGER"}),
		makeTypeDef("ApplicationSyntax", &TypeSyntaxTypeRef{Name: "IpAddress"}),
	}
}

// === SMIv1 type definitions ===

func createSMIv1TypeDefinitions() []Definition {
	uint32Max := uint64(4294967295)

	return []Definition{
		// Counter ::= [APPLICATION 1] IMPLICIT INTEGER (0..4294967295)
		makeTypeDefWithBase("Counter", constrainedUintRange(uint32Max), types.BaseCounter32),
		// Gauge ::= [APPLICATION 2] IMPLICIT INTEGER (0..4294967295)
		makeTypeDefWithBase("Gauge", constrainedUintRange(uint32Max), types.BaseGauge32),
		// IpAddress ::= [APPLICATION 0] IMPLICIT OCTET STRING (SIZE (4))
		makeTypeDefWithBase("IpAddress", constrainedOctetFixed(4), types.BaseIpAddress),
		// NetworkAddress ::= CHOICE { internet IpAddress }
		makeTypeDefWithBase("NetworkAddress", &TypeSyntaxTypeRef{Name: "IpAddress"}, types.BaseIpAddress),
		// TimeTicks ::= [APPLICATION 3] IMPLICIT INTEGER (0..4294967295)
		makeTypeDefWithBase("TimeTicks", constrainedUintRange(uint32Max), types.BaseTimeTicks),
		// Opaque ::= [APPLICATION 4] IMPLICIT OCTET STRING
		makeTypeDefWithBase("Opaque", &TypeSyntaxOctetString{}, types.BaseOpaque),
		// ObjectName ::= OBJECT IDENTIFIER
		makeTypeDef("ObjectName", &TypeSyntaxObjectIdentifier{}),
	}
}

// === SMIv1 OID definitions ===

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

// === Textual convention definitions ===

func createTCDefinitions() []Definition {
	int32Max := int64(2147483647)

	return []Definition{
		// DisplayString ::= TEXTUAL-CONVENTION DISPLAY-HINT "255a" SYNTAX OCTET STRING (SIZE (0..255))
		makeTC("DisplayString", "255a", constrainedOctetRange(0, 255)),
		// PhysAddress ::= TEXTUAL-CONVENTION DISPLAY-HINT "1x:" SYNTAX OCTET STRING
		makeTC("PhysAddress", "1x:", &TypeSyntaxOctetString{}),
		// MacAddress ::= TEXTUAL-CONVENTION DISPLAY-HINT "1x:" SYNTAX OCTET STRING (SIZE (6))
		makeTC("MacAddress", "1x:", constrainedOctetFixed(6)),
		// TruthValue ::= TEXTUAL-CONVENTION SYNTAX INTEGER { true(1), false(2) }
		makeTCWithEnum("TruthValue", []NamedNumber{
			{Name: "true", Value: 1},
			{Name: "false", Value: 2},
		}),
		// RowStatus ::= TEXTUAL-CONVENTION SYNTAX INTEGER { active(1), ... }
		makeTCWithEnum("RowStatus", []NamedNumber{
			{Name: "active", Value: 1},
			{Name: "notInService", Value: 2},
			{Name: "notReady", Value: 3},
			{Name: "createAndGo", Value: 4},
			{Name: "createAndWait", Value: 5},
			{Name: "destroy", Value: 6},
		}),
		// StorageType ::= TEXTUAL-CONVENTION SYNTAX INTEGER { other(1), ... }
		makeTCWithEnum("StorageType", []NamedNumber{
			{Name: "other", Value: 1},
			{Name: "volatile", Value: 2},
			{Name: "nonVolatile", Value: 3},
			{Name: "permanent", Value: 4},
			{Name: "readOnly", Value: 5},
		}),
		// TimeStamp ::= TEXTUAL-CONVENTION SYNTAX TimeTicks
		makeTC("TimeStamp", "", &TypeSyntaxTypeRef{Name: "TimeTicks"}),
		// TimeInterval ::= TEXTUAL-CONVENTION SYNTAX INTEGER (0..2147483647)
		makeTC("TimeInterval", "",
			constrainedIntRange(
				&RangeValueUnsigned{Value: 0},
				&RangeValueSigned{Value: int32Max},
			),
		),
		// DateAndTime ::= TEXTUAL-CONVENTION DISPLAY-HINT "2d-1d-1d,1d:1d:1d.1d,1a1d:1d" SYNTAX OCTET STRING (SIZE (8 | 11))
		makeTC("DateAndTime", "2d-1d-1d,1d:1d:1d.1d,1a1d:1d",
			constrainedOctetSize([]Range{
				{Min: &RangeValueUnsigned{Value: 8}, Max: nil},
				{Min: &RangeValueUnsigned{Value: 11}, Max: nil},
			}),
		),
		// TestAndIncr ::= TEXTUAL-CONVENTION SYNTAX INTEGER (0..2147483647)
		makeTC("TestAndIncr", "",
			constrainedIntRange(
				&RangeValueUnsigned{Value: 0},
				&RangeValueSigned{Value: int32Max},
			),
		),
		// AutonomousType ::= TEXTUAL-CONVENTION SYNTAX OBJECT IDENTIFIER
		makeTC("AutonomousType", "", &TypeSyntaxObjectIdentifier{}),
		// InstancePointer ::= TEXTUAL-CONVENTION (obsolete) SYNTAX OBJECT IDENTIFIER
		makeTCObsolete("InstancePointer", "", &TypeSyntaxObjectIdentifier{}),
		// VariablePointer ::= TEXTUAL-CONVENTION SYNTAX OBJECT IDENTIFIER
		makeTC("VariablePointer", "", &TypeSyntaxObjectIdentifier{}),
		// RowPointer ::= TEXTUAL-CONVENTION SYNTAX OBJECT IDENTIFIER
		makeTC("RowPointer", "", &TypeSyntaxObjectIdentifier{}),
		// TDomain ::= TEXTUAL-CONVENTION SYNTAX OBJECT IDENTIFIER
		makeTC("TDomain", "", &TypeSyntaxObjectIdentifier{}),
		// TAddress ::= TEXTUAL-CONVENTION SYNTAX OCTET STRING (SIZE (1..255))
		makeTC("TAddress", "", constrainedOctetRange(1, 255)),
	}
}
