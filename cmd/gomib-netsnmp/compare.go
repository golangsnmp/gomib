//go:build cgo

//nolint:errcheck // CLI output, errors not critical
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
)

// ComparisonResult holds the results of comparing gomib against net-snmp.
type ComparisonResult struct {
	TotalNetSnmp     int         `json:"total_netsnmp"`
	TotalGomib       int         `json:"total_gomib"`
	MatchedNodes     int         `json:"matched_nodes"`
	Mismatches       []Mismatch  `json:"mismatches,omitempty"`
	MissingInGomib   []string    `json:"missing_in_gomib,omitempty"`
	MissingInNetSnmp []string    `json:"missing_in_netsnmp,omitempty"`
	Summary          FieldCounts `json:"summary"`
}

// Mismatch describes a difference between gomib and net-snmp.
type Mismatch struct {
	OID          string `json:"oid"`
	Name         string `json:"name"`
	Module       string `json:"module"`
	Field        string `json:"field"`
	Gomib        string `json:"gomib"`
	NetSnmp      string `json:"netsnmp"`
	GomibModule  string `json:"gomib_module,omitempty"`  // Module per gomib (for overlap detection)
	NetSnmpModule string `json:"netsnmp_module,omitempty"` // Module per net-snmp (for overlap detection)
}

// FieldCounts tracks match/mismatch counts per field.
type FieldCounts struct {
	Type         CountPair `json:"type"`
	Access       CountPair `json:"access"`
	Status       CountPair `json:"status"`
	Enums        CountPair `json:"enums"`
	Index        CountPair `json:"index"`
	Hint         CountPair `json:"hint"`
	TCName       CountPair `json:"tc_name"`
	Units        CountPair `json:"units"`
	Ranges       CountPair `json:"ranges"`
	DefaultValue CountPair `json:"default_value"`
	Bits         CountPair `json:"bits"`
	Varbinds     CountPair `json:"varbinds"`
}

// CountPair holds match and mismatch counts.
type CountPair struct {
	Match    int `json:"match"`
	Mismatch int `json:"mismatch"`
}

func cmdCompare(args []string) int {
	fs := flag.NewFlagSet("compare", flag.ExitOnError)

	var fieldFilter string
	var exampleLimit int
	var categorize bool
	var investigateOnly bool

	fs.StringVar(&fieldFilter, "field", "", "Show only mismatches for this field (type, access, status, enums, index, hint, tc_name, units, ranges, defval, bits, varbinds)")
	fs.IntVar(&exampleLimit, "limit", 5, "Number of examples to show per category (0 for all)")
	fs.BoolVar(&categorize, "categorize", false, "Categorize mismatches by likely cause")
	fs.BoolVar(&investigateOnly, "investigate", false, "Only show mismatches that need investigation (hide known benign differences)")

	fs.Usage = func() {
		fmt.Fprintf(fs.Output(), `Usage: gomib-netsnmp compare [options] [MODULE...]

Compares all nodes between gomib and net-snmp:
- OID resolution
- Type mapping
- Access levels
- Enum values
- Index structures
- AUGMENTS relationships

Options:
`)
		fs.PrintDefaults()
	}

	if err := fs.Parse(args); err != nil {
		return 1
	}

	modules := fs.Args()
	mibPaths := getMIBPaths()

	out, cleanup, err := getOutput()
	if err != nil {
		printError("cannot open output: %v", err)
		return 1
	}
	defer cleanup()

	// Progress to stderr, results to out
	fmt.Fprintln(os.Stderr, "Loading MIBs with net-snmp...")
	netsnmpNodes, err := loadNetSnmpNodes(mibPaths, modules)
	if err != nil {
		printError("net-snmp load failed: %v", err)
		return 1
	}

	fmt.Fprintln(os.Stderr, "Loading MIBs with gomib...")
	gomibNodes, err := loadGomibNodes(mibPaths, modules)
	if err != nil {
		printError("gomib load failed: %v", err)
		return 1
	}

	// Filter by modules if specified
	if len(modules) > 0 {
		netsnmpNodes = filterByModules(netsnmpNodes, modules)
		gomibNodes = filterByModules(gomibNodes, modules)
	}

	fmt.Fprintf(os.Stderr, "net-snmp: %d nodes, gomib: %d nodes\n", len(netsnmpNodes), len(gomibNodes))

	result := compareNodes(netsnmpNodes, gomibNodes)

	// Filter mismatches by field if requested
	if fieldFilter != "" {
		var filtered []Mismatch
		for _, m := range result.Mismatches {
			if m.Field == fieldFilter {
				filtered = append(filtered, m)
			}
		}
		result.Mismatches = filtered
	}

	if jsonOutput {
		enc := json.NewEncoder(out)
		enc.SetIndent("", "  ")
		if err := enc.Encode(result); err != nil {
			printError("json encode failed: %v", err)
			return 1
		}
	} else {
		printComparisonResult(out, result, exampleLimit, categorize, investigateOnly)
	}

	return 0
}

// makeMismatch creates a Mismatch, including module diff info if modules differ.
func makeMismatch(oid, field, gomibVal, netsnmpVal string, gNode, nsNode *NormalizedNode) Mismatch {
	m := Mismatch{
		OID:     oid,
		Name:    nsNode.Name,
		Module:  nsNode.Module,
		Field:   field,
		Gomib:   gomibVal,
		NetSnmp: netsnmpVal,
	}
	if gNode.Module != nsNode.Module {
		m.GomibModule = gNode.Module
		m.NetSnmpModule = nsNode.Module
	}
	return m
}

// compareNodes performs a full comparison between net-snmp and gomib nodes.
func compareNodes(netsnmp, gomib map[string]*NormalizedNode) *ComparisonResult {
	result := &ComparisonResult{
		TotalNetSnmp: len(netsnmp),
		TotalGomib:   len(gomib),
	}

	// Find all OIDs
	allOIDs := make(map[string]bool)
	for oid := range netsnmp {
		allOIDs[oid] = true
	}
	for oid := range gomib {
		allOIDs[oid] = true
	}

	for oid := range allOIDs {
		nsNode := netsnmp[oid]
		gNode := gomib[oid]

		if nsNode == nil {
			result.MissingInNetSnmp = append(result.MissingInNetSnmp, oid)
			continue
		}
		if gNode == nil {
			result.MissingInGomib = append(result.MissingInGomib, oid)
			continue
		}

		result.MatchedNodes++

		// Compare type (using normalized forms for semantic equivalence)
		if nsNode.Type != "" && nsNode.Type != "OTHER" && nsNode.Type != "UNKNOWN" {
			if typesEquivalent(gNode.Type, nsNode.Type) {
				result.Summary.Type.Match++
			} else if gNode.Type != "" {
				result.Summary.Type.Mismatch++
				result.Mismatches = append(result.Mismatches, makeMismatch(oid, "type", gNode.Type, nsNode.Type, gNode, nsNode))
			}
		}

		// Compare access
		if nsNode.Access != "" {
			if gNode.Access == nsNode.Access {
				result.Summary.Access.Match++
			} else if gNode.Access != "" {
				result.Summary.Access.Mismatch++
				result.Mismatches = append(result.Mismatches, makeMismatch(oid, "access", gNode.Access, nsNode.Access, gNode, nsNode))
			}
		}

		// Compare status
		if nsNode.Status != "" {
			if gNode.Status == nsNode.Status {
				result.Summary.Status.Match++
			} else if gNode.Status != "" {
				result.Summary.Status.Mismatch++
				result.Mismatches = append(result.Mismatches, makeMismatch(oid, "status", gNode.Status, nsNode.Status, gNode, nsNode))
			}
		}

		// Compare enums
		if len(nsNode.EnumValues) > 0 {
			if enumsEqual(nsNode.EnumValues, gNode.EnumValues) {
				result.Summary.Enums.Match++
			} else {
				result.Summary.Enums.Mismatch++
				result.Mismatches = append(result.Mismatches, makeMismatch(oid, "enums", formatEnums(gNode.EnumValues), formatEnums(nsNode.EnumValues), gNode, nsNode))
			}
		}

		// Compare indexes
		if len(nsNode.Indexes) > 0 {
			if indexesEqual(nsNode.Indexes, gNode.Indexes) {
				result.Summary.Index.Match++
			} else {
				result.Summary.Index.Mismatch++
				result.Mismatches = append(result.Mismatches, makeMismatch(oid, "index", indexString(gNode.Indexes), indexString(nsNode.Indexes), gNode, nsNode))
			}
		}

		// Compare display hint
		if nsNode.Hint != "" {
			if hintsEquivalent(gNode.Hint, nsNode.Hint) {
				result.Summary.Hint.Match++
			} else if gNode.Hint != "" {
				result.Summary.Hint.Mismatch++
				result.Mismatches = append(result.Mismatches, makeMismatch(oid, "hint", gNode.Hint, nsNode.Hint, gNode, nsNode))
			}
		}

		// Compare TC name
		if nsNode.TCName != "" {
			if gNode.TCName == nsNode.TCName {
				result.Summary.TCName.Match++
			} else if gNode.TCName != "" {
				result.Summary.TCName.Mismatch++
				result.Mismatches = append(result.Mismatches, makeMismatch(oid, "tc_name", gNode.TCName, nsNode.TCName, gNode, nsNode))
			}
		}

		// Compare units
		if nsNode.Units != "" {
			if gNode.Units == nsNode.Units {
				result.Summary.Units.Match++
			} else if gNode.Units != "" {
				result.Summary.Units.Mismatch++
				result.Mismatches = append(result.Mismatches, makeMismatch(oid, "units", gNode.Units, nsNode.Units, gNode, nsNode))
			}
		}

		// Compare ranges
		if len(nsNode.Ranges) > 0 {
			if rangesEqual(nsNode.Ranges, gNode.Ranges) {
				result.Summary.Ranges.Match++
			} else {
				result.Summary.Ranges.Mismatch++
				result.Mismatches = append(result.Mismatches, makeMismatch(oid, "ranges", rangesString(gNode.Ranges), rangesString(nsNode.Ranges), gNode, nsNode))
			}
		}

		// Compare default value
		if nsNode.DefaultValue != "" {
			if defaultValuesEquivalent(gNode.DefaultValue, nsNode.DefaultValue) {
				result.Summary.DefaultValue.Match++
			} else if gNode.DefaultValue != "" {
				result.Summary.DefaultValue.Mismatch++
				result.Mismatches = append(result.Mismatches, makeMismatch(oid, "defval", gNode.DefaultValue, nsNode.DefaultValue, gNode, nsNode))
			}
		}

		// Compare BITS values
		if len(nsNode.BitValues) > 0 {
			if enumsEqual(nsNode.BitValues, gNode.BitValues) {
				result.Summary.Bits.Match++
			} else {
				result.Summary.Bits.Mismatch++
				result.Mismatches = append(result.Mismatches, makeMismatch(oid, "bits", bitsString(gNode.BitValues), bitsString(nsNode.BitValues), gNode, nsNode))
			}
		}

		// Compare varbinds (notification OBJECTS)
		if len(nsNode.Varbinds) > 0 {
			if varbindsEqual(nsNode.Varbinds, gNode.Varbinds) {
				result.Summary.Varbinds.Match++
			} else {
				result.Summary.Varbinds.Mismatch++
				result.Mismatches = append(result.Mismatches, makeMismatch(oid, "varbinds", varbindsString(gNode.Varbinds), varbindsString(nsNode.Varbinds), gNode, nsNode))
			}
		}
	}

	// Sort missing lists for deterministic output
	sort.Strings(result.MissingInGomib)
	sort.Strings(result.MissingInNetSnmp)

	return result
}

// typesEquivalent checks if two type names are semantically equivalent.
// Handles differences in naming conventions between net-snmp and gomib.
func typesEquivalent(a, b string) bool {
	if a == b {
		return true
	}
	return normalizeTypeName(a) == normalizeTypeName(b)
}

// normalizeTypeName maps type names to canonical forms for comparison.
func normalizeTypeName(t string) string {
	switch t {
	// INTEGER and Integer32 are semantically equivalent
	case "INTEGER", "Integer32":
		return "Integer32"
	// Counter and Counter32 are equivalent
	case "COUNTER", "Counter", "Counter32":
		return "Counter32"
	// Gauge and Gauge32 are equivalent
	case "GAUGE", "Gauge", "Gauge32":
		return "Gauge32"
	// Unsigned32 variations
	case "UNSIGNED32", "Unsigned32", "UInteger32":
		return "Unsigned32"
	// TimeTicks variations
	case "TIMETICKS", "TimeTicks":
		return "TimeTicks"
	// IpAddress variations
	case "IPADDR", "IpAddress":
		return "IpAddress"
	// OctetString variations
	case "OCTETSTR", "OCTET STRING", "OctetString":
		return "OCTET STRING"
	// ObjectIdentifier variations
	case "OBJID", "OBJECT IDENTIFIER", "ObjectIdentifier":
		return "OBJECT IDENTIFIER"
	// Counter64 variations
	case "COUNTER64", "Counter64":
		return "Counter64"
	// BITS variations
	case "BITS", "BITSTRING":
		return "BITS"
	// Opaque variations
	case "OPAQUE", "Opaque":
		return "Opaque"
	default:
		return t
	}
}

func enumsEqual(a, b map[int]string) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if b[k] != v {
			return false
		}
	}
	return true
}

func indexesEqual(a, b []IndexInfo) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].Name != b[i].Name || a[i].Implied != b[i].Implied {
			return false
		}
	}
	return true
}

// hintsEquivalent checks if two display hints are semantically equivalent.
func hintsEquivalent(a, b string) bool {
	if a == b {
		return true
	}
	// Normalize common variations (whitespace, case for hex digits)
	return strings.EqualFold(strings.TrimSpace(a), strings.TrimSpace(b))
}

// rangesEqual checks if two range lists are equivalent.
func rangesEqual(a, b []RangeInfo) bool {
	if len(a) != len(b) {
		return false
	}
	// Sort both for comparison (order may differ)
	aCopy := make([]RangeInfo, len(a))
	bCopy := make([]RangeInfo, len(b))
	copy(aCopy, a)
	copy(bCopy, b)
	sort.Slice(aCopy, func(i, j int) bool {
		if aCopy[i].Low != aCopy[j].Low {
			return aCopy[i].Low < aCopy[j].Low
		}
		return aCopy[i].High < aCopy[j].High
	})
	sort.Slice(bCopy, func(i, j int) bool {
		if bCopy[i].Low != bCopy[j].Low {
			return bCopy[i].Low < bCopy[j].Low
		}
		return bCopy[i].High < bCopy[j].High
	})
	for i := range aCopy {
		if aCopy[i].Low != bCopy[i].Low || aCopy[i].High != bCopy[i].High {
			return false
		}
	}
	return true
}

// defaultValuesEquivalent checks if two default values are semantically equivalent.
func defaultValuesEquivalent(a, b string) bool {
	if a == b {
		return true
	}
	// Normalize: strip quotes, whitespace
	aNorm := strings.Trim(strings.TrimSpace(a), "\"'")
	bNorm := strings.Trim(strings.TrimSpace(b), "\"'")
	if aNorm == bNorm {
		return true
	}

	// Check hex zeros equivalence: 0x0000... == 0
	if hexZerosEquivalent(aNorm, bNorm) {
		return true
	}

	// Check hex all-ones equivalence: 0xFFFF... == -1 (signed interpretation)
	if hexOnesEquivalent(aNorm, bNorm) {
		return true
	}

	// Check OID symbolic equivalence: 0.0 == zeroDotZero, etc.
	if oidSymbolicEquivalent(aNorm, bNorm) {
		return true
	}

	return false
}

// hexZerosEquivalent checks if one value is hex zeros and the other is "0".
func hexZerosEquivalent(a, b string) bool {
	return (isHexZeros(a) && b == "0") || (isHexZeros(b) && a == "0")
}

// hexOnesEquivalent checks if one value is hex all-ones (0xFFF...) and the other is "-1".
func hexOnesEquivalent(a, b string) bool {
	return (isHexAllOnes(a) && b == "-1") || (isHexAllOnes(b) && a == "-1")
}

// isHexAllOnes checks if a string is 0x followed by only F's (all bits set).
func isHexAllOnes(s string) bool {
	if !strings.HasPrefix(s, "0x") && !strings.HasPrefix(s, "0X") {
		return false
	}
	hex := strings.ToUpper(s[2:])
	if len(hex) == 0 {
		return false
	}
	for _, c := range hex {
		if c != 'F' {
			return false
		}
	}
	return true
}

// isHexZeros checks if a string is 0x followed by only zeros.
func isHexZeros(s string) bool {
	if !strings.HasPrefix(s, "0x") && !strings.HasPrefix(s, "0X") {
		return false
	}
	hex := s[2:]
	if len(hex) == 0 {
		return false
	}
	for _, c := range hex {
		if c != '0' {
			return false
		}
	}
	return true
}

// oidSymbolicEquivalent checks if two OID representations are equivalent.
// Handles numeric OID vs well-known symbolic names.
func oidSymbolicEquivalent(a, b string) bool {
	// Well-known OID symbolic names and their numeric equivalents
	knownOIDs := map[string]string{
		"zeroDotZero":          "0.0",
		"snmpUDPDomain":        "1.3.6.1.6.1.1",
		"usmNoAuthProtocol":    "1.3.6.1.6.3.10.1.1.1",
		"usmNoPrivProtocol":    "1.3.6.1.6.3.10.1.2.1",
		"usmHMACMD5AuthProtocol": "1.3.6.1.6.3.10.1.1.2",
		"usmHMACSHAAuthProtocol": "1.3.6.1.6.3.10.1.1.3",
		"usmDESPrivProtocol":   "1.3.6.1.6.3.10.1.2.2",
		"pingIcmpEcho":         "1.3.6.1.2.1.80.3.1",
		"traceRouteUsingUdpProbes": "1.3.6.1.2.1.81.3.1",
		"sysUpTimeInstance":    "1.3.6.1.2.1.1.3.0",
	}

	// Check if one is a known symbol and the other is its numeric form
	if numeric, ok := knownOIDs[a]; ok && numeric == b {
		return true
	}
	if numeric, ok := knownOIDs[b]; ok && numeric == a {
		return true
	}

	// Check pattern: if one looks like a numeric OID and the other is a symbol
	// that might resolve to it (heuristic for vendor-specific OIDs)
	aIsNumeric := isNumericOID(a)
	bIsNumeric := isNumericOID(b)

	if aIsNumeric && !bIsNumeric && isLikelyOIDSymbol(b) {
		// a is numeric, b is symbolic - likely equivalent
		return true
	}
	if bIsNumeric && !aIsNumeric && isLikelyOIDSymbol(a) {
		// b is numeric, a is symbolic - likely equivalent
		return true
	}

	return false
}

// isNumericOID checks if a string looks like a numeric OID (e.g., "1.3.6.1.2.1").
func isNumericOID(s string) bool {
	if s == "" {
		return false
	}
	for _, c := range s {
		if c != '.' && (c < '0' || c > '9') {
			return false
		}
	}
	// Must have at least one dot to be an OID (not just an integer)
	return strings.Contains(s, ".")
}

// isLikelyOIDSymbol checks if a string looks like an OID symbolic name.
// OID symbols are typically camelCase or contain specific patterns.
func isLikelyOIDSymbol(s string) bool {
	if s == "" {
		return false
	}
	// Must start with a letter
	if (s[0] < 'a' || s[0] > 'z') && (s[0] < 'A' || s[0] > 'Z') {
		return false
	}
	// Should not contain dots (that would be a numeric OID)
	if strings.Contains(s, ".") {
		return false
	}
	// Should be alphanumeric (possibly with some special chars)
	for _, c := range s {
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '-' || c == '_') {
			return false
		}
	}
	return true
}

// varbindsEqual checks if two varbind lists are equivalent.
func varbindsEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func formatEnums(enums map[int]string) string {
	if len(enums) == 0 {
		return "{}"
	}

	var keys []int
	for k := range enums {
		keys = append(keys, k)
	}
	sort.Ints(keys)

	var parts []string
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s(%d)", enums[k], k))
	}
	return "{ " + strings.Join(parts, ", ") + " }"
}

func printComparisonResult(w io.Writer, result *ComparisonResult, exampleLimit int, categorize bool, investigateOnly bool) {
	fmt.Fprintln(w, strings.Repeat("=", 70))
	fmt.Fprintln(w, "GOMIB vs NET-SNMP COMPARISON RESULTS")
	fmt.Fprintln(w, strings.Repeat("=", 70))

	fmt.Fprintf(w, "\nNode counts:\n")
	fmt.Fprintf(w, "  net-snmp nodes:      %6d\n", result.TotalNetSnmp)
	fmt.Fprintf(w, "  gomib nodes:         %6d\n", result.TotalGomib)
	fmt.Fprintf(w, "  common nodes:        %6d\n", result.MatchedNodes)
	fmt.Fprintf(w, "  missing in gomib:    %6d\n", len(result.MissingInGomib))
	fmt.Fprintf(w, "  missing in net-snmp: %6d\n", len(result.MissingInNetSnmp))

	fmt.Fprintf(w, "\nField accuracy (for common nodes):\n")
	printFieldAccuracy(w, "type", result.Summary.Type)
	printFieldAccuracy(w, "access", result.Summary.Access)
	printFieldAccuracy(w, "status", result.Summary.Status)
	printFieldAccuracy(w, "enums", result.Summary.Enums)
	printFieldAccuracy(w, "index", result.Summary.Index)
	printFieldAccuracy(w, "hint", result.Summary.Hint)
	printFieldAccuracy(w, "tc_name", result.Summary.TCName)
	printFieldAccuracy(w, "units", result.Summary.Units)
	printFieldAccuracy(w, "ranges", result.Summary.Ranges)
	printFieldAccuracy(w, "defval", result.Summary.DefaultValue)
	printFieldAccuracy(w, "bits", result.Summary.Bits)
	printFieldAccuracy(w, "varbinds", result.Summary.Varbinds)

	if len(result.Mismatches) > 0 {
		// Count benign vs investigate
		benign, investigate := countBenignAndInvestigate(result.Mismatches)
		fmt.Fprintf(w, "\nMismatch classification:\n")
		fmt.Fprintf(w, "  total:       %6d\n", len(result.Mismatches))
		fmt.Fprintf(w, "  benign:      %6d  (known representation differences)\n", benign)
		fmt.Fprintf(w, "  investigate: %6d  (potential real issues)\n", investigate)

		// Group mismatches by field type
		byField := make(map[string][]Mismatch)
		for _, m := range result.Mismatches {
			byField[m.Field] = append(byField[m.Field], m)
		}

		fieldOrder := []string{"type", "access", "status", "enums", "index", "hint", "tc_name", "units", "ranges", "defval", "bits", "varbinds"}

		if categorize || investigateOnly {
			if investigateOnly {
				fmt.Fprintf(w, "\nMismatches needing investigation:\n")
			} else {
				fmt.Fprintf(w, "\nMismatches by field and category:\n")
			}
			for _, field := range fieldOrder {
				mismatches, ok := byField[field]
				if !ok || len(mismatches) == 0 {
					continue
				}
				printCategorizedMismatches(w, field, mismatches, exampleLimit, investigateOnly)
			}
		} else {
			limitStr := fmt.Sprintf("up to %d each", exampleLimit)
			if exampleLimit == 0 {
				limitStr = "all"
			}
			fmt.Fprintf(w, "\nMismatches by field (%s):\n", limitStr)
			for _, field := range fieldOrder {
				mismatches, ok := byField[field]
				if !ok || len(mismatches) == 0 {
					continue
				}
				fmt.Fprintf(w, "\n  [%s] (%d total)\n", field, len(mismatches))
				limit := exampleLimit
				if limit == 0 || len(mismatches) < limit {
					limit = len(mismatches)
				}
				for _, m := range mismatches[:limit] {
					fmt.Fprintf(w, "    %s (%s::%s)\n", m.OID, m.Module, m.Name)
					if m.GomibModule != "" && m.NetSnmpModule != "" {
						fmt.Fprintf(w, "      modules: gomib=%s net-snmp=%s\n", m.GomibModule, m.NetSnmpModule)
					}
					fmt.Fprintf(w, "      gomib=%q net-snmp=%q\n", m.Gomib, m.NetSnmp)
				}
				if exampleLimit > 0 && len(mismatches) > limit {
					fmt.Fprintf(w, "    ... and %d more\n", len(mismatches)-limit)
				}
			}
		}
	}

	if len(result.MissingInGomib) > 0 {
		fmt.Fprintf(w, "\nMissing in gomib (first 20):\n")
		limit := 20
		if len(result.MissingInGomib) < limit {
			limit = len(result.MissingInGomib)
		}
		for _, oid := range result.MissingInGomib[:limit] {
			fmt.Fprintf(w, "  %s\n", oid)
		}
		if len(result.MissingInGomib) > limit {
			fmt.Fprintf(w, "  ... and %d more\n", len(result.MissingInGomib)-limit)
		}
	}
}

// MismatchCategory describes a category of mismatch with likely cause.
type MismatchCategory struct {
	Name        string
	Description string
	Benign      bool // True if this is a known representation difference, not a real semantic mismatch
	Mismatches  []Mismatch
}

// printCategorizedMismatches prints mismatches grouped by likely cause.
func printCategorizedMismatches(w io.Writer, field string, mismatches []Mismatch, limit int, investigateOnly bool) {
	categories := categorizeMismatches(field, mismatches)

	// Count what we'll show
	var totalToShow int
	for _, cat := range categories {
		if investigateOnly && cat.Benign {
			continue
		}
		totalToShow += len(cat.Mismatches)
	}

	if totalToShow == 0 {
		return
	}

	if investigateOnly {
		fmt.Fprintf(w, "\n  [%s] (%d investigate, %d total)\n", field, totalToShow, len(mismatches))
	} else {
		fmt.Fprintf(w, "\n  [%s] (%d total)\n", field, len(mismatches))
	}

	for _, cat := range categories {
		if len(cat.Mismatches) == 0 {
			continue
		}
		if investigateOnly && cat.Benign {
			continue
		}

		benignTag := ""
		if cat.Benign {
			benignTag = " [benign]"
		}
		fmt.Fprintf(w, "\n    %s (%d)%s - %s\n", cat.Name, len(cat.Mismatches), benignTag, cat.Description)

		showLimit := limit
		if showLimit == 0 || len(cat.Mismatches) < showLimit {
			showLimit = len(cat.Mismatches)
		}
		for _, m := range cat.Mismatches[:showLimit] {
			fmt.Fprintf(w, "      %s (%s::%s)\n", m.OID, m.Module, m.Name)
			if m.GomibModule != "" && m.NetSnmpModule != "" {
				fmt.Fprintf(w, "        modules: gomib=%s net-snmp=%s\n", m.GomibModule, m.NetSnmpModule)
			}
			fmt.Fprintf(w, "        gomib=%q net-snmp=%q\n", m.Gomib, m.NetSnmp)
		}
		if limit > 0 && len(cat.Mismatches) > showLimit {
			fmt.Fprintf(w, "      ... and %d more\n", len(cat.Mismatches)-showLimit)
		}
	}
}

// categorizeMismatches groups mismatches by likely cause based on field type.
func categorizeMismatches(field string, mismatches []Mismatch) []MismatchCategory {
	switch field {
	case "ranges":
		return categorizeRanges(mismatches)
	case "defval":
		return categorizeDefval(mismatches)
	case "status":
		return categorizeStatus(mismatches)
	case "access":
		return categorizeAccess(mismatches)
	case "type":
		return categorizeType(mismatches)
	case "enums":
		return categorizeEnums(mismatches)
	case "varbinds":
		return categorizeVarbinds(mismatches)
	case "index":
		return categorizeIndex(mismatches)
	default:
		return []MismatchCategory{{Name: "uncategorized", Description: "all mismatches", Benign: false, Mismatches: mismatches}}
	}
}

// countBenignAndInvestigate counts mismatches by benign status across all fields.
func countBenignAndInvestigate(mismatches []Mismatch) (benign, investigate int) {
	byField := make(map[string][]Mismatch)
	for _, m := range mismatches {
		byField[m.Field] = append(byField[m.Field], m)
	}

	for field, fieldMismatches := range byField {
		categories := categorizeMismatches(field, fieldMismatches)
		for _, cat := range categories {
			if cat.Benign {
				benign += len(cat.Mismatches)
			} else {
				investigate += len(cat.Mismatches)
			}
		}
	}
	return
}

func categorizeRanges(mismatches []Mismatch) []MismatchCategory {
	var overlap, signedUnsigned, countDiff, valueDiff, other []Mismatch

	for _, m := range mismatches {
		// Check for overlapping definitions first
		if m.GomibModule != "" && m.NetSnmpModule != "" {
			overlap = append(overlap, m)
			continue
		}

		switch {
		case isSignedUnsignedDiff(m.Gomib, m.NetSnmp):
			// net-snmp shows signed interpretation of unsigned values
			signedUnsigned = append(signedUnsigned, m)
		case countRanges(m.Gomib) != countRanges(m.NetSnmp):
			countDiff = append(countDiff, m)
		case m.Gomib != m.NetSnmp:
			valueDiff = append(valueDiff, m)
		default:
			other = append(other, m)
		}
	}

	return []MismatchCategory{
		{Name: "overlap", Description: "same OID defined in different modules", Benign: true, Mismatches: overlap},
		{Name: "signed/unsigned", Description: "net-snmp shows signed interpretation of unsigned values", Benign: true, Mismatches: signedUnsigned},
		{Name: "range-count", Description: "different number of range constraints", Benign: false, Mismatches: countDiff},
		{Name: "value-diff", Description: "different range values", Benign: false, Mismatches: valueDiff},
		{Name: "other", Description: "uncategorized", Benign: false, Mismatches: other},
	}
}

func categorizeDefval(mismatches []Mismatch) []MismatchCategory {
	var overlap, quoteDiff, hexZeros, hexDiff, oidSymbolic, enumDiff, emptyVsValue, spaceDiff, other []Mismatch

	for _, m := range mismatches {
		// Check for overlapping definitions first
		if m.GomibModule != "" && m.NetSnmpModule != "" {
			overlap = append(overlap, m)
			continue
		}

		// Normalize: strip all quote escaping and outer quotes
		gNorm := normalizeDefval(m.Gomib)
		nNorm := normalizeDefval(m.NetSnmp)

		switch {
		case gNorm == nNorm:
			// Only differs by quoting/escaping
			quoteDiff = append(quoteDiff, m)
		case strings.ReplaceAll(gNorm, " ", "") == strings.ReplaceAll(nNorm, " ", ""):
			// Only differs by whitespace
			spaceDiff = append(spaceDiff, m)
		case gNorm == "" && nNorm != "":
			emptyVsValue = append(emptyVsValue, m)
		case nNorm == "" && gNorm != "":
			emptyVsValue = append(emptyVsValue, m)
		case isHexZeroDiff(m.Gomib, m.NetSnmp):
			// gomib shows 0x0000... net-snmp shows 0
			hexZeros = append(hexZeros, m)
		case strings.HasPrefix(m.Gomib, "0x") || strings.HasPrefix(m.NetSnmp, "0x") ||
			strings.Contains(m.Gomib, "'H") || strings.Contains(m.NetSnmp, "'H"):
			hexDiff = append(hexDiff, m)
		case isOidSymbolicDiff(m.Gomib, m.NetSnmp):
			// gomib shows numeric OID, net-snmp shows symbolic name
			oidSymbolic = append(oidSymbolic, m)
		case strings.Contains(m.Gomib, "(") || strings.Contains(m.NetSnmp, "("):
			// Enum name vs value
			enumDiff = append(enumDiff, m)
		default:
			other = append(other, m)
		}
	}

	return []MismatchCategory{
		{Name: "overlap", Description: "same OID defined in different modules", Benign: true, Mismatches: overlap},
		{Name: "quoting", Description: "only differs in quote/escape style", Benign: true, Mismatches: quoteDiff},
		{Name: "whitespace", Description: "only differs in whitespace", Benign: true, Mismatches: spaceDiff},
		{Name: "hex-zeros", Description: "gomib 0x0000... vs net-snmp 0 (same semantic value)", Benign: true, Mismatches: hexZeros},
		{Name: "hex-format", Description: "hex string format difference", Benign: false, Mismatches: hexDiff},
		{Name: "oid-symbolic", Description: "gomib numeric OID vs net-snmp symbolic name", Benign: false, Mismatches: oidSymbolic},
		{Name: "enum-format", Description: "enum name vs numeric value", Benign: false, Mismatches: enumDiff},
		{Name: "empty-vs-value", Description: "one side has value, other empty", Benign: false, Mismatches: emptyVsValue},
		{Name: "other", Description: "uncategorized value difference", Benign: false, Mismatches: other},
	}
}

// isHexZeroDiff checks if gomib shows 0x0000... and net-snmp shows "0".
func isHexZeroDiff(gomib, netsnmp string) bool {
	// gomib shows hex zeros like 0x00000000...
	if !strings.HasPrefix(gomib, "0x") {
		return false
	}
	// Check if all hex digits are zeros
	hexPart := strings.TrimPrefix(gomib, "0x")
	for _, c := range hexPart {
		if c != '0' {
			return false
		}
	}
	// net-snmp shows just "0"
	return netsnmp == "0"
}

// isOidSymbolicDiff checks if gomib shows numeric OID and net-snmp shows symbolic name.
func isOidSymbolicDiff(gomib, netsnmp string) bool {
	// gomib shows numeric OID like "0.0" or "1.3.6.1..."
	if !strings.Contains(gomib, ".") {
		return false
	}
	// Check if gomib looks like a numeric OID (digits and dots only)
	isNumericOID := true
	for _, c := range gomib {
		if c != '.' && (c < '0' || c > '9') {
			isNumericOID = false
			break
		}
	}
	if !isNumericOID {
		return false
	}
	// net-snmp shows symbolic name (no dots, contains letters)
	hasLetter := false
	for _, c := range netsnmp {
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') {
			hasLetter = true
			break
		}
	}
	return hasLetter && !strings.Contains(netsnmp, ".")
}

// normalizeDefval removes quoting/escaping to get the semantic value.
func normalizeDefval(s string) string {
	// Remove escaped quotes: \" -> "
	s = strings.ReplaceAll(s, "\\\"", "\"")
	// Remove backslash escapes
	s = strings.ReplaceAll(s, "\\\\", "\\")
	// Trim outer quotes
	s = strings.Trim(strings.TrimSpace(s), "\"'")
	return s
}

func categorizeStatus(mismatches []Mismatch) []MismatchCategory {
	var overlap, deprecatedObsolete, currentMandatory, other []Mismatch

	for _, m := range mismatches {
		// Check for overlapping definitions first
		if m.GomibModule != "" && m.NetSnmpModule != "" {
			overlap = append(overlap, m)
			continue
		}

		g, n := strings.ToLower(m.Gomib), strings.ToLower(m.NetSnmp)
		switch {
		case (g == "deprecated" && n == "obsolete") || (g == "obsolete" && n == "deprecated"):
			deprecatedObsolete = append(deprecatedObsolete, m)
		case (g == "current" && n == "mandatory") || (g == "mandatory" && n == "current"):
			currentMandatory = append(currentMandatory, m)
		default:
			other = append(other, m)
		}
	}

	return []MismatchCategory{
		{Name: "overlap", Description: "same OID defined in different modules", Benign: true, Mismatches: overlap},
		{Name: "deprecated/obsolete", Description: "deprecated vs obsolete (often equivalent)", Benign: false, Mismatches: deprecatedObsolete},
		{Name: "current/mandatory", Description: "SMIv1 mandatory vs SMIv2 current", Benign: true, Mismatches: currentMandatory},
		{Name: "other", Description: "other status differences", Benign: false, Mismatches: other},
	}
}

func categorizeAccess(mismatches []Mismatch) []MismatchCategory {
	var overlap, rwCreate, naReadOnly, other []Mismatch

	for _, m := range mismatches {
		// Check for overlapping definitions first
		if m.GomibModule != "" && m.NetSnmpModule != "" {
			overlap = append(overlap, m)
			continue
		}

		g, n := strings.ToLower(m.Gomib), strings.ToLower(m.NetSnmp)
		switch {
		case (g == "read-create" && n == "read-write") || (g == "read-write" && n == "read-create"):
			rwCreate = append(rwCreate, m)
		case (g == "not-accessible" && n == "read-only") || (g == "read-only" && n == "not-accessible"):
			naReadOnly = append(naReadOnly, m)
		default:
			other = append(other, m)
		}
	}

	return []MismatchCategory{
		{Name: "overlap", Description: "same OID defined in different modules", Benign: true, Mismatches: overlap},
		{Name: "read-write/read-create", Description: "SMIv1 read-write vs SMIv2 read-create", Benign: true, Mismatches: rwCreate},
		{Name: "access-level", Description: "not-accessible vs read-only", Benign: false, Mismatches: naReadOnly},
		{Name: "other", Description: "other access differences", Benign: false, Mismatches: other},
	}
}

func categorizeType(mismatches []Mismatch) []MismatchCategory {
	var overlap, networkAddr, intVariants, other []Mismatch

	for _, m := range mismatches {
		// Check for overlapping definitions first
		if m.GomibModule != "" && m.NetSnmpModule != "" {
			overlap = append(overlap, m)
			continue
		}

		switch {
		case strings.Contains(m.Gomib, "Address") || strings.Contains(m.NetSnmp, "Address"):
			networkAddr = append(networkAddr, m)
		case strings.Contains(m.Gomib, "Integer") || strings.Contains(m.Gomib, "INTEGER") ||
			strings.Contains(m.NetSnmp, "Integer") || strings.Contains(m.NetSnmp, "INTEGER"):
			intVariants = append(intVariants, m)
		default:
			other = append(other, m)
		}
	}

	return []MismatchCategory{
		{Name: "overlap", Description: "same OID defined in different modules", Benign: true, Mismatches: overlap},
		{Name: "address-types", Description: "NetworkAddress vs IpAddress (SMIv1 legacy)", Benign: true, Mismatches: networkAddr},
		{Name: "integer-variants", Description: "INTEGER vs Integer32 variants", Benign: true, Mismatches: intVariants},
		{Name: "other", Description: "other type differences", Benign: false, Mismatches: other},
	}
}

func categorizeEnums(mismatches []Mismatch) []MismatchCategory {
	var overlap, gomibMoreValues, netsnmpMoreValues, valueDiff, other []Mismatch

	for _, m := range mismatches {
		// Check for overlapping definitions (different modules for same OID)
		if m.GomibModule != "" && m.NetSnmpModule != "" {
			overlap = append(overlap, m)
			continue
		}

		gCount := strings.Count(m.Gomib, "(")
		nCount := strings.Count(m.NetSnmp, "(")
		switch {
		case gCount > nCount:
			// gomib has more enum values
			gomibMoreValues = append(gomibMoreValues, m)
		case nCount > gCount:
			// net-snmp has more enum values
			netsnmpMoreValues = append(netsnmpMoreValues, m)
		case gCount != nCount:
			other = append(other, m)
		default:
			valueDiff = append(valueDiff, m)
		}
	}

	return []MismatchCategory{
		{Name: "overlap", Description: "same OID defined in different modules", Benign: true, Mismatches: overlap},
		{Name: "gomib-more-values", Description: "gomib has more enum values (check MIB source)", Benign: false, Mismatches: gomibMoreValues},
		{Name: "netsnmp-more-values", Description: "net-snmp has more enum values (check for import shadowing)", Benign: false, Mismatches: netsnmpMoreValues},
		{Name: "enum-values", Description: "different enum names or numbers", Benign: false, Mismatches: valueDiff},
		{Name: "other", Description: "uncategorized", Benign: false, Mismatches: other},
	}
}

func categorizeVarbinds(mismatches []Mismatch) []MismatchCategory {
	var netsnmpMore, gomibMore, different, other []Mismatch

	for _, m := range mismatches {
		gCount := strings.Count(m.Gomib, ",") + 1
		nCount := strings.Count(m.NetSnmp, ",") + 1
		if m.Gomib == "" || m.Gomib == "{}" {
			gCount = 0
		}
		if m.NetSnmp == "" || m.NetSnmp == "{}" {
			nCount = 0
		}

		switch {
		case nCount > gCount:
			// net-snmp has more OBJECTS (possibly unresolved refs gomib excludes)
			netsnmpMore = append(netsnmpMore, m)
		case gCount > nCount:
			// gomib has more OBJECTS
			gomibMore = append(gomibMore, m)
		case gCount == nCount && gCount > 0:
			// Same count but different names
			different = append(different, m)
		default:
			other = append(other, m)
		}
	}

	return []MismatchCategory{
		{Name: "netsnmp-more-objects", Description: "net-snmp has more OBJECTS (check for unresolved refs)", Benign: false, Mismatches: netsnmpMore},
		{Name: "gomib-more-objects", Description: "gomib has more OBJECTS", Benign: false, Mismatches: gomibMore},
		{Name: "different-objects", Description: "same count but different object names", Benign: false, Mismatches: different},
		{Name: "other", Description: "uncategorized", Benign: false, Mismatches: other},
	}
}

func categorizeIndex(mismatches []Mismatch) []MismatchCategory {
	var netsnmpMore, gomibMore, different, other []Mismatch

	for _, m := range mismatches {
		gCount := strings.Count(m.Gomib, ",") + 1
		nCount := strings.Count(m.NetSnmp, ",") + 1
		if m.Gomib == "" || m.Gomib == "{}" {
			gCount = 0
		}
		if m.NetSnmp == "" || m.NetSnmp == "{}" {
			nCount = 0
		}

		switch {
		case nCount > gCount:
			// net-snmp has more INDEX items (possibly unresolved refs gomib excludes)
			netsnmpMore = append(netsnmpMore, m)
		case gCount > nCount:
			// gomib has more INDEX items
			gomibMore = append(gomibMore, m)
		case gCount == nCount && gCount > 0:
			// Same count but different names
			different = append(different, m)
		default:
			other = append(other, m)
		}
	}

	return []MismatchCategory{
		{Name: "netsnmp-more-indexes", Description: "net-snmp has more INDEX items (check for unresolved refs)", Benign: false, Mismatches: netsnmpMore},
		{Name: "gomib-more-indexes", Description: "gomib has more INDEX items", Benign: false, Mismatches: gomibMore},
		{Name: "different-indexes", Description: "same count but different index names", Benign: false, Mismatches: different},
		{Name: "other", Description: "uncategorized", Benign: false, Mismatches: other},
	}
}

func countRanges(s string) int {
	return strings.Count(s, "..")
}

// isSignedUnsignedDiff checks if the range difference is due to signed vs unsigned interpretation.
// Examples: "4294967295" vs "-1", "4294967294" vs "-2", "2147483648" vs "-2147483648"
func isSignedUnsignedDiff(gomib, netsnmp string) bool {
	// Quick check: net-snmp must contain a negative number
	if !strings.Contains(netsnmp, "-") {
		return false
	}

	// Check for patterns like "..-1)", "..-2)", "| -1)", etc.
	// These indicate net-snmp is showing signed interpretation
	if strings.Contains(netsnmp, "..-") || strings.Contains(netsnmp, "| -") ||
		strings.Contains(netsnmp, "(-") {
		return true
	}

	return false
}

func printFieldAccuracy(w io.Writer, name string, c CountPair) {
	total := c.Match + c.Mismatch
	if total == 0 {
		return
	}
	pct := 100.0 * float64(c.Match) / float64(total)
	fmt.Fprintf(w, "  %-10s %5d match, %5d mismatch (%.1f%% accurate)\n",
		name+":", c.Match, c.Mismatch, pct)
}
