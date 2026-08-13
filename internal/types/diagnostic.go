package types

import (
	"fmt"
	"path"
	"slices"
	"strings"
)

// Diagnostic represents an issue found during parsing or resolution.
type Diagnostic struct {
	Severity Severity
	Code     string // e.g., "identifier-underscore", "import-not-found"
	Message  string
	Module   string // source module name
	Line     int    // 1-based line number, 0 if not applicable
	Column   int    // 1-based column, 0 if not applicable
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
// fallback behavior. Reporting, Overrides, Ignore, and FailAt are independent
// policies for retaining diagnostics, storing their effective severity,
// suppressing them, and deciding whether loading fails. For best-effort loading
// of vendor MIB sets, configure both permissive resolution and
// FailAt = SeverityFatal.
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

	// Overrides change severity for specific diagnostic codes. The effective
	// severity is stored on emitted diagnostics and used by failure checks.
	// Demoting a diagnostic does not suppress it; use Ignore for suppression.
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

// EffectiveSeverity returns the configured severity for a registered
// diagnostic code.
func (c DiagnosticConfig) EffectiveSeverity(code string) Severity {
	return c.effectiveSeverity(code, SeverityForCode(code))
}

func (c DiagnosticConfig) effectiveSeverity(code string, registered Severity) Severity {
	if override, ok := c.Overrides[code]; ok {
		return override
	}
	return registered
}

// IsIgnored reports whether code matches an Ignore pattern.
func (c DiagnosticConfig) IsIgnored(code string) bool {
	return slices.ContainsFunc(c.Ignore, func(pattern string) bool {
		return MatchGlob(pattern, code)
	})
}

// ReportsSeverity reports whether the reporting level includes sev. Fatal
// diagnostics are always reported, including at ReportingSilent.
func (c DiagnosticConfig) ReportsSeverity(sev Severity) bool {
	return sev <= SeverityFatal || int(sev) <= c.maxReportedSeverity()
}

// ShouldRetain reports whether a diagnostic should be stored. Promotions can
// bring diagnostics into the reporting level, while demotions do not discard
// diagnostics included by their registered severity. Effective fatal
// diagnostics are retained even when ignored or reporting is silent.
func (c DiagnosticConfig) ShouldRetain(code string) bool {
	registered := SeverityForCode(code)
	effective := c.effectiveSeverity(code, registered)
	if effective <= SeverityFatal {
		return true
	}
	if c.IsIgnored(code) {
		return false
	}
	return c.ReportsSeverity(registered) || c.ReportsSeverity(effective)
}

// ShouldReport reports whether the configured reporting and ignore policies
// select a diagnostic at its effective severity. It accepts an explicit
// registered severity for compatibility with callers using custom codes.
func (c DiagnosticConfig) ShouldReport(code string, registered Severity) bool {
	effective := c.effectiveSeverity(code, registered)
	if effective <= SeverityFatal {
		return true
	}
	return !c.IsIgnored(code) && c.ReportsSeverity(effective)
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
