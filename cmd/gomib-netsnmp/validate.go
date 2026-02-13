//go:build cgo

//nolint:errcheck // CLI output, errors not critical
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"io/fs"
	"path/filepath"
	"strconv"
	"strings"
)

// ValidationResult tallies test cases checked against net-snmp ground truth.
type ValidationResult struct {
	FilesChecked   int               `json:"files_checked"`
	TestsValidated int               `json:"tests_validated"`
	TestsPassed    int               `json:"tests_passed"`
	TestsFailed    int               `json:"tests_failed"`
	Failures       []ValidationIssue `json:"failures,omitempty"`
	Warnings       []ValidationIssue `json:"warnings,omitempty"`
}

// ValidationIssue describes a test case that disagrees with net-snmp.
type ValidationIssue struct {
	File     string `json:"file"`
	TestName string `json:"test_name"`
	Field    string `json:"field"`
	Expected string `json:"expected"`
	Actual   string `json:"actual"`
	Message  string `json:"message"`
}

func cmdValidate(args []string) int {
	fs := flag.NewFlagSet("validate", flag.ContinueOnError)
	testDir := fs.String("tests", "", "Directory containing test files (default: ./integration)")

	fs.Usage = func() {
		fmt.Fprintf(fs.Output(), `Usage: gomib-netsnmp validate [options]

Reads existing test cases and validates against net-snmp:
- Reports mismatches
- Identifies obsolete tests
- Suggests corrections

Options:
`)
		fs.PrintDefaults()
	}

	if err := fs.Parse(args); err != nil {
		return 1
	}

	mibPaths := getMIBPaths()

	out, cleanup, err := getOutput()
	if err != nil {
		printError("cannot open output: %v", err)
		return 1
	}
	defer cleanup()

	fmt.Fprintln(out, "Loading MIBs with net-snmp...")
	netsnmpNodes, err := loadNetSnmpNodes(mibPaths, nil)
	if err != nil {
		printError("net-snmp load failed: %v", err)
		return 1
	}
	fmt.Fprintf(out, "Loaded %d nodes from net-snmp\n", len(netsnmpNodes))

	dir := *testDir
	if dir == "" {
		dir = "./integration"
	}

	result := validateTestFiles(dir, netsnmpNodes)

	if jsonOutput {
		enc := json.NewEncoder(out)
		enc.SetIndent("", "  ")
		if err := enc.Encode(result); err != nil {
			printError("json encode failed: %v", err)
			return 1
		}
	} else {
		printValidationResult(out, result)
	}

	if result.TestsFailed > 0 {
		return 1
	}
	return 0
}

func validateTestFiles(dir string, netsnmp map[string]*NormalizedNode) *ValidationResult {
	result := &ValidationResult{}

	byName := make(map[string]*NormalizedNode)
	for _, node := range netsnmp {
		if node.Name != "" {
			key := node.Module + "::" + node.Name
			byName[key] = node
			byName[node.Name] = node
		}
	}

	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if !d.IsDir() && strings.HasSuffix(path, "_test.go") {
			result.FilesChecked++
			validateFile(path, byName, result)
		}
		return nil
	})

	if err != nil {
		result.Warnings = append(result.Warnings, ValidationIssue{
			Message: fmt.Sprintf("error walking directory: %v", err),
		})
	}

	return result
}

func validateFile(path string, netsnmp map[string]*NormalizedNode, result *ValidationResult) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
	if err != nil {
		result.Warnings = append(result.Warnings, ValidationIssue{
			File:    path,
			Message: fmt.Sprintf("parse error: %v", err),
		})
		return
	}

	ast.Inspect(f, func(n ast.Node) bool {
		cl, ok := n.(*ast.CompositeLit)
		if !ok {
			return true
		}

		testCase := extractTableTestCase(cl)
		if testCase != nil {
			validateTableTestCase(path, testCase, netsnmp, result)
		}

		return true
	})
}

type extractedTestCase struct {
	TableName  string
	RowName    string
	Module     string
	IndexNames []string
	HasImplied bool
}

func extractTableTestCase(cl *ast.CompositeLit) *extractedTestCase {
	tc := &extractedTestCase{}
	hasRowName := false
	hasIndexNames := false

	for _, elt := range cl.Elts {
		kv, ok := elt.(*ast.KeyValueExpr)
		if !ok {
			continue
		}

		key, ok := kv.Key.(*ast.Ident)
		if !ok {
			continue
		}

		switch key.Name {
		case "TableName":
			if lit, ok := kv.Value.(*ast.BasicLit); ok && lit.Kind == token.STRING {
				tc.TableName, _ = strconv.Unquote(lit.Value)
			}
		case "RowName":
			if lit, ok := kv.Value.(*ast.BasicLit); ok && lit.Kind == token.STRING {
				tc.RowName, _ = strconv.Unquote(lit.Value)
				hasRowName = true
			}
		case "Module":
			if lit, ok := kv.Value.(*ast.BasicLit); ok && lit.Kind == token.STRING {
				tc.Module, _ = strconv.Unquote(lit.Value)
			}
		case "IndexNames":
			if cl, ok := kv.Value.(*ast.CompositeLit); ok {
				for _, e := range cl.Elts {
					if lit, ok := e.(*ast.BasicLit); ok && lit.Kind == token.STRING {
						s, _ := strconv.Unquote(lit.Value)
						tc.IndexNames = append(tc.IndexNames, s)
					}
				}
				hasIndexNames = true
			}
		case "HasImplied":
			if id, ok := kv.Value.(*ast.Ident); ok {
				tc.HasImplied = id.Name == "true"
			}
		}
	}

	if hasRowName && hasIndexNames {
		return tc
	}
	return nil
}

func validateTableTestCase(file string, tc *extractedTestCase, netsnmp map[string]*NormalizedNode, result *ValidationResult) {
	result.TestsValidated++

	key := tc.Module + "::" + tc.RowName
	nsNode := netsnmp[key]
	if nsNode == nil {
		nsNode = netsnmp[tc.RowName]
	}

	if nsNode == nil {
		result.TestsFailed++
		result.Failures = append(result.Failures, ValidationIssue{
			File:     file,
			TestName: tc.RowName,
			Field:    "existence",
			Expected: "found in net-snmp",
			Actual:   "not found",
			Message:  fmt.Sprintf("row %s::%s not found in net-snmp", tc.Module, tc.RowName),
		})
		return
	}

	var nsIndexNames []string
	for _, idx := range nsNode.Indexes {
		nsIndexNames = append(nsIndexNames, idx.Name)
	}

	if !stringsEqual(tc.IndexNames, nsIndexNames) {
		result.TestsFailed++
		result.Failures = append(result.Failures, ValidationIssue{
			File:     file,
			TestName: tc.RowName,
			Field:    "IndexNames",
			Expected: fmt.Sprintf("%v", nsIndexNames),
			Actual:   fmt.Sprintf("%v", tc.IndexNames),
			Message:  "index names mismatch",
		})
		return
	}

	nsHasImplied := false
	for _, idx := range nsNode.Indexes {
		if idx.Implied {
			nsHasImplied = true
			break
		}
	}

	if tc.HasImplied != nsHasImplied {
		result.TestsFailed++
		result.Failures = append(result.Failures, ValidationIssue{
			File:     file,
			TestName: tc.RowName,
			Field:    "HasImplied",
			Expected: fmt.Sprintf("%v", nsHasImplied),
			Actual:   fmt.Sprintf("%v", tc.HasImplied),
			Message:  "IMPLIED flag mismatch",
		})
		return
	}

	result.TestsPassed++
}

func stringsEqual(a, b []string) bool {
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

func printValidationResult(w io.Writer, result *ValidationResult) {
	fmt.Fprintln(w, strings.Repeat("=", 70))
	fmt.Fprintln(w, "TEST VALIDATION RESULTS")
	fmt.Fprintln(w, strings.Repeat("=", 70))

	fmt.Fprintf(w, "\nFiles checked:   %d\n", result.FilesChecked)
	fmt.Fprintf(w, "Tests validated: %d\n", result.TestsValidated)
	fmt.Fprintf(w, "Tests passed:    %d\n", result.TestsPassed)
	fmt.Fprintf(w, "Tests failed:    %d\n", result.TestsFailed)

	if len(result.Failures) > 0 {
		fmt.Fprintf(w, "\nFailures:\n")
		for _, f := range result.Failures {
			fmt.Fprintf(w, "  %s: %s\n", f.TestName, f.Message)
			fmt.Fprintf(w, "    file: %s\n", f.File)
			fmt.Fprintf(w, "    expected: %s\n", f.Expected)
			fmt.Fprintf(w, "    actual:   %s\n", f.Actual)
		}
	}

	if len(result.Warnings) > 0 {
		fmt.Fprintf(w, "\nWarnings:\n")
		for _, warn := range result.Warnings {
			fmt.Fprintf(w, "  %s\n", warn.Message)
		}
	}

	if result.TestsFailed == 0 && result.TestsValidated > 0 {
		fmt.Fprintf(w, "\nAll tests validated successfully!\n")
	}
}
