package mib

import (
	"strconv"
	"strings"
)

// Range for size/value constraints.
type Range struct {
	Min, Max int64
}

// String returns the range as "min..max" or just "value" if min equals max.
func (r Range) String() string {
	if r.Min == r.Max {
		return strconv.FormatInt(r.Min, 10)
	}
	return strconv.FormatInt(r.Min, 10) + ".." + strconv.FormatInt(r.Max, 10)
}

// NamedValue represents a labeled integer from an enum or BITS definition.
type NamedValue struct {
	Label string
	Value int64
}

// Revision describes a module revision.
type Revision struct {
	Date        string // "YYYY-MM-DD" or original format
	Description string
}

// IndexEntry describes an index component for a table row.
type IndexEntry struct {
	Name    string // object name (may be from another module)
	Implied bool   // IMPLIED keyword present
}

// DefValKind identifies the type of default value.
type DefValKind int

const (
	DefValKindInt    DefValKind = iota // int64
	DefValKindUint                     // uint64
	DefValKindString                   // string (quoted)
	DefValKindBytes                    // []byte (from hex/binary string)
	DefValKindEnum                     // string (enum label)
	DefValKindBits                     // []string (bit labels)
	DefValKindOID                      // Oid
)

// DefVal represents a default value with both interpreted value and raw MIB syntax.
type DefVal struct {
	kind  DefValKind
	value any
	raw   string
}

// NewDefValInt creates a DefVal for a signed integer.
func NewDefValInt(v int64, raw string) DefVal {
	return DefVal{kind: DefValKindInt, value: v, raw: raw}
}

// NewDefValUint creates a DefVal for an unsigned integer.
func NewDefValUint(v uint64, raw string) DefVal {
	return DefVal{kind: DefValKindUint, value: v, raw: raw}
}

// NewDefValString creates a DefVal for a quoted string.
func NewDefValString(v string, raw string) DefVal {
	return DefVal{kind: DefValKindString, value: v, raw: raw}
}

// NewDefValBytes creates a DefVal for bytes (from hex/binary string).
func NewDefValBytes(v []byte, raw string) DefVal {
	return DefVal{kind: DefValKindBytes, value: v, raw: raw}
}

// NewDefValEnum creates a DefVal for an enum label.
func NewDefValEnum(label string, raw string) DefVal {
	return DefVal{kind: DefValKindEnum, value: label, raw: raw}
}

// NewDefValBits creates a DefVal for BITS (list of bit labels).
func NewDefValBits(labels []string, raw string) DefVal {
	return DefVal{kind: DefValKindBits, value: labels, raw: raw}
}

// NewDefValOID creates a DefVal for an OID.
func NewDefValOID(oid Oid, raw string) DefVal {
	return DefVal{kind: DefValKindOID, value: oid, raw: raw}
}

// Kind returns the type of the default value.
func (d DefVal) Kind() DefValKind { return d.kind }

// Value returns the interpreted value.
func (d DefVal) Value() any { return d.value }

// Raw returns the original MIB syntax.
func (d DefVal) Raw() string { return d.raw }

// String returns a user-friendly representation.
func (d DefVal) String() string {
	switch d.kind {
	case DefValKindInt:
		return strconv.FormatInt(d.value.(int64), 10)
	case DefValKindUint:
		return strconv.FormatUint(d.value.(uint64), 10)
	case DefValKindString:
		return `"` + d.value.(string) + `"`
	case DefValKindBytes:
		b := d.value.([]byte)
		if len(b) == 0 {
			return "0"
		}
		if len(b) <= 8 {
			var n uint64
			for _, v := range b {
				n = n<<8 | uint64(v)
			}
			return strconv.FormatUint(n, 10)
		}
		return "0x" + bytesToHex(b)
	case DefValKindEnum:
		return d.value.(string)
	case DefValKindBits:
		labels := d.value.([]string)
		if len(labels) == 0 {
			return "{ }"
		}
		return "{ " + strings.Join(labels, ", ") + " }"
	case DefValKindOID:
		return d.value.(Oid).String()
	default:
		return d.raw
	}
}

// IsZero returns true if this is the zero value (no default set).
func (d DefVal) IsZero() bool {
	return d.value == nil
}

// DefValAs returns the value as type T if compatible.
func DefValAs[T any](d DefVal) (T, bool) {
	v, ok := d.value.(T)
	return v, ok
}

func bytesToHex(b []byte) string {
	const hex = "0123456789ABCDEF"
	result := make([]byte, len(b)*2)
	for i, v := range b {
		result[i*2] = hex[v>>4]
		result[i*2+1] = hex[v&0x0f]
	}
	return string(result)
}

// UnresolvedRef describes a symbol that could not be resolved.
type UnresolvedRef struct {
	Kind   string // "type", "object", "import"
	Symbol string
	Module string
}
