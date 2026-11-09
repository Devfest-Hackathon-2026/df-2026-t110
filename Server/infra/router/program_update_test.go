package router

import (
	"encoding/json/jsontext"
	json "encoding/json/v2"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/dvpico/Programming-Learning-AR/Server/domain"
	"github.com/dvpico/Programming-Learning-AR/Server/infra/router/gen/api"
	"github.com/dvpico/Programming-Learning-AR/Server/usecase"
	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
)

func TestUpdateProgram(t *testing.T) {
	t.Parallel()
	source, err := os.ReadFile("../tutorialrun/testdata/forward.json")
	if err != nil {
		t.Fatal(err)
	}
	compact := jsontext.Value(source)
	if err := compact.Compact(); err != nil {
		t.Fatal(err)
	}
	validSource := string(compact)
	valid := `{"name":"edited","source":` + validSource + `}`
	for _, tt := range []struct {
		name, body, owner           string
		cookie, unknownLength       bool
		getErr, sessionErr, saveErr error
		want                        int
	}{
		{name: "update", body: valid, cookie: true, want: 200},
		{name: "no session", body: valid, want: 401},
		{name: "other", body: valid, cookie: true, owner: "other", want: 403},
		{name: "missing", body: valid, cookie: true, getErr: usecase.ErrNotFound, want: 404},
		{name: "save error", body: valid, cookie: true, saveErr: errors.New("failure"), want: 500},
		{name: "missing source", body: `{"name":"edited"}`, cookie: true, want: 400},
		{name: "blank name", body: strings.Replace(valid, `"edited"`, `" "`, 1), cookie: true, want: 400},
		{name: "syntax", body: strings.Replace(valid, validSource, `{}`, 1), cookie: true, want: 400},
		{name: "body id", body: strings.Replace(valid, `"name":`, `"id":"x","name":`, 1), cookie: true, want: 400},
		{name: "purpose", body: strings.Replace(valid, `"name":`, `"isBattle":true,"name":`, 1), cookie: true, want: 400},
		{name: "tutorial", body: strings.Replace(valid, `"name":`, `"tutorialID":"x","name":`, 1), cookie: true, want: 400},
		{name: "unknown function", body: strings.Replace(valid, `"func":"print_integer"`, `"func":"missing"`, 1), cookie: true, want: 400},
		{name: "invalid main", body: strings.Replace(valid, `"returnType":"unit"`, `"returnType":"integer"`, 1), cookie: true, want: 400},
		{name: "multiple bodies", body: valid + `{}`, cookie: true, want: 400},
		{name: "type error", body: strings.Replace(valid, `"func":"print_integer"`, `"func":"print_bool"`, 1), cookie: true, want: 400},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			owner := tt.owner
			if owner == "" {
				owner = "verified-subject"
			}
			store := &programStore{program: domain.Program{ID: "path-id", GoogleSubject: owner, IsBattle: true}, getErr: tt.getErr, err: tt.saveErr}
			r := NewRouter(Config{Programs: usecase.NewPrograms(store, ProgramSourceValidator{})})
			e := echo.New()
			api.RegisterHandlersWithOptions(e, r, api.RegisterHandlersOptions{
				BaseURL: "/api",
				OperationMiddlewares: map[string][]echo.MiddlewareFunc{
					"updateProgram": {Authentication(sessionLookupStub{err: tt.sessionErr}), middleware.BodyLimit(1 << 20)},
				},
			})
			req := httptest.NewRequest(http.MethodPut, "/api/program/path-id", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			if tt.unknownLength {
				req.ContentLength = -1
			}
			if tt.cookie {
				req.AddCookie(&http.Cookie{Name: "session", Value: "session-id", Secure: true, HttpOnly: true, SameSite: http.SameSiteLaxMode})
			}
			res := httptest.NewRecorder()
			e.ServeHTTP(res, req)
			if res.Code != tt.want {
				t.Errorf("status = %d, want %d; body = %s", res.Code, tt.want, res.Body.String())
			}
			if tt.want != http.StatusOK {
				if tt.saveErr == nil && len(store.updated) != 0 {
					t.Error("updated an invalid or unauthorized draft")
				}
				if strings.Contains(res.Body.String(), "unavailable") || strings.Contains(res.Body.String(), "other-subject") {
					t.Errorf("error response exposes internal data: %s", res.Body.String())
				}
				var publicError map[string]string
				if err := json.Unmarshal(res.Body.Bytes(), &publicError); err != nil {
					t.Errorf("decode error response: %v", err)
				} else if len(publicError) != 1 || publicError["message"] != http.StatusText(tt.want) {
					t.Errorf("unexpected public error response: %s", res.Body.String())
				}
				return
			}
			if len(store.updated) != 1 || len(store.saved) != 0 {
				t.Errorf("updated = %d, created = %d", len(store.updated), len(store.saved))
				return
			}
			p := store.updated[0]
			if store.getID != "path-id" || p.ID != "path-id" || p.GoogleSubject != "verified-subject" {
				t.Errorf("incorrect target or owner: lookup = %q, updated = %+v", store.getID, p)
			}
			var response map[string]jsontext.Value
			if err := json.Unmarshal(res.Body.Bytes(), &response); err != nil {
				t.Errorf("decode response: %v", err)
				return
			}
			if string(response["id"]) != `"path-id"` || string(response["name"]) != `"edited"` || string(response["isBattle"]) != strconv.FormatBool(p.IsBattle) {
				t.Errorf("response = %s", res.Body.String())
			}
			if len(response) != 4 && (len(response) != 5 || p.TutorialID == nil) {
				t.Errorf("unexpected response fields: %s", res.Body.String())
			}
			if p.TutorialID != nil && string(response["tutorialID"]) != `"`+*p.TutorialID+`"` {
				t.Errorf("tutorial ID missing from response: %s", res.Body.String())
			}
			if !strings.Contains(string(response["source"]), "9223372036854775807") || !strings.Contains(string(p.Source), "9223372036854775807") {
				t.Error("source lost integer precision")
			}
		})
	}
}
