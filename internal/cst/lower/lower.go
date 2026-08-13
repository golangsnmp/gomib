// Package lower transforms CST nodes into module IR types.
//
// The lowered output matches the existing parser's output field by field.
// Diagnostics are passed through from the CST parser, not regenerated.
package lower

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/golangsnmp/gomib/internal/cst"
	"github.com/golangsnmp/gomib/internal/module"
	"github.com/golangsnmp/gomib/internal/types"
)

// Lower transforms a CST ModuleFile into module IR.
func Lower(file *cst.ModuleFile, source []byte, diags []types.SpanDiagnostic, diagConfig types.DiagnosticConfig) []*module.Module {
	l := &lowerer{
		source:      source,
		parserDiags: diags,
		diagConfig:  diagConfig,
	}

	var modules []*module.Module
	for i := range file.Modules {
		mod := l.lowerModule(&file.Modules[i])
		if mod.Name == "" {
			continue
		}
		modules = append(modules, mod)
	}
	return modules
}

type lowerer struct {
	source      []byte
	moduleName  string
	parserDiags []types.SpanDiagnostic
	diagConfig  types.DiagnosticConfig
	types.SpanDiagnosticCollector
}

// text extracts source text for a SyntaxToken.
func (l *lowerer) text(tok cst.SyntaxToken) string {
	return string(l.source[tok.Span.Start:tok.Span.End])
}

// stripQuotes strips surrounding double quotes from a quoted string token.
func (l *lowerer) stripQuotes(tok cst.SyntaxToken) string {
	fullText := l.text(tok)
	if len(fullText) >= 2 && fullText[len(fullText)-1] == '"' {
		return fullText[1 : len(fullText)-1]
	}
	if len(fullText) >= 1 {
		return fullText[1:]
	}
	return ""
}

// convertU32 parses a number token as uint32.
func (l *lowerer) convertU32(tok cst.SyntaxToken) uint32 {
	text := l.text(tok)
	v, err := strconv.ParseUint(text, 10, 32)
	if err != nil {
		return 0
	}
	return uint32(v)
}

// convertI64 parses a number token as int64.
func (l *lowerer) convertI64(tok cst.SyntaxToken) int64 {
	text := l.text(tok)
	v, err := strconv.ParseInt(text, 10, 64)
	if err != nil {
		return 0
	}
	return v
}

// detectLanguage determines the SMI version from explicit base-module identity,
// imports from base modules, and syntax that is strong version evidence. It runs
// after definitions are lowered so syntax evidence reflects what was retained.
func detectLanguage(m *module.Module) types.Language {
	if bm, ok := module.BaseModuleFromName(m.Name); ok {
		if bm.IsSMIv1() {
			return types.LanguageSMIv1
		}
		if bm.IsSMIv2() {
			return types.LanguageSMIv2
		}
	}

	hasSMIv1Evidence := false
	hasSMIv2Evidence := false

	for _, imp := range m.Imports {
		bm, ok := module.BaseModuleFromName(imp.Module)
		if !ok {
			continue
		}
		hasSMIv1Evidence = hasSMIv1Evidence || bm.IsSMIv1()
		hasSMIv2Evidence = hasSMIv2Evidence || bm.IsSMIv2()
	}

	if len(m.ModuleIdentities) > 0 {
		hasSMIv2Evidence = true
	}
	for _, notification := range m.Notifications {
		if notification.TrapInfo != nil {
			hasSMIv1Evidence = true
			break
		}
	}

	switch {
	case hasSMIv1Evidence && !hasSMIv2Evidence:
		return types.LanguageSMIv1
	case hasSMIv2Evidence && !hasSMIv1Evidence:
		return types.LanguageSMIv2
	default:
		return types.LanguageUnknown
	}
}

// lowerModule converts a CST ModuleNode to a module.Module.
func (l *lowerer) lowerModule(mod *cst.ModuleNode) *module.Module {
	name := l.text(mod.Name)
	l.moduleName = name

	// Reset collector for this module so diagnostics don't leak between modules.
	l.SpanDiagnosticCollector = types.NewSpanDiagnosticCollector(l.diagConfig)

	// Record parser diagnostics that fall within this module's span.
	for _, d := range l.parserDiags {
		if d.Span.Start >= mod.Span.Start && d.Span.Start < mod.Span.End {
			l.RecordDiagnostic(d)
		}
	}

	m := module.NewModule(name, mod.Span)

	if mod.Imports != nil {
		m.Imports = l.lowerImports(mod.Imports)
	}

	// Lower definitions before finalizing language so retained strong syntax can
	// contribute evidence alongside imports.
	for _, def := range mod.Body {
		l.lowerDefinition(def, m)
	}
	m.Language = detectLanguage(m)

	// Build line table.
	lineTable := types.BuildLineTable(l.source)
	m.LineTable = lineTable

	// Cache LAST-UPDATED from first MODULE-IDENTITY.
	if len(m.ModuleIdentities) > 0 {
		m.LastUpdated = strings.TrimSpace(m.ModuleIdentities[0].LastUpdated)
	}

	// Convert span diagnostics to module diagnostics.
	m.Diagnostics = convertDiagnostics(l.Diagnostics(), name, lineTable, l.diagConfig)

	return m
}

// lowerImports flattens CST import groups into one Import per symbol.
func (l *lowerer) lowerImports(imports *cst.ImportsNode) []module.Import {
	var result []module.Import
	for i := range imports.Groups {
		group := &imports.Groups[i]
		fromModule := l.text(group.Module)
		for _, sym := range group.Symbols {
			symName := l.text(sym)
			result = append(result, module.NewImport(fromModule, symName, sym.Span))
		}
	}
	return result
}

// convertDiagnostics converts SpanDiagnostics to Diagnostics using the line table.
func convertDiagnostics(spanDiags []types.SpanDiagnostic, moduleName string, lineTable []int, config types.DiagnosticConfig) []types.Diagnostic {
	if len(spanDiags) == 0 {
		return nil
	}
	diags := make([]types.Diagnostic, 0, len(spanDiags))
	for _, sd := range spanDiags {
		if !config.ShouldRetain(sd.Code) {
			continue
		}
		line, col := types.LineColFromTable(lineTable, sd.Span.Start)
		diags = append(diags, types.Diagnostic{
			Severity: config.EffectiveSeverity(sd.Code),
			Code:     sd.Code,
			Message:  sd.Message,
			Module:   moduleName,
			Line:     line,
			Column:   col,
		})
	}
	return diags
}

// lowerDefinition dispatches a CST definition node to its specific lowering function
// and appends the result to the module.
func (l *lowerer) lowerDefinition(def cst.DefinitionNode, m *module.Module) {
	switch d := def.(type) {
	case *cst.ObjectTypeNode:
		m.ObjectTypes = append(m.ObjectTypes, l.lowerObjectType(d))
	case *cst.ModuleIdentityNode:
		m.ModuleIdentities = append(m.ModuleIdentities, l.lowerModuleIdentity(d))
	case *cst.ObjectIdentityNode:
		m.ObjectIdentities = append(m.ObjectIdentities, l.lowerObjectIdentity(d))
	case *cst.NotificationTypeNode:
		m.Notifications = append(m.Notifications, l.lowerNotificationType(d))
	case *cst.TrapTypeNode:
		m.Notifications = append(m.Notifications, l.lowerTrapType(d))
	case *cst.TextualConventionNode:
		m.TypeDefs = append(m.TypeDefs, l.lowerTextualConvention(d))
	case *cst.TypeAssignmentNode:
		m.TypeDefs = append(m.TypeDefs, l.lowerTypeAssignment(d))
	case *cst.ValueAssignmentNode:
		m.ValueAssignments = append(m.ValueAssignments, l.lowerValueAssignment(d))
	case *cst.ObjectGroupNode:
		m.ObjectGroups = append(m.ObjectGroups, l.lowerObjectGroup(d))
	case *cst.NotificationGroupNode:
		m.NotificationGroups = append(m.NotificationGroups, l.lowerNotificationGroup(d))
	case *cst.ModuleComplianceNode:
		m.Compliances = append(m.Compliances, l.lowerModuleCompliance(d))
	case *cst.AgentCapabilitiesNode:
		m.Capabilities = append(m.Capabilities, l.lowerAgentCapabilities(d))
	case *cst.MacroDefinitionNode:
		if !module.IsBaseModule(l.moduleName) {
			name := l.text(d.Name)
			l.EmitDiagnostic(types.DiagMacroNotAllowed, d.Span,
				fmt.Sprintf("MACRO definition %q not allowed outside base modules", name))
		}
	case *cst.ErrorNode:
		// Skip error nodes.
	}
}

// checkEmptyString emits a diagnostic if a quoted string token is present but empty.
func (l *lowerer) checkEmptyString(tok cst.SyntaxToken, defName, clause, code string) {
	value := l.stripQuotes(tok)
	if value == "" {
		l.EmitDiagnostic(code, tok.Span,
			fmt.Sprintf("%q: empty %s clause", defName, clause))
	}
}

// stripQuotedLiteral strips the 'xxx'H or 'xxx'B quoting from a hex or binary
// string literal, returning just the inner content.
func stripQuotedLiteral(s string) string {
	s, _ = strings.CutPrefix(s, "'")
	if inner, ok := strings.CutSuffix(s, "'H"); ok {
		return inner
	}
	if inner, ok := strings.CutSuffix(s, "'h"); ok {
		return inner
	}
	if inner, ok := strings.CutSuffix(s, "'B"); ok {
		return inner
	}
	if inner, ok := strings.CutSuffix(s, "'b"); ok {
		return inner
	}
	return s
}
