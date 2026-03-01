package smiwrite

import (
	"fmt"

	"github.com/golangsnmp/gomib/mib"
)

// emitModuleIdentity writes the MODULE-IDENTITY definition.
func (e *emitter) emitModuleIdentity(mod *mib.Module, nd *mib.Node) error {
	name := nd.Name()
	if name == "" {
		name = mod.Name()
	}

	if _, err := fmt.Fprintf(e.w, "%s MODULE-IDENTITY\n", name); err != nil {
		return err
	}

	if lu := mod.LastUpdated(); lu != "" {
		if _, err := fmt.Fprintf(e.w, "    LAST-UPDATED %q\n", lu); err != nil {
			return err
		}
	}

	if org := mod.Organization(); org != "" {
		if _, err := fmt.Fprintf(e.w, "    ORGANIZATION\n        %s\n", formatDescription(org)); err != nil {
			return err
		}
	}

	if ci := mod.ContactInfo(); ci != "" {
		if _, err := fmt.Fprintf(e.w, "    CONTACT-INFO\n        %s\n", formatDescription(ci)); err != nil {
			return err
		}
	}

	if err := e.writeDescription(mod.Description()); err != nil {
		return err
	}

	for _, rev := range mod.Revisions() {
		if _, err := fmt.Fprintf(e.w, "    REVISION %q\n", rev.Date); err != nil {
			return err
		}
		if err := e.writeOptionalDescription("    ", rev.Description); err != nil {
			return err
		}
	}

	_, err := fmt.Fprintf(e.w, "    ::= %s\n", formatOIDAssignment(nd))
	return err
}

// emitOIDAssignment writes a plain OID value assignment (OBJECT IDENTIFIER).
func (e *emitter) emitOIDAssignment(nd *mib.Node) error {
	name := nd.Name()
	if name == "" {
		return nil
	}
	_, err := fmt.Fprintf(e.w, "%s OBJECT IDENTIFIER ::= %s\n", name, formatOIDAssignment(nd))
	return err
}
