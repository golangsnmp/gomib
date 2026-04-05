package model

import (
	"testing"
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

// FuzzFormatOID moved to internal/resolver/fuzz_test.go
