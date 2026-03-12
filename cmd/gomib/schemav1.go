package main

import (
	"encoding/hex"
	"runtime/debug"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/golangsnmp/gomib/internal/types"
	"github.com/golangsnmp/gomib/mib"
)

// V1 schema JSON types per compare-schema-v1.md

type V1Export struct {
	SchemaVersion int              `json:"schemaVersion"`
	ExportKind    string           `json:"exportKind"`
	Strictness    string           `json:"strictness"`
	Exporter      V1Exporter       `json:"exporter"`
	Modules       []V1Module       `json:"modules"`
	Types         []V1Type         `json:"types"`
	Nodes         []V1Node         `json:"nodes"`
	Objects       []V1Object       `json:"objects"`
	Notifications []V1Notification `json:"notifications"`
	Groups        []V1Group        `json:"groups"`
	Compliances   []V1Compliance   `json:"compliances"`
	Capabilities  []V1Capability   `json:"capabilities"`
	Diagnostics   []V1Diagnostic   `json:"diagnostics"`
}

type V1Exporter struct {
	Implementation string `json:"implementation"`
	Version        string `json:"version"`
	Commit         string `json:"commit"`
}

type V1Module struct {
	Name         string       `json:"name"`
	OID          *string      `json:"oid"`
	Language     *string      `json:"language"`
	Organization *string      `json:"organization"`
	ContactInfo  *string      `json:"contactInfo"`
	Description  *string      `json:"description"`
	LastUpdated  *string      `json:"lastUpdated"`
	Revisions    []V1Revision `json:"revisions"`
}

type V1Revision struct {
	Date        string  `json:"date"`
	Description *string `json:"description"`
}

type V1Type struct {
	Key                 string        `json:"key"`
	Name                string        `json:"name"`
	Module              string        `json:"module"`
	Parent              *string       `json:"parent"`
	Base                string        `json:"base"`
	Status              *string       `json:"status"`
	DisplayHint         *string       `json:"displayHint"`
	Description         *string       `json:"description"`
	Reference           *string       `json:"reference"`
	IsTextualConvention bool          `json:"isTextualConvention"`
	Constraints         V1Constraints `json:"constraints"`
}

type V1Constraints struct {
	Sizes  []V1Range     `json:"sizes"`
	Ranges []V1Range     `json:"ranges"`
	Enums  []V1EnumEntry `json:"enums"`
	Bits   []V1BitEntry  `json:"bits"`
}

type V1Range struct {
	Min string `json:"min"`
	Max string `json:"max"`
}

type V1EnumEntry struct {
	Name  string `json:"name"`
	Value int64  `json:"value"`
}

type V1BitEntry struct {
	Name     string `json:"name"`
	Position int64  `json:"position"`
}

type V1Node struct {
	Key         string  `json:"key"`
	OID         string  `json:"oid"`
	Name        string  `json:"name"`
	Module      string  `json:"module"`
	Kind        string  `json:"kind"`
	Status      *string `json:"status"`
	Description *string `json:"description"`
	Reference   *string `json:"reference"`
}

type V1Object struct {
	Key              string             `json:"key"`
	OID              string             `json:"oid"`
	Name             string             `json:"name"`
	Module           string             `json:"module"`
	Kind             string             `json:"kind"`
	Status           string             `json:"status"`
	Access           string             `json:"access"`
	Description      *string            `json:"description"`
	Reference        *string            `json:"reference"`
	Units            *string            `json:"units"`
	DefaultValue     *V1DefaultValue    `json:"defaultValue"`
	Syntax           *V1EffectiveSyntax `json:"syntax"`
	Indexes          []V1IndexEntry     `json:"indexes"`
	EffectiveIndexes []V1IndexEntry     `json:"effectiveIndexes"`
	Augments         *V1ObjRef          `json:"augments"`
	AugmentedBy      []V1ObjRef         `json:"augmentedBy"`
	Table            *V1ObjRef          `json:"table"`
	Row              *V1ObjRef          `json:"row"`
	Columns          []V1ObjRef         `json:"columns"`
}

type V1EffectiveSyntax struct {
	TypeRef     *string       `json:"typeRef"`
	Base        string        `json:"base"`
	DisplayHint *string       `json:"displayHint"`
	Constraints V1Constraints `json:"constraints"`
}

type V1ObjRef struct {
	Name   string `json:"name"`
	Module string `json:"module"`
	OID    string `json:"oid"`
}

type V1IndexEntry struct {
	Object  *V1ObjRef          `json:"object"`
	Implied bool               `json:"implied"`
	Syntax  *V1EffectiveSyntax `json:"syntax"`
}

type V1DefaultValue struct {
	Kind  string `json:"kind"`
	Value any    `json:"value"`
	Raw   string `json:"raw"`
}

type V1Notification struct {
	Key         string     `json:"key"`
	OID         string     `json:"oid"`
	Name        string     `json:"name"`
	Module      string     `json:"module"`
	Status      string     `json:"status"`
	Kind        string     `json:"kind"`
	Trap        *V1Trap    `json:"trap"`
	Description string     `json:"description"`
	Reference   *string    `json:"reference"`
	Objects     []V1ObjRef `json:"objects"`
}

type V1Trap struct {
	Enterprise V1ObjRef `json:"enterprise"`
	TrapNumber uint32   `json:"trapNumber"`
}

type V1Group struct {
	Key         string     `json:"key"`
	OID         string     `json:"oid"`
	Name        string     `json:"name"`
	Module      string     `json:"module"`
	Kind        string     `json:"kind"`
	Status      string     `json:"status"`
	Description string     `json:"description"`
	Reference   *string    `json:"reference"`
	Members     []V1ObjRef `json:"members"`
}

type V1Compliance struct {
	Key         string               `json:"key"`
	OID         string               `json:"oid"`
	Name        string               `json:"name"`
	Module      string               `json:"module"`
	Status      string               `json:"status"`
	Description string               `json:"description"`
	Reference   *string              `json:"reference"`
	Modules     []V1ComplianceModule `json:"modules"`
}

type V1ComplianceModule struct {
	Module          string               `json:"module"`
	IsCurrentModule bool                 `json:"isCurrentModule"`
	MandatoryGroups []V1ObjRef           `json:"mandatoryGroups"`
	Groups          []V1ComplianceGroup  `json:"groups"`
	Objects         []V1ComplianceObject `json:"objects"`
}

type V1ComplianceGroup struct {
	Group       V1ObjRef `json:"group"`
	Description *string  `json:"description"`
}

type V1ComplianceObject struct {
	Object      V1ObjRef           `json:"object"`
	Syntax      *V1EffectiveSyntax `json:"syntax"`
	WriteSyntax *V1EffectiveSyntax `json:"writeSyntax"`
	MinAccess   *string            `json:"minAccess"`
	Description *string            `json:"description"`
}

type V1Capability struct {
	Key            string                 `json:"key"`
	OID            string                 `json:"oid"`
	Name           string                 `json:"name"`
	Module         string                 `json:"module"`
	Status         string                 `json:"status"`
	ProductRelease *string                `json:"productRelease"`
	Description    string                 `json:"description"`
	Reference      *string                `json:"reference"`
	Supports       []V1CapabilitySupports `json:"supports"`
}

type V1CapabilitySupports struct {
	Module                 string                    `json:"module"`
	Includes               []V1ObjRef                `json:"includes"`
	ObjectVariations       []V1ObjectVariation       `json:"objectVariations"`
	NotificationVariations []V1NotificationVariation `json:"notificationVariations"`
}

type V1ObjectVariation struct {
	Object           V1ObjRef           `json:"object"`
	Syntax           *V1EffectiveSyntax `json:"syntax"`
	WriteSyntax      *V1EffectiveSyntax `json:"writeSyntax"`
	Access           *string            `json:"access"`
	CreationRequires []V1ObjRef         `json:"creationRequires"`
	DefaultValue     *V1DefaultValue    `json:"defaultValue"`
	Description      *string            `json:"description"`
}

type V1NotificationVariation struct {
	Notification V1ObjRef `json:"notification"`
	Access       *string  `json:"access"`
	Description  *string  `json:"description"`
}

type V1Diagnostic struct {
	Phase    string  `json:"phase"`
	Code     string  `json:"code"`
	Severity string  `json:"severity"`
	Module   *string `json:"module"`
	Line     *int    `json:"line"`
	Column   *int    `json:"column"`
	Message  string  `json:"message"`
}

// v1BaseType maps BaseType to the v1 schema vocabulary.
func v1BaseType(b mib.BaseType) string {
	switch b {
	case mib.BaseUnknown:
		return "Unknown"
	case mib.BaseInteger32:
		return "Integer32"
	case mib.BaseUnsigned32:
		return "Unsigned32"
	case mib.BaseCounter32:
		return "Counter32"
	case mib.BaseCounter64:
		return "Counter64"
	case mib.BaseGauge32:
		return "Gauge32"
	case mib.BaseTimeTicks:
		return "TimeTicks"
	case mib.BaseIpAddress:
		return "IpAddress"
	case mib.BaseOctetString:
		return "OctetString"
	case mib.BaseObjectIdentifier:
		return "ObjectIdentifier"
	case mib.BaseBits:
		return "Bits"
	case mib.BaseOpaque:
		return "Opaque"
	default:
		return "Unknown"
	}
}

// v1Language maps Language to the v1 schema vocabulary.
func v1Language(l mib.Language) *string {
	switch l {
	case mib.LanguageSMIv1:
		return strPtr("SMIv1")
	case mib.LanguageSMIv2:
		return strPtr("SMIv2")
	default:
		return nil
	}
}

// v1Severity maps Severity to the v1 schema vocabulary.
func v1Severity(s mib.Severity) string {
	switch s {
	case mib.SeverityFatal, mib.SeveritySevere, mib.SeverityError:
		return "error"
	case mib.SeverityMinor:
		return "warning"
	case mib.SeverityStyle:
		return "style"
	case mib.SeverityWarning:
		return "warning"
	case mib.SeverityInfo:
		return "info"
	default:
		return "error"
	}
}

// v1Status maps Status to the v1 schema string, or nil for zero-value.
func v1Status(s mib.Status) string {
	return s.String()
}

// diagCodePhase is built once from AllDiagnosticCodes.
var diagCodePhase map[string]string

func init() {
	codes := types.AllDiagnosticCodes()
	diagCodePhase = make(map[string]string, len(codes))
	for _, c := range codes {
		phase := c.Phase
		// Map "lowering" to schema vocabulary "lower"
		if phase == "lowering" {
			phase = "lower"
		}
		diagCodePhase[c.Code] = phase
	}
}

func v1DiagPhase(code string) string {
	if p, ok := diagCodePhase[code]; ok {
		return p
	}
	return "unknown"
}

func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func intPtr(n int) *int {
	if n == 0 {
		return nil
	}
	return &n
}

// isUserDefinedType returns true if the type represents a user-defined named
// type rather than a primitive base type alias from a base module.
// Textual conventions are always user-defined. Non-TC types from base modules
// are schema built-ins and should not be emitted as qualified parent/typeRef
// values in the v1 export.
func isUserDefinedType(t *mib.Type) bool {
	if t.IsTextualConvention() {
		return true
	}
	if t.Module() != nil && t.Module().IsBase() {
		return false
	}
	return true
}

func v1ObjectKind(k mib.Kind) string {
	switch k {
	case mib.KindScalar:
		return "scalar"
	case mib.KindTable:
		return "table"
	case mib.KindRow:
		return "row"
	case mib.KindColumn:
		return "column"
	default:
		return "object"
	}
}

func v1Key(mod, name string) string {
	return mod + "::" + name
}

func v1ObjRef(obj *mib.Object) V1ObjRef {
	ref := V1ObjRef{Name: obj.Name()}
	if obj.Module() != nil {
		ref.Module = obj.Module().Name()
	}
	if obj.OID() != nil {
		ref.OID = obj.OID().String()
	}
	return ref
}

func v1NodeRef(node *mib.Node) V1ObjRef {
	ref := V1ObjRef{Name: node.Name()}
	if node.Module() != nil {
		ref.Module = node.Module().Name()
	}
	if node.OID() != nil {
		ref.OID = node.OID().String()
	}
	return ref
}

func v1Ranges(ranges []mib.Range) []V1Range {
	result := make([]V1Range, 0, len(ranges))
	for _, r := range ranges {
		result = append(result, V1Range{
			Min: strconv.FormatInt(r.Min, 10),
			Max: strconv.FormatInt(r.Max, 10),
		})
	}
	return result
}

func v1Enums(enums []mib.NamedValue) []V1EnumEntry {
	result := make([]V1EnumEntry, 0, len(enums))
	for _, e := range enums {
		result = append(result, V1EnumEntry{Name: e.Label, Value: e.Value})
	}
	return result
}

func v1Bits(bits []mib.NamedValue) []V1BitEntry {
	result := make([]V1BitEntry, 0, len(bits))
	for _, b := range bits {
		result = append(result, V1BitEntry{Name: b.Label, Position: b.Value})
	}
	return result
}

func v1Constraints(sizes, ranges []mib.Range, enums, bits []mib.NamedValue) V1Constraints {
	return V1Constraints{
		Sizes:  v1Ranges(sizes),
		Ranges: v1Ranges(ranges),
		Enums:  v1Enums(enums),
		Bits:   v1Bits(bits),
	}
}

func v1EffectiveSyntax(obj *mib.Object) *V1EffectiveSyntax {
	t := obj.Type()
	if t == nil {
		return nil
	}
	syn := &V1EffectiveSyntax{
		Base:        v1BaseType(t.EffectiveBase()),
		DisplayHint: strPtr(obj.EffectiveDisplayHint()),
		Constraints: v1Constraints(
			obj.EffectiveSizes(),
			obj.EffectiveRanges(),
			obj.EffectiveEnums(),
			obj.EffectiveBits(),
		),
	}
	if t.Name() != "" && isUserDefinedType(t) {
		modName := ""
		if t.Module() != nil {
			modName = t.Module().Name()
		}
		ref := v1Key(modName, t.Name())
		syn.TypeRef = &ref
	}
	return syn
}

func v1SyntaxConstraints(sc *mib.SyntaxConstraints) *V1EffectiveSyntax {
	if sc == nil {
		return nil
	}
	syn := &V1EffectiveSyntax{
		Base:        "Unknown",
		Constraints: v1Constraints(sc.Sizes, sc.Ranges, v1SCEnums(sc.Enums), v1SCBits(sc.Bits)),
	}
	if sc.Type != nil {
		syn.Base = v1BaseType(sc.Type.EffectiveBase())
		syn.DisplayHint = strPtr(sc.Type.EffectiveDisplayHint())
		if sc.Type.Name() != "" && isUserDefinedType(sc.Type) {
			modName := ""
			if sc.Type.Module() != nil {
				modName = sc.Type.Module().Name()
			}
			ref := v1Key(modName, sc.Type.Name())
			syn.TypeRef = &ref
		}
	}
	return syn
}

// v1SCEnums converts NamedValue slice to the types expected by v1Constraints.
func v1SCEnums(nv []mib.NamedValue) []mib.NamedValue { return nv }

// v1SCBits converts NamedValue slice to the types expected by v1Constraints.
func v1SCBits(nv []mib.NamedValue) []mib.NamedValue { return nv }

func v1DefaultValue(dv mib.DefVal) *V1DefaultValue {
	if dv.IsZero() {
		return nil
	}
	result := &V1DefaultValue{
		Kind: dv.Kind().String(),
		Raw:  dv.Raw(),
	}
	switch dv.Kind() { //nolint:exhaustive // DefValKindUnset handled by IsZero check above
	case mib.DefValKindInt:
		v, _ := mib.DefValAs[int64](dv)
		result.Value = strconv.FormatInt(v, 10)
	case mib.DefValKindUint:
		v, _ := mib.DefValAs[uint64](dv)
		result.Value = strconv.FormatUint(v, 10)
	case mib.DefValKindString:
		v, _ := mib.DefValAs[string](dv)
		result.Value = v
	case mib.DefValKindBytes:
		v, _ := mib.DefValAs[[]byte](dv)
		result.Value = strings.ToUpper(hex.EncodeToString(v))
	case mib.DefValKindEnum:
		v, _ := mib.DefValAs[string](dv)
		result.Value = v
	case mib.DefValKindBits:
		v, _ := mib.DefValAs[[]string](dv)
		if v == nil {
			v = []string{}
		}
		result.Value = v
	case mib.DefValKindOID:
		v, _ := mib.DefValAs[mib.OID](dv)
		result.Value = v.String()
	}
	return result
}

func v1IndexEntries(m *mib.Mib, indexes []mib.IndexEntry) []V1IndexEntry {
	result := make([]V1IndexEntry, 0, len(indexes))
	for _, idx := range indexes {
		entry := V1IndexEntry{Implied: idx.Implied}
		if idx.Object != nil {
			ref := v1ObjRef(idx.Object)
			entry.Object = &ref
		} else {
			// Type-backed index - resolve effective syntax from type name
			syn := &V1EffectiveSyntax{
				Base: "Unknown",
				Constraints: V1Constraints{
					Sizes: []V1Range{}, Ranges: []V1Range{},
					Enums: []V1EnumEntry{}, Bits: []V1BitEntry{},
				},
			}
			if t := m.Type(idx.TypeName); t != nil {
				syn.Base = v1BaseType(t.EffectiveBase())
				syn.DisplayHint = strPtr(t.EffectiveDisplayHint())
				if t.Name() != "" {
					modName := ""
					if t.Module() != nil {
						modName = t.Module().Name()
					}
					ref := v1Key(modName, t.Name())
					syn.TypeRef = &ref
					syn.Constraints = v1Constraints(
						t.Sizes(), t.Ranges(), t.Enums(), t.Bits(),
					)
				}
			}
			entry.Syntax = syn
		}
		result = append(result, entry)
	}
	return result
}

// compareOID compares two OID strings numerically.
func compareOID(a, b string) int {
	aParts := strings.Split(a, ".")
	bParts := strings.Split(b, ".")
	minLen := min(len(aParts), len(bParts))
	for i := range minLen {
		ai, _ := strconv.ParseUint(aParts[i], 10, 64)
		bi, _ := strconv.ParseUint(bParts[i], 10, 64)
		if ai != bi {
			if ai < bi {
				return -1
			}
			return 1
		}
	}
	if len(aParts) < len(bParts) {
		return -1
	}
	if len(aParts) > len(bParts) {
		return 1
	}
	return 0
}

func v1ExporterInfo() V1Exporter {
	version := ""
	commit := ""
	if info, ok := debug.ReadBuildInfo(); ok {
		if info.Main.Version != "" && info.Main.Version != "(devel)" {
			version = info.Main.Version
		}
		for _, s := range info.Settings {
			if s.Key == "vcs.revision" {
				commit = s.Value
				break
			}
		}
	}
	return V1Exporter{
		Implementation: "gomib",
		Version:        version,
		Commit:         commit,
	}
}

func buildV1Export(m *mib.Mib, strictness string) *V1Export {
	export := &V1Export{
		SchemaVersion: 1,
		ExportKind:    "resolved-mib",
		Strictness:    strictness,
		Exporter:      v1ExporterInfo(),
		Modules:       make([]V1Module, 0),
		Types:         make([]V1Type, 0),
		Nodes:         make([]V1Node, 0),
		Objects:       make([]V1Object, 0),
		Notifications: make([]V1Notification, 0),
		Groups:        make([]V1Group, 0),
		Compliances:   make([]V1Compliance, 0),
		Capabilities:  make([]V1Capability, 0),
		Diagnostics:   make([]V1Diagnostic, 0),
	}

	// Modules - sorted by name
	for _, mod := range m.Modules() {
		export.Modules = append(export.Modules, buildV1Module(mod))
	}
	sort.Slice(export.Modules, func(i, j int) bool {
		return export.Modules[i].Name < export.Modules[j].Name
	})

	// Types - sorted by module, then name
	for _, typ := range m.Types() {
		export.Types = append(export.Types, buildV1Type(typ))
	}
	sort.Slice(export.Types, func(i, j int) bool {
		if export.Types[i].Module != export.Types[j].Module {
			return export.Types[i].Module < export.Types[j].Module
		}
		return export.Types[i].Name < export.Types[j].Name
	})

	// Nodes - collect KindNode nodes, sorted by module then name
	for node := range m.Nodes() {
		if node.Name() == "" || node.Kind() != mib.KindNode {
			continue
		}
		// Skip nodes that have entities attached (they go in other sections)
		if node.Object() != nil || node.Notification() != nil || node.Group() != nil || node.Compliance() != nil || node.Capability() != nil {
			continue
		}
		export.Nodes = append(export.Nodes, buildV1Node(node, m))
	}
	sort.Slice(export.Nodes, func(i, j int) bool {
		if export.Nodes[i].Module != export.Nodes[j].Module {
			return export.Nodes[i].Module < export.Nodes[j].Module
		}
		return export.Nodes[i].Name < export.Nodes[j].Name
	})

	// Objects - sorted by numeric OID, then module, then name
	for _, obj := range m.Objects() {
		export.Objects = append(export.Objects, buildV1Object(m, obj))
	}
	sort.Slice(export.Objects, func(i, j int) bool {
		c := compareOID(export.Objects[i].OID, export.Objects[j].OID)
		if c != 0 {
			return c < 0
		}
		if export.Objects[i].Module != export.Objects[j].Module {
			return export.Objects[i].Module < export.Objects[j].Module
		}
		return export.Objects[i].Name < export.Objects[j].Name
	})

	// Notifications - sorted by numeric OID, then module, then name
	for _, notif := range m.Notifications() {
		export.Notifications = append(export.Notifications, buildV1Notification(notif, m))
	}
	sort.Slice(export.Notifications, func(i, j int) bool {
		c := compareOID(export.Notifications[i].OID, export.Notifications[j].OID)
		if c != 0 {
			return c < 0
		}
		if export.Notifications[i].Module != export.Notifications[j].Module {
			return export.Notifications[i].Module < export.Notifications[j].Module
		}
		if export.Notifications[i].Name != export.Notifications[j].Name {
			return export.Notifications[i].Name < export.Notifications[j].Name
		}
		// "notification" < "trap" alphabetically, preferring NOTIFICATION-TYPE over TRAP-TYPE
		return export.Notifications[i].Kind < export.Notifications[j].Kind
	})

	// Groups - sorted by numeric OID, then module, then name
	for _, grp := range m.Groups() {
		export.Groups = append(export.Groups, buildV1Group(grp))
	}
	sort.Slice(export.Groups, func(i, j int) bool {
		c := compareOID(export.Groups[i].OID, export.Groups[j].OID)
		if c != 0 {
			return c < 0
		}
		if export.Groups[i].Module != export.Groups[j].Module {
			return export.Groups[i].Module < export.Groups[j].Module
		}
		if export.Groups[i].Name != export.Groups[j].Name {
			return export.Groups[i].Name < export.Groups[j].Name
		}
		// Prefer larger groups first for duplicate keys
		return len(export.Groups[i].Members) > len(export.Groups[j].Members)
	})

	// Compliances
	for _, comp := range m.Compliances() {
		export.Compliances = append(export.Compliances, buildV1Compliance(comp, m))
	}
	sort.Slice(export.Compliances, func(i, j int) bool {
		c := compareOID(export.Compliances[i].OID, export.Compliances[j].OID)
		if c != 0 {
			return c < 0
		}
		if export.Compliances[i].Module != export.Compliances[j].Module {
			return export.Compliances[i].Module < export.Compliances[j].Module
		}
		return export.Compliances[i].Name < export.Compliances[j].Name
	})

	// Capabilities
	for _, cap := range m.Capabilities() {
		export.Capabilities = append(export.Capabilities, buildV1Capability(cap, m))
	}
	sort.Slice(export.Capabilities, func(i, j int) bool {
		c := compareOID(export.Capabilities[i].OID, export.Capabilities[j].OID)
		if c != 0 {
			return c < 0
		}
		if export.Capabilities[i].Module != export.Capabilities[j].Module {
			return export.Capabilities[i].Module < export.Capabilities[j].Module
		}
		return export.Capabilities[i].Name < export.Capabilities[j].Name
	})

	// Diagnostics - sorted canonically
	for _, d := range m.Diagnostics() {
		export.Diagnostics = append(export.Diagnostics, buildV1Diagnostic(d))
	}
	sort.Slice(export.Diagnostics, func(i, j int) bool {
		a, b := export.Diagnostics[i], export.Diagnostics[j]
		if a.Phase != b.Phase {
			return a.Phase < b.Phase
		}
		if a.Code != b.Code {
			return a.Code < b.Code
		}
		if a.Severity != b.Severity {
			return a.Severity < b.Severity
		}
		am, bm := "", ""
		if a.Module != nil {
			am = *a.Module
		}
		if b.Module != nil {
			bm = *b.Module
		}
		if am != bm {
			// null sorts before non-null
			if a.Module == nil {
				return true
			}
			if b.Module == nil {
				return false
			}
			return am < bm
		}
		al, bl := 0, 0
		if a.Line != nil {
			al = *a.Line
		}
		if b.Line != nil {
			bl = *b.Line
		}
		if al != bl {
			if a.Line == nil {
				return true
			}
			if b.Line == nil {
				return false
			}
			return al < bl
		}
		ac, bc := 0, 0
		if a.Column != nil {
			ac = *a.Column
		}
		if b.Column != nil {
			bc = *b.Column
		}
		if ac != bc {
			if a.Column == nil {
				return true
			}
			if b.Column == nil {
				return false
			}
			return ac < bc
		}
		return a.Message < b.Message
	})

	return export
}

func buildV1Module(mod *mib.Module) V1Module {
	m := V1Module{
		Name:      mod.Name(),
		Language:  v1Language(mod.Language()),
		Revisions: make([]V1Revision, 0),
	}
	if oid := mod.OID(); oid != nil {
		s := oid.String()
		m.OID = &s
	}
	m.Organization = strPtr(mod.Organization())
	m.ContactInfo = strPtr(mod.ContactInfo())
	m.Description = strPtr(mod.Description())
	m.LastUpdated = strPtr(mod.LastUpdated())

	for _, rev := range mod.Revisions() {
		m.Revisions = append(m.Revisions, V1Revision{
			Date:        rev.Date,
			Description: strPtr(rev.Description),
		})
	}
	return m
}

func buildV1Type(typ *mib.Type) V1Type {
	modName := ""
	if typ.Module() != nil {
		modName = typ.Module().Name()
	}
	t := V1Type{
		Key:                 v1Key(modName, typ.Name()),
		Name:                typ.Name(),
		Module:              modName,
		Base:                v1BaseType(typ.EffectiveBase()),
		IsTextualConvention: typ.IsTextualConvention(),
		Constraints: v1Constraints(
			typ.Sizes(), typ.Ranges(), typ.Enums(), typ.Bits(),
		),
	}

	if typ.Parent() != nil && typ.Parent().Name() != "" && isUserDefinedType(typ.Parent()) {
		pModName := ""
		if typ.Parent().Module() != nil {
			pModName = typ.Parent().Module().Name()
		}
		ref := v1Key(pModName, typ.Parent().Name())
		t.Parent = &ref
	}

	if typ.Status() != 0 || typ.IsTextualConvention() {
		s := v1Status(typ.Status())
		t.Status = &s
	}

	t.DisplayHint = strPtr(typ.DisplayHint())
	t.Description = strPtr(typ.Description())
	t.Reference = strPtr(typ.Reference())

	return t
}

func buildV1Node(node *mib.Node, _ *mib.Mib) V1Node {
	modName := ""
	if node.Module() != nil {
		modName = node.Module().Name()
	}

	kind := "node"
	if node.IsObjectIdentity() {
		kind = "object-identity"
	}
	// Check for module-identity: module's OID matches this node's OID
	if mod := node.Module(); mod != nil && mod.OID() != nil && slices.Equal(mod.OID(), node.OID()) {
		kind = "module-identity"
	}

	n := V1Node{
		Key:    v1Key(modName, node.Name()),
		OID:    node.OID().String(),
		Name:   node.Name(),
		Module: modName,
		Kind:   kind,
	}

	// Status
	switch kind {
	case "object-identity":
		if st, ok := node.ObjectIdentityStatus(); ok {
			s := v1Status(st)
			n.Status = &s
		}
	case "module-identity":
		// MODULE-IDENTITY does not have STATUS
		n.Status = nil
	default:
		n.Status = nil
	}

	n.Description = strPtr(node.Description())
	n.Reference = strPtr(node.Reference())
	return n
}

func buildV1Object(m *mib.Mib, obj *mib.Object) V1Object {
	modName := ""
	if obj.Module() != nil {
		modName = obj.Module().Name()
	}

	o := V1Object{
		Key:              v1Key(modName, obj.Name()),
		OID:              obj.OID().String(),
		Name:             obj.Name(),
		Module:           modName,
		Kind:             v1ObjectKind(obj.Kind()),
		Status:           v1Status(obj.Status()),
		Access:           obj.Access().String(),
		Description:      strPtr(obj.Description()),
		Reference:        strPtr(obj.Reference()),
		Units:            strPtr(obj.Units()),
		DefaultValue:     v1DefaultValue(obj.DefaultValue()),
		Syntax:           v1EffectiveSyntax(obj),
		Indexes:          make([]V1IndexEntry, 0),
		EffectiveIndexes: make([]V1IndexEntry, 0),
		AugmentedBy:      make([]V1ObjRef, 0),
		Columns:          make([]V1ObjRef, 0),
	}

	// Augments
	if aug := obj.Augments(); aug != nil {
		ref := v1ObjRef(aug)
		o.Augments = &ref
	}

	switch obj.Kind() {
	case mib.KindTable:
		o.Syntax = nil
		o.DefaultValue = nil
		o.Units = nil
		o.Augments = nil
		if entry := obj.Entry(); entry != nil {
			ref := v1ObjRef(entry)
			o.Row = &ref
		}
		for _, col := range obj.Columns() {
			o.Columns = append(o.Columns, v1ObjRef(col))
		}
		sortObjRefs(o.Columns)
	case mib.KindRow:
		if tbl := obj.Table(); tbl != nil {
			ref := v1ObjRef(tbl)
			o.Table = &ref
		}
		o.Indexes = v1IndexEntries(m, obj.Index())
		o.EffectiveIndexes = v1IndexEntries(m, obj.EffectiveIndexes())
		for _, col := range obj.Columns() {
			o.Columns = append(o.Columns, v1ObjRef(col))
		}
		sortObjRefs(o.Columns)
		for _, aug := range obj.AugmentedBy() {
			o.AugmentedBy = append(o.AugmentedBy, v1ObjRef(aug))
		}
		sortObjRefs(o.AugmentedBy)
	case mib.KindColumn:
		if tbl := obj.Table(); tbl != nil {
			ref := v1ObjRef(tbl)
			o.Table = &ref
		}
		if row := obj.Row(); row != nil {
			ref := v1ObjRef(row)
			o.Row = &ref
		}
		o.Augments = nil
		o.AugmentedBy = make([]V1ObjRef, 0)
	case mib.KindScalar:
		// scalar: table, row, augments must be null; columns, augmentedBy must be empty
		o.Table = nil
		o.Row = nil
		o.Augments = nil
	}

	return o
}

func buildV1Notification(notif *mib.Notification, m *mib.Mib) V1Notification {
	modName := ""
	if notif.Module() != nil {
		modName = notif.Module().Name()
	}

	n := V1Notification{
		Key:         v1Key(modName, notif.Name()),
		OID:         notif.OID().String(),
		Name:        notif.Name(),
		Module:      modName,
		Status:      v1Status(notif.Status()),
		Description: notif.Description(),
		Objects:     make([]V1ObjRef, 0),
	}

	if notif.TrapInfo() != nil {
		n.Kind = "trap"
		ti := notif.TrapInfo()
		trap := &V1Trap{
			TrapNumber: ti.TrapNumber,
		}
		// Resolve enterprise
		entNode := m.Resolve(ti.Enterprise)
		if entNode != nil {
			trap.Enterprise = v1NodeRef(entNode)
		} else {
			trap.Enterprise = V1ObjRef{Name: ti.Enterprise}
		}
		n.Trap = trap
	} else {
		n.Kind = "notification"
	}

	if ref := notif.Reference(); ref != "" {
		n.Reference = &ref
	}

	for _, obj := range notif.Objects() {
		n.Objects = append(n.Objects, v1ObjRef(obj))
	}

	return n
}

func buildV1Group(grp *mib.Group) V1Group {
	modName := ""
	if grp.Module() != nil {
		modName = grp.Module().Name()
	}

	g := V1Group{
		Key:         v1Key(modName, grp.Name()),
		OID:         grp.OID().String(),
		Name:        grp.Name(),
		Module:      modName,
		Status:      v1Status(grp.Status()),
		Description: grp.Description(),
		Members:     make([]V1ObjRef, 0),
	}

	if grp.IsNotificationGroup() {
		g.Kind = "notification-group"
	} else {
		g.Kind = "object-group"
	}

	if ref := grp.Reference(); ref != "" {
		g.Reference = &ref
	}

	for _, member := range grp.Members() {
		g.Members = append(g.Members, v1NodeRef(member))
	}

	return g
}

func buildV1Compliance(comp *mib.Compliance, m *mib.Mib) V1Compliance {
	modName := ""
	if comp.Module() != nil {
		modName = comp.Module().Name()
	}

	c := V1Compliance{
		Key:         v1Key(modName, comp.Name()),
		OID:         comp.OID().String(),
		Name:        comp.Name(),
		Module:      modName,
		Status:      v1Status(comp.Status()),
		Description: comp.Description(),
		Modules:     make([]V1ComplianceModule, 0),
	}

	if ref := comp.Reference(); ref != "" {
		c.Reference = &ref
	}

	for _, cm := range comp.Modules() {
		effectiveModName := cm.ModuleName
		if effectiveModName == "" {
			effectiveModName = modName
		}

		vcm := V1ComplianceModule{
			Module:          effectiveModName,
			IsCurrentModule: cm.ModuleName == "",
			MandatoryGroups: make([]V1ObjRef, 0),
			Groups:          make([]V1ComplianceGroup, 0),
			Objects:         make([]V1ComplianceObject, 0),
		}

		for _, grpName := range cm.MandatoryGroups {
			ref := resolveGroupRef(m, effectiveModName, grpName)
			vcm.MandatoryGroups = append(vcm.MandatoryGroups, ref)
		}

		for _, cg := range cm.Groups {
			ref := resolveGroupRef(m, effectiveModName, cg.Group)
			vcg := V1ComplianceGroup{
				Group:       ref,
				Description: strPtr(cg.Description),
			}
			vcm.Groups = append(vcm.Groups, vcg)
		}

		for _, co := range cm.Objects {
			ref := resolveObjRefByName(m, effectiveModName, co.Object)
			vco := V1ComplianceObject{
				Object:      ref,
				Syntax:      v1SyntaxConstraints(co.Syntax),
				WriteSyntax: v1SyntaxConstraints(co.WriteSyntax),
				Description: strPtr(co.Description),
			}
			if co.MinAccess != nil {
				s := co.MinAccess.String()
				vco.MinAccess = &s
			}
			vcm.Objects = append(vcm.Objects, vco)
		}

		c.Modules = append(c.Modules, vcm)
	}

	return c
}

func buildV1Capability(cap *mib.Capability, m *mib.Mib) V1Capability {
	modName := ""
	if cap.Module() != nil {
		modName = cap.Module().Name()
	}

	c := V1Capability{
		Key:         v1Key(modName, cap.Name()),
		OID:         cap.OID().String(),
		Name:        cap.Name(),
		Module:      modName,
		Status:      v1Status(cap.Status()),
		Description: cap.Description(),
		Supports:    make([]V1CapabilitySupports, 0),
	}

	if pr := cap.ProductRelease(); pr != "" {
		c.ProductRelease = &pr
	}
	if ref := cap.Reference(); ref != "" {
		c.Reference = &ref
	}

	for _, s := range cap.Supports() {
		vs := V1CapabilitySupports{
			Module:                 s.ModuleName,
			Includes:               make([]V1ObjRef, 0),
			ObjectVariations:       make([]V1ObjectVariation, 0),
			NotificationVariations: make([]V1NotificationVariation, 0),
		}

		for _, grpName := range s.Includes {
			ref := resolveGroupRef(m, s.ModuleName, grpName)
			vs.Includes = append(vs.Includes, ref)
		}

		for i := range s.ObjectVariations {
			ov := &s.ObjectVariations[i]
			ref := resolveObjRefByName(m, s.ModuleName, ov.Object)
			vov := V1ObjectVariation{
				Object:           ref,
				Syntax:           v1SyntaxConstraints(ov.Syntax),
				WriteSyntax:      v1SyntaxConstraints(ov.WriteSyntax),
				DefaultValue:     v1DefaultValue(ov.DefVal),
				Description:      strPtr(ov.Description),
				CreationRequires: make([]V1ObjRef, 0),
			}
			if ov.Access != nil {
				a := ov.Access.String()
				vov.Access = &a
			}
			for _, cr := range ov.CreationRequires {
				crRef := resolveObjRefByName(m, s.ModuleName, cr)
				vov.CreationRequires = append(vov.CreationRequires, crRef)
			}
			vs.ObjectVariations = append(vs.ObjectVariations, vov)
		}

		for _, nv := range s.NotificationVariations {
			ref := resolveNotifRefByName(m, s.ModuleName, nv.Notification)
			vnv := V1NotificationVariation{
				Notification: ref,
				Description:  strPtr(nv.Description),
			}
			if nv.Access != nil {
				a := nv.Access.String()
				vnv.Access = &a
			}
			vs.NotificationVariations = append(vs.NotificationVariations, vnv)
		}

		c.Supports = append(c.Supports, vs)
	}

	return c
}

func buildV1Diagnostic(d mib.Diagnostic) V1Diagnostic {
	return V1Diagnostic{
		Phase:    v1DiagPhase(d.Code),
		Code:     d.Code,
		Severity: v1Severity(d.Severity),
		Module:   strPtr(d.Module),
		Line:     intPtr(d.Line),
		Column:   intPtr(d.Column),
		Message:  d.Message,
	}
}

// resolveGroupRef looks up a group by name and returns an ObjRef.
func resolveGroupRef(m *mib.Mib, moduleName, groupName string) V1ObjRef {
	qualified := moduleName + "::" + groupName
	node := m.Resolve(qualified)
	if node == nil {
		node = m.Resolve(groupName)
	}
	if node != nil {
		return v1NodeRef(node)
	}
	return V1ObjRef{Name: groupName, Module: moduleName}
}

// resolveObjRefByName looks up an object by name and returns an ObjRef.
func resolveObjRefByName(m *mib.Mib, moduleName, objName string) V1ObjRef {
	qualified := moduleName + "::" + objName
	node := m.Resolve(qualified)
	if node == nil {
		node = m.Resolve(objName)
	}
	if node != nil {
		return v1NodeRef(node)
	}
	return V1ObjRef{Name: objName, Module: moduleName}
}

// resolveNotifRefByName looks up a notification by name and returns an ObjRef.
func resolveNotifRefByName(m *mib.Mib, moduleName, notifName string) V1ObjRef {
	qualified := moduleName + "::" + notifName
	node := m.Resolve(qualified)
	if node == nil {
		node = m.Resolve(notifName)
	}
	if node != nil {
		return v1NodeRef(node)
	}
	return V1ObjRef{Name: notifName, Module: moduleName}
}

func sortObjRefs(refs []V1ObjRef) {
	sort.Slice(refs, func(i, j int) bool {
		c := compareOID(refs[i].OID, refs[j].OID)
		if c != 0 {
			return c < 0
		}
		if refs[i].Module != refs[j].Module {
			return refs[i].Module < refs[j].Module
		}
		return refs[i].Name < refs[j].Name
	})
}
