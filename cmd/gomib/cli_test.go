package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const testCLIMIB = `TEST-MIB DEFINITIONS ::= BEGIN
IMPORTS
    MODULE-IDENTITY, OBJECT-TYPE, NOTIFICATION-TYPE, enterprises, Integer32
        FROM SNMPv2-SMI
    OBJECT-GROUP, MODULE-COMPLIANCE, AGENT-CAPABILITIES
        FROM SNMPv2-CONF;

testMib MODULE-IDENTITY
    LAST-UPDATED "202603150000Z"
    ORGANIZATION "Test"
    CONTACT-INFO "Test"
    DESCRIPTION "Test module."
    ::= { enterprises 99999 }

testScalar OBJECT-TYPE
    SYNTAX Integer32
    MAX-ACCESS read-only
    STATUS current
    DESCRIPTION "A test scalar."
    ::= { testMib 1 }

testNotification NOTIFICATION-TYPE
    OBJECTS { testScalar }
    STATUS current
    DESCRIPTION "A test notification."
    ::= { testMib 0 1 }

testBare OBJECT IDENTIFIER ::= { testMib 2 }

testGroup OBJECT-GROUP
    OBJECTS { testScalar }
    STATUS current
    DESCRIPTION "A test group."
    ::= { testMib 3 }

testCompliance MODULE-COMPLIANCE
    STATUS current
    DESCRIPTION "A test compliance statement."
    MODULE
        MANDATORY-GROUPS { testGroup }
    ::= { testMib 4 }

testCapabilities AGENT-CAPABILITIES
    PRODUCT-RELEASE "test"
    STATUS current
    DESCRIPTION "Test capabilities."
    SUPPORTS TEST-MIB
        INCLUDES { testGroup }
    ::= { testMib 5 }

END
`

func TestLoadWithoutModulesLoadsAll(t *testing.T) {
	dir := writeTestMIB(t)

	code, stdout, stderr := runCLI(t, "-p", dir, "load")
	if code != exitOK {
		t.Fatalf("load exited %d, stderr=%q", code, stderr)
	}
	if !strings.Contains(stdout, "Loaded ") || !strings.Contains(stdout, "1 objects, 1 notifications") {
		t.Fatalf("expected load output to show successful all-module load, got %q", stdout)
	}
}

func TestGetMaxDepthImpliesTree(t *testing.T) {
	dir := writeTestMIB(t)

	code, stdout, stderr := runCLI(t, "-p", dir, "get", "-m", "TEST-MIB", "--max-depth", "0", "testMib")
	if code != exitOK {
		t.Fatalf("get exited %d, stderr=%q", code, stderr)
	}
	if strings.Contains(stdout, "Name:") {
		t.Fatalf("expected tree output, got detail output %q", stdout)
	}
	if !strings.Contains(stdout, "TEST-MIB::testMib") {
		t.Fatalf("expected tree output to contain module-qualified node, got %q", stdout)
	}
}

func TestGetRejectsMultiplePositionalArgs(t *testing.T) {
	dir := writeTestMIB(t)

	code, _, stderr := runCLI(t, "-p", dir, "get", "testMib", "testScalar")
	if code != exitError {
		t.Fatalf("expected exit %d, got %d", exitError, code)
	}
	if !strings.Contains(stderr, "too many arguments") {
		t.Fatalf("expected too-many-args error, got %q", stderr)
	}
}

func TestGetKeyValueOutput(t *testing.T) {
	dir := writeTestMIB(t)

	code, stdout, stderr := runCLI(t, "-p", dir, "get", "-m", "TEST-MIB", "testScalar")
	if code != exitOK {
		t.Fatalf("get exited %d, stderr=%q", code, stderr)
	}
	for _, field := range []string{"Name:", "Module:", "OID:", "Kind:", "Access:", "Status:"} {
		if !strings.Contains(stdout, field) {
			t.Fatalf("expected key-value field %q in output, got %q", field, stdout)
		}
	}
}

func TestGetNotFoundExitsIssue(t *testing.T) {
	dir := writeTestMIB(t)

	code, _, _ := runCLI(t, "-p", dir, "get", "-m", "TEST-MIB", "nonExistentOID")
	if code != exitIssue {
		t.Fatalf("expected exit %d for not found, got %d", exitIssue, code)
	}
}

func TestLoadRejectsStrictAndPermissiveTogether(t *testing.T) {
	dir := writeTestMIB(t)

	code, _, stderr := runCLI(t, "-p", dir, "load", "--strict", "--permissive", "TEST-MIB")
	if code != exitError {
		t.Fatalf("expected exit %d, got %d", exitError, code)
	}
	if !strings.Contains(stderr, "mutually exclusive") {
		t.Fatalf("expected exclusivity error, got %q", stderr)
	}
}

func TestFindKindNotificationIncludesNotifications(t *testing.T) {
	dir := writeTestMIB(t)

	code, stdout, stderr := runCLI(t, "-p", dir, "find", "-m", "TEST-MIB", "--kind", "notification", "testNotification")
	if code != exitOK {
		t.Fatalf("find exited %d, stderr=%q", code, stderr)
	}
	if !strings.Contains(stdout, "TEST-MIB::testNotification") {
		t.Fatalf("expected notification in output, got %q", stdout)
	}
}

func TestFindUsesModuleFlag(t *testing.T) {
	dir := writeTestMIB(t)

	code, stdout, stderr := runCLI(t, "-p", dir, "find", "-m", "TEST-MIB", "testScalar")
	if code != exitOK {
		t.Fatalf("find exited %d, stderr=%q", code, stderr)
	}
	if !strings.Contains(stdout, "TEST-MIB::testScalar") {
		t.Fatalf("expected object in output, got %q", stdout)
	}
}

func TestFindNoMatchExitsIssue(t *testing.T) {
	dir := writeTestMIB(t)

	code, _, _ := runCLI(t, "-p", dir, "find", "-m", "TEST-MIB", "zzz_nonexistent_*")
	if code != exitIssue {
		t.Fatalf("expected exit %d for no matches, got %d", exitIssue, code)
	}
}

func TestFindTypeExcludesEntitiesWithoutTypesText(t *testing.T) {
	dir := writeTestMIB(t)

	code, stdout, stderr := runCLI(t, "-p", dir, "find", "-m", "TEST-MIB", "--type", "Integer32", "test*")
	if code != exitOK {
		t.Fatalf("find --type exited %d, stderr=%q", code, stderr)
	}
	want := "TEST-MIB::testScalar  1.3.6.1.4.1.99999.1  scalar\n"
	if stdout != want {
		t.Fatalf("unexpected text output:\n got: %q\nwant: %q", stdout, want)
	}
}

func TestFindTypeExcludesEntitiesWithoutTypesJSON(t *testing.T) {
	dir := writeTestMIB(t)

	code, stdout, stderr := runCLI(t, "-p", dir, "find", "-m", "TEST-MIB", "--type", "Integer32", "--format", "json", "test*")
	if code != exitOK {
		t.Fatalf("find --type --format json exited %d, stderr=%q", code, stderr)
	}
	var matches []findMatch
	if err := json.Unmarshal([]byte(stdout), &matches); err != nil {
		t.Fatalf("decode JSON output %q: %v", stdout, err)
	}
	want := []findMatch{{Name: "testScalar", Module: "TEST-MIB", OID: "1.3.6.1.4.1.99999.1", Kind: "scalar"}}
	if len(matches) != len(want) || matches[0] != want[0] {
		t.Fatalf("unexpected JSON matches: got %#v, want %#v", matches, want)
	}
}

func TestFindTypeExcludesEntitiesWithoutTypesCount(t *testing.T) {
	dir := writeTestMIB(t)

	code, stdout, stderr := runCLI(t, "-p", dir, "find", "-m", "TEST-MIB", "--type", "Integer32", "--count", "test*")
	if code != exitOK {
		t.Fatalf("find --type --count exited %d, stderr=%q", code, stderr)
	}
	if stdout != "1\n" {
		t.Fatalf("expected count 1, got %q", stdout)
	}
}

func TestFindExpandedKinds(t *testing.T) {
	dir := writeTestMIB(t)

	// module-identity kind should match the testMib MODULE-IDENTITY node
	code, stdout, stderr := runCLI(t, "-p", dir, "find", "-m", "TEST-MIB", "--kind", "node", "testMib")
	if code != exitOK {
		t.Fatalf("find --kind node exited %d, stderr=%q", code, stderr)
	}
	if !strings.Contains(stdout, "testMib") {
		t.Fatalf("expected module-identity node in output, got %q", stdout)
	}
}

func TestPathsReturnsErrorWhenNoPathsFound(t *testing.T) {
	code, _, stderr := runCLIWithSystemPaths(t, func() []string { return nil }, "paths")
	if code != exitIssue {
		t.Fatalf("expected exit %d, got %d", exitIssue, code)
	}
	if !strings.Contains(stderr, "no search paths found") {
		t.Fatalf("expected no-paths error, got %q", stderr)
	}
}

func TestPathsShowsBothCustomAndSystem(t *testing.T) {
	code, stdout, stderr := runCLIWithSystemPaths(t, func() []string { return []string{"/sys/path"} }, "-p", "/custom/path", "paths")
	if code != exitOK {
		t.Fatalf("paths exited %d, stderr=%q", code, stderr)
	}
	if !strings.Contains(stdout, "/custom/path (custom)") {
		t.Fatalf("expected custom path with annotation, got %q", stdout)
	}
	if !strings.Contains(stdout, "/sys/path (system)") {
		t.Fatalf("expected system path with annotation, got %q", stdout)
	}
}

func TestVersionFlagShowsVersion(t *testing.T) {
	code, stdout, stderr := runCLI(t, "--version")
	if code != exitOK {
		t.Fatalf("version exited %d, stderr=%q", code, stderr)
	}
	if !strings.HasPrefix(stdout, "gomib ") {
		t.Fatalf("expected version output, got %q", stdout)
	}
}

// --- list command ---

func TestListShowsModuleName(t *testing.T) {
	dir := writeTestMIB(t)

	code, stdout, stderr := runCLI(t, "-p", dir, "list")
	if code != exitOK {
		t.Fatalf("list exited %d, stderr=%q", code, stderr)
	}
	if !strings.Contains(stdout, "TEST-MIB") {
		t.Fatalf("expected TEST-MIB in list output, got %q", stdout)
	}
}

func TestListCount(t *testing.T) {
	dir := writeTestMIB(t)

	code, stdout, stderr := runCLI(t, "-p", dir, "list", "--count")
	if code != exitOK {
		t.Fatalf("list --count exited %d, stderr=%q", code, stderr)
	}
	if strings.TrimSpace(stdout) != "1" {
		t.Fatalf("expected count 1, got %q", stdout)
	}
}

func TestListFormatJSON(t *testing.T) {
	dir := writeTestMIB(t)

	code, stdout, stderr := runCLI(t, "-p", dir, "list", "--format", "json")
	if code != exitOK {
		t.Fatalf("list --format json exited %d, stderr=%q", code, stderr)
	}
	if !strings.Contains(stdout, `"TEST-MIB"`) {
		t.Fatalf("expected JSON with TEST-MIB, got %q", stdout)
	}
}

// --- dump command ---

func TestDumpBasicJSON(t *testing.T) {
	dir := writeTestMIB(t)

	code, stdout, stderr := runCLI(t, "-p", dir, "dump", "TEST-MIB")
	if code != exitOK {
		t.Fatalf("dump exited %d, stderr=%q", code, stderr)
	}
	for _, field := range []string{`"modules"`, `"objects"`, `"nodes"`, `"testScalar"`} {
		if !strings.Contains(stdout, field) {
			t.Fatalf("expected %s in dump output, got %q", field, stdout)
		}
	}
}

func TestDumpCompact(t *testing.T) {
	dir := writeTestMIB(t)

	code, stdout, stderr := runCLI(t, "-p", dir, "dump", "--compact", "TEST-MIB")
	if code != exitOK {
		t.Fatalf("dump --compact exited %d, stderr=%q", code, stderr)
	}
	// Compact output should not have indented lines
	for line := range strings.SplitSeq(stdout, "\n") {
		if strings.HasPrefix(line, "  ") {
			t.Fatalf("compact output should not be indented, got line: %q", line)
		}
	}
}

func TestDumpNoDescriptions(t *testing.T) {
	dir := writeTestMIB(t)

	code, stdout, stderr := runCLI(t, "-p", dir, "dump", "--no-descriptions", "TEST-MIB")
	if code != exitOK {
		t.Fatalf("dump --no-descriptions exited %d, stderr=%q", code, stderr)
	}
	if strings.Contains(stdout, "A test scalar") {
		t.Fatalf("--no-descriptions should omit description text, got %q", stdout)
	}
}

func TestDumpOIDFilter(t *testing.T) {
	dir := writeTestMIB(t)

	code, stdout, stderr := runCLI(t, "-p", dir, "dump", "-o", "1.3.6.1.4.1.99999.1", "TEST-MIB")
	if code != exitOK {
		t.Fatalf("dump -o exited %d, stderr=%q", code, stderr)
	}
	if !strings.Contains(stdout, "testScalar") {
		t.Fatalf("OID filter should include testScalar, got %q", stdout)
	}
}

// --- lint command ---

func TestLintBasicText(t *testing.T) {
	dir := writeTestMIB(t)

	code, stdout, _ := runCLI(t, "-p", dir, "lint", "TEST-MIB")
	// Exit 0 or 1 are both valid (depends on whether test MIB has issues)
	if code == exitError {
		t.Fatalf("lint returned operational error %d", code)
	}
	// Should mention module count
	if !strings.Contains(stdout, "module") {
		t.Fatalf("expected module reference in lint output, got %q", stdout)
	}
}

func TestLintFormatJSON(t *testing.T) {
	dir := writeTestMIB(t)

	code, stdout, _ := runCLI(t, "-p", dir, "lint", "--format", "json", "TEST-MIB")
	if code == exitError {
		t.Fatalf("lint --format json returned operational error %d", code)
	}
	if !strings.Contains(stdout, `"summary"`) {
		t.Fatalf("expected JSON summary field, got %q", stdout)
	}
}

func TestLintFormatSARIF(t *testing.T) {
	dir := writeTestMIB(t)

	code, stdout, _ := runCLI(t, "-p", dir, "lint", "--format", "sarif", "TEST-MIB")
	if code == exitError {
		t.Fatalf("lint --format sarif returned operational error %d", code)
	}
	if !strings.Contains(stdout, `"version"`) || !strings.Contains(stdout, `"runs"`) {
		t.Fatalf("expected SARIF structure, got %q", stdout)
	}
}

func TestLintFormatCompact(t *testing.T) {
	dir := writeTestMIB(t)

	code, _, _ := runCLI(t, "-p", dir, "lint", "--format", "compact", "TEST-MIB")
	if code == exitError {
		t.Fatalf("lint --format compact returned operational error %d", code)
	}
}

func TestLintSummary(t *testing.T) {
	dir := writeTestMIB(t)

	code, stdout, _ := runCLI(t, "-p", dir, "lint", "--summary", "TEST-MIB")
	if code == exitError {
		t.Fatalf("lint --summary returned operational error %d", code)
	}
	if !strings.Contains(stdout, "Checked") {
		t.Fatalf("expected summary header, got %q", stdout)
	}
}

func TestLintListCodes(t *testing.T) {
	code, stdout, stderr := runCLI(t, "lint", "--list-codes")
	if code != exitOK {
		t.Fatalf("lint --list-codes exited %d, stderr=%q", code, stderr)
	}
	// Should list at least some diagnostic codes
	if !strings.Contains(stdout, "import") && !strings.Contains(stdout, "type") {
		t.Fatalf("expected diagnostic code listing, got %q", stdout)
	}
}

func TestLintLevelSeverityNumberRange(t *testing.T) {
	testLintSeverityNumberRange(t, "--level")
}

func TestLintFailOnSeverityNumberRange(t *testing.T) {
	testLintSeverityNumberRange(t, "--fail-on")
}

func testLintSeverityNumberRange(t *testing.T, flag string) {
	t.Helper()

	for _, value := range []string{"-1", "7"} {
		t.Run("reject_"+value, func(t *testing.T) {
			code, _, _ := runCLI(t, "lint", flag+"="+value, "--list-codes")
			if code != exitError {
				t.Fatalf("lint %s=%s exited %d, want %d", flag, value, code, exitError)
			}
		})
	}

	for _, value := range []string{"0", "6"} {
		t.Run("accept_"+value, func(t *testing.T) {
			code, _, stderr := runCLI(t, "lint", flag+"="+value, "--list-codes")
			if code != exitOK {
				t.Fatalf("lint %s=%s exited %d, want %d; stderr=%q", flag, value, code, exitOK, stderr)
			}
		})
	}
}

func TestLintQuiet(t *testing.T) {
	dir := writeTestMIB(t)

	code, stdout, _ := runCLI(t, "-p", dir, "lint", "--quiet", "TEST-MIB")
	if code == exitError {
		t.Fatalf("lint --quiet returned operational error %d", code)
	}
	if stdout != "" {
		t.Fatalf("--quiet should produce no stdout, got %q", stdout)
	}
}

func TestLintGroupBy(t *testing.T) {
	dir := writeTestMIB(t)

	for _, groupBy := range []string{"module", "code", "severity"} {
		t.Run(groupBy, func(t *testing.T) {
			code, _, _ := runCLI(t, "-p", dir, "lint", "--group-by", groupBy, "TEST-MIB")
			if code == exitError {
				t.Fatalf("lint --group-by %s returned operational error", groupBy)
			}
		})
	}
}

// --- inspect command ---

func TestInspectObject(t *testing.T) {
	dir := writeTestMIB(t)

	code, stdout, stderr := runCLI(t, "-p", dir, "inspect", "-m", "TEST-MIB", "testScalar")
	if code != exitOK {
		t.Fatalf("inspect exited %d, stderr=%q", code, stderr)
	}
	for _, field := range []string{"testScalar", "TEST-MIB", "Integer32"} {
		if !strings.Contains(stdout, field) {
			t.Fatalf("expected %q in inspect output, got %q", field, stdout)
		}
	}
}

func TestInspectNotFound(t *testing.T) {
	dir := writeTestMIB(t)

	code, _, _ := runCLI(t, "-p", dir, "inspect", "-m", "TEST-MIB", "nonExistentSymbol")
	if code != exitIssue {
		t.Fatalf("expected exit %d for not found, got %d", exitIssue, code)
	}
}

func TestInspectRequiresArgument(t *testing.T) {
	dir := writeTestMIB(t)

	code, _, _ := runCLI(t, "-p", dir, "inspect", "-m", "TEST-MIB")
	if code != exitError {
		t.Fatalf("expected exit %d for missing argument, got %d", exitError, code)
	}
}

// --- trace command ---

func TestTraceObject(t *testing.T) {
	dir := writeTestMIB(t)

	code, stdout, stderr := runCLI(t, "-p", dir, "trace", "-m", "TEST-MIB", "testScalar")
	if code != exitOK {
		t.Fatalf("trace exited %d, stderr=%q", code, stderr)
	}
	if !strings.Contains(stdout, "testScalar") {
		t.Fatalf("expected testScalar in trace output, got %q", stdout)
	}
}

func TestTraceNotFound(t *testing.T) {
	dir := writeTestMIB(t)

	// trace always exits 0 - it reports whatever it finds, including "(none found)"
	code, stdout, stderr := runCLI(t, "-p", dir, "trace", "-m", "TEST-MIB", "nonExistentSymbol")
	if code != exitOK {
		t.Fatalf("trace exited %d, stderr=%q", code, stderr)
	}
	if !strings.Contains(stdout, "(none found)") {
		t.Fatalf("expected '(none found)' for missing symbol, got %q", stdout)
	}
}

func TestTraceRequiresArgument(t *testing.T) {
	dir := writeTestMIB(t)

	code, _, _ := runCLI(t, "-p", dir, "trace", "-m", "TEST-MIB")
	if code != exitError {
		t.Fatalf("expected exit %d for missing argument, got %d", exitError, code)
	}
}

// --- normalize command ---

func TestNormalizeBasic(t *testing.T) {
	dir := writeTestMIB(t)

	code, stdout, stderr := runCLI(t, "-p", dir, "normalize", "TEST-MIB")
	if code != exitOK {
		t.Fatalf("normalize exited %d, stderr=%q", code, stderr)
	}
	for _, keyword := range []string{"DEFINITIONS", "BEGIN", "END", "testScalar"} {
		if !strings.Contains(stdout, keyword) {
			t.Fatalf("expected %q in normalize output, got %q", keyword, stdout)
		}
	}
}

func TestNormalizeNoDescriptions(t *testing.T) {
	dir := writeTestMIB(t)

	code, stdout, stderr := runCLI(t, "-p", dir, "normalize", "--no-descriptions", "TEST-MIB")
	if code != exitOK {
		t.Fatalf("normalize --no-descriptions exited %d, stderr=%q", code, stderr)
	}
	if strings.Contains(stdout, "A test scalar") {
		t.Fatalf("--no-descriptions should omit description text, got %q", stdout)
	}
}

func TestNormalizeToDir(t *testing.T) {
	dir := writeTestMIB(t)
	outDir := t.TempDir()

	code, _, stderr := runCLI(t, "-p", dir, "normalize", "-o", outDir, "TEST-MIB")
	if code != exitOK {
		t.Fatalf("normalize -o exited %d, stderr=%q", code, stderr)
	}

	entries, err := os.ReadDir(outDir)
	if err != nil {
		t.Fatalf("reading output dir: %v", err)
	}
	if len(entries) == 0 {
		t.Fatal("expected output file in directory")
	}
}

func writeTestMIB(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	path := filepath.Join(dir, "TEST-MIB.mib")
	if err := os.WriteFile(path, []byte(testCLIMIB), 0o600); err != nil {
		t.Fatalf("write test MIB: %v", err)
	}
	return dir
}

func runCLI(t *testing.T, args ...string) (int, string, string) {
	t.Helper()

	var stdout, stderr bytes.Buffer
	code := run(args, &stdout, &stderr)
	return code, stdout.String(), stderr.String()
}

func runCLIWithSystemPaths(t *testing.T, discover func() []string, args ...string) (int, string, string) {
	t.Helper()

	var stdout, stderr bytes.Buffer
	c := cli{
		stdout:              &stdout,
		stderr:              &stderr,
		discoverSystemPaths: discover,
	}

	var cmdArgs []string
	var cmd string
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "-p", "--path":
			i++
			c.paths = append(c.paths, args[i])
		default:
			if cmd == "" {
				cmd = arg
			} else {
				cmdArgs = append(cmdArgs, arg)
			}
		}
	}

	var code int
	switch cmd {
	case "paths":
		code = c.cmdPaths(cmdArgs)
	default:
		t.Fatalf("runCLIWithSystemPaths: unsupported command %q", cmd)
	}
	return code, stdout.String(), stderr.String()
}
