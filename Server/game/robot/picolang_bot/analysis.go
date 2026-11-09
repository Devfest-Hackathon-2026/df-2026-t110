package picolangbot

import (
	"maps"
	"reflect"
	"slices"
)

// FunctionNames returns names in deterministic order for semantic analysis.
func FunctionNames[V Variable, A Annotation](program Program[V, A]) []Name {
	return slices.Sorted(maps.Keys(program.Functions))
}

// NodeValue normalizes non-nil node pointers before matching concrete variants;
// node interfaces admit both values and pointers. Nil pointers stay invalid.
func NodeValue[T any](value T) T {
	v := reflect.ValueOf(value)
	if v.IsValid() && v.Kind() == reflect.Pointer && !v.IsNil() {
		if result, ok := v.Elem().Interface().(T); ok {
			return result
		}
	}
	return value
}

// SourceOf returns a node's source, using the parent for absent nodes.
func SourceOf(node any, fallback Source) Source {
	v := reflect.ValueOf(node)
	if !v.IsValid() || (v.Kind() == reflect.Pointer && v.IsNil()) {
		return fallback
	}
	if n, ok := node.(interface{ source() Source }); ok {
		return n.source()
	}
	return fallback
}

// Signature describes a function. BuiltinSignatures may contain private type
// parameters; use Inference.Instantiate to obtain fresh types for each call.
type Signature struct {
	Params []Type
	Result Type
}

//go:generate go run ./internal/genbuiltins

// typeParameter names a generic builtin parameter, freshly instantiated per call.
type typeParameter struct {
	Name string
	Type
}
