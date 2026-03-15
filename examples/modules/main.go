// Load embedded modules and inspect module metadata, imports, and visible symbols.
package main

import (
	"context"
	"embed"
	"fmt"
	"log"
	"strings"

	"github.com/golangsnmp/gomib"
	"github.com/golangsnmp/gomib/mib"
)

//go:embed mibs
var mibFS embed.FS

func main() {
	src := gomib.FS("embedded", mibFS)

	modules, err := src.ListModules()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Embedded source modules: %s\n\n", strings.Join(modules, ", "))

	m, err := gomib.Load(context.Background(),
		gomib.WithSource(src),
		gomib.WithModules("EXAMPLE-MODULES-MIB"),
	)
	if err != nil {
		log.Fatal(err)
	}

	mod := m.Module("EXAMPLE-MODULES-MIB")
	if mod == nil {
		log.Fatal("EXAMPLE-MODULES-MIB not loaded")
	}

	fmt.Printf("Module:       %s\n", mod.Name())
	fmt.Printf("Language:     %s\n", mod.Language())
	fmt.Printf("Source path:  %s\n", mod.SourcePath())
	fmt.Printf("Is base:      %v\n", mod.IsBase())
	fmt.Printf("OID:          %s\n", mod.OID())
	fmt.Printf("Organization: %s\n", mod.Organization())
	fmt.Printf("Last updated: %s\n", mod.LastUpdated())
	fmt.Println()

	fmt.Println("Imports:")
	for _, imp := range mod.Imports() {
		names := make([]string, len(imp.Symbols))
		for i, sym := range imp.Symbols {
			names[i] = sym.Name
		}
		fmt.Printf("  FROM %-20s %s\n", imp.Module, strings.Join(names, ", "))
	}
	fmt.Println()

	fmt.Printf("ImportsSymbol(%q)  = %v\n", "ExampleName", mod.ImportsSymbol("ExampleName"))
	fmt.Printf("IsImportUsed(%q)   = %v\n", "ExampleName", mod.IsImportUsed("ExampleName"))
	if srcMod := mod.ImportSource("ExampleName"); srcMod != nil {
		fmt.Printf("ImportSource(%q) = %s\n", "ExampleName", srcMod.Name())
	}
	fmt.Printf("DefinesSymbol(%q) = %v\n", "exampleDeviceName", mod.DefinesSymbol("exampleDeviceName"))
	fmt.Println()

	fmt.Println("Definitions:")
	for sym := range mod.Definitions() {
		fmt.Printf("  %-18s %s\n", sym.Name(), symbolKind(sym))
	}
	fmt.Println()

	if obj := mod.Object("exampleDeviceName"); obj != nil {
		fmt.Printf("Object: %s  %s  type=%s\n", obj.Name(), obj.OID(), obj.Type().Name())
	}

	base := m.Module("SNMPv2-SMI")
	if base != nil {
		fmt.Printf("Base module: %s (isBase=%v, sourcePath=%q)\n", base.Name(), base.IsBase(), base.SourcePath())
		if node := base.Node("enterprises"); node != nil {
			fmt.Printf("  enterprises -> %s\n", node.OID())
		}
	}
}

func symbolKind(sym mib.Symbol) string {
	switch {
	case sym.Object() != nil:
		return "object"
	case sym.Type() != nil:
		return "type"
	case sym.Notification() != nil:
		return "notification"
	case sym.Group() != nil:
		return "group"
	case sym.Compliance() != nil:
		return "compliance"
	case sym.Capability() != nil:
		return "capability"
	case sym.Node() != nil:
		return "node"
	default:
		return "unknown"
	}
}
