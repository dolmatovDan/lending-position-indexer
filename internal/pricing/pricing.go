package pricing

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/shopspring/decimal"

	"github.com/dolmatovDan/lending-position-indexer/internal/bindings"
)

func Scale(raw *big.Int, decimals uint8) decimal.Decimal {
	if raw == nil {
		return decimal.Zero
	}
	return decimal.NewFromBigInt(raw, 0).Shift(-int32(decimals))
}

func TokenAmount(raw *big.Int, tokenDecimals uint8) decimal.Decimal {
	return Scale(raw, tokenDecimals)
}

func HealthFactor(raw *big.Int) decimal.Decimal {
	return Scale(raw, 18)
}

func Ratio(numerator, denominator *big.Int) (decimal.Decimal, bool) {
	if denominator == nil || denominator.Sign() == 0 {
		return decimal.Zero, false
	}
	num := decimal.NewFromBigInt(numerator, 0)
	den := decimal.NewFromBigInt(denominator, 0)
	return num.Div(den), true
}

func ReadChainlinkPrice(ctx context.Context, caller bind.ContractCaller, aggregator common.Address, block *big.Int) (decimal.Decimal, error) {
	decOut, err := bindings.Call(ctx, caller, bindings.ChainlinkAggregatorABI, aggregator, block, "decimals")
	if err != nil {
		return decimal.Zero, err
	}
	dec, ok := decOut[0].(uint8)
	if !ok {
		return decimal.Zero, fmt.Errorf("pricing: unexpected decimals type %T", decOut[0])
	}

	roundOut, err := bindings.Call(ctx, caller, bindings.ChainlinkAggregatorABI, aggregator, block, "latestRoundData")
	if err != nil {
		return decimal.Zero, err
	}
	answer, ok := roundOut[1].(*big.Int)
	if !ok {
		return decimal.Zero, fmt.Errorf("pricing: unexpected answer type %T", roundOut[1])
	}
	if answer.Sign() <= 0 {
		return decimal.Zero, fmt.Errorf("pricing: non-positive chainlink answer")
	}
	return Scale(answer, dec), nil
}
