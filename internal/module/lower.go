package module

import (
	"fmt"
	"log/slog"

	"github.com/golangsnmp/gomib/internal/ast"
	"github.com/golangsnmp/gomib/internal/types"
)

// LoweringContext tracks state during the lowering process.
type LoweringContext struct {
	// Diagnostics collected during lowering.
	Diagnostics []types.Diagnostic
	// Language detected from imports (may be updated as imports are processed).
	Language SmiLanguage
	types.Logger
}

// newLoweringContext creates a new lowering context with an optional logger.
// If logger is nil, logging is disabled (zero overhead).
func newLoweringContext(logger *slog.Logger) *LoweringContext {
	return &LoweringContext{
		Language: SmiLanguageUnknown,
		Logger:   types.Logger{L: logger},
	}
}

// AddDiagnostic adds a diagnostic.
func (ctx *LoweringContext) AddDiagnostic(d types.Diagnostic) {
	ctx.Diagnostics = append(ctx.Diagnostics, d)
}

// isSMIv2BaseModule returns true if the module name is an SMIv2 base module.
func isSMIv2BaseModule(module string) bool {
	switch module {
	case "SNMPv2-SMI", "SNMPv2-TC", "SNMPv2-CONF", "SNMPv2-MIB":
		return true
	default:
		return false
	}
}

// Lower transforms an AST module into a normalized Module.
//
// This is the main entry point for lowering. It:
//  1. Detects the SMI language from imports
//  2. Lowers imports
//  3. Lowers each definition
//
// The AST is not needed after lowering.
// If logger is nil, logging is disabled (zero overhead).
func Lower(astModule *ast.Module, logger *slog.Logger) *Module {
	ctx := newLoweringContext(logger)

	// Create module
	module := NewModule(astModule.Name.Name, astModule.Span)

	ctx.Log(slog.LevelDebug, "lowering module", slog.String("module", module.Name))

	// Lower imports and detect language
	module.Imports = lowerImports(astModule.Imports, ctx)
	module.Language = ctx.Language

	ctx.Log(slog.LevelDebug, "detected language",
		slog.String("module", module.Name),
		slog.String("language", module.Language.String()))

	// Lower definitions
	for _, def := range astModule.Body {
		if lowered := lowerDefinition(def, ctx); lowered != nil {
			module.Definitions = append(module.Definitions, lowered)
		}
	}

	ctx.Log(slog.LevelDebug, "lowering complete",
		slog.String("module", module.Name),
		slog.Int("definitions", len(module.Definitions)))

	// Move diagnostics from AST and add lowering diagnostics
	module.Diagnostics = append(module.Diagnostics, astModule.Diagnostics...)
	module.Diagnostics = append(module.Diagnostics, ctx.Diagnostics...)

	return module
}

// lowerImports lowers import clauses and detects SMI language.
func lowerImports(importClauses []ast.ImportClause, ctx *LoweringContext) []Import {
	var imports []Import

	for _, clause := range importClauses {
		fromModule := clause.FromModule.Name

		// Detect language from imports
		if isSMIv2BaseModule(fromModule) {
			ctx.Language = SmiLanguageSMIv2
		}

		// Flatten each symbol
		for _, symbol := range clause.Symbols {
			imports = append(imports, NewImport(fromModule, symbol.Name, clause.Span))
		}
	}

	// Default to SMIv1 if no SMIv2 imports detected
	if ctx.Language == SmiLanguageUnknown {
		ctx.Language = SmiLanguageSMIv1
	}

	return imports
}

// lowerDefinition lowers a single definition.
// Returns nil for definitions that are filtered out (MACRO, Error).
func lowerDefinition(def ast.Definition, ctx *LoweringContext) Definition {
	switch d := def.(type) {
	case *ast.ObjectTypeDef:
		return lowerObjectType(d, ctx)
	case *ast.ModuleIdentityDef:
		return lowerModuleIdentity(d)
	case *ast.ObjectIdentityDef:
		return lowerObjectIdentity(d)
	case *ast.NotificationTypeDef:
		return lowerNotificationType(d)
	case *ast.TrapTypeDef:
		return lowerTrapType(d)
	case *ast.TextualConventionDef:
		return lowerTextualConvention(d)
	case *ast.TypeAssignmentDef:
		return lowerTypeAssignment(d)
	case *ast.ValueAssignmentDef:
		return lowerValueAssignment(d)
	case *ast.ObjectGroupDef:
		return lowerObjectGroup(d)
	case *ast.NotificationGroupDef:
		return lowerNotificationGroup(d)
	case *ast.ModuleComplianceDef:
		return lowerModuleCompliance(d)
	case *ast.AgentCapabilitiesDef:
		return lowerAgentCapabilities(d)
	case *ast.MacroDefinitionDef, *ast.ErrorDef:
		// Filter out non-semantic definitions
		return nil
	default:
		ctx.Log(slog.LevelWarn, "unknown definition type",
			slog.String("type", fmt.Sprintf("%T", def)))
		return nil
	}
}

// === Definition lowering functions ===

func lowerObjectType(def *ast.ObjectTypeDef, _ *LoweringContext) *ObjectType {
	var status types.Status
	if def.Status != nil {
		status = lowerStatus(def.Status.Value)
	} else {
		status = types.StatusCurrent
	}

	var units string
	if def.Units != nil {
		units = def.Units.Value
	}

	var description string
	if def.Description != nil {
		description = def.Description.Value
	}

	var reference string
	if def.Reference != nil {
		reference = def.Reference.Value
	}

	var augments string
	if def.Augments != nil {
		augments = def.Augments.Target.Name
	}

	var defval DefVal
	if def.DefVal != nil {
		defval = lowerDefVal(def.DefVal)
	}

	return &ObjectType{
		Name:        def.Name.Name,
		Syntax:      lowerTypeSyntax(def.Syntax.Syntax),
		Units:       units,
		Access:      lowerAccess(def.Access.Value),
		Status:      status,
		Description: description,
		Reference:   reference,
		Index:       lowerIndexClause(def.Index),
		Augments:    augments,
		DefVal:      defval,
		Oid:         lowerOidAssignment(def.OidAssignment),
		Span:        def.Span,
	}
}

func lowerModuleIdentity(def *ast.ModuleIdentityDef) *ModuleIdentity {
	revisions := make([]Revision, len(def.Revisions))
	for i, r := range def.Revisions {
		revisions[i] = Revision{
			Date:        r.Date.Value,
			Description: r.Description.Value,
		}
	}

	return &ModuleIdentity{
		Name:         def.Name.Name,
		LastUpdated:  def.LastUpdated.Value,
		Organization: def.Organization.Value,
		ContactInfo:  def.ContactInfo.Value,
		Description:  def.Description.Value,
		Revisions:    revisions,
		Oid:          lowerOidAssignment(def.OidAssignment),
		Span:         def.Span,
	}
}

func lowerObjectIdentity(def *ast.ObjectIdentityDef) *ObjectIdentity {
	var reference string
	if def.Reference != nil {
		reference = def.Reference.Value
	}

	return &ObjectIdentity{
		Name:        def.Name.Name,
		Status:      lowerStatus(def.Status.Value),
		Description: def.Description.Value,
		Reference:   reference,
		Oid:         lowerOidAssignment(def.OidAssignment),
		Span:        def.Span,
	}
}

func lowerNotificationType(def *ast.NotificationTypeDef) *Notification {
	objects := make([]string, len(def.Objects))
	for i, o := range def.Objects {
		objects[i] = o.Name
	}

	var reference string
	if def.Reference != nil {
		reference = def.Reference.Value
	}

	oid := lowerOidAssignment(def.OidAssignment)

	return &Notification{
		Name:        def.Name.Name,
		Objects:     objects,
		Status:      lowerStatus(def.Status.Value),
		Description: def.Description.Value,
		Reference:   reference,
		TrapInfo:    nil,
		Oid:         &oid,
		Span:        def.Span,
	}
}

func lowerTrapType(def *ast.TrapTypeDef) *Notification {
	variables := make([]string, len(def.Variables))
	for i, v := range def.Variables {
		variables[i] = v.Name
	}

	var description string
	if def.Description != nil {
		description = def.Description.Value
	}

	var reference string
	if def.Reference != nil {
		reference = def.Reference.Value
	}

	return &Notification{
		Name:        def.Name.Name,
		Objects:     variables,
		Status:      types.StatusCurrent, // TRAP-TYPE doesn't have STATUS
		Description: description,
		Reference:   reference,
		TrapInfo: &TrapInfo{
			Enterprise: def.Enterprise.Name,
			TrapNumber: def.TrapNumber,
		},
		Oid:  nil, // TRAP-TYPE OID is derived from enterprise + trap_number
		Span: def.Span,
	}
}

func lowerTextualConvention(def *ast.TextualConventionDef) *TypeDef {
	var displayHint string
	if def.DisplayHint != nil {
		displayHint = def.DisplayHint.Value
	}

	var reference string
	if def.Reference != nil {
		reference = def.Reference.Value
	}

	return &TypeDef{
		Name:                def.Name.Name,
		Syntax:              lowerTypeSyntax(def.Syntax.Syntax),
		BaseType:            nil, // Derived from syntax during resolution
		DisplayHint:         displayHint,
		Status:              lowerStatus(def.Status.Value),
		Description:         def.Description.Value,
		Reference:           reference,
		IsTextualConvention: true,
		Span:                def.Span,
	}
}

func lowerTypeAssignment(def *ast.TypeAssignmentDef) *TypeDef {
	return &TypeDef{
		Name:                def.Name.Name,
		Syntax:              lowerTypeSyntax(def.Syntax),
		BaseType:            nil, // Derived from syntax during resolution
		DisplayHint:         "",
		Status:              types.StatusCurrent,
		Description:         "",
		Reference:           "",
		IsTextualConvention: false,
		Span:                def.Span,
	}
}

func lowerValueAssignment(def *ast.ValueAssignmentDef) *ValueAssignment {
	return &ValueAssignment{
		Name: def.Name.Name,
		Oid:  lowerOidAssignment(def.OidAssignment),
		Span: def.Span,
	}
}

func lowerObjectGroup(def *ast.ObjectGroupDef) *ObjectGroup {
	objects := make([]string, len(def.Objects))
	for i, o := range def.Objects {
		objects[i] = o.Name
	}

	var reference string
	if def.Reference != nil {
		reference = def.Reference.Value
	}

	return &ObjectGroup{
		Name:        def.Name.Name,
		Objects:     objects,
		Status:      lowerStatus(def.Status.Value),
		Description: def.Description.Value,
		Reference:   reference,
		Oid:         lowerOidAssignment(def.OidAssignment),
		Span:        def.Span,
	}
}

func lowerNotificationGroup(def *ast.NotificationGroupDef) *NotificationGroup {
	notifications := make([]string, len(def.Notifications))
	for i, n := range def.Notifications {
		notifications[i] = n.Name
	}

	var reference string
	if def.Reference != nil {
		reference = def.Reference.Value
	}

	return &NotificationGroup{
		Name:          def.Name.Name,
		Notifications: notifications,
		Status:        lowerStatus(def.Status.Value),
		Description:   def.Description.Value,
		Reference:     reference,
		Oid:           lowerOidAssignment(def.OidAssignment),
		Span:          def.Span,
	}
}

func lowerModuleCompliance(def *ast.ModuleComplianceDef) *ModuleCompliance {
	modules := make([]ComplianceModule, len(def.Modules))
	for i, m := range def.Modules {
		modules[i] = lowerComplianceModule(m)
	}

	var reference string
	if def.Reference != nil {
		reference = def.Reference.Value
	}

	return &ModuleCompliance{
		Name:        def.Name.Name,
		Status:      lowerStatus(def.Status.Value),
		Description: def.Description.Value,
		Reference:   reference,
		Modules:     modules,
		Oid:         lowerOidAssignment(def.OidAssignment),
		Span:        def.Span,
	}
}

func lowerComplianceModule(m ast.ComplianceModule) ComplianceModule {
	var groups []ComplianceGroup
	var objects []ComplianceObject

	for _, c := range m.Compliances {
		switch comp := c.(type) {
		case *ast.ComplianceGroup:
			groups = append(groups, ComplianceGroup{
				Group:       comp.Group.Name,
				Description: comp.Description.Value,
			})
		case *ast.ComplianceObject:
			objects = append(objects, lowerComplianceObject(comp))
		}
	}

	mandatoryGroups := make([]string, len(m.MandatoryGroups))
	for i, g := range m.MandatoryGroups {
		mandatoryGroups[i] = g.Name
	}

	var moduleName string
	if m.ModuleName != nil {
		moduleName = m.ModuleName.Name
	}

	return ComplianceModule{
		ModuleName:      moduleName,
		MandatoryGroups: mandatoryGroups,
		Groups:          groups,
		Objects:         objects,
	}
}

func lowerComplianceObject(o *ast.ComplianceObject) ComplianceObject {
	var syntax TypeSyntax
	if o.Syntax != nil {
		syntax = lowerTypeSyntax(o.Syntax.Syntax)
	}

	var writeSyntax TypeSyntax
	if o.WriteSyntax != nil {
		writeSyntax = lowerTypeSyntax(o.WriteSyntax.Syntax)
	}

	var minAccess *types.Access
	if o.MinAccess != nil {
		a := lowerAccess(o.MinAccess.Value)
		minAccess = &a
	}

	return ComplianceObject{
		Object:      o.Object.Name,
		Syntax:      syntax,
		WriteSyntax: writeSyntax,
		MinAccess:   minAccess,
		Description: o.Description.Value,
	}
}

func lowerAgentCapabilities(def *ast.AgentCapabilitiesDef) *AgentCapabilities {
	supports := make([]SupportsModule, len(def.Supports))
	for i, s := range def.Supports {
		supports[i] = lowerSupportsModule(s)
	}

	var reference string
	if def.Reference != nil {
		reference = def.Reference.Value
	}

	return &AgentCapabilities{
		Name:           def.Name.Name,
		ProductRelease: def.ProductRelease.Value,
		Status:         lowerStatus(def.Status.Value),
		Description:    def.Description.Value,
		Reference:      reference,
		Supports:       supports,
		Oid:            lowerOidAssignment(def.OidAssignment),
		Span:           def.Span,
	}
}

func lowerSupportsModule(s ast.SupportsModule) SupportsModule {
	var objectVariations []ObjectVariation
	var notificationVariations []NotificationVariation

	for _, v := range s.Variations {
		switch variation := v.(type) {
		case *ast.ObjectVariation:
			objectVariations = append(objectVariations, lowerObjectVariation(variation))
		case *ast.NotificationVariation:
			notificationVariations = append(notificationVariations, lowerNotificationVariation(variation))
		}
	}

	includes := make([]string, len(s.Includes))
	for i, inc := range s.Includes {
		includes[i] = inc.Name
	}

	return SupportsModule{
		ModuleName:             s.ModuleName.Name,
		Includes:               includes,
		ObjectVariations:       objectVariations,
		NotificationVariations: notificationVariations,
	}
}

func lowerObjectVariation(v *ast.ObjectVariation) ObjectVariation {
	var syntax TypeSyntax
	if v.Syntax != nil {
		syntax = lowerTypeSyntax(v.Syntax.Syntax)
	}

	var writeSyntax TypeSyntax
	if v.WriteSyntax != nil {
		writeSyntax = lowerTypeSyntax(v.WriteSyntax.Syntax)
	}

	var access *types.Access
	if v.Access != nil {
		a := lowerAccess(v.Access.Value)
		access = &a
	}

	var creationRequires []string
	if len(v.CreationRequires) > 0 {
		creationRequires = make([]string, len(v.CreationRequires))
		for i, cr := range v.CreationRequires {
			creationRequires[i] = cr.Name
		}
	}

	var defval DefVal
	if v.DefVal != nil {
		defval = lowerDefVal(v.DefVal)
	}

	return ObjectVariation{
		Object:           v.Object.Name,
		Syntax:           syntax,
		WriteSyntax:      writeSyntax,
		Access:           access,
		CreationRequires: creationRequires,
		DefVal:           defval,
		Description:      v.Description.Value,
	}
}

func lowerNotificationVariation(v *ast.NotificationVariation) NotificationVariation {
	var access *types.Access
	if v.Access != nil {
		a := lowerAccess(v.Access.Value)
		access = &a
	}

	return NotificationVariation{
		Notification: v.Notification.Name,
		Access:       access,
		Description:  v.Description.Value,
	}
}

// === Helper lowering functions ===

func lowerTypeSyntax(syntax ast.TypeSyntax) TypeSyntax {
	switch s := syntax.(type) {
	case *ast.TypeSyntaxTypeRef:
		return &TypeSyntaxTypeRef{Name: s.Name.Name}

	case *ast.TypeSyntaxIntegerEnum:
		namedNumbers := make([]NamedNumber, len(s.NamedNumbers))
		for i, nn := range s.NamedNumbers {
			namedNumbers[i] = NewNamedNumber(nn.Name.Name, nn.Value)
		}
		var base string
		if s.Base != nil {
			base = s.Base.Name
		}
		return &TypeSyntaxIntegerEnum{Base: base, NamedNumbers: namedNumbers}

	case *ast.TypeSyntaxBits:
		namedBits := make([]NamedBit, len(s.NamedBits))
		for i, nb := range s.NamedBits {
			// BITS positions are small non-negative integers (0-127)
			namedBits[i] = NewNamedBit(nb.Name.Name, uint32(nb.Value))
		}
		return &TypeSyntaxBits{NamedBits: namedBits}

	case *ast.TypeSyntaxConstrained:
		return &TypeSyntaxConstrained{
			Base:       lowerTypeSyntax(s.Base),
			Constraint: lowerConstraint(s.Constraint),
		}

	case *ast.TypeSyntaxSequenceOf:
		return &TypeSyntaxSequenceOf{EntryType: s.EntryType.Name}

	case *ast.TypeSyntaxSequence:
		fields := make([]SequenceField, len(s.Fields))
		for i, f := range s.Fields {
			fields[i] = NewSequenceField(f.Name.Name, lowerTypeSyntax(f.Syntax))
		}
		return &TypeSyntaxSequence{Fields: fields}

	case *ast.TypeSyntaxChoice:
		// CHOICE is normalized to its first alternative's type.
		// CHOICE only appears in SMI base modules (not user MIBs), and the only
		// CHOICE usable as OBJECT-TYPE SYNTAX is NetworkAddress which has one alternative.
		if len(s.Alternatives) > 0 {
			return lowerTypeSyntax(s.Alternatives[0].Syntax)
		}
		// Empty CHOICE (shouldn't happen) - fall back to OCTET STRING
		return &TypeSyntaxOctetString{}

	case *ast.TypeSyntaxOctetString:
		return &TypeSyntaxOctetString{}

	case *ast.TypeSyntaxObjectIdentifier:
		return &TypeSyntaxObjectIdentifier{}

	default:
		// Unknown type - fall back to type ref
		return &TypeSyntaxOctetString{}
	}
}

func lowerConstraint(constraint ast.Constraint) Constraint {
	switch c := constraint.(type) {
	case *ast.ConstraintSize:
		ranges := make([]Range, len(c.Ranges))
		for i, r := range c.Ranges {
			ranges[i] = lowerRange(r)
		}
		return &ConstraintSize{Ranges: ranges}

	case *ast.ConstraintRange:
		ranges := make([]Range, len(c.Ranges))
		for i, r := range c.Ranges {
			ranges[i] = lowerRange(r)
		}
		return &ConstraintRange{Ranges: ranges}

	default:
		return &ConstraintRange{}
	}
}

func lowerRange(r ast.Range) Range {
	return Range{
		Min: lowerRangeValue(r.Min),
		Max: lowerRangeValue(r.Max),
	}
}

func lowerRangeValue(value ast.RangeValue) RangeValue {
	if value == nil {
		return nil
	}

	switch v := value.(type) {
	case *ast.RangeValueSigned:
		return &RangeValueSigned{Value: v.Value}

	case *ast.RangeValueUnsigned:
		return &RangeValueUnsigned{Value: v.Value}

	case *ast.RangeValueIdent:
		// Handle MIN/MAX keywords
		switch v.Name.Name {
		case "MIN":
			return &RangeValueMin{}
		case "MAX":
			return &RangeValueMax{}
		default:
			// Shouldn't happen, but fallback to unsigned 0
			return &RangeValueUnsigned{Value: 0}
		}

	default:
		return &RangeValueUnsigned{Value: 0}
	}
}

func lowerOidAssignment(oid ast.OidAssignment) OidAssignment {
	components := make([]OidComponent, len(oid.Components))
	for i, c := range oid.Components {
		components[i] = lowerOidComponent(c)
	}
	return NewOidAssignment(components, oid.Span)
}

func lowerOidComponent(comp ast.OidComponent) OidComponent {
	switch c := comp.(type) {
	case *ast.OidComponentName:
		return &OidComponentName{NameValue: c.Name.Name}

	case *ast.OidComponentNumber:
		return &OidComponentNumber{Value: c.Value}

	case *ast.OidComponentNamedNumber:
		return &OidComponentNamedNumber{
			NameValue:   c.Name.Name,
			NumberValue: c.Num,
		}

	case *ast.OidComponentQualifiedName:
		return &OidComponentQualifiedName{
			ModuleValue: c.ModuleName.Name,
			NameValue:   c.Name.Name,
		}

	case *ast.OidComponentQualifiedNamedNumber:
		return &OidComponentQualifiedNamedNumber{
			ModuleValue: c.ModuleName.Name,
			NameValue:   c.Name.Name,
			NumberValue: c.Num,
		}

	default:
		return &OidComponentNumber{Value: 0}
	}
}

func lowerAccess(access ast.AccessValue) types.Access {
	switch access {
	case ast.AccessValueReadOnly, ast.AccessValueReportOnly:
		return types.AccessReadOnly
	case ast.AccessValueReadWrite, ast.AccessValueInstall, ast.AccessValueInstallNotify:
		return types.AccessReadWrite
	case ast.AccessValueReadCreate:
		return types.AccessReadCreate
	case ast.AccessValueNotAccessible, ast.AccessValueNotImplemented:
		return types.AccessNotAccessible
	case ast.AccessValueAccessibleForNotify:
		return types.AccessAccessibleForNotify
	case ast.AccessValueWriteOnly:
		return types.AccessWriteOnly
	default:
		return types.AccessNotAccessible
	}
}

func lowerStatus(status ast.StatusValue) types.Status {
	switch status {
	case ast.StatusValueCurrent, ast.StatusValueMandatory:
		return types.StatusCurrent
	case ast.StatusValueDeprecated, ast.StatusValueOptional:
		return types.StatusDeprecated
	case ast.StatusValueObsolete:
		return types.StatusObsolete
	default:
		return types.StatusCurrent
	}
}

func lowerIndexClause(clause ast.IndexClause) []IndexItem {
	if clause == nil {
		return nil
	}

	indexes := clause.Indexes()
	items := make([]IndexItem, len(indexes))
	for i, idx := range indexes {
		items[i] = IndexItem{Object: idx.Object.Name, Implied: idx.Implied}
	}
	return items
}

func lowerDefVal(clause *ast.DefValClause) DefVal {
	return lowerDefValContent(clause.Value)
}

func lowerDefValContent(content ast.DefValContent) DefVal {
	switch c := content.(type) {
	case *ast.DefValContentInteger:
		return &DefValInteger{Value: c.Value}

	case *ast.DefValContentUnsigned:
		return &DefValUnsigned{Value: c.Value}

	case *ast.DefValContentString:
		return &DefValString{Value: c.Value.Value}

	case *ast.DefValContentIdentifier:
		// Could be enum label or OID reference - we can't distinguish
		// until semantic analysis, so treat as Enum (most common case)
		return &DefValEnum{Name: c.Name.Name}

	case *ast.DefValContentBits:
		labels := make([]string, len(c.Labels))
		for i, l := range c.Labels {
			labels[i] = l.Name
		}
		return &DefValBits{Labels: labels}

	case *ast.DefValContentHexString:
		return &DefValHexString{Value: c.Content}

	case *ast.DefValContentBinaryString:
		return &DefValBinaryString{Value: c.Content}

	case *ast.DefValContentObjectIdentifier:
		components := make([]OidComponent, len(c.Components))
		for i, comp := range c.Components {
			components[i] = lowerOidComponent(comp)
		}
		return &DefValOidValue{Components: components}

	default:
		return &DefValInteger{Value: 0}
	}
}
