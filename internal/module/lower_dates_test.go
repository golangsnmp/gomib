package module

import (
	"strings"
	"testing"

	"github.com/golangsnmp/gomib/internal/testutil"
	"github.com/golangsnmp/gomib/internal/types"
)

// smiv2Preamble is a minimal SMIv2 module header for date validation tests.
// The LAST-UPDATED and REVISION dates are replaced per test case.
func dateTestSource(lastUpdated, revisions string) []byte {
	return []byte(`DATE-TEST-MIB DEFINITIONS ::= BEGIN

IMPORTS
    MODULE-IDENTITY, enterprises
        FROM SNMPv2-SMI;

dateTest MODULE-IDENTITY
    LAST-UPDATED "` + lastUpdated + `"
    ORGANIZATION "Test"
    CONTACT-INFO "Test"
    DESCRIPTION  "Test"
` + revisions + `
    ::= { enterprises 99999 }

END
`)
}

func dateTestSourceSingleRev(date string) []byte {
	return dateTestSource(date, `    REVISION "`+date+`"
    DESCRIPTION "Test"`)
}

func TestLower_DateLength_TooShort(t *testing.T) {
	src := dateTestSourceSingleRev("200001Z")
	d := lowerAndFindDiagnostic(t, src, types.VerboseConfig(), types.DiagDateLength)
	testutil.NotNil(t, d, "expected %s diagnostic", types.DiagDateLength)
	testutil.Equal(t, types.SeverityError, d.Severity, "severity")
	testutil.True(t, strings.Contains(d.Message, "200001Z"), "message should mention date: %s", d.Message)
}

func TestLower_DateLength_TooLong(t *testing.T) {
	src := dateTestSourceSingleRev("20000101000000Z")
	d := lowerAndFindDiagnostic(t, src, types.VerboseConfig(), types.DiagDateLength)
	testutil.NotNil(t, d, "expected %s diagnostic", types.DiagDateLength)
}

func TestLower_DateCharacter_NonDigit(t *testing.T) {
	src := dateTestSourceSingleRev("2000XX010000Z")
	d := lowerAndFindDiagnostic(t, src, types.VerboseConfig(), types.DiagDateCharacter)
	testutil.NotNil(t, d, "expected %s diagnostic", types.DiagDateCharacter)
	testutil.Equal(t, types.SeverityError, d.Severity, "severity")
}

func TestLower_DateCharacter_NoTrailingZ(t *testing.T) {
	src := dateTestSourceSingleRev("2000010100001")
	d := lowerAndFindDiagnostic(t, src, types.VerboseConfig(), types.DiagDateCharacter)
	testutil.NotNil(t, d, "expected %s diagnostic for missing Z", types.DiagDateCharacter)
}

func TestLower_DateMonth_Zero(t *testing.T) {
	src := dateTestSourceSingleRev("200000010000Z")
	d := lowerAndFindDiagnostic(t, src, types.VerboseConfig(), types.DiagDateMonth)
	testutil.NotNil(t, d, "expected %s diagnostic for month 00", types.DiagDateMonth)
	testutil.Equal(t, types.SeverityError, d.Severity, "severity")
}

func TestLower_DateMonth_Thirteen(t *testing.T) {
	src := dateTestSourceSingleRev("200013010000Z")
	d := lowerAndFindDiagnostic(t, src, types.VerboseConfig(), types.DiagDateMonth)
	testutil.NotNil(t, d, "expected %s diagnostic for month 13", types.DiagDateMonth)
}

func TestLower_DateDay_Zero(t *testing.T) {
	src := dateTestSourceSingleRev("200001000000Z")
	d := lowerAndFindDiagnostic(t, src, types.VerboseConfig(), types.DiagDateDay)
	testutil.NotNil(t, d, "expected %s diagnostic for day 00", types.DiagDateDay)
}

func TestLower_DateDay_ThirtyTwo(t *testing.T) {
	src := dateTestSourceSingleRev("200001320000Z")
	d := lowerAndFindDiagnostic(t, src, types.VerboseConfig(), types.DiagDateDay)
	testutil.NotNil(t, d, "expected %s diagnostic for day 32", types.DiagDateDay)
}

func TestLower_DateHour_TwentyFour(t *testing.T) {
	src := dateTestSourceSingleRev("200001012400Z")
	d := lowerAndFindDiagnostic(t, src, types.VerboseConfig(), types.DiagDateHour)
	testutil.NotNil(t, d, "expected %s diagnostic for hour 24", types.DiagDateHour)
}

func TestLower_DateMinutes_Sixty(t *testing.T) {
	src := dateTestSourceSingleRev("200001010060Z")
	d := lowerAndFindDiagnostic(t, src, types.VerboseConfig(), types.DiagDateMinutes)
	testutil.NotNil(t, d, "expected %s diagnostic for minutes 60", types.DiagDateMinutes)
}

func TestLower_DateValue_Feb30(t *testing.T) {
	src := dateTestSourceSingleRev("200002300000Z")
	d := lowerAndFindDiagnostic(t, src, types.VerboseConfig(), types.DiagDateValue)
	testutil.NotNil(t, d, "expected %s diagnostic for Feb 30", types.DiagDateValue)
}

func TestLower_DateValue_Feb29NonLeap(t *testing.T) {
	src := dateTestSourceSingleRev("200102290000Z")
	d := lowerAndFindDiagnostic(t, src, types.VerboseConfig(), types.DiagDateValue)
	testutil.NotNil(t, d, "expected %s diagnostic for Feb 29 in non-leap year", types.DiagDateValue)
}

func TestLower_DateValue_Feb29Leap(t *testing.T) {
	// 2000 is a leap year; Feb 29 is valid.
	src := dateTestSourceSingleRev("200002290000Z")
	d := lowerAndFindDiagnostic(t, src, types.VerboseConfig(), types.DiagDateValue)
	testutil.Nil(t, d, "Feb 29 in leap year should not trigger %s", types.DiagDateValue)
}

func TestLower_DateYear2Digits(t *testing.T) {
	// 11-char format: "9501010000Z" = year 1995 (>=70 maps to 19xx)
	src := dateTestSourceSingleRev("9501010000Z")
	d := lowerAndFindDiagnostic(t, src, types.VerboseConfig(), types.DiagDateYear2Digits)
	testutil.NotNil(t, d, "expected %s diagnostic for 2-digit year", types.DiagDateYear2Digits)
	testutil.Equal(t, types.SeverityWarning, d.Severity, "severity")
	testutil.True(t, strings.Contains(d.Message, "1995"), "message should mention interpreted year: %s", d.Message)
}

func TestLower_DateYear2Digits_Post2000(t *testing.T) {
	// 11-char format: "0501010000Z" = year 2005 (<70 maps to 20xx)
	src := dateTestSourceSingleRev("0501010000Z")
	d := lowerAndFindDiagnostic(t, src, types.VerboseConfig(), types.DiagDateYear2Digits)
	testutil.NotNil(t, d, "expected %s diagnostic for 2-digit year", types.DiagDateYear2Digits)
	testutil.Equal(t, types.SeverityWarning, d.Severity, "severity")
	testutil.True(t, strings.Contains(d.Message, "2005"), "message should mention interpreted year: %s", d.Message)
}

func TestLower_DateInPast(t *testing.T) {
	// 1985 is before the SMI epoch (1990).
	src := dateTestSourceSingleRev("198501010000Z")
	d := lowerAndFindDiagnostic(t, src, types.VerboseConfig(), types.DiagDateInPast)
	testutil.NotNil(t, d, "expected %s diagnostic for pre-1990 date", types.DiagDateInPast)
	testutil.Equal(t, types.SeverityStyle, d.Severity, "severity")
}

func TestLower_DateInFuture(t *testing.T) {
	// Use a far-future date to avoid flakiness.
	src := dateTestSourceSingleRev("209901010000Z")
	d := lowerAndFindDiagnostic(t, src, types.VerboseConfig(), types.DiagDateInFuture)
	testutil.NotNil(t, d, "expected %s diagnostic for future date", types.DiagDateInFuture)
	testutil.Equal(t, types.SeverityStyle, d.Severity, "severity")
}

func TestLower_DateValid_NoDiagnostic(t *testing.T) {
	// A well-formed date should not trigger any date-* diagnostic.
	src := dateTestSourceSingleRev("200501150000Z")
	codes := []string{
		types.DiagDateLength, types.DiagDateCharacter,
		types.DiagDateMonth, types.DiagDateDay,
		types.DiagDateHour, types.DiagDateMinutes,
		types.DiagDateValue, types.DiagDateInPast,
	}
	for _, code := range codes {
		d := lowerAndFindDiagnostic(t, src, types.VerboseConfig(), code)
		testutil.Nil(t, d, "valid date should not trigger %s", code)
	}
}

func TestLower_RevisionNotDescending(t *testing.T) {
	src := dateTestSource("200501010000Z", `
    REVISION "200301010000Z"
    DESCRIPTION "2003 listed first"
    REVISION "200501010000Z"
    DESCRIPTION "2005 listed second - wrong, should be first"
    REVISION "200401010000Z"
    DESCRIPTION "2004 listed third"
`)
	d := lowerAndFindDiagnostic(t, src, types.VerboseConfig(), types.DiagRevisionNotDescending)
	testutil.NotNil(t, d, "expected %s diagnostic", types.DiagRevisionNotDescending)
	testutil.Equal(t, types.SeverityMinor, d.Severity, "severity")
}

func TestLower_RevisionNotDescending_CorrectOrder(t *testing.T) {
	// Revisions in correct descending order.
	src := dateTestSource("200501010000Z", `
    REVISION "200501010000Z"
    DESCRIPTION "2005"
    REVISION "200401010000Z"
    DESCRIPTION "2004"
    REVISION "200301010000Z"
    DESCRIPTION "2003"
`)
	d := lowerAndFindDiagnostic(t, src, types.VerboseConfig(), types.DiagRevisionNotDescending)
	testutil.Nil(t, d, "correct ordering should not trigger %s", types.DiagRevisionNotDescending)
}

func TestLower_RevisionAfterUpdate(t *testing.T) {
	src := dateTestSource("200401010000Z", `
    REVISION "200501010000Z"
    DESCRIPTION "revision after LAST-UPDATED"
    REVISION "200401010000Z"
    DESCRIPTION "matches LAST-UPDATED"
`)
	d := lowerAndFindDiagnostic(t, src, types.VerboseConfig(), types.DiagRevisionAfterUpdate)
	testutil.NotNil(t, d, "expected %s diagnostic", types.DiagRevisionAfterUpdate)
	testutil.Equal(t, types.SeverityMinor, d.Severity, "severity")
	testutil.True(t, strings.Contains(d.Message, "200501010000Z"), "message should mention revision date: %s", d.Message)
}

func TestLower_RevisionAfterUpdate_NoneAfter(t *testing.T) {
	src := dateTestSource("200501010000Z", `
    REVISION "200501010000Z"
    DESCRIPTION "matches"
    REVISION "200401010000Z"
    DESCRIPTION "before"
`)
	d := lowerAndFindDiagnostic(t, src, types.VerboseConfig(), types.DiagRevisionAfterUpdate)
	testutil.Nil(t, d, "revision matching LAST-UPDATED should not trigger %s", types.DiagRevisionAfterUpdate)
}

func TestLower_DateChecks_SMIv1_Skipped(t *testing.T) {
	// SMIv1 modules don't have MODULE-IDENTITY, so no date checks run.
	src := []byte(`SMIV1-MIB DEFINITIONS ::= BEGIN

IMPORTS
    enterprises
        FROM RFC1155-SMI;

testObj OBJECT-TYPE
    SYNTAX      INTEGER
    ACCESS      read-only
    STATUS      mandatory
    DESCRIPTION "Test"
    ::= { enterprises 99999 }

END
`)
	for _, code := range []string{
		types.DiagDateLength, types.DiagDateCharacter,
		types.DiagRevisionNotDescending, types.DiagRevisionAfterUpdate,
	} {
		d := lowerAndFindDiagnostic(t, src, types.VerboseConfig(), code)
		testutil.Nil(t, d, "SMIv1 module should not get %s", code)
	}
}
