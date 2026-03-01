package smiwrite

import (
	"cmp"
	"fmt"
	"io"
	"slices"

	"github.com/golangsnmp/gomib/mib"
)

// emitter writes canonical SMIv2 text for a single module.
type emitter struct {
	w   io.Writer
	m   *mib.Mib
	cfg config
	it  *importTracker
}

func newEmitter(w io.Writer, m *mib.Mib, cfg config) *emitter {
	return &emitter{w: w, m: m, cfg: cfg}
}

// definitions holds the categorized entities for a module, sorted for output.
type definitions struct {
	moduleIdentity *mib.Node
	textualConvs   []*mib.Type
	oidAssignments []*mib.Node
	objects        []*mib.Object
	notifications  []*mib.Notification
	groups         []*mib.Group
	compliances    []*mib.Compliance
	capabilities   []*mib.Capability
	rows           []*mib.Object // for SEQUENCE reconstruction
}

// collectDefinitions gathers and sorts all entities for a module.
func collectDefinitions(mod *mib.Module, cfg config) definitions {
	var defs definitions

	// Module identity node
	if mod.OID() != nil {
		for _, nd := range mod.Nodes() {
			if nd.OID() != nil && nd.OID().Equal(mod.OID()) {
				defs.moduleIdentity = nd
				break
			}
		}
	}

	// Textual conventions and types
	for _, t := range mod.Types() {
		if t.IsTextualConvention() || (t.Name() != "" && !t.Module().IsBase()) {
			defs.textualConvs = append(defs.textualConvs, t)
		}
	}
	slices.SortFunc(defs.textualConvs, func(a, b *mib.Type) int {
		return cmp.Compare(a.Name(), b.Name())
	})

	// OID assignments (OBJECT-IDENTITY and plain value assignments)
	for _, nd := range mod.Nodes() {
		if nd == defs.moduleIdentity {
			continue
		}
		if nd.Kind() == mib.KindNode && nd.Object() == nil && nd.Notification() == nil &&
			nd.Group() == nil && nd.Compliance() == nil && nd.Capability() == nil {
			defs.oidAssignments = append(defs.oidAssignments, nd)
		}
	}
	sortNodesByOID(defs.oidAssignments)

	// Objects
	defs.objects = mod.Objects()
	slices.SortFunc(defs.objects, func(a, b *mib.Object) int {
		return compareOIDs(a.OID(), b.OID())
	})

	// Notifications
	defs.notifications = mod.Notifications()
	slices.SortFunc(defs.notifications, func(a, b *mib.Notification) int {
		return compareOIDs(a.OID(), b.OID())
	})

	if cfg.conformance {
		defs.groups = mod.Groups()
		slices.SortFunc(defs.groups, func(a, b *mib.Group) int {
			return compareOIDs(a.OID(), b.OID())
		})

		defs.compliances = mod.Compliances()
		slices.SortFunc(defs.compliances, func(a, b *mib.Compliance) int {
			return compareOIDs(a.OID(), b.OID())
		})

		defs.capabilities = mod.Capabilities()
		slices.SortFunc(defs.capabilities, func(a, b *mib.Capability) int {
			return compareOIDs(a.OID(), b.OID())
		})
	}

	if cfg.sequences {
		defs.rows = mod.Rows()
	}

	return defs
}

func (e *emitter) emitModule(mod *mib.Module) error {
	defs := collectDefinitions(mod, e.cfg)

	// Build import tracker and collect all referenced symbols
	e.it = newImportTracker(mod.Name())
	e.collectImports(&defs)

	// Module header
	if _, err := fmt.Fprintf(e.w, "%s DEFINITIONS ::= BEGIN\n", mod.Name()); err != nil {
		return err
	}

	// IMPORTS
	if err := e.it.writeImports(e.w); err != nil {
		return err
	}

	// MODULE-IDENTITY
	if defs.moduleIdentity != nil {
		if _, err := fmt.Fprintln(e.w); err != nil {
			return err
		}
		if err := e.emitModuleIdentity(mod, defs.moduleIdentity); err != nil {
			return err
		}
	}

	// Textual conventions
	for _, t := range defs.textualConvs {
		if _, err := fmt.Fprintln(e.w); err != nil {
			return err
		}
		if err := e.emitTextualConvention(t); err != nil {
			return err
		}
	}

	// OID assignments
	for _, nd := range defs.oidAssignments {
		if _, err := fmt.Fprintln(e.w); err != nil {
			return err
		}
		if err := e.emitOIDAssignment(nd); err != nil {
			return err
		}
	}

	// OBJECT-TYPEs
	for _, obj := range defs.objects {
		if _, err := fmt.Fprintln(e.w); err != nil {
			return err
		}
		if err := e.emitObjectType(obj); err != nil {
			return err
		}
	}

	// NOTIFICATION-TYPEs
	for _, notif := range defs.notifications {
		if _, err := fmt.Fprintln(e.w); err != nil {
			return err
		}
		if err := e.emitNotificationType(notif); err != nil {
			return err
		}
	}

	// Conformance constructs
	if e.cfg.conformance {
		for _, grp := range defs.groups {
			if _, err := fmt.Fprintln(e.w); err != nil {
				return err
			}
			if err := e.emitGroup(grp); err != nil {
				return err
			}
		}

		for _, comp := range defs.compliances {
			if _, err := fmt.Fprintln(e.w); err != nil {
				return err
			}
			if err := e.emitModuleCompliance(comp); err != nil {
				return err
			}
		}

		for _, cap := range defs.capabilities {
			if _, err := fmt.Fprintln(e.w); err != nil {
				return err
			}
			if err := e.emitAgentCapabilities(cap); err != nil {
				return err
			}
		}
	}

	// Optional SEQUENCE types
	if e.cfg.sequences {
		for _, row := range defs.rows {
			if _, err := fmt.Fprintln(e.w); err != nil {
				return err
			}
			if err := e.emitSequence(row); err != nil {
				return err
			}
		}
	}

	// END
	_, err := fmt.Fprintf(e.w, "\nEND\n")
	return err
}

func sortNodesByOID(nodes []*mib.Node) {
	slices.SortFunc(nodes, func(a, b *mib.Node) int {
		return compareOIDs(a.OID(), b.OID())
	})
}

func compareOIDs(a, b mib.OID) int {
	if a == nil && b == nil {
		return 0
	}
	if a == nil {
		return -1
	}
	if b == nil {
		return 1
	}
	return a.Compare(b)
}
