package module

import (
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/golangsnmp/gomib/internal/types"
)

// smiEpoch is Jan 1, 1990 00:00:00 UTC, the start of the SMI standard era.
var smiEpoch = time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC)

// digit returns the numeric value of a decimal digit at position i in date.
func digit(date string, i int) int {
	return int(date[i] - '0')
}

// validateContext tracks state during module-level validation.
type validateContext struct {
	diagnostics []types.Diagnostic
	diagConfig  types.DiagnosticConfig
	source      []byte
	lineTable   []int
	moduleName  string
	types.Logger
}

// newValidateContext returns a validateContext with a precomputed line table.
func newValidateContext(source []byte, logger *slog.Logger, diagConfig types.DiagnosticConfig, moduleName string) *validateContext {
	return &validateContext{
		diagConfig: diagConfig,
		source:     source,
		lineTable:  types.BuildLineTable(source),
		moduleName: moduleName,
		Logger:     types.Logger{L: logger},
	}
}

// emitDiagnostic records a diagnostic if the current config allows it.
func (ctx *validateContext) emitDiagnostic(code string, span types.Span, message string) {
	sev := types.SeverityForCode(code)
	if !ctx.diagConfig.ShouldReport(code, sev) {
		return
	}
	line, col := types.LineColFromTable(ctx.lineTable, span.Start)
	ctx.diagnostics = append(ctx.diagnostics, types.Diagnostic{
		Severity: sev,
		Code:     code,
		Message:  message,
		Module:   ctx.moduleName,
		Line:     line,
		Column:   col,
	})
}

// ValidateModule runs module-level validation checks on a parsed module,
// emitting diagnostics for structural issues. Call this after parsing and
// before resolution. Source is the original source text, used to compute
// diagnostic line/column from byte offset spans.
func ValidateModule(mod *Module, source []byte, logger *slog.Logger, diagConfig types.DiagnosticConfig) {
	ctx := newValidateContext(source, logger, diagConfig, mod.Name)

	ctx.Log(slog.LevelDebug, "validating module", slog.String("module", mod.Name))

	validateModuleNameSuffix(ctx, mod)

	if mod.Language == types.LanguageSMIv2 && !IsBaseModule(mod.Name) {
		var moduleIdentities []*ModuleIdentity
		firstIndex := -1
		for i, def := range mod.Definitions {
			if mi, ok := def.(*ModuleIdentity); ok {
				if firstIndex == -1 {
					firstIndex = i
				}
				moduleIdentities = append(moduleIdentities, mi)
			}
		}

		if len(moduleIdentities) == 0 {
			ctx.emitDiagnostic(types.DiagMissingModuleIdentity, mod.Span,
				fmt.Sprintf("SMIv2 module %s lacks MODULE-IDENTITY", mod.Name))
		} else {
			if firstIndex > 0 {
				ctx.emitDiagnostic(types.DiagModuleIdentityNotFirst, moduleIdentities[0].Span,
					"MODULE-IDENTITY should be the first definition in "+mod.Name)
			}

			for _, mi := range moduleIdentities[1:] {
				ctx.emitDiagnostic(types.DiagModuleIdentityMultiple, mi.Span,
					"multiple MODULE-IDENTITY definitions in "+mod.Name)
			}
		}

		validateMacroImports(ctx, mod)
	}

	// Validate dates on all MODULE-IDENTITY definitions.
	for _, def := range mod.Definitions {
		if mi, ok := def.(*ModuleIdentity); ok {
			validateModuleIdentityDates(ctx, mi)
			validateRevisionLastUpdated(ctx, mi)
		}
	}

	ctx.Log(slog.LevelDebug, "validation complete",
		slog.String("module", mod.Name),
		slog.Int("diagnostics", len(ctx.diagnostics)))

	mod.Diagnostics = append(mod.Diagnostics, ctx.diagnostics...)
}

// validateModuleNameSuffix warns when an SMIv2 module name does not end with -MIB.
func validateModuleNameSuffix(ctx *validateContext, mod *Module) {
	if mod.Language != types.LanguageSMIv2 || IsBaseModule(mod.Name) {
		return
	}
	if !strings.HasSuffix(mod.Name, "-MIB") {
		ctx.emitDiagnostic(types.DiagModuleNameSuffix, mod.Span,
			fmt.Sprintf("module name %q does not end with -MIB", mod.Name))
	}
}

// validateMacroImports warns when SMIv2 macro keywords are used without being imported.
func validateMacroImports(ctx *validateContext, mod *Module) {
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

// validateModuleIdentityDates validates date formats and revision ordering
// for a MODULE-IDENTITY definition.
func validateModuleIdentityDates(ctx *validateContext, mi *ModuleIdentity) {
	now := time.Now().UTC()

	// Validate LAST-UPDATED date.
	lastUpdatedTime, lastUpdatedValid := validateDate(ctx, mi.LastUpdated, mi.Span, now)

	// Validate each REVISION date and collect parsed times for ordering checks.
	type revEntry struct {
		t     time.Time
		valid bool
		span  types.Span
	}
	revs := make([]revEntry, len(mi.Revisions))
	for i, r := range mi.Revisions {
		t, valid := validateDate(ctx, r.Date, r.Span, now)
		revs[i] = revEntry{t: t, valid: valid, span: r.Span}
	}

	// Check revision ordering: must be reverse chronological (descending).
	for i := 1; i < len(revs); i++ {
		if !revs[i-1].valid || !revs[i].valid {
			continue
		}
		if !revs[i].t.Before(revs[i-1].t) {
			ctx.emitDiagnostic(types.DiagRevisionNotDescending, revs[i].span,
				fmt.Sprintf("revision %s is not in reverse chronological order", mi.Revisions[i].Date))
		}
	}

	// Check revision-after-update: no revision may exceed LAST-UPDATED.
	if lastUpdatedValid {
		for i, r := range revs {
			if !r.valid {
				continue
			}
			if r.t.After(lastUpdatedTime) {
				ctx.emitDiagnostic(types.DiagRevisionAfterUpdate, r.span,
					fmt.Sprintf("revision %s is after LAST-UPDATED %s",
						mi.Revisions[i].Date, mi.LastUpdated))
			}
		}
	}
}

// validateRevisionLastUpdated warns if LAST-UPDATED has no matching REVISION.
func validateRevisionLastUpdated(ctx *validateContext, mi *ModuleIdentity) {
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

// validateDate validates a single SMI date string (ExtUTCTime format).
// Format: YYMMDDHHMMZ (11 chars) or YYYYMMDDHHMMZ (13 chars).
// Returns the parsed time and whether the date is structurally valid.
func validateDate(ctx *validateContext, date string, span types.Span, now time.Time) (time.Time, bool) {
	if date == "" {
		return time.Time{}, false
	}

	// Check length: must be 11 (2-digit year) or 13 (4-digit year).
	if len(date) != 11 && len(date) != 13 {
		ctx.emitDiagnostic(types.DiagDateLength, span,
			fmt.Sprintf("date %q has illegal length %d (expected 11 or 13)", date, len(date)))
		return time.Time{}, false
	}

	// All characters before final must be digits, final must be 'Z'.
	for i := range len(date) - 1 {
		if date[i] < '0' || date[i] > '9' {
			ctx.emitDiagnostic(types.DiagDateCharacter, span,
				fmt.Sprintf("date %q contains illegal character at position %d", date, i+1))
			return time.Time{}, false
		}
	}
	if date[len(date)-1] != 'Z' {
		ctx.emitDiagnostic(types.DiagDateCharacter, span,
			fmt.Sprintf("date %q must end with 'Z'", date))
		return time.Time{}, false
	}

	// Parse numeric components.
	var year int
	var offset int
	if len(date) == 11 {
		yy := digit(date, 0)*10 + digit(date, 1)
		if yy >= 70 {
			year = 1900 + yy
		} else {
			year = 2000 + yy
		}
		offset = 2
		ctx.emitDiagnostic(types.DiagDateYear2Digits, span,
			fmt.Sprintf("date %q uses 2-digit year representing %d", date, year))
	} else {
		year = digit(date, 0)*1000 + digit(date, 1)*100 + digit(date, 2)*10 + digit(date, 3)
		offset = 4
	}

	month := digit(date, offset)*10 + digit(date, offset+1)
	day := digit(date, offset+2)*10 + digit(date, offset+3)
	hour := digit(date, offset+4)*10 + digit(date, offset+5)
	min := digit(date, offset+6)*10 + digit(date, offset+7)

	// Validate ranges.
	if month < 1 || month > 12 {
		ctx.emitDiagnostic(types.DiagDateMonth, span,
			fmt.Sprintf("date %q has illegal month %02d", date, month))
		return time.Time{}, false
	}
	if day < 1 || day > 31 {
		ctx.emitDiagnostic(types.DiagDateDay, span,
			fmt.Sprintf("date %q has illegal day %02d", date, day))
		return time.Time{}, false
	}
	if hour > 23 {
		ctx.emitDiagnostic(types.DiagDateHour, span,
			fmt.Sprintf("date %q has illegal hour %02d", date, hour))
		return time.Time{}, false
	}
	if min > 59 {
		ctx.emitDiagnostic(types.DiagDateMinutes, span,
			fmt.Sprintf("date %q has illegal minutes %02d", date, min))
		return time.Time{}, false
	}

	// Check calendar validity (e.g. Feb 30 is not valid).
	t := time.Date(year, time.Month(month), day, hour, min, 0, 0, time.UTC)
	if t.Month() != time.Month(month) || t.Day() != day {
		ctx.emitDiagnostic(types.DiagDateValue, span,
			fmt.Sprintf("date %q is not a valid calendar date", date))
		return time.Time{}, false
	}

	// Style warnings for dates outside reasonable range.
	if t.Before(smiEpoch) {
		ctx.emitDiagnostic(types.DiagDateInPast, span,
			fmt.Sprintf("date %q predates the SMI standard", date))
	}
	if t.After(now) {
		ctx.emitDiagnostic(types.DiagDateInFuture, span,
			fmt.Sprintf("date %q is in the future", date))
	}

	return t, true
}
