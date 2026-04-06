package types

import "testing"

func TestSyntaxBaseTypeNames(t *testing.T) {
	names := SyntaxBaseTypeNames()
	if len(names) == 0 {
		t.Fatal("expected non-empty list")
	}

	set := make(map[string]bool)
	for _, n := range names {
		set[n] = true
	}

	// Key types should be present.
	for _, want := range []string{"INTEGER", "Integer32", "OCTET STRING", "OBJECT IDENTIFIER", "Counter32", "BITS"} {
		if !set[want] {
			t.Errorf("missing expected base type %q", want)
		}
	}

	// Internal-only types should be excluded.
	for _, bad := range []string{"unknown", "SEQUENCE"} {
		if set[bad] {
			t.Errorf("should not contain %q", bad)
		}
	}
}
