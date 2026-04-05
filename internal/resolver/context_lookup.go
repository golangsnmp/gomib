package resolver

import (
	"github.com/golangsnmp/gomib/internal/model"
	"github.com/golangsnmp/gomib/internal/module"
)

// Lookup naming convention:
//
//	lookup*              bounded scope: own module + 1 import hop
//	lookupDirect         single module only, no import traversal
//	lookup*Global        iterate all modules in registration order
//	lookup*ByModuleName  iterate all versions of a named module
//	resolve*             lookup + fallback strategy (well-known types, global scan)

// lookupInScope looks up a symbol in the module's own symbols, then
// follows a single import hop. importSources entries are expected to already
// be transitively resolved to the defining module by resolveTransitiveImports.
// If onImportHit is non-nil, it is called when a symbol is resolved via import.
func lookupInScope[T any](
	mod *module.Module,
	name string,
	symbols symbolTable[T],
	imports symbolTable[*module.Module],
	onImportHit func(*module.Module, string),
) (T, bool) {
	var zero T

	if val, ok := symbols.get(mod, name); ok {
		return val, true
	}

	if source, ok := imports.get(mod, name); ok {
		if val, ok := symbols.get(source, name); ok {
			if onImportHit != nil {
				onImportHit(mod, name)
			}
			return val, true
		}
	}

	return zero, false
}

// lookupNodeByModuleName resolves a node across all versions of a named module.
func (c *resolverContext) lookupNodeByModuleName(moduleName, name string) (*model.Node, bool) {
	candidates := c.moduleIndex[moduleName]
	for _, mod := range candidates {
		if node, ok := c.lookupNode(mod, name); ok {
			return node, true
		}
	}
	return nil, false
}

// lookupNodeGlobal searches all modules for a node with the given name.
// Iterates in module-list order for deterministic results.
func (c *resolverContext) lookupNodeGlobal(name string) (*model.Node, bool) {
	for _, mod := range c.modules {
		if node, ok := c.nodeSymbols.get(mod, name); ok {
			return node, true
		}
	}
	return nil, false
}

// lookupTypeDirect looks up a type directly in a module's symbol table.
func (c *resolverContext) lookupTypeDirect(mod *module.Module, name string) (*model.Type, bool) {
	if mod == nil {
		return nil, false
	}
	return c.typeSymbols.get(mod, name)
}

// lookupNode resolves a node by name, traversing imports from mod.
func (c *resolverContext) lookupNode(mod *module.Module, name string) (*model.Node, bool) {
	return lookupInScope(mod, name, c.nodeSymbols, c.importSources, c.markImportUsed)
}

func (c *resolverContext) lookupType(mod *module.Module, name string) (*model.Type, bool) {
	return lookupInScope(mod, name, c.typeSymbols, c.importSources, c.markImportUsed)
}

// lookupObject finds the resolved Object for a name within a
// specific module's scope. During resolution, module-scoped lookups are the
// correct approach since node.Object() returns the globally preferred object
// which may belong to a different module.
func (c *resolverContext) lookupObject(mod *module.Module, name string) *model.Object {
	// Check the current module first.
	if resolved := c.moduleToResolved[mod]; resolved != nil {
		if obj := resolved.Object(name); obj != nil {
			return obj
		}
	}
	// Check imported modules.
	if source, ok := c.importSources.get(mod, name); ok {
		if resolved := c.moduleToResolved[source]; resolved != nil {
			if obj := resolved.Object(name); obj != nil {
				c.markImportUsed(mod, name)
				return obj
			}
		}
	}
	return nil
}

// resolveType searches for a type by name, trying well-known modules first.
// Beyond those deterministic/constrained sets, global search is only enabled
// in permissive mode.
func (c *resolverContext) resolveType(name string) (*model.Type, bool) {
	if t, ok := c.tryWellKnownTypeFallbacks(name); ok {
		return t, true
	}

	if !c.strictness.AllowGlobalFallbacks() {
		return nil, false
	}

	// Permissive only: scan all modules for unknown types.
	for _, mod := range c.modules {
		if t, ok := c.typeSymbols.get(mod, name); ok {
			return t, true
		}
	}
	return nil, false
}

// resolveTypeForModule resolves a type by name, traversing imports from mod.
// Falls back to well-known base modules when constrained fallbacks are enabled.
func (c *resolverContext) resolveTypeForModule(mod *module.Module, name string) (*model.Type, bool) {
	if t, ok := c.lookupType(mod, name); ok {
		return t, true
	}
	return c.tryWellKnownTypeFallbacks(name)
}

// resolveObject resolves a name to its Object, trying module-scoped
// lookup first, then global fallback if strictness allows. Returns the
// object and whether the name resolved to a node at all (needed for
// diagnostic decisions when the object is nil).
func (c *resolverContext) resolveObject(mod *module.Module, name string) (obj *model.Object, nodeFound bool) {
	if _, ok := c.lookupNode(mod, name); ok {
		return c.lookupObject(mod, name), true
	}
	if c.ResolverStrictness().AllowGlobalFallbacks() {
		if node, ok := c.lookupNodeGlobal(name); ok {
			return node.Object(), true
		}
	}
	return nil, false
}
