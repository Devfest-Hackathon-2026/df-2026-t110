package repo

import (
	"context"
	"sync"
	"time"

	"github.com/dvpico/Programming-Learning-AR/Server/domain"
	"github.com/dvpico/Programming-Learning-AR/Server/usecase"
)

var (
	_ usecase.LoginAttemptRepository = (*LoginAttempts)(nil)
	_ usecase.UserRepository         = (*Users)(nil)
	_ usecase.SessionRepository      = (*Sessions)(nil)
)

// LoginAttempts keeps login attempts in memory. Attempts that are never
// consumed are pruned once expired so abandoned logins do not accumulate.
type LoginAttempts struct {
	mu       sync.Mutex
	attempts map[string]usecase.LoginAttempt
	now      func() time.Time
}

// NewLoginAttempts returns an empty store; now defaults to time.Now when nil.
func NewLoginAttempts(now func() time.Time) *LoginAttempts {
	if now == nil {
		now = time.Now
	}
	return &LoginAttempts{attempts: map[string]usecase.LoginAttempt{}, now: now}
}

func (r *LoginAttempts) Save(_ context.Context, attempt usecase.LoginAttempt) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	now := r.now()
	for id, a := range r.attempts {
		if !now.Before(a.ExpiresAt) {
			delete(r.attempts, id)
		}
	}
	r.attempts[attempt.ID] = attempt
	return nil
}

func (r *LoginAttempts) Consume(_ context.Context, id string) (usecase.LoginAttempt, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	attempt, ok := r.attempts[id]
	if !ok {
		return usecase.LoginAttempt{}, usecase.ErrNotFound
	}
	delete(r.attempts, id)
	return attempt, nil
}

// Users keeps users in memory, indexed by ID and Google subject.
type Users struct {
	mu        sync.RWMutex
	byID      map[string]domain.User
	bySubject map[string]string
}

func NewUsers() *Users {
	return &Users{byID: map[string]domain.User{}, bySubject: map[string]string{}}
}

func (r *Users) CreateIfAbsentByGoogleSubject(_ context.Context, candidate domain.User) (domain.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if id, ok := r.bySubject[candidate.GoogleSubject]; ok {
		return r.byID[id], nil
	}
	r.byID[candidate.ID] = candidate
	r.bySubject[candidate.GoogleSubject] = candidate.ID
	return candidate, nil
}

func (r *Users) Get(_ context.Context, id string) (domain.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	user, ok := r.byID[id]
	if !ok {
		return domain.User{}, usecase.ErrNotFound
	}
	return user, nil
}

// Sessions keeps sessions in memory.
type Sessions struct {
	mu       sync.RWMutex
	sessions map[string]domain.Session
}

func NewSessions() *Sessions {
	return &Sessions{sessions: map[string]domain.Session{}}
}

func (r *Sessions) Save(_ context.Context, session domain.Session) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.sessions[session.ID] = session
	return nil
}

func (r *Sessions) Get(_ context.Context, id string) (domain.Session, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	session, ok := r.sessions[id]
	if !ok {
		return domain.Session{}, usecase.ErrNotFound
	}
	return session, nil
}

func (r *Sessions) Delete(_ context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.sessions, id)
	return nil
}
