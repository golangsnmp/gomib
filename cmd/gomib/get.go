package main

import (
	"flag"
	"fmt"
	"os"
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
  --full                Show full descriptions (no truncation)
  --format FMT          Output format: text, json (default: text)
  -h, --help            Show help

Examples:
  gomib get -m IF-MIB ifIndex
  gomib get -m IF-MIB 1.3.6.1.2.1.2.2.1.1
  gomib get IF-MIB SNMPv2-MIB -- sysDescr
  gomib get -m IF-MIB -t ifTable
  gomib get --all ifIndex
`

type moduleList []string

func (m *moduleList) String() string { return fmt.Sprintf("%v", *m) }
func (m *moduleList) Set(value string) error {
	*m = append(*m, value)
	return nil
}

func (c *cli) cmdGet(args []string) int {
	fs := flag.NewFlagSet("get", flag.ContinueOnError)
	fs.Usage = func() { fmt.Fprint(os.Stderr, getUsage) }

	var modules moduleList
	fs.Var(&modules, "m", "module to load")
	fs.Var(&modules, "module", "module to load")
	loadAll := fs.Bool("all", false, "load all MIBs from search path")
	tree := fs.Bool("t", false, "show subtree")
	fs.BoolVar(tree, "tree", false, "show subtree")
	maxDepth := fs.Int("max-depth", 0, "limit subtree depth")
	full := fs.Bool("full", false, "show full descriptions")
	format := fs.String("format", "text", "output format: text, json")
	help := fs.Bool("h", false, "show help")
	fs.BoolVar(help, "help", false, "show help")

	if err := fs.Parse(args); err != nil {
		return 1
	}

	if *help || c.helpFlag {
		_, _ = fmt.Fprint(os.Stdout, getUsage)
		return 0
	}

	remaining := fs.Args()

	var query string
	dashIdx := -1
	for i, arg := range remaining {
		if arg == "--" {
			dashIdx = i
			break
		}
	}

	if dashIdx >= 0 {
		modules = append(modules, remaining[:dashIdx]...)
		if dashIdx+1 < len(remaining) {
			query = remaining[dashIdx+1]
		}
	} else if len(remaining) > 0 {
		query = remaining[len(remaining)-1]
		if !*loadAll && len(modules) == 0 && len(remaining) > 1 {
			modules = remaining[:len(remaining)-1]
		}
	}

	if !*loadAll && len(modules) == 0 {
		printError("specify -m MODULE or --all")
		fmt.Fprint(os.Stderr, getUsage)
		return 1
	}

	if query == "" {
		printError("no query specified")
		fmt.Fprint(os.Stderr, getUsage)
		return 1
	}

	var loadModules []string
	if !*loadAll {
		loadModules = modules
	}
	m, err := c.loadMib(loadModules)
	if err != nil {
		printError("failed to load: %v", err)
		return exitError
	}

	node := resolveQuery(m, query)
	if node == nil {
		printError("not found: %s", query)
		return 1
	}

	descLimit := 200
	if *full {
		descLimit = 0
	}

	switch *format {
	case formatJSON:
		return printNodeJSON(node, *tree, *maxDepth)
	case formatText, "":
		if *tree {
			printNodeTree(node, *maxDepth)
		} else {
			printNode(node, descLimit)
		}
		return 0
	default:
		printError("unknown format: %s", *format)
		return 1
	}
}

func printNodeJSON(node *mib.Node, tree bool, maxDepth int) int {
	opts := JSONOptions{IncludeDescr: true}
	if tree {
		output := buildTreeJSON(node, opts)
		if maxDepth > 0 {
			trimTreeDepth(output, 0, maxDepth)
		}
		data, err := marshalJSON(output, true)
		if err != nil {
			printError("encoding JSON: %v", err)
			return exitError
		}
		fmt.Println(string(data))
		return 0
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

	data, err := marshalJSON(out, true)
	if err != nil {
		printError("encoding JSON: %v", err)
		return exitError
	}
	fmt.Println(string(data))
	return 0
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

// resolveQuery parses a user query string and returns the matching node.
// Supports: plain name, MODULE::name, numeric OID (with optional leading dot).
func resolveQuery(m *mib.Mib, query string) *mib.Node {
	// Qualified name: MODULE::name
	if modName, itemName, ok := strings.Cut(query, "::"); ok {
		mod := m.Module(modName)
		if mod == nil {
			return nil
		}
		return mod.Node(itemName)
	}

	// Numeric OID string
	q := query
	if len(q) > 0 && q[0] == '.' {
		q = q[1:]
	}
	if len(q) > 0 && q[0] >= '0' && q[0] <= '9' {
		oid, err := mib.ParseOID(q)
		if err != nil || len(oid) == 0 {
			return nil
		}
		return m.NodeByOID(oid)
	}

	// Plain name
	return m.Node(query)
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
		indexStrs := make([]string, 0, len(obj.Index()))
		for _, idx := range obj.Index() {
			name := "(unknown)"
			if idx.Object != nil {
				name = idx.Object.Name()
			}
			if idx.Implied {
				name = "IMPLIED " + name
			}
			indexStrs = append(indexStrs, name)
		}
		fmt.Printf("  index:  [%s]\n", strings.Join(indexStrs, ", "))
	}

	if obj.Augments() != nil {
		fmt.Printf("  augments: %s\n", obj.Augments().Name())
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
	s = strings.ReplaceAll(s, "\n", " ")
	return strings.Join(strings.Fields(s), " ")
}

func printNodeTree(node *mib.Node, maxDepth int) {
	printNodeTreeRecursive(node, 0, maxDepth)
}

func printNodeTreeRecursive(node *mib.Node, depth int, maxDepth int) {
	if maxDepth > 0 && depth > maxDepth {
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
