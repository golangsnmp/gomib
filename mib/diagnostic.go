package mib

import "github.com/golangsnmp/gomib/internal/types"

// Diagnostic represents an issue found during parsing or resolution.
type Diagnostic = types.Diagnostic

// DiagnosticConfig controls strictness and diagnostic filtering.
type DiagnosticConfig = types.DiagnosticConfig

// ConfigForLevel returns the diagnostic configuration preset for the given
// strictness level. Unknown levels fall back to DefaultConfig.
func ConfigForLevel(level StrictnessLevel) DiagnosticConfig { return types.ConfigForLevel(level) }

// DefaultConfig returns the default diagnostic configuration (Normal strictness).
func DefaultConfig() DiagnosticConfig { return types.DefaultConfig() }

// StrictConfig returns a strict configuration for RFC compliance checking.
func StrictConfig() DiagnosticConfig { return types.StrictConfig() }

// PermissiveConfig returns a permissive configuration for legacy/vendor MIBs.
// Suppresses common vendor MIB violations like underscores in identifiers.
func PermissiveConfig() DiagnosticConfig { return types.PermissiveConfig() }

// SilentConfig returns a silent configuration that suppresses all diagnostics.
func SilentConfig() DiagnosticConfig { return types.SilentConfig() }
