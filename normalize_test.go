package gomib_test

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/golangsnmp/gomib/internal/testutil"
	"github.com/golangsnmp/gomib/mib"
)

// normalizeType converts a gomib Type to the normalized string used in fixtures.
func normalizeType(t *mib.Type) string {
	if t == nil {
		return ""
	}
	base := t.EffectiveBase()
	switch base {
	case mib.BaseInteger32:
		return "Integer32"
	case mib.BaseUnsigned32:
		return "Unsigned32"
	case mib.BaseCounter32:
		return "Counter32"
	case mib.BaseCounter64:
		return "Counter64"
	case mib.BaseGauge32:
		return "Gauge32"
	case mib.BaseTimeTicks:
		return "TimeTicks"
	case mib.BaseIpAddress:
		return "IpAddress"
	case mib.BaseOctetString:
		return "OCTET STRING"
	case mib.BaseObjectIdentifier:
		return "OBJECT IDENTIFIER"
	case mib.BaseBits:
		return "BITS"
	case mib.BaseOpaque:
		return "Opaque"
	case mib.BaseSequence:
		return "OTHER"
	default:
		return base.String()
	}
}

// normalizeAccess converts a gomib Access to the normalized string used in fixtures.
func normalizeAccess(a mib.Access) string {
	return a.String()
}

// normalizeStatus converts a gomib Status to the normalized string used in fixtures.
func normalizeStatus(s mib.Status) string {
	return s.String()
}

// normalizeKind converts a gomib Kind to the normalized string used in fixtures.
func normalizeKind(k mib.Kind) string {
	return k.String()
}

// normalizeEnums converts gomib NamedValue slice to the map[int]string format used in fixtures.
func normalizeEnums(nvs []mib.NamedValue) map[int]string {
	if len(nvs) == 0 {
		return nil
	}
	m := make(map[int]string, len(nvs))
	for _, nv := range nvs {
		m[int(nv.Value)] = nv.Label
	}
	return m
}

// normalizeRanges converts gomib Range slice to the RangeInfo format used in fixtures.
func normalizeRanges(rs []mib.Range) []testutil.RangeInfo {
	if len(rs) == 0 {
		return nil
	}
	result := make([]testutil.RangeInfo, 0, len(rs))
	for _, r := range rs {
		low, lowOK := r.Min.AsInt64()
		high, highOK := r.Max.AsInt64()
		if lowOK && highOK {
			result = append(result, testutil.RangeInfo{Low: low, High: high})
		}
	}
	return result
}

// normalizeIndexes converts gomib IndexEntry slice to the IndexInfo format used in fixtures.
func normalizeIndexes(entries []mib.IndexEntry) []testutil.IndexInfo {
	if len(entries) == 0 {
		return nil
	}
	result := make([]testutil.IndexInfo, 0, len(entries))
	for _, e := range entries {
		if e.Object != nil {
			result = append(result, testutil.IndexInfo{
				Name:    e.Object.Name(),
				Implied: e.Implied,
			})
		} else if e.TypeName != "" {
			result = append(result, testutil.IndexInfo{
				Name:    e.TypeName,
				Implied: e.Implied,
			})
		}
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

// normalizeNodeType converts a gomib Node to the NodeType string used in fixtures.
// net-snmp uses the data type string for OBJECT-TYPE nodes and the macro type
// string for all others.
func normalizeNodeType(node *mib.Node) string {
	if node == nil {
		return ""
	}
	if obj := node.Object(); obj != nil {
		if obj.Type() == nil {
			// Table/row objects have SEQUENCE syntax which doesn't resolve
			// to a data type. Net-snmp reports OTHER for these.
			return "OTHER"
		}
		return normalizeType(obj.Type())
	}
	if node.Notification() != nil {
		return "NOTIFICATION-TYPE"
	}
	if g := node.Group(); g != nil {
		if g.IsNotificationGroup() {
			return "NOTIFICATION-GROUP"
		}
		return "OBJECT-GROUP"
	}
	if node.Compliance() != nil {
		return "MODULE-COMPLIANCE"
	}
	if node.Capability() != nil {
		return "AGENT-CAPABILITIES"
	}
	// MODULE-IDENTITY: the module's OID matches this node's OID.
	if mod := node.Module(); mod != nil {
		modOID := mod.OID()
		nodeOID := node.OID()
		if len(modOID) > 0 && slices.Equal(modOID, nodeOID) {
			return "MODULE-IDENTITY"
		}
	}
	return "OTHER"
}

// normalizeVarbinds converts gomib Object slice (notification OBJECTS) to name strings.
func normalizeVarbinds(objects []*mib.Object) []string {
	if len(objects) == 0 {
		return nil
	}
	result := make([]string, len(objects))
	for i, obj := range objects {
		result[i] = obj.Name()
	}
	return result
}

// formatEnums formats an enum map as a human-readable string for error messages.
func formatEnums(enums map[int]string) string {
	if len(enums) == 0 {
		return "{}"
	}
	var keys []int
	for k := range enums {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	var parts []string
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s(%d)", enums[k], k))
	}
	return "{ " + strings.Join(parts, ", ") + " }"
}

// formatRanges formats a range list as a human-readable string for error messages.
func formatRanges(ranges []testutil.RangeInfo) string {
	if len(ranges) == 0 {
		return "()"
	}
	var parts []string
	for _, r := range ranges {
		if r.Low == r.High {
			parts = append(parts, strconv.FormatInt(r.Low, 10))
		} else {
			parts = append(parts, fmt.Sprintf("%d..%d", r.Low, r.High))
		}
	}
	return "(" + strings.Join(parts, " | ") + ")"
}

// formatIndexes formats an index list as a human-readable string for error messages.
func formatIndexes(indexes []testutil.IndexInfo) string {
	if len(indexes) == 0 {
		return ""
	}
	var parts []string
	for _, idx := range indexes {
		if idx.Implied {
			parts = append(parts, "IMPLIED "+idx.Name)
		} else {
			parts = append(parts, idx.Name)
		}
	}
	return "{ " + strings.Join(parts, ", ") + " }"
}
