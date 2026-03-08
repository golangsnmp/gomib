package lexer

import (
	"strings"
	"testing"

	"github.com/golangsnmp/gomib/internal/testutil"
	"github.com/golangsnmp/gomib/internal/types"
)

func tokenKinds(source string) []TokenKind {
	lexer := New([]byte(source), nil, types.DefaultConfig())
	tokens, _ := lexer.Tokenize()
	kinds := make([]TokenKind, len(tokens))
	for i, t := range tokens {
		kinds[i] = t.Kind
	}
	return kinds
}

func tokenTexts(source string) []string {
	lexer := New([]byte(source), nil, types.DefaultConfig())
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
		TokKwInteger, TokUppercaseIdent, TokKwCounter32,
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
		TokComment,
		TokUppercaseIdent,
		TokEOF,
	}
	testutil.SliceEqual(t, expected, kinds, "token kinds")
}

func TestCommentsInline(t *testing.T) {
	kinds := tokenKinds("OBJECT -- comment -- TYPE")
	expected := []TokenKind{
		TokKwObject,
		TokComment,
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

func TestMacroSkipENDInsideIdentifier(t *testing.T) {
	// END appearing as a suffix of an identifier (e.g. PRETEND) must not
	// terminate the macro body early.
	tests := []struct {
		name   string
		source string
	}{
		{"suffix PRETEND", `OBJECT-TYPE MACRO ::= BEGIN PRETEND END` + "\n" + `ifIndex OBJECT-TYPE`},
		{"prefix ENDORSE", `OBJECT-TYPE MACRO ::= BEGIN ENDORSE END` + "\n" + `ifIndex OBJECT-TYPE`},
		{"prefix ENDING", `OBJECT-TYPE MACRO ::= BEGIN ENDING END` + "\n" + `ifIndex OBJECT-TYPE`},
		{"prefix ENDS", `OBJECT-TYPE MACRO ::= BEGIN ENDS END` + "\n" + `ifIndex OBJECT-TYPE`},
	}

	expected := []TokenKind{
		TokKwObjectType,
		TokKwMacro,
		TokKwEnd,
		TokLowercaseIdent,
		TokKwObjectType,
		TokEOF,
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			kinds := tokenKinds(tc.source)
			testutil.SliceEqual(t, expected, kinds, "token kinds")
		})
	}
}

func TestMacroSkipENDAtEOF(t *testing.T) {
	// END at end of input (no trailing newline) should still match.
	source := `OBJECT-TYPE MACRO ::= BEGIN stuff END`
	kinds := tokenKinds(source)
	expected := []TokenKind{
		TokKwObjectType,
		TokKwMacro,
		TokKwEnd,
		TokEOF,
	}
	testutil.SliceEqual(t, expected, kinds, "END at EOF")
}

func TestMacroSkipENDBeforeComment(t *testing.T) {
	// END immediately followed by a comment delimiter should match.
	source := "OBJECT-TYPE MACRO ::= BEGIN stuff END-- comment\nifIndex OBJECT-TYPE"
	kinds := tokenKinds(source)
	expected := []TokenKind{
		TokKwObjectType,
		TokKwMacro,
		TokKwEnd,
		TokComment,
		TokLowercaseIdent,
		TokKwObjectType,
		TokEOF,
	}
	testutil.SliceEqual(t, expected, kinds, "END before comment")
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
	lexer := New([]byte(source), nil, types.DefaultConfig())
	tokens, diagnostics := lexer.Tokenize()

	testutil.Equal(t, TokKwDefval, tokens[0].Kind, "first token")
	testutil.Equal(t, TokLBrace, tokens[1].Kind, "second token")
	testutil.Equal(t, TokForbiddenKeyword, tokens[2].Kind, "third token")
	testutil.Equal(t, TokRBrace, tokens[3].Kind, "fourth token")
	testutil.Len(t, diagnostics, 0, "diagnostics")
}

func TestDoubleHyphenBreaksIdentifier(t *testing.T) {
	source := "foo--bar"
	lexer := New([]byte(source), nil, types.DefaultConfig())
	tokens, _ := lexer.Tokenize()

	// foo--bar should tokenize as identifier "foo" then comment "--bar"
	testutil.Equal(t, TokLowercaseIdent, tokens[0].Kind, "first token kind")
	text := source[tokens[0].Span.Start:tokens[0].Span.End]
	testutil.Equal(t, "foo", text, "first token text")
	testutil.Equal(t, TokComment, tokens[1].Kind, "second token (comment)")
	testutil.Equal(t, TokEOF, tokens[2].Kind, "third token (EOF)")
}

func TestDoubleHyphenAfterIdentifierPreservesComment(t *testing.T) {
	// Realistic MIB pattern: identifier immediately before -- comment,
	// then more tokens on the next line
	source := "ifIndex--index column\nOBJECT-TYPE"
	lexer := New([]byte(source), nil, types.DefaultConfig())
	tokens, _ := lexer.Tokenize()

	testutil.Equal(t, TokLowercaseIdent, tokens[0].Kind, "identifier")
	text := source[tokens[0].Span.Start:tokens[0].Span.End]
	testutil.Equal(t, "ifIndex", text, "identifier text should not include hyphen")
	testutil.Equal(t, TokComment, tokens[1].Kind, "comment token")
	testutil.Equal(t, TokKwObjectType, tokens[2].Kind, "next token after comment")
}

func TestManyJunkLinesNoStackOverflow(t *testing.T) {
	// Verify iterative handling of many consecutive lines of unexpected chars.
	// Previously this would recurse once per line and risk stack overflow.
	var sb strings.Builder
	for range 10000 {
		sb.WriteString("@@@@\n")
	}
	sb.WriteString("OBJECT")
	source := sb.String()
	lexer := New([]byte(source), nil, types.DefaultConfig())
	tokens, diags := lexer.Tokenize()

	// Should find OBJECT after all the junk
	var found bool
	for _, tok := range tokens {
		if tok.Kind == TokKwObject {
			found = true
			break
		}
	}
	testutil.True(t, found, "should find OBJECT token after junk lines")
	testutil.Greater(t, len(diags), 0, "should have diagnostics for unexpected chars")
}

func TestManyConsecutiveCommentsNoStackOverflow(t *testing.T) {
	var sb strings.Builder
	for range 10000 {
		sb.WriteString("-- comment\n")
	}
	sb.WriteString("OBJECT")
	source := sb.String()
	lexer := New([]byte(source), nil, types.DefaultConfig())
	tokens, _ := lexer.Tokenize()

	// 10000 comment tokens + OBJECT + EOF
	testutil.Equal(t, TokComment, tokens[0].Kind, "first token should be comment")
	testutil.Equal(t, TokKwObject, tokens[10000].Kind, "should find OBJECT after many comments")
}

func TestUnterminatedQuotedString(t *testing.T) {
	source := `"unterminated string`
	lexer := New([]byte(source), nil, types.DefaultConfig())
	tokens, diagnostics := lexer.Tokenize()

	// Should produce a TokQuotedString token despite being unterminated
	testutil.Equal(t, TokQuotedString, tokens[0].Kind, "unterminated string token kind")
	testutil.Greater(t, len(diagnostics), 0, "should emit diagnostic for unterminated string")
	testutil.Contains(t, diagnostics[0].Message, "unterminated", "diagnostic message")
}

func TestUnterminatedHexString(t *testing.T) {
	source := "'0A1B"
	lexer := New([]byte(source), nil, types.DefaultConfig())
	tokens, diagnostics := lexer.Tokenize()

	// Should produce a TokError for unterminated hex/bin string
	testutil.Equal(t, TokError, tokens[0].Kind, "unterminated hex string token kind")
	testutil.Greater(t, len(diagnostics), 0, "should emit diagnostic for unterminated hex string")
}

func TestHexStringMissingSuffix(t *testing.T) {
	source := "'0A1B'X"
	lexer := New([]byte(source), nil, types.DefaultConfig())
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

func TestHexStringMul2(t *testing.T) {
	tests := []struct {
		source   string
		wantDiag bool
	}{
		{"'0A'H", false},       // 2 chars, OK
		{"'0A1B'H", false},     // 4 chars, OK
		{"'F'H", true},         // 1 char, odd
		{"'FFF'H", true},       // 3 chars, odd
		{"'ABCDEF0'H", true},   // 7 chars, odd
		{"''H", false},         // empty, OK
		{"'deadbeef'H", false}, // 8 chars, OK
	}
	cfg := types.VerboseConfig()
	for _, tt := range tests {
		lexer := New([]byte(tt.source), nil, cfg)
		tokens, diagnostics := lexer.Tokenize()
		testutil.Equal(t, TokHexString, tokens[0].Kind, "token kind for %s", tt.source)
		hasDiag := false
		for _, d := range diagnostics {
			if d.Code == types.DiagHexStringMul2 {
				hasDiag = true
			}
		}
		testutil.Equal(t, tt.wantDiag, hasDiag, "hex-string-mul2 diagnostic for %s", tt.source)
	}
}

func TestBinStringMul8(t *testing.T) {
	tests := []struct {
		source   string
		wantDiag bool
	}{
		{"'01010101'B", false},         // 8 bits, OK
		{"'1111000011110000'B", false}, // 16 bits, OK
		{"'10101'B", true},             // 5 bits, not multiple of 8
		{"'101010101010'B", true},      // 12 bits, not multiple of 8
		{"'1'B", true},                 // 1 bit, not multiple of 8
		{"''B", false},                 // empty, OK
	}
	cfg := types.VerboseConfig()
	for _, tt := range tests {
		lexer := New([]byte(tt.source), nil, cfg)
		tokens, diagnostics := lexer.Tokenize()
		testutil.Equal(t, TokBinString, tokens[0].Kind, "token kind for %s", tt.source)
		hasDiag := false
		for _, d := range diagnostics {
			if d.Code == types.DiagBinStringMul8 {
				hasDiag = true
			}
		}
		testutil.Equal(t, tt.wantDiag, hasDiag, "bin-string-mul8 diagnostic for %s", tt.source)
	}
}

func TestUnknownCharacter(t *testing.T) {
	// The lexer skips to end-of-line on unknown characters, so TYPE
	// on the next line should still be found.
	source := "OBJECT @ stuff\nTYPE"
	lexer := New([]byte(source), nil, types.DefaultConfig())
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
	lexer := New([]byte(source), nil, types.DefaultConfig())
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
	expected := []TokenKind{TokKwObject, TokComment, TokEOF}
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

func TestIsKeywordCoversAllKeywords(t *testing.T) {
	// Validate that every token in the keywords map passes IsKeyword().
	// Guards against reordering the iota constants in token.go.
	for text, kind := range keywords {
		if !kind.IsKeyword() {
			t.Errorf("keyword %q (token %d) not in IsKeyword() range", text, kind)
		}
	}

	// Non-keywords must not pass.
	nonKeywords := []TokenKind{
		TokError, TokEOF, TokForbiddenKeyword, TokComment,
		TokUppercaseIdent, TokLowercaseIdent,
		TokNumber, TokNegativeNumber, TokQuotedString, TokHexString, TokBinString,
		TokLBracket, TokRBracket, TokLBrace, TokRBrace, TokLParen, TokRParen,
		TokColon, TokSemicolon, TokComma, TokDot, TokPipe, TokMinus,
		TokDotDot, TokColonColonEqual,
	}
	for _, kind := range nonKeywords {
		if kind.IsKeyword() {
			t.Errorf("non-keyword token %d incorrectly passes IsKeyword()", kind)
		}
	}
}

func TestDiagnosticsHaveCode(t *testing.T) {
	tests := []struct {
		name   string
		source string
	}{
		{"unexpected character", "OBJECT \x01 TYPE"},
		{"unterminated string", `"unterminated`},
		{"unterminated hex string", "'0A1B"},
		{"missing suffix at EOF", "'0A1B'"},
		{"bad suffix", "'0A1B'X"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			lexer := New([]byte(tc.source), nil, types.DefaultConfig())
			_, diagnostics := lexer.Tokenize()

			testutil.Greater(t, len(diagnostics), 0, "expected at least one diagnostic")
			for _, d := range diagnostics {
				if d.Code == "" {
					t.Errorf("diagnostic has empty Code field: %q", d.Message)
				}
			}
		})
	}
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
		{"Integer32", TokError, false},
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

func TestCommentTokenSpan(t *testing.T) {
	tests := []struct {
		name     string
		source   string
		wantText string
	}{
		{"line comment", "OBJECT -- a comment\nTYPE", "-- a comment"},
		{"inline comment", "OBJECT -- inline -- TYPE", "-- inline --"},
		{"comment at EOF", "-- eof comment", "-- eof comment"},
		{"empty comment", "--\nTYPE", "--"},
		{"comment with dashes", "-- foo-bar --", "-- foo-bar --"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			lexer := New([]byte(tc.source), nil, types.DefaultConfig())
			tokens, _ := lexer.Tokenize()
			var found bool
			for _, tok := range tokens {
				if tok.Kind == TokComment {
					text := tc.source[tok.Span.Start:tok.Span.End]
					testutil.Equal(t, tc.wantText, text, "comment span text")
					found = true
					break
				}
			}
			testutil.True(t, found, "should find comment token")
		})
	}
}

func TestDoubleColonWithoutEquals(t *testing.T) {
	// "::" without "=" should produce two TokColon tokens, since
	// the lexer only combines "::=" as a single token.
	kinds := tokenKinds(":: OBJECT")
	expected := []TokenKind{
		TokColon, TokColon, TokKwObject, TokEOF,
	}
	testutil.SliceEqual(t, expected, kinds, ":: without =")
}

func TestDoubleColonFollowedByNonEquals(t *testing.T) {
	// "::x" should produce two colons then the identifier
	kinds := tokenKinds("::x")
	expected := []TokenKind{
		TokColon, TokColon, TokLowercaseIdent, TokEOF,
	}
	testutil.SliceEqual(t, expected, kinds, "::x")
}

func TestSingleColon(t *testing.T) {
	kinds := tokenKinds(": OBJECT")
	expected := []TokenKind{
		TokColon, TokKwObject, TokEOF,
	}
	testutil.SliceEqual(t, expected, kinds, "single colon")
}

func TestMacroBodyWithComments(t *testing.T) {
	// Comments inside a macro body should be skipped, and END should
	// still be found after the comment.
	source := `OBJECT-TYPE MACRO ::=
BEGIN
    TYPE NOTATION ::= -- this is a comment
        type(Syntax)
    VALUE NOTATION ::= -- another comment --
        value(VALUE ObjectName)
END

ifIndex OBJECT-TYPE`

	kinds := tokenKinds(source)
	expected := []TokenKind{
		TokKwObjectType,
		TokKwMacro,
		TokKwEnd,
		TokLowercaseIdent,
		TokKwObjectType,
		TokEOF,
	}
	testutil.SliceEqual(t, expected, kinds, "macro body with comments")
}

func TestMacroBodyWithCommentContainingEND(t *testing.T) {
	// A comment inside a macro body that contains "END" should not
	// cause the macro body to terminate early.
	source := `OBJECT-TYPE MACRO ::=
BEGIN
    -- this comment mentions END but should not terminate
    TYPE NOTATION ::= type(Syntax)
END

ifIndex OBJECT-TYPE`

	kinds := tokenKinds(source)
	expected := []TokenKind{
		TokKwObjectType,
		TokKwMacro,
		TokKwEnd,
		TokLowercaseIdent,
		TokKwObjectType,
		TokEOF,
	}
	testutil.SliceEqual(t, expected, kinds, "macro body comment containing END")
}

func TestMacroBodyEOFWithoutEnd(t *testing.T) {
	// Macro body that hits EOF without finding END should produce
	// TokEOF without crashing.
	source := `OBJECT-TYPE MACRO ::=
BEGIN
    TYPE NOTATION ::= type(Syntax)
    VALUE NOTATION ::= value(VALUE ObjectName)`

	kinds := tokenKinds(source)
	// Should get exactly: OBJECT-TYPE, MACRO, EOF (no END found)
	expected := []TokenKind{TokKwObjectType, TokKwMacro, TokEOF}
	testutil.SliceEqual(t, expected, kinds, "macro body EOF without END")
}

func TestExportsEOFWithoutSemicolon(t *testing.T) {
	// EXPORTS that hits EOF without a semicolon should produce TokEOF.
	source := "EXPORTS foo, bar, baz"
	kinds := tokenKinds(source)
	// Should get exactly: EXPORTS, EOF (body is consumed without semicolon)
	expected := []TokenKind{TokKwExports, TokEOF}
	testutil.SliceEqual(t, expected, kinds, "exports EOF without semicolon")
}

func TestNumberLeadingZero(t *testing.T) {
	tests := []struct {
		name     string
		source   string
		wantDiag bool
	}{
		{"plain zero", "0", false},
		{"single digit", "5", false},
		{"multi digit", "42", false},
		{"leading zero", "007", true},
		{"leading zeros", "00123", true},
		{"zero prefix", "01", true},
		{"negative no leading zero", "-5", false},
		{"negative leading zero", "-007", true},
		{"negative zero prefix", "-01", true},
		{"negative plain zero", "-0", false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg := types.VerboseConfig()
			lexer := New([]byte(tc.source), nil, cfg)
			_, diags := lexer.Tokenize()
			var found bool
			for _, d := range diags {
				if d.Code == types.DiagNumberLeadingZero {
					found = true
					break
				}
			}
			if tc.wantDiag && !found {
				t.Errorf("expected %s diagnostic for %q", types.DiagNumberLeadingZero, tc.source)
			}
			if !tc.wantDiag && found {
				t.Errorf("unexpected %s diagnostic for %q", types.DiagNumberLeadingZero, tc.source)
			}
		})
	}
}

func TestLibsmiNamesCompleteness(t *testing.T) {
	// Every valid TokenKind (excluding the sentinel) must have a
	// non-empty entry in libsmiNames so that LibsmiName() never
	// returns "UNKNOWN" for a real token kind.
	for k := range tokKeywordEnd {
		name := k.LibsmiName()
		if name == "UNKNOWN" {
			t.Errorf("TokenKind %d has no libsmiNames entry", k)
		}
	}
}
