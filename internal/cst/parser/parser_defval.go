package parser

import (
	"github.com/golangsnmp/gomib/internal/cst"
	"github.com/golangsnmp/gomib/internal/lexer"
	"github.com/golangsnmp/gomib/internal/types"
)

// parseDefValClause parses: DEFVAL { content }
// The content is stored as opaque tokens.
func (p *Parser) parseDefValClause() (*cst.DefValClauseNode, *types.SpanDiagnostic) {
	start := p.currentSpan().Start
	node := &cst.DefValClauseNode{}

	kwTok, err := p.expect(lexer.TokKwDefval)
	if err != nil {
		return nil, err
	}
	node.Keyword = kwTok

	lbrace, err := p.expect(lexer.TokLBrace)
	if err != nil {
		return nil, err
	}
	node.LBrace = lbrace

	// Collect all tokens between braces as opaque content.
	// Track brace depth for nested braces.
	depth := 0
	for !p.isEOF() {
		if p.check(lexer.TokRBrace) && depth == 0 {
			break
		}
		if p.check(lexer.TokLBrace) {
			depth++
		} else if p.check(lexer.TokRBrace) {
			depth--
		}
		node.Content = append(node.Content, p.advance())
	}

	rbrace, err := p.expect(lexer.TokRBrace)
	if err != nil {
		return nil, err
	}
	node.RBrace = rbrace

	node.Span = types.NewSpan(start, p.lastEnd)
	return node, nil
}
