package main

import (
	"errors"
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
  gomib get [options] QUERY

Query formats:
  Numeric OID:     1.3.6.1.2.1.2.2.1.1
  Name:            ifIndex
  Qualified:       IF-MIB::ifIndex

Options:
  -m, --module MODULE   Module to load (repeatable, default: all)
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
  gomib get ifIndex
  gomib get -m IF-MIB -t ifTable
  gomib get IF-MIB::ifIndex
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
	return strconv.Itoa(f.value)
}

func (f *depthFlag) Set(value string) error {
	n, err := strconv.Atoi(value)
	if err != nil {
		return err
	}
	if n < 0 {
		return errors.New("depth must be >= 0")
	}
	f.value = n
	f.set = true
	return nil
}

func (c *cli) cmdGet(args []string) int {
	fs := flag.NewFlagSet("get", flag.ContinueOnError)
	fs.Usage = func() { fmt.Fprint(os.Stderr, getUsage) }

	modules := addModuleFlag(fs)
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

	positional := fs.Args()
	if len(positional) == 0 {
		printError("no query specified")
		fmt.Fprint(os.Stderr, getUsage)
		return exitError
	}
	if len(positional) > 1 {
		printError("too many arguments (use -m for module selection)")
		fmt.Fprint(os.Stderr, getUsage)
		return exitError
	}
	query := positional[0]

	// Default to permissive+silent for query commands
	opts := strictnessOpts(*strict, *permissive)
	if len(opts) == 0 {
		opts = permissiveSilentOpts()
	}

	var mods []string
	if len(*modules) > 0 {
		mods = *modules
	}

	m, err := c.loadMibWithOpts(mods, opts...)
	if err != nil && m == nil {
		printError("failed to load: %v", err)
		return exitError
	}

	node := m.Resolve(query)
	if node == nil {
		printError("not found: %s", query)
		return exitIssue
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

	fmt.Printf("Name:    %s\n", label)
	if moduleName != "" {
		fmt.Printf("Module:  %s\n", moduleName)
	}
	fmt.Printf("OID:     %s\n", node.OID().String())
	fmt.Printf("Kind:    %s\n", node.Kind().String())

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
		fmt.Printf("Type:    %s\n", typeDesc)
	} else {
		enums := obj.EffectiveEnums()
		bits := obj.EffectiveBits()
		if len(bits) > 0 {
			fmt.Printf("Type:    BITS\n")
		} else if len(enums) > 0 {
			fmt.Printf("Type:    INTEGER (enum)\n")
		}
	}

	fmt.Printf("Access:  %s\n", obj.Access().String())
	fmt.Printf("Status:  %s\n", obj.Status().String())

	printObjectIndexAugments(obj)

	if dv := obj.DefaultValue(); !dv.IsZero() {
		fmt.Printf("DefVal:  %s\n", dv.String())
	}

	if obj.Units() != "" {
		fmt.Printf("Units:   %s\n", obj.Units())
	}

	if obj.Description() != "" {
		fmt.Printf("Descr:   %s\n", normalizeDescription(obj.Description(), descLimit))
	}

	if obj.Reference() != "" {
		fmt.Printf("Ref:     %s\n", normalizeDescription(obj.Reference(), descLimit))
	}

	printObjectEnumsBits(obj)

	// Column details: isIndex, row, table.
	printObjectColumnContext(obj)

	// Column table for tables and rows.
	printObjectColumns(obj)
}

func printNotificationDetails(notif *mib.Notification, descLimit int) {
	fmt.Printf("Status:  %s\n", notif.Status().String())

	if len(notif.Objects()) > 0 {
		fmt.Println("Objects:")
		for _, obj := range notif.Objects() {
			fmt.Printf("  %s\n", obj.Name())
		}
	}

	if notif.Description() != "" {
		fmt.Printf("Descr:   %s\n", normalizeDescription(notif.Description(), descLimit))
	}

	if notif.Reference() != "" {
		fmt.Printf("Ref:     %s\n", normalizeDescription(notif.Reference(), descLimit))
	}
}

func normalizeDescription(s string, maxLen int) string {
	s = strings.Join(strings.Fields(s), " ")
	if maxLen > 0 && len(s) > maxLen {
		s = s[:maxLen] + "..."
	}
	return s
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
