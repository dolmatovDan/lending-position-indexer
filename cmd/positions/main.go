package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	httpapi "github.com/dolmatovDan/lending-position-indexer/internal/api/http"
)

func main() {
	if err := run(); err != nil {
		slog.Error("fatal", "err", err)
		os.Exit(1)
	}
}

func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg, err := LoadConfig()
	if err != nil {
		return err
	}

	logger := setupCore(cfg)
	logger.Info("service starting", "wallets", len(cfg.Wallets), "confirmations", cfg.Confirmations)

	cl, err := setupClients(ctx, cfg)
	if err != nil {
		return err
	}
	defer cl.node.Close()

	repo, err := setupRepositories(ctx, cfg)
	if err != nil {
		return err
	}
	defer repo.Close()

	manager := setupManagers(cfg, repo, cl, logger)
	processor := setupProcessors(cl, manager, cfg, logger)

	servant := httpapi.NewServant(manager, repo, rpcPinger{node: cl.node}, logger)
	srv := &http.Server{Addr: ":" + cfg.HTTPPort, Handler: servant.Router()}

	go func() {
		logger.Info("http server listening", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("http server failed", "err", err)
			stop()
		}
	}()

	go func() {
		if err := processor.Run(ctx); err != nil {
			logger.Error("processor stopped", "err", err)
			stop()
		}
	}()

	<-ctx.Done()
	logger.Info("shutdown signal received")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("http shutdown failed", "err", err)
	}

	logger.Info("service stopped")
	return nil
}
