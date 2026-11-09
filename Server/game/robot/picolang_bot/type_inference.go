package picolangbot

import "fmt"

// invalidType suppresses secondary type errors after an earlier diagnostic.
// Neither it nor inferenceVariable may occur in a successful TypedProgram.
type invalidType struct{}

func (invalidType) annotation()   {}
func (invalidType) languageType() {}

// InvalidType marks an already diagnosed expression, suppressing secondary
// errors. It must never occur in a successful TypedProgram.
func InvalidType() Type { return invalidType{} }

func IsInvalidType(t Type) bool {
	_, invalid := t.(invalidType)
	return invalid
}

// Inference owns fresh type parameters for one function analysis. Keeping the
// implementation here lets intermediate types remain in Type's closed family.
type Inference struct {
	variables []*inferenceVariable
}

func (inference *Inference) Instantiate(sig Signature, source Source) Signature {
	var variable *inferenceVariable
	params := make([]Type, len(sig.Params))
	for i, param := range sig.Params {
		params[i] = inference.instantiate(param, source, &variable)
	}
	return Signature{Params: params, Result: inference.instantiate(sig.Result, source, &variable)}
}

func (inference *Inference) instantiate(t Type, source Source, variable **inferenceVariable) Type {
	switch t := t.(type) {
	case typeParameter:
		if *variable == nil {
			*variable = &inferenceVariable{source: source}
			inference.variables = append(inference.variables, *variable)
		}
		return *variable
	case StackType:
		return StackType{ElementType: inference.instantiate(t.ElementType, source, variable)}
	case OptionType:
		return OptionType{ElementType: inference.instantiate(t.ElementType, source, variable)}
	default:
		return t
	}
}

// Unresolved returns one source hint for each unconstrained type parameter.
func (inference *Inference) Unresolved() []Source {
	var sources []Source
	seen := make(map[*inferenceVariable]struct{})
	for _, variable := range inference.variables {
		if root, free := prune(variable).(*inferenceVariable); free {
			if _, reported := seen[root]; !reported {
				sources = append(sources, root.source)
				seen[root] = struct{}{}
			}
		}
	}
	return sources
}

type inferenceVariable struct {
	source  Source
	binding Type
}

func (*inferenceVariable) annotation()   {}
func (*inferenceVariable) languageType() {}

// CanonicalType copies and validates declared types, also detecting cycles in
// pointer-based compound types supplied through the public AST API.
func CanonicalType(t Type) (Type, bool) {
	return canonicalType(t, make(map[Type]struct{}))
}

func canonicalType(t Type, visiting map[Type]struct{}) (Type, bool) {
	switch t.(type) {
	case *StackType, *OptionType:
		if _, cycle := visiting[t]; cycle {
			return invalidType{}, false
		}
		visiting[t] = struct{}{}
		defer delete(visiting, t)
	}
	switch t := NodeValue(t).(type) {
	case BoolType, IntegerType, UnitType, TerrainType, RotationType, DistanceTargetType:
		return t, true
	case StackType:
		element, ok := canonicalType(t.ElementType, visiting)
		if ok {
			return StackType{ElementType: element}, true
		}
	case OptionType:
		element, ok := canonicalType(t.ElementType, visiting)
		if ok {
			return OptionType{ElementType: element}, true
		}
	}
	return invalidType{}, false
}

func prune(t Type) Type {
	if variable, ok := t.(*inferenceVariable); ok && variable.binding != nil {
		variable.binding = prune(variable.binding)
		return variable.binding
	}
	return t
}

func occurs(variable *inferenceVariable, t Type) bool {
	t = prune(t)
	if t == variable {
		return true
	}
	switch t := t.(type) {
	case StackType:
		return occurs(variable, t.ElementType)
	case OptionType:
		return occurs(variable, t.ElementType)
	}
	return false
}

// UnifyTypes constrains type parameters, returning false on a mismatch. After
// failure, unconstrained parts are marked invalid to avoid cascading errors.
func UnifyTypes(expected, actual Type) bool {
	if unifyTypes(expected, actual) {
		return true
	}
	invalidateInference(expected)
	invalidateInference(actual)
	return false
}

func invalidateInference(t Type) {
	switch t := prune(t).(type) {
	case *inferenceVariable:
		t.binding = invalidType{}
	case StackType:
		invalidateInference(t.ElementType)
	case OptionType:
		invalidateInference(t.ElementType)
	}
}

func unifyTypes(expected, actual Type) bool {
	expected, actual = prune(expected), prune(actual)
	if _, bad := expected.(invalidType); bad {
		invalidateInference(actual)
		return true
	}
	if _, bad := actual.(invalidType); bad {
		invalidateInference(expected)
		return true
	}
	if variable, ok := expected.(*inferenceVariable); ok {
		if variable == actual {
			return true
		}
		if occurs(variable, actual) {
			return false
		}
		variable.binding = actual
		return true
	}
	if _, ok := actual.(*inferenceVariable); ok {
		return unifyTypes(actual, expected)
	}
	switch expected := expected.(type) {
	case BoolType:
		_, ok := actual.(BoolType)
		return ok
	case IntegerType:
		_, ok := actual.(IntegerType)
		return ok
	case UnitType:
		_, ok := actual.(UnitType)
		return ok
	case TerrainType:
		_, ok := actual.(TerrainType)
		return ok
	case RotationType:
		_, ok := actual.(RotationType)
		return ok
	case DistanceTargetType:
		_, ok := actual.(DistanceTargetType)
		return ok
	case StackType:
		stack, ok := actual.(StackType)
		return ok && unifyTypes(expected.ElementType, stack.ElementType)
	case OptionType:
		option, ok := actual.(OptionType)
		return ok && unifyTypes(expected.ElementType, option.ElementType)
	default:
		return false
	}
}

// ResolveType replaces solved parameters with concrete value types.
func ResolveType(t Type) Type {
	t = prune(t)
	switch t := t.(type) {
	case StackType:
		return StackType{ElementType: ResolveType(t.ElementType)}
	case OptionType:
		return OptionType{ElementType: ResolveType(t.ElementType)}
	}
	return t
}

func TypeName(t Type) string {
	switch t := prune(t).(type) {
	case BoolType:
		return "bool"
	case IntegerType:
		return "integer"
	case UnitType:
		return "unit"
	case TerrainType:
		return "terrain"
	case RotationType:
		return "rotation"
	case DistanceTargetType:
		return "distance_target"
	case StackType:
		return "stack<" + TypeName(t.ElementType) + ">"
	case OptionType:
		return "option<" + TypeName(t.ElementType) + ">"
	case *inferenceVariable:
		return "?"
	case invalidType:
		return "<invalid>"
	default:
		return fmt.Sprintf("<invalid %T>", t)
	}
}
