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
		name     string
		input    string
		rightPad bool
		want     []byte
	}{
		{"empty", "", false, []byte{}},
		{"one byte all ones", "11111111", false, []byte{0xFF}},
		{"one byte all zeros", "00000000", false, []byte{0x00}},
		{"one byte pattern", "10101010", false, []byte{0xAA}},
		{"two bytes", "1111111100000000", false, []byte{0xFF, 0x00}},
		{"short left-padded to 8", "1", false, []byte{0x01}},
		{"short left-padded 4 bits", "1010", false, []byte{0x0A}},
		{"non-multiple of 8 nine bits left-pad", "101010101", false, []byte{0x01, 0x55}},
		{"three bits left-pad", "111", false, []byte{0x07}},
		{"nine bits left-pad", "100000001", false, []byte{0x01, 0x01}},
		// BITS right-padding: bits are MSB-first per RFC 2578
		{"short right-padded to 8", "1", true, []byte{0x80}},
		{"short right-padded 4 bits", "1010", true, []byte{0xA0}},
		{"three bits right-pad", "101", true, []byte{0xA0}},
		{"nine bits right-pad", "101010101", true, []byte{0xAA, 0x80}},
		{"full byte no padding needed", "11111111", true, []byte{0xFF}},
		{"empty right-pad", "", true, []byte{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := binaryToBytes(tt.input, tt.rightPad)
			testutil.Len(t, got, len(tt.want), "binaryToBytes() len")
			for i := range got {
				testutil.Equal(t, tt.want[i], got[i], "binaryToBytes()")
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
	ctx := newResolverContext(nil, nil, PermissiveConfig())
	mod := &module.Module{Name: "TEST-MIB"}

	dv := convertDefVal(ctx, &module.DefValUnparsed{}, mod, nil)
	testutil.Nil(t, dv, "expected nil for unparsed DefVal, got")

	var found bool
	for _, d := range ctx.Diagnostics() {
		if d.Code == types.DiagDefvalUnresolved {
			found = true
			break
		}
	}
	testutil.True(t, found, "expected DiagDefvalUnresolved diagnostic for unparsed DefVal")
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
	t.Run("nil type is a no-op", func(_ *testing.T) {
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

	// registerSupportsNode registers a named node with a given kind in a
	// SUPPORTS module so the resolver can classify variations correctly.
	registerSupportsNode := func(moduleName, name string, kind Kind) {
		supMod := &module.Module{Name: moduleName}
		ctx.ModuleIndex[moduleName] = append(ctx.ModuleIndex[moduleName], supMod)
		node := newTestNode(name)
		node.setKind(kind)
		ctx.registerModuleNodeSymbol(supMod, name, node)
	}

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
		registerSupportsNode("OBJ-MIB", "ifAdminStatus", KindColumn)
		registerSupportsNode("OBJ-MIB", "ifOperStatus", KindColumn)

		readOnly := types.AccessReadOnly
		input := []module.SupportsModule{
			{
				ModuleName: "OBJ-MIB",
				Variations: []module.Variation{
					{Name: "ifAdminStatus", Access: &readOnly, Description: "read only"},
					{Name: "ifOperStatus", Access: nil, Description: "full access"},
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
		registerSupportsNode("NOTIF-MIB", "linkDown", KindNotification)
		registerSupportsNode("NOTIF-MIB", "linkUp", KindNotification)

		notImpl := types.AccessNotImplemented
		input := []module.SupportsModule{
			{
				ModuleName: "NOTIF-MIB",
				Variations: []module.Variation{
					{Name: "linkDown", Access: &notImpl, Description: "not supported"},
					{Name: "linkUp", Access: nil},
				},
			},
		}
		result := convertSupportsModules(ctx, mod, input)
		vars := result[0].NotificationVariations
		testutil.Len(t, vars, 2, "notification variations")
		testutil.Equal(t, "linkDown", vars[0].Notification, "notification")
		testutil.NotNil(t, vars[0].Access, "access")
		testutil.Equal(t, AccessNotImplemented, *vars[0].Access, "access")
		testutil.Equal(t, "not supported", vars[0].Description, "desc")
		testutil.Nil(t, vars[1].Access, "expected nil access for second variation")
	})

	t.Run("not-implemented access preserved in variations", func(t *testing.T) {
		registerSupportsNode("IMPL-MIB", "testObj", KindScalar)

		notImpl := types.AccessNotImplemented
		input := []module.SupportsModule{
			{
				ModuleName: "IMPL-MIB",
				Variations: []module.Variation{
					{Name: "testObj", Access: &notImpl, Description: "not implemented"},
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
		registerSupportsNode("MIX-MIB", "ifAdminStatus", KindColumn)
		registerSupportsNode("MIX-MIB", "linkDown", KindNotification)

		readOnly := types.AccessReadOnly
		notImpl := types.AccessNotImplemented
		input := []module.SupportsModule{
			{
				ModuleName: "MIX-MIB",
				Includes:   []string{"ifGeneralGroup"},
				Variations: []module.Variation{
					{Name: "ifAdminStatus", Access: &readOnly},
					{Name: "linkDown", Access: &notImpl},
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
				Variations: []module.Variation{
					{
						Name: "ifAdminStatus",
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
				Variations: []module.Variation{
					{
						Name:        "ifAdminStatus",
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
	smiMod := &module.Module{Name: "SNMPv2-SMI"}
	ctx.ModuleIndex["SNMPv2-SMI"] = []*module.Module{smiMod}

	root := ctx.Mib.Root()
	child := root.getOrCreateChild(1)
	child2 := child.getOrCreateChild(3)
	child2.setName("enterprises")
	ctx.registerModuleNodeSymbol(mod, "enterprises", child2)
	ctx.registerModuleNodeSymbol(smiMod, "enterprises", child2)

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
		testutil.Equal(t, DefValKindOID, dv.Kind(), "kind")
		oid, ok := DefValAs[OID](*dv)
		testutil.True(t, ok, "expected OID value")
		want := OID{1, 3}
		testutil.Equal(t, want.String(), oid.String(), "oid")
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
		testutil.Equal(t, DefValKindOID, dv.Kind(), "kind")
		oid, ok := DefValAs[OID](*dv)
		testutil.True(t, ok, "expected OID value")
		want := OID{1, 3}
		testutil.Equal(t, want.String(), oid.String(), "oid")
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

func TestInferNodeKinds(t *testing.T) {
	t.Run("classifies table row scalar", func(t *testing.T) {
		ctx := newTestContext()
		mod := &module.Module{Name: "TEST-MIB"}
		ctx.Modules = append(ctx.Modules, mod)
		ctx.ModuleToResolved[mod] = newModule(mod.Name)

		root := ctx.Mib.Root()
		tableNode := buildOIDPath(root, 1, 1)
		tableNode.setName("myTable")
		ctx.registerModuleNodeSymbol(mod, "myTable", tableNode)

		rowNode := buildOIDPath(root, 1, 1, 1)
		rowNode.setName("myEntry")
		ctx.registerModuleNodeSymbol(mod, "myEntry", rowNode)

		scalarNode := buildOIDPath(root, 1, 2)
		scalarNode.setName("myScalar")
		ctx.registerModuleNodeSymbol(mod, "myScalar", scalarNode)

		objRefs := []objectTypeRef{
			{mod: mod, obj: &module.ObjectType{
				Name:   "myTable",
				Syntax: &module.TypeSyntaxSequenceOf{EntryType: "MyEntry"},
			}},
			{mod: mod, obj: &module.ObjectType{
				Name:   "myEntry",
				Syntax: &module.TypeSyntaxTypeRef{Name: "MyEntry"},
				Index:  []module.IndexItem{{Object: "myIndex"}},
			}},
			{mod: mod, obj: &module.ObjectType{
				Name:   "myScalar",
				Syntax: &module.TypeSyntaxTypeRef{Name: "Integer32"},
			}},
		}

		inferNodeKinds(ctx, objRefs)

		testutil.Equal(t, KindTable, tableNode.Kind(), "table kind")
		testutil.Equal(t, KindRow, rowNode.Kind(), "row kind")
		testutil.Equal(t, KindScalar, scalarNode.Kind(), "scalar kind")
	})

	t.Run("augments classifies as row", func(t *testing.T) {
		ctx := newTestContext()
		mod := &module.Module{Name: "TEST-MIB"}
		ctx.Modules = append(ctx.Modules, mod)
		ctx.ModuleToResolved[mod] = newModule(mod.Name)

		root := ctx.Mib.Root()
		augNode := buildOIDPath(root, 1, 1)
		augNode.setName("myAugEntry")
		ctx.registerModuleNodeSymbol(mod, "myAugEntry", augNode)

		objRefs := []objectTypeRef{
			{mod: mod, obj: &module.ObjectType{
				Name:     "myAugEntry",
				Syntax:   &module.TypeSyntaxTypeRef{Name: "MyAugEntry"},
				Augments: "otherEntry",
			}},
		}

		inferNodeKinds(ctx, objRefs)

		testutil.Equal(t, KindRow, augNode.Kind(), "augments node kind")
	})

	t.Run("reclassifies scalar children of rows as columns", func(t *testing.T) {
		ctx := newTestContext()
		mod := &module.Module{Name: "TEST-MIB"}
		ctx.Modules = append(ctx.Modules, mod)
		ctx.ModuleToResolved[mod] = newModule(mod.Name)

		root := ctx.Mib.Root()
		rowNode := buildOIDPath(root, 1, 1, 1)
		rowNode.setName("myEntry")
		ctx.registerModuleNodeSymbol(mod, "myEntry", rowNode)

		col1 := buildOIDPath(root, 1, 1, 1, 1)
		col1.setName("myCol1")
		ctx.registerModuleNodeSymbol(mod, "myCol1", col1)

		col2 := buildOIDPath(root, 1, 1, 1, 2)
		col2.setName("myCol2")
		ctx.registerModuleNodeSymbol(mod, "myCol2", col2)

		objRefs := []objectTypeRef{
			{mod: mod, obj: &module.ObjectType{
				Name:   "myEntry",
				Syntax: &module.TypeSyntaxTypeRef{Name: "MyEntry"},
				Index:  []module.IndexItem{{Object: "myCol1"}},
			}},
			{mod: mod, obj: &module.ObjectType{
				Name:   "myCol1",
				Syntax: &module.TypeSyntaxTypeRef{Name: "Integer32"},
			}},
			{mod: mod, obj: &module.ObjectType{
				Name:   "myCol2",
				Syntax: &module.TypeSyntaxTypeRef{Name: "DisplayString"},
			}},
		}

		inferNodeKinds(ctx, objRefs)

		testutil.Equal(t, KindRow, rowNode.Kind(), "row kind")
		testutil.Equal(t, KindColumn, col1.Kind(), "col1 kind")
		testutil.Equal(t, KindColumn, col2.Kind(), "col2 kind")
	})

	t.Run("skips objects with unresolved nodes", func(_ *testing.T) {
		ctx := newTestContext()
		mod := &module.Module{Name: "TEST-MIB"}
		ctx.Modules = append(ctx.Modules, mod)
		ctx.ModuleToResolved[mod] = newModule(mod.Name)

		// Don't register any node - the lookup should fail silently.
		objRefs := []objectTypeRef{
			{mod: mod, obj: &module.ObjectType{
				Name:   "missing",
				Syntax: &module.TypeSyntaxTypeRef{Name: "Integer32"},
			}},
		}

		// Should not panic.
		inferNodeKinds(ctx, objRefs)
	})

	t.Run("non-scalar children of rows not reclassified", func(t *testing.T) {
		ctx := newTestContext()
		mod := &module.Module{Name: "TEST-MIB"}
		ctx.Modules = append(ctx.Modules, mod)
		ctx.ModuleToResolved[mod] = newModule(mod.Name)

		root := ctx.Mib.Root()
		rowNode := buildOIDPath(root, 1, 1, 1)
		rowNode.setName("myEntry")
		ctx.registerModuleNodeSymbol(mod, "myEntry", rowNode)

		// Child node that is not an OBJECT-TYPE (no objRef), so it stays KindUnknown.
		child := buildOIDPath(root, 1, 1, 1, 1)
		child.setName("childNode")

		objRefs := []objectTypeRef{
			{mod: mod, obj: &module.ObjectType{
				Name:   "myEntry",
				Syntax: &module.TypeSyntaxTypeRef{Name: "MyEntry"},
				Index:  []module.IndexItem{{Object: "someIndex"}},
			}},
		}

		inferNodeKinds(ctx, objRefs)

		testutil.Equal(t, KindRow, rowNode.Kind(), "row kind")
		// child was never classified as scalar (it starts as KindInternal from
		// getOrCreateChild), so it should not become column.
		testutil.Equal(t, KindInternal, child.Kind(), "unclassified child kind")
	})
}

func TestResolveTableSemantics(t *testing.T) {
	t.Run("resolved indexes produce no diagnostic", func(t *testing.T) {
		ctx := newTestContext()
		mod := &module.Module{Name: "TEST-MIB"}
		ctx.Modules = append(ctx.Modules, mod)

		root := ctx.Mib.Root()
		idxNode := buildOIDPath(root, 1, 1)
		idxNode.setName("myIndex")
		ctx.registerModuleNodeSymbol(mod, "myIndex", idxNode)

		objRefs := []objectTypeRef{
			{mod: mod, obj: &module.ObjectType{
				Name:   "myEntry",
				Syntax: &module.TypeSyntaxTypeRef{Name: "MyEntry"},
				Index:  []module.IndexItem{{Object: "myIndex"}},
			}},
		}

		resolveTableSemantics(ctx, objRefs)

		testutil.Len(t, ctx.Diagnostics(), 0, "expected no diagnostics for resolved index")
	})

	t.Run("unresolved index emits diagnostic", func(t *testing.T) {
		ctx := newTestContext()
		mod := &module.Module{Name: "TEST-MIB"}
		ctx.Modules = append(ctx.Modules, mod)

		objRefs := []objectTypeRef{
			{mod: mod, obj: &module.ObjectType{
				Name:   "myEntry",
				Syntax: &module.TypeSyntaxTypeRef{Name: "MyEntry"},
				Index:  []module.IndexItem{{Object: "missingIndex"}},
			}},
		}

		resolveTableSemantics(ctx, objRefs)

		diags := ctx.Diagnostics()
		testutil.Len(t, diags, 1, "expected 1 diagnostic for unresolved index")
		testutil.Equal(t, types.DiagIndexUnresolved, diags[0].Code, "diagnostic code")
	})

	t.Run("bare type index skipped", func(t *testing.T) {
		ctx := newTestContext()
		mod := &module.Module{Name: "TEST-MIB"}
		ctx.Modules = append(ctx.Modules, mod)

		// INTEGER is a bare type, so it should not produce a diagnostic
		// even though no node exists for it.
		objRefs := []objectTypeRef{
			{mod: mod, obj: &module.ObjectType{
				Name:   "myEntry",
				Syntax: &module.TypeSyntaxTypeRef{Name: "MyEntry"},
				Index:  []module.IndexItem{{Object: "INTEGER"}},
			}},
		}

		resolveTableSemantics(ctx, objRefs)

		testutil.Len(t, ctx.Diagnostics(), 0, "expected no diagnostic for bare type index")
	})

	t.Run("resolved augments produce no diagnostic", func(t *testing.T) {
		ctx := newTestContext()
		mod := &module.Module{Name: "TEST-MIB"}
		ctx.Modules = append(ctx.Modules, mod)

		root := ctx.Mib.Root()
		augTarget := buildOIDPath(root, 1, 1)
		augTarget.setName("otherEntry")
		ctx.registerModuleNodeSymbol(mod, "otherEntry", augTarget)

		objRefs := []objectTypeRef{
			{mod: mod, obj: &module.ObjectType{
				Name:     "myAugEntry",
				Syntax:   &module.TypeSyntaxTypeRef{Name: "MyAugEntry"},
				Augments: "otherEntry",
			}},
		}

		resolveTableSemantics(ctx, objRefs)

		testutil.Len(t, ctx.Diagnostics(), 0, "expected no diagnostics for resolved augments")
	})

	t.Run("unresolved augments emits diagnostic", func(t *testing.T) {
		ctx := newTestContext()
		mod := &module.Module{Name: "TEST-MIB"}
		ctx.Modules = append(ctx.Modules, mod)

		objRefs := []objectTypeRef{
			{mod: mod, obj: &module.ObjectType{
				Name:     "myAugEntry",
				Syntax:   &module.TypeSyntaxTypeRef{Name: "MyAugEntry"},
				Augments: "missingEntry",
			}},
		}

		resolveTableSemantics(ctx, objRefs)

		diags := ctx.Diagnostics()
		testutil.Len(t, diags, 1, "expected 1 diagnostic for unresolved augments")
		testutil.Equal(t, types.DiagOidOrphan, diags[0].Code, "diagnostic code")
	})

	t.Run("skips objects without index or augments", func(t *testing.T) {
		ctx := newTestContext()
		mod := &module.Module{Name: "TEST-MIB"}
		ctx.Modules = append(ctx.Modules, mod)

		objRefs := []objectTypeRef{
			{mod: mod, obj: &module.ObjectType{
				Name:   "myScalar",
				Syntax: &module.TypeSyntaxTypeRef{Name: "Integer32"},
			}},
		}

		resolveTableSemantics(ctx, objRefs)

		testutil.Len(t, ctx.Diagnostics(), 0, "expected no diagnostics for scalar")
	})

	t.Run("mixed resolved and unresolved indexes", func(t *testing.T) {
		ctx := newTestContext()
		mod := &module.Module{Name: "TEST-MIB"}
		ctx.Modules = append(ctx.Modules, mod)

		root := ctx.Mib.Root()
		idx1 := buildOIDPath(root, 1, 1)
		idx1.setName("goodIndex")
		ctx.registerModuleNodeSymbol(mod, "goodIndex", idx1)

		objRefs := []objectTypeRef{
			{mod: mod, obj: &module.ObjectType{
				Name:   "myEntry",
				Syntax: &module.TypeSyntaxTypeRef{Name: "MyEntry"},
				Index: []module.IndexItem{
					{Object: "goodIndex"},
					{Object: "badIndex"},
				},
			}},
		}

		resolveTableSemantics(ctx, objRefs)

		diags := ctx.Diagnostics()
		testutil.Len(t, diags, 1, "expected 1 diagnostic for unresolved index")
		testutil.Equal(t, types.DiagIndexUnresolved, diags[0].Code, "diagnostic code")
	})
}

func TestLinkObjectIndexes(t *testing.T) {
	t.Run("links indexes with implied flag", func(t *testing.T) {
		ctx := newTestContext()
		mod := &module.Module{Name: "TEST-MIB"}
		ctx.Modules = append(ctx.Modules, mod)
		resolvedMod := newModule(mod.Name)
		ctx.ModuleToResolved[mod] = resolvedMod

		root := ctx.Mib.Root()

		// Create row node and object.
		rowNode := buildOIDPath(root, 1, 1, 1)
		rowNode.setName("myEntry")
		ctx.registerModuleNodeSymbol(mod, "myEntry", rowNode)
		rowObj := newObject("myEntry")
		rowObj.setNode(rowNode)
		resolvedMod.addObject(rowObj)
		rowNode.setObject(rowObj)

		// Create index node and object.
		idxNode := buildOIDPath(root, 1, 1, 1, 1)
		idxNode.setName("myIndex")
		ctx.registerModuleNodeSymbol(mod, "myIndex", idxNode)
		idxObj := newObject("myIndex")
		idxObj.setNode(idxNode)
		idxNode.setObject(idxObj)

		objRefs := []objectTypeRef{
			{mod: mod, obj: &module.ObjectType{
				Name:   "myEntry",
				Syntax: &module.TypeSyntaxTypeRef{Name: "MyEntry"},
				Index:  []module.IndexItem{{Object: "myIndex", Implied: true}},
			}},
		}

		linkObjectIndexes(ctx, objRefs)

		idx := rowObj.Index()
		testutil.Len(t, idx, 1, "index entries")
		testutil.Equal(t, idxObj, idx[0].Object, "index object")
		testutil.True(t, idx[0].Implied, "implied flag")
	})

	t.Run("links augments", func(t *testing.T) {
		ctx := newTestContext()
		mod := &module.Module{Name: "TEST-MIB"}
		ctx.Modules = append(ctx.Modules, mod)
		resolvedMod := newModule(mod.Name)
		ctx.ModuleToResolved[mod] = resolvedMod

		root := ctx.Mib.Root()

		// Create the augmenting row.
		augNode := buildOIDPath(root, 1, 2, 1)
		augNode.setName("myAugEntry")
		ctx.registerModuleNodeSymbol(mod, "myAugEntry", augNode)
		augObj := newObject("myAugEntry")
		augObj.setNode(augNode)
		resolvedMod.addObject(augObj)
		augNode.setObject(augObj)

		// Create the target row being augmented.
		targetNode := buildOIDPath(root, 1, 1, 1)
		targetNode.setName("targetEntry")
		ctx.registerModuleNodeSymbol(mod, "targetEntry", targetNode)
		targetObj := newObject("targetEntry")
		targetObj.setNode(targetNode)
		targetNode.setObject(targetObj)

		objRefs := []objectTypeRef{
			{mod: mod, obj: &module.ObjectType{
				Name:     "myAugEntry",
				Syntax:   &module.TypeSyntaxTypeRef{Name: "MyAugEntry"},
				Augments: "targetEntry",
			}},
		}

		linkObjectIndexes(ctx, objRefs)

		testutil.NotNil(t, augObj.Augments(), "augments should be set")
		testutil.Equal(t, targetObj, augObj.Augments(), "augments target")

		// Verify reverse mapping
		augBy := targetObj.AugmentedBy()
		testutil.Len(t, augBy, 1, "augmented by")
		testutil.Equal(t, augObj, augBy[0], "augmented by entry")
	})

	t.Run("index node without object emits diagnostic", func(t *testing.T) {
		ctx := newTestContext()
		mod := &module.Module{Name: "TEST-MIB"}
		ctx.Modules = append(ctx.Modules, mod)
		resolvedMod := newModule(mod.Name)
		ctx.ModuleToResolved[mod] = resolvedMod

		root := ctx.Mib.Root()

		rowNode := buildOIDPath(root, 1, 1, 1)
		rowNode.setName("myEntry")
		ctx.registerModuleNodeSymbol(mod, "myEntry", rowNode)
		rowObj := newObject("myEntry")
		rowObj.setNode(rowNode)
		resolvedMod.addObject(rowObj)
		rowNode.setObject(rowObj)

		// Index node exists but has no Object attached.
		idxNode := buildOIDPath(root, 1, 1, 1, 1)
		idxNode.setName("bareNode")
		ctx.registerModuleNodeSymbol(mod, "bareNode", idxNode)

		objRefs := []objectTypeRef{
			{mod: mod, obj: &module.ObjectType{
				Name:   "myEntry",
				Syntax: &module.TypeSyntaxTypeRef{Name: "MyEntry"},
				Index:  []module.IndexItem{{Object: "bareNode"}},
			}},
		}

		linkObjectIndexes(ctx, objRefs)

		testutil.Len(t, rowObj.Index(), 0, "index entries should be empty when index node has no object")

		var found bool
		for _, d := range ctx.Diagnostics() {
			if d.Code == types.DiagIndexNotObject {
				found = true
				break
			}
		}
		testutil.True(t, found, "expected DiagIndexNotObject diagnostic")
	})

	t.Run("augments node without object emits diagnostic", func(t *testing.T) {
		ctx := newTestContext()
		mod := &module.Module{Name: "TEST-MIB"}
		ctx.Modules = append(ctx.Modules, mod)
		resolvedMod := newModule(mod.Name)
		ctx.ModuleToResolved[mod] = resolvedMod

		root := ctx.Mib.Root()

		// Create the augmenting row.
		augNode := buildOIDPath(root, 1, 2, 1)
		augNode.setName("myAugEntry")
		ctx.registerModuleNodeSymbol(mod, "myAugEntry", augNode)
		augObj := newObject("myAugEntry")
		augObj.setNode(augNode)
		resolvedMod.addObject(augObj)
		augNode.setObject(augObj)

		// Target node exists but has no Object attached.
		targetNode := buildOIDPath(root, 1, 1, 1)
		targetNode.setName("targetEntry")
		ctx.registerModuleNodeSymbol(mod, "targetEntry", targetNode)

		objRefs := []objectTypeRef{
			{mod: mod, obj: &module.ObjectType{
				Name:     "myAugEntry",
				Syntax:   &module.TypeSyntaxTypeRef{Name: "MyAugEntry"},
				Augments: "targetEntry",
			}},
		}

		linkObjectIndexes(ctx, objRefs)

		testutil.Nil(t, augObj.Augments(), "augments should be nil when target has no object")

		var found bool
		for _, d := range ctx.Diagnostics() {
			if d.Code == types.DiagAugmentsNotObject {
				found = true
				break
			}
		}
		testutil.True(t, found, "expected DiagAugmentsNotObject diagnostic")
	})

	t.Run("bare type index creates TypeName entry", func(t *testing.T) {
		ctx := newTestContext()
		mod := &module.Module{Name: "TEST-MIB"}
		ctx.Modules = append(ctx.Modules, mod)
		resolvedMod := newModule(mod.Name)
		ctx.ModuleToResolved[mod] = resolvedMod

		root := ctx.Mib.Root()

		rowNode := buildOIDPath(root, 1, 1, 1)
		rowNode.setName("myEntry")
		ctx.registerModuleNodeSymbol(mod, "myEntry", rowNode)
		rowObj := newObject("myEntry")
		rowObj.setNode(rowNode)
		resolvedMod.addObject(rowObj)
		rowNode.setObject(rowObj)

		objRefs := []objectTypeRef{
			{mod: mod, obj: &module.ObjectType{
				Name:   "myEntry",
				Syntax: &module.TypeSyntaxTypeRef{Name: "MyEntry"},
				Index:  []module.IndexItem{{Object: "INTEGER"}},
			}},
		}

		linkObjectIndexes(ctx, objRefs)

		idx := rowObj.Index()
		testutil.Len(t, idx, 1, "index entries")
		testutil.Nil(t, idx[0].Object, "bare type index Object should be nil")
		testutil.Equal(t, "INTEGER", idx[0].TypeName, "bare type index TypeName")
	})

	t.Run("nil resolved module skipped", func(_ *testing.T) {
		ctx := newTestContext()
		mod := &module.Module{Name: "TEST-MIB"}
		ctx.Modules = append(ctx.Modules, mod)
		// Don't set ModuleToResolved - should skip without panic.

		objRefs := []objectTypeRef{
			{mod: mod, obj: &module.ObjectType{
				Name:   "myEntry",
				Syntax: &module.TypeSyntaxTypeRef{Name: "MyEntry"},
				Index:  []module.IndexItem{{Object: "myIndex"}},
			}},
		}

		// Should not panic.
		linkObjectIndexes(ctx, objRefs)
	})
}

func TestLookupMemberNode(t *testing.T) {
	t.Run("direct lookup", func(t *testing.T) {
		ctx := newTestContext()
		mod := &module.Module{Name: "TEST-MIB"}
		node := newTestNode("member1")
		ctx.registerModuleNodeSymbol(mod, "member1", node)

		got, ok := lookupMemberNode(ctx, mod, "member1")
		testutil.True(t, ok, "expected to find member1")
		testutil.Equal(t, node, got, "expected member1 node")
	})

	t.Run("import chain lookup", func(t *testing.T) {
		ctx := newTestContext()
		modA := &module.Module{Name: "A"}
		modB := &module.Module{Name: "B"}
		node := newTestNode("member1")
		ctx.registerModuleNodeSymbol(modB, "member1", node)
		ctx.registerImport(modA, "member1", modB)

		got, ok := lookupMemberNode(ctx, modA, "member1")
		testutil.True(t, ok, "expected to find member1 via import")
		testutil.Equal(t, node, got, "expected member1 node via import")
	})

	t.Run("permissive global fallback", func(t *testing.T) {
		modA := &module.Module{Name: "A"}
		modB := &module.Module{Name: "B"}
		node := newTestNode("member1")

		ctx := newResolverContext([]*module.Module{modA, modB}, nil, PermissiveConfig())
		ctx.registerModuleNodeSymbol(modB, "member1", node)
		// No import from A to B, but permissive mode should find it globally.

		got, ok := lookupMemberNode(ctx, modA, "member1")
		testutil.True(t, ok, "expected permissive fallback to find member1")
		testutil.Equal(t, node, got, "expected member1 via permissive fallback")
	})

	t.Run("strict mode no fallback", func(t *testing.T) {
		modA := &module.Module{Name: "A"}
		modB := &module.Module{Name: "B"}
		node := newTestNode("member1")

		ctx := newResolverContext([]*module.Module{modA, modB}, nil, StrictConfig())
		ctx.registerModuleNodeSymbol(modB, "member1", node)
		// No import from A to B.

		_, ok := lookupMemberNode(ctx, modA, "member1")
		testutil.False(t, ok, "expected strict mode to not find member1 without import")
	})

	t.Run("not found in any mode", func(t *testing.T) {
		modA := &module.Module{Name: "A"}
		ctx := newResolverContext([]*module.Module{modA}, nil, PermissiveConfig())

		_, ok := lookupMemberNode(ctx, modA, "nonexistent")
		testutil.False(t, ok, "expected false for nonexistent member")
	})
}

func TestCreateResolvedObjectGroups(t *testing.T) {
	t.Run("creates group with members", func(t *testing.T) {
		mod := &module.Module{
			Name: "TEST-MIB",
			Definitions: []module.Definition{
				&module.ObjectGroup{
					Name:        "testGroup",
					Objects:     []string{"obj1", "obj2"},
					Status:      types.StatusCurrent,
					Description: "test group",
				},
			},
		}

		ctx := newResolverContext([]*module.Module{mod}, nil, DefaultConfig())
		ctx.ModuleIndex[mod.Name] = []*module.Module{mod}
		resolvedMod := newModule(mod.Name)
		ctx.ModuleToResolved[mod] = resolvedMod
		ctx.ResolvedToModule[resolvedMod] = mod

		root := ctx.Mib.Root()

		// Group node.
		grpNode := buildOIDPath(root, 1, 1)
		grpNode.setName("testGroup")
		ctx.registerModuleNodeSymbol(mod, "testGroup", grpNode)

		// Member nodes.
		m1 := buildOIDPath(root, 1, 2)
		m1.setName("obj1")
		ctx.registerModuleNodeSymbol(mod, "obj1", m1)

		m2 := buildOIDPath(root, 1, 3)
		m2.setName("obj2")
		ctx.registerModuleNodeSymbol(mod, "obj2", m2)

		count := createResolvedObjectGroups(ctx)
		testutil.Equal(t, 1, count, "created count")

		groups := ctx.Mib.Groups()
		testutil.Len(t, groups, 1, "mib groups")
		testutil.Equal(t, "testGroup", groups[0].Name(), "group name")
		testutil.False(t, groups[0].IsNotificationGroup(), "should not be notification group")
		testutil.Len(t, groups[0].Members(), 2, "members")
		testutil.Equal(t, m1, groups[0].Members()[0], "member 0")
		testutil.Equal(t, m2, groups[0].Members()[1], "member 1")

		// Verify registered on node.
		testutil.NotNil(t, grpNode.Group(), "group should be set on node")
		testutil.Equal(t, "testGroup", grpNode.Group().Name(), "group name on node")

		// Verify registered on module.
		testutil.Len(t, resolvedMod.Groups(), 1, "module groups")
	})

	t.Run("not-accessible member emits diagnostic", func(t *testing.T) {
		mod := &module.Module{
			Name: "TEST-MIB",
			Definitions: []module.Definition{
				&module.ObjectGroup{
					Name:    "testGroup",
					Objects: []string{"notAccessObj"},
				},
			},
		}

		ctx := newResolverContext([]*module.Module{mod}, nil, DefaultConfig())
		ctx.ModuleIndex[mod.Name] = []*module.Module{mod}
		resolvedMod := newModule(mod.Name)
		ctx.ModuleToResolved[mod] = resolvedMod
		ctx.ResolvedToModule[resolvedMod] = mod

		root := ctx.Mib.Root()

		grpNode := buildOIDPath(root, 1, 1)
		grpNode.setName("testGroup")
		ctx.registerModuleNodeSymbol(mod, "testGroup", grpNode)

		memberNode := buildOIDPath(root, 1, 2)
		memberNode.setName("notAccessObj")
		ctx.registerModuleNodeSymbol(mod, "notAccessObj", memberNode)

		// Attach an Object with not-accessible access.
		memberObj := newObject("notAccessObj")
		memberObj.setAccess(types.AccessNotAccessible)
		memberNode.setObject(memberObj)

		createResolvedObjectGroups(ctx)

		var found bool
		for _, d := range ctx.Diagnostics() {
			if d.Code == types.DiagGroupNotAccessible {
				found = true
				break
			}
		}
		testutil.True(t, found, "expected not-accessible diagnostic")
	})

	t.Run("unresolved group node skipped", func(t *testing.T) {
		mod := &module.Module{
			Name: "TEST-MIB",
			Definitions: []module.Definition{
				&module.ObjectGroup{
					Name:    "missingGroup",
					Objects: []string{"obj1"},
				},
			},
		}

		ctx := newResolverContext([]*module.Module{mod}, nil, DefaultConfig())
		ctx.ModuleIndex[mod.Name] = []*module.Module{mod}
		resolvedMod := newModule(mod.Name)
		ctx.ModuleToResolved[mod] = resolvedMod
		ctx.ResolvedToModule[resolvedMod] = mod

		count := createResolvedObjectGroups(ctx)
		testutil.Equal(t, 0, count, "should create no groups when node is missing")
	})

	t.Run("unresolved member emits diagnostic", func(t *testing.T) {
		mod := &module.Module{
			Name: "TEST-MIB",
			Definitions: []module.Definition{
				&module.ObjectGroup{
					Name:    "testGroup",
					Objects: []string{"obj1", "missing"},
				},
			},
		}

		ctx := newResolverContext([]*module.Module{mod}, nil, DefaultConfig())
		ctx.ModuleIndex[mod.Name] = []*module.Module{mod}
		resolvedMod := newModule(mod.Name)
		ctx.ModuleToResolved[mod] = resolvedMod
		ctx.ResolvedToModule[resolvedMod] = mod

		root := ctx.Mib.Root()

		grpNode := buildOIDPath(root, 1, 1)
		grpNode.setName("testGroup")
		ctx.registerModuleNodeSymbol(mod, "testGroup", grpNode)

		m1 := buildOIDPath(root, 1, 2)
		m1.setName("obj1")
		ctx.registerModuleNodeSymbol(mod, "obj1", m1)
		// "missing" is not registered.

		createResolvedObjectGroups(ctx)

		groups := ctx.Mib.Groups()
		testutil.Len(t, groups, 1, "mib groups")
		testutil.Len(t, groups[0].Members(), 1, "only resolved member should be present")
		testutil.Equal(t, m1, groups[0].Members()[0], "resolved member")

		var found bool
		for _, d := range ctx.Diagnostics() {
			if d.Code == types.DiagGroupMemberUnresolved {
				found = true
				break
			}
		}
		testutil.True(t, found, "expected DiagGroupMemberUnresolved diagnostic")
	})
}

func TestCreateResolvedNotificationGroups(t *testing.T) {
	t.Run("creates notification group with members", func(t *testing.T) {
		mod := &module.Module{
			Name: "TEST-MIB",
			Definitions: []module.Definition{
				&module.NotificationGroup{
					Name:          "testNotifGroup",
					Notifications: []string{"notif1", "notif2"},
					Status:        types.StatusCurrent,
					Description:   "notification group",
				},
			},
		}

		ctx := newResolverContext([]*module.Module{mod}, nil, DefaultConfig())
		ctx.ModuleIndex[mod.Name] = []*module.Module{mod}
		resolvedMod := newModule(mod.Name)
		ctx.ModuleToResolved[mod] = resolvedMod
		ctx.ResolvedToModule[resolvedMod] = mod

		root := ctx.Mib.Root()

		grpNode := buildOIDPath(root, 1, 1)
		grpNode.setName("testNotifGroup")
		ctx.registerModuleNodeSymbol(mod, "testNotifGroup", grpNode)

		n1 := buildOIDPath(root, 1, 2)
		n1.setName("notif1")
		ctx.registerModuleNodeSymbol(mod, "notif1", n1)

		n2 := buildOIDPath(root, 1, 3)
		n2.setName("notif2")
		ctx.registerModuleNodeSymbol(mod, "notif2", n2)

		count := createResolvedNotificationGroups(ctx)
		testutil.Equal(t, 1, count, "created count")

		groups := ctx.Mib.Groups()
		testutil.Len(t, groups, 1, "mib groups")
		testutil.True(t, groups[0].IsNotificationGroup(), "should be notification group")
		testutil.Len(t, groups[0].Members(), 2, "members")
	})

	t.Run("unresolved member emits diagnostic", func(t *testing.T) {
		mod := &module.Module{
			Name: "TEST-MIB",
			Definitions: []module.Definition{
				&module.NotificationGroup{
					Name:          "testNotifGroup",
					Notifications: []string{"notif1", "missing"},
				},
			},
		}

		ctx := newResolverContext([]*module.Module{mod}, nil, DefaultConfig())
		ctx.ModuleIndex[mod.Name] = []*module.Module{mod}
		resolvedMod := newModule(mod.Name)
		ctx.ModuleToResolved[mod] = resolvedMod
		ctx.ResolvedToModule[resolvedMod] = mod

		root := ctx.Mib.Root()

		grpNode := buildOIDPath(root, 1, 1)
		grpNode.setName("testNotifGroup")
		ctx.registerModuleNodeSymbol(mod, "testNotifGroup", grpNode)

		n1 := buildOIDPath(root, 1, 2)
		n1.setName("notif1")
		ctx.registerModuleNodeSymbol(mod, "notif1", n1)
		// "missing" is not registered.

		createResolvedNotificationGroups(ctx)

		groups := ctx.Mib.Groups()
		testutil.Len(t, groups, 1, "mib groups")
		testutil.Len(t, groups[0].Members(), 1, "only resolved member should be present")
		testutil.Equal(t, n1, groups[0].Members()[0], "resolved member")

		var found bool
		for _, d := range ctx.Diagnostics() {
			if d.Code == types.DiagGroupMemberUnresolved {
				found = true
				break
			}
		}
		testutil.True(t, found, "expected DiagGroupMemberUnresolved diagnostic")
	})
}

func TestRegisterNotification(t *testing.T) {
	mod := &module.Module{Name: "TEST-MIB"}
	ctx := newResolverContext([]*module.Module{mod}, nil, DefaultConfig())
	resolvedMod := newModule(mod.Name)
	ctx.ModuleToResolved[mod] = resolvedMod

	node := buildOIDPath(ctx.Mib.Root(), 1, 1)
	notif := newNotification("testNotif")

	registerNotification(ctx, mod, node, notif)

	testutil.Len(t, ctx.Mib.Notifications(), 1, "mib notifications")
	testutil.NotNil(t, node.Notification(), "node notification")
	testutil.Equal(t, notif, node.Notification(), "node notification")
	testutil.Len(t, resolvedMod.Notifications(), 1, "module notifications")
}

func TestRegisterGroup(t *testing.T) {
	mod := &module.Module{Name: "TEST-MIB"}
	ctx := newResolverContext([]*module.Module{mod}, nil, DefaultConfig())
	resolvedMod := newModule(mod.Name)
	ctx.ModuleToResolved[mod] = resolvedMod

	node := buildOIDPath(ctx.Mib.Root(), 1, 1)
	grp := newGroup("testGroup")

	registerGroup(ctx, mod, node, grp)

	testutil.Len(t, ctx.Mib.Groups(), 1, "mib groups")
	testutil.NotNil(t, node.Group(), "node group")
	testutil.Equal(t, grp, node.Group(), "node group")
	testutil.Len(t, resolvedMod.Groups(), 1, "module groups")
}

func TestRegisterCompliance(t *testing.T) {
	mod := &module.Module{Name: "TEST-MIB"}
	ctx := newResolverContext([]*module.Module{mod}, nil, DefaultConfig())
	resolvedMod := newModule(mod.Name)
	ctx.ModuleToResolved[mod] = resolvedMod

	node := buildOIDPath(ctx.Mib.Root(), 1, 1)
	comp := newCompliance("testCompliance")

	registerCompliance(ctx, mod, node, comp)

	testutil.Len(t, ctx.Mib.Compliances(), 1, "mib compliances")
	testutil.NotNil(t, node.Compliance(), "node compliance")
	testutil.Equal(t, comp, node.Compliance(), "node compliance")
	testutil.Len(t, resolvedMod.Compliances(), 1, "module compliances")
}

func TestRegisterCapability(t *testing.T) {
	mod := &module.Module{Name: "TEST-MIB"}
	ctx := newResolverContext([]*module.Module{mod}, nil, DefaultConfig())
	resolvedMod := newModule(mod.Name)
	ctx.ModuleToResolved[mod] = resolvedMod

	node := buildOIDPath(ctx.Mib.Root(), 1, 1)
	capability := newCapability("testCapability")

	registerCapability(ctx, mod, node, capability)

	testutil.Len(t, ctx.Mib.Capabilities(), 1, "mib capabilities")
	testutil.NotNil(t, node.Capability(), "node capability")
	testutil.Equal(t, capability, node.Capability(), "node capability")
	testutil.Len(t, resolvedMod.Capabilities(), 1, "module capabilities")
}

func TestCreateResolvedNotifications_FullCycle(t *testing.T) {
	t.Run("creates notification with objects", func(t *testing.T) {
		mod := &module.Module{
			Name: "TEST-MIB",
			Definitions: []module.Definition{
				&module.Notification{
					Name:    "myNotif",
					Objects: []string{"obj1", "obj2"},
					Status:  types.StatusCurrent,
				},
			},
		}

		ctx := newResolverContext([]*module.Module{mod}, nil, DefaultConfig())
		ctx.ModuleIndex[mod.Name] = []*module.Module{mod}
		resolvedMod := newModule(mod.Name)
		ctx.ModuleToResolved[mod] = resolvedMod
		ctx.ResolvedToModule[resolvedMod] = mod

		root := ctx.Mib.Root()

		notifNode := buildOIDPath(root, 1, 1)
		notifNode.setName("myNotif")
		ctx.registerModuleNodeSymbol(mod, "myNotif", notifNode)

		obj1Node := buildOIDPath(root, 1, 2)
		obj1Node.setName("obj1")
		ctx.registerModuleNodeSymbol(mod, "obj1", obj1Node)
		obj1 := newObject("obj1")
		obj1Node.setObject(obj1)

		obj2Node := buildOIDPath(root, 1, 3)
		obj2Node.setName("obj2")
		ctx.registerModuleNodeSymbol(mod, "obj2", obj2Node)
		obj2 := newObject("obj2")
		obj2Node.setObject(obj2)

		createResolvedNotifications(ctx)

		notifs := ctx.Mib.Notifications()
		testutil.Len(t, notifs, 1, "mib notifications")
		testutil.Equal(t, "myNotif", notifs[0].Name(), "notification name")
		testutil.Len(t, notifs[0].Objects(), 2, "notification objects")
	})

	t.Run("unresolved object emits diagnostic", func(t *testing.T) {
		mod := &module.Module{
			Name: "TEST-MIB",
			Definitions: []module.Definition{
				&module.Notification{
					Name:    "myNotif",
					Objects: []string{"missingObj"},
				},
			},
		}

		ctx := newResolverContext([]*module.Module{mod}, nil, DefaultConfig())
		ctx.ModuleIndex[mod.Name] = []*module.Module{mod}
		resolvedMod := newModule(mod.Name)
		ctx.ModuleToResolved[mod] = resolvedMod
		ctx.ResolvedToModule[resolvedMod] = mod

		root := ctx.Mib.Root()
		notifNode := buildOIDPath(root, 1, 1)
		notifNode.setName("myNotif")
		ctx.registerModuleNodeSymbol(mod, "myNotif", notifNode)

		createResolvedNotifications(ctx)

		diags := ctx.Diagnostics()
		testutil.Len(t, diags, 1, "expected 1 diagnostic for unresolved notification object")
		testutil.Equal(t, types.DiagObjectsUnresolved, diags[0].Code, "diagnostic code")
	})

	t.Run("permissive global lookup for notification objects", func(t *testing.T) {
		modA := &module.Module{
			Name: "A-MIB",
			Definitions: []module.Definition{
				&module.Notification{
					Name:    "myNotif",
					Objects: []string{"foreignObj"},
				},
			},
		}
		modB := &module.Module{Name: "B-MIB"}

		ctx := newResolverContext([]*module.Module{modA, modB}, nil, PermissiveConfig())
		ctx.ModuleIndex[modA.Name] = []*module.Module{modA}
		ctx.ModuleIndex[modB.Name] = []*module.Module{modB}
		resolvedModA := newModule(modA.Name)
		ctx.ModuleToResolved[modA] = resolvedModA
		ctx.ResolvedToModule[resolvedModA] = modA
		resolvedModB := newModule(modB.Name)
		ctx.ModuleToResolved[modB] = resolvedModB
		ctx.ResolvedToModule[resolvedModB] = modB

		root := ctx.Mib.Root()

		notifNode := buildOIDPath(root, 1, 1)
		notifNode.setName("myNotif")
		ctx.registerModuleNodeSymbol(modA, "myNotif", notifNode)

		// Object registered only in modB, not imported by modA.
		foreignNode := buildOIDPath(root, 2, 1)
		foreignNode.setName("foreignObj")
		ctx.registerModuleNodeSymbol(modB, "foreignObj", foreignNode)
		foreignObj := newObject("foreignObj")
		foreignNode.setObject(foreignObj)

		createResolvedNotifications(ctx)

		notifs := ctx.Mib.Notifications()
		testutil.Len(t, notifs, 1, "mib notifications")
		testutil.Len(t, notifs[0].Objects(), 1, "notification objects via permissive fallback")
	})

	t.Run("trap info preserved", func(t *testing.T) {
		mod := &module.Module{
			Name: "TEST-MIB",
			Definitions: []module.Definition{
				&module.Notification{
					Name: "myTrap",
					TrapInfo: &module.TrapInfo{
						Enterprise: "enterprises",
						TrapNumber: 42,
					},
				},
			},
		}

		ctx := newResolverContext([]*module.Module{mod}, nil, DefaultConfig())
		ctx.ModuleIndex[mod.Name] = []*module.Module{mod}
		resolvedMod := newModule(mod.Name)
		ctx.ModuleToResolved[mod] = resolvedMod
		ctx.ResolvedToModule[resolvedMod] = mod

		root := ctx.Mib.Root()
		trapNode := buildOIDPath(root, 1, 1)
		trapNode.setName("myTrap")
		ctx.registerModuleNodeSymbol(mod, "myTrap", trapNode)

		createResolvedNotifications(ctx)

		notifs := ctx.Mib.Notifications()
		testutil.Len(t, notifs, 1, "mib notifications")
		ti := notifs[0].TrapInfo()
		testutil.NotNil(t, ti, "trap info")
		testutil.Equal(t, "enterprises", ti.Enterprise, "enterprise")
		testutil.Equal(t, 42, ti.TrapNumber, "trap number")
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
		if d.Code == types.DiagNotifObjectNotObject {
			found = true
			break
		}
	}
	testutil.True(t, found, "expected diagnostic for notification object with nil Object, got none")
}
