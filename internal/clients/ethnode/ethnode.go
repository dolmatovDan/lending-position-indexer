package ethnode

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

type Head struct {
	Number uint64
	Time   uint64
	Hash   common.Hash
}

type Client interface {
	SupportsSubscriptions() bool
	SubscribeNewHeads(ctx context.Context, ch chan<- Head) (ethereum.Subscription, error)
	HeaderByNumber(ctx context.Context, number *big.Int) (Head, error)
	Caller() bind.ContractCaller
	Close()
}

type client struct {
	ws   *ethclient.Client
	http *ethclient.Client
}

func New(ctx context.Context, wsURL, httpURL string) (Client, error) {
	c := &client{}
	if wsURL != "" {
		ws, err := ethclient.DialContext(ctx, wsURL)
		if err != nil {
			return nil, fmt.Errorf("ethnode: dial ws %q: %w", wsURL, err)
		}
		c.ws = ws
	}
	if httpURL != "" {
		h, err := ethclient.DialContext(ctx, httpURL)
		if err != nil {
			return nil, fmt.Errorf("ethnode: dial http %q: %w", httpURL, err)
		}
		c.http = h
	}
	if c.ws == nil && c.http == nil {
		return nil, fmt.Errorf("ethnode: no rpc endpoint configured")
	}
	return c, nil
}

func (c *client) SupportsSubscriptions() bool {
	return c.ws != nil
}

func (c *client) SubscribeNewHeads(ctx context.Context, out chan<- Head) (ethereum.Subscription, error) {
	if c.ws == nil {
		return nil, fmt.Errorf("ethnode: websocket endpoint not configured")
	}
	headers := make(chan *types.Header)
	sub, err := c.ws.SubscribeNewHead(ctx, headers)
	if err != nil {
		return nil, fmt.Errorf("ethnode: subscribe new head: %w", err)
	}
	proxy := newRelay(sub)
	go proxy.pump(ctx, headers, out)
	return proxy, nil
}

func (c *client) HeaderByNumber(ctx context.Context, number *big.Int) (Head, error) {
	src := c.http
	if src == nil {
		src = c.ws
	}
	h, err := src.HeaderByNumber(ctx, number)
	if err != nil {
		return Head{}, fmt.Errorf("ethnode: header by number: %w", err)
	}
	return Head{Number: h.Number.Uint64(), Time: h.Time, Hash: h.Hash()}, nil
}

func (c *client) Caller() bind.ContractCaller {
	if c.http != nil {
		return c.http
	}
	return c.ws
}

func (c *client) Close() {
	if c.ws != nil {
		c.ws.Close()
	}
	if c.http != nil {
		c.http.Close()
	}
}

type relay struct {
	inner ethereum.Subscription
	errs  chan error
}

func newRelay(inner ethereum.Subscription) *relay {
	return &relay{inner: inner, errs: make(chan error, 1)}
}

func (r *relay) pump(ctx context.Context, headers <-chan *types.Header, out chan<- Head) {
	for {
		select {
		case <-ctx.Done():
			return
		case err := <-r.inner.Err():
			if err != nil {
				select {
				case r.errs <- err:
				default:
				}
			}
			return
		case h := <-headers:
			select {
			case out <- Head{Number: h.Number.Uint64(), Time: h.Time, Hash: h.Hash()}:
			case <-ctx.Done():
				return
			}
		}
	}
}

func (r *relay) Unsubscribe() {
	r.inner.Unsubscribe()
}

func (r *relay) Err() <-chan error {
	return r.errs
}
