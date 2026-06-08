package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum/common"
)

type Config struct {
	RPCWSURL              string
	RPCHTTPURL            string
	Wallets               []string
	Confirmations         uint64
	EulerSubaccountsDepth uint64
	DatabaseURL           string
	LogLevel              string
	HTTPPort              string
}

func LoadConfig() (*Config, error) {
	cfg := &Config{
		RPCWSURL:    os.Getenv("RPC_WS_URL"),
		RPCHTTPURL:  os.Getenv("RPC_HTTP_URL"),
		DatabaseURL: os.Getenv("DATABASE_URL"),
		LogLevel:    getEnvDefault("LOG_LEVEL", "info"),
		HTTPPort:    getEnvDefault("HTTP_PORT", "8080"),
	}

	cfg.Wallets = parseWallets(os.Getenv("WALLETS"))

	var err error
	if cfg.Confirmations, err = parseUintDefault("CONFIRMATIONS", 0); err != nil {
		return nil, err
	}
	if cfg.EulerSubaccountsDepth, err = parseUintDefault("EULER_SUBACCOUNTS_DEPTH", 1); err != nil {
		return nil, err
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

func (c *Config) Validate() error {
	if c.RPCWSURL == "" && c.RPCHTTPURL == "" {
		return fmt.Errorf("config: at least one of RPC_WS_URL or RPC_HTTP_URL must be set")
	}
	if c.DatabaseURL == "" {
		return fmt.Errorf("config: DATABASE_URL is required")
	}
	if len(c.Wallets) == 0 {
		return fmt.Errorf("config: WALLETS must contain at least one address")
	}
	for _, w := range c.Wallets {
		if !common.IsHexAddress(w) {
			return fmt.Errorf("config: invalid wallet address %q", w)
		}
	}
	return nil
}

func parseWallets(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func getEnvDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func parseUintDefault(key string, def uint64) (uint64, error) {
	v := os.Getenv(key)
	if v == "" {
		return def, nil
	}
	parsed, err := strconv.ParseUint(v, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("config: %s must be a non-negative integer: %w", key, err)
	}
	return parsed, nil
}
