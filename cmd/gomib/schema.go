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

// Export schema JSON types.

type ExportPayload struct {
	SchemaVersion int                  `json:"schemaVersion"`
	ExportKind    string               `json:"exportKind"`
	Strictness    string               `json:"strictness"`
	Exporter      Exporter             `json:"exporter"`
	Modules       []ExportModule       `json:"modules"`
	Types         []ExportType         `json:"types"`
	Nodes         []ExportNode         `json:"nodes"`
	Objects       []ExportObject       `json:"objects"`
	Notifications []ExportNotification `json:"notifications"`
	Groups        []ExportGroup        `json:"groups"`
	Compliances   []ExportCompliance   `json:"compliances"`
	Capabilities  []ExportCapability   `json:"capabilities"`
	Diagnostics   []ExportDiagnostic   `json:"diagnostics"`
}

type Exporter struct {
	Implementation string `json:"implementation"`
	Version        string `json:"version"`
	Commit         string `json:"commit"`
}

type ExportModule struct {
	Name         string           `json:"name"`
	OID          *string          `json:"oid"`
	Language     *string          `json:"language"`
	Organization *string          `json:"organization"`
	ContactInfo  *string          `json:"contactInfo"`
	Description  *string          `json:"description"`
	LastUpdated  *string          `json:"lastUpdated"`
	Revisions    []ExportRevision `json:"revisions"`
}

type ExportRevision struct {
	Date        string  `json:"date"`
	Description *string `json:"description"`
}

type ExportType struct {
	Key                 string            `json:"key"`
	Name                string            `json:"name"`
	Module              string            `json:"module"`
	Parent              *string           `json:"parent"`
	Base                string            `json:"base"`
	Status              *string           `json:"status"`
	DisplayHint         *string           `json:"displayHint"`
	Description         *string           `json:"description"`
	Reference           *string           `json:"reference"`
	IsTextualConvention bool              `json:"isTextualConvention"`
	Constraints         ExportConstraints `json:"constraints"`
}

type ExportConstraints struct {
	Sizes  []ExportRange     `json:"sizes"`
	Ranges []ExportRange     `json:"ranges"`
	Enums  []ExportEnumEntry `json:"enums"`
	Bits   []ExportBitEntry  `json:"bits"`
}

type ExportRange struct {
	Min string `json:"min"`
	Max string `json:"max"`
}

type ExportEnumEntry struct {
	Name  string `json:"name"`
	Value int64  `json:"value"`
}

type ExportBitEntry struct {
	Name     string `json:"name"`
	Position int64  `json:"position"`
}

type ExportNode struct {
	Key         string  `json:"key"`
	OID         string  `json:"oid"`
	Name        string  `json:"name"`
	Module      string  `json:"module"`
	Kind        string  `json:"kind"`
	Status      *string `json:"status"`
	Description *string `json:"description"`
	Reference   *string `json:"reference"`
}

type ExportObject struct {
	Key              string                 `json:"key"`
	OID              string                 `json:"oid"`
	Name             string                 `json:"name"`
	Module           string                 `json:"module"`
	Kind             string                 `json:"kind"`
	Status           string                 `json:"status"`
	Access           string                 `json:"access"`
	Description      *string                `json:"description"`
	Reference        *string                `json:"reference"`
	Units            *string                `json:"units"`
	DefaultValue     *ExportDefaultValue    `json:"defaultValue"`
	Syntax           *ExportEffectiveSyntax `json:"syntax"`
	Indexes          []ExportIndexEntry     `json:"indexes"`
	EffectiveIndexes []ExportIndexEntry     `json:"effectiveIndexes"`
	Augments         *ExportObjRef          `json:"augments"`
	AugmentedBy      []ExportObjRef         `json:"augmentedBy"`
	Table            *ExportObjRef          `json:"table"`
	Row              *ExportObjRef          `json:"row"`
	Columns          []ExportObjRef         `json:"columns"`
}

type ExportEffectiveSyntax struct {
	TypeRef     *string           `json:"typeRef"`
	Base        string            `json:"base"`
	DisplayHint *string           `json:"displayHint"`
	Constraints ExportConstraints `json:"constraints"`
}

type ExportObjRef struct {
	Name   string `json:"name"`
	Module string `json:"module"`
	OID    string `json:"oid"`
}

type ExportIndexEntry struct {
	Object  *ExportObjRef          `json:"object"`
	Implied bool                   `json:"implied"`
	Syntax  *ExportEffectiveSyntax `json:"syntax"`
}

type ExportDefaultValue struct {
	Kind  string `json:"kind"`
	Value any    `json:"value"`
	Raw   string `json:"raw"`
}

type ExportNotification struct {
	Key         string         `json:"key"`
	OID         string         `json:"oid"`
	Name        string         `json:"name"`
	Module      string         `json:"module"`
	Status      string         `json:"status"`
	Kind        string         `json:"kind"`
	Trap        *ExportTrap    `json:"trap"`
	Description *string        `json:"description"`
	Reference   *string        `json:"reference"`
	Objects     []ExportObjRef `json:"objects"`
}

type ExportTrap struct {
	Enterprise ExportObjRef `json:"enterprise"`
	TrapNumber uint32       `json:"trapNumber"`
}

type ExportGroup struct {
	Key         string         `json:"key"`
	OID         string         `json:"oid"`
	Name        string         `json:"name"`
	Module      string         `json:"module"`
	Kind        string         `json:"kind"`
	Status      string         `json:"status"`
	Description *string        `json:"description"`
	Reference   *string        `json:"reference"`
	Members     []ExportObjRef `json:"members"`
}

type ExportCompliance struct {
	Key         string                   `json:"key"`
	OID         string                   `json:"oid"`
	Name        string                   `json:"name"`
	Module      string                   `json:"module"`
	Status      string                   `json:"status"`
	Description *string                  `json:"description"`
	Reference   *string                  `json:"reference"`
	Modules     []ExportComplianceModule `json:"modules"`
}

type ExportComplianceModule struct {
	Module          string                   `json:"module"`
	IsCurrentModule bool                     `json:"isCurrentModule"`
	MandatoryGroups []ExportObjRef           `json:"mandatoryGroups"`
	Groups          []ExportComplianceGroup  `json:"groups"`
	Objects         []ExportComplianceObject `json:"objects"`
}

type ExportComplianceGroup struct {
	Group       ExportObjRef `json:"group"`
	Description *string      `json:"description"`
}

type ExportComplianceObject struct {
	Object      ExportObjRef           `json:"object"`
	Syntax      *ExportEffectiveSyntax `json:"syntax"`
	WriteSyntax *ExportEffectiveSyntax `json:"writeSyntax"`
	MinAccess   *string                `json:"minAccess"`
	Description *string                `json:"description"`
}

type ExportCapability struct {
	Key            string                     `json:"key"`
	OID            string                     `json:"oid"`
	Name           string                     `json:"name"`
	Module         string                     `json:"module"`
	Status         string                     `json:"status"`
	ProductRelease *string                    `json:"productRelease"`
	Description    *string                    `json:"description"`
	Reference      *string                    `json:"reference"`
	Supports       []ExportCapabilitySupports `json:"supports"`
}

type ExportCapabilitySupports struct {
	Module                 string                        `json:"module"`
	Includes               []ExportObjRef                `json:"includes"`
	ObjectVariations       []ExportObjectVariation       `json:"objectVariations"`
	NotificationVariations []ExportNotificationVariation `json:"notificationVariations"`
}

type ExportObjectVariation struct {
	Object           ExportObjRef           `json:"object"`
	Syntax           *ExportEffectiveSyntax `json:"syntax"`
	WriteSyntax      *ExportEffectiveSyntax `json:"writeSyntax"`
	Access           *string                `json:"access"`
	CreationRequires []ExportObjRef         `json:"creationRequires"`
	DefaultValue     *ExportDefaultValue    `json:"defaultValue"`
	Description      *string                `json:"description"`
}

type ExportNotificationVariation struct {
	Notification ExportObjRef `json:"notification"`
	Access       *string      `json:"access"`
	Description  *string      `json:"description"`
}

type ExportDiagnostic struct {
	Phase    string  `json:"phase"`
	Code     string  `json:"code"`
	Severity string  `json:"severity"`
	Module   *string `json:"module"`
	Line     *int    `json:"line"`
	Column   *int    `json:"column"`
	Message  string  `json:"message"`
}

// exportBaseType maps BaseType to the export schema vocabulary.
func exportBaseType(b mib.BaseType) string {
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
	case mib.BaseSequence:
		return "Sequence"
	default:
		return "Unknown"
	}
}

// exportLanguage maps Language to the export schema vocabulary.
func exportLanguage(l mib.Language) *string {
	switch l {
	case mib.LanguageSMIv1:
		return strPtr("SMIv1")
	case mib.LanguageSMIv2:
		return strPtr("SMIv2")
	default:
		return nil
	}
}

// exportSeverity maps Severity to the export schema vocabulary.
func exportSeverity(s mib.Severity) string {
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

// exportStatus maps Status to the export schema string, or nil for zero-value.
func exportStatus(s mib.Status) string {
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

func exportDiagPhase(code string) string {
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
// values in the export.
func isUserDefinedType(t *mib.Type) bool {
	if t.IsTextualConvention() {
		return true
	}
	if t.Module() != nil && t.Module().IsBase() {
		return false
	}
	return true
}

func exportObjectKind(k mib.Kind) string {
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

func exportKey(mod, name string) string {
	return mod + "::" + name
}

func exportObjRef(obj *mib.Object) ExportObjRef {
	ref := ExportObjRef{Name: obj.Name()}
	if obj.Module() != nil {
		ref.Module = obj.Module().Name()
	}
	if obj.OID() != nil {
		ref.OID = obj.OID().String()
	}
	return ref
}

func exportNodeRef(node *mib.Node) ExportObjRef {
	ref := ExportObjRef{Name: node.Name()}
	if node.Module() != nil {
		ref.Module = node.Module().Name()
	}
	if node.OID() != nil {
		ref.OID = node.OID().String()
	}
	return ref
}

func exportRanges(ranges []mib.Range) []ExportRange {
	result := make([]ExportRange, 0, len(ranges))
	for _, r := range ranges {
		result = append(result, ExportRange{
			Min: strconv.FormatInt(r.Min, 10),
			Max: strconv.FormatInt(r.Max, 10),
		})
	}
	return result
}

func exportEnums(enums []mib.NamedValue) []ExportEnumEntry {
	result := make([]ExportEnumEntry, 0, len(enums))
	for _, e := range enums {
		result = append(result, ExportEnumEntry{Name: e.Label, Value: e.Value})
	}
	return result
}

func exportBits(bits []mib.NamedValue) []ExportBitEntry {
	result := make([]ExportBitEntry, 0, len(bits))
	for _, b := range bits {
		result = append(result, ExportBitEntry{Name: b.Label, Position: b.Value})
	}
	return result
}

func exportConstraints(sizes, ranges []mib.Range, enums, bits []mib.NamedValue) ExportConstraints {
	return ExportConstraints{
		Sizes:  exportRanges(sizes),
		Ranges: exportRanges(ranges),
		Enums:  exportEnums(enums),
		Bits:   exportBits(bits),
	}
}

func exportEffectiveSyntax(obj *mib.Object) *ExportEffectiveSyntax {
	t := obj.Type()
	if t == nil {
		return nil
	}
	syn := &ExportEffectiveSyntax{
		Base:        exportBaseType(t.EffectiveBase()),
		DisplayHint: strPtr(obj.EffectiveDisplayHint()),
		Constraints: exportConstraints(
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
		ref := exportKey(modName, t.Name())
		syn.TypeRef = &ref
	}
	return syn
}

func exportSyntaxConstraints(sc *mib.SyntaxConstraints) *ExportEffectiveSyntax {
	if sc == nil {
		return nil
	}
	syn := &ExportEffectiveSyntax{
		Base:        "Unknown",
		Constraints: exportConstraints(sc.Sizes, sc.Ranges, sc.Enums, sc.Bits),
	}
	if sc.Type != nil {
		syn.Base = exportBaseType(sc.Type.EffectiveBase())
		syn.DisplayHint = strPtr(sc.Type.EffectiveDisplayHint())
		if sc.Type.Name() != "" && isUserDefinedType(sc.Type) {
			modName := ""
			if sc.Type.Module() != nil {
				modName = sc.Type.Module().Name()
			}
			ref := exportKey(modName, sc.Type.Name())
			syn.TypeRef = &ref
		}
	}
	return syn
}

func exportDefaultValue(dv mib.DefVal) *ExportDefaultValue {
	if dv.IsZero() {
		return nil
	}
	result := &ExportDefaultValue{
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

func exportIndexEntries(m *mib.Mib, indexes []mib.IndexEntry) []ExportIndexEntry {
	result := make([]ExportIndexEntry, 0, len(indexes))
	for _, idx := range indexes {
		entry := ExportIndexEntry{Implied: idx.Implied}
		if idx.Object != nil {
			ref := exportObjRef(idx.Object)
			entry.Object = &ref
		} else {
			// Type-backed index - resolve effective syntax from type name
			syn := &ExportEffectiveSyntax{
				Base: "Unknown",
				Constraints: ExportConstraints{
					Sizes: []ExportRange{}, Ranges: []ExportRange{},
					Enums: []ExportEnumEntry{}, Bits: []ExportBitEntry{},
				},
			}
			if t := m.Type(idx.TypeName); t != nil {
				syn.Base = exportBaseType(t.EffectiveBase())
				syn.DisplayHint = strPtr(t.EffectiveDisplayHint())
				if t.Name() != "" {
					modName := ""
					if t.Module() != nil {
						modName = t.Module().Name()
					}
					ref := exportKey(modName, t.Name())
					syn.TypeRef = &ref
					syn.Constraints = exportConstraints(
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

func exporterInfo() Exporter {
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
	return Exporter{
		Implementation: "gomib",
		Version:        version,
		Commit:         commit,
	}
}

func buildExportPayload(m *mib.Mib, strictness string) *ExportPayload {
	export := &ExportPayload{
		SchemaVersion: 1,
		ExportKind:    "resolved-mib",
		Strictness:    strictness,
		Exporter:      exporterInfo(),
		Modules:       make([]ExportModule, 0),
		Types:         make([]ExportType, 0),
		Nodes:         make([]ExportNode, 0),
		Objects:       make([]ExportObject, 0),
		Notifications: make([]ExportNotification, 0),
		Groups:        make([]ExportGroup, 0),
		Compliances:   make([]ExportCompliance, 0),
		Capabilities:  make([]ExportCapability, 0),
		Diagnostics:   make([]ExportDiagnostic, 0),
	}

	// Modules - sorted by name
	for _, mod := range m.Modules() {
		export.Modules = append(export.Modules, buildExportModule(mod))
	}
	sort.Slice(export.Modules, func(i, j int) bool {
		return export.Modules[i].Name < export.Modules[j].Name
	})

	// Types - sorted by module, then name
	for _, typ := range m.Types() {
		export.Types = append(export.Types, buildExportType(typ))
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
		export.Nodes = append(export.Nodes, buildExportNode(node, m))
	}
	sort.Slice(export.Nodes, func(i, j int) bool {
		if export.Nodes[i].Module != export.Nodes[j].Module {
			return export.Nodes[i].Module < export.Nodes[j].Module
		}
		return export.Nodes[i].Name < export.Nodes[j].Name
	})

	// Objects - sorted by numeric OID, then module, then name
	for _, obj := range m.Objects() {
		export.Objects = append(export.Objects, buildExportObject(m, obj))
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
		export.Notifications = append(export.Notifications, buildExportNotification(notif, m))
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
		export.Groups = append(export.Groups, buildExportGroup(grp))
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
		export.Compliances = append(export.Compliances, buildExportCompliance(comp, m))
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
		export.Capabilities = append(export.Capabilities, buildExportCapability(cap, m))
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
		export.Diagnostics = append(export.Diagnostics, buildExportDiagnostic(d))
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

func buildExportModule(mod *mib.Module) ExportModule {
	m := ExportModule{
		Name:      mod.Name(),
		Language:  exportLanguage(mod.Language()),
		Revisions: make([]ExportRevision, 0),
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
		m.Revisions = append(m.Revisions, ExportRevision{
			Date:        rev.Date,
			Description: strPtr(rev.Description),
		})
	}
	return m
}

func buildExportType(typ *mib.Type) ExportType {
	modName := ""
	if typ.Module() != nil {
		modName = typ.Module().Name()
	}
	t := ExportType{
		Key:                 exportKey(modName, typ.Name()),
		Name:                typ.Name(),
		Module:              modName,
		Base:                exportBaseType(typ.EffectiveBase()),
		IsTextualConvention: typ.IsTextualConvention(),
		Constraints: exportConstraints(
			typ.Sizes(), typ.Ranges(), typ.Enums(), typ.Bits(),
		),
	}

	if typ.Parent() != nil && typ.Parent().Name() != "" && isUserDefinedType(typ.Parent()) {
		pModName := ""
		if typ.Parent().Module() != nil {
			pModName = typ.Parent().Module().Name()
		}
		ref := exportKey(pModName, typ.Parent().Name())
		t.Parent = &ref
	}

	if typ.Status() != 0 || typ.IsTextualConvention() {
		s := exportStatus(typ.Status())
		t.Status = &s
	}

	t.DisplayHint = strPtr(typ.DisplayHint())
	t.Description = strPtr(typ.Description())
	t.Reference = strPtr(typ.Reference())

	return t
}

func buildExportNode(node *mib.Node, _ *mib.Mib) ExportNode {
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

	n := ExportNode{
		Key:    exportKey(modName, node.Name()),
		OID:    node.OID().String(),
		Name:   node.Name(),
		Module: modName,
		Kind:   kind,
	}

	// Status
	switch kind {
	case "object-identity":
		if st, ok := node.ObjectIdentityStatus(); ok {
			s := exportStatus(st)
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

func buildExportObject(m *mib.Mib, obj *mib.Object) ExportObject {
	modName := ""
	if obj.Module() != nil {
		modName = obj.Module().Name()
	}

	o := ExportObject{
		Key:              exportKey(modName, obj.Name()),
		OID:              obj.OID().String(),
		Name:             obj.Name(),
		Module:           modName,
		Kind:             exportObjectKind(obj.Kind()),
		Status:           exportStatus(obj.Status()),
		Access:           obj.Access().String(),
		Description:      strPtr(obj.Description()),
		Reference:        strPtr(obj.Reference()),
		Units:            strPtr(obj.Units()),
		DefaultValue:     exportDefaultValue(obj.DefaultValue()),
		Syntax:           exportEffectiveSyntax(obj),
		Indexes:          make([]ExportIndexEntry, 0),
		EffectiveIndexes: make([]ExportIndexEntry, 0),
		AugmentedBy:      make([]ExportObjRef, 0),
		Columns:          make([]ExportObjRef, 0),
	}

	// Augments
	if aug := obj.Augments(); aug != nil {
		ref := exportObjRef(aug)
		o.Augments = &ref
	}

	switch obj.Kind() {
	case mib.KindTable:
		o.Syntax = nil
		o.DefaultValue = nil
		o.Units = nil
		o.Augments = nil
		if entry := obj.Entry(); entry != nil {
			ref := exportObjRef(entry)
			o.Row = &ref
		}
		for _, col := range obj.Columns() {
			o.Columns = append(o.Columns, exportObjRef(col))
		}
		sortObjRefs(o.Columns)
	case mib.KindRow:
		if tbl := obj.Table(); tbl != nil {
			ref := exportObjRef(tbl)
			o.Table = &ref
		}
		o.Indexes = exportIndexEntries(m, obj.Index())
		o.EffectiveIndexes = exportIndexEntries(m, obj.EffectiveIndexes())
		for _, col := range obj.Columns() {
			o.Columns = append(o.Columns, exportObjRef(col))
		}
		sortObjRefs(o.Columns)
		for _, aug := range obj.AugmentedBy() {
			o.AugmentedBy = append(o.AugmentedBy, exportObjRef(aug))
		}
		sortObjRefs(o.AugmentedBy)
	case mib.KindColumn:
		if tbl := obj.Table(); tbl != nil {
			ref := exportObjRef(tbl)
			o.Table = &ref
		}
		if row := obj.Row(); row != nil {
			ref := exportObjRef(row)
			o.Row = &ref
		}
		o.Augments = nil
		o.AugmentedBy = make([]ExportObjRef, 0)
	case mib.KindScalar:
		// scalar: table, row, augments must be null; columns, augmentedBy must be empty
		o.Table = nil
		o.Row = nil
		o.Augments = nil
	}

	return o
}

func buildExportNotification(notif *mib.Notification, m *mib.Mib) ExportNotification {
	modName := ""
	if notif.Module() != nil {
		modName = notif.Module().Name()
	}

	n := ExportNotification{
		Key:         exportKey(modName, notif.Name()),
		OID:         notif.OID().String(),
		Name:        notif.Name(),
		Module:      modName,
		Status:      exportStatus(notif.Status()),
		Description: strPtr(notif.Description()),
		Objects:     make([]ExportObjRef, 0),
	}

	if notif.TrapInfo() != nil {
		n.Kind = "trap"
		ti := notif.TrapInfo()
		trap := &ExportTrap{
			TrapNumber: ti.TrapNumber,
		}
		// Resolve enterprise
		entNode := m.Resolve(ti.Enterprise)
		if entNode != nil {
			trap.Enterprise = exportNodeRef(entNode)
		} else {
			trap.Enterprise = ExportObjRef{Name: ti.Enterprise}
		}
		n.Trap = trap
	} else {
		n.Kind = "notification"
	}

	if ref := notif.Reference(); ref != "" {
		n.Reference = &ref
	}

	for _, obj := range notif.Objects() {
		n.Objects = append(n.Objects, exportObjRef(obj))
	}

	return n
}

func buildExportGroup(grp *mib.Group) ExportGroup {
	modName := ""
	if grp.Module() != nil {
		modName = grp.Module().Name()
	}

	g := ExportGroup{
		Key:         exportKey(modName, grp.Name()),
		OID:         grp.OID().String(),
		Name:        grp.Name(),
		Module:      modName,
		Status:      exportStatus(grp.Status()),
		Description: strPtr(grp.Description()),
		Members:     make([]ExportObjRef, 0),
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
		g.Members = append(g.Members, exportNodeRef(member))
	}

	return g
}

func buildExportCompliance(comp *mib.Compliance, m *mib.Mib) ExportCompliance {
	modName := ""
	if comp.Module() != nil {
		modName = comp.Module().Name()
	}

	c := ExportCompliance{
		Key:         exportKey(modName, comp.Name()),
		OID:         comp.OID().String(),
		Name:        comp.Name(),
		Module:      modName,
		Status:      exportStatus(comp.Status()),
		Description: strPtr(comp.Description()),
		Modules:     make([]ExportComplianceModule, 0),
	}

	if ref := comp.Reference(); ref != "" {
		c.Reference = &ref
	}

	for _, cm := range comp.Modules() {
		effectiveModName := cm.ModuleName
		if effectiveModName == "" {
			effectiveModName = modName
		}

		vcm := ExportComplianceModule{
			Module:          effectiveModName,
			IsCurrentModule: cm.ModuleName == "",
			MandatoryGroups: make([]ExportObjRef, 0),
			Groups:          make([]ExportComplianceGroup, 0),
			Objects:         make([]ExportComplianceObject, 0),
		}

		for _, grpName := range cm.MandatoryGroups {
			ref := resolveNamedRef(m, effectiveModName, grpName)
			vcm.MandatoryGroups = append(vcm.MandatoryGroups, ref)
		}

		for _, cg := range cm.Groups {
			ref := resolveNamedRef(m, effectiveModName, cg.Group)
			vcg := ExportComplianceGroup{
				Group:       ref,
				Description: strPtr(cg.Description),
			}
			vcm.Groups = append(vcm.Groups, vcg)
		}

		for _, co := range cm.Objects {
			ref := resolveNamedRef(m, effectiveModName, co.Object)
			vco := ExportComplianceObject{
				Object:      ref,
				Syntax:      exportSyntaxConstraints(co.Syntax),
				WriteSyntax: exportSyntaxConstraints(co.WriteSyntax),
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

func buildExportCapability(cap *mib.Capability, m *mib.Mib) ExportCapability {
	modName := ""
	if cap.Module() != nil {
		modName = cap.Module().Name()
	}

	c := ExportCapability{
		Key:         exportKey(modName, cap.Name()),
		OID:         cap.OID().String(),
		Name:        cap.Name(),
		Module:      modName,
		Status:      exportStatus(cap.Status()),
		Description: strPtr(cap.Description()),
		Supports:    make([]ExportCapabilitySupports, 0),
	}

	if pr := cap.ProductRelease(); pr != "" {
		c.ProductRelease = &pr
	}
	if ref := cap.Reference(); ref != "" {
		c.Reference = &ref
	}

	for _, s := range cap.Supports() {
		vs := ExportCapabilitySupports{
			Module:                 s.ModuleName,
			Includes:               make([]ExportObjRef, 0),
			ObjectVariations:       make([]ExportObjectVariation, 0),
			NotificationVariations: make([]ExportNotificationVariation, 0),
		}

		for _, grpName := range s.Includes {
			ref := resolveNamedRef(m, s.ModuleName, grpName)
			vs.Includes = append(vs.Includes, ref)
		}

		for i := range s.ObjectVariations {
			ov := &s.ObjectVariations[i]
			ref := resolveNamedRef(m, s.ModuleName, ov.Object)
			vov := ExportObjectVariation{
				Object:           ref,
				Syntax:           exportSyntaxConstraints(ov.Syntax),
				WriteSyntax:      exportSyntaxConstraints(ov.WriteSyntax),
				DefaultValue:     exportDefaultValue(ov.DefVal),
				Description:      strPtr(ov.Description),
				CreationRequires: make([]ExportObjRef, 0),
			}
			if ov.Access != nil {
				a := ov.Access.String()
				vov.Access = &a
			}
			for _, cr := range ov.CreationRequires {
				crRef := resolveNamedRef(m, s.ModuleName, cr)
				vov.CreationRequires = append(vov.CreationRequires, crRef)
			}
			vs.ObjectVariations = append(vs.ObjectVariations, vov)
		}

		for _, nv := range s.NotificationVariations {
			ref := resolveNamedRef(m, s.ModuleName, nv.Notification)
			vnv := ExportNotificationVariation{
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

func buildExportDiagnostic(d mib.Diagnostic) ExportDiagnostic {
	return ExportDiagnostic{
		Phase:    exportDiagPhase(d.Code),
		Code:     d.Code,
		Severity: exportSeverity(d.Severity),
		Module:   strPtr(d.Module),
		Line:     intPtr(d.Line),
		Column:   intPtr(d.Column),
		Message:  d.Message,
	}
}

// resolveNamedRef looks up a node by name (qualified then unqualified) and returns an ObjRef.
func resolveNamedRef(m *mib.Mib, moduleName, name string) ExportObjRef {
	qualified := moduleName + "::" + name
	node := m.Resolve(qualified)
	if node == nil {
		node = m.Resolve(name)
	}
	if node != nil {
		return exportNodeRef(node)
	}
	return ExportObjRef{Name: name, Module: moduleName}
}

func sortObjRefs(refs []ExportObjRef) {
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
