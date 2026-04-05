package parser

import (
	"github.com/golangsnmp/gomib/internal/cst"
	"github.com/golangsnmp/gomib/internal/lexer"
	"github.com/golangsnmp/gomib/internal/types"
)

// parseOidValue parses: { component component ... }
func (p *Parser) parseOidValue() (*cst.OidValueNode, *types.SpanDiagnostic) {
	start := p.currentSpan().Start
	node := &cst.OidValueNode{}

	lbrace, err := p.expect(lexer.TokLBrace)
	if err != nil {
		return nil, err
	}
	node.LBrace = lbrace

	for !p.check(lexer.TokRBrace) && !p.isEOF() {
		kind := p.peek().Kind
		if kind == lexer.TokNumber || kind == lexer.TokLowercaseIdent || kind == lexer.TokUppercaseIdent {
			comp, err := p.parseOidComponent()
			if err != nil {
				return nil, err
			}
			node.Components = append(node.Components, comp)
		} else {
			diag := p.makeError("expected OID component")
			return nil, &diag
		}
	}

	rbrace, err := p.expect(lexer.TokRBrace)
	if err != nil {
		return nil, err
	}
	node.RBrace = rbrace

	node.Span = types.NewSpan(start, p.lastEnd)
	return node, nil
}

// parseOidComponent parses a single OID component:
//   - number
//   - name
//   - name(number)
//   - Module.name
//   - Module.name(number)
func (p *Parser) parseOidComponent() (cst.OidComponentNode, *types.SpanDiagnostic) {
	start := p.currentSpan().Start

	if p.check(lexer.TokNumber) {
		numTok := p.advance()
		return cst.OidComponentNode{
			Number: numTok,
			Span:   types.NewSpan(start, p.lastEnd),
		}, nil
	}

	firstTok := p.advance() // identifier

	// Qualified reference: Module.name or Module.name(number)
	if p.check(lexer.TokDot) {
		dotTok := p.advance()

		nameTok, err := p.expect(lexer.TokLowercaseIdent)
		if err != nil {
			return cst.OidComponentNode{}, err
		}

		if p.check(lexer.TokLParen) {
			lparen := p.advance()
			numTok, err := p.expect(lexer.TokNumber)
			if err != nil {
				return cst.OidComponentNode{}, err
			}
			rparen, err := p.expect(lexer.TokRParen)
			if err != nil {
				return cst.OidComponentNode{}, err
			}
			return cst.OidComponentNode{
				Module: firstTok,
				Dot:    dotTok,
				Name:   nameTok,
				LParen: lparen,
				Number: numTok,
				RParen: rparen,
				Span:   types.NewSpan(start, p.lastEnd),
			}, nil
		}

		return cst.OidComponentNode{
			Module: firstTok,
			Dot:    dotTok,
			Name:   nameTok,
			Span:   types.NewSpan(start, p.lastEnd),
		}, nil
	}

	// Named number: name(number)
	if p.check(lexer.TokLParen) {
		lparen := p.advance()
		numTok, err := p.expect(lexer.TokNumber)
		if err != nil {
			return cst.OidComponentNode{}, err
		}
		rparen, err := p.expect(lexer.TokRParen)
		if err != nil {
			return cst.OidComponentNode{}, err
		}
		return cst.OidComponentNode{
			Name:   firstTok,
			LParen: lparen,
			Number: numTok,
			RParen: rparen,
			Span:   types.NewSpan(start, p.lastEnd),
		}, nil
	}

	// Just name
	return cst.OidComponentNode{
		Name: firstTok,
		Span: types.NewSpan(start, p.lastEnd),
	}, nil
}
