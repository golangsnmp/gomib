package gomib

// Equivalence helpers for comparing gomib output against net-snmp
// ground-truth fixtures, accounting for known benign divergences.

import (
	"cmp"
	"slices"
	"strings"

	"github.com/golangsnmp/gomib/internal/testutil"
)

func typesEquivalent(gomibType, fixtureType string) bool {
	if gomibType == fixtureType {
		return true
	}
	return normalizeTypeName(gomibType) == normalizeTypeName(fixtureType)
}

func normalizeTypeName(t string) string {
	switch t {
	case "INTEGER", "Integer32":
		return "Integer32"
	case "COUNTER", "Counter", "Counter32":
		return "Counter32"
	case "GAUGE", "Gauge", "Gauge32":
		return "Gauge32"
	case "UNSIGNED32", "Unsigned32", "UInteger32":
		return "Unsigned32"
	case "TIMETICKS", "TimeTicks":
		return "TimeTicks"
	case "IPADDR", "IpAddress":
		return "IpAddress"
	case "OCTETSTR", "OCTET STRING", "OctetString":
		return "OCTET STRING"
	case "OBJID", "OBJECT IDENTIFIER", "ObjectIdentifier":
		return "OBJECT IDENTIFIER"
	case "COUNTER64", "Counter64":
		return "Counter64"
	case "BITS", "BITSTRING":
		return "BITS"
	case "OPAQUE", "Opaque":
		return "Opaque"
	default:
		return t
	}
}

// rangesEquivalent allows order differences and the signed/unsigned
// divergence where net-snmp displays unsigned values as signed
// (e.g., 4294967295 as -1).
func rangesEquivalent(gomibRanges, fixtureRanges []testutil.RangeInfo) bool {
	if len(gomibRanges) != len(fixtureRanges) {
		return false
	}
	if len(gomibRanges) == 0 {
		return true
	}

	g := make([]testutil.RangeInfo, len(gomibRanges))
	f := make([]testutil.RangeInfo, len(fixtureRanges))
	copy(g, gomibRanges)
	copy(f, fixtureRanges)
	sortRanges(g)
	sortRanges(f)

	for i := range g {
		if g[i] == f[i] {
			continue
		}
		if isSignedUnsignedEquivalent(g[i], f[i]) {
			continue
		}
		return false
	}
	return true
}

func sortRanges(rs []testutil.RangeInfo) {
	slices.SortFunc(rs, func(a, b testutil.RangeInfo) int {
		if c := cmp.Compare(a.Low, b.Low); c != 0 {
			return c
		}
		return cmp.Compare(a.High, b.High)
	})
}

func isSignedUnsignedEquivalent(gomibR, fixtureR testutil.RangeInfo) bool {
	return signedEquiv(gomibR.Low, fixtureR.Low) && signedEquiv(gomibR.High, fixtureR.High)
}

func signedEquiv(a, b int64) bool {
	if a == b {
		return true
	}
	if a >= 0 && b < 0 && a == b+1<<32 {
		return true
	}
	if b >= 0 && a < 0 && b == a+1<<32 {
		return true
	}
	return false
}

func enumsEquivalent(gomibEnums, fixtureEnums map[int]string) bool {
	if len(gomibEnums) == 0 && len(fixtureEnums) == 0 {
		return true
	}
	if len(gomibEnums) != len(fixtureEnums) {
		return false
	}
	for k, v := range fixtureEnums {
		if gomibEnums[k] != v {
			return false
		}
	}
	return true
}

func hintsEquivalent(gomibHint, fixtureHint string) bool {
	if gomibHint == fixtureHint {
		return true
	}
	return strings.EqualFold(strings.TrimSpace(gomibHint), strings.TrimSpace(fixtureHint))
}

func indexesEquivalent(gomibIndexes, fixtureIndexes []testutil.IndexInfo) bool {
	if len(gomibIndexes) != len(fixtureIndexes) {
		return false
	}
	for i := range gomibIndexes {
		if gomibIndexes[i].Name != fixtureIndexes[i].Name {
			return false
		}
		if gomibIndexes[i].Implied != fixtureIndexes[i].Implied {
			return false
		}
	}
	return true
}

func varbindsEquivalent(gomibVarbinds, fixtureVarbinds []string) bool {
	if len(gomibVarbinds) != len(fixtureVarbinds) {
		return false
	}
	for i := range gomibVarbinds {
		if gomibVarbinds[i] != fixtureVarbinds[i] {
			return false
		}
	}
	return true
}

// accessEquivalent treats read-write and read-create as equivalent
// only for SMIv1 MIBs, where read-create did not exist.
// For SMIv2 MIBs, read-write and read-create are distinct values.
func accessEquivalent(gomibAccess, fixtureAccess string, isSMIv1 bool) bool {
	if gomibAccess == fixtureAccess {
		return true
	}
	if isSMIv1 {
		if (gomibAccess == "read-write" && fixtureAccess == "read-create") ||
			(gomibAccess == "read-create" && fixtureAccess == "read-write") {
			return true
		}
	}
	return false
}

// statusEquivalent treats mandatory and current as equivalent
// (SMIv1 vs SMIv2 difference).
func statusEquivalent(gomibStatus, fixtureStatus string) bool {
	if gomibStatus == fixtureStatus {
		return true
	}
	if (gomibStatus == "mandatory" && fixtureStatus == "current") ||
		(gomibStatus == "current" && fixtureStatus == "mandatory") {
		return true
	}
	return false
}

// defvalEquivalent accounts for representation differences: quoting,
// hex zero bytes (0x00000000... vs 0), and hex all-ones (0xFFFF... vs -1).
// The hex-ones case is the same root cause as the range signed/unsigned
// divergence: net-snmp interprets the value through a C int, which
// overflows to -1 for all-ones bit patterns.
func defvalEquivalent(gomibDefval, fixtureDefval string) bool {
	if gomibDefval == fixtureDefval {
		return true
	}
	gNorm := strings.Trim(strings.TrimSpace(gomibDefval), "\"'")
	fNorm := strings.Trim(strings.TrimSpace(fixtureDefval), "\"'")
	if gNorm == fNorm {
		return true
	}
	if isHexZeros(gNorm) && fNorm == "0" || isHexZeros(fNorm) && gNorm == "0" {
		return true
	}
	if isHexAllOnes(gNorm) && fNorm == "-1" || isHexAllOnes(fNorm) && gNorm == "-1" {
		return true
	}
	return false
}

func isHexZeros(s string) bool {
	if !strings.HasPrefix(s, "0x") && !strings.HasPrefix(s, "0X") {
		return false
	}
	for _, c := range s[2:] {
		if c != '0' {
			return false
		}
	}
	return len(s) > 2
}

func isHexAllOnes(s string) bool {
	if !strings.HasPrefix(s, "0x") && !strings.HasPrefix(s, "0X") {
		return false
	}
	for _, c := range strings.ToUpper(s[2:]) {
		if c != 'F' {
			return false
		}
	}
	return len(s) > 2
}

func referenceEquivalent(gomibRef, fixtureRef string) bool {
	if gomibRef == fixtureRef {
		return true
	}
	return normalizeWhitespace(gomibRef) == normalizeWhitespace(fixtureRef)
}

func normalizeWhitespace(s string) string {
	fields := strings.Fields(s)
	return strings.Join(fields, " ")
}
