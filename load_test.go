package gomib

import (
	"context"
	"testing"

	"github.com/golangsnmp/gomib/internal/testutil"
)

func TestLoadSingleMIB(t *testing.T) {
	src, err := DirTree("testdata/corpus/primary")
	if err != nil {
		t.Fatalf("DirTree failed: %v", err)
	}

	ctx := context.Background()
	mib, err := LoadModules(ctx, []string{"IF-MIB"}, src)
	if err != nil {
		t.Fatalf("LoadModules failed: %v", err)
	}

	// Basic sanity checks
	testutil.NotNil(t, mib, "mib should not be nil")
	testutil.Greater(t, mib.ModuleCount(), 0, "should have loaded modules")
	testutil.Greater(t, mib.ObjectCount(), 0, "should have resolved objects")

	// Check IF-MIB specifically
	ifMIB := mib.Module("IF-MIB")
	testutil.NotNil(t, ifMIB, "IF-MIB module should be found")

	// Check a well-known object
	ifIndex := mib.FindObject("ifIndex")
	testutil.NotNil(t, ifIndex, "ifIndex object should be found")
}

func TestLoadAllCorpus(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping corpus load in short mode")
	}

	src, err := DirTree("testdata/corpus/primary")
	if err != nil {
		t.Fatalf("DirTree failed: %v", err)
	}

	ctx := context.Background()
	mib, err := Load(ctx, src)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	testutil.Greater(t, mib.ModuleCount(), 50, "should have loaded many modules")
	testutil.Greater(t, mib.ObjectCount(), 1000, "should have resolved many objects")

	t.Logf("Loaded %d modules, %d objects, %d types",
		mib.ModuleCount(), mib.ObjectCount(), mib.TypeCount())
}
