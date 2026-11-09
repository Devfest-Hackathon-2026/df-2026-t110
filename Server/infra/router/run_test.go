package router

import (
	"context"
	"encoding/json/jsontext"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dvpico/Programming-Learning-AR/Server/infra/router/gen/api"
	"github.com/dvpico/Programming-Learning-AR/Server/usecase"
	"github.com/labstack/echo/v5"
)

type runResultStub struct {
	result       jsontext.Value
	err          error
	subject, id  string
	cancel       context.CancelFunc
	calls        int
	pendingFirst bool
}

func (s *runResultStub) Result(_ context.Context, subject, id string) (jsontext.Value, error) {
	s.subject, s.id = subject, id
	s.calls++
	if s.pendingFirst && s.calls == 1 {
		return nil, nil
	}
	if s.cancel != nil {
		s.cancel()
	}
	return s.result, s.err
}

func TestRunEvents(t *testing.T) {
	t.Parallel()
	const id = "019a5aa3-c000-7000-8000-000000000001"
	for _, tt := range []struct {
		name               string
		err                error
		noSession, pending bool
		wait               bool
		want               int
	}{
		{name: "completed", want: 200},
		{name: "pending then completed", wait: true, want: 200},
		{name: "missing session", noSession: true, want: 401},
		{name: "other owner", err: usecase.ErrForbidden, want: 403},
		{name: "unknown run", err: usecase.ErrNotFound, want: 404},
		{name: "storage failure", err: errors.New("database unavailable"), want: 500},
		{name: "pending subscription canceled", pending: true, want: 200},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			getter := &runResultStub{result: jsontext.Value("{\n\"history\":[],\"summary\":{\"tick\":9223372036854775807}}"), err: tt.err, pendingFirst: tt.wait}
			req := httptest.NewRequest(http.MethodGet, "/run/"+id+"/events", nil)
			if !tt.noSession {
				req.AddCookie(&http.Cookie{Name: "session", Value: "s", Secure: true, HttpOnly: true, SameSite: http.SameSiteLaxMode})
			}
			if tt.pending {
				ctx, cancel := context.WithCancel(req.Context())
				defer cancel()
				req = req.WithContext(ctx)
				getter.result, getter.cancel = nil, cancel
			}
			e := echo.New()
			api.RegisterHandlersWithOptions(e, NewRouter(Config{RunResults: getter}), api.RegisterHandlersOptions{
				OperationMiddlewares: map[string][]echo.MiddlewareFunc{"runEvents": {Authentication(sessionLookupStub{})}},
			})
			res := httptest.NewRecorder()
			e.ServeHTTP(res, req)
			if res.Code != tt.want {
				t.Errorf("status = %d, want %d; %s", res.Code, tt.want, res.Body.String())
			}
			if tt.want != 200 {
				return
			}
			if getter.subject != "verified-subject" || getter.id != id {
				t.Errorf("lookup identity = %+v", getter)
			}
			if res.Header().Get("Content-Type") != "text/event-stream" || res.Header().Get("Cache-Control") != "no-store" {
				t.Errorf("SSE headers = %v", res.Header())
			}
			if tt.pending {
				if res.Body.Len() != 0 {
					t.Errorf("pending result = %s", res.Body.String())
				}
				return
			}
			want := "event: result\ndata: {\"history\":[],\"summary\":{\"tick\":9223372036854775807}}\n\n"
			if res.Body.String() != want {
				t.Errorf("SSE = %q, want %q", res.Body.String(), want)
			}
		})
	}
}
