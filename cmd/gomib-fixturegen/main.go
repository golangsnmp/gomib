// gomib-fixturegen generates NormalizedNode JSON fixtures from gomib
// for cross-validation with mib-rs. No CGO required.
//
// Modes:
//
//	stdout mode (no -outdir): write all nodes as a single JSON object to stdout
//	file mode (-outdir DIR):  write per-module JSON files to DIR
//
// When no MODULE args are given, all modules from the corpus are loaded.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/golangsnmp/gomib"
	"github.com/golangsnmp/gomib/mib"
)

// NormalizedNode matches the fixture schema used by mib-rs tests.
type NormalizedNode struct {
	OID                string                       `json:"OID"`
	Name               string                       `json:"Name"`
	Module             string                       `json:"Module"`
	Description        string                       `json:"Description"`
	Type               string                       `json:"Type"`
	Access             string                       `json:"Access"`
	Status             string                       `json:"Status"`
	Hint               string                       `json:"Hint"`
	TCName             string                       `json:"TCName"`
	Units              string                       `json:"Units"`
	EnumValues         map[int]string               `json:"EnumValues"`
	Indexes            []IndexInfo                  `json:"Indexes"`
	Augments           string                       `json:"Augments"`
	Ranges             []RangeInfo                  `json:"Ranges"`
	DefaultValue       string                       `json:"DefaultValue"`
	Kind               string                       `json:"Kind"`
	Varbinds           []string                     `json:"Varbinds"`
	GroupMembers       []string                     `json:"GroupMembers"`
	NodeType           string                       `json:"NodeType"`
	BitValues          map[int]string               `json:"BitValues"`
	Reference          string                       `json:"Reference"`
	ProductRelease     string                       `json:"ProductRelease"`
	ComplianceModules  []NormalizedComplianceModule `json:"ComplianceModules,omitempty"`
	CapabilitySupports []NormalizedCapabilityModule `json:"CapabilitySupports,omitempty"`
}

type NormalizedComplianceModule struct {
	ModuleName      string                       `json:"ModuleName"`
	MandatoryGroups []string                     `json:"MandatoryGroups,omitempty"`
	Groups          []NormalizedComplianceGroup  `json:"Groups,omitempty"`
	Objects         []NormalizedComplianceObject `json:"Objects,omitempty"`
}

type NormalizedComplianceGroup struct {
	Group       string `json:"Group"`
	Description string `json:"Description"`
}

type NormalizedComplianceObject struct {
	Object      string                       `json:"Object"`
	Syntax      *NormalizedSyntaxConstraints `json:"Syntax,omitempty"`
	WriteSyntax *NormalizedSyntaxConstraints `json:"WriteSyntax,omitempty"`
	MinAccess   string                       `json:"MinAccess"`
	Description string                       `json:"Description"`
}

type NormalizedCapabilityModule struct {
	ModuleName             string                            `json:"ModuleName"`
	Includes               []string                          `json:"Includes,omitempty"`
	ObjectVariations       []NormalizedObjectVariation       `json:"ObjectVariations,omitempty"`
	NotificationVariations []NormalizedNotificationVariation `json:"NotificationVariations,omitempty"`
}

type NormalizedObjectVariation struct {
	Object           string                       `json:"Object"`
	Syntax           *NormalizedSyntaxConstraints `json:"Syntax,omitempty"`
	WriteSyntax      *NormalizedSyntaxConstraints `json:"WriteSyntax,omitempty"`
	Access           string                       `json:"Access"`
	CreationRequires []string                     `json:"CreationRequires,omitempty"`
	DefaultValue     string                       `json:"DefaultValue"`
	Description      string                       `json:"Description"`
}

type NormalizedNotificationVariation struct {
	Notification string `json:"Notification"`
	Access       string `json:"Access"`
	Description  string `json:"Description"`
}

type NormalizedSyntaxConstraints struct {
	TypeName string         `json:"TypeName"`
	Sizes    []RangeInfo    `json:"Sizes,omitempty"`
	Ranges   []RangeInfo    `json:"Ranges,omitempty"`
	Enums    map[int]string `json:"Enums,omitempty"`
	Bits     map[int]string `json:"Bits,omitempty"`
}

type NormalizedModule struct {
	Name         string         `json:"Name"`
	OID          string         `json:"OID"`
	Organization string         `json:"Organization"`
	ContactInfo  string         `json:"ContactInfo"`
	Description  string         `json:"Description"`
	LastUpdated  string         `json:"LastUpdated"`
	Revisions    []RevisionInfo `json:"Revisions"`
}

type NormalizedType struct {
	Name        string         `json:"Name"`
	Module      string         `json:"Module"`
	Parent      string         `json:"Parent"`
	Base        string         `json:"Base"`
	Status      string         `json:"Status"`
	DisplayHint string         `json:"DisplayHint"`
	Description string         `json:"Description"`
	Reference   string         `json:"Reference"`
	IsTC        bool           `json:"IsTextualConvention"`
	Sizes       []RangeInfo    `json:"Sizes,omitempty"`
	Ranges      []RangeInfo    `json:"Ranges,omitempty"`
	Enums       map[int]string `json:"Enums,omitempty"`
	Bits        map[int]string `json:"Bits,omitempty"`
}

type RevisionInfo struct {
	Date        string `json:"Date"`
	Description string `json:"Description"`
}

type stdoutPayload struct {
	Nodes       map[string]*NormalizedNode   `json:"Nodes"`
	Modules     map[string]*NormalizedModule `json:"Modules,omitempty"`
	Types       map[string]*NormalizedType   `json:"Types,omitempty"`
	Diagnostics []NormalizedDiagnostic       `json:"Diagnostics,omitempty"`
}

type RangeInfo struct {
	Low  int64 `json:"Low"`
	High int64 `json:"High"`
}

type IndexInfo struct {
	Name    string `json:"Name"`
	Implied bool   `json:"Implied"`
}

type NormalizedDiagnostic struct {
	Code     string `json:"Code"`
	Severity string `json:"Severity"`
	Module   string `json:"Module,omitempty"`
	Line     int    `json:"Line,omitempty"`
	Column   int    `json:"Column,omitempty"`
	Message  string `json:"Message"`
}

var strictnessLevels = map[string]mib.ResolverStrictness{
	"permissive": mib.ResolverPermissive,
	"normal":     mib.ResolverNormal,
	"strict":     mib.ResolverStrict,
}

func main() {
	outDir := flag.String("outdir", "", "Output directory (file mode). Omit for stdout mode.")
	corpus := flag.String("corpus", "", "MIB corpus directory (required)")
	strictness := flag.String("strictness", "normal", "Resolver strictness: permissive, normal, strict")
	includeModules := flag.Bool("include-modules", false, "Include module metadata in stdout mode.")
	includeDiagnostics := flag.Bool("include-diagnostics", false, "Include normalized diagnostics in stdout mode.")
	flag.Parse()

	if *corpus == "" {
		fmt.Fprintf(os.Stderr, "Usage: gomib-fixturegen -corpus DIR [-outdir DIR] [-strictness LEVEL] [MODULE...]\n")
		flag.PrintDefaults()
		os.Exit(1)
	}

	level, ok := strictnessLevels[strings.ToLower(*strictness)]
	if !ok {
		fmt.Fprintf(os.Stderr, "error: unknown strictness level %q (use permissive, normal, strict)\n", *strictness)
		os.Exit(1)
	}

	src, err := gomib.Dir(*corpus)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	modules := flag.Args()
	opts := []gomib.LoadOption{
		gomib.WithSource(src),
		gomib.WithResolverStrictness(level),
	}
	if *includeDiagnostics {
		opts = append(opts, gomib.WithDiagnosticConfig(mib.DiagnosticConfig{
			Reporting: mib.ReportingVerbose,
			FailAt:    mib.SeverityFatal,
		}))
	}
	if len(modules) > 0 {
		opts = append(opts, gomib.WithModules(modules...))
	}

	ctx := context.Background()
	m, err := gomib.Load(ctx, opts...)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	allNodes := extractNodes(m)

	if *outDir == "" {
		// Stdout mode: write all nodes as one JSON object.
		if !*includeModules {
			data, err := json.Marshal(allNodes)
			if err != nil {
				fmt.Fprintf(os.Stderr, "error: marshal: %v\n", err)
				os.Exit(1)
			}
			_, _ = os.Stdout.Write(data)
			return
		}

		payload := stdoutPayload{
			Nodes:   allNodes,
			Modules: extractModules(m),
			Types:   extractTypes(m),
		}
		if *includeDiagnostics {
			payload.Diagnostics = extractDiagnostics(m)
		}
		data, err := json.Marshal(payload)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: marshal: %v\n", err)
			os.Exit(1)
		}
		_, _ = os.Stdout.Write(data)
		return
	}

	// File mode: write per-module files.
	if len(modules) == 0 {
		fmt.Fprintf(os.Stderr, "error: file mode (-outdir) requires MODULE arguments\n")
		os.Exit(1)
	}

	if err := os.MkdirAll(*outDir, 0o750); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	for _, mod := range modules {
		filtered := filterByModule(allNodes, mod)
		if len(filtered) == 0 {
			fmt.Fprintf(os.Stderr, "warning: no nodes for module %s\n", mod)
			continue
		}

		data, err := json.MarshalIndent(filtered, "", "  ")
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: marshal %s: %v\n", mod, err)
			os.Exit(1)
		}

		path := filepath.Join(*outDir, mod+".json")
		if err := os.WriteFile(path, append(data, '\n'), 0o600); err != nil {
			fmt.Fprintf(os.Stderr, "error: write %s: %v\n", path, err)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "wrote %s (%d nodes)\n", path, len(filtered))
	}
}

func extractNodes(m *mib.Mib) map[string]*NormalizedNode {
	nodes := make(map[string]*NormalizedNode)

	for node := range m.Nodes() {
		oid := node.OID().String()
		if oid == "" {
			continue
		}

		n := &NormalizedNode{
			OID:         oid,
			Name:        node.Name(),
			EnumValues:  make(map[int]string),
			BitValues:   make(map[int]string),
			Description: node.Description(),
			Reference:   node.Reference(),
		}

		if mod := node.Module(); mod != nil {
			n.Module = mod.Name()
			if slices.Equal(mod.OID(), node.OID()) {
				n.NodeType = "MODULE-IDENTITY"
			}
		}

		if obj := node.Object(); obj != nil {
			n.Type = normalizeType(obj.Type())
			n.Access = obj.Access().String()
			n.Status = obj.Status().String()
			n.Units = obj.Units()
			n.Hint = obj.EffectiveDisplayHint()
			n.NodeType = "OBJECT-TYPE"
			n.Kind = normalizeKind(obj.Kind())
			n.Description = obj.Description()
			n.Reference = obj.Reference()

			if t := obj.Type(); t != nil && t.IsTextualConvention() {
				n.TCName = t.Name()
			}

			for _, ev := range obj.EffectiveEnums() {
				n.EnumValues[int(ev.Value)] = ev.Label
			}
			for _, bv := range obj.EffectiveBits() {
				n.BitValues[int(bv.Value)] = bv.Label
			}

			for _, r := range obj.EffectiveRanges() {
				low, lowOK := r.Min.AsInt64()
				high, highOK := r.Max.AsInt64()
				if lowOK && highOK {
					n.Ranges = append(n.Ranges, RangeInfo{Low: low, High: high})
				}
			}
			for _, r := range obj.EffectiveSizes() {
				low, lowOK := r.Min.AsInt64()
				high, highOK := r.Max.AsInt64()
				if lowOK && highOK {
					n.Ranges = append(n.Ranges, RangeInfo{Low: low, High: high})
				}
			}

			if dv := obj.DefaultValue(); !dv.IsZero() {
				n.DefaultValue = dv.String()
			}

			for _, idx := range obj.Index() {
				if idx.Object != nil {
					n.Indexes = append(n.Indexes, IndexInfo{
						Name:    idx.Object.Name(),
						Implied: idx.Implied,
					})
				} else if idx.TypeName != "" {
					n.Indexes = append(n.Indexes, IndexInfo{
						Name:    idx.TypeName,
						Implied: idx.Implied,
					})
				}
			}

			if aug := obj.Augments(); aug != nil {
				n.Augments = aug.Name()
			}
		}

		if notif := node.Notification(); notif != nil {
			n.Status = notif.Status().String()
			n.Description = notif.Description()
			n.Reference = notif.Reference()
			n.NodeType = "NOTIFICATION-TYPE"
			for _, vb := range notif.Objects() {
				n.Varbinds = append(n.Varbinds, vb.Name())
			}
		}

		if group := node.Group(); group != nil {
			n.Status = group.Status().String()
			n.Description = group.Description()
			n.Reference = group.Reference()
			if group.IsNotificationGroup() {
				n.NodeType = "NOTIFICATION-GROUP"
			} else {
				n.NodeType = "OBJECT-GROUP"
			}
			for _, member := range group.Members() {
				n.GroupMembers = append(n.GroupMembers, member.Name())
			}
		}

		if comp := node.Compliance(); comp != nil {
			n.Status = comp.Status().String()
			n.Description = comp.Description()
			n.Reference = comp.Reference()
			n.NodeType = "MODULE-COMPLIANCE"
			for _, cm := range comp.Modules() {
				n.ComplianceModules = append(n.ComplianceModules, normalizeComplianceModule(&cm))
			}
		}

		if cap := node.Capability(); cap != nil {
			n.Status = cap.Status().String()
			n.Description = cap.Description()
			n.Reference = cap.Reference()
			n.NodeType = "AGENT-CAPABILITIES"
			n.ProductRelease = cap.ProductRelease()
			for _, sm := range cap.Supports() {
				n.CapabilitySupports = append(n.CapabilitySupports, normalizeCapabilityModule(&sm))
			}
		}

		nodes[oid] = n
	}

	return nodes
}

func normalizeComplianceModule(cm *mib.ComplianceModule) NormalizedComplianceModule {
	out := NormalizedComplianceModule{
		ModuleName:      cm.ModuleName,
		MandatoryGroups: slices.Clone(mib.NameRefNames(cm.MandatoryGroups)),
	}
	for _, group := range cm.Groups {
		out.Groups = append(out.Groups, NormalizedComplianceGroup{
			Group:       group.Group,
			Description: group.Description,
		})
	}
	for _, obj := range cm.Objects {
		minAccess := ""
		if obj.MinAccess != nil {
			minAccess = obj.MinAccess.String()
		}
		out.Objects = append(out.Objects, NormalizedComplianceObject{
			Object:      obj.Object,
			Syntax:      normalizeSyntaxConstraints(obj.Syntax),
			WriteSyntax: normalizeSyntaxConstraints(obj.WriteSyntax),
			MinAccess:   minAccess,
			Description: obj.Description,
		})
	}
	return out
}

func normalizeCapabilityModule(cm *mib.CapabilitiesModule) NormalizedCapabilityModule {
	out := NormalizedCapabilityModule{
		ModuleName: cm.ModuleName,
		Includes:   slices.Clone(mib.NameRefNames(cm.Includes)),
	}
	for i := range cm.ObjectVariations {
		ov := &cm.ObjectVariations[i]
		access := ""
		if ov.Access != nil {
			access = ov.Access.String()
		}
		defval := ""
		if !ov.DefVal.IsZero() {
			defval = ov.DefVal.String()
		}
		out.ObjectVariations = append(out.ObjectVariations, NormalizedObjectVariation{
			Object:           ov.Object,
			Syntax:           normalizeSyntaxConstraints(ov.Syntax),
			WriteSyntax:      normalizeSyntaxConstraints(ov.WriteSyntax),
			Access:           access,
			CreationRequires: slices.Clone(mib.NameRefNames(ov.CreationRequires)),
			DefaultValue:     defval,
			Description:      ov.Description,
		})
	}
	for _, nv := range cm.NotificationVariations {
		access := ""
		if nv.Access != nil {
			access = nv.Access.String()
		}
		out.NotificationVariations = append(out.NotificationVariations, NormalizedNotificationVariation{
			Notification: nv.Notification,
			Access:       access,
			Description:  nv.Description,
		})
	}
	return out
}

func normalizeSyntaxConstraints(sc *mib.SyntaxConstraints) *NormalizedSyntaxConstraints {
	if sc == nil {
		return nil
	}
	out := &NormalizedSyntaxConstraints{
		TypeName: normalizedTypeName(sc.Type),
	}
	if len(sc.Sizes) > 0 {
		out.Sizes = normalizeRanges(sc.Sizes)
	}
	if len(sc.Ranges) > 0 {
		out.Ranges = normalizeRanges(sc.Ranges)
	}
	if len(sc.Enums) > 0 {
		out.Enums = normalizeNamedValues(sc.Enums)
	}
	if len(sc.Bits) > 0 {
		out.Bits = normalizeNamedValues(sc.Bits)
	}
	return out
}

func normalizeRanges(ranges []mib.Range) []RangeInfo {
	out := make([]RangeInfo, 0, len(ranges))
	for _, r := range ranges {
		low, lowOK := r.Min.AsInt64()
		high, highOK := r.Max.AsInt64()
		if lowOK && highOK {
			out = append(out, RangeInfo{Low: low, High: high})
		}
	}
	return out
}

func normalizeNamedValues(values []mib.NamedValue) map[int]string {
	out := make(map[int]string, len(values))
	for _, nv := range values {
		out[int(nv.Value)] = nv.Label
	}
	return out
}

func normalizedTypeName(t *mib.Type) string {
	if t == nil {
		return ""
	}
	if name := t.Name(); name != "" {
		return name
	}
	return baseTypeSyntax(t.EffectiveBase())
}

func baseTypeSyntax(b mib.BaseType) string {
	switch b {
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
		return "OCTET STRING"
	case mib.BaseObjectIdentifier:
		return "OBJECT IDENTIFIER"
	case mib.BaseBits:
		return "BITS"
	case mib.BaseOpaque:
		return "Opaque"
	default:
		return b.String()
	}
}

func extractModules(m *mib.Mib) map[string]*NormalizedModule {
	modules := make(map[string]*NormalizedModule)

	for _, mod := range m.Modules() {
		nm := &NormalizedModule{
			Name:         mod.Name(),
			Organization: mod.Organization(),
			ContactInfo:  mod.ContactInfo(),
			Description:  mod.Description(),
			LastUpdated:  mod.LastUpdated(),
		}
		if oid := mod.OID(); len(oid) != 0 {
			nm.OID = oid.String()
		}
		for _, rev := range mod.Revisions() {
			nm.Revisions = append(nm.Revisions, RevisionInfo{
				Date:        rev.Date,
				Description: rev.Description,
			})
		}
		modules[mod.Name()] = nm
	}

	return modules
}

func extractTypes(m *mib.Mib) map[string]*NormalizedType {
	types := make(map[string]*NormalizedType)

	for _, typ := range m.Types() {
		if typ.Name() == "" {
			continue
		}

		nt := &NormalizedType{
			Name:        typ.Name(),
			Base:        typ.EffectiveBase().String(),
			Status:      typ.Status().String(),
			DisplayHint: typ.EffectiveDisplayHint(),
			Description: typ.Description(),
			Reference:   typ.Reference(),
			IsTC:        typ.IsTextualConvention(),
			Sizes:       normalizeRanges(typ.EffectiveSizes()),
			Ranges:      normalizeRanges(typ.EffectiveRanges()),
			Enums:       normalizeNamedValues(typ.EffectiveEnums()),
			Bits:        normalizeNamedValues(typ.EffectiveBits()),
		}
		if mod := typ.Module(); mod != nil {
			nt.Module = mod.Name()
		}
		if parent := typ.Parent(); parent != nil {
			nt.Parent = normalizedParentTypeName(parent)
		}

		types[qualifiedTypeName(nt.Module, nt.Name)] = nt
	}

	return types
}

func extractDiagnostics(m *mib.Mib) []NormalizedDiagnostic {
	diags := make([]NormalizedDiagnostic, 0, len(m.Diagnostics()))
	for _, d := range m.Diagnostics() {
		diags = append(diags, NormalizedDiagnostic{
			Code:     d.Code,
			Severity: d.Severity.String(),
			Module:   d.Module,
			Line:     d.Line,
			Column:   d.Column,
			Message:  normalizeDiagnosticMessage(d.Message),
		})
	}
	slices.SortFunc(diags, func(a, b NormalizedDiagnostic) int {
		if a.Code != b.Code {
			return strings.Compare(a.Code, b.Code)
		}
		if a.Severity != b.Severity {
			return strings.Compare(a.Severity, b.Severity)
		}
		if a.Module != b.Module {
			return strings.Compare(a.Module, b.Module)
		}
		if a.Line != b.Line {
			return a.Line - b.Line
		}
		if a.Column != b.Column {
			return a.Column - b.Column
		}
		return strings.Compare(a.Message, b.Message)
	})
	return diags
}

func normalizeDiagnosticMessage(msg string) string {
	return strings.Join(strings.Fields(msg), " ")
}

func normalizeType(t *mib.Type) string {
	if t == nil {
		return ""
	}
	return t.EffectiveBase().String()
}

func normalizeKind(k mib.Kind) string {
	if k.IsObjectType() {
		return k.String()
	}
	return ""
}

func filterByModule(nodes map[string]*NormalizedNode, mod string) map[string]*NormalizedNode {
	filtered := make(map[string]*NormalizedNode)
	for oid, node := range nodes {
		if node.Module == mod {
			filtered[oid] = node
		}
	}
	return filtered
}

func qualifiedTypeName(module, name string) string {
	if module == "" {
		return name
	}
	return module + "::" + name
}

func normalizedParentTypeName(t *mib.Type) string {
	if name := t.Name(); name != "" {
		return name
	}
	return normalizedTypeName(t)
}
