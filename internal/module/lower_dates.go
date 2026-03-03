package module

import (
	"fmt"
	"time"

	"github.com/golangsnmp/gomib/internal/ast"
	"github.com/golangsnmp/gomib/internal/types"
)

// smiEpoch is Jan 1, 1990 00:00:00 UTC, the start of the SMI standard era.
var smiEpoch = time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC)

// checkModuleIdentityDates validates date formats and revision ordering
// for a MODULE-IDENTITY definition. Uses the AST form for per-date spans.
func checkModuleIdentityDates(ctx *LoweringContext, def *ast.ModuleIdentityDef) {
	now := time.Now().UTC()

	// Validate LAST-UPDATED date.
	lastUpdatedTime, lastUpdatedValid := checkDate(ctx, def.LastUpdated.Value, def.LastUpdated.Span, now)

	// Validate each REVISION date and collect parsed times for ordering checks.
	type revEntry struct {
		t     time.Time
		valid bool
		span  types.Span
	}
	revs := make([]revEntry, len(def.Revisions))
	for i, r := range def.Revisions {
		t, valid := checkDate(ctx, r.Date.Value, r.Date.Span, now)
		revs[i] = revEntry{t: t, valid: valid, span: r.Date.Span}
	}

	// Check revision ordering: must be reverse chronological (descending).
	for i := 1; i < len(revs); i++ {
		if !revs[i-1].valid || !revs[i].valid {
			continue
		}
		if !revs[i].t.Before(revs[i-1].t) {
			ctx.emitDiagnostic(types.DiagRevisionNotDescending, revs[i].span,
				fmt.Sprintf("revision %s is not in reverse chronological order", def.Revisions[i].Date.Value))
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
						def.Revisions[i].Date.Value, def.LastUpdated.Value))
			}
		}
	}
}

// checkDate validates a single SMI date string (ExtUTCTime format).
// Format: YYMMDDHHMMZ (11 chars) or YYYYMMDDHHMMZ (13 chars).
// Returns the parsed time and whether the date is structurally valid.
func checkDate(ctx *LoweringContext, date string, span types.Span, now time.Time) (time.Time, bool) {
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
		year = digit(date, 0)*10 + digit(date, 1) + 1900
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

// digit returns the numeric value of date[i] as an int.
func digit(date string, i int) int {
	return int(date[i] - '0')
}
