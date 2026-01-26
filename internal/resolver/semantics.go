package resolver

import (
	"log/slog"
	"slices"

	"github.com/golangsnmp/gomib/internal/mibimpl"
	"github.com/golangsnmp/gomib/internal/module"
	"github.com/golangsnmp/gomib/internal/types"
	"github.com/golangsnmp/gomib/mib"
)

func analyzeSemantics(ctx *ResolverContext) {
	// Collect object type refs once and pass to all functions
	objRefs := collectObjectTypeRefs(ctx)
	inferNodeKinds(ctx, objRefs)
	resolveTableSemantics(ctx, objRefs)
	createResolvedObjects(ctx, objRefs)
	createResolvedNotifications(ctx)
}

func inferNodeKinds(ctx *ResolverContext, objRefs []objectTypeRef) {
	tables, rows, scalars := 0, 0, 0
	var rowNodes []*mibimpl.Node

	for _, ref := range objRefs {
		obj := ref.obj
		node, ok := ctx.LookupNodeForModule(ref.mod, obj.Name)
		if !ok {
			continue
		}

		if _, isSequenceOf := obj.Syntax.(*module.TypeSyntaxSequenceOf); isSequenceOf {
			node.SetKind(mib.KindTable)
			tables++
		} else if len(obj.Index) > 0 || obj.Augments != "" {
			node.SetKind(mib.KindRow)
			rows++
			rowNodes = append(rowNodes, node)
		} else {
			node.SetKind(mib.KindScalar)
			scalars++
		}
	}

	// Collect row children inline instead of separate tree walk
	columns := 0
	for _, row := range rowNodes {
		for _, child := range row.Children() {
			// Type assert to get concrete node
			if childNode, ok := child.(*mibimpl.Node); ok {
				if childNode.Kind() == mib.KindScalar {
					childNode.SetKind(mib.KindColumn)
					columns++
				}
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

// collectDefinitionRefs iterates all definitions across modules, applying
// matchFn to each. If matchFn returns a non-nil result, it's appended to the
// returned slice.
func collectDefinitionRefs[T any](ctx *ResolverContext, matchFn func(*module.Module, module.Definition) (T, bool)) []T {
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

func collectObjectTypeRefs(ctx *ResolverContext) []objectTypeRef {
	return collectDefinitionRefs(ctx, func(mod *module.Module, def module.Definition) (objectTypeRef, bool) {
		if obj, ok := def.(*module.ObjectType); ok {
			return objectTypeRef{mod: mod, obj: obj}, true
		}
		return objectTypeRef{}, false
	})
}

func collectNotificationRefs(ctx *ResolverContext) []notificationRef {
	return collectDefinitionRefs(ctx, func(mod *module.Module, def module.Definition) (notificationRef, bool) {
		if notif, ok := def.(*module.Notification); ok {
			return notificationRef{mod: mod, notif: notif}, true
		}
		return notificationRef{}, false
	})
}

func resolveTableSemantics(ctx *ResolverContext, objRefs []objectTypeRef) {
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

func createResolvedObjects(ctx *ResolverContext, objRefs []objectTypeRef) {
	created := 0
	for _, ref := range objRefs {
		obj := ref.obj

		node, ok := ctx.LookupNodeForModule(ref.mod, obj.Name)
		if !ok {
			continue
		}

		resolved := mibimpl.NewObject(obj.Name)
		resolved.SetNode(node)
		resolved.SetModule(ctx.ModuleToResolved[ref.mod])
		resolved.SetAccess(convertAccess(obj.Access))
		resolved.SetStatus(convertStatus(obj.Status))
		resolved.SetDescription(obj.Description)
		resolved.SetUnits(obj.Units)
		resolved.SetReference(obj.Reference)

		// Resolve type and extract inline constraints
		if t, ok := resolveTypeSyntax(ctx, obj.Syntax, ref.mod, obj.Name, obj.Span); ok {
			resolved.SetType(t)
		}

		// Extract inline constraints and named values
		sizes, ranges := extractConstraints(obj.Syntax)
		resolved.SetEffectiveSizes(sizes)
		resolved.SetEffectiveRanges(ranges)
		resolved.SetEffectiveEnums(extractNamedValues(obj.Syntax))

		// INDEX and AUGMENTS are resolved in a second pass after all objects exist
		// This ensures index objects exist when we try to link them

		// Convert DEFVAL
		if obj.DefVal != nil {
			resolved.SetDefaultValue(convertDefVal(ctx, obj.DefVal, ref.mod))
		}

		// Pre-compute effective values from type chain
		computeEffectiveValues(resolved)

		ctx.Builder.AddObject(resolved)

		// Only set this Object on the node if this module is preferred over
		// any existing module. This handles cases where multiple modules define
		// the same OID (e.g., IF-MIB and RFC1213-MIB both define ifEntry).
		// We prefer SMIv2 modules over SMIv1.
		currentObj := node.InternalObject()
		var currentMod *mibimpl.Module
		if currentObj != nil {
			currentMod = currentObj.InternalModule()
		}
		newMod := ctx.ModuleToResolved[ref.mod]
		if shouldPreferModule(newMod, currentMod, ref.mod, ctx) {
			node.SetObject(resolved)
		}
		created++

		if resolvedMod := ctx.ModuleToResolved[ref.mod]; resolvedMod != nil {
			resolvedMod.AddObject(resolved)
		}
	}

	// Second pass: resolve INDEX and AUGMENTS references now that all objects exist
	for _, ref := range objRefs {
		obj := ref.obj

		// Get the Object from the module's collection, not from the shared node.
		// Multiple modules can define objects at the same OID (e.g., IF-MIB and
		// RFC1213-MIB both define ifEntry). Each module has its own Object instance.
		resolvedMod := ctx.ModuleToResolved[ref.mod]
		if resolvedMod == nil {
			continue
		}
		resolvedObj := resolvedMod.InternalObject(obj.Name)
		if resolvedObj == nil {
			continue
		}

		// Resolve INDEX
		if len(obj.Index) > 0 {
			var indexEntries []mib.IndexEntry
			for _, item := range obj.Index {
				if indexNode, ok := ctx.LookupNodeForModule(ref.mod, item.Object); ok {
					if indexNode.InternalObject() != nil {
						indexEntries = append(indexEntries, mib.IndexEntry{
							Object:  indexNode.InternalObject(),
							Implied: item.Implied,
						})
					}
				}
			}
			resolvedObj.SetIndex(indexEntries)
		}

		// Resolve AUGMENTS
		if obj.Augments != "" {
			if augNode, ok := ctx.LookupNodeForModule(ref.mod, obj.Augments); ok {
				if augNode.InternalObject() != nil {
					resolvedObj.SetAugments(augNode.InternalObject())
				}
			}
		}
	}

	if ctx.TraceEnabled() {
		ctx.Trace("created resolved objects", slog.Int("count", created))
	}
}

func computeEffectiveValues(obj *mibimpl.Object) {
	t := obj.InternalType()
	if t == nil {
		return
	}

	// Walk the type chain to find effective values
	for t != nil {
		if obj.EffectiveDisplayHint() == "" && t.DisplayHint() != "" {
			obj.SetEffectiveHint(t.DisplayHint())
		}
		if len(obj.EffectiveSizes()) == 0 && len(t.Sizes()) > 0 {
			obj.SetEffectiveSizes(t.Sizes())
		}
		if len(obj.EffectiveRanges()) == 0 && len(t.Ranges()) > 0 {
			obj.SetEffectiveRanges(t.Ranges())
		}
		if len(obj.EffectiveEnums()) == 0 && len(t.Enums()) > 0 {
			obj.SetEffectiveEnums(t.Enums())
		}
		if len(obj.EffectiveBits()) == 0 && len(t.Bits()) > 0 {
			obj.SetEffectiveBits(t.Bits())
		}
		t = t.InternalParent()
	}
}

func createResolvedNotifications(ctx *ResolverContext) {
	created := 0
	for _, ref := range collectNotificationRefs(ctx) {
		notif := ref.notif

		node, ok := ctx.LookupNodeForModule(ref.mod, notif.Name)
		if !ok {
			continue
		}

		resolved := mibimpl.NewNotification(notif.Name)
		resolved.SetNode(node)
		resolved.SetModule(ctx.ModuleToResolved[ref.mod])
		resolved.SetStatus(convertStatus(notif.Status))
		resolved.SetDescription(notif.Description)
		resolved.SetReference(notif.Reference)

		for _, objName := range notif.Objects {
			if objNode, ok := ctx.LookupNodeForModule(ref.mod, objName); ok {
				if objNode.InternalObject() != nil {
					resolved.AddObject(objNode.InternalObject())
				}
			} else {
				ctx.RecordUnresolvedNotificationObject(ref.mod, notif.Name, objName, notif.Span)
			}
		}

		ctx.Builder.AddNotification(resolved)
		node.SetNotification(resolved)
		created++

		if resolvedMod := ctx.ModuleToResolved[ref.mod]; resolvedMod != nil {
			resolvedMod.AddNotification(resolved)
		}
	}

	if ctx.TraceEnabled() {
		ctx.Trace("created resolved notifications", slog.Int("count", created))
	}
}

func resolveTypeSyntax(ctx *ResolverContext, syntax module.TypeSyntax, mod *module.Module, objectName string, span types.Span) (*mibimpl.Type, bool) {
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
		if t, ok := ctx.LookupType("Integer32"); ok {
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

func convertDefVal(ctx *ResolverContext, defval module.DefVal, mod *module.Module) mib.DefVal {
	switch v := defval.(type) {
	case *module.DefValInteger:
		return mib.DefValInt(v.Value)
	case *module.DefValUnsigned:
		return mib.DefValUnsigned(v.Value)
	case *module.DefValString:
		return mib.DefValString(v.Value)
	case *module.DefValHexString:
		return mib.DefValHexString(v.Value)
	case *module.DefValBinaryString:
		return mib.DefValBinaryString(v.Value)
	case *module.DefValEnum:
		return mib.DefValEnum(v.Name)
	case *module.DefValBits:
		return mib.DefValBits(slices.Clone(v.Labels))
	case *module.DefValOidRef:
		if node, ok := ctx.LookupNodeForModule(mod, v.Name); ok {
			return mib.DefValOID(node.OID())
		}
		return nil
	case *module.DefValOidValue:
		if len(v.Components) > 0 {
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
			if name != "" {
				if node, ok := ctx.LookupNodeForModule(mod, name); ok {
					return mib.DefValOID(node.OID())
				}
			}
		}
		return nil
	default:
		return nil
	}
}

func convertAccess(a types.Access) mib.Access {
	switch a {
	case types.AccessNotAccessible:
		return mib.AccessNotAccessible
	case types.AccessAccessibleForNotify:
		return mib.AccessAccessibleForNotify
	case types.AccessReadOnly:
		return mib.AccessReadOnly
	case types.AccessReadWrite:
		return mib.AccessReadWrite
	case types.AccessReadCreate:
		return mib.AccessReadCreate
	case types.AccessWriteOnly:
		return mib.AccessWriteOnly
	default:
		return mib.AccessNotAccessible
	}
}

func isBareTypeIndex(name string) bool {
	switch name {
	case "INTEGER", "Integer32", "Unsigned32", "Counter32", "Counter64", "Gauge32",
		"IpAddress", "Opaque", "TimeTicks", "BITS", "OCTET", "STRING", "Counter", "Gauge", "NetworkAddress":
		return true
	default:
		return false
	}
}
