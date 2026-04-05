package cst

import "bytes"

// walkToken calls fn for a non-zero SyntaxToken.
func walkToken(tok SyntaxToken, fn func(SyntaxToken)) {
	if !tok.IsZero() {
		fn(tok)
	}
}

// walkInterleavedTokens emits items and separators in alternating order:
// items[0], seps[0], items[1], seps[1], ...
func walkInterleavedTokens(items, seps []SyntaxToken, fn func(SyntaxToken)) {
	for i, item := range items {
		walkToken(item, fn)
		if i < len(seps) {
			walkToken(seps[i], fn)
		}
	}
}

// ReconstructText rebuilds the source text by walking all tokens in order and
// concatenating their trivia and token text. A correct CST round-trips to the
// original source.
func ReconstructText(file *ModuleFile, source []byte) []byte {
	var buf bytes.Buffer
	file.WalkTokens(func(tok SyntaxToken) {
		for _, t := range tok.LeadingTrivia {
			buf.Write(source[t.Span.Start:t.Span.End])
		}
		buf.Write(source[tok.Span.Start:tok.Span.End])
		for _, t := range tok.TrailingTrivia {
			buf.Write(source[t.Span.Start:t.Span.End])
		}
	})
	return buf.Bytes()
}

// --- ModuleFile, ModuleNode, ImportsNode, ImportGroupNode ---

// WalkTokens visits all tokens in source order.
func (n *ModuleFile) WalkTokens(fn func(SyntaxToken)) {
	for i := range n.Modules {
		n.Modules[i].WalkTokens(fn)
	}
	walkToken(n.EOF, fn)
}

// WalkTokens visits all tokens in source order.
func (n *ModuleNode) WalkTokens(fn func(SyntaxToken)) {
	walkToken(n.Name, fn)
	walkToken(n.Definitions, fn)
	walkToken(n.Assign, fn)
	walkToken(n.Begin, fn)
	if n.Imports != nil {
		n.Imports.WalkTokens(fn)
	}
	for _, d := range n.Body {
		d.WalkTokens(fn)
	}
	walkToken(n.End, fn)
}

// WalkTokens visits all tokens in source order.
func (n *ImportsNode) WalkTokens(fn func(SyntaxToken)) {
	walkToken(n.Keyword, fn)
	for i := range n.Groups {
		n.Groups[i].WalkTokens(fn)
	}
	walkToken(n.Semi, fn)
}

// WalkTokens visits all tokens in source order.
func (n *ImportGroupNode) WalkTokens(fn func(SyntaxToken)) {
	walkInterleavedTokens(n.Symbols, n.Commas, fn)
	walkToken(n.From, fn)
	walkToken(n.Module, fn)
}

// --- Definition nodes ---

// WalkTokens visits all tokens in source order.
func (n *ObjectTypeNode) WalkTokens(fn func(SyntaxToken)) {
	walkToken(n.Name, fn)
	walkToken(n.Keyword, fn)
	if n.Syntax != nil {
		n.Syntax.WalkTokens(fn)
	}
	if n.Units != nil {
		n.Units.WalkTokens(fn)
	}
	if n.Access != nil {
		n.Access.WalkTokens(fn)
	}
	if n.Status != nil {
		n.Status.WalkTokens(fn)
	}
	if n.Description != nil {
		n.Description.WalkTokens(fn)
	}
	if n.Reference != nil {
		n.Reference.WalkTokens(fn)
	}
	if n.Index != nil {
		n.Index.WalkTokens(fn)
	}
	if n.Augments != nil {
		n.Augments.WalkTokens(fn)
	}
	if n.DefVal != nil {
		n.DefVal.WalkTokens(fn)
	}
	walkToken(n.Assign, fn)
	if n.Oid != nil {
		n.Oid.WalkTokens(fn)
	}
}

// WalkTokens visits all tokens in source order.
func (n *ModuleIdentityNode) WalkTokens(fn func(SyntaxToken)) {
	walkToken(n.Name, fn)
	walkToken(n.Keyword, fn)
	if n.LastUpdated != nil {
		n.LastUpdated.WalkTokens(fn)
	}
	if n.Organization != nil {
		n.Organization.WalkTokens(fn)
	}
	if n.ContactInfo != nil {
		n.ContactInfo.WalkTokens(fn)
	}
	if n.Description != nil {
		n.Description.WalkTokens(fn)
	}
	for i := range n.Revisions {
		n.Revisions[i].WalkTokens(fn)
	}
	walkToken(n.Assign, fn)
	if n.Oid != nil {
		n.Oid.WalkTokens(fn)
	}
}

// WalkTokens visits all tokens in source order.
func (n *ObjectIdentityNode) WalkTokens(fn func(SyntaxToken)) {
	walkToken(n.Name, fn)
	walkToken(n.Keyword, fn)
	if n.Status != nil {
		n.Status.WalkTokens(fn)
	}
	if n.Description != nil {
		n.Description.WalkTokens(fn)
	}
	if n.Reference != nil {
		n.Reference.WalkTokens(fn)
	}
	walkToken(n.Assign, fn)
	if n.Oid != nil {
		n.Oid.WalkTokens(fn)
	}
}

// WalkTokens visits all tokens in source order.
func (n *NotificationTypeNode) WalkTokens(fn func(SyntaxToken)) {
	walkToken(n.Name, fn)
	walkToken(n.Keyword, fn)
	if n.Objects != nil {
		n.Objects.WalkTokens(fn)
	}
	if n.Status != nil {
		n.Status.WalkTokens(fn)
	}
	if n.Description != nil {
		n.Description.WalkTokens(fn)
	}
	if n.Reference != nil {
		n.Reference.WalkTokens(fn)
	}
	walkToken(n.Assign, fn)
	if n.Oid != nil {
		n.Oid.WalkTokens(fn)
	}
}

// WalkTokens visits all tokens in source order.
func (n *TrapTypeNode) WalkTokens(fn func(SyntaxToken)) {
	walkToken(n.Name, fn)
	walkToken(n.Keyword, fn)
	if n.Enterprise != nil {
		n.Enterprise.WalkTokens(fn)
	}
	if n.Variables != nil {
		n.Variables.WalkTokens(fn)
	}
	if n.Description != nil {
		n.Description.WalkTokens(fn)
	}
	if n.Reference != nil {
		n.Reference.WalkTokens(fn)
	}
	walkToken(n.Assign, fn)
	walkToken(n.Value, fn)
}

// WalkTokens visits all tokens in source order.
func (n *TextualConventionNode) WalkTokens(fn func(SyntaxToken)) {
	walkToken(n.Name, fn)
	walkToken(n.Assign, fn)
	walkToken(n.Keyword, fn)
	if n.DisplayHint != nil {
		n.DisplayHint.WalkTokens(fn)
	}
	if n.Status != nil {
		n.Status.WalkTokens(fn)
	}
	if n.Description != nil {
		n.Description.WalkTokens(fn)
	}
	if n.Reference != nil {
		n.Reference.WalkTokens(fn)
	}
	if n.Syntax != nil {
		n.Syntax.WalkTokens(fn)
	}
}

// WalkTokens visits all tokens in source order.
func (n *TypeAssignmentNode) WalkTokens(fn func(SyntaxToken)) {
	walkToken(n.Name, fn)
	walkToken(n.Assign, fn)
	if n.Syntax != nil {
		n.Syntax.WalkTokens(fn)
	}
}

// WalkTokens visits all tokens in source order.
func (n *ValueAssignmentNode) WalkTokens(fn func(SyntaxToken)) {
	walkToken(n.Name, fn)
	walkToken(n.Object, fn)
	walkToken(n.Identifier, fn)
	walkToken(n.Assign, fn)
	if n.Oid != nil {
		n.Oid.WalkTokens(fn)
	}
}

// WalkTokens visits all tokens in source order.
func (n *ObjectGroupNode) WalkTokens(fn func(SyntaxToken)) {
	walkToken(n.Name, fn)
	walkToken(n.Keyword, fn)
	if n.Objects != nil {
		n.Objects.WalkTokens(fn)
	}
	if n.Status != nil {
		n.Status.WalkTokens(fn)
	}
	if n.Description != nil {
		n.Description.WalkTokens(fn)
	}
	if n.Reference != nil {
		n.Reference.WalkTokens(fn)
	}
	walkToken(n.Assign, fn)
	if n.Oid != nil {
		n.Oid.WalkTokens(fn)
	}
}

// WalkTokens visits all tokens in source order.
func (n *NotificationGroupNode) WalkTokens(fn func(SyntaxToken)) {
	walkToken(n.Name, fn)
	walkToken(n.Keyword, fn)
	if n.Notifications != nil {
		n.Notifications.WalkTokens(fn)
	}
	if n.Status != nil {
		n.Status.WalkTokens(fn)
	}
	if n.Description != nil {
		n.Description.WalkTokens(fn)
	}
	if n.Reference != nil {
		n.Reference.WalkTokens(fn)
	}
	walkToken(n.Assign, fn)
	if n.Oid != nil {
		n.Oid.WalkTokens(fn)
	}
}

// WalkTokens visits all tokens in source order.
func (n *ModuleComplianceNode) WalkTokens(fn func(SyntaxToken)) {
	walkToken(n.Name, fn)
	walkToken(n.Keyword, fn)
	if n.Status != nil {
		n.Status.WalkTokens(fn)
	}
	if n.Description != nil {
		n.Description.WalkTokens(fn)
	}
	if n.Reference != nil {
		n.Reference.WalkTokens(fn)
	}
	for i := range n.Modules {
		n.Modules[i].WalkTokens(fn)
	}
	walkToken(n.Assign, fn)
	if n.Oid != nil {
		n.Oid.WalkTokens(fn)
	}
}

// WalkTokens visits all tokens in source order.
func (n *AgentCapabilitiesNode) WalkTokens(fn func(SyntaxToken)) {
	walkToken(n.Name, fn)
	walkToken(n.Keyword, fn)
	if n.ProductRelease != nil {
		n.ProductRelease.WalkTokens(fn)
	}
	if n.Status != nil {
		n.Status.WalkTokens(fn)
	}
	if n.Description != nil {
		n.Description.WalkTokens(fn)
	}
	if n.Reference != nil {
		n.Reference.WalkTokens(fn)
	}
	for i := range n.Modules {
		n.Modules[i].WalkTokens(fn)
	}
	walkToken(n.Assign, fn)
	if n.Oid != nil {
		n.Oid.WalkTokens(fn)
	}
}

// WalkTokens visits all tokens in source order.
func (n *MacroDefinitionNode) WalkTokens(fn func(SyntaxToken)) {
	walkToken(n.Name, fn)
	walkToken(n.Macro, fn)
	walkToken(n.Assign, fn)
	walkToken(n.Begin, fn)
	for _, tok := range n.Body {
		walkToken(tok, fn)
	}
	walkToken(n.End, fn)
}

// --- Clause nodes ---

// WalkTokens visits all tokens in source order.
func (n *SyntaxClauseNode) WalkTokens(fn func(SyntaxToken)) {
	walkToken(n.Keyword, fn)
	if n.Syntax != nil {
		n.Syntax.WalkTokens(fn)
	}
}

// WalkTokens visits all tokens in source order.
func (n *AccessClauseNode) WalkTokens(fn func(SyntaxToken)) {
	walkToken(n.Keyword, fn)
	walkToken(n.Value, fn)
}

// WalkTokens visits all tokens in source order.
func (n *StatusClauseNode) WalkTokens(fn func(SyntaxToken)) {
	walkToken(n.Keyword, fn)
	walkToken(n.Value, fn)
}

// WalkTokens visits all tokens in source order.
func (n *DescriptionClauseNode) WalkTokens(fn func(SyntaxToken)) {
	walkToken(n.Keyword, fn)
	walkToken(n.Value, fn)
}

// WalkTokens visits all tokens in source order.
func (n *ReferenceClauseNode) WalkTokens(fn func(SyntaxToken)) {
	walkToken(n.Keyword, fn)
	walkToken(n.Value, fn)
}

// WalkTokens visits all tokens in source order.
func (n *UnitsClauseNode) WalkTokens(fn func(SyntaxToken)) {
	walkToken(n.Keyword, fn)
	walkToken(n.Value, fn)
}

// WalkTokens visits all tokens in source order.
func (n *DisplayHintClauseNode) WalkTokens(fn func(SyntaxToken)) {
	walkToken(n.Keyword, fn)
	walkToken(n.Value, fn)
}

// WalkTokens visits all tokens in source order.
func (n *IndexClauseNode) WalkTokens(fn func(SyntaxToken)) {
	walkToken(n.Keyword, fn)
	walkToken(n.LBrace, fn)
	// Items and commas are interleaved: item[0], comma[0], item[1], ...
	for i := range n.Items {
		n.Items[i].WalkTokens(fn)
		if i < len(n.Commas) {
			walkToken(n.Commas[i], fn)
		}
	}
	walkToken(n.RBrace, fn)
}

// WalkTokens visits all tokens in source order.
func (n *IndexItemNode) WalkTokens(fn func(SyntaxToken)) {
	walkToken(n.Implied, fn)
	walkToken(n.Object, fn)
	walkToken(n.StringKeyword, fn)
}

// WalkTokens visits all tokens in source order.
func (n *AugmentsClauseNode) WalkTokens(fn func(SyntaxToken)) {
	walkToken(n.Keyword, fn)
	walkToken(n.LBrace, fn)
	walkToken(n.Target, fn)
	walkToken(n.RBrace, fn)
}

// WalkTokens visits all tokens in source order.
func (n *DefValClauseNode) WalkTokens(fn func(SyntaxToken)) {
	walkToken(n.Keyword, fn)
	walkToken(n.LBrace, fn)
	for _, tok := range n.Content {
		walkToken(tok, fn)
	}
	walkToken(n.RBrace, fn)
}

// WalkTokens visits all tokens in source order.
func (n *ObjectsClauseNode) WalkTokens(fn func(SyntaxToken)) {
	walkToken(n.Keyword, fn)
	walkToken(n.LBrace, fn)
	walkInterleavedTokens(n.Names, n.Commas, fn)
	walkToken(n.RBrace, fn)
}

// WalkTokens visits all tokens in source order.
func (n *NotificationsClauseNode) WalkTokens(fn func(SyntaxToken)) {
	walkToken(n.Keyword, fn)
	walkToken(n.LBrace, fn)
	walkInterleavedTokens(n.Names, n.Commas, fn)
	walkToken(n.RBrace, fn)
}

// WalkTokens visits all tokens in source order.
func (n *RevisionNode) WalkTokens(fn func(SyntaxToken)) {
	walkToken(n.Keyword, fn)
	walkToken(n.Date, fn)
	if n.Description != nil {
		n.Description.WalkTokens(fn)
	}
}

// WalkTokens visits all tokens in source order.
func (n *LastUpdatedClauseNode) WalkTokens(fn func(SyntaxToken)) {
	walkToken(n.Keyword, fn)
	walkToken(n.Value, fn)
}

// WalkTokens visits all tokens in source order.
func (n *OrganizationClauseNode) WalkTokens(fn func(SyntaxToken)) {
	walkToken(n.Keyword, fn)
	walkToken(n.Value, fn)
}

// WalkTokens visits all tokens in source order.
func (n *ContactInfoClauseNode) WalkTokens(fn func(SyntaxToken)) {
	walkToken(n.Keyword, fn)
	walkToken(n.Value, fn)
}

// WalkTokens visits all tokens in source order.
func (n *EnterpriseClauseNode) WalkTokens(fn func(SyntaxToken)) {
	walkToken(n.Keyword, fn)
	walkToken(n.Name, fn)
}

// WalkTokens visits all tokens in source order.
func (n *VariablesClauseNode) WalkTokens(fn func(SyntaxToken)) {
	walkToken(n.Keyword, fn)
	walkToken(n.LBrace, fn)
	walkInterleavedTokens(n.Names, n.Commas, fn)
	walkToken(n.RBrace, fn)
}

// WalkTokens visits all tokens in source order.
func (n *ProductReleaseClauseNode) WalkTokens(fn func(SyntaxToken)) {
	walkToken(n.Keyword, fn)
	walkToken(n.Value, fn)
}

// --- OID nodes ---

// WalkTokens visits all tokens in source order.
func (n *OidValueNode) WalkTokens(fn func(SyntaxToken)) {
	walkToken(n.LBrace, fn)
	for i := range n.Components {
		n.Components[i].WalkTokens(fn)
	}
	walkToken(n.RBrace, fn)
}

// WalkTokens visits all tokens in source order.
func (n *OidComponentNode) WalkTokens(fn func(SyntaxToken)) {
	walkToken(n.Module, fn)
	walkToken(n.Dot, fn)
	walkToken(n.Name, fn)
	walkToken(n.LParen, fn)
	walkToken(n.Number, fn)
	walkToken(n.RParen, fn)
}

// --- Syntax nodes ---

// WalkTokens visits all tokens in source order.
func (n *TypeRefSyntaxNode) WalkTokens(fn func(SyntaxToken)) {
	walkToken(n.Name, fn)
}

// WalkTokens visits all tokens in source order.
func (n *IntegerEnumSyntaxNode) WalkTokens(fn func(SyntaxToken)) {
	walkToken(n.Base, fn)
	walkToken(n.LBrace, fn)
	for i := range n.Values {
		n.Values[i].WalkTokens(fn)
		if i < len(n.Commas) {
			walkToken(n.Commas[i], fn)
		}
	}
	walkToken(n.RBrace, fn)
}

// WalkTokens visits all tokens in source order.
func (n *EnumValueNode) WalkTokens(fn func(SyntaxToken)) {
	walkToken(n.Label, fn)
	walkToken(n.LParen, fn)
	walkToken(n.Value, fn)
	walkToken(n.RParen, fn)
}

// WalkTokens visits all tokens in source order.
func (n *BitsSyntaxNode) WalkTokens(fn func(SyntaxToken)) {
	walkToken(n.Keyword, fn)
	walkToken(n.LBrace, fn)
	for i := range n.Values {
		n.Values[i].WalkTokens(fn)
		if i < len(n.Commas) {
			walkToken(n.Commas[i], fn)
		}
	}
	walkToken(n.RBrace, fn)
}

// WalkTokens visits all tokens in source order.
func (n *ConstrainedSyntaxNode) WalkTokens(fn func(SyntaxToken)) {
	if n.Base != nil {
		n.Base.WalkTokens(fn)
	}
	if n.Constraint != nil {
		n.Constraint.WalkTokens(fn)
	}
}

// WalkTokens visits all tokens in source order.
func (n *ConstraintNode) WalkTokens(fn func(SyntaxToken)) {
	walkToken(n.LParen, fn)
	walkToken(n.Size, fn)
	walkToken(n.InnerLParen, fn)
	// Ranges and pipes are interleaved: range[0], pipe[0], range[1], ...
	for i := range n.Ranges {
		n.Ranges[i].WalkTokens(fn)
		if i < len(n.Pipes) {
			walkToken(n.Pipes[i], fn)
		}
	}
	walkToken(n.InnerRParen, fn)
	walkToken(n.RParen, fn)
}

// WalkTokens visits all tokens in source order.
func (n *RangeNode) WalkTokens(fn func(SyntaxToken)) {
	walkToken(n.Min, fn)
	walkToken(n.DotDot, fn)
	walkToken(n.Max, fn)
}

// WalkTokens visits all tokens in source order.
func (n *SequenceOfSyntaxNode) WalkTokens(fn func(SyntaxToken)) {
	walkToken(n.Sequence, fn)
	walkToken(n.Of, fn)
	walkToken(n.Entry, fn)
}

// WalkTokens visits all tokens in source order.
func (n *SequenceSyntaxNode) WalkTokens(fn func(SyntaxToken)) {
	walkToken(n.Keyword, fn)
	walkToken(n.LBrace, fn)
	for i := range n.Fields {
		n.Fields[i].WalkTokens(fn)
		if i < len(n.Commas) {
			walkToken(n.Commas[i], fn)
		}
	}
	walkToken(n.RBrace, fn)
}

// WalkTokens visits all tokens in source order.
func (n *SequenceFieldNode) WalkTokens(fn func(SyntaxToken)) {
	walkToken(n.Name, fn)
	if n.Syntax != nil {
		n.Syntax.WalkTokens(fn)
	}
}

// WalkTokens visits all tokens in source order.
func (n *OctetStringSyntaxNode) WalkTokens(fn func(SyntaxToken)) {
	walkToken(n.Octet, fn)
	walkToken(n.String, fn)
}

// WalkTokens visits all tokens in source order.
func (n *ObjectIdentifierSyntaxNode) WalkTokens(fn func(SyntaxToken)) {
	walkToken(n.Object, fn)
	walkToken(n.Identifier, fn)
}

// WalkTokens visits all tokens in source order.
func (n *TaggedSyntaxNode) WalkTokens(fn func(SyntaxToken)) {
	walkToken(n.LBracket, fn)
	walkToken(n.TagClass, fn)
	walkToken(n.TagNum, fn)
	walkToken(n.RBracket, fn)
	walkToken(n.Implicit, fn)
	if n.Inner != nil {
		n.Inner.WalkTokens(fn)
	}
}

// WalkTokens visits all tokens in source order.
func (n *ChoiceSyntaxNode) WalkTokens(fn func(SyntaxToken)) {
	walkToken(n.Keyword, fn)
	walkToken(n.LBrace, fn)
	for i := range n.Fields {
		n.Fields[i].WalkTokens(fn)
		if i < len(n.Commas) {
			walkToken(n.Commas[i], fn)
		}
	}
	walkToken(n.RBrace, fn)
}

// --- Conformance nodes ---

// WalkTokens visits all tokens in source order.
// Groups and Objects are interleaved by their source span to preserve
// the original ordering (they can appear in any order in the source).
func (n *ComplianceModuleNode) WalkTokens(fn func(SyntaxToken)) {
	walkToken(n.Keyword, fn)
	walkToken(n.ModuleName, fn)
	if n.Oid != nil {
		n.Oid.WalkTokens(fn)
	}
	if n.MandatoryGroups != nil {
		n.MandatoryGroups.WalkTokens(fn)
	}
	gi, oi := 0, 0
	for gi < len(n.Groups) || oi < len(n.Objects) {
		switch {
		case gi >= len(n.Groups):
			n.Objects[oi].WalkTokens(fn)
			oi++
		case oi >= len(n.Objects):
			n.Groups[gi].WalkTokens(fn)
			gi++
		case n.Groups[gi].Span.Start < n.Objects[oi].Span.Start:
			n.Groups[gi].WalkTokens(fn)
			gi++
		default:
			n.Objects[oi].WalkTokens(fn)
			oi++
		}
	}
}

// WalkTokens visits all tokens in source order.
func (n *MandatoryGroupsNode) WalkTokens(fn func(SyntaxToken)) {
	walkToken(n.Keyword, fn)
	walkToken(n.LBrace, fn)
	walkInterleavedTokens(n.Names, n.Commas, fn)
	walkToken(n.RBrace, fn)
}

// WalkTokens visits all tokens in source order.
func (n *ComplianceGroupNode) WalkTokens(fn func(SyntaxToken)) {
	walkToken(n.Keyword, fn)
	walkToken(n.Name, fn)
	if n.Description != nil {
		n.Description.WalkTokens(fn)
	}
}

// WalkTokens visits all tokens in source order.
func (n *ComplianceObjectNode) WalkTokens(fn func(SyntaxToken)) {
	walkToken(n.Keyword, fn)
	walkToken(n.Name, fn)
	if n.Syntax != nil {
		n.Syntax.WalkTokens(fn)
	}
	if n.WriteSyntax != nil {
		n.WriteSyntax.WalkTokens(fn)
	}
	if n.Access != nil {
		n.Access.WalkTokens(fn)
	}
	if n.Description != nil {
		n.Description.WalkTokens(fn)
	}
}

// WalkTokens visits all tokens in source order.
func (n *WriteSyntaxClauseNode) WalkTokens(fn func(SyntaxToken)) {
	walkToken(n.Keyword, fn)
	if n.Syntax != nil {
		n.Syntax.WalkTokens(fn)
	}
}

// WalkTokens visits all tokens in source order.
func (n *CapabilityModuleNode) WalkTokens(fn func(SyntaxToken)) {
	walkToken(n.Keyword, fn)
	walkToken(n.ModuleName, fn)
	if n.Oid != nil {
		n.Oid.WalkTokens(fn)
	}
	if n.Includes != nil {
		n.Includes.WalkTokens(fn)
	}
	for i := range n.Variations {
		n.Variations[i].WalkTokens(fn)
	}
}

// WalkTokens visits all tokens in source order.
func (n *IncludesNode) WalkTokens(fn func(SyntaxToken)) {
	walkToken(n.Keyword, fn)
	walkToken(n.LBrace, fn)
	walkInterleavedTokens(n.Names, n.Commas, fn)
	walkToken(n.RBrace, fn)
}

// WalkTokens visits all tokens in source order.
func (n *CapabilityVariationNode) WalkTokens(fn func(SyntaxToken)) {
	walkToken(n.Keyword, fn)
	walkToken(n.Name, fn)
	if n.Syntax != nil {
		n.Syntax.WalkTokens(fn)
	}
	if n.WriteSyntax != nil {
		n.WriteSyntax.WalkTokens(fn)
	}
	if n.Access != nil {
		n.Access.WalkTokens(fn)
	}
	if n.CreationRequires != nil {
		n.CreationRequires.WalkTokens(fn)
	}
	if n.DefVal != nil {
		n.DefVal.WalkTokens(fn)
	}
	if n.Description != nil {
		n.Description.WalkTokens(fn)
	}
}

// WalkTokens visits all tokens in source order.
func (n *CreationRequiresNode) WalkTokens(fn func(SyntaxToken)) {
	walkToken(n.Keyword, fn)
	walkToken(n.LBrace, fn)
	walkInterleavedTokens(n.Names, n.Commas, fn)
	walkToken(n.RBrace, fn)
}

// --- Error node ---

// WalkTokens visits all tokens in source order.
func (n *ErrorNode) WalkTokens(fn func(SyntaxToken)) {
	for _, tok := range n.Tokens {
		walkToken(tok, fn)
	}
}
