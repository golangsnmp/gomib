package lexer

import (
	"testing"

	"github.com/golangsnmp/gomib/internal/testutil"
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

// === Error handling and edge cases ===

func TestUnterminatedQuotedString(t *testing.T) {
	source := `"unterminated string`
	lexer := New([]byte(source), nil)
	tokens, diagnostics := lexer.Tokenize()

	// Should produce a TokQuotedString token despite being unterminated
	testutil.Equal(t, TokQuotedString, tokens[0].Kind, "unterminated string token kind")
	testutil.Greater(t, len(diagnostics), 0, "should emit diagnostic for unterminated string")
	testutil.Contains(t, diagnostics[0].Message, "unterminated", "diagnostic message")
}

func TestUnterminatedHexString(t *testing.T) {
	source := "'0A1B"
	lexer := New([]byte(source), nil)
	tokens, diagnostics := lexer.Tokenize()

	// Should produce a TokError for unterminated hex/bin string
	testutil.Equal(t, TokError, tokens[0].Kind, "unterminated hex string token kind")
	testutil.Greater(t, len(diagnostics), 0, "should emit diagnostic for unterminated hex string")
}

func TestHexStringMissingSuffix(t *testing.T) {
	source := "'0A1B'X"
	lexer := New([]byte(source), nil)
	tokens, diagnostics := lexer.Tokenize()

	// 'X' is not H or B, should produce error
	testutil.Equal(t, TokError, tokens[0].Kind, "bad suffix should produce error token")
	testutil.Greater(t, len(diagnostics), 0, "should emit diagnostic for bad suffix")
}

func TestEmptyHexString(t *testing.T) {
	kinds := tokenKinds("''H")
	testutil.Equal(t, TokHexString, kinds[0], "empty hex string should tokenize")
}

func TestEmptyBinString(t *testing.T) {
	kinds := tokenKinds("''B")
	testutil.Equal(t, TokBinString, kinds[0], "empty bin string should tokenize")
}

func TestUnknownCharacter(t *testing.T) {
	// The lexer skips to end-of-line on unknown characters, so TYPE
	// on the next line should still be found.
	source := "OBJECT @ stuff\nTYPE"
	lexer := New([]byte(source), nil)
	tokens, diagnostics := lexer.Tokenize()

	var hasObject, hasType bool
	for _, tok := range tokens {
		text := source[tok.Span.Start:tok.Span.End]
		if text == "OBJECT" {
			hasObject = true
		}
		if text == "TYPE" {
			hasType = true
		}
	}
	testutil.True(t, hasObject, "should parse OBJECT before unknown char")
	testutil.True(t, hasType, "should parse TYPE on next line after unknown char")
	testutil.Greater(t, len(diagnostics), 0, "should emit diagnostic for unknown character")
}

func TestLargeNumber(t *testing.T) {
	texts := tokenTexts("4294967295 99999999999999")
	testutil.SliceEqual(t, []string{"4294967295", "99999999999999"}, texts, "large numbers")
}

func TestIdentifierWithUnderscore(t *testing.T) {
	// Underscores are allowed in identifiers (real-world vendor MIBs use them)
	texts := tokenTexts("my_object MY_MODULE")
	testutil.SliceEqual(t, []string{"my_object", "MY_MODULE"}, texts, "identifiers with underscores")
}

func TestIdentifierEndingWithHyphen(t *testing.T) {
	// An identifier ending with hyphen before non-identifier chars
	source := "test- OBJECT"
	lexer := New([]byte(source), nil)
	tokens, _ := lexer.Tokenize()

	// "test-" is the identifier (hyphen is consumed as part of identifier),
	// then whitespace, then OBJECT
	testutil.Equal(t, TokLowercaseIdent, tokens[0].Kind, "first token kind")
	text := source[tokens[0].Span.Start:tokens[0].Span.End]
	testutil.Equal(t, "test-", text, "identifier with trailing hyphen")
}

func TestMultilineQuotedString(t *testing.T) {
	source := "\"line1\nline2\nline3\""
	kinds := tokenKinds(source)
	testutil.Equal(t, TokQuotedString, kinds[0], "multiline string should tokenize as quoted string")
}

func TestCommentAtEOF(t *testing.T) {
	source := "OBJECT -- comment at end"
	kinds := tokenKinds(source)
	expected := []TokenKind{TokKwObject, TokEOF}
	testutil.SliceEqual(t, expected, kinds, "comment at EOF")
}

func TestOnlyWhitespace(t *testing.T) {
	kinds := tokenKinds("   \t\n\r\n  ")
	testutil.SliceEqual(t, []TokenKind{TokEOF}, kinds, "whitespace only")
}

func TestZeroNumber(t *testing.T) {
	kinds := tokenKinds("0")
	testutil.Equal(t, TokNumber, kinds[0], "zero is a number")
}

func TestConsecutivePunctuation(t *testing.T) {
	kinds := tokenKinds("{{}}()")
	expected := []TokenKind{
		TokLBrace, TokLBrace, TokRBrace, TokRBrace,
		TokLParen, TokRParen, TokEOF,
	}
	testutil.SliceEqual(t, expected, kinds, "consecutive punctuation")
}

func TestSMIv1TypeKeywords(t *testing.T) {
	kinds := tokenKinds("Counter Gauge NetworkAddress")
	expected := []TokenKind{
		TokKwCounter, TokKwGauge, TokKwNetworkAddress, TokEOF,
	}
	testutil.SliceEqual(t, expected, kinds, "SMIv1 type keywords")
}

func TestASN1TagKeywords(t *testing.T) {
	kinds := tokenKinds("APPLICATION IMPLICIT UNIVERSAL")
	expected := []TokenKind{
		TokKwApplication, TokKwImplicit, TokKwUniversal, TokEOF,
	}
	testutil.SliceEqual(t, expected, kinds, "ASN.1 tag keywords")
}

func TestAccessKeywords(t *testing.T) {
	kinds := tokenKinds("read-only read-write read-create write-only not-accessible accessible-for-notify not-implemented")
	expected := []TokenKind{
		TokKwReadOnly, TokKwReadWrite, TokKwReadCreate, TokKwWriteOnly,
		TokKwNotAccessible, TokKwAccessibleForNotify, TokKwNotImplemented,
		TokEOF,
	}
	testutil.SliceEqual(t, expected, kinds, "access keywords")
}

func TestStatusKeywords(t *testing.T) {
	kinds := tokenKinds("current deprecated obsolete mandatory optional")
	expected := []TokenKind{
		TokKwCurrent, TokKwDeprecated, TokKwObsolete,
		TokKwMandatory, TokKwOptional, TokEOF,
	}
	testutil.SliceEqual(t, expected, kinds, "status keywords")
}

func TestMacroKeywords(t *testing.T) {
	kinds := tokenKinds("MODULE-IDENTITY OBJECT-TYPE NOTIFICATION-TYPE TEXTUAL-CONVENTION TRAP-TYPE")
	expected := []TokenKind{
		TokKwModuleIdentity, TokKwObjectType, TokKwNotificationType,
		TokKwTextualConvention, TokKwTrapType, TokEOF,
	}
	testutil.SliceEqual(t, expected, kinds, "macro keywords")
}

func TestConformanceKeywords(t *testing.T) {
	kinds := tokenKinds("OBJECT-GROUP NOTIFICATION-GROUP MODULE-COMPLIANCE AGENT-CAPABILITIES")
	expected := []TokenKind{
		TokKwObjectGroup, TokKwNotificationGroup, TokKwModuleCompliance,
		TokKwAgentCapabilities, TokEOF,
	}
	testutil.SliceEqual(t, expected, kinds, "conformance keywords")
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
