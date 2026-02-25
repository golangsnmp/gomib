package testutil

import (
	"os"
	"path/filepath"
	"testing"
)

// moduleRoot returns the absolute path to the module root by walking up
// from the current working directory to find go.mod.
// Returns "" if go.mod cannot be found.
func moduleRoot() string {
	dir, err := os.Getwd()
	if err != nil {
		return ""
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

// TestdataDir returns the absolute path to a subdirectory under testdata/.
// It finds the module root by walking up to go.mod, so it works from any
// package depth. Example: TestdataDir("corpus", "primary") returns the
// path to testdata/corpus/primary.
func TestdataDir(parts ...string) string {
	elems := append([]string{moduleRoot(), "testdata"}, parts...)
	return filepath.Join(elems...)
}

// PrimaryCorpusDir returns the absolute path to testdata/corpus/primary.
func PrimaryCorpusDir() string {
	return TestdataDir("corpus", "primary")
}

// ProblemsCorpusDir returns the absolute path to testdata/corpus/problems.
func ProblemsCorpusDir() string {
	return TestdataDir("corpus", "problems")
}

// AddCorpusDirSeeds loads all files from a directory as fuzz seeds.
func AddCorpusDirSeeds(f *testing.F, dir string) {
	f.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
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
	AddCorpusDirSeeds(f, ProblemsCorpusDir())
}

// AddPrimaryCorpusSeeds loads real-world vendor MIBs as fuzz seeds.
func AddPrimaryCorpusSeeds(f *testing.F) {
	f.Helper()
	primaryDir := PrimaryCorpusDir()
	entries, err := os.ReadDir(primaryDir)
	if err != nil {
		return
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		AddCorpusDirSeeds(f, filepath.Join(primaryDir, e.Name()))
	}
}
