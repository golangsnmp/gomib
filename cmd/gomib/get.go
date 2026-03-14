package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/golangsnmp/gomib/mib"
)

const (
	formatText = "text"
	formatJSON = "json"
)

const getUsage = `gomib get - Query OID or name lookups

Usage:
  gomib get [options] -m MODULE QUERY
  gomib get [options] MODULE... -- QUERY
  gomib get [options] MODULE... QUERY
  gomib get [options] --all QUERY

Query formats:
  Numeric OID:     1.3.6.1.2.1.2.2.1.1
  Name:            ifIndex
  Qualified:       IF-MIB::ifIndex

Options:
  -m, --module MODULE   Module to load (repeatable)
  --all                 Load all MIBs from search path
  -t, --tree            Show subtree instead of single node
  --max-depth N         Limit subtree depth (default: unlimited)
  --full                Show full descriptions in text output
  --format FMT          Output format: text, json (default: text)
  --strict              Resolver strictness: strict (tier-1 only)
  --permissive          Resolver strictness: permissive (tier-1/2/3)
  -h, --help            Show help

Examples:
  gomib get -m IF-MIB ifIndex
  gomib get -m IF-MIB 1.3.6.1.2.1.2.2.1.1
  gomib get IF-MIB SNMPv2-MIB -- sysDescr
  gomib get IF-MIB sysDescr
  gomib get -m IF-MIB -t ifTable
  gomib get --all ifIndex
`

type moduleList []string

func (m *moduleList) String() string { return fmt.Sprintf("%v", *m) }
func (m *moduleList) Set(value string) error {
	*m = append(*m, value)
	return nil
}

type depthFlag struct {
	value int
	set   bool
}

func (f *depthFlag) String() string {
	return fmt.Sprintf("%d", f.value)
}

func (f *depthFlag) Set(value string) error {
	n, err := strconv.Atoi(value)
	if err != nil {
		return err
	}
	if n < 0 {
		return fmt.Errorf("depth must be >= 0")
	}
	f.value = n
	f.set = true
	return nil
}

func (c *cli) cmdGet(args []string) int {
	fs := flag.NewFlagSet("get", flag.ContinueOnError)
	fs.Usage = func() { fmt.Fprint(os.Stderr, getUsage) }

	modules, loadAll := addModuleFlags(fs)
	tree := fs.Bool("t", false, "show subtree")
	fs.BoolVar(tree, "tree", false, "show subtree")
	var maxDepth depthFlag
	fs.Var(&maxDepth, "max-depth", "limit subtree depth")
	full := fs.Bool("full", false, "show full descriptions")
	format := fs.String("format", "text", "output format: text, json")
	strict, permissive := addStrictnessFlags(fs)
	help := addHelpFlag(fs)

	if err := fs.Parse(args); err != nil {
		return exitError
	}

	if c.checkHelp(help, getUsage) {
		return exitOK
	}

	if code, failed := validateStrictnessFlags(*strict, *permissive); failed {
		return code
	}

	query, code, failed := parseSingleTargetArgs(modules, *loadAll, fs.Args(), getUsage, "query")
	if failed {
		return code
	}

	if code, ok := requireModuleOrAll(*modules, *loadAll, getUsage); ok {
		return code
	}

	m, err := c.loadMibWithOpts(modulesToLoad(*modules, *loadAll), strictnessOpts(*strict, *permissive)...)
	if err != nil {
		printError("failed to load: %v", err)
		return exitError
	}

	node := m.Resolve(query)
	if node == nil {
		printError("not found: %s", query)
		return exitError
	}

	descLimit := 200
	if *full {
		descLimit = 0
	}

	switch *format {
	case formatJSON:
		return printNodeJSON(node, *tree || maxDepth.set, maxDepth)
	case formatText, "":
		if *tree || maxDepth.set {
			printNodeTree(node, maxDepth)
		} else {
			printNode(node, descLimit)
		}
		return exitOK
	default:
		printError("unknown format: %s", *format)
		return exitError
	}
}

func printNodeJSON(node *mib.Node, tree bool, maxDepth depthFlag) int {
	opts := JSONOptions{IncludeDescr: true}
	if tree {
		output := buildTreeJSON(node, opts)
		if maxDepth.set {
			trimTreeDepth(output, 0, maxDepth.value)
		}
		if err := writeJSON(os.Stdout, output, true); err != nil {
			printError("encoding JSON: %v", err)
			return exitError
		}
		return exitOK
	}

	// Single node: include object or notification detail
	type nodeJSON struct {
		Name         string            `json:"name,omitempty"`
		Module       string            `json:"module,omitempty"`
		OID          string            `json:"oid"`
		Kind         string            `json:"kind"`
		Object       *ObjectJSON       `json:"object,omitempty"`
		Notification *NotificationJSON `json:"notification,omitempty"`
	}

	out := nodeJSON{
		Name: node.Name(),
		OID:  node.OID().String(),
		Kind: node.Kind().String(),
	}
	if node.Module() != nil {
		out.Module = node.Module().Name()
	}
	if node.Object() != nil {
		obj := buildObjectJSON(node.Object(), opts)
		out.Object = &obj
	}
	if node.Notification() != nil {
		notif := buildNotificationJSON(node.Notification(), opts)
		out.Notification = &notif
	}

	if err := writeJSON(os.Stdout, out, true); err != nil {
		printError("encoding JSON: %v", err)
		return exitError
	}
	return exitOK
}

func trimTreeDepth(node *TreeNodeJSON, depth, maxDepth int) {
	if depth >= maxDepth {
		node.Children = nil
		return
	}
	for _, child := range node.Children {
		trimTreeDepth(child, depth+1, maxDepth)
	}
}

func printNode(node *mib.Node, descLimit int) {
	label := node.Name()
	if label == "" {
		label = fmt.Sprintf("(%d)", node.Arc())
	}

	moduleName := ""
	if node.Module() != nil {
		moduleName = node.Module().Name()
	}

	oid := node.OID().String()

	if moduleName != "" {
		fmt.Printf("%s  %s::%s  %s\n", label, moduleName, label, oid)
	} else {
		fmt.Printf("%s  %s\n", label, oid)
	}

	fmt.Printf("  kind:   %s\n", node.Kind().String())

	if node.Object() != nil {
		printObjectDetails(node.Object(), descLimit)
	}

	if node.Notification() != nil {
		printNotificationDetails(node.Notification(), descLimit)
	}
}

func printObjectDetails(obj *mib.Object, descLimit int) {
	if obj.Type() != nil {
		typ := obj.Type()
		typeName := typ.Name()
		if typeName == "" {
			typeName = typ.Base().String()
		}
		typeDesc := typeName
		if typ.Parent() != nil {
			typeDesc = fmt.Sprintf("%s (%s)", typeName, typ.Base().String())
		}
		ranges := obj.EffectiveRanges()
		if len(ranges) > 0 {
			vr := ranges[0]
			if vr.Min == vr.Max {
				typeDesc += fmt.Sprintf(" (%d)", vr.Min)
			} else {
				typeDesc += fmt.Sprintf(" (%d..%d)", vr.Min, vr.Max)
			}
		}
		sizes := obj.EffectiveSizes()
		if len(sizes) > 0 {
			sr := sizes[0]
			if sr.Min == sr.Max {
				typeDesc += fmt.Sprintf(" (SIZE(%d))", sr.Min)
			} else {
				typeDesc += fmt.Sprintf(" (SIZE(%d..%d))", sr.Min, sr.Max)
			}
		}
		fmt.Printf("  type:   %s\n", typeDesc)
	} else {
		enums := obj.EffectiveEnums()
		bits := obj.EffectiveBits()
		if len(bits) > 0 {
			fmt.Printf("  type:   BITS\n")
		} else if len(enums) > 0 {
			fmt.Printf("  type:   INTEGER (enum)\n")
		}
	}

	fmt.Printf("  access: %s\n", obj.Access().String())
	fmt.Printf("  status: %s\n", obj.Status().String())

	if len(obj.Index()) > 0 {
		fmt.Printf("  index:  [%s]\n", formatIndexList(obj.Index()))
	}

	if obj.Augments() != nil {
		fmt.Printf("  augments: %s\n", obj.Augments().Name())
	}

	if augBy := obj.AugmentedBy(); len(augBy) > 0 {
		names := make([]string, len(augBy))
		for i, a := range augBy {
			names[i] = a.Name()
		}
		fmt.Printf("  augmentedBy: %s\n", strings.Join(names, ", "))
	}

	if obj.Augments() != nil {
		effIdx := obj.EffectiveIndexes()
		if len(effIdx) > 0 {
			fmt.Printf("  effectiveIndex: [%s]\n", formatIndexList(effIdx))
		}
	}

	if dv := obj.DefaultValue(); !dv.IsZero() {
		fmt.Printf("  defval: %s\n", dv.String())
	}

	if obj.Units() != "" {
		fmt.Printf("  units:  %s\n", obj.Units())
	}

	if obj.Description() != "" {
		fmt.Printf("  descr:  %s\n", normalizeDescription(obj.Description(), descLimit))
	}

	if obj.Reference() != "" {
		fmt.Printf("  ref:    %s\n", normalizeDescription(obj.Reference(), descLimit))
	}

	enums := obj.EffectiveEnums()
	bits := obj.EffectiveBits()
	if len(enums) > 0 && len(bits) == 0 {
		fmt.Println("  values:")
		for _, v := range enums {
			fmt.Printf("    %s(%d)\n", v.Label, v.Value)
		}
	}

	if len(bits) > 0 {
		fmt.Println("  bits:")
		for _, b := range bits {
			fmt.Printf("    %s(%d)\n", b.Label, b.Value)
		}
	}

	// Column details: isIndex, row, table.
	if obj.Kind() == mib.KindColumn {
		fmt.Printf("  isIndex: %v\n", obj.IsIndex())
		if row := obj.Row(); row != nil {
			fmt.Printf("  row:    %s\n", row.Name())
		}
		if tbl := obj.Table(); tbl != nil {
			fmt.Printf("  table:  %s\n", tbl.Name())
		}
	}

	// Column table for tables and rows.
	if obj.Kind() == mib.KindTable || obj.Kind() == mib.KindRow {
		cols := obj.Columns()
		if len(cols) > 0 {
			fmt.Println("  columns:")
			printColumnTable(cols)
		}
	}
}

func printNotificationDetails(notif *mib.Notification, descLimit int) {
	fmt.Printf("  status: %s\n", notif.Status().String())

	if len(notif.Objects()) > 0 {
		fmt.Println("  objects:")
		for _, obj := range notif.Objects() {
			fmt.Printf("    %s\n", obj.Name())
		}
	}

	if notif.Description() != "" {
		fmt.Printf("  descr:  %s\n", normalizeDescription(notif.Description(), descLimit))
	}

	if notif.Reference() != "" {
		fmt.Printf("  ref:    %s\n", normalizeDescription(notif.Reference(), descLimit))
	}
}

func normalizeDescription(s string, maxLen int) string {
	if maxLen > 0 && len(s) > maxLen {
		s = s[:maxLen] + "..."
	}
	return strings.Join(strings.Fields(s), " ")
}

func printNodeTree(node *mib.Node, maxDepth depthFlag) {
	printNodeTreeRecursive(node, 0, maxDepth)
}

func printNodeTreeRecursive(node *mib.Node, depth int, maxDepth depthFlag) {
	if maxDepth.set && depth > maxDepth.value {
		return
	}

	indent := strings.Repeat("  ", depth)

	label := node.Name()
	if label == "" {
		label = fmt.Sprintf("(%d)", node.Arc())
	}

	oid := node.OID().String()
	kind := node.Kind().String()

	moduleName := ""
	if node.Module() != nil {
		moduleName = node.Module().Name()
	}

	extra := ""
	if node.Object() != nil {
		obj := node.Object()
		typeName := ""
		if obj.Type() != nil {
			typeName = obj.Type().Name()
			if typeName == "" {
				typeName = obj.Type().Base().String()
			}
		}
		extra = fmt.Sprintf("  %s  %s", typeName, obj.Access().String())
	}

	if moduleName != "" {
		fmt.Printf("%s%s  %s::%s  %s  %s%s\n", indent, label, moduleName, label, oid, kind, extra)
	} else {
		fmt.Printf("%s%s  %s  %s%s\n", indent, label, oid, kind, extra)
	}

	for _, child := range node.Children() {
		printNodeTreeRecursive(child, depth+1, maxDepth)
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

func printColumnTable(cols []*mib.Object) {
	fmt.Printf("    %-28s %-20s %-18s %-18s %s\n",
		"COLUMN", "TYPE", "BASE", "ACCESS", "ROLE")
	fmt.Printf("    %-28s %-20s %-18s %-18s %s\n",
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
		fmt.Printf("    %-28s %-20s %-18s %-18s %s\n",
			col.Name(), typeName, base, col.Access(), role)
	}
}
