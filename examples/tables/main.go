// Load several MIBs and demonstrate the table API: structure, indexes,
// columns, AUGMENTS relationships, index encoding, and navigation.
//
// Modules are chosen to cover diverse index patterns:
//   - IF-MIB: integer indexes, AUGMENTS, compound indexes, length-prefixed
//   - OSPF-MIB: IpAddress indexes, 3-component compound indexes
//   - BRIDGE-MIB: fixed-size OCTET STRING (MacAddress) indexes
//   - SNMP-TARGET-MIB: IMPLIED keyword
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"strings"

	"github.com/golangsnmp/gomib"
	"github.com/golangsnmp/gomib/mib"
)

var modules = []string{
	"IF-MIB",
	"OSPF-MIB",
	"BRIDGE-MIB",
	"SNMP-TARGET-MIB",
}

func main() {
	path := flag.String("p", "", "MIB search path (default: system paths)")
	flag.Parse()

	var src gomib.Source
	if *path != "" {
		var err error
		src, err = gomib.Dir(*path)
		if err != nil {
			log.Fatal(err)
		}
	}

	var opts []gomib.LoadOption
	if src != nil {
		opts = append(opts, gomib.WithSource(src))
	}
	opts = append(opts, gomib.WithModules(modules...), gomib.WithSystemPaths())
	m, err := gomib.Load(context.Background(), opts...)
	if err != nil {
		log.Fatal(err)
	}

	for _, modName := range modules {
		mod := m.Module(modName)
		if mod == nil {
			continue
		}
		tables := mod.Tables()
		if len(tables) == 0 {
			continue
		}

		fmt.Printf("=== %s ===\n\n", modName)
		for _, table := range tables {
			printTable(table)
		}
	}
}

func printTable(table *mib.Object) {
	row := table.Entry()
	if row == nil {
		return
	}

	// Table and row identification.
	fmt.Printf("TABLE %s (%s)\n", table.Name(), table.OID())
	fmt.Printf("  ROW %s\n", row.Name())

	// Kind checks.
	fmt.Printf("  table.IsTable()=%v  row.IsRow()=%v\n",
		table.IsTable(), row.IsRow())

	// Navigation: row -> table, table -> row.
	fmt.Printf("  row.Table()=%s  table.Entry()=%s\n",
		row.Table().Name(), table.Entry().Name())

	// INDEX entries with encoding classification.
	idxs := row.EffectiveIndexes()
	var parts []string
	for _, idx := range idxs {
		s := indexName(idx)
		if idx.Implied {
			s += " IMPLIED"
		}
		s += " [" + idx.Encoding.String() + "]"
		parts = append(parts, s)
	}
	fmt.Printf("  INDEX %s\n", strings.Join(parts, ", "))

	// AUGMENTS relationships.
	if aug := row.Augments(); aug != nil {
		fmt.Printf("  AUGMENTS %s\n", aug.Name())
	}
	if augBy := row.AugmentedBy(); len(augBy) > 0 {
		names := make([]string, len(augBy))
		for i, a := range augBy {
			names[i] = a.Name()
		}
		fmt.Printf("  AUGMENTED-BY %s\n", strings.Join(names, ", "))
	}

	// Columns with index/data distinction.
	fmt.Println()
	fmt.Printf("  %-28s %-20s %-18s %-18s %s\n",
		"COLUMN", "TYPE", "BASE", "ACCESS", "ROLE")
	fmt.Printf("  %-28s %-20s %-18s %-18s %s\n",
		"------", "----", "----", "------", "----")
	for _, col := range row.Columns() {
		typeName, base := "", ""
		if t := col.Type(); t != nil {
			typeName = t.Name()
			base = t.EffectiveBase().String()
		}
		role := "data"
		if col.IsIndex() {
			role = "index"
		}

		// EffectiveIndexes works on columns too, delegating to the parent row.
		colIdxs := col.EffectiveIndexes()
		_ = colIdxs // same result as row.EffectiveIndexes()

		fmt.Printf("  %-28s %-20s %-18s %-18s %s\n",
			col.Name(), typeName, base, col.Access(), role)
	}
	fmt.Println()
}

func indexName(idx mib.IndexEntry) string {
	if idx.Object != nil {
		return idx.Object.Name()
	}
	return idx.TypeName
}
