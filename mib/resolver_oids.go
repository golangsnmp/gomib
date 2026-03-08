package mib

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/golangsnmp/gomib/internal/graph"
	"github.com/golangsnmp/gomib/internal/module"
	"github.com/golangsnmp/gomib/internal/types"
)

const (
	moduleSNMPv2SMI  = "SNMPv2-SMI"
	moduleRFC1155SMI = "RFC1155-SMI"
	moduleSNMPv2TC   = "SNMPv2-TC"
)

var smiGlobalOidRoots = map[string]struct{}{
	"internet":     {},
	"directory":    {},
	"mgmt":         {},
	"mib-2":        {},
	"transmission": {},
	"experimental": {},
	"private":      {},
	"enterprises":  {},
	"security":     {},
	"snmpV2":       {},
	"snmpDomains":  {},
	"snmpProxys":   {},
	"snmpModules":  {},
	"zeroDotZero":  {},
	"snmp":         {},
}

// resolveOids is the OID resolution phase entry point.
func resolveOids(ctx *resolverContext) {
	defs := collectOidDefinitions(ctx)
	checkSmiv2IdentifierHyphens(ctx, defs.oidDefs)

	nDefs := len(defs.oidDefs)
	g := graph.New(nDefs)
	defIndex := make(map[graph.Symbol]oidDefinition, nDefs)

	for _, def := range defs.oidDefs {
		sym := graph.Symbol{Module: def.mod.Name, Name: def.defName()}
		g.AddNode(sym)
		defIndex[sym] = def

		if parentSym, ok := getOidParentSymbol(ctx, def); ok {
			g.AddEdge(sym, parentSym)
		}
	}

	order, cycles := g.ResolutionOrder()
	logCycles(ctx, cycles, "OID cycle detected")

	if ctx.TraceEnabled() {
		ctx.Trace("OID resolution order",
			slog.Int("total", len(order)),
			slog.Int("cycles", len(cycles)))
	}

	resolved := 0
	for _, sym := range order {
		def, ok := defIndex[sym]
		if !ok {
			continue
		}
		if tryResolveOidDefinition(ctx, def) {
			resolved++
		}
	}

	for _, scc := range cycles {
		for _, sym := range scc {
			def, ok := defIndex[sym]
			if !ok {
				continue
			}
			oid := def.oid()
			if oid == nil || len(oid.Components) == 0 {
				continue
			}
			recordRecursiveOid(ctx, def, scc)
		}
	}

	if ctx.TraceEnabled() {
		ctx.Trace("OID resolution complete",
			slog.Int("resolved", resolved),
			slog.Int("cycles", len(cycles)))
	}

	resolveTrapTypeDefinitions(ctx, defs.trapDefs)
}

// getOidParentSymbol returns the symbol that the first component of the OID references.
func getOidParentSymbol(ctx *resolverContext, def oidDefinition) (graph.Symbol, bool) {
	oid := def.oid()
	if oid == nil || len(oid.Components) == 0 {
		return graph.Symbol{}, false
	}

	first := oid.Components[0]
	switch c := first.(type) {
	case *module.OidComponentName:
		return lookupNamedParentSymbol(ctx, def, c.NameValue)
	case *module.OidComponentNumber:
		return graph.Symbol{}, false // Numeric roots have no dependency.
	case *module.OidComponentNamedNumber:
		if sym, ok := lookupNamedParentSymbol(ctx, def, c.NameValue); ok {
			return sym, true
		}
		// Has a number, so can be resolved without the name.
		return graph.Symbol{}, false
	case *module.OidComponentQualifiedName:
		return graph.Symbol{Module: c.ModuleValue, Name: c.NameValue}, true
	case *module.OidComponentQualifiedNamedNumber:
		return graph.Symbol{Module: c.ModuleValue, Name: c.NameValue}, true
	}

	return graph.Symbol{}, false
}

// lookupNamedParentSymbol resolves a named OID parent by checking well-known roots,
// local/imported definitions, and (with constrained fallbacks) SMI global roots.
func lookupNamedParentSymbol(ctx *resolverContext, def oidDefinition, name string) (graph.Symbol, bool) {
	if wellKnownRootArc(name) >= 0 {
		return graph.Symbol{}, false
	}
	if parentMod := findOidDefiningModule(ctx, def.mod, name); parentMod != "" {
		return graph.Symbol{Module: parentMod, Name: name}, true
	}
	if ctx.ResolverStrictness().AllowConstrainedFallbacks() {
		if _, ok := smiGlobalOidRoots[name]; ok {
			if ctx.TraceEnabled() {
				ctx.Trace("permissive: using SMI global root for graph edge",
					slog.String("name", name),
					slog.String("module", def.mod.Name))
			}
			return graph.Symbol{Module: moduleSNMPv2SMI, Name: name}, true
		}
	}
	return graph.Symbol{}, false
}

// findOidDefiningModule finds the module that defines an OID symbol,
// checking local definitions first, then imports.
func findOidDefiningModule(ctx *resolverContext, fromMod *module.Module, name string) string {
	if oidDefs := ctx.moduleOidDefNames[fromMod]; oidDefs != nil {
		if _, ok := oidDefs[name]; ok {
			return fromMod.Name
		}
	}

	if imports := ctx.moduleImports[fromMod]; imports != nil {
		if srcMod := imports[name]; srcMod != nil {
			return srcMod.Name
		}
	}

	return ""
}

// recordRecursiveOid records an OID definition that participates in a dependency cycle.
func recordRecursiveOid(ctx *resolverContext, def oidDefinition, scc []graph.Symbol) {
	defName := def.defName()
	span := def.oid().Span

	others := make([]string, 0, len(scc)-1)
	for _, s := range scc {
		name := s.Module + "::" + s.Name
		if s.Module == def.mod.Name && s.Name == defName {
			continue
		}
		others = append(others, name)
	}

	var msg string
	if len(others) == 0 {
		msg = fmt.Sprintf("recursive OID: %s references itself", defName)
	} else {
		msg = fmt.Sprintf("recursive OID: %s is in a cycle with %s", defName, strings.Join(others, ", "))
	}

	recordUnresolved(ctx, &ctx.unresolvedOids, unresolvedOid{
		module: def.mod, definition: defName, span: span,
	}, def.mod, span, types.DiagOidRecursive, msg)
}

// checkSmiv2IdentifierHyphens emits a diagnostic for OID definition names
// containing hyphens in SMIv2 modules. smilint flags this at level 5.
func checkSmiv2IdentifierHyphens(ctx *resolverContext, defs []oidDefinition) {
	for _, def := range defs {
		if def.mod.Language != types.LanguageSMIv2 || module.IsBaseModule(def.mod.Name) {
			continue
		}
		name := def.defName()
		if strings.Contains(name, "-") {
			ctx.EmitDiagnostic(types.DiagIdentifierHyphenSMI,
				def.mod, def.def.DefinitionSpan(),
				fmt.Sprintf("identifier %q should not contain hyphens in SMIv2 MIB", name))
		}
	}
}

type oidDefinition struct {
	mod  *module.Module
	def  module.Definition
	kind definitionKind
}

type definitionKind int

const (
	defObjectType        definitionKind = iota // OBJECT-TYPE (RFC 2578)
	defModuleIdentity                          // MODULE-IDENTITY (RFC 2578)
	defObjectIdentity                          // OBJECT-IDENTITY (RFC 2578)
	defNotification                            // NOTIFICATION-TYPE or TRAP-TYPE with OID
	defValueAssignment                         // Plain OID value assignment (e.g., enterprises OBJECT IDENTIFIER ::= ...)
	defObjectGroup                             // OBJECT-GROUP (RFC 2580)
	defNotificationGroup                       // NOTIFICATION-GROUP (RFC 2580)
	defModuleCompliance                        // MODULE-COMPLIANCE (RFC 2580)
	defAgentCapabilities                       // AGENT-CAPABILITIES (RFC 2580)
)

func (d oidDefinition) defName() string {
	return d.def.DefinitionName()
}

func (d oidDefinition) oid() *module.OidAssignment {
	return d.def.DefinitionOid()
}

type trapTypeRef struct {
	mod   *module.Module
	notif *module.Notification
}

func (d trapTypeRef) defName() string {
	return d.notif.Name
}

func (d trapTypeRef) trapInfo() (enterprise string, trapNumber uint32, span types.Span, ok bool) {
	if d.notif.TrapInfo == nil {
		return "", 0, types.Span{}, false
	}
	return d.notif.TrapInfo.Enterprise, d.notif.TrapInfo.TrapNumber, d.notif.Span, true
}

type collectedOidDefinitions struct {
	oidDefs  []oidDefinition
	trapDefs []trapTypeRef
}

func collectOidDefinitions(ctx *resolverContext) collectedOidDefinitions {
	// Estimate capacity: most definitions are OID-bearing (TypeDefs are the exception).
	totalDefs := 0
	for _, mod := range ctx.modules {
		totalDefs += len(mod.Definitions)
	}
	defs := collectedOidDefinitions{
		oidDefs: make([]oidDefinition, 0, totalDefs),
	}

	for _, mod := range ctx.modules {
		for _, def := range mod.Definitions {
			var kind definitionKind
			switch d := def.(type) {
			case *module.ObjectType:
				kind = defObjectType
			case *module.ModuleIdentity:
				kind = defModuleIdentity
			case *module.ObjectIdentity:
				kind = defObjectIdentity
			case *module.Notification:
				switch {
				case d.Oid != nil:
					kind = defNotification
				case d.TrapInfo != nil:
					defs.trapDefs = append(defs.trapDefs, trapTypeRef{mod: mod, notif: d})
					continue
				default:
					ctx.EmitDiagnostic(types.DiagNotifNoOid,
						mod, d.Span,
						fmt.Sprintf("notification %q has no OID or trap info", d.Name))
					continue
				}
			case *module.ValueAssignment:
				kind = defValueAssignment
			case *module.ObjectGroup:
				kind = defObjectGroup
			case *module.NotificationGroup:
				kind = defNotificationGroup
			case *module.ModuleCompliance:
				kind = defModuleCompliance
			case *module.AgentCapabilities:
				kind = defAgentCapabilities
			case *module.TypeDef:
				continue
			default:
				continue
			}
			defs.oidDefs = append(defs.oidDefs, oidDefinition{mod: mod, def: def, kind: kind})
		}
	}

	return defs
}

func tryResolveOidDefinition(ctx *resolverContext, def oidDefinition) bool {
	oid := def.oid()
	if oid == nil {
		return false
	}
	components := oid.Components
	if len(components) == 0 {
		return false
	}
	var currentNode *Node
	for idx, component := range components {
		isLast := idx == len(components)-1
		node, ok := resolveOidComponent(ctx, def, currentNode, component, isLast)
		if !ok {
			return false
		}
		currentNode = node
	}

	if currentNode != nil {
		finalizeOidDefinition(ctx, def, currentNode, def.defName())
	}

	return true
}

func resolveOidComponent(ctx *resolverContext, def oidDefinition, currentNode *Node, component module.OidComponent, isLast bool) (*Node, bool) {
	switch c := component.(type) {
	case *module.OidComponentName:
		return resolveNameComponent(ctx, def, c.NameValue)
	case *module.OidComponentNumber:
		return resolveNumericComponent(ctx, currentNode, c.Value), true
	case *module.OidComponentNamedNumber:
		return resolveNamedNumberComponent(ctx, def, currentNode, c.NameValue, c.NumberValue, isLast)
	case *module.OidComponentQualifiedName:
		return resolveQualifiedNameComponent(ctx, def, c.ModuleValue, c.NameValue)
	case *module.OidComponentQualifiedNamedNumber:
		return resolveQualifiedNamedNumberComponent(ctx, def, currentNode, c.ModuleValue, c.NameValue, c.NumberValue, isLast)
	default:
		ctx.RecordUnresolvedOid(def.mod, def.defName(), "", def.oid().Span)
		return nil, false
	}
}

func resolveNameComponent(ctx *resolverContext, def oidDefinition, name string) (*Node, bool) {
	if node, ok := ctx.LookupNodeForModule(def.mod, name); ok {
		return node, true
	}
	// RFC-compliant: well-known roots (iso, ccitt, joint-iso-ccitt)
	if node, ok := lookupOrCreateWellKnownRoot(ctx, name); ok {
		return node, true
	}
	// Normal/Permissive: SMI global OID roots without explicit import
	if ctx.ResolverStrictness().AllowConstrainedFallbacks() {
		if node, ok := lookupSmiGlobalOidRoot(ctx, name); ok {
			if ctx.TraceEnabled() {
				ctx.Trace("permissive: resolved OID via SMI global root",
					slog.String("name", name),
					slog.String("module", def.mod.Name))
			}
			return node, true
		}
	}
	ctx.RecordUnresolvedOid(def.mod, def.defName(), name, def.oid().Span)
	return nil, false
}

func resolveNamedNumberComponent(ctx *resolverContext, def oidDefinition, currentNode *Node, name string, number uint32, isLast bool) (*Node, bool) {
	if node, ok := ctx.LookupNodeForModule(def.mod, name); ok {
		return node, true
	}
	return createNamedChild(ctx, def, currentNode, name, number, isLast)
}

func resolveQualifiedNameComponent(ctx *resolverContext, def oidDefinition, moduleName, name string) (*Node, bool) {
	if node, ok := ctx.LookupNodeInModule(moduleName, name); ok {
		return node, true
	}
	ctx.RecordUnresolvedOid(def.mod, def.defName(), moduleName+"."+name, def.oid().Span)
	return nil, false
}

func resolveQualifiedNamedNumberComponent(ctx *resolverContext, def oidDefinition, currentNode *Node, moduleName, name string, number uint32, isLast bool) (*Node, bool) {
	if node, ok := ctx.LookupNodeInModule(moduleName, name); ok {
		ctx.registerModuleNodeSymbol(def.mod, name, node)
		return node, true
	}
	return createNamedChild(ctx, def, currentNode, name, number, isLast)
}

// createNamedChild resolves a numeric component and registers it with a name.
// Shared by resolveNamedNumberComponent and resolveQualifiedNamedNumberComponent.
func createNamedChild(ctx *resolverContext, def oidDefinition, currentNode *Node, name string, number uint32, isLast bool) (*Node, bool) {
	child := resolveNumericComponent(ctx, currentNode, number)
	if child == nil {
		return nil, false
	}
	ctx.registerModuleNodeSymbol(def.mod, name, child)
	if !isLast {
		child.setName(name)
		child.setModule(ctx.moduleToResolved[def.mod])
		ctx.mib.registerNode(name, child)
		if child.Kind() == KindInternal {
			child.setKind(KindNode)
		}
	}
	return child, true
}

func finalizeOidDefinition(ctx *resolverContext, def oidDefinition, node *Node, label string) {
	// Check for OID reuse: another definition already claimed this node
	// with a different name. Same-name cross-module overlap (e.g.,
	// RFC1213-MIB and SNMPv2-MIB both defining sysDescr) is expected.
	if existing := node.Name(); existing != "" && existing != label {
		if isRegisteredKind(node.Kind()) {
			ctx.EmitDiagnostic(types.DiagOidRegistered,
				def.mod, def.def.DefinitionSpan(),
				fmt.Sprintf("%q: registers OID already registered by %q", label, existing))
		} else {
			ctx.EmitDiagnostic(types.DiagOidReuse,
				def.mod, def.def.DefinitionSpan(),
				fmt.Sprintf("%q: reuses OID assigned to %q", label, existing))
		}
	}

	// RFC 2578 section 7.10: the last sub-identifier of an administrative
	// OID assignment must not be zero. Exempt NODE kinds (MODULE-IDENTITY,
	// OBJECT-IDENTITY, value assignments) and zeroDotZero (0.0).
	if def.kind != defModuleIdentity && def.kind != defObjectIdentity && def.kind != defValueAssignment {
		oid := node.OID()
		if len(oid) > 0 && oid[len(oid)-1] == 0 {
			if len(oid) != 2 || oid[0] != 0 { // exempt zeroDotZero
				ctx.EmitDiagnostic(types.DiagLastSubidZero,
					def.mod, def.def.DefinitionSpan(),
					fmt.Sprintf("%q: last sub-identifier must not be zero", label))
			}
		}
	}

	// Prefer SMIv2 over SMIv1 when multiple modules define the same OID
	newMod := ctx.moduleToResolved[def.mod]
	currentMod := node.Module()
	if currentMod != nil && ctx.TraceEnabled() {
		ctx.Trace("node already has module",
			slog.String("node", label),
			slog.String("current", currentMod.Name()),
			slog.String("new", def.mod.Name))
	}
	// The node's kind, name, span, description, and module back-reference
	// all reflect the preferred module. Gate all of them behind the same
	// preference check so they stay in sync.
	if shouldPreferModule(ctx, currentMod, def.mod) {
		switch def.kind {
		case defObjectType:
			node.setKind(KindScalar)
		case defModuleIdentity, defObjectIdentity, defValueAssignment:
			node.setKind(KindNode)
		case defNotification:
			node.setKind(KindNotification)
		case defObjectGroup, defNotificationGroup:
			node.setKind(KindGroup)
		case defModuleCompliance:
			node.setKind(KindCompliance)
		case defAgentCapabilities:
			node.setKind(KindCapability)
		}
		node.setName(label)
		node.setSpan(def.def.DefinitionSpan())
		switch d := def.def.(type) {
		case *module.ValueAssignment:
			node.setDescription(d.Description)
			node.setReference(d.Reference)
		case *module.ObjectIdentity:
			node.setDescription(d.Description)
			node.setReference(d.Reference)
		}
		node.setModule(newMod)
		if def.kind == defModuleIdentity {
			newMod.setOID(node.OID())
		}
	}

	// Register non-semantic definitions in the defining module's node list
	// regardless of preference. Both modules that define the same OID should
	// list it in their Nodes(). Semantic definitions (object types, notifications,
	// etc.) are registered in the semantics phase instead. This must happen
	// after the preference block so the node has its name set for addNode's
	// name-keyed index.
	switch def.kind {
	case defValueAssignment, defObjectIdentity, defModuleIdentity:
		newMod.addNode(node)
	default:
		// Semantic definitions (object types, notifications, groups, etc.)
		// are registered in the semantics phase.
	}

	ctx.registerModuleNodeSymbol(def.mod, label, node)
	ctx.mib.registerNode(label, node)

	if ctx.TraceEnabled() {
		ctx.Trace("resolved OID definition",
			slog.String("name", label),
			slog.Uint64("arc", uint64(node.Arc())),
			slog.String("kind", node.Kind().String()))
	}
}

func resolveNumericComponent(ctx *resolverContext, parent *Node, arc uint32) *Node {
	if parent != nil {
		return parent.getOrCreateChild(arc)
	}
	return ctx.mib.Root().getOrCreateChild(arc)
}

func resolveTrapTypeDefinitions(ctx *resolverContext, defs []trapTypeRef) {
	for _, def := range defs {
		enterprise, trapNumber, span, ok := def.trapInfo()
		if !ok {
			continue
		}
		defName := def.defName()

		enterpriseNode, found := ctx.LookupNodeForModule(def.mod, enterprise)
		// Normal/Permissive: SMI global OID roots as fallback
		if !found && ctx.ResolverStrictness().AllowConstrainedFallbacks() {
			if node, ok := lookupSmiGlobalOidRoot(ctx, enterprise); ok {
				enterpriseNode = node
				found = true
			}
		}
		if !found {
			ctx.RecordUnresolvedOid(def.mod, defName, enterprise, span)
			continue
		}

		// RFC 3584 section 3 defines two OID conventions for TRAP-TYPE:
		//  - Generic traps (ENTERPRISE snmpTraps): snmpTraps.(trapNumber+1)
		//  - Enterprise-specific traps:            enterprise.0.trapNumber
		var trapNode *Node
		if isSnmpTrapsOID(enterpriseNode.OID()) {
			trapNode = enterpriseNode.getOrCreateChild(trapNumber + 1)
		} else {
			zeroNode := enterpriseNode.getOrCreateChild(0)
			trapNode = zeroNode.getOrCreateChild(trapNumber)
		}

		trapNode.setName(defName)
		trapNode.setKind(KindNotification)
		newMod := ctx.moduleToResolved[def.mod]
		if shouldPreferModule(ctx, trapNode.Module(), def.mod) {
			trapNode.setModule(newMod)
		}
		ctx.registerModuleNodeSymbol(def.mod, defName, trapNode)
		ctx.mib.registerNode(defName, trapNode)

		if ctx.TraceEnabled() {
			ctx.Trace("resolved TRAP-TYPE",
				slog.String("name", defName),
				slog.String("enterprise", enterprise),
				slog.Uint64("trapNumber", uint64(trapNumber)))
		}
	}
}

func lookupOrCreateWellKnownRoot(ctx *resolverContext, name string) (*Node, bool) {
	arc := wellKnownRootArc(name)
	if arc < 0 {
		return nil, false
	}
	return ctx.mib.Root().getOrCreateChild(uint32(arc)), true
}

func lookupSmiGlobalOidRoot(ctx *resolverContext, name string) (*Node, bool) {
	if _, ok := smiGlobalOidRoots[name]; !ok {
		return nil, false
	}
	if node, ok := ctx.LookupNodeInModule(moduleSNMPv2SMI, name); ok {
		return node, true
	}
	if node, ok := ctx.LookupNodeInModule(moduleRFC1155SMI, name); ok {
		return node, true
	}
	return nil, false
}

// snmpTrapsOID is 1.3.6.1.6.3.1.1.5 (SNMPv2-MIB::snmpTraps).
var snmpTrapsOID = OID{1, 3, 6, 1, 6, 3, 1, 1, 5}

// isSnmpTrapsOID returns true if oid matches the snmpTraps OID,
// identifying the enterprise node for the six standard generic traps.
func isSnmpTrapsOID(oid OID) bool {
	return oid.Equal(snmpTrapsOID)
}

// isRegisteredKind reports whether a node kind represents an actual SMI
// definition (OBJECT-TYPE, NOTIFICATION-TYPE, etc.) as opposed to an
// intermediate or value-assignment node.
func isRegisteredKind(k Kind) bool {
	switch k {
	case KindScalar, KindTable, KindRow, KindColumn,
		KindNotification, KindGroup, KindCompliance, KindCapability:
		return true
	default:
		return false
	}
}

func wellKnownRootArc(name string) int {
	switch name {
	case "ccitt":
		return 0
	case "iso":
		return 1
	case "joint-iso-ccitt":
		return 2
	default:
		return -1
	}
}

// shouldPreferModule determines if newMod should replace currentMod as the node's module.
// Preference order: base modules first, then SMIv2 > SMIv1 > Unknown, with newer
// LAST-UPDATED as tiebreaker.
func shouldPreferModule(ctx *resolverContext, currentMod *Module, srcMod *module.Module) bool {
	if currentMod == nil {
		return true
	}

	currentSrcMod := ctx.resolvedToModule[currentMod]
	if currentSrcMod == nil {
		return true
	}

	// Base modules always win. They are the authoritative source for
	// well-known OIDs (iso, org, dod, internet, etc.).
	newIsBase := module.IsBaseModule(srcMod.Name)
	currentIsBase := module.IsBaseModule(currentSrcMod.Name)
	if newIsBase != currentIsBase {
		return newIsBase
	}

	newRank := languageRank(srcMod.Language)
	currentRank := languageRank(currentSrcMod.Language)

	if ctx.TraceEnabled() {
		ctx.Trace("module preference check",
			slog.Group("new",
				slog.String("module", srcMod.Name),
				slog.String("language", srcMod.Language.String()),
				slog.Int("rank", newRank)),
			slog.Group("current",
				slog.String("module", currentSrcMod.Name),
				slog.String("language", currentSrcMod.Language.String()),
				slog.Int("rank", currentRank)))
	}

	if newRank > currentRank {
		return true
	}
	if newRank < currentRank {
		return false
	}

	// Same language - use LAST-UPDATED as tiebreaker (newer wins)
	newUpdated := extractLastUpdated(srcMod)
	currentUpdated := extractLastUpdated(currentSrcMod)
	return newUpdated > currentUpdated
}

// languageRank returns a numeric rank for language preference.
// Higher is better: SMIv2(2) > SMIv1(1) > Unknown(0)
func languageRank(lang types.Language) int {
	switch lang {
	case types.LanguageSMIv2:
		return 2
	case types.LanguageSMIv1:
		return 1
	default:
		return 0
	}
}
