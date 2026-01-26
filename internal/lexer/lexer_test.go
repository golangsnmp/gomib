package lexer

import (
	"testing"

	"github.com/golangsnmp/gomib/internal/testutil"
	"github.com/golangsnmp/gomib/internal/types"
)

// Helper to tokenize and get kinds only.
func tokenKinds(source string) []TokenKind {
	lexer := New([]byte(source), nil)
	tokens, _ := lexer.Tokenize()
	kinds := make([]TokenKind, len(tokens))
	for i, t := range tokens {
		kinds[i] = t.Kind
	}
	return kinds
}

// Helper to tokenize and get text slices.
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

func TestWhitespaceOnly(t *testing.T) {
	kinds := tokenKinds("   \t\n\r\n  ")
	testutil.SliceEqual(t, []TokenKind{TokEOF}, kinds, "whitespace only")
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

	kinds := tokenKinds("0 1 42 12345")
	expected := []TokenKind{
		TokNumber, TokNumber, TokNumber, TokNumber, TokEOF,
	}
	testutil.SliceEqual(t, expected, kinds, "token kinds")
}

func TestNegativeNumbers(t *testing.T) {
	texts := tokenTexts("-1 -42 -0")
	expectedTexts := []string{"-1", "-42", "-0"}
	testutil.SliceEqual(t, expectedTexts, texts, "token texts")

	kinds := tokenKinds("-1 -42")
	expected := []TokenKind{
		TokNegativeNumber, TokNegativeNumber, TokEOF,
	}
	testutil.SliceEqual(t, expected, kinds, "token kinds")
}

func TestIdentifiers(t *testing.T) {
	texts := tokenTexts("ifIndex myObject IF-MIB MyModule")
	expectedTexts := []string{"ifIndex", "myObject", "IF-MIB", "MyModule"}
	testutil.SliceEqual(t, expectedTexts, texts, "token texts")

	kinds := tokenKinds("ifIndex myObject IF-MIB MyModule")
	expected := []TokenKind{
		TokLowercaseIdent, TokLowercaseIdent,
		TokUppercaseIdent, TokUppercaseIdent, TokEOF,
	}
	testutil.SliceEqual(t, expected, kinds, "token kinds")
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

func TestMacroKeywords(t *testing.T) {
	kinds := tokenKinds("OBJECT-TYPE OBJECT-IDENTITY MODULE-IDENTITY")
	expected := []TokenKind{
		TokKwObjectType, TokKwObjectIdentity, TokKwModuleIdentity, TokEOF,
	}
	testutil.SliceEqual(t, expected, kinds, "token kinds")
}

func TestQuotedString(t *testing.T) {
	texts := tokenTexts(`"hello" "world" "with spaces"`)
	expectedTexts := []string{`"hello"`, `"world"`, `"with spaces"`}
	testutil.SliceEqual(t, expectedTexts, texts, "token texts")

	kinds := tokenKinds(`"hello"`)
	expected := []TokenKind{TokQuotedString, TokEOF}
	testutil.SliceEqual(t, expected, kinds, "token kinds")
}

func TestMultilineString(t *testing.T) {
	source := "\"line1\nline2\nline3\""
	kinds := tokenKinds(source)
	expected := []TokenKind{TokQuotedString, TokEOF}
	testutil.SliceEqual(t, expected, kinds, "token kinds")
}

func TestHexString(t *testing.T) {
	texts := tokenTexts("'0A1B'H 'ff00'h")
	expectedTexts := []string{"'0A1B'H", "'ff00'h"}
	testutil.SliceEqual(t, expectedTexts, texts, "token texts")

	kinds := tokenKinds("'0A1B'H")
	expected := []TokenKind{TokHexString, TokEOF}
	testutil.SliceEqual(t, expected, kinds, "token kinds")
}

func TestBinString(t *testing.T) {
	texts := tokenTexts("'01010101'B '11110000'b")
	expectedTexts := []string{"'01010101'B", "'11110000'b"}
	testutil.SliceEqual(t, expectedTexts, texts, "token texts")

	kinds := tokenKinds("'01010101'B")
	expected := []TokenKind{TokBinString, TokEOF}
	testutil.SliceEqual(t, expected, kinds, "token kinds")
}

func TestCommentsDashDash(t *testing.T) {
	// Comment ends at end of line
	kinds := tokenKinds("OBJECT -- comment\nTYPE")
	expected := []TokenKind{
		TokKwObject,
		TokUppercaseIdent, // TYPE is not a keyword
		TokEOF,
	}
	testutil.SliceEqual(t, expected, kinds, "token kinds")
}

func TestCommentsInline(t *testing.T) {
	// Comment ends at --
	kinds := tokenKinds("OBJECT -- comment -- TYPE")
	expected := []TokenKind{
		TokKwObject,
		TokUppercaseIdent, // TYPE
		TokEOF,
	}
	testutil.SliceEqual(t, expected, kinds, "token kinds")
}

func TestModuleHeader(t *testing.T) {
	source := "IF-MIB DEFINITIONS ::= BEGIN"
	kinds := tokenKinds(source)
	expected := []TokenKind{
		TokUppercaseIdent, // IF-MIB
		TokKwDefinitions,
		TokColonColonEqual,
		TokKwBegin,
		TokEOF,
	}
	testutil.SliceEqual(t, expected, kinds, "token kinds")
}

func TestObjectTypeDeclaration(t *testing.T) {
	source := `
		ifIndex OBJECT-TYPE
			SYNTAX      Integer32
			MAX-ACCESS  read-only
			STATUS      current
			DESCRIPTION "The index."
			::= { ifEntry 1 }
	`
	kinds := tokenKinds(source)
	expected := []TokenKind{
		TokLowercaseIdent, // ifIndex
		TokKwObjectType,
		TokKwSyntax,
		TokKwInteger32,
		TokKwMaxAccess,
		TokKwReadOnly,
		TokKwStatus,
		TokKwCurrent,
		TokKwDescription,
		TokQuotedString,
		TokColonColonEqual,
		TokLBrace,
		TokLowercaseIdent, // ifEntry
		TokNumber,         // 1
		TokRBrace,
		TokEOF,
	}
	testutil.SliceEqual(t, expected, kinds, "token kinds")
}

func TestImportsClause(t *testing.T) {
	source := `
		IMPORTS
			MODULE-IDENTITY, OBJECT-TYPE
				FROM SNMPv2-SMI
			DisplayString
				FROM SNMPv2-TC;
	`
	kinds := tokenKinds(source)
	expected := []TokenKind{
		TokKwImports,
		TokKwModuleIdentity,
		TokComma,
		TokKwObjectType,
		TokKwFrom,
		TokUppercaseIdent, // SNMPv2-SMI
		TokUppercaseIdent, // DisplayString
		TokKwFrom,
		TokUppercaseIdent, // SNMPv2-TC
		TokSemicolon,
		TokEOF,
	}
	testutil.SliceEqual(t, expected, kinds, "token kinds")
}

func TestMacroSkip(t *testing.T) {
	// MACRO body should be skipped entirely
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
		TokKwEnd,          // From the MACRO END
		TokLowercaseIdent, // ifIndex
		TokKwObjectType,
		TokEOF,
	}
	testutil.SliceEqual(t, expected, kinds, "token kinds")
}

func TestMacroSkipHyphenatedEnd(t *testing.T) {
	// END followed by single hyphen should NOT terminate macro
	source := `
		OBJECT-TYPE MACRO ::=
		BEGIN
			END-something is not a delimiter
		END

		ifIndex OBJECT-TYPE
	`
	kinds := tokenKinds(source)
	expected := []TokenKind{
		TokKwObjectType,
		TokKwMacro,
		TokKwEnd,          // From final "END" (not "END-something")
		TokLowercaseIdent, // ifIndex
		TokKwObjectType,
		TokEOF,
	}
	testutil.SliceEqual(t, expected, kinds, "token kinds")
}

func TestMacroEndWithDoubleHyphen(t *testing.T) {
	// END followed by -- (comment) should terminate macro
	source := `
		OBJECT-TYPE MACRO ::=
		BEGIN
			some content
		END-- comment terminates macro

		ifIndex OBJECT-TYPE
	`
	kinds := tokenKinds(source)
	expected := []TokenKind{
		TokKwObjectType,
		TokKwMacro,
		TokKwEnd,          // END-- works as delimiter
		TokLowercaseIdent, // ifIndex
		TokKwObjectType,
		TokEOF,
	}
	testutil.SliceEqual(t, expected, kinds, "token kinds")
}

func TestExportsSkip(t *testing.T) {
	// EXPORTS clause should be skipped
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

func TestChoiceTokenized(t *testing.T) {
	// CHOICE is now fully tokenized (not skipped)
	source := "NetworkAddress ::= CHOICE { internet IpAddress }Counter"
	kinds := tokenKinds(source)
	expected := []TokenKind{
		TokKwNetworkAddress,
		TokColonColonEqual,
		TokKwChoice,
		TokLBrace,
		TokLowercaseIdent, // internet
		TokKwIpAddress,
		TokRBrace,
		TokKwCounter,
		TokEOF,
	}
	testutil.SliceEqual(t, expected, kinds, "token kinds")
}

func TestOidValue(t *testing.T) {
	source := "{ iso org(3) dod(6) internet(1) }"
	kinds := tokenKinds(source)
	expected := []TokenKind{
		TokLBrace,
		TokLowercaseIdent, // iso
		TokLowercaseIdent, // org
		TokLParen,
		TokNumber, // 3
		TokRParen,
		TokLowercaseIdent, // dod
		TokLParen,
		TokNumber, // 6
		TokRParen,
		TokLowercaseIdent, // internet
		TokLParen,
		TokNumber, // 1
		TokRParen,
		TokRBrace,
		TokEOF,
	}
	testutil.SliceEqual(t, expected, kinds, "token kinds")
}

func TestRangeConstraint(t *testing.T) {
	source := "INTEGER (0..255)"
	kinds := tokenKinds(source)
	expected := []TokenKind{
		TokKwInteger,
		TokLParen,
		TokNumber, // 0
		TokDotDot,
		TokNumber, // 255
		TokRParen,
		TokEOF,
	}
	testutil.SliceEqual(t, expected, kinds, "token kinds")
}

func TestSizeConstraint(t *testing.T) {
	source := "OCTET STRING (SIZE (0..255))"
	kinds := tokenKinds(source)
	expected := []TokenKind{
		TokKwOctet,
		TokKwString,
		TokLParen,
		TokKwSize,
		TokLParen,
		TokNumber, // 0
		TokDotDot,
		TokNumber, // 255
		TokRParen,
		TokRParen,
		TokEOF,
	}
	testutil.SliceEqual(t, expected, kinds, "token kinds")
}

func TestEnumValues(t *testing.T) {
	source := "INTEGER { up(1), down(2), testing(3) }"
	kinds := tokenKinds(source)
	expected := []TokenKind{
		TokKwInteger,
		TokLBrace,
		TokLowercaseIdent, // up
		TokLParen,
		TokNumber, // 1
		TokRParen,
		TokComma,
		TokLowercaseIdent, // down
		TokLParen,
		TokNumber, // 2
		TokRParen,
		TokComma,
		TokLowercaseIdent, // testing
		TokLParen,
		TokNumber, // 3
		TokRParen,
		TokRBrace,
		TokEOF,
	}
	testutil.SliceEqual(t, expected, kinds, "token kinds")
}

func TestIdentifierWithUnderscore(t *testing.T) {
	// Per leniency philosophy, underscores should be accepted
	texts := tokenTexts("my_identifier SOME_TYPE")
	expectedTexts := []string{"my_identifier", "SOME_TYPE"}
	testutil.SliceEqual(t, expectedTexts, texts, "token texts")
}

func TestTrailingHyphenAccepted(t *testing.T) {
	lexer := New([]byte("bad-"), nil)
	tokens, diagnostics := lexer.Tokenize()

	testutil.Equal(t, TokLowercaseIdent, tokens[0].Kind, "should produce valid token")
	testutil.Len(t, diagnostics, 0, "should NOT produce diagnostics")
}

func TestLeadingZerosAccepted(t *testing.T) {
	lexer := New([]byte("007"), nil)
	tokens, diagnostics := lexer.Tokenize()

	testutil.Equal(t, TokNumber, tokens[0].Kind, "should produce number token")
	testutil.Len(t, diagnostics, 0, "should NOT produce diagnostics")
}

func TestSpanTracking(t *testing.T) {
	source := []byte("BEGIN END")
	lexer := New(source, nil)
	tokens, _ := lexer.Tokenize()

	testutil.Equal(t, TokKwBegin, tokens[0].Kind, "first token kind")
	testutil.Equal(t, 0, tokens[0].Span.Start, "first token span start")
	testutil.Equal(t, 5, tokens[0].Span.End, "first token span end")

	testutil.Equal(t, TokKwEnd, tokens[1].Kind, "second token kind")
	testutil.Equal(t, 6, tokens[1].Span.Start, "second token span start")
	testutil.Equal(t, 9, tokens[1].Span.End, "second token span end")
}

func TestStatusKeywords(t *testing.T) {
	kinds := tokenKinds("current deprecated obsolete mandatory optional")
	expected := []TokenKind{
		TokKwCurrent,
		TokKwDeprecated,
		TokKwObsolete,
		TokKwMandatory,
		TokKwOptional,
		TokEOF,
	}
	testutil.SliceEqual(t, expected, kinds, "token kinds")
}

func TestAccessKeywords(t *testing.T) {
	kinds := tokenKinds("read-only read-write read-create not-accessible")
	expected := []TokenKind{
		TokKwReadOnly,
		TokKwReadWrite,
		TokKwReadCreate,
		TokKwNotAccessible,
		TokEOF,
	}
	testutil.SliceEqual(t, expected, kinds, "token kinds")
}

func TestNonUtf8InString(t *testing.T) {
	// Latin-1 encoded "Vaclav" (common in vendor MIBs)
	// \xe1 is 'a' in Latin-1
	source := []byte("DESCRIPTION \"V\xe1clav\"")
	lexer := New(source, nil)
	tokens, diagnostics := lexer.Tokenize()

	// Should successfully tokenize despite non-UTF-8
	testutil.Equal(t, TokKwDescription, tokens[0].Kind, "first token")
	testutil.Equal(t, TokQuotedString, tokens[1].Kind, "second token")
	testutil.Equal(t, TokEOF, tokens[2].Kind, "third token")

	// No errors - non-UTF-8 in strings is accepted silently
	errors := filterErrors(diagnostics)
	testutil.Len(t, errors, 0, "errors")
}

func TestNonUtf8InComment(t *testing.T) {
	// Non-UTF-8 in comments should be skipped entirely
	source := []byte("BEGIN -- V\xe1clav --\nEND")
	lexer := New(source, nil)
	tokens, _ := lexer.Tokenize()

	testutil.Equal(t, TokKwBegin, tokens[0].Kind, "first token")
	testutil.Equal(t, TokKwEnd, tokens[1].Kind, "second token")
	testutil.Equal(t, TokEOF, tokens[2].Kind, "third token")
}

// ========================================================================
// Tests for identifiers containing double hyphens
// ========================================================================

func TestDoubleHyphenBreaksIdentifier(t *testing.T) {
	// Identifier with -- should be split
	source := "rfu-1plus1-tx-mhsb--rx-sd"
	lexer := New([]byte(source), nil)
	tokens, _ := lexer.Tokenize()

	// Should be: identifier, minus, identifier, EOF
	testutil.Len(t, tokens, 4, "token count")
	testutil.Equal(t, TokLowercaseIdent, tokens[0].Kind, "first token kind")
	text := source[tokens[0].Span.Start:tokens[0].Span.End]
	testutil.Equal(t, "rfu-1plus1-tx-mhsb-", text, "first token text")

	testutil.Equal(t, TokMinus, tokens[1].Kind, "second token kind")

	testutil.Equal(t, TokLowercaseIdent, tokens[2].Kind, "third token kind")
	text2 := source[tokens[2].Span.Start:tokens[2].Span.End]
	testutil.Equal(t, "rx-sd", text2, "third token text")

	testutil.Equal(t, TokEOF, tokens[3].Kind, "fourth token kind")
}

func TestDoubleHyphenSimple(t *testing.T) {
	// Simple case: foo--bar should become foo- + - + bar
	source := "foo--bar"
	lexer := New([]byte(source), nil)
	tokens, _ := lexer.Tokenize()

	testutil.Len(t, tokens, 4, "token count")
	testutil.Equal(t, TokLowercaseIdent, tokens[0].Kind, "first token kind")
	text := source[tokens[0].Span.Start:tokens[0].Span.End]
	testutil.Equal(t, "foo-", text, "first token text")

	testutil.Equal(t, TokMinus, tokens[1].Kind, "second token kind")

	testutil.Equal(t, TokLowercaseIdent, tokens[2].Kind, "third token kind")
	text2 := source[tokens[2].Span.Start:tokens[2].Span.End]
	testutil.Equal(t, "bar", text2, "third token text")
}

func TestDoubleHyphenAtStartIsComment(t *testing.T) {
	// --foo at start of input is a comment
	source := "--foo\nbar"
	lexer := New([]byte(source), nil)
	tokens, _ := lexer.Tokenize()

	// The -- starts a comment that runs to end of line
	testutil.Equal(t, TokLowercaseIdent, tokens[0].Kind, "first token kind")
	text := source[tokens[0].Span.Start:tokens[0].Span.End]
	testutil.Equal(t, "bar", text, "first token text")
}

func TestSingleHyphenInIdentifierOk(t *testing.T) {
	// Single hyphens are fine in identifiers
	source := "if-index my-object"
	kinds := tokenKinds(source)
	expected := []TokenKind{
		TokLowercaseIdent,
		TokLowercaseIdent,
		TokEOF,
	}
	testutil.SliceEqual(t, expected, kinds, "token kinds")

	texts := tokenTexts(source)
	expectedTexts := []string{"if-index", "my-object"}
	testutil.SliceEqual(t, expectedTexts, texts, "token texts")
}

// ========================================================================
// Tests for forbidden keyword detection
// ========================================================================

func TestForbiddenKeywordFalse(t *testing.T) {
	source := "DEFVAL {FALSE}"
	lexer := New([]byte(source), nil)
	tokens, diagnostics := lexer.Tokenize()

	testutil.Equal(t, TokKwDefval, tokens[0].Kind, "first token")
	testutil.Equal(t, TokLBrace, tokens[1].Kind, "second token")
	testutil.Equal(t, TokForbiddenKeyword, tokens[2].Kind, "third token")
	testutil.Equal(t, TokRBrace, tokens[3].Kind, "fourth token")

	// Should NOT produce diagnostics (lenient parsing)
	testutil.Len(t, diagnostics, 0, "diagnostics")
}

func TestForbiddenKeywordTrue(t *testing.T) {
	source := "DEFVAL {TRUE}"
	lexer := New([]byte(source), nil)
	tokens, diagnostics := lexer.Tokenize()

	testutil.Equal(t, TokForbiddenKeyword, tokens[2].Kind, "third token")

	// Should NOT produce diagnostics (lenient parsing)
	testutil.Len(t, diagnostics, 0, "diagnostics")
}

func TestForbiddenKeywordsList(t *testing.T) {
	forbidden := []string{
		"ABSENT", "ANY", "BIT", "BOOLEAN", "DEFAULT", "NULL", "PRIVATE", "SET",
	}

	for _, kw := range forbidden {
		lexer := New([]byte(kw), nil)
		tokens, diagnostics := lexer.Tokenize()

		testutil.Equal(t, TokForbiddenKeyword, tokens[0].Kind, kw+" should be forbidden keyword")
		// Should NOT produce diagnostics (lenient parsing)
		testutil.Len(t, diagnostics, 0, kw+" diagnostics")
	}
}

func TestMaxMinForbidden(t *testing.T) {
	// MAX and MIN are forbidden ASN.1 keywords
	source := "MAX MIN"
	lexer := New([]byte(source), nil)
	tokens, diagnostics := lexer.Tokenize()

	testutil.Equal(t, TokForbiddenKeyword, tokens[0].Kind, "MAX should be forbidden")
	testutil.Equal(t, TokForbiddenKeyword, tokens[1].Kind, "MIN should be forbidden")
	// Should NOT produce diagnostics (lenient parsing)
	testutil.Len(t, diagnostics, 0, "diagnostics")
}

func TestValidKeywordsNotForbidden(t *testing.T) {
	// Make sure valid SMI keywords are not mistakenly marked as forbidden
	valid := []string{"BEGIN", "END", "OBJECT", "INTEGER", "SYNTAX", "STATUS"}

	for _, kw := range valid {
		lexer := New([]byte(kw), nil)
		tokens, _ := lexer.Tokenize()

		testutil.True(t, tokens[0].Kind != TokError, kw+" should NOT be a forbidden keyword")
	}
}

func TestForbiddenKeywordCaseSensitive(t *testing.T) {
	// Forbidden keywords are case-sensitive (all uppercase)
	source := "false"
	lexer := New([]byte(source), nil)
	tokens, _ := lexer.Tokenize()

	testutil.Equal(t, TokLowercaseIdent, tokens[0].Kind, "'false' should be lowercase identifier")
}

// ========================================================================
// Tests for skip-to-EOL error recovery (matching libsmi behavior)
// ========================================================================

func TestErrorRecoverySkipsToEOL(t *testing.T) {
	// When encountering an unexpected character, skip to end of line
	// Fullwidth comma U+FF0C is 3 bytes: ef bc 8c
	source := []byte("foo\xef\xbc\x8c\nbar")
	lexer := New(source, nil)
	tokens, diagnostics := lexer.Tokenize()

	// Should be: foo, (error skips rest of line), bar, EOF
	testutil.Len(t, tokens, 3, "token count")
	testutil.Equal(t, TokLowercaseIdent, tokens[0].Kind, "first token")
	testutil.Equal(t, TokLowercaseIdent, tokens[1].Kind, "second token")
	testutil.Equal(t, TokEOF, tokens[2].Kind, "third token")

	// Should have one error diagnostic
	testutil.Len(t, diagnostics, 1, "diagnostics")
}

func TestErrorRecoveryMultipleBadCharsSameLine(t *testing.T) {
	// Multiple bad characters on same line should result in one skip
	source := []byte("foo \x80\x81\x82 ignored\nbar")
	lexer := New(source, nil)
	tokens, diagnostics := lexer.Tokenize()

	// Should skip from first bad char to EOL
	testutil.Len(t, tokens, 3, "token count")
	testutil.Equal(t, TokLowercaseIdent, tokens[0].Kind, "first token")
	testutil.Equal(t, TokLowercaseIdent, tokens[1].Kind, "second token")
	testutil.Len(t, diagnostics, 1, "diagnostics")
}

func TestErrorRecoveryPreservesSameLineBeforeError(t *testing.T) {
	// Tokens before the error on the same line should be preserved
	source := []byte("BEGIN foo \x80 ignored\nEND")
	lexer := New(source, nil)
	tokens, _ := lexer.Tokenize()

	testutil.Equal(t, TokKwBegin, tokens[0].Kind, "first token")
	testutil.Equal(t, TokLowercaseIdent, tokens[1].Kind, "second token (foo)")
	testutil.Equal(t, TokKwEnd, tokens[2].Kind, "third token")
	testutil.Equal(t, TokEOF, tokens[3].Kind, "fourth token")
}

func TestErrorAtEofNoInfiniteLoop(t *testing.T) {
	// Error at EOF should not cause issues
	source := []byte("foo\x80")
	lexer := New(source, nil)
	tokens, diagnostics := lexer.Tokenize()

	testutil.Len(t, tokens, 2, "token count")
	testutil.Equal(t, TokLowercaseIdent, tokens[0].Kind, "first token")
	testutil.Equal(t, TokEOF, tokens[1].Kind, "second token")
	testutil.Len(t, diagnostics, 1, "diagnostics")
}

// ========================================================================
// Tests for odd-dashes comment handling (matching libsmi behavior)
// ========================================================================

func TestOddDashesThreeAtEOL(t *testing.T) {
	// Three dashes at end of line when in comment state should not emit MINUS
	source := "foo ---\nbar"
	lexer := New([]byte(source), nil)
	tokens, _ := lexer.Tokenize()

	testutil.Len(t, tokens, 3, "token count")
	testutil.Equal(t, TokLowercaseIdent, tokens[0].Kind, "first token")
	testutil.Equal(t, TokLowercaseIdent, tokens[1].Kind, "second token")
	testutil.Equal(t, TokEOF, tokens[2].Kind, "third token")
}

func TestOddDashes81Dashes(t *testing.T) {
	// 81 dashes = separator line common in MIBs
	dashes := ""
	for i := 0; i < 81; i++ {
		dashes += "-"
	}
	source := "foo\n" + dashes + "\nbar"
	lexer := New([]byte(source), nil)
	tokens, _ := lexer.Tokenize()

	// Should only have: foo + bar + EOF (no MINUS tokens)
	testutil.Len(t, tokens, 3, "token count")
	testutil.Equal(t, TokLowercaseIdent, tokens[0].Kind, "first token")
	testutil.Equal(t, TokLowercaseIdent, tokens[1].Kind, "second token")
	testutil.Equal(t, TokEOF, tokens[2].Kind, "third token")
}

func TestOddDashesFiveDashesAtEOL(t *testing.T) {
	// 5 dashes followed by EOL
	source := "foo -----\nbar"
	lexer := New([]byte(source), nil)
	tokens, _ := lexer.Tokenize()

	testutil.Len(t, tokens, 3, "token count")
	testutil.Equal(t, TokLowercaseIdent, tokens[0].Kind, "first token (foo)")
	testutil.Equal(t, TokLowercaseIdent, tokens[1].Kind, "second token (bar)")
	testutil.Equal(t, TokEOF, tokens[2].Kind, "third token")
}

func TestOddDashesFiveNotAtEOL(t *testing.T) {
	// 5 dashes NOT followed by EOL: -- (enter) -- (exit) - (MINUS)
	source := "foo ----- bar"
	lexer := New([]byte(source), nil)
	tokens, _ := lexer.Tokenize()

	// foo, then 5 dashes: -- enters comment, -- exits, - becomes MINUS
	testutil.Equal(t, TokLowercaseIdent, tokens[0].Kind, "first token (foo)")
	testutil.Equal(t, TokMinus, tokens[1].Kind, "second token (5th dash)")
	testutil.Equal(t, TokLowercaseIdent, tokens[2].Kind, "third token (bar)")
	testutil.Equal(t, TokEOF, tokens[3].Kind, "fourth token")
}

func TestOddDashesSevenDashes(t *testing.T) {
	// 7 dashes: --(enter) --(exit) --(enter) -(in comment) \n(end comment)
	source := "foo -------\nbar"
	lexer := New([]byte(source), nil)
	tokens, _ := lexer.Tokenize()

	// No MINUS emitted because the single dash is inside a comment
	testutil.Len(t, tokens, 3, "token count")
	testutil.Equal(t, TokLowercaseIdent, tokens[0].Kind, "first token (foo)")
	testutil.Equal(t, TokLowercaseIdent, tokens[1].Kind, "second token (bar)")
	testutil.Equal(t, TokEOF, tokens[2].Kind, "third token")
}

func TestOddDashesNineAtEOL(t *testing.T) {
	// 9 dashes followed by EOL
	source := "foo ---------\nbar"
	lexer := New([]byte(source), nil)
	tokens, _ := lexer.Tokenize()

	testutil.Len(t, tokens, 3, "token count")
	testutil.Equal(t, TokLowercaseIdent, tokens[0].Kind, "first token (foo)")
	testutil.Equal(t, TokLowercaseIdent, tokens[1].Kind, "second token (bar)")
	testutil.Equal(t, TokEOF, tokens[2].Kind, "third token")
}

// ========================================================================
// Keyword tests
// ========================================================================

func TestKeywordsSorted(t *testing.T) {
	// Verify the keyword table is sorted
	for i := 1; i < len(keywords); i++ {
		testutil.True(t, keywords[i-1].text < keywords[i].text,
			"keywords not sorted: %s should come before %s", keywords[i-1].text, keywords[i].text)
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
		{"END", TokKwEnd, true},
		{"Integer32", TokKwInteger32, true},
		{"Counter32", TokKwCounter32, true},
		{"current", TokKwCurrent, true},
		{"read-only", TokKwReadOnly, true},
		{"ifIndex", TokError, false},
		{"MyModule", TokError, false},
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

func TestKeywordCaseSensitive(t *testing.T) {
	// Keywords are case-sensitive
	kind, ok := LookupKeyword("OBJECT-TYPE")
	testutil.True(t, ok && kind == TokKwObjectType, "OBJECT-TYPE should be a keyword")

	_, ok = LookupKeyword("object-type")
	testutil.True(t, !ok, "object-type should NOT be a keyword")

	_, ok = LookupKeyword("Object-Type")
	testutil.True(t, !ok, "Object-Type should NOT be a keyword")

	kind, ok = LookupKeyword("Integer32")
	testutil.True(t, ok && kind == TokKwInteger32, "Integer32 should be a keyword")

	_, ok = LookupKeyword("INTEGER32")
	testutil.True(t, !ok, "INTEGER32 should NOT be a keyword")

	_, ok = LookupKeyword("integer32")
	testutil.True(t, !ok, "integer32 should NOT be a keyword")
}

func TestForbiddenKeywordsSorted(t *testing.T) {
	// Verify the forbidden keyword table is sorted
	for i := 1; i < len(forbiddenKeywords); i++ {
		testutil.True(t, forbiddenKeywords[i-1] < forbiddenKeywords[i],
			"forbidden keywords not sorted: %s should come before %s", forbiddenKeywords[i-1], forbiddenKeywords[i])
	}
}

func TestForbiddenKeywordLookup(t *testing.T) {
	// Test some forbidden keywords
	forbidden := []string{"FALSE", "TRUE", "NULL", "ABSENT", "MAX", "MIN", "PRIVATE", "SET"}
	for _, kw := range forbidden {
		testutil.True(t, IsForbiddenKeyword(kw), "%s should be a forbidden keyword", kw)
	}

	// Test non-forbidden keywords
	notForbidden := []string{"BEGIN", "END", "OBJECT-TYPE", "INTEGER", "ifIndex"}
	for _, kw := range notForbidden {
		testutil.True(t, !IsForbiddenKeyword(kw), "%s should NOT be a forbidden keyword", kw)
	}

	// Test case sensitivity
	testutil.True(t, IsForbiddenKeyword("FALSE"), "FALSE should be a forbidden keyword")
	testutil.True(t, !IsForbiddenKeyword("false"), "false should NOT be a forbidden keyword (case-sensitive)")
	testutil.True(t, !IsForbiddenKeyword("False"), "False should NOT be a forbidden keyword (case-sensitive)")
}

// ========================================================================
// Helper functions
// ========================================================================

func filterErrors(diags []types.Diagnostic) []types.Diagnostic {
	var errors []types.Diagnostic
	for _, d := range diags {
		if d.Severity == types.SeverityError {
			errors = append(errors, d)
		}
	}
	return errors
}

// ========================================================================
// Benchmarks
// ========================================================================

func BenchmarkTokenize(b *testing.B) {
	// A realistic MIB snippet with many identifiers and keywords
	source := []byte(`
IF-MIB DEFINITIONS ::= BEGIN

IMPORTS
    MODULE-IDENTITY, OBJECT-TYPE, Counter32, Gauge32,
    Integer32, TimeTicks, Counter64, NOTIFICATION-TYPE
        FROM SNMPv2-SMI
    DisplayString, TruthValue, RowStatus
        FROM SNMPv2-TC
    MODULE-COMPLIANCE, OBJECT-GROUP
        FROM SNMPv2-CONF;

ifTable OBJECT-TYPE
    SYNTAX      SEQUENCE OF IfEntry
    MAX-ACCESS  not-accessible
    STATUS      current
    DESCRIPTION "A list of interface entries."
    ::= { interfaces 2 }

ifEntry OBJECT-TYPE
    SYNTAX      IfEntry
    MAX-ACCESS  not-accessible
    STATUS      current
    DESCRIPTION "An entry containing management information."
    INDEX   { ifIndex }
    ::= { ifTable 1 }

ifIndex OBJECT-TYPE
    SYNTAX      Integer32
    MAX-ACCESS  read-only
    STATUS      current
    DESCRIPTION "A unique value for each interface."
    ::= { ifEntry 1 }

ifDescr OBJECT-TYPE
    SYNTAX      DisplayString
    MAX-ACCESS  read-only
    STATUS      current
    DESCRIPTION "A textual string containing information."
    ::= { ifEntry 2 }

END
`)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		lexer := New(source, nil)
		lexer.Tokenize()
	}
}

func BenchmarkLookupKeyword(b *testing.B) {
	keywords := []string{
		"OBJECT-TYPE", "INTEGER", "SYNTAX", "STATUS", "DESCRIPTION",
		"read-only", "current", "Counter32", "ifIndex", "sysDescr",
	}

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		for _, kw := range keywords {
			LookupKeyword(kw)
		}
	}
}
