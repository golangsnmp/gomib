package resolver

import (
	"log/slog"

	"github.com/golangsnmp/gomib/internal/mibimpl"
	"github.com/golangsnmp/gomib/internal/module"
	"github.com/golangsnmp/gomib/internal/types"
	"github.com/golangsnmp/gomib/mib"
)

// ResolverContext holds indices and working state during resolution.
type ResolverContext struct {
	Builder *mibimpl.Builder

	Modules []*module.Module

	// ModuleIndex maps module name to parsed modules (multiple versions possible)
	ModuleIndex map[string][]*module.Module

	// ModuleToResolved maps parsed module to resolved module
	ModuleToResolved map[*module.Module]*mibimpl.Module

	// ModuleSymbolToNode maps module -> symbol -> Node for OID lookups
	ModuleSymbolToNode map[*module.Module]map[string]*mibimpl.Node

	// ModuleImports maps module -> symbol -> source module for import chain traversal
	ModuleImports map[*module.Module]map[string]*module.Module

	// ModuleSymbolToType maps module -> symbol -> Type for type lookups
	ModuleSymbolToType map[*module.Module]map[string]*mibimpl.Type

	// ModuleDefNames caches definition names per module for import resolution
	ModuleDefNames map[*module.Module]map[string]struct{}

	// Snmpv2SMIModule is the SNMPv2-SMI base module (for primitive types)
	Snmpv2SMIModule *module.Module

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

// capacityHints holds pre-computed capacity hints from scanning modules.
type capacityHints struct {
	modules       int
	objects       int
	types         int
	notifications int
	nodes         int
	imports       int
}

// scanCapacityHints scans modules to compute capacity hints.
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

// newResolverContext creates a new context with an optional logger and diagnostic config.
func newResolverContext(mods []*module.Module, logger *slog.Logger, diagConfig mib.DiagnosticConfig) *ResolverContext {
	h := scanCapacityHints(mods)
	return &ResolverContext{
		Builder:            mibimpl.NewBuilder(),
		Modules:            mods,
		ModuleIndex:        make(map[string][]*module.Module, h.modules),
		ModuleToResolved:   make(map[*module.Module]*mibimpl.Module, h.modules),
		ModuleSymbolToNode: make(map[*module.Module]map[string]*mibimpl.Node, h.modules),
		ModuleImports:      make(map[*module.Module]map[string]*module.Module, h.modules),
		ModuleSymbolToType: make(map[*module.Module]map[string]*mibimpl.Type, h.modules),
		ModuleDefNames:     make(map[*module.Module]map[string]struct{}, h.modules),
		diagConfig:         diagConfig,
		Logger:             types.Logger{L: logger},
	}
}

// LookupNodeForModule resolves a node by name in a module scope.
func (c *ResolverContext) LookupNodeForModule(mod *module.Module, name string) (*mibimpl.Node, bool) {
	return c.lookupNodeInModuleScope(mod, name)
}

// LookupNodeInModule resolves a node by module name and symbol.
func (c *ResolverContext) LookupNodeInModule(moduleName, name string) (*mibimpl.Node, bool) {
	candidates := c.ModuleIndex[moduleName]
	for _, mod := range candidates {
		if node, ok := c.LookupNodeForModule(mod, name); ok {
			return node, true
		}
	}
	return nil, false
}

// LookupNodeGlobal looks up a node by name across all modules.
func (c *ResolverContext) LookupNodeGlobal(name string) (*mibimpl.Node, bool) {
	for _, symbols := range c.ModuleSymbolToNode {
		if node, ok := symbols[name]; ok {
			return node, true
		}
	}
	return nil, false
}

// LookupType looks up a type by name globally.
func (c *ResolverContext) LookupType(name string) (*mibimpl.Type, bool) {
	// First try SNMPv2-SMI for primitives
	if c.Snmpv2SMIModule != nil {
		if t, ok := c.LookupTypeForModule(c.Snmpv2SMIModule, name); ok {
			return t, true
		}
	}

	for _, symbols := range c.ModuleSymbolToType {
		if t, ok := symbols[name]; ok {
			return t, true
		}
	}
	return nil, false
}

// LookupTypeForModule looks up a type by name in a module scope.
func (c *ResolverContext) LookupTypeForModule(mod *module.Module, name string) (*mibimpl.Type, bool) {
	// Try the module scope traversal first
	if t, ok := c.lookupTypeInModuleScope(mod, name); ok {
		return t, true
	}

	// ASN.1 primitives and global SMI types (SNMPv2-SMI fallback)
	if c.Snmpv2SMIModule != nil {
		if isASN1Primitive(name) || isSmiGlobalType(name) {
			if symbols := c.ModuleSymbolToType[c.Snmpv2SMIModule]; symbols != nil {
				if t, ok := symbols[name]; ok {
					return t, true
				}
			}
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

// lookupNodeInModuleScope traverses module imports to find a node.
func (c *ResolverContext) lookupNodeInModuleScope(mod *module.Module, name string) (*mibimpl.Node, bool) {
	return lookupInModuleScope(mod, name,
		func(m *module.Module) map[string]*mibimpl.Node { return c.ModuleSymbolToNode[m] },
		func(m *module.Module) map[string]*module.Module { return c.ModuleImports[m] },
	)
}

// lookupTypeInModuleScope traverses module imports to find a type.
func (c *ResolverContext) lookupTypeInModuleScope(mod *module.Module, name string) (*mibimpl.Type, bool) {
	return lookupInModuleScope(mod, name,
		func(m *module.Module) map[string]*mibimpl.Type { return c.ModuleSymbolToType[m] },
		func(m *module.Module) map[string]*module.Module { return c.ModuleImports[m] },
	)
}

// RegisterImport records an import declaration for later lookup.
func (c *ResolverContext) RegisterImport(importingModule *module.Module, symbol string, sourceModule *module.Module) {
	imports := c.ModuleImports[importingModule]
	if imports == nil {
		imports = make(map[string]*module.Module)
		c.ModuleImports[importingModule] = imports
	}
	imports[symbol] = sourceModule
}

// RegisterModuleNodeSymbol registers a module-scoped symbol -> node mapping.
func (c *ResolverContext) RegisterModuleNodeSymbol(mod *module.Module, symbol string, node *mibimpl.Node) {
	symbols := c.ModuleSymbolToNode[mod]
	if symbols == nil {
		symbols = make(map[string]*mibimpl.Node)
		c.ModuleSymbolToNode[mod] = symbols
	}
	symbols[symbol] = node
}

// RegisterModuleTypeSymbol registers a module-scoped symbol -> type mapping.
func (c *ResolverContext) RegisterModuleTypeSymbol(mod *module.Module, name string, t *mibimpl.Type) {
	symbols := c.ModuleSymbolToType[mod]
	if symbols == nil {
		symbols = make(map[string]*mibimpl.Type)
		c.ModuleSymbolToType[mod] = symbols
	}
	symbols[name] = t
}

// EmitDiagnostic records a diagnostic if it should be reported under the current config.
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

// Diagnostics returns all collected diagnostics.
func (c *ResolverContext) Diagnostics() []mib.Diagnostic {
	return c.diagnostics
}

// DiagnosticConfig returns the diagnostic configuration.
func (c *ResolverContext) DiagnosticConfig() mib.DiagnosticConfig {
	return c.diagConfig
}

// RecordUnresolvedImport records an unresolved import.
func (c *ResolverContext) RecordUnresolvedImport(importingModule *module.Module, fromModule, symbol, reason string, span types.Span) {
	c.unresolvedImports = append(c.unresolvedImports, unresolvedImport{
		importingModule: importingModule,
		fromModule:      fromModule,
		symbol:          symbol,
		reason:          reason,
		span:            span,
	})

	// Also emit a diagnostic
	modName := ""
	if importingModule != nil {
		modName = importingModule.Name
	}
	code := "import-not-found"
	if reason == "module not found" {
		code = "import-module-not-found"
	}
	// Note: Line/Column are 0 since we only have byte offsets in Span
	c.EmitDiagnostic(code, mib.SeverityError, modName, 0, 0,
		"unresolved import: "+symbol+" from "+fromModule+" ("+reason+")")
}

// RecordUnresolvedType records an unresolved type reference.
func (c *ResolverContext) RecordUnresolvedType(mod *module.Module, referrer, referenced string, span types.Span) {
	c.unresolvedTypes = append(c.unresolvedTypes, unresolvedType{
		module:     mod,
		referrer:   referrer,
		referenced: referenced,
		span:       span,
	})

	// Also emit a diagnostic
	modName := ""
	if mod != nil {
		modName = mod.Name
	}
	c.EmitDiagnostic("type-unknown", mib.SeverityError, modName, 0, 0,
		"unresolved type: "+referrer+" references unknown type "+referenced)
}

// RecordUnresolvedOid records an unresolved OID component.
func (c *ResolverContext) RecordUnresolvedOid(mod *module.Module, defName, component string, span types.Span) {
	c.unresolvedOids = append(c.unresolvedOids, unresolvedOid{
		module:     mod,
		definition: defName,
		component:  component,
		span:       span,
	})

	// Also emit a diagnostic
	modName := ""
	if mod != nil {
		modName = mod.Name
	}
	c.EmitDiagnostic("oid-orphan", mib.SeverityWarning, modName, 0, 0,
		"unresolved OID: "+defName+" references unknown parent "+component)
}

// RecordUnresolvedIndex records an unresolved index object.
func (c *ResolverContext) RecordUnresolvedIndex(mod *module.Module, row, indexObject string, span types.Span) {
	c.unresolvedIndexes = append(c.unresolvedIndexes, unresolvedIndex{
		module:      mod,
		row:         row,
		indexObject: indexObject,
		span:        span,
	})

	// Also emit a diagnostic
	modName := ""
	if mod != nil {
		modName = mod.Name
	}
	c.EmitDiagnostic("index-unresolved", mib.SeverityError, modName, 0, 0,
		"unresolved INDEX: "+row+" references unknown object "+indexObject)
}

// RecordUnresolvedNotificationObject records an unresolved notification object reference.
func (c *ResolverContext) RecordUnresolvedNotificationObject(mod *module.Module, notification, object string, span types.Span) {
	c.unresolvedNotifObjects = append(c.unresolvedNotifObjects, unresolvedNotifObject{
		module:       mod,
		notification: notification,
		object:       object,
		span:         span,
	})

	// Also emit a diagnostic
	modName := ""
	if mod != nil {
		modName = mod.Name
	}
	c.EmitDiagnostic("objects-unresolved", mib.SeverityWarning, modName, 0, 0,
		"unresolved OBJECTS: "+notification+" references unknown object "+object)
}

// DropModules drops modules to free memory after resolution.
func (c *ResolverContext) DropModules() {
	c.Modules = nil
	c.ModuleIndex = nil
	c.ModuleDefNames = nil
}

// FinalizeUnresolved copies unresolved references and diagnostics to the Mib.
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

	// Copy diagnostics to the Mib
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
