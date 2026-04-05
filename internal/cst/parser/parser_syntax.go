package parser

import (
	"github.com/golangsnmp/gomib/internal/cst"
	"github.com/golangsnmp/gomib/internal/lexer"
	"github.com/golangsnmp/gomib/internal/types"
)

// parseTypeSyntax parses a type expression: builtin types, type references,
// constrained types, SEQUENCE, CHOICE, tagged types, etc.
func (p *Parser) parseTypeSyntax() (cst.TypeSyntaxNode, *types.SpanDiagnostic) {
	start := p.currentSpan().Start
	var baseSyntax cst.TypeSyntaxNode

	switch p.peek().Kind {
	case lexer.TokKwInteger:
		intTok := p.advance()
		if p.check(lexer.TokLBrace) {
			// INTEGER { name(val), ... }
			lbrace, values, commas, rbrace, err := p.parseEnumValues()
			if err != nil {
				return nil, err
			}
			baseSyntax = &cst.IntegerEnumSyntaxNode{
				Base:   intTok,
				LBrace: lbrace,
				Values: values,
				Commas: commas,
				RBrace: rbrace,
				Span:   types.NewSpan(start, p.lastEnd),
			}
		} else {
			baseSyntax = &cst.TypeRefSyntaxNode{
				Name: intTok,
				Span: intTok.Span,
			}
		}

	case lexer.TokKwBits:
		bitsTok := p.advance()
		if p.check(lexer.TokLBrace) {
			lbrace := p.advance()
			values, commas, err := p.parseEnumValueList()
			if err != nil {
				return nil, err
			}
			rbrace, err := p.expect(lexer.TokRBrace)
			if err != nil {
				return nil, err
			}
			baseSyntax = &cst.BitsSyntaxNode{
				Keyword: bitsTok,
				LBrace:  lbrace,
				Values:  values,
				Commas:  commas,
				RBrace:  rbrace,
				Span:    types.NewSpan(start, p.lastEnd),
			}
		} else {
			baseSyntax = &cst.TypeRefSyntaxNode{
				Name: bitsTok,
				Span: bitsTok.Span,
			}
		}

	case lexer.TokKwOctet:
		octetTok := p.advance()
		stringTok, err := p.expect(lexer.TokKwString)
		if err != nil {
			return nil, err
		}
		osNode := &cst.OctetStringSyntaxNode{
			Octet:  octetTok,
			String: stringTok,
			Span:   types.NewSpan(start, p.lastEnd),
		}
		if p.check(lexer.TokLParen) {
			constraint, err := p.parseConstraint()
			if err != nil {
				return nil, err
			}
			baseSyntax = &cst.ConstrainedSyntaxNode{
				Base:       osNode,
				Constraint: constraint,
				Span:       types.NewSpan(start, p.lastEnd),
			}
		} else {
			baseSyntax = osNode
		}

	case lexer.TokKwObject:
		objTok := p.advance()
		idTok, err := p.expect(lexer.TokKwIdentifier)
		if err != nil {
			return nil, err
		}
		baseSyntax = &cst.ObjectIdentifierSyntaxNode{
			Object:     objTok,
			Identifier: idTok,
			Span:       types.NewSpan(start, p.lastEnd),
		}

	case lexer.TokKwSequence:
		seqTok := p.advance()
		if p.check(lexer.TokKwOf) {
			ofTok := p.advance()
			entryTok, err := p.expectIdentifier()
			if err != nil {
				return nil, err
			}
			baseSyntax = &cst.SequenceOfSyntaxNode{
				Sequence: seqTok,
				Of:       ofTok,
				Entry:    entryTok,
				Span:     types.NewSpan(start, p.lastEnd),
			}
		} else {
			lbrace, err := p.expect(lexer.TokLBrace)
			if err != nil {
				return nil, err
			}
			fields, commas, err := p.parseSequenceFields()
			if err != nil {
				return nil, err
			}
			rbrace, err := p.expect(lexer.TokRBrace)
			if err != nil {
				return nil, err
			}
			baseSyntax = &cst.SequenceSyntaxNode{
				Keyword: seqTok,
				LBrace:  lbrace,
				Fields:  fields,
				Commas:  commas,
				RBrace:  rbrace,
				Span:    types.NewSpan(start, p.lastEnd),
			}
		}

	case lexer.TokKwChoice:
		choiceTok := p.advance()
		lbrace, err := p.expect(lexer.TokLBrace)
		if err != nil {
			return nil, err
		}
		fields, commas, err := p.parseSequenceFields()
		if err != nil {
			return nil, err
		}
		rbrace, err := p.expect(lexer.TokRBrace)
		if err != nil {
			return nil, err
		}
		baseSyntax = &cst.ChoiceSyntaxNode{
			Keyword: choiceTok,
			LBrace:  lbrace,
			Fields:  fields,
			Commas:  commas,
			RBrace:  rbrace,
			Span:    types.NewSpan(start, p.lastEnd),
		}

	case lexer.TokKwCounter32, lexer.TokKwCounter64, lexer.TokKwGauge32,
		lexer.TokKwUnsigned32, lexer.TokKwTimeTicks, lexer.TokKwIpAddress,
		lexer.TokKwOpaque, lexer.TokKwCounter, lexer.TokKwGauge, lexer.TokKwNetworkAddress:
		tok := p.advance()
		baseSyntax = &cst.TypeRefSyntaxNode{
			Name: tok,
			Span: tok.Span,
		}

	case lexer.TokUppercaseIdent, lexer.TokLowercaseIdent:
		tok := p.advance()
		switch {
		case p.check(lexer.TokLParen):
			constraint, err := p.parseConstraint()
			if err != nil {
				return nil, err
			}
			baseSyntax = &cst.ConstrainedSyntaxNode{
				Base: &cst.TypeRefSyntaxNode{
					Name: tok,
					Span: tok.Span,
				},
				Constraint: constraint,
				Span:       types.NewSpan(start, p.lastEnd),
			}
		case p.check(lexer.TokLBrace):
			// Enum value restriction: TypeRef { value1(1), value2(2) }
			lbrace, values, commas, rbrace, err := p.parseEnumValues()
			if err != nil {
				return nil, err
			}
			baseSyntax = &cst.IntegerEnumSyntaxNode{
				Base:   tok,
				LBrace: lbrace,
				Values: values,
				Commas: commas,
				RBrace: rbrace,
				Span:   types.NewSpan(start, p.lastEnd),
			}
		default:
			baseSyntax = &cst.TypeRefSyntaxNode{
				Name: tok,
				Span: tok.Span,
			}
		}

	case lexer.TokLBracket:
		// Tagged type: [APPLICATION n] IMPLICIT Type
		lbracket := p.advance()
		var tagClass, tagNum cst.SyntaxToken
		if p.check(lexer.TokKwApplication) || p.check(lexer.TokKwUniversal) {
			tagClass = p.advance()
		}
		if p.check(lexer.TokNumber) {
			tagNum = p.advance()
		}
		rbracket, err := p.expect(lexer.TokRBracket)
		if err != nil {
			return nil, err
		}
		var implicit cst.SyntaxToken
		if p.check(lexer.TokKwImplicit) {
			implicit = p.advance()
		}
		inner, err := p.parseTypeSyntax()
		if err != nil {
			return nil, err
		}
		baseSyntax = &cst.TaggedSyntaxNode{
			LBracket: lbracket,
			TagClass: tagClass,
			TagNum:   tagNum,
			RBracket: rbracket,
			Implicit: implicit,
			Inner:    inner,
			Span:     types.NewSpan(start, p.lastEnd),
		}

	default:
		diag := p.makeError("expected type syntax")
		return nil, &diag
	}

	// Check for trailing constraint on the base syntax.
	if p.check(lexer.TokLParen) {
		if _, ok := baseSyntax.(*cst.ConstrainedSyntaxNode); !ok {
			constraint, err := p.parseConstraint()
			if err != nil {
				return nil, err
			}
			return &cst.ConstrainedSyntaxNode{
				Base:       baseSyntax,
				Constraint: constraint,
				Span:       types.NewSpan(start, p.lastEnd),
			}, nil
		}
	}

	return baseSyntax, nil
}

// parseEnumValues parses { name(value), ... } consuming the braces.
func (p *Parser) parseEnumValues() (lbrace cst.SyntaxToken, values []cst.EnumValueNode, commas []cst.SyntaxToken, rbrace cst.SyntaxToken, parseErr *types.SpanDiagnostic) {
	lbrace, parseErr = p.expect(lexer.TokLBrace)
	if parseErr != nil {
		return lbrace, nil, nil, rbrace, parseErr
	}
	values, commas, parseErr = p.parseEnumValueList()
	if parseErr != nil {
		return lbrace, values, commas, rbrace, parseErr
	}
	rbrace, parseErr = p.expect(lexer.TokRBrace)
	return lbrace, values, commas, rbrace, parseErr
}

// parseEnumValueList parses: name(value), name(value), ... (no braces)
func (p *Parser) parseEnumValueList() ([]cst.EnumValueNode, []cst.SyntaxToken, *types.SpanDiagnostic) {
	var values []cst.EnumValueNode
	var commas []cst.SyntaxToken

	for !p.check(lexer.TokRBrace) && !p.isEOF() {
		start := p.currentSpan().Start

		labelTok, err := p.expectEnumLabel()
		if err != nil {
			return nil, nil, err
		}

		lparenTok, err := p.expect(lexer.TokLParen)
		if err != nil {
			return nil, nil, err
		}

		var valueTok cst.SyntaxToken
		if p.check(lexer.TokNegativeNumber) {
			valueTok = p.advance()
		} else {
			valueTok, err = p.expect(lexer.TokNumber)
			if err != nil {
				return nil, nil, err
			}
		}

		rparenTok, err := p.expect(lexer.TokRParen)
		if err != nil {
			return nil, nil, err
		}

		values = append(values, cst.EnumValueNode{
			Label:  labelTok,
			LParen: lparenTok,
			Value:  valueTok,
			RParen: rparenTok,
			Span:   types.NewSpan(start, p.lastEnd),
		})

		if p.check(lexer.TokComma) {
			commas = append(commas, p.advance())
		} else if next := p.peek(); next.Kind.IsIdentifier() || next.Kind.IsKeyword() {
			// Recovery: missing comma before another named number.
			if !p.check(lexer.TokRBrace) {
				p.EmitDiagnostic(types.DiagMissingComma, next.Span, "missing comma in named number list")
			}
		} else {
			break
		}
	}

	return values, commas, nil
}

// parseConstraint parses: (SIZE (ranges)) or (ranges)
func (p *Parser) parseConstraint() (*cst.ConstraintNode, *types.SpanDiagnostic) {
	start := p.currentSpan().Start
	node := &cst.ConstraintNode{}

	lparen, err := p.expect(lexer.TokLParen)
	if err != nil {
		return nil, err
	}
	node.LParen = lparen

	if p.check(lexer.TokKwSize) {
		node.Size = p.advance()
		innerLParen, err := p.expect(lexer.TokLParen)
		if err != nil {
			return nil, err
		}
		node.InnerLParen = innerLParen

		ranges, pipes, err := p.parseRangeList()
		if err != nil {
			return nil, err
		}
		node.Ranges = ranges
		node.Pipes = pipes

		innerRParen, err := p.expect(lexer.TokRParen)
		if err != nil {
			return nil, err
		}
		node.InnerRParen = innerRParen
	} else {
		ranges, pipes, err := p.parseRangeList()
		if err != nil {
			return nil, err
		}
		node.Ranges = ranges
		node.Pipes = pipes
	}

	rparen, err := p.expect(lexer.TokRParen)
	if err != nil {
		return nil, err
	}
	node.RParen = rparen

	node.Span = types.NewSpan(start, p.lastEnd)
	return node, nil
}

// parseRangeList parses: range | range | ...
func (p *Parser) parseRangeList() ([]cst.RangeNode, []cst.SyntaxToken, *types.SpanDiagnostic) {
	var ranges []cst.RangeNode
	var pipes []cst.SyntaxToken

	for {
		r, err := p.parseRange()
		if err != nil {
			return nil, nil, err
		}
		ranges = append(ranges, *r)

		if p.check(lexer.TokPipe) {
			pipes = append(pipes, p.advance())
		} else {
			break
		}
	}

	return ranges, pipes, nil
}

// parseRange parses: value or value..value
func (p *Parser) parseRange() (*cst.RangeNode, *types.SpanDiagnostic) {
	start := p.currentSpan().Start
	node := &cst.RangeNode{}

	minTok, err := p.parseRangeValue()
	if err != nil {
		return nil, err
	}
	node.Min = minTok

	if p.check(lexer.TokDotDot) {
		node.DotDot = p.advance()
		maxTok, err := p.parseRangeValue()
		if err != nil {
			return nil, err
		}
		node.Max = maxTok
	}

	node.Span = types.NewSpan(start, p.lastEnd)
	return node, nil
}

// parseRangeValue parses a single range endpoint.
func (p *Parser) parseRangeValue() (cst.SyntaxToken, *types.SpanDiagnostic) {
	switch {
	case p.check(lexer.TokNumber):
		return p.advance(), nil
	case p.check(lexer.TokNegativeNumber):
		return p.advance(), nil
	case p.check(lexer.TokHexString):
		return p.advance(), nil
	case p.check(lexer.TokUppercaseIdent) || p.check(lexer.TokForbiddenKeyword):
		return p.advance(), nil
	default:
		diag := p.makeError("expected range value")
		return cst.SyntaxToken{}, &diag
	}
}

// parseSequenceFields parses comma-separated name/type pairs for SEQUENCE or CHOICE.
func (p *Parser) parseSequenceFields() ([]cst.SequenceFieldNode, []cst.SyntaxToken, *types.SpanDiagnostic) {
	var fields []cst.SequenceFieldNode
	var commas []cst.SyntaxToken

	for !p.check(lexer.TokRBrace) && !p.isEOF() {
		start := p.currentSpan().Start

		nameTok, err := p.expectIdentifier()
		if err != nil {
			return nil, nil, err
		}

		syntax, err := p.parseTypeSyntax()
		if err != nil {
			return nil, nil, err
		}

		fields = append(fields, cst.SequenceFieldNode{
			Name:   nameTok,
			Syntax: syntax,
			Span:   types.NewSpan(start, p.lastEnd),
		})

		if p.check(lexer.TokComma) {
			commas = append(commas, p.advance())
		}
	}

	return fields, commas, nil
}
