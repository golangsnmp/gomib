package mib

import "testing"

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

func TestDefValIsZero(t *testing.T) {
	var zero DefVal
	if !zero.IsZero() {
		t.Error("zero DefVal should report IsZero() true")
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
