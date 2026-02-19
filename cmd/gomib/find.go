package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/golangsnmp/gomib/mib"
)

const findUsage = `gomib find - Search for names across loaded MIBs

Usage:
  gomib find [options] PATTERN

Searches object and type names using glob-style patterns (*, ?).
Requires either -m MODULE or --all.

Options:
  -m, --module MODULE   Module to load (repeatable)
  --all                 Load all MIBs from search path
  --kind KIND           Filter by node kind (scalar, table, row, column, notification)
  --type BASE           Filter by base type (Integer32, OctetString, Counter32, etc.)
  --count               Print only the match count
  -h, --help            Show help

Examples:
  gomib find --all -p testdata/corpus/primary 'if*'
  gomib find --all -p testdata/corpus/primary --kind table '*'
  gomib find --all -p testdata/corpus/primary --type Counter32 '*'
  gomib find -m IF-MIB -p testdata/corpus/primary 'if*'
`

func (c *cli) cmdFind(args []string) int {
	fs := flag.NewFlagSet("find", flag.ContinueOnError)
	fs.Usage = func() { fmt.Fprint(os.Stderr, findUsage) }

	var modules moduleList
	fs.Var(&modules, "m", "module to load")
	fs.Var(&modules, "module", "module to load")
	loadAll := fs.Bool("all", false, "load all MIBs from search path")
	kindFilter := fs.String("kind", "", "filter by node kind")
	typeFilter := fs.String("type", "", "filter by base type name")
	count := fs.Bool("count", false, "print only match count")
	help := fs.Bool("h", false, "show help")
	fs.BoolVar(help, "help", false, "show help")

	if err := fs.Parse(args); err != nil {
		return 1
	}

	if *help || c.helpFlag {
		_, _ = fmt.Fprint(os.Stdout, findUsage)
		return 0
	}

	remaining := fs.Args()
	if len(remaining) == 0 {
		printError("no pattern specified")
		fmt.Fprint(os.Stderr, findUsage)
		return 1
	}
	pattern := strings.ToLower(remaining[0])

	if !*loadAll && len(modules) == 0 {
		printError("specify -m MODULE or --all")
		fmt.Fprint(os.Stderr, findUsage)
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

	var kind mib.Kind
	if *kindFilter != "" {
		var ok bool
		kind, ok = parseKindFilter(*kindFilter)
		if !ok {
			printError("unknown kind: %s", *kindFilter)
			return 1
		}
	}

	baseLower := strings.ToLower(*typeFilter)
	matches := 0

	for _, obj := range m.Objects() {
		if !matchGlob(pattern, strings.ToLower(obj.Name())) {
			continue
		}
		if *kindFilter != "" && obj.Kind() != kind {
			continue
		}
		if *typeFilter != "" && !matchBaseType(obj, baseLower) {
			continue
		}
		matches++
		if !*count {
			modName := ""
			if obj.Module() != nil {
				modName = obj.Module().Name()
			}
			fmt.Printf("%s::%s  %s  %s\n", modName, obj.Name(), obj.OID(), obj.Kind())
		}
	}

	if *count {
		fmt.Println(matches)
	}
	return 0
}

func matchGlob(pattern, name string) bool {
	ok, _ := filepath.Match(pattern, name)
	return ok
}

func matchBaseType(obj *mib.Object, baseLower string) bool {
	if obj.Type() == nil {
		return false
	}
	return strings.ToLower(obj.Type().Base().String()) == baseLower
}

func parseKindFilter(s string) (mib.Kind, bool) {
	switch strings.ToLower(s) {
	case "scalar":
		return mib.KindScalar, true
	case "table":
		return mib.KindTable, true
	case "row":
		return mib.KindRow, true
	case "column":
		return mib.KindColumn, true
	case "notification":
		return mib.KindNotification, true
	case "node":
		return mib.KindNode, true
	default:
		return 0, false
	}
}
