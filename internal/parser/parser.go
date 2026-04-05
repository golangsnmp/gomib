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
	"github.com/golangsnmp/gomib/internal/module"
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

// ParseModule parses all MIB modules in the source and returns their IR.
// A single file may contain multiple DEFINITIONS ::= BEGIN ... END blocks.
// Parse errors are collected in each module's diagnostics rather than
// causing immediate failure.
func (p *Parser) ParseModule() []*module.Module {
	var modules []*module.Module
	for !p.isEOF() {
		mod := p.parseOneModule()
		modules = append(modules, mod)
		if mod.Name == "UNKNOWN" {
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
func (p *Parser) parseOneModule() *module.Module {
	// Snapshot lexer diagnostic count so we only attach new ones to this module.
	lexDiagsBefore := p.lex.DiagnosticCount()
	p.ResetDiagnostics()
	p.language = types.LanguageUnknown

	start := p.currentSpan().Start

	name, nameSpan, err := p.parseModuleHeader()
	if err != nil {
		p.recordParseError(*err)
		p.Log(slog.LevelDebug, "failed to parse module header")
		span := types.NewSpan(start, p.currentSpan().End)
		mod := module.NewModule("UNKNOWN", span)
		lineTable := types.BuildLineTable(p.source)
		mod.LineTable = lineTable
		mod.Diagnostics = p.convertDiagnostics(p.collectSpanDiagnostics(lexDiagsBefore), "UNKNOWN", lineTable)
		return mod
	}

	p.moduleName = name
	_ = nameSpan
	p.Log(slog.LevelDebug, "parsing module", slog.String("module", name))

	mod := module.NewModule(name, types.NewSpan(start, 0))

	if p.check(lexer.TokKwImports) {
		imports, err := p.parseImports()
		if err != nil {
			p.recordParseError(*err)
			p.Log(slog.LevelDebug, "failed to parse imports", slog.String("module", name))
		} else {
			mod.Imports = imports
			p.Log(slog.LevelDebug, "parsed imports",
				slog.String("module", name),
				slog.Int("count", len(imports)))
		}
	}

	// If no imports detected language, default to SMIv1.
	if p.language == types.LanguageUnknown {
		p.language = types.LanguageSMIv1
	}
	mod.Language = p.language

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
		} else if def != nil {
			mod.Definitions = append(mod.Definitions, def)
			switch d := def.(type) {
			case *module.ObjectType:
				mod.ObjectTypes = append(mod.ObjectTypes, d)
			case *module.TypeDef:
				mod.TypeDefs = append(mod.TypeDefs, d)
			case *module.Notification:
				mod.Notifications = append(mod.Notifications, d)
			case *module.ModuleIdentity:
				mod.ModuleIdentities = append(mod.ModuleIdentities, d)
			case *module.ObjectIdentity:
				mod.ObjectIdentities = append(mod.ObjectIdentities, d)
			case *module.ValueAssignment:
				mod.ValueAssignments = append(mod.ValueAssignments, d)
			case *module.ObjectGroup:
				mod.ObjectGroups = append(mod.ObjectGroups, d)
			case *module.NotificationGroup:
				mod.NotificationGroups = append(mod.NotificationGroups, d)
			case *module.ModuleCompliance:
				mod.Compliances = append(mod.Compliances, d)
			case *module.AgentCapabilities:
				mod.Capabilities = append(mod.Capabilities, d)
			}
		}
	}

	if p.check(lexer.TokKwEnd) {
		p.advance()
	} else {
		p.recordParseError(p.makeError("expected END"))
	}

	mod.Span = types.NewSpan(start, p.currentSpan().End)
	lineTable := types.BuildLineTable(p.source)
	mod.LineTable = lineTable

	// Cache LAST-UPDATED for efficient access during resolution.
	for _, def := range mod.Definitions {
		if mi, ok := def.(*module.ModuleIdentity); ok && strings.TrimSpace(mi.LastUpdated) != "" {
			mod.LastUpdated = mi.LastUpdated
			break
		}
	}

	mod.Diagnostics = p.convertDiagnostics(p.collectSpanDiagnostics(lexDiagsBefore), name, lineTable)

	p.Log(slog.LevelDebug, "parsing complete",
		slog.String("module", name),
		slog.Int("definitions", len(mod.Definitions)),
		slog.Int("diagnostics", p.DiagnosticCount()))

	return mod
}

// collectSpanDiagnostics builds the span diagnostics slice for a single module
// by combining new lexer diagnostics (since lexDiagsBefore) with parser
// diagnostics accumulated for this module.
func (p *Parser) collectSpanDiagnostics(lexDiagsBefore int) []types.SpanDiagnostic {
	newLexDiags := p.lex.DiagnosticsFrom(lexDiagsBefore)
	parserDiags := p.Diagnostics()
	combined := make([]types.SpanDiagnostic, 0, len(newLexDiags)+len(parserDiags))
	combined = append(combined, newLexDiags...)
	combined = append(combined, parserDiags...)
	return combined
}

// convertDiagnostics converts SpanDiagnostics to Diagnostics using the line table.
func (p *Parser) convertDiagnostics(spanDiags []types.SpanDiagnostic, moduleName string, lineTable []int) []types.Diagnostic {
	if len(spanDiags) == 0 {
		return nil
	}
	diags := make([]types.Diagnostic, 0, len(spanDiags))
	for _, sd := range spanDiags {
		if !p.DiagConfig.ShouldReport(sd.Code, sd.Severity) {
			continue
		}
		line, col := types.LineColFromTable(lineTable, sd.Span.Start)
		diags = append(diags, types.Diagnostic{
			Severity: sd.Severity,
			Code:     sd.Code,
			Message:  sd.Message,
			Module:   moduleName,
			Line:     line,
			Column:   col,
		})
	}
	return diags
}

// checkEmptyOptionalString emits a diagnostic when an optional quoted string
// clause is present but empty.
func (p *Parser) checkEmptyOptionalString(value string, present bool, span types.Span, defName, clause, code string) {
	if present && value == "" {
		p.EmitDiagnostic(code, span,
			fmt.Sprintf("%q: empty %s clause", defName, clause))
	}
}

// checkEmptyRequiredString emits a diagnostic when a required string clause
// has an empty value.
func (p *Parser) checkEmptyRequiredString(value string, span types.Span, defName, clause, code string) {
	if value == "" {
		p.EmitDiagnostic(code, span,
			fmt.Sprintf("%q: empty %s clause", defName, clause))
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

// parseModuleHeader parses: ModuleName [{ oid }] DEFINITIONS ::= BEGIN
func (p *Parser) parseModuleHeader() (string, types.Span, *types.SpanDiagnostic) {
	nameToken, err := p.expectIdentifier()
	if err != nil {
		return "", types.Span{}, err
	}
	name, nameSpan := p.makeIdentStrWithValidation(nameToken)

	// Skip obsolete module OID that some MIBs include before DEFINITIONS
	if p.check(lexer.TokLBrace) {
		p.advance() // consume opening brace
		p.skipBracedContent(true)
	}

	_, err = p.expect(lexer.TokKwDefinitions)
	if err != nil {
		return "", types.Span{}, err
	}

	_, err = p.expect(lexer.TokColonColonEqual)
	if err != nil {
		return "", types.Span{}, err
	}

	_, err = p.expect(lexer.TokKwBegin)
	if err != nil {
		return "", types.Span{}, err
	}

	return name, nameSpan, nil
}

func (p *Parser) expectIdentifier() (lexer.Token, *types.SpanDiagnostic) {
	if p.check(lexer.TokUppercaseIdent) || p.check(lexer.TokLowercaseIdent) {
		return p.advance(), nil
	}
	// Accept forbidden keywords as identifiers for lenient parsing
	if p.check(lexer.TokForbiddenKeyword) {
		token := p.advance()
		name := p.text(token.Span)
		p.EmitDiagnostic(types.DiagKeywordReserved, token.Span,
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

// isSMIv2Import reports whether importing from this module indicates SMIv2.
func isSMIv2Import(name string) bool {
	bm, ok := module.BaseModuleFromName(name)
	return ok && bm.IsSMIv2()
}

// parseImports parses: IMPORTS symbols FROM Module ... ;
// Produces a flat list of imports (one per symbol) and detects language.
func (p *Parser) parseImports() ([]module.Import, *types.SpanDiagnostic) {
	_, err := p.expect(lexer.TokKwImports)
	if err != nil {
		return nil, err
	}

	var imports []module.Import

	for {
		if p.check(lexer.TokSemicolon) {
			p.advance()
			break
		}

		if p.isEOF() || p.check(lexer.TokKwEnd) {
			diag := p.makeError("unexpected end of imports")
			return imports, &diag
		}

		type symbolInfo struct {
			name string
			span types.Span
		}
		var symbols []symbolInfo

	symbolLoop:
		for {
			kind := p.peek().Kind
			switch {
			case kind.IsMacroKeyword() || kind.IsTypeKeyword() || kind.IsIdentifier():
				symToken := p.advance()
				symName, symSpan := p.makeIdentStr(symToken)
				symbols = append(symbols, symbolInfo{name: symName, span: symSpan})
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
		fromModule := p.text(moduleToken.Span)

		// Detect language from imports.
		if isSMIv2Import(fromModule) {
			p.language = types.LanguageSMIv2
		}

		for _, sym := range symbols {
			imports = append(imports, module.NewImport(fromModule, sym.name, sym.span))
		}
	}

	return imports, nil
}

// parseDefinition dispatches to the appropriate definition parser
// based on lookahead tokens.
func (p *Parser) parseDefinition() (module.Definition, *types.SpanDiagnostic) {
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
			p.EmitDiagnostic(types.DiagBadIdentifierCase, p.peek().Span,
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
func (p *Parser) parseValueAssignment() (module.Definition, *types.SpanDiagnostic) {
	start := p.currentSpan().Start

	nameToken := p.advance()
	name, _ := p.makeIdentStrWithValidation(nameToken)

	// Value references should start with lowercase per RFC 2578
	p.validateValueReference(name, nameToken.Span)

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
	return &module.ValueAssignment{
		DefBase: module.DefBase{Name: name, Span: span},
		Oid:     oid,
	}, nil
}

// parseOidAssignment parses: { parent subid ... }
func (p *Parser) parseOidAssignment() (module.OidAssignment, *types.SpanDiagnostic) {
	start := p.currentSpan().Start
	if _, err := p.expect(lexer.TokLBrace); err != nil {
		return module.OidAssignment{}, err
	}

	var components []module.OidComponent

	for !p.check(lexer.TokRBrace) && !p.isEOF() {
		compStart := p.currentSpan().Start

		if p.check(lexer.TokNumber) || p.check(lexer.TokLowercaseIdent) || p.check(lexer.TokUppercaseIdent) {
			comp, err := p.parseOidComponent(compStart)
			if err != nil {
				return module.OidAssignment{}, err
			}
			components = append(components, comp)
		} else {
			diag := p.makeError("expected OID component")
			return module.OidAssignment{}, &diag
		}
	}

	endToken, err := p.expect(lexer.TokRBrace)
	if err != nil {
		return module.OidAssignment{}, err
	}
	return module.NewOidAssignment(components, types.NewSpan(start, endToken.Span.End)), nil
}

// parseOidComponent parses a single OID component: a number, name, name(number),
// or a qualified reference (Module.name or Module.name(number)).
// The caller must have verified the current token is a Number or identifier.
func (p *Parser) parseOidComponent(compStart types.ByteOffset) (module.OidComponent, *types.SpanDiagnostic) {
	if p.check(lexer.TokNumber) {
		token := p.advance()
		value := p.convertU32(token.Span, "OID component")
		return &module.OidComponentNumber{Value: value, Span: token.Span}, nil
	}

	firstToken := p.advance()
	firstName, firstSpan := p.makeIdentStr(firstToken)

	if p.check(lexer.TokDot) {
		// Qualified reference: Module.name or Module.name(number)
		p.advance() // consume dot

		nameToken, err := p.expect(lexer.TokLowercaseIdent)
		if err != nil {
			return nil, err
		}
		qname := p.text(nameToken.Span)

		if p.check(lexer.TokLParen) {
			// Qualified named number, e.g. Module.name(123)
			p.advance() // (
			numToken, err := p.expect(lexer.TokNumber)
			if err != nil {
				return nil, err
			}
			number := p.convertU32(numToken.Span, "OID component")
			endToken, err := p.expect(lexer.TokRParen)
			if err != nil {
				return nil, err
			}
			return &module.OidComponentQualifiedNamedNumber{
				ModuleValue: firstName,
				NameValue:   qname,
				NumberValue: number,
				Span:        types.NewSpan(compStart, endToken.Span.End),
			}, nil
		}
		// QualifiedName: Module.name
		return &module.OidComponentQualifiedName{
			ModuleValue: firstName,
			NameValue:   qname,
			Span:        types.NewSpan(compStart, nameToken.Span.End),
		}, nil
	}

	if p.check(lexer.TokLParen) {
		// Named number: iso(1), org(3)
		p.advance() // (
		numToken, err := p.expect(lexer.TokNumber)
		if err != nil {
			return nil, err
		}
		number := p.convertU32(numToken.Span, "OID component")
		endToken, err := p.expect(lexer.TokRParen)
		if err != nil {
			return nil, err
		}
		return &module.OidComponentNamedNumber{
			NameValue:   firstName,
			NumberValue: number,
			Span:        types.NewSpan(compStart, endToken.Span.End),
		}, nil
	}

	// Just name: internet, ifEntry
	return &module.OidComponentName{NameValue: firstName, Span: firstSpan}, nil
}

// parseObjectType parses an OBJECT-TYPE macro invocation.
func (p *Parser) parseObjectType() (module.Definition, *types.SpanDiagnostic) {
	start := p.currentSpan().Start

	nameToken := p.advance()
	name, _ := p.makeIdentStrWithValidation(nameToken)
	p.validateValueReference(name, nameToken.Span)

	if _, err := p.expect(lexer.TokKwObjectType); err != nil {
		return nil, err
	}

	// SYNTAX clause (required)
	if _, err := p.expect(lexer.TokKwSyntax); err != nil {
		return nil, err
	}
	syntaxStart := p.currentSpan().Start
	syntax, err := p.parseTypeSyntax()
	if err != nil {
		return nil, err
	}
	syntaxSpan := types.NewSpan(syntaxStart, syntax.SyntaxSpan().End)

	// Optional UNITS
	var units string
	var unitsSpan types.Span
	unitsPresent := false
	if p.check(lexer.TokKwUnits) {
		p.advance()
		unitsPresent = true
		var qs string
		qs, unitsSpan, err = p.parseQuotedString()
		if err != nil {
			return nil, err
		}
		units = qs
	}

	// MAX-ACCESS or ACCESS (required)
	accessValue, accessKeyword, accessSpan, err := p.parseAccessClause()
	if err != nil {
		return nil, err
	}

	// STATUS (technically required but some vendor MIBs omit it)
	var statusValue types.Status
	var statusSpan types.Span
	if p.check(lexer.TokKwStatus) {
		statusValue, statusSpan, err = p.parseStatusClause()
		if err != nil {
			return nil, err
		}
	} else {
		statusValue = types.StatusCurrent
	}

	// DESCRIPTION (optional but common)
	var description string
	var descriptionSpan types.Span
	hasDescription := false
	if p.check(lexer.TokKwDescription) {
		p.advance()
		hasDescription = true
		description, descriptionSpan, err = p.parseQuotedString()
		if err != nil {
			return nil, err
		}
	}

	// REFERENCE (optional)
	var reference string
	var referenceSpan types.Span
	var referencePresent bool
	reference, referenceSpan, referencePresent, err = p.parseOptionalReference()
	if err != nil {
		return nil, err
	}

	// INDEX or AUGMENTS (optional)
	indexItems, augmentsTarget, augmentsSpan, indexSpan, err := p.parseIndexOrAugments()
	if err != nil {
		return nil, err
	}

	// DEFVAL (optional)
	var defval module.DefVal
	var defvalSpan types.Span
	if p.check(lexer.TokKwDefval) {
		defval, defvalSpan, err = p.parseDefValClause()
		if err != nil {
			return nil, err
		}
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

	// Empty string checks
	p.checkEmptyOptionalString(description, hasDescription, span, name, "DESCRIPTION", types.DiagEmptyDescription)
	p.checkEmptyOptionalString(reference, referencePresent, span, name, "REFERENCE", types.DiagEmptyReference)
	p.checkEmptyOptionalString(units, unitsPresent, span, name, "UNITS", types.DiagEmptyUnits)

	return &module.ObjectType{
		DefBase:        module.DefBase{Name: name, Span: span},
		Syntax:         syntax,
		Units:          units,
		Access:         accessValue,
		AccessKeyword:  accessKeyword,
		Status:         statusValue,
		Description:    description,
		HasDescription: hasDescription,
		Reference:      reference,
		Index:          indexItems,
		Augments:       augmentsTarget,
		DefVal:         defval,
		Oid:            oid,
		Spans: module.ObjectTypeSpans{
			Syntax:      syntaxSpan,
			Access:      accessSpan,
			Status:      statusSpan,
			Description: descriptionSpan,
			Units:       unitsSpan,
			Reference:   referenceSpan,
			Index:       indexSpan,
			Augments:    augmentsSpan,
			DefVal:      defvalSpan,
		},
	}, nil
}

// parseTypeSyntax parses a type expression (builtin types, type
// references, constrained types, SEQUENCE, CHOICE, etc.).
func (p *Parser) parseTypeSyntax() (module.TypeSyntax, *types.SpanDiagnostic) {
	start := p.currentSpan().Start

	var baseSyntax module.TypeSyntax

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
			baseSyntax = &module.TypeSyntaxIntegerEnum{
				NamedNumbers: namedNumbers,
				Span:         span,
			}
		} else {
			baseSyntax = &module.TypeSyntaxTypeRef{
				Name: "INTEGER",
				Span: types.NewSpan(start, p.lastEnd),
			}
		}

	case lexer.TokKwBits:
		p.advance()
		if p.check(lexer.TokLBrace) {
			p.advance()
			namedBits, err := p.parseNamedBitList()
			if err != nil {
				return nil, err
			}
			if _, err := p.expect(lexer.TokRBrace); err != nil {
				return nil, err
			}
			span := types.NewSpan(start, p.lastEnd)
			baseSyntax = &module.TypeSyntaxBits{
				NamedBits: namedBits,
				Span:      span,
			}
		} else {
			baseSyntax = &module.TypeSyntaxTypeRef{
				Name: "BITS",
				Span: types.NewSpan(start, p.lastEnd),
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
			baseSyntax = &module.TypeSyntaxConstrained{
				Base:       &module.TypeSyntaxOctetString{Span: span},
				Constraint: constraint,
				Span:       types.NewSpan(start, constraint.ConstraintSpan().End),
			}
		} else {
			baseSyntax = &module.TypeSyntaxOctetString{Span: span}
		}

	case lexer.TokKwObject:
		p.advance()
		if _, err := p.expect(lexer.TokKwIdentifier); err != nil {
			return nil, err
		}
		span := types.NewSpan(start, p.lastEnd)
		baseSyntax = &module.TypeSyntaxObjectIdentifier{Span: span}

	case lexer.TokKwSequence:
		p.advance()
		if p.check(lexer.TokKwOf) {
			// SEQUENCE OF EntryType
			p.advance()
			entryToken, err := p.expectIdentifier()
			if err != nil {
				return nil, err
			}
			entryType := p.text(entryToken.Span)
			span := types.NewSpan(start, entryToken.Span.End)
			baseSyntax = &module.TypeSyntaxSequenceOf{
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
			baseSyntax = &module.TypeSyntaxSequence{
				Fields: fields,
				Span:   span,
			}
		}

	case lexer.TokKwChoice:
		// CHOICE is normalized to the first alternative.
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
		choiceSpan := types.NewSpan(start, endToken.Span.End)

		if !module.IsBaseModule(p.moduleName) {
			p.EmitDiagnostic(types.DiagChoiceNotAllowed, choiceSpan,
				fmt.Sprintf("CHOICE type not allowed outside base module %q", p.moduleName))
		}
		if len(alternatives) > 0 {
			baseSyntax = alternatives[0].Syntax
		} else {
			baseSyntax = &module.TypeSyntaxOctetString{}
		}

	case lexer.TokKwCounter32, lexer.TokKwCounter64, lexer.TokKwGauge32,
		lexer.TokKwUnsigned32, lexer.TokKwTimeTicks, lexer.TokKwIpAddress,
		lexer.TokKwOpaque, lexer.TokKwCounter, lexer.TokKwGauge, lexer.TokKwNetworkAddress:
		token := p.advance()
		name := p.text(token.Span)
		baseSyntax = &module.TypeSyntaxTypeRef{
			Name: name,
			Span: token.Span,
		}

	case lexer.TokUppercaseIdent, lexer.TokLowercaseIdent:
		token := p.advance()
		name := p.text(token.Span)
		if token.Kind == lexer.TokLowercaseIdent {
			p.EmitDiagnostic(types.DiagBadIdentifierCase, token.Span,
				fmt.Sprintf("type reference %q should start with an uppercase letter", name))
		}

		switch {
		case p.check(lexer.TokLParen):
			constraint, err := p.parseConstraint()
			if err != nil {
				return nil, err
			}
			span := types.NewSpan(start, constraint.ConstraintSpan().End)
			baseSyntax = &module.TypeSyntaxConstrained{
				Base:       &module.TypeSyntaxTypeRef{Name: name, Span: token.Span},
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
			baseSyntax = &module.TypeSyntaxIntegerEnum{
				Base:         name,
				NamedNumbers: namedNumbers,
				Span:         span,
			}
		default:
			baseSyntax = &module.TypeSyntaxTypeRef{Name: name, Span: token.Span}
		}

	case lexer.TokLBracket:
		// Tagged type: [APPLICATION n] IMPLICIT Type
		// Normalized to the underlying type.
		p.advance() // consume '['
		// Skip optional tag class (APPLICATION, UNIVERSAL) and tag number
		if p.check(lexer.TokKwApplication) || p.check(lexer.TokKwUniversal) {
			p.advance()
		}
		if p.check(lexer.TokNumber) {
			p.advance()
		}
		if _, err := p.expect(lexer.TokRBracket); err != nil {
			return nil, err
		}
		// Skip optional IMPLICIT keyword
		if p.check(lexer.TokKwImplicit) {
			p.advance()
		}
		taggedSpan := types.NewSpan(start, p.lastEnd)

		if !module.IsBaseModule(p.moduleName) {
			p.EmitDiagnostic(types.DiagTaggedTypeNotAllowed, taggedSpan,
				fmt.Sprintf("tagged type not allowed outside base module %q", p.moduleName))
		}

		// Parse the underlying type and return it directly (normalization).
		underlying, err := p.parseTypeSyntax()
		if err != nil {
			return nil, err
		}
		baseSyntax = underlying

	default:
		diag := p.makeError("expected type syntax")
		return nil, &diag
	}

	// Check for constraint on the base syntax
	if p.check(lexer.TokLParen) {
		if _, ok := baseSyntax.(*module.TypeSyntaxConstrained); !ok {
			constraint, err := p.parseConstraint()
			if err != nil {
				return nil, err
			}
			span := types.NewSpan(start, constraint.ConstraintSpan().End)
			return &module.TypeSyntaxConstrained{
				Base:       baseSyntax,
				Constraint: constraint,
				Span:       span,
			}, nil
		}
	}

	return baseSyntax, nil
}

// parseNamedNumbers parses: { name(value), ... }
func (p *Parser) parseNamedNumbers() ([]module.NamedNumber, *types.SpanDiagnostic) {
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
// Includes enum duplicate validation.
func (p *Parser) parseNamedNumberList() ([]module.NamedNumber, *types.SpanDiagnostic) {
	var namedNumbers []module.NamedNumber
	seenNames := make(map[string]bool)
	seenValues := make(map[int64]bool)

	for !p.check(lexer.TokRBrace) && !p.isEOF() {
		start := p.currentSpan().Start
		nameToken, err := p.expectEnumLabel()
		if err != nil {
			return nil, err
		}
		name := p.text(nameToken.Span)

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
		value := p.convertI64(numToken.Span, "named number value")

		endToken, err := p.expect(lexer.TokRParen)
		if err != nil {
			return nil, err
		}
		span := types.NewSpan(start, endToken.Span.End)

		// Enum validation.
		if seenNames[name] {
			p.EmitDiagnostic(types.DiagEnumNameRedefinition, span,
				fmt.Sprintf("duplicate enum name %q", name))
		}
		seenNames[name] = true
		if seenValues[value] {
			p.EmitDiagnostic(types.DiagEnumValueRedefinition, span,
				fmt.Sprintf("duplicate enum value %d", value))
		}
		seenValues[value] = true
		if value == 0 && p.language == types.LanguageSMIv1 {
			p.EmitDiagnostic(types.DiagEnumZero, span,
				fmt.Sprintf("enumeration contains zero value %q(0) in SMIv1 module", name))
		}

		namedNumbers = append(namedNumbers, module.NewNamedNumber(name, value, span))

		if p.check(lexer.TokComma) {
			p.advance()
		} else if next := p.peek(); next.Kind.IsIdentifier() || next.Kind.IsKeyword() {
			// Recovery: missing comma before another named number.
			p.EmitDiagnostic(types.DiagMissingComma, next.Span, "missing comma in named number list")
		} else {
			break
		}
	}

	return namedNumbers, nil
}

// parseNamedBitList parses a list of named bit positions (without braces).
// Includes BITS duplicate and range validation.
func (p *Parser) parseNamedBitList() ([]module.NamedBit, *types.SpanDiagnostic) {
	var namedBits []module.NamedBit
	seenNames := make(map[string]bool)
	seenPositions := make(map[int64]bool)

	for !p.check(lexer.TokRBrace) && !p.isEOF() {
		start := p.currentSpan().Start
		nameToken, err := p.expectEnumLabel()
		if err != nil {
			return nil, err
		}
		name := p.text(nameToken.Span)

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
		value := p.convertI64(numToken.Span, "named number value")

		endToken, err := p.expect(lexer.TokRParen)
		if err != nil {
			return nil, err
		}
		span := types.NewSpan(start, endToken.Span.End)

		// BITS validation.
		if seenNames[name] {
			p.EmitDiagnostic(types.DiagBitsNameRedefinition, span,
				fmt.Sprintf("duplicate BITS name %q", name))
		}
		seenNames[name] = true
		if seenPositions[value] {
			p.EmitDiagnostic(types.DiagBitsValueRedefinition, span,
				fmt.Sprintf("duplicate BITS position %d", value))
		}
		seenPositions[value] = true

		pos := value
		switch {
		case pos < 0:
			p.EmitDiagnostic(types.DiagBitsNumberNegative, span,
				fmt.Sprintf("negative BITS position %d for %q", pos, name))
			pos = 0
		case pos >= 65535*8:
			p.EmitDiagnostic(types.DiagBitsNumberTooLarge, span,
				fmt.Sprintf("BITS position %d for %q exceeds maximum (%d)", pos, name, 65535*8-1))
			pos = 0
		case pos >= 128:
			p.EmitDiagnostic(types.DiagBitsNumberLarge, span,
				fmt.Sprintf("BITS position %d for %q may cause interoperability problems", pos, name))
		}

		namedBits = append(namedBits, module.NewNamedBit(name, uint32(pos), span))

		if p.check(lexer.TokComma) {
			p.advance()
		} else if next := p.peek(); next.Kind.IsIdentifier() || next.Kind.IsKeyword() {
			// Recovery: missing comma before another named number.
			p.EmitDiagnostic(types.DiagMissingComma, next.Span, "missing comma in named number list")
		} else {
			break
		}
	}

	return namedBits, nil
}

// parseConstraint parses: (SIZE (0..255)) or (0..65535)
func (p *Parser) parseConstraint() (module.Constraint, *types.SpanDiagnostic) {
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
		return &module.ConstraintSize{
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
	return &module.ConstraintRange{
		Ranges: ranges,
		Span:   types.NewSpan(start, endToken.Span.End),
	}, nil
}

// parseRangeList parses: 0..255 | 1024..65535
func (p *Parser) parseRangeList() ([]module.Range, *types.SpanDiagnostic) {
	var ranges []module.Range

	for {
		start := p.currentSpan().Start
		lo, err := p.parseRangeValue()
		if err != nil {
			return nil, err
		}

		var hi module.RangeValue
		if p.check(lexer.TokDotDot) {
			p.advance()
			hi, err = p.parseRangeValue()
			if err != nil {
				return nil, err
			}
		}

		ranges = append(ranges, module.Range{
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
func (p *Parser) parseRangeValue() (module.RangeValue, *types.SpanDiagnostic) {
	switch {
	case p.check(lexer.TokNumber):
		token := p.advance()
		text := p.text(token.Span)
		// Try parsing as u64 first to handle large unsigned values
		if value, err := strconv.ParseUint(text, 10, 64); err == nil {
			return &module.RangeValueUnsigned{Value: value}, nil
		}
		// Fallback to signed
		value := p.convertI64(token.Span, "range value")
		return &module.RangeValueSigned{Value: value}, nil
	case p.check(lexer.TokNegativeNumber):
		token := p.advance()
		value := p.convertI64(token.Span, "range value")
		return &module.RangeValueSigned{Value: value}, nil
	case p.check(lexer.TokHexString):
		token := p.advance()
		text := p.text(token.Span)
		// Parse hex string to unsigned
		hexPart := stripQuotedLiteral(text)
		value, err := strconv.ParseUint(hexPart, 16, 64)
		if err != nil {
			p.EmitDiagnostic(types.DiagInvalidHexRange, token.Span, "invalid hex value in range")
		}
		return &module.RangeValueUnsigned{Value: value}, nil
	case p.check(lexer.TokUppercaseIdent) || p.check(lexer.TokForbiddenKeyword):
		token := p.advance()
		name := p.text(token.Span)
		// Convert MIN/MAX identifiers to typed range values.
		switch name {
		case "MIN":
			return &module.RangeValueMin{}, nil
		case "MAX":
			return &module.RangeValueMax{}, nil
		default:
			p.EmitDiagnostic(types.DiagUnknownRangeValue, token.Span,
				fmt.Sprintf("unknown range identifier %s, defaulting to 0", name))
			return &module.RangeValueUnsigned{Value: 0}, nil
		}
	default:
		diag := p.makeError("expected range value")
		return nil, &diag
	}
}

// parseSequenceFields parses comma-separated name/type pairs within
// SEQUENCE { ... }.
func (p *Parser) parseSequenceFields() ([]module.SequenceField, *types.SpanDiagnostic) {
	var fields []module.SequenceField

	for !p.check(lexer.TokRBrace) && !p.isEOF() {
		start := p.currentSpan().Start
		nameToken, err := p.expectIdentifier()
		if err != nil {
			return nil, err
		}
		name := p.text(nameToken.Span)

		syntax, err := p.parseTypeSyntax()
		if err != nil {
			return nil, err
		}
		span := types.NewSpan(start, syntax.SyntaxSpan().End)

		fields = append(fields, module.NewSequenceField(name, syntax, span))

		if p.check(lexer.TokComma) {
			p.advance()
		}
	}

	return fields, nil
}

// parseChoiceAlternatives parses comma-separated name/type pairs within
// CHOICE { ... }. Identical structure to SEQUENCE fields.
func (p *Parser) parseChoiceAlternatives() ([]module.SequenceField, *types.SpanDiagnostic) {
	return p.parseSequenceFields()
}

// parseAccessClause parses ACCESS, MAX-ACCESS, or MIN-ACCESS with its value.
func (p *Parser) parseAccessClause() (types.Access, types.AccessKeyword, types.Span, *types.SpanDiagnostic) {
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
		return 0, 0, types.Span{}, &diag
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
		return 0, 0, types.Span{}, &diag
	}

	span := types.NewSpan(start, p.lastEnd)
	return value, keyword, span, nil
}

// parseStatusClause parses STATUS with its value keyword.
func (p *Parser) parseStatusClause() (types.Status, types.Span, *types.SpanDiagnostic) {
	start := p.currentSpan().Start
	if _, err := p.expect(lexer.TokKwStatus); err != nil {
		return 0, types.Span{}, err
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
		return 0, types.Span{}, &diag
	}

	span := types.NewSpan(start, p.lastEnd)
	return value, span, nil
}

// parseIndexOrAugments parses an optional INDEX or AUGMENTS clause.
func (p *Parser) parseIndexOrAugments() (indexItems []module.IndexItem, augmentsTarget string, augmentsSpan, indexSpan types.Span, err *types.SpanDiagnostic) {
	if p.check(lexer.TokKwIndex) {
		start := p.currentSpan().Start
		p.advance()
		if _, err := p.expect(lexer.TokLBrace); err != nil {
			return nil, "", types.Span{}, types.Span{}, err
		}

		var indexes []module.IndexItem
		for !p.check(lexer.TokRBrace) && !p.isEOF() {
			itemStart := p.currentSpan().Start
			implied := false
			if p.check(lexer.TokKwImplied) {
				p.advance()
				implied = true
			}

			objToken, err := p.expectIndexObject()
			if err != nil {
				return nil, "", types.Span{}, types.Span{}, err
			}

			// Combine "OCTET" + "STRING" into a single "OCTET STRING" index item.
			var objectName string
			var objectSpan types.Span
			if objToken.Kind == lexer.TokKwOctet && p.check(lexer.TokKwString) {
				strToken := p.advance()
				objectSpan = types.NewSpan(objToken.Span.Start, strToken.Span.End)
				objectName = "OCTET STRING"
			} else {
				objectName = p.text(objToken.Span)
				objectSpan = objToken.Span
			}

			span := types.NewSpan(itemStart, objectSpan.End)
			indexes = append(indexes, module.IndexItem{
				Implied: implied,
				Object:  objectName,
				Span:    span,
			})

			if p.check(lexer.TokComma) {
				p.advance()
			}
		}

		endToken, err := p.expect(lexer.TokRBrace)
		if err != nil {
			return nil, "", types.Span{}, types.Span{}, err
		}
		indexSpan := types.NewSpan(start, endToken.Span.End)
		return indexes, "", types.Span{}, indexSpan, nil
	} else if p.check(lexer.TokKwAugments) {
		start := p.currentSpan().Start
		p.advance()
		if _, err := p.expect(lexer.TokLBrace); err != nil {
			return nil, "", types.Span{}, types.Span{}, err
		}

		targetToken, err := p.expectIdentifier()
		if err != nil {
			return nil, "", types.Span{}, types.Span{}, err
		}
		targetName := p.text(targetToken.Span)

		endToken, err := p.expect(lexer.TokRBrace)
		if err != nil {
			return nil, "", types.Span{}, types.Span{}, err
		}
		augmentsSpan := types.NewSpan(start, endToken.Span.End)
		return nil, targetName, augmentsSpan, types.Span{}, nil
	}

	return nil, "", types.Span{}, types.Span{}, nil
}

// parseDefValClause parses: DEFVAL { content }.
func (p *Parser) parseDefValClause() (module.DefVal, types.Span, *types.SpanDiagnostic) {
	start := p.currentSpan().Start
	if _, err := p.expect(lexer.TokKwDefval); err != nil {
		return nil, types.Span{}, err
	}
	if _, err := p.expect(lexer.TokLBrace); err != nil {
		return nil, types.Span{}, err
	}

	value, err := p.parseDefVal()
	if err != nil {
		return nil, types.Span{}, err
	}

	endToken, err := p.expect(lexer.TokRBrace)
	if err != nil {
		return nil, types.Span{}, err
	}
	span := types.NewSpan(start, endToken.Span.End)

	return value, span, nil
}

// parseDefVal parses the value inside DEFVAL { ... } braces.
func (p *Parser) parseDefVal() (module.DefVal, *types.SpanDiagnostic) {
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
		name := p.text(token.Span)
		return &module.DefValEnum{Name: name}, nil
	case lexer.TokLBrace:
		return p.parseDefValBracedContent()
	default:
		// Keywords can be valid enum labels in DEFVAL (e.g., mandatory, optional, true, false)
		if kind.IsKeyword() {
			token := p.advance()
			name := p.text(token.Span)
			return &module.DefValEnum{Name: name}, nil
		}
		return p.parseDefValSkipUnknown(contentStart)
	}
}

func (p *Parser) parseDefValNumber() (module.DefVal, *types.SpanDiagnostic) {
	token := p.advance()
	if token.Kind == lexer.TokNegativeNumber {
		value := p.convertI64(token.Span, "DEFVAL integer")
		return &module.DefValInteger{Value: value}, nil
	}

	text := p.text(token.Span)
	if value, err := strconv.ParseInt(text, 10, 64); err == nil {
		return &module.DefValInteger{Value: value}, nil
	}
	if value, err := strconv.ParseUint(text, 10, 64); err == nil {
		return &module.DefValUnsigned{Value: value}, nil
	}
	value := p.convertI64(token.Span, "DEFVAL integer")
	return &module.DefValInteger{Value: value}, nil
}

func (p *Parser) parseDefValString() (module.DefVal, *types.SpanDiagnostic) {
	str, _, err := p.parseQuotedString()
	if err != nil {
		return nil, err
	}
	return &module.DefValString{Value: str}, nil
}

func (p *Parser) parseDefValHexString() (module.DefVal, *types.SpanDiagnostic) {
	token := p.advance()
	content := stripQuotedLiteral(p.text(token.Span))
	return &module.DefValHexString{Value: content}, nil
}

func (p *Parser) parseDefValBinaryString() (module.DefVal, *types.SpanDiagnostic) {
	token := p.advance()
	content := stripQuotedLiteral(p.text(token.Span))
	return &module.DefValBinaryString{Value: content}, nil
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

func (p *Parser) parseDefValBracedContent() (module.DefVal, *types.SpanDiagnostic) {
	p.advance() // consume opening brace
	innerStart := p.currentSpan().Start

	// Empty braces: BITS { {} }
	if p.check(lexer.TokRBrace) {
		p.advance()
		return &module.DefValBits{Labels: nil}, nil
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

func (p *Parser) parseDefValBracedIdent(innerStart types.ByteOffset) (module.DefVal, *types.SpanDiagnostic) {
	identToken := p.advance()
	identName := p.text(identToken.Span)

	if p.check(lexer.TokComma) || p.check(lexer.TokRBrace) {
		// This is BITS: { flag1, flag2 }
		return p.parseDefValBitsLabels(identName, innerStart)
	}
	// This is OID: { sysName 0 } or { iso 3 6 1 }
	return p.parseDefValOidWithFirstIdent(identName, identToken, innerStart)
}

func (p *Parser) parseDefValBitsLabels(first string, _ types.ByteOffset) (module.DefVal, *types.SpanDiagnostic) {
	labels := []string{first}
	for p.check(lexer.TokComma) {
		p.advance()
		kind := p.peek().Kind
		// Accept identifiers or keywords as BITS labels
		if kind.IsIdentifier() || kind.IsKeyword() {
			token := p.advance()
			labels = append(labels, p.text(token.Span))
		}
	}
	_, err := p.expect(lexer.TokRBrace)
	if err != nil {
		return nil, err
	}
	return &module.DefValBits{Labels: labels}, nil
}

func (p *Parser) parseDefValOidWithFirstIdent(identName string, identToken lexer.Token, innerStart types.ByteOffset) (module.DefVal, *types.SpanDiagnostic) {
	var components []module.OidComponent

	// First component is the identifier we already parsed
	if p.check(lexer.TokLParen) {
		p.advance()
		numToken, err := p.expect(lexer.TokNumber)
		if err != nil {
			return nil, err
		}
		number := p.convertU32(numToken.Span, "OID component")
		endParen, err := p.expect(lexer.TokRParen)
		if err != nil {
			return nil, err
		}
		components = append(components, &module.OidComponentNamedNumber{
			NameValue:   identName,
			NumberValue: number,
			Span:        types.NewSpan(identToken.Span.Start, endParen.Span.End),
		})
	} else {
		components = append(components, &module.OidComponentName{NameValue: identName, Span: identToken.Span})
	}

	// Parse remaining components and closing brace
	return p.finishDefValOid(components, innerStart)
}

func (p *Parser) parseDefValOidNumeric(innerStart types.ByteOffset) (module.DefVal, *types.SpanDiagnostic) {
	return p.finishDefValOid(nil, innerStart)
}

func (p *Parser) finishDefValOid(components []module.OidComponent, _ types.ByteOffset) (module.DefVal, *types.SpanDiagnostic) {
	components, err := p.parseDefValOidComponents(components)
	if err != nil {
		return nil, err
	}

	_, err = p.expect(lexer.TokRBrace)
	if err != nil {
		return nil, err
	}
	return &module.DefValOidValue{Components: components}, nil
}

func (p *Parser) parseDefValOidComponents(components []module.OidComponent) ([]module.OidComponent, *types.SpanDiagnostic) {
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

func (p *Parser) parseDefValSkipBraced(_ types.ByteOffset) (module.DefVal, *types.SpanDiagnostic) {
	p.skipBracedContent(false)
	_, err := p.expect(lexer.TokRBrace)
	if err != nil {
		return nil, err
	}
	return &module.DefValUnparsed{}, nil
}

func (p *Parser) parseDefValSkipUnknown(_ types.ByteOffset) (module.DefVal, *types.SpanDiagnostic) {
	depth := 0
	for !p.isEOF() {
		switch p.peek().Kind {
		case lexer.TokLBrace:
			depth++
			p.advance()
		case lexer.TokRBrace:
			if depth == 0 {
				return &module.DefValUnparsed{}, nil
			}
			depth--
			p.advance()
		default:
			p.advance()
		}
	}
	return &module.DefValUnparsed{}, nil
}

// parseQuotedString consumes a quoted string token and strips the quotes.
// Returns (value, span, error).
func (p *Parser) parseQuotedString() (string, types.Span, *types.SpanDiagnostic) {
	if !p.check(lexer.TokQuotedString) {
		diag := p.makeError("expected quoted string")
		return "", types.Span{}, &diag
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
	return value, token.Span, nil
}

// parseOptionalReference parses an optional REFERENCE clause.
func (p *Parser) parseOptionalReference() (value string, span types.Span, present bool, err *types.SpanDiagnostic) {
	if !p.check(lexer.TokKwReference) {
		return "", types.Span{}, false, nil
	}
	p.advance()
	value, span, err = p.parseQuotedString()
	if err != nil {
		return "", types.Span{}, false, err
	}
	return value, span, true, nil
}

// parseModuleIdentity parses a MODULE-IDENTITY macro invocation.
func (p *Parser) parseModuleIdentity() (module.Definition, *types.SpanDiagnostic) {
	start := p.currentSpan().Start

	nameToken := p.advance()
	name, _ := p.makeIdentStrWithValidation(nameToken)
	p.validateValueReference(name, nameToken.Span)

	if _, err := p.expect(lexer.TokKwModuleIdentity); err != nil {
		return nil, err
	}

	// LAST-UPDATED
	if _, err := p.expect(lexer.TokKwLastUpdated); err != nil {
		return nil, err
	}
	lastUpdated, _, err := p.parseQuotedString()
	if err != nil {
		return nil, err
	}

	// ORGANIZATION
	if _, err := p.expect(lexer.TokKwOrganization); err != nil {
		return nil, err
	}
	organization, _, err := p.parseQuotedString()
	if err != nil {
		return nil, err
	}

	// CONTACT-INFO
	if _, err := p.expect(lexer.TokKwContactInfo); err != nil {
		return nil, err
	}
	contactInfo, _, err := p.parseQuotedString()
	if err != nil {
		return nil, err
	}

	// DESCRIPTION
	if _, err := p.expect(lexer.TokKwDescription); err != nil {
		return nil, err
	}
	description, _, err := p.parseQuotedString()
	if err != nil {
		return nil, err
	}

	// REVISION clauses (optional, multiple)
	var revisions []module.Revision
	for p.check(lexer.TokKwRevision) {
		revStart := p.currentSpan().Start
		p.advance()
		date, _, err := p.parseQuotedString()
		if err != nil {
			return nil, err
		}
		if _, err := p.expect(lexer.TokKwDescription); err != nil {
			return nil, err
		}
		revDescription, revDescSpan, err := p.parseQuotedString()
		if err != nil {
			return nil, err
		}
		span := types.NewSpan(revStart, revDescSpan.End)
		p.checkEmptyRequiredString(revDescription, span, name, "REVISION DESCRIPTION", types.DiagEmptyDescription)
		revisions = append(revisions, module.Revision{
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

	// Empty string checks
	p.checkEmptyRequiredString(description, span, name, "DESCRIPTION", types.DiagEmptyDescription)
	p.checkEmptyRequiredString(organization, span, name, "ORGANIZATION", types.DiagEmptyOrganization)
	p.checkEmptyRequiredString(contactInfo, span, name, "CONTACT-INFO", types.DiagEmptyContact)

	return &module.ModuleIdentity{
		DefBase:      module.DefBase{Name: name, Span: span},
		LastUpdated:  lastUpdated,
		Organization: organization,
		ContactInfo:  contactInfo,
		Description:  description,
		Revisions:    revisions,
		Oid:          oid,
	}, nil
}

// parseObjectIdentity parses an OBJECT-IDENTITY macro invocation.
func (p *Parser) parseObjectIdentity() (module.Definition, *types.SpanDiagnostic) {
	start := p.currentSpan().Start

	nameToken := p.advance()
	name, _ := p.makeIdentStrWithValidation(nameToken)
	p.validateValueReference(name, nameToken.Span)

	if _, err := p.expect(lexer.TokKwObjectIdentity); err != nil {
		return nil, err
	}

	// STATUS
	statusValue, _, err := p.parseStatusClause()
	if err != nil {
		return nil, err
	}

	// DESCRIPTION
	if _, err := p.expect(lexer.TokKwDescription); err != nil {
		return nil, err
	}
	description, _, err := p.parseQuotedString()
	if err != nil {
		return nil, err
	}

	// REFERENCE (optional)
	reference, _, referencePresent, err := p.parseOptionalReference()
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

	// Empty string checks
	p.checkEmptyRequiredString(description, span, name, "DESCRIPTION", types.DiagEmptyDescription)
	p.checkEmptyOptionalString(reference, referencePresent, span, name, "REFERENCE", types.DiagEmptyReference)

	return &module.ObjectIdentity{
		DefBase:     module.DefBase{Name: name, Span: span},
		Status:      statusValue,
		Description: description,
		Reference:   reference,
		Oid:         oid,
	}, nil
}

// parseNotificationType parses a NOTIFICATION-TYPE macro invocation.
func (p *Parser) parseNotificationType() (module.Definition, *types.SpanDiagnostic) {
	start := p.currentSpan().Start

	nameToken := p.advance()
	name, _ := p.makeIdentStrWithValidation(nameToken)
	p.validateValueReference(name, nameToken.Span)

	if _, err := p.expect(lexer.TokKwNotificationType); err != nil {
		return nil, err
	}

	// OBJECTS (optional)
	var objects []string
	if p.check(lexer.TokKwObjects) {
		p.advance()
		objs, err := p.parseBracedIdentifierList()
		if err != nil {
			return nil, err
		}
		objects = objs
	}

	// STATUS
	statusValue, _, err := p.parseStatusClause()
	if err != nil {
		return nil, err
	}

	// DESCRIPTION
	if _, err := p.expect(lexer.TokKwDescription); err != nil {
		return nil, err
	}
	description, _, err := p.parseQuotedString()
	if err != nil {
		return nil, err
	}

	// REFERENCE (optional)
	reference, _, referencePresent, err := p.parseOptionalReference()
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

	// Empty string checks
	p.checkEmptyRequiredString(description, span, name, "DESCRIPTION", types.DiagEmptyDescription)
	p.checkEmptyOptionalString(reference, referencePresent, span, name, "REFERENCE", types.DiagEmptyReference)

	return &module.Notification{
		DefBase:        module.DefBase{Name: name, Span: span},
		Objects:        objects,
		Status:         statusValue,
		Description:    description,
		HasDescription: true, // NOTIFICATION-TYPE DESCRIPTION is required
		Reference:      reference,
		Oid:            &oid,
	}, nil
}

// parseTrapType parses a TRAP-TYPE macro invocation (SMIv1).
func (p *Parser) parseTrapType() (module.Definition, *types.SpanDiagnostic) {
	start := p.currentSpan().Start

	nameToken := p.advance()
	name, _ := p.makeIdentStrWithValidation(nameToken)
	p.validateValueReference(name, nameToken.Span)

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
	enterprise := p.text(enterpriseToken.Span)

	// VARIABLES (optional)
	var variables []string
	if p.check(lexer.TokKwVariables) {
		p.advance()
		vars, err := p.parseBracedIdentifierList()
		if err != nil {
			return nil, err
		}
		variables = vars
	}

	// DESCRIPTION (optional)
	var description string
	hasDescription := false
	if p.check(lexer.TokKwDescription) {
		p.advance()
		hasDescription = true
		description, _, err = p.parseQuotedString()
		if err != nil {
			return nil, err
		}
	}

	// REFERENCE (optional)
	var reference string
	var referencePresent bool
	reference, _, referencePresent, err = p.parseOptionalReference()
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
	trapNumber := p.convertU32(numToken.Span, "trap number")

	span := types.NewSpan(start, numToken.Span.End)

	// Empty string checks
	p.checkEmptyOptionalString(description, hasDescription, span, name, "DESCRIPTION", types.DiagEmptyDescription)
	p.checkEmptyOptionalString(reference, referencePresent, span, name, "REFERENCE", types.DiagEmptyReference)

	return &module.Notification{
		DefBase:        module.DefBase{Name: name, Span: span},
		Objects:        variables,
		Status:         types.StatusCurrent, // TRAP-TYPE has no STATUS clause
		Description:    description,
		HasDescription: hasDescription,
		Reference:      reference,
		TrapInfo: &module.TrapInfo{
			Enterprise: enterprise,
			TrapNumber: trapNumber,
		},
		// Oid is nil for TRAP-TYPE
	}, nil
}

// parseTextualConvention parses: Name TEXTUAL-CONVENTION ...
func (p *Parser) parseTextualConvention() (module.Definition, *types.SpanDiagnostic) {
	start := p.currentSpan().Start

	nameToken := p.advance()
	name, _ := p.makeIdentStrWithValidation(nameToken)

	if _, err := p.expect(lexer.TokKwTextualConvention); err != nil {
		return nil, err
	}

	return p.parseTextualConventionBody(name, start)
}

// parseTextualConventionWithAssignment parses the alternate form:
// Name ::= TEXTUAL-CONVENTION ...
func (p *Parser) parseTextualConventionWithAssignment() (module.Definition, *types.SpanDiagnostic) {
	start := p.currentSpan().Start

	nameToken := p.advance()
	name, _ := p.makeIdentStrWithValidation(nameToken)

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
func (p *Parser) parseTextualConventionBody(name string, start types.ByteOffset) (module.Definition, *types.SpanDiagnostic) {
	// DISPLAY-HINT (optional)
	var displayHint string
	var displayHintSpan types.Span
	displayHintPresent := false
	if p.check(lexer.TokKwDisplayHint) {
		p.advance()
		displayHintPresent = true
		var err *types.SpanDiagnostic
		displayHint, displayHintSpan, err = p.parseQuotedString()
		if err != nil {
			return nil, err
		}
	}

	// STATUS
	statusValue, statusSpan, err := p.parseStatusClause()
	if err != nil {
		return nil, err
	}

	// DESCRIPTION
	if _, err := p.expect(lexer.TokKwDescription); err != nil {
		return nil, err
	}
	description, descriptionSpan, err := p.parseQuotedString()
	if err != nil {
		return nil, err
	}

	// REFERENCE (optional)
	reference, referenceSpan, referencePresent, err := p.parseOptionalReference()
	if err != nil {
		return nil, err
	}

	// SYNTAX
	if _, err := p.expect(lexer.TokKwSyntax); err != nil {
		return nil, err
	}
	syntaxStart := p.currentSpan().Start
	syntax, err := p.parseTypeSyntax()
	if err != nil {
		return nil, err
	}
	syntaxSpan := types.NewSpan(syntaxStart, syntax.SyntaxSpan().End)

	span := types.NewSpan(start, syntaxSpan.End)

	// Empty string checks
	p.checkEmptyRequiredString(description, span, name, "DESCRIPTION", types.DiagEmptyDescription)
	p.checkEmptyOptionalString(reference, referencePresent, span, name, "REFERENCE", types.DiagEmptyReference)
	p.checkEmptyOptionalString(displayHint, displayHintPresent, span, name, "DISPLAY-HINT", types.DiagEmptyFormat)

	return &module.TypeDef{
		DefBase:             module.DefBase{Name: name, Span: span},
		Syntax:              syntax,
		DisplayHint:         displayHint,
		Status:              statusValue,
		Description:         description,
		Reference:           reference,
		IsTextualConvention: true,
		Spans: module.TypeDefSpans{
			Syntax:      syntaxSpan,
			Status:      statusSpan,
			Description: descriptionSpan,
			Reference:   referenceSpan,
			DisplayHint: displayHintSpan,
		},
	}, nil
}

// parseTypeAssignment parses: TypeName ::= TypeSyntax
func (p *Parser) parseTypeAssignment() (module.Definition, *types.SpanDiagnostic) {
	start := p.currentSpan().Start

	nameToken := p.advance()
	name, _ := p.makeIdentStrWithValidation(nameToken)

	if _, err := p.expect(lexer.TokColonColonEqual); err != nil {
		return nil, err
	}

	syntax, err := p.parseTypeSyntax()
	if err != nil {
		return nil, err
	}
	span := types.NewSpan(start, syntax.SyntaxSpan().End)

	return &module.TypeDef{
		DefBase: module.DefBase{Name: name, Span: span},
		Syntax:  syntax,
		Status:  types.StatusCurrent, // Default status for plain type assignments
		Spans:   module.TypeDefSpans{Syntax: syntax.SyntaxSpan()},
	}, nil
}

// parseObjectGroup parses an OBJECT-GROUP macro invocation.
func (p *Parser) parseObjectGroup() (module.Definition, *types.SpanDiagnostic) {
	start := p.currentSpan().Start

	nameToken := p.advance()
	name, _ := p.makeIdentStrWithValidation(nameToken)
	p.validateValueReference(name, nameToken.Span)

	if _, err := p.expect(lexer.TokKwObjectGroup); err != nil {
		return nil, err
	}

	if _, err := p.expect(lexer.TokKwObjects); err != nil {
		return nil, err
	}
	members, err := p.parseBracedIdentifierList()
	if err != nil {
		return nil, err
	}

	statusValue, _, err := p.parseStatusClause()
	if err != nil {
		return nil, err
	}

	if _, err := p.expect(lexer.TokKwDescription); err != nil {
		return nil, err
	}
	description, _, err := p.parseQuotedString()
	if err != nil {
		return nil, err
	}

	reference, _, referencePresent, err := p.parseOptionalReference()
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

	// Empty string checks
	p.checkEmptyRequiredString(description, span, name, "DESCRIPTION", types.DiagEmptyDescription)
	p.checkEmptyOptionalString(reference, referencePresent, span, name, "REFERENCE", types.DiagEmptyReference)

	return &module.ObjectGroup{
		DefBase:     module.DefBase{Name: name, Span: span},
		Objects:     members,
		Status:      statusValue,
		Description: description,
		Reference:   reference,
		Oid:         oid,
	}, nil
}

// parseNotificationGroup parses a NOTIFICATION-GROUP macro invocation.
func (p *Parser) parseNotificationGroup() (module.Definition, *types.SpanDiagnostic) {
	start := p.currentSpan().Start

	nameToken := p.advance()
	name, _ := p.makeIdentStrWithValidation(nameToken)
	p.validateValueReference(name, nameToken.Span)

	if _, err := p.expect(lexer.TokKwNotificationGroup); err != nil {
		return nil, err
	}

	if _, err := p.expect(lexer.TokKwNotifications); err != nil {
		return nil, err
	}
	members, err := p.parseBracedIdentifierList()
	if err != nil {
		return nil, err
	}

	statusValue, _, err := p.parseStatusClause()
	if err != nil {
		return nil, err
	}

	if _, err := p.expect(lexer.TokKwDescription); err != nil {
		return nil, err
	}
	description, _, err := p.parseQuotedString()
	if err != nil {
		return nil, err
	}

	reference, _, referencePresent, err := p.parseOptionalReference()
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

	// Empty string checks
	p.checkEmptyRequiredString(description, span, name, "DESCRIPTION", types.DiagEmptyDescription)
	p.checkEmptyOptionalString(reference, referencePresent, span, name, "REFERENCE", types.DiagEmptyReference)

	return &module.NotificationGroup{
		DefBase:       module.DefBase{Name: name, Span: span},
		Notifications: members,
		Status:        statusValue,
		Description:   description,
		Reference:     reference,
		Oid:           oid,
	}, nil
}

// parseModuleCompliance parses a MODULE-COMPLIANCE macro invocation.
func (p *Parser) parseModuleCompliance() (module.Definition, *types.SpanDiagnostic) {
	start := p.currentSpan().Start

	nameToken := p.advance()
	name, _ := p.makeIdentStrWithValidation(nameToken)
	p.validateValueReference(name, nameToken.Span)

	if _, err := p.expect(lexer.TokKwModuleCompliance); err != nil {
		return nil, err
	}

	// STATUS
	statusValue, _, err := p.parseStatusClause()
	if err != nil {
		return nil, err
	}

	// DESCRIPTION
	if _, err := p.expect(lexer.TokKwDescription); err != nil {
		return nil, err
	}
	description, _, err := p.parseQuotedString()
	if err != nil {
		return nil, err
	}

	// REFERENCE (optional)
	reference, _, referencePresent, err := p.parseOptionalReference()
	if err != nil {
		return nil, err
	}

	// Parse MODULE clauses
	var modules []module.ComplianceModule
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

	// Empty string checks
	p.checkEmptyRequiredString(description, span, name, "DESCRIPTION", types.DiagEmptyDescription)
	p.checkEmptyOptionalString(reference, referencePresent, span, name, "REFERENCE", types.DiagEmptyReference)

	return &module.ModuleCompliance{
		DefBase:     module.DefBase{Name: name, Span: span},
		Status:      statusValue,
		Description: description,
		Reference:   reference,
		Modules:     modules,
		Oid:         oid,
	}, nil
}

func (p *Parser) parseComplianceModule() (module.ComplianceModule, *types.SpanDiagnostic) {
	start := p.currentSpan().Start
	if _, err := p.expect(lexer.TokKwModule); err != nil {
		return module.ComplianceModule{}, err
	}

	// Optional module name
	var moduleName string
	if p.check(lexer.TokUppercaseIdent) {
		nameToken := p.advance()
		moduleName = p.text(nameToken.Span)
	}

	// Optional module OID (intentionally not preserved)
	if p.check(lexer.TokLBrace) {
		_, err := p.parseOidAssignment()
		if err != nil {
			return module.ComplianceModule{}, err
		}
	}

	// MANDATORY-GROUPS (optional)
	var mandatoryGroups []string
	if p.check(lexer.TokKwMandatoryGroups) {
		groups, err := p.parseMandatoryGroups()
		if err != nil {
			return module.ComplianceModule{}, err
		}
		mandatoryGroups = groups
	}

	// GROUP and OBJECT refinements
	var groups []module.ComplianceGroup
	var objects []module.ComplianceObject
	for p.check(lexer.TokKwGroup) || p.check(lexer.TokKwObject) {
		if p.check(lexer.TokKwGroup) {
			group, err := p.parseComplianceGroup()
			if err != nil {
				return module.ComplianceModule{}, err
			}
			groups = append(groups, *group)
		} else {
			obj, err := p.parseComplianceObject()
			if err != nil {
				return module.ComplianceModule{}, err
			}
			objects = append(objects, *obj)
		}
	}

	return module.ComplianceModule{
		ModuleName:      moduleName,
		MandatoryGroups: mandatoryGroups,
		Groups:          groups,
		Objects:         objects,
		Span:            types.NewSpan(start, p.lastEnd),
	}, nil
}

func (p *Parser) parseMandatoryGroups() ([]string, *types.SpanDiagnostic) {
	if _, err := p.expect(lexer.TokKwMandatoryGroups); err != nil {
		return nil, err
	}
	return p.parseBracedIdentifierList()
}

func (p *Parser) parseComplianceGroup() (*module.ComplianceGroup, *types.SpanDiagnostic) {
	start := p.currentSpan().Start
	if _, err := p.expect(lexer.TokKwGroup); err != nil {
		return nil, err
	}
	groupName, err := p.parseIdentifierAsString()
	if err != nil {
		return nil, err
	}
	if _, err := p.expect(lexer.TokKwDescription); err != nil {
		return nil, err
	}
	description, descSpan, err := p.parseQuotedString()
	if err != nil {
		return nil, err
	}
	end := descSpan.End
	return &module.ComplianceGroup{
		Group:       groupName,
		Description: description,
		Span:        types.NewSpan(start, end),
	}, nil
}

// parseOptionalSyntaxClauses parses optional SYNTAX and WRITE-SYNTAX clauses,
// shared by MODULE-COMPLIANCE objects and AGENT-CAPABILITIES variations.
func (p *Parser) parseOptionalSyntaxClauses() (syntax, writeSyntax module.TypeSyntax, err *types.SpanDiagnostic) {
	if p.check(lexer.TokKwSyntax) {
		p.advance()
		s, e := p.parseTypeSyntax()
		if e != nil {
			return nil, nil, e
		}
		syntax = s
	}
	if p.check(lexer.TokKwWriteSyntax) {
		p.advance()
		s, e := p.parseTypeSyntax()
		if e != nil {
			return nil, nil, e
		}
		writeSyntax = s
	}
	return syntax, writeSyntax, nil
}

func (p *Parser) parseComplianceObject() (*module.ComplianceObject, *types.SpanDiagnostic) {
	start := p.currentSpan().Start
	if _, err := p.expect(lexer.TokKwObject); err != nil {
		return nil, err
	}
	objectName, err := p.parseIdentifierAsString()
	if err != nil {
		return nil, err
	}

	syntax, writeSyntax, err := p.parseOptionalSyntaxClauses()
	if err != nil {
		return nil, err
	}

	// Optional MIN-ACCESS
	var minAccess *types.Access
	if p.check(lexer.TokKwMinAccess) {
		accessValue, _, _, err := p.parseAccessClause()
		if err != nil {
			return nil, err
		}
		minAccess = &accessValue
	}

	// Required DESCRIPTION
	if _, err := p.expect(lexer.TokKwDescription); err != nil {
		return nil, err
	}
	description, descSpan, err := p.parseQuotedString()
	if err != nil {
		return nil, err
	}

	end := descSpan.End
	return &module.ComplianceObject{
		Object:      objectName,
		Syntax:      syntax,
		WriteSyntax: writeSyntax,
		MinAccess:   minAccess,
		Description: description,
		Span:        types.NewSpan(start, end),
	}, nil
}

func (p *Parser) parseIdentifierAsString() (string, *types.SpanDiagnostic) {
	token, err := p.expectIdentifier()
	if err != nil {
		return "", err
	}
	return p.text(token.Span), nil
}

// parseAgentCapabilities parses an AGENT-CAPABILITIES macro invocation.
func (p *Parser) parseAgentCapabilities() (module.Definition, *types.SpanDiagnostic) {
	start := p.currentSpan().Start

	nameToken := p.advance()
	name, _ := p.makeIdentStrWithValidation(nameToken)
	p.validateValueReference(name, nameToken.Span)

	if _, err := p.expect(lexer.TokKwAgentCapabilities); err != nil {
		return nil, err
	}

	// PRODUCT-RELEASE
	if _, err := p.expect(lexer.TokKwProductRelease); err != nil {
		return nil, err
	}
	productRelease, _, err := p.parseQuotedString()
	if err != nil {
		return nil, err
	}

	// STATUS
	statusValue, _, err := p.parseStatusClause()
	if err != nil {
		return nil, err
	}

	// DESCRIPTION
	if _, err := p.expect(lexer.TokKwDescription); err != nil {
		return nil, err
	}
	description, _, err := p.parseQuotedString()
	if err != nil {
		return nil, err
	}

	// REFERENCE (optional)
	reference, _, referencePresent, err := p.parseOptionalReference()
	if err != nil {
		return nil, err
	}

	// Parse SUPPORTS clauses
	var supports []module.SupportsModule
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

	// Empty string checks
	p.checkEmptyRequiredString(description, span, name, "DESCRIPTION", types.DiagEmptyDescription)
	p.checkEmptyOptionalString(reference, referencePresent, span, name, "REFERENCE", types.DiagEmptyReference)

	return &module.AgentCapabilities{
		DefBase:        module.DefBase{Name: name, Span: span},
		ProductRelease: productRelease,
		Status:         statusValue,
		Description:    description,
		Reference:      reference,
		Supports:       supports,
		Oid:            oid,
	}, nil
}

func (p *Parser) parseSupportsModule() (module.SupportsModule, *types.SpanDiagnostic) {
	start := p.currentSpan().Start
	if _, err := p.expect(lexer.TokKwSupports); err != nil {
		return module.SupportsModule{}, err
	}

	// Module name
	moduleName, err := p.parseIdentifierAsString()
	if err != nil {
		return module.SupportsModule{}, err
	}

	// Optional module OID (intentionally not preserved)
	if p.check(lexer.TokLBrace) {
		_, err := p.parseOidAssignment()
		if err != nil {
			return module.SupportsModule{}, err
		}
	}

	// INCLUDES { groups }
	if _, err := p.expect(lexer.TokKwIncludes); err != nil {
		return module.SupportsModule{}, err
	}
	includes, err := p.parseBracedIdentifierList()
	if err != nil {
		return module.SupportsModule{}, err
	}

	// VARIATION clauses
	var variations []module.Variation
	for p.check(lexer.TokKwVariation) {
		v, err := p.parseVariationClause()
		if err != nil {
			return module.SupportsModule{}, err
		}
		variations = append(variations, *v)
	}

	return module.SupportsModule{
		ModuleName: moduleName,
		Includes:   includes,
		Variations: variations,
		Span:       types.NewSpan(start, p.lastEnd),
	}, nil
}

func (p *Parser) parseVariationClause() (*module.Variation, *types.SpanDiagnostic) {
	start := p.currentSpan().Start
	if _, err := p.expect(lexer.TokKwVariation); err != nil {
		return nil, err
	}

	// Object or notification name
	varName, err := p.parseIdentifierAsString()
	if err != nil {
		return nil, err
	}

	syntax, writeSyntax, err := p.parseOptionalSyntaxClauses()
	if err != nil {
		return nil, err
	}

	// Optional ACCESS
	var access *types.Access
	if p.check(lexer.TokKwAccess) {
		accessValue, _, _, err := p.parseAccessClause()
		if err != nil {
			return nil, err
		}
		access = &accessValue
	}

	// Optional CREATION-REQUIRES
	var creationRequires []string
	if p.check(lexer.TokKwCreationRequires) {
		p.advance()
		objs, err := p.parseBracedIdentifierList()
		if err != nil {
			return nil, err
		}
		creationRequires = objs
	}

	// Optional DEFVAL
	var defval module.DefVal
	if p.check(lexer.TokKwDefval) {
		dv, _, err := p.parseDefValClause()
		if err != nil {
			return nil, err
		}
		defval = dv
	}

	// Required DESCRIPTION
	if _, err := p.expect(lexer.TokKwDescription); err != nil {
		return nil, err
	}
	description, descSpan, err := p.parseQuotedString()
	if err != nil {
		return nil, err
	}

	end := descSpan.End

	return &module.Variation{
		Name:             varName,
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
// to END. Returns nil (MACRO definitions are not semantic definitions).
func (p *Parser) parseMacroDefinition() (module.Definition, *types.SpanDiagnostic) {
	nameToken := p.advance()
	name := p.text(nameToken.Span)

	if _, err := p.expect(lexer.TokKwMacro); err != nil {
		return nil, err
	}

	for !p.check(lexer.TokKwEnd) && !p.isEOF() {
		p.advance()
	}

	if p.check(lexer.TokKwEnd) {
		p.advance()
	} else {
		diag := p.makeError("expected END for MACRO")
		return nil, &diag
	}

	if !module.IsBaseModule(p.moduleName) {
		span := types.NewSpan(nameToken.Span.Start, p.lastEnd)
		p.EmitDiagnostic(types.DiagMacroNotAllowed, span,
			fmt.Sprintf("MACRO definition %q not allowed outside base modules", name))
	}

	return nil, nil
}

// parseIdentifierList parses a comma-separated list of identifiers.
func (p *Parser) parseIdentifierList() ([]string, *types.SpanDiagnostic) {
	var names []string

	for !p.check(lexer.TokRBrace) && !p.isEOF() {
		token, err := p.expectIdentifier()
		if err != nil {
			return nil, err
		}
		names = append(names, p.text(token.Span))

		if p.check(lexer.TokComma) {
			p.advance()
		} else {
			break
		}
	}

	return names, nil
}

func (p *Parser) parseBracedIdentifierList() ([]string, *types.SpanDiagnostic) {
	if _, err := p.expect(lexer.TokLBrace); err != nil {
		return nil, err
	}
	names, err := p.parseIdentifierList()
	if err != nil {
		return nil, err
	}
	if _, err := p.expect(lexer.TokRBrace); err != nil {
		return nil, err
	}
	return names, nil
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
			(current.IsIdentifier() && next == lexer.TokColonColonEqual) ||
			(current == lexer.TokUppercaseIdent && next == lexer.TokKwTextualConvention) ||
			(current == lexer.TokUppercaseIdent && next == lexer.TokKwMacro) ||
			(current.IsIdentifier() && next == lexer.TokKwObject &&
				p.peekNth(2).Kind == lexer.TokKwIdentifier) {
			break
		}

		p.advance()
	}
}
