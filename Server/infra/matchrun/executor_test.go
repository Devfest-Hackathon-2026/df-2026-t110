package matchrun

import (
	"context"
	"encoding/json/jsontext"
	json "encoding/json/v2"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/dvpico/Programming-Learning-AR/Server/domain"
	"github.com/dvpico/Programming-Learning-AR/Server/infra/router/gen/api"
	wire "github.com/dvpico/Programming-Learning-AR/Server/infra/router/gen/history"
	"github.com/dvpico/Programming-Learning-AR/Server/infra/router/gen/schema"
)

func source(actions ...string) jsontext.Value {
	stmts := make([]string, 0, len(actions)+1)
	for i, action := range actions {
		stmts = append(stmts, fmt.Sprintf(`{"kind":"call","hintId":"00000000-0000-4000-8000-%012x","func":%q,"args":[]}`, i+10, action))
	}
	stmts = append(stmts, `{"kind":"return","hintId":"00000000-0000-4000-8000-000000000002","value":{"kind":"literal","hintId":"00000000-0000-4000-8000-000000000003","type":"unit","value":null}}`)
	return jsontext.Value(`{"picolangVersion":0,"picolang":{"v0":{"program":{"main":{"kind":"function","hintId":"00000000-0000-4000-8000-000000000000","params":[],"returnType":"unit","body":{"kind":"block","hintId":"00000000-0000-4000-8000-000000000001","stmts":[` + strings.Join(stmts, ",") + `]}}}}}}`)
}

func team(program jsontext.Value) []domain.Program {
	programs := make([]domain.Program, domain.BattleRobotCount)
	for i := range programs {
		programs[i] = domain.Program{ID: "repeated-program", Source: program, IsBattle: true}
	}
	return programs
}

func TestExecuteScoresAndRecordsIndependentRobots(t *testing.T) {
	t.Parallel()
	layouts, err := testLayouts(t).List(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	for _, layout := range layouts {
		t.Run(layout.ID, func(t *testing.T) {
			t.Parallel()
			for _, y := range []int{1, 3, 5} {
				layout.Territories[0] = append(layout.Territories[0], schema.Position{X: 1, Y: y})
				layout.Luggage = append(layout.Luggage, schema.Position{X: 2, Y: y})
			}
			result, err := NewExecutor().Execute(t.Context(), team(source("forward", "pick_up", "back", "release")), team(source()), layout)
			if err != nil {
				t.Errorf("Execute: %v", err)
				return
			}
			var decoded struct {
				History []wire.FieldEvent `json:"history"`
				Summary api.MatchSummary  `json:"summary"`
			}
			if err := json.Unmarshal(result, &decoded); err != nil {
				t.Errorf("decode result: %v", err)
				return
			}
			if decoded.Summary.Score.Player != 3 || decoded.Summary.Score.Opponent != 0 {
				t.Errorf("score=%+v", decoded.Summary.Score)
			}
			if len(decoded.History) == 0 || decoded.History[0].State.Initial == nil {
				t.Errorf("missing initial board: %s", result)
				return
			}
			initial := decoded.History[0].State.Initial
			ids := make(map[wire.RobotId]struct{})
			var teams [3]int
			for _, robot := range initial.Robots {
				ids[robot.Id] = struct{}{}
				if robot.TeamId != nil {
					switch *robot.TeamId {
					case 1:
						teams[1]++
					case 2:
						teams[2]++
					default:
						t.Errorf("invalid robot team: %v", *robot.TeamId)
					}
				}
			}
			if len(ids) != 6 || teams[1] != 3 || teams[2] != 3 {
				t.Errorf("initial robots=%+v", initial.Robots)
			}
			for _, event := range decoded.History {
				if event.Type != "field" {
					t.Errorf("execution logs were recorded: %s", event.Type)
				}
			}
		})
	}
}

func TestExecuteLimitsAndInvalidPrograms(t *testing.T) {
	t.Parallel()
	layouts, err := testLayouts(t).List(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	loop, err := os.ReadFile("../tutorialrun/testdata/loop.json")
	if err != nil {
		t.Fatal(err)
	}
	actionLoop := strings.Replace(string(loop), `"stmts": []`, `"stmts": [{"kind":"call","hintId":"00000000-0000-4000-8000-000000000092","func":"forward","args":[]}]`, 1)
	queryMove, err := os.ReadFile("testdata/query_move.json")
	if err != nil {
		t.Fatal(err)
	}
	queryOnly := jsontext.Value(strings.Replace(string(queryMove), `"func": "forward"`, `"func": "search_now"`, 1))
	for _, tt := range []struct {
		name             string
		player, opponent jsontext.Value
		wantErr          error
		wantTick         uint64
	}{
		{name: "all stopped", player: source(), opponent: source()},
		{name: "invalid challenger", player: jsontext.Value(`{}`), opponent: source(), wantErr: domain.ErrInvalidProgram},
		{name: "invalid defense stops", player: source(), opponent: jsontext.Value(`{}`)},
		{name: "pure computation limit", player: loop, opponent: loop, wantTick: 1},
		{name: "turn limit", player: jsontext.Value(actionLoop), opponent: jsontext.Value(actionLoop), wantTick: layouts[0].Turns},
		{name: "queries before every move", player: queryMove, opponent: queryMove, wantTick: layouts[0].Turns},
		{name: "query heavy defense", player: jsontext.Value(actionLoop), opponent: queryMove, wantTick: layouts[0].Turns},
		{name: "queries without advancing turns", player: queryOnly, opponent: queryOnly},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result, err := NewExecutor().Execute(t.Context(), team(tt.player), team(tt.opponent), layouts[0])
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("Execute error=%v, want %v", err, tt.wantErr)
			}
			if err != nil || tt.wantErr != nil {
				return
			}
			var decoded struct {
				History []wire.FieldEvent `json:"history"`
			}
			if err := json.Unmarshal(result, &decoded); err != nil {
				t.Errorf("decode: %v", err)
				return
			}
			if len(decoded.History) == 0 {
				t.Error("missing execution history")
			} else {
				got := decoded.History[len(decoded.History)-1].Tick
				if got < 0 {
					t.Errorf("negative final tick=%d", got)
				} else if uint64(got) != tt.wantTick {
					t.Errorf("final tick=%d, want %d", got, tt.wantTick)
				}
			}
		})
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err := NewExecutor().Execute(ctx, team(source()), team(source()), layouts[0]); !errors.Is(err, context.Canceled) {
		t.Errorf("canceled execution: %v", err)
	}
}
