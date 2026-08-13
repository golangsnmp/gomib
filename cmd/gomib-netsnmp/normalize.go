//go:build cgo

package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/golangsnmp/gomib"
	"github.com/golangsnmp/gomib/cmd/internal/cliutil"
	"github.com/golangsnmp/gomib/mib"
)

const maxNetSnmpDirs = 500 // sanity limit to prevent accidental "/" or similar

// NormalizedNode is a library-independent MIB node used for cross-validation
// between gomib and net-snmp.
type NormalizedNode struct {
	OID        string
	Name       string
	Module     string
	Type       string // Normalized base type name
	Access     string
	Status     string
	Hint       string
	TCName     string // Textual convention name
	Units      string
	EnumValues map[int]string
	Indexes    []IndexInfo
	Augments   string

	Ranges       []RangeInfo    // Size/value constraints
	DefaultValue string         // DEFVAL clause
	Kind         string         // table, row, column, scalar, or empty
	Varbinds     []string       // OBJECTS clause for notifications
	NodeType     string         // NOTIFICATION-TYPE, TRAP-TYPE, OBJECT-TYPE, etc.
	BitValues    map[int]string // BITS named values (separate from enums)
	Reference    string         // REFERENCE clause
}

// RangeInfo holds a min/max constraint pair.
type RangeInfo struct {
	Low  int64
	High int64
}

// IndexInfo holds an INDEX entry name and its implied flag.
type IndexInfo struct {
	Name    string
	Implied bool
}

func indexString(indexes []IndexInfo) string {
	if len(indexes) == 0 {
		return ""
	}
	var parts []string
	for _, idx := range indexes {
		if idx.Implied {
			parts = append(parts, "IMPLIED "+idx.Name)
		} else {
			parts = append(parts, idx.Name)
		}
	}
	return "{ " + strings.Join(parts, ", ") + " }"
}

func rangesString(ranges []RangeInfo) string {
	if len(ranges) == 0 {
		return ""
	}
	var parts []string
	for _, r := range ranges {
		if r.Low == r.High {
			parts = append(parts, strconv.FormatInt(r.Low, 10))
		} else {
			parts = append(parts, fmt.Sprintf("%d..%d", r.Low, r.High))
		}
	}
	return "(" + strings.Join(parts, " | ") + ")"
}

func varbindsString(varbinds []string) string {
	if len(varbinds) == 0 {
		return ""
	}
	return "{ " + strings.Join(varbinds, ", ") + " }"
}

// loadNetSnmpNodes loads MIBs with net-snmp and returns normalized nodes.
// net-snmp only reads flat directories, so all subdirectories are
// discovered and joined into a colon-separated path.
func loadNetSnmpNodes(mibPaths, modules []string) (map[string]*NormalizedNode, error) {
	if len(mibPaths) == 0 {
		return nil, errors.New("no MIB paths specified (use -p flag)")
	}

	for _, root := range mibPaths {
		info, err := os.Stat(root)
		if err != nil {
			return nil, fmt.Errorf("cannot access %s: %w", root, err)
		}
		if !info.IsDir() {
			return nil, fmt.Errorf("%s is not a directory", root)
		}
	}

	allDirs := cliutil.ExpandDirs(mibPaths)

	if len(allDirs) > maxNetSnmpDirs {
		return nil, fmt.Errorf("too many directories (%d > %d) - check your -p paths", len(allDirs), maxNetSnmpDirs)
	}

	mibDir := strings.Join(allDirs, ":")
	initNetSnmp(mibDir, modules)
	return collectNetSnmpNodes(), nil
}

func loadGomibNodes(mibPaths, modules []string) (map[string]*NormalizedNode, error) {
	if len(mibPaths) == 0 {
		return nil, errors.New("no MIB paths specified (use -p flag)")
	}

	var sources []gomib.Source
	for _, p := range mibPaths {
		src, err := gomib.Dir(p)
		if err != nil {
			return nil, fmt.Errorf("invalid path %s: %w", p, err)
		}
		sources = append(sources, src)
	}

	var source gomib.Source
	if len(sources) == 1 {
		source = sources[0]
	} else {
		source = gomib.Multi(sources...)
	}

	ctx := context.Background()
	var m *mib.Mib
	var err error

	loadOpts := []gomib.LoadOption{gomib.WithSource(source)}
	if len(modules) > 0 {
		loadOpts = append(loadOpts, gomib.WithModules(modules...))
	}
	m, err = gomib.Load(ctx, loadOpts...)
	if err != nil {
		return nil, fmt.Errorf("gomib load failed: %w", err)
	}

	nodes := make(map[string]*NormalizedNode)
	for node := range m.Nodes() {
		if n := normalizeGomibNode(node); n != nil {
			nodes[n.OID] = n
		}
	}
	return nodes, nil
}

// normalizeGomibNode converts a single resolved mib.Node into a NormalizedNode.
// Returns nil for nodes with no OID.
func normalizeGomibNode(node *mib.Node) *NormalizedNode {
	oid := node.OID().String()
	if oid == "" {
		return nil
	}

	n := &NormalizedNode{
		OID:        oid,
		Name:       node.Name(),
		EnumValues: make(map[int]string),
		BitValues:  make(map[int]string),
	}

	if mod := node.Module(); mod != nil {
		n.Module = mod.Name()
	}

	if obj := node.Object(); obj != nil {
		normalizeGomibObject(n, obj)
	}

	if notif := node.Notification(); notif != nil {
		normalizeGomibNotification(n, notif)
	}

	return n
}

func normalizeGomibObject(n *NormalizedNode, obj *mib.Object) {
	n.Type = normalizeGomibType(obj.Type())
	n.Access = obj.Access().String()
	n.Status = obj.Status().String()
	n.Units = obj.Units()
	n.Hint = obj.EffectiveDisplayHint()
	n.NodeType = "OBJECT-TYPE"
	n.Kind = normalizeGomibKind(obj.Kind())
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
		if r.IsResolved() {
			n.Ranges = append(n.Ranges, RangeInfo{Low: r.Min, High: r.Max})
		}
	}
	for _, r := range obj.EffectiveSizes() {
		if r.IsResolved() {
			n.Ranges = append(n.Ranges, RangeInfo{Low: r.Min, High: r.Max})
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

func normalizeGomibNotification(n *NormalizedNode, notif *mib.Notification) {
	n.Status = notif.Status().String()
	n.Reference = notif.Reference()
	n.NodeType = "NOTIFICATION-TYPE"
	for _, vb := range notif.Objects() {
		n.Varbinds = append(n.Varbinds, vb.Name())
	}
}

func normalizeGomibType(t *mib.Type) string {
	if t == nil {
		return ""
	}
	return t.EffectiveBase().String()
}

func normalizeGomibKind(k mib.Kind) string {
	if k.IsObjectType() {
		return k.String()
	}
	return ""
}

func filterByModules(nodes map[string]*NormalizedNode, modules []string) map[string]*NormalizedNode {
	if len(modules) == 0 {
		return nodes
	}

	modSet := make(map[string]bool)
	for _, m := range modules {
		modSet[m] = true
	}

	filtered := make(map[string]*NormalizedNode)
	for oid, node := range nodes {
		if modSet[node.Module] {
			filtered[oid] = node
		}
	}
	return filtered
}
