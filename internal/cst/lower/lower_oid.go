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

// lowerOidComponent converts a single CST OidComponentNode to the appropriate
// module.OidComponent variant.
func (l *lowerer) lowerOidComponent(comp *cst.OidComponentNode) module.OidComponent {
	hasModule := !comp.Module.IsZero()
	hasName := !comp.Name.IsZero()
	hasNumber := !comp.Number.IsZero()

	switch { //nolint:gocritic
	case hasModule && hasName && hasNumber:
		// Qualified named number: Module.name(number)
		return &module.OidComponentQualifiedNamedNumber{
			ModuleValue: l.text(comp.Module),
			NameValue:   l.text(comp.Name),
			NumberValue: l.convertU32(comp.Number),
			Span:        comp.Span,
		}
	case hasModule && hasName:
		// Module.name
		return &module.OidComponentQualifiedName{
			ModuleValue: l.text(comp.Module),
			NameValue:   l.text(comp.Name),
			Span:        comp.Span,
		}
	case hasName && hasNumber:
		// name(number)
		return &module.OidComponentNamedNumber{
			NameValue:   l.text(comp.Name),
			NumberValue: l.convertU32(comp.Number),
			Span:        comp.Span,
		}
	case hasName:
		// Just name
		return &module.OidComponentName{
			NameValue: l.text(comp.Name),
			Span:      comp.Name.Span,
		}
	case hasNumber:
		// Just number
		return &module.OidComponentNumber{
			Value: l.convertU32(comp.Number),
			Span:  comp.Number.Span,
		}
	default:
		// Should not happen with valid CST
		return &module.OidComponentNumber{
			Value: 0,
			Span:  comp.Span,
		}
	}
}
