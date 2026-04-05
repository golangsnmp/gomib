package model

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"unicode/utf8"
)

// HexCase controls the case of hexadecimal digits in display hint output.
type HexCase int

const (
	// HexUpper uses uppercase hex digits (A-F), matching RFC 3419 examples.
	HexUpper HexCase = iota
	// HexLower uses lowercase hex digits (a-f).
	HexLower
)

// DisplayHint is a parsed and validated RFC 2579 DISPLAY-HINT.
//
// Integer hints and octet-string hints are syntactically distinct: integer
// hints start with a format letter (d, x, o, b), while octet-string hints
// start with a digit or *.
type DisplayHint struct {
	integer     *IntegerHint
	octetString *OctetStringHint
}

// IsInteger reports whether this is an integer display hint.
func (h *DisplayHint) IsInteger() bool { return h != nil && h.integer != nil }

// IsOctetString reports whether this is an octet-string display hint.
func (h *DisplayHint) IsOctetString() bool { return h != nil && h.octetString != nil }

// Integer returns the integer hint, or nil if this is not an integer hint.
func (h *DisplayHint) Integer() *IntegerHint {
	if h == nil {
		return nil
	}
	return h.integer
}

// OctetString returns the octet-string hint, or nil.
func (h *DisplayHint) OctetString() *OctetStringHint {
	if h == nil {
		return nil
	}
	return h.octetString
}

// IntegerHint is a parsed integer display hint.
type IntegerHint struct {
	Format        IntegerFormat
	DecimalPlaces int
}

// IntegerFormat is the format character for an integer display hint.
type IntegerFormat int

const (
	IntegerDecimal IntegerFormat = iota
	IntegerHex
	IntegerOctal
	IntegerBinary
)

// String returns the format character as a string.
func (f IntegerFormat) String() string {
	switch f {
	case IntegerDecimal:
		return "d"
	case IntegerHex:
		return "x"
	case IntegerOctal:
		return "o"
	case IntegerBinary:
		return "b"
	default:
		return "?"
	}
}

// OctetStringHint is a parsed octet-string display hint.
type OctetStringHint struct {
	Segments []OctetSegment
}

// IsText reports whether every segment uses a text format (a or t).
// When true, the hint produces literal character data (like DisplayString's
// "255a"). When false, the hint produces structured formatted output (like
// MacAddress's "1x:").
func (h *OctetStringHint) IsText() bool {
	for i := range h.Segments {
		if h.Segments[i].Format != OctetAscii && h.Segments[i].Format != OctetUtf8 {
			return false
		}
	}
	return true
}

// OctetSegment is one format segment within an octet-string display hint.
type OctetSegment struct {
	Repeat     bool
	Length     int
	Format     OctetFormat
	Separator  byte // 0 means no separator
	Terminator byte // 0 means no terminator
}

// OctetFormat is the format character for an octet-string segment.
type OctetFormat int

const (
	OctetDecimal OctetFormat = iota
	OctetHex
	OctetOctal
	OctetAscii
	OctetUtf8
)

// String returns the format character as a string.
func (f OctetFormat) String() string {
	switch f {
	case OctetDecimal:
		return "d"
	case OctetHex:
		return "x"
	case OctetOctal:
		return "o"
	case OctetAscii:
		return "a"
	case OctetUtf8:
		return "t"
	default:
		return "?"
	}
}

// ParseDisplayHint parses and validates a display hint string per RFC 2579
// Section 3.1. Returns nil if the hint is empty or malformed.
func ParseDisplayHint(hint string) *DisplayHint {
	if hint == "" {
		return nil
	}
	switch hint[0] {
	case 'd', 'x', 'o', 'b':
		if ih := parseIntegerHint(hint); ih != nil {
			return &DisplayHint{integer: ih}
		}
		return nil
	default:
		if hint[0] >= '0' && hint[0] <= '9' || hint[0] == '*' {
			if oh := parseOctetStringHint(hint); oh != nil {
				return &DisplayHint{octetString: oh}
			}
		}
		return nil
	}
}

func parseIntegerHint(hint string) *IntegerHint {
	switch hint[0] {
	case 'x':
		if len(hint) == 1 {
			return &IntegerHint{Format: IntegerHex}
		}
	case 'o':
		if len(hint) == 1 {
			return &IntegerHint{Format: IntegerOctal}
		}
	case 'b':
		if len(hint) == 1 {
			return &IntegerHint{Format: IntegerBinary}
		}
	case 'd':
		if len(hint) == 1 {
			return &IntegerHint{Format: IntegerDecimal}
		}
		if len(hint) < 3 || hint[1] != '-' {
			return nil
		}
		for i := 2; i < len(hint); i++ {
			if hint[i] < '0' || hint[i] > '9' {
				return nil
			}
		}
		places, err := strconv.Atoi(hint[2:])
		if err != nil || places > 255 {
			return nil
		}
		return &IntegerHint{Format: IntegerDecimal, DecimalPlaces: places}
	}
	return nil
}

func parseOctetStringHint(hint string) *OctetStringHint {
	p := 0
	var segments []OctetSegment
	lastSpecConsumes := false

	for p < len(hint) {
		var seg OctetSegment

		// (1) Optional '*' repeat indicator
		if hint[p] == '*' {
			seg.Repeat = true
			p++
		}

		// (2) Required octet count (one or more digits)
		digitStart := p
		length := 0
		for p < len(hint) && hint[p] >= '0' && hint[p] <= '9' {
			next := length*10 + int(hint[p]-'0')
			if next < length { // overflow
				return nil
			}
			length = next
			p++
		}
		if p == digitStart {
			return nil
		}
		seg.Length = length

		// (3) Required format character
		if p >= len(hint) {
			return nil
		}
		switch hint[p] {
		case 'd':
			seg.Format = OctetDecimal
		case 'x':
			seg.Format = OctetHex
		case 'o':
			seg.Format = OctetOctal
		case 'a':
			seg.Format = OctetAscii
		case 't':
			seg.Format = OctetUtf8
		default:
			return nil
		}
		p++

		// (4) Optional separator (not a digit, not *)
		if p < len(hint) && hint[p] != '*' && (hint[p] < '0' || hint[p] > '9') {
			seg.Separator = hint[p]
			p++

			// (5) Optional terminator (only with repeat and separator)
			if seg.Repeat && p < len(hint) && hint[p] != '*' && (hint[p] < '0' || hint[p] > '9') {
				seg.Terminator = hint[p]
				p++
			}
		}

		lastSpecConsumes = seg.Length > 0 || seg.Repeat
		segments = append(segments, seg)
	}

	if !lastSpecConsumes {
		return nil
	}

	return &OctetStringHint{Segments: segments}
}

// IsValidIntegerHint reports whether hint is a valid integer display hint.
func IsValidIntegerHint(hint string) bool {
	if hint == "" {
		return false
	}
	return parseIntegerHint(hint) != nil
}

// IsValidOctetStringHint reports whether hint is a valid octet-string display hint.
func IsValidOctetStringHint(hint string) bool {
	if hint == "" {
		return false
	}
	return parseOctetStringHint(hint) != nil
}

// FormatInteger formats an integer value according to an RFC 2579 integer
// display hint. Returns the formatted string and true, or "" and false if the
// hint is malformed.
func FormatInteger(hint string, value int64, hexCase HexCase) (string, bool) {
	if hint == "" {
		return "", false
	}

	rest := hint[1:]
	switch hint[0] {
	case 'x':
		if rest == "" {
			return formatSigned(value, 16, hexCase), true
		}
	case 'o':
		if rest == "" {
			return formatSigned(value, 8, hexCase), true
		}
	case 'b':
		if rest == "" {
			return formatSigned(value, 2, hexCase), true
		}
	case 'd':
		if rest == "" {
			return strconv.FormatInt(value, 10), true
		}
		if len(rest) < 2 || rest[0] != '-' {
			return "", false
		}
		places, err := strconv.Atoi(rest[1:])
		if err != nil || places > 100 {
			return "", false
		}
		if places == 0 {
			return strconv.FormatInt(value, 10), true
		}
		return formatDecimalWithPoint(value, places), true
	}
	return "", false
}

// ScaleInteger applies an RFC 2579 integer display hint as numeric scaling.
// Only "d" and "d-N" hints produce a result: "d" returns the value as-is,
// "d-N" divides by 10^N. Returns 0 and false for non-decimal or malformed hints.
func ScaleInteger(hint string, value int64) (float64, bool) {
	if hint == "" || hint[0] != 'd' {
		return 0, false
	}
	rest := hint[1:]
	if rest == "" {
		return float64(value), true
	}
	if rest[0] != '-' {
		return 0, false
	}
	places, err := strconv.Atoi(rest[1:])
	if err != nil || places > 20 {
		return 0, false
	}
	if places == 0 {
		return float64(value), true
	}
	return float64(value) / math.Pow(10, float64(places)), true
}

// FormatOctets formats an octet string according to an RFC 2579 octet-string
// display hint. Returns the formatted string and true, or "" and false if the
// hint is malformed or data is empty.
func FormatOctets(hint string, data []byte, hexCase HexCase) (string, bool) {
	if hint == "" || len(data) == 0 {
		return "", false
	}

	var result strings.Builder
	result.Grow(len(data) * 4)

	hintPos := 0
	dataPos := 0

	// Cached spec fields from the last parsed hint segment, used for
	// implicit repetition of the last spec.
	var cached struct {
		star, consumes  bool
		take            int
		fmtChar         byte
		hasSep, hasTerm bool
		sep, term       byte
	}

	hexTable := hexUpperTable
	if hexCase == HexLower {
		hexTable = hexLowerTable
	}

	for dataPos < len(data) {
		var (
			starPrefix      bool
			take            int
			fmtChar         byte
			hasSep, hasTerm bool
			sep, term       byte
		)

		if hintPos >= len(hint) {
			// Hint exhausted: reuse the cached last spec (implicit repetition).
			if !cached.consumes {
				return "", false
			}
			starPrefix = cached.star
			take = cached.take
			fmtChar = cached.fmtChar
			hasSep = cached.hasSep
			sep = cached.sep
			hasTerm = cached.hasTerm
			term = cached.term
		} else {
			// (1) Optional '*' repeat indicator.
			if hint[hintPos] == '*' {
				starPrefix = true
				hintPos++
			}

			// (2) Octet length (required, one or more decimal digits).
			if hintPos >= len(hint) || hint[hintPos] < '0' || hint[hintPos] > '9' {
				return "", false
			}
			take = 0
			for hintPos < len(hint) && hint[hintPos] >= '0' && hint[hintPos] <= '9' {
				next := take*10 + int(hint[hintPos]-'0')
				if next < take { // overflow
					return "", false
				}
				take = next
				hintPos++
			}

			// (3) Format character (required).
			if hintPos >= len(hint) {
				return "", false
			}
			fmtChar = hint[hintPos]
			switch fmtChar {
			case 'd', 'x', 'o', 'a', 't':
			default:
				return "", false
			}
			hintPos++

			// (4) Optional separator (any char that isn't a digit or '*').
			if hintPos < len(hint) && hint[hintPos] != '*' && (hint[hintPos] < '0' || hint[hintPos] > '9') {
				hasSep = true
				sep = hint[hintPos]
				hintPos++

				// (5) Optional terminator (only valid with star prefix).
				if starPrefix && hintPos < len(hint) && hint[hintPos] != '*' && (hint[hintPos] < '0' || hint[hintPos] > '9') {
					hasTerm = true
					term = hint[hintPos]
					hintPos++
				}
			}

			cached.star = starPrefix
			cached.take = take
			cached.fmtChar = fmtChar
			cached.hasSep = hasSep
			cached.sep = sep
			cached.hasTerm = hasTerm
			cached.term = term
			cached.consumes = take > 0 || starPrefix
		}

		// Determine repeat count.
		repeatCount := 1
		if starPrefix && dataPos < len(data) {
			repeatCount = int(data[dataPos])
			dataPos++
		}

		for r := range repeatCount {
			if dataPos >= len(data) {
				break
			}

			end := dataPos + take
			if end > len(data) || end < dataPos { // overflow or clamp
				end = len(data)
			}
			chunk := data[dataPos:end]

			switch fmtChar {
			case 'd':
				if len(chunk) > 8 {
					return "", false
				}
				val := bigEndianU64(chunk)
				fmt.Fprintf(&result, "%d", val)
			case 'x':
				for _, b := range chunk {
					result.WriteByte(hexTable[b>>4])
					result.WriteByte(hexTable[b&0x0F])
				}
			case 'o':
				if len(chunk) > 8 {
					return "", false
				}
				val := bigEndianU64(chunk)
				fmt.Fprintf(&result, "%o", val)
			case 'a':
				// ASCII with Latin-1 fallback for non-UTF8 bytes.
				if utf8.Valid(chunk) {
					result.Write(chunk)
				} else {
					for _, b := range chunk {
						result.WriteRune(rune(b))
					}
				}
			case 't':
				// UTF-8: emit valid prefix, discard trailing incomplete chars.
				if utf8.Valid(chunk) {
					result.Write(chunk)
				} else {
					// Find the longest valid UTF-8 prefix.
					valid := chunk
					for len(valid) > 0 && !utf8.Valid(valid) {
						valid = valid[:len(valid)-1]
					}
					result.Write(valid)
				}
			}
			dataPos = end

			// Emit separator (suppressed at end of data or before terminator).
			moreData := dataPos < len(data)
			if hasSep && moreData && (!hasTerm || r != repeatCount-1) {
				result.WriteByte(sep)
			}
		}

		// Emit terminator after repeat group.
		if hasTerm && dataPos < len(data) {
			result.WriteByte(term)
		}
	}

	return result.String(), true
}

var (
	hexUpperTable = [16]byte{'0', '1', '2', '3', '4', '5', '6', '7', '8', '9', 'A', 'B', 'C', 'D', 'E', 'F'}
	hexLowerTable = [16]byte{'0', '1', '2', '3', '4', '5', '6', '7', '8', '9', 'a', 'b', 'c', 'd', 'e', 'f'}
)

func formatSigned(value int64, base int, hexCase HexCase) string {
	abs := uint64(value)
	prefix := ""
	if value < 0 {
		// Two's complement: -MinInt64 overflows int64, but uint64 handles it.
		abs = uint64(-value)
		if value == math.MinInt64 {
			abs = uint64(math.MaxInt64) + 1
		}
		prefix = "-"
	}
	switch base {
	case 16:
		if abs == 0 {
			return prefix + "0"
		}
		table := hexUpperTable
		if hexCase == HexLower {
			table = hexLowerTable
		}
		// Find highest nibble, then emit from there.
		var buf [16]byte
		n := 0
		for v := abs; v > 0; v >>= 4 {
			buf[n] = table[v&0x0F]
			n++
		}
		// Reverse into result.
		result := make([]byte, len(prefix)+n)
		copy(result, prefix)
		for i := range n {
			result[len(prefix)+i] = buf[n-1-i]
		}
		return string(result)
	case 8:
		return prefix + strconv.FormatUint(abs, 8)
	case 2:
		return prefix + strconv.FormatUint(abs, 2)
	default:
		return prefix + strconv.FormatUint(abs, 10)
	}
}

func formatDecimalWithPoint(value int64, places int) string {
	negative := value < 0
	abs := uint64(value)
	if negative {
		abs = uint64(-value)
		if value == math.MinInt64 {
			abs = uint64(math.MaxInt64) + 1
		}
	}
	digits := strconv.FormatUint(abs, 10)

	var b strings.Builder
	b.Grow(len(digits) + 2 + places)

	if negative {
		b.WriteByte('-')
	}

	if len(digits) <= places {
		b.WriteString("0.")
		for range places - len(digits) {
			b.WriteByte('0')
		}
		b.WriteString(digits)
	} else {
		split := len(digits) - places
		b.WriteString(digits[:split])
		b.WriteByte('.')
		b.WriteString(digits[split:])
	}

	return b.String()
}

func bigEndianU64(b []byte) uint64 {
	var v uint64
	for _, x := range b {
		v = v<<8 | uint64(x)
	}
	return v
}
