package cst

import "github.com/golangsnmp/gomib/internal/types"

// ErrorNode wraps tokens that could not be parsed into a valid production.
type ErrorNode struct {
	Tokens []SyntaxToken
	Span   types.Span
}

func (*ErrorNode) definitionNode()              {}
func (n *ErrorNode) DefinitionSpan() types.Span { return n.Span }
