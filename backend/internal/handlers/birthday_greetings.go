package handlers

import (
	"errors"
	"math"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"corporate.admsr.ru/backend/internal/auth"
	"corporate.admsr.ru/backend/internal/httpx"
)

// Поздравление с днём рождения — запись на стене именинника с отметкой
// wall_posts.birthday_year (V17). Поздравить можно в день рождения и
// birthdayGreetWindowDays дней после, один раз в год от одного автора.
// Окно и «уже поздравил» дублируются на фронте (src/composables/useBirthdayGreetings.ts)
// только для показа кнопки — решает сервер.

const birthdayGreetWindowDays = 3

// birthdayGreetingYear — год дня рождения, с которым сейчас можно поздравить,
// или ok=false, если сегодня вне окна. Учитывает переход через Новый год:
// 2 января ещё можно поздравить с 31 декабря прошлого года.
// 29 февраля в невисокосный год time.Date переносит на 1 марта.
func birthdayGreetingYear(month, day int, now time.Time) (int, bool) {
	if month < 1 || month > 12 || day < 1 || day > 31 {
		return 0, false
	}
	loc := now.Location()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
	for _, y := range []int{today.Year(), today.Year() - 1} {
		bday := time.Date(y, time.Month(month), day, 0, 0, 0, 0, loc)
		days := int(math.Round(today.Sub(bday).Hours() / 24))
		if days >= 0 && days <= birthdayGreetWindowDays {
			return y, true
		}
	}
	return 0, false
}

func (h *ProfileWall) greet(w http.ResponseWriter, r *http.Request, cur *auth.User, body map[string]any) {
	ownerID := int64Of(body["userId"])
	if ownerID <= 0 {
		httpx.Fail(w, http.StatusBadRequest, "Некорректный userId")
		return
	}
	if ownerID == cur.ID {
		httpx.Fail(w, http.StatusUnprocessableEntity, "Себя поздравить нельзя")
		return
	}
	content, msg := normalizeWallContent(body["content"])
	if msg != "" {
		httpx.Fail(w, http.StatusUnprocessableEntity, msg)
		return
	}

	var surname, firstname, lastname *string
	err := h.Pool.QueryRow(r.Context(), `
		SELECT surname, firstname, lastname FROM public.user_info
		WHERE id = $1 AND status = true`, ownerID).Scan(&surname, &firstname, &lastname)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			httpx.Fail(w, http.StatusNotFound, "Сотрудник не найден")
			return
		}
		wallFailDB(w, "greet owner", err)
		return
	}
	month, day, ok := h.Birthdays.BirthdayOf(joinNames(surname, firstname, lastname))
	if !ok {
		httpx.Fail(w, http.StatusUnprocessableEntity, "День рождения сотрудника неизвестен")
		return
	}
	year, ok := birthdayGreetingYear(month, day, time.Now())
	if !ok {
		httpx.Fail(w, http.StatusUnprocessableEntity, "Поздравить можно в день рождения и в течение трёх дней после")
		return
	}
	if content, ok = profanityGate(w, cur.ID, body, content); !ok {
		return
	}

	var id int64
	err = h.Pool.QueryRow(r.Context(), `
		INSERT INTO public.wall_posts (owner_id, author_id, content, birthday_year)
		VALUES ($1, $2, $3, $4) RETURNING id`, ownerID, cur.ID, content, year).Scan(&id)
	if err != nil {
		if isUniqueViolation(err) {
			httpx.Fail(w, http.StatusConflict, "Вы уже поздравили этого сотрудника")
			return
		}
		if isUndefinedColumn(err) {
			httpx.Fail(w, http.StatusServiceUnavailable, "Поздравления ещё не включены на сервере: нужна миграция V17__birthday_greetings_notifications.sql")
			return
		}
		wallFailDB(w, "greet", err)
		return
	}
	notifyWallPost(r.Context(), h.Pool, ownerID, cur.ID, id, notifyKindBirthdayGreeting)

	p, err := h.fetchPost(r.Context(), id)
	if err != nil {
		wallFailDB(w, "fetch greeting", err)
		return
	}
	out := h.fmtPosts(r.Context(), cur, []wallPostRow{p})[0]
	out["birthdayYear"] = year
	httpx.OK(w, out, "Поздравление опубликовано")
}

// myGreetings — мои поздравления за этот и прошлый год: фронт по ним
// показывает «Вы поздравили» вместо кнопки.
func (h *ProfileWall) myGreetings(w http.ResponseWriter, r *http.Request, cur *auth.User) {
	rows, err := h.Pool.Query(r.Context(), `
		SELECT id, owner_id, birthday_year FROM public.wall_posts
		WHERE author_id = $1 AND birthday_year >= $2
		ORDER BY id`, cur.ID, time.Now().Year()-1)
	if err != nil {
		if isUndefinedColumn(err) {
			httpx.OK(w, []map[string]any{}, "OK")
			return
		}
		wallFailDB(w, "my greetings", err)
		return
	}
	defer rows.Close()
	list := []map[string]any{}
	for rows.Next() {
		var id, ownerID int64
		var year int
		if rows.Scan(&id, &ownerID, &year) != nil {
			continue
		}
		list = append(list, map[string]any{"postId": id, "userId": ownerID, "year": year})
	}
	httpx.OK(w, list, "OK")
}

func isUndefinedColumn(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "42703" // undefined_column: нет V17
}
