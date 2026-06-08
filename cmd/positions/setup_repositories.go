package main

import (
	"context"

	"github.com/dolmatovDan/lending-position-indexer/internal/repositories/positions"
	"github.com/dolmatovDan/lending-position-indexer/migrations/postgres"
)

func setupRepositories(ctx context.Context, cfg *Config) (positions.Repository, error) {
	if err := migrations.Run(cfg.DatabaseURL); err != nil {
		return nil, err
	}
	return positions.New(ctx, cfg.DatabaseURL)
}
