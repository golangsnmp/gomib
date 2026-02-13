package mibimpl

import "github.com/golangsnmp/gomib/mib"

// mapSlice converts a concrete slice to an interface slice using f.
func mapSlice[S any, T any](src []S, f func(S) T) []T {
	if src == nil {
		return nil
	}
	result := make([]T, len(src))
	for i, v := range src {
		result[i] = f(v)
	}
	return result
}

// objectsByKind returns objects whose node matches the given kind.
func objectsByKind(objs []*Object, kind mib.Kind) []mib.Object {
	var result []mib.Object
	for _, obj := range objs {
		if obj.node != nil && obj.node.kind == kind {
			result = append(result, obj)
		}
	}
	return result
}
