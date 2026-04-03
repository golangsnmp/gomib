package mib

import (
	"fmt"
	"net"
	"slices"
	"strconv"
	"strings"
)

// IndexValueKind identifies the type of a decoded index value.
type IndexValueKind int

const (
	IndexValueInteger          IndexValueKind = iota + 1 // single-arc integer
	IndexValueIpAddress                                  // four-arc IPv4 address
	IndexValueOctetString                                // octet string (fixed, length-prefixed, or implied)
	IndexValueObjectIdentifier                           // sub-OID (length-prefixed or implied)
)

// String returns a short name for the kind.
func (k IndexValueKind) String() string {
	switch k {
	case IndexValueInteger:
		return "integer"
	case IndexValueIpAddress:
		return "ip-address"
	case IndexValueOctetString:
		return "octet-string"
	case IndexValueObjectIdentifier:
		return "object-identifier"
	default:
		return "unknown"
	}
}

// IndexValue is a typed value decoded from OID instance suffix arcs.
//
// The kind is determined by the index component's [IndexEncoding] and
// base type, following RFC 2578 section 7.7:
//
//   - Integer types produce [IndexValueInteger] (one arc).
//   - IpAddress produces [IndexValueIpAddress] (four arcs).
//   - OCTET STRING / Opaque / BITS produce [IndexValueOctetString],
//     either fixed-length, length-prefixed, or implied.
//   - OBJECT IDENTIFIER produces [IndexValueObjectIdentifier],
//     either length-prefixed or implied.
type IndexValue struct {
	kind  IndexValueKind
	value any // uint32 | [4]byte | []byte | OID
}

// Kind returns the type of the index value.
func (v IndexValue) Kind() IndexValueKind { return v.kind }

// Integer returns the integer value and true if this is an integer index,
// or (0, false) otherwise.
func (v IndexValue) Integer() (uint32, bool) {
	if v.kind == IndexValueInteger {
		return v.value.(uint32), true
	}
	return 0, false
}

// IpAddress returns the four-octet IPv4 address and true if this is an
// IP address index, or (zero, false) otherwise.
func (v IndexValue) IpAddress() ([4]byte, bool) {
	if v.kind == IndexValueIpAddress {
		return v.value.([4]byte), true
	}
	return [4]byte{}, false
}

// IP returns the IPv4 address as [net.IP], or nil if this is not an
// IP address index.
func (v IndexValue) IP() net.IP {
	if v.kind == IndexValueIpAddress {
		b := v.value.([4]byte)
		return net.IPv4(b[0], b[1], b[2], b[3])
	}
	return nil
}

// Bytes returns the octet string value and true, or (nil, false).
func (v IndexValue) Bytes() ([]byte, bool) {
	if v.kind == IndexValueOctetString {
		return slices.Clone(v.value.([]byte)), true
	}
	return nil, false
}

// OID returns the object identifier arcs and true, or (nil, false).
func (v IndexValue) OID() (OID, bool) {
	if v.kind == IndexValueObjectIdentifier {
		return slices.Clone(v.value.(OID)), true
	}
	return nil, false
}

// String returns a human-readable representation of the value.
func (v IndexValue) String() string {
	switch v.kind {
	case IndexValueInteger:
		return strconv.FormatUint(uint64(v.value.(uint32)), 10)
	case IndexValueIpAddress:
		b := v.value.([4]byte)
		return net.IPv4(b[0], b[1], b[2], b[3]).String()
	case IndexValueOctetString:
		data := v.value.([]byte)
		if isPrintableASCII(data) {
			return string(data)
		}
		var sb strings.Builder
		for i, b := range data {
			if i > 0 {
				sb.WriteByte(':')
			}
			fmt.Fprintf(&sb, "%02X", b)
		}
		return sb.String()
	case IndexValueObjectIdentifier:
		return v.value.(OID).String()
	default:
		return ""
	}
}

func isPrintableASCII(data []byte) bool {
	if len(data) == 0 {
		return false
	}
	for _, b := range data {
		if b < 0x20 || b > 0x7E {
			return false
		}
	}
	return true
}

// DecodedIndex is a single decoded index component from an instance OID suffix.
// It pairs the index component name (from the INDEX clause) with the
// decoded [IndexValue].
type DecodedIndex struct {
	name  string
	value IndexValue
}

// Name returns the index component name from the INDEX clause (e.g. "ifIndex").
func (d DecodedIndex) Name() string { return d.name }

// Value returns the decoded value.
func (d DecodedIndex) Value() IndexValue { return d.value }

// String returns a "name=value" representation.
func (d DecodedIndex) String() string {
	return d.name + "=" + d.value.String()
}

// DecodeSuffix decodes an OID instance suffix into typed index values.
//
// Takes the index entries from a row's effective INDEX clause and the raw
// suffix arcs (the portion of an instance OID that follows the column or
// row OID). Returns one [DecodedIndex] per successfully decoded component.
//
// Decoding stops early if:
//   - The suffix arcs are exhausted before all indexes are decoded
//   - An index component has [IndexEncodingUnknown]
//   - A fixed-size index has no determinable SIZE constraint
//   - There are not enough arcs for a component
//   - An octet-valued arc (IpAddress, OCTET STRING) exceeds 255
//
// Encoding rules (RFC 2578 section 7.7):
//
//   - Integer: 1 arc (single sub-identifier)
//   - IpAddress: 4 arcs (one per octet)
//   - FixedString: n arcs (n = fixed SIZE constraint)
//   - LengthPrefixed: 1 + n arcs (first arc is length)
//   - Implied: remaining arcs (last index only)
func DecodeSuffix(indexes []IndexEntry, suffix OID) []DecodedIndex {
	var result []DecodedIndex
	pos := 0

	for _, idx := range indexes {
		if pos >= len(suffix) {
			break
		}

		name := indexName(idx)
		var value IndexValue
		var consumed int

		switch idx.Encoding {
		case IndexEncodingInteger:
			value = IndexValue{kind: IndexValueInteger, value: suffix[pos]}
			consumed = 1

		case IndexEncodingIpAddress:
			if pos+4 > len(suffix) {
				return result
			}
			bytes, ok := arcsToBytes(suffix[pos : pos+4])
			if !ok {
				return result
			}
			value = IndexValue{kind: IndexValueIpAddress, value: [4]byte{bytes[0], bytes[1], bytes[2], bytes[3]}}
			consumed = 4

		case IndexEncodingFixedString:
			size, _ := idx.FixedSize()
			if size == 0 || pos+size > len(suffix) {
				return result
			}
			bytes, ok := arcsToBytes(suffix[pos : pos+size])
			if !ok {
				return result
			}
			value = IndexValue{kind: IndexValueOctetString, value: bytes}
			consumed = size

		case IndexEncodingLengthPrefixed:
			length := int(suffix[pos])
			if pos+1+length > len(suffix) {
				return result
			}
			data := suffix[pos+1 : pos+1+length]
			if isOIDBaseType(idx) {
				arcs := make(OID, len(data))
				copy(arcs, data)
				value = IndexValue{kind: IndexValueObjectIdentifier, value: arcs}
			} else {
				bytes, ok := arcsToBytes(data)
				if !ok {
					return result
				}
				value = IndexValue{kind: IndexValueOctetString, value: bytes}
			}
			consumed = 1 + length

		case IndexEncodingImplied:
			remaining := suffix[pos:]
			if isOIDBaseType(idx) {
				arcs := make(OID, len(remaining))
				copy(arcs, remaining)
				value = IndexValue{kind: IndexValueObjectIdentifier, value: arcs}
			} else {
				bytes, ok := arcsToBytes(remaining)
				if !ok {
					return result
				}
				value = IndexValue{kind: IndexValueOctetString, value: bytes}
			}
			consumed = len(remaining)

		default:
			return result
		}

		result = append(result, DecodedIndex{name: name, value: value})
		pos += consumed
	}

	return result
}

// indexName returns the name for an index entry.
func indexName(idx IndexEntry) string {
	if idx.Object != nil {
		return idx.Object.Name()
	}
	return idx.TypeName
}

// arcsToBytes converts OID arcs to bytes, returning false if any arc exceeds 255.
func arcsToBytes(arcs []uint32) ([]byte, bool) {
	bytes := make([]byte, len(arcs))
	for i, a := range arcs {
		if a > 255 {
			return nil, false
		}
		bytes[i] = byte(a)
	}
	return bytes, true
}

// isOIDBaseType reports whether the index entry's effective base type is ObjectIdentifier.
func isOIDBaseType(idx IndexEntry) bool {
	if idx.Object == nil {
		return false
	}
	t := idx.Object.Type()
	if t == nil {
		return false
	}
	return t.EffectiveBase() == BaseObjectIdentifier
}
