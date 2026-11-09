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

type tutorialRunnerStub struct {
	TutorialService
	subject, tutorialID, programID string
	err                            error
}

func (s *tutorialRunnerStub) Run(_ context.Context, subject, tutorialID, programID string) (domain.TutorialRun, error) {
	s.subject, s.tutorialID, s.programID = subject, tutorialID, programID
	return domain.TutorialRun{ID: "019a5aa3-c000-7000-8000-000000000001", FieldID: "field-b"}, s.err
}

func TestRunTutorial(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name, body string
		noSession  bool
		err        error
		want       int
	}{
		{name: "created", body: `{"programId":"program-1"}`, want: 201},
		{name: "unauthenticated", body: `{"programId":"program-1"}`, noSession: true, want: 401},
		{name: "required program", body: `{}`, want: 400},
		{name: "null program", body: `{"programId":null}`, want: 400},
		{name: "unknown member", body: `{"programId":"program-1","fieldId":"chosen-by-client"}`, want: 400},
		{name: "multiple bodies", body: `{"programId":"program-1"}{}`, want: 400},
		{name: "invalid program", body: `{"programId":"program-1"}`, err: domain.ErrInvalidProgram, want: 400},
		{name: "forbidden", body: `{"programId":"program-1"}`, err: usecase.ErrForbidden, want: 403},
		{name: "not found", body: `{"programId":"program-1"}`, err: usecase.ErrNotFound, want: 404},
		{name: "execution not configured", body: `{"programId":"program-1"}`, err: usecase.ErrRunUnavailable, want: 501},
		{name: "internal error", body: `{"programId":"program-1"}`, err: errors.New("database unavailable"), want: 500},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			runner := &tutorialRunnerStub{err: tt.err}
			e := echo.New()
			api.RegisterHandlersWithOptions(e, NewRouter(Config{Tutorials: runner}), api.RegisterHandlersOptions{
				OperationMiddlewares: map[string][]echo.MiddlewareFunc{"runTutorial": {Authentication(sessionLookupStub{}), middleware.BodyLimit(1 << 20)}},
			})
			req := httptest.NewRequest(http.MethodPost, "/tutorial/tutorial-1/run", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			if !tt.noSession {
				req.AddCookie(&http.Cookie{Name: "session", Value: "session-id", Secure: true, HttpOnly: true, SameSite: http.SameSiteLaxMode})
			}
			res := httptest.NewRecorder()
			e.ServeHTTP(res, req)
			if res.Code != tt.want {
				t.Errorf("status = %d, want %d; body = %s", res.Code, tt.want, res.Body.String())
			}
			if tt.want == 201 {
				var response api.TutorialRunCreated
				if err := json.Unmarshal(res.Body.Bytes(), &response); err != nil {
					t.Errorf("decode response: %v", err)
					return
				}
				if response.RunId.String() != "019a5aa3-c000-7000-8000-000000000001" || response.FieldId != "field-b" || runner.subject != "verified-subject" || runner.tutorialID != "tutorial-1" || runner.programID != "program-1" {
					t.Errorf("response = %+v, runner = %+v", response, runner)
				}
			} else if tt.err == nil && runner.subject != "" {
				t.Error("invalid request reached the use case")
			}
		})
	}
}
