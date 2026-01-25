// Example: basic - load MIBs and explore the resolved model.
package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/golangsnmp/gomib"
)

func main() {
	// Find the test corpus relative to this example
	corpusPath := findCorpus()

	// Create a source from the corpus directory tree
	source, err := gomib.DirTree(corpusPath)
	if err != nil {
		log.Fatalf("failed to open MIB directory: %v", err)
	}

	// Load all MIBs from the source
	mib, err := gomib.Load(source)
	if err != nil {
		log.Fatalf("failed to load MIBs: %v", err)
	}

	// Print summary
	fmt.Printf("Loaded %d modules, %d objects, %d types, %d notifications\n",
		mib.ModuleCount(), mib.ObjectCount(), mib.TypeCount(), mib.NotificationCount())

	// Check for unresolved references
	if !mib.IsComplete() {
		fmt.Printf("\nUnresolved references: %d\n", len(mib.Unresolved()))
		for _, ref := range mib.Unresolved()[:min(5, len(mib.Unresolved()))] {
			fmt.Printf("  %s: %s (in %s)\n", ref.Kind, ref.Symbol, ref.Module)
		}
	}

	// List loaded modules
	fmt.Println("\nModules:")
	for _, mod := range mib.Modules()[:min(10, len(mib.Modules()))] {
		fmt.Printf("  %s (%s)\n", mod.Name, mod.Language)
	}
	if len(mib.Modules()) > 10 {
		fmt.Printf("  ... and %d more\n", len(mib.Modules())-10)
	}
}

func findCorpus() string {
	// Try relative to working directory
	candidates := []string{
		"testdata/corpus/primary",
		"../testdata/corpus/primary",
		"gomib/testdata/corpus/primary",
	}
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			abs, _ := filepath.Abs(p)
			return abs
		}
	}
	log.Fatal("could not find test corpus; run from gomib directory")
	return ""
}
