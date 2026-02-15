//go:build cgo

package main

/*
#cgo LDFLAGS: -lnetsnmp
#cgo CFLAGS: -I/usr/include

#include <net-snmp/net-snmp-config.h>
#include <net-snmp/net-snmp-includes.h>
#include <net-snmp/mib_api.h>
#include <net-snmp/library/snmp_api.h>
#include <net-snmp/library/parse.h>
#include <net-snmp/library/default_store.h>
#include <stdlib.h>
#include <stdio.h>
#include <string.h>

// Type codes from parse.h
#define TYPE_OTHER          0
#define TYPE_OBJID          1
#define TYPE_OCTETSTR       2
#define TYPE_INTEGER        3
#define TYPE_NETADDR        4
#define TYPE_IPADDR         5
#define TYPE_COUNTER        6
#define TYPE_GAUGE          7
#define TYPE_TIMETICKS      8
#define TYPE_OPAQUE         9
#define TYPE_NULL           10
#define TYPE_COUNTER64      11
#define TYPE_BITSTRING      12
#define TYPE_NSAPADDRESS    13
#define TYPE_UINTEGER       14
#define TYPE_UNSIGNED32     15
#define TYPE_INTEGER32      16
#define TYPE_TRAPTYPE       20
#define TYPE_NOTIFTYPE      21
#define TYPE_OBJGROUP       22
#define TYPE_NOTIFGROUP     23
#define TYPE_MODID          24
#define TYPE_AGENTCAP       25
#define TYPE_MODCOMP        26
#define TYPE_OBJIDENTITY    27

// Access codes
#define MIB_ACCESS_READONLY    18
#define MIB_ACCESS_READWRITE   19
#define MIB_ACCESS_WRITEONLY   20
#define MIB_ACCESS_NOACCESS    21
#define MIB_ACCESS_NOTIFY      67
#define MIB_ACCESS_CREATE      48

// Status codes
#define MIB_STATUS_MANDATORY   23
#define MIB_STATUS_OPTIONAL    24
#define MIB_STATUS_OBSOLETE    25
#define MIB_STATUS_DEPRECATED  39
#define MIB_STATUS_CURRENT     57

// Node info structure to pass to Go
typedef struct {
    char oid[256];
    char name[128];
    char module[128];
    int type;
    int access;
    int status;
    char hint[64];
    char tc_name[128];
    char units[64];
    int enum_count;
    int* enum_values;
    char** enum_names;
    int index_count;
    char** indexes;
    int* implied_flags;
    char augments[128];
    // Additional fields
    int range_count;
    int* range_lows;
    int* range_highs;
    char default_value[256];
    int varbind_count;
    char** varbinds;
    char reference[512];
} NodeInfo;

static int node_count = 0;
static int node_capacity = 0;
static NodeInfo* nodes = NULL;

void build_oid_string(struct tree* tp, char* buf, int buflen) {
    if (tp->parent) {
        build_oid_string(tp->parent, buf, buflen);
        char tmp[32];
        snprintf(tmp, sizeof(tmp), ".%lu", tp->subid);
        strncat(buf, tmp, buflen - strlen(buf) - 1);
    } else {
        snprintf(buf, buflen, "%lu", tp->subid);
    }
}

void collect_node(struct tree* tp) {
    if (node_count >= node_capacity) {
        node_capacity = node_capacity == 0 ? 10000 : node_capacity * 2;
        nodes = realloc(nodes, node_capacity * sizeof(NodeInfo));
    }

    NodeInfo* n = &nodes[node_count];
    memset(n, 0, sizeof(NodeInfo));

    build_oid_string(tp, n->oid, sizeof(n->oid));

    if (tp->label) strncpy(n->name, tp->label, sizeof(n->name) - 1);

    struct module* mod = find_module(tp->modid);
    if (mod && mod->name) strncpy(n->module, mod->name, sizeof(n->module) - 1);

    n->type = tp->type;
    n->access = tp->access;
    n->status = tp->status;

    if (tp->hint) strncpy(n->hint, tp->hint, sizeof(n->hint) - 1);
    if (tp->units) strncpy(n->units, tp->units, sizeof(n->units) - 1);
    if (tp->augments) strncpy(n->augments, tp->augments, sizeof(n->augments) - 1);

    if (tp->tc_index >= 0) {
        const char* tc = get_tc_descriptor(tp->tc_index);
        if (tc) strncpy(n->tc_name, tc, sizeof(n->tc_name) - 1);
    }

    // Collect enums
    struct enum_list* ep = tp->enums;
    int enum_count = 0;
    while (ep) { enum_count++; ep = ep->next; }

    if (enum_count > 0) {
        n->enum_count = enum_count;
        n->enum_values = malloc(enum_count * sizeof(int));
        n->enum_names = malloc(enum_count * sizeof(char*));

        ep = tp->enums;
        int i = 0;
        while (ep) {
            n->enum_values[i] = ep->value;
            n->enum_names[i] = strdup(ep->label ? ep->label : "");
            i++;
            ep = ep->next;
        }
    }

    // Collect indexes with IMPLIED flags
    struct index_list* idx = tp->indexes;
    int idx_count = 0;
    while (idx) { idx_count++; idx = idx->next; }

    if (idx_count > 0) {
        n->index_count = idx_count;
        n->indexes = malloc(idx_count * sizeof(char*));
        n->implied_flags = malloc(idx_count * sizeof(int));

        idx = tp->indexes;
        int i = 0;
        while (idx) {
            n->indexes[i] = strdup(idx->ilabel ? idx->ilabel : "");
            n->implied_flags[i] = idx->isimplied ? 1 : 0;
            i++;
            idx = idx->next;
        }
    }

    // Collect ranges
    struct range_list* rp = tp->ranges;
    int range_count = 0;
    while (rp) { range_count++; rp = rp->next; }

    if (range_count > 0) {
        n->range_count = range_count;
        n->range_lows = malloc(range_count * sizeof(int));
        n->range_highs = malloc(range_count * sizeof(int));

        rp = tp->ranges;
        int i = 0;
        while (rp) {
            n->range_lows[i] = rp->low;
            n->range_highs[i] = rp->high;
            i++;
            rp = rp->next;
        }
    }

    // Collect default value
    if (tp->defaultValue) {
        strncpy(n->default_value, tp->defaultValue, sizeof(n->default_value) - 1);
    }

    // Collect varbinds (OBJECTS clause for notifications)
    struct varbind_list* vb = tp->varbinds;
    int vb_count = 0;
    while (vb) { vb_count++; vb = vb->next; }

    if (vb_count > 0) {
        n->varbind_count = vb_count;
        n->varbinds = malloc(vb_count * sizeof(char*));

        vb = tp->varbinds;
        int i = 0;
        while (vb) {
            n->varbinds[i] = strdup(vb->vblabel ? vb->vblabel : "");
            i++;
            vb = vb->next;
        }
    }

    // Collect reference
    if (tp->reference) {
        strncpy(n->reference, tp->reference, sizeof(n->reference) - 1);
    }

    node_count++;
}

void walk_tree(struct tree* tp) {
    if (!tp) return;

    collect_node(tp);

    struct tree* child = tp->child_list;
    while (child) {
        walk_tree(child);
        child = child->next_peer;
    }
}

int init_netsnmp(const char* mib_dir) {
    // Clear environment to prevent system defaults from being used
    unsetenv("MIBDIRS");
    unsetenv("SNMPCONFPATH");
    unsetenv("SNMP_PERSISTENT_DIR");
    setenv("MIBS", "ALL", 1);

    // Disable config file processing
    netsnmp_ds_set_boolean(NETSNMP_DS_LIBRARY_ID, NETSNMP_DS_LIB_NO_TOKEN_WARNINGS, 1);
    netsnmp_ds_set_boolean(NETSNMP_DS_LIBRARY_ID, NETSNMP_DS_LIB_DONT_READ_CONFIGS, 1);

    // Set our MIB directory (replaces defaults when not prefixed with +)
    netsnmp_set_mib_directory(mib_dir);
    snmp_set_save_descriptions(1);
    netsnmp_init_mib();
    return 0;
}

int init_netsnmp_modules(const char* mib_dir, const char* modules) {
    // Clear environment to prevent system defaults from being used
    unsetenv("MIBDIRS");
    unsetenv("SNMPCONFPATH");
    unsetenv("SNMP_PERSISTENT_DIR");
    setenv("MIBS", modules, 1);

    // Disable config file processing
    netsnmp_ds_set_boolean(NETSNMP_DS_LIBRARY_ID, NETSNMP_DS_LIB_NO_TOKEN_WARNINGS, 1);
    netsnmp_ds_set_boolean(NETSNMP_DS_LIBRARY_ID, NETSNMP_DS_LIB_DONT_READ_CONFIGS, 1);

    // Set our MIB directory (replaces defaults when not prefixed with +)
    netsnmp_set_mib_directory(mib_dir);
    snmp_set_save_descriptions(1);
    netsnmp_init_mib();
    return 0;
}

int get_collected_node_count() {
    return node_count;
}

NodeInfo* get_collected_node(int index) {
    if (index < 0 || index >= node_count) return NULL;
    return &nodes[index];
}

void collect_all_nodes() {
    struct tree* tp = get_tree_head();
    if (tp) {
        walk_tree(tp);
    }
}

void cleanup_nodes() {
    for (int i = 0; i < node_count; i++) {
        NodeInfo* n = &nodes[i];
        if (n->enum_values) free(n->enum_values);
        if (n->enum_names) {
            for (int j = 0; j < n->enum_count; j++) {
                free(n->enum_names[j]);
            }
            free(n->enum_names);
        }
        if (n->indexes) {
            for (int j = 0; j < n->index_count; j++) {
                free(n->indexes[j]);
            }
            free(n->indexes);
        }
        if (n->implied_flags) free(n->implied_flags);
        if (n->range_lows) free(n->range_lows);
        if (n->range_highs) free(n->range_highs);
        if (n->varbinds) {
            for (int j = 0; j < n->varbind_count; j++) {
                free(n->varbinds[j]);
            }
            free(n->varbinds);
        }
    }
    free(nodes);
    nodes = NULL;
    node_count = 0;
    node_capacity = 0;
}

const char* type_to_string(int type) {
    switch (type) {
        case TYPE_OTHER: return "OTHER";
        case TYPE_OBJID: return "OBJECT IDENTIFIER";
        case TYPE_OCTETSTR: return "OCTET STRING";
        case TYPE_INTEGER: return "INTEGER";
        case TYPE_NETADDR: return "NetworkAddress";
        case TYPE_IPADDR: return "IpAddress";
        case TYPE_COUNTER: return "Counter32";
        case TYPE_GAUGE: return "Gauge32";
        case TYPE_TIMETICKS: return "TimeTicks";
        case TYPE_OPAQUE: return "Opaque";
        case TYPE_NULL: return "NULL";
        case TYPE_COUNTER64: return "Counter64";
        case TYPE_BITSTRING: return "BITS";
        case TYPE_NSAPADDRESS: return "NsapAddress";
        case TYPE_UINTEGER: return "UInteger32";
        case TYPE_UNSIGNED32: return "Unsigned32";
        case TYPE_INTEGER32: return "Integer32";
        case TYPE_TRAPTYPE: return "TRAP-TYPE";
        case TYPE_NOTIFTYPE: return "NOTIFICATION-TYPE";
        case TYPE_OBJGROUP: return "OBJECT-GROUP";
        case TYPE_NOTIFGROUP: return "NOTIFICATION-GROUP";
        case TYPE_MODID: return "MODULE-IDENTITY";
        case TYPE_AGENTCAP: return "AGENT-CAPABILITIES";
        case TYPE_MODCOMP: return "MODULE-COMPLIANCE";
        case TYPE_OBJIDENTITY: return "OBJECT-IDENTITY";
        default: return "UNKNOWN";
    }
}

const char* access_to_string(int access) {
    switch (access) {
        case MIB_ACCESS_READONLY: return "read-only";
        case MIB_ACCESS_READWRITE: return "read-write";
        case MIB_ACCESS_WRITEONLY: return "write-only";
        case MIB_ACCESS_NOACCESS: return "not-accessible";
        case MIB_ACCESS_NOTIFY: return "accessible-for-notify";
        case MIB_ACCESS_CREATE: return "read-create";
        default: return "";
    }
}

const char* status_to_string(int status) {
    switch (status) {
        case MIB_STATUS_CURRENT: return "current";
        case MIB_STATUS_DEPRECATED: return "deprecated";
        case MIB_STATUS_OBSOLETE: return "obsolete";
        case MIB_STATUS_MANDATORY: return "mandatory";
        case MIB_STATUS_OPTIONAL: return "optional";
        default: return "";
    }
}

*/
import "C"

import (
	"strings"
	"unsafe"
)

func initNetSnmp(mibDir string, modules []string) {
	cMibDir := C.CString(mibDir)
	defer C.free(unsafe.Pointer(cMibDir))

	if len(modules) == 0 {
		C.init_netsnmp(cMibDir)
	} else {
		modList := C.CString(strings.Join(modules, ":"))
		defer C.free(unsafe.Pointer(modList))
		C.init_netsnmp_modules(cMibDir, modList)
	}
}

func collectNetSnmpNodes() map[string]*NormalizedNode {
	C.collect_all_nodes()
	defer C.cleanup_nodes()

	nodeCount := int(C.get_collected_node_count())
	nodes := make(map[string]*NormalizedNode)

	for i := 0; i < nodeCount; i++ {
		cNode := C.get_collected_node(C.int(i))
		if cNode == nil {
			continue
		}

		oid := C.GoString(&cNode.oid[0])
		if oid == "" {
			continue
		}

		node := &NormalizedNode{
			OID:          oid,
			Name:         C.GoString(&cNode.name[0]),
			Module:       C.GoString(&cNode.module[0]),
			Type:         C.GoString(C.type_to_string(cNode._type)),
			Access:       C.GoString(C.access_to_string(cNode.access)),
			Status:       C.GoString(C.status_to_string(cNode.status)),
			Hint:         C.GoString(&cNode.hint[0]),
			TCName:       C.GoString(&cNode.tc_name[0]),
			Units:        C.GoString(&cNode.units[0]),
			Augments:     C.GoString(&cNode.augments[0]),
			DefaultValue: C.GoString(&cNode.default_value[0]),
			Reference:    C.GoString(&cNode.reference[0]),
			NodeType:     C.GoString(C.type_to_string(cNode._type)),
			EnumValues:   make(map[int]string),
			BitValues:    make(map[int]string),
		}

		node.Kind = inferNetSnmpKind(cNode._type, int(cNode.index_count), node.Augments)

		// net-snmp uses the same enum list for INTEGER enums and BITS;
		// distinguish based on type code
		isBitsType := cNode._type == 12 // TYPE_BITSTRING
		for j := 0; j < int(cNode.enum_count); j++ {
			val := int(*(*C.int)(unsafe.Pointer(uintptr(unsafe.Pointer(cNode.enum_values)) + uintptr(j)*unsafe.Sizeof(C.int(0)))))
			namePtr := *(**C.char)(unsafe.Pointer(uintptr(unsafe.Pointer(cNode.enum_names)) + uintptr(j)*unsafe.Sizeof((*C.char)(nil))))
			name := C.GoString(namePtr)
			if isBitsType {
				node.BitValues[val] = name
			} else {
				node.EnumValues[val] = name
			}
		}

		for j := 0; j < int(cNode.index_count); j++ {
			idxPtr := *(**C.char)(unsafe.Pointer(uintptr(unsafe.Pointer(cNode.indexes)) + uintptr(j)*unsafe.Sizeof((*C.char)(nil))))
			implied := *(*C.int)(unsafe.Pointer(uintptr(unsafe.Pointer(cNode.implied_flags)) + uintptr(j)*unsafe.Sizeof(C.int(0))))
			node.Indexes = append(node.Indexes, IndexInfo{
				Name:    C.GoString(idxPtr),
				Implied: implied != 0,
			})
		}

		for j := 0; j < int(cNode.range_count); j++ {
			low := int64(*(*C.int)(unsafe.Pointer(uintptr(unsafe.Pointer(cNode.range_lows)) + uintptr(j)*unsafe.Sizeof(C.int(0)))))
			high := int64(*(*C.int)(unsafe.Pointer(uintptr(unsafe.Pointer(cNode.range_highs)) + uintptr(j)*unsafe.Sizeof(C.int(0)))))
			node.Ranges = append(node.Ranges, RangeInfo{Low: low, High: high})
		}

		for j := 0; j < int(cNode.varbind_count); j++ {
			vbPtr := *(**C.char)(unsafe.Pointer(uintptr(unsafe.Pointer(cNode.varbinds)) + uintptr(j)*unsafe.Sizeof((*C.char)(nil))))
			node.Varbinds = append(node.Varbinds, C.GoString(vbPtr))
		}

		nodes[oid] = node
	}

	return nodes
}

// inferNetSnmpKind infers the node kind from net-snmp type and structure.
// net-snmp does not directly expose table/row/column/scalar, so we
// use heuristics: nodes with INDEX or AUGMENTS are rows; the rest
// cannot be reliably classified without tree context.
func inferNetSnmpKind(nodeType C.int, indexCount int, augments string) string {
	if indexCount > 0 || augments != "" {
		return "row"
	}
	return ""
}
