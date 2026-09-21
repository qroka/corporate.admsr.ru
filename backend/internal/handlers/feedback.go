package handlers

import (
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/jackc/pgx/v5/pgxpool"

	"corporate.admsr.ru/backend/internal/auth"
	"corporate.admsr.ru/backend/internal/httpx"
)

type Feedback struct {
	Pool *pgxpool.Pool
	Auth *auth.Service
}

var feedbackCategories = map[string]struct{}{
	"question": {},
	"bug":      {},
	"idea":     {},
	"other":    {},
}

func (h *Feedback) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httpx.Fail(w, http.StatusMethodNotAllowed, "Метод не поддерживается")
		return
	}

	u, ok := requireUser(w, r, h.Auth)
	if !ok {
		return
	}

	var body map[string]any
	if err := httpx.DecodeJSON(r, &body); err != nil {
		httpx.Fail(w, http.StatusBadRequest, "Некорректный JSON")
		return
	}

	category := strings.TrimSpace(strVal(body["category"]))
	if category == "" {
		category = "other"
	}
	if _, allowed := feedbackCategories[category]; !allowed {
		httpx.Fail(w, http.StatusBadRequest, "Некорректный тип обращения")
		return
	}

	subject := strings.TrimSpace(strVal(body["subject"]))
	message := strings.TrimSpace(strVal(body["message"]))

	if subject == "" {
		httpx.Fail(w, http.StatusBadRequest, "Укажите тему")
		return
	}
	if utf8.RuneCountInString(subject) > 200 {
		httpx.Fail(w, http.StatusBadRequest, "Тема слишком длинная")
		return
	}
	if utf8.RuneCountInString(message) < 10 {
		httpx.Fail(w, http.StatusBadRequest, "Сообщение слишком короткое")
		return
	}
	if utf8.RuneCountInString(message) > 5000 {
		httpx.Fail(w, http.StatusBadRequest, "Сообщение слишком длинное")
		return
	}

	fio := strings.TrimSpace(strings.Join(filterNonEmpty([]string{u.Surname, u.Firstname, u.Lastname}), " "))
	var fioPtr *string
	if fio != "" {
		fioPtr = &fio
	}

	ctx := r.Context()
	_, err := h.Pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS public.portal_feedback (
			id BIGSERIAL PRIMARY KEY,
			user_id INTEGER NOT NULL,
			fio TEXT,
			category TEXT NOT NULL,
			subject TEXT NOT NULL,
			message TEXT NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`)
	if err != nil {
		httpx.Fail(w, http.StatusInternalServerError, "Не удалось сохранить обращение")
		return
	}

	var (
		newID     int64
		createdAt any
	)
	err = h.Pool.QueryRow(ctx, `
		INSERT INTO public.portal_feedback (user_id, fio, category, subject, message)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at`,
		u.ID, fioPtr, category, subject, message,
	).Scan(&newID, &createdAt)
	if err != nil {
		httpx.Fail(w, http.StatusInternalServerError, "Не удалось сохранить обращение")
		return
	}

	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"success": true,
		"data": map[string]any{
			"id":         newID,
			"created_at": createdAt,
		},
		"message": nil,
	})
}

func filterNonEmpty(parts []string) []string {
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if s := strings.TrimSpace(p); s != "" {
			out = append(out, s)
		}
	}
	return out
}
