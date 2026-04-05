package resolver

import (
	"github.com/golangsnmp/gomib/internal/model"
	"github.com/golangsnmp/gomib/internal/module"
)

// typeClass classifies well-known SMI type names by their defining module.
type typeClass int

const (
	typeClassASN1Primitive typeClass = iota + 1 // INTEGER, OCTET STRING, OBJECT IDENTIFIER, BITS
	typeClassSmiGlobal                          // Integer32, Counter32, etc. (SNMPv2-SMI)
	typeClassSmiV1Global                        // Counter, Gauge, NetworkAddress (RFC1155-SMI)
	typeClassSNMPv2TC                           // DisplayString, TruthValue, etc. (SNMPv2-TC)
)

// wellKnownTypes maps type names to their class. Used by findWellKnownModuleForType
// and isBareTypeIndex to avoid repeated switch statements.
var wellKnownTypes = map[string]typeClass{
	// ASN.1 primitives
	"INTEGER": typeClassASN1Primitive, "OCTET STRING": typeClassASN1Primitive,
	"OBJECT IDENTIFIER": typeClassASN1Primitive, "BITS": typeClassASN1Primitive,
	// SMI global types (SNMPv2-SMI)
	"Integer32": typeClassSmiGlobal, "Counter32": typeClassSmiGlobal,
	"Counter64": typeClassSmiGlobal, "Gauge32": typeClassSmiGlobal,
	"Unsigned32": typeClassSmiGlobal, "TimeTicks": typeClassSmiGlobal,
	"IpAddress": typeClassSmiGlobal, "Opaque": typeClassSmiGlobal,
	// SMIv1 types (RFC1155-SMI only)
	"Counter": typeClassSmiV1Global, "Gauge": typeClassSmiV1Global,
	"NetworkAddress": typeClassSmiV1Global,
	// SNMPv2-TC textual conventions (RFC 2579)
	"DisplayString": typeClassSNMPv2TC, "TruthValue": typeClassSNMPv2TC,
	"PhysAddress": typeClassSNMPv2TC, "MacAddress": typeClassSNMPv2TC,
	"RowStatus": typeClassSNMPv2TC, "TimeStamp": typeClassSNMPv2TC,
	"TimeInterval": typeClassSNMPv2TC, "DateAndTime": typeClassSNMPv2TC,
	"StorageType": typeClassSNMPv2TC, "TestAndIncr": typeClassSNMPv2TC,
	"AutonomousType": typeClassSNMPv2TC, "VariablePointer": typeClassSNMPv2TC,
	"RowPointer": typeClassSNMPv2TC, "InstancePointer": typeClassSNMPv2TC,
	"TDomain": typeClassSNMPv2TC, "TAddress": typeClassSNMPv2TC,
}

// findWellKnownModuleForType returns the well-known base module expected to
// define the given type name. ASN.1 primitives are always checked; other
// well-known modules (SMI globals, SMIv1 types, SNMPv2-TC) require constrained
// fallbacks (Normal+). Returns nil if no well-known module matches.
func (c *resolverContext) findWellKnownModuleForType(name string) *module.Module {
	cls := wellKnownTypes[name]

	// RFC-compliant: ASN.1 primitives are always available
	if cls == typeClassASN1Primitive {
		return c.snmpv2SMIModule
	}

	if !c.strictness.AllowConstrainedFallbacks() {
		return nil
	}

	// Normal+: well-known modules for SMI globals, SMIv1 types, SNMPv2-TC
	switch cls {
	case typeClassSmiGlobal:
		return c.snmpv2SMIModule
	case typeClassSmiV1Global:
		return c.rfc1155SMIModule
	case typeClassSNMPv2TC:
		return c.snmpv2TCModule
	default:
		return nil
	}
}

// tryWellKnownTypeFallbacks searches ASN.1 primitives (always) and well-known
// base modules (Normal+) for a type by name.
func (c *resolverContext) tryWellKnownTypeFallbacks(name string) (*model.Type, bool) {
	if m := c.findWellKnownModuleForType(name); m != nil {
		return c.lookupTypeDirect(m, name)
	}
	return nil, false
}
