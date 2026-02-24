package gomib

import (
	"testing"

	"github.com/golangsnmp/gomib/internal/testutil"
)

func TestNormalizeTypeName(t *testing.T) {
	tests := []struct {
		input, want string
	}{
		// INTEGER / Integer32 variants
		{"INTEGER", "Integer32"},
		{"Integer32", "Integer32"},
		// Counter variants
		{"COUNTER", "Counter32"},
		{"Counter", "Counter32"},
		{"Counter32", "Counter32"},
		// Gauge variants
		{"GAUGE", "Gauge32"},
		{"Gauge", "Gauge32"},
		{"Gauge32", "Gauge32"},
		// Unsigned32 variants
		{"UNSIGNED32", "Unsigned32"},
		{"Unsigned32", "Unsigned32"},
		{"UInteger32", "Unsigned32"},
		// TimeTicks
		{"TIMETICKS", "TimeTicks"},
		{"TimeTicks", "TimeTicks"},
		// IpAddress
		{"IPADDR", "IpAddress"},
		{"IpAddress", "IpAddress"},
		// OCTET STRING
		{"OCTETSTR", "OCTET STRING"},
		{"OCTET STRING", "OCTET STRING"},
		{"OctetString", "OCTET STRING"},
		// OBJECT IDENTIFIER
		{"OBJID", "OBJECT IDENTIFIER"},
		{"OBJECT IDENTIFIER", "OBJECT IDENTIFIER"},
		{"ObjectIdentifier", "OBJECT IDENTIFIER"},
		// Counter64
		{"COUNTER64", "Counter64"},
		{"Counter64", "Counter64"},
		// BITS
		{"BITS", "BITS"},
		{"BITSTRING", "BITS"},
		// Opaque
		{"OPAQUE", "Opaque"},
		{"Opaque", "Opaque"},
		// Unknown type passes through unchanged
		{"NetworkAddress", "NetworkAddress"},
		{"SomeCustomType", "SomeCustomType"},
	}
	for _, tt := range tests {
		got := normalizeTypeName(tt.input)
		if got != tt.want {
			t.Errorf("normalizeTypeName(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestTypesEquivalent(t *testing.T) {
	tests := []struct {
		a, b string
		want bool
	}{
		// Exact match
		{"Integer32", "Integer32", true},
		// Normalized equivalence
		{"INTEGER", "Integer32", true},
		{"COUNTER", "Counter32", true},
		{"OCTETSTR", "OCTET STRING", true},
		{"OBJID", "OBJECT IDENTIFIER", true},
		{"BITSTRING", "BITS", true},
		// Non-equivalent
		{"Integer32", "Counter32", false},
		{"OCTET STRING", "Integer32", false},
		// Unknown types only match themselves
		{"CustomType", "CustomType", true},
		{"CustomType", "OtherType", false},
	}
	for _, tt := range tests {
		got := typesEquivalent(tt.a, tt.b)
		if got != tt.want {
			t.Errorf("typesEquivalent(%q, %q) = %v, want %v", tt.a, tt.b, got, tt.want)
		}
	}
}

func TestSignedEquiv(t *testing.T) {
	tests := []struct {
		a, b int64
		want bool
	}{
		{0, 0, true},
		{100, 100, true},
		{-1, -1, true},
		// Unsigned 4294967295 (0xFFFFFFFF) wraps to signed -1
		{4294967295, -1, true},
		{-1, 4294967295, true},
		// Unsigned 2147483648 (0x80000000) wraps to signed -2147483648
		{2147483648, -2147483648, true},
		{-2147483648, 2147483648, true},
		// Not equivalent
		{1, 2, false},
		{0, -1, false},
		{100, -100, false},
		// Values outside 32-bit range are not equivalent
		{1 << 33, -(1 << 33) + (1 << 32), false},
	}
	for _, tt := range tests {
		got := signedEquiv(tt.a, tt.b)
		if got != tt.want {
			t.Errorf("signedEquiv(%d, %d) = %v, want %v", tt.a, tt.b, got, tt.want)
		}
	}
}

func TestRangesEquivalent(t *testing.T) {
	r := func(lo, hi int64) testutil.RangeInfo {
		return testutil.RangeInfo{Low: lo, High: hi}
	}

	tests := []struct {
		name string
		a, b []testutil.RangeInfo
		want bool
	}{
		{"both nil", nil, nil, true},
		{"both empty", []testutil.RangeInfo{}, []testutil.RangeInfo{}, true},
		{"identical single", []testutil.RangeInfo{r(0, 255)}, []testutil.RangeInfo{r(0, 255)}, true},
		{"different order", []testutil.RangeInfo{r(0, 255), r(1000, 2000)}, []testutil.RangeInfo{r(1000, 2000), r(0, 255)}, true},
		{"signed/unsigned equiv", []testutil.RangeInfo{r(0, 4294967295)}, []testutil.RangeInfo{r(0, -1)}, true},
		{"length mismatch", []testutil.RangeInfo{r(0, 255)}, []testutil.RangeInfo{r(0, 255), r(1, 2)}, false},
		{"value mismatch", []testutil.RangeInfo{r(0, 255)}, []testutil.RangeInfo{r(0, 256)}, false},
		{"nil vs non-empty", nil, []testutil.RangeInfo{r(0, 1)}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := rangesEquivalent(tt.a, tt.b)
			if got != tt.want {
				t.Errorf("rangesEquivalent(%v, %v) = %v, want %v", tt.a, tt.b, got, tt.want)
			}
		})
	}
}

func TestEnumsEquivalent(t *testing.T) {
	tests := []struct {
		name string
		a, b map[int]string
		want bool
	}{
		{"both nil", nil, nil, true},
		{"both empty", map[int]string{}, map[int]string{}, true},
		{"nil vs empty", nil, map[int]string{}, true},
		{"identical", map[int]string{1: "up", 2: "down"}, map[int]string{1: "up", 2: "down"}, true},
		{"length mismatch", map[int]string{1: "up"}, map[int]string{1: "up", 2: "down"}, false},
		{"value mismatch", map[int]string{1: "up"}, map[int]string{1: "down"}, false},
		{"key mismatch", map[int]string{1: "up"}, map[int]string{2: "up"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := enumsEquivalent(tt.a, tt.b)
			if got != tt.want {
				t.Errorf("enumsEquivalent = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHintsEquivalent(t *testing.T) {
	tests := []struct {
		a, b string
		want bool
	}{
		{"255a", "255a", true},
		{"255A", "255a", true},
		{" 255a ", "255a", true},
		{"1x:", "1X:", true},
		{"255a", "128a", false},
		{"", "", true},
	}
	for _, tt := range tests {
		got := hintsEquivalent(tt.a, tt.b)
		if got != tt.want {
			t.Errorf("hintsEquivalent(%q, %q) = %v, want %v", tt.a, tt.b, got, tt.want)
		}
	}
}

func TestIndexesEquivalent(t *testing.T) {
	idx := func(name string, implied bool) testutil.IndexInfo {
		return testutil.IndexInfo{Name: name, Implied: implied}
	}

	tests := []struct {
		name string
		a, b []testutil.IndexInfo
		want bool
	}{
		{"both nil", nil, nil, true},
		{"identical", []testutil.IndexInfo{idx("ifIndex", false)}, []testutil.IndexInfo{idx("ifIndex", false)}, true},
		{"implied mismatch", []testutil.IndexInfo{idx("ifIndex", true)}, []testutil.IndexInfo{idx("ifIndex", false)}, false},
		{"name mismatch", []testutil.IndexInfo{idx("ifIndex", false)}, []testutil.IndexInfo{idx("ipIndex", false)}, false},
		{"length mismatch", []testutil.IndexInfo{idx("a", false)}, []testutil.IndexInfo{idx("a", false), idx("b", false)}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := indexesEquivalent(tt.a, tt.b)
			if got != tt.want {
				t.Errorf("indexesEquivalent = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestVarbindsEquivalent(t *testing.T) {
	tests := []struct {
		name string
		a, b []string
		want bool
	}{
		{"both nil", nil, nil, true},
		{"identical", []string{"a", "b"}, []string{"a", "b"}, true},
		{"value mismatch", []string{"a"}, []string{"b"}, false},
		{"length mismatch", []string{"a"}, []string{"a", "b"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := varbindsEquivalent(tt.a, tt.b)
			if got != tt.want {
				t.Errorf("varbindsEquivalent = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAccessEquivalent(t *testing.T) {
	tests := []struct {
		a, b  string
		smiv1 bool
		want  bool
	}{
		// Exact match always works
		{"read-only", "read-only", false, true},
		{"read-write", "read-write", false, true},
		// SMIv2: read-write and read-create are distinct
		{"read-write", "read-create", false, false},
		{"read-create", "read-write", false, false},
		// SMIv1: read-write and read-create are equivalent
		{"read-write", "read-create", true, true},
		{"read-create", "read-write", true, true},
		// Unrelated values
		{"read-only", "not-accessible", false, false},
		{"read-only", "not-accessible", true, false},
	}
	for _, tt := range tests {
		got := accessEquivalent(tt.a, tt.b, tt.smiv1)
		if got != tt.want {
			t.Errorf("accessEquivalent(%q, %q, smiv1=%v) = %v, want %v", tt.a, tt.b, tt.smiv1, got, tt.want)
		}
	}
}

func TestStatusEquivalent(t *testing.T) {
	tests := []struct {
		a, b string
		want bool
	}{
		{"current", "current", true},
		{"mandatory", "mandatory", true},
		// SMIv1 mandatory = SMIv2 current
		{"mandatory", "current", true},
		{"current", "mandatory", true},
		// Non-equivalent
		{"deprecated", "current", false},
		{"obsolete", "mandatory", false},
	}
	for _, tt := range tests {
		got := statusEquivalent(tt.a, tt.b)
		if got != tt.want {
			t.Errorf("statusEquivalent(%q, %q) = %v, want %v", tt.a, tt.b, got, tt.want)
		}
	}
}

func TestIsHexZeros(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"0x00000000", true},
		{"0X00000000", true},
		{"0x0", true},
		{"0x00", true},
		{"0x0001", false},
		{"0xFF", false},
		{"0x", false},    // no digits after prefix
		{"00000", false}, // no hex prefix
		{"", false},
	}
	for _, tt := range tests {
		got := isHexZeros(tt.input)
		if got != tt.want {
			t.Errorf("isHexZeros(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestIsHexAllOnes(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"0xFFFFFFFF", true},
		{"0xffffffff", true},
		{"0xFF", true},
		{"0XFF", true},
		{"0xFFFE", false},
		{"0x00", false},
		{"0x", false},
		{"FFFF", false},
		{"", false},
	}
	for _, tt := range tests {
		got := isHexAllOnes(tt.input)
		if got != tt.want {
			t.Errorf("isHexAllOnes(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestDefvalEquivalent(t *testing.T) {
	tests := []struct {
		name string
		a, b string
		want bool
	}{
		// Exact match
		{"exact", "42", "42", true},
		// Quote stripping
		{"quoted", `"hello"`, "hello", true},
		{"single-quoted", "'hello'", "hello", true},
		// Whitespace tolerance
		{"whitespace", " 42 ", "42", true},
		// Hex zeros vs "0"
		{"hex-zeros-vs-0", "0x00000000", "0", true},
		{"0-vs-hex-zeros", "0", "0x00000000", true},
		{"hex-zeros-upper", "0X00000000", "0", true},
		// Hex all-ones vs "-1"
		{"hex-ones-vs-neg1", "0xFFFFFFFF", "-1", true},
		{"neg1-vs-hex-ones", "-1", "0xFFFFFFFF", true},
		{"hex-ones-lower", "0xffffffff", "-1", true},
		// Non-equivalent
		{"different", "42", "43", false},
		{"hex-non-zero", "0xFF00", "0", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := defvalEquivalent(tt.a, tt.b)
			if got != tt.want {
				t.Errorf("defvalEquivalent(%q, %q) = %v, want %v", tt.a, tt.b, got, tt.want)
			}
		})
	}
}

func TestReferenceEquivalent(t *testing.T) {
	tests := []struct {
		a, b string
		want bool
	}{
		{"RFC 2863", "RFC 2863", true},
		{"RFC  2863", "RFC 2863", true},   // extra space
		{"RFC\t2863", "RFC 2863", true},   // tab
		{" RFC  2863 ", "RFC 2863", true}, // leading/trailing
		{"RFC 2863", "RFC 1234", false},
	}
	for _, tt := range tests {
		got := referenceEquivalent(tt.a, tt.b)
		if got != tt.want {
			t.Errorf("referenceEquivalent(%q, %q) = %v, want %v", tt.a, tt.b, got, tt.want)
		}
	}
}
