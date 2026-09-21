package handlers

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"corporate.admsr.ru/backend/internal/auth"
	"corporate.admsr.ru/backend/internal/httpx"
)

type News struct {
	Pool *pgxpool.Pool
	Auth *auth.Service
}

type newsRow struct {
	ID          int64
	Title       string
	Category    string
	Description string
	Date        *time.Time
	ImagePath   *string
	CreatedAt   *time.Time
	UpdatedAt   *time.Time
	Likes       int64
	Views       int64
}

func (h *News) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.get(w, r)
	case http.MethodPost:
		h.post(w, r)
	case http.MethodPut:
		h.put(w, r)
	case http.MethodDelete:
		h.del(w, r)
	default:
		httpx.MethodNotAllowed(w)
	}
}

func (h *News) get(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	if idStr := q.Get("id"); idStr != "" {
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil || id <= 0 {
			httpx.Fail(w, http.StatusBadRequest, "Некорректный id")
			return
		}
		row, err := h.fetchOne(r.Context(), id)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				httpx.Fail(w, http.StatusNotFound, "Новость не найдена")
				return
			}
			httpx.Fail(w, http.StatusInternalServerError, "Ошибка подключения к БД")
			return
		}
		httpx.OK(w, fmtNews(row), "OK")
		return
	}

	if q.Has("limit") || q.Has("cursor") {
		page, err := h.fetchCursorPage(r.Context(), q)
		if err != nil {
			if err.Error() == "bad_cursor" {
				httpx.Fail(w, http.StatusBadRequest, "Некорректный cursor")
				return
			}
			httpx.Fail(w, http.StatusInternalServerError, "Ошибка подключения к БД")
			return
		}
		httpx.OK(w, page, "OK")
		return
	}

	items, err := h.fetchList(r.Context(), q)
	if err != nil {
		httpx.Fail(w, http.StatusInternalServerError, "Ошибка подключения к БД")
		return
	}
	httpx.OK(w, items, "OK")
}

func (h *News) post(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	idStr := q.Get("id")
	action := strings.ToLower(strings.TrimSpace(q.Get("action")))

	if idStr != "" && action != "" {
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil || id <= 0 {
			httpx.Fail(w, http.StatusBadRequest, "Некорректный id")
			return
		}
		switch action {
		case "view":
			_, _ = h.Pool.Exec(r.Context(), `
				UPDATE public.news
				SET views = COALESCE(views, 0) + 1, updated_at = NOW()
				WHERE id = $1`, id)
			row, err := h.fetchOne(r.Context(), id)
			if err != nil {
				httpx.Fail(w, http.StatusNotFound, "Новость не найдена")
				return
			}
			httpx.OK(w, fmtNews(row), "Просмотр учтён")
			return
		case "like":
			var body struct {
				Liked *bool `json:"liked"`
			}
			if err := httpx.DecodeJSON(r, &body); err != nil || body.Liked == nil {
				httpx.Fail(w, http.StatusUnprocessableEntity, "Поле «liked» обязательно (true/false)")
				return
			}
			liked := 0
			if *body.Liked {
				liked = 1
			}
			_, err := h.Pool.Exec(r.Context(), `
				UPDATE public.news
				SET likes = CASE
					WHEN $2 = 1 THEN COALESCE(likes, 0) + 1
					ELSE GREATEST(COALESCE(likes, 0) - 1, 0)
				END,
				updated_at = NOW()
				WHERE id = $1`, id, liked)
			if err != nil {
				httpx.Fail(w, http.StatusInternalServerError, "Ошибка подключения к БД")
				return
			}
			row, err := h.fetchOne(r.Context(), id)
			if err != nil {
				httpx.Fail(w, http.StatusNotFound, "Новость не найдена")
				return
			}
			httpx.OK(w, fmtNews(row), "Лайк обновлён")
			return
		default:
			httpx.Fail(w, http.StatusBadRequest, "Неизвестное действие")
			return
		}
	}

	if err := h.requireNewsSection(w, r); err != nil {
		return
	}

	var d map[string]any
	if err := httpx.DecodeJSON(r, &d); err != nil {
		httpx.Fail(w, http.StatusBadRequest, "Некорректный JSON")
		return
	}
	if strings.TrimSpace(strVal(d["title"])) == "" {
		httpx.Fail(w, http.StatusUnprocessableEntity, "Поле `title` обязательно")
		return
	}
	if strings.TrimSpace(strVal(d["date"])) == "" {
		httpx.Fail(w, http.StatusUnprocessableEntity, "Поле `date` обязательно")
		return
	}

	category := strings.TrimSpace(strVal(d["category"]))
	description := strings.TrimSpace(strVal(d["description"]))
	var imagePath *string
	if ip := strings.TrimSpace(strVal(d["image_path"])); ip != "" {
		imagePath = &ip
	}

	var newID int64
	err := h.Pool.QueryRow(r.Context(), `
		INSERT INTO public.news (title, category, description, date, image_path)
		VALUES ($1, $2, $3, $4, $5) RETURNING id`,
		strings.TrimSpace(strVal(d["title"])),
		category,
		description,
		strings.TrimSpace(strVal(d["date"])),
		imagePath,
	).Scan(&newID)
	if err != nil {
		httpx.Fail(w, http.StatusInternalServerError, "Ошибка подключения к БД")
		return
	}
	row, err := h.fetchOne(r.Context(), newID)
	if err != nil {
		httpx.Fail(w, http.StatusInternalServerError, "Ошибка подключения к БД")
		return
	}
	httpx.Created(w, fmtNews(row), "Новость создана")
}

func (h *News) put(w http.ResponseWriter, r *http.Request) {
	if err := h.requireNewsSection(w, r); err != nil {
		return
	}
	id, ok := queryID(r)
	if !ok {
		httpx.Fail(w, http.StatusBadRequest, "Укажите ?id=...")
		return
	}
	var d map[string]any
	if err := httpx.DecodeJSON(r, &d); err != nil {
		httpx.Fail(w, http.StatusBadRequest, "Некорректный JSON")
		return
	}
	if strings.TrimSpace(strVal(d["title"])) == "" {
		httpx.Fail(w, http.StatusUnprocessableEntity, "Поле `title` обязательно")
		return
	}
	if strings.TrimSpace(strVal(d["date"])) == "" {
		httpx.Fail(w, http.StatusUnprocessableEntity, "Поле `date` обязательно")
		return
	}

	var exists int64
	err := h.Pool.QueryRow(r.Context(), `SELECT id FROM public.news WHERE id = $1`, id).Scan(&exists)
	if err != nil {
		httpx.Fail(w, http.StatusNotFound, "Новость не найдена")
		return
	}

	var imagePath *string
	if ip := strings.TrimSpace(strVal(d["image_path"])); ip != "" {
		imagePath = &ip
	}
	_, err = h.Pool.Exec(r.Context(), `
		UPDATE public.news
		SET title = $1, category = $2, description = $3, date = $4, image_path = $5, updated_at = NOW()
		WHERE id = $6`,
		strings.TrimSpace(strVal(d["title"])),
		strings.TrimSpace(strVal(d["category"])),
		strings.TrimSpace(strVal(d["description"])),
		strings.TrimSpace(strVal(d["date"])),
		imagePath,
		id,
	)
	if err != nil {
		httpx.Fail(w, http.StatusInternalServerError, "Ошибка подключения к БД")
		return
	}
	row, err := h.fetchOne(r.Context(), id)
	if err != nil {
		httpx.Fail(w, http.StatusInternalServerError, "Ошибка подключения к БД")
		return
	}
	httpx.OK(w, fmtNews(row), "Новость обновлена")
}

func (h *News) del(w http.ResponseWriter, r *http.Request) {
	if err := h.requireNewsSection(w, r); err != nil {
		return
	}
	id, ok := queryID(r)
	if !ok {
		httpx.Fail(w, http.StatusBadRequest, "Укажите ?id=...")
		return
	}
	var exists int64
	err := h.Pool.QueryRow(r.Context(), `SELECT id FROM public.news WHERE id = $1`, id).Scan(&exists)
	if err != nil {
		httpx.Fail(w, http.StatusNotFound, "Новость не найдена")
		return
	}
	_, _ = h.Pool.Exec(r.Context(), `DELETE FROM public.news WHERE id = $1`, id)
	httpx.OK(w, nil, "Новость удалена")
}

func (h *News) requireNewsSection(w http.ResponseWriter, r *http.Request) error {
	u, err := h.Auth.RequireUser(r.Context(), r)
	if err != nil {
		if errors.Is(err, auth.ErrUnauthorized) {
			httpx.Fail(w, http.StatusUnauthorized, "Требуется авторизация")
			return err
		}
		httpx.Fail(w, http.StatusInternalServerError, "Ошибка подключения к БД")
		return err
	}
	if !auth.CanEditSection(r.Context(), h.Pool, u, "news") {
		httpx.Fail(w, http.StatusForbidden, "Недостаточно прав для этого раздела")
		return auth.ErrForbidden
	}
	return nil
}

func (h *News) fetchOne(ctx context.Context, id int64) (newsRow, error) {
	var row newsRow
	var likes, views *int64
	err := h.Pool.QueryRow(ctx, `SELECT id, title, category, description, date, image_path, created_at, updated_at, likes, views FROM public.news WHERE id = $1`, id).
		Scan(&row.ID, &row.Title, &row.Category, &row.Description, &row.Date, &row.ImagePath, &row.CreatedAt, &row.UpdatedAt, &likes, &views)
	if err != nil {
		return row, err
	}
	if likes != nil {
		row.Likes = *likes
	}
	if views != nil {
		row.Views = *views
	}
	return row, nil
}

func (h *News) fetchList(ctx context.Context, q interface{ Get(string) string }) ([]map[string]any, error) {
	cond := []string{}
	args := []any{}
	argN := 1

	if s := strings.TrimSpace(q.Get("search")); s != "" {
		cond = append(cond, fmt.Sprintf("(title ILIKE $%d OR description ILIKE $%d OR category ILIKE $%d)", argN, argN, argN))
		args = append(args, "%"+s+"%")
		argN++
	}
	if c := strings.TrimSpace(q.Get("category")); c != "" {
		cond = append(cond, fmt.Sprintf("category ILIKE $%d", argN))
		args = append(args, "%"+c+"%")
		argN++
	}

	allowed := map[string]bool{"date": true, "created_at": true, "id": true, "title": true}
	order := q.Get("order")
	if !allowed[order] {
		order = "date"
	}
	dir := "DESC"
	if strings.EqualFold(q.Get("dir"), "asc") {
		dir = "ASC"
	}

	sql := "SELECT id, title, category, description, date, image_path, created_at, updated_at, likes, views FROM public.news"
	if len(cond) > 0 {
		sql += " WHERE " + strings.Join(cond, " AND ")
	}
	sql += fmt.Sprintf(" ORDER BY %s %s, id DESC", order, dir)

	rows, err := h.Pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanNewsRows(rows)
}

func (h *News) fetchCursorPage(ctx context.Context, q interface {
	Get(string) string
}) (map[string]any, error) {
	limit := 8
	if v := q.Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			limit = n
		}
	}
	if limit < 1 {
		limit = 8
	}
	if limit > 30 {
		limit = 30
	}

	cond := []string{}
	args := []any{}
	argN := 1

	if s := strings.TrimSpace(q.Get("search")); s != "" {
		cond = append(cond, fmt.Sprintf("(title ILIKE $%d OR description ILIKE $%d OR category ILIKE $%d)", argN, argN, argN))
		args = append(args, "%"+s+"%")
		argN++
	}
	if c := strings.TrimSpace(q.Get("category")); c != "" {
		cond = append(cond, fmt.Sprintf("category ILIKE $%d", argN))
		args = append(args, "%"+c+"%")
		argN++
	}

	if cursorRaw := strings.TrimSpace(q.Get("cursor")); cursorRaw != "" {
		cur, ok := decodeNewsCursor(cursorRaw)
		if !ok {
			return nil, errors.New("bad_cursor")
		}
		cond = append(cond, fmt.Sprintf("(date < $%d OR (date = $%d AND id < $%d))", argN, argN, argN+1))
		args = append(args, cur.Date, cur.ID)
		argN += 2
	}

	fetchLimit := limit + 1
	sql := "SELECT id, title, category, description, date, image_path, created_at, updated_at, likes, views FROM public.news"
	if len(cond) > 0 {
		sql += " WHERE " + strings.Join(cond, " AND ")
	}
	sql += fmt.Sprintf(" ORDER BY date DESC, id DESC LIMIT $%d", argN)
	args = append(args, fetchLimit)

	rows, err := h.Pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items, err := scanNewsRows(rows)
	if err != nil {
		return nil, err
	}

	hasMore := len(items) > limit
	if hasMore {
		items = items[:limit]
	}
	var nextCursor any
	if hasMore && len(items) > 0 {
		last := items[len(items)-1]
		dateStr := ""
		if d, ok := last["date"].(string); ok {
			dateStr = d
		}
		id, _ := last["id"].(int64)
		if id == 0 {
			if f, ok := last["id"].(float64); ok {
				id = int64(f)
			}
		}
		nextCursor = encodeNewsCursor(dateStr, id)
	} else {
		nextCursor = nil
	}

	return map[string]any{
		"items":      items,
		"nextCursor": nextCursor,
		"hasMore":    hasMore,
	}, nil
}

type newsScanner interface {
	Next() bool
	Scan(dest ...any) error
	Err() error
}

func scanNewsRows(rows newsScanner) ([]map[string]any, error) {
	var out []map[string]any
	for rows.Next() {
		var row newsRow
		var likes, views *int64
		if err := rows.Scan(&row.ID, &row.Title, &row.Category, &row.Description, &row.Date, &row.ImagePath, &row.CreatedAt, &row.UpdatedAt, &likes, &views); err != nil {
			return nil, err
		}
		if likes != nil {
			row.Likes = *likes
		}
		if views != nil {
			row.Views = *views
		}
		out = append(out, fmtNews(row))
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if out == nil {
		out = []map[string]any{}
	}
	return out, nil
}

func fmtNews(r newsRow) map[string]any {
	return map[string]any{
		"id":          r.ID,
		"title":       nullStrDef(r.Title),
		"category":    nullStrDef(r.Category),
		"description": nullStrDef(r.Description),
		"date":        formatDate(r.Date),
		"image_path":  r.ImagePath,
		"created_at":  formatTime(r.CreatedAt),
		"updated_at":  formatTime(r.UpdatedAt),
		"likes":       r.Likes,
		"views":       r.Views,
	}
}

func nullStrDef(s string) string { return s }

func formatDate(t *time.Time) any {
	if t == nil {
		return nil
	}
	return t.Format("2006-01-02")
}

func formatTime(t *time.Time) any {
	if t == nil {
		return nil
	}
	return t.UTC().Format(time.RFC3339Nano)
}

func queryID(r *http.Request) (int64, bool) {
	idStr := r.URL.Query().Get("id")
	if idStr == "" {
		return 0, false
	}
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		return 0, false
	}
	return id, true
}

func strVal(v any) string {
	if v == nil {
		return ""
	}
	switch t := v.(type) {
	case string:
		return t
	default:
		return fmt.Sprint(t)
	}
}

type newsCursor struct {
	Date string
	ID   int64
}

func encodeNewsCursor(date string, id int64) string {
	payload, _ := json.Marshal(map[string]any{"d": date, "i": id})
	s := base64.RawURLEncoding.EncodeToString(payload)
	return s
}

func decodeNewsCursor(raw string) (newsCursor, bool) {
	b, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		// try padded std with url alphabet
		padded := raw
		switch len(padded) % 4 {
		case 2:
			padded += "=="
		case 3:
			padded += "="
		}
		b, err = base64.URLEncoding.DecodeString(padded)
		if err != nil {
			return newsCursor{}, false
		}
	}
	var data struct {
		D string `json:"d"`
		I int64  `json:"i"`
	}
	if err := json.Unmarshal(b, &data); err != nil {
		return newsCursor{}, false
	}
	date := strings.TrimSpace(data.D)
	if len(date) != 10 || data.I < 1 {
		return newsCursor{}, false
	}
	return newsCursor{Date: date, ID: data.I}, true
}
