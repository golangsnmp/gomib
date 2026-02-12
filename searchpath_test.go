package gomib

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestParseNetSNMPLine(t *testing.T) {
	tests := []struct {
		line   string
		wantOp pathOp
		want   []string
		wantOk bool
	}{
		// Replace
		{"mibdirs /usr/share/snmp/mibs", pathReplace, []string{"/usr/share/snmp/mibs"}, true},
		{"mibdirs /a:/b:/c", pathReplace, []string{"/a", "/b", "/c"}, true},
		// Append (+ prefix on value)
		{"mibdirs +/extra/mibs", pathAppend, []string{"/extra/mibs"}, true},
		{"mibdirs +/a:/b", pathAppend, []string{"/a", "/b"}, true},
		// Prepend (- prefix on value)
		{"mibdirs -/first/mibs", pathPrepend, []string{"/first/mibs"}, true},
		// Append (+mibdirs directive)
		{"+mibdirs /extra", pathAppend, []string{"/extra"}, true},
		// Prepend (-mibdirs directive)
		{"-mibdirs /first", pathPrepend, []string{"/first"}, true},
		// Whitespace variations
		{"  mibdirs  /path  ", pathReplace, []string{"/path"}, true},
		{"mibdirs\t/path", pathReplace, []string{"/path"}, true},
		// Not a mibdirs line
		{"mibs +ALL", 0, nil, false},
		{"persistentDir /var/net-snmp", 0, nil, false},
		// Comments and blanks
		{"# mibdirs /foo", 0, nil, false},
		{"", 0, nil, false},
		{"  ", 0, nil, false},
		// No value
		{"mibdirs", 0, nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.line, func(t *testing.T) {
			op, dirs, ok := parseNetSNMPLine(tt.line)
			if ok != tt.wantOk {
				t.Fatalf("ok = %v, want %v", ok, tt.wantOk)
			}
			if !ok {
				return
			}
			if op != tt.wantOp {
				t.Errorf("op = %v, want %v", op, tt.wantOp)
			}
			if !slices.Equal(dirs, tt.want) {
				t.Errorf("dirs = %v, want %v", dirs, tt.want)
			}
		})
	}
}

func TestParseLibSMILine(t *testing.T) {
	tests := []struct {
		line   string
		wantOp pathOp
		want   []string
		wantOk bool
	}{
		// Replace
		{"path /usr/share/mibs/ietf", pathReplace, []string{"/usr/share/mibs/ietf"}, true},
		{"path /a:/b", pathReplace, []string{"/a", "/b"}, true},
		// Append (leading colon)
		{"path :/extra/mibs", pathAppend, []string{"/extra/mibs"}, true},
		{"path :/a:/b", pathAppend, []string{"/a", "/b"}, true},
		// Prepend (trailing colon)
		{"path /first/mibs:", pathPrepend, []string{"/first/mibs"}, true},
		{"path /a:/b:", pathPrepend, []string{"/a", "/b"}, true},
		// Tagged lines - skipped
		{"smilint: path /foo", 0, nil, false},
		{"smiquery: path /foo", 0, nil, false},
		{"tcpdump: load IF-MIB", 0, nil, false},
		// Not a path line
		{"load IF-MIB", 0, nil, false},
		{"level 9", 0, nil, false},
		// Comments and blanks
		{"# path /foo", 0, nil, false},
		{"", 0, nil, false},
		{"  ", 0, nil, false},
		// No value
		{"path", 0, nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.line, func(t *testing.T) {
			op, dirs, ok := parseLibSMILine(tt.line)
			if ok != tt.wantOk {
				t.Fatalf("ok = %v, want %v", ok, tt.wantOk)
			}
			if !ok {
				return
			}
			if op != tt.wantOp {
				t.Errorf("op = %v, want %v", op, tt.wantOp)
			}
			if !slices.Equal(dirs, tt.want) {
				t.Errorf("dirs = %v, want %v", dirs, tt.want)
			}
		})
	}
}

func TestApplyOp(t *testing.T) {
	current := []string{"/default"}

	tests := []struct {
		name string
		op   pathOp
		dirs []string
		want []string
	}{
		{"replace", pathReplace, []string{"/new"}, []string{"/new"}},
		{"append", pathAppend, []string{"/extra"}, []string{"/default", "/extra"}},
		{"prepend", pathPrepend, []string{"/first"}, []string{"/first", "/default"}},
		{"append multiple", pathAppend, []string{"/a", "/b"}, []string{"/default", "/a", "/b"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := applyOp(tt.op, tt.dirs, current)
			if !slices.Equal(got, tt.want) {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestApplyNetSNMPEnv(t *testing.T) {
	current := []string{"/default/mibs"}

	tests := []struct {
		name  string
		value string
		want  []string
	}{
		{"replace", "/new/mibs", []string{"/new/mibs"}},
		{"replace multiple", "/a:/b", []string{"/a", "/b"}},
		{"append", "+/extra/mibs", []string{"/default/mibs", "/extra/mibs"}},
		{"append multiple", "+/a:/b", []string{"/default/mibs", "/a", "/b"}},
		{"prepend", "-/first/mibs", []string{"/first/mibs", "/default/mibs"}},
		{"prepend multiple", "-/a:/b", []string{"/a", "/b", "/default/mibs"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := applyNetSNMPEnv(tt.value, current)
			if !slices.Equal(got, tt.want) {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestApplyLibSMIEnv(t *testing.T) {
	current := []string{"/default/mibs"}

	tests := []struct {
		name  string
		value string
		want  []string
	}{
		{"replace", "/new/mibs", []string{"/new/mibs"}},
		{"replace multiple", "/a:/b", []string{"/a", "/b"}},
		{"append", ":/extra/mibs", []string{"/default/mibs", "/extra/mibs"}},
		{"append multiple", ":/a:/b", []string{"/default/mibs", "/a", "/b"}},
		{"prepend", "/first/mibs:", []string{"/first/mibs", "/default/mibs"}},
		{"prepend multiple", "/a:/b:", []string{"/a", "/b", "/default/mibs"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := applyLibSMIEnv(tt.value, current)
			if !slices.Equal(got, tt.want) {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDedup(t *testing.T) {
	tests := []struct {
		name  string
		input []string
		want  []string
	}{
		{"no dups", []string{"/a", "/b", "/c"}, []string{"/a", "/b", "/c"}},
		{"with dups", []string{"/a", "/b", "/a", "/c", "/b"}, []string{"/a", "/b", "/c"}},
		{"all same", []string{"/a", "/a", "/a"}, []string{"/a"}},
		{"empty", nil, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := dedup(tt.input)
			if !slices.Equal(got, tt.want) {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFilterExistingDirs(t *testing.T) {
	dir := t.TempDir()
	existing := filepath.Join(dir, "exists")
	if err := os.Mkdir(existing, 0o755); err != nil {
		t.Fatal(err)
	}

	// Create a file (not a directory) to verify it's excluded
	filePath := filepath.Join(dir, "afile")
	if err := os.WriteFile(filePath, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	got := filterExistingDirs([]string{existing, filepath.Join(dir, "missing"), filePath, "/nonexistent"})
	want := []string{existing}
	if !slices.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestApplyConfigFileNetSNMP(t *testing.T) {
	dir := t.TempDir()
	confPath := filepath.Join(dir, "snmp.conf")
	if err := os.WriteFile(confPath, []byte("# Comment\nmibdirs /base/mibs\n+mibdirs /extra/mibs\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	got := applyConfigFile(confPath, []string{"/original"}, parseNetSNMPLine)
	want := []string{"/base/mibs", "/extra/mibs"}
	if !slices.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestApplyConfigFileNetSNMPPrepend(t *testing.T) {
	dir := t.TempDir()
	confPath := filepath.Join(dir, "snmp.conf")
	if err := os.WriteFile(confPath, []byte("-mibdirs /first\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	got := applyConfigFile(confPath, []string{"/default"}, parseNetSNMPLine)
	want := []string{"/first", "/default"}
	if !slices.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestApplyConfigFileLibSMI(t *testing.T) {
	dir := t.TempDir()
	confPath := filepath.Join(dir, "smi.conf")
	if err := os.WriteFile(confPath, []byte("# Comment\npath /base/mibs\npath :/extra/mibs\nsmilint: path /ignored\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	got := applyConfigFile(confPath, []string{"/original"}, parseLibSMILine)
	want := []string{"/base/mibs", "/extra/mibs"}
	if !slices.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestApplyConfigFileMissing(t *testing.T) {
	current := []string{"/keep"}
	got := applyConfigFile("/nonexistent/file", current, parseNetSNMPLine)
	if !slices.Equal(got, current) {
		t.Errorf("missing config should return current paths unchanged, got %v", got)
	}
}

func TestSplitPaths(t *testing.T) {
	tests := []struct {
		input string
		want  []string
	}{
		{"/a:/b:/c", []string{"/a", "/b", "/c"}},
		{"/single", []string{"/single"}},
		{"", nil},
		{":/a", []string{"/a"}},          // leading empty segment skipped
		{"/a:", []string{"/a"}},          // trailing empty segment skipped
		{"/a::/b", []string{"/a", "/b"}}, // double colon, empty segment skipped
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := splitPaths(tt.input)
			if !slices.Equal(got, tt.want) {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEnvVarOverride(t *testing.T) {
	// Test that env var applied to defaults produces expected result
	defaults := []string{"/usr/share/snmp/mibs"}

	// MIBDIRS replaces
	t.Setenv("MIBDIRS", "/custom/mibs")
	got := applyNetSNMPEnv(os.Getenv("MIBDIRS"), defaults)
	want := []string{"/custom/mibs"}
	if !slices.Equal(got, want) {
		t.Errorf("MIBDIRS replace: got %v, want %v", got, want)
	}

	// MIBDIRS appends
	t.Setenv("MIBDIRS", "+/extra/mibs")
	got = applyNetSNMPEnv(os.Getenv("MIBDIRS"), defaults)
	want = []string{"/usr/share/snmp/mibs", "/extra/mibs"}
	if !slices.Equal(got, want) {
		t.Errorf("MIBDIRS append: got %v, want %v", got, want)
	}

	// SMIPATH replaces
	t.Setenv("SMIPATH", "/custom/smi")
	got = applyLibSMIEnv(os.Getenv("SMIPATH"), defaults)
	want = []string{"/custom/smi"}
	if !slices.Equal(got, want) {
		t.Errorf("SMIPATH replace: got %v, want %v", got, want)
	}

	// SMIPATH appends
	t.Setenv("SMIPATH", ":/extra/smi")
	got = applyLibSMIEnv(os.Getenv("SMIPATH"), defaults)
	want = []string{"/usr/share/snmp/mibs", "/extra/smi"}
	if !slices.Equal(got, want) {
		t.Errorf("SMIPATH append: got %v, want %v", got, want)
	}
}
