package smiwrite

import (
	"fmt"
	"io"
	"slices"
	"strings"

	"github.com/golangsnmp/gomib/mib"
)

// importTracker collects symbols referenced by the emitted definitions
// and generates a minimal IMPORTS section.
type importTracker struct {
	moduleName string
	// module -> set of symbol names
	imports map[string]map[string]struct{}
}

func newImportTracker(moduleName string) *importTracker {
	return &importTracker{
		moduleName: moduleName,
		imports:    make(map[string]map[string]struct{}),
	}
}

// addSymbol records that the given symbol from the given module is needed.
func (it *importTracker) addSymbol(module, symbol string) {
	if module == "" || module == it.moduleName {
		return
	}
	if it.imports[module] == nil {
		it.imports[module] = make(map[string]struct{})
	}
	it.imports[module][symbol] = struct{}{}
}

// addMacro adds a macro keyword import (OBJECT-TYPE, MODULE-IDENTITY, etc.)
func (it *importTracker) addMacro(macro string) {
	mod := macroModule(macro)
	if mod != "" {
		it.addSymbol(mod, macro)
	}
}

// addType records an import for a type if it comes from another module.
// ASN.1 built-in type keywords are never imported.
func (it *importTracker) addType(t *mib.Type) {
	if t == nil {
		return
	}
	name := t.Name()
	if name == "" || isBuiltinKeyword(name) {
		return
	}
	mod := t.Module()
	if mod == nil {
		return
	}
	it.addSymbol(mod.Name(), name)
}

// isBuiltinKeyword returns true for ASN.1 built-in type keywords
// that should never appear in IMPORTS.
func isBuiltinKeyword(name string) bool {
	switch name {
	case "INTEGER", "OCTET STRING", "OBJECT IDENTIFIER", "BITS",
		"NULL", "SEQUENCE", "SEQUENCE OF":
		return true
	}
	return false
}

// addNodeParent records an import for a node's parent if from another module.
func (it *importTracker) addNodeParent(nd *mib.Node) {
	if nd == nil {
		return
	}
	parent := nd.Parent()
	if parent == nil || parent.Name() == "" {
		return
	}
	parentMod := parent.Module()
	if parentMod == nil {
		return
	}
	it.addSymbol(parentMod.Name(), parent.Name())
}

// writeImports writes the IMPORTS section.
func (it *importTracker) writeImports(w io.Writer) error {
	if len(it.imports) == 0 {
		return nil
	}

	if _, err := fmt.Fprintln(w); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w, "IMPORTS"); err != nil {
		return err
	}

	// Sort modules for deterministic output
	modules := make([]string, 0, len(it.imports))
	for mod := range it.imports {
		modules = append(modules, mod)
	}
	slices.Sort(modules)

	for i, mod := range modules {
		symbols := make([]string, 0, len(it.imports[mod]))
		for sym := range it.imports[mod] {
			symbols = append(symbols, sym)
		}
		slices.Sort(symbols)

		if _, err := fmt.Fprintf(w, "    %s\n", strings.Join(symbols, ", ")); err != nil {
			return err
		}
		// No commas between FROM groups; semicolon after last one
		terminator := ""
		if i == len(modules)-1 {
			terminator = ";"
		}
		if _, err := fmt.Fprintf(w, "        FROM %s%s\n", mod, terminator); err != nil {
			return err
		}
	}

	return nil
}

// collectImports does a pass over all definitions to determine needed imports.
func (e *emitter) collectImports(defs *definitions) {
	// MODULE-IDENTITY
	if defs.moduleIdentity != nil {
		e.it.addMacro("MODULE-IDENTITY")
		e.it.addNodeParent(defs.moduleIdentity)
	}

	// Textual conventions
	for _, t := range defs.textualConvs {
		e.it.addMacro("TEXTUAL-CONVENTION")
		// Import the effective base type if it comes from another module
		collectTypeSyntaxImports(e.it, t)
	}

	// OID assignments
	for _, nd := range defs.oidAssignments {
		e.it.addNodeParent(nd)
	}

	// Objects
	for _, obj := range defs.objects {
		e.it.addMacro("OBJECT-TYPE")
		e.it.addNodeParent(obj.Node())
		collectObjectSyntaxImports(e.it, obj)
		// INDEX references
		for _, idx := range obj.Index() {
			if idx.Object != nil && idx.Object.Module() != nil {
				e.it.addSymbol(idx.Object.Module().Name(), idx.Object.Name())
			}
		}
		// AUGMENTS reference
		if aug := obj.Augments(); aug != nil && aug.Module() != nil {
			e.it.addSymbol(aug.Module().Name(), aug.Name())
		}
	}

	// Notifications
	for _, notif := range defs.notifications {
		e.it.addMacro("NOTIFICATION-TYPE")
		e.it.addNodeParent(notif.Node())
		for _, obj := range notif.Objects() {
			if obj.Module() != nil {
				e.it.addSymbol(obj.Module().Name(), obj.Name())
			}
		}
	}

	// Conformance
	if e.cfg.conformance {
		for _, grp := range defs.groups {
			if grp.IsNotificationGroup() {
				e.it.addMacro("NOTIFICATION-GROUP")
			} else {
				e.it.addMacro("OBJECT-GROUP")
			}
			e.it.addNodeParent(grp.Node())
			for _, member := range grp.Members() {
				if member.Module() != nil {
					e.it.addSymbol(member.Module().Name(), member.Name())
				}
			}
		}

		for _, comp := range defs.compliances {
			e.it.addMacro("MODULE-COMPLIANCE")
			e.it.addNodeParent(comp.Node())
			for _, cm := range comp.Modules() {
				for _, gref := range cm.MandatoryGroups {
					// Try to find the group's module
					if nd := e.m.Node(gref.Name); nd != nil && nd.Module() != nil {
						e.it.addSymbol(nd.Module().Name(), gref.Name)
					}
				}
				for _, cg := range cm.Groups {
					if nd := e.m.Node(cg.Group); nd != nil && nd.Module() != nil {
						e.it.addSymbol(nd.Module().Name(), cg.Group)
					}
				}
				for _, co := range cm.Objects {
					if nd := e.m.Node(co.Object); nd != nil && nd.Module() != nil {
						e.it.addSymbol(nd.Module().Name(), co.Object)
					}
				}
			}
		}

		for _, cap := range defs.capabilities {
			e.it.addMacro("AGENT-CAPABILITIES")
			e.it.addNodeParent(cap.Node())
		}
	}
}

// collectObjectSyntaxImports adds imports for types referenced in an object's SYNTAX.
func collectObjectSyntaxImports(it *importTracker, obj *mib.Object) {
	t := obj.Type()
	if t == nil {
		return
	}

	// Table/Row: SEQUENCE type is local, no import needed
	if obj.Kind() == mib.KindTable {
		return
	}
	if obj.Kind() == mib.KindRow {
		return
	}

	// Named type: import the type name from its module
	if t.Name() != "" {
		it.addType(t)
		return
	}

	// Anonymous type: import parent type name if from another module
	if parent := t.Parent(); parent != nil && parent.Name() != "" {
		it.addType(parent)
		return
	}

	// Base type: some base types need imports (Integer32, Counter32, etc.)
	collectBaseTypeImport(it, t.EffectiveBase())
}

// collectTypeSyntaxImports adds imports for the type named in a TC's SYNTAX.
// TCs with enums or bits use the INTEGER/BITS keywords directly, so no type
// import is needed.
func collectTypeSyntaxImports(it *importTracker, t *mib.Type) {
	if len(t.Enums()) > 0 || len(t.Bits()) > 0 {
		return
	}
	if parent := t.Parent(); parent != nil && parent.Name() != "" {
		it.addType(parent)
		return
	}
	collectBaseTypeImport(it, t.EffectiveBase())
}

// collectBaseTypeImport imports base SMI types that aren't built-in keywords.
func collectBaseTypeImport(it *importTracker, base mib.BaseType) {
	switch base {
	case mib.BaseInteger32:
		it.addSymbol("SNMPv2-SMI", "Integer32")
	case mib.BaseUnsigned32:
		it.addSymbol("SNMPv2-SMI", "Unsigned32")
	case mib.BaseCounter32:
		it.addSymbol("SNMPv2-SMI", "Counter32")
	case mib.BaseCounter64:
		it.addSymbol("SNMPv2-SMI", "Counter64")
	case mib.BaseGauge32:
		it.addSymbol("SNMPv2-SMI", "Gauge32")
	case mib.BaseTimeTicks:
		it.addSymbol("SNMPv2-SMI", "TimeTicks")
	case mib.BaseIpAddress:
		it.addSymbol("SNMPv2-SMI", "IpAddress")
	case mib.BaseOpaque:
		it.addSymbol("SNMPv2-SMI", "Opaque")
	}
	// OCTET STRING, OBJECT IDENTIFIER, BITS, INTEGER are built-in keywords
}

// macroModule returns the source module for a macro keyword.
func macroModule(macro string) string {
	switch macro {
	case "MODULE-IDENTITY", "OBJECT-TYPE", "NOTIFICATION-TYPE",
		"OBJECT-IDENTITY":
		return "SNMPv2-SMI"
	case "TEXTUAL-CONVENTION":
		return "SNMPv2-TC"
	case "MODULE-COMPLIANCE", "OBJECT-GROUP", "NOTIFICATION-GROUP",
		"AGENT-CAPABILITIES":
		return "SNMPv2-CONF"
	}
	return ""
}
