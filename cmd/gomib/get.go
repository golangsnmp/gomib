package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/golangsnmp/gomib"
)

const getUsage = `gomib get - Query OID or name lookups

Usage:
  gomib get [options] -m MODULE QUERY
  gomib get [options] MODULE... -- QUERY

Query formats:
  Numeric OID:     1.3.6.1.2.1.2.2.1.1
  Name:            ifIndex
  Qualified:       IF-MIB::ifIndex
  Partial OID:     .1.2.1.2 (relative lookup)

Options:
  -m, --module MODULE   Module to load (repeatable)
  -t, --tree            Show subtree instead of single node
  --max-depth N         Limit subtree depth (default: unlimited)
  -h, --help            Show help

Examples:
  gomib get -m IF-MIB ifIndex
  gomib get -m IF-MIB 1.3.6.1.2.1.2.2.1.1
  gomib get IF-MIB SNMPv2-MIB -- sysDescr
  gomib get -m IF-MIB -t ifTable
`

type moduleList []string

func (m *moduleList) String() string { return fmt.Sprintf("%v", *m) }
func (m *moduleList) Set(value string) error {
	*m = append(*m, value)
	return nil
}

func cmdGet(args []string) int {
	fs := flag.NewFlagSet("get", flag.ContinueOnError)
	fs.Usage = func() { fmt.Fprint(os.Stderr, getUsage) }

	var modules moduleList
	fs.Var(&modules, "m", "module to load")
	fs.Var(&modules, "module", "module to load")
	tree := fs.Bool("t", false, "show subtree")
	fs.BoolVar(tree, "tree", false, "show subtree")
	maxDepth := fs.Int("max-depth", 0, "limit subtree depth")
	help := fs.Bool("h", false, "show help")
	fs.BoolVar(help, "help", false, "show help")

	if err := fs.Parse(args); err != nil {
		return 1
	}

	if *help || helpFlag {
		_, _ = fmt.Fprint(os.Stdout, getUsage)
		return 0
	}

	remaining := fs.Args()

	// Parse MODULE... -- QUERY format
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
		if len(modules) == 0 && len(remaining) > 1 {
			// First args are modules, last is query
			modules = remaining[:len(remaining)-1]
		}
	}

	if len(modules) == 0 {
		printError("no modules specified")
		fmt.Fprint(os.Stderr, getUsage)
		return 1
	}

	if query == "" {
		printError("no query specified")
		fmt.Fprint(os.Stderr, getUsage)
		return 1
	}

	mib, err := loadMib(modules)
	if err != nil {
		printError("failed to load: %v", err)
		return 2
	}

	// Find the node
	node := mib.FindNode(query)
	if node == nil {
		printError("not found: %s", query)
		return 1
	}

	if *tree {
		printNodeTree(node, *maxDepth)
	} else {
		printNode(node)
	}

	return 0
}

// printNode prints a single node's details.
func printNode(node *gomib.Node) {
	// Header: name  MODULE::name  oid
	label := node.Name
	if label == "" {
		label = fmt.Sprintf("(%d)", node.Arc())
	}

	moduleName := ""
	if node.Module != nil {
		moduleName = node.Module.Name
	}

	oid := node.OID().String()

	if moduleName != "" {
		fmt.Printf("%s  %s::%s  %s\n", label, moduleName, label, oid)
	} else {
		fmt.Printf("%s  %s\n", label, oid)
	}

	fmt.Printf("  kind:   %s\n", node.Kind.String())

	// Print object details if available
	if node.Object != nil {
		printObjectDetails(node.Object)
	}

	// Print notification details if available
	if node.Notif != nil {
		printNotificationDetails(node.Notif)
	}
}

// printObjectDetails prints object-specific information.
func printObjectDetails(obj *gomib.Object) {
	// Type
	if obj.Type != nil {
		typeName := obj.Type.Name
		if typeName == "" {
			typeName = obj.Type.Base.String()
		}
		typeDesc := typeName
		if obj.Type.Parent != nil {
			typeDesc = fmt.Sprintf("%s (%s)", typeName, obj.Type.Base.String())
		}
		// Add constraints
		if len(obj.ValueRange) > 0 {
			vr := obj.ValueRange[0]
			if vr.Min == vr.Max {
				typeDesc += fmt.Sprintf(" (%d)", vr.Min)
			} else {
				typeDesc += fmt.Sprintf(" (%d..%d)", vr.Min, vr.Max)
			}
		}
		if len(obj.Size) > 0 {
			sr := obj.Size[0]
			if sr.Min == sr.Max {
				typeDesc += fmt.Sprintf(" (SIZE(%d))", sr.Min)
			} else {
				typeDesc += fmt.Sprintf(" (SIZE(%d..%d))", sr.Min, sr.Max)
			}
		}
		fmt.Printf("  type:   %s\n", typeDesc)
	} else if len(obj.NamedValues) > 0 {
		if obj.Type != nil && obj.Type.Base == gomib.BaseBits {
			fmt.Printf("  type:   BITS\n")
		} else {
			fmt.Printf("  type:   INTEGER (enum)\n")
		}
	}

	fmt.Printf("  access: %s\n", obj.Access.String())
	fmt.Printf("  status: %s\n", obj.Status.String())

	// Index
	if len(obj.Index) > 0 {
		indexStrs := make([]string, 0, len(obj.Index))
		for _, idx := range obj.Index {
			name := "(unknown)"
			if idx.Object != nil {
				name = idx.Object.Name
			}
			if idx.Implied {
				name = "IMPLIED " + name
			}
			indexStrs = append(indexStrs, name)
		}
		fmt.Printf("  index:  [%s]\n", strings.Join(indexStrs, ", "))
	}

	// Augments
	if obj.Augments != nil {
		fmt.Printf("  augments: %s\n", obj.Augments.Name)
	}

	// Units
	if obj.Units != "" {
		fmt.Printf("  units:  %s\n", obj.Units)
	}

	// Description (truncated)
	if obj.Description != "" {
		fmt.Printf("  descr:  %s\n", normalizeDescription(obj.Description, 200))
	}

	// Enum values
	if len(obj.NamedValues) > 0 && (obj.Type == nil || obj.Type.Base != gomib.BaseBits) {
		fmt.Println("  values:")
		for _, v := range obj.NamedValues {
			fmt.Printf("    %s(%d)\n", v.Label, v.Value)
		}
	}

	// BITS
	if len(obj.NamedValues) > 0 && obj.Type != nil && obj.Type.Base == gomib.BaseBits {
		fmt.Println("  bits:")
		for _, b := range obj.NamedValues {
			fmt.Printf("    %s(%d)\n", b.Label, b.Value)
		}
	}
}

// printNotificationDetails prints notification-specific information.
func printNotificationDetails(notif *gomib.Notification) {
	fmt.Printf("  status: %s\n", notif.Status.String())

	if len(notif.Objects) > 0 {
		fmt.Println("  objects:")
		for _, obj := range notif.Objects {
			fmt.Printf("    %s\n", obj.Name)
		}
	}

	if notif.Description != "" {
		fmt.Printf("  descr:  %s\n", normalizeDescription(notif.Description, 200))
	}
}

// normalizeDescription truncates and normalizes a description for display.
func normalizeDescription(s string, maxLen int) string {
	if len(s) > maxLen {
		s = s[:maxLen] + "..."
	}
	s = strings.ReplaceAll(s, "\n", " ")
	return strings.Join(strings.Fields(s), " ")
}

// printNodeTree prints a subtree.
func printNodeTree(node *gomib.Node, maxDepth int) {
	printNodeTreeRecursive(node, 0, maxDepth)
}

func printNodeTreeRecursive(node *gomib.Node, depth int, maxDepth int) {
	if maxDepth > 0 && depth > maxDepth {
		return
	}

	indent := strings.Repeat("  ", depth)

	label := node.Name
	if label == "" {
		label = fmt.Sprintf("(%d)", node.Arc())
	}

	oid := node.OID().String()
	kind := node.Kind.String()

	// Module name
	moduleName := ""
	if node.Module != nil {
		moduleName = node.Module.Name
	}

	// For objects, show type and access
	extra := ""
	if node.Object != nil {
		obj := node.Object
		typeName := ""
		if obj.Type != nil {
			typeName = obj.Type.Name
			if typeName == "" {
				typeName = obj.Type.Base.String()
			}
		}
		extra = fmt.Sprintf("  %s  %s", typeName, obj.Access.String())
	}

	if moduleName != "" {
		fmt.Printf("%s%s  %s::%s  %s  %s%s\n", indent, label, moduleName, label, oid, kind, extra)
	} else {
		fmt.Printf("%s%s  %s  %s%s\n", indent, label, oid, kind, extra)
	}

	// Print children
	for _, child := range node.Children() {
		printNodeTreeRecursive(child, depth+1, maxDepth)
	}
}
