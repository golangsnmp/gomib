package cst

import (
	"testing"

	"github.com/golangsnmp/gomib/internal/lexer"
	"github.com/golangsnmp/gomib/internal/types"
)

func TestReconstructEmptyModule(t *testing.T) {
	// Source: TEST-MIB DEFINITIONS ::= BEGIN\nEND\n
	source := []byte("TEST-MIB DEFINITIONS ::= BEGIN\nEND\n")

	// Build tokens with trivia for whitespace between them.
	// Offsets:
	//   0..8   "TEST-MIB"
	//   8..9   " "
	//   9..20  "DEFINITIONS"
	//  20..21  " "
	//  21..24  "::="
	//  24..25  " "
	//  25..30  "BEGIN"
	//  30..31  "\n"
	//  31..34  "END"
	//  34..35  "\n"

	sp := func(start, end int) types.Span {
		return types.Span{Start: types.ByteOffset(start), End: types.ByteOffset(end)}
	}
	mkTrivia := func(start, end int) Trivia {
		return Trivia{Kind: TriviaWhitespace, Span: sp(start, end)}
	}

	name := SyntaxToken{
		Kind: lexer.TokUppercaseIdent,
		Span: sp(0, 8),
	}
	definitions := SyntaxToken{
		Kind:          lexer.TokKwDefinitions,
		Span:          sp(9, 20),
		LeadingTrivia: []Trivia{mkTrivia(8, 9)},
	}
	assign := SyntaxToken{
		Kind:          lexer.TokColonColonEqual,
		Span:          sp(21, 24),
		LeadingTrivia: []Trivia{mkTrivia(20, 21)},
	}
	begin := SyntaxToken{
		Kind:          lexer.TokKwBegin,
		Span:          sp(25, 30),
		LeadingTrivia: []Trivia{mkTrivia(24, 25)},
	}
	end := SyntaxToken{
		Kind:          lexer.TokKwEnd,
		Span:          sp(31, 34),
		LeadingTrivia: []Trivia{mkTrivia(30, 31)},
	}
	eof := SyntaxToken{
		Kind:          lexer.TokEOF,
		Span:          sp(35, 35),
		LeadingTrivia: []Trivia{mkTrivia(34, 35)},
	}

	file := &ModuleFile{
		Modules: []ModuleNode{
			{
				Name:        name,
				Definitions: definitions,
				Assign:      assign,
				Begin:       begin,
				End:         end,
				Span:        sp(0, 34),
			},
		},
		EOF:  eof,
		Span: sp(0, 35),
	}

	got := ReconstructText(file, source)
	if string(got) != string(source) {
		t.Errorf("round-trip mismatch\ngot:  %q\nwant: %q", got, source)
	}
}

func TestWalkTokensCount(t *testing.T) {
	// Verify WalkTokens visits the expected number of tokens for a simple module.
	sp := func(start, end int) types.Span {
		return types.Span{Start: types.ByteOffset(start), End: types.ByteOffset(end)}
	}
	file := &ModuleFile{
		Modules: []ModuleNode{
			{
				Name:        SyntaxToken{Kind: lexer.TokUppercaseIdent, Span: sp(0, 4)},
				Definitions: SyntaxToken{Kind: lexer.TokKwDefinitions, Span: sp(5, 16)},
				Assign:      SyntaxToken{Kind: lexer.TokColonColonEqual, Span: sp(17, 20)},
				Begin:       SyntaxToken{Kind: lexer.TokKwBegin, Span: sp(21, 26)},
				End:         SyntaxToken{Kind: lexer.TokKwEnd, Span: sp(27, 30)},
			},
		},
		EOF: SyntaxToken{Kind: lexer.TokEOF, Span: sp(31, 31)},
	}

	var count int
	file.WalkTokens(func(_ SyntaxToken) { count++ })
	// 5 tokens for the module + 1 EOF = 6
	if count != 6 {
		t.Errorf("expected 6 tokens, got %d", count)
	}
}
