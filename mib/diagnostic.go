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
func PermissiveConfig() DiagnosticConfig {
	return DiagnosticConfig{
		Level:  StrictnessPermissive,
		FailAt: SeverityFatal,
	}
}

// ShouldReport returns true if a diagnostic with the given code and severity
// should be reported under this configuration.
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

	// Report if severity <= level (lower = more severe)
	return int(sev) <= int(c.Level)
}

// ShouldFail returns true if a diagnostic with the given severity should
// cause loading to fail.
func (c DiagnosticConfig) ShouldFail(sev Severity) bool {
	return sev <= c.FailAt
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
