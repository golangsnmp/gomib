// Example: modules - load specific modules and explore their contents.
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/golangsnmp/gomib"
)

func main() {
	corpusPath := findCorpus()

	// Create source from directory tree
	source, err := gomib.DirTree(corpusPath)
	if err != nil {
		log.Fatalf("failed to open MIB directory: %v", err)
	}

	// Load only specific modules (with their dependencies)
	fmt.Println("=== Loading IF-MIB and SNMPv2-MIB ===")
	mib, err := gomib.LoadModules(context.Background(),
		[]string{"IF-MIB", "SNMPv2-MIB"},
		source,
	)
	if err != nil {
		log.Fatalf("failed to load modules: %v", err)
	}

	fmt.Printf("Loaded %d modules (including dependencies)\n", mib.ModuleCount())
	fmt.Println("\nLoaded modules:")
	for _, mod := range mib.Modules() {
		fmt.Printf("  %s (%s)\n", mod.Name, mod.Language)
	}

	// Explore a specific module
	fmt.Println("\n=== IF-MIB details ===")
	ifMIB := mib.Module("IF-MIB")
	if ifMIB != nil {
		printModuleDetails(ifMIB)
	}

	// Module identity OID
	fmt.Println("\n=== Module identity OIDs ===")
	for _, mod := range mib.Modules() {
		if mod.OID != nil {
			fmt.Printf("  %s: %s\n", mod.Name, mod.OID)
		}
	}

	// List objects by module
	fmt.Println("\n=== Objects per module ===")
	for _, mod := range mib.Modules() {
		objs := mod.Objects()
		if len(objs) > 0 {
			fmt.Printf("  %s: %d objects\n", mod.Name, len(objs))
		}
	}

	// List types by module
	fmt.Println("\n=== Types per module ===")
	for _, mod := range mib.Modules() {
		types := mod.Types()
		if len(types) > 0 {
			fmt.Printf("  %s: %d types\n", mod.Name, len(types))
		}
	}
}

func printModuleDetails(mod *gomib.Module) {
	fmt.Printf("Name: %s\n", mod.Name)
	fmt.Printf("Language: %s\n", mod.Language)
	if mod.OID != nil {
		fmt.Printf("OID: %s\n", mod.OID)
	}
	if mod.Organization != "" {
		org := mod.Organization
		if len(org) > 60 {
			org = org[:57] + "..."
		}
		fmt.Printf("Organization: %s\n", org)
	}
	fmt.Printf("Objects: %d\n", len(mod.Objects()))
	fmt.Printf("Types: %d\n", len(mod.Types()))
	fmt.Printf("Notifications: %d\n", len(mod.Notifications()))

	if len(mod.Revisions) > 0 {
		fmt.Printf("Revisions: %d\n", len(mod.Revisions))
		for i, rev := range mod.Revisions {
			if i >= 3 {
				fmt.Printf("  ... and %d more\n", len(mod.Revisions)-3)
				break
			}
			fmt.Printf("  %s\n", rev.Date)
		}
	}
}

func findCorpus() string {
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
	log.Fatal("could not find test corpus")
	return ""
}
