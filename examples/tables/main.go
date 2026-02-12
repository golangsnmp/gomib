// Load IF-MIB and print table structure: row, indexes, and columns with types.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"strings"

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
	for _, table := range mod.Tables() {
		row := table.Entry()
		if row == nil {
			continue
		}

		fmt.Printf("TABLE %s (%s)\n", table.Name(), table.OID())
		fmt.Printf("  ROW   %s\n", row.Name())

		idxs := row.EffectiveIndexes()
		idxNames := make([]string, len(idxs))
		for i, idx := range idxs {
			name := idx.Object.Name()
			if idx.Implied {
				name += " (IMPLIED)"
			}
			idxNames[i] = name
		}
		fmt.Printf("  INDEX %s\n", strings.Join(idxNames, ", "))

		fmt.Println()
		fmt.Printf("  %-28s %-20s %-16s %s\n", "COLUMN", "TYPE", "BASE", "ACCESS")
		fmt.Printf("  %-28s %-20s %-16s %s\n", "------", "----", "----", "------")
		for _, col := range row.Columns() {
			base := ""
			if col.Type() != nil {
				base = col.Type().EffectiveBase().String()
			}
			typeName := ""
			if col.Type() != nil {
				typeName = col.Type().Name()
			}
			fmt.Printf("  %-28s %-20s %-16s %s\n",
				col.Name(), typeName, base, col.Access())
		}
		fmt.Println()
	}
}
