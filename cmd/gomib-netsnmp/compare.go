//go:build cgo

//nolint:errcheck // CLI output, errors not critical
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
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
	OID     string `json:"oid"`
	Name    string `json:"name"`
	Module  string `json:"module"`
	Field   string `json:"field"`
	Gomib   string `json:"gomib"`
	NetSnmp string `json:"netsnmp"`
}

// FieldCounts tracks match/mismatch counts per field.
type FieldCounts struct {
	Type   CountPair `json:"type"`
	Access CountPair `json:"access"`
	Status CountPair `json:"status"`
	Enums  CountPair `json:"enums"`
	Index  CountPair `json:"index"`
}

// CountPair holds match and mismatch counts.
type CountPair struct {
	Match    int `json:"match"`
	Mismatch int `json:"mismatch"`
}

func cmdCompare(args []string) int {
	fs := flag.NewFlagSet("compare", flag.ExitOnError)
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

	fmt.Fprintln(out, "Loading MIBs with net-snmp...")
	netsnmpNodes, err := loadNetSnmpNodes(mibPaths, modules)
	if err != nil {
		printError("net-snmp load failed: %v", err)
		return 1
	}

	fmt.Fprintln(out, "Loading MIBs with gomib...")
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

	fmt.Fprintf(out, "net-snmp: %d nodes, gomib: %d nodes\n\n", len(netsnmpNodes), len(gomibNodes))

	result := compareNodes(netsnmpNodes, gomibNodes)

	if jsonOutput {
		enc := json.NewEncoder(out)
		enc.SetIndent("", "  ")
		if err := enc.Encode(result); err != nil {
			printError("json encode failed: %v", err)
			return 1
		}
	} else {
		printComparisonResult(out, result)
	}

	return 0
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
				result.Mismatches = append(result.Mismatches, Mismatch{
					OID:     oid,
					Name:    nsNode.Name,
					Module:  nsNode.Module,
					Field:   "type",
					Gomib:   gNode.Type,
					NetSnmp: nsNode.Type,
				})
			}
		}

		// Compare access
		if nsNode.Access != "" {
			if gNode.Access == nsNode.Access {
				result.Summary.Access.Match++
			} else if gNode.Access != "" {
				result.Summary.Access.Mismatch++
				result.Mismatches = append(result.Mismatches, Mismatch{
					OID:     oid,
					Name:    nsNode.Name,
					Module:  nsNode.Module,
					Field:   "access",
					Gomib:   gNode.Access,
					NetSnmp: nsNode.Access,
				})
			}
		}

		// Compare status
		if nsNode.Status != "" {
			if gNode.Status == nsNode.Status {
				result.Summary.Status.Match++
			} else if gNode.Status != "" {
				result.Summary.Status.Mismatch++
				result.Mismatches = append(result.Mismatches, Mismatch{
					OID:     oid,
					Name:    nsNode.Name,
					Module:  nsNode.Module,
					Field:   "status",
					Gomib:   gNode.Status,
					NetSnmp: nsNode.Status,
				})
			}
		}

		// Compare enums
		if len(nsNode.EnumValues) > 0 {
			if enumsEqual(nsNode.EnumValues, gNode.EnumValues) {
				result.Summary.Enums.Match++
			} else {
				result.Summary.Enums.Mismatch++
				result.Mismatches = append(result.Mismatches, Mismatch{
					OID:     oid,
					Name:    nsNode.Name,
					Module:  nsNode.Module,
					Field:   "enums",
					Gomib:   formatEnums(gNode.EnumValues),
					NetSnmp: formatEnums(nsNode.EnumValues),
				})
			}
		}

		// Compare indexes
		if len(nsNode.Indexes) > 0 {
			if indexesEqual(nsNode.Indexes, gNode.Indexes) {
				result.Summary.Index.Match++
			} else {
				result.Summary.Index.Mismatch++
				result.Mismatches = append(result.Mismatches, Mismatch{
					OID:     oid,
					Name:    nsNode.Name,
					Module:  nsNode.Module,
					Field:   "index",
					Gomib:   indexString(gNode.Indexes),
					NetSnmp: indexString(nsNode.Indexes),
				})
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

func printComparisonResult(w io.Writer, result *ComparisonResult) {
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

	if len(result.Mismatches) > 0 {
		fmt.Fprintf(w, "\nMismatches (first 50):\n")
		limit := 50
		if len(result.Mismatches) < limit {
			limit = len(result.Mismatches)
		}
		for _, m := range result.Mismatches[:limit] {
			fmt.Fprintf(w, "  %s (%s::%s)\n", m.OID, m.Module, m.Name)
			fmt.Fprintf(w, "    %s: gomib=%q net-snmp=%q\n", m.Field, m.Gomib, m.NetSnmp)
		}
		if len(result.Mismatches) > limit {
			fmt.Fprintf(w, "  ... and %d more\n", len(result.Mismatches)-limit)
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

func printFieldAccuracy(w io.Writer, name string, c CountPair) {
	total := c.Match + c.Mismatch
	if total == 0 {
		return
	}
	pct := 100.0 * float64(c.Match) / float64(total)
	fmt.Fprintf(w, "  %-10s %5d match, %5d mismatch (%.1f%% accurate)\n",
		name+":", c.Match, c.Mismatch, pct)
}
