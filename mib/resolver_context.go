package mib

import (
	"fmt"
	"log/slog"

	"github.com/golangsnmp/gomib/internal/graph"
	"github.com/golangsnmp/gomib/internal/module"
	"github.com/golangsnmp/gomib/internal/types"
)

// Import failure reason strings, shared between imports.go and context.go.
const (
	reasonModuleNotFound    = "module_not_found"
	reasonSymbolNotExported = "symbol_not_exported"
)

// resolverContext holds indices and working state for all resolution phases.
type resolverContext struct {
	Mib *Mib

	Modules []*module.Module

	// ModuleIndex maps module name to parsed modules (multiple versions possible).
	ModuleIndex map[string][]*module.Module

	// ModuleToResolved maps parsed module to resolved module.
	ModuleToResolved map[*module.Module]*Module

	// ResolvedToModule is the reverse of ModuleToResolved.
	ResolvedToModule map[*Module]*module.Module

	// ModuleSymbolToNode maps module -> symbol -> Node for OID lookups.
	ModuleSymbolToNode map[*module.Module]map[string]*Node

	// ModuleImports maps module -> symbol -> source module for import chain traversal.
	ModuleImports map[*module.Module]map[string]*module.Module

	// ModuleSymbolToType maps module -> symbol -> Type for type lookups.
	ModuleSymbolToType map[*module.Module]map[string]*Type

	// ModuleDefNames caches definition names per module for import resolution.
	ModuleDefNames map[*module.Module]map[string]struct{}

	// ModuleOidDefNames caches names of definitions that have OIDs, per module.
	// Used by findOidDefiningModule to avoid O(n) linear scans.
	ModuleOidDefNames map[*module.Module]map[string]struct{}

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
	diagConfig  DiagnosticConfig
	diagnostics []Diagnostic

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

func newResolverContext(mods []*module.Module, logger *slog.Logger, diagConfig DiagnosticConfig) *resolverContext {
	n := len(mods)
	ctx := &resolverContext{
		Mib:                newMib(),
		Modules:            mods,
		ModuleIndex:        make(map[string][]*module.Module, n),
		ModuleToResolved:   make(map[*module.Module]*Module, n),
		ResolvedToModule:   make(map[*Module]*module.Module, n),
		ModuleSymbolToNode: make(map[*module.Module]map[string]*Node, n),
		ModuleImports:      make(map[*module.Module]map[string]*module.Module, n),
		ModuleSymbolToType: make(map[*module.Module]map[string]*Type, n),
		ModuleDefNames:     make(map[*module.Module]map[string]struct{}, n),
		ModuleOidDefNames:  make(map[*module.Module]map[string]struct{}, n),
		diagConfig:         diagConfig,
		Logger:             types.Logger{L: logger},
	}
	// Pre-populate OID definition name index for all initial modules.
	// This allows findOidDefiningModule to use O(1) lookups instead of
	// scanning all definitions. registerModules rebuilds this with
	// base modules included.
	for _, mod := range mods {
		oidDefs := make(map[string]struct{})
		for _, def := range mod.Definitions {
			if def.DefinitionOid() != nil {
				oidDefs[def.DefinitionName()] = struct{}{}
			}
		}
		ctx.ModuleOidDefNames[mod] = oidDefs
	}
	return ctx
}

// LookupNodeForModule resolves a node by name, traversing imports from mod.
func (c *resolverContext) LookupNodeForModule(mod *module.Module, name string) (*Node, bool) {
	return c.lookupNodeInModuleScope(mod, name)
}

// LookupNodeInModule resolves a node across all versions of a named module.
func (c *resolverContext) LookupNodeInModule(moduleName, name string) (*Node, bool) {
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
func (c *resolverContext) LookupNodeGlobal(name string) (*Node, bool) {
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
func (c *resolverContext) lookupTypeInModule(mod *module.Module, name string) (*Type, bool) {
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

// tryWellKnownTypeFallbacks searches ASN.1 primitives (always) and well-known
// base modules (permissive only) for a type by name.
func (c *resolverContext) tryWellKnownTypeFallbacks(name string) (*Type, bool) {
	// RFC-compliant: ASN.1 primitives are always available
	if isASN1Primitive(name) {
		if t, ok := c.lookupTypeInModule(c.Snmpv2SMIModule, name); ok {
			return t, true
		}
	}

	if !c.diagConfig.AllowBestGuessFallbacks() {
		return nil, false
	}

	// Permissive only: SMI global types from SNMPv2-SMI
	if isSmiGlobalType(name) {
		if t, ok := c.lookupTypeInModule(c.Snmpv2SMIModule, name); ok {
			return t, true
		}
	}

	// Permissive only: SMIv1 types (Counter, Gauge, NetworkAddress) from RFC1155-SMI
	if isSmiV1GlobalType(name) {
		if t, ok := c.lookupTypeInModule(c.Rfc1155SMIModule, name); ok {
			return t, true
		}
	}

	// Permissive only: SNMPv2-TC textual conventions (DisplayString, TruthValue, etc.)
	if isSNMPv2TCType(name) {
		if t, ok := c.lookupTypeInModule(c.Snmpv2TCModule, name); ok {
			return t, true
		}
	}

	return nil, false
}

// LookupType searches for a type by name, trying well-known modules first.
// Beyond ASN.1 primitives, global search is only enabled in permissive mode.
func (c *resolverContext) LookupType(name string) (*Type, bool) {
	if t, ok := c.tryWellKnownTypeFallbacks(name); ok {
		return t, true
	}

	if !c.diagConfig.AllowBestGuessFallbacks() {
		return nil, false
	}

	// Permissive only: scan all modules for unknown types.
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
func (c *resolverContext) LookupTypeForModule(mod *module.Module, name string) (*Type, bool) {
	if t, ok := c.lookupTypeInModuleScope(mod, name); ok {
		return t, true
	}
	return c.tryWellKnownTypeFallbacks(name)
}

// lookupInModuleScope looks up a symbol in the module's own symbols, then
// follows a single import hop. ModuleImports entries are expected to already
// be transitively resolved to the defining module by resolveTransitiveImports.
func lookupInModuleScope[T any](
	mod *module.Module,
	name string,
	getSymbols func(*module.Module) map[string]T,
	getImports func(*module.Module) map[string]*module.Module,
) (T, bool) {
	var zero T

	if symbols := getSymbols(mod); symbols != nil {
		if val, ok := symbols[name]; ok {
			return val, true
		}
	}

	if imports := getImports(mod); imports != nil {
		if source, ok := imports[name]; ok {
			if symbols := getSymbols(source); symbols != nil {
				if val, ok := symbols[name]; ok {
					return val, true
				}
			}
		}
	}

	return zero, false
}

func (c *resolverContext) lookupNodeInModuleScope(mod *module.Module, name string) (*Node, bool) {
	return lookupInModuleScope(mod, name,
		func(m *module.Module) map[string]*Node { return c.ModuleSymbolToNode[m] },
		func(m *module.Module) map[string]*module.Module { return c.ModuleImports[m] },
	)
}

func (c *resolverContext) lookupTypeInModuleScope(mod *module.Module, name string) (*Type, bool) {
	return lookupInModuleScope(mod, name,
		func(m *module.Module) map[string]*Type { return c.ModuleSymbolToType[m] },
		func(m *module.Module) map[string]*module.Module { return c.ModuleImports[m] },
	)
}

// registerImport maps a symbol in importingModule to its source module.
func (c *resolverContext) registerImport(importingModule *module.Module, symbol string, sourceModule *module.Module) {
	imports := c.ModuleImports[importingModule]
	if imports == nil {
		imports = make(map[string]*module.Module)
		c.ModuleImports[importingModule] = imports
	}
	imports[symbol] = sourceModule
}

// registerModuleNodeSymbol binds a symbol name to a node within a module scope.
func (c *resolverContext) registerModuleNodeSymbol(mod *module.Module, symbol string, node *Node) {
	symbols := c.ModuleSymbolToNode[mod]
	if symbols == nil {
		symbols = make(map[string]*Node)
		c.ModuleSymbolToNode[mod] = symbols
	}
	if _, exists := symbols[symbol]; exists && c.TraceEnabled() {
		c.Trace("overwriting node symbol registration",
			slog.String("module", mod.Name),
			slog.String("symbol", symbol))
	}
	symbols[symbol] = node
}

// registerModuleTypeSymbol binds a symbol name to a type within a module scope.
func (c *resolverContext) registerModuleTypeSymbol(mod *module.Module, name string, t *Type) {
	symbols := c.ModuleSymbolToType[mod]
	if symbols == nil {
		symbols = make(map[string]*Type)
		c.ModuleSymbolToType[mod] = symbols
	}
	symbols[name] = t
}

// EmitDiagnostic records a diagnostic, filtered by the current config's severity and code rules.
// If mod is non-nil and has a line table, the span is converted to line/column numbers.
func (c *resolverContext) EmitDiagnostic(code string, severity Severity, mod *module.Module, span types.Span, message string) {
	if !c.diagConfig.ShouldReport(code, severity) {
		return
	}
	var moduleName string
	var line, col int
	if mod != nil {
		moduleName = mod.Name
		line, col = module.LineColFromLineTable(mod.LineTable, span)
	}
	c.diagnostics = append(c.diagnostics, Diagnostic{
		Severity: severity,
		Code:     code,
		Message:  message,
		Module:   moduleName,
		Line:     line,
		Column:   col,
	})
}

// Diagnostics returns all diagnostics collected during resolution.
func (c *resolverContext) Diagnostics() []Diagnostic {
	return c.diagnostics
}

// DiagnosticConfig returns the active strictness and filtering configuration.
func (c *resolverContext) DiagnosticConfig() DiagnosticConfig {
	return c.diagConfig
}

// TypeCount returns the total number of registered types across all modules.
func (c *resolverContext) TypeCount() int {
	n := 0
	for _, symbols := range c.ModuleSymbolToType {
		n += len(symbols)
	}
	return n
}

// logCycles logs detected dependency cycles at trace level.
func logCycles(ctx *resolverContext, cycles [][]graph.Symbol, msg string) {
	if len(cycles) == 0 || !ctx.TraceEnabled() {
		return
	}
	for _, cycle := range cycles {
		names := make([]string, len(cycle))
		for i, s := range cycle {
			names[i] = s.Module + "::" + s.Name
		}
		ctx.Trace(msg, slog.Any("cycle", names))
	}
}

// recordUnresolved appends an entry to a typed slice and emits a diagnostic.
func recordUnresolved[T any](c *resolverContext, list *[]T, entry T, mod *module.Module, span types.Span, code, msg string) {
	*list = append(*list, entry)
	c.EmitDiagnostic(code, SeverityError, mod, span, msg)
}

// RecordUnresolvedImport tracks a symbol that could not be resolved from its source module.
func (c *resolverContext) RecordUnresolvedImport(importingModule *module.Module, fromModule, symbol, reason string, span types.Span) {
	code := types.DiagImportNotFound
	if reason == reasonModuleNotFound {
		code = types.DiagImportModuleNotFound
	}
	recordUnresolved(c, &c.unresolvedImports, unresolvedImport{
		importingModule: importingModule,
		fromModule:      fromModule,
		symbol:          symbol,
		reason:          reason,
		span:            span,
	}, importingModule, span, code, fmt.Sprintf("unresolved import: %s from %s (%s)", symbol, fromModule, reason))
}

// RecordUnresolvedType tracks a type definition whose parent type could not be found.
func (c *resolverContext) RecordUnresolvedType(mod *module.Module, referrer, referenced string, span types.Span) {
	recordUnresolved(c, &c.unresolvedTypes, unresolvedType{
		module: mod, referrer: referrer, referenced: referenced, span: span,
	}, mod, span, types.DiagTypeUnknown, fmt.Sprintf("unresolved type: %s references unknown type %s", referrer, referenced))
}

// RecordUnresolvedOid tracks an OID definition whose parent component could not be resolved.
func (c *resolverContext) RecordUnresolvedOid(mod *module.Module, defName, component string, span types.Span) {
	recordUnresolved(c, &c.unresolvedOids, unresolvedOid{
		module: mod, definition: defName, component: component, span: span,
	}, mod, span, types.DiagOidOrphan, fmt.Sprintf("unresolved OID: %s references unknown parent %s", defName, component))
}

// RecordUnresolvedIndex tracks a row's INDEX entry that references a missing object.
func (c *resolverContext) RecordUnresolvedIndex(mod *module.Module, row, indexObject string, span types.Span) {
	recordUnresolved(c, &c.unresolvedIndexes, unresolvedIndex{
		module: mod, row: row, indexObject: indexObject, span: span,
	}, mod, span, types.DiagIndexUnresolved, fmt.Sprintf("unresolved INDEX: %s references unknown object %s", row, indexObject))
}

// RecordUnresolvedNotificationObject tracks a notification's OBJECTS entry that references a missing object.
func (c *resolverContext) RecordUnresolvedNotificationObject(mod *module.Module, notification, object string, span types.Span) {
	recordUnresolved(c, &c.unresolvedNotifObjects, unresolvedNotifObject{
		module: mod, notification: notification, object: object, span: span,
	}, mod, span, types.DiagObjectsUnresolved, fmt.Sprintf("unresolved OBJECTS: %s references unknown object %s", notification, object))
}

// DropModules releases parsed module data to free memory after resolution completes.
func (c *resolverContext) DropModules() {
	c.Modules = nil
	c.ModuleIndex = nil
	c.ModuleDefNames = nil
	c.ModuleOidDefNames = nil
}

func addUnresolved(m *Mib, kind UnresolvedKind, symbol string, mod *module.Module) {
	modName := ""
	if mod != nil {
		modName = mod.Name
	}
	m.addUnresolved(UnresolvedRef{
		Kind:   kind,
		Symbol: symbol,
		Module: modName,
	})
}

// FinalizeUnresolved copies collected unresolved references and diagnostics into the Mib builder.
func (c *resolverContext) FinalizeUnresolved() {
	for _, u := range c.unresolvedImports {
		addUnresolved(c.Mib, UnresolvedImport, u.symbol, u.importingModule)
	}
	for _, u := range c.unresolvedTypes {
		addUnresolved(c.Mib, UnresolvedType, u.referenced, u.module)
	}
	for _, u := range c.unresolvedOids {
		addUnresolved(c.Mib, UnresolvedOID, u.component, u.module)
	}
	for _, u := range c.unresolvedIndexes {
		addUnresolved(c.Mib, UnresolvedIndex, u.indexObject, u.module)
	}
	for _, u := range c.unresolvedNotifObjects {
		addUnresolved(c.Mib, UnresolvedNotificationObject, u.object, u.module)
	}

	for _, d := range c.diagnostics {
		c.Mib.addDiagnostic(d)
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
