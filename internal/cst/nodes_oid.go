package cst

import "github.com/golangsnmp/gomib/internal/types"

// OidValueNode represents a braced OID value: { component component ... }
type OidValueNode struct {
	LBrace     SyntaxToken
	Components []OidComponentNode
	RBrace     SyntaxToken
	Span       types.Span
}

// OidComponentNode is a single OID component. Forms:
//   - number
//   - name
//   - name(number)
//   - Module.name
//   - Module.name(number)
//
// Absent parts use zero-value SyntaxToken (detected via IsZero).
type OidComponentNode struct {
	Module SyntaxToken // module qualifier (zero if unqualified)
	Dot    SyntaxToken // . separator (zero if unqualified)
	Name   SyntaxToken // name part (zero if pure number)
	LParen SyntaxToken // ( before number (zero if no number)
	Number SyntaxToken // numeric value (zero if pure name)
	RParen SyntaxToken // ) after number (zero if no number)
	Span   types.Span
}
