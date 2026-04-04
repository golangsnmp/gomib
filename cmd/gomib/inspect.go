package main

import (
	"flag"
	"fmt"
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
	fs.Usage = func() { fmt.Fprint(c.stderr, inspectUsage) }

	modules := addModuleFlag(fs)
	strict, permissive := addStrictnessFlags(fs)
	help := addHelpFlag(fs)

	if err := fs.Parse(args); err != nil {
		return exitError
	}

	if c.checkHelp(help, inspectUsage) {
		return exitOK
	}

	if code, failed := c.validateStrictnessFlags(*strict, *permissive); failed {
		return code
	}

	positional := fs.Args()
	if len(positional) == 0 {
		c.printError("no query specified")
		fmt.Fprint(c.stderr, inspectUsage)
		return exitError
	}
	if len(positional) > 1 {
		c.printError("too many arguments (use -m for module selection)")
		fmt.Fprint(c.stderr, inspectUsage)
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
		c.printError("failed to load: %v", err)
		return exitError
	}

	// Try node resolution first, then fall back to type lookup.
	node := m.Resolve(query)
	if node != nil {
		c.inspectNode(m, node)
		return exitOK
	}

	// Try as a type name (types don't always have nodes).
	typName := query
	if modName, name, ok := strings.Cut(query, "::"); ok {
		mod := m.Module(modName)
		if mod != nil {
			if typ := mod.Type(name); typ != nil {
				c.inspectStandaloneType(m, typ)
				return exitOK
			}
		}
	} else if typ := m.Type(typName); typ != nil {
		c.inspectStandaloneType(m, typ)
		return exitOK
	}

	c.printError("not found: %s", query)
	return exitIssue
}

func (c *cli) inspectNode(m *mib.Mib, node *mib.Node) {
	c.printIdentity(node)

	switch {
	case node.Object() != nil:
		c.inspectObject(m, node.Object())
	case node.Notification() != nil:
		c.inspectNotification(m, node.Notification())
	case node.Group() != nil:
		c.inspectGroup(m, node.Group())
	case node.Compliance() != nil:
		c.inspectCompliance(m, node.Compliance())
	case node.Capability() != nil:
		c.inspectCapability(m, node.Capability())
	default:
		c.inspectBareNode(m, node)
	}
}

func (c *cli) printIdentity(node *mib.Node) {
	label := node.Name()
	if label == "" {
		label = fmt.Sprintf("(%d)", node.Arc())
	}
	fmt.Fprintf(c.stdout, "Name:    %s\n", label)
	if node.Module() != nil {
		fmt.Fprintf(c.stdout, "Module:  %s\n", node.Module().Name())
	}
	fmt.Fprintf(c.stdout, "OID:     %s\n", node.OID().String())
	fmt.Fprintf(c.stdout, "Kind:    %s\n", node.Kind().String())
}

// inspectObject prints full detail for an OBJECT-TYPE.
func (c *cli) inspectObject(m *mib.Mib, obj *mib.Object) {
	fmt.Fprintf(c.stdout, "Status:  %s\n", obj.Status().String())
	fmt.Fprintf(c.stdout, "Access:  %s\n", obj.Access().String())

	if obj.Units() != "" {
		fmt.Fprintf(c.stdout, "Units:   %s\n", obj.Units())
	}

	if dv := obj.DefaultValue(); !dv.IsZero() {
		fmt.Fprintf(c.stdout, "DefVal:  %s\n", dv.String())
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
			fmt.Fprintf(c.stdout, "Type:    %s (%s)\n", typeName, base)
		} else {
			fmt.Fprintf(c.stdout, "Type:    %s\n", typeName)
		}
	}

	// Index / augments.
	c.printObjectIndexAugments(obj)

	// Column context.
	c.printObjectColumnContext(obj)

	// Type chain.
	if obj.Type() != nil {
		c.printTypeChain(obj.Type())
	}

	// Enum/BITS.
	c.printObjectEnumsBits(obj)

	// Column table for tables and rows.
	c.printObjectColumns(obj)

	// Provenance.
	c.printProvenance(obj.Name(), obj.Module(), obj.Type())

	// Group membership.
	if obj.Node() != nil {
		c.printGroupMembership(m, obj.Node())
	}

	// Diagnostics.
	c.printScopedDiagnostics(m, obj.Module(), obj.Span())
	c.printRelatedUnresolved(m, obj.Name())

	// Description / Reference.
	c.printDescriptionReference(obj.Description(), obj.Reference())
}

// inspectNotification prints full detail for a NOTIFICATION-TYPE or TRAP-TYPE.
func (c *cli) inspectNotification(m *mib.Mib, notif *mib.Notification) {
	fmt.Fprintf(c.stdout, "Status:  %s\n", notif.Status().String())

	if ti := notif.TrapInfo(); ti != nil {
		fmt.Fprintf(c.stdout, "Enterprise: %s\n", ti.Enterprise)
		fmt.Fprintf(c.stdout, "TrapNumber: %d\n", ti.TrapNumber)
	}

	if objs := notif.Objects(); len(objs) > 0 {
		fmt.Fprintln(c.stdout, "\nObjects:")
		for _, obj := range objs {
			modName := ""
			if obj.Module() != nil {
				modName = obj.Module().Name() + "::"
			}
			fmt.Fprintf(c.stdout, "  %s%s  %s\n", modName, obj.Name(), obj.OID())
		}
	}

	// Group membership.
	if notif.Node() != nil {
		c.printGroupMembership(m, notif.Node())
	}

	// Diagnostics.
	c.printScopedDiagnostics(m, notif.Module(), notif.Span())
	c.printRelatedUnresolved(m, notif.Name())

	c.printDescriptionReference(notif.Description(), notif.Reference())
}

// inspectGroup prints full detail for an OBJECT-GROUP or NOTIFICATION-GROUP.
func (c *cli) inspectGroup(m *mib.Mib, g *mib.Group) {
	fmt.Fprintf(c.stdout, "Status:  %s\n", g.Status().String())
	if g.IsNotificationGroup() {
		fmt.Fprintf(c.stdout, "Type:    notification-group\n")
	} else {
		fmt.Fprintf(c.stdout, "Type:    object-group\n")
	}

	if members := g.Members(); len(members) > 0 {
		fmt.Fprintln(c.stdout, "\nMembers:")
		for _, nd := range members {
			modName := ""
			if nd.Module() != nil {
				modName = nd.Module().Name() + "::"
			}
			fmt.Fprintf(c.stdout, "  %s%s  %s  %s\n", modName, nd.Name(), nd.OID(), nd.Kind())
		}
	}

	// Which compliances reference this group.
	c.printComplianceReferences(m, g.Name())

	// Diagnostics.
	c.printScopedDiagnostics(m, g.Module(), g.Span())
	c.printRelatedUnresolved(m, g.Name())

	c.printDescriptionReference(g.Description(), g.Reference())
}

// inspectCompliance prints full detail for a MODULE-COMPLIANCE.
func (c *cli) inspectCompliance(m *mib.Mib, comp *mib.Compliance) {
	fmt.Fprintf(c.stdout, "Status:  %s\n", comp.Status().String())

	for _, cm := range comp.Modules() {
		modName := cm.ModuleName
		if modName == "" {
			modName = "(this module)"
		}
		fmt.Fprintf(c.stdout, "\nModule: %s\n", modName)
		if len(cm.MandatoryGroups) > 0 {
			fmt.Fprintf(c.stdout, "  Mandatory groups: %s\n", strings.Join(cm.MandatoryGroups, ", "))
		}
		for _, cg := range cm.Groups {
			fmt.Fprintf(c.stdout, "  Group: %s\n", cg.Group)
			if cg.Description != "" {
				fmt.Fprintf(c.stdout, "    %s\n", normalizeDescription(cg.Description, 200))
			}
		}
		for _, co := range cm.Objects {
			fmt.Fprintf(c.stdout, "  Object: %s\n", co.Object)
			if co.MinAccess != nil {
				fmt.Fprintf(c.stdout, "    MIN-ACCESS: %s\n", co.MinAccess.String())
			}
			if co.Description != "" {
				fmt.Fprintf(c.stdout, "    %s\n", normalizeDescription(co.Description, 200))
			}
		}
	}

	// Diagnostics.
	c.printScopedDiagnostics(m, comp.Module(), comp.Span())
	c.printRelatedUnresolved(m, comp.Name())

	c.printDescriptionReference(comp.Description(), comp.Reference())
}

// inspectCapability prints full detail for an AGENT-CAPABILITIES.
func (c *cli) inspectCapability(m *mib.Mib, cap *mib.Capability) {
	fmt.Fprintf(c.stdout, "Status:  %s\n", cap.Status().String())

	if cap.ProductRelease() != "" {
		fmt.Fprintf(c.stdout, "Product: %s\n", cap.ProductRelease())
	}

	for _, sm := range cap.Supports() {
		fmt.Fprintf(c.stdout, "\nSupports: %s\n", sm.ModuleName)
		if len(sm.Includes) > 0 {
			fmt.Fprintf(c.stdout, "  Includes: %s\n", strings.Join(sm.Includes, ", "))
		}
		for i := range sm.ObjectVariations {
			ov := &sm.ObjectVariations[i]
			fmt.Fprintf(c.stdout, "  Variation: %s\n", ov.Object)
			if ov.Access != nil {
				fmt.Fprintf(c.stdout, "    ACCESS: %s\n", ov.Access.String())
			}
			if ov.Description != "" {
				fmt.Fprintf(c.stdout, "    %s\n", normalizeDescription(ov.Description, 200))
			}
		}
		for _, nv := range sm.NotificationVariations {
			fmt.Fprintf(c.stdout, "  Variation: %s\n", nv.Notification)
			if nv.Access != nil {
				fmt.Fprintf(c.stdout, "    ACCESS: %s\n", nv.Access.String())
			}
			if nv.Description != "" {
				fmt.Fprintf(c.stdout, "    %s\n", normalizeDescription(nv.Description, 200))
			}
		}
	}

	// Diagnostics.
	c.printScopedDiagnostics(m, cap.Module(), cap.Span())
	c.printRelatedUnresolved(m, cap.Name())

	c.printDescriptionReference(cap.Description(), cap.Reference())
}

// inspectBareNode prints detail for a node with no entity attachment.
func (c *cli) inspectBareNode(m *mib.Mib, node *mib.Node) {
	if node.IsObjectIdentity() {
		if s, ok := node.ObjectIdentityStatus(); ok {
			fmt.Fprintf(c.stdout, "Status:  %s\n", s.String())
		}
		fmt.Fprintln(c.stdout, "Macro:   OBJECT-IDENTITY")
	}

	c.printScopedDiagnostics(m, node.Module(), node.Span())
	c.printRelatedUnresolved(m, node.Name())

	c.printDescriptionReference(node.Description(), node.Reference())
}

// inspectStandaloneType prints detail for a type not attached to a node.
func (c *cli) inspectStandaloneType(m *mib.Mib, typ *mib.Type) {
	fmt.Fprintf(c.stdout, "Name:    %s\n", typ.Name())
	if typ.Module() != nil {
		fmt.Fprintf(c.stdout, "Module:  %s\n", typ.Module().Name())
	}
	fmt.Fprintf(c.stdout, "Kind:    type\n")
	if typ.Status() != 0 {
		fmt.Fprintf(c.stdout, "Status:  %s\n", typ.Status().String())
	}
	if typ.IsTextualConvention() {
		fmt.Fprintf(c.stdout, "Macro:   TEXTUAL-CONVENTION\n")
	}
	fmt.Fprintf(c.stdout, "Base:    %s\n", typ.EffectiveBase().String())

	c.printTypeChain(typ)

	c.printScopedDiagnostics(m, typ.Module(), typ.Span())
	c.printRelatedUnresolved(m, typ.Name())

	c.printDescriptionReference(typ.Description(), typ.Reference())
}

// printTypeChain walks the type parent chain showing each level's constraints.
func (c *cli) printTypeChain(typ *mib.Type) {
	fmt.Fprintln(c.stdout, "\nType chain:")
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
			fmt.Fprintf(c.stdout, "  %-28s %s%s\n", name, modName, tagStr)
		} else {
			fmt.Fprintf(c.stdout, "  %s%s\n", name, tagStr)
		}

		// Constraints declared at this level.
		if h := cur.DisplayHint(); h != "" {
			fmt.Fprintf(c.stdout, "    DISPLAY-HINT %q\n", h)
		}
		if sizes := cur.Sizes(); len(sizes) > 0 {
			fmt.Fprintf(c.stdout, "    SIZE (%s)\n", formatRangeList(sizes))
		}
		if ranges := cur.Ranges(); len(ranges) > 0 {
			fmt.Fprintf(c.stdout, "    RANGE (%s)\n", formatRangeList(ranges))
		}
		if enums := cur.Enums(); len(enums) > 0 {
			labels := make([]string, len(enums))
			for i, e := range enums {
				labels[i] = fmt.Sprintf("%s(%d)", e.Label, e.Value)
			}
			fmt.Fprintf(c.stdout, "    VALUES: %s\n", strings.Join(labels, ", "))
		}
		if bits := cur.Bits(); len(bits) > 0 {
			labels := make([]string, len(bits))
			for i, b := range bits {
				labels[i] = fmt.Sprintf("%s(%d)", b.Label, b.Value)
			}
			fmt.Fprintf(c.stdout, "    BITS: %s\n", strings.Join(labels, ", "))
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
func (c *cli) printProvenance(name string, mod *mib.Module, typ *mib.Type) {
	fmt.Fprintln(c.stdout, "\nProvenance:")
	if mod != nil {
		fmt.Fprintf(c.stdout, "  %-24s defined in %s\n", name, mod.Name())
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
		fmt.Fprintf(c.stdout, "  %-24s%s\n", label, source)
	}
}

// printGroupMembership finds and prints groups that include the given node.
func (c *cli) printGroupMembership(m *mib.Mib, node *mib.Node) {
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
		fmt.Fprintln(c.stdout, "\nGroup membership:")
		for _, g := range groups {
			fmt.Fprintln(c.stdout, g)
		}
	}
}

// printComplianceReferences finds compliances that reference the given group name.
func (c *cli) printComplianceReferences(m *mib.Mib, groupName string) {
	var refs []string
	for _, comp := range m.Compliances() {
		for _, cm := range comp.Modules() {
			if slices.Contains(cm.MandatoryGroups, groupName) {
				modName := ""
				if comp.Module() != nil {
					modName = comp.Module().Name()
				}
				refs = append(refs, fmt.Sprintf("  %s  (%s, mandatory)", comp.Name(), modName))
			}
			for _, cg := range cm.Groups {
				if cg.Group == groupName {
					modName := ""
					if comp.Module() != nil {
						modName = comp.Module().Name()
					}
					refs = append(refs, fmt.Sprintf("  %s  (%s, conditional)", comp.Name(), modName))
				}
			}
		}
	}
	if len(refs) > 0 {
		fmt.Fprintln(c.stdout, "\nReferenced by compliances:")
		for _, r := range refs {
			fmt.Fprintln(c.stdout, r)
		}
	}
}

// printScopedDiagnostics filters diagnostics to those within the entity's source span.
func (c *cli) printScopedDiagnostics(m *mib.Mib, mod *mib.Module, span mib.Span) {
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
		fmt.Fprintln(c.stdout, "\nDiagnostics:")
		for _, d := range scoped {
			fmt.Fprintf(c.stdout, "  [%s] %s: %s\n", d.Severity, d.Code, d.Message)
		}
	}
}

// printRelatedUnresolved shows unresolved references that mention the symbol.
func (c *cli) printRelatedUnresolved(m *mib.Mib, name string) {
	var related []mib.UnresolvedRef
	for _, u := range m.Unresolved() {
		if u.Symbol == name {
			related = append(related, u)
		}
	}
	if len(related) > 0 {
		fmt.Fprintln(c.stdout, "\nUnresolved references:")
		for _, u := range related {
			entry := fmt.Sprintf("  [%s] %s in %s", u.Kind, u.Symbol, u.Module)
			if u.Reason != "" {
				entry += ": " + u.Reason
			}
			fmt.Fprintln(c.stdout, entry)
		}
	}
}

// printDescriptionReference prints description and reference in full.
func (c *cli) printDescriptionReference(desc, ref string) {
	if desc != "" {
		fmt.Fprintf(c.stdout, "\nDescription:\n  %s\n", strings.Join(strings.Fields(desc), " "))
	}
	if ref != "" {
		fmt.Fprintf(c.stdout, "\nReference:\n  %s\n", strings.Join(strings.Fields(ref), " "))
	}
}
