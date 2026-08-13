package lower_test

import (
	"fmt"
	"testing"

	"github.com/golangsnmp/gomib/internal/module"
	"github.com/golangsnmp/gomib/internal/types"
)

func parseAndValidateLanguageModule(t *testing.T, source string) *module.Module {
	t.Helper()
	mods := parseMods(t, source)
	if len(mods) != 1 {
		t.Fatalf("got %d modules, want 1", len(mods))
	}
	module.ValidateModule(mods[0], []byte(source), nil, types.VerboseConfig())
	return mods[0]
}

func moduleHasDiagnostic(mod *module.Module, code string) bool {
	for _, diag := range mod.Diagnostics {
		if diag.Code == code {
			return true
		}
	}
	return false
}

func TestLanguageModuleIdentityWithoutImportsIsSMIv2(t *testing.T) {
	const source = `
BROKEN-MIB DEFINITIONS ::= BEGIN

brokenMIB MODULE-IDENTITY
    LAST-UPDATED "200901080000Z"
    ORGANIZATION "Test"
    CONTACT-INFO "test"
    DESCRIPTION "test module"
    REVISION "200901080000Z"
    DESCRIPTION "Initial version"
    ::= { enterprises 99999 }

END
`
	mod := parseAndValidateLanguageModule(t, source)
	if mod.Language != types.LanguageSMIv2 {
		t.Fatalf("language = %v, want %v", mod.Language, types.LanguageSMIv2)
	}
	if !moduleHasDiagnostic(mod, types.DiagMacroNotImported) {
		t.Error("missing SMIv2 macro import diagnostic")
	}
}

func TestLanguageModuleIdentityWithBrokenImportIsSMIv2(t *testing.T) {
	const source = `
BROKEN DEFINITIONS ::= BEGIN
IMPORTS MODULE-IDENTITY FROM MISSING-SMI;

brokenMIB MODULE-IDENTITY
    LAST-UPDATED "200901080000Z"
    ORGANIZATION "Test"
    CONTACT-INFO "test"
    DESCRIPTION "test module"
    ::= { enterprises 99999 }

END
`
	mod := parseAndValidateLanguageModule(t, source)
	if mod.Language != types.LanguageSMIv2 {
		t.Fatalf("language = %v, want %v", mod.Language, types.LanguageSMIv2)
	}
	if moduleHasDiagnostic(mod, types.DiagMacroNotImported) {
		t.Error("imported MODULE-IDENTITY should satisfy SMIv2 macro import validation")
	}
	if !moduleHasDiagnostic(mod, types.DiagModuleNameSuffix) {
		t.Error("missing SMIv2 module name validation diagnostic")
	}
}

func TestLanguageSMIv1ImportOnlyIsSMIv1(t *testing.T) {
	const source = `
IMPORT-V1-MIB DEFINITIONS ::= BEGIN
IMPORTS enterprises FROM RFC1155-SMI;
END
`
	mod := parseAndValidateLanguageModule(t, source)
	if mod.Language != types.LanguageSMIv1 {
		t.Fatalf("language = %v, want %v", mod.Language, types.LanguageSMIv1)
	}
}

func TestLanguageSMIv2ImportOnlyIsSMIv2(t *testing.T) {
	const source = `
IMPORT-V2-MIB DEFINITIONS ::= BEGIN
IMPORTS enterprises FROM SNMPv2-SMI;
END
`
	mod := parseAndValidateLanguageModule(t, source)
	if mod.Language != types.LanguageSMIv2 {
		t.Fatalf("language = %v, want %v", mod.Language, types.LanguageSMIv2)
	}
}

func TestLanguageConflictingImportsAreUnknown(t *testing.T) {
	const source = `
CONFLICTING-IMPORTS-MIB DEFINITIONS ::= BEGIN
IMPORTS
    enterprises FROM RFC1155-SMI
    MODULE-IDENTITY FROM SNMPv2-SMI;
END
`
	mod := parseAndValidateLanguageModule(t, source)
	if mod.Language != types.LanguageUnknown {
		t.Fatalf("language = %v, want %v", mod.Language, types.LanguageUnknown)
	}
}

func TestLanguageTrapTypeOnlyIsSMIv1(t *testing.T) {
	const source = `
TRAP-MIB DEFINITIONS ::= BEGIN

testTrap TRAP-TYPE
    ENTERPRISE enterprises
    DESCRIPTION "test trap"
    ::= 1

END
`
	mod := parseAndValidateLanguageModule(t, source)
	if mod.Language != types.LanguageSMIv1 {
		t.Fatalf("language = %v, want %v", mod.Language, types.LanguageSMIv1)
	}
}

func TestLanguageConflictingEvidenceIsUnknown(t *testing.T) {
	const source = `
HYBRID-MIB DEFINITIONS ::= BEGIN
IMPORTS enterprises FROM RFC1155-SMI;

hybridMIB MODULE-IDENTITY
    LAST-UPDATED "200901080000Z"
    ORGANIZATION "Test"
    CONTACT-INFO "test"
    DESCRIPTION "test module"
    ::= { enterprises 99999 }

END
`
	mod := parseAndValidateLanguageModule(t, source)
	if mod.Language != types.LanguageUnknown {
		t.Fatalf("language = %v, want %v", mod.Language, types.LanguageUnknown)
	}
	if moduleHasDiagnostic(mod, types.DiagMacroNotImported) {
		t.Error("conflicting evidence should not run SMIv2-specific validation")
	}
}

func TestLanguageHybridStrongSyntaxIsUnknown(t *testing.T) {
	const source = `
HYBRID-MIB DEFINITIONS ::= BEGIN

hybridMIB MODULE-IDENTITY
    LAST-UPDATED "200901080000Z"
    ORGANIZATION "Test"
    CONTACT-INFO "test"
    DESCRIPTION "test module"
    ::= { enterprises 99999 }

hybridTrap TRAP-TYPE
    ENTERPRISE enterprises
    VARIABLES { hybridMIB }
    DESCRIPTION "hybrid trap"
    ::= 1

END
`
	mod := parseAndValidateLanguageModule(t, source)
	if mod.Language != types.LanguageUnknown {
		t.Fatalf("language = %v, want %v", mod.Language, types.LanguageUnknown)
	}
	if moduleHasDiagnostic(mod, types.DiagMacroNotImported) {
		t.Error("hybrid syntax should not run SMIv2-specific validation")
	}
}

func TestLanguageSMIv2AssociatedSyntaxIsInsufficientEvidence(t *testing.T) {
	for _, test := range []struct {
		name       string
		definition string
	}{
		{
			name: "OBJECT-TYPE with MAX-ACCESS",
			definition: `testObject OBJECT-TYPE
    SYNTAX INTEGER
    MAX-ACCESS read-only
    STATUS current
    DESCRIPTION "test object"
    ::= { iso 3 }`,
		},
		{
			name: "OBJECT-IDENTITY",
			definition: `testIdentity OBJECT-IDENTITY
    STATUS current
    DESCRIPTION "test identity"
    ::= { iso 3 }`,
		},
		{
			name: "NOTIFICATION-TYPE",
			definition: `testNotification NOTIFICATION-TYPE
    STATUS current
    DESCRIPTION "test notification"
    ::= { iso 3 }`,
		},
		{
			name: "TEXTUAL-CONVENTION",
			definition: `TestConvention ::= TEXTUAL-CONVENTION
    STATUS current
    DESCRIPTION "test textual convention"
    SYNTAX INTEGER`,
		},
		{
			name: "OBJECT-GROUP",
			definition: `testObjectGroup OBJECT-GROUP
    OBJECTS { testObject }
    STATUS current
    DESCRIPTION "test object group"
    ::= { iso 3 }`,
		},
		{
			name: "NOTIFICATION-GROUP",
			definition: `testNotificationGroup NOTIFICATION-GROUP
    NOTIFICATIONS { testNotification }
    STATUS current
    DESCRIPTION "test notification group"
    ::= { iso 3 }`,
		},
		{
			name: "MODULE-COMPLIANCE",
			definition: `testCompliance MODULE-COMPLIANCE
    STATUS current
    DESCRIPTION "test compliance"
    MODULE
        MANDATORY-GROUPS { testObjectGroup }
    ::= { iso 3 }`,
		},
		{
			name: "AGENT-CAPABILITIES",
			definition: `testCapabilities AGENT-CAPABILITIES
    PRODUCT-RELEASE "test"
    STATUS current
    DESCRIPTION "test capabilities"
    SUPPORTS TEST-MIB
        INCLUDES { testObjectGroup }
    ::= { iso 3 }`,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			source := fmt.Sprintf("UNKNOWN-MIB DEFINITIONS ::= BEGIN\n%s\nEND\n", test.definition)
			mod := parseAndValidateLanguageModule(t, source)
			definitionCount := 0
			for range mod.AllDefinitions() {
				definitionCount++
			}
			if definitionCount == 0 {
				t.Fatal("test construct was not lowered")
			}
			if mod.Language != types.LanguageUnknown {
				t.Fatalf("language = %v, want %v", mod.Language, types.LanguageUnknown)
			}
		})
	}
}

func TestLanguageBaseModulesAreExplicit(t *testing.T) {
	for _, test := range []struct {
		name string
		want types.Language
	}{
		{name: "SNMPv2-SMI", want: types.LanguageSMIv2},
		{name: "SNMPv2-TC", want: types.LanguageSMIv2},
		{name: "SNMPv2-CONF", want: types.LanguageSMIv2},
		{name: "RFC1155-SMI", want: types.LanguageSMIv1},
		{name: "RFC1065-SMI", want: types.LanguageSMIv1},
		{name: "RFC-1212", want: types.LanguageSMIv1},
		{name: "RFC-1215", want: types.LanguageSMIv1},
	} {
		t.Run(test.name, func(t *testing.T) {
			mod := parseAndValidateLanguageModule(t, test.name+" DEFINITIONS ::= BEGIN\nEND\n")
			if mod.Language != test.want {
				t.Fatalf("language = %v, want %v", mod.Language, test.want)
			}
		})
	}
}
