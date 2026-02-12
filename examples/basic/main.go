// Load IF-MIB and print a module summary with selected objects.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"

	"github.com/golangsnmp/gomib"
)

func main() {
	path := flag.String("p", "", "MIB search path (default: system paths)")
	flag.Parse()

	var src gomib.Source
	if *path != "" {
		var err error
		src, err = gomib.DirTree(*path)
		if err != nil {
			log.Fatal(err)
		}
	}

	m, err := gomib.LoadModules(context.Background(), []string{"IF-MIB"}, src, gomib.WithSystemPaths())
	if err != nil {
		log.Fatal(err)
	}

	mod := m.Module("IF-MIB")
	fmt.Printf("Module:    %s\n", mod.Name())
	fmt.Printf("Language:  %s\n", mod.Language())
	fmt.Printf("Objects:   %d\n", len(mod.Objects()))
	fmt.Printf("Tables:    %d\n", len(mod.Tables()))
	fmt.Printf("Scalars:   %d\n", len(mod.Scalars()))
	fmt.Printf("Types:     %d\n", len(mod.Types()))
	fmt.Println()

	fmt.Printf("%-24s %-30s %-16s %s\n", "NAME", "OID", "TYPE", "ACCESS")
	fmt.Printf("%-24s %-30s %-16s %s\n", "----", "---", "----", "------")
	for _, obj := range mod.Objects() {
		if !obj.IsScalar() && !obj.IsColumn() {
			continue
		}
		fmt.Printf("%-24s %-30s %-16s %s\n",
			obj.Name(), obj.OID(), obj.Type().Name(), obj.Access())
	}
}
