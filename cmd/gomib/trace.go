package main

import (
	"flag"
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/golangsnmp/gomib/mib"
)

const traceUsage = `gomib trace - Trace symbol resolution for debugging

Usage:
  gomib trace [options] SYMBOL
  gomib trace [options] -m MODULE SYMBOL
  gomib trace [options] MODULE... -- SYMBOL
  gomib trace [options] MODULE... SYMBOL

Traces how a symbol is resolved across loaded modules. Useful for debugging
resolution issues like missing INDEX references or duplicate definitions.

Options:
  -m, --module MODULE   Module to load (repeatable, uses WithModules)
  --all                 Load all MIBs from search path (uses Load)
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
  gomib trace IF-MIB ifEntry
  gomib trace --all ifIndex
  gomib trace --all -p testdata/corpus/primary ifEntry
`

func (c *cli) cmdTrace(args []string) int {
	fs := flag.NewFlagSet("trace", flag.ContinueOnError)
	fs.Usage = func() { fmt.Fprint(os.Stderr, traceUsage) }

	modules, loadAll := addModuleFlags(fs)
	strict, permissive := addStrictnessFlags(fs)
	help := addHelpFlag(fs)

	if err := fs.Parse(args); err != nil {
		return exitError
	}

	if c.checkHelp(help, traceUsage) {
		return exitOK
	}

	if code, failed := validateStrictnessFlags(*strict, *permissive); failed {
		return code
	}

	symbol, code, failed := parseSingleTargetArgs(modules, *loadAll, fs.Args(), traceUsage, "symbol")
	if failed {
		return code
	}

	if code, ok := requireModuleOrAll(*modules, *loadAll, traceUsage); ok {
		return code
	}

	var m *mib.Mib
	var err error
	var loadMode string

	if *loadAll {
		loadMode = "Load - all modules from search path"
		m, err = c.loadMibWithOpts(nil, strictnessOpts(*strict, *permissive)...)
	} else {
		loadMode = fmt.Sprintf("Load WithModules(%v)", *modules)
		m, err = c.loadMibWithOpts(*modules, strictnessOpts(*strict, *permissive)...)
	}

	if err != nil {
		printError("failed to load: %v", err)
		return exitError
	}

	fmt.Printf("Load mode: %s\n", loadMode)
	fmt.Printf("Loaded: %d modules, %d objects, %d types\n\n",
		len(m.Modules()), len(m.Objects()), len(m.Types()))

	c.traceSymbol(m, symbol)

	return exitOK
}

func (c *cli) traceSymbol(m *mib.Mib, symbol string) {
	fmt.Printf("=== Tracing symbol: %s ===\n\n", symbol)

	var definingModules []string
	for _, mod := range m.Modules() {
		if mod.Object(symbol) != nil || mod.Type(symbol) != nil || mod.Node(symbol) != nil {
			definingModules = append(definingModules, mod.Name())
		}
	}

	node := m.Node(symbol)
	obj := m.Object(symbol)
	typ := m.Type(symbol)

	fmt.Println("DEFINITIONS:")
	if len(definingModules) == 0 {
		fmt.Println("  (none found)")
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
			fmt.Printf("  %s: %s\n", modName, strings.Join(kinds, ", "))
		}
	}
	fmt.Println()

	fmt.Println("GLOBAL LOOKUPS (unqualified):")
	if node != nil {
		modName := "(no module)"
		if node.Module() != nil {
			modName = node.Module().Name()
		}
		fmt.Printf("  Node:        %s::%s  OID=%s  Kind=%s\n",
			modName, node.Name(), node.OID(), node.Kind())
		fmt.Printf("               Object attached: %v\n", node.Object() != nil)
		fmt.Printf("               Notification attached: %v\n", node.Notification() != nil)
	} else {
		fmt.Println("  Node:        (not found)")
	}

	if obj != nil {
		modName := "(no module)"
		if obj.Module() != nil {
			modName = obj.Module().Name()
		}
		fmt.Printf("  Object:      %s::%s  OID=%s  Kind=%s\n",
			modName, obj.Name(), obj.OID(), obj.Kind())
	} else {
		fmt.Println("  Object:      (not found)")
	}

	if typ != nil {
		modName := "(no module)"
		if typ.Module() != nil {
			modName = typ.Module().Name()
		}
		fmt.Printf("  Type:        %s::%s  Base=%s\n",
			modName, typ.Name(), typ.Base())
	} else {
		fmt.Println("  Type:        (not found)")
	}
	fmt.Println()

	if len(definingModules) > 1 {
		fmt.Println("PER-MODULE LOOKUPS:")
		for _, modName := range definingModules {
			mod := m.Module(modName)
			modObj := mod.Object(symbol)
			if modObj != nil {
				fmt.Printf("  %s::%s:\n", modName, symbol)
				fmt.Printf("    OID=%s  Kind=%s\n", modObj.OID(), modObj.Kind())
				if modObj.Kind() == mib.KindRow {
					fmt.Printf("    Index count: %d\n", len(modObj.Index()))
					for i, idx := range modObj.Index() {
						name := "(nil!)"
						if idx.Object != nil {
							name = idx.Object.Name()
						}
						enc := ""
						if idx.Encoding != mib.IndexEncodingUnknown {
							enc = fmt.Sprintf("  encoding=%s", idx.Encoding)
						}
						fmt.Printf("      [%d] %s%s\n", i, name, enc)
					}
				}
			}
		}
		fmt.Println()
	}

	if obj != nil && obj.Kind() == mib.KindRow {
		fmt.Println("INDEX RESOLUTION (row object):")
		directIndex := obj.Index()
		effectiveIndex := obj.EffectiveIndexes()

		if len(directIndex) == 0 && obj.Augments() == nil {
			fmt.Println("  WARNING: No INDEX clause resolved!")
			fmt.Println("  This row has no index entries, which may indicate a resolution failure.")
		} else if len(directIndex) == 0 && obj.Augments() != nil {
			fmt.Printf("  AUGMENTS: %s\n", obj.Augments().Name())
		}

		if len(directIndex) > 0 {
			fmt.Printf("  Direct INDEX (%d entries):\n", len(directIndex))
			printIndexDetails(directIndex)
		}

		if len(effectiveIndex) > 0 && len(effectiveIndex) != len(directIndex) {
			fmt.Printf("  Effective INDEX (%d entries, via AUGMENTS chain):\n", len(effectiveIndex))
			printIndexDetails(effectiveIndex)
		}
		fmt.Println()
	}

	if obj != nil && obj.Kind() == mib.KindTable {
		fmt.Println("TABLE STRUCTURE:")
		if obj.Entry() != nil {
			entry := obj.Entry()
			fmt.Printf("  Row entry: %s\n", entry.Name())
			fmt.Printf("  Row INDEX count: %d\n", len(entry.Index()))
			if len(entry.Index()) == 0 && entry.Augments() == nil {
				fmt.Println("  WARNING: Row has no INDEX resolved!")
			}

			effIdx := entry.EffectiveIndexes()
			if len(effIdx) > 0 {
				fmt.Printf("  Effective indexes (%d):\n", len(effIdx))
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
					fmt.Printf("    [%d] %s%s%s\n", i, name, implied, enc)
				}
			}

			if augBy := entry.AugmentedBy(); len(augBy) > 0 {
				names := make([]string, len(augBy))
				for i, a := range augBy {
					names[i] = a.Name()
				}
				fmt.Printf("  Augmented by: %s\n", strings.Join(names, ", "))
			}

			cols := entry.Columns()
			if len(cols) > 0 {
				fmt.Println("  Columns:")
				printColumnTable(cols)
			}
		} else {
			fmt.Println("  WARNING: No row entry found!")
		}
		fmt.Println()
	}

	unresolved := m.Unresolved()
	var related []mib.UnresolvedRef
	for _, u := range unresolved {
		if u.Symbol == symbol || strings.Contains(u.Symbol, symbol) {
			related = append(related, u)
		}
	}

	if len(related) > 0 {
		fmt.Println("RELATED UNRESOLVED REFERENCES:")
		for _, u := range related {
			fmt.Printf("  [%s] %s in module %s\n", u.Kind, u.Symbol, u.Module)
		}
		fmt.Println()
	}

	if len(definingModules) > 1 {
		fmt.Println("WARNING: Multiple modules define this symbol!")
		fmt.Println("  This can cause resolution ambiguity depending on load order.")
		fmt.Println("  Modules:", strings.Join(definingModules, ", "))

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
			fmt.Printf("  Objects share same OID node: %v\n", same)
			for _, info := range nodeInfo {
				fmt.Printf("    %s\n", info)
			}
		}
		fmt.Println()
	}

	if c.verbose > 0 && len(unresolved) > 0 {
		fmt.Printf("ALL UNRESOLVED REFERENCES (%d total):\n", len(unresolved))
		for _, u := range unresolved {
			fmt.Printf("  [%s] %s in module %s\n", u.Kind, u.Symbol, u.Module)
		}
	}
}

func printIndexDetails(indexes []mib.IndexEntry) {
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
		fmt.Printf("    [%d] %s  OID=%s%s%s\n", i, name, oid, implied, enc)
	}
}
