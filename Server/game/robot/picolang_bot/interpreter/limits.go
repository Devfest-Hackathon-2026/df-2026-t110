package interpreter

import (
	"errors"
	"fmt"

	"github.com/dvpico/Programming-Learning-AR/Server/game/robot/picolang_bot/interpreter/function"
)

var (
	ErrStepLimit      = errors.New("VM step limit exceeded")
	ErrMemoryLimit    = errors.New("VM memory limit exceeded")
	ErrCallDepthLimit = errors.New("VM call depth limit exceeded")
)

// Limits apply to the whole run, including across action suspensions. All limits
// must be positive. MaxSteps uses the logical execution counter; MaxCallDepth
// includes main. MaxMemoryBytes uses the memory store's live-entry estimate,
// including its shared root, rather than the Go process's memory usage.
type Limits struct {
	MaxSteps       uint64
	MaxMemoryBytes uint64
	MaxCallDepth   uint64
}

func (v *VM) checkLimits() error {
	for _, resource := range []struct {
		used, limit uint64
		err         error
	}{
		{v.time.Value(), v.limits.MaxSteps, ErrStepLimit},
		{v.memory.PeakMemoryUsage(), v.limits.MaxMemoryBytes, ErrMemoryLimit},
		{uint64(len(v.stack)), v.limits.MaxCallDepth, ErrCallDepthLimit},
	} {
		if resource.used > resource.limit {
			return &function.RuntimeError{Source: v.source, Err: fmt.Errorf("%w: used %d, limit %d", resource.err, resource.used, resource.limit)}
		}
	}
	return nil
}
