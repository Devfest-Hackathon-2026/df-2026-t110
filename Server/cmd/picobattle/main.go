// picobattle generates practice opponents and runs offline, reproducible battles.
package main

import (
	"context"
	"encoding/json/jsontext"
	json "encoding/json/v2"
	"errors"
	"flag"
	"fmt"
	"io"
	"math/rand/v2"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/dvpico/Programming-Learning-AR/Server/domain"
	"github.com/dvpico/Programming-Learning-AR/Server/game/hub"
	"github.com/dvpico/Programming-Learning-AR/Server/game/protocol"
	boardmsg "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/board"
	"github.com/dvpico/Programming-Learning-AR/Server/infra/matchrun"
)

const help = `picobattle: サクラ生成とオフライン対戦（picolang v0 JSON）

  picobattle generate --out bots --count 100 --seed 42
  picobattle generate --out couriers --profile courier --count 30
  picobattle layouts
  picobattle run --player bots/bot-000003-courier.json --out battle
  picobattle run --player a.json --player b.json --player c.json --opponent d.json --out battle

各チームは3台。--player / --opponent は1ファイルなら3台に複製、3ファイルなら指定順。
相手を省略するとシードから courier / wanderer / miner を生成。
--out は新規ディレクトリを指定（既存ファイルは上書きしない）。
生成: bot-*.json と manifest.json（性格・パラメータ・シード・ファイル名）。
対戦: history.jsonl（初期盤面＋各ターンの差分を実行中に逐次書き込み）、
      result.json（勝敗・得点・終了理由・各ロボットの状態とエラー・実行設定）。
同じソース・レイアウト・シード・制限なら再現可能。DBやサーバーは不要。
終了コード: 0=正常終了（負け・引き分けを含む）、1=入力/保存/実行エラー、
            2=対戦完了したがロボットに実行エラーあり（結果と履歴は保存）。
`

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := execute(ctx, os.Args[1:], os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, "picobattle:", err)
		if errors.Is(err, errRobotFailed) {
			os.Exit(2)
		}
		os.Exit(1)
	}
}

var errRobotFailed = errors.New("one or more robots failed; see result.json")

func execute(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		_, err := io.WriteString(stdout, help)
		return err
	}
	switch args[0] {
	case "help", "-h", "--help":
		_, err := io.WriteString(stdout, help)
		return err
	case "generate":
		return generate(ctx, args[1:], stdout, stderr)
	case "run":
		return run(ctx, args[1:], stdout, stderr)
	case "layouts":
		if len(args) != 1 {
			return errors.New("layouts takes no arguments")
		}
		layouts, err := loadLayouts(ctx)
		if err != nil {
			return err
		}
		return json.MarshalWrite(stdout, layouts, jsontext.WithIndent("  "))
	default:
		return fmt.Errorf("unknown command %q; use --help", args[0])
	}
}

func flags(name string, stderr io.Writer) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(stderr)
	return fs
}
func parseFlags(fs *flag.FlagSet, args []string) error {
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("unexpected arguments: %s", strings.Join(fs.Args(), " "))
	}
	return nil
}
func newDirectory(path string) error {
	if strings.TrimSpace(path) == "" {
		return errors.New("--out is required")
	}
	if err := os.Mkdir(path, 0o750); err != nil {
		return fmt.Errorf("create output directory: %w", err)
	}
	return nil
}
func writeJSON(path string, value any) error {
	data, err := json.Marshal(value, json.Deterministic(true), jsontext.WithIndent("  "))
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o600)
}

type manifestEntry struct {
	File string `json:"file"`
	botProgram
}

func generate(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	fs := flags("generate", stderr)
	out := fs.String("out", "", "new output directory (required)")
	count := fs.Int("count", 30, "number of sources")
	seed := fs.Uint64("seed", 1, "generation seed")
	profile := fs.String("profile", "mixed", "mixed, sentry, wanderer, courier, miner")
	if err := parseFlags(fs, args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if *count <= 0 {
		return errors.New("--count must be positive")
	}
	profiles := botProfiles()
	if *profile != "mixed" {
		if _, err := generateBot(*profile, *seed); err != nil {
			return err
		}
		profiles = []string{*profile}
	}
	if err := newDirectory(*out); err != nil {
		return err
	}
	entries := make([]manifestEntry, 0)
	rng := rand.New(rand.NewPCG(*seed, *seed^0x9e3779b97f4a7c15)) //nolint:gosec // Seeds intentionally reproduce generated opponents.
	for i := 0; i < *count; i++ {
		if err := ctx.Err(); err != nil {
			return err
		}
		p, err := generateBot(profiles[i%len(profiles)], rng.Uint64())
		if err != nil {
			return err
		}
		name := fmt.Sprintf("bot-%06d-%s.json", i+1, p.Profile)
		// Keep canonical source files readable and directly usable in the app/API.
		if err := writeJSON(filepath.Join(*out, name), jsontext.Value(p.Source)); err != nil {
			return err
		}
		p.Source = nil
		entries = append(entries, manifestEntry{File: name, botProgram: p})
	}
	if err := writeJSON(filepath.Join(*out, "manifest.json"), struct {
		Seed     uint64          `json:"seed"`
		Programs []manifestEntry `json:"programs"`
	}{*seed, entries}); err != nil {
		return err
	}
	_, err := fmt.Fprintf(stdout, "%d programs generated: %s\n", *count, *out)
	return err
}

type paths []string

func (p *paths) String() string         { return strings.Join(*p, ", ") }
func (p *paths) Set(value string) error { *p = append(*p, value); return nil }

func readTeam(files []string) ([]domain.Program, []string, error) {
	if len(files) == 1 {
		files = []string{files[0], files[0], files[0]}
	}
	if len(files) != domain.BattleRobotCount {
		return nil, nil, errors.New("each team requires 1 or 3 source files")
	}
	programs := make([]domain.Program, len(files))
	for i, name := range files {
		data, err := os.ReadFile(name) //nolint:gosec // The local CLI explicitly accepts source paths from its caller.
		if err != nil {
			return nil, nil, fmt.Errorf("read source %s: %w", name, err)
		}
		programs[i] = domain.Program{Source: data, IsBattle: true}
	}
	return programs, files, nil
}
func loadLayouts(ctx context.Context) ([]domain.BattleLayout, error) {
	repo, err := matchrun.NewLayouts()
	if err != nil {
		return nil, err
	}
	return repo.List(ctx)
}
func chooseLayout(ctx context.Context, id, file string) (domain.BattleLayout, error) {
	if file != "" {
		data, err := os.ReadFile(file) //nolint:gosec // The local CLI explicitly accepts a layout path from its caller.
		if err != nil {
			return domain.BattleLayout{}, err
		}
		var layout domain.BattleLayout
		if err := json.Unmarshal(data, &layout, json.RejectUnknownMembers(true)); err != nil {
			return layout, err
		}
		if id != "" && layout.ID != id {
			return layout, fmt.Errorf("layout file ID %q does not match --layout %q", layout.ID, id)
		}
		return layout, nil
	}
	layouts, err := loadLayouts(ctx)
	if err != nil {
		return domain.BattleLayout{}, err
	}
	for _, layout := range layouts {
		if id == "" || layout.ID == id {
			return layout, nil
		}
	}
	return domain.BattleLayout{}, fmt.Errorf("unknown layout %q; use layouts", id)
}

type robotResult struct {
	ID     string `json:"id"`
	Source string `json:"source"`
	Seed   uint64 `json:"seed"`
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
}
type teamResult struct {
	Score    int64  `json:"score"`
	Luggage  uint64 `json:"luggage"`
	Discards uint64 `json:"discards"`
}
type result struct {
	Seed      uint64              `json:"seed"`
	Layout    domain.BattleLayout `json:"layout"`
	Limits    matchLimits         `json:"limits"`
	Ticks     uint64              `json:"ticks"`
	Resumes   uint64              `json:"resumes"`
	EndReason boardmsg.EndReason  `json:"endReason"`
	Winner    string              `json:"winner"`
	Teams     [2]teamResult       `json:"teams"`
	Robots    []robotResult       `json:"robots"`
	Error     string              `json:"error,omitempty"`
}
type matchLimits struct {
	Steps  uint64 `json:"steps"`
	Memory uint64 `json:"memoryBytes"`
	Depth  uint64 `json:"callDepth"`
}

func run(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	fs := flags("run", stderr)
	var playerFiles, opponentFiles paths
	fs.Var(&playerFiles, "player", "player source JSON; repeat 3 times for different robots (required)")
	fs.Var(&opponentFiles, "opponent", "opponent source JSON; omit for generated opponents")
	out := fs.String("out", "", "new output directory (required)")
	seed := fs.Uint64("seed", 1, "battle seed")
	layoutID := fs.String("layout", "", "built-in layout ID (default: first)")
	layoutFile := fs.String("layout-file", "", "custom BattleLayout JSON (one object; see layouts)")
	turns := fs.Uint64("turns", 0, "override turn count (0: layout default)")
	limits := defaultLimits()
	fs.Uint64Var(&limits.MaxSteps, "max-steps", limits.MaxSteps, "VM step limit per robot")
	fs.Uint64Var(&limits.MaxMemoryBytes, "max-memory", limits.MaxMemoryBytes, "VM memory limit per robot in bytes")
	fs.Uint64Var(&limits.MaxCallDepth, "max-depth", limits.MaxCallDepth, "VM call depth limit per robot")
	if err := parseFlags(fs, args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if limits.MaxSteps == 0 || limits.MaxMemoryBytes == 0 || limits.MaxCallDepth == 0 {
		return errors.New("VM limits must be positive")
	}
	player, playerNames, err := readTeam(playerFiles)
	if err != nil {
		return fmt.Errorf("player: %w", err)
	}
	layout, err := chooseLayout(ctx, *layoutID, *layoutFile)
	if err != nil {
		return err
	}
	if *turns != 0 {
		layout.Turns = *turns
	}
	var opponent []domain.Program
	var opponentNames []string
	rng := rand.New(rand.NewPCG(*seed, *seed^0x9e3779b97f4a7c15)) //nolint:gosec // Simulation seeds must be reproducible and are not security tokens.
	if len(opponentFiles) == 0 {
		for _, profile := range []string{"courier", "wanderer", "miner"} {
			p, err := generateBot(profile, rng.Uint64())
			if err != nil {
				return err
			}
			opponent = append(opponent, domain.Program{Source: p.Source, IsBattle: true})
			opponentNames = append(opponentNames, "generated:"+profile)
		}
	} else {
		opponent, opponentNames, err = readTeam(opponentFiles)
		if err != nil {
			return fmt.Errorf("opponent: %w", err)
		}
	}
	options := battleOptions{Limits: limits}
	for team := range options.Seeds {
		for slot := range options.Seeds[team] {
			options.Seeds[team][slot] = rng.Uint64()
		}
	}
	if err := newDirectory(*out); err != nil {
		return err
	}
	// Preserve exact inputs as well as settings; original files may later change.
	for team, programs := range [][]domain.Program{player, opponent} {
		for slot, program := range programs {
			if err := os.WriteFile(filepath.Join(*out, fmt.Sprintf("team-%d-robot-%d.json", team+1, slot+1)), program.Source, 0o600); err != nil {
				return err
			}
		}
	}
	file, err := os.OpenFile(filepath.Join(*out, "history.jsonl"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	options.Recorder = lineRecorder{ctx: ctx, writer: file}
	report, runErr := simulate(ctx, player, opponent, layout, options)
	closeErr := file.Close()
	r := result{Seed: *seed, Layout: layout, Limits: matchLimits{limits.MaxSteps, limits.MaxMemoryBytes, limits.MaxCallDepth}, Resumes: report.Resumes(), Winner: "undetermined"}
	if outcome, ok := report.Result().(boardmsg.BattleResult); ok {
		r.Ticks, r.EndReason = outcome.Tick(), outcome.EndReason()
		r.Winner = "draw"
		if outcome.Winner() != 0 {
			r.Winner = fmt.Sprintf("team-%d", outcome.Winner())
		}
		for i, score := range []boardmsg.TeamResult{outcome.Team1(), outcome.Team2()} {
			r.Teams[i] = teamResult{score.Score(), score.Luggage(), score.Discards()}
		}
	}
	failed := false
	for team, names := range [][]string{playerNames, opponentNames} {
		for slot, name := range names {
			id := fmt.Sprintf("team-%d-robot-%d", team+1, slot+1)
			robot := robotResult{ID: id, Source: name, Seed: options.Seeds[team][slot], Status: "unstarted"}
			switch outcome := report.Outcome(protocol.RobotID(id)).(type) {
			case hub.CompletedOutcome:
				robot.Status = "completed"
			case hub.InterruptedOutcome:
				robot.Status = "interrupted"
			case hub.StoppedOutcome:
				robot.Status = "stopped"
			case hub.FailedOutcome:
				robot.Status, robot.Error, failed = "failed", outcome.Err.Error(), true
			}
			r.Robots = append(r.Robots, robot)
		}
	}
	if err := errors.Join(runErr, closeErr); err != nil {
		r.Error = err.Error()
	}
	if err := writeJSON(filepath.Join(*out, "result.json"), r); err != nil {
		return errors.Join(runErr, closeErr, err)
	}
	if _, err := fmt.Fprintf(stdout, "winner=%s score=%d:%d ticks=%d reason=%s\nhistory=%s\nresult=%s\n", r.Winner, r.Teams[0].Score, r.Teams[1].Score, r.Ticks, r.EndReason, filepath.Join(*out, "history.jsonl"), filepath.Join(*out, "result.json")); err != nil {
		return errors.Join(runErr, closeErr, err)
	}
	if err := errors.Join(runErr, closeErr); err != nil {
		return err
	}
	if failed {
		return errRobotFailed
	}
	return nil
}
