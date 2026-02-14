//go:build cgo

//nolint:errcheck // CLI output, errors not critical
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/golangsnmp/gomib"
	"github.com/golangsnmp/gomib/mib"
)

// DiagComparison holds side-by-side diagnostic output from gomib and libsmi.
type DiagComparison struct {
	Module       string      `json:"module"`
	Level        int         `json:"level"`
	GomibDiags   []DiagEntry `json:"gomib_diagnostics"`
	LibsmiDiags  []DiagEntry `json:"libsmi_diagnostics"`
	OnlyInGomib  []DiagEntry `json:"only_in_gomib,omitempty"`
	OnlyInLibsmi []DiagEntry `json:"only_in_libsmi,omitempty"`
	Summary      DiagSummary `json:"summary"`
}

// DiagEntry holds one diagnostic from either parser.
type DiagEntry struct {
	Line     int    `json:"line"`
	Severity int    `json:"severity"`
	SevName  string `json:"severity_name"`
	Code     string `json:"code,omitempty"`
	Message  string `json:"message"`
}

// DiagSummary tallies diagnostics from each parser by severity.
type DiagSummary struct {
	GomibTotal   int            `json:"gomib_total"`
	LibsmiTotal  int            `json:"libsmi_total"`
	BySeverity   map[string]int `json:"by_severity"`
	CommonIssues int            `json:"common_issues"`
}

func cmdDiag(args []string) int {
	fs := flag.NewFlagSet("diag", flag.ContinueOnError)
	level := fs.Int("level", 3, "Error level threshold (0-6, lower=stricter)")

	fs.Usage = func() {
		fmt.Fprintf(fs.Output(), `Usage: gomib-libsmi diag [options] MODULE...

Compare parser diagnostics between gomib and libsmi (smilint).

Severity levels (libsmi-compatible):
  0 = fatal    (cannot continue)
  1 = severe   (semantics changed)
  2 = error    (should correct)
  3 = minor    (minor issue)
  4 = style    (style recommendation)
  5 = warning  (might be correct)
  6 = info     (informational)

Options:
`)
		fs.PrintDefaults()
	}

	if err := fs.Parse(args); err != nil {
		return 1
	}

	modules := fs.Args()
	if len(modules) == 0 {
		printError("at least one MODULE is required")
		return 1
	}

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

	results := make([]*DiagComparison, 0, len(modules))

	for _, mod := range modules {
		result := compareDiagnostics(mod, mibPaths, *level)
		results = append(results, result)
	}

	if jsonOutput {
		enc := json.NewEncoder(out)
		enc.SetIndent("", "  ")
		if err := enc.Encode(results); err != nil {
			printError("json encode failed: %v", err)
			return 1
		}
	} else {
		for _, result := range results {
			printDiagComparison(out, result)
		}
	}

	return 0
}

func compareDiagnostics(module string, mibPaths []string, level int) *DiagComparison {
	result := &DiagComparison{
		Module: module,
		Level:  level,
		Summary: DiagSummary{
			BySeverity: make(map[string]int),
		},
	}

	libsmiPath := BuildMIBPath(expandDirs(mibPaths))
	InitLibsmi(libsmiPath, level)
	defer CleanupLibsmi()

	ClearErrors()
	LoadModule(module)
	libsmiDiags := GetDiagnostics()

	for _, d := range libsmiDiags {
		entry := DiagEntry{
			Line:     d.Line,
			Severity: d.Severity,
			SevName:  SeverityName(d.Severity),
			Code:     d.Tag,
			Message:  d.Message,
		}
		result.LibsmiDiags = append(result.LibsmiDiags, entry)
		result.Summary.LibsmiTotal++
		result.Summary.BySeverity["libsmi:"+entry.SevName]++
	}

	if source := buildSource(mibPaths); source != nil {
		cfg := mib.DiagnosticConfig{
			Level:  mib.StrictnessLevel(level),
			FailAt: mib.SeverityFatal,
		}

		ctx := context.Background()
		m, err := gomib.Load(ctx, gomib.WithSource(source), gomib.WithModules(module), gomib.WithDiagnosticConfig(cfg))
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: gomib load failed for %s: %v\n", module, err)
		}

		if m != nil {
			for _, d := range m.Diagnostics() {
				entry := DiagEntry{
					Line:     d.Line,
					Severity: int(d.Severity),
					SevName:  severityToString(d.Severity),
					Code:     d.Code,
					Message:  d.Message,
				}
				result.GomibDiags = append(result.GomibDiags, entry)
				result.Summary.GomibTotal++
				result.Summary.BySeverity["gomib:"+entry.SevName]++
			}
		}
	}

	// Simple line-based matching to approximate common diagnostics
	gomibLines := make(map[int]bool)
	for _, d := range result.GomibDiags {
		gomibLines[d.Line] = true
	}

	libsmiLines := make(map[int]bool)
	for _, d := range result.LibsmiDiags {
		libsmiLines[d.Line] = true
	}

	for _, d := range result.GomibDiags {
		if !libsmiLines[d.Line] {
			result.OnlyInGomib = append(result.OnlyInGomib, d)
		} else {
			result.Summary.CommonIssues++
		}
	}

	for _, d := range result.LibsmiDiags {
		if !gomibLines[d.Line] {
			result.OnlyInLibsmi = append(result.OnlyInLibsmi, d)
		}
	}

	return result
}

func printDiagComparison(w io.Writer, result *DiagComparison) {
	fmt.Fprintln(w, strings.Repeat("=", 70))
	fmt.Fprintf(w, "DIAGNOSTIC COMPARISON: %s (level %d)\n", result.Module, result.Level)
	fmt.Fprintln(w, strings.Repeat("=", 70))

	fmt.Fprintf(w, "\nSummary:\n")
	fmt.Fprintf(w, "  gomib diagnostics:  %d\n", result.Summary.GomibTotal)
	fmt.Fprintf(w, "  libsmi diagnostics: %d\n", result.Summary.LibsmiTotal)
	fmt.Fprintf(w, "  common (by line):   %d\n", result.Summary.CommonIssues)

	if len(result.Summary.BySeverity) > 0 {
		fmt.Fprintf(w, "\nBy severity:\n")
		var keys []string
		for k := range result.Summary.BySeverity {
			keys = append(keys, k)
		}
		slices.Sort(keys)
		for _, k := range keys {
			fmt.Fprintf(w, "  %-20s %d\n", k+":", result.Summary.BySeverity[k])
		}
	}

	if len(result.OnlyInGomib) > 0 {
		fmt.Fprintf(w, "\nOnly in gomib (%d):\n", len(result.OnlyInGomib))
		for _, d := range result.OnlyInGomib[:min(10, len(result.OnlyInGomib))] {
			fmt.Fprintf(w, "  line %d [%s] %s: %s\n", d.Line, d.SevName, d.Code, truncate(d.Message, 50))
		}
		if len(result.OnlyInGomib) > 10 {
			fmt.Fprintf(w, "  ... and %d more\n", len(result.OnlyInGomib)-10)
		}
	}

	if len(result.OnlyInLibsmi) > 0 {
		fmt.Fprintf(w, "\nOnly in libsmi (%d):\n", len(result.OnlyInLibsmi))
		for _, d := range result.OnlyInLibsmi[:min(10, len(result.OnlyInLibsmi))] {
			fmt.Fprintf(w, "  line %d [%s] %s: %s\n", d.Line, d.SevName, d.Code, truncate(d.Message, 50))
		}
		if len(result.OnlyInLibsmi) > 10 {
			fmt.Fprintf(w, "  ... and %d more\n", len(result.OnlyInLibsmi)-10)
		}
	}

	fmt.Fprintln(w)
}

func severityToString(s gomib.Severity) string {
	return s.String()
}

func expandDirs(roots []string) []string {
	var dirs []string
	seen := make(map[string]bool)

	for _, root := range roots {
		info, err := os.Stat(root)
		if err != nil || !info.IsDir() {
			continue
		}

		filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			if d.IsDir() && !seen[path] {
				seen[path] = true
				dirs = append(dirs, path)
			}
			return nil
		})
	}

	return dirs
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
