package main

import (
	"cmp"
	"flag"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/golangsnmp/gomib"
	"github.com/golangsnmp/gomib/internal/types"
	"github.com/golangsnmp/gomib/mib"
)

const lintUsage = `gomib lint - Check MIB modules for issues

Usage:
  gomib lint [options] [MODULE...]

Lints all available modules when no MODULE is specified.

Options:
  --level N       Report errors and warnings up to severity N (0-6, default: 3)
  --fail-on N     Exit non-zero if any diagnostic at severity N or below (default: 2)
  --ignore CODE   Ignore diagnostic codes (repeatable, supports globs like "identifier-*")
  --only CODE     Only report these codes (repeatable)
  --format FMT    Output format: text, json, sarif, compact (default: text)
  --group-by KEY  Group output: module, code, severity (default: none)
  --summary       Show summary only (counts by severity)
  --quiet         No output, exit code only
  --list-codes    List all diagnostic codes and exit
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
  gomib lint -p testdata/corpus/primary
  gomib lint --level 6 IF-MIB                 # Show all diagnostics
  gomib lint --level 1 IF-MIB                 # Only fatal and severe
  gomib lint --fail-on 3 IF-MIB              # Fail on minor or worse
  gomib lint --ignore "identifier-*" IF-MIB  # Skip identifier checks
  gomib lint --format json IF-MIB            # JSON output
  gomib lint --format sarif IF-MIB           # SARIF for IDE/CI
  gomib lint --group-by code IF-MIB          # Group by diagnostic code
  gomib lint --summary IF-MIB                # Just show counts
`

type lintConfig struct {
	level   mib.Severity
	failOn  mib.Severity
	ignore  []string
	only    []string
	format  string
	groupBy string
	summary bool
	quiet   bool
}

// severityFlag implements flag.Value for mib.Severity.
type severityFlag mib.Severity

func (f *severityFlag) Set(s string) error {
	v, err := strconv.Atoi(s)
	if err != nil || v < 0 || v > int(mib.SeverityInfo) {
		return fmt.Errorf("severity must be 0-%d", mib.SeverityInfo)
	}
	*f = severityFlag(v)
	return nil
}

func (f *severityFlag) String() string { return strconv.Itoa(int(*f)) }

type lintResult struct {
	Diagnostics []lintDiagnostic `json:"diagnostics,omitempty"`
	Summary     lintSummary      `json:"summary"`
	ExitCode    int              `json:"-"`
}

type lintDiagnostic struct {
	Severity    string       `json:"severity"`
	SeverityNum mib.Severity `json:"severity_num"`
	Code        string       `json:"code"`
	Message     string       `json:"message"`
	Module      string       `json:"module,omitempty"`
	Line        int          `json:"line,omitempty"`
	Column      int          `json:"column,omitempty"`
	RuleID      string       `json:"rule_id,omitempty"` // For SARIF
}

type lintSummary struct {
	Total      int            `json:"total"`
	BySeverity map[string]int `json:"by_severity"`
	ByCode     map[string]int `json:"by_code,omitempty"`
	Modules    int            `json:"modules"`
}

func (c *cli) cmdLint(args []string) int {
	fs := flag.NewFlagSet("lint", flag.ContinueOnError)
	fs.Usage = func() { fmt.Fprint(c.stderr, lintUsage) }

	cfg := lintConfig{
		level:  mib.SeverityMinor,
		failOn: mib.SeverityError,
		format: "text",
	}

	fs.Var((*severityFlag)(&cfg.level), "level", "report threshold")
	fs.Var((*severityFlag)(&cfg.failOn), "fail-on", "failure threshold")
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
	listCodes := fs.Bool("list-codes", false, "list all diagnostic codes")
	help := addHelpFlag(fs)

	if err := fs.Parse(args); err != nil {
		return exitError
	}

	if c.checkHelp(help, lintUsage) {
		return exitOK
	}

	if *listCodes {
		c.printDiagnosticCodes()
		return exitOK
	}

	modules := fs.Args()
	if len(modules) == 0 {
		modules = nil
	}

	switch cfg.format {
	case "text", "json", "sarif", "compact":
		// ok
	default:
		c.printError("unknown format: %s", cfg.format)
		return exitError
	}

	switch cfg.groupBy {
	case "", "module", "code", "severity":
		// ok
	default:
		c.printError("unknown group-by: %s", cfg.groupBy)
		return exitError
	}

	result := c.runLint(modules, &cfg)

	if !cfg.quiet {
		var err error
		switch cfg.format {
		case "json":
			err = c.printLintJSON(result)
		case "sarif":
			err = c.printLintSARIF(result)
		case "compact":
			c.printLintCompact(result, &cfg)
		default:
			c.printLintText(result, &cfg)
		}
		if err != nil {
			c.printError("output encoding failed: %v", err)
			return exitError
		}
	}

	return result.ExitCode
}

func (c *cli) runLint(modules []string, cfg *lintConfig) *lintResult {
	// Use strict mode for collection so all diagnostics are gathered.
	// The lint command filters by --level itself (smilint-style: higher = more verbose).
	diagCfg := mib.DiagnosticConfig{
		Reporting: mib.ReportingVerbose,
		FailAt:    mib.SeverityFatal, // We handle failure ourselves
		Ignore:    cfg.ignore,
	}

	m, err := c.loadMibWithOpts(modules, gomib.WithDiagnosticConfig(diagCfg))

	result := &lintResult{
		Summary: lintSummary{
			BySeverity: make(map[string]int),
			ByCode:     make(map[string]int),
		},
	}

	if err != nil {
		result.Diagnostics = append(result.Diagnostics, lintDiagnostic{
			Severity:    mib.SeverityFatal.String(),
			SeverityNum: mib.SeverityFatal,
			Code:        types.DiagParseError,
			Message:     err.Error(),
		})
		result.Summary.Total = 1
		result.Summary.BySeverity["fatal"] = 1
		result.ExitCode = 2
		return result
	}

	result.Summary.Modules = len(m.Modules())

	for _, d := range m.Diagnostics() {
		if d.Severity > cfg.level {
			continue
		}
		if len(cfg.only) > 0 && !matchesAny(d.Code, cfg.only) {
			continue
		}

		ld := lintDiagnostic{
			Severity:    d.Severity.String(),
			SeverityNum: d.Severity,
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

		if d.Severity <= cfg.failOn {
			result.ExitCode = 1
		}
	}

	return result
}

func matchesAny(code string, patterns []string) bool {
	for _, p := range patterns {
		if types.MatchGlob(p, code) {
			return true
		}
	}
	return false
}

func (c *cli) printLintText(result *lintResult, cfg *lintConfig) {
	if cfg.summary {
		c.printLintSummary(result)
		return
	}

	switch cfg.groupBy {
	case "module":
		c.printLintByModule(result)
	case "code":
		c.printLintByCode(result)
	case "severity":
		c.printLintBySeverity(result)
	default:
		c.printLintFlat(result)
	}

	if result.Summary.Total > 0 {
		fmt.Fprintln(c.stdout)
		c.printLintSummary(result)
	} else {
		fmt.Fprintf(c.stdout, "No issues found in %d modules\n", result.Summary.Modules)
	}
}

func (c *cli) printLintFlat(result *lintResult) {
	for i := range result.Diagnostics {
		c.printLintDiag(&result.Diagnostics[i], true)
	}
}

func (c *cli) printLintByModule(result *lintResult) {
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
		fmt.Fprintf(c.stdout, "\n%s:\n", mod)
		for i := range byMod[mod] {
			fmt.Fprintf(c.stdout, "  ")
			c.printLintDiag(&byMod[mod][i], true)
		}
	}
}

func (c *cli) printLintByCode(result *lintResult) {
	byCode := make(map[string][]lintDiagnostic)
	for _, d := range result.Diagnostics {
		code := d.Code
		if code == "" {
			code = "(unknown)"
		}
		byCode[code] = append(byCode[code], d)
	}

	codes := make([]string, 0, len(byCode))
	for cd := range byCode {
		codes = append(codes, cd)
	}
	slices.Sort(codes)

	for _, code := range codes {
		diags := byCode[code]
		fmt.Fprintf(c.stdout, "\n%s (%d):\n", code, len(diags))
		for _, d := range diags {
			if d.Module != "" {
				if d.Line > 0 {
					fmt.Fprintf(c.stdout, "  %s:%d: %s\n", d.Module, d.Line, d.Message)
				} else {
					fmt.Fprintf(c.stdout, "  %s: %s\n", d.Module, d.Message)
				}
			} else {
				fmt.Fprintf(c.stdout, "  %s\n", d.Message)
			}
		}
	}
}

func (c *cli) printLintBySeverity(result *lintResult) {
	bySev := make(map[mib.Severity][]lintDiagnostic)
	for _, d := range result.Diagnostics {
		bySev[d.SeverityNum] = append(bySev[d.SeverityNum], d)
	}

	sevs := make([]mib.Severity, 0, len(bySev))
	for s := range bySev {
		sevs = append(sevs, s)
	}
	slices.Sort(sevs)

	for _, sev := range sevs {
		diags := bySev[sev]
		if len(diags) > 0 {
			fmt.Fprintf(c.stdout, "\n%s (%d):\n", diags[0].Severity, len(diags))
			for i := range diags {
				fmt.Fprintf(c.stdout, "  ")
				c.printLintDiag(&diags[i], false)
			}
		}
	}
}

func (c *cli) printLintDiag(d *lintDiagnostic, includeSeverity bool) {
	var parts []string
	if includeSeverity {
		parts = append(parts, d.Severity+":")
	}
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
	fmt.Fprintln(c.stdout, strings.Join(parts, " "))
}

func (c *cli) printLintSummary(result *lintResult) {
	fmt.Fprintf(c.stdout, "Checked %d modules, found %d issues:\n", result.Summary.Modules, result.Summary.Total)

	sevOrder := mib.SeverityNames()
	for _, sev := range sevOrder {
		if count := result.Summary.BySeverity[sev]; count > 0 {
			fmt.Fprintf(c.stdout, "  %-8s %d\n", sev+":", count)
		}
	}
}

func (c *cli) printLintCompact(result *lintResult, cfg *lintConfig) {
	if cfg.summary {
		fmt.Fprintf(c.stdout, "%d issues", result.Summary.Total)
		parts := []string{}
		if n := result.Summary.BySeverity["error"]; n > 0 {
			parts = append(parts, fmt.Sprintf("%d errors", n))
		}
		if n := result.Summary.BySeverity["minor"]; n > 0 {
			parts = append(parts, fmt.Sprintf("%d minor", n))
		}
		if n := result.Summary.BySeverity["style"]; n > 0 {
			parts = append(parts, fmt.Sprintf("%d style", n))
		}
		if len(parts) > 0 {
			fmt.Fprintf(c.stdout, " (%s)", strings.Join(parts, ", "))
		}
		fmt.Fprintln(c.stdout)
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
		fmt.Fprintf(c.stdout, "%s: %s [%s] %s\n", loc, d.Severity, d.Code, d.Message)
	}
}

func (c *cli) printLintJSON(result *lintResult) error {
	return writeJSON(c.stdout, result, true)
}

// SARIF (Static Analysis Results Interchange Format) output
// https://sarifweb.azurewebsites.net/
func (c *cli) printLintSARIF(result *lintResult) error {
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

	return writeJSON(c.stdout, sarif, true)
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
	DefaultConfig    sarifDefaultConfig `json:"defaultConfiguration,omitzero"`
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

func severityToSARIF(sev mib.Severity) string {
	switch {
	case sev <= mib.SeverityError:
		return "error"
	case sev <= mib.SeverityStyle:
		return "warning"
	default:
		return "note"
	}
}

func (c *cli) printDiagnosticCodes() {
	currentPhase := ""
	for _, info := range types.AllDiagnosticCodes() {
		if info.Phase != currentPhase {
			if currentPhase != "" {
				fmt.Fprintln(c.stdout)
			}
			fmt.Fprintf(c.stdout, "%s:\n", info.Phase)
			currentPhase = info.Phase
		}
		fmt.Fprintf(c.stdout, "  %-36s %s\n", info.Code, info.Severity)
	}
}
