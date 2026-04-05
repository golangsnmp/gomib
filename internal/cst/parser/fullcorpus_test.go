package parser_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/golangsnmp/gomib/internal/cst"
	"github.com/golangsnmp/gomib/internal/cst/parser"
	"github.com/golangsnmp/gomib/internal/types"
)

func TestFullCorpusRoundTrip(t *testing.T) {
	dirs := []string{
		filepath.Join("..", "..", "..", "testdata", "corpus", "primary"),
		filepath.Join("..", "..", "..", "testdata", "corpus", "problems"),
	}
	for _, dir := range dirs {
		err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return err
			}
			if filepath.Ext(path) == ".md" {
				return nil
			}
			rel, _ := filepath.Rel(dirs[0], path)
			if rel == path {
				rel, _ = filepath.Rel(dirs[1], path)
			}
			t.Run(rel, func(t *testing.T) {
				source, err := os.ReadFile(path)
				if err != nil {
					t.Fatal(err)
				}
				p := parser.New(source, nil, types.DefaultConfig())
				file := p.ParseModule()
				got := cst.ReconstructText(file, source)
				if string(got) != string(source) {
					for i := 0; i < len(got) && i < len(source); i++ {
						if got[i] != source[i] {
							start := max(0, i-40)
							end := min(len(source), i+40)
							t.Fatalf("diff at byte %d:\nsource: ...%q...\ngot:    ...%q...",
								i, source[start:end], got[max(0, i-40):min(len(got), i+40)])
						}
					}
					t.Fatalf("length: source=%d got=%d", len(source), len(got))
				}
			})
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
}
