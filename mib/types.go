package mib

import (
	"encoding/hex"
	"strconv"
	"strings"
)

// Import describes a group of symbols imported from a single source module.
type Import struct {
	Module  string   // source module name
	Symbols []string // imported symbol names
}

// Range represents a min..max constraint for sizes or values.
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

func findNamedValue(values []NamedValue, label string) (NamedValue, bool) {
	for _, nv := range values {
		if nv.Label == label {
			return nv, true
		}
	}
	return NamedValue{}, false
}

// Revision describes a module revision.
type Revision struct {
	Date        string // "YYYY-MM-DD" or original format
	Description string
}

// IndexEntry describes an index component for a table row.
type IndexEntry struct {
	Object  *Object // always non-nil in resolved model
	Implied bool    // IMPLIED keyword present
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
	DefValKindOID                      // OID
)

// DefVal represents a default value with both interpreted value and raw MIB syntax.
type DefVal struct {
	kind  DefValKind
	value any
	raw   string
}

// newDefValInt creates a DefVal for a signed integer.
func newDefValInt(v int64, raw string) DefVal {
	return DefVal{kind: DefValKindInt, value: v, raw: raw}
}

// newDefValUint creates a DefVal for an unsigned integer.
func newDefValUint(v uint64, raw string) DefVal {
	return DefVal{kind: DefValKindUint, value: v, raw: raw}
}

// newDefValString creates a DefVal for a quoted string.
func newDefValString(v string, raw string) DefVal {
	return DefVal{kind: DefValKindString, value: v, raw: raw}
}

// newDefValBytes creates a DefVal for bytes (from hex/binary string).
func newDefValBytes(v []byte, raw string) DefVal {
	return DefVal{kind: DefValKindBytes, value: v, raw: raw}
}

// newDefValEnum creates a DefVal for an enum label.
func newDefValEnum(label string, raw string) DefVal {
	return DefVal{kind: DefValKindEnum, value: label, raw: raw}
}

// newDefValBits creates a DefVal for BITS (list of bit labels).
func newDefValBits(labels []string, raw string) DefVal {
	return DefVal{kind: DefValKindBits, value: labels, raw: raw}
}

// newDefValOID creates a DefVal for an OID.
func newDefValOID(oid OID, raw string) DefVal {
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
	if d.value == nil {
		return ""
	}
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
		return d.raw
	default:
		return d.raw
	}
}

// IsZero returns true if this is the zero value (no default set).
func (d DefVal) IsZero() bool {
	return d.value == nil
}

// String returns a human-readable name for the kind.
func (k DefValKind) String() string {
	switch k {
	case DefValKindInt:
		return "int"
	case DefValKindUint:
		return "uint"
	case DefValKindString:
		return "string"
	case DefValKindBytes:
		return "bytes"
	case DefValKindEnum:
		return "enum"
	case DefValKindBits:
		return "bits"
	case DefValKindOID:
		return "oid"
	default:
		return "DefValKind(" + strconv.Itoa(int(k)) + ")"
	}
}

// DefValAs returns the value as type T if compatible.
func DefValAs[T any](d DefVal) (T, bool) {
	v, ok := d.value.(T)
	return v, ok
}

func bytesToHex(b []byte) string {
	return strings.ToUpper(hex.EncodeToString(b))
}

// ComplianceModule is a MODULE clause within a MODULE-COMPLIANCE definition.
type ComplianceModule struct {
	ModuleName      string             // module name (empty = current module)
	MandatoryGroups []string           // MANDATORY-GROUPS references
	Groups          []ComplianceGroup  // GROUP refinements
	Objects         []ComplianceObject // OBJECT refinements
}

// ComplianceGroup is a GROUP clause within MODULE-COMPLIANCE.
type ComplianceGroup struct {
	Group       string // group reference name
	Description string
}

// ComplianceObject is an OBJECT refinement within MODULE-COMPLIANCE.
type ComplianceObject struct {
	Object      string  // object reference name
	MinAccess   *Access // MIN-ACCESS restriction (nil if not specified)
	Description string
}

// CapabilitiesModule is a SUPPORTS clause within an AGENT-CAPABILITIES definition.
type CapabilitiesModule struct {
	ModuleName             string                  // supported module name
	Includes               []string                // INCLUDES group references
	ObjectVariations       []ObjectVariation       // object VARIATION clauses
	NotificationVariations []NotificationVariation // notification VARIATION clauses
}

// ObjectVariation is an object VARIATION within AGENT-CAPABILITIES.
type ObjectVariation struct {
	Object      string  // object reference name
	Access      *Access // ACCESS restriction (nil if not specified)
	DefVal      DefVal  // overridden default value (zero if not specified)
	Description string
}

// NotificationVariation is a notification VARIATION within AGENT-CAPABILITIES.
type NotificationVariation struct {
	Notification string  // notification reference name
	Access       *Access // ACCESS restriction (nil if not specified)
	Description  string
}

// UnresolvedKind identifies the category of an unresolved reference.
type UnresolvedKind int

const (
	UnresolvedImport             UnresolvedKind = iota // cross-module import
	UnresolvedType                                     // type reference
	UnresolvedOID                                      // OID component
	UnresolvedIndex                                    // INDEX object reference
	UnresolvedNotificationObject                       // OBJECTS entry in notification
)

func (k UnresolvedKind) String() string {
	switch k {
	case UnresolvedImport:
		return "import"
	case UnresolvedType:
		return "type"
	case UnresolvedOID:
		return "oid"
	case UnresolvedIndex:
		return "index"
	case UnresolvedNotificationObject:
		return "notification-object"
	default:
		return "unknown"
	}
}

// UnresolvedRef describes a symbol that could not be resolved.
type UnresolvedRef struct {
	Kind   UnresolvedKind
	Symbol string
	Module string
}
