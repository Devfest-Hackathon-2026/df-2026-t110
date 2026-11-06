package picolangbot

import "github.com/dvpico/Programming-Learning-AR/Server/game"

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

type VectorType struct {
	ElementType Type
}

func (BoolType) annotation()     {}
func (IntegerType) annotation()  {}
func (UnitType) annotation()     {}
func (TerrainType) annotation()  {}
func (RotationType) annotation() {}
func (VectorType) annotation()   {}

func (BoolType) languageType()     {}
func (IntegerType) languageType()  {}
func (UnitType) languageType()     {}
func (TerrainType) languageType()  {}
func (RotationType) languageType() {}
func (VectorType) languageType()   {}

func (BoolType) primitiveType()     {}
func (IntegerType) primitiveType()  {}
func (UnitType) primitiveType()     {}
func (TerrainType) primitiveType()  {}
func (RotationType) primitiveType() {}
func (VectorType) compoundType()    {}

// Literal contains only source literals. Terrain and vector values are
// produced by builtins, so they have no literal representation.
type Literal interface{ literal() }

type BoolLiteral struct{ Value bool }
type IntegerLiteral struct{ Value int64 }
type UnitLiteral struct{}
type RotationLiteral struct{ Value game.Rotation }

func (BoolLiteral) literal()     {}
func (IntegerLiteral) literal()  {}
func (UnitLiteral) literal()     {}
func (RotationLiteral) literal() {}
