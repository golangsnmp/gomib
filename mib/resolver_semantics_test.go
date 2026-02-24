package mib

import (
	"testing"

	"github.com/golangsnmp/gomib/internal/module"
	"github.com/golangsnmp/gomib/internal/testutil"
	"github.com/golangsnmp/gomib/internal/types"
)

func TestIsBareTypeIndex(t *testing.T) {
	tests := []struct {
		name string
		want bool
	}{
		{"INTEGER", true},
		{"Integer32", true},
		{"Unsigned32", true},
		{"Counter32", true},
		{"Counter64", true},
		{"Gauge32", true},
		{"IpAddress", true},
		{"Opaque", true},
		{"TimeTicks", true},
		{"BITS", true},
		{"OCTET STRING", true},
		{"Counter", true},
		{"Gauge", true},
		{"NetworkAddress", true},
		// Not bare types
		{"DisplayString", false},
		{"IfIndex", false},
		{"", false},
		{"integer", false},
		{"OBJECT IDENTIFIER", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testutil.Equal(t, tt.want, isBareTypeIndex(tt.name), "isBareTypeIndex()")
		})
	}
}

func TestIsOIDType(t *testing.T) {
	tests := []struct {
		name   string
		syntax module.TypeSyntax
		want   bool
	}{
		{
			name:   "TypeSyntaxObjectIdentifier",
			syntax: &module.TypeSyntaxObjectIdentifier{},
			want:   true,
		},
		{
			name:   "TypeRef OBJECT IDENTIFIER",
			syntax: &module.TypeSyntaxTypeRef{Name: "OBJECT IDENTIFIER"},
			want:   true,
		},
		{
			name:   "TypeRef AutonomousType",
			syntax: &module.TypeSyntaxTypeRef{Name: "AutonomousType"},
			want:   true,
		},
		{
			name:   "TypeRef Integer32",
			syntax: &module.TypeSyntaxTypeRef{Name: "Integer32"},
			want:   false,
		},
		{
			name:   "TypeRef DisplayString",
			syntax: &module.TypeSyntaxTypeRef{Name: "DisplayString"},
			want:   false,
		},
		{
			name: "Constrained wrapping ObjectIdentifier",
			syntax: &module.TypeSyntaxConstrained{
				Base: &module.TypeSyntaxObjectIdentifier{},
			},
			want: true,
		},
		{
			name: "Constrained wrapping TypeRef OBJECT IDENTIFIER",
			syntax: &module.TypeSyntaxConstrained{
				Base: &module.TypeSyntaxTypeRef{Name: "OBJECT IDENTIFIER"},
			},
			want: true,
		},
		{
			name: "Constrained wrapping TypeRef AutonomousType",
			syntax: &module.TypeSyntaxConstrained{
				Base: &module.TypeSyntaxTypeRef{Name: "AutonomousType"},
			},
			want: true,
		},
		{
			name: "Nested constrained wrapping ObjectIdentifier",
			syntax: &module.TypeSyntaxConstrained{
				Base: &module.TypeSyntaxConstrained{
					Base: &module.TypeSyntaxObjectIdentifier{},
				},
			},
			want: true,
		},
		{
			name: "Constrained wrapping OctetString",
			syntax: &module.TypeSyntaxConstrained{
				Base: &module.TypeSyntaxOctetString{},
			},
			want: false,
		},
		{
			name:   "IntegerEnum",
			syntax: &module.TypeSyntaxIntegerEnum{},
			want:   false,
		},
		{
			name:   "Bits",
			syntax: &module.TypeSyntaxBits{},
			want:   false,
		},
		{
			name:   "OctetString",
			syntax: &module.TypeSyntaxOctetString{},
			want:   false,
		},
		{
			name:   "SequenceOf",
			syntax: &module.TypeSyntaxSequenceOf{},
			want:   false,
		},
		{
			name:   "Sequence",
			syntax: &module.TypeSyntaxSequence{},
			want:   false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testutil.Equal(t, tt.want, isOIDType(tt.syntax), "isOIDType()")
		})
	}
}

func TestHexToBytes(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []byte
	}{
		{"empty", "", []byte{}},
		{"single byte", "FF", []byte{0xFF}},
		{"two bytes", "00FF", []byte{0x00, 0xFF}},
		{"lowercase", "abcd", []byte{0xAB, 0xCD}},
		{"uppercase", "ABCD", []byte{0xAB, 0xCD}},
		{"mixed case", "aBcD", []byte{0xAB, 0xCD}},
		{"odd length padded", "F", []byte{0x0F}},
		{"odd length three chars", "1A2", []byte{0x01, 0xA2}},
		{"all zeros", "0000", []byte{0x00, 0x00}},
		{"typical MAC", "001122AABBCC", []byte{0x00, 0x11, 0x22, 0xAA, 0xBB, 0xCC}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := hexToBytes(tt.input)
			testutil.NoError(t, err, "hexToBytes(%q)", tt.input)
			testutil.Len(t, got, len(tt.want), "hexToBytes() len")
			for i := range got {
				testutil.Equal(t, tt.want[i], got[i], "hexToBytes(")
			}
		})
	}
}

func TestBinaryToBytes(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []byte
	}{
		{"empty", "", []byte{}},
		{"one byte all ones", "11111111", []byte{0xFF}},
		{"one byte all zeros", "00000000", []byte{0x00}},
		{"one byte pattern", "10101010", []byte{0xAA}},
		{"two bytes", "1111111100000000", []byte{0xFF, 0x00}},
		{"short padded to 8", "1", []byte{0x01}},
		{"short padded 4 bits", "1010", []byte{0x0A}},
		{"non-multiple of 8 nine bits", "101010101", []byte{0x01, 0x55}},
		{"three bits", "111", []byte{0x07}},
		{"nine bits", "100000001", []byte{0x01, 0x01}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := binaryToBytes(tt.input)
			testutil.Len(t, got, len(tt.want), "binaryToBytes() len")
			for i := range got {
				testutil.Equal(t, tt.want[i], got[i], "binaryToBytes(")
			}
		})
	}
}

func TestConvertDefValInteger(t *testing.T) {
	ctx := newTestContext()
	mod := &module.Module{Name: "TEST-MIB"}
	syntax := &module.TypeSyntaxTypeRef{Name: "Integer32"}

	dv := convertDefVal(ctx, &module.DefValInteger{Value: 42}, mod, syntax)
	testutil.NotNil(t, dv, "convertDefVal returned nil")
	testutil.Equal(t, DefValKindInt, dv.Kind(), "kind")
	v, ok := DefValAs[int64](*dv)
	testutil.True(t, ok, "DefValAs[int64] ok")
	testutil.Equal(t, int64(42), v, "value")
	testutil.Equal(t, "42", dv.Raw(), "raw")
}

func TestConvertDefValNegativeInteger(t *testing.T) {
	ctx := newTestContext()
	mod := &module.Module{Name: "TEST-MIB"}
	syntax := &module.TypeSyntaxTypeRef{Name: "Integer32"}

	dv := convertDefVal(ctx, &module.DefValInteger{Value: -1}, mod, syntax)
	testutil.NotNil(t, dv, "convertDefVal returned nil")
	testutil.Equal(t, DefValKindInt, dv.Kind(), "kind")
	v, ok := DefValAs[int64](*dv)
	testutil.True(t, ok, "DefValAs[int64] ok")
	testutil.Equal(t, int64(-1), v, "value")
	testutil.Equal(t, "-1", dv.Raw(), "raw")
}

func TestConvertDefValUnsigned(t *testing.T) {
	ctx := newTestContext()
	mod := &module.Module{Name: "TEST-MIB"}
	syntax := &module.TypeSyntaxTypeRef{Name: "Counter64"}

	dv := convertDefVal(ctx, &module.DefValUnsigned{Value: 12345}, mod, syntax)
	testutil.NotNil(t, dv, "convertDefVal returned nil")
	testutil.Equal(t, DefValKindUint, dv.Kind(), "kind")
	v, ok := DefValAs[uint64](*dv)
	testutil.True(t, ok, "DefValAs[uint64] ok")
	testutil.Equal(t, uint64(12345), v, "value")
	testutil.Equal(t, "12345", dv.Raw(), "raw")
}

func TestConvertDefValString(t *testing.T) {
	ctx := newTestContext()
	mod := &module.Module{Name: "TEST-MIB"}
	syntax := &module.TypeSyntaxTypeRef{Name: "DisplayString"}

	dv := convertDefVal(ctx, &module.DefValString{Value: "public"}, mod, syntax)
	testutil.NotNil(t, dv, "convertDefVal returned nil")
	testutil.Equal(t, DefValKindString, dv.Kind(), "kind")
	v, ok := DefValAs[string](*dv)
	testutil.True(t, ok, "DefValAs[string] ok")
	testutil.Equal(t, "public", v, "value")
	testutil.Equal(t, `"public"`, dv.Raw(), "raw")
}

func TestConvertDefValHexString(t *testing.T) {
	ctx := newTestContext()
	mod := &module.Module{Name: "TEST-MIB"}
	syntax := &module.TypeSyntaxOctetString{}

	dv := convertDefVal(ctx, &module.DefValHexString{Value: "FF00"}, mod, syntax)
	testutil.NotNil(t, dv, "convertDefVal returned nil")
	testutil.Equal(t, DefValKindBytes, dv.Kind(), "kind")
	bytes, ok := DefValAs[[]byte](*dv)
	testutil.True(t, ok, "value is not []byte")
	want := []byte{0xFF, 0x00}
	testutil.Len(t, bytes, len(want), "bytes len")
	for i := range bytes {
		testutil.Equal(t, want[i], bytes[i], "bytes[")
	}
	testutil.Equal(t, "'FF00'H", dv.Raw(), "raw")
}

func TestConvertDefValBinaryString(t *testing.T) {
	ctx := newTestContext()
	mod := &module.Module{Name: "TEST-MIB"}
	syntax := &module.TypeSyntaxOctetString{}

	dv := convertDefVal(ctx, &module.DefValBinaryString{Value: "10101010"}, mod, syntax)
	testutil.NotNil(t, dv, "convertDefVal returned nil")
	testutil.Equal(t, DefValKindBytes, dv.Kind(), "kind")
	bytes, ok := DefValAs[[]byte](*dv)
	testutil.True(t, ok, "value is not []byte")
	testutil.Len(t, bytes, 1, "bytes len")
	testutil.Equal(t, byte(0xAA), bytes[0], "bytes[0]")
	testutil.Equal(t, "'10101010'B", dv.Raw(), "raw")
}

func TestConvertDefValEnum(t *testing.T) {
	ctx := newTestContext()
	mod := &module.Module{Name: "TEST-MIB"}
	syntax := &module.TypeSyntaxIntegerEnum{
		NamedNumbers: []module.NamedNumber{
			{Name: "enabled", Value: 1},
			{Name: "disabled", Value: 2},
		},
	}

	dv := convertDefVal(ctx, &module.DefValEnum{Name: "enabled"}, mod, syntax)
	testutil.NotNil(t, dv, "convertDefVal returned nil")
	testutil.Equal(t, DefValKindEnum, dv.Kind(), "kind")
	v, ok := DefValAs[string](*dv)
	testutil.True(t, ok, "DefValAs[string] ok")
	testutil.Equal(t, "enabled", v, "value")
	testutil.Equal(t, "enabled", dv.Raw(), "raw")
}

func TestConvertDefValEnumOnOIDType(t *testing.T) {
	// When syntax is OID type, DefValEnum is treated as an OID reference.
	// If the node exists, it resolves to an OID DefVal.
	ctx := newTestContext()
	mod := &module.Module{Name: "TEST-MIB"}

	// Set up a node so the OID lookup succeeds
	root := ctx.Mib.Root()
	child := root.getOrCreateChild(1)
	grandchild := child.getOrCreateChild(3)
	grandchild.setName("myTarget")
	ctx.registerModuleNodeSymbol(mod, "myTarget", grandchild)

	syntax := &module.TypeSyntaxObjectIdentifier{}
	dv := convertDefVal(ctx, &module.DefValEnum{Name: "myTarget"}, mod, syntax)
	testutil.NotNil(t, dv, "convertDefVal returned nil")
	testutil.Equal(t, DefValKindOID, dv.Kind(), "kind")
	oid, ok := DefValAs[OID](*dv)
	testutil.True(t, ok, "value is not OID")
	testutil.Len(t, oid, 2, "oid len")
	testutil.Equal(t, uint32(1), oid[0], "oid[0]")
	testutil.Equal(t, uint32(3), oid[1], "oid[1]")
}

func TestConvertDefValEnumOnOIDTypeFallback(t *testing.T) {
	// When syntax is OID type but node lookup fails, falls back to enum.
	ctx := newTestContext()
	mod := &module.Module{Name: "TEST-MIB"}
	syntax := &module.TypeSyntaxObjectIdentifier{}

	dv := convertDefVal(ctx, &module.DefValEnum{Name: "unknownOid"}, mod, syntax)
	testutil.NotNil(t, dv, "convertDefVal returned nil")
	testutil.Equal(t, DefValKindEnum, dv.Kind(), "kind")
}

func TestConvertDefValBits(t *testing.T) {
	ctx := newTestContext()
	mod := &module.Module{Name: "TEST-MIB"}
	syntax := &module.TypeSyntaxBits{}

	dv := convertDefVal(ctx, &module.DefValBits{Labels: []string{"flag1", "flag2"}}, mod, syntax)
	testutil.NotNil(t, dv, "convertDefVal returned nil")
	testutil.Equal(t, DefValKindBits, dv.Kind(), "kind")
	labels, ok := DefValAs[[]string](*dv)
	testutil.True(t, ok, "value is not []string")
	testutil.Len(t, labels, 2, "labels len")
	testutil.Equal(t, "flag1", labels[0], "labels[0]")
	testutil.Equal(t, "flag2", labels[1], "labels[1]")
	testutil.Equal(t, "{ flag1, flag2 }", dv.Raw(), "raw")
}

func TestConvertDefValBitsEmpty(t *testing.T) {
	ctx := newTestContext()
	mod := &module.Module{Name: "TEST-MIB"}
	syntax := &module.TypeSyntaxBits{}

	dv := convertDefVal(ctx, &module.DefValBits{Labels: []string{}}, mod, syntax)
	testutil.NotNil(t, dv, "convertDefVal returned nil")
	testutil.Equal(t, "{ }", dv.Raw(), "raw")
}

func TestConvertDefValOidRef(t *testing.T) {
	ctx := newTestContext()
	mod := &module.Module{Name: "TEST-MIB"}

	root := ctx.Mib.Root()
	child := root.getOrCreateChild(1)
	child.setName("sysName")
	ctx.registerModuleNodeSymbol(mod, "sysName", child)

	dv := convertDefVal(ctx, &module.DefValOidRef{Name: "sysName"}, mod, nil)
	testutil.NotNil(t, dv, "convertDefVal returned nil")
	testutil.Equal(t, DefValKindOID, dv.Kind(), "kind")
}

func TestConvertDefValOidRefUnresolved(t *testing.T) {
	ctx := newTestContext()
	mod := &module.Module{Name: "TEST-MIB"}

	dv := convertDefVal(ctx, &module.DefValOidRef{Name: "missing"}, mod, nil)
	testutil.Nil(t, dv, "expected nil for unresolved OID ref, got")
}

func TestConvertDefValUnknown(t *testing.T) {
	ctx := newTestContext()
	mod := &module.Module{Name: "TEST-MIB"}

	dv := convertDefVal(ctx, &module.DefValUnparsed{}, mod, nil)
	testutil.Nil(t, dv, "expected nil for unparsed DefVal, got")
}

func TestResolveTypeSyntax(t *testing.T) {
	ctx := newTestContext()
	mod := &module.Module{Name: "TEST-MIB"}
	ctx.Modules = append(ctx.Modules, mod)

	intType := newType("Integer32")
	intType.setBase(BaseInteger32)
	ctx.registerModuleTypeSymbol(mod, "Integer32", intType)

	t.Run("TypeRef found", func(t *testing.T) {
		syntax := &module.TypeSyntaxTypeRef{Name: "Integer32"}
		typ, ok := resolveTypeSyntax(ctx, syntax, mod, "testObj", module.OidAssignment{}.Span)
		testutil.True(t, ok, "expected ok=true")
		testutil.Equal(t, intType, typ, "got different type pointer")
	})

	t.Run("TypeRef not found", func(t *testing.T) {
		syntax := &module.TypeSyntaxTypeRef{Name: "Unknown"}
		_, ok := resolveTypeSyntax(ctx, syntax, mod, "testObj", module.OidAssignment{}.Span)
		testutil.False(t, ok, "expected ok=false for unknown type")
	})

	t.Run("Constrained delegates to base", func(t *testing.T) {
		syntax := &module.TypeSyntaxConstrained{
			Base: &module.TypeSyntaxTypeRef{Name: "Integer32"},
		}
		typ, ok := resolveTypeSyntax(ctx, syntax, mod, "testObj", module.OidAssignment{}.Span)
		testutil.True(t, ok, "expected ok=true")
		testutil.Equal(t, intType, typ, "got different type pointer")
	})

	t.Run("IntegerEnum with base", func(t *testing.T) {
		syntax := &module.TypeSyntaxIntegerEnum{
			Base:         "Integer32",
			NamedNumbers: []module.NamedNumber{{Name: "up", Value: 1}},
		}
		typ, ok := resolveTypeSyntax(ctx, syntax, mod, "testObj", module.OidAssignment{}.Span)
		testutil.True(t, ok, "expected ok=true")
		testutil.Equal(t, intType, typ, "got different type pointer")
	})

	t.Run("IntegerEnum with unknown base", func(t *testing.T) {
		syntax := &module.TypeSyntaxIntegerEnum{
			Base:         "MissingType",
			NamedNumbers: []module.NamedNumber{{Name: "up", Value: 1}},
		}
		_, ok := resolveTypeSyntax(ctx, syntax, mod, "testObj", module.OidAssignment{}.Span)
		testutil.False(t, ok, "expected ok=false for unknown base type")
	})

	t.Run("SequenceOf returns false", func(t *testing.T) {
		syntax := &module.TypeSyntaxSequenceOf{EntryType: "MyEntry"}
		_, ok := resolveTypeSyntax(ctx, syntax, mod, "testObj", module.OidAssignment{}.Span)
		testutil.False(t, ok, "expected ok=false for SequenceOf")
	})

	t.Run("Sequence returns false", func(t *testing.T) {
		syntax := &module.TypeSyntaxSequence{}
		_, ok := resolveTypeSyntax(ctx, syntax, mod, "testObj", module.OidAssignment{}.Span)
		testutil.False(t, ok, "expected ok=false for Sequence")
	})
}

func TestResolveTypeSyntaxBaseTypes(t *testing.T) {
	// Bare IntegerEnum, Bits, OctetString, ObjectIdentifier resolve through
	// global type lookup. Set up a context with an SNMPv2-SMI module
	// that has the needed base types registered.
	smiMod := &module.Module{Name: "SNMPv2-SMI"}
	ctx := newResolverContext([]*module.Module{smiMod}, nil, DefaultConfig())
	ctx.Snmpv2SMIModule = smiMod

	integerType := newType("INTEGER")
	integerType.setBase(BaseInteger32)
	ctx.registerModuleTypeSymbol(smiMod, "INTEGER", integerType)

	bitsType := newType("BITS")
	bitsType.setBase(BaseBits)
	ctx.registerModuleTypeSymbol(smiMod, "BITS", bitsType)

	octetType := newType("OCTET STRING")
	octetType.setBase(BaseOctetString)
	ctx.registerModuleTypeSymbol(smiMod, "OCTET STRING", octetType)

	oidType := newType("OBJECT IDENTIFIER")
	oidType.setBase(BaseObjectIdentifier)
	ctx.registerModuleTypeSymbol(smiMod, "OBJECT IDENTIFIER", oidType)

	mod := &module.Module{Name: "TEST-MIB"}

	t.Run("bare IntegerEnum resolves to INTEGER", func(t *testing.T) {
		syntax := &module.TypeSyntaxIntegerEnum{
			NamedNumbers: []module.NamedNumber{{Name: "up", Value: 1}},
		}
		typ, ok := resolveTypeSyntax(ctx, syntax, mod, "testObj", module.OidAssignment{}.Span)
		testutil.True(t, ok, "expected ok=true")
		testutil.Equal(t, integerType, typ, "expected INTEGER type")
	})

	t.Run("Bits resolves to BITS", func(t *testing.T) {
		syntax := &module.TypeSyntaxBits{}
		typ, ok := resolveTypeSyntax(ctx, syntax, mod, "testObj", module.OidAssignment{}.Span)
		testutil.True(t, ok, "expected ok=true")
		testutil.Equal(t, bitsType, typ, "expected BITS type")
	})

	t.Run("OctetString resolves to OCTET STRING", func(t *testing.T) {
		syntax := &module.TypeSyntaxOctetString{}
		typ, ok := resolveTypeSyntax(ctx, syntax, mod, "testObj", module.OidAssignment{}.Span)
		testutil.True(t, ok, "expected ok=true")
		testutil.Equal(t, octetType, typ, "expected OCTET STRING type")
	})

	t.Run("ObjectIdentifier resolves to OBJECT IDENTIFIER", func(t *testing.T) {
		syntax := &module.TypeSyntaxObjectIdentifier{}
		typ, ok := resolveTypeSyntax(ctx, syntax, mod, "testObj", module.OidAssignment{}.Span)
		testutil.True(t, ok, "expected ok=true")
		testutil.Equal(t, oidType, typ, "expected OBJECT IDENTIFIER type")
	})
}

func TestComputeEffectiveValues(t *testing.T) {
	t.Run("nil type is a no-op", func(t *testing.T) {
		obj := newObject("testObj")
		// No type set, should not panic
		computeEffectiveValues(obj)
	})

	t.Run("inherits display hint from type", func(t *testing.T) {
		typ := newType("DisplayString")
		typ.setDisplayHint("255a")

		obj := newObject("testObj")
		obj.setType(typ)
		computeEffectiveValues(obj)

		testutil.Equal(t, "255a", obj.EffectiveDisplayHint(), "hint")
	})

	t.Run("object hint takes precedence over type hint", func(t *testing.T) {
		typ := newType("DisplayString")
		typ.setDisplayHint("255a")

		obj := newObject("testObj")
		obj.setType(typ)
		obj.setEffectiveHint("1d")
		computeEffectiveValues(obj)

		testutil.Equal(t, "1d", obj.EffectiveDisplayHint(), "hint")
	})

	t.Run("inherits sizes from type chain", func(t *testing.T) {
		parentType := newType("OCTET STRING")
		parentType.setSizes([]Range{{Min: 0, Max: 255}})

		childType := newType("DisplayString")
		childType.setParent(parentType)

		obj := newObject("testObj")
		obj.setType(childType)
		computeEffectiveValues(obj)

		sizes := obj.EffectiveSizes()
		testutil.Len(t, sizes, 1, "sizes len")
		testutil.Equal(t, int64(0), sizes[0].Min, "sizes[0].Min")
		testutil.Equal(t, int64(255), sizes[0].Max, "sizes[0].Max")
	})

	t.Run("object sizes take precedence", func(t *testing.T) {
		typ := newType("DisplayString")
		typ.setSizes([]Range{{Min: 0, Max: 255}})

		obj := newObject("testObj")
		obj.setType(typ)
		obj.setEffectiveSizes([]Range{{Min: 1, Max: 32}})
		computeEffectiveValues(obj)

		sizes := obj.EffectiveSizes()
		testutil.Len(t, sizes, 1, "sizes len")
		testutil.Equal(t, int64(1), sizes[0].Min, "sizes[0].Min")
		testutil.Equal(t, int64(32), sizes[0].Max, "sizes[0].Max")
	})

	t.Run("inherits ranges from ancestor", func(t *testing.T) {
		grandparent := newType("INTEGER")
		grandparent.setRanges([]Range{{Min: -128, Max: 127}})

		parent := newType("Integer32")
		parent.setParent(grandparent)

		child := newType("MyInt")
		child.setParent(parent)

		obj := newObject("testObj")
		obj.setType(child)
		computeEffectiveValues(obj)

		ranges := obj.EffectiveRanges()
		testutil.Len(t, ranges, 1, "ranges len")
		testutil.Equal(t, int64(-128), ranges[0].Min, "ranges[0].Min")
		testutil.Equal(t, int64(127), ranges[0].Max, "ranges[0].Max")
	})

	t.Run("inherits enums from type", func(t *testing.T) {
		typ := newType("StatusType")
		typ.setEnums([]NamedValue{
			{Label: "up", Value: 1},
			{Label: "down", Value: 2},
		})

		obj := newObject("testObj")
		obj.setType(typ)
		computeEffectiveValues(obj)

		enums := obj.EffectiveEnums()
		testutil.Len(t, enums, 2, "enums len")
		testutil.Equal(t, "up", enums[0].Label, "enums[0].Label")
		testutil.Equal(t, "down", enums[1].Label, "enums[1].Label")
	})

	t.Run("object enums take precedence over type enums", func(t *testing.T) {
		typ := newType("StatusType")
		typ.setEnums([]NamedValue{
			{Label: "up", Value: 1},
			{Label: "down", Value: 2},
		})

		obj := newObject("testObj")
		obj.setType(typ)
		obj.setEffectiveEnums([]NamedValue{
			{Label: "active", Value: 1},
		})
		computeEffectiveValues(obj)

		enums := obj.EffectiveEnums()
		testutil.Len(t, enums, 1, "enums len")
		testutil.Equal(t, "active", enums[0].Label, "enums[0].Label")
	})

	t.Run("inherits bits from type", func(t *testing.T) {
		typ := newType("Capabilities")
		typ.setBits([]NamedValue{
			{Label: "feature1", Value: 0},
			{Label: "feature2", Value: 1},
		})

		obj := newObject("testObj")
		obj.setType(typ)
		computeEffectiveValues(obj)

		bits := obj.EffectiveBits()
		testutil.Len(t, bits, 2, "bits len")
		testutil.Equal(t, "feature1", bits[0].Label, "bits[0].Label")
	})
}

func TestConvertComplianceModules(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		result := convertComplianceModules(nil, nil, nil)
		testutil.Len(t, result, 0, "expected empty, got")
	})

	t.Run("basic module with mandatory groups", func(t *testing.T) {
		input := []module.ComplianceModule{
			{
				ModuleName:      "IF-MIB",
				MandatoryGroups: []string{"ifGeneralGroup", "ifStackGroup"},
			},
		}
		result := convertComplianceModules(nil, nil, input)
		testutil.Len(t, result, 1, "expected 1, got")
		testutil.Equal(t, "IF-MIB", result[0].ModuleName, "module name")
		testutil.Len(t, result[0].MandatoryGroups, 2, "mandatory groups")
	})

	t.Run("module with groups", func(t *testing.T) {
		input := []module.ComplianceModule{
			{
				ModuleName: "IF-MIB",
				Groups: []module.ComplianceGroup{
					{Group: "ifCounterGroup", Description: "counter desc"},
				},
			},
		}
		result := convertComplianceModules(nil, nil, input)
		testutil.Len(t, result[0].Groups, 1, "groups")
		testutil.Equal(t, "ifCounterGroup", result[0].Groups[0].Group, "group")
		testutil.Equal(t, "counter desc", result[0].Groups[0].Description, "desc")
	})

	t.Run("module with objects and min-access", func(t *testing.T) {
		readOnly := types.AccessReadOnly
		input := []module.ComplianceModule{
			{
				ModuleName: "IF-MIB",
				Objects: []module.ComplianceObject{
					{Object: "ifAdminStatus", MinAccess: &readOnly, Description: "obj desc"},
					{Object: "ifOperStatus", MinAccess: nil},
				},
			},
		}
		result := convertComplianceModules(nil, nil, input)
		testutil.Len(t, result[0].Objects, 2, "objects")
		obj0 := result[0].Objects[0]
		testutil.Equal(t, "ifAdminStatus", obj0.Object, "object")
		testutil.NotNil(t, obj0.MinAccess, "min-access")
		testutil.Equal(t, AccessReadOnly, *obj0.MinAccess, "min-access")
		testutil.Equal(t, "obj desc", obj0.Description, "desc")
		obj1 := result[0].Objects[1]
		testutil.Nil(t, obj1.MinAccess, "expected nil min-access for second object")
	})

	t.Run("multiple modules", func(t *testing.T) {
		input := []module.ComplianceModule{
			{ModuleName: "IF-MIB"},
			{ModuleName: "IP-MIB"},
		}
		result := convertComplianceModules(nil, nil, input)
		testutil.Len(t, result, 2, "expected 2, got")
		testutil.Equal(t, "IF-MIB", result[0].ModuleName, "result[0].ModuleName")
		testutil.Equal(t, "IP-MIB", result[1].ModuleName, "result[1].ModuleName")
	})

	t.Run("object with syntax constraints", func(t *testing.T) {
		ctx := newTestContext()
		mod := &module.Module{Name: "TEST-MIB"}

		intType := newType("Integer32")
		intType.setBase(BaseInteger32)
		ctx.registerModuleTypeSymbol(mod, "Integer32", intType)

		input := []module.ComplianceModule{
			{
				ModuleName: "TEST-MIB",
				Objects: []module.ComplianceObject{
					{
						Object: "testObj",
						Syntax: &module.TypeSyntaxConstrained{
							Base:       &module.TypeSyntaxTypeRef{Name: "Integer32"},
							Constraint: &module.ConstraintRange{Ranges: []module.Range{module.NewRangeSigned(0, 100)}},
						},
						WriteSyntax: &module.TypeSyntaxConstrained{
							Base:       &module.TypeSyntaxTypeRef{Name: "Integer32"},
							Constraint: &module.ConstraintRange{Ranges: []module.Range{module.NewRangeSigned(1, 50)}},
						},
					},
				},
			},
		}
		result := convertComplianceModules(ctx, mod, input)
		obj := result[0].Objects[0]
		testutil.NotNil(t, obj.Syntax, "expected non-nil Syntax")
		testutil.Equal(t, intType, obj.Syntax.Type, "Syntax.Type does not match")
		testutil.Len(t, obj.Syntax.Ranges, 1, "Syntax.Ranges len")
		testutil.Equal(t, int64(0), obj.Syntax.Ranges[0].Min, "Syntax.Ranges[0].Min")
		testutil.Equal(t, int64(100), obj.Syntax.Ranges[0].Max, "Syntax.Ranges[0].Max")
		testutil.NotNil(t, obj.WriteSyntax, "expected non-nil WriteSyntax")
		testutil.Len(t, obj.WriteSyntax.Ranges, 1, "WriteSyntax.Ranges len")
		testutil.Equal(t, int64(1), obj.WriteSyntax.Ranges[0].Min, "WriteSyntax.Ranges[0].Min")
		testutil.Equal(t, int64(50), obj.WriteSyntax.Ranges[0].Max, "WriteSyntax.Ranges[0].Max")
	})
}

func TestConvertSupportsModules(t *testing.T) {
	ctx := newTestContext()
	mod := &module.Module{Name: "TEST-MIB"}

	t.Run("empty", func(t *testing.T) {
		result := convertSupportsModules(ctx, mod, nil)
		testutil.Len(t, result, 0, "expected empty, got")
	})

	t.Run("basic with includes", func(t *testing.T) {
		input := []module.SupportsModule{
			{
				ModuleName: "IF-MIB",
				Includes:   []string{"ifGeneralGroup"},
			},
		}
		result := convertSupportsModules(ctx, mod, input)
		testutil.Len(t, result, 1, "expected 1, got")
		testutil.Equal(t, "IF-MIB", result[0].ModuleName, "module name")
		testutil.Len(t, result[0].Includes, 1, "includes len")
		testutil.Equal(t, "ifGeneralGroup", result[0].Includes[0], "includes[0]")
	})

	t.Run("object variations with access", func(t *testing.T) {
		readOnly := types.AccessReadOnly
		input := []module.SupportsModule{
			{
				ModuleName: "IF-MIB",
				ObjectVariations: []module.ObjectVariation{
					{Object: "ifAdminStatus", Access: &readOnly, Description: "read only"},
					{Object: "ifOperStatus", Access: nil, Description: "full access"},
				},
			},
		}
		result := convertSupportsModules(ctx, mod, input)
		vars := result[0].ObjectVariations
		testutil.Len(t, vars, 2, "variations")
		testutil.Equal(t, "ifAdminStatus", vars[0].Object, "object")
		testutil.NotNil(t, vars[0].Access, "access")
		testutil.Equal(t, AccessReadOnly, *vars[0].Access, "access")
		testutil.Equal(t, "read only", vars[0].Description, "desc")
		testutil.Nil(t, vars[1].Access, "expected nil access for second variation")
	})

	t.Run("notification variations with access", func(t *testing.T) {
		readOnly := types.AccessReadOnly
		input := []module.SupportsModule{
			{
				ModuleName: "IF-MIB",
				NotificationVariations: []module.NotificationVariation{
					{Notification: "linkDown", Access: &readOnly, Description: "not supported"},
					{Notification: "linkUp", Access: nil},
				},
			},
		}
		result := convertSupportsModules(ctx, mod, input)
		vars := result[0].NotificationVariations
		testutil.Len(t, vars, 2, "notification variations")
		testutil.Equal(t, "linkDown", vars[0].Notification, "notification")
		testutil.NotNil(t, vars[0].Access, "access")
		testutil.Equal(t, AccessReadOnly, *vars[0].Access, "access")
		testutil.Equal(t, "not supported", vars[0].Description, "desc")
		testutil.Nil(t, vars[1].Access, "expected nil access for second variation")
	})

	t.Run("not-implemented access preserved in variations", func(t *testing.T) {
		notImpl := types.AccessNotImplemented
		input := []module.SupportsModule{
			{
				ModuleName: "TEST-MIB",
				ObjectVariations: []module.ObjectVariation{
					{Object: "testObj", Access: &notImpl, Description: "not implemented"},
				},
			},
		}
		result := convertSupportsModules(ctx, mod, input)
		vars := result[0].ObjectVariations
		testutil.Len(t, vars, 1, "variations")
		testutil.NotNil(t, vars[0].Access, "access is nil, want not-implemented")
		testutil.Equal(t, AccessNotImplemented, *vars[0].Access, "access")
	})

	t.Run("mixed object and notification variations", func(t *testing.T) {
		readOnly := types.AccessReadOnly
		readWrite := types.AccessReadWrite
		input := []module.SupportsModule{
			{
				ModuleName: "IF-MIB",
				Includes:   []string{"ifGeneralGroup"},
				ObjectVariations: []module.ObjectVariation{
					{Object: "ifAdminStatus", Access: &readOnly},
				},
				NotificationVariations: []module.NotificationVariation{
					{Notification: "linkDown", Access: &readWrite},
				},
			},
		}
		result := convertSupportsModules(ctx, mod, input)
		testutil.Len(t, result[0].ObjectVariations, 1, "object variations")
		testutil.Len(t, result[0].NotificationVariations, 1, "notification variations")
	})

	t.Run("object variation with syntax and creation-requires", func(t *testing.T) {
		intType := newType("Integer32")
		intType.setBase(BaseInteger32)
		ctx.registerModuleTypeSymbol(mod, "Integer32", intType)

		input := []module.SupportsModule{
			{
				ModuleName: "IF-MIB",
				ObjectVariations: []module.ObjectVariation{
					{
						Object: "ifAdminStatus",
						Syntax: &module.TypeSyntaxConstrained{
							Base:       &module.TypeSyntaxTypeRef{Name: "Integer32"},
							Constraint: &module.ConstraintRange{Ranges: []module.Range{module.NewRangeSigned(1, 2)}},
						},
						WriteSyntax:      &module.TypeSyntaxTypeRef{Name: "Integer32"},
						CreationRequires: []string{"ifType", "ifSpeed"},
						Description:      "restricted",
					},
				},
			},
		}
		result := convertSupportsModules(ctx, mod, input)
		v := result[0].ObjectVariations[0]
		testutil.NotNil(t, v.Syntax, "expected non-nil Syntax")
		testutil.Equal(t, intType, v.Syntax.Type, "Syntax.Type does not match")
		testutil.Len(t, v.Syntax.Ranges, 1, "Syntax.Ranges len")
		testutil.Equal(t, int64(1), v.Syntax.Ranges[0].Min, "Syntax.Ranges[0].Min")
		testutil.Equal(t, int64(2), v.Syntax.Ranges[0].Max, "Syntax.Ranges[0].Max")
		testutil.NotNil(t, v.WriteSyntax, "expected non-nil WriteSyntax")
		testutil.Equal(t, intType, v.WriteSyntax.Type, "WriteSyntax.Type does not match")
		testutil.Len(t, v.CreationRequires, 2, "CreationRequires len")
		testutil.Equal(t, "ifType", v.CreationRequires[0], "CreationRequires[0]")
		testutil.Equal(t, "ifSpeed", v.CreationRequires[1], "CreationRequires[1]")
	})

	t.Run("object variation with defval", func(t *testing.T) {
		input := []module.SupportsModule{
			{
				ModuleName: "IF-MIB",
				ObjectVariations: []module.ObjectVariation{
					{
						Object:      "ifAdminStatus",
						DefVal:      &module.DefValInteger{Value: 2},
						Syntax:      &module.TypeSyntaxTypeRef{Name: "Integer32"},
						Description: "default down",
					},
				},
			},
		}
		result := convertSupportsModules(ctx, mod, input)
		vars := result[0].ObjectVariations
		testutil.Len(t, vars, 1, "variations")
		testutil.False(t, vars[0].DefVal.IsZero(), "expected non-zero DefVal")
		testutil.Equal(t, DefValKindInt, vars[0].DefVal.Kind(), "defval kind")
		v, ok := DefValAs[int64](vars[0].DefVal)
		testutil.True(t, ok, "DefValAs[int64] ok")
		testutil.Equal(t, int64(2), v, "defval value")
	})
}

func TestConvertDefValOidValue(t *testing.T) {
	ctx := newTestContext()
	mod := &module.Module{Name: "TEST-MIB"}

	root := ctx.Mib.Root()
	child := root.getOrCreateChild(1)
	child2 := child.getOrCreateChild(3)
	child2.setName("enterprises")
	ctx.registerModuleNodeSymbol(mod, "enterprises", child2)

	t.Run("resolves name with trailing numeric arcs", func(t *testing.T) {
		dv := convertDefVal(ctx, &module.DefValOidValue{
			Components: []module.OidComponent{
				&module.OidComponentName{NameValue: "enterprises"},
				&module.OidComponentNumber{Value: 42},
				&module.OidComponentNumber{Value: 1},
			},
		}, mod, nil)
		testutil.NotNil(t, dv, "expected non-nil")
		testutil.Equal(t, DefValKindOID, dv.Kind(), "kind")
		oid, ok := DefValAs[OID](*dv)
		testutil.True(t, ok, "expected OID value")
		want := OID{1, 3, 42, 1}
		testutil.Equal(t, want.String(), oid.String(), "oid")
	})

	t.Run("resolves single name component", func(t *testing.T) {
		dv := convertDefVal(ctx, &module.DefValOidValue{
			Components: []module.OidComponent{
				&module.OidComponentName{NameValue: "enterprises"},
			},
		}, mod, nil)
		testutil.NotNil(t, dv, "expected non-nil")
		oid, ok := DefValAs[OID](*dv)
		testutil.True(t, ok, "expected OID value")
		want := OID{1, 3}
		testutil.Equal(t, want.String(), oid.String(), "oid")
	})

	t.Run("resolves named number with trailing arcs", func(t *testing.T) {
		dv := convertDefVal(ctx, &module.DefValOidValue{
			Components: []module.OidComponent{
				&module.OidComponentNamedNumber{NameValue: "enterprises", NumberValue: 1},
				&module.OidComponentNumber{Value: 5},
			},
		}, mod, nil)
		testutil.NotNil(t, dv, "expected non-nil")
		oid, ok := DefValAs[OID](*dv)
		testutil.True(t, ok, "expected OID value")
		want := OID{1, 3, 5}
		testutil.Equal(t, want.String(), oid.String(), "oid")
	})

	t.Run("resolves qualified name", func(t *testing.T) {
		dv := convertDefVal(ctx, &module.DefValOidValue{
			Components: []module.OidComponent{
				&module.OidComponentQualifiedName{ModuleValue: "SNMPv2-SMI", NameValue: "enterprises"},
			},
		}, mod, nil)
		testutil.NotNil(t, dv, "expected non-nil")
	})

	t.Run("resolves qualified named number", func(t *testing.T) {
		dv := convertDefVal(ctx, &module.DefValOidValue{
			Components: []module.OidComponent{
				&module.OidComponentQualifiedNamedNumber{
					ModuleValue: "SNMPv2-SMI",
					NameValue:   "enterprises",
					NumberValue: 1,
				},
			},
		}, mod, nil)
		testutil.NotNil(t, dv, "expected non-nil")
	})

	t.Run("returns nil for unresolved name", func(t *testing.T) {
		dv := convertDefVal(ctx, &module.DefValOidValue{
			Components: []module.OidComponent{
				&module.OidComponentName{NameValue: "nonexistent"},
			},
		}, mod, nil)
		testutil.Nil(t, dv, "expected nil for unresolved name, got")
	})

	t.Run("returns nil for empty components", func(t *testing.T) {
		dv := convertDefVal(ctx, &module.DefValOidValue{
			Components: nil,
		}, mod, nil)
		testutil.Nil(t, dv, "expected nil for empty components, got")
	})

	t.Run("returns nil when first component is numeric only", func(t *testing.T) {
		dv := convertDefVal(ctx, &module.DefValOidValue{
			Components: []module.OidComponent{
				&module.OidComponentNumber{Value: 1},
			},
		}, mod, nil)
		testutil.Nil(t, dv, "expected nil when first component is numeric, got")
	})
}

func TestCreateResolvedNotifications_NilObjectDiagnostic(t *testing.T) {
	// When a notification references an object whose node exists but has no
	// Object (e.g., an intermediate node), a diagnostic should be
	// emitted rather than silently dropping the reference.
	mod := &module.Module{
		Name: "TEST-MIB",
		Definitions: []module.Definition{
			&module.Notification{
				Name:    "testNotif",
				Objects: []string{"intermediateNode"},
			},
		},
	}

	ctx := newResolverContext([]*module.Module{mod}, nil, DefaultConfig())
	ctx.ModuleIndex[mod.Name] = []*module.Module{mod}
	resolvedMod := newModule(mod.Name)
	ctx.ModuleToResolved[mod] = resolvedMod
	ctx.ResolvedToModule[resolvedMod] = mod

	// Create a node for the notification itself
	root := ctx.Mib.Root()
	notifNode := root.getOrCreateChild(1)
	notifNode.setName("testNotif")
	ctx.registerModuleNodeSymbol(mod, "testNotif", notifNode)

	// Create a node for the referenced object, but do NOT set an Object on it.
	// This simulates an intermediate node or non-object definition.
	objNode := root.getOrCreateChild(2)
	objNode.setName("intermediateNode")
	ctx.registerModuleNodeSymbol(mod, "intermediateNode", objNode)

	createResolvedNotifications(ctx)

	// Should have emitted a diagnostic for the nil-object case.
	var found bool
	for _, d := range ctx.Diagnostics() {
		if d.Code == "notification-object-not-object" {
			found = true
			break
		}
	}
	testutil.True(t, found, "expected diagnostic for notification object with nil Object, got none")
}
