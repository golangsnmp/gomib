package model

// RefKind classifies a symbol reference.
type RefKind int

const (
	RefImport   RefKind = iota + 1 // IMPORTS clause reference
	RefOidValue                     // OID value assignment parent reference
	RefIndex                        // INDEX clause reference
)

// SymbolRef is a source-level reference to a named symbol.
type SymbolRef struct {
	Kind   RefKind
	Module *Module
	Span   Span
}

// SymbolReferences returns all source-level references to the named symbol
// across all loaded modules. This includes import references, OID parent
// references, and INDEX clause references. The definition site itself is
// not included.
func (m *Mib) SymbolReferences(name string) []SymbolRef {
	var refs []SymbolRef

	for _, mod := range m.modules {
		if mod.base {
			continue
		}

		// Import references.
		for _, imp := range mod.rawImports {
			if imp.Symbol == name {
				refs = append(refs, SymbolRef{
					Kind:   RefImport,
					Module: mod,
					Span:   imp.Span,
				})
			}
		}

		// OID parent references.
		refs = collectEntityRefs(refs, mod, mod.objects, name)
		refs = collectEntityRefs(refs, mod, mod.notifications, name)
		refs = collectEntityRefs(refs, mod, mod.groups, name)
		refs = collectEntityRefs(refs, mod, mod.compliances, name)
		refs = collectEntityRefs(refs, mod, mod.capabilities, name)

		// INDEX clause references.
		for _, obj := range mod.objects {
			for _, idx := range obj.index {
				if idx.Object != nil && idx.Object.Name() == name {
					refs = append(refs, SymbolRef{
						Kind:   RefIndex,
						Module: mod,
						Span:   idx.Span,
					})
				}
			}
		}
	}

	return refs
}

func collectEntityRefs[T entityWithOidRefs](refs []SymbolRef, mod *Module, entities []T, name string) []SymbolRef {
	for _, e := range entities {
		for _, ref := range e.OidRefs() {
			if ref.Name == name {
				refs = append(refs, SymbolRef{
					Kind:   RefOidValue,
					Module: mod,
					Span:   ref.Span,
				})
			}
		}
	}
	return refs
}
