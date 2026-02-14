//go:build cgo

//nolint:errcheck // CLI output, errors not critical
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"slices"
	"strings"

	"github.com/golangsnmp/gomib"
)

// SemanticComparison holds the results of comparing gomib and libsmi output.
type SemanticComparison struct {
	TotalLibsmi     int            `json:"total_libsmi"`
	TotalGomib      int            `json:"total_gomib"`
	MatchedNodes    int            `json:"matched_nodes"`
	MissingInGomib  []string       `json:"missing_in_gomib,omitempty"`
	MissingInLibsmi []string       `json:"missing_in_libsmi,omitempty"`
	Mismatches      []NodeCompare  `json:"mismatches,omitempty"`
	Summary         CompareSummary `json:"summary"`
}

// NodeCompare describes a field-level difference for one OID node.
type NodeCompare struct {
	OID    string `json:"oid"`
	Name   string `json:"name"`
	Module string `json:"module"`
	Field  string `json:"field"`
	Gomib  string `json:"gomib"`
	Libsmi string `json:"libsmi"`
}

// CompareSummary tracks per-field match and mismatch counts.
type CompareSummary struct {
	Kind     MatchCount `json:"kind"`
	Status   MatchCount `json:"status"`
	Access   MatchCount `json:"access"`
	BaseType MatchCount `json:"basetype"`
}

// MatchCount holds a match and mismatch tally for one field.
type MatchCount struct {
	Match    int `json:"match"`
	Mismatch int `json:"mismatch"`
}

func cmdCompare(args []string) int {
	fs := flag.NewFlagSet("compare", flag.ContinueOnError)

	fs.Usage = func() {
		fmt.Fprintf(fs.Output(), `Usage: gomib-libsmi compare [options] [MODULE...]

Semantic comparison between gomib and libsmi:
- OID resolution
- Node kinds (table, row, column, scalar)
- Status values
- Access levels
- Base types

Options:
`)
		fs.PrintDefaults()
	}

	if err := fs.Parse(args); err != nil {
		return 1
	}

	modules := fs.Args()
	mibPaths := getMIBPaths()
	if len(mibPaths) == 0 {
		printError("at least one -p PATH is required")
		return 1
	}

	out, cleanup, err := getOutput()
	if err != nil {
		printError("cannot open output: %v", err)
		return 1
	}
	defer cleanup()

	result := compareSemantics(modules, mibPaths)

	if jsonOutput {
		enc := json.NewEncoder(out)
		enc.SetIndent("", "  ")
		if err := enc.Encode(result); err != nil {
			printError("json encode failed: %v", err)
			return 1
		}
	} else {
		printSemanticComparison(out, result)
	}

	return 0
}

func compareSemantics(modules []string, mibPaths []string) *SemanticComparison {
	result := &SemanticComparison{}

	libsmiPath := BuildMIBPath(expandDirs(mibPaths))
	InitLibsmi(libsmiPath, 6) // Info level - collect everything
	defer CleanupLibsmi()

	for _, mod := range modules {
		LoadModule(mod)
		CollectNodes(mod)
	}

	libsmiNodes := make(map[string]LibsmiNode)
	for _, n := range GetNodes() {
		if n.OID != "" {
			libsmiNodes[n.OID] = n
		}
	}
	ClearNodes()

	result.TotalLibsmi = len(libsmiNodes)

	gomibNodes := make(map[string]gomibNode)

	if source := buildSource(mibPaths); source != nil {
		ctx := context.Background()
		var m *gomib.Mib
		var err error

		loadOpts := []gomib.LoadOption{gomib.WithSource(source)}
		if len(modules) > 0 {
			loadOpts = append(loadOpts, gomib.WithModules(modules...))
		}
		m, err = gomib.Load(ctx, loadOpts...)

		if err == nil && m != nil {
			for node := range m.Nodes() {
				oid := node.OID().String()
				if oid == "" {
					continue
				}

				gn := gomibNode{
					OID:  oid,
					Name: node.Name(),
					Kind: kindToString(node.Kind()),
				}

				if mod := node.Module(); mod != nil {
					gn.Module = mod.Name()
				}

				if obj := node.Object(); obj != nil {
					gn.Status = statusToString(obj.Status())
					gn.Access = accessToString(obj.Access())
					if t := obj.Type(); t != nil {
						gn.BaseType = baseTypeToString(t.EffectiveBase())
					}
				}

				gomibNodes[oid] = gn
			}
		}
	}

	result.TotalGomib = len(gomibNodes)

	allOIDs := make(map[string]bool)
	for oid := range libsmiNodes {
		allOIDs[oid] = true
	}
	for oid := range gomibNodes {
		allOIDs[oid] = true
	}

	for oid := range allOIDs {
		ls, hasLibsmi := libsmiNodes[oid]
		gm, hasGomib := gomibNodes[oid]

		if !hasLibsmi {
			result.MissingInLibsmi = append(result.MissingInLibsmi, oid)
			continue
		}
		if !hasGomib {
			result.MissingInGomib = append(result.MissingInGomib, oid)
			continue
		}

		result.MatchedNodes++

		if ls.NodeKind != "" && gm.Kind != "" {
			if kindsEquivalent(gm.Kind, ls.NodeKind) {
				result.Summary.Kind.Match++
			} else {
				result.Summary.Kind.Mismatch++
				result.Mismatches = append(result.Mismatches, NodeCompare{
					OID:    oid,
					Name:   ls.Name,
					Module: ls.Module,
					Field:  "kind",
					Gomib:  gm.Kind,
					Libsmi: ls.NodeKind,
				})
			}
		}

		if ls.Status != "" && gm.Status != "" {
			if gm.Status == ls.Status {
				result.Summary.Status.Match++
			} else {
				result.Summary.Status.Mismatch++
				result.Mismatches = append(result.Mismatches, NodeCompare{
					OID:    oid,
					Name:   ls.Name,
					Module: ls.Module,
					Field:  "status",
					Gomib:  gm.Status,
					Libsmi: ls.Status,
				})
			}
		}

		if ls.Access != "" && gm.Access != "" {
			if accessEquivalent(gm.Access, ls.Access) {
				result.Summary.Access.Match++
			} else {
				result.Summary.Access.Mismatch++
				result.Mismatches = append(result.Mismatches, NodeCompare{
					OID:    oid,
					Name:   ls.Name,
					Module: ls.Module,
					Field:  "access",
					Gomib:  gm.Access,
					Libsmi: ls.Access,
				})
			}
		}

		if ls.BaseType != "" && gm.BaseType != "" {
			if baseTypesEquivalent(gm.BaseType, ls.BaseType) {
				result.Summary.BaseType.Match++
			} else {
				result.Summary.BaseType.Mismatch++
				result.Mismatches = append(result.Mismatches, NodeCompare{
					OID:    oid,
					Name:   ls.Name,
					Module: ls.Module,
					Field:  "basetype",
					Gomib:  gm.BaseType,
					Libsmi: ls.BaseType,
				})
			}
		}
	}

	slices.Sort(result.MissingInGomib)
	slices.Sort(result.MissingInLibsmi)

	return result
}

type gomibNode struct {
	OID      string
	Name     string
	Module   string
	Kind     string
	Status   string
	Access   string
	BaseType string
}

func kindToString(k gomib.Kind) string {
	if k == gomib.KindUnknown {
		return ""
	}
	return k.String()
}

func statusToString(s gomib.Status) string {
	return s.String()
}

func accessToString(a gomib.Access) string {
	return a.String()
}

func baseTypeToString(b gomib.BaseType) string {
	if b == gomib.BaseUnknown {
		return ""
	}
	return b.String()
}

func kindsEquivalent(a, b string) bool {
	if a == b {
		return true
	}
	// libsmi uses "node" for MODULE-IDENTITY, OBJECT-IDENTITY, and value
	// assignments. Only accept equivalence when gomib also reports "node",
	// which covers the same macro types. Don't give a free pass when gomib
	// reports a specific kind like "scalar" or "table".
	if (b == "node" || b == "unknown") && a == "node" {
		return true
	}
	return false
}

func accessEquivalent(a, b string) bool {
	if a == b {
		return true
	}
	// Handle SMIv1/v2 naming differences
	norm := func(s string) string {
		switch s {
		case "read-only", "readonly":
			return "read-only"
		case "read-write", "readwrite":
			return "read-write"
		case "not-accessible", "notaccessible":
			return "not-accessible"
		default:
			return s
		}
	}
	return norm(a) == norm(b)
}

func baseTypesEquivalent(a, b string) bool {
	if a == b {
		return true
	}
	norm := func(s string) string {
		switch s {
		case "INTEGER", "Integer32":
			return "Integer32"
		case "OCTET STRING", "OctetString":
			return "OCTET STRING"
		case "OBJECT IDENTIFIER", "ObjectIdentifier":
			return "OBJECT IDENTIFIER"
		default:
			return s
		}
	}
	return norm(a) == norm(b)
}

func printSemanticComparison(w io.Writer, result *SemanticComparison) {
	fmt.Fprintln(w, strings.Repeat("=", 70))
	fmt.Fprintln(w, "GOMIB vs LIBSMI SEMANTIC COMPARISON")
	fmt.Fprintln(w, strings.Repeat("=", 70))

	fmt.Fprintf(w, "\nNode counts:\n")
	fmt.Fprintf(w, "  libsmi nodes:        %6d\n", result.TotalLibsmi)
	fmt.Fprintf(w, "  gomib nodes:         %6d\n", result.TotalGomib)
	fmt.Fprintf(w, "  common nodes:        %6d\n", result.MatchedNodes)
	fmt.Fprintf(w, "  missing in gomib:    %6d\n", len(result.MissingInGomib))
	fmt.Fprintf(w, "  missing in libsmi:   %6d\n", len(result.MissingInLibsmi))

	fmt.Fprintf(w, "\nField accuracy (for common nodes):\n")
	printMatchCount(w, "kind", result.Summary.Kind)
	printMatchCount(w, "status", result.Summary.Status)
	printMatchCount(w, "access", result.Summary.Access)
	printMatchCount(w, "basetype", result.Summary.BaseType)

	if len(result.Mismatches) > 0 {
		byField := make(map[string][]NodeCompare)
		for _, m := range result.Mismatches {
			byField[m.Field] = append(byField[m.Field], m)
		}

		fmt.Fprintf(w, "\nMismatches by field (up to 5 each):\n")
		for _, field := range []string{"kind", "status", "access", "basetype"} {
			mismatches := byField[field]
			if len(mismatches) == 0 {
				continue
			}
			fmt.Fprintf(w, "\n  [%s] (%d total)\n", field, len(mismatches))
			limit := 5
			if len(mismatches) < limit {
				limit = len(mismatches)
			}
			for _, m := range mismatches[:limit] {
				fmt.Fprintf(w, "    %s (%s::%s)\n", m.OID, m.Module, m.Name)
				fmt.Fprintf(w, "      gomib=%q libsmi=%q\n", m.Gomib, m.Libsmi)
			}
			if len(mismatches) > limit {
				fmt.Fprintf(w, "    ... and %d more\n", len(mismatches)-limit)
			}
		}
	}

	if len(result.MissingInGomib) > 0 {
		fmt.Fprintf(w, "\nMissing in gomib (first 10):\n")
		limit := 10
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

func printMatchCount(w io.Writer, name string, c MatchCount) {
	total := c.Match + c.Mismatch
	if total == 0 {
		return
	}
	pct := 100.0 * float64(c.Match) / float64(total)
	fmt.Fprintf(w, "  %-10s %5d match, %5d mismatch (%.1f%% accurate)\n",
		name+":", c.Match, c.Mismatch, pct)
}
