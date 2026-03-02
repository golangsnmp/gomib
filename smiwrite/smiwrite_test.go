package smiwrite

import (
	"bytes"
	"context"
	"os"
	"strings"
	"testing"

	"github.com/golangsnmp/gomib"
	"github.com/golangsnmp/gomib/mib"
)

func loadTestMib(t *testing.T, modules ...string) *mib.Mib {
	t.Helper()
	src, err := gomib.Dir("../testdata/corpus/primary")
	if err != nil {
		t.Fatal(err)
	}
	m, err := gomib.Load(context.Background(),
		gomib.WithSource(src),
		gomib.WithModules(modules...),
	)
	if err != nil {
		t.Fatal(err)
	}
	return m
}

func TestFormatStatus(t *testing.T) {
	tests := []struct {
		input mib.Status
		want  string
	}{
		{mib.StatusCurrent, "current"},
		{mib.StatusDeprecated, "deprecated"},
		{mib.StatusObsolete, "obsolete"},
		{mib.StatusMandatory, "current"},
		{mib.StatusOptional, "deprecated"},
	}
	for _, tt := range tests {
		got := formatStatus(tt.input)
		if got != tt.want {
			t.Errorf("formatStatus(%v) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestFormatAccess(t *testing.T) {
	tests := []struct {
		input mib.Access
		want  string
	}{
		{mib.AccessReadOnly, "read-only"},
		{mib.AccessReadWrite, "read-write"},
		{mib.AccessReadCreate, "read-create"},
		{mib.AccessNotAccessible, "not-accessible"},
		{mib.AccessWriteOnly, "read-write"},
	}
	for _, tt := range tests {
		got := formatAccess(tt.input)
		if got != tt.want {
			t.Errorf("formatAccess(%v) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestBaseTypeSyntax(t *testing.T) {
	tests := []struct {
		input mib.BaseType
		want  string
	}{
		{mib.BaseInteger32, "Integer32"},
		{mib.BaseOctetString, "OCTET STRING"},
		{mib.BaseObjectIdentifier, "OBJECT IDENTIFIER"},
		{mib.BaseCounter32, "Counter32"},
		{mib.BaseGauge32, "Gauge32"},
		{mib.BaseBits, "BITS"},
	}
	for _, tt := range tests {
		got := baseTypeSyntax(tt.input)
		if got != tt.want {
			t.Errorf("baseTypeSyntax(%v) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestFormatEnums(t *testing.T) {
	values := []mib.NamedValue{
		{Label: "up", Value: 1},
		{Label: "down", Value: 2},
	}
	got := formatEnums(values)
	if !strings.Contains(got, "up(1)") || !strings.Contains(got, "down(2)") {
		t.Errorf("formatEnums returned %q, expected up(1) and down(2)", got)
	}
}

func TestFormatRanges(t *testing.T) {
	tests := []struct {
		input []mib.Range
		want  string
	}{
		{nil, ""},
		{[]mib.Range{{Min: 0, Max: 255}}, "(0..255)"},
		{[]mib.Range{{Min: 1, Max: 1}}, "(1)"},
		{[]mib.Range{{Min: 0, Max: 10}, {Min: 20, Max: 30}}, "(0..10 | 20..30)"},
	}
	for _, tt := range tests {
		got := formatRanges(tt.input)
		if got != tt.want {
			t.Errorf("formatRanges(%v) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestFormatSizes(t *testing.T) {
	tests := []struct {
		input []mib.Range
		want  string
	}{
		{nil, ""},
		{[]mib.Range{{Min: 0, Max: 255}}, "(SIZE (0..255))"},
		{[]mib.Range{{Min: 6, Max: 6}}, "(SIZE (6))"},
	}
	for _, tt := range tests {
		got := formatSizes(tt.input)
		if got != tt.want {
			t.Errorf("formatSizes(%v) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestFormatDescription(t *testing.T) {
	got := formatDescription("A simple description.")
	if got != `"A simple description."` {
		t.Errorf("formatDescription returned %q", got)
	}

	got = formatDescription(`Has "quotes" inside.`)
	if got != `"Has \"quotes\" inside."` {
		t.Errorf("formatDescription with quotes returned %q", got)
	}
}

func TestSequenceTypeName(t *testing.T) {
	tests := []struct {
		input, want string
	}{
		{"ifEntry", "IfEntry"},
		{"IfEntry", "IfEntry"},
		{"", ""},
	}
	for _, tt := range tests {
		got := sequenceTypeName(tt.input)
		if got != tt.want {
			t.Errorf("sequenceTypeName(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestIsBuiltinKeyword(t *testing.T) {
	for _, kw := range []string{"INTEGER", "OCTET STRING", "OBJECT IDENTIFIER", "BITS"} {
		if !isBuiltinKeyword(kw) {
			t.Errorf("isBuiltinKeyword(%q) = false, want true", kw)
		}
	}
	for _, kw := range []string{"Integer32", "Counter32", "DisplayString"} {
		if isBuiltinKeyword(kw) {
			t.Errorf("isBuiltinKeyword(%q) = true, want false", kw)
		}
	}
}

func TestWriteModuleNotFound(t *testing.T) {
	m := loadTestMib(t, "SNMPv2-MIB")
	var buf bytes.Buffer
	err := Write(&buf, m, "NONEXISTENT")
	if err == nil {
		t.Fatal("expected error for nonexistent module")
	}
	if !strings.Contains(err.Error(), "module not found") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestWriteSNMPv2MIB(t *testing.T) {
	m := loadTestMib(t, "SNMPv2-MIB")
	var buf bytes.Buffer
	if err := Write(&buf, m, "SNMPv2-MIB"); err != nil {
		t.Fatal(err)
	}
	output := buf.String()

	// Check structure
	if !strings.HasPrefix(output, "SNMPv2-MIB DEFINITIONS ::= BEGIN\n") {
		t.Error("missing module header")
	}
	if !strings.HasSuffix(output, "\nEND\n") {
		t.Error("missing END")
	}

	// Check IMPORTS present and correct
	if !strings.Contains(output, "IMPORTS") {
		t.Error("missing IMPORTS")
	}
	if !strings.Contains(output, "FROM SNMPv2-SMI") {
		t.Error("missing FROM SNMPv2-SMI")
	}
	// Should NOT import built-in keywords
	for line := range strings.SplitSeq(output, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "INTEGER,") || strings.HasSuffix(trimmed, ", INTEGER") {
			t.Error("should not import INTEGER keyword")
		}
	}

	// Check MODULE-IDENTITY
	if !strings.Contains(output, "snmpMIB MODULE-IDENTITY") {
		t.Error("missing MODULE-IDENTITY")
	}
	if !strings.Contains(output, "LAST-UPDATED") {
		t.Error("missing LAST-UPDATED")
	}

	// Check OID assignments
	if !strings.Contains(output, "system OBJECT IDENTIFIER ::=") {
		t.Error("missing system OID assignment")
	}

	// Check OBJECT-TYPE
	if !strings.Contains(output, "sysDescr OBJECT-TYPE") {
		t.Error("missing sysDescr")
	}
	if !strings.Contains(output, "MAX-ACCESS read-only") {
		t.Error("missing MAX-ACCESS")
	}

	// Check NOTIFICATION-TYPE
	if !strings.Contains(output, "coldStart NOTIFICATION-TYPE") {
		t.Error("missing coldStart notification")
	}
}

func TestWriteIFMIB(t *testing.T) {
	m := loadTestMib(t, "IF-MIB")
	var buf bytes.Buffer
	if err := Write(&buf, m, "IF-MIB"); err != nil {
		t.Fatal(err)
	}
	output := buf.String()

	// Table/Row SYNTAX
	if !strings.Contains(output, "SYNTAX SEQUENCE OF IfEntry") {
		t.Error("missing SEQUENCE OF IfEntry for ifTable")
	}
	if !strings.Contains(output, "ifEntry OBJECT-TYPE\n    SYNTAX IfEntry") {
		t.Error("missing IfEntry for ifEntry row")
	}

	// Enums
	if !strings.Contains(output, "up(1)") {
		t.Error("missing enum value up(1) for ifAdminStatus")
	}

	// INDEX clause
	if !strings.Contains(output, "INDEX { ifIndex }") {
		t.Error("missing INDEX clause for ifEntry")
	}

	// AUGMENTS clause
	if !strings.Contains(output, "AUGMENTS { ifEntry }") {
		t.Error("missing AUGMENTS clause for ifXEntry")
	}

	// DEFVAL
	if !strings.Contains(output, "DEFVAL") {
		t.Error("missing DEFVAL in output")
	}

	// Notifications
	if !strings.Contains(output, "linkDown NOTIFICATION-TYPE") {
		t.Error("missing linkDown notification")
	}
	if !strings.Contains(output, "OBJECTS { ifIndex, ifAdminStatus, ifOperStatus }") {
		t.Error("missing OBJECTS clause for linkDown")
	}

	// Conformance
	if !strings.Contains(output, "OBJECT-GROUP") {
		t.Error("missing OBJECT-GROUP")
	}
	if !strings.Contains(output, "MODULE-COMPLIANCE") {
		t.Error("missing MODULE-COMPLIANCE")
	}

	// Textual conventions
	if !strings.Contains(output, "InterfaceIndex ::= TEXTUAL-CONVENTION") {
		t.Error("missing InterfaceIndex TC")
	}
}

func TestWithConformanceFalse(t *testing.T) {
	m := loadTestMib(t, "IF-MIB")
	var buf bytes.Buffer
	if err := Write(&buf, m, "IF-MIB", WithConformance(false)); err != nil {
		t.Fatal(err)
	}
	output := buf.String()

	if strings.Contains(output, "OBJECT-GROUP") {
		t.Error("OBJECT-GROUP should be omitted with WithConformance(false)")
	}
	if strings.Contains(output, "MODULE-COMPLIANCE") {
		t.Error("MODULE-COMPLIANCE should be omitted with WithConformance(false)")
	}
	// Objects should still be present
	if !strings.Contains(output, "ifDescr OBJECT-TYPE") {
		t.Error("objects should still be present")
	}
}

func TestWithDescriptionsFalse(t *testing.T) {
	m := loadTestMib(t, "SNMPv2-MIB")
	var buf bytes.Buffer
	if err := Write(&buf, m, "SNMPv2-MIB", WithDescriptions(false)); err != nil {
		t.Fatal(err)
	}
	output := buf.String()

	if strings.Contains(output, "DESCRIPTION") {
		t.Error("DESCRIPTION should be omitted with WithDescriptions(false)")
	}
}

func TestWithSequencesDefault(t *testing.T) {
	m := loadTestMib(t, "IF-MIB")
	var buf bytes.Buffer
	if err := Write(&buf, m, "IF-MIB"); err != nil {
		t.Fatal(err)
	}
	output := buf.String()

	if !strings.Contains(output, "IfEntry ::= SEQUENCE {") {
		t.Error("missing IfEntry SEQUENCE in default output")
	}
}

func TestWithSequencesFalse(t *testing.T) {
	m := loadTestMib(t, "IF-MIB")
	var buf bytes.Buffer
	if err := Write(&buf, m, "IF-MIB", WithSequences(false)); err != nil {
		t.Fatal(err)
	}
	output := buf.String()

	if strings.Contains(output, "::= SEQUENCE {") {
		t.Error("SEQUENCE should be omitted with WithSequences(false)")
	}
}

func TestWriteAll(t *testing.T) {
	m := loadTestMib(t, "IF-MIB", "SNMPv2-MIB")
	var buf bytes.Buffer
	if err := WriteAll(&buf, m, []string{"SNMPv2-MIB", "IF-MIB"}); err != nil {
		t.Fatal(err)
	}
	output := buf.String()

	if !strings.Contains(output, "SNMPv2-MIB DEFINITIONS ::= BEGIN") {
		t.Error("missing SNMPv2-MIB")
	}
	if !strings.Contains(output, "IF-MIB DEFINITIONS ::= BEGIN") {
		t.Error("missing IF-MIB")
	}
}

func TestRoundTrip(t *testing.T) {
	m1 := loadTestMib(t, "SNMPv2-MIB")
	var buf bytes.Buffer
	if err := Write(&buf, m1, "SNMPv2-MIB"); err != nil {
		t.Fatal(err)
	}

	// Write normalized output to a temp dir and reload
	dir := t.TempDir()
	normalized := buf.String()
	if err := os.WriteFile(dir+"/SNMPv2-MIB", []byte(normalized), 0o644); err != nil {
		t.Fatal(err)
	}

	// Reload from the normalized file
	src1, err := gomib.Dir("../testdata/corpus/primary")
	if err != nil {
		t.Fatal(err)
	}
	src2, err := gomib.Dir(dir)
	if err != nil {
		t.Fatal(err)
	}
	m2, err := gomib.Load(context.Background(),
		gomib.WithSource(src2, src1),
		gomib.WithModules("SNMPv2-MIB"),
	)
	if err != nil {
		t.Fatal(err)
	}

	// Compare key properties
	mod1 := m1.Module("SNMPv2-MIB")
	mod2 := m2.Module("SNMPv2-MIB")
	if mod1 == nil || mod2 == nil {
		t.Fatal("module not found after round-trip")
	}

	// Compare object counts
	if len(mod1.Objects()) != len(mod2.Objects()) {
		t.Errorf("object count mismatch: %d vs %d", len(mod1.Objects()), len(mod2.Objects()))
	}

	// Compare object names and OIDs
	for _, obj1 := range mod1.Objects() {
		obj2 := mod2.Object(obj1.Name())
		if obj2 == nil {
			t.Errorf("object %q missing after round-trip", obj1.Name())
			continue
		}
		if !obj1.OID().Equal(obj2.OID()) {
			t.Errorf("OID mismatch for %s: %v vs %v", obj1.Name(), obj1.OID(), obj2.OID())
		}
	}

	// Compare notifications
	if len(mod1.Notifications()) != len(mod2.Notifications()) {
		t.Errorf("notification count mismatch: %d vs %d",
			len(mod1.Notifications()), len(mod2.Notifications()))
	}
}
