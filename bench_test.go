package gomib

import (
	"context"
	"testing"
)

func BenchmarkLoadAllCorpus(b *testing.B) {
	src := mustDirTree(b, "testdata/corpus/primary")

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
	src := mustDirTree(b, "testdata/corpus/primary")

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
