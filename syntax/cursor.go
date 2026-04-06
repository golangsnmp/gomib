package syntax

// ClauseKind identifies which clause the cursor is inside within a definition.
// The zero value (empty string) means not inside any clause.
type ClauseKind string

const (
	ClauseSyntax         ClauseKind = "SYNTAX"
	ClauseWriteSyntax    ClauseKind = "WRITE-SYNTAX"
	ClauseAccess         ClauseKind = "ACCESS"
	ClauseStatus         ClauseKind = "STATUS"
	ClauseDescription    ClauseKind = "DESCRIPTION"
	ClauseReference      ClauseKind = "REFERENCE"
	ClauseIndex          ClauseKind = "INDEX"
	ClauseAugments       ClauseKind = "AUGMENTS"
	ClauseDefVal         ClauseKind = "DEFVAL"
	ClauseUnits          ClauseKind = "UNITS"
	ClauseDisplayHint    ClauseKind = "DISPLAY-HINT"
	ClauseObjects        ClauseKind = "OBJECTS"
	ClauseNotifications  ClauseKind = "NOTIFICATIONS"
	ClauseLastUpdated    ClauseKind = "LAST-UPDATED"
	ClauseOrganization   ClauseKind = "ORGANIZATION"
	ClauseContactInfo    ClauseKind = "CONTACT-INFO"
	ClauseEnterprise     ClauseKind = "ENTERPRISE"
	ClauseVariables      ClauseKind = "VARIABLES"
	ClauseProductRelease ClauseKind = "PRODUCT-RELEASE"
)

// CursorContext describes the syntactic context at a byte offset in a parsed
// module file. It identifies the containing module, definition, and
// sub-structures, and whether the offset falls inside a comment or string.
type CursorContext struct {
	// Module is the module containing the offset, or nil if outside all modules.
	Module *ModuleNode

	// Definition is the definition containing the offset, or nil if the offset
	// is between definitions or outside the module body.
	Definition DefinitionNode

	// Imports is non-nil when the offset falls inside the IMPORTS section.
	Imports *ImportsNode

	// ImportGroup is non-nil when the offset falls inside a specific
	// import group (symbols FROM module).
	ImportGroup *ImportGroupNode

	// OidValue is non-nil when the offset falls inside an OID value { ... }.
	OidValue *OidValueNode

	// Clause identifies which clause the cursor is inside (e.g. ClauseSyntax,
	// ClauseStatus). Empty when not inside any clause.
	Clause ClauseKind

	// InComment is true when the offset falls inside comment trivia.
	InComment bool

	// InString is true when the offset falls inside a string literal token.
	InString bool
}

// CursorContextAt determines the syntactic context at the given byte offset
// in a parsed module file. It walks the CST structure using span containment
// to identify the innermost context, then checks token-level trivia for
// comment and string detection.
func CursorContextAt(file *ModuleFile, offset ByteOffset) CursorContext {
	var ctx CursorContext

	// Find containing module.
	for i := range file.Modules {
		m := &file.Modules[i]
		if m.Span.Contains(offset) {
			ctx.Module = m
			break
		}
	}
	if ctx.Module == nil {
		return ctx
	}

	// Check imports section.
	if imp := ctx.Module.Imports; imp != nil && imp.Span.Contains(offset) {
		ctx.Imports = imp
		for i := range imp.Groups {
			g := &imp.Groups[i]
			if g.Span.Contains(offset) {
				ctx.ImportGroup = g
				break
			}
		}
	}

	// Find containing definition (skip if inside imports).
	if ctx.Imports == nil {
		for _, def := range ctx.Module.Body {
			if def.DefinitionSpan().Contains(offset) {
				ctx.Definition = def
				break
			}
		}
	}

	// Check OID values within the definition.
	if ctx.Definition != nil {
		ctx.OidValue = findOidValue(ctx.Definition, offset)
	}

	// Determine clause context within the definition.
	if ctx.Definition != nil {
		ctx.Clause = findClause(ctx.Definition, offset)
	}

	// Check whether offset falls inside comment trivia or a string token.
	ctx.InComment, ctx.InString = checkTokenContext(ctx.Module, offset)

	return ctx
}

// findOidValue checks whether the offset falls inside any OID value node
// within a definition, including nested OIDs in compliance/capability modules.
func findOidValue(def DefinitionNode, offset ByteOffset) *OidValueNode {
	if oid := definitionOid(def); oid != nil && oid.Span.Contains(offset) {
		return oid
	}

	// Check nested OIDs in compliance/capability sub-modules.
	switch d := def.(type) {
	case *ModuleComplianceNode:
		for i := range d.Modules {
			if oid := d.Modules[i].Oid; oid != nil && oid.Span.Contains(offset) {
				return oid
			}
		}
	case *AgentCapabilitiesNode:
		for i := range d.Modules {
			if oid := d.Modules[i].Oid; oid != nil && oid.Span.Contains(offset) {
				return oid
			}
		}
	}

	return nil
}

// definitionOid returns the top-level OID value node from a definition.
func definitionOid(def DefinitionNode) *OidValueNode {
	switch d := def.(type) {
	case *ObjectTypeNode:
		return d.Oid
	case *ModuleIdentityNode:
		return d.Oid
	case *ObjectIdentityNode:
		return d.Oid
	case *NotificationTypeNode:
		return d.Oid
	case *ObjectGroupNode:
		return d.Oid
	case *NotificationGroupNode:
		return d.Oid
	case *ModuleComplianceNode:
		return d.Oid
	case *AgentCapabilitiesNode:
		return d.Oid
	case *ValueAssignmentNode:
		return d.Oid
	default:
		return nil
	}
}

// findClause determines which clause the offset falls inside for a definition.
func findClause(def DefinitionNode, offset ByteOffset) ClauseKind {
	type check struct {
		span Span
		kind ClauseKind
	}
	var cc []check

	add := func(span Span, kind ClauseKind) {
		if span != (Span{}) {
			cc = append(cc, check{span, kind})
		}
	}

	switch d := def.(type) {
	case *ObjectTypeNode:
		if d.Syntax != nil {
			add(d.Syntax.Span, ClauseSyntax)
		}
		if d.Units != nil {
			add(d.Units.Span, ClauseUnits)
		}
		if d.Access != nil {
			add(d.Access.Span, ClauseAccess)
		}
		if d.Status != nil {
			add(d.Status.Span, ClauseStatus)
		}
		if d.Description != nil {
			add(d.Description.Span, ClauseDescription)
		}
		if d.Reference != nil {
			add(d.Reference.Span, ClauseReference)
		}
		if d.Index != nil {
			add(d.Index.Span, ClauseIndex)
		}
		if d.Augments != nil {
			add(d.Augments.Span, ClauseAugments)
		}
		if d.DefVal != nil {
			add(d.DefVal.Span, ClauseDefVal)
		}

	case *TextualConventionNode:
		if d.DisplayHint != nil {
			add(d.DisplayHint.Span, ClauseDisplayHint)
		}
		if d.Status != nil {
			add(d.Status.Span, ClauseStatus)
		}
		if d.Description != nil {
			add(d.Description.Span, ClauseDescription)
		}
		if d.Reference != nil {
			add(d.Reference.Span, ClauseReference)
		}
		if d.Syntax != nil {
			add(d.Syntax.Span, ClauseSyntax)
		}

	case *ModuleIdentityNode:
		if d.LastUpdated != nil {
			add(d.LastUpdated.Span, ClauseLastUpdated)
		}
		if d.Organization != nil {
			add(d.Organization.Span, ClauseOrganization)
		}
		if d.ContactInfo != nil {
			add(d.ContactInfo.Span, ClauseContactInfo)
		}
		if d.Description != nil {
			add(d.Description.Span, ClauseDescription)
		}

	case *ObjectIdentityNode:
		if d.Status != nil {
			add(d.Status.Span, ClauseStatus)
		}
		if d.Description != nil {
			add(d.Description.Span, ClauseDescription)
		}
		if d.Reference != nil {
			add(d.Reference.Span, ClauseReference)
		}

	case *NotificationTypeNode:
		if d.Objects != nil {
			add(d.Objects.Span, ClauseObjects)
		}
		if d.Status != nil {
			add(d.Status.Span, ClauseStatus)
		}
		if d.Description != nil {
			add(d.Description.Span, ClauseDescription)
		}
		if d.Reference != nil {
			add(d.Reference.Span, ClauseReference)
		}

	case *TrapTypeNode:
		if d.Enterprise != nil {
			add(d.Enterprise.Span, ClauseEnterprise)
		}
		if d.Variables != nil {
			add(d.Variables.Span, ClauseVariables)
		}
		if d.Description != nil {
			add(d.Description.Span, ClauseDescription)
		}
		if d.Reference != nil {
			add(d.Reference.Span, ClauseReference)
		}

	case *ObjectGroupNode:
		if d.Objects != nil {
			add(d.Objects.Span, ClauseObjects)
		}
		if d.Status != nil {
			add(d.Status.Span, ClauseStatus)
		}
		if d.Description != nil {
			add(d.Description.Span, ClauseDescription)
		}
		if d.Reference != nil {
			add(d.Reference.Span, ClauseReference)
		}

	case *NotificationGroupNode:
		if d.Notifications != nil {
			add(d.Notifications.Span, ClauseNotifications)
		}
		if d.Status != nil {
			add(d.Status.Span, ClauseStatus)
		}
		if d.Description != nil {
			add(d.Description.Span, ClauseDescription)
		}
		if d.Reference != nil {
			add(d.Reference.Span, ClauseReference)
		}

	case *ModuleComplianceNode:
		if d.Status != nil {
			add(d.Status.Span, ClauseStatus)
		}
		if d.Description != nil {
			add(d.Description.Span, ClauseDescription)
		}
		if d.Reference != nil {
			add(d.Reference.Span, ClauseReference)
		}

	case *AgentCapabilitiesNode:
		if d.ProductRelease != nil {
			add(d.ProductRelease.Span, ClauseProductRelease)
		}
		if d.Status != nil {
			add(d.Status.Span, ClauseStatus)
		}
		if d.Description != nil {
			add(d.Description.Span, ClauseDescription)
		}
		if d.Reference != nil {
			add(d.Reference.Span, ClauseReference)
		}

	case *TypeAssignmentNode:
		if d.Syntax != nil {
			add(d.Syntax.TypeSyntaxSpan(), ClauseSyntax)
		}
	}

	for _, c := range cc {
		if c.span.Contains(offset) {
			return c.kind
		}
	}
	return ""
}

// checkTokenContext walks the tokens of a module to determine whether the
// offset falls inside a comment or string literal.
func checkTokenContext(mod *ModuleNode, offset ByteOffset) (inComment, inString bool) {
	done := false
	mod.WalkTokens(func(tok SyntaxToken) {
		if done {
			return
		}

		// Check leading trivia.
		for _, t := range tok.LeadingTrivia {
			if t.Span.Contains(offset) {
				if t.Kind == TriviaComment {
					inComment = true
				}
				done = true
				return
			}
		}

		// Check the token itself.
		if tok.Span.Contains(offset) {
			if tok.Kind == TokQuotedString || tok.Kind == TokHexString || tok.Kind == TokBinString {
				inString = true
			}
			done = true
			return
		}

		// Check trailing trivia.
		for _, t := range tok.TrailingTrivia {
			if t.Span.Contains(offset) {
				if t.Kind == TriviaComment {
					inComment = true
				}
				done = true
				return
			}
		}

		// Once past the offset, stop checking.
		if tok.Span.Start > offset {
			done = true
		}
	})
	return inComment, inString
}
