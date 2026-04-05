package lower

import (
	"github.com/golangsnmp/gomib/internal/cst"
	"github.com/golangsnmp/gomib/internal/module"
	"github.com/golangsnmp/gomib/internal/types"
)

func (l *lowerer) lowerObjectType(n *cst.ObjectTypeNode) *module.ObjectType {
	name := l.text(n.Name)

	syntax := l.lowerTypeSyntax(n.Syntax.Syntax)
	syntaxSpan := types.NewSpan(n.Syntax.Syntax.TypeSyntaxSpan().Start, syntax.SyntaxSpan().End)

	var units string
	var unitsSpan types.Span
	if n.Units != nil {
		units = l.stripQuotes(n.Units.Value)
		unitsSpan = n.Units.Value.Span
		l.checkEmptyString(n.Units.Value, name, "UNITS", types.DiagEmptyUnits)
	}

	accessValue, accessKeyword, accessSpan := l.lowerAccess(n.Access)

	var statusValue types.Status
	var statusSpan types.Span
	if n.Status != nil {
		statusValue, statusSpan = l.lowerStatus(n.Status)
	} else {
		statusValue = types.StatusCurrent
	}

	var description string
	var descriptionSpan types.Span
	hasDescription := false
	if n.Description != nil {
		hasDescription = true
		description = l.stripQuotes(n.Description.Value)
		descriptionSpan = n.Description.Value.Span
		l.checkEmptyString(n.Description.Value, name, "DESCRIPTION", types.DiagEmptyDescription)
	}

	var reference string
	var referenceSpan types.Span
	if n.Reference != nil {
		reference = l.stripQuotes(n.Reference.Value)
		referenceSpan = n.Reference.Value.Span
		l.checkEmptyString(n.Reference.Value, name, "REFERENCE", types.DiagEmptyReference)
	}

	indexItems, augmentsTarget, augmentsSpan, indexSpan := l.lowerIndexOrAugments(n.Index, n.Augments)

	var defval module.DefVal
	var defvalSpan types.Span
	if n.DefVal != nil {
		defval = l.lowerDefVal(n.DefVal.Content)
		defvalSpan = n.DefVal.Span
	}

	oid := l.lowerOidValue(n.Oid)

	span := types.NewSpan(n.Span.Start, oid.Span.End)

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
	}
}

func (l *lowerer) lowerModuleIdentity(n *cst.ModuleIdentityNode) *module.ModuleIdentity {
	name := l.text(n.Name)

	lastUpdated := l.stripQuotes(n.LastUpdated.Value)
	organization := l.stripQuotes(n.Organization.Value)
	contactInfo := l.stripQuotes(n.ContactInfo.Value)
	description := l.stripQuotes(n.Description.Value)

	l.checkEmptyString(n.Description.Value, name, "DESCRIPTION", types.DiagEmptyDescription)
	l.checkEmptyString(n.Organization.Value, name, "ORGANIZATION", types.DiagEmptyOrganization)
	l.checkEmptyString(n.ContactInfo.Value, name, "CONTACT-INFO", types.DiagEmptyContact)

	var revisions []module.Revision
	for i := range n.Revisions {
		rev := &n.Revisions[i]
		date := l.stripQuotes(rev.Date)
		revDesc := ""
		if rev.Description != nil {
			revDesc = l.stripQuotes(rev.Description.Value)
			l.checkEmptyString(rev.Description.Value, name, "DESCRIPTION", types.DiagEmptyDescription)
		}
		revisions = append(revisions, module.Revision{
			Date:        date,
			Description: revDesc,
			Span:        rev.Span,
		})
	}

	oid := l.lowerOidValue(n.Oid)
	span := types.NewSpan(n.Span.Start, oid.Span.End)

	return &module.ModuleIdentity{
		DefBase:      module.DefBase{Name: name, Span: span},
		LastUpdated:  lastUpdated,
		Organization: organization,
		ContactInfo:  contactInfo,
		Description:  description,
		Revisions:    revisions,
		Oid:          oid,
	}
}

func (l *lowerer) lowerObjectIdentity(n *cst.ObjectIdentityNode) *module.ObjectIdentity {
	name := l.text(n.Name)
	statusValue, _ := l.lowerStatus(n.Status)
	description := l.stripQuotes(n.Description.Value)
	l.checkEmptyString(n.Description.Value, name, "DESCRIPTION", types.DiagEmptyDescription)

	var reference string
	if n.Reference != nil {
		reference = l.stripQuotes(n.Reference.Value)
		l.checkEmptyString(n.Reference.Value, name, "REFERENCE", types.DiagEmptyReference)
	}

	oid := l.lowerOidValue(n.Oid)
	span := types.NewSpan(n.Span.Start, oid.Span.End)

	return &module.ObjectIdentity{
		DefBase:     module.DefBase{Name: name, Span: span},
		Status:      statusValue,
		Description: description,
		Reference:   reference,
		Oid:         oid,
	}
}

func (l *lowerer) lowerNotificationType(n *cst.NotificationTypeNode) *module.Notification {
	name := l.text(n.Name)

	var objects []module.NameRef
	if n.Objects != nil {
		for _, nameTok := range n.Objects.Names {
			objects = append(objects, module.NameRef{Name: l.text(nameTok), Span: nameTok.Span})
		}
	}

	statusValue, _ := l.lowerStatus(n.Status)
	description := l.stripQuotes(n.Description.Value)
	l.checkEmptyString(n.Description.Value, name, "DESCRIPTION", types.DiagEmptyDescription)

	var reference string
	if n.Reference != nil {
		reference = l.stripQuotes(n.Reference.Value)
		l.checkEmptyString(n.Reference.Value, name, "REFERENCE", types.DiagEmptyReference)
	}

	oid := l.lowerOidValue(n.Oid)
	span := types.NewSpan(n.Span.Start, oid.Span.End)

	return &module.Notification{
		DefBase:        module.DefBase{Name: name, Span: span},
		Objects:        objects,
		Status:         statusValue,
		Description:    description,
		HasDescription: true, // NOTIFICATION-TYPE DESCRIPTION is required
		Reference:      reference,
		Oid:            &oid,
	}
}

func (l *lowerer) lowerTrapType(n *cst.TrapTypeNode) *module.Notification {
	name := l.text(n.Name)

	var enterprise string
	if n.Enterprise != nil {
		enterprise = l.text(n.Enterprise.Name)
	}

	var variables []module.NameRef
	if n.Variables != nil {
		for _, nameTok := range n.Variables.Names {
			variables = append(variables, module.NameRef{Name: l.text(nameTok), Span: nameTok.Span})
		}
	}

	var description string
	hasDescription := false
	if n.Description != nil {
		hasDescription = true
		description = l.stripQuotes(n.Description.Value)
		l.checkEmptyString(n.Description.Value, name, "DESCRIPTION", types.DiagEmptyDescription)
	}

	var reference string
	if n.Reference != nil {
		reference = l.stripQuotes(n.Reference.Value)
		l.checkEmptyString(n.Reference.Value, name, "REFERENCE", types.DiagEmptyReference)
	}

	trapNumber := l.convertU32(n.Value)
	span := types.NewSpan(n.Span.Start, n.Value.Span.End)

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
	}
}

func (l *lowerer) lowerTextualConvention(n *cst.TextualConventionNode) *module.TypeDef {
	name := l.text(n.Name)

	var displayHint string
	var displayHintSpan types.Span
	if n.DisplayHint != nil {
		displayHint = l.stripQuotes(n.DisplayHint.Value)
		displayHintSpan = n.DisplayHint.Value.Span
		l.checkEmptyString(n.DisplayHint.Value, name, "DISPLAY-HINT", types.DiagEmptyFormat)
	}

	statusValue, statusSpan := l.lowerStatus(n.Status)

	description := l.stripQuotes(n.Description.Value)
	descriptionSpan := n.Description.Value.Span
	l.checkEmptyString(n.Description.Value, name, "DESCRIPTION", types.DiagEmptyDescription)

	var reference string
	var referenceSpan types.Span
	if n.Reference != nil {
		reference = l.stripQuotes(n.Reference.Value)
		referenceSpan = n.Reference.Value.Span
		l.checkEmptyString(n.Reference.Value, name, "REFERENCE", types.DiagEmptyReference)
	}

	syntax := l.lowerTypeSyntax(n.Syntax.Syntax)
	syntaxSpan := types.NewSpan(n.Syntax.Syntax.TypeSyntaxSpan().Start, syntax.SyntaxSpan().End)

	span := types.NewSpan(n.Span.Start, syntaxSpan.End)

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
	}
}

func (l *lowerer) lowerTypeAssignment(n *cst.TypeAssignmentNode) *module.TypeDef {
	name := l.text(n.Name)
	syntax := l.lowerTypeSyntax(n.Syntax)
	span := types.NewSpan(n.Span.Start, syntax.SyntaxSpan().End)

	return &module.TypeDef{
		DefBase: module.DefBase{Name: name, Span: span},
		Syntax:  syntax,
		Status:  types.StatusCurrent, // Default status for plain type assignments
		Spans:   module.TypeDefSpans{Syntax: syntax.SyntaxSpan()},
	}
}

func (l *lowerer) lowerValueAssignment(n *cst.ValueAssignmentNode) *module.ValueAssignment {
	name := l.text(n.Name)
	oid := l.lowerOidValue(n.Oid)
	span := types.NewSpan(n.Span.Start, oid.Span.End)

	return &module.ValueAssignment{
		DefBase: module.DefBase{Name: name, Span: span},
		Oid:     oid,
	}
}

func (l *lowerer) lowerObjectGroup(n *cst.ObjectGroupNode) *module.ObjectGroup {
	name := l.text(n.Name)

	var members []module.NameRef
	if n.Objects != nil {
		for _, nameTok := range n.Objects.Names {
			members = append(members, module.NameRef{Name: l.text(nameTok), Span: nameTok.Span})
		}
	}

	statusValue, _ := l.lowerStatus(n.Status)
	description := l.stripQuotes(n.Description.Value)
	l.checkEmptyString(n.Description.Value, name, "DESCRIPTION", types.DiagEmptyDescription)

	var reference string
	if n.Reference != nil {
		reference = l.stripQuotes(n.Reference.Value)
		l.checkEmptyString(n.Reference.Value, name, "REFERENCE", types.DiagEmptyReference)
	}

	oid := l.lowerOidValue(n.Oid)
	span := types.NewSpan(n.Span.Start, oid.Span.End)

	return &module.ObjectGroup{
		DefBase:     module.DefBase{Name: name, Span: span},
		Objects:     members,
		Status:      statusValue,
		Description: description,
		Reference:   reference,
		Oid:         oid,
	}
}

func (l *lowerer) lowerNotificationGroup(n *cst.NotificationGroupNode) *module.NotificationGroup {
	name := l.text(n.Name)

	var members []module.NameRef
	if n.Notifications != nil {
		for _, nameTok := range n.Notifications.Names {
			members = append(members, module.NameRef{Name: l.text(nameTok), Span: nameTok.Span})
		}
	}

	statusValue, _ := l.lowerStatus(n.Status)
	description := l.stripQuotes(n.Description.Value)
	l.checkEmptyString(n.Description.Value, name, "DESCRIPTION", types.DiagEmptyDescription)

	var reference string
	if n.Reference != nil {
		reference = l.stripQuotes(n.Reference.Value)
		l.checkEmptyString(n.Reference.Value, name, "REFERENCE", types.DiagEmptyReference)
	}

	oid := l.lowerOidValue(n.Oid)
	span := types.NewSpan(n.Span.Start, oid.Span.End)

	return &module.NotificationGroup{
		DefBase:       module.DefBase{Name: name, Span: span},
		Notifications: members,
		Status:        statusValue,
		Description:   description,
		Reference:     reference,
		Oid:           oid,
	}
}
