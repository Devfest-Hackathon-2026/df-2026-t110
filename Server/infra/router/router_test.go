package router

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/dvpico/Programming-Learning-AR/Server/infra/pubsub"
	"github.com/dvpico/Programming-Learning-AR/Server/infra/router/gen/api"
	"github.com/dvpico/Programming-Learning-AR/Server/usecase"
	"github.com/labstack/echo/v5"
)

type fixture struct {
	e          *echo.Echo
	broker     *pubsub.RoomBroker
	roomEvents *usecase.RoomEvents
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	broker := pubsub.NewRoomBroker(0)
	roomEvents := usecase.NewRoomEvents(broker)
	r := NewRouter(roomEvents, Config{HeartbeatInterval: 20 * time.Millisecond})
	e := echo.New()
	api.RegisterHandlers(e, r)
	return &fixture{e: e, broker: broker, roomEvents: roomEvents}
}

func TestRouterHandlersAreNotImplementedYet(t *testing.T) {
	t.Parallel()
	f := newFixture(t)

	tests := []struct {
		name   string
		method string
		path   string
		want   int
	}{
		{name: "registered route is not implemented yet", method: http.MethodGet, path: "/tutorial", want: http.StatusNotImplemented},
		{name: "path parameter route is not implemented yet", method: http.MethodGet, path: "/field/field-1", want: http.StatusNotImplemented},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			response := httptest.NewRecorder()
			f.e.ServeHTTP(response, httptest.NewRequest(tt.method, tt.path, nil))
			if response.Code != tt.want {
				t.Errorf("%s %s status = %d, want %d", tt.method, tt.path, response.Code, tt.want)
			}
		})
	}
}
