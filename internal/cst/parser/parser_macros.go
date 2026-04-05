package parser

import (
	"github.com/golangsnmp/gomib/internal/cst"
	"github.com/golangsnmp/gomib/internal/lexer"
	"github.com/golangsnmp/gomib/internal/types"
)

// parseValueAssignment parses: name OBJECT IDENTIFIER ::= { ... }
func (p *Parser) parseValueAssignment() (cst.DefinitionNode, *types.SpanDiagnostic) {
	start := p.currentSpan().Start
	node := &cst.ValueAssignmentNode{}

	nameTok := p.advance()
	node.Name = nameTok
	name := p.text(nameTok.Span)
	p.validateIdentifier(name, nameTok.Span)
	p.validateValueReference(name, nameTok.Span)

	objTok, err := p.expect(lexer.TokKwObject)
	if err != nil {
		return nil, err
	}
	node.Object = objTok

	idTok, err := p.expect(lexer.TokKwIdentifier)
	if err != nil {
		return nil, err
	}
	node.Identifier = idTok

	assignTok, err := p.expect(lexer.TokColonColonEqual)
	if err != nil {
		return nil, err
	}
	node.Assign = assignTok

	oid, err := p.parseOidValue()
	if err != nil {
		return nil, err
	}
	node.Oid = oid

	node.Span = types.NewSpan(start, p.lastEnd)
	return node, nil
}

// parseObjectType parses an OBJECT-TYPE macro invocation.
func (p *Parser) parseObjectType() (cst.DefinitionNode, *types.SpanDiagnostic) {
	start := p.currentSpan().Start
	node := &cst.ObjectTypeNode{}

	nameTok := p.advance()
	node.Name = nameTok
	name := p.text(nameTok.Span)
	p.validateIdentifier(name, nameTok.Span)
	p.validateValueReference(name, nameTok.Span)

	kwTok, err := p.expect(lexer.TokKwObjectType)
	if err != nil {
		return nil, err
	}
	node.Keyword = kwTok

	// SYNTAX (required)
	syntax, err := p.parseSyntaxClause()
	if err != nil {
		return nil, err
	}
	node.Syntax = syntax

	// UNITS (optional)
	if p.check(lexer.TokKwUnits) {
		units, err := p.parseUnitsClause()
		if err != nil {
			return nil, err
		}
		node.Units = units
	}

	// MAX-ACCESS / ACCESS (required)
	access, err := p.parseAccessClause()
	if err != nil {
		return nil, err
	}
	node.Access = access

	// STATUS (technically required but some vendor MIBs omit it)
	if p.check(lexer.TokKwStatus) {
		status, err := p.parseStatusClause()
		if err != nil {
			return nil, err
		}
		node.Status = status
	}

	// DESCRIPTION (optional)
	if p.check(lexer.TokKwDescription) {
		desc, err := p.parseDescriptionClause()
		if err != nil {
			return nil, err
		}
		node.Description = desc
	}

	// REFERENCE (optional)
	if p.check(lexer.TokKwReference) {
		ref, err := p.parseReferenceClause()
		if err != nil {
			return nil, err
		}
		node.Reference = ref
	}

	// INDEX or AUGMENTS (optional)
	if p.check(lexer.TokKwIndex) {
		index, err := p.parseIndexClause()
		if err != nil {
			return nil, err
		}
		node.Index = index
	} else if p.check(lexer.TokKwAugments) {
		augments, err := p.parseAugmentsClause()
		if err != nil {
			return nil, err
		}
		node.Augments = augments
	}

	// DEFVAL (optional)
	if p.check(lexer.TokKwDefval) {
		defval, err := p.parseDefValClause()
		if err != nil {
			return nil, err
		}
		node.DefVal = defval
	}

	// ::=
	assignTok, err := p.expect(lexer.TokColonColonEqual)
	if err != nil {
		return nil, err
	}
	node.Assign = assignTok

	// OID
	oid, err := p.parseOidValue()
	if err != nil {
		return nil, err
	}
	node.Oid = oid

	node.Span = types.NewSpan(start, p.lastEnd)
	return node, nil
}

// parseModuleIdentity parses a MODULE-IDENTITY macro invocation.
func (p *Parser) parseModuleIdentity() (cst.DefinitionNode, *types.SpanDiagnostic) {
	start := p.currentSpan().Start
	node := &cst.ModuleIdentityNode{}

	nameTok := p.advance()
	node.Name = nameTok
	name := p.text(nameTok.Span)
	p.validateIdentifier(name, nameTok.Span)
	p.validateValueReference(name, nameTok.Span)

	kwTok, err := p.expect(lexer.TokKwModuleIdentity)
	if err != nil {
		return nil, err
	}
	node.Keyword = kwTok

	// LAST-UPDATED
	lastUpdated, err := p.parseLastUpdatedClause()
	if err != nil {
		return nil, err
	}
	node.LastUpdated = lastUpdated

	// ORGANIZATION
	org, err := p.parseOrganizationClause()
	if err != nil {
		return nil, err
	}
	node.Organization = org

	// CONTACT-INFO
	contact, err := p.parseContactInfoClause()
	if err != nil {
		return nil, err
	}
	node.ContactInfo = contact

	// DESCRIPTION
	desc, err := p.parseDescriptionClause()
	if err != nil {
		return nil, err
	}
	node.Description = desc

	// REVISION clauses (optional, multiple)
	for p.check(lexer.TokKwRevision) {
		rev, err := p.parseRevisionClause()
		if err != nil {
			return nil, err
		}
		node.Revisions = append(node.Revisions, *rev)
	}

	// ::=
	assignTok, err := p.expect(lexer.TokColonColonEqual)
	if err != nil {
		return nil, err
	}
	node.Assign = assignTok

	// OID
	oid, err := p.parseOidValue()
	if err != nil {
		return nil, err
	}
	node.Oid = oid

	node.Span = types.NewSpan(start, p.lastEnd)
	return node, nil
}

// parseObjectIdentity parses an OBJECT-IDENTITY macro invocation.
func (p *Parser) parseObjectIdentity() (cst.DefinitionNode, *types.SpanDiagnostic) {
	start := p.currentSpan().Start
	node := &cst.ObjectIdentityNode{}

	nameTok := p.advance()
	node.Name = nameTok
	name := p.text(nameTok.Span)
	p.validateIdentifier(name, nameTok.Span)
	p.validateValueReference(name, nameTok.Span)

	kwTok, err := p.expect(lexer.TokKwObjectIdentity)
	if err != nil {
		return nil, err
	}
	node.Keyword = kwTok

	// STATUS
	status, err := p.parseStatusClause()
	if err != nil {
		return nil, err
	}
	node.Status = status

	// DESCRIPTION
	desc, err := p.parseDescriptionClause()
	if err != nil {
		return nil, err
	}
	node.Description = desc

	// REFERENCE (optional)
	if p.check(lexer.TokKwReference) {
		ref, err := p.parseReferenceClause()
		if err != nil {
			return nil, err
		}
		node.Reference = ref
	}

	// ::=
	assignTok, err := p.expect(lexer.TokColonColonEqual)
	if err != nil {
		return nil, err
	}
	node.Assign = assignTok

	// OID
	oid, err := p.parseOidValue()
	if err != nil {
		return nil, err
	}
	node.Oid = oid

	node.Span = types.NewSpan(start, p.lastEnd)
	return node, nil
}

// parseNotificationType parses a NOTIFICATION-TYPE macro invocation.
func (p *Parser) parseNotificationType() (cst.DefinitionNode, *types.SpanDiagnostic) {
	start := p.currentSpan().Start
	node := &cst.NotificationTypeNode{}

	nameTok := p.advance()
	node.Name = nameTok
	name := p.text(nameTok.Span)
	p.validateIdentifier(name, nameTok.Span)
	p.validateValueReference(name, nameTok.Span)

	kwTok, err := p.expect(lexer.TokKwNotificationType)
	if err != nil {
		return nil, err
	}
	node.Keyword = kwTok

	// OBJECTS (optional)
	if p.check(lexer.TokKwObjects) {
		objects, err := p.parseObjectsClause()
		if err != nil {
			return nil, err
		}
		node.Objects = objects
	}

	// STATUS
	status, err := p.parseStatusClause()
	if err != nil {
		return nil, err
	}
	node.Status = status

	// DESCRIPTION
	desc, err := p.parseDescriptionClause()
	if err != nil {
		return nil, err
	}
	node.Description = desc

	// REFERENCE (optional)
	if p.check(lexer.TokKwReference) {
		ref, err := p.parseReferenceClause()
		if err != nil {
			return nil, err
		}
		node.Reference = ref
	}

	// ::=
	assignTok, err := p.expect(lexer.TokColonColonEqual)
	if err != nil {
		return nil, err
	}
	node.Assign = assignTok

	// OID
	oid, err := p.parseOidValue()
	if err != nil {
		return nil, err
	}
	node.Oid = oid

	node.Span = types.NewSpan(start, p.lastEnd)
	return node, nil
}

// parseTrapType parses a TRAP-TYPE macro invocation (SMIv1).
func (p *Parser) parseTrapType() (cst.DefinitionNode, *types.SpanDiagnostic) {
	start := p.currentSpan().Start
	node := &cst.TrapTypeNode{}

	nameTok := p.advance()
	node.Name = nameTok
	name := p.text(nameTok.Span)
	p.validateIdentifier(name, nameTok.Span)
	p.validateValueReference(name, nameTok.Span)

	kwTok, err := p.expect(lexer.TokKwTrapType)
	if err != nil {
		return nil, err
	}
	node.Keyword = kwTok

	// ENTERPRISE
	enterprise, err := p.parseEnterpriseClause()
	if err != nil {
		return nil, err
	}
	node.Enterprise = enterprise

	// VARIABLES (optional)
	if p.check(lexer.TokKwVariables) {
		vars, err := p.parseVariablesClause()
		if err != nil {
			return nil, err
		}
		node.Variables = vars
	}

	// DESCRIPTION (optional)
	if p.check(lexer.TokKwDescription) {
		desc, err := p.parseDescriptionClause()
		if err != nil {
			return nil, err
		}
		node.Description = desc
	}

	// REFERENCE (optional)
	if p.check(lexer.TokKwReference) {
		ref, err := p.parseReferenceClause()
		if err != nil {
			return nil, err
		}
		node.Reference = ref
	}

	// ::=
	assignTok, err := p.expect(lexer.TokColonColonEqual)
	if err != nil {
		return nil, err
	}
	node.Assign = assignTok

	// Trap number
	numTok, err := p.expect(lexer.TokNumber)
	if err != nil {
		return nil, err
	}
	node.Value = numTok

	node.Span = types.NewSpan(start, p.lastEnd)
	return node, nil
}

// parseTextualConvention parses a TEXTUAL-CONVENTION definition.
// If withAssign is true, parses: Name ::= TEXTUAL-CONVENTION ...
// Otherwise parses: Name TEXTUAL-CONVENTION ...
func (p *Parser) parseTextualConvention(withAssign bool) (cst.DefinitionNode, *types.SpanDiagnostic) {
	start := p.currentSpan().Start
	node := &cst.TextualConventionNode{}

	nameTok := p.advance()
	node.Name = nameTok
	name := p.text(nameTok.Span)
	p.validateIdentifier(name, nameTok.Span)

	if withAssign {
		assignTok, err := p.expect(lexer.TokColonColonEqual)
		if err != nil {
			return nil, err
		}
		node.Assign = assignTok
	}

	kwTok, err := p.expect(lexer.TokKwTextualConvention)
	if err != nil {
		return nil, err
	}
	node.Keyword = kwTok

	// DISPLAY-HINT (optional)
	if p.check(lexer.TokKwDisplayHint) {
		dh, err := p.parseDisplayHintClause()
		if err != nil {
			return nil, err
		}
		node.DisplayHint = dh
	}

	// STATUS
	status, err := p.parseStatusClause()
	if err != nil {
		return nil, err
	}
	node.Status = status

	// DESCRIPTION
	desc, err := p.parseDescriptionClause()
	if err != nil {
		return nil, err
	}
	node.Description = desc

	// REFERENCE (optional)
	if p.check(lexer.TokKwReference) {
		ref, err := p.parseReferenceClause()
		if err != nil {
			return nil, err
		}
		node.Reference = ref
	}

	// SYNTAX
	syntax, err := p.parseSyntaxClause()
	if err != nil {
		return nil, err
	}
	node.Syntax = syntax

	node.Span = types.NewSpan(start, p.lastEnd)
	return node, nil
}

// parseTypeAssignment parses: TypeName ::= TypeSyntax
func (p *Parser) parseTypeAssignment() (cst.DefinitionNode, *types.SpanDiagnostic) {
	start := p.currentSpan().Start
	node := &cst.TypeAssignmentNode{}

	nameTok := p.advance()
	node.Name = nameTok
	name := p.text(nameTok.Span)
	p.validateIdentifier(name, nameTok.Span)

	assignTok, err := p.expect(lexer.TokColonColonEqual)
	if err != nil {
		return nil, err
	}
	node.Assign = assignTok

	syntax, err := p.parseTypeSyntax()
	if err != nil {
		return nil, err
	}
	node.Syntax = syntax

	node.Span = types.NewSpan(start, p.lastEnd)
	return node, nil
}

// parseObjectGroup parses an OBJECT-GROUP macro invocation.
func (p *Parser) parseObjectGroup() (cst.DefinitionNode, *types.SpanDiagnostic) {
	start := p.currentSpan().Start
	node := &cst.ObjectGroupNode{}

	nameTok := p.advance()
	node.Name = nameTok
	name := p.text(nameTok.Span)
	p.validateIdentifier(name, nameTok.Span)
	p.validateValueReference(name, nameTok.Span)

	kwTok, err := p.expect(lexer.TokKwObjectGroup)
	if err != nil {
		return nil, err
	}
	node.Keyword = kwTok

	// OBJECTS
	objects, err := p.parseObjectsClause()
	if err != nil {
		return nil, err
	}
	node.Objects = objects

	// STATUS
	status, err := p.parseStatusClause()
	if err != nil {
		return nil, err
	}
	node.Status = status

	// DESCRIPTION
	desc, err := p.parseDescriptionClause()
	if err != nil {
		return nil, err
	}
	node.Description = desc

	// REFERENCE (optional)
	if p.check(lexer.TokKwReference) {
		ref, err := p.parseReferenceClause()
		if err != nil {
			return nil, err
		}
		node.Reference = ref
	}

	// ::=
	assignTok, err := p.expect(lexer.TokColonColonEqual)
	if err != nil {
		return nil, err
	}
	node.Assign = assignTok

	// OID
	oid, err := p.parseOidValue()
	if err != nil {
		return nil, err
	}
	node.Oid = oid

	node.Span = types.NewSpan(start, p.lastEnd)
	return node, nil
}

// parseNotificationGroup parses a NOTIFICATION-GROUP macro invocation.
func (p *Parser) parseNotificationGroup() (cst.DefinitionNode, *types.SpanDiagnostic) {
	start := p.currentSpan().Start
	node := &cst.NotificationGroupNode{}

	nameTok := p.advance()
	node.Name = nameTok
	name := p.text(nameTok.Span)
	p.validateIdentifier(name, nameTok.Span)
	p.validateValueReference(name, nameTok.Span)

	kwTok, err := p.expect(lexer.TokKwNotificationGroup)
	if err != nil {
		return nil, err
	}
	node.Keyword = kwTok

	// NOTIFICATIONS
	notifications, err := p.parseNotificationsClause()
	if err != nil {
		return nil, err
	}
	node.Notifications = notifications

	// STATUS
	status, err := p.parseStatusClause()
	if err != nil {
		return nil, err
	}
	node.Status = status

	// DESCRIPTION
	desc, err := p.parseDescriptionClause()
	if err != nil {
		return nil, err
	}
	node.Description = desc

	// REFERENCE (optional)
	if p.check(lexer.TokKwReference) {
		ref, err := p.parseReferenceClause()
		if err != nil {
			return nil, err
		}
		node.Reference = ref
	}

	// ::=
	assignTok, err := p.expect(lexer.TokColonColonEqual)
	if err != nil {
		return nil, err
	}
	node.Assign = assignTok

	// OID
	oid, err := p.parseOidValue()
	if err != nil {
		return nil, err
	}
	node.Oid = oid

	node.Span = types.NewSpan(start, p.lastEnd)
	return node, nil
}

// parseModuleCompliance parses a MODULE-COMPLIANCE macro invocation.
func (p *Parser) parseModuleCompliance() (cst.DefinitionNode, *types.SpanDiagnostic) {
	start := p.currentSpan().Start
	node := &cst.ModuleComplianceNode{}

	nameTok := p.advance()
	node.Name = nameTok
	name := p.text(nameTok.Span)
	p.validateIdentifier(name, nameTok.Span)
	p.validateValueReference(name, nameTok.Span)

	kwTok, err := p.expect(lexer.TokKwModuleCompliance)
	if err != nil {
		return nil, err
	}
	node.Keyword = kwTok

	// STATUS
	status, err := p.parseStatusClause()
	if err != nil {
		return nil, err
	}
	node.Status = status

	// DESCRIPTION
	desc, err := p.parseDescriptionClause()
	if err != nil {
		return nil, err
	}
	node.Description = desc

	// REFERENCE (optional)
	if p.check(lexer.TokKwReference) {
		ref, err := p.parseReferenceClause()
		if err != nil {
			return nil, err
		}
		node.Reference = ref
	}

	// MODULE clauses
	for p.check(lexer.TokKwModule) {
		mod, err := p.parseComplianceModule()
		if err != nil {
			return nil, err
		}
		node.Modules = append(node.Modules, *mod)
	}

	// ::=
	assignTok, err := p.expect(lexer.TokColonColonEqual)
	if err != nil {
		return nil, err
	}
	node.Assign = assignTok

	// OID
	oid, err := p.parseOidValue()
	if err != nil {
		return nil, err
	}
	node.Oid = oid

	node.Span = types.NewSpan(start, p.lastEnd)
	return node, nil
}

// parseComplianceModule parses a MODULE clause within MODULE-COMPLIANCE.
func (p *Parser) parseComplianceModule() (*cst.ComplianceModuleNode, *types.SpanDiagnostic) {
	start := p.currentSpan().Start
	node := &cst.ComplianceModuleNode{}

	kwTok, err := p.expect(lexer.TokKwModule)
	if err != nil {
		return nil, err
	}
	node.Keyword = kwTok

	// Optional module name
	if p.check(lexer.TokUppercaseIdent) {
		node.ModuleName = p.advance()
	}

	// Optional module OID.
	if p.check(lexer.TokLBrace) {
		oid, err := p.parseOidValue()
		if err != nil {
			return nil, err
		}
		node.Oid = oid
	}

	// MANDATORY-GROUPS (optional)
	if p.check(lexer.TokKwMandatoryGroups) {
		mg, err := p.parseMandatoryGroups()
		if err != nil {
			return nil, err
		}
		node.MandatoryGroups = mg
	}

	// GROUP and OBJECT refinements
	for p.check(lexer.TokKwGroup) || p.check(lexer.TokKwObject) {
		if p.check(lexer.TokKwGroup) {
			group, err := p.parseComplianceGroup()
			if err != nil {
				return nil, err
			}
			node.Groups = append(node.Groups, *group)
		} else {
			obj, err := p.parseComplianceObject()
			if err != nil {
				return nil, err
			}
			node.Objects = append(node.Objects, *obj)
		}
	}

	node.Span = types.NewSpan(start, p.lastEnd)
	return node, nil
}

// parseMandatoryGroups parses: MANDATORY-GROUPS { name, name, ... }
func (p *Parser) parseMandatoryGroups() (*cst.MandatoryGroupsNode, *types.SpanDiagnostic) {
	start := p.currentSpan().Start
	node := &cst.MandatoryGroupsNode{}

	kwTok, err := p.expect(lexer.TokKwMandatoryGroups)
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

// parseComplianceGroup parses: GROUP name DESCRIPTION "..."
func (p *Parser) parseComplianceGroup() (*cst.ComplianceGroupNode, *types.SpanDiagnostic) {
	start := p.currentSpan().Start
	node := &cst.ComplianceGroupNode{}

	kwTok, err := p.expect(lexer.TokKwGroup)
	if err != nil {
		return nil, err
	}
	node.Keyword = kwTok

	nameTok, err := p.expectIdentifier()
	if err != nil {
		return nil, err
	}
	node.Name = nameTok

	desc, err := p.parseDescriptionClause()
	if err != nil {
		return nil, err
	}
	node.Description = desc

	node.Span = types.NewSpan(start, p.lastEnd)
	return node, nil
}

// parseComplianceObject parses: OBJECT name [SYNTAX ...] [WRITE-SYNTAX ...] [MIN-ACCESS ...] DESCRIPTION "..."
func (p *Parser) parseComplianceObject() (*cst.ComplianceObjectNode, *types.SpanDiagnostic) {
	start := p.currentSpan().Start
	node := &cst.ComplianceObjectNode{}

	kwTok, err := p.expect(lexer.TokKwObject)
	if err != nil {
		return nil, err
	}
	node.Keyword = kwTok

	nameTok, err := p.expectIdentifier()
	if err != nil {
		return nil, err
	}
	node.Name = nameTok

	// Optional SYNTAX
	if p.check(lexer.TokKwSyntax) {
		syntax, err := p.parseSyntaxClause()
		if err != nil {
			return nil, err
		}
		node.Syntax = syntax
	}

	// Optional WRITE-SYNTAX
	if p.check(lexer.TokKwWriteSyntax) {
		ws, err := p.parseWriteSyntaxClause()
		if err != nil {
			return nil, err
		}
		node.WriteSyntax = ws
	}

	// Optional MIN-ACCESS
	if p.check(lexer.TokKwMinAccess) {
		access, err := p.parseAccessClause()
		if err != nil {
			return nil, err
		}
		node.Access = access
	}

	// Required DESCRIPTION
	desc, err := p.parseDescriptionClause()
	if err != nil {
		return nil, err
	}
	node.Description = desc

	node.Span = types.NewSpan(start, p.lastEnd)
	return node, nil
}

// parseAgentCapabilities parses an AGENT-CAPABILITIES macro invocation.
func (p *Parser) parseAgentCapabilities() (cst.DefinitionNode, *types.SpanDiagnostic) {
	start := p.currentSpan().Start
	node := &cst.AgentCapabilitiesNode{}

	nameTok := p.advance()
	node.Name = nameTok
	name := p.text(nameTok.Span)
	p.validateIdentifier(name, nameTok.Span)
	p.validateValueReference(name, nameTok.Span)

	kwTok, err := p.expect(lexer.TokKwAgentCapabilities)
	if err != nil {
		return nil, err
	}
	node.Keyword = kwTok

	// PRODUCT-RELEASE
	pr, err := p.parseProductReleaseClause()
	if err != nil {
		return nil, err
	}
	node.ProductRelease = pr

	// STATUS
	status, err := p.parseStatusClause()
	if err != nil {
		return nil, err
	}
	node.Status = status

	// DESCRIPTION
	desc, err := p.parseDescriptionClause()
	if err != nil {
		return nil, err
	}
	node.Description = desc

	// REFERENCE (optional)
	if p.check(lexer.TokKwReference) {
		ref, err := p.parseReferenceClause()
		if err != nil {
			return nil, err
		}
		node.Reference = ref
	}

	// SUPPORTS clauses
	for p.check(lexer.TokKwSupports) {
		mod, err := p.parseCapabilityModule()
		if err != nil {
			return nil, err
		}
		node.Modules = append(node.Modules, *mod)
	}

	// ::=
	assignTok, err := p.expect(lexer.TokColonColonEqual)
	if err != nil {
		return nil, err
	}
	node.Assign = assignTok

	// OID
	oid, err := p.parseOidValue()
	if err != nil {
		return nil, err
	}
	node.Oid = oid

	node.Span = types.NewSpan(start, p.lastEnd)
	return node, nil
}

// parseCapabilityModule parses a SUPPORTS clause within AGENT-CAPABILITIES.
func (p *Parser) parseCapabilityModule() (*cst.CapabilityModuleNode, *types.SpanDiagnostic) {
	start := p.currentSpan().Start
	node := &cst.CapabilityModuleNode{}

	kwTok, err := p.expect(lexer.TokKwSupports)
	if err != nil {
		return nil, err
	}
	node.Keyword = kwTok

	// Module name
	modNameTok, err := p.expectIdentifier()
	if err != nil {
		return nil, err
	}
	node.ModuleName = modNameTok

	// Optional module OID.
	if p.check(lexer.TokLBrace) {
		oid, err := p.parseOidValue()
		if err != nil {
			return nil, err
		}
		node.Oid = oid
	}

	// INCLUDES { groups }
	includes, err := p.parseIncludesClause()
	if err != nil {
		return nil, err
	}
	node.Includes = includes

	// VARIATION clauses
	for p.check(lexer.TokKwVariation) {
		v, err := p.parseCapabilityVariation()
		if err != nil {
			return nil, err
		}
		node.Variations = append(node.Variations, *v)
	}

	node.Span = types.NewSpan(start, p.lastEnd)
	return node, nil
}

// parseIncludesClause parses: INCLUDES { name, name, ... }
func (p *Parser) parseIncludesClause() (*cst.IncludesNode, *types.SpanDiagnostic) {
	start := p.currentSpan().Start
	node := &cst.IncludesNode{}

	kwTok, err := p.expect(lexer.TokKwIncludes)
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

// parseCapabilityVariation parses a VARIATION clause.
func (p *Parser) parseCapabilityVariation() (*cst.CapabilityVariationNode, *types.SpanDiagnostic) {
	start := p.currentSpan().Start
	node := &cst.CapabilityVariationNode{}

	kwTok, err := p.expect(lexer.TokKwVariation)
	if err != nil {
		return nil, err
	}
	node.Keyword = kwTok

	nameTok, err := p.expectIdentifier()
	if err != nil {
		return nil, err
	}
	node.Name = nameTok

	// Optional SYNTAX
	if p.check(lexer.TokKwSyntax) {
		syntax, err := p.parseSyntaxClause()
		if err != nil {
			return nil, err
		}
		node.Syntax = syntax
	}

	// Optional WRITE-SYNTAX
	if p.check(lexer.TokKwWriteSyntax) {
		ws, err := p.parseWriteSyntaxClause()
		if err != nil {
			return nil, err
		}
		node.WriteSyntax = ws
	}

	// Optional ACCESS
	if p.check(lexer.TokKwAccess) {
		access, err := p.parseAccessClause()
		if err != nil {
			return nil, err
		}
		node.Access = access
	}

	// Optional CREATION-REQUIRES
	if p.check(lexer.TokKwCreationRequires) {
		cr, err := p.parseCreationRequiresClause()
		if err != nil {
			return nil, err
		}
		node.CreationRequires = cr
	}

	// Optional DEFVAL
	if p.check(lexer.TokKwDefval) {
		defval, err := p.parseDefValClause()
		if err != nil {
			return nil, err
		}
		node.DefVal = defval
	}

	// Required DESCRIPTION
	desc, err := p.parseDescriptionClause()
	if err != nil {
		return nil, err
	}
	node.Description = desc

	node.Span = types.NewSpan(start, p.lastEnd)
	return node, nil
}

// parseCreationRequiresClause parses: CREATION-REQUIRES { name, name, ... }
func (p *Parser) parseCreationRequiresClause() (*cst.CreationRequiresNode, *types.SpanDiagnostic) {
	start := p.currentSpan().Start
	node := &cst.CreationRequiresNode{}

	kwTok, err := p.expect(lexer.TokKwCreationRequires)
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

// parseMacroDefinition parses a MACRO definition. The lexer skips macro bodies,
// so we see name MACRO then body tokens are consumed by the lexer until END.
func (p *Parser) parseMacroDefinition() (cst.DefinitionNode, *types.SpanDiagnostic) {
	start := p.currentSpan().Start
	node := &cst.MacroDefinitionNode{}

	nameTok := p.advance()
	node.Name = nameTok

	// The lexer enters stateInMacro when it sees MACRO keyword, and skips
	// body until END. So we won't see ::= BEGIN body in the token stream.
	// We'll see: MACRO keyword (which triggered state change), then END.
	macroTok, err := p.expect(lexer.TokKwMacro)
	if err != nil {
		return nil, err
	}
	node.Macro = macroTok

	// The lexer's macro-skipping mode means we may not see ::= BEGIN in the
	// token stream at all. The lexer skips everything until END.
	// We just look for END.
	for !p.check(lexer.TokKwEnd) && !p.isEOF() {
		node.Body = append(node.Body, p.advance())
	}

	if p.check(lexer.TokKwEnd) {
		node.End = p.advance()
	} else {
		diag := p.makeError("expected END for MACRO")
		return nil, &diag
	}

	node.Span = types.NewSpan(start, p.lastEnd)
	return node, nil
}
