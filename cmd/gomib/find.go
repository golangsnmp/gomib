package main

import (
	"flag"
	"fmt"
	"path"
	"strings"

	"github.com/golangsnmp/gomib/internal/types"
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
	fs.Usage = func() { fmt.Fprint(c.stderr, findUsage) }

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

	if code, failed := c.validateStrictnessFlags(*strict, *permissive); failed {
		return code
	}

	positional := fs.Args()
	if len(positional) == 0 {
		c.printError("no pattern specified")
		fmt.Fprint(c.stderr, findUsage)
		return exitError
	}
	if len(positional) > 1 {
		c.printError("too many arguments (use -m for module selection)")
		fmt.Fprint(c.stderr, findUsage)
		return exitError
	}
	pattern := strings.ToLower(positional[0])

	if _, err := path.Match(pattern, ""); err != nil {
		c.printError("invalid pattern: %v", err)
		return exitError
	}

	switch *format {
	case "text", "json":
	default:
		c.printError("unknown format: %s", *format)
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
	if err != nil && m == nil {
		c.printError("failed to load: %v", err)
		return exitError
	}

	var kind mib.Kind
	var kindIsModIdent, kindIsObjIdent bool
	if *kindFilter != "" {
		var ok bool
		kind, kindIsModIdent, kindIsObjIdent, ok = parseKindFilter(*kindFilter)
		if !ok {
			c.printError("unknown kind: %s (valid: scalar, table, row, column, notification, node, group, compliance, capability, module-identity, object-identity)", *kindFilter)
			return exitError
		}
	}

	f := findFilter{
		pattern:    pattern,
		kind:       kind,
		hasKind:    *kindFilter != "",
		isModIdent: kindIsModIdent,
		isObjIdent: kindIsObjIdent,
		baseLower:  strings.ToLower(*typeFilter),
	}
	var matches []findMatch

	// Search objects (kind is per-object, type filter applies)
	for _, obj := range m.Objects() {
		if !f.matchName(obj.Name()) || !f.matchKind(obj.Kind()) || !f.matchType(obj) {
			continue
		}
		matches = append(matches, findMatch{
			Name:   obj.Name(),
			Module: moduleName(obj.Module()),
			OID:    obj.OID().String(),
			Kind:   obj.Kind().String(),
		})
	}

	if f.baseLower == "" {
		matches = collectMatches(matches, m.Notifications(), mib.KindNotification, &f)
		matches = collectMatches(matches, m.Groups(), mib.KindGroup, &f)
		matches = collectMatches(matches, m.Compliances(), mib.KindCompliance, &f)
		matches = collectMatches(matches, m.Capabilities(), mib.KindCapability, &f)

		// Walk OID tree for node/module-identity/object-identity
		for node := range m.Nodes() {
			if node.Kind() != mib.KindNode || node.Name() == "" {
				continue
			}
			if !f.matchName(node.Name()) || !f.matchNodeKind(node) {
				continue
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
	}

	switch {
	case *count:
		fmt.Fprintln(c.stdout, len(matches))
	case *format == "json":
		if matches == nil {
			matches = []findMatch{}
		}
		if err := writeJSON(c.stdout, matches, true); err != nil {
			c.printError("encoding JSON: %v", err)
			return exitError
		}
	default:
		for _, match := range matches {
			fmt.Fprintf(c.stdout, "%s::%s  %s  %s\n", match.Module, match.Name, match.OID, match.Kind)
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

func matchBaseType(obj *mib.Object, baseLower string) bool {
	if obj.Type() == nil {
		return false
	}
	return strings.ToLower(obj.Type().Base().String()) == baseLower
}

// findFilter holds parsed search criteria for the find command.
type findFilter struct {
	pattern    string
	kind       mib.Kind
	hasKind    bool
	isModIdent bool
	isObjIdent bool
	baseLower  string
}

func (f *findFilter) matchName(name string) bool {
	return types.MatchGlob(f.pattern, strings.ToLower(name))
}

// matchKind checks whether an entity kind passes the kind filter.
// Works for objects (dynamic kind) and fixed-kind entities alike.
func (f *findFilter) matchKind(entityKind mib.Kind) bool {
	if !f.hasKind {
		return true
	}
	if f.isModIdent || f.isObjIdent {
		return false
	}
	return entityKind == f.kind
}

// matchNodeKind checks whether a bare node passes the kind filter,
// handling the module-identity vs object-identity distinction.
func (f *findFilter) matchNodeKind(node *mib.Node) bool {
	if !f.hasKind {
		return true
	}
	if f.isModIdent {
		return !node.IsObjectIdentity()
	}
	if f.isObjIdent {
		return node.IsObjectIdentity()
	}
	return f.kind == mib.KindNode
}

func (f *findFilter) matchType(obj *mib.Object) bool {
	if f.baseLower == "" {
		return true
	}
	return matchBaseType(obj, f.baseLower)
}

// findable is the common interface shared by Notification, Group,
// Compliance, and Capability entity types.
type findable interface {
	Name() string
	Module() *mib.Module
	OID() mib.OID
}

// collectMatches appends matching entities of a fixed kind to the results.
func collectMatches[E findable](matches []findMatch, entities []E, kind mib.Kind, f *findFilter) []findMatch {
	if !f.matchKind(kind) {
		return matches
	}
	kindStr := kind.String()
	for _, e := range entities {
		if !f.matchName(e.Name()) {
			continue
		}
		matches = append(matches, findMatch{
			Name:   e.Name(),
			Module: moduleName(e.Module()),
			OID:    e.OID().String(),
			Kind:   kindStr,
		})
	}
	return matches
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
