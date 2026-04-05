package parser

import (
	"fmt"
	"strconv"

	"github.com/golangsnmp/gomib/internal/lexer"
	"github.com/golangsnmp/gomib/internal/module"
	"github.com/golangsnmp/gomib/internal/types"
)

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
