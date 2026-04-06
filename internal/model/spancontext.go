package model

// SpanContextKind classifies what construct a byte offset falls within.
type SpanContextKind int

const (
	SpanContextNone       SpanContextKind = iota
	SpanContextDefinition                 // offset is on a definition name
	SpanContextImport                     // offset is on an imported symbol
	SpanContextOidRef                     // offset is on a symbolic OID reference
	SpanContextSyntax                     // offset is on a SYNTAX type reference
)

// SpanContext describes what construct a byte offset falls within in a module.
type SpanContext struct {
	Kind   SpanContextKind
	Name   string // the symbol/identifier name at this position
	Module string // source module name (for imports: the FROM module)
}

// SpanContext classifies what is at the given byte offset in this module.
// Returns SpanContext with Kind == SpanContextNone if the offset does not
// fall within any recognized span.
//
// Priority order: imports, OID refs, syntax refs, definition names.
func (m *Module) SpanContext(offset ByteOffset) SpanContext {
	if m.base {
		return SpanContext{}
	}

	// Check imports first (most specific for go-to-definition).
	for _, imp := range m.rawImports {
		if imp.Span.Contains(offset) {
			return SpanContext{
				Kind:   SpanContextImport,
				Name:   imp.Symbol,
				Module: imp.Module,
			}
		}
	}

	// Check OID refs across all entity types.
	if sc := checkEntityOidRefs(m.objects, offset); sc.Kind != SpanContextNone {
		return sc
	}
	if sc := checkEntityOidRefs(m.notifications, offset); sc.Kind != SpanContextNone {
		return sc
	}
	if sc := checkEntityOidRefs(m.groups, offset); sc.Kind != SpanContextNone {
		return sc
	}
	if sc := checkEntityOidRefs(m.compliances, offset); sc.Kind != SpanContextNone {
		return sc
	}
	if sc := checkEntityOidRefs(m.capabilities, offset); sc.Kind != SpanContextNone {
		return sc
	}

	// Check SYNTAX spans on objects and types.
	for _, obj := range m.objects {
		if obj.syntaxSpan.Contains(offset) {
			typeName := ""
			if obj.typ != nil {
				typeName = obj.typ.Name()
			}
			return SpanContext{Kind: SpanContextSyntax, Name: typeName}
		}
	}
	for _, td := range m.types {
		if td.syntaxSpan.Contains(offset) {
			parentName := ""
			if td.Parent() != nil {
				parentName = td.Parent().Name()
			}
			return SpanContext{Kind: SpanContextSyntax, Name: parentName}
		}
	}

	// Check definition name spans.
	for _, obj := range m.objects {
		if obj.span.Contains(offset) {
			return SpanContext{Kind: SpanContextDefinition, Name: obj.name, Module: m.name}
		}
	}
	for _, td := range m.types {
		if td.span.Contains(offset) {
			return SpanContext{Kind: SpanContextDefinition, Name: td.Name(), Module: m.name}
		}
	}
	for _, n := range m.notifications {
		if n.span.Contains(offset) {
			return SpanContext{Kind: SpanContextDefinition, Name: n.name, Module: m.name}
		}
	}
	for _, g := range m.groups {
		if g.span.Contains(offset) {
			return SpanContext{Kind: SpanContextDefinition, Name: g.name, Module: m.name}
		}
	}
	for _, c := range m.compliances {
		if c.span.Contains(offset) {
			return SpanContext{Kind: SpanContextDefinition, Name: c.name, Module: m.name}
		}
	}
	for _, c := range m.capabilities {
		if c.span.Contains(offset) {
			return SpanContext{Kind: SpanContextDefinition, Name: c.name, Module: m.name}
		}
	}

	return SpanContext{}
}

// entityWithOidRefs is the subset of entity needed for OID ref checking.
type entityWithOidRefs interface {
	OidRefs() []OidRef
}

func checkEntityOidRefs[T entityWithOidRefs](entities []T, offset ByteOffset) SpanContext {
	for _, e := range entities {
		for _, ref := range e.OidRefs() {
			if ref.Span.Contains(offset) {
				return SpanContext{Kind: SpanContextOidRef, Name: ref.Name}
			}
		}
	}
	return SpanContext{}
}
