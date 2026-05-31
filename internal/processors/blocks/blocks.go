package blocks

import (
	"context"
	"log/slog"
	"math/big"
	"time"

	"github.com/dolmatovDan/lending-position-indexer/internal/clients/ethnode"
)

type Updater interface {
	UpdateForBlock(ctx context.Context, block *big.Int, blockNumber, timestamp uint64) error
}

type Processor struct {
	node          ethnode.Client
	updater       Updater
	confirmations uint64
	pollInterval  time.Duration
	maxRetries    int
	logger        *slog.Logger

	lastProcessed uint64
	hasProcessed  bool
}

func New(node ethnode.Client, updater Updater, confirmations uint64, logger *slog.Logger) *Processor {
	return &Processor{
		node:          node,
		updater:       updater,
		confirmations: confirmations,
		pollInterval:  12 * time.Second,
		maxRetries:    3,
		logger:        logger,
	}
}

func (p *Processor) Run(ctx context.Context) error {
	if p.node.SupportsSubscriptions() {
		if err := p.runSubscription(ctx); err != nil && ctx.Err() == nil {
			p.logger.Warn("subscription failed, falling back to polling", "err", err)
			return p.runPolling(ctx)
		}
		return nil
	}
	return p.runPolling(ctx)
}

func (p *Processor) runSubscription(ctx context.Context) error {
	heads := make(chan ethnode.Head, 16)
	sub, err := p.node.SubscribeNewHeads(ctx, heads)
	if err != nil {
		return err
	}
	defer sub.Unsubscribe()

	for {
		select {
		case <-ctx.Done():
			return nil
		case err := <-sub.Err():
			return err
		case head := <-heads:
			p.onHead(ctx, head.Number)
		}
	}
}

func (p *Processor) runPolling(ctx context.Context) error {
	ticker := time.NewTicker(p.pollInterval)
	defer ticker.Stop()

	p.pollOnce(ctx)
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			p.pollOnce(ctx)
		}
	}
}

func (p *Processor) pollOnce(ctx context.Context) {
	head, err := p.node.HeaderByNumber(ctx, nil)
	if err != nil {
		p.logger.Error("poll head failed", "err", err)
		return
	}
	p.onHead(ctx, head.Number)
}

func (p *Processor) onHead(ctx context.Context, height uint64) {
	if height < p.confirmations {
		return
	}
	target := height - p.confirmations

	from := target
	if p.hasProcessed && p.lastProcessed+1 < target {
		from = p.lastProcessed + 1
	}
	for n := from; n <= target; n++ {
		if p.hasProcessed && n <= p.lastProcessed {
			continue
		}
		if ctx.Err() != nil {
			return
		}
		p.processBlock(ctx, n)
		p.lastProcessed = n
		p.hasProcessed = true
	}
}

func (p *Processor) processBlock(ctx context.Context, number uint64) {
	var lastErr error
	for attempt := 0; attempt < p.maxRetries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return
			case <-time.After(backoff(attempt)):
			}
		}
		header, err := p.node.HeaderByNumber(ctx, new(big.Int).SetUint64(number))
		if err != nil {
			lastErr = err
			continue
		}
		if err := p.updater.UpdateForBlock(ctx, new(big.Int).SetUint64(number), header.Number, header.Time); err != nil {
			lastErr = err
			continue
		}
		return
	}
	p.logger.Error("block processing failed after retries", "block", number, "err", lastErr)
}

func backoff(attempt int) time.Duration {
	d := time.Duration(attempt) * 500 * time.Millisecond
	if d > 5*time.Second {
		return 5 * time.Second
	}
	return d
}
