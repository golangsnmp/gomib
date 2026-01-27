package lexer

import (
	"testing"

	"github.com/golangsnmp/gomib/internal/testutil"
	"github.com/golangsnmp/gomib/internal/types"
)

func tokenKinds(source string) []TokenKind {
	lexer := New([]byte(source), nil)
	tokens, _ := lexer.Tokenize()
	kinds := make([]TokenKind, len(tokens))
	for i, t := range tokens {
		kinds[i] = t.Kind
	}
	return kinds
}

func tokenTexts(source string) []string {
	lexer := New([]byte(source), nil)
	tokens, _ := lexer.Tokenize()
	var texts []string
	for _, t := range tokens {
		if t.Kind != TokEOF {
			texts = append(texts, source[t.Span.Start:t.Span.End])
		}
	}
	return texts
}

func TestEmptyInput(t *testing.T) {
	kinds := tokenKinds("")
	testutil.SliceEqual(t, []TokenKind{TokEOF}, kinds, "empty input")
}

func TestPunctuation(t *testing.T) {
	kinds := tokenKinds("[ ] { } ( ) ; , . |")
	expected := []TokenKind{
		TokLBracket, TokRBracket, TokLBrace, TokRBrace,
		TokLParen, TokRParen, TokSemicolon, TokComma,
		TokDot, TokPipe, TokEOF,
	}
	testutil.SliceEqual(t, expected, kinds, "token kinds")
}

func TestOperators(t *testing.T) {
	kinds := tokenKinds(".. ::= : -")
	expected := []TokenKind{
		TokDotDot, TokColonColonEqual, TokColon, TokMinus, TokEOF,
	}
	testutil.SliceEqual(t, expected, kinds, "token kinds")
}

func TestNumbers(t *testing.T) {
	texts := tokenTexts("0 1 42 12345")
	expectedTexts := []string{"0", "1", "42", "12345"}
	testutil.SliceEqual(t, expectedTexts, texts, "token texts")
}

func TestNegativeNumbers(t *testing.T) {
	texts := tokenTexts("-1 -42 -0")
	expectedTexts := []string{"-1", "-42", "-0"}
	testutil.SliceEqual(t, expectedTexts, texts, "token texts")
}

func TestIdentifiers(t *testing.T) {
	texts := tokenTexts("ifIndex myObject IF-MIB MyModule")
	expectedTexts := []string{"ifIndex", "myObject", "IF-MIB", "MyModule"}
	testutil.SliceEqual(t, expectedTexts, texts, "token texts")
}

func TestKeywords(t *testing.T) {
	kinds := tokenKinds("DEFINITIONS BEGIN END IMPORTS FROM")
	expected := []TokenKind{
		TokKwDefinitions, TokKwBegin, TokKwEnd,
		TokKwImports, TokKwFrom, TokEOF,
	}
	testutil.SliceEqual(t, expected, kinds, "token kinds")
}

func TestTypeKeywords(t *testing.T) {
	kinds := tokenKinds("INTEGER Integer32 Counter32 Counter64 Gauge32")
	expected := []TokenKind{
		TokKwInteger, TokKwInteger32, TokKwCounter32,
		TokKwCounter64, TokKwGauge32, TokEOF,
	}
	testutil.SliceEqual(t, expected, kinds, "token kinds")
}

func TestQuotedString(t *testing.T) {
	texts := tokenTexts(`"hello" "world" "with spaces"`)
	expectedTexts := []string{`"hello"`, `"world"`, `"with spaces"`}
	testutil.SliceEqual(t, expectedTexts, texts, "token texts")
}

func TestHexString(t *testing.T) {
	texts := tokenTexts("'0A1B'H 'ff00'h")
	expectedTexts := []string{"'0A1B'H", "'ff00'h"}
	testutil.SliceEqual(t, expectedTexts, texts, "token texts")
}

func TestBinString(t *testing.T) {
	texts := tokenTexts("'01010101'B '11110000'b")
	expectedTexts := []string{"'01010101'B", "'11110000'b"}
	testutil.SliceEqual(t, expectedTexts, texts, "token texts")
}

func TestCommentsDashDash(t *testing.T) {
	kinds := tokenKinds("OBJECT -- comment\nTYPE")
	expected := []TokenKind{
		TokKwObject,
		TokUppercaseIdent,
		TokEOF,
	}
	testutil.SliceEqual(t, expected, kinds, "token kinds")
}

func TestCommentsInline(t *testing.T) {
	kinds := tokenKinds("OBJECT -- comment -- TYPE")
	expected := []TokenKind{
		TokKwObject,
		TokUppercaseIdent,
		TokEOF,
	}
	testutil.SliceEqual(t, expected, kinds, "token kinds")
}

func TestModuleHeader(t *testing.T) {
	source := "IF-MIB DEFINITIONS ::= BEGIN"
	kinds := tokenKinds(source)
	expected := []TokenKind{
		TokUppercaseIdent,
		TokKwDefinitions,
		TokColonColonEqual,
		TokKwBegin,
		TokEOF,
	}
	testutil.SliceEqual(t, expected, kinds, "token kinds")
}

func TestMacroSkip(t *testing.T) {
	source := `
		OBJECT-TYPE MACRO ::=
		BEGIN
			TYPE NOTATION ::= ...lots of content...
			VALUE NOTATION ::= value
		END

		ifIndex OBJECT-TYPE
	`
	kinds := tokenKinds(source)
	expected := []TokenKind{
		TokKwObjectType,
		TokKwMacro,
		TokKwEnd,
		TokLowercaseIdent,
		TokKwObjectType,
		TokEOF,
	}
	testutil.SliceEqual(t, expected, kinds, "token kinds")
}

func TestExportsSkip(t *testing.T) {
	source := "EXPORTS foo, bar, baz;OBJECT-TYPE"
	kinds := tokenKinds(source)
	expected := []TokenKind{
		TokKwExports,
		TokSemicolon,
		TokKwObjectType,
		TokEOF,
	}
	testutil.SliceEqual(t, expected, kinds, "token kinds")
}

func TestForbiddenKeywordFalse(t *testing.T) {
	source := "DEFVAL {FALSE}"
	lexer := New([]byte(source), nil)
	tokens, diagnostics := lexer.Tokenize()

	testutil.Equal(t, TokKwDefval, tokens[0].Kind, "first token")
	testutil.Equal(t, TokLBrace, tokens[1].Kind, "second token")
	testutil.Equal(t, TokForbiddenKeyword, tokens[2].Kind, "third token")
	testutil.Equal(t, TokRBrace, tokens[3].Kind, "fourth token")
	testutil.Len(t, diagnostics, 0, "diagnostics")
}

func TestDoubleHyphenBreaksIdentifier(t *testing.T) {
	source := "foo--bar"
	lexer := New([]byte(source), nil)
	tokens, _ := lexer.Tokenize()

	testutil.Len(t, tokens, 4, "token count")
	testutil.Equal(t, TokLowercaseIdent, tokens[0].Kind, "first token kind")
	text := source[tokens[0].Span.Start:tokens[0].Span.End]
	testutil.Equal(t, "foo-", text, "first token text")
}

func TestKeywordLookup(t *testing.T) {
	tests := []struct {
		text     string
		expected TokenKind
		found    bool
	}{
		{"OBJECT-TYPE", TokKwObjectType, true},
		{"DEFINITIONS", TokKwDefinitions, true},
		{"BEGIN", TokKwBegin, true},
		{"Integer32", TokKwInteger32, true},
		{"current", TokKwCurrent, true},
		{"ifIndex", TokError, false},
		{"", TokError, false},
	}

	for _, tc := range tests {
		kind, found := LookupKeyword(tc.text)
		testutil.Equal(t, tc.found, found, "LookupKeyword(%q) found", tc.text)
		if found {
			testutil.Equal(t, tc.expected, kind, "LookupKeyword(%q) kind", tc.text)
		}
	}
}

func filterErrors(diags []types.Diagnostic) []types.Diagnostic {
	var errors []types.Diagnostic
	for _, d := range diags {
		if d.Severity == types.SeverityError {
			errors = append(errors, d)
		}
	}
	return errors
}
