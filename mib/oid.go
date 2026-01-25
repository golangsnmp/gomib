package mib

import (
	"fmt"
	"slices"
	"strings"
)

// Oid is a sequence of arc values representing an SNMP Object Identifier.
// It is a defined type (not alias) so methods can be attached.
type Oid []uint32

// ParseOID parses an OID from a dotted string (e.g., "1.3.6.1.2.1").
func ParseOID(s string) (Oid, error) {
	if s == "" {
		return nil, nil
	}
	// Handle leading dot (e.g., ".1.3.6.1")
	if len(s) > 0 && s[0] == '.' {
		s = s[1:]
	}
	if s == "" {
		return nil, nil
	}

	var arcs []uint32
	var current uint32
	var hasDigit bool
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= '0' && c <= '9' {
			current = current*10 + uint32(c-'0')
			hasDigit = true
		} else if c == '.' {
			if !hasDigit {
				return nil, fmt.Errorf("empty arc in OID: %s", s)
			}
			arcs = append(arcs, current)
			current = 0
			hasDigit = false
		} else {
			return nil, fmt.Errorf("invalid character in OID: %c", c)
		}
	}
	if hasDigit {
		arcs = append(arcs, current)
	}
	return arcs, nil
}

// String returns the dotted string representation (e.g., "1.3.6.1.2.1").
func (o Oid) String() string {
	if len(o) == 0 {
		return ""
	}
	var b strings.Builder
	fmt.Fprintf(&b, "%d", o[0])
	for _, arc := range o[1:] {
		fmt.Fprintf(&b, ".%d", arc)
	}
	return b.String()
}

// Parent returns the parent OID (all arcs except the last).
// Returns nil if the OID is empty or has only one arc.
func (o Oid) Parent() Oid {
	if len(o) <= 1 {
		return nil
	}
	return slices.Clone(o[:len(o)-1])
}

// Child returns a new OID with the given arc appended.
func (o Oid) Child(arc uint32) Oid {
	result := make(Oid, len(o)+1)
	copy(result, o)
	result[len(result)-1] = arc
	return result
}

// HasPrefix returns true if this OID starts with the given prefix.
func (o Oid) HasPrefix(prefix Oid) bool {
	if len(prefix) > len(o) {
		return false
	}
	for i, arc := range prefix {
		if o[i] != arc {
			return false
		}
	}
	return true
}

// Equal returns true if the OIDs are identical.
func (o Oid) Equal(other Oid) bool {
	return slices.Equal(o, other)
}

// Compare returns -1 if o < other, 0 if equal, 1 if o > other.
// Comparison is lexicographic by arc value.
func (o Oid) Compare(other Oid) int {
	return slices.Compare(o, other)
}

// LastArc returns the last arc value, or 0 if empty.
func (o Oid) LastArc() uint32 {
	if len(o) == 0 {
		return 0
	}
	return o[len(o)-1]
}
