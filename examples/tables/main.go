// Example: tables - explore SNMP table structure.
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

	// Explore ifTable structure
	fmt.Println("=== IF-MIB::ifTable structure ===")
	ifTable := mib.ObjectByQualified("IF-MIB::ifTable")
	if ifTable != nil {
		printTableStructure(ifTable)
	}

	// Find the row entry
	fmt.Println("\n=== Row entry details ===")
	ifEntry := mib.ObjectByQualified("IF-MIB::ifEntry")
	if ifEntry != nil {
		fmt.Printf("Row: %s (%s)\n", ifEntry.Name, ifEntry.OID())
		fmt.Printf("Index columns:\n")
		for _, idx := range ifEntry.Index {
			implied := ""
			if idx.Implied {
				implied = " (IMPLIED)"
			}
			fmt.Printf("  %s%s\n", idx.Object.Name, implied)
		}
	}

	// List all columns
	fmt.Println("\n=== ifTable columns ===")
	if ifEntry != nil && ifEntry.Node != nil {
		for _, child := range ifEntry.Node.Children() {
			if child.Object != nil {
				obj := child.Object
				typeName := "<unknown>"
				if obj.Type != nil {
					typeName = obj.Type.Name
					if typeName == "" {
						typeName = obj.Type.Base.String()
					}
				}
				fmt.Printf("  .%d %s (%s, %s)\n",
					child.Arc(), obj.Name, typeName, obj.Access)
			}
		}
	}

	// Find tables with AUGMENTS
	fmt.Println("\n=== Tables with AUGMENTS ===")
	count := 0
	for _, obj := range mib.Objects() {
		if obj.Augments != nil && count < 5 {
			fmt.Printf("  %s::%s AUGMENTS %s::%s\n",
				obj.Module.Name, obj.Name,
				obj.Augments.Module.Name, obj.Augments.Name)
			count++
		}
	}

	// Find tables with compound indices
	fmt.Println("\n=== Tables with compound indices ===")
	count = 0
	for _, obj := range mib.Objects() {
		if obj.Kind() == gomib.KindRow && len(obj.Index) > 1 && count < 5 {
			fmt.Printf("  %s::%s (", obj.Module.Name, obj.Name)
			for i, idx := range obj.Index {
				if i > 0 {
					fmt.Print(", ")
				}
				fmt.Print(idx.Object.Name)
			}
			fmt.Println(")")
			count++
		}
	}
}

func printTableStructure(table *gomib.Object) {
	fmt.Printf("Table: %s\n", table.Name)
	fmt.Printf("OID: %s\n", table.OID())
	fmt.Printf("Status: %s\n", table.Status)
	if table.Description != "" {
		desc := table.Description
		if len(desc) > 100 {
			desc = desc[:97] + "..."
		}
		fmt.Printf("Description: %s\n", desc)
	}

	// Find the entry (row)
	if table.Node != nil {
		for _, child := range table.Node.Children() {
			if child.Kind == gomib.KindRow && child.Object != nil {
				entry := child.Object
				fmt.Printf("\nEntry: %s\n", entry.Name)
				fmt.Printf("  Columns: %d\n", len(child.Children()))
				fmt.Printf("  Indices: %d\n", len(entry.Index))
			}
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
