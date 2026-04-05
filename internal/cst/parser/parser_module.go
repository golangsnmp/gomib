package parser

import (
	"fmt"

	"github.com/golangsnmp/gomib/internal/cst"
	"github.com/golangsnmp/gomib/internal/lexer"
	"github.com/golangsnmp/gomib/internal/types"
)

// ParseModule parses all MIB modules in the source and returns the CST root.
func (p *Parser) ParseModule() *cst.ModuleFile {
	file := &cst.ModuleFile{}

	for !p.isEOF() {
		mod, ok := p.parseOneModule()
		file.Modules = append(file.Modules, mod)
		if !ok {
			break
		}
	}

	if len(file.Modules) == 0 {
		mod, _ := p.parseOneModule()
		file.Modules = append(file.Modules, mod)
	}

	// Consume EOF with trivia (captures trailing whitespace/comments).
	file.EOF = p.advance()

	if len(file.Modules) > 0 {
		file.Span = types.NewSpan(
			file.Modules[0].Span.Start,
			file.EOF.Span.End,
		)
	} else {
		file.Span = file.EOF.Span
	}

	return file
}

// parseOneModule parses a single DEFINITIONS ::= BEGIN ... END block.
func (p *Parser) parseOneModule() (cst.ModuleNode, bool) {
	var mod cst.ModuleNode
	start := p.currentSpan().Start

	// Module name
	nameTok, err := p.expectIdentifier()
	if err != nil {
		p.recordParseError(*err)
		mod.Name = nameTok
		mod.Span = types.NewSpan(start, p.currentSpan().End)
		return mod, false
	}
	mod.Name = nameTok
	name := p.text(nameTok.Span)
	p.validateIdentifier(name, nameTok.Span)

	// Skip obsolete module OID that some MIBs include before DEFINITIONS.
	if p.check(lexer.TokLBrace) {
		p.advance() // consume opening brace
		p.skipBracedContent(true)
	}

	// DEFINITIONS
	defTok, err := p.expect(lexer.TokKwDefinitions)
	if err != nil {
		p.recordParseError(*err)
		mod.Span = types.NewSpan(start, p.currentSpan().End)
		return mod, false
	}
	mod.Definitions = defTok

	// ::=
	assignTok, err := p.expect(lexer.TokColonColonEqual)
	if err != nil {
		p.recordParseError(*err)
		mod.Span = types.NewSpan(start, p.currentSpan().End)
		return mod, false
	}
	mod.Assign = assignTok

	// BEGIN
	beginTok, err := p.expect(lexer.TokKwBegin)
	if err != nil {
		p.recordParseError(*err)
		mod.Span = types.NewSpan(start, p.currentSpan().End)
		return mod, false
	}
	mod.Begin = beginTok

	// IMPORTS
	if p.check(lexer.TokKwImports) {
		imports, parseErr := p.parseImports()
		if parseErr != nil {
			p.recordParseError(*parseErr)
		}
		mod.Imports = imports
	}

	// Definition body
	for !p.check(lexer.TokKwEnd) && !p.isEOF() {
		posBefore := p.currentSpan().Start
		def, parseErr := p.parseDefinition()
		if parseErr != nil {
			p.recordParseError(*parseErr)
			skipped := p.recoverToDefinition()
			if len(skipped) > 0 {
				errNode := &cst.ErrorNode{
					Tokens: skipped,
					Span:   types.NewSpan(skipped[0].Span.Start, skipped[len(skipped)-1].Span.End),
				}
				mod.Body = append(mod.Body, errNode)
			}
			// Force progress if recovery didn't advance.
			if p.currentSpan().Start == posBefore {
				tok := p.advance()
				errNode := &cst.ErrorNode{
					Tokens: []cst.SyntaxToken{tok},
					Span:   tok.Span,
				}
				mod.Body = append(mod.Body, errNode)
			}
		} else if def != nil {
			mod.Body = append(mod.Body, def)
		}
	}

	// END
	if p.check(lexer.TokKwEnd) {
		mod.End = p.advance()
	} else {
		p.recordParseError(p.makeError("expected END"))
	}

	mod.Span = types.NewSpan(start, p.lastEnd)
	return mod, true
}

// parseImports parses: IMPORTS symbols FROM Module ... ;
func (p *Parser) parseImports() (*cst.ImportsNode, *types.SpanDiagnostic) {
	node := &cst.ImportsNode{}
	start := p.currentSpan().Start

	kwTok, err := p.expect(lexer.TokKwImports)
	if err != nil {
		return nil, err
	}
	node.Keyword = kwTok

	for {
		if p.check(lexer.TokSemicolon) {
			node.Semi = p.advance()
			break
		}

		if p.isEOF() || p.check(lexer.TokKwEnd) {
			diag := p.makeError("unexpected end of imports")
			node.Span = types.NewSpan(start, p.lastEnd)
			return node, &diag
		}

		group, err := p.parseImportGroup()
		if err != nil {
			node.Span = types.NewSpan(start, p.lastEnd)
			return node, err
		}
		node.Groups = append(node.Groups, group)
	}

	node.Span = types.NewSpan(start, p.lastEnd)
	return node, nil
}

// parseImportGroup parses: symbol, symbol, ... FROM ModuleName
func (p *Parser) parseImportGroup() (cst.ImportGroupNode, *types.SpanDiagnostic) {
	var group cst.ImportGroupNode
	start := p.currentSpan().Start

	// Collect symbols separated by commas until FROM.
	// commas[i] separates symbols[i] and symbols[i+1].
	for {
		kind := p.peek().Kind
		switch {
		case kind.IsMacroKeyword() || kind.IsTypeKeyword() || kind.IsIdentifier():
			symTok := p.advance()
			group.Symbols = append(group.Symbols, symTok)
		case p.check(lexer.TokKwFrom):
			goto doneSymbols
		default:
			diag := p.makeError("expected symbol or FROM")
			group.Span = types.NewSpan(start, p.lastEnd)
			return group, &diag
		}

		if p.check(lexer.TokComma) {
			group.Commas = append(group.Commas, p.advance())
		} else if !p.check(lexer.TokKwFrom) {
			// No comma and not FROM: missing separator.
			diag := p.makeError("expected ',' or FROM")
			group.Span = types.NewSpan(start, p.lastEnd)
			return group, &diag
		}
	}

doneSymbols:
	fromTok, err := p.expect(lexer.TokKwFrom)
	if err != nil {
		group.Span = types.NewSpan(start, p.lastEnd)
		return group, err
	}
	group.From = fromTok

	if !p.check(lexer.TokUppercaseIdent) {
		diag := p.makeError("expected module name after FROM")
		group.Span = types.NewSpan(start, p.lastEnd)
		return group, &diag
	}
	group.Module = p.advance()
	group.Span = types.NewSpan(start, p.lastEnd)

	return group, nil
}

// parseDefinition dispatches to the appropriate definition parser based on
// lookahead tokens.
func (p *Parser) parseDefinition() (cst.DefinitionNode, *types.SpanDiagnostic) {
	first := p.peek().Kind
	second := p.peekNth(1).Kind

	switch {
	// MACRO definition: name MACRO (lexer skips body until END).
	// Must be checked early because the name may be a macro keyword
	// (e.g. MODULE-IDENTITY MACRO in base modules).
	case second == lexer.TokKwMacro && (first.IsIdentifier() || first.IsMacroKeyword()):
		return p.parseMacroDefinition()

	// Value assignment: name OBJECT IDENTIFIER ::=
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

	// TEXTUAL-CONVENTION (Name TEXTUAL-CONVENTION ...)
	case first == lexer.TokUppercaseIdent && second == lexer.TokKwTextualConvention:
		return p.parseTextualConvention(false)

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

	// Type assignment or TEXTUAL-CONVENTION with ::=
	// Accept identifiers and type keywords as names (type keywords like
	// IpAddress, Counter32 etc. appear as type assignment names in base modules).
	case (first.IsIdentifier() || first.IsTypeKeyword()) && second == lexer.TokColonColonEqual:
		// Name ::= TEXTUAL-CONVENTION ...
		if p.peekNth(2).Kind == lexer.TokKwTextualConvention {
			return p.parseTextualConvention(true)
		}
		if first == lexer.TokLowercaseIdent {
			name := p.text(p.peek().Span)
			p.EmitDiagnostic(types.DiagBadIdentifierCase, p.peek().Span,
				fmt.Sprintf("type assignment %q should start with an uppercase letter", name))
		}
		return p.parseTypeAssignment()

	// EXPORTS (skipped by lexer, but we may see the semicolon)
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

// parseBracedIdentifierList parses { name, name, ... }
func (p *Parser) parseBracedIdentifierList() (lbrace cst.SyntaxToken, names, commas []cst.SyntaxToken, rbrace cst.SyntaxToken, parseErr *types.SpanDiagnostic) {
	lbrace, parseErr = p.expect(lexer.TokLBrace)
	if parseErr != nil {
		return lbrace, nil, nil, rbrace, parseErr
	}

	for !p.check(lexer.TokRBrace) && !p.isEOF() {
		tok, err := p.expectIdentifier()
		if err != nil {
			return lbrace, names, commas, rbrace, err
		}
		names = append(names, tok)

		if p.check(lexer.TokComma) {
			commas = append(commas, p.advance())
		} else {
			break
		}
	}

	rbrace, parseErr = p.expect(lexer.TokRBrace)
	return lbrace, names, commas, rbrace, parseErr
}
