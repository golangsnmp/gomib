package main

import (
	"encoding/json"
	"io"

	"github.com/golangsnmp/gomib/mib"
)

// JSONOptions controls which fields are included in JSON output for the get command.
type JSONOptions struct {
	Compact      bool
	IncludeDescr bool
}

// ObjectJSON holds the JSON-serializable form of a resolved object.
type ObjectJSON struct {
	Name             string              `json:"name"`
	Module           string              `json:"module,omitempty"`
	OID              string              `json:"oid"`
	Kind             string              `json:"kind"`
	Type             string              `json:"type,omitempty"`
	BaseType         string              `json:"baseType,omitempty"`
	Access           string              `json:"access"`
	Status           string              `json:"status"`
	Description      string              `json:"description,omitempty"`
	Units            string              `json:"units,omitempty"`
	DefaultValue     string              `json:"defaultValue,omitempty"`
	Index            []IndexJSON         `json:"index,omitempty"`
	Augments         string              `json:"augments,omitempty"`
	Entry            string              `json:"entry,omitempty"`
	Table            string              `json:"table,omitempty"`
	Row              string              `json:"row,omitempty"`
	AugmentedBy      []string            `json:"augmentedBy,omitempty"`
	EffectiveIndexes []IndexJSON         `json:"effectiveIndexes,omitempty"`
	IsIndex          *bool               `json:"isIndex,omitempty"`
	Columns          []ColumnSummaryJSON `json:"columns,omitempty"`
	Enums            []EnumJSON          `json:"enums,omitempty"`
	Bits             []BitJSON           `json:"bits,omitempty"`
}

// IndexJSON holds an INDEX entry with its implied flag.
type IndexJSON struct {
	Object   string `json:"object"`
	Implied  bool   `json:"implied,omitempty"`
	Encoding string `json:"encoding,omitempty"`
}

// ColumnSummaryJSON holds a brief summary of a table column.
type ColumnSummaryJSON struct {
	Name     string `json:"name"`
	Type     string `json:"type,omitempty"`
	BaseType string `json:"baseType,omitempty"`
	Access   string `json:"access"`
	IsIndex  bool   `json:"isIndex"`
}

// EnumJSON holds a named enumeration value.
type EnumJSON struct {
	Label string `json:"label"`
	Value int64  `json:"value"`
}

// BitJSON holds a named BITS position.
type BitJSON struct {
	Label    string `json:"label"`
	Position int    `json:"position"`
}

// NotificationJSON holds the JSON-serializable form of a notification.
type NotificationJSON struct {
	Name        string   `json:"name"`
	Module      string   `json:"module,omitempty"`
	OID         string   `json:"oid"`
	Status      string   `json:"status"`
	Description string   `json:"description,omitempty"`
	Objects     []string `json:"objects,omitempty"`
}

// TreeNodeJSON holds a node in the OID tree hierarchy.
type TreeNodeJSON struct {
	Arc      uint32          `json:"arc"`
	Label    string          `json:"label,omitempty"`
	Module   string          `json:"module,omitempty"`
	OID      string          `json:"oid"`
	Kind     string          `json:"kind,omitempty"`
	Children []*TreeNodeJSON `json:"children,omitempty"`
}

func buildObjectJSON(obj *mib.Object, opts JSONOptions) ObjectJSON {
	o := ObjectJSON{
		Name:   obj.Name(),
		OID:    obj.OID().String(),
		Kind:   obj.Kind().String(),
		Access: obj.Access().String(),
		Status: obj.Status().String(),
		Units:  obj.Units(),
	}

	if obj.Module() != nil {
		o.Module = obj.Module().Name()
	}

	if obj.Type() != nil {
		o.Type = obj.Type().Name()
		if o.Type == "" {
			o.Type = obj.Type().Base().String()
		}
		o.BaseType = obj.Type().Base().String()
	}

	if opts.IncludeDescr {
		o.Description = obj.Description()
	}

	for _, idx := range obj.Index() {
		idxJSON := IndexJSON{Implied: idx.Implied}
		if idx.Object != nil {
			idxJSON.Object = idx.Object.Name()
		}
		if idx.Encoding != mib.IndexEncodingUnknown {
			idxJSON.Encoding = idx.Encoding.String()
		}
		o.Index = append(o.Index, idxJSON)
	}

	if dv := obj.DefaultValue(); !dv.IsZero() {
		o.DefaultValue = dv.String()
	}

	if obj.Augments() != nil {
		o.Augments = obj.Augments().Name()
	}

	switch obj.Kind() {
	case mib.KindTable:
		if entry := obj.Entry(); entry != nil {
			o.Entry = entry.Name()
		}
		o.Columns = buildColumnSummaries(obj)
	case mib.KindRow:
		if tbl := obj.Table(); tbl != nil {
			o.Table = tbl.Name()
		}
		o.Columns = buildColumnSummaries(obj)
		for _, aug := range obj.AugmentedBy() {
			o.AugmentedBy = append(o.AugmentedBy, aug.Name())
		}
		if obj.Augments() != nil {
			for _, idx := range obj.EffectiveIndexes() {
				ij := IndexJSON{Implied: idx.Implied}
				if idx.Object != nil {
					ij.Object = idx.Object.Name()
				}
				if idx.Encoding != mib.IndexEncodingUnknown {
					ij.Encoding = idx.Encoding.String()
				}
				o.EffectiveIndexes = append(o.EffectiveIndexes, ij)
			}
		}
	case mib.KindColumn:
		if tbl := obj.Table(); tbl != nil {
			o.Table = tbl.Name()
		}
		if row := obj.Row(); row != nil {
			o.Row = row.Name()
		}
		isIdx := obj.IsIndex()
		o.IsIndex = &isIdx
	}

	for _, nv := range obj.EffectiveEnums() {
		o.Enums = append(o.Enums, EnumJSON{Label: nv.Label, Value: nv.Value})
	}
	for _, nv := range obj.EffectiveBits() {
		o.Bits = append(o.Bits, BitJSON{Label: nv.Label, Position: int(nv.Value)})
	}

	return o
}

func buildNotificationJSON(notif *mib.Notification, opts JSONOptions) NotificationJSON {
	n := NotificationJSON{
		Name:   notif.Name(),
		OID:    notif.OID().String(),
		Status: notif.Status().String(),
	}

	if notif.Module() != nil {
		n.Module = notif.Module().Name()
	}

	if opts.IncludeDescr {
		n.Description = notif.Description()
	}

	for _, obj := range notif.Objects() {
		n.Objects = append(n.Objects, obj.Name())
	}

	return n
}

func buildTreeJSON(node *mib.Node, opts JSONOptions) *TreeNodeJSON { //nolint:unparam // kept for API consistency
	t := &TreeNodeJSON{
		Arc:   node.Arc(),
		OID:   node.OID().String(),
		Label: node.Name(),
		Kind:  node.Kind().String(),
	}

	if node.Module() != nil {
		t.Module = node.Module().Name()
	}

	for _, child := range node.Children() {
		t.Children = append(t.Children, buildTreeJSON(child, opts))
	}

	return t
}

func buildColumnSummaries(obj *mib.Object) []ColumnSummaryJSON {
	cols := obj.Columns()
	if len(cols) == 0 {
		return nil
	}
	summaries := make([]ColumnSummaryJSON, 0, len(cols))
	for _, col := range cols {
		cs := ColumnSummaryJSON{
			Name:   col.Name(),
			Access: col.Access().String(),
		}
		if t := col.Type(); t != nil {
			cs.Type = t.Name()
			if cs.Type == "" {
				cs.Type = t.Base().String()
			}
			cs.BaseType = t.EffectiveBase().String()
		}
		cs.IsIndex = col.IsIndex()
		summaries = append(summaries, cs)
	}
	return summaries
}

func writeJSON(w io.Writer, v any, indent bool) error {
	enc := json.NewEncoder(w)
	if indent {
		enc.SetIndent("", "  ")
	}
	return enc.Encode(v)
}
