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
	mib *Mib

	modules []*module.Module

	// moduleIndex maps module name to parsed modules (multiple versions possible).
	moduleIndex map[string][]*module.Module

	// moduleToResolved maps parsed module to resolved module.
	moduleToResolved map[*module.Module]*Module

	// resolvedToModule is the reverse of moduleToResolved.
	resolvedToModule map[*Module]*module.Module

	// moduleSymbolToNode maps module -> symbol -> Node for OID lookups.
	moduleSymbolToNode map[*module.Module]map[string]*Node

	// moduleImports maps module -> symbol -> source module for import chain traversal.
	moduleImports map[*module.Module]map[string]*module.Module

	// moduleSymbolToType maps module -> symbol -> Type for type lookups.
	moduleSymbolToType map[*module.Module]map[string]*Type

	// moduleDefNames caches definition names per module for import resolution.
	moduleDefNames map[*module.Module]map[string]struct{}

	// moduleOidDefNames caches names of definitions that have OIDs, per module.
	// Used by findOidDefiningModule to avoid O(n) linear scans.
	moduleOidDefNames map[*module.Module]map[string]struct{}

	// snmpv2SMIModule is the SNMPv2-SMI base module (for primitive types).
	snmpv2SMIModule *module.Module

	// rfc1155SMIModule is the RFC1155-SMI base module (for SMIv1 types).
	rfc1155SMIModule *module.Module

	// snmpv2TCModule is the SNMPv2-TC module (for standard textual conventions).
	snmpv2TCModule *module.Module

	// Unresolved references collected during resolution
	unresolvedImports      []unresolvedImport
	unresolvedTypes        []unresolvedType
	unresolvedOids         []unresolvedOid
	unresolvedIndexes      []unresolvedIndex
	unresolvedNotifObjects []unresolvedNotifObject

	// usedImports tracks which imported symbols are actually referenced during resolution.
	// Populated by markImportUsed, consumed by checkUnusedImports.
	usedImports map[*module.Module]map[string]struct{}

	// Diagnostic configuration and collection
	strictness  ResolverStrictness
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

func newResolverContext(mods []*module.Module, logger *slog.Logger, strictness ResolverStrictness, diagConfig DiagnosticConfig) *resolverContext {
	n := len(mods)
	ctx := &resolverContext{
		mib:                newMib(),
		modules:            mods,
		moduleIndex:        make(map[string][]*module.Module, n),
		moduleToResolved:   make(map[*module.Module]*Module, n),
		resolvedToModule:   make(map[*Module]*module.Module, n),
		moduleSymbolToNode: make(map[*module.Module]map[string]*Node, n),
		moduleImports:      make(map[*module.Module]map[string]*module.Module, n),
		moduleSymbolToType: make(map[*module.Module]map[string]*Type, n),
		moduleDefNames:     make(map[*module.Module]map[string]struct{}, n),
		moduleOidDefNames:  make(map[*module.Module]map[string]struct{}, n),
		usedImports:        make(map[*module.Module]map[string]struct{}, n),
		strictness:         strictness,
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
		ctx.moduleOidDefNames[mod] = oidDefs
	}
	return ctx
}

// LookupNodeForModule resolves a node by name, traversing imports from mod.
func (c *resolverContext) LookupNodeForModule(mod *module.Module, name string) (*Node, bool) {
	return c.lookupNodeInModuleScope(mod, name)
}

// LookupNodeInModule resolves a node across all versions of a named module.
func (c *resolverContext) LookupNodeInModule(moduleName, name string) (*Node, bool) {
	candidates := c.moduleIndex[moduleName]
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
	for _, mod := range c.modules {
		if symbols := c.moduleSymbolToNode[mod]; symbols != nil {
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
	if symbols := c.moduleSymbolToType[mod]; symbols != nil {
		if t, ok := symbols[name]; ok {
			return t, true
		}
	}
	return nil, false
}

// findWellKnownModuleForType returns the well-known base module expected to
// define the given type name. ASN.1 primitives are always checked; other
// well-known modules (SMI globals, SMIv1 types, SNMPv2-TC) require constrained
// fallbacks (Normal/Permissive). Returns nil if no well-known module matches.
func (c *resolverContext) findWellKnownModuleForType(name string) *module.Module {
	cls := wellKnownTypes[name]

	// RFC-compliant: ASN.1 primitives are always available
	if cls == typeClassASN1Primitive {
		return c.snmpv2SMIModule
	}

	if !c.strictness.AllowConstrainedFallbacks() {
		return nil
	}

	// Normal/Permissive: well-known modules for SMI globals, SMIv1 types, SNMPv2-TC
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
// base modules (permissive only) for a type by name.
func (c *resolverContext) tryWellKnownTypeFallbacks(name string) (*Type, bool) {
	if m := c.findWellKnownModuleForType(name); m != nil {
		return c.lookupTypeInModule(m, name)
	}
	return nil, false
}

// LookupType searches for a type by name, trying well-known modules first.
// Beyond well-known sets, global search is only enabled in permissive mode.
func (c *resolverContext) LookupType(name string) (*Type, bool) {
	if t, ok := c.tryWellKnownTypeFallbacks(name); ok {
		return t, true
	}

	if !c.strictness.AllowGlobalFallbacks() {
		return nil, false
	}

	// Permissive only: scan all modules for unknown types.
	for _, mod := range c.modules {
		if symbols := c.moduleSymbolToType[mod]; symbols != nil {
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
// If onImportHit is non-nil, it is called when a symbol is resolved via import.
func lookupInModuleScope[T any](
	mod *module.Module,
	name string,
	getSymbols func(*module.Module) map[string]T,
	getImports func(*module.Module) map[string]*module.Module,
	onImportHit func(*module.Module, string),
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
					if onImportHit != nil {
						onImportHit(mod, name)
					}
					return val, true
				}
			}
		}
	}

	return zero, false
}

func (c *resolverContext) lookupNodeInModuleScope(mod *module.Module, name string) (*Node, bool) {
	return lookupInModuleScope(mod, name,
		func(m *module.Module) map[string]*Node { return c.moduleSymbolToNode[m] },
		func(m *module.Module) map[string]*module.Module { return c.moduleImports[m] },
		c.markImportUsed,
	)
}

func (c *resolverContext) lookupTypeInModuleScope(mod *module.Module, name string) (*Type, bool) {
	return lookupInModuleScope(mod, name,
		func(m *module.Module) map[string]*Type { return c.moduleSymbolToType[m] },
		func(m *module.Module) map[string]*module.Module { return c.moduleImports[m] },
		c.markImportUsed,
	)
}

// lookupObjectInModuleScope finds the resolved Object for a name using the
// correct module's object. This avoids using node.Object() which may return
// an object from a different module when multiple modules define the same OID.
func (c *resolverContext) lookupObjectInModuleScope(mod *module.Module, name string) *Object {
	// Check the current module first.
	if resolved := c.moduleToResolved[mod]; resolved != nil {
		if obj := resolved.Object(name); obj != nil {
			return obj
		}
	}
	// Check imported modules.
	if imports := c.moduleImports[mod]; imports != nil {
		if source, ok := imports[name]; ok {
			if resolved := c.moduleToResolved[source]; resolved != nil {
				if obj := resolved.Object(name); obj != nil {
					c.markImportUsed(mod, name)
					return obj
				}
			}
		}
	}
	return nil
}

// markImportUsed records that an imported symbol was referenced during resolution.
func (c *resolverContext) markImportUsed(mod *module.Module, name string) {
	used := c.usedImports[mod]
	if used == nil {
		used = make(map[string]struct{})
		c.usedImports[mod] = used
	}
	used[name] = struct{}{}
}

// registerImport maps a symbol in importingModule to its source module.
// If the symbol is already registered, the existing registration is kept
// (first-writer-wins) to avoid nondeterminism from map iteration order
// when a MIB imports the same symbol from multiple modules.
func (c *resolverContext) registerImport(importingModule *module.Module, symbol string, sourceModule *module.Module) {
	imports := c.moduleImports[importingModule]
	if imports == nil {
		imports = make(map[string]*module.Module)
		c.moduleImports[importingModule] = imports
	}
	if _, exists := imports[symbol]; exists {
		if c.TraceEnabled() {
			c.Trace("import already registered, keeping first",
				slog.String("module", importingModule.Name),
				slog.String("symbol", symbol),
				slog.String("existing", imports[symbol].Name),
				slog.String("ignored", sourceModule.Name))
		}
		return
	}
	imports[symbol] = sourceModule
}

// registerModuleNodeSymbol binds a symbol name to a node within a module scope.
func (c *resolverContext) registerModuleNodeSymbol(mod *module.Module, symbol string, node *Node) {
	symbols := c.moduleSymbolToNode[mod]
	if symbols == nil {
		symbols = make(map[string]*Node)
		c.moduleSymbolToNode[mod] = symbols
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
	symbols := c.moduleSymbolToType[mod]
	if symbols == nil {
		symbols = make(map[string]*Type)
		c.moduleSymbolToType[mod] = symbols
	}
	symbols[name] = t
}

// EmitDiagnostic records a diagnostic, filtered by the current config's severity and code rules.
// If mod is non-nil and has a line table, the span is converted to line/column numbers.
func (c *resolverContext) EmitDiagnostic(code string, mod *module.Module, span types.Span, message string) {
	sev := types.SeverityForCode(code)
	if !c.diagConfig.ShouldReport(code, sev) {
		return
	}
	var moduleName string
	var line, col int
	if mod != nil {
		moduleName = mod.Name
		line, col = module.LineColFromLineTable(mod.LineTable, span)
	}
	c.diagnostics = append(c.diagnostics, Diagnostic{
		Severity: sev,
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

// ResolverStrictness returns the active resolver strictness policy.
func (c *resolverContext) ResolverStrictness() ResolverStrictness {
	return c.strictness
}

// TypeCount returns the total number of registered types across all modules.
func (c *resolverContext) TypeCount() int {
	n := 0
	for _, symbols := range c.moduleSymbolToType {
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
func recordUnresolved[T any](ctx *resolverContext, list *[]T, entry T, mod *module.Module, span types.Span, code, msg string) {
	*list = append(*list, entry)
	ctx.EmitDiagnostic(code, mod, span, msg)
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
	}, importingModule, span, code, fmt.Sprintf("unresolved import: %q from %q (%s)", symbol, fromModule, reason))
}

// RecordUnresolvedType tracks a type definition whose parent type could not be found.
func (c *resolverContext) RecordUnresolvedType(mod *module.Module, referrer, referenced string, span types.Span) {
	recordUnresolved(c, &c.unresolvedTypes, unresolvedType{
		module: mod, referrer: referrer, referenced: referenced, span: span,
	}, mod, span, types.DiagTypeUnknown, fmt.Sprintf("unresolved type: %q references unknown type %q", referrer, referenced))
}

// RecordUnresolvedOid tracks an OID definition whose parent component could not be resolved.
func (c *resolverContext) RecordUnresolvedOid(mod *module.Module, defName, component string, span types.Span) {
	recordUnresolved(c, &c.unresolvedOids, unresolvedOid{
		module: mod, definition: defName, component: component, span: span,
	}, mod, span, types.DiagOidOrphan, fmt.Sprintf("unresolved OID: %q references unknown parent %q", defName, component))
}

// RecordUnresolvedIndex tracks a row's INDEX entry that references a missing object.
func (c *resolverContext) RecordUnresolvedIndex(mod *module.Module, row, indexObject string, span types.Span) {
	recordUnresolved(c, &c.unresolvedIndexes, unresolvedIndex{
		module: mod, row: row, indexObject: indexObject, span: span,
	}, mod, span, types.DiagIndexUnresolved, fmt.Sprintf("unresolved INDEX: %q references unknown object %q", row, indexObject))
}

// RecordUnresolvedNotificationObject tracks a notification's OBJECTS entry that references a missing object.
func (c *resolverContext) RecordUnresolvedNotificationObject(mod *module.Module, notification, object string, span types.Span) {
	recordUnresolved(c, &c.unresolvedNotifObjects, unresolvedNotifObject{
		module: mod, notification: notification, object: object, span: span,
	}, mod, span, types.DiagObjectsUnresolved, fmt.Sprintf("unresolved OBJECTS: %q references unknown object %q", notification, object))
}

// DropModules releases parsed module data to free memory after resolution completes.
func (c *resolverContext) DropModules() {
	c.modules = nil
	c.moduleIndex = nil
	c.moduleDefNames = nil
	c.moduleOidDefNames = nil
}

func addUnresolved(m *Mib, kind UnresolvedKind, symbol, reason string, mod *module.Module) {
	modName := ""
	if mod != nil {
		modName = mod.Name
	}
	m.addUnresolved(UnresolvedRef{
		Kind:   kind,
		Symbol: symbol,
		Module: modName,
		Reason: reason,
	})
}

// FinalizeUnresolved copies collected unresolved references and diagnostics into the Mib builder.
func (c *resolverContext) FinalizeUnresolved() {
	for _, u := range c.unresolvedImports {
		addUnresolved(c.mib, UnresolvedImport, u.symbol, u.reason, u.importingModule)
	}
	for _, u := range c.unresolvedTypes {
		addUnresolved(c.mib, UnresolvedType, u.referenced, "unknown_type", u.module)
	}
	for _, u := range c.unresolvedOids {
		addUnresolved(c.mib, UnresolvedOID, u.component, "unknown_parent", u.module)
	}
	for _, u := range c.unresolvedIndexes {
		addUnresolved(c.mib, UnresolvedIndex, u.indexObject, "unknown_index_object", u.module)
	}
	for _, u := range c.unresolvedNotifObjects {
		addUnresolved(c.mib, UnresolvedNotificationObject, u.object, "unknown_object", u.module)
	}

	for _, d := range c.diagnostics {
		c.mib.addDiagnostic(d)
	}
}

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
