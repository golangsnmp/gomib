//go:build cgo

package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/golangsnmp/gomib"
)

const maxNetSnmpDirs = 500 // sanity limit to prevent accidental "/" or similar

// NormalizedNode is a common representation for comparison between libraries.
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
}

// IndexInfo describes an index component.
type IndexInfo struct {
	Name    string
	Implied bool
}

// String returns a human-readable representation of an index list.
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

// loadNetSnmpNodes loads MIBs with net-snmp and returns normalized nodes.
// Since net-snmp only reads flat directories, we find all subdirectories
// and pass them as colon-separated paths.
func loadNetSnmpNodes(mibPaths []string, modules []string) (map[string]*NormalizedNode, error) {
	if len(mibPaths) == 0 {
		return nil, fmt.Errorf("no MIB paths specified (use -p flag)")
	}

	// Find all directories (net-snmp doesn't recurse into subdirs)
	allDirs, err := findAllDirs(mibPaths)
	if err != nil {
		return nil, err
	}

	if len(allDirs) > maxNetSnmpDirs {
		return nil, fmt.Errorf("too many directories (%d > %d) - check your -p paths", len(allDirs), maxNetSnmpDirs)
	}

	// net-snmp uses colon-separated paths
	mibDir := strings.Join(allDirs, ":")
	initNetSnmp(mibDir, modules)
	return collectNetSnmpNodes(), nil
}

// findAllDirs returns all directories under the given roots.
func findAllDirs(roots []string) ([]string, error) {
	var dirs []string
	seen := make(map[string]bool)

	for _, root := range roots {
		info, err := os.Stat(root)
		if err != nil {
			return nil, fmt.Errorf("cannot access %s: %w", root, err)
		}
		if !info.IsDir() {
			return nil, fmt.Errorf("%s is not a directory", root)
		}

		err = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil // skip inaccessible
			}
			if info.IsDir() && !seen[path] {
				seen[path] = true
				dirs = append(dirs, path)
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}

	return dirs, nil
}

// loadGomibNodes loads MIBs with gomib and returns normalized nodes.
// Note: gomib's DirTree handles nested directories, but for fair comparison
// with net-snmp, callers should provide flat directory paths.
func loadGomibNodes(mibPaths []string, modules []string) (map[string]*NormalizedNode, error) {
	if len(mibPaths) == 0 {
		return nil, fmt.Errorf("no MIB paths specified (use -p flag)")
	}

	var sources []gomib.Source
	for _, p := range mibPaths {
		src, err := gomib.DirTree(p)
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
	var mib gomib.Mib
	var err error

	if len(modules) > 0 {
		mib, err = gomib.LoadModules(ctx, modules, source)
	} else {
		mib, err = gomib.Load(ctx, source)
	}
	if err != nil {
		return nil, fmt.Errorf("gomib load failed: %w", err)
	}

	nodes := make(map[string]*NormalizedNode)

	for node := range mib.Nodes() {
		oid := node.OID().String()
		if oid == "" {
			continue
		}

		n := &NormalizedNode{
			OID:        oid,
			Name:       node.Name(),
			EnumValues: make(map[int]string),
		}

		if mod := node.Module(); mod != nil {
			n.Module = mod.Name()
		}

		if obj := node.Object(); obj != nil {
			n.Type = normalizeGomibType(obj.Type())
			n.Access = normalizeGomibAccess(obj.Access())
			n.Status = normalizeGomibStatus(obj.Status())
			n.Units = obj.Units()
			n.Hint = obj.EffectiveDisplayHint()

			if t := obj.Type(); t != nil {
				n.TCName = t.Name()
			}

			// Get enums
			for _, ev := range obj.EffectiveEnums() {
				n.EnumValues[int(ev.Value)] = ev.Label
			}

			// Get indexes
			for _, idx := range obj.Index() {
				if idx.Object != nil {
					n.Indexes = append(n.Indexes, IndexInfo{
						Name:    idx.Object.Name(),
						Implied: idx.Implied,
					})
				}
			}

			// Get augments
			if aug := obj.Augments(); aug != nil {
				n.Augments = aug.Name()
			}
		}

		nodes[oid] = n
	}

	return nodes, nil
}

// normalizeGomibType converts gomib type to normalized string.
func normalizeGomibType(t gomib.Type) string {
	if t == nil {
		return ""
	}
	base := t.EffectiveBase()
	switch base {
	case gomib.BaseInteger32:
		return "Integer32"
	case gomib.BaseUnsigned32:
		return "Unsigned32"
	case gomib.BaseCounter32:
		return "Counter32"
	case gomib.BaseCounter64:
		return "Counter64"
	case gomib.BaseGauge32:
		return "Gauge32"
	case gomib.BaseTimeTicks:
		return "TimeTicks"
	case gomib.BaseIpAddress:
		return "IpAddress"
	case gomib.BaseOctetString:
		return "OCTET STRING"
	case gomib.BaseObjectIdentifier:
		return "OBJECT IDENTIFIER"
	case gomib.BaseBits:
		return "BITS"
	case gomib.BaseOpaque:
		return "Opaque"
	default:
		return base.String()
	}
}

// normalizeGomibAccess converts gomib access to normalized string.
func normalizeGomibAccess(a gomib.Access) string {
	switch a {
	case gomib.AccessNotAccessible:
		return "not-accessible"
	case gomib.AccessAccessibleForNotify:
		return "accessible-for-notify"
	case gomib.AccessReadOnly:
		return "read-only"
	case gomib.AccessReadWrite:
		return "read-write"
	case gomib.AccessReadCreate:
		return "read-create"
	case gomib.AccessWriteOnly:
		return "write-only"
	default:
		return ""
	}
}

// normalizeGomibStatus converts gomib status to normalized string.
func normalizeGomibStatus(s gomib.Status) string {
	switch s {
	case gomib.StatusCurrent:
		return "current"
	case gomib.StatusDeprecated:
		return "deprecated"
	case gomib.StatusObsolete:
		return "obsolete"
	default:
		return ""
	}
}

// filterByModules filters nodes to only include those from specified modules.
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
