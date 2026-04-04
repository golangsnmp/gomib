package mib

// Style and lint-like checks for the resolver semantic analysis phase.
//
// These checks flag code-quality issues, naming convention violations,
// deprecated constructs, and MIB authoring best practices. A MIB that
// fails only style checks is structurally valid but could be improved.

import (
	"fmt"
	"strings"

	"github.com/golangsnmp/gomib/internal/module"
	"github.com/golangsnmp/gomib/internal/types"
)

// checkIntegerMisuse flags bare INTEGER usage in SMIv2 modules. RFC 2578
// section 7.1.1 says INTEGER and Integer32 are "indistinguishable" for
// non-enum use, and RFC 3584 section 2.1.1 item (2) says bare INTEGER
// SHOULD become Integer32. INTEGER with named-number enumerations is the
// only correct use of the INTEGER keyword in SMIv2.
func checkIntegerMisuse(ctx *resolverContext) {
	for _, mod := range ctx.modules {
		if mod.Language != types.LanguageSMIv2 || module.IsBaseModule(mod.Name) {
			continue
		}
		for _, def := range mod.Definitions {
			switch d := def.(type) {
			case *module.ObjectType:
				if isIntegerKeywordSyntax(d.Syntax) {
					ctx.EmitDiagnostic(types.DiagIntegerInSMIv2,
						mod, d.Span,
						fmt.Sprintf("%q: use Integer32 instead of INTEGER in SMIv2", d.Name))
				}
			case *module.TypeDef:
				if isIntegerKeywordSyntax(d.Syntax) {
					ctx.EmitDiagnostic(types.DiagIntegerInSMIv2,
						mod, d.Span,
						fmt.Sprintf("%q: use Integer32 instead of INTEGER in SMIv2", d.Name))
				}
			}
		}
	}
}

// isIntegerKeywordSyntax returns true if the syntax uses the INTEGER keyword
// without named-number enumerations (bare INTEGER or INTEGER with range).
func isIntegerKeywordSyntax(syntax module.TypeSyntax) bool {
	switch s := syntax.(type) {
	case *module.TypeSyntaxTypeRef:
		return s.Name == "INTEGER"
	case *module.TypeSyntaxConstrained:
		return isIntegerKeywordSyntax(s.Base)
	default:
		return false
	}
}

// checkDescriptionMissing flags SMIv2 OBJECT-TYPE definitions that lack a
// DESCRIPTION clause. OBJECT-TYPE is the only definition type where
// DESCRIPTION is grammatically optional (also TRAP-TYPE, but that is SMIv1).
// For all other SMIv2 types, DESCRIPTION is a required clause in the grammar,
// and empty descriptions are caught by the empty-description lowering check.
func checkDescriptionMissing(ctx *resolverContext, objRefs []objectTypeRef) {
	for _, ref := range objRefs {
		if ref.mod.Language != types.LanguageSMIv2 || module.IsBaseModule(ref.mod.Name) {
			continue
		}
		if !ref.obj.HasDescription {
			ctx.EmitDiagnostic(types.DiagDescriptionMissing,
				ref.mod, ref.obj.Span,
				fmt.Sprintf("%q: OBJECT-TYPE should have a DESCRIPTION clause", ref.obj.Name))
		}
	}
}

// checkTextualConventionNested flags TEXTUAL-CONVENTIONs whose parent type
// is itself a TEXTUAL-CONVENTION. RFC 2579 section 3.5 specifies that a TC's
// SYNTAX clause should reference a base type, not another TC. Many real MIBs
// violate this, so it is reported at style severity.
func checkTextualConventionNested(ctx *resolverContext) {
	for _, mod := range ctx.modules {
		if module.IsBaseModule(mod.Name) {
			continue
		}
		for _, def := range mod.Definitions {
			td, ok := def.(*module.TypeDef)
			if !ok || !td.IsTextualConvention {
				continue
			}
			resolved, ok := ctx.LookupTypeForModule(mod, td.Name)
			if !ok {
				continue
			}
			parent := resolved.Parent()
			if parent != nil && parent.IsTextualConvention() {
				ctx.EmitDiagnostic(types.DiagTCNested,
					mod, td.Span,
					fmt.Sprintf("%q: textual convention derived from textual convention %q", td.Name, parent.Name()))
			}
		}
	}
}

// checkTypeAssignmentSMIv2 flags plain type assignments in SMIv2 modules.
// RFC 2579 recommends using TEXTUAL-CONVENTION instead of bare type
// assignments in SMIv2.
func checkTypeAssignmentSMIv2(ctx *resolverContext) {
	for _, mod := range ctx.modules {
		if mod.Language != types.LanguageSMIv2 || module.IsBaseModule(mod.Name) {
			continue
		}
		for _, def := range mod.Definitions {
			td, ok := def.(*module.TypeDef)
			if !ok || td.IsTextualConvention {
				continue
			}
			// Skip SEQUENCE type assignments (used for table rows, not real types).
			if _, isSeq := td.Syntax.(*module.TypeSyntaxSequence); isSeq {
				continue
			}
			ctx.EmitDiagnostic(types.DiagTypeAssignmentSMIv2,
				mod, td.Span,
				fmt.Sprintf("%q: type assignment in SMIv2 should be a TEXTUAL-CONVENTION", td.Name))
		}
	}
}

// checkTableRowNaming validates table/row naming conventions:
//   - Table names should end in "Table"
//   - Row names should end in "Entry"
//   - Row name prefix should match table name prefix
func checkTableRowNaming(ctx *resolverContext, objRefs []objectTypeRef) {
	for _, ref := range objRefs {
		if module.IsBaseModule(ref.mod.Name) {
			continue
		}
		node, ok := ctx.LookupNodeForModule(ref.mod, ref.obj.Name)
		if !ok {
			continue
		}
		switch node.Kind() {
		case KindTable:
			if !strings.HasSuffix(ref.obj.Name, "Table") {
				ctx.EmitDiagnostic(types.DiagTableNameTable,
					ref.mod, ref.obj.Span,
					fmt.Sprintf("%q: table name should end in \"Table\"", ref.obj.Name))
			}
		case KindRow:
			if !strings.HasSuffix(ref.obj.Name, "Entry") {
				ctx.EmitDiagnostic(types.DiagRowNameEntry,
					ref.mod, ref.obj.Span,
					fmt.Sprintf("%q: row name should end in \"Entry\"", ref.obj.Name))
			}
			// Check that row name prefix matches table name prefix.
			if node.Parent() != nil && node.Parent().Kind() == KindTable {
				tableName := node.Parent().Name()
				tablePrefix := strings.TrimSuffix(tableName, "Table")
				rowPrefix := strings.TrimSuffix(ref.obj.Name, "Entry")
				if tablePrefix != "" && rowPrefix != "" && tablePrefix != rowPrefix {
					ctx.EmitDiagnostic(types.DiagRowNameTableName,
						ref.mod, ref.obj.Span,
						fmt.Sprintf("%q: row prefix %q does not match table prefix %q", ref.obj.Name, rowPrefix, tablePrefix))
				}
			}
		}
	}
}

// forEachDefinitionSyntax calls fn for each ObjectType and TypeDef definition
// across all non-base modules. The callback receives the unwrapped syntax
// (TypeSyntaxConstrained layers stripped), definition name, module, and span.
func forEachDefinitionSyntax(ctx *resolverContext, fn func(module.TypeSyntax, string, *module.Module, types.Span)) {
	for _, mod := range ctx.modules {
		if module.IsBaseModule(mod.Name) {
			continue
		}
		for _, def := range mod.Definitions {
			switch d := def.(type) {
			case *module.ObjectType:
				fn(unwrapConstrainedSyntax(d.Syntax), d.Name, mod, d.Span)
			case *module.TypeDef:
				fn(unwrapConstrainedSyntax(d.Syntax), d.Name, mod, d.Span)
			}
		}
	}
}

// unwrapConstrainedSyntax strips TypeSyntaxConstrained wrappers to reach the
// underlying enum, bits, or type reference syntax.
func unwrapConstrainedSyntax(syntax module.TypeSyntax) module.TypeSyntax {
	for {
		c, ok := syntax.(*module.TypeSyntaxConstrained)
		if !ok {
			return syntax
		}
		syntax = c.Base
	}
}

// checkNamedNumberOrdering flags enumerations and BITS where named values
// are not in ascending order by value. This is a style recommendation per
// common MIB authoring practice.
func checkNamedNumberOrdering(ctx *resolverContext) {
	forEachDefinitionSyntax(ctx, func(syntax module.TypeSyntax, name string, mod *module.Module, span types.Span) {
		switch s := syntax.(type) {
		case *module.TypeSyntaxIntegerEnum:
			for i := 1; i < len(s.NamedNumbers); i++ {
				if s.NamedNumbers[i].Value < s.NamedNumbers[i-1].Value {
					ctx.EmitDiagnostic(types.DiagNamedNumbersAscending,
						mod, span,
						fmt.Sprintf("%q: named numbers not in ascending order (%s(%d) before %s(%d))",
							name,
							s.NamedNumbers[i-1].Name, s.NamedNumbers[i-1].Value,
							s.NamedNumbers[i].Name, s.NamedNumbers[i].Value))
					return
				}
			}
		case *module.TypeSyntaxBits:
			for i := 1; i < len(s.NamedBits); i++ {
				if s.NamedBits[i].Position < s.NamedBits[i-1].Position {
					ctx.EmitDiagnostic(types.DiagNamedNumbersAscending,
						mod, span,
						fmt.Sprintf("%q: BITS positions not in ascending order (%s(%d) before %s(%d))",
							name,
							s.NamedBits[i-1].Name, s.NamedBits[i-1].Position,
							s.NamedBits[i].Name, s.NamedBits[i].Position))
					return
				}
			}
		}
	})
}

// checkHyphenInLabel flags named numbers and BITS labels that contain hyphens
// in SMIv2 modules. RFC 2578 section 7.1.4 says named values must not use
// hyphens in SMIv2 (they are lowercase identifiers without hyphens).
func checkHyphenInLabel(ctx *resolverContext) {
	forEachDefinitionSyntax(ctx, func(syntax module.TypeSyntax, name string, mod *module.Module, span types.Span) {
		if mod.Language != types.LanguageSMIv2 {
			return
		}
		switch s := syntax.(type) {
		case *module.TypeSyntaxIntegerEnum:
			for _, nn := range s.NamedNumbers {
				if strings.Contains(nn.Name, "-") {
					ctx.EmitDiagnostic(types.DiagHyphenInLabel,
						mod, span,
						fmt.Sprintf("%q: named number %q contains a hyphen", name, nn.Name))
				}
			}
		case *module.TypeSyntaxBits:
			for _, nb := range s.NamedBits {
				if strings.Contains(nb.Name, "-") {
					ctx.EmitDiagnostic(types.DiagHyphenInLabel,
						mod, span,
						fmt.Sprintf("%q: BITS label %q contains a hyphen", name, nb.Name))
				}
			}
		}
	})
}

// checkOpaqueSMIv2 flags OBJECT-TYPE definitions that use the Opaque base type
// in SMIv2 modules. RFC 2578 section 7.1.3 says Opaque is provided solely for
// backward-compatibility and should not be used for new definitions.
func checkOpaqueSMIv2(ctx *resolverContext, objRefs []objectTypeRef) {
	for _, ref := range objRefs {
		if ref.mod.Language != types.LanguageSMIv2 || module.IsBaseModule(ref.mod.Name) {
			continue
		}
		resolved := ctx.lookupObjectInModuleScope(ref.mod, ref.obj.Name)
		if resolved == nil {
			continue
		}
		t := resolved.Type()
		if t == nil {
			continue
		}
		if t.EffectiveBase() == BaseOpaque {
			ctx.EmitDiagnostic(types.DiagOpaqueSMIv2,
				ref.mod, ref.obj.Span,
				fmt.Sprintf("%q: Opaque type should not be used in SMIv2", ref.obj.Name))
		}
	}
}

// checkFormatHints validates DISPLAY-HINT usage on textual conventions.
// It combines two checks:
//   - DiagInvalidFormat: format string is syntactically invalid for the base
//     type, or DISPLAY-HINT is present on an inapplicable base type (RFC 2579
//     section 3.1).
//   - DiagTypeWithoutFormat: TC for OCTET STRING or integer-based type has no
//     DISPLAY-HINT (RFC 2579 section 3.1 recommends one).
func checkFormatHints(ctx *resolverContext) {
	for _, mod := range ctx.modules {
		if module.IsBaseModule(mod.Name) {
			continue
		}
		for _, def := range mod.Definitions {
			td, ok := def.(*module.TypeDef)
			if !ok || !td.IsTextualConvention {
				continue
			}
			resolved, ok := ctx.LookupTypeForModule(mod, td.Name)
			if !ok {
				continue
			}
			base := resolved.EffectiveBase()

			if td.DisplayHint != "" {
				var valid bool
				switch base {
				case BaseInteger32, BaseUnsigned32, BaseGauge32, BaseTimeTicks:
					valid = IsValidIntegerHint(td.DisplayHint)
				case BaseOctetString, BaseOpaque:
					valid = IsValidOctetStringHint(td.DisplayHint)
				default:
					// DISPLAY-HINT is not applicable to this basetype.
					valid = false
				}
				if !valid {
					ctx.EmitDiagnostic(types.DiagInvalidFormat,
						mod, td.Span,
						fmt.Sprintf("%q: invalid DISPLAY-HINT %q for base type %s",
							td.Name, td.DisplayHint, base))
				}
			} else if resolved.EffectiveDisplayHint() == "" {
				switch base {
				case BaseOctetString, BaseInteger32, BaseUnsigned32, BaseGauge32:
					ctx.EmitDiagnostic(types.DiagTypeWithoutFormat,
						mod, td.Span,
						fmt.Sprintf("%q: textual convention without DISPLAY-HINT", td.Name))
				}
			}
		}
	}
}

// checkTypeUnreferenced flags type definitions that are never referenced by
// any OBJECT-TYPE SYNTAX, other type definition, or compliance refinement.
// Only checks within the loaded module set, not external consumers.
func checkTypeUnreferenced(ctx *resolverContext) {
	// Build set of referenced type names per module.
	referenced := make(map[*module.Module]map[string]struct{})
	markRef := func(mod *module.Module, name string) {
		refs := referenced[mod]
		if refs == nil {
			refs = make(map[string]struct{})
			referenced[mod] = refs
		}
		refs[name] = struct{}{}
	}

	// Collect type references from all definitions.
	for _, mod := range ctx.modules {
		for _, def := range mod.Definitions {
			switch d := def.(type) {
			case *module.ObjectType:
				collectSyntaxTypeRefs(d.Syntax, mod, markRef)
			case *module.TypeDef:
				collectSyntaxTypeRefs(d.Syntax, mod, markRef)
			case *module.ModuleCompliance:
				for _, cm := range d.Modules {
					for _, obj := range cm.Objects {
						collectSyntaxTypeRefs(obj.Syntax, mod, markRef)
						collectSyntaxTypeRefs(obj.WriteSyntax, mod, markRef)
					}
				}
			case *module.AgentCapabilities:
				for _, sup := range d.Supports {
					for _, v := range sup.Variations {
						collectSyntaxTypeRefs(v.Syntax, mod, markRef)
						collectSyntaxTypeRefs(v.WriteSyntax, mod, markRef)
					}
				}
			}
		}
	}

	// Also mark types that are exported via imports from other modules.
	for _, mod := range ctx.modules {
		for _, imp := range mod.Imports {
			// If another module imports a type name, mark it as used
			// in the source module.
			for _, srcMod := range ctx.moduleIndex[imp.Module] {
				markRef(srcMod, imp.Symbol)
			}
		}
	}

	// Check each type definition.
	for _, mod := range ctx.modules {
		if module.IsBaseModule(mod.Name) {
			continue
		}
		refs := referenced[mod]
		for _, def := range mod.Definitions {
			td, ok := def.(*module.TypeDef)
			if !ok {
				continue
			}
			// Skip SEQUENCE types (used internally for row definitions).
			if _, isSeq := td.Syntax.(*module.TypeSyntaxSequence); isSeq {
				continue
			}
			if refs != nil {
				if _, used := refs[td.Name]; used {
					continue
				}
			}
			ctx.EmitDiagnostic(types.DiagTypeUnreferenced,
				mod, td.Span,
				fmt.Sprintf("%q: type defined but never referenced", td.Name))
		}
	}
}

// isSequenceTypeDefDirect reports whether a definition is a SEQUENCE type
// assignment. These conventionally share names with row OBJECT-TYPEs
// differing only in initial case (e.g., FooEntry SEQUENCE vs fooEntry
// OBJECT-TYPE), so they are excluded from case-collision checks.
func isSequenceTypeDefDirect(def module.Definition) bool {
	td, ok := def.(*module.TypeDef)
	if !ok {
		return false
	}
	_, isSeq := td.Syntax.(*module.TypeSyntaxSequence)
	return isSeq
}

// checkIdentifierCaseMatch flags identifiers within a module that differ only
// in case. ASN.1 identifiers are case-sensitive, but case-only differences are
// a common source of confusion. SEQUENCE type definitions are excluded since
// they conventionally mirror row entry names with different initial case.
func checkIdentifierCaseMatch(ctx *resolverContext) {
	for _, mod := range ctx.modules {
		if module.IsBaseModule(mod.Name) {
			continue
		}
		// Group definitions by lowercased name, excluding SEQUENCE types.
		byLower := make(map[string][]module.Definition)
		for _, def := range mod.Definitions {
			if isSequenceTypeDefDirect(def) {
				continue
			}
			key := strings.ToLower(def.DefinitionName())
			byLower[key] = append(byLower[key], def)
		}
		for _, defs := range byLower {
			if len(defs) < 2 {
				continue
			}
			// Check for actual case differences (skip exact duplicates).
			seen := make(map[string]struct{})
			var distinct []module.Definition
			for _, def := range defs {
				if _, ok := seen[def.DefinitionName()]; !ok {
					seen[def.DefinitionName()] = struct{}{}
					distinct = append(distinct, def)
				}
			}
			if len(distinct) < 2 {
				continue
			}
			// Report on all but the first definition.
			firstName := distinct[0].DefinitionName()
			for _, def := range distinct[1:] {
				ctx.EmitDiagnostic(types.DiagIdentifierCaseMatch,
					mod, def.DefinitionSpan(),
					fmt.Sprintf("%q differs from %q only in case", def.DefinitionName(), firstName))
			}
		}
	}
}

// checkTrapInSMIv2 flags TRAP-TYPE definitions in SMIv2 modules. TRAP-TYPE
// is an SMIv1 construct; SMIv2 modules should use NOTIFICATION-TYPE instead.
func checkTrapInSMIv2(ctx *resolverContext) {
	for _, mod := range ctx.modules {
		if mod.Language != types.LanguageSMIv2 || module.IsBaseModule(mod.Name) {
			continue
		}
		for _, def := range mod.Definitions {
			notif, ok := def.(*module.Notification)
			if !ok || !notif.IsTrap() {
				continue
			}
			ctx.EmitDiagnostic(types.DiagTrapInSMIv2,
				mod, notif.Span,
				fmt.Sprintf("%q: TRAP-TYPE should not be used in SMIv2 module, use NOTIFICATION-TYPE", notif.Name))
		}
	}
}

// collectSyntaxTypeRefs extracts type name references from a TypeSyntax and
// calls markRef for each one.
func collectSyntaxTypeRefs(syntax module.TypeSyntax, mod *module.Module, markRef func(*module.Module, string)) {
	if syntax == nil {
		return
	}
	switch s := syntax.(type) {
	case *module.TypeSyntaxTypeRef:
		markRef(mod, s.Name)
	case *module.TypeSyntaxIntegerEnum:
		if s.Base != "" {
			markRef(mod, s.Base)
		}
	case *module.TypeSyntaxConstrained:
		collectSyntaxTypeRefs(s.Base, mod, markRef)
	case *module.TypeSyntaxSequenceOf:
		markRef(mod, s.EntryType)
	}
}

// smiBaseTypes lists types that must be explicitly imported in SMIv2 modules.
// ASN.1 primitives (INTEGER, OCTET STRING, OBJECT IDENTIFIER, BITS) are excluded
// because they're implicitly available.
var smiBaseTypes = map[string]string{
	"Integer32":  moduleSNMPv2SMI,
	"Counter32":  moduleSNMPv2SMI,
	"Counter64":  moduleSNMPv2SMI,
	"Gauge32":    moduleSNMPv2SMI,
	"Unsigned32": moduleSNMPv2SMI,
	"TimeTicks":  moduleSNMPv2SMI,
	"IpAddress":  moduleSNMPv2SMI,
	"Opaque":     moduleSNMPv2SMI,
}

// checkBasetypeImports verifies that SMIv2 modules explicitly import SMI base
// types they reference. RFC 2578 requires these to be imported from SNMPv2-SMI.
func checkBasetypeImports(ctx *resolverContext) {
	for _, mod := range ctx.modules {
		if module.IsBaseModule(mod.Name) {
			continue
		}
		if mod.Language != types.LanguageSMIv2 {
			continue
		}

		// Collect imported symbol names for this module.
		imported := make(map[string]struct{})
		for _, imp := range mod.Imports {
			imported[imp.Symbol] = struct{}{}
		}

		// Collect base types referenced in definitions.
		referenced := make(map[string]struct{})
		for _, def := range mod.Definitions {
			collectBaseTypeRefs(def, referenced)
		}

		for typeName := range referenced {
			if _, ok := imported[typeName]; ok {
				continue
			}
			expectedMod := smiBaseTypes[typeName]
			ctx.EmitDiagnostic(types.DiagBasetypeNotImported, mod, mod.Span,
				fmt.Sprintf("%s used but not imported from %s in %s", typeName, expectedMod, mod.Name))
		}
	}
}

// collectBaseTypeRefs adds SMI base type names referenced by a definition to the set.
func collectBaseTypeRefs(def module.Definition, refs map[string]struct{}) {
	switch d := def.(type) {
	case *module.ObjectType:
		collectSyntaxBaseTypeRefs(d.Syntax, refs)
	case *module.TypeDef:
		collectSyntaxBaseTypeRefs(d.Syntax, refs)
	}
}

func collectSyntaxBaseTypeRefs(syntax module.TypeSyntax, refs map[string]struct{}) {
	switch s := syntax.(type) {
	case *module.TypeSyntaxTypeRef:
		if _, ok := smiBaseTypes[s.Name]; ok {
			refs[s.Name] = struct{}{}
		}
	case *module.TypeSyntaxConstrained:
		collectSyntaxBaseTypeRefs(s.Base, refs)
	case *module.TypeSyntaxIntegerEnum:
		if s.Base != "" {
			if _, ok := smiBaseTypes[s.Base]; ok {
				refs[s.Base] = struct{}{}
			}
		}
	}
}

// checkNodeImplicit flags OID tree nodes that were created implicitly by
// numeric arc resolution without any explicit definition. These are unnamed
// intermediate nodes (KindInternal) that have children with actual
// definitions.
func checkNodeImplicit(ctx *resolverContext) {
	root := ctx.mib.Root()
	if root == nil {
		return
	}
	for node := range root.Subtree() {
		if node.IsRoot() || node.Kind() != KindInternal {
			continue
		}
		if node.Name() != "" {
			continue
		}
		// Only flag implicit nodes that have descendants with definitions,
		// not leaf stubs.
		if len(node.Children()) == 0 {
			continue
		}
		// Find the module of the first named child to attribute the diagnostic.
		var mod *module.Module
		for _, child := range node.Children() {
			if rm := child.Module(); rm != nil {
				if m, ok := ctx.resolvedToModule[rm]; ok {
					mod = m
					break
				}
			}
		}
		ctx.EmitDiagnostic(types.DiagNodeImplicit,
			mod, types.Span{},
			fmt.Sprintf("implicit node at OID %s", node.OID()))
	}
}

// checkModuleIdentityRegistration checks that MODULE-IDENTITY OIDs under the
// IETF mgmt subtree (1.3.6.1.2) are registered under a controlled IANA
// branch: mib-2 (1.3.6.1.2.1), transmission (1.3.6.1.2.1.10), or
// snmpModules (1.3.6.1.6.3).
func checkModuleIdentityRegistration(ctx *resolverContext) {
	// mgmt prefix: 1.3.6.1.2
	mgmtPrefix := OID{1, 3, 6, 1, 2}
	// Controlled registration points.
	mib2Prefix := OID{1, 3, 6, 1, 2, 1}
	transmissionPrefix := OID{1, 3, 6, 1, 2, 1, 10}
	snmpModulesPrefix := OID{1, 3, 6, 1, 6, 3}

	for _, mod := range ctx.modules {
		if module.IsBaseModule(mod.Name) {
			continue
		}
		for _, def := range mod.Definitions {
			mi, ok := def.(*module.ModuleIdentity)
			if !ok {
				continue
			}
			node, ok := ctx.LookupNodeForModule(mod, mi.Name)
			if !ok {
				continue
			}
			oid := node.OID()
			if len(oid) < 2 {
				ctx.EmitDiagnostic(types.DiagModuleIdentityReg,
					mod, mi.Span,
					fmt.Sprintf("%q: MODULE-IDENTITY OID too short for valid registration", mi.Name))
				continue
			}
			if !oid.HasPrefix(mgmtPrefix) {
				continue
			}
			if oid.HasPrefix(mib2Prefix) || oid.HasPrefix(transmissionPrefix) || oid.HasPrefix(snmpModulesPrefix) {
				continue
			}
			ctx.EmitDiagnostic(types.DiagModuleIdentityReg,
				mod, mi.Span,
				fmt.Sprintf("%q: MODULE-IDENTITY registered under uncontrolled mgmt OID %s", mi.Name, oid))
		}
	}
}

// checkIpAddressDeprecation warns when SMIv2 OBJECT-TYPEs use IpAddress,
// which is deprecated in favor of InetAddress (RFC 4001).
func checkIpAddressDeprecation(ctx *resolverContext, objRefs []objectTypeRef) {
	for _, ref := range objRefs {
		if ref.mod.Language != types.LanguageSMIv2 || module.IsBaseModule(ref.mod.Name) {
			continue
		}
		resolved := ctx.lookupObjectInModuleScope(ref.mod, ref.obj.Name)
		if resolved == nil {
			continue
		}
		t := resolved.Type()
		if t == nil {
			continue
		}
		if t.EffectiveBase() == BaseIpAddress {
			ctx.EmitDiagnostic(types.DiagIpAddressInSyntax,
				ref.mod, ref.obj.Span,
				fmt.Sprintf("%q: IpAddress is deprecated, use InetAddress (RFC 4001)", ref.obj.Name))
		}
	}
}
