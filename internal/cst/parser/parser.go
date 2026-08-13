// Package parser provides a lossless CST parser for SMI MIB modules.
//
// The parser consumes the full token stream (including comments) from
// lexer.Tokenize and produces a *cst.ModuleFile where every byte of
// the original source is accounted for as either a SyntaxToken span
// or attached trivia (whitespace/comments).
//
// The round-trip invariant: cst.ReconstructText(file, source) reproduces
// the original source bytes exactly.
package parser

import (
	"bytes"
	"fmt"
	"log/slog"
	"strings"

	"github.com/golangsnmp/gomib/internal/cst"
	"github.com/golangsnmp/gomib/internal/lexer"
	"github.com/golangsnmp/gomib/internal/types"
)

// Parser converts a token stream into a lossless concrete syntax tree.
type Parser struct {
	source  []byte
	tokens  []lexer.Token // full stream including comments and EOF
	pos     int           // current position in tokens
	lastEnd types.ByteOffset
	types.SpanDiagnosticCollector
	types.Logger
}

// New returns a Parser that tokenizes the source and prepares for parsing.
func New(source []byte, logger *slog.Logger, diagConfig types.DiagnosticConfig) *Parser {
	var lexLogger *slog.Logger
	if logger != nil {
		lexLogger = logger.With(slog.String("component", "lexer"))
	}
	lex := lexer.New(source, lexLogger, diagConfig)
	tokens, lexDiags := lex.Tokenize()

	p := &Parser{
		source:                  source,
		tokens:                  tokens,
		SpanDiagnosticCollector: types.NewSpanDiagnosticCollector(diagConfig),
		Logger:                  types.Logger{L: logger},
	}
	// Record lexer diagnostics into our collector.
	for _, d := range lexDiags {
		p.RecordDiagnostic(d)
	}
	return p
}

// peek returns the current meaningful (non-comment) token.
func (p *Parser) peek() lexer.Token {
	return p.peekNth(0)
}

// peekNth returns the n-th meaningful (non-comment) token ahead.
func (p *Parser) peekNth(n int) lexer.Token {
	count := 0
	for i := p.pos; i < len(p.tokens); i++ {
		if p.tokens[i].Kind != lexer.TokComment {
			if count == n {
				return p.tokens[i]
			}
			count++
		}
	}
	return p.tokens[len(p.tokens)-1] // EOF
}

func (p *Parser) isEOF() bool {
	return p.peek().Kind == lexer.TokEOF
}

func (p *Parser) check(kind lexer.TokenKind) bool {
	return p.peek().Kind == kind
}

func (p *Parser) currentSpan() types.Span {
	return p.peek().Span
}

func (p *Parser) text(span types.Span) string {
	return string(p.source[span.Start:span.End])
}

// Trivia is a re-export for convenience within this package.
type Trivia = cst.Trivia

// findNextMeaningful returns the index of the next non-comment token at or
// after p.pos.
func (p *Parser) findNextMeaningful() int {
	for i := p.pos; i < len(p.tokens); i++ {
		if p.tokens[i].Kind != lexer.TokComment {
			return i
		}
	}
	return len(p.tokens) - 1
}

// collectTriviaUpTo builds leading trivia from lastEnd to tokens[targetIdx],
// consuming all intervening comment tokens. It also detects a same-line
// trailing comment after the target token.
func (p *Parser) collectTriviaUpTo(targetIdx int) (leading, trailing []Trivia) {
	// Collect leading trivia: whitespace gaps and comments before the target.
	for i := p.pos; i < targetIdx; i++ {
		tok := p.tokens[i]
		if tok.Kind == lexer.TokComment {
			// Whitespace gap before the comment
			if tok.Span.Start > p.lastEnd {
				leading = append(leading, cst.Trivia{
					Kind: cst.TriviaWhitespace,
					Span: types.NewSpan(p.lastEnd, tok.Span.Start),
				})
			}
			leading = append(leading, cst.Trivia{
				Kind: cst.TriviaComment,
				Span: tok.Span,
			})
			p.lastEnd = tok.Span.End
			// After a comment, the lexer may have consumed the newline that follows
			// but it's NOT part of the comment span. The gap between the comment end
			// and the next token start will be captured as whitespace.
		}
	}

	// Whitespace gap between the last thing and the target token
	targetTok := p.tokens[targetIdx]
	if targetTok.Span.Start > p.lastEnd {
		leading = append(leading, cst.Trivia{
			Kind: cst.TriviaWhitespace,
			Span: types.NewSpan(p.lastEnd, targetTok.Span.Start),
		})
	}

	// Now check for trailing comment: a comment on the same line after the target.
	// We peek at the raw token right after the target. If it's a comment and
	// there's no newline between the target's end and the comment's start,
	// attach it as trailing trivia.
	nextRawIdx := targetIdx + 1
	if nextRawIdx < len(p.tokens) {
		nextTok := p.tokens[nextRawIdx]
		if nextTok.Kind == lexer.TokComment {
			gapStart := targetTok.Span.End
			gapEnd := nextTok.Span.Start
			gap := p.source[gapStart:gapEnd]
			if !bytes.ContainsAny(gap, "\n\r") {
				// Same-line comment - attach as trailing trivia.
				if gapEnd > gapStart {
					trailing = append(trailing, cst.Trivia{
						Kind: cst.TriviaWhitespace,
						Span: types.NewSpan(gapStart, gapEnd),
					})
				}
				trailing = append(trailing, cst.Trivia{
					Kind: cst.TriviaComment,
					Span: nextTok.Span,
				})
				// Update pos to skip past the consumed trailing comment.
				p.pos = nextRawIdx + 1
				p.lastEnd = nextTok.Span.End
				return leading, trailing
			}
		}
	}

	return leading, trailing
}

// advance consumes the current meaningful token and returns it as a
// SyntaxToken with trivia attached.
func (p *Parser) advance() cst.SyntaxToken {
	targetIdx := p.findNextMeaningful()
	leading, trailing := p.collectTriviaUpTo(targetIdx)

	tok := p.tokens[targetIdx]
	st := cst.SyntaxToken{
		Kind:           tok.Kind,
		Span:           tok.Span,
		LeadingTrivia:  leading,
		TrailingTrivia: trailing,
	}

	// If collectTriviaUpTo already advanced pos past a trailing comment,
	// don't re-advance.
	if p.pos <= targetIdx {
		p.pos = targetIdx + 1
	}
	if p.lastEnd < tok.Span.End {
		p.lastEnd = tok.Span.End
	}

	return st
}

// expect consumes a token of the given kind, or records a parse error.
func (p *Parser) expect(kind lexer.TokenKind) (cst.SyntaxToken, *types.SpanDiagnostic) {
	if p.check(kind) {
		return p.advance(), nil
	}
	diag := p.makeError("expected " + kind.LibsmiName())
	return cst.SyntaxToken{}, &diag
}

// expectIdentifier consumes an identifier token (uppercase or lowercase),
// accepting forbidden keywords with a diagnostic.
func (p *Parser) expectIdentifier() (cst.SyntaxToken, *types.SpanDiagnostic) {
	kind := p.peek().Kind
	if kind == lexer.TokUppercaseIdent || kind == lexer.TokLowercaseIdent {
		return p.advance(), nil
	}
	if kind == lexer.TokForbiddenKeyword {
		tok := p.advance()
		name := p.text(tok.Span)
		p.EmitDiagnostic(types.DiagKeywordReserved, tok.Span,
			fmt.Sprintf("identifier %q is a reserved ASN.1 keyword", name))
		return tok, nil
	}
	diag := p.makeError("expected identifier")
	return cst.SyntaxToken{}, &diag
}

// expectIndexObject expects an identifier or bare type keyword. Type keywords
// are accepted because vendor MIBs use them as index objects.
func (p *Parser) expectIndexObject() (cst.SyntaxToken, *types.SpanDiagnostic) {
	kind := p.peek().Kind
	if kind.IsIdentifier() || kind.IsTypeKeyword() {
		return p.advance(), nil
	}
	diag := p.makeError("expected index object")
	return cst.SyntaxToken{}, &diag
}

// expectEnumLabel expects an identifier or keyword usable as an enum label.
func (p *Parser) expectEnumLabel() (cst.SyntaxToken, *types.SpanDiagnostic) {
	kind := p.peek().Kind
	if kind.IsIdentifier() || kind.IsKeyword() {
		return p.advance(), nil
	}
	diag := p.makeError("expected enum label")
	return cst.SyntaxToken{}, &diag
}

func (p *Parser) makeError(message string) types.SpanDiagnostic {
	return types.SpanDiagnostic{
		Severity: p.DiagConfig.EffectiveSeverity(types.DiagParseError),
		Code:     types.DiagParseError,
		Span:     p.currentSpan(),
		Message:  message,
	}
}

func (p *Parser) recordParseError(diag types.SpanDiagnostic) {
	p.RecordDiagnostic(diag)
}

// validateIdentifier checks for RFC 2578 identifier violations.
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
func (p *Parser) validateValueReference(name string, span types.Span) {
	if name != "" && name[0] >= 'A' && name[0] <= 'Z' {
		p.EmitDiagnostic(types.DiagBadIdentifierCase, span,
			fmt.Sprintf("%q should start with a lowercase letter", name))
	}
}

// recoverToDefinition skips tokens until the start of a new definition or END.
func (p *Parser) recoverToDefinition() []cst.SyntaxToken {
	var skipped []cst.SyntaxToken
	for !p.isEOF() && !p.check(lexer.TokKwEnd) {
		current := p.peek().Kind
		next := p.peekNth(1).Kind

		if (current.IsIdentifier() && next.IsMacroKeyword()) ||
			(current.IsIdentifier() && next == lexer.TokColonColonEqual) ||
			(current == lexer.TokUppercaseIdent && next == lexer.TokKwTextualConvention) ||
			(current == lexer.TokUppercaseIdent && next == lexer.TokKwMacro) ||
			(current.IsIdentifier() && next == lexer.TokKwObject &&
				p.peekNth(2).Kind == lexer.TokKwIdentifier) {
			break
		}

		skipped = append(skipped, p.advance())
	}
	return skipped
}

// skipBracedContent skips tokens until the matching closing brace. The opening
// brace must already be consumed.
func (p *Parser) skipBracedContent(consumeClose bool) []cst.SyntaxToken {
	var tokens []cst.SyntaxToken
	depth := 1
	for depth > 0 && !p.isEOF() {
		switch p.peek().Kind {
		case lexer.TokLBrace:
			depth++
			tokens = append(tokens, p.advance())
		case lexer.TokRBrace:
			depth--
			if depth > 0 || consumeClose {
				tokens = append(tokens, p.advance())
			}
		default:
			tokens = append(tokens, p.advance())
		}
	}
	return tokens
}
