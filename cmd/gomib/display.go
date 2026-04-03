package main

import (
	"fmt"
	"strings"

	"github.com/golangsnmp/gomib/mib"
)

// printObjectIndexAugments prints Index, Augments, AugmentedBy, and EffectiveIndex.
func printObjectIndexAugments(obj *mib.Object) {
	if len(obj.Index()) > 0 {
		fmt.Printf("Index:   [%s]\n", formatIndexList(obj.Index()))
	}
	if obj.Augments() != nil {
		fmt.Printf("Augments: %s\n", obj.Augments().Name())
	}
	if augBy := obj.AugmentedBy(); len(augBy) > 0 {
		names := make([]string, len(augBy))
		for i, a := range augBy {
			names[i] = a.Name()
		}
		fmt.Printf("AugmentedBy: %s\n", strings.Join(names, ", "))
	}
	if obj.Augments() != nil {
		if effIdx := obj.EffectiveIndexes(); len(effIdx) > 0 {
			fmt.Printf("EffectiveIndex: [%s]\n", formatIndexList(effIdx))
		}
	}
}

// printObjectEnumsBits prints enum values or BITS for an object.
func printObjectEnumsBits(obj *mib.Object) {
	enums := obj.EffectiveEnums()
	bits := obj.EffectiveBits()
	if len(enums) > 0 && len(bits) == 0 {
		fmt.Println("Values:")
		for _, v := range enums {
			fmt.Printf("  %s(%d)\n", v.Label, v.Value)
		}
	}
	if len(bits) > 0 {
		fmt.Println("Bits:")
		for _, b := range bits {
			fmt.Printf("  %s(%d)\n", b.Label, b.Value)
		}
	}
}

// printObjectColumnContext prints IsIndex, Row, and Table for column objects.
func printObjectColumnContext(obj *mib.Object) {
	if obj.Kind() != mib.KindColumn {
		return
	}
	fmt.Printf("IsIndex: %v\n", obj.IsIndex())
	if row := obj.Row(); row != nil {
		fmt.Printf("Row:     %s\n", row.Name())
	}
	if tbl := obj.Table(); tbl != nil {
		fmt.Printf("Table:   %s\n", tbl.Name())
	}
}

// printObjectColumns prints the column table for table or row objects.
func printObjectColumns(obj *mib.Object) {
	if obj.Kind() != mib.KindTable && obj.Kind() != mib.KindRow {
		return
	}
	cols := obj.Columns()
	if len(cols) > 0 {
		fmt.Println("Columns:")
		printColumnTable(cols)
	}
}

// formatIndexList formats a slice of index entries as a comma-separated
// string with IMPLIED prefix and encoding suffix.
func formatIndexList(indexes []mib.IndexEntry) string {
	strs := make([]string, 0, len(indexes))
	for _, idx := range indexes {
		name := "(unknown)"
		if idx.Object != nil {
			name = idx.Object.Name()
		}
		if idx.Implied {
			name = "IMPLIED " + name
		}
		if idx.Encoding != mib.IndexEncodingUnknown {
			name += " [" + idx.Encoding.String() + "]"
		}
		strs = append(strs, name)
	}
	return strings.Join(strs, ", ")
}

// printColumnTable prints a formatted table of columns for a table/row object.
func printColumnTable(cols []*mib.Object) {
	fmt.Printf("  %-28s %-20s %-18s %-18s %s\n",
		"COLUMN", "TYPE", "BASE", "ACCESS", "ROLE")
	fmt.Printf("  %-28s %-20s %-18s %-18s %s\n",
		"------", "----", "----", "------", "----")
	for _, col := range cols {
		typeName, base := "", ""
		if t := col.Type(); t != nil {
			typeName = t.Name()
			if typeName == "" {
				typeName = t.Base().String()
			}
			base = t.EffectiveBase().String()
		}
		role := "data"
		if col.IsIndex() {
			role = "index"
		}
		fmt.Printf("  %-28s %-20s %-18s %-18s %s\n",
			col.Name(), typeName, base, col.Access(), role)
	}
}
