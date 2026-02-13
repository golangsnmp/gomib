package resolver

import (
	"testing"

	"github.com/golangsnmp/gomib/internal/mibimpl"
	"github.com/golangsnmp/gomib/internal/module"
	"github.com/golangsnmp/gomib/mib"
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
			got := hexToBytes(tt.input)
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

func TestConvertAccess(t *testing.T) {
	tests := []struct {
		input module.Access
		want  mib.Access
	}{
		{module.AccessNotAccessible, mib.AccessNotAccessible},
		{module.AccessAccessibleForNotify, mib.AccessAccessibleForNotify},
		{module.AccessReadOnly, mib.AccessReadOnly},
		{module.AccessReadWrite, mib.AccessReadWrite},
		{module.AccessReadCreate, mib.AccessReadCreate},
		{module.AccessWriteOnly, mib.AccessWriteOnly},
	}
	for _, tt := range tests {
		t.Run(tt.want.String(), func(t *testing.T) {
			if got := convertAccess(tt.input); got != tt.want {
				t.Errorf("convertAccess(%d) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestConvertAccessUnknown(t *testing.T) {
	// Unknown access values should default to not-accessible
	got := convertAccess(module.Access(999))
	if got != mib.AccessNotAccessible {
		t.Errorf("convertAccess(999) = %v, want %v", got, mib.AccessNotAccessible)
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
	if dv.Kind() != mib.DefValKindInt {
		t.Errorf("kind = %v, want DefValKindInt", dv.Kind())
	}
	if v, ok := mib.DefValAs[int64](*dv); !ok || v != 42 {
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
	if dv.Kind() != mib.DefValKindInt {
		t.Errorf("kind = %v, want DefValKindInt", dv.Kind())
	}
	if v, ok := mib.DefValAs[int64](*dv); !ok || v != -1 {
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
	if dv.Kind() != mib.DefValKindUint {
		t.Errorf("kind = %v, want DefValKindUint", dv.Kind())
	}
	if v, ok := mib.DefValAs[uint64](*dv); !ok || v != 12345 {
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
	if dv.Kind() != mib.DefValKindString {
		t.Errorf("kind = %v, want DefValKindString", dv.Kind())
	}
	if v, ok := mib.DefValAs[string](*dv); !ok || v != "public" {
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
	if dv.Kind() != mib.DefValKindBytes {
		t.Errorf("kind = %v, want DefValKindBytes", dv.Kind())
	}
	bytes, ok := mib.DefValAs[[]byte](*dv)
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
	if dv.Kind() != mib.DefValKindBytes {
		t.Errorf("kind = %v, want DefValKindBytes", dv.Kind())
	}
	bytes, ok := mib.DefValAs[[]byte](*dv)
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
	if dv.Kind() != mib.DefValKindEnum {
		t.Errorf("kind = %v, want DefValKindEnum", dv.Kind())
	}
	if v, ok := mib.DefValAs[string](*dv); !ok || v != "enabled" {
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
	root := ctx.Builder.Root()
	child := root.GetOrCreateChild(1)
	grandchild := child.GetOrCreateChild(3)
	grandchild.SetName("myTarget")
	ctx.RegisterModuleNodeSymbol(mod, "myTarget", grandchild)

	syntax := &module.TypeSyntaxObjectIdentifier{}
	dv := convertDefVal(ctx, &module.DefValEnum{Name: "myTarget"}, mod, syntax)
	if dv == nil {
		t.Fatal("convertDefVal returned nil")
	}
	if dv.Kind() != mib.DefValKindOID {
		t.Errorf("kind = %v, want DefValKindOID", dv.Kind())
	}
	oid, ok := mib.DefValAs[mib.Oid](*dv)
	if !ok {
		t.Fatal("value is not mib.Oid")
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
	if dv.Kind() != mib.DefValKindEnum {
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
	if dv.Kind() != mib.DefValKindBits {
		t.Errorf("kind = %v, want DefValKindBits", dv.Kind())
	}
	labels, ok := mib.DefValAs[[]string](*dv)
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

	root := ctx.Builder.Root()
	child := root.GetOrCreateChild(1)
	child.SetName("sysName")
	ctx.RegisterModuleNodeSymbol(mod, "sysName", child)

	dv := convertDefVal(ctx, &module.DefValOidRef{Name: "sysName"}, mod, nil)
	if dv == nil {
		t.Fatal("convertDefVal returned nil")
	}
	if dv.Kind() != mib.DefValKindOID {
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

	intType := mibimpl.NewType("Integer32")
	intType.SetBase(mib.BaseInteger32)
	ctx.RegisterModuleTypeSymbol(mod, "Integer32", intType)

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
	ctx := newResolverContext([]*module.Module{smiMod}, nil, mib.DefaultConfig())
	ctx.Snmpv2SMIModule = smiMod

	integerType := mibimpl.NewType("INTEGER")
	integerType.SetBase(mib.BaseInteger32)
	ctx.RegisterModuleTypeSymbol(smiMod, "INTEGER", integerType)

	bitsType := mibimpl.NewType("BITS")
	bitsType.SetBase(mib.BaseBits)
	ctx.RegisterModuleTypeSymbol(smiMod, "BITS", bitsType)

	octetType := mibimpl.NewType("OCTET STRING")
	octetType.SetBase(mib.BaseOctetString)
	ctx.RegisterModuleTypeSymbol(smiMod, "OCTET STRING", octetType)

	oidType := mibimpl.NewType("OBJECT IDENTIFIER")
	oidType.SetBase(mib.BaseObjectIdentifier)
	ctx.RegisterModuleTypeSymbol(smiMod, "OBJECT IDENTIFIER", oidType)

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
		obj := mibimpl.NewObject("testObj")
		// No type set, should not panic
		computeEffectiveValues(obj)
	})

	t.Run("inherits display hint from type", func(t *testing.T) {
		typ := mibimpl.NewType("DisplayString")
		typ.SetDisplayHint("255a")

		obj := mibimpl.NewObject("testObj")
		obj.SetType(typ)
		computeEffectiveValues(obj)

		if obj.EffectiveDisplayHint() != "255a" {
			t.Errorf("hint = %q, want %q", obj.EffectiveDisplayHint(), "255a")
		}
	})

	t.Run("object hint takes precedence over type hint", func(t *testing.T) {
		typ := mibimpl.NewType("DisplayString")
		typ.SetDisplayHint("255a")

		obj := mibimpl.NewObject("testObj")
		obj.SetType(typ)
		obj.SetEffectiveHint("1d")
		computeEffectiveValues(obj)

		if obj.EffectiveDisplayHint() != "1d" {
			t.Errorf("hint = %q, want %q", obj.EffectiveDisplayHint(), "1d")
		}
	})

	t.Run("inherits sizes from type chain", func(t *testing.T) {
		parentType := mibimpl.NewType("OCTET STRING")
		parentType.SetSizes([]mib.Range{{Min: 0, Max: 255}})

		childType := mibimpl.NewType("DisplayString")
		childType.SetParent(parentType)

		obj := mibimpl.NewObject("testObj")
		obj.SetType(childType)
		computeEffectiveValues(obj)

		sizes := obj.EffectiveSizes()
		if len(sizes) != 1 || sizes[0].Min != 0 || sizes[0].Max != 255 {
			t.Errorf("sizes = %v, want [{0 255}]", sizes)
		}
	})

	t.Run("object sizes take precedence", func(t *testing.T) {
		typ := mibimpl.NewType("DisplayString")
		typ.SetSizes([]mib.Range{{Min: 0, Max: 255}})

		obj := mibimpl.NewObject("testObj")
		obj.SetType(typ)
		obj.SetEffectiveSizes([]mib.Range{{Min: 1, Max: 32}})
		computeEffectiveValues(obj)

		sizes := obj.EffectiveSizes()
		if len(sizes) != 1 || sizes[0].Min != 1 || sizes[0].Max != 32 {
			t.Errorf("sizes = %v, want [{1 32}]", sizes)
		}
	})

	t.Run("inherits ranges from ancestor", func(t *testing.T) {
		grandparent := mibimpl.NewType("INTEGER")
		grandparent.SetRanges([]mib.Range{{Min: -128, Max: 127}})

		parent := mibimpl.NewType("Integer32")
		parent.SetParent(grandparent)

		child := mibimpl.NewType("MyInt")
		child.SetParent(parent)

		obj := mibimpl.NewObject("testObj")
		obj.SetType(child)
		computeEffectiveValues(obj)

		ranges := obj.EffectiveRanges()
		if len(ranges) != 1 || ranges[0].Min != -128 || ranges[0].Max != 127 {
			t.Errorf("ranges = %v, want [{-128 127}]", ranges)
		}
	})

	t.Run("inherits enums from type", func(t *testing.T) {
		typ := mibimpl.NewType("StatusType")
		typ.SetEnums([]mib.NamedValue{
			{Label: "up", Value: 1},
			{Label: "down", Value: 2},
		})

		obj := mibimpl.NewObject("testObj")
		obj.SetType(typ)
		computeEffectiveValues(obj)

		enums := obj.EffectiveEnums()
		if len(enums) != 2 || enums[0].Label != "up" || enums[1].Label != "down" {
			t.Errorf("enums = %v, want up/down", enums)
		}
	})

	t.Run("object enums take precedence over type enums", func(t *testing.T) {
		typ := mibimpl.NewType("StatusType")
		typ.SetEnums([]mib.NamedValue{
			{Label: "up", Value: 1},
			{Label: "down", Value: 2},
		})

		obj := mibimpl.NewObject("testObj")
		obj.SetType(typ)
		obj.SetEffectiveEnums([]mib.NamedValue{
			{Label: "active", Value: 1},
		})
		computeEffectiveValues(obj)

		enums := obj.EffectiveEnums()
		if len(enums) != 1 || enums[0].Label != "active" {
			t.Errorf("enums = %v, want [active]", enums)
		}
	})

	t.Run("inherits bits from type", func(t *testing.T) {
		typ := mibimpl.NewType("Capabilities")
		typ.SetBits([]mib.NamedValue{
			{Label: "feature1", Value: 0},
			{Label: "feature2", Value: 1},
		})

		obj := mibimpl.NewObject("testObj")
		obj.SetType(typ)
		computeEffectiveValues(obj)

		bits := obj.EffectiveBits()
		if len(bits) != 2 || bits[0].Label != "feature1" {
			t.Errorf("bits = %v, want feature1/feature2", bits)
		}
	})
}

func TestConvertComplianceModules(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		result := convertComplianceModules(nil)
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
		result := convertComplianceModules(input)
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
		result := convertComplianceModules(input)
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
		readOnly := module.AccessReadOnly
		input := []module.ComplianceModule{
			{
				ModuleName: "IF-MIB",
				Objects: []module.ComplianceObject{
					{Object: "ifAdminStatus", MinAccess: &readOnly, Description: "obj desc"},
					{Object: "ifOperStatus", MinAccess: nil},
				},
			},
		}
		result := convertComplianceModules(input)
		if len(result[0].Objects) != 2 {
			t.Fatalf("objects = %d, want 2", len(result[0].Objects))
		}
		obj0 := result[0].Objects[0]
		if obj0.Object != "ifAdminStatus" {
			t.Errorf("object = %q", obj0.Object)
		}
		if obj0.MinAccess == nil || *obj0.MinAccess != mib.AccessReadOnly {
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
		result := convertComplianceModules(input)
		if len(result) != 2 {
			t.Fatalf("expected 2, got %d", len(result))
		}
		if result[0].ModuleName != "IF-MIB" || result[1].ModuleName != "IP-MIB" {
			t.Errorf("module names = %q, %q", result[0].ModuleName, result[1].ModuleName)
		}
	})
}

func TestConvertSupportsModules(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		result := convertSupportsModules(nil)
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
		result := convertSupportsModules(input)
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
		readOnly := module.AccessReadOnly
		input := []module.SupportsModule{
			{
				ModuleName: "IF-MIB",
				ObjectVariations: []module.ObjectVariation{
					{Object: "ifAdminStatus", Access: &readOnly, Description: "read only"},
					{Object: "ifOperStatus", Access: nil, Description: "full access"},
				},
			},
		}
		result := convertSupportsModules(input)
		vars := result[0].ObjectVariations
		if len(vars) != 2 {
			t.Fatalf("variations = %d, want 2", len(vars))
		}
		if vars[0].Object != "ifAdminStatus" {
			t.Errorf("object = %q", vars[0].Object)
		}
		if vars[0].Access == nil || *vars[0].Access != mib.AccessReadOnly {
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
		readOnly := module.AccessReadOnly
		input := []module.SupportsModule{
			{
				ModuleName: "IF-MIB",
				NotificationVariations: []module.NotificationVariation{
					{Notification: "linkDown", Access: &readOnly, Description: "not supported"},
					{Notification: "linkUp", Access: nil},
				},
			},
		}
		result := convertSupportsModules(input)
		vars := result[0].NotificationVariations
		if len(vars) != 2 {
			t.Fatalf("notification variations = %d, want 2", len(vars))
		}
		if vars[0].Notification != "linkDown" {
			t.Errorf("notification = %q", vars[0].Notification)
		}
		if vars[0].Access == nil || *vars[0].Access != mib.AccessReadOnly {
			t.Errorf("access = %v, want read-only", vars[0].Access)
		}
		if vars[0].Description != "not supported" {
			t.Errorf("desc = %q", vars[0].Description)
		}
		if vars[1].Access != nil {
			t.Errorf("expected nil access for second variation")
		}
	})

	t.Run("mixed object and notification variations", func(t *testing.T) {
		readOnly := module.AccessReadOnly
		readWrite := module.AccessReadWrite
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
		result := convertSupportsModules(input)
		if len(result[0].ObjectVariations) != 1 {
			t.Errorf("object variations = %d", len(result[0].ObjectVariations))
		}
		if len(result[0].NotificationVariations) != 1 {
			t.Errorf("notification variations = %d", len(result[0].NotificationVariations))
		}
	})
}

func TestConvertDefValOidValue(t *testing.T) {
	ctx := newTestContext()
	mod := &module.Module{Name: "TEST-MIB"}

	root := ctx.Builder.Root()
	child := root.GetOrCreateChild(1)
	child2 := child.GetOrCreateChild(3)
	child2.SetName("enterprises")
	ctx.RegisterModuleNodeSymbol(mod, "enterprises", child2)

	t.Run("resolves first component name", func(t *testing.T) {
		dv := convertDefVal(ctx, &module.DefValOidValue{
			Components: []module.OidComponent{
				&module.OidComponentName{NameValue: "enterprises"},
				&module.OidComponentNumber{Value: 42},
			},
		}, mod, nil)
		if dv == nil {
			t.Fatal("expected non-nil")
		}
		if dv.Kind() != mib.DefValKindOID {
			t.Errorf("kind = %v, want DefValKindOID", dv.Kind())
		}
	})

	t.Run("resolves first component named number", func(t *testing.T) {
		dv := convertDefVal(ctx, &module.DefValOidValue{
			Components: []module.OidComponent{
				&module.OidComponentNamedNumber{NameValue: "enterprises", NumberValue: 1},
			},
		}, mod, nil)
		if dv == nil {
			t.Fatal("expected non-nil")
		}
		if dv.Kind() != mib.DefValKindOID {
			t.Errorf("kind = %v, want DefValKindOID", dv.Kind())
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
