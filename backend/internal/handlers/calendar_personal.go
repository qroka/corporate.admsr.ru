package handlers

import (
	"errors"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"corporate.admsr.ru/backend/internal/auth"
	"corporate.admsr.ru/backend/internal/httpx"
)

// Запись на мероприятия и личный календарь (V16, Q-03). Раньше это жило в localStorage
// браузера (`events-rsvp:v1`, `portal-calendar-local:v1`) и не было видно нигде, кроме
// устройства. Личность — только из серверной сессии: чужие записи не читаются и не меняются.

const (
	calendarTitleMaxRunes    = 200
	calendarLocationMaxRunes = 200
	calendarMaxPerUser       = 500
)

var calendarTimeRe = regexp.MustCompile(`^([01]?\d|2[0-3]):[0-5]\d$`)

func calendarFailDB(w http.ResponseWriter, op string, err error) {
	log.Printf("calendar personal: %s: %v", op, err)
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "42P01" { // undefined_table
		httpx.Fail(w, http.StatusServiceUnavailable, "Запись на мероприятия и личный календарь ещё не включены на сервере: нужна миграция V16__calendar_personal.sql")
		return
	}
	httpx.Fail(w, http.StatusInternalServerError, "Ошибка подключения к БД")
}

// ── /api/event_rsvp.php ──────────────────────────────────────────────────────

// EventRSVP: GET — id мероприятий, на которые записан текущий пользователь;
// POST {eventId, joined} — записаться или отменить запись (идемпотентно).
type EventRSVP struct {
	Pool *pgxpool.Pool
	Auth *auth.Service
}

func (h *EventRSVP) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		httpx.MethodNotAllowed(w)
		return
	}
	cur, ok := requireUser(w, r, h.Auth)
	if !ok {
		return
	}
	if r.Method == http.MethodGet {
		h.list(w, r, cur)
		return
	}
	var body map[string]any
	if err := httpx.DecodeJSON(r, &body); err != nil {
		httpx.Fail(w, http.StatusBadRequest, "Некорректный JSON")
		return
	}
	eventID := int64Of(body["eventId"])
	if eventID <= 0 {
		httpx.Fail(w, http.StatusBadRequest, "Не передан eventId")
		return
	}
	joined, _ := body["joined"].(bool)
	if joined {
		var exists int64
		if err := h.Pool.QueryRow(r.Context(), `SELECT id FROM public.events WHERE id = $1`, eventID).Scan(&exists); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				httpx.Fail(w, http.StatusNotFound, "Мероприятие не найдено")
				return
			}
			calendarFailDB(w, "rsvp event", err)
			return
		}
		if _, err := h.Pool.Exec(r.Context(), `
			INSERT INTO public.event_rsvps (event_id, user_id) VALUES ($1, $2)
			ON CONFLICT (event_id, user_id) DO NOTHING`, eventID, cur.ID); err != nil {
			calendarFailDB(w, "rsvp add", err)
			return
		}
	} else if _, err := h.Pool.Exec(r.Context(),
		`DELETE FROM public.event_rsvps WHERE event_id = $1 AND user_id = $2`, eventID, cur.ID); err != nil {
		calendarFailDB(w, "rsvp remove", err)
		return
	}
	httpx.OK(w, map[string]any{"eventId": eventID, "joined": joined}, "")
}

func (h *EventRSVP) list(w http.ResponseWriter, r *http.Request, cur *auth.User) {
	rows, err := h.Pool.Query(r.Context(),
		`SELECT event_id FROM public.event_rsvps WHERE user_id = $1 ORDER BY event_id`, cur.ID)
	if err != nil {
		calendarFailDB(w, "rsvp list", err)
		return
	}
	defer rows.Close()
	ids := []int64{}
	for rows.Next() {
		var id int64
		if rows.Scan(&id) == nil {
			ids = append(ids, id)
		}
	}
	httpx.OK(w, map[string]any{"eventIds": ids}, "")
}

// ── /api/calendar_entries.php ────────────────────────────────────────────────

// CalendarEntries: GET — личные записи и встречи текущего пользователя;
// POST {action:"create", source, dateKey, title, timeStart?, timeEnd?, location?};
// POST {action:"delete", id} — только свои.
type CalendarEntries struct {
	Pool *pgxpool.Pool
	Auth *auth.Service
}

func (h *CalendarEntries) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		httpx.MethodNotAllowed(w)
		return
	}
	cur, ok := requireUser(w, r, h.Auth)
	if !ok {
		return
	}
	if r.Method == http.MethodGet {
		h.list(w, r, cur)
		return
	}
	var body map[string]any
	if err := httpx.DecodeJSON(r, &body); err != nil {
		httpx.Fail(w, http.StatusBadRequest, "Некорректный JSON")
		return
	}
	switch strVal(body["action"]) {
	case "create":
		h.create(w, r, cur, body)
	case "delete":
		h.remove(w, r, cur, body)
	default:
		httpx.Fail(w, http.StatusBadRequest, "Неизвестное действие")
	}
}

func (h *CalendarEntries) list(w http.ResponseWriter, r *http.Request, cur *auth.User) {
	rows, err := h.Pool.Query(r.Context(), `
		SELECT id, source, to_char(date_key, 'YYYY-MM-DD'), title,
		       COALESCE(time_start, ''), COALESCE(time_end, ''), COALESCE(location, '')
		FROM public.calendar_entries WHERE user_id = $1
		ORDER BY date_key, COALESCE(time_start, ''), id`, cur.ID)
	if err != nil {
		calendarFailDB(w, "entries list", err)
		return
	}
	defer rows.Close()
	out := []map[string]any{}
	for rows.Next() {
		var id int64
		var source, dateKey, title, ts, te, loc string
		if rows.Scan(&id, &source, &dateKey, &title, &ts, &te, &loc) == nil {
			out = append(out, calendarEntryJSON(id, source, dateKey, title, ts, te, loc))
		}
	}
	httpx.OK(w, map[string]any{"entries": out}, "")
}

// calendarEntryJSON отдаёт пустые необязательные поля как отсутствующие (фронт ждёт undefined).
func calendarEntryJSON(id int64, source, dateKey, title, ts, te, loc string) map[string]any {
	m := map[string]any{"id": id, "source": source, "dateKey": dateKey, "title": title}
	if ts != "" {
		m["timeStart"] = ts
	}
	if te != "" {
		m["timeEnd"] = te
	}
	if loc != "" {
		m["location"] = loc
	}
	return m
}

// calendarEntryInput проверяет тело create; возвращает текст ошибки или "".
func calendarEntryInput(body map[string]any) (source, title, dateKey, ts, te, loc, problem string) {
	source = strVal(body["source"])
	if source != "meeting" && source != "personal" {
		return "", "", "", "", "", "", "Тип записи: встреча или личное"
	}
	title = strings.TrimSpace(strVal(body["title"]))
	if title == "" {
		return "", "", "", "", "", "", "Укажите название"
	}
	if utf8.RuneCountInString(title) > calendarTitleMaxRunes {
		return "", "", "", "", "", "", "Название длиннее 200 символов"
	}
	dateKey = strVal(body["dateKey"])
	if _, err := time.Parse("2006-01-02", dateKey); err != nil {
		return "", "", "", "", "", "", "Некорректная дата"
	}
	ts = strings.TrimSpace(strVal(body["timeStart"]))
	te = strings.TrimSpace(strVal(body["timeEnd"]))
	for _, t := range []string{ts, te} {
		if t != "" && !calendarTimeRe.MatchString(t) {
			return "", "", "", "", "", "", "Некорректное время"
		}
	}
	loc = strings.TrimSpace(strVal(body["location"]))
	if utf8.RuneCountInString(loc) > calendarLocationMaxRunes {
		return "", "", "", "", "", "", "Место длиннее 200 символов"
	}
	return source, title, dateKey, ts, te, loc, ""
}

func (h *CalendarEntries) create(w http.ResponseWriter, r *http.Request, cur *auth.User, body map[string]any) {
	source, title, dateKey, ts, te, loc, problem := calendarEntryInput(body)
	if problem != "" {
		httpx.Fail(w, http.StatusUnprocessableEntity, problem)
		return
	}
	var n int64
	if err := h.Pool.QueryRow(r.Context(),
		`SELECT COUNT(*) FROM public.calendar_entries WHERE user_id = $1`, cur.ID).Scan(&n); err != nil {
		calendarFailDB(w, "entries count", err)
		return
	}
	if n >= calendarMaxPerUser {
		httpx.Fail(w, http.StatusUnprocessableEntity, "Слишком много записей в календаре — удалите ненужные")
		return
	}
	var id int64
	if err := h.Pool.QueryRow(r.Context(), `
		INSERT INTO public.calendar_entries (user_id, source, date_key, title, time_start, time_end, location)
		VALUES ($1, $2, $3::date, $4, NULLIF($5, ''), NULLIF($6, ''), NULLIF($7, ''))
		RETURNING id`, cur.ID, source, dateKey, title, ts, te, loc).Scan(&id); err != nil {
		calendarFailDB(w, "entries create", err)
		return
	}
	httpx.OK(w, map[string]any{"entry": calendarEntryJSON(id, source, dateKey, title, ts, te, loc)}, "Запись добавлена")
}

func (h *CalendarEntries) remove(w http.ResponseWriter, r *http.Request, cur *auth.User, body map[string]any) {
	id := int64Of(body["id"])
	if id <= 0 {
		httpx.Fail(w, http.StatusBadRequest, "Некорректный id")
		return
	}
	// user_id в условии: чужую запись удалить нельзя, а о её существовании не сообщаем.
	tag, err := h.Pool.Exec(r.Context(),
		`DELETE FROM public.calendar_entries WHERE id = $1 AND user_id = $2`, id, cur.ID)
	if err != nil {
		calendarFailDB(w, "entries delete", err)
		return
	}
	if tag.RowsAffected() == 0 {
		httpx.Fail(w, http.StatusNotFound, "Запись не найдена")
		return
	}
	httpx.OK(w, nil, "Запись удалена")
}
