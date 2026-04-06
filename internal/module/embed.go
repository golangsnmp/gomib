package module

import "embed"

//go:embed embedded/*
var embeddedFS embed.FS

// EmbeddedContent returns the SMI source bytes for a named module,
// or nil if not embedded.
func EmbeddedContent(name string) []byte {
	data, err := embeddedFS.ReadFile("embedded/" + name)
	if err != nil {
		return nil
	}
	return data
}

// EmbeddedModuleNames returns the names of all embedded modules.
func EmbeddedModuleNames() []string {
	entries, err := embeddedFS.ReadDir("embedded")
	if err != nil {
		return nil
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if !e.IsDir() {
			names = append(names, e.Name())
		}
	}
	return names
}
