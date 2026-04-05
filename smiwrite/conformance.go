package smiwrite

import (
	"fmt"

	"github.com/golangsnmp/gomib/mib"
)

// emitGroup writes an OBJECT-GROUP or NOTIFICATION-GROUP definition.
// The macro name and members keyword differ between the two types:
// OBJECT-GROUP uses OBJECTS, NOTIFICATION-GROUP uses NOTIFICATIONS.
func (e *emitter) emitGroup(grp *mib.Group) error {
	macro := "OBJECT-GROUP"
	keyword := "OBJECTS"
	if grp.IsNotificationGroup() {
		macro = "NOTIFICATION-GROUP"
		keyword = "NOTIFICATIONS"
	}

	if _, err := fmt.Fprintf(e.w, "%s %s\n", grp.Name(), macro); err != nil {
		return err
	}

	if members := grp.Members(); len(members) > 0 {
		names := make([]string, len(members))
		for i, nd := range members {
			names[i] = nd.Name()
		}
		if err := e.writeInlineList("    ", keyword, names); err != nil {
			return err
		}
	}

	if _, err := fmt.Fprintf(e.w, "    STATUS %s\n", formatStatus(grp.Status())); err != nil {
		return err
	}

	if err := e.writeDescription(grp.Description()); err != nil {
		return err
	}

	nd := grp.Node()
	_, err := fmt.Fprintf(e.w, "    ::= %s\n", formatOIDAssignment(nd))
	return err
}

// emitModuleCompliance writes a MODULE-COMPLIANCE definition.
func (e *emitter) emitModuleCompliance(comp *mib.Compliance) error {
	if _, err := fmt.Fprintf(e.w, "%s MODULE-COMPLIANCE\n", comp.Name()); err != nil {
		return err
	}

	if _, err := fmt.Fprintf(e.w, "    STATUS %s\n", formatStatus(comp.Status())); err != nil {
		return err
	}

	if err := e.writeDescription(comp.Description()); err != nil {
		return err
	}

	for _, cm := range comp.Modules() {
		modName := cm.ModuleName
		if modName == "" {
			modName = "-- this module"
		}
		if _, err := fmt.Fprintf(e.w, "    MODULE %s\n", modName); err != nil {
			return err
		}

		if err := e.writeNameRefList("        ", "MANDATORY-GROUPS", cm.MandatoryGroups); err != nil {
			return err
		}

		for _, cg := range cm.Groups {
			if _, err := fmt.Fprintf(e.w, "        GROUP %s\n", cg.Group); err != nil {
				return err
			}
			if err := e.writeOptionalDescription("        ", cg.Description); err != nil {
				return err
			}
		}

		for _, co := range cm.Objects {
			if _, err := fmt.Fprintf(e.w, "        OBJECT %s\n", co.Object); err != nil {
				return err
			}
			if co.Syntax != nil {
				if _, err := fmt.Fprintf(e.w, "            SYNTAX %s\n", formatSyntaxConstraints(co.Syntax)); err != nil {
					return err
				}
			}
			if co.WriteSyntax != nil {
				if _, err := fmt.Fprintf(e.w, "            WRITE-SYNTAX %s\n", formatSyntaxConstraints(co.WriteSyntax)); err != nil {
					return err
				}
			}
			if co.MinAccess != nil {
				if _, err := fmt.Fprintf(e.w, "            MIN-ACCESS %s\n", formatAccess(*co.MinAccess)); err != nil {
					return err
				}
			}
			if err := e.writeOptionalDescription("            ", co.Description); err != nil {
				return err
			}
		}
	}

	nd := comp.Node()
	_, err := fmt.Fprintf(e.w, "    ::= %s\n", formatOIDAssignment(nd))
	return err
}

// emitAgentCapabilities writes an AGENT-CAPABILITIES definition.
func (e *emitter) emitAgentCapabilities(cap *mib.Capability) error {
	if _, err := fmt.Fprintf(e.w, "%s AGENT-CAPABILITIES\n", cap.Name()); err != nil {
		return err
	}

	if pr := cap.ProductRelease(); pr != "" {
		if _, err := fmt.Fprintf(e.w, "    PRODUCT-RELEASE\n        %s\n", formatDescription(pr)); err != nil {
			return err
		}
	}

	if _, err := fmt.Fprintf(e.w, "    STATUS %s\n", formatStatus(cap.Status())); err != nil {
		return err
	}

	if err := e.writeDescription(cap.Description()); err != nil {
		return err
	}

	for _, sm := range cap.Supports() {
		if _, err := fmt.Fprintf(e.w, "    SUPPORTS %s\n", sm.ModuleName); err != nil {
			return err
		}
		if err := e.writeNameRefList("        ", "INCLUDES", sm.Includes); err != nil {
			return err
		}

		for i := range sm.ObjectVariations {
			ov := &sm.ObjectVariations[i]
			if _, err := fmt.Fprintf(e.w, "        VARIATION %s\n", ov.Object); err != nil {
				return err
			}
			if ov.Syntax != nil {
				if _, err := fmt.Fprintf(e.w, "            SYNTAX %s\n", formatSyntaxConstraints(ov.Syntax)); err != nil {
					return err
				}
			}
			if ov.WriteSyntax != nil {
				if _, err := fmt.Fprintf(e.w, "            WRITE-SYNTAX %s\n", formatSyntaxConstraints(ov.WriteSyntax)); err != nil {
					return err
				}
			}
			if ov.Access != nil {
				if _, err := fmt.Fprintf(e.w, "            ACCESS %s\n", formatAccess(*ov.Access)); err != nil {
					return err
				}
			}
			if err := e.writeNameRefList("            ", "CREATION-REQUIRES", ov.CreationRequires); err != nil {
				return err
			}
			if !ov.DefVal.IsZero() {
				if _, err := fmt.Fprintf(e.w, "            %s\n", formatDefVal(ov.DefVal)); err != nil {
					return err
				}
			}
			if err := e.writeOptionalDescription("            ", ov.Description); err != nil {
				return err
			}
		}

		for _, nv := range sm.NotificationVariations {
			if _, err := fmt.Fprintf(e.w, "        VARIATION %s\n", nv.Notification); err != nil {
				return err
			}
			if nv.Access != nil {
				if _, err := fmt.Fprintf(e.w, "            ACCESS %s\n", formatAccess(*nv.Access)); err != nil {
					return err
				}
			}
			if err := e.writeOptionalDescription("            ", nv.Description); err != nil {
				return err
			}
		}
	}

	nd := cap.Node()
	_, err := fmt.Fprintf(e.w, "    ::= %s\n", formatOIDAssignment(nd))
	return err
}

// formatSyntaxConstraints formats a SyntaxConstraints from compliance/capabilities.
func formatSyntaxConstraints(sc *mib.SyntaxConstraints) string {
	if sc == nil {
		return ""
	}

	if len(sc.Bits) > 0 {
		return "BITS " + formatBits(sc.Bits)
	}
	if len(sc.Enums) > 0 {
		return "INTEGER " + formatEnums(sc.Enums)
	}

	typeName := ""
	if sc.Type != nil {
		if sc.Type.Name() != "" {
			typeName = sc.Type.Name()
		} else {
			typeName = baseTypeSyntax(sc.Type.EffectiveBase())
		}
	}

	if len(sc.Sizes) > 0 {
		return typeName + " " + formatSizes(sc.Sizes)
	}
	if len(sc.Ranges) > 0 {
		return typeName + " " + formatRanges(sc.Ranges)
	}

	return typeName
}
