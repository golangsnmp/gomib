package main

import (
	"cmp"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/golangsnmp/gomib"
	"github.com/golangsnmp/gomib/mib"
)

const lintUsage = `gomib lint - Check MIB modules for issues

Usage:
  gomib lint [options] MODULE...

Options:
  --level N       Report diagnostics at severity N or below (0-6, default: 3)
  --fail-on N     Exit non-zero if any diagnostic at severity N or below (default: 2)
  --ignore CODE   Ignore diagnostic codes (repeatable, supports globs like "identifier-*")
  --only CODE     Only report these codes (repeatable)
  --format FMT    Output format: text, json, sarif, compact (default: text)
  --group-by KEY  Group output: module, code, severity (default: none)
  --summary       Show summary only (counts by severity)
  --quiet         No output, exit code only
  -h, --help      Show help

Severity Levels:
  0 = fatal       Cannot continue
  1 = severe      Semantics changed
  2 = error       Should correct
  3 = minor       Minor issue
  4 = style       Style recommendation
  5 = warning     Might be correct
  6 = info        Informational

Examples:
  gomib lint IF-MIB
  gomib lint --level 4 IF-MIB                 # Include style warnings
  gomib lint --fail-on 3 IF-MIB              # Fail on minor or worse
  gomib lint --ignore "identifier-*" IF-MIB  # Skip identifier checks
  gomib lint --format json IF-MIB            # JSON output
  gomib lint --format sarif IF-MIB           # SARIF for IDE/CI
  gomib lint --group-by code IF-MIB          # Group by diagnostic code
  gomib lint --summary IF-MIB                # Just show counts
`

type lintConfig struct {
	level   int
	failOn  int
	ignore  []string
	only    []string
	format  string
	groupBy string
	summary bool
	quiet   bool
}

type lintResult struct {
	Diagnostics []lintDiagnostic `json:"diagnostics,omitempty"`
	Summary     lintSummary      `json:"summary"`
	ExitCode    int              `json:"-"`
}

type lintDiagnostic struct {
	Severity    string `json:"severity"`
	SeverityNum int    `json:"severity_num"`
	Code        string `json:"code"`
	Message     string `json:"message"`
	Module      string `json:"module,omitempty"`
	Line        int    `json:"line,omitempty"`
	Column      int    `json:"column,omitempty"`
	RuleID      string `json:"rule_id,omitempty"` // For SARIF
}

type lintSummary struct {
	Total      int            `json:"total"`
	BySeverity map[string]int `json:"by_severity"`
	ByCode     map[string]int `json:"by_code,omitempty"`
	Modules    int            `json:"modules"`
}

func cmdLint(args []string) int {
	fs := flag.NewFlagSet("lint", flag.ContinueOnError)
	fs.Usage = func() { fmt.Fprint(os.Stderr, lintUsage) }

	cfg := lintConfig{
		level:  3, // Default: report minor and above
		failOn: 2, // Default: fail on error and above
		format: "text",
	}

	fs.IntVar(&cfg.level, "level", cfg.level, "report threshold")
	fs.IntVar(&cfg.failOn, "fail-on", cfg.failOn, "failure threshold")
	fs.Func("ignore", "ignore codes", func(s string) error {
		cfg.ignore = append(cfg.ignore, s)
		return nil
	})
	fs.Func("only", "only report these codes", func(s string) error {
		cfg.only = append(cfg.only, s)
		return nil
	})
	fs.StringVar(&cfg.format, "format", cfg.format, "output format")
	fs.StringVar(&cfg.groupBy, "group-by", cfg.groupBy, "grouping key")
	fs.BoolVar(&cfg.summary, "summary", false, "summary only")
	fs.BoolVar(&cfg.quiet, "quiet", false, "no output")
	help := fs.Bool("h", false, "show help")
	fs.BoolVar(help, "help", false, "show help")

	if err := fs.Parse(args); err != nil {
		return 1
	}

	if *help || helpFlag {
		_, _ = fmt.Fprint(os.Stdout, lintUsage)
		return 0
	}

	modules := fs.Args()
	if len(modules) == 0 {
		printError("no modules specified")
		fmt.Fprint(os.Stderr, lintUsage)
		return 1
	}

	switch cfg.format {
	case "text", "json", "sarif", "compact":
		// ok
	default:
		printError("unknown format: %s", cfg.format)
		return 1
	}

	switch cfg.groupBy {
	case "", "module", "code", "severity":
		// ok
	default:
		printError("unknown group-by: %s", cfg.groupBy)
		return 1
	}

	result := runLint(modules, cfg)

	if !cfg.quiet {
		var err error
		switch cfg.format {
		case "json":
			err = printLintJSON(result)
		case "sarif":
			err = printLintSARIF(result)
		case "compact":
			printLintCompact(result, cfg)
		default:
			printLintText(result, cfg)
		}
		if err != nil {
			printError("output encoding failed: %v", err)
			return 1
		}
	}

	return result.ExitCode
}

func runLint(modules []string, cfg lintConfig) *lintResult {
	diagCfg := mib.DiagnosticConfig{
		Level:  mib.StrictnessLevel(cfg.level),
		FailAt: mib.SeverityFatal, // We handle failure ourselves
		Ignore: cfg.ignore,
	}

	m, err := loadMibWithOpts(modules, gomib.WithDiagnosticConfig(diagCfg))

	result := &lintResult{
		Summary: lintSummary{
			BySeverity: make(map[string]int),
			ByCode:     make(map[string]int),
		},
	}

	if err != nil {
		result.Diagnostics = append(result.Diagnostics, lintDiagnostic{
			Severity:    "fatal",
			SeverityNum: 0,
			Code:        "parse-error",
			Message:     err.Error(),
		})
		result.Summary.Total = 1
		result.Summary.BySeverity["fatal"] = 1
		result.ExitCode = 2
		return result
	}

	result.Summary.Modules = len(m.Modules())

	for _, d := range m.Diagnostics() {
		if len(cfg.only) > 0 && !matchesAny(d.Code, cfg.only) {
			continue
		}

		ld := lintDiagnostic{
			Severity:    d.Severity.String(),
			SeverityNum: int(d.Severity),
			Code:        d.Code,
			Message:     d.Message,
			Module:      d.Module,
			Line:        d.Line,
			Column:      d.Column,
			RuleID:      d.Code,
		}
		result.Diagnostics = append(result.Diagnostics, ld)
		result.Summary.Total++
		result.Summary.BySeverity[d.Severity.String()]++
		result.Summary.ByCode[d.Code]++

		if int(d.Severity) <= cfg.failOn {
			result.ExitCode = 1
		}
	}

	return result
}

func matchesAny(code string, patterns []string) bool {
	for _, p := range patterns {
		if matchGlob(p, code) {
			return true
		}
	}
	return false
}

// matchGlob performs simple glob matching with * wildcard.
func matchGlob(pattern, s string) bool {
	if pattern == "*" {
		return true
	}
	if strings.HasSuffix(pattern, "*") {
		return strings.HasPrefix(s, pattern[:len(pattern)-1])
	}
	if strings.HasPrefix(pattern, "*") {
		return strings.HasSuffix(s, pattern[1:])
	}
	return pattern == s
}

func printLintText(result *lintResult, cfg lintConfig) {
	if cfg.summary {
		printLintSummary(result)
		return
	}

	switch cfg.groupBy {
	case "module":
		printLintByModule(result)
	case "code":
		printLintByCode(result)
	case "severity":
		printLintBySeverity(result)
	default:
		printLintFlat(result)
	}

	if result.Summary.Total > 0 {
		fmt.Println()
		printLintSummary(result)
	} else {
		fmt.Printf("No issues found in %d modules\n", result.Summary.Modules)
	}
}

func printLintFlat(result *lintResult) {
	for _, d := range result.Diagnostics {
		printLintDiagLine(d)
	}
}

func printLintByModule(result *lintResult) {
	byMod := make(map[string][]lintDiagnostic)
	for _, d := range result.Diagnostics {
		mod := d.Module
		if mod == "" {
			mod = "(unknown)"
		}
		byMod[mod] = append(byMod[mod], d)
	}

	mods := make([]string, 0, len(byMod))
	for m := range byMod {
		mods = append(mods, m)
	}
	slices.Sort(mods)

	for _, mod := range mods {
		fmt.Printf("\n%s:\n", mod)
		for _, d := range byMod[mod] {
			fmt.Printf("  ")
			printLintDiagLine(d)
		}
	}
}

func printLintByCode(result *lintResult) {
	byCode := make(map[string][]lintDiagnostic)
	for _, d := range result.Diagnostics {
		code := d.Code
		if code == "" {
			code = "(unknown)"
		}
		byCode[code] = append(byCode[code], d)
	}

	codes := make([]string, 0, len(byCode))
	for c := range byCode {
		codes = append(codes, c)
	}
	slices.Sort(codes)

	for _, code := range codes {
		diags := byCode[code]
		fmt.Printf("\n%s (%d):\n", code, len(diags))
		for _, d := range diags {
			if d.Module != "" {
				if d.Line > 0 {
					fmt.Printf("  %s:%d: %s\n", d.Module, d.Line, d.Message)
				} else {
					fmt.Printf("  %s: %s\n", d.Module, d.Message)
				}
			} else {
				fmt.Printf("  %s\n", d.Message)
			}
		}
	}
}

func printLintBySeverity(result *lintResult) {
	bySev := make(map[int][]lintDiagnostic)
	for _, d := range result.Diagnostics {
		bySev[d.SeverityNum] = append(bySev[d.SeverityNum], d)
	}

	sevs := make([]int, 0, len(bySev))
	for s := range bySev {
		sevs = append(sevs, s)
	}
	slices.Sort(sevs)

	for _, sev := range sevs {
		diags := bySev[sev]
		if len(diags) > 0 {
			fmt.Printf("\n%s (%d):\n", diags[0].Severity, len(diags))
			for _, d := range diags {
				fmt.Printf("  ")
				printLintDiagLineNoSeverity(d)
			}
		}
	}
}

func printLintDiagLine(d lintDiagnostic) {
	parts := []string{d.Severity + ":"}
	if d.Code != "" {
		parts = append(parts, "["+d.Code+"]")
	}
	if d.Module != "" {
		if d.Line > 0 {
			parts = append(parts, fmt.Sprintf("%s:%d:", d.Module, d.Line))
		} else {
			parts = append(parts, d.Module+":")
		}
	}
	parts = append(parts, d.Message)
	fmt.Println(strings.Join(parts, " "))
}

func printLintDiagLineNoSeverity(d lintDiagnostic) {
	parts := []string{}
	if d.Code != "" {
		parts = append(parts, "["+d.Code+"]")
	}
	if d.Module != "" {
		if d.Line > 0 {
			parts = append(parts, fmt.Sprintf("%s:%d:", d.Module, d.Line))
		} else {
			parts = append(parts, d.Module+":")
		}
	}
	parts = append(parts, d.Message)
	fmt.Println(strings.Join(parts, " "))
}

func printLintSummary(result *lintResult) {
	fmt.Printf("Checked %d modules, found %d issues:\n", result.Summary.Modules, result.Summary.Total)

	sevOrder := []string{"fatal", "severe", "error", "minor", "style", "warning", "info"}
	for _, sev := range sevOrder {
		if count := result.Summary.BySeverity[sev]; count > 0 {
			fmt.Printf("  %-8s %d\n", sev+":", count)
		}
	}
}

func printLintCompact(result *lintResult, cfg lintConfig) {
	if cfg.summary {
		fmt.Printf("%d issues", result.Summary.Total)
		parts := []string{}
		if c := result.Summary.BySeverity["error"]; c > 0 {
			parts = append(parts, fmt.Sprintf("%d errors", c))
		}
		if c := result.Summary.BySeverity["minor"]; c > 0 {
			parts = append(parts, fmt.Sprintf("%d minor", c))
		}
		if c := result.Summary.BySeverity["style"]; c > 0 {
			parts = append(parts, fmt.Sprintf("%d style", c))
		}
		if len(parts) > 0 {
			fmt.Printf(" (%s)", strings.Join(parts, ", "))
		}
		fmt.Println()
		return
	}

	for _, d := range result.Diagnostics {
		loc := d.Module
		if d.Line > 0 {
			loc = fmt.Sprintf("%s:%d", d.Module, d.Line)
			if d.Column > 0 {
				loc = fmt.Sprintf("%s:%d:%d", d.Module, d.Line, d.Column)
			}
		}
		fmt.Printf("%s: %s [%s] %s\n", loc, d.Severity, d.Code, d.Message)
	}
}

func printLintJSON(result *lintResult) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(result)
}

// SARIF (Static Analysis Results Interchange Format) output
// https://sarifweb.azurewebsites.net/
func printLintSARIF(result *lintResult) error {
	sarif := sarifOutput{
		Schema:  "https://raw.githubusercontent.com/oasis-tcs/sarif-spec/master/Schemata/sarif-schema-2.1.0.json",
		Version: "2.1.0",
		Runs: []sarifRun{{
			Tool: sarifTool{
				Driver: sarifDriver{
					Name:           "gomib",
					InformationURI: "https://github.com/golangsnmp/gomib",
					Rules:          buildSARIFRules(result),
				},
			},
			Results: buildSARIFResults(result),
		}},
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(sarif)
}

type sarifOutput struct {
	Schema  string     `json:"$schema"`
	Version string     `json:"version"`
	Runs    []sarifRun `json:"runs"`
}

type sarifRun struct {
	Tool    sarifTool     `json:"tool"`
	Results []sarifResult `json:"results"`
}

type sarifTool struct {
	Driver sarifDriver `json:"driver"`
}

type sarifDriver struct {
	Name           string      `json:"name"`
	InformationURI string      `json:"informationUri"`
	Rules          []sarifRule `json:"rules,omitempty"`
}

type sarifRule struct {
	ID               string             `json:"id"`
	ShortDescription sarifMessage       `json:"shortDescription"`
	DefaultConfig    sarifDefaultConfig `json:"defaultConfiguration,omitempty"`
}

type sarifDefaultConfig struct {
	Level string `json:"level"`
}

type sarifMessage struct {
	Text string `json:"text"`
}

type sarifResult struct {
	RuleID    string          `json:"ruleId"`
	Level     string          `json:"level"`
	Message   sarifMessage    `json:"message"`
	Locations []sarifLocation `json:"locations,omitempty"`
}

type sarifLocation struct {
	PhysicalLocation sarifPhysicalLocation `json:"physicalLocation"`
}

type sarifPhysicalLocation struct {
	ArtifactLocation sarifArtifact `json:"artifactLocation"`
	Region           *sarifRegion  `json:"region,omitempty"`
}

type sarifArtifact struct {
	URI string `json:"uri"`
}

type sarifRegion struct {
	StartLine   int `json:"startLine,omitempty"`
	StartColumn int `json:"startColumn,omitempty"`
}

func buildSARIFRules(result *lintResult) []sarifRule {
	seen := make(map[string]bool)
	var rules []sarifRule

	for _, d := range result.Diagnostics {
		if d.Code == "" || seen[d.Code] {
			continue
		}
		seen[d.Code] = true
		rules = append(rules, sarifRule{
			ID:               d.Code,
			ShortDescription: sarifMessage{Text: d.Code},
			DefaultConfig:    sarifDefaultConfig{Level: severityToSARIF(d.SeverityNum)},
		})
	}

	slices.SortFunc(rules, func(a, b sarifRule) int {
		return cmp.Compare(a.ID, b.ID)
	})
	return rules
}

func buildSARIFResults(result *lintResult) []sarifResult {
	var results []sarifResult

	for _, d := range result.Diagnostics {
		r := sarifResult{
			RuleID:  d.Code,
			Level:   severityToSARIF(d.SeverityNum),
			Message: sarifMessage{Text: d.Message},
		}

		if d.Module != "" {
			loc := sarifLocation{
				PhysicalLocation: sarifPhysicalLocation{
					ArtifactLocation: sarifArtifact{URI: d.Module},
				},
			}
			if d.Line > 0 {
				loc.PhysicalLocation.Region = &sarifRegion{
					StartLine:   d.Line,
					StartColumn: d.Column,
				}
			}
			r.Locations = append(r.Locations, loc)
		}

		results = append(results, r)
	}

	return results
}

func severityToSARIF(sev int) string {
	switch {
	case sev <= 2: // fatal, severe, error
		return "error"
	case sev <= 4: // minor, style
		return "warning"
	default: // warning, info
		return "note"
	}
}
