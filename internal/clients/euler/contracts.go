package euler

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"

	"github.com/dolmatovDan/lending-position-indexer/internal/bindings"
)

type TokenMeta struct {
	Symbol   string
	Decimals uint8
}

type Liquidity struct {
	CollateralValue *big.Int
	LiabilityValue  *big.Int
}

type Contracts interface {
	Controllers(ctx context.Context, account common.Address, block *big.Int) ([]common.Address, error)
	Collaterals(ctx context.Context, account common.Address, block *big.Int) ([]common.Address, error)
	VaultAsset(ctx context.Context, vault common.Address, block *big.Int) (common.Address, error)
	VaultOracle(ctx context.Context, vault common.Address, block *big.Int) (common.Address, error)
	VaultUnitOfAccount(ctx context.Context, vault common.Address, block *big.Int) (common.Address, error)
	BalanceOf(ctx context.Context, vault, account common.Address, block *big.Int) (*big.Int, error)
	ConvertToAssets(ctx context.Context, vault common.Address, shares *big.Int, block *big.Int) (*big.Int, error)
	DebtOf(ctx context.Context, vault, account common.Address, block *big.Int) (*big.Int, error)
	AccountLiquidity(ctx context.Context, vault, account common.Address, block *big.Int) (Liquidity, error)
	Quote(ctx context.Context, oracle common.Address, amount *big.Int, base, quote common.Address, block *big.Int) (*big.Int, error)
	TokenMeta(ctx context.Context, token common.Address, block *big.Int) (TokenMeta, error)
}

type ethContracts struct {
	caller bind.ContractCaller
	evc    common.Address
}

func newEthContracts(caller bind.ContractCaller, cfg Config) *ethContracts {
	return &ethContracts{caller: caller, evc: cfg.EVCAddress}
}

func (e *ethContracts) addressList(ctx context.Context, method string, account common.Address, block *big.Int) ([]common.Address, error) {
	out, err := bindings.Call(ctx, e.caller, bindings.EulerEVCABI, e.evc, block, method, account)
	if err != nil {
		return nil, err
	}
	list, ok := out[0].([]common.Address)
	if !ok {
		return nil, fmt.Errorf("euler: unexpected list type %T", out[0])
	}
	return list, nil
}

func (e *ethContracts) Controllers(ctx context.Context, account common.Address, block *big.Int) ([]common.Address, error) {
	return e.addressList(ctx, "getControllers", account, block)
}

func (e *ethContracts) Collaterals(ctx context.Context, account common.Address, block *big.Int) ([]common.Address, error) {
	return e.addressList(ctx, "getCollaterals", account, block)
}

func (e *ethContracts) vaultAddress(ctx context.Context, vault common.Address, method string, block *big.Int) (common.Address, error) {
	out, err := bindings.Call(ctx, e.caller, bindings.EulerEVaultABI, vault, block, method)
	if err != nil {
		return common.Address{}, err
	}
	addr, ok := out[0].(common.Address)
	if !ok {
		return common.Address{}, fmt.Errorf("euler: unexpected address type %T", out[0])
	}
	return addr, nil
}

func (e *ethContracts) VaultAsset(ctx context.Context, vault common.Address, block *big.Int) (common.Address, error) {
	return e.vaultAddress(ctx, vault, "asset", block)
}

func (e *ethContracts) VaultOracle(ctx context.Context, vault common.Address, block *big.Int) (common.Address, error) {
	return e.vaultAddress(ctx, vault, "oracle", block)
}

func (e *ethContracts) VaultUnitOfAccount(ctx context.Context, vault common.Address, block *big.Int) (common.Address, error) {
	return e.vaultAddress(ctx, vault, "unitOfAccount", block)
}

func (e *ethContracts) BalanceOf(ctx context.Context, vault, account common.Address, block *big.Int) (*big.Int, error) {
	out, err := bindings.Call(ctx, e.caller, bindings.EulerEVaultABI, vault, block, "balanceOf", account)
	if err != nil {
		return nil, err
	}
	return asBig(out[0]), nil
}

func (e *ethContracts) ConvertToAssets(ctx context.Context, vault common.Address, shares *big.Int, block *big.Int) (*big.Int, error) {
	out, err := bindings.Call(ctx, e.caller, bindings.EulerEVaultABI, vault, block, "convertToAssets", shares)
	if err != nil {
		return nil, err
	}
	return asBig(out[0]), nil
}

func (e *ethContracts) DebtOf(ctx context.Context, vault, account common.Address, block *big.Int) (*big.Int, error) {
	out, err := bindings.Call(ctx, e.caller, bindings.EulerEVaultABI, vault, block, "debtOf", account)
	if err != nil {
		return nil, err
	}
	return asBig(out[0]), nil
}

func (e *ethContracts) AccountLiquidity(ctx context.Context, vault, account common.Address, block *big.Int) (Liquidity, error) {
	out, err := bindings.Call(ctx, e.caller, bindings.EulerEVaultABI, vault, block, "accountLiquidity", account, false)
	if err != nil {
		return Liquidity{}, err
	}
	return Liquidity{CollateralValue: asBig(out[0]), LiabilityValue: asBig(out[1])}, nil
}

func (e *ethContracts) Quote(ctx context.Context, oracle common.Address, amount *big.Int, base, quote common.Address, block *big.Int) (*big.Int, error) {
	out, err := bindings.Call(ctx, e.caller, bindings.EulerOracleABI, oracle, block, "getQuote", amount, base, quote)
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
