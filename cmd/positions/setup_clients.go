package main

import (
	"context"

	"github.com/dolmatovDan/lending-position-indexer/internal/clients/aave"
	"github.com/dolmatovDan/lending-position-indexer/internal/clients/ethnode"
	"github.com/dolmatovDan/lending-position-indexer/internal/clients/euler"
	"github.com/dolmatovDan/lending-position-indexer/internal/protocols"
)

type clients struct {
	node    ethnode.Client
	readers []protocols.Reader
}

func setupClients(ctx context.Context, cfg *Config) (*clients, error) {
	node, err := ethnode.New(ctx, cfg.RPCWSURL, cfg.RPCHTTPURL)
	if err != nil {
		return nil, err
	}

	caller := node.Caller()

	aaveCfg := aave.MainnetConfig()
	eulerCfg := euler.MainnetConfig()
	eulerCfg.SubaccountsDepth = cfg.EulerSubaccountsDepth

	readers := []protocols.Reader{
		aave.New(caller, aaveCfg),
		euler.New(caller, eulerCfg),
	}

	return &clients{node: node, readers: readers}, nil
}

type rpcPinger struct {
	node ethnode.Client
}

func (p rpcPinger) Ping(ctx context.Context) error {
	_, err := p.node.HeaderByNumber(ctx, nil)
	return err
}
