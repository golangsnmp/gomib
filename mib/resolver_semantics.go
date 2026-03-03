package mib

import (
	"encoding/hex"
	"log/slog"
	"math"
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
	checkIntegerMisuse(ctx)
	checkRangeConstraints(ctx)
	checkIndexConstraints(ctx, objRefs)
	checkDefvalConstraints(ctx, objRefs)
}

func inferNodeKinds(ctx *resolverContext, objRefs []objectTypeRef) {
	tables, rows, scalars := 0, 0, 0
	var rowNodes []*Node

	for _, ref := range objRefs {
		obj := ref.obj
		node, ok := ctx.LookupNodeForModule(ref.mod, obj.Name)
		if !ok {
			continue
		}

		if _, isSequenceOf := obj.Syntax.(*module.TypeSyntaxSequenceOf); isSequenceOf {
			node.setKind(KindTable)
			tables++
		} else if len(obj.Index) > 0 || obj.Augments != "" {
			node.setKind(KindRow)
			rows++
			rowNodes = append(rowNodes, node)
		} else {
			node.setKind(KindScalar)
			scalars++
		}
	}

	// Reclassify scalar children of row nodes as columns
	columns := 0
	for _, row := range rowNodes {
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

// collectDefinitionRefs collects definitions matching matchFn across all modules.
func collectDefinitionRefs[T any](ctx *resolverContext, matchFn func(*module.Module, module.Definition) (T, bool)) []T {
	var refs []T
	for _, mod := range ctx.modules {
		for _, def := range mod.Definitions {
			if ref, ok := matchFn(mod, def); ok {
				refs = append(refs, ref)
			}
		}
	}
	return refs
}

func collectObjectTypeRefs(ctx *resolverContext) []objectTypeRef {
	return collectDefinitionRefs(ctx, func(mod *module.Module, def module.Definition) (objectTypeRef, bool) {
		if obj, ok := def.(*module.ObjectType); ok {
			return objectTypeRef{mod: mod, obj: obj}, true
		}
		return objectTypeRef{}, false
	})
}

func collectNotificationRefs(ctx *resolverContext) []notificationRef {
	return collectDefinitionRefs(ctx, func(mod *module.Module, def module.Definition) (notificationRef, bool) {
		if notif, ok := def.(*module.Notification); ok {
			return notificationRef{mod: mod, notif: notif}, true
		}
		return notificationRef{}, false
	})
}

func resolveTableSemantics(ctx *resolverContext, objRefs []objectTypeRef) {
	for _, ref := range objRefs {
		obj := ref.obj
		if len(obj.Index) == 0 && obj.Augments == "" {
			continue
		}

		if len(obj.Index) > 0 {
			for _, item := range obj.Index {
				if _, ok := ctx.LookupNodeForModule(ref.mod, item.Object); !ok {
					if isBareTypeIndex(item.Object) {
						continue
					}
					ctx.RecordUnresolvedIndex(ref.mod, obj.Name, item.Object, obj.Span)
				}
			}
		}

		if obj.Augments != "" {
			if _, ok := ctx.LookupNodeForModule(ref.mod, obj.Augments); !ok {
				ctx.RecordUnresolvedOid(ref.mod, obj.Name, obj.Augments, obj.Span)
			}
		}
	}
}

func createResolvedObjects(ctx *resolverContext, objRefs []objectTypeRef) {
	created := 0
	for _, ref := range objRefs {
		obj := ref.obj

		node, ok := ctx.LookupNodeForModule(ref.mod, obj.Name)
		if !ok {
			continue
		}

		resolved := newObject(obj.Name)
		resolved.setNode(node)
		if resolvedMod := ctx.moduleToResolved[ref.mod]; resolvedMod != nil {
			resolved.setModule(resolvedMod)
		}
		resolved.setAccess(obj.Access)
		resolved.setStatus(obj.Status)
		resolved.setDescription(obj.Description)
		resolved.setUnits(obj.Units)
		resolved.setReference(obj.Reference)

		if t, ok := resolveTypeSyntax(ctx, obj.Syntax, ref.mod, obj.Name, obj.Span); ok {
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
			resolved.setDefaultValue(convertDefVal(ctx, obj.DefVal, ref.mod, obj.Syntax))
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
		}
	}

	linkObjectIndexes(ctx, objRefs)

	if ctx.TraceEnabled() {
		ctx.Trace("created resolved objects", slog.Int("count", created))
	}
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
				if _, ok := ctx.LookupNodeForModule(ref.mod, item.Object); ok {
					// Use the module-scoped object lookup to get the correct
					// module's object, not the shared node's object which may
					// belong to a different module.
					if idxObj := ctx.lookupObjectInModuleScope(ref.mod, item.Object); idxObj != nil {
						indexEntries = append(indexEntries, IndexEntry{
							Object:   idxObj,
							Implied:  item.Implied,
							Encoding: classifyIndexEncoding(idxObj, item.Implied),
						})
					} else if !isBareTypeIndex(item.Object) {
						ctx.EmitDiagnostic(types.DiagIndexNotObject, SeverityMinor,
							ref.mod, obj.Span,
							"INDEX "+item.Object+" of "+obj.Name+" resolves to a node without an object definition")
					}
				} else if isBareTypeIndex(item.Object) {
					indexEntries = append(indexEntries, IndexEntry{
						TypeName: item.Object,
						Implied:  item.Implied,
					})
				}
			}
			resolvedObj.setIndex(indexEntries)
		}

		if obj.Augments != "" {
			if _, ok := ctx.LookupNodeForModule(ref.mod, obj.Augments); ok {
				// Use module-scoped lookup to get the correct module's object.
				if target := ctx.lookupObjectInModuleScope(ref.mod, obj.Augments); target != nil {
					resolvedObj.setAugments(target)
					target.addAugmentedBy(resolvedObj)
				} else {
					ctx.EmitDiagnostic(types.DiagAugmentsNotObject, SeverityMinor,
						ref.mod, obj.Span,
						"AUGMENTS target "+obj.Augments+" of "+obj.Name+" resolves to a node without an object definition")
				}
			}
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

		node, ok := ctx.LookupNodeForModule(ref.mod, notif.Name)
		if !ok {
			continue
		}

		resolved := newNotification(notif.Name)
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
			var objNode *Node
			var ok bool

			objNode, ok = ctx.LookupNodeForModule(ref.mod, objName)

			// Permissive only: global lookup for objects not explicitly imported
			if !ok && ctx.DiagnosticConfig().AllowBestGuessFallbacks() {
				objNode, ok = ctx.LookupNodeGlobal(objName)
				if ok && ctx.TraceEnabled() {
					ctx.Trace("permissive: resolved notification object via global lookup",
						slog.String("object", objName),
						slog.String("notification", notif.Name))
				}
			}

			switch {
			case ok && objNode.Object() != nil:
				resolved.addObject(objNode.Object())
			case !ok:
				ctx.RecordUnresolvedNotificationObject(ref.mod, notif.Name, objName, notif.Span)
			default:
				// Node exists but has no object definition (intermediate node
				// or non-object definition).
				ctx.EmitDiagnostic(types.DiagNotifObjectNotObject, SeverityMinor, ref.mod, notif.Span,
					"notification "+notif.Name+" references "+objName+" which is not an object definition")
			}
		}

		registerNotification(ctx, ref.mod, node, resolved)
		created++
	}

	if ctx.TraceEnabled() {
		ctx.Trace("created resolved notifications", slog.Int("count", created))
	}
}

type objectGroupRef struct {
	mod *module.Module
	grp *module.ObjectGroup
}

type notificationGroupRef struct {
	mod *module.Module
	grp *module.NotificationGroup
}

func createResolvedGroups(ctx *resolverContext) {
	objCount := createResolvedObjectGroups(ctx)
	notifCount := createResolvedNotificationGroups(ctx)

	if ctx.TraceEnabled() {
		ctx.Trace("created resolved groups", slog.Int("count", objCount+notifCount))
	}
}

func createResolvedObjectGroups(ctx *resolverContext) int {
	created := 0
	for _, ref := range collectDefinitionRefs(ctx, func(mod *module.Module, def module.Definition) (objectGroupRef, bool) {
		if grp, ok := def.(*module.ObjectGroup); ok {
			return objectGroupRef{mod: mod, grp: grp}, true
		}
		return objectGroupRef{}, false
	}) {
		grp := ref.grp
		node, ok := ctx.LookupNodeForModule(ref.mod, grp.Name)
		if !ok {
			continue
		}

		resolved := newGroup(grp.Name)
		resolved.setNode(node)
		if resolvedMod := ctx.moduleToResolved[ref.mod]; resolvedMod != nil {
			resolved.setModule(resolvedMod)
		}
		resolved.setStatus(grp.Status)
		resolved.setDescription(grp.Description)
		resolved.setReference(grp.Reference)

		for _, memberName := range grp.Objects {
			if memberNode, ok := lookupMemberNode(ctx, ref.mod, memberName); ok {
				resolved.addMember(memberNode)
				if obj := memberNode.Object(); obj != nil && obj.Access() == AccessNotAccessible {
					ctx.EmitDiagnostic(types.DiagGroupNotAccessible, SeverityMinor,
						ref.mod, grp.DefinitionSpan(),
						"object "+memberName+" of group "+grp.Name+" must not be not-accessible")
				}
			} else {
				ctx.EmitDiagnostic(types.DiagGroupMemberUnresolved, SeverityError,
					ref.mod, grp.DefinitionSpan(),
					"group "+grp.Name+" references unresolved member "+memberName)
			}
		}

		registerGroup(ctx, ref.mod, node, resolved)
		created++
	}
	return created
}

func createResolvedNotificationGroups(ctx *resolverContext) int {
	created := 0
	for _, ref := range collectDefinitionRefs(ctx, func(mod *module.Module, def module.Definition) (notificationGroupRef, bool) {
		if grp, ok := def.(*module.NotificationGroup); ok {
			return notificationGroupRef{mod: mod, grp: grp}, true
		}
		return notificationGroupRef{}, false
	}) {
		grp := ref.grp
		node, ok := ctx.LookupNodeForModule(ref.mod, grp.Name)
		if !ok {
			continue
		}

		resolved := newGroup(grp.Name)
		resolved.setNode(node)
		if resolvedMod := ctx.moduleToResolved[ref.mod]; resolvedMod != nil {
			resolved.setModule(resolvedMod)
		}
		resolved.setStatus(grp.Status)
		resolved.setDescription(grp.Description)
		resolved.setReference(grp.Reference)
		resolved.setIsNotificationGroup(true)

		for _, memberName := range grp.Notifications {
			if memberNode, ok := lookupMemberNode(ctx, ref.mod, memberName); ok {
				resolved.addMember(memberNode)
			} else {
				ctx.EmitDiagnostic(types.DiagGroupMemberUnresolved, SeverityError,
					ref.mod, grp.DefinitionSpan(),
					"notification group "+grp.Name+" references unresolved member "+memberName)
			}
		}

		registerGroup(ctx, ref.mod, node, resolved)
		created++
	}
	return created
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
}

func registerGroup(ctx *resolverContext, mod *module.Module, node *Node, resolved *Group) {
	var currentMod *Module
	if g := node.Group(); g != nil {
		currentMod = g.Module()
	}
	registerResolvedEntity(ctx, mod, currentMod, resolved, ctx.mib.addGroup, node.setGroup, (*Module).addGroup)
}

func registerCompliance(ctx *resolverContext, mod *module.Module, node *Node, resolved *Compliance) {
	var currentMod *Module
	if c := node.Compliance(); c != nil {
		currentMod = c.Module()
	}
	registerResolvedEntity(ctx, mod, currentMod, resolved, ctx.mib.addCompliance, node.setCompliance, (*Module).addCompliance)
}

func registerCapability(ctx *resolverContext, mod *module.Module, node *Node, resolved *Capability) {
	var currentMod *Module
	if c := node.Capability(); c != nil {
		currentMod = c.Module()
	}
	registerResolvedEntity(ctx, mod, currentMod, resolved, ctx.mib.addCapability, node.setCapability, (*Module).addCapability)
}

type complianceRef struct {
	mod  *module.Module
	comp *module.ModuleCompliance
}

type capabilitiesRef struct {
	mod *module.Module
	cap *module.AgentCapabilities
}

func createResolvedCompliances(ctx *resolverContext) {
	created := 0
	for _, ref := range collectDefinitionRefs(ctx, func(mod *module.Module, def module.Definition) (complianceRef, bool) {
		if comp, ok := def.(*module.ModuleCompliance); ok {
			return complianceRef{mod: mod, comp: comp}, true
		}
		return complianceRef{}, false
	}) {
		comp := ref.comp
		node, ok := ctx.LookupNodeForModule(ref.mod, comp.Name)
		if !ok {
			continue
		}

		resolved := newCompliance(comp.Name)
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
		}
		if len(m.Groups) > 0 {
			groups := make([]ComplianceGroup, len(m.Groups))
			for j, g := range m.Groups {
				groups[j] = ComplianceGroup{
					Group:       g.Group,
					Description: g.Description,
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
				}
				objects[j].Syntax = resolveSyntaxConstraints(ctx, o.Syntax, mod, o.Object)
				objects[j].WriteSyntax = resolveSyntaxConstraints(ctx, o.WriteSyntax, mod, o.Object)
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
	for _, ref := range collectDefinitionRefs(ctx, func(mod *module.Module, def module.Definition) (capabilitiesRef, bool) {
		if ac, ok := def.(*module.AgentCapabilities); ok {
			return capabilitiesRef{mod: mod, cap: ac}, true
		}
		return capabilitiesRef{}, false
	}) {
		ac := ref.cap
		node, ok := ctx.LookupNodeForModule(ref.mod, ac.Name)
		if !ok {
			continue
		}

		resolved := newCapability(ac.Name)
		resolved.setNode(node)
		if resolvedMod := ctx.moduleToResolved[ref.mod]; resolvedMod != nil {
			resolved.setModule(resolvedMod)
		}
		resolved.setStatus(ac.Status)
		resolved.setDescription(ac.Description)
		resolved.setReference(ac.Reference)
		resolved.setProductRelease(ac.ProductRelease)
		resolved.setSupports(convertSupportsModules(ctx, ref.mod, ac.Supports))

		registerCapability(ctx, ref.mod, node, resolved)
		created++
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
		}
		for _, v := range m.Variations {
			if isNotificationVariation(ctx, mod, m.ModuleName, &v) {
				nv := NotificationVariation{
					Notification: v.Name,
					Description:  v.Description,
				}
				if v.Access != nil {
					nv.Access = v.Access
					if *v.Access != AccessNotImplemented {
						ctx.EmitDiagnostic(types.DiagVariationAccessNotifOnly, SeverityMinor,
							mod, types.Span{},
							"notification variation "+v.Name+" ACCESS should be not-implemented per RFC 2580")
					}
				}
				result[i].NotificationVariations = append(result[i].NotificationVariations, nv)
			} else {
				ov := ObjectVariation{
					Object:      v.Name,
					Description: v.Description,
				}
				ov.Syntax = resolveSyntaxConstraints(ctx, v.Syntax, mod, v.Name)
				ov.WriteSyntax = resolveSyntaxConstraints(ctx, v.WriteSyntax, mod, v.Name)
				if len(v.CreationRequires) > 0 {
					ov.CreationRequires = slices.Clone(v.CreationRequires)
				}
				if v.Access != nil {
					ov.Access = v.Access
				}
				if v.DefVal != nil {
					if dv := convertDefVal(ctx, v.DefVal, mod, v.Syntax); dv != nil {
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
// the defining module's imports, and (permissively) globally. If the name
// cannot be resolved, it falls back to a syntactic heuristic: variations with
// SYNTAX, WRITE-SYNTAX, CREATION-REQUIRES, or DEFVAL are object variations.
func isNotificationVariation(ctx *resolverContext, mod *module.Module, supportsModule string, v *module.Variation) bool {
	// Try the SUPPORTS module first (most correct per RFC 2580).
	if node, ok := ctx.LookupNodeInModule(supportsModule, v.Name); ok {
		return node.Kind() == KindNotification
	}
	// Try the defining module's import chain.
	if node, ok := ctx.LookupNodeForModule(mod, v.Name); ok {
		return node.Kind() == KindNotification
	}
	// Permissive: try global lookup.
	if ctx.DiagnosticConfig().AllowBestGuessFallbacks() {
		if node, ok := ctx.LookupNodeGlobal(v.Name); ok {
			return node.Kind() == KindNotification
		}
	}
	// Cannot resolve; fall back to syntactic heuristic.
	return v.Syntax == nil && v.WriteSyntax == nil && len(v.CreationRequires) == 0 && v.DefVal == nil
}

func lookupMemberNode(ctx *resolverContext, mod *module.Module, name string) (*Node, bool) {
	node, ok := ctx.LookupNodeForModule(mod, name)
	if ok {
		return node, true
	}
	if ctx.DiagnosticConfig().AllowBestGuessFallbacks() {
		node, ok = ctx.LookupNodeGlobal(name)
		if ok && ctx.TraceEnabled() {
			ctx.Trace("permissive: resolved group member via global lookup",
				slog.String("member", name),
				slog.String("module", mod.Name))
		}
		return node, ok
	}
	return nil, false
}

func resolveSyntaxConstraints(ctx *resolverContext, syntax module.TypeSyntax, mod *module.Module, ownerName string) *SyntaxConstraints {
	if syntax == nil {
		return nil
	}
	sc := &SyntaxConstraints{}
	if t, ok := resolveTypeSyntax(ctx, syntax, mod, ownerName, types.Span{}); ok {
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
		if t, ok := ctx.LookupTypeForModule(mod, s.Name); ok {
			return t, true
		}
		// Don't emit unresolved error for SEQUENCE type references (row SYNTAX).
		// SEQUENCE types are structural and intentionally not registered.
		if isSequenceTypeDef(ctx, mod, s.Name) {
			return nil, false
		}
		ctx.RecordUnresolvedType(mod, objectName, s.Name, span)
		return nil, false
	case *module.TypeSyntaxConstrained:
		return resolveTypeSyntax(ctx, s.Base, mod, objectName, span)
	case *module.TypeSyntaxIntegerEnum:
		// Named base type (e.g., TPSPRateType { kbps(1) })
		if s.Base != "" {
			if t, ok := ctx.LookupTypeForModule(mod, s.Base); ok {
				return t, true
			}
			ctx.RecordUnresolvedType(mod, objectName, s.Base, span)
			return nil, false
		}
		// Bare INTEGER { ... } enum with no named base
		if t, ok := ctx.LookupType("INTEGER"); ok {
			return t, true
		}
		ctx.EmitDiagnostic(types.DiagPrimitiveTypeMissing, SeverityError,
			mod, span, "primitive type INTEGER not found for "+objectName)
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
	if t, ok := ctx.LookupType(typeName); ok {
		return t, true
	}
	ctx.EmitDiagnostic(types.DiagPrimitiveTypeMissing, SeverityError,
		mod, span, "primitive type "+typeName+" not found for "+objectName)
	return nil, false
}

// isSequenceTypeDef checks whether name refers to a SEQUENCE type definition
// in the module or its imports. Used to suppress spurious unresolved type
// diagnostics for row OBJECT-TYPE SYNTAX references (e.g., SYNTAX IfEntry).
func isSequenceTypeDef(ctx *resolverContext, mod *module.Module, name string) bool {
	if hasSequenceTypeDef(mod, name) {
		return true
	}
	if imports := ctx.moduleImports[mod]; imports != nil {
		if srcMod := imports[name]; srcMod != nil {
			return hasSequenceTypeDef(srcMod, name)
		}
	}
	return false
}

func hasSequenceTypeDef(mod *module.Module, name string) bool {
	for _, def := range mod.Definitions {
		if td, ok := def.(*module.TypeDef); ok && td.Name == name {
			_, isSeq := td.Syntax.(*module.TypeSyntaxSequence)
			return isSeq
		}
	}
	return false
}

func convertDefVal(ctx *resolverContext, defval module.DefVal, mod *module.Module, syntax module.TypeSyntax) *DefVal {
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
			ctx.EmitDiagnostic(types.DiagMalformedHexDefval, SeverityWarning,
				mod, types.Span{}, "malformed hex DEFVAL "+raw+": "+err.Error())
			return nil
		}
		dv := newDefValBytes(bytes, raw)
		return &dv
	case *module.DefValBinaryString:
		raw := "'" + v.Value + "'B"
		_, isBits := syntax.(*module.TypeSyntaxBits)
		bytes, valid := binaryToBytes(v.Value, isBits)
		if !valid {
			ctx.EmitDiagnostic(types.DiagMalformedBinDefval, SeverityWarning,
				mod, types.Span{}, "binary DEFVAL contains non-binary digits: "+raw)
		}
		dv := newDefValBytes(bytes, raw)
		return &dv
	case *module.DefValEnum:
		// Parser emits bare names as DefValEnum, but for OID-typed objects
		// the name is actually an OID reference.
		if isOIDType(ctx, mod, syntax) {
			if node, ok := ctx.LookupNodeForModule(mod, v.Name); ok {
				oid := node.OID()
				dv := newDefValOID(oid, v.Name)
				return &dv
			}
			ctx.EmitDiagnostic(types.DiagDefvalUnresolved, SeverityWarning,
				mod, types.Span{}, "DEFVAL OID reference "+v.Name+" could not be resolved")
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
		if node, ok := ctx.LookupNodeForModule(mod, v.Name); ok {
			oid := node.OID()
			dv := newDefValOID(oid, v.Name)
			return &dv
		}
		ctx.EmitDiagnostic(types.DiagDefvalUnresolved, SeverityWarning,
			mod, types.Span{}, "DEFVAL OID reference "+v.Name+" could not be resolved")
		return nil
	case *module.DefValOidValue:
		if len(v.Components) == 0 {
			ctx.EmitDiagnostic(types.DiagDefvalUnresolved, SeverityWarning,
				mod, types.Span{}, "DEFVAL OID value has no components")
			return nil
		}
		var name, qualModule string
		switch c := v.Components[0].(type) {
		case *module.OidComponentName:
			name = c.NameValue
		case *module.OidComponentNamedNumber:
			name = c.NameValue
		case *module.OidComponentQualifiedName:
			qualModule = c.ModuleValue
			name = c.NameValue
		case *module.OidComponentQualifiedNamedNumber:
			qualModule = c.ModuleValue
			name = c.NameValue
		}
		if name == "" {
			ctx.EmitDiagnostic(types.DiagDefvalUnresolved, SeverityWarning,
				mod, types.Span{}, "DEFVAL OID value has no named root component")
			return nil
		}
		var node *Node
		var ok bool
		if qualModule != "" {
			node, ok = ctx.LookupNodeInModule(qualModule, name)
		} else {
			node, ok = ctx.LookupNodeForModule(mod, name)
		}
		if !ok {
			ctx.EmitDiagnostic(types.DiagDefvalUnresolved, SeverityWarning,
				mod, types.Span{}, "DEFVAL OID root "+name+" could not be resolved")
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
				ctx.EmitDiagnostic(types.DiagDefvalUnresolved, SeverityWarning,
					mod, types.Span{}, "DEFVAL OID component "+c.NameValue+" has no numeric value")
				return nil
			case *module.OidComponentQualifiedName:
				ctx.EmitDiagnostic(types.DiagDefvalUnresolved, SeverityWarning,
					mod, types.Span{}, "DEFVAL OID component "+c.ModuleValue+"."+c.NameValue+" has no numeric value")
				return nil
			}
		}
		dv := newDefValOID(oid, name)
		return &dv
	default:
		ctx.EmitDiagnostic(types.DiagDefvalUnresolved, SeverityWarning,
			mod, types.Span{}, "DEFVAL could not be parsed")
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
		if t, ok := ctx.LookupTypeForModule(mod, s.Name); ok {
			return t.EffectiveBase() == BaseObjectIdentifier
		}
		return false
	case *module.TypeSyntaxConstrained:
		return isOIDType(ctx, mod, s.Base)
	default:
		return false
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
					ctx.EmitDiagnostic(types.DiagIntegerInSMIv2, SeverityWarning,
						mod, d.Span,
						d.Name+": use Integer32 instead of INTEGER in SMIv2")
				}
			case *module.TypeDef:
				if isIntegerKeywordSyntax(d.Syntax) {
					ctx.EmitDiagnostic(types.DiagIntegerInSMIv2, SeverityWarning,
						mod, d.Span,
						d.Name+": use Integer32 instead of INTEGER in SMIv2")
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

// checkIndexConstraints checks integer-valued INDEX elements for missing
// range restrictions and negative ranges. Per RFC 2578 section 7.7,
// integer index values encode as a single OID sub-identifier, which
// requires non-negative bounded values.
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
			t := idxObj.Type()
			if t == nil {
				continue
			}
			if t.EffectiveBase() != BaseInteger32 {
				continue
			}

			ranges := idxObj.EffectiveRanges()
			enums := idxObj.EffectiveEnums()

			// Enum-valued indexes: check for negative enum values.
			if len(enums) > 0 {
				for _, e := range enums {
					if e.Value < 0 {
						ctx.EmitDiagnostic(types.DiagIndexNegativeRange, SeverityError,
							ref.mod, obj.Span,
							"INDEX "+item.Object+" of "+obj.Name+" has negative enumeration value "+e.Label)
						break
					}
				}
				continue
			}

			// No range restriction on an integer index.
			if len(ranges) == 0 {
				ctx.EmitDiagnostic(types.DiagIndexIntegerNoRange, SeverityError,
					ref.mod, obj.Span,
					"INDEX "+item.Object+" of "+obj.Name+" has no range restriction")
				continue
			}

			// Range includes negative values.
			for _, r := range ranges {
				if r.Min < 0 {
					ctx.EmitDiagnostic(types.DiagIndexNegativeRange, SeverityError,
						ref.mod, obj.Span,
						"INDEX "+item.Object+" of "+obj.Name+" has range permitting negative values")
					break
				}
			}
		}
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
		ranges := resolved.EffectiveRanges()
		enums := resolved.EffectiveEnums()
		bits := resolved.EffectiveBits()

		switch dv.Kind() {
		case DefValKindInt:
			v, _ := DefValAs[int64](dv)
			checkDefvalInt(ctx, ref.mod, obj, v, base, ranges, enums)
		case DefValKindUint:
			v, _ := DefValAs[uint64](dv)
			checkDefvalUint(ctx, ref.mod, obj, v, base, ranges, enums)
		case DefValKindEnum:
			label, _ := DefValAs[string](dv)
			if len(enums) > 0 {
				if _, ok := findNamedValue(enums, label); !ok {
					ctx.EmitDiagnostic(types.DiagDefvalEnum, SeverityWarning,
						ref.mod, obj.Span,
						obj.Name+": DEFVAL enum label "+label+" not defined in type")
				}
			}
		case DefValKindBits:
			labels, _ := DefValAs[[]string](dv)
			for _, label := range labels {
				if _, ok := findNamedValue(bits, label); !ok {
					ctx.EmitDiagnostic(types.DiagDefvalEnum, SeverityWarning,
						ref.mod, obj.Span,
						obj.Name+": DEFVAL BITS label "+label+" not defined in type")
				}
			}
		case DefValKindUnset, DefValKindString, DefValKindBytes, DefValKindOID:
			// No validation for these kinds.
		}
	}
}

func checkDefvalInt(ctx *resolverContext, mod *module.Module, obj *module.ObjectType, v int64, base BaseType, ranges []Range, enums []NamedValue) {
	// Check basetype limits.
	switch base {
	case BaseInteger32:
		if v < math.MinInt32 || v > math.MaxInt32 {
			ctx.EmitDiagnostic(types.DiagDefvalBasetype, SeverityWarning,
				mod, obj.Span,
				obj.Name+": DEFVAL "+strconv.FormatInt(v, 10)+" exceeds Integer32 range")
		}
	case BaseUnsigned32, BaseGauge32, BaseCounter32, BaseTimeTicks:
		if v < 0 || v > math.MaxUint32 {
			ctx.EmitDiagnostic(types.DiagDefvalBasetype, SeverityWarning,
				mod, obj.Span,
				obj.Name+": DEFVAL "+strconv.FormatInt(v, 10)+" exceeds unsigned32 range")
		}
	}

	// Check RANGE constraints.
	if len(ranges) > 0 {
		if !valueInRanges(v, ranges) {
			ctx.EmitDiagnostic(types.DiagDefvalRange, SeverityWarning,
				mod, obj.Span,
				obj.Name+": DEFVAL "+strconv.FormatInt(v, 10)+" outside RANGE constraint")
		}
	}

	// Check enum membership for integer DEFVALs.
	if len(enums) > 0 {
		found := false
		for _, e := range enums {
			if e.Value == v {
				found = true
				break
			}
		}
		if !found {
			ctx.EmitDiagnostic(types.DiagDefvalEnum, SeverityWarning,
				mod, obj.Span,
				obj.Name+": DEFVAL "+strconv.FormatInt(v, 10)+" does not match any enumeration value")
		}
	}
}

func checkDefvalUint(ctx *resolverContext, mod *module.Module, obj *module.ObjectType, v uint64, base BaseType, ranges []Range, enums []NamedValue) {
	// Check basetype limits.
	switch base {
	case BaseInteger32:
		if v > math.MaxInt32 {
			ctx.EmitDiagnostic(types.DiagDefvalBasetype, SeverityWarning,
				mod, obj.Span,
				obj.Name+": DEFVAL "+strconv.FormatUint(v, 10)+" exceeds Integer32 range")
		}
	case BaseUnsigned32, BaseGauge32, BaseCounter32, BaseTimeTicks:
		if v > math.MaxUint32 {
			ctx.EmitDiagnostic(types.DiagDefvalBasetype, SeverityWarning,
				mod, obj.Span,
				obj.Name+": DEFVAL "+strconv.FormatUint(v, 10)+" exceeds unsigned32 range")
		}
	}

	// Check RANGE constraints.
	if len(ranges) > 0 {
		if !uvalueInRanges(v, ranges) {
			ctx.EmitDiagnostic(types.DiagDefvalRange, SeverityWarning,
				mod, obj.Span,
				obj.Name+": DEFVAL "+strconv.FormatUint(v, 10)+" outside RANGE constraint")
		}
	}

	// Check enum membership for unsigned integer DEFVALs.
	if len(enums) > 0 {
		found := false
		for _, e := range enums {
			if e.Value >= 0 && uint64(e.Value) == v {
				found = true
				break
			}
		}
		if !found {
			ctx.EmitDiagnostic(types.DiagDefvalEnum, SeverityWarning,
				mod, obj.Span,
				obj.Name+": DEFVAL "+strconv.FormatUint(v, 10)+" does not match any enumeration value")
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
		ctx.EmitDiagnostic(types.DiagSizeIllegal, SeverityError,
			mod, span,
			name+": SIZE constraint illegal for non-octet-string type")
		return
	}

	// Value range only applies to numeric types.
	if !isSize && !isNumericBase(base) {
		ctx.EmitDiagnostic(types.DiagRangeIllegal, SeverityError,
			mod, span,
			name+": range constraint illegal for non-numerical type")
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
			ctx.EmitDiagnostic(types.DiagRangeExchanged, SeverityError,
				mod, span,
				name+": range "+r.String()+" has exchanged limits")
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
				ctx.EmitDiagnostic(types.DiagRangeAscending, SeverityWarning,
					mod, span,
					name+": ranges not in ascending order")
			}
			if r.Min <= prev.Max {
				ctx.EmitDiagnostic(types.DiagRangeOverlap, SeverityError,
					mod, span,
					name+": range "+r.String()+" overlaps with "+prev.String())
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
		ctx.EmitDiagnostic(types.DiagRangeBounds, SeverityError,
			mod, span,
			name+": range "+which+" bound "+strconv.FormatInt(v, 10)+" exceeds basetype")
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
