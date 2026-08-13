package resolver

import (
	"log/slog"

	"github.com/golangsnmp/gomib/internal/graph"
	"github.com/golangsnmp/gomib/internal/model"
	"github.com/golangsnmp/gomib/internal/module"
	"github.com/golangsnmp/gomib/internal/types"
)

// Unresolved reference reason strings used by recordUnresolved call sites.
const (
	reasonModuleNotFound     = "module_not_found"
	reasonSymbolNotExported  = "symbol_not_exported"
	reasonUnknownType        = "unknown_type"
	reasonDependencyCycle    = "dependency_cycle"
	reasonUnknownParent      = "unknown_parent"
	reasonTrapNumberOverflow = "trap_number_overflow"
	reasonRecursiveOid       = "recursive_oid"
	reasonUnknownIndex       = "unknown_index_object"
	reasonUnknownObject      = "unknown_object"
)

// resolverContext holds indices and working state for all resolution phases.
type resolverContext struct {
	mib *model.Mib

	// modules is the working set of modules for resolution. In the normal
	// pipeline this is populated by registerModules (base modules first,
	// then user modules). Test code may set it directly.
	modules []*module.Module

	// moduleIndex maps module name to parsed modules (multiple versions possible).
	moduleIndex map[string][]*module.Module

	// moduleToResolved maps parsed module to resolved module.
	moduleToResolved map[*module.Module]*model.Module

	// resolvedToModule is the reverse of moduleToResolved.
	resolvedToModule map[*model.Module]*module.Module

	// nodeSymbols maps module -> symbol -> Node for OID lookups.
	nodeSymbols symbolTable[*model.Node]

	// importSources maps module -> symbol -> source module for import chain traversal.
	importSources symbolTable[*module.Module]

	// typeSymbols maps module -> symbol -> Type for type lookups.
	typeSymbols symbolTable[*model.Type]

	// defNames caches definition names per module for import resolution.
	defNames symbolTable[struct{}]

	// oidDefNames caches names of definitions that have OIDs, per module.
	// Used by findOidDefiningModule to avoid O(n) linear scans.
	oidDefNames symbolTable[struct{}]

	// snmpv2SMIModule is the SNMPv2-SMI base module (for primitive types).
	snmpv2SMIModule *module.Module

	// rfc1155SMIModule is the RFC1155-SMI base module (for SMIv1 types).
	rfc1155SMIModule *module.Module

	// snmpv2TCModule is the SNMPv2-TC module (for standard textual conventions).
	snmpv2TCModule *module.Module

	// usedImports tracks which imported symbols are actually referenced during resolution.
	// Populated by markImportUsed, consumed by checkUnusedImports.
	usedImports symbolTable[struct{}]

	// Diagnostic configuration and collection
	strictness  model.ResolverStrictness
	diagConfig  model.DiagnosticConfig
	diagnostics []model.Diagnostic

	types.Logger
}

func newResolverContext(logger *slog.Logger, strictness model.ResolverStrictness, diagConfig model.DiagnosticConfig) *resolverContext {
	return &resolverContext{
		mib:              model.NewMib(),
		moduleIndex:      make(map[string][]*module.Module),
		moduleToResolved: make(map[*module.Module]*model.Module),
		resolvedToModule: make(map[*model.Module]*module.Module),
		nodeSymbols:      make(symbolTable[*model.Node]),
		importSources:    make(symbolTable[*module.Module]),
		typeSymbols:      make(symbolTable[*model.Type]),
		defNames:         make(symbolTable[struct{}]),
		oidDefNames:      make(symbolTable[struct{}]),
		usedImports:      make(symbolTable[struct{}]),
		strictness:       strictness,
		diagConfig:       diagConfig,
		Logger:           types.Logger{L: logger},
	}
}

// markImportUsed records that an imported symbol was referenced during resolution.
func (c *resolverContext) markImportUsed(mod *module.Module, name string) {
	c.usedImports.set(mod, name, struct{}{})
}

// registerImport maps a symbol in importingModule to its source module.
// If the symbol is already registered, the existing registration is kept
// (first-writer-wins) to avoid nondeterminism from map iteration order
// when a MIB imports the same symbol from multiple modules.
func (c *resolverContext) registerImport(importingModule *module.Module, symbol string, sourceModule *module.Module) {
	if existing, ok := c.importSources.get(importingModule, symbol); ok {
		if c.TraceEnabled() {
			c.Trace("import already registered, keeping first",
				slog.String("module", importingModule.Name),
				slog.String("symbol", symbol),
				slog.String("existing", existing.Name),
				slog.String("ignored", sourceModule.Name))
		}
		return
	}
	c.importSources.set(importingModule, symbol, sourceModule)
}

// registerModuleNodeSymbol binds a symbol name to a node within a module scope.
func (c *resolverContext) registerModuleNodeSymbol(mod *module.Module, symbol string, node *model.Node) {
	if c.nodeSymbols.has(mod, symbol) && c.TraceEnabled() {
		c.Trace("overwriting node symbol registration",
			slog.String("module", mod.Name),
			slog.String("symbol", symbol))
	}
	c.nodeSymbols.set(mod, symbol, node)
}

// registerModuleTypeSymbol binds a symbol name to a type within a module scope.
func (c *resolverContext) registerModuleTypeSymbol(mod *module.Module, name string, t *model.Type) {
	c.typeSymbols.set(mod, name, t)
}

// EmitDiagnostic records a diagnostic, filtered by the current config's retention rules.
// If mod is non-nil and has a line table, the span is converted to line/column numbers.
func (c *resolverContext) EmitDiagnostic(code string, mod *module.Module, span types.Span, message string) {
	if !c.diagConfig.ShouldRetain(code) {
		return
	}
	var moduleName string
	var line, col, endLine, endCol int
	if mod != nil {
		moduleName = mod.Name
		line, col, endLine, endCol = types.LineColRangeFromTable(mod.LineTable, span)
	}
	c.diagnostics = append(c.diagnostics, model.Diagnostic{
		Severity:  c.diagConfig.EffectiveSeverity(code),
		Code:      code,
		Message:   message,
		Module:    moduleName,
		Line:      line,
		Column:    col,
		EndLine:   endLine,
		EndColumn: endCol,
	})
}

// Diagnostics returns all diagnostics collected during resolution.
func (c *resolverContext) Diagnostics() []model.Diagnostic {
	return c.diagnostics
}

// DiagnosticConfig returns the active strictness and filtering configuration.
func (c *resolverContext) DiagnosticConfig() model.DiagnosticConfig {
	return c.diagConfig
}

// ResolverStrictness returns the active resolver strictness policy.
func (c *resolverContext) ResolverStrictness() model.ResolverStrictness {
	return c.strictness
}

// TypeCount returns the total number of registered types across all modules.
func (c *resolverContext) TypeCount() int {
	n := 0
	for _, symbols := range c.typeSymbols {
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

func modName(mod *module.Module) string {
	if mod == nil {
		return ""
	}
	return mod.Name
}

// recordUnresolved emits a diagnostic and tracks an unresolved reference.
// All resolution failures that should appear in Mib.Unresolved() go through
// this single method to keep the two stores (diagnostics + unresolved list)
// in sync.
func (c *resolverContext) recordUnresolved(code string, mod *module.Module, span types.Span, message string, ref model.UnresolvedRef) {
	c.EmitDiagnostic(code, mod, span, message)
	model.AddMibUnresolved(c.mib, ref)
}

// FinalizeDiagnostics copies collected diagnostics into the Mib.
func (c *resolverContext) FinalizeDiagnostics() {
	for i := range c.diagnostics {
		model.AddMibDiagnostic(c.mib, &c.diagnostics[i])
	}
}
