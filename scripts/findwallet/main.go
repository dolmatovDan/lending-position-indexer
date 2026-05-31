package main

import (
	"context"
	"fmt"
	"math/big"
	"os"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"

	"github.com/dolmatovDan/lending-position-indexer/internal/bindings"
	"github.com/dolmatovDan/lending-position-indexer/internal/clients/aave"
	"github.com/dolmatovDan/lending-position-indexer/internal/pricing"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run() error {
	key, err := os.ReadFile(os.Getenv("HOME") + "/.alchemy")
	if err != nil {
		return fmt.Errorf("read ~/.alchemy: %w", err)
	}
	url := "https://eth-mainnet.g.alchemy.com/v2/" + strings.TrimSpace(string(key))

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	cl, err := ethclient.DialContext(ctx, url)
	if err != nil {
		return err
	}
	defer cl.Close()

	head, err := cl.BlockNumber(ctx)
	if err != nil {
		return err
	}

	cfg := aave.MainnetConfig()
	borrowTopic := crypto.Keccak256Hash([]byte("Borrow(address,address,address,uint256,uint8,uint256,uint16)"))
	block := big.NewInt(0).SetUint64(head)

	const (
		window     = 10
		maxWindows = 400
		wantFound  = 10
	)

	seen := map[common.Address]bool{}
	found := 0
	fmt.Println("scanning recent Aave v3 Borrow events (10-block windows)...")
	fmt.Println("\nwallets with live Aave v3 debt:")

	for i := uint64(0); i < maxWindows && found < wantFound; i++ {
		hi := head - i*window
		if hi < window {
			break
		}
		lo := hi - (window - 1)

		logs, err := cl.FilterLogs(ctx, ethereum.FilterQuery{
			FromBlock: big.NewInt(0).SetUint64(lo),
			ToBlock:   big.NewInt(0).SetUint64(hi),
			Addresses: []common.Address{cfg.PoolAddress},
			Topics:    [][]common.Hash{{borrowTopic}},
		})
		if err != nil {
			return fmt.Errorf("getLogs: %w", err)
		}

		for _, lg := range logs {
			if len(lg.Topics) < 3 {
				continue
			}
			acc := common.HexToAddress(lg.Topics[2].Hex())
			if seen[acc] {
				continue
			}
			seen[acc] = true

			out, err := bindings.Call(ctx, cl, bindings.AavePoolABI, cfg.PoolAddress, block, "getUserAccountData", acc)
			if err != nil {
				continue
			}
			totalDebt := out[1].(*big.Int)
			if totalDebt.Sign() == 0 {
				continue
			}
			hf := pricing.HealthFactor(out[5].(*big.Int))
			fmt.Printf("  %s  debtBase=%s  HF=%s\n", acc.Hex(), totalDebt.String(), hf.StringFixed(3))
			found++
			if found >= wantFound {
				break
			}
		}
	}

	if found == 0 {
		fmt.Println("  (none found — increase maxWindows)")
	}
	return nil
}
