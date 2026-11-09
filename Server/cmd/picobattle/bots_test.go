package main

import (
	"bytes"
	json "encoding/json/v2"
	"fmt"
	"testing"

	"github.com/dvpico/Programming-Learning-AR/Server/domain"
	"github.com/dvpico/Programming-Learning-AR/Server/game/hub"
	"github.com/dvpico/Programming-Learning-AR/Server/game/protocol"
	boardmsg "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/board"
)

func team(source []byte) []domain.Program {
	programs := make([]domain.Program, domain.BattleRobotCount)
	for i := range programs {
		programs[i] = domain.Program{Source: source, IsBattle: true}
	}
	return programs
}

func TestCourierDeliversOnOpenLanes(t *testing.T) {
	t.Parallel()
	var layout domain.BattleLayout
	if err := json.Unmarshal(readTestFile(t, "../../testdata/battle/cargo-lanes.json"), &layout, json.RejectUnknownMembers(true)); err != nil {
		t.Fatal(err)
	}
	sentry, err := generateBot("sentry", 0)
	if err != nil {
		t.Fatal(err)
	}
	for seed := uint64(0); seed < 8; seed++ {
		t.Run(fmt.Sprint(seed), func(t *testing.T) {
			t.Parallel()
			courier, err := generateBot("courier", seed)
			if err != nil {
				t.Fatal(err)
			}
			report, err := simulate(t.Context(), team(courier.Source), team(sentry.Source), layout, battleOptions{Limits: defaultLimits()})
			if err != nil {
				t.Fatal(err)
			}
			outcome := report.Result().(boardmsg.BattleResult)
			if outcome.Team1().Score() != 3 || outcome.Team2().Score() != 0 {
				t.Errorf("courier score=%d-%d, want 3-0", outcome.Team1().Score(), outcome.Team2().Score())
			}
		})
	}
}

func TestGeneratedProgramsRunProductionBattles(t *testing.T) {
	t.Parallel()
	layouts, err := loadLayouts(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	sentry, err := generateBot("sentry", 0)
	if err != nil {
		t.Fatal(err)
	}
	for _, profile := range botProfiles() {
		t.Run(profile, func(t *testing.T) {
			t.Parallel()
			for seed := uint64(0); seed < 8; seed++ {
				program, err := generateBot(profile, seed)
				if err != nil {
					t.Fatal(err)
				}
				again, err := generateBot(profile, seed)
				if err != nil {
					t.Fatal(err)
				}
				if !bytes.Equal(program.Source, again.Source) {
					t.Errorf("seed %d is not reproducible", seed)
				}
				for _, layout := range layouts {
					opts := battleOptions{Limits: defaultLimits()}
					for team := range opts.Seeds {
						for slot := range opts.Seeds[team] {
							opts.Seeds[team][slot] = seed + uint64(slot)
						}
					}
					report, err := simulate(t.Context(), team(program.Source), team(sentry.Source), layout, opts)
					if err != nil {
						t.Errorf("%s seed %d: %v", layout.FieldID, seed, err)
						continue
					}
					for _, id := range []protocol.RobotID{"team-1-robot-1", "team-1-robot-2", "team-1-robot-3", "team-2-robot-1", "team-2-robot-2", "team-2-robot-3"} {
						if failed, ok := report.Outcome(id).(hub.FailedOutcome); ok {
							t.Errorf("%s seed %d %s failed: %v", layout.FieldID, seed, id, failed.Err)
						}
					}
					outcome := report.Result().(boardmsg.BattleResult)
					if outcome.Team1().Discards() != 0 || outcome.Team2().Discards() != 0 {
						t.Errorf("%s seed %d discarded luggage", layout.FieldID, seed)
					}
					if outcome.Team1().Luggage()+outcome.Team2().Luggage() > uint64(len(layout.Luggage)) {
						t.Errorf("%s seed %d created extra luggage", layout.FieldID, seed)
					}
					if outcome.Team2().Score() != 0 {
						t.Errorf("%s seed %d: stationary opponent scored %d", layout.FieldID, seed, outcome.Team2().Score())
					}
				}
			}
		})
	}
}
func TestUnknownProfile(t *testing.T) {
	t.Parallel()
	if _, err := generateBot("unknown", 1); err == nil {
		t.Error("accepted unknown profile")
	}
}
