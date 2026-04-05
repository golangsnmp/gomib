package model

import (
	"testing"

	"github.com/golangsnmp/gomib/internal/testutil"
)

func TestParseOID(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
		wantNil bool
	}{
		{"simple", "1.3.6.1", "1.3.6.1", false, false},
		{"single arc", "1", "1", false, false},
		{"leading dot", ".1.3.6.1", "1.3.6.1", false, false},
		{"empty string", "", "", true, false},
		{"leading dot only", ".", "", true, false},
		{"zero arc", "0", "0", false, false},
		{"large arc", "4294967295", "4294967295", false, false},
		{"overflow", "4294967296", "", true, false},
		{"overflow mid", "1.3.4294967296.1", "", true, false},
		{"overflow large", "1.3.99999999999.1", "", true, false},
		{"invalid char", "1.3.x.1", "", true, false},
		{"empty arc", "1..3", "", true, false},
		{"trailing dot", "1.3.", "", true, false},
		{"leading and trailing dot", ".1.3.", "", true, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseOID(tt.input)
			if tt.wantErr {
				testutil.Error(t, err, "ParseOID() expected error, got")
				return
			}
			testutil.NoError(t, err, "ParseOID(%q)", tt.input)
			if tt.wantNil {
				testutil.Nil(t, got, "ParseOID() expected nil, got")
				return
			}
			testutil.Equal(t, tt.want, got.String(), "ParseOID()")
		})
	}
}

func TestParseOIDTrailingDot(t *testing.T) {
	_, err := ParseOID("1.3.")
	testutil.Error(t, err, "ParseOID(\"1.3.\") should return error for trailing dot")
}

func TestOidString(t *testing.T) {
	tests := []struct {
		name string
		oid  OID
		want string
	}{
		{"nil", nil, ""},
		{"empty", OID{}, ""},
		{"single", OID{1}, "1"},
		{"multi", OID{1, 3, 6, 1}, "1.3.6.1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.oid.String()
			testutil.Equal(t, tt.want, got, "OID.String()")
		})
	}
}

func TestOidParent(t *testing.T) {
	tests := []struct {
		name    string
		oid     OID
		wantNil bool
		want    string
	}{
		{"nil", nil, true, ""},
		{"single", OID{1}, true, ""},
		{"two arcs", OID{1, 3}, false, "1"},
		{"long", OID{1, 3, 6, 1, 2, 1}, false, "1.3.6.1.2"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.oid.Parent()
			if tt.wantNil {
				testutil.Nil(t, got, "Parent()")
				return
			}
			testutil.Equal(t, tt.want, got.String(), "Parent()")
		})
	}
}

func TestOidParentDoesNotMutate(t *testing.T) {
	original := OID{1, 3, 6}
	parent := original.Parent()
	// Mutating parent should not affect original
	if parent != nil {
		parent[0] = 99
	}
	testutil.Equal(t, 1, original[0], "Parent() returned a slice that shares backing array with original")
}

func TestOidChild(t *testing.T) {
	oid := OID{1, 3, 6}
	child := oid.Child(1)
	testutil.Equal(t, "1.3.6.1", child.String(), "Child(1)")

	// Original should be unchanged
	testutil.Equal(t, "1.3.6", oid.String(), "original mutated: got")

	// Nil oid
	var nilOid OID
	c := nilOid.Child(1)
	testutil.Equal(t, "1", c.String(), "nil.Child(1)")
}

func TestOidHasPrefix(t *testing.T) {
	tests := []struct {
		name   string
		oid    OID
		prefix OID
		want   bool
	}{
		{"exact match", OID{1, 3, 6}, OID{1, 3, 6}, true},
		{"prefix match", OID{1, 3, 6, 1}, OID{1, 3}, true},
		{"no match", OID{1, 3, 6}, OID{1, 4}, false},
		{"prefix longer", OID{1, 3}, OID{1, 3, 6}, false},
		{"empty prefix", OID{1, 3, 6}, OID{}, true},
		{"nil prefix", OID{1, 3}, nil, true},
		{"nil oid", nil, OID{1}, false},
		{"both empty", OID{}, OID{}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.oid.HasPrefix(tt.prefix)
			testutil.Equal(t, tt.want, got, "HasPrefix()")
		})
	}
}

func TestOidEqual(t *testing.T) {
	tests := []struct {
		name string
		a, b OID
		want bool
	}{
		{"equal", OID{1, 3, 6}, OID{1, 3, 6}, true},
		{"different", OID{1, 3, 6}, OID{1, 3, 7}, false},
		{"different length", OID{1, 3}, OID{1, 3, 6}, false},
		{"both nil", nil, nil, true},
		{"one nil", OID{1}, nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.a.Equal(tt.b)
			testutil.Equal(t, tt.want, got, "Equal()")
		})
	}
}

func TestOidCompare(t *testing.T) {
	tests := []struct {
		name string
		a, b OID
		want int
	}{
		{"equal", OID{1, 3, 6}, OID{1, 3, 6}, 0},
		{"less by value", OID{1, 3, 5}, OID{1, 3, 6}, -1},
		{"greater by value", OID{1, 3, 7}, OID{1, 3, 6}, 1},
		{"less by length", OID{1, 3}, OID{1, 3, 6}, -1},
		{"greater by length", OID{1, 3, 6}, OID{1, 3}, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.a.Compare(tt.b)
			testutil.Equal(t, tt.want, got, "Compare()")
		})
	}
}

func TestOidLastArc(t *testing.T) {
	tests := []struct {
		name string
		oid  OID
		want uint32
	}{
		{"normal", OID{1, 3, 6}, 6},
		{"single", OID{42}, 42},
		{"empty", OID{}, 0},
		{"nil", nil, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.oid.LastArc()
			testutil.Equal(t, tt.want, got, "LastArc()")
		})
	}
}
