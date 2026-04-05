package resolver

import (
	"testing"

	"github.com/golangsnmp/gomib/internal/model"
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
	// Build a resolver context with base modules so type lookups work.
	ctx := newResolverContext(nil, model.ResolverNormal, model.DefaultConfig())
	registerModules(ctx, nil)
	resolveTypes(ctx)
	// Use SNMPv2-TC module for lookups (where AutonomousType etc. are defined).
	var tcMod *module.Module
	for _, m := range ctx.modules {
		if m.Name == "SNMPv2-TC" {
			tcMod = m
			break
		}
	}

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
			name:   "TypeRef VariablePointer",
			syntax: &module.TypeSyntaxTypeRef{Name: "VariablePointer"},
			want:   true,
		},
		{
			name:   "TypeRef RowPointer",
			syntax: &module.TypeSyntaxTypeRef{Name: "RowPointer"},
			want:   true,
		},
		{
			name:   "TypeRef TDomain",
			syntax: &module.TypeSyntaxTypeRef{Name: "TDomain"},
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
			testutil.Equal(t, tt.want, isOIDType(ctx, tcMod, tt.syntax), "isOIDType()")
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
			got, valid := binaryToBytes(tt.input, tt.rightPad)
			testutil.True(t, valid, "binaryToBytes() valid")
			testutil.Len(t, got, len(tt.want), "binaryToBytes() len")
			for i := range got {
				testutil.Equal(t, tt.want[i], got[i], "binaryToBytes()")
			}
		})
	}

	// Invalid binary digits
	t.Run("invalid digits", func(t *testing.T) {
		got, valid := binaryToBytes("102", false)
		testutil.False(t, valid, "binaryToBytes() should report invalid")
		testutil.Len(t, got, 1, "binaryToBytes() still produces output")
	})
}

func TestConvertDefVal(t *testing.T) {
	t.Run("integer", func(t *testing.T) {
		ctx := newTestContext()
		mod := &module.Module{Name: "TEST-MIB"}
		syntax := &module.TypeSyntaxTypeRef{Name: "Integer32"}

		tests := []struct {
			name  string
			input int64
			raw   string
		}{
			{"positive", 42, "42"},
			{"negative", -1, "-1"},
			{"zero", 0, "0"},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				dv := convertDefVal(ctx, &module.DefValInteger{Value: tt.input}, mod, syntax, types.Span{})
				testutil.NotNil(t, dv, "convertDefVal returned nil")
				testutil.Equal(t, model.DefValKindInt, dv.Kind(), "kind")
				v, ok := model.DefValAs[int64](*dv)
				testutil.True(t, ok, "model.DefValAs[int64] ok")
				testutil.Equal(t, tt.input, v, "value")
				testutil.Equal(t, tt.raw, dv.Raw(), "raw")
			})
		}
	})

	t.Run("unsigned", func(t *testing.T) {
		ctx := newTestContext()
		mod := &module.Module{Name: "TEST-MIB"}
		syntax := &module.TypeSyntaxTypeRef{Name: "Counter64"}

		dv := convertDefVal(ctx, &module.DefValUnsigned{Value: 12345}, mod, syntax, types.Span{})
		testutil.NotNil(t, dv, "convertDefVal returned nil")
		testutil.Equal(t, model.DefValKindUint, dv.Kind(), "kind")
		v, ok := model.DefValAs[uint64](*dv)
		testutil.True(t, ok, "model.DefValAs[uint64] ok")
		testutil.Equal(t, uint64(12345), v, "value")
		testutil.Equal(t, "12345", dv.Raw(), "raw")
	})

	t.Run("string", func(t *testing.T) {
		ctx := newTestContext()
		mod := &module.Module{Name: "TEST-MIB"}

		dv := convertDefVal(ctx, &module.DefValString{Value: "public"}, mod, &module.TypeSyntaxTypeRef{Name: "DisplayString"}, types.Span{})
		testutil.NotNil(t, dv, "convertDefVal returned nil")
		testutil.Equal(t, model.DefValKindString, dv.Kind(), "kind")
		v, ok := model.DefValAs[string](*dv)
		testutil.True(t, ok, "model.DefValAs[string] ok")
		testutil.Equal(t, "public", v, "value")
		testutil.Equal(t, `"public"`, dv.Raw(), "raw")
	})

	t.Run("hex string", func(t *testing.T) {
		ctx := newTestContext()
		mod := &module.Module{Name: "TEST-MIB"}

		dv := convertDefVal(ctx, &module.DefValHexString{Value: "FF00"}, mod, &module.TypeSyntaxOctetString{}, types.Span{})
		testutil.NotNil(t, dv, "convertDefVal returned nil")
		testutil.Equal(t, model.DefValKindBytes, dv.Kind(), "kind")
		bytes, ok := model.DefValAs[[]byte](*dv)
		testutil.True(t, ok, "value is not []byte")
		want := []byte{0xFF, 0x00}
		testutil.Len(t, bytes, len(want), "bytes len")
		for i := range bytes {
			testutil.Equal(t, want[i], bytes[i], "bytes[")
		}
		testutil.Equal(t, "'FF00'H", dv.Raw(), "raw")
	})

	t.Run("binary string", func(t *testing.T) {
		ctx := newTestContext()
		mod := &module.Module{Name: "TEST-MIB"}

		dv := convertDefVal(ctx, &module.DefValBinaryString{Value: "10101010"}, mod, &module.TypeSyntaxOctetString{}, types.Span{})
		testutil.NotNil(t, dv, "convertDefVal returned nil")
		testutil.Equal(t, model.DefValKindBytes, dv.Kind(), "kind")
		bytes, ok := model.DefValAs[[]byte](*dv)
		testutil.True(t, ok, "value is not []byte")
		testutil.Len(t, bytes, 1, "bytes len")
		testutil.Equal(t, byte(0xAA), bytes[0], "bytes[0]")
		testutil.Equal(t, "'10101010'B", dv.Raw(), "raw")
	})

	t.Run("enum", func(t *testing.T) {
		ctx := newTestContext()
		mod := &module.Module{Name: "TEST-MIB"}
		syntax := &module.TypeSyntaxIntegerEnum{
			NamedNumbers: []module.NamedNumber{
				{Name: "enabled", Value: 1},
				{Name: "disabled", Value: 2},
			},
		}

		dv := convertDefVal(ctx, &module.DefValEnum{Name: "enabled"}, mod, syntax, types.Span{})
		testutil.NotNil(t, dv, "convertDefVal returned nil")
		testutil.Equal(t, model.DefValKindEnum, dv.Kind(), "kind")
		v, ok := model.DefValAs[string](*dv)
		testutil.True(t, ok, "model.DefValAs[string] ok")
		testutil.Equal(t, "enabled", v, "value")
		testutil.Equal(t, "enabled", dv.Raw(), "raw")
	})

	t.Run("enum on OID type resolves to OID", func(t *testing.T) {
		ctx := newTestContext()
		mod := &module.Module{Name: "TEST-MIB"}

		root := ctx.mib.Root()
		child := model.GetOrCreateChild(root, 1)
		grandchild := model.GetOrCreateChild(child, 3)
		model.SetNodeName(grandchild, "myTarget")
		ctx.registerModuleNodeSymbol(mod, "myTarget", grandchild)

		syntax := &module.TypeSyntaxObjectIdentifier{}
		dv := convertDefVal(ctx, &module.DefValEnum{Name: "myTarget"}, mod, syntax, types.Span{})
		testutil.NotNil(t, dv, "convertDefVal returned nil")
		testutil.Equal(t, model.DefValKindOID, dv.Kind(), "kind")
		oid, ok := model.DefValAs[model.OID](*dv)
		testutil.True(t, ok, "value is not OID")
		testutil.Len(t, oid, 2, "oid len")
		testutil.Equal(t, uint32(1), oid[0], "oid[0]")
		testutil.Equal(t, uint32(3), oid[1], "oid[1]")
	})

	t.Run("enum on OID type falls back when unresolved", func(t *testing.T) {
		ctx := newTestContext()
		mod := &module.Module{Name: "TEST-MIB"}
		syntax := &module.TypeSyntaxObjectIdentifier{}

		dv := convertDefVal(ctx, &module.DefValEnum{Name: "unknownOid"}, mod, syntax, types.Span{})
		testutil.NotNil(t, dv, "convertDefVal returned nil")
		testutil.Equal(t, model.DefValKindEnum, dv.Kind(), "kind")
	})

	t.Run("bits", func(t *testing.T) {
		ctx := newTestContext()
		mod := &module.Module{Name: "TEST-MIB"}
		syntax := &module.TypeSyntaxBits{}

		dv := convertDefVal(ctx, &module.DefValBits{Labels: []string{"flag1", "flag2"}}, mod, syntax, types.Span{})
		testutil.NotNil(t, dv, "convertDefVal returned nil")
		testutil.Equal(t, model.DefValKindBits, dv.Kind(), "kind")
		labels, ok := model.DefValAs[[]string](*dv)
		testutil.True(t, ok, "value is not []string")
		testutil.Len(t, labels, 2, "labels len")
		testutil.Equal(t, "flag1", labels[0], "labels[0]")
		testutil.Equal(t, "flag2", labels[1], "labels[1]")
		testutil.Equal(t, "{ flag1, flag2 }", dv.Raw(), "raw")
	})

	t.Run("bits empty", func(t *testing.T) {
		ctx := newTestContext()
		mod := &module.Module{Name: "TEST-MIB"}
		syntax := &module.TypeSyntaxBits{}

		dv := convertDefVal(ctx, &module.DefValBits{Labels: []string{}}, mod, syntax, types.Span{})
		testutil.NotNil(t, dv, "convertDefVal returned nil")
		testutil.Equal(t, "{ }", dv.Raw(), "raw")
	})

	t.Run("OID ref resolved", func(t *testing.T) {
		ctx := newTestContext()
		mod := &module.Module{Name: "TEST-MIB"}

		root := ctx.mib.Root()
		child := model.GetOrCreateChild(root, 1)
		model.SetNodeName(child, "sysName")
		ctx.registerModuleNodeSymbol(mod, "sysName", child)

		dv := convertDefVal(ctx, &module.DefValOidRef{Name: "sysName"}, mod, nil, types.Span{})
		testutil.NotNil(t, dv, "convertDefVal returned nil")
		testutil.Equal(t, model.DefValKindOID, dv.Kind(), "kind")
	})

	t.Run("OID ref unresolved", func(t *testing.T) {
		ctx := newTestContext()
		mod := &module.Module{Name: "TEST-MIB"}

		dv := convertDefVal(ctx, &module.DefValOidRef{Name: "missing"}, mod, nil, types.Span{})
		testutil.Nil(t, dv, "expected nil for unresolved OID ref, got")
	})

	t.Run("unparsed emits diagnostic", func(t *testing.T) {
		ctx := newResolverContext(nil, model.ResolverNormal, model.VerboseConfig())
		mod := &module.Module{Name: "TEST-MIB"}

		dv := convertDefVal(ctx, &module.DefValUnparsed{}, mod, nil, types.Span{})
		testutil.Nil(t, dv, "expected nil for unparsed DefVal, got")

		hasDiag(t, ctx.Diagnostics(), types.DiagDefvalUnresolved)
	})
}

func TestResolveTypeSyntax(t *testing.T) {
	ctx := newTestContext()
	mod := &module.Module{Name: "TEST-MIB"}
	ctx.modules = append(ctx.modules, mod)

	intType := model.NewType("Integer32")
	model.SetTypeBase(intType, model.BaseInteger32)
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
	ctx := newResolverContext(nil, model.ResolverNormal, model.DefaultConfig())
	ctx.modules = []*module.Module{smiMod}
	ctx.snmpv2SMIModule = smiMod

	integerType := model.NewType("INTEGER")
	model.SetTypeBase(integerType, model.BaseInteger32)
	ctx.registerModuleTypeSymbol(smiMod, "INTEGER", integerType)

	bitsType := model.NewType("BITS")
	model.SetTypeBase(bitsType, model.BaseBits)
	ctx.registerModuleTypeSymbol(smiMod, "BITS", bitsType)

	octetType := model.NewType("OCTET STRING")
	model.SetTypeBase(octetType, model.BaseOctetString)
	ctx.registerModuleTypeSymbol(smiMod, "OCTET STRING", octetType)

	oidType := model.NewType("OBJECT IDENTIFIER")
	model.SetTypeBase(oidType, model.BaseObjectIdentifier)
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
		obj := model.NewObject("testObj")
		// No type set, should not panic
		computeEffectiveValues(obj)
	})

	t.Run("inherits display hint from type", func(t *testing.T) {
		typ := model.NewType("DisplayString")
		model.SetTypeDisplayHint(typ, "255a")

		obj := model.NewObject("testObj")
		model.SetObjectType(obj, typ)
		computeEffectiveValues(obj)

		testutil.Equal(t, "255a", obj.EffectiveDisplayHint(), "hint")
	})

	t.Run("object hint takes precedence over type hint", func(t *testing.T) {
		typ := model.NewType("DisplayString")
		model.SetTypeDisplayHint(typ, "255a")

		obj := model.NewObject("testObj")
		model.SetObjectType(obj, typ)
		model.SetObjectEffectiveHint(obj, "1d")
		computeEffectiveValues(obj)

		testutil.Equal(t, "1d", obj.EffectiveDisplayHint(), "hint")
	})

	t.Run("inherits sizes from type chain", func(t *testing.T) {
		parentType := model.NewType("OCTET STRING")
		model.SetTypeSizes(parentType, []model.Range{{Min: 0, Max: 255}})

		childType := model.NewType("DisplayString")
		model.SetTypeParent(childType, parentType)

		obj := model.NewObject("testObj")
		model.SetObjectType(obj, childType)
		computeEffectiveValues(obj)

		sizes := obj.EffectiveSizes()
		testutil.Len(t, sizes, 1, "sizes len")
		testutil.Equal(t, int64(0), sizes[0].Min, "sizes[0].Min")
		testutil.Equal(t, int64(255), sizes[0].Max, "sizes[0].Max")
	})

	t.Run("object sizes take precedence", func(t *testing.T) {
		typ := model.NewType("DisplayString")
		model.SetTypeSizes(typ, []model.Range{{Min: 0, Max: 255}})

		obj := model.NewObject("testObj")
		model.SetObjectType(obj, typ)
		model.SetObjectEffectiveSizes(obj, []model.Range{{Min: 1, Max: 32}})
		computeEffectiveValues(obj)

		sizes := obj.EffectiveSizes()
		testutil.Len(t, sizes, 1, "sizes len")
		testutil.Equal(t, int64(1), sizes[0].Min, "sizes[0].Min")
		testutil.Equal(t, int64(32), sizes[0].Max, "sizes[0].Max")
	})

	t.Run("inherits ranges from ancestor", func(t *testing.T) {
		grandparent := model.NewType("INTEGER")
		model.SetTypeRanges(grandparent, []model.Range{{Min: -128, Max: 127}})

		parent := model.NewType("Integer32")
		model.SetTypeParent(parent, grandparent)

		child := model.NewType("MyInt")
		model.SetTypeParent(child, parent)

		obj := model.NewObject("testObj")
		model.SetObjectType(obj, child)
		computeEffectiveValues(obj)

		ranges := obj.EffectiveRanges()
		testutil.Len(t, ranges, 1, "ranges len")
		testutil.Equal(t, int64(-128), ranges[0].Min, "ranges[0].Min")
		testutil.Equal(t, int64(127), ranges[0].Max, "ranges[0].Max")
	})

	t.Run("inherits enums from type", func(t *testing.T) {
		typ := model.NewType("StatusType")
		model.SetTypeEnums(typ, []model.NamedValue{
			{Label: "up", Value: 1},
			{Label: "down", Value: 2},
		})

		obj := model.NewObject("testObj")
		model.SetObjectType(obj, typ)
		computeEffectiveValues(obj)

		enums := obj.EffectiveEnums()
		testutil.Len(t, enums, 2, "enums len")
		testutil.Equal(t, "up", enums[0].Label, "enums[0].Label")
		testutil.Equal(t, "down", enums[1].Label, "enums[1].Label")
	})

	t.Run("object enums take precedence over type enums", func(t *testing.T) {
		typ := model.NewType("StatusType")
		model.SetTypeEnums(typ, []model.NamedValue{
			{Label: "up", Value: 1},
			{Label: "down", Value: 2},
		})

		obj := model.NewObject("testObj")
		model.SetObjectType(obj, typ)
		model.SetObjectEffectiveEnums(obj, []model.NamedValue{
			{Label: "active", Value: 1},
		})
		computeEffectiveValues(obj)

		enums := obj.EffectiveEnums()
		testutil.Len(t, enums, 1, "enums len")
		testutil.Equal(t, "active", enums[0].Label, "enums[0].Label")
	})

	t.Run("inherits bits from type", func(t *testing.T) {
		typ := model.NewType("Capabilities")
		model.SetTypeBits(typ, []model.NamedValue{
			{Label: "feature1", Value: 0},
			{Label: "feature2", Value: 1},
		})

		obj := model.NewObject("testObj")
		model.SetObjectType(obj, typ)
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
				MandatoryGroups: nameRefs("ifGeneralGroup", "ifStackGroup"),
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
		testutil.Equal(t, model.AccessReadOnly, *obj0.MinAccess, "min-access")
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

		intType := model.NewType("Integer32")
		model.SetTypeBase(intType, model.BaseInteger32)
		ctx.registerModuleTypeSymbol(mod, "Integer32", intType)

		input := []module.ComplianceModule{
			{
				ModuleName: "TEST-MIB",
				Objects: []module.ComplianceObject{
					{
						Object: "testObj",
						Syntax: &module.TypeSyntaxConstrained{
							Base:       &module.TypeSyntaxTypeRef{Name: "Integer32"},
							Constraint: &module.ConstraintRange{Ranges: []module.Range{module.NewRangeSigned(0, 100, types.Span{})}},
						},
						WriteSyntax: &module.TypeSyntaxConstrained{
							Base:       &module.TypeSyntaxTypeRef{Name: "Integer32"},
							Constraint: &module.ConstraintRange{Ranges: []module.Range{module.NewRangeSigned(1, 50, types.Span{})}},
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
	registerSupportsNode := func(moduleName, name string, kind model.Kind) {
		supMod := &module.Module{Name: moduleName}
		ctx.moduleIndex[moduleName] = append(ctx.moduleIndex[moduleName], supMod)
		node := newTestNode(name)
		model.SetNodeKind(node, kind)
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
		registerSupportsNode("OBJ-MIB", "ifAdminStatus", model.KindColumn)
		registerSupportsNode("OBJ-MIB", "ifOperStatus", model.KindColumn)

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
		testutil.Equal(t, model.AccessReadOnly, *vars[0].Access, "access")
		testutil.Equal(t, "read only", vars[0].Description, "desc")
		testutil.Nil(t, vars[1].Access, "expected nil access for second variation")
	})

	t.Run("notification variations with access", func(t *testing.T) {
		registerSupportsNode("NOTIF-MIB", "linkDown", model.KindNotification)
		registerSupportsNode("NOTIF-MIB", "linkUp", model.KindNotification)

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
		testutil.Equal(t, model.AccessNotImplemented, *vars[0].Access, "access")
		testutil.Equal(t, "not supported", vars[0].Description, "desc")
		testutil.Nil(t, vars[1].Access, "expected nil access for second variation")
	})

	t.Run("not-implemented access preserved in variations", func(t *testing.T) {
		registerSupportsNode("IMPL-MIB", "testObj", model.KindScalar)

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
		testutil.Equal(t, model.AccessNotImplemented, *vars[0].Access, "access")
	})

	t.Run("mixed object and notification variations", func(t *testing.T) {
		registerSupportsNode("MIX-MIB", "ifAdminStatus", model.KindColumn)
		registerSupportsNode("MIX-MIB", "linkDown", model.KindNotification)

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
		intType := model.NewType("Integer32")
		model.SetTypeBase(intType, model.BaseInteger32)
		ctx.registerModuleTypeSymbol(mod, "Integer32", intType)

		input := []module.SupportsModule{
			{
				ModuleName: "IF-MIB",
				Variations: []module.Variation{
					{
						Name: "ifAdminStatus",
						Syntax: &module.TypeSyntaxConstrained{
							Base:       &module.TypeSyntaxTypeRef{Name: "Integer32"},
							Constraint: &module.ConstraintRange{Ranges: []module.Range{module.NewRangeSigned(1, 2, types.Span{})}},
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
		testutil.Equal(t, model.DefValKindInt, vars[0].DefVal.Kind(), "defval kind")
		v, ok := model.DefValAs[int64](vars[0].DefVal)
		testutil.True(t, ok, "model.DefValAs[int64] ok")
		testutil.Equal(t, int64(2), v, "defval value")
	})
}

func TestIsNotificationVariation(t *testing.T) {
	t.Run("permissive global fallback classifies notification", func(t *testing.T) {
		// The variation name is NOT in the SUPPORTS module or the defining
		// module's imports; it exists only in a third module. The node has
		// model.KindNotification. The variation carries a Syntax field, so the
		// syntactic heuristic would incorrectly classify it as an object
		// variation. Permissive global lookup should override the heuristic.
		modDef := &module.Module{Name: "DEF-MIB"}
		modThird := &module.Module{Name: "THIRD-MIB"}

		ctx := newResolverContext(nil, model.ResolverPermissive, model.DefaultConfig())
		ctx.modules = []*module.Module{modDef, modThird}

		notifNode := newTestNode("someNotif")
		model.SetNodeKind(notifNode, model.KindNotification)
		ctx.registerModuleNodeSymbol(modThird, "someNotif", notifNode)

		v := &module.Variation{
			Name:   "someNotif",
			Syntax: &module.TypeSyntaxTypeRef{Name: "INTEGER"},
		}

		got := isNotificationVariation(ctx, modDef, "SUPPORTS-MIB", v)
		testutil.True(t, got, "permissive mode should classify as notification via global lookup")
	})

	t.Run("strict mode falls back to heuristic", func(t *testing.T) {
		// Same setup as above, but strict mode. Global lookup is disabled,
		// so the heuristic runs. Because v.Syntax is non-nil, the heuristic
		// classifies this as an object variation (returns false).
		modDef := &module.Module{Name: "DEF-MIB"}
		modThird := &module.Module{Name: "THIRD-MIB"}

		ctx := newResolverContext(nil, model.ResolverStrict, model.VerboseConfig())
		ctx.modules = []*module.Module{modDef, modThird}

		notifNode := newTestNode("someNotif")
		model.SetNodeKind(notifNode, model.KindNotification)
		ctx.registerModuleNodeSymbol(modThird, "someNotif", notifNode)

		v := &module.Variation{
			Name:   "someNotif",
			Syntax: &module.TypeSyntaxTypeRef{Name: "INTEGER"},
		}

		got := isNotificationVariation(ctx, modDef, "SUPPORTS-MIB", v)
		testutil.False(t, got, "strict mode should use heuristic (Syntax non-nil -> object variation)")
	})

	t.Run("supports module lookup takes priority", func(t *testing.T) {
		// Name found in the SUPPORTS module itself - no global fallback needed.
		modDef := &module.Module{Name: "DEF-MIB"}
		supMod := &module.Module{Name: "SUP-MIB"}

		ctx := newTestContext()
		ctx.modules = []*module.Module{modDef, supMod}
		ctx.moduleIndex["SUP-MIB"] = append(ctx.moduleIndex["SUP-MIB"], supMod)

		notifNode := newTestNode("linkDown")
		model.SetNodeKind(notifNode, model.KindNotification)
		ctx.registerModuleNodeSymbol(supMod, "linkDown", notifNode)

		v := &module.Variation{Name: "linkDown"}
		got := isNotificationVariation(ctx, modDef, "SUP-MIB", v)
		testutil.True(t, got, "should find notification in SUPPORTS module")
	})

	t.Run("import chain lookup takes priority over global", func(t *testing.T) {
		// Name found via the defining module's imports.
		modDef := &module.Module{Name: "DEF-MIB"}
		modImport := &module.Module{Name: "IMPORT-MIB"}

		ctx := newTestContext()
		ctx.modules = []*module.Module{modDef, modImport}
		ctx.registerImport(modDef, "linkDown", modImport)

		notifNode := newTestNode("linkDown")
		model.SetNodeKind(notifNode, model.KindNotification)
		ctx.registerModuleNodeSymbol(modImport, "linkDown", notifNode)

		v := &module.Variation{Name: "linkDown"}
		got := isNotificationVariation(ctx, modDef, "MISSING-MIB", v)
		testutil.True(t, got, "should find notification via import chain")
	})
}

func TestExtractOidRefs(t *testing.T) {
	t.Run("nil oid", func(t *testing.T) {
		testutil.Nil(t, extractOidRefs(nil), "expected nil refs for nil oid")
	})

	t.Run("extracts named symbolic references only", func(t *testing.T) {
		oid := module.NewOidAssignment([]module.OidComponent{
			&module.OidComponentNumber{Value: 1},
			&module.OidComponentName{NameValue: "internet", Span: types.Span{Start: 1, End: 9}},
			&module.OidComponentNamedNumber{NameValue: "private", NumberValue: 4, Span: types.Span{Start: 10, End: 20}},
			&module.OidComponentQualifiedName{ModuleValue: "SNMPv2-SMI", NameValue: "enterprises", Span: types.Span{Start: 21, End: 36}},
			&module.OidComponentQualifiedNamedNumber{ModuleValue: "RFC1155-SMI", NameValue: "private", NumberValue: 4, Span: types.Span{Start: 37, End: 52}},
		}, types.Span{})

		refs := extractOidRefs(&oid)
		testutil.Len(t, refs, 4, "oid refs len")
		testutil.Equal(t, "internet", refs[0].Name, "refs[0].Name")
		testutil.Equal(t, model.ByteOffset(1), refs[0].Span.Start, "refs[0].Span.Start")
		testutil.Equal(t, "private", refs[1].Name, "refs[1].Name")
		testutil.Equal(t, "enterprises", refs[2].Name, "refs[2].Name")
		testutil.Equal(t, "private", refs[3].Name, "refs[3].Name")
	})
}

func TestIsSequenceTypeDef(t *testing.T) {
	imported := &module.Module{
		Name: "IMPORTED-MIB",
	}
	addDefs(imported, []module.Definition{
		&module.TypeDef{
			DefBase: module.DefBase{Name: "ImportedEntry"},
			Syntax:  &module.TypeSyntaxSequence{},
		},
		&module.TypeDef{
			DefBase: module.DefBase{Name: "ImportedText"},
			Syntax:  &module.TypeSyntaxOctetString{},
		},
	})
	local := &module.Module{
		Name: "LOCAL-MIB",
	}
	addDefs(local, []module.Definition{
		&module.TypeDef{
			DefBase: module.DefBase{Name: "LocalEntry"},
			Syntax:  &module.TypeSyntaxSequence{},
		},
	})

	ctx := newTestContextForModules(model.DefaultConfig(), local, imported)
	ctx.importSources[local] = map[string]*module.Module{
		"ImportedEntry": imported,
		"ImportedText":  imported,
	}

	tests := []struct {
		name string
		mod  *module.Module
		typ  string
		want bool
	}{
		{name: "local sequence typedef", mod: local, typ: "LocalEntry", want: true},
		{name: "imported sequence typedef", mod: local, typ: "ImportedEntry", want: true},
		{name: "imported non-sequence typedef", mod: local, typ: "ImportedText", want: false},
		{name: "missing typedef", mod: local, typ: "MissingEntry", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testutil.Equal(t, tt.want, isSequenceTypeDef(ctx, tt.mod, tt.typ), "isSequenceTypeDef()")
		})
	}
}

func TestResolveSyntaxConstraints(t *testing.T) {
	mod := &module.Module{Name: "TEST-MIB"}
	ctx := newTestContextForModules(model.DefaultConfig(), mod)

	intType := model.NewType("Integer32")
	model.SetTypeBase(intType, model.BaseInteger32)
	ctx.registerModuleTypeSymbol(mod, "Integer32", intType)

	t.Run("nil syntax", func(t *testing.T) {
		testutil.Nil(t, resolveSyntaxConstraints(ctx, nil, mod, "testObj", types.Span{}), "expected nil constraints for nil syntax")
	})

	t.Run("type and range constraints", func(t *testing.T) {
		sc := resolveSyntaxConstraints(ctx, &module.TypeSyntaxConstrained{
			Base: &module.TypeSyntaxTypeRef{Name: "Integer32"},
			Constraint: &module.ConstraintRange{
				Ranges: []module.Range{module.NewRangeSigned(1, 10, types.Span{})},
			},
		}, mod, "testObj", types.Span{})

		testutil.NotNil(t, sc, "syntax constraints")
		testutil.Equal(t, intType, sc.Type, "constraint type")
		testutil.Len(t, sc.Ranges, 1, "ranges len")
		testutil.Equal(t, int64(1), sc.Ranges[0].Min, "range min")
		testutil.Equal(t, int64(10), sc.Ranges[0].Max, "range max")
	})

	t.Run("bits syntax keeps bit labels", func(t *testing.T) {
		sc := resolveSyntaxConstraints(ctx, &module.TypeSyntaxBits{
			NamedBits: []module.NamedBit{
				{Name: "featureA", Position: 0},
				{Name: "featureB", Position: 1},
			},
		}, mod, "bitsObj", types.Span{})

		testutil.NotNil(t, sc, "syntax constraints")
		testutil.Len(t, sc.Bits, 2, "bits len")
		testutil.Equal(t, "featureA", sc.Bits[0].Label, "bits[0].Label")
		testutil.Len(t, sc.Enums, 0, "enums len")
	})
}

func TestCreateResolvedObjects(t *testing.T) {
	mod := &module.Module{
		Name: "TEST-MIB",
	}
	addDefs(mod, []module.Definition{
		&module.TypeDef{
			DefBase: module.DefBase{Name: "IfEntry"},
			Syntax:  &module.TypeSyntaxSequence{},
		},
		&module.ObjectType{
			DefBase:     module.DefBase{Name: "sysName"},
			Syntax:      &module.TypeSyntaxTypeRef{Name: "DisplayString"},
			Access:      types.AccessReadOnly,
			Status:      types.StatusCurrent,
			Description: "system name",
			Reference:   "RFC1213",
			Units:       "chars",
			DefVal:      &module.DefValString{Value: "public"},
			Oid: module.NewOidAssignment([]module.OidComponent{
				&module.OidComponentName{NameValue: "testRoot"},
				&module.OidComponentNumber{Value: 1},
			}, types.Span{}),
		},
		&module.ObjectType{
			DefBase: module.DefBase{Name: "ifIndex"},
			Syntax:  &module.TypeSyntaxTypeRef{Name: "Integer32"},
			Access:  types.AccessNotAccessible,
			Status:  types.StatusCurrent,
			Oid: module.NewOidAssignment([]module.OidComponent{
				&module.OidComponentName{NameValue: "ifEntry"},
				&module.OidComponentNumber{Value: 1},
			}, types.Span{}),
		},
		&module.ObjectType{
			DefBase: module.DefBase{Name: "ifEntry"},
			Syntax:  &module.TypeSyntaxTypeRef{Name: "IfEntry"},
			Access:  types.AccessNotAccessible,
			Status:  types.StatusCurrent,
			Index: []module.IndexItem{
				{Object: "ifIndex"},
			},
			Oid: module.NewOidAssignment([]module.OidComponent{
				&module.OidComponentName{NameValue: "testRoot"},
				&module.OidComponentNumber{Value: 2},
			}, types.Span{}),
		},
	})

	ctx := newTestContextForModules(model.DefaultConfig(), mod)

	displayType := model.NewType("DisplayString")
	model.SetTypeBase(displayType, model.BaseOctetString)
	model.SetTypeDisplayHint(displayType, "255a")
	model.SetTypeSizes(displayType, []model.Range{{Min: 0, Max: 255}})
	ctx.registerModuleTypeSymbol(mod, "DisplayString", displayType)

	intType := model.NewType("Integer32")
	model.SetTypeBase(intType, model.BaseInteger32)
	ctx.registerModuleTypeSymbol(mod, "Integer32", intType)

	root := ctx.mib.Root()
	sysNameNode := registerNodeWithSymbol(ctx, mod, root, "sysName", 1, 3, 6, 1, 4, 1)
	model.SetNodeKind(sysNameNode, model.KindScalar)
	ifEntryNode := registerNodeWithSymbol(ctx, mod, root, "ifEntry", 1, 3, 6, 1, 4, 2)
	model.SetNodeKind(ifEntryNode, model.KindRow)
	ifIndexNode := registerNodeWithSymbol(ctx, mod, ifEntryNode, "ifIndex", 1)
	model.SetNodeKind(ifIndexNode, model.KindColumn)

	createResolvedObjects(ctx, collectObjectTypeRefs(ctx))

	resolvedMod := ctx.moduleToResolved[mod]
	testutil.NotNil(t, resolvedMod, "resolved module")

	sysName := resolvedMod.Object("sysName")
	testutil.NotNil(t, sysName, "sysName object")
	testutil.Equal(t, resolvedMod, sysName.Module(), "sysName module")
	testutil.Equal(t, sysNameNode, sysName.Node(), "sysName node")
	testutil.Equal(t, displayType, sysName.Type(), "sysName type")
	testutil.Equal(t, model.AccessReadOnly, sysName.Access(), "sysName access")
	testutil.Equal(t, model.StatusCurrent, sysName.Status(), "sysName status")
	testutil.Equal(t, "system name", sysName.Description(), "sysName description")
	testutil.Equal(t, "RFC1213", sysName.Reference(), "sysName reference")
	testutil.Equal(t, "chars", sysName.Units(), "sysName units")
	testutil.Equal(t, "255a", sysName.EffectiveDisplayHint(), "sysName display hint")
	testutil.Len(t, sysName.EffectiveSizes(), 1, "sysName sizes")
	testutil.Len(t, sysName.OidRefs(), 1, "sysName oid refs")
	testutil.Equal(t, "testRoot", sysName.OidRefs()[0].Name, "sysName oid ref")
	testutil.Equal(t, model.DefValKindString, sysName.DefaultValue().Kind(), "sysName defval kind")
	defval, ok := model.DefValAs[string](sysName.DefaultValue())
	testutil.True(t, ok, "sysName defval type")
	testutil.Equal(t, "public", defval, "sysName defval")
	testutil.Equal(t, sysName, sysNameNode.Object(), "sysName node object")

	ifEntry := resolvedMod.Object("ifEntry")
	testutil.NotNil(t, ifEntry, "ifEntry object")
	testutil.Nil(t, ifEntry.Type(), "sequence row should not resolve to a Type")
	testutil.Equal(t, "IfEntry", ifEntry.SequenceTypeName(), "row sequence type name")
	testutil.Len(t, ifEntry.Index(), 1, "ifEntry index len")
	testutil.NotNil(t, ifEntry.Index()[0].Object, "ifEntry index object")
	testutil.Equal(t, "ifIndex", ifEntry.Index()[0].Object.Name(), "ifEntry index object name")
	testutil.Equal(t, model.IndexEncodingInteger, ifEntry.Index()[0].Encoding, "ifEntry index encoding")
}

func TestCreateResolvedCapabilities(t *testing.T) {
	defMod := &module.Module{
		Name: "CAP-MIB",
	}
	addDefs(defMod, []module.Definition{
		&module.AgentCapabilities{
			DefBase:        module.DefBase{Name: "testAgent"},
			ProductRelease: "1.0",
			Status:         types.StatusCurrent,
			Description:    "capability description",
			Reference:      "RFC2580",
			Supports: []module.SupportsModule{
				{
					ModuleName: "IF-MIB",
					Includes:   []string{"ifGeneralGroup"},
					Variations: []module.Variation{
						{
							Name:             "ifAdminStatus",
							Syntax:           &module.TypeSyntaxTypeRef{Name: "DisplayString"},
							Access:           func() *types.Access { v := types.AccessReadOnly; return &v }(),
							CreationRequires: []string{"ifIndex"},
							DefVal:           &module.DefValString{Value: "up"},
							Description:      "object variation",
						},
						{
							Name:        "linkDown",
							Access:      func() *types.Access { v := types.AccessReadOnly; return &v }(),
							Description: "notification variation",
						},
					},
				},
			},
			Oid: module.NewOidAssignment([]module.OidComponent{
				&module.OidComponentName{NameValue: "testRoot"},
				&module.OidComponentNumber{Value: 100},
			}, types.Span{}),
		},
	})
	supportsMod := &module.Module{Name: "IF-MIB"}

	ctx := newTestContextForModules(model.VerboseConfig(), defMod, supportsMod)

	displayType := model.NewType("DisplayString")
	model.SetTypeBase(displayType, model.BaseOctetString)
	ctx.registerModuleTypeSymbol(defMod, "DisplayString", displayType)

	root := ctx.mib.Root()
	capNode := registerNodeWithSymbol(ctx, defMod, root, "testAgent", 1, 3, 6, 1, 4, 100)
	model.SetNodeKind(capNode, model.KindCapability)

	objNode := registerNodeWithSymbol(ctx, supportsMod, root, "ifAdminStatus", 1, 3, 6, 1, 2, 1)
	model.SetNodeKind(objNode, model.KindColumn)
	notifNode := registerNodeWithSymbol(ctx, supportsMod, root, "linkDown", 1, 3, 6, 1, 6, 3)
	model.SetNodeKind(notifNode, model.KindNotification)

	createResolvedCapabilities(ctx)

	resolvedMod := ctx.moduleToResolved[defMod]
	cap := resolvedMod.Capability("testAgent")
	testutil.NotNil(t, cap, "capability")
	testutil.Equal(t, resolvedMod, cap.Module(), "capability module")
	testutil.Equal(t, capNode, cap.Node(), "capability node")
	testutil.Equal(t, "1.0", cap.ProductRelease(), "product release")
	testutil.Equal(t, "capability description", cap.Description(), "description")
	testutil.Equal(t, "RFC2580", cap.Reference(), "reference")
	testutil.Len(t, cap.OidRefs(), 1, "oid refs len")
	testutil.Equal(t, "testRoot", cap.OidRefs()[0].Name, "oid ref")
	testutil.Equal(t, cap, capNode.Capability(), "node capability")

	supports := cap.Supports()
	testutil.Len(t, supports, 1, "supports len")
	testutil.Equal(t, "IF-MIB", supports[0].ModuleName, "supports module")
	testutil.Len(t, supports[0].Includes, 1, "supports includes len")
	testutil.Len(t, supports[0].ObjectVariations, 1, "object variations len")
	testutil.Len(t, supports[0].NotificationVariations, 1, "notification variations len")

	objVar := supports[0].ObjectVariations[0]
	testutil.Equal(t, "ifAdminStatus", objVar.Object, "object variation name")
	testutil.NotNil(t, objVar.Syntax, "object variation syntax")
	testutil.Equal(t, displayType, objVar.Syntax.Type, "object variation syntax type")
	testutil.NotNil(t, objVar.Access, "object variation access")
	testutil.Equal(t, model.AccessReadOnly, *objVar.Access, "object variation access")
	testutil.Len(t, objVar.CreationRequires, 1, "object variation creation-requires")
	testutil.Equal(t, "ifIndex", objVar.CreationRequires[0], "object variation creation-requires")
	testutil.Equal(t, model.DefValKindString, objVar.DefVal.Kind(), "object variation defval kind")

	notifVar := supports[0].NotificationVariations[0]
	testutil.Equal(t, "linkDown", notifVar.Notification, "notification variation name")
	testutil.NotNil(t, notifVar.Access, "notification variation access")
	testutil.Equal(t, model.AccessReadOnly, *notifVar.Access, "notification variation access")

	hasDiag(t, ctx.Diagnostics(), types.DiagVariationAccessNotifOnly)
}

func TestConvertDefValOidValue(t *testing.T) {
	ctx := newTestContext()
	mod := &module.Module{Name: "TEST-MIB"}
	smiMod := &module.Module{Name: "SNMPv2-SMI"}
	ctx.moduleIndex["SNMPv2-SMI"] = []*module.Module{smiMod}

	root := ctx.mib.Root()
	child := model.GetOrCreateChild(root, 1)
	child2 := model.GetOrCreateChild(child, 3)
	model.SetNodeName(child2, "enterprises")
	ctx.registerModuleNodeSymbol(mod, "enterprises", child2)
	ctx.registerModuleNodeSymbol(smiMod, "enterprises", child2)

	t.Run("resolves name with trailing numeric arcs", func(t *testing.T) {
		dv := convertDefVal(ctx, &module.DefValOidValue{
			Components: []module.OidComponent{
				&module.OidComponentName{NameValue: "enterprises"},
				&module.OidComponentNumber{Value: 42},
				&module.OidComponentNumber{Value: 1},
			},
		}, mod, nil, types.Span{})
		testutil.NotNil(t, dv, "expected non-nil")
		testutil.Equal(t, model.DefValKindOID, dv.Kind(), "kind")
		oid, ok := model.DefValAs[model.OID](*dv)
		testutil.True(t, ok, "expected OID value")
		want := model.OID{1, 3, 42, 1}
		testutil.Equal(t, want.String(), oid.String(), "oid")
	})

	t.Run("resolves single name component", func(t *testing.T) {
		dv := convertDefVal(ctx, &module.DefValOidValue{
			Components: []module.OidComponent{
				&module.OidComponentName{NameValue: "enterprises"},
			},
		}, mod, nil, types.Span{})
		testutil.NotNil(t, dv, "expected non-nil")
		oid, ok := model.DefValAs[model.OID](*dv)
		testutil.True(t, ok, "expected OID value")
		want := model.OID{1, 3}
		testutil.Equal(t, want.String(), oid.String(), "oid")
	})

	t.Run("resolves named number with trailing arcs", func(t *testing.T) {
		dv := convertDefVal(ctx, &module.DefValOidValue{
			Components: []module.OidComponent{
				&module.OidComponentNamedNumber{NameValue: "enterprises", NumberValue: 1},
				&module.OidComponentNumber{Value: 5},
			},
		}, mod, nil, types.Span{})
		testutil.NotNil(t, dv, "expected non-nil")
		oid, ok := model.DefValAs[model.OID](*dv)
		testutil.True(t, ok, "expected OID value")
		want := model.OID{1, 3, 5}
		testutil.Equal(t, want.String(), oid.String(), "oid")
	})

	t.Run("resolves qualified name", func(t *testing.T) {
		dv := convertDefVal(ctx, &module.DefValOidValue{
			Components: []module.OidComponent{
				&module.OidComponentQualifiedName{ModuleValue: "SNMPv2-SMI", NameValue: "enterprises"},
			},
		}, mod, nil, types.Span{})
		testutil.NotNil(t, dv, "expected non-nil")
		testutil.Equal(t, model.DefValKindOID, dv.Kind(), "kind")
		oid, ok := model.DefValAs[model.OID](*dv)
		testutil.True(t, ok, "expected OID value")
		want := model.OID{1, 3}
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
		}, mod, nil, types.Span{})
		testutil.NotNil(t, dv, "expected non-nil")
		testutil.Equal(t, model.DefValKindOID, dv.Kind(), "kind")
		oid, ok := model.DefValAs[model.OID](*dv)
		testutil.True(t, ok, "expected OID value")
		want := model.OID{1, 3}
		testutil.Equal(t, want.String(), oid.String(), "oid")
	})

	t.Run("returns nil for unresolved name", func(t *testing.T) {
		dv := convertDefVal(ctx, &module.DefValOidValue{
			Components: []module.OidComponent{
				&module.OidComponentName{NameValue: "nonexistent"},
			},
		}, mod, nil, types.Span{})
		testutil.Nil(t, dv, "expected nil for unresolved name, got")
	})

	t.Run("returns nil for empty components", func(t *testing.T) {
		dv := convertDefVal(ctx, &module.DefValOidValue{
			Components: nil,
		}, mod, nil, types.Span{})
		testutil.Nil(t, dv, "expected nil for empty components, got")
	})

	t.Run("returns nil when first component is numeric only", func(t *testing.T) {
		dv := convertDefVal(ctx, &module.DefValOidValue{
			Components: []module.OidComponent{
				&module.OidComponentNumber{Value: 1},
			},
		}, mod, nil, types.Span{})
		testutil.Nil(t, dv, "expected nil when first component is numeric, got")
	})
}

func TestInferNodeKinds(t *testing.T) {
	t.Run("classifies table row scalar", func(t *testing.T) {
		ctx := newTestContext()
		mod := &module.Module{Name: "TEST-MIB"}
		ctx.modules = append(ctx.modules, mod)
		ctx.moduleToResolved[mod] = model.NewModule(mod.Name)

		root := ctx.mib.Root()
		tableNode := buildOIDPath(root, 1, 1)
		model.SetNodeName(tableNode, "myTable")
		ctx.registerModuleNodeSymbol(mod, "myTable", tableNode)

		rowNode := buildOIDPath(root, 1, 1, 1)
		model.SetNodeName(rowNode, "myEntry")
		ctx.registerModuleNodeSymbol(mod, "myEntry", rowNode)

		scalarNode := buildOIDPath(root, 1, 2)
		model.SetNodeName(scalarNode, "myScalar")
		ctx.registerModuleNodeSymbol(mod, "myScalar", scalarNode)

		objRefs := []objectTypeRef{
			{mod: mod, obj: &module.ObjectType{
				DefBase: module.DefBase{Name: "myTable"},
				Syntax:  &module.TypeSyntaxSequenceOf{EntryType: "MyEntry"},
			}},
			{mod: mod, obj: &module.ObjectType{
				DefBase: module.DefBase{Name: "myEntry"},
				Syntax:  &module.TypeSyntaxTypeRef{Name: "MyEntry"},
				Index:   []module.IndexItem{{Object: "myIndex"}},
			}},
			{mod: mod, obj: &module.ObjectType{
				DefBase: module.DefBase{Name: "myScalar"},
				Syntax:  &module.TypeSyntaxTypeRef{Name: "Integer32"},
			}},
		}

		inferNodeKinds(ctx, objRefs)

		testutil.Equal(t, model.KindTable, tableNode.Kind(), "table kind")
		testutil.Equal(t, model.KindRow, rowNode.Kind(), "row kind")
		testutil.Equal(t, model.KindScalar, scalarNode.Kind(), "scalar kind")
	})

	t.Run("augments classifies as row", func(t *testing.T) {
		ctx := newTestContext()
		mod := &module.Module{Name: "TEST-MIB"}
		ctx.modules = append(ctx.modules, mod)
		ctx.moduleToResolved[mod] = model.NewModule(mod.Name)

		root := ctx.mib.Root()
		augNode := buildOIDPath(root, 1, 1)
		model.SetNodeName(augNode, "myAugEntry")
		ctx.registerModuleNodeSymbol(mod, "myAugEntry", augNode)

		objRefs := []objectTypeRef{
			{mod: mod, obj: &module.ObjectType{
				DefBase:  module.DefBase{Name: "myAugEntry"},
				Syntax:   &module.TypeSyntaxTypeRef{Name: "MyAugEntry"},
				Augments: "otherEntry",
			}},
		}

		inferNodeKinds(ctx, objRefs)

		testutil.Equal(t, model.KindRow, augNode.Kind(), "augments node kind")
	})

	t.Run("reclassifies scalar children of rows as columns", func(t *testing.T) {
		ctx := newTestContext()
		mod := &module.Module{Name: "TEST-MIB"}
		ctx.modules = append(ctx.modules, mod)
		ctx.moduleToResolved[mod] = model.NewModule(mod.Name)

		root := ctx.mib.Root()
		rowNode := buildOIDPath(root, 1, 1, 1)
		model.SetNodeName(rowNode, "myEntry")
		ctx.registerModuleNodeSymbol(mod, "myEntry", rowNode)

		col1 := buildOIDPath(root, 1, 1, 1, 1)
		model.SetNodeName(col1, "myCol1")
		ctx.registerModuleNodeSymbol(mod, "myCol1", col1)

		col2 := buildOIDPath(root, 1, 1, 1, 2)
		model.SetNodeName(col2, "myCol2")
		ctx.registerModuleNodeSymbol(mod, "myCol2", col2)

		objRefs := []objectTypeRef{
			{mod: mod, obj: &module.ObjectType{
				DefBase: module.DefBase{Name: "myEntry"},
				Syntax:  &module.TypeSyntaxTypeRef{Name: "MyEntry"},
				Index:   []module.IndexItem{{Object: "myCol1"}},
			}},
			{mod: mod, obj: &module.ObjectType{
				DefBase: module.DefBase{Name: "myCol1"},
				Syntax:  &module.TypeSyntaxTypeRef{Name: "Integer32"},
			}},
			{mod: mod, obj: &module.ObjectType{
				DefBase: module.DefBase{Name: "myCol2"},
				Syntax:  &module.TypeSyntaxTypeRef{Name: "DisplayString"},
			}},
		}

		inferNodeKinds(ctx, objRefs)

		testutil.Equal(t, model.KindRow, rowNode.Kind(), "row kind")
		testutil.Equal(t, model.KindColumn, col1.Kind(), "col1 kind")
		testutil.Equal(t, model.KindColumn, col2.Kind(), "col2 kind")
	})

	t.Run("skips objects with unresolved nodes", func(_ *testing.T) {
		ctx := newTestContext()
		mod := &module.Module{Name: "TEST-MIB"}
		ctx.modules = append(ctx.modules, mod)
		ctx.moduleToResolved[mod] = model.NewModule(mod.Name)

		// Don't register any node - the lookup should fail silently.
		objRefs := []objectTypeRef{
			{mod: mod, obj: &module.ObjectType{
				DefBase: module.DefBase{Name: "missing"},
				Syntax:  &module.TypeSyntaxTypeRef{Name: "Integer32"},
			}},
		}

		// Should not panic.
		inferNodeKinds(ctx, objRefs)
	})

	t.Run("non-scalar children of rows not reclassified", func(t *testing.T) {
		ctx := newTestContext()
		mod := &module.Module{Name: "TEST-MIB"}
		ctx.modules = append(ctx.modules, mod)
		ctx.moduleToResolved[mod] = model.NewModule(mod.Name)

		root := ctx.mib.Root()
		rowNode := buildOIDPath(root, 1, 1, 1)
		model.SetNodeName(rowNode, "myEntry")
		ctx.registerModuleNodeSymbol(mod, "myEntry", rowNode)

		// Child node that is not an OBJECT-TYPE (no objRef), so it stays model.KindUnknown.
		child := buildOIDPath(root, 1, 1, 1, 1)
		model.SetNodeName(child, "childNode")

		objRefs := []objectTypeRef{
			{mod: mod, obj: &module.ObjectType{
				DefBase: module.DefBase{Name: "myEntry"},
				Syntax:  &module.TypeSyntaxTypeRef{Name: "MyEntry"},
				Index:   []module.IndexItem{{Object: "someIndex"}},
			}},
		}

		inferNodeKinds(ctx, objRefs)

		testutil.Equal(t, model.KindRow, rowNode.Kind(), "row kind")
		// child was never classified as scalar (it starts as model.KindInternal from
		// getOrCreateChild), so it should not become column.
		testutil.Equal(t, model.KindInternal, child.Kind(), "unclassified child kind")
	})
}

func TestResolveTableSemantics(t *testing.T) {
	t.Run("resolved indexes produce no diagnostic", func(t *testing.T) {
		ctx := newTestContext()
		mod := &module.Module{Name: "TEST-MIB"}
		ctx.modules = append(ctx.modules, mod)

		root := ctx.mib.Root()
		idxNode := buildOIDPath(root, 1, 1)
		model.SetNodeName(idxNode, "myIndex")
		ctx.registerModuleNodeSymbol(mod, "myIndex", idxNode)

		objRefs := []objectTypeRef{
			{mod: mod, obj: &module.ObjectType{
				DefBase: module.DefBase{Name: "myEntry"},
				Syntax:  &module.TypeSyntaxTypeRef{Name: "MyEntry"},
				Index:   []module.IndexItem{{Object: "myIndex"}},
			}},
		}

		resolveTableSemantics(ctx, objRefs)

		testutil.Len(t, ctx.Diagnostics(), 0, "expected no diagnostics for resolved index")
	})

	t.Run("unresolved index emits diagnostic", func(t *testing.T) {
		ctx := newTestContext()
		mod := &module.Module{Name: "TEST-MIB"}
		ctx.modules = append(ctx.modules, mod)

		objRefs := []objectTypeRef{
			{mod: mod, obj: &module.ObjectType{
				DefBase: module.DefBase{Name: "myEntry"},
				Syntax:  &module.TypeSyntaxTypeRef{Name: "MyEntry"},
				Index:   []module.IndexItem{{Object: "missingIndex"}},
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
		ctx.modules = append(ctx.modules, mod)

		// INTEGER is a bare type, so it should not produce a diagnostic
		// even though no node exists for it.
		objRefs := []objectTypeRef{
			{mod: mod, obj: &module.ObjectType{
				DefBase: module.DefBase{Name: "myEntry"},
				Syntax:  &module.TypeSyntaxTypeRef{Name: "MyEntry"},
				Index:   []module.IndexItem{{Object: "INTEGER"}},
			}},
		}

		resolveTableSemantics(ctx, objRefs)

		testutil.Len(t, ctx.Diagnostics(), 0, "expected no diagnostic for bare type index")
	})

	t.Run("resolved augments produce no diagnostic", func(t *testing.T) {
		ctx := newTestContext()
		mod := &module.Module{Name: "TEST-MIB"}
		ctx.modules = append(ctx.modules, mod)

		root := ctx.mib.Root()
		augTarget := buildOIDPath(root, 1, 1)
		model.SetNodeName(augTarget, "otherEntry")
		ctx.registerModuleNodeSymbol(mod, "otherEntry", augTarget)

		objRefs := []objectTypeRef{
			{mod: mod, obj: &module.ObjectType{
				DefBase:  module.DefBase{Name: "myAugEntry"},
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
		ctx.modules = append(ctx.modules, mod)

		objRefs := []objectTypeRef{
			{mod: mod, obj: &module.ObjectType{
				DefBase:  module.DefBase{Name: "myAugEntry"},
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
		ctx.modules = append(ctx.modules, mod)

		objRefs := []objectTypeRef{
			{mod: mod, obj: &module.ObjectType{
				DefBase: module.DefBase{Name: "myScalar"},
				Syntax:  &module.TypeSyntaxTypeRef{Name: "Integer32"},
			}},
		}

		resolveTableSemantics(ctx, objRefs)

		testutil.Len(t, ctx.Diagnostics(), 0, "expected no diagnostics for scalar")
	})

	t.Run("mixed resolved and unresolved indexes", func(t *testing.T) {
		ctx := newTestContext()
		mod := &module.Module{Name: "TEST-MIB"}
		ctx.modules = append(ctx.modules, mod)

		root := ctx.mib.Root()
		idx1 := buildOIDPath(root, 1, 1)
		model.SetNodeName(idx1, "goodIndex")
		ctx.registerModuleNodeSymbol(mod, "goodIndex", idx1)

		objRefs := []objectTypeRef{
			{mod: mod, obj: &module.ObjectType{
				DefBase: module.DefBase{Name: "myEntry"},
				Syntax:  &module.TypeSyntaxTypeRef{Name: "MyEntry"},
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
		ctx.modules = append(ctx.modules, mod)
		resolvedMod := model.NewModule(mod.Name)
		ctx.moduleToResolved[mod] = resolvedMod

		root := ctx.mib.Root()

		// Create row node and object.
		rowNode := buildOIDPath(root, 1, 1, 1)
		model.SetNodeName(rowNode, "myEntry")
		ctx.registerModuleNodeSymbol(mod, "myEntry", rowNode)
		rowObj := model.NewObject("myEntry")
		model.SetObjectNode(rowObj, rowNode)
		model.AddModuleObject(resolvedMod, rowObj)
		model.SetNodeObject(rowNode, rowObj)

		// Create index node and object.
		idxNode := buildOIDPath(root, 1, 1, 1, 1)
		model.SetNodeName(idxNode, "myIndex")
		ctx.registerModuleNodeSymbol(mod, "myIndex", idxNode)
		idxObj := model.NewObject("myIndex")
		model.SetObjectNode(idxObj, idxNode)
		model.SetNodeObject(idxNode, idxObj)
		model.AddModuleObject(resolvedMod, idxObj)

		objRefs := []objectTypeRef{
			{mod: mod, obj: &module.ObjectType{
				DefBase: module.DefBase{Name: "myEntry"},
				Syntax:  &module.TypeSyntaxTypeRef{Name: "MyEntry"},
				Index:   []module.IndexItem{{Object: "myIndex", Implied: true}},
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
		ctx.modules = append(ctx.modules, mod)
		resolvedMod := model.NewModule(mod.Name)
		ctx.moduleToResolved[mod] = resolvedMod

		root := ctx.mib.Root()

		// Create the augmenting row.
		augNode := buildOIDPath(root, 1, 2, 1)
		model.SetNodeName(augNode, "myAugEntry")
		ctx.registerModuleNodeSymbol(mod, "myAugEntry", augNode)
		augObj := model.NewObject("myAugEntry")
		model.SetObjectNode(augObj, augNode)
		model.AddModuleObject(resolvedMod, augObj)
		model.SetNodeObject(augNode, augObj)

		// Create the target row being augmented.
		targetNode := buildOIDPath(root, 1, 1, 1)
		model.SetNodeName(targetNode, "targetEntry")
		ctx.registerModuleNodeSymbol(mod, "targetEntry", targetNode)
		targetObj := model.NewObject("targetEntry")
		model.SetObjectNode(targetObj, targetNode)
		model.SetNodeObject(targetNode, targetObj)
		model.AddModuleObject(resolvedMod, targetObj)

		objRefs := []objectTypeRef{
			{mod: mod, obj: &module.ObjectType{
				DefBase:  module.DefBase{Name: "myAugEntry"},
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
		ctx.modules = append(ctx.modules, mod)
		resolvedMod := model.NewModule(mod.Name)
		ctx.moduleToResolved[mod] = resolvedMod

		root := ctx.mib.Root()

		rowNode := buildOIDPath(root, 1, 1, 1)
		model.SetNodeName(rowNode, "myEntry")
		ctx.registerModuleNodeSymbol(mod, "myEntry", rowNode)
		rowObj := model.NewObject("myEntry")
		model.SetObjectNode(rowObj, rowNode)
		model.AddModuleObject(resolvedMod, rowObj)
		model.SetNodeObject(rowNode, rowObj)

		// Index node exists but has no Object attached.
		idxNode := buildOIDPath(root, 1, 1, 1, 1)
		model.SetNodeName(idxNode, "bareNode")
		ctx.registerModuleNodeSymbol(mod, "bareNode", idxNode)

		objRefs := []objectTypeRef{
			{mod: mod, obj: &module.ObjectType{
				DefBase: module.DefBase{Name: "myEntry"},
				Syntax:  &module.TypeSyntaxTypeRef{Name: "MyEntry"},
				Index:   []module.IndexItem{{Object: "bareNode"}},
			}},
		}

		linkObjectIndexes(ctx, objRefs)

		testutil.Len(t, rowObj.Index(), 0, "index entries should be empty when index node has no object")

		hasDiag(t, ctx.Diagnostics(), types.DiagIndexNotObject)
	})

	t.Run("augments node without object emits diagnostic", func(t *testing.T) {
		ctx := newTestContext()
		mod := &module.Module{Name: "TEST-MIB"}
		ctx.modules = append(ctx.modules, mod)
		resolvedMod := model.NewModule(mod.Name)
		ctx.moduleToResolved[mod] = resolvedMod

		root := ctx.mib.Root()

		// Create the augmenting row.
		augNode := buildOIDPath(root, 1, 2, 1)
		model.SetNodeName(augNode, "myAugEntry")
		ctx.registerModuleNodeSymbol(mod, "myAugEntry", augNode)
		augObj := model.NewObject("myAugEntry")
		model.SetObjectNode(augObj, augNode)
		model.AddModuleObject(resolvedMod, augObj)
		model.SetNodeObject(augNode, augObj)

		// Target node exists but has no Object attached.
		targetNode := buildOIDPath(root, 1, 1, 1)
		model.SetNodeName(targetNode, "targetEntry")
		ctx.registerModuleNodeSymbol(mod, "targetEntry", targetNode)

		objRefs := []objectTypeRef{
			{mod: mod, obj: &module.ObjectType{
				DefBase:  module.DefBase{Name: "myAugEntry"},
				Syntax:   &module.TypeSyntaxTypeRef{Name: "MyAugEntry"},
				Augments: "targetEntry",
			}},
		}

		linkObjectIndexes(ctx, objRefs)

		testutil.Nil(t, augObj.Augments(), "augments should be nil when target has no object")

		hasDiag(t, ctx.Diagnostics(), types.DiagAugmentsNotObject)
	})

	t.Run("nested augments emits diagnostic", func(t *testing.T) {
		ctx := newTestContext()
		mod := &module.Module{Name: "TEST-MIB"}
		ctx.modules = append(ctx.modules, mod)
		resolvedMod := model.NewModule(mod.Name)
		ctx.moduleToResolved[mod] = resolvedMod

		root := ctx.mib.Root()

		// Base row with its own INDEX.
		baseNode := buildOIDPath(root, 1, 1, 1)
		model.SetNodeName(baseNode, "baseEntry")
		ctx.registerModuleNodeSymbol(mod, "baseEntry", baseNode)
		baseObj := model.NewObject("baseEntry")
		model.SetObjectNode(baseObj, baseNode)
		model.SetNodeObject(baseNode, baseObj)
		model.AddModuleObject(resolvedMod, baseObj)

		// Middle row that augments the base.
		midNode := buildOIDPath(root, 1, 2, 1)
		model.SetNodeName(midNode, "midEntry")
		ctx.registerModuleNodeSymbol(mod, "midEntry", midNode)
		midObj := model.NewObject("midEntry")
		model.SetObjectNode(midObj, midNode)
		model.SetNodeObject(midNode, midObj)
		model.AddModuleObject(resolvedMod, midObj)

		// Leaf row that augments the middle (nested).
		leafNode := buildOIDPath(root, 1, 3, 1)
		model.SetNodeName(leafNode, "leafEntry")
		ctx.registerModuleNodeSymbol(mod, "leafEntry", leafNode)
		leafObj := model.NewObject("leafEntry")
		model.SetObjectNode(leafObj, leafNode)
		model.SetNodeObject(leafNode, leafObj)
		model.AddModuleObject(resolvedMod, leafObj)

		objRefs := []objectTypeRef{
			{mod: mod, obj: &module.ObjectType{
				DefBase:  module.DefBase{Name: "midEntry"},
				Syntax:   &module.TypeSyntaxTypeRef{Name: "MidEntry"},
				Augments: "baseEntry",
			}},
			{mod: mod, obj: &module.ObjectType{
				DefBase:  module.DefBase{Name: "leafEntry"},
				Syntax:   &module.TypeSyntaxTypeRef{Name: "LeafEntry"},
				Augments: "midEntry",
			}},
		}

		// Link first, then check nesting.
		linkObjectIndexes(ctx, objRefs)
		checkAugmentsNesting(ctx, objRefs)

		// midEntry augments baseEntry (base row) - no diagnostic.
		// leafEntry augments midEntry (itself an augmentation) - should emit diagnostic.
		d := hasDiag(t, ctx.Diagnostics(), types.DiagAugmentNested)
		testutil.Contains(t, d.Message, "leafEntry", "diagnostic should mention augmenting row")
		testutil.Contains(t, d.Message, "midEntry", "diagnostic should mention target row")
	})

	t.Run("augments base row emits no nesting diagnostic", func(t *testing.T) {
		ctx := newTestContext()
		mod := &module.Module{Name: "TEST-MIB"}
		ctx.modules = append(ctx.modules, mod)
		resolvedMod := model.NewModule(mod.Name)
		ctx.moduleToResolved[mod] = resolvedMod

		root := ctx.mib.Root()

		// Base row with its own INDEX.
		baseNode := buildOIDPath(root, 1, 1, 1)
		model.SetNodeName(baseNode, "baseEntry")
		ctx.registerModuleNodeSymbol(mod, "baseEntry", baseNode)
		baseObj := model.NewObject("baseEntry")
		model.SetObjectNode(baseObj, baseNode)
		model.SetNodeObject(baseNode, baseObj)
		model.AddModuleObject(resolvedMod, baseObj)

		// Augmenting row that augments the base row.
		augNode := buildOIDPath(root, 1, 2, 1)
		model.SetNodeName(augNode, "augEntry")
		ctx.registerModuleNodeSymbol(mod, "augEntry", augNode)
		augObj := model.NewObject("augEntry")
		model.SetObjectNode(augObj, augNode)
		model.SetNodeObject(augNode, augObj)
		model.AddModuleObject(resolvedMod, augObj)

		objRefs := []objectTypeRef{
			{mod: mod, obj: &module.ObjectType{
				DefBase:  module.DefBase{Name: "augEntry"},
				Syntax:   &module.TypeSyntaxTypeRef{Name: "AugEntry"},
				Augments: "baseEntry",
			}},
		}

		linkObjectIndexes(ctx, objRefs)
		checkAugmentsNesting(ctx, objRefs)

		noDiag(t, ctx.Diagnostics(), types.DiagAugmentNested)
	})

	t.Run("bare type index creates TypeName entry", func(t *testing.T) {
		ctx := newTestContext()
		mod := &module.Module{Name: "TEST-MIB"}
		ctx.modules = append(ctx.modules, mod)
		resolvedMod := model.NewModule(mod.Name)
		ctx.moduleToResolved[mod] = resolvedMod

		root := ctx.mib.Root()

		rowNode := buildOIDPath(root, 1, 1, 1)
		model.SetNodeName(rowNode, "myEntry")
		ctx.registerModuleNodeSymbol(mod, "myEntry", rowNode)
		rowObj := model.NewObject("myEntry")
		model.SetObjectNode(rowObj, rowNode)
		model.AddModuleObject(resolvedMod, rowObj)
		model.SetNodeObject(rowNode, rowObj)

		objRefs := []objectTypeRef{
			{mod: mod, obj: &module.ObjectType{
				DefBase: module.DefBase{Name: "myEntry"},
				Syntax:  &module.TypeSyntaxTypeRef{Name: "MyEntry"},
				Index:   []module.IndexItem{{Object: "INTEGER"}},
			}},
		}

		linkObjectIndexes(ctx, objRefs)

		idx := rowObj.Index()
		testutil.Len(t, idx, 1, "index entries")
		testutil.Nil(t, idx[0].Object, "bare type index Object should be nil")
		testutil.Equal(t, "INTEGER", idx[0].TypeName, "bare type index TypeName")
	})

	t.Run("permissive global fallback resolves index", func(t *testing.T) {
		modA := &module.Module{Name: "A-MIB"}
		modB := &module.Module{Name: "B-MIB"}

		ctx := newResolverContext(nil, model.ResolverPermissive, model.DefaultConfig())
		ctx.modules = []*module.Module{modA, modB}

		resolvedModA := model.NewModule(modA.Name)
		ctx.moduleToResolved[modA] = resolvedModA
		resolvedModB := model.NewModule(modB.Name)
		ctx.moduleToResolved[modB] = resolvedModB

		root := ctx.mib.Root()

		// Row node and object in module A.
		rowNode := buildOIDPath(root, 1, 1, 1)
		model.SetNodeName(rowNode, "myEntry")
		ctx.registerModuleNodeSymbol(modA, "myEntry", rowNode)
		rowObj := model.NewObject("myEntry")
		model.SetObjectNode(rowObj, rowNode)
		model.AddModuleObject(resolvedModA, rowObj)
		model.SetNodeObject(rowNode, rowObj)

		// Index node and object in module B, NOT imported into A.
		idxNode := buildOIDPath(root, 2, 1, 1, 1)
		model.SetNodeName(idxNode, "foreignIndex")
		ctx.registerModuleNodeSymbol(modB, "foreignIndex", idxNode)
		idxObj := model.NewObject("foreignIndex")
		model.SetObjectNode(idxObj, idxNode)
		model.AddModuleObject(resolvedModB, idxObj)
		model.SetNodeObject(idxNode, idxObj)

		objRefs := []objectTypeRef{
			{mod: modA, obj: &module.ObjectType{
				DefBase: module.DefBase{Name: "myEntry"},
				Syntax:  &module.TypeSyntaxTypeRef{Name: "MyEntry"},
				Index:   []module.IndexItem{{Object: "foreignIndex"}},
			}},
		}

		linkObjectIndexes(ctx, objRefs)

		idx := rowObj.Index()
		testutil.Len(t, idx, 1, "permissive mode should resolve index via global fallback")
		testutil.Equal(t, idxObj, idx[0].Object, "index object from global fallback")
	})

	t.Run("permissive global fallback resolves augments", func(t *testing.T) {
		modA := &module.Module{Name: "A-MIB"}
		modB := &module.Module{Name: "B-MIB"}

		ctx := newResolverContext(nil, model.ResolverPermissive, model.DefaultConfig())
		ctx.modules = []*module.Module{modA, modB}

		resolvedModA := model.NewModule(modA.Name)
		ctx.moduleToResolved[modA] = resolvedModA
		resolvedModB := model.NewModule(modB.Name)
		ctx.moduleToResolved[modB] = resolvedModB

		root := ctx.mib.Root()

		// Augmenting row in module A.
		augNode := buildOIDPath(root, 1, 2, 1)
		model.SetNodeName(augNode, "myAugEntry")
		ctx.registerModuleNodeSymbol(modA, "myAugEntry", augNode)
		augObj := model.NewObject("myAugEntry")
		model.SetObjectNode(augObj, augNode)
		model.AddModuleObject(resolvedModA, augObj)
		model.SetNodeObject(augNode, augObj)

		// Target row in module B, NOT imported into A.
		targetNode := buildOIDPath(root, 2, 1, 1)
		model.SetNodeName(targetNode, "foreignEntry")
		ctx.registerModuleNodeSymbol(modB, "foreignEntry", targetNode)
		targetObj := model.NewObject("foreignEntry")
		model.SetObjectNode(targetObj, targetNode)
		model.AddModuleObject(resolvedModB, targetObj)
		model.SetNodeObject(targetNode, targetObj)

		objRefs := []objectTypeRef{
			{mod: modA, obj: &module.ObjectType{
				DefBase:  module.DefBase{Name: "myAugEntry"},
				Syntax:   &module.TypeSyntaxTypeRef{Name: "MyAugEntry"},
				Augments: "foreignEntry",
			}},
		}

		linkObjectIndexes(ctx, objRefs)

		testutil.NotNil(t, augObj.Augments(), "permissive mode should resolve augments via global fallback")
		testutil.Equal(t, targetObj, augObj.Augments(), "augments target from global fallback")

		augBy := targetObj.AugmentedBy()
		testutil.Len(t, augBy, 1, "augmented by via global fallback")
		testutil.Equal(t, augObj, augBy[0], "augmented by entry from global fallback")
	})

	t.Run("strict mode no global fallback for index", func(t *testing.T) {
		modA := &module.Module{Name: "A-MIB"}
		modB := &module.Module{Name: "B-MIB"}

		ctx := newResolverContext(nil, model.ResolverStrict, model.VerboseConfig())
		ctx.modules = []*module.Module{modA, modB}

		resolvedModA := model.NewModule(modA.Name)
		ctx.moduleToResolved[modA] = resolvedModA
		resolvedModB := model.NewModule(modB.Name)
		ctx.moduleToResolved[modB] = resolvedModB

		root := ctx.mib.Root()

		rowNode := buildOIDPath(root, 1, 1, 1)
		model.SetNodeName(rowNode, "myEntry")
		ctx.registerModuleNodeSymbol(modA, "myEntry", rowNode)
		rowObj := model.NewObject("myEntry")
		model.SetObjectNode(rowObj, rowNode)
		model.AddModuleObject(resolvedModA, rowObj)
		model.SetNodeObject(rowNode, rowObj)

		idxNode := buildOIDPath(root, 2, 1, 1, 1)
		model.SetNodeName(idxNode, "foreignIndex")
		ctx.registerModuleNodeSymbol(modB, "foreignIndex", idxNode)
		idxObj := model.NewObject("foreignIndex")
		model.SetObjectNode(idxObj, idxNode)
		model.AddModuleObject(resolvedModB, idxObj)
		model.SetNodeObject(idxNode, idxObj)

		objRefs := []objectTypeRef{
			{mod: modA, obj: &module.ObjectType{
				DefBase: module.DefBase{Name: "myEntry"},
				Syntax:  &module.TypeSyntaxTypeRef{Name: "MyEntry"},
				Index:   []module.IndexItem{{Object: "foreignIndex"}},
			}},
		}

		linkObjectIndexes(ctx, objRefs)

		testutil.Len(t, rowObj.Index(), 0, "strict mode should not resolve index via global fallback")
	})

	t.Run("strict mode no global fallback for augments", func(t *testing.T) {
		modA := &module.Module{Name: "A-MIB"}
		modB := &module.Module{Name: "B-MIB"}

		ctx := newResolverContext(nil, model.ResolverStrict, model.VerboseConfig())
		ctx.modules = []*module.Module{modA, modB}

		resolvedModA := model.NewModule(modA.Name)
		ctx.moduleToResolved[modA] = resolvedModA
		resolvedModB := model.NewModule(modB.Name)
		ctx.moduleToResolved[modB] = resolvedModB

		root := ctx.mib.Root()

		augNode := buildOIDPath(root, 1, 2, 1)
		model.SetNodeName(augNode, "myAugEntry")
		ctx.registerModuleNodeSymbol(modA, "myAugEntry", augNode)
		augObj := model.NewObject("myAugEntry")
		model.SetObjectNode(augObj, augNode)
		model.AddModuleObject(resolvedModA, augObj)
		model.SetNodeObject(augNode, augObj)

		targetNode := buildOIDPath(root, 2, 1, 1)
		model.SetNodeName(targetNode, "foreignEntry")
		ctx.registerModuleNodeSymbol(modB, "foreignEntry", targetNode)
		targetObj := model.NewObject("foreignEntry")
		model.SetObjectNode(targetObj, targetNode)
		model.AddModuleObject(resolvedModB, targetObj)
		model.SetNodeObject(targetNode, targetObj)

		objRefs := []objectTypeRef{
			{mod: modA, obj: &module.ObjectType{
				DefBase:  module.DefBase{Name: "myAugEntry"},
				Syntax:   &module.TypeSyntaxTypeRef{Name: "MyAugEntry"},
				Augments: "foreignEntry",
			}},
		}

		linkObjectIndexes(ctx, objRefs)

		testutil.Nil(t, augObj.Augments(), "strict mode should not resolve augments via global fallback")
	})

	t.Run("nil resolved module skipped", func(_ *testing.T) {
		ctx := newTestContext()
		mod := &module.Module{Name: "TEST-MIB"}
		ctx.modules = append(ctx.modules, mod)
		// Don't set ModuleToResolved - should skip without panic.

		objRefs := []objectTypeRef{
			{mod: mod, obj: &module.ObjectType{
				DefBase: module.DefBase{Name: "myEntry"},
				Syntax:  &module.TypeSyntaxTypeRef{Name: "MyEntry"},
				Index:   []module.IndexItem{{Object: "myIndex"}},
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

		ctx := newResolverContext(nil, model.ResolverPermissive, model.DefaultConfig())
		ctx.modules = []*module.Module{modA, modB}
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

		ctx := newResolverContext(nil, model.ResolverStrict, model.VerboseConfig())
		ctx.modules = []*module.Module{modA, modB}
		ctx.registerModuleNodeSymbol(modB, "member1", node)
		// No import from A to B.

		_, ok := lookupMemberNode(ctx, modA, "member1")
		testutil.False(t, ok, "expected strict mode to not find member1 without import")
	})

	t.Run("not found in any mode", func(t *testing.T) {
		modA := &module.Module{Name: "A"}
		ctx := newResolverContext(nil, model.ResolverNormal, model.DefaultConfig())
		ctx.modules = []*module.Module{modA}

		_, ok := lookupMemberNode(ctx, modA, "nonexistent")
		testutil.False(t, ok, "expected false for nonexistent member")
	})
}

// groupTestDef creates a module definition for either an ObjectGroup or
// NotificationGroup depending on isNotifGrp, with the given name and members.
func groupTestDef(name string, members []module.NameRef, isNotifGrp bool) module.Definition {
	if isNotifGrp {
		return &module.NotificationGroup{
			DefBase:       module.DefBase{Name: name},
			Notifications: members,
		}
	}
	return &module.ObjectGroup{
		DefBase: module.DefBase{Name: name},
		Objects: members,
	}
}

func TestCreateResolvedGroups(t *testing.T) {
	for _, isNotifGrp := range []bool{false, true} {
		label := "object-group"
		if isNotifGrp {
			label = "notification-group"
		}

		t.Run(label+"/creates group with members", func(t *testing.T) {
			def := groupTestDef("testGroup", nameRefs("m1", "m2"), isNotifGrp)
			mod := &module.Module{
				Name: "TEST-MIB",
			}
			addDefs(mod, []module.Definition{def})

			ctx := newTestContextForModules(model.DefaultConfig(), mod)
			root := ctx.mib.Root()

			grpNode := registerNodeWithSymbol(ctx, mod, root, "testGroup", 1, 1)
			n1 := registerNodeWithSymbol(ctx, mod, root, "m1", 1, 2)
			n2 := registerNodeWithSymbol(ctx, mod, root, "m2", 1, 3)

			createResolvedGroups(ctx)

			groups := ctx.mib.Groups()
			testutil.Len(t, groups, 1, "mib groups")
			testutil.Equal(t, "testGroup", groups[0].Name(), "group name")
			testutil.Equal(t, isNotifGrp, groups[0].IsNotificationGroup(), "IsNotificationGroup")
			testutil.Len(t, groups[0].Members(), 2, "members")
			testutil.Equal(t, n1, groups[0].Members()[0], "member 0")
			testutil.Equal(t, n2, groups[0].Members()[1], "member 1")

			testutil.NotNil(t, grpNode.Group(), "group should be set on node")
			testutil.Equal(t, "testGroup", grpNode.Group().Name(), "group name on node")
			testutil.Len(t, ctx.moduleToResolved[mod].Groups(), 1, "module groups")
		})

		t.Run(label+"/unresolved group node skipped", func(t *testing.T) {
			def := groupTestDef("missingGroup", nameRefs("m1"), isNotifGrp)
			mod := &module.Module{
				Name: "TEST-MIB",
			}
			addDefs(mod, []module.Definition{def})

			ctx := newTestContextForModules(model.DefaultConfig(), mod)

			createResolvedGroups(ctx)
			testutil.Len(t, ctx.mib.Groups(), 0, "should create no groups when node is missing")
		})

		t.Run(label+"/unresolved member emits diagnostic", func(t *testing.T) {
			def := groupTestDef("testGroup", nameRefs("m1", "missing"), isNotifGrp)
			mod := &module.Module{
				Name: "TEST-MIB",
			}
			addDefs(mod, []module.Definition{def})

			ctx := newTestContextForModules(model.DefaultConfig(), mod)
			root := ctx.mib.Root()

			registerNodeWithSymbol(ctx, mod, root, "testGroup", 1, 1)
			n1 := registerNodeWithSymbol(ctx, mod, root, "m1", 1, 2)

			createResolvedGroups(ctx)

			groups := ctx.mib.Groups()
			testutil.Len(t, groups, 1, "mib groups")
			testutil.Len(t, groups[0].Members(), 1, "only resolved member should be present")
			testutil.Equal(t, n1, groups[0].Members()[0], "resolved member")

			hasDiag(t, ctx.Diagnostics(), types.DiagGroupMemberUnresolved)
		})

		t.Run(label+"/wrong-kind member emits diagnostic", func(t *testing.T) {
			def := groupTestDef("testGroup", nameRefs("wrong1"), isNotifGrp)
			mod := &module.Module{
				Name: "TEST-MIB",
			}
			addDefs(mod, []module.Definition{def})

			ctx := newTestContextForModules(model.DefaultConfig(), mod)
			root := ctx.mib.Root()

			registerNodeWithSymbol(ctx, mod, root, "testGroup", 1, 1)

			m := registerNodeWithSymbol(ctx, mod, root, "wrong1", 1, 2)
			if isNotifGrp {
				model.SetNodeKind(m, types.KindScalar)
			} else {
				model.SetNodeKind(m, types.KindNotification)
			}

			createResolvedGroups(ctx)

			if isNotifGrp {
				hasDiag(t, ctx.Diagnostics(), types.DiagGroupNotificationsObject)
			} else {
				hasDiag(t, ctx.Diagnostics(), types.DiagGroupObjectsNotification)
			}
		})

		t.Run(label+"/mixed members emit group-member-mixed", func(t *testing.T) {
			def := groupTestDef("mixedGroup", nameRefs("obj1", "notif1"), isNotifGrp)
			mod := &module.Module{
				Name: "TEST-MIB",
			}
			addDefs(mod, []module.Definition{def})

			ctx := newTestContextForModules(model.DefaultConfig(), mod)
			root := ctx.mib.Root()

			registerNodeWithSymbol(ctx, mod, root, "mixedGroup", 1, 1)

			m1 := registerNodeWithSymbol(ctx, mod, root, "obj1", 1, 2)
			model.SetNodeKind(m1, types.KindScalar)

			n1 := registerNodeWithSymbol(ctx, mod, root, "notif1", 1, 3)
			model.SetNodeKind(n1, types.KindNotification)

			createResolvedGroups(ctx)

			hasDiag(t, ctx.Diagnostics(), types.DiagGroupMemberMixed)
			if isNotifGrp {
				hasDiag(t, ctx.Diagnostics(), types.DiagGroupNotificationsObject)
			} else {
				hasDiag(t, ctx.Diagnostics(), types.DiagGroupObjectsNotification)
			}
		})

		t.Run(label+"/deprecated member in current group emits status diagnostic", func(t *testing.T) {
			def := groupTestDef("currentGroup", nameRefs("depMember"), isNotifGrp)
			mod := &module.Module{
				Name: "TEST-MIB",
			}
			addDefs(mod, []module.Definition{def})

			ctx := newTestContextForModules(model.VerboseConfig(), mod)
			root := ctx.mib.Root()

			registerNodeWithSymbol(ctx, mod, root, "currentGroup", 1, 1)

			m := registerNodeWithSymbol(ctx, mod, root, "depMember", 1, 2)
			if isNotifGrp {
				model.SetNodeKind(m, types.KindNotification)
				notif := model.NewNotification("depMember")
				model.SetNotificationStatus(notif, types.StatusDeprecated)
				model.SetNodeNotification(m, notif)
			} else {
				model.SetNodeKind(m, types.KindScalar)
				obj := model.NewObject("depMember")
				model.SetObjectStatus(obj, types.StatusObsolete)
				model.SetNodeObject(m, obj)
			}

			createResolvedGroups(ctx)

			hasDiag(t, ctx.Diagnostics(), types.DiagGroupObjectStatus)
		})
	}

	// Object-group-specific tests.

	t.Run("object-group/not-accessible member emits diagnostic", func(t *testing.T) {
		mod := &module.Module{
			Name: "TEST-MIB",
		}
		addDefs(mod, []module.Definition{
			&module.ObjectGroup{
				DefBase: module.DefBase{Name: "testGroup"},
				Objects: nameRefs("notAccessObj"),
			},
		})

		ctx := newTestContextForModules(model.DefaultConfig(), mod)
		root := ctx.mib.Root()

		registerNodeWithSymbol(ctx, mod, root, "testGroup", 1, 1)

		memberNode := registerNodeWithSymbol(ctx, mod, root, "notAccessObj", 1, 2)

		memberObj := model.NewObject("notAccessObj")
		model.SetObjectAccess(memberObj, types.AccessNotAccessible)
		model.SetNodeObject(memberNode, memberObj)

		createResolvedGroups(ctx)

		hasDiag(t, ctx.Diagnostics(), types.DiagGroupNotAccessible)
	})

	t.Run("object-group/current member in current group no status diagnostic", func(t *testing.T) {
		mod := &module.Module{
			Name: "TEST-MIB",
		}
		addDefs(mod, []module.Definition{
			&module.ObjectGroup{
				DefBase: module.DefBase{Name: "currentGroup"},
				Objects: nameRefs("currentObj"),
				Status:  types.StatusCurrent,
			},
		})

		ctx := newTestContextForModules(model.VerboseConfig(), mod)
		root := ctx.mib.Root()

		registerNodeWithSymbol(ctx, mod, root, "currentGroup", 1, 1)

		m1 := registerNodeWithSymbol(ctx, mod, root, "currentObj", 1, 2)
		model.SetNodeKind(m1, types.KindScalar)

		obj := model.NewObject("currentObj")
		model.SetObjectStatus(obj, types.StatusCurrent)
		model.SetNodeObject(m1, obj)

		createResolvedGroups(ctx)

		noDiag(t, ctx.Diagnostics(), types.DiagGroupObjectStatus)
	})

	t.Run("object-group/smiv1 status member skips status check", func(t *testing.T) {
		mod := &module.Module{
			Name: "TEST-MIB",
		}
		addDefs(mod, []module.Definition{
			&module.ObjectGroup{
				DefBase: module.DefBase{Name: "currentGroup"},
				Objects: nameRefs("mandatoryObj"),
				Status:  types.StatusCurrent,
			},
		})

		ctx := newTestContextForModules(model.VerboseConfig(), mod)
		root := ctx.mib.Root()

		registerNodeWithSymbol(ctx, mod, root, "currentGroup", 1, 1)

		m1 := registerNodeWithSymbol(ctx, mod, root, "mandatoryObj", 1, 2)
		model.SetNodeKind(m1, types.KindScalar)

		obj := model.NewObject("mandatoryObj")
		model.SetObjectStatus(obj, types.StatusMandatory)
		model.SetNodeObject(m1, obj)

		createResolvedGroups(ctx)

		noDiag(t, ctx.Diagnostics(), types.DiagGroupObjectStatus)
	})
}

func TestRegisterNotification(t *testing.T) {
	mod := &module.Module{Name: "TEST-MIB"}
	ctx := newTestContextForModules(model.DefaultConfig(), mod)
	resolvedMod := ctx.moduleToResolved[mod]

	node := buildOIDPath(ctx.mib.Root(), 1, 1)
	notif := model.NewNotification("testNotif")

	registerNotification(ctx, mod, node, notif)

	testutil.Len(t, ctx.mib.Notifications(), 1, "mib notifications")
	testutil.NotNil(t, node.Notification(), "node notification")
	testutil.Equal(t, notif, node.Notification(), "node notification")
	testutil.Len(t, resolvedMod.Notifications(), 1, "module notifications")
}

func TestRegisterGroup(t *testing.T) {
	mod := &module.Module{Name: "TEST-MIB"}
	ctx := newTestContextForModules(model.DefaultConfig(), mod)

	node := buildOIDPath(ctx.mib.Root(), 1, 1)
	grp := model.NewGroup("testGroup")

	registerGroup(ctx, mod, node, grp)

	testutil.Len(t, ctx.mib.Groups(), 1, "mib groups")
	testutil.NotNil(t, node.Group(), "node group")
	testutil.Equal(t, grp, node.Group(), "node group")
	testutil.Len(t, ctx.moduleToResolved[mod].Groups(), 1, "module groups")
}

func TestRegisterCompliance(t *testing.T) {
	mod := &module.Module{Name: "TEST-MIB"}
	ctx := newTestContextForModules(model.DefaultConfig(), mod)
	resolvedMod := ctx.moduleToResolved[mod]

	node := buildOIDPath(ctx.mib.Root(), 1, 1)
	comp := model.NewCompliance("testCompliance")

	registerCompliance(ctx, mod, node, comp)

	testutil.Len(t, ctx.mib.Compliances(), 1, "mib compliances")
	testutil.NotNil(t, node.Compliance(), "node compliance")
	testutil.Equal(t, comp, node.Compliance(), "node compliance")
	testutil.Len(t, resolvedMod.Compliances(), 1, "module compliances")
}

func TestRegisterCapability(t *testing.T) {
	mod := &module.Module{Name: "TEST-MIB"}
	ctx := newTestContextForModules(model.DefaultConfig(), mod)
	resolvedMod := ctx.moduleToResolved[mod]

	node := buildOIDPath(ctx.mib.Root(), 1, 1)
	capability := model.NewCapability("testCapability")

	registerCapability(ctx, mod, node, capability)

	testutil.Len(t, ctx.mib.Capabilities(), 1, "mib capabilities")
	testutil.NotNil(t, node.Capability(), "node capability")
	testutil.Equal(t, capability, node.Capability(), "node capability")
	testutil.Len(t, resolvedMod.Capabilities(), 1, "module capabilities")
}

func TestCreateResolvedNotifications_FullCycle(t *testing.T) {
	t.Run("creates notification with objects", func(t *testing.T) {
		mod := &module.Module{
			Name: "TEST-MIB",
		}
		addDefs(mod, []module.Definition{
			&module.Notification{
				DefBase: module.DefBase{Name: "myNotif"},
				Objects: nameRefs("obj1", "obj2"),
				Status:  types.StatusCurrent,
			},
		})

		ctx := newTestContextForModules(model.DefaultConfig(), mod)

		root := ctx.mib.Root()

		registerNodeWithSymbol(ctx, mod, root, "myNotif", 1, 1)

		obj1Node := registerNodeWithSymbol(ctx, mod, root, "obj1", 1, 2)
		obj1 := model.NewObject("obj1")
		model.SetNodeObject(obj1Node, obj1)

		obj2Node := registerNodeWithSymbol(ctx, mod, root, "obj2", 1, 3)
		obj2 := model.NewObject("obj2")
		model.SetNodeObject(obj2Node, obj2)

		createResolvedNotifications(ctx)

		notifs := ctx.mib.Notifications()
		testutil.Len(t, notifs, 1, "mib notifications")
		testutil.Equal(t, "myNotif", notifs[0].Name(), "notification name")
		testutil.Len(t, notifs[0].Objects(), 2, "notification objects")
	})

	t.Run("unresolved object emits diagnostic", func(t *testing.T) {
		mod := &module.Module{
			Name: "TEST-MIB",
		}
		addDefs(mod, []module.Definition{
			&module.Notification{
				DefBase: module.DefBase{Name: "myNotif"},
				Objects: nameRefs("missingObj"),
			},
		})

		ctx := newTestContextForModules(model.DefaultConfig(), mod)

		root := ctx.mib.Root()
		registerNodeWithSymbol(ctx, mod, root, "myNotif", 1, 1)

		createResolvedNotifications(ctx)

		diags := ctx.Diagnostics()
		testutil.Len(t, diags, 1, "expected 1 diagnostic for unresolved notification object")
		testutil.Equal(t, types.DiagObjectsUnresolved, diags[0].Code, "diagnostic code")
	})

	t.Run("permissive global lookup for notification objects", func(t *testing.T) {
		modA := &module.Module{
			Name: "A-MIB",
		}
		addDefs(modA, []module.Definition{
			&module.Notification{
				DefBase: module.DefBase{Name: "myNotif"},
				Objects: nameRefs("foreignObj"),
			},
		})
		modB := &module.Module{Name: "B-MIB"}

		ctx := newTestContextForModulesWithPolicy(model.ResolverPermissive, model.DefaultConfig(), modA, modB)

		root := ctx.mib.Root()

		registerNodeWithSymbol(ctx, modA, root, "myNotif", 1, 1)

		// Object registered only in modB, not imported by modA.
		foreignNode := registerNodeWithSymbol(ctx, modB, root, "foreignObj", 2, 1)
		foreignObj := model.NewObject("foreignObj")
		model.SetNodeObject(foreignNode, foreignObj)

		createResolvedNotifications(ctx)

		notifs := ctx.mib.Notifications()
		testutil.Len(t, notifs, 1, "mib notifications")
		testutil.Len(t, notifs[0].Objects(), 1, "notification objects via permissive fallback")
	})

	t.Run("strict mode no global lookup for notification objects", func(t *testing.T) {
		modA := &module.Module{
			Name: "A-MIB",
		}
		addDefs(modA, []module.Definition{
			&module.Notification{
				DefBase: module.DefBase{Name: "myNotif"},
				Objects: nameRefs("foreignObj"),
			},
		})
		modB := &module.Module{Name: "B-MIB"}

		ctx := newTestContextForModulesWithPolicy(model.ResolverStrict, model.VerboseConfig(), modA, modB)

		root := ctx.mib.Root()

		registerNodeWithSymbol(ctx, modA, root, "myNotif", 1, 1)

		// Object registered only in modB, not imported by modA.
		foreignNode := registerNodeWithSymbol(ctx, modB, root, "foreignObj", 2, 1)
		foreignObj := model.NewObject("foreignObj")
		model.SetNodeObject(foreignNode, foreignObj)

		createResolvedNotifications(ctx)

		notifs := ctx.mib.Notifications()
		testutil.Len(t, notifs, 1, "mib notifications")
		testutil.Len(t, notifs[0].Objects(), 0, "strict mode should not resolve foreign notification objects")
	})

	t.Run("trap info preserved", func(t *testing.T) {
		mod := &module.Module{
			Name: "TEST-MIB",
		}
		addDefs(mod, []module.Definition{
			&module.Notification{
				DefBase: module.DefBase{Name: "myTrap"},
				TrapInfo: &module.TrapInfo{
					Enterprise: "enterprises",
					TrapNumber: 42,
				},
			},
		})

		ctx := newTestContextForModules(model.DefaultConfig(), mod)

		root := ctx.mib.Root()
		registerNodeWithSymbol(ctx, mod, root, "myTrap", 1, 1)

		createResolvedNotifications(ctx)

		notifs := ctx.mib.Notifications()
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
	}
	addDefs(mod, []module.Definition{
		&module.Notification{
			DefBase: module.DefBase{Name: "testNotif"},
			Objects: nameRefs("intermediateNode"),
		},
	})

	ctx := newTestContextForModules(model.DefaultConfig(), mod)

	// Create a node for the notification itself
	root := ctx.mib.Root()
	registerNodeWithSymbol(ctx, mod, root, "testNotif", 1)

	// Create a node for the referenced object, but do NOT set an Object on it.
	// This simulates an intermediate node or non-object definition.
	registerNodeWithSymbol(ctx, mod, root, "intermediateNode", 2)

	createResolvedNotifications(ctx)

	// Should have emitted a diagnostic for the nil-object case.
	hasDiag(t, ctx.Diagnostics(), types.DiagNotifObjectNotObject)
}
