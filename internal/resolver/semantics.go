package resolver

import (
	"encoding/hex"
	"fmt"
	"log/slog"
	"slices"
	"strconv"
	"strings"

	"github.com/golangsnmp/gomib/internal/model"
	"github.com/golangsnmp/gomib/internal/module"
	"github.com/golangsnmp/gomib/internal/types"
)

// resolveSemantics is the semantic analysis phase entry point.
func resolveSemantics(ctx *resolverContext) {
	objRefs := collectObjectTypeRefs(ctx)
	inferNodeKinds(ctx, objRefs)
	resolveTableSemantics(ctx, objRefs)
	createResolvedObjects(ctx, objRefs)
	createResolvedNotifications(ctx)
	createResolvedGroups(ctx)
	createResolvedCompliances(ctx)
	createResolvedCapabilities(ctx)
	checkAugmentsNesting(ctx, objRefs)
	checkNodeParentKinds(ctx, objRefs)
	checkIntegerMisuse(ctx)
	checkEnumSubtyping(ctx)
	checkRangeConstraints(ctx)
	checkIndexConstraints(ctx, objRefs)
	checkDefvalConstraints(ctx, objRefs)
	checkSequenceFields(ctx, objRefs)
	checkAccessAndStatus(ctx, objRefs)
	checkGroupMembership(ctx, objRefs)
	checkNotificationReversibility(ctx)
	checkComplianceStatus(ctx)
	checkComplianceStructure(ctx)
	checkGroupMemberLocality(ctx)
	checkDescriptionMissing(ctx, objRefs)
	checkTextualConventionNested(ctx)
	checkTypeAssignmentSMIv2(ctx)
	checkTableRowNaming(ctx, objRefs)
	checkNamedNumberOrdering(ctx)
	checkHyphenInLabel(ctx)
	checkOpaqueSMIv2(ctx, objRefs)
	checkFormatHints(ctx)
	checkTypeUnreferenced(ctx)
	checkGroupUnreferenced(ctx)
	checkIdentifierCaseMatch(ctx)
	checkTrapInSMIv2(ctx)
	checkNodeImplicit(ctx)
	checkModuleIdentityRegistration(ctx)
	checkRowStatusDefaults(ctx, objRefs)
	checkStorageTypeDefaults(ctx, objRefs)
	checkTAddressTDomain(ctx, objRefs)
	checkIncludesGroups(ctx)
	checkCreationRequires(ctx)
	checkIpAddressDeprecation(ctx, objRefs)
	checkInetAddressPairing(ctx, objRefs)
	checkTransportAddressPairing(ctx, objRefs)
}

func inferNodeKinds(ctx *resolverContext, objRefs []objectTypeRef) {
	tables, rows, scalars := 0, 0, 0
	var rowNodes []*model.Node

	// Track which module's OBJECT-TYPE determined each node's kind so
	// that only the preferred module's structural classification wins
	// when multiple modules define the same model.OID.
	nodeKindModule := make(map[*model.Node]*model.Module)

	for _, ref := range objRefs {
		obj := ref.obj
		node, ok := ctx.lookupNode(ref.mod, obj.Name)
		if !ok {
			continue
		}

		if !shouldPreferModule(ctx, nodeKindModule[node], ref.mod) {
			continue
		}

		prevKind := node.Kind()
		if _, isSequenceOf := obj.Syntax.(*module.TypeSyntaxSequenceOf); isSequenceOf {
			model.SetNodeKind(node, model.KindTable)
			if prevKind != model.KindTable {
				tables++
			}
		} else if len(obj.Index) > 0 || obj.Augments != "" {
			model.SetNodeKind(node, model.KindRow)
			if prevKind != model.KindRow {
				rows++
			}
			rowNodes = append(rowNodes, node)
		} else {
			model.SetNodeKind(node, model.KindScalar)
			if prevKind != model.KindScalar {
				scalars++
			}
		}
		nodeKindModule[node] = ctx.moduleToResolved[ref.mod]
	}

	// Reclassify scalar children of row nodes as columns
	columns := 0
	seen := make(map[*model.Node]struct{})
	for _, row := range rowNodes {
		if _, dup := seen[row]; dup {
			continue
		}
		seen[row] = struct{}{}
		if row.Kind() != model.KindRow {
			continue
		}
		for _, child := range row.Children() {
			if child.Kind() == model.KindScalar {
				model.SetNodeKind(child, model.KindColumn)
				columns++
			}
		}
	}

	if ctx.TraceEnabled() {
		ctx.Trace("inferred node kinds",
			slog.Int("tables", tables),
			slog.Int("rows", rows),
			slog.Int("columns", columns),
			slog.Int("scalars", scalars))
	}
}

type objectTypeRef struct {
	mod *module.Module
	obj *module.ObjectType
}

type notificationRef struct {
	mod   *module.Module
	notif *module.Notification
}

func collectObjectTypeRefs(ctx *resolverContext) []objectTypeRef {
	var refs []objectTypeRef
	for _, mod := range ctx.modules {
		for _, obj := range mod.ObjectTypes {
			refs = append(refs, objectTypeRef{mod: mod, obj: obj})
		}
	}
	return refs
}

func collectNotificationRefs(ctx *resolverContext) []notificationRef {
	var refs []notificationRef
	for _, mod := range ctx.modules {
		for _, notif := range mod.Notifications {
			refs = append(refs, notificationRef{mod: mod, notif: notif})
		}
	}
	return refs
}

func collectComplianceRefs(ctx *resolverContext) []complianceRef {
	var refs []complianceRef
	for _, mod := range ctx.modules {
		for _, comp := range mod.Compliances {
			refs = append(refs, complianceRef{mod: mod, comp: comp})
		}
	}
	return refs
}

func resolveTableSemantics(ctx *resolverContext, objRefs []objectTypeRef) {
	for _, ref := range objRefs {
		obj := ref.obj
		if len(obj.Index) == 0 && obj.Augments == "" {
			continue
		}

		if len(obj.Index) > 0 {
			for _, item := range obj.Index {
				if _, ok := ctx.lookupNode(ref.mod, item.Object); !ok {
					if isBareTypeIndex(item.Object) {
						continue
					}
					ctx.recordUnresolved(types.DiagIndexUnresolved, ref.mod, item.Span,
						fmt.Sprintf("unresolved INDEX: %q references unknown object %q", obj.Name, item.Object),
						model.UnresolvedRef{Kind: model.UnresolvedIndex, Symbol: item.Object, Module: modName(ref.mod), Reason: reasonUnknownIndex})
				}
			}
		}

		if obj.Augments != "" {
			if _, ok := ctx.lookupNode(ref.mod, obj.Augments); !ok {
				ctx.recordUnresolved(types.DiagOidOrphan, ref.mod, obj.Spans.Augments,
					fmt.Sprintf("unresolved model.OID: %q references unknown parent %q", obj.Name, obj.Augments),
					model.UnresolvedRef{Kind: model.UnresolvedOID, Symbol: obj.Augments, Module: modName(ref.mod), Reason: reasonUnknownParent})
			}
		}
	}
}

func createResolvedObjects(ctx *resolverContext, objRefs []objectTypeRef) {
	created := 0
	for _, ref := range objRefs {
		obj := ref.obj

		node, ok := ctx.lookupNode(ref.mod, obj.Name)
		if !ok {
			continue
		}

		resolved := model.NewObject(obj.Name)
		model.SetObjectSpan(resolved, obj.Span)
		model.SetObjectSyntaxSpan(resolved, obj.Spans.Syntax)
		model.SetObjectAccessSpan(resolved, obj.Spans.Access)
		model.SetObjectStatusSpan(resolved, obj.Spans.Status)
		model.SetObjectDescriptionSpan(resolved, obj.Spans.Description)
		model.SetObjectReferenceSpan(resolved, obj.Spans.Reference)
		model.SetObjectUnitsSpan(resolved, obj.Spans.Units)
		model.SetObjectAugmentsSpan(resolved, obj.Spans.Augments)
		model.SetObjectDefaultValueSpan(resolved, obj.Spans.DefVal)
		model.SetObjectOidRefs(resolved, extractOidRefs(obj.DefinitionOid()))
		model.SetObjectNode(resolved, node)
		if resolvedMod := ctx.moduleToResolved[ref.mod]; resolvedMod != nil {
			model.SetObjectModule(resolved, resolvedMod)
		}
		model.SetObjectAccess(resolved, obj.Access)
		model.SetObjectStatus(resolved, obj.Status)
		model.SetObjectDescription(resolved, obj.Description)
		model.SetObjectUnits(resolved, obj.Units)
		model.SetObjectReference(resolved, obj.Reference)

		if t, ok := resolveTypeSyntax(ctx, obj.Syntax, ref.mod, obj.Name, obj.Spans.Syntax); ok {
			model.SetObjectType(resolved, t)
		}

		sizes, ranges := extractConstraints(obj.Syntax)
		if len(sizes) > 0 {
			model.SetObjectDeclaredSizes(resolved, sizes)
			model.SetObjectEffectiveSizes(resolved, sizes)
		}
		if len(ranges) > 0 {
			model.SetObjectDeclaredRanges(resolved, ranges)
			model.SetObjectEffectiveRanges(resolved, ranges)
		}
		if _, isBits := obj.Syntax.(*module.TypeSyntaxBits); isBits {
			model.SetObjectEffectiveBits(resolved, extractNamedValues(obj.Syntax))
		} else {
			model.SetObjectEffectiveEnums(resolved, extractNamedValues(obj.Syntax))
		}

		if obj.DefVal != nil {
			model.SetObjectDefaultValue(resolved, convertDefVal(ctx, obj.DefVal, ref.mod, obj.Syntax, obj.Span))
		}

		computeEffectiveValues(resolved)

		// Preserve the SEQUENCE type name from the row's SYNTAX clause.
		if node.Kind() == model.KindRow {
			if ref, ok := obj.Syntax.(*module.TypeSyntaxTypeRef); ok {
				model.SetObjectSequenceTypeName(resolved, ref.Name)
			}
		}

		// Prefer SMIv2 modules when multiple modules define the same model.OID
		// (e.g., IF-MIB and RFC1213-MIB both define ifEntry). Select the node's
		// preferred object before updating the global name index so same-name
		// collisions use the same deterministic preference.
		currentObj := node.Object()
		var currentMod *model.Module
		if currentObj != nil {
			currentMod = currentObj.Module()
		}
		if shouldPreferModule(ctx, currentMod, ref.mod) {
			model.SetNodeObject(node, resolved)
		}
		model.AddMibObject(ctx.mib, resolved)
		created++

		if resolvedMod := ctx.moduleToResolved[ref.mod]; resolvedMod != nil {
			model.AddModuleObject(resolvedMod, resolved)
			model.AddModuleNodeNamed(resolvedMod, obj.Name, node)
		}
	}

	linkObjectIndexes(ctx, objRefs)

	if ctx.TraceEnabled() {
		ctx.Trace("created resolved objects", slog.Int("count", created))
	}
}

// extractOidRefs extracts named symbolic references from an OID assignment.
func extractOidRefs(oid *module.OidAssignment) []model.OidRef {
	if oid == nil {
		return nil
	}
	var refs []model.OidRef
	for _, comp := range oid.Components {
		if comp.Name != "" {
			refs = append(refs, model.OidRef{Name: comp.Name, Span: comp.Span})
		}
	}
	return refs
}

// linkObjectIndexes resolves INDEX and AUGMENTS references now that all
// objects exist. Uses the module's own Object instance, not the shared
// node's, because multiple modules can define objects at the same model.OID.
func linkObjectIndexes(ctx *resolverContext, objRefs []objectTypeRef) {
	for _, ref := range objRefs {
		obj := ref.obj

		resolvedMod := ctx.moduleToResolved[ref.mod]
		if resolvedMod == nil {
			continue
		}
		resolvedObj := resolvedMod.Object(obj.Name)
		if resolvedObj == nil {
			continue
		}

		if len(obj.Index) > 0 {
			var indexEntries []model.IndexEntry
			for _, item := range obj.Index {
				if idxObj, nodeFound := ctx.resolveObject(ref.mod, item.Object); idxObj != nil {
					indexEntries = append(indexEntries, model.IndexEntry{
						Object:   idxObj,
						Implied:  item.Implied,
						Encoding: model.ClassifyIndexEncoding(idxObj, item.Implied),
						Span:     item.Span,
					})
				} else if isBareTypeIndex(item.Object) {
					indexEntries = append(indexEntries, model.IndexEntry{
						TypeName: item.Object,
						Implied:  item.Implied,
						Span:     item.Span,
					})
				} else if nodeFound {
					ctx.EmitDiagnostic(types.DiagIndexNotObject,
						ref.mod, obj.Span,
						fmt.Sprintf("INDEX %q of %q resolves to a node without an object definition", item.Object, obj.Name))
				}
			}
			model.SetObjectIndex(resolvedObj, indexEntries)
		}

		if obj.Augments != "" {
			if target, nodeFound := ctx.resolveObject(ref.mod, obj.Augments); target != nil {
				model.SetObjectAugments(resolvedObj, target)
				model.AddObjectAugmentedBy(target, resolvedObj)
			} else if nodeFound {
				ctx.EmitDiagnostic(types.DiagAugmentsNotObject,
					ref.mod, obj.Span,
					fmt.Sprintf("AUGMENTS target %q of %q resolves to a node without an object definition", obj.Augments, obj.Name))
			}
		}
	}
}

// checkAugmentsNesting verifies that AUGMENTS targets are base table rows
// (rows with their own INDEX clause), not rows that themselves use AUGMENTS.
// This must run after linkObjectIndexes so all augments links are established.
func checkAugmentsNesting(ctx *resolverContext, objRefs []objectTypeRef) {
	for _, ref := range objRefs {
		obj := ref.obj
		if obj.Augments == "" {
			continue
		}
		resolvedObj := ctx.lookupObject(ref.mod, obj.Name)
		if resolvedObj == nil || resolvedObj.Augments() == nil {
			continue
		}
		target := resolvedObj.Augments()
		if target.Augments() != nil {
			ctx.EmitDiagnostic(types.DiagAugmentNested,
				ref.mod, obj.Span,
				fmt.Sprintf("%q augments %q which is not a base table row", obj.Name, obj.Augments))
		}
	}
}

// computeEffectiveValues fills in display hints and effective constraints.
// Inline object constraints narrow the type's effective constraints rather
// than replacing them.
func computeEffectiveValues(obj *model.Object) {
	t := obj.Type()
	if t == nil {
		return
	}

	if obj.EffectiveDisplayHint() == "" {
		model.SetObjectEffectiveHint(obj, t.EffectiveDisplayHint())
	}
	if obj.EffectiveSizesConstrained() {
		parent := t.EffectiveSizes()
		parentConstrained := t.EffectiveSizesConstrained()
		if !parentConstrained {
			if base, ok := model.BaseConstraint(t.EffectiveBase(), true, obj.SyntaxSpan()); ok {
				parent = []model.Range{base}
				parentConstrained = true
			}
		}
		if parentConstrained && len(parent) == 0 {
			model.SetObjectEffectiveSizes(obj, nil)
		} else {
			model.SetObjectEffectiveSizes(obj, model.IntersectRanges(obj.EffectiveSizes(), parent))
		}
	} else {
		model.SetObjectEffectiveSizes(obj, t.EffectiveSizes())
		model.SetObjectEffectiveSizesConstrained(obj, t.EffectiveSizesConstrained())
	}
	if obj.EffectiveRangesConstrained() {
		parent := t.EffectiveRanges()
		parentConstrained := t.EffectiveRangesConstrained()
		if !parentConstrained {
			if base, ok := model.BaseConstraint(t.EffectiveBase(), false, obj.SyntaxSpan()); ok {
				parent = []model.Range{base}
				parentConstrained = true
			}
		}
		if parentConstrained && len(parent) == 0 {
			model.SetObjectEffectiveRanges(obj, nil)
		} else {
			model.SetObjectEffectiveRanges(obj, model.IntersectRanges(obj.EffectiveRanges(), parent))
		}
	} else {
		model.SetObjectEffectiveRanges(obj, t.EffectiveRanges())
		model.SetObjectEffectiveRangesConstrained(obj, t.EffectiveRangesConstrained())
	}
	if len(obj.EffectiveEnums()) == 0 {
		model.SetObjectEffectiveEnums(obj, t.EffectiveEnums())
	}
	if len(obj.EffectiveBits()) == 0 {
		model.SetObjectEffectiveBits(obj, t.EffectiveBits())
	}
}

func createResolvedNotifications(ctx *resolverContext) {
	created := 0
	for _, ref := range collectNotificationRefs(ctx) {
		notif := ref.notif

		node, ok := ctx.lookupNode(ref.mod, notif.Name)
		if !ok {
			continue
		}

		resolved := model.NewNotification(notif.Name)
		model.SetNotificationSpan(resolved, notif.Span)
		model.SetNotificationOidRefs(resolved, extractOidRefs(notif.DefinitionOid()))
		model.SetNotificationNode(resolved, node)
		if resolvedMod := ctx.moduleToResolved[ref.mod]; resolvedMod != nil {
			model.SetNotificationModule(resolved, resolvedMod)
		}
		model.SetNotificationStatus(resolved, notif.Status)
		model.SetNotificationDescription(resolved, notif.Description)
		model.SetNotificationReference(resolved, notif.Reference)
		if notif.TrapInfo != nil {
			model.SetNotificationTrapInfo(resolved, &model.TrapInfo{
				Enterprise: notif.TrapInfo.Enterprise,
				TrapNumber: notif.TrapInfo.TrapNumber,
			})
		}

		for _, objRef := range notif.Objects {
			objName := objRef.Name

			// addNotifObject adds an object to the notification, checking
			// not-accessible and emitting a diagnostic if needed.
			addNotifObject := func(obj *model.Object) {
				model.AddNotificationObject(resolved, obj)
				if obj.Access() == model.AccessNotAccessible {
					ctx.EmitDiagnostic(types.DiagNotifObjectAccess,
						ref.mod, objRef.Span,
						fmt.Sprintf("notification %q references %q which is not-accessible", notif.Name, objName))
				}
			}

			// Use module-scoped object lookup to get the object from the
			// correct module, not whichever module won node preference.
			if obj := ctx.lookupObject(ref.mod, objName); obj != nil {
				addNotifObject(obj)
				continue
			}

			// Permissive only: global lookup for objects not explicitly imported.
			// No module scope available, so use the node-attached object.
			if ctx.ResolverStrictness().AllowGlobalFallbacks() {
				if objNode, ok := ctx.lookupNodeGlobal(objName); ok {
					if ctx.TraceEnabled() {
						ctx.Trace("permissive: resolved notification object via global lookup",
							slog.String("object", objName),
							slog.String("notification", notif.Name))
					}
					if objNode.Object() != nil {
						addNotifObject(objNode.Object())
						continue
					}
					// Node exists but has no object definition.
					ctx.EmitDiagnostic(types.DiagNotifObjectNotObject, ref.mod, objRef.Span,
						fmt.Sprintf("notification %q references %q which is not an object definition", notif.Name, objName))
					continue
				}
			}

			// Try node lookup to distinguish "not found" from "not an object".
			objNode, ok := ctx.lookupNode(ref.mod, objName)
			switch {
			case !ok:
				ctx.recordUnresolved(types.DiagObjectsUnresolved, ref.mod, objRef.Span,
					fmt.Sprintf("unresolved OBJECTS: %q references unknown object %q", notif.Name, objName),
					model.UnresolvedRef{Kind: model.UnresolvedNotificationObject, Symbol: objName, Module: modName(ref.mod), Reason: reasonUnknownObject})
			case objNode.Object() == nil:
				ctx.EmitDiagnostic(types.DiagNotifObjectNotObject, ref.mod, objRef.Span,
					fmt.Sprintf("notification %q references %q which is not an object definition", notif.Name, objName))
			default:
				// Node has an object but it belongs to a different module
				// (not in our import scope). Still add it since we found it.
				addNotifObject(objNode.Object())
			}
		}

		registerNotification(ctx, ref.mod, node, resolved)
		created++
	}

	if ctx.TraceEnabled() {
		ctx.Trace("created resolved notifications", slog.Int("count", created))
	}
}

// groupInfo holds the common fields extracted from ObjectGroup or NotificationGroup
// for the shared group resolution logic.
type groupInfo struct {
	mod         *module.Module
	name        string
	span        types.Span
	status      model.Status
	description string
	reference   string
	members     []module.NameRef
	oidRefs     []model.OidRef
	isNotifGrp  bool
}

func createResolvedGroups(ctx *resolverContext) {
	var groups []groupInfo
	for _, mod := range ctx.modules {
		for _, d := range mod.ObjectGroups {
			groups = append(groups, groupInfo{
				mod: mod, name: d.Name, span: d.DefinitionSpan(),
				status: d.Status, description: d.Description,
				reference: d.Reference, members: d.Objects,
				oidRefs: extractOidRefs(d.DefinitionOid()),
			})
		}
		for _, d := range mod.NotificationGroups {
			groups = append(groups, groupInfo{
				mod: mod, name: d.Name, span: d.DefinitionSpan(),
				status: d.Status, description: d.Description,
				reference: d.Reference, members: d.Notifications,
				oidRefs:    extractOidRefs(d.DefinitionOid()),
				isNotifGrp: true,
			})
		}
	}

	created := 0
	for i := range groups {
		gi := &groups[i]
		node, ok := ctx.lookupNode(gi.mod, gi.name)
		if !ok {
			continue
		}

		resolved := model.NewGroup(gi.name)
		model.SetGroupSpan(resolved, gi.span)
		model.SetGroupOidRefs(resolved, gi.oidRefs)
		model.SetGroupNode(resolved, node)
		if resolvedMod := ctx.moduleToResolved[gi.mod]; resolvedMod != nil {
			model.SetGroupModule(resolved, resolvedMod)
		}
		model.SetGroupStatus(resolved, gi.status)
		model.SetGroupDescription(resolved, gi.description)
		model.SetGroupReference(resolved, gi.reference)
		model.SetGroupIsNotificationGroup(resolved, gi.isNotifGrp)

		var hasObjects, hasNotifications bool
		for _, memberRef := range gi.members {
			memberName := memberRef.Name
			if memberNode, ok := lookupMemberNode(ctx, gi.mod, memberName); ok {
				model.AddGroupMember(resolved, memberNode)
				kind := memberNode.Kind()
				if gi.isNotifGrp {
					if kind.IsObjectType() {
						hasObjects = true
						ctx.EmitDiagnostic(types.DiagGroupNotificationsObject,
							gi.mod, memberRef.Span,
							fmt.Sprintf("notification group %q includes object %q", gi.name, memberName))
					} else if kind == types.KindNotification {
						hasNotifications = true
					}
				} else {
					if kind == types.KindNotification {
						hasNotifications = true
						ctx.EmitDiagnostic(types.DiagGroupObjectsNotification,
							gi.mod, memberRef.Span,
							fmt.Sprintf("object group %q includes notification %q", gi.name, memberName))
					} else if kind.IsObjectType() {
						hasObjects = true
					}
					if obj := memberNode.Object(); obj != nil && obj.Access() == model.AccessNotAccessible {
						ctx.EmitDiagnostic(types.DiagGroupNotAccessible,
							gi.mod, memberRef.Span,
							fmt.Sprintf("object %q of group %q must not be not-accessible", memberName, gi.name))
					}
				}
				checkGroupMemberStatus(ctx, gi.mod, memberRef.Span, gi.status, gi.name, memberNode, memberName)
			} else {
				ctx.EmitDiagnostic(types.DiagGroupMemberUnresolved,
					gi.mod, memberRef.Span,
					fmt.Sprintf("group %q references unresolved member %q", gi.name, memberName))
			}
		}
		emitMixedGroupDiagnostic(ctx, gi.mod, gi.span, gi.name, hasObjects, hasNotifications)

		registerGroup(ctx, gi.mod, node, resolved)
		created++
	}

	if ctx.TraceEnabled() {
		ctx.Trace("created resolved groups", slog.Int("count", created))
	}
}

// registerResolvedEntity handles the common pattern for registering a resolved entity:
// conditionally set it on the node (preferring newer modules), add it to the global
// Mib, then add it to the per-module collection. Node selection precedes global
// indexing so same-name collisions follow the node's deterministic preference.
func registerResolvedEntity[T any](ctx *resolverContext, mod *module.Module, currentMod *model.Module, resolved T, addToMib, setOnNode func(T), addToModule func(*model.Module, T)) {
	if shouldPreferModule(ctx, currentMod, mod) {
		setOnNode(resolved)
	}
	addToMib(resolved)
	if resolvedMod := ctx.moduleToResolved[mod]; resolvedMod != nil {
		addToModule(resolvedMod, resolved)
	}
}

func registerNotification(ctx *resolverContext, mod *module.Module, node *model.Node, resolved *model.Notification) {
	var currentMod *model.Module
	if n := node.Notification(); n != nil {
		currentMod = n.Module()
	}
	registerResolvedEntity(ctx, mod, currentMod, resolved,
		func(n *model.Notification) { model.AddMibNotification(ctx.mib, n) },
		func(n *model.Notification) { model.SetNodeNotification(node, n) },
		func(m *model.Module, n *model.Notification) { model.AddModuleNotification(m, n) })
	if resolvedMod := ctx.moduleToResolved[mod]; resolvedMod != nil {
		model.AddModuleNodeNamed(resolvedMod, resolved.Name(), node)
	}
}

func registerGroup(ctx *resolverContext, mod *module.Module, node *model.Node, resolved *model.Group) {
	var currentMod *model.Module
	if g := node.Group(); g != nil {
		currentMod = g.Module()
	}
	registerResolvedEntity(ctx, mod, currentMod, resolved,
		func(g *model.Group) { model.AddMibGroup(ctx.mib, g) },
		func(g *model.Group) { model.SetNodeGroup(node, g) },
		func(m *model.Module, g *model.Group) { model.AddModuleGroup(m, g) })
	if resolvedMod := ctx.moduleToResolved[mod]; resolvedMod != nil {
		model.AddModuleNodeNamed(resolvedMod, resolved.Name(), node)
	}
}

func registerCompliance(ctx *resolverContext, mod *module.Module, node *model.Node, resolved *model.Compliance) {
	var currentMod *model.Module
	if c := node.Compliance(); c != nil {
		currentMod = c.Module()
	}
	registerResolvedEntity(ctx, mod, currentMod, resolved,
		func(c *model.Compliance) { model.AddMibCompliance(ctx.mib, c) },
		func(c *model.Compliance) { model.SetNodeCompliance(node, c) },
		func(m *model.Module, c *model.Compliance) { model.AddModuleCompliance(m, c) })
	if resolvedMod := ctx.moduleToResolved[mod]; resolvedMod != nil {
		model.AddModuleNodeNamed(resolvedMod, resolved.Name(), node)
	}
}

func registerCapability(ctx *resolverContext, mod *module.Module, node *model.Node, resolved *model.Capability) {
	var currentMod *model.Module
	if c := node.Capability(); c != nil {
		currentMod = c.Module()
	}
	registerResolvedEntity(ctx, mod, currentMod, resolved,
		func(c *model.Capability) { model.AddMibCapability(ctx.mib, c) },
		func(c *model.Capability) { model.SetNodeCapability(node, c) },
		func(m *model.Module, c *model.Capability) { model.AddModuleCapability(m, c) })
	if resolvedMod := ctx.moduleToResolved[mod]; resolvedMod != nil {
		model.AddModuleNodeNamed(resolvedMod, resolved.Name(), node)
	}
}

type complianceRef struct {
	mod  *module.Module
	comp *module.ModuleCompliance
}

func createResolvedCompliances(ctx *resolverContext) {
	created := 0
	for _, ref := range collectComplianceRefs(ctx) {
		comp := ref.comp
		node, ok := ctx.lookupNode(ref.mod, comp.Name)
		if !ok {
			continue
		}

		resolved := model.NewCompliance(comp.Name)
		model.SetComplianceSpan(resolved, comp.Span)
		model.SetComplianceOidRefs(resolved, extractOidRefs(comp.DefinitionOid()))
		model.SetComplianceNode(resolved, node)
		if resolvedMod := ctx.moduleToResolved[ref.mod]; resolvedMod != nil {
			model.SetComplianceModule(resolved, resolvedMod)
		}
		model.SetComplianceStatus(resolved, comp.Status)
		model.SetComplianceDescription(resolved, comp.Description)
		model.SetComplianceReference(resolved, comp.Reference)
		model.SetComplianceModules(resolved, convertComplianceModules(ctx, ref.mod, comp.Modules))

		registerCompliance(ctx, ref.mod, node, resolved)
		created++
	}

	if ctx.TraceEnabled() {
		ctx.Trace("created resolved compliances", slog.Int("count", created))
	}
}

func convertComplianceModules(ctx *resolverContext, mod *module.Module, modules []module.ComplianceModule) []model.ComplianceModule {
	result := make([]model.ComplianceModule, len(modules))
	for i, m := range modules {
		result[i] = model.ComplianceModule{
			ModuleName:      m.ModuleName,
			MandatoryGroups: convertNameRefs(m.MandatoryGroups),
			Span:            m.Span,
		}
		if len(m.Groups) > 0 {
			groups := make([]model.ComplianceGroup, len(m.Groups))
			for j, g := range m.Groups {
				groups[j] = model.ComplianceGroup{
					Group:       g.Group,
					Description: g.Description,
					Span:        g.Span,
				}
			}
			result[i].Groups = groups
		}
		if len(m.Objects) > 0 {
			objects := make([]model.ComplianceObject, len(m.Objects))
			for j, o := range m.Objects {
				objects[j] = model.ComplianceObject{
					Object:      o.Object,
					Description: o.Description,
					Span:        o.Span,
				}
				objects[j].Syntax = resolveSyntaxConstraints(ctx, o.Syntax, mod, o.Object, o.Span)
				objects[j].WriteSyntax = resolveSyntaxConstraints(ctx, o.WriteSyntax, mod, o.Object, o.Span)
				if o.MinAccess != nil {
					objects[j].MinAccess = o.MinAccess
				}
			}
			result[i].Objects = objects
		}
	}
	return result
}

func createResolvedCapabilities(ctx *resolverContext) {
	created := 0
	for _, mod := range ctx.modules {
		for _, ac := range mod.Capabilities {
			node, ok := ctx.lookupNode(mod, ac.Name)
			if !ok {
				continue
			}

			resolved := model.NewCapability(ac.Name)
			model.SetCapabilitySpan(resolved, ac.Span)
			model.SetCapabilityOidRefs(resolved, extractOidRefs(ac.DefinitionOid()))
			model.SetCapabilityNode(resolved, node)
			if resolvedMod := ctx.moduleToResolved[mod]; resolvedMod != nil {
				model.SetCapabilityModule(resolved, resolvedMod)
			}
			model.SetCapabilityStatus(resolved, ac.Status)
			model.SetCapabilityDescription(resolved, ac.Description)
			model.SetCapabilityReference(resolved, ac.Reference)
			model.SetCapabilityProductRelease(resolved, ac.ProductRelease)
			model.SetCapabilitySupports(resolved, convertSupportsModules(ctx, mod, ac.Supports))

			registerCapability(ctx, mod, node, resolved)
			created++
		}
	}

	if ctx.TraceEnabled() {
		ctx.Trace("created resolved capabilities", slog.Int("count", created))
	}
}

func convertSupportsModules(ctx *resolverContext, mod *module.Module, modules []module.SupportsModule) []model.CapabilitiesModule {
	result := make([]model.CapabilitiesModule, len(modules))
	for i, m := range modules {
		result[i] = model.CapabilitiesModule{
			ModuleName: m.ModuleName,
			Includes:   convertNameRefs(m.Includes),
			Span:       m.Span,
		}
		for _, v := range m.Variations {
			if isNotificationVariation(ctx, mod, m.ModuleName, &v) {
				nv := model.NotificationVariation{
					Notification: v.Name,
					Description:  v.Description,
					Span:         v.Span,
				}
				if v.Access != nil {
					nv.Access = v.Access
					if *v.Access != model.AccessNotImplemented {
						ctx.EmitDiagnostic(types.DiagVariationAccessNotifOnly,
							mod, v.Span,
							fmt.Sprintf("notification variation %q ACCESS should be not-implemented per RFC 2580", v.Name))
					}
				}
				result[i].NotificationVariations = append(result[i].NotificationVariations, nv)
			} else {
				ov := model.ObjectVariation{
					Object:      v.Name,
					Description: v.Description,
					Span:        v.Span,
				}
				ov.Syntax = resolveSyntaxConstraints(ctx, v.Syntax, mod, v.Name, v.Span)
				ov.WriteSyntax = resolveSyntaxConstraints(ctx, v.WriteSyntax, mod, v.Name, v.Span)
				if len(v.CreationRequires) > 0 {
					ov.CreationRequires = convertNameRefs(v.CreationRequires)
				}
				if v.Access != nil {
					ov.Access = v.Access
				}
				if v.DefVal != nil {
					if dv := convertDefVal(ctx, v.DefVal, mod, v.Syntax, v.Span); dv != nil {
						ov.DefVal = *dv
					}
				}
				result[i].ObjectVariations = append(result[i].ObjectVariations, ov)
			}
		}
	}
	return result
}

// isNotificationVariation determines whether a VARIATION clause references a
// NOTIFICATION-TYPE. It looks up the referenced name in the SUPPORTS module,
// the defining module's imports, and, in permissive mode only, via global
// search. If the name cannot be resolved, it falls back to a syntactic
// heuristic: variations with SYNTAX, WRITE-SYNTAX, CREATION-REQUIRES, or
// DEFVAL are object variations.
func isNotificationVariation(ctx *resolverContext, mod *module.Module, supportsModule string, v *module.Variation) bool {
	// Try the SUPPORTS module first (most correct per RFC 2580).
	if node, ok := ctx.lookupNodeByModuleName(supportsModule, v.Name); ok {
		return node.Kind() == model.KindNotification
	}
	// Try the defining module's import chain.
	if node, ok := ctx.lookupNode(mod, v.Name); ok {
		return node.Kind() == model.KindNotification
	}
	// Permissive only: try global lookup.
	if ctx.ResolverStrictness().AllowGlobalFallbacks() {
		if node, ok := ctx.lookupNodeGlobal(v.Name); ok {
			return node.Kind() == model.KindNotification
		}
	}
	// Cannot resolve; fall back to syntactic heuristic.
	return v.Syntax == nil && v.WriteSyntax == nil && len(v.CreationRequires) == 0 && v.DefVal == nil
}

// convertNameRefs converts module-IR NameRefs to model NameRefs, preserving
// per-item spans for diagnostic positioning by downstream consumers.
func convertNameRefs(refs []module.NameRef) []model.NameRef {
	result := make([]model.NameRef, len(refs))
	for i, r := range refs {
		result[i] = model.NameRef{Name: r.Name, Span: r.Span}
	}
	return result
}

func lookupMemberNode(ctx *resolverContext, mod *module.Module, name string) (*model.Node, bool) {
	node, ok := ctx.lookupNode(mod, name)
	if ok {
		return node, true
	}
	if ctx.ResolverStrictness().AllowGlobalFallbacks() {
		node, ok = ctx.lookupNodeGlobal(name)
		if ok && ctx.TraceEnabled() {
			ctx.Trace("permissive: resolved group member via global lookup",
				slog.String("member", name),
				slog.String("module", mod.Name))
		}
		return node, ok
	}
	return nil, false
}

// memberNodeStatus returns the status of a group member node from its
// attached Object or Notification entity.
func memberNodeStatus(n *model.Node) (model.Status, bool) {
	if obj := n.Object(); obj != nil {
		return obj.Status(), true
	}
	if notif := n.Notification(); notif != nil {
		return notif.Status(), true
	}
	return 0, false
}

// checkGroupMemberStatus emits DiagGroupObjectStatus when a member's
// SMIv2 status exceeds the group's status (e.g. obsolete member in a
// current group). SMIv1 statuses are skipped since they aren't
// comparable on the same scale.
func checkGroupMemberStatus(ctx *resolverContext, mod *module.Module, span types.Span, groupStatus model.Status, groupName string, memberNode *model.Node, memberName string) {
	ms, ok := memberNodeStatus(memberNode)
	if !ok || ms.IsSMIv1() || groupStatus.IsSMIv1() {
		return
	}
	if ms > groupStatus {
		ctx.EmitDiagnostic(types.DiagGroupObjectStatus,
			mod, span,
			fmt.Sprintf("%s group %q includes %s member %q", groupStatus, groupName, ms, memberName))
	}
}

// emitMixedGroupDiagnostic emits DiagGroupMemberMixed when a group
// contains both object-type and notification-type members.
func emitMixedGroupDiagnostic(ctx *resolverContext, mod *module.Module, span types.Span, groupName string, hasObjects, hasNotifications bool) {
	if hasObjects && hasNotifications {
		ctx.EmitDiagnostic(types.DiagGroupMemberMixed,
			mod, span,
			fmt.Sprintf("group %q contains scalars/columns and notifications", groupName))
	}
}

func resolveSyntaxConstraints(ctx *resolverContext, syntax module.TypeSyntax, mod *module.Module, ownerName string, syntaxSpan types.Span) *model.SyntaxConstraints {
	if syntax == nil {
		return nil
	}
	sc := &model.SyntaxConstraints{}
	if t, ok := resolveTypeSyntax(ctx, syntax, mod, ownerName, syntaxSpan); ok {
		sc.Type = t
	}
	sc.Sizes, sc.Ranges = extractConstraints(syntax)
	sc.DeclaredSizes = slices.Clone(sc.Sizes)
	sc.DeclaredRanges = slices.Clone(sc.Ranges)
	sc.SizesConstrained = len(sc.Sizes) > 0
	sc.RangesConstrained = len(sc.Ranges) > 0
	if sc.Type != nil {
		if sc.SizesConstrained {
			parent := sc.Type.EffectiveSizes()
			parentConstrained := sc.Type.EffectiveSizesConstrained()
			if !parentConstrained {
				if base, ok := model.BaseConstraint(sc.Type.EffectiveBase(), true, syntaxSpan); ok {
					parent = []model.Range{base}
					parentConstrained = true
				}
			}
			if parentConstrained && len(parent) == 0 {
				sc.Sizes = nil
			} else {
				sc.Sizes = model.IntersectRanges(sc.Sizes, parent)
			}
		}
		if sc.RangesConstrained {
			parent := sc.Type.EffectiveRanges()
			parentConstrained := sc.Type.EffectiveRangesConstrained()
			if !parentConstrained {
				if base, ok := model.BaseConstraint(sc.Type.EffectiveBase(), false, syntaxSpan); ok {
					parent = []model.Range{base}
					parentConstrained = true
				}
			}
			if parentConstrained && len(parent) == 0 {
				sc.Ranges = nil
			} else {
				sc.Ranges = model.IntersectRanges(sc.Ranges, parent)
			}
		}
	}
	if _, isBits := syntax.(*module.TypeSyntaxBits); isBits {
		sc.Bits = extractNamedValues(syntax)
	} else {
		sc.Enums = extractNamedValues(syntax)
	}
	return sc
}

func resolveTypeSyntax(ctx *resolverContext, syntax module.TypeSyntax, mod *module.Module, objectName string, span types.Span) (*model.Type, bool) {
	switch s := syntax.(type) {
	case *module.TypeSyntaxTypeRef:
		if t, ok := ctx.resolveTypeForModule(mod, s.Name); ok {
			return t, true
		}
		// Don't emit unresolved error for SEQUENCE type references (row SYNTAX).
		// SEQUENCE types are structural and intentionally not registered.
		if isSequenceTypeDef(ctx, mod, s.Name) {
			return nil, false
		}
		ctx.recordUnresolved(types.DiagTypeUnknown, mod, span,
			fmt.Sprintf("unresolved type: %q references unknown type %q", objectName, s.Name),
			model.UnresolvedRef{Kind: model.UnresolvedType, Symbol: s.Name, Module: modName(mod), Reason: reasonUnknownType})
		return nil, false
	case *module.TypeSyntaxConstrained:
		return resolveTypeSyntax(ctx, s.Base, mod, objectName, span)
	case *module.TypeSyntaxIntegerEnum:
		// Named base type (e.g., TPSPRateType { kbps(1) })
		if s.Base != "" {
			if t, ok := ctx.resolveTypeForModule(mod, s.Base); ok {
				return t, true
			}
			ctx.recordUnresolved(types.DiagTypeUnknown, mod, span,
				fmt.Sprintf("unresolved type: %q references unknown type %q", objectName, s.Base),
				model.UnresolvedRef{Kind: model.UnresolvedType, Symbol: s.Base, Module: modName(mod), Reason: reasonUnknownType})
			return nil, false
		}
		// Bare INTEGER { ... } enum with no named base
		if t, ok := ctx.resolveType("INTEGER"); ok {
			return t, true
		}
		ctx.EmitDiagnostic(types.DiagPrimitiveTypeMissing,
			mod, span, fmt.Sprintf("primitive type INTEGER not found for %q", objectName))
		return nil, false
	case *module.TypeSyntaxBits:
		return lookupPrimitiveType(ctx, mod, span, objectName, "BITS")
	case *module.TypeSyntaxOctetString:
		return lookupPrimitiveType(ctx, mod, span, objectName, "OCTET STRING")
	case *module.TypeSyntaxObjectIdentifier:
		return lookupPrimitiveType(ctx, mod, span, objectName, "OBJECT IDENTIFIER")
	case *module.TypeSyntaxSequenceOf, *module.TypeSyntaxSequence:
		return nil, false
	default:
		return nil, false
	}
}

func lookupPrimitiveType(ctx *resolverContext, mod *module.Module, span types.Span, objectName, typeName string) (*model.Type, bool) {
	if t, ok := ctx.resolveType(typeName); ok {
		return t, true
	}
	ctx.EmitDiagnostic(types.DiagPrimitiveTypeMissing,
		mod, span, fmt.Sprintf("primitive type %s not found for %q", typeName, objectName))
	return nil, false
}

// isSequenceTypeDef checks whether name refers to a SEQUENCE type definition
// in the module or its imports. Used to suppress spurious unresolved type
// diagnostics for row OBJECT-TYPE SYNTAX references (e.g., SYNTAX IfEntry).
func isSequenceTypeDef(ctx *resolverContext, mod *module.Module, name string) bool {
	return findSequenceTypeDef(ctx, mod, name) != nil
}

func emitDefvalUnresolved(ctx *resolverContext, mod *module.Module, span types.Span, message string) {
	ctx.EmitDiagnostic(types.DiagDefvalUnresolved, mod, span, message)
}

func convertDefVal(ctx *resolverContext, defval module.DefVal, mod *module.Module, syntax module.TypeSyntax, span types.Span) *model.DefVal {
	switch v := defval.(type) {
	case *module.DefValInteger:
		raw := strconv.FormatInt(v.Value, 10)
		dv := model.NewDefValInt(v.Value, raw)
		return &dv
	case *module.DefValUnsigned:
		raw := strconv.FormatUint(v.Value, 10)
		dv := model.NewDefValUint(v.Value, raw)
		return &dv
	case *module.DefValString:
		raw := `"` + v.Value + `"`
		dv := model.NewDefValString(v.Value, raw)
		return &dv
	case *module.DefValHexString:
		raw := "'" + v.Value + "'H"
		bytes, err := hexToBytes(v.Value)
		if err != nil {
			ctx.EmitDiagnostic(types.DiagMalformedHexDefval,
				mod, span, fmt.Sprintf("malformed hex DEFVAL %q: %s", raw, err.Error()))
			return nil
		}
		dv := model.NewDefValBytes(bytes, raw)
		return &dv
	case *module.DefValBinaryString:
		raw := "'" + v.Value + "'B"
		_, isBits := syntax.(*module.TypeSyntaxBits)
		bytes, valid := binaryToBytes(v.Value, isBits)
		if !valid {
			ctx.EmitDiagnostic(types.DiagMalformedBinDefval,
				mod, span, fmt.Sprintf("binary DEFVAL contains non-binary digits: %q", raw))
		}
		dv := model.NewDefValBytes(bytes, raw)
		return &dv
	case *module.DefValEnum:
		// Parser emits bare names as DefValEnum, but for model.OID-typed objects
		// the name is actually an OID reference.
		if isOIDType(ctx, mod, syntax) {
			if node, ok := ctx.lookupNode(mod, v.Name); ok {
				oid := node.OID()
				dv := model.NewDefValOID(oid, v.Name)
				return &dv
			}
			emitDefvalUnresolved(ctx, mod, span, fmt.Sprintf("DEFVAL OID reference %q could not be resolved", v.Name))
		}
		dv := model.NewDefValEnum(v.Name, v.Name)
		return &dv
	case *module.DefValBits:
		raw := "{ " + strings.Join(v.Labels, ", ") + " }"
		if len(v.Labels) == 0 {
			raw = "{ }"
		}
		dv := model.NewDefValBits(slices.Clone(v.Labels), raw)
		return &dv
	case *module.DefValOidRef:
		if node, ok := ctx.lookupNode(mod, v.Name); ok {
			oid := node.OID()
			dv := model.NewDefValOID(oid, v.Name)
			return &dv
		}
		emitDefvalUnresolved(ctx, mod, span, fmt.Sprintf("DEFVAL OID reference %q could not be resolved", v.Name))
		return nil
	case *module.DefValOidValue:
		if len(v.Components) == 0 {
			emitDefvalUnresolved(ctx, mod, span, "DEFVAL OID value has no components")
			return nil
		}
		first := v.Components[0]
		if first.Name == "" {
			emitDefvalUnresolved(ctx, mod, span, "DEFVAL OID value has no named root component")
			return nil
		}
		var node *model.Node
		var ok bool
		if first.Module != "" {
			node, ok = ctx.lookupNodeByModuleName(first.Module, first.Name)
		} else {
			node, ok = ctx.lookupNode(mod, first.Name)
		}
		if !ok {
			emitDefvalUnresolved(ctx, mod, span, fmt.Sprintf("DEFVAL OID root %q could not be resolved", first.Name))
			return nil
		}
		oid := node.OID()
		for _, comp := range v.Components[1:] {
			if !comp.HasNumber {
				label := comp.Name
				if comp.Module != "" {
					label = comp.Module + "." + comp.Name
				}
				emitDefvalUnresolved(ctx, mod, span, fmt.Sprintf("DEFVAL OID component %q has no numeric value", label))
				return nil
			}
			oid = append(oid, comp.Number)
		}
		dv := model.NewDefValOID(oid, first.Name)
		return &dv
	default:
		emitDefvalUnresolved(ctx, mod, span, "DEFVAL could not be parsed")
		return nil
	}
}

// hexToBytes converts a hex string (e.g., "00FF1A") to bytes.
func hexToBytes(s string) ([]byte, error) {
	if s == "" {
		return []byte{}, nil
	}
	// Handle odd-length hex strings by padding
	if len(s)%2 != 0 {
		s = "0" + s
	}
	return hex.DecodeString(s)
}

// binaryToBytes converts a binary string (e.g., "10101010") to bytes.
// For BITS-typed values (RFC 2578), bits are MSB-first so the string is
// right-padded to a byte boundary. For other types (OCTET STRING), the
// string is left-padded (treated as a numeric value).
// Returns the converted bytes and whether the input contained only valid
// binary digits (0 and 1). Invalid characters are treated as 0.
func binaryToBytes(s string, rightPad bool) ([]byte, bool) {
	if s == "" {
		return []byte{}, true
	}
	valid := true
	for i := range len(s) {
		if s[i] != '0' && s[i] != '1' {
			valid = false
			break
		}
	}
	// Pad to multiple of 8
	if padding := (8 - len(s)%8) % 8; padding > 0 {
		pad := strings.Repeat("0", padding)
		if rightPad {
			s += pad
		} else {
			s = pad + s
		}
	}
	result := make([]byte, len(s)/8)
	for i := 0; i < len(s); i += 8 {
		var b byte
		for j := range 8 {
			b <<= 1
			if s[i+j] == '1' {
				b |= 1
			}
		}
		result[i/8] = b
	}
	return result, valid
}

// isBareTypeIndex returns true for primitive/global type names that can appear
// directly in INDEX clauses without being object definitions.
func isBareTypeIndex(name string) bool {
	cls := wellKnownTypes[name]
	return cls != 0 && cls != typeClassSNMPv2TC && name != "OBJECT IDENTIFIER"
}

// isOIDType checks if the syntax resolves to OBJECT IDENTIFIER.
// For type references, it resolves the type and checks the effective base type,
// which handles all model.OID-based textual conventions (AutonomousType, VariablePointer,
// RowPointer, TDomain, etc.) without hardcoding names.
func isOIDType(ctx *resolverContext, mod *module.Module, syntax module.TypeSyntax) bool {
	switch s := syntax.(type) {
	case *module.TypeSyntaxObjectIdentifier:
		return true
	case *module.TypeSyntaxTypeRef:
		if s.Name == "OBJECT IDENTIFIER" {
			return true
		}
		if t, ok := ctx.resolveTypeForModule(mod, s.Name); ok {
			return t.EffectiveBase() == model.BaseObjectIdentifier
		}
		return false
	case *module.TypeSyntaxConstrained:
		return isOIDType(ctx, mod, s.Base)
	default:
		return false
	}
}
