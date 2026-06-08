package positions

import (
	"context"

	"github.com/dolmatovDan/lending-position-indexer/internal/models"
	"github.com/dolmatovDan/lending-position-indexer/internal/repositories/positions/postgres"
)

type Repository interface {
	Save(ctx context.Context, positions []models.Position) error
	ByWallet(ctx context.Context, wallet, protocol string) ([]models.Position, error)
	Ping(ctx context.Context) error
	Close()
}

func New(ctx context.Context, databaseURL string) (Repository, error) {
	return postgres.New(ctx, databaseURL)
}
