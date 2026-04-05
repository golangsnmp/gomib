package parser

import (
	"github.com/golangsnmp/gomib/internal/lexer"
	"github.com/golangsnmp/gomib/internal/module"
	"github.com/golangsnmp/gomib/internal/types"
)

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
