package handlers

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"corporate.admsr.ru/backend/internal/auth"
	"corporate.admsr.ru/backend/internal/httpx"
)

type Absence struct {
	Pool *pgxpool.Pool
	Auth *auth.Service
}

func (h *Absence) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Ensure portal TZ for this connection's queries
	_, _ = h.Pool.Exec(r.Context(), `SET TIME ZONE 'Asia/Yekaterinburg'`)

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
		absenceFail(w, http.StatusMethodNotAllowed, "Метод не разрешён")
	}
}

func absenceOK(w http.ResponseWriter, data any) {
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"success": true, "data": data})
}

func absenceCreated(w http.ResponseWriter, data any) {
	httpx.WriteJSON(w, http.StatusCreated, map[string]any{"success": true, "data": data})
}

func absenceFail(w http.ResponseWriter, code int, msg string) {
	httpx.WriteJSON(w, code, map[string]any{"success": false, "error": msg})
}

func (h *Absence) get(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	if idStr := q.Get("id"); idStr != "" {
		id, _ := strconv.ParseInt(idStr, 10, 64)
		row, err := h.fetchOne(r.Context(), id)
		if err != nil {
			absenceFail(w, http.StatusNotFound, "Запись не найдена")
			return
		}
		absenceOK(w, row)
		return
	}

	cond := []string{}
	args := []any{}
	n := 1

	if v := q.Get("user_id"); v != "" {
		uid, _ := strconv.ParseInt(v, 10, 64)
		cond = append(cond, fmt.Sprintf("user_id=$%d", n))
		args = append(args, uid)
		n++
	}
	if v := q.Get("ofo"); v != "" {
		if _, ok := requireSection(w, r, h.Pool, h.Auth, "absence_journal"); !ok {
			return
		}
		ofo, _ := strconv.ParseInt(v, 10, 64)
		cond = append(cond, fmt.Sprintf("ofo=$%d", n))
		args = append(args, ofo)
		n++
	}
	switch strings.ToLower(strings.TrimSpace(q.Get("status"))) {
	case "active":
		cond = append(cond, "end_datetime IS NULL")
	case "completed":
		cond = append(cond, "end_datetime IS NOT NULL")
	}
	if s := strings.TrimSpace(q.Get("q")); s != "" {
		cond = append(cond, fmt.Sprintf("(fio ILIKE $%d OR COALESCE(reason,'') ILIKE $%d)", n, n))
		args = append(args, "%"+s+"%")
		n++
	}
	switch strings.ToLower(strings.TrimSpace(q.Get("period"))) {
	case "today":
		cond = append(cond, "start_datetime::date = CURRENT_DATE")
	case "week":
		cond = append(cond, "start_datetime >= CURRENT_DATE - INTERVAL '7 days'")
	case "month":
		cond = append(cond, "start_datetime >= CURRENT_DATE - INTERVAL '30 days'")
	}

	limit := 500
	if v := q.Get("limit"); v != "" {
		if x, err := strconv.Atoi(v); err == nil {
			limit = x
		}
	}
	if limit < 1 {
		limit = 1
	}
	if limit > 5000 {
		limit = 5000
	}
	offset := 0
	if v := q.Get("offset"); v != "" {
		if x, err := strconv.Atoi(v); err == nil && x > 0 {
			offset = x
		}
	}

	sql := `SELECT id, user_id, fio, ofo, role, start_datetime, end_datetime, reason, created_at FROM public.absence_journal`
	if len(cond) > 0 {
		sql += " WHERE " + strings.Join(cond, " AND ")
	}
	sql += fmt.Sprintf(" ORDER BY start_datetime DESC LIMIT $%d OFFSET $%d", n, n+1)
	args = append(args, limit, offset)

	rows, err := h.Pool.Query(r.Context(), sql, args...)
	if err != nil {
		absenceFail(w, http.StatusInternalServerError, "Ошибка подключения к БД")
		return
	}
	defer rows.Close()
	var items []map[string]any
	for rows.Next() {
		item, err := scanAbsence(rows)
		if err != nil {
			absenceFail(w, http.StatusInternalServerError, "Ошибка подключения к БД")
			return
		}
		items = append(items, item)
	}
	if items == nil {
		items = []map[string]any{}
	}
	absenceOK(w, items)
}

func (h *Absence) post(w http.ResponseWriter, r *http.Request) {
	var body map[string]any
	if err := httpx.DecodeJSON(r, &body); err != nil {
		absenceFail(w, http.StatusBadRequest, "Некорректный JSON")
		return
	}
	userID := int64(num(body["user_id"]))
	fio := strings.TrimSpace(strVal(body["fio"]))
	ofo := int64(num(body["ofo"]))
	role := strings.TrimSpace(strVal(body["role"]))
	start := strings.TrimSpace(strVal(body["start_datetime"]))
	if userID <= 0 {
		absenceFail(w, http.StatusBadRequest, "user_id обязателен")
		return
	}
	if fio == "" {
		absenceFail(w, http.StatusBadRequest, "fio обязателен")
		return
	}
	if ofo <= 0 {
		absenceFail(w, http.StatusBadRequest, "ofo обязателен")
		return
	}
	if role == "" {
		absenceFail(w, http.StatusBadRequest, "role обязателен")
		return
	}
	if start == "" || !validDatetime(start) {
		absenceFail(w, http.StatusBadRequest, "start_datetime обязателен")
		return
	}

	actor, _ := h.Auth.CurrentUser(r.Context(), r)
	if actor != nil {
		can := auth.CanEditSection(r.Context(), h.Pool, actor, "absence_journal")
		if !can && actor.ID != userID {
			absenceFail(w, http.StatusForbidden, "Недостаточно прав")
			return
		}
	}

	var end *string
	if v := strings.TrimSpace(strVal(body["end_datetime"])); v != "" {
		e := roundDtToStep(normDt(v))
		if !validDatetime(e) {
			absenceFail(w, http.StatusBadRequest, "Некорректный формат end_datetime")
			return
		}
		end = &e
	}
	var reason *string
	if _, ok := body["reason"]; ok {
		r := strings.TrimSpace(strVal(body["reason"]))
		if r != "" {
			reason = &r
		}
	}
	startNorm := roundDtToStep(normDt(start))
	created := nowPortal()

	row := h.Pool.QueryRow(r.Context(), `
		INSERT INTO public.absence_journal
			(user_id, fio, ofo, role, start_datetime, end_datetime, reason, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
		RETURNING id, user_id, fio, ofo, role, start_datetime, end_datetime, reason, created_at`,
		userID, fio, ofo, role, startNorm, end, reason, created)

	item, err := scanAbsence(row)
	if err != nil {
		absenceFail(w, http.StatusInternalServerError, "Не удалось создать запись")
		return
	}
	absenceCreated(w, item)
}

func (h *Absence) put(w http.ResponseWriter, r *http.Request) {
	id, ok := queryID(r)
	if !ok {
		absenceFail(w, http.StatusBadRequest, "Не указан id")
		return
	}
	existing, err := h.fetchOne(r.Context(), id)
	if err != nil {
		absenceFail(w, http.StatusNotFound, "Запись не найдена")
		return
	}
	var body map[string]any
	if err := httpx.DecodeJSON(r, &body); err != nil {
		absenceFail(w, http.StatusBadRequest, "Некорректный JSON")
		return
	}
	actor, _ := h.Auth.CurrentUser(r.Context(), r)
	if actor != nil {
		uid, _ := existing["user_id"].(int64)
		can := auth.CanEditSection(r.Context(), h.Pool, actor, "absence_journal")
		if !can && actor.ID != uid {
			absenceFail(w, http.StatusForbidden, "Недостаточно прав")
			return
		}
	}
	sets := []string{}
	args := []any{}
	n := 1
	if v := strings.TrimSpace(strVal(body["start_datetime"])); v != "" {
		s := roundDtToStep(normDt(v))
		if !validDatetime(s) {
			absenceFail(w, http.StatusBadRequest, "Некорректный формат start_datetime")
			return
		}
		sets = append(sets, fmt.Sprintf("start_datetime=$%d", n))
		args = append(args, s)
		n++
	}
	if _, has := body["end_datetime"]; has {
		end := strings.TrimSpace(strVal(body["end_datetime"]))
		if end == "" {
			sets = append(sets, "end_datetime=NULL")
		} else {
			e := roundDtToStep(normDt(end))
			if !validDatetime(e) {
				absenceFail(w, http.StatusBadRequest, "Некорректный формат end_datetime")
				return
			}
			sets = append(sets, fmt.Sprintf("end_datetime=$%d", n))
			args = append(args, e)
			n++
		}
	}
	if _, has := body["reason"]; has {
		reason := strings.TrimSpace(strVal(body["reason"]))
		sets = append(sets, fmt.Sprintf("reason=$%d", n))
		if reason == "" {
			args = append(args, nil)
		} else {
			args = append(args, reason)
		}
		n++
	}
	if len(sets) == 0 {
		absenceFail(w, http.StatusBadRequest, "Нет полей для обновления")
		return
	}
	args = append(args, id)
	row := h.Pool.QueryRow(r.Context(),
		`UPDATE public.absence_journal SET `+strings.Join(sets, ", ")+
			fmt.Sprintf(` WHERE id=$%d RETURNING id, user_id, fio, ofo, role, start_datetime, end_datetime, reason, created_at`, n),
		args...)
	item, err := scanAbsence(row)
	if err != nil {
		absenceFail(w, http.StatusInternalServerError, "Не удалось обновить запись")
		return
	}
	absenceOK(w, item)
}

func (h *Absence) del(w http.ResponseWriter, r *http.Request) {
	id, ok := queryID(r)
	if !ok {
		absenceFail(w, http.StatusBadRequest, "Не указан id")
		return
	}
	var deleted int64
	err := h.Pool.QueryRow(r.Context(), `DELETE FROM public.absence_journal WHERE id=$1 RETURNING id`, id).Scan(&deleted)
	if err != nil {
		absenceFail(w, http.StatusNotFound, "Запись не найдена")
		return
	}
	absenceOK(w, map[string]any{"deleted_id": deleted})
}

func (h *Absence) fetchOne(ctx context.Context, id int64) (map[string]any, error) {
	row := h.Pool.QueryRow(ctx, `
		SELECT id, user_id, fio, ofo, role, start_datetime, end_datetime, reason, created_at
		FROM public.absence_journal WHERE id=$1`, id)
	return scanAbsence(row)
}

type absenceScanner interface {
	Scan(dest ...any) error
}

func scanAbsence(row absenceScanner) (map[string]any, error) {
	var (
		id, userID, ofo                                   int64
		fio, role                                         string
		start, end, reason, created                       *string
		startT, endT, createdT                            *time.Time
	)
	// Try time.Time first via interface — use *string from to_char for simplicity
	err := row.Scan(&id, &userID, &fio, &ofo, &role, &startT, &endT, &reason, &createdT)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, err
		}
		return nil, err
	}
	_ = start
	_ = end
	_ = created
	status := "completed"
	var endOut any
	if endT == nil {
		status = "active"
		endOut = nil
	} else {
		endOut = endT.Format("2006-01-02 15:04:05")
	}
	var createdOut any
	if createdT != nil {
		createdOut = createdT.Format("2006-01-02 15:04:05")
	}
	startOut := ""
	if startT != nil {
		startOut = startT.Format("2006-01-02 15:04:05")
	}
	var reasonOut any
	if reason != nil {
		reasonOut = *reason
	}
	return map[string]any{
		"id": id, "user_id": userID, "fio": fio, "ofo": ofo, "role": role,
		"start_datetime": startOut, "end_datetime": endOut, "reason": reasonOut,
		"created_at": createdOut, "status": status,
	}, nil
}

func num(v any) float64 {
	switch t := v.(type) {
	case float64:
		return t
	case int:
		return float64(t)
	case int64:
		return float64(t)
	case string:
		f, _ := strconv.ParseFloat(t, 64)
		return f
	default:
		return 0
	}
}

func validDatetime(v string) bool {
	v = normDt(v)
	layouts := []string{"2006-01-02 15:04:05", "2006-01-02T15:04:05", "2006-01-02T15:04"}
	for _, l := range layouts {
		if _, err := time.ParseInLocation(l, v, portalLoc); err == nil {
			return true
		}
	}
	return false
}

func normDt(v string) string {
	v = strings.ReplaceAll(strings.TrimSpace(v), "T", " ")
	if hmOnly.MatchString(v) {
		v += ":00"
	}
	return v
}

var hmOnly = regexp.MustCompile(`^\d{4}-\d{2}-\d{2} \d{2}:\d{2}$`)

var portalLoc *time.Location

func init() {
	var err error
	portalLoc, err = time.LoadLocation("Asia/Yekaterinburg")
	if err != nil {
		portalLoc = time.FixedZone("YEKT", 5*3600)
	}
}

func nowPortal() string {
	return time.Now().In(portalLoc).Format("2006-01-02 15:04:05")
}

func roundDtToStep(v string) string {
	v = normDt(v)
	dt, err := time.ParseInLocation("2006-01-02 15:04:05", v, portalLoc)
	if err != nil {
		return v
	}
	m := dt.Minute()
	rounded := int(float64(m)/5.0+0.5) * 5
	if rounded >= 60 {
		dt = dt.Add(time.Hour)
		rounded = 0
	}
	return time.Date(dt.Year(), dt.Month(), dt.Day(), dt.Hour(), rounded, 0, 0, portalLoc).Format("2006-01-02 15:04:05")
}
