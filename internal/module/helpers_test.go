package module

import "testing"

// requireDef finds the first definition of the given type with the given name.
// Fails the test if not found or wrong type.
func requireDef[T Definition](t *testing.T, mod *Module, name string) T {
	t.Helper()
	for d := range mod.AllDefinitions() {
		if d.DefinitionName() == name {
			v, ok := d.(T)
			if !ok {
				t.Fatalf("%s is %T, not %T", name, d, *new(T))
			}
			return v
		}
	}
	t.Fatalf("definition %q not found", name)
	return *new(T)
}
