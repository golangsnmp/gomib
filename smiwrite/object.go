package smiwrite

import (
	"fmt"
	"strings"

	"github.com/golangsnmp/gomib/mib"
)

// emitObjectType writes an OBJECT-TYPE definition.
func (e *emitter) emitObjectType(obj *mib.Object) error {
	if _, err := fmt.Fprintf(e.w, "%s OBJECT-TYPE\n", obj.Name()); err != nil {
		return err
	}

	syntax := formatObjectSyntax(obj)
	if _, err := fmt.Fprintf(e.w, "    SYNTAX %s\n", syntax); err != nil {
		return err
	}

	if units := obj.Units(); units != "" {
		if _, err := fmt.Fprintf(e.w, "    UNITS %q\n", units); err != nil {
			return err
		}
	}

	if _, err := fmt.Fprintf(e.w, "    MAX-ACCESS %s\n", formatAccess(obj.Access())); err != nil {
		return err
	}

	if _, err := fmt.Fprintf(e.w, "    STATUS %s\n", formatStatus(obj.Status())); err != nil {
		return err
	}

	if err := e.writeDescription(obj.Description()); err != nil {
		return err
	}

	if ref := obj.Reference(); ref != "" {
		if _, err := fmt.Fprintf(e.w, "    REFERENCE\n        %s\n", formatDescription(ref)); err != nil {
			return err
		}
	}

	// INDEX or AUGMENTS clause (for rows)
	if obj.Kind() == mib.KindRow {
		if aug := obj.Augments(); aug != nil {
			if _, err := fmt.Fprintf(e.w, "    AUGMENTS { %s }\n", aug.Name()); err != nil {
				return err
			}
		} else if idx := obj.Index(); len(idx) > 0 {
			if err := e.writeIndex(idx); err != nil {
				return err
			}
		}
	}

	// DEFVAL
	if dv := obj.DefaultValue(); !dv.IsZero() {
		if _, err := fmt.Fprintf(e.w, "    %s\n", formatDefVal(dv)); err != nil {
			return err
		}
	}

	nd := obj.Node()
	_, err := fmt.Fprintf(e.w, "    ::= %s\n", formatOIDAssignment(nd))
	return err
}

// writeIndex writes the INDEX clause.
func (e *emitter) writeIndex(entries []mib.IndexEntry) error {
	if _, err := fmt.Fprint(e.w, "    INDEX { "); err != nil {
		return err
	}
	for i, idx := range entries {
		if i > 0 {
			if _, err := fmt.Fprint(e.w, ", "); err != nil {
				return err
			}
		}
		if idx.Implied {
			if _, err := fmt.Fprint(e.w, "IMPLIED "); err != nil {
				return err
			}
		}
		if idx.Object != nil {
			if _, err := fmt.Fprint(e.w, idx.Object.Name()); err != nil {
				return err
			}
		} else if idx.TypeName != "" {
			if _, err := fmt.Fprint(e.w, idx.TypeName); err != nil {
				return err
			}
		}
	}
	_, err := fmt.Fprintln(e.w, " }")
	return err
}

// formatObjectSyntax formats the SYNTAX clause for an OBJECT-TYPE.
//
// Logic:
//   - KindTable: SEQUENCE OF EntryTypeName
//   - KindRow: EntryTypeName (derived from row name convention)
//   - Named type (obj.Type().Name() != ""): emit the type name
//   - Anonymous type with named parent: parent name + inline constraints
//   - Anonymous type with no named parent: base type keyword + inline constraints
//   - If has enums: INTEGER { name(val), ... }
//   - If has bits: BITS { name(pos), ... }
func formatObjectSyntax(obj *mib.Object) string {
	// Table: SEQUENCE OF <SequenceName>
	if obj.Kind() == mib.KindTable {
		if entry := obj.Entry(); entry != nil {
			return "SEQUENCE OF " + sequenceTypeName(entry.Name())
		}
		return "SEQUENCE OF " + sequenceTypeName(obj.Name()+"Entry")
	}

	// Row: emit the sequence type name
	if obj.Kind() == mib.KindRow {
		return sequenceTypeName(obj.Name())
	}

	t := obj.Type()
	if t == nil {
		return "INTEGER" // fallback
	}

	// The resolver stores effective enums/bits/ranges/sizes on the object.
	// Enums and bits are always passed for inline formatting. Ranges and sizes
	// are only passed when they differ from the type's own constraints, meaning
	// they came from an inline subtype constraint on the OBJECT-TYPE rather
	// than being inherited from the type definition.
	var inlineRanges, inlineSizes []mib.Range
	if objRanges := obj.EffectiveRanges(); !rangesEqual(objRanges, t.EffectiveRanges()) {
		inlineRanges = objRanges
	}
	if objSizes := obj.EffectiveSizes(); !rangesEqual(objSizes, t.EffectiveSizes()) {
		inlineSizes = objSizes
	}

	return formatObjectTypeSyntax(t, obj.EffectiveEnums(), obj.EffectiveBits(), inlineRanges, inlineSizes)
}

// formatObjectTypeSyntax formats the SYNTAX clause for an OBJECT-TYPE.
//
// The resolver stores effective enums/bits on the object rather than the type,
// so they must be passed explicitly. Inline ranges/sizes are constraints from
// the OBJECT-TYPE's SYNTAX clause that differ from the type's own constraints.
// For types without enums/bits, uses the type's own direct constraints
// (sizes/ranges), falling back to the type name or base keyword.
func formatObjectTypeSyntax(t *mib.Type, enums, bits []mib.NamedValue, inlineRanges, inlineSizes []mib.Range) string {
	// Named type: preserve TC/type references, but not built-in keywords
	// like INTEGER or BITS that may have inline enums/bits on the object.
	if name := t.Name(); name != "" && !isBuiltinKeyword(name) {
		if len(inlineSizes) > 0 {
			return name + " " + formatSizes(inlineSizes)
		}
		if len(inlineRanges) > 0 {
			return name + " " + formatRanges(inlineRanges)
		}
		return name
	}

	// Inline BITS definition
	if len(bits) > 0 {
		return "BITS " + formatBits(bits)
	}

	// Inline enum definition
	if len(enums) > 0 {
		return "INTEGER " + formatEnums(enums)
	}

	// Anonymous type: determine prefix from parent or base keyword
	var prefix string
	if parent := t.Parent(); parent != nil && parent.Name() != "" {
		prefix = parent.Name()
	} else {
		prefix = baseTypeSyntax(t.EffectiveBase())
	}

	// Add the anonymous type's own direct constraints
	if sizes := t.Sizes(); len(sizes) > 0 {
		return prefix + " " + formatSizes(sizes)
	}
	if ranges := t.Ranges(); len(ranges) > 0 {
		return prefix + " " + formatRanges(ranges)
	}

	return prefix
}

// rangesEqual reports whether two Range slices are identical.
func rangesEqual(a, b []mib.Range) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// sequenceTypeName converts a row entry name to the conventional SEQUENCE type name.
// The convention is to capitalize the first letter: ifEntry -> IfEntry.
func sequenceTypeName(name string) string {
	if name == "" {
		return ""
	}
	return strings.ToUpper(name[:1]) + name[1:]
}
