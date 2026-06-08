package main

import (
	"log/slog"

	managers "github.com/dolmatovDan/lending-position-indexer/internal/managers/positions"
	"github.com/dolmatovDan/lending-position-indexer/internal/repositories/positions"
)

func setupManagers(cfg *Config, repo positions.Repository, cl *clients, logger *slog.Logger) *managers.Manager {
	return managers.New(repo, cl.readers, cfg.Wallets, logger)
}
