package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"sort"

	"github.com/dvpico/Programming-Learning-AR/Server/game/board"
	"github.com/dvpico/Programming-Learning-AR/Server/game/hub"
	"github.com/dvpico/Programming-Learning-AR/Server/game/protocol"
	gameevent "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/game_event"
	values "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/value"
	"github.com/dvpico/Programming-Learning-AR/Server/game/recorder"
	"github.com/dvpico/Programming-Learning-AR/Server/game/robot"
)

func main() {
	maxTicks := flag.Uint64("max-ticks", 1000, "maximum logical tick")
	maxResumes := flag.Uint64("max-resumes", 10000, "maximum robot resumptions")
	flag.Parse()
	err := run(os.Stdin, os.Stdout, *maxTicks, *maxResumes)
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(input io.ReadCloser, output io.Writer, maxTicks, maxResumes uint64) error {
	if maxTicks == 0 {
		return fmt.Errorf("max-ticks must be positive")
	}
	plain := values.LandTerrain{Kind: values.PlainLand{}}
	b, err := board.New(board.Config{
		MaxTicks: maxTicks,
		Terrain: [][]values.TerrainValue{
			{plain, plain, plain, plain, plain},
			{plain, plain, values.RockTerrain{}, plain, plain},
			{plain, plain, values.SeaTerrain{}, plain, plain},
		},
		Robots: map[protocol.RobotID]board.RobotState{
			"p1": {Position: board.Position{X: 0, Y: 0}, Direction: board.East, DrillDurability: 3},
			"p2": {Position: board.Position{X: 4, Y: 0}, Direction: board.West, DrillDurability: 3},
		},
	})
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
	if _, err := fmt.Fprintln(output, "Commands: <id> forward | back | rotate left/right | drill_durability | done; help"); err != nil {
		return err
	}
	report, runErr := h.Run(hub.Options{
		MaxResumes: maxResumes,
		Recorder:   &boardRenderer{output: output, board: b},
	})
	if _, err := fmt.Fprintf(output, "\nfinished at tick %d (%d resumptions)\n", b.Snapshot().Tick, report.Resumes); err != nil && runErr == nil {
		return err
	}
	for _, p := range participants {
		var status string
		switch outcome := report.Outcomes[p.ID].(type) {
		case hub.UnstartedOutcome, hub.StoppedOutcome:
			status = "stopped"
		case hub.CompletedOutcome:
			status = "completed"
		case hub.FailedOutcome:
			status = fmt.Sprintf("failed (%v)", outcome.Err)
		default:
			return fmt.Errorf("invalid outcome for robot %q", p.ID)
		}
		if _, err := fmt.Fprintf(output, "%s: %s\n", p.ID, status); err != nil {
			return err
		}
	}
	return runErr
}

func render(output io.Writer, snapshot board.Snapshot) error {
	if _, err := fmt.Fprintf(output, "\ntick %d\n", snapshot.Tick); err != nil {
		return err
	}
	for y, row := range snapshot.Terrain {
		for x, terrain := range row {
			var cell string
			switch terrain := terrain.(type) {
			case values.LandTerrain:
				switch terrain.Kind.(type) {
				case values.PlainLand:
					cell = "."
				case values.GoalLand:
					cell = "G"
				case values.CheckpointLand:
					cell = "C"
				case values.LuggageLand:
					cell = "L"
				default:
					return fmt.Errorf("invalid land kind %T", terrain.Kind)
				}
			case values.RockTerrain:
				cell = "#"
			case values.SeaTerrain:
				cell = "~"
			default:
				return fmt.Errorf("invalid terrain %T", terrain)
			}
			for id, robot := range snapshot.Robots {
				if robot.Position == (board.Position{X: x, Y: y}) {
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
		if _, err := fmt.Fprintf(output, "%s: (%d,%d) %s\n", id, r.Position.X, r.Position.Y, r.Direction); err != nil {
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
	if _, field := event.(gameevent.FieldEvent); field {
		return render(r.output, r.board.Snapshot())
	}
	return nil
}
