package resolver

import (
	"github.com/golangsnmp/gomib/mib"
	"log/slog"

	"github.com/golangsnmp/gomib/internal/module"
	"github.com/golangsnmp/gomib/internal/types"
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
				if name, ok := first.Name(); ok {
					ctx.RecordUnresolvedOid(def.mod, defName, name, span)
				} else if moduleName, ok := first.Module(); ok {
					if name, ok := first.Name(); ok {
						ctx.RecordUnresolvedOid(def.mod, defName, moduleName+"."+name, span)
					}
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
	if name, ok := first.Name(); ok {
		if _, ok := ctx.LookupNodeForModule(def.mod, name); ok {
			return true
		}
		if wellKnownRootArc(name) >= 0 {
			return true
		}
		if _, ok := lookupSmiGlobalOidRoot(ctx, name); ok {
			return true
		}
		return false
	}
	if _, ok := first.Number(); ok {
		return true
	}
	if moduleName, ok := first.Module(); ok {
		name, ok := first.Name()
		if !ok {
			return false
		}
		_, ok = ctx.LookupNodeInModule(moduleName, name)
		return ok
	}
	return false
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

	var currentNode *mib.Node
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

func resolveOidComponent(ctx *ResolverContext, def oidDefinition, currentNode *mib.Node, component module.OidComponent, isLast bool, span types.Span, defName string) (*mib.Node, bool) {
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

func resolveNameComponent(ctx *ResolverContext, def oidDefinition, name string, span types.Span, defName string) (*mib.Node, bool) {
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

func resolveNamedNumberComponent(ctx *ResolverContext, def oidDefinition, currentNode *mib.Node, name string, number uint32, isLast bool) (*mib.Node, bool) {
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
		child.Name = name
		child.Module = ctx.ModuleToResolved[def.mod]
		ctx.Builder.RegisterNode(name, child)
		if child.Kind == mib.KindInternal {
			child.Kind = mib.KindNode
		}
	}
	return child, true
}

func resolveQualifiedNameComponent(ctx *ResolverContext, def oidDefinition, moduleName, name string, span types.Span, defName string) (*mib.Node, bool) {
	if node, ok := ctx.LookupNodeInModule(moduleName, name); ok {
		return node, true
	}
	ctx.RecordUnresolvedOid(def.mod, defName, moduleName+"."+name, span)
	return nil, false
}

func resolveQualifiedNamedNumberComponent(ctx *ResolverContext, def oidDefinition, currentNode *mib.Node, moduleName, name string, number uint32, isLast bool) (*mib.Node, bool) {
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
		child.Name = name
		child.Module = ctx.ModuleToResolved[def.mod]
		ctx.Builder.RegisterNode(name, child)
		if child.Kind == mib.KindInternal {
			child.Kind = mib.KindNode
		}
	}
	return child, true
}

func finalizeOidDefinition(ctx *ResolverContext, def oidDefinition, node *mib.Node, label string) {
	switch def.kind {
	case defObjectType:
		node.Kind = mib.KindScalar
	case defModuleIdentity, defObjectIdentity, defValueAssignment:
		node.Kind = mib.KindNode
	case defNotification:
		node.Kind = mib.KindNotification
	case defObjectGroup, defNotificationGroup:
		node.Kind = mib.KindGroup
	case defModuleCompliance:
		node.Kind = mib.KindCompliance
	case defAgentCapabilities:
		node.Kind = mib.KindCapabilities
	}
	node.Name = label
	node.Module = ctx.ModuleToResolved[def.mod]
	ctx.RegisterModuleNodeSymbol(def.mod, label, node)
	ctx.Builder.RegisterNode(label, node)

	if ctx.TraceEnabled() {
		ctx.Trace("resolved OID definition",
			slog.String("name", label),
			slog.Uint64("arc", uint64(node.Arc())),
			slog.String("kind", node.Kind.String()))
	}
}

func resolveNumericComponent(ctx *ResolverContext, parent *mib.Node, arc uint32) *mib.Node {
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

		trapNode.Name = defName
		trapNode.Kind = mib.KindNotification
		trapNode.Module = ctx.ModuleToResolved[def.mod]
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

func lookupOrCreateWellKnownRoot(ctx *ResolverContext, name string) (*mib.Node, bool) {
	arc := wellKnownRootArc(name)
	if arc < 0 {
		return nil, false
	}
	return ctx.Builder.GetOrCreateRoot(uint32(arc)), true
}

func lookupSmiGlobalOidRoot(ctx *ResolverContext, name string) (*mib.Node, bool) {
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
