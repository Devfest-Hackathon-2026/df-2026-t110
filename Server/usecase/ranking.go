package usecase

import (
	"context"

	"github.com/dvpico/Programming-Learning-AR/Server/domain"
)

type RankingRepository interface {
	// Ranking lists named challengers by rating, highest first.
	Ranking(ctx context.Context) ([]domain.RankingEntry, error)
}

type Rankings struct{ repo RankingRepository }

func NewRankings(repo RankingRepository) *Rankings { return &Rankings{repo: repo} }

func (r *Rankings) List(ctx context.Context) ([]domain.RankingEntry, error) {
	return r.repo.Ranking(ctx)
}
