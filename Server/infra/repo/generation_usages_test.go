package repo

import (
	"context"
	"errors"
	"math"
	"sync"
	"testing"
	"time"

	"github.com/dvpico/Programming-Learning-AR/Server/usecase"
)

func TestGenerationUsagesReserveGenerationTokens(t *testing.T) {
	t.Parallel()
	pool := newTestPool(t)
	mustCreateUser(t, pool, "sub", nil)
	mustCreateUser(t, pool, "other", nil)
	usages := NewGenerationUsages(pool)
	ctx := context.Background()
	past := time.Now().Add(-time.Hour)
	errReject := errors.New("reject")

	reserve := func(subject string, windowStart time.Time, tokens int, decision error) (used int, err error) {
		used = -1
		err = usages.ReserveGenerationTokens(ctx, subject, windowStart, tokens, func(u int) error {
			used = u
			return decision
		})
		return used, err
	}

	if used, err := reserve("sub", past, 10, nil); err != nil || used != 0 {
		t.Errorf("first reservation: used = %d, err = %v; want 0, nil", used, err)
	}
	if used, err := reserve("sub", past, 5, errReject); !errors.Is(err, errReject) || used != 10 {
		t.Errorf("rejected reservation: used = %d, err = %v; want 10, %v", used, err, errReject)
	}
	if used, err := reserve("sub", past, 1, nil); err != nil || used != 10 {
		t.Errorf("after rejection: used = %d, err = %v; want 10 (rejected tokens unrecorded), nil", used, err)
	}
	if used, err := reserve("other", past, 1, nil); err != nil || used != 0 {
		t.Errorf("other subject: used = %d, err = %v; want 0, nil", used, err)
	}
	if used, err := reserve("sub", time.Now().Add(time.Hour), 1, nil); err != nil || used != 0 {
		t.Errorf("later window: used = %d, err = %v; want 0 (earlier usage excluded), nil", used, err)
	}
}

func TestGenerationUsagesReserveGenerationTokensRejects(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		subject string
		tokens  int
		wantErr error
	}{
		{name: "zero tokens", subject: "sub", tokens: 0, wantErr: usecase.ErrInvalidQuotaReservation},
		{name: "tokens beyond int32", subject: "sub", tokens: math.MaxInt32 + 1, wantErr: usecase.ErrInvalidQuotaReservation},
		{name: "missing user", subject: "missing", tokens: 1, wantErr: usecase.ErrNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			pool := newTestPool(t)
			mustCreateUser(t, pool, "sub", nil)

			err := NewGenerationUsages(pool).ReserveGenerationTokens(context.Background(), tt.subject, time.Now(), tt.tokens, func(int) error { return nil })
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("ReserveGenerationTokens() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

// Concurrent reservations must observe each other's usage, so the total never
// exceeds the limit enforced by decide.
func TestGenerationUsagesReserveGenerationTokensSerializes(t *testing.T) {
	t.Parallel()
	pool := newTestPool(t)
	mustCreateUser(t, pool, "sub", nil)
	usages := NewGenerationUsages(pool)
	const limit, tokens, attempts = 50, 10, 20
	errExceeded := errors.New("exceeded")
	start := time.Now().Add(-time.Hour)

	var wg sync.WaitGroup
	results := make(chan error, attempts)
	for range attempts {
		wg.Go(func() {
			results <- usages.ReserveGenerationTokens(context.Background(), "sub", start, tokens, func(used int) error {
				if used+tokens > limit {
					return errExceeded
				}
				return nil
			})
		})
	}
	wg.Wait()
	close(results)

	succeeded := 0
	for err := range results {
		switch {
		case err == nil:
			succeeded++
		case !errors.Is(err, errExceeded):
			t.Errorf("ReserveGenerationTokens() unexpected error = %v", err)
		}
	}
	if succeeded != limit/tokens {
		t.Errorf("succeeded reservations = %d, want %d", succeeded, limit/tokens)
	}
}

// While one reservation is deciding, a reservation by the same subject waits
// for it and then observes its usage, whereas another subject's reservation
// proceeds without waiting.
func TestGenerationUsagesReserveGenerationTokensLocksPerSubject(t *testing.T) {
	t.Parallel()

	pool := newTestPool(t)
	mustCreateUser(t, pool, "sub", nil)
	mustCreateUser(t, pool, "other", nil)
	usages := NewGenerationUsages(pool)
	ctx := context.Background()
	start := time.Now().Add(-time.Hour)
	const tokens = 10

	deciding := make(chan struct{})
	release := make(chan struct{})
	firstDone := make(chan error, 1)

	go func() {
		firstDone <- usages.ReserveGenerationTokens(ctx, "sub", start, tokens, func(int) error {
			close(deciding)
			<-release
			return nil
		})
	}()
	<-deciding

	type result struct {
		used int
		err  error
	}
	sameDone := make(chan result, 1)
	go func() {
		used := -1
		err := usages.ReserveGenerationTokens(ctx, "sub", start, tokens, func(u int) error {
			used = u
			return nil
		})
		sameDone <- result{used, err}
	}()

	otherCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := usages.ReserveGenerationTokens(otherCtx, "other", start, tokens, func(int) error { return nil }); err != nil {
		t.Errorf("other subject's reservation while sub is locked: error = %v, want nil", err)
	}
	select {
	case r := <-sameDone:
		t.Errorf("same subject's reservation finished while sub was locked: %+v", r)
	case <-time.After(200 * time.Millisecond):
	}

	close(release)
	if err := <-firstDone; err != nil {
		t.Errorf("first reservation error = %v", err)
	}
	select {
	case r := <-sameDone:
		if r.err != nil || r.used != tokens {
			t.Errorf("same subject's reservation: used = %d, err = %v; want %d, nil", r.used, r.err, tokens)
		}
	case <-time.After(5 * time.Second):
		t.Error("same subject's reservation did not finish after the lock was released")
	}
}
