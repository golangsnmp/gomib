package mib

import "testing"

// --- Integer formatting ---

func TestFormatIntegerDecimal(t *testing.T) {
	for _, tc := range []struct {
		value int64
		want  string
	}{
		{0, "0"},
		{42, "42"},
		{-42, "-42"},
	} {
		got, ok := FormatInteger("d", tc.value, HexUpper)
		if !ok || got != tc.want {
			t.Errorf("FormatInteger(d, %d) = %q, %v; want %q", tc.value, got, ok, tc.want)
		}
	}
}

func TestFormatIntegerDecimalWithPoint(t *testing.T) {
	for _, tc := range []struct {
		hint  string
		value int64
		want  string
	}{
		{"d-2", 1234, "12.34"},
		{"d-2", 5, "0.05"},
		{"d-2", 0, "0.00"},
		{"d-2", 100, "1.00"},
		{"d-2", -1234, "-12.34"},
		{"d-2", -5, "-0.05"},
		{"d-1", 15, "1.5"},
		{"d-3", 12345, "12.345"},
		{"d-5", 123, "0.00123"},
		{"d-0", 42, "42"},
	} {
		got, ok := FormatInteger(tc.hint, tc.value, HexUpper)
		if !ok || got != tc.want {
			t.Errorf("FormatInteger(%s, %d) = %q, %v; want %q", tc.hint, tc.value, got, ok, tc.want)
		}
	}
}

func TestFormatIntegerHex(t *testing.T) {
	for _, tc := range []struct {
		value int64
		want  string
	}{
		{0, "0"},
		{255, "FF"},
		{256, "100"},
		{-255, "-FF"},
	} {
		got, ok := FormatInteger("x", tc.value, HexUpper)
		if !ok || got != tc.want {
			t.Errorf("FormatInteger(x, %d) = %q, %v; want %q", tc.value, got, ok, tc.want)
		}
	}
}

func TestFormatIntegerHexLower(t *testing.T) {
	got, ok := FormatInteger("x", 255, HexLower)
	if !ok || got != "ff" {
		t.Errorf("FormatInteger(x, 255, Lower) = %q, %v; want ff", got, ok)
	}
	got, ok = FormatInteger("x", -255, HexLower)
	if !ok || got != "-ff" {
		t.Errorf("FormatInteger(x, -255, Lower) = %q, %v; want -ff", got, ok)
	}
}

func TestFormatIntegerOctal(t *testing.T) {
	for _, tc := range []struct {
		value int64
		want  string
	}{
		{0, "0"},
		{8, "10"},
		{63, "77"},
		{-8, "-10"},
	} {
		got, ok := FormatInteger("o", tc.value, HexUpper)
		if !ok || got != tc.want {
			t.Errorf("FormatInteger(o, %d) = %q, %v; want %q", tc.value, got, ok, tc.want)
		}
	}
}

func TestFormatIntegerBinary(t *testing.T) {
	for _, tc := range []struct {
		value int64
		want  string
	}{
		{0, "0"},
		{5, "101"},
		{255, "11111111"},
		{-5, "-101"},
	} {
		got, ok := FormatInteger("b", tc.value, HexUpper)
		if !ok || got != tc.want {
			t.Errorf("FormatInteger(b, %d) = %q, %v; want %q", tc.value, got, ok, tc.want)
		}
	}
}

func TestFormatIntegerErrors(t *testing.T) {
	for _, hint := range []string{"", "z", "x1", "o1", "b1", "d-", "d-abc", "dd"} {
		if got, ok := FormatInteger(hint, 0, HexUpper); ok {
			t.Errorf("FormatInteger(%q, 0) = %q, true; want false", hint, got)
		}
	}
}

// --- Integer scaling ---

func TestScaleInteger(t *testing.T) {
	for _, tc := range []struct {
		hint  string
		value int64
		want  float64
		ok    bool
	}{
		{"d", 0, 0, true},
		{"d", 42, 42, true},
		{"d", -42, -42, true},
		{"d-2", 1234, 12.34, true},
		{"d-2", 5, 0.05, true},
		{"d-2", 0, 0, true},
		{"d-2", -1234, -12.34, true},
		{"d-1", 15, 1.5, true},
		{"d-3", 12345, 12.345, true},
		{"d-0", 42, 42, true},
		// Non-decimal hints return false.
		{"x", 255, 0, false},
		{"o", 8, 0, false},
		{"b", 5, 0, false},
		// Errors.
		{"", 0, 0, false},
		{"z", 0, 0, false},
		{"d-", 0, 0, false},
		{"d-abc", 0, 0, false},
		{"dd", 0, 0, false},
	} {
		got, ok := ScaleInteger(tc.hint, tc.value)
		if ok != tc.ok || got != tc.want {
			t.Errorf("ScaleInteger(%q, %d) = %v, %v; want %v, %v", tc.hint, tc.value, got, ok, tc.want, tc.ok)
		}
	}
}

// --- Octet-string formatting ---

func TestFormatOctetsIPv4(t *testing.T) {
	got, ok := FormatOctets("1d.1d.1d.1d", []byte{192, 168, 1, 1}, HexUpper)
	if !ok || got != "192.168.1.1" {
		t.Errorf("got %q, %v; want 192.168.1.1", got, ok)
	}
}

func TestFormatOctetsIPv4WithZone(t *testing.T) {
	got, ok := FormatOctets("1d.1d.1d.1d%4d", []byte{192, 168, 1, 1, 0, 0, 0, 3}, HexUpper)
	if !ok || got != "192.168.1.1%3" {
		t.Errorf("got %q, %v; want 192.168.1.1%%3", got, ok)
	}
}

func TestFormatOctetsMacAddress(t *testing.T) {
	got, ok := FormatOctets("1x:", []byte{0x00, 0x1a, 0x2b, 0x3c, 0x4d, 0x5e}, HexUpper)
	if !ok || got != "00:1A:2B:3C:4D:5E" {
		t.Errorf("got %q, %v; want 00:1A:2B:3C:4D:5E", got, ok)
	}
}

func TestFormatOctetsMacAddressLower(t *testing.T) {
	got, ok := FormatOctets("1x:", []byte{0x00, 0x1a, 0x2b}, HexLower)
	if !ok || got != "00:1a:2b" {
		t.Errorf("got %q, %v; want 00:1a:2b", got, ok)
	}
}

func TestFormatOctetsIPv6(t *testing.T) {
	data := []byte{0x20, 0x01, 0x0d, 0xb8, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0x01}
	got, ok := FormatOctets("2x:2x:2x:2x:2x:2x:2x:2x", data, HexUpper)
	if !ok || got != "2001:0DB8:0000:0000:0000:0000:0000:0001" {
		t.Errorf("got %q", got)
	}
}

func TestFormatOctetsIPv6WithZone(t *testing.T) {
	data := []byte{0xfe, 0x80, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0x01, 0, 0, 0, 0x05}
	got, ok := FormatOctets("2x:2x:2x:2x:2x:2x:2x:2x%4d", data, HexUpper)
	if !ok || got != "FE80:0000:0000:0000:0000:0000:0000:0001%5" {
		t.Errorf("got %q", got)
	}
}

func TestFormatOctetsDisplayString(t *testing.T) {
	got, ok := FormatOctets("255a", []byte("Hello, World!"), HexUpper)
	if !ok || got != "Hello, World!" {
		t.Errorf("got %q", got)
	}
}

func TestFormatOctetsSimpleDecimal(t *testing.T) {
	got, ok := FormatOctets("1d", []byte{42}, HexUpper)
	if !ok || got != "42" {
		t.Errorf("got %q, %v; want 42", got, ok)
	}
}

func TestFormatOctetsMultiByteDecimal(t *testing.T) {
	got, ok := FormatOctets("4d", []byte{0x00, 0x01, 0x00, 0x00}, HexUpper)
	if !ok || got != "65536" {
		t.Errorf("got %q", got)
	}
}

func TestFormatOctetsOctal(t *testing.T) {
	got, ok := FormatOctets("1o", []byte{8}, HexUpper)
	if !ok || got != "10" {
		t.Errorf("got %q", got)
	}
}

func TestFormatOctetsHexDashSeparator(t *testing.T) {
	got, ok := FormatOctets("1x-", []byte{0xaa, 0xbb, 0xcc}, HexUpper)
	if !ok || got != "AA-BB-CC" {
		t.Errorf("got %q", got)
	}
}

func TestFormatOctetsStarRepeat(t *testing.T) {
	got, ok := FormatOctets("*1x:", []byte{3, 0xaa, 0xbb, 0xcc}, HexUpper)
	if !ok || got != "AA:BB:CC" {
		t.Errorf("got %q", got)
	}
}

func TestFormatOctetsStarWithTerminator(t *testing.T) {
	got, ok := FormatOctets("*1d./1d", []byte{3, 10, 20, 30, 40}, HexUpper)
	if !ok || got != "10.20.30/40" {
		t.Errorf("got %q", got)
	}
}

func TestFormatOctetsTrailingSeparatorSuppressed(t *testing.T) {
	got, ok := FormatOctets("1d.", []byte{1, 2, 3}, HexUpper)
	if !ok || got != "1.2.3" {
		t.Errorf("got %q", got)
	}
}

func TestFormatOctetsDateAndTime(t *testing.T) {
	got, ok := FormatOctets("2d-1d-1d,1d:1d:1d.1d", []byte{0x07, 0xE6, 8, 15, 8, 1, 15, 0}, HexUpper)
	if !ok || got != "2022-8-15,8:1:15.0" {
		t.Errorf("got %q", got)
	}
}

func TestFormatOctetsDataShorterThanSpec(t *testing.T) {
	got, ok := FormatOctets("1d.1d.1d.1d", []byte{10, 20}, HexUpper)
	if !ok || got != "10.20" {
		t.Errorf("got %q", got)
	}
}

func TestFormatOctetsUtf8(t *testing.T) {
	got, ok := FormatOctets("10t", []byte("hello"), HexUpper)
	if !ok || got != "hello" {
		t.Errorf("got %q", got)
	}
}

func TestFormatOctetsUUID(t *testing.T) {
	data := []byte{
		0x12, 0x34, 0x56, 0x78, 0xab, 0xcd, 0xef, 0x01,
		0x23, 0x45, 0x00, 0x11, 0x22, 0x33, 0x44, 0x55,
	}
	got, ok := FormatOctets("4x-2x-2x-1x1x-6x", data, HexUpper)
	if !ok || got != "12345678-ABCD-EF01-2345-001122334455" {
		t.Errorf("got %q", got)
	}
}

func TestFormatOctetsIPv4WithPrefix(t *testing.T) {
	got, ok := FormatOctets("1d.1d.1d.1d/1d", []byte{10, 0, 0, 0, 24}, HexUpper)
	if !ok || got != "10.0.0.0/24" {
		t.Errorf("got %q", got)
	}
}

func TestFormatOctetsTwoDigitTake(t *testing.T) {
	got, ok := FormatOctets("10d", []byte{0, 0, 0, 0, 0, 0, 0, 1}, HexUpper)
	if !ok || got != "1" {
		t.Errorf("got %q", got)
	}
}

func TestFormatOctetsZeroPaddedHex(t *testing.T) {
	got, ok := FormatOctets("1x", []byte{0x0f}, HexUpper)
	if !ok || got != "0F" {
		t.Errorf("got %q", got)
	}
}

func TestFormatOctetsSingleByteSeparatorSuppressed(t *testing.T) {
	got, ok := FormatOctets("1d.", []byte{42}, HexUpper)
	if !ok || got != "42" {
		t.Errorf("got %q", got)
	}
}

func TestFormatOctetsImplicitRepetition(t *testing.T) {
	got, ok := FormatOctets("1d.", []byte{1, 2, 3, 4, 5}, HexUpper)
	if !ok || got != "1.2.3.4.5" {
		t.Errorf("got %q", got)
	}
}

func TestFormatOctetsLastSpecRepeats(t *testing.T) {
	got, ok := FormatOctets("1d-1d.", []byte{1, 2, 3, 4, 5, 6}, HexUpper)
	if !ok || got != "1-2.3.4.5.6" {
		t.Errorf("got %q", got)
	}
}

func TestFormatOctetsValidUtf8Preserved(t *testing.T) {
	data := []byte("Hello, 世界!")
	got, ok := FormatOctets("255t", data, HexUpper)
	if !ok || got != "Hello, 世界!" {
		t.Errorf("got %q", got)
	}
}

func TestFormatOctetsUtf8TrailingInvalidDiscarded(t *testing.T) {
	// Valid e-acute (U+00E9 = C3 A9) followed by orphaned continuation byte.
	got, ok := FormatOctets("10t", []byte{0xC3, 0xA9, 0x80}, HexUpper)
	if !ok || got != "\u00E9" {
		t.Errorf("got %q; want e-acute", got)
	}

	// Orphaned lead byte at end of chunk.
	got, ok = FormatOctets("5t", []byte{0x41, 0x42, 0xC3}, HexUpper)
	if !ok || got != "AB" {
		t.Errorf("got %q; want AB", got)
	}

	// All invalid - everything discarded.
	got, ok = FormatOctets("3t", []byte{0x80, 0x80, 0x80}, HexUpper)
	if !ok || got != "" {
		t.Errorf("got %q; want empty", got)
	}
}

func TestFormatOctetsAsciiNonUtf8(t *testing.T) {
	got, ok := FormatOctets("10a", []byte{0x48, 0x69, 0x80, 0xFF, 0x21}, HexUpper)
	if !ok {
		t.Fatal("expected ok")
	}
	if got[:2] != "Hi" || got[len(got)-1] != '!' {
		t.Errorf("got %q; want Hi...!", got)
	}
}

func TestFormatOctetsStarRepeatCountZero(t *testing.T) {
	got, ok := FormatOctets("*1d./1d", []byte{0, 42}, HexUpper)
	if !ok || got != "/42" {
		t.Errorf("got %q; want /42", got)
	}
}

// --- Zero-width specs ---

func TestFormatOctetsZeroWidthBracketPrefix(t *testing.T) {
	got, ok := FormatOctets("0a[1a]1a", []byte{0x41, 0x42}, HexUpper)
	if !ok || got != "[A]B" {
		t.Errorf("got %q; want [A]B", got)
	}
}

func TestFormatOctetsZeroWidthPrefixTrailingSuppressed(t *testing.T) {
	got, ok := FormatOctets("0a[1a", []byte{0x41}, HexUpper)
	if !ok || got != "[A" {
		t.Errorf("got %q; want [A", got)
	}
}

func TestFormatOctetsTransportAddressIPv6Style(t *testing.T) {
	got, ok := FormatOctets("0a[2x]0a:2d", []byte{0x20, 0x01, 0x00, 0x50}, HexUpper)
	if !ok || got != "[2001]:80" {
		t.Errorf("got %q; want [2001]:80", got)
	}
}

func TestFormatOctetsZeroWidthPrefixOnly(t *testing.T) {
	got, ok := FormatOctets("0a<1d-1d-1d", []byte{1, 2, 3}, HexUpper)
	if !ok || got != "<1-2-3" {
		t.Errorf("got %q; want <1-2-3", got)
	}
}

func TestFormatOctetsZeroWidthMidHint(t *testing.T) {
	got, ok := FormatOctets("1d-0a.1d", []byte{10, 20}, HexUpper)
	if !ok || got != "10-.20" {
		t.Errorf("got %q; want 10-.20", got)
	}
}

// --- Error cases ---

func TestFormatOctetsErrors(t *testing.T) {
	for _, tc := range []struct {
		hint string
		data []byte
	}{
		{"", []byte{1, 2, 3}},
		{"1d", nil},
		{"1d", []byte{}},
		{"1z", []byte{1, 2, 3}},
		{"1", []byte{1, 2, 3}},
		{"9d", []byte{1, 0, 0, 0, 0, 0, 0, 0, 0}},
		{"9o", []byte{1, 0, 0, 0, 0, 0, 0, 0, 0}},
		{"0x", []byte{0x41, 0x42}},
		{"0d", []byte{1, 2, 3}},
		{"0o", []byte{8, 9}},
		{"0a.", []byte{1, 2, 3}},
	} {
		if got, ok := FormatOctets(tc.hint, tc.data, HexUpper); ok {
			t.Errorf("FormatOctets(%q, %v) = %q, true; want false", tc.hint, tc.data, got)
		}
	}
}

// --- ParseDisplayHint ---

func TestParseDisplayHintIntegerDecimal(t *testing.T) {
	h := ParseDisplayHint("d")
	if h == nil || !h.IsInteger() || h.IsOctetString() {
		t.Fatal("expected integer hint")
	}
	ih := h.Integer()
	if ih.Format != IntegerDecimal || ih.DecimalPlaces != 0 {
		t.Errorf("got format=%v places=%d", ih.Format, ih.DecimalPlaces)
	}
}

func TestParseDisplayHintIntegerDecimalWithPlaces(t *testing.T) {
	h := ParseDisplayHint("d-2")
	if h == nil {
		t.Fatal("expected non-nil")
	}
	ih := h.Integer()
	if ih.Format != IntegerDecimal || ih.DecimalPlaces != 2 {
		t.Errorf("got format=%v places=%d", ih.Format, ih.DecimalPlaces)
	}
}

func TestParseDisplayHintIntegerHex(t *testing.T) {
	h := ParseDisplayHint("x")
	if h == nil || h.Integer().Format != IntegerHex {
		t.Errorf("expected hex hint")
	}
}

func TestParseDisplayHintIntegerOctal(t *testing.T) {
	h := ParseDisplayHint("o")
	if h == nil || h.Integer().Format != IntegerOctal {
		t.Errorf("expected octal hint")
	}
}

func TestParseDisplayHintIntegerBinary(t *testing.T) {
	h := ParseDisplayHint("b")
	if h == nil || h.Integer().Format != IntegerBinary {
		t.Errorf("expected binary hint")
	}
}

func TestParseDisplayHintIntegerInvalid(t *testing.T) {
	for _, hint := range []string{"", "z", "d-", "d-abc", "dd", "x1"} {
		if h := ParseDisplayHint(hint); h != nil {
			t.Errorf("ParseDisplayHint(%q) = non-nil; want nil", hint)
		}
	}
}

func TestParseDisplayHintOctetDisplayString(t *testing.T) {
	h := ParseDisplayHint("255a")
	if h == nil || !h.IsOctetString() {
		t.Fatal("expected octet string hint")
	}
	os := h.OctetString()
	if len(os.Segments) != 1 {
		t.Fatalf("expected 1 segment, got %d", len(os.Segments))
	}
	s := os.Segments[0]
	if s.Length != 255 || s.Format != OctetAscii || s.Repeat || s.Separator != 0 {
		t.Errorf("unexpected segment: %+v", s)
	}
	if !os.IsText() {
		t.Error("expected IsText true")
	}
}

func TestParseDisplayHintOctetMacAddress(t *testing.T) {
	h := ParseDisplayHint("1x:")
	os := h.OctetString()
	if len(os.Segments) != 1 {
		t.Fatalf("expected 1 segment, got %d", len(os.Segments))
	}
	s := os.Segments[0]
	if s.Length != 1 || s.Format != OctetHex || s.Separator != ':' {
		t.Errorf("unexpected segment: %+v", s)
	}
	if os.IsText() {
		t.Error("expected IsText false")
	}
}

func TestParseDisplayHintOctetDateAndTime(t *testing.T) {
	h := ParseDisplayHint("2d-1d-1d,1d:1d:1d.1d")
	os := h.OctetString()
	if len(os.Segments) != 7 {
		t.Fatalf("expected 7 segments, got %d", len(os.Segments))
	}
	if os.Segments[0].Length != 2 || os.Segments[0].Format != OctetDecimal || os.Segments[0].Separator != '-' {
		t.Errorf("seg 0: %+v", os.Segments[0])
	}
	if os.Segments[3].Separator != ':' {
		t.Errorf("seg 3 sep: %v", os.Segments[3].Separator)
	}
	if os.Segments[6].Separator != 0 {
		t.Errorf("seg 6 sep: %v", os.Segments[6].Separator)
	}
}

func TestParseDisplayHintOctetStarWithTerminator(t *testing.T) {
	h := ParseDisplayHint("*1d./1d")
	os := h.OctetString()
	if len(os.Segments) != 2 {
		t.Fatalf("expected 2 segments, got %d", len(os.Segments))
	}
	if !os.Segments[0].Repeat || os.Segments[0].Separator != '.' || os.Segments[0].Terminator != '/' {
		t.Errorf("seg 0: %+v", os.Segments[0])
	}
	if os.Segments[1].Repeat {
		t.Error("seg 1 should not repeat")
	}
}

func TestParseDisplayHintOctetUtf8(t *testing.T) {
	h := ParseDisplayHint("255t")
	os := h.OctetString()
	if !os.IsText() || os.Segments[0].Format != OctetUtf8 {
		t.Error("expected utf8 text hint")
	}
}

func TestParseDisplayHintOctetIPv4(t *testing.T) {
	h := ParseDisplayHint("1d.1d.1d.1d")
	os := h.OctetString()
	if len(os.Segments) != 4 {
		t.Fatalf("expected 4 segments, got %d", len(os.Segments))
	}
	for i, seg := range os.Segments {
		if seg.Length != 1 || seg.Format != OctetDecimal {
			t.Errorf("seg %d: %+v", i, seg)
		}
		if i < 3 && seg.Separator != '.' {
			t.Errorf("seg %d sep: %v", i, seg.Separator)
		}
		if i == 3 && seg.Separator != 0 {
			t.Errorf("seg 3 sep: %v", seg.Separator)
		}
	}
}

func TestParseDisplayHintOctetUUID(t *testing.T) {
	h := ParseDisplayHint("4x-2x-2x-1x1x-6x")
	os := h.OctetString()
	if len(os.Segments) != 6 {
		t.Fatalf("expected 6 segments, got %d", len(os.Segments))
	}
	if os.Segments[0].Length != 4 || os.Segments[0].Separator != '-' {
		t.Errorf("seg 0: %+v", os.Segments[0])
	}
	if os.Segments[3].Length != 1 || os.Segments[3].Separator != 0 {
		t.Errorf("seg 3: %+v", os.Segments[3])
	}
	if os.Segments[4].Length != 1 || os.Segments[4].Separator != '-' {
		t.Errorf("seg 4: %+v", os.Segments[4])
	}
}

func TestParseDisplayHintOctetZeroWidth(t *testing.T) {
	h := ParseDisplayHint("0a[1a]1a")
	os := h.OctetString()
	if len(os.Segments) != 3 {
		t.Fatalf("expected 3 segments, got %d", len(os.Segments))
	}
	if os.Segments[0].Length != 0 || os.Segments[0].Format != OctetAscii || os.Segments[0].Separator != '[' {
		t.Errorf("seg 0: %+v", os.Segments[0])
	}
}

func TestParseDisplayHintOctetInvalid(t *testing.T) {
	for _, hint := range []string{"1", "1z", "0d", "0x"} {
		if h := ParseDisplayHint(hint); h != nil {
			t.Errorf("ParseDisplayHint(%q) = non-nil; want nil", hint)
		}
	}
}

// --- Validation convenience functions ---

func TestIsValidIntegerHint(t *testing.T) {
	for _, hint := range []string{"d", "d-0", "d-1", "d-2", "d-10", "d-99", "x", "o", "b"} {
		if !IsValidIntegerHint(hint) {
			t.Errorf("IsValidIntegerHint(%q) = false; want true", hint)
		}
	}
	for _, hint := range []string{"", "a", "t", "z", "d-", "d-x", "d--2", "d-abc", "dd", "x1", "D", "X", "1x:", "255a"} {
		if IsValidIntegerHint(hint) {
			t.Errorf("IsValidIntegerHint(%q) = true; want false", hint)
		}
	}
}

func TestIsValidOctetStringHint(t *testing.T) {
	for _, hint := range []string{"255a", "1x:", "1x", "1d", "4d", "2d-1d-1d,1d:1d:1d.1d", "2d-1d-1d,1d:1d:1d.1d,1a1d:1d", "1d.1d.1d.1d", "2x:2x:2x:2x:2x:2x:2x:2x", "4x-2x-2x-1x1x-6x", "*1x:", "*1x:.", "*1d./1d", "0a[1a", "0a[1a]1a", "0a[2x]0a:2d", "255t"} {
		if !IsValidOctetStringHint(hint) {
			t.Errorf("IsValidOctetStringHint(%q) = false; want true", hint)
		}
	}
	for _, hint := range []string{"", "a", "d", "x", "*", "*a", "1", "1z", "1x*", "1d1d0d", "0d", "0x", "0a", "1d0a"} {
		if IsValidOctetStringHint(hint) {
			t.Errorf("IsValidOctetStringHint(%q) = true; want false", hint)
		}
	}
}

// --- IsText classification ---

func TestIsTextClassification(t *testing.T) {
	for _, tc := range []struct {
		hint string
		text bool
	}{
		{"255a", true},
		{"255t", true},
		{"1x:", false},
		{"1d.", false},
		{"2d-1d-1d,1d:1d:1d.1d", false},
		{"0a[2x]0a:2d", false},
	} {
		h := ParseDisplayHint(tc.hint)
		if h == nil {
			t.Fatalf("ParseDisplayHint(%q) = nil", tc.hint)
		}
		if got := h.OctetString().IsText(); got != tc.text {
			t.Errorf("ParseDisplayHint(%q).OctetString().IsText() = %v; want %v", tc.hint, got, tc.text)
		}
	}
}
