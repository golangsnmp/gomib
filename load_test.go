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

	testutil.NotNil(t, mib, "mib should not be nil")
	testutil.Greater(t, mib.ModuleCount(), 0, "should have loaded modules")
	testutil.Greater(t, mib.ObjectCount(), 0, "should have resolved objects")

	ifMIB := mib.Module("IF-MIB")
	testutil.NotNil(t, ifMIB, "IF-MIB module should be found")

	ifIndex := mib.Object("ifIndex")
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

func TestDirSource(t *testing.T) {
	src, err := Dir("testdata/corpus/primary/ietf")
	if err != nil {
		t.Fatalf("Dir failed: %v", err)
	}

	names, err := src.ListModules()
	if err != nil {
		t.Fatalf("ListModules failed: %v", err)
	}
	testutil.Greater(t, len(names), 0, "should list modules")
}

func TestDirTreeSource(t *testing.T) {
	src, err := DirTree("testdata/corpus/primary")
	if err != nil {
		t.Fatalf("DirTree failed: %v", err)
	}

	names, err := src.ListModules()
	if err != nil {
		t.Fatalf("ListModules failed: %v", err)
	}
	testutil.Greater(t, len(names), 10, "should list many modules recursively")
}

func TestMultiSource(t *testing.T) {
	primary, err := DirTree("testdata/corpus/primary")
	if err != nil {
		t.Fatalf("DirTree primary failed: %v", err)
	}
	problems, err := DirTree("testdata/corpus/problems")
	if err != nil {
		t.Fatalf("DirTree problems failed: %v", err)
	}

	src := Multi(primary, problems)

	ctx := context.Background()
	m, err := LoadModules(ctx, []string{"IF-MIB"}, src)
	if err != nil {
		t.Fatalf("LoadModules from multi source failed: %v", err)
	}
	testutil.NotNil(t, m.Module("IF-MIB"), "should find IF-MIB from primary source")
}

func TestLoadNonexistentModule(t *testing.T) {
	src, err := DirTree("testdata/corpus/primary")
	if err != nil {
		t.Fatalf("DirTree failed: %v", err)
	}

	ctx := context.Background()
	m, err := LoadModules(ctx, []string{"TOTALLY-FAKE-MIB-THAT-DOES-NOT-EXIST"}, src)
	testutil.NoError(t, err, "LoadModules should not error for nonexistent module")
	testutil.NotNil(t, m, "LoadModules should return a Mib even for nonexistent module")

	mod := m.Module("TOTALLY-FAKE-MIB-THAT-DOES-NOT-EXIST")
	testutil.Nil(t, mod, "nonexistent module should not be in the result")
}

func TestLoadNoSources(t *testing.T) {
	ctx := context.Background()
	_, err := Load(ctx, nil)
	testutil.Error(t, err, "loading with nil source should fail")
}

func TestNodeByName(t *testing.T) {
	m := loadTestMIB(t)

	node := m.Node("ifIndex")
	testutil.NotNil(t, node, "should find ifIndex by name")
}

func TestNodeByNumericOID(t *testing.T) {
	m := loadTestMIB(t)

	node := m.Node("1.3.6.1.2.1.2.2.1.1")
	testutil.NotNil(t, node, "Node(1.3.6.1.2.1.2.2.1.1)")
	testutil.Equal(t, "ifIndex", node.Name(), "node found by OID should be ifIndex")
}

func TestNodeByDottedOID(t *testing.T) {
	m := loadTestMIB(t)

	node := m.Node(".1.3.6.1.2.1.2.2.1.1")
	testutil.NotNil(t, node, "Node(.1.3.6.1.2.1.2.2.1.1)")
	testutil.Equal(t, "ifIndex", node.Name(), "node found by dotted OID should be ifIndex")
}

func TestNodeByQualifiedName(t *testing.T) {
	m := loadTestMIB(t)

	node := m.Node("IF-MIB::ifIndex")
	testutil.NotNil(t, node, "Node(IF-MIB::ifIndex)")
	testutil.Equal(t, "ifIndex", node.Name(), "node found by qualified name should be ifIndex")
}

func TestNodeNotFound(t *testing.T) {
	m := loadTestMIB(t)

	node := m.Node("totallyNonExistentSymbol")
	testutil.Nil(t, node, "nonexistent symbol should return nil")
}

func TestObjectByName(t *testing.T) {
	m := loadTestMIB(t)

	obj := m.Object("sysDescr")
	testutil.NotNil(t, obj, "should find sysDescr by name")
	testutil.Equal(t, "sysDescr", obj.Name(), "object name")
}

func TestObjectByQualifiedName(t *testing.T) {
	m := loadTestMIB(t)

	obj := m.Object("SNMPv2-MIB::sysDescr")
	testutil.NotNil(t, obj, "Object(SNMPv2-MIB::sysDescr)")
	testutil.Equal(t, "sysDescr", obj.Name(), "qualified object name")
}

func TestType(t *testing.T) {
	m := loadTestMIB(t)

	typ := m.Type("DisplayString")
	testutil.NotNil(t, typ, "Type(DisplayString)")
	testutil.Equal(t, "DisplayString", typ.Name(), "type name")
	testutil.True(t, typ.IsTextualConvention(), "DisplayString should be a TC")
}

func TestNotification(t *testing.T) {
	m := loadTestMIB(t)

	notif := m.Notification("linkDown")
	testutil.NotNil(t, notif, "Notification(linkDown)")
	testutil.Equal(t, "linkDown", notif.Name(), "notification name")
}

func TestModulesCollection(t *testing.T) {
	m := loadTestMIB(t)

	mods := m.Modules()
	testutil.Greater(t, len(mods), 0, "should have modules")

	found := false
	for _, mod := range mods {
		if mod.Name() == "IF-MIB" {
			found = true
			break
		}
	}
	testutil.True(t, found, "should find IF-MIB in modules list")
}

func TestNodesIteration(t *testing.T) {
	m := loadTestMIB(t)

	count := 0
	for range m.Nodes() {
		count++
	}
	testutil.Greater(t, count, 0, "should have nodes")
	testutil.Equal(t, m.NodeCount(), count, "NodeCount should match iteration count")
}

func TestObjectsCollection(t *testing.T) {
	m := loadTestMIB(t)

	objs := m.Objects()
	testutil.Equal(t, m.ObjectCount(), len(objs), "ObjectCount should match Objects() length")
}

func TestTablesAndScalars(t *testing.T) {
	m := loadTestMIB(t)

	tables := m.Tables()
	scalars := m.Scalars()

	testutil.Greater(t, len(tables), 0, "should have tables (IF-MIB has ifTable)")
	testutil.Greater(t, len(scalars), 0, "should have scalars (SNMPv2-MIB has sysDescr)")
}

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

func TestUppercaseIdentifierEmitsDiagnostic(t *testing.T) {
	corpus, err := DirTree("testdata/corpus/primary")
	if err != nil {
		t.Fatalf("DirTree corpus failed: %v", err)
	}
	problems, err := DirTree("testdata/corpus/problems")
	if err != nil {
		t.Fatalf("DirTree problems failed: %v", err)
	}
	src := Multi(corpus, problems)

	ctx := context.Background()

	m, err := LoadModules(ctx, []string{"PROBLEM-NAMING-MIB"}, src, WithStrictness(StrictnessNormal))
	if err != nil {
		t.Fatalf("LoadModules failed: %v", err)
	}

	var caseDiags int
	for _, d := range m.Diagnostics() {
		if d.Code == "bad-identifier-case" {
			caseDiags++
		}
	}
	testutil.Equal(t, 4, caseDiags, "expected 4 bad-identifier-case diagnostics in normal mode")

	node := m.Node("NetEngine8000SysOid")
	testutil.NotNil(t, node, "uppercase identifier should resolve in normal mode")

	m, err = LoadModules(ctx, []string{"PROBLEM-NAMING-MIB"}, src, WithStrictness(StrictnessPermissive))
	if err != nil {
		t.Fatalf("LoadModules failed: %v", err)
	}

	caseDiags = 0
	for _, d := range m.Diagnostics() {
		if d.Code == "bad-identifier-case" {
			caseDiags++
		}
	}
	testutil.Equal(t, 0, caseDiags, "expected no bad-identifier-case diagnostics in permissive mode")
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

func TestMissingImportFailsInStrictMode(t *testing.T) {
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

	mib, err := LoadModules(ctx, []string{"MISSING-IMPORT-TEST-MIB"}, src, WithStrictness(StrictnessStrict))
	if err != nil {
		t.Fatalf("LoadModules failed: %v", err)
	}

	unresolved := mib.Unresolved()
	var oidUnresolved int
	for _, u := range unresolved {
		if u.Kind == UnresolvedOID {
			oidUnresolved++
		}
	}
	testutil.Greater(t, oidUnresolved, 0, "strict mode should have unresolved OID references")

	testObj := mib.Object("testObject")
	testutil.Nil(t, testObj, "testObject should not resolve in strict mode")
}

func TestMissingImportWorksInPermissiveMode(t *testing.T) {
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

	mib, err := LoadModules(ctx, []string{"MISSING-IMPORT-TEST-MIB"}, src, WithStrictness(StrictnessPermissive))
	if err != nil {
		t.Fatalf("LoadModules failed: %v", err)
	}

	unresolved := mib.Unresolved()
	var oidUnresolved int
	for _, u := range unresolved {
		if u.Kind == UnresolvedOID && u.Module == "MISSING-IMPORT-TEST-MIB" {
			oidUnresolved++
		}
	}
	testutil.Equal(t, 0, oidUnresolved, "permissive mode should resolve enterprises via fallback")

	testObj := mib.Object("testObject")
	testutil.NotNil(t, testObj, "testObject should resolve in permissive mode")
}

func TestMissingImportFailsInNormalMode(t *testing.T) {
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

	mib, err := LoadModules(ctx, []string{"MISSING-IMPORT-TEST-MIB"}, src, WithStrictness(StrictnessNormal))
	if err != nil {
		t.Fatalf("LoadModules failed: %v", err)
	}

	unresolved := mib.Unresolved()
	var oidUnresolved int
	for _, u := range unresolved {
		if u.Kind == UnresolvedOID && u.Module == "MISSING-IMPORT-TEST-MIB" {
			oidUnresolved++
		}
	}
	testutil.Greater(t, oidUnresolved, 0, "normal mode should have unresolved OID references")
}

func loadInvalidMIB(t testing.TB, name string, level StrictnessLevel) Mib {
	t.Helper()
	corpus, err := DirTree("testdata/corpus/primary")
	if err != nil {
		t.Fatalf("DirTree corpus failed: %v", err)
	}
	invalid, err := DirTree("testdata/strictness/invalid")
	if err != nil {
		t.Fatalf("DirTree invalid failed: %v", err)
	}
	src := Multi(corpus, invalid)
	ctx := context.Background()
	m, err := LoadModules(ctx, []string{name}, src, WithStrictness(level))
	if err != nil {
		t.Fatalf("LoadModules(%s) failed: %v", name, err)
	}
	return m
}

func moduleObjects(m Mib, moduleName string) []Object {
	var result []Object
	for _, o := range m.Objects() {
		if o.Module().Name() == moduleName {
			result = append(result, o)
		}
	}
	return result
}

func moduleDiagnostics(m Mib, moduleName string) []Diagnostic {
	var result []Diagnostic
	for _, d := range m.Diagnostics() {
		if d.Module == moduleName {
			result = append(result, d)
		}
	}
	return result
}

func TestInvalidSyntaxMIBProducesNoBrokenObjects(t *testing.T) {
	levels := []struct {
		name  string
		level StrictnessLevel
	}{
		{"strict", StrictnessStrict},
		{"normal", StrictnessNormal},
		{"permissive", StrictnessPermissive},
	}

	for _, lvl := range levels {
		t.Run(lvl.name, func(t *testing.T) {
			m := loadInvalidMIB(t, "INVALID-SYNTAX-MIB", lvl.level)

			objs := moduleObjects(m, "INVALID-SYNTAX-MIB")
			testutil.Equal(t, 0, len(objs),
				"broken OBJECT-TYPE should not produce objects at %s level", lvl.name)

			diags := moduleDiagnostics(m, "INVALID-SYNTAX-MIB")
			testutil.Greater(t, len(diags), 0,
				"should emit diagnostic for missing SYNTAX at %s level", lvl.name)
		})
	}
}

func TestInvalidTruncatedMIBProducesNoObjects(t *testing.T) {
	levels := []struct {
		name  string
		level StrictnessLevel
	}{
		{"strict", StrictnessStrict},
		{"normal", StrictnessNormal},
		{"permissive", StrictnessPermissive},
	}

	for _, lvl := range levels {
		t.Run(lvl.name, func(t *testing.T) {
			m := loadInvalidMIB(t, "INVALID-TRUNCATED-MIB", lvl.level)

			objs := moduleObjects(m, "INVALID-TRUNCATED-MIB")
			testutil.Equal(t, 0, len(objs),
				"truncated OBJECT-TYPE should not produce objects at %s level", lvl.name)

			diags := moduleDiagnostics(m, "INVALID-TRUNCATED-MIB")
			testutil.Greater(t, len(diags), 0,
				"should emit diagnostic for truncated definition at %s level", lvl.name)
		})
	}
}

func TestInvalidDuplicateOIDMIBBothObjectsLoad(t *testing.T) {
	m := loadInvalidMIB(t, "INVALID-DUPLICATE-OID-MIB", StrictnessPermissive)

	objs := moduleObjects(m, "INVALID-DUPLICATE-OID-MIB")
	testutil.Equal(t, 2, len(objs),
		"both duplicate-OID objects should load")

	if len(objs) == 2 {
		testutil.Equal(t, objs[0].OID().String(), objs[1].OID().String(),
			"duplicate objects should share the same OID")
	}
}
