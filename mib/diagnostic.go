package mib

import "github.com/golangsnmp/gomib/internal/types"

// Diagnostic represents an issue found during parsing or resolution.
type Diagnostic = types.Diagnostic

// DiagnosticConfig controls strictness and diagnostic filtering.
type DiagnosticConfig = types.DiagnosticConfig

// DefaultConfig returns the default diagnostic configuration (Normal strictness).
func DefaultConfig() DiagnosticConfig { return types.DefaultConfig() }

// StrictConfig returns a strict configuration for RFC compliance checking.
func StrictConfig() DiagnosticConfig { return types.StrictConfig() }

// PermissiveConfig returns a permissive configuration for legacy/vendor MIBs.
// Suppresses common vendor MIB violations like underscores in identifiers.
func PermissiveConfig() DiagnosticConfig { return types.PermissiveConfig() }
