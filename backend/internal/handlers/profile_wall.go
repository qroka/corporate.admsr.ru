package handlers

import (
	"context"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"corporate.admsr.ru/backend/internal/auth"
	"corporate.admsr.ru/backend/internal/httpx"
)

// ProfileWall — стена профиля (/api/profile_wall.php). Писать на стену может
// любой вошедший сотрудник; править — только автор; удалять — автор, владелец
// стены или администратор. Личность — только из сессии.
type ProfileWall struct {
	Pool *pgxpool.Pool
	Auth *auth.Service
}

const (
	wallPostMaxRunes = 4000
	wallPageDefault  = 10
	wallPageMax      = 50
)

func wallTableMissing(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "42P01" // undefined_table
}

func wallFailDB(w http.ResponseWriter, op string, err error) {
	log.Printf("profile wall: %s: %v", op, err)
	if wallTableMissing(err) {
		httpx.Fail(w, http.StatusServiceUnavailable, "Стена ещё не включена на сервере: нужна миграция V14__profile_wall.sql")
		return
	}
	httpx.Fail(w, http.StatusInternalServerError, "Ошибка подключения к БД")
}

func (h *ProfileWall) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	cur, ok := requireUser(w, r, h.Auth)
	if !ok {
		return
	}
	switch r.Method {
	case http.MethodGet:
		if r.URL.Query().Get("action") == "reactors" {
			h.reactors(w, r)
			return
		}
		h.list(w, r, cur)
	case http.MethodPost:
		var body map[string]any
		if err := httpx.DecodeJSON(r, &body); err != nil {
			httpx.Fail(w, http.StatusBadRequest, "Некорректный JSON")
			return
		}
		switch strVal(body["action"]) {
		case "create":
			h.create(w, r, cur, body)
		case "update":
			h.update(w, r, cur, body)
		case "delete":
			h.delete(w, r, cur, body)
		case "react":
			h.react(w, r, cur, body)
		default:
			httpx.Fail(w, http.StatusBadRequest, "Неизвестное действие")
		}
	default:
		httpx.MethodNotAllowed(w)
	}
}

// normalizeWallContent — простой текст: без HTML-разметки на сервере не храним
// ничего, что браузер мог бы исполнить; фронт выводит текст как текст.
func normalizeWallContent(v any) (string, string) {
	s := strings.ReplaceAll(strVal(v), "\r\n", "\n")
	s = strings.TrimSpace(s)
	if s == "" {
		return "", "Запись пустая"
	}
	if utf8.RuneCountInString(s) > wallPostMaxRunes {
		return "", "Запись длиннее " + strconv.Itoa(wallPostMaxRunes) + " символов"
	}
	return s, ""
}

func int64Of(v any) int64 {
	switch x := v.(type) {
	case float64:
		return int64(x)
	case int64:
		return x
	case string:
		n, _ := strconv.ParseInt(strings.TrimSpace(x), 10, 64)
		return n
	}
	return 0
}

type wallPostRow struct {
	ID           int64
	OwnerID      int64
	AuthorID     int64
	AuthorName   string
	AuthorAvatar string
	Content      string
	CreatedAt    time.Time
	UpdatedAt    *time.Time
}

const wallPostSelect = `
	SELECT p.id, p.owner_id, p.author_id,
		COALESCE(NULLIF(TRIM(CONCAT_WS(' ', u.firstname, u.surname)), ''), 'Сотрудник'),
		COALESCE(u.avatar_url, ''),
		p.content, p.created_at, p.updated_at
	FROM public.wall_posts p
	LEFT JOIN public.user_info u ON u.id = p.author_id`

func scanWallPost(row pgx.Row) (wallPostRow, error) {
	var p wallPostRow
	err := row.Scan(&p.ID, &p.OwnerID, &p.AuthorID, &p.AuthorName, &p.AuthorAvatar, &p.Content, &p.CreatedAt, &p.UpdatedAt)
	return p, err
}

func (h *ProfileWall) fetchPost(ctx context.Context, id int64) (wallPostRow, error) {
	return scanWallPost(h.Pool.QueryRow(ctx, wallPostSelect+` WHERE p.id = $1`, id))
}

func (h *ProfileWall) fmtPosts(ctx context.Context, cur *auth.User, posts []wallPostRow) []map[string]any {
	ids := make([]int64, 0, len(posts))
	for _, p := range posts {
		ids = append(ids, p.ID)
	}
	sums := h.reactionSummaries(ctx, ids, cur.ID)
	out := make([]map[string]any, 0, len(posts))
	for _, p := range posts {
		var updated any
		if p.UpdatedAt != nil {
			updated = p.UpdatedAt.Format(time.RFC3339)
		}
		out = append(out, map[string]any{
			"id":      p.ID,
			"ownerId": p.OwnerID,
			"author": map[string]any{
				"id":         p.AuthorID,
				"name":       p.AuthorName,
				"avatar_url": p.AuthorAvatar,
			},
			"content":   p.Content,
			"createdAt": p.CreatedAt.Format(time.RFC3339),
			"updatedAt": updated,
			"reactions": sums[p.ID],
			"canEdit":   p.AuthorID == cur.ID,
			"canDelete": p.AuthorID == cur.ID || p.OwnerID == cur.ID || auth.IsAdmin(cur),
		})
	}
	return out
}

func (h *ProfileWall) list(w http.ResponseWriter, r *http.Request, cur *auth.User) {
	q := r.URL.Query()
	ownerID, _ := strconv.ParseInt(q.Get("userId"), 10, 64)
	if ownerID <= 0 {
		httpx.Fail(w, http.StatusBadRequest, "Некорректный userId")
		return
	}
	limit, _ := strconv.Atoi(q.Get("limit"))
	if limit <= 0 {
		limit = wallPageDefault
	}
	if limit > wallPageMax {
		limit = wallPageMax
	}
	offset, _ := strconv.Atoi(q.Get("offset"))
	if offset < 0 {
		offset = 0
	}

	var total int64
	if err := h.Pool.QueryRow(r.Context(),
		`SELECT COUNT(*) FROM public.wall_posts WHERE owner_id = $1`, ownerID).Scan(&total); err != nil {
		wallFailDB(w, "count", err)
		return
	}
	rows, err := h.Pool.Query(r.Context(),
		wallPostSelect+` WHERE p.owner_id = $1 ORDER BY p.created_at DESC, p.id DESC LIMIT $2 OFFSET $3`,
		ownerID, limit, offset)
	if err != nil {
		wallFailDB(w, "list", err)
		return
	}
	posts := []wallPostRow{}
	for rows.Next() {
		p, err := scanWallPost(rows)
		if err != nil {
			rows.Close()
			wallFailDB(w, "scan", err)
			return
		}
		posts = append(posts, p)
	}
	rows.Close()
	httpx.OK(w, map[string]any{"total": total, "items": h.fmtPosts(r.Context(), cur, posts)}, "OK")
}

func (h *ProfileWall) create(w http.ResponseWriter, r *http.Request, cur *auth.User, body map[string]any) {
	ownerID := int64Of(body["userId"])
	if ownerID <= 0 {
		httpx.Fail(w, http.StatusBadRequest, "Некорректный userId")
		return
	}
	content, msg := normalizeWallContent(body["content"])
	if msg != "" {
		httpx.Fail(w, http.StatusUnprocessableEntity, msg)
		return
	}
	var exists bool
	if err := h.Pool.QueryRow(r.Context(),
		`SELECT EXISTS (SELECT 1 FROM public.user_info WHERE id = $1)`, ownerID).Scan(&exists); err != nil {
		wallFailDB(w, "owner check", err)
		return
	}
	if !exists {
		httpx.Fail(w, http.StatusNotFound, "Сотрудник не найден")
		return
	}
	var id int64
	if err := h.Pool.QueryRow(r.Context(), `
		INSERT INTO public.wall_posts (owner_id, author_id, content)
		VALUES ($1, $2, $3) RETURNING id`, ownerID, cur.ID, content).Scan(&id); err != nil {
		wallFailDB(w, "create", err)
		return
	}
	p, err := h.fetchPost(r.Context(), id)
	if err != nil {
		wallFailDB(w, "fetch created", err)
		return
	}
	httpx.OK(w, h.fmtPosts(r.Context(), cur, []wallPostRow{p})[0], "Запись опубликована")
}

// loadPost — запись по id из тела; 404, если её нет.
func (h *ProfileWall) loadPost(w http.ResponseWriter, r *http.Request, body map[string]any) (wallPostRow, bool) {
	id := int64Of(body["id"])
	if id <= 0 {
		httpx.Fail(w, http.StatusBadRequest, "Некорректный id")
		return wallPostRow{}, false
	}
	p, err := h.fetchPost(r.Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			httpx.Fail(w, http.StatusNotFound, "Запись не найдена")
			return wallPostRow{}, false
		}
		wallFailDB(w, "fetch", err)
		return wallPostRow{}, false
	}
	return p, true
}

func (h *ProfileWall) update(w http.ResponseWriter, r *http.Request, cur *auth.User, body map[string]any) {
	p, ok := h.loadPost(w, r, body)
	if !ok {
		return
	}
	if p.AuthorID != cur.ID {
		httpx.Fail(w, http.StatusForbidden, "Править запись может только её автор")
		return
	}
	content, msg := normalizeWallContent(body["content"])
	if msg != "" {
		httpx.Fail(w, http.StatusUnprocessableEntity, msg)
		return
	}
	if _, err := h.Pool.Exec(r.Context(),
		`UPDATE public.wall_posts SET content = $1, updated_at = now() WHERE id = $2`, content, p.ID); err != nil {
		wallFailDB(w, "update", err)
		return
	}
	p, err := h.fetchPost(r.Context(), p.ID)
	if err != nil {
		wallFailDB(w, "fetch updated", err)
		return
	}
	httpx.OK(w, h.fmtPosts(r.Context(), cur, []wallPostRow{p})[0], "Запись изменена")
}

func (h *ProfileWall) delete(w http.ResponseWriter, r *http.Request, cur *auth.User, body map[string]any) {
	p, ok := h.loadPost(w, r, body)
	if !ok {
		return
	}
	if p.AuthorID != cur.ID && p.OwnerID != cur.ID && !auth.IsAdmin(cur) {
		httpx.Fail(w, http.StatusForbidden, "Удалить запись может автор или владелец стены")
		return
	}
	if _, err := h.Pool.Exec(r.Context(), `DELETE FROM public.wall_posts WHERE id = $1`, p.ID); err != nil {
		wallFailDB(w, "delete", err)
		return
	}
	httpx.OK(w, map[string]any{"id": p.ID}, "Запись удалена")
}

// reactionSummaries — для каждой записи {key, count, mine} в порядке
// NewsReactionKeys (ключи общие с новостями); нулевые не попадают.
func (h *ProfileWall) reactionSummaries(ctx context.Context, ids []int64, viewer int64) map[int64][]map[string]any {
	type agg struct {
		count int64
		mine  bool
	}
	byPost := map[int64]map[string]*agg{}
	for _, id := range ids {
		byPost[id] = map[string]*agg{}
	}
	if len(ids) > 0 {
		rows, err := h.Pool.Query(ctx, `
			SELECT post_id, reaction, COUNT(*)::bigint, BOOL_OR(user_id = $2)
			FROM public.wall_post_reactions
			WHERE post_id = ANY($1)
			GROUP BY post_id, reaction`, ids, viewer)
		if err != nil {
			log.Printf("profile wall: reactions summary: %v", err)
		} else {
			for rows.Next() {
				var postID, count int64
				var key string
				var mine bool
				if rows.Scan(&postID, &key, &count, &mine) != nil {
					continue
				}
				if m, ok := byPost[postID]; ok && isNewsReaction(key) {
					m[key] = &agg{count: count, mine: mine}
				}
			}
			rows.Close()
		}
	}
	out := map[int64][]map[string]any{}
	for _, id := range ids {
		list := []map[string]any{}
		for _, key := range NewsReactionKeys {
			if a := byPost[id][key]; a != nil && a.count > 0 {
				list = append(list, map[string]any{"key": key, "count": a.count, "mine": a.mine})
			}
		}
		out[id] = list
	}
	return out
}

func (h *ProfileWall) react(w http.ResponseWriter, r *http.Request, cur *auth.User, body map[string]any) {
	p, ok := h.loadPost(w, r, body)
	if !ok {
		return
	}
	reaction := strings.ToLower(strings.TrimSpace(strVal(body["reaction"])))
	if !isNewsReaction(reaction) {
		httpx.Fail(w, http.StatusUnprocessableEntity, "Неизвестная реакция")
		return
	}
	active, _ := body["active"].(bool)
	var err error
	if active {
		_, err = h.Pool.Exec(r.Context(), `
			INSERT INTO public.wall_post_reactions (post_id, user_id, reaction)
			VALUES ($1, $2, $3)
			ON CONFLICT (post_id, user_id, reaction) DO NOTHING`, p.ID, cur.ID, reaction)
	} else {
		_, err = h.Pool.Exec(r.Context(), `
			DELETE FROM public.wall_post_reactions
			WHERE post_id = $1 AND user_id = $2 AND reaction = $3`, p.ID, cur.ID, reaction)
	}
	if err != nil {
		wallFailDB(w, "react", err)
		return
	}
	httpx.OK(w, map[string]any{
		"id":        p.ID,
		"reactions": h.reactionSummaries(r.Context(), []int64{p.ID}, cur.ID)[p.ID],
	}, "Реакция сохранена")
}

func (h *ProfileWall) reactors(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	id, _ := strconv.ParseInt(q.Get("id"), 10, 64)
	reaction := strings.ToLower(strings.TrimSpace(q.Get("reaction")))
	if id <= 0 || !isNewsReaction(reaction) {
		httpx.Fail(w, http.StatusBadRequest, "Некорректный запрос")
		return
	}
	rows, err := h.Pool.Query(r.Context(), `
		SELECT wr.user_id,
			COALESCE(NULLIF(TRIM(CONCAT_WS(' ', u.surname, u.firstname, u.lastname)), ''), 'Сотрудник'),
			COALESCE(u.avatar_url, '')
		FROM public.wall_post_reactions wr
		LEFT JOIN public.user_info u ON u.id = wr.user_id
		WHERE wr.post_id = $1 AND wr.reaction = $2
		ORDER BY wr.created_at DESC
		LIMIT 50`, id, reaction)
	if err != nil {
		wallFailDB(w, "reactors", err)
		return
	}
	defer rows.Close()
	list := []map[string]any{}
	for rows.Next() {
		var uid int64
		var name, avatar string
		if rows.Scan(&uid, &name, &avatar) != nil {
			continue
		}
		list = append(list, map[string]any{"id": uid, "name": name, "avatar_url": avatar})
	}
	httpx.OK(w, list, "OK")
}
