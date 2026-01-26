package resolver

import (
	"log/slog"

	"github.com/golangsnmp/gomib/internal/mibimpl"
	"github.com/golangsnmp/gomib/internal/module"
	"github.com/golangsnmp/gomib/internal/types"
	"github.com/golangsnmp/gomib/mib"
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

func resolveOids(ctx *ResolverContext) {
	defs := collectOidDefinitions(ctx)
	pending := defs.oidDefs
	maxIterations := 20

	for iter := 0; iter < maxIterations && len(pending) > 0; iter++ {
		initial := len(pending)
		var still []oidDefinition
		for _, def := range pending {
			if !isFirstComponentResolvable(ctx, def) {
				still = append(still, def)
				continue
			}
			if !resolveOidDefinition(ctx, def) {
				still = append(still, def)
			}
		}

		if ctx.TraceEnabled() {
			resolved := initial - len(still)
			ctx.Trace("OID resolution pass",
				slog.Int("iteration", iter+1),
				slog.Int("resolved", resolved),
				slog.Int("still_pending", len(still)))
		}

		if len(still) == initial {
			for _, def := range still {
				oid := def.oid()
				if oid == nil || len(oid.Components) == 0 {
					continue
				}
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
			break
		}
		pending = still
	}

	resolveTrapTypeDefinitions(ctx, defs.trapDefs)
}

type oidDefinition struct {
	mod  *module.Module
	def  module.Definition
	kind definitionKind
}

type definitionKind int

const (
	defObjectType definitionKind = iota
	defModuleIdentity
	defObjectIdentity
	defNotification
	defValueAssignment
	defObjectGroup
	defNotificationGroup
	defModuleCompliance
	defAgentCapabilities
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

func collectOidDefinitions(ctx *ResolverContext) collectedOidDefinitions {
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

func isFirstComponentResolvable(ctx *ResolverContext, def oidDefinition) bool {
	oid := def.oid()
	if oid == nil || len(oid.Components) == 0 {
		return false
	}
	first := oid.Components[0]
	switch c := first.(type) {
	case *module.OidComponentName:
		if _, ok := ctx.LookupNodeForModule(def.mod, c.NameValue); ok {
			return true
		}
		if wellKnownRootArc(c.NameValue) >= 0 {
			return true
		}
		if _, ok := lookupSmiGlobalOidRoot(ctx, c.NameValue); ok {
			return true
		}
		return false
	case *module.OidComponentNumber:
		return true
	case *module.OidComponentNamedNumber:
		if _, ok := ctx.LookupNodeForModule(def.mod, c.NameValue); ok {
			return true
		}
		if wellKnownRootArc(c.NameValue) >= 0 {
			return true
		}
		if _, ok := lookupSmiGlobalOidRoot(ctx, c.NameValue); ok {
			return true
		}
		return true // Has a number, can resolve via that
	case *module.OidComponentQualifiedName:
		_, ok := ctx.LookupNodeInModule(c.ModuleValue, c.NameValue)
		return ok
	case *module.OidComponentQualifiedNamedNumber:
		_, ok := ctx.LookupNodeInModule(c.ModuleValue, c.NameValue)
		return ok
	default:
		return false
	}
}

func resolveOidDefinition(ctx *ResolverContext, def oidDefinition) bool {
	oid := def.oid()
	if oid == nil {
		return false
	}
	components := oid.Components
	if len(components) == 0 {
		return false
	}
	defName := def.defName()

	var currentNode *mibimpl.Node
	for idx, component := range components {
		isLast := idx == len(components)-1
		node, ok := resolveOidComponent(ctx, def, currentNode, component, isLast, oid.Span, defName)
		if !ok {
			return false
		}
		currentNode = node
	}

	if currentNode != nil {
		finalizeOidDefinition(ctx, def, currentNode, defName)
	}

	return true
}

func resolveOidComponent(ctx *ResolverContext, def oidDefinition, currentNode *mibimpl.Node, component module.OidComponent, isLast bool, span types.Span, defName string) (*mibimpl.Node, bool) {
	switch c := component.(type) {
	case *module.OidComponentName:
		return resolveNameComponent(ctx, def, c.NameValue, span, defName)
	case *module.OidComponentNumber:
		return resolveNumericComponent(ctx, currentNode, c.Value), true
	case *module.OidComponentNamedNumber:
		return resolveNamedNumberComponent(ctx, def, currentNode, c.NameValue, c.NumberValue, isLast)
	case *module.OidComponentQualifiedName:
		return resolveQualifiedNameComponent(ctx, def, c.ModuleValue, c.NameValue, span, defName)
	case *module.OidComponentQualifiedNamedNumber:
		return resolveQualifiedNamedNumberComponent(ctx, def, currentNode, c.ModuleValue, c.NameValue, c.NumberValue, isLast)
	default:
		ctx.RecordUnresolvedOid(def.mod, defName, "", span)
		return nil, false
	}
}

func resolveNameComponent(ctx *ResolverContext, def oidDefinition, name string, span types.Span, defName string) (*mibimpl.Node, bool) {
	if node, ok := ctx.LookupNodeForModule(def.mod, name); ok {
		return node, true
	}
	if node, ok := lookupOrCreateWellKnownRoot(ctx, name); ok {
		return node, true
	}
	if node, ok := lookupSmiGlobalOidRoot(ctx, name); ok {
		return node, true
	}
	ctx.RecordUnresolvedOid(def.mod, defName, name, span)
	return nil, false
}

func resolveNamedNumberComponent(ctx *ResolverContext, def oidDefinition, currentNode *mibimpl.Node, name string, number uint32, isLast bool) (*mibimpl.Node, bool) {
	if node, ok := ctx.LookupNodeForModule(def.mod, name); ok {
		ctx.RegisterModuleNodeSymbol(def.mod, name, node)
		return node, true
	}
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

func resolveQualifiedNameComponent(ctx *ResolverContext, def oidDefinition, moduleName, name string, span types.Span, defName string) (*mibimpl.Node, bool) {
	if node, ok := ctx.LookupNodeInModule(moduleName, name); ok {
		return node, true
	}
	ctx.RecordUnresolvedOid(def.mod, defName, moduleName+"."+name, span)
	return nil, false
}

func resolveQualifiedNamedNumberComponent(ctx *ResolverContext, def oidDefinition, currentNode *mibimpl.Node, moduleName, name string, number uint32, isLast bool) (*mibimpl.Node, bool) {
	if node, ok := ctx.LookupNodeInModule(moduleName, name); ok {
		ctx.RegisterModuleNodeSymbol(def.mod, name, node)
		return node, true
	}
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

func finalizeOidDefinition(ctx *ResolverContext, def oidDefinition, node *mibimpl.Node, label string) {
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
	node.SetModule(ctx.ModuleToResolved[def.mod])
	ctx.RegisterModuleNodeSymbol(def.mod, label, node)
	ctx.Builder.RegisterNode(label, node)

	if ctx.TraceEnabled() {
		ctx.Trace("resolved OID definition",
			slog.String("name", label),
			slog.Uint64("arc", uint64(node.Arc())),
			slog.String("kind", node.Kind().String()))
	}
}

func resolveNumericComponent(ctx *ResolverContext, parent *mibimpl.Node, arc uint32) *mibimpl.Node {
	if parent != nil {
		return parent.GetOrCreateChild(arc)
	}
	// No parent - this is a root
	return ctx.Builder.GetOrCreateRoot(arc)
}

func resolveTrapTypeDefinitions(ctx *ResolverContext, defs []trapTypeRef) {
	for _, def := range defs {
		enterprise, trapNumber, span, ok := def.trapInfo()
		if !ok {
			continue
		}
		defName := def.defName()

		enterpriseNode, found := ctx.LookupNodeForModule(def.mod, enterprise)
		if !found {
			if node, ok := lookupSmiGlobalOidRoot(ctx, enterprise); ok {
				enterpriseNode = node
				found = true
			}
		}
		if !found {
			ctx.RecordUnresolvedOid(def.mod, defName, enterprise, span)
			continue
		}

		// .0 node
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

func lookupOrCreateWellKnownRoot(ctx *ResolverContext, name string) (*mibimpl.Node, bool) {
	arc := wellKnownRootArc(name)
	if arc < 0 {
		return nil, false
	}
	return ctx.Builder.GetOrCreateRoot(uint32(arc)), true
}

func lookupSmiGlobalOidRoot(ctx *ResolverContext, name string) (*mibimpl.Node, bool) {
	if _, ok := smiGlobalOidRoots[name]; !ok {
		return nil, false
	}
	if node, ok := ctx.LookupNodeInModule("SNMPv2-SMI", name); ok {
		return node, true
	}
	if node, ok := ctx.LookupNodeInModule("RFC1155-SMI", name); ok {
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
