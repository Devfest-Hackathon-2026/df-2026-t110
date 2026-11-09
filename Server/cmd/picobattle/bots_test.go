package main

import (
	"bytes"
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
					if profile == "courier" && outcome.Team1().Score() != 3 {
						t.Errorf("courier %s seed %d score=%d, want 3", layout.FieldID, seed, outcome.Team1().Score())
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
