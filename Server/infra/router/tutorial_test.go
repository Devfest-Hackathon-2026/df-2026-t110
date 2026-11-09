package router

import (
	"context"
	"encoding/json/jsontext"
	json "encoding/json/v2"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/dvpico/Programming-Learning-AR/Server/domain"
	"github.com/dvpico/Programming-Learning-AR/Server/infra/repo"
	"github.com/dvpico/Programming-Learning-AR/Server/infra/router/gen/api"
	"github.com/dvpico/Programming-Learning-AR/Server/usecase"
	"github.com/labstack/echo/v5"
)

func TestTutorialRoutes(t *testing.T) {
	t.Parallel()
	pages := fstest.MapFS{
		"01.yaml": {Data: []byte(`id: first-step
title: First step
description: Move to the goal
stage:
  variants:
    - id: first-step-1
      width: 2
      height: 1
      terrain: [[land, land]]
      objects:
        - type: goal
          position: {x: 1, y: 0}
      robot:
        position: {x: 0, y: 0}
        direction: east
      drillLimit: 0
  clearCondition:
    type: reach_goal
`)},
	}
	pages["02.yaml"] = &fstest.MapFile{Data: []byte(strings.ReplaceAll(string(pages["01.yaml"].Data), "first-step", "turn"))}
	pages["14.yaml"] = &fstest.MapFile{Data: []byte(strings.ReplaceAll(string(pages["01.yaml"].Data), "first-step", "deliver-luggage"))}
	e := echo.New()
	textbooks, err := repo.NewTextbooks(context.Background(), pages)
	if err != nil {
		t.Fatal(err)
	}
	api.RegisterHandlersWithBaseURL(e, NewRouter(Config{Tutorials: usecase.NewTutorials(textbooks, nil, nil)}), "/api")
	for _, tt := range []struct {
		path       string
		wantStatus int
		wantBody   string
	}{
		{"/api/tutorial", http.StatusOK, `[{"id":"first-step","title":"First step"},{"id":"turn","title":"First step"},{"id":"deliver-luggage","title":"First step"}]`},
		{"/api/tutorial/first-step", http.StatusOK, `"description":"Move to the goal"`},
		{"/api/tutorial/turn", http.StatusOK, `"id":"turn"`},
		{"/api/tutorial/deliver-luggage", http.StatusOK, `"id":"deliver-luggage"`},
		{"/api/tutorial/missing", http.StatusNotFound, `Not Found`},
	} {
		t.Run(tt.path, func(t *testing.T) {
			t.Parallel()
			response := httptest.NewRecorder()
			e.ServeHTTP(response, httptest.NewRequest(http.MethodGet, tt.path, nil))
			if response.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d; body = %s", response.Code, tt.wantStatus, response.Body.String())
			}
			if !strings.Contains(response.Body.String(), tt.wantBody) {
				t.Errorf("body = %s, want %q", response.Body.String(), tt.wantBody)
			}
		})
	}
}

type tutorialProgramServiceStub struct {
	ProgramService
	programs   []domain.Program
	tutorialID string
	subject    string
	err        error
}

func (s *tutorialProgramServiceStub) ListByTutorialID(_ context.Context, subject, tutorialID string) ([]domain.Program, error) {
	s.tutorialID = tutorialID
	s.subject = subject
	return s.programs, s.err
}

func TestListTutorialPrograms(t *testing.T) {
	t.Parallel()
	tutorialID := "first-step"
	type savedProgram struct {
		ID         string         `json:"id"`
		Name       string         `json:"name"`
		IsBattle   bool           `json:"isBattle"`
		Source     jsontext.Value `json:"source"`
		TutorialID *string        `json:"tutorialID"`
	}
	for _, tc := range []struct {
		name      string
		programs  []domain.Program
		err       error
		status    int
		noSession bool
		want      []savedProgram
	}{
		{
			name: "matches",
			programs: []domain.Program{
				{ID: "draft-1", GoogleSubject: "owner", Name: "first draft", Source: jsontext.Value(programSource), TutorialID: &tutorialID},
				{ID: "draft-2", GoogleSubject: "owner", Name: "second draft", Source: jsontext.Value(programSource), TutorialID: &tutorialID},
			},
			status: http.StatusOK,
			want: []savedProgram{
				{ID: "draft-1", Name: "first draft", Source: jsontext.Value(programSource), TutorialID: &tutorialID},
				{ID: "draft-2", Name: "second draft", Source: jsontext.Value(programSource), TutorialID: &tutorialID},
			},
		},
		{name: "no matches", status: http.StatusOK, want: []savedProgram{}},
		{name: "missing session", status: http.StatusUnauthorized, noSession: true},
		{name: "service failure", err: errors.New("database unavailable"), status: http.StatusInternalServerError},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			service := &tutorialProgramServiceStub{programs: tc.programs, err: tc.err}
			e := echo.New()
			api.RegisterHandlersWithOptions(e, NewRouter(Config{Programs: service}), api.RegisterHandlersOptions{
				BaseURL: "/api",
				OperationMiddlewares: map[string][]echo.MiddlewareFunc{
					"listTutorialPrograms": {Authentication(sessionLookupStub{})},
				},
			})
			request := httptest.NewRequest(http.MethodGet, "/api/tutorial/"+tutorialID+"/program", nil)
			if !tc.noSession {
				request.AddCookie(&http.Cookie{Name: "session", Value: "session-id", Secure: true, HttpOnly: true, SameSite: http.SameSiteLaxMode})
			}
			response := httptest.NewRecorder()
			e.ServeHTTP(response, request)
			if tc.noSession {
				if service.subject != "" || service.tutorialID != "" {
					t.Error("service called without a session")
				}
			} else if service.subject != "verified-subject" || service.tutorialID != tutorialID {
				t.Errorf("service arguments = (%q, %q), want (verified-subject, %q)", service.subject, service.tutorialID, tutorialID)
			}
			if response.Code != tc.status {
				t.Errorf("status = %d, want %d; body = %s", response.Code, tc.status, response.Body.String())
			}
			if tc.want == nil {
				return
			}
			var got []savedProgram
			if err := json.Unmarshal(response.Body.Bytes(), &got, json.RejectUnknownMembers(true)); err != nil {
				t.Fatalf("decode response: %v", err)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("decoded programs = %+v, want %+v", got, tc.want)
			}
		})
	}
}
