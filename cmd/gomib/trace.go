package main

import (
	"flag"
	"fmt"
	"slices"
	"strings"

	"github.com/golangsnmp/gomib/mib"
)

const traceUsage = `gomib trace - Trace symbol resolution for debugging

Usage:
  gomib trace [options] SYMBOL

Traces how a symbol is resolved across loaded modules. Useful for debugging
resolution issues like missing INDEX references or duplicate definitions.

Options:
  -m, --module MODULE   Module to load (repeatable, default: all)
  --strict              Resolver strictness: strict (tier-1 only)
  --permissive          Resolver strictness: permissive (tier-1/2/3)
  -h, --help            Show help

Output shows:
  - All modules that define the symbol
  - OID tree location and kind
  - Whether Object/Type is attached
  - For rows: INDEX resolution status
  - Unresolved references related to the symbol

Examples:
  gomib trace -m IF-MIB ifIndex
  gomib trace -m IF-MIB ifEntry
  gomib trace ifIndex
  gomib trace -p testdata/corpus/primary ifEntry
`

func (c *cli) cmdTrace(args []string) int {
	fs := flag.NewFlagSet("trace", flag.ContinueOnError)
	fs.Usage = func() { fmt.Fprint(c.stderr, traceUsage) }

	modules := addModuleFlag(fs)
	strict, permissive := addStrictnessFlags(fs)
	help := addHelpFlag(fs)

	if err := fs.Parse(args); err != nil {
		return exitError
	}

	if c.checkHelp(help, traceUsage) {
		return exitOK
	}

	if code, failed := c.validateStrictnessFlags(*strict, *permissive); failed {
		return code
	}

	positional := fs.Args()
	if len(positional) == 0 {
		c.printError("no symbol specified")
		fmt.Fprint(c.stderr, traceUsage)
		return exitError
	}
	if len(positional) > 1 {
		c.printError("too many arguments (use -m for module selection)")
		fmt.Fprint(c.stderr, traceUsage)
		return exitError
	}
	symbol := positional[0]

	var mods []string
	if len(*modules) > 0 {
		mods = *modules
	}

	var loadMode string
	if len(mods) == 0 {
		loadMode = "Load - all modules from search path"
	} else {
		loadMode = fmt.Sprintf("Load WithModules(%v)", mods)
	}

	m, err := c.loadMibWithOpts(mods, strictnessOpts(*strict, *permissive)...)
	if err != nil && m == nil {
		c.printError("failed to load: %v", err)
		return exitError
	}

	fmt.Fprintf(c.stdout, "Load mode: %s\n", loadMode)
	fmt.Fprintf(c.stdout, "Loaded: %d modules, %d objects, %d types\n\n",
		len(m.Modules()), len(m.Objects()), len(m.Types()))

	c.traceSymbol(m, symbol)

	return exitOK
}

func (c *cli) traceSymbol(m *mib.Mib, symbol string) {
	fmt.Fprintf(c.stdout, "=== Tracing symbol: %s ===\n\n", symbol)

	var definingModules []string
	for _, mod := range m.Modules() {
		if mod.Object(symbol) != nil || mod.Type(symbol) != nil || mod.Node(symbol) != nil {
			definingModules = append(definingModules, mod.Name())
		}
	}

	node := m.Node(symbol)
	obj := m.Object(symbol)
	typ := m.Type(symbol)

	fmt.Fprintln(c.stdout, "DEFINITIONS:")
	if len(definingModules) == 0 {
		fmt.Fprintln(c.stdout, "  (none found)")
	} else {
		slices.Sort(definingModules)
		for _, modName := range definingModules {
			mod := m.Module(modName)
			var kinds []string
			if mod.Object(symbol) != nil {
				kinds = append(kinds, "Object")
			}
			if mod.Type(symbol) != nil {
				kinds = append(kinds, "Type")
			}
			if mod.Node(symbol) != nil && mod.Object(symbol) == nil {
				kinds = append(kinds, "Node")
			}
			fmt.Fprintf(c.stdout, "  %s: %s\n", modName, strings.Join(kinds, ", "))
		}
	}
	fmt.Fprintln(c.stdout)

	fmt.Fprintln(c.stdout, "GLOBAL LOOKUPS (unqualified):")
	if node != nil {
		modName := "(no module)"
		if node.Module() != nil {
			modName = node.Module().Name()
		}
		fmt.Fprintf(c.stdout, "  Node:        %s::%s  OID=%s  Kind=%s\n",
			modName, node.Name(), node.OID(), node.Kind())
		fmt.Fprintf(c.stdout, "               Object attached: %v\n", node.Object() != nil)
		fmt.Fprintf(c.stdout, "               Notification attached: %v\n", node.Notification() != nil)
	} else {
		fmt.Fprintln(c.stdout, "  Node:        (not found)")
	}

	if obj != nil {
		modName := "(no module)"
		if obj.Module() != nil {
			modName = obj.Module().Name()
		}
		fmt.Fprintf(c.stdout, "  Object:      %s::%s  OID=%s  Kind=%s\n",
			modName, obj.Name(), obj.OID(), obj.Kind())
	} else {
		fmt.Fprintln(c.stdout, "  Object:      (not found)")
	}

	if typ != nil {
		modName := "(no module)"
		if typ.Module() != nil {
			modName = typ.Module().Name()
		}
		fmt.Fprintf(c.stdout, "  Type:        %s::%s  Base=%s\n",
			modName, typ.Name(), typ.Base())
	} else {
		fmt.Fprintln(c.stdout, "  Type:        (not found)")
	}
	fmt.Fprintln(c.stdout)

	if len(definingModules) > 1 {
		fmt.Fprintln(c.stdout, "PER-MODULE LOOKUPS:")
		for _, modName := range definingModules {
			mod := m.Module(modName)
			modObj := mod.Object(symbol)
			if modObj != nil {
				fmt.Fprintf(c.stdout, "  %s::%s:\n", modName, symbol)
				fmt.Fprintf(c.stdout, "    OID=%s  Kind=%s\n", modObj.OID(), modObj.Kind())
				if modObj.Kind() == mib.KindRow {
					fmt.Fprintf(c.stdout, "    Index count: %d\n", len(modObj.Index()))
					for i, idx := range modObj.Index() {
						name := "(nil!)"
						if idx.Object != nil {
							name = idx.Object.Name()
						}
						enc := ""
						if idx.Encoding != mib.IndexEncodingUnknown {
							enc = fmt.Sprintf("  encoding=%s", idx.Encoding)
						}
						fmt.Fprintf(c.stdout, "      [%d] %s%s\n", i, name, enc)
					}
				}
			}
		}
		fmt.Fprintln(c.stdout)
	}

	if obj != nil && obj.Kind() == mib.KindRow {
		fmt.Fprintln(c.stdout, "INDEX RESOLUTION (row object):")
		directIndex := obj.Index()
		effectiveIndex := obj.EffectiveIndexes()

		if len(directIndex) == 0 && obj.Augments() == nil {
			fmt.Fprintln(c.stdout, "  WARNING: No INDEX clause resolved!")
			fmt.Fprintln(c.stdout, "  This row has no index entries, which may indicate a resolution failure.")
		} else if len(directIndex) == 0 && obj.Augments() != nil {
			fmt.Fprintf(c.stdout, "  AUGMENTS: %s\n", obj.Augments().Name())
		}

		if len(directIndex) > 0 {
			fmt.Fprintf(c.stdout, "  Direct INDEX (%d entries):\n", len(directIndex))
			c.printIndexDetails(directIndex)
		}

		if len(effectiveIndex) > 0 && len(effectiveIndex) != len(directIndex) {
			fmt.Fprintf(c.stdout, "  Effective INDEX (%d entries, via AUGMENTS chain):\n", len(effectiveIndex))
			c.printIndexDetails(effectiveIndex)
		}
		fmt.Fprintln(c.stdout)
	}

	if obj != nil && obj.Kind() == mib.KindTable {
		fmt.Fprintln(c.stdout, "TABLE STRUCTURE:")
		if obj.Entry() != nil {
			entry := obj.Entry()
			fmt.Fprintf(c.stdout, "  Row entry: %s\n", entry.Name())
			fmt.Fprintf(c.stdout, "  Row INDEX count: %d\n", len(entry.Index()))
			if len(entry.Index()) == 0 && entry.Augments() == nil {
				fmt.Fprintln(c.stdout, "  WARNING: Row has no INDEX resolved!")
			}

			effIdx := entry.EffectiveIndexes()
			if len(effIdx) > 0 {
				fmt.Fprintf(c.stdout, "  Effective indexes (%d):\n", len(effIdx))
				for i, idx := range effIdx {
					name := "(nil!)"
					if idx.Object != nil {
						name = idx.Object.Name()
					}
					enc := ""
					if idx.Encoding != mib.IndexEncodingUnknown {
						enc = fmt.Sprintf("  encoding=%s", idx.Encoding)
					}
					implied := ""
					if idx.Implied {
						implied = " (IMPLIED)"
					}
					fmt.Fprintf(c.stdout, "    [%d] %s%s%s\n", i, name, implied, enc)
				}
			}

			if augBy := entry.AugmentedBy(); len(augBy) > 0 {
				names := make([]string, len(augBy))
				for i, a := range augBy {
					names[i] = a.Name()
				}
				fmt.Fprintf(c.stdout, "  Augmented by: %s\n", strings.Join(names, ", "))
			}

			cols := entry.Columns()
			if len(cols) > 0 {
				fmt.Fprintln(c.stdout, "  Columns:")
				c.printColumnTable(cols)
			}
		} else {
			fmt.Fprintln(c.stdout, "  WARNING: No row entry found!")
		}
		fmt.Fprintln(c.stdout)
	}

	unresolved := m.Unresolved()
	var related []mib.UnresolvedRef
	for _, u := range unresolved {
		if u.Symbol == symbol || strings.Contains(u.Symbol, symbol) {
			related = append(related, u)
		}
	}

	if len(related) > 0 {
		fmt.Fprintln(c.stdout, "RELATED UNRESOLVED REFERENCES:")
		for _, u := range related {
			fmt.Fprintf(c.stdout, "  [%s] %s in module %s\n", u.Kind, u.Symbol, u.Module)
		}
		fmt.Fprintln(c.stdout)
	}

	if len(definingModules) > 1 {
		fmt.Fprintln(c.stdout, "WARNING: Multiple modules define this symbol!")
		fmt.Fprintln(c.stdout, "  This can cause resolution ambiguity depending on load order.")
		fmt.Fprintln(c.stdout, "  Modules:", strings.Join(definingModules, ", "))

		var nodes []*mib.Node
		var nodeInfo []string
		for _, modName := range definingModules {
			mod := m.Module(modName)
			if modObj := mod.Object(symbol); modObj != nil {
				nodes = append(nodes, modObj.Node())
				nodeInfo = append(nodeInfo, fmt.Sprintf("%s -> node.Module=%v",
					modName, modObj.Node().Module().Name()))
			}
		}
		if len(nodes) > 1 {
			same := nodes[0].OID().String() == nodes[1].OID().String()
			fmt.Fprintf(c.stdout, "  Objects share same OID node: %v\n", same)
			for _, info := range nodeInfo {
				fmt.Fprintf(c.stdout, "    %s\n", info)
			}
		}
		fmt.Fprintln(c.stdout)
	}

	if c.verbose > 0 && len(unresolved) > 0 {
		fmt.Fprintf(c.stdout, "ALL UNRESOLVED REFERENCES (%d total):\n", len(unresolved))
		for _, u := range unresolved {
			fmt.Fprintf(c.stdout, "  [%s] %s in module %s\n", u.Kind, u.Symbol, u.Module)
		}
	}
}

func (c *cli) printIndexDetails(indexes []mib.IndexEntry) {
	for i, idx := range indexes {
		name := "(nil object!)"
		oid := ""
		if idx.Object != nil {
			name = idx.Object.Name()
			oid = idx.Object.OID().String()
		}
		implied := ""
		if idx.Implied {
			implied = " (IMPLIED)"
		}
		enc := ""
		if idx.Encoding != mib.IndexEncodingUnknown {
			enc = fmt.Sprintf("  encoding=%s", idx.Encoding)
		}
		fmt.Fprintf(c.stdout, "    [%d] %s  OID=%s%s%s\n", i, name, oid, implied, enc)
	}
}
