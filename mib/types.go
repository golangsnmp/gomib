package mib

import (
	"strconv"
	"strings"
)

// IndexEntry describes an index component for a table row.
type IndexEntry struct {
	Object  Object // always non-nil in resolved model
	Implied bool   // IMPLIED keyword present
}

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
// For INTEGER enums, Value is the enum constant.
// For BITS, Value is the bit position (0-based).
type NamedValue struct {
	Label string
	Value int64
}

// DefVal is the interface for default values.
// All DefVal types implement String() for display.
type DefVal interface {
	String() string
	defVal()
}

// DefValInt is a signed integer default value.
type DefValInt int64

func (DefValInt) defVal() {}

// String returns the integer as a decimal string.
func (d DefValInt) String() string { return strconv.FormatInt(int64(d), 10) }

// DefValUnsigned is an unsigned integer default value.
type DefValUnsigned uint64

func (DefValUnsigned) defVal() {}

// String returns the integer as a decimal string.
func (d DefValUnsigned) String() string { return strconv.FormatUint(uint64(d), 10) }

// DefValString is a quoted string default value.
type DefValString string

func (DefValString) defVal() {}

// String returns the string value with quotes.
func (d DefValString) String() string { return `"` + string(d) + `"` }

// DefValHexString is a hex string default value (e.g., '1F2E'H).
type DefValHexString string

func (DefValHexString) defVal() {}

// String returns the hex string in MIB format (e.g., '1F2E'H).
func (d DefValHexString) String() string { return "'" + string(d) + "'H" }

// DefValBinaryString is a binary string default value (e.g., '1010'B).
type DefValBinaryString string

func (DefValBinaryString) defVal() {}

// String returns the binary string in MIB format (e.g., '1010'B).
func (d DefValBinaryString) String() string { return "'" + string(d) + "'B" }

// DefValEnum is an enumeration label default value.
type DefValEnum string

func (DefValEnum) defVal() {}

// String returns the enum label.
func (d DefValEnum) String() string { return string(d) }

// DefValBits is a BITS default value (list of bit labels).
type DefValBits []string

func (DefValBits) defVal() {}

// String returns the bit labels in braces (e.g., { bit1, bit2 }).
func (d DefValBits) String() string {
	if len(d) == 0 {
		return "{ }"
	}
	return "{ " + strings.Join(d, ", ") + " }"
}

// DefValOID is an OID default value.
type DefValOID Oid

func (DefValOID) defVal() {}

// String returns the OID as a dotted string.
func (d DefValOID) String() string { return Oid(d).String() }

// Revision describes a module revision.
type Revision struct {
	Date        string // "YYYY-MM-DD" or original format
	Description string
}

// Diagnostic represents a parse or resolution issue.
type Diagnostic struct {
	Severity Severity
	Module   string // source module name
	Message  string
	Line     int // 0 if not applicable
}

// UnresolvedRef describes a symbol that could not be resolved.
type UnresolvedRef struct {
	Kind   string // "type", "object", "import"
	Symbol string // the unresolved symbol
	Module string // where it was referenced
}
