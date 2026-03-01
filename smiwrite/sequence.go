package smiwrite

import (
	"fmt"

	"github.com/golangsnmp/gomib/mib"
)

// emitSequence writes a reconstructed SEQUENCE type for a row object.
// The SEQUENCE is reconstructed from the row's column children.
func (e *emitter) emitSequence(row *mib.Object) error {
	cols := row.Columns()
	if len(cols) == 0 {
		return nil
	}

	seqName := sequenceTypeName(row.Name())
	if _, err := fmt.Fprintf(e.w, "%s ::= SEQUENCE {\n", seqName); err != nil {
		return err
	}

	for i, col := range cols {
		typeName := sequenceFieldType(col)
		comma := ","
		if i == len(cols)-1 {
			comma = ""
		}
		if _, err := fmt.Fprintf(e.w, "    %s %s%s\n", col.Name(), typeName, comma); err != nil {
			return err
		}
	}

	_, err := fmt.Fprintln(e.w, "}")
	return err
}

// sequenceFieldType returns the type name for a SEQUENCE field.
func sequenceFieldType(col *mib.Object) string {
	t := col.Type()
	if t == nil {
		return "INTEGER"
	}
	if t.Name() != "" {
		return t.Name()
	}
	return baseTypeSyntax(t.EffectiveBase())
}
