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

// analyzeSemantics is the semantic analysis phase entry point.
func analyzeSemantics(ctx *resolverContext) {
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
		resolved.setModule(ctx.ModuleToResolved[ref.mod])
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
		newMod := ctx.ModuleToResolved[ref.mod]
		if shouldPreferModule(ctx, newMod, currentMod, ref.mod) {
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
					}
				}
			}
			resolvedObj.setIndex(indexEntries)
		}

		if obj.Augments != "" {
			if augNode, ok := ctx.LookupNodeForModule(ref.mod, obj.Augments); ok {
				if augNode.Object() != nil {
					resolvedObj.setAugments(augNode.Object())
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

	for t != nil {
		if obj.EffectiveDisplayHint() == "" && t.DisplayHint() != "" {
			obj.setEffectiveHint(t.DisplayHint())
		}
		if len(obj.EffectiveSizes()) == 0 && len(t.Sizes()) > 0 {
			obj.setEffectiveSizes(t.Sizes())
		}
		if len(obj.EffectiveRanges()) == 0 && len(t.Ranges()) > 0 {
			obj.setEffectiveRanges(t.Ranges())
		}
		if len(obj.EffectiveEnums()) == 0 && len(t.Enums()) > 0 {
			obj.setEffectiveEnums(t.Enums())
		}
		if len(obj.EffectiveBits()) == 0 && len(t.Bits()) > 0 {
			obj.setEffectiveBits(t.Bits())
		}
		t = t.Parent()
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
		resolved.setModule(ctx.ModuleToResolved[ref.mod])
		resolved.setStatus(notif.Status)
		resolved.setDescription(notif.Description)
		resolved.setReference(notif.Reference)

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

			if ok && objNode.Object() != nil {
				resolved.addObject(objNode.Object())
			} else if !ok {
				ctx.RecordUnresolvedNotificationObject(ref.mod, notif.Name, objName, notif.Span)
			} else if ok {
				// Node exists but has no object definition (intermediate node
				// or non-object definition).
				modName := ""
				if ref.mod != nil {
					modName = ref.mod.Name
				}
				ctx.EmitDiagnostic(types.DiagNotifObjectNotObject, SeverityMinor, modName, 0, 0,
					"notification "+notif.Name+" references "+objName+" which is not an object definition")
			}
		}

		ctx.Mib.addNotification(resolved)
		node.setNotification(resolved)
		created++

		if resolvedMod := ctx.ModuleToResolved[ref.mod]; resolvedMod != nil {
			resolvedMod.addNotification(resolved)
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
		resolved.setModule(ctx.ModuleToResolved[ref.mod])
		resolved.setStatus(grp.Status)
		resolved.setDescription(grp.Description)
		resolved.setReference(grp.Reference)

		for _, memberName := range grp.Objects {
			if memberNode, ok := lookupMemberNode(ctx, ref.mod, memberName); ok {
				resolved.addMember(memberNode)
				if obj := memberNode.Object(); obj != nil && obj.Access() == AccessNotAccessible {
					ctx.EmitDiagnostic(types.DiagGroupNotAccessible, SeverityMinor,
						ref.mod.Name, 0, 0,
						"object "+memberName+" of group "+grp.Name+" must not be not-accessible")
				}
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
		resolved.setModule(ctx.ModuleToResolved[ref.mod])
		resolved.setStatus(grp.Status)
		resolved.setDescription(grp.Description)
		resolved.setReference(grp.Reference)
		resolved.setIsNotificationGroup(true)

		for _, memberName := range grp.Notifications {
			if memberNode, ok := lookupMemberNode(ctx, ref.mod, memberName); ok {
				resolved.addMember(memberNode)
			}
		}

		registerGroup(ctx, ref.mod, node, resolved)
		created++
	}
	return created
}

func registerGroup(ctx *resolverContext, mod *module.Module, node *Node, resolved *Group) {
	ctx.Mib.addGroup(resolved)
	node.setGroup(resolved)
	if resolvedMod := ctx.ModuleToResolved[mod]; resolvedMod != nil {
		resolvedMod.addGroup(resolved)
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
		resolved.setModule(ctx.ModuleToResolved[ref.mod])
		resolved.setStatus(comp.Status)
		resolved.setDescription(comp.Description)
		resolved.setReference(comp.Reference)
		resolved.setModules(convertComplianceModules(comp.Modules))

		ctx.Mib.addCompliance(resolved)
		node.setCompliance(resolved)
		created++

		if resolvedMod := ctx.ModuleToResolved[ref.mod]; resolvedMod != nil {
			resolvedMod.addCompliance(resolved)
		}
	}

	if ctx.TraceEnabled() {
		ctx.Trace("created resolved compliances", slog.Int("count", created))
	}
}

func convertComplianceModules(modules []module.ComplianceModule) []ComplianceModule {
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

		resolved := newCapability(cap.Name)
		resolved.setNode(node)
		resolved.setModule(ctx.ModuleToResolved[ref.mod])
		resolved.setStatus(cap.Status)
		resolved.setDescription(cap.Description)
		resolved.setReference(cap.Reference)
		resolved.setProductRelease(cap.ProductRelease)
		resolved.setSupports(convertSupportsModules(cap.Supports))

		ctx.Mib.addCapability(resolved)
		node.setCapability(resolved)
		created++

		if resolvedMod := ctx.ModuleToResolved[ref.mod]; resolvedMod != nil {
			resolvedMod.addCapability(resolved)
		}
	}

	if ctx.TraceEnabled() {
		ctx.Trace("created resolved capabilities", slog.Int("count", created))
	}
}

func convertSupportsModules(modules []module.SupportsModule) []CapabilitiesModule {
	result := make([]CapabilitiesModule, len(modules))
	for i, m := range modules {
		result[i] = CapabilitiesModule{
			ModuleName: m.ModuleName,
			Includes:   m.Includes,
		}
		if len(m.ObjectVariations) > 0 {
			vars := make([]ObjectVariation, len(m.ObjectVariations))
			for j, v := range m.ObjectVariations {
				vars[j] = ObjectVariation{
					Object:      v.Object,
					Description: v.Description,
				}
				if v.Access != nil {
					vars[j].Access = v.Access
				}
			}
			result[i].ObjectVariations = vars
		}
		if len(m.NotificationVariations) > 0 {
			vars := make([]NotificationVariation, len(m.NotificationVariations))
			for j, v := range m.NotificationVariations {
				vars[j] = NotificationVariation{
					Notification: v.Notification,
					Description:  v.Description,
				}
				if v.Access != nil {
					vars[j].Access = v.Access
				}
			}
			result[i].NotificationVariations = vars
		}
	}
	return result
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
				mod.Name, 0, 0, "malformed hex DEFVAL "+raw+": "+err.Error())
			return nil
		}
		dv := newDefValBytes(bytes, raw)
		return &dv
	case *module.DefValBinaryString:
		raw := "'" + v.Value + "'B"
		bytes := binaryToBytes(v.Value)
		dv := newDefValBytes(bytes, raw)
		return &dv
	case *module.DefValEnum:
		// Parser emits bare names as DefValEnum, but for OID-typed objects
		// the name is actually an OID reference.
		if isOIDType(syntax) {
			if node, ok := ctx.LookupNodeForModule(mod, v.Name); ok {
				oid := OID(node.OID())
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
			oid := OID(node.OID())
			dv := newDefValOID(oid, v.Name)
			return &dv
		}
		ctx.EmitDiagnostic(types.DiagDefvalUnresolved, SeverityWarning,
			mod.Name, 0, 0, "DEFVAL OID reference "+v.Name+" could not be resolved")
		return nil
	case *module.DefValOidValue:
		if len(v.Components) == 0 {
			ctx.EmitDiagnostic(types.DiagDefvalUnresolved, SeverityWarning,
				mod.Name, 0, 0, "DEFVAL OID value has no components")
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
				mod.Name, 0, 0, "DEFVAL OID value has no named root component")
			return nil
		}
		node, ok := ctx.LookupNodeForModule(mod, name)
		if !ok {
			ctx.EmitDiagnostic(types.DiagDefvalUnresolved, SeverityWarning,
				mod.Name, 0, 0, "DEFVAL OID root "+name+" could not be resolved")
			return nil
		}
		oid := OID(node.OID())
		for _, comp := range v.Components[1:] {
			switch c := comp.(type) {
			case *module.OidComponentNumber:
				oid = append(oid, c.Value)
			case *module.OidComponentNamedNumber:
				oid = append(oid, c.NumberValue)
			case *module.OidComponentQualifiedNamedNumber:
				oid = append(oid, c.NumberValue)
			}
		}
		dv := newDefValOID(oid, name)
		return &dv
	default:
		return nil
	}
}

// hexToBytes converts a hex string (e.g., "00FF1A") to bytes.
func hexToBytes(s string) ([]byte, error) {
	if len(s) == 0 {
		return []byte{}, nil
	}
	// Handle odd-length hex strings by padding
	if len(s)%2 != 0 {
		s = "0" + s
	}
	return hex.DecodeString(s)
}

// binaryToBytes converts a binary string (e.g., "10101010") to bytes.
func binaryToBytes(s string) []byte {
	if len(s) == 0 {
		return []byte{}
	}
	// Pad to multiple of 8
	if padding := (8 - len(s)%8) % 8; padding > 0 {
		s = strings.Repeat("0", padding) + s
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
