package gomib

import (
	"io/fs"

	"github.com/golangsnmp/gomib/internal/module"
)

// embeddedSource serves embedded base module SMI text as a Source.
// It acts as the lowest-priority fallback when no local copy is found.
type embeddedSource struct{}

func (s embeddedSource) Find(name string) (FindResult, error) {
	content := module.EmbeddedContent(name)
	if content == nil {
		return FindResult{}, fs.ErrNotExist
	}
	return FindResult{
		Content: content,
		Path:    "embedded:" + name,
	}, nil
}

func (s embeddedSource) ListModules() ([]string, error) {
	return module.EmbeddedModuleNames(), nil
}
