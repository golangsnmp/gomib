package mib

import (
	"testing"
)

func TestDefValInt(t *testing.T) {
	dv := NewDefValInt(42, "42")

	if dv.Kind() != DefValKindInt {
		t.Errorf("Kind = %v, want DefValKindInt", dv.Kind())
	}
	if dv.String() != "42" {
		t.Errorf("String() = %q, want %q", dv.String(), "42")
	}
	if dv.Raw() != "42" {
		t.Errorf("Raw() = %q, want %q", dv.Raw(), "42")
	}
	if v, ok := DefValAs[int64](dv); !ok || v != 42 {
		t.Errorf("DefValAs[int64] = %v, %v, want 42, true", v, ok)
	}
	if dv.IsZero() {
		t.Error("IsZero() = true, want false")
	}
}

func TestDefValUint(t *testing.T) {
	dv := NewDefValUint(4294967295, "4294967295")

	if dv.Kind() != DefValKindUint {
		t.Errorf("Kind = %v, want DefValKindUint", dv.Kind())
	}
	if dv.String() != "4294967295" {
		t.Errorf("String() = %q, want %q", dv.String(), "4294967295")
	}
	if v, ok := DefValAs[uint64](dv); !ok || v != 4294967295 {
		t.Errorf("DefValAs[uint64] = %v, %v, want 4294967295, true", v, ok)
	}
}

func TestDefValString(t *testing.T) {
	dv := NewDefValString("public", `"public"`)

	if dv.Kind() != DefValKindString {
		t.Errorf("Kind = %v, want DefValKindString", dv.Kind())
	}
	if dv.String() != `"public"` {
		t.Errorf("String() = %q, want %q", dv.String(), `"public"`)
	}
	if dv.Raw() != `"public"` {
		t.Errorf("Raw() = %q, want %q", dv.Raw(), `"public"`)
	}
	if v, ok := DefValAs[string](dv); !ok || v != "public" {
		t.Errorf("DefValAs[string] = %v, %v, want public, true", v, ok)
	}
}

func TestDefValBytes(t *testing.T) {
	tests := []struct {
		name       string
		bytes      []byte
		raw        string
		wantString string
	}{
		{
			name:       "empty hex string",
			bytes:      []byte{},
			raw:        "''H",
			wantString: "0",
		},
		{
			name:       "zero bytes",
			bytes:      []byte{0, 0, 0, 0},
			raw:        "'00000000'H",
			wantString: "0",
		},
		{
			name:       "single byte",
			bytes:      []byte{0xFF},
			raw:        "'FF'H",
			wantString: "255",
		},
		{
			name:       "two bytes",
			bytes:      []byte{0x01, 0x00},
			raw:        "'0100'H",
			wantString: "256",
		},
		{
			name:       "four bytes max uint32",
			bytes:      []byte{0xFF, 0xFF, 0xFF, 0xFF},
			raw:        "'FFFFFFFF'H",
			wantString: "4294967295",
		},
		{
			name:       "eight bytes",
			bytes:      []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01, 0x00},
			raw:        "'0000000000000100'H",
			wantString: "256",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dv := NewDefValBytes(tt.bytes, tt.raw)

			if dv.Kind() != DefValKindBytes {
				t.Errorf("Kind = %v, want DefValKindBytes", dv.Kind())
			}
			if dv.String() != tt.wantString {
				t.Errorf("String() = %q, want %q", dv.String(), tt.wantString)
			}
			if dv.Raw() != tt.raw {
				t.Errorf("Raw() = %q, want %q", dv.Raw(), tt.raw)
			}
			if v, ok := DefValAs[[]byte](dv); !ok {
				t.Error("DefValAs[[]byte] failed")
			} else if len(v) != len(tt.bytes) {
				t.Errorf("DefValAs[[]byte] len = %d, want %d", len(v), len(tt.bytes))
			}
		})
	}
}

func TestDefValBytesLarge(t *testing.T) {
	// More than 8 bytes should show as hex
	bytes := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09}
	dv := NewDefValBytes(bytes, "'010203040506070809'H")

	if dv.String() != "0x010203040506070809" {
		t.Errorf("String() = %q, want %q", dv.String(), "0x010203040506070809")
	}
}

func TestDefValEnum(t *testing.T) {
	dv := NewDefValEnum("enabled", "enabled")

	if dv.Kind() != DefValKindEnum {
		t.Errorf("Kind = %v, want DefValKindEnum", dv.Kind())
	}
	if dv.String() != "enabled" {
		t.Errorf("String() = %q, want %q", dv.String(), "enabled")
	}
	if v, ok := DefValAs[string](dv); !ok || v != "enabled" {
		t.Errorf("DefValAs[string] = %v, %v, want enabled, true", v, ok)
	}
}

func TestDefValBits(t *testing.T) {
	tests := []struct {
		name       string
		labels     []string
		wantString string
	}{
		{
			name:       "empty",
			labels:     []string{},
			wantString: "{ }",
		},
		{
			name:       "single",
			labels:     []string{"flag1"},
			wantString: "{ flag1 }",
		},
		{
			name:       "multiple",
			labels:     []string{"flag1", "flag2", "flag3"},
			wantString: "{ flag1, flag2, flag3 }",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dv := NewDefValBits(tt.labels, tt.wantString)

			if dv.Kind() != DefValKindBits {
				t.Errorf("Kind = %v, want DefValKindBits", dv.Kind())
			}
			if dv.String() != tt.wantString {
				t.Errorf("String() = %q, want %q", dv.String(), tt.wantString)
			}
		})
	}
}

func TestDefValOID(t *testing.T) {
	oid := Oid{1, 3, 6, 1, 2, 1}
	dv := NewDefValOID(oid, "sysDescr")

	if dv.Kind() != DefValKindOID {
		t.Errorf("Kind = %v, want DefValKindOID", dv.Kind())
	}
	if dv.String() != "1.3.6.1.2.1" {
		t.Errorf("String() = %q, want %q", dv.String(), "1.3.6.1.2.1")
	}
	if dv.Raw() != "sysDescr" {
		t.Errorf("Raw() = %q, want %q", dv.Raw(), "sysDescr")
	}
}

func TestDefValIsZero(t *testing.T) {
	var zero DefVal
	if !zero.IsZero() {
		t.Error("zero value IsZero() = false, want true")
	}

	nonZero := NewDefValInt(0, "0")
	if nonZero.IsZero() {
		t.Error("NewDefValInt(0) IsZero() = true, want false")
	}
}

func TestDefValAsWrongType(t *testing.T) {
	dv := NewDefValInt(42, "42")

	if _, ok := DefValAs[string](dv); ok {
		t.Error("DefValAs[string] on int should return false")
	}
	if _, ok := DefValAs[[]byte](dv); ok {
		t.Error("DefValAs[[]byte] on int should return false")
	}
}
