package mib

import "slices"

// Type is a type definition (textual convention or type reference).
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

// newType returns a Type initialized with the given name.
func newType(name string) *Type {
	return &Type{name: name}
}

func (t *Type) Name() string        { return t.name }
func (t *Type) Module() *Module     { return t.module }
func (t *Type) Base() BaseType      { return t.base }
func (t *Type) Parent() *Type       { return t.parent }
func (t *Type) Status() Status      { return t.status }
func (t *Type) DisplayHint() string { return t.hint }
func (t *Type) Description() string { return t.desc }
func (t *Type) Reference() string   { return t.ref }
func (t *Type) Sizes() []Range      { return slices.Clone(t.sizes) }
func (t *Type) Ranges() []Range     { return slices.Clone(t.ranges) }
func (t *Type) Enums() []NamedValue { return slices.Clone(t.enums) }
func (t *Type) Bits() []NamedValue  { return slices.Clone(t.bits) }

func (t *Type) Enum(label string) (NamedValue, bool) {
	for _, nv := range t.enums {
		if nv.Label == label {
			return nv, true
		}
	}
	return NamedValue{}, false
}

func (t *Type) Bit(label string) (NamedValue, bool) {
	for _, nv := range t.bits {
		if nv.Label == label {
			return nv, true
		}
	}
	return NamedValue{}, false
}

func (t *Type) IsTextualConvention() bool { return t.isTC }

func (t *Type) IsCounter() bool {
	b := t.EffectiveBase()
	return b == BaseCounter32 || b == BaseCounter64
}

func (t *Type) IsGauge() bool  { return t.EffectiveBase() == BaseGauge32 }
func (t *Type) IsString() bool { return t.EffectiveBase() == BaseOctetString }

func (t *Type) IsEnumeration() bool {
	return t.EffectiveBase() == BaseInteger32 && len(t.EffectiveEnums()) > 0
}

func (t *Type) IsBits() bool { return len(t.EffectiveBits()) > 0 }

func (t *Type) EffectiveBase() BaseType { return t.base }

func (t *Type) EffectiveDisplayHint() string {
	for current := t; current != nil; current = current.parent {
		if current.hint != "" {
			return current.hint
		}
	}
	return ""
}

func (t *Type) EffectiveSizes() []Range {
	for current := t; current != nil; current = current.parent {
		if len(current.sizes) > 0 {
			return slices.Clone(current.sizes)
		}
	}
	return nil
}

func (t *Type) EffectiveRanges() []Range {
	for current := t; current != nil; current = current.parent {
		if len(current.ranges) > 0 {
			return slices.Clone(current.ranges)
		}
	}
	return nil
}

func (t *Type) EffectiveEnums() []NamedValue {
	for current := t; current != nil; current = current.parent {
		if len(current.enums) > 0 {
			return slices.Clone(current.enums)
		}
	}
	return nil
}

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
