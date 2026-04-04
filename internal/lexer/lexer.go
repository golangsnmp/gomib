package lexer

import (
	"bytes"
	"fmt"
	"log/slog"

	"github.com/golangsnmp/gomib/internal/types"
)

type lexerState int

const (
	stateNormal lexerState = iota
	stateInMacro
	stateInExports
	stateInComment
)

// punctuation maps single-character punctuation to their token kinds.
var punctuation = map[byte]TokenKind{
	'[': TokLBracket,
	']': TokRBracket,
	'{': TokLBrace,
	'}': TokRBrace,
	'(': TokLParen,
	')': TokRParen,
	';': TokSemicolon,
	',': TokComma,
	'|': TokPipe,
}

// Lexer tokenizes SMIv1/SMIv2 MIB source text.
type Lexer struct {
	source       []byte
	pos          int
	state        lexerState
	commentStart int // saved position of '--' when entering comment state
	types.SpanDiagnosticCollector
	types.Logger
}

// New returns a Lexer that tokenizes the given source bytes.
func New(source []byte, logger *slog.Logger, diagConfig types.DiagnosticConfig) *Lexer {
	l := &Lexer{
		source:                  source,
		state:                   stateNormal, // stateNormal == 0, explicit for clarity
		SpanDiagnosticCollector: types.NewSpanDiagnosticCollector(diagConfig),
		Logger:                  types.Logger{L: logger},
	}
	l.Log(slog.LevelDebug, "lexer initialized", slog.Int("bytes", len(source)))
	return l
}

func (l *Lexer) traceToken(tok Token) {
	if l.TraceEnabled() {
		l.Trace("token",
			slog.Int("kind", int(tok.Kind)),
			slog.Int("start", int(tok.Span.Start)),
			slog.Int("end", int(tok.Span.End)))
	}
}

// Tokenize consumes all source text and returns the token stream
// along with any diagnostics generated during lexing.
func (l *Lexer) Tokenize() ([]Token, []types.SpanDiagnostic) {
	estimatedTokens := max(len(l.source)/6, 64)
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
		slog.Int("diagnostics", l.DiagnosticCount()))
	return tokens, l.Diagnostics()
}

// NextToken advances the lexer and returns the next token.
// Returns TokEOF when all input is consumed.
func (l *Lexer) NextToken() Token {
	for {
		switch l.state {
		case stateInComment:
			return l.emitComment()
		case stateInMacro:
			return l.skipMacroBody()
		case stateInExports:
			return l.skipExportsBody()
		default:
			tok, retry := l.nextNormalToken()
			if retry {
				continue
			}
			return tok
		}
	}
}

func (l *Lexer) isEOF() bool {
	return l.pos >= len(l.source)
}

func (l *Lexer) peek() (byte, bool) {
	if l.pos >= len(l.source) {
		return 0, false
	}
	return l.source[l.pos], true
}

func (l *Lexer) peekAt(offset int) (byte, bool) {
	idx := l.pos + offset
	if idx >= len(l.source) {
		return 0, false
	}
	return l.source[idx], true
}

func (l *Lexer) advance() (byte, bool) {
	if l.pos >= len(l.source) {
		return 0, false
	}
	b := l.source[l.pos]
	l.pos++
	return b, true
}

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

func (l *Lexer) skipLineEnding() {
	b, ok := l.advance()
	if !ok {
		return
	}
	if b == '\r' {
		if next, ok := l.peek(); ok && next == '\n' {
			l.advance()
		}
	}
}

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

func (l *Lexer) spanFrom(start int) types.Span {
	return types.Span{
		Start: types.ByteOffset(start),
		End:   types.ByteOffset(l.pos),
	}
}

func (l *Lexer) token(kind TokenKind, start int) Token {
	tok := Token{
		Kind: kind,
		Span: l.spanFrom(start),
	}
	l.traceToken(tok)
	return tok
}

// nextNormalToken scans the next token in normal state. Returns (token, retry)
// where retry=true means the caller should loop (e.g. after skipping junk or
// entering comment state).
func (l *Lexer) nextNormalToken() (Token, bool) {
	l.skipWhitespace()

	start := l.pos

	b, ok := l.peek()
	if !ok {
		return l.token(TokEOF, start), false
	}

	if l.isCommentStart() {
		l.commentStart = start
		l.advance()
		l.advance()
		l.state = stateInComment
		l.Log(slog.LevelDebug, "entering comment", slog.Int("offset", start))
		return Token{}, true
	}

	if kind, ok := punctuation[b]; ok {
		l.advance()
		return l.token(kind, start), false
	}

	if b == '.' {
		l.advance()
		if next, ok := l.peek(); ok && next == '.' {
			l.advance()
			return l.token(TokDotDot, start), false
		}
		return l.token(TokDot, start), false
	}

	if b == ':' {
		l.advance()
		if next, ok := l.peek(); ok && next == ':' {
			if after, ok := l.peekAt(1); ok && after == '=' {
				l.advance()
				l.advance()
				return l.token(TokColonColonEqual, start), false
			}
		}
		return l.token(TokColon, start), false
	}

	if b == '-' {
		if next, ok := l.peekAt(1); ok && isDigit(next) {
			return l.scanNegativeNumber(), false
		}
		l.advance()
		return l.token(TokMinus, start), false
	}

	if isDigit(b) {
		return l.scanNumber(), false
	}

	if b == '"' {
		return l.scanQuotedString(), false
	}

	if b == '\'' {
		return l.scanHexOrBinString(), false
	}

	if isAlpha(b) {
		return l.scanIdentifierOrKeyword(), false
	}

	l.advance()
	span := l.spanFrom(start)
	l.EmitDiagnostic(types.DiagUnexpectedCharacter, span, fmt.Sprintf("unexpected character: 0x%02x", b))
	l.skipToEOL()
	return Token{}, true
}

func (l *Lexer) tryConsumeTripleDashEOL() bool {
	b1, ok1 := l.peek()
	b2, ok2 := l.peekAt(1)
	b3, ok3 := l.peekAt(2)

	if !ok1 || !ok2 || !ok3 || b1 != '-' || b2 != '-' || b3 != '-' {
		return false
	}

	b4, ok4 := l.peekAt(3)
	if !ok4 || b4 == '\n' || b4 == '\r' {
		l.advance()
		l.advance()
		l.advance()
		l.skipLineEnding()
		return true
	}
	return false
}

// emitComment consumes comment text and returns a TokComment token.
// The span covers from '--' through the comment text, not including
// any trailing newline. Called from the NextToken loop when state is
// stateInComment.
func (l *Lexer) emitComment() Token {
	l.skipCommentBody(true)
	tok := l.token(TokComment, l.commentStart)
	// Consume trailing newline if present (not part of comment span).
	if b, ok := l.peek(); ok && (b == '\n' || b == '\r') {
		l.skipLineEnding()
	}
	l.state = stateNormal
	return tok
}

func (l *Lexer) skipMacroBody() Token {
	for {
		l.skipWhitespace()

		if l.isEOF() {
			start := l.pos
			l.state = stateNormal
			return l.token(TokEOF, start)
		}

		if l.matchesKeyword([]byte("END")) {
			// Check that the preceding character is not alphanumeric or hyphen,
			// so we don't match "END" at the tail of identifiers like "PRETEND".
			prevIsDelimiter := l.pos == 0 ||
				(!isAlphanumeric(l.source[l.pos-1]) && l.source[l.pos-1] != '-')
			if prevIsDelimiter {
				saved := l.pos
				l.pos += 3
				b, ok := l.peek()
				isDelimiter := !ok ||
					(b == '-' && l.peekAtEquals(1, '-')) ||
					(!isAlphanumeric(b) && b != '-')
				if isDelimiter {
					l.state = stateNormal
					return l.token(TokKwEnd, saved)
				}
				// Restore position on failed match so we don't skip bytes.
				l.pos = saved
			}
		}

		if l.isCommentStart() {
			l.skipCommentInline()
			continue
		}

		l.advance()
	}
}

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

func (l *Lexer) skipCommentInline() {
	l.advance() // consume first '-'
	l.advance() // consume second '-'
	l.skipCommentBody(false)
}

// skipCommentBody scans forward past comment text. It stops at EOF, a newline
// (without consuming either), or a "--" terminator (consuming both dashes).
// When handleTripleDash is true, "---" followed by EOL is also treated as a
// terminator (consuming the three dashes and the line ending).
func (l *Lexer) skipCommentBody(handleTripleDash bool) {
	for {
		b, ok := l.peek()
		if !ok || b == '\n' || b == '\r' {
			return
		}
		if b == '-' {
			if handleTripleDash && l.tryConsumeTripleDashEOL() {
				return
			}
			if l.isCommentStart() {
				l.advance()
				l.advance()
				return
			}
		}
		l.advance()
	}
}

func (l *Lexer) matchesKeyword(keyword []byte) bool {
	return bytes.HasPrefix(l.source[l.pos:], keyword)
}

func (l *Lexer) peekAtEquals(offset int, expected byte) bool {
	b, ok := l.peekAt(offset)
	return ok && b == expected
}

// isCommentStart reports whether the current position starts a -- comment.
func (l *Lexer) isCommentStart() bool {
	b, ok := l.peek()
	if !ok || b != '-' {
		return false
	}
	next, ok := l.peekAt(1)
	return ok && next == '-'
}

func (l *Lexer) scanIdentifierOrKeyword() Token {
	start := l.pos
	firstChar, _ := l.advance()
	isUppercase := isUpperAlpha(firstChar)

loop:
	for {
		b, ok := l.peek()
		if !ok {
			break
		}
		switch {
		case isAlphanumeric(b) || b == '_':
			l.advance()
		case b == '-':
			if l.isCommentStart() {
				break loop
			}
			l.advance()
		default:
			break loop
		}
	}

	text := string(l.source[start:l.pos])

	if kind, ok := LookupKeyword(text); ok {
		switch kind {
		case TokKwMacro:
			l.state = stateInMacro
			l.Log(slog.LevelDebug, "entering macro", slog.Int("offset", start))
		case TokKwExports:
			l.state = stateInExports
			l.Log(slog.LevelDebug, "entering exports", slog.Int("offset", start))
		default:
			// no state transition needed for other keywords
		}
		return l.token(kind, start)
	}

	if IsForbiddenKeyword(text) {
		return l.token(TokForbiddenKeyword, start)
	}

	kind := TokLowercaseIdent
	if isUppercase {
		kind = TokUppercaseIdent
	}
	return l.token(kind, start)
}

func (l *Lexer) scanNumber() Token {
	start := l.pos
	l.scanDigits()
	tok := l.token(TokNumber, start)
	// Check for leading zeros: more than one digit starting with '0'
	if l.pos-start > 1 && l.source[start] == '0' {
		l.EmitDiagnostic(types.DiagNumberLeadingZero, l.spanFrom(start),
			"leading zero(s) on a number")
	}
	return tok
}

func (l *Lexer) scanNegativeNumber() Token {
	start := l.pos
	l.advance() // consume -
	digitStart := l.pos
	l.scanDigits()
	tok := l.token(TokNegativeNumber, start)
	// Check for leading zeros in the digit portion
	if l.pos-digitStart > 1 && l.source[digitStart] == '0' {
		l.EmitDiagnostic(types.DiagNumberLeadingZero, l.spanFrom(start),
			"leading zero(s) on a number")
	}
	return tok
}

func (l *Lexer) scanDigits() {
	for {
		b, ok := l.peek()
		if !ok || !isDigit(b) {
			break
		}
		l.advance()
	}
}

func (l *Lexer) scanQuotedString() Token {
	start := l.pos
	l.advance() // consume opening quote

	for {
		b, ok := l.peek()
		if !ok {
			span := l.spanFrom(start)
			l.EmitDiagnostic(types.DiagUnterminatedString, span, "unterminated string literal")
			return l.token(TokQuotedString, start)
		}
		if b == '"' {
			l.advance()
			return l.token(TokQuotedString, start)
		}
		l.advance()
	}
}

func (l *Lexer) scanHexOrBinString() Token {
	start := l.pos
	l.advance() // consume opening quote

	contentLen := 0
	hasNonHex := false
	hasNonBin := false
	for {
		b, ok := l.peek()
		if !ok || b == '\'' {
			break
		}
		if b != ' ' && b != '\t' && b != '\n' && b != '\r' {
			contentLen++
			if !isHexDigit(b) {
				hasNonHex = true
			}
			if b != '0' && b != '1' {
				hasNonBin = true
			}
		}
		l.advance()
	}

	if b, ok := l.peek(); !ok || b != '\'' {
		span := l.spanFrom(start)
		l.EmitDiagnostic(types.DiagUnterminatedHexBinStr, span, "unterminated hex/binary string")
		return l.token(TokError, start)
	}
	l.advance() // consume closing quote

	suffix, ok := l.peek()
	if !ok {
		span := l.spanFrom(start)
		l.EmitDiagnostic(types.DiagMissingHexBinSuffix, span, "expected 'H' or 'B' suffix for hex/binary string")
		return l.token(TokError, start)
	}

	var kind TokenKind
	switch suffix {
	case 'H', 'h':
		l.advance()
		kind = TokHexString
		if hasNonHex && contentLen > 0 {
			span := l.spanFrom(start)
			l.EmitDiagnostic(types.DiagHexStringInvalidChar, span,
				"hex string contains non-hexadecimal characters")
		}
		if contentLen > 0 && contentLen%2 != 0 {
			span := l.spanFrom(start)
			l.EmitDiagnostic(types.DiagHexStringMul2, span,
				fmt.Sprintf("hex string length %d is not a multiple of 2", contentLen))
		}

	case 'B', 'b':
		l.advance()
		kind = TokBinString
		if hasNonBin && contentLen > 0 {
			span := l.spanFrom(start)
			l.EmitDiagnostic(types.DiagBinStringInvalidChar, span,
				"binary string contains non-binary characters")
		}
		if contentLen > 0 && contentLen%8 != 0 {
			span := l.spanFrom(start)
			l.EmitDiagnostic(types.DiagBinStringMul8, span,
				fmt.Sprintf("binary string length %d is not a multiple of 8", contentLen))
		}

	default:
		l.advance() // consume bad suffix so span includes it
		span := l.spanFrom(start)
		l.EmitDiagnostic(types.DiagMissingHexBinSuffix, span, "expected 'H' or 'B' suffix for hex/binary string")
		kind = TokError
	}

	return l.token(kind, start)
}

func isHexDigit(b byte) bool {
	return (b >= '0' && b <= '9') || (b >= 'a' && b <= 'f') || (b >= 'A' && b <= 'F')
}

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
