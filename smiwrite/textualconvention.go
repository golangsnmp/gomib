package smiwrite

import (
	"fmt"
	"strings"

	"github.com/golangsnmp/gomib/mib"
)

// emitTextualConvention writes a TEXTUAL-CONVENTION definition.
// Bare type assignments are emitted as TC with STATUS current and DESCRIPTION "".
func (e *emitter) emitTextualConvention(t *mib.Type) error {
	if _, err := fmt.Fprintf(e.w, "%s ::= TEXTUAL-CONVENTION\n", t.Name()); err != nil {
		return err
	}

	if hint := t.DisplayHint(); hint != "" {
		if _, err := fmt.Fprintf(e.w, "    DISPLAY-HINT %q\n", hint); err != nil {
			return err
		}
	}

	status := t.Status()
	if status == 0 {
		status = mib.StatusCurrent
	}
	if _, err := fmt.Fprintf(e.w, "    STATUS %s\n", formatStatus(status)); err != nil {
		return err
	}

	if err := e.writeDescription(t.Description()); err != nil {
		return err
	}

	if ref := t.Reference(); ref != "" {
		if _, err := fmt.Fprintf(e.w, "    REFERENCE\n        %s\n", formatDescription(ref)); err != nil {
			return err
		}
	}

	syntax := formatTypeSyntax(t)
	_, err := fmt.Fprintf(e.w, "    SYNTAX %s\n", syntax)
	return err
}

// formatTypeSyntax formats the SYNTAX clause for a TEXTUAL-CONVENTION.
// Uses EffectiveBase() keyword plus direct constraints on this type.
func formatTypeSyntax(t *mib.Type) string {
	var b strings.Builder

	enums := t.Enums()
	bits := t.Bits()

	if len(bits) > 0 {
		b.WriteString("BITS ")
		b.WriteString(formatBits(bits))
		return b.String()
	}

	if len(enums) > 0 {
		b.WriteString("INTEGER ")
		b.WriteString(formatEnums(enums))
		return b.String()
	}

	base := t.EffectiveBase()
	b.WriteString(baseTypeSyntax(base))

	if sizes := t.Sizes(); len(sizes) > 0 {
		b.WriteString(" ")
		b.WriteString(formatSizes(sizes))
	} else if ranges := t.Ranges(); len(ranges) > 0 {
		b.WriteString(" ")
		b.WriteString(formatRanges(ranges))
	}

	return b.String()
}
