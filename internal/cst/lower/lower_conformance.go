package lower

import (
	"github.com/golangsnmp/gomib/internal/cst"
	"github.com/golangsnmp/gomib/internal/module"
	"github.com/golangsnmp/gomib/internal/types"
)

func (l *lowerer) lowerModuleCompliance(n *cst.ModuleComplianceNode) *module.ModuleCompliance {
	name := l.text(n.Name)
	statusValue, _ := l.lowerStatus(n.Status)
	description := l.stripQuotes(n.Description.Value)
	l.checkEmptyString(n.Description.Value, name, "DESCRIPTION", types.DiagEmptyDescription)

	var reference string
	if n.Reference != nil {
		reference = l.stripQuotes(n.Reference.Value)
		l.checkEmptyString(n.Reference.Value, name, "REFERENCE", types.DiagEmptyReference)
	}

	var modules []module.ComplianceModule
	for i := range n.Modules {
		modules = append(modules, l.lowerComplianceModule(&n.Modules[i]))
	}

	oid := l.lowerOidValue(n.Oid)
	span := types.NewSpan(n.Span.Start, oid.Span.End)

	return &module.ModuleCompliance{
		DefBase:     module.DefBase{Name: name, Span: span},
		Status:      statusValue,
		Description: description,
		Reference:   reference,
		Modules:     modules,
		Oid:         oid,
	}
}

func (l *lowerer) lowerComplianceModule(n *cst.ComplianceModuleNode) module.ComplianceModule {
	var moduleName string
	if !n.ModuleName.IsZero() {
		moduleName = l.text(n.ModuleName)
	}

	var mandatoryGroups []string
	if n.MandatoryGroups != nil {
		for _, name := range n.MandatoryGroups.Names {
			mandatoryGroups = append(mandatoryGroups, l.text(name))
		}
	}

	var groups []module.ComplianceGroup
	for i := range n.Groups {
		groups = append(groups, l.lowerComplianceGroup(&n.Groups[i]))
	}

	var objects []module.ComplianceObject
	for i := range n.Objects {
		objects = append(objects, l.lowerComplianceObject(&n.Objects[i]))
	}

	return module.ComplianceModule{
		ModuleName:      moduleName,
		MandatoryGroups: mandatoryGroups,
		Groups:          groups,
		Objects:         objects,
		Span:            n.Span,
	}
}

func (l *lowerer) lowerComplianceGroup(n *cst.ComplianceGroupNode) module.ComplianceGroup {
	groupName := l.text(n.Name)
	description := ""
	var end types.ByteOffset
	if n.Description != nil {
		description = l.stripQuotes(n.Description.Value)
		end = n.Description.Value.Span.End
	} else {
		end = n.Span.End
	}

	return module.ComplianceGroup{
		Group:       groupName,
		Description: description,
		Span:        types.NewSpan(n.Span.Start, end),
	}
}

func (l *lowerer) lowerComplianceObject(n *cst.ComplianceObjectNode) module.ComplianceObject {
	objectName := l.text(n.Name)

	var syntax module.TypeSyntax
	if n.Syntax != nil {
		syntax = l.lowerTypeSyntax(n.Syntax.Syntax)
	}

	var writeSyntax module.TypeSyntax
	if n.WriteSyntax != nil {
		writeSyntax = l.lowerTypeSyntax(n.WriteSyntax.Syntax)
	}

	var minAccess *types.Access
	if n.Access != nil {
		accessValue, _, _ := l.lowerAccess(n.Access)
		minAccess = &accessValue
	}

	description := ""
	var end types.ByteOffset
	if n.Description != nil {
		description = l.stripQuotes(n.Description.Value)
		end = n.Description.Value.Span.End
	} else {
		end = n.Span.End
	}

	return module.ComplianceObject{
		Object:      objectName,
		Syntax:      syntax,
		WriteSyntax: writeSyntax,
		MinAccess:   minAccess,
		Description: description,
		Span:        types.NewSpan(n.Span.Start, end),
	}
}

func (l *lowerer) lowerAgentCapabilities(n *cst.AgentCapabilitiesNode) *module.AgentCapabilities {
	name := l.text(n.Name)

	productRelease := ""
	if n.ProductRelease != nil {
		productRelease = l.stripQuotes(n.ProductRelease.Value)
	}

	statusValue, _ := l.lowerStatus(n.Status)
	description := l.stripQuotes(n.Description.Value)
	l.checkEmptyString(n.Description.Value, name, "DESCRIPTION", types.DiagEmptyDescription)

	var reference string
	if n.Reference != nil {
		reference = l.stripQuotes(n.Reference.Value)
		l.checkEmptyString(n.Reference.Value, name, "REFERENCE", types.DiagEmptyReference)
	}

	var supports []module.SupportsModule
	for i := range n.Modules {
		supports = append(supports, l.lowerSupportsModule(&n.Modules[i]))
	}

	oid := l.lowerOidValue(n.Oid)
	span := types.NewSpan(n.Span.Start, oid.Span.End)

	return &module.AgentCapabilities{
		DefBase:        module.DefBase{Name: name, Span: span},
		ProductRelease: productRelease,
		Status:         statusValue,
		Description:    description,
		Reference:      reference,
		Supports:       supports,
		Oid:            oid,
	}
}

func (l *lowerer) lowerSupportsModule(n *cst.CapabilityModuleNode) module.SupportsModule {
	moduleName := l.text(n.ModuleName)

	var includes []string
	if n.Includes != nil {
		for _, name := range n.Includes.Names {
			includes = append(includes, l.text(name))
		}
	}

	var variations []module.Variation
	for i := range n.Variations {
		variations = append(variations, l.lowerVariation(&n.Variations[i]))
	}

	return module.SupportsModule{
		ModuleName: moduleName,
		Includes:   includes,
		Variations: variations,
		Span:       n.Span,
	}
}

func (l *lowerer) lowerVariation(n *cst.CapabilityVariationNode) module.Variation {
	varName := l.text(n.Name)

	var syntax module.TypeSyntax
	if n.Syntax != nil {
		syntax = l.lowerTypeSyntax(n.Syntax.Syntax)
	}

	var writeSyntax module.TypeSyntax
	if n.WriteSyntax != nil {
		writeSyntax = l.lowerTypeSyntax(n.WriteSyntax.Syntax)
	}

	var access *types.Access
	if n.Access != nil {
		accessValue, _, _ := l.lowerAccess(n.Access)
		access = &accessValue
	}

	var creationRequires []string
	if n.CreationRequires != nil {
		for _, name := range n.CreationRequires.Names {
			creationRequires = append(creationRequires, l.text(name))
		}
	}

	var defval module.DefVal
	if n.DefVal != nil {
		defval = l.lowerDefVal(n.DefVal.Content)
	}

	description := ""
	var end types.ByteOffset
	if n.Description != nil {
		description = l.stripQuotes(n.Description.Value)
		end = n.Description.Value.Span.End
	} else {
		end = n.Span.End
	}

	return module.Variation{
		Name:             varName,
		Syntax:           syntax,
		WriteSyntax:      writeSyntax,
		Access:           access,
		CreationRequires: creationRequires,
		DefVal:           defval,
		Description:      description,
		Span:             types.NewSpan(n.Span.Start, end),
	}
}
