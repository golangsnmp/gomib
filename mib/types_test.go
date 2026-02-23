package mib

import (
	"slices"
	"testing"
)

func TestDefValString(t *testing.T) {
	tests := []struct {
		name string
		dv   DefVal
		want string
	}{
		{"int positive", newDefValInt(42, "42"), "42"},
		{"int negative", newDefValInt(-1, "-1"), "-1"},
		{"int zero", newDefValInt(0, "0"), "0"},
		{"uint", newDefValUint(100, "100"), "100"},
		{"uint zero", newDefValUint(0, "0"), "0"},
		{"string", newDefValString("hello", `"hello"`), `"hello"`},
		{"string empty", newDefValString("", `""`), `""`},
		{"string with quotes", newDefValString(`say "hi"`, `"say \"hi\""`), `"say \"hi\""`},
		{"enum label", newDefValEnum("active", "active"), "active"},
		{"bits multiple", newDefValBits([]string{"read", "write"}, "{ read, write }"), "{ read, write }"},
		{"bits empty", newDefValBits([]string{}, "{ }"), "{ }"},
		{"bits single", newDefValBits([]string{"read"}, "{ read }"), "{ read }"},
		{"oid", newDefValOID(OID{0, 0}, "zeroDotZero"), "zeroDotZero"},
		{"bytes empty", newDefValBytes([]byte{}, "''H"), "0"},
		{"bytes 1 byte", newDefValBytes([]byte{0xFF}, "'FF'H"), "255"},
		{"bytes 4 bytes", newDefValBytes([]byte{0xDE, 0xAD, 0xBE, 0xEF}, "'DEADBEEF'H"), "3735928559"},
		{"bytes 8 bytes", newDefValBytes([]byte{0, 0, 0, 0, 0, 0, 0, 1}, "'0000000000000001'H"), "1"},
		{"bytes >8 bytes", newDefValBytes([]byte{0xAB, 0xCD, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01}, "x"), "0xABCD00000000000001"},
		{"bytes all zero >8", newDefValBytes(make([]byte, 16), "x"), "0x00000000000000000000000000000000"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.dv.String()
			if got != tt.want {
				t.Errorf("DefVal.String() = %q, want %q", got, tt.want)
			}
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
			if got != tt.want {
				t.Errorf("DefValKind(%d).String() = %q, want %q", int(tt.kind), got, tt.want)
			}
		})
	}
}

func TestDefValIsZero(t *testing.T) {
	var zero DefVal
	if !zero.IsZero() {
		t.Error("zero DefVal should report IsZero() true")
	}
	if zero.Kind() != DefValKindUnset {
		t.Errorf("zero DefVal Kind() = %v, want DefValKindUnset", zero.Kind())
	}

	nonZero := newDefValInt(0, "0")
	if nonZero.IsZero() {
		t.Error("newDefValInt(0) should not be IsZero (value is set, just happens to be 0)")
	}
}

func TestDefValKind(t *testing.T) {
	tests := []struct {
		name string
		dv   DefVal
		want DefValKind
	}{
		{"int", newDefValInt(1, "1"), DefValKindInt},
		{"uint", newDefValUint(1, "1"), DefValKindUint},
		{"string", newDefValString("x", "x"), DefValKindString},
		{"bytes", newDefValBytes([]byte{1}, "x"), DefValKindBytes},
		{"enum", newDefValEnum("x", "x"), DefValKindEnum},
		{"bits", newDefValBits([]string{"x"}, "x"), DefValKindBits},
		{"oid", newDefValOID(OID{1}, "1"), DefValKindOID},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.dv.Kind() != tt.want {
				t.Errorf("Kind() = %v, want %v", tt.dv.Kind(), tt.want)
			}
		})
	}
}

func TestDefValValue(t *testing.T) {
	dv := newDefValInt(42, "42")
	v := dv.Value()
	if v.(int64) != 42 {
		t.Errorf("Value() = %v, want 42", v)
	}
}

func TestDefValRaw(t *testing.T) {
	dv := newDefValInt(42, "42")
	if dv.Raw() != "42" {
		t.Errorf("Raw() = %q, want %q", dv.Raw(), "42")
	}
}

func TestDefValAs(t *testing.T) {
	t.Run("int64 match", func(t *testing.T) {
		dv := newDefValInt(42, "42")
		v, ok := DefValAs[int64](dv)
		if !ok {
			t.Fatal("DefValAs[int64] should succeed")
		}
		if v != 42 {
			t.Errorf("got %d, want 42", v)
		}
	})

	t.Run("uint64 match", func(t *testing.T) {
		dv := newDefValUint(100, "100")
		v, ok := DefValAs[uint64](dv)
		if !ok {
			t.Fatal("DefValAs[uint64] should succeed")
		}
		if v != 100 {
			t.Errorf("got %d, want 100", v)
		}
	})

	t.Run("string match", func(t *testing.T) {
		dv := newDefValString("hello", `"hello"`)
		v, ok := DefValAs[string](dv)
		if !ok {
			t.Fatal("DefValAs[string] should succeed")
		}
		if v != "hello" {
			t.Errorf("got %q, want %q", v, "hello")
		}
	})

	t.Run("bytes match", func(t *testing.T) {
		dv := newDefValBytes([]byte{0xAB}, "x")
		v, ok := DefValAs[[]byte](dv)
		if !ok {
			t.Fatal("DefValAs[[]byte] should succeed")
		}
		if len(v) != 1 || v[0] != 0xAB {
			t.Errorf("got %x, want [AB]", v)
		}
	})

	t.Run("type mismatch", func(t *testing.T) {
		dv := newDefValInt(42, "42")
		_, ok := DefValAs[string](dv)
		if ok {
			t.Error("DefValAs[string] on int DefVal should return false")
		}
	})

	t.Run("bits match", func(t *testing.T) {
		dv := newDefValBits([]string{"a", "b"}, "x")
		v, ok := DefValAs[[]string](dv)
		if !ok {
			t.Fatal("DefValAs[[]string] should succeed")
		}
		if len(v) != 2 {
			t.Errorf("got %v, want [a b]", v)
		}
	})

	t.Run("oid match", func(t *testing.T) {
		dv := newDefValOID(OID{1, 3}, "1.3")
		v, ok := DefValAs[OID](dv)
		if !ok {
			t.Fatal("DefValAs[OID] should succeed")
		}
		if v.String() != "1.3" {
			t.Errorf("got %v, want 1.3", v)
		}
	})
}

func TestDefValStringZeroValue(t *testing.T) {
	var zero DefVal
	if !zero.IsZero() {
		t.Fatal("zero DefVal should report IsZero() true")
	}
	got := zero.String()
	if got != "" {
		t.Errorf("zero DefVal.String() = %q, want %q", got, "")
	}
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
			if got != tt.want {
				t.Errorf("Range.String() = %q, want %q", got, tt.want)
			}
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
			if got != tt.want {
				t.Errorf("bytesToHex(%x) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestModuleImportsDeepClone(t *testing.T) {
	m := newModule("TEST-MIB")
	m.setImports([]Import{
		{Module: "SNMPv2-SMI", Symbols: []string{"MODULE-IDENTITY", "OBJECT-TYPE"}},
	})

	got := m.Imports()
	got[0].Symbols[0] = "MUTATED"

	if m.imports[0].Symbols[0] == "MUTATED" {
		t.Error("mutating Imports() return value should not affect module internal state")
	}
}

func TestComplianceModulesDeepClone(t *testing.T) {
	c := newCompliance("testCompliance")
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

	got := c.Modules()

	// Mutate the returned MandatoryGroups
	got[0].MandatoryGroups[0] = "MUTATED"
	if c.modules[0].MandatoryGroups[0] == "MUTATED" {
		t.Error("mutating Modules()[].MandatoryGroups should not affect internal state")
	}

	// Mutate the returned Groups
	got[0].Groups[0].Group = "MUTATED"
	if c.modules[0].Groups[0].Group == "MUTATED" {
		t.Error("mutating Modules()[].Groups should not affect internal state")
	}

	// Mutate the returned Objects
	got[0].Objects[0].Object = "MUTATED"
	if c.modules[0].Objects[0].Object == "MUTATED" {
		t.Error("mutating Modules()[].Objects should not affect internal state")
	}

	// Mutate the Syntax constraints through the returned value
	got[0].Objects[0].Syntax.Sizes[0].Min = 999
	if c.modules[0].Objects[0].Syntax.Sizes[0].Min == 999 {
		t.Error("mutating Modules()[].Objects[].Syntax should not affect internal state")
	}

	// Mutate the WriteSyntax constraints
	got[0].Objects[0].WriteSyntax.Ranges[0].Min = 999
	if c.modules[0].Objects[0].WriteSyntax.Ranges[0].Min == 999 {
		t.Error("mutating Modules()[].Objects[].WriteSyntax should not affect internal state")
	}
}

func TestCapabilitySupportsDeepClone(t *testing.T) {
	cap := newCapability("testCap")
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

	got := cap.Supports()

	// Mutate Includes
	got[0].Includes[0] = "MUTATED"
	if cap.supports[0].Includes[0] == "MUTATED" {
		t.Error("mutating Supports()[].Includes should not affect internal state")
	}

	// Mutate ObjectVariations
	got[0].ObjectVariations[0].Object = "MUTATED"
	if cap.supports[0].ObjectVariations[0].Object == "MUTATED" {
		t.Error("mutating Supports()[].ObjectVariations should not affect internal state")
	}

	// Mutate CreationRequires
	got[0].ObjectVariations[0].CreationRequires[0] = "MUTATED"
	if cap.supports[0].ObjectVariations[0].CreationRequires[0] == "MUTATED" {
		t.Error("mutating Supports()[].ObjectVariations[].CreationRequires should not affect internal state")
	}

	// Mutate Syntax through returned value
	got[0].ObjectVariations[0].Syntax.Enums[0].Label = "MUTATED"
	if cap.supports[0].ObjectVariations[0].Syntax.Enums[0].Label == "MUTATED" {
		t.Error("mutating Supports()[].ObjectVariations[].Syntax should not affect internal state")
	}

	// Mutate WriteSyntax through returned value
	got[0].ObjectVariations[0].WriteSyntax.Bits[0].Label = "MUTATED"
	if cap.supports[0].ObjectVariations[0].WriteSyntax.Bits[0].Label == "MUTATED" {
		t.Error("mutating Supports()[].ObjectVariations[].WriteSyntax should not affect internal state")
	}

	// Mutate NotificationVariations
	got[0].NotificationVariations[0].Notification = "MUTATED"
	if cap.supports[0].NotificationVariations[0].Notification == "MUTATED" {
		t.Error("mutating Supports()[].NotificationVariations should not affect internal state")
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

	// Verify values match
	if !slices.Equal(cloned.Sizes, orig.Sizes) {
		t.Error("cloned Sizes should equal original")
	}
	if !slices.Equal(cloned.Ranges, orig.Ranges) {
		t.Error("cloned Ranges should equal original")
	}

	// Mutate clone, verify original unchanged
	cloned.Sizes[0].Min = 999
	if orig.Sizes[0].Min == 999 {
		t.Error("mutating cloned Sizes should not affect original")
	}

	cloned.Enums[0].Label = "MUTATED"
	if orig.Enums[0].Label == "MUTATED" {
		t.Error("mutating cloned Enums should not affect original")
	}

	cloned.Bits[0].Label = "MUTATED"
	if orig.Bits[0].Label == "MUTATED" {
		t.Error("mutating cloned Bits should not affect original")
	}
}

func TestSyntaxConstraintsCloneNil(t *testing.T) {
	var sc *SyntaxConstraints
	if sc.clone() != nil {
		t.Error("clone of nil SyntaxConstraints should return nil")
	}
}
