package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"

	"corporate.admsr.ru/backend/internal/auth"
	"corporate.admsr.ru/backend/internal/httpx"
)

// Services — CRUD списка сервисов портала (/api/portal_services.php).
func (h *Portal) Services(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.servicesGet(w, r)
	case http.MethodPost:
		h.servicesPost(w, r)
	case http.MethodPut:
		h.servicesPut(w, r)
	case http.MethodDelete:
		h.servicesDelete(w, r)
	default:
		httpx.Fail(w, http.StatusMethodNotAllowed, "Метод не поддерживается")
	}
}

func (h *Portal) servicesGet(w http.ResponseWriter, r *http.Request) {
	u, ok := requireUser(w, r, h.Auth)
	if !ok {
		return
	}
	ctx := r.Context()
	admin := auth.IsAdmin(u)
	q := `
		SELECT id, kind, label, description, icon, internal_key, path, external_url,
		       sort_order, is_enabled, created_at, updated_at
		FROM public.portal_services`
	if !admin || r.URL.Query().Get("all") != "1" {
		q += ` WHERE is_enabled IS TRUE`
	}
	q += ` ORDER BY sort_order ASC, id ASC`

	rows, err := h.Pool.Query(ctx, q)
	if err != nil {
		if portalMigrationErr(w, err) {
			return
		}
		httpx.Fail(w, http.StatusInternalServerError, "Ошибка сервера")
		return
	}
	defer rows.Close()

	list := []map[string]any{}
	for rows.Next() {
		item, err := scanPortalService(rows)
		if err != nil {
			httpx.Fail(w, http.StatusInternalServerError, "Ошибка сервера")
			return
		}
		list = append(list, item)
	}
	portalOK(w, list)
}

func (h *Portal) servicesPost(w http.ResponseWriter, r *http.Request) {
	if _, ok := requireAdmin(w, r, h.Auth); !ok {
		return
	}
	var body map[string]any
	if err := httpx.DecodeJSON(r, &body); err != nil {
		httpx.Fail(w, http.StatusBadRequest, "Некорректный JSON")
		return
	}
	// reorder: { action: "reorder", ids: [1,2,3] }
	if fmt.Sprint(body["action"]) == "reorder" {
		h.servicesReorder(w, r, body)
		return
	}
	item, errMsg := normalizePortalServiceInput(body, false)
	if errMsg != "" {
		httpx.Fail(w, http.StatusBadRequest, errMsg)
		return
	}
	ctx := r.Context()
	sortOrder := item.sortOrder
	if sortOrder == 0 {
		_ = h.Pool.QueryRow(ctx, `SELECT COALESCE(MAX(sort_order), 0) + 10 FROM public.portal_services`).Scan(&sortOrder)
	}
	var id int64
	err := h.Pool.QueryRow(ctx, `
		INSERT INTO public.portal_services (
			kind, label, description, icon, internal_key, path, external_url, sort_order, is_enabled
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
		RETURNING id`,
		item.kind, item.label, item.description, item.icon, item.internalKey,
		item.path, item.externalURL, sortOrder, item.isEnabled,
	).Scan(&id)
	if err != nil {
		if portalMigrationErr(w, err) {
			return
		}
		httpx.Fail(w, http.StatusInternalServerError, "Не удалось создать сервис")
		return
	}
	out, err := h.fetchPortalService(ctx, id)
	if err != nil {
		httpx.Fail(w, http.StatusInternalServerError, "Ошибка сервера")
		return
	}
	portalOK(w, out)
}

func (h *Portal) servicesPut(w http.ResponseWriter, r *http.Request) {
	if _, ok := requireAdmin(w, r, h.Auth); !ok {
		return
	}
	var body map[string]any
	if err := httpx.DecodeJSON(r, &body); err != nil {
		httpx.Fail(w, http.StatusBadRequest, "Некорректный JSON")
		return
	}
	id, err := toPositiveInt64(body["id"])
	if err != nil || id <= 0 {
		httpx.Fail(w, http.StatusBadRequest, "Не передан id")
		return
	}
	item, errMsg := normalizePortalServiceInput(body, true)
	if errMsg != "" {
		httpx.Fail(w, http.StatusBadRequest, errMsg)
		return
	}
	ctx := r.Context()
	tag, err := h.Pool.Exec(ctx, `
		UPDATE public.portal_services SET
			kind = $2,
			label = $3,
			description = $4,
			icon = $5,
			internal_key = $6,
			path = $7,
			external_url = $8,
			sort_order = $9,
			is_enabled = $10,
			updated_at = now()
		WHERE id = $1`,
		id, item.kind, item.label, item.description, item.icon, item.internalKey,
		item.path, item.externalURL, item.sortOrder, item.isEnabled,
	)
	if err != nil {
		if portalMigrationErr(w, err) {
			return
		}
		httpx.Fail(w, http.StatusInternalServerError, "Не удалось сохранить")
		return
	}
	if tag.RowsAffected() == 0 {
		httpx.Fail(w, http.StatusNotFound, "Сервис не найден")
		return
	}
	out, err := h.fetchPortalService(ctx, id)
	if err != nil {
		httpx.Fail(w, http.StatusInternalServerError, "Ошибка сервера")
		return
	}
	portalOK(w, out)
}

func (h *Portal) servicesDelete(w http.ResponseWriter, r *http.Request) {
	if _, ok := requireAdmin(w, r, h.Auth); !ok {
		return
	}
	idStr := r.URL.Query().Get("id")
	if idStr == "" {
		var body map[string]any
		_ = httpx.DecodeJSON(r, &body)
		if body != nil {
			idStr = fmt.Sprint(body["id"])
		}
	}
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		httpx.Fail(w, http.StatusBadRequest, "Не передан id")
		return
	}
	tag, err := h.Pool.Exec(r.Context(), `DELETE FROM public.portal_services WHERE id = $1`, id)
	if err != nil {
		if portalMigrationErr(w, err) {
			return
		}
		httpx.Fail(w, http.StatusInternalServerError, "Не удалось удалить")
		return
	}
	if tag.RowsAffected() == 0 {
		httpx.Fail(w, http.StatusNotFound, "Сервис не найден")
		return
	}
	portalOK(w, map[string]any{"deleted": true, "id": id})
}

func (h *Portal) servicesReorder(w http.ResponseWriter, r *http.Request, body map[string]any) {
	raw, ok := body["ids"].([]any)
	if !ok || len(raw) == 0 {
		httpx.Fail(w, http.StatusBadRequest, "Передайте ids")
		return
	}
	ctx := r.Context()
	tx, err := h.Pool.Begin(ctx)
	if err != nil {
		httpx.Fail(w, http.StatusInternalServerError, "Ошибка сервера")
		return
	}
	defer tx.Rollback(ctx)
	for i, v := range raw {
		id, err := toPositiveInt64(v)
		if err != nil || id <= 0 {
			continue
		}
		_, _ = tx.Exec(ctx, `UPDATE public.portal_services SET sort_order = $2, updated_at = now() WHERE id = $1`, id, (i+1)*10)
	}
	if err := tx.Commit(ctx); err != nil {
		httpx.Fail(w, http.StatusInternalServerError, "Не удалось сохранить порядок")
		return
	}
	portalOK(w, map[string]any{"reordered": true})
}

type portalServiceInput struct {
	kind         string
	label        string
	description  string
	icon         string
	internalKey  *string
	path         *string
	externalURL  *string
	sortOrder    int
	isEnabled    bool
}

func normalizePortalServiceInput(body map[string]any, requireSort bool) (portalServiceInput, string) {
	kind := strings.TrimSpace(fmt.Sprint(body["kind"]))
	if kind != "internal" && kind != "external" {
		return portalServiceInput{}, "Укажите kind: internal или external"
	}
	label := strings.TrimSpace(fmt.Sprint(body["label"]))
	if label == "" || label == "<nil>" {
		return portalServiceInput{}, "Укажите название"
	}
	description := strings.TrimSpace(fmt.Sprint(body["description"]))
	if description == "<nil>" {
		description = ""
	}
	icon := strings.TrimSpace(fmt.Sprint(body["icon"]))
	if icon == "" || icon == "<nil>" {
		icon = "i-lucide-layout-grid"
	}
	out := portalServiceInput{
		kind:        kind,
		label:       label,
		description: description,
		icon:        icon,
		isEnabled:   true,
	}
	if v, ok := body["isEnabled"]; ok {
		out.isEnabled = truthy(v)
	}
	if v, ok := body["sortOrder"]; ok {
		if n, err := toPositiveInt64(v); err == nil {
			out.sortOrder = int(n)
		}
	} else if requireSort {
		out.sortOrder = 0
	}

	if kind == "internal" {
		path := strings.TrimSpace(fmt.Sprint(body["path"]))
		if path == "" || path == "<nil>" {
			return portalServiceInput{}, "Укажите path для внутреннего сервиса"
		}
		if !strings.HasPrefix(path, "/") {
			path = "/" + path
		}
		out.path = &path
		key := strings.TrimSpace(fmt.Sprint(body["internalKey"]))
		if key != "" && key != "<nil>" {
			out.internalKey = &key
		}
	} else {
		ext := strings.TrimSpace(fmt.Sprint(body["externalUrl"]))
		if ext == "" || ext == "<nil>" {
			return portalServiceInput{}, "Укажите externalUrl"
		}
		u, err := url.Parse(ext)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
			return portalServiceInput{}, "externalUrl должен быть http(s) ссылкой"
		}
		out.externalURL = &ext
	}
	return out, ""
}

func (h *Portal) fetchPortalService(ctx context.Context, id int64) (map[string]any, error) {
	row, err := h.Pool.Query(ctx, `
		SELECT id, kind, label, description, icon, internal_key, path, external_url,
		       sort_order, is_enabled, created_at, updated_at
		FROM public.portal_services WHERE id = $1`, id)
	if err != nil {
		return nil, err
	}
	defer row.Close()
	if !row.Next() {
		return nil, pgx.ErrNoRows
	}
	return scanPortalService(row)
}

func scanPortalService(rows pgx.Rows) (map[string]any, error) {
	var (
		id                       int64
		kind, label, description string
		icon                     string
		internalKey, path, ext   *string
		sortOrder                int
		isEnabled                bool
		createdAt, updatedAt     any
	)
	if err := rows.Scan(&id, &kind, &label, &description, &icon, &internalKey, &path, &ext, &sortOrder, &isEnabled, &createdAt, &updatedAt); err != nil {
		return nil, err
	}
	return map[string]any{
		"id": id, "kind": kind, "label": label, "description": description, "icon": icon,
		"internalKey": internalKey, "path": path, "externalUrl": ext,
		"sortOrder": sortOrder, "isEnabled": isEnabled,
		"createdAt": createdAt, "updatedAt": updatedAt,
	}, nil
}

func toPositiveInt64(v any) (int64, error) {
	switch t := v.(type) {
	case float64:
		return int64(t), nil
	case int64:
		return t, nil
	case int:
		return int64(t), nil
	case json.Number:
		return t.Int64()
	case string:
		return strconv.ParseInt(strings.TrimSpace(t), 10, 64)
	default:
		s := strings.TrimSpace(fmt.Sprint(v))
		if s == "" || s == "<nil>" {
			return 0, errors.New("empty")
		}
		return strconv.ParseInt(s, 10, 64)
	}
}

func truthy(v any) bool {
	switch t := v.(type) {
	case bool:
		return t
	case string:
		s := strings.ToLower(strings.TrimSpace(t))
		return s == "1" || s == "true" || s == "yes"
	case float64:
		return t != 0
	case int:
		return t != 0
	default:
		return false
	}
}
