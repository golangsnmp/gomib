package gomib

import (
	"fmt"
	"slices"
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
	result := make([]testutil.RangeInfo, len(rs))
	for i, r := range rs {
		result[i] = testutil.RangeInfo{Low: r.Min, High: r.Max}
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
		}
	}
	if len(result) == 0 {
		return nil
	}
	return result
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
			parts = append(parts, fmt.Sprintf("%d", r.Low))
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
