package main

import (
	"encoding/json"
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

Searches object, notification, group, compliance, capability, and node names
using glob-style patterns (*, ?).

Options:
  -m, --module MODULE   Module to load (repeatable, default: all)
  --kind KIND           Filter by kind (scalar, table, row, column, notification,
                        node, group, compliance, capability, module-identity,
                        object-identity)
  --type BASE           Filter by base type (Integer32, OctetString, Counter32, etc.)
  --count               Print only the match count
  --format FMT          Output format: text, json (default: text)
  --strict              Resolver strictness: strict (tier-1 only)
  --permissive          Resolver strictness: permissive (tier-1/2/3)
  -h, --help            Show help

Examples:
  gomib find -p testdata/corpus/primary 'if*'
  gomib find -p testdata/corpus/primary --kind table '*'
  gomib find -p testdata/corpus/primary --type Counter32 '*'
  gomib find -m IF-MIB 'if*'
  gomib find --kind notification '*linkDown*'
`

type findMatch struct {
	Name   string `json:"name"`
	Module string `json:"module,omitempty"`
	OID    string `json:"oid"`
	Kind   string `json:"kind"`
}

func (c *cli) cmdFind(args []string) int {
	fs := flag.NewFlagSet("find", flag.ContinueOnError)
	fs.Usage = func() { fmt.Fprint(os.Stderr, findUsage) }

	modules := addModuleFlag(fs)
	kindFilter := fs.String("kind", "", "filter by node kind")
	typeFilter := fs.String("type", "", "filter by base type name")
	count := fs.Bool("count", false, "print only match count")
	format := fs.String("format", "text", "output format: text, json")
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

	positional := fs.Args()
	if len(positional) == 0 {
		printError("no pattern specified")
		fmt.Fprint(os.Stderr, findUsage)
		return exitError
	}
	if len(positional) > 1 {
		printError("too many arguments (use -m for module selection)")
		fmt.Fprint(os.Stderr, findUsage)
		return exitError
	}
	pattern := strings.ToLower(positional[0])

	if _, err := filepath.Match(pattern, ""); err != nil {
		printError("invalid pattern: %v", err)
		return exitError
	}

	switch *format {
	case "text", "json":
	default:
		printError("unknown format: %s", *format)
		return exitError
	}

	// Default to permissive+silent for query commands
	opts := strictnessOpts(*strict, *permissive)
	if len(opts) == 0 {
		opts = permissiveSilentOpts()
	}

	var mods []string
	if len(*modules) > 0 {
		mods = *modules
	}

	m, err := c.loadMibWithOpts(mods, opts...)
	if err != nil {
		printError("failed to load: %v", err)
		return exitError
	}

	var kind mib.Kind
	var kindIsModIdent, kindIsObjIdent bool
	if *kindFilter != "" {
		var ok bool
		kind, kindIsModIdent, kindIsObjIdent, ok = parseKindFilter(*kindFilter)
		if !ok {
			printError("unknown kind: %s (valid: scalar, table, row, column, notification, node, group, compliance, capability, module-identity, object-identity)", *kindFilter)
			return exitError
		}
	}

	baseLower := strings.ToLower(*typeFilter)
	var matches []findMatch

	// Search objects (scalar, table, row, column)
	for _, obj := range m.Objects() {
		if !matchGlob(pattern, strings.ToLower(obj.Name())) {
			continue
		}
		if *kindFilter != "" {
			if kindIsModIdent || kindIsObjIdent {
				continue // objects are never module-identity or object-identity
			}
			if obj.Kind() != kind {
				continue
			}
		}
		if *typeFilter != "" && !matchBaseType(obj, baseLower) {
			continue
		}
		matches = append(matches, findMatch{
			Name:   obj.Name(),
			Module: moduleName(obj.Module()),
			OID:    obj.OID().String(),
			Kind:   obj.Kind().String(),
		})
	}

	// Search notifications
	for _, notif := range m.Notifications() {
		if !matchGlob(pattern, strings.ToLower(notif.Name())) {
			continue
		}
		if *kindFilter != "" && !kindIsModIdent && !kindIsObjIdent && kind != mib.KindNotification {
			continue
		}
		if kindIsModIdent || kindIsObjIdent {
			continue
		}
		matches = append(matches, findMatch{
			Name:   notif.Name(),
			Module: moduleName(notif.Module()),
			OID:    notif.OID().String(),
			Kind:   mib.KindNotification.String(),
		})
	}

	// Search groups
	for _, grp := range m.Groups() {
		if !matchGlob(pattern, strings.ToLower(grp.Name())) {
			continue
		}
		if *kindFilter != "" && kind != mib.KindGroup && !kindIsModIdent && !kindIsObjIdent {
			continue
		}
		if kindIsModIdent || kindIsObjIdent {
			continue
		}
		matches = append(matches, findMatch{
			Name:   grp.Name(),
			Module: moduleName(grp.Module()),
			OID:    grp.OID().String(),
			Kind:   mib.KindGroup.String(),
		})
	}

	// Search compliances
	for _, comp := range m.Compliances() {
		if !matchGlob(pattern, strings.ToLower(comp.Name())) {
			continue
		}
		if *kindFilter != "" && kind != mib.KindCompliance && !kindIsModIdent && !kindIsObjIdent {
			continue
		}
		if kindIsModIdent || kindIsObjIdent {
			continue
		}
		matches = append(matches, findMatch{
			Name:   comp.Name(),
			Module: moduleName(comp.Module()),
			OID:    comp.OID().String(),
			Kind:   mib.KindCompliance.String(),
		})
	}

	// Search capabilities
	for _, cap := range m.Capabilities() {
		if !matchGlob(pattern, strings.ToLower(cap.Name())) {
			continue
		}
		if *kindFilter != "" && kind != mib.KindCapability && !kindIsModIdent && !kindIsObjIdent {
			continue
		}
		if kindIsModIdent || kindIsObjIdent {
			continue
		}
		matches = append(matches, findMatch{
			Name:   cap.Name(),
			Module: moduleName(cap.Module()),
			OID:    cap.OID().String(),
			Kind:   mib.KindCapability.String(),
		})
	}

	// Walk the OID tree for node/module-identity/object-identity
	for node := range m.Nodes() {
		if node.Kind() != mib.KindNode {
			continue
		}
		if node.Name() == "" {
			continue
		}
		if !matchGlob(pattern, strings.ToLower(node.Name())) {
			continue
		}
		if *kindFilter != "" {
			if kindIsModIdent && node.IsObjectIdentity() {
				continue
			}
			if kindIsObjIdent && !node.IsObjectIdentity() {
				continue
			}
			if !kindIsModIdent && !kindIsObjIdent && kind != mib.KindNode {
				continue
			}
		}
		kindLabel := "node"
		if node.IsObjectIdentity() {
			kindLabel = "object-identity"
		}
		matches = append(matches, findMatch{
			Name:   node.Name(),
			Module: moduleName(node.Module()),
			OID:    node.OID().String(),
			Kind:   kindLabel,
		})
	}

	switch {
	case *count:
		fmt.Println(len(matches))
	case *format == "json":
		if matches == nil {
			matches = []findMatch{}
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(matches); err != nil {
			printError("encoding JSON: %v", err)
			return exitError
		}
	default:
		for _, match := range matches {
			fmt.Printf("%s::%s  %s  %s\n", match.Module, match.Name, match.OID, match.Kind)
		}
	}

	if len(matches) == 0 {
		return exitIssue
	}
	return exitOK
}

func moduleName(mod *mib.Module) string {
	if mod != nil {
		return mod.Name()
	}
	return ""
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

func parseKindFilter(s string) (kind mib.Kind, isModIdent, isObjIdent, ok bool) {
	switch strings.ToLower(s) {
	case "scalar":
		return mib.KindScalar, false, false, true
	case "table":
		return mib.KindTable, false, false, true
	case "row":
		return mib.KindRow, false, false, true
	case "column":
		return mib.KindColumn, false, false, true
	case "notification":
		return mib.KindNotification, false, false, true
	case "node":
		return mib.KindNode, false, false, true
	case "group":
		return mib.KindGroup, false, false, true
	case "compliance":
		return mib.KindCompliance, false, false, true
	case "capability":
		return mib.KindCapability, false, false, true
	case "module-identity":
		return mib.KindNode, true, false, true
	case "object-identity":
		return mib.KindNode, false, true, true
	default:
		return 0, false, false, false
	}
}
