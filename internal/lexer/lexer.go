package lexer

import (
	"bytes"
	"fmt"
	"log/slog"
	"slices"

	"github.com/golangsnmp/gomib/internal/types"
)

// lexerState represents the lexer's current mode.
type lexerState int

const (
	stateNormal lexerState = iota
	stateInMacro
	stateInExports
	stateInComment
)

// Lexer tokenizes SMIv1/SMIv2 MIB source text.
//
// The lexer is lenient by default and collects diagnostics rather than
// failing early. It operates on raw bytes, allowing it to handle non-UTF-8
// input (e.g., MIB files with Latin-1 encoded characters in descriptions).
type Lexer struct {
	source      []byte
	pos         int
	state       lexerState
	diagnostics []types.Diagnostic
	types.Logger
}

// New creates a new lexer for the given source bytes.
// The logger parameter is optional; pass nil to disable logging.
func New(source []byte, logger *slog.Logger) *Lexer {
	l := &Lexer{
		source: source,
		pos:    0,
		state:  stateNormal,
		Logger: types.Logger{L: logger},
	}
	l.Log(slog.LevelDebug, "lexer initialized", slog.Int("source_len", len(source)))
	return l
}

// Diagnostics returns a copy of all collected diagnostics.
// The returned slice is owned by the caller.
func (l *Lexer) Diagnostics() []types.Diagnostic {
	return slices.Clone(l.diagnostics)
}

// traceToken logs a token at trace level with inline guard for zero cost when disabled.
// This is a hot path - uses inline check to avoid function call overhead.
func (l *Lexer) traceToken(tok Token) {
	if l.TraceEnabled() {
		l.Trace("token",
			slog.Int("kind", int(tok.Kind)),
			slog.Int("start", int(tok.Span.Start)),
			slog.Int("end", int(tok.Span.End)))
	}
}

// Tokenize tokenizes the entire source and returns all tokens and diagnostics.
func (l *Lexer) Tokenize() ([]Token, []types.Diagnostic) {
	// Pre-allocate token slice based on source size.
	// Empirically, MIB files average ~6-8 bytes per token.
	estimatedTokens := len(l.source) / 6
	if estimatedTokens < 64 {
		estimatedTokens = 64
	}
	tokens := make([]Token, 0, estimatedTokens)
	for {
		tok := l.NextToken()
		tokens = append(tokens, tok)
		if tok.Kind == TokEOF {
			break
		}
	}
	l.Log(slog.LevelDebug, "tokenization complete",
		slog.Int("tokens", len(tokens)),
		slog.Int("diagnostics", len(l.diagnostics)))
	return tokens, l.diagnostics
}

// NextToken returns the next token.
func (l *Lexer) NextToken() Token {
	switch l.state {
	case stateNormal:
		return l.nextNormalToken()
	case stateInMacro:
		return l.skipMacroBody()
	case stateInExports:
		return l.skipExportsBody()
	case stateInComment:
		return l.skipComment()
	default:
		return l.nextNormalToken()
	}
}

// isEOF returns true if at end of input.
func (l *Lexer) isEOF() bool {
	return l.pos >= len(l.source)
}

// peek returns the current byte without advancing.
func (l *Lexer) peek() (byte, bool) {
	if l.pos >= len(l.source) {
		return 0, false
	}
	return l.source[l.pos], true
}

// peekAt returns the byte at offset from current position.
func (l *Lexer) peekAt(offset int) (byte, bool) {
	idx := l.pos + offset
	if idx >= len(l.source) {
		return 0, false
	}
	return l.source[idx], true
}

// advance moves forward by one byte and returns the current byte.
func (l *Lexer) advance() (byte, bool) {
	if l.pos >= len(l.source) {
		return 0, false
	}
	b := l.source[l.pos]
	l.pos++
	return b, true
}

// skipWhitespace skips space, tab, CR, LF.
func (l *Lexer) skipWhitespace() {
	for {
		b, ok := l.peek()
		if !ok {
			return
		}
		if b == ' ' || b == '\t' || b == '\r' || b == '\n' {
			l.advance()
		} else {
			return
		}
	}
}

// skipLineEnding consumes a line ending sequence (\n, \r, or \r\n).
// Assumes the current byte is \n or \r.
func (l *Lexer) skipLineEnding() {
	b, ok := l.advance()
	if !ok {
		return
	}
	// Only consume following \n if we just consumed \r (CRLF)
	if b == '\r' {
		if next, ok := l.peek(); ok && next == '\n' {
			l.advance()
		}
	}
}

// skipToEOL skips to end of line (for error recovery, matching libsmi behavior).
func (l *Lexer) skipToEOL() {
	for {
		b, ok := l.peek()
		if !ok {
			return
		}
		if b == '\n' || b == '\r' {
			l.skipLineEnding()
			return
		}
		l.advance()
	}
}

// error adds an error diagnostic.
func (l *Lexer) error(span types.Span, message string) {
	l.diagnostics = append(l.diagnostics, types.Diagnostic{
		Severity: types.SeverityError,
		Span:     span,
		Message:  message,
	})
}

// spanFrom creates a span from start to current position.
func (l *Lexer) spanFrom(start int) types.Span {
	return types.Span{
		Start: types.ByteOffset(start),
		End:   types.ByteOffset(l.pos),
	}
}

// token creates a token from start to current position.
func (l *Lexer) token(kind TokenKind, start int) Token {
	tok := Token{
		Kind: kind,
		Span: l.spanFrom(start),
	}
	l.traceToken(tok)
	return tok
}

// nextNormalToken gets the next token in normal state.
func (l *Lexer) nextNormalToken() Token {
	l.skipWhitespace()

	start := l.pos

	// Check for EOF
	b, ok := l.peek()
	if !ok {
		return l.token(TokEOF, start)
	}

	// Check for comment start
	if b == '-' {
		if next, ok := l.peekAt(1); ok && next == '-' {
			l.advance() // first -
			l.advance() // second -
			l.state = stateInComment
			l.Log(slog.LevelDebug, "entering comment", slog.Int("offset", start))
			return l.skipComment()
		}
	}

	// Single-character tokens
	switch b {
	case '[':
		l.advance()
		return l.token(TokLBracket, start)
	case ']':
		l.advance()
		return l.token(TokRBracket, start)
	case '{':
		l.advance()
		return l.token(TokLBrace, start)
	case '}':
		l.advance()
		return l.token(TokRBrace, start)
	case '(':
		l.advance()
		return l.token(TokLParen, start)
	case ')':
		l.advance()
		return l.token(TokRParen, start)
	case ';':
		l.advance()
		return l.token(TokSemicolon, start)
	case ',':
		l.advance()
		return l.token(TokComma, start)
	case '|':
		l.advance()
		return l.token(TokPipe, start)
	}

	// Dot or DotDot
	if b == '.' {
		l.advance()
		if next, ok := l.peek(); ok && next == '.' {
			l.advance()
			return l.token(TokDotDot, start)
		}
		return l.token(TokDot, start)
	}

	// ColonColonEqual (::=) or just colon
	if b == ':' {
		l.advance()
		if next, ok := l.peek(); ok && next == ':' {
			if after, ok := l.peekAt(1); ok && after == '=' {
				l.advance() // second :
				l.advance() // =
				return l.token(TokColonColonEqual, start)
			}
		}
		return l.token(TokColon, start)
	}

	// Minus (could be negative number or standalone)
	if b == '-' {
		if next, ok := l.peekAt(1); ok && isDigit(next) {
			return l.scanNegativeNumber()
		}
		l.advance()
		return l.token(TokMinus, start)
	}

	// Numbers
	if isDigit(b) {
		return l.scanNumber()
	}

	// Quoted string
	if b == '"' {
		return l.scanQuotedString()
	}

	// Hex or binary string
	if b == '\'' {
		return l.scanHexOrBinString()
	}

	// Identifiers and keywords
	if isAlpha(b) {
		return l.scanIdentifierOrKeyword()
	}

	// Unknown character - skip to end of line (matching libsmi behavior)
	l.advance() // consume the bad character for the span
	span := l.spanFrom(start)
	l.error(span, fmt.Sprintf("unexpected character: 0x%02x", b))
	l.skipToEOL()
	return l.NextToken()
}

// tryConsumeTripleDashEOL checks for and consumes ---{eol} pattern.
// Returns true if the pattern was found and consumed.
func (l *Lexer) tryConsumeTripleDashEOL() bool {
	// Must have --- followed by EOL or EOF
	b1, ok1 := l.peek()
	b2, ok2 := l.peekAt(1)
	b3, ok3 := l.peekAt(2)

	if !ok1 || !ok2 || !ok3 || b1 != '-' || b2 != '-' || b3 != '-' {
		return false
	}

	b4, ok4 := l.peekAt(3)
	if !ok4 || b4 == '\n' || b4 == '\r' {
		// Consume the three dashes
		l.advance()
		l.advance()
		l.advance()
		// Consume newline(s) if present
		if b, ok := l.peek(); ok && b == '\r' {
			l.advance()
		}
		if b, ok := l.peek(); ok && b == '\n' {
			l.advance()
		}
		return true
	}
	return false
}

// skipComment skips comment body and returns the next real token.
func (l *Lexer) skipComment() Token {
	// We're already past the initial --
	for {
		b, ok := l.peek()
		if !ok {
			// EOF in comment is fine
			l.state = stateNormal
			return l.NextToken()
		}

		if b == '\n' || b == '\r' {
			// End of line ends comment
			l.skipLineEnding()
			l.state = stateNormal
			return l.NextToken()
		}

		if b == '-' {
			// Check for ---{eol} special case first
			if l.tryConsumeTripleDashEOL() {
				l.state = stateNormal
				return l.NextToken()
			}
			// Check for -- (regular comment terminator)
			if next, ok := l.peekAt(1); ok && next == '-' {
				l.advance()
				l.advance()
				l.state = stateNormal
				return l.NextToken()
			}
			// Single dash in comment, just advance
			l.advance()
			continue
		}

		l.advance()
	}
}

// skipMacroBody skips MACRO body until END keyword.
func (l *Lexer) skipMacroBody() Token {
	for {
		l.skipWhitespace()

		if l.isEOF() {
			start := l.pos
			l.state = stateNormal
			return l.token(TokEOF, start)
		}

		// Check for END keyword
		if l.matchesKeyword([]byte("END")) {
			start := l.pos
			l.pos += 3
			// Verify delimiter follows per libsmi: ([^a-zA-Z0-9-])|--
			b, ok := l.peek()
			isDelimiter := !ok || // EOF is a valid delimiter
				(b == '-' && l.peekAtEquals(1, '-')) || // -- is a valid delimiter
				(!isAlphanumeric(b) && b != '-') // Non-alphanumeric (except hyphen) is a delimiter
			if isDelimiter {
				l.state = stateNormal
				return l.token(TokKwEnd, start)
			}
			// Not actually END keyword, continue
		}

		// Skip comments in macro body
		if b, ok := l.peek(); ok && b == '-' {
			if next, ok := l.peekAt(1); ok && next == '-' {
				l.skipCommentInline()
				continue
			}
		}

		// Skip any other content
		l.advance()
	}
}

// skipExportsBody skips EXPORTS body until semicolon.
func (l *Lexer) skipExportsBody() Token {
	for {
		b, ok := l.peek()
		if !ok {
			start := l.pos
			l.state = stateNormal
			return l.token(TokEOF, start)
		}

		if b == ';' {
			start := l.pos
			l.advance()
			l.state = stateNormal
			return l.token(TokSemicolon, start)
		}

		l.advance()
	}
}

// skipCommentInline skips a comment inline without changing state.
func (l *Lexer) skipCommentInline() {
	// Skip the --
	l.advance()
	l.advance()
	for {
		b, ok := l.peek()
		if !ok || b == '\n' || b == '\r' {
			return
		}
		if b == '-' {
			if next, ok := l.peekAt(1); ok && next == '-' {
				l.advance()
				l.advance()
				return
			}
		}
		l.advance()
	}
}

// matchesKeyword checks if source matches a keyword at current position.
func (l *Lexer) matchesKeyword(keyword []byte) bool {
	return bytes.HasPrefix(l.source[l.pos:], keyword)
}

// peekAtEquals checks if the byte at offset equals the given byte.
func (l *Lexer) peekAtEquals(offset int, expected byte) bool {
	b, ok := l.peekAt(offset)
	return ok && b == expected
}

// scanIdentifierOrKeyword scans an identifier or keyword.
func (l *Lexer) scanIdentifierOrKeyword() Token {
	start := l.pos
	firstChar, _ := l.advance()
	isUppercase := isUpperAlpha(firstChar)

	// Continue scanning identifier characters
	// Pattern: [a-zA-Z0-9_-]* but with restrictions
	// Note: Double hyphen (--) starts a comment, so we must stop before it
	for {
		b, ok := l.peek()
		if !ok {
			break
		}
		if isAlphanumeric(b) || b == '_' {
			l.advance()
		} else if b == '-' {
			// Always include this hyphen
			l.advance()
			// But if next is also a hyphen, stop here
			if next, ok := l.peek(); ok && next == '-' {
				break
			}
		} else {
			break
		}
	}

	text := string(l.source[start:l.pos])

	// Check if it's a keyword
	if kind, ok := LookupKeyword(text); ok {
		// Handle state transitions for skip keywords
		switch kind {
		case TokKwMacro:
			l.state = stateInMacro
			l.Log(slog.LevelDebug, "entering macro", slog.Int("offset", start))
		case TokKwExports:
			l.state = stateInExports
			l.Log(slog.LevelDebug, "entering exports", slog.Int("offset", start))
		}
		return l.token(kind, start)
	}

	// Check if it's a forbidden ASN.1 keyword
	if IsForbiddenKeyword(text) {
		return l.token(TokForbiddenKeyword, start)
	}

	// It's an identifier
	kind := TokLowercaseIdent
	if isUppercase {
		kind = TokUppercaseIdent
	}
	return l.token(kind, start)
}

// scanNumber scans a number literal.
func (l *Lexer) scanNumber() Token {
	start := l.pos

	// Consume all digits
	for {
		b, ok := l.peek()
		if !ok || !isDigit(b) {
			break
		}
		l.advance()
	}

	return l.token(TokNumber, start)
}

// scanNegativeNumber scans a negative number literal.
func (l *Lexer) scanNegativeNumber() Token {
	start := l.pos
	l.advance() // consume -

	// Consume all digits
	for {
		b, ok := l.peek()
		if !ok || !isDigit(b) {
			break
		}
		l.advance()
	}

	return l.token(TokNegativeNumber, start)
}

// scanQuotedString scans a quoted string literal.
func (l *Lexer) scanQuotedString() Token {
	start := l.pos
	l.advance() // consume opening quote

	for {
		b, ok := l.peek()
		if !ok {
			span := l.spanFrom(start)
			l.error(span, "unterminated string literal")
			return l.token(TokQuotedString, start)
		}
		if b == '"' {
			l.advance() // consume closing quote
			return l.token(TokQuotedString, start)
		}
		// Non-ASCII in strings is silently accepted
		l.advance()
	}
}

// scanHexOrBinString scans a hex or binary string literal.
func (l *Lexer) scanHexOrBinString() Token {
	start := l.pos
	l.advance() // consume opening quote

	// Skip content until closing quote
	for {
		b, ok := l.peek()
		if !ok || b == '\'' {
			break
		}
		l.advance()
	}

	// Expect closing quote
	if b, ok := l.peek(); !ok || b != '\'' {
		span := l.spanFrom(start)
		l.error(span, "unterminated hex/binary string")
		return l.token(TokError, start)
	}
	l.advance() // consume closing quote

	// Expect H/h or B/b suffix
	suffix, ok := l.peek()
	if !ok {
		span := l.spanFrom(start)
		l.error(span, "expected 'H' or 'B' suffix for hex/binary string")
		return l.token(TokError, start)
	}

	var kind TokenKind
	switch suffix {
	case 'H', 'h':
		l.advance()
		kind = TokHexString

	case 'B', 'b':
		l.advance()
		kind = TokBinString

	default:
		span := l.spanFrom(start)
		l.error(span, "expected 'H' or 'B' suffix for hex/binary string")
		kind = TokError
	}

	return l.token(kind, start)
}

// Helper functions

func isDigit(b byte) bool {
	return b >= '0' && b <= '9'
}

func isAlpha(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z')
}

func isUpperAlpha(b byte) bool {
	return b >= 'A' && b <= 'Z'
}

func isAlphanumeric(b byte) bool {
	return isAlpha(b) || isDigit(b)
}
