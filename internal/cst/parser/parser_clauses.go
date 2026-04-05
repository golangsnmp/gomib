package parser

import (
	"github.com/golangsnmp/gomib/internal/cst"
	"github.com/golangsnmp/gomib/internal/lexer"
	"github.com/golangsnmp/gomib/internal/types"
)

// parseSyntaxClause parses: SYNTAX <type-syntax>
func (p *Parser) parseSyntaxClause() (*cst.SyntaxClauseNode, *types.SpanDiagnostic) {
	start := p.currentSpan().Start
	kwTok, err := p.expect(lexer.TokKwSyntax)
	if err != nil {
		return nil, err
	}

	syntax, err := p.parseTypeSyntax()
	if err != nil {
		return nil, err
	}

	return &cst.SyntaxClauseNode{
		Keyword: kwTok,
		Syntax:  syntax,
		Span:    types.NewSpan(start, p.lastEnd),
	}, nil
}

// parseWriteSyntaxClause parses: WRITE-SYNTAX <type-syntax>
func (p *Parser) parseWriteSyntaxClause() (*cst.WriteSyntaxClauseNode, *types.SpanDiagnostic) {
	start := p.currentSpan().Start
	kwTok, err := p.expect(lexer.TokKwWriteSyntax)
	if err != nil {
		return nil, err
	}

	syntax, err := p.parseTypeSyntax()
	if err != nil {
		return nil, err
	}

	return &cst.WriteSyntaxClauseNode{
		Keyword: kwTok,
		Syntax:  syntax,
		Span:    types.NewSpan(start, p.lastEnd),
	}, nil
}

// parseAccessClause parses: MAX-ACCESS | ACCESS | MIN-ACCESS <value>
func (p *Parser) parseAccessClause() (*cst.AccessClauseNode, *types.SpanDiagnostic) {
	start := p.currentSpan().Start

	var kwTok cst.SyntaxToken
	switch {
	case p.check(lexer.TokKwMaxAccess):
		kwTok = p.advance()
	case p.check(lexer.TokKwAccess):
		kwTok = p.advance()
	case p.check(lexer.TokKwMinAccess):
		kwTok = p.advance()
	default:
		diag := p.makeError("expected MAX-ACCESS, MIN-ACCESS, or ACCESS")
		return nil, &diag
	}

	// Access value
	kind := p.peek().Kind
	switch kind {
	case lexer.TokKwReadOnly, lexer.TokKwReadWrite, lexer.TokKwReadCreate,
		lexer.TokKwNotAccessible, lexer.TokKwAccessibleForNotify,
		lexer.TokKwWriteOnly, lexer.TokKwNotImplemented:
		valueTok := p.advance()
		return &cst.AccessClauseNode{
			Keyword: kwTok,
			Value:   valueTok,
			Span:    types.NewSpan(start, p.lastEnd),
		}, nil
	default:
		diag := p.makeError("expected access value")
		return nil, &diag
	}
}

// parseStatusClause parses: STATUS <value>
func (p *Parser) parseStatusClause() (*cst.StatusClauseNode, *types.SpanDiagnostic) {
	start := p.currentSpan().Start
	kwTok, err := p.expect(lexer.TokKwStatus)
	if err != nil {
		return nil, err
	}

	kind := p.peek().Kind
	switch kind {
	case lexer.TokKwCurrent, lexer.TokKwDeprecated, lexer.TokKwObsolete,
		lexer.TokKwMandatory, lexer.TokKwOptional:
		valueTok := p.advance()
		return &cst.StatusClauseNode{
			Keyword: kwTok,
			Value:   valueTok,
			Span:    types.NewSpan(start, p.lastEnd),
		}, nil
	default:
		diag := p.makeError("expected status value")
		return nil, &diag
	}
}

// parseDescriptionClause parses: DESCRIPTION "..."
func (p *Parser) parseDescriptionClause() (*cst.DescriptionClauseNode, *types.SpanDiagnostic) {
	start := p.currentSpan().Start
	kwTok, err := p.expect(lexer.TokKwDescription)
	if err != nil {
		return nil, err
	}

	valueTok, err := p.expect(lexer.TokQuotedString)
	if err != nil {
		return nil, err
	}

	return &cst.DescriptionClauseNode{
		Keyword: kwTok,
		Value:   valueTok,
		Span:    types.NewSpan(start, p.lastEnd),
	}, nil
}

// parseReferenceClause parses: REFERENCE "..."
func (p *Parser) parseReferenceClause() (*cst.ReferenceClauseNode, *types.SpanDiagnostic) {
	start := p.currentSpan().Start
	kwTok, err := p.expect(lexer.TokKwReference)
	if err != nil {
		return nil, err
	}

	valueTok, err := p.expect(lexer.TokQuotedString)
	if err != nil {
		return nil, err
	}

	return &cst.ReferenceClauseNode{
		Keyword: kwTok,
		Value:   valueTok,
		Span:    types.NewSpan(start, p.lastEnd),
	}, nil
}

// parseUnitsClause parses: UNITS "..."
func (p *Parser) parseUnitsClause() (*cst.UnitsClauseNode, *types.SpanDiagnostic) {
	start := p.currentSpan().Start
	kwTok, err := p.expect(lexer.TokKwUnits)
	if err != nil {
		return nil, err
	}

	valueTok, err := p.expect(lexer.TokQuotedString)
	if err != nil {
		return nil, err
	}

	return &cst.UnitsClauseNode{
		Keyword: kwTok,
		Value:   valueTok,
		Span:    types.NewSpan(start, p.lastEnd),
	}, nil
}

// parseDisplayHintClause parses: DISPLAY-HINT "..."
func (p *Parser) parseDisplayHintClause() (*cst.DisplayHintClauseNode, *types.SpanDiagnostic) {
	start := p.currentSpan().Start
	kwTok, err := p.expect(lexer.TokKwDisplayHint)
	if err != nil {
		return nil, err
	}

	valueTok, err := p.expect(lexer.TokQuotedString)
	if err != nil {
		return nil, err
	}

	return &cst.DisplayHintClauseNode{
		Keyword: kwTok,
		Value:   valueTok,
		Span:    types.NewSpan(start, p.lastEnd),
	}, nil
}

// parseIndexClause parses: INDEX { [IMPLIED] name, ... }
func (p *Parser) parseIndexClause() (*cst.IndexClauseNode, *types.SpanDiagnostic) {
	start := p.currentSpan().Start
	node := &cst.IndexClauseNode{}

	kwTok, err := p.expect(lexer.TokKwIndex)
	if err != nil {
		return nil, err
	}
	node.Keyword = kwTok

	lbrace, err := p.expect(lexer.TokLBrace)
	if err != nil {
		return nil, err
	}
	node.LBrace = lbrace

	for !p.check(lexer.TokRBrace) && !p.isEOF() {
		item, err := p.parseIndexItem()
		if err != nil {
			return nil, err
		}
		node.Items = append(node.Items, *item)

		if p.check(lexer.TokComma) {
			node.Commas = append(node.Commas, p.advance())
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

// parseIndexItem parses: [IMPLIED] objectName
func (p *Parser) parseIndexItem() (*cst.IndexItemNode, *types.SpanDiagnostic) {
	start := p.currentSpan().Start
	node := &cst.IndexItemNode{}

	if p.check(lexer.TokKwImplied) {
		node.Implied = p.advance()
	}

	objTok, err := p.expectIndexObject()
	if err != nil {
		return nil, err
	}

	// Handle OCTET STRING as a two-token index object. IndexItemNode
	// has a single Object field, so STRING is consumed but not stored.
	// This is a rare vendor MIB pattern; the node type would need a
	// second field to preserve it for round-trip.
	if objTok.Kind == lexer.TokKwOctet && p.check(lexer.TokKwString) {
		node.StringKeyword = p.advance()
	}

	node.Object = objTok
	node.Span = types.NewSpan(start, p.lastEnd)
	return node, nil
}

// parseAugmentsClause parses: AUGMENTS { name }
func (p *Parser) parseAugmentsClause() (*cst.AugmentsClauseNode, *types.SpanDiagnostic) {
	start := p.currentSpan().Start
	node := &cst.AugmentsClauseNode{}

	kwTok, err := p.expect(lexer.TokKwAugments)
	if err != nil {
		return nil, err
	}
	node.Keyword = kwTok

	lbrace, err := p.expect(lexer.TokLBrace)
	if err != nil {
		return nil, err
	}
	node.LBrace = lbrace

	target, err := p.expectIdentifier()
	if err != nil {
		return nil, err
	}
	node.Target = target

	rbrace, err := p.expect(lexer.TokRBrace)
	if err != nil {
		return nil, err
	}
	node.RBrace = rbrace

	node.Span = types.NewSpan(start, p.lastEnd)
	return node, nil
}

// parseObjectsClause parses: OBJECTS { name, name, ... }
func (p *Parser) parseObjectsClause() (*cst.ObjectsClauseNode, *types.SpanDiagnostic) {
	start := p.currentSpan().Start
	node := &cst.ObjectsClauseNode{}

	kwTok, err := p.expect(lexer.TokKwObjects)
	if err != nil {
		return nil, err
	}
	node.Keyword = kwTok

	lbrace, names, commas, rbrace, err := p.parseBracedIdentifierList()
	if err != nil {
		return nil, err
	}
	node.LBrace = lbrace
	node.Names = names
	node.Commas = commas
	node.RBrace = rbrace

	node.Span = types.NewSpan(start, p.lastEnd)
	return node, nil
}

// parseNotificationsClause parses: NOTIFICATIONS { name, name, ... }
func (p *Parser) parseNotificationsClause() (*cst.NotificationsClauseNode, *types.SpanDiagnostic) {
	start := p.currentSpan().Start
	node := &cst.NotificationsClauseNode{}

	kwTok, err := p.expect(lexer.TokKwNotifications)
	if err != nil {
		return nil, err
	}
	node.Keyword = kwTok

	lbrace, names, commas, rbrace, err := p.parseBracedIdentifierList()
	if err != nil {
		return nil, err
	}
	node.LBrace = lbrace
	node.Names = names
	node.Commas = commas
	node.RBrace = rbrace

	node.Span = types.NewSpan(start, p.lastEnd)
	return node, nil
}

// parseRevisionClause parses: REVISION "date" DESCRIPTION "..."
func (p *Parser) parseRevisionClause() (*cst.RevisionNode, *types.SpanDiagnostic) {
	start := p.currentSpan().Start
	node := &cst.RevisionNode{}

	kwTok, err := p.expect(lexer.TokKwRevision)
	if err != nil {
		return nil, err
	}
	node.Keyword = kwTok

	dateTok, err := p.expect(lexer.TokQuotedString)
	if err != nil {
		return nil, err
	}
	node.Date = dateTok

	desc, err := p.parseDescriptionClause()
	if err != nil {
		return nil, err
	}
	node.Description = desc

	node.Span = types.NewSpan(start, p.lastEnd)
	return node, nil
}

// parseLastUpdatedClause parses: LAST-UPDATED "date"
func (p *Parser) parseLastUpdatedClause() (*cst.LastUpdatedClauseNode, *types.SpanDiagnostic) {
	start := p.currentSpan().Start
	kwTok, err := p.expect(lexer.TokKwLastUpdated)
	if err != nil {
		return nil, err
	}

	valueTok, err := p.expect(lexer.TokQuotedString)
	if err != nil {
		return nil, err
	}

	return &cst.LastUpdatedClauseNode{
		Keyword: kwTok,
		Value:   valueTok,
		Span:    types.NewSpan(start, p.lastEnd),
	}, nil
}

// parseOrganizationClause parses: ORGANIZATION "..."
func (p *Parser) parseOrganizationClause() (*cst.OrganizationClauseNode, *types.SpanDiagnostic) {
	start := p.currentSpan().Start
	kwTok, err := p.expect(lexer.TokKwOrganization)
	if err != nil {
		return nil, err
	}

	valueTok, err := p.expect(lexer.TokQuotedString)
	if err != nil {
		return nil, err
	}

	return &cst.OrganizationClauseNode{
		Keyword: kwTok,
		Value:   valueTok,
		Span:    types.NewSpan(start, p.lastEnd),
	}, nil
}

// parseContactInfoClause parses: CONTACT-INFO "..."
func (p *Parser) parseContactInfoClause() (*cst.ContactInfoClauseNode, *types.SpanDiagnostic) {
	start := p.currentSpan().Start
	kwTok, err := p.expect(lexer.TokKwContactInfo)
	if err != nil {
		return nil, err
	}

	valueTok, err := p.expect(lexer.TokQuotedString)
	if err != nil {
		return nil, err
	}

	return &cst.ContactInfoClauseNode{
		Keyword: kwTok,
		Value:   valueTok,
		Span:    types.NewSpan(start, p.lastEnd),
	}, nil
}

// parseEnterpriseClause parses: ENTERPRISE name
func (p *Parser) parseEnterpriseClause() (*cst.EnterpriseClauseNode, *types.SpanDiagnostic) {
	start := p.currentSpan().Start
	kwTok, err := p.expect(lexer.TokKwEnterprise)
	if err != nil {
		return nil, err
	}

	nameTok, err := p.expectIdentifier()
	if err != nil {
		return nil, err
	}

	return &cst.EnterpriseClauseNode{
		Keyword: kwTok,
		Name:    nameTok,
		Span:    types.NewSpan(start, p.lastEnd),
	}, nil
}

// parseVariablesClause parses: VARIABLES { name, name, ... }
func (p *Parser) parseVariablesClause() (*cst.VariablesClauseNode, *types.SpanDiagnostic) {
	start := p.currentSpan().Start
	node := &cst.VariablesClauseNode{}

	kwTok, err := p.expect(lexer.TokKwVariables)
	if err != nil {
		return nil, err
	}
	node.Keyword = kwTok

	lbrace, names, commas, rbrace, err := p.parseBracedIdentifierList()
	if err != nil {
		return nil, err
	}
	node.LBrace = lbrace
	node.Names = names
	node.Commas = commas
	node.RBrace = rbrace

	node.Span = types.NewSpan(start, p.lastEnd)
	return node, nil
}

// parseProductReleaseClause parses: PRODUCT-RELEASE "..."
func (p *Parser) parseProductReleaseClause() (*cst.ProductReleaseClauseNode, *types.SpanDiagnostic) {
	start := p.currentSpan().Start
	kwTok, err := p.expect(lexer.TokKwProductRelease)
	if err != nil {
		return nil, err
	}

	valueTok, err := p.expect(lexer.TokQuotedString)
	if err != nil {
		return nil, err
	}

	return &cst.ProductReleaseClauseNode{
		Keyword: kwTok,
		Value:   valueTok,
		Span:    types.NewSpan(start, p.lastEnd),
	}, nil
}
