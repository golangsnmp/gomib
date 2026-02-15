//go:build cgo

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

func cmdFixturegen(args []string) int {
	fs := flag.NewFlagSet("fixturegen", flag.ContinueOnError)

	var outDir string
	fs.StringVar(&outDir, "dir", "", "Output directory for fixture files (default: stdout for single module)")

	fs.Usage = func() {
		_, _ = fmt.Fprintf(fs.Output(), `Usage: gomib-netsnmp fixturegen [options] MODULE [MODULE...]

Generate JSON fixture files from net-snmp for ground-truth testing.
Each module produces a JSON file containing all NormalizedNode entries
keyed by OID.

Options:
`)
		fs.PrintDefaults()
	}

	if err := fs.Parse(args); err != nil {
		return 1
	}

	modules := fs.Args()
	if len(modules) == 0 {
		fmt.Fprintln(os.Stderr, "error: at least one MODULE argument is required")
		fs.Usage()
		return 1
	}

	mibPaths := getMIBPaths()

	fmt.Fprintln(os.Stderr, "Loading MIBs with net-snmp...")
	netsnmpNodes, err := loadNetSnmpNodes(mibPaths, modules)
	if err != nil {
		printError("net-snmp load failed: %v", err)
		return 1
	}

	for _, mod := range modules {
		filtered := filterByModules(netsnmpNodes, []string{mod})
		if len(filtered) == 0 {
			fmt.Fprintf(os.Stderr, "warning: no nodes found for module %s\n", mod)
			continue
		}

		data, err := json.MarshalIndent(filtered, "", "  ")
		if err != nil {
			printError("json marshal failed for %s: %v", mod, err)
			return 1
		}

		if outDir != "" {
			if err := os.MkdirAll(outDir, 0o755); err != nil {
				printError("cannot create output directory: %v", err)
				return 1
			}
			path := filepath.Join(outDir, mod+".json")
			if err := os.WriteFile(path, append(data, '\n'), 0o644); err != nil {
				printError("cannot write %s: %v", path, err)
				return 1
			}
			fmt.Fprintf(os.Stderr, "wrote %s (%d nodes)\n", path, len(filtered))
		} else {
			fmt.Println(string(data))
		}
	}

	return 0
}
