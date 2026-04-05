package resolver

import (
	"testing"

	"github.com/golangsnmp/gomib/internal/module"
	"github.com/golangsnmp/gomib/internal/testutil"
)

func TestSymbolTable_GetSet(t *testing.T) {
	st := make(symbolTable[string])
	mod := &module.Module{Name: "A"}

	val, ok := st.get(mod, "x")
	testutil.False(t, ok, "get on empty table")
	testutil.Equal(t, "", val, "zero value")

	st.set(mod, "x", "hello")
	val, ok = st.get(mod, "x")
	testutil.True(t, ok, "get after set")
	testutil.Equal(t, "hello", val, "value")

	_, ok = st.get(mod, "y")
	testutil.False(t, ok, "get missing key")
}

func TestSymbolTable_GetNilModule(t *testing.T) {
	st := make(symbolTable[int])
	val, ok := st.get(nil, "x")
	testutil.False(t, ok, "get with nil module")
	testutil.Equal(t, 0, val, "zero value for nil module")
}

func TestSymbolTable_Has(t *testing.T) {
	st := make(symbolTable[int])
	mod := &module.Module{Name: "A"}

	testutil.False(t, st.has(mod, "x"), "has on empty")
	st.set(mod, "x", 42)
	testutil.True(t, st.has(mod, "x"), "has after set")
	testutil.False(t, st.has(mod, "y"), "has missing key")
}

func TestSymbolTable_ForModule(t *testing.T) {
	st := make(symbolTable[string])
	mod := &module.Module{Name: "A"}

	testutil.Nil(t, st.forModule(mod), "forModule on empty")

	st.set(mod, "x", "val")
	m := st.forModule(mod)
	testutil.NotNil(t, m, "forModule after set")
	testutil.Equal(t, "val", m["x"], "forModule value")
}

func TestSymbolTable_SetOverwrite(t *testing.T) {
	st := make(symbolTable[int])
	mod := &module.Module{Name: "A"}

	st.set(mod, "x", 1)
	st.set(mod, "x", 2)
	val, _ := st.get(mod, "x")
	testutil.Equal(t, 2, val, "overwritten value")
}

func TestSymbolTable_MultipleModules(t *testing.T) {
	st := make(symbolTable[string])
	modA := &module.Module{Name: "A"}
	modB := &module.Module{Name: "B"}

	st.set(modA, "x", "fromA")
	st.set(modB, "x", "fromB")

	val, _ := st.get(modA, "x")
	testutil.Equal(t, "fromA", val, "module A value")

	val, _ = st.get(modB, "x")
	testutil.Equal(t, "fromB", val, "module B value")
}
