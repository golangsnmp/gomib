package mib

import (
	"encoding/hex"
	"fmt"
	"log/slog"
	"slices"
	"strconv"
	"strings"

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
	checkIpAddressDeprecation(ctx, objRefs)
	checkInetAddressPairing(ctx, objRefs)
	checkTransportAddressPairing(ctx, objRefs)
}

func inferNodeKinds(ctx *resolverContext, objRefs []objectTypeRef) {
	tables, rows, scalars := 0, 0, 0
	var rowNodes []*Node

	// Track which module's OBJECT-TYPE determined each node's kind so
	// that only the preferred module's structural classification wins
	// when multiple modules define the same OID.
	nodeKindModule := make(map[*Node]*Module)

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
			node.setKind(KindTable)
			if prevKind != KindTable {
				tables++
			}
		} else if len(obj.Index) > 0 || obj.Augments != "" {
			node.setKind(KindRow)
			if prevKind != KindRow {
				rows++
			}
			rowNodes = append(rowNodes, node)
		} else {
			node.setKind(KindScalar)
			if prevKind != KindScalar {
				scalars++
			}
		}
		nodeKindModule[node] = ctx.moduleToResolved[ref.mod]
	}

	// Reclassify scalar children of row nodes as columns
	columns := 0
	seen := make(map[*Node]struct{})
	for _, row := range rowNodes {
		if _, dup := seen[row]; dup {
			continue
		}
		seen[row] = struct{}{}
		for _, child := range row.Children() {
			if child.Kind() == KindScalar {
				child.setKind(KindColumn)
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
						UnresolvedRef{Kind: UnresolvedIndex, Symbol: item.Object, Module: modName(ref.mod), Reason: reasonUnknownIndex})
				}
			}
		}

		if obj.Augments != "" {
			if _, ok := ctx.lookupNode(ref.mod, obj.Augments); !ok {
				ctx.recordUnresolved(types.DiagOidOrphan, ref.mod, obj.Spans.Augments,
					fmt.Sprintf("unresolved OID: %q references unknown parent %q", obj.Name, obj.Augments),
					UnresolvedRef{Kind: UnresolvedOID, Symbol: obj.Augments, Module: modName(ref.mod), Reason: reasonUnknownParent})
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

		resolved := newObject(obj.Name)
		resolved.setSpan(obj.Span)
		resolved.setSyntaxSpan(obj.Spans.Syntax)
		resolved.setAccessSpan(obj.Spans.Access)
		resolved.setStatusSpan(obj.Spans.Status)
		resolved.setDescriptionSpan(obj.Spans.Description)
		resolved.setReferenceSpan(obj.Spans.Reference)
		resolved.setUnitsSpan(obj.Spans.Units)
		resolved.setAugmentsSpan(obj.Spans.Augments)
		resolved.setDefaultValueSpan(obj.Spans.DefVal)
		resolved.setOidRefs(extractOidRefs(obj.DefinitionOid()))
		resolved.setNode(node)
		if resolvedMod := ctx.moduleToResolved[ref.mod]; resolvedMod != nil {
			resolved.setModule(resolvedMod)
		}
		resolved.setAccess(obj.Access)
		resolved.setStatus(obj.Status)
		resolved.setDescription(obj.Description)
		resolved.setUnits(obj.Units)
		resolved.setReference(obj.Reference)

		if t, ok := resolveTypeSyntax(ctx, obj.Syntax, ref.mod, obj.Name, obj.Spans.Syntax); ok {
			resolved.setType(t)
		}

		sizes, ranges := extractConstraints(obj.Syntax)
		resolved.setEffectiveSizes(sizes)
		resolved.setEffectiveRanges(ranges)
		if _, isBits := obj.Syntax.(*module.TypeSyntaxBits); isBits {
			resolved.setEffectiveBits(extractNamedValues(obj.Syntax))
		} else {
			resolved.setEffectiveEnums(extractNamedValues(obj.Syntax))
		}

		if obj.DefVal != nil {
			resolved.setDefaultValue(convertDefVal(ctx, obj.DefVal, ref.mod, obj.Syntax, obj.Span))
		}

		computeEffectiveValues(resolved)

		// Preserve the SEQUENCE type name from the row's SYNTAX clause.
		if node.Kind() == KindRow {
			if ref, ok := obj.Syntax.(*module.TypeSyntaxTypeRef); ok {
				resolved.setSequenceTypeName(ref.Name)
			}
		}

		ctx.mib.addObject(resolved)

		// Prefer SMIv2 modules when multiple modules define the same OID
		// (e.g., IF-MIB and RFC1213-MIB both define ifEntry).
		currentObj := node.Object()
		var currentMod *Module
		if currentObj != nil {
			currentMod = currentObj.Module()
		}
		if shouldPreferModule(ctx, currentMod, ref.mod) {
			node.setObject(resolved)
		}
		created++

		if resolvedMod := ctx.moduleToResolved[ref.mod]; resolvedMod != nil {
			resolvedMod.addObject(resolved)
			resolvedMod.addNode(node)
		}
	}

	linkObjectIndexes(ctx, objRefs)

	if ctx.TraceEnabled() {
		ctx.Trace("created resolved objects", slog.Int("count", created))
	}
}

// oidComponentNameAndModule extracts the symbolic name and optional module
// qualifier from an OID component. Returns empty strings for numeric-only
// components.
func oidComponentNameAndModule(comp module.OidComponent) (name, qualModule string) {
	switch c := comp.(type) {
	case *module.OidComponentName:
		return c.NameValue, ""
	case *module.OidComponentNamedNumber:
		return c.NameValue, ""
	case *module.OidComponentQualifiedName:
		return c.NameValue, c.ModuleValue
	case *module.OidComponentQualifiedNamedNumber:
		return c.NameValue, c.ModuleValue
	}
	return "", ""
}

// extractOidRefs extracts named symbolic references from an OID assignment.
func extractOidRefs(oid *module.OidAssignment) []OidRef {
	if oid == nil {
		return nil
	}
	var refs []OidRef
	for _, comp := range oid.Components {
		if name, _ := oidComponentNameAndModule(comp); name != "" {
			refs = append(refs, OidRef{Name: name, Span: comp.ComponentSpan()})
		}
	}
	return refs
}

// linkObjectIndexes resolves INDEX and AUGMENTS references now that all
// objects exist. Uses the module's own Object instance, not the shared
// node's, because multiple modules can define objects at the same OID.
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
			var indexEntries []IndexEntry
			for _, item := range obj.Index {
				if idxObj, nodeFound := ctx.resolveObject(ref.mod, item.Object); idxObj != nil {
					indexEntries = append(indexEntries, IndexEntry{
						Object:   idxObj,
						Implied:  item.Implied,
						Encoding: classifyIndexEncoding(idxObj, item.Implied),
						Span:     item.Span,
					})
				} else if isBareTypeIndex(item.Object) {
					indexEntries = append(indexEntries, IndexEntry{
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
			resolvedObj.setIndex(indexEntries)
		}

		if obj.Augments != "" {
			if target, nodeFound := ctx.resolveObject(ref.mod, obj.Augments); target != nil {
				resolvedObj.setAugments(target)
				target.addAugmentedBy(resolvedObj)
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

// computeEffectiveValues fills in display hints, size/range constraints,
// enums, and bits on the object by walking the type chain from child to root.
// Object-level values (set from the OBJECT-TYPE syntax) take precedence;
// only missing values are inherited from ancestor types. The first non-empty
// value found in the chain wins.
func computeEffectiveValues(obj *Object) {
	t := obj.Type()
	if t == nil {
		return
	}

	if obj.hint == "" {
		obj.hint = t.EffectiveDisplayHint()
	}
	if len(obj.sizes) == 0 {
		obj.sizes = t.EffectiveSizes()
	}
	if len(obj.ranges) == 0 {
		obj.ranges = t.EffectiveRanges()
	}
	if len(obj.enums) == 0 {
		obj.enums = t.EffectiveEnums()
	}
	if len(obj.bits) == 0 {
		obj.bits = t.EffectiveBits()
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

		resolved := newNotification(notif.Name)
		resolved.setSpan(notif.Span)
		resolved.setOidRefs(extractOidRefs(notif.DefinitionOid()))
		resolved.setNode(node)
		if resolvedMod := ctx.moduleToResolved[ref.mod]; resolvedMod != nil {
			resolved.setModule(resolvedMod)
		}
		resolved.setStatus(notif.Status)
		resolved.setDescription(notif.Description)
		resolved.setReference(notif.Reference)
		if notif.TrapInfo != nil {
			resolved.setTrapInfo(&TrapInfo{
				Enterprise: notif.TrapInfo.Enterprise,
				TrapNumber: notif.TrapInfo.TrapNumber,
			})
		}

		for _, objName := range notif.Objects {
			// addNotifObject adds an object to the notification, checking
			// not-accessible and emitting a diagnostic if needed.
			addNotifObject := func(obj *Object) {
				resolved.addObject(obj)
				if obj.Access() == AccessNotAccessible {
					ctx.EmitDiagnostic(types.DiagNotifObjectAccess,
						ref.mod, notif.Span,
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
					ctx.EmitDiagnostic(types.DiagNotifObjectNotObject, ref.mod, notif.Span,
						fmt.Sprintf("notification %q references %q which is not an object definition", notif.Name, objName))
					continue
				}
			}

			// Try node lookup to distinguish "not found" from "not an object".
			objNode, ok := ctx.lookupNode(ref.mod, objName)
			switch {
			case !ok:
				ctx.recordUnresolved(types.DiagObjectsUnresolved, ref.mod, notif.Span,
					fmt.Sprintf("unresolved OBJECTS: %q references unknown object %q", notif.Name, objName),
					UnresolvedRef{Kind: UnresolvedNotificationObject, Symbol: objName, Module: modName(ref.mod), Reason: reasonUnknownObject})
			case objNode.Object() == nil:
				ctx.EmitDiagnostic(types.DiagNotifObjectNotObject, ref.mod, notif.Span,
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
	status      Status
	description string
	reference   string
	members     []string
	oidRefs     []OidRef
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

		resolved := newGroup(gi.name)
		resolved.setSpan(gi.span)
		resolved.setOidRefs(gi.oidRefs)
		resolved.setNode(node)
		if resolvedMod := ctx.moduleToResolved[gi.mod]; resolvedMod != nil {
			resolved.setModule(resolvedMod)
		}
		resolved.setStatus(gi.status)
		resolved.setDescription(gi.description)
		resolved.setReference(gi.reference)
		resolved.setIsNotificationGroup(gi.isNotifGrp)

		var hasObjects, hasNotifications bool
		for _, memberName := range gi.members {
			if memberNode, ok := lookupMemberNode(ctx, gi.mod, memberName); ok {
				resolved.addMember(memberNode)
				kind := memberNode.Kind()
				if gi.isNotifGrp {
					if kind.IsObjectType() {
						hasObjects = true
						ctx.EmitDiagnostic(types.DiagGroupNotificationsObject,
							gi.mod, gi.span,
							fmt.Sprintf("notification group %q includes object %q", gi.name, memberName))
					} else if kind == types.KindNotification {
						hasNotifications = true
					}
				} else {
					if kind == types.KindNotification {
						hasNotifications = true
						ctx.EmitDiagnostic(types.DiagGroupObjectsNotification,
							gi.mod, gi.span,
							fmt.Sprintf("object group %q includes notification %q", gi.name, memberName))
					} else if kind.IsObjectType() {
						hasObjects = true
					}
					if obj := memberNode.Object(); obj != nil && obj.Access() == AccessNotAccessible {
						ctx.EmitDiagnostic(types.DiagGroupNotAccessible,
							gi.mod, gi.span,
							fmt.Sprintf("object %q of group %q must not be not-accessible", memberName, gi.name))
					}
				}
				checkGroupMemberStatus(ctx, gi.mod, gi.span, gi.status, gi.name, memberNode, memberName)
			} else {
				ctx.EmitDiagnostic(types.DiagGroupMemberUnresolved,
					gi.mod, gi.span,
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
// add to global Mib, conditionally set on node (preferring newer modules), add to per-module collection.
func registerResolvedEntity[T any](ctx *resolverContext, mod *module.Module, currentMod *Module, resolved T, addToMib, setOnNode func(T), addToModule func(*Module, T)) {
	addToMib(resolved)
	if shouldPreferModule(ctx, currentMod, mod) {
		setOnNode(resolved)
	}
	if resolvedMod := ctx.moduleToResolved[mod]; resolvedMod != nil {
		addToModule(resolvedMod, resolved)
	}
}

func registerNotification(ctx *resolverContext, mod *module.Module, node *Node, resolved *Notification) {
	var currentMod *Module
	if n := node.Notification(); n != nil {
		currentMod = n.Module()
	}
	registerResolvedEntity(ctx, mod, currentMod, resolved, ctx.mib.addNotification, node.setNotification, (*Module).addNotification)
	if resolvedMod := ctx.moduleToResolved[mod]; resolvedMod != nil {
		resolvedMod.addNode(node)
	}
}

func registerGroup(ctx *resolverContext, mod *module.Module, node *Node, resolved *Group) {
	var currentMod *Module
	if g := node.Group(); g != nil {
		currentMod = g.Module()
	}
	registerResolvedEntity(ctx, mod, currentMod, resolved, ctx.mib.addGroup, node.setGroup, (*Module).addGroup)
	if resolvedMod := ctx.moduleToResolved[mod]; resolvedMod != nil {
		resolvedMod.addNode(node)
	}
}

func registerCompliance(ctx *resolverContext, mod *module.Module, node *Node, resolved *Compliance) {
	var currentMod *Module
	if c := node.Compliance(); c != nil {
		currentMod = c.Module()
	}
	registerResolvedEntity(ctx, mod, currentMod, resolved, ctx.mib.addCompliance, node.setCompliance, (*Module).addCompliance)
	if resolvedMod := ctx.moduleToResolved[mod]; resolvedMod != nil {
		resolvedMod.addNode(node)
	}
}

func registerCapability(ctx *resolverContext, mod *module.Module, node *Node, resolved *Capability) {
	var currentMod *Module
	if c := node.Capability(); c != nil {
		currentMod = c.Module()
	}
	registerResolvedEntity(ctx, mod, currentMod, resolved, ctx.mib.addCapability, node.setCapability, (*Module).addCapability)
	if resolvedMod := ctx.moduleToResolved[mod]; resolvedMod != nil {
		resolvedMod.addNode(node)
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

		resolved := newCompliance(comp.Name)
		resolved.setSpan(comp.Span)
		resolved.setOidRefs(extractOidRefs(comp.DefinitionOid()))
		resolved.setNode(node)
		if resolvedMod := ctx.moduleToResolved[ref.mod]; resolvedMod != nil {
			resolved.setModule(resolvedMod)
		}
		resolved.setStatus(comp.Status)
		resolved.setDescription(comp.Description)
		resolved.setReference(comp.Reference)
		resolved.setModules(convertComplianceModules(ctx, ref.mod, comp.Modules))

		registerCompliance(ctx, ref.mod, node, resolved)
		created++
	}

	if ctx.TraceEnabled() {
		ctx.Trace("created resolved compliances", slog.Int("count", created))
	}
}

func convertComplianceModules(ctx *resolverContext, mod *module.Module, modules []module.ComplianceModule) []ComplianceModule {
	result := make([]ComplianceModule, len(modules))
	for i, m := range modules {
		result[i] = ComplianceModule{
			ModuleName:      m.ModuleName,
			MandatoryGroups: m.MandatoryGroups,
			Span:            m.Span,
		}
		if len(m.Groups) > 0 {
			groups := make([]ComplianceGroup, len(m.Groups))
			for j, g := range m.Groups {
				groups[j] = ComplianceGroup{
					Group:       g.Group,
					Description: g.Description,
					Span:        g.Span,
				}
			}
			result[i].Groups = groups
		}
		if len(m.Objects) > 0 {
			objects := make([]ComplianceObject, len(m.Objects))
			for j, o := range m.Objects {
				objects[j] = ComplianceObject{
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

			resolved := newCapability(ac.Name)
			resolved.setSpan(ac.Span)
			resolved.setOidRefs(extractOidRefs(ac.DefinitionOid()))
			resolved.setNode(node)
			if resolvedMod := ctx.moduleToResolved[mod]; resolvedMod != nil {
				resolved.setModule(resolvedMod)
			}
			resolved.setStatus(ac.Status)
			resolved.setDescription(ac.Description)
			resolved.setReference(ac.Reference)
			resolved.setProductRelease(ac.ProductRelease)
			resolved.setSupports(convertSupportsModules(ctx, mod, ac.Supports))

			registerCapability(ctx, mod, node, resolved)
			created++
		}
	}

	if ctx.TraceEnabled() {
		ctx.Trace("created resolved capabilities", slog.Int("count", created))
	}
}

func convertSupportsModules(ctx *resolverContext, mod *module.Module, modules []module.SupportsModule) []CapabilitiesModule {
	result := make([]CapabilitiesModule, len(modules))
	for i, m := range modules {
		result[i] = CapabilitiesModule{
			ModuleName: m.ModuleName,
			Includes:   m.Includes,
			Span:       m.Span,
		}
		for _, v := range m.Variations {
			if isNotificationVariation(ctx, mod, m.ModuleName, &v) {
				nv := NotificationVariation{
					Notification: v.Name,
					Description:  v.Description,
					Span:         v.Span,
				}
				if v.Access != nil {
					nv.Access = v.Access
					if *v.Access != AccessNotImplemented {
						ctx.EmitDiagnostic(types.DiagVariationAccessNotifOnly,
							mod, v.Span,
							fmt.Sprintf("notification variation %q ACCESS should be not-implemented per RFC 2580", v.Name))
					}
				}
				result[i].NotificationVariations = append(result[i].NotificationVariations, nv)
			} else {
				ov := ObjectVariation{
					Object:      v.Name,
					Description: v.Description,
					Span:        v.Span,
				}
				ov.Syntax = resolveSyntaxConstraints(ctx, v.Syntax, mod, v.Name, v.Span)
				ov.WriteSyntax = resolveSyntaxConstraints(ctx, v.WriteSyntax, mod, v.Name, v.Span)
				if len(v.CreationRequires) > 0 {
					ov.CreationRequires = slices.Clone(v.CreationRequires)
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
		return node.Kind() == KindNotification
	}
	// Try the defining module's import chain.
	if node, ok := ctx.lookupNode(mod, v.Name); ok {
		return node.Kind() == KindNotification
	}
	// Permissive only: try global lookup.
	if ctx.ResolverStrictness().AllowGlobalFallbacks() {
		if node, ok := ctx.lookupNodeGlobal(v.Name); ok {
			return node.Kind() == KindNotification
		}
	}
	// Cannot resolve; fall back to syntactic heuristic.
	return v.Syntax == nil && v.WriteSyntax == nil && len(v.CreationRequires) == 0 && v.DefVal == nil
}

func lookupMemberNode(ctx *resolverContext, mod *module.Module, name string) (*Node, bool) {
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
func memberNodeStatus(n *Node) (Status, bool) {
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
func checkGroupMemberStatus(ctx *resolverContext, mod *module.Module, span types.Span, groupStatus Status, groupName string, memberNode *Node, memberName string) {
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

func resolveSyntaxConstraints(ctx *resolverContext, syntax module.TypeSyntax, mod *module.Module, ownerName string, syntaxSpan types.Span) *SyntaxConstraints {
	if syntax == nil {
		return nil
	}
	sc := &SyntaxConstraints{}
	if t, ok := resolveTypeSyntax(ctx, syntax, mod, ownerName, syntaxSpan); ok {
		sc.Type = t
	}
	sc.Sizes, sc.Ranges = extractConstraints(syntax)
	if _, isBits := syntax.(*module.TypeSyntaxBits); isBits {
		sc.Bits = extractNamedValues(syntax)
	} else {
		sc.Enums = extractNamedValues(syntax)
	}
	return sc
}

func resolveTypeSyntax(ctx *resolverContext, syntax module.TypeSyntax, mod *module.Module, objectName string, span types.Span) (*Type, bool) {
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
			UnresolvedRef{Kind: UnresolvedType, Symbol: s.Name, Module: modName(mod), Reason: reasonUnknownType})
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
				UnresolvedRef{Kind: UnresolvedType, Symbol: s.Base, Module: modName(mod), Reason: reasonUnknownType})
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

func lookupPrimitiveType(ctx *resolverContext, mod *module.Module, span types.Span, objectName, typeName string) (*Type, bool) {
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

func convertDefVal(ctx *resolverContext, defval module.DefVal, mod *module.Module, syntax module.TypeSyntax, span types.Span) *DefVal {
	switch v := defval.(type) {
	case *module.DefValInteger:
		raw := strconv.FormatInt(v.Value, 10)
		dv := newDefValInt(v.Value, raw)
		return &dv
	case *module.DefValUnsigned:
		raw := strconv.FormatUint(v.Value, 10)
		dv := newDefValUint(v.Value, raw)
		return &dv
	case *module.DefValString:
		raw := `"` + v.Value + `"`
		dv := newDefValString(v.Value, raw)
		return &dv
	case *module.DefValHexString:
		raw := "'" + v.Value + "'H"
		bytes, err := hexToBytes(v.Value)
		if err != nil {
			ctx.EmitDiagnostic(types.DiagMalformedHexDefval,
				mod, span, fmt.Sprintf("malformed hex DEFVAL %q: %s", raw, err.Error()))
			return nil
		}
		dv := newDefValBytes(bytes, raw)
		return &dv
	case *module.DefValBinaryString:
		raw := "'" + v.Value + "'B"
		_, isBits := syntax.(*module.TypeSyntaxBits)
		bytes, valid := binaryToBytes(v.Value, isBits)
		if !valid {
			ctx.EmitDiagnostic(types.DiagMalformedBinDefval,
				mod, span, fmt.Sprintf("binary DEFVAL contains non-binary digits: %q", raw))
		}
		dv := newDefValBytes(bytes, raw)
		return &dv
	case *module.DefValEnum:
		// Parser emits bare names as DefValEnum, but for OID-typed objects
		// the name is actually an OID reference.
		if isOIDType(ctx, mod, syntax) {
			if node, ok := ctx.lookupNode(mod, v.Name); ok {
				oid := node.OID()
				dv := newDefValOID(oid, v.Name)
				return &dv
			}
			emitDefvalUnresolved(ctx, mod, span, fmt.Sprintf("DEFVAL OID reference %q could not be resolved", v.Name))
		}
		dv := newDefValEnum(v.Name, v.Name)
		return &dv
	case *module.DefValBits:
		raw := "{ " + strings.Join(v.Labels, ", ") + " }"
		if len(v.Labels) == 0 {
			raw = "{ }"
		}
		dv := newDefValBits(slices.Clone(v.Labels), raw)
		return &dv
	case *module.DefValOidRef:
		if node, ok := ctx.lookupNode(mod, v.Name); ok {
			oid := node.OID()
			dv := newDefValOID(oid, v.Name)
			return &dv
		}
		emitDefvalUnresolved(ctx, mod, span, fmt.Sprintf("DEFVAL OID reference %q could not be resolved", v.Name))
		return nil
	case *module.DefValOidValue:
		if len(v.Components) == 0 {
			emitDefvalUnresolved(ctx, mod, span, "DEFVAL OID value has no components")
			return nil
		}
		name, qualModule := oidComponentNameAndModule(v.Components[0])
		if name == "" {
			emitDefvalUnresolved(ctx, mod, span, "DEFVAL OID value has no named root component")
			return nil
		}
		var node *Node
		var ok bool
		if qualModule != "" {
			node, ok = ctx.lookupNodeByModuleName(qualModule, name)
		} else {
			node, ok = ctx.lookupNode(mod, name)
		}
		if !ok {
			emitDefvalUnresolved(ctx, mod, span, fmt.Sprintf("DEFVAL OID root %q could not be resolved", name))
			return nil
		}
		oid := node.OID()
		for _, comp := range v.Components[1:] {
			switch c := comp.(type) {
			case *module.OidComponentNumber:
				oid = append(oid, c.Value)
			case *module.OidComponentNamedNumber:
				oid = append(oid, c.NumberValue)
			case *module.OidComponentQualifiedNamedNumber:
				oid = append(oid, c.NumberValue)
			case *module.OidComponentName:
				emitDefvalUnresolved(ctx, mod, span, fmt.Sprintf("DEFVAL OID component %q has no numeric value", c.NameValue))
				return nil
			case *module.OidComponentQualifiedName:
				emitDefvalUnresolved(ctx, mod, span, fmt.Sprintf("DEFVAL OID component %q has no numeric value", c.ModuleValue+"."+c.NameValue))
				return nil
			}
		}
		dv := newDefValOID(oid, name)
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
// which handles all OID-based textual conventions (AutonomousType, VariablePointer,
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
			return t.EffectiveBase() == BaseObjectIdentifier
		}
		return false
	case *module.TypeSyntaxConstrained:
		return isOIDType(ctx, mod, s.Base)
	default:
		return false
	}
}
