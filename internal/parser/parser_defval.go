package parser

import (
	"strconv"
	"strings"

	"github.com/golangsnmp/gomib/internal/lexer"
	"github.com/golangsnmp/gomib/internal/module"
	"github.com/golangsnmp/gomib/internal/types"
)

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
		return p.finishDefValOid(nil)
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
	return p.parseDefValOidWithFirstIdent(identName, identToken)
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

func (p *Parser) parseDefValOidWithFirstIdent(identName string, identToken lexer.Token) (module.DefVal, *types.SpanDiagnostic) {
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
	return p.finishDefValOid(components)
}

func (p *Parser) finishDefValOid(components []module.OidComponent) (module.DefVal, *types.SpanDiagnostic) {
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
