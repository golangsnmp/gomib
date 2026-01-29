package gomib

// divergences_test.go documents known benign differences between gomib and
// net-snmp ground-truth fixtures, along with equivalence helpers used by
// the ground-truth test suite.

import (
	"sort"
	"strings"

	"github.com/golangsnmp/gomib/internal/testutil"
)

// typesEquivalent checks if gomib and fixture type strings are semantically equivalent.
// Handles naming differences between net-snmp and gomib representations.
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

// rangesEquivalent compares two range lists allowing for order differences
// and the known signed/unsigned divergence where net-snmp displays unsigned
// values as signed (e.g., 4294967295 as -1).
func rangesEquivalent(gomibRanges, fixtureRanges []testutil.RangeInfo) bool {
	if len(gomibRanges) != len(fixtureRanges) {
		return false
	}
	if len(gomibRanges) == 0 {
		return true
	}

	// Sort both for order-independent comparison
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
		// Known divergence: signed/unsigned representation
		// net-snmp may show large unsigned values as negative signed values
		if isSignedUnsignedEquivalent(g[i], f[i]) {
			continue
		}
		return false
	}
	return true
}

func sortRanges(rs []testutil.RangeInfo) {
	sort.Slice(rs, func(i, j int) bool {
		if rs[i].Low != rs[j].Low {
			return rs[i].Low < rs[j].Low
		}
		return rs[i].High < rs[j].High
	})
}

// isSignedUnsignedEquivalent checks if a range difference is due to
// signed vs unsigned interpretation. net-snmp sometimes shows unsigned
// values (like 4294967295) as signed (-1).
func isSignedUnsignedEquivalent(gomibR, fixtureR testutil.RangeInfo) bool {
	return signedEquiv(gomibR.Low, fixtureR.Low) && signedEquiv(gomibR.High, fixtureR.High)
}

func signedEquiv(a, b int64) bool {
	if a == b {
		return true
	}
	// Check 32-bit signed/unsigned wrap
	if a >= 0 && b < 0 && a == b+1<<32 {
		return true
	}
	if b >= 0 && a < 0 && b == a+1<<32 {
		return true
	}
	return false
}

// enumsEquivalent compares two enum maps, accounting for the known
// import-shadowing divergence where gomib may resolve a different set of
// enum values than net-snmp for objects using imported textual conventions.
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

// hintsEquivalent checks if two display hints are semantically equivalent,
// accounting for whitespace and case differences.
func hintsEquivalent(gomibHint, fixtureHint string) bool {
	if gomibHint == fixtureHint {
		return true
	}
	return strings.EqualFold(strings.TrimSpace(gomibHint), strings.TrimSpace(fixtureHint))
}

// indexesEquivalent compares two index lists.
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

// varbindsEquivalent compares two varbind (OBJECTS) lists.
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

// accessEquivalent compares access levels, treating read-write/read-create
// as equivalent (SMIv1 vs SMIv2 difference).
func accessEquivalent(gomibAccess, fixtureAccess string) bool {
	if gomibAccess == fixtureAccess {
		return true
	}
	// SMIv1 read-write is equivalent to SMIv2 read-create in many cases
	if (gomibAccess == "read-write" && fixtureAccess == "read-create") ||
		(gomibAccess == "read-create" && fixtureAccess == "read-write") {
		return true
	}
	return false
}

// statusEquivalent compares status values, treating mandatory/current
// as equivalent (SMIv1 vs SMIv2 difference).
func statusEquivalent(gomibStatus, fixtureStatus string) bool {
	if gomibStatus == fixtureStatus {
		return true
	}
	// SMIv1 mandatory is equivalent to SMIv2 current
	if (gomibStatus == "mandatory" && fixtureStatus == "current") ||
		(gomibStatus == "current" && fixtureStatus == "mandatory") {
		return true
	}
	return false
}

// defvalEquivalent compares default value strings, accounting for known
// representation differences between gomib and net-snmp:
//   - quoting differences ("" vs \"\")
//   - hex zero bytes (0x00000000... vs 0)
//   - OID symbolic names (zeroDotZero vs 0.0)
func defvalEquivalent(gomibDefval, fixtureDefval string) bool {
	if gomibDefval == fixtureDefval {
		return true
	}
	// Normalize: strip quotes, whitespace
	gNorm := strings.Trim(strings.TrimSpace(gomibDefval), "\"'")
	fNorm := strings.Trim(strings.TrimSpace(fixtureDefval), "\"'")
	if gNorm == fNorm {
		return true
	}
	// Hex zeros: 0x0000... == 0
	if isHexZeros(gNorm) && fNorm == "0" || isHexZeros(fNorm) && gNorm == "0" {
		return true
	}
	// OID symbolic equivalence
	if oidDefvalEquivalent(gNorm, fNorm) {
		return true
	}
	return false
}

// isHexZeros checks if a string is 0x followed by only zeroes.
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

// oidDefvalEquivalent checks if one value is a well-known OID symbolic name
// and the other is its numeric form.
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

// referenceEquivalent compares REFERENCE clause strings, accounting for
// whitespace normalization differences.
func referenceEquivalent(gomibRef, fixtureRef string) bool {
	if gomibRef == fixtureRef {
		return true
	}
	return normalizeWhitespace(gomibRef) == normalizeWhitespace(fixtureRef)
}

// normalizeWhitespace collapses runs of whitespace to single spaces and trims.
func normalizeWhitespace(s string) string {
	fields := strings.Fields(s)
	return strings.Join(fields, " ")
}
