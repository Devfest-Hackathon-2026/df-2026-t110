package router

import (
	"context"
	"encoding/json/jsontext"
	json "encoding/json/v2"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/dvpico/Programming-Learning-AR/Server/infra/repo"
	"github.com/dvpico/Programming-Learning-AR/Server/infra/router/gen/api"
	"github.com/dvpico/Programming-Learning-AR/Server/usecase"
	"github.com/labstack/echo/v5"
)

const fieldTestPage = `id: lesson
title: Lesson
description: A lesson
stage:
  variants:
    - id: field-a
      width: 2
      height: 1
      terrain: [[land, rock]]
      objects:
        - type: goal
          position: {x: 1, y: 0}
      robot:
        position: {x: 0, y: 0}
        direction: east
      drillLimit: 1
    - id: field-b
      width: 1
      height: 2
      terrain: [[sea], [land]]
      objects:
        - type: luggage
          id: cargo
          position: {x: 0, y: 1}
      robot:
        position: {x: 0, y: 1}
        direction: north
      drillLimit: 2
  clearCondition:
    type: reach_goal
`

const fieldA = `{"id":"field-a","width":2,"height":1,"terrain":[["land","rock"]],"objects":[{"type":"goal","position":{"x":1,"y":0}}],"robot":{"position":{"x":0,"y":0},"direction":"east"},"drillLimit":1}`
const fieldB = `{"id":"field-b","width":1,"height":2,"terrain":[["sea"],["land"]],"objects":[{"type":"luggage","id":"cargo","position":{"x":0,"y":1}}],"robot":{"position":{"x":0,"y":1},"direction":"north"},"drillLimit":2}`

func testFields(t *testing.T) *usecase.Fields {
	t.Helper()
	textbooks, err := repo.NewTextbooks(t.Context(), fstest.MapFS{"lesson.yaml": {Data: []byte(fieldTestPage)}})
	if err != nil {
		t.Fatal(err)
	}
	return usecase.NewFields(textbooks)
}

func assertFieldJSON(t *testing.T, got []byte, want string) {
	t.Helper()
	var actual, expected any
	if err := json.Unmarshal(got, &actual); err != nil {
		t.Fatalf("decode actual field: %v", err)
	}
	if err := json.Unmarshal([]byte(want), &expected); err != nil {
		t.Fatalf("decode expected field: %v", err)
	}
	if !reflect.DeepEqual(actual, expected) {
		t.Errorf("field = %s, want %s", got, want)
	}
}

func TestFieldRoutesWithTextbooks(t *testing.T) {
	t.Parallel()

	e := echo.New()
	api.RegisterHandlersWithBaseURL(e, NewRouter(Config{Fields: testFields(t)}), "/api")

	cases := []struct {
		name   string
		id     string
		status int
		want   string
	}{
		{"first variant", "field-a", http.StatusOK, fieldA},
		{"second variant", "field-b", http.StatusOK, fieldB},
		{"tutorial id", "lesson", http.StatusNotFound, ""},
		{"unknown id", "missing", http.StatusNotFound, ""},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			res := httptest.NewRecorder()

			e.ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/api/field/"+tt.id, nil))
			if res.Code != tt.status {
				t.Fatalf("status = %d, want %d; body = %s", res.Code, tt.status, res.Body.String())
			}
			if tt.status == http.StatusOK {
				assertFieldJSON(t, res.Body.Bytes(), tt.want)
			}
		})
	}
}

type fieldGenerationStub struct {
	reserves int
	inputs   []usecase.ProgramGenerationInput
}

func (s *fieldGenerationStub) Reserve(context.Context, string, int) error {
	s.reserves++
	return nil
}

func (s *fieldGenerationStub) GenerateProgram(_ context.Context, in usecase.ProgramGenerationInput) (jsontext.Value, error) {
	s.inputs = append(s.inputs, in)
	return jsontext.Value(`{"picolangVersion":0}`), nil
}

func (*fieldGenerationStub) ValidateGeneratedProgram(_ context.Context, program jsontext.Value) (jsontext.Value, error) {
	return program, nil
}

func TestGenerationUsesTextbookFieldID(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		id     string
		status int
		want   string
	}{
		{"first variant", "field-a", http.StatusCreated, fieldA},
		{"second variant", "field-b", http.StatusCreated, fieldB},
		{"tutorial id", "lesson", http.StatusNotFound, ""},
		{"unknown id", "missing", http.StatusNotFound, ""},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			stub := &fieldGenerationStub{}
			generation := usecase.NewProgramGeneration(usecase.ProgramGenerationDeps{Fields: testFields(t), Quota: stub, Generator: stub, Validator: stub, MaxTokens: 100})
			e := echo.New()
			api.RegisterHandlersWithOptions(e, NewRouter(Config{ProgramGeneration: generation}), api.RegisterHandlersOptions{
				BaseURL:              "/api",
				OperationMiddlewares: map[string][]echo.MiddlewareFunc{"createProgramWithVibe": {Authentication(sessionLookupStub{})}},
			})
			req := httptest.NewRequest(http.MethodPost, "/api/program/create-with-vibe", strings.NewReader(`{"prompt":"go","fieldId":"`+tt.id+`"}`))
			req.Header.Set("Content-Type", "application/json")
			req.AddCookie(&http.Cookie{Name: "session", Value: "session-id"})
			res := httptest.NewRecorder()

			e.ServeHTTP(res, req)
			if res.Code != tt.status {
				t.Fatalf("status = %d, want %d; body = %s", res.Code, tt.status, res.Body.String())
			}

			if tt.status != http.StatusCreated {
				if stub.reserves != 0 || len(stub.inputs) != 0 {
					t.Errorf("quota reserves = %d, generator calls = %d; want none for unknown field", stub.reserves, len(stub.inputs))
				}
				return
			}
			if len(stub.inputs) != 1 {
				t.Fatalf("generator calls = %d, want 1", len(stub.inputs))
			}
			assertFieldJSON(t, stub.inputs[0].Field, tt.want)
		})
	}
}
