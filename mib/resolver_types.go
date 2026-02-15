package mib

import (
	"log/slog"
	"math"

	"github.com/golangsnmp/gomib/internal/graph"
	"github.com/golangsnmp/gomib/internal/module"
)

// resolveTypes is the type resolution phase entry point.
func resolveTypes(ctx *resolverContext) {
	seedPrimitiveTypes(ctx)
	createUserTypes(ctx)
	resolveTypeBases(ctx)
}

func seedPrimitiveTypes(ctx *resolverContext) {
	if ctx.Snmpv2SMIModule == nil {
		return
	}
	mod := ctx.Snmpv2SMIModule
	resolved := ctx.ModuleToResolved[mod]

	seeded := 0
	seedType := func(name string, base BaseType) {
		typ := newType(name)
		typ.setModule(resolved)
		typ.setBase(base)

		ctx.Mib.addType(typ)
		ctx.registerModuleTypeSymbol(mod, name, typ)
		if resolved != nil {
			resolved.addType(typ)
		}
		seeded++
		if ctx.TraceEnabled() {
			ctx.Trace("seeded primitive type", slog.String("name", name))
		}
	}

	seedType("INTEGER", BaseInteger32)
	seedType("OCTET STRING", BaseOctetString)
	seedType("OBJECT IDENTIFIER", BaseObjectIdentifier)
	seedType("BITS", BaseBits)
}

func createUserTypes(ctx *resolverContext) {
	for _, mod := range ctx.Modules {
		resolved := ctx.ModuleToResolved[mod]

		for _, def := range mod.Definitions {
			td, ok := def.(*module.TypeDef)
			if !ok {
				continue
			}

			base, hasBase := syntaxToBaseType(td.Syntax)
			if td.BaseType != nil {
				base = *td.BaseType
				hasBase = true
			}
			if !hasBase {
				// Type references (e.g., DisplayString) don't have an intrinsic
				// base type. Use BaseUnknown as a placeholder; the real base
				// is inherited from the parent during resolveTypeBases.
				base = BaseUnknown
				if ctx.TraceEnabled() {
					ctx.Trace("type has no intrinsic base, will inherit from parent",
						slog.String("type", td.Name),
						slog.String("module", mod.Name))
				}
			}

			typ := newType(td.Name)
			typ.setModule(resolved)
			typ.setBase(base)
			typ.setIsTC(td.IsTextualConvention)
			typ.setStatus(td.Status)
			typ.setDisplayHint(td.DisplayHint)
			typ.setDescription(td.Description)

			namedValues := extractNamedValues(td.Syntax)
			if base == BaseBits {
				typ.setBits(namedValues)
			} else {
				typ.setEnums(namedValues)
			}
			sizes, ranges := extractConstraints(td.Syntax)
			typ.setSizes(sizes)
			typ.setRanges(ranges)

			ctx.Mib.addType(typ)
			ctx.registerModuleTypeSymbol(mod, td.Name, typ)

			if ctx.TraceEnabled() {
				ctx.Trace("created user type",
					slog.String("name", td.Name),
					slog.String("base", base.String()))
			}

			if resolved != nil {
				resolved.addType(typ)
			}
		}
	}
}

// typeResolutionEntry pairs a parsed type definition with its resolved type.
type typeResolutionEntry struct {
	mod *module.Module
	td  *module.TypeDef
	typ *Type
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
	entries := make(map[graph.Symbol]typeResolutionEntry)
	g := graph.New()

	for _, mod := range ctx.Modules {
		for _, def := range mod.Definitions {
			td, ok := def.(*module.TypeDef)
			if !ok {
				continue
			}
			typ, ok := ctx.LookupTypeForModule(mod, td.Name)
			if !ok {
				continue
			}

			sym := graph.Symbol{Module: mod.Name, Name: td.Name}
			g.AddNode(sym, graph.NodeKindType)
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

	cycles := g.FindCycles()
	logCycles(ctx, cycles, "type cycle detected")

	order, cyclic := g.ResolutionOrder()

	if ctx.TraceEnabled() {
		ctx.Trace("type resolution order",
			slog.Int("total", len(order)),
			slog.Int("cyclic", len(cyclic)))
	}

	resolved := 0
	for _, sym := range order {
		entry, ok := entries[sym]
		if !ok {
			continue // External dependency (primitive or from another module)
		}
		if resolveTypeParent(ctx, entry) {
			resolved++
		}
	}

	for _, sym := range cyclic {
		entry, ok := entries[sym]
		if !ok {
			continue
		}
		baseName := getTypeRefBaseName(entry.td.Syntax)
		if baseName != "" {
			ctx.RecordUnresolvedType(entry.mod, entry.td.Name, baseName, entry.td.Span)
		}
	}

	if ctx.TraceEnabled() {
		ctx.Trace("type resolution complete",
			slog.Int("resolved", resolved),
			slog.Int("unresolved", len(cyclic)))
	}
}

// findTypeDefiningModule finds the module that defines a type, following imports.
func findTypeDefiningModule(ctx *resolverContext, fromMod *module.Module, typeName string) string {
	for _, def := range fromMod.Definitions {
		if td, ok := def.(*module.TypeDef); ok && td.Name == typeName {
			return fromMod.Name
		}
	}

	if imports := ctx.ModuleImports[fromMod]; imports != nil {
		if srcMod := imports[typeName]; srcMod != nil {
			return srcMod.Name
		}
	}

	// RFC-compliant: ASN.1 primitives are always available
	if ctx.Snmpv2SMIModule != nil && isASN1Primitive(typeName) {
		return ctx.Snmpv2SMIModule.Name
	}

	if !ctx.diagConfig.AllowBestGuessFallbacks() {
		return ""
	}

	// Permissive only: SMI global types from SNMPv2-SMI
	if ctx.Snmpv2SMIModule != nil && isSmiGlobalType(typeName) {
		return ctx.Snmpv2SMIModule.Name
	}

	// Permissive only: SMIv1 types from RFC1155-SMI
	if ctx.Rfc1155SMIModule != nil && isSmiV1GlobalType(typeName) {
		return ctx.Rfc1155SMIModule.Name
	}

	// Permissive only: SNMPv2-TC textual conventions
	if ctx.Snmpv2TCModule != nil && isSNMPv2TCType(typeName) {
		return ctx.Snmpv2TCModule.Name
	}

	return ""
}

func resolveTypeParent(ctx *resolverContext, entry typeResolutionEntry) bool {
	baseName := getTypeRefBaseName(entry.td.Syntax)
	if baseName == "" {
		return false
	}

	parent, ok := ctx.LookupTypeForModule(entry.mod, baseName)
	if !ok {
		return false
	}

	entry.typ.setParent(parent)
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
		switch s.Base.(type) {
		case *module.TypeSyntaxOctetString:
			return "OCTET STRING"
		case *module.TypeSyntaxObjectIdentifier:
			return "OBJECT IDENTIFIER"
		}
	}
	return ""
}

func linkPrimitiveSyntaxParents(ctx *resolverContext) {
	for _, mod := range ctx.Modules {
		for _, def := range mod.Definitions {
			td, ok := def.(*module.TypeDef)
			if !ok {
				continue
			}
			primitiveName := getPrimitiveParentName(td.Syntax)
			if primitiveName == "" {
				continue
			}

			typ, ok := ctx.LookupTypeForModule(mod, td.Name)
			if !ok {
				continue
			}
			parent, ok := ctx.LookupType(primitiveName)
			if !ok {
				continue
			}
			if typ.Parent() == nil {
				typ.setParent(parent)
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
		{"DisplayString", "RFC1213-MIB", "SNMPv2-TC"},
		{"PhysAddress", "RFC1213-MIB", "SNMPv2-TC"},
	}

	for _, pair := range pairs {
		sourceMods := ctx.ModuleIndex[pair.sourceModule]
		targetMods := ctx.ModuleIndex[pair.targetModule]
		if len(sourceMods) == 0 || len(targetMods) == 0 {
			continue
		}

		sourceType, ok := ctx.LookupTypeForModule(sourceMods[0], pair.typeName)
		if !ok {
			continue
		}
		targetType, ok := ctx.LookupTypeForModule(targetMods[0], pair.typeName)
		if !ok {
			continue
		}
		if sourceType != targetType {
			sourceType.setParent(targetType)
		}
	}
}

func inheritBaseTypes(ctx *resolverContext) {
	for _, t := range ctx.Mib.Types() {
		if t.Parent() != nil && !isApplicationBaseType(t.Base()) {
			if base, ok := resolveBaseFromChain(t); ok {
				t.setBase(base)
			}
		}
	}
}

// isApplicationBaseType returns true for SMI application types that should not
// have their base type overwritten by inheritance. These types are defined with
// explicit base types in SNMPv2-SMI and should be preserved.
func isApplicationBaseType(b BaseType) bool {
	switch b {
	case BaseCounter32, BaseCounter64, BaseGauge32,
		BaseUnsigned32, BaseTimeTicks, BaseIpAddress, BaseOpaque:
		return true
	default:
		return false
	}
}

func resolveBaseFromChain(t *Type) (BaseType, bool) {
	visited := make(map[*Type]struct{})
	current := t
	for current != nil {
		if _, seen := visited[current]; seen {
			return 0, false
		}
		visited[current] = struct{}{}
		if current.Parent() == nil {
			return current.Base(), true
		}
		// Stop at application base types - their base is explicitly set
		// and should not be overridden by walking further up the chain.
		if isApplicationBaseType(current.Base()) {
			return current.Base(), true
		}
		current = current.Parent()
	}
	return 0, false
}

func syntaxToBaseType(syntax module.TypeSyntax) (BaseType, bool) {
	switch s := syntax.(type) {
	case *module.TypeSyntaxTypeRef:
		switch s.Name {
		case "Integer32", "INTEGER":
			return BaseInteger32, true
		case "Counter32":
			return BaseCounter32, true
		case "Counter64":
			return BaseCounter64, true
		case "Gauge32":
			return BaseGauge32, true
		case "Unsigned32":
			return BaseUnsigned32, true
		case "TimeTicks":
			return BaseTimeTicks, true
		case "IpAddress":
			return BaseIpAddress, true
		case "Opaque":
			return BaseOpaque, true
		case "OCTET STRING":
			return BaseOctetString, true
		case "OBJECT IDENTIFIER":
			return BaseObjectIdentifier, true
		case "BITS":
			return BaseBits, true
		default:
			return 0, false
		}
	case *module.TypeSyntaxIntegerEnum:
		return BaseInteger32, true
	case *module.TypeSyntaxBits:
		return BaseBits, true
	case *module.TypeSyntaxOctetString:
		return BaseOctetString, true
	case *module.TypeSyntaxObjectIdentifier:
		return BaseObjectIdentifier, true
	case *module.TypeSyntaxConstrained:
		return syntaxToBaseType(s.Base)
	default:
		return 0, false
	}
}

func rangesToConstraint(ranges []module.Range) []Range {
	out := make([]Range, 0, len(ranges))
	for _, r := range ranges {
		min := rangeValueToI64(r.Min)
		max := min
		if r.Max != nil {
			max = rangeValueToI64(r.Max)
		}
		out = append(out, Range{Min: min, Max: max})
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
func extractNamedValues(syntax module.TypeSyntax) []NamedValue {
	switch s := syntax.(type) {
	case *module.TypeSyntaxIntegerEnum:
		values := make([]NamedValue, 0, len(s.NamedNumbers))
		for _, nn := range s.NamedNumbers {
			values = append(values, NamedValue{Label: nn.Name, Value: nn.Value})
		}
		return values
	case *module.TypeSyntaxBits:
		bits := make([]NamedValue, 0, len(s.NamedBits))
		for _, nb := range s.NamedBits {
			bits = append(bits, NamedValue{Label: nb.Name, Value: int64(nb.Position)})
		}
		return bits
	default:
		return nil
	}
}

// extractConstraints extracts size and value range constraints from syntax.
func extractConstraints(syntax module.TypeSyntax) (size, valueRange []Range) {
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
