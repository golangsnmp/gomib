package resolver

import (
	"testing"

	"github.com/golangsnmp/gomib/internal/module"
	"github.com/golangsnmp/gomib/internal/types"
	"github.com/golangsnmp/gomib/mib"
)

func newTestContext() *ResolverContext {
	return newResolverContext(nil, nil, mib.DefaultConfig())
}

func TestRecordUnresolvedSeverityConsistency(t *testing.T) {
	// All RecordUnresolved* methods should emit diagnostics at SeverityError.
	// Unresolved references represent failed symbol resolution regardless of
	// category, so the severity should be uniform.

	mod := &module.Module{Name: "TEST-MIB"}
	span := types.Span{}

	tests := []struct {
		name string
		code string
		emit func(c *ResolverContext)
	}{
		{
			name: "import",
			code: "import-not-found",
			emit: func(c *ResolverContext) {
				c.RecordUnresolvedImport(mod, "OTHER-MIB", "someSymbol", "not found", span)
			},
		},
		{
			name: "import module not found",
			code: "import-module-not-found",
			emit: func(c *ResolverContext) {
				c.RecordUnresolvedImport(mod, "MISSING-MIB", "someSymbol", "module not found", span)
			},
		},
		{
			name: "type",
			code: "type-unknown",
			emit: func(c *ResolverContext) {
				c.RecordUnresolvedType(mod, "myType", "UnknownType", span)
			},
		},
		{
			name: "oid",
			code: "oid-orphan",
			emit: func(c *ResolverContext) {
				c.RecordUnresolvedOid(mod, "myObject", "unknownParent", span)
			},
		},
		{
			name: "index",
			code: "index-unresolved",
			emit: func(c *ResolverContext) {
				c.RecordUnresolvedIndex(mod, "myRow", "missingIndex", span)
			},
		},
		{
			name: "notification object",
			code: "objects-unresolved",
			emit: func(c *ResolverContext) {
				c.RecordUnresolvedNotificationObject(mod, "myNotif", "missingObject", span)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := newTestContext()
			tt.emit(ctx)

			diags := ctx.Diagnostics()
			var found bool
			for _, d := range diags {
				if d.Code == tt.code {
					found = true
					if d.Severity != mib.SeverityError {
						t.Errorf("diagnostic %q has severity %d, want %d (SeverityError)",
							tt.code, d.Severity, mib.SeverityError)
					}
					if d.Module != "TEST-MIB" {
						t.Errorf("diagnostic %q has module %q, want %q",
							tt.code, d.Module, "TEST-MIB")
					}
				}
			}
			if !found {
				t.Errorf("no diagnostic with code %q emitted", tt.code)
			}
		})
	}
}
