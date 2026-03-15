// Package token provides public access to gomib's lexer token types and a
// simple tokenization entry point.
//
// # Overview
//
// Use [Tokenize] when you need lexical access to MIB source without running
// the full parser and resolver. Typical uses include syntax highlighting,
// editor tooling, lightweight inspection, and custom linting.
//
// # Classification Helpers
//
// [TokenKind] includes classification helpers such as [TokenKind.IsKeyword],
// [TokenKind.IsMacroKeyword], [TokenKind.IsClauseKeyword], and
// [TokenKind.IsIdentifier] so callers can group tokens without reimplementing
// SMI keyword tables.
package token
