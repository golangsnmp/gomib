package lower

import (
	"github.com/golangsnmp/gomib/internal/cst"
	"github.com/golangsnmp/gomib/internal/module"
)

// lowerOidValue converts a CST OidValueNode to module.OidAssignment.
func (l *lowerer) lowerOidValue(n *cst.OidValueNode) module.OidAssignment {
	if n == nil {
		return module.OidAssignment{}
	}

	var components []module.OidComponent
	for i := range n.Components {
		components = append(components, l.lowerOidComponent(&n.Components[i]))
	}

	return module.NewOidAssignment(components, n.Span)
}

// lowerOidComponent converts a single CST OidComponentNode to a module.OidComponent.
func (l *lowerer) lowerOidComponent(comp *cst.OidComponentNode) module.OidComponent {
	c := module.OidComponent{Span: comp.Span}
	if !comp.Module.IsZero() {
		c.Module = l.text(comp.Module)
	}
	if !comp.Name.IsZero() {
		c.Name = l.text(comp.Name)
	}
	if !comp.Number.IsZero() {
		c.Number = l.convertU32(comp.Number)
		c.HasNumber = true
	}
	return c
}
