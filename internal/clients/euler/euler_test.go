package euler

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
	owner      = common.HexToAddress("0x0000000000000000000000000000000000000001")
	controller = common.HexToAddress("0x00000000000000000000000000000000000000c1")
	collVault  = common.HexToAddress("0x00000000000000000000000000000000000000c2")
	usdc       = common.HexToAddress("0x00000000000000000000000000000000000000a1")
	weth       = common.HexToAddress("0x00000000000000000000000000000000000000a2")
	oracleC    = common.HexToAddress("0x00000000000000000000000000000000000000b1")
	oracleV    = common.HexToAddress("0x00000000000000000000000000000000000000b2")
	uoa        = common.HexToAddress("0x00000000000000000000000000000000000000d1")
)

func bigStr(s string) *big.Int {
	v, _ := new(big.Int).SetString(s, 10)
	return v
}

func TestSubAccounts(t *testing.T) {
	accs := subAccounts(owner, 3)
	require.Len(t, accs, 3)
	require.Equal(t, owner, accs[0])
	require.Equal(t, byte(0x01)^byte(1), accs[1].Bytes()[19])
	require.Equal(t, byte(0x01)^byte(2), accs[2].Bytes()[19])
}

func TestPositions_WithController(t *testing.T) {
	stub := &StubContracts{
		ControllersByA: map[common.Address][]common.Address{owner: {controller}},
		CollateralsByA: map[common.Address][]common.Address{owner: {collVault}},
		AssetByVault:   map[common.Address]common.Address{controller: usdc, collVault: weth},
		OracleByVault:  map[common.Address]common.Address{controller: oracleC, collVault: oracleV},
		UoAByVault:     map[common.Address]common.Address{controller: uoa, collVault: uoa},
		DebtByVA:       map[string]*big.Int{vaKey(controller, owner): big.NewInt(1000000000)},
		BalanceByVA:    map[string]*big.Int{vaKey(collVault, owner): bigStr("1000000000000000000")},
		AssetsByShares: map[string]*big.Int{collVault.Hex() + ":" + "1000000000000000000": bigStr("2000000000000000000")},
		LiquidityByVA:  map[string]Liquidity{vaKey(controller, owner): {CollateralValue: big.NewInt(1500), LiabilityValue: big.NewInt(1000)}},
		QuoteByKey: map[string]*big.Int{
			quoteKey(oracleC, usdc, uoa): bigStr("1000000000000000000"),
			quoteKey(oracleV, weth, uoa): bigStr("3000000000000000000000"),
		},
		MetaByToken: map[common.Address]TokenMeta{
			usdc: {Symbol: "USDC", Decimals: 6},
			weth: {Symbol: "WETH", Decimals: 18},
			uoa:  {Symbol: "USD", Decimals: 18},
		},
	}
	a := NewWithContracts(stub, 1)

	rows, err := a.Positions(context.Background(), []string{owner.Hex()}, nil, 100, 1700000000)
	require.NoError(t, err)
	require.Len(t, rows, 2)

	debt := findRow(rows, usdc, models.SideDebt)
	require.NotNil(t, debt)
	require.Equal(t, controller.Hex(), debt.MarketID)
	require.True(t, debt.Amount.Equal(decimal.RequireFromString("1000")))
	require.True(t, debt.Price.Equal(decimal.RequireFromString("1")))
	require.True(t, debt.HealthFactor.Equal(decimal.RequireFromString("1.5")))

	col := findRow(rows, weth, models.SideCollateral)
	require.NotNil(t, col)
	require.Equal(t, controller.Hex(), col.MarketID)
	require.True(t, col.Amount.Equal(decimal.RequireFromString("2")))
	require.True(t, col.Price.Equal(decimal.RequireFromString("3000")))
	require.True(t, col.HealthFactor.Equal(decimal.RequireFromString("1.5")))
}

func TestPositions_CollateralOnly(t *testing.T) {
	stub := &StubContracts{
		ControllersByA: map[common.Address][]common.Address{},
		CollateralsByA: map[common.Address][]common.Address{owner: {collVault}},
		AssetByVault:   map[common.Address]common.Address{collVault: weth},
		OracleByVault:  map[common.Address]common.Address{collVault: oracleV},
		UoAByVault:     map[common.Address]common.Address{collVault: uoa},
		BalanceByVA:    map[string]*big.Int{vaKey(collVault, owner): bigStr("1000000000000000000")},
		QuoteByKey:     map[string]*big.Int{quoteKey(oracleV, weth, uoa): bigStr("3000000000000000000000")},
		MetaByToken: map[common.Address]TokenMeta{
			weth: {Symbol: "WETH", Decimals: 18},
			uoa:  {Symbol: "USD", Decimals: 18},
		},
	}
	a := NewWithContracts(stub, 1)

	rows, err := a.Positions(context.Background(), []string{owner.Hex()}, nil, 100, 1700000000)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.Equal(t, collVault.Hex(), rows[0].MarketID)
	require.True(t, rows[0].HealthFactor.IsZero())
	require.Equal(t, models.SideCollateral, rows[0].Side)
}

func findRow(rows []models.Position, token common.Address, side models.Side) *models.Position {
	for i := range rows {
		if rows[i].Token.Address == token.Hex() && rows[i].Side == side {
			return &rows[i]
		}
	}
	return nil
}
