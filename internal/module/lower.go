package module

import (
	"fmt"
	"log/slog"
	"math"

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
func (ctx *LoweringContext) emitDiagnostic(code string, severity types.Severity, span types.Span, message string) {
	if !ctx.DiagConfig.ShouldReport(code, severity) {
		return
	}
	line, col := types.LineColFromTable(ctx.lineTable, span.Start)
	ctx.Diagnostics = append(ctx.Diagnostics, types.Diagnostic{
		Severity: severity,
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

	ctx.Log(slog.LevelDebug, "lowering complete",
		slog.String("module", module.Name),
		slog.Int("definitions", len(module.Definitions)))

	if module.Language == types.LanguageSMIv2 && !IsBaseModule(module.Name) {
		hasModuleIdentity := false
		for _, def := range module.Definitions {
			if mi, ok := def.(*ModuleIdentity); ok {
				hasModuleIdentity = true
				checkRevisionLastUpdated(ctx, mi)
				break
			}
		}
		if !hasModuleIdentity {
			ctx.emitDiagnostic(types.DiagMissingModuleIdentity, types.SeverityWarning, module.Span,
				fmt.Sprintf("SMIv2 module %s lacks MODULE-IDENTITY", module.Name))
		}
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

// lowerImports flattens import clauses and detects the SMI language.
func lowerImports(importClauses []ast.ImportClause, ctx *LoweringContext) []Import {
	var imports []Import

	for _, clause := range importClauses {
		fromModule := clause.FromModule.Name

		if isSMIv2Import(fromModule) {
			ctx.Language = types.LanguageSMIv2
		}

		for _, symbol := range clause.Symbols {
			imports = append(imports, NewImport(fromModule, symbol.Name, clause.Span))
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
	case *ast.MacroDefinitionDef, *ast.ErrorDef:
		// Non-semantic definitions
		return nil
	default:
		ctx.emitDiagnostic(types.DiagUnknownDefinitionType, types.SeverityWarning, def.DefinitionSpan(),
			fmt.Sprintf("unknown definition type %T", def))
		return nil
	}
}

func lowerObjectType(def *ast.ObjectTypeDef, ctx *LoweringContext) *ObjectType {
	var augments string
	if def.Augments != nil {
		augments = def.Augments.Target.Name
	}

	return &ObjectType{
		DefBase:       DefBase{Name: def.Name.Name, Span: def.Span},
		Syntax:        lowerTypeSyntax(def.Syntax.Syntax, ctx),
		Units:         optionalString(def.Units),
		Access:        def.Access.Value,
		AccessKeyword: def.Access.Keyword,
		Status:        optionalStatus(def.Status, types.StatusCurrent),
		Description:   optionalString(def.Description),
		Reference:     optionalString(def.Reference),
		Index:         lowerIndexClause(def.Index),
		Augments:      augments,
		DefVal:        lowerOptionalDefVal(def.DefVal, ctx),
		Oid:           lowerOidAssignment(def.OidAssignment, ctx),
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
		DefBase:      DefBase{Name: def.Name.Name, Span: def.Span},
		LastUpdated:  def.LastUpdated.Value,
		Organization: def.Organization.Value,
		ContactInfo:  def.ContactInfo.Value,
		Description:  def.Description.Value,
		Revisions:    revisions,
		Oid:          lowerOidAssignment(def.OidAssignment, ctx),
	}
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
	ctx.emitDiagnostic(types.DiagRevisionLastUpdated, types.SeverityMinor, mi.Span,
		fmt.Sprintf("revision for LAST-UPDATED %s is missing", mi.LastUpdated))
}

func lowerObjectIdentity(def *ast.ObjectIdentityDef, ctx *LoweringContext) *ObjectIdentity {
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

	return &Notification{
		DefBase:     DefBase{Name: def.Name.Name, Span: def.Span},
		Objects:     identNames(def.Objects),
		Status:      def.Status.Value,
		Description: def.Description.Value,
		Reference:   optionalString(def.Reference),
		Oid:         &oid,
	}
}

func lowerTrapType(def *ast.TrapTypeDef, _ *LoweringContext) *Notification {
	return &Notification{
		DefBase:     DefBase{Name: def.Name.Name, Span: def.Span},
		Objects:     identNames(def.Variables),
		Status:      types.StatusCurrent, // TRAP-TYPE has no STATUS clause
		Description: optionalString(def.Description),
		Reference:   optionalString(def.Reference),
		TrapInfo: &TrapInfo{
			Enterprise: def.Enterprise.Name,
			TrapNumber: def.TrapNumber,
		},
	}
}

func lowerTextualConvention(def *ast.TextualConventionDef, ctx *LoweringContext) *TypeDef {
	return &TypeDef{
		DefBase:             DefBase{Name: def.Name.Name, Span: def.Span},
		Syntax:              lowerTypeSyntax(def.Syntax.Syntax, ctx),
		DisplayHint:         optionalString(def.DisplayHint),
		Status:              def.Status.Value,
		Description:         def.Description.Value,
		Reference:           optionalString(def.Reference),
		IsTextualConvention: true,
	}
}

func lowerTypeAssignment(def *ast.TypeAssignmentDef, ctx *LoweringContext) *TypeDef {
	return &TypeDef{
		DefBase: DefBase{Name: def.Name.Name, Span: def.Span},
		Syntax:  lowerTypeSyntax(def.Syntax, ctx),
		Status:  types.StatusCurrent,
	}
}

func lowerValueAssignment(def *ast.ValueAssignmentDef, ctx *LoweringContext) *ValueAssignment {
	return &ValueAssignment{
		DefBase: DefBase{Name: def.Name.Name, Span: def.Span},
		Oid:     lowerOidAssignment(def.OidAssignment, ctx),
	}
}

func lowerObjectGroup(def *ast.ObjectGroupDef, ctx *LoweringContext) *ObjectGroup {
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
	}
}

func lowerAgentCapabilities(def *ast.AgentCapabilitiesDef, ctx *LoweringContext) *AgentCapabilities {
	supports := make([]SupportsModule, len(def.Supports))
	for i := range def.Supports {
		supports[i] = lowerSupportsModule(&def.Supports[i], ctx)
	}

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

	return SupportsModule{
		ModuleName: s.ModuleName.Name,
		Includes:   identNames(s.Includes),
		Variations: variations,
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
			pos := nb.Value
			if pos < 0 || pos > math.MaxUint32 {
				ctx.emitDiagnostic(types.DiagInvalidBitsPosition, types.SeverityWarning, nb.Span,
					fmt.Sprintf("invalid BITS position %d for %q, must be 0..%d", pos, nb.Name.Name, math.MaxUint32))
				pos = 0
			}
			namedBits[i] = NewNamedBit(nb.Name.Name, uint32(pos))
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
		ctx.emitDiagnostic(types.DiagUnknownTypeSyntax, types.SeverityWarning, syntax.SyntaxSpan(),
			fmt.Sprintf("unknown type syntax %T, defaulting to OCTET STRING", syntax))
		return &TypeSyntaxOctetString{}
	}
}

func lowerConstraint(constraint ast.Constraint, ctx *LoweringContext) Constraint {
	switch c := constraint.(type) {
	case *ast.ConstraintSize:
		return &ConstraintSize{Ranges: lowerRanges(c.Ranges, ctx)}

	case *ast.ConstraintRange:
		return &ConstraintRange{Ranges: lowerRanges(c.Ranges, ctx)}

	default:
		ctx.emitDiagnostic(types.DiagUnknownConstraintType, types.SeverityWarning, constraint.ConstraintSpan(),
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
			ctx.emitDiagnostic(types.DiagUnknownRangeValue, types.SeverityWarning, v.Name.Span,
				fmt.Sprintf("unknown range identifier %s, defaulting to 0", v.Name.Name))
			return &RangeValueUnsigned{Value: 0}
		}

	default:
		ctx.emitDiagnostic(types.DiagUnknownRangeValue, types.SeverityWarning, types.Span{},
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
		ctx.emitDiagnostic(types.DiagUnknownOidComponent, types.SeverityWarning, comp.ComponentSpan(),
			fmt.Sprintf("unknown OID component type %T, defaulting to sub-id 0", comp))
		return &OidComponentNumber{Value: 0}
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
		ctx.emitDiagnostic(types.DiagUnknownDefvalType, types.SeverityWarning, types.Span{},
			fmt.Sprintf("unknown DEFVAL content type %T, defaulting to integer 0", content))
		return &DefValInteger{Value: 0}
	}
}
