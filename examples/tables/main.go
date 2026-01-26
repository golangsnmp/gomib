// Example: tables - explore SNMP table structure.
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
	source, err := gomib.DirTree(findCorpus())
	if err != nil {
		log.Fatalf("failed to open MIB directory: %v", err)
	}

	mib, err := gomib.Load(context.Background(), source)
	if err != nil {
		log.Fatalf("failed to load MIBs: %v", err)
	}

	// Explore ifTable structure
	fmt.Println("=== IF-MIB::ifTable structure ===")
	ifTable := mib.FindObject("IF-MIB::ifTable")
	if ifTable != nil {
		printTableStructure(ifTable)
	}

	// Find the row entry
	fmt.Println("\n=== Row entry details ===")
	ifEntry := mib.FindObject("IF-MIB::ifEntry")
	if ifEntry != nil {
		fmt.Printf("Row: %s (%s)\n", ifEntry.Name(), ifEntry.OID())
		fmt.Printf("Index columns:\n")
		for _, idx := range ifEntry.Index() {
			implied := ""
			if idx.Implied {
				implied = " (IMPLIED)"
			}
			fmt.Printf("  %s%s\n", idx.Object.Name(), implied)
		}
	}

	// List all columns using Columns() method
	fmt.Println("\n=== ifTable columns ===")
	for _, col := range ifTable.Columns() {
		typeName := "<unknown>"
		if col.Type() != nil {
			typeName = col.Type().Name()
			if typeName == "" {
				typeName = col.Type().Base().String()
			}
		}
		fmt.Printf("  .%d %s (%s, %s)\n",
			col.Node().Arc(), col.Name(), typeName, col.Access())
	}

	// Find tables with AUGMENTS
	fmt.Println("\n=== Tables with AUGMENTS ===")
	count := 0
	for _, obj := range mib.Objects() {
		if obj.Augments() != nil && count < 5 {
			fmt.Printf("  %s::%s AUGMENTS %s::%s\n",
				obj.Module().Name(), obj.Name(),
				obj.Augments().Module().Name(), obj.Augments().Name())
			count++
		}
	}

	// Find tables with compound indices
	fmt.Println("\n=== Tables with compound indices ===")
	count = 0
	for _, obj := range mib.Objects() {
		if obj.Kind() == gomib.KindRow && len(obj.Index()) > 1 && count < 5 {
			fmt.Printf("  %s::%s (", obj.Module().Name(), obj.Name())
			for i, idx := range obj.Index() {
				if i > 0 {
					fmt.Print(", ")
				}
				fmt.Print(idx.Object.Name())
			}
			fmt.Println(")")
			count++
		}
	}
}

func printTableStructure(table gomib.Object) {
	fmt.Printf("Table: %s\n", table.Name())
	fmt.Printf("OID: %s\n", table.OID())
	fmt.Printf("Status: %s\n", table.Status())
	if desc := table.Description(); desc != "" {
		if len(desc) > 100 {
			desc = desc[:97] + "..."
		}
		fmt.Printf("Description: %s\n", desc)
	}

	// Get the entry (row) using Entry() method
	if entry := table.Entry(); entry != nil {
		fmt.Printf("\nEntry: %s\n", entry.Name())
		fmt.Printf("  Columns: %d\n", len(table.Columns()))
		fmt.Printf("  Indices: %d\n", len(entry.Index()))
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
