package picolangbot

import "testing"

func TestNamedSignatureParameters(t *testing.T) {
	t.Parallel()
	a, b := typeParameter{Name: "A"}, typeParameter{Name: "B"}
	template := Signature{
		Params: []Type{StackType{ElementType: OptionType{ElementType: a}}, b, a},
		Result: OptionType{ElementType: b},
	}
	var inference Inference
	first := inference.Instantiate(template, Source{})
	second := inference.Instantiate(template, Source{})
	for _, constraint := range []struct{ expected, actual Type }{
		{first.Params[0], StackType{ElementType: OptionType{ElementType: BoolType{}}}},
		{first.Params[1], IntegerType{}},
		{second.Params[0], StackType{ElementType: OptionType{ElementType: RotationType{}}}},
		{second.Params[1], UnitType{}},
	} {
		if !UnifyTypes(constraint.expected, constraint.actual) {
			t.Errorf("failed to unify %s and %s", TypeName(constraint.expected), TypeName(constraint.actual))
		}
	}
	for _, tc := range []struct {
		typ  Type
		want string
	}{
		{first.Params[2], "bool"}, {first.Result, "option<integer>"},
		{second.Params[2], "rotation"}, {second.Result, "option<unit>"},
	} {
		if got := TypeName(ResolveType(tc.typ)); got != tc.want {
			t.Errorf("resolved type = %s, want %s", got, tc.want)
		}
	}
	if unresolved := inference.Unresolved(); len(unresolved) != 0 {
		t.Errorf("unresolved parameters: %v", unresolved)
	}
	if template.Params[2] != a {
		t.Error("instantiation mutated the signature template")
	}
}

func TestBuiltinSignatureTablesAreIndependent(t *testing.T) {
	t.Parallel()
	first := BuiltinSignatures()
	first["add"].Params[0] = BoolType{}
	delete(first, "some")
	second := BuiltinSignatures()
	if got := TypeName(second["add"].Params[0]); got != "integer" {
		t.Errorf("mutating one table affected another: %s", got)
	}
	if _, exists := second["some"]; !exists {
		t.Error("deleting from one table affected another")
	}
}
