package robot

import (
	"github.com/dvpico/Programming-Learning-AR/Server/game"
	"github.com/dvpico/Programming-Learning-AR/Server/game/robot/picolang_bot/index"
	"github.com/dvpico/Programming-Learning-AR/Server/game/robot/picolang_bot/interpreter"
	picolangparse "github.com/dvpico/Programming-Learning-AR/Server/game/robot/picolang_bot/parse"
	"github.com/dvpico/Programming-Learning-AR/Server/game/robot/picolang_bot/typecheck"
)

// PicolangBot owns a validated program and its VM. Start, Next, Close and Collect
// use the VM's existing lifecycle and event delivery behavior.
type PicolangBot struct {
	*interpreter.VM
}

var _ game.Robot = (*PicolangBot)(nil)

// NewPicolangBot parses the versioned program JSON, resolves variable bindings,
// checks types and constructs a fresh VM. It does not execute the program.
// On any failure it returns nil and the original error, preserving Diagnostics
// from indexing and type checking. The source bytes may be reused after return.
// logExecution controls whether the VM creates execution events.
func NewPicolangBot(source []byte, seed uint64, limits interpreter.Limits, logExecution bool) (*PicolangBot, error) {
	parsed, err := picolangparse.Parse(source)
	if err != nil {
		return nil, err
	}
	indexed, err := index.Index(parsed)
	if err != nil {
		return nil, err
	}
	typed, err := typecheck.TypeCheck(indexed)
	if err != nil {
		return nil, err
	}
	vm, err := interpreter.New(typed, seed, limits, logExecution)
	if err != nil {
		return nil, err
	}
	return &PicolangBot{VM: vm}, nil
}
