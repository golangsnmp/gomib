package cst

import (
	"github.com/golangsnmp/gomib/internal/lexer"
	"github.com/golangsnmp/gomib/internal/types"
)

// TriviaKind classifies a trivia element (whitespace or comment).
type TriviaKind int

const (
	TriviaWhitespace TriviaKind = iota
	TriviaComment
)

// Trivia is a whitespace or comment fragment attached to a token.
type Trivia struct {
	Kind TriviaKind
	Span types.Span
}

// SyntaxToken is a meaningful token with attached leading and trailing trivia.
// Leading trivia is whitespace/comments before the token. Trailing trivia is
// a same-line comment immediately after the token (no intervening newline).
type SyntaxToken struct {
	Kind           lexer.TokenKind
	Span           types.Span
	LeadingTrivia  []Trivia
	TrailingTrivia []Trivia
}

// IsZero reports whether this token is absent (zero value).
func (t SyntaxToken) IsZero() bool {
	return t.Span.Start == 0 && t.Span.End == 0 && t.Kind == 0
}
