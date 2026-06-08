package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"

	"github.com/dolmatovDan/lending-position-indexer/internal/models"
)

type pgxPool interface {
	Ping(ctx context.Context) error
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	SendBatch(ctx context.Context, batch *pgx.Batch) pgx.BatchResults
	Close()
}

type Repo struct {
	pool pgxPool
}

func New(ctx context.Context, databaseURL string) (*Repo, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("positions/postgres: connect: %w", err)
	}
	return &Repo{pool: pool}, nil
}

func NewWithPool(pool pgxPool) *Repo {
	return &Repo{pool: pool}
}

func (r *Repo) Close() { r.pool.Close() }

func (r *Repo) Ping(ctx context.Context) error { return r.pool.Ping(ctx) }

const upsertSQL = `
INSERT INTO positions (
    protocol, wallet_address, market_id, token_address, token_symbol, token_decimals,
    side, amount, price, health_factor, block_number, block_timestamp
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
ON CONFLICT (protocol, wallet_address, market_id, token_address, side, block_number)
DO UPDATE SET
    token_symbol   = EXCLUDED.token_symbol,
    token_decimals = EXCLUDED.token_decimals,
    amount         = EXCLUDED.amount,
    price          = EXCLUDED.price,
    health_factor  = EXCLUDED.health_factor,
    block_timestamp = EXCLUDED.block_timestamp
`

func (r *Repo) Save(ctx context.Context, positions []models.Position) error {
	if len(positions) == 0 {
		return nil
	}
	batch := &pgx.Batch{}
	for _, p := range positions {
		batch.Queue(upsertSQL,
			string(p.Protocol), p.WalletAddress, p.MarketID, p.Token.Address,
			p.Token.Symbol, int16(p.Token.Decimals), string(p.Side),
			p.Amount.String(), p.Price.String(), p.HealthFactor.String(),
			int64(p.BlockNumber), p.Timestamp,
		)
	}
	br := r.pool.SendBatch(ctx, batch)
	defer br.Close()
	for range positions {
		if _, err := br.Exec(); err != nil {
			return fmt.Errorf("positions/postgres: save: %w", err)
		}
	}
	return nil
}

const selectSQL = `
SELECT protocol, wallet_address, market_id, token_address, token_symbol, token_decimals,
       side, amount::text, price::text, health_factor::text, block_number, block_timestamp
FROM positions
WHERE wallet_address = $1
  AND ($2 = '' OR protocol = $2)
  AND block_number = (SELECT MAX(block_number) FROM positions WHERE wallet_address = $1)
ORDER BY protocol, market_id, side, token_address
`

func (r *Repo) ByWallet(ctx context.Context, wallet, protocol string) ([]models.Position, error) {
	rows, err := r.pool.Query(ctx, selectSQL, wallet, protocol)
	if err != nil {
		return nil, fmt.Errorf("positions/postgres: query: %w", err)
	}
	defer rows.Close()

	var out []models.Position
	for rows.Next() {
		var (
			p                          models.Position
			decimals                   int16
			amount, price, healthFactor string
		)
		if err := rows.Scan(
			&p.Protocol, &p.WalletAddress, &p.MarketID, &p.Token.Address, &p.Token.Symbol,
			&decimals, &p.Side, &amount, &price, &healthFactor, &p.BlockNumber, &p.Timestamp,
		); err != nil {
			return nil, fmt.Errorf("positions/postgres: scan: %w", err)
		}
		p.Token.Decimals = uint8(decimals)
		if p.Amount, err = decimal.NewFromString(amount); err != nil {
			return nil, err
		}
		if p.Price, err = decimal.NewFromString(price); err != nil {
			return nil, err
		}
		if p.HealthFactor, err = decimal.NewFromString(healthFactor); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}
