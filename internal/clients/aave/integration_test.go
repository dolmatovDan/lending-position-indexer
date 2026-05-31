//go:build integration

package aave_test

import (
	"context"
	"math/big"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/dolmatovDan/lending-position-indexer/internal/clients/aave"
	"github.com/dolmatovDan/lending-position-indexer/internal/clients/ethnode"
)

func TestAaveIntegration(t *testing.T) {
	rpc := os.Getenv("RPC_HTTP_URL")
	wallet := os.Getenv("TEST_WALLET")
	if rpc == "" || wallet == "" {
		t.Skip("RPC_HTTP_URL and TEST_WALLET required for integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	node, err := ethnode.New(ctx, "", rpc)
	require.NoError(t, err)
	defer node.Close()

	head, err := node.HeaderByNumber(ctx, nil)
	require.NoError(t, err)

	adapter := aave.New(node.Caller(), aave.MainnetConfig())
	rows, err := adapter.Positions(ctx, []string{wallet}, new(big.Int).SetUint64(head.Number), head.Number, head.Time)
	require.NoError(t, err)
	t.Logf("collected %d aave leg rows at block %d", len(rows), head.Number)
}
