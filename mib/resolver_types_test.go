package mib

import (
	"math"
	"testing"

	"github.com/golangsnmp/gomib/internal/module"
	"github.com/golangsnmp/gomib/internal/testutil"
	"github.com/golangsnmp/gomib/internal/types"
)

func TestSyntaxToBaseType(t *testing.T) {
	tests := []struct {
		name   string
		syntax module.TypeSyntax
		want   BaseType
		wantOK bool
	}{
		// TypeRef variants - known base types
		{"typeref Integer32", &module.TypeSyntaxTypeRef{Name: "Integer32"}, BaseInteger32, true},
		{"typeref INTEGER", &module.TypeSyntaxTypeRef{Name: "INTEGER"}, BaseInteger32, true},
		{"typeref Counter32", &module.TypeSyntaxTypeRef{Name: "Counter32"}, BaseCounter32, true},
		{"typeref Counter64", &module.TypeSyntaxTypeRef{Name: "Counter64"}, BaseCounter64, true},
		{"typeref Gauge32", &module.TypeSyntaxTypeRef{Name: "Gauge32"}, BaseGauge32, true},
		{"typeref Unsigned32", &module.TypeSyntaxTypeRef{Name: "Unsigned32"}, BaseUnsigned32, true},
		{"typeref TimeTicks", &module.TypeSyntaxTypeRef{Name: "TimeTicks"}, BaseTimeTicks, true},
		{"typeref IpAddress", &module.TypeSyntaxTypeRef{Name: "IpAddress"}, BaseIpAddress, true},
		{"typeref Opaque", &module.TypeSyntaxTypeRef{Name: "Opaque"}, BaseOpaque, true},
		{"typeref OCTET STRING", &module.TypeSyntaxTypeRef{Name: "OCTET STRING"}, BaseOctetString, true},
		{"typeref OBJECT IDENTIFIER", &module.TypeSyntaxTypeRef{Name: "OBJECT IDENTIFIER"}, BaseObjectIdentifier, true},
		{"typeref BITS", &module.TypeSyntaxTypeRef{Name: "BITS"}, BaseBits, true},

		// TypeRef - unknown name (user-defined type)
		{"typeref DisplayString", &module.TypeSyntaxTypeRef{Name: "DisplayString"}, 0, false},
		{"typeref unknown", &module.TypeSyntaxTypeRef{Name: "MyCustomType"}, 0, false},

		// Primitive syntax types
		{"IntegerEnum", &module.TypeSyntaxIntegerEnum{
			NamedNumbers: []module.NamedNumber{{Name: "up", Value: 1}},
		}, BaseInteger32, true},
		{"Bits", &module.TypeSyntaxBits{
			NamedBits: []module.NamedBit{{Name: "flag0", Position: 0}},
		}, BaseBits, true},
		{"OctetString", &module.TypeSyntaxOctetString{}, BaseOctetString, true},
		{"ObjectIdentifier", &module.TypeSyntaxObjectIdentifier{}, BaseObjectIdentifier, true},

		// Constrained wrapping - delegates to inner syntax
		{"constrained Integer32", &module.TypeSyntaxConstrained{
			Base:       &module.TypeSyntaxTypeRef{Name: "Integer32"},
			Constraint: &module.ConstraintRange{Ranges: []module.Range{module.NewRangeSigned(0, 100, types.Span{})}},
		}, BaseInteger32, true},
		{"constrained OCTET STRING", &module.TypeSyntaxConstrained{
			Base:       &module.TypeSyntaxTypeRef{Name: "OCTET STRING"},
			Constraint: &module.ConstraintSize{Ranges: []module.Range{module.NewRangeUnsigned(0, 255, types.Span{})}},
		}, BaseOctetString, true},
		{"constrained OctetString primitive", &module.TypeSyntaxConstrained{
			Base:       &module.TypeSyntaxOctetString{},
			Constraint: &module.ConstraintSize{Ranges: []module.Range{module.NewRangeUnsigned(0, 255, types.Span{})}},
		}, BaseOctetString, true},
		{"constrained ObjectIdentifier primitive", &module.TypeSyntaxConstrained{
			Base:       &module.TypeSyntaxObjectIdentifier{},
			Constraint: &module.ConstraintSize{},
		}, BaseObjectIdentifier, true},
		{"constrained unknown typeref", &module.TypeSyntaxConstrained{
			Base:       &module.TypeSyntaxTypeRef{Name: "DisplayString"},
			Constraint: &module.ConstraintSize{},
		}, 0, false},

		// SequenceOf and Sequence resolve to BaseSequence
		{"SequenceOf", &module.TypeSyntaxSequenceOf{EntryType: "FooEntry"}, BaseSequence, true},
		{"Sequence", &module.TypeSyntaxSequence{Fields: []module.SequenceField{}}, BaseSequence, true},

		// nil syntax
		{"nil", nil, 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, gotOK := syntaxToBaseType(tt.syntax)
			testutil.Equal(t, tt.want, got, "base type")
			testutil.Equal(t, tt.wantOK, gotOK, "ok")
		})
	}
}

func TestGetTypeRefBaseName(t *testing.T) {
	tests := []struct {
		name   string
		syntax module.TypeSyntax
		want   string
	}{
		{"TypeRef", &module.TypeSyntaxTypeRef{Name: "DisplayString"}, "DisplayString"},
		{"TypeRef Integer32", &module.TypeSyntaxTypeRef{Name: "Integer32"}, "Integer32"},
		{"Constrained TypeRef", &module.TypeSyntaxConstrained{
			Base:       &module.TypeSyntaxTypeRef{Name: "Counter32"},
			Constraint: &module.ConstraintRange{},
		}, "Counter32"},
		{"Constrained non-TypeRef", &module.TypeSyntaxConstrained{
			Base:       &module.TypeSyntaxOctetString{},
			Constraint: &module.ConstraintSize{},
		}, ""},
		{"IntegerEnum", &module.TypeSyntaxIntegerEnum{}, ""},
		{"Bits", &module.TypeSyntaxBits{}, ""},
		{"OctetString", &module.TypeSyntaxOctetString{}, ""},
		{"ObjectIdentifier", &module.TypeSyntaxObjectIdentifier{}, ""},
		{"nil", nil, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getTypeRefBaseName(tt.syntax)
			testutil.Equal(t, tt.want, got, "getTypeRefBaseName()")
		})
	}
}

func TestGetPrimitiveParentName(t *testing.T) {
	tests := []struct {
		name   string
		syntax module.TypeSyntax
		want   string
	}{
		{"OctetString", &module.TypeSyntaxOctetString{}, "OCTET STRING"},
		{"ObjectIdentifier", &module.TypeSyntaxObjectIdentifier{}, "OBJECT IDENTIFIER"},
		{"IntegerEnum", &module.TypeSyntaxIntegerEnum{}, "INTEGER"},
		{"Bits", &module.TypeSyntaxBits{}, "BITS"},
		{"Constrained OctetString", &module.TypeSyntaxConstrained{
			Base:       &module.TypeSyntaxOctetString{},
			Constraint: &module.ConstraintSize{},
		}, "OCTET STRING"},
		{"Constrained ObjectIdentifier", &module.TypeSyntaxConstrained{
			Base:       &module.TypeSyntaxObjectIdentifier{},
			Constraint: &module.ConstraintSize{},
		}, "OBJECT IDENTIFIER"},
		{"Constrained IntegerEnum", &module.TypeSyntaxConstrained{
			Base:       &module.TypeSyntaxIntegerEnum{},
			Constraint: &module.ConstraintRange{},
		}, "INTEGER"},
		{"Constrained Bits", &module.TypeSyntaxConstrained{
			Base:       &module.TypeSyntaxBits{},
			Constraint: &module.ConstraintSize{},
		}, "BITS"},
		{"Constrained TypeRef", &module.TypeSyntaxConstrained{
			Base:       &module.TypeSyntaxTypeRef{Name: "Integer32"},
			Constraint: &module.ConstraintRange{},
		}, ""},
		{"TypeRef", &module.TypeSyntaxTypeRef{Name: "Integer32"}, ""},
		{"SequenceOf", &module.TypeSyntaxSequenceOf{EntryType: "FooEntry"}, ""},
		{"nil", nil, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getPrimitiveParentName(tt.syntax)
			testutil.Equal(t, tt.want, got, "getPrimitiveParentName()")
		})
	}
}

func TestIsApplicationBaseType(t *testing.T) {
	tests := []struct {
		name string
		base BaseType
		want bool
	}{
		{"Counter32", BaseCounter32, true},
		{"Counter64", BaseCounter64, true},
		{"Gauge32", BaseGauge32, true},
		{"Unsigned32", BaseUnsigned32, true},
		{"TimeTicks", BaseTimeTicks, true},
		{"IpAddress", BaseIpAddress, true},
		{"Opaque", BaseOpaque, true},
		{"Integer32", BaseInteger32, false},
		{"OctetString", BaseOctetString, false},
		{"ObjectIdentifier", BaseObjectIdentifier, false},
		{"Bits", BaseBits, false},
		{"Sequence", BaseSequence, false},
		{"Unknown", BaseUnknown, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isApplicationBaseType(tt.base)
			testutil.Equal(t, tt.want, got, "isApplicationBaseType()")
		})
	}
}

func TestExtractNamedValues(t *testing.T) {
	t.Run("IntegerEnum", func(t *testing.T) {
		syntax := &module.TypeSyntaxIntegerEnum{
			NamedNumbers: []module.NamedNumber{
				{Name: "up", Value: 1},
				{Name: "down", Value: 2},
				{Name: "testing", Value: 3},
			},
		}
		got := extractNamedValues(syntax)
		testutil.Len(t, got, 3, "named values")
		assertNamedValue(t, got[0], "up", 1)
		assertNamedValue(t, got[1], "down", 2)
		assertNamedValue(t, got[2], "testing", 3)
	})

	t.Run("IntegerEnum empty", func(t *testing.T) {
		syntax := &module.TypeSyntaxIntegerEnum{}
		got := extractNamedValues(syntax)
		testutil.Len(t, got, 0, "named values")
	})

	t.Run("Bits", func(t *testing.T) {
		syntax := &module.TypeSyntaxBits{
			NamedBits: []module.NamedBit{
				{Name: "overflow", Position: 0},
				{Name: "underflow", Position: 1},
				{Name: "reserved", Position: 7},
			},
		}
		got := extractNamedValues(syntax)
		testutil.Len(t, got, 3, "named values")
		assertNamedValue(t, got[0], "overflow", 0)
		assertNamedValue(t, got[1], "underflow", 1)
		assertNamedValue(t, got[2], "reserved", 7)
	})

	t.Run("Bits empty", func(t *testing.T) {
		syntax := &module.TypeSyntaxBits{}
		got := extractNamedValues(syntax)
		testutil.Len(t, got, 0, "named values")
	})

	t.Run("non-enum types return nil", func(t *testing.T) {
		for _, syntax := range []module.TypeSyntax{
			&module.TypeSyntaxTypeRef{Name: "Integer32"},
			&module.TypeSyntaxOctetString{},
			nil,
		} {
			got := extractNamedValues(syntax)
			testutil.Nil(t, got, "expected nil")
		}
	})

	t.Run("negative enum values", func(t *testing.T) {
		syntax := &module.TypeSyntaxIntegerEnum{
			NamedNumbers: []module.NamedNumber{
				{Name: "neg", Value: -1},
				{Name: "zero", Value: 0},
			},
		}
		got := extractNamedValues(syntax)
		testutil.Len(t, got, 2, "named values")
		assertNamedValue(t, got[0], "neg", -1)
		assertNamedValue(t, got[1], "zero", 0)
	})
}

func assertNamedValue(t *testing.T, nv NamedValue, wantLabel string, wantValue int64) {
	t.Helper()
	testutil.Equal(t, wantLabel, nv.Label, "label")
	testutil.Equal(t, wantValue, nv.Value, "value")
}

func TestExtractConstraints(t *testing.T) {
	t.Run("size constraint", func(t *testing.T) {
		syntax := &module.TypeSyntaxConstrained{
			Base: &module.TypeSyntaxOctetString{},
			Constraint: &module.ConstraintSize{
				Ranges: []module.Range{
					module.NewRangeUnsigned(0, 255, types.Span{}),
				},
			},
		}
		sizes, ranges := extractConstraints(syntax)
		testutil.Len(t, sizes, 1, "sizes")
		testutil.Equal(t, int64(0), sizes[0].Min, "min")
		testutil.Equal(t, int64(255), sizes[0].Max, "max")
		testutil.Nil(t, ranges, "ranges")
	})

	t.Run("range constraint", func(t *testing.T) {
		syntax := &module.TypeSyntaxConstrained{
			Base: &module.TypeSyntaxTypeRef{Name: "Integer32"},
			Constraint: &module.ConstraintRange{
				Ranges: []module.Range{
					module.NewRangeSigned(-128, 127, types.Span{}),
				},
			},
		}
		sizes, ranges := extractConstraints(syntax)
		testutil.Nil(t, sizes, "sizes")
		testutil.Len(t, ranges, 1, "ranges")
		testutil.Equal(t, int64(-128), ranges[0].Min, "min")
		testutil.Equal(t, int64(127), ranges[0].Max, "max")
	})

	t.Run("multiple size ranges", func(t *testing.T) {
		syntax := &module.TypeSyntaxConstrained{
			Base: &module.TypeSyntaxOctetString{},
			Constraint: &module.ConstraintSize{
				Ranges: []module.Range{
					module.NewRangeSingleUnsigned(0, types.Span{}),
					module.NewRangeUnsigned(4, 255, types.Span{}),
				},
			},
		}
		sizes, ranges := extractConstraints(syntax)
		testutil.Len(t, sizes, 2, "sizes")
		// Single value: max = min
		testutil.Equal(t, int64(0), sizes[0].Min, "sizes[0] min")
		testutil.Equal(t, int64(0), sizes[0].Max, "sizes[0] max")
		testutil.Equal(t, int64(4), sizes[1].Min, "sizes[1] min")
		testutil.Equal(t, int64(255), sizes[1].Max, "sizes[1] max")
		testutil.Nil(t, ranges, "ranges")
	})

	t.Run("non-constrained syntax returns nil", func(t *testing.T) {
		sizes, ranges := extractConstraints(&module.TypeSyntaxTypeRef{Name: "Integer32"})
		testutil.Nil(t, sizes, "sizes")
		testutil.Nil(t, ranges, "ranges")
	})

	t.Run("nil syntax returns nil", func(t *testing.T) {
		sizes, ranges := extractConstraints(nil)
		testutil.Nil(t, sizes, "sizes")
		testutil.Nil(t, ranges, "ranges")
	})
}

func TestRangesToConstraint(t *testing.T) {
	t.Run("signed range", func(t *testing.T) {
		ranges := []module.Range{
			module.NewRangeSigned(-100, 100, types.Span{}),
		}
		got := rangesToConstraint(ranges)
		testutil.Len(t, got, 1, "ranges")
		testutil.Equal(t, int64(-100), got[0].Min, "min")
		testutil.Equal(t, int64(100), got[0].Max, "max")
	})

	t.Run("single value range", func(t *testing.T) {
		ranges := []module.Range{
			module.NewRangeSingleSigned(42, types.Span{}),
		}
		got := rangesToConstraint(ranges)
		testutil.Len(t, got, 1, "ranges")
		// Single value: Max is nil, so max = min
		testutil.Equal(t, int64(42), got[0].Min, "min")
		testutil.Equal(t, int64(42), got[0].Max, "max")
	})

	t.Run("multiple ranges", func(t *testing.T) {
		ranges := []module.Range{
			module.NewRangeSigned(0, 10, types.Span{}),
			module.NewRangeSigned(100, 200, types.Span{}),
		}
		got := rangesToConstraint(ranges)
		testutil.Len(t, got, 2, "ranges")
		testutil.Equal(t, int64(0), got[0].Min, "got[0] min")
		testutil.Equal(t, int64(10), got[0].Max, "got[0] max")
		testutil.Equal(t, int64(100), got[1].Min, "got[1] min")
		testutil.Equal(t, int64(200), got[1].Max, "got[1] max")
	})

	t.Run("empty ranges", func(t *testing.T) {
		got := rangesToConstraint(nil)
		testutil.Len(t, got, 0, "ranges")
	})
}

func TestRangeValueToI64(t *testing.T) {
	tests := []struct {
		name string
		val  module.RangeValue
		want int64
	}{
		{"signed positive", &module.RangeValueSigned{Value: 42}, 42},
		{"signed zero", &module.RangeValueSigned{Value: 0}, 0},
		{"signed negative", &module.RangeValueSigned{Value: -100}, -100},
		{"signed max int64", &module.RangeValueSigned{Value: math.MaxInt64}, math.MaxInt64},
		{"signed min int64", &module.RangeValueSigned{Value: math.MinInt64}, math.MinInt64},

		{"unsigned small", &module.RangeValueUnsigned{Value: 255}, 255},
		{"unsigned zero", &module.RangeValueUnsigned{Value: 0}, 0},
		{"unsigned max safe", &module.RangeValueUnsigned{Value: uint64(math.MaxInt64)}, math.MaxInt64},
		{"unsigned overflow", &module.RangeValueUnsigned{Value: uint64(math.MaxInt64) + 1}, math.MaxInt64},
		{"unsigned max uint64", &module.RangeValueUnsigned{Value: math.MaxUint64}, math.MaxInt64},

		{"MIN", &module.RangeValueMin{}, math.MinInt64},
		{"MAX", &module.RangeValueMax{}, math.MaxInt64},

		{"nil", nil, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := rangeValueToI64(tt.val)
			testutil.Equal(t, tt.want, got, "rangeValueToI64()")
		})
	}
}

func TestResolveBaseFromChain(t *testing.T) {
	t.Run("no parent returns own base", func(t *testing.T) {
		typ := newType("MyType")
		typ.setBase(BaseOctetString)

		got, ok := resolveBaseFromChain(typ)
		testutil.True(t, ok, "expected ok=true")
		testutil.Equal(t, BaseOctetString, got, "got")
	})

	t.Run("walks chain to root", func(t *testing.T) {
		root := newType("INTEGER")
		root.setBase(BaseInteger32)

		mid := newType("MyInt")
		mid.setBase(BaseInteger32)
		mid.setParent(root)

		leaf := newType("MySpecificInt")
		leaf.setBase(BaseInteger32)
		leaf.setParent(mid)

		got, ok := resolveBaseFromChain(leaf)
		testutil.True(t, ok, "expected ok=true")
		testutil.Equal(t, BaseInteger32, got, "got")
	})

	t.Run("stops at each application type", func(t *testing.T) {
		appTypes := []BaseType{
			BaseCounter32, BaseCounter64, BaseGauge32,
			BaseUnsigned32, BaseTimeTicks, BaseIpAddress, BaseOpaque,
		}
		for _, appBase := range appTypes {
			root := newType("root")
			root.setBase(BaseInteger32)

			appType := newType("app")
			appType.setBase(appBase)
			appType.setParent(root)

			child := newType("child")
			child.setBase(appBase)
			child.setParent(appType)

			got, ok := resolveBaseFromChain(child)
			testutil.True(t, ok, "expected ok=true for")
			testutil.Equal(t, appBase, got, "for")
		}
	})

	t.Run("cycle detection", func(t *testing.T) {
		a := newType("A")
		a.setBase(BaseInteger32)

		b := newType("B")
		b.setBase(BaseInteger32)

		// Create cycle: a -> b -> a
		a.setParent(b)
		b.setParent(a)

		_, ok := resolveBaseFromChain(a)
		testutil.False(t, ok, "expected ok=false for cycle")
	})

	t.Run("self-referencing cycle", func(t *testing.T) {
		a := newType("A")
		a.setBase(BaseInteger32)
		a.setParent(a)

		_, ok := resolveBaseFromChain(a)
		testutil.False(t, ok, "expected ok=false for self-referencing cycle")
	})

	t.Run("long chain", func(t *testing.T) {
		root := newType("root")
		root.setBase(BaseOctetString)

		prev := root
		for range 10 {
			typ := newType("type")
			typ.setBase(BaseOctetString)
			typ.setParent(prev)
			prev = typ
		}

		got, ok := resolveBaseFromChain(prev)
		testutil.True(t, ok, "expected ok=true")
		testutil.Equal(t, BaseOctetString, got, "got")
	})

	t.Run("inherits base from root of chain", func(t *testing.T) {
		root := newType("OCTET STRING")
		root.setBase(BaseOctetString)

		displayString := newType("DisplayString")
		displayString.setBase(BaseInteger32) // wrong base, should be overridden
		displayString.setParent(root)

		got, ok := resolveBaseFromChain(displayString)
		testutil.True(t, ok, "expected ok=true")
		// Should return the root's base type
		testutil.Equal(t, BaseOctetString, got, "got")
	})
}

func TestResolveTypeRefParentsGraph(t *testing.T) {
	makeTypeDef := func(name, baseName string) *module.TypeDef {
		return &module.TypeDef{
			DefBase: module.DefBase{Name: name},
			Syntax:  &module.TypeSyntaxTypeRef{Name: baseName},
		}
	}

	t.Run("links local and imported parents", func(t *testing.T) {
		baseMod := module.NewModule("BASE-MIB", types.Span{})
		addDefs(baseMod, []module.Definition{
			makeTypeDef("BaseText", "INTEGER"),
		})

		userMod := module.NewModule("USER-MIB", types.Span{})
		userMod.Imports = []module.Import{
			module.NewImport("BASE-MIB", "BaseText", types.Span{}),
		}
		addDefs(userMod, []module.Definition{
			makeTypeDef("MidText", "BaseText"),
			makeTypeDef("LeafText", "MidText"),
		})

		ctx := newTestContextForModules(DefaultConfig(), baseMod, userMod)
		ctx.snmpv2SMIModule = baseMod
		ctx.defNames[baseMod] = map[string]struct{}{
			"BaseText": {},
			"INTEGER":  {},
		}
		ctx.defNames[userMod] = map[string]struct{}{
			"MidText":  {},
			"LeafText": {},
		}
		ctx.registerImport(userMod, "BaseText", baseMod)

		integerType := newType("INTEGER")
		integerType.setBase(BaseInteger32)
		ctx.registerModuleTypeSymbol(baseMod, "INTEGER", integerType)

		createUserTypes(ctx)
		resolveTypeRefParentsGraph(ctx)

		baseText, ok := ctx.resolveTypeForModule(baseMod, "BaseText")
		testutil.True(t, ok, "BaseText should exist")
		midText, ok := ctx.resolveTypeForModule(userMod, "MidText")
		testutil.True(t, ok, "MidText should exist")
		leafText, ok := ctx.resolveTypeForModule(userMod, "LeafText")
		testutil.True(t, ok, "LeafText should exist")

		testutil.Equal(t, integerType, baseText.Parent(), "BaseText parent")
		testutil.Equal(t, baseText, midText.Parent(), "MidText parent")
		testutil.Equal(t, midText, leafText.Parent(), "LeafText parent")
		testutil.Len(t, unresolvedByKind(ctx, UnresolvedType), 0, "unresolved types")
	})

	t.Run("records each cycle participant as unresolved", func(t *testing.T) {
		mod := module.NewModule("CYCLE-MIB", types.Span{})
		addDefs(mod, []module.Definition{
			makeTypeDef("TypeA", "TypeB"),
			makeTypeDef("TypeB", "TypeA"),
		})

		ctx := newTestContextForModules(DefaultConfig(), mod)
		ctx.defNames[mod] = map[string]struct{}{
			"TypeA": {},
			"TypeB": {},
		}

		createUserTypes(ctx)
		resolveTypeRefParentsGraph(ctx)

		unresolvedTypes := unresolvedByKind(ctx, UnresolvedType)
		testutil.Len(t, unresolvedTypes, 2, "unresolved types")
		symbols := map[string]struct{}{}
		for _, u := range unresolvedTypes {
			symbols[u.Symbol] = struct{}{}
		}
		_, hasA := symbols["TypeA"]
		_, hasB := symbols["TypeB"]
		testutil.True(t, hasA, "expected TypeA as unresolved symbol")
		testutil.True(t, hasB, "expected TypeB as unresolved symbol")
	})
}

func TestSeedPrimitiveTypes(t *testing.T) {
	t.Run("seeds ASN.1 primitives into SNMPv2-SMI module", func(t *testing.T) {
		smiMod := module.NewModule(moduleSNMPv2SMI, types.Span{})
		ctx := newTestContextForModules(DefaultConfig(), smiMod)
		ctx.snmpv2SMIModule = smiMod

		seedPrimitiveTypes(ctx)

		tests := []struct {
			name string
			base BaseType
		}{
			{"INTEGER", BaseInteger32},
			{"OCTET STRING", BaseOctetString},
			{"OBJECT IDENTIFIER", BaseObjectIdentifier},
			{"BITS", BaseBits},
		}
		for _, tt := range tests {
			typ, ok := ctx.resolveTypeForModule(smiMod, tt.name)
			testutil.True(t, ok, "expected primitive type to be registered")
			testutil.Equal(t, tt.base, typ.Base(), tt.name+" base")
		}
		testutil.Equal(t, 4, ctx.TypeCount(), "type count")
	})

	t.Run("without SNMPv2-SMI module does nothing", func(t *testing.T) {
		ctx := newTestContext()
		seedPrimitiveTypes(ctx)
		testutil.Equal(t, 0, ctx.TypeCount(), "type count")
	})
}

func TestFindTypeDefiningModule(t *testing.T) {
	localMod := module.NewModule("LOCAL-MIB", types.Span{})
	sourceMod := module.NewModule("SOURCE-MIB", types.Span{})
	smiMod := module.NewModule(moduleSNMPv2SMI, types.Span{})

	ctx := newTestContextForModules(DefaultConfig(), localMod, sourceMod, smiMod)
	ctx.snmpv2SMIModule = smiMod
	ctx.defNames[localMod] = map[string]struct{}{"LocalType": {}}
	ctx.registerImport(localMod, "ImportedType", sourceMod)

	t.Run("prefers local definition", func(t *testing.T) {
		got := findTypeDefiningModule(ctx, localMod, "LocalType")
		testutil.Equal(t, "LOCAL-MIB", got, "module")
	})

	t.Run("follows imports", func(t *testing.T) {
		got := findTypeDefiningModule(ctx, localMod, "ImportedType")
		testutil.Equal(t, "SOURCE-MIB", got, "module")
	})

	t.Run("falls back to well-known module", func(t *testing.T) {
		got := findTypeDefiningModule(ctx, localMod, "INTEGER")
		testutil.Equal(t, moduleSNMPv2SMI, got, "module")
	})

	t.Run("missing type returns empty string", func(t *testing.T) {
		got := findTypeDefiningModule(ctx, localMod, "MissingType")
		testutil.Equal(t, "", got, "module")
	})
}

func TestTryResolveTypeParent(t *testing.T) {
	mod := module.NewModule("TEST-MIB", types.Span{})
	ctx := newTestContextForModules(DefaultConfig(), mod)

	parent := newType("ParentType")
	parent.setBase(BaseInteger32)
	ctx.registerModuleTypeSymbol(mod, "ParentType", parent)

	child := newType("ChildType")
	entry := typeResolutionEntry{
		mod: mod,
		td: &module.TypeDef{
			DefBase: module.DefBase{Name: "ChildType"},
			Syntax:  &module.TypeSyntaxTypeRef{Name: "ParentType"},
		},
		typ: child,
	}

	t.Run("sets parent when referenced type exists", func(t *testing.T) {
		ok := tryResolveTypeParent(ctx, entry)
		testutil.True(t, ok, "expected parent resolution to succeed")
		testutil.Equal(t, parent, child.Parent(), "parent")
	})

	t.Run("returns false for unresolved type", func(t *testing.T) {
		other := newType("OtherType")
		ok := tryResolveTypeParent(ctx, typeResolutionEntry{
			mod: mod,
			td: &module.TypeDef{
				DefBase: module.DefBase{Name: "OtherType"},
				Syntax:  &module.TypeSyntaxTypeRef{Name: "MissingType"},
			},
			typ: other,
		})
		testutil.False(t, ok, "expected parent resolution to fail")
		testutil.Nil(t, other.Parent(), "parent")
	})
}

func TestLinkPrimitiveSyntaxParents(t *testing.T) {
	smiMod := module.NewModule(moduleSNMPv2SMI, types.Span{})
	userMod := module.NewModule("USER-MIB", types.Span{})
	addDefs(userMod, []module.Definition{
		&module.TypeDef{
			DefBase: module.DefBase{Name: "MyOctets"},
			Syntax:  &module.TypeSyntaxOctetString{},
		},
		&module.TypeDef{
			DefBase: module.DefBase{Name: "MyEnum"},
			Syntax:  &module.TypeSyntaxIntegerEnum{NamedNumbers: []module.NamedNumber{{Name: "up", Value: 1}}},
		},
		&module.TypeDef{
			DefBase: module.DefBase{Name: "MyBits"},
			Syntax:  &module.TypeSyntaxBits{NamedBits: []module.NamedBit{{Name: "flag", Position: 0}}},
		},
		&module.TypeDef{
			DefBase: module.DefBase{Name: "MyOid"},
			Syntax:  &module.TypeSyntaxObjectIdentifier{},
		},
	})

	ctx := newTestContextForModules(DefaultConfig(), smiMod, userMod)
	ctx.snmpv2SMIModule = smiMod
	seedPrimitiveTypes(ctx)
	createUserTypes(ctx)

	linkPrimitiveSyntaxParents(ctx)

	cases := []struct {
		name       string
		parentName string
	}{
		{"MyOctets", "OCTET STRING"},
		{"MyEnum", "INTEGER"},
		{"MyBits", "BITS"},
		{"MyOid", "OBJECT IDENTIFIER"},
	}
	for _, tc := range cases {
		typ, ok := ctx.resolveTypeForModule(userMod, tc.name)
		testutil.True(t, ok, "expected type to exist")
		parent, ok := ctx.resolveType(tc.parentName)
		testutil.True(t, ok, "expected primitive parent to exist")
		testutil.Equal(t, parent, typ.Parent(), tc.name+" parent")
	}

	t.Run("existing parent is preserved", func(t *testing.T) {
		mod := module.NewModule("KEEP-MIB", types.Span{})
		addDefs(mod, []module.Definition{
			&module.TypeDef{
				DefBase: module.DefBase{Name: "AlreadyLinked"},
				Syntax:  &module.TypeSyntaxOctetString{},
			},
		})
		ctx := newTestContextForModules(DefaultConfig(), smiMod, mod)
		ctx.snmpv2SMIModule = smiMod
		seedPrimitiveTypes(ctx)
		createUserTypes(ctx)

		typ, _ := ctx.resolveTypeForModule(mod, "AlreadyLinked")
		customParent := newType("CustomParent")
		typ.setParent(customParent)

		linkPrimitiveSyntaxParents(ctx)

		testutil.Equal(t, customParent, typ.Parent(), "parent")
	})
}

func TestLinkRFC1213TypesToTCs(t *testing.T) {
	rfc1213 := module.NewModule("RFC1213-MIB", types.Span{})
	addDefs(rfc1213, []module.Definition{
		&module.TypeDef{DefBase: module.DefBase{Name: "DisplayString"}, Syntax: &module.TypeSyntaxTypeRef{Name: "OCTET STRING"}},
		&module.TypeDef{DefBase: module.DefBase{Name: "PhysAddress"}, Syntax: &module.TypeSyntaxTypeRef{Name: "OCTET STRING"}},
	})
	snmpv2tc := module.NewModule(moduleSNMPv2TC, types.Span{})
	addDefs(snmpv2tc, []module.Definition{
		&module.TypeDef{DefBase: module.DefBase{Name: "DisplayString"}, Syntax: &module.TypeSyntaxTypeRef{Name: "OCTET STRING"}},
		&module.TypeDef{DefBase: module.DefBase{Name: "PhysAddress"}, Syntax: &module.TypeSyntaxTypeRef{Name: "OCTET STRING"}},
	})

	ctx := newTestContextForModules(DefaultConfig(), rfc1213, snmpv2tc)
	createUserTypes(ctx)

	linkRFC1213TypesToTCs(ctx)

	rfcDisplay, _ := ctx.resolveTypeForModule(rfc1213, "DisplayString")
	tcDisplay, _ := ctx.resolveTypeForModule(snmpv2tc, "DisplayString")
	testutil.Equal(t, tcDisplay, rfcDisplay.Parent(), "DisplayString parent")

	rfcPhys, _ := ctx.resolveTypeForModule(rfc1213, "PhysAddress")
	tcPhys, _ := ctx.resolveTypeForModule(snmpv2tc, "PhysAddress")
	testutil.Equal(t, tcPhys, rfcPhys.Parent(), "PhysAddress parent")
}

func TestInheritBaseTypes(t *testing.T) {
	t.Run("inherits base type from parent chain", func(t *testing.T) {
		ctx := newTestContext()

		root := newType("OCTET STRING")
		root.setBase(BaseOctetString)
		mid := newType("DisplayString")
		mid.setBase(BaseUnknown)
		mid.setParent(root)
		leaf := newType("VendorString")
		leaf.setBase(BaseUnknown)
		leaf.setParent(mid)

		ctx.mib.addType(root)
		ctx.mib.addType(mid)
		ctx.mib.addType(leaf)

		inheritBaseTypes(ctx)

		testutil.Equal(t, BaseOctetString, mid.Base(), "mid base")
		testutil.Equal(t, BaseOctetString, leaf.Base(), "leaf base")
	})

	t.Run("application base types are preserved", func(t *testing.T) {
		ctx := newTestContext()

		root := newType("INTEGER")
		root.setBase(BaseInteger32)
		app := newType("Counter32")
		app.setBase(BaseCounter32)
		app.setParent(root)

		ctx.mib.addType(root)
		ctx.mib.addType(app)

		inheritBaseTypes(ctx)

		testutil.Equal(t, BaseCounter32, app.Base(), "application base")
	})
}

func TestCreateUserTypes_Reference(t *testing.T) {
	// A TEXTUAL-CONVENTION with a REFERENCE clause should have its
	// Reference() populated on the resolved Type.
	mod := module.NewModule("REF-TEST-MIB", types.Span{})
	mod.Language = types.LanguageSMIv2
	addDefs(mod, []module.Definition{
		&module.TypeDef{
			DefBase:             module.DefBase{Name: "MyTC"},
			Syntax:              &module.TypeSyntaxTypeRef{Name: "DisplayString"},
			Status:              types.StatusCurrent,
			Description:         "A test textual convention",
			Reference:           "RFC 1234, Section 5",
			IsTextualConvention: true,
		},
	})

	m := Resolve([]*module.Module{mod}, nil, nil, nil)
	testutil.NotNil(t, m, "Resolve returned nil Mib")

	typ := m.Type("MyTC")
	testutil.NotNil(t, typ, "type MyTC not found after resolution")

	testutil.Equal(t, "RFC 1234, Section 5", typ.Reference(), "Type.Reference()")
}
