package resolver

import (
	"math"
	"testing"

	"github.com/golangsnmp/gomib/internal/mibimpl"
	"github.com/golangsnmp/gomib/internal/module"
	"github.com/golangsnmp/gomib/mib"
)

func TestSyntaxToBaseType(t *testing.T) {
	tests := []struct {
		name   string
		syntax module.TypeSyntax
		want   mib.BaseType
		wantOK bool
	}{
		// TypeRef variants - known base types
		{"typeref Integer32", &module.TypeSyntaxTypeRef{Name: "Integer32"}, mib.BaseInteger32, true},
		{"typeref INTEGER", &module.TypeSyntaxTypeRef{Name: "INTEGER"}, mib.BaseInteger32, true},
		{"typeref Counter32", &module.TypeSyntaxTypeRef{Name: "Counter32"}, mib.BaseCounter32, true},
		{"typeref Counter64", &module.TypeSyntaxTypeRef{Name: "Counter64"}, mib.BaseCounter64, true},
		{"typeref Gauge32", &module.TypeSyntaxTypeRef{Name: "Gauge32"}, mib.BaseGauge32, true},
		{"typeref Unsigned32", &module.TypeSyntaxTypeRef{Name: "Unsigned32"}, mib.BaseUnsigned32, true},
		{"typeref TimeTicks", &module.TypeSyntaxTypeRef{Name: "TimeTicks"}, mib.BaseTimeTicks, true},
		{"typeref IpAddress", &module.TypeSyntaxTypeRef{Name: "IpAddress"}, mib.BaseIpAddress, true},
		{"typeref Opaque", &module.TypeSyntaxTypeRef{Name: "Opaque"}, mib.BaseOpaque, true},
		{"typeref OCTET STRING", &module.TypeSyntaxTypeRef{Name: "OCTET STRING"}, mib.BaseOctetString, true},
		{"typeref OBJECT IDENTIFIER", &module.TypeSyntaxTypeRef{Name: "OBJECT IDENTIFIER"}, mib.BaseObjectIdentifier, true},
		{"typeref BITS", &module.TypeSyntaxTypeRef{Name: "BITS"}, mib.BaseBits, true},

		// TypeRef - unknown name (user-defined type)
		{"typeref DisplayString", &module.TypeSyntaxTypeRef{Name: "DisplayString"}, 0, false},
		{"typeref unknown", &module.TypeSyntaxTypeRef{Name: "MyCustomType"}, 0, false},

		// Primitive syntax types
		{"IntegerEnum", &module.TypeSyntaxIntegerEnum{
			NamedNumbers: []module.NamedNumber{{Name: "up", Value: 1}},
		}, mib.BaseInteger32, true},
		{"Bits", &module.TypeSyntaxBits{
			NamedBits: []module.NamedBit{{Name: "flag0", Position: 0}},
		}, mib.BaseBits, true},
		{"OctetString", &module.TypeSyntaxOctetString{}, mib.BaseOctetString, true},
		{"ObjectIdentifier", &module.TypeSyntaxObjectIdentifier{}, mib.BaseObjectIdentifier, true},

		// Constrained wrapping - delegates to inner syntax
		{"constrained Integer32", &module.TypeSyntaxConstrained{
			Base:       &module.TypeSyntaxTypeRef{Name: "Integer32"},
			Constraint: &module.ConstraintRange{Ranges: []module.Range{module.NewRangeSigned(0, 100)}},
		}, mib.BaseInteger32, true},
		{"constrained OCTET STRING", &module.TypeSyntaxConstrained{
			Base:       &module.TypeSyntaxTypeRef{Name: "OCTET STRING"},
			Constraint: &module.ConstraintSize{Ranges: []module.Range{module.NewRangeUnsigned(0, 255)}},
		}, mib.BaseOctetString, true},
		{"constrained OctetString primitive", &module.TypeSyntaxConstrained{
			Base:       &module.TypeSyntaxOctetString{},
			Constraint: &module.ConstraintSize{Ranges: []module.Range{module.NewRangeUnsigned(0, 255)}},
		}, mib.BaseOctetString, true},
		{"constrained ObjectIdentifier primitive", &module.TypeSyntaxConstrained{
			Base:       &module.TypeSyntaxObjectIdentifier{},
			Constraint: &module.ConstraintSize{},
		}, mib.BaseObjectIdentifier, true},
		{"constrained unknown typeref", &module.TypeSyntaxConstrained{
			Base:       &module.TypeSyntaxTypeRef{Name: "DisplayString"},
			Constraint: &module.ConstraintSize{},
		}, 0, false},

		// SequenceOf and Sequence have no base type
		{"SequenceOf", &module.TypeSyntaxSequenceOf{EntryType: "FooEntry"}, 0, false},
		{"Sequence", &module.TypeSyntaxSequence{Fields: []module.SequenceField{}}, 0, false},

		// nil syntax
		{"nil", nil, 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, gotOK := syntaxToBaseType(tt.syntax)
			if got != tt.want || gotOK != tt.wantOK {
				t.Errorf("syntaxToBaseType() = (%v, %v), want (%v, %v)",
					got, gotOK, tt.want, tt.wantOK)
			}
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
			if got != tt.want {
				t.Errorf("getTypeRefBaseName() = %q, want %q", got, tt.want)
			}
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
			if got != tt.want {
				t.Errorf("getPrimitiveParentName() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestIsApplicationBaseType(t *testing.T) {
	tests := []struct {
		name string
		base mib.BaseType
		want bool
	}{
		{"Counter32", mib.BaseCounter32, true},
		{"Counter64", mib.BaseCounter64, true},
		{"Gauge32", mib.BaseGauge32, true},
		{"Unsigned32", mib.BaseUnsigned32, true},
		{"TimeTicks", mib.BaseTimeTicks, true},
		{"IpAddress", mib.BaseIpAddress, true},
		{"Opaque", mib.BaseOpaque, true},
		{"Integer32", mib.BaseInteger32, false},
		{"OctetString", mib.BaseOctetString, false},
		{"ObjectIdentifier", mib.BaseObjectIdentifier, false},
		{"Bits", mib.BaseBits, false},
		{"Sequence", mib.BaseSequence, false},
		{"Unknown", mib.BaseUnknown, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isApplicationBaseType(tt.base)
			if got != tt.want {
				t.Errorf("isApplicationBaseType(%v) = %v, want %v", tt.base, got, tt.want)
			}
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
		if len(got) != 3 {
			t.Fatalf("got %d named values, want 3", len(got))
		}
		assertNamedValue(t, got[0], "up", 1)
		assertNamedValue(t, got[1], "down", 2)
		assertNamedValue(t, got[2], "testing", 3)
	})

	t.Run("IntegerEnum empty", func(t *testing.T) {
		syntax := &module.TypeSyntaxIntegerEnum{}
		got := extractNamedValues(syntax)
		if len(got) != 0 {
			t.Errorf("got %d named values, want 0", len(got))
		}
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
		if len(got) != 3 {
			t.Fatalf("got %d named values, want 3", len(got))
		}
		assertNamedValue(t, got[0], "overflow", 0)
		assertNamedValue(t, got[1], "underflow", 1)
		assertNamedValue(t, got[2], "reserved", 7)
	})

	t.Run("Bits empty", func(t *testing.T) {
		syntax := &module.TypeSyntaxBits{}
		got := extractNamedValues(syntax)
		if len(got) != 0 {
			t.Errorf("got %d named values, want 0", len(got))
		}
	})

	t.Run("TypeRef returns nil", func(t *testing.T) {
		got := extractNamedValues(&module.TypeSyntaxTypeRef{Name: "Integer32"})
		if got != nil {
			t.Errorf("got %v, want nil", got)
		}
	})

	t.Run("OctetString returns nil", func(t *testing.T) {
		got := extractNamedValues(&module.TypeSyntaxOctetString{})
		if got != nil {
			t.Errorf("got %v, want nil", got)
		}
	})

	t.Run("nil returns nil", func(t *testing.T) {
		got := extractNamedValues(nil)
		if got != nil {
			t.Errorf("got %v, want nil", got)
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
		if len(got) != 2 {
			t.Fatalf("got %d named values, want 2", len(got))
		}
		assertNamedValue(t, got[0], "neg", -1)
		assertNamedValue(t, got[1], "zero", 0)
	})
}

func assertNamedValue(t *testing.T, nv mib.NamedValue, wantLabel string, wantValue int64) {
	t.Helper()
	if nv.Label != wantLabel {
		t.Errorf("label = %q, want %q", nv.Label, wantLabel)
	}
	if nv.Value != wantValue {
		t.Errorf("value = %d, want %d", nv.Value, wantValue)
	}
}

func TestExtractConstraints(t *testing.T) {
	t.Run("size constraint", func(t *testing.T) {
		syntax := &module.TypeSyntaxConstrained{
			Base: &module.TypeSyntaxOctetString{},
			Constraint: &module.ConstraintSize{
				Ranges: []module.Range{
					module.NewRangeUnsigned(0, 255),
				},
			},
		}
		sizes, ranges := extractConstraints(syntax)
		if len(sizes) != 1 {
			t.Fatalf("got %d sizes, want 1", len(sizes))
		}
		if sizes[0].Min != 0 || sizes[0].Max != 255 {
			t.Errorf("size = %v, want {0 255}", sizes[0])
		}
		if ranges != nil {
			t.Errorf("ranges = %v, want nil", ranges)
		}
	})

	t.Run("range constraint", func(t *testing.T) {
		syntax := &module.TypeSyntaxConstrained{
			Base: &module.TypeSyntaxTypeRef{Name: "Integer32"},
			Constraint: &module.ConstraintRange{
				Ranges: []module.Range{
					module.NewRangeSigned(-128, 127),
				},
			},
		}
		sizes, ranges := extractConstraints(syntax)
		if sizes != nil {
			t.Errorf("sizes = %v, want nil", sizes)
		}
		if len(ranges) != 1 {
			t.Fatalf("got %d ranges, want 1", len(ranges))
		}
		if ranges[0].Min != -128 || ranges[0].Max != 127 {
			t.Errorf("range = %v, want {-128 127}", ranges[0])
		}
	})

	t.Run("multiple size ranges", func(t *testing.T) {
		syntax := &module.TypeSyntaxConstrained{
			Base: &module.TypeSyntaxOctetString{},
			Constraint: &module.ConstraintSize{
				Ranges: []module.Range{
					module.NewRangeSingleUnsigned(0),
					module.NewRangeUnsigned(4, 255),
				},
			},
		}
		sizes, ranges := extractConstraints(syntax)
		if len(sizes) != 2 {
			t.Fatalf("got %d sizes, want 2", len(sizes))
		}
		// Single value: max = min
		if sizes[0].Min != 0 || sizes[0].Max != 0 {
			t.Errorf("sizes[0] = %v, want {0 0}", sizes[0])
		}
		if sizes[1].Min != 4 || sizes[1].Max != 255 {
			t.Errorf("sizes[1] = %v, want {4 255}", sizes[1])
		}
		if ranges != nil {
			t.Errorf("ranges = %v, want nil", ranges)
		}
	})

	t.Run("non-constrained syntax returns nil", func(t *testing.T) {
		sizes, ranges := extractConstraints(&module.TypeSyntaxTypeRef{Name: "Integer32"})
		if sizes != nil || ranges != nil {
			t.Errorf("got sizes=%v ranges=%v, want nil nil", sizes, ranges)
		}
	})

	t.Run("nil syntax returns nil", func(t *testing.T) {
		sizes, ranges := extractConstraints(nil)
		if sizes != nil || ranges != nil {
			t.Errorf("got sizes=%v ranges=%v, want nil nil", sizes, ranges)
		}
	})
}

func TestRangesToConstraint(t *testing.T) {
	t.Run("signed range", func(t *testing.T) {
		ranges := []module.Range{
			module.NewRangeSigned(-100, 100),
		}
		got := rangesToConstraint(ranges)
		if len(got) != 1 {
			t.Fatalf("got %d ranges, want 1", len(got))
		}
		if got[0].Min != -100 || got[0].Max != 100 {
			t.Errorf("got %v, want {-100 100}", got[0])
		}
	})

	t.Run("single value range", func(t *testing.T) {
		ranges := []module.Range{
			module.NewRangeSingleSigned(42),
		}
		got := rangesToConstraint(ranges)
		if len(got) != 1 {
			t.Fatalf("got %d ranges, want 1", len(got))
		}
		// Single value: Max is nil, so max = min
		if got[0].Min != 42 || got[0].Max != 42 {
			t.Errorf("got %v, want {42 42}", got[0])
		}
	})

	t.Run("multiple ranges", func(t *testing.T) {
		ranges := []module.Range{
			module.NewRangeSigned(0, 10),
			module.NewRangeSigned(100, 200),
		}
		got := rangesToConstraint(ranges)
		if len(got) != 2 {
			t.Fatalf("got %d ranges, want 2", len(got))
		}
		if got[0].Min != 0 || got[0].Max != 10 {
			t.Errorf("got[0] = %v, want {0 10}", got[0])
		}
		if got[1].Min != 100 || got[1].Max != 200 {
			t.Errorf("got[1] = %v, want {100 200}", got[1])
		}
	})

	t.Run("empty ranges", func(t *testing.T) {
		got := rangesToConstraint(nil)
		if len(got) != 0 {
			t.Errorf("got %d ranges, want 0", len(got))
		}
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
			if got != tt.want {
				t.Errorf("rangeValueToI64() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestResolveBaseFromChain(t *testing.T) {
	t.Run("no parent returns own base", func(t *testing.T) {
		typ := mibimpl.NewType("MyType")
		typ.SetBase(mib.BaseOctetString)

		got, ok := resolveBaseFromChain(typ)
		if !ok {
			t.Fatal("expected ok=true")
		}
		if got != mib.BaseOctetString {
			t.Errorf("got %v, want %v", got, mib.BaseOctetString)
		}
	})

	t.Run("walks chain to root", func(t *testing.T) {
		root := mibimpl.NewType("INTEGER")
		root.SetBase(mib.BaseInteger32)

		mid := mibimpl.NewType("MyInt")
		mid.SetBase(mib.BaseInteger32)
		mid.SetParent(root)

		leaf := mibimpl.NewType("MySpecificInt")
		leaf.SetBase(mib.BaseInteger32)
		leaf.SetParent(mid)

		got, ok := resolveBaseFromChain(leaf)
		if !ok {
			t.Fatal("expected ok=true")
		}
		if got != mib.BaseInteger32 {
			t.Errorf("got %v, want %v", got, mib.BaseInteger32)
		}
	})

	t.Run("stops at application base type", func(t *testing.T) {
		root := mibimpl.NewType("INTEGER")
		root.SetBase(mib.BaseInteger32)

		counter := mibimpl.NewType("Counter32")
		counter.SetBase(mib.BaseCounter32)
		counter.SetParent(root)

		myCounter := mibimpl.NewType("MyCounter")
		myCounter.SetBase(mib.BaseCounter32)
		myCounter.SetParent(counter)

		got, ok := resolveBaseFromChain(myCounter)
		if !ok {
			t.Fatal("expected ok=true")
		}
		// Should stop at Counter32 (application base type), not walk to INTEGER
		if got != mib.BaseCounter32 {
			t.Errorf("got %v, want %v", got, mib.BaseCounter32)
		}
	})

	t.Run("stops at each application type", func(t *testing.T) {
		appTypes := []mib.BaseType{
			mib.BaseCounter32, mib.BaseCounter64, mib.BaseGauge32,
			mib.BaseUnsigned32, mib.BaseTimeTicks, mib.BaseIpAddress, mib.BaseOpaque,
		}
		for _, appBase := range appTypes {
			root := mibimpl.NewType("root")
			root.SetBase(mib.BaseInteger32)

			appType := mibimpl.NewType("app")
			appType.SetBase(appBase)
			appType.SetParent(root)

			child := mibimpl.NewType("child")
			child.SetBase(appBase)
			child.SetParent(appType)

			got, ok := resolveBaseFromChain(child)
			if !ok {
				t.Fatalf("expected ok=true for %v", appBase)
			}
			if got != appBase {
				t.Errorf("for %v: got %v, want %v", appBase, got, appBase)
			}
		}
	})

	t.Run("cycle detection", func(t *testing.T) {
		a := mibimpl.NewType("A")
		a.SetBase(mib.BaseInteger32)

		b := mibimpl.NewType("B")
		b.SetBase(mib.BaseInteger32)

		// Create cycle: a -> b -> a
		a.SetParent(b)
		b.SetParent(a)

		_, ok := resolveBaseFromChain(a)
		if ok {
			t.Error("expected ok=false for cycle")
		}
	})

	t.Run("self-referencing cycle", func(t *testing.T) {
		a := mibimpl.NewType("A")
		a.SetBase(mib.BaseInteger32)
		a.SetParent(a)

		_, ok := resolveBaseFromChain(a)
		if ok {
			t.Error("expected ok=false for self-referencing cycle")
		}
	})

	t.Run("long chain", func(t *testing.T) {
		root := mibimpl.NewType("root")
		root.SetBase(mib.BaseOctetString)

		prev := root
		for i := 0; i < 10; i++ {
			typ := mibimpl.NewType("type")
			typ.SetBase(mib.BaseOctetString)
			typ.SetParent(prev)
			prev = typ
		}

		got, ok := resolveBaseFromChain(prev)
		if !ok {
			t.Fatal("expected ok=true")
		}
		if got != mib.BaseOctetString {
			t.Errorf("got %v, want %v", got, mib.BaseOctetString)
		}
	})

	t.Run("inherits base from root of chain", func(t *testing.T) {
		root := mibimpl.NewType("OCTET STRING")
		root.SetBase(mib.BaseOctetString)

		displayString := mibimpl.NewType("DisplayString")
		displayString.SetBase(mib.BaseInteger32) // wrong base, should be overridden
		displayString.SetParent(root)

		got, ok := resolveBaseFromChain(displayString)
		if !ok {
			t.Fatal("expected ok=true")
		}
		// Should return the root's base type
		if got != mib.BaseOctetString {
			t.Errorf("got %v, want %v", got, mib.BaseOctetString)
		}
	})
}
