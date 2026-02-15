package gomib

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"

	"github.com/golangsnmp/gomib/internal/testutil"
)

func TestDirNonExistentPath(t *testing.T) {
	_, err := Dir("/this/path/does/not/exist/at/all")
	testutil.Error(t, err, "Dir with non-existent path should fail")
}

func TestDirNotADirectory(t *testing.T) {
	_, err := Dir("testdata/corpus/primary/ietf/IF-MIB.mib")
	testutil.Error(t, err, "Dir with a file path should fail")
}

func TestMustDirPanicsOnError(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Error("MustDir with non-existent path should panic")
		}
	}()
	MustDir("/this/path/does/not/exist")
}

func TestMustDirSucceeds(t *testing.T) {
	src := MustDir("testdata/corpus/primary/ietf")
	names, err := src.ListModules()
	testutil.NoError(t, err, "ListModules")
	testutil.Greater(t, len(names), 0, "should list modules")
}

func TestDirTreeNonExistentPath(t *testing.T) {
	_, err := DirTree("/this/path/does/not/exist/at/all")
	testutil.Error(t, err, "DirTree with non-existent path should fail")
}

func TestDirTreeNotADirectory(t *testing.T) {
	_, err := DirTree("testdata/corpus/primary/ietf/IF-MIB.mib")
	testutil.Error(t, err, "DirTree with a file path should fail")
}

func TestMustDirTreePanicsOnError(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Error("MustDirTree with non-existent path should panic")
		}
	}()
	MustDirTree("/this/path/does/not/exist")
}

func TestMustDirTreeSucceeds(t *testing.T) {
	src := MustDirTree("testdata/corpus/primary")
	names, err := src.ListModules()
	testutil.NoError(t, err, "ListModules")
	testutil.Greater(t, len(names), 10, "should list many modules")
}

func TestDirSourceFindExisting(t *testing.T) {
	src, err := Dir("testdata/corpus/primary/ietf")
	testutil.NoError(t, err, "Dir")

	result, err := src.Find("IF-MIB")
	testutil.NoError(t, err, "Find IF-MIB")
	testutil.NotNil(t, result.Reader, "Reader should not be nil")
	_ = result.Reader.Close()
	testutil.True(t, result.Path != "", "Path should be set")
}

func TestDirSourceFindNotExist(t *testing.T) {
	src, err := Dir("testdata/corpus/primary/ietf")
	testutil.NoError(t, err, "Dir")

	_, err = src.Find("TOTALLY-NONEXISTENT-MODULE")
	if err == nil {
		t.Fatal("Find should fail for non-existent module")
	}
	testutil.True(t, err == fs.ErrNotExist, "error should be fs.ErrNotExist, got %v", err)
}

func TestDirTreeSourceFindAcrossSubdirs(t *testing.T) {
	src, err := DirTree("testdata/corpus/primary")
	testutil.NoError(t, err, "DirTree")

	result, err := src.Find("IF-MIB")
	testutil.NoError(t, err, "Find IF-MIB across subdirs")
	testutil.NotNil(t, result.Reader, "Reader should not be nil")
	_ = result.Reader.Close()
}

func TestDirTreeSourceFindNotExist(t *testing.T) {
	src, err := DirTree("testdata/corpus/primary")
	testutil.NoError(t, err, "DirTree")

	_, err = src.Find("TOTALLY-NONEXISTENT-MODULE")
	if err == nil {
		t.Fatal("Find should fail for non-existent module")
	}
	testutil.True(t, err == fs.ErrNotExist, "error should be fs.ErrNotExist")
}

func TestFSSource(t *testing.T) {
	mibContent := `TEST-FS-MIB DEFINITIONS ::= BEGIN
IMPORTS
    MODULE-IDENTITY, enterprises
        FROM SNMPv2-SMI;
testFsMIB MODULE-IDENTITY
    LAST-UPDATED "202501010000Z"
    ORGANIZATION "Test"
    CONTACT-INFO "Test"
    DESCRIPTION "Test FS source"
    ::= { enterprises 99999 }
END
`
	memFS := fstest.MapFS{
		"mibs/TEST-FS-MIB.mib": &fstest.MapFile{
			Data: []byte(mibContent),
		},
	}

	src := FS("test-fs", memFS)

	result, err := src.Find("TEST-FS-MIB")
	testutil.NoError(t, err, "Find TEST-FS-MIB in FS source")
	testutil.NotNil(t, result.Reader, "Reader should not be nil")
	_ = result.Reader.Close()
	testutil.Contains(t, result.Path, "test-fs:", "Path should contain FS name prefix")

	names, err := src.ListModules()
	testutil.NoError(t, err, "ListModules")
	testutil.Equal(t, 1, len(names), "should list 1 module")
}

func TestFSSourceFindNotExist(t *testing.T) {
	memFS := fstest.MapFS{}
	src := FS("empty", memFS)

	_, err := src.Find("NONEXISTENT")
	if err == nil {
		t.Fatal("Find should fail in empty FS source")
	}
	testutil.True(t, err == fs.ErrNotExist, "error should be fs.ErrNotExist")
}

func TestFSSourceListModulesEmpty(t *testing.T) {
	memFS := fstest.MapFS{}
	src := FS("empty", memFS)

	names, err := src.ListModules()
	testutil.NoError(t, err, "ListModules on empty FS")
	testutil.Equal(t, 0, len(names), "empty FS should have 0 modules")
}

func TestFSSourceWithLoad(t *testing.T) {
	mibContent := `TEST-FS-LOAD-MIB DEFINITIONS ::= BEGIN
IMPORTS
    MODULE-IDENTITY, OBJECT-TYPE, Integer32, enterprises
        FROM SNMPv2-SMI;
testFsLoadMIB MODULE-IDENTITY
    LAST-UPDATED "202501010000Z"
    ORGANIZATION "Test"
    CONTACT-INFO "Test"
    DESCRIPTION "Test FS load integration"
    ::= { enterprises 99997 }
testFsScalar OBJECT-TYPE
    SYNTAX Integer32
    MAX-ACCESS read-only
    STATUS current
    DESCRIPTION "Test scalar"
    ::= { testFsLoadMIB 1 }
END
`
	memFS := fstest.MapFS{
		"TEST-FS-LOAD-MIB.mib": &fstest.MapFile{
			Data: []byte(mibContent),
		},
	}

	src := FS("test", memFS)
	ctx := context.Background()
	m, err := Load(ctx, WithSource(src), WithModules("TEST-FS-LOAD-MIB"))
	testutil.NoError(t, err, "Load with FS source")

	obj := m.Object("testFsScalar")
	testutil.NotNil(t, obj, "testFsScalar should resolve from FS source")
}

func TestMultiSourceFindOrder(t *testing.T) {
	src1, err := Dir("testdata/corpus/primary/ietf")
	testutil.NoError(t, err, "Dir ietf")

	src2, err := Dir("testdata/corpus/primary/iana")
	testutil.NoError(t, err, "Dir iana")

	multi := Multi(src1, src2)

	result, err := multi.Find("IF-MIB")
	testutil.NoError(t, err, "Find IF-MIB from multi source")
	_ = result.Reader.Close()
}

func TestMultiSourceListModulesCombines(t *testing.T) {
	src1, err := Dir("testdata/corpus/primary/ietf")
	testutil.NoError(t, err, "Dir ietf")

	src2, err := Dir("testdata/corpus/primary/iana")
	testutil.NoError(t, err, "Dir iana")

	multi := Multi(src1, src2)

	names, err := multi.ListModules()
	testutil.NoError(t, err, "ListModules")

	names1, _ := src1.ListModules()
	names2, _ := src2.ListModules()
	// ietf and iana have no overlapping names, so combined == sum
	testutil.Equal(t, len(names1)+len(names2), len(names),
		"Multi should combine module lists from all sources")
}

func TestMultiSourceListModulesDeduplicates(t *testing.T) {
	tmpDir := t.TempDir()
	content := []byte("X DEFINITIONS ::= BEGIN\nEND\n")
	err := os.WriteFile(filepath.Join(tmpDir, "SHARED-MIB.mib"), content, 0644)
	testutil.NoError(t, err, "write file")

	src1, err := Dir(tmpDir)
	testutil.NoError(t, err, "Dir 1")
	src2, err := Dir(tmpDir)
	testutil.NoError(t, err, "Dir 2")

	multi := Multi(src1, src2)
	names, err := multi.ListModules()
	testutil.NoError(t, err, "ListModules")

	// Count occurrences of SHARED-MIB
	count := 0
	for _, n := range names {
		if n == "SHARED-MIB" {
			count++
		}
	}
	testutil.Equal(t, 1, count, "SHARED-MIB should appear exactly once, got %d", count)
}

func TestMultiSourceFindNotExist(t *testing.T) {
	src1, err := Dir("testdata/corpus/primary/ietf")
	testutil.NoError(t, err, "Dir")

	multi := Multi(src1)
	_, err = multi.Find("TOTALLY-NONEXISTENT-MODULE")
	testutil.True(t, err == fs.ErrNotExist, "Multi.Find should return fs.ErrNotExist")
}

func TestWithExtensions(t *testing.T) {
	tmpDir := t.TempDir()
	content := `EXT-TEST-MIB DEFINITIONS ::= BEGIN
IMPORTS MODULE-IDENTITY, enterprises FROM SNMPv2-SMI;
extTestMIB MODULE-IDENTITY
    LAST-UPDATED "202501010000Z"
    ORGANIZATION "Test"
    CONTACT-INFO "Test"
    DESCRIPTION "Test"
    ::= { enterprises 99996 }
END
`
	err := os.WriteFile(filepath.Join(tmpDir, "EXT-TEST-MIB.custom"), []byte(content), 0644)
	testutil.NoError(t, err, "write test file")

	srcDefault, err := Dir(tmpDir)
	testutil.NoError(t, err, "Dir default")
	names, err := srcDefault.ListModules()
	testutil.NoError(t, err, "ListModules default")
	testutil.Equal(t, 0, len(names), "default extensions should not find .custom files")

	srcCustom, err := Dir(tmpDir, WithExtensions(".custom"))
	testutil.NoError(t, err, "Dir custom")
	names, err = srcCustom.ListModules()
	testutil.NoError(t, err, "ListModules custom")
	testutil.Equal(t, 1, len(names), "custom extensions should find .custom files")
}

func TestLoadContextCancellation(t *testing.T) {
	src, err := DirTree("testdata/corpus/primary")
	testutil.NoError(t, err, "DirTree")

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	_, err = Load(ctx, WithSource(src))
	// Pre-cancelled context should produce context.Canceled.
	testutil.Error(t, err, "Load with cancelled context should return error")
	testutil.Equal(t, context.Canceled, err, "error should be context.Canceled")
}

func TestLoadWithModulesContextCancellation(t *testing.T) {
	src, err := DirTree("testdata/corpus/primary")
	testutil.NoError(t, err, "DirTree")

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err = Load(ctx, WithSource(src), WithModules("IF-MIB"))
	// Pre-cancelled context should produce context.Canceled.
	testutil.Error(t, err, "Load with cancelled context should return error")
	testutil.Equal(t, context.Canceled, err, "error should be context.Canceled")
}

func TestLooksLikeMIBContent(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    bool
	}{
		{"valid MIB", "FOO-MIB DEFINITIONS ::= BEGIN\n", true},
		{"empty", "", false},
		{"no DEFINITIONS", "foo bar ::= BEGIN\n", false},
		{"no ::=", "FOO-MIB DEFINITIONS BEGIN\n", false},
		{"binary null byte", "FOO-MIB DEFINITIONS\x00::= BEGIN\n", false},
		{"binary at start", "\x00FOO-MIB DEFINITIONS ::= BEGIN\n", false},
		{"just DEFINITIONS", "DEFINITIONS", false},
		{"just ::=", "::=", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := looksLikeMIBContent([]byte(tt.content))
			if got != tt.want {
				t.Errorf("looksLikeMIBContent(%q) = %v, want %v", tt.content, got, tt.want)
			}
		})
	}
}

func TestModuleNameFromPath(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{"/usr/share/snmp/mibs/IF-MIB.mib", "IF-MIB"},
		{"/usr/share/snmp/mibs/IF-MIB", "IF-MIB"},
		{"IF-MIB.mib", "IF-MIB"},
		{"IF-MIB.smi", "IF-MIB"},
		{"IF-MIB.txt", "IF-MIB"},
		{"IF-MIB.my", "IF-MIB"},
		{"IF-MIB", "IF-MIB"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := moduleNameFromPath(tt.path)
			if got != tt.want {
				t.Errorf("moduleNameFromPath(%q) = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}

func TestDefaultExtensions(t *testing.T) {
	exts := DefaultExtensions()
	if len(exts) == 0 {
		t.Fatal("DefaultExtensions should return non-empty list")
	}

	extSet := make(map[string]bool)
	for _, ext := range exts {
		extSet[ext] = true
	}

	testutil.True(t, extSet[""], "should include empty string (extensionless files)")
	testutil.True(t, extSet[".mib"], "should include .mib")
}

func TestLoadEmptyDirProducesEmptyMib(t *testing.T) {
	tmpDir := t.TempDir()
	src, err := Dir(tmpDir)
	testutil.NoError(t, err, "Dir empty")

	ctx := context.Background()
	m, err := Load(ctx, WithSource(src))
	testutil.NoError(t, err, "Load from empty dir should succeed")
	testutil.NotNil(t, m, "should return non-nil Mib")
	testutil.Equal(t, 0, len(m.Objects()), "empty source should have no user objects")
}

func TestLoadMultipleModules(t *testing.T) {
	src, err := DirTree("testdata/corpus/primary")
	testutil.NoError(t, err, "DirTree")

	ctx := context.Background()
	m, err := Load(ctx, WithSource(src), WithModules("IF-MIB", "SNMPv2-MIB"))
	testutil.NoError(t, err, "Load")

	testutil.NotNil(t, m.Module("IF-MIB"), "IF-MIB should be loaded")
	testutil.NotNil(t, m.Module("SNMPv2-MIB"), "SNMPv2-MIB should be loaded")
}

func TestLoadModulesEmptyList(t *testing.T) {
	src, err := DirTree("testdata/corpus/primary")
	testutil.NoError(t, err, "DirTree")

	ctx := context.Background()
	m, err := Load(ctx, WithSource(src), WithModules())
	testutil.NoError(t, err, "Load with empty module list should succeed")
	testutil.NotNil(t, m, "should return non-nil Mib")
}

func TestLoadNoSourceModules(t *testing.T) {
	ctx := context.Background()
	_, err := Load(ctx, WithModules("IF-MIB"))
	testutil.Error(t, err, "Load with no source should fail")
}
