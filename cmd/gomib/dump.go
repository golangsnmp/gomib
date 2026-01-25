package main

import (
	"flag"
	"fmt"
	"os"
	"slices"

	"github.com/golangsnmp/gomib"
)

const dumpUsage = `gomib dump - Output modules or subtrees as JSON

Usage:
  gomib dump [options] MODULE...

Options:
  -o, --oid OID          Dump subtree starting at OID
  --compact              Minified JSON (no indentation)
  --no-tree              Omit tree structure from output
  --no-descriptions      Omit description fields (smaller output)
  -h, --help             Show help

Examples:
  gomib dump IF-MIB
  gomib dump -o 1.3.6.1.2.1.2 IF-MIB
  gomib dump --compact IF-MIB
  gomib dump IF-MIB | jq '.objects'
`

func cmdDump(args []string) int {
	fs := flag.NewFlagSet("dump", flag.ContinueOnError)
	fs.Usage = func() { fmt.Fprint(os.Stderr, dumpUsage) }

	oidFilter := fs.String("o", "", "dump subtree starting at OID")
	fs.StringVar(oidFilter, "oid", "", "dump subtree starting at OID")
	compact := fs.Bool("compact", false, "minified JSON")
	noTree := fs.Bool("no-tree", false, "omit tree structure")
	noDescriptions := fs.Bool("no-descriptions", false, "omit descriptions")
	help := fs.Bool("h", false, "show help")
	fs.BoolVar(help, "help", false, "show help")

	if err := fs.Parse(args); err != nil {
		return 1
	}

	if *help || helpFlag {
		_, _ = fmt.Fprint(os.Stdout, dumpUsage)
		return 0
	}

	modules := fs.Args()
	if len(modules) == 0 {
		printError("no modules specified")
		fmt.Fprint(os.Stderr, dumpUsage)
		return 1
	}

	mib, err := loadMib(modules)
	if err != nil {
		printError("failed to load: %v", err)
		return 1
	}

	opts := JSONOptions{
		Compact:       *compact,
		IncludeTree:   !*noTree,
		IncludeDescr:  !*noDescriptions,
		IncludeDiags:  len(mib.Diagnostics()) > 0,
		RequestedMods: modules,
		OidFilter:     *oidFilter,
	}

	output := buildDumpOutput(mib, opts)

	json, err := marshalJSON(output, !*compact)
	if err != nil {
		printError("failed to marshal JSON: %v", err)
		return 1
	}

	fmt.Println(string(json))
	return 0
}

// JSONOptions controls what gets included in JSON output.
type JSONOptions struct {
	Compact       bool
	IncludeTree   bool
	IncludeDescr  bool
	IncludeDiags  bool
	RequestedMods []string
	OidFilter     string
}

// buildDumpOutput creates the JSON output structure.
func buildDumpOutput(m *gomib.Mib, opts JSONOptions) *DumpOutput {
	output := &DumpOutput{}

	// Modules
	for _, mod := range m.Modules() {
		if !shouldIncludeModule(mod.Name, opts.RequestedMods) {
			continue
		}
		output.Modules = append(output.Modules, buildModuleJSON(mod, opts))
	}

	// Types
	for _, typ := range m.Types() {
		if typ.Module != nil && !shouldIncludeModule(typ.Module.Name, opts.RequestedMods) {
			continue
		}
		output.Types = append(output.Types, buildTypeJSON(typ, opts))
	}

	// Objects
	for _, obj := range m.Objects() {
		if obj.Module != nil && !shouldIncludeModule(obj.Module.Name, opts.RequestedMods) {
			continue
		}
		output.Objects = append(output.Objects, buildObjectJSON(obj, opts))
	}

	// Notifications
	for _, notif := range m.Notifications() {
		if notif.Module != nil && !shouldIncludeModule(notif.Module.Name, opts.RequestedMods) {
			continue
		}
		output.Notifications = append(output.Notifications, buildNotificationJSON(notif, opts))
	}

	// Tree
	if opts.IncludeTree {
		if opts.OidFilter != "" {
			node := m.Node(opts.OidFilter)
			if node != nil {
				output.Tree = buildTreeJSON(node, opts)
			}
		} else {
			// Build tree from root's children
			root := m.Root()
			if root != nil {
				children := root.Children()
				if len(children) == 1 {
					output.Tree = buildTreeJSON(children[0], opts)
				} else if len(children) > 1 {
					// Wrap multiple roots
					var trees []*TreeNodeJSON
					for _, child := range children {
						trees = append(trees, buildTreeJSON(child, opts))
					}
					output.Tree = &TreeNodeJSON{
						Label:    "root",
						Children: trees,
					}
				}
			}
		}
	}

	// Diagnostics
	if opts.IncludeDiags {
		for _, d := range m.Diagnostics() {
			output.Diagnostics = append(output.Diagnostics, buildDiagnosticJSON(d))
		}
	}

	return output
}

func shouldIncludeModule(name string, requested []string) bool {
	if len(requested) == 0 {
		return true
	}
	return slices.Contains(requested, name)
}

func buildModuleJSON(mod *gomib.Module, opts JSONOptions) ModuleJSON {
	m := ModuleJSON{
		Name:         mod.Name,
		Language:     mod.Language.String(),
		Organization: mod.Organization,
		ContactInfo:  mod.ContactInfo,
	}
	if opts.IncludeDescr {
		m.Description = mod.Description
	}
	if mod.OID != nil {
		m.OID = mod.OID.String()
	}
	for _, rev := range mod.Revisions {
		r := RevisionJSON{Date: rev.Date}
		if opts.IncludeDescr {
			r.Description = rev.Description
		}
		m.Revisions = append(m.Revisions, r)
	}
	return m
}

func buildTypeJSON(typ *gomib.Type, opts JSONOptions) TypeJSON {
	t := TypeJSON{
		Name:   typ.Name,
		Base:   typ.Base.String(),
		Status: typ.Status.String(),
		Hint:   typ.Hint,
		IsTC:   typ.IsTC,
	}

	if typ.Module != nil {
		t.Module = typ.Module.Name
	}

	if typ.Parent != nil {
		t.Parent = typ.Parent.Name
	}

	if opts.IncludeDescr {
		t.Description = typ.Description
	}

	// Constraints
	for _, sr := range typ.Size {
		t.Size = append(t.Size, RangeJSON{Min: sr.Min, Max: sr.Max})
	}
	for _, vr := range typ.ValueRange {
		t.Range = append(t.Range, RangeJSON{Min: vr.Min, Max: vr.Max})
	}

	// Named values (enums/bits)
	for _, nv := range typ.NamedValues {
		if typ.Base == gomib.BaseBits {
			t.Bits = append(t.Bits, BitJSON{Label: nv.Label, Position: int(nv.Value)})
		} else {
			t.Enums = append(t.Enums, EnumJSON{Label: nv.Label, Value: nv.Value})
		}
	}

	return t
}

func buildObjectJSON(obj *gomib.Object, opts JSONOptions) ObjectJSON {
	o := ObjectJSON{
		Name:   obj.Name,
		OID:    obj.OID().String(),
		Kind:   obj.Kind().String(),
		Access: obj.Access.String(),
		Status: obj.Status.String(),
		Units:  obj.Units,
	}

	if obj.Module != nil {
		o.Module = obj.Module.Name
	}

	if obj.Type != nil {
		o.Type = obj.Type.Name
		if o.Type == "" {
			o.Type = obj.Type.Base.String()
		}
		o.BaseType = obj.Type.Base.String()
	}

	if opts.IncludeDescr {
		o.Description = obj.Description
	}

	// Index
	for _, idx := range obj.Index {
		idxJSON := IndexJSON{Implied: idx.Implied}
		if idx.Object != nil {
			idxJSON.Object = idx.Object.Name
		}
		o.Index = append(o.Index, idxJSON)
	}

	// Augments
	if obj.Augments != nil {
		o.Augments = obj.Augments.Name
	}

	// Named values
	for _, nv := range obj.NamedValues {
		if obj.Type != nil && obj.Type.Base == gomib.BaseBits {
			o.Bits = append(o.Bits, BitJSON{Label: nv.Label, Position: int(nv.Value)})
		} else {
			o.Enums = append(o.Enums, EnumJSON{Label: nv.Label, Value: nv.Value})
		}
	}

	return o
}

func buildNotificationJSON(notif *gomib.Notification, opts JSONOptions) NotificationJSON {
	n := NotificationJSON{
		Name:   notif.Name,
		OID:    notif.OID().String(),
		Status: notif.Status.String(),
	}

	if notif.Module != nil {
		n.Module = notif.Module.Name
	}

	if opts.IncludeDescr {
		n.Description = notif.Description
	}

	for _, obj := range notif.Objects {
		n.Objects = append(n.Objects, obj.Name)
	}

	return n
}

func buildTreeJSON(node *gomib.Node, opts JSONOptions) *TreeNodeJSON {
	t := &TreeNodeJSON{
		Arc:   node.Arc(),
		OID:   node.OID().String(),
		Label: node.Name,
		Kind:  node.Kind.String(),
	}

	if node.Module != nil {
		t.Module = node.Module.Name
	}

	for _, child := range node.Children() {
		t.Children = append(t.Children, buildTreeJSON(child, opts))
	}

	return t
}

func buildDiagnosticJSON(d gomib.Diagnostic) DiagnosticJSON {
	dj := DiagnosticJSON{
		Severity: d.Severity.String(),
		Message:  d.Message,
	}
	if d.Module != "" {
		dj.Module = d.Module
	}
	if d.Line > 0 {
		dj.Line = d.Line
	}
	return dj
}
