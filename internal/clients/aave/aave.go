package aave

import (
	"context"
	"math/big"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/shopspring/decimal"

	"github.com/dolmatovDan/lending-position-indexer/internal/models"
	"github.com/dolmatovDan/lending-position-indexer/internal/pricing"
)

const priceDecimals = 8

type Config struct {
	PoolAddress         common.Address
	DataProviderAddress common.Address
	OracleAddress       common.Address
}

func MainnetConfig() Config {
	return Config{
		PoolAddress:         common.HexToAddress("0x87870Bca3F3fD6335C3F4ce8392D69350B4fA4E2"),
		DataProviderAddress: common.HexToAddress("0x0a16f2FCC0D44FaE41cc54e079281D84A363bECD"),
		OracleAddress:       common.HexToAddress("0x54586bE62E3c3580375aE3723C145253060Ca0C2"),
	}
}

type Adapter struct {
	contracts Contracts
	pool      common.Address

	mu        sync.Mutex
	metaCache map[common.Address]TokenMeta
}

func New(caller bind.ContractCaller, cfg Config) *Adapter {
	return newAdapter(newEthContracts(caller, cfg), cfg.PoolAddress)
}

func newAdapter(contracts Contracts, pool common.Address) *Adapter {
	return &Adapter{
		contracts: contracts,
		pool:      pool,
		metaCache: make(map[common.Address]TokenMeta),
	}
}

func (a *Adapter) Name() models.ProtocolKind { return models.ProtocolAaveV3 }

func (a *Adapter) Positions(ctx context.Context, wallets []string, block *big.Int, blockNumber uint64, timestamp uint64) ([]models.Position, error) {
	reserves, err := a.contracts.ReservesList(ctx, block)
	if err != nil {
		return nil, err
	}
	ts := time.Unix(int64(timestamp), 0).UTC()

	var out []models.Position
	for _, w := range wallets {
		wallet := common.HexToAddress(w)
		rows, err := a.walletPositions(ctx, wallet, reserves, block, blockNumber, ts)
		if err != nil {
			return nil, err
		}
		out = append(out, rows...)
	}
	return out, nil
}

func (a *Adapter) walletPositions(ctx context.Context, wallet common.Address, reserves []common.Address, block *big.Int, blockNumber uint64, ts time.Time) ([]models.Position, error) {
	acct, err := a.contracts.UserAccountData(ctx, wallet, block)
	if err != nil {
		return nil, err
	}
	hf := accountHealthFactor(acct)

	var rows []models.Position
	for _, asset := range reserves {
		rd, err := a.contracts.UserReserveData(ctx, asset, wallet, block)
		if err != nil {
			return nil, err
		}
		hasCollateral := rd.ATokenBalance != nil && rd.ATokenBalance.Sign() > 0
		debt := new(big.Int)
		if rd.StableDebt != nil {
			debt.Add(debt, rd.StableDebt)
		}
		if rd.VariableDebt != nil {
			debt.Add(debt, rd.VariableDebt)
		}
		hasDebt := debt.Sign() > 0
		if !hasCollateral && !hasDebt {
			continue
		}

		meta, err := a.tokenMeta(ctx, asset, block)
		if err != nil {
			return nil, err
		}
		priceRaw, err := a.contracts.AssetPrice(ctx, asset, block)
		if err != nil {
			return nil, err
		}
		price := pricing.Scale(priceRaw, priceDecimals)
		token := models.Token{Address: asset.Hex(), Symbol: meta.Symbol, Decimals: meta.Decimals}

		if hasCollateral {
			rows = append(rows, models.Position{
				Protocol:      models.ProtocolAaveV3,
				WalletAddress: wallet.Hex(),
				MarketID:      a.pool.Hex(),
				Token:         token,
				Side:          models.SideCollateral,
				Amount:        pricing.TokenAmount(rd.ATokenBalance, meta.Decimals),
				Price:         price,
				HealthFactor:  hf,
				BlockNumber:   blockNumber,
				Timestamp:     ts,
			})
		}
		if hasDebt {
			rows = append(rows, models.Position{
				Protocol:      models.ProtocolAaveV3,
				WalletAddress: wallet.Hex(),
				MarketID:      a.pool.Hex(),
				Token:         token,
				Side:          models.SideDebt,
				Amount:        pricing.TokenAmount(debt, meta.Decimals),
				Price:         price,
				HealthFactor:  hf,
				BlockNumber:   blockNumber,
				Timestamp:     ts,
			})
		}
	}
	return rows, nil
}

func accountHealthFactor(acct AccountData) decimal.Decimal {
	if acct.TotalDebtBase == nil || acct.TotalDebtBase.Sign() == 0 {
		return decimal.Zero
	}
	return pricing.HealthFactor(acct.HealthFactor)
}

func (a *Adapter) tokenMeta(ctx context.Context, token common.Address, block *big.Int) (TokenMeta, error) {
	a.mu.Lock()
	if m, ok := a.metaCache[token]; ok {
		a.mu.Unlock()
		return m, nil
	}
	a.mu.Unlock()

	m, err := a.contracts.TokenMeta(ctx, token, block)
	if err != nil {
		return TokenMeta{}, err
	}
	a.mu.Lock()
	a.metaCache[token] = m
	a.mu.Unlock()
	return m, nil
}
