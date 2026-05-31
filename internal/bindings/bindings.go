package bindings

import (
	"context"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"

	abifs "github.com/dolmatovDan/lending-position-indexer/abi"
)

//go:generate sh -c "abigen --abi ../../abi/aave_pool.json --pkg bindings --type AavePool --out gen_aave_pool.go"
//go:generate sh -c "abigen --abi ../../abi/aave_data_provider.json --pkg bindings --type AaveDataProvider --out gen_aave_data_provider.go"
//go:generate sh -c "abigen --abi ../../abi/aave_oracle.json --pkg bindings --type AaveOracle --out gen_aave_oracle.go"
//go:generate sh -c "abigen --abi ../../abi/euler_evc.json --pkg bindings --type EulerEVC --out gen_euler_evc.go"
//go:generate sh -c "abigen --abi ../../abi/euler_evault.json --pkg bindings --type EulerEVault --out gen_euler_evault.go"
//go:generate sh -c "abigen --abi ../../abi/euler_oracle.json --pkg bindings --type EulerOracle --out gen_euler_oracle.go"
//go:generate sh -c "abigen --abi ../../abi/erc20.json --pkg bindings --type ERC20 --out gen_erc20.go"

var (
	AavePoolABI         abi.ABI
	AaveDataProviderABI abi.ABI
	AaveOracleABI       abi.ABI
	EulerEVCABI         abi.ABI
	EulerEVaultABI      abi.ABI
	EulerOracleABI      abi.ABI
	ERC20ABI            abi.ABI
)

func init() {
	AavePoolABI = mustLoad("aave_pool.json")
	AaveDataProviderABI = mustLoad("aave_data_provider.json")
	AaveOracleABI = mustLoad("aave_oracle.json")
	EulerEVCABI = mustLoad("euler_evc.json")
	EulerEVaultABI = mustLoad("euler_evault.json")
	EulerOracleABI = mustLoad("euler_oracle.json")
	ERC20ABI = mustLoad("erc20.json")
}

func mustLoad(name string) abi.ABI {
	raw, err := abifs.FS.ReadFile(name)
	if err != nil {
		panic(fmt.Sprintf("bindings: read embedded abi %s: %v", name, err))
	}
	parsed, err := abi.JSON(strings.NewReader(string(raw)))
	if err != nil {
		panic(fmt.Sprintf("bindings: parse abi %s: %v", name, err))
	}
	return parsed
}

func Call(ctx context.Context, caller bind.ContractCaller, parsed abi.ABI, addr common.Address, block *big.Int, method string, args ...interface{}) ([]interface{}, error) {
	contract := bind.NewBoundContract(addr, parsed, caller, nil, nil)
	var out []interface{}
	opts := &bind.CallOpts{Context: ctx, BlockNumber: block}
	if err := contract.Call(opts, &out, method, args...); err != nil {
		return nil, fmt.Errorf("bindings: call %s at %s: %w", method, addr.Hex(), err)
	}
	return out, nil
}
