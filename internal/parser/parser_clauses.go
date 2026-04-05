package parser

import (
	"github.com/golangsnmp/gomib/internal/lexer"
	"github.com/golangsnmp/gomib/internal/module"
	"github.com/golangsnmp/gomib/internal/types"
)

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
