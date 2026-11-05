package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"sort"

	"github.com/dvpico/Programming-Learning-AR/Server/game"
	"github.com/dvpico/Programming-Learning-AR/Server/game/board"
	"github.com/dvpico/Programming-Learning-AR/Server/game/hub"
	"github.com/dvpico/Programming-Learning-AR/Server/game/robot"
)

func main() {
	maxTicks := flag.Uint64("max-ticks", 1000, "maximum logical tick")
	maxResumes := flag.Uint64("max-resumes", 10000, "maximum robot resumptions")
	flag.Parse()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	err := run(ctx, os.Stdin, os.Stdout, *maxTicks, *maxResumes)
	stop()
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(ctx context.Context, input io.ReadCloser, output io.Writer, maxTicks, maxResumes uint64) error {
	b, err := board.New(board.Config{
		Terrain: [][]board.Terrain{
			{board.Land, board.Land, board.Land, board.Land, board.Land},
			{board.Land, board.Land, board.Rock, board.Land, board.Land},
			{board.Land, board.Land, board.Sea, board.Land, board.Land},
		},
		Robots: map[game.RobotID]board.RobotState{
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
	for _, id := range []game.RobotID{"p1", "p2"} {
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
	if err := render(output, b.Snapshot()); err != nil {
		return err
	}
	report, runErr := h.Run(ctx, hub.Options{
		MaxTicks: maxTicks, MaxResumes: maxResumes,
		OnTick: func(_ game.TickUpdate) error { return render(output, b.Snapshot()) },
	})
	if _, err := fmt.Fprintf(output, "\nfinished at tick %d (%d resumptions)\n", report.Tick, report.Resumes); err != nil && runErr == nil {
		return err
	}
	for _, p := range participants {
		outcome := report.Outcomes[p.ID]
		if _, err := fmt.Fprintf(output, "%s: %s", p.ID, outcome.Status); err != nil {
			return err
		}
		if outcome.Err != nil {
			if _, err := fmt.Fprintf(output, " (%v)", outcome.Err); err != nil {
				return err
			}
		}
		if _, err := fmt.Fprintln(output); err != nil {
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
			cell := map[board.Terrain]string{board.Land: ".", board.Rock: "#", board.Sea: "~"}[terrain]
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
	ids := make([]game.RobotID, 0, len(snapshot.Robots))
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
