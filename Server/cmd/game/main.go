package main

import (
	json "encoding/json/v2"
	"flag"
	"fmt"
	"io"
	"os"
	"sort"

	"github.com/dvpico/Programming-Learning-AR/Server/game/board"
	"github.com/dvpico/Programming-Learning-AR/Server/game/hub"
	"github.com/dvpico/Programming-Learning-AR/Server/game/protocol"
	boardmsg "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/board"
	fieldevent "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/field_event"
	gameevent "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/game_event"

	"github.com/dvpico/Programming-Learning-AR/Server/game/recorder"
	"github.com/dvpico/Programming-Learning-AR/Server/game/robot"
)

func main() {
	maxTicks := flag.Uint64("max-ticks", 1000, "maximum logical tick")
	maxResumes := flag.Uint64("max-resumes", 10000, "maximum robot resumptions")
	battleTurns := flag.Uint64("battle-turns", 0, "turns before normal battle completion (zero disables battle scoring)")
	luggagePoints := flag.Int64("luggage-points", 1, "points per piece of luggage in a team's territory")
	discardPenalty := flag.Int64("discard-penalty", 1, "points deducted per piece of luggage discarded at sea")
	flag.Parse()
	completion, err := completionRules(*battleTurns, *luggagePoints, *discardPenalty)
	if err == nil {
		err = run(os.Stdin, os.Stdout, *maxTicks, *maxResumes, completion)
	}
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func completionRules(turns uint64, points, penalty int64) (board.CompletionRules, error) {
	if turns == 0 {
		return board.AllStoppedRules{}, nil
	}
	return board.NewBattleConfig(turns, points, penalty).CompletionRules()
}

func run(input io.ReadCloser, output io.Writer, maxTicks, maxResumes uint64, completion board.CompletionRules) error {
	if maxTicks == 0 {
		return fmt.Errorf("max-ticks must be positive")
	}
	plain := board.NewLandTerrain(board.PlainLand{}, false, 0)
	luggage := board.NewLandTerrain(board.PlainLand{}, true, board.NoTeam)
	team1Territory := board.NewLandTerrain(board.PlainLand{}, false, board.TeamID1)
	team2Territory := board.NewLandTerrain(board.PlainLand{}, false, board.TeamID2)
	config, err := board.NewConfig(
		[][]board.Terrain{
			{team1Territory, plain, plain, plain, team2Territory},
			{plain, luggage, board.RockTerrain{}, plain, plain},
			{plain, plain, board.SeaTerrain{}, plain, plain},
		},
		map[protocol.RobotID]board.RobotState{
			"p1": board.NewRobotState(board.NewPosition(0, 0), board.East, 3).WithTeam(board.TeamID1),
			"p2": board.NewRobotState(board.NewPosition(4, 0), board.West, 3).WithTeam(board.TeamID2),
		},
		maxTicks,
		completion,
	)
	if err != nil {
		return err
	}
	b, err := board.New(config)
	if err != nil {
		return err
	}
	session, err := robot.NewSession(input, output)
	if err != nil {
		return err
	}
	defer func() { _ = session.Close() }()
	participants := make([]hub.Participant, 0, 2)
	for _, id := range []protocol.RobotID{"p1", "p2"} {
		r, err := robot.NewStdio(id, session)
		if err != nil {
			return err
		}
		participants = append(participants, hub.Participant{ID: id, Robot: r})
	}
	h, err := hub.New(b, participants)
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintln(output, "Commands:", robot.CommandUsage); err != nil {
		return err
	}
	report, runErr := h.Run(hub.Options{
		MaxResumes: maxResumes,
		Recorder:   &boardRenderer{output: output, board: b},
	})
	if _, err := fmt.Fprintf(output, "\nfinished at tick %d (%d resumptions)\n", b.Snapshot().Tick, report.Resumes()); err != nil && runErr == nil {
		return err
	}
	for _, p := range participants {
		var status string
		switch outcome := report.Outcome(p.ID).(type) {
		case hub.UnstartedOutcome, hub.StoppedOutcome:
			status = "stopped"
		case hub.CompletedOutcome:
			status = "completed"
		case hub.InterruptedOutcome:
			status = "interrupted by game completion"
		case hub.FailedOutcome:
			status = fmt.Sprintf("failed (%v)", outcome.Err)
		default:
			return fmt.Errorf("invalid outcome for robot %q", p.ID)
		}
		if _, err := fmt.Fprintf(output, "%s: %s\n", p.ID, status); err != nil {
			return err
		}
	}
	if runErr != nil {
		return runErr
	}
	return renderResult(output, report.Result())
}

func renderResult(output io.Writer, result boardmsg.CompletionResult) error {
	switch result := result.(type) {
	case nil:
		return nil
	case boardmsg.BattleResult:
		winner := "draw"
		if result.Winner() != 0 {
			winner = fmt.Sprintf("team %d wins", result.Winner())
		}
		_, err := fmt.Fprintf(output, "battle ended (%s): team 1 = %d, team 2 = %d, %s\n", result.EndReason(), result.Team1().Score(), result.Team2().Score(), winner)
		return err
	default:
		return fmt.Errorf("unsupported game result %T", result)
	}
}

func render(output io.Writer, snapshot board.Snapshot) error {
	if _, err := fmt.Fprintf(output, "\ntick %d\n", snapshot.Tick); err != nil {
		return err
	}
	for y, row := range snapshot.Terrain {
		for x, terrain := range row {
			var cell string
			switch terrain := terrain.(type) {
			case board.LandTerrain:
				switch kind := terrain.Kind().(type) {
				case board.PlainLand:
					cell = "."
				case board.GoalLand:
					cell = "G"
				case board.CheckpointLand:
					cell = "C"
				default:
					return fmt.Errorf("invalid land kind %T", kind)
				}
				if terrain.HasLuggage() {
					cell = "L"
				}
			case board.RockTerrain:
				cell = "#"
			case board.SeaTerrain:
				cell = "~"
			default:
				return fmt.Errorf("invalid terrain %T", terrain)
			}
			for id, robot := range snapshot.Robots {
				if robot.Position() == board.NewPosition(x, y) {
					cell = string(id)
				}
			}
			if _, err := fmt.Fprintf(output, "%3s", cell); err != nil {
				return err
			}
		}
		if _, err := fmt.Fprintln(output); err != nil {
			return err
		}
	}
	ids := make([]protocol.RobotID, 0, len(snapshot.Robots))
	for id := range snapshot.Robots {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	for _, id := range ids {
		r := snapshot.Robots[id]
		position := r.Position()
		cargo := ""
		if r.CarryingLuggage() {
			cargo = " carrying luggage"
		}
		if _, err := fmt.Fprintf(output, "%s: (%d,%d) %s%s\n", id, position.X(), position.Y(), r.Direction(), cargo); err != nil {
			return err
		}
	}
	return nil
}

// boardRenderer records events and displays field updates for the CLI.
type boardRenderer struct {
	recorder.Recorder
	output io.Writer
	board  *board.Board
}

func (r *boardRenderer) Record(event gameevent.GameEvent) error {
	if err := r.Recorder.Record(event); err != nil {
		return err
	}
	if field, ok := event.(gameevent.FieldEvent); ok {
		if update, ok := field.Event.State.(fieldevent.UpdateState); ok {
			for _, action := range update.Actions {
				var speech string
				switch action := action.(type) {
				case fieldevent.PrintIntegerResult:
					speech = fmt.Sprintf("%s says %d\n", action.RobotID, action.Integer)
				case fieldevent.PrintBoolResult:
					speech = fmt.Sprintf("%s says %t\n", action.RobotID, action.Bool)
				case fieldevent.PrintUnitResult:
					speech = fmt.Sprintf("%s says ()\n", action.RobotID)
				case fieldevent.PrintTerrainResult:
					terrain, err := json.Marshal(action.Terrain)
					if err != nil {
						return err
					}
					speech = fmt.Sprintf("%s says %s\n", action.RobotID, terrain)
				case fieldevent.PrintRotationResult:
					speech = fmt.Sprintf("%s says %s\n", action.RobotID, action.Rotation)
				case fieldevent.PrintDistanceTargetResult:
					speech = fmt.Sprintf("%s says %s\n", action.RobotID, action.DistanceTarget)
				}
				if speech != "" {
					if _, err := io.WriteString(r.output, speech); err != nil {
						return err
					}
				}
			}
		}
		snapshot := r.board.Snapshot()
		snapshot.Tick = field.Event.Tick
		return render(r.output, snapshot)
	}
	return nil
}
