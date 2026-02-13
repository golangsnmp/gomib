package resolver

import (
	"log/slog"
	"strings"

	"github.com/golangsnmp/gomib/internal/graph"
	"github.com/golangsnmp/gomib/internal/mibimpl"
	"github.com/golangsnmp/gomib/internal/module"
	"github.com/golangsnmp/gomib/internal/types"
	"github.com/golangsnmp/gomib/mib"
)

const (
	moduleSNMPv2SMI  = "SNMPv2-SMI"
	moduleRFC1155SMI = "RFC1155-SMI"
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

	g := graph.New()
	defIndex := make(map[graph.Symbol]oidDefinition)

	for _, def := range defs.oidDefs {
		sym := graph.Symbol{Module: def.mod.Name, Name: def.defName()}
		g.AddNode(sym, graph.NodeKindOID)
		defIndex[sym] = def

		if parentSym, ok := getOidParentSymbol(ctx, def); ok {
			g.AddEdge(sym, parentSym)
		}
	}

	cycles := g.FindCycles()
	logCycles(ctx, cycles, "OID cycle detected")

	order, cyclic := g.ResolutionOrder()

	if ctx.TraceEnabled() {
		ctx.Trace("OID resolution order",
			slog.Int("total", len(order)),
			slog.Int("cyclic", len(cyclic)))
	}

	resolved := 0
	for _, sym := range order {
		def, ok := defIndex[sym]
		if !ok {
			continue
		}
		if resolveOidDefinition(ctx, def) {
			resolved++
		}
	}

	for _, sym := range cyclic {
		def, ok := defIndex[sym]
		if !ok {
			continue
		}
		oid := def.oid()
		if oid == nil || len(oid.Components) == 0 {
			continue
		}
		recordUnresolvedFirstComponent(ctx, def, oid)
	}

	if ctx.TraceEnabled() {
		ctx.Trace("OID resolution complete",
			slog.Int("resolved", resolved),
			slog.Int("unresolved", len(cyclic)))
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
// local/imported definitions, and (in permissive mode) SMI global roots.
func lookupNamedParentSymbol(ctx *resolverContext, def oidDefinition, name string) (graph.Symbol, bool) {
	if wellKnownRootArc(name) >= 0 {
		return graph.Symbol{}, false
	}
	if parentMod := findOidDefiningModule(ctx, def.mod, name); parentMod != "" {
		return graph.Symbol{Module: parentMod, Name: name}, true
	}
	if ctx.DiagnosticConfig().AllowBestGuessFallbacks() {
		if _, ok := smiGlobalOidRoots[name]; ok {
			return graph.Symbol{Module: moduleSNMPv2SMI, Name: name}, true
		}
	}
	return graph.Symbol{}, false
}

// findOidDefiningModule finds the module that defines an OID symbol,
// checking local definitions first, then imports.
func findOidDefiningModule(ctx *resolverContext, fromMod *module.Module, name string) string {
	for _, def := range fromMod.Definitions {
		if def.DefinitionName() == name && def.DefinitionOid() != nil {
			return fromMod.Name
		}
	}

	if imports := ctx.ModuleImports[fromMod]; imports != nil {
		if srcMod := imports[name]; srcMod != nil {
			return srcMod.Name
		}
	}

	return ""
}

// recordUnresolvedFirstComponent records an unresolved OID based on its first component.
func recordUnresolvedFirstComponent(ctx *resolverContext, def oidDefinition, oid *module.OidAssignment) {
	defName := def.defName()
	span := oid.Span
	first := oid.Components[0]

	switch c := first.(type) {
	case *module.OidComponentName:
		ctx.RecordUnresolvedOid(def.mod, defName, c.NameValue, span)
	case *module.OidComponentNamedNumber:
		ctx.RecordUnresolvedOid(def.mod, defName, c.NameValue, span)
	case *module.OidComponentQualifiedName:
		ctx.RecordUnresolvedOid(def.mod, defName, c.ModuleValue+"."+c.NameValue, span)
	case *module.OidComponentQualifiedNamedNumber:
		ctx.RecordUnresolvedOid(def.mod, defName, c.ModuleValue+"."+c.NameValue, span)
	}
}

// checkSmiv2IdentifierHyphens emits a diagnostic for OID definition names
// containing hyphens in SMIv2 modules. smilint flags this at level 5.
func checkSmiv2IdentifierHyphens(ctx *resolverContext, defs []oidDefinition) {
	for _, def := range defs {
		if def.mod.Language != module.LanguageSMIv2 || module.IsBaseModule(def.mod.Name) {
			continue
		}
		name := def.defName()
		if strings.Contains(name, "-") {
			ctx.EmitDiagnostic("identifier-hyphen-smiv2", mib.SeverityWarning,
				def.mod.Name, 0, 0,
				"identifier "+name+" should not contain hyphens in SMIv2 MIB")
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

func (d trapTypeRef) trapInfo() (string, uint32, types.Span, bool) {
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
	var defs collectedOidDefinitions

	for _, mod := range ctx.Modules {
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
				if d.Oid != nil {
					kind = defNotification
				} else if d.TrapInfo != nil {
					defs.trapDefs = append(defs.trapDefs, trapTypeRef{mod: mod, notif: d})
					continue
				} else {
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

func resolveOidDefinition(ctx *resolverContext, def oidDefinition) bool {
	oid := def.oid()
	if oid == nil {
		return false
	}
	components := oid.Components
	if len(components) == 0 {
		return false
	}
	var currentNode *mibimpl.Node
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

func resolveOidComponent(ctx *resolverContext, def oidDefinition, currentNode *mibimpl.Node, component module.OidComponent, isLast bool) (*mibimpl.Node, bool) {
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

func resolveNameComponent(ctx *resolverContext, def oidDefinition, name string) (*mibimpl.Node, bool) {
	if node, ok := ctx.LookupNodeForModule(def.mod, name); ok {
		return node, true
	}
	// RFC-compliant: well-known roots (iso, ccitt, joint-iso-ccitt)
	if node, ok := lookupOrCreateWellKnownRoot(ctx, name); ok {
		return node, true
	}
	// Permissive only: SMI global OID roots without explicit import
	if ctx.DiagnosticConfig().AllowBestGuessFallbacks() {
		if node, ok := lookupSmiGlobalOidRoot(ctx, name); ok {
			return node, true
		}
	}
	ctx.RecordUnresolvedOid(def.mod, def.defName(), name, def.oid().Span)
	return nil, false
}

func resolveNamedNumberComponent(ctx *resolverContext, def oidDefinition, currentNode *mibimpl.Node, name string, number uint32, isLast bool) (*mibimpl.Node, bool) {
	if node, ok := ctx.LookupNodeForModule(def.mod, name); ok {
		ctx.RegisterModuleNodeSymbol(def.mod, name, node)
		return node, true
	}
	return createNamedChild(ctx, def, currentNode, name, number, isLast)
}

func resolveQualifiedNameComponent(ctx *resolverContext, def oidDefinition, moduleName, name string) (*mibimpl.Node, bool) {
	if node, ok := ctx.LookupNodeInModule(moduleName, name); ok {
		return node, true
	}
	ctx.RecordUnresolvedOid(def.mod, def.defName(), moduleName+"."+name, def.oid().Span)
	return nil, false
}

func resolveQualifiedNamedNumberComponent(ctx *resolverContext, def oidDefinition, currentNode *mibimpl.Node, moduleName, name string, number uint32, isLast bool) (*mibimpl.Node, bool) {
	if node, ok := ctx.LookupNodeInModule(moduleName, name); ok {
		ctx.RegisterModuleNodeSymbol(def.mod, name, node)
		return node, true
	}
	return createNamedChild(ctx, def, currentNode, name, number, isLast)
}

// createNamedChild resolves a numeric component and registers it with a name.
// Shared by resolveNamedNumberComponent and resolveQualifiedNamedNumberComponent.
func createNamedChild(ctx *resolverContext, def oidDefinition, currentNode *mibimpl.Node, name string, number uint32, isLast bool) (*mibimpl.Node, bool) {
	child := resolveNumericComponent(ctx, currentNode, number)
	if child == nil {
		return nil, false
	}
	ctx.RegisterModuleNodeSymbol(def.mod, name, child)
	if !isLast {
		child.SetName(name)
		child.SetModule(ctx.ModuleToResolved[def.mod])
		ctx.Builder.RegisterNode(name, child)
		if child.Kind() == mib.KindInternal {
			child.SetKind(mib.KindNode)
		}
	}
	return child, true
}

func finalizeOidDefinition(ctx *resolverContext, def oidDefinition, node *mibimpl.Node, label string) {
	switch def.kind {
	case defObjectType:
		node.SetKind(mib.KindScalar)
	case defModuleIdentity, defObjectIdentity, defValueAssignment:
		node.SetKind(mib.KindNode)
	case defNotification:
		node.SetKind(mib.KindNotification)
	case defObjectGroup, defNotificationGroup:
		node.SetKind(mib.KindGroup)
	case defModuleCompliance:
		node.SetKind(mib.KindCompliance)
	case defAgentCapabilities:
		node.SetKind(mib.KindCapabilities)
	}
	node.SetName(label)

	// Prefer SMIv2 over SMIv1 when multiple modules define the same OID
	newMod := ctx.ModuleToResolved[def.mod]
	currentMod := node.InternalModule()
	if currentMod != nil && ctx.TraceEnabled() {
		ctx.Trace("node already has module",
			slog.String("node", label),
			slog.String("current", currentMod.Name()),
			slog.String("new", def.mod.Name))
	}
	if shouldPreferModule(ctx, newMod, currentMod, def.mod) {
		node.SetModule(newMod)
		// Only register non-semantic definitions here; object types,
		// notifications, etc. are registered in the semantics phase.
		switch def.kind {
		case defValueAssignment, defObjectIdentity, defModuleIdentity:
			newMod.AddNode(node)
		}
		if def.kind == defModuleIdentity {
			newMod.SetOID(node.OID())
		}
	}

	ctx.RegisterModuleNodeSymbol(def.mod, label, node)
	ctx.Builder.RegisterNode(label, node)

	if ctx.TraceEnabled() {
		ctx.Trace("resolved OID definition",
			slog.String("name", label),
			slog.Uint64("arc", uint64(node.Arc())),
			slog.String("kind", node.Kind().String()))
	}
}

func resolveNumericComponent(ctx *resolverContext, parent *mibimpl.Node, arc uint32) *mibimpl.Node {
	if parent != nil {
		return parent.GetOrCreateChild(arc)
	}
	return ctx.Builder.GetOrCreateRoot(arc)
}

func resolveTrapTypeDefinitions(ctx *resolverContext, defs []trapTypeRef) {
	for _, def := range defs {
		enterprise, trapNumber, span, ok := def.trapInfo()
		if !ok {
			continue
		}
		defName := def.defName()

		enterpriseNode, found := ctx.LookupNodeForModule(def.mod, enterprise)
		// Permissive only: SMI global OID roots as fallback
		if !found && ctx.DiagnosticConfig().AllowBestGuessFallbacks() {
			if node, ok := lookupSmiGlobalOidRoot(ctx, enterprise); ok {
				enterpriseNode = node
				found = true
			}
		}
		if !found {
			ctx.RecordUnresolvedOid(def.mod, defName, enterprise, span)
			continue
		}

		// SNMPv1 trap OID convention: enterprise.0.trapNumber
		zeroNode := enterpriseNode.GetOrCreateChild(0)
		trapNode := zeroNode.GetOrCreateChild(trapNumber)

		trapNode.SetName(defName)
		trapNode.SetKind(mib.KindNotification)
		trapNode.SetModule(ctx.ModuleToResolved[def.mod])
		ctx.RegisterModuleNodeSymbol(def.mod, defName, trapNode)
		ctx.Builder.RegisterNode(defName, trapNode)

		if ctx.TraceEnabled() {
			ctx.Trace("resolved TRAP-TYPE",
				slog.String("name", defName),
				slog.String("enterprise", enterprise),
				slog.Uint64("trapNumber", uint64(trapNumber)))
		}
	}
}

func lookupOrCreateWellKnownRoot(ctx *resolverContext, name string) (*mibimpl.Node, bool) {
	arc := wellKnownRootArc(name)
	if arc < 0 {
		return nil, false
	}
	return ctx.Builder.GetOrCreateRoot(uint32(arc)), true
}

func lookupSmiGlobalOidRoot(ctx *resolverContext, name string) (*mibimpl.Node, bool) {
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
// Preference order: SMIv2 > SMIv1 > Unknown, with newer LAST-UPDATED as tiebreaker.
func shouldPreferModule(ctx *resolverContext, newMod, currentMod *mibimpl.Module, srcMod *module.Module) bool {
	if currentMod == nil {
		return true
	}

	currentSrcMod := ctx.ResolvedToModule[currentMod]
	if currentSrcMod == nil {
		return true
	}

	newRank := languageRank(srcMod.Language)
	currentRank := languageRank(currentSrcMod.Language)

	if ctx.TraceEnabled() {
		ctx.Trace("module preference check",
			slog.String("new", srcMod.Name),
			slog.String("newLang", srcMod.Language.String()),
			slog.Int("newRank", newRank),
			slog.String("current", currentSrcMod.Name),
			slog.String("currentLang", currentSrcMod.Language.String()),
			slog.Int("currentRank", currentRank))
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
// Higher is better: SMIv2(2) > SMIv1(1) > Unknown/SPPI(0)
func languageRank(lang module.Language) int {
	switch lang {
	case module.LanguageSMIv2:
		return 2
	case module.LanguageSMIv1:
		return 1
	default:
		return 0
	}
}
