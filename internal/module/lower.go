package module

import (
	"fmt"
	"log/slog"

	"github.com/golangsnmp/gomib/internal/ast"
	"github.com/golangsnmp/gomib/internal/types"
	"github.com/golangsnmp/gomib/mib"
)

// LoweringContext tracks state accumulated during the lowering pass.
type LoweringContext struct {
	Diagnostics []mib.Diagnostic
	Language    Language
	DiagConfig  mib.DiagnosticConfig
	source      []byte // source text for span-to-line/column conversion
	types.Logger
}

// newLoweringContext returns a LoweringContext. A nil logger disables logging.
func newLoweringContext(source []byte, logger *slog.Logger, diagConfig mib.DiagnosticConfig) *LoweringContext {
	return &LoweringContext{
		Language:   LanguageUnknown,
		DiagConfig: diagConfig,
		source:     source,
		Logger:     types.Logger{L: logger},
	}
}

// spanToLineCol converts a byte offset to 1-based line and column numbers.
// Returns (0, 0) if the source is nil or the offset is out of range.
func spanToLineCol(source []byte, offset types.ByteOffset) (line, col int) {
	if source == nil || int(offset) > len(source) {
		return 0, 0
	}
	line = 1
	lastNewline := -1
	for i := 0; i < int(offset); i++ {
		if source[i] == '\n' {
			line++
			lastNewline = i
		}
	}
	col = int(offset) - lastNewline
	return line, col
}

// emitDiagnostic records a diagnostic if the current config allows it.
func (ctx *LoweringContext) emitDiagnostic(code string, severity mib.Severity, moduleName string, message string) {
	if !ctx.DiagConfig.ShouldReport(code, severity) {
		return
	}
	ctx.Diagnostics = append(ctx.Diagnostics, mib.Diagnostic{
		Severity: severity,
		Code:     code,
		Message:  message,
		Module:   moduleName,
		Line:     0,
		Column:   0,
	})
}

// AddDiagnostic appends a diagnostic to the context.
func (ctx *LoweringContext) AddDiagnostic(d mib.Diagnostic) {
	ctx.Diagnostics = append(ctx.Diagnostics, d)
}

// isSMIv2Import reports whether importing from this module indicates SMIv2.
func isSMIv2Import(name string) bool {
	bm, ok := BaseModuleFromName(name)
	return ok && bm.IsSMIv2()
}

// Lower transforms an AST module into a normalized Module. The AST is not
// needed after lowering. Source is the original source text used to compute
// diagnostic line/column from byte offset spans. A nil logger disables logging.
func Lower(astModule *ast.Module, source []byte, logger *slog.Logger, diagConfig mib.DiagnosticConfig) *Module {
	ctx := newLoweringContext(source, logger, diagConfig)

	module := NewModule(astModule.Name.Name, astModule.Span)

	ctx.Log(slog.LevelDebug, "lowering module", slog.String("module", module.Name))

	module.Imports = lowerImports(astModule.Imports, ctx)
	module.Language = ctx.Language

	ctx.Log(slog.LevelDebug, "detected language",
		slog.String("module", module.Name),
		slog.String("language", module.Language.String()))

	for _, def := range astModule.Body {
		if lowered := lowerDefinition(def, ctx); lowered != nil {
			module.Definitions = append(module.Definitions, lowered)
		}
	}

	ctx.Log(slog.LevelDebug, "lowering complete",
		slog.String("module", module.Name),
		slog.Int("definitions", len(module.Definitions)))

	if module.Language == LanguageSMIv2 && !IsBaseModule(module.Name) {
		hasModuleIdentity := false
		for _, def := range module.Definitions {
			if mi, ok := def.(*ModuleIdentity); ok {
				hasModuleIdentity = true
				checkRevisionLastUpdated(ctx, module.Name, mi)
				break
			}
		}
		if !hasModuleIdentity {
			ctx.emitDiagnostic("missing-module-identity", mib.SeverityError, module.Name,
				fmt.Sprintf("SMIv2 module %s lacks MODULE-IDENTITY", module.Name))
		}
	}

	for _, d := range astModule.Diagnostics {
		line, col := spanToLineCol(ctx.source, d.Span.Start)
		module.Diagnostics = append(module.Diagnostics, mib.Diagnostic{
			Severity: d.Severity,
			Code:     d.Code,
			Message:  d.Message,
			Module:   module.Name,
			Line:     line,
			Column:   col,
		})
	}

	module.Diagnostics = append(module.Diagnostics, ctx.Diagnostics...)

	return module
}

// identNames extracts the Name field from each Ident.
func identNames(idents []ast.Ident) []string {
	names := make([]string, len(idents))
	for i, id := range idents {
		names[i] = id.Name
	}
	return names
}

// lowerImports flattens import clauses and detects the SMI language.
func lowerImports(importClauses []ast.ImportClause, ctx *LoweringContext) []Import {
	var imports []Import

	for _, clause := range importClauses {
		fromModule := clause.FromModule.Name

		if isSMIv2Import(fromModule) {
			ctx.Language = LanguageSMIv2
		}

		for _, symbol := range clause.Symbols {
			imports = append(imports, NewImport(fromModule, symbol.Name, clause.Span))
		}
	}

	if ctx.Language == LanguageUnknown {
		ctx.Language = LanguageSMIv1
	}

	return imports
}

// lowerDefinition converts an AST definition into a normalized Definition.
// Returns nil for non-semantic definitions (MACROs, errors).
func lowerDefinition(def ast.Definition, ctx *LoweringContext) Definition {
	switch d := def.(type) {
	case *ast.ObjectTypeDef:
		return lowerObjectType(d, ctx)
	case *ast.ModuleIdentityDef:
		return lowerModuleIdentity(d, ctx)
	case *ast.ObjectIdentityDef:
		return lowerObjectIdentity(d, ctx)
	case *ast.NotificationTypeDef:
		return lowerNotificationType(d, ctx)
	case *ast.TrapTypeDef:
		return lowerTrapType(d)
	case *ast.TextualConventionDef:
		return lowerTextualConvention(d, ctx)
	case *ast.TypeAssignmentDef:
		return lowerTypeAssignment(d, ctx)
	case *ast.ValueAssignmentDef:
		return lowerValueAssignment(d, ctx)
	case *ast.ObjectGroupDef:
		return lowerObjectGroup(d, ctx)
	case *ast.NotificationGroupDef:
		return lowerNotificationGroup(d, ctx)
	case *ast.ModuleComplianceDef:
		return lowerModuleCompliance(d, ctx)
	case *ast.AgentCapabilitiesDef:
		return lowerAgentCapabilities(d, ctx)
	case *ast.MacroDefinitionDef, *ast.ErrorDef:
		// Non-semantic definitions
		return nil
	default:
		ctx.Log(slog.LevelWarn, "unknown definition type",
			slog.String("type", fmt.Sprintf("%T", def)))
		return nil
	}
}

func lowerObjectType(def *ast.ObjectTypeDef, ctx *LoweringContext) *ObjectType {
	var status Status
	if def.Status != nil {
		status = lowerStatus(def.Status.Value)
	} else {
		status = StatusCurrent
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
		defval = lowerDefVal(def.DefVal, ctx)
	}

	return &ObjectType{
		Name:          def.Name.Name,
		Syntax:        lowerTypeSyntax(def.Syntax.Syntax, ctx),
		Units:         units,
		Access:        lowerAccess(def.Access.Value),
		AccessKeyword: lowerAccessKeyword(def.Access.Keyword),
		Status:        status,
		Description:   description,
		Reference:     reference,
		Index:         lowerIndexClause(def.Index),
		Augments:      augments,
		DefVal:        defval,
		Oid:           lowerOidAssignment(def.OidAssignment, ctx),
		Span:          def.Span,
	}
}

func lowerModuleIdentity(def *ast.ModuleIdentityDef, ctx *LoweringContext) *ModuleIdentity {
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
		Oid:          lowerOidAssignment(def.OidAssignment, ctx),
		Span:         def.Span,
	}
}

// checkRevisionLastUpdated warns if LAST-UPDATED has no matching REVISION.
func checkRevisionLastUpdated(ctx *LoweringContext, moduleName string, mi *ModuleIdentity) {
	if mi.LastUpdated == "" {
		return
	}
	for _, r := range mi.Revisions {
		if r.Date == mi.LastUpdated {
			return
		}
	}
	ctx.emitDiagnostic("revision-last-updated", mib.SeverityMinor, moduleName,
		fmt.Sprintf("revision for LAST-UPDATED %s is missing", mi.LastUpdated))
}

func lowerObjectIdentity(def *ast.ObjectIdentityDef, ctx *LoweringContext) *ObjectIdentity {
	var reference string
	if def.Reference != nil {
		reference = def.Reference.Value
	}

	return &ObjectIdentity{
		Name:        def.Name.Name,
		Status:      lowerStatus(def.Status.Value),
		Description: def.Description.Value,
		Reference:   reference,
		Oid:         lowerOidAssignment(def.OidAssignment, ctx),
		Span:        def.Span,
	}
}

func lowerNotificationType(def *ast.NotificationTypeDef, ctx *LoweringContext) *Notification {
	var reference string
	if def.Reference != nil {
		reference = def.Reference.Value
	}

	oid := lowerOidAssignment(def.OidAssignment, ctx)

	return &Notification{
		Name:        def.Name.Name,
		Objects:     identNames(def.Objects),
		Status:      lowerStatus(def.Status.Value),
		Description: def.Description.Value,
		Reference:   reference,
		TrapInfo:    nil,
		Oid:         &oid,
		Span:        def.Span,
	}
}

func lowerTrapType(def *ast.TrapTypeDef) *Notification {
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
		Objects:     identNames(def.Variables),
		Status:      StatusCurrent, // TRAP-TYPE has no STATUS clause
		Description: description,
		Reference:   reference,
		TrapInfo: &TrapInfo{
			Enterprise: def.Enterprise.Name,
			TrapNumber: def.TrapNumber,
		},
		Oid:  nil, // derived from enterprise + trap number
		Span: def.Span,
	}
}

func lowerTextualConvention(def *ast.TextualConventionDef, ctx *LoweringContext) *TypeDef {
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
		Syntax:              lowerTypeSyntax(def.Syntax.Syntax, ctx),
		BaseType:            nil,
		DisplayHint:         displayHint,
		Status:              lowerStatus(def.Status.Value),
		Description:         def.Description.Value,
		Reference:           reference,
		IsTextualConvention: true,
		Span:                def.Span,
	}
}

func lowerTypeAssignment(def *ast.TypeAssignmentDef, ctx *LoweringContext) *TypeDef {
	return &TypeDef{
		Name:                def.Name.Name,
		Syntax:              lowerTypeSyntax(def.Syntax, ctx),
		BaseType:            nil,
		DisplayHint:         "",
		Status:              StatusCurrent,
		Description:         "",
		Reference:           "",
		IsTextualConvention: false,
		Span:                def.Span,
	}
}

func lowerValueAssignment(def *ast.ValueAssignmentDef, ctx *LoweringContext) *ValueAssignment {
	return &ValueAssignment{
		Name: def.Name.Name,
		Oid:  lowerOidAssignment(def.OidAssignment, ctx),
		Span: def.Span,
	}
}

func lowerObjectGroup(def *ast.ObjectGroupDef, ctx *LoweringContext) *ObjectGroup {
	var reference string
	if def.Reference != nil {
		reference = def.Reference.Value
	}

	return &ObjectGroup{
		Name:        def.Name.Name,
		Objects:     identNames(def.Objects),
		Status:      lowerStatus(def.Status.Value),
		Description: def.Description.Value,
		Reference:   reference,
		Oid:         lowerOidAssignment(def.OidAssignment, ctx),
		Span:        def.Span,
	}
}

func lowerNotificationGroup(def *ast.NotificationGroupDef, ctx *LoweringContext) *NotificationGroup {
	var reference string
	if def.Reference != nil {
		reference = def.Reference.Value
	}

	return &NotificationGroup{
		Name:          def.Name.Name,
		Notifications: identNames(def.Notifications),
		Status:        lowerStatus(def.Status.Value),
		Description:   def.Description.Value,
		Reference:     reference,
		Oid:           lowerOidAssignment(def.OidAssignment, ctx),
		Span:          def.Span,
	}
}

func lowerModuleCompliance(def *ast.ModuleComplianceDef, ctx *LoweringContext) *ModuleCompliance {
	modules := make([]ComplianceModule, len(def.Modules))
	for i, m := range def.Modules {
		modules[i] = lowerComplianceModule(m, ctx)
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
		Oid:         lowerOidAssignment(def.OidAssignment, ctx),
		Span:        def.Span,
	}
}

func lowerComplianceModule(m ast.ComplianceModule, ctx *LoweringContext) ComplianceModule {
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
			objects = append(objects, lowerComplianceObject(comp, ctx))
		}
	}

	var moduleName string
	if m.ModuleName != nil {
		moduleName = m.ModuleName.Name
	}

	return ComplianceModule{
		ModuleName:      moduleName,
		MandatoryGroups: identNames(m.MandatoryGroups),
		Groups:          groups,
		Objects:         objects,
	}
}

func lowerComplianceObject(o *ast.ComplianceObject, ctx *LoweringContext) ComplianceObject {
	var syntax TypeSyntax
	if o.Syntax != nil {
		syntax = lowerTypeSyntax(o.Syntax.Syntax, ctx)
	}

	var writeSyntax TypeSyntax
	if o.WriteSyntax != nil {
		writeSyntax = lowerTypeSyntax(o.WriteSyntax.Syntax, ctx)
	}

	var minAccess *Access
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

func lowerAgentCapabilities(def *ast.AgentCapabilitiesDef, ctx *LoweringContext) *AgentCapabilities {
	supports := make([]SupportsModule, len(def.Supports))
	for i, s := range def.Supports {
		supports[i] = lowerSupportsModule(s, ctx)
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
		Oid:            lowerOidAssignment(def.OidAssignment, ctx),
		Span:           def.Span,
	}
}

func lowerSupportsModule(s ast.SupportsModule, ctx *LoweringContext) SupportsModule {
	var objectVariations []ObjectVariation
	var notificationVariations []NotificationVariation

	for _, v := range s.Variations {
		switch variation := v.(type) {
		case *ast.ObjectVariation:
			objectVariations = append(objectVariations, lowerObjectVariation(variation, ctx))
		case *ast.NotificationVariation:
			notificationVariations = append(notificationVariations, lowerNotificationVariation(variation))
		}
	}

	return SupportsModule{
		ModuleName:             s.ModuleName.Name,
		Includes:               identNames(s.Includes),
		ObjectVariations:       objectVariations,
		NotificationVariations: notificationVariations,
	}
}

func lowerObjectVariation(v *ast.ObjectVariation, ctx *LoweringContext) ObjectVariation {
	var syntax TypeSyntax
	if v.Syntax != nil {
		syntax = lowerTypeSyntax(v.Syntax.Syntax, ctx)
	}

	var writeSyntax TypeSyntax
	if v.WriteSyntax != nil {
		writeSyntax = lowerTypeSyntax(v.WriteSyntax.Syntax, ctx)
	}

	var access *Access
	if v.Access != nil {
		a := lowerAccess(v.Access.Value)
		access = &a
	}

	var creationRequires []string
	if len(v.CreationRequires) > 0 {
		creationRequires = identNames(v.CreationRequires)
	}

	var defval DefVal
	if v.DefVal != nil {
		defval = lowerDefVal(v.DefVal, ctx)
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
	var access *Access
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

func lowerTypeSyntax(syntax ast.TypeSyntax, ctx *LoweringContext) TypeSyntax {
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
			Base:       lowerTypeSyntax(s.Base, ctx),
			Constraint: lowerConstraint(s.Constraint, ctx),
		}

	case *ast.TypeSyntaxSequenceOf:
		return &TypeSyntaxSequenceOf{EntryType: s.EntryType.Name}

	case *ast.TypeSyntaxSequence:
		fields := make([]SequenceField, len(s.Fields))
		for i, f := range s.Fields {
			fields[i] = NewSequenceField(f.Name.Name, lowerTypeSyntax(f.Syntax, ctx))
		}
		return &TypeSyntaxSequence{Fields: fields}

	case *ast.TypeSyntaxChoice:
		// CHOICE only appears in SMI base modules; normalize to the first alternative.
		if len(s.Alternatives) > 0 {
			return lowerTypeSyntax(s.Alternatives[0].Syntax, ctx)
		}
		// Empty CHOICE fallback
		return &TypeSyntaxOctetString{}

	case *ast.TypeSyntaxOctetString:
		return &TypeSyntaxOctetString{}

	case *ast.TypeSyntaxObjectIdentifier:
		return &TypeSyntaxObjectIdentifier{}

	default:
		ctx.Log(slog.LevelWarn, "unknown type syntax in lowering, defaulting to OCTET STRING",
			slog.String("type", fmt.Sprintf("%T", syntax)))
		return &TypeSyntaxOctetString{}
	}
}

func lowerConstraint(constraint ast.Constraint, ctx *LoweringContext) Constraint {
	switch c := constraint.(type) {
	case *ast.ConstraintSize:
		ranges := make([]Range, len(c.Ranges))
		for i, r := range c.Ranges {
			ranges[i] = lowerRange(r, ctx)
		}
		return &ConstraintSize{Ranges: ranges}

	case *ast.ConstraintRange:
		ranges := make([]Range, len(c.Ranges))
		for i, r := range c.Ranges {
			ranges[i] = lowerRange(r, ctx)
		}
		return &ConstraintRange{Ranges: ranges}

	default:
		ctx.Log(slog.LevelWarn, "unknown constraint type in lowering, defaulting to empty range",
			slog.String("type", fmt.Sprintf("%T", constraint)))
		return &ConstraintRange{}
	}
}

func lowerRange(r ast.Range, ctx *LoweringContext) Range {
	return Range{
		Min: lowerRangeValue(r.Min, ctx),
		Max: lowerRangeValue(r.Max, ctx),
	}
}

func lowerRangeValue(value ast.RangeValue, ctx *LoweringContext) RangeValue {
	if value == nil {
		return nil
	}

	switch v := value.(type) {
	case *ast.RangeValueSigned:
		return &RangeValueSigned{Value: v.Value}

	case *ast.RangeValueUnsigned:
		return &RangeValueUnsigned{Value: v.Value}

	case *ast.RangeValueIdent:
		switch v.Name.Name {
		case "MIN":
			return &RangeValueMin{}
		case "MAX":
			return &RangeValueMax{}
		default:
			ctx.Log(slog.LevelWarn, "unknown range identifier, defaulting to 0",
				slog.String("name", v.Name.Name))
			return &RangeValueUnsigned{Value: 0}
		}

	default:
		ctx.Log(slog.LevelWarn, "unknown range value type in lowering, defaulting to 0",
			slog.String("type", fmt.Sprintf("%T", value)))
		return &RangeValueUnsigned{Value: 0}
	}
}

func lowerOidAssignment(oid ast.OidAssignment, ctx *LoweringContext) OidAssignment {
	components := make([]OidComponent, len(oid.Components))
	for i, c := range oid.Components {
		components[i] = lowerOidComponent(c, ctx)
	}
	return NewOidAssignment(components, oid.Span)
}

func lowerOidComponent(comp ast.OidComponent, ctx *LoweringContext) OidComponent {
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
		ctx.Log(slog.LevelWarn, "unknown OID component type in lowering, defaulting to sub-id 0",
			slog.String("type", fmt.Sprintf("%T", comp)))
		return &OidComponentNumber{Value: 0}
	}
}

// lowerAccess converts AST access values without normalization.
func lowerAccess(access ast.AccessValue) Access {
	switch access {
	case ast.AccessValueReadOnly:
		return AccessReadOnly
	case ast.AccessValueReadWrite:
		return AccessReadWrite
	case ast.AccessValueReadCreate:
		return AccessReadCreate
	case ast.AccessValueNotAccessible:
		return AccessNotAccessible
	case ast.AccessValueAccessibleForNotify:
		return AccessAccessibleForNotify
	case ast.AccessValueWriteOnly:
		return AccessWriteOnly
	case ast.AccessValueInstall:
		return AccessInstall
	case ast.AccessValueInstallNotify:
		return AccessInstallNotify
	case ast.AccessValueReportOnly:
		return AccessReportOnly
	case ast.AccessValueNotImplemented:
		return AccessNotImplemented
	default:
		return AccessNotAccessible
	}
}

func lowerAccessKeyword(keyword ast.AccessKeyword) AccessKeyword {
	switch keyword {
	case ast.AccessKeywordAccess:
		return AccessKeywordAccess
	case ast.AccessKeywordMaxAccess:
		return AccessKeywordMaxAccess
	case ast.AccessKeywordMinAccess:
		return AccessKeywordMinAccess
	case ast.AccessKeywordPibAccess:
		return AccessKeywordPibAccess
	default:
		return AccessKeywordAccess
	}
}

// lowerStatus converts AST status values without normalization. SMIv1
// mandatory/optional are kept distinct from SMIv2 current/deprecated.
func lowerStatus(status ast.StatusValue) Status {
	switch status {
	case ast.StatusValueCurrent:
		return StatusCurrent
	case ast.StatusValueDeprecated:
		return StatusDeprecated
	case ast.StatusValueObsolete:
		return StatusObsolete
	case ast.StatusValueMandatory:
		return StatusMandatory
	case ast.StatusValueOptional:
		return StatusOptional
	default:
		return StatusCurrent
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

func lowerDefVal(clause *ast.DefValClause, ctx *LoweringContext) DefVal {
	return lowerDefValContent(clause.Value, ctx)
}

func lowerDefValContent(content ast.DefValContent, ctx *LoweringContext) DefVal {
	switch c := content.(type) {
	case *ast.DefValContentInteger:
		return &DefValInteger{Value: c.Value}

	case *ast.DefValContentUnsigned:
		return &DefValUnsigned{Value: c.Value}

	case *ast.DefValContentString:
		return &DefValString{Value: c.Value.Value}

	case *ast.DefValContentIdentifier:
		// Could be enum label or OID reference; treat as enum until semantic analysis.
		return &DefValEnum{Name: c.Name.Name}

	case *ast.DefValContentBits:
		return &DefValBits{Labels: identNames(c.Labels)}

	case *ast.DefValContentUnparsed:
		return &DefValUnparsed{}

	case *ast.DefValContentHexString:
		return &DefValHexString{Value: c.Content}

	case *ast.DefValContentBinaryString:
		return &DefValBinaryString{Value: c.Content}

	case *ast.DefValContentObjectIdentifier:
		components := make([]OidComponent, len(c.Components))
		for i, comp := range c.Components {
			components[i] = lowerOidComponent(comp, ctx)
		}
		return &DefValOidValue{Components: components}

	default:
		ctx.Log(slog.LevelWarn, "unknown DEFVAL content type in lowering, defaulting to integer 0",
			slog.String("type", fmt.Sprintf("%T", content)))
		return &DefValInteger{Value: 0}
	}
}
