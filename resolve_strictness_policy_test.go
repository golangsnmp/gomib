package gomib

import (
	"testing"

	"github.com/golangsnmp/gomib/internal/testutil"
	"github.com/golangsnmp/gomib/mib"
)

// resolve_strictness_policy_test.go covers the resolver-level fallback policy
// and a few compact OID cases that pin down those boundaries.

func TestResolverStrictnessBoundaries(t *testing.T) {
	tests := []struct {
		level           mib.ResolverStrictness
		wantConstrained bool
		wantGlobal      bool
	}{
		{mib.ResolverStrict, false, false},
		{mib.ResolverNormal, true, false},
		{mib.ResolverPermissive, true, true},
	}

	for _, tt := range tests {
		t.Run(tt.level.String(), func(t *testing.T) {
			testutil.Equal(t, tt.wantConstrained, tt.level.AllowConstrainedFallbacks(),
				"AllowConstrainedFallbacks at level %d", tt.level)
			testutil.Equal(t, tt.wantGlobal, tt.level.AllowGlobalFallbacks(),
				"AllowGlobalFallbacks at level %d", tt.level)
		})
	}
}

func TestOIDGlobalRootStrictness(t *testing.T) {
	tests := []struct {
		strictnessCase
		wantUnresolved bool
		wantObject     bool
	}{
		{strictnessCase: strictnessCase{name: "strict", level: mib.ResolverStrict}, wantUnresolved: true},
		{strictnessCase: strictnessCase{name: "normal", level: mib.ResolverNormal}, wantObject: true},
		{strictnessCase: strictnessCase{name: "permissive", level: mib.ResolverPermissive}, wantObject: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := loadViolationMIB(t, "MISSING-IMPORT-TEST-MIB", tt.level)
			unresolvedOids := unresolvedSymbols(m, "MISSING-IMPORT-TEST-MIB", mib.UnresolvedOID)

			testutil.Equal(t, tt.wantUnresolved, unresolvedOids["enterprises"],
				"unexpected unresolved state for enterprises at %s", tt.name)

			obj := m.Object("testObject")
			if !tt.wantObject {
				testutil.Nil(t, obj, "testObject should not resolve at %s", tt.name)
				return
			}

			testutil.NotNil(t, obj, "testObject should resolve at %s", tt.name)
			if obj != nil {
				testutil.Equal(t, "1.3.6.1.4.1.99999.1", obj.OID().String(),
					"testObject OID at %s", tt.name)
			}
		})
	}
}

func TestImportForwardingOidResolution(t *testing.T) {
	m := loadAtStrictness(t, "PROBLEM-FORWARDING-MIB", mib.ResolverNormal)

	obj := m.Object("problemForwardedOidObject")
	testutil.NotNil(t, obj, "Object(problemForwardedOidObject)")

	testutil.Equal(t, "1.3.6.1.4.1.99998.20.1.10", obj.OID().String(),
		"OID should resolve through forwarded parent")
}
