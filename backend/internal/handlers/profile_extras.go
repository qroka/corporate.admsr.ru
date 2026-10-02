package handlers

import (
	"context"
	"errors"
	"log"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"corporate.admsr.ru/backend/internal/auth"
	"corporate.admsr.ru/backend/internal/httpx"
)

// ProfileExtras — «Желания» и «Награды» на странице профиля (/api/profile_extras.php,
// V15). Желания пишет и удаляет сам сотрудник (удалить может и администратор),
// награды выдаёт и удаляет только администратор. Читаются они вместе с профилем:
// profile.php?view=page.
type ProfileExtras struct {
	Pool *pgxpool.Pool
	Auth *auth.Service
}

const (
	wishMaxRunes       = 200
	wishMaxPerUser     = 20
	awardTitleMaxRunes = 120
	awardDescMaxRunes  = 500
)

func extrasTableMissing(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "42P01" // undefined_table
}

func extrasFailDB(w http.ResponseWriter, op string, err error) {
	log.Printf("profile extras: %s: %v", op, err)
	if extrasTableMissing(err) {
		httpx.Fail(w, http.StatusServiceUnavailable, "Желания и награды ещё не включены на сервере: нужна миграция V15__profile_extras.sql")
		return
	}
	httpx.Fail(w, http.StatusInternalServerError, "Ошибка подключения к БД")
}

func (h *ProfileExtras) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httpx.MethodNotAllowed(w)
		return
	}
	cur, ok := requireUser(w, r, h.Auth)
	if !ok {
		return
	}
	var body map[string]any
	if err := httpx.DecodeJSON(r, &body); err != nil {
		httpx.Fail(w, http.StatusBadRequest, "Некорректный JSON")
		return
	}
	switch strVal(body["action"]) {
	case "wish_add":
		h.wishAdd(w, r, cur, body)
	case "wish_delete":
		h.wishDelete(w, r, cur, body)
	case "award_add":
		h.awardAdd(w, r, cur, body)
	case "award_delete":
		h.awardDelete(w, r, cur, body)
	default:
		httpx.Fail(w, http.StatusBadRequest, "Неизвестное действие")
	}
}

// ── Желания ──────────────────────────────────────────────────────────────────

func loadWishes(ctx context.Context, pool *pgxpool.Pool, userID int64) []map[string]any {
	out := []map[string]any{}
	rows, err := pool.Query(ctx, `SELECT id, text FROM public.profile_wishes WHERE user_id = $1 ORDER BY id`, userID)
	if err != nil {
		log.Printf("profile: wishes for user %d: %v", userID, err)
		return out
	}
	defer rows.Close()
	for rows.Next() {
		var id int64
		var text string
		if rows.Scan(&id, &text) == nil {
			out = append(out, map[string]any{"id": id, "text": text})
		}
	}
	return out
}

func (h *ProfileExtras) wishAdd(w http.ResponseWriter, r *http.Request, cur *auth.User, body map[string]any) {
	text := strings.TrimSpace(strings.ReplaceAll(strVal(body["text"]), "\n", " "))
	if text == "" {
		httpx.Fail(w, http.StatusUnprocessableEntity, "Желание пустое")
		return
	}
	if utf8.RuneCountInString(text) > wishMaxRunes {
		httpx.Fail(w, http.StatusUnprocessableEntity, "Желание длиннее 200 символов")
		return
	}
	// Своё желание — личность только из сессии; чужое добавить нельзя.
	var n int64
	if err := h.Pool.QueryRow(r.Context(), `SELECT COUNT(*) FROM public.profile_wishes WHERE user_id = $1`, cur.ID).Scan(&n); err != nil {
		extrasFailDB(w, "wish count", err)
		return
	}
	if n >= wishMaxPerUser {
		httpx.Fail(w, http.StatusUnprocessableEntity, "Можно записать не больше 20 желаний — удалите исполненные")
		return
	}
	if _, err := h.Pool.Exec(r.Context(),
		`INSERT INTO public.profile_wishes (user_id, text) VALUES ($1, $2)`, cur.ID, text); err != nil {
		extrasFailDB(w, "wish add", err)
		return
	}
	httpx.OK(w, map[string]any{"wishes": loadWishes(r.Context(), h.Pool, cur.ID)}, "Желание добавлено")
}

func (h *ProfileExtras) wishDelete(w http.ResponseWriter, r *http.Request, cur *auth.User, body map[string]any) {
	id := int64Of(body["id"])
	if id <= 0 {
		httpx.Fail(w, http.StatusBadRequest, "Некорректный id")
		return
	}
	var owner int64
	if err := h.Pool.QueryRow(r.Context(), `SELECT user_id FROM public.profile_wishes WHERE id = $1`, id).Scan(&owner); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			httpx.Fail(w, http.StatusNotFound, "Желание не найдено")
			return
		}
		extrasFailDB(w, "wish fetch", err)
		return
	}
	if owner != cur.ID && !auth.IsAdmin(cur) {
		httpx.Fail(w, http.StatusForbidden, "Удалить желание может только его автор")
		return
	}
	if _, err := h.Pool.Exec(r.Context(), `DELETE FROM public.profile_wishes WHERE id = $1`, id); err != nil {
		extrasFailDB(w, "wish delete", err)
		return
	}
	httpx.OK(w, map[string]any{"wishes": loadWishes(r.Context(), h.Pool, owner)}, "Желание удалено")
}

// ── Награды ──────────────────────────────────────────────────────────────────

func loadAwards(ctx context.Context, pool *pgxpool.Pool, userID int64) []map[string]any {
	out := []map[string]any{}
	rows, err := pool.Query(ctx, `
		SELECT a.id, a.title, a.description, a.awarded_on,
			COALESCE(NULLIF(TRIM(CONCAT_WS(' ', u.surname, u.firstname)), ''), '')
		FROM public.profile_awards a
		LEFT JOIN public.user_info u ON u.id = a.issued_by
		WHERE a.user_id = $1
		ORDER BY a.awarded_on DESC, a.id DESC`, userID)
	if err != nil {
		log.Printf("profile: awards for user %d: %v", userID, err)
		return out
	}
	defer rows.Close()
	for rows.Next() {
		var id int64
		var title, desc, issuer string
		var on time.Time
		if rows.Scan(&id, &title, &desc, &on, &issuer) == nil {
			out = append(out, map[string]any{
				"id": id, "title": title, "description": desc,
				"awardedOn": on.Format("2006-01-02"), "issuedBy": issuer,
			})
		}
	}
	return out
}

func (h *ProfileExtras) awardAdd(w http.ResponseWriter, r *http.Request, cur *auth.User, body map[string]any) {
	if !auth.IsAdmin(cur) {
		httpx.Fail(w, http.StatusForbidden, "Награды выдаёт администратор портала")
		return
	}
	userID := int64Of(body["userId"])
	if userID <= 0 {
		httpx.Fail(w, http.StatusBadRequest, "Некорректный userId")
		return
	}
	title := strings.TrimSpace(strVal(body["title"]))
	if title == "" {
		httpx.Fail(w, http.StatusUnprocessableEntity, "Укажите название награды")
		return
	}
	if utf8.RuneCountInString(title) > awardTitleMaxRunes {
		httpx.Fail(w, http.StatusUnprocessableEntity, "Название длиннее 120 символов")
		return
	}
	desc := strings.TrimSpace(strVal(body["description"]))
	if utf8.RuneCountInString(desc) > awardDescMaxRunes {
		httpx.Fail(w, http.StatusUnprocessableEntity, "Описание длиннее 500 символов")
		return
	}
	on, err := time.Parse("2006-01-02", strings.TrimSpace(strVal(body["awardedOn"])))
	if err != nil {
		httpx.Fail(w, http.StatusUnprocessableEntity, "Укажите дату вручения")
		return
	}
	var exists bool
	if err := h.Pool.QueryRow(r.Context(),
		`SELECT EXISTS (SELECT 1 FROM public.user_info WHERE id = $1)`, userID).Scan(&exists); err != nil {
		extrasFailDB(w, "award owner check", err)
		return
	}
	if !exists {
		httpx.Fail(w, http.StatusNotFound, "Сотрудник не найден")
		return
	}
	if _, err := h.Pool.Exec(r.Context(), `
		INSERT INTO public.profile_awards (user_id, title, description, awarded_on, issued_by)
		VALUES ($1, $2, $3, $4, $5)`, userID, title, desc, on, cur.ID); err != nil {
		extrasFailDB(w, "award add", err)
		return
	}
	httpx.OK(w, map[string]any{"awards": loadAwards(r.Context(), h.Pool, userID)}, "Награда выдана")
}

func (h *ProfileExtras) awardDelete(w http.ResponseWriter, r *http.Request, cur *auth.User, body map[string]any) {
	if !auth.IsAdmin(cur) {
		httpx.Fail(w, http.StatusForbidden, "Награды удаляет администратор портала")
		return
	}
	id := int64Of(body["id"])
	if id <= 0 {
		httpx.Fail(w, http.StatusBadRequest, "Некорректный id")
		return
	}
	var owner int64
	if err := h.Pool.QueryRow(r.Context(),
		`DELETE FROM public.profile_awards WHERE id = $1 RETURNING user_id`, id).Scan(&owner); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			httpx.Fail(w, http.StatusNotFound, "Награда не найдена")
			return
		}
		extrasFailDB(w, "award delete", err)
		return
	}
	httpx.OK(w, map[string]any{"awards": loadAwards(r.Context(), h.Pool, owner)}, "Награда удалена")
}
