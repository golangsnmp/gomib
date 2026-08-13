package lower

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/golangsnmp/gomib/internal/cst"
	"github.com/golangsnmp/gomib/internal/lexer"
	"github.com/golangsnmp/gomib/internal/module"
	"github.com/golangsnmp/gomib/internal/types"
)

// lowerTypeSyntax converts a CST TypeSyntaxNode to module.TypeSyntax.
func (l *lowerer) lowerTypeSyntax(n cst.TypeSyntaxNode) module.TypeSyntax {
	if n == nil {
		return nil
	}

	switch node := n.(type) {
	case *cst.TypeRefSyntaxNode:
		return l.lowerTypeRef(node)
	case *cst.IntegerEnumSyntaxNode:
		return l.lowerIntegerEnum(node)
	case *cst.BitsSyntaxNode:
		return l.lowerBits(node)
	case *cst.ConstrainedSyntaxNode:
		return l.lowerConstrained(node)
	case *cst.SequenceOfSyntaxNode:
		return l.lowerSequenceOf(node)
	case *cst.SequenceSyntaxNode:
		return l.lowerSequence(node)
	case *cst.OctetStringSyntaxNode:
		return &module.TypeSyntaxOctetString{Span: node.Span}
	case *cst.ObjectIdentifierSyntaxNode:
		return &module.TypeSyntaxObjectIdentifier{Span: node.Span}
	case *cst.TaggedSyntaxNode:
		if !module.IsBaseModule(l.moduleName) {
			l.EmitDiagnostic(types.DiagTaggedTypeNotAllowed, node.Span,
				fmt.Sprintf("tagged type not allowed outside base module %q", l.moduleName))
		}
		return l.lowerTypeSyntax(node.Inner)
	case *cst.ChoiceSyntaxNode:
		if !module.IsBaseModule(l.moduleName) {
			l.EmitDiagnostic(types.DiagChoiceNotAllowed, node.Span,
				fmt.Sprintf("CHOICE type not allowed outside base module %q", l.moduleName))
		}
		if len(node.Fields) > 0 {
			return l.lowerTypeSyntax(node.Fields[0].Syntax)
		}
		return &module.TypeSyntaxOctetString{}
	default:
		return nil
	}
}

func (l *lowerer) lowerTypeRef(n *cst.TypeRefSyntaxNode) module.TypeSyntax {
	var name string
	switch n.Name.Kind {
	case lexer.TokKwInteger:
		name = "INTEGER"
	case lexer.TokKwBits:
		name = "BITS"
	default:
		name = l.text(n.Name)
	}
	if n.Name.Kind == lexer.TokLowercaseIdent {
		l.EmitDiagnostic(types.DiagBadIdentifierCase, n.Name.Span,
			fmt.Sprintf("type reference %q should start with an uppercase letter", name))
	}
	return &module.TypeSyntaxTypeRef{
		Name: name,
		Span: n.Name.Span,
	}
}

func (l *lowerer) lowerIntegerEnum(n *cst.IntegerEnumSyntaxNode) module.TypeSyntax {
	var base string
	if n.Base.Kind == lexer.TokKwInteger {
		base = ""
	} else {
		base = l.text(n.Base)
	}

	namedNumbers := l.lowerNamedNumbers(n.Values)

	seenNames := make(map[string]bool)
	seenValues := make(map[int64]bool)
	for i := range n.Values {
		v := &n.Values[i]
		name := l.text(v.Label)
		value := l.convertI64(v.Value)
		if seenNames[name] {
			l.EmitDiagnostic(types.DiagEnumNameRedefinition, v.Span,
				fmt.Sprintf("duplicate enum name %q", name))
		}
		seenNames[name] = true
		if seenValues[value] {
			l.EmitDiagnostic(types.DiagEnumValueRedefinition, v.Span,
				fmt.Sprintf("duplicate enum value %d", value))
		}
		seenValues[value] = true
	}

	return &module.TypeSyntaxIntegerEnum{
		Base:         base,
		NamedNumbers: namedNumbers,
		Span:         n.Span,
	}
}

func (l *lowerer) lowerNamedNumbers(values []cst.EnumValueNode) []module.NamedNumber {
	var result []module.NamedNumber
	for i := range values {
		v := &values[i]
		name := l.text(v.Label)
		value := l.convertI64(v.Value)
		if value < -1<<31 || value > 1<<31-1 {
			l.EmitDiagnostic(types.DiagEnumValueOutOfRange, v.Value.Span,
				fmt.Sprintf("enum value %d is outside the Integer32 range", value))
		}
		result = append(result, module.NewNamedNumber(name, value, v.Span))
	}
	return result
}

func (l *lowerer) lowerBits(n *cst.BitsSyntaxNode) module.TypeSyntax {
	seenNames := make(map[string]bool)
	seenPositions := make(map[int64]bool)
	var namedBits []module.NamedBit
	for i := range n.Values {
		v := &n.Values[i]
		name := l.text(v.Label)
		pos := l.convertI64(v.Value)

		if seenNames[name] {
			l.EmitDiagnostic(types.DiagBitsNameRedefinition, v.Span,
				fmt.Sprintf("duplicate BITS name %q", name))
		}
		seenNames[name] = true
		if seenPositions[pos] {
			l.EmitDiagnostic(types.DiagBitsValueRedefinition, v.Span,
				fmt.Sprintf("duplicate BITS position %d", pos))
		}
		seenPositions[pos] = true

		switch {
		case pos < 0:
			pos = 0
		case pos >= 65535*8:
			pos = 0
		}
		namedBits = append(namedBits, module.NewNamedBit(name, uint32(pos), v.Span))
	}

	return &module.TypeSyntaxBits{
		NamedBits: namedBits,
		Span:      n.Span,
	}
}

func (l *lowerer) lowerConstrained(n *cst.ConstrainedSyntaxNode) module.TypeSyntax {
	base := l.lowerTypeSyntax(n.Base)
	constraint := l.lowerConstraint(n.Constraint)

	return &module.TypeSyntaxConstrained{
		Base:       base,
		Constraint: constraint,
		Span:       n.Span,
	}
}

func (l *lowerer) lowerConstraint(n *cst.ConstraintNode) module.Constraint {
	if n == nil {
		return nil
	}

	ranges := l.lowerRanges(n.Ranges)

	if !n.Size.IsZero() {
		return &module.ConstraintSize{
			Ranges: ranges,
			Span:   n.Span,
		}
	}

	return &module.ConstraintRange{
		Ranges: ranges,
		Span:   n.Span,
	}
}

func (l *lowerer) lowerRanges(nodes []cst.RangeNode) []module.Range {
	var ranges []module.Range
	for i := range nodes {
		rn := &nodes[i]
		lo := l.lowerRangeValue(rn.Min)
		var hi module.RangeValue
		if !rn.DotDot.IsZero() {
			hi = l.lowerRangeValue(rn.Max)
		}
		ranges = append(ranges, module.Range{
			Min:  lo,
			Max:  hi,
			Span: rn.Span,
		})
	}
	return ranges
}

func (l *lowerer) lowerRangeValue(tok cst.SyntaxToken) module.RangeValue {
	text := l.text(tok)

	switch tok.Kind { //nolint:exhaustive // only relevant token kinds handled
	case lexer.TokNumber:
		// Try unsigned first to handle large values.
		if value, err := strconv.ParseUint(text, 10, 64); err == nil {
			return &module.RangeValueUnsigned{Value: value}
		}
		if value, err := strconv.ParseInt(text, 10, 64); err == nil {
			return &module.RangeValueSigned{Value: value}
		}
		l.EmitDiagnostic(types.DiagUnknownRangeValue, tok.Span,
			fmt.Sprintf("range value %q is outside the supported integer range", text))
		return &module.RangeValueRaw{Value: text}

	case lexer.TokNegativeNumber:
		if value, err := strconv.ParseInt(text, 10, 64); err == nil {
			return &module.RangeValueSigned{Value: value}
		}
		l.EmitDiagnostic(types.DiagUnknownRangeValue, tok.Span,
			fmt.Sprintf("range value %q is outside the supported integer range", text))
		return &module.RangeValueRaw{Value: text}

	case lexer.TokHexString:
		hexPart := stripQuotedLiteral(text)
		normalized := strings.Map(func(r rune) rune {
			switch r {
			case ' ', '\t', '\n', '\r':
				return -1
			default:
				return r
			}
		}, hexPart)
		if value, err := strconv.ParseUint(normalized, 16, 64); err == nil {
			return &module.RangeValueUnsigned{Value: value}
		}
		l.EmitDiagnostic(types.DiagInvalidHexRange, tok.Span, "invalid hex range value")
		return &module.RangeValueRaw{Value: text}

	case lexer.TokUppercaseIdent, lexer.TokForbiddenKeyword:
		switch text {
		case "MIN":
			l.EmitDiagnostic(types.DiagMinMaxRange, tok.Span,
				"MIN is not valid in SMI constraints; preserving it as an open endpoint")
			return &module.RangeValueMin{}
		case "MAX":
			l.EmitDiagnostic(types.DiagMinMaxRange, tok.Span,
				"MAX is not valid in SMI constraints; preserving it as an open endpoint")
			return &module.RangeValueMax{}
		default:
			l.EmitDiagnostic(types.DiagUnknownRangeValue, tok.Span,
				fmt.Sprintf("unknown symbolic range value %q", text))
			return &module.RangeValueRaw{Value: text}
		}
	}

	l.EmitDiagnostic(types.DiagUnknownRangeValue, tok.Span,
		fmt.Sprintf("unknown range value %q", text))
	return &module.RangeValueRaw{Value: text}
}

func (l *lowerer) lowerSequenceOf(n *cst.SequenceOfSyntaxNode) module.TypeSyntax {
	entryType := l.text(n.Entry)
	return &module.TypeSyntaxSequenceOf{
		EntryType: entryType,
		Span:      n.Span,
	}
}

func (l *lowerer) lowerSequence(n *cst.SequenceSyntaxNode) module.TypeSyntax {
	var fields []module.SequenceField
	for _, f := range n.Fields {
		name := l.text(f.Name)
		syntax := l.lowerTypeSyntax(f.Syntax)
		span := types.NewSpan(f.Span.Start, syntax.SyntaxSpan().End)
		fields = append(fields, module.NewSequenceField(name, syntax, span))
	}

	return &module.TypeSyntaxSequence{
		Fields: fields,
		Span:   n.Span,
	}
}
