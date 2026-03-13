package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
)

const dumpUsage = `gomib dump - Export resolved MIB data as canonical JSON

Usage:
  gomib dump [options] MODULE...
  gomib dump --all [options]

Options:
  -m, --module MODULE    Module to load (repeatable)
  --all                  Load all MIBs from search path
  -o, --oid OID          Filter to subtree starting at OID
  --compact              Minified JSON (no indentation)
  --no-descriptions      Omit description fields
  --strict               Resolver strictness: strict (tier-1 only)
  --permissive           Resolver strictness: permissive (tier-1/2/3)
  -h, --help             Show help

Examples:
  gomib dump IF-MIB
  gomib dump -o 1.3.6.1.2.1.2 IF-MIB
  gomib dump --all -p testdata/corpus/primary
  gomib dump --no-descriptions --compact IF-MIB | jq '.modules'
`

func (c *cli) cmdDump(args []string) int {
	fs := flag.NewFlagSet("dump", flag.ContinueOnError)
	fs.Usage = func() { fmt.Fprint(os.Stderr, dumpUsage) }

	modules, loadAll := addModuleFlags(fs)
	oidFilter := fs.String("o", "", "filter to subtree starting at OID")
	fs.StringVar(oidFilter, "oid", "", "filter to subtree starting at OID")
	compact := fs.Bool("compact", false, "minified JSON")
	noDescriptions := fs.Bool("no-descriptions", false, "omit descriptions")
	strict, permissive := addStrictnessFlags(fs)
	help := addHelpFlag(fs)

	if err := fs.Parse(args); err != nil {
		return exitError
	}

	if c.checkHelp(help, dumpUsage) {
		return exitOK
	}

	// Accept positional module names as well
	for _, arg := range fs.Args() {
		*modules = append(*modules, arg)
	}

	if code, failed := requireModuleOrAll(*modules, *loadAll, dumpUsage); failed {
		return code
	}

	mods := modulesToLoad(*modules, *loadAll)
	m, err := c.loadMibWithOpts(mods, strictnessOpts(*strict, *permissive)...)
	if err != nil && m == nil {
		printError("failed to load: %v", err)
		return exitError
	}

	strictnessName := "normal"
	if *strict {
		strictnessName = "strict"
	} else if *permissive {
		strictnessName = "permissive"
	}
	export := buildV1Export(m, strictnessName)

	if *oidFilter != "" {
		node := m.Resolve(*oidFilter)
		if node == nil {
			printError("OID not found: %s", *oidFilter)
			return exitError
		}
		filterV1ByOID(export, node.OID().String())
	}

	if *noDescriptions {
		stripV1Descriptions(export)
	}

	if err := writeJSON(os.Stdout, export, !*compact); err != nil {
		printError("failed to marshal JSON: %v", err)
		return exitError
	}
	return exitOK
}

func filterV1ByOID(export *V1Export, prefix string) {
	dotPrefix := prefix + "."
	matches := func(oid string) bool {
		return oid == prefix || strings.HasPrefix(oid, dotPrefix)
	}

	export.Nodes = filterSlice(export.Nodes, func(n V1Node) bool { return matches(n.OID) })
	export.Objects = filterSlice(export.Objects, func(o V1Object) bool { return matches(o.OID) })
	export.Notifications = filterSlice(export.Notifications, func(n V1Notification) bool { return matches(n.OID) })
	export.Groups = filterSlice(export.Groups, func(g V1Group) bool { return matches(g.OID) })
	export.Compliances = filterSlice(export.Compliances, func(c V1Compliance) bool { return matches(c.OID) })
	export.Capabilities = filterSlice(export.Capabilities, func(c V1Capability) bool { return matches(c.OID) })
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

func stripV1Descriptions(export *V1Export) {
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
		export.Notifications[i].Description = ""
	}
	for i := range export.Groups {
		export.Groups[i].Description = ""
	}
	for i := range export.Compliances {
		export.Compliances[i].Description = ""
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
		export.Capabilities[i].Description = ""
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
