package ethnode

import (
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
)

type Stub struct {
	Heads        []Head
	HeaderByNum  func(ctx context.Context, number *big.Int) (Head, error)
	CallerImpl   bind.ContractCaller
	HasSubs      bool
}

func (s *Stub) SupportsSubscriptions() bool { return s.HasSubs }

func (s *Stub) SubscribeNewHeads(ctx context.Context, out chan<- Head) (ethereum.Subscription, error) {
	sub := &stubSub{quit: make(chan struct{}), errs: make(chan error, 1)}
	go func() {
		for _, h := range s.Heads {
			select {
			case out <- h:
			case <-ctx.Done():
				return
			case <-sub.quit:
				return
			}
		}
	}()
	return sub, nil
}

func (s *Stub) HeaderByNumber(ctx context.Context, number *big.Int) (Head, error) {
	if s.HeaderByNum != nil {
		return s.HeaderByNum(ctx, number)
	}
	if len(s.Heads) > 0 {
		return s.Heads[len(s.Heads)-1], nil
	}
	return Head{}, nil
}

func (s *Stub) Caller() bind.ContractCaller { return s.CallerImpl }

func (s *Stub) Close() {}

type stubSub struct {
	quit chan struct{}
	errs chan error
}

func (s *stubSub) Unsubscribe()      { close(s.quit) }
func (s *stubSub) Err() <-chan error { return s.errs }
