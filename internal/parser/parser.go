// Package parser provides MIB parsing into module IR types.
//
// The parser supports configurable strictness via DiagnosticConfig:
//   - Strict mode: Emits diagnostics for RFC violations (underscores, long identifiers, etc.)
//   - Normal mode: Emits diagnostics for significant issues, warns on RFC violations
//   - Permissive mode: Accepts most vendor MIBs, minimal diagnostics
//
// Regardless of strictness level, the parser attempts to recover from errors and
// continue parsing. Parse errors are collected as diagnostics rather than causing
// immediate failure.
package parser

import (
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	"github.com/golangsnmp/gomib/internal/lexer"
	"github.com/golangsnmp/gomib/internal/types"
)

// Parser converts a token stream into module IR types with diagnostics.
type Parser struct {
	source     []byte
	lex        *lexer.Lexer
	buf        [3]lexer.Token   // lookahead buffer: buf[0]=current, buf[1]=peek(1), buf[2]=peek(2)
	lastEnd    types.ByteOffset // end position of last consumed token
	eofToken   lexer.Token
	moduleName string         // set after parsing header, for IsBaseModule checks
	language   types.Language // detected during import parsing
	types.SpanDiagnosticCollector
	types.Logger
}

// New returns a Parser that lexes the source and prepares for parsing.
// Pass nil for logger to disable logging. The diagConfig controls which
// RFC violations are reported.
func New(source []byte, logger *slog.Logger, diagConfig types.DiagnosticConfig) *Parser {
	var lexLogger *slog.Logger
	if logger != nil {
		lexLogger = logger.With(slog.String("component", "lexer"))
	}
	lex := lexer.New(source, lexLogger, diagConfig)
	eofSpan := types.NewSpan(types.ByteOffset(len(source)), types.ByteOffset(len(source)))
	eofToken := lexer.Token{Kind: lexer.TokEOF, Span: eofSpan}
	p := &Parser{
		source:                  source,
		lex:                     lex,
		eofToken:                eofToken,
		language:                types.LanguageUnknown,
		SpanDiagnosticCollector: types.NewSpanDiagnosticCollector(diagConfig),
		Logger:                  types.Logger{L: logger},
	}
	p.buf[0] = p.nextNonComment()
	p.buf[1] = p.nextNonComment()
	p.buf[2] = p.nextNonComment()
	p.Log(slog.LevelDebug, "parser initialized")
	return p
}

// nextNonComment returns the next token from the lexer, skipping comment tokens.
func (p *Parser) nextNonComment() lexer.Token {
	for {
		tok := p.lex.NextToken()
		if tok.Kind != lexer.TokComment {
			return tok
		}
	}
}

// validateIdentifier checks for RFC 2578 identifier violations
// (underscores, trailing hyphens, length limits).
func (p *Parser) validateIdentifier(name string, span types.Span) {
	if strings.Contains(name, "_") {
		p.EmitDiagnostic(types.DiagIdentifierUnderscore, span,
			fmt.Sprintf("identifier %q contains underscore (RFC violation)", name))
	}

	if strings.HasSuffix(name, "-") {
		p.EmitDiagnostic(types.DiagIdentifierHyphenEnd, span,
			fmt.Sprintf("identifier %q ends with hyphen", name))
	}

	if len(name) > 64 {
		p.EmitDiagnostic(types.DiagIdentifierLength64, span,
			fmt.Sprintf("identifier %q exceeds 64 character limit (%d chars)", name, len(name)))
	} else if len(name) > 32 {
		p.EmitDiagnostic(types.DiagIdentifierLength32, span,
			fmt.Sprintf("identifier %q exceeds 32 character recommendation (%d chars)", name, len(name)))
	}
}

// validateValueReference checks that a value reference starts with lowercase.
// Per RFC 2578, value references (used in OID assignments) should start with lowercase.
func (p *Parser) validateValueReference(name string, span types.Span) {
	if name != "" && name[0] >= 'A' && name[0] <= 'Z' {
		p.EmitDiagnostic(types.DiagBadIdentifierCase, span,
			fmt.Sprintf("%q should start with a lowercase letter", name))
	}
}

func (p *Parser) isEOF() bool {
	return p.peek().Kind == lexer.TokEOF
}

func (p *Parser) peek() lexer.Token {
	return p.buf[0]
}

func (p *Parser) peekNth(n int) lexer.Token {
	if n < len(p.buf) {
		return p.buf[n]
	}
	return p.eofToken
}

func (p *Parser) advance() lexer.Token {
	tok := p.buf[0]
	p.buf[0] = p.buf[1]
	p.buf[1] = p.buf[2]
	p.buf[2] = p.nextNonComment()
	p.lastEnd = tok.Span.End
	return tok
}

func (p *Parser) check(kind lexer.TokenKind) bool {
	return p.peek().Kind == kind
}

// Naming convention for parser methods:
//   - expect*: consume a single token matching a condition, return lexer.Token
//   - parse*:  consume tokens and build a typed IR fragment
//   - skip*:   discard tokens without producing a value
//   - convert*: transform already-consumed span text into a Go value
func (p *Parser) expect(kind lexer.TokenKind) (lexer.Token, *types.SpanDiagnostic) {
	if p.check(kind) {
		return p.advance(), nil
	}
	diag := p.makeError("expected " + kind.LibsmiName())
	return lexer.Token{}, &diag
}

func (p *Parser) currentSpan() types.Span {
	return p.peek().Span
}

func (p *Parser) text(span types.Span) string {
	return string(p.source[span.Start:span.End])
}

func (p *Parser) makeIdentStr(token lexer.Token) (string, types.Span) {
	return p.text(token.Span), token.Span
}

// makeIdentStrWithValidation creates an identifier string and checks for RFC violations.
// Use for definition names, not type references.
func (p *Parser) makeIdentStrWithValidation(token lexer.Token) (string, types.Span) {
	name := p.text(token.Span)
	p.validateIdentifier(name, token.Span)
	return name, token.Span
}

// recordParseError appends a structural parse error unconditionally.
// Parse errors bypass ShouldReport() filtering because they indicate
// a syntax problem that must be reported at any strictness level.
func (p *Parser) recordParseError(diag types.SpanDiagnostic) {
	p.RecordDiagnostic(diag)
}

func (p *Parser) makeError(message string) types.SpanDiagnostic {
	return types.SpanDiagnostic{
		Severity: types.SeverityForCode(types.DiagParseError),
		Code:     types.DiagParseError,
		Span:     p.currentSpan(),
		Message:  message,
	}
}

func (p *Parser) convertU32(span types.Span, context string) uint32 {
	text := p.text(span)
	v, err := strconv.ParseUint(text, 10, 32)
	if err != nil {
		p.EmitDiagnostic(types.DiagInvalidU32, span,
			fmt.Sprintf("invalid %s (not a valid u32)", context))
		return 0
	}
	return uint32(v)
}

func (p *Parser) convertI64(span types.Span, context string) int64 {
	text := p.text(span)
	v, err := strconv.ParseInt(text, 10, 64)
	if err != nil {
		p.EmitDiagnostic(types.DiagInvalidI64, span,
			fmt.Sprintf("invalid %s (not a valid integer)", context))
		return 0
	}
	return v
}
