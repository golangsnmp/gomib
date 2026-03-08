package mib

import "github.com/golangsnmp/gomib/internal/types"

// Diagnostic represents an issue found during parsing or resolution.
type Diagnostic = types.Diagnostic

// DiagnosticConfig controls diagnostic reporting and failure policy.
type DiagnosticConfig = types.DiagnosticConfig

// ConfigForReporting returns the diagnostic configuration preset for the given
// reporting level. Unknown levels fall back to DefaultConfig.
func ConfigForReporting(level ReportingLevel) DiagnosticConfig {
	return types.ConfigForReporting(level)
}

// DefaultConfig returns the default diagnostic configuration.
func DefaultConfig() DiagnosticConfig { return types.DefaultConfig() }

// VerboseConfig returns a verbose reporting profile.
func VerboseConfig() DiagnosticConfig { return types.VerboseConfig() }

// QuietConfig returns a low-noise reporting profile.
func QuietConfig() DiagnosticConfig { return types.QuietConfig() }

// SilentConfig returns a silent configuration that suppresses all diagnostics.
func SilentConfig() DiagnosticConfig { return types.SilentConfig() }
