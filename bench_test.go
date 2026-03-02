package gomib

import (
	"context"
	"testing"

	"github.com/golangsnmp/gomib/internal/testutil"
)

func BenchmarkLoadAllCorpus(b *testing.B) {
	src := mustDir(b, testutil.PrimaryCorpusDir())

	ctx := context.Background()

	b.ResetTimer()
	for b.Loop() {
		m, err := Load(ctx, WithSource(src))
		if err != nil {
			b.Fatalf("Load failed: %v", err)
		}
		_ = m
	}
}

func BenchmarkLoadSingleMIB(b *testing.B) {
	src := mustDir(b, testutil.PrimaryCorpusDir())

	ctx := context.Background()

	b.ResetTimer()
	for b.Loop() {
		m, err := Load(ctx, WithSource(src), WithModules("IF-MIB"))
		if err != nil {
			b.Fatalf("Load failed: %v", err)
		}
		_ = m
	}
}
