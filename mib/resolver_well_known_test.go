package mib

import (
	"testing"

	"github.com/golangsnmp/gomib/internal/testutil"
)

// Test-only helpers for wellKnownTypes classification.
func isASN1Primitive(name string) bool   { return wellKnownTypes[name] == typeClassASN1Primitive }
func isSmiGlobalType(name string) bool   { return wellKnownTypes[name] == typeClassSmiGlobal }
func isSmiV1GlobalType(name string) bool { return wellKnownTypes[name] == typeClassSmiV1Global }
func isSNMPv2TCType(name string) bool    { return wellKnownTypes[name] == typeClassSNMPv2TC }

func TestIsASN1Primitive(t *testing.T) {
	positives := []string{"INTEGER", "OCTET STRING", "OBJECT IDENTIFIER", "BITS"}
	for _, name := range positives {
		testutil.True(t, isASN1Primitive(name), "isASN1Primitive() = false, want true")
	}

	negatives := []string{
		"Integer32", "Counter32", "DisplayString", "integer",
		"OCTETSTRING", "OBJECT-IDENTIFIER", "", "Counter",
	}
	for _, name := range negatives {
		testutil.False(t, isASN1Primitive(name), "isASN1Primitive(%q) = true, want false", name)
	}
}

func TestIsSmiGlobalType(t *testing.T) {
	positives := []string{
		"Integer32", "Counter32", "Counter64", "Gauge32",
		"Unsigned32", "TimeTicks", "IpAddress", "Opaque",
	}
	for _, name := range positives {
		testutil.True(t, isSmiGlobalType(name), "isSmiGlobalType() = false, want true")
	}

	negatives := []string{
		"INTEGER", "Counter", "Gauge", "DisplayString",
		"integer32", "NetworkAddress", "",
	}
	for _, name := range negatives {
		testutil.False(t, isSmiGlobalType(name), "isSmiGlobalType(%q) = true, want false", name)
	}
}

func TestIsSmiV1GlobalType(t *testing.T) {
	positives := []string{"Counter", "Gauge", "NetworkAddress"}
	for _, name := range positives {
		testutil.True(t, isSmiV1GlobalType(name), "isSmiV1GlobalType() = false, want true")
	}

	negatives := []string{
		"Counter32", "Gauge32", "IpAddress", "INTEGER",
		"counter", "TimeTicks", "",
	}
	for _, name := range negatives {
		testutil.False(t, isSmiV1GlobalType(name), "isSmiV1GlobalType(%q) = true, want false", name)
	}
}

func TestIsSNMPv2TCType(t *testing.T) {
	positives := []string{
		"DisplayString", "TruthValue", "PhysAddress", "MacAddress",
		"RowStatus", "TimeStamp", "TimeInterval", "DateAndTime",
		"StorageType", "TestAndIncr", "AutonomousType",
		"VariablePointer", "RowPointer", "InstancePointer",
		"TDomain", "TAddress",
	}
	for _, name := range positives {
		testutil.True(t, isSNMPv2TCType(name), "isSNMPv2TCType() = false, want true")
	}

	negatives := []string{
		"INTEGER", "Counter32", "IpAddress", "displaystring",
		"Counter", "Gauge32", "",
	}
	for _, name := range negatives {
		testutil.False(t, isSNMPv2TCType(name), "isSNMPv2TCType(%q) = true, want false", name)
	}
}
