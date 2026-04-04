package main

import (
	"fmt"
	"strings"

	"github.com/golangsnmp/gomib/mib"
)

// printObjectIndexAugments prints Index, Augments, AugmentedBy, and EffectiveIndex.
func (c *cli) printObjectIndexAugments(obj *mib.Object) {
	if len(obj.Index()) > 0 {
		fmt.Fprintf(c.stdout, "Index:   [%s]\n", formatIndexList(obj.Index()))
	}
	if obj.Augments() != nil {
		fmt.Fprintf(c.stdout, "Augments: %s\n", obj.Augments().Name())
	}
	if augBy := obj.AugmentedBy(); len(augBy) > 0 {
		names := make([]string, len(augBy))
		for i, a := range augBy {
			names[i] = a.Name()
		}
		fmt.Fprintf(c.stdout, "AugmentedBy: %s\n", strings.Join(names, ", "))
	}
	if obj.Augments() != nil {
		if effIdx := obj.EffectiveIndexes(); len(effIdx) > 0 {
			fmt.Fprintf(c.stdout, "EffectiveIndex: [%s]\n", formatIndexList(effIdx))
		}
	}
}

// printObjectEnumsBits prints enum values or BITS for an object.
func (c *cli) printObjectEnumsBits(obj *mib.Object) {
	enums := obj.EffectiveEnums()
	bits := obj.EffectiveBits()
	if len(enums) > 0 && len(bits) == 0 {
		fmt.Fprintln(c.stdout, "Values:")
		for _, v := range enums {
			fmt.Fprintf(c.stdout, "  %s(%d)\n", v.Label, v.Value)
		}
	}
	if len(bits) > 0 {
		fmt.Fprintln(c.stdout, "Bits:")
		for _, b := range bits {
			fmt.Fprintf(c.stdout, "  %s(%d)\n", b.Label, b.Value)
		}
	}
}

// printObjectColumnContext prints IsIndex, Row, and Table for column objects.
func (c *cli) printObjectColumnContext(obj *mib.Object) {
	if obj.Kind() != mib.KindColumn {
		return
	}
	fmt.Fprintf(c.stdout, "IsIndex: %v\n", obj.IsIndex())
	if row := obj.Row(); row != nil {
		fmt.Fprintf(c.stdout, "Row:     %s\n", row.Name())
	}
	if tbl := obj.Table(); tbl != nil {
		fmt.Fprintf(c.stdout, "Table:   %s\n", tbl.Name())
	}
}

// printObjectColumns prints the column table for table or row objects.
func (c *cli) printObjectColumns(obj *mib.Object) {
	if obj.Kind() != mib.KindTable && obj.Kind() != mib.KindRow {
		return
	}
	cols := obj.Columns()
	if len(cols) > 0 {
		fmt.Fprintln(c.stdout, "Columns:")
		c.printColumnTable(cols)
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
func (c *cli) printColumnTable(cols []*mib.Object) {
	fmt.Fprintf(c.stdout, "  %-28s %-20s %-18s %-18s %s\n",
		"COLUMN", "TYPE", "BASE", "ACCESS", "ROLE")
	fmt.Fprintf(c.stdout, "  %-28s %-20s %-18s %-18s %s\n",
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
		fmt.Fprintf(c.stdout, "  %-28s %-20s %-18s %-18s %s\n",
			col.Name(), typeName, base, col.Access(), role)
	}
}
