package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLoadConfig_Defaults(t *testing.T) {
	t.Setenv("RPC_HTTP_URL", "http://localhost:8545")
	t.Setenv("DATABASE_URL", "postgres://localhost/db")
	t.Setenv("WALLETS", "0x0000000000000000000000000000000000000001")

	cfg, err := LoadConfig()
	require.NoError(t, err)
	require.Equal(t, uint64(0), cfg.Confirmations)
	require.Equal(t, uint64(1), cfg.EulerSubaccountsDepth)
	require.Equal(t, "info", cfg.LogLevel)
	require.Equal(t, "8080", cfg.HTTPPort)
	require.Len(t, cfg.Wallets, 1)
}

func TestLoadConfig_MissingRPC(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://localhost/db")
	t.Setenv("WALLETS", "0x0000000000000000000000000000000000000001")

	_, err := LoadConfig()
	require.Error(t, err)
}

func TestLoadConfig_MissingDatabase(t *testing.T) {
	t.Setenv("RPC_HTTP_URL", "http://localhost:8545")
	t.Setenv("WALLETS", "0x0000000000000000000000000000000000000001")

	_, err := LoadConfig()
	require.Error(t, err)
}

func TestLoadConfig_NoWallets(t *testing.T) {
	t.Setenv("RPC_HTTP_URL", "http://localhost:8545")
	t.Setenv("DATABASE_URL", "postgres://localhost/db")

	_, err := LoadConfig()
	require.Error(t, err)
}

func TestLoadConfig_InvalidWallet(t *testing.T) {
	t.Setenv("RPC_HTTP_URL", "http://localhost:8545")
	t.Setenv("DATABASE_URL", "postgres://localhost/db")
	t.Setenv("WALLETS", "not-an-address")

	_, err := LoadConfig()
	require.Error(t, err)
}

func TestLoadConfig_MultipleWalletsAndOverrides(t *testing.T) {
	t.Setenv("RPC_WS_URL", "ws://localhost:8546")
	t.Setenv("DATABASE_URL", "postgres://localhost/db")
	t.Setenv("WALLETS", "0x0000000000000000000000000000000000000001, 0x0000000000000000000000000000000000000002")
	t.Setenv("CONFIRMATIONS", "3")
	t.Setenv("EULER_SUBACCOUNTS_DEPTH", "5")
	t.Setenv("LOG_LEVEL", "debug")
	t.Setenv("HTTP_PORT", "9000")

	cfg, err := LoadConfig()
	require.NoError(t, err)
	require.Len(t, cfg.Wallets, 2)
	require.Equal(t, uint64(3), cfg.Confirmations)
	require.Equal(t, uint64(5), cfg.EulerSubaccountsDepth)
	require.Equal(t, "debug", cfg.LogLevel)
	require.Equal(t, "9000", cfg.HTTPPort)
}
