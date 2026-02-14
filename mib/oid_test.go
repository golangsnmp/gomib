package mib

import "testing"

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
				if err == nil {
					t.Errorf("ParseOID(%q) expected error, got %v", tt.input, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseOID(%q) unexpected error: %v", tt.input, err)
			}
			if tt.wantNil {
				if got != nil {
					t.Errorf("ParseOID(%q) expected nil, got %v", tt.input, got)
				}
				return
			}
			if got.String() != tt.want {
				t.Errorf("ParseOID(%q) = %q, want %q", tt.input, got.String(), tt.want)
			}
		})
	}
}

func TestParseOIDTrailingDot(t *testing.T) {
	_, err := ParseOID("1.3.")
	if err == nil {
		t.Fatal("ParseOID(\"1.3.\") should return error for trailing dot")
	}
}

func TestOidString(t *testing.T) {
	tests := []struct {
		name string
		oid  Oid
		want string
	}{
		{"nil", nil, ""},
		{"empty", Oid{}, ""},
		{"single", Oid{1}, "1"},
		{"multi", Oid{1, 3, 6, 1}, "1.3.6.1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.oid.String()
			if got != tt.want {
				t.Errorf("Oid.String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestOidParent(t *testing.T) {
	tests := []struct {
		name    string
		oid     Oid
		wantNil bool
		want    string
	}{
		{"nil", nil, true, ""},
		{"single", Oid{1}, true, ""},
		{"two arcs", Oid{1, 3}, false, "1"},
		{"long", Oid{1, 3, 6, 1, 2, 1}, false, "1.3.6.1.2"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.oid.Parent()
			if tt.wantNil {
				if got != nil {
					t.Errorf("Parent() = %v, want nil", got)
				}
				return
			}
			if got.String() != tt.want {
				t.Errorf("Parent() = %q, want %q", got.String(), tt.want)
			}
		})
	}
}

func TestOidParentDoesNotMutate(t *testing.T) {
	original := Oid{1, 3, 6}
	parent := original.Parent()
	// Mutating parent should not affect original
	if parent != nil {
		parent[0] = 99
	}
	if original[0] != 1 {
		t.Error("Parent() returned a slice that shares backing array with original")
	}
}

func TestOidChild(t *testing.T) {
	oid := Oid{1, 3, 6}
	child := oid.Child(1)
	if child.String() != "1.3.6.1" {
		t.Errorf("Child(1) = %q, want 1.3.6.1", child.String())
	}

	// Original should be unchanged
	if oid.String() != "1.3.6" {
		t.Errorf("original mutated: got %q", oid.String())
	}

	// Nil oid
	var nilOid Oid
	c := nilOid.Child(1)
	if c.String() != "1" {
		t.Errorf("nil.Child(1) = %q, want 1", c.String())
	}
}

func TestOidHasPrefix(t *testing.T) {
	tests := []struct {
		name   string
		oid    Oid
		prefix Oid
		want   bool
	}{
		{"exact match", Oid{1, 3, 6}, Oid{1, 3, 6}, true},
		{"prefix match", Oid{1, 3, 6, 1}, Oid{1, 3}, true},
		{"no match", Oid{1, 3, 6}, Oid{1, 4}, false},
		{"prefix longer", Oid{1, 3}, Oid{1, 3, 6}, false},
		{"empty prefix", Oid{1, 3, 6}, Oid{}, true},
		{"nil prefix", Oid{1, 3}, nil, true},
		{"nil oid", nil, Oid{1}, false},
		{"both empty", Oid{}, Oid{}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.oid.HasPrefix(tt.prefix)
			if got != tt.want {
				t.Errorf("HasPrefix() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestOidEqual(t *testing.T) {
	tests := []struct {
		name string
		a, b Oid
		want bool
	}{
		{"equal", Oid{1, 3, 6}, Oid{1, 3, 6}, true},
		{"different", Oid{1, 3, 6}, Oid{1, 3, 7}, false},
		{"different length", Oid{1, 3}, Oid{1, 3, 6}, false},
		{"both nil", nil, nil, true},
		{"one nil", Oid{1}, nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.a.Equal(tt.b)
			if got != tt.want {
				t.Errorf("Equal() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestOidCompare(t *testing.T) {
	tests := []struct {
		name string
		a, b Oid
		want int
	}{
		{"equal", Oid{1, 3, 6}, Oid{1, 3, 6}, 0},
		{"less by value", Oid{1, 3, 5}, Oid{1, 3, 6}, -1},
		{"greater by value", Oid{1, 3, 7}, Oid{1, 3, 6}, 1},
		{"less by length", Oid{1, 3}, Oid{1, 3, 6}, -1},
		{"greater by length", Oid{1, 3, 6}, Oid{1, 3}, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.a.Compare(tt.b)
			if got != tt.want {
				t.Errorf("Compare() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestOidLastArc(t *testing.T) {
	tests := []struct {
		name string
		oid  Oid
		want uint32
	}{
		{"normal", Oid{1, 3, 6}, 6},
		{"single", Oid{42}, 42},
		{"empty", Oid{}, 0},
		{"nil", nil, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.oid.LastArc()
			if got != tt.want {
				t.Errorf("LastArc() = %d, want %d", got, tt.want)
			}
		})
	}
}
