package types

import (
	"fmt"
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

// DiagnosticConfig controls strictness and diagnostic filtering.
type DiagnosticConfig struct {
	// Level sets the base strictness level.
	// Diagnostics with severity > Level are suppressed.
	Level StrictnessLevel

	// FailAt sets the severity threshold for failure.
	// If any diagnostic has severity <= FailAt, loading fails.
	// Default (0) means fail on Fatal only.
	FailAt Severity

	// Overrides change severity for specific diagnostic codes.
	// Use to upgrade/downgrade specific checks.
	Overrides map[string]Severity

	// Ignore lists diagnostic codes to suppress entirely.
	// Supports glob patterns (e.g., "identifier-*").
	Ignore []string
}

// DefaultConfig returns the default diagnostic configuration (Normal strictness).
func DefaultConfig() DiagnosticConfig {
	return DiagnosticConfig{
		Level:  StrictnessNormal,
		FailAt: SeveritySevere,
	}
}

// StrictConfig returns a strict configuration for RFC compliance checking.
func StrictConfig() DiagnosticConfig {
	return DiagnosticConfig{
		Level:  StrictnessStrict,
		FailAt: SeveritySevere,
	}
}

// PermissiveConfig returns a permissive configuration for legacy/vendor MIBs.
// Suppresses common vendor MIB violations like underscores in identifiers.
//
// Ignored codes:
//   - identifier-underscore: vendor MIBs routinely use underscores (RFC style violation)
//   - identifier-length-32: emitted at SeverityWarning by the parser for identifiers
//     exceeding the 32-char recommendation (not the 64-char hard limit). Many vendor MIBs
//     use long descriptive names, so this is noise in permissive mode.
//   - bad-identifier-case: vendor MIBs frequently violate case conventions
func PermissiveConfig() DiagnosticConfig {
	return DiagnosticConfig{
		Level:  StrictnessPermissive,
		FailAt: SeverityFatal,
		Ignore: []string{
			DiagIdentifierUnderscore,
			DiagIdentifierLength32,
			DiagBadIdentifierCase,
		},
	}
}

// ShouldReport returns true if a diagnostic with the given code and severity
// should be reported under this configuration.
//
// The Level controls reporting threshold:
//   - Level 0 (Strict): Report all diagnostics (Info and above)
//   - Level 3 (Normal): Report Minor and above (0-3)
//   - Level 5 (Permissive): Report Warning and above (0-5)
//   - Level 6 (Silent): Report nothing
//
// Lower severity numbers are more severe (Fatal=0, Info=6).
func (c DiagnosticConfig) ShouldReport(code string, sev Severity) bool {
	if slices.ContainsFunc(c.Ignore, func(pattern string) bool {
		return MatchGlob(pattern, code)
	}) {
		return false
	}

	if override, ok := c.Overrides[code]; ok {
		sev = override
	}

	if c.Level >= StrictnessSilent {
		return false
	}

	if c.Level == StrictnessStrict {
		return true
	}

	return int(sev) <= int(c.Level)
}

// ShouldFail returns true if a diagnostic with the given severity should
// cause loading to fail.
func (c DiagnosticConfig) ShouldFail(sev Severity) bool {
	return sev <= c.FailAt
}

// IsStrict returns true if strict RFC compliance is required.
// In strict mode, no fallback resolution strategies are used.
func (c DiagnosticConfig) IsStrict() bool {
	return c.Level < StrictnessNormal
}

// AllowSafeFallbacks returns true if safe fallback strategies should be used.
// Safe fallbacks have high confidence of matching MIB author intent.
// Enabled at normal strictness (level 3) and above.
func (c DiagnosticConfig) AllowSafeFallbacks() bool {
	return c.Level >= StrictnessNormal
}

// AllowBestGuessFallbacks returns true if best-guess fallback strategies should be used.
// Best-guess fallbacks may resolve incorrectly but help with broken vendor MIBs.
// Enabled at permissive strictness (level 5) and above.
func (c DiagnosticConfig) AllowBestGuessFallbacks() bool {
	return c.Level >= StrictnessPermissive
}

// MatchGlob performs simple glob matching with * wildcard.
func MatchGlob(pattern, s string) bool {
	if prefix, ok := strings.CutSuffix(pattern, "*"); ok {
		return strings.HasPrefix(s, prefix)
	}
	if suffix, ok := strings.CutPrefix(pattern, "*"); ok {
		return strings.HasSuffix(s, suffix)
	}
	return pattern == s
}
