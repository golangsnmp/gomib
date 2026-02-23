package cliutil

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGetOutputReturnsNoopCleanupOnError(t *testing.T) {
	// Use a path that cannot be created.
	badPath := filepath.Join(t.TempDir(), "no-such-dir", "file.txt")
	_, cleanup, err := GetOutput(badPath)
	if err == nil {
		t.Fatal("expected error for invalid path")
	}
	// cleanup must be non-nil so callers can safely defer it.
	if cleanup == nil {
		t.Fatal("cleanup function must not be nil on error")
	}
	cleanup() // must not panic
}

func TestGetOutputStdout(t *testing.T) {
	f, cleanup, err := GetOutput("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer cleanup()
	if f != os.Stdout {
		t.Error("expected os.Stdout for empty path")
	}
}

func TestGetOutputFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "out.txt")
	f, cleanup, err := GetOutput(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer cleanup()
	if f == nil {
		t.Fatal("expected non-nil file")
	}
	if f.Name() != path {
		t.Errorf("file name = %q, want %q", f.Name(), path)
	}
}
