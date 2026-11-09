package router

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/dvpico/Programming-Learning-AR/Server/infra/router/gen/api"
	"github.com/dvpico/Programming-Learning-AR/Server/usecase"
	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
)

type generationServiceStub struct {
	request usecase.ProgramGenerationRequest
	subject string
	err     error
	calls   int
}

func (s *generationServiceStub) GenerateWithOptions(_ context.Context, subject string, request usecase.ProgramGenerationRequest) (json.RawMessage, error) {
	s.calls++
	s.subject, s.request = subject, request
	return json.RawMessage(`{"picolangVersion":0}`), s.err
}

func TestCreateProgramWithVibe(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name, body string
		cookie     bool
		configured bool
		serviceErr error
		want       int
	}{
		{"tutorial", `{"prompt":"go forward","fieldId":"first-step"}`, true, true, nil, 201},
		{"battle without field", `{"prompt":"collect luggage","isBattle":true}`, true, true, nil, 201},
		{"missing session", `{"prompt":"go forward"}`, false, true, nil, 401},
		{"missing prompt", `{}`, true, true, nil, 400},
		{"null field", `{"prompt":"go","fieldId":null}`, true, true, nil, 400},
		{"blank prompt", `{"prompt":" "}`, true, true, usecase.ErrInvalidGenerationRequest, 400},
		{"unknown member", `{"prompt":"go","owner":"attacker"}`, true, true, nil, 400},
		{"missing field", `{"prompt":"go","fieldId":"missing"}`, true, true, usecase.ErrNotFound, 404},
		{"quota", `{"prompt":"go"}`, true, true, usecase.ErrGenerationQuotaExceeded, 429},
		{"invalid output", `{"prompt":"go"}`, true, true, usecase.ErrInvalidGeneratedProgram, 422},
		{"timeout", `{"prompt":"go"}`, true, true, context.DeadlineExceeded, 504},
		{"not configured", `{"prompt":"go"}`, true, false, nil, 501},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			service := &generationServiceStub{err: tt.serviceErr}
			cfg := Config{}
			if tt.configured {
				cfg.ProgramGeneration = service
			}
			e := echo.New()
			api.RegisterHandlersWithOptions(e, NewRouter(cfg), api.RegisterHandlersOptions{
				BaseURL: "/api",
				OperationMiddlewares: map[string][]echo.MiddlewareFunc{
					"createProgramWithVibe": {Authentication(sessionLookupStub{}), middleware.BodyLimit(16 << 10)},
				},
			})
			req := httptest.NewRequest(http.MethodPost, "/api/program/create-with-vibe", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			if tt.cookie {
				req.AddCookie(&http.Cookie{Name: "session", Value: "session-id"})
			}
			res := httptest.NewRecorder()
			e.ServeHTTP(res, req)
			if res.Code != tt.want {
				t.Errorf("status = %d, want %d; body = %s", res.Code, tt.want, res.Body.String())
			}
			if tt.want == 201 {
				if service.calls != 1 || service.subject != "verified-subject" || res.Body.String() != `{"picolangVersion":0}` {
					t.Errorf("generation = %+v, body = %s", service, res.Body.String())
				}
				if tt.name == "battle without field" && (!service.request.IsBattle || service.request.FieldID != "") {
					t.Errorf("battle options = %+v", service.request)
				}
			} else if !errors.Is(tt.serviceErr, usecase.ErrInvalidGenerationRequest) && tt.serviceErr == nil && service.calls != 0 {
				t.Errorf("invalid request reached generation: %+v", service)
			}
		})
	}
}
