package resolver

import (
	"log/slog"

	"github.com/golangsnmp/gomib/internal/mibimpl"
	"github.com/golangsnmp/gomib/internal/module"
	"github.com/golangsnmp/gomib/internal/types"
	"github.com/golangsnmp/gomib/mib"
)

// ResolverContext holds indices and working state for all resolution phases.
type ResolverContext struct {
	Builder *mibimpl.Builder

	Modules []*module.Module

	// ModuleIndex maps module name to parsed modules (multiple versions possible).
	ModuleIndex map[string][]*module.Module

	// ModuleToResolved maps parsed module to resolved module.
	ModuleToResolved map[*module.Module]*mibimpl.Module

	// ResolvedToModule is the reverse of ModuleToResolved.
	ResolvedToModule map[*mibimpl.Module]*module.Module

	// ModuleSymbolToNode maps module -> symbol -> Node for OID lookups.
	ModuleSymbolToNode map[*module.Module]map[string]*mibimpl.Node

	// ModuleImports maps module -> symbol -> source module for import chain traversal.
	ModuleImports map[*module.Module]map[string]*module.Module

	// ModuleSymbolToType maps module -> symbol -> Type for type lookups.
	ModuleSymbolToType map[*module.Module]map[string]*mibimpl.Type

	// ModuleDefNames caches definition names per module for import resolution.
	ModuleDefNames map[*module.Module]map[string]struct{}

	// Snmpv2SMIModule is the SNMPv2-SMI base module (for primitive types).
	Snmpv2SMIModule *module.Module

	// Rfc1155SMIModule is the RFC1155-SMI base module (for SMIv1 types).
	Rfc1155SMIModule *module.Module

	// Snmpv2TCModule is the SNMPv2-TC module (for standard textual conventions).
	Snmpv2TCModule *module.Module

	// Unresolved references collected during resolution
	unresolvedImports      []unresolvedImport
	unresolvedTypes        []unresolvedType
	unresolvedOids         []unresolvedOid
	unresolvedIndexes      []unresolvedIndex
	unresolvedNotifObjects []unresolvedNotifObject

	// Diagnostic configuration and collection
	diagConfig  mib.DiagnosticConfig
	diagnostics []mib.Diagnostic

	types.Logger
}

type unresolvedImport struct {
	importingModule *module.Module
	fromModule      string
	symbol          string
	reason          string
	span            types.Span
}

type unresolvedType struct {
	module     *module.Module
	referrer   string
	referenced string
	span       types.Span
}

type unresolvedOid struct {
	module     *module.Module
	definition string
	component  string
	span       types.Span
}

type unresolvedIndex struct {
	module      *module.Module
	row         string
	indexObject string
	span        types.Span
}

type unresolvedNotifObject struct {
	module       *module.Module
	notification string
	object       string
	span         types.Span
}

type capacityHints struct {
	modules       int
	objects       int
	types         int
	notifications int
	nodes         int
	imports       int
}

func scanCapacityHints(mods []*module.Module) capacityHints {
	h := capacityHints{modules: len(mods)}
	for _, mod := range mods {
		h.imports += len(mod.Imports)
		for _, def := range mod.Definitions {
			if def.DefinitionOid() != nil {
				h.nodes++
			}
			switch def.(type) {
			case *module.ObjectType:
				h.objects++
			case *module.TypeDef:
				h.types++
			case *module.Notification:
				h.notifications++
			}
		}
	}
	return h
}

func newResolverContext(mods []*module.Module, logger *slog.Logger, diagConfig mib.DiagnosticConfig) *ResolverContext {
	h := scanCapacityHints(mods)
	return &ResolverContext{
		Builder:            mibimpl.NewBuilder(),
		Modules:            mods,
		ModuleIndex:        make(map[string][]*module.Module, h.modules),
		ModuleToResolved:   make(map[*module.Module]*mibimpl.Module, h.modules),
		ResolvedToModule:   make(map[*mibimpl.Module]*module.Module, h.modules),
		ModuleSymbolToNode: make(map[*module.Module]map[string]*mibimpl.Node, h.modules),
		ModuleImports:      make(map[*module.Module]map[string]*module.Module, h.modules),
		ModuleSymbolToType: make(map[*module.Module]map[string]*mibimpl.Type, h.modules),
		ModuleDefNames:     make(map[*module.Module]map[string]struct{}, h.modules),
		diagConfig:         diagConfig,
		Logger:             types.Logger{L: logger},
	}
}

// LookupNodeForModule resolves a node by name, traversing imports from mod.
func (c *ResolverContext) LookupNodeForModule(mod *module.Module, name string) (*mibimpl.Node, bool) {
	return c.lookupNodeInModuleScope(mod, name)
}

// LookupNodeInModule resolves a node across all versions of a named module.
func (c *ResolverContext) LookupNodeInModule(moduleName, name string) (*mibimpl.Node, bool) {
	candidates := c.ModuleIndex[moduleName]
	for _, mod := range candidates {
		if node, ok := c.LookupNodeForModule(mod, name); ok {
			return node, true
		}
	}
	return nil, false
}

// LookupNodeGlobal searches all modules for a node with the given name.
// Iterates in module-list order for deterministic results.
func (c *ResolverContext) LookupNodeGlobal(name string) (*mibimpl.Node, bool) {
	for _, mod := range c.Modules {
		if symbols := c.ModuleSymbolToNode[mod]; symbols != nil {
			if node, ok := symbols[name]; ok {
				return node, true
			}
		}
	}
	return nil, false
}

// lookupTypeInModule looks up a type directly in a module's symbol table.
func (c *ResolverContext) lookupTypeInModule(mod *module.Module, name string) (*mibimpl.Type, bool) {
	if mod == nil {
		return nil, false
	}
	if symbols := c.ModuleSymbolToType[mod]; symbols != nil {
		if t, ok := symbols[name]; ok {
			return t, true
		}
	}
	return nil, false
}

// LookupType searches for a type by name, trying well-known modules first.
// Beyond ASN.1 primitives, global search is only enabled in permissive mode.
func (c *ResolverContext) LookupType(name string) (*mibimpl.Type, bool) {
	// RFC-compliant: ASN.1 primitives are always available
	if isASN1Primitive(name) {
		if t, ok := c.lookupTypeInModule(c.Snmpv2SMIModule, name); ok {
			return t, ok
		}
	}

	// Permissive only: global type search
	if !c.diagConfig.AllowBestGuessFallbacks() {
		return nil, false
	}

	// Try SNMPv2-SMI for SMI global types
	if c.Snmpv2SMIModule != nil {
		if t, ok := c.LookupTypeForModule(c.Snmpv2SMIModule, name); ok {
			return t, true
		}
	}

	// Try RFC1155-SMI for SMIv1 types (Counter, Gauge, NetworkAddress)
	if isSmiV1GlobalType(name) {
		if t, ok := c.lookupTypeInModule(c.Rfc1155SMIModule, name); ok {
			return t, ok
		}
	}

	// Try SNMPv2-TC for standard textual conventions (DisplayString, TruthValue, etc.)
	if isSNMPv2TCType(name) {
		if t, ok := c.lookupTypeInModule(c.Snmpv2TCModule, name); ok {
			return t, ok
		}
	}

	// Iterate in module-list order for deterministic results.
	for _, mod := range c.Modules {
		if symbols := c.ModuleSymbolToType[mod]; symbols != nil {
			if t, ok := symbols[name]; ok {
				return t, true
			}
		}
	}
	return nil, false
}

// LookupTypeForModule resolves a type by name, traversing imports from mod.
// Falls back to well-known base modules when permissive mode is enabled.
func (c *ResolverContext) LookupTypeForModule(mod *module.Module, name string) (*mibimpl.Type, bool) {
	if t, ok := c.lookupTypeInModuleScope(mod, name); ok {
		return t, true
	}

	// RFC-compliant: ASN.1 primitives are always available
	if isASN1Primitive(name) {
		if t, ok := c.lookupTypeInModule(c.Snmpv2SMIModule, name); ok {
			return t, ok
		}
	}

	if !c.diagConfig.AllowBestGuessFallbacks() {
		return nil, false
	}

	// Permissive only: SMI global types without explicit import
	if isSmiGlobalType(name) {
		if t, ok := c.lookupTypeInModule(c.Snmpv2SMIModule, name); ok {
			return t, ok
		}
	}

	// Permissive only: SMIv1 types (Counter, Gauge, NetworkAddress) from RFC1155-SMI
	if isSmiV1GlobalType(name) {
		if t, ok := c.lookupTypeInModule(c.Rfc1155SMIModule, name); ok {
			return t, ok
		}
	}

	// Permissive only: SNMPv2-TC textual conventions (DisplayString, TruthValue, etc.)
	if isSNMPv2TCType(name) {
		if t, ok := c.lookupTypeInModule(c.Snmpv2TCModule, name); ok {
			return t, ok
		}
	}

	return nil, false
}

// maxImportChainDepth is the maximum depth of import chains to traverse.
// This is generous for real-world MIBs (typically 2-4 deep) while preventing
// infinite loops from circular imports.
const maxImportChainDepth = 8

// lookupInModuleScope traverses module imports to find a symbol, using the
// provided symbol table getter.
func lookupInModuleScope[T any](
	mod *module.Module,
	name string,
	getSymbols func(*module.Module) map[string]T,
	getImports func(*module.Module) map[string]*module.Module,
) (T, bool) {
	var zero T
	var visitedStack [maxImportChainDepth]*module.Module
	visitedCount := 0
	current := mod

	for {
		for i := 0; i < visitedCount; i++ {
			if visitedStack[i] == current {
				return zero, false
			}
		}
		if visitedCount < len(visitedStack) {
			visitedStack[visitedCount] = current
			visitedCount++
		} else {
			return zero, false
		}

		if symbols := getSymbols(current); symbols != nil {
			if val, ok := symbols[name]; ok {
				return val, true
			}
		}

		if imports := getImports(current); imports != nil {
			if next, ok := imports[name]; ok {
				current = next
				continue
			}
		}
		return zero, false
	}
}

func (c *ResolverContext) lookupNodeInModuleScope(mod *module.Module, name string) (*mibimpl.Node, bool) {
	return lookupInModuleScope(mod, name,
		func(m *module.Module) map[string]*mibimpl.Node { return c.ModuleSymbolToNode[m] },
		func(m *module.Module) map[string]*module.Module { return c.ModuleImports[m] },
	)
}

func (c *ResolverContext) lookupTypeInModuleScope(mod *module.Module, name string) (*mibimpl.Type, bool) {
	return lookupInModuleScope(mod, name,
		func(m *module.Module) map[string]*mibimpl.Type { return c.ModuleSymbolToType[m] },
		func(m *module.Module) map[string]*module.Module { return c.ModuleImports[m] },
	)
}

// RegisterImport maps a symbol in importingModule to its source module.
func (c *ResolverContext) RegisterImport(importingModule *module.Module, symbol string, sourceModule *module.Module) {
	imports := c.ModuleImports[importingModule]
	if imports == nil {
		imports = make(map[string]*module.Module)
		c.ModuleImports[importingModule] = imports
	}
	imports[symbol] = sourceModule
}

// RegisterModuleNodeSymbol binds a symbol name to a node within a module scope.
func (c *ResolverContext) RegisterModuleNodeSymbol(mod *module.Module, symbol string, node *mibimpl.Node) {
	symbols := c.ModuleSymbolToNode[mod]
	if symbols == nil {
		symbols = make(map[string]*mibimpl.Node)
		c.ModuleSymbolToNode[mod] = symbols
	}
	if _, exists := symbols[symbol]; exists && c.TraceEnabled() {
		c.Trace("overwriting node symbol registration",
			slog.String("module", mod.Name),
			slog.String("symbol", symbol))
	}
	symbols[symbol] = node
}

// RegisterModuleTypeSymbol binds a symbol name to a type within a module scope.
func (c *ResolverContext) RegisterModuleTypeSymbol(mod *module.Module, name string, t *mibimpl.Type) {
	symbols := c.ModuleSymbolToType[mod]
	if symbols == nil {
		symbols = make(map[string]*mibimpl.Type)
		c.ModuleSymbolToType[mod] = symbols
	}
	symbols[name] = t
}

// EmitDiagnostic records a diagnostic, filtered by the current config's severity and code rules.
func (c *ResolverContext) EmitDiagnostic(code string, severity mib.Severity, moduleName string, line, col int, message string) {
	if !c.diagConfig.ShouldReport(code, severity) {
		return
	}
	c.diagnostics = append(c.diagnostics, mib.Diagnostic{
		Severity: severity,
		Code:     code,
		Message:  message,
		Module:   moduleName,
		Line:     line,
		Column:   col,
	})
}

// Diagnostics returns all diagnostics collected during resolution.
func (c *ResolverContext) Diagnostics() []mib.Diagnostic {
	return c.diagnostics
}

// DiagnosticConfig returns the active strictness and filtering configuration.
func (c *ResolverContext) DiagnosticConfig() mib.DiagnosticConfig {
	return c.diagConfig
}

func (c *ResolverContext) emitUnresolvedDiagnostic(mod *module.Module, code string, severity mib.Severity, msg string) {
	modName := ""
	if mod != nil {
		modName = mod.Name
	}
	c.EmitDiagnostic(code, severity, modName, 0, 0, msg)
}

// RecordUnresolvedImport tracks a symbol that could not be resolved from its source module.
func (c *ResolverContext) RecordUnresolvedImport(importingModule *module.Module, fromModule, symbol, reason string, span types.Span) {
	c.unresolvedImports = append(c.unresolvedImports, unresolvedImport{
		importingModule: importingModule,
		fromModule:      fromModule,
		symbol:          symbol,
		reason:          reason,
		span:            span,
	})
	code := "import-not-found"
	if reason == "module not found" {
		code = "import-module-not-found"
	}
	c.emitUnresolvedDiagnostic(importingModule, code, mib.SeverityError,
		"unresolved import: "+symbol+" from "+fromModule+" ("+reason+")")
}

// RecordUnresolvedType tracks a type definition whose parent type could not be found.
func (c *ResolverContext) RecordUnresolvedType(mod *module.Module, referrer, referenced string, span types.Span) {
	c.unresolvedTypes = append(c.unresolvedTypes, unresolvedType{
		module:     mod,
		referrer:   referrer,
		referenced: referenced,
		span:       span,
	})
	c.emitUnresolvedDiagnostic(mod, "type-unknown", mib.SeverityError,
		"unresolved type: "+referrer+" references unknown type "+referenced)
}

// RecordUnresolvedOid tracks an OID definition whose parent component could not be resolved.
func (c *ResolverContext) RecordUnresolvedOid(mod *module.Module, defName, component string, span types.Span) {
	c.unresolvedOids = append(c.unresolvedOids, unresolvedOid{
		module:     mod,
		definition: defName,
		component:  component,
		span:       span,
	})
	c.emitUnresolvedDiagnostic(mod, "oid-orphan", mib.SeverityWarning,
		"unresolved OID: "+defName+" references unknown parent "+component)
}

// RecordUnresolvedIndex tracks a row's INDEX entry that references a missing object.
func (c *ResolverContext) RecordUnresolvedIndex(mod *module.Module, row, indexObject string, span types.Span) {
	c.unresolvedIndexes = append(c.unresolvedIndexes, unresolvedIndex{
		module:      mod,
		row:         row,
		indexObject: indexObject,
		span:        span,
	})
	c.emitUnresolvedDiagnostic(mod, "index-unresolved", mib.SeverityError,
		"unresolved INDEX: "+row+" references unknown object "+indexObject)
}

// RecordUnresolvedNotificationObject tracks a notification's OBJECTS entry that references a missing object.
func (c *ResolverContext) RecordUnresolvedNotificationObject(mod *module.Module, notification, object string, span types.Span) {
	c.unresolvedNotifObjects = append(c.unresolvedNotifObjects, unresolvedNotifObject{
		module:       mod,
		notification: notification,
		object:       object,
		span:         span,
	})
	c.emitUnresolvedDiagnostic(mod, "objects-unresolved", mib.SeverityWarning,
		"unresolved OBJECTS: "+notification+" references unknown object "+object)
}

// DropModules releases parsed module data to free memory after resolution completes.
func (c *ResolverContext) DropModules() {
	c.Modules = nil
	c.ModuleIndex = nil
	c.ModuleDefNames = nil
}

// FinalizeUnresolved copies collected unresolved references and diagnostics into the Mib builder.
func (c *ResolverContext) FinalizeUnresolved() {
	for _, u := range c.unresolvedImports {
		modName := ""
		if u.importingModule != nil {
			modName = u.importingModule.Name
		}
		c.Builder.AddUnresolved(mib.UnresolvedRef{
			Kind:   "import",
			Symbol: u.symbol,
			Module: modName,
		})
	}
	for _, u := range c.unresolvedTypes {
		modName := ""
		if u.module != nil {
			modName = u.module.Name
		}
		c.Builder.AddUnresolved(mib.UnresolvedRef{
			Kind:   "type",
			Symbol: u.referenced,
			Module: modName,
		})
	}
	for _, u := range c.unresolvedOids {
		modName := ""
		if u.module != nil {
			modName = u.module.Name
		}
		c.Builder.AddUnresolved(mib.UnresolvedRef{
			Kind:   "oid",
			Symbol: u.component,
			Module: modName,
		})
	}
	for _, u := range c.unresolvedIndexes {
		modName := ""
		if u.module != nil {
			modName = u.module.Name
		}
		c.Builder.AddUnresolved(mib.UnresolvedRef{
			Kind:   "index",
			Symbol: u.indexObject,
			Module: modName,
		})
	}
	for _, u := range c.unresolvedNotifObjects {
		modName := ""
		if u.module != nil {
			modName = u.module.Name
		}
		c.Builder.AddUnresolved(mib.UnresolvedRef{
			Kind:   "notification-object",
			Symbol: u.object,
			Module: modName,
		})
	}

	for _, d := range c.diagnostics {
		c.Builder.AddDiagnostic(d)
	}
}

func isASN1Primitive(name string) bool {
	switch name {
	case "INTEGER", "OCTET STRING", "OBJECT IDENTIFIER", "BITS":
		return true
	default:
		return false
	}
}

func isSmiGlobalType(name string) bool {
	switch name {
	case "Integer32", "Counter32", "Counter64", "Gauge32", "Unsigned32", "TimeTicks", "IpAddress", "Opaque":
		return true
	default:
		return false
	}
}

// isSmiV1GlobalType returns true for SMIv1 type names that only exist in
// RFC1155-SMI (Counter, Gauge, NetworkAddress). Types shared with SMIv2
// (IpAddress, TimeTicks, Opaque) are handled by isSmiGlobalType.
func isSmiV1GlobalType(name string) bool {
	switch name {
	case "Counter", "Gauge", "NetworkAddress":
		return true
	default:
		return false
	}
}

// isSNMPv2TCType returns true for well-known textual conventions defined
// in SNMPv2-TC (RFC 2579) that vendor MIBs commonly use without imports.
func isSNMPv2TCType(name string) bool {
	switch name {
	case "DisplayString", "TruthValue", "PhysAddress", "MacAddress",
		"RowStatus", "TimeStamp", "TimeInterval", "DateAndTime",
		"StorageType", "TestAndIncr", "AutonomousType",
		"VariablePointer", "RowPointer", "InstancePointer",
		"TDomain", "TAddress":
		return true
	default:
		return false
	}
}
