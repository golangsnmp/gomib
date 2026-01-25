package lexer

import (
	"testing"

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
	if len(kinds) != 1 || kinds[0] != TokEOF {
		t.Errorf("expected [TokEOF], got %v", kinds)
	}
}

func TestWhitespaceOnly(t *testing.T) {
	kinds := tokenKinds("   \t\n\r\n  ")
	if len(kinds) != 1 || kinds[0] != TokEOF {
		t.Errorf("expected [TokEOF], got %v", kinds)
	}
}

func TestPunctuation(t *testing.T) {
	kinds := tokenKinds("[ ] { } ( ) ; , . |")
	expected := []TokenKind{
		TokLBracket, TokRBracket, TokLBrace, TokRBrace,
		TokLParen, TokRParen, TokSemicolon, TokComma,
		TokDot, TokPipe, TokEOF,
	}
	assertTokenKinds(t, expected, kinds)
}

func TestOperators(t *testing.T) {
	kinds := tokenKinds(".. ::= : -")
	expected := []TokenKind{
		TokDotDot, TokColonColonEqual, TokColon, TokMinus, TokEOF,
	}
	assertTokenKinds(t, expected, kinds)
}

func TestNumbers(t *testing.T) {
	texts := tokenTexts("0 1 42 12345")
	expectedTexts := []string{"0", "1", "42", "12345"}
	assertTexts(t, expectedTexts, texts)

	kinds := tokenKinds("0 1 42 12345")
	expected := []TokenKind{
		TokNumber, TokNumber, TokNumber, TokNumber, TokEOF,
	}
	assertTokenKinds(t, expected, kinds)
}

func TestNegativeNumbers(t *testing.T) {
	texts := tokenTexts("-1 -42 -0")
	expectedTexts := []string{"-1", "-42", "-0"}
	assertTexts(t, expectedTexts, texts)

	kinds := tokenKinds("-1 -42")
	expected := []TokenKind{
		TokNegativeNumber, TokNegativeNumber, TokEOF,
	}
	assertTokenKinds(t, expected, kinds)
}

func TestIdentifiers(t *testing.T) {
	texts := tokenTexts("ifIndex myObject IF-MIB MyModule")
	expectedTexts := []string{"ifIndex", "myObject", "IF-MIB", "MyModule"}
	assertTexts(t, expectedTexts, texts)

	kinds := tokenKinds("ifIndex myObject IF-MIB MyModule")
	expected := []TokenKind{
		TokLowercaseIdent, TokLowercaseIdent,
		TokUppercaseIdent, TokUppercaseIdent, TokEOF,
	}
	assertTokenKinds(t, expected, kinds)
}

func TestKeywords(t *testing.T) {
	kinds := tokenKinds("DEFINITIONS BEGIN END IMPORTS FROM")
	expected := []TokenKind{
		TokKwDefinitions, TokKwBegin, TokKwEnd,
		TokKwImports, TokKwFrom, TokEOF,
	}
	assertTokenKinds(t, expected, kinds)
}

func TestTypeKeywords(t *testing.T) {
	kinds := tokenKinds("INTEGER Integer32 Counter32 Counter64 Gauge32")
	expected := []TokenKind{
		TokKwInteger, TokKwInteger32, TokKwCounter32,
		TokKwCounter64, TokKwGauge32, TokEOF,
	}
	assertTokenKinds(t, expected, kinds)
}

func TestMacroKeywords(t *testing.T) {
	kinds := tokenKinds("OBJECT-TYPE OBJECT-IDENTITY MODULE-IDENTITY")
	expected := []TokenKind{
		TokKwObjectType, TokKwObjectIdentity, TokKwModuleIdentity, TokEOF,
	}
	assertTokenKinds(t, expected, kinds)
}

func TestQuotedString(t *testing.T) {
	texts := tokenTexts(`"hello" "world" "with spaces"`)
	expectedTexts := []string{`"hello"`, `"world"`, `"with spaces"`}
	assertTexts(t, expectedTexts, texts)

	kinds := tokenKinds(`"hello"`)
	expected := []TokenKind{TokQuotedString, TokEOF}
	assertTokenKinds(t, expected, kinds)
}

func TestMultilineString(t *testing.T) {
	source := "\"line1\nline2\nline3\""
	kinds := tokenKinds(source)
	expected := []TokenKind{TokQuotedString, TokEOF}
	assertTokenKinds(t, expected, kinds)
}

func TestHexString(t *testing.T) {
	texts := tokenTexts("'0A1B'H 'ff00'h")
	expectedTexts := []string{"'0A1B'H", "'ff00'h"}
	assertTexts(t, expectedTexts, texts)

	kinds := tokenKinds("'0A1B'H")
	expected := []TokenKind{TokHexString, TokEOF}
	assertTokenKinds(t, expected, kinds)
}

func TestBinString(t *testing.T) {
	texts := tokenTexts("'01010101'B '11110000'b")
	expectedTexts := []string{"'01010101'B", "'11110000'b"}
	assertTexts(t, expectedTexts, texts)

	kinds := tokenKinds("'01010101'B")
	expected := []TokenKind{TokBinString, TokEOF}
	assertTokenKinds(t, expected, kinds)
}

func TestCommentsDashDash(t *testing.T) {
	// Comment ends at end of line
	kinds := tokenKinds("OBJECT -- comment\nTYPE")
	expected := []TokenKind{
		TokKwObject,
		TokUppercaseIdent, // TYPE is not a keyword
		TokEOF,
	}
	assertTokenKinds(t, expected, kinds)
}

func TestCommentsInline(t *testing.T) {
	// Comment ends at --
	kinds := tokenKinds("OBJECT -- comment -- TYPE")
	expected := []TokenKind{
		TokKwObject,
		TokUppercaseIdent, // TYPE
		TokEOF,
	}
	assertTokenKinds(t, expected, kinds)
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
	assertTokenKinds(t, expected, kinds)
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
	assertTokenKinds(t, expected, kinds)
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
	assertTokenKinds(t, expected, kinds)
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
	assertTokenKinds(t, expected, kinds)
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
	assertTokenKinds(t, expected, kinds)
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
	assertTokenKinds(t, expected, kinds)
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
	assertTokenKinds(t, expected, kinds)
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
	assertTokenKinds(t, expected, kinds)
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
	assertTokenKinds(t, expected, kinds)
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
	assertTokenKinds(t, expected, kinds)
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
	assertTokenKinds(t, expected, kinds)
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
	assertTokenKinds(t, expected, kinds)
}

func TestIdentifierWithUnderscore(t *testing.T) {
	// Per leniency philosophy, underscores should be accepted
	texts := tokenTexts("my_identifier SOME_TYPE")
	expectedTexts := []string{"my_identifier", "SOME_TYPE"}
	assertTexts(t, expectedTexts, texts)
}

func TestTrailingHyphenAccepted(t *testing.T) {
	lexer := New([]byte("bad-"), nil)
	tokens, diagnostics := lexer.Tokenize()

	// Should produce a valid token
	if tokens[0].Kind != TokLowercaseIdent {
		t.Errorf("expected TokLowercaseIdent, got %v", tokens[0].Kind)
	}

	// Should NOT produce diagnostics (lenient parsing)
	if len(diagnostics) != 0 {
		t.Errorf("expected no diagnostics, got %d", len(diagnostics))
	}
}

func TestLeadingZerosAccepted(t *testing.T) {
	lexer := New([]byte("007"), nil)
	tokens, diagnostics := lexer.Tokenize()

	// Should produce a valid number token
	if tokens[0].Kind != TokNumber {
		t.Errorf("expected TokNumber, got %v", tokens[0].Kind)
	}

	// Should NOT produce diagnostics (lenient parsing)
	if len(diagnostics) != 0 {
		t.Errorf("expected no diagnostics, got %d", len(diagnostics))
	}
}

func TestSpanTracking(t *testing.T) {
	source := []byte("BEGIN END")
	lexer := New(source, nil)
	tokens, _ := lexer.Tokenize()

	if tokens[0].Kind != TokKwBegin {
		t.Errorf("expected TokKwBegin, got %v", tokens[0].Kind)
	}
	if tokens[0].Span.Start != 0 || tokens[0].Span.End != 5 {
		t.Errorf("expected span [0,5), got [%d,%d)", tokens[0].Span.Start, tokens[0].Span.End)
	}

	if tokens[1].Kind != TokKwEnd {
		t.Errorf("expected TokKwEnd, got %v", tokens[1].Kind)
	}
	if tokens[1].Span.Start != 6 || tokens[1].Span.End != 9 {
		t.Errorf("expected span [6,9), got [%d,%d)", tokens[1].Span.Start, tokens[1].Span.End)
	}
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
	assertTokenKinds(t, expected, kinds)
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
	assertTokenKinds(t, expected, kinds)
}

func TestNonUtf8InString(t *testing.T) {
	// Latin-1 encoded "Vaclav" (common in vendor MIBs)
	// \xe1 is 'a' in Latin-1
	source := []byte("DESCRIPTION \"V\xe1clav\"")
	lexer := New(source, nil)
	tokens, diagnostics := lexer.Tokenize()

	// Should successfully tokenize despite non-UTF-8
	if tokens[0].Kind != TokKwDescription {
		t.Errorf("expected TokKwDescription, got %v", tokens[0].Kind)
	}
	if tokens[1].Kind != TokQuotedString {
		t.Errorf("expected TokQuotedString, got %v", tokens[1].Kind)
	}
	if tokens[2].Kind != TokEOF {
		t.Errorf("expected TokEOF, got %v", tokens[2].Kind)
	}

	// No errors - non-UTF-8 in strings is accepted silently
	errors := filterErrors(diagnostics)
	if len(errors) != 0 {
		t.Errorf("expected no errors, got %v", errors)
	}
}

func TestNonUtf8InComment(t *testing.T) {
	// Non-UTF-8 in comments should be skipped entirely
	source := []byte("BEGIN -- V\xe1clav --\nEND")
	lexer := New(source, nil)
	tokens, _ := lexer.Tokenize()

	if tokens[0].Kind != TokKwBegin {
		t.Errorf("expected TokKwBegin, got %v", tokens[0].Kind)
	}
	if tokens[1].Kind != TokKwEnd {
		t.Errorf("expected TokKwEnd, got %v", tokens[1].Kind)
	}
	if tokens[2].Kind != TokEOF {
		t.Errorf("expected TokEOF, got %v", tokens[2].Kind)
	}
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
	if len(tokens) != 4 {
		t.Fatalf("expected 4 tokens, got %d", len(tokens))
	}
	if tokens[0].Kind != TokLowercaseIdent {
		t.Errorf("expected TokLowercaseIdent, got %v", tokens[0].Kind)
	}
	text := source[tokens[0].Span.Start:tokens[0].Span.End]
	if text != "rfu-1plus1-tx-mhsb-" {
		t.Errorf("expected 'rfu-1plus1-tx-mhsb-', got '%s'", text)
	}

	if tokens[1].Kind != TokMinus {
		t.Errorf("expected TokMinus, got %v", tokens[1].Kind)
	}

	if tokens[2].Kind != TokLowercaseIdent {
		t.Errorf("expected TokLowercaseIdent, got %v", tokens[2].Kind)
	}
	text2 := source[tokens[2].Span.Start:tokens[2].Span.End]
	if text2 != "rx-sd" {
		t.Errorf("expected 'rx-sd', got '%s'", text2)
	}

	if tokens[3].Kind != TokEOF {
		t.Errorf("expected TokEOF, got %v", tokens[3].Kind)
	}
}

func TestDoubleHyphenSimple(t *testing.T) {
	// Simple case: foo--bar should become foo- + - + bar
	source := "foo--bar"
	lexer := New([]byte(source), nil)
	tokens, _ := lexer.Tokenize()

	if len(tokens) != 4 {
		t.Fatalf("expected 4 tokens, got %d", len(tokens))
	}
	if tokens[0].Kind != TokLowercaseIdent {
		t.Errorf("expected TokLowercaseIdent, got %v", tokens[0].Kind)
	}
	text := source[tokens[0].Span.Start:tokens[0].Span.End]
	if text != "foo-" {
		t.Errorf("expected 'foo-', got '%s'", text)
	}

	if tokens[1].Kind != TokMinus {
		t.Errorf("expected TokMinus, got %v", tokens[1].Kind)
	}

	if tokens[2].Kind != TokLowercaseIdent {
		t.Errorf("expected TokLowercaseIdent, got %v", tokens[2].Kind)
	}
	text2 := source[tokens[2].Span.Start:tokens[2].Span.End]
	if text2 != "bar" {
		t.Errorf("expected 'bar', got '%s'", text2)
	}
}

func TestDoubleHyphenAtStartIsComment(t *testing.T) {
	// --foo at start of input is a comment
	source := "--foo\nbar"
	lexer := New([]byte(source), nil)
	tokens, _ := lexer.Tokenize()

	// The -- starts a comment that runs to end of line
	if tokens[0].Kind != TokLowercaseIdent {
		t.Errorf("expected TokLowercaseIdent, got %v", tokens[0].Kind)
	}
	text := source[tokens[0].Span.Start:tokens[0].Span.End]
	if text != "bar" {
		t.Errorf("expected 'bar', got '%s'", text)
	}
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
	assertTokenKinds(t, expected, kinds)

	texts := tokenTexts(source)
	expectedTexts := []string{"if-index", "my-object"}
	assertTexts(t, expectedTexts, texts)
}

// ========================================================================
// Tests for forbidden keyword detection
// ========================================================================

func TestForbiddenKeywordFalse(t *testing.T) {
	source := "DEFVAL {FALSE}"
	lexer := New([]byte(source), nil)
	tokens, diagnostics := lexer.Tokenize()

	if tokens[0].Kind != TokKwDefval {
		t.Errorf("expected TokKwDefval, got %v", tokens[0].Kind)
	}
	if tokens[1].Kind != TokLBrace {
		t.Errorf("expected TokLBrace, got %v", tokens[1].Kind)
	}
	if tokens[2].Kind != TokForbiddenKeyword {
		t.Errorf("expected TokForbiddenKeyword, got %v", tokens[2].Kind)
	}
	if tokens[3].Kind != TokRBrace {
		t.Errorf("expected TokRBrace, got %v", tokens[3].Kind)
	}

	// Should NOT produce diagnostics (lenient parsing)
	if len(diagnostics) != 0 {
		t.Errorf("expected no diagnostics, got %d", len(diagnostics))
	}
}

func TestForbiddenKeywordTrue(t *testing.T) {
	source := "DEFVAL {TRUE}"
	lexer := New([]byte(source), nil)
	tokens, diagnostics := lexer.Tokenize()

	if tokens[2].Kind != TokForbiddenKeyword {
		t.Errorf("expected TokForbiddenKeyword, got %v", tokens[2].Kind)
	}

	// Should NOT produce diagnostics (lenient parsing)
	if len(diagnostics) != 0 {
		t.Errorf("expected no diagnostics, got %d", len(diagnostics))
	}
}

func TestForbiddenKeywordsList(t *testing.T) {
	forbidden := []string{
		"ABSENT", "ANY", "BIT", "BOOLEAN", "DEFAULT", "NULL", "PRIVATE", "SET",
	}

	for _, kw := range forbidden {
		lexer := New([]byte(kw), nil)
		tokens, diagnostics := lexer.Tokenize()

		if tokens[0].Kind != TokForbiddenKeyword {
			t.Errorf("%s should be a forbidden keyword, got %v", kw, tokens[0].Kind)
		}
		// Should NOT produce diagnostics (lenient parsing)
		if len(diagnostics) != 0 {
			t.Errorf("%s should not generate diagnostics, got %d", kw, len(diagnostics))
		}
	}
}

func TestMaxMinForbidden(t *testing.T) {
	// MAX and MIN are forbidden ASN.1 keywords
	source := "MAX MIN"
	lexer := New([]byte(source), nil)
	tokens, diagnostics := lexer.Tokenize()

	if tokens[0].Kind != TokForbiddenKeyword {
		t.Errorf("expected TokForbiddenKeyword for MAX, got %v", tokens[0].Kind)
	}
	if tokens[1].Kind != TokForbiddenKeyword {
		t.Errorf("expected TokForbiddenKeyword for MIN, got %v", tokens[1].Kind)
	}
	// Should NOT produce diagnostics (lenient parsing)
	if len(diagnostics) != 0 {
		t.Errorf("expected no diagnostics, got %d", len(diagnostics))
	}
}

func TestValidKeywordsNotForbidden(t *testing.T) {
	// Make sure valid SMI keywords are not mistakenly marked as forbidden
	valid := []string{"BEGIN", "END", "OBJECT", "INTEGER", "SYNTAX", "STATUS"}

	for _, kw := range valid {
		lexer := New([]byte(kw), nil)
		tokens, _ := lexer.Tokenize()

		if tokens[0].Kind == TokError {
			t.Errorf("%s should NOT be a forbidden keyword", kw)
		}
	}
}

func TestForbiddenKeywordCaseSensitive(t *testing.T) {
	// Forbidden keywords are case-sensitive (all uppercase)
	source := "false"
	lexer := New([]byte(source), nil)
	tokens, _ := lexer.Tokenize()

	if tokens[0].Kind != TokLowercaseIdent {
		t.Errorf("expected TokLowercaseIdent for 'false', got %v", tokens[0].Kind)
	}
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
	if len(tokens) != 3 {
		t.Fatalf("expected 3 tokens, got %d", len(tokens))
	}
	if tokens[0].Kind != TokLowercaseIdent {
		t.Errorf("expected TokLowercaseIdent, got %v", tokens[0].Kind)
	}
	if tokens[1].Kind != TokLowercaseIdent {
		t.Errorf("expected TokLowercaseIdent, got %v", tokens[1].Kind)
	}
	if tokens[2].Kind != TokEOF {
		t.Errorf("expected TokEOF, got %v", tokens[2].Kind)
	}

	// Should have one error diagnostic
	if len(diagnostics) != 1 {
		t.Errorf("expected 1 diagnostic, got %d", len(diagnostics))
	}
}

func TestErrorRecoveryMultipleBadCharsSameLine(t *testing.T) {
	// Multiple bad characters on same line should result in one skip
	source := []byte("foo \x80\x81\x82 ignored\nbar")
	lexer := New(source, nil)
	tokens, diagnostics := lexer.Tokenize()

	// Should skip from first bad char to EOL
	if len(tokens) != 3 {
		t.Fatalf("expected 3 tokens, got %d", len(tokens))
	}
	if tokens[0].Kind != TokLowercaseIdent {
		t.Errorf("expected TokLowercaseIdent, got %v", tokens[0].Kind)
	}
	if tokens[1].Kind != TokLowercaseIdent {
		t.Errorf("expected TokLowercaseIdent, got %v", tokens[1].Kind)
	}
	if len(diagnostics) != 1 {
		t.Errorf("expected 1 diagnostic, got %d", len(diagnostics))
	}
}

func TestErrorRecoveryPreservesSameLineBeforeError(t *testing.T) {
	// Tokens before the error on the same line should be preserved
	source := []byte("BEGIN foo \x80 ignored\nEND")
	lexer := New(source, nil)
	tokens, _ := lexer.Tokenize()

	if tokens[0].Kind != TokKwBegin {
		t.Errorf("expected TokKwBegin, got %v", tokens[0].Kind)
	}
	if tokens[1].Kind != TokLowercaseIdent { // foo
		t.Errorf("expected TokLowercaseIdent, got %v", tokens[1].Kind)
	}
	if tokens[2].Kind != TokKwEnd {
		t.Errorf("expected TokKwEnd, got %v", tokens[2].Kind)
	}
	if tokens[3].Kind != TokEOF {
		t.Errorf("expected TokEOF, got %v", tokens[3].Kind)
	}
}

func TestErrorAtEofNoInfiniteLoop(t *testing.T) {
	// Error at EOF should not cause issues
	source := []byte("foo\x80")
	lexer := New(source, nil)
	tokens, diagnostics := lexer.Tokenize()

	if len(tokens) != 2 {
		t.Fatalf("expected 2 tokens, got %d", len(tokens))
	}
	if tokens[0].Kind != TokLowercaseIdent {
		t.Errorf("expected TokLowercaseIdent, got %v", tokens[0].Kind)
	}
	if tokens[1].Kind != TokEOF {
		t.Errorf("expected TokEOF, got %v", tokens[1].Kind)
	}
	if len(diagnostics) != 1 {
		t.Errorf("expected 1 diagnostic, got %d", len(diagnostics))
	}
}

// ========================================================================
// Tests for odd-dashes comment handling (matching libsmi behavior)
// ========================================================================

func TestOddDashesThreeAtEOL(t *testing.T) {
	// Three dashes at end of line when in comment state should not emit MINUS
	source := "foo ---\nbar"
	lexer := New([]byte(source), nil)
	tokens, _ := lexer.Tokenize()

	if len(tokens) != 3 {
		t.Fatalf("expected 3 tokens, got %d: %v", len(tokens), tokens)
	}
	if tokens[0].Kind != TokLowercaseIdent {
		t.Errorf("expected TokLowercaseIdent, got %v", tokens[0].Kind)
	}
	if tokens[1].Kind != TokLowercaseIdent {
		t.Errorf("expected TokLowercaseIdent, got %v", tokens[1].Kind)
	}
	if tokens[2].Kind != TokEOF {
		t.Errorf("expected TokEOF, got %v", tokens[2].Kind)
	}
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
	if len(tokens) != 3 {
		t.Fatalf("expected 3 tokens, got %d", len(tokens))
	}
	if tokens[0].Kind != TokLowercaseIdent {
		t.Errorf("expected TokLowercaseIdent, got %v", tokens[0].Kind)
	}
	if tokens[1].Kind != TokLowercaseIdent {
		t.Errorf("expected TokLowercaseIdent, got %v", tokens[1].Kind)
	}
	if tokens[2].Kind != TokEOF {
		t.Errorf("expected TokEOF, got %v", tokens[2].Kind)
	}
}

func TestOddDashesFiveDashesAtEOL(t *testing.T) {
	// 5 dashes followed by EOL
	source := "foo -----\nbar"
	lexer := New([]byte(source), nil)
	tokens, _ := lexer.Tokenize()

	if len(tokens) != 3 {
		t.Fatalf("expected 3 tokens, got %d", len(tokens))
	}
	if tokens[0].Kind != TokLowercaseIdent { // foo
		t.Errorf("expected TokLowercaseIdent, got %v", tokens[0].Kind)
	}
	if tokens[1].Kind != TokLowercaseIdent { // bar
		t.Errorf("expected TokLowercaseIdent, got %v", tokens[1].Kind)
	}
	if tokens[2].Kind != TokEOF {
		t.Errorf("expected TokEOF, got %v", tokens[2].Kind)
	}
}

func TestOddDashesFiveNotAtEOL(t *testing.T) {
	// 5 dashes NOT followed by EOL: -- (enter) -- (exit) - (MINUS)
	source := "foo ----- bar"
	lexer := New([]byte(source), nil)
	tokens, _ := lexer.Tokenize()

	// foo, then 5 dashes: -- enters comment, -- exits, - becomes MINUS
	if tokens[0].Kind != TokLowercaseIdent { // foo
		t.Errorf("expected TokLowercaseIdent, got %v", tokens[0].Kind)
	}
	if tokens[1].Kind != TokMinus { // the 5th dash
		t.Errorf("expected TokMinus, got %v", tokens[1].Kind)
	}
	if tokens[2].Kind != TokLowercaseIdent { // bar
		t.Errorf("expected TokLowercaseIdent, got %v", tokens[2].Kind)
	}
	if tokens[3].Kind != TokEOF {
		t.Errorf("expected TokEOF, got %v", tokens[3].Kind)
	}
}

func TestOddDashesSevenDashes(t *testing.T) {
	// 7 dashes: --(enter) --(exit) --(enter) -(in comment) \n(end comment)
	source := "foo -------\nbar"
	lexer := New([]byte(source), nil)
	tokens, _ := lexer.Tokenize()

	// No MINUS emitted because the single dash is inside a comment
	if len(tokens) != 3 {
		t.Fatalf("expected 3 tokens, got %d", len(tokens))
	}
	if tokens[0].Kind != TokLowercaseIdent { // foo
		t.Errorf("expected TokLowercaseIdent, got %v", tokens[0].Kind)
	}
	if tokens[1].Kind != TokLowercaseIdent { // bar
		t.Errorf("expected TokLowercaseIdent, got %v", tokens[1].Kind)
	}
	if tokens[2].Kind != TokEOF {
		t.Errorf("expected TokEOF, got %v", tokens[2].Kind)
	}
}

func TestOddDashesNineAtEOL(t *testing.T) {
	// 9 dashes followed by EOL
	source := "foo ---------\nbar"
	lexer := New([]byte(source), nil)
	tokens, _ := lexer.Tokenize()

	if len(tokens) != 3 {
		t.Fatalf("expected 3 tokens, got %d", len(tokens))
	}
	if tokens[0].Kind != TokLowercaseIdent { // foo
		t.Errorf("expected TokLowercaseIdent, got %v", tokens[0].Kind)
	}
	if tokens[1].Kind != TokLowercaseIdent { // bar
		t.Errorf("expected TokLowercaseIdent, got %v", tokens[1].Kind)
	}
	if tokens[2].Kind != TokEOF {
		t.Errorf("expected TokEOF, got %v", tokens[2].Kind)
	}
}

// ========================================================================
// Keyword tests
// ========================================================================

func TestKeywordsSorted(t *testing.T) {
	// Verify the keyword table is sorted
	for i := 1; i < len(keywords); i++ {
		if keywords[i-1].text >= keywords[i].text {
			t.Errorf("Keywords not sorted: %s should come before %s",
				keywords[i-1].text, keywords[i].text)
		}
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
		if found != tc.found {
			t.Errorf("LookupKeyword(%q): expected found=%v, got %v", tc.text, tc.found, found)
		}
		if found && kind != tc.expected {
			t.Errorf("LookupKeyword(%q): expected %v, got %v", tc.text, tc.expected, kind)
		}
	}
}

func TestKeywordCaseSensitive(t *testing.T) {
	// Keywords are case-sensitive
	if kind, ok := LookupKeyword("OBJECT-TYPE"); !ok || kind != TokKwObjectType {
		t.Error("OBJECT-TYPE should be a keyword")
	}
	if _, ok := LookupKeyword("object-type"); ok {
		t.Error("object-type should NOT be a keyword")
	}
	if _, ok := LookupKeyword("Object-Type"); ok {
		t.Error("Object-Type should NOT be a keyword")
	}

	if kind, ok := LookupKeyword("Integer32"); !ok || kind != TokKwInteger32 {
		t.Error("Integer32 should be a keyword")
	}
	if _, ok := LookupKeyword("INTEGER32"); ok {
		t.Error("INTEGER32 should NOT be a keyword")
	}
	if _, ok := LookupKeyword("integer32"); ok {
		t.Error("integer32 should NOT be a keyword")
	}
}

func TestForbiddenKeywordsSorted(t *testing.T) {
	// Verify the forbidden keyword table is sorted
	for i := 1; i < len(forbiddenKeywords); i++ {
		if forbiddenKeywords[i-1] >= forbiddenKeywords[i] {
			t.Errorf("Forbidden keywords not sorted: %s should come before %s",
				forbiddenKeywords[i-1], forbiddenKeywords[i])
		}
	}
}

func TestForbiddenKeywordLookup(t *testing.T) {
	// Test some forbidden keywords
	forbidden := []string{"FALSE", "TRUE", "NULL", "ABSENT", "MAX", "MIN", "PRIVATE", "SET"}
	for _, kw := range forbidden {
		if !IsForbiddenKeyword(kw) {
			t.Errorf("%s should be a forbidden keyword", kw)
		}
	}

	// Test non-forbidden keywords
	notForbidden := []string{"BEGIN", "END", "OBJECT-TYPE", "INTEGER", "ifIndex"}
	for _, kw := range notForbidden {
		if IsForbiddenKeyword(kw) {
			t.Errorf("%s should NOT be a forbidden keyword", kw)
		}
	}

	// Test case sensitivity
	if !IsForbiddenKeyword("FALSE") {
		t.Error("FALSE should be a forbidden keyword")
	}
	if IsForbiddenKeyword("false") {
		t.Error("false should NOT be a forbidden keyword (case-sensitive)")
	}
	if IsForbiddenKeyword("False") {
		t.Error("False should NOT be a forbidden keyword (case-sensitive)")
	}
}

// ========================================================================
// Helper functions
// ========================================================================

func assertTokenKinds(t *testing.T, expected, actual []TokenKind) {
	t.Helper()
	if len(expected) != len(actual) {
		t.Errorf("expected %d tokens, got %d\nexpected: %v\nactual: %v",
			len(expected), len(actual), expected, actual)
		return
	}
	for i := range expected {
		if expected[i] != actual[i] {
			t.Errorf("token %d: expected %v, got %v", i, expected[i], actual[i])
		}
	}
}

func assertTexts(t *testing.T, expected, actual []string) {
	t.Helper()
	if len(expected) != len(actual) {
		t.Errorf("expected %d texts, got %d\nexpected: %v\nactual: %v",
			len(expected), len(actual), expected, actual)
		return
	}
	for i := range expected {
		if expected[i] != actual[i] {
			t.Errorf("text %d: expected %q, got %q", i, expected[i], actual[i])
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
