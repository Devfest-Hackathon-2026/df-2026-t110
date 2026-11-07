// Package builtin handles the builtins specified in Docs/picolang.md.
package builtin

import (
	"errors"
	"fmt"
	"math/rand/v2"

	actions "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/action"
	bot "github.com/dvpico/Programming-Learning-AR/Server/game/robot/picolang_bot"
	"github.com/dvpico/Programming-Learning-AR/Server/game/robot/picolang_bot/interpreter/memory"
)

// ErrUnknownFunction lets the caller distinguish user functions from builtins.
var ErrUnknownFunction = errors.New("unknown builtin function")

// ErrDivisionByZero identifies a language failure independently of Go's panic text.
var ErrDivisionByZero = errors.New("division by zero")

var signatures = bot.BuiltinSignatures()

type Result interface{ builtinResult() }

// Immediate transfers one owned memory reference to the caller. After supplying
// it to function.Machine.Resume, the caller must release that reference.
type Immediate struct{ Value memory.ID }

// Action suspends interpreter execution until the board supplies its result.
// Complete converts that result into an owned memory reference for Resume.
type Action struct {
	Action     actions.Action
	ResultType bot.Type
}

func (Immediate) builtinResult() {}
func (Action) builtinResult()    {}

// Evaluator shares one interpreter's memory and deterministic random stream.
// Create it once per run with New and keep it across function calls and action
// suspensions. Calls must be sequential. Evaluation does not charge time or emit
// variable updates: the function machine owns those operations.
type Evaluator struct {
	memory *memory.Store
	random *rand.Rand
}

func New(store *memory.Store, seed uint64) (*Evaluator, error) {
	if store == nil {
		return nil, errors.New("memory store is required")
	}
	return &Evaluator{memory: store, random: rand.New(rand.NewPCG(seed, 0))}, nil //nolint:gosec // Seeded gameplay randomness must remain reproducible.
}

// Evaluate borrows args; it never consumes or mutates their values. All argument
// expressions must already have been evaluated, including both and/or operands.
// Arity, IDs, and outer kinds are checked here. Vector element-type consistency
// is guaranteed by the preceding type-checking stage, as for function.Machine.
// Language panics, including division by zero and empty-list access, become errors.
func (e *Evaluator) Evaluate(name bot.Name, args []memory.ID) (result Result, err error) {
	if e == nil || e.memory == nil || e.random == nil {
		return nil, errors.New("builtin evaluator must be constructed with New")
	}
	sig, exists := signatures[name]
	if !exists {
		return nil, fmt.Errorf("%w: %q", ErrUnknownFunction, name)
	}
	if len(args) != len(sig.Params) {
		return nil, fmt.Errorf("builtin %q expects %d arguments, got %d", name, len(sig.Params), len(args))
	}
	values := make([]memory.Value, len(args))
	for i, id := range args {
		value, found := e.memory.Get(id)
		if !found || !matches(value, sig.Params[i]) {
			return nil, fmt.Errorf("invalid argument %d to builtin %q", i+1, name)
		}
		values[i] = value
	}
	defer func() {
		if failure := recover(); failure != nil {
			cause, ok := failure.(error)
			if !ok {
				cause = fmt.Errorf("%v", failure)
			}
			result, err = nil, fmt.Errorf("builtin %q: %w", name, cause)
		}
	}()
	if action := actionFor(name, values); action != nil {
		return Action{Action: action, ResultType: sig.Result}, nil
	}
	id, err := e.evaluate(name, args, values)
	if err != nil {
		return nil, fmt.Errorf("builtin %q: %w", name, err)
	}
	return Immediate{Value: id}, nil
}

func matches(value memory.Value, typ bot.Type) bool {
	switch typ.(type) {
	case bot.BoolType:
		_, ok := value.(memory.Bool)
		return ok
	case bot.IntegerType:
		_, ok := value.(memory.Integer)
		return ok
	case bot.RotationType:
		_, ok := value.(memory.Rotation)
		return ok
	case bot.DistanceTargetType:
		_, ok := value.(memory.DistanceTarget)
		return ok
	case bot.TerrainType:
		_, ok := value.(memory.Terrain)
		return ok
	case bot.VectorType:
		switch value.(type) {
		case memory.ListRoot, memory.ListNode:
			return true
		default:
			return false
		}
	default:
		// The only remaining builtin parameter is push_back's generic T.
		return true
	}
}
