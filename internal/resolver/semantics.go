package resolver

import (
	"github.com/golangsnmp/gomib/mib"
	"log/slog"
	"slices"

	"github.com/golangsnmp/gomib/internal/module"
	"github.com/golangsnmp/gomib/internal/types"
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
	var rowNodes []*mib.Node

	for _, ref := range objRefs {
		obj := ref.obj
		node, ok := ctx.LookupNodeForModule(ref.mod, obj.Name)
		if !ok {
			continue
		}

		if obj.Syntax.IsSequenceOf() {
			node.Kind = mib.KindTable
			tables++
		} else if len(obj.Index) > 0 || obj.Augments != "" {
			node.Kind = mib.KindRow
			rows++
			rowNodes = append(rowNodes, node)
		} else {
			node.Kind = mib.KindScalar
			scalars++
		}
	}

	// Collect row children inline instead of separate tree walk
	columns := 0
	for _, row := range rowNodes {
		for _, child := range row.Children() {
			if child.Kind == mib.KindScalar {
				child.Kind = mib.KindColumn
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

		resolved := &mib.Object{
			Name:        obj.Name,
			Node:        node,
			Module:      ctx.ModuleToResolved[ref.mod],
			Access:      convertAccess(obj.Access),
			Status:      convertStatus(obj.Status),
			Description: obj.Description,
			Units:       obj.Units,
			Reference:   obj.Reference,
		}

		// Resolve type and extract inline constraints
		if t, ok := resolveTypeSyntax(ctx, obj.Syntax, ref.mod, obj.Name, obj.Span); ok {
			resolved.Type = t
		}

		// Extract inline constraints and named values
		resolved.Size, resolved.ValueRange = extractConstraints(obj.Syntax)
		resolved.NamedValues = extractNamedValues(obj.Syntax)

		// INDEX and AUGMENTS are resolved in a second pass after all objects exist
		// This ensures index objects exist when we try to link them

		// Convert DEFVAL
		if obj.DefVal != nil {
			resolved.DefVal = convertDefVal(ctx, obj.DefVal, ref.mod)
		}

		// Pre-compute effective values from type chain
		computeEffectiveValues(resolved)

		ctx.Builder.AddObject(resolved)
		node.Object = resolved
		created++

		if resolvedMod := ctx.ModuleToResolved[ref.mod]; resolvedMod != nil {
			resolvedMod.AddObject(resolved)
		}
	}

	// Second pass: resolve INDEX and AUGMENTS references now that all objects exist
	for _, ref := range objRefs {
		obj := ref.obj
		node, ok := ctx.LookupNodeForModule(ref.mod, obj.Name)
		if !ok || node.Object == nil {
			continue
		}

		// Resolve INDEX
		if len(obj.Index) > 0 {
			for _, item := range obj.Index {
				if indexNode, ok := ctx.LookupNodeForModule(ref.mod, item.Object); ok {
					if indexNode.Object != nil {
						node.Object.Index = append(node.Object.Index, mib.IndexEntry{
							Object:  indexNode.Object,
							Implied: item.Implied,
						})
					}
				}
			}
		}

		// Resolve AUGMENTS
		if obj.Augments != "" {
			if augNode, ok := ctx.LookupNodeForModule(ref.mod, obj.Augments); ok {
				if augNode.Object != nil {
					node.Object.Augments = augNode.Object
				}
			}
		}
	}

	if ctx.TraceEnabled() {
		ctx.Trace("created resolved objects", slog.Int("count", created))
	}
}

func computeEffectiveValues(obj *mib.Object) {
	if obj.Type == nil {
		return
	}

	// Walk the type chain to find effective values
	t := obj.Type
	for t != nil {
		if obj.Hint == "" && t.Hint != "" {
			obj.Hint = t.Hint
		}
		if len(obj.Size) == 0 && len(t.Size) > 0 {
			obj.Size = t.Size
		}
		if len(obj.ValueRange) == 0 && len(t.ValueRange) > 0 {
			obj.ValueRange = t.ValueRange
		}
		if len(obj.NamedValues) == 0 && len(t.NamedValues) > 0 {
			obj.NamedValues = t.NamedValues
		}
		t = t.Parent
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

		resolved := &mib.Notification{
			Name:        notif.Name,
			Node:        node,
			Module:      ctx.ModuleToResolved[ref.mod],
			Status:      convertStatus(notif.Status),
			Description: notif.Description,
			Reference:   notif.Reference,
		}

		for _, objName := range notif.Objects {
			if objNode, ok := ctx.LookupNodeForModule(ref.mod, objName); ok {
				if objNode.Object != nil {
					resolved.Objects = append(resolved.Objects, objNode.Object)
				}
			} else {
				ctx.RecordUnresolvedNotificationObject(ref.mod, notif.Name, objName, notif.Span)
			}
		}

		ctx.Builder.AddNotification(resolved)
		node.Notif = resolved
		created++

		if resolvedMod := ctx.ModuleToResolved[ref.mod]; resolvedMod != nil {
			resolvedMod.AddNotification(resolved)
		}
	}

	if ctx.TraceEnabled() {
		ctx.Trace("created resolved notifications", slog.Int("count", created))
	}
}

func resolveTypeSyntax(ctx *ResolverContext, syntax module.TypeSyntax, mod *module.Module, objectName string, span types.Span) (*mib.Type, bool) {
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
			if name, ok := v.Components[0].Name(); ok {
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
