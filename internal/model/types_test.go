package model

import (
	"testing"

	"github.com/golangsnmp/gomib/internal/module"
	"github.com/golangsnmp/gomib/internal/testutil"
	"github.com/golangsnmp/gomib/internal/types"
)

func TestDefValString(t *testing.T) {
	tests := []struct {
		name string
		dv   DefVal
		want string
	}{
		{"int positive", NewDefValInt(42, "42"), "42"},
		{"int negative", NewDefValInt(-1, "-1"), "-1"},
		{"int zero", NewDefValInt(0, "0"), "0"},
		{"uint", NewDefValUint(100, "100"), "100"},
		{"uint zero", NewDefValUint(0, "0"), "0"},
		{"string", NewDefValString("hello", `"hello"`), `"hello"`},
		{"string empty", NewDefValString("", `""`), `""`},
		{"string with quotes", NewDefValString(`say "hi"`, `"say \"hi\""`), `"say \"hi\""`},
		{"enum label", NewDefValEnum("active", "active"), "active"},
		{"bits multiple", NewDefValBits([]string{"read", "write"}, "{ read, write }"), "{ read, write }"},
		{"bits empty", NewDefValBits([]string{}, "{ }"), "{ }"},
		{"bits single", NewDefValBits([]string{"read"}, "{ read }"), "{ read }"},
		{"oid", NewDefValOID(OID{0, 0}, "zeroDotZero"), "zeroDotZero"},
		{"bytes empty", NewDefValBytes([]byte{}, "''H"), "0x"},
		{"bytes 1 byte", NewDefValBytes([]byte{0xFF}, "'FF'H"), "0xFF"},
		{"bytes 4 bytes", NewDefValBytes([]byte{0xDE, 0xAD, 0xBE, 0xEF}, "'DEADBEEF'H"), "0xDEADBEEF"},
		{"bytes 8 bytes", NewDefValBytes([]byte{0, 0, 0, 0, 0, 0, 0, 1}, "'0000000000000001'H"), "0x0000000000000001"},
		{"bytes >8 bytes", NewDefValBytes([]byte{0xAB, 0xCD, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01}, "x"), "0xABCD00000000000001"},
		{"bytes all zero >8", NewDefValBytes(make([]byte, 16), "x"), "0x00000000000000000000000000000000"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.dv.String()
			testutil.Equal(t, tt.want, got, "DefVal.String()")
		})
	}
}

func TestDefValKindString(t *testing.T) {
	tests := []struct {
		kind DefValKind
		want string
	}{
		{DefValKindUnset, "unset"},
		{DefValKindInt, "int"},
		{DefValKindUint, "uint"},
		{DefValKindString, "string"},
		{DefValKindBytes, "bytes"},
		{DefValKindEnum, "enum"},
		{DefValKindBits, "bits"},
		{DefValKindOID, "oid"},
		{DefValKind(99), "DefValKind(99)"},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := tt.kind.String()
			testutil.Equal(t, tt.want, got, "DefValKind().String()")
		})
	}
}

func TestDefValIsZero(t *testing.T) {
	var zero DefVal
	testutil.True(t, zero.IsZero(), "zero DefVal should report IsZero() true")
	testutil.Equal(t, DefValKindUnset, zero.Kind(), "zero DefVal Kind()")

	nonZero := NewDefValInt(0, "0")
	testutil.False(t, nonZero.IsZero(), "NewDefValInt(0) should not be IsZero (value is set, just happens to be 0)")
}

func TestDefValKind(t *testing.T) {
	tests := []struct {
		name string
		dv   DefVal
		want DefValKind
	}{
		{"int", NewDefValInt(1, "1"), DefValKindInt},
		{"uint", NewDefValUint(1, "1"), DefValKindUint},
		{"string", NewDefValString("x", "x"), DefValKindString},
		{"bytes", NewDefValBytes([]byte{1}, "x"), DefValKindBytes},
		{"enum", NewDefValEnum("x", "x"), DefValKindEnum},
		{"bits", NewDefValBits([]string{"x"}, "x"), DefValKindBits},
		{"oid", NewDefValOID(OID{1}, "1"), DefValKindOID},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testutil.Equal(t, tt.want, tt.dv.Kind(), "Kind()")
		})
	}
}

func TestDefValValue(t *testing.T) {
	dv := NewDefValInt(42, "42")
	v := dv.Value()
	testutil.Equal(t, 42, v.(int64), "Value()")
}

func TestDefValRaw(t *testing.T) {
	dv := NewDefValInt(42, "42")
	testutil.Equal(t, "42", dv.Raw(), "Raw()")
}

func TestDefValAs(t *testing.T) {
	t.Run("int64 match", func(t *testing.T) {
		dv := NewDefValInt(42, "42")
		v, ok := DefValAs[int64](dv)
		testutil.True(t, ok, "DefValAs[int64] should succeed")
		testutil.Equal(t, 42, v, "got")
	})

	t.Run("uint64 match", func(t *testing.T) {
		dv := NewDefValUint(100, "100")
		v, ok := DefValAs[uint64](dv)
		testutil.True(t, ok, "DefValAs[uint64] should succeed")
		testutil.Equal(t, 100, v, "got")
	})

	t.Run("string match", func(t *testing.T) {
		dv := NewDefValString("hello", `"hello"`)
		v, ok := DefValAs[string](dv)
		testutil.True(t, ok, "DefValAs[string] should succeed")
		testutil.Equal(t, "hello", v, "got")
	})

	t.Run("bytes match", func(t *testing.T) {
		dv := NewDefValBytes([]byte{0xAB}, "x")
		v, ok := DefValAs[[]byte](dv)
		testutil.True(t, ok, "DefValAs[[]byte] should succeed")
		testutil.Len(t, v, 1, "DefValAs[[]byte] length")
		testutil.Equal(t, byte(0xAB), v[0], "DefValAs[[]byte] value")
	})

	t.Run("type mismatch", func(t *testing.T) {
		dv := NewDefValInt(42, "42")
		_, ok := DefValAs[string](dv)
		testutil.False(t, ok, "DefValAs[string] on int DefVal should return false")
	})

	t.Run("bits match", func(t *testing.T) {
		dv := NewDefValBits([]string{"a", "b"}, "x")
		v, ok := DefValAs[[]string](dv)
		testutil.True(t, ok, "DefValAs[[]string] should succeed")
		testutil.Len(t, v, 2, "got")
	})

	t.Run("oid match", func(t *testing.T) {
		dv := NewDefValOID(OID{1, 3}, "1.3")
		v, ok := DefValAs[OID](dv)
		testutil.True(t, ok, "DefValAs[OID] should succeed")
		testutil.Equal(t, "1.3", v.String(), "got")
	})
}

func TestDefValStringZeroValue(t *testing.T) {
	var zero DefVal
	testutil.True(t, zero.IsZero(), "zero DefVal should report IsZero() true")
	got := zero.String()
	testutil.Equal(t, "", got, "zero DefVal.String()")
}

func TestRangeString(t *testing.T) {
	tests := []struct {
		name string
		r    Range
		want string
	}{
		{"single value", Range{Min: 5, Max: 5}, "5"},
		{"range", Range{Min: 0, Max: 255}, "0..255"},
		{"negative", Range{Min: -1, Max: 100}, "-1..100"},
		{"zero range", Range{Min: 0, Max: 0}, "0"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.r.String()
			testutil.Equal(t, tt.want, got, "Range.String()")
		})
	}
}

func TestBytesToHex(t *testing.T) {
	tests := []struct {
		name  string
		input []byte
		want  string
	}{
		{"single byte", []byte{0xAB}, "AB"},
		{"multiple", []byte{0xDE, 0xAD}, "DEAD"},
		{"zeros", []byte{0x00, 0x00}, "0000"},
		{"all FF", []byte{0xFF, 0xFF}, "FFFF"},
		{"empty", []byte{}, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := bytesToHex(tt.input)
			testutil.Equal(t, tt.want, got, "bytesToHex()")
		})
	}
}

func TestModuleImportsDeepClone(t *testing.T) {
	m := NewModule("TEST-MIB")
	m.setRawImports([]module.Import{
		module.NewImport("SNMPv2-SMI", "MODULE-IDENTITY", types.Span{}),
		module.NewImport("SNMPv2-SMI", "OBJECT-TYPE", types.Span{}),
	})

	got := m.Imports()
	got[0].Symbols[0] = ImportSymbol{Name: "MUTATED"}

	// Verify rawImports are unaffected (Imports() returns a freshly built slice).
	testutil.Equal(t, "MODULE-IDENTITY", m.rawImports[0].Symbol, "mutating Imports() return value should not affect module internal state")
}

func TestComplianceModulesDeepClone(t *testing.T) {
	c := NewCompliance("testCompliance")
	access := AccessReadOnly
	c.setModules([]ComplianceModule{
		{
			ModuleName:      "IF-MIB",
			MandatoryGroups: []string{"ifGeneralGroup"},
			Groups:          []ComplianceGroup{{Group: "ifGroup", Description: "desc"}},
			Objects: []ComplianceObject{{
				Object:      "ifType",
				Syntax:      &SyntaxConstraints{Sizes: []Range{{Min: 1, Max: 255}}},
				WriteSyntax: &SyntaxConstraints{Ranges: []Range{{Min: 0, Max: 100}}},
				MinAccess:   &access,
				Description: "obj desc",
			}},
		},
	})

	tests := []struct {
		name       string
		mutate     func([]ComplianceModule)
		wasMutated func() bool
	}{
		{
			"MandatoryGroups",
			func(m []ComplianceModule) { m[0].MandatoryGroups[0] = "X" },
			func() bool { return c.modules[0].MandatoryGroups[0] == "X" },
		},
		{
			"Groups",
			func(m []ComplianceModule) { m[0].Groups[0].Group = "X" },
			func() bool { return c.modules[0].Groups[0].Group == "X" },
		},
		{
			"Objects",
			func(m []ComplianceModule) { m[0].Objects[0].Object = "X" },
			func() bool { return c.modules[0].Objects[0].Object == "X" },
		},
		{
			"Syntax",
			func(m []ComplianceModule) { m[0].Objects[0].Syntax.Sizes[0].Min = 999 },
			func() bool { return c.modules[0].Objects[0].Syntax.Sizes[0].Min == 999 },
		},
		{
			"WriteSyntax",
			func(m []ComplianceModule) { m[0].Objects[0].WriteSyntax.Ranges[0].Min = 999 },
			func() bool { return c.modules[0].Objects[0].WriteSyntax.Ranges[0].Min == 999 },
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := c.Modules()
			tt.mutate(got)
			testutil.False(t, tt.wasMutated(), "mutating Modules()[]. should not affect internal state")
		})
	}
}

func TestCapabilitySupportsDeepClone(t *testing.T) {
	cap := NewCapability("testCap")
	cap.setSupports([]CapabilitiesModule{
		{
			ModuleName: "IF-MIB",
			Includes:   []string{"ifGeneralGroup"},
			ObjectVariations: []ObjectVariation{{
				Object:           "ifType",
				Syntax:           &SyntaxConstraints{Enums: []NamedValue{{Label: "e1", Value: 1}}},
				WriteSyntax:      &SyntaxConstraints{Bits: []NamedValue{{Label: "b1", Value: 0}}},
				CreationRequires: []string{"ifName"},
				Description:      "var desc",
			}},
			NotificationVariations: []NotificationVariation{{
				Notification: "linkDown",
				Description:  "notif desc",
			}},
		},
	})

	tests := []struct {
		name       string
		mutate     func([]CapabilitiesModule)
		wasMutated func() bool
	}{
		{
			"Includes",
			func(m []CapabilitiesModule) { m[0].Includes[0] = "X" },
			func() bool { return cap.supports[0].Includes[0] == "X" },
		},
		{
			"ObjectVariations",
			func(m []CapabilitiesModule) { m[0].ObjectVariations[0].Object = "X" },
			func() bool { return cap.supports[0].ObjectVariations[0].Object == "X" },
		},
		{
			"CreationRequires",
			func(m []CapabilitiesModule) { m[0].ObjectVariations[0].CreationRequires[0] = "X" },
			func() bool { return cap.supports[0].ObjectVariations[0].CreationRequires[0] == "X" },
		},
		{
			"Syntax",
			func(m []CapabilitiesModule) { m[0].ObjectVariations[0].Syntax.Enums[0].Label = "X" },
			func() bool { return cap.supports[0].ObjectVariations[0].Syntax.Enums[0].Label == "X" },
		},
		{
			"WriteSyntax",
			func(m []CapabilitiesModule) { m[0].ObjectVariations[0].WriteSyntax.Bits[0].Label = "X" },
			func() bool { return cap.supports[0].ObjectVariations[0].WriteSyntax.Bits[0].Label == "X" },
		},
		{
			"NotificationVariations",
			func(m []CapabilitiesModule) { m[0].NotificationVariations[0].Notification = "X" },
			func() bool { return cap.supports[0].NotificationVariations[0].Notification == "X" },
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := cap.Supports()
			tt.mutate(got)
			testutil.False(t, tt.wasMutated(), "mutating Supports()[]. should not affect internal state")
		})
	}
}

func TestSyntaxConstraintsClone(t *testing.T) {
	orig := &SyntaxConstraints{
		Sizes:  []Range{{Min: 1, Max: 10}},
		Ranges: []Range{{Min: 0, Max: 255}},
		Enums:  []NamedValue{{Label: "up", Value: 1}},
		Bits:   []NamedValue{{Label: "bit0", Value: 0}},
	}

	cloned := orig.clone()

	// Verify cloned values match original
	testutil.SliceEqual(t, orig.Sizes, cloned.Sizes, "cloned Sizes should equal original")
	testutil.SliceEqual(t, orig.Ranges, cloned.Ranges, "cloned Ranges should equal original")

	// Verify mutations to clone don't propagate to original
	tests := []struct {
		name       string
		mutate     func(*SyntaxConstraints)
		wasMutated func() bool
	}{
		{
			"Sizes",
			func(c *SyntaxConstraints) { c.Sizes[0].Min = 999 },
			func() bool { return orig.Sizes[0].Min == 999 },
		},
		{
			"Enums",
			func(c *SyntaxConstraints) { c.Enums[0].Label = "X" },
			func() bool { return orig.Enums[0].Label == "X" },
		},
		{
			"Bits",
			func(c *SyntaxConstraints) { c.Bits[0].Label = "X" },
			func() bool { return orig.Bits[0].Label == "X" },
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := orig.clone()
			tt.mutate(c)
			testutil.False(t, tt.wasMutated(), "mutating cloned  should not affect original")
		})
	}
}

func TestSyntaxConstraintsCloneNil(t *testing.T) {
	var sc *SyntaxConstraints
	testutil.Nil(t, sc.clone(), "clone of nil SyntaxConstraints should return nil")
}
