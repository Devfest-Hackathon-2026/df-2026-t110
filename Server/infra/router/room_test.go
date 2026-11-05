package router

import (
	"bufio"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/dvpico/Programming-Learning-AR/Server/domain"
)

type eventStream struct {
	lines  *bufio.Scanner
	cancel context.CancelFunc
}

func (f *fixture) joinRoom(t *testing.T, roomID string) *eventStream {
	t.Helper()
	server := httptest.NewServer(f.e)
	t.Cleanup(server.Close)
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, server.URL+"/room/"+roomID+"/join", nil)
	if err != nil {
		t.Fatal(err)
	}
	res, err := server.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = res.Body.Close() })
	if res.StatusCode != http.StatusOK || res.Header.Get("Content-Type") != "text/event-stream" {
		t.Fatalf("response = %d %q, want 200 text/event-stream", res.StatusCode, res.Header.Get("Content-Type"))
	}
	return &eventStream{lines: bufio.NewScanner(res.Body), cancel: cancel}
}

// next returns the next non-heartbeat SSE message as its lines.
func (s *eventStream) next(t *testing.T) []string {
	t.Helper()
	var message []string
	for s.lines.Scan() {
		line := s.lines.Text()
		switch {
		case line == "" && len(message) > 0:
			return message
		case line == "", strings.HasPrefix(line, ":"):
		default:
			message = append(message, line)
		}
	}
	t.Fatalf("stream ended before a message: %v", s.lines.Err())
	return nil
}

func (s *eventStream) waitHeartbeat(t *testing.T) {
	t.Helper()
	for s.lines.Scan() {
		if s.lines.Text() == ": heartbeat" {
			return
		}
	}
	t.Fatalf("stream ended before a heartbeat: %v", s.lines.Err())
}

func waitFor(t *testing.T, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for %s", what)
		}
		time.Sleep(5 * time.Millisecond)
	}
}

func TestJoinRoomEventsStreamsPublishedEvents(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	stream := f.joinRoom(t, "room-1")
	publish := func(event domain.RoomEvent) {
		t.Helper()
		if err := f.roomEvents.Publish(context.Background(), "room-1", event); err != nil {
			t.Fatal(err)
		}
	}

	stream.waitHeartbeat(t)
	publish(domain.RoomEvent{Name: "unencodable", Data: make(chan int)})
	publish(domain.RoomEvent{Name: "match-started", Data: map[string]string{"matchId": "match-1"}})

	got := stream.next(t)
	want := []string{"event: match-started", `data: {"matchId":"match-1"}`}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Errorf("message = %q, want %q (unencodable events are skipped)", got, want)
	}
}

func TestJoinRoomEventsUnsubscribesOnDisconnect(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	stream := f.joinRoom(t, "room-1")
	if got := f.broker.SubscriberCount("room-1"); got != 1 {
		t.Fatalf("SubscriberCount() = %d, want 1 while connected", got)
	}

	stream.cancel()

	waitFor(t, "unsubscription", func() bool { return f.broker.SubscriberCount("room-1") == 0 })
}
