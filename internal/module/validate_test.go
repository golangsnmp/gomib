package module

import (
	"strings"
	"testing"

	"github.com/golangsnmp/gomib/internal/testutil"
	"github.com/golangsnmp/gomib/internal/types"
)

// validateAndFindDiagnostic runs ValidateModule on a module and returns the
// first diagnostic matching code. Returns nil if no match.
func validateAndFindDiagnostic(t *testing.T, mod *Module, config types.DiagnosticConfig, code string) *types.Diagnostic {
	t.Helper()
	ValidateModule(mod, nil, nil, config)
	for _, d := range mod.Diagnostics {
		if d.Code == code {
			return &d
		}
	}
	return nil
}

// addDefs adds definitions to a module, populating the appropriate typed slices.
func addDefs(mod *Module, defs ...Definition) {
	for _, def := range defs {
		switch d := def.(type) {
		case *ModuleIdentity:
			mod.ModuleIdentities = append(mod.ModuleIdentities, d)
		case *ObjectIdentity:
			mod.ObjectIdentities = append(mod.ObjectIdentities, d)
		case *ObjectType:
			mod.ObjectTypes = append(mod.ObjectTypes, d)
		case *Notification:
			mod.Notifications = append(mod.Notifications, d)
		case *TypeDef:
			mod.TypeDefs = append(mod.TypeDefs, d)
		case *ObjectGroup:
			mod.ObjectGroups = append(mod.ObjectGroups, d)
		case *NotificationGroup:
			mod.NotificationGroups = append(mod.NotificationGroups, d)
		case *ModuleCompliance:
			mod.Compliances = append(mod.Compliances, d)
		case *AgentCapabilities:
			mod.Capabilities = append(mod.Capabilities, d)
		}
	}
}

// smiv2Module returns a minimal SMIv2 module with the given name and definitions.
func smiv2Module(name string, defs ...Definition) *Module {
	mod := &Module{
		Name:     name,
		Language: types.LanguageSMIv2,
		Imports: []Import{
			{Module: "SNMPv2-SMI", Symbol: "MODULE-IDENTITY"},
		},
		Span: types.Span{Start: 0, End: 100},
	}
	addDefs(mod, defs...)
	return mod
}

// smiv1Module returns a minimal SMIv1 module with the given name and definitions.
func smiv1Module(name string, defs ...Definition) *Module {
	mod := &Module{
		Name:     name,
		Language: types.LanguageSMIv1,
		Imports: []Import{
			{Module: "RFC1155-SMI", Symbol: "enterprises"},
		},
		Span: types.Span{Start: 0, End: 100},
	}
	addDefs(mod, defs...)
	return mod
}

// testMI returns a ModuleIdentity definition with sensible defaults.
func testMI(name, lastUpdated string, revisions ...Revision) *ModuleIdentity {
	return &ModuleIdentity{
		DefBase:      DefBase{Name: name, Span: types.Span{Start: 10, End: 90}},
		LastUpdated:  lastUpdated,
		Organization: "Test Org",
		ContactInfo:  "Test Contact",
		Description:  "Test Description",
		Revisions:    revisions,
	}
}

// testOT returns a simple ObjectType definition.
func testOT(name string) *ObjectType {
	return &ObjectType{
		DefBase:     DefBase{Name: name, Span: types.Span{Start: 50, End: 80}},
		Description: "Test",
		Status:      types.StatusCurrent,
		Access:      types.AccessReadOnly,
	}
}

// --- MODULE-IDENTITY presence ---

func TestValidate_MissingModuleIdentity(t *testing.T) {
	mod := smiv2Module("NO-IDENTITY-MIB",
		testOT("someObject"),
	)
	// Need OBJECT-TYPE import for this to not also trigger macro-not-imported.
	mod.Imports = append(mod.Imports, Import{Module: "SNMPv2-SMI", Symbol: "OBJECT-TYPE"})

	d := validateAndFindDiagnostic(t, mod, types.VerboseConfig(), types.DiagMissingModuleIdentity)
	testutil.NotNil(t, d, "expected %s diagnostic", types.DiagMissingModuleIdentity)
	testutil.Equal(t, types.SeverityWarning, d.Severity, "severity")
}

func TestValidate_MissingModuleIdentity_BaseModuleSkipped(t *testing.T) {
	// Actual base modules (SNMPv2-SMI) should NOT get the missing-module-identity check.
	mod := &Module{
		Name:     "SNMPv2-SMI",
		Language: types.LanguageSMIv2,
		Span:     types.Span{Start: 0, End: 100},
	}
	d := validateAndFindDiagnostic(t, mod, types.VerboseConfig(), types.DiagMissingModuleIdentity)
	testutil.Nil(t, d, "base module SNMPv2-SMI should not get %s diagnostic", types.DiagMissingModuleIdentity)
}

func TestValidate_MissingModuleIdentity_SNMPv2MIBNotBase(t *testing.T) {
	// SNMPv2-MIB is NOT a base module. It should get the diagnostic.
	mod := smiv2Module("SNMPv2-MIB",
		testOT("sysDescr"),
	)
	mod.Imports = append(mod.Imports, Import{Module: "SNMPv2-SMI", Symbol: "OBJECT-TYPE"})

	d := validateAndFindDiagnostic(t, mod, types.VerboseConfig(), types.DiagMissingModuleIdentity)
	testutil.NotNil(t, d, "SNMPv2-MIB without MODULE-IDENTITY should get %s diagnostic", types.DiagMissingModuleIdentity)
}

// --- MODULE-IDENTITY position ---

func TestValidate_ModuleIdentityNotFirst(t *testing.T) {
	ot := testOT("someObject")
	ot.Span = types.Span{Start: 10, End: 30} // OT comes first in source
	mi := testMI("notFirstTest", "200001010000Z",
		Revision{Date: "200001010000Z", Description: "Test", Span: types.Span{Start: 50, End: 70}},
	)
	mi.Span = types.Span{Start: 40, End: 90} // MI comes after OT
	mod := smiv2Module("NOT-FIRST-MIB", ot, mi)
	mod.Imports = append(mod.Imports, Import{Module: "SNMPv2-SMI", Symbol: "OBJECT-TYPE"})

	d := validateAndFindDiagnostic(t, mod, types.VerboseConfig(), types.DiagModuleIdentityNotFirst)
	testutil.NotNil(t, d, "expected %s diagnostic", types.DiagModuleIdentityNotFirst)
	testutil.Equal(t, types.SeverityWarning, d.Severity, "severity")
	testutil.True(t, strings.Contains(d.Message, "NOT-FIRST-MIB"), "message should mention module name: %s", d.Message)
}

func TestValidate_ModuleIdentityNotFirst_WhenFirst(t *testing.T) {
	mod := smiv2Module("FIRST-MIB",
		testMI("firstTest", "200001010000Z",
			Revision{Date: "200001010000Z", Description: "Test", Span: types.Span{Start: 30, End: 50}},
		),
		testOT("someObject"),
	)
	mod.Imports = append(mod.Imports, Import{Module: "SNMPv2-SMI", Symbol: "OBJECT-TYPE"})

	d := validateAndFindDiagnostic(t, mod, types.VerboseConfig(), types.DiagModuleIdentityNotFirst)
	testutil.Nil(t, d, "should not get %s when MODULE-IDENTITY is first", types.DiagModuleIdentityNotFirst)
}

// --- MODULE-IDENTITY count ---

func TestValidate_ModuleIdentityMultiple(t *testing.T) {
	mod := smiv2Module("MULTI-MI-MIB",
		testMI("first", "200001010000Z",
			Revision{Date: "200001010000Z", Description: "First", Span: types.Span{Start: 20, End: 40}},
		),
		testMI("second", "200001010000Z",
			Revision{Date: "200001010000Z", Description: "Second", Span: types.Span{Start: 50, End: 70}},
		),
	)

	d := validateAndFindDiagnostic(t, mod, types.VerboseConfig(), types.DiagModuleIdentityMultiple)
	testutil.NotNil(t, d, "expected %s diagnostic", types.DiagModuleIdentityMultiple)
	testutil.Equal(t, types.SeverityError, d.Severity, "severity")
}

// --- Macro import checks ---

func TestValidate_MacroNotImported(t *testing.T) {
	// Uses OBJECT-TYPE but doesn't import it.
	mod := smiv2Module("MACRO-TEST-MIB",
		testMI("macroTest", "200001010000Z",
			Revision{Date: "200001010000Z", Description: "Test", Span: types.Span{Start: 20, End: 40}},
		),
		testOT("someObj"),
	)
	// Only MODULE-IDENTITY imported, not OBJECT-TYPE.

	d := validateAndFindDiagnostic(t, mod, types.VerboseConfig(), types.DiagMacroNotImported)
	testutil.NotNil(t, d, "expected %s diagnostic", types.DiagMacroNotImported)
	testutil.Equal(t, types.SeverityMinor, d.Severity, "severity")
	testutil.True(t, strings.Contains(d.Message, "OBJECT-TYPE"), "message should mention OBJECT-TYPE: %s", d.Message)
}

func TestValidate_MacroNotImported_WhenImported(t *testing.T) {
	mod := smiv2Module("MACRO-OK-MIB",
		testMI("macroOk", "200001010000Z",
			Revision{Date: "200001010000Z", Description: "Test", Span: types.Span{Start: 20, End: 40}},
		),
		testOT("someObj"),
	)
	mod.Imports = append(mod.Imports, Import{Module: "SNMPv2-SMI", Symbol: "OBJECT-TYPE"})

	d := validateAndFindDiagnostic(t, mod, types.VerboseConfig(), types.DiagMacroNotImported)
	testutil.Nil(t, d, "should not get %s when macro is imported", types.DiagMacroNotImported)
}

func TestValidate_MacroNotImported_SMIv1(t *testing.T) {
	// SMIv1 modules should NOT get macro-not-imported diagnostics.
	mod := smiv1Module("SMIV1-MACRO-MIB",
		testOT("someObj"),
	)

	d := validateAndFindDiagnostic(t, mod, types.VerboseConfig(), types.DiagMacroNotImported)
	testutil.Nil(t, d, "SMIv1 module should not get %s", types.DiagMacroNotImported)
}

// --- Module name suffix ---

func TestValidate_ModuleNameSuffix(t *testing.T) {
	mod := smiv2Module("MY-MODULE",
		testMI("myModule", "200001010000Z",
			Revision{Date: "200001010000Z", Description: "Test", Span: types.Span{Start: 20, End: 40}},
		),
	)

	d := validateAndFindDiagnostic(t, mod, types.VerboseConfig(), types.DiagModuleNameSuffix)
	testutil.NotNil(t, d, "expected %s diagnostic for module without -MIB suffix", types.DiagModuleNameSuffix)
	testutil.Equal(t, types.SeverityStyle, d.Severity, "severity")
	testutil.True(t, strings.Contains(d.Message, "MY-MODULE"), "message should mention module name: %s", d.Message)
}

func TestValidate_ModuleNameSuffix_WithMIB_NoDiag(t *testing.T) {
	mod := smiv2Module("MY-MODULE-MIB",
		testMI("myModule", "200001010000Z",
			Revision{Date: "200001010000Z", Description: "Test", Span: types.Span{Start: 20, End: 40}},
		),
	)

	d := validateAndFindDiagnostic(t, mod, types.VerboseConfig(), types.DiagModuleNameSuffix)
	testutil.Nil(t, d, "should not get %s with -MIB suffix", types.DiagModuleNameSuffix)
}

func TestValidate_ModuleNameSuffix_SMIv1_NoDiag(t *testing.T) {
	mod := smiv1Module("MY-MODULE",
		testOT("testObj"),
	)

	d := validateAndFindDiagnostic(t, mod, types.VerboseConfig(), types.DiagModuleNameSuffix)
	testutil.Nil(t, d, "SMIv1 module should not get %s", types.DiagModuleNameSuffix)
}

func TestValidate_ModuleNameSuffix_BaseModule_NoDiag(t *testing.T) {
	mod := &Module{
		Name:     "SNMPv2-TC",
		Language: types.LanguageSMIv2,
		Span:     types.Span{Start: 0, End: 100},
	}

	d := validateAndFindDiagnostic(t, mod, types.VerboseConfig(), types.DiagModuleNameSuffix)
	testutil.Nil(t, d, "base module should not get %s", types.DiagModuleNameSuffix)
}

// --- Date validation ---

func TestValidate_DateLength_TooShort(t *testing.T) {
	mod := smiv2Module("DATE-TEST-MIB",
		testMI("dateTest", "200001Z",
			Revision{Date: "200001Z", Description: "Test", Span: types.Span{Start: 20, End: 40}},
		),
	)

	d := validateAndFindDiagnostic(t, mod, types.VerboseConfig(), types.DiagDateLength)
	testutil.NotNil(t, d, "expected %s diagnostic", types.DiagDateLength)
	testutil.Equal(t, types.SeverityError, d.Severity, "severity")
	testutil.True(t, strings.Contains(d.Message, "200001Z"), "message should mention date: %s", d.Message)
}

func TestValidate_DateLength_TooLong(t *testing.T) {
	mod := smiv2Module("DATE-TEST-MIB",
		testMI("dateTest", "20000101000000Z",
			Revision{Date: "20000101000000Z", Description: "Test", Span: types.Span{Start: 20, End: 40}},
		),
	)

	d := validateAndFindDiagnostic(t, mod, types.VerboseConfig(), types.DiagDateLength)
	testutil.NotNil(t, d, "expected %s diagnostic", types.DiagDateLength)
}

func TestValidate_DateCharacter_NonDigit(t *testing.T) {
	mod := smiv2Module("DATE-TEST-MIB",
		testMI("dateTest", "2000XX010000Z",
			Revision{Date: "2000XX010000Z", Description: "Test", Span: types.Span{Start: 20, End: 40}},
		),
	)

	d := validateAndFindDiagnostic(t, mod, types.VerboseConfig(), types.DiagDateCharacter)
	testutil.NotNil(t, d, "expected %s diagnostic", types.DiagDateCharacter)
	testutil.Equal(t, types.SeverityError, d.Severity, "severity")
}

func TestValidate_DateCharacter_NoTrailingZ(t *testing.T) {
	mod := smiv2Module("DATE-TEST-MIB",
		testMI("dateTest", "2000010100001",
			Revision{Date: "2000010100001", Description: "Test", Span: types.Span{Start: 20, End: 40}},
		),
	)

	d := validateAndFindDiagnostic(t, mod, types.VerboseConfig(), types.DiagDateCharacter)
	testutil.NotNil(t, d, "expected %s diagnostic for missing Z", types.DiagDateCharacter)
}

func TestValidate_DateMonth_Zero(t *testing.T) {
	mod := smiv2Module("DATE-TEST-MIB",
		testMI("dateTest", "200000010000Z",
			Revision{Date: "200000010000Z", Description: "Test", Span: types.Span{Start: 20, End: 40}},
		),
	)

	d := validateAndFindDiagnostic(t, mod, types.VerboseConfig(), types.DiagDateMonth)
	testutil.NotNil(t, d, "expected %s diagnostic for month 00", types.DiagDateMonth)
	testutil.Equal(t, types.SeverityError, d.Severity, "severity")
}

func TestValidate_DateMonth_Thirteen(t *testing.T) {
	mod := smiv2Module("DATE-TEST-MIB",
		testMI("dateTest", "200013010000Z",
			Revision{Date: "200013010000Z", Description: "Test", Span: types.Span{Start: 20, End: 40}},
		),
	)

	d := validateAndFindDiagnostic(t, mod, types.VerboseConfig(), types.DiagDateMonth)
	testutil.NotNil(t, d, "expected %s diagnostic for month 13", types.DiagDateMonth)
}

func TestValidate_DateDay_Zero(t *testing.T) {
	mod := smiv2Module("DATE-TEST-MIB",
		testMI("dateTest", "200001000000Z",
			Revision{Date: "200001000000Z", Description: "Test", Span: types.Span{Start: 20, End: 40}},
		),
	)

	d := validateAndFindDiagnostic(t, mod, types.VerboseConfig(), types.DiagDateDay)
	testutil.NotNil(t, d, "expected %s diagnostic for day 00", types.DiagDateDay)
}

func TestValidate_DateDay_ThirtyTwo(t *testing.T) {
	mod := smiv2Module("DATE-TEST-MIB",
		testMI("dateTest", "200001320000Z",
			Revision{Date: "200001320000Z", Description: "Test", Span: types.Span{Start: 20, End: 40}},
		),
	)

	d := validateAndFindDiagnostic(t, mod, types.VerboseConfig(), types.DiagDateDay)
	testutil.NotNil(t, d, "expected %s diagnostic for day 32", types.DiagDateDay)
}

func TestValidate_DateHour_TwentyFour(t *testing.T) {
	mod := smiv2Module("DATE-TEST-MIB",
		testMI("dateTest", "200001012400Z",
			Revision{Date: "200001012400Z", Description: "Test", Span: types.Span{Start: 20, End: 40}},
		),
	)

	d := validateAndFindDiagnostic(t, mod, types.VerboseConfig(), types.DiagDateHour)
	testutil.NotNil(t, d, "expected %s diagnostic for hour 24", types.DiagDateHour)
}

func TestValidate_DateMinutes_Sixty(t *testing.T) {
	mod := smiv2Module("DATE-TEST-MIB",
		testMI("dateTest", "200001010060Z",
			Revision{Date: "200001010060Z", Description: "Test", Span: types.Span{Start: 20, End: 40}},
		),
	)

	d := validateAndFindDiagnostic(t, mod, types.VerboseConfig(), types.DiagDateMinutes)
	testutil.NotNil(t, d, "expected %s diagnostic for minutes 60", types.DiagDateMinutes)
}

func TestValidate_DateValue_Feb30(t *testing.T) {
	mod := smiv2Module("DATE-TEST-MIB",
		testMI("dateTest", "200002300000Z",
			Revision{Date: "200002300000Z", Description: "Test", Span: types.Span{Start: 20, End: 40}},
		),
	)

	d := validateAndFindDiagnostic(t, mod, types.VerboseConfig(), types.DiagDateValue)
	testutil.NotNil(t, d, "expected %s diagnostic for Feb 30", types.DiagDateValue)
}

func TestValidate_DateValue_Feb29NonLeap(t *testing.T) {
	mod := smiv2Module("DATE-TEST-MIB",
		testMI("dateTest", "200102290000Z",
			Revision{Date: "200102290000Z", Description: "Test", Span: types.Span{Start: 20, End: 40}},
		),
	)

	d := validateAndFindDiagnostic(t, mod, types.VerboseConfig(), types.DiagDateValue)
	testutil.NotNil(t, d, "expected %s diagnostic for Feb 29 in non-leap year", types.DiagDateValue)
}

func TestValidate_DateValue_Feb29Leap(t *testing.T) {
	mod := smiv2Module("DATE-TEST-MIB",
		testMI("dateTest", "200002290000Z",
			Revision{Date: "200002290000Z", Description: "Test", Span: types.Span{Start: 20, End: 40}},
		),
	)

	d := validateAndFindDiagnostic(t, mod, types.VerboseConfig(), types.DiagDateValue)
	testutil.Nil(t, d, "Feb 29 in leap year should not trigger %s", types.DiagDateValue)
}

func TestValidate_DateYear2Digits(t *testing.T) {
	// 11-char format: "9501010000Z" = year 1995 (>=70 maps to 19xx)
	mod := smiv2Module("DATE-TEST-MIB",
		testMI("dateTest", "9501010000Z",
			Revision{Date: "9501010000Z", Description: "Test", Span: types.Span{Start: 20, End: 40}},
		),
	)

	d := validateAndFindDiagnostic(t, mod, types.VerboseConfig(), types.DiagDateYear2Digits)
	testutil.NotNil(t, d, "expected %s diagnostic for 2-digit year", types.DiagDateYear2Digits)
	testutil.Equal(t, types.SeverityWarning, d.Severity, "severity")
	testutil.True(t, strings.Contains(d.Message, "1995"), "message should mention interpreted year: %s", d.Message)
}

func TestValidate_DateYear2Digits_Post2000(t *testing.T) {
	// 11-char format: "0501010000Z" = year 2005 (<70 maps to 20xx)
	mod := smiv2Module("DATE-TEST-MIB",
		testMI("dateTest", "0501010000Z",
			Revision{Date: "0501010000Z", Description: "Test", Span: types.Span{Start: 20, End: 40}},
		),
	)

	d := validateAndFindDiagnostic(t, mod, types.VerboseConfig(), types.DiagDateYear2Digits)
	testutil.NotNil(t, d, "expected %s diagnostic for 2-digit year", types.DiagDateYear2Digits)
	testutil.Equal(t, types.SeverityWarning, d.Severity, "severity")
	testutil.True(t, strings.Contains(d.Message, "2005"), "message should mention interpreted year: %s", d.Message)
}

func TestValidate_DateInPast(t *testing.T) {
	// 1985 is before the SMI epoch (1990).
	mod := smiv2Module("DATE-TEST-MIB",
		testMI("dateTest", "198501010000Z",
			Revision{Date: "198501010000Z", Description: "Test", Span: types.Span{Start: 20, End: 40}},
		),
	)

	d := validateAndFindDiagnostic(t, mod, types.VerboseConfig(), types.DiagDateInPast)
	testutil.NotNil(t, d, "expected %s diagnostic for pre-1990 date", types.DiagDateInPast)
	testutil.Equal(t, types.SeverityStyle, d.Severity, "severity")
}

func TestValidate_DateInFuture(t *testing.T) {
	// Use a far-future date to avoid flakiness.
	mod := smiv2Module("DATE-TEST-MIB",
		testMI("dateTest", "209901010000Z",
			Revision{Date: "209901010000Z", Description: "Test", Span: types.Span{Start: 20, End: 40}},
		),
	)

	d := validateAndFindDiagnostic(t, mod, types.VerboseConfig(), types.DiagDateInFuture)
	testutil.NotNil(t, d, "expected %s diagnostic for future date", types.DiagDateInFuture)
	testutil.Equal(t, types.SeverityStyle, d.Severity, "severity")
}

func TestValidate_DateValid_NoDiagnostic(t *testing.T) {
	mod := smiv2Module("DATE-TEST-MIB",
		testMI("dateTest", "200501150000Z",
			Revision{Date: "200501150000Z", Description: "Test", Span: types.Span{Start: 20, End: 40}},
		),
	)

	ValidateModule(mod, nil, nil, types.VerboseConfig())

	codes := []string{
		types.DiagDateLength, types.DiagDateCharacter,
		types.DiagDateMonth, types.DiagDateDay,
		types.DiagDateHour, types.DiagDateMinutes,
		types.DiagDateValue, types.DiagDateInPast,
	}
	for _, code := range codes {
		var found *types.Diagnostic
		for _, d := range mod.Diagnostics {
			if d.Code == code {
				found = &d
				break
			}
		}
		testutil.Nil(t, found, "valid date should not trigger %s", code)
	}
}

// --- Revision ordering ---

func TestValidate_RevisionNotDescending(t *testing.T) {
	mod := smiv2Module("REV-TEST-MIB",
		testMI("revTest", "200501010000Z",
			Revision{Date: "200301010000Z", Description: "2003 first", Span: types.Span{Start: 20, End: 30}},
			Revision{Date: "200501010000Z", Description: "2005 second - wrong", Span: types.Span{Start: 30, End: 40}},
			Revision{Date: "200401010000Z", Description: "2004 third", Span: types.Span{Start: 40, End: 50}},
		),
	)

	d := validateAndFindDiagnostic(t, mod, types.VerboseConfig(), types.DiagRevisionNotDescending)
	testutil.NotNil(t, d, "expected %s diagnostic", types.DiagRevisionNotDescending)
	testutil.Equal(t, types.SeverityMinor, d.Severity, "severity")
}

func TestValidate_RevisionNotDescending_CorrectOrder(t *testing.T) {
	mod := smiv2Module("REV-TEST-MIB",
		testMI("revTest", "200501010000Z",
			Revision{Date: "200501010000Z", Description: "2005", Span: types.Span{Start: 20, End: 30}},
			Revision{Date: "200401010000Z", Description: "2004", Span: types.Span{Start: 30, End: 40}},
			Revision{Date: "200301010000Z", Description: "2003", Span: types.Span{Start: 40, End: 50}},
		),
	)

	d := validateAndFindDiagnostic(t, mod, types.VerboseConfig(), types.DiagRevisionNotDescending)
	testutil.Nil(t, d, "correct ordering should not trigger %s", types.DiagRevisionNotDescending)
}

func TestValidate_RevisionAfterUpdate(t *testing.T) {
	mod := smiv2Module("REV-TEST-MIB",
		testMI("revTest", "200401010000Z",
			Revision{Date: "200501010000Z", Description: "after LAST-UPDATED", Span: types.Span{Start: 20, End: 30}},
			Revision{Date: "200401010000Z", Description: "matches", Span: types.Span{Start: 30, End: 40}},
		),
	)

	d := validateAndFindDiagnostic(t, mod, types.VerboseConfig(), types.DiagRevisionAfterUpdate)
	testutil.NotNil(t, d, "expected %s diagnostic", types.DiagRevisionAfterUpdate)
	testutil.Equal(t, types.SeverityMinor, d.Severity, "severity")
	testutil.True(t, strings.Contains(d.Message, "200501010000Z"), "message should mention revision date: %s", d.Message)
}

func TestValidate_RevisionAfterUpdate_NoneAfter(t *testing.T) {
	mod := smiv2Module("REV-TEST-MIB",
		testMI("revTest", "200501010000Z",
			Revision{Date: "200501010000Z", Description: "matches", Span: types.Span{Start: 20, End: 30}},
			Revision{Date: "200401010000Z", Description: "before", Span: types.Span{Start: 30, End: 40}},
		),
	)

	d := validateAndFindDiagnostic(t, mod, types.VerboseConfig(), types.DiagRevisionAfterUpdate)
	testutil.Nil(t, d, "revision matching LAST-UPDATED should not trigger %s", types.DiagRevisionAfterUpdate)
}

// --- Revision LAST-UPDATED matching ---

func TestValidate_RevisionLastUpdated_Missing(t *testing.T) {
	mod := smiv2Module("REV-TEST-MIB",
		testMI("revTest", "200501010000Z",
			Revision{Date: "200401010000Z", Description: "no match", Span: types.Span{Start: 20, End: 30}},
		),
	)

	d := validateAndFindDiagnostic(t, mod, types.VerboseConfig(), types.DiagRevisionLastUpdated)
	testutil.NotNil(t, d, "expected %s diagnostic", types.DiagRevisionLastUpdated)
	testutil.Equal(t, types.SeverityMinor, d.Severity, "severity")
}

func TestValidate_RevisionLastUpdated_Present(t *testing.T) {
	mod := smiv2Module("REV-TEST-MIB",
		testMI("revTest", "200501010000Z",
			Revision{Date: "200501010000Z", Description: "matches", Span: types.Span{Start: 20, End: 30}},
		),
	)

	d := validateAndFindDiagnostic(t, mod, types.VerboseConfig(), types.DiagRevisionLastUpdated)
	testutil.Nil(t, d, "should not trigger %s when revision matches LAST-UPDATED", types.DiagRevisionLastUpdated)
}

// --- SMIv1 modules skip date checks ---

func TestValidate_DateChecks_SMIv1_Skipped(t *testing.T) {
	// SMIv1 modules don't have MODULE-IDENTITY, so no date checks run.
	mod := smiv1Module("SMIV1-MIB",
		testOT("testObj"),
	)

	for _, code := range []string{
		types.DiagDateLength, types.DiagDateCharacter,
		types.DiagRevisionNotDescending, types.DiagRevisionAfterUpdate,
	} {
		d := validateAndFindDiagnostic(t, mod, types.VerboseConfig(), code)
		testutil.Nil(t, d, "SMIv1 module should not get %s", code)
	}
}

// --- All macro types ---

func TestValidate_MacroNotImported_AllTypes(t *testing.T) {
	// Verify that all SMIv2 macro types are checked.
	macroTests := []struct {
		name  string
		macro string
		def   Definition
	}{
		{"OBJECT-IDENTITY", "OBJECT-IDENTITY", &ObjectIdentity{DefBase: DefBase{Name: "oi"}}},
		{"NOTIFICATION-TYPE", "NOTIFICATION-TYPE", &Notification{DefBase: DefBase{Name: "n"}}},
		{"TEXTUAL-CONVENTION", "TEXTUAL-CONVENTION", &TypeDef{DefBase: DefBase{Name: "tc"}, IsTextualConvention: true}},
		{"OBJECT-GROUP", "OBJECT-GROUP", &ObjectGroup{DefBase: DefBase{Name: "og"}}},
		{"NOTIFICATION-GROUP", "NOTIFICATION-GROUP", &NotificationGroup{DefBase: DefBase{Name: "ng"}}},
		{"MODULE-COMPLIANCE", "MODULE-COMPLIANCE", &ModuleCompliance{DefBase: DefBase{Name: "mc"}}},
		{"AGENT-CAPABILITIES", "AGENT-CAPABILITIES", &AgentCapabilities{DefBase: DefBase{Name: "ac"}}},
	}

	for _, tc := range macroTests {
		t.Run(tc.name, func(t *testing.T) {
			mod := smiv2Module("MACRO-CHECK-MIB",
				testMI("macroCheck", "200001010000Z",
					Revision{Date: "200001010000Z", Description: "Test", Span: types.Span{Start: 20, End: 40}},
				),
				tc.def,
			)

			d := validateAndFindDiagnostic(t, mod, types.VerboseConfig(), types.DiagMacroNotImported)
			testutil.NotNil(t, d, "expected %s diagnostic for %s", types.DiagMacroNotImported, tc.macro)
			testutil.True(t, strings.Contains(d.Message, tc.macro), "message should mention %s: %s", tc.macro, d.Message)
		})
	}
}

// --- TRAP-TYPE does not trigger macro import check ---

func TestValidate_MacroNotImported_TrapType_NoDiag(t *testing.T) {
	// TRAP-TYPE is SMIv1, so it should not trigger the macro import check
	// even if present in an SMIv2 module.
	mod := smiv2Module("TRAP-TEST-MIB",
		testMI("trapTest", "200001010000Z",
			Revision{Date: "200001010000Z", Description: "Test", Span: types.Span{Start: 20, End: 40}},
		),
		&Notification{
			DefBase:  DefBase{Name: "testTrap"},
			TrapInfo: &TrapInfo{Enterprise: "enterprises", TrapNumber: 1},
		},
	)

	// Should not get NOTIFICATION-TYPE import warning since it's a trap.
	d := validateAndFindDiagnostic(t, mod, types.VerboseConfig(), types.DiagMacroNotImported)
	testutil.Nil(t, d, "TRAP-TYPE should not trigger %s for NOTIFICATION-TYPE", types.DiagMacroNotImported)
}
