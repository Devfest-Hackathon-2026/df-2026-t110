package domain

// GenerationQuota is a user's token budget for program generation in the
// current accounting window.
//
// Used is derived from the usage log by the caller; this type only holds the
// rule. Which window Used covers and how reservations are serialized are
// storage concerns and live outside the domain.
type GenerationQuota struct {
	GoogleSubject string
	Limit         int
	Used          int
}

// Remaining returns the tokens still available. It is never negative, even when
// Used exceeds a Limit that was lowered after usage was recorded.
func (q GenerationQuota) Remaining() int {
	return max(q.Limit-q.Used, 0)
}

// CanReserve reports whether tokens (a positive count) fit in the remaining budget.
func (q GenerationQuota) CanReserve(tokens int) bool {
	return tokens > 0 && tokens <= q.Remaining()
}
