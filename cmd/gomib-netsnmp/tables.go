//go:build cgo

//nolint:errcheck // CLI output, errors not critical
package main

import (
	"cmp"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"slices"
	"strings"
)

// TableComparisonResult holds results of table-focused comparison.
type TableComparisonResult struct {
	TotalTables     int               `json:"total_tables"`
	MatchedTables   int               `json:"matched_tables"`
	IndexMatches    int               `json:"index_matches"`
	IndexMismatches []TableMismatch   `json:"index_mismatches,omitempty"`
	AugmentMatches  int               `json:"augment_matches"`
	AugmentMisses   []TableMismatch   `json:"augment_mismatches,omitempty"`
	Tables          []TableComparison `json:"tables,omitempty"`
}

// TableMismatch describes a table-specific difference.
type TableMismatch struct {
	RowName string `json:"row_name"`
	Module  string `json:"module"`
	OID     string `json:"oid"`
	Field   string `json:"field"`
	Gomib   string `json:"gomib"`
	NetSnmp string `json:"netsnmp"`
}

// TableComparison holds per-table comparison data.
type TableComparison struct {
	TableName     string      `json:"table_name"`
	RowName       string      `json:"row_name"`
	Module        string      `json:"module"`
	OID           string      `json:"oid"`
	NetSnmpIndex  []IndexInfo `json:"netsnmp_index"`
	GomibIndex    []IndexInfo `json:"gomib_index"`
	IndexMatch    bool        `json:"index_match"`
	NetSnmpAug    string      `json:"netsnmp_augments,omitempty"`
	GomibAug      string      `json:"gomib_augments,omitempty"`
	AugmentsMatch bool        `json:"augments_match"`
}

func cmdTables(args []string) int {
	fs := flag.NewFlagSet("tables", flag.ExitOnError)
	detailed := fs.Bool("detailed", false, "Show detailed per-table comparison")

	fs.Usage = func() {
		fmt.Fprintf(fs.Output(), `Usage: gomib-netsnmp tables [options] [MODULE...]

Table-specific comparison:
- INDEX clause (names, order, IMPLIED)
- AUGMENTS targets
- Column membership

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

	if len(modules) > 0 {
		netsnmpNodes = filterByModules(netsnmpNodes, modules)
		gomibNodes = filterByModules(gomibNodes, modules)
	}

	result := compareTables(netsnmpNodes, gomibNodes, *detailed)

	if jsonOutput {
		enc := json.NewEncoder(out)
		enc.SetIndent("", "  ")
		if err := enc.Encode(result); err != nil {
			printError("json encode failed: %v", err)
			return 1
		}
	} else {
		printTableComparisonResult(out, result, *detailed)
	}

	return 0
}

// compareTables compares table structures between net-snmp and gomib.
func compareTables(netsnmp, gomib map[string]*NormalizedNode, detailed bool) *TableComparisonResult {
	result := &TableComparisonResult{}

	// Find nodes with indexes (row entries)
	type rowEntry struct {
		oid   string
		node  *NormalizedNode
		gomib *NormalizedNode
	}

	var rows []rowEntry
	for oid, node := range netsnmp {
		if len(node.Indexes) > 0 || node.Augments != "" {
			rows = append(rows, rowEntry{
				oid:   oid,
				node:  node,
				gomib: gomib[oid],
			})
		}
	}

	// Sort by OID for deterministic output
	slices.SortFunc(rows, func(a, b rowEntry) int {
		return cmp.Compare(a.oid, b.oid)
	})

	result.TotalTables = len(rows)

	for _, row := range rows {
		ns := row.node
		g := row.gomib

		tc := TableComparison{
			RowName:      ns.Name,
			Module:       ns.Module,
			OID:          row.oid,
			NetSnmpIndex: ns.Indexes,
			NetSnmpAug:   ns.Augments,
		}

		if g != nil {
			result.MatchedTables++
			tc.GomibIndex = g.Indexes
			tc.GomibAug = g.Augments

			// Compare indexes
			if indexesEqual(ns.Indexes, g.Indexes) {
				result.IndexMatches++
				tc.IndexMatch = true
			} else {
				result.IndexMismatches = append(result.IndexMismatches, TableMismatch{
					RowName: ns.Name,
					Module:  ns.Module,
					OID:     row.oid,
					Field:   "INDEX",
					Gomib:   indexString(g.Indexes),
					NetSnmp: indexString(ns.Indexes),
				})
			}

			// Compare augments
			if ns.Augments != "" || g.Augments != "" {
				if ns.Augments == g.Augments {
					result.AugmentMatches++
					tc.AugmentsMatch = true
				} else {
					result.AugmentMisses = append(result.AugmentMisses, TableMismatch{
						RowName: ns.Name,
						Module:  ns.Module,
						OID:     row.oid,
						Field:   "AUGMENTS",
						Gomib:   g.Augments,
						NetSnmp: ns.Augments,
					})
				}
			}
		} else {
			result.IndexMismatches = append(result.IndexMismatches, TableMismatch{
				RowName: ns.Name,
				Module:  ns.Module,
				OID:     row.oid,
				Field:   "missing",
				Gomib:   "(not found)",
				NetSnmp: indexString(ns.Indexes),
			})
		}

		if detailed {
			result.Tables = append(result.Tables, tc)
		}
	}

	return result
}

// getTableName attempts to derive table name from row name.
func getTableName(rowName string) string {
	if strings.HasSuffix(rowName, "Entry") {
		return strings.TrimSuffix(rowName, "Entry") + "Table"
	}
	return rowName + "Table"
}

func printTableComparisonResult(w io.Writer, result *TableComparisonResult, detailed bool) {
	fmt.Fprintln(w, strings.Repeat("=", 70))
	fmt.Fprintln(w, "TABLE COMPARISON RESULTS")
	fmt.Fprintln(w, strings.Repeat("=", 70))

	fmt.Fprintf(w, "\nTable/Row counts:\n")
	fmt.Fprintf(w, "  Total rows with INDEX/AUGMENTS: %d\n", result.TotalTables)
	fmt.Fprintf(w, "  Matched in both libraries:      %d\n", result.MatchedTables)

	if result.MatchedTables > 0 {
		idxTotal := result.IndexMatches + len(result.IndexMismatches)
		fmt.Fprintf(w, "\nINDEX accuracy:\n")
		fmt.Fprintf(w, "  Matches:    %d / %d\n", result.IndexMatches, idxTotal)
		fmt.Fprintf(w, "  Mismatches: %d\n", len(result.IndexMismatches))

		augTotal := result.AugmentMatches + len(result.AugmentMisses)
		if augTotal > 0 {
			fmt.Fprintf(w, "\nAUGMENTS accuracy:\n")
			fmt.Fprintf(w, "  Matches:    %d / %d\n", result.AugmentMatches, augTotal)
			fmt.Fprintf(w, "  Mismatches: %d\n", len(result.AugmentMisses))
		}
	}

	if len(result.IndexMismatches) > 0 {
		fmt.Fprintf(w, "\nINDEX mismatches:\n")
		for _, m := range result.IndexMismatches {
			fmt.Fprintf(w, "  %s::%s (%s)\n", m.Module, m.RowName, m.OID)
			fmt.Fprintf(w, "    net-snmp: %s\n", m.NetSnmp)
			fmt.Fprintf(w, "    gomib:    %s\n", m.Gomib)
		}
	}

	if len(result.AugmentMisses) > 0 {
		fmt.Fprintf(w, "\nAUGMENTS mismatches:\n")
		for _, m := range result.AugmentMisses {
			fmt.Fprintf(w, "  %s::%s (%s)\n", m.Module, m.RowName, m.OID)
			fmt.Fprintf(w, "    net-snmp: AUGMENTS { %s }\n", m.NetSnmp)
			fmt.Fprintf(w, "    gomib:    AUGMENTS { %s }\n", m.Gomib)
		}
	}

	if detailed && len(result.Tables) > 0 {
		fmt.Fprintf(w, "\nDetailed table comparison:\n")
		for _, tc := range result.Tables {
			status := "OK"
			if !tc.IndexMatch {
				status = "INDEX MISMATCH"
			}
			if tc.NetSnmpAug != "" && !tc.AugmentsMatch {
				if status == "OK" {
					status = "AUGMENTS MISMATCH"
				} else {
					status += ", AUGMENTS MISMATCH"
				}
			}

			fmt.Fprintf(w, "\n  %s::%s [%s]\n", tc.Module, tc.RowName, status)
			fmt.Fprintf(w, "    OID: %s\n", tc.OID)
			fmt.Fprintf(w, "    net-snmp INDEX: %s\n", indexString(tc.NetSnmpIndex))
			fmt.Fprintf(w, "    gomib INDEX:    %s\n", indexString(tc.GomibIndex))
			if tc.NetSnmpAug != "" || tc.GomibAug != "" {
				fmt.Fprintf(w, "    net-snmp AUGMENTS: %s\n", tc.NetSnmpAug)
				fmt.Fprintf(w, "    gomib AUGMENTS:    %s\n", tc.GomibAug)
			}
		}
	}
}
