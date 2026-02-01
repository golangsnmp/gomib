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
	"path/filepath"
	"slices"
	"strings"

	"github.com/golangsnmp/gomib"
	"github.com/golangsnmp/gomib/mib"
)

// AcceptResult holds the acceptance test results.
type AcceptResult struct {
	TotalModules   int            `json:"total_modules"`
	BothPass       int            `json:"both_pass"`
	BothFail       int            `json:"both_fail"`
	OnlyGomibPass  int            `json:"only_gomib_pass"`
	OnlyLibsmiPass int            `json:"only_libsmi_pass"`
	Modules        []ModuleAccept `json:"modules,omitempty"`
	Discrepancies  []ModuleAccept `json:"discrepancies,omitempty"`
}

// ModuleAccept holds per-module acceptance status.
type ModuleAccept struct {
	Name         string   `json:"name"`
	GomibPass    bool     `json:"gomib_pass"`
	LibsmiPass   bool     `json:"libsmi_pass"`
	GomibErrors  int      `json:"gomib_errors,omitempty"`
	LibsmiErrors int      `json:"libsmi_errors,omitempty"`
	GomibDiags   []string `json:"gomib_diags,omitempty"`
	LibsmiDiags  []string `json:"libsmi_diags,omitempty"`
}

func cmdAccept(args []string) int {
	fs := flag.NewFlagSet("accept", flag.ExitOnError)
	level := fs.Int("level", 2, "Error level threshold (0-6)")
	showAll := fs.Bool("all", false, "Show all modules, not just discrepancies")
	details := fs.Bool("details", false, "Show diagnostic messages for discrepancies")
	diagLimit := fs.Int("diag-limit", 3, "Max diagnostics to show per module (with -details)")

	fs.Usage = func() {
		fmt.Fprintf(fs.Output(), `Usage: gomib-libsmi accept [options] [MODULE...]

Test which MIBs pass/fail in gomib vs libsmi.
If no modules specified, tests all MIBs found in search paths.

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

	// If no modules specified, find all
	if len(modules) == 0 {
		modules = findAllModules(mibPaths)
	}

	result := testAcceptance(modules, mibPaths, *level, *showAll, *details, *diagLimit)

	if jsonOutput {
		enc := json.NewEncoder(out)
		enc.SetIndent("", "  ")
		if err := enc.Encode(result); err != nil {
			printError("json encode failed: %v", err)
			return 1
		}
	} else {
		printAcceptResult(out, result, *showAll, *details)
	}

	if len(result.Discrepancies) > 0 {
		return 1
	}
	return 0
}

func testAcceptance(modules []string, mibPaths []string, level int, showAll bool, collectDiags bool, diagLimit int) *AcceptResult {
	result := &AcceptResult{
		TotalModules: len(modules),
	}

	// Build gomib source
	var sources []gomib.Source
	for _, p := range mibPaths {
		src, err := gomib.DirTree(p)
		if err != nil {
			continue
		}
		sources = append(sources, src)
	}

	var gomibSource gomib.Source
	if len(sources) == 1 {
		gomibSource = sources[0]
	} else if len(sources) > 1 {
		gomibSource = gomib.Multi(sources...)
	}

	// Initialize libsmi
	libsmiPath := BuildMIBPath(expandDirs(mibPaths))
	InitLibsmi(libsmiPath, level)
	defer CleanupLibsmi()

	// Test each module
	for _, mod := range modules {
		ma := ModuleAccept{Name: mod}

		// Test with gomib
		if gomibSource != nil {
			cfg := mib.DiagnosticConfig{
				Level:  mib.StrictnessLevel(level),
				FailAt: mib.Severity(level),
			}

			ctx := context.Background()
			m, err := gomib.LoadModules(ctx, []string{mod}, gomibSource, gomib.WithDiagnosticConfig(cfg))
			if err == nil && m != nil {
				// Count errors at or above threshold
				for _, d := range m.Diagnostics() {
					if int(d.Severity) <= level {
						ma.GomibErrors++
						if collectDiags && len(ma.GomibDiags) < diagLimit {
							ma.GomibDiags = append(ma.GomibDiags, fmt.Sprintf("[%s] %s", d.Code, d.Message))
						}
					}
				}
				ma.GomibPass = ma.GomibErrors == 0
			}
		}

		// Test with libsmi
		ClearErrors()
		loaded := LoadModule(mod)
		diags := GetDiagnostics()

		for _, d := range diags {
			if d.Severity <= level {
				ma.LibsmiErrors++
				if collectDiags && len(ma.LibsmiDiags) < diagLimit {
					ma.LibsmiDiags = append(ma.LibsmiDiags, fmt.Sprintf("[%s] %s", d.Tag, d.Message))
				}
			}
		}
		ma.LibsmiPass = loaded && ma.LibsmiErrors == 0

		// Categorize
		isDiscrepancy := false
		if ma.GomibPass && ma.LibsmiPass {
			result.BothPass++
		} else if !ma.GomibPass && !ma.LibsmiPass {
			result.BothFail++
		} else if ma.GomibPass && !ma.LibsmiPass {
			result.OnlyGomibPass++
			isDiscrepancy = true
		} else {
			result.OnlyLibsmiPass++
			isDiscrepancy = true
		}

		if isDiscrepancy {
			result.Discrepancies = append(result.Discrepancies, ma)
		} else if !collectDiags {
			// Clear diags if not a discrepancy and not collecting
			ma.GomibDiags = nil
			ma.LibsmiDiags = nil
		}

		if showAll {
			result.Modules = append(result.Modules, ma)
		}
	}

	return result
}

func findAllModules(mibPaths []string) []string {
	seen := make(map[string]bool)
	var modules []string

	exts := map[string]bool{
		"":     true,
		".mib": true,
		".smi": true,
		".txt": true,
		".my":  true,
	}

	for _, root := range mibPaths {
		filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return nil
			}

			ext := strings.ToLower(filepath.Ext(path))
			if !exts[ext] {
				return nil
			}

			base := filepath.Base(path)
			name := strings.TrimSuffix(base, ext)

			if !seen[name] {
				seen[name] = true
				modules = append(modules, name)
			}
			return nil
		})
	}

	slices.Sort(modules)
	return modules
}

func printAcceptResult(w io.Writer, result *AcceptResult, showAll bool, showDetails bool) {
	fmt.Fprintln(w, strings.Repeat("=", 70))
	fmt.Fprintln(w, "PARSER ACCEPTANCE TEST RESULTS")
	fmt.Fprintln(w, strings.Repeat("=", 70))

	fmt.Fprintf(w, "\nTotal modules tested: %d\n", result.TotalModules)
	fmt.Fprintf(w, "\nResults:\n")
	fmt.Fprintf(w, "  Both pass:        %d\n", result.BothPass)
	fmt.Fprintf(w, "  Both fail:        %d\n", result.BothFail)
	fmt.Fprintf(w, "  Only gomib pass:  %d\n", result.OnlyGomibPass)
	fmt.Fprintf(w, "  Only libsmi pass: %d\n", result.OnlyLibsmiPass)

	if len(result.Discrepancies) > 0 {
		fmt.Fprintf(w, "\nDiscrepancies (%d):\n", len(result.Discrepancies))
		for _, m := range result.Discrepancies {
			gomibStatus := "FAIL"
			if m.GomibPass {
				gomibStatus = "PASS"
			}
			libsmiStatus := "FAIL"
			if m.LibsmiPass {
				libsmiStatus = "PASS"
			}
			fmt.Fprintf(w, "  %-40s gomib=%s (errors=%d) libsmi=%s (errors=%d)\n",
				m.Name, gomibStatus, m.GomibErrors, libsmiStatus, m.LibsmiErrors)

			if showDetails {
				if len(m.GomibDiags) > 0 {
					fmt.Fprintf(w, "    gomib diagnostics:\n")
					for _, d := range m.GomibDiags {
						fmt.Fprintf(w, "      %s\n", truncateMsg(d, 70))
					}
					if m.GomibErrors > len(m.GomibDiags) {
						fmt.Fprintf(w, "      ... and %d more\n", m.GomibErrors-len(m.GomibDiags))
					}
				}
				if len(m.LibsmiDiags) > 0 {
					fmt.Fprintf(w, "    libsmi diagnostics:\n")
					for _, d := range m.LibsmiDiags {
						fmt.Fprintf(w, "      %s\n", truncateMsg(d, 70))
					}
					if m.LibsmiErrors > len(m.LibsmiDiags) {
						fmt.Fprintf(w, "      ... and %d more\n", m.LibsmiErrors-len(m.LibsmiDiags))
					}
				}
			}
		}
	}

	if showAll && len(result.Modules) > 0 {
		fmt.Fprintf(w, "\nAll modules:\n")
		for _, m := range result.Modules {
			gomibStatus := "FAIL"
			if m.GomibPass {
				gomibStatus = "PASS"
			}
			libsmiStatus := "FAIL"
			if m.LibsmiPass {
				libsmiStatus = "PASS"
			}
			fmt.Fprintf(w, "  %-40s gomib=%s libsmi=%s\n", m.Name, gomibStatus, libsmiStatus)
		}
	}

	if len(result.Discrepancies) == 0 {
		fmt.Fprintf(w, "\nAll modules have consistent acceptance between gomib and libsmi.\n")
	}
}

func truncateMsg(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
