package protocols

import (
	"context"
	"math/big"

	"github.com/dolmatovDan/lending-position-indexer/internal/models"
)

type Reader interface {
	Name() models.ProtocolKind
	Positions(ctx context.Context, wallets []string, block *big.Int, blockNumber uint64, timestamp uint64) ([]models.Position, error)
}
