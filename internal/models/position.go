package models

import (
	"time"

	"github.com/shopspring/decimal"
)

type ProtocolKind string

const (
	ProtocolAaveV3  ProtocolKind = "aave-v3"
	ProtocolEulerV2 ProtocolKind = "euler-v2"
)

type Side string

const (
	SideCollateral Side = "collateral"
	SideDebt       Side = "debt"
)

type Token struct {
	Address  string `json:"address"`
	Symbol   string `json:"symbol"`
	Decimals uint8  `json:"decimals"`
}

type Position struct {
	Protocol      ProtocolKind    `json:"protocol"`
	WalletAddress string          `json:"wallet_address"`
	MarketID      string          `json:"market_id"`
	Token         Token           `json:"token"`
	Side          Side            `json:"side"`
	Amount        decimal.Decimal `json:"amount"`
	Price         decimal.Decimal `json:"price"`
	HealthFactor  decimal.Decimal `json:"health_factor"`
	BlockNumber   uint64          `json:"block_number"`
	Timestamp     time.Time       `json:"timestamp"`
}
