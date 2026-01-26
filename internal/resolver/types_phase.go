package resolver

import (
	"github.com/golangsnmp/gomib/mib"
	"log/slog"

	"github.com/golangsnmp/gomib/internal/module"
	"github.com/golangsnmp/gomib/internal/types"
)

// resolveTypes resolves all types across all modules.
func resolveTypes(ctx *ResolverContext) {
	seedPrimitiveTypes(ctx)
	createUserTypes(ctx)
	resolveTypeBases(ctx)
}

func seedPrimitiveTypes(ctx *ResolverContext) {
	if ctx.Snmpv2SMIModule == nil {
		return
	}
	mod := ctx.Snmpv2SMIModule
	resolved := ctx.ModuleToResolved[mod]

	seeded := 0
	seedType := func(name string, base mib.BaseType) {
		typ := &mib.Type{
			Name:   name,
			Module: resolved,
			Base:   base,
		}
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

func createUserTypes(ctx *ResolverContext) {
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
				base = mib.BaseInteger32
			}

			typ := &mib.Type{
				Name:        td.Name,
				Module:      resolved,
				Base:        base,
				IsTC:        td.IsTextualConvention,
				Status:      convertStatus(td.Status),
				Hint:        td.DisplayHint,
				Description: td.Description,
			}

			// Extract constraints and named values
			typ.NamedValues = extractNamedValues(td.Syntax)
			typ.Size, typ.ValueRange = extractConstraints(td.Syntax)

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

// typeResolutionEntry tracks a type that needs parent resolution.
type typeResolutionEntry struct {
	mod *module.Module
	td  *module.TypeDef
	typ *mib.Type
}

func resolveTypeBases(ctx *ResolverContext) {
	resolveTypeRefParentsMultipass(ctx)
	linkPrimitiveSyntaxParents(ctx)
	linkRFC1213TypesToTCs(ctx)
	inheritBaseTypes(ctx)
}

func resolveTypeRefParentsMultipass(ctx *ResolverContext) {
	var pending []typeResolutionEntry
	for _, mod := range ctx.Modules {
		for _, def := range mod.Definitions {
			td, ok := def.(*module.TypeDef)
			if !ok {
				continue
			}
			if hasTypeRefSyntax(td.Syntax) {
				if typ, ok := ctx.LookupTypeForModule(mod, td.Name); ok {
					pending = append(pending, typeResolutionEntry{mod: mod, td: td, typ: typ})
				}
			}
		}
	}

	maxIterations := 20
	for iter := 0; iter < maxIterations && len(pending) > 0; iter++ {
		initial := len(pending)
		var still []typeResolutionEntry
		for _, entry := range pending {
			if !tryResolveTypeParent(ctx, entry) {
				still = append(still, entry)
			}
		}

		if ctx.TraceEnabled() {
			resolved := initial - len(still)
			ctx.Trace("type resolution pass",
				slog.Int("iteration", iter+1),
				slog.Int("resolved", resolved),
				slog.Int("still_pending", len(still)))
		}

		if len(still) == initial {
			for _, entry := range still {
				baseName := getTypeRefBaseName(entry.td.Syntax)
				if baseName == "" {
					continue
				}
				ctx.RecordUnresolvedType(entry.mod, entry.td.Name, baseName, entry.td.Span)
			}
			break
		}
		pending = still
	}
}

func hasTypeRefSyntax(syntax module.TypeSyntax) bool {
	switch s := syntax.(type) {
	case *module.TypeSyntaxTypeRef:
		return true
	case *module.TypeSyntaxConstrained:
		_, ok := s.Base.(*module.TypeSyntaxTypeRef)
		return ok
	default:
		return false
	}
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

func tryResolveTypeParent(ctx *ResolverContext, entry typeResolutionEntry) bool {
	baseName := getTypeRefBaseName(entry.td.Syntax)
	if baseName == "" {
		return false
	}

	parent, ok := ctx.LookupTypeForModule(entry.mod, baseName)
	if !ok {
		return false
	}

	// Check if parent is ready (has its own parent resolved if needed)
	if parent.Parent != nil || !hasTypeRefSyntax(findTypeDef(ctx, entry.mod, baseName)) {
		entry.typ.Parent = parent
		return true
	}

	return false
}

func findTypeDef(ctx *ResolverContext, mod *module.Module, name string) module.TypeSyntax {
	for _, def := range mod.Definitions {
		if td, ok := def.(*module.TypeDef); ok && td.Name == name {
			return td.Syntax
		}
	}
	return nil
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

func linkPrimitiveSyntaxParents(ctx *ResolverContext) {
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
			if typ.Parent == nil {
				typ.Parent = parent
			}
		}
	}
}

func linkRFC1213TypesToTCs(ctx *ResolverContext) {
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
			sourceType.Parent = targetType
		}
	}
}

func inheritBaseTypes(ctx *ResolverContext) {
	for _, t := range ctx.Builder.Types() {
		// Skip types that already have an application base type (explicitly set)
		if t.Parent != nil && !isApplicationBaseType(t.Base) {
			if base, ok := resolveBaseFromChain(t); ok {
				t.Base = base
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

func resolveBaseFromChain(t *mib.Type) (mib.BaseType, bool) {
	visited := make(map[*mib.Type]struct{})
	current := t
	for current != nil {
		if _, seen := visited[current]; seen {
			return 0, false
		}
		visited[current] = struct{}{}
		if current.Parent == nil {
			return current.Base, true
		}
		current = current.Parent
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

func convertBaseType(b types.BaseType) mib.BaseType {
	switch b {
	case types.BaseInteger32:
		return mib.BaseInteger32
	case types.BaseUnsigned32:
		return mib.BaseUnsigned32
	case types.BaseCounter32:
		return mib.BaseCounter32
	case types.BaseCounter64:
		return mib.BaseCounter64
	case types.BaseGauge32:
		return mib.BaseGauge32
	case types.BaseTimeTicks:
		return mib.BaseTimeTicks
	case types.BaseIpAddress:
		return mib.BaseIpAddress
	case types.BaseOctetString:
		return mib.BaseOctetString
	case types.BaseObjectIdentifier:
		return mib.BaseObjectIdentifier
	case types.BaseOpaque:
		return mib.BaseOpaque
	case types.BaseBits:
		return mib.BaseBits
	default:
		return mib.BaseInteger32
	}
}

func convertStatus(s types.Status) mib.Status {
	switch s {
	case types.StatusCurrent:
		return mib.StatusCurrent
	case types.StatusDeprecated:
		return mib.StatusDeprecated
	case types.StatusObsolete:
		return mib.StatusObsolete
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
