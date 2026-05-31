package euler

import (
	"context"
	"log/slog"
	"math/big"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/shopspring/decimal"

	"github.com/dolmatovDan/lending-position-indexer/internal/models"
	"github.com/dolmatovDan/lending-position-indexer/internal/pricing"
)

const defaultUnitOfAccountDecimals = 18

type Config struct {
	EVCAddress       common.Address
	SubaccountsDepth uint64
}

func MainnetConfig() Config {
	return Config{
		EVCAddress:       common.HexToAddress("0x0C9a3dd6b8F28529d72d7f9cE918D493519EE383"),
		SubaccountsDepth: 1,
	}
}

type Adapter struct {
	contracts Contracts
	depth     uint64

	mu        sync.Mutex
	metaCache map[common.Address]TokenMeta
}

func New(caller bind.ContractCaller, cfg Config) *Adapter {
	return newAdapter(newEthContracts(caller, cfg), cfg.SubaccountsDepth)
}

func newAdapter(contracts Contracts, depth uint64) *Adapter {
	if depth == 0 {
		depth = 1
	}
	if depth > 256 {
		depth = 256
	}
	return &Adapter{
		contracts: contracts,
		depth:     depth,
		metaCache: make(map[common.Address]TokenMeta),
	}
}

func (a *Adapter) Name() models.ProtocolKind { return models.ProtocolEulerV2 }

func (a *Adapter) Positions(ctx context.Context, wallets []string, block *big.Int, blockNumber uint64, timestamp uint64) ([]models.Position, error) {
	ts := time.Unix(int64(timestamp), 0).UTC()
	var out []models.Position
	for _, w := range wallets {
		owner := common.HexToAddress(w)
		for _, account := range subAccounts(owner, a.depth) {
			rows, err := a.accountPositions(ctx, owner, account, block, blockNumber, ts)
			if err != nil {
				return nil, err
			}
			out = append(out, rows...)
		}
	}
	return out, nil
}

func subAccounts(owner common.Address, depth uint64) []common.Address {
	prefix := owner.Bytes()
	last := prefix[19]
	out := make([]common.Address, 0, depth)
	for i := uint64(0); i < depth; i++ {
		b := make([]byte, 20)
		copy(b, prefix[:19])
		b[19] = last ^ byte(i)
		out = append(out, common.BytesToAddress(b))
	}
	return out
}

func (a *Adapter) accountPositions(ctx context.Context, owner, account common.Address, block *big.Int, blockNumber uint64, ts time.Time) ([]models.Position, error) {
	controllers, err := a.contracts.Controllers(ctx, account, block)
	if err != nil {
		return nil, err
	}
	collaterals, err := a.contracts.Collaterals(ctx, account, block)
	if err != nil {
		return nil, err
	}

	var rows []models.Position

	if len(controllers) > 0 {
		controller := controllers[0]
		hf, err := a.healthFactor(ctx, controller, account, block)
		if err != nil {
			return nil, err
		}
		debtRow, err := a.debtLeg(ctx, owner, controller, account, controller, hf, block, blockNumber, ts)
		if err != nil {
			return nil, err
		}
		if debtRow != nil {
			rows = append(rows, *debtRow)
		}
		for _, vault := range collaterals {
			row, err := a.collateralLeg(ctx, owner, vault, account, controller, hf, block, blockNumber, ts)
			if err != nil {
				return nil, err
			}
			if row != nil {
				rows = append(rows, *row)
			}
		}
		return rows, nil
	}

	for _, vault := range collaterals {
		row, err := a.collateralLeg(ctx, owner, vault, account, vault, decimal.Zero, block, blockNumber, ts)
		if err != nil {
			return nil, err
		}
		if row != nil {
			rows = append(rows, *row)
		}
	}
	return rows, nil
}

func (a *Adapter) healthFactor(ctx context.Context, controller, account common.Address, block *big.Int) (decimal.Decimal, error) {
	liq, err := a.contracts.AccountLiquidity(ctx, controller, account, block)
	if err != nil {
		return decimal.Zero, err
	}
	hf, ok := pricing.Ratio(liq.CollateralValue, liq.LiabilityValue)
	if !ok {
		return decimal.Zero, nil
	}
	return hf, nil
}

func (a *Adapter) collateralLeg(ctx context.Context, owner, vault, account, scope common.Address, hf decimal.Decimal, block *big.Int, blockNumber uint64, ts time.Time) (*models.Position, error) {
	shares, err := a.contracts.BalanceOf(ctx, vault, account, block)
	if err != nil {
		return nil, err
	}
	if shares == nil || shares.Sign() == 0 {
		return nil, nil
	}
	assets, err := a.contracts.ConvertToAssets(ctx, vault, shares, block)
	if err != nil {
		return nil, err
	}
	return a.leg(ctx, owner, vault, scope, models.SideCollateral, assets, hf, block, blockNumber, ts)
}

func (a *Adapter) debtLeg(ctx context.Context, owner, vault, account, scope common.Address, hf decimal.Decimal, block *big.Int, blockNumber uint64, ts time.Time) (*models.Position, error) {
	debt, err := a.contracts.DebtOf(ctx, vault, account, block)
	if err != nil {
		return nil, err
	}
	if debt == nil || debt.Sign() == 0 {
		return nil, nil
	}
	return a.leg(ctx, owner, vault, scope, models.SideDebt, debt, hf, block, blockNumber, ts)
}

func (a *Adapter) leg(ctx context.Context, owner, vault, scope common.Address, side models.Side, rawAmount *big.Int, hf decimal.Decimal, block *big.Int, blockNumber uint64, ts time.Time) (*models.Position, error) {
	asset, err := a.contracts.VaultAsset(ctx, vault, block)
	if err != nil {
		return nil, err
	}
	meta, err := a.tokenMeta(ctx, asset, block)
	if err != nil {
		return nil, err
	}
	price, ok := a.price(ctx, vault, asset, meta.Decimals, block)
	if !ok {
		slog.Warn("euler: price unavailable, skipping leg", "vault", vault.Hex(), "asset", asset.Hex())
		return nil, nil
	}
	return &models.Position{
		Protocol:      models.ProtocolEulerV2,
		WalletAddress: owner.Hex(),
		MarketID:      scope.Hex(),
		Token:         models.Token{Address: asset.Hex(), Symbol: meta.Symbol, Decimals: meta.Decimals},
		Side:          side,
		Amount:        pricing.TokenAmount(rawAmount, meta.Decimals),
		Price:         price,
		HealthFactor:  hf,
		BlockNumber:   blockNumber,
		Timestamp:     ts,
	}, nil
}

func (a *Adapter) price(ctx context.Context, vault, asset common.Address, assetDecimals uint8, block *big.Int) (decimal.Decimal, bool) {
	oracle, err := a.contracts.VaultOracle(ctx, vault, block)
	if err != nil || oracle == (common.Address{}) {
		return decimal.Zero, false
	}
	uoa, err := a.contracts.VaultUnitOfAccount(ctx, vault, block)
	if err != nil {
		return decimal.Zero, false
	}
	uoaDecimals := uint8(defaultUnitOfAccountDecimals)
	if meta, err := a.tokenMeta(ctx, uoa, block); err == nil && meta.Decimals > 0 {
		uoaDecimals = meta.Decimals
	}
	one := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(assetDecimals)), nil)
	raw, err := a.contracts.Quote(ctx, oracle, one, asset, uoa, block)
	if err != nil || raw == nil || raw.Sign() <= 0 {
		return decimal.Zero, false
	}
	return pricing.Scale(raw, uoaDecimals), true
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
