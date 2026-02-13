// Package mibimpl provides concrete implementations of the mib package
// interfaces, along with a Builder for constructing a Mib incrementally.
package mibimpl

import (
	"slices"

	"github.com/golangsnmp/gomib/mib"
)

// Type implements mib.Type with a parent chain for textual convention
// inheritance.
type Type struct {
	name   string
	module *Module
	base   mib.BaseType
	parent *Type
	status mib.Status
	hint   string
	desc   string
	ref    string
	sizes  []mib.Range
	ranges []mib.Range
	enums  []mib.NamedValue
	bits   []mib.NamedValue
	isTC   bool
}

func (t *Type) Name() string {
	return t.name
}

func (t *Type) Module() mib.Module {
	if t.module == nil {
		return nil
	}
	return t.module
}

func (t *Type) Base() mib.BaseType {
	return t.base
}

func (t *Type) Parent() mib.Type {
	if t.parent == nil {
		return nil
	}
	return t.parent
}

func (t *Type) Status() mib.Status {
	return t.status
}

func (t *Type) DisplayHint() string {
	return t.hint
}

func (t *Type) Description() string {
	return t.desc
}

func (t *Type) Reference() string {
	return t.ref
}

func (t *Type) Sizes() []mib.Range {
	return slices.Clone(t.sizes)
}

func (t *Type) Ranges() []mib.Range {
	return slices.Clone(t.ranges)
}

func (t *Type) Enums() []mib.NamedValue {
	return slices.Clone(t.enums)
}

func (t *Type) Bits() []mib.NamedValue {
	return slices.Clone(t.bits)
}

func (t *Type) Enum(label string) (mib.NamedValue, bool) {
	for _, nv := range t.enums {
		if nv.Label == label {
			return nv, true
		}
	}
	return mib.NamedValue{}, false
}

func (t *Type) Bit(label string) (mib.NamedValue, bool) {
	for _, nv := range t.bits {
		if nv.Label == label {
			return nv, true
		}
	}
	return mib.NamedValue{}, false
}

func (t *Type) IsTextualConvention() bool {
	return t.isTC
}

func (t *Type) IsCounter() bool {
	b := t.EffectiveBase()
	return b == mib.BaseCounter32 || b == mib.BaseCounter64
}

func (t *Type) IsGauge() bool {
	return t.EffectiveBase() == mib.BaseGauge32
}

func (t *Type) IsString() bool {
	return t.EffectiveBase() == mib.BaseOctetString
}

func (t *Type) IsEnumeration() bool {
	return t.EffectiveBase() == mib.BaseInteger32 && len(t.EffectiveEnums()) > 0
}

func (t *Type) IsBits() bool {
	return len(t.EffectiveBits()) > 0
}

func (t *Type) EffectiveBase() mib.BaseType {
	return t.base
}

func (t *Type) EffectiveDisplayHint() string {
	for current := t; current != nil; current = current.parent {
		if current.hint != "" {
			return current.hint
		}
	}
	return ""
}

func (t *Type) EffectiveSizes() []mib.Range {
	for current := t; current != nil; current = current.parent {
		if len(current.sizes) > 0 {
			return slices.Clone(current.sizes)
		}
	}
	return nil
}

func (t *Type) EffectiveRanges() []mib.Range {
	for current := t; current != nil; current = current.parent {
		if len(current.ranges) > 0 {
			return slices.Clone(current.ranges)
		}
	}
	return nil
}

func (t *Type) EffectiveEnums() []mib.NamedValue {
	for current := t; current != nil; current = current.parent {
		if len(current.enums) > 0 {
			return slices.Clone(current.enums)
		}
	}
	return nil
}

func (t *Type) EffectiveBits() []mib.NamedValue {
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

func (t *Type) SetName(name string) {
	t.name = name
}

func (t *Type) SetModule(m *Module) {
	t.module = m
}

func (t *Type) SetBase(b mib.BaseType) {
	t.base = b
}

func (t *Type) SetParent(p *Type) {
	t.parent = p
}

func (t *Type) SetStatus(s mib.Status) {
	t.status = s
}

func (t *Type) SetDisplayHint(h string) {
	t.hint = h
}

func (t *Type) SetDescription(d string) {
	t.desc = d
}

func (t *Type) SetReference(r string) {
	t.ref = r
}

func (t *Type) SetSizes(s []mib.Range) {
	t.sizes = s
}

func (t *Type) SetRanges(r []mib.Range) {
	t.ranges = r
}

func (t *Type) SetEnums(e []mib.NamedValue) {
	t.enums = e
}

func (t *Type) SetBits(b []mib.NamedValue) {
	t.bits = b
}

func (t *Type) SetIsTC(isTC bool) {
	t.isTC = isTC
}

// InternalParent returns the concrete parent type.
func (t *Type) InternalParent() *Type {
	return t.parent
}

// InternalModule returns the concrete module.
func (t *Type) InternalModule() *Module {
	return t.module
}
