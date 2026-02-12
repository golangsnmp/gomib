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
// (SMIv1 vs SMIv2 difference).
func accessEquivalent(gomibAccess, fixtureAccess string) bool {
	if gomibAccess == fixtureAccess {
		return true
	}
	if (gomibAccess == "read-write" && fixtureAccess == "read-create") ||
		(gomibAccess == "read-create" && fixtureAccess == "read-write") {
		return true
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
// hex zero bytes (0x00000000... vs 0), and OID symbolic names
// (zeroDotZero vs 0.0).
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
	if oidDefvalEquivalent(gNorm, fNorm) {
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

func oidDefvalEquivalent(a, b string) bool {
	known := map[string]string{
		"zeroDotZero": "0.0",
	}
	for sym, num := range known {
		if (a == sym && b == num) || (a == num && b == sym) {
			return true
		}
	}
	return false
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
