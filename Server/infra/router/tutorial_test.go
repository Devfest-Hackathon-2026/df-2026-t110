package router

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/dvpico/Programming-Learning-AR/Server/infra/repo"
	"github.com/dvpico/Programming-Learning-AR/Server/infra/router/gen/api"
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
	e := echo.New()
	textbooks, err := repo.NewTextbooks(context.Background(), pages)
	if err != nil {
		t.Fatal(err)
	}
	api.RegisterHandlersWithBaseURL(e, NewRouter(Config{Tutorials: textbooks}), "/api")
	for _, tt := range []struct {
		path       string
		wantStatus int
		wantBody   string
	}{
		{"/api/tutorial", http.StatusOK, `"id":"first-step"`},
		{"/api/tutorial/first-step", http.StatusOK, `"description":"Move to the goal"`},
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
