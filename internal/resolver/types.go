package resolver

import (
	"fmt"
	"log/slog"
	"math"

	"github.com/golangsnmp/gomib/internal/graph"
	"github.com/golangsnmp/gomib/internal/model"
	"github.com/golangsnmp/gomib/internal/module"
	"github.com/golangsnmp/gomib/internal/types"
)

// resolveTypes is the type resolution phase entry point.
func resolveTypes(ctx *resolverContext) {
	seedPrimitiveTypes(ctx)
	createUserTypes(ctx)
	resolveTypeBases(ctx)
}

func seedPrimitiveTypes(ctx *resolverContext) {
	if ctx.snmpv2SMIModule == nil {
		return
	}
	mod := ctx.snmpv2SMIModule
	resolved := ctx.moduleToResolved[mod]

	seeded := 0
	seedType := func(name string, base model.BaseType) {
		typ := model.NewType(name)
		model.SetTypeModule(typ, resolved)
		model.SetTypeBase(typ, base)

		model.AddMibType(ctx.mib, typ)
		ctx.registerModuleTypeSymbol(mod, name, typ)
		if resolved != nil {
			model.AddModuleType(resolved, typ)
		}
		seeded++
		if ctx.TraceEnabled() {
			ctx.Trace("seeded primitive type", slog.String("name", name))
		}
	}

	seedType("INTEGER", model.BaseInteger32)
	seedType("OCTET STRING", model.BaseOctetString)
	seedType("OBJECT IDENTIFIER", model.BaseObjectIdentifier)
	seedType("BITS", model.BaseBits)
}

func createUserTypes(ctx *resolverContext) {
	for _, mod := range ctx.modules {
		resolved := ctx.moduleToResolved[mod]

		for _, td := range mod.TypeDefs {
			// Skip SEQUENCE type assignments (e.g., IfEntry ::= SEQUENCE { ... }).
			// These are structural declarations for table rows, not data types.
			if _, isSeq := td.Syntax.(*module.TypeSyntaxSequence); isSeq {
				continue
			}

			base, hasBase := syntaxToBaseType(td.Syntax)
			if td.BaseType != nil {
				base = *td.BaseType
				hasBase = true
			}
			if !hasBase {
				// Type references (e.g., DisplayString) don't have an intrinsic
				// base type. Use model.BaseUnknown as a placeholder; the real base
				// is inherited from the parent during resolveTypeBases.
				base = model.BaseUnknown
				if ctx.TraceEnabled() {
					ctx.Trace("type has no intrinsic base, will inherit from parent",
						slog.String("type", td.Name),
						slog.String("module", mod.Name))
				}
			}

			typ := model.NewType(td.Name)
			model.SetTypeSpan(typ, td.Span)
			model.SetTypeSyntaxSpan(typ, td.Spans.Syntax)
			model.SetTypeModule(typ, resolved)
			model.SetTypeBase(typ, base)
			model.SetTypeIsTC(typ, td.IsTextualConvention)
			model.SetTypeStatus(typ, td.Status)
			model.SetTypeDisplayHint(typ, td.DisplayHint)
			model.SetTypeDescription(typ, td.Description)
			model.SetTypeReference(typ, td.Reference)

			namedValues := extractNamedValues(td.Syntax)
			if base == model.BaseBits {
				model.SetTypeBits(typ, namedValues)
			} else {
				model.SetTypeEnums(typ, namedValues)
			}
			sizes, ranges := extractConstraints(td.Syntax)
			model.SetTypeSizes(typ, sizes)
			model.SetTypeRanges(typ, ranges)

			model.AddMibType(ctx.mib, typ)
			ctx.registerModuleTypeSymbol(mod, td.Name, typ)

			if ctx.TraceEnabled() {
				ctx.Trace("created user type",
					slog.String("name", td.Name),
					slog.String("base", base.String()))
			}

			if resolved != nil {
				model.AddModuleType(resolved, typ)
			}
		}
	}
}

// typeResolutionEntry pairs a parsed type definition with its resolved type.
type typeResolutionEntry struct {
	mod *module.Module
	td  *module.TypeDef
	typ *model.Type
}

func resolveTypeBases(ctx *resolverContext) {
	resolveTypeRefParentsGraph(ctx)
	linkPrimitiveSyntaxParents(ctx)
	linkRFC1213TypesToTCs(ctx)
	inheritBaseTypes(ctx)
}

// resolveTypeRefParentsGraph uses a dependency graph to resolve type parents
// in topological order (single pass).
func resolveTypeRefParentsGraph(ctx *resolverContext) {
	nTypes := ctx.TypeCount()
	entries := make(map[graph.Symbol]typeResolutionEntry, nTypes)
	g := graph.New(nTypes)

	for _, mod := range ctx.modules {
		for _, td := range mod.TypeDefs {
			if _, isSeq := td.Syntax.(*module.TypeSyntaxSequence); isSeq {
				continue
			}
			typ, ok := ctx.resolveTypeForModule(mod, td.Name)
			if !ok {
				continue
			}

			sym := graph.Symbol{Module: mod.Name, Name: td.Name}
			g.AddNode(sym)
			entries[sym] = typeResolutionEntry{mod: mod, td: td, typ: typ}

			if baseName := getTypeRefBaseName(td.Syntax); baseName != "" {
				parentMod := findTypeDefiningModule(ctx, mod, baseName)
				if parentMod != "" {
					parentSym := graph.Symbol{Module: parentMod, Name: baseName}
					g.AddEdge(sym, parentSym)
				}
			}
		}
	}

	order, cycles := g.ResolutionOrder()
	logCycles(ctx, cycles, "type cycle detected")

	if ctx.TraceEnabled() {
		ctx.Trace("type resolution order",
			slog.Int("total", len(order)),
			slog.Int("cycles", len(cycles)))
	}

	resolved := 0
	for _, sym := range order {
		entry, ok := entries[sym]
		if !ok {
			continue // External dependency (primitive or from another module)
		}
		if tryResolveTypeParent(ctx, entry) {
			resolved++
		} else if baseName := getTypeRefBaseName(entry.td.Syntax); baseName != "" {
			ctx.recordUnresolved(types.DiagTypeUnknown, entry.mod, entry.td.Spans.Syntax,
				fmt.Sprintf("unresolved type: %q references unknown type %q", entry.td.Name, baseName),
				model.UnresolvedRef{Kind: model.UnresolvedType, Symbol: baseName, Module: modName(entry.mod), Reason: reasonUnknownType})
		}
	}

	for _, scc := range cycles {
		for _, sym := range scc {
			entry, ok := entries[sym]
			if !ok {
				continue
			}
			baseName := getTypeRefBaseName(entry.td.Syntax)
			if baseName != "" {
				ctx.recordUnresolved(types.DiagTypeUnknown, entry.mod, entry.td.Spans.Syntax,
					fmt.Sprintf("unresolved type: %q references unknown type %q", entry.td.Name, baseName),
					model.UnresolvedRef{Kind: model.UnresolvedType, Symbol: baseName, Module: modName(entry.mod), Reason: reasonUnknownType})
			}
		}
	}

	if ctx.TraceEnabled() {
		ctx.Trace("type resolution complete",
			slog.Int("resolved", resolved),
			slog.Int("cycles", len(cycles)))
	}
}

// findTypeDefiningModule finds the module that defines a type, following imports.
func findTypeDefiningModule(ctx *resolverContext, fromMod *module.Module, typeName string) string {
	if ctx.defNames.has(fromMod, typeName) {
		return fromMod.Name
	}

	if srcMod, ok := ctx.importSources.get(fromMod, typeName); ok {
		return srcMod.Name
	}

	if m := ctx.findWellKnownModuleForType(typeName); m != nil {
		return m.Name
	}

	return ""
}

func tryResolveTypeParent(ctx *resolverContext, entry typeResolutionEntry) bool {
	baseName := getTypeRefBaseName(entry.td.Syntax)
	if baseName == "" {
		return false
	}

	parent, ok := ctx.resolveTypeForModule(entry.mod, baseName)
	if !ok {
		return false
	}

	model.SetTypeParent(entry.typ, parent)
	return true
}

func getTypeRefBaseName(syntax module.TypeSyntax) string {
	switch s := syntax.(type) {
	case *module.TypeSyntaxTypeRef:
		return s.Name
	case *module.TypeSyntaxConstrained:
		if base, ok := s.Base.(*module.TypeSyntaxTypeRef); ok {
			return base.Name
		}
	}
	return ""
}

func getPrimitiveParentName(syntax module.TypeSyntax) string {
	switch s := syntax.(type) {
	case *module.TypeSyntaxOctetString:
		return "OCTET STRING"
	case *module.TypeSyntaxObjectIdentifier:
		return "OBJECT IDENTIFIER"
	case *module.TypeSyntaxIntegerEnum:
		return "INTEGER"
	case *module.TypeSyntaxBits:
		return "BITS"
	case *module.TypeSyntaxConstrained:
		return getPrimitiveParentName(s.Base)
	}
	return ""
}

func linkPrimitiveSyntaxParents(ctx *resolverContext) {
	for _, mod := range ctx.modules {
		for _, td := range mod.TypeDefs {
			primitiveName := getPrimitiveParentName(td.Syntax)
			if primitiveName == "" {
				continue
			}

			typ, ok := ctx.resolveTypeForModule(mod, td.Name)
			if !ok {
				continue
			}
			parent, ok := ctx.resolveType(primitiveName)
			if !ok {
				continue
			}
			if typ.Parent() == nil {
				model.SetTypeParent(typ, parent)
			}
		}
	}
}

func linkRFC1213TypesToTCs(ctx *resolverContext) {
	pairs := []struct {
		typeName     string
		sourceModule string
		targetModule string
	}{
		{"DisplayString", "RFC1213-MIB", moduleSNMPv2TC},
		{"PhysAddress", "RFC1213-MIB", moduleSNMPv2TC},
	}

	for _, pair := range pairs {
		sourceMods := ctx.moduleIndex[pair.sourceModule]
		targetMods := ctx.moduleIndex[pair.targetModule]
		if len(sourceMods) == 0 || len(targetMods) == 0 {
			continue
		}

		sourceType, ok := ctx.resolveTypeForModule(sourceMods[0], pair.typeName)
		if !ok {
			continue
		}
		targetType, ok := ctx.resolveTypeForModule(targetMods[0], pair.typeName)
		if !ok {
			continue
		}
		if sourceType != targetType {
			model.SetTypeParent(sourceType, targetType)
		}
	}
}

func inheritBaseTypes(ctx *resolverContext) {
	for _, t := range ctx.mib.Types() {
		if t.Parent() != nil && !isApplicationBaseType(t.Base()) {
			if base, ok := resolveBaseFromChain(t); ok {
				model.SetTypeBase(t, base)
			}
		}
	}
}

// isApplicationBaseType returns true for SMI application types that should not
// have their base type overwritten by inheritance. These types are defined with
// explicit base types in SNMPv2-SMI and should be preserved.
func isApplicationBaseType(b model.BaseType) bool {
	switch b {
	case model.BaseCounter32, model.BaseCounter64, model.BaseGauge32,
		model.BaseUnsigned32, model.BaseTimeTicks, model.BaseIpAddress, model.BaseOpaque:
		return true
	default:
		return false
	}
}

func resolveBaseFromChain(t *model.Type) (model.BaseType, bool) {
	for current, depth := t, 0; current != nil && depth < maxTypeChainDepth; current, depth = current.Parent(), depth+1 {
		if current.Parent() == nil {
			return current.Base(), true
		}
		// Stop at application base types - their base is explicitly set
		// and should not be overridden by walking further up the chain.
		if isApplicationBaseType(current.Base()) {
			return current.Base(), true
		}
	}
	return 0, false
}

func syntaxToBaseType(syntax module.TypeSyntax) (model.BaseType, bool) {
	switch s := syntax.(type) {
	case *module.TypeSyntaxTypeRef:
		switch s.Name {
		case "Integer32", "INTEGER":
			return model.BaseInteger32, true
		case "Counter32":
			return model.BaseCounter32, true
		case "Counter64":
			return model.BaseCounter64, true
		case "Gauge32":
			return model.BaseGauge32, true
		case "Unsigned32":
			return model.BaseUnsigned32, true
		case "TimeTicks":
			return model.BaseTimeTicks, true
		case "IpAddress":
			return model.BaseIpAddress, true
		case "Opaque":
			return model.BaseOpaque, true
		case "OCTET STRING":
			return model.BaseOctetString, true
		case "OBJECT IDENTIFIER":
			return model.BaseObjectIdentifier, true
		case "BITS":
			return model.BaseBits, true
		default:
			return 0, false
		}
	case *module.TypeSyntaxIntegerEnum:
		return model.BaseInteger32, true
	case *module.TypeSyntaxBits:
		return model.BaseBits, true
	case *module.TypeSyntaxOctetString:
		return model.BaseOctetString, true
	case *module.TypeSyntaxObjectIdentifier:
		return model.BaseObjectIdentifier, true
	case *module.TypeSyntaxConstrained:
		return syntaxToBaseType(s.Base)
	case *module.TypeSyntaxSequenceOf, *module.TypeSyntaxSequence:
		return model.BaseSequence, true
	default:
		return 0, false
	}
}

func rangesToConstraint(ranges []module.Range) []model.Range {
	out := make([]model.Range, 0, len(ranges))
	for _, r := range ranges {
		lo := rangeValueToI64(r.Min)
		hi := lo
		if r.Max != nil {
			hi = rangeValueToI64(r.Max)
		}
		out = append(out, model.Range{Min: lo, Max: hi, Span: r.Span})
	}
	return out
}

func rangeValueToI64(value module.RangeValue) int64 {
	switch v := value.(type) {
	case *module.RangeValueSigned:
		return v.Value
	case *module.RangeValueUnsigned:
		if v.Value > uint64(math.MaxInt64) {
			return math.MaxInt64
		}
		return int64(v.Value)
	case *module.RangeValueMin:
		return math.MinInt64
	case *module.RangeValueMax:
		return math.MaxInt64
	default:
		return 0
	}
}

// extractNamedValues extracts named values from IntegerEnum or Bits syntax.
func extractNamedValues(syntax module.TypeSyntax) []model.NamedValue {
	switch s := syntax.(type) {
	case *module.TypeSyntaxIntegerEnum:
		values := make([]model.NamedValue, 0, len(s.NamedNumbers))
		for _, nn := range s.NamedNumbers {
			values = append(values, model.NamedValue{Label: nn.Name, Value: nn.Value, Span: nn.Span})
		}
		return values
	case *module.TypeSyntaxBits:
		bits := make([]model.NamedValue, 0, len(s.NamedBits))
		for _, nb := range s.NamedBits {
			bits = append(bits, model.NamedValue{Label: nb.Name, Value: int64(nb.Position), Span: nb.Span})
		}
		return bits
	default:
		return nil
	}
}

// extractConstraints extracts size and value range constraints from syntax.
func extractConstraints(syntax module.TypeSyntax) (size, valueRange []model.Range) {
	constrained, ok := syntax.(*module.TypeSyntaxConstrained)
	if !ok {
		return nil, nil
	}
	switch c := constrained.Constraint.(type) {
	case *module.ConstraintSize:
		return rangesToConstraint(c.Ranges), nil
	case *module.ConstraintRange:
		return nil, rangesToConstraint(c.Ranges)
	}
	return nil, nil
}
