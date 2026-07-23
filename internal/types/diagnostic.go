package types

import (
	"fmt"
	"path"
	"slices"
	"strings"
)

// Diagnostic represents an issue found during parsing or resolution.
//
// Line/Column mark the inclusive start of the offending region. EndLine/EndColumn
// mark the exclusive end (i.e. the line/column of the byte just past the region),
// matching the [Start, End) semantics of [Span]. Both end fields are 0 when no
// useful end position is available, in which case consumers should treat the
// region as point-sized at Line/Column.
type Diagnostic struct {
	Severity  Severity
	Code      string // e.g., "identifier-underscore", "import-not-found"
	Message   string
	Module    string // source module name
	Line      int    // 1-based start line, 0 if not applicable
	Column    int    // 1-based start column, 0 if not applicable
	EndLine   int    // 1-based end line (exclusive), 0 if not applicable
	EndColumn int    // 1-based end column (exclusive), 0 if not applicable
}

// String returns a human-readable representation of the diagnostic.
// Format: "[severity] module:line:col: message" with location parts omitted when zero.
func (d Diagnostic) String() string {
	var b strings.Builder
	b.WriteByte('[')
	b.WriteString(d.Severity.String())
	b.WriteByte(']')
	b.WriteByte(' ')
	if d.Module != "" {
		b.WriteString(d.Module)
		if d.Line > 0 {
			fmt.Fprintf(&b, ":%d", d.Line)
			if d.Column > 0 {
				fmt.Fprintf(&b, ":%d", d.Column)
			}
		}
		b.WriteString(": ")
	}
	b.WriteString(d.Message)
	return b.String()
}

// DiagnosticConfig controls diagnostic reporting and failure policy.
//
// This is independent of [ResolverStrictness], which controls resolver
// fallback behavior. DiagnosticConfig determines which diagnostics are
// reported and which cause loading to fail, regardless of how permissive
// the resolver is. For best-effort loading of vendor MIB sets, configure
// both permissive resolution and FailAt = SeverityFatal.
type DiagnosticConfig struct {
	// Reporting controls baseline diagnostic output verbosity.
	Reporting ReportingLevel

	// FailAt sets the severity threshold for failure.
	// If any diagnostic has severity <= FailAt, loading returns
	// ErrDiagnosticThreshold (the Mib is still returned with all
	// resolved data). Lower severity numbers are more severe.
	// Default is SeveritySevere. Set to SeverityFatal for best-effort
	// loading that only fails on unrecoverable parse errors.
	FailAt Severity

	// Overrides change severity for specific diagnostic codes.
	// Use to upgrade/downgrade specific checks.
	Overrides map[string]Severity

	// Ignore lists diagnostic codes to suppress entirely.
	// Supports glob patterns (e.g., "identifier-*").
	Ignore []string
}

// ConfigForReporting returns the diagnostic configuration preset for the given
// reporting level. Unknown levels fall back to DefaultConfig.
func ConfigForReporting(level ReportingLevel) DiagnosticConfig {
	switch level {
	case ReportingVerbose:
		return VerboseConfig()
	case ReportingDefault:
		return DefaultConfig()
	case ReportingQuiet:
		return QuietConfig()
	case ReportingSilent:
		return SilentConfig()
	default:
		return DefaultConfig()
	}
}

// DefaultConfig returns the default diagnostic configuration.
func DefaultConfig() DiagnosticConfig {
	return DiagnosticConfig{
		Reporting: ReportingDefault,
		FailAt:    SeveritySevere,
	}
}

// VerboseConfig returns a verbose reporting profile.
func VerboseConfig() DiagnosticConfig {
	return DiagnosticConfig{
		Reporting: ReportingVerbose,
		FailAt:    SeveritySevere,
	}
}

// QuietConfig returns a low-noise reporting profile.
func QuietConfig() DiagnosticConfig {
	return DiagnosticConfig{
		Reporting: ReportingQuiet,
		FailAt:    SeveritySevere,
	}
}

// SilentConfig returns a silent configuration that suppresses all diagnostics.
// Only fatal errors that prevent parsing are reported.
func SilentConfig() DiagnosticConfig {
	return DiagnosticConfig{
		Reporting: ReportingSilent,
		FailAt:    SeverityFatal,
	}
}

// ShouldReport returns true if a diagnostic with the given code and severity
// should be reported under this configuration.
//
// Evaluation order: Overrides are applied first, then fatal check, then
// Ignore, then Reporting. Fatal diagnostics are always reported regardless of
// Ignore list or Reporting (unless overridden to a non-fatal severity).
//
// Reporting controls the reporting threshold:
//   - Verbose: report all diagnostics (sev 0-6)
//   - Default: report Minor and above (sev 0-3)
//   - Quiet: report Error and above (sev 0-2)
//   - Silent: report nothing (except fatal, handled above)
//
// Lower severity numbers are more severe (Fatal=0, Info=6).
func (c DiagnosticConfig) ShouldReport(code string, sev Severity) bool {
	if override, ok := c.Overrides[code]; ok {
		sev = override
	}

	// Fatal diagnostics are always reported regardless of Ignore or Level.
	if sev <= SeverityFatal {
		return true
	}

	if slices.ContainsFunc(c.Ignore, func(pattern string) bool {
		return MatchGlob(pattern, code)
	}) {
		return false
	}

	return int(sev) <= c.maxReportedSeverity()
}

// maxReportedSeverity returns the maximum severity number (least severe)
// that should be reported at the current reporting level.
// Returns -1 for Silent (report nothing except fatal, handled above).
func (c DiagnosticConfig) maxReportedSeverity() int {
	switch c.Reporting {
	case ReportingVerbose:
		return int(SeverityInfo) // report everything
	case ReportingDefault:
		return int(SeverityMinor) // report sev 0-3
	case ReportingQuiet:
		return int(SeverityError) // report sev 0-2
	default:
		return -1 // silent: report nothing (fatal handled by caller)
	}
}

// ShouldFail returns true if a diagnostic with the given severity should
// cause loading to fail.
func (c DiagnosticConfig) ShouldFail(sev Severity) bool {
	return sev <= c.FailAt
}

// MatchGlob performs glob matching on diagnostic codes using path.Match
// semantics. Supports *, ?, and [] character classes. Diagnostic codes
// contain no slashes, so * matches any sequence of characters.
func MatchGlob(pattern, s string) bool {
	ok, _ := path.Match(pattern, s)
	return ok
}
