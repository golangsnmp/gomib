package smiwrite

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/golangsnmp/gomib/mib"
)

// formatStatus normalizes SMIv1 status values to SMIv2 equivalents.
// mandatory -> current, optional -> deprecated (per RFC 3584 s2.1.3(6)).
func formatStatus(s mib.Status) string {
	switch s {
	case mib.StatusMandatory:
		return "current"
	case mib.StatusOptional:
		return "deprecated"
	default:
		return s.String()
	}
}

// formatAccess normalizes access values to SMIv2 MAX-ACCESS equivalents.
// write-only -> read-write (per RFC 3584).
func formatAccess(a mib.Access) string {
	switch a {
	case mib.AccessWriteOnly:
		return "read-write"
	default:
		return a.String()
	}
}

// baseTypeSyntax returns the SMIv2 keyword for a base type.
func baseTypeSyntax(b mib.BaseType) string {
	switch b {
	case mib.BaseInteger32:
		return "Integer32"
	case mib.BaseUnsigned32:
		return "Unsigned32"
	case mib.BaseCounter32:
		return "Counter32"
	case mib.BaseCounter64:
		return "Counter64"
	case mib.BaseGauge32:
		return "Gauge32"
	case mib.BaseTimeTicks:
		return "TimeTicks"
	case mib.BaseIpAddress:
		return "IpAddress"
	case mib.BaseOctetString:
		return "OCTET STRING"
	case mib.BaseObjectIdentifier:
		return "OBJECT IDENTIFIER"
	case mib.BaseBits:
		return "BITS"
	case mib.BaseOpaque:
		return "Opaque"
	default:
		return b.String()
	}
}

// formatEnums formats named values as an inline enum definition.
// Output: "{ name1(val1), name2(val2) }"
func formatEnums(values []mib.NamedValue) string {
	if len(values) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("{\n")
	for i, nv := range values {
		b.WriteString("        ")
		b.WriteString(nv.Label)
		b.WriteByte('(')
		b.WriteString(strconv.FormatInt(nv.Value, 10))
		b.WriteByte(')')
		if i < len(values)-1 {
			b.WriteByte(',')
		}
		b.WriteByte('\n')
	}
	b.WriteString("    }")
	return b.String()
}

// formatBits formats BITS named values.
// Output: "{ name1(pos1), name2(pos2) }"
func formatBits(values []mib.NamedValue) string {
	return formatEnums(values) // same format
}

// formatRanges formats range constraints.
// Output: "(min..max | min..max)"
func formatRanges(ranges []mib.Range) string {
	if len(ranges) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteByte('(')
	for i, r := range ranges {
		if i > 0 {
			b.WriteString(" | ")
		}
		b.WriteString(r.String())
	}
	b.WriteByte(')')
	return b.String()
}

// formatSizes formats SIZE constraints.
// Output: "(SIZE (min..max | min..max))"
func formatSizes(sizes []mib.Range) string {
	if len(sizes) == 0 {
		return ""
	}
	return "(SIZE " + formatRanges(sizes) + ")"
}

// formatDescription formats a DESCRIPTION clause with proper quoting.
func formatDescription(desc string) string {
	return `"` + strings.ReplaceAll(desc, `"`, `\"`) + `"`
}

// writeDescription writes a mandatory DESCRIPTION clause if descriptions are enabled.
func (e *emitter) writeDescription(desc string) error {
	if !e.cfg.descriptions {
		return nil
	}
	_, err := fmt.Fprintf(e.w, "    DESCRIPTION\n        %s\n", formatDescription(desc))
	return err
}

// writeOptionalDescription writes an optional DESCRIPTION subclause.
// Unlike writeDescription, this skips empty descriptions entirely since the
// clause is not required by the macro grammar.
func (e *emitter) writeOptionalDescription(indent, desc string) error {
	if !e.cfg.descriptions || desc == "" {
		return nil
	}
	_, err := fmt.Fprintf(e.w, "%sDESCRIPTION\n%s    %s\n", indent, indent, formatDescription(desc))
	return err
}

// writeNameRefList writes a NameRef list as a comma-separated keyword clause.
func (e *emitter) writeNameRefList(indent, keyword string, refs []mib.NameRef) error {
	return e.writeInlineList(indent, keyword, mib.NameRefNames(refs))
}

// writeInlineList writes a comma-separated list of names inside braces.
// Example output: "    OBJECTS { foo, bar, baz }\n"
func (e *emitter) writeInlineList(indent, keyword string, names []string) error {
	if len(names) == 0 {
		return nil
	}
	if _, err := fmt.Fprintf(e.w, "%s%s { ", indent, keyword); err != nil {
		return err
	}
	for i, name := range names {
		if i > 0 {
			if _, err := fmt.Fprint(e.w, ", "); err != nil {
				return err
			}
		}
		if _, err := fmt.Fprint(e.w, name); err != nil {
			return err
		}
	}
	_, err := fmt.Fprintln(e.w, " }")
	return err
}

// formatOIDAssignment formats an OID value using parent-relative syntax.
// Output: "{ parentName arc }" or "{ numeric-parent arc }" as fallback.
func formatOIDAssignment(nd *mib.Node) string {
	if nd == nil {
		return ""
	}
	parent := nd.Parent()
	if parent == nil {
		return fmt.Sprintf("{ %d }", nd.Arc())
	}
	if parent.Name() != "" {
		return fmt.Sprintf("{ %s %d }", parent.Name(), nd.Arc())
	}
	// Fallback to numeric parent OID
	parentOID := parent.OID()
	if parentOID == nil {
		return fmt.Sprintf("{ %d }", nd.Arc())
	}
	return fmt.Sprintf("{ %s %d }", parentOID.String(), nd.Arc())
}

// formatDefVal formats a DEFVAL clause from the raw syntax.
func formatDefVal(dv mib.DefVal) string {
	if dv.IsZero() {
		return ""
	}
	return fmt.Sprintf("DEFVAL { %s }", dv.Raw())
}
