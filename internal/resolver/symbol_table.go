package resolver

import "github.com/golangsnmp/gomib/internal/module"

// symbolTable is a per-module symbol map used throughout resolution.
// The zero value is not usable; create with make(symbolTable[T]).
type symbolTable[T any] map[*module.Module]map[string]T

// get returns the value for a symbol in a module's scope.
// Returns zero value and false if the module or symbol is not present.
func (st symbolTable[T]) get(mod *module.Module, name string) (T, bool) {
	var zero T
	if syms := st[mod]; syms != nil {
		if v, ok := syms[name]; ok {
			return v, true
		}
	}
	return zero, false
}

// set binds a symbol to a value in a module's scope.
// Lazily initializes the inner map on first write.
func (st symbolTable[T]) set(mod *module.Module, name string, val T) {
	syms := st[mod]
	if syms == nil {
		syms = make(map[string]T)
		st[mod] = syms
	}
	syms[name] = val
}

// has reports whether a symbol exists in a module's scope.
func (st symbolTable[T]) has(mod *module.Module, name string) bool {
	_, ok := st.get(mod, name)
	return ok
}

// forModule returns the inner symbol map for a module, or nil if the
// module has no entries. The caller must not modify the returned map.
func (st symbolTable[T]) forModule(mod *module.Module) map[string]T {
	return st[mod]
}
