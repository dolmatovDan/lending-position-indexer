package positions

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"math/big"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/dolmatovDan/lending-position-indexer/internal/models"
	"github.com/dolmatovDan/lending-position-indexer/internal/protocols"
)

type fakeRepo struct {
	mu    sync.Mutex
	saved []models.Position
}

func (f *fakeRepo) Save(ctx context.Context, p []models.Position) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.saved = append(f.saved, p...)
	return nil
}
func (f *fakeRepo) ByWallet(ctx context.Context, wallet, protocol string) ([]models.Position, error) {
	return f.saved, nil
}
func (f *fakeRepo) Ping(ctx context.Context) error { return nil }
func (f *fakeRepo) Close()                          {}

type fakeReader struct {
	name models.ProtocolKind
	rows []models.Position
	err  error
}

func (r fakeReader) Name() models.ProtocolKind { return r.name }
func (r fakeReader) Positions(ctx context.Context, wallets []string, block *big.Int, blockNumber, timestamp uint64) ([]models.Position, error) {
	return r.rows, r.err
}

func logger() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }

func TestUpdateForBlock_OneProtocolFails(t *testing.T) {
	repo := &fakeRepo{}
	ok := fakeReader{name: models.ProtocolAaveV3, rows: []models.Position{{Protocol: models.ProtocolAaveV3}}}
	bad := fakeReader{name: models.ProtocolEulerV2, err: errors.New("rpc down")}

	m := New(repo, []protocols.Reader{ok, bad}, []string{"0x1"}, logger())
	err := m.UpdateForBlock(context.Background(), big.NewInt(100), 100, 1700000000)
	require.NoError(t, err)
	require.Len(t, repo.saved, 1)
}

func TestPositions_InvalidWallet(t *testing.T) {
	m := New(&fakeRepo{}, nil, nil, logger())
	_, err := m.Positions(context.Background(), "bad", "")
	require.ErrorIs(t, err, models.ErrInvalidWallet)
}

func TestPositions_InvalidProtocol(t *testing.T) {
	m := New(&fakeRepo{}, nil, nil, logger())
	_, err := m.Positions(context.Background(), "0x0000000000000000000000000000000000000001", "nope")
	require.ErrorIs(t, err, models.ErrInvalidProtocol)
}
