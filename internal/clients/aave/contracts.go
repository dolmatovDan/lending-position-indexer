package aave

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"

	"github.com/dolmatovDan/lending-position-indexer/internal/bindings"
)

type AccountData struct {
	TotalCollateralBase *big.Int
	TotalDebtBase       *big.Int
	HealthFactor        *big.Int
}

type ReserveData struct {
	ATokenBalance *big.Int
	StableDebt    *big.Int
	VariableDebt  *big.Int
}

type TokenMeta struct {
	Symbol   string
	Decimals uint8
}

type Contracts interface {
	ReservesList(ctx context.Context, block *big.Int) ([]common.Address, error)
	UserAccountData(ctx context.Context, user common.Address, block *big.Int) (AccountData, error)
	UserReserveData(ctx context.Context, asset, user common.Address, block *big.Int) (ReserveData, error)
	AssetPrice(ctx context.Context, asset common.Address, block *big.Int) (*big.Int, error)
	TokenMeta(ctx context.Context, token common.Address, block *big.Int) (TokenMeta, error)
}

type ethContracts struct {
	caller       bind.ContractCaller
	pool         common.Address
	dataProvider common.Address
	oracle       common.Address
}

func newEthContracts(caller bind.ContractCaller, cfg Config) *ethContracts {
	return &ethContracts{
		caller:       caller,
		pool:         cfg.PoolAddress,
		dataProvider: cfg.DataProviderAddress,
		oracle:       cfg.OracleAddress,
	}
}

func (e *ethContracts) ReservesList(ctx context.Context, block *big.Int) ([]common.Address, error) {
	out, err := bindings.Call(ctx, e.caller, bindings.AavePoolABI, e.pool, block, "getReservesList")
	if err != nil {
		return nil, err
	}
	list, ok := out[0].([]common.Address)
	if !ok {
		return nil, fmt.Errorf("aave: unexpected reserves type %T", out[0])
	}
	return list, nil
}

func (e *ethContracts) UserAccountData(ctx context.Context, user common.Address, block *big.Int) (AccountData, error) {
	out, err := bindings.Call(ctx, e.caller, bindings.AavePoolABI, e.pool, block, "getUserAccountData", user)
	if err != nil {
		return AccountData{}, err
	}
	return AccountData{
		TotalCollateralBase: asBig(out[0]),
		TotalDebtBase:       asBig(out[1]),
		HealthFactor:        asBig(out[5]),
	}, nil
}

func (e *ethContracts) UserReserveData(ctx context.Context, asset, user common.Address, block *big.Int) (ReserveData, error) {
	out, err := bindings.Call(ctx, e.caller, bindings.AaveDataProviderABI, e.dataProvider, block, "getUserReserveData", asset, user)
	if err != nil {
		return ReserveData{}, err
	}
	return ReserveData{
		ATokenBalance: asBig(out[0]),
		StableDebt:    asBig(out[1]),
		VariableDebt:  asBig(out[2]),
	}, nil
}

func (e *ethContracts) AssetPrice(ctx context.Context, asset common.Address, block *big.Int) (*big.Int, error) {
	out, err := bindings.Call(ctx, e.caller, bindings.AaveOracleABI, e.oracle, block, "getAssetPrice", asset)
	if err != nil {
		return nil, err
	}
	return asBig(out[0]), nil
}

func (e *ethContracts) TokenMeta(ctx context.Context, token common.Address, block *big.Int) (TokenMeta, error) {
	symOut, err := bindings.Call(ctx, e.caller, bindings.ERC20ABI, token, block, "symbol")
	if err != nil {
		return TokenMeta{}, err
	}
	decOut, err := bindings.Call(ctx, e.caller, bindings.ERC20ABI, token, block, "decimals")
	if err != nil {
		return TokenMeta{}, err
	}
	sym, _ := symOut[0].(string)
	dec, _ := decOut[0].(uint8)
	return TokenMeta{Symbol: sym, Decimals: dec}, nil
}

func asBig(v interface{}) *big.Int {
	if b, ok := v.(*big.Int); ok {
		return b
	}
	return big.NewInt(0)
}
