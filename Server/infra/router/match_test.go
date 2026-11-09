package router

import (
	"context"
	json "encoding/json/v2"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/dvpico/Programming-Learning-AR/Server/domain"
	"github.com/dvpico/Programming-Learning-AR/Server/infra/router/gen/api"
	"github.com/dvpico/Programming-Learning-AR/Server/usecase"
	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
)

type matchCreatorStub struct {
	subject string
	ids     []string
	err     error
}

func (s *matchCreatorStub) Create(_ context.Context, subject string, ids []string) (domain.Match, error) {
	s.subject, s.ids = subject, ids
	return domain.Match{
		ID: "match-id", RunID: "019a5aa3-c000-7000-8000-000000000001",
		FieldID: "field", OpponentName: "Opponent",
	}, s.err
}

func TestPostMatch(t *testing.T) {
	t.Parallel()
	const valid = `[{"programId":"b"},{"programId":"a"},{"programId":"b"}]`
	for _, tt := range []struct {
		name, body string
		noSession  bool
		err        error
		want       int
	}{
		{name: "created", body: valid, want: 201},
		{name: "no session", body: valid, noSession: true, want: 401},
		{name: "null", body: `null`, want: 400},
		{name: "empty", body: `[]`, want: 400},
		{name: "too few", body: `[{"programId":"a"}]`, want: 400},
		{name: "too many", body: `[{"programId":"a"},{"programId":"a"},{"programId":"a"},{"programId":"a"}]`, want: 400},
		{name: "missing ID", body: `[{},{"programId":"a"},{"programId":"a"}]`, want: 400},
		{name: "null ID", body: `[{"programId":null},{"programId":"a"},{"programId":"a"}]`, want: 400},
		{name: "blank ID", body: `[{"programId":" "},{"programId":"a"},{"programId":"a"}]`, want: 400},
		{name: "unknown member", body: `[{"programId":"a","opponent":"chosen"},{"programId":"a"},{"programId":"a"}]`, want: 400},
		{name: "multiple bodies", body: valid + `[]`, want: 400},
		{name: "large body", body: `[{"programId":"` + strings.Repeat("x", 1<<20) + `"}]`, want: 413},
		{name: "invalid program", body: valid, err: domain.ErrInvalidProgram, want: 400},
		{name: "forbidden", body: valid, err: usecase.ErrForbidden, want: 403},
		{name: "not found", body: valid, err: usecase.ErrNotFound, want: 404},
		{name: "no opponent", body: valid, err: usecase.ErrNoOpponent, want: 409},
		{name: "internal error", body: valid, err: errors.New("database unavailable"), want: 500},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			creator := &matchCreatorStub{err: tt.err}
			e := echo.New()
			api.RegisterHandlersWithOptions(e, NewRouter(Config{Matches: creator}), api.RegisterHandlersOptions{
				BaseURL: "/api", OperationMiddlewares: map[string][]echo.MiddlewareFunc{
					"createMatch": {Authentication(sessionLookupStub{}), middleware.BodyLimit(1 << 20)},
				},
			})
			req := httptest.NewRequest(http.MethodPost, "/api/match", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			if !tt.noSession {
				req.AddCookie(&http.Cookie{Name: "session", Value: "session-id", Secure: true, HttpOnly: true, SameSite: http.SameSiteLaxMode})
			}
			res := httptest.NewRecorder()
			e.ServeHTTP(res, req)
			if res.Code != tt.want {
				t.Errorf("status=%d, want %d; body=%s", res.Code, tt.want, res.Body.String())
			}
			if tt.want == 201 {
				var match api.Match
				if err := json.Unmarshal(res.Body.Bytes(), &match); err != nil {
					t.Errorf("decode response: %v", err)
					return
				}
				if match.Id != "match-id" || match.RunId.Version() != 7 || match.Status == nil || *match.Status != api.Finished || match.FieldId != "field" || match.OpponentName != "Opponent" {
					t.Errorf("match=%+v", match)
				}
				if creator.subject != "verified-subject" || !reflect.DeepEqual(creator.ids, []string{"b", "a", "b"}) {
					t.Errorf("creator=%+v", creator)
				}
			} else if tt.err == nil && creator.subject != "" {
				t.Error("invalid request reached the use case")
			}
		})
	}
}
