package resolver

// Structural correctness checks for the resolver semantic analysis phase.
//
// These checks validate that the MIB structure conforms to SMI rules:
// violations here generally mean the MIB is broken or non-conformant.
//
// Check index (see also resolver_checks_conformance.go, resolver_checks_style.go).
// Severity abbreviations: E=error, S=severe, W=warning, M=minor, Y=style.
//
// Structural (this file):
//   checkNodeParentKinds           DiagParent*                  [E]  parent node kind constraints
//   checkEnumSubtyping             DiagSubtypeEnumIllegal       [E]  enum/BITS subtype validity
//                                  DiagSubtypeBitsIllegal       [E]
//   checkRangeConstraints          DiagSizeIllegal              [E]  range/SIZE constraint correctness
//                                  DiagRangeIllegal             [E]
//                                  DiagCounterRangeIllegal      [E]
//                                  DiagTimeticksRangeIllegal    [E]
//                                  DiagRangeExchanged           [E]
//                                  DiagRangeBounds              [E]
//                                  DiagRangeOverlap             [E]
//                                  DiagRangeAscending           [W]
//   checkIndexConstraints          DiagIndexAccessible          [M]  INDEX element constraints
//                                  DiagIndexNotAccessible       [M]
//                                  DiagIndexDefval              [W]
//                                  DiagIndexCounterIllegal      [W]
//                                  DiagIndexIllegalBasetype     [S]
//                                  DiagIndexElementNoSize       [M]
//                                  DiagIndexIntegerNoRange      [E]
//                                  DiagIndexNegativeRange       [E]
//                                  DiagIndexExceedsTooLarge     [W]
//   checkDefvalConstraints         DiagCounterDefvalIllegal     [W]  DEFVAL validation
//                                  DiagDefvalBasetype           [W]
//                                  DiagDefvalRange              [W]
//                                  DiagDefvalEnum               [W]
//                                  DiagDefvalBits               [W]
//   checkSequenceFields            DiagSequenceNoColumn         [M]  SEQUENCE/row consistency
//                                  DiagSequenceMissingColumn    [M]
//                                  DiagSequenceOrder            [W]
//                                  DiagSequenceTypeMismatch     [E]
//   checkAccessAndStatus           DiagAccessInvalidSMIv1       [E]  access/status rules
//                                  DiagAccessWriteOnlySMIv1     [Y]
//                                  DiagAccessWriteOnlySMIv2     [E]
//                                  DiagMaxAccessInSMIv1         [E]
//                                  DiagAccessInSMIv2            [E]
//                                  DiagAccessTableIllegal       [M]
//                                  DiagAccessRowIllegal         [M]
//                                  DiagScalarNotCreatable       [M]
//                                  DiagAccessCounterIllegal     [Y]
//                                  DiagStatusInvalidSMIv1       [E]
//                                  DiagStatusInvalidSMIv2       [E]
//                                  DiagStatusInvalidCapabilities [E]
//                                  DiagTypeStatusDeprecated     [W]
//                                  DiagTypeStatusObsolete       [W]
//   checkNotificationReversibility DiagNotifIdTooLarge          [W]  notification OID structure
//                                  DiagNotifNotReversible       [W]
//
// Conformance (resolver_checks_conformance.go):
//   checkGroupMembership           DiagGroupMembership          [M]  group membership completeness
//   checkComplianceStatus          DiagComplianceGroupStatus    [W]  compliance status ordering
//                                  DiagComplianceObjectStatus   [W]
//   checkComplianceStructure       DiagComplianceGroupInvalid   [W]  compliance structural rules
//                                  DiagOptionalGroupExists      [W]
//                                  DiagRefinementExists         [W]
//                                  DiagRefinementNotListed      [W]
//   checkGroupMemberLocality       DiagComplianceMemberNotLocal [W]  group member locality
//   checkGroupUnreferenced         DiagGroupUnreferenced        [Y]  unreferenced groups
//   checkRowStatusDefaults         DiagRowStatusDefault          [Y]  RowStatus DEFVAL/access
//                                  DiagRowStatusAccess          [Y]
//   checkStorageTypeDefaults       DiagStorageTypeDefault       [Y]  StorageType DEFVAL
//   checkTAddressTDomain           DiagTAddressTDomain          [W]  TAddress/TDomain pairing
//   checkInetAddressPairing        DiagInetAddressPairing       [W]  InetAddress pairing
//                                  DiagInetAddressTypeSubtyped  [W]
//                                  DiagInetAddressSpecific      [Y]
//   checkTransportAddressPairing   DiagTransportAddressPairing  [W]  TransportAddress pairing
//                                  DiagTransportAddressTypeSubtyped [W]
//                                  DiagTransportAddressSpecific [Y]
//
// Style (resolver_checks_style.go):
//   checkIntegerMisuse             DiagIntegerInSMIv2           [W]  bare INTEGER in SMIv2
//   checkDescriptionMissing        DiagDescriptionMissing       [M]  missing DESCRIPTION
//   checkTextualConventionNested   DiagTCNested                 [Y]  TC derived from TC
//   checkTypeAssignmentSMIv2       DiagTypeAssignmentSMIv2      [Y]  plain type assignment in v2
//   checkTableRowNaming            DiagTableNameTable           [Y]  table/row naming conventions
//                                  DiagRowNameEntry             [Y]
//                                  DiagRowNameTableName         [Y]
//   checkNamedNumberOrdering       DiagNamedNumbersAscending    [Y]  enum/BITS value ordering
//   checkHyphenInLabel             DiagHyphenInLabel            [Y]  hyphens in named values
//   checkOpaqueSMIv2               DiagOpaqueSMIv2              [W]  Opaque in SMIv2
//   checkFormatHints               DiagInvalidFormat            [E]  DISPLAY-HINT validation
//                                  DiagTypeWithoutFormat        [Y]
//   checkTypeUnreferenced          DiagTypeUnreferenced         [Y]  unreferenced types
//   checkIdentifierCaseMatch       DiagIdentifierCaseMatch      [Y]  case-only name collisions
//   checkTrapInSMIv2               DiagTrapInSMIv2              [W]  TRAP-TYPE in SMIv2
//   checkBasetypeImports           DiagBasetypeNotImported      [M]  missing basetype imports
//   checkNodeImplicit              DiagNodeImplicit             [Y]  implicit OID nodes
//   checkModuleIdentityRegistration DiagModuleIdentityReg       [W]  MODULE-IDENTITY OID checks
//   checkIpAddressDeprecation      DiagIpAddressInSyntax        [Y]  deprecated IpAddress usage

import (
	"fmt"
	"math"

	"github.com/golangsnmp/gomib/internal/model"
	"github.com/golangsnmp/gomib/internal/module"
	"github.com/golangsnmp/gomib/internal/types"
)

// isSimpleParentKind returns true for kinds that are valid parents of most
// definition types: plain nodes, internal path components, and unknown nodes.
func isSimpleParentKind(k model.Kind) bool {
	return k == model.KindNode || k == model.KindInternal || k == model.KindUnknown
}

// checkNodeParentKinds validates that each node's parent has the correct kind
// per SMI structural rules. For example, a column's parent must be a row,
// a row's parent must be a table, and most other definitions must have a
// simple node parent.
func checkNodeParentKinds(ctx *resolverContext, objRefs []objectTypeRef) {
	// Check OBJECT-TYPE definitions (table, row, column, scalar).
	for _, ref := range objRefs {
		obj := ref.obj
		node, ok := ctx.lookupNode(ref.mod, obj.Name)
		if !ok || node.Parent() == nil || node.Parent().IsRoot() {
			continue
		}
		parentKind := node.Parent().Kind()
		switch node.Kind() {
		case model.KindTable:
			if !isSimpleParentKind(parentKind) {
				ctx.EmitDiagnostic(types.DiagParentTable,
					ref.mod, obj.Span,
					fmt.Sprintf("%q: table's parent node must be a simple node", obj.Name))
			}
		case model.KindRow:
			if parentKind != model.KindTable {
				ctx.EmitDiagnostic(types.DiagParentRow,
					ref.mod, obj.Span,
					fmt.Sprintf("%q: row's parent node must be a table", obj.Name))
			} else if node.Arc() != 1 {
				ctx.EmitDiagnostic(types.DiagRowSubidentifierOne,
					ref.mod, obj.Span,
					fmt.Sprintf("%q: row node must have sub-identifier 1", obj.Name))
			}
		case model.KindColumn:
			if parentKind != model.KindRow {
				ctx.EmitDiagnostic(types.DiagParentColumn,
					ref.mod, obj.Span,
					fmt.Sprintf("%q: column's parent node must be a row", obj.Name))
			}
		case model.KindScalar:
			if !isSimpleParentKind(parentKind) {
				ctx.EmitDiagnostic(types.DiagParentScalar,
					ref.mod, obj.Span,
					fmt.Sprintf("%q: scalar's parent node must be a simple node", obj.Name))
			}
		}
	}

	// Check definitions with OIDs that aren't OBJECT-TYPE.
	for _, mod := range ctx.modules {
		if module.IsBaseModule(mod.Name) {
			continue
		}
		for _, d := range mod.Notifications {
			checkSimpleParentKind(ctx, mod, d.Name, d.Span, types.DiagParentNotification, "notification")
		}
		for _, d := range mod.ObjectIdentities {
			checkSimpleParentKind(ctx, mod, d.Name, d.Span, types.DiagParentNode, "node")
		}
		for _, d := range mod.ModuleIdentities {
			checkSimpleParentKind(ctx, mod, d.Name, d.Span, types.DiagParentNode, "node")
		}
		for _, d := range mod.ValueAssignments {
			checkSimpleParentKind(ctx, mod, d.Name, d.Span, types.DiagParentNode, "node")
		}
		for _, d := range mod.ObjectGroups {
			checkSimpleParentKind(ctx, mod, d.Name, d.Span, types.DiagParentGroup, "group")
		}
		for _, d := range mod.NotificationGroups {
			checkSimpleParentKind(ctx, mod, d.Name, d.Span, types.DiagParentGroup, "group")
		}
		for _, d := range mod.Compliances {
			checkSimpleParentKind(ctx, mod, d.Name, d.Span, types.DiagParentCompliance, "compliance")
		}
		for _, d := range mod.Capabilities {
			checkSimpleParentKind(ctx, mod, d.Name, d.Span, types.DiagParentCapabilities, "capabilities")
		}
	}
}

// checkSimpleParentKind validates that a named definition's parent node is a
// simple node (model.KindNode, model.KindInternal, or model.KindUnknown).
func checkSimpleParentKind(ctx *resolverContext, mod *module.Module, name string, span types.Span, code, label string) {
	node, ok := ctx.lookupNode(mod, name)
	if !ok || node.Parent() == nil || node.Parent().IsRoot() {
		return
	}
	if !isSimpleParentKind(node.Parent().Kind()) {
		ctx.EmitDiagnostic(code,
			mod, span,
			fmt.Sprintf("%q: %s's parent node must be a simple node", name, label))
	}
}

// checkEnumSubtyping validates that when a derived type or object restricts
// an enum or BITS parent, every declared named value exists in the parent.
func checkEnumSubtyping(ctx *resolverContext) {
	for _, mod := range ctx.modules {
		if module.IsBaseModule(mod.Name) {
			continue
		}
		for _, d := range mod.TypeDefs {
			checkEnumSubtypingSyntax(ctx, d.Syntax, d.Name, mod, d.Spans.Syntax)
		}
		for _, d := range mod.ObjectTypes {
			checkEnumSubtypingSyntax(ctx, d.Syntax, d.Name, mod, d.Spans.Syntax)
		}
	}
}

// checkEnumSubtypingSyntax checks a single TypeSyntaxIntegerEnum for
// named values not present in the parent type.
func checkEnumSubtypingSyntax(ctx *resolverContext, syntax module.TypeSyntax, name string, mod *module.Module, span types.Span) {
	enumSyntax, ok := syntax.(*module.TypeSyntaxIntegerEnum)
	if !ok || enumSyntax.Base == "" || len(enumSyntax.NamedNumbers) == 0 {
		return
	}

	parentType, ok := ctx.resolveTypeForModule(mod, enumSyntax.Base)
	if !ok {
		return
	}

	var parentValues []model.NamedValue
	var diagCode string
	var label string
	if bits := parentType.EffectiveBits(); len(bits) > 0 {
		parentValues, diagCode, label = bits, types.DiagSubtypeBitsIllegal, "BITS value"
	} else if enums := parentType.EffectiveEnums(); len(enums) > 0 {
		parentValues, diagCode, label = enums, types.DiagSubtypeEnumIllegal, "enum value"
	} else {
		return
	}

	for _, nn := range enumSyntax.NamedNumbers {
		found := false
		for _, pv := range parentValues {
			if pv.Label == nn.Name && pv.Value == nn.Value {
				found = true
				break
			}
		}
		if !found {
			ctx.EmitDiagnostic(diagCode,
				mod, span,
				fmt.Sprintf("%q: %s %s(%d) not in parent type %q", name, label, nn.Name, nn.Value, enumSyntax.Base))
		}
	}
}

// checkRangeConstraints validates range and SIZE constraints on type
// definitions and object-type inline syntax. Checks for:
//   - size-illegal: SIZE on non-OCTET-STRING type
//   - range-illegal: range on non-numeric type
//   - range-exchanged: min > max
//   - range-bounds: limits exceed basetype
//   - range-overlap: adjacent ranges overlap
//   - range-ascending: ranges not in ascending order
func checkRangeConstraints(ctx *resolverContext) {
	for _, mod := range ctx.modules {
		if module.IsBaseModule(mod.Name) {
			continue
		}
		for _, d := range mod.TypeDefs {
			if _, isSeq := d.Syntax.(*module.TypeSyntaxSequence); isSeq {
				continue
			}
			typ, ok := ctx.resolveTypeForModule(mod, d.Name)
			if !ok {
				continue
			}
			base := typ.EffectiveBase()
			sizes, ranges := extractConstraints(d.Syntax)
			checkRangeList(ctx, ranges, base, false, d.Name, mod, d.Spans.Syntax)
			checkRangeList(ctx, sizes, base, true, d.Name, mod, d.Spans.Syntax)
			if len(ranges) > 0 && typ.EffectiveRangesConstrained() && len(typ.EffectiveRanges()) == 0 {
				ctx.EmitDiagnostic(types.DiagConstraintEmptyIntersection, mod, d.Spans.Syntax,
					fmt.Sprintf("%q: range constraint has an empty intersection with its parent", d.Name))
			}
			if len(sizes) > 0 && typ.EffectiveSizesConstrained() && len(typ.EffectiveSizes()) == 0 {
				ctx.EmitDiagnostic(types.DiagConstraintEmptyIntersection, mod, d.Spans.Syntax,
					fmt.Sprintf("%q: SIZE constraint has an empty intersection with its parent", d.Name))
			}
		}
		for _, d := range mod.ObjectTypes {
			sizes, ranges := extractConstraints(d.Syntax)
			if len(sizes) == 0 && len(ranges) == 0 {
				continue
			}
			resolved := ctx.lookupObject(mod, d.Name)
			if resolved == nil || resolved.Type() == nil {
				continue
			}
			base := resolved.Type().EffectiveBase()
			checkRangeList(ctx, ranges, base, false, d.Name, mod, d.Spans.Syntax)
			checkRangeList(ctx, sizes, base, true, d.Name, mod, d.Spans.Syntax)
			if len(ranges) > 0 && resolved.EffectiveRangesConstrained() && len(resolved.EffectiveRanges()) == 0 {
				ctx.EmitDiagnostic(types.DiagConstraintEmptyIntersection, mod, d.Spans.Syntax,
					fmt.Sprintf("%q: range constraint has an empty intersection with its parent", d.Name))
			}
			if len(sizes) > 0 && resolved.EffectiveSizesConstrained() && len(resolved.EffectiveSizes()) == 0 {
				ctx.EmitDiagnostic(types.DiagConstraintEmptyIntersection, mod, d.Spans.Syntax,
					fmt.Sprintf("%q: SIZE constraint has an empty intersection with its parent", d.Name))
			}
		}
	}
}

// checkRangeList validates a slice of ranges for internal consistency
// and basetype conformance.
func checkRangeList(ctx *resolverContext, ranges []model.Range, base model.BaseType, isSize bool, name string, mod *module.Module, span types.Span) {
	if len(ranges) == 0 {
		return
	}

	// SIZE only applies to OCTET STRING and Opaque (which is OCTET STRING-based).
	if isSize && base != model.BaseOctetString && base != model.BaseOpaque {
		ctx.EmitDiagnostic(types.DiagSizeIllegal,
			mod, span,
			fmt.Sprintf("%q: SIZE constraint illegal for non-octet-string type", name))
		return
	}

	// Value range only applies to numeric types.
	if !isSize && !isNumericBase(base) {
		ctx.EmitDiagnostic(types.DiagRangeIllegal,
			mod, span,
			fmt.Sprintf("%q: range constraint illegal for non-numerical type", name))
		return
	}

	// Counter types must not have range restrictions (RFC 2578).
	if !isSize && (base == model.BaseCounter32 || base == model.BaseCounter64) {
		ctx.EmitDiagnostic(types.DiagCounterRangeIllegal,
			mod, span,
			fmt.Sprintf("%q: range constraint illegal for Counter type", name))
		return
	}

	// TimeTicks may not be sub-typed (RFC 2578 s7.1.8).
	if !isSize && base == model.BaseTimeTicks {
		ctx.EmitDiagnostic(types.DiagTimeticksRangeIllegal,
			mod, span,
			fmt.Sprintf("%q: range constraint illegal for TimeTicks type", name))
		return
	}

	bounds, hasBounds := model.BaseConstraint(base, isSize, span)

	var previous *model.Range
	for i := range ranges {
		r := &ranges[i]
		// Exchanged limits (min > max).
		comparison, comparable := r.Min.Compare(r.Max)
		if comparable && comparison > 0 {
			ctx.EmitDiagnostic(types.DiagRangeExchanged,
				mod, span,
				fmt.Sprintf("%q: range %s has exchanged limits", name, r.String()))
		}

		// Bounds checking against basetype.
		if hasBounds {
			checkRangeBound(ctx, r.Min, bounds.Min, bounds.Max, name, "lower", mod, span)
			checkRangeBound(ctx, r.Max, bounds.Min, bounds.Max, name, "upper", mod, span)
		}

		// Multi-range checks against the previous resolved range.
		if previous != nil {
			if comparison, ok := r.Min.Compare(previous.Min); ok && comparison < 0 {
				ctx.EmitDiagnostic(types.DiagRangeAscending,
					mod, span,
					fmt.Sprintf("%q: ranges not in ascending order", name))
			}
			if comparison, ok := r.Min.Compare(previous.Max); ok && comparison <= 0 {
				ctx.EmitDiagnostic(types.DiagRangeOverlap,
					mod, span,
					fmt.Sprintf("%q: range %s overlaps with %s", name, r.String(), previous.String()))
			}
		}
		previous = r
	}
}

// checkRangeBound emits a range-bounds diagnostic for a concrete endpoint
// outside the base type. Symbolic and raw endpoints are preserved and skipped.
func checkRangeBound(ctx *resolverContext, value, boundsMin, boundsMax model.RangeBound, name, which string, mod *module.Module, span types.Span) {
	if !value.IsConcrete() {
		return
	}
	below, belowOK := value.Compare(boundsMin)
	above, aboveOK := value.Compare(boundsMax)
	if (belowOK && below < 0) || (aboveOK && above > 0) {
		ctx.EmitDiagnostic(types.DiagRangeBounds,
			mod, span,
			fmt.Sprintf("%q: range %s bound %s exceeds basetype", name, which, value.String()))
	}
}

// isNumericBase reports whether the base type supports value range constraints.
func isNumericBase(base model.BaseType) bool {
	_, ok := model.BaseConstraint(base, false, types.Span{})
	return ok
}

// isLegalIndexBasetype reports whether a base type is legal for INDEX
// elements per RFC 2578 section 7.7.
func isLegalIndexBasetype(base model.BaseType) bool {
	switch base {
	case model.BaseInteger32, model.BaseUnsigned32, model.BaseGauge32, model.BaseTimeTicks,
		model.BaseIpAddress, model.BaseOctetString, model.BaseOpaque, model.BaseBits, model.BaseObjectIdentifier:
		return true
	default:
		return false
	}
}

// maxSizeFromRanges returns the maximum Max value across all size ranges,
// or -1 if sizes is empty.
func maxSizeFromRanges(sizes []model.Range) int64 {
	m := int64(-1)
	for _, r := range sizes {
		if value, ok := r.Max.AsInt64(); ok && value > m {
			m = value
		}
	}
	return m
}

// indexElementSubIds returns the maximum number of sub-identifiers an index
// element contributes to an instance model.OID. Returns (count, true) if computable,
// or (0, false) if the encoding or type constraints are insufficient.
func indexElementSubIds(entry model.IndexEntry) (int, bool) {
	obj := entry.Object
	if obj == nil {
		return 0, false
	}
	switch entry.Encoding {
	case model.IndexEncodingInteger, model.IndexEncodingIpAddress, model.IndexEncodingFixedString:
		return entry.FixedSize()
	case model.IndexEncodingLengthPrefixed:
		t := obj.Type()
		if t == nil {
			return 0, false
		}
		if t.EffectiveBase() == model.BaseObjectIdentifier {
			return 128 + 1, true
		}
		m := maxSizeFromRanges(obj.EffectiveSizes())
		if m < 0 {
			return 0, false
		}
		return int(m) + 1, true
	case model.IndexEncodingImplied:
		t := obj.Type()
		if t == nil {
			return 0, false
		}
		if t.EffectiveBase() == model.BaseObjectIdentifier {
			return 128, true
		}
		m := maxSizeFromRanges(obj.EffectiveSizes())
		if m < 0 {
			return 0, false
		}
		return int(m), true
	default:
		return 0, false
	}
}

// checkIndexConstraints validates INDEX elements: illegal basetypes,
// missing range restrictions, negative ranges, and total OID length.
// Per RFC 2578 section 7.7, index syntax must resolve to INTEGER,
// OCTET STRING, IpAddress, or OBJECT IDENTIFIER. Per section 3.5,
// an OID may have at most 128 sub-identifiers.
func checkIndexConstraints(ctx *resolverContext, objRefs []objectTypeRef) {
	for _, ref := range objRefs {
		obj := ref.obj
		if len(obj.Index) == 0 {
			continue
		}
		for _, item := range obj.Index {
			if isBareTypeIndex(item.Object) {
				continue
			}
			idxObj := ctx.lookupObject(ref.mod, item.Object)
			if idxObj == nil {
				continue
			}

			// INDEX column access: SMIv2 requires not-accessible,
			// SMIv1 requires accessible (RFC 2578 s7.7, RFC 1212 s4.1).
			switch ref.mod.Language {
			case types.LanguageSMIv2:
				if idxObj.Access() != model.AccessNotAccessible {
					ctx.EmitDiagnostic(types.DiagIndexAccessible,
						ref.mod, item.Span,
						fmt.Sprintf("INDEX %q of %q should be not-accessible in SMIv2", item.Object, obj.Name))
				}
			case types.LanguageSMIv1:
				if idxObj.Access() == model.AccessNotAccessible {
					ctx.EmitDiagnostic(types.DiagIndexNotAccessible,
						ref.mod, item.Span,
						fmt.Sprintf("INDEX %q of %q should be accessible in SMIv1", item.Object, obj.Name))
				}
			case types.LanguageUnknown:
				// Cannot validate index access without known language.
			}

			// INDEX elements should not have DEFVAL (RFC 2578 s7.7).
			if !idxObj.DefaultValue().IsZero() {
				ctx.EmitDiagnostic(types.DiagIndexDefval,
					ref.mod, item.Span,
					fmt.Sprintf("INDEX %q of %q has a DEFVAL", item.Object, obj.Name))
			}

			t := idxObj.Type()
			if t == nil {
				continue
			}
			base := t.EffectiveBase()
			if base == model.BaseCounter32 {
				// Counter32 is forbidden per RFC 2578 s7.7 but used in real MIBs.
				// Emit warning but continue - Counter32 is functionally valid for indexing.
				ctx.EmitDiagnostic(types.DiagIndexCounterIllegal,
					ref.mod, item.Span,
					fmt.Sprintf("INDEX %q of %q has counter base type", item.Object, obj.Name))
			} else if !isLegalIndexBasetype(base) {
				ctx.EmitDiagnostic(types.DiagIndexIllegalBasetype,
					ref.mod, item.Span,
					fmt.Sprintf("INDEX %q of %q has illegal base type %s", item.Object, obj.Name, base.String()))
				continue
			}
			// OCTET STRING/Opaque index elements must have a SIZE constraint
			// so the encoding length is bounded.
			if base == model.BaseOctetString || base == model.BaseOpaque {
				if !idxObj.EffectiveSizesConstrained() {
					ctx.EmitDiagnostic(types.DiagIndexElementNoSize,
						ref.mod, item.Span,
						fmt.Sprintf("INDEX %q of %q has no SIZE restriction", item.Object, obj.Name))
				}
				continue
			}
			if base != model.BaseInteger32 {
				continue
			}

			ranges := idxObj.EffectiveRanges()
			enums := idxObj.EffectiveEnums()

			// Enum-valued indexes: check for negative enum values.
			if len(enums) > 0 {
				for _, e := range enums {
					if e.Value < 0 {
						ctx.EmitDiagnostic(types.DiagIndexNegativeRange,
							ref.mod, item.Span,
							fmt.Sprintf("INDEX %q of %q has negative enumeration value %q", item.Object, obj.Name, e.Label))
						break
					}
				}
				continue
			}

			// No range restriction on an integer index.
			if !idxObj.EffectiveRangesConstrained() {
				ctx.EmitDiagnostic(types.DiagIndexIntegerNoRange,
					ref.mod, item.Span,
					fmt.Sprintf("INDEX %q of %q has no range restriction", item.Object, obj.Name))
				continue
			}

			// model.Range includes negative values.
			for _, r := range ranges {
				if comparison, ok := r.Min.Compare(model.NewSignedRangeBound(0)); ok && comparison < 0 {
					ctx.EmitDiagnostic(types.DiagIndexNegativeRange,
						ref.mod, item.Span,
						fmt.Sprintf("INDEX %q of %q has range permitting negative values", item.Object, obj.Name))
					break
				}
			}
		}

		// Check total index OID length against the 128 sub-identifier limit.
		// Instance OID = row OID + index sub-identifiers.
		checkIndexOIDLength(ctx, ref)
	}
}

// checkIndexOIDLength checks that a row's instance OID does not exceed the
// 128 sub-identifier maximum from RFC 2578 section 3.5. The instance model.OID
// is the row's OID plus one sub-identifier per index encoding.
func checkIndexOIDLength(ctx *resolverContext, ref objectTypeRef) {
	rowNode, ok := ctx.lookupNode(ref.mod, ref.obj.Name)
	if !ok {
		return
	}
	rowOID := rowNode.OID()
	if rowOID == nil {
		return
	}

	resolvedObj := ctx.lookupObject(ref.mod, ref.obj.Name)
	if resolvedObj == nil {
		return
	}

	totalLen := len(rowOID)
	for _, entry := range resolvedObj.Index() {
		n, ok := indexElementSubIds(entry)
		if !ok {
			return // can't compute, skip the check
		}
		totalLen += n
	}

	if totalLen > 128 {
		excess := totalLen - 128
		ctx.EmitDiagnostic(types.DiagIndexExceedsTooLarge,
			ref.mod, ref.obj.Spans.Index,
			fmt.Sprintf("%q index OID exceeds 128 sub-identifiers by %d", ref.obj.Name, excess))
	}
}

// checkDefvalConstraints validates DEFVAL values against the object's type.
// Checks basetype range, RANGE constraints, and enum/BITS label validity.
func checkDefvalConstraints(ctx *resolverContext, objRefs []objectTypeRef) {
	for _, ref := range objRefs {
		obj := ref.obj
		if obj.DefVal == nil {
			continue
		}

		resolved := ctx.lookupObject(ref.mod, obj.Name)
		if resolved == nil {
			continue
		}
		dv := resolved.DefaultValue()
		if dv.IsZero() {
			continue
		}
		t := resolved.Type()
		if t == nil {
			continue
		}
		base := t.EffectiveBase()

		// Counter32 and Counter64 must not have DEFVAL (RFC 2578 s7.9).
		if base == model.BaseCounter32 || base == model.BaseCounter64 {
			ctx.EmitDiagnostic(types.DiagCounterDefvalIllegal,
				ref.mod, obj.Spans.DefVal,
				fmt.Sprintf("%q: DEFVAL not allowed for counter type", obj.Name))
			continue
		}

		ranges := resolved.EffectiveRanges()
		enums := resolved.EffectiveEnums()
		bits := resolved.EffectiveBits()

		switch dv.Kind() {
		case model.DefValKindInt:
			v, _ := model.DefValAs[int64](dv)
			checkDefvalNumeric(ctx, ref.mod, obj, v, 0, false, base, ranges, resolved.EffectiveRangesConstrained(), enums)
		case model.DefValKindUint:
			v, _ := model.DefValAs[uint64](dv)
			checkDefvalNumeric(ctx, ref.mod, obj, 0, v, true, base, ranges, resolved.EffectiveRangesConstrained(), enums)
		case model.DefValKindEnum:
			label, _ := model.DefValAs[string](dv)
			if len(enums) > 0 {
				if !findNamedValue(enums, label) {
					ctx.EmitDiagnostic(types.DiagDefvalEnum,
						ref.mod, obj.Spans.DefVal,
						fmt.Sprintf("%q: DEFVAL enum label %q not defined in type", obj.Name, label))
				}
			}
		case model.DefValKindBits:
			labels, _ := model.DefValAs[[]string](dv)
			for _, label := range labels {
				if !findNamedValue(bits, label) {
					ctx.EmitDiagnostic(types.DiagDefvalBits,
						ref.mod, obj.Spans.DefVal,
						fmt.Sprintf("%q: DEFVAL BITS label %q not defined in type", obj.Name, label))
				}
			}
		case model.DefValKindUnset, model.DefValKindString, model.DefValKindBytes, model.DefValKindOID:
			// No validation for these kinds.
		}
	}
}

// checkDefvalNumeric validates a numeric DEFVAL value against basetype limits,
// RANGE constraints, and enum membership. Works for both signed and unsigned
// values: signed values have isUnsigned=false and uval=0, unsigned values have
// isUnsigned=true and ival=0.
func checkDefvalNumeric(ctx *resolverContext, mod *module.Module, obj *module.ObjectType, ival int64, uval uint64, isUnsigned bool, base model.BaseType, ranges []model.Range, rangesConstrained bool, enums []model.NamedValue) {
	// Check basetype limits.
	switch base {
	case model.BaseInteger32:
		if isUnsigned {
			if uval > math.MaxInt32 {
				ctx.EmitDiagnostic(types.DiagDefvalBasetype,
					mod, obj.Spans.DefVal,
					fmt.Sprintf("%q: DEFVAL %d exceeds Integer32 range", obj.Name, uval))
			}
		} else {
			if ival < math.MinInt32 || ival > math.MaxInt32 {
				ctx.EmitDiagnostic(types.DiagDefvalBasetype,
					mod, obj.Spans.DefVal,
					fmt.Sprintf("%q: DEFVAL %d exceeds Integer32 range", obj.Name, ival))
			}
		}
	case model.BaseUnsigned32, model.BaseGauge32, model.BaseCounter32, model.BaseTimeTicks:
		if isUnsigned {
			if uval > math.MaxUint32 {
				ctx.EmitDiagnostic(types.DiagDefvalBasetype,
					mod, obj.Spans.DefVal,
					fmt.Sprintf("%q: DEFVAL %d exceeds unsigned32 range", obj.Name, uval))
			}
		} else {
			if ival < 0 || ival > math.MaxUint32 {
				ctx.EmitDiagnostic(types.DiagDefvalBasetype,
					mod, obj.Spans.DefVal,
					fmt.Sprintf("%q: DEFVAL %d exceeds unsigned32 range", obj.Name, ival))
			}
		}
	}

	// Check RANGE constraints. An unresolved endpoint makes non-membership
	// unknowable, so only diagnose when every range is resolved.
	if rangesConstrained {
		var inRange bool
		if isUnsigned {
			inRange = uvalueInRanges(uval, ranges)
		} else {
			inRange = valueInRanges(ival, ranges)
		}
		if !inRange && (len(ranges) == 0 || !hasUnresolvedRange(ranges)) {
			v := any(ival)
			if isUnsigned {
				v = uval
			}
			ctx.EmitDiagnostic(types.DiagDefvalRange,
				mod, obj.Spans.DefVal,
				fmt.Sprintf("%q: DEFVAL %v outside RANGE constraint", obj.Name, v))
		}
	}

	// Check enum membership.
	if len(enums) > 0 {
		found := false
		for _, e := range enums {
			if isUnsigned {
				if e.Value >= 0 && uint64(e.Value) == uval {
					found = true
					break
				}
			} else {
				if e.Value == ival {
					found = true
					break
				}
			}
		}
		if !found {
			v := any(ival)
			if isUnsigned {
				v = uval
			}
			ctx.EmitDiagnostic(types.DiagDefvalEnum,
				mod, obj.Spans.DefVal,
				fmt.Sprintf("%q: DEFVAL %v does not match any enumeration value", obj.Name, v))
		}
	}
}

func hasUnresolvedRange(ranges []model.Range) bool {
	for i := range ranges {
		if !ranges[i].IsResolved() {
			return true
		}
	}
	return false
}

func valueInRanges(v int64, ranges []model.Range) bool {
	for i := range ranges {
		if model.RangeContainsInt64(&ranges[i], v) {
			return true
		}
	}
	return false
}

// findNamedValue reports whether a named value with the given label exists.
func findNamedValue(values []model.NamedValue, label string) bool {
	for _, nv := range values {
		if nv.Label == label {
			return true
		}
	}
	return false
}

func uvalueInRanges(v uint64, ranges []model.Range) bool {
	for i := range ranges {
		if model.RangeContainsUint64(&ranges[i], v) {
			return true
		}
	}
	return false
}

// checkSequenceFields validates that SEQUENCE definitions match their row's
// actual columnar children: fields exist as columns, columns appear in the
// SEQUENCE, field order matches column OID order, and field types match.
func checkSequenceFields(ctx *resolverContext, objRefs []objectTypeRef) {
	for _, ref := range objRefs {
		obj := ref.obj
		if len(obj.Index) == 0 && obj.Augments == "" {
			continue
		}

		// Get the SEQUENCE type name from the row's SYNTAX clause.
		syntaxRef, ok := obj.Syntax.(*module.TypeSyntaxTypeRef)
		if !ok {
			continue
		}

		td := findSequenceTypeDef(ctx, ref.mod, syntaxRef.Name)
		if td == nil {
			continue
		}
		seqSyntax, ok := td.Syntax.(*module.TypeSyntaxSequence)
		if !ok {
			continue
		}

		// Get the row node and its column children in OID order.
		rowNode, ok := ctx.lookupNode(ref.mod, obj.Name)
		if !ok {
			continue
		}
		columns := rowNode.Children()

		// Build lookup maps.
		columnByName := make(map[string]*model.Node, len(columns))
		for _, col := range columns {
			columnByName[col.Name()] = col
		}
		fieldNames := make(map[string]struct{}, len(seqSyntax.Fields))
		for _, f := range seqSyntax.Fields {
			fieldNames[f.Name] = struct{}{}
		}

		// Check 1: sequence-no-column - SEQUENCE field has no matching column.
		for _, field := range seqSyntax.Fields {
			if _, found := columnByName[field.Name]; !found {
				ctx.EmitDiagnostic(types.DiagSequenceNoColumn,
					ref.mod, field.Span,
					fmt.Sprintf("SEQUENCE %q field %q is not a column of row %q", syntaxRef.Name, field.Name, obj.Name))
			}
		}

		// Check 2: sequence-missing-column - column not in SEQUENCE.
		for _, col := range columns {
			if _, found := fieldNames[col.Name()]; !found {
				ctx.EmitDiagnostic(types.DiagSequenceMissingColumn,
					ref.mod, obj.Span,
					fmt.Sprintf("column %q of row %q is not in SEQUENCE %q", col.Name(), obj.Name, syntaxRef.Name))
			}
		}

		// Check 3: sequence-order - field order doesn't match column OID order.
		// Filter SEQUENCE fields to only those that are actual columns,
		// then compare against columns that are in the SEQUENCE.
		var seqOrder []string
		for _, field := range seqSyntax.Fields {
			if _, found := columnByName[field.Name]; found {
				seqOrder = append(seqOrder, field.Name)
			}
		}
		var colOrder []string
		for _, col := range columns {
			if _, found := fieldNames[col.Name()]; found {
				colOrder = append(colOrder, col.Name())
			}
		}
		if len(seqOrder) > 0 && len(colOrder) > 0 {
			mismatch := false
			for i := range seqOrder {
				if i >= len(colOrder) || seqOrder[i] != colOrder[i] {
					mismatch = true
					break
				}
			}
			if mismatch {
				ctx.EmitDiagnostic(types.DiagSequenceOrder,
					ref.mod, obj.Span,
					fmt.Sprintf("SEQUENCE %q field order does not match column OID order in row %q", syntaxRef.Name, obj.Name))
			}
		}

		// Check 4: sequence-type-mismatch - field type doesn't match column type.
		for _, field := range seqSyntax.Fields {
			col, found := columnByName[field.Name]
			if !found {
				continue
			}
			colObj := col.Object()
			if colObj == nil || colObj.Type() == nil {
				continue
			}
			fieldTypeName := sequenceFieldTypeName(field.Syntax)
			if fieldTypeName == "" {
				continue
			}
			colTypeName := colObj.Type().Name()
			if sequenceTypesCompatible(fieldTypeName, colTypeName, colObj.Type().EffectiveBase()) {
				continue
			}
			ctx.EmitDiagnostic(types.DiagSequenceTypeMismatch,
				ref.mod, field.Span,
				fmt.Sprintf("SEQUENCE %q field %q type %q does not match column type %q", syntaxRef.Name, field.Name, fieldTypeName, colTypeName))
		}
	}
}

// findSequenceTypeDef looks up the SEQUENCE TypeDef by name in the module or
// its imports. Returns nil if not found.
func findSequenceTypeDef(ctx *resolverContext, mod *module.Module, name string) *module.TypeDef {
	if td := getSequenceTypeDef(mod, name); td != nil {
		return td
	}
	if srcMod, ok := ctx.importSources.get(mod, name); ok {
		return getSequenceTypeDef(srcMod, name)
	}
	return nil
}

func getSequenceTypeDef(mod *module.Module, name string) *module.TypeDef {
	for _, td := range mod.TypeDefs {
		if td.Name == name {
			if _, isSeq := td.Syntax.(*module.TypeSyntaxSequence); isSeq {
				return td
			}
		}
	}
	return nil
}

// sequenceFieldTypeName extracts the type name from a SEQUENCE field's syntax.
func sequenceFieldTypeName(syntax module.TypeSyntax) string {
	switch s := syntax.(type) {
	case *module.TypeSyntaxTypeRef:
		return s.Name
	case *module.TypeSyntaxIntegerEnum:
		return "INTEGER"
	case *module.TypeSyntaxBits:
		return "BITS"
	case *module.TypeSyntaxOctetString:
		return "OCTET STRING"
	case *module.TypeSyntaxObjectIdentifier:
		return "OBJECT IDENTIFIER"
	case *module.TypeSyntaxConstrained:
		return sequenceFieldTypeName(s.Base)
	default:
		return ""
	}
}

// sequenceTypesCompatible checks whether a SEQUENCE field type name is
// compatible with the column's resolved type. Handles cases where the
// SEQUENCE uses a primitive name but the column uses a named type with
// compatible base, including SMIv1/SMIv2 aliases.
func sequenceTypesCompatible(fieldType, colType string, colBase model.BaseType) bool {
	if fieldType == colType {
		return true
	}
	// Normalize SMIv1 type names to their SMIv2 equivalents for comparison.
	fieldNorm := normalizeTypeName(fieldType)
	colNorm := normalizeTypeName(colType)
	if fieldNorm == colNorm {
		return true
	}
	// INTEGER/Integer32 in SEQUENCE is compatible with Integer32-based columns (covers enums).
	if (fieldNorm == "Integer32") && colBase == model.BaseInteger32 {
		return true
	}
	// OCTET STRING/BITS in SEQUENCE is compatible with BITS-based columns
	// (BITS is encoded as OCTET STRING at wire level).
	if (fieldType == "OCTET STRING" || fieldType == "BITS") && colBase == model.BaseBits {
		return true
	}
	return false
}

// normalizeTypeName maps SMIv1 type names to their SMIv2 equivalents.
func normalizeTypeName(name string) string {
	switch name {
	case "Counter":
		return "Counter32"
	case "Gauge":
		return "Gauge32"
	case "INTEGER":
		return "Integer32"
	case "NetworkAddress":
		return "IpAddress"
	default:
		return name
	}
}

// checkAccessAndStatus validates access values, access keywords, and status
// values per SMI version, and checks kind-specific access constraints.
func checkAccessAndStatus(ctx *resolverContext, objRefs []objectTypeRef) {
	for _, ref := range objRefs {
		obj := ref.obj
		mod := ref.mod
		if module.IsBaseModule(mod.Name) {
			continue
		}

		node, ok := ctx.lookupNode(mod, obj.Name)
		if !ok {
			continue
		}

		checkAccessPerVersion(ctx, mod, obj)
		checkAccessKeywordPerVersion(ctx, mod, obj)
		checkKindAccess(ctx, mod, obj, node)
		checkCounterAccess(ctx, mod, obj)
	}
	checkStatusPerVersion(ctx)
	checkCapabilitiesStatus(ctx)
	checkTypeStatusUsage(ctx, objRefs)
}

// checkAccessPerVersion validates that the access value is legal for the
// module's SMI version.
func checkAccessPerVersion(ctx *resolverContext, mod *module.Module, obj *module.ObjectType) {
	switch mod.Language {
	case types.LanguageSMIv1:
		if obj.Access == types.AccessAccessibleForNotify || obj.Access == types.AccessReadCreate {
			ctx.EmitDiagnostic(types.DiagAccessInvalidSMIv1,
				mod, obj.Spans.Access,
				fmt.Sprintf("%q: invalid access %s in SMIv1", obj.Name, obj.Access.String()))
		}
		if obj.Access == types.AccessWriteOnly {
			ctx.EmitDiagnostic(types.DiagAccessWriteOnlySMIv1,
				mod, obj.Spans.Access,
				fmt.Sprintf("%q: write-only is valid but discouraged in SMIv1", obj.Name))
		}
	case types.LanguageSMIv2:
		if obj.Access == types.AccessWriteOnly {
			ctx.EmitDiagnostic(types.DiagAccessWriteOnlySMIv2,
				mod, obj.Spans.Access,
				fmt.Sprintf("%q: write-only is no longer allowed in SMIv2", obj.Name))
		}
	case types.LanguageUnknown:
		// Cannot validate version-specific access rules without known language.
	}
}

// checkAccessKeywordPerVersion validates that ACCESS is used in SMIv1 and
// MAX-ACCESS in SMIv2.
func checkAccessKeywordPerVersion(ctx *resolverContext, mod *module.Module, obj *module.ObjectType) {
	// MIN-ACCESS is always SMIv2 (MODULE-COMPLIANCE) and not applicable here.
	if obj.AccessKeyword == types.AccessKeywordMinAccess {
		return
	}
	switch mod.Language {
	case types.LanguageSMIv1:
		if obj.AccessKeyword == types.AccessKeywordMaxAccess {
			ctx.EmitDiagnostic(types.DiagMaxAccessInSMIv1,
				mod, obj.Spans.Access,
				fmt.Sprintf("%q: MAX-ACCESS is SMIv2 style, use ACCESS in SMIv1", obj.Name))
		}
	case types.LanguageSMIv2:
		if obj.AccessKeyword == types.AccessKeywordAccess {
			ctx.EmitDiagnostic(types.DiagAccessInSMIv2,
				mod, obj.Spans.Access,
				fmt.Sprintf("%q: ACCESS is SMIv1 style, use MAX-ACCESS in SMIv2", obj.Name))
		}
	case types.LanguageUnknown:
		// Cannot validate version-specific keyword rules without known language.
	}
}

// checkKindAccess validates access values for specific node kinds:
// tables and rows must be not-accessible, scalars must not be read-create.
func checkKindAccess(ctx *resolverContext, mod *module.Module, obj *module.ObjectType, node *model.Node) {
	switch node.Kind() {
	case model.KindTable:
		if obj.Access != types.AccessNotAccessible {
			ctx.EmitDiagnostic(types.DiagAccessTableIllegal,
				mod, obj.Spans.Access,
				fmt.Sprintf("%q: table must be not-accessible", obj.Name))
		}
	case model.KindRow:
		if obj.Access != types.AccessNotAccessible {
			ctx.EmitDiagnostic(types.DiagAccessRowIllegal,
				mod, obj.Spans.Access,
				fmt.Sprintf("%q: row must be not-accessible", obj.Name))
		}
	case model.KindScalar:
		if obj.Access == types.AccessReadCreate {
			ctx.EmitDiagnostic(types.DiagScalarNotCreatable,
				mod, obj.Spans.Access,
				fmt.Sprintf("%q: scalar must not be read-create", obj.Name))
		}
	}
}

// checkCounterAccess validates that Counter32/Counter64 objects have read-only
// or accessible-for-notify access.
func checkCounterAccess(ctx *resolverContext, mod *module.Module, obj *module.ObjectType) {
	if obj.Access == types.AccessReadOnly || obj.Access == types.AccessAccessibleForNotify {
		return
	}
	resolved := ctx.lookupObject(mod, obj.Name)
	if resolved == nil {
		return
	}
	t := resolved.Type()
	if t == nil {
		return
	}
	base := t.EffectiveBase()
	if base == model.BaseCounter32 || base == model.BaseCounter64 {
		ctx.EmitDiagnostic(types.DiagAccessCounterIllegal,
			mod, obj.Spans.Access,
			fmt.Sprintf("%q: counter must be read-only or accessible-for-notify", obj.Name))
	}
}

// checkStatusPerVersion validates that status values are legal for the
// module's SMI version across all definition types.
func checkStatusPerVersion(ctx *resolverContext) {
	for _, mod := range ctx.modules {
		if module.IsBaseModule(mod.Name) {
			continue
		}
		for _, d := range mod.ObjectTypes {
			checkDefinitionStatus(ctx, mod, d.Status, d.Name, d.Spans.Status)
		}
		for _, d := range mod.ObjectIdentities {
			checkDefinitionStatus(ctx, mod, d.Status, d.Name, d.Span)
		}
		for _, d := range mod.Notifications {
			// TRAP-TYPE has no STATUS clause; the synthetic default
			// would produce false positives.
			if d.IsTrap() {
				continue
			}
			checkDefinitionStatus(ctx, mod, d.Status, d.Name, d.Span)
		}
		for _, d := range mod.TypeDefs {
			// Only textual conventions have a STATUS clause; plain
			// type assignments and SEQUENCE defs default to current.
			if !d.IsTextualConvention {
				continue
			}
			checkDefinitionStatus(ctx, mod, d.Status, d.Name, d.Spans.Status)
		}
		for _, d := range mod.ObjectGroups {
			checkDefinitionStatus(ctx, mod, d.Status, d.Name, d.Span)
		}
		for _, d := range mod.NotificationGroups {
			checkDefinitionStatus(ctx, mod, d.Status, d.Name, d.Span)
		}
		for _, d := range mod.Compliances {
			checkDefinitionStatus(ctx, mod, d.Status, d.Name, d.Span)
		}
		for _, d := range mod.Capabilities {
			checkDefinitionStatus(ctx, mod, d.Status, d.Name, d.Span)
		}
	}
}

// checkDefinitionStatus validates that a single definition's status is legal
// for the module's SMI version.
func checkDefinitionStatus(ctx *resolverContext, mod *module.Module, status types.Status, name string, span types.Span) {
	switch mod.Language {
	case types.LanguageSMIv1:
		if status == types.StatusCurrent {
			ctx.EmitDiagnostic(types.DiagStatusInvalidSMIv1,
				mod, span,
				fmt.Sprintf("%q: invalid status current in SMIv1", name))
		}
	case types.LanguageSMIv2:
		if status == types.StatusMandatory || status == types.StatusOptional {
			ctx.EmitDiagnostic(types.DiagStatusInvalidSMIv2,
				mod, span,
				fmt.Sprintf("%q: invalid status %s in SMIv2", name, status.String()))
		}
	case types.LanguageUnknown:
		// Cannot validate version-specific status rules without known language.
	}
}

// checkCapabilitiesStatus validates that AGENT-CAPABILITIES STATUS is
// either current or obsolete (RFC 2580 s6.2).
func checkCapabilitiesStatus(ctx *resolverContext) {
	for _, mod := range ctx.modules {
		if module.IsBaseModule(mod.Name) {
			continue
		}
		for _, ac := range mod.Capabilities {
			if ac.Status != types.StatusCurrent && ac.Status != types.StatusObsolete {
				ctx.EmitDiagnostic(types.DiagStatusInvalidCapabilities,
					mod, ac.DefinitionSpan(),
					fmt.Sprintf("%q: AGENT-CAPABILITIES STATUS must be current or obsolete, got %s", ac.Name, ac.Status.String()))
			}
		}
	}
}

// checkTypeStatusUsage warns when an OBJECT-TYPE references a type whose
// status is deprecated or obsolete.
func checkTypeStatusUsage(ctx *resolverContext, objRefs []objectTypeRef) {
	for _, ref := range objRefs {
		if module.IsBaseModule(ref.mod.Name) {
			continue
		}
		resolved := ctx.lookupObject(ref.mod, ref.obj.Name)
		if resolved == nil {
			continue
		}
		t := resolved.Type()
		if t == nil {
			continue
		}
		switch t.Status() {
		case model.StatusDeprecated:
			ctx.EmitDiagnostic(types.DiagTypeStatusDeprecated,
				ref.mod, ref.obj.Span,
				fmt.Sprintf("type %q used by %q is deprecated", t.Name(), ref.obj.Name))
		case model.StatusObsolete:
			ctx.EmitDiagnostic(types.DiagTypeStatusObsolete,
				ref.mod, ref.obj.Span,
				fmt.Sprintf("type %q used by %q is obsolete", t.Name(), ref.obj.Name))
		}
	}
}

// checkNotificationReversibility validates notification OID structure per
// RFC 2578 section 8.5. In SMIv2, the next-to-last sub-identifier of a
// notification's OID must be 0 for SNMPv1/v2 reverse mapping. Also checks
// that the last sub-identifier fits in a signed 32-bit integer.
// Five well-known notifications predate the .0. convention and are exempt.
func checkNotificationReversibility(ctx *resolverContext) {
	for _, ref := range collectNotificationRefs(ctx) {
		if ref.mod.Language != types.LanguageSMIv2 {
			continue
		}

		node, ok := ctx.lookupNode(ref.mod, ref.notif.Name)
		if !ok || node.Parent() == nil {
			continue
		}

		// Check last sub-id fits in int32 (for SNMPv1 specific-trap field).
		if node.Arc() > math.MaxInt32 {
			ctx.EmitDiagnostic(types.DiagNotifIdTooLarge,
				ref.mod, ref.notif.Span,
				fmt.Sprintf("last sub-identifier of notification %q is too large", ref.notif.Name))
		}

		// Five well-known notifications predate the .0. convention.
		if isExemptNotification(ref.mod.Name, ref.notif.Name) {
			continue
		}

		// Parent's arc must be 0 for reverse mapping to SNMPv1 traps.
		if node.Parent().Arc() != 0 {
			ctx.EmitDiagnostic(types.DiagNotifNotReversible,
				ref.mod, ref.notif.Span,
				fmt.Sprintf("notification %q is not reverse mappable", ref.notif.Name))
		}
	}
}

// isExemptNotification returns true for the five well-known notifications that
// predate the .0. convention (RFC 2578 section 8.5).
func isExemptNotification(moduleName, notifName string) bool {
	switch moduleName {
	case "SNMPv2-MIB":
		return notifName == "coldStart" ||
			notifName == "warmStart" ||
			notifName == "authenticationFailure"
	case "IF-MIB":
		return notifName == "linkDown" ||
			notifName == "linkUp"
	}
	return false
}
