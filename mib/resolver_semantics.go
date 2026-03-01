package mib

import (
	"encoding/hex"
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
	for _, mod := range ctx.Modules {
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
		if resolvedMod := ctx.ModuleToResolved[ref.mod]; resolvedMod != nil {
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

		ctx.Mib.addObject(resolved)

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

		if resolvedMod := ctx.ModuleToResolved[ref.mod]; resolvedMod != nil {
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

		resolvedMod := ctx.ModuleToResolved[ref.mod]
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
				if indexNode, ok := ctx.LookupNodeForModule(ref.mod, item.Object); ok {
					if indexNode.Object() != nil {
						indexEntries = append(indexEntries, IndexEntry{
							Object:  indexNode.Object(),
							Implied: item.Implied,
						})
					} else if !isBareTypeIndex(item.Object) {
						ctx.EmitDiagnostic(types.DiagIndexNotObject, SeverityMinor,
							ref.mod, obj.Span,
							"INDEX "+item.Object+" of "+obj.Name+" resolves to a node without an object definition")
					}
				}
			}
			resolvedObj.setIndex(indexEntries)
		}

		if obj.Augments != "" {
			if augNode, ok := ctx.LookupNodeForModule(ref.mod, obj.Augments); ok {
				if augNode.Object() != nil {
					resolvedObj.setAugments(augNode.Object())
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
	t := obj.typ
	if t == nil {
		return
	}

	for t != nil {
		if obj.hint == "" && t.hint != "" {
			obj.hint = t.hint
		}
		if len(obj.sizes) == 0 && len(t.sizes) > 0 {
			obj.sizes = t.sizes
		}
		if len(obj.ranges) == 0 && len(t.ranges) > 0 {
			obj.ranges = t.ranges
		}
		if len(obj.enums) == 0 && len(t.enums) > 0 {
			obj.enums = t.enums
		}
		if len(obj.bits) == 0 && len(t.bits) > 0 {
			obj.bits = t.bits
		}
		t = t.parent
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
		if resolvedMod := ctx.ModuleToResolved[ref.mod]; resolvedMod != nil {
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
		if resolvedMod := ctx.ModuleToResolved[ref.mod]; resolvedMod != nil {
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
		if resolvedMod := ctx.ModuleToResolved[ref.mod]; resolvedMod != nil {
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

func registerNotification(ctx *resolverContext, mod *module.Module, node *Node, resolved *Notification) {
	ctx.Mib.addNotification(resolved)
	var currentMod *Module
	if current := node.Notification(); current != nil {
		currentMod = current.Module()
	}
	if shouldPreferModule(ctx, currentMod, mod) {
		node.setNotification(resolved)
	}
	if resolvedMod := ctx.ModuleToResolved[mod]; resolvedMod != nil {
		resolvedMod.addNotification(resolved)
	}
}

func registerGroup(ctx *resolverContext, mod *module.Module, node *Node, resolved *Group) {
	ctx.Mib.addGroup(resolved)
	var currentMod *Module
	if current := node.Group(); current != nil {
		currentMod = current.Module()
	}
	if shouldPreferModule(ctx, currentMod, mod) {
		node.setGroup(resolved)
	}
	if resolvedMod := ctx.ModuleToResolved[mod]; resolvedMod != nil {
		resolvedMod.addGroup(resolved)
	}
}

func registerCompliance(ctx *resolverContext, mod *module.Module, node *Node, resolved *Compliance) {
	ctx.Mib.addCompliance(resolved)
	var currentMod *Module
	if current := node.Compliance(); current != nil {
		currentMod = current.Module()
	}
	if shouldPreferModule(ctx, currentMod, mod) {
		node.setCompliance(resolved)
	}
	if resolvedMod := ctx.ModuleToResolved[mod]; resolvedMod != nil {
		resolvedMod.addCompliance(resolved)
	}
}

func registerCapability(ctx *resolverContext, mod *module.Module, node *Node, resolved *Capability) {
	ctx.Mib.addCapability(resolved)
	var currentMod *Module
	if current := node.Capability(); current != nil {
		currentMod = current.Module()
	}
	if shouldPreferModule(ctx, currentMod, mod) {
		node.setCapability(resolved)
	}
	if resolvedMod := ctx.ModuleToResolved[mod]; resolvedMod != nil {
		resolvedMod.addCapability(resolved)
	}
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
		if resolvedMod := ctx.ModuleToResolved[ref.mod]; resolvedMod != nil {
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
		if resolvedMod := ctx.ModuleToResolved[ref.mod]; resolvedMod != nil {
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
		return nil, false
	case *module.TypeSyntaxBits:
		if t, ok := ctx.LookupType("BITS"); ok {
			return t, true
		}
		return nil, false
	case *module.TypeSyntaxOctetString:
		if t, ok := ctx.LookupType("OCTET STRING"); ok {
			return t, true
		}
		return nil, false
	case *module.TypeSyntaxObjectIdentifier:
		if t, ok := ctx.LookupType("OBJECT IDENTIFIER"); ok {
			return t, true
		}
		return nil, false
	case *module.TypeSyntaxSequenceOf, *module.TypeSyntaxSequence:
		return nil, false
	default:
		return nil, false
	}
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
		bytes := binaryToBytes(v.Value, isBits)
		dv := newDefValBytes(bytes, raw)
		return &dv
	case *module.DefValEnum:
		// Parser emits bare names as DefValEnum, but for OID-typed objects
		// the name is actually an OID reference.
		if isOIDType(syntax) {
			if node, ok := ctx.LookupNodeForModule(mod, v.Name); ok {
				oid := node.OID()
				dv := newDefValOID(oid, v.Name)
				return &dv
			}
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
		var name string
		switch c := v.Components[0].(type) {
		case *module.OidComponentName:
			name = c.NameValue
		case *module.OidComponentNamedNumber:
			name = c.NameValue
		case *module.OidComponentQualifiedName:
			name = c.NameValue
		case *module.OidComponentQualifiedNamedNumber:
			name = c.NameValue
		}
		if name == "" {
			ctx.EmitDiagnostic(types.DiagDefvalUnresolved, SeverityWarning,
				mod, types.Span{}, "DEFVAL OID value has no named root component")
			return nil
		}
		node, ok := ctx.LookupNodeForModule(mod, name)
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
			case *module.OidComponentQualifiedName:
				ctx.EmitDiagnostic(types.DiagDefvalUnresolved, SeverityWarning,
					mod, types.Span{}, "DEFVAL OID component "+c.ModuleValue+"."+c.NameValue+" has no numeric value")
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
func binaryToBytes(s string, rightPad bool) []byte {
	if s == "" {
		return []byte{}
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
	return result
}

// isBareTypeIndex returns true for primitive/global type names that can appear
// directly in INDEX clauses without being object definitions.
func isBareTypeIndex(name string) bool {
	if name == "OBJECT IDENTIFIER" {
		return false
	}
	return isASN1Primitive(name) || isSmiGlobalType(name) || isSmiV1GlobalType(name)
}

// isOIDType checks if the syntax resolves to OBJECT IDENTIFIER.
func isOIDType(syntax module.TypeSyntax) bool {
	switch s := syntax.(type) {
	case *module.TypeSyntaxObjectIdentifier:
		return true
	case *module.TypeSyntaxTypeRef:
		return s.Name == "OBJECT IDENTIFIER" || s.Name == "AutonomousType"
	case *module.TypeSyntaxConstrained:
		return isOIDType(s.Base)
	default:
		return false
	}
}
