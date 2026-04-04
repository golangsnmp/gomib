package mib

// Structural correctness checks for the resolver semantic analysis phase.
//
// These checks validate that the MIB structure conforms to SMI rules:
// violations here generally mean the MIB is broken or non-conformant.
//
// Check index (see also resolver_checks_conformance.go, resolver_checks_style.go):
//
// Structural (this file):
//   checkNodeParentKinds           DiagParent*                  parent node kind constraints
//   checkEnumSubtyping             DiagSubtypeEnumIllegal       enum/BITS subtype validity
//                                  DiagSubtypeBitsIllegal
//   checkRangeConstraints          DiagSizeIllegal              range/SIZE constraint correctness
//                                  DiagRangeIllegal
//                                  DiagCounterRangeIllegal
//                                  DiagTimeticksRangeIllegal
//                                  DiagRangeExchanged
//                                  DiagRangeBounds
//                                  DiagRangeOverlap
//                                  DiagRangeAscending
//   checkIndexConstraints          DiagIndexAccessible          INDEX element constraints
//                                  DiagIndexNotAccessible
//                                  DiagIndexDefval
//                                  DiagIndexCounterIllegal
//                                  DiagIndexIllegalBasetype
//                                  DiagIndexElementNoSize
//                                  DiagIndexIntegerNoRange
//                                  DiagIndexNegativeRange
//                                  DiagIndexExceedsTooLarge
//   checkDefvalConstraints         DiagCounterDefvalIllegal     DEFVAL validation
//                                  DiagDefvalBasetype
//                                  DiagDefvalRange
//                                  DiagDefvalEnum
//                                  DiagDefvalBits
//   checkSequenceFields            DiagSequenceNoColumn         SEQUENCE/row consistency
//                                  DiagSequenceMissingColumn
//                                  DiagSequenceOrder
//                                  DiagSequenceTypeMismatch
//   checkAccessAndStatus           DiagAccessInvalidSMIv1       access/status rules
//                                  DiagAccessWriteOnlySMIv1
//                                  DiagAccessWriteOnlySMIv2
//                                  DiagMaxAccessInSMIv1
//                                  DiagAccessInSMIv2
//                                  DiagAccessTableIllegal
//                                  DiagAccessRowIllegal
//                                  DiagScalarNotCreatable
//                                  DiagAccessCounterIllegal
//                                  DiagStatusInvalidSMIv1
//                                  DiagStatusInvalidSMIv2
//                                  DiagStatusInvalidCapabilities
//                                  DiagTypeStatusDeprecated
//                                  DiagTypeStatusObsolete
//   checkNotificationReversibility DiagNotifIdTooLarge          notification OID structure
//                                  DiagNotifNotReversible
//
// Conformance (resolver_checks_conformance.go):
//   checkGroupMembership           DiagGroupMembership          group membership completeness
//   checkComplianceStatus          DiagComplianceGroupStatus    compliance status ordering
//                                  DiagComplianceObjectStatus
//   checkComplianceStructure       DiagComplianceGroupInvalid   compliance structural rules
//                                  DiagOptionalGroupExists
//                                  DiagRefinementExists
//                                  DiagRefinementNotListed
//   checkGroupMemberLocality       DiagComplianceMemberNotLocal group member locality
//   checkGroupUnreferenced         DiagGroupUnreferenced        unreferenced groups
//   checkRowStatusDefaults         DiagRowStatusDefault         RowStatus DEFVAL/access
//                                  DiagRowStatusAccess
//   checkStorageTypeDefaults       DiagStorageTypeDefault       StorageType DEFVAL
//   checkTAddressTDomain           DiagTAddressTDomain          TAddress/TDomain pairing
//   checkInetAddressPairing        DiagInetAddressPairing       InetAddress pairing
//                                  DiagInetAddressTypeSubtyped
//                                  DiagInetAddressSpecific
//   checkTransportAddressPairing   DiagTransportAddressPairing  TransportAddress pairing
//                                  DiagTransportAddressTypeSubtyped
//                                  DiagTransportAddressSpecific
//
// Style (resolver_checks_style.go):
//   checkIntegerMisuse             DiagIntegerInSMIv2           bare INTEGER in SMIv2
//   checkDescriptionMissing        DiagDescriptionMissing       missing DESCRIPTION
//   checkTextualConventionNested   DiagTCNested                 TC derived from TC
//   checkTypeAssignmentSMIv2       DiagTypeAssignmentSMIv2      plain type assignment in v2
//   checkTableRowNaming            DiagTableNameTable           table/row naming conventions
//                                  DiagRowNameEntry
//                                  DiagRowNameTableName
//   checkNamedNumberOrdering       DiagNamedNumbersAscending    enum/BITS value ordering
//   checkHyphenInLabel             DiagHyphenInLabel            hyphens in named values
//   checkOpaqueSMIv2               DiagOpaqueSMIv2              Opaque in SMIv2
//   checkFormatHints               DiagInvalidFormat            DISPLAY-HINT validation
//                                  DiagTypeWithoutFormat
//   checkTypeUnreferenced          DiagTypeUnreferenced         unreferenced types
//   checkIdentifierCaseMatch       DiagIdentifierCaseMatch      case-only name collisions
//   checkTrapInSMIv2               DiagTrapInSMIv2              TRAP-TYPE in SMIv2
//   checkBasetypeImports           DiagBasetypeNotImported      missing basetype imports
//   checkNodeImplicit              DiagNodeImplicit             implicit OID nodes
//   checkModuleIdentityRegistration DiagModuleIdentityReg       MODULE-IDENTITY OID checks
//   checkIpAddressDeprecation      DiagIpAddressInSyntax        deprecated IpAddress usage

import (
	"fmt"
	"math"

	"github.com/golangsnmp/gomib/internal/module"
	"github.com/golangsnmp/gomib/internal/types"
)

// isSimpleParentKind returns true for kinds that are valid parents of most
// definition types: plain nodes, internal path components, and unknown nodes.
func isSimpleParentKind(k Kind) bool {
	return k == KindNode || k == KindInternal || k == KindUnknown
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
		case KindTable:
			if !isSimpleParentKind(parentKind) {
				ctx.EmitDiagnostic(types.DiagParentTable,
					ref.mod, obj.Span,
					fmt.Sprintf("%q: table's parent node must be a simple node", obj.Name))
			}
		case KindRow:
			if parentKind != KindTable {
				ctx.EmitDiagnostic(types.DiagParentRow,
					ref.mod, obj.Span,
					fmt.Sprintf("%q: row's parent node must be a table", obj.Name))
			} else if node.Arc() != 1 {
				ctx.EmitDiagnostic(types.DiagRowSubidentifierOne,
					ref.mod, obj.Span,
					fmt.Sprintf("%q: row node must have sub-identifier 1", obj.Name))
			}
		case KindColumn:
			if parentKind != KindRow {
				ctx.EmitDiagnostic(types.DiagParentColumn,
					ref.mod, obj.Span,
					fmt.Sprintf("%q: column's parent node must be a row", obj.Name))
			}
		case KindScalar:
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
		for _, def := range mod.Definitions {
			var code string
			var label string
			switch def.(type) {
			case *module.Notification:
				code = types.DiagParentNotification
				label = "notification"
			case *module.ObjectIdentity, *module.ModuleIdentity, *module.ValueAssignment:
				code = types.DiagParentNode
				label = "node"
			case *module.ObjectGroup, *module.NotificationGroup:
				code = types.DiagParentGroup
				label = "group"
			case *module.ModuleCompliance:
				code = types.DiagParentCompliance
				label = "compliance"
			case *module.AgentCapabilities:
				code = types.DiagParentCapabilities
				label = "capabilities"
			default:
				continue
			}
			node, ok := ctx.lookupNode(mod, def.DefinitionName())
			if !ok || node.Parent() == nil || node.Parent().IsRoot() {
				continue
			}
			if !isSimpleParentKind(node.Parent().Kind()) {
				ctx.EmitDiagnostic(code,
					mod, def.DefinitionSpan(),
					fmt.Sprintf("%q: %s's parent node must be a simple node", def.DefinitionName(), label))
			}
		}
	}
}

// checkEnumSubtyping validates that when a derived type or object restricts
// an enum or BITS parent, every declared named value exists in the parent.
func checkEnumSubtyping(ctx *resolverContext) {
	for _, mod := range ctx.modules {
		if module.IsBaseModule(mod.Name) {
			continue
		}
		for _, def := range mod.Definitions {
			switch d := def.(type) {
			case *module.TypeDef:
				checkEnumSubtypingSyntax(ctx, d.Syntax, d.Name, mod, d.Span)
			case *module.ObjectType:
				checkEnumSubtypingSyntax(ctx, d.Syntax, d.Name, mod, d.Span)
			}
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

	var parentValues []NamedValue
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
		for _, def := range mod.Definitions {
			switch d := def.(type) {
			case *module.TypeDef:
				if _, isSeq := d.Syntax.(*module.TypeSyntaxSequence); isSeq {
					continue
				}
				typ, ok := ctx.resolveTypeForModule(mod, d.Name)
				if !ok {
					continue
				}
				base := typ.EffectiveBase()
				sizes, ranges := extractConstraints(d.Syntax)
				checkRangeList(ctx, ranges, base, false, d.Name, mod, d.Span)
				checkRangeList(ctx, sizes, base, true, d.Name, mod, d.Span)

			case *module.ObjectType:
				sizes, ranges := extractConstraints(d.Syntax)
				if len(sizes) == 0 && len(ranges) == 0 {
					continue
				}
				resolved := ctx.lookupObject(mod, d.Name)
				if resolved == nil || resolved.Type() == nil {
					continue
				}
				base := resolved.Type().EffectiveBase()
				checkRangeList(ctx, ranges, base, false, d.Name, mod, d.Span)
				checkRangeList(ctx, sizes, base, true, d.Name, mod, d.Span)
			}
		}
	}
}

// checkRangeList validates a slice of ranges for internal consistency
// and basetype conformance.
func checkRangeList(ctx *resolverContext, ranges []Range, base BaseType, isSize bool, name string, mod *module.Module, span types.Span) {
	if len(ranges) == 0 {
		return
	}

	// SIZE only applies to OCTET STRING and Opaque (which is OCTET STRING-based).
	if isSize && base != BaseOctetString && base != BaseOpaque {
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
	if !isSize && (base == BaseCounter32 || base == BaseCounter64) {
		ctx.EmitDiagnostic(types.DiagCounterRangeIllegal,
			mod, span,
			fmt.Sprintf("%q: range constraint illegal for Counter type", name))
		return
	}

	// TimeTicks may not be sub-typed (RFC 2578 s7.1.8).
	if !isSize && base == BaseTimeTicks {
		ctx.EmitDiagnostic(types.DiagTimeticksRangeIllegal,
			mod, span,
			fmt.Sprintf("%q: range constraint illegal for TimeTicks type", name))
		return
	}

	var boundsMin, boundsMax int64
	var hasBounds bool
	if isSize {
		boundsMin, boundsMax, hasBounds = 0, 65535, true
	} else {
		boundsMin, boundsMax, hasBounds = basetypeBounds(base)
	}

	for i, r := range ranges {
		// Exchanged limits (min > max).
		if r.Min > r.Max {
			ctx.EmitDiagnostic(types.DiagRangeExchanged,
				mod, span,
				fmt.Sprintf("%q: range %s has exchanged limits", name, r.String()))
		}

		// Bounds checking against basetype.
		if hasBounds {
			checkRangeBound(ctx, r.Min, boundsMin, boundsMax, name, "lower", mod, span)
			checkRangeBound(ctx, r.Max, boundsMin, boundsMax, name, "upper", mod, span)
		}

		// Multi-range checks against previous range.
		if i > 0 {
			prev := ranges[i-1]
			if r.Min < prev.Min {
				ctx.EmitDiagnostic(types.DiagRangeAscending,
					mod, span,
					fmt.Sprintf("%q: ranges not in ascending order", name))
			}
			if r.Min <= prev.Max {
				ctx.EmitDiagnostic(types.DiagRangeOverlap,
					mod, span,
					fmt.Sprintf("%q: range %s overlaps with %s", name, r.String(), prev.String()))
			}
		}
	}
}

// checkRangeBound emits a range-bounds diagnostic if v falls outside
// boundsMin..boundsMax. Values equal to MinInt64 or MaxInt64 are skipped
// because they represent the MIN/MAX keywords.
func checkRangeBound(ctx *resolverContext, v, boundsMin, boundsMax int64, name, which string, mod *module.Module, span types.Span) {
	if v == math.MinInt64 || v == math.MaxInt64 {
		return
	}
	if v < boundsMin || v > boundsMax {
		ctx.EmitDiagnostic(types.DiagRangeBounds,
			mod, span,
			fmt.Sprintf("%q: range %s bound %d exceeds basetype", name, which, v))
	}
}

// basetypeBounds returns the valid min/max range for a basetype.
func basetypeBounds(base BaseType) (min, max int64, ok bool) {
	switch base {
	case BaseInteger32:
		return math.MinInt32, math.MaxInt32, true
	case BaseUnsigned32, BaseGauge32, BaseTimeTicks, BaseCounter32:
		return 0, math.MaxUint32, true
	case BaseCounter64:
		// Bounds are never used (counter range/DEFVAL checks reject earlier),
		// but Counter64 must return ok=true so isNumericBase reports correctly.
		return 0, math.MaxInt64, true
	default:
		return 0, 0, false
	}
}

// isNumericBase reports whether the base type supports value range constraints.
func isNumericBase(base BaseType) bool {
	_, _, ok := basetypeBounds(base)
	return ok
}

// isLegalIndexBasetype reports whether a base type is legal for INDEX
// elements per RFC 2578 section 7.7.
func isLegalIndexBasetype(base BaseType) bool {
	switch base {
	case BaseInteger32, BaseUnsigned32, BaseGauge32, BaseTimeTicks,
		BaseIpAddress, BaseOctetString, BaseOpaque, BaseBits, BaseObjectIdentifier:
		return true
	default:
		return false
	}
}

// maxSizeFromRanges returns the maximum Max value across all size ranges,
// or -1 if sizes is empty.
func maxSizeFromRanges(sizes []Range) int64 {
	if len(sizes) == 0 {
		return -1
	}
	m := sizes[0].Max
	for _, r := range sizes[1:] {
		if r.Max > m {
			m = r.Max
		}
	}
	return m
}

// indexElementSubIds returns the maximum number of sub-identifiers an index
// element contributes to an instance OID. Returns (count, true) if computable,
// or (0, false) if the encoding or type constraints are insufficient.
func indexElementSubIds(entry IndexEntry) (int, bool) {
	obj := entry.Object
	if obj == nil {
		return 0, false
	}
	switch entry.Encoding {
	case IndexEncodingInteger, IndexEncodingIpAddress, IndexEncodingFixedString:
		return entry.FixedSize()
	case IndexEncodingLengthPrefixed:
		t := obj.Type()
		if t == nil {
			return 0, false
		}
		if t.EffectiveBase() == BaseObjectIdentifier {
			return 128 + 1, true
		}
		m := maxSizeFromRanges(obj.sizes)
		if m < 0 {
			return 0, false
		}
		return int(m) + 1, true
	case IndexEncodingImplied:
		t := obj.Type()
		if t == nil {
			return 0, false
		}
		if t.EffectiveBase() == BaseObjectIdentifier {
			return 128, true
		}
		m := maxSizeFromRanges(obj.sizes)
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
				if idxObj.Access() != AccessNotAccessible {
					ctx.EmitDiagnostic(types.DiagIndexAccessible,
						ref.mod, obj.Span,
						fmt.Sprintf("INDEX %q of %q should be not-accessible in SMIv2", item.Object, obj.Name))
				}
			case types.LanguageSMIv1:
				if idxObj.Access() == AccessNotAccessible {
					ctx.EmitDiagnostic(types.DiagIndexNotAccessible,
						ref.mod, obj.Span,
						fmt.Sprintf("INDEX %q of %q should be accessible in SMIv1", item.Object, obj.Name))
				}
			case types.LanguageUnknown:
				// Cannot validate index access without known language.
			}

			// INDEX elements should not have DEFVAL (RFC 2578 s7.7).
			if !idxObj.DefaultValue().IsZero() {
				ctx.EmitDiagnostic(types.DiagIndexDefval,
					ref.mod, obj.Span,
					fmt.Sprintf("INDEX %q of %q has a DEFVAL", item.Object, obj.Name))
			}

			t := idxObj.Type()
			if t == nil {
				continue
			}
			base := t.EffectiveBase()
			if base == BaseCounter32 {
				// Counter32 is forbidden per RFC 2578 s7.7 but used in real MIBs.
				// Emit warning but continue - Counter32 is functionally valid for indexing.
				ctx.EmitDiagnostic(types.DiagIndexCounterIllegal,
					ref.mod, obj.Span,
					fmt.Sprintf("INDEX %q of %q has counter base type", item.Object, obj.Name))
			} else if !isLegalIndexBasetype(base) {
				ctx.EmitDiagnostic(types.DiagIndexIllegalBasetype,
					ref.mod, obj.Span,
					fmt.Sprintf("INDEX %q of %q has illegal base type %s", item.Object, obj.Name, base.String()))
				continue
			}
			// OCTET STRING/Opaque index elements must have a SIZE constraint
			// so the encoding length is bounded.
			if base == BaseOctetString || base == BaseOpaque {
				if len(idxObj.EffectiveSizes()) == 0 {
					ctx.EmitDiagnostic(types.DiagIndexElementNoSize,
						ref.mod, obj.Span,
						fmt.Sprintf("INDEX %q of %q has no SIZE restriction", item.Object, obj.Name))
				}
				continue
			}
			if base != BaseInteger32 {
				continue
			}

			ranges := idxObj.EffectiveRanges()
			enums := idxObj.EffectiveEnums()

			// Enum-valued indexes: check for negative enum values.
			if len(enums) > 0 {
				for _, e := range enums {
					if e.Value < 0 {
						ctx.EmitDiagnostic(types.DiagIndexNegativeRange,
							ref.mod, obj.Span,
							fmt.Sprintf("INDEX %q of %q has negative enumeration value %q", item.Object, obj.Name, e.Label))
						break
					}
				}
				continue
			}

			// No range restriction on an integer index.
			if len(ranges) == 0 {
				ctx.EmitDiagnostic(types.DiagIndexIntegerNoRange,
					ref.mod, obj.Span,
					fmt.Sprintf("INDEX %q of %q has no range restriction", item.Object, obj.Name))
				continue
			}

			// Range includes negative values.
			for _, r := range ranges {
				if r.Min < 0 {
					ctx.EmitDiagnostic(types.DiagIndexNegativeRange,
						ref.mod, obj.Span,
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
// 128 sub-identifier maximum from RFC 2578 section 3.5. The instance OID
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
	for _, entry := range resolvedObj.index {
		n, ok := indexElementSubIds(entry)
		if !ok {
			return // can't compute, skip the check
		}
		totalLen += n
	}

	if totalLen > 128 {
		excess := totalLen - 128
		ctx.EmitDiagnostic(types.DiagIndexExceedsTooLarge,
			ref.mod, ref.obj.Span,
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
		if base == BaseCounter32 || base == BaseCounter64 {
			ctx.EmitDiagnostic(types.DiagCounterDefvalIllegal,
				ref.mod, obj.Span,
				fmt.Sprintf("%q: DEFVAL not allowed for counter type", obj.Name))
			continue
		}

		ranges := resolved.EffectiveRanges()
		enums := resolved.EffectiveEnums()
		bits := resolved.EffectiveBits()

		switch dv.Kind() {
		case DefValKindInt:
			v, _ := DefValAs[int64](dv)
			checkDefvalNumeric(ctx, ref.mod, obj, v, 0, false, base, ranges, enums)
		case DefValKindUint:
			v, _ := DefValAs[uint64](dv)
			checkDefvalNumeric(ctx, ref.mod, obj, 0, v, true, base, ranges, enums)
		case DefValKindEnum:
			label, _ := DefValAs[string](dv)
			if len(enums) > 0 {
				if _, ok := findNamedValue(enums, label); !ok {
					ctx.EmitDiagnostic(types.DiagDefvalEnum,
						ref.mod, obj.Span,
						fmt.Sprintf("%q: DEFVAL enum label %q not defined in type", obj.Name, label))
				}
			}
		case DefValKindBits:
			labels, _ := DefValAs[[]string](dv)
			for _, label := range labels {
				if _, ok := findNamedValue(bits, label); !ok {
					ctx.EmitDiagnostic(types.DiagDefvalBits,
						ref.mod, obj.Span,
						fmt.Sprintf("%q: DEFVAL BITS label %q not defined in type", obj.Name, label))
				}
			}
		case DefValKindUnset, DefValKindString, DefValKindBytes, DefValKindOID:
			// No validation for these kinds.
		}
	}
}

// checkDefvalNumeric validates a numeric DEFVAL value against basetype limits,
// RANGE constraints, and enum membership. Works for both signed and unsigned
// values: signed values have isUnsigned=false and uval=0, unsigned values have
// isUnsigned=true and ival=0.
func checkDefvalNumeric(ctx *resolverContext, mod *module.Module, obj *module.ObjectType, ival int64, uval uint64, isUnsigned bool, base BaseType, ranges []Range, enums []NamedValue) {
	// Check basetype limits.
	switch base {
	case BaseInteger32:
		if isUnsigned {
			if uval > math.MaxInt32 {
				ctx.EmitDiagnostic(types.DiagDefvalBasetype,
					mod, obj.Span,
					fmt.Sprintf("%q: DEFVAL %d exceeds Integer32 range", obj.Name, uval))
			}
		} else {
			if ival < math.MinInt32 || ival > math.MaxInt32 {
				ctx.EmitDiagnostic(types.DiagDefvalBasetype,
					mod, obj.Span,
					fmt.Sprintf("%q: DEFVAL %d exceeds Integer32 range", obj.Name, ival))
			}
		}
	case BaseUnsigned32, BaseGauge32, BaseCounter32, BaseTimeTicks:
		if isUnsigned {
			if uval > math.MaxUint32 {
				ctx.EmitDiagnostic(types.DiagDefvalBasetype,
					mod, obj.Span,
					fmt.Sprintf("%q: DEFVAL %d exceeds unsigned32 range", obj.Name, uval))
			}
		} else {
			if ival < 0 || ival > math.MaxUint32 {
				ctx.EmitDiagnostic(types.DiagDefvalBasetype,
					mod, obj.Span,
					fmt.Sprintf("%q: DEFVAL %d exceeds unsigned32 range", obj.Name, ival))
			}
		}
	}

	// Check RANGE constraints.
	if len(ranges) > 0 {
		var inRange bool
		if isUnsigned {
			inRange = uvalueInRanges(uval, ranges)
		} else {
			inRange = valueInRanges(ival, ranges)
		}
		if !inRange {
			v := any(ival)
			if isUnsigned {
				v = uval
			}
			ctx.EmitDiagnostic(types.DiagDefvalRange,
				mod, obj.Span,
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
				mod, obj.Span,
				fmt.Sprintf("%q: DEFVAL %v does not match any enumeration value", obj.Name, v))
		}
	}
}

func valueInRanges(v int64, ranges []Range) bool {
	for _, r := range ranges {
		if v >= r.Min && v <= r.Max {
			return true
		}
	}
	return false
}

func uvalueInRanges(v uint64, ranges []Range) bool {
	for _, r := range ranges {
		// Range Min/Max are int64. A uint64 value can only fall within a range
		// if it fits in int64 (i.e. v <= MaxInt64) and the range bound is non-negative.
		if r.Max < 0 {
			continue
		}
		min := uint64(0)
		if r.Min > 0 {
			min = uint64(r.Min)
		}
		if v > math.MaxInt64 {
			// v exceeds int64 range, so it can't match any Range (which uses int64).
			continue
		}
		if v >= min && int64(v) <= r.Max {
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
		columns := rowNode.sortedChildren()

		// Build lookup maps.
		columnByName := make(map[string]*Node, len(columns))
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
	for _, def := range mod.Definitions {
		if td, ok := def.(*module.TypeDef); ok && td.Name == name {
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
func sequenceTypesCompatible(fieldType, colType string, colBase BaseType) bool {
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
	if (fieldNorm == "Integer32") && colBase == BaseInteger32 {
		return true
	}
	// OCTET STRING/BITS in SEQUENCE is compatible with BITS-based columns
	// (BITS is encoded as OCTET STRING at wire level).
	if (fieldType == "OCTET STRING" || fieldType == "BITS") && colBase == BaseBits {
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
				mod, obj.Span,
				fmt.Sprintf("%q: invalid access %s in SMIv1", obj.Name, obj.Access.String()))
		}
		if obj.Access == types.AccessWriteOnly {
			ctx.EmitDiagnostic(types.DiagAccessWriteOnlySMIv1,
				mod, obj.Span,
				fmt.Sprintf("%q: write-only is valid but discouraged in SMIv1", obj.Name))
		}
	case types.LanguageSMIv2:
		if obj.Access == types.AccessWriteOnly {
			ctx.EmitDiagnostic(types.DiagAccessWriteOnlySMIv2,
				mod, obj.Span,
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
				mod, obj.Span,
				fmt.Sprintf("%q: MAX-ACCESS is SMIv2 style, use ACCESS in SMIv1", obj.Name))
		}
	case types.LanguageSMIv2:
		if obj.AccessKeyword == types.AccessKeywordAccess {
			ctx.EmitDiagnostic(types.DiagAccessInSMIv2,
				mod, obj.Span,
				fmt.Sprintf("%q: ACCESS is SMIv1 style, use MAX-ACCESS in SMIv2", obj.Name))
		}
	case types.LanguageUnknown:
		// Cannot validate version-specific keyword rules without known language.
	}
}

// checkKindAccess validates access values for specific node kinds:
// tables and rows must be not-accessible, scalars must not be read-create.
func checkKindAccess(ctx *resolverContext, mod *module.Module, obj *module.ObjectType, node *Node) {
	switch node.Kind() {
	case KindTable:
		if obj.Access != types.AccessNotAccessible {
			ctx.EmitDiagnostic(types.DiagAccessTableIllegal,
				mod, obj.Span,
				fmt.Sprintf("%q: table must be not-accessible", obj.Name))
		}
	case KindRow:
		if obj.Access != types.AccessNotAccessible {
			ctx.EmitDiagnostic(types.DiagAccessRowIllegal,
				mod, obj.Span,
				fmt.Sprintf("%q: row must be not-accessible", obj.Name))
		}
	case KindScalar:
		if obj.Access == types.AccessReadCreate {
			ctx.EmitDiagnostic(types.DiagScalarNotCreatable,
				mod, obj.Span,
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
	if base == BaseCounter32 || base == BaseCounter64 {
		ctx.EmitDiagnostic(types.DiagAccessCounterIllegal,
			mod, obj.Span,
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
		for _, def := range mod.Definitions {
			var status types.Status
			var hasStatus bool
			switch d := def.(type) {
			case *module.ObjectType:
				status, hasStatus = d.Status, true
			case *module.ObjectIdentity:
				status, hasStatus = d.Status, true
			case *module.Notification:
				// TRAP-TYPE has no STATUS clause; the synthetic default
				// would produce false positives.
				if d.IsTrap() {
					continue
				}
				status, hasStatus = d.Status, true
			case *module.TypeDef:
				// Only textual conventions have a STATUS clause; plain
				// type assignments and SEQUENCE defs default to current.
				if !d.IsTextualConvention {
					continue
				}
				status, hasStatus = d.Status, true
			case *module.ObjectGroup:
				status, hasStatus = d.Status, true
			case *module.NotificationGroup:
				status, hasStatus = d.Status, true
			case *module.ModuleCompliance:
				status, hasStatus = d.Status, true
			case *module.AgentCapabilities:
				status, hasStatus = d.Status, true
			}
			if !hasStatus {
				continue
			}
			switch mod.Language {
			case types.LanguageSMIv1:
				if status == types.StatusCurrent {
					ctx.EmitDiagnostic(types.DiagStatusInvalidSMIv1,
						mod, def.DefinitionSpan(),
						fmt.Sprintf("%q: invalid status current in SMIv1", def.DefinitionName()))
				}
			case types.LanguageSMIv2:
				if status == types.StatusMandatory || status == types.StatusOptional {
					ctx.EmitDiagnostic(types.DiagStatusInvalidSMIv2,
						mod, def.DefinitionSpan(),
						fmt.Sprintf("%q: invalid status %s in SMIv2", def.DefinitionName(), status.String()))
				}
			case types.LanguageUnknown:
				// Cannot validate version-specific status rules without known language.
			}
		}
	}
}

// checkCapabilitiesStatus validates that AGENT-CAPABILITIES STATUS is
// either current or obsolete (RFC 2580 s6.2).
func checkCapabilitiesStatus(ctx *resolverContext) {
	for _, mod := range ctx.modules {
		if module.IsBaseModule(mod.Name) {
			continue
		}
		for _, def := range mod.Definitions {
			ac, ok := def.(*module.AgentCapabilities)
			if !ok {
				continue
			}
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
		case StatusDeprecated:
			ctx.EmitDiagnostic(types.DiagTypeStatusDeprecated,
				ref.mod, ref.obj.Span,
				fmt.Sprintf("type %q used by %q is deprecated", t.Name(), ref.obj.Name))
		case StatusObsolete:
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
