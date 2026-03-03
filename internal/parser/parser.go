// Package parser provides MIB parsing into an AST.
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

	"github.com/golangsnmp/gomib/internal/ast"
	"github.com/golangsnmp/gomib/internal/lexer"
	"github.com/golangsnmp/gomib/internal/types"
)

// Parser converts a token stream into an AST module with diagnostics.
type Parser struct {
	source      []byte
	lex         *lexer.Lexer
	buf         [3]lexer.Token   // lookahead buffer: buf[0]=current, buf[1]=peek(1), buf[2]=peek(2)
	lastEnd     types.ByteOffset // end position of last consumed token
	diagnostics []types.SpanDiagnostic
	diagConfig  types.DiagnosticConfig
	eofToken    lexer.Token
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
		source:     source,
		lex:        lex,
		diagConfig: diagConfig,
		eofToken:   eofToken,
		Logger:     types.Logger{L: logger},
	}
	p.buf[0] = lex.NextToken()
	p.buf[1] = lex.NextToken()
	p.buf[2] = lex.NextToken()
	p.Log(slog.LevelDebug, "parser initialized")
	return p
}

// emitDiagnostic records a diagnostic if the current config reports it.
func (p *Parser) emitDiagnostic(code string, span types.Span, message string) {
	sev := types.SeverityForCode(code)
	if !p.diagConfig.ShouldReport(code, sev) {
		return
	}
	p.diagnostics = append(p.diagnostics, types.SpanDiagnostic{
		Severity: sev,
		Code:     code,
		Span:     span,
		Message:  message,
	})
}

// validateIdentifier checks for RFC 2578 identifier violations
// (underscores, trailing hyphens, length limits).
func (p *Parser) validateIdentifier(name string, span types.Span) {
	if strings.Contains(name, "_") {
		p.emitDiagnostic(types.DiagIdentifierUnderscore, span,
			fmt.Sprintf("identifier %q contains underscore (RFC violation)", name))
	}

	if strings.HasSuffix(name, "-") {
		p.emitDiagnostic(types.DiagIdentifierHyphenEnd, span,
			fmt.Sprintf("identifier %q ends with hyphen", name))
	}

	if len(name) > 64 {
		p.emitDiagnostic(types.DiagIdentifierLength64, span,
			fmt.Sprintf("identifier %q exceeds 64 character limit (%d chars)", name, len(name)))
	} else if len(name) > 32 {
		p.emitDiagnostic(types.DiagIdentifierLength32, span,
			fmt.Sprintf("identifier %q exceeds 32 character recommendation (%d chars)", name, len(name)))
	}
}

// validateValueReference checks that a value reference starts with lowercase.
// Per RFC 2578, value references (used in OID assignments) should start with lowercase.
func (p *Parser) validateValueReference(name string, span types.Span) {
	if name != "" && name[0] >= 'A' && name[0] <= 'Z' {
		p.emitDiagnostic(types.DiagBadIdentifierCase, span,
			fmt.Sprintf("%q should start with a lowercase letter", name))
	}
}

// ParseModule parses all MIB modules in the source and returns their ASTs.
// A single file may contain multiple DEFINITIONS ::= BEGIN ... END blocks.
// Parse errors are collected in each module's diagnostics rather than
// causing immediate failure.
func (p *Parser) ParseModule() []*ast.Module {
	var modules []*ast.Module
	for !p.isEOF() {
		mod := p.parseOneModule()
		modules = append(modules, mod)
		if mod.Name.Name == "UNKNOWN" {
			// Header parse failed, no point continuing.
			break
		}
	}
	if len(modules) == 0 {
		// Empty or whitespace-only input: attempt a parse so we get a
		// diagnostic explaining why it failed.
		mod := p.parseOneModule()
		modules = append(modules, mod)
	}
	return modules
}

// parseOneModule parses a single DEFINITIONS ::= BEGIN ... END block.
func (p *Parser) parseOneModule() *ast.Module {
	// Snapshot lexer diagnostic count so we only attach new ones to this module.
	lexDiagsBefore := len(p.lex.Diagnostics())
	p.diagnostics = p.diagnostics[:0]

	start := p.currentSpan().Start

	name, err := p.parseModuleHeader()
	if err != nil {
		p.recordParseError(*err)
		p.Log(slog.LevelDebug, "failed to parse module header")
		span := types.NewSpan(start, p.currentSpan().End)
		return &ast.Module{
			Name:        ast.Ident{Name: "UNKNOWN", Span: span},
			Span:        span,
			Diagnostics: p.collectModuleDiagnostics(lexDiagsBefore),
		}
	}

	p.Log(slog.LevelDebug, "parsing module", slog.String("module", name.Name))

	module := ast.NewModule(name, types.NewSpan(start, 0))

	if p.check(lexer.TokKwImports) {
		imports, err := p.parseImports()
		if err != nil {
			p.recordParseError(*err)
			p.Log(slog.LevelDebug, "failed to parse imports", slog.String("module", name.Name))
		} else {
			module.Imports = imports
			p.Log(slog.LevelDebug, "parsed imports",
				slog.String("module", name.Name),
				slog.Int("count", len(imports)))
		}
	}

	for !p.check(lexer.TokKwEnd) && !p.isEOF() {
		posBefore := p.currentSpan().Start
		def, err := p.parseDefinition()
		if err != nil {
			p.recordParseError(*err)
			p.recoverToDefinition()
			// If recovery didn't advance past the error, force progress
			// to prevent an infinite loop when the current token pair
			// looks like a definition start but can't actually be parsed.
			if p.currentSpan().Start == posBefore {
				p.advance()
			}
		} else {
			module.Body = append(module.Body, def)
		}
	}

	if p.check(lexer.TokKwEnd) {
		p.advance()
	} else {
		p.recordParseError(p.makeError("expected END"))
	}

	module.Span = types.NewSpan(start, p.currentSpan().End)
	module.Diagnostics = p.collectModuleDiagnostics(lexDiagsBefore)

	p.Log(slog.LevelDebug, "parsing complete",
		slog.String("module", name.Name),
		slog.Int("definitions", len(module.Body)),
		slog.Int("diagnostics", len(p.diagnostics)))

	return module
}

// collectModuleDiagnostics builds the diagnostics slice for a single module
// by combining new lexer diagnostics (since lexDiagsBefore) with parser
// diagnostics accumulated for this module.
func (p *Parser) collectModuleDiagnostics(lexDiagsBefore int) []types.SpanDiagnostic {
	lexDiags := p.lex.Diagnostics()
	newLexDiags := lexDiags[lexDiagsBefore:]
	combined := make([]types.SpanDiagnostic, 0, len(newLexDiags)+len(p.diagnostics))
	combined = append(combined, newLexDiags...)
	combined = append(combined, p.diagnostics...)
	return combined
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
	p.buf[2] = p.lex.NextToken()
	p.lastEnd = tok.Span.End
	return tok
}

func (p *Parser) check(kind lexer.TokenKind) bool {
	return p.peek().Kind == kind
}

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

func (p *Parser) makeIdent(token lexer.Token) ast.Ident {
	return ast.Ident{Name: p.text(token.Span), Span: token.Span}
}

// makeIdentWithValidation creates an Ident and checks for RFC violations.
// Use for definition names, not type references.
func (p *Parser) makeIdentWithValidation(token lexer.Token) ast.Ident {
	name := p.text(token.Span)
	p.validateIdentifier(name, token.Span)
	return ast.Ident{Name: name, Span: token.Span}
}

// recordParseError appends a structural parse error unconditionally.
// Parse errors bypass ShouldReport() filtering because they indicate
// a syntax problem that must be reported at any strictness level.
func (p *Parser) recordParseError(diag types.SpanDiagnostic) {
	p.diagnostics = append(p.diagnostics, diag)
}

func (p *Parser) makeError(message string) types.SpanDiagnostic {
	return types.SpanDiagnostic{
		Severity: types.SeverityForCode(types.DiagParseError),
		Code:     types.DiagParseError,
		Span:     p.currentSpan(),
		Message:  message,
	}
}

func (p *Parser) parseU32(span types.Span, context string) uint32 {
	text := p.text(span)
	v, err := strconv.ParseUint(text, 10, 32)
	if err != nil {
		p.emitDiagnostic(types.DiagInvalidU32, span,
			fmt.Sprintf("invalid %s (not a valid u32)", context))
		return 0
	}
	return uint32(v)
}

func (p *Parser) parseI64(span types.Span, context string) int64 {
	text := p.text(span)
	v, err := strconv.ParseInt(text, 10, 64)
	if err != nil {
		p.emitDiagnostic(types.DiagInvalidI64, span,
			fmt.Sprintf("invalid %s (not a valid integer)", context))
		return 0
	}
	return v
}

// parseModuleHeader parses: ModuleName [{ oid }] DEFINITIONS ::= BEGIN
func (p *Parser) parseModuleHeader() (ast.Ident, *types.SpanDiagnostic) {
	nameToken, err := p.expectIdentifier()
	if err != nil {
		return ast.Ident{}, err
	}
	name := p.makeIdentWithValidation(nameToken)

	// Skip obsolete module OID that some MIBs include before DEFINITIONS
	if p.check(lexer.TokLBrace) {
		p.advance() // consume opening brace
		p.skipBracedContent(true)
	}

	_, err = p.expect(lexer.TokKwDefinitions)
	if err != nil {
		return ast.Ident{}, err
	}

	_, err = p.expect(lexer.TokColonColonEqual)
	if err != nil {
		return ast.Ident{}, err
	}

	_, err = p.expect(lexer.TokKwBegin)
	if err != nil {
		return ast.Ident{}, err
	}

	return name, nil
}

func (p *Parser) expectIdentifier() (lexer.Token, *types.SpanDiagnostic) {
	if p.check(lexer.TokUppercaseIdent) || p.check(lexer.TokLowercaseIdent) {
		return p.advance(), nil
	}
	// Accept forbidden keywords as identifiers for lenient parsing
	if p.check(lexer.TokForbiddenKeyword) {
		token := p.advance()
		name := p.text(token.Span)
		p.emitDiagnostic(types.DiagKeywordReserved, token.Span,
			fmt.Sprintf("identifier %q is a reserved ASN.1 keyword", name))
		return token, nil
	}
	diag := p.makeError("expected identifier")
	return lexer.Token{}, &diag
}

// expectIndexObject expects an identifier or bare type keyword.
// Type keywords are accepted because vendor MIBs use them as index objects.
func (p *Parser) expectIndexObject() (lexer.Token, *types.SpanDiagnostic) {
	kind := p.peek().Kind
	if kind.IsIdentifier() || kind.IsTypeKeyword() {
		return p.advance(), nil
	}
	diag := p.makeError("expected index object")
	return lexer.Token{}, &diag
}

// expectEnumLabel expects an identifier or keyword usable as an enum label.
// Any keyword can appear as an enum label in real MIBs (current, deprecated,
// mandatory, optional, object, module, etc.).
func (p *Parser) expectEnumLabel() (lexer.Token, *types.SpanDiagnostic) {
	kind := p.peek().Kind
	if kind.IsIdentifier() || kind.IsKeyword() {
		return p.advance(), nil
	}
	diag := p.makeError("expected enum label")
	return lexer.Token{}, &diag
}

// parseImports parses: IMPORTS symbols FROM Module ... ;
func (p *Parser) parseImports() ([]ast.ImportClause, *types.SpanDiagnostic) {
	_, err := p.expect(lexer.TokKwImports)
	if err != nil {
		return nil, err
	}

	var imports []ast.ImportClause

	for {
		if p.check(lexer.TokSemicolon) {
			p.advance()
			break
		}

		if p.isEOF() || p.check(lexer.TokKwEnd) {
			diag := p.makeError("unexpected end of imports")
			return imports, &diag
		}

		start := p.currentSpan().Start
		var symbols []ast.Ident

	symbolLoop:
		for {
			kind := p.peek().Kind
			switch {
			case kind.IsMacroKeyword() || kind.IsTypeKeyword() || kind.IsIdentifier():
				symToken := p.advance()
				symbols = append(symbols, p.makeIdent(symToken))
			case p.check(lexer.TokKwFrom):
				break symbolLoop
			default:
				diag := p.makeError("expected symbol or FROM")
				return imports, &diag
			}

			if p.check(lexer.TokComma) {
				p.advance()
			}
		}

		_, err := p.expect(lexer.TokKwFrom)
		if err != nil {
			return imports, err
		}

		if !p.check(lexer.TokUppercaseIdent) {
			diag := p.makeError("expected module name after FROM")
			return imports, &diag
		}
		moduleToken := p.advance()
		fromModule := p.makeIdent(moduleToken)
		span := types.NewSpan(start, moduleToken.Span.End)

		imports = append(imports, ast.ImportClause{
			Symbols:    symbols,
			FromModule: fromModule,
			Span:       span,
		})
	}

	return imports, nil
}

// parseDefinition dispatches to the appropriate definition parser
// based on lookahead tokens.
func (p *Parser) parseDefinition() (ast.Definition, *types.SpanDiagnostic) {
	first := p.peek().Kind
	second := p.peekNth(1).Kind

	p.Trace("parsing definition",
		slog.Int("offset", int(p.currentSpan().Start)),
		slog.String("first", first.LibsmiName()),
		slog.String("second", second.LibsmiName()))

	switch {
	// Value assignment: name OBJECT IDENTIFIER ::=
	// Accept both lowercase (RFC-compliant) and uppercase (vendor violation) identifiers
	case first.IsIdentifier() && second == lexer.TokKwObject && p.peekNth(2).Kind == lexer.TokKwIdentifier:
		return p.parseValueAssignment()

	// OBJECT-TYPE
	case first.IsIdentifier() && second == lexer.TokKwObjectType:
		return p.parseObjectType()

	// MODULE-IDENTITY
	case first.IsIdentifier() && second == lexer.TokKwModuleIdentity:
		return p.parseModuleIdentity()

	// OBJECT-IDENTITY
	case first.IsIdentifier() && second == lexer.TokKwObjectIdentity:
		return p.parseObjectIdentity()

	// NOTIFICATION-TYPE
	case first.IsIdentifier() && second == lexer.TokKwNotificationType:
		return p.parseNotificationType()

	// TRAP-TYPE
	case first.IsIdentifier() && second == lexer.TokKwTrapType:
		return p.parseTrapType()

	// TEXTUAL-CONVENTION
	case first == lexer.TokUppercaseIdent && second == lexer.TokKwTextualConvention:
		return p.parseTextualConvention()

	// OBJECT-GROUP
	case first.IsIdentifier() && second == lexer.TokKwObjectGroup:
		return p.parseObjectGroup()

	// NOTIFICATION-GROUP
	case first.IsIdentifier() && second == lexer.TokKwNotificationGroup:
		return p.parseNotificationGroup()

	// MODULE-COMPLIANCE
	case first.IsIdentifier() && second == lexer.TokKwModuleCompliance:
		return p.parseModuleCompliance()

	// AGENT-CAPABILITIES
	case first.IsIdentifier() && second == lexer.TokKwAgentCapabilities:
		return p.parseAgentCapabilities()

	// Type assignment or TEXTUAL-CONVENTION: TypeName ::=
	// Accept lowercase type names (vendor violation) with diagnostic
	case (first == lexer.TokUppercaseIdent || first == lexer.TokLowercaseIdent) && second == lexer.TokColonColonEqual:
		// Check if this is a TC: TypeName ::= TEXTUAL-CONVENTION
		if p.peekNth(2).Kind == lexer.TokKwTextualConvention {
			return p.parseTextualConventionWithAssignment()
		}
		if first == lexer.TokLowercaseIdent {
			name := p.text(p.peek().Span)
			p.emitDiagnostic(types.DiagBadIdentifierCase, p.peek().Span,
				fmt.Sprintf("type assignment %q should start with an uppercase letter", name))
		}
		return p.parseTypeAssignment()

	// MACRO definition
	case first == lexer.TokUppercaseIdent && second == lexer.TokKwMacro:
		return p.parseMacroDefinition()

	// EXPORTS (skipped by lexer)
	case first == lexer.TokKwExports:
		p.advance() // EXPORTS
		if p.check(lexer.TokSemicolon) {
			p.advance()
		}
		return p.parseDefinition()

	default:
		diag := p.makeError("unexpected token: " + p.peek().Kind.LibsmiName())
		return nil, &diag
	}
}

// parseValueAssignment parses: name OBJECT IDENTIFIER ::= { ... }
func (p *Parser) parseValueAssignment() (ast.Definition, *types.SpanDiagnostic) {
	start := p.currentSpan().Start

	nameToken := p.advance()
	name := p.makeIdentWithValidation(nameToken)

	// Value references should start with lowercase per RFC 2578
	p.validateValueReference(name.Name, nameToken.Span)

	if _, err := p.expect(lexer.TokKwObject); err != nil {
		return nil, err
	}
	if _, err := p.expect(lexer.TokKwIdentifier); err != nil {
		return nil, err
	}
	if _, err := p.expect(lexer.TokColonColonEqual); err != nil {
		return nil, err
	}

	oid, err := p.parseOidAssignment()
	if err != nil {
		return nil, err
	}

	span := types.NewSpan(start, oid.Span.End)
	return &ast.ValueAssignmentDef{
		DefBase:       ast.DefBase{Name: name, Span: span},
		OidAssignment: oid,
	}, nil
}

// parseOidAssignment parses: { parent subid ... }
func (p *Parser) parseOidAssignment() (ast.OidAssignment, *types.SpanDiagnostic) {
	start := p.currentSpan().Start
	if _, err := p.expect(lexer.TokLBrace); err != nil {
		return ast.OidAssignment{}, err
	}

	var components []ast.OidComponent

	for !p.check(lexer.TokRBrace) && !p.isEOF() {
		compStart := p.currentSpan().Start

		if p.check(lexer.TokNumber) || p.check(lexer.TokLowercaseIdent) || p.check(lexer.TokUppercaseIdent) {
			comp, err := p.parseOidComponent(compStart)
			if err != nil {
				return ast.OidAssignment{}, err
			}
			components = append(components, comp)
		} else {
			diag := p.makeError("expected OID component")
			return ast.OidAssignment{}, &diag
		}
	}

	endToken, err := p.expect(lexer.TokRBrace)
	if err != nil {
		return ast.OidAssignment{}, err
	}
	return ast.OidAssignment{Components: components, Span: types.NewSpan(start, endToken.Span.End)}, nil
}

// parseOidComponent parses a single OID component: a number, name, name(number),
// or a qualified reference (Module.name or Module.name(number)).
// The caller must have verified the current token is a Number or identifier.
func (p *Parser) parseOidComponent(compStart types.ByteOffset) (ast.OidComponent, *types.SpanDiagnostic) {
	if p.check(lexer.TokNumber) {
		token := p.advance()
		value := p.parseU32(token.Span, "OID component")
		return &ast.OidComponentNumber{Value: value, Span: token.Span}, nil
	}

	firstToken := p.advance()
	firstName := p.makeIdent(firstToken)

	if p.check(lexer.TokDot) {
		// Qualified reference: Module.name or Module.name(number)
		p.advance() // consume dot

		nameToken, err := p.expect(lexer.TokLowercaseIdent)
		if err != nil {
			return nil, err
		}
		qname := p.makeIdent(nameToken)

		if p.check(lexer.TokLParen) {
			// Qualified named number, e.g. Module.name(123)
			p.advance() // (
			numToken, err := p.expect(lexer.TokNumber)
			if err != nil {
				return nil, err
			}
			number := p.parseU32(numToken.Span, "OID component")
			endToken, err := p.expect(lexer.TokRParen)
			if err != nil {
				return nil, err
			}
			return &ast.OidComponentQualifiedNamedNumber{
				ModuleName: firstName,
				Name:       qname,
				Num:        number,
				Span:       types.NewSpan(compStart, endToken.Span.End),
			}, nil
		}
		// QualifiedName: Module.name
		return &ast.OidComponentQualifiedName{
			ModuleName: firstName,
			Name:       qname,
			Span:       types.NewSpan(compStart, nameToken.Span.End),
		}, nil
	}

	if p.check(lexer.TokLParen) {
		// Named number: iso(1), org(3)
		p.advance() // (
		numToken, err := p.expect(lexer.TokNumber)
		if err != nil {
			return nil, err
		}
		number := p.parseU32(numToken.Span, "OID component")
		endToken, err := p.expect(lexer.TokRParen)
		if err != nil {
			return nil, err
		}
		return &ast.OidComponentNamedNumber{
			Name: firstName,
			Num:  number,
			Span: types.NewSpan(compStart, endToken.Span.End),
		}, nil
	}

	// Just name: internet, ifEntry
	return &ast.OidComponentName{Name: firstName}, nil
}

// parseObjectType parses an OBJECT-TYPE macro invocation.
func (p *Parser) parseObjectType() (ast.Definition, *types.SpanDiagnostic) {
	start := p.currentSpan().Start

	nameToken := p.advance()
	name := p.makeIdentWithValidation(nameToken)
	p.validateValueReference(name.Name, nameToken.Span)

	if _, err := p.expect(lexer.TokKwObjectType); err != nil {
		return nil, err
	}

	// SYNTAX clause (required)
	if _, err := p.expect(lexer.TokKwSyntax); err != nil {
		return nil, err
	}
	syntax, err := p.parseSyntaxClause()
	if err != nil {
		return nil, err
	}

	// Optional UNITS
	var units *ast.QuotedString
	if p.check(lexer.TokKwUnits) {
		p.advance()
		qs, err := p.parseQuotedString()
		if err != nil {
			return nil, err
		}
		units = &qs
	}

	// MAX-ACCESS or ACCESS (required)
	access, err := p.parseAccessClause()
	if err != nil {
		return nil, err
	}

	// STATUS (technically required but some vendor MIBs omit it)
	var status *ast.StatusClause
	if p.check(lexer.TokKwStatus) {
		sc, err := p.parseStatusClause()
		if err != nil {
			return nil, err
		}
		status = &sc
	}

	// DESCRIPTION (optional but common)
	var description *ast.QuotedString
	if p.check(lexer.TokKwDescription) {
		p.advance()
		qs, err := p.parseQuotedString()
		if err != nil {
			return nil, err
		}
		description = &qs
	}

	// REFERENCE (optional)
	reference, err := p.parseOptionalReference()
	if err != nil {
		return nil, err
	}

	// INDEX or AUGMENTS (optional)
	index, augments, err := p.parseIndexOrAugments()
	if err != nil {
		return nil, err
	}

	// DEFVAL (optional)
	var defval *ast.DefValClause
	if p.check(lexer.TokKwDefval) {
		dv, err := p.parseDefValClause()
		if err != nil {
			return nil, err
		}
		defval = &dv
	}

	// ::= { oid }
	if _, err := p.expect(lexer.TokColonColonEqual); err != nil {
		return nil, err
	}
	oid, err := p.parseOidAssignment()
	if err != nil {
		return nil, err
	}

	span := types.NewSpan(start, oid.Span.End)
	return &ast.ObjectTypeDef{
		DefBase:       ast.DefBase{Name: name, Span: span},
		Syntax:        syntax,
		Units:         units,
		Access:        access,
		Status:        status,
		Description:   description,
		Reference:     reference,
		Index:         index,
		Augments:      augments,
		DefVal:        defval,
		OidAssignment: oid,
	}, nil
}

// parseSyntaxClause parses the type expression following a SYNTAX keyword.
func (p *Parser) parseSyntaxClause() (ast.SyntaxClause, *types.SpanDiagnostic) {
	start := p.currentSpan().Start
	syntax, err := p.parseTypeSyntax()
	if err != nil {
		return ast.SyntaxClause{}, err
	}
	span := types.NewSpan(start, syntax.SyntaxSpan().End)
	return ast.NewSyntaxClause(syntax, span), nil
}

// parseTypeSyntax parses a type expression (builtin types, type
// references, constrained types, SEQUENCE, CHOICE, etc.).
func (p *Parser) parseTypeSyntax() (ast.TypeSyntax, *types.SpanDiagnostic) {
	start := p.currentSpan().Start

	var baseSyntax ast.TypeSyntax

	switch p.peek().Kind {
	case lexer.TokKwInteger:
		p.advance()
		// Check for enum: INTEGER { ... }
		if p.check(lexer.TokLBrace) {
			namedNumbers, err := p.parseNamedNumbers()
			if err != nil {
				return nil, err
			}
			span := types.NewSpan(start, p.lastEnd)
			baseSyntax = &ast.TypeSyntaxIntegerEnum{
				Base:         nil,
				NamedNumbers: namedNumbers,
				Span:         span,
			}
		} else {
			baseSyntax = &ast.TypeSyntaxTypeRef{
				Name: ast.Ident{Name: "INTEGER", Span: types.NewSpan(start, p.lastEnd)},
			}
		}

	case lexer.TokKwBits:
		p.advance()
		if p.check(lexer.TokLBrace) {
			p.advance()
			namedBits, err := p.parseNamedNumberList()
			if err != nil {
				return nil, err
			}
			if _, err := p.expect(lexer.TokRBrace); err != nil {
				return nil, err
			}
			span := types.NewSpan(start, p.lastEnd)
			baseSyntax = &ast.TypeSyntaxBits{
				NamedBits: namedBits,
				Span:      span,
			}
		} else {
			baseSyntax = &ast.TypeSyntaxTypeRef{
				Name: ast.Ident{Name: "BITS", Span: types.NewSpan(start, p.lastEnd)},
			}
		}

	case lexer.TokKwOctet:
		p.advance()
		if _, err := p.expect(lexer.TokKwString); err != nil {
			return nil, err
		}
		span := types.NewSpan(start, p.lastEnd)
		if p.check(lexer.TokLParen) {
			constraint, err := p.parseConstraint()
			if err != nil {
				return nil, err
			}
			baseSyntax = &ast.TypeSyntaxConstrained{
				Base:       &ast.TypeSyntaxOctetString{Span: span},
				Constraint: constraint,
				Span:       types.NewSpan(start, constraint.ConstraintSpan().End),
			}
		} else {
			baseSyntax = &ast.TypeSyntaxOctetString{Span: span}
		}

	case lexer.TokKwObject:
		p.advance()
		if _, err := p.expect(lexer.TokKwIdentifier); err != nil {
			return nil, err
		}
		span := types.NewSpan(start, p.lastEnd)
		baseSyntax = &ast.TypeSyntaxObjectIdentifier{Span: span}

	case lexer.TokKwSequence:
		p.advance()
		if p.check(lexer.TokKwOf) {
			// SEQUENCE OF EntryType
			p.advance()
			entryToken, err := p.expectIdentifier()
			if err != nil {
				return nil, err
			}
			entryType := p.makeIdent(entryToken)
			span := types.NewSpan(start, entryToken.Span.End)
			baseSyntax = &ast.TypeSyntaxSequenceOf{
				EntryType: entryType,
				Span:      span,
			}
		} else {
			// SEQUENCE { ... }
			if _, err := p.expect(lexer.TokLBrace); err != nil {
				return nil, err
			}
			fields, err := p.parseSequenceFields()
			if err != nil {
				return nil, err
			}
			endToken, err := p.expect(lexer.TokRBrace)
			if err != nil {
				return nil, err
			}
			span := types.NewSpan(start, endToken.Span.End)
			baseSyntax = &ast.TypeSyntaxSequence{
				Fields: fields,
				Span:   span,
			}
		}

	case lexer.TokKwChoice:
		p.advance()
		if _, err := p.expect(lexer.TokLBrace); err != nil {
			return nil, err
		}
		alternatives, err := p.parseChoiceAlternatives()
		if err != nil {
			return nil, err
		}
		endToken, err := p.expect(lexer.TokRBrace)
		if err != nil {
			return nil, err
		}
		span := types.NewSpan(start, endToken.Span.End)
		baseSyntax = &ast.TypeSyntaxChoice{
			Alternatives: alternatives,
			Span:         span,
		}

	case lexer.TokKwCounter32, lexer.TokKwCounter64, lexer.TokKwGauge32,
		lexer.TokKwUnsigned32, lexer.TokKwTimeTicks, lexer.TokKwIpAddress,
		lexer.TokKwOpaque, lexer.TokKwCounter, lexer.TokKwGauge, lexer.TokKwNetworkAddress:
		token := p.advance()
		name := p.text(token.Span)
		baseSyntax = &ast.TypeSyntaxTypeRef{
			Name: ast.Ident{Name: name, Span: token.Span},
		}

	case lexer.TokUppercaseIdent, lexer.TokLowercaseIdent:
		token := p.advance()
		name := p.text(token.Span)
		if token.Kind == lexer.TokLowercaseIdent {
			p.emitDiagnostic(types.DiagBadIdentifierCase, token.Span,
				fmt.Sprintf("type reference %q should start with an uppercase letter", name))
		}
		ident := ast.Ident{Name: name, Span: token.Span}

		switch {
		case p.check(lexer.TokLParen):
			constraint, err := p.parseConstraint()
			if err != nil {
				return nil, err
			}
			span := types.NewSpan(start, constraint.ConstraintSpan().End)
			baseSyntax = &ast.TypeSyntaxConstrained{
				Base:       &ast.TypeSyntaxTypeRef{Name: ident},
				Constraint: constraint,
				Span:       span,
			}
		case p.check(lexer.TokLBrace):
			// Enum value restriction: TypeRef { value1(1), value2(2) }
			namedNumbers, err := p.parseNamedNumbers()
			if err != nil {
				return nil, err
			}
			span := types.NewSpan(start, p.lastEnd)
			baseSyntax = &ast.TypeSyntaxIntegerEnum{
				Base:         &ident,
				NamedNumbers: namedNumbers,
				Span:         span,
			}
		default:
			baseSyntax = &ast.TypeSyntaxTypeRef{Name: ident}
		}

	default:
		diag := p.makeError("expected type syntax")
		return nil, &diag
	}

	// Check for constraint on the base syntax
	if p.check(lexer.TokLParen) {
		if _, ok := baseSyntax.(*ast.TypeSyntaxConstrained); !ok {
			constraint, err := p.parseConstraint()
			if err != nil {
				return nil, err
			}
			span := types.NewSpan(start, constraint.ConstraintSpan().End)
			return &ast.TypeSyntaxConstrained{
				Base:       baseSyntax,
				Constraint: constraint,
				Span:       span,
			}, nil
		}
	}

	return baseSyntax, nil
}

// parseNamedNumbers parses: { name(value), ... }
func (p *Parser) parseNamedNumbers() ([]ast.NamedNumber, *types.SpanDiagnostic) {
	if _, err := p.expect(lexer.TokLBrace); err != nil {
		return nil, err
	}
	result, err := p.parseNamedNumberList()
	if err != nil {
		return nil, err
	}
	if _, err := p.expect(lexer.TokRBrace); err != nil {
		return nil, err
	}
	return result, nil
}

// parseNamedNumberList parses a list of named numbers (without braces).
func (p *Parser) parseNamedNumberList() ([]ast.NamedNumber, *types.SpanDiagnostic) {
	var namedNumbers []ast.NamedNumber

	for !p.check(lexer.TokRBrace) && !p.isEOF() {
		start := p.currentSpan().Start
		nameToken, err := p.expectEnumLabel()
		if err != nil {
			return nil, err
		}
		name := p.makeIdent(nameToken)

		if _, err := p.expect(lexer.TokLParen); err != nil {
			return nil, err
		}

		isNegative := p.check(lexer.TokNegativeNumber)
		var numToken lexer.Token
		if isNegative {
			numToken = p.advance()
		} else {
			numToken, err = p.expect(lexer.TokNumber)
			if err != nil {
				return nil, err
			}
		}
		value := p.parseI64(numToken.Span, "named number value")

		endToken, err := p.expect(lexer.TokRParen)
		if err != nil {
			return nil, err
		}
		span := types.NewSpan(start, endToken.Span.End)

		namedNumbers = append(namedNumbers, ast.NamedNumber{Name: name, Value: value, Span: span})

		if p.check(lexer.TokComma) {
			p.advance()
		} else {
			break
		}
	}

	return namedNumbers, nil
}

// parseConstraint parses: (SIZE (0..255)) or (0..65535)
func (p *Parser) parseConstraint() (ast.Constraint, *types.SpanDiagnostic) {
	start := p.currentSpan().Start
	if _, err := p.expect(lexer.TokLParen); err != nil {
		return nil, err
	}

	if p.check(lexer.TokKwSize) {
		p.advance()
		if _, err := p.expect(lexer.TokLParen); err != nil {
			return nil, err
		}
		ranges, err := p.parseRangeList()
		if err != nil {
			return nil, err
		}
		if _, err := p.expect(lexer.TokRParen); err != nil {
			return nil, err
		}
		endToken, err := p.expect(lexer.TokRParen)
		if err != nil {
			return nil, err
		}
		return &ast.ConstraintSize{
			Ranges: ranges,
			Span:   types.NewSpan(start, endToken.Span.End),
		}, nil
	}

	ranges, err := p.parseRangeList()
	if err != nil {
		return nil, err
	}
	endToken, err := p.expect(lexer.TokRParen)
	if err != nil {
		return nil, err
	}
	return &ast.ConstraintRange{
		Ranges: ranges,
		Span:   types.NewSpan(start, endToken.Span.End),
	}, nil
}

// parseRangeList parses: 0..255 | 1024..65535
func (p *Parser) parseRangeList() ([]ast.Range, *types.SpanDiagnostic) {
	var ranges []ast.Range

	for {
		start := p.currentSpan().Start
		lo, err := p.parseRangeValue()
		if err != nil {
			return nil, err
		}

		var hi ast.RangeValue
		if p.check(lexer.TokDotDot) {
			p.advance()
			hi, err = p.parseRangeValue()
			if err != nil {
				return nil, err
			}
		}

		ranges = append(ranges, ast.Range{
			Min:  lo,
			Max:  hi,
			Span: types.NewSpan(start, p.lastEnd),
		})

		if p.check(lexer.TokPipe) {
			p.advance()
		} else {
			break
		}
	}

	return ranges, nil
}

// parseRangeValue parses a single range endpoint (number, hex, or
// identifier like MIN/MAX).
func (p *Parser) parseRangeValue() (ast.RangeValue, *types.SpanDiagnostic) {
	switch {
	case p.check(lexer.TokNumber):
		token := p.advance()
		text := p.text(token.Span)
		// Try parsing as u64 first to handle large unsigned values
		if value, err := strconv.ParseUint(text, 10, 64); err == nil {
			return &ast.RangeValueUnsigned{Value: value}, nil
		}
		// Fallback to signed
		value := p.parseI64(token.Span, "range value")
		return &ast.RangeValueSigned{Value: value}, nil
	case p.check(lexer.TokNegativeNumber):
		token := p.advance()
		value := p.parseI64(token.Span, "range value")
		return &ast.RangeValueSigned{Value: value}, nil
	case p.check(lexer.TokHexString):
		token := p.advance()
		text := p.text(token.Span)
		// Parse hex string to unsigned
		hexPart := stripQuotedLiteral(text)
		value, err := strconv.ParseUint(hexPart, 16, 64)
		if err != nil {
			p.emitDiagnostic(types.DiagInvalidHexRange, token.Span, "invalid hex value in range")
		}
		return &ast.RangeValueUnsigned{Value: value}, nil
	case p.check(lexer.TokUppercaseIdent) || p.check(lexer.TokForbiddenKeyword):
		token := p.advance()
		name := p.text(token.Span)
		return &ast.RangeValueIdent{Name: ast.Ident{Name: name, Span: token.Span}}, nil
	default:
		diag := p.makeError("expected range value")
		return nil, &diag
	}
}

// parseSequenceFields parses comma-separated name/type pairs within
// SEQUENCE { ... }.
func (p *Parser) parseSequenceFields() ([]ast.SequenceField, *types.SpanDiagnostic) {
	var fields []ast.SequenceField

	for !p.check(lexer.TokRBrace) && !p.isEOF() {
		start := p.currentSpan().Start
		nameToken, err := p.expectIdentifier()
		if err != nil {
			return nil, err
		}
		name := p.makeIdent(nameToken)

		syntax, err := p.parseTypeSyntax()
		if err != nil {
			return nil, err
		}
		span := types.NewSpan(start, syntax.SyntaxSpan().End)

		fields = append(fields, ast.SequenceField{
			Name:   name,
			Syntax: syntax,
			Span:   span,
		})

		if p.check(lexer.TokComma) {
			p.advance()
		}
	}

	return fields, nil
}

// parseChoiceAlternatives parses comma-separated name/type pairs within
// CHOICE { ... }. Identical structure to SEQUENCE fields.
func (p *Parser) parseChoiceAlternatives() ([]ast.SequenceField, *types.SpanDiagnostic) {
	return p.parseSequenceFields()
}

// parseAccessClause parses ACCESS, MAX-ACCESS, or MIN-ACCESS with its value.
func (p *Parser) parseAccessClause() (ast.AccessClause, *types.SpanDiagnostic) {
	start := p.currentSpan().Start

	var keyword types.AccessKeyword
	switch {
	case p.check(lexer.TokKwMaxAccess):
		p.advance()
		keyword = types.AccessKeywordMaxAccess
	case p.check(lexer.TokKwAccess):
		p.advance()
		keyword = types.AccessKeywordAccess
	case p.check(lexer.TokKwMinAccess):
		p.advance()
		keyword = types.AccessKeywordMinAccess
	default:
		diag := p.makeError("expected MAX-ACCESS, MIN-ACCESS, or ACCESS")
		return ast.AccessClause{}, &diag
	}

	var value types.Access
	switch p.peek().Kind {
	case lexer.TokKwReadOnly:
		p.advance()
		value = types.AccessReadOnly
	case lexer.TokKwReadWrite:
		p.advance()
		value = types.AccessReadWrite
	case lexer.TokKwReadCreate:
		p.advance()
		value = types.AccessReadCreate
	case lexer.TokKwNotAccessible:
		p.advance()
		value = types.AccessNotAccessible
	case lexer.TokKwAccessibleForNotify:
		p.advance()
		value = types.AccessAccessibleForNotify
	case lexer.TokKwWriteOnly:
		p.advance()
		value = types.AccessWriteOnly
	case lexer.TokKwNotImplemented:
		p.advance()
		value = types.AccessNotImplemented
	default:
		diag := p.makeError("expected access value")
		return ast.AccessClause{}, &diag
	}

	span := types.NewSpan(start, p.lastEnd)
	return ast.AccessClause{
		Keyword: keyword,
		Value:   value,
		Span:    span,
	}, nil
}

// parseStatusClause parses STATUS with its value keyword.
func (p *Parser) parseStatusClause() (ast.StatusClause, *types.SpanDiagnostic) {
	start := p.currentSpan().Start
	if _, err := p.expect(lexer.TokKwStatus); err != nil {
		return ast.StatusClause{}, err
	}

	var value types.Status
	switch p.peek().Kind {
	case lexer.TokKwCurrent:
		p.advance()
		value = types.StatusCurrent
	case lexer.TokKwDeprecated:
		p.advance()
		value = types.StatusDeprecated
	case lexer.TokKwObsolete:
		p.advance()
		value = types.StatusObsolete
	case lexer.TokKwMandatory:
		p.advance()
		value = types.StatusMandatory
	case lexer.TokKwOptional:
		p.advance()
		value = types.StatusOptional
	default:
		diag := p.makeError("expected status value")
		return ast.StatusClause{}, &diag
	}

	span := types.NewSpan(start, p.lastEnd)
	return ast.StatusClause{Value: value, Span: span}, nil
}

// parseIndexOrAugments parses an optional INDEX or AUGMENTS clause.
// Returns nil for both if neither is present.
func (p *Parser) parseIndexOrAugments() (ast.IndexClause, *ast.AugmentsClause, *types.SpanDiagnostic) {
	if p.check(lexer.TokKwIndex) {
		start := p.currentSpan().Start
		p.advance()
		if _, err := p.expect(lexer.TokLBrace); err != nil {
			return nil, nil, err
		}

		var indexes []ast.IndexItem
		for !p.check(lexer.TokRBrace) && !p.isEOF() {
			itemStart := p.currentSpan().Start
			implied := false
			if p.check(lexer.TokKwImplied) {
				p.advance()
				implied = true
			}

			objToken, err := p.expectIndexObject()
			if err != nil {
				return nil, nil, err
			}

			// Combine "OCTET" + "STRING" into a single "OCTET STRING" index item.
			// Vendor MIBs sometimes use bare type names in INDEX clauses, and the
			// lexer tokenizes OCTET and STRING as separate keywords.
			var object ast.Ident
			if objToken.Kind == lexer.TokKwOctet && p.check(lexer.TokKwString) {
				strToken := p.advance()
				span := types.NewSpan(objToken.Span.Start, strToken.Span.End)
				object = ast.Ident{Name: "OCTET STRING", Span: span}
			} else {
				object = p.makeIdent(objToken)
			}

			span := types.NewSpan(itemStart, object.Span.End)
			indexes = append(indexes, ast.IndexItem{
				Implied: implied,
				Object:  object,
				Span:    span,
			})

			if p.check(lexer.TokComma) {
				p.advance()
			}
		}

		endToken, err := p.expect(lexer.TokRBrace)
		if err != nil {
			return nil, nil, err
		}
		span := types.NewSpan(start, endToken.Span.End)
		return &ast.IndexClauseIndex{Items: indexes, Span: span}, nil, nil
	} else if p.check(lexer.TokKwAugments) {
		start := p.currentSpan().Start
		p.advance()
		if _, err := p.expect(lexer.TokLBrace); err != nil {
			return nil, nil, err
		}

		targetToken, err := p.expectIdentifier()
		if err != nil {
			return nil, nil, err
		}
		target := p.makeIdent(targetToken)

		endToken, err := p.expect(lexer.TokRBrace)
		if err != nil {
			return nil, nil, err
		}
		span := types.NewSpan(start, endToken.Span.End)
		return nil, &ast.AugmentsClause{Target: target, Span: span}, nil
	}

	return nil, nil, nil
}

// parseDefValClause parses: DEFVAL { content }.
func (p *Parser) parseDefValClause() (ast.DefValClause, *types.SpanDiagnostic) {
	start := p.currentSpan().Start
	if _, err := p.expect(lexer.TokKwDefval); err != nil {
		return ast.DefValClause{}, err
	}
	if _, err := p.expect(lexer.TokLBrace); err != nil {
		return ast.DefValClause{}, err
	}

	value, err := p.parseDefVal()
	if err != nil {
		return ast.DefValClause{}, err
	}

	endToken, err := p.expect(lexer.TokRBrace)
	if err != nil {
		return ast.DefValClause{}, err
	}
	span := types.NewSpan(start, endToken.Span.End)

	return ast.DefValClause{Value: value, Span: span}, nil
}

// parseDefVal parses the value inside DEFVAL { ... } braces.
func (p *Parser) parseDefVal() (ast.DefVal, *types.SpanDiagnostic) {
	contentStart := p.currentSpan().Start

	kind := p.peek().Kind
	switch kind {
	case lexer.TokNegativeNumber, lexer.TokNumber:
		return p.parseDefValNumber()
	case lexer.TokQuotedString:
		return p.parseDefValString()
	case lexer.TokHexString:
		return p.parseDefValHexString()
	case lexer.TokBinString:
		return p.parseDefValBinaryString()
	case lexer.TokLowercaseIdent, lexer.TokUppercaseIdent:
		token := p.advance()
		ident := p.makeIdent(token)
		return &ast.DefValContentIdentifier{Name: ident}, nil
	case lexer.TokLBrace:
		return p.parseDefValBracedContent()
	default:
		// Keywords can be valid enum labels in DEFVAL (e.g., mandatory, optional, true, false)
		if kind.IsKeyword() {
			token := p.advance()
			ident := p.makeIdent(token)
			return &ast.DefValContentIdentifier{Name: ident}, nil
		}
		return p.parseDefValSkipUnknown(contentStart)
	}
}

func (p *Parser) parseDefValNumber() (ast.DefVal, *types.SpanDiagnostic) {
	token := p.advance()
	if token.Kind == lexer.TokNegativeNumber {
		value := p.parseI64(token.Span, "DEFVAL integer")
		return &ast.DefValContentInteger{Value: value}, nil
	}

	text := p.text(token.Span)
	if value, err := strconv.ParseInt(text, 10, 64); err == nil {
		return &ast.DefValContentInteger{Value: value}, nil
	}
	if value, err := strconv.ParseUint(text, 10, 64); err == nil {
		return &ast.DefValContentUnsigned{Value: value}, nil
	}
	value := p.parseI64(token.Span, "DEFVAL integer")
	return &ast.DefValContentInteger{Value: value}, nil
}

func (p *Parser) parseDefValString() (ast.DefVal, *types.SpanDiagnostic) {
	qs, err := p.parseQuotedString()
	if err != nil {
		return nil, err
	}
	return &ast.DefValContentString{Value: qs}, nil
}

func (p *Parser) parseDefValHexString() (ast.DefVal, *types.SpanDiagnostic) {
	token := p.advance()
	content := stripQuotedLiteral(p.text(token.Span))
	return &ast.DefValContentHexString{Content: content, Span: token.Span}, nil
}

func (p *Parser) parseDefValBinaryString() (ast.DefVal, *types.SpanDiagnostic) {
	token := p.advance()
	content := stripQuotedLiteral(p.text(token.Span))
	return &ast.DefValContentBinaryString{Content: content, Span: token.Span}, nil
}

// stripQuotedLiteral strips the 'xxx'H or 'xxx'B quoting from a hex or binary
// string literal, returning just the inner content.
func stripQuotedLiteral(s string) string {
	s, _ = strings.CutPrefix(s, "'")
	if inner, ok := strings.CutSuffix(s, "'H"); ok {
		return inner
	}
	if inner, ok := strings.CutSuffix(s, "'h"); ok {
		return inner
	}
	if inner, ok := strings.CutSuffix(s, "'B"); ok {
		return inner
	}
	if inner, ok := strings.CutSuffix(s, "'b"); ok {
		return inner
	}
	return s
}

func (p *Parser) parseDefValBracedContent() (ast.DefVal, *types.SpanDiagnostic) {
	p.advance() // consume opening brace
	innerStart := p.currentSpan().Start

	// Empty braces: BITS { {} }
	if p.check(lexer.TokRBrace) {
		endToken := p.advance()
		span := types.NewSpan(innerStart, endToken.Span.End)
		return &ast.DefValContentBits{Labels: nil, Span: span}, nil
	}

	kind := p.peek().Kind
	switch kind {
	case lexer.TokLowercaseIdent, lexer.TokUppercaseIdent:
		return p.parseDefValBracedIdent(innerStart)
	case lexer.TokNumber:
		return p.parseDefValOidNumeric(innerStart)
	default:
		// Keywords can be valid BITS labels (e.g., mandatory, optional)
		if kind.IsKeyword() {
			return p.parseDefValBracedIdent(innerStart)
		}
		return p.parseDefValSkipBraced(innerStart)
	}
}

func (p *Parser) parseDefValBracedIdent(innerStart types.ByteOffset) (ast.DefVal, *types.SpanDiagnostic) {
	identToken := p.advance()
	ident := p.makeIdent(identToken)

	if p.check(lexer.TokComma) || p.check(lexer.TokRBrace) {
		// This is BITS: { flag1, flag2 }
		return p.parseDefValBitsLabels(ident, innerStart)
	}
	// This is OID: { sysName 0 } or { iso 3 6 1 }
	return p.parseDefValOidWithFirstIdent(ident, identToken, innerStart)
}

func (p *Parser) parseDefValBitsLabels(first ast.Ident, innerStart types.ByteOffset) (ast.DefVal, *types.SpanDiagnostic) {
	labels := []ast.Ident{first}
	for p.check(lexer.TokComma) {
		p.advance()
		kind := p.peek().Kind
		// Accept identifiers or keywords as BITS labels
		if kind.IsIdentifier() || kind.IsKeyword() {
			token := p.advance()
			labels = append(labels, ast.Ident{Name: p.text(token.Span), Span: token.Span})
		}
	}
	endToken, err := p.expect(lexer.TokRBrace)
	if err != nil {
		return nil, err
	}
	span := types.NewSpan(innerStart, endToken.Span.End)
	return &ast.DefValContentBits{Labels: labels, Span: span}, nil
}

func (p *Parser) parseDefValOidWithFirstIdent(ident ast.Ident, identToken lexer.Token, innerStart types.ByteOffset) (ast.DefVal, *types.SpanDiagnostic) {
	var components []ast.OidComponent

	// First component is the identifier we already parsed
	if p.check(lexer.TokLParen) {
		p.advance()
		numToken, err := p.expect(lexer.TokNumber)
		if err != nil {
			return nil, err
		}
		number := p.parseU32(numToken.Span, "OID component")
		endParen, err := p.expect(lexer.TokRParen)
		if err != nil {
			return nil, err
		}
		components = append(components, &ast.OidComponentNamedNumber{
			Name: ident,
			Num:  number,
			Span: types.NewSpan(identToken.Span.Start, endParen.Span.End),
		})
	} else {
		components = append(components, &ast.OidComponentName{Name: ident})
	}

	// Parse remaining components and closing brace
	return p.finishDefValOid(components, innerStart)
}

func (p *Parser) parseDefValOidNumeric(innerStart types.ByteOffset) (ast.DefVal, *types.SpanDiagnostic) {
	return p.finishDefValOid(nil, innerStart)
}

func (p *Parser) finishDefValOid(components []ast.OidComponent, innerStart types.ByteOffset) (ast.DefVal, *types.SpanDiagnostic) {
	components, err := p.parseDefValOidComponents(components)
	if err != nil {
		return nil, err
	}

	endToken, err := p.expect(lexer.TokRBrace)
	if err != nil {
		return nil, err
	}
	span := types.NewSpan(innerStart, endToken.Span.End)
	return &ast.DefValContentObjectIdentifier{Components: components, Span: span}, nil
}

func (p *Parser) parseDefValOidComponents(components []ast.OidComponent) ([]ast.OidComponent, *types.SpanDiagnostic) {
	for !p.check(lexer.TokRBrace) && !p.isEOF() {
		if !p.check(lexer.TokNumber) && !p.check(lexer.TokLowercaseIdent) && !p.check(lexer.TokUppercaseIdent) {
			break
		}
		compStart := p.currentSpan().Start
		comp, err := p.parseOidComponent(compStart)
		if err != nil {
			return components, err
		}
		components = append(components, comp)
	}
	return components, nil
}

func (p *Parser) parseDefValSkipBraced(start types.ByteOffset) (ast.DefVal, *types.SpanDiagnostic) {
	p.skipBracedContent(false)
	endToken, err := p.expect(lexer.TokRBrace)
	if err != nil {
		return nil, err
	}
	span := types.NewSpan(start, endToken.Span.End)
	return &ast.DefValContentUnparsed{Span: span}, nil
}

func (p *Parser) parseDefValSkipUnknown(contentStart types.ByteOffset) (ast.DefVal, *types.SpanDiagnostic) {
	depth := 0
	for !p.isEOF() {
		switch p.peek().Kind {
		case lexer.TokLBrace:
			depth++
			p.advance()
		case lexer.TokRBrace:
			if depth == 0 {
				span := types.NewSpan(contentStart, p.currentSpan().Start)
				return &ast.DefValContentUnparsed{Span: span}, nil
			}
			depth--
			p.advance()
		default:
			p.advance()
		}
	}
	span := types.NewSpan(contentStart, p.currentSpan().Start)
	return &ast.DefValContentUnparsed{Span: span}, nil
}

// parseQuotedString consumes a quoted string token and strips the quotes.
func (p *Parser) parseQuotedString() (ast.QuotedString, *types.SpanDiagnostic) {
	if !p.check(lexer.TokQuotedString) {
		diag := p.makeError("expected quoted string")
		return ast.QuotedString{}, &diag
	}
	token := p.advance()
	fullText := p.text(token.Span)
	value := ""
	if len(fullText) >= 2 && fullText[len(fullText)-1] == '"' {
		// Properly terminated: strip both quotes
		value = fullText[1 : len(fullText)-1]
	} else if len(fullText) >= 1 {
		// Unterminated: strip only the opening quote
		value = fullText[1:]
	}
	return ast.QuotedString{Value: value, Span: token.Span}, nil
}

// parseOptionalReference parses an optional REFERENCE clause, returning nil
// if not present.
func (p *Parser) parseOptionalReference() (*ast.QuotedString, *types.SpanDiagnostic) {
	if !p.check(lexer.TokKwReference) {
		return nil, nil
	}
	p.advance()
	qs, err := p.parseQuotedString()
	if err != nil {
		return nil, err
	}
	return &qs, nil
}

// parseModuleIdentity parses a MODULE-IDENTITY macro invocation.
func (p *Parser) parseModuleIdentity() (ast.Definition, *types.SpanDiagnostic) {
	start := p.currentSpan().Start

	nameToken := p.advance()
	name := p.makeIdentWithValidation(nameToken)
	p.validateValueReference(name.Name, nameToken.Span)

	if _, err := p.expect(lexer.TokKwModuleIdentity); err != nil {
		return nil, err
	}

	// LAST-UPDATED
	if _, err := p.expect(lexer.TokKwLastUpdated); err != nil {
		return nil, err
	}
	lastUpdated, err := p.parseQuotedString()
	if err != nil {
		return nil, err
	}

	// ORGANIZATION
	if _, err := p.expect(lexer.TokKwOrganization); err != nil {
		return nil, err
	}
	organization, err := p.parseQuotedString()
	if err != nil {
		return nil, err
	}

	// CONTACT-INFO
	if _, err := p.expect(lexer.TokKwContactInfo); err != nil {
		return nil, err
	}
	contactInfo, err := p.parseQuotedString()
	if err != nil {
		return nil, err
	}

	// DESCRIPTION
	if _, err := p.expect(lexer.TokKwDescription); err != nil {
		return nil, err
	}
	description, err := p.parseQuotedString()
	if err != nil {
		return nil, err
	}

	// REVISION clauses (optional, multiple)
	var revisions []ast.RevisionClause
	for p.check(lexer.TokKwRevision) {
		revStart := p.currentSpan().Start
		p.advance()
		date, err := p.parseQuotedString()
		if err != nil {
			return nil, err
		}
		if _, err := p.expect(lexer.TokKwDescription); err != nil {
			return nil, err
		}
		revDescription, err := p.parseQuotedString()
		if err != nil {
			return nil, err
		}
		span := types.NewSpan(revStart, revDescription.Span.End)
		revisions = append(revisions, ast.RevisionClause{
			Date:        date,
			Description: revDescription,
			Span:        span,
		})
	}

	// ::= { oid }
	if _, err := p.expect(lexer.TokColonColonEqual); err != nil {
		return nil, err
	}
	oid, err := p.parseOidAssignment()
	if err != nil {
		return nil, err
	}

	span := types.NewSpan(start, oid.Span.End)
	return &ast.ModuleIdentityDef{
		DefBase:       ast.DefBase{Name: name, Span: span},
		LastUpdated:   lastUpdated,
		Organization:  organization,
		ContactInfo:   contactInfo,
		Description:   description,
		Revisions:     revisions,
		OidAssignment: oid,
	}, nil
}

// parseObjectIdentity parses an OBJECT-IDENTITY macro invocation.
func (p *Parser) parseObjectIdentity() (ast.Definition, *types.SpanDiagnostic) {
	start := p.currentSpan().Start

	nameToken := p.advance()
	name := p.makeIdentWithValidation(nameToken)
	p.validateValueReference(name.Name, nameToken.Span)

	if _, err := p.expect(lexer.TokKwObjectIdentity); err != nil {
		return nil, err
	}

	// STATUS
	status, err := p.parseStatusClause()
	if err != nil {
		return nil, err
	}

	// DESCRIPTION
	if _, err := p.expect(lexer.TokKwDescription); err != nil {
		return nil, err
	}
	description, err := p.parseQuotedString()
	if err != nil {
		return nil, err
	}

	// REFERENCE (optional)
	reference, err := p.parseOptionalReference()
	if err != nil {
		return nil, err
	}

	// ::= { oid }
	if _, err := p.expect(lexer.TokColonColonEqual); err != nil {
		return nil, err
	}
	oid, err := p.parseOidAssignment()
	if err != nil {
		return nil, err
	}

	span := types.NewSpan(start, oid.Span.End)
	return &ast.ObjectIdentityDef{
		DefBase:       ast.DefBase{Name: name, Span: span},
		Status:        status,
		Description:   description,
		Reference:     reference,
		OidAssignment: oid,
	}, nil
}

// parseNotificationType parses a NOTIFICATION-TYPE macro invocation.
func (p *Parser) parseNotificationType() (ast.Definition, *types.SpanDiagnostic) {
	start := p.currentSpan().Start

	nameToken := p.advance()
	name := p.makeIdentWithValidation(nameToken)
	p.validateValueReference(name.Name, nameToken.Span)

	if _, err := p.expect(lexer.TokKwNotificationType); err != nil {
		return nil, err
	}

	// OBJECTS (optional)
	var objects []ast.Ident
	if p.check(lexer.TokKwObjects) {
		p.advance()
		objs, err := p.parseBracedIdentifierList()
		if err != nil {
			return nil, err
		}
		objects = objs
	}

	// STATUS
	status, err := p.parseStatusClause()
	if err != nil {
		return nil, err
	}

	// DESCRIPTION
	if _, err := p.expect(lexer.TokKwDescription); err != nil {
		return nil, err
	}
	description, err := p.parseQuotedString()
	if err != nil {
		return nil, err
	}

	// REFERENCE (optional)
	reference, err := p.parseOptionalReference()
	if err != nil {
		return nil, err
	}

	// ::= { oid }
	if _, err := p.expect(lexer.TokColonColonEqual); err != nil {
		return nil, err
	}
	oid, err := p.parseOidAssignment()
	if err != nil {
		return nil, err
	}

	span := types.NewSpan(start, oid.Span.End)
	return &ast.NotificationTypeDef{
		DefBase:       ast.DefBase{Name: name, Span: span},
		Objects:       objects,
		Status:        status,
		Description:   description,
		Reference:     reference,
		OidAssignment: oid,
	}, nil
}

// parseTrapType parses a TRAP-TYPE macro invocation (SMIv1).
func (p *Parser) parseTrapType() (ast.Definition, *types.SpanDiagnostic) {
	start := p.currentSpan().Start

	nameToken := p.advance()
	name := p.makeIdentWithValidation(nameToken)
	p.validateValueReference(name.Name, nameToken.Span)

	if _, err := p.expect(lexer.TokKwTrapType); err != nil {
		return nil, err
	}

	// ENTERPRISE
	if _, err := p.expect(lexer.TokKwEnterprise); err != nil {
		return nil, err
	}
	enterpriseToken, err := p.expectIdentifier()
	if err != nil {
		return nil, err
	}
	enterprise := p.makeIdent(enterpriseToken)

	// VARIABLES (optional)
	var variables []ast.Ident
	if p.check(lexer.TokKwVariables) {
		p.advance()
		vars, err := p.parseBracedIdentifierList()
		if err != nil {
			return nil, err
		}
		variables = vars
	}

	// DESCRIPTION (optional)
	var description *ast.QuotedString
	if p.check(lexer.TokKwDescription) {
		p.advance()
		qs, err := p.parseQuotedString()
		if err != nil {
			return nil, err
		}
		description = &qs
	}

	// REFERENCE (optional)
	reference, err := p.parseOptionalReference()
	if err != nil {
		return nil, err
	}

	// ::= number
	if _, err := p.expect(lexer.TokColonColonEqual); err != nil {
		return nil, err
	}
	numToken, err := p.expect(lexer.TokNumber)
	if err != nil {
		return nil, err
	}
	trapNumber := p.parseU32(numToken.Span, "trap number")

	span := types.NewSpan(start, numToken.Span.End)
	return &ast.TrapTypeDef{
		DefBase:     ast.DefBase{Name: name, Span: span},
		Enterprise:  enterprise,
		Variables:   variables,
		Description: description,
		Reference:   reference,
		TrapNumber:  trapNumber,
	}, nil
}

// parseTextualConvention parses: Name TEXTUAL-CONVENTION ...
func (p *Parser) parseTextualConvention() (ast.Definition, *types.SpanDiagnostic) {
	start := p.currentSpan().Start

	nameToken := p.advance()
	name := p.makeIdentWithValidation(nameToken)

	if _, err := p.expect(lexer.TokKwTextualConvention); err != nil {
		return nil, err
	}

	return p.parseTextualConventionBody(name, start)
}

// parseTextualConventionWithAssignment parses the alternate form:
// Name ::= TEXTUAL-CONVENTION ...
func (p *Parser) parseTextualConventionWithAssignment() (ast.Definition, *types.SpanDiagnostic) {
	start := p.currentSpan().Start

	nameToken := p.advance()
	name := p.makeIdentWithValidation(nameToken)

	if _, err := p.expect(lexer.TokColonColonEqual); err != nil {
		return nil, err
	}
	if _, err := p.expect(lexer.TokKwTextualConvention); err != nil {
		return nil, err
	}

	return p.parseTextualConventionBody(name, start)
}

// parseTextualConventionBody parses the shared body of a TEXTUAL-CONVENTION
// (DISPLAY-HINT, STATUS, DESCRIPTION, REFERENCE, SYNTAX).
func (p *Parser) parseTextualConventionBody(name ast.Ident, start types.ByteOffset) (ast.Definition, *types.SpanDiagnostic) {
	// DISPLAY-HINT (optional)
	var displayHint *ast.QuotedString
	if p.check(lexer.TokKwDisplayHint) {
		p.advance()
		qs, err := p.parseQuotedString()
		if err != nil {
			return nil, err
		}
		displayHint = &qs
	}

	// STATUS
	status, err := p.parseStatusClause()
	if err != nil {
		return nil, err
	}

	// DESCRIPTION
	if _, err := p.expect(lexer.TokKwDescription); err != nil {
		return nil, err
	}
	description, err := p.parseQuotedString()
	if err != nil {
		return nil, err
	}

	// REFERENCE (optional)
	reference, err := p.parseOptionalReference()
	if err != nil {
		return nil, err
	}

	// SYNTAX
	if _, err := p.expect(lexer.TokKwSyntax); err != nil {
		return nil, err
	}
	syntax, err := p.parseSyntaxClause()
	if err != nil {
		return nil, err
	}

	span := types.NewSpan(start, syntax.Span.End)
	return &ast.TextualConventionDef{
		DefBase:     ast.DefBase{Name: name, Span: span},
		DisplayHint: displayHint,
		Status:      status,
		Description: description,
		Reference:   reference,
		Syntax:      syntax,
	}, nil
}

// parseTypeAssignment parses: TypeName ::= TypeSyntax
func (p *Parser) parseTypeAssignment() (ast.Definition, *types.SpanDiagnostic) {
	start := p.currentSpan().Start

	nameToken := p.advance()
	name := p.makeIdentWithValidation(nameToken)

	if _, err := p.expect(lexer.TokColonColonEqual); err != nil {
		return nil, err
	}

	syntax, err := p.parseTypeSyntax()
	if err != nil {
		return nil, err
	}
	span := types.NewSpan(start, syntax.SyntaxSpan().End)

	return &ast.TypeAssignmentDef{
		DefBase: ast.DefBase{Name: name, Span: span},
		Syntax:  syntax,
	}, nil
}

// parseObjectGroup parses an OBJECT-GROUP macro invocation.
func (p *Parser) parseObjectGroup() (ast.Definition, *types.SpanDiagnostic) {
	return p.parseGroupDef(lexer.TokKwObjectGroup, lexer.TokKwObjects, func(base ast.DefBase, members []ast.Ident, status ast.StatusClause, desc ast.QuotedString, ref *ast.QuotedString, oid ast.OidAssignment) ast.Definition {
		return &ast.ObjectGroupDef{DefBase: base, Objects: members, Status: status, Description: desc, Reference: ref, OidAssignment: oid}
	})
}

// parseNotificationGroup parses a NOTIFICATION-GROUP macro invocation.
func (p *Parser) parseNotificationGroup() (ast.Definition, *types.SpanDiagnostic) {
	return p.parseGroupDef(lexer.TokKwNotificationGroup, lexer.TokKwNotifications, func(base ast.DefBase, members []ast.Ident, status ast.StatusClause, desc ast.QuotedString, ref *ast.QuotedString, oid ast.OidAssignment) ast.Definition {
		return &ast.NotificationGroupDef{DefBase: base, Notifications: members, Status: status, Description: desc, Reference: ref, OidAssignment: oid}
	})
}

// parseGroupDef is the shared implementation for OBJECT-GROUP and NOTIFICATION-GROUP.
func (p *Parser) parseGroupDef(macroKw, membersKw lexer.TokenKind, build func(ast.DefBase, []ast.Ident, ast.StatusClause, ast.QuotedString, *ast.QuotedString, ast.OidAssignment) ast.Definition) (ast.Definition, *types.SpanDiagnostic) {
	start := p.currentSpan().Start

	nameToken := p.advance()
	name := p.makeIdentWithValidation(nameToken)
	p.validateValueReference(name.Name, nameToken.Span)

	if _, err := p.expect(macroKw); err != nil {
		return nil, err
	}

	if _, err := p.expect(membersKw); err != nil {
		return nil, err
	}
	members, err := p.parseBracedIdentifierList()
	if err != nil {
		return nil, err
	}

	status, err := p.parseStatusClause()
	if err != nil {
		return nil, err
	}

	if _, err := p.expect(lexer.TokKwDescription); err != nil {
		return nil, err
	}
	description, err := p.parseQuotedString()
	if err != nil {
		return nil, err
	}

	reference, err := p.parseOptionalReference()
	if err != nil {
		return nil, err
	}

	if _, err := p.expect(lexer.TokColonColonEqual); err != nil {
		return nil, err
	}
	oid, err := p.parseOidAssignment()
	if err != nil {
		return nil, err
	}

	span := types.NewSpan(start, oid.Span.End)
	return build(ast.DefBase{Name: name, Span: span}, members, status, description, reference, oid), nil
}

// parseModuleCompliance parses a MODULE-COMPLIANCE macro invocation.
func (p *Parser) parseModuleCompliance() (ast.Definition, *types.SpanDiagnostic) {
	start := p.currentSpan().Start

	nameToken := p.advance()
	name := p.makeIdentWithValidation(nameToken)
	p.validateValueReference(name.Name, nameToken.Span)

	if _, err := p.expect(lexer.TokKwModuleCompliance); err != nil {
		return nil, err
	}

	// STATUS
	status, err := p.parseStatusClause()
	if err != nil {
		return nil, err
	}

	// DESCRIPTION
	if _, err := p.expect(lexer.TokKwDescription); err != nil {
		return nil, err
	}
	description, err := p.parseQuotedString()
	if err != nil {
		return nil, err
	}

	// REFERENCE (optional)
	reference, err := p.parseOptionalReference()
	if err != nil {
		return nil, err
	}

	// Parse MODULE clauses
	var modules []ast.ComplianceModule
	for p.check(lexer.TokKwModule) {
		mod, err := p.parseComplianceModule()
		if err != nil {
			return nil, err
		}
		modules = append(modules, mod)
	}

	// ::= { oid }
	if _, err := p.expect(lexer.TokColonColonEqual); err != nil {
		return nil, err
	}
	oid, err := p.parseOidAssignment()
	if err != nil {
		return nil, err
	}

	span := types.NewSpan(start, oid.Span.End)
	return &ast.ModuleComplianceDef{
		DefBase:       ast.DefBase{Name: name, Span: span},
		Status:        status,
		Description:   description,
		Reference:     reference,
		Modules:       modules,
		OidAssignment: oid,
	}, nil
}

func (p *Parser) parseComplianceModule() (ast.ComplianceModule, *types.SpanDiagnostic) {
	start := p.currentSpan().Start
	if _, err := p.expect(lexer.TokKwModule); err != nil {
		return ast.ComplianceModule{}, err
	}

	// Optional module name
	var moduleName *ast.Ident
	if p.check(lexer.TokUppercaseIdent) {
		nameToken := p.advance()
		ident := p.makeIdent(nameToken)
		moduleName = &ident
	}

	// Optional module OID
	var moduleOid *ast.OidAssignment
	if p.check(lexer.TokLBrace) {
		oid, err := p.parseOidAssignment()
		if err != nil {
			return ast.ComplianceModule{}, err
		}
		moduleOid = &oid
	}

	// MANDATORY-GROUPS (optional)
	var mandatoryGroups []ast.Ident
	if p.check(lexer.TokKwMandatoryGroups) {
		groups, err := p.parseMandatoryGroups()
		if err != nil {
			return ast.ComplianceModule{}, err
		}
		mandatoryGroups = groups
	}

	// GROUP and OBJECT refinements
	var compliances []ast.Compliance
	for p.check(lexer.TokKwGroup) || p.check(lexer.TokKwObject) {
		if p.check(lexer.TokKwGroup) {
			group, err := p.parseComplianceGroup()
			if err != nil {
				return ast.ComplianceModule{}, err
			}
			compliances = append(compliances, group)
		} else {
			obj, err := p.parseComplianceObject()
			if err != nil {
				return ast.ComplianceModule{}, err
			}
			compliances = append(compliances, obj)
		}
	}

	return ast.ComplianceModule{
		ModuleName:      moduleName,
		ModuleOid:       moduleOid,
		MandatoryGroups: mandatoryGroups,
		Compliances:     compliances,
		Span:            types.NewSpan(start, p.lastEnd),
	}, nil
}

func (p *Parser) parseMandatoryGroups() ([]ast.Ident, *types.SpanDiagnostic) {
	if _, err := p.expect(lexer.TokKwMandatoryGroups); err != nil {
		return nil, err
	}
	return p.parseBracedIdentifierList()
}

func (p *Parser) parseComplianceGroup() (*ast.ComplianceGroup, *types.SpanDiagnostic) {
	start := p.currentSpan().Start
	if _, err := p.expect(lexer.TokKwGroup); err != nil {
		return nil, err
	}
	groupIdent, err := p.parseIdentifierAsIdent()
	if err != nil {
		return nil, err
	}
	if _, err := p.expect(lexer.TokKwDescription); err != nil {
		return nil, err
	}
	description, err := p.parseQuotedString()
	if err != nil {
		return nil, err
	}
	end := description.Span.End
	return &ast.ComplianceGroup{
		Group:       groupIdent,
		Description: description,
		Span:        types.NewSpan(start, end),
	}, nil
}

// parseOptionalSyntaxClauses parses optional SYNTAX and WRITE-SYNTAX clauses,
// shared by MODULE-COMPLIANCE objects and AGENT-CAPABILITIES variations.
func (p *Parser) parseOptionalSyntaxClauses() (syntax, writeSyntax *ast.SyntaxClause, err *types.SpanDiagnostic) {
	if p.check(lexer.TokKwSyntax) {
		p.advance()
		sc, e := p.parseSyntaxClause()
		if e != nil {
			return nil, nil, e
		}
		syntax = &sc
	}
	if p.check(lexer.TokKwWriteSyntax) {
		p.advance()
		sc, e := p.parseSyntaxClause()
		if e != nil {
			return nil, nil, e
		}
		writeSyntax = &sc
	}
	return syntax, writeSyntax, nil
}

func (p *Parser) parseComplianceObject() (*ast.ComplianceObject, *types.SpanDiagnostic) {
	start := p.currentSpan().Start
	if _, err := p.expect(lexer.TokKwObject); err != nil {
		return nil, err
	}
	objectIdent, err := p.parseIdentifierAsIdent()
	if err != nil {
		return nil, err
	}

	syntax, writeSyntax, err := p.parseOptionalSyntaxClauses()
	if err != nil {
		return nil, err
	}

	// Optional MIN-ACCESS
	var minAccess *ast.AccessClause
	if p.check(lexer.TokKwMinAccess) {
		ac, err := p.parseAccessClause()
		if err != nil {
			return nil, err
		}
		minAccess = &ac
	}

	// Required DESCRIPTION
	if _, err := p.expect(lexer.TokKwDescription); err != nil {
		return nil, err
	}
	description, err := p.parseQuotedString()
	if err != nil {
		return nil, err
	}

	end := description.Span.End
	return &ast.ComplianceObject{
		Object:      objectIdent,
		Syntax:      syntax,
		WriteSyntax: writeSyntax,
		MinAccess:   minAccess,
		Description: description,
		Span:        types.NewSpan(start, end),
	}, nil
}

func (p *Parser) parseIdentifierAsIdent() (ast.Ident, *types.SpanDiagnostic) {
	token, err := p.expectIdentifier()
	if err != nil {
		return ast.Ident{}, err
	}
	return p.makeIdent(token), nil
}

// parseAgentCapabilities parses an AGENT-CAPABILITIES macro invocation.
func (p *Parser) parseAgentCapabilities() (ast.Definition, *types.SpanDiagnostic) {
	start := p.currentSpan().Start

	nameToken := p.advance()
	name := p.makeIdentWithValidation(nameToken)
	p.validateValueReference(name.Name, nameToken.Span)

	if _, err := p.expect(lexer.TokKwAgentCapabilities); err != nil {
		return nil, err
	}

	// PRODUCT-RELEASE
	if _, err := p.expect(lexer.TokKwProductRelease); err != nil {
		return nil, err
	}
	productRelease, err := p.parseQuotedString()
	if err != nil {
		return nil, err
	}

	// STATUS
	status, err := p.parseStatusClause()
	if err != nil {
		return nil, err
	}

	// DESCRIPTION
	if _, err := p.expect(lexer.TokKwDescription); err != nil {
		return nil, err
	}
	description, err := p.parseQuotedString()
	if err != nil {
		return nil, err
	}

	// REFERENCE (optional)
	reference, err := p.parseOptionalReference()
	if err != nil {
		return nil, err
	}

	// Parse SUPPORTS clauses
	var supports []ast.SupportsModule
	for p.check(lexer.TokKwSupports) {
		sup, err := p.parseSupportsModule()
		if err != nil {
			return nil, err
		}
		supports = append(supports, sup)
	}

	// ::= { oid }
	if _, err := p.expect(lexer.TokColonColonEqual); err != nil {
		return nil, err
	}
	oid, err := p.parseOidAssignment()
	if err != nil {
		return nil, err
	}

	span := types.NewSpan(start, oid.Span.End)
	return &ast.AgentCapabilitiesDef{
		DefBase:        ast.DefBase{Name: name, Span: span},
		ProductRelease: productRelease,
		Status:         status,
		Description:    description,
		Reference:      reference,
		Supports:       supports,
		OidAssignment:  oid,
	}, nil
}

func (p *Parser) parseSupportsModule() (ast.SupportsModule, *types.SpanDiagnostic) {
	start := p.currentSpan().Start
	if _, err := p.expect(lexer.TokKwSupports); err != nil {
		return ast.SupportsModule{}, err
	}

	// Module name
	moduleName, err := p.parseIdentifierAsIdent()
	if err != nil {
		return ast.SupportsModule{}, err
	}

	// Optional module OID
	var moduleOid *ast.OidAssignment
	if p.check(lexer.TokLBrace) {
		oid, err := p.parseOidAssignment()
		if err != nil {
			return ast.SupportsModule{}, err
		}
		moduleOid = &oid
	}

	// INCLUDES { groups }
	if _, err := p.expect(lexer.TokKwIncludes); err != nil {
		return ast.SupportsModule{}, err
	}
	includes, err := p.parseBracedIdentifierList()
	if err != nil {
		return ast.SupportsModule{}, err
	}

	// VARIATION clauses
	var variations []ast.Variation
	for p.check(lexer.TokKwVariation) {
		v, err := p.parseVariationClause()
		if err != nil {
			return ast.SupportsModule{}, err
		}
		variations = append(variations, *v)
	}

	return ast.SupportsModule{
		ModuleName: moduleName,
		ModuleOid:  moduleOid,
		Includes:   includes,
		Variations: variations,
		Span:       types.NewSpan(start, p.lastEnd),
	}, nil
}

func (p *Parser) parseVariationClause() (*ast.Variation, *types.SpanDiagnostic) {
	start := p.currentSpan().Start
	if _, err := p.expect(lexer.TokKwVariation); err != nil {
		return nil, err
	}

	// Object or notification name
	name, err := p.parseIdentifierAsIdent()
	if err != nil {
		return nil, err
	}

	syntax, writeSyntax, err := p.parseOptionalSyntaxClauses()
	if err != nil {
		return nil, err
	}

	// Optional ACCESS
	var access *ast.AccessClause
	if p.check(lexer.TokKwAccess) {
		ac, err := p.parseAccessClause()
		if err != nil {
			return nil, err
		}
		access = &ac
	}

	// Optional CREATION-REQUIRES
	var creationRequires []ast.Ident
	if p.check(lexer.TokKwCreationRequires) {
		p.advance()
		objs, err := p.parseBracedIdentifierList()
		if err != nil {
			return nil, err
		}
		creationRequires = objs
	}

	// Optional DEFVAL
	var defval *ast.DefValClause
	if p.check(lexer.TokKwDefval) {
		dv, err := p.parseDefValClause()
		if err != nil {
			return nil, err
		}
		defval = &dv
	}

	// Required DESCRIPTION
	if _, err := p.expect(lexer.TokKwDescription); err != nil {
		return nil, err
	}
	description, err := p.parseQuotedString()
	if err != nil {
		return nil, err
	}

	end := description.Span.End

	return &ast.Variation{
		Name:             name,
		Syntax:           syntax,
		WriteSyntax:      writeSyntax,
		Access:           access,
		CreationRequires: creationRequires,
		DefVal:           defval,
		Description:      description,
		Span:             types.NewSpan(start, end),
	}, nil
}

// parseMacroDefinition parses a MACRO definition header and skips
// to END.
func (p *Parser) parseMacroDefinition() (ast.Definition, *types.SpanDiagnostic) {
	start := p.currentSpan().Start

	nameToken := p.advance()
	name := p.makeIdent(nameToken)

	if _, err := p.expect(lexer.TokKwMacro); err != nil {
		return nil, err
	}

	for !p.check(lexer.TokKwEnd) && !p.isEOF() {
		p.advance()
	}

	var endToken lexer.Token
	if p.check(lexer.TokKwEnd) {
		endToken = p.advance()
	} else {
		diag := p.makeError("expected END for MACRO")
		return nil, &diag
	}

	span := types.NewSpan(start, endToken.Span.End)
	return &ast.MacroDefinitionDef{
		DefBase: ast.DefBase{Name: name, Span: span},
	}, nil
}

// parseIdentifierList parses a comma-separated list of identifiers.
func (p *Parser) parseIdentifierList() ([]ast.Ident, *types.SpanDiagnostic) {
	var idents []ast.Ident

	for !p.check(lexer.TokRBrace) && !p.isEOF() {
		token, err := p.expectIdentifier()
		if err != nil {
			return nil, err
		}
		idents = append(idents, p.makeIdent(token))

		if p.check(lexer.TokComma) {
			p.advance()
		} else {
			break
		}
	}

	return idents, nil
}

func (p *Parser) parseBracedIdentifierList() ([]ast.Ident, *types.SpanDiagnostic) {
	if _, err := p.expect(lexer.TokLBrace); err != nil {
		return nil, err
	}
	idents, err := p.parseIdentifierList()
	if err != nil {
		return nil, err
	}
	if _, err := p.expect(lexer.TokRBrace); err != nil {
		return nil, err
	}
	return idents, nil
}

// skipBracedContent skips tokens until the matching closing brace. The opening
// brace must already be consumed. If consumeClose is true, the closing brace
// is also consumed; otherwise it is left for the caller.
func (p *Parser) skipBracedContent(consumeClose bool) {
	depth := 1
	for depth > 0 && !p.isEOF() {
		switch p.peek().Kind {
		case lexer.TokLBrace:
			depth++
			p.advance()
		case lexer.TokRBrace:
			depth--
			if depth > 0 || consumeClose {
				p.advance()
			}
		default:
			p.advance()
		}
	}
}

// recoverToDefinition skips tokens until the start of a new definition
// or END, allowing the parser to continue after an error.
func (p *Parser) recoverToDefinition() {
	for !p.isEOF() && !p.check(lexer.TokKwEnd) {
		current := p.peek().Kind
		next := p.peekNth(1).Kind

		if (current.IsIdentifier() && next.IsMacroKeyword()) ||
			(current == lexer.TokUppercaseIdent && next == lexer.TokColonColonEqual) ||
			(current == lexer.TokUppercaseIdent && next == lexer.TokKwTextualConvention) ||
			(current == lexer.TokUppercaseIdent && next == lexer.TokKwMacro) ||
			(current.IsIdentifier() && next == lexer.TokKwObject &&
				p.peekNth(2).Kind == lexer.TokKwIdentifier) {
			break
		}

		p.advance()
	}
}
