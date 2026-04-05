package parser

import (
	"fmt"

	"github.com/golangsnmp/gomib/internal/lexer"
	"github.com/golangsnmp/gomib/internal/module"
	"github.com/golangsnmp/gomib/internal/types"
)

// parseModuleIdentity parses a MODULE-IDENTITY macro invocation.
func (p *Parser) parseModuleIdentity() (module.Definition, *types.SpanDiagnostic) {
	start := p.currentSpan().Start

	nameToken := p.advance()
	name, _ := p.makeIdentStrWithValidation(nameToken)
	p.validateValueReference(name, nameToken.Span)

	if _, err := p.expect(lexer.TokKwModuleIdentity); err != nil {
		return nil, err
	}

	// LAST-UPDATED
	if _, err := p.expect(lexer.TokKwLastUpdated); err != nil {
		return nil, err
	}
	lastUpdated, _, err := p.parseQuotedString()
	if err != nil {
		return nil, err
	}

	// ORGANIZATION
	if _, err := p.expect(lexer.TokKwOrganization); err != nil {
		return nil, err
	}
	organization, _, err := p.parseQuotedString()
	if err != nil {
		return nil, err
	}

	// CONTACT-INFO
	if _, err := p.expect(lexer.TokKwContactInfo); err != nil {
		return nil, err
	}
	contactInfo, _, err := p.parseQuotedString()
	if err != nil {
		return nil, err
	}

	// DESCRIPTION
	if _, err := p.expect(lexer.TokKwDescription); err != nil {
		return nil, err
	}
	description, _, err := p.parseQuotedString()
	if err != nil {
		return nil, err
	}

	// REVISION clauses (optional, multiple)
	var revisions []module.Revision
	for p.check(lexer.TokKwRevision) {
		revStart := p.currentSpan().Start
		p.advance()
		date, _, err := p.parseQuotedString()
		if err != nil {
			return nil, err
		}
		if _, err := p.expect(lexer.TokKwDescription); err != nil {
			return nil, err
		}
		revDescription, revDescSpan, err := p.parseQuotedString()
		if err != nil {
			return nil, err
		}
		span := types.NewSpan(revStart, revDescSpan.End)
		p.checkEmptyRequiredString(revDescription, span, name, "REVISION DESCRIPTION", types.DiagEmptyDescription)
		revisions = append(revisions, module.Revision{
			Date:        date,
			Description: revDescription,
			Span:        span,
		})
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
	p.checkEmptyRequiredString(description, span, name, "DESCRIPTION", types.DiagEmptyDescription)
	p.checkEmptyRequiredString(organization, span, name, "ORGANIZATION", types.DiagEmptyOrganization)
	p.checkEmptyRequiredString(contactInfo, span, name, "CONTACT-INFO", types.DiagEmptyContact)

	return &module.ModuleIdentity{
		DefBase:      module.DefBase{Name: name, Span: span},
		LastUpdated:  lastUpdated,
		Organization: organization,
		ContactInfo:  contactInfo,
		Description:  description,
		Revisions:    revisions,
		Oid:          oid,
	}, nil
}

// parseObjectIdentity parses an OBJECT-IDENTITY macro invocation.
func (p *Parser) parseObjectIdentity() (module.Definition, *types.SpanDiagnostic) {
	start := p.currentSpan().Start

	nameToken := p.advance()
	name, _ := p.makeIdentStrWithValidation(nameToken)
	p.validateValueReference(name, nameToken.Span)

	if _, err := p.expect(lexer.TokKwObjectIdentity); err != nil {
		return nil, err
	}

	// STATUS
	statusValue, _, err := p.parseStatusClause()
	if err != nil {
		return nil, err
	}

	// DESCRIPTION
	if _, err := p.expect(lexer.TokKwDescription); err != nil {
		return nil, err
	}
	description, _, err := p.parseQuotedString()
	if err != nil {
		return nil, err
	}

	// REFERENCE (optional)
	reference, _, referencePresent, err := p.parseOptionalReference()
	if err != nil {
		return nil, err
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
	p.checkEmptyRequiredString(description, span, name, "DESCRIPTION", types.DiagEmptyDescription)
	p.checkEmptyOptionalString(reference, referencePresent, span, name, "REFERENCE", types.DiagEmptyReference)

	return &module.ObjectIdentity{
		DefBase:     module.DefBase{Name: name, Span: span},
		Status:      statusValue,
		Description: description,
		Reference:   reference,
		Oid:         oid,
	}, nil
}

// parseNotificationType parses a NOTIFICATION-TYPE macro invocation.
func (p *Parser) parseNotificationType() (module.Definition, *types.SpanDiagnostic) {
	start := p.currentSpan().Start

	nameToken := p.advance()
	name, _ := p.makeIdentStrWithValidation(nameToken)
	p.validateValueReference(name, nameToken.Span)

	if _, err := p.expect(lexer.TokKwNotificationType); err != nil {
		return nil, err
	}

	// OBJECTS (optional)
	var objects []string
	if p.check(lexer.TokKwObjects) {
		p.advance()
		objs, err := p.parseBracedIdentifierList()
		if err != nil {
			return nil, err
		}
		objects = objs
	}

	// STATUS
	statusValue, _, err := p.parseStatusClause()
	if err != nil {
		return nil, err
	}

	// DESCRIPTION
	if _, err := p.expect(lexer.TokKwDescription); err != nil {
		return nil, err
	}
	description, _, err := p.parseQuotedString()
	if err != nil {
		return nil, err
	}

	// REFERENCE (optional)
	reference, _, referencePresent, err := p.parseOptionalReference()
	if err != nil {
		return nil, err
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
	p.checkEmptyRequiredString(description, span, name, "DESCRIPTION", types.DiagEmptyDescription)
	p.checkEmptyOptionalString(reference, referencePresent, span, name, "REFERENCE", types.DiagEmptyReference)

	return &module.Notification{
		DefBase:        module.DefBase{Name: name, Span: span},
		Objects:        objects,
		Status:         statusValue,
		Description:    description,
		HasDescription: true, // NOTIFICATION-TYPE DESCRIPTION is required
		Reference:      reference,
		Oid:            &oid,
	}, nil
}

// parseTrapType parses a TRAP-TYPE macro invocation (SMIv1).
func (p *Parser) parseTrapType() (module.Definition, *types.SpanDiagnostic) {
	start := p.currentSpan().Start

	nameToken := p.advance()
	name, _ := p.makeIdentStrWithValidation(nameToken)
	p.validateValueReference(name, nameToken.Span)

	if _, err := p.expect(lexer.TokKwTrapType); err != nil {
		return nil, err
	}

	// ENTERPRISE
	if _, err := p.expect(lexer.TokKwEnterprise); err != nil {
		return nil, err
	}
	enterpriseToken, err := p.expectIdentifier()
	if err != nil {
		return nil, err
	}
	enterprise := p.text(enterpriseToken.Span)

	// VARIABLES (optional)
	var variables []string
	if p.check(lexer.TokKwVariables) {
		p.advance()
		vars, err := p.parseBracedIdentifierList()
		if err != nil {
			return nil, err
		}
		variables = vars
	}

	// DESCRIPTION (optional)
	var description string
	hasDescription := false
	if p.check(lexer.TokKwDescription) {
		p.advance()
		hasDescription = true
		description, _, err = p.parseQuotedString()
		if err != nil {
			return nil, err
		}
	}

	// REFERENCE (optional)
	var reference string
	var referencePresent bool
	reference, _, referencePresent, err = p.parseOptionalReference()
	if err != nil {
		return nil, err
	}

	// ::= number
	if _, err := p.expect(lexer.TokColonColonEqual); err != nil {
		return nil, err
	}
	numToken, err := p.expect(lexer.TokNumber)
	if err != nil {
		return nil, err
	}
	trapNumber := p.convertU32(numToken.Span, "trap number")

	span := types.NewSpan(start, numToken.Span.End)

	// Empty string checks
	p.checkEmptyOptionalString(description, hasDescription, span, name, "DESCRIPTION", types.DiagEmptyDescription)
	p.checkEmptyOptionalString(reference, referencePresent, span, name, "REFERENCE", types.DiagEmptyReference)

	return &module.Notification{
		DefBase:        module.DefBase{Name: name, Span: span},
		Objects:        variables,
		Status:         types.StatusCurrent, // TRAP-TYPE has no STATUS clause
		Description:    description,
		HasDescription: hasDescription,
		Reference:      reference,
		TrapInfo: &module.TrapInfo{
			Enterprise: enterprise,
			TrapNumber: trapNumber,
		},
		// Oid is nil for TRAP-TYPE
	}, nil
}

// parseTextualConvention parses: Name TEXTUAL-CONVENTION ...
func (p *Parser) parseTextualConvention() (module.Definition, *types.SpanDiagnostic) {
	start := p.currentSpan().Start

	nameToken := p.advance()
	name, _ := p.makeIdentStrWithValidation(nameToken)

	if _, err := p.expect(lexer.TokKwTextualConvention); err != nil {
		return nil, err
	}

	return p.parseTextualConventionBody(name, start)
}

// parseTextualConventionWithAssignment parses the alternate form:
// Name ::= TEXTUAL-CONVENTION ...
func (p *Parser) parseTextualConventionWithAssignment() (module.Definition, *types.SpanDiagnostic) {
	start := p.currentSpan().Start

	nameToken := p.advance()
	name, _ := p.makeIdentStrWithValidation(nameToken)

	if _, err := p.expect(lexer.TokColonColonEqual); err != nil {
		return nil, err
	}
	if _, err := p.expect(lexer.TokKwTextualConvention); err != nil {
		return nil, err
	}

	return p.parseTextualConventionBody(name, start)
}

// parseTextualConventionBody parses the shared body of a TEXTUAL-CONVENTION
// (DISPLAY-HINT, STATUS, DESCRIPTION, REFERENCE, SYNTAX).
func (p *Parser) parseTextualConventionBody(name string, start types.ByteOffset) (module.Definition, *types.SpanDiagnostic) {
	// DISPLAY-HINT (optional)
	var displayHint string
	var displayHintSpan types.Span
	displayHintPresent := false
	if p.check(lexer.TokKwDisplayHint) {
		p.advance()
		displayHintPresent = true
		var err *types.SpanDiagnostic
		displayHint, displayHintSpan, err = p.parseQuotedString()
		if err != nil {
			return nil, err
		}
	}

	// STATUS
	statusValue, statusSpan, err := p.parseStatusClause()
	if err != nil {
		return nil, err
	}

	// DESCRIPTION
	if _, err := p.expect(lexer.TokKwDescription); err != nil {
		return nil, err
	}
	description, descriptionSpan, err := p.parseQuotedString()
	if err != nil {
		return nil, err
	}

	// REFERENCE (optional)
	reference, referenceSpan, referencePresent, err := p.parseOptionalReference()
	if err != nil {
		return nil, err
	}

	// SYNTAX
	if _, err := p.expect(lexer.TokKwSyntax); err != nil {
		return nil, err
	}
	syntaxStart := p.currentSpan().Start
	syntax, err := p.parseTypeSyntax()
	if err != nil {
		return nil, err
	}
	syntaxSpan := types.NewSpan(syntaxStart, syntax.SyntaxSpan().End)

	span := types.NewSpan(start, syntaxSpan.End)

	// Empty string checks
	p.checkEmptyRequiredString(description, span, name, "DESCRIPTION", types.DiagEmptyDescription)
	p.checkEmptyOptionalString(reference, referencePresent, span, name, "REFERENCE", types.DiagEmptyReference)
	p.checkEmptyOptionalString(displayHint, displayHintPresent, span, name, "DISPLAY-HINT", types.DiagEmptyFormat)

	return &module.TypeDef{
		DefBase:             module.DefBase{Name: name, Span: span},
		Syntax:              syntax,
		DisplayHint:         displayHint,
		Status:              statusValue,
		Description:         description,
		Reference:           reference,
		IsTextualConvention: true,
		Spans: module.TypeDefSpans{
			Syntax:      syntaxSpan,
			Status:      statusSpan,
			Description: descriptionSpan,
			Reference:   referenceSpan,
			DisplayHint: displayHintSpan,
		},
	}, nil
}

// parseTypeAssignment parses: TypeName ::= TypeSyntax
func (p *Parser) parseTypeAssignment() (module.Definition, *types.SpanDiagnostic) {
	start := p.currentSpan().Start

	nameToken := p.advance()
	name, _ := p.makeIdentStrWithValidation(nameToken)

	if _, err := p.expect(lexer.TokColonColonEqual); err != nil {
		return nil, err
	}

	syntax, err := p.parseTypeSyntax()
	if err != nil {
		return nil, err
	}
	span := types.NewSpan(start, syntax.SyntaxSpan().End)

	return &module.TypeDef{
		DefBase: module.DefBase{Name: name, Span: span},
		Syntax:  syntax,
		Status:  types.StatusCurrent, // Default status for plain type assignments
		Spans:   module.TypeDefSpans{Syntax: syntax.SyntaxSpan()},
	}, nil
}

// parseObjectGroup parses an OBJECT-GROUP macro invocation.
func (p *Parser) parseObjectGroup() (module.Definition, *types.SpanDiagnostic) {
	start := p.currentSpan().Start

	nameToken := p.advance()
	name, _ := p.makeIdentStrWithValidation(nameToken)
	p.validateValueReference(name, nameToken.Span)

	if _, err := p.expect(lexer.TokKwObjectGroup); err != nil {
		return nil, err
	}

	if _, err := p.expect(lexer.TokKwObjects); err != nil {
		return nil, err
	}
	members, err := p.parseBracedIdentifierList()
	if err != nil {
		return nil, err
	}

	statusValue, _, err := p.parseStatusClause()
	if err != nil {
		return nil, err
	}

	if _, err := p.expect(lexer.TokKwDescription); err != nil {
		return nil, err
	}
	description, _, err := p.parseQuotedString()
	if err != nil {
		return nil, err
	}

	reference, _, referencePresent, err := p.parseOptionalReference()
	if err != nil {
		return nil, err
	}

	if _, err := p.expect(lexer.TokColonColonEqual); err != nil {
		return nil, err
	}
	oid, err := p.parseOidAssignment()
	if err != nil {
		return nil, err
	}

	span := types.NewSpan(start, oid.Span.End)

	// Empty string checks
	p.checkEmptyRequiredString(description, span, name, "DESCRIPTION", types.DiagEmptyDescription)
	p.checkEmptyOptionalString(reference, referencePresent, span, name, "REFERENCE", types.DiagEmptyReference)

	return &module.ObjectGroup{
		DefBase:     module.DefBase{Name: name, Span: span},
		Objects:     members,
		Status:      statusValue,
		Description: description,
		Reference:   reference,
		Oid:         oid,
	}, nil
}

// parseNotificationGroup parses a NOTIFICATION-GROUP macro invocation.
func (p *Parser) parseNotificationGroup() (module.Definition, *types.SpanDiagnostic) {
	start := p.currentSpan().Start

	nameToken := p.advance()
	name, _ := p.makeIdentStrWithValidation(nameToken)
	p.validateValueReference(name, nameToken.Span)

	if _, err := p.expect(lexer.TokKwNotificationGroup); err != nil {
		return nil, err
	}

	if _, err := p.expect(lexer.TokKwNotifications); err != nil {
		return nil, err
	}
	members, err := p.parseBracedIdentifierList()
	if err != nil {
		return nil, err
	}

	statusValue, _, err := p.parseStatusClause()
	if err != nil {
		return nil, err
	}

	if _, err := p.expect(lexer.TokKwDescription); err != nil {
		return nil, err
	}
	description, _, err := p.parseQuotedString()
	if err != nil {
		return nil, err
	}

	reference, _, referencePresent, err := p.parseOptionalReference()
	if err != nil {
		return nil, err
	}

	if _, err := p.expect(lexer.TokColonColonEqual); err != nil {
		return nil, err
	}
	oid, err := p.parseOidAssignment()
	if err != nil {
		return nil, err
	}

	span := types.NewSpan(start, oid.Span.End)

	// Empty string checks
	p.checkEmptyRequiredString(description, span, name, "DESCRIPTION", types.DiagEmptyDescription)
	p.checkEmptyOptionalString(reference, referencePresent, span, name, "REFERENCE", types.DiagEmptyReference)

	return &module.NotificationGroup{
		DefBase:       module.DefBase{Name: name, Span: span},
		Notifications: members,
		Status:        statusValue,
		Description:   description,
		Reference:     reference,
		Oid:           oid,
	}, nil
}

// parseModuleCompliance parses a MODULE-COMPLIANCE macro invocation.
func (p *Parser) parseModuleCompliance() (module.Definition, *types.SpanDiagnostic) {
	start := p.currentSpan().Start

	nameToken := p.advance()
	name, _ := p.makeIdentStrWithValidation(nameToken)
	p.validateValueReference(name, nameToken.Span)

	if _, err := p.expect(lexer.TokKwModuleCompliance); err != nil {
		return nil, err
	}

	// STATUS
	statusValue, _, err := p.parseStatusClause()
	if err != nil {
		return nil, err
	}

	// DESCRIPTION
	if _, err := p.expect(lexer.TokKwDescription); err != nil {
		return nil, err
	}
	description, _, err := p.parseQuotedString()
	if err != nil {
		return nil, err
	}

	// REFERENCE (optional)
	reference, _, referencePresent, err := p.parseOptionalReference()
	if err != nil {
		return nil, err
	}

	// Parse MODULE clauses
	var modules []module.ComplianceModule
	for p.check(lexer.TokKwModule) {
		mod, err := p.parseComplianceModule()
		if err != nil {
			return nil, err
		}
		modules = append(modules, mod)
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
	p.checkEmptyRequiredString(description, span, name, "DESCRIPTION", types.DiagEmptyDescription)
	p.checkEmptyOptionalString(reference, referencePresent, span, name, "REFERENCE", types.DiagEmptyReference)

	return &module.ModuleCompliance{
		DefBase:     module.DefBase{Name: name, Span: span},
		Status:      statusValue,
		Description: description,
		Reference:   reference,
		Modules:     modules,
		Oid:         oid,
	}, nil
}

func (p *Parser) parseComplianceModule() (module.ComplianceModule, *types.SpanDiagnostic) {
	start := p.currentSpan().Start
	if _, err := p.expect(lexer.TokKwModule); err != nil {
		return module.ComplianceModule{}, err
	}

	// Optional module name
	var moduleName string
	if p.check(lexer.TokUppercaseIdent) {
		nameToken := p.advance()
		moduleName = p.text(nameToken.Span)
	}

	// Optional module OID (intentionally not preserved)
	if p.check(lexer.TokLBrace) {
		_, err := p.parseOidAssignment()
		if err != nil {
			return module.ComplianceModule{}, err
		}
	}

	// MANDATORY-GROUPS (optional)
	var mandatoryGroups []string
	if p.check(lexer.TokKwMandatoryGroups) {
		groups, err := p.parseMandatoryGroups()
		if err != nil {
			return module.ComplianceModule{}, err
		}
		mandatoryGroups = groups
	}

	// GROUP and OBJECT refinements
	var groups []module.ComplianceGroup
	var objects []module.ComplianceObject
	for p.check(lexer.TokKwGroup) || p.check(lexer.TokKwObject) {
		if p.check(lexer.TokKwGroup) {
			group, err := p.parseComplianceGroup()
			if err != nil {
				return module.ComplianceModule{}, err
			}
			groups = append(groups, *group)
		} else {
			obj, err := p.parseComplianceObject()
			if err != nil {
				return module.ComplianceModule{}, err
			}
			objects = append(objects, *obj)
		}
	}

	return module.ComplianceModule{
		ModuleName:      moduleName,
		MandatoryGroups: mandatoryGroups,
		Groups:          groups,
		Objects:         objects,
		Span:            types.NewSpan(start, p.lastEnd),
	}, nil
}

func (p *Parser) parseMandatoryGroups() ([]string, *types.SpanDiagnostic) {
	if _, err := p.expect(lexer.TokKwMandatoryGroups); err != nil {
		return nil, err
	}
	return p.parseBracedIdentifierList()
}

func (p *Parser) parseComplianceGroup() (*module.ComplianceGroup, *types.SpanDiagnostic) {
	start := p.currentSpan().Start
	if _, err := p.expect(lexer.TokKwGroup); err != nil {
		return nil, err
	}
	groupName, err := p.parseIdentifierAsString()
	if err != nil {
		return nil, err
	}
	if _, err := p.expect(lexer.TokKwDescription); err != nil {
		return nil, err
	}
	description, descSpan, err := p.parseQuotedString()
	if err != nil {
		return nil, err
	}
	end := descSpan.End
	return &module.ComplianceGroup{
		Group:       groupName,
		Description: description,
		Span:        types.NewSpan(start, end),
	}, nil
}

// parseOptionalSyntaxClauses parses optional SYNTAX and WRITE-SYNTAX clauses,
// shared by MODULE-COMPLIANCE objects and AGENT-CAPABILITIES variations.
func (p *Parser) parseOptionalSyntaxClauses() (syntax, writeSyntax module.TypeSyntax, err *types.SpanDiagnostic) {
	if p.check(lexer.TokKwSyntax) {
		p.advance()
		s, e := p.parseTypeSyntax()
		if e != nil {
			return nil, nil, e
		}
		syntax = s
	}
	if p.check(lexer.TokKwWriteSyntax) {
		p.advance()
		s, e := p.parseTypeSyntax()
		if e != nil {
			return nil, nil, e
		}
		writeSyntax = s
	}
	return syntax, writeSyntax, nil
}

func (p *Parser) parseComplianceObject() (*module.ComplianceObject, *types.SpanDiagnostic) {
	start := p.currentSpan().Start
	if _, err := p.expect(lexer.TokKwObject); err != nil {
		return nil, err
	}
	objectName, err := p.parseIdentifierAsString()
	if err != nil {
		return nil, err
	}

	syntax, writeSyntax, err := p.parseOptionalSyntaxClauses()
	if err != nil {
		return nil, err
	}

	// Optional MIN-ACCESS
	var minAccess *types.Access
	if p.check(lexer.TokKwMinAccess) {
		accessValue, _, _, err := p.parseAccessClause()
		if err != nil {
			return nil, err
		}
		minAccess = &accessValue
	}

	// Required DESCRIPTION
	if _, err := p.expect(lexer.TokKwDescription); err != nil {
		return nil, err
	}
	description, descSpan, err := p.parseQuotedString()
	if err != nil {
		return nil, err
	}

	end := descSpan.End
	return &module.ComplianceObject{
		Object:      objectName,
		Syntax:      syntax,
		WriteSyntax: writeSyntax,
		MinAccess:   minAccess,
		Description: description,
		Span:        types.NewSpan(start, end),
	}, nil
}

// parseAgentCapabilities parses an AGENT-CAPABILITIES macro invocation.
func (p *Parser) parseAgentCapabilities() (module.Definition, *types.SpanDiagnostic) {
	start := p.currentSpan().Start

	nameToken := p.advance()
	name, _ := p.makeIdentStrWithValidation(nameToken)
	p.validateValueReference(name, nameToken.Span)

	if _, err := p.expect(lexer.TokKwAgentCapabilities); err != nil {
		return nil, err
	}

	// PRODUCT-RELEASE
	if _, err := p.expect(lexer.TokKwProductRelease); err != nil {
		return nil, err
	}
	productRelease, _, err := p.parseQuotedString()
	if err != nil {
		return nil, err
	}

	// STATUS
	statusValue, _, err := p.parseStatusClause()
	if err != nil {
		return nil, err
	}

	// DESCRIPTION
	if _, err := p.expect(lexer.TokKwDescription); err != nil {
		return nil, err
	}
	description, _, err := p.parseQuotedString()
	if err != nil {
		return nil, err
	}

	// REFERENCE (optional)
	reference, _, referencePresent, err := p.parseOptionalReference()
	if err != nil {
		return nil, err
	}

	// Parse SUPPORTS clauses
	var supports []module.SupportsModule
	for p.check(lexer.TokKwSupports) {
		sup, err := p.parseSupportsModule()
		if err != nil {
			return nil, err
		}
		supports = append(supports, sup)
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
	p.checkEmptyRequiredString(description, span, name, "DESCRIPTION", types.DiagEmptyDescription)
	p.checkEmptyOptionalString(reference, referencePresent, span, name, "REFERENCE", types.DiagEmptyReference)

	return &module.AgentCapabilities{
		DefBase:        module.DefBase{Name: name, Span: span},
		ProductRelease: productRelease,
		Status:         statusValue,
		Description:    description,
		Reference:      reference,
		Supports:       supports,
		Oid:            oid,
	}, nil
}

func (p *Parser) parseSupportsModule() (module.SupportsModule, *types.SpanDiagnostic) {
	start := p.currentSpan().Start
	if _, err := p.expect(lexer.TokKwSupports); err != nil {
		return module.SupportsModule{}, err
	}

	// Module name
	moduleName, err := p.parseIdentifierAsString()
	if err != nil {
		return module.SupportsModule{}, err
	}

	// Optional module OID (intentionally not preserved)
	if p.check(lexer.TokLBrace) {
		_, err := p.parseOidAssignment()
		if err != nil {
			return module.SupportsModule{}, err
		}
	}

	// INCLUDES { groups }
	if _, err := p.expect(lexer.TokKwIncludes); err != nil {
		return module.SupportsModule{}, err
	}
	includes, err := p.parseBracedIdentifierList()
	if err != nil {
		return module.SupportsModule{}, err
	}

	// VARIATION clauses
	var variations []module.Variation
	for p.check(lexer.TokKwVariation) {
		v, err := p.parseVariationClause()
		if err != nil {
			return module.SupportsModule{}, err
		}
		variations = append(variations, *v)
	}

	return module.SupportsModule{
		ModuleName: moduleName,
		Includes:   includes,
		Variations: variations,
		Span:       types.NewSpan(start, p.lastEnd),
	}, nil
}

func (p *Parser) parseVariationClause() (*module.Variation, *types.SpanDiagnostic) {
	start := p.currentSpan().Start
	if _, err := p.expect(lexer.TokKwVariation); err != nil {
		return nil, err
	}

	// Object or notification name
	varName, err := p.parseIdentifierAsString()
	if err != nil {
		return nil, err
	}

	syntax, writeSyntax, err := p.parseOptionalSyntaxClauses()
	if err != nil {
		return nil, err
	}

	// Optional ACCESS
	var access *types.Access
	if p.check(lexer.TokKwAccess) {
		accessValue, _, _, err := p.parseAccessClause()
		if err != nil {
			return nil, err
		}
		access = &accessValue
	}

	// Optional CREATION-REQUIRES
	var creationRequires []string
	if p.check(lexer.TokKwCreationRequires) {
		p.advance()
		objs, err := p.parseBracedIdentifierList()
		if err != nil {
			return nil, err
		}
		creationRequires = objs
	}

	// Optional DEFVAL
	var defval module.DefVal
	if p.check(lexer.TokKwDefval) {
		dv, _, err := p.parseDefValClause()
		if err != nil {
			return nil, err
		}
		defval = dv
	}

	// Required DESCRIPTION
	if _, err := p.expect(lexer.TokKwDescription); err != nil {
		return nil, err
	}
	description, descSpan, err := p.parseQuotedString()
	if err != nil {
		return nil, err
	}

	end := descSpan.End

	return &module.Variation{
		Name:             varName,
		Syntax:           syntax,
		WriteSyntax:      writeSyntax,
		Access:           access,
		CreationRequires: creationRequires,
		DefVal:           defval,
		Description:      description,
		Span:             types.NewSpan(start, end),
	}, nil
}

// parseMacroDefinition parses a MACRO definition header and skips
// to END. Returns nil (MACRO definitions are not semantic definitions).
func (p *Parser) parseMacroDefinition() (module.Definition, *types.SpanDiagnostic) {
	nameToken := p.advance()
	name := p.text(nameToken.Span)

	if _, err := p.expect(lexer.TokKwMacro); err != nil {
		return nil, err
	}

	for !p.check(lexer.TokKwEnd) && !p.isEOF() {
		p.advance()
	}

	if p.check(lexer.TokKwEnd) {
		p.advance()
	} else {
		diag := p.makeError("expected END for MACRO")
		return nil, &diag
	}

	if !module.IsBaseModule(p.moduleName) {
		span := types.NewSpan(nameToken.Span.Start, p.lastEnd)
		p.EmitDiagnostic(types.DiagMacroNotAllowed, span,
			fmt.Sprintf("MACRO definition %q not allowed outside base modules", name))
	}

	return nil, nil
}
