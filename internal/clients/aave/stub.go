package aave

import (
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

type StubContracts struct {
	Reserves    []common.Address
	AccountByU  map[common.Address]AccountData
	ReserveByUA map[string]ReserveData
	PriceByA    map[common.Address]*big.Int
	MetaByA     map[common.Address]TokenMeta
}

func key(asset, user common.Address) string {
	return asset.Hex() + ":" + user.Hex()
}

func (s *StubContracts) ReservesList(ctx context.Context, block *big.Int) ([]common.Address, error) {
	return s.Reserves, nil
}

func (s *StubContracts) UserAccountData(ctx context.Context, user common.Address, block *big.Int) (AccountData, error) {
	return s.AccountByU[user], nil
}

func (s *StubContracts) UserReserveData(ctx context.Context, asset, user common.Address, block *big.Int) (ReserveData, error) {
	return s.ReserveByUA[key(asset, user)], nil
}

func (s *StubContracts) AssetPrice(ctx context.Context, asset common.Address, block *big.Int) (*big.Int, error) {
	return s.PriceByA[asset], nil
}

func (s *StubContracts) TokenMeta(ctx context.Context, token common.Address, block *big.Int) (TokenMeta, error) {
	return s.MetaByA[token], nil
}

func NewWithContracts(c Contracts, pool common.Address) *Adapter {
	return newAdapter(c, pool)
}
