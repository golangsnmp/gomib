package types

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
	"testing"
)

// TestAllDiagnosticCodesComplete verifies that AllDiagnosticCodes contains
// every Diag* constant defined in diagcode.go.
func TestAllDiagnosticCodesComplete(t *testing.T) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "diagcode.go", nil, 0)
	if err != nil {
		t.Fatalf("failed to parse diagcode.go: %v", err)
	}

	// Collect all Diag* string constants from source.
	sourceConstants := make(map[string]string) // const name -> string value
	for _, decl := range f.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.CONST {
			continue
		}
		for _, spec := range genDecl.Specs {
			vs, ok := spec.(*ast.ValueSpec)
			if !ok {
				continue
			}
			for i, name := range vs.Names {
				if !strings.HasPrefix(name.Name, "Diag") {
					continue
				}
				if i >= len(vs.Values) {
					continue
				}
				lit, ok := vs.Values[i].(*ast.BasicLit)
				if !ok || lit.Kind != token.STRING {
					continue
				}
				// Strip quotes from the string literal value.
				val := strings.Trim(lit.Value, `"`)
				sourceConstants[name.Name] = val
			}
		}
	}

	if len(sourceConstants) == 0 {
		t.Fatal("found no Diag* constants in diagcode.go")
	}

	// Build set of codes present in AllDiagnosticCodes.
	allCodes := make(map[string]struct{})
	for _, info := range AllDiagnosticCodes() {
		allCodes[info.Code] = struct{}{}
	}

	// Every source constant must appear in AllDiagnosticCodes.
	for name, code := range sourceConstants {
		if _, ok := allCodes[code]; !ok {
			t.Errorf("constant %s (%q) is missing from AllDiagnosticCodes()", name, code)
		}
	}

	// Every entry in AllDiagnosticCodes must correspond to a source constant.
	sourceValues := make(map[string]struct{})
	for _, code := range sourceConstants {
		sourceValues[code] = struct{}{}
	}
	for _, info := range AllDiagnosticCodes() {
		if _, ok := sourceValues[info.Code]; !ok {
			t.Errorf("AllDiagnosticCodes contains %q which has no corresponding Diag* constant", info.Code)
		}
	}

	// No two Diag* constants should share the same string value.
	valueToName := make(map[string]string)
	for name, code := range sourceConstants {
		if prev, ok := valueToName[code]; ok {
			t.Errorf("duplicate diagnostic code value %q: used by both %s and %s", code, prev, name)
		}
		valueToName[code] = name
	}
}
