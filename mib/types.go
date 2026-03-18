package mib

import (
	"encoding/hex"
	"slices"
	"strconv"
	"strings"
)

// ImportSymbol is a single imported symbol with its source location.
type ImportSymbol struct {
	Name string
	Span Span
}

// Import describes a group of symbols imported from a single source module.
type Import struct {
	Module  string
	Symbols []ImportSymbol
}

// Range represents a min..max constraint for sizes or values.
type Range struct {
	Min, Max int64
	Span     Span
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
	Span  Span
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
	Span        Span
}

// IndexEntry describes an index component for a table row.
// Exactly one of Object or TypeName is set: Object for object-reference
// indexes, TypeName for bare-type indexes (e.g. INDEX { INTEGER }).
type IndexEntry struct {
	Object   *Object       // nil for bare type indexes
	TypeName string        // non-empty for bare type indexes
	Implied  bool          // IMPLIED keyword present
	Encoding IndexEncoding // RFC 2578 s7.7 encoding classification, set during resolution
	Span     Span
}

// classifyIndexEncoding determines the index encoding from the object's
// resolved type and effective constraints. Returns IndexEncodingUnknown
// for bare type indexes or unresolved types.
func classifyIndexEncoding(obj *Object, implied bool) IndexEncoding {
	if obj == nil {
		return IndexEncodingUnknown
	}
	t := obj.Type()
	if t == nil {
		return IndexEncodingUnknown
	}
	base := t.EffectiveBase()
	switch base {
	case BaseInteger32, BaseUnsigned32, BaseGauge32, BaseTimeTicks,
		BaseCounter32, BaseCounter64:
		return IndexEncodingInteger
	case BaseIpAddress:
		return IndexEncodingIpAddress
	case BaseOctetString, BaseOpaque, BaseBits:
		if implied {
			return IndexEncodingImplied
		}
		if isFixedSize(obj.sizes) {
			return IndexEncodingFixedString
		}
		return IndexEncodingLengthPrefixed
	case BaseObjectIdentifier:
		if implied {
			return IndexEncodingImplied
		}
		return IndexEncodingLengthPrefixed
	default:
		return IndexEncodingUnknown
	}
}

// FixedSize returns the fixed encoding width for this index entry, if
// determinable. Returns the width in sub-identifiers and true for
// fixed-width encodings (integer, IP address, fixed-size string).
// Returns 0 and false for variable-length or unknown encodings.
func (e IndexEntry) FixedSize() (int, bool) {
	switch e.Encoding {
	case IndexEncodingInteger:
		return 1, true
	case IndexEncodingIpAddress:
		return 4, true
	case IndexEncodingFixedString:
		if e.Object != nil && isFixedSize(e.Object.sizes) {
			return int(e.Object.sizes[0].Max), true
		}
		return 0, false
	default:
		return 0, false
	}
}

// isFixedSize reports whether sizes contains exactly one constraint with
// min == max (and > 0), indicating a fixed-length string.
func isFixedSize(sizes []Range) bool {
	return len(sizes) == 1 && sizes[0].Min == sizes[0].Max && sizes[0].Min > 0
}

// DefValKind identifies the type of default value.
type DefValKind int

const (
	DefValKindUnset  DefValKind = iota // zero value, no default set
	DefValKindInt                      // int64
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
func newDefValString(v, raw string) DefVal {
	return DefVal{kind: DefValKindString, value: v, raw: raw}
}

// newDefValBytes creates a DefVal for bytes (from hex/binary string).
func newDefValBytes(v []byte, raw string) DefVal {
	return DefVal{kind: DefValKindBytes, value: slices.Clone(v), raw: raw}
}

// newDefValEnum creates a DefVal for an enum label.
func newDefValEnum(label, raw string) DefVal {
	return DefVal{kind: DefValKindEnum, value: label, raw: raw}
}

// newDefValBits creates a DefVal for BITS (list of bit labels).
func newDefValBits(labels []string, raw string) DefVal {
	return DefVal{kind: DefValKindBits, value: slices.Clone(labels), raw: raw}
}

// newDefValOID creates a DefVal for an OID.
func newDefValOID(oid OID, raw string) DefVal {
	return DefVal{kind: DefValKindOID, value: slices.Clone(oid), raw: raw}
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
		return `"` + strings.ReplaceAll(d.value.(string), `"`, `\"`) + `"`
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

// clone returns a deep copy of the DefVal, cloning any slice-typed values.
func (d DefVal) clone() DefVal {
	switch v := d.value.(type) {
	case []byte:
		d.value = slices.Clone(v)
	case []string:
		d.value = slices.Clone(v)
	case OID:
		d.value = slices.Clone(v)
	}
	return d
}

// String returns a human-readable name for the default value kind
// (e.g. "int", "string", "enum").
func (k DefValKind) String() string {
	switch k {
	case DefValKindUnset:
		return "unset"
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

// DefValAs extracts the default value as the concrete type T.
// It returns the value and true if the underlying value is assignable to T,
// or the zero value and false otherwise. For example, DefValAs[int64](d)
// succeeds when d.Kind() is DefValKindInt.
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
	Span            Span
}

// ComplianceGroup is a GROUP clause within MODULE-COMPLIANCE.
type ComplianceGroup struct {
	Group       string // group reference name
	Description string
	Span        Span
}

// ComplianceObject is an OBJECT refinement within MODULE-COMPLIANCE.
type ComplianceObject struct {
	Object      string             // object reference name
	Syntax      *SyntaxConstraints // SYNTAX refinement (nil if not specified)
	WriteSyntax *SyntaxConstraints // WRITE-SYNTAX refinement (nil if not specified)
	MinAccess   *Access            // MIN-ACCESS restriction (nil if not specified)
	Description string
	Span        Span
}

// clone returns a deep copy of the ComplianceModule.
func (cm *ComplianceModule) clone() ComplianceModule {
	result := *cm
	result.MandatoryGroups = slices.Clone(cm.MandatoryGroups)
	result.Groups = slices.Clone(cm.Groups)
	result.Objects = slices.Clone(cm.Objects)
	for i := range result.Objects {
		result.Objects[i].Syntax = result.Objects[i].Syntax.clone()
		result.Objects[i].WriteSyntax = result.Objects[i].WriteSyntax.clone()
	}
	return result
}

// CapabilitiesModule is a SUPPORTS clause within an AGENT-CAPABILITIES definition.
type CapabilitiesModule struct {
	ModuleName             string                  // supported module name
	Includes               []string                // INCLUDES group references
	ObjectVariations       []ObjectVariation       // object VARIATION clauses
	NotificationVariations []NotificationVariation // notification VARIATION clauses
	Span                   Span
}

// clone returns a deep copy of the CapabilitiesModule.
func (cm *CapabilitiesModule) clone() CapabilitiesModule {
	result := *cm
	result.Includes = slices.Clone(cm.Includes)
	result.ObjectVariations = slices.Clone(cm.ObjectVariations)
	for i := range result.ObjectVariations {
		result.ObjectVariations[i].Syntax = result.ObjectVariations[i].Syntax.clone()
		result.ObjectVariations[i].WriteSyntax = result.ObjectVariations[i].WriteSyntax.clone()
		result.ObjectVariations[i].CreationRequires = slices.Clone(result.ObjectVariations[i].CreationRequires)
		result.ObjectVariations[i].DefVal = result.ObjectVariations[i].DefVal.clone()
	}
	result.NotificationVariations = slices.Clone(cm.NotificationVariations)
	return result
}

// ObjectVariation is an object VARIATION within AGENT-CAPABILITIES.
type ObjectVariation struct {
	Object           string             // object reference name
	Syntax           *SyntaxConstraints // SYNTAX refinement (nil if not specified)
	WriteSyntax      *SyntaxConstraints // WRITE-SYNTAX refinement (nil if not specified)
	Access           *Access            // ACCESS restriction (nil if not specified)
	CreationRequires []string           // CREATION-REQUIRES columns (nil if not specified)
	DefVal           DefVal             // overridden default value (zero if not specified)
	Description      string
	Span             Span
}

// NotificationVariation is a notification VARIATION within AGENT-CAPABILITIES.
type NotificationVariation struct {
	Notification string  // notification reference name
	Access       *Access // ACCESS restriction (nil if not specified)
	Description  string
	Span         Span
}

// SyntaxConstraints holds a resolved type reference and any inline
// constraints from a VARIATION SYNTAX/WRITE-SYNTAX clause or
// MODULE-COMPLIANCE OBJECT SYNTAX/WRITE-SYNTAX refinement.
type SyntaxConstraints struct {
	Type   *Type
	Sizes  []Range
	Ranges []Range
	Enums  []NamedValue
	Bits   []NamedValue
}

// clone returns a deep copy of sc, or nil if sc is nil.
func (sc *SyntaxConstraints) clone() *SyntaxConstraints {
	if sc == nil {
		return nil
	}
	c := *sc
	c.Sizes = slices.Clone(sc.Sizes)
	c.Ranges = slices.Clone(sc.Ranges)
	c.Enums = slices.Clone(sc.Enums)
	c.Bits = slices.Clone(sc.Bits)
	return &c
}

// TrapInfo holds SMIv1 TRAP-TYPE fields. Only present on notifications
// originating from TRAP-TYPE definitions.
type TrapInfo struct {
	Enterprise string
	TrapNumber uint32
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

// String returns a human-readable name for the unresolved reference kind
// (e.g. "import", "type", "oid").
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
	Reason string
}
