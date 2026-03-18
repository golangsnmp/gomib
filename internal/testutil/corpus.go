package testutil

import (
	"os"
	"path/filepath"
	"testing"
)

// moduleRoot returns the absolute path to the module root by walking up
// from the current working directory to find go.mod. Fails the test if
// go.mod cannot be found.
func moduleRoot(t testing.TB) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("testutil: os.Getwd failed: %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("testutil: could not find go.mod in any parent directory")
		}
		dir = parent
	}
}

// TestdataDir returns the absolute path to a subdirectory under testdata/.
// It finds the module root by walking up to go.mod, so it works from any
// package depth. Example: TestdataDir(t, "corpus", "primary") returns the
// path to testdata/corpus/primary.
func TestdataDir(t testing.TB, parts ...string) string {
	t.Helper()
	elems := append([]string{moduleRoot(t), "testdata"}, parts...)
	return filepath.Join(elems...)
}

// PrimaryCorpusDir returns the absolute path to testdata/corpus/primary.
func PrimaryCorpusDir(t testing.TB) string {
	t.Helper()
	return TestdataDir(t, "corpus", "primary")
}

// ProblemsCorpusDir returns the absolute path to testdata/corpus/problems.
func ProblemsCorpusDir(t testing.TB) string {
	t.Helper()
	return TestdataDir(t, "corpus", "problems")
}

// AddCorpusDirSeeds loads all files from a directory as fuzz seeds.
func AddCorpusDirSeeds(f *testing.F, dir string) {
	f.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		f.Fatalf("testutil: failed to read corpus dir %s: %v", dir, err)
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err == nil {
			f.Add(data)
		}
	}
}

// AddProblemCorpusSeeds loads synthetic problem MIBs as fuzz seeds.
func AddProblemCorpusSeeds(f *testing.F) {
	f.Helper()
	AddCorpusDirSeeds(f, ProblemsCorpusDir(f))
}

// AddPrimaryCorpusSeeds loads real-world vendor MIBs as fuzz seeds.
func AddPrimaryCorpusSeeds(f *testing.F) {
	f.Helper()
	primaryDir := PrimaryCorpusDir(f)
	entries, err := os.ReadDir(primaryDir)
	if err != nil {
		f.Fatalf("testutil: failed to read primary corpus dir %s: %v", primaryDir, err)
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		AddCorpusDirSeeds(f, filepath.Join(primaryDir, e.Name()))
	}
}
