package module

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/golangsnmp/gomib/internal/ast"
	"github.com/golangsnmp/gomib/internal/types"
)

// LoweringContext tracks state accumulated during the lowering pass.
type LoweringContext struct {
	Diagnostics []types.Diagnostic
	Language    types.Language
	DiagConfig  types.DiagnosticConfig
	source      []byte // source text for span-to-line/column conversion
	lineTable   []int  // precomputed line table for O(log n) span-to-line/col
	moduleName  string // current module name for diagnostics
	types.Logger
}

// newLoweringContext returns a LoweringContext. A nil logger disables logging.
func newLoweringContext(source []byte, logger *slog.Logger, diagConfig types.DiagnosticConfig) *LoweringContext {
	return &LoweringContext{
		Language:   types.LanguageUnknown,
		DiagConfig: diagConfig,
		source:     source,
		lineTable:  types.BuildLineTable(source),
		Logger:     types.Logger{L: logger},
	}
}

// LineColFromLineTable converts a span to line/col using a module's precomputed
// line table. This is the public entry point for code outside the lowering pass
// (e.g., the resolver) that no longer has access to source bytes.
func LineColFromLineTable(lineTable []int, span types.Span) (line, col int) {
	return types.LineColFromTable(lineTable, span.Start)
}

// emitDiagnostic records a diagnostic if the current config allows it.
// The span is converted to line/column using the precomputed line table.
func (ctx *LoweringContext) emitDiagnostic(code string, span types.Span, message string) {
	sev := types.SeverityForCode(code)
	if !ctx.DiagConfig.ShouldReport(code, sev) {
		return
	}
	line, col := types.LineColFromTable(ctx.lineTable, span.Start)
	ctx.Diagnostics = append(ctx.Diagnostics, types.Diagnostic{
		Severity: sev,
		Code:     code,
		Message:  message,
		Module:   ctx.moduleName,
		Line:     line,
		Column:   col,
	})
}

// isSMIv2Import reports whether importing from this module indicates SMIv2.
func isSMIv2Import(name string) bool {
	bm, ok := BaseModuleFromName(name)
	return ok && bm.IsSMIv2()
}

// Lower transforms an AST module into a normalized Module. The AST is not
// needed after lowering. Source is the original source text used to compute
// diagnostic line/column from byte offset spans. A nil logger disables logging.
func Lower(astModule *ast.Module, source []byte, logger *slog.Logger, diagConfig types.DiagnosticConfig) *Module {
	ctx := newLoweringContext(source, logger, diagConfig)

	module := NewModule(astModule.Name.Name, astModule.Span)
	module.LineTable = ctx.lineTable
	ctx.moduleName = module.Name

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

	// Cache LAST-UPDATED for efficient access during resolution.
	for _, def := range module.Definitions {
		if mi, ok := def.(*ModuleIdentity); ok && strings.TrimSpace(mi.LastUpdated) != "" {
			module.LastUpdated = mi.LastUpdated
			break
		}
	}

	ctx.Log(slog.LevelDebug, "lowering complete",
		slog.String("module", module.Name),
		slog.Int("definitions", len(module.Definitions)))

	checkModuleNameSuffix(ctx, module)

	if module.Language == types.LanguageSMIv2 && !IsBaseModule(module.Name) {
		var moduleIdentities []*ModuleIdentity
		firstIndex := -1
		for i, def := range module.Definitions {
			if mi, ok := def.(*ModuleIdentity); ok {
				if firstIndex == -1 {
					firstIndex = i
				}
				moduleIdentities = append(moduleIdentities, mi)
			}
		}

		if len(moduleIdentities) == 0 {
			ctx.emitDiagnostic(types.DiagMissingModuleIdentity, module.Span,
				fmt.Sprintf("SMIv2 module %s lacks MODULE-IDENTITY", module.Name))
		} else {
			if firstIndex > 0 {
				ctx.emitDiagnostic(types.DiagModuleIdentityNotFirst, moduleIdentities[0].Span,
					"MODULE-IDENTITY should be the first definition in "+module.Name)
			}

			for _, mi := range moduleIdentities[1:] {
				ctx.emitDiagnostic(types.DiagModuleIdentityMultiple, mi.Span,
					"multiple MODULE-IDENTITY definitions in "+module.Name)
			}
		}

		checkMacroImports(ctx, module)
	}

	for _, d := range astModule.Diagnostics {
		if !ctx.DiagConfig.ShouldReport(d.Code, d.Severity) {
			continue
		}
		line, col := types.LineColFromTable(ctx.lineTable, d.Span.Start)
		module.Diagnostics = append(module.Diagnostics, types.Diagnostic{
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
	if len(idents) == 0 {
		return nil
	}
	names := make([]string, len(idents))
	for i, id := range idents {
		names[i] = id.Name
	}
	return names
}

// optionalString returns qs.Value if qs is non-nil, or "" otherwise.
func optionalString(qs *ast.QuotedString) string {
	if qs != nil {
		return qs.Value
	}
	return ""
}

// optionalAccess returns a pointer to a.Value if a is non-nil, or nil otherwise.
func optionalAccess(a *ast.AccessClause) *types.Access {
	if a != nil {
		v := a.Value
		return &v
	}
	return nil
}

// optionalStatus returns s.Value if s is non-nil, or defaultStatus otherwise.
func optionalStatus(s *ast.StatusClause, defaultStatus types.Status) types.Status {
	if s != nil {
		return s.Value
	}
	return defaultStatus
}

// optionalQuotedStringSpan returns qs.Span if qs is non-nil, or zero otherwise.
func optionalQuotedStringSpan(qs *ast.QuotedString) types.Span {
	if qs != nil {
		return qs.Span
	}
	return types.Span{}
}

// optionalStatusSpan returns s.Span if s is non-nil, or zero otherwise.
func optionalStatusSpan(s *ast.StatusClause) types.Span {
	if s != nil {
		return s.Span
	}
	return types.Span{}
}

// optionalIndexClauseSpan returns the IndexClauseSpan if clause is non-nil.
func optionalIndexClauseSpan(clause ast.IndexClause) types.Span {
	if clause != nil {
		return clause.IndexClauseSpan()
	}
	return types.Span{}
}

// optionalAugmentsSpan returns a.Span if a is non-nil, or zero otherwise.
func optionalAugmentsSpan(a *ast.AugmentsClause) types.Span {
	if a != nil {
		return a.Span
	}
	return types.Span{}
}

// optionalDefValSpan returns d.Span if d is non-nil, or zero otherwise.
func optionalDefValSpan(d *ast.DefValClause) types.Span {
	if d != nil {
		return d.Span
	}
	return types.Span{}
}

// checkEmptyOptionalString emits a diagnostic when an optional quoted string
// clause is present but empty.
func checkEmptyOptionalString(ctx *LoweringContext, qs *ast.QuotedString, span types.Span, defName, clause, code string) {
	if qs != nil && qs.Value == "" {
		ctx.emitDiagnostic(code, span,
			fmt.Sprintf("%q: empty %s clause", defName, clause))
	}
}

// checkEmptyRequiredString emits a diagnostic when a required string clause
// has an empty value.
func checkEmptyRequiredString(ctx *LoweringContext, value string, span types.Span, defName, clause, code string) {
	if value == "" {
		ctx.emitDiagnostic(code, span,
			fmt.Sprintf("%q: empty %s clause", defName, clause))
	}
}

// checkModuleNameSuffix warns when an SMIv2 module name does not end with -MIB.
func checkModuleNameSuffix(ctx *LoweringContext, mod *Module) {
	if mod.Language != types.LanguageSMIv2 || IsBaseModule(mod.Name) {
		return
	}
	if !strings.HasSuffix(mod.Name, "-MIB") {
		ctx.emitDiagnostic(types.DiagModuleNameSuffix, mod.Span,
			fmt.Sprintf("module name %q does not end with -MIB", mod.Name))
	}
}

// lowerImports flattens import clauses and detects the SMI language.
func lowerImports(importClauses []ast.ImportClause, ctx *LoweringContext) []Import {
	var imports []Import

	for _, clause := range importClauses {
		fromModule := clause.FromModule.Name

		if isSMIv2Import(fromModule) {
			ctx.Language = types.LanguageSMIv2
		}

		for _, symbol := range clause.Symbols {
			imports = append(imports, NewImport(fromModule, symbol.Name, symbol.Span))
		}
	}

	if ctx.Language == types.LanguageUnknown {
		ctx.Language = types.LanguageSMIv1
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
		return lowerTrapType(d, ctx)
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
	case *ast.MacroDefinitionDef:
		if !IsBaseModule(ctx.moduleName) {
			ctx.emitDiagnostic(types.DiagMacroNotAllowed, def.DefinitionSpan(),
				fmt.Sprintf("MACRO definition %q not allowed outside base modules", def.DefinitionName()))
		}
		return nil
	case *ast.ErrorDef:
		return nil
	default:
		ctx.emitDiagnostic(types.DiagUnknownDefinitionType, def.DefinitionSpan(),
			fmt.Sprintf("unknown definition type %T", def))
		return nil
	}
}

func lowerObjectType(def *ast.ObjectTypeDef, ctx *LoweringContext) *ObjectType {
	var augments string
	if def.Augments != nil {
		augments = def.Augments.Target.Name
	}

	name := def.Name.Name
	checkEmptyOptionalString(ctx, def.Description, def.Span, name, "DESCRIPTION", types.DiagEmptyDescription)
	checkEmptyOptionalString(ctx, def.Reference, def.Span, name, "REFERENCE", types.DiagEmptyReference)
	checkEmptyOptionalString(ctx, def.Units, def.Span, name, "UNITS", types.DiagEmptyUnits)

	return &ObjectType{
		DefBase:        DefBase{Name: def.Name.Name, Span: def.Span},
		Syntax:         lowerTypeSyntax(def.Syntax.Syntax, ctx),
		Units:          optionalString(def.Units),
		Access:         def.Access.Value,
		AccessKeyword:  def.Access.Keyword,
		Status:         optionalStatus(def.Status, types.StatusCurrent),
		Description:    optionalString(def.Description),
		HasDescription: def.Description != nil,
		Reference:      optionalString(def.Reference),
		Index:          lowerIndexClause(def.Index),
		Augments:       augments,
		DefVal:         lowerOptionalDefVal(def.DefVal, ctx),
		Oid:            lowerOidAssignment(def.OidAssignment, ctx),
		Spans: ObjectTypeSpans{
			Syntax:      def.Syntax.Span,
			Access:      def.Access.Span,
			Status:      optionalStatusSpan(def.Status),
			Description: optionalQuotedStringSpan(def.Description),
			Units:       optionalQuotedStringSpan(def.Units),
			Reference:   optionalQuotedStringSpan(def.Reference),
			Index:       optionalIndexClauseSpan(def.Index),
			Augments:    optionalAugmentsSpan(def.Augments),
			DefVal:      optionalDefValSpan(def.DefVal),
		},
	}
}

func lowerModuleIdentity(def *ast.ModuleIdentityDef, ctx *LoweringContext) *ModuleIdentity {
	revisions := make([]Revision, len(def.Revisions))
	for i, r := range def.Revisions {
		revisions[i] = Revision{
			Date:        r.Date.Value,
			Description: r.Description.Value,
			Span:        r.Span,
		}
		checkEmptyRequiredString(ctx, r.Description.Value, r.Span, def.Name.Name, "REVISION DESCRIPTION", types.DiagEmptyDescription)
	}

	name := def.Name.Name
	checkEmptyRequiredString(ctx, def.Description.Value, def.Span, name, "DESCRIPTION", types.DiagEmptyDescription)
	checkEmptyRequiredString(ctx, def.Organization.Value, def.Span, name, "ORGANIZATION", types.DiagEmptyOrganization)
	checkEmptyRequiredString(ctx, def.ContactInfo.Value, def.Span, name, "CONTACT-INFO", types.DiagEmptyContact)

	mi := &ModuleIdentity{
		DefBase:      DefBase{Name: def.Name.Name, Span: def.Span},
		LastUpdated:  def.LastUpdated.Value,
		Organization: def.Organization.Value,
		ContactInfo:  def.ContactInfo.Value,
		Description:  def.Description.Value,
		Revisions:    revisions,
		Oid:          lowerOidAssignment(def.OidAssignment, ctx),
	}

	checkModuleIdentityDates(ctx, def)
	checkRevisionLastUpdated(ctx, mi)

	return mi
}

// checkRevisionLastUpdated warns if LAST-UPDATED has no matching REVISION.
func checkRevisionLastUpdated(ctx *LoweringContext, mi *ModuleIdentity) {
	if mi.LastUpdated == "" {
		return
	}
	for _, r := range mi.Revisions {
		if r.Date == mi.LastUpdated {
			return
		}
	}
	ctx.emitDiagnostic(types.DiagRevisionLastUpdated, mi.Span,
		fmt.Sprintf("revision for LAST-UPDATED %s is missing", mi.LastUpdated))
}

// checkMacroImports warns when SMIv2 macro keywords are used without being imported.
func checkMacroImports(ctx *LoweringContext, mod *Module) {
	// Collect imported macro names.
	importedMacros := make(map[string]struct{})
	for _, imp := range mod.Imports {
		switch imp.Symbol {
		case "MODULE-IDENTITY", "OBJECT-IDENTITY", "OBJECT-TYPE",
			"NOTIFICATION-TYPE", "TEXTUAL-CONVENTION", "OBJECT-GROUP",
			"NOTIFICATION-GROUP", "MODULE-COMPLIANCE", "AGENT-CAPABILITIES":
			importedMacros[imp.Symbol] = struct{}{}
		}
	}

	// Determine which macros are used by scanning definitions.
	usedMacros := make(map[string]struct{})
	for _, def := range mod.Definitions {
		switch d := def.(type) {
		case *ModuleIdentity:
			usedMacros["MODULE-IDENTITY"] = struct{}{}
		case *ObjectIdentity:
			usedMacros["OBJECT-IDENTITY"] = struct{}{}
		case *ObjectType:
			usedMacros["OBJECT-TYPE"] = struct{}{}
		case *Notification:
			// Could be NOTIFICATION-TYPE or TRAP-TYPE; only NOTIFICATION-TYPE
			// needs an import in SMIv2.
			if d.TrapInfo == nil {
				usedMacros["NOTIFICATION-TYPE"] = struct{}{}
			}
		case *TypeDef:
			if d.IsTextualConvention {
				usedMacros["TEXTUAL-CONVENTION"] = struct{}{}
			}
		case *ObjectGroup:
			usedMacros["OBJECT-GROUP"] = struct{}{}
		case *NotificationGroup:
			usedMacros["NOTIFICATION-GROUP"] = struct{}{}
		case *ModuleCompliance:
			usedMacros["MODULE-COMPLIANCE"] = struct{}{}
		case *AgentCapabilities:
			usedMacros["AGENT-CAPABILITIES"] = struct{}{}
		}
	}

	for macro := range usedMacros {
		if _, ok := importedMacros[macro]; !ok {
			ctx.emitDiagnostic(types.DiagMacroNotImported, mod.Span,
				fmt.Sprintf("%s used but not imported in %s", macro, mod.Name))
		}
	}
}

func lowerObjectIdentity(def *ast.ObjectIdentityDef, ctx *LoweringContext) *ObjectIdentity {
	name := def.Name.Name
	checkEmptyRequiredString(ctx, def.Description.Value, def.Span, name, "DESCRIPTION", types.DiagEmptyDescription)
	checkEmptyOptionalString(ctx, def.Reference, def.Span, name, "REFERENCE", types.DiagEmptyReference)

	return &ObjectIdentity{
		DefBase:     DefBase{Name: def.Name.Name, Span: def.Span},
		Status:      def.Status.Value,
		Description: def.Description.Value,
		Reference:   optionalString(def.Reference),
		Oid:         lowerOidAssignment(def.OidAssignment, ctx),
	}
}

func lowerNotificationType(def *ast.NotificationTypeDef, ctx *LoweringContext) *Notification {
	oid := lowerOidAssignment(def.OidAssignment, ctx)
	name := def.Name.Name

	checkEmptyRequiredString(ctx, def.Description.Value, def.Span, name, "DESCRIPTION", types.DiagEmptyDescription)
	checkEmptyOptionalString(ctx, def.Reference, def.Span, name, "REFERENCE", types.DiagEmptyReference)

	return &Notification{
		DefBase:        DefBase{Name: def.Name.Name, Span: def.Span},
		Objects:        identNames(def.Objects),
		Status:         def.Status.Value,
		Description:    def.Description.Value,
		HasDescription: true, // NOTIFICATION-TYPE DESCRIPTION is required
		Reference:      optionalString(def.Reference),
		Oid:            &oid,
	}
}

func lowerTrapType(def *ast.TrapTypeDef, ctx *LoweringContext) *Notification {
	name := def.Name.Name
	checkEmptyOptionalString(ctx, def.Description, def.Span, name, "DESCRIPTION", types.DiagEmptyDescription)
	checkEmptyOptionalString(ctx, def.Reference, def.Span, name, "REFERENCE", types.DiagEmptyReference)

	return &Notification{
		DefBase:        DefBase{Name: def.Name.Name, Span: def.Span},
		Objects:        identNames(def.Variables),
		Status:         types.StatusCurrent, // TRAP-TYPE has no STATUS clause
		Description:    optionalString(def.Description),
		HasDescription: def.Description != nil,
		Reference:      optionalString(def.Reference),
		TrapInfo: &TrapInfo{
			Enterprise: def.Enterprise.Name,
			TrapNumber: def.TrapNumber,
		},
	}
}

func lowerTextualConvention(def *ast.TextualConventionDef, ctx *LoweringContext) *TypeDef {
	name := def.Name.Name
	checkEmptyRequiredString(ctx, def.Description.Value, def.Span, name, "DESCRIPTION", types.DiagEmptyDescription)
	checkEmptyOptionalString(ctx, def.Reference, def.Span, name, "REFERENCE", types.DiagEmptyReference)
	checkEmptyOptionalString(ctx, def.DisplayHint, def.Span, name, "DISPLAY-HINT", types.DiagEmptyFormat)

	return &TypeDef{
		DefBase:             DefBase{Name: def.Name.Name, Span: def.Span},
		Syntax:              lowerTypeSyntax(def.Syntax.Syntax, ctx),
		DisplayHint:         optionalString(def.DisplayHint),
		Status:              def.Status.Value,
		Description:         def.Description.Value,
		Reference:           optionalString(def.Reference),
		IsTextualConvention: true,
		Spans: TypeDefSpans{
			Syntax:      def.Syntax.Span,
			Status:      def.Status.Span,
			Description: def.Description.Span,
			Reference:   optionalQuotedStringSpan(def.Reference),
			DisplayHint: optionalQuotedStringSpan(def.DisplayHint),
		},
	}
}

func lowerTypeAssignment(def *ast.TypeAssignmentDef, ctx *LoweringContext) *TypeDef {
	return &TypeDef{
		DefBase: DefBase{Name: def.Name.Name, Span: def.Span},
		Syntax:  lowerTypeSyntax(def.Syntax, ctx),
		Status:  types.StatusCurrent,
		Spans:   TypeDefSpans{Syntax: def.Syntax.SyntaxSpan()},
	}
}

func lowerValueAssignment(def *ast.ValueAssignmentDef, ctx *LoweringContext) *ValueAssignment {
	return &ValueAssignment{
		DefBase: DefBase{Name: def.Name.Name, Span: def.Span},
		Oid:     lowerOidAssignment(def.OidAssignment, ctx),
	}
}

func lowerObjectGroup(def *ast.ObjectGroupDef, ctx *LoweringContext) *ObjectGroup {
	name := def.Name.Name
	checkEmptyRequiredString(ctx, def.Description.Value, def.Span, name, "DESCRIPTION", types.DiagEmptyDescription)
	checkEmptyOptionalString(ctx, def.Reference, def.Span, name, "REFERENCE", types.DiagEmptyReference)

	return &ObjectGroup{
		DefBase:     DefBase{Name: def.Name.Name, Span: def.Span},
		Objects:     identNames(def.Objects),
		Status:      def.Status.Value,
		Description: def.Description.Value,
		Reference:   optionalString(def.Reference),
		Oid:         lowerOidAssignment(def.OidAssignment, ctx),
	}
}

func lowerNotificationGroup(def *ast.NotificationGroupDef, ctx *LoweringContext) *NotificationGroup {
	name := def.Name.Name
	checkEmptyRequiredString(ctx, def.Description.Value, def.Span, name, "DESCRIPTION", types.DiagEmptyDescription)
	checkEmptyOptionalString(ctx, def.Reference, def.Span, name, "REFERENCE", types.DiagEmptyReference)

	return &NotificationGroup{
		DefBase:       DefBase{Name: def.Name.Name, Span: def.Span},
		Notifications: identNames(def.Notifications),
		Status:        def.Status.Value,
		Description:   def.Description.Value,
		Reference:     optionalString(def.Reference),
		Oid:           lowerOidAssignment(def.OidAssignment, ctx),
	}
}

func lowerModuleCompliance(def *ast.ModuleComplianceDef, ctx *LoweringContext) *ModuleCompliance {
	modules := make([]ComplianceModule, len(def.Modules))
	for i, m := range def.Modules {
		modules[i] = lowerComplianceModule(m, ctx)
	}

	name := def.Name.Name
	checkEmptyRequiredString(ctx, def.Description.Value, def.Span, name, "DESCRIPTION", types.DiagEmptyDescription)
	checkEmptyOptionalString(ctx, def.Reference, def.Span, name, "REFERENCE", types.DiagEmptyReference)

	return &ModuleCompliance{
		DefBase:     DefBase{Name: def.Name.Name, Span: def.Span},
		Status:      def.Status.Value,
		Description: def.Description.Value,
		Reference:   optionalString(def.Reference),
		Modules:     modules,
		Oid:         lowerOidAssignment(def.OidAssignment, ctx),
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
				Span:        comp.Span,
			})
		case *ast.ComplianceObject:
			objects = append(objects, lowerComplianceObject(comp, ctx))
		}
	}

	var moduleName string
	if m.ModuleName != nil {
		moduleName = m.ModuleName.Name
	}

	// m.ModuleOid is intentionally not preserved. The optional OID after the
	// module name (RFC 2580 ModuleIdentifier) is a disambiguation mechanism
	// that no real-world MIBs use, and neither the resolver nor public model
	// consume it.
	return ComplianceModule{
		ModuleName:      moduleName,
		MandatoryGroups: identNames(m.MandatoryGroups),
		Groups:          groups,
		Objects:         objects,
		Span:            m.Span,
	}
}

func lowerOptionalSyntax(clause *ast.SyntaxClause, ctx *LoweringContext) TypeSyntax {
	if clause == nil {
		return nil
	}
	return lowerTypeSyntax(clause.Syntax, ctx)
}

func lowerOptionalDefVal(clause *ast.DefValClause, ctx *LoweringContext) DefVal {
	if clause == nil {
		return nil
	}
	return lowerDefValValue(clause.Value, ctx)
}

func lowerComplianceObject(o *ast.ComplianceObject, ctx *LoweringContext) ComplianceObject {
	return ComplianceObject{
		Object:      o.Object.Name,
		Syntax:      lowerOptionalSyntax(o.Syntax, ctx),
		WriteSyntax: lowerOptionalSyntax(o.WriteSyntax, ctx),
		MinAccess:   optionalAccess(o.MinAccess),
		Description: o.Description.Value,
		Span:        o.Span,
	}
}

func lowerAgentCapabilities(def *ast.AgentCapabilitiesDef, ctx *LoweringContext) *AgentCapabilities {
	supports := make([]SupportsModule, len(def.Supports))
	for i := range def.Supports {
		supports[i] = lowerSupportsModule(&def.Supports[i], ctx)
	}

	name := def.Name.Name
	checkEmptyRequiredString(ctx, def.Description.Value, def.Span, name, "DESCRIPTION", types.DiagEmptyDescription)
	checkEmptyOptionalString(ctx, def.Reference, def.Span, name, "REFERENCE", types.DiagEmptyReference)

	return &AgentCapabilities{
		DefBase:        DefBase{Name: def.Name.Name, Span: def.Span},
		ProductRelease: def.ProductRelease.Value,
		Status:         def.Status.Value,
		Description:    def.Description.Value,
		Reference:      optionalString(def.Reference),
		Supports:       supports,
		Oid:            lowerOidAssignment(def.OidAssignment, ctx),
	}
}

func lowerSupportsModule(s *ast.SupportsModule, ctx *LoweringContext) SupportsModule {
	var variations []Variation
	for _, v := range s.Variations {
		variations = append(variations, lowerVariation(&v, ctx))
	}

	// s.ModuleOid is intentionally not preserved. See lowerComplianceModule.
	return SupportsModule{
		ModuleName: s.ModuleName.Name,
		Includes:   identNames(s.Includes),
		Variations: variations,
		Span:       s.Span,
	}
}

func lowerVariation(v *ast.Variation, ctx *LoweringContext) Variation {
	creationRequires := identNames(v.CreationRequires)

	return Variation{
		Name:             v.Name.Name,
		Syntax:           lowerOptionalSyntax(v.Syntax, ctx),
		WriteSyntax:      lowerOptionalSyntax(v.WriteSyntax, ctx),
		Access:           optionalAccess(v.Access),
		CreationRequires: creationRequires,
		DefVal:           lowerOptionalDefVal(v.DefVal, ctx),
		Description:      v.Description.Value,
		Span:             v.Span,
	}
}

func lowerTypeSyntax(syntax ast.TypeSyntax, ctx *LoweringContext) TypeSyntax {
	switch s := syntax.(type) {
	case *ast.TypeSyntaxTypeRef:
		return &TypeSyntaxTypeRef{Name: s.Name.Name, Span: s.Name.Span}

	case *ast.TypeSyntaxIntegerEnum:
		namedNumbers := make([]NamedNumber, len(s.NamedNumbers))
		seenNames := make(map[string]bool, len(s.NamedNumbers))
		seenValues := make(map[int64]bool, len(s.NamedNumbers))
		for i, nn := range s.NamedNumbers {
			if seenNames[nn.Name.Name] {
				ctx.emitDiagnostic(types.DiagEnumNameRedefinition, nn.Span,
					fmt.Sprintf("duplicate enum name %q", nn.Name.Name))
			}
			seenNames[nn.Name.Name] = true
			if seenValues[nn.Value] {
				ctx.emitDiagnostic(types.DiagEnumValueRedefinition, nn.Span,
					fmt.Sprintf("duplicate enum value %d", nn.Value))
			}
			seenValues[nn.Value] = true
			if nn.Value == 0 && ctx.Language == types.LanguageSMIv1 {
				ctx.emitDiagnostic(types.DiagEnumZero, nn.Span,
					fmt.Sprintf("enumeration contains zero value %q(0) in SMIv1 module", nn.Name.Name))
			}
			namedNumbers[i] = NewNamedNumber(nn.Name.Name, nn.Value, nn.Span)
		}
		var base string
		if s.Base != nil {
			base = s.Base.Name
		}
		return &TypeSyntaxIntegerEnum{Base: base, NamedNumbers: namedNumbers, Span: s.SyntaxSpan()}

	case *ast.TypeSyntaxBits:
		namedBits := make([]NamedBit, len(s.NamedBits))
		seenNames := make(map[string]bool, len(s.NamedBits))
		seenPositions := make(map[int64]bool, len(s.NamedBits))
		for i, nb := range s.NamedBits {
			if seenNames[nb.Name.Name] {
				ctx.emitDiagnostic(types.DiagBitsNameRedefinition, nb.Span,
					fmt.Sprintf("duplicate BITS name %q", nb.Name.Name))
			}
			seenNames[nb.Name.Name] = true
			if seenPositions[nb.Value] {
				ctx.emitDiagnostic(types.DiagBitsValueRedefinition, nb.Span,
					fmt.Sprintf("duplicate BITS position %d", nb.Value))
			}
			seenPositions[nb.Value] = true
			pos := nb.Value
			switch {
			case pos < 0:
				ctx.emitDiagnostic(types.DiagBitsNumberNegative, nb.Span,
					fmt.Sprintf("negative BITS position %d for %q", pos, nb.Name.Name))
				pos = 0
			case pos >= 65535*8:
				ctx.emitDiagnostic(types.DiagBitsNumberTooLarge, nb.Span,
					fmt.Sprintf("BITS position %d for %q exceeds maximum (%d)", pos, nb.Name.Name, 65535*8-1))
				pos = 0
			case pos >= 128:
				ctx.emitDiagnostic(types.DiagBitsNumberLarge, nb.Span,
					fmt.Sprintf("BITS position %d for %q may cause interoperability problems", pos, nb.Name.Name))
			}
			namedBits[i] = NewNamedBit(nb.Name.Name, uint32(pos), nb.Span)
		}
		return &TypeSyntaxBits{NamedBits: namedBits, Span: s.SyntaxSpan()}

	case *ast.TypeSyntaxConstrained:
		return &TypeSyntaxConstrained{
			Base:       lowerTypeSyntax(s.Base, ctx),
			Constraint: lowerConstraint(s.Constraint, ctx),
			Span:       s.SyntaxSpan(),
		}

	case *ast.TypeSyntaxSequenceOf:
		return &TypeSyntaxSequenceOf{EntryType: s.EntryType.Name, Span: s.SyntaxSpan()}

	case *ast.TypeSyntaxSequence:
		fields := make([]SequenceField, len(s.Fields))
		for i, f := range s.Fields {
			fields[i] = NewSequenceField(f.Name.Name, lowerTypeSyntax(f.Syntax, ctx), f.Span)
		}
		return &TypeSyntaxSequence{Fields: fields, Span: s.SyntaxSpan()}

	case *ast.TypeSyntaxChoice:
		// CHOICE only appears in SMI base modules; normalize to the first alternative.
		if !IsBaseModule(ctx.moduleName) {
			ctx.emitDiagnostic(types.DiagChoiceNotAllowed, syntax.SyntaxSpan(),
				fmt.Sprintf("CHOICE type not allowed outside base module %q", ctx.moduleName))
		}
		if len(s.Alternatives) > 0 {
			return lowerTypeSyntax(s.Alternatives[0].Syntax, ctx)
		}
		// Empty CHOICE fallback
		return &TypeSyntaxOctetString{}

	case *ast.TypeSyntaxTagged:
		// Tagged types only appear in SMI base modules; normalize to the underlying type.
		if !IsBaseModule(ctx.moduleName) {
			ctx.emitDiagnostic(types.DiagTaggedTypeNotAllowed, syntax.SyntaxSpan(),
				fmt.Sprintf("tagged type not allowed outside base module %q", ctx.moduleName))
		}
		return lowerTypeSyntax(s.Underlying, ctx)

	case *ast.TypeSyntaxOctetString:
		return &TypeSyntaxOctetString{Span: s.Span}

	case *ast.TypeSyntaxObjectIdentifier:
		return &TypeSyntaxObjectIdentifier{Span: s.Span}

	default:
		ctx.emitDiagnostic(types.DiagUnknownTypeSyntax, syntax.SyntaxSpan(),
			fmt.Sprintf("unknown type syntax %T, defaulting to OCTET STRING", syntax))
		return &TypeSyntaxOctetString{}
	}
}

func lowerConstraint(constraint ast.Constraint, ctx *LoweringContext) Constraint {
	switch c := constraint.(type) {
	case *ast.ConstraintSize:
		return &ConstraintSize{Ranges: lowerRanges(c.Ranges, ctx), Span: c.ConstraintSpan()}

	case *ast.ConstraintRange:
		return &ConstraintRange{Ranges: lowerRanges(c.Ranges, ctx), Span: c.ConstraintSpan()}

	default:
		ctx.emitDiagnostic(types.DiagUnknownConstraintType, constraint.ConstraintSpan(),
			fmt.Sprintf("unknown constraint type %T, defaulting to empty range", constraint))
		return &ConstraintRange{}
	}
}

func lowerRanges(astRanges []ast.Range, ctx *LoweringContext) []Range {
	ranges := make([]Range, len(astRanges))
	for i, r := range astRanges {
		ranges[i] = lowerRange(r, ctx)
	}
	return ranges
}

func lowerRange(r ast.Range, ctx *LoweringContext) Range {
	return Range{
		Min:  lowerRangeValue(r.Min, ctx),
		Max:  lowerRangeValue(r.Max, ctx),
		Span: r.Span,
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
			ctx.emitDiagnostic(types.DiagUnknownRangeValue, v.Name.Span,
				fmt.Sprintf("unknown range identifier %s, defaulting to 0", v.Name.Name))
			return &RangeValueUnsigned{Value: 0}
		}

	default:
		ctx.emitDiagnostic(types.DiagUnknownRangeValue, types.Span{},
			fmt.Sprintf("unknown range value type %T, defaulting to 0", value))
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
		return &OidComponentName{NameValue: c.Name.Name, Span: c.ComponentSpan()}

	case *ast.OidComponentNumber:
		return &OidComponentNumber{Value: c.Value, Span: c.ComponentSpan()}

	case *ast.OidComponentNamedNumber:
		return &OidComponentNamedNumber{
			NameValue:   c.Name.Name,
			NumberValue: c.Num,
			Span:        c.ComponentSpan(),
		}

	case *ast.OidComponentQualifiedName:
		return &OidComponentQualifiedName{
			ModuleValue: c.ModuleName.Name,
			NameValue:   c.Name.Name,
			Span:        c.ComponentSpan(),
		}

	case *ast.OidComponentQualifiedNamedNumber:
		return &OidComponentQualifiedNamedNumber{
			ModuleValue: c.ModuleName.Name,
			NameValue:   c.Name.Name,
			NumberValue: c.Num,
			Span:        c.ComponentSpan(),
		}

	default:
		ctx.emitDiagnostic(types.DiagUnknownOidComponent, comp.ComponentSpan(),
			fmt.Sprintf("unknown OID component type %T, defaulting to sub-id 0", comp))
		return &OidComponentNumber{Value: 0, Span: types.Span{}}
	}
}

func lowerIndexClause(clause ast.IndexClause) []IndexItem {
	if clause == nil {
		return nil
	}

	indexes := clause.Indexes()
	items := make([]IndexItem, len(indexes))
	for i, idx := range indexes {
		items[i] = IndexItem{Object: idx.Object.Name, Implied: idx.Implied, Span: idx.Span}
	}
	return items
}

func lowerDefValValue(content ast.DefVal, ctx *LoweringContext) DefVal {
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
		ctx.emitDiagnostic(types.DiagUnknownDefvalType, types.Span{},
			fmt.Sprintf("unknown DEFVAL content type %T, defaulting to integer 0", content))
		return &DefValInteger{Value: 0}
	}
}
