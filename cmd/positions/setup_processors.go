package main

import (
	"log/slog"

	managers "github.com/dolmatovDan/lending-position-indexer/internal/managers/positions"
	"github.com/dolmatovDan/lending-position-indexer/internal/processors/blocks"
)

func setupProcessors(cl *clients, manager *managers.Manager, cfg *Config, logger *slog.Logger) *blocks.Processor {
	return blocks.New(cl.node, manager, cfg.Confirmations, logger)
}
