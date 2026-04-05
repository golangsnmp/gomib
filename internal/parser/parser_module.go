package parser

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/golangsnmp/gomib/internal/lexer"
	"github.com/golangsnmp/gomib/internal/module"
	"github.com/golangsnmp/gomib/internal/types"
)

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
	if len(mod.ModuleIdentities) > 0 {
		mod.LastUpdated = strings.TrimSpace(mod.ModuleIdentities[0].LastUpdated)
	}

	mod.Diagnostics = p.convertDiagnostics(p.collectSpanDiagnostics(lexDiagsBefore), name, lineTable)

	p.Log(slog.LevelDebug, "parsing complete",
		slog.String("module", name),
		slog.Int("definitions", mod.DefinitionCount()),
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

func (p *Parser) parseIdentifierAsString() (string, *types.SpanDiagnostic) {
	token, err := p.expectIdentifier()
	if err != nil {
		return "", err
	}
	return p.text(token.Span), nil
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
