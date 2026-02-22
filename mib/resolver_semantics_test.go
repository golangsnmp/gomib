package mib

import (
	"testing"

	"github.com/golangsnmp/gomib/internal/module"
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
			if got := isBareTypeIndex(tt.name); got != tt.want {
				t.Errorf("isBareTypeIndex(%q) = %v, want %v", tt.name, got, tt.want)
			}
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
			if got := isOIDType(tt.syntax); got != tt.want {
				t.Errorf("isOIDType() = %v, want %v", got, tt.want)
			}
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
			if err != nil {
				t.Fatalf("hexToBytes(%q) unexpected error: %v", tt.input, err)
			}
			if len(got) != len(tt.want) {
				t.Fatalf("hexToBytes(%q) len = %d, want %d", tt.input, len(got), len(tt.want))
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("hexToBytes(%q)[%d] = 0x%02X, want 0x%02X", tt.input, i, got[i], tt.want[i])
				}
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
			if len(got) != len(tt.want) {
				t.Fatalf("binaryToBytes(%q) len = %d, want %d", tt.input, len(got), len(tt.want))
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("binaryToBytes(%q)[%d] = 0x%02X, want 0x%02X", tt.input, i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestConvertDefValInteger(t *testing.T) {
	ctx := newTestContext()
	mod := &module.Module{Name: "TEST-MIB"}
	syntax := &module.TypeSyntaxTypeRef{Name: "Integer32"}

	dv := convertDefVal(ctx, &module.DefValInteger{Value: 42}, mod, syntax)
	if dv == nil {
		t.Fatal("convertDefVal returned nil")
	}
	if dv.Kind() != DefValKindInt {
		t.Errorf("kind = %v, want DefValKindInt", dv.Kind())
	}
	if v, ok := DefValAs[int64](*dv); !ok || v != 42 {
		t.Errorf("value = %v (ok=%v), want 42", v, ok)
	}
	if dv.Raw() != "42" {
		t.Errorf("raw = %q, want %q", dv.Raw(), "42")
	}
}

func TestConvertDefValNegativeInteger(t *testing.T) {
	ctx := newTestContext()
	mod := &module.Module{Name: "TEST-MIB"}
	syntax := &module.TypeSyntaxTypeRef{Name: "Integer32"}

	dv := convertDefVal(ctx, &module.DefValInteger{Value: -1}, mod, syntax)
	if dv == nil {
		t.Fatal("convertDefVal returned nil")
	}
	if dv.Kind() != DefValKindInt {
		t.Errorf("kind = %v, want DefValKindInt", dv.Kind())
	}
	if v, ok := DefValAs[int64](*dv); !ok || v != -1 {
		t.Errorf("value = %v (ok=%v), want -1", v, ok)
	}
	if dv.Raw() != "-1" {
		t.Errorf("raw = %q, want %q", dv.Raw(), "-1")
	}
}

func TestConvertDefValUnsigned(t *testing.T) {
	ctx := newTestContext()
	mod := &module.Module{Name: "TEST-MIB"}
	syntax := &module.TypeSyntaxTypeRef{Name: "Counter64"}

	dv := convertDefVal(ctx, &module.DefValUnsigned{Value: 12345}, mod, syntax)
	if dv == nil {
		t.Fatal("convertDefVal returned nil")
	}
	if dv.Kind() != DefValKindUint {
		t.Errorf("kind = %v, want DefValKindUint", dv.Kind())
	}
	if v, ok := DefValAs[uint64](*dv); !ok || v != 12345 {
		t.Errorf("value = %v (ok=%v), want 12345", v, ok)
	}
	if dv.Raw() != "12345" {
		t.Errorf("raw = %q, want %q", dv.Raw(), "12345")
	}
}

func TestConvertDefValString(t *testing.T) {
	ctx := newTestContext()
	mod := &module.Module{Name: "TEST-MIB"}
	syntax := &module.TypeSyntaxTypeRef{Name: "DisplayString"}

	dv := convertDefVal(ctx, &module.DefValString{Value: "public"}, mod, syntax)
	if dv == nil {
		t.Fatal("convertDefVal returned nil")
	}
	if dv.Kind() != DefValKindString {
		t.Errorf("kind = %v, want DefValKindString", dv.Kind())
	}
	if v, ok := DefValAs[string](*dv); !ok || v != "public" {
		t.Errorf("value = %v (ok=%v), want %q", v, ok, "public")
	}
	if dv.Raw() != `"public"` {
		t.Errorf("raw = %q, want %q", dv.Raw(), `"public"`)
	}
}

func TestConvertDefValHexString(t *testing.T) {
	ctx := newTestContext()
	mod := &module.Module{Name: "TEST-MIB"}
	syntax := &module.TypeSyntaxOctetString{}

	dv := convertDefVal(ctx, &module.DefValHexString{Value: "FF00"}, mod, syntax)
	if dv == nil {
		t.Fatal("convertDefVal returned nil")
	}
	if dv.Kind() != DefValKindBytes {
		t.Errorf("kind = %v, want DefValKindBytes", dv.Kind())
	}
	bytes, ok := DefValAs[[]byte](*dv)
	if !ok {
		t.Fatal("value is not []byte")
	}
	want := []byte{0xFF, 0x00}
	if len(bytes) != len(want) {
		t.Fatalf("bytes len = %d, want %d", len(bytes), len(want))
	}
	for i := range bytes {
		if bytes[i] != want[i] {
			t.Errorf("bytes[%d] = 0x%02X, want 0x%02X", i, bytes[i], want[i])
		}
	}
	if dv.Raw() != "'FF00'H" {
		t.Errorf("raw = %q, want %q", dv.Raw(), "'FF00'H")
	}
}

func TestConvertDefValBinaryString(t *testing.T) {
	ctx := newTestContext()
	mod := &module.Module{Name: "TEST-MIB"}
	syntax := &module.TypeSyntaxOctetString{}

	dv := convertDefVal(ctx, &module.DefValBinaryString{Value: "10101010"}, mod, syntax)
	if dv == nil {
		t.Fatal("convertDefVal returned nil")
	}
	if dv.Kind() != DefValKindBytes {
		t.Errorf("kind = %v, want DefValKindBytes", dv.Kind())
	}
	bytes, ok := DefValAs[[]byte](*dv)
	if !ok {
		t.Fatal("value is not []byte")
	}
	if len(bytes) != 1 || bytes[0] != 0xAA {
		t.Errorf("bytes = %v, want [0xAA]", bytes)
	}
	if dv.Raw() != "'10101010'B" {
		t.Errorf("raw = %q, want %q", dv.Raw(), "'10101010'B")
	}
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
	if dv == nil {
		t.Fatal("convertDefVal returned nil")
	}
	if dv.Kind() != DefValKindEnum {
		t.Errorf("kind = %v, want DefValKindEnum", dv.Kind())
	}
	if v, ok := DefValAs[string](*dv); !ok || v != "enabled" {
		t.Errorf("value = %v (ok=%v), want %q", v, ok, "enabled")
	}
	if dv.Raw() != "enabled" {
		t.Errorf("raw = %q, want %q", dv.Raw(), "enabled")
	}
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
	if dv == nil {
		t.Fatal("convertDefVal returned nil")
	}
	if dv.Kind() != DefValKindOID {
		t.Errorf("kind = %v, want DefValKindOID", dv.Kind())
	}
	oid, ok := DefValAs[OID](*dv)
	if !ok {
		t.Fatal("value is not OID")
	}
	if len(oid) != 2 || oid[0] != 1 || oid[1] != 3 {
		t.Errorf("oid = %v, want [1 3]", oid)
	}
}

func TestConvertDefValEnumOnOIDTypeFallback(t *testing.T) {
	// When syntax is OID type but node lookup fails, falls back to enum.
	ctx := newTestContext()
	mod := &module.Module{Name: "TEST-MIB"}
	syntax := &module.TypeSyntaxObjectIdentifier{}

	dv := convertDefVal(ctx, &module.DefValEnum{Name: "unknownOid"}, mod, syntax)
	if dv == nil {
		t.Fatal("convertDefVal returned nil")
	}
	if dv.Kind() != DefValKindEnum {
		t.Errorf("kind = %v, want DefValKindEnum (fallback)", dv.Kind())
	}
}

func TestConvertDefValBits(t *testing.T) {
	ctx := newTestContext()
	mod := &module.Module{Name: "TEST-MIB"}
	syntax := &module.TypeSyntaxBits{}

	dv := convertDefVal(ctx, &module.DefValBits{Labels: []string{"flag1", "flag2"}}, mod, syntax)
	if dv == nil {
		t.Fatal("convertDefVal returned nil")
	}
	if dv.Kind() != DefValKindBits {
		t.Errorf("kind = %v, want DefValKindBits", dv.Kind())
	}
	labels, ok := DefValAs[[]string](*dv)
	if !ok {
		t.Fatal("value is not []string")
	}
	if len(labels) != 2 || labels[0] != "flag1" || labels[1] != "flag2" {
		t.Errorf("labels = %v, want [flag1 flag2]", labels)
	}
	if dv.Raw() != "{ flag1, flag2 }" {
		t.Errorf("raw = %q, want %q", dv.Raw(), "{ flag1, flag2 }")
	}
}

func TestConvertDefValBitsEmpty(t *testing.T) {
	ctx := newTestContext()
	mod := &module.Module{Name: "TEST-MIB"}
	syntax := &module.TypeSyntaxBits{}

	dv := convertDefVal(ctx, &module.DefValBits{Labels: []string{}}, mod, syntax)
	if dv == nil {
		t.Fatal("convertDefVal returned nil")
	}
	if dv.Raw() != "{ }" {
		t.Errorf("raw = %q, want %q", dv.Raw(), "{ }")
	}
}

func TestConvertDefValOidRef(t *testing.T) {
	ctx := newTestContext()
	mod := &module.Module{Name: "TEST-MIB"}

	root := ctx.Mib.Root()
	child := root.getOrCreateChild(1)
	child.setName("sysName")
	ctx.registerModuleNodeSymbol(mod, "sysName", child)

	dv := convertDefVal(ctx, &module.DefValOidRef{Name: "sysName"}, mod, nil)
	if dv == nil {
		t.Fatal("convertDefVal returned nil")
	}
	if dv.Kind() != DefValKindOID {
		t.Errorf("kind = %v, want DefValKindOID", dv.Kind())
	}
}

func TestConvertDefValOidRefUnresolved(t *testing.T) {
	ctx := newTestContext()
	mod := &module.Module{Name: "TEST-MIB"}

	dv := convertDefVal(ctx, &module.DefValOidRef{Name: "missing"}, mod, nil)
	if dv != nil {
		t.Errorf("expected nil for unresolved OID ref, got %v", dv)
	}
}

func TestConvertDefValUnknown(t *testing.T) {
	ctx := newTestContext()
	mod := &module.Module{Name: "TEST-MIB"}

	dv := convertDefVal(ctx, &module.DefValUnparsed{}, mod, nil)
	if dv != nil {
		t.Errorf("expected nil for unparsed DefVal, got %v", dv)
	}
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
		if !ok {
			t.Fatal("expected ok=true")
		}
		if typ != intType {
			t.Errorf("got different type pointer")
		}
	})

	t.Run("TypeRef not found", func(t *testing.T) {
		syntax := &module.TypeSyntaxTypeRef{Name: "Unknown"}
		_, ok := resolveTypeSyntax(ctx, syntax, mod, "testObj", module.OidAssignment{}.Span)
		if ok {
			t.Error("expected ok=false for unknown type")
		}
	})

	t.Run("Constrained delegates to base", func(t *testing.T) {
		syntax := &module.TypeSyntaxConstrained{
			Base: &module.TypeSyntaxTypeRef{Name: "Integer32"},
		}
		typ, ok := resolveTypeSyntax(ctx, syntax, mod, "testObj", module.OidAssignment{}.Span)
		if !ok {
			t.Fatal("expected ok=true")
		}
		if typ != intType {
			t.Errorf("got different type pointer")
		}
	})

	t.Run("IntegerEnum with base", func(t *testing.T) {
		syntax := &module.TypeSyntaxIntegerEnum{
			Base:         "Integer32",
			NamedNumbers: []module.NamedNumber{{Name: "up", Value: 1}},
		}
		typ, ok := resolveTypeSyntax(ctx, syntax, mod, "testObj", module.OidAssignment{}.Span)
		if !ok {
			t.Fatal("expected ok=true")
		}
		if typ != intType {
			t.Errorf("got different type pointer")
		}
	})

	t.Run("IntegerEnum with unknown base", func(t *testing.T) {
		syntax := &module.TypeSyntaxIntegerEnum{
			Base:         "MissingType",
			NamedNumbers: []module.NamedNumber{{Name: "up", Value: 1}},
		}
		_, ok := resolveTypeSyntax(ctx, syntax, mod, "testObj", module.OidAssignment{}.Span)
		if ok {
			t.Error("expected ok=false for unknown base type")
		}
	})

	t.Run("SequenceOf returns false", func(t *testing.T) {
		syntax := &module.TypeSyntaxSequenceOf{EntryType: "MyEntry"}
		_, ok := resolveTypeSyntax(ctx, syntax, mod, "testObj", module.OidAssignment{}.Span)
		if ok {
			t.Error("expected ok=false for SequenceOf")
		}
	})

	t.Run("Sequence returns false", func(t *testing.T) {
		syntax := &module.TypeSyntaxSequence{}
		_, ok := resolveTypeSyntax(ctx, syntax, mod, "testObj", module.OidAssignment{}.Span)
		if ok {
			t.Error("expected ok=false for Sequence")
		}
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
		if !ok {
			t.Fatal("expected ok=true")
		}
		if typ != integerType {
			t.Errorf("expected INTEGER type")
		}
	})

	t.Run("Bits resolves to BITS", func(t *testing.T) {
		syntax := &module.TypeSyntaxBits{}
		typ, ok := resolveTypeSyntax(ctx, syntax, mod, "testObj", module.OidAssignment{}.Span)
		if !ok {
			t.Fatal("expected ok=true")
		}
		if typ != bitsType {
			t.Errorf("expected BITS type")
		}
	})

	t.Run("OctetString resolves to OCTET STRING", func(t *testing.T) {
		syntax := &module.TypeSyntaxOctetString{}
		typ, ok := resolveTypeSyntax(ctx, syntax, mod, "testObj", module.OidAssignment{}.Span)
		if !ok {
			t.Fatal("expected ok=true")
		}
		if typ != octetType {
			t.Errorf("expected OCTET STRING type")
		}
	})

	t.Run("ObjectIdentifier resolves to OBJECT IDENTIFIER", func(t *testing.T) {
		syntax := &module.TypeSyntaxObjectIdentifier{}
		typ, ok := resolveTypeSyntax(ctx, syntax, mod, "testObj", module.OidAssignment{}.Span)
		if !ok {
			t.Fatal("expected ok=true")
		}
		if typ != oidType {
			t.Errorf("expected OBJECT IDENTIFIER type")
		}
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

		if obj.EffectiveDisplayHint() != "255a" {
			t.Errorf("hint = %q, want %q", obj.EffectiveDisplayHint(), "255a")
		}
	})

	t.Run("object hint takes precedence over type hint", func(t *testing.T) {
		typ := newType("DisplayString")
		typ.setDisplayHint("255a")

		obj := newObject("testObj")
		obj.setType(typ)
		obj.setEffectiveHint("1d")
		computeEffectiveValues(obj)

		if obj.EffectiveDisplayHint() != "1d" {
			t.Errorf("hint = %q, want %q", obj.EffectiveDisplayHint(), "1d")
		}
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
		if len(sizes) != 1 || sizes[0].Min != 0 || sizes[0].Max != 255 {
			t.Errorf("sizes = %v, want [{0 255}]", sizes)
		}
	})

	t.Run("object sizes take precedence", func(t *testing.T) {
		typ := newType("DisplayString")
		typ.setSizes([]Range{{Min: 0, Max: 255}})

		obj := newObject("testObj")
		obj.setType(typ)
		obj.setEffectiveSizes([]Range{{Min: 1, Max: 32}})
		computeEffectiveValues(obj)

		sizes := obj.EffectiveSizes()
		if len(sizes) != 1 || sizes[0].Min != 1 || sizes[0].Max != 32 {
			t.Errorf("sizes = %v, want [{1 32}]", sizes)
		}
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
		if len(ranges) != 1 || ranges[0].Min != -128 || ranges[0].Max != 127 {
			t.Errorf("ranges = %v, want [{-128 127}]", ranges)
		}
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
		if len(enums) != 2 || enums[0].Label != "up" || enums[1].Label != "down" {
			t.Errorf("enums = %v, want up/down", enums)
		}
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
		if len(enums) != 1 || enums[0].Label != "active" {
			t.Errorf("enums = %v, want [active]", enums)
		}
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
		if len(bits) != 2 || bits[0].Label != "feature1" {
			t.Errorf("bits = %v, want feature1/feature2", bits)
		}
	})
}

func TestConvertComplianceModules(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		result := convertComplianceModules(nil, nil, nil)
		if len(result) != 0 {
			t.Errorf("expected empty, got %d", len(result))
		}
	})

	t.Run("basic module with mandatory groups", func(t *testing.T) {
		input := []module.ComplianceModule{
			{
				ModuleName:      "IF-MIB",
				MandatoryGroups: []string{"ifGeneralGroup", "ifStackGroup"},
			},
		}
		result := convertComplianceModules(nil, nil, input)
		if len(result) != 1 {
			t.Fatalf("expected 1, got %d", len(result))
		}
		if result[0].ModuleName != "IF-MIB" {
			t.Errorf("module name = %q, want %q", result[0].ModuleName, "IF-MIB")
		}
		if len(result[0].MandatoryGroups) != 2 {
			t.Errorf("mandatory groups = %d, want 2", len(result[0].MandatoryGroups))
		}
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
		if len(result[0].Groups) != 1 {
			t.Fatalf("groups = %d, want 1", len(result[0].Groups))
		}
		if result[0].Groups[0].Group != "ifCounterGroup" {
			t.Errorf("group = %q, want %q", result[0].Groups[0].Group, "ifCounterGroup")
		}
		if result[0].Groups[0].Description != "counter desc" {
			t.Errorf("desc = %q", result[0].Groups[0].Description)
		}
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
		if len(result[0].Objects) != 2 {
			t.Fatalf("objects = %d, want 2", len(result[0].Objects))
		}
		obj0 := result[0].Objects[0]
		if obj0.Object != "ifAdminStatus" {
			t.Errorf("object = %q", obj0.Object)
		}
		if obj0.MinAccess == nil || *obj0.MinAccess != AccessReadOnly {
			t.Errorf("min-access = %v, want read-only", obj0.MinAccess)
		}
		if obj0.Description != "obj desc" {
			t.Errorf("desc = %q", obj0.Description)
		}
		obj1 := result[0].Objects[1]
		if obj1.MinAccess != nil {
			t.Errorf("expected nil min-access for second object")
		}
	})

	t.Run("multiple modules", func(t *testing.T) {
		input := []module.ComplianceModule{
			{ModuleName: "IF-MIB"},
			{ModuleName: "IP-MIB"},
		}
		result := convertComplianceModules(nil, nil, input)
		if len(result) != 2 {
			t.Fatalf("expected 2, got %d", len(result))
		}
		if result[0].ModuleName != "IF-MIB" || result[1].ModuleName != "IP-MIB" {
			t.Errorf("module names = %q, %q", result[0].ModuleName, result[1].ModuleName)
		}
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
		if obj.Syntax == nil {
			t.Fatal("expected non-nil Syntax")
		}
		if obj.Syntax.Type != intType {
			t.Error("Syntax.Type does not match")
		}
		if len(obj.Syntax.Ranges) != 1 || obj.Syntax.Ranges[0].Min != 0 || obj.Syntax.Ranges[0].Max != 100 {
			t.Errorf("Syntax.Ranges = %v, want [{0 100}]", obj.Syntax.Ranges)
		}
		if obj.WriteSyntax == nil {
			t.Fatal("expected non-nil WriteSyntax")
		}
		if len(obj.WriteSyntax.Ranges) != 1 || obj.WriteSyntax.Ranges[0].Min != 1 || obj.WriteSyntax.Ranges[0].Max != 50 {
			t.Errorf("WriteSyntax.Ranges = %v, want [{1 50}]", obj.WriteSyntax.Ranges)
		}
	})
}

func TestConvertSupportsModules(t *testing.T) {
	ctx := newTestContext()
	mod := &module.Module{Name: "TEST-MIB"}

	t.Run("empty", func(t *testing.T) {
		result := convertSupportsModules(ctx, mod, nil)
		if len(result) != 0 {
			t.Errorf("expected empty, got %d", len(result))
		}
	})

	t.Run("basic with includes", func(t *testing.T) {
		input := []module.SupportsModule{
			{
				ModuleName: "IF-MIB",
				Includes:   []string{"ifGeneralGroup"},
			},
		}
		result := convertSupportsModules(ctx, mod, input)
		if len(result) != 1 {
			t.Fatalf("expected 1, got %d", len(result))
		}
		if result[0].ModuleName != "IF-MIB" {
			t.Errorf("module name = %q", result[0].ModuleName)
		}
		if len(result[0].Includes) != 1 || result[0].Includes[0] != "ifGeneralGroup" {
			t.Errorf("includes = %v", result[0].Includes)
		}
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
		if len(vars) != 2 {
			t.Fatalf("variations = %d, want 2", len(vars))
		}
		if vars[0].Object != "ifAdminStatus" {
			t.Errorf("object = %q", vars[0].Object)
		}
		if vars[0].Access == nil || *vars[0].Access != AccessReadOnly {
			t.Errorf("access = %v, want read-only", vars[0].Access)
		}
		if vars[0].Description != "read only" {
			t.Errorf("desc = %q", vars[0].Description)
		}
		if vars[1].Access != nil {
			t.Errorf("expected nil access for second variation")
		}
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
		if len(vars) != 2 {
			t.Fatalf("notification variations = %d, want 2", len(vars))
		}
		if vars[0].Notification != "linkDown" {
			t.Errorf("notification = %q", vars[0].Notification)
		}
		if vars[0].Access == nil || *vars[0].Access != AccessReadOnly {
			t.Errorf("access = %v, want read-only", vars[0].Access)
		}
		if vars[0].Description != "not supported" {
			t.Errorf("desc = %q", vars[0].Description)
		}
		if vars[1].Access != nil {
			t.Errorf("expected nil access for second variation")
		}
	})

	t.Run("SPPI access values preserved in variations", func(t *testing.T) {
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
		if len(vars) != 1 {
			t.Fatalf("variations = %d, want 1", len(vars))
		}
		if vars[0].Access == nil {
			t.Fatal("access is nil, want not-implemented")
		}
		if *vars[0].Access != AccessNotImplemented {
			t.Errorf("access = %v, want not-implemented (%v)", *vars[0].Access, AccessNotImplemented)
		}
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
		if len(result[0].ObjectVariations) != 1 {
			t.Errorf("object variations = %d", len(result[0].ObjectVariations))
		}
		if len(result[0].NotificationVariations) != 1 {
			t.Errorf("notification variations = %d", len(result[0].NotificationVariations))
		}
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
		if v.Syntax == nil {
			t.Fatal("expected non-nil Syntax")
		}
		if v.Syntax.Type != intType {
			t.Error("Syntax.Type does not match")
		}
		if len(v.Syntax.Ranges) != 1 || v.Syntax.Ranges[0].Min != 1 || v.Syntax.Ranges[0].Max != 2 {
			t.Errorf("Syntax.Ranges = %v, want [{1 2}]", v.Syntax.Ranges)
		}
		if v.WriteSyntax == nil {
			t.Fatal("expected non-nil WriteSyntax")
		}
		if v.WriteSyntax.Type != intType {
			t.Error("WriteSyntax.Type does not match")
		}
		if len(v.CreationRequires) != 2 || v.CreationRequires[0] != "ifType" || v.CreationRequires[1] != "ifSpeed" {
			t.Errorf("CreationRequires = %v, want [ifType ifSpeed]", v.CreationRequires)
		}
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
		if len(vars) != 1 {
			t.Fatalf("variations = %d, want 1", len(vars))
		}
		if vars[0].DefVal.IsZero() {
			t.Fatal("expected non-zero DefVal")
		}
		if vars[0].DefVal.Kind() != DefValKindInt {
			t.Errorf("defval kind = %v, want DefValKindInt", vars[0].DefVal.Kind())
		}
		v, ok := DefValAs[int64](vars[0].DefVal)
		if !ok || v != 2 {
			t.Errorf("defval value = %v (ok=%v), want 2", v, ok)
		}
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
		if dv == nil {
			t.Fatal("expected non-nil")
		}
		if dv.Kind() != DefValKindOID {
			t.Errorf("kind = %v, want DefValKindOID", dv.Kind())
		}
		oid, ok := DefValAs[OID](*dv)
		if !ok {
			t.Fatal("expected OID value")
		}
		want := OID{1, 3, 42, 1}
		if oid.String() != want.String() {
			t.Errorf("oid = %v, want %v", oid, want)
		}
	})

	t.Run("resolves single name component", func(t *testing.T) {
		dv := convertDefVal(ctx, &module.DefValOidValue{
			Components: []module.OidComponent{
				&module.OidComponentName{NameValue: "enterprises"},
			},
		}, mod, nil)
		if dv == nil {
			t.Fatal("expected non-nil")
		}
		oid, ok := DefValAs[OID](*dv)
		if !ok {
			t.Fatal("expected OID value")
		}
		want := OID{1, 3}
		if oid.String() != want.String() {
			t.Errorf("oid = %v, want %v", oid, want)
		}
	})

	t.Run("resolves named number with trailing arcs", func(t *testing.T) {
		dv := convertDefVal(ctx, &module.DefValOidValue{
			Components: []module.OidComponent{
				&module.OidComponentNamedNumber{NameValue: "enterprises", NumberValue: 1},
				&module.OidComponentNumber{Value: 5},
			},
		}, mod, nil)
		if dv == nil {
			t.Fatal("expected non-nil")
		}
		oid, ok := DefValAs[OID](*dv)
		if !ok {
			t.Fatal("expected OID value")
		}
		want := OID{1, 3, 5}
		if oid.String() != want.String() {
			t.Errorf("oid = %v, want %v", oid, want)
		}
	})

	t.Run("resolves qualified name", func(t *testing.T) {
		dv := convertDefVal(ctx, &module.DefValOidValue{
			Components: []module.OidComponent{
				&module.OidComponentQualifiedName{ModuleValue: "SNMPv2-SMI", NameValue: "enterprises"},
			},
		}, mod, nil)
		if dv == nil {
			t.Fatal("expected non-nil")
		}
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
		if dv == nil {
			t.Fatal("expected non-nil")
		}
	})

	t.Run("returns nil for unresolved name", func(t *testing.T) {
		dv := convertDefVal(ctx, &module.DefValOidValue{
			Components: []module.OidComponent{
				&module.OidComponentName{NameValue: "nonexistent"},
			},
		}, mod, nil)
		if dv != nil {
			t.Errorf("expected nil for unresolved name, got %v", dv)
		}
	})

	t.Run("returns nil for empty components", func(t *testing.T) {
		dv := convertDefVal(ctx, &module.DefValOidValue{
			Components: nil,
		}, mod, nil)
		if dv != nil {
			t.Errorf("expected nil for empty components, got %v", dv)
		}
	})

	t.Run("returns nil when first component is numeric only", func(t *testing.T) {
		dv := convertDefVal(ctx, &module.DefValOidValue{
			Components: []module.OidComponent{
				&module.OidComponentNumber{Value: 1},
			},
		}, mod, nil)
		if dv != nil {
			t.Errorf("expected nil when first component is numeric, got %v", dv)
		}
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
	if !found {
		t.Error("expected diagnostic for notification object with nil Object, got none")
	}
}
