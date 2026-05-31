package http

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/dolmatovDan/lending-position-indexer/internal/models"
)

type PositionsService interface {
	Positions(ctx context.Context, wallet, protocol string) ([]models.Position, error)
}

type Pinger interface {
	Ping(ctx context.Context) error
}

type Servant struct {
	positions PositionsService
	db        Pinger
	rpc       Pinger
	logger    *slog.Logger
}

func NewServant(positions PositionsService, db, rpc Pinger, logger *slog.Logger) *Servant {
	return &Servant{positions: positions, db: db, rpc: rpc, logger: logger}
}

func (s *Servant) Router() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.Recoverer)
	r.Get("/healthz", s.handleHealthz)
	r.Get("/positions", s.handlePositions)
	return r
}

func (s *Servant) handlePositions(w http.ResponseWriter, r *http.Request) {
	wallet := r.URL.Query().Get("wallet")
	protocol := r.URL.Query().Get("protocol")
	if wallet == "" {
		writeError(w, http.StatusBadRequest, "wallet query parameter is required")
		return
	}

	rows, err := s.positions.Positions(r.Context(), wallet, protocol)
	if err != nil {
		s.mapError(w, err)
		return
	}
	if rows == nil {
		rows = []models.Position{}
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"wallet":    wallet,
		"protocol":  protocol,
		"positions": rows,
	})
}

func (s *Servant) handleHealthz(w http.ResponseWriter, r *http.Request) {
	dbOK := s.db == nil || s.db.Ping(r.Context()) == nil
	rpcOK := s.rpc == nil || s.rpc.Ping(r.Context()) == nil

	status := http.StatusOK
	state := "ok"
	if !dbOK || !rpcOK {
		status = http.StatusServiceUnavailable
		state = "degraded"
	}
	writeJSON(w, status, map[string]interface{}{
		"status": state,
		"db":     dbOK,
		"rpc":    rpcOK,
	})
}

func (s *Servant) mapError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, models.ErrInvalidWallet):
		writeError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, models.ErrInvalidProtocol):
		writeError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, models.ErrNotFound):
		writeError(w, http.StatusNotFound, err.Error())
	default:
		s.logger.Error("positions request failed", "err", err)
		writeError(w, http.StatusInternalServerError, "internal error")
	}
}

func writeJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
