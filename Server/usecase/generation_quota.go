package usecase

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/dvpico/Programming-Learning-AR/Server/domain"
)

var (
	ErrGenerationQuotaExceeded = errors.New("program generation quota exceeded")
	ErrInvalidQuotaReservation = errors.New("invalid quota reservation")
)

// GenerationQuota reserves a token budget for a generation attempt. Reservations
// are consumed even if generation fails.
type GenerationQuota interface {
	Reserve(ctx context.Context, subject string, maxTokens int) error
}

// GenerationQuotaRepository serializes reservations per subject. It must lock
// the subject, pass the tokens already reserved since windowStart to decide, and
// append a usage record of tokens only if decide returns nil; otherwise it
// returns decide's error without recording. The lock must be released before
// returning so that it is not held while generation runs.
type GenerationQuotaRepository interface {
	ReserveGenerationTokens(ctx context.Context, subject string, windowStart time.Time, tokens int, decide func(used int) error) error
}

// GenerationTokenQuota enforces a fixed token budget per Google subject and time window.
type GenerationTokenQuota struct {
	repo   GenerationQuotaRepository
	limit  int
	window time.Duration
	now    func() time.Time
}

// NewGenerationTokenQuota requires a positive limit and window. now can be nil
// to use the system clock; window boundaries use time.Time.Truncate in UTC.
func NewGenerationTokenQuota(repo GenerationQuotaRepository, limit int, window time.Duration, now func() time.Time) (*GenerationTokenQuota, error) {
	if repo == nil || limit <= 0 || window <= 0 {
		return nil, errors.New("invalid generation quota configuration")
	}
	if now == nil {
		now = time.Now
	}
	return &GenerationTokenQuota{repo: repo, limit: limit, window: window, now: now}, nil
}

func (q *GenerationTokenQuota) Reserve(ctx context.Context, subject string, maxTokens int) error {
	if strings.TrimSpace(subject) == "" {
		return ErrUnauthenticated
	}
	if maxTokens <= 0 {
		return ErrInvalidQuotaReservation
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if maxTokens > q.limit {
		return ErrGenerationQuotaExceeded
	}
	start := q.now().UTC().Truncate(q.window)
	decide := func(used int) error {
		if !(domain.GenerationQuota{GoogleSubject: subject, Limit: q.limit, Used: used}).CanReserve(maxTokens) {
			return ErrGenerationQuotaExceeded
		}
		return nil
	}
	if err := q.repo.ReserveGenerationTokens(ctx, subject, start, maxTokens, decide); err != nil {
		return fmt.Errorf("reserve generation tokens: %w", err)
	}
	return nil
}
