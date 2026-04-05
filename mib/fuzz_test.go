package mib

import (
	"testing"

	"github.com/golangsnmp/gomib/internal/module"
	"github.com/golangsnmp/gomib/internal/parser"
	"github.com/golangsnmp/gomib/internal/types"
)

func FuzzParseOID(f *testing.F) {
	f.Add("1.3.6.1.2.1")
	f.Add(".1.3.6.1.2.1")
	f.Add("0.0")
	f.Add("")
	f.Add(".")
	f.Add("..")
	f.Add("1")
	f.Add("4294967295")
	f.Add("4294967296")
	f.Add("1.3.6.1.2.1.1.1.0")
	f.Add("99999999999999999999")
	f.Add("1..2")
	f.Add("1.2.")
	f.Add("0.40.1")
	f.Add("1.40.1")

	f.Fuzz(func(t *testing.T, s string) {
		oid, err := ParseOID(s)
		if err != nil {
			return
		}

		// Round-trip: String() -> ParseOID should produce the same OID.
		s2 := oid.String()
		oid2, err := ParseOID(s2)
		if err != nil {
			t.Fatalf("ParseOID(%q) succeeded, but ParseOID(String()) = %q failed: %v", s, s2, err)
		}
		if !oid.Equal(oid2) {
			t.Fatalf("round-trip mismatch: ParseOID(%q) = %v, String() = %q, re-parsed = %v", s, oid, s2, oid2)
		}

		// Parent invariant: Parent() should be a strict prefix.
		if p := oid.Parent(); p != nil {
			if !oid.HasPrefix(p) {
				t.Fatalf("OID %v does not have its Parent %v as prefix", oid, p)
			}
			if len(p) != len(oid)-1 {
				t.Fatalf("Parent length %d, want %d", len(p), len(oid)-1)
			}
		} else if len(oid) > 1 {
			t.Fatalf("OID with %d arcs returned nil Parent", len(oid))
		}

		// HasPrefix reflexivity.
		if !oid.HasPrefix(oid) {
			t.Fatal("OID is not a prefix of itself")
		}

		// Compare/Equal consistency.
		if oid.Compare(oid) != 0 {
			t.Fatal("OID.Compare(self) != 0")
		}

		// Child extends by one arc.
		child := oid.Child(42)
		if len(child) != len(oid)+1 {
			t.Fatalf("Child length %d, want %d", len(child), len(oid)+1)
		}
		if child.LastArc() != 42 {
			t.Fatalf("Child.LastArc() = %d, want 42", child.LastArc())
		}
		if !child.HasPrefix(oid) {
			t.Fatal("Child does not have parent as prefix")
		}
	})
}

// FuzzFormatOID fuzzes the FormatOID method against a resolved MIB tree.
// This catches panics in the suffix slicing and string formatting logic
// that only surface with specific OID prefix/suffix combinations.
func FuzzFormatOID(f *testing.F) {
	f.Add("1.3.6.1.2.1")
	f.Add("1.3.6.1.2.1.1.1.0")
	f.Add("1.3.6.1.4.1.99999.1.2.3")
	f.Add("0.0")
	f.Add("1")
	f.Add("")
	f.Add("1.3.6.1.2.1.999.999.999")

	// Build a small resolved MIB to test FormatOID against.
	src := []byte(`TEST-MIB DEFINITIONS ::= BEGIN
		IMPORTS MODULE-IDENTITY, OBJECT-TYPE, Integer32, enterprises FROM SNMPv2-SMI;
		testMIB MODULE-IDENTITY
			LAST-UPDATED "200601010000Z"
			ORGANIZATION ""
			CONTACT-INFO ""
			DESCRIPTION ""
			::= { enterprises 99999 }
		testObj OBJECT-TYPE
			SYNTAX Integer32
			MAX-ACCESS read-only
			STATUS current
			DESCRIPTION ""
			::= { testMIB 1 }
		END`)
	cfg := types.DefaultConfig()
	p := parser.New(src, nil, cfg)
	mods := p.ParseModule()
	mod := mods[0]
	module.ValidateModule(mod, src, nil, cfg)
	resolverCfg := DefaultConfig()
	mibTree := Resolve([]*module.Module{mod}, nil, nil, &resolverCfg)

	f.Fuzz(func(t *testing.T, s string) {
		oid, err := ParseOID(s)
		if err != nil {
			return
		}

		// FormatOID must not panic regardless of input.
		result := mibTree.FormatOID(oid)
		if result == "" && len(oid) > 0 {
			t.Fatal("FormatOID returned empty string for non-empty OID")
		}

		// LongestPrefixByOID must not panic.
		_ = mibTree.LongestPrefixByOID(oid)

		// NodeByOID must not panic.
		_ = mibTree.NodeByOID(oid)
	})
}
