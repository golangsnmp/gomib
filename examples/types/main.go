// Example: types - explore type definitions and textual conventions.
package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/golangsnmp/gomib"
)

func main() {
	source, err := gomib.DirTree(findCorpus())
	if err != nil {
		log.Fatalf("failed to open MIB directory: %v", err)
	}

	mib, err := gomib.Load(source)
	if err != nil {
		log.Fatalf("failed to load MIBs: %v", err)
	}

	// Look up a textual convention
	fmt.Println("=== Textual Convention: DisplayString ===")
	displayString := mib.Type("DisplayString")
	if displayString != nil {
		printType(displayString)
	}

	// Type with enumerations
	fmt.Println("\n=== Enumerated type from object ===")
	ifAdminStatus := mib.ObjectByQualified("IF-MIB::ifAdminStatus")
	if ifAdminStatus != nil && ifAdminStatus.Type != nil {
		fmt.Printf("ifAdminStatus type: %s (base: %s)\n",
			ifAdminStatus.Type.Name, ifAdminStatus.Type.Base)
		if len(ifAdminStatus.NamedValues) > 0 {
			fmt.Println("  Named values:")
			for _, nv := range ifAdminStatus.NamedValues {
				fmt.Printf("    %s(%d)\n", nv.Label, nv.Value)
			}
		}
	}

	// Find all textual conventions
	fmt.Println("\n=== Textual Conventions (first 10) ===")
	count := 0
	for _, t := range mib.Types() {
		if t.IsTC && count < 10 {
			fmt.Printf("  %s (%s) from %s\n", t.Name, t.Base, t.Module.Name)
			count++
		}
	}

	// Object with size constraint
	fmt.Println("\n=== Object with SIZE constraint ===")
	sysDescr := mib.Object("sysDescr")
	if sysDescr != nil {
		fmt.Printf("sysDescr:\n")
		fmt.Printf("  Type: %s\n", sysDescr.Type.Name)
		fmt.Printf("  Base: %s\n", sysDescr.Type.Base)
		if len(sysDescr.Size) > 0 {
			fmt.Printf("  Size: ")
			for i, r := range sysDescr.Size {
				if i > 0 {
					fmt.Print(" | ")
				}
				if r.Min == r.Max {
					fmt.Printf("%d", r.Min)
				} else {
					fmt.Printf("%d..%d", r.Min, r.Max)
				}
			}
			fmt.Println()
		}
		if sysDescr.Hint != "" {
			fmt.Printf("  Hint: %s\n", sysDescr.Hint)
		}
	}

	// Counter64 type
	fmt.Println("\n=== Counter64 objects (first 5) ===")
	count = 0
	for _, obj := range mib.Objects() {
		if obj.Type != nil && obj.Type.Base == gomib.BaseCounter64 && count < 5 {
			fmt.Printf("  %s::%s\n", obj.Module.Name, obj.Name)
			count++
		}
	}
}

func printType(t *gomib.Type) {
	fmt.Printf("Name: %s\n", t.Name)
	fmt.Printf("Module: %s\n", t.Module.Name)
	fmt.Printf("Base: %s\n", t.Base)
	fmt.Printf("IsTC: %v\n", t.IsTC)
	if t.Parent != nil {
		fmt.Printf("Parent: %s\n", t.Parent.Name)
	}
	if len(t.Size) > 0 {
		fmt.Printf("Size: %v\n", t.Size)
	}
	if t.Hint != "" {
		fmt.Printf("Hint: %s\n", t.Hint)
	}
	if t.Description != "" {
		desc := t.Description
		if len(desc) > 80 {
			desc = desc[:77] + "..."
		}
		fmt.Printf("Description: %s\n", desc)
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
