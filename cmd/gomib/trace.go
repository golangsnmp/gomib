package main

import (
	"flag"
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/golangsnmp/gomib"
)

const traceUsage = `gomib trace - Trace symbol resolution for debugging

Usage:
  gomib trace [options] SYMBOL
  gomib trace [options] -m MODULE SYMBOL

Traces how a symbol is resolved across loaded modules. Useful for debugging
resolution issues like missing INDEX references or duplicate definitions.

Options:
  -m, --module MODULE   Module to load (repeatable, uses LoadModules)
  --all                 Load all MIBs from search path (uses Load)
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
  gomib trace --all ifIndex
  gomib trace --all -p testdata/corpus/primary ifEntry
`

func cmdTrace(args []string) int {
	fs := flag.NewFlagSet("trace", flag.ContinueOnError)
	fs.Usage = func() { fmt.Fprint(os.Stderr, traceUsage) }

	var modules moduleList
	fs.Var(&modules, "m", "module to load")
	fs.Var(&modules, "module", "module to load")
	loadAll := fs.Bool("all", false, "load all MIBs from search path")
	help := fs.Bool("h", false, "show help")
	fs.BoolVar(help, "help", false, "show help")

	if err := fs.Parse(args); err != nil {
		return 1
	}

	if *help || helpFlag {
		_, _ = fmt.Fprint(os.Stdout, traceUsage)
		return 0
	}

	remaining := fs.Args()
	if len(remaining) == 0 {
		printError("no symbol specified")
		fmt.Fprint(os.Stderr, traceUsage)
		return 1
	}

	symbol := remaining[0]

	if !*loadAll && len(modules) == 0 {
		printError("specify -m MODULE or --all")
		fmt.Fprint(os.Stderr, traceUsage)
		return 1
	}

	var mib *gomib.Mib
	var err error
	var loadMode string

	if *loadAll {
		loadMode = "Load() - all modules from search path"
		mib, err = loadMib(nil)
	} else {
		loadMode = fmt.Sprintf("LoadModules(%v)", modules)
		mib, err = loadMib(modules)
	}

	if err != nil {
		printError("failed to load: %v", err)
		return 2
	}

	fmt.Printf("Load mode: %s\n", loadMode)
	fmt.Printf("Loaded: %d modules, %d objects, %d types\n\n",
		len(mib.Modules()), len(mib.Objects()), len(mib.Types()))

	traceSymbol(mib, symbol)

	return 0
}

func traceSymbol(mib *gomib.Mib, symbol string) {
	fmt.Printf("=== Tracing symbol: %s ===\n\n", symbol)

	var definingModules []string
	for _, mod := range mib.Modules() {
		if mod.Object(symbol) != nil || mod.Type(symbol) != nil || mod.Node(symbol) != nil {
			definingModules = append(definingModules, mod.Name())
		}
	}

	node := mib.Node(symbol)
	obj := mib.Object(symbol)
	typ := mib.Type(symbol)

	fmt.Println("DEFINITIONS:")
	if len(definingModules) == 0 {
		fmt.Println("  (none found)")
	} else {
		slices.Sort(definingModules)
		for _, modName := range definingModules {
			mod := mib.Module(modName)
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
			mod := mib.Module(modName)
			modObj := mod.Object(symbol)
			if modObj != nil {
				fmt.Printf("  %s::%s:\n", modName, symbol)
				fmt.Printf("    OID=%s  Kind=%s\n", modObj.OID(), modObj.Kind())
				if modObj.Kind() == gomib.KindRow {
					fmt.Printf("    Index count: %d\n", len(modObj.Index()))
					for i, idx := range modObj.Index() {
						name := "(nil!)"
						if idx.Object != nil {
							name = idx.Object.Name()
						}
						fmt.Printf("      [%d] %s\n", i, name)
					}
				}
			}
		}
		fmt.Println()
	}

	if obj != nil && obj.Kind() == gomib.KindRow {
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
			for i, idx := range directIndex {
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
				fmt.Printf("    [%d] %s  OID=%s%s\n", i, name, oid, implied)
			}
		}

		if len(effectiveIndex) > 0 && len(effectiveIndex) != len(directIndex) {
			fmt.Printf("  Effective INDEX (%d entries, via AUGMENTS chain):\n", len(effectiveIndex))
			for i, idx := range effectiveIndex {
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
				fmt.Printf("    [%d] %s  OID=%s%s\n", i, name, oid, implied)
			}
		}
		fmt.Println()
	}

	if obj != nil && obj.Kind() == gomib.KindTable {
		fmt.Println("TABLE STRUCTURE:")
		if obj.Entry() != nil {
			entry := obj.Entry()
			fmt.Printf("  Row entry: %s\n", entry.Name())
			fmt.Printf("  Row INDEX count: %d\n", len(entry.Index()))
			if len(entry.Index()) == 0 && entry.Augments() == nil {
				fmt.Println("  WARNING: Row has no INDEX resolved!")
			}
		} else {
			fmt.Println("  WARNING: No row entry found!")
		}
		fmt.Println()
	}

	unresolved := mib.Unresolved()
	var related []gomib.UnresolvedRef
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

		var nodes []*gomib.Node
		var nodeInfo []string
		for _, modName := range definingModules {
			mod := mib.Module(modName)
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

	if verbose > 0 && len(unresolved) > 0 {
		fmt.Printf("ALL UNRESOLVED REFERENCES (%d total):\n", len(unresolved))
		for _, u := range unresolved {
			fmt.Printf("  [%s] %s in module %s\n", u.Kind, u.Symbol, u.Module)
		}
	}
}
