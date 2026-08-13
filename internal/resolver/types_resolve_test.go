package resolver

import (
	"math"
	"testing"

	"github.com/golangsnmp/gomib/internal/cst/lower"
	cstparser "github.com/golangsnmp/gomib/internal/cst/parser"
	"github.com/golangsnmp/gomib/internal/model"
	"github.com/golangsnmp/gomib/internal/module"
	"github.com/golangsnmp/gomib/internal/testutil"
	"github.com/golangsnmp/gomib/internal/types"
)

func TestSyntaxToBaseType(t *testing.T) {
	tests := []struct {
		name   string
		syntax module.TypeSyntax
		want   model.BaseType
		wantOK bool
	}{
		// TypeRef variants - known base types
		{"typeref Integer32", &module.TypeSyntaxTypeRef{Name: "Integer32"}, model.BaseInteger32, true},
		{"typeref INTEGER", &module.TypeSyntaxTypeRef{Name: "INTEGER"}, model.BaseInteger32, true},
		{"typeref Counter32", &module.TypeSyntaxTypeRef{Name: "Counter32"}, model.BaseCounter32, true},
		{"typeref Counter64", &module.TypeSyntaxTypeRef{Name: "Counter64"}, model.BaseCounter64, true},
		{"typeref Gauge32", &module.TypeSyntaxTypeRef{Name: "Gauge32"}, model.BaseGauge32, true},
		{"typeref Unsigned32", &module.TypeSyntaxTypeRef{Name: "Unsigned32"}, model.BaseUnsigned32, true},
		{"typeref TimeTicks", &module.TypeSyntaxTypeRef{Name: "TimeTicks"}, model.BaseTimeTicks, true},
		{"typeref IpAddress", &module.TypeSyntaxTypeRef{Name: "IpAddress"}, model.BaseIpAddress, true},
		{"typeref Opaque", &module.TypeSyntaxTypeRef{Name: "Opaque"}, model.BaseOpaque, true},
		{"typeref OCTET STRING", &module.TypeSyntaxTypeRef{Name: "OCTET STRING"}, model.BaseOctetString, true},
		{"typeref OBJECT IDENTIFIER", &module.TypeSyntaxTypeRef{Name: "OBJECT IDENTIFIER"}, model.BaseObjectIdentifier, true},
		{"typeref BITS", &module.TypeSyntaxTypeRef{Name: "BITS"}, model.BaseBits, true},

		// TypeRef - unknown name (user-defined type)
		{"typeref DisplayString", &module.TypeSyntaxTypeRef{Name: "DisplayString"}, 0, false},
		{"typeref unknown", &module.TypeSyntaxTypeRef{Name: "MyCustomType"}, 0, false},

		// Primitive syntax types
		{"IntegerEnum", &module.TypeSyntaxIntegerEnum{
			NamedNumbers: []module.NamedNumber{{Name: "up", Value: 1}},
		}, model.BaseInteger32, true},
		{"Bits", &module.TypeSyntaxBits{
			NamedBits: []module.NamedBit{{Name: "flag0", Position: 0}},
		}, model.BaseBits, true},
		{"OctetString", &module.TypeSyntaxOctetString{}, model.BaseOctetString, true},
		{"ObjectIdentifier", &module.TypeSyntaxObjectIdentifier{}, model.BaseObjectIdentifier, true},

		// Constrained wrapping - delegates to inner syntax
		{"constrained Integer32", &module.TypeSyntaxConstrained{
			Base:       &module.TypeSyntaxTypeRef{Name: "Integer32"},
			Constraint: &module.ConstraintRange{Ranges: []module.Range{module.NewRangeSigned(0, 100, types.Span{})}},
		}, model.BaseInteger32, true},
		{"constrained OCTET STRING", &module.TypeSyntaxConstrained{
			Base:       &module.TypeSyntaxTypeRef{Name: "OCTET STRING"},
			Constraint: &module.ConstraintSize{Ranges: []module.Range{module.NewRangeUnsigned(0, 255, types.Span{})}},
		}, model.BaseOctetString, true},
		{"constrained OctetString primitive", &module.TypeSyntaxConstrained{
			Base:       &module.TypeSyntaxOctetString{},
			Constraint: &module.ConstraintSize{Ranges: []module.Range{module.NewRangeUnsigned(0, 255, types.Span{})}},
		}, model.BaseOctetString, true},
		{"constrained ObjectIdentifier primitive", &module.TypeSyntaxConstrained{
			Base:       &module.TypeSyntaxObjectIdentifier{},
			Constraint: &module.ConstraintSize{},
		}, model.BaseObjectIdentifier, true},
		{"constrained unknown typeref", &module.TypeSyntaxConstrained{
			Base:       &module.TypeSyntaxTypeRef{Name: "DisplayString"},
			Constraint: &module.ConstraintSize{},
		}, 0, false},

		// SequenceOf and Sequence resolve to model.BaseSequence
		{"SequenceOf", &module.TypeSyntaxSequenceOf{EntryType: "FooEntry"}, model.BaseSequence, true},
		{"Sequence", &module.TypeSyntaxSequence{Fields: []module.SequenceField{}}, model.BaseSequence, true},

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
		base model.BaseType
		want bool
	}{
		{"Counter32", model.BaseCounter32, true},
		{"Counter64", model.BaseCounter64, true},
		{"Gauge32", model.BaseGauge32, true},
		{"Unsigned32", model.BaseUnsigned32, true},
		{"TimeTicks", model.BaseTimeTicks, true},
		{"IpAddress", model.BaseIpAddress, true},
		{"Opaque", model.BaseOpaque, true},
		{"Integer32", model.BaseInteger32, false},
		{"OctetString", model.BaseOctetString, false},
		{"ObjectIdentifier", model.BaseObjectIdentifier, false},
		{"Bits", model.BaseBits, false},
		{"Sequence", model.BaseSequence, false},
		{"Unknown", model.BaseUnknown, false},
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

func assertNamedValue(t *testing.T, nv model.NamedValue, wantLabel string, wantValue int64) {
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
		testutil.Equal(t, model.NewUnsignedRangeBound(0), sizes[0].Min, "min")
		testutil.Equal(t, model.NewUnsignedRangeBound(255), sizes[0].Max, "max")
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
		testutil.Equal(t, model.NewSignedRangeBound(-128), ranges[0].Min, "min")
		testutil.Equal(t, model.NewSignedRangeBound(127), ranges[0].Max, "max")
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
		testutil.Equal(t, model.NewUnsignedRangeBound(0), sizes[0].Min, "sizes[0] min")
		testutil.Equal(t, model.NewUnsignedRangeBound(0), sizes[0].Max, "sizes[0] max")
		testutil.Equal(t, model.NewUnsignedRangeBound(4), sizes[1].Min, "sizes[1] min")
		testutil.Equal(t, model.NewUnsignedRangeBound(255), sizes[1].Max, "sizes[1] max")
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
		testutil.Equal(t, model.NewSignedRangeBound(-100), got[0].Min, "min")
		testutil.Equal(t, model.NewSignedRangeBound(100), got[0].Max, "max")
	})

	t.Run("single value range", func(t *testing.T) {
		ranges := []module.Range{
			module.NewRangeSingleSigned(42, types.Span{}),
		}
		got := rangesToConstraint(ranges)
		testutil.Len(t, got, 1, "ranges")
		// Single value: Max is nil, so max = min
		testutil.Equal(t, model.NewSignedRangeBound(42), got[0].Min, "min")
		testutil.Equal(t, model.NewSignedRangeBound(42), got[0].Max, "max")
	})

	t.Run("multiple ranges", func(t *testing.T) {
		ranges := []module.Range{
			module.NewRangeSigned(0, 10, types.Span{}),
			module.NewRangeSigned(100, 200, types.Span{}),
		}
		got := rangesToConstraint(ranges)
		testutil.Len(t, got, 2, "ranges")
		testutil.Equal(t, model.NewSignedRangeBound(0), got[0].Min, "got[0] min")
		testutil.Equal(t, model.NewSignedRangeBound(10), got[0].Max, "got[0] max")
		testutil.Equal(t, model.NewSignedRangeBound(100), got[1].Min, "got[1] min")
		testutil.Equal(t, model.NewSignedRangeBound(200), got[1].Max, "got[1] max")
	})

	t.Run("raw endpoints", func(t *testing.T) {
		ranges := []module.Range{{
			Min: &module.RangeValueRaw{Value: "'0G'H"},
			Max: &module.RangeValueRaw{Value: "'10000000000000000'H"},
		}}
		got := rangesToConstraint(ranges)
		testutil.Len(t, got, 1, "ranges")
		testutil.Equal(t, model.NewRawRangeBound("'0G'H"), got[0].Min, "raw min")
		testutil.Equal(t, model.NewRawRangeBound("'10000000000000000'H"), got[0].Max, "raw max")
		testutil.False(t, got[0].IsResolved(), "raw range should remain unresolved")
	})

	t.Run("empty ranges", func(t *testing.T) {
		got := rangesToConstraint(nil)
		testutil.Len(t, got, 0, "ranges")
	})
}

func TestResolveHexRangeLiterals(t *testing.T) {
	source := []byte(`
TEST-MIB DEFINITIONS ::= BEGIN
Valid ::= INTEGER ('7f ff'H..'80 00'H)
Invalid ::= INTEGER ('0G'H..'10000000000000000'H)
END
`)
	cfg := types.DefaultConfig()
	p := cstparser.New(source, nil, cfg)
	mods := lower.Lower(p.ParseModule(), source, p.Diagnostics(), cfg)
	m := Resolve(mods, nil, nil, &cfg)

	valid := m.Module("TEST-MIB").Type("Valid").Ranges()
	testutil.Len(t, valid, 1, "valid ranges")
	testutil.Equal(t, model.NewUnsignedRangeBound(0x7fff), valid[0].Min, "valid min")
	testutil.Equal(t, model.NewUnsignedRangeBound(0x8000), valid[0].Max, "valid max")
	testutil.True(t, valid[0].IsResolved(), "valid range should resolve")

	invalid := m.Module("TEST-MIB").Type("Invalid").Ranges()
	testutil.Len(t, invalid, 1, "invalid ranges")
	testutil.Equal(t, model.NewRawRangeBound("'0G'H"), invalid[0].Min, "invalid raw min")
	testutil.Equal(t, model.NewRawRangeBound("'10000000000000000'H"), invalid[0].Max, "overflow raw max")
	testutil.False(t, invalid[0].IsResolved(), "invalid range should remain unresolved")

	var invalidDiags int
	for _, diag := range m.Diagnostics() {
		if diag.Code == types.DiagInvalidHexRange {
			invalidDiags++
		}
	}
	testutil.Equal(t, 2, invalidDiags, "invalid hex range diagnostics")
}

func TestRangeValueToConstraintPreservesKind(t *testing.T) {
	tests := []struct {
		name string
		val  module.RangeValue
		want model.RangeBound
	}{
		{"signed", &module.RangeValueSigned{Value: -100}, model.NewSignedRangeBound(-100)},
		{"unsigned", &module.RangeValueUnsigned{Value: math.MaxUint64}, model.NewUnsignedRangeBound(math.MaxUint64)},
		{"MIN", &module.RangeValueMin{}, model.RangeBound{Kind: model.RangeBoundMin}},
		{"MAX", &module.RangeValueMax{}, model.RangeBound{Kind: model.RangeBoundMax}},
		{"raw", &module.RangeValueRaw{Value: "vendor"}, model.NewRawRangeBound("vendor")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testutil.Equal(t, tt.want, rangeValueToConstraint(tt.val), "rangeValueToConstraint()")
		})
	}
}

func TestEffectiveRangeSemantics(t *testing.T) {
	min := model.RangeBound{Kind: model.RangeBoundMin}
	max := model.RangeBound{Kind: model.RangeBoundMax}
	span := types.Span{}
	parent := model.NewType("Parent")
	model.SetTypeBase(parent, model.BaseInteger32)
	model.SetTypeRanges(parent, []model.Range{{
		Min: model.NewSignedRangeBound(-10), Max: model.NewSignedRangeBound(20), Span: span,
	}})
	child := model.NewType("Child")
	model.SetTypeBase(child, model.BaseInteger32)
	model.SetTypeParent(child, parent)
	model.SetTypeRanges(child, []model.Range{{Min: min, Max: max, Span: span}})

	direct := child.Ranges()
	testutil.Equal(t, min, direct[0].Min, "direct MIN")
	testutil.Equal(t, max, direct[0].Max, "direct MAX")
	effective := child.EffectiveRanges()
	testutil.Len(t, effective, 1, "effective intersection")
	testutil.Equal(t, model.NewSignedRangeBound(-10), effective[0].Min, "effective min")
	testutil.Equal(t, model.NewSignedRangeBound(20), effective[0].Max, "effective max")

	disjoint := model.NewType("Disjoint")
	model.SetTypeBase(disjoint, model.BaseInteger32)
	model.SetTypeParent(disjoint, parent)
	model.SetTypeRanges(disjoint, []model.Range{{
		Min: model.NewSignedRangeBound(30), Max: model.NewSignedRangeBound(40), Span: span,
	}})
	testutil.Len(t, disjoint.EffectiveRanges(), 0, "disjoint intersection")
	testutil.True(t, disjoint.EffectiveRangesConstrained(), "disjoint remains constrained")

	descendant := model.NewType("Descendant")
	model.SetTypeBase(descendant, model.BaseInteger32)
	model.SetTypeParent(descendant, disjoint)
	model.SetTypeRanges(descendant, []model.Range{{
		Min: model.NewSignedRangeBound(35), Max: model.NewSignedRangeBound(36), Span: span,
	}})
	testutil.Len(t, descendant.EffectiveRanges(), 0, "empty ancestor intersection remains empty")
	testutil.True(t, descendant.EffectiveRangesConstrained(), "empty descendant remains constrained")

	multi := model.NewType("Multi")
	model.SetTypeBase(multi, model.BaseInteger32)
	model.SetTypeRanges(multi, []model.Range{
		{Min: model.NewSignedRangeBound(0), Max: model.NewSignedRangeBound(10), Span: span},
		{Min: model.NewSignedRangeBound(20), Max: model.NewSignedRangeBound(30), Span: span},
	})
	maxOnly := model.NewType("MaxOnly")
	model.SetTypeBase(maxOnly, model.BaseInteger32)
	model.SetTypeParent(maxOnly, multi)
	model.SetTypeRanges(maxOnly, []model.Range{{Min: max, Max: max, Span: span}})
	effective = maxOnly.EffectiveRanges()
	testutil.Len(t, effective, 1, "MAX is not a false exact sentinel match")
	testutil.Equal(t, model.NewSignedRangeBound(30), effective[0].Min, "MAX resolves to parent maximum")
	testutil.Equal(t, model.NewSignedRangeBound(30), effective[0].Max, "MAX resolves to parent maximum")
}

func TestResolveBaseFromChain(t *testing.T) {
	t.Run("no parent returns own base", func(t *testing.T) {
		typ := model.NewType("MyType")
		model.SetTypeBase(typ, model.BaseOctetString)

		got, ok := resolveBaseFromChain(typ)
		testutil.True(t, ok, "expected ok=true")
		testutil.Equal(t, model.BaseOctetString, got, "got")
	})

	t.Run("walks chain to root", func(t *testing.T) {
		root := model.NewType("INTEGER")
		model.SetTypeBase(root, model.BaseInteger32)

		mid := model.NewType("MyInt")
		model.SetTypeBase(mid, model.BaseInteger32)
		model.SetTypeParent(mid, root)

		leaf := model.NewType("MySpecificInt")
		model.SetTypeBase(leaf, model.BaseInteger32)
		model.SetTypeParent(leaf, mid)

		got, ok := resolveBaseFromChain(leaf)
		testutil.True(t, ok, "expected ok=true")
		testutil.Equal(t, model.BaseInteger32, got, "got")
	})

	t.Run("stops at each application type", func(t *testing.T) {
		appTypes := []model.BaseType{
			model.BaseCounter32, model.BaseCounter64, model.BaseGauge32,
			model.BaseUnsigned32, model.BaseTimeTicks, model.BaseIpAddress, model.BaseOpaque,
		}
		for _, appBase := range appTypes {
			root := model.NewType("root")
			model.SetTypeBase(root, model.BaseInteger32)

			appType := model.NewType("app")
			model.SetTypeBase(appType, appBase)
			model.SetTypeParent(appType, root)

			child := model.NewType("child")
			model.SetTypeBase(child, appBase)
			model.SetTypeParent(child, appType)

			got, ok := resolveBaseFromChain(child)
			testutil.True(t, ok, "expected ok=true for")
			testutil.Equal(t, appBase, got, "for")
		}
	})

	t.Run("cycle detection", func(t *testing.T) {
		a := model.NewType("A")
		model.SetTypeBase(a, model.BaseInteger32)

		b := model.NewType("B")
		model.SetTypeBase(b, model.BaseInteger32)

		// Create cycle: a -> b -> a
		model.SetTypeParent(a, b)
		model.SetTypeParent(b, a)

		_, ok := resolveBaseFromChain(a)
		testutil.False(t, ok, "expected ok=false for cycle")
	})

	t.Run("self-referencing cycle", func(t *testing.T) {
		a := model.NewType("A")
		model.SetTypeBase(a, model.BaseInteger32)
		model.SetTypeParent(a, a)

		_, ok := resolveBaseFromChain(a)
		testutil.False(t, ok, "expected ok=false for self-referencing cycle")
	})

	t.Run("long chain", func(t *testing.T) {
		root := model.NewType("root")
		model.SetTypeBase(root, model.BaseOctetString)

		prev := root
		for range 10 {
			typ := model.NewType("type")
			model.SetTypeBase(typ, model.BaseOctetString)
			model.SetTypeParent(typ, prev)
			prev = typ
		}

		got, ok := resolveBaseFromChain(prev)
		testutil.True(t, ok, "expected ok=true")
		testutil.Equal(t, model.BaseOctetString, got, "got")
	})

	t.Run("inherits base from root of chain", func(t *testing.T) {
		root := model.NewType("OCTET STRING")
		model.SetTypeBase(root, model.BaseOctetString)

		displayString := model.NewType("DisplayString")
		model.SetTypeBase(displayString, model.BaseInteger32) // wrong base, should be overridden
		model.SetTypeParent(displayString, root)

		got, ok := resolveBaseFromChain(displayString)
		testutil.True(t, ok, "expected ok=true")
		// Should return the root's base type
		testutil.Equal(t, model.BaseOctetString, got, "got")
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

		ctx := newTestContextForModules(model.DefaultConfig(), baseMod, userMod)
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

		integerType := model.NewType("INTEGER")
		model.SetTypeBase(integerType, model.BaseInteger32)
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
		testutil.Len(t, unresolvedByKind(ctx, model.UnresolvedType), 0, "unresolved types")
	})

	t.Run("two-node cycle keeps nil parents and records cycle metadata", func(t *testing.T) {
		mod := module.NewModule("CYCLE-MIB", types.Span{})
		addDefs(mod, []module.Definition{
			makeTypeDef("TypeA", "TypeB"),
			makeTypeDef("TypeB", "TypeA"),
		})

		ctx := newTestContextForModules(model.DefaultConfig(), mod)
		ctx.defNames[mod] = map[string]struct{}{
			"TypeA": {},
			"TypeB": {},
		}

		createUserTypes(ctx)
		resolveTypeRefParentsGraph(ctx)

		for _, name := range []string{"TypeA", "TypeB"} {
			typ, ok := ctx.resolveTypeForModule(mod, name)
			testutil.True(t, ok, name+" should exist")
			testutil.Nil(t, typ.Parent(), name+" parent")
		}

		unresolvedTypes := unresolvedByKind(ctx, model.UnresolvedType)
		testutil.Len(t, unresolvedTypes, 2, "unresolved types")
		reasonsBySymbol := make(map[string]string, len(unresolvedTypes))
		for _, u := range unresolvedTypes {
			reasonsBySymbol[u.Symbol] = u.Reason
		}
		testutil.Equal(t, reasonDependencyCycle, reasonsBySymbol["TypeA"], "TypeA unresolved reason")
		testutil.Equal(t, reasonDependencyCycle, reasonsBySymbol["TypeB"], "TypeB unresolved reason")

		testutil.Equal(t, 2, countDiagnostics(ctx.Diagnostics(), types.DiagTypeCycle), "type-cycle diagnostics")
		noDiag(t, ctx.Diagnostics(), types.DiagTypeUnknown)
		for _, diag := range ctx.Diagnostics() {
			testutil.Contains(t, diag.Message, "dependency cycle", "cycle diagnostic message")
		}
	})

	t.Run("self cycle keeps nil parent and records cycle metadata", func(t *testing.T) {
		mod := module.NewModule("SELF-CYCLE-MIB", types.Span{})
		addDefs(mod, []module.Definition{
			makeTypeDef("SelfType", "SelfType"),
		})

		ctx := newTestContextForModules(model.DefaultConfig(), mod)
		ctx.defNames[mod] = map[string]struct{}{"SelfType": {}}

		createUserTypes(ctx)
		resolveTypeRefParentsGraph(ctx)

		selfType, ok := ctx.resolveTypeForModule(mod, "SelfType")
		testutil.True(t, ok, "SelfType should exist")
		testutil.Nil(t, selfType.Parent(), "SelfType parent")

		unresolvedTypes := unresolvedByKind(ctx, model.UnresolvedType)
		testutil.Len(t, unresolvedTypes, 1, "unresolved types")
		testutil.Equal(t, "SelfType", unresolvedTypes[0].Symbol, "unresolved symbol")
		testutil.Equal(t, reasonDependencyCycle, unresolvedTypes[0].Reason, "unresolved reason")

		diag := hasDiag(t, ctx.Diagnostics(), types.DiagTypeCycle)
		testutil.Contains(t, diag.Message, "references itself", "self-cycle diagnostic message")
		noDiag(t, ctx.Diagnostics(), types.DiagTypeUnknown)
	})
}

func TestSeedPrimitiveTypes(t *testing.T) {
	t.Run("seeds ASN.1 primitives into SNMPv2-SMI module", func(t *testing.T) {
		smiMod := module.NewModule(moduleSNMPv2SMI, types.Span{})
		ctx := newTestContextForModules(model.DefaultConfig(), smiMod)
		ctx.snmpv2SMIModule = smiMod

		seedPrimitiveTypes(ctx)

		tests := []struct {
			name string
			base model.BaseType
		}{
			{"INTEGER", model.BaseInteger32},
			{"OCTET STRING", model.BaseOctetString},
			{"OBJECT IDENTIFIER", model.BaseObjectIdentifier},
			{"BITS", model.BaseBits},
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

	ctx := newTestContextForModules(model.DefaultConfig(), localMod, sourceMod, smiMod)
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
	ctx := newTestContextForModules(model.DefaultConfig(), mod)

	parent := model.NewType("ParentType")
	model.SetTypeBase(parent, model.BaseInteger32)
	ctx.registerModuleTypeSymbol(mod, "ParentType", parent)

	child := model.NewType("ChildType")
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
		other := model.NewType("OtherType")
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

	ctx := newTestContextForModules(model.DefaultConfig(), smiMod, userMod)
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
		ctx := newTestContextForModules(model.DefaultConfig(), smiMod, mod)
		ctx.snmpv2SMIModule = smiMod
		seedPrimitiveTypes(ctx)
		createUserTypes(ctx)

		typ, _ := ctx.resolveTypeForModule(mod, "AlreadyLinked")
		customParent := model.NewType("CustomParent")
		model.SetTypeParent(typ, customParent)

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

	ctx := newTestContextForModules(model.DefaultConfig(), rfc1213, snmpv2tc)
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

		root := model.NewType("OCTET STRING")
		model.SetTypeBase(root, model.BaseOctetString)
		mid := model.NewType("DisplayString")
		model.SetTypeBase(mid, model.BaseUnknown)
		model.SetTypeParent(mid, root)
		leaf := model.NewType("VendorString")
		model.SetTypeBase(leaf, model.BaseUnknown)
		model.SetTypeParent(leaf, mid)

		model.AddMibType(ctx.mib, root)
		model.AddMibType(ctx.mib, mid)
		model.AddMibType(ctx.mib, leaf)

		inheritBaseTypes(ctx)

		testutil.Equal(t, model.BaseOctetString, mid.Base(), "mid base")
		testutil.Equal(t, model.BaseOctetString, leaf.Base(), "leaf base")
	})

	t.Run("application base types are preserved", func(t *testing.T) {
		ctx := newTestContext()

		root := model.NewType("INTEGER")
		model.SetTypeBase(root, model.BaseInteger32)
		app := model.NewType("Counter32")
		model.SetTypeBase(app, model.BaseCounter32)
		model.SetTypeParent(app, root)

		model.AddMibType(ctx.mib, root)
		model.AddMibType(ctx.mib, app)

		inheritBaseTypes(ctx)

		testutil.Equal(t, model.BaseCounter32, app.Base(), "application base")
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

	m := resolveWithBase([]*module.Module{mod}, nil, nil)
	testutil.NotNil(t, m, "Resolve returned nil Mib")

	typ := m.Type("MyTC")
	testutil.NotNil(t, typ, "type MyTC not found after resolution")

	testutil.Equal(t, "RFC 1234, Section 5", typ.Reference(), "Type.Reference()")
}
