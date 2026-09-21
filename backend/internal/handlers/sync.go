package handlers

import (
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"

	"corporate.admsr.ru/backend/internal/httpx"
)

type Sync struct {
	Pool *pgxpool.Pool
}

func (h *Sync) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httpx.WriteJSON(w, http.StatusMethodNotAllowed, map[string]any{
			"success": false, "message": "Method not allowed",
		})
		return
	}
	var body struct {
		Action string `json:"action"`
		User   *struct {
			ID       any    `json:"id"`
			Name     string `json:"name"`
			Login    string `json:"login"`
			Password string `json:"password"`
			Email    string `json:"email"`
			Phone    string `json:"phone"`
		} `json:"user"`
	}
	if err := httpx.DecodeJSON(r, &body); err != nil || body.Action != "create" || body.User == nil {
		httpx.WriteJSON(w, http.StatusBadRequest, map[string]any{
			"success": false, "message": "Invalid request",
		})
		return
	}
	u := body.User
	parts := strings.SplitN(strings.TrimSpace(u.Name), " ", 3)
	surname, firstname, lastname := "", "", ""
	if len(parts) > 0 {
		surname = parts[0]
	}
	if len(parts) > 1 {
		firstname = parts[1]
	}
	if len(parts) > 2 {
		lastname = parts[2]
	}
	id := asInt64(u.ID)
	_, err := h.Pool.Exec(r.Context(), `
		INSERT INTO public.user_info
			(id, status, login, password, firstname, surname, lastname, phone, email, ofo, user_group, role, auth)
		VALUES ($1, true, $2, $3, $4, $5, $6, $7, $8, -1, 'user', '', false)
		ON CONFLICT (id) DO NOTHING`,
		id, u.Login, u.Password, firstname, surname, lastname, u.Phone, u.Email,
	)
	if err != nil {
		httpx.WriteJSON(w, http.StatusInternalServerError, map[string]any{
			"success": false, "message": "DB connection error",
		})
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"success": true})
}
