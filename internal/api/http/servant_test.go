package http

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"

	"github.com/dolmatovDan/lending-position-indexer/internal/models"
)

type fakeService struct {
	rows []models.Position
	err  error
}

func (f fakeService) Positions(ctx context.Context, wallet, protocol string) ([]models.Position, error) {
	return f.rows, f.err
}

type fakePinger struct{ err error }

func (f fakePinger) Ping(ctx context.Context) error { return f.err }

func newServant(svc PositionsService, db, rpc Pinger) *Servant {
	return NewServant(svc, db, rpc, slog.New(slog.NewTextHandler(io.Discard, nil)))
}

func TestPositions_OK(t *testing.T) {
	svc := fakeService{rows: []models.Position{{
		Protocol:      models.ProtocolAaveV3,
		WalletAddress: "0x0000000000000000000000000000000000000001",
		Side:          models.SideCollateral,
		Amount:        decimal.RequireFromString("1000"),
		Price:         decimal.RequireFromString("1"),
		HealthFactor:  decimal.RequireFromString("2"),
	}}}
	srv := newServant(svc, fakePinger{}, fakePinger{})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/positions?wallet=0x0000000000000000000000000000000000000001", nil)
	srv.Router().ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	var body struct {
		Positions []models.Position `json:"positions"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Len(t, body.Positions, 1)
}

func TestPositions_MissingWallet(t *testing.T) {
	srv := newServant(fakeService{}, fakePinger{}, fakePinger{})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/positions", nil)
	srv.Router().ServeHTTP(rec, req)
	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestPositions_InvalidWallet(t *testing.T) {
	srv := newServant(fakeService{err: models.ErrInvalidWallet}, fakePinger{}, fakePinger{})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/positions?wallet=bad", nil)
	srv.Router().ServeHTTP(rec, req)
	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestPositions_InvalidProtocol(t *testing.T) {
	srv := newServant(fakeService{err: models.ErrInvalidProtocol}, fakePinger{}, fakePinger{})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/positions?wallet=0x0000000000000000000000000000000000000001&protocol=x", nil)
	srv.Router().ServeHTTP(rec, req)
	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestPositions_InternalError(t *testing.T) {
	srv := newServant(fakeService{err: errors.New("boom")}, fakePinger{}, fakePinger{})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/positions?wallet=0x0000000000000000000000000000000000000001", nil)
	srv.Router().ServeHTTP(rec, req)
	require.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestHealthz_OK(t *testing.T) {
	srv := newServant(fakeService{}, fakePinger{}, fakePinger{})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	srv.Router().ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
}

func TestHealthz_Degraded(t *testing.T) {
	srv := newServant(fakeService{}, fakePinger{err: errors.New("db down")}, fakePinger{})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	srv.Router().ServeHTTP(rec, req)
	require.Equal(t, http.StatusServiceUnavailable, rec.Code)
}
