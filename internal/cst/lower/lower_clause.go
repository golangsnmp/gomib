package lower

import (
	"github.com/golangsnmp/gomib/internal/cst"
	"github.com/golangsnmp/gomib/internal/lexer"
	"github.com/golangsnmp/gomib/internal/module"
	"github.com/golangsnmp/gomib/internal/types"
)

// lowerAccess converts a CST AccessClauseNode to types.Access and types.AccessKeyword.
func (l *lowerer) lowerAccess(n *cst.AccessClauseNode) (types.Access, types.AccessKeyword, types.Span) {
	if n == nil {
		return 0, 0, types.Span{}
	}

	var keyword types.AccessKeyword
	switch n.Keyword.Kind { //nolint:exhaustive // only valid keyword/value tokens appear here
	case lexer.TokKwMaxAccess:
		keyword = types.AccessKeywordMaxAccess
	case lexer.TokKwAccess:
		keyword = types.AccessKeywordAccess
	case lexer.TokKwMinAccess:
		keyword = types.AccessKeywordMinAccess
	}

	var value types.Access
	switch n.Value.Kind { //nolint:exhaustive // only valid keyword/value tokens appear here
	case lexer.TokKwReadOnly:
		value = types.AccessReadOnly
	case lexer.TokKwReadWrite:
		value = types.AccessReadWrite
	case lexer.TokKwReadCreate:
		value = types.AccessReadCreate
	case lexer.TokKwNotAccessible:
		value = types.AccessNotAccessible
	case lexer.TokKwAccessibleForNotify:
		value = types.AccessAccessibleForNotify
	case lexer.TokKwWriteOnly:
		value = types.AccessWriteOnly
	case lexer.TokKwNotImplemented:
		value = types.AccessNotImplemented
	}

	return value, keyword, n.Span
}

// lowerStatus converts a CST StatusClauseNode to types.Status.
func (l *lowerer) lowerStatus(n *cst.StatusClauseNode) (types.Status, types.Span) {
	if n == nil {
		return types.StatusCurrent, types.Span{}
	}

	var value types.Status
	switch n.Value.Kind { //nolint:exhaustive // only valid keyword/value tokens appear here
	case lexer.TokKwCurrent:
		value = types.StatusCurrent
	case lexer.TokKwDeprecated:
		value = types.StatusDeprecated
	case lexer.TokKwObsolete:
		value = types.StatusObsolete
	case lexer.TokKwMandatory:
		value = types.StatusMandatory
	case lexer.TokKwOptional:
		value = types.StatusOptional
	}

	return value, n.Span
}

// lowerIndexOrAugments converts INDEX or AUGMENTS clause nodes.
func (l *lowerer) lowerIndexOrAugments(index *cst.IndexClauseNode, augments *cst.AugmentsClauseNode) (
	indexItems []module.IndexItem, augmentsTarget string, augmentsSpan, indexSpan types.Span,
) {
	if index != nil {
		for i := range index.Items {
			item := &index.Items[i]
			implied := !item.Implied.IsZero()

			objectName := l.text(item.Object)

			// Handle OCTET STRING: the CST parser stores the OCTET token,
			// but we know it consumed STRING too (see parseIndexItem). The
			// existing parser combines them into "OCTET STRING".
			if item.Object.Kind == lexer.TokKwOctet {
				objectName = "OCTET STRING"
			}

			indexItems = append(indexItems, module.IndexItem{
				Implied: implied,
				Object:  objectName,
				Span:    item.Span,
			})
		}
		indexSpan = index.Span
		return indexItems, "", types.Span{}, indexSpan
	}

	if augments != nil {
		augmentsTarget = l.text(augments.Target)
		augmentsSpan = augments.Span
		return nil, augmentsTarget, augmentsSpan, types.Span{}
	}

	return nil, "", types.Span{}, types.Span{}
}
