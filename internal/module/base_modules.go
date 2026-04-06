package module

import (
	"slices"
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
