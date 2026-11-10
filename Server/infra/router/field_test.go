package router

import (
	"context"
	json "encoding/json/v2"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/dvpico/Programming-Learning-AR/Server/infra/router/gen/api"
	"github.com/dvpico/Programming-Learning-AR/Server/infra/router/gen/schema"
	"github.com/dvpico/Programming-Learning-AR/Server/usecase"
	"github.com/labstack/echo/v5"
)

var stubField = api.Field{
	Id:      "variant-1",
	Width:   2,
	Height:  1,
	Terrain: [][]schema.Terrain{{schema.TerrainLand, schema.TerrainLand}},
	Objects: []schema.FieldObject{},
	Robot:   schema.FieldDefinitionRobot{Direction: schema.DirectionEast},
}

type fieldServiceStub struct{}

func (fieldServiceStub) Get(_ context.Context, id string) (api.Field, error) {
	switch id {
	case stubField.Id:
		return stubField, nil
	case "error":
		return api.Field{}, errors.New("failed")
	default:
		return api.Field{}, usecase.ErrNotFound
	}
}

func TestGetField(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		id     string
		status int
	}{
		{"found", stubField.Id, http.StatusOK},
		{"not found", "missing", http.StatusNotFound},
		{"service error", "error", http.StatusInternalServerError},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			api.RegisterHandlers(e, NewRouter(Config{Fields: fieldServiceStub{}}))
			rec := httptest.NewRecorder()

			e.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/field/"+tt.id, nil))
			if rec.Code != tt.status {
				t.Fatalf("status = %d, want %d; body = %s", rec.Code, tt.status, rec.Body.String())
			}
			if tt.status != http.StatusOK {
				return
			}

			var got api.Field
			if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
				t.Fatalf("decode body %s: %v", rec.Body.String(), err)
			}
			if !reflect.DeepEqual(got, stubField) {
				t.Errorf("field = %+v, want %+v", got, stubField)
			}
		})
	}
}
