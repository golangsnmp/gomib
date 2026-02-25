package lexer

import (
	"testing"

	"github.com/golangsnmp/gomib/internal/testutil"
)

func FuzzTokenize(f *testing.F) {
	f.Add([]byte(""))
	f.Add([]byte("EXAMPLE-MIB DEFINITIONS ::= BEGIN\nEND"))
	f.Add([]byte("IMPORTS MODULE-IDENTITY, OBJECT-TYPE FROM SNMPv2-SMI;"))
	f.Add([]byte("testObj OBJECT-TYPE\n\tSYNTAX Integer32\n\tMAX-ACCESS read-only\n\tSTATUS current\n\tDESCRIPTION \"test\"\n\t::= { test 1 }"))
	f.Add([]byte("'0a1b2c3d'H"))
	f.Add([]byte("'01101001'B"))
	f.Add([]byte("-- comment\n-- another comment\nIDENT"))
	f.Add([]byte("TEXTUAL-CONVENTION DISPLAY-HINT \"255a\" STATUS current DESCRIPTION \"\" SYNTAX OCTET STRING (SIZE (0..255))"))
	f.Add([]byte("INTEGER { up(1), down(2), testing(3) }"))
	f.Add([]byte("( 0..2147483647 )"))
	f.Add([]byte("::= { iso 3 6 1 2 1 }"))

	testutil.AddProblemCorpusSeeds(f)
	testutil.AddPrimaryCorpusSeeds(f)

	f.Fuzz(func(t *testing.T, data []byte) {
		// Skip pathologically large inputs to avoid excessive memory use.
		if len(data) > 1<<20 {
			return
		}

		l := New(data, nil)
		tokens, diags := l.Tokenize()

		if len(tokens) == 0 {
			t.Fatal("Tokenize returned no tokens")
		}
		if tokens[len(tokens)-1].Kind != TokEOF {
			t.Fatal("last token is not TokEOF")
		}
		for _, d := range diags {
			if d.Code == "" {
				t.Fatal("diagnostic has empty Code")
			}
		}

		// Validate token spans are within input bounds.
		inputLen := uint32(len(data))
		for i, tok := range tokens {
			start := uint32(tok.Span.Start)
			end := uint32(tok.Span.End)

			if start > end {
				t.Fatalf("token %d (%v): span start %d > end %d", i, tok.Kind, start, end)
			}
			// EOF can have a span at the end of input.
			if tok.Kind == TokEOF {
				if start > inputLen {
					t.Fatalf("EOF token: span start %d > input length %d", start, inputLen)
				}
				continue
			}
			if end > inputLen {
				t.Fatalf("token %d (%v): span end %d > input length %d", i, tok.Kind, end, inputLen)
			}
		}

		// Non-EOF tokens should appear in source order (non-decreasing starts).
		for i := 1; i < len(tokens)-1; i++ {
			if uint32(tokens[i].Span.Start) < uint32(tokens[i-1].Span.Start) {
				t.Fatalf("token %d start %d < token %d start %d (out of order)",
					i, tokens[i].Span.Start, i-1, tokens[i-1].Span.Start)
			}
		}
	})
}
