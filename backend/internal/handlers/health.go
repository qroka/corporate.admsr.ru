package handlers

import (
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"corporate.admsr.ru/backend/internal/httpx"
)

type Health struct {
	Pool *pgxpool.Pool
}

func (h *Health) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		httpx.MethodNotAllowed(w)
		return
	}

	payload := map[string]any{
		"ok":      true,
		"service": "corporate-portal",
		"time":    time.Now().UTC().Format(time.RFC3339),
		"backend": "go",
	}

	if h.Pool != nil {
		if err := h.Pool.Ping(r.Context()); err != nil {
			payload["ok"] = false
			payload["database"] = "error"
			httpx.WriteJSON(w, http.StatusServiceUnavailable, payload)
			return
		}
		payload["database"] = "ok"
	}

	httpx.WriteJSON(w, http.StatusOK, payload)
}
