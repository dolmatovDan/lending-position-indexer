package euler

import (
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

type StubContracts struct {
	ControllersByA map[common.Address][]common.Address
	CollateralsByA map[common.Address][]common.Address
	AssetByVault   map[common.Address]common.Address
	OracleByVault  map[common.Address]common.Address
	UoAByVault     map[common.Address]common.Address
	BalanceByVA    map[string]*big.Int
	AssetsByShares map[string]*big.Int
	DebtByVA       map[string]*big.Int
	LiquidityByVA  map[string]Liquidity
	QuoteByKey     map[string]*big.Int
	MetaByToken    map[common.Address]TokenMeta
}

func vaKey(vault, account common.Address) string { return vault.Hex() + ":" + account.Hex() }
func quoteKey(oracle, base, quote common.Address) string {
	return oracle.Hex() + ":" + base.Hex() + ":" + quote.Hex()
}

func (s *StubContracts) Controllers(ctx context.Context, account common.Address, block *big.Int) ([]common.Address, error) {
	return s.ControllersByA[account], nil
}

func (s *StubContracts) Collaterals(ctx context.Context, account common.Address, block *big.Int) ([]common.Address, error) {
	return s.CollateralsByA[account], nil
}

func (s *StubContracts) VaultAsset(ctx context.Context, vault common.Address, block *big.Int) (common.Address, error) {
	return s.AssetByVault[vault], nil
}

func (s *StubContracts) VaultOracle(ctx context.Context, vault common.Address, block *big.Int) (common.Address, error) {
	return s.OracleByVault[vault], nil
}

func (s *StubContracts) VaultUnitOfAccount(ctx context.Context, vault common.Address, block *big.Int) (common.Address, error) {
	return s.UoAByVault[vault], nil
}

func (s *StubContracts) BalanceOf(ctx context.Context, vault, account common.Address, block *big.Int) (*big.Int, error) {
	return s.BalanceByVA[vaKey(vault, account)], nil
}

func (s *StubContracts) ConvertToAssets(ctx context.Context, vault common.Address, shares *big.Int, block *big.Int) (*big.Int, error) {
	if v, ok := s.AssetsByShares[vault.Hex()+":"+shares.String()]; ok {
		return v, nil
	}
	return shares, nil
}

func (s *StubContracts) DebtOf(ctx context.Context, vault, account common.Address, block *big.Int) (*big.Int, error) {
	return s.DebtByVA[vaKey(vault, account)], nil
}

func (s *StubContracts) AccountLiquidity(ctx context.Context, vault, account common.Address, block *big.Int) (Liquidity, error) {
	return s.LiquidityByVA[vaKey(vault, account)], nil
}

func (s *StubContracts) Quote(ctx context.Context, oracle common.Address, amount *big.Int, base, quote common.Address, block *big.Int) (*big.Int, error) {
	return s.QuoteByKey[quoteKey(oracle, base, quote)], nil
}

func (s *StubContracts) TokenMeta(ctx context.Context, token common.Address, block *big.Int) (TokenMeta, error) {
	return s.MetaByToken[token], nil
}

func NewWithContracts(c Contracts, depth uint64) *Adapter {
	return newAdapter(c, depth)
}
