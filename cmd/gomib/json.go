package main

import (
	"encoding/json"
)

// DumpOutput is the top-level JSON output for the dump command.
type DumpOutput struct {
	Modules       []ModuleJSON       `json:"modules,omitempty"`
	Types         []TypeJSON         `json:"types,omitempty"`
	Objects       []ObjectJSON       `json:"objects,omitempty"`
	Notifications []NotificationJSON `json:"notifications,omitempty"`
	Tree          *TreeNodeJSON      `json:"tree,omitempty"`
	Diagnostics   []DiagnosticJSON   `json:"diagnostics,omitempty"`
}

// ModuleJSON holds the JSON-serializable form of a resolved module.
type ModuleJSON struct {
	Name         string         `json:"name"`
	Language     string         `json:"language,omitempty"`
	SourcePath   string         `json:"sourcePath,omitempty"`
	OID          string         `json:"oid,omitempty"`
	Organization string         `json:"organization,omitempty"`
	ContactInfo  string         `json:"contactInfo,omitempty"`
	Description  string         `json:"description,omitempty"`
	Revisions    []RevisionJSON `json:"revisions,omitempty"`
}

// RevisionJSON holds a module revision entry.
type RevisionJSON struct {
	Date        string `json:"date"`
	Description string `json:"description,omitempty"`
}

// TypeJSON holds the JSON-serializable form of a resolved type.
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

// RangeJSON holds a size or value constraint range.
type RangeJSON struct {
	Min int64 `json:"min"`
	Max int64 `json:"max"`
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

// ObjectJSON holds the JSON-serializable form of a resolved object.
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

// IndexJSON holds an INDEX entry with its implied flag.
type IndexJSON struct {
	Object  string `json:"object"`
	Implied bool   `json:"implied,omitempty"`
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

// DiagnosticJSON holds a parser or resolver diagnostic.
type DiagnosticJSON struct {
	Severity string `json:"severity,omitempty"`
	Module   string `json:"module,omitempty"`
	Line     int    `json:"line,omitempty"`
	Message  string `json:"message"`
}

func marshalJSON(v any, indent bool) ([]byte, error) {
	if indent {
		return json.MarshalIndent(v, "", "  ")
	}
	return json.Marshal(v)
}
