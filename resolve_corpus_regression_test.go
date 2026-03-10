package gomib

import (
	"testing"

	"github.com/golangsnmp/gomib/internal/testutil"
	"github.com/golangsnmp/gomib/internal/types"
	"github.com/golangsnmp/gomib/mib"
)

func TestCorpusPartialImportForwardingResolvesPRVTServIndexes(t *testing.T) {
	m := loadCorpusMIB(t, "PRVT-SERV-MIB", WithResolverStrictness(mib.ResolverNormal))

	testutil.Equal(t, 0, countModuleDiagnostics(m, "PRVT-SERV-MIB", types.DiagOidOrphan),
		"valid forwarded serviceAccessSwitch import should prevent orphaned PRVT-SERV-MIB OIDs")
	testutil.Equal(t, 0, countModuleDiagnostics(m, "PRVT-SERV-MIB", types.DiagIndexUnresolved),
		"row indexes should resolve once the forwarded root OID import resolves")

	row := requireObject(t, m, "sapBaseInfoEntry")
	indexes := row.Index()
	testutil.Len(t, indexes, 3, "sapBaseInfoEntry index count")
	if len(indexes) != 3 {
		return
	}

	testutil.NotNil(t, indexes[0].Object, "svcId index object")
	testutil.NotNil(t, indexes[1].Object, "sapPortId index object")
	testutil.NotNil(t, indexes[2].Object, "sapEncapValue index object")
}
