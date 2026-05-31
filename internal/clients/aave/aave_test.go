package aave

import (
	"context"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"

	"github.com/dolmatovDan/lending-position-indexer/internal/models"
)

var (
	pool   = common.HexToAddress("0x0000000000000000000000000000000000000099")
	wallet = common.HexToAddress("0x0000000000000000000000000000000000000001")
	usdc   = common.HexToAddress("0x00000000000000000000000000000000000000a1")
	weth   = common.HexToAddress("0x00000000000000000000000000000000000000a2")
)

func bigStr(s string) *big.Int {
	v, _ := new(big.Int).SetString(s, 10)
	return v
}

func TestPositions_CollateralAndDebt(t *testing.T) {
	stub := &StubContracts{
		Reserves: []common.Address{usdc, weth},
		AccountByU: map[common.Address]AccountData{
			wallet: {TotalDebtBase: big.NewInt(1000), HealthFactor: bigStr("2000000000000000000")},
		},
		ReserveByUA: map[string]ReserveData{
			key(usdc, wallet): {ATokenBalance: big.NewInt(1000000000), StableDebt: big.NewInt(0), VariableDebt: big.NewInt(0)},
			key(weth, wallet): {ATokenBalance: big.NewInt(0), StableDebt: big.NewInt(0), VariableDebt: bigStr("500000000000000000")},
		},
		PriceByA: map[common.Address]*big.Int{
			usdc: big.NewInt(100000000),
			weth: bigStr("300000000000"),
		},
		MetaByA: map[common.Address]TokenMeta{
			usdc: {Symbol: "USDC", Decimals: 6},
			weth: {Symbol: "WETH", Decimals: 18},
		},
	}
	a := NewWithContracts(stub, pool)

	rows, err := a.Positions(context.Background(), []string{wallet.Hex()}, nil, 100, 1700000000)
	require.NoError(t, err)
	require.Len(t, rows, 2)

	col := findRow(rows, usdc, models.SideCollateral)
	require.NotNil(t, col)
	require.Equal(t, pool.Hex(), col.MarketID)
	require.True(t, col.Amount.Equal(decimal.RequireFromString("1000")))
	require.True(t, col.Price.Equal(decimal.RequireFromString("1")))
	require.True(t, col.HealthFactor.Equal(decimal.RequireFromString("2")))
	require.Equal(t, uint64(100), col.BlockNumber)

	debt := findRow(rows, weth, models.SideDebt)
	require.NotNil(t, debt)
	require.True(t, debt.Amount.Equal(decimal.RequireFromString("0.5")))
	require.True(t, debt.Price.Equal(decimal.RequireFromString("3000")))
	require.True(t, debt.HealthFactor.Equal(decimal.RequireFromString("2")))
}

func TestPositions_NoDebtHealthFactorZero(t *testing.T) {
	stub := &StubContracts{
		Reserves: []common.Address{usdc},
		AccountByU: map[common.Address]AccountData{
			wallet: {TotalDebtBase: big.NewInt(0), HealthFactor: bigStr("115792089237316195423570985008687907853269984665640564039457584007913129639935")},
		},
		ReserveByUA: map[string]ReserveData{
			key(usdc, wallet): {ATokenBalance: big.NewInt(1000000000), StableDebt: big.NewInt(0), VariableDebt: big.NewInt(0)},
		},
		PriceByA: map[common.Address]*big.Int{usdc: big.NewInt(100000000)},
		MetaByA:  map[common.Address]TokenMeta{usdc: {Symbol: "USDC", Decimals: 6}},
	}
	a := NewWithContracts(stub, pool)

	rows, err := a.Positions(context.Background(), []string{wallet.Hex()}, nil, 100, 1700000000)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.True(t, rows[0].HealthFactor.IsZero())
}

func TestPositions_EmptyWalletSkipped(t *testing.T) {
	stub := &StubContracts{
		Reserves:   []common.Address{usdc},
		AccountByU: map[common.Address]AccountData{wallet: {TotalDebtBase: big.NewInt(0)}},
		ReserveByUA: map[string]ReserveData{
			key(usdc, wallet): {ATokenBalance: big.NewInt(0), StableDebt: big.NewInt(0), VariableDebt: big.NewInt(0)},
		},
		PriceByA: map[common.Address]*big.Int{usdc: big.NewInt(100000000)},
		MetaByA:  map[common.Address]TokenMeta{usdc: {Symbol: "USDC", Decimals: 6}},
	}
	a := NewWithContracts(stub, pool)

	rows, err := a.Positions(context.Background(), []string{wallet.Hex()}, nil, 100, 1700000000)
	require.NoError(t, err)
	require.Empty(t, rows)
}

func findRow(rows []models.Position, token common.Address, side models.Side) *models.Position {
	for i := range rows {
		if rows[i].Token.Address == token.Hex() && rows[i].Side == side {
			return &rows[i]
		}
	}
	return nil
}
