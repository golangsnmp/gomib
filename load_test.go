package gomib

import (
	"context"
	"testing"

	"github.com/golangsnmp/gomib/internal/testutil"
)

func TestLoadSingleMIB(t *testing.T) {
	src, err := DirTree("testdata/corpus/primary")
	if err != nil {
		t.Fatalf("DirTree failed: %v", err)
	}

	ctx := context.Background()
	mib, err := LoadModules(ctx, []string{"IF-MIB"}, src)
	if err != nil {
		t.Fatalf("LoadModules failed: %v", err)
	}

	// Basic sanity checks
	testutil.NotNil(t, mib, "mib should not be nil")
	testutil.Greater(t, mib.ModuleCount(), 0, "should have loaded modules")
	testutil.Greater(t, mib.ObjectCount(), 0, "should have resolved objects")

	// Check IF-MIB specifically
	ifMIB := mib.Module("IF-MIB")
	testutil.NotNil(t, ifMIB, "IF-MIB module should be found")

	// Check a well-known object
	ifIndex := mib.FindObject("ifIndex")
	testutil.NotNil(t, ifIndex, "ifIndex object should be found")
}

func TestLoadAllCorpus(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping corpus load in short mode")
	}

	src, err := DirTree("testdata/corpus/primary")
	if err != nil {
		t.Fatalf("DirTree failed: %v", err)
	}

	ctx := context.Background()
	mib, err := Load(ctx, src)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	testutil.Greater(t, mib.ModuleCount(), 50, "should have loaded many modules")
	testutil.Greater(t, mib.ObjectCount(), 1000, "should have resolved many objects")

	t.Logf("Loaded %d modules, %d objects, %d types",
		mib.ModuleCount(), mib.ObjectCount(), mib.TypeCount())
}

// === Strictness Tests ===

func TestStrictMIBsPassAtStrictLevel(t *testing.T) {
	corpus, err := DirTree("testdata/corpus/primary")
	if err != nil {
		t.Fatalf("DirTree corpus failed: %v", err)
	}
	strict, err := DirTree("testdata/strictness/strict")
	if err != nil {
		t.Fatalf("DirTree strict failed: %v", err)
	}
	src := Multi(corpus, strict)

	tests := []string{"STRICT-TEST-MIB", "STRICT-TABLE-MIB"}
	for _, name := range tests {
		t.Run(name, func(t *testing.T) {
			ctx := context.Background()
			mib, err := LoadModules(ctx, []string{name}, src, WithStrictness(StrictnessStrict))
			if err != nil {
				t.Fatalf("LoadModules failed: %v", err)
			}

			diags := mib.Diagnostics()
			if len(diags) > 0 {
				for _, d := range diags {
					t.Errorf("unexpected diagnostic: [%s] %s: %s", d.Code, d.Severity, d.Message)
				}
			}
		})
	}
}

func TestUnderscoreViolationEmitsDiagnostic(t *testing.T) {
	corpus, err := DirTree("testdata/corpus/primary")
	if err != nil {
		t.Fatalf("DirTree corpus failed: %v", err)
	}
	violations, err := DirTree("testdata/strictness/violations")
	if err != nil {
		t.Fatalf("DirTree violations failed: %v", err)
	}
	src := Multi(corpus, violations)

	ctx := context.Background()

	// In strict mode, should emit identifier-underscore diagnostics
	mib, err := LoadModules(ctx, []string{"UNDERSCORE-TEST-MIB"}, src, WithStrictness(StrictnessStrict))
	if err != nil {
		t.Fatalf("LoadModules failed: %v", err)
	}

	var underscoreDiags int
	for _, d := range mib.Diagnostics() {
		if d.Code == "identifier-underscore" {
			underscoreDiags++
		}
	}
	testutil.Equal(t, 2, underscoreDiags, "expected 2 identifier-underscore diagnostics")

	// In permissive mode, diagnostics should be suppressed
	mib, err = LoadModules(ctx, []string{"UNDERSCORE-TEST-MIB"}, src, WithStrictness(StrictnessPermissive))
	if err != nil {
		t.Fatalf("LoadModules failed: %v", err)
	}

	underscoreDiags = 0
	for _, d := range mib.Diagnostics() {
		if d.Code == "identifier-underscore" {
			underscoreDiags++
		}
	}
	testutil.Equal(t, 0, underscoreDiags, "expected no identifier-underscore diagnostics in permissive mode")
}

func TestHyphenEndViolationEmitsDiagnostic(t *testing.T) {
	corpus, err := DirTree("testdata/corpus/primary")
	if err != nil {
		t.Fatalf("DirTree corpus failed: %v", err)
	}
	violations, err := DirTree("testdata/strictness/violations")
	if err != nil {
		t.Fatalf("DirTree violations failed: %v", err)
	}
	src := Multi(corpus, violations)

	ctx := context.Background()
	mib, err := LoadModules(ctx, []string{"HYPHEN-END-TEST-MIB"}, src, WithStrictness(StrictnessStrict))
	if err != nil {
		t.Fatalf("LoadModules failed: %v", err)
	}

	var hyphenDiags int
	for _, d := range mib.Diagnostics() {
		if d.Code == "identifier-hyphen-end" {
			hyphenDiags++
		}
	}
	testutil.Equal(t, 1, hyphenDiags, "expected 1 identifier-hyphen-end diagnostic")
}

func TestLongIdentifierViolationEmitsDiagnostic(t *testing.T) {
	corpus, err := DirTree("testdata/corpus/primary")
	if err != nil {
		t.Fatalf("DirTree corpus failed: %v", err)
	}
	violations, err := DirTree("testdata/strictness/violations")
	if err != nil {
		t.Fatalf("DirTree violations failed: %v", err)
	}
	src := Multi(corpus, violations)

	ctx := context.Background()
	mib, err := LoadModules(ctx, []string{"LONG-IDENT-TEST-MIB"}, src, WithStrictness(StrictnessStrict))
	if err != nil {
		t.Fatalf("LoadModules failed: %v", err)
	}

	var lengthDiags int
	for _, d := range mib.Diagnostics() {
		if d.Code == "identifier-length-64" {
			lengthDiags++
		}
	}
	testutil.Equal(t, 1, lengthDiags, "expected 1 identifier-length-64 diagnostic")
}

func TestMissingModuleIdentityEmitsDiagnostic(t *testing.T) {
	corpus, err := DirTree("testdata/corpus/primary")
	if err != nil {
		t.Fatalf("DirTree corpus failed: %v", err)
	}
	violations, err := DirTree("testdata/strictness/violations")
	if err != nil {
		t.Fatalf("DirTree violations failed: %v", err)
	}
	src := Multi(corpus, violations)

	ctx := context.Background()
	mib, err := LoadModules(ctx, []string{"MISSING-IDENTITY-MIB"}, src, WithStrictness(StrictnessStrict))
	if err != nil {
		t.Fatalf("LoadModules failed: %v", err)
	}

	var identityDiags int
	for _, d := range mib.Diagnostics() {
		if d.Code == "missing-module-identity" {
			identityDiags++
		}
	}
	testutil.Equal(t, 1, identityDiags, "expected 1 missing-module-identity diagnostic")
}
