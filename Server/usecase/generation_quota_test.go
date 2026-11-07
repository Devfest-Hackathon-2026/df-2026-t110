package usecase_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/dvpico/Programming-Learning-AR/Server/usecase"
)

type quotaKey struct {
	subject string
	window  time.Time
}
type quotaRepository struct {
	mu    sync.Mutex
	usage map[quotaKey]int
	err   error
	calls int
}

func (r *quotaRepository) totalUsage() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	total := 0
	for _, used := range r.usage {
		total += used
	}
	return total
}

// ReserveGenerationTokens models the contract: the decision runs under the
// lock and a record is added only when it accepts. The limit rule itself is
// not duplicated here.
func (r *quotaRepository) ReserveGenerationTokens(_ context.Context, subject string, window time.Time, tokens int, decide func(used int) error) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.calls++
	if r.err != nil {
		return r.err
	}
	key := quotaKey{subject, window}
	if err := decide(r.usage[key]); err != nil {
		return err
	}
	r.usage[key] += tokens
	return nil
}

func TestGenerationTokenQuotaReserve(t *testing.T) {
	t.Parallel()
	const limit = 100
	for _, tc := range []struct {
		name    string
		subject string
		used    int // tokens already reserved by subject in the current window
		tokens  int
		want    error
	}{
		{"within the remaining budget", "alice", 60, 40, nil},
		{"beyond the remaining budget", "alice", 100, 1, usecase.ErrGenerationQuotaExceeded},
		{"single reservation above the limit", "alice", 0, 101, usecase.ErrGenerationQuotaExceeded},
		{"empty subject", "", 0, 1, usecase.ErrUnauthenticated},
		{"non-positive tokens", "alice", 0, 0, usecase.ErrInvalidQuotaReservation},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			repo := &quotaRepository{usage: make(map[quotaKey]int)}
			quota, err := usecase.NewGenerationTokenQuota(repo, limit, time.Hour, nil)
			if err != nil {
				t.Fatal(err)
			}
			if tc.used > 0 {
				if err := quota.Reserve(context.Background(), tc.subject, tc.used); err != nil {
					t.Fatalf("setup reservation: %v", err)
				}
			}
			err = quota.Reserve(context.Background(), tc.subject, tc.tokens)
			if !errors.Is(err, tc.want) {
				t.Errorf("Reserve(%q, %d) = %v, want %v", tc.subject, tc.tokens, err, tc.want)
			}
		})
	}
}

func TestGenerationTokenQuotaIsPerSubject(t *testing.T) {
	t.Parallel()
	repo := &quotaRepository{usage: make(map[quotaKey]int)}
	quota, err := usecase.NewGenerationTokenQuota(repo, 100, time.Hour, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := quota.Reserve(context.Background(), "alice", 100); err != nil {
		t.Fatalf("alice: %v", err)
	}
	if err := quota.Reserve(context.Background(), "bob", 100); err != nil {
		t.Errorf("bob must have an independent budget: %v", err)
	}
}

func TestGenerationTokenQuotaConcurrentReservations(t *testing.T) {
	t.Parallel()
	repo := &quotaRepository{usage: make(map[quotaKey]int)}
	quota, err := usecase.NewGenerationTokenQuota(repo, 10, time.Hour, nil)
	if err != nil {
		t.Fatal(err)
	}
	results := make(chan error, 20)
	for range 20 {
		go func() { results <- quota.Reserve(context.Background(), "alice", 1) }()
	}
	var accepted, rejected int
	for range 20 {
		err := <-results
		switch {
		case err == nil:
			accepted++
		case errors.Is(err, usecase.ErrGenerationQuotaExceeded):
			rejected++
		default:
			t.Errorf("unexpected error: %v", err)
		}
	}
	if accepted != 10 || rejected != 10 {
		t.Errorf("accepted %d, rejected %d", accepted, rejected)
	}
}

func TestGenerationTokenQuotaErrors(t *testing.T) {
	t.Parallel()
	repo := &quotaRepository{usage: make(map[quotaKey]int), err: errors.New("storage failed")}
	for _, tc := range []struct {
		repo   usecase.GenerationQuotaRepository
		limit  int
		window time.Duration
	}{{nil, 1, time.Hour}, {repo, 0, time.Hour}, {repo, 1, 0}} {
		if _, err := usecase.NewGenerationTokenQuota(tc.repo, tc.limit, tc.window, nil); err == nil {
			t.Errorf("invalid configuration accepted: %+v", tc)
		}
	}
	quota, err := usecase.NewGenerationTokenQuota(repo, 10, time.Hour, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := quota.Reserve(context.Background(), "alice", 1); !errors.Is(err, repo.err) {
		t.Errorf("storage error = %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := quota.Reserve(ctx, "alice", 1); !errors.Is(err, context.Canceled) {
		t.Errorf("canceled error = %v", err)
	}
}

func TestGenerationTokenQuotaWindowBoundaries(t *testing.T) {
	t.Parallel()
	start := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	now := start
	repo := &quotaRepository{usage: make(map[quotaKey]int)}
	quota, err := usecase.NewGenerationTokenQuota(repo, 10, time.Hour, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	reserve := func(at time.Time, tokens int) error {
		now = at
		return quota.Reserve(context.Background(), "alice", tokens)
	}
	if err := reserve(start, 10); err != nil {
		t.Fatalf("first reservation: %v", err)
	}
	if err := reserve(start.Add(59*time.Minute+59*time.Second), 1); !errors.Is(err, usecase.ErrGenerationQuotaExceeded) {
		t.Errorf("last second of the window = %v, want quota exceeded", err)
	}
	if err := reserve(start.Add(time.Hour), 10); err != nil {
		t.Errorf("first instant of the next window: %v", err)
	}
}

func TestGenerationTokenQuotaRejectsOversizedReservationWithoutRepository(t *testing.T) {
	t.Parallel()
	repo := &quotaRepository{usage: make(map[quotaKey]int)}
	quota, err := usecase.NewGenerationTokenQuota(repo, 10, time.Hour, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := quota.Reserve(context.Background(), "alice", 11); !errors.Is(err, usecase.ErrGenerationQuotaExceeded) {
		t.Errorf("error = %v", err)
	}
	if repo.calls != 0 {
		t.Errorf("repository calls = %d, want 0", repo.calls)
	}
}
