package main

import (
	"flag"
	"fmt"
	"strings"
)

const dumpUsage = `gomib dump - Export resolved MIB data as canonical JSON

Usage:
  gomib dump [options] [MODULE...]

Dumps all available modules when no MODULE is specified.

Options:
  -o, --oid OID          Filter to subtree starting at OID
  --compact              Minified JSON (no indentation)
  --no-descriptions      Omit description fields
  --report LEVEL         Diagnostic reporting: silent|quiet|default|verbose
  --strict               Resolver strictness: strict (tier-1 only)
  --permissive           Resolver strictness: permissive (tier-1/2/3)
  -h, --help             Show help

Examples:
  gomib dump IF-MIB
  gomib dump -o 1.3.6.1.2.1.2 IF-MIB
  gomib dump -p testdata/corpus/primary
  gomib dump --no-descriptions --compact IF-MIB | jq '.modules'
`

func (c *cli) cmdDump(args []string) int {
	fs := flag.NewFlagSet("dump", flag.ContinueOnError)
	fs.Usage = func() { fmt.Fprint(c.stderr, dumpUsage) }

	oidFilter := fs.String("o", "", "filter to subtree starting at OID")
	fs.StringVar(oidFilter, "oid", "", "filter to subtree starting at OID")
	compact := fs.Bool("compact", false, "minified JSON")
	noDescriptions := fs.Bool("no-descriptions", false, "omit descriptions")
	report := fs.String("report", "default", "diagnostic reporting: silent|quiet|default|verbose")
	strict, permissive := addStrictnessFlags(fs)
	help := addHelpFlag(fs)

	if err := fs.Parse(args); err != nil {
		return exitError
	}

	if c.checkHelp(help, dumpUsage) {
		return exitOK
	}

	if code, failed := c.validateStrictnessFlags(*strict, *permissive); failed {
		return code
	}

	modules := fs.Args()
	if len(modules) == 0 {
		modules = nil
	}

	opts := strictnessOpts(*strict, *permissive)
	reportOption, ok := c.reportOpt(*report)
	if !ok {
		return exitError
	}
	opts = append(opts, reportOption)

	m, err := c.loadMibWithOpts(modules, opts...)
	if err != nil && m == nil {
		c.printError("failed to load: %v", err)
		return exitError
	}

	// Print diagnostics to stderr
	diags := m.Diagnostics()
	if len(diags) > 0 {
		fmt.Fprintln(c.stderr)
		fmt.Fprintln(c.stderr, "Diagnostics:")
		for i := range diags {
			c.printDiagnostic(&diags[i])
		}
	}

	strictnessName := "normal"
	if *strict {
		strictnessName = "strict"
	} else if *permissive {
		strictnessName = "permissive"
	}
	export := buildExportPayload(m, strictnessName)

	if *oidFilter != "" {
		node := m.Resolve(*oidFilter)
		if node == nil {
			c.printError("OID not found: %s", *oidFilter)
			return exitIssue
		}
		filterByOID(export, node.OID().String())
	}

	if *noDescriptions {
		stripDescriptions(export)
	}

	if err := writeJSON(c.stdout, export, !*compact); err != nil {
		c.printError("failed to marshal JSON: %v", err)
		return exitIssue
	}
	return exitOK
}

func filterByOID(export *ExportPayload, prefix string) {
	dotPrefix := prefix + "."
	matches := func(oid string) bool {
		return oid == prefix || strings.HasPrefix(oid, dotPrefix)
	}

	export.Nodes = filterSlice(export.Nodes, func(n ExportNode) bool { return matches(n.OID) })
	export.Objects = filterSlice(export.Objects, func(o ExportObject) bool { return matches(o.OID) })
	export.Notifications = filterSlice(export.Notifications, func(n ExportNotification) bool { return matches(n.OID) })
	export.Groups = filterSlice(export.Groups, func(g ExportGroup) bool { return matches(g.OID) })
	export.Compliances = filterSlice(export.Compliances, func(c ExportCompliance) bool { return matches(c.OID) })
	export.Capabilities = filterSlice(export.Capabilities, func(c ExportCapability) bool { return matches(c.OID) })
}

func filterSlice[T any](s []T, keep func(T) bool) []T {
	result := make([]T, 0)
	for _, v := range s {
		if keep(v) {
			result = append(result, v)
		}
	}
	return result
}

func stripDescriptions(export *ExportPayload) {
	for i := range export.Modules {
		export.Modules[i].Description = nil
		for j := range export.Modules[i].Revisions {
			export.Modules[i].Revisions[j].Description = nil
		}
	}
	for i := range export.Types {
		export.Types[i].Description = nil
	}
	for i := range export.Nodes {
		export.Nodes[i].Description = nil
	}
	for i := range export.Objects {
		export.Objects[i].Description = nil
	}
	for i := range export.Notifications {
		export.Notifications[i].Description = nil
	}
	for i := range export.Groups {
		export.Groups[i].Description = nil
	}
	for i := range export.Compliances {
		export.Compliances[i].Description = nil
		for j := range export.Compliances[i].Modules {
			for k := range export.Compliances[i].Modules[j].Groups {
				export.Compliances[i].Modules[j].Groups[k].Description = nil
			}
			for k := range export.Compliances[i].Modules[j].Objects {
				export.Compliances[i].Modules[j].Objects[k].Description = nil
			}
		}
	}
	for i := range export.Capabilities {
		export.Capabilities[i].Description = nil
		for j := range export.Capabilities[i].Supports {
			for k := range export.Capabilities[i].Supports[j].ObjectVariations {
				export.Capabilities[i].Supports[j].ObjectVariations[k].Description = nil
			}
			for k := range export.Capabilities[i].Supports[j].NotificationVariations {
				export.Capabilities[i].Supports[j].NotificationVariations[k].Description = nil
			}
		}
	}
}
