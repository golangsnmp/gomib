package resolver

// Conformance and group-related checks for the resolver semantic analysis phase.
//
// These checks validate compliance module structure, group membership
// completeness, and well-known TC usage patterns (RowStatus, StorageType,
// TAddress/TDomain, InetAddress, TransportAddress).

import (
	"fmt"

	"github.com/golangsnmp/gomib/internal/model"
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
		groupedNodes         map[*model.Node]bool
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
				info = &moduleGroupInfo{groupedNodes: make(map[*model.Node]bool)}
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
		node, ok := ctx.lookupNode(ref.mod, ref.obj.Name)
		if !ok {
			continue
		}
		kind := node.Kind()
		if kind != model.KindScalar && kind != model.KindColumn {
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
		node, ok := ctx.lookupNode(ref.mod, ref.notif.Name)
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
func lookupComplianceMember(ctx *resolverContext, compMod *module.Module, moduleName, name string) *model.Node {
	if moduleName != "" {
		node, ok := ctx.lookupNodeByModuleName(moduleName, name)
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
		for _, d := range mod.ObjectGroups {
			checkMemberLocality(ctx, mod, d.DefinitionSpan(), d.Objects)
		}
		for _, d := range mod.NotificationGroups {
			checkMemberLocality(ctx, mod, d.DefinitionSpan(), d.Notifications)
		}
	}
}

func checkMemberLocality(ctx *resolverContext, mod *module.Module, span types.Span, members []string) {
	localSymbols := ctx.nodeSymbols.forModule(mod)
	for _, memberName := range members {
		if localSymbols == nil || localSymbols[memberName] == nil {
			ctx.EmitDiagnostic(types.DiagComplianceMemberNotLocal,
				mod, span,
				fmt.Sprintf("group member %q is not defined in module %q", memberName, mod.Name))
		}
	}
}

// checkGroupUnreferenced flags OBJECT-GROUP and NOTIFICATION-GROUP definitions
// that are never referenced in any MODULE-COMPLIANCE (as mandatory or optional
// groups). Only checks within the loaded module set.
func checkGroupUnreferenced(ctx *resolverContext) {
	// Collect all group names referenced by compliance and capabilities modules.
	referencedGroups := make(map[string]struct{})
	for _, mod := range ctx.modules {
		for _, d := range mod.Compliances {
			for _, cm := range d.Modules {
				for _, name := range cm.MandatoryGroups {
					referencedGroups[name] = struct{}{}
				}
				for _, grp := range cm.Groups {
					referencedGroups[grp.Group] = struct{}{}
				}
			}
		}
		for _, d := range mod.Capabilities {
			for _, sup := range d.Supports {
				for _, name := range sup.Includes {
					referencedGroups[name] = struct{}{}
				}
			}
		}
	}

	// Check each group definition.
	for _, mod := range ctx.modules {
		if module.IsBaseModule(mod.Name) {
			continue
		}
		for _, d := range mod.ObjectGroups {
			if _, ok := referencedGroups[d.Name]; !ok {
				ctx.EmitDiagnostic(types.DiagGroupUnreferenced,
					mod, d.Span,
					fmt.Sprintf("%q: OBJECT-GROUP not referenced in any compliance module", d.Name))
			}
		}
		for _, d := range mod.NotificationGroups {
			if _, ok := referencedGroups[d.Name]; !ok {
				ctx.EmitDiagnostic(types.DiagGroupUnreferenced,
					mod, d.Span,
					fmt.Sprintf("%q: NOTIFICATION-GROUP not referenced in any compliance module", d.Name))
			}
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
		resolved := ctx.lookupObject(ref.mod, ref.obj.Name)
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
		resolved := ctx.lookupObject(ref.mod, ref.obj.Name)
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

// defvalAsInt64 extracts a numeric value from a model.DefVal, mapping enum labels
// via the provided name map. Returns the value and true if extraction
// succeeded, false otherwise.
func defvalAsInt64(dv model.DefVal, labelToValue map[int64]string) (int64, bool) {
	switch dv.Kind() {
	case model.DefValKindInt:
		v, _ := model.DefValAs[int64](dv)
		return v, true
	case model.DefValKindUint:
		uv, _ := model.DefValAs[uint64](dv)
		return int64(uv), true
	case model.DefValKindEnum:
		label, _ := model.DefValAs[string](dv)
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
		resolved := ctx.lookupObject(ref.mod, ref.obj.Name)
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

// isDerivedFromType walks the parent chain of t and returns true if any
// type in the chain is target.
const maxTypeChainDepth = 1000

func isDerivedFromType(t, target *model.Type) bool {
	for current, depth := t, 0; current != nil && depth < maxTypeChainDepth; current, depth = current.Parent(), depth+1 {
		if current == target {
			return true
		}
	}
	return false
}

// lookupWellKnownTC looks up a textual convention from SNMPv2-TC by name.
func lookupWellKnownTC(ctx *resolverContext, name string) *model.Type {
	if ctx.snmpv2TCModule == nil {
		return nil
	}
	t, _ := ctx.lookupTypeDirect(ctx.snmpv2TCModule, name)
	return t
}

// lookupTypeInNamedModule looks up a type from a module identified by name.
// Returns nil if the module is not loaded or the type is not found.
func lookupTypeInNamedModule(ctx *resolverContext, moduleName, typeName string) *model.Type {
	mods := ctx.moduleIndex[moduleName]
	if len(mods) == 0 {
		return nil
	}
	t, _ := ctx.lookupTypeDirect(mods[0], typeName)
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
	var specificTypes []*model.Type
	for _, name := range cfg.specificTypes {
		if t := lookupTypeInNamedModule(ctx, cfg.moduleName, name); t != nil {
			specificTypes = append(specificTypes, t)
		}
	}

	for _, ref := range objRefs {
		resolved := ctx.lookupObject(ref.mod, ref.obj.Name)
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
func checkAddressPairingSibling(ctx *resolverContext, ref objectTypeRef, resolved *model.Object, addrTypeType *model.Type, diagCode, addrName, addrTypeName string) {
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
func checkAddressTypeSubtyped(ctx *resolverContext, ref objectTypeRef, resolved *model.Object, addrType *model.Type, objRefs []objectTypeRef, diagCode, addrName, addrTypeName string) {
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
