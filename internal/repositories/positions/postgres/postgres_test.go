package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/pashagolub/pgxmock/v4"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"

	"github.com/dolmatovDan/lending-position-indexer/internal/models"
)

func samplePosition() models.Position {
	return models.Position{
		Protocol:      models.ProtocolAaveV3,
		WalletAddress: "0x0000000000000000000000000000000000000001",
		MarketID:      "0x0000000000000000000000000000000000000099",
		Token:         models.Token{Address: "0x00000000000000000000000000000000000000a1", Symbol: "USDC", Decimals: 6},
		Side:          models.SideCollateral,
		Amount:        decimal.RequireFromString("1000"),
		Price:         decimal.RequireFromString("1"),
		HealthFactor:  decimal.RequireFromString("2"),
		BlockNumber:   100,
		Timestamp:     time.Unix(1700000000, 0).UTC(),
	}
}

func TestSave_Upsert(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	p := samplePosition()
	eb := mock.ExpectBatch()
	eb.ExpectExec("INSERT INTO positions").
		WithArgs("aave-v3", p.WalletAddress, p.MarketID, p.Token.Address, "USDC", int16(6),
			"collateral", "1000", "1", "2", int64(100), p.Timestamp).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	repo := NewWithPool(mock)
	require.NoError(t, repo.Save(context.Background(), []models.Position{p}))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSave_Empty(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	repo := NewWithPool(mock)
	require.NoError(t, repo.Save(context.Background(), nil))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestByWallet(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	p := samplePosition()
	rows := pgxmock.NewRows([]string{
		"protocol", "wallet_address", "market_id", "token_address", "token_symbol",
		"token_decimals", "side", "amount", "price", "health_factor", "block_number", "block_timestamp",
	}).AddRow(
		string(p.Protocol), p.WalletAddress, p.MarketID, p.Token.Address, p.Token.Symbol,
		int16(6), string(p.Side), "1000", "1", "2", uint64(100), p.Timestamp,
	)

	mock.ExpectQuery("SELECT protocol").
		WithArgs(p.WalletAddress, "").
		WillReturnRows(rows)

	repo := NewWithPool(mock)
	got, err := repo.ByWallet(context.Background(), p.WalletAddress, "")
	require.NoError(t, err)
	require.Len(t, got, 1)
	require.Equal(t, models.ProtocolAaveV3, got[0].Protocol)
	require.True(t, got[0].Amount.Equal(decimal.RequireFromString("1000")))
	require.True(t, got[0].HealthFactor.Equal(decimal.RequireFromString("2")))
	require.Equal(t, uint8(6), got[0].Token.Decimals)
	require.NoError(t, mock.ExpectationsWereMet())
}
