package router

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/dvpico/Programming-Learning-AR/Server/domain"
	"github.com/dvpico/Programming-Learning-AR/Server/infra/router/gen/api"
	"github.com/labstack/echo/v5"
)

func (r *Router) ListRooms(ctx *echo.Context) error {
	return errNotImplemented()
}

func (r *Router) GetRoom(ctx *echo.Context, roomID api.RoomId) error {
	return errNotImplemented()
}

// JoinRoomEvents streams the room's events as Server-Sent Events until the
// client disconnects, with a comment line every heartbeat interval.
func (r *Router) JoinRoomEvents(ctx *echo.Context, roomID api.RoomId) error {
	reqCtx := ctx.Request().Context()
	sub, err := r.roomEvents.Join(reqCtx, roomID)
	if err != nil {
		return err
	}
	defer sub.Close()

	w := ctx.Response()
	h := w.Header()
	h.Set("Content-Type", "text/event-stream")
	h.Set("Cache-Control", "no-cache")
	h.Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)
	flusher := http.NewResponseController(w)

	heartbeat := time.NewTicker(r.cfg.HeartbeatInterval)
	defer heartbeat.Stop()
	// Write errors mean the client is gone; the response is already committed,
	// so there is nothing left to report and the stream just ends.
	if err := flusher.Flush(); err != nil {
		return nil
	}
	for {
		select {
		case <-reqCtx.Done():
			return nil
		case <-heartbeat.C:
			if _, err := io.WriteString(w, ": heartbeat\n\n"); err != nil {
				return nil
			}
		case event, ok := <-sub.Events():
			if !ok {
				return nil
			}
			message, err := formatEvent(event)
			if err != nil {
				// One malformed event should not cut the client off from later ones.
				ctx.Logger().Error("skip unencodable room event", "room", roomID, "event", event.Name, "error", err)
				continue
			}
			if _, err := io.WriteString(w, message); err != nil {
				return nil
			}
		}
		if err := flusher.Flush(); err != nil {
			return nil
		}
	}
}

// formatEvent frames event as one SSE message; JSON encoding keeps data on a single line.
func formatEvent(event domain.RoomEvent) (string, error) {
	data, err := json.Marshal(event.Data)
	if err != nil {
		return "", fmt.Errorf("encode data: %w", err)
	}
	return fmt.Sprintf("event: %s\ndata: %s\n\n", event.Name, data), nil
}

func (r *Router) StartRoomMatch(ctx *echo.Context, roomID api.RoomId, matchID api.MatchId) error {
	return errNotImplemented()
}
