package lower

import (
	"strconv"

	"github.com/golangsnmp/gomib/internal/cst"
	"github.com/golangsnmp/gomib/internal/lexer"
	"github.com/golangsnmp/gomib/internal/module"
	"github.com/golangsnmp/gomib/internal/types"
)

// lowerDefVal converts opaque DefVal content tokens into the appropriate
// module.DefVal variant. This mirrors the existing parser's parseDefVal logic.
func (l *lowerer) lowerDefVal(content []cst.SyntaxToken) module.DefVal {
	if len(content) == 0 {
		return nil
	}

	// Filter to meaningful tokens (skip whitespace-only trivia artifacts).
	// Content tokens from the CST parser are already meaningful.
	first := content[0]

	switch first.Kind {
	case lexer.TokNegativeNumber:
		value := l.convertI64(first)
		return &module.DefValInteger{Value: value}

	case lexer.TokNumber:
		return l.lowerDefValNumber(first)

	case lexer.TokQuotedString:
		str := l.stripQuotes(first)
		return &module.DefValString{Value: str}

	case lexer.TokHexString:
		text := l.text(first)
		inner := stripQuotedLiteral(text)
		return &module.DefValHexString{Value: inner}

	case lexer.TokBinString:
		text := l.text(first)
		inner := stripQuotedLiteral(text)
		return &module.DefValBinaryString{Value: inner}

	case lexer.TokLowercaseIdent, lexer.TokUppercaseIdent:
		name := l.text(first)
		return &module.DefValEnum{Name: name}

	case lexer.TokLBrace:
		return l.lowerDefValBracedContent(content[1:])

	default:
		// Keywords can be valid enum labels
		if first.Kind.IsKeyword() {
			name := l.text(first)
			return &module.DefValEnum{Name: name}
		}
		return &module.DefValUnparsed{}
	}
}

func (l *lowerer) lowerDefValNumber(tok cst.SyntaxToken) module.DefVal {
	text := l.text(tok)
	if value, err := strconv.ParseInt(text, 10, 64); err == nil {
		return &module.DefValInteger{Value: value}
	}
	if value, err := strconv.ParseUint(text, 10, 64); err == nil {
		return &module.DefValUnsigned{Value: value}
	}
	value := l.convertI64(tok)
	return &module.DefValInteger{Value: value}
}

// lowerDefValBracedContent handles the content inside { } in a DEFVAL.
// The opening brace has been consumed; content starts after it.
// The content includes the inner closing brace from the CST content tokens.
// We strip the trailing RBrace since it's the inner closing brace.
func (l *lowerer) lowerDefValBracedContent(content []cst.SyntaxToken) module.DefVal {
	// Strip the trailing RBrace (inner closing brace).
	if len(content) > 0 && content[len(content)-1].Kind == lexer.TokRBrace {
		content = content[:len(content)-1]
	}

	// Empty braces: DEFVAL { { } }
	if len(content) == 0 {
		return &module.DefValBits{Labels: nil}
	}

	first := content[0]

	switch first.Kind {
	case lexer.TokLowercaseIdent, lexer.TokUppercaseIdent:
		return l.lowerDefValBracedIdent(content)

	case lexer.TokNumber:
		// Starts with number -> OID value
		return l.lowerDefValOidFromTokens(content)

	default:
		// Keywords can be valid BITS labels
		if first.Kind.IsKeyword() {
			return l.lowerDefValBracedIdent(content)
		}
		return &module.DefValUnparsed{}
	}
}

func (l *lowerer) lowerDefValBracedIdent(content []cst.SyntaxToken) module.DefVal {
	if len(content) == 0 {
		return &module.DefValBits{Labels: nil}
	}

	first := content[0]
	identName := l.text(first)

	// Look at second token to decide BITS vs OID.
	if len(content) == 1 {
		// Single identifier in braces: this is BITS with one label.
		return &module.DefValBits{Labels: []string{identName}}
	}

	second := content[1]
	if second.Kind == lexer.TokComma || second.Kind == lexer.TokRBrace {
		// This is BITS: { flag1, flag2, ... }
		return l.lowerDefValBitsLabels(content)
	}

	// This is OID: { sysName 0 } or { iso 3 6 1 }
	return l.lowerDefValOidWithFirstIdent(content)
}

func (l *lowerer) lowerDefValBitsLabels(content []cst.SyntaxToken) module.DefVal {
	var labels []string
	for _, tok := range content {
		if tok.Kind == lexer.TokComma {
			continue
		}
		if tok.Kind.IsIdentifier() || tok.Kind.IsKeyword() {
			labels = append(labels, l.text(tok))
		}
	}
	return &module.DefValBits{Labels: labels}
}

func (l *lowerer) lowerDefValOidWithFirstIdent(content []cst.SyntaxToken) module.DefVal {
	if len(content) == 0 {
		return &module.DefValOidValue{Components: nil}
	}

	var components []module.OidComponent
	i := 0

	// First token is the identifier we already know about.
	firstTok := content[i]
	identName := l.text(firstTok)
	i++

	// Check if followed by ( for named number.
	if i < len(content) && content[i].Kind == lexer.TokLParen {
		i++ // skip (
		if i < len(content) && content[i].Kind == lexer.TokNumber {
			numTok := content[i]
			number := l.convertU32(numTok)
			i++
			var endOffset types.ByteOffset
			if i < len(content) && content[i].Kind == lexer.TokRParen {
				endOffset = content[i].Span.End
				i++
			} else {
				endOffset = numTok.Span.End
			}
			components = append(components, &module.OidComponentNamedNumber{
				NameValue:   identName,
				NumberValue: number,
				Span:        types.NewSpan(firstTok.Span.Start, endOffset),
			})
		} else {
			// Malformed, just add as name
			components = append(components, &module.OidComponentName{NameValue: identName, Span: firstTok.Span})
		}
	} else {
		components = append(components, &module.OidComponentName{NameValue: identName, Span: firstTok.Span})
	}

	// Parse remaining OID components.
	for i < len(content) {
		tok := content[i]
		switch tok.Kind {
		case lexer.TokNumber:
			number := l.convertU32(tok)
			components = append(components, &module.OidComponentNumber{Value: number, Span: tok.Span})
			i++

		case lexer.TokLowercaseIdent, lexer.TokUppercaseIdent:
			name := l.text(tok)
			compStart := tok.Span.Start
			i++

			comp := l.parseDefValOidIdentComponent(content, &i, name, compStart, tok.Span)
			components = append(components, comp)

		default:
			// Skip unexpected tokens
			i++
		}
	}

	return &module.DefValOidValue{Components: components}
}

func (l *lowerer) lowerDefValOidFromTokens(content []cst.SyntaxToken) module.DefVal {
	return l.lowerDefValOidWithFirstIdent(content)
}

// parseDefValOidIdentComponent parses a single OID component starting with an
// identifier from DEFVAL content tokens. Updates *idx to reflect consumed tokens.
func (l *lowerer) parseDefValOidIdentComponent(content []cst.SyntaxToken, idx *int, name string, compStart types.ByteOffset, nameSpan types.Span) module.OidComponent {
	i := *idx
	defer func() { *idx = i }()

	if i >= len(content) {
		return &module.OidComponentName{NameValue: name, Span: nameSpan}
	}

	switch content[i].Kind {
	case lexer.TokDot:
		i++ // skip dot
		if i < len(content) && content[i].Kind == lexer.TokLowercaseIdent {
			qname := l.text(content[i])
			qnameEnd := content[i].Span.End
			i++
			if i < len(content) && content[i].Kind == lexer.TokLParen {
				i++
				if i < len(content) && content[i].Kind == lexer.TokNumber {
					numTok := content[i]
					number := l.convertU32(numTok)
					i++
					endOffset := numTok.Span.End
					if i < len(content) && content[i].Kind == lexer.TokRParen {
						endOffset = content[i].Span.End
						i++
					}
					return &module.OidComponentQualifiedNamedNumber{
						ModuleValue: name,
						NameValue:   qname,
						NumberValue: number,
						Span:        types.NewSpan(compStart, endOffset),
					}
				}
			}
			return &module.OidComponentQualifiedName{
				ModuleValue: name,
				NameValue:   qname,
				Span:        types.NewSpan(compStart, qnameEnd),
			}
		}
		return &module.OidComponentName{NameValue: name, Span: nameSpan}

	case lexer.TokLParen:
		i++
		if i < len(content) && content[i].Kind == lexer.TokNumber {
			numTok := content[i]
			number := l.convertU32(numTok)
			i++
			endOffset := numTok.Span.End
			if i < len(content) && content[i].Kind == lexer.TokRParen {
				endOffset = content[i].Span.End
				i++
			}
			return &module.OidComponentNamedNumber{
				NameValue:   name,
				NumberValue: number,
				Span:        types.NewSpan(compStart, endOffset),
			}
		}
		return &module.OidComponentName{NameValue: name, Span: nameSpan}

	default:
		return &module.OidComponentName{NameValue: name, Span: nameSpan}
	}
}
