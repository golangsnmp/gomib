// Package module provides a normalized representation of MIB modules.
//
// This package transforms AST structures into a simplified module representation
// independent of whether the source was SMIv1 or SMIv2. Key transformations:
//
//   - Language detection from imports
//   - Status normalization (mandatory→Current)
//   - Access normalization (ACCESS→MAX-ACCESS representation)
//   - Unified notification type (TRAP-TYPE and NOTIFICATION-TYPE)
//
// # Pipeline Position
//
//	Source → Lexer → Tokens → Parser → AST → [Lowering] → Module → [Resolver] → Model
//	                                         ^^^^^^^^^^^^^
//	                                         This package
//
// # What Lowering Does NOT Do
//
// These are resolver responsibilities:
//   - OID resolution (keeps OID components as symbols)
//   - Type resolution (keeps type references as symbols)
//   - Nodekind inference (requires resolved OID tree)
//   - Import resolution (just normalize; actual lookup is resolver's job)
//   - Built-in type injection
package module

import "github.com/golangsnmp/gomib/internal/types"

// SmiLanguage is an alias to the shared type.
type SmiLanguage = types.SmiLanguage

// Re-export constants for convenience.
const (
	SmiLanguageUnknown = types.SmiLanguageUnknown
	SmiLanguageSMIv1   = types.SmiLanguageSMIv1
	SmiLanguageSMIv2   = types.SmiLanguageSMIv2
	SmiLanguageSPPI    = types.SmiLanguageSPPI
)
