package mib

import (
	"fmt"
	"math"
	"strings"

	"github.com/golangsnmp/gomib/internal/module"
	"github.com/golangsnmp/gomib/internal/types"
)

// checkGroupMembership verifies that every accessible scalar, column, and
// notification in a module appears in at least one conformance group defined
// in the same module. Only modules that define groups are checked.
func checkGroupMembership(ctx *resolverContext, objRefs []objectTypeRef) {
	// Build per-module sets of grouped nodes.
	type moduleGroupInfo struct {
		hasObjectGroup       bool
		hasNotificationGroup bool
		groupedNodes         map[*Node]bool
	}
	modGroups := make(map[*module.Module]*moduleGroupInfo)

	for _, mod := range ctx.modules {
		if module.IsBaseModule(mod.Name) {
			continue
		}
		resolvedMod := ctx.moduleToResolved[mod]
		if resolvedMod == nil {
			continue
		}
		for _, grp := range resolvedMod.Groups() {
			info := modGroups[mod]
			if info == nil {
				info = &moduleGroupInfo{groupedNodes: make(map[*Node]bool)}
				modGroups[mod] = info
			}
			if grp.IsNotificationGroup() {
				info.hasNotificationGroup = true
			} else {
				info.hasObjectGroup = true
			}
			for _, member := range grp.Members() {
				info.groupedNodes[member] = true
			}
		}
	}

	// Check accessible scalars and columns.
	for _, ref := range objRefs {
		info := modGroups[ref.mod]
		if info == nil || !info.hasObjectGroup {
			continue
		}
		node, ok := ctx.LookupNodeForModule(ref.mod, ref.obj.Name)
		if !ok {
			continue
		}
		kind := node.Kind()
		if kind != KindScalar && kind != KindColumn {
			continue
		}
		if ref.obj.Access == types.AccessNotAccessible {
			continue
		}
		if !info.groupedNodes[node] {
			ctx.EmitDiagnostic(types.DiagGroupMembership,
				ref.mod, ref.obj.Span,
				fmt.Sprintf("%q is not in any OBJECT-GROUP", ref.obj.Name))
		}
	}

	// Check notifications.
	for _, ref := range collectNotificationRefs(ctx) {
		info := modGroups[ref.mod]
		if info == nil || !info.hasNotificationGroup {
			continue
		}
		node, ok := ctx.LookupNodeForModule(ref.mod, ref.notif.Name)
		if !ok {
			continue
		}
		if !info.groupedNodes[node] {
			ctx.EmitDiagnostic(types.DiagGroupMembership,
				ref.mod, ref.notif.DefinitionSpan(),
				fmt.Sprintf("%q is not in any NOTIFICATION-GROUP", ref.notif.Name))
		}
	}
}

// checkComplianceStatus emits DiagComplianceGroupStatus when a mandatory or
// optional group's status exceeds the MODULE-COMPLIANCE's status, and
// DiagComplianceObjectStatus when a refined object's status exceeds it.
// SMIv1 statuses are skipped since they aren't comparable on the same scale.
func checkComplianceStatus(ctx *resolverContext) {
	for _, ref := range collectComplianceRefs(ctx) {
		comp := ref.comp
		if comp.Status.IsSMIv1() {
			continue
		}
		for _, cm := range comp.Modules {
			// Check mandatory groups.
			for _, groupName := range cm.MandatoryGroups {
				checkComplianceGroupStatus(ctx, ref.mod, comp, cm.ModuleName, groupName)
			}
			// Check optional (GROUP) groups.
			for _, cg := range cm.Groups {
				checkComplianceGroupStatus(ctx, ref.mod, comp, cm.ModuleName, cg.Group)
			}
			// Check refined objects.
			for _, co := range cm.Objects {
				checkComplianceObjectStatus(ctx, ref.mod, comp, cm.ModuleName, co.Object)
			}
		}
	}
}

func checkComplianceGroupStatus(ctx *resolverContext, mod *module.Module, comp *module.ModuleCompliance, moduleName, groupName string) {
	node := lookupComplianceMember(ctx, mod, moduleName, groupName)
	if node == nil {
		return
	}
	grp := node.Group()
	if grp == nil {
		return
	}
	gs := grp.Status()
	if gs.IsSMIv1() {
		return
	}
	if gs > comp.Status {
		ctx.EmitDiagnostic(types.DiagComplianceGroupStatus,
			mod, comp.DefinitionSpan(),
			fmt.Sprintf("%s compliance %q references %s group %q", comp.Status, comp.Name, gs, groupName))
	}
}

func checkComplianceObjectStatus(ctx *resolverContext, mod *module.Module, comp *module.ModuleCompliance, moduleName, objectName string) {
	node := lookupComplianceMember(ctx, mod, moduleName, objectName)
	if node == nil {
		return
	}
	ms, ok := memberNodeStatus(node)
	if !ok || ms.IsSMIv1() {
		return
	}
	if ms > comp.Status {
		ctx.EmitDiagnostic(types.DiagComplianceObjectStatus,
			mod, comp.DefinitionSpan(),
			fmt.Sprintf("%s compliance %q references %s object %q", comp.Status, comp.Name, ms, objectName))
	}
}

// lookupComplianceMember resolves a node referenced from a MODULE-COMPLIANCE
// MODULE clause. When moduleName is non-empty, the lookup targets that specific
// module; otherwise the compliance's own module is used.
func lookupComplianceMember(ctx *resolverContext, compMod *module.Module, moduleName, name string) *Node {
	if moduleName != "" {
		node, ok := ctx.LookupNodeInModule(moduleName, name)
		if ok {
			return node
		}
	}
	node, ok := lookupMemberNode(ctx, compMod, name)
	if ok {
		return node
	}
	return nil
}

// checkComplianceStructure validates MODULE-COMPLIANCE structural constraints:
// - compliance-group-invalid: group both mandatory and optional
// - refinement-exists: duplicate OBJECT refinement
// - optional-group-exists: duplicate GROUP clause
// - refinement-not-listed: refined object not in any listed group
func checkComplianceStructure(ctx *resolverContext) {
	for _, ref := range collectComplianceRefs(ctx) {
		comp := ref.comp
		for _, cm := range comp.Modules {
			checkComplianceDuplicates(ctx, ref.mod, comp, &cm)
			checkRefinementListed(ctx, ref.mod, comp, &cm)
		}
	}
}

func checkComplianceDuplicates(ctx *resolverContext, mod *module.Module, comp *module.ModuleCompliance, cm *module.ComplianceModule) {
	// Build mandatory group set.
	mandatory := make(map[string]bool, len(cm.MandatoryGroups))
	for _, g := range cm.MandatoryGroups {
		mandatory[g] = true
	}

	// Check optional groups for duplicates and overlap with mandatory.
	optionalSeen := make(map[string]bool, len(cm.Groups))
	for _, cg := range cm.Groups {
		if mandatory[cg.Group] {
			ctx.EmitDiagnostic(types.DiagComplianceGroupInvalid,
				mod, comp.DefinitionSpan(),
				fmt.Sprintf("group %q is both mandatory and optional in %q", cg.Group, comp.Name))
		}
		if optionalSeen[cg.Group] {
			ctx.EmitDiagnostic(types.DiagOptionalGroupExists,
				mod, comp.DefinitionSpan(),
				fmt.Sprintf("duplicate optional group %q in %q", cg.Group, comp.Name))
		}
		optionalSeen[cg.Group] = true
	}

	// Check for duplicate object refinements.
	refinementSeen := make(map[string]bool, len(cm.Objects))
	for _, co := range cm.Objects {
		if refinementSeen[co.Object] {
			ctx.EmitDiagnostic(types.DiagRefinementExists,
				mod, comp.DefinitionSpan(),
				fmt.Sprintf("duplicate refinement for %q in %q", co.Object, comp.Name))
		}
		refinementSeen[co.Object] = true
	}
}

// checkRefinementListed verifies that each refined object in a MODULE-COMPLIANCE
// MODULE clause is a member of at least one mandatory or optional group.
func checkRefinementListed(ctx *resolverContext, mod *module.Module, comp *module.ModuleCompliance, cm *module.ComplianceModule) {
	if len(cm.Objects) == 0 {
		return
	}

	// Collect all group member names from mandatory and optional groups.
	memberNames := make(map[string]bool)
	for _, groupName := range cm.MandatoryGroups {
		collectGroupMemberNames(ctx, mod, cm.ModuleName, groupName, memberNames)
	}
	for _, cg := range cm.Groups {
		collectGroupMemberNames(ctx, mod, cm.ModuleName, cg.Group, memberNames)
	}

	for _, co := range cm.Objects {
		if !memberNames[co.Object] {
			ctx.EmitDiagnostic(types.DiagRefinementNotListed,
				mod, comp.DefinitionSpan(),
				fmt.Sprintf("refined object %q not in any mandatory or optional group of %q", co.Object, comp.Name))
		}
	}
}

// collectGroupMemberNames looks up a group and adds its member names to the set.
func collectGroupMemberNames(ctx *resolverContext, mod *module.Module, moduleName, groupName string, names map[string]bool) {
	node := lookupComplianceMember(ctx, mod, moduleName, groupName)
	if node == nil {
		return
	}
	grp := node.Group()
	if grp == nil {
		return
	}
	for _, member := range grp.Members() {
		names[member.Name()] = true
	}
}

// checkGroupMemberLocality validates that OBJECT-GROUP and NOTIFICATION-GROUP
// members are defined in the same module as the group (RFC 2580 sections 3.1, 4.1).
func checkGroupMemberLocality(ctx *resolverContext) {
	for _, mod := range ctx.modules {
		for _, def := range mod.Definitions {
			switch d := def.(type) {
			case *module.ObjectGroup:
				checkMemberLocality(ctx, mod, d.DefinitionSpan(), d.Objects)
			case *module.NotificationGroup:
				checkMemberLocality(ctx, mod, d.DefinitionSpan(), d.Notifications)
			}
		}
	}
}

func checkMemberLocality(ctx *resolverContext, mod *module.Module, span types.Span, members []string) {
	localSymbols := ctx.moduleSymbolToNode[mod]
	for _, memberName := range members {
		if localSymbols == nil || localSymbols[memberName] == nil {
			ctx.EmitDiagnostic(types.DiagComplianceMemberNotLocal,
				mod, span,
				fmt.Sprintf("group member %q is not defined in module %q", memberName, mod.Name))
		}
	}
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
	case IndexEncodingInteger:
		return 1, true
	case IndexEncodingIpAddress:
		return 4, true
	case IndexEncodingFixedString:
		if isFixedSize(obj.sizes) {
			return int(obj.sizes[0].Max), true
		}
		return 0, false
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
		node, ok := ctx.LookupNodeForModule(ref.mod, obj.Name)
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
			node, ok := ctx.LookupNodeForModule(mod, def.DefinitionName())
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
			idxObj := ctx.lookupObjectInModuleScope(ref.mod, item.Object)
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
	rowNode, ok := ctx.LookupNodeForModule(ref.mod, ref.obj.Name)
	if !ok {
		return
	}
	rowOID := rowNode.OID()
	if rowOID == nil {
		return
	}

	resolvedObj := ctx.lookupObjectInModuleScope(ref.mod, ref.obj.Name)
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

		resolved := ctx.lookupObjectInModuleScope(ref.mod, obj.Name)
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

	parentType, ok := ctx.LookupTypeForModule(mod, enumSyntax.Base)
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
				typ, ok := ctx.LookupTypeForModule(mod, d.Name)
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
				resolved := ctx.lookupObjectInModuleScope(mod, d.Name)
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
		rowNode, ok := ctx.LookupNodeForModule(ref.mod, obj.Name)
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
	if imports := ctx.moduleImports[mod]; imports != nil {
		if srcMod := imports[name]; srcMod != nil {
			return getSequenceTypeDef(srcMod, name)
		}
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

// checkAccessAndStatus validates access values, access keywords, and status
// values per SMI version, and checks kind-specific access constraints.
func checkAccessAndStatus(ctx *resolverContext, objRefs []objectTypeRef) {
	for _, ref := range objRefs {
		obj := ref.obj
		mod := ref.mod
		if module.IsBaseModule(mod.Name) {
			continue
		}

		node, ok := ctx.LookupNodeForModule(mod, obj.Name)
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
	resolved := ctx.lookupObjectInModuleScope(mod, obj.Name)
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
		resolved := ctx.lookupObjectInModuleScope(ref.mod, ref.obj.Name)
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

		node, ok := ctx.LookupNodeForModule(ref.mod, ref.notif.Name)
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

// checkGroupUnreferenced flags OBJECT-GROUP and NOTIFICATION-GROUP definitions
// that are never referenced in any MODULE-COMPLIANCE (as mandatory or optional
// groups). Only checks within the loaded module set.
func checkGroupUnreferenced(ctx *resolverContext) {
	// Collect all group names referenced by compliance and capabilities modules.
	referencedGroups := make(map[string]struct{})
	for _, mod := range ctx.modules {
		for _, def := range mod.Definitions {
			switch d := def.(type) {
			case *module.ModuleCompliance:
				for _, cm := range d.Modules {
					for _, name := range cm.MandatoryGroups {
						referencedGroups[name] = struct{}{}
					}
					for _, grp := range cm.Groups {
						referencedGroups[grp.Group] = struct{}{}
					}
				}
			case *module.AgentCapabilities:
				for _, sup := range d.Supports {
					for _, name := range sup.Includes {
						referencedGroups[name] = struct{}{}
					}
				}
			}
		}
	}

	// Check each group definition.
	for _, mod := range ctx.modules {
		if module.IsBaseModule(mod.Name) {
			continue
		}
		for _, def := range mod.Definitions {
			switch d := def.(type) {
			case *module.ObjectGroup:
				if _, ok := referencedGroups[d.Name]; !ok {
					ctx.EmitDiagnostic(types.DiagGroupUnreferenced,
						mod, d.Span,
						fmt.Sprintf("%q: OBJECT-GROUP not referenced in any compliance module", d.Name))
				}
			case *module.NotificationGroup:
				if _, ok := referencedGroups[d.Name]; !ok {
					ctx.EmitDiagnostic(types.DiagGroupUnreferenced,
						mod, d.Span,
						fmt.Sprintf("%q: NOTIFICATION-GROUP not referenced in any compliance module", d.Name))
				}
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

// checkRowStatusDefaults checks RowStatus objects for valid DEFVAL values and
// access levels. RowStatus DEFVAL must be 1-3 (active, notInService, notReady),
// not 4-6 (createAndGo, createAndWait, destroy). RowStatus access must be
// read-create (SMIv2) or read-write (SMIv1).
func checkRowStatusDefaults(ctx *resolverContext, objRefs []objectTypeRef) {
	rowStatusType := lookupWellKnownTC(ctx, "RowStatus")
	if rowStatusType == nil {
		return
	}

	rowStatusActionValues := map[int64]string{
		4: "createAndGo", 5: "createAndWait", 6: "destroy",
	}

	for _, ref := range objRefs {
		resolved := ctx.lookupObjectInModuleScope(ref.mod, ref.obj.Name)
		if resolved == nil {
			continue
		}
		t := resolved.Type()
		if t != rowStatusType {
			continue
		}

		// Check access: must be read-create (or read-write in SMIv1).
		access := ref.obj.Access
		if ref.mod.Language == types.LanguageSMIv2 {
			if access != types.AccessReadCreate {
				ctx.EmitDiagnostic(types.DiagRowStatusAccess,
					ref.mod, ref.obj.Span,
					fmt.Sprintf("%q: RowStatus should have MAX-ACCESS read-create, has %s", ref.obj.Name, access))
			}
		} else if access != types.AccessReadWrite {
			ctx.EmitDiagnostic(types.DiagRowStatusAccess,
				ref.mod, ref.obj.Span,
				fmt.Sprintf("%q: RowStatus should have ACCESS read-write, has %s", ref.obj.Name, access))
		}

		// Check DEFVAL.
		dv := resolved.DefaultValue()
		if dv.IsZero() {
			continue
		}
		val, ok := defvalAsInt64(dv, rowStatusActionValues)
		if !ok {
			continue
		}
		if name, illegal := rowStatusActionValues[val]; illegal {
			ctx.EmitDiagnostic(types.DiagRowStatusDefault,
				ref.mod, ref.obj.Span,
				fmt.Sprintf("%q: RowStatus DEFVAL %s(%d) is an action value, must be active(1), notInService(2), or notReady(3)", ref.obj.Name, name, val))
		}
	}
}

// checkStorageTypeDefaults checks StorageType objects for valid DEFVAL values.
// StorageType DEFVAL must be 1-3 (other, volatile, nonVolatile), not 4-5
// (permanent, readOnly).
func checkStorageTypeDefaults(ctx *resolverContext, objRefs []objectTypeRef) {
	storageType := lookupWellKnownTC(ctx, "StorageType")
	if storageType == nil {
		return
	}

	illegalValues := map[int64]string{
		4: "permanent", 5: "readOnly",
	}

	for _, ref := range objRefs {
		resolved := ctx.lookupObjectInModuleScope(ref.mod, ref.obj.Name)
		if resolved == nil {
			continue
		}
		t := resolved.Type()
		if t != storageType {
			continue
		}
		dv := resolved.DefaultValue()
		if dv.IsZero() {
			continue
		}
		val, ok := defvalAsInt64(dv, illegalValues)
		if !ok {
			continue
		}
		if name, illegal := illegalValues[val]; illegal {
			ctx.EmitDiagnostic(types.DiagStorageTypeDefault,
				ref.mod, ref.obj.Span,
				fmt.Sprintf("%q: StorageType DEFVAL %s(%d) is not a valid default, must be other(1), volatile(2), or nonVolatile(3)", ref.obj.Name, name, val))
		}
	}
}

// defvalAsInt64 extracts a numeric value from a DefVal, mapping enum labels
// via the provided name map. Returns the value and true if extraction
// succeeded, false otherwise.
func defvalAsInt64(dv DefVal, labelToValue map[int64]string) (int64, bool) {
	switch dv.Kind() {
	case DefValKindInt:
		v, _ := DefValAs[int64](dv)
		return v, true
	case DefValKindUint:
		uv, _ := DefValAs[uint64](dv)
		return int64(uv), true
	case DefValKindEnum:
		label, _ := DefValAs[string](dv)
		for v, name := range labelToValue {
			if name == label {
				return v, true
			}
		}
		return 0, false
	default:
		return 0, false
	}
}

// checkTAddressTDomain verifies that column objects derived from TAddress have
// a sibling column with TDomain type. Only checks column objects within table
// rows.
func checkTAddressTDomain(ctx *resolverContext, objRefs []objectTypeRef) {
	tAddressType := lookupWellKnownTC(ctx, "TAddress")
	tDomainType := lookupWellKnownTC(ctx, "TDomain")
	if tAddressType == nil || tDomainType == nil {
		return
	}

	for _, ref := range objRefs {
		resolved := ctx.lookupObjectInModuleScope(ref.mod, ref.obj.Name)
		if resolved == nil || !resolved.IsColumn() {
			continue
		}
		t := resolved.Type()
		if t == nil || !isDerivedFromType(t, tAddressType) {
			continue
		}
		row := resolved.Row()
		if row == nil {
			continue
		}
		found := false
		for _, col := range row.Columns() {
			ct := col.Type()
			if ct != nil && isDerivedFromType(ct, tDomainType) {
				found = true
				break
			}
		}
		if !found {
			ctx.EmitDiagnostic(types.DiagTAddressTDomain,
				ref.mod, ref.obj.Span,
				fmt.Sprintf("%q: TAddress column has no sibling with TDomain type", ref.obj.Name))
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

// isDerivedFromType walks the parent chain of t and returns true if any
// type in the chain is target.
func isDerivedFromType(t, target *Type) bool {
	for current, depth := t, 0; current != nil && depth < maxTypeChainDepth; current, depth = current.parent, depth+1 {
		if current == target {
			return true
		}
	}
	return false
}

// lookupWellKnownTC looks up a textual convention from SNMPv2-TC by name.
func lookupWellKnownTC(ctx *resolverContext, name string) *Type {
	if ctx.snmpv2TCModule == nil {
		return nil
	}
	t, _ := ctx.lookupTypeInModule(ctx.snmpv2TCModule, name)
	return t
}

// lookupTypeInNamedModule looks up a type from a module identified by name.
// Returns nil if the module is not loaded or the type is not found.
func lookupTypeInNamedModule(ctx *resolverContext, moduleName, typeName string) *Type {
	mods := ctx.moduleIndex[moduleName]
	if len(mods) == 0 {
		return nil
	}
	t, _ := ctx.lookupTypeInModule(mods[0], typeName)
	return t
}

// addressPairingConfig defines the parameters for an address/address-type
// pairing check (InetAddress or TransportAddress patterns).
type addressPairingConfig struct {
	moduleName      string   // source module, e.g. "INET-ADDRESS-MIB"
	addressType     string   // address TC name, e.g. "InetAddress"
	addressTypeType string   // address-type TC name, e.g. "InetAddressType"
	specificTypes   []string // specific variant TC names, e.g. "InetAddressIPv4"
	diagPairing     string   // diagnostic code for missing pairing
	diagSubtyped    string   // diagnostic code for subtyped address-type
	diagSpecific    string   // diagnostic code for specific variant usage
}

var inetAddressConfig = addressPairingConfig{
	moduleName:      "INET-ADDRESS-MIB",
	addressType:     "InetAddress",
	addressTypeType: "InetAddressType",
	specificTypes: []string{
		"InetAddressIPv4",
		"InetAddressIPv6",
		"InetAddressIPv4z",
		"InetAddressIPv6z",
		"InetAddressDNS",
	},
	diagPairing:  types.DiagInetAddressPairing,
	diagSubtyped: types.DiagInetAddressTypeSubtyped,
	diagSpecific: types.DiagInetAddressSpecific,
}

var transportAddressConfig = addressPairingConfig{
	moduleName:      "TRANSPORT-ADDRESS-MIB",
	addressType:     "TransportAddress",
	addressTypeType: "TransportAddressType",
	specificTypes: []string{
		"TransportAddressIPv4",
		"TransportAddressIPv6",
		"TransportAddressIPv4z",
		"TransportAddressIPv6z",
		"TransportAddressLocal",
		"TransportAddressDns",
	},
	diagPairing:  types.DiagTransportAddressPairing,
	diagSubtyped: types.DiagTransportAddressTypeSubtyped,
	diagSpecific: types.DiagTransportAddressSpecific,
}

// checkInetAddressPairing validates InetAddress/InetAddressType pairing per RFC 4001.
func checkInetAddressPairing(ctx *resolverContext, objRefs []objectTypeRef) {
	checkAddressTypePairing(ctx, objRefs, &inetAddressConfig)
}

// checkTransportAddressPairing validates TransportAddress/TransportAddressType pairing per RFC 3419.
func checkTransportAddressPairing(ctx *resolverContext, objRefs []objectTypeRef) {
	checkAddressTypePairing(ctx, objRefs, &transportAddressConfig)
}

// checkAddressTypePairing implements three related checks for address/address-type
// pairing patterns (used by both InetAddress and TransportAddress):
//
//   - Pairing: address column should have a sibling address-type column
//   - Subtyped: address-type with enum refinement should have correspondingly
//     constrained address sibling
//   - Specific: specific address variants should not be used when the generic
//     address type is appropriate
func checkAddressTypePairing(ctx *resolverContext, objRefs []objectTypeRef, cfg *addressPairingConfig) {
	addrType := lookupTypeInNamedModule(ctx, cfg.moduleName, cfg.addressType)
	addrTypeType := lookupTypeInNamedModule(ctx, cfg.moduleName, cfg.addressTypeType)
	if addrType == nil || addrTypeType == nil {
		return
	}

	// Resolve specific variant types for the specific-variant check.
	var specificTypes []*Type
	for _, name := range cfg.specificTypes {
		if t := lookupTypeInNamedModule(ctx, cfg.moduleName, name); t != nil {
			specificTypes = append(specificTypes, t)
		}
	}

	for _, ref := range objRefs {
		resolved := ctx.lookupObjectInModuleScope(ref.mod, ref.obj.Name)
		if resolved == nil || !resolved.IsColumn() {
			continue
		}
		t := resolved.Type()
		if t == nil {
			continue
		}

		// Check 1: address column without address-type sibling.
		if isDerivedFromType(t, addrType) {
			checkAddressPairingSibling(ctx, ref, resolved, addrTypeType, cfg.diagPairing, cfg.addressType, cfg.addressTypeType)
			continue
		}

		// Check 2: address-type column with enum refinement but address sibling lacks SIZE.
		if isDerivedFromType(t, addrTypeType) {
			checkAddressTypeSubtyped(ctx, ref, resolved, addrType, objRefs, cfg.diagSubtyped, cfg.addressType, cfg.addressTypeType)
			continue
		}

		// Check 3: specific variant used instead of generic address type.
		for _, specific := range specificTypes {
			if isDerivedFromType(t, specific) {
				ctx.EmitDiagnostic(cfg.diagSpecific,
					ref.mod, ref.obj.Span,
					fmt.Sprintf("%q: %s is a specific variant, use %s with %s",
						ref.obj.Name, specific.Name(), cfg.addressType, cfg.addressTypeType))
				break
			}
		}
	}
}

// checkAddressPairingSibling checks that an address column has a sibling
// address-type column in the same row.
func checkAddressPairingSibling(ctx *resolverContext, ref objectTypeRef, resolved *Object, addrTypeType *Type, diagCode, addrName, addrTypeName string) {
	row := resolved.Row()
	if row == nil {
		return
	}
	for _, col := range row.Columns() {
		ct := col.Type()
		if ct != nil && isDerivedFromType(ct, addrTypeType) {
			return
		}
	}
	ctx.EmitDiagnostic(diagCode,
		ref.mod, ref.obj.Span,
		fmt.Sprintf("%q: %s column has no sibling with %s type", ref.obj.Name, addrName, addrTypeName))
}

// checkAddressTypeSubtyped checks that if an address-type column has enum
// refinements, the paired address column also has explicit SIZE constraints
// in its OBJECT-TYPE SYNTAX (not just inherited from the TC).
func checkAddressTypeSubtyped(ctx *resolverContext, ref objectTypeRef, resolved *Object, addrType *Type, objRefs []objectTypeRef, diagCode, addrName, addrTypeName string) {
	// Only check if the OBJECT-TYPE SYNTAX has explicit enum refinement.
	if _, isEnum := ref.obj.Syntax.(*module.TypeSyntaxIntegerEnum); !isEnum {
		return
	}
	row := resolved.Row()
	if row == nil {
		return
	}
	// Find sibling address column.
	for _, col := range row.Columns() {
		ct := col.Type()
		if ct == nil || !isDerivedFromType(ct, addrType) {
			continue
		}
		// Check whether the sibling's OBJECT-TYPE SYNTAX has an explicit
		// SIZE constraint (not just the TC-inherited one). Look up the
		// module IR entry for the sibling column.
		if hasExplicitSizeConstraint(col.Name(), ref.mod, objRefs) {
			return
		}
		ctx.EmitDiagnostic(diagCode,
			ref.mod, ref.obj.Span,
			fmt.Sprintf("%q: %s is subtyped but sibling %q (%s) has no SIZE constraint",
				ref.obj.Name, addrTypeName, col.Name(), addrName))
		return
	}
}

// hasExplicitSizeConstraint checks whether the OBJECT-TYPE definition for the
// named object has an explicit SIZE constraint in its SYNTAX clause.
func hasExplicitSizeConstraint(name string, mod *module.Module, objRefs []objectTypeRef) bool {
	for _, ref := range objRefs {
		if ref.mod != mod || ref.obj.Name != name {
			continue
		}
		c, ok := ref.obj.Syntax.(*module.TypeSyntaxConstrained)
		if !ok {
			return false
		}
		_, isSize := c.Constraint.(*module.ConstraintSize)
		return isSize
	}
	return false
}
