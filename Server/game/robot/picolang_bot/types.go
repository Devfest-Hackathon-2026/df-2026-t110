package picolangbot

import (
	values "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/value"
)

// Type is a picolang value type and the annotation on a checked expression.
// Parsed type declarations also use this closed family of language types.
type Type interface {
	Annotation
	languageType()
}

type PrimitiveType interface {
	Type
	primitiveType()
}

type CompoundType interface {
	Type
	compoundType()
}

type BoolType struct{}
type IntegerType struct{}
type UnitType struct{}
type TerrainType struct{}
type RotationType struct{}
type DistanceTargetType struct{}

type StackType struct {
	ElementType Type
}

type OptionType struct {
	ElementType Type
	CompoundType
}

func (BoolType) annotation()           {}
func (IntegerType) annotation()        {}
func (UnitType) annotation()           {}
func (TerrainType) annotation()        {}
func (RotationType) annotation()       {}
func (DistanceTargetType) annotation() {}
func (StackType) annotation()          {}

func (BoolType) languageType()           {}
func (IntegerType) languageType()        {}
func (UnitType) languageType()           {}
func (TerrainType) languageType()        {}
func (RotationType) languageType()       {}
func (DistanceTargetType) languageType() {}
func (StackType) languageType()          {}

func (BoolType) primitiveType()           {}
func (IntegerType) primitiveType()        {}
func (UnitType) primitiveType()           {}
func (TerrainType) primitiveType()        {}
func (RotationType) primitiveType()       {}
func (DistanceTargetType) primitiveType() {}
func (StackType) compoundType()           {}

// Literal contains only source literals. Terrain, stack and option values are
// produced by builtins, so they have no literal representation.
type Literal interface{ literal() }

type BoolLiteral struct{ Value bool }
type IntegerLiteral struct{ Value int64 }
type UnitLiteral struct{}
type RotationLiteral struct{ Value values.Rotation }
type DistanceTargetLiteral struct{ Value values.DistanceTarget }

func (BoolLiteral) literal()           {}
func (IntegerLiteral) literal()        {}
func (UnitLiteral) literal()           {}
func (RotationLiteral) literal()       {}
func (DistanceTargetLiteral) literal() {}
