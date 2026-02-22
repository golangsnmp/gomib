package mib

import "slices"

// Type is a named type definition, either a TEXTUAL-CONVENTION or an inline
// type refinement. Types form chains via [Type.Parent]; the chain terminates
// at a base SMI type. Walking the chain with the Effective* methods resolves
// inherited constraints, display hints, and enum/BITS definitions.
type Type struct {
	name   string
	module *Module
	base   BaseType
	parent *Type
	status Status
	hint   string
	desc   string
	ref    string
	sizes  []Range
	ranges []Range
	enums  []NamedValue
	bits   []NamedValue
	isTC   bool
}

func newType(name string) *Type {
	return &Type{name: name}
}

// Name returns the type's name (e.g. "DisplayString"), or "" for anonymous types.
func (t *Type) Name() string { return t.name }

// Module returns the module that defines this type.
func (t *Type) Module() *Module { return t.module }

// Base returns the directly assigned base type, or 0 if inherited from the parent.
func (t *Type) Base() BaseType { return t.base }

// Parent returns the parent type in the type chain, or nil for root types.
func (t *Type) Parent() *Type { return t.parent }

// Status returns the STATUS clause value.
func (t *Type) Status() Status { return t.status }

// DisplayHint returns the DISPLAY-HINT string declared on this type, or "".
func (t *Type) DisplayHint() string { return t.hint }

// Description returns the DESCRIPTION clause text.
func (t *Type) Description() string { return t.desc }

// Reference returns the REFERENCE clause text, or "".
func (t *Type) Reference() string { return t.ref }

// Sizes returns the SIZE constraints declared directly on this type.
func (t *Type) Sizes() []Range { return slices.Clone(t.sizes) }

// Ranges returns the range constraints declared directly on this type.
func (t *Type) Ranges() []Range { return slices.Clone(t.ranges) }

// Enums returns the enumeration values declared directly on this type.
func (t *Type) Enums() []NamedValue { return slices.Clone(t.enums) }

// Bits returns the BITS definitions declared directly on this type.
func (t *Type) Bits() []NamedValue { return slices.Clone(t.bits) }

// Enum looks up an enumeration value by label.
func (t *Type) Enum(label string) (NamedValue, bool) { return findNamedValue(t.enums, label) }

// Bit looks up a BITS value by label.
func (t *Type) Bit(label string) (NamedValue, bool) { return findNamedValue(t.bits, label) }

// IsTextualConvention reports whether this type was defined as a TEXTUAL-CONVENTION.
func (t *Type) IsTextualConvention() bool { return t.isTC }

// IsCounter reports whether the resolved base type is Counter32 or Counter64.
func (t *Type) IsCounter() bool {
	b := t.EffectiveBase()
	return b == BaseCounter32 || b == BaseCounter64
}

// IsGauge reports whether the resolved base type is Gauge32.
func (t *Type) IsGauge() bool { return t.EffectiveBase() == BaseGauge32 }

// IsString reports whether the resolved base type is OCTET STRING.
func (t *Type) IsString() bool { return t.EffectiveBase() == BaseOctetString }

// IsEnumeration reports whether this is an INTEGER type with named values.
func (t *Type) IsEnumeration() bool {
	return t.EffectiveBase() == BaseInteger32 && len(t.EffectiveEnums()) > 0
}

// IsBits reports whether this type has BITS definitions.
func (t *Type) IsBits() bool { return len(t.EffectiveBits()) > 0 }

// EffectiveBase walks the parent type chain and returns the first non-zero
// base type, or 0 if none is set.
func (t *Type) EffectiveBase() BaseType {
	for current := t; current != nil; current = current.parent {
		if current.base != 0 {
			return current.base
		}
	}
	return 0
}

// EffectiveDisplayHint walks the parent type chain and returns the first
// non-empty display hint.
func (t *Type) EffectiveDisplayHint() string {
	for current := t; current != nil; current = current.parent {
		if current.hint != "" {
			return current.hint
		}
	}
	return ""
}

// EffectiveSizes walks the parent type chain and returns the first non-empty
// size constraint list.
func (t *Type) EffectiveSizes() []Range {
	for current := t; current != nil; current = current.parent {
		if len(current.sizes) > 0 {
			return slices.Clone(current.sizes)
		}
	}
	return nil
}

// EffectiveRanges walks the parent type chain and returns the first non-empty
// range constraint list.
func (t *Type) EffectiveRanges() []Range {
	for current := t; current != nil; current = current.parent {
		if len(current.ranges) > 0 {
			return slices.Clone(current.ranges)
		}
	}
	return nil
}

// EffectiveEnums walks the parent type chain and returns the first non-empty
// enumeration value list.
func (t *Type) EffectiveEnums() []NamedValue {
	for current := t; current != nil; current = current.parent {
		if len(current.enums) > 0 {
			return slices.Clone(current.enums)
		}
	}
	return nil
}

// EffectiveBits walks the parent type chain and returns the first non-empty
// BITS definition list.
func (t *Type) EffectiveBits() []NamedValue {
	for current := t; current != nil; current = current.parent {
		if len(current.bits) > 0 {
			return slices.Clone(current.bits)
		}
	}
	return nil
}

// String returns a brief summary: "Name (BaseType)" or just "BaseType"
// for anonymous types.
func (t *Type) String() string {
	if t == nil {
		return "<nil>"
	}
	if t.name == "" {
		return t.base.String()
	}
	return t.name + " (" + t.base.String() + ")"
}

func (t *Type) setModule(m *Module)     { t.module = m }
func (t *Type) setBase(b BaseType)      { t.base = b }
func (t *Type) setParent(p *Type)       { t.parent = p }
func (t *Type) setStatus(s Status)      { t.status = s }
func (t *Type) setDisplayHint(h string) { t.hint = h }
func (t *Type) setDescription(d string) { t.desc = d }
func (t *Type) setSizes(s []Range)      { t.sizes = s }
func (t *Type) setRanges(r []Range)     { t.ranges = r }
func (t *Type) setEnums(e []NamedValue) { t.enums = e }
func (t *Type) setBits(b []NamedValue)  { t.bits = b }
func (t *Type) setIsTC(isTC bool)       { t.isTC = isTC }
