package router

import (
	"context"
	json "encoding/json/v2"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/dvpico/Programming-Learning-AR/Server/domain"
	"github.com/dvpico/Programming-Learning-AR/Server/infra/router/gen/api"
	"github.com/labstack/echo/v5"
)

type rankingStub struct {
	entries []domain.RankingEntry
	err     error
}

func (s rankingStub) List(context.Context) ([]domain.RankingEntry, error) { return s.entries, s.err }

func TestGetRanking(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name       string
		stub       rankingStub
		wantStatus int
		want       []api.RankingEntry
	}{
		{
			name: "entries", wantStatus: http.StatusOK,
			stub: rankingStub{entries: []domain.RankingEntry{
				{Rank: 1, Name: "alice", Rating: 1030, Wins: 1},
				{Rank: 2, Name: "bob", Rating: 975, Losses: 1, Draws: 2},
			}},
			want: []api.RankingEntry{
				{Rank: 1, Name: "alice", Rating: 1030, Wins: 1},
				{Rank: 2, Name: "bob", Rating: 975, Losses: 1, Draws: 2},
			},
		},
		{name: "empty is an array", wantStatus: http.StatusOK, want: []api.RankingEntry{}},
		{name: "storage failure", stub: rankingStub{err: errors.New("down")}, wantStatus: http.StatusInternalServerError},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			api.RegisterHandlersWithBaseURL(e, NewRouter(Config{Rankings: tt.stub}), "/api")

			res := httptest.NewRecorder()
			e.ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/api/ranking", nil))
			if res.Code != tt.wantStatus {
				t.Errorf("status=%d, want %d", res.Code, tt.wantStatus)
			}
			if tt.want == nil {
				return
			}

			var got []api.RankingEntry
			if err := json.Unmarshal(res.Body.Bytes(), &got); err != nil {
				t.Fatalf("decode %s: %v", res.Body.String(), err)
			}
			// A JSON null would decode to a nil slice; clients expect an array.
			if got == nil || !reflect.DeepEqual(got, tt.want) {
				t.Errorf("body=%s, want %+v", res.Body.String(), tt.want)
			}
		})
	}
}
