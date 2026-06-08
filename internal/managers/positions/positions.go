package positions

import (
	"context"
	"log/slog"
	"math/big"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"

	"github.com/dolmatovDan/lending-position-indexer/internal/models"
	"github.com/dolmatovDan/lending-position-indexer/internal/protocols"
	repo "github.com/dolmatovDan/lending-position-indexer/internal/repositories/positions"
)

type Manager struct {
	repo        repo.Repository
	readers     []protocols.Reader
	wallets     []string
	logger      *slog.Logger
	concurrency int
}

func New(repository repo.Repository, readers []protocols.Reader, wallets []string, logger *slog.Logger) *Manager {
	return &Manager{
		repo:        repository,
		readers:     readers,
		wallets:     wallets,
		logger:      logger,
		concurrency: len(readers),
	}
}

func (m *Manager) UpdateForBlock(ctx context.Context, block *big.Int, blockNumber, timestamp uint64) error {
	start := time.Now()

	var (
		mu       sync.Mutex
		all      []models.Position
		wg       sync.WaitGroup
		sem      = make(chan struct{}, max(1, m.concurrency))
	)

	for _, reader := range m.readers {
		wg.Add(1)
		sem <- struct{}{}
		go func(r protocols.Reader) {
			defer wg.Done()
			defer func() { <-sem }()

			rows, err := r.Positions(ctx, m.wallets, block, blockNumber, timestamp)
			if err != nil {
				m.logger.Error("protocol positions failed",
					"protocol", r.Name(), "block", blockNumber, "err", err)
				return
			}
			mu.Lock()
			all = append(all, rows...)
			mu.Unlock()
		}(reader)
	}
	wg.Wait()

	if err := m.repo.Save(ctx, all); err != nil {
		m.logger.Error("save positions failed", "block", blockNumber, "err", err)
		return err
	}

	m.logger.Info("block processed",
		"block", blockNumber, "positions", len(all), "duration_ms", time.Since(start).Milliseconds())
	return nil
}

func (m *Manager) Positions(ctx context.Context, wallet, protocol string) ([]models.Position, error) {
	if !common.IsHexAddress(wallet) {
		return nil, models.ErrInvalidWallet
	}
	if protocol != "" && !knownProtocol(protocol) {
		return nil, models.ErrInvalidProtocol
	}
	return m.repo.ByWallet(ctx, common.HexToAddress(wallet).Hex(), protocol)
}

func knownProtocol(p string) bool {
	switch models.ProtocolKind(p) {
	case models.ProtocolAaveV3, models.ProtocolEulerV2:
		return true
	default:
		return false
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
