package resolver

import (
	"log/slog"
	"slices"
	"strconv"
	"strings"

	"github.com/golangsnmp/gomib/internal/mibimpl"
	"github.com/golangsnmp/gomib/internal/module"
	"github.com/golangsnmp/gomib/internal/types"
	"github.com/golangsnmp/gomib/mib"
)

// analyzeSemantics is the semantic analysis phase entry point.
func analyzeSemantics(ctx *ResolverContext) {
	objRefs := collectObjectTypeRefs(ctx)
	inferNodeKinds(ctx, objRefs)
	resolveTableSemantics(ctx, objRefs)
	createResolvedObjects(ctx, objRefs)
	createResolvedNotifications(ctx)
	createResolvedGroups(ctx)
	createResolvedCompliances(ctx)
	createResolvedCapabilities(ctx)
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

	// Reclassify scalar children of row nodes as columns
	columns := 0
	for _, row := range rowNodes {
		for _, child := range row.Children() {
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

// collectDefinitionRefs collects definitions matching matchFn across all modules.
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

		if t, ok := resolveTypeSyntax(ctx, obj.Syntax, ref.mod, obj.Name, obj.Span); ok {
			resolved.SetType(t)
		}

		sizes, ranges := extractConstraints(obj.Syntax)
		resolved.SetEffectiveSizes(sizes)
		resolved.SetEffectiveRanges(ranges)
		if _, isBits := obj.Syntax.(*module.TypeSyntaxBits); isBits {
			resolved.SetEffectiveBits(extractNamedValues(obj.Syntax))
		} else {
			resolved.SetEffectiveEnums(extractNamedValues(obj.Syntax))
		}

		// INDEX and AUGMENTS are resolved in a second pass after all
		// objects exist so that cross-references can be linked.

		if obj.DefVal != nil {
			resolved.SetDefaultValue(convertDefVal(ctx, obj.DefVal, ref.mod, obj.Syntax))
		}

		computeEffectiveValues(resolved)

		ctx.Builder.AddObject(resolved)

		// Prefer SMIv2 modules when multiple modules define the same OID
		// (e.g., IF-MIB and RFC1213-MIB both define ifEntry).
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

	// Second pass: link INDEX and AUGMENTS now that all objects exist.
	// Use the module's own Object instance, not the shared node's, because
	// multiple modules can define objects at the same OID.
	for _, ref := range objRefs {
		obj := ref.obj

		resolvedMod := ctx.ModuleToResolved[ref.mod]
		if resolvedMod == nil {
			continue
		}
		resolvedObj := resolvedMod.InternalObject(obj.Name)
		if resolvedObj == nil {
			continue
		}

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

	// Inherit display hint, constraints, and enums from ancestor types
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
			var objNode *mibimpl.Node
			var ok bool

			objNode, ok = ctx.LookupNodeForModule(ref.mod, objName)

			// Permissive only: global lookup for objects not explicitly imported
			if !ok && ctx.DiagnosticConfig().AllowBestGuessFallbacks() {
				objNode, ok = ctx.LookupNodeGlobal(objName)
			}

			if ok && objNode.InternalObject() != nil {
				resolved.AddObject(objNode.InternalObject())
			} else if !ok {
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

type objectGroupRef struct {
	mod *module.Module
	grp *module.ObjectGroup
}

type notificationGroupRef struct {
	mod *module.Module
	grp *module.NotificationGroup
}

func createResolvedGroups(ctx *ResolverContext) {
	created := 0

	// OBJECT-GROUP definitions
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

		resolved := mibimpl.NewGroup(grp.Name)
		resolved.SetNode(node)
		resolved.SetModule(ctx.ModuleToResolved[ref.mod])
		resolved.SetStatus(convertStatus(grp.Status))
		resolved.SetDescription(grp.Description)
		resolved.SetReference(grp.Reference)

		for _, memberName := range grp.Objects {
			if memberNode, ok := lookupMemberNode(ctx, ref.mod, memberName); ok {
				resolved.AddMember(memberNode)
				if obj := memberNode.InternalObject(); obj != nil && obj.Access() == mib.AccessNotAccessible {
					ctx.EmitDiagnostic("group-not-accessible", mib.SeverityMinor,
						ref.mod.Name, 0, 0,
						"object "+memberName+" of group "+grp.Name+" must not be not-accessible")
				}
			}
		}

		ctx.Builder.AddGroup(resolved)
		node.SetGroup(resolved)
		created++

		if resolvedMod := ctx.ModuleToResolved[ref.mod]; resolvedMod != nil {
			resolvedMod.AddGroup(resolved)
		}
	}

	// NOTIFICATION-GROUP definitions
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

		resolved := mibimpl.NewGroup(grp.Name)
		resolved.SetNode(node)
		resolved.SetModule(ctx.ModuleToResolved[ref.mod])
		resolved.SetStatus(convertStatus(grp.Status))
		resolved.SetDescription(grp.Description)
		resolved.SetReference(grp.Reference)
		resolved.SetIsNotificationGroup(true)

		for _, memberName := range grp.Notifications {
			if memberNode, ok := lookupMemberNode(ctx, ref.mod, memberName); ok {
				resolved.AddMember(memberNode)
			}
		}

		ctx.Builder.AddGroup(resolved)
		node.SetGroup(resolved)
		created++

		if resolvedMod := ctx.ModuleToResolved[ref.mod]; resolvedMod != nil {
			resolvedMod.AddGroup(resolved)
		}
	}

	if ctx.TraceEnabled() {
		ctx.Trace("created resolved groups", slog.Int("count", created))
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

func createResolvedCompliances(ctx *ResolverContext) {
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

		resolved := mibimpl.NewCompliance(comp.Name)
		resolved.SetNode(node)
		resolved.SetModule(ctx.ModuleToResolved[ref.mod])
		resolved.SetStatus(convertStatus(comp.Status))
		resolved.SetDescription(comp.Description)
		resolved.SetReference(comp.Reference)
		resolved.SetModules(convertComplianceModules(comp.Modules))

		ctx.Builder.AddCompliance(resolved)
		node.SetCompliance(resolved)
		created++

		if resolvedMod := ctx.ModuleToResolved[ref.mod]; resolvedMod != nil {
			resolvedMod.AddCompliance(resolved)
		}
	}

	if ctx.TraceEnabled() {
		ctx.Trace("created resolved compliances", slog.Int("count", created))
	}
}

func convertComplianceModules(modules []module.ComplianceModule) []mib.ComplianceModule {
	result := make([]mib.ComplianceModule, len(modules))
	for i, m := range modules {
		result[i] = mib.ComplianceModule{
			ModuleName:      m.ModuleName,
			MandatoryGroups: m.MandatoryGroups,
		}
		if len(m.Groups) > 0 {
			groups := make([]mib.ComplianceGroup, len(m.Groups))
			for j, g := range m.Groups {
				groups[j] = mib.ComplianceGroup{
					Group:       g.Group,
					Description: g.Description,
				}
			}
			result[i].Groups = groups
		}
		if len(m.Objects) > 0 {
			objects := make([]mib.ComplianceObject, len(m.Objects))
			for j, o := range m.Objects {
				objects[j] = mib.ComplianceObject{
					Object:      o.Object,
					Description: o.Description,
				}
				if o.MinAccess != nil {
					a := convertAccess(*o.MinAccess)
					objects[j].MinAccess = &a
				}
			}
			result[i].Objects = objects
		}
	}
	return result
}

func createResolvedCapabilities(ctx *ResolverContext) {
	created := 0
	for _, ref := range collectDefinitionRefs(ctx, func(mod *module.Module, def module.Definition) (capabilitiesRef, bool) {
		if cap, ok := def.(*module.AgentCapabilities); ok {
			return capabilitiesRef{mod: mod, cap: cap}, true
		}
		return capabilitiesRef{}, false
	}) {
		cap := ref.cap
		node, ok := ctx.LookupNodeForModule(ref.mod, cap.Name)
		if !ok {
			continue
		}

		resolved := mibimpl.NewCapabilities(cap.Name)
		resolved.SetNode(node)
		resolved.SetModule(ctx.ModuleToResolved[ref.mod])
		resolved.SetStatus(convertStatus(cap.Status))
		resolved.SetDescription(cap.Description)
		resolved.SetReference(cap.Reference)
		resolved.SetProductRelease(cap.ProductRelease)
		resolved.SetSupports(convertSupportsModules(cap.Supports))

		ctx.Builder.AddCapabilities(resolved)
		node.SetCapabilities(resolved)
		created++

		if resolvedMod := ctx.ModuleToResolved[ref.mod]; resolvedMod != nil {
			resolvedMod.AddCapabilities(resolved)
		}
	}

	if ctx.TraceEnabled() {
		ctx.Trace("created resolved capabilities", slog.Int("count", created))
	}
}

func convertSupportsModules(modules []module.SupportsModule) []mib.CapabilitiesModule {
	result := make([]mib.CapabilitiesModule, len(modules))
	for i, m := range modules {
		result[i] = mib.CapabilitiesModule{
			ModuleName: m.ModuleName,
			Includes:   m.Includes,
		}
		if len(m.ObjectVariations) > 0 {
			vars := make([]mib.ObjectVariation, len(m.ObjectVariations))
			for j, v := range m.ObjectVariations {
				vars[j] = mib.ObjectVariation{
					Object:      v.Object,
					Description: v.Description,
				}
				if v.Access != nil {
					a := convertAccess(*v.Access)
					vars[j].Access = &a
				}
			}
			result[i].ObjectVariations = vars
		}
		if len(m.NotificationVariations) > 0 {
			vars := make([]mib.NotificationVariation, len(m.NotificationVariations))
			for j, v := range m.NotificationVariations {
				vars[j] = mib.NotificationVariation{
					Notification: v.Notification,
					Description:  v.Description,
				}
				if v.Access != nil {
					a := convertAccess(*v.Access)
					vars[j].Access = &a
				}
			}
			result[i].NotificationVariations = vars
		}
	}
	return result
}

func lookupMemberNode(ctx *ResolverContext, mod *module.Module, name string) (*mibimpl.Node, bool) {
	node, ok := ctx.LookupNodeForModule(mod, name)
	if ok {
		return node, true
	}
	if ctx.DiagnosticConfig().AllowBestGuessFallbacks() {
		return ctx.LookupNodeGlobal(name)
	}
	return nil, false
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

func convertDefVal(ctx *ResolverContext, defval module.DefVal, mod *module.Module, syntax module.TypeSyntax) *mib.DefVal {
	switch v := defval.(type) {
	case *module.DefValInteger:
		raw := strconv.FormatInt(v.Value, 10)
		dv := mib.NewDefValInt(v.Value, raw)
		return &dv
	case *module.DefValUnsigned:
		raw := strconv.FormatUint(v.Value, 10)
		dv := mib.NewDefValUint(v.Value, raw)
		return &dv
	case *module.DefValString:
		raw := `"` + v.Value + `"`
		dv := mib.NewDefValString(v.Value, raw)
		return &dv
	case *module.DefValHexString:
		raw := "'" + v.Value + "'H"
		bytes := hexToBytes(v.Value)
		dv := mib.NewDefValBytes(bytes, raw)
		return &dv
	case *module.DefValBinaryString:
		raw := "'" + v.Value + "'B"
		bytes := binaryToBytes(v.Value)
		dv := mib.NewDefValBytes(bytes, raw)
		return &dv
	case *module.DefValEnum:
		// Parser emits bare names as DefValEnum, but for OID-typed objects
		// the name is actually an OID reference.
		if isOIDType(syntax) {
			if node, ok := ctx.LookupNodeForModule(mod, v.Name); ok {
				oid := mib.Oid(node.OID())
				dv := mib.NewDefValOID(oid, v.Name)
				return &dv
			}
		}
		dv := mib.NewDefValEnum(v.Name, v.Name)
		return &dv
	case *module.DefValBits:
		raw := "{ " + strings.Join(v.Labels, ", ") + " }"
		if len(v.Labels) == 0 {
			raw = "{ }"
		}
		dv := mib.NewDefValBits(slices.Clone(v.Labels), raw)
		return &dv
	case *module.DefValOidRef:
		if node, ok := ctx.LookupNodeForModule(mod, v.Name); ok {
			oid := mib.Oid(node.OID())
			dv := mib.NewDefValOID(oid, v.Name)
			return &dv
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
					oid := mib.Oid(node.OID())
					dv := mib.NewDefValOID(oid, name)
					return &dv
				}
			}
		}
		return nil
	default:
		return nil
	}
}

// hexToBytes converts a hex string (e.g., "00FF1A") to bytes.
func hexToBytes(s string) []byte {
	if len(s) == 0 {
		return []byte{}
	}
	// Handle odd-length hex strings by padding
	if len(s)%2 != 0 {
		s = "0" + s
	}
	result := make([]byte, len(s)/2)
	for i := 0; i < len(s); i += 2 {
		var b byte
		for j := 0; j < 2; j++ {
			c := s[i+j]
			b <<= 4
			switch {
			case c >= '0' && c <= '9':
				b |= c - '0'
			case c >= 'A' && c <= 'F':
				b |= c - 'A' + 10
			case c >= 'a' && c <= 'f':
				b |= c - 'a' + 10
			}
		}
		result[i/2] = b
	}
	return result
}

// binaryToBytes converts a binary string (e.g., "10101010") to bytes.
func binaryToBytes(s string) []byte {
	if len(s) == 0 {
		return []byte{}
	}
	// Pad to multiple of 8
	padding := (8 - len(s)%8) % 8
	for i := 0; i < padding; i++ {
		s = "0" + s
	}
	result := make([]byte, len(s)/8)
	for i := 0; i < len(s); i += 8 {
		var b byte
		for j := 0; j < 8; j++ {
			b <<= 1
			if s[i+j] == '1' {
				b |= 1
			}
		}
		result[i/8] = b
	}
	return result
}

func convertAccess(a module.Access) mib.Access {
	switch a {
	case module.AccessNotAccessible:
		return mib.AccessNotAccessible
	case module.AccessAccessibleForNotify:
		return mib.AccessAccessibleForNotify
	case module.AccessReadOnly:
		return mib.AccessReadOnly
	case module.AccessReadWrite:
		return mib.AccessReadWrite
	case module.AccessReadCreate:
		return mib.AccessReadCreate
	case module.AccessWriteOnly:
		return mib.AccessWriteOnly
	default:
		return mib.AccessNotAccessible
	}
}

func isBareTypeIndex(name string) bool {
	switch name {
	case "INTEGER", "Integer32", "Unsigned32", "Counter32", "Counter64", "Gauge32",
		"IpAddress", "Opaque", "TimeTicks", "BITS", "OCTET STRING", "Counter", "Gauge", "NetworkAddress":
		return true
	default:
		return false
	}
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
