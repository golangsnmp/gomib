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

	var opts []gomib.LoadOption
	if src != nil {
		opts = append(opts, gomib.WithSource(src))
	}
	opts = append(opts, gomib.WithModules("IF-MIB"), gomib.WithSystemPaths())
	m, err := gomib.Load(context.Background(), opts...)
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
		typeName := ""
		if t := obj.Type(); t != nil {
			typeName = t.Name()
		}
		fmt.Printf("%-24s %-30s %-16s %s\n",
			obj.Name(), obj.OID(), typeName, obj.Access())
	}
}
