package resolver

import (
	"log/slog"

	"github.com/golangsnmp/gomib/internal/graph"
	"github.com/golangsnmp/gomib/internal/mibimpl"
	"github.com/golangsnmp/gomib/internal/module"
	"github.com/golangsnmp/gomib/mib"
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
	seedType := func(name string, base mib.BaseType) {
		typ := mibimpl.NewType(name)
		typ.SetModule(resolved)
		typ.SetBase(base)

		ctx.Builder.AddType(typ)
		ctx.RegisterModuleTypeSymbol(mod, name, typ)
		if resolved != nil {
			resolved.AddType(typ)
		}
		seeded++
		if ctx.TraceEnabled() {
			ctx.Trace("seeded primitive type", slog.String("name", name))
		}
	}

	seedType("INTEGER", mib.BaseInteger32)
	seedType("OCTET STRING", mib.BaseOctetString)
	seedType("OBJECT IDENTIFIER", mib.BaseObjectIdentifier)
	seedType("BITS", mib.BaseBits)
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
				base = convertBaseType(*td.BaseType)
				hasBase = true
			}
			if !hasBase {
				// Type references (e.g., DisplayString) don't have an intrinsic
				// base type. Default to Integer32 as a placeholder; the real base
				// is inherited from the parent during resolveTypeBases.
				base = mib.BaseInteger32
				if ctx.TraceEnabled() {
					ctx.Trace("type has no intrinsic base, defaulting to Integer32",
						slog.String("type", td.Name),
						slog.String("module", mod.Name))
				}
			}

			typ := mibimpl.NewType(td.Name)
			typ.SetModule(resolved)
			typ.SetBase(base)
			typ.SetIsTC(td.IsTextualConvention)
			typ.SetStatus(convertStatus(td.Status))
			typ.SetDisplayHint(td.DisplayHint)
			typ.SetDescription(td.Description)

			namedValues := extractNamedValues(td.Syntax)
			if base == mib.BaseBits {
				typ.SetBits(namedValues)
			} else {
				typ.SetEnums(namedValues)
			}
			sizes, ranges := extractConstraints(td.Syntax)
			typ.SetSizes(sizes)
			typ.SetRanges(ranges)

			ctx.Builder.AddType(typ)
			ctx.RegisterModuleTypeSymbol(mod, td.Name, typ)

			if ctx.TraceEnabled() {
				ctx.Trace("created user type",
					slog.String("name", td.Name),
					slog.String("base", base.String()))
			}

			if resolved != nil {
				resolved.AddType(typ)
			}
		}
	}
}

// typeResolutionEntry pairs a parsed type definition with its resolved type.
type typeResolutionEntry struct {
	mod *module.Module
	td  *module.TypeDef
	typ *mibimpl.Type
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

	if ctx.Snmpv2SMIModule != nil {
		if isASN1Primitive(typeName) || isSmiGlobalType(typeName) {
			return ctx.Snmpv2SMIModule.Name
		}
	}

	if ctx.Rfc1155SMIModule != nil && isSmiV1GlobalType(typeName) {
		return ctx.Rfc1155SMIModule.Name
	}

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

	entry.typ.SetParent(parent)
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
			if typ.InternalParent() == nil {
				typ.SetParent(parent)
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
			sourceType.SetParent(targetType)
		}
	}
}

func inheritBaseTypes(ctx *resolverContext) {
	for _, t := range ctx.Builder.Types() {
		if t.InternalParent() != nil && !isApplicationBaseType(t.Base()) {
			if base, ok := resolveBaseFromChain(t); ok {
				t.SetBase(base)
			}
		}
	}
}

// isApplicationBaseType returns true for SMI application types that should not
// have their base type overwritten by inheritance. These types are defined with
// explicit base types in SNMPv2-SMI and should be preserved.
func isApplicationBaseType(b mib.BaseType) bool {
	switch b {
	case mib.BaseCounter32, mib.BaseCounter64, mib.BaseGauge32,
		mib.BaseUnsigned32, mib.BaseTimeTicks, mib.BaseIpAddress, mib.BaseOpaque:
		return true
	default:
		return false
	}
}

func resolveBaseFromChain(t *mibimpl.Type) (mib.BaseType, bool) {
	visited := make(map[*mibimpl.Type]struct{})
	current := t
	for current != nil {
		if _, seen := visited[current]; seen {
			return 0, false
		}
		visited[current] = struct{}{}
		if current.InternalParent() == nil {
			return current.Base(), true
		}
		// Stop at application base types - their base is explicitly set
		// and should not be overridden by walking further up the chain.
		if isApplicationBaseType(current.Base()) {
			return current.Base(), true
		}
		current = current.InternalParent()
	}
	return 0, false
}

func syntaxToBaseType(syntax module.TypeSyntax) (mib.BaseType, bool) {
	switch s := syntax.(type) {
	case *module.TypeSyntaxTypeRef:
		switch s.Name {
		case "Integer32", "INTEGER":
			return mib.BaseInteger32, true
		case "Counter32":
			return mib.BaseCounter32, true
		case "Counter64":
			return mib.BaseCounter64, true
		case "Gauge32":
			return mib.BaseGauge32, true
		case "Unsigned32":
			return mib.BaseUnsigned32, true
		case "TimeTicks":
			return mib.BaseTimeTicks, true
		case "IpAddress":
			return mib.BaseIpAddress, true
		case "Opaque":
			return mib.BaseOpaque, true
		case "OCTET STRING":
			return mib.BaseOctetString, true
		case "OBJECT IDENTIFIER":
			return mib.BaseObjectIdentifier, true
		case "BITS":
			return mib.BaseBits, true
		default:
			return 0, false
		}
	case *module.TypeSyntaxIntegerEnum:
		return mib.BaseInteger32, true
	case *module.TypeSyntaxBits:
		return mib.BaseBits, true
	case *module.TypeSyntaxOctetString:
		return mib.BaseOctetString, true
	case *module.TypeSyntaxObjectIdentifier:
		return mib.BaseObjectIdentifier, true
	case *module.TypeSyntaxConstrained:
		return syntaxToBaseType(s.Base)
	default:
		return 0, false
	}
}

func convertBaseType(b module.BaseType) mib.BaseType {
	switch b {
	case module.BaseInteger32:
		return mib.BaseInteger32
	case module.BaseUnsigned32:
		return mib.BaseUnsigned32
	case module.BaseCounter32:
		return mib.BaseCounter32
	case module.BaseCounter64:
		return mib.BaseCounter64
	case module.BaseGauge32:
		return mib.BaseGauge32
	case module.BaseTimeTicks:
		return mib.BaseTimeTicks
	case module.BaseIpAddress:
		return mib.BaseIpAddress
	case module.BaseOctetString:
		return mib.BaseOctetString
	case module.BaseObjectIdentifier:
		return mib.BaseObjectIdentifier
	case module.BaseOpaque:
		return mib.BaseOpaque
	case module.BaseBits:
		return mib.BaseBits
	case module.BaseSequence:
		return mib.BaseSequence
	default:
		return mib.BaseUnknown
	}
}

func convertStatus(s module.Status) mib.Status {
	switch s {
	case module.StatusCurrent:
		return mib.StatusCurrent
	case module.StatusDeprecated:
		return mib.StatusDeprecated
	case module.StatusObsolete:
		return mib.StatusObsolete
	case module.StatusMandatory:
		return mib.StatusMandatory
	case module.StatusOptional:
		return mib.StatusOptional
	default:
		return mib.StatusCurrent
	}
}

func rangesToConstraint(ranges []module.Range) []mib.Range {
	out := make([]mib.Range, 0, len(ranges))
	for _, r := range ranges {
		min := rangeValueToI64(r.Min)
		max := min
		if r.Max != nil {
			max = rangeValueToI64(r.Max)
		}
		out = append(out, mib.Range{Min: min, Max: max})
	}
	return out
}

func rangeValueToI64(value module.RangeValue) int64 {
	switch v := value.(type) {
	case *module.RangeValueSigned:
		return v.Value
	case *module.RangeValueUnsigned:
		if v.Value > uint64(^uint64(0)>>1) {
			return ^int64(0) >> 1 // cap at max int64
		}
		return int64(v.Value)
	case *module.RangeValueMin:
		return ^int64(0) << 63
	case *module.RangeValueMax:
		return ^int64(0) >> 1
	default:
		return 0
	}
}

// extractNamedValues extracts named values from IntegerEnum or Bits syntax.
func extractNamedValues(syntax module.TypeSyntax) []mib.NamedValue {
	switch s := syntax.(type) {
	case *module.TypeSyntaxIntegerEnum:
		values := make([]mib.NamedValue, 0, len(s.NamedNumbers))
		for _, nn := range s.NamedNumbers {
			values = append(values, mib.NamedValue{Label: nn.Name, Value: nn.Value})
		}
		return values
	case *module.TypeSyntaxBits:
		bits := make([]mib.NamedValue, 0, len(s.NamedBits))
		for _, nb := range s.NamedBits {
			bits = append(bits, mib.NamedValue{Label: nb.Name, Value: int64(nb.Position)})
		}
		return bits
	default:
		return nil
	}
}

// extractConstraints extracts size and value range constraints from syntax.
func extractConstraints(syntax module.TypeSyntax) (size, valueRange []mib.Range) {
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
