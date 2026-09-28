package handlers

import (
	"context"
	"errors"
	"log"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"

	"corporate.admsr.ru/backend/internal/httpx"
)

// NewsReactionKeys — допустимые реакции в порядке показа. Дублируется во фронте:
// src/composables/useNewsReactions.ts (NEWS_REACTIONS) — менять парами.
var NewsReactionKeys = []string{"like", "love", "haha", "wow", "sad", "fire", "clap", "party"}

// reactionsTableMissing — на сервере не применена V11 (deploy.sh пропускает
// миграции без PGPASSWORD / psql). Отвечаем понятно, а не безликим 500.
func reactionsTableMissing(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "42P01" // undefined_table
	}
	return strings.Contains(err.Error(), "news_reactions") && strings.Contains(err.Error(), "does not exist")
}

func isNewsReaction(key string) bool {
	for _, k := range NewsReactionKeys {
		if k == key {
			return true
		}
	}
	return false
}

// viewerID — id вошедшего сотрудника или 0. Новости читают и без входа (киоск),
// поэтому отсутствие сессии здесь не ошибка: просто нет «моих» реакций.
func (h *News) viewerID(r *http.Request) int64 {
	if h.Auth == nil {
		return 0
	}
	u, err := h.Auth.CurrentUser(r.Context(), r)
	if err != nil || u == nil {
		return 0
	}
	return u.ID
}

// reactionSummaries — для каждой новости список {key, count, mine} в порядке
// NewsReactionKeys; реакции с нулём не попадают. legacyLikes — накопленные до
// появления реакций анонимные лайки (news.likes), прибавляются к «like».
func (h *News) reactionSummaries(ctx context.Context, ids []int64, viewer int64, legacyLikes map[int64]int64) map[int64][]map[string]any {
	type agg struct {
		count int64
		mine  bool
	}
	byNews := map[int64]map[string]*agg{}
	for _, id := range ids {
		byNews[id] = map[string]*agg{}
	}
	if len(ids) > 0 {
		rows, err := h.Pool.Query(ctx, `
			SELECT news_id, reaction, COUNT(*)::bigint, BOOL_OR(user_id = $2)
			FROM public.news_reactions
			WHERE news_id = ANY($1)
			GROUP BY news_id, reaction`, ids, viewer)
		if err != nil {
			// Ленту не роняем: без таблицы новости читаются, просто без реакций.
			log.Printf("news reactions: summary query failed: %v", err)
		} else {
			for rows.Next() {
				var newsID, count int64
				var key string
				var mine bool
				if rows.Scan(&newsID, &key, &count, &mine) != nil {
					continue
				}
				if m, ok := byNews[newsID]; ok && isNewsReaction(key) {
					m[key] = &agg{count: count, mine: mine}
				}
			}
			rows.Close()
		}
	}
	out := map[int64][]map[string]any{}
	for _, id := range ids {
		m := byNews[id]
		if extra := legacyLikes[id]; extra > 0 {
			if m["like"] == nil {
				m["like"] = &agg{}
			}
			m["like"].count += extra
		}
		list := []map[string]any{}
		for _, key := range NewsReactionKeys {
			if a := m[key]; a != nil && a.count > 0 {
				list = append(list, map[string]any{"key": key, "count": a.count, "mine": a.mine})
			}
		}
		out[id] = list
	}
	return out
}

// attachReactions дописывает поле reactions в уже отформатированные новости.
func (h *News) attachReactions(r *http.Request, items []map[string]any) {
	if len(items) == 0 {
		return
	}
	ids := make([]int64, 0, len(items))
	legacy := map[int64]int64{}
	for _, it := range items {
		id, _ := it["id"].(int64)
		if id <= 0 {
			continue
		}
		ids = append(ids, id)
		if l, ok := it["likes"].(int64); ok {
			legacy[id] = l
		}
	}
	sums := h.reactionSummaries(r.Context(), ids, h.viewerID(r), legacy)
	for _, it := range items {
		id, _ := it["id"].(int64)
		if s, ok := sums[id]; ok {
			it["reactions"] = s
		} else {
			it["reactions"] = []map[string]any{}
		}
	}
}

func (h *News) oneWithReactions(r *http.Request, row newsRow) map[string]any {
	item := fmtNews(row)
	h.attachReactions(r, []map[string]any{item})
	return item
}

// react — поставить или снять реакцию. Личность — только из сессии.
func (h *News) react(w http.ResponseWriter, r *http.Request, newsID int64, reaction string, active bool) {
	user, ok := requireUser(w, r, h.Auth)
	if !ok {
		return
	}
	reaction = strings.ToLower(strings.TrimSpace(reaction))
	if !isNewsReaction(reaction) {
		httpx.Fail(w, http.StatusUnprocessableEntity, "Неизвестная реакция")
		return
	}
	row, err := h.fetchOne(r.Context(), newsID)
	if err != nil {
		httpx.Fail(w, http.StatusNotFound, "Новость не найдена")
		return
	}
	if active {
		_, err = h.Pool.Exec(r.Context(), `
			INSERT INTO public.news_reactions (news_id, user_id, reaction)
			VALUES ($1, $2, $3)
			ON CONFLICT (news_id, user_id, reaction) DO NOTHING`, newsID, user.ID, reaction)
	} else {
		_, err = h.Pool.Exec(r.Context(), `
			DELETE FROM public.news_reactions
			WHERE news_id = $1 AND user_id = $2 AND reaction = $3`, newsID, user.ID, reaction)
	}
	if err != nil {
		log.Printf("news reactions: save news=%d user=%d reaction=%s: %v", newsID, user.ID, reaction, err)
		if reactionsTableMissing(err) {
			httpx.Fail(w, http.StatusServiceUnavailable, "Реакции ещё не включены на сервере: нужна миграция V11__news_reactions.sql")
			return
		}
		httpx.Fail(w, http.StatusInternalServerError, "Не удалось сохранить реакцию")
		return
	}
	httpx.OK(w, h.oneWithReactions(r, row), "Реакция сохранена")
}

// reactors — кто поставил реакцию (для подсказки над реакцией). Только для вошедших:
// список сотрудников — не публичные данные.
func (h *News) reactors(w http.ResponseWriter, r *http.Request, newsID int64, reaction string) {
	if _, ok := requireUser(w, r, h.Auth); !ok {
		return
	}
	reaction = strings.ToLower(strings.TrimSpace(reaction))
	if !isNewsReaction(reaction) {
		httpx.Fail(w, http.StatusUnprocessableEntity, "Неизвестная реакция")
		return
	}
	rows, err := h.Pool.Query(r.Context(), `
		SELECT nr.user_id,
			COALESCE(NULLIF(TRIM(CONCAT_WS(' ', u.surname, u.firstname, u.lastname)), ''), 'Сотрудник'),
			COALESCE(u.avatar_url, '')
		FROM public.news_reactions nr
		LEFT JOIN public.user_info u ON u.id = nr.user_id
		WHERE nr.news_id = $1 AND nr.reaction = $2
		ORDER BY nr.created_at DESC
		LIMIT 50`, newsID, reaction)
	if err != nil {
		log.Printf("news reactions: reactors news=%d reaction=%s: %v", newsID, reaction, err)
		if reactionsTableMissing(err) {
			httpx.Fail(w, http.StatusServiceUnavailable, "Реакции ещё не включены на сервере: нужна миграция V11__news_reactions.sql")
			return
		}
		httpx.Fail(w, http.StatusInternalServerError, "Ошибка подключения к БД")
		return
	}
	defer rows.Close()
	list := []map[string]any{}
	for rows.Next() {
		var id int64
		var name, avatar string
		if rows.Scan(&id, &name, &avatar) != nil {
			continue
		}
		list = append(list, map[string]any{"id": id, "name": name, "avatar_url": avatar})
	}
	httpx.OK(w, list, "OK")
}
