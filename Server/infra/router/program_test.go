package router

import (
	"context"
	json "encoding/json/v2"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/dvpico/Programming-Learning-AR/Server/domain"
	"github.com/dvpico/Programming-Learning-AR/Server/infra/router/gen/api"
	"github.com/dvpico/Programming-Learning-AR/Server/usecase"
	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
)

const programSource = `{"picolangVersion":0,"picolang":{"v0":{"program":{"main":{"kind":"function","hintId":"00000000-0000-4000-8000-000000000001","params":[],"returnType":"integer","body":{"kind":"block","hintId":"00000000-0000-4000-8000-000000000002","stmts":[{"kind":"return","hintId":"00000000-0000-4000-8000-000000000003","value":{"kind":"literal","hintId":"00000000-0000-4000-8000-000000000004","type":"integer","value":9223372036854775807}}]}}}}}}`

type programStore struct {
	saved   []domain.Program
	updated []domain.Program
	program domain.Program
	getID   string
	getErr  error
	err     error
}

func (s *programStore) Get(_ context.Context, id string) (domain.Program, error) {
	s.getID = id
	return s.program, s.getErr
}

func (s *programStore) Update(_ context.Context, p domain.Program) (domain.Program, error) {
	s.updated = append(s.updated, p)
	return p, s.err
}

func (s *programStore) Create(_ context.Context, p domain.Program) error {
	s.saved = append(s.saved, p)
	return s.err
}

func (s *programStore) GetByTutorialID(_ context.Context, _, _ string) ([]domain.Program, error) {
	return nil, s.err
}

func TestCreateProgram(t *testing.T) {
	t.Parallel()
	valid := `{"id":"client-id","name":"draft","isBattle":true,"source":` + programSource + `}`
	for _, tt := range []struct {
		name, body          string
		cookie              bool
		unknownLength       bool
		sessionErr, saveErr error
		want                int
	}{
		{name: "battle", body: valid, cookie: true, want: 201},
		{name: "tutorial", body: strings.Replace(valid, `"isBattle":true`, `"isBattle":false,"tutorialID":"tutorial-1"`, 1), cookie: true, want: 201},
		{name: "missing session", body: valid, want: 401},
		{name: "expired session", body: valid, cookie: true, sessionErr: usecase.ErrUnauthenticated, want: 401},
		{name: "session failure", body: valid, cookie: true, sessionErr: errors.New("database unavailable"), want: 500},
		{name: "save failure", body: valid, cookie: true, saveErr: errors.New("database unavailable"), want: 500},
		{name: "missing battle flag", body: strings.Replace(valid, `"isBattle":true,`, "", 1), cookie: true, want: 400},
		{name: "missing name", body: strings.Replace(valid, `"name":"draft",`, "", 1), cookie: true, want: 400},
		{name: "blank name", body: strings.Replace(valid, `"draft"`, `" "`, 1), cookie: true, want: 400},
		{name: "missing tutorial", body: strings.Replace(valid, `"isBattle":true`, `"isBattle":false`, 1), cookie: true, want: 400},
		{name: "battle with tutorial", body: strings.Replace(valid, `"isBattle":true`, `"isBattle":true,"tutorialID":"tutorial-1"`, 1), cookie: true, want: 400},
		{name: "null tutorial", body: strings.Replace(valid, `"isBattle":true`, `"isBattle":true,"tutorialID":null`, 1), cookie: true, want: 400},
		{name: "invalid source", body: strings.Replace(valid, programSource, `{}`, 1), cookie: true, want: 400},
		{name: "missing source", body: `{"name":"draft","isBattle":true}`, cookie: true, want: 400},
		{name: "null source", body: strings.Replace(valid, programSource, `null`, 1), cookie: true, want: 400},
		{name: "unknown field", body: strings.Replace(valid, `"name":`, `"owner":"attacker","name":`, 1), cookie: true, want: 400},
		{name: "duplicate key", body: strings.Replace(valid, `"isBattle":true`, `"isBattle":false,"isBattle":true`, 1), cookie: true, want: 400},
		{name: "multiple bodies", body: valid + `{}`, cookie: true, want: 400},
		{name: "oversized", body: strings.Repeat(" ", 1<<20) + valid, cookie: true, want: 413},
		{name: "oversized without content length", body: strings.Repeat(" ", 1<<20) + valid, cookie: true, unknownLength: true, want: 400},
		{name: "exactly at body limit", body: strings.Repeat(" ", (1<<20)-len(valid)) + valid, cookie: true, want: 201},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			store := &programStore{err: tt.saveErr}
			r := NewRouter(Config{Programs: usecase.NewPrograms(store, ProgramSourceValidator{})})
			e := echo.New()
			var handlerErr error
			e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
				return func(ctx *echo.Context) error {
					handlerErr = next(ctx)
					return handlerErr
				}
			})
			api.RegisterHandlersWithOptions(e, r, api.RegisterHandlersOptions{
				BaseURL: "/api",
				OperationMiddlewares: map[string][]echo.MiddlewareFunc{
					"createProgram": {Authentication(sessionLookupStub{err: tt.sessionErr}), middleware.BodyLimit(1 << 20)},
				},
			})
			req := httptest.NewRequest(http.MethodPost, "/api/program", strings.NewReader(tt.body))
			if tt.unknownLength {
				req.ContentLength = -1
			}
			req.Header.Set("Content-Type", "application/json")
			if tt.cookie {
				req.AddCookie(&http.Cookie{Name: "session", Value: "session-id", Secure: true, HttpOnly: true, SameSite: http.SameSiteLaxMode})
			}
			res := httptest.NewRecorder()
			e.ServeHTTP(res, req)
			if res.Code != tt.want {
				t.Errorf("status = %d, want %d; body = %s", res.Code, tt.want, res.Body.String())
			}
			for _, cause := range []error{tt.sessionErr, tt.saveErr} {
				if cause != nil && (!errors.Is(handlerErr, cause) || errors.Unwrap(handlerErr) == nil) {
					t.Errorf("handler error = %v, want wrapped cause %v", handlerErr, cause)
				}
			}
			if tt.want == http.StatusInternalServerError && strings.TrimSpace(res.Body.String()) != `{"message":"Internal Server Error"}` {
				t.Errorf("unexpected public error response: %s", res.Body.String())
			}
			if tt.want != 201 && tt.saveErr == nil && len(store.saved) != 0 {
				t.Error("invalid request was saved")
			}
			if tt.want != 201 {
				return
			}
			if len(store.saved) != 1 {
				t.Errorf("saved %d programs, want 1", len(store.saved))
				return
			}
			p := store.saved[0]
			if p.GoogleSubject != "verified-subject" || p.ID == "" || p.ID == "client-id" {
				t.Errorf("incorrect identity: %+v", p)
			}
			var response map[string]any
			if err := json.Unmarshal(res.Body.Bytes(), &response); err != nil {
				t.Errorf("invalid response: %v", err)
				return
			}
			if response["id"] != p.ID || response["name"] != "draft" {
				t.Errorf("response = %v", response)
			}
			if _, exists := response["GoogleSubject"]; exists {
				t.Error("response exposes owner")
			}
			if !strings.Contains(res.Body.String(), "9223372036854775807") {
				t.Error("response lost integer precision")
			}
			if !strings.Contains(string(p.Source), "9223372036854775807") {
				t.Error("stored source lost integer precision")
			}
		})
	}
}
