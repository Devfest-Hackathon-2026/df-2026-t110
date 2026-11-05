package domain

import "github.com/dvpico/Programming-Learning-AR/Server/infra/router/gen/schema"

// Tutorial is a guided stage with an explanation.
//
// Stage reuses the type generated from Schema/field/stage.schema.yaml rather than
// a hand-written copy: that schema is the shared contract, and the generated
// package holds plain data types without infrastructure dependencies.
type Tutorial struct {
	ID          string
	Title       string
	Description string
	Stage       schema.StageDefinition
}
