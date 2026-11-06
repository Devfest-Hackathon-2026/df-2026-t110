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

// typeParameter denotes the vector builtin's T, freshly instantiated per call.
type typeParameter struct{}

func (typeParameter) annotation()   {}
func (typeParameter) languageType() {}

// BuiltinSignatures returns an independent signature table for the builtins.
func BuiltinSignatures() map[Name]Signature {
	b, i, u, t, r := BoolType{}, IntegerType{}, UnitType{}, TerrainType{}, RotationType{}
	x := typeParameter{}
	v := VectorType{ElementType: x}
	return map[Name]Signature{
		"not":                {[]Type{b}, b},
		"and":                {[]Type{b, b}, b},
		"or":                 {[]Type{b, b}, b},
		"bool_eq":            {[]Type{b, b}, b},
		"bool_neq":           {[]Type{b, b}, b},
		"add":                {[]Type{i, i}, i},
		"minus":              {[]Type{i}, i},
		"sub":                {[]Type{i, i}, i},
		"mul":                {[]Type{i, i}, i},
		"div":                {[]Type{i, i}, i},
		"integer_eq":         {[]Type{i, i}, b},
		"integer_neq":        {[]Type{i, i}, b},
		"gt":                 {[]Type{i, i}, b},
		"lt":                 {[]Type{i, i}, b},
		"geq":                {[]Type{i, i}, b},
		"leq":                {[]Type{i, i}, b},
		"bool_to_integer":    {[]Type{b}, i},
		"integer_to_bool":    {[]Type{i}, b},
		"random_bool":        {nil, b},
		"random_integer":     {[]Type{i, i}, i},
		"new":                {nil, v},
		"push_back":          {[]Type{v, x}, v},
		"pop_back":           {[]Type{v}, v},
		"get_back":           {[]Type{v}, x},
		"size":               {[]Type{v}, i},
		"is_land":            {[]Type{t}, b},
		"is_goal":            {[]Type{t}, b},
		"is_own_territory":   {[]Type{t}, b},
		"is_enemy_territory": {[]Type{t}, b},
		"is_checkpoint":      {[]Type{t}, b},
		"is_luggage":         {[]Type{t}, b},
		"is_sea":             {[]Type{t}, b},
		"is_rock":            {[]Type{t}, b},
		"forward":            {nil, b},
		"back":               {nil, b},
		"rotate":             {[]Type{r}, u},
		"wait":               {[]Type{i}, u},
		"pick_up":            {nil, b},
		"release":            {nil, b},
		"drill":              {nil, b},
		"drill_durability":   {nil, i},
		"distance":           {[]Type{DistanceTargetType{}}, i},
		"search_front":       {nil, t},
		"search_now":         {nil, t},
		"print_integer":      {[]Type{i}, u},
		"print_bool":         {[]Type{b}, u},
	}
}
