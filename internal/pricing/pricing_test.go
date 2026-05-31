package pricing

import (
	"math/big"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

func TestScale(t *testing.T) {
	require.True(t, Scale(big.NewInt(12345678), 8).Equal(decimal.RequireFromString("0.12345678")))
	require.True(t, Scale(nil, 8).IsZero())
}

func TestTokenAmount(t *testing.T) {
	raw, _ := new(big.Int).SetString("1000000000000000000", 10)
	require.True(t, TokenAmount(raw, 18).Equal(decimal.RequireFromString("1")))

	usdc := big.NewInt(1000000000)
	require.True(t, TokenAmount(usdc, 6).Equal(decimal.RequireFromString("1000")))
}

func TestHealthFactor(t *testing.T) {
	raw, _ := new(big.Int).SetString("1500000000000000000", 10)
	require.True(t, HealthFactor(raw).Equal(decimal.RequireFromString("1.5")))
}

func TestRatio(t *testing.T) {
	r, ok := Ratio(big.NewInt(150), big.NewInt(100))
	require.True(t, ok)
	require.True(t, r.Equal(decimal.RequireFromString("1.5")))

	_, ok = Ratio(big.NewInt(1), big.NewInt(0))
	require.False(t, ok)
}
