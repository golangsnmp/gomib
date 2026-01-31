package mib

// Diagnostic represents an issue found during parsing or resolution.
type Diagnostic struct {
	Severity Severity
	Code     string // e.g., "identifier-underscore", "import-not-found"
	Message  string
	Module   string // source module name
	Line     int    // 1-based line number, 0 if not applicable
	Column   int    // 1-based column, 0 if not applicable
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
func PermissiveConfig() DiagnosticConfig {
	return DiagnosticConfig{
		Level:  StrictnessPermissive,
		FailAt: SeverityFatal,
		// Suppress diagnostics that are allowed in permissive mode
		Ignore: []string{
			"identifier-underscore",
			"identifier-length-32",
			"bad-identifier-case",
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
	// Check ignore list
	for _, pattern := range c.Ignore {
		if matchGlob(pattern, code) {
			return false
		}
	}

	// Check overrides
	if override, ok := c.Overrides[code]; ok {
		sev = override
	}

	// Silent mode suppresses all reporting
	if c.Level >= StrictnessSilent {
		return false
	}

	// Strict mode reports all diagnostics
	if c.Level == StrictnessStrict {
		return true
	}

	// Normal/Permissive: Report if severity is at or below the threshold
	// Level 3 (Normal): report sev 0-3 (Fatal, Severe, Error, Minor)
	// Level 5 (Permissive): report sev 0-5 (all except Info)
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
	return c.Level <= 2
}

// AllowSafeFallbacks returns true if safe fallback strategies should be used.
// Safe fallbacks have high confidence of matching MIB author intent.
// Enabled at normal strictness (level 3) and above.
func (c DiagnosticConfig) AllowSafeFallbacks() bool {
	return c.Level >= 3
}

// AllowBestGuessFallbacks returns true if best-guess fallback strategies should be used.
// Best-guess fallbacks may resolve incorrectly but help with broken vendor MIBs.
// Enabled at permissive strictness (level 5) and above.
func (c DiagnosticConfig) AllowBestGuessFallbacks() bool {
	return c.Level >= 5
}

// matchGlob performs simple glob matching with * wildcard.
func matchGlob(pattern, s string) bool {
	if pattern == "*" {
		return true
	}

	// Handle trailing wildcard
	if len(pattern) > 0 && pattern[len(pattern)-1] == '*' {
		prefix := pattern[:len(pattern)-1]
		return len(s) >= len(prefix) && s[:len(prefix)] == prefix
	}

	// Handle leading wildcard
	if len(pattern) > 0 && pattern[0] == '*' {
		suffix := pattern[1:]
		return len(s) >= len(suffix) && s[len(s)-len(suffix):] == suffix
	}

	return pattern == s
}
