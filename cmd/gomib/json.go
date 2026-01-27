package main

import (
	"encoding/json"
)

// DumpOutput is the top-level JSON structure for the dump command.
type DumpOutput struct {
	Modules       []ModuleJSON       `json:"modules,omitempty"`
	Types         []TypeJSON         `json:"types,omitempty"`
	Objects       []ObjectJSON       `json:"objects,omitempty"`
	Notifications []NotificationJSON `json:"notifications,omitempty"`
	Tree          *TreeNodeJSON      `json:"tree,omitempty"`
	Diagnostics   []DiagnosticJSON   `json:"diagnostics,omitempty"`
}

// ModuleJSON represents a resolved module in JSON.
type ModuleJSON struct {
	Name         string         `json:"name"`
	Language     string         `json:"language,omitempty"`
	OID          string         `json:"oid,omitempty"`
	Organization string         `json:"organization,omitempty"`
	ContactInfo  string         `json:"contactInfo,omitempty"`
	Description  string         `json:"description,omitempty"`
	Revisions    []RevisionJSON `json:"revisions,omitempty"`
}

// RevisionJSON represents a module revision.
type RevisionJSON struct {
	Date        string `json:"date"`
	Description string `json:"description,omitempty"`
}

// TypeJSON represents a resolved type in JSON.
type TypeJSON struct {
	Name        string      `json:"name"`
	Module      string      `json:"module,omitempty"`
	Base        string      `json:"base"`
	Parent      string      `json:"parent,omitempty"`
	Status      string      `json:"status"`
	Description string      `json:"description,omitempty"`
	Hint        string      `json:"hint,omitempty"`
	Size        []RangeJSON `json:"size,omitempty"`
	Range       []RangeJSON `json:"range,omitempty"`
	Enums       []EnumJSON  `json:"enums,omitempty"`
	Bits        []BitJSON   `json:"bits,omitempty"`
	IsTC        bool        `json:"isTextualConvention,omitempty"`
}

// RangeJSON represents a size or value range.
type RangeJSON struct {
	Min int64 `json:"min"`
	Max int64 `json:"max"`
}

// EnumJSON represents an enumeration value.
type EnumJSON struct {
	Label string `json:"label"`
	Value int64  `json:"value"`
}

// BitJSON represents a BITS position.
type BitJSON struct {
	Label    string `json:"label"`
	Position int    `json:"position"`
}

// ObjectJSON represents a resolved object in JSON.
type ObjectJSON struct {
	Name        string      `json:"name"`
	Module      string      `json:"module,omitempty"`
	OID         string      `json:"oid"`
	Kind        string      `json:"kind"`
	Type        string      `json:"type,omitempty"`
	BaseType    string      `json:"baseType,omitempty"`
	Access      string      `json:"access"`
	Status      string      `json:"status"`
	Description string      `json:"description,omitempty"`
	Units       string      `json:"units,omitempty"`
	Index       []IndexJSON `json:"index,omitempty"`
	Augments    string      `json:"augments,omitempty"`
	Enums       []EnumJSON  `json:"enums,omitempty"`
	Bits        []BitJSON   `json:"bits,omitempty"`
}

// IndexJSON represents an INDEX item.
type IndexJSON struct {
	Object  string `json:"object"`
	Implied bool   `json:"implied,omitempty"`
}

// NotificationJSON represents a resolved notification in JSON.
type NotificationJSON struct {
	Name        string   `json:"name"`
	Module      string   `json:"module,omitempty"`
	OID         string   `json:"oid"`
	Status      string   `json:"status"`
	Description string   `json:"description,omitempty"`
	Objects     []string `json:"objects,omitempty"`
}

// TreeNodeJSON represents a node in the OID tree.
type TreeNodeJSON struct {
	Arc      uint32          `json:"arc"`
	Label    string          `json:"label,omitempty"`
	Module   string          `json:"module,omitempty"`
	OID      string          `json:"oid"`
	Kind     string          `json:"kind,omitempty"`
	Children []*TreeNodeJSON `json:"children,omitempty"`
}

// DiagnosticJSON represents a diagnostic message.
type DiagnosticJSON struct {
	Severity string `json:"severity,omitempty"`
	Module   string `json:"module,omitempty"`
	Line     int    `json:"line,omitempty"`
	Message  string `json:"message"`
}

// marshalJSON serializes the output to JSON.
func marshalJSON(v any, indent bool) ([]byte, error) {
	if indent {
		return json.MarshalIndent(v, "", "  ")
	}
	return json.Marshal(v)
}
