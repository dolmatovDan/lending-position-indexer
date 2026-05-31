package models

import "errors"

var (
	ErrInvalidWallet    = errors.New("invalid wallet address")
	ErrInvalidProtocol  = errors.New("unknown protocol")
	ErrNotFound         = errors.New("positions not found")
	ErrPriceUnavailable = errors.New("price source unavailable")
)
