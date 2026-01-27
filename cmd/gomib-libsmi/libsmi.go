//go:build cgo

package main

/*
#cgo LDFLAGS: -lsmi
#cgo CFLAGS: -I/usr/include

#include <smi.h>
#include <stdlib.h>
#include <string.h>
#include <stdio.h>

// Error collection structure
typedef struct {
    char* path;
    int line;
    int severity;
    char* message;
    char* tag;
} CollectedError;

static CollectedError* collected_errors = NULL;
static int error_count = 0;
static int error_capacity = 0;

// Custom error handler to collect all diagnostics
void collect_error_handler(char* path, int line, int severity, char* msg, char* tag) {
    if (error_count >= error_capacity) {
        error_capacity = error_capacity == 0 ? 100 : error_capacity * 2;
        collected_errors = realloc(collected_errors, error_capacity * sizeof(CollectedError));
    }

    CollectedError* e = &collected_errors[error_count];
    e->path = path ? strdup(path) : NULL;
    e->line = line;
    e->severity = severity;
    e->message = msg ? strdup(msg) : NULL;
    e->tag = tag ? strdup(tag) : NULL;
    error_count++;
}

int init_libsmi(const char* mib_path, int error_level) {
    // Clear environment to prevent system defaults from being used
    unsetenv("SMIPATH");

    smiInit("gomib-libsmi");
    smiSetErrorLevel(error_level);
    smiSetErrorHandler(collect_error_handler);

    // Enable semantic checking (like smilint -s)
    smiSetFlags(smiGetFlags() | SMI_FLAG_ERRORS);

    // Set MIB path (replaces any defaults)
    if (mib_path && strlen(mib_path) > 0) {
        smiSetPath(mib_path);
    }

    return 0;
}

void cleanup_libsmi() {
    // Free collected errors
    for (int i = 0; i < error_count; i++) {
        if (collected_errors[i].path) free(collected_errors[i].path);
        if (collected_errors[i].message) free(collected_errors[i].message);
        if (collected_errors[i].tag) free(collected_errors[i].tag);
    }
    if (collected_errors) {
        free(collected_errors);
        collected_errors = NULL;
    }
    error_count = 0;
    error_capacity = 0;

    smiExit();
}

void clear_errors() {
    for (int i = 0; i < error_count; i++) {
        if (collected_errors[i].path) free(collected_errors[i].path);
        if (collected_errors[i].message) free(collected_errors[i].message);
        if (collected_errors[i].tag) free(collected_errors[i].tag);
    }
    error_count = 0;
}

int load_module(const char* module_name) {
    char* name = smiLoadModule(module_name);
    return name != NULL ? 1 : 0;
}

int get_error_count() {
    return error_count;
}

CollectedError* get_error(int index) {
    if (index < 0 || index >= error_count) return NULL;
    return &collected_errors[index];
}

// Module information
typedef struct {
    char name[128];
    char path[512];
    int language;  // SMI_LANGUAGE_SMIV1 = 1, SMI_LANGUAGE_SMIV2 = 2
    int conformance;
    char organization[256];
    char contactinfo[512];
    char description[2048];
} ModuleInfo;

ModuleInfo* get_module_info(const char* module_name) {
    SmiModule* mod = smiGetModule(module_name);
    if (!mod) return NULL;

    static ModuleInfo info;
    memset(&info, 0, sizeof(info));

    if (mod->name) strncpy(info.name, mod->name, sizeof(info.name) - 1);
    if (mod->path) strncpy(info.path, mod->path, sizeof(info.path) - 1);
    info.language = mod->language;
    info.conformance = mod->conformance;
    if (mod->organization) strncpy(info.organization, mod->organization, sizeof(info.organization) - 1);
    if (mod->contactinfo) strncpy(info.contactinfo, mod->contactinfo, sizeof(info.contactinfo) - 1);
    if (mod->description) strncpy(info.description, mod->description, sizeof(info.description) - 1);

    return &info;
}

// Node information for comparison
typedef struct {
    char oid[256];
    char name[128];
    char module[128];
    int nodekind;
    int status;
    int access;
    int basetype;
    char typename_[128];
    char description[2048];
} NodeInfo;

static NodeInfo* nodes = NULL;
static int node_count = 0;
static int node_capacity = 0;

void build_oid_str(SmiSubid* oid, unsigned int oidlen, char* buf, int buflen) {
    buf[0] = '\0';
    for (unsigned int i = 0; i < oidlen; i++) {
        char tmp[32];
        if (i == 0) {
            snprintf(tmp, sizeof(tmp), "%u", oid[i]);
        } else {
            snprintf(tmp, sizeof(tmp), ".%u", oid[i]);
        }
        strncat(buf, tmp, buflen - strlen(buf) - 1);
    }
}

void collect_node(SmiNode* node) {
    if (!node || !node->name) return;

    if (node_count >= node_capacity) {
        node_capacity = node_capacity == 0 ? 10000 : node_capacity * 2;
        nodes = realloc(nodes, node_capacity * sizeof(NodeInfo));
    }

    NodeInfo* n = &nodes[node_count];
    memset(n, 0, sizeof(NodeInfo));

    build_oid_str(node->oid, node->oidlen, n->oid, sizeof(n->oid));
    strncpy(n->name, node->name, sizeof(n->name) - 1);

    SmiModule* mod = smiGetNodeModule(node);
    if (mod && mod->name) {
        strncpy(n->module, mod->name, sizeof(n->module) - 1);
    }

    n->nodekind = node->nodekind;
    n->status = node->status;
    n->access = node->access;

    SmiType* type = smiGetNodeType(node);
    if (type) {
        n->basetype = type->basetype;
        if (type->name) strncpy(n->typename_, type->name, sizeof(n->typename_) - 1);
    }

    if (node->description) {
        strncpy(n->description, node->description, sizeof(n->description) - 1);
    }

    node_count++;
}

void collect_all_nodes_from_module(const char* module_name) {
    SmiModule* mod = smiGetModule(module_name);
    if (!mod) return;

    SmiNode* node = smiGetFirstNode(mod, SMI_NODEKIND_ANY);
    while (node) {
        collect_node(node);
        node = smiGetNextNode(node, SMI_NODEKIND_ANY);
    }
}

int get_node_count() {
    return node_count;
}

NodeInfo* get_node(int index) {
    if (index < 0 || index >= node_count) return NULL;
    return &nodes[index];
}

void clear_nodes() {
    if (nodes) {
        free(nodes);
        nodes = NULL;
    }
    node_count = 0;
    node_capacity = 0;
}

// Severity level names
const char* severity_name(int severity) {
    switch (severity) {
        case 0: return "fatal";
        case 1: return "severe";
        case 2: return "error";
        case 3: return "minor";
        case 4: return "style";
        case 5: return "warning";
        case 6: return "info";
        default: return "unknown";
    }
}

// Node kind names
const char* nodekind_name(int kind) {
    switch (kind) {
        case SMI_NODEKIND_UNKNOWN: return "unknown";
        case SMI_NODEKIND_NODE: return "node";
        case SMI_NODEKIND_SCALAR: return "scalar";
        case SMI_NODEKIND_TABLE: return "table";
        case SMI_NODEKIND_ROW: return "row";
        case SMI_NODEKIND_COLUMN: return "column";
        case SMI_NODEKIND_NOTIFICATION: return "notification";
        case SMI_NODEKIND_GROUP: return "group";
        case SMI_NODEKIND_COMPLIANCE: return "compliance";
        case SMI_NODEKIND_CAPABILITIES: return "capabilities";
        default: return "other";
    }
}

// Status names
const char* status_name(int status) {
    switch (status) {
        case SMI_STATUS_UNKNOWN: return "";
        case SMI_STATUS_CURRENT: return "current";
        case SMI_STATUS_DEPRECATED: return "deprecated";
        case SMI_STATUS_MANDATORY: return "mandatory";
        case SMI_STATUS_OPTIONAL: return "optional";
        case SMI_STATUS_OBSOLETE: return "obsolete";
        default: return "";
    }
}

// Access names
const char* access_name(int access) {
    switch (access) {
        case SMI_ACCESS_UNKNOWN: return "";
        case SMI_ACCESS_NOT_IMPLEMENTED: return "not-implemented";
        case SMI_ACCESS_NOT_ACCESSIBLE: return "not-accessible";
        case SMI_ACCESS_NOTIFY: return "accessible-for-notify";
        case SMI_ACCESS_READ_ONLY: return "read-only";
        case SMI_ACCESS_READ_WRITE: return "read-write";
        case SMI_ACCESS_INSTALL: return "install";
        case SMI_ACCESS_INSTALL_NOTIFY: return "install-notify";
        case SMI_ACCESS_REPORT_ONLY: return "report-only";
        case SMI_ACCESS_EVENT_ONLY: return "event-only";
        default: return "";
    }
}

// Basetype names
const char* basetype_name(int basetype) {
    switch (basetype) {
        case SMI_BASETYPE_UNKNOWN: return "";
        case SMI_BASETYPE_INTEGER32: return "Integer32";
        case SMI_BASETYPE_OCTETSTRING: return "OCTET STRING";
        case SMI_BASETYPE_OBJECTIDENTIFIER: return "OBJECT IDENTIFIER";
        case SMI_BASETYPE_UNSIGNED32: return "Unsigned32";
        case SMI_BASETYPE_INTEGER64: return "Integer64";
        case SMI_BASETYPE_UNSIGNED64: return "Unsigned64";
        case SMI_BASETYPE_FLOAT32: return "Float32";
        case SMI_BASETYPE_FLOAT64: return "Float64";
        case SMI_BASETYPE_FLOAT128: return "Float128";
        case SMI_BASETYPE_ENUM: return "INTEGER";
        case SMI_BASETYPE_BITS: return "BITS";
        default: return "";
    }
}

*/
import "C"

import (
	"strings"
	"unsafe"
)

// LibsmiDiagnostic represents a diagnostic from libsmi parsing.
type LibsmiDiagnostic struct {
	Path     string
	Line     int
	Severity int
	Message  string
	Tag      string
}

// LibsmiModule holds parsed module information.
type LibsmiModule struct {
	Name         string
	Path         string
	Language     int // 1=SMIv1, 2=SMIv2
	Conformance  int
	Organization string
	ContactInfo  string
	Description  string
}

// LibsmiNode holds parsed node information for comparison.
type LibsmiNode struct {
	OID         string
	Name        string
	Module      string
	NodeKind    string
	Status      string
	Access      string
	BaseType    string
	TypeName    string
	Description string
}

// InitLibsmi initializes libsmi with the given MIB path and error level (0-6).
func InitLibsmi(mibPath string, errorLevel int) {
	cPath := C.CString(mibPath)
	defer C.free(unsafe.Pointer(cPath))
	C.init_libsmi(cPath, C.int(errorLevel))
}

// CleanupLibsmi releases libsmi resources.
func CleanupLibsmi() {
	C.cleanup_libsmi()
}

// ClearErrors clears collected errors for a new parsing session.
func ClearErrors() {
	C.clear_errors()
}

// LoadModule loads a MIB module. Returns true if successful.
func LoadModule(name string) bool {
	cName := C.CString(name)
	defer C.free(unsafe.Pointer(cName))
	return C.load_module(cName) != 0
}

// GetDiagnostics returns all diagnostics collected during parsing.
func GetDiagnostics() []LibsmiDiagnostic {
	count := int(C.get_error_count())
	diags := make([]LibsmiDiagnostic, 0, count)

	for i := 0; i < count; i++ {
		e := C.get_error(C.int(i))
		if e == nil {
			continue
		}

		d := LibsmiDiagnostic{
			Line:     int(e.line),
			Severity: int(e.severity),
		}

		if e.path != nil {
			d.Path = C.GoString(e.path)
		}
		if e.message != nil {
			d.Message = C.GoString(e.message)
		}
		if e.tag != nil {
			d.Tag = C.GoString(e.tag)
		}

		diags = append(diags, d)
	}

	return diags
}

// GetModuleInfo returns parsed module information.
func GetModuleInfo(name string) *LibsmiModule {
	cName := C.CString(name)
	defer C.free(unsafe.Pointer(cName))

	info := C.get_module_info(cName)
	if info == nil {
		return nil
	}

	return &LibsmiModule{
		Name:         C.GoString(&info.name[0]),
		Path:         C.GoString(&info.path[0]),
		Language:     int(info.language),
		Conformance:  int(info.conformance),
		Organization: C.GoString(&info.organization[0]),
		ContactInfo:  C.GoString(&info.contactinfo[0]),
		Description:  C.GoString(&info.description[0]),
	}
}

// CollectNodes collects all nodes from a loaded module.
func CollectNodes(moduleName string) {
	cName := C.CString(moduleName)
	defer C.free(unsafe.Pointer(cName))
	C.collect_all_nodes_from_module(cName)
}

// GetNodes returns all collected nodes.
func GetNodes() []LibsmiNode {
	count := int(C.get_node_count())
	nodes := make([]LibsmiNode, 0, count)

	for i := 0; i < count; i++ {
		n := C.get_node(C.int(i))
		if n == nil {
			continue
		}

		node := LibsmiNode{
			OID:         C.GoString(&n.oid[0]),
			Name:        C.GoString(&n.name[0]),
			Module:      C.GoString(&n.module[0]),
			NodeKind:    C.GoString(C.nodekind_name(n.nodekind)),
			Status:      C.GoString(C.status_name(n.status)),
			Access:      C.GoString(C.access_name(n.access)),
			BaseType:    C.GoString(C.basetype_name(n.basetype)),
			TypeName:    C.GoString(&n.typename_[0]),
			Description: C.GoString(&n.description[0]),
		}

		nodes = append(nodes, node)
	}

	return nodes
}

// ClearNodes clears collected nodes.
func ClearNodes() {
	C.clear_nodes()
}

// SeverityName returns the name for a severity level.
func SeverityName(severity int) string {
	return C.GoString(C.severity_name(C.int(severity)))
}

// BuildMIBPath creates a colon-separated path from multiple directories.
func BuildMIBPath(paths []string) string {
	return strings.Join(paths, ":")
}
