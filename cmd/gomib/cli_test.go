package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const testCLIMIB = `TEST-MIB DEFINITIONS ::= BEGIN
IMPORTS
    MODULE-IDENTITY, OBJECT-TYPE, NOTIFICATION-TYPE, enterprises, Integer32
        FROM SNMPv2-SMI;

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

func TestGetRejectsExtraArgsAfterSeparator(t *testing.T) {
	dir := writeTestMIB(t)

	code, _, stderr := runCLI(t, "-p", dir, "get", "TEST-MIB", "--", "testMib", "testScalar")
	if code != exitError {
		t.Fatalf("expected exit %d, got %d", exitError, code)
	}
	if !strings.Contains(stderr, "too many query arguments after --") {
		t.Fatalf("expected extra-args error, got %q", stderr)
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

func TestFindSupportsPositionalModules(t *testing.T) {
	dir := writeTestMIB(t)

	code, stdout, stderr := runCLI(t, "-p", dir, "find", "TEST-MIB", "testScalar")
	if code != exitOK {
		t.Fatalf("find exited %d, stderr=%q", code, stderr)
	}
	if !strings.Contains(stdout, "TEST-MIB::testScalar") {
		t.Fatalf("expected object in output, got %q", stdout)
	}
}

func TestPathsReturnsErrorWhenNoPathsFound(t *testing.T) {
	orig := discoverSystemPaths
	discoverSystemPaths = func() []string { return nil }
	t.Cleanup(func() { discoverSystemPaths = orig })

	code, _, stderr := runCLI(t, "paths")
	if code != exitError {
		t.Fatalf("expected exit %d, got %d", exitError, code)
	}
	if !strings.Contains(stderr, "no search paths found") {
		t.Fatalf("expected no-paths error, got %q", stderr)
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

	oldArgs := os.Args
	oldStdout := os.Stdout
	oldStderr := os.Stderr

	stdoutR, stdoutW, err := os.Pipe()
	if err != nil {
		t.Fatalf("stdout pipe: %v", err)
	}
	stderrR, stderrW, err := os.Pipe()
	if err != nil {
		t.Fatalf("stderr pipe: %v", err)
	}

	os.Args = append([]string{"gomib"}, args...)
	os.Stdout = stdoutW
	os.Stderr = stderrW

	code := run()

	_ = stdoutW.Close()
	_ = stderrW.Close()
	os.Args = oldArgs
	os.Stdout = oldStdout
	os.Stderr = oldStderr

	var stdoutBuf bytes.Buffer
	var stderrBuf bytes.Buffer
	_, _ = stdoutBuf.ReadFrom(stdoutR)
	_, _ = stderrBuf.ReadFrom(stderrR)
	_ = stdoutR.Close()
	_ = stderrR.Close()

	return code, stdoutBuf.String(), stderrBuf.String()
}
