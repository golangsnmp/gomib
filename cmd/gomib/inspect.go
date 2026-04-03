package main

import (
	"flag"
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/golangsnmp/gomib"
	"github.com/golangsnmp/gomib/mib"
)

const inspectUsage = `gomib inspect - Deep-dive inspection of a MIB symbol

Usage:
  gomib inspect [options] QUERY

Shows detailed information about a symbol including type chain,
import provenance, group membership, and scoped diagnostics.

Query formats:
  Name:            ifIndex
  Qualified:       IF-MIB::ifIndex
  Numeric OID:     1.3.6.1.2.1.2.2.1.1
  Type name:       DisplayString

Options:
  -m, --module MODULE   Module to load (repeatable, default: all)
  --strict              Resolver strictness: strict (tier-1 only)
  --permissive          Resolver strictness: permissive (tier-1/2/3)
  -h, --help            Show help

Examples:
  gomib inspect ifDescr
  gomib inspect -m IF-MIB ifEntry
  gomib inspect IF-MIB::ifIndex
  gomib inspect DisplayString
`

func (c *cli) cmdInspect(args []string) int {
	fs := flag.NewFlagSet("inspect", flag.ContinueOnError)
	fs.Usage = func() { fmt.Fprint(os.Stderr, inspectUsage) }

	modules := addModuleFlag(fs)
	strict, permissive := addStrictnessFlags(fs)
	help := addHelpFlag(fs)

	if err := fs.Parse(args); err != nil {
		return exitError
	}

	if c.checkHelp(help, inspectUsage) {
		return exitOK
	}

	if code, failed := validateStrictnessFlags(*strict, *permissive); failed {
		return code
	}

	positional := fs.Args()
	if len(positional) == 0 {
		printError("no query specified")
		fmt.Fprint(os.Stderr, inspectUsage)
		return exitError
	}
	if len(positional) > 1 {
		printError("too many arguments (use -m for module selection)")
		fmt.Fprint(os.Stderr, inspectUsage)
		return exitError
	}
	query := positional[0]

	// Use verbose diagnostics so we collect everything for display.
	// Default to permissive resolver to maximize resolution.
	opts := strictnessOpts(*strict, *permissive)
	if len(opts) == 0 {
		opts = []gomib.LoadOption{
			gomib.WithResolverStrictness(mib.ResolverPermissive),
		}
	}
	opts = append(opts, gomib.WithDiagnosticConfig(mib.VerboseConfig()))

	var mods []string
	if len(*modules) > 0 {
		mods = *modules
	}

	m, err := c.loadMibWithOpts(mods, opts...)
	if err != nil && m == nil {
		printError("failed to load: %v", err)
		return exitError
	}

	// Try node resolution first, then fall back to type lookup.
	node := m.Resolve(query)
	if node != nil {
		inspectNode(m, node)
		return exitOK
	}

	// Try as a type name (types don't always have nodes).
	typName := query
	if modName, name, ok := strings.Cut(query, "::"); ok {
		mod := m.Module(modName)
		if mod != nil {
			if typ := mod.Type(name); typ != nil {
				inspectStandaloneType(m, typ)
				return exitOK
			}
		}
	} else if typ := m.Type(typName); typ != nil {
		inspectStandaloneType(m, typ)
		return exitOK
	}

	printError("not found: %s", query)
	return exitIssue
}

func inspectNode(m *mib.Mib, node *mib.Node) {
	printIdentity(node)

	switch {
	case node.Object() != nil:
		inspectObject(m, node.Object())
	case node.Notification() != nil:
		inspectNotification(m, node.Notification())
	case node.Group() != nil:
		inspectGroup(m, node.Group())
	case node.Compliance() != nil:
		inspectCompliance(m, node.Compliance())
	case node.Capability() != nil:
		inspectCapability(m, node.Capability())
	default:
		inspectBareNode(m, node)
	}
}

func printIdentity(node *mib.Node) {
	label := node.Name()
	if label == "" {
		label = fmt.Sprintf("(%d)", node.Arc())
	}
	fmt.Printf("Name:    %s\n", label)
	if node.Module() != nil {
		fmt.Printf("Module:  %s\n", node.Module().Name())
	}
	fmt.Printf("OID:     %s\n", node.OID().String())
	fmt.Printf("Kind:    %s\n", node.Kind().String())
}

// inspectObject prints full detail for an OBJECT-TYPE.
func inspectObject(m *mib.Mib, obj *mib.Object) {
	fmt.Printf("Status:  %s\n", obj.Status().String())
	fmt.Printf("Access:  %s\n", obj.Access().String())

	if obj.Units() != "" {
		fmt.Printf("Units:   %s\n", obj.Units())
	}

	if dv := obj.DefaultValue(); !dv.IsZero() {
		fmt.Printf("DefVal:  %s\n", dv.String())
	}

	// Type summary line (like get).
	if obj.Type() != nil {
		typ := obj.Type()
		typeName := typ.Name()
		if typeName == "" {
			typeName = typ.EffectiveBase().String()
		}
		base := typ.EffectiveBase().String()
		if typeName != base {
			fmt.Printf("Type:    %s (%s)\n", typeName, base)
		} else {
			fmt.Printf("Type:    %s\n", typeName)
		}
	}

	// Index / augments.
	printObjectIndexAugments(obj)

	// Column context.
	printObjectColumnContext(obj)

	// Type chain.
	if obj.Type() != nil {
		printTypeChain(obj.Type())
	}

	// Enum/BITS.
	printObjectEnumsBits(obj)

	// Column table for tables and rows.
	printObjectColumns(obj)

	// Provenance.
	printProvenance(obj.Name(), obj.Module(), obj.Type())

	// Group membership.
	if obj.Node() != nil {
		printGroupMembership(m, obj.Node())
	}

	// Diagnostics.
	printScopedDiagnostics(m, obj.Module(), obj.Span())
	printRelatedUnresolved(m, obj.Name())

	// Description / Reference.
	printDescriptionReference(obj.Description(), obj.Reference())
}

// inspectNotification prints full detail for a NOTIFICATION-TYPE or TRAP-TYPE.
func inspectNotification(m *mib.Mib, notif *mib.Notification) {
	fmt.Printf("Status:  %s\n", notif.Status().String())

	if ti := notif.TrapInfo(); ti != nil {
		fmt.Printf("Enterprise: %s\n", ti.Enterprise)
		fmt.Printf("TrapNumber: %d\n", ti.TrapNumber)
	}

	if objs := notif.Objects(); len(objs) > 0 {
		fmt.Println("\nObjects:")
		for _, obj := range objs {
			modName := ""
			if obj.Module() != nil {
				modName = obj.Module().Name() + "::"
			}
			fmt.Printf("  %s%s  %s\n", modName, obj.Name(), obj.OID())
		}
	}

	// Group membership.
	if notif.Node() != nil {
		printGroupMembership(m, notif.Node())
	}

	// Diagnostics.
	printScopedDiagnostics(m, notif.Module(), notif.Span())
	printRelatedUnresolved(m, notif.Name())

	printDescriptionReference(notif.Description(), notif.Reference())
}

// inspectGroup prints full detail for an OBJECT-GROUP or NOTIFICATION-GROUP.
func inspectGroup(m *mib.Mib, g *mib.Group) {
	fmt.Printf("Status:  %s\n", g.Status().String())
	if g.IsNotificationGroup() {
		fmt.Printf("Type:    notification-group\n")
	} else {
		fmt.Printf("Type:    object-group\n")
	}

	if members := g.Members(); len(members) > 0 {
		fmt.Println("\nMembers:")
		for _, nd := range members {
			modName := ""
			if nd.Module() != nil {
				modName = nd.Module().Name() + "::"
			}
			fmt.Printf("  %s%s  %s  %s\n", modName, nd.Name(), nd.OID(), nd.Kind())
		}
	}

	// Which compliances reference this group.
	printComplianceReferences(m, g.Name())

	// Diagnostics.
	printScopedDiagnostics(m, g.Module(), g.Span())
	printRelatedUnresolved(m, g.Name())

	printDescriptionReference(g.Description(), g.Reference())
}

// inspectCompliance prints full detail for a MODULE-COMPLIANCE.
func inspectCompliance(m *mib.Mib, c *mib.Compliance) {
	fmt.Printf("Status:  %s\n", c.Status().String())

	for _, cm := range c.Modules() {
		modName := cm.ModuleName
		if modName == "" {
			modName = "(this module)"
		}
		fmt.Printf("\nModule: %s\n", modName)
		if len(cm.MandatoryGroups) > 0 {
			fmt.Printf("  Mandatory groups: %s\n", strings.Join(cm.MandatoryGroups, ", "))
		}
		for _, cg := range cm.Groups {
			fmt.Printf("  Group: %s\n", cg.Group)
			if cg.Description != "" {
				fmt.Printf("    %s\n", normalizeDescription(cg.Description, 200))
			}
		}
		for _, co := range cm.Objects {
			fmt.Printf("  Object: %s\n", co.Object)
			if co.MinAccess != nil {
				fmt.Printf("    MIN-ACCESS: %s\n", co.MinAccess.String())
			}
			if co.Description != "" {
				fmt.Printf("    %s\n", normalizeDescription(co.Description, 200))
			}
		}
	}

	// Diagnostics.
	printScopedDiagnostics(m, c.Module(), c.Span())
	printRelatedUnresolved(m, c.Name())

	printDescriptionReference(c.Description(), c.Reference())
}

// inspectCapability prints full detail for an AGENT-CAPABILITIES.
func inspectCapability(m *mib.Mib, cap *mib.Capability) {
	fmt.Printf("Status:  %s\n", cap.Status().String())

	if cap.ProductRelease() != "" {
		fmt.Printf("Product: %s\n", cap.ProductRelease())
	}

	for _, sm := range cap.Supports() {
		fmt.Printf("\nSupports: %s\n", sm.ModuleName)
		if len(sm.Includes) > 0 {
			fmt.Printf("  Includes: %s\n", strings.Join(sm.Includes, ", "))
		}
		for i := range sm.ObjectVariations {
			ov := &sm.ObjectVariations[i]
			fmt.Printf("  Variation: %s\n", ov.Object)
			if ov.Access != nil {
				fmt.Printf("    ACCESS: %s\n", ov.Access.String())
			}
			if ov.Description != "" {
				fmt.Printf("    %s\n", normalizeDescription(ov.Description, 200))
			}
		}
		for _, nv := range sm.NotificationVariations {
			fmt.Printf("  Variation: %s\n", nv.Notification)
			if nv.Access != nil {
				fmt.Printf("    ACCESS: %s\n", nv.Access.String())
			}
			if nv.Description != "" {
				fmt.Printf("    %s\n", normalizeDescription(nv.Description, 200))
			}
		}
	}

	// Diagnostics.
	printScopedDiagnostics(m, cap.Module(), cap.Span())
	printRelatedUnresolved(m, cap.Name())

	printDescriptionReference(cap.Description(), cap.Reference())
}

// inspectBareNode prints detail for a node with no entity attachment.
func inspectBareNode(m *mib.Mib, node *mib.Node) {
	if node.IsObjectIdentity() {
		if s, ok := node.ObjectIdentityStatus(); ok {
			fmt.Printf("Status:  %s\n", s.String())
		}
		fmt.Println("Macro:   OBJECT-IDENTITY")
	}

	printScopedDiagnostics(m, node.Module(), node.Span())
	printRelatedUnresolved(m, node.Name())

	printDescriptionReference(node.Description(), node.Reference())
}

// inspectStandaloneType prints detail for a type not attached to a node.
func inspectStandaloneType(m *mib.Mib, typ *mib.Type) {
	fmt.Printf("Name:    %s\n", typ.Name())
	if typ.Module() != nil {
		fmt.Printf("Module:  %s\n", typ.Module().Name())
	}
	fmt.Printf("Kind:    type\n")
	if typ.Status() != 0 {
		fmt.Printf("Status:  %s\n", typ.Status().String())
	}
	if typ.IsTextualConvention() {
		fmt.Printf("Macro:   TEXTUAL-CONVENTION\n")
	}
	fmt.Printf("Base:    %s\n", typ.EffectiveBase().String())

	printTypeChain(typ)

	printScopedDiagnostics(m, typ.Module(), typ.Span())
	printRelatedUnresolved(m, typ.Name())

	printDescriptionReference(typ.Description(), typ.Reference())
}

// printTypeChain walks the type parent chain showing each level's constraints.
func printTypeChain(typ *mib.Type) {
	fmt.Println("\nType chain:")
	depth := 0
	for cur := typ; cur != nil && depth < 100; cur, depth = cur.Parent(), depth+1 {
		name := cur.Name()
		if name == "" {
			name = "(inline)"
		}

		modName := ""
		if cur.Module() != nil {
			modName = cur.Module().Name()
		}

		// Build annotations.
		var tags []string
		if cur.IsTextualConvention() {
			tags = append(tags, "textual-convention")
		}
		if cur.Base() != 0 {
			tags = append(tags, "base: "+cur.Base().String())
		}

		tagStr := ""
		if len(tags) > 0 {
			tagStr = "  (" + strings.Join(tags, ", ") + ")"
		}

		if modName != "" {
			fmt.Printf("  %-28s %s%s\n", name, modName, tagStr)
		} else {
			fmt.Printf("  %s%s\n", name, tagStr)
		}

		// Constraints declared at this level.
		if h := cur.DisplayHint(); h != "" {
			fmt.Printf("    DISPLAY-HINT %q\n", h)
		}
		if sizes := cur.Sizes(); len(sizes) > 0 {
			fmt.Printf("    SIZE (%s)\n", formatRangeList(sizes))
		}
		if ranges := cur.Ranges(); len(ranges) > 0 {
			fmt.Printf("    RANGE (%s)\n", formatRangeList(ranges))
		}
		if enums := cur.Enums(); len(enums) > 0 {
			labels := make([]string, len(enums))
			for i, e := range enums {
				labels[i] = fmt.Sprintf("%s(%d)", e.Label, e.Value)
			}
			fmt.Printf("    VALUES: %s\n", strings.Join(labels, ", "))
		}
		if bits := cur.Bits(); len(bits) > 0 {
			labels := make([]string, len(bits))
			for i, b := range bits {
				labels[i] = fmt.Sprintf("%s(%d)", b.Label, b.Value)
			}
			fmt.Printf("    BITS: %s\n", strings.Join(labels, ", "))
		}
	}
}

// formatRangeList formats a slice of Range as "min..max | min..max".
func formatRangeList(ranges []mib.Range) string {
	parts := make([]string, len(ranges))
	for i, r := range ranges {
		parts[i] = r.String()
	}
	return strings.Join(parts, " | ")
}

// printProvenance shows where the object and its type were defined/imported.
func printProvenance(name string, mod *mib.Module, typ *mib.Type) {
	fmt.Println("\nProvenance:")
	if mod != nil {
		fmt.Printf("  %-24s defined in %s\n", name, mod.Name())
	}

	if typ == nil {
		return
	}

	// Walk the type chain showing where each named type comes from.
	seen := make(map[string]bool)
	for cur := typ; cur != nil; cur = cur.Parent() {
		tName := cur.Name()
		if tName == "" || seen[tName] {
			continue
		}
		seen[tName] = true

		tMod := ""
		if cur.Module() != nil {
			tMod = cur.Module().Name()
		}

		// Check if this type was imported by the object's module.
		source := ""
		if mod != nil && tMod != "" && tMod != mod.Name() {
			if mod.ImportsSymbol(tName) {
				if srcMod := mod.ImportSource(tName); srcMod != nil {
					source = fmt.Sprintf("  imported from %s (via %s IMPORTS)", srcMod.Name(), mod.Name())
				} else {
					source = fmt.Sprintf("  imported from %s (via %s IMPORTS)", tMod, mod.Name())
				}
			} else {
				source = "  defined in " + tMod
			}
		} else if tMod != "" {
			source = "  defined in " + tMod
		}

		label := "Type " + tName
		if cur.IsTextualConvention() {
			label = "TC " + tName
		}
		fmt.Printf("  %-24s%s\n", label, source)
	}
}

// printGroupMembership finds and prints groups that include the given node.
func printGroupMembership(m *mib.Mib, node *mib.Node) {
	var groups []string
	for _, g := range m.Groups() {
		if !slices.Contains(g.Members(), node) {
			continue
		}
		modName := ""
		if g.Module() != nil {
			modName = g.Module().Name()
		}
		kind := "object-group"
		if g.IsNotificationGroup() {
			kind = "notification-group"
		}
		groups = append(groups, fmt.Sprintf("  %s  (%s, %s)", g.Name(), modName, kind))
	}
	if len(groups) > 0 {
		fmt.Println("\nGroup membership:")
		for _, g := range groups {
			fmt.Println(g)
		}
	}
}

// printComplianceReferences finds compliances that reference the given group name.
func printComplianceReferences(m *mib.Mib, groupName string) {
	var refs []string
	for _, c := range m.Compliances() {
		for _, cm := range c.Modules() {
			if slices.Contains(cm.MandatoryGroups, groupName) {
				modName := ""
				if c.Module() != nil {
					modName = c.Module().Name()
				}
				refs = append(refs, fmt.Sprintf("  %s  (%s, mandatory)", c.Name(), modName))
			}
			for _, cg := range cm.Groups {
				if cg.Group == groupName {
					modName := ""
					if c.Module() != nil {
						modName = c.Module().Name()
					}
					refs = append(refs, fmt.Sprintf("  %s  (%s, conditional)", c.Name(), modName))
				}
			}
		}
	}
	if len(refs) > 0 {
		fmt.Println("\nReferenced by compliances:")
		for _, r := range refs {
			fmt.Println(r)
		}
	}
}

// printScopedDiagnostics filters diagnostics to those within the entity's source span.
func printScopedDiagnostics(m *mib.Mib, mod *mib.Module, span mib.Span) {
	if mod == nil || span == (mib.Span{}) {
		return
	}

	startLine, _ := mod.LineCol(span.Start)
	endLine, _ := mod.LineCol(span.End)
	if startLine == 0 {
		return
	}

	moduleName := mod.Name()
	var scoped []mib.Diagnostic
	for _, d := range m.Diagnostics() {
		if d.Module != moduleName {
			continue
		}
		if d.Line >= startLine && d.Line <= endLine {
			scoped = append(scoped, d)
		}
	}

	if len(scoped) > 0 {
		fmt.Println("\nDiagnostics:")
		for _, d := range scoped {
			fmt.Printf("  [%s] %s: %s\n", d.Severity, d.Code, d.Message)
		}
	}
}

// printRelatedUnresolved shows unresolved references that mention the symbol.
func printRelatedUnresolved(m *mib.Mib, name string) {
	var related []mib.UnresolvedRef
	for _, u := range m.Unresolved() {
		if u.Symbol == name {
			related = append(related, u)
		}
	}
	if len(related) > 0 {
		fmt.Println("\nUnresolved references:")
		for _, u := range related {
			entry := fmt.Sprintf("  [%s] %s in %s", u.Kind, u.Symbol, u.Module)
			if u.Reason != "" {
				entry += ": " + u.Reason
			}
			fmt.Println(entry)
		}
	}
}

// printDescriptionReference prints description and reference in full.
func printDescriptionReference(desc, ref string) {
	if desc != "" {
		fmt.Printf("\nDescription:\n  %s\n", strings.Join(strings.Fields(desc), " "))
	}
	if ref != "" {
		fmt.Printf("\nReference:\n  %s\n", strings.Join(strings.Fields(ref), " "))
	}
}
