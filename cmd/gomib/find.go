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
  gomib find [options] -m MODULE PATTERN
  gomib find [options] MODULE... -- PATTERN
  gomib find [options] MODULE... PATTERN
  gomib find [options] --all PATTERN

Searches object and notification names using glob-style patterns (*, ?).

Options:
  -m, --module MODULE   Module to load (repeatable)
  --all                 Load all MIBs from search path
  --kind KIND           Filter by node kind (scalar, table, row, column, notification)
  --type BASE           Filter by base type (Integer32, OctetString, Counter32, etc.)
  --count               Print only the match count
  --strict              Resolver strictness: strict (tier-1 only)
  --permissive          Resolver strictness: permissive (tier-1/2/3)
  -h, --help            Show help

Examples:
  gomib find --all -p testdata/corpus/primary 'if*'
  gomib find --all -p testdata/corpus/primary --kind table '*'
  gomib find --all -p testdata/corpus/primary --type Counter32 '*'
  gomib find IF-MIB 'if*'
  gomib find -m IF-MIB -p testdata/corpus/primary 'if*'
`

func (c *cli) cmdFind(args []string) int {
	fs := flag.NewFlagSet("find", flag.ContinueOnError)
	fs.Usage = func() { fmt.Fprint(os.Stderr, findUsage) }

	modules, loadAll := addModuleFlags(fs)
	kindFilter := fs.String("kind", "", "filter by node kind")
	typeFilter := fs.String("type", "", "filter by base type name")
	count := fs.Bool("count", false, "print only match count")
	strict, permissive := addStrictnessFlags(fs)
	help := addHelpFlag(fs)

	if err := fs.Parse(args); err != nil {
		return exitError
	}

	if c.checkHelp(help, findUsage) {
		return exitOK
	}

	if code, failed := validateStrictnessFlags(*strict, *permissive); failed {
		return code
	}

	pattern, code, failed := parseSingleTargetArgs(modules, *loadAll, fs.Args(), findUsage, "pattern")
	if failed {
		return code
	}
	pattern = strings.ToLower(pattern)

	if _, err := filepath.Match(pattern, ""); err != nil {
		printError("invalid pattern: %v", err)
		return exitError
	}

	if code, ok := requireModuleOrAll(*modules, *loadAll, findUsage); ok {
		return code
	}

	m, err := c.loadMibWithOpts(modulesToLoad(*modules, *loadAll), strictnessOpts(*strict, *permissive)...)
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
			return exitError
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

	for _, notif := range m.Notifications() {
		if !matchGlob(pattern, strings.ToLower(notif.Name())) {
			continue
		}
		if *kindFilter != "" && mib.KindNotification != kind {
			continue
		}
		matches++
		if !*count {
			modName := ""
			if notif.Module() != nil {
				modName = notif.Module().Name()
			}
			fmt.Printf("%s::%s  %s  %s\n", modName, notif.Name(), notif.OID(), mib.KindNotification)
		}
	}

	if *count {
		fmt.Println(matches)
	}
	return exitOK
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
	default:
		return 0, false
	}
}
