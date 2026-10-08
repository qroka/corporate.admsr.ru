package handlers

import (
	"context"
	"errors"
	"fmt"
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

// NewsComments — комментарии к новостям (/api/news_comments.php, V18). Два уровня,
// как во ВКонтакте: комментарий к новости и ответы в его ветке; ответ на ответ
// попадает в ту же ветку с пометкой «в ответ …». Популярность — число всех
// реакций; при равенстве выше более новый. Всё — только для вошедших (в киоске
// комментариев нет). Править — автор; удалять — автор, редактор раздела «Новости»
// или администратор. Личность — только из сессии.
type NewsComments struct {
	Pool *pgxpool.Pool
	Auth *auth.Service
}

const (
	newsCommentMaxRunes  = 2000
	newsCommentsPageSize = 20 // комментариев первого уровня за запрос на странице новости
	newsCommentsPageMax  = 50
)

func newsCommentsFailDB(w http.ResponseWriter, op string, err error) {
	log.Printf("news comments: %s: %v", op, err)
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && (pgErr.Code == "42P01" || pgErr.Code == "42703") {
		httpx.Fail(w, http.StatusServiceUnavailable, "Комментарии ещё не включены на сервере: нужна миграция V18__news_comments.sql")
		return
	}
	httpx.Fail(w, http.StatusInternalServerError, "Ошибка подключения к БД")
}

func (h *NewsComments) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	cur, ok := requireUser(w, r, h.Auth)
	if !ok {
		return
	}
	switch r.Method {
	case http.MethodGet:
		q := r.URL.Query()
		switch {
		case q.Get("action") == "replies":
			h.replies(w, r, cur)
		case q.Get("action") == "thread":
			h.thread(w, r, cur)
		case q.Get("action") == "reactors":
			h.reactors(w, r)
		default:
			h.list(w, r, cur)
		}
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

// ── Чтение ─────────────────────────────────────────────────────────────────────

// newsCommentFrom — комментарии с популярностью (score — все реакции) и числом
// ответов в ветке (replies). Подставляется внутрь подзапросов ниже.
const newsCommentFrom = `
	FROM public.news_comments c
	LEFT JOIN LATERAL (SELECT COUNT(*) AS n FROM public.news_comment_reactions r WHERE r.comment_id = c.id) sc ON true
	LEFT JOIN LATERAL (SELECT COUNT(*) AS n FROM public.news_comments x WHERE x.root_id = c.id) rc ON true`

// newsCommentSelect — внешний SELECT вокруг подзапроса %s, который отдаёт
// c.* + score + replies (+ что угодно ещё).
const newsCommentSelect = `
	SELECT c.id, c.news_id, c.root_id, c.reply_to_id, c.reply_to_user_id,
		COALESCE(NULLIF(TRIM(CONCAT_WS(' ', ru.firstname, ru.surname)), ''), ''),
		c.author_id,
		COALESCE(NULLIF(TRIM(CONCAT_WS(' ', u.firstname, u.surname)), ''), 'Сотрудник'),
		COALESCE(u.avatar_url, ''),
		c.content, c.created_at, c.updated_at, c.deleted_at IS NOT NULL, c.score, c.replies
	FROM (%s) c
	LEFT JOIN public.user_info u ON u.id = c.author_id
	LEFT JOIN public.user_info ru ON ru.id = c.reply_to_user_id`

// Видимые комментарии первого уровня: удалённый показывается только ради его ответов.
const newsCommentVisibleRoot = `c.root_id IS NULL AND (c.deleted_at IS NULL OR rc.n > 0)`

type newsCommentRow struct {
	ID            int64
	NewsID        int64
	RootID        *int64
	ReplyToID     *int64
	ReplyToUserID *int64
	ReplyToName   string
	AuthorID      int64
	AuthorName    string
	AuthorAvatar  string
	Content       string
	CreatedAt     time.Time
	UpdatedAt     *time.Time
	Deleted       bool
	Score         int64
	Replies       int64
}

// newsCommentsQuerier — общее у пула и транзакции: запросы ниже проверяются на
// настоящей БД и внутри транзакции с откатом.
type newsCommentsQuerier interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

// SQL-запросы чтения. Аргументы — в комментарии к каждому.

// sqlCommentByID: $1 — id.
func sqlCommentByID() string {
	return fmt.Sprintf(newsCommentSelect, `SELECT c.*, sc.n AS score, rc.n AS replies `+newsCommentFrom+` WHERE c.id = $1`)
}

// sqlTopReplies: $1 — id комментариев первого уровня.
func sqlTopReplies() string {
	return fmt.Sprintf(newsCommentSelect,
		`SELECT DISTINCT ON (c.root_id) c.*, sc.n AS score, rc.n AS replies `+newsCommentFrom+`
		WHERE c.root_id = ANY($1)
		ORDER BY c.root_id, sc.n DESC, c.created_at DESC, c.id DESC`)
}

// sqlCountRoots: $1 — id новости.
func sqlCountRoots() string {
	return `SELECT COUNT(*) ` + newsCommentFrom + ` WHERE c.news_id = $1 AND ` + newsCommentVisibleRoot
}

// sqlListRoots: $1 — id новости, $2 — limit, $3 — offset. sort: "new" или популярные.
func sqlListRoots(sort string) string {
	order := ` ORDER BY c.score DESC, c.created_at DESC, c.id DESC`
	if sort == "new" {
		order = ` ORDER BY c.created_at DESC, c.id DESC`
	}
	return fmt.Sprintf(newsCommentSelect+order+` LIMIT $2 OFFSET $3`,
		`SELECT c.*, sc.n AS score, rc.n AS replies `+newsCommentFrom+`
		WHERE c.news_id = $1 AND `+newsCommentVisibleRoot)
}

// sqlReplies: $1 — id комментария первого уровня; по времени.
func sqlReplies() string {
	return fmt.Sprintf(newsCommentSelect+` ORDER BY c.created_at, c.id`,
		`SELECT c.*, sc.n AS score, rc.n AS replies `+newsCommentFrom+` WHERE c.root_id = $1`)
}

// sqlDeleteEmptyDeletedRoot: $1 — id комментария первого уровня; удаляет его,
// если он мягко удалён и ответов в ветке не осталось.
const sqlDeleteEmptyDeletedRoot = `
	DELETE FROM public.news_comments p
	WHERE p.id = $1 AND p.deleted_at IS NOT NULL
	  AND NOT EXISTS (SELECT 1 FROM public.news_comments x WHERE x.root_id = p.id)`

func queryNewsComments(ctx context.Context, q newsCommentsQuerier, sql string, args ...any) ([]newsCommentRow, error) {
	rows, err := q.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []newsCommentRow{}
	for rows.Next() {
		var c newsCommentRow
		if err := rows.Scan(&c.ID, &c.NewsID, &c.RootID, &c.ReplyToID, &c.ReplyToUserID, &c.ReplyToName,
			&c.AuthorID, &c.AuthorName, &c.AuthorAvatar, &c.Content, &c.CreatedAt, &c.UpdatedAt,
			&c.Deleted, &c.Score, &c.Replies); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (h *NewsComments) query(ctx context.Context, sql string, args ...any) ([]newsCommentRow, error) {
	return queryNewsComments(ctx, h.Pool, sql, args...)
}

func (h *NewsComments) fetchOne(ctx context.Context, id int64) (newsCommentRow, error) {
	list, err := h.query(ctx, sqlCommentByID(), id)
	if err != nil {
		return newsCommentRow{}, err
	}
	if len(list) == 0 {
		return newsCommentRow{}, pgx.ErrNoRows
	}
	return list[0], nil
}

// topReplies — самый популярный ответ в каждой ветке.
func (h *NewsComments) topReplies(ctx context.Context, rootIDs []int64) (map[int64]newsCommentRow, error) {
	out := map[int64]newsCommentRow{}
	if len(rootIDs) == 0 {
		return out, nil
	}
	list, err := h.query(ctx, sqlTopReplies(), rootIDs)
	if err != nil {
		return nil, err
	}
	for _, c := range list {
		out[*c.RootID] = c
	}
	return out, nil
}

// commentViewer — кто смотрит: для canEdit / canDelete и «моих» реакций.
type commentViewer struct {
	user        *auth.User
	canModerate bool
}

func (h *NewsComments) viewer(ctx context.Context, cur *auth.User) commentViewer {
	return commentViewer{user: cur, canModerate: auth.CanEditSection(ctx, h.Pool, cur, "news")}
}

// format — комментарии в JSON; у комментариев первого уровня — topReply.
func (h *NewsComments) format(ctx context.Context, v commentViewer, list []newsCommentRow, top map[int64]newsCommentRow) []map[string]any {
	ids := make([]int64, 0, len(list)*2)
	for _, c := range list {
		ids = append(ids, c.ID)
		if t, ok := top[c.ID]; ok {
			ids = append(ids, t.ID)
		}
	}
	sums := h.reactionSummaries(ctx, ids, v.user.ID)
	out := make([]map[string]any, 0, len(list))
	for _, c := range list {
		item := h.formatOne(v, c, sums)
		if c.RootID == nil {
			if t, ok := top[c.ID]; ok {
				item["topReply"] = h.formatOne(v, t, sums)
			} else {
				item["topReply"] = nil
			}
		}
		out = append(out, item)
	}
	return out
}

func (h *NewsComments) formatOne(v commentViewer, c newsCommentRow, sums map[int64][]map[string]any) map[string]any {
	var updated any
	if c.UpdatedAt != nil {
		updated = c.UpdatedAt.Format(time.RFC3339)
	}
	// «В ответ …» — только когда ответили не на сам комментарий ветки, а на ответ в ней.
	var replyTo any
	if c.RootID != nil && c.ReplyToUserID != nil && (c.ReplyToID == nil || *c.ReplyToID != *c.RootID) {
		replyTo = map[string]any{"id": c.ReplyToID, "userId": *c.ReplyToUserID, "name": c.ReplyToName}
	}
	content := c.Content
	if c.Deleted {
		content = ""
	}
	reactions := sums[c.ID]
	if reactions == nil {
		reactions = []map[string]any{}
	}
	mine := c.AuthorID == v.user.ID
	return map[string]any{
		"id":      c.ID,
		"newsId":  c.NewsID,
		"rootId":  c.RootID,
		"replyTo": replyTo,
		"author": map[string]any{
			"id":         c.AuthorID,
			"name":       c.AuthorName,
			"avatar_url": c.AuthorAvatar,
		},
		"content":    content,
		"deleted":    c.Deleted,
		"createdAt":  c.CreatedAt.Format(time.RFC3339),
		"updatedAt":  updated,
		"reactions":  reactions,
		"score":      c.Score,
		"replyCount": c.Replies,
		"canEdit":    mine && !c.Deleted,
		"canDelete":  !c.Deleted && (mine || v.canModerate),
	}
}

// commentTotals — сколько всего (не удалённых) комментариев и ответов у новостей.
func (h *NewsComments) commentTotals(ctx context.Context, newsIDs []int64) (map[int64]int64, error) {
	out := map[int64]int64{}
	rows, err := h.Pool.Query(ctx, `
		SELECT news_id, COUNT(*) FROM public.news_comments
		WHERE news_id = ANY($1) AND deleted_at IS NULL
		GROUP BY news_id`, newsIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var id, n int64
		if err := rows.Scan(&id, &n); err != nil {
			return nil, err
		}
		out[id] = n
	}
	return out, rows.Err()
}

// list — страница новости: комментарии первого уровня, «Популярные» или «Новые».
func (h *NewsComments) list(w http.ResponseWriter, r *http.Request, cur *auth.User) {
	q := r.URL.Query()
	newsID, _ := strconv.ParseInt(q.Get("newsId"), 10, 64)
	if newsID <= 0 {
		httpx.Fail(w, http.StatusBadRequest, "Некорректный newsId")
		return
	}
	limit, _ := strconv.Atoi(q.Get("limit"))
	if limit <= 0 {
		limit = newsCommentsPageSize
	}
	if limit > newsCommentsPageMax {
		limit = newsCommentsPageMax
	}
	offset, _ := strconv.Atoi(q.Get("offset"))
	if offset < 0 {
		offset = 0
	}
	ctx := r.Context()
	var rootsTotal int64
	if err := h.Pool.QueryRow(ctx, sqlCountRoots(), newsID).Scan(&rootsTotal); err != nil {
		newsCommentsFailDB(w, "count roots", err)
		return
	}
	roots, err := h.query(ctx, sqlListRoots(q.Get("sort")), newsID, limit, offset)
	if err != nil {
		newsCommentsFailDB(w, "list", err)
		return
	}
	rootIDs := make([]int64, 0, len(roots))
	for _, c := range roots {
		rootIDs = append(rootIDs, c.ID)
	}
	top, err := h.topReplies(ctx, rootIDs)
	if err != nil {
		newsCommentsFailDB(w, "list replies", err)
		return
	}
	totals, err := h.commentTotals(ctx, []int64{newsID})
	if err != nil {
		newsCommentsFailDB(w, "list totals", err)
		return
	}
	httpx.OK(w, map[string]any{
		"total":      totals[newsID],
		"rootsTotal": rootsTotal,
		"items":      h.format(ctx, h.viewer(ctx, cur), roots, top),
	}, "OK")
}

func (h *NewsComments) loadReplies(ctx context.Context, rootID int64) ([]newsCommentRow, error) {
	return h.query(ctx, sqlReplies(), rootID)
}

// replies — вся ветка по «Развернуть», по времени: так читается разговор.
func (h *NewsComments) replies(w http.ResponseWriter, r *http.Request, cur *auth.User) {
	rootID, _ := strconv.ParseInt(r.URL.Query().Get("rootId"), 10, 64)
	if rootID <= 0 {
		httpx.Fail(w, http.StatusBadRequest, "Некорректный rootId")
		return
	}
	list, err := h.loadReplies(r.Context(), rootID)
	if err != nil {
		newsCommentsFailDB(w, "replies", err)
		return
	}
	httpx.OK(w, h.format(r.Context(), h.viewer(r.Context(), cur), list, nil), "OK")
}

// thread — ветка, в которой лежит комментарий id (переход из уведомления):
// комментарий первого уровня и все ответы.
func (h *NewsComments) thread(w http.ResponseWriter, r *http.Request, cur *auth.User) {
	id, _ := strconv.ParseInt(r.URL.Query().Get("id"), 10, 64)
	if id <= 0 {
		httpx.Fail(w, http.StatusBadRequest, "Некорректный id")
		return
	}
	ctx := r.Context()
	c, err := h.fetchOne(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			httpx.Fail(w, http.StatusNotFound, "Комментарий удалён")
			return
		}
		newsCommentsFailDB(w, "thread", err)
		return
	}
	root := c
	if c.RootID != nil {
		if root, err = h.fetchOne(ctx, *c.RootID); err != nil {
			newsCommentsFailDB(w, "thread root", err)
			return
		}
	}
	replies, err := h.loadReplies(ctx, root.ID)
	if err != nil {
		newsCommentsFailDB(w, "thread replies", err)
		return
	}
	top, err := h.topReplies(ctx, []int64{root.ID})
	if err != nil {
		newsCommentsFailDB(w, "thread top", err)
		return
	}
	v := h.viewer(ctx, cur)
	httpx.OK(w, map[string]any{
		"root":    h.format(ctx, v, []newsCommentRow{root}, top)[0],
		"replies": h.format(ctx, v, replies, nil),
	}, "OK")
}

// ── Запись ─────────────────────────────────────────────────────────────────────

func normalizeCommentContent(v any) (string, string) {
	s := strings.TrimSpace(strings.ReplaceAll(strVal(v), "\r\n", "\n"))
	if s == "" {
		return "", "Комментарий пустой"
	}
	if utf8.RuneCountInString(s) > newsCommentMaxRunes {
		return "", "Комментарий длиннее " + strconv.Itoa(newsCommentMaxRunes) + " символов"
	}
	return s, ""
}

func (h *NewsComments) loadComment(w http.ResponseWriter, r *http.Request, id int64) (newsCommentRow, bool) {
	if id <= 0 {
		httpx.Fail(w, http.StatusBadRequest, "Некорректный id")
		return newsCommentRow{}, false
	}
	c, err := h.fetchOne(r.Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			httpx.Fail(w, http.StatusNotFound, "Комментарий не найден")
			return newsCommentRow{}, false
		}
		newsCommentsFailDB(w, "fetch", err)
		return newsCommentRow{}, false
	}
	return c, true
}

func (h *NewsComments) respondOne(w http.ResponseWriter, r *http.Request, cur *auth.User, id int64, msg string) {
	ctx := r.Context()
	c, err := h.fetchOne(ctx, id)
	if err != nil {
		newsCommentsFailDB(w, "fetch saved", err)
		return
	}
	var top map[int64]newsCommentRow
	if c.RootID == nil {
		if top, err = h.topReplies(ctx, []int64{c.ID}); err != nil {
			newsCommentsFailDB(w, "fetch saved top", err)
			return
		}
	}
	httpx.OK(w, h.format(ctx, h.viewer(ctx, cur), []newsCommentRow{c}, top)[0], msg)
}

func (h *NewsComments) create(w http.ResponseWriter, r *http.Request, cur *auth.User, body map[string]any) {
	ctx := r.Context()
	content, msg := normalizeCommentContent(body["content"])
	if msg != "" {
		httpx.Fail(w, http.StatusUnprocessableEntity, msg)
		return
	}
	newsID := int64Of(body["newsId"])
	var rootID, replyToID, replyToUser *int64
	if rid := int64Of(body["replyToId"]); rid > 0 {
		target, ok := h.loadComment(w, r, rid)
		if !ok {
			return
		}
		if target.Deleted {
			httpx.Fail(w, http.StatusUnprocessableEntity, "Комментарий удалён — ответить нельзя")
			return
		}
		newsID = target.NewsID
		root := target.ID
		if target.RootID != nil {
			root = *target.RootID
		}
		author := target.AuthorID
		rootID, replyToID, replyToUser = &root, &target.ID, &author
	}
	if newsID <= 0 {
		httpx.Fail(w, http.StatusBadRequest, "Некорректный newsId")
		return
	}
	var exists bool
	if err := h.Pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM public.news WHERE id = $1)`, newsID).Scan(&exists); err != nil {
		newsCommentsFailDB(w, "news check", err)
		return
	}
	if !exists {
		httpx.Fail(w, http.StatusNotFound, "Новость не найдена")
		return
	}
	var id int64
	if err := h.Pool.QueryRow(ctx, `
		INSERT INTO public.news_comments (news_id, root_id, reply_to_id, reply_to_user_id, author_id, content)
		VALUES ($1, $2, $3, $4, $5, $6) RETURNING id`,
		newsID, rootID, replyToID, replyToUser, cur.ID, content).Scan(&id); err != nil {
		newsCommentsFailDB(w, "create", err)
		return
	}
	if replyToUser != nil && *replyToUser != cur.ID {
		if _, err := h.Pool.Exec(ctx, `
			INSERT INTO public.notifications (user_id, actor_id, kind, comment_id)
			VALUES ($1, $2, $3, $4)`, *replyToUser, cur.ID, notifyKindCommentReply, id); err != nil {
			log.Printf("notifications: insert %s for user %d: %v", notifyKindCommentReply, *replyToUser, err)
		}
	}
	h.respondOne(w, r, cur, id, "Комментарий опубликован")
}

func (h *NewsComments) update(w http.ResponseWriter, r *http.Request, cur *auth.User, body map[string]any) {
	c, ok := h.loadComment(w, r, int64Of(body["id"]))
	if !ok {
		return
	}
	if c.AuthorID != cur.ID || c.Deleted {
		httpx.Fail(w, http.StatusForbidden, "Править комментарий может только его автор")
		return
	}
	content, msg := normalizeCommentContent(body["content"])
	if msg != "" {
		httpx.Fail(w, http.StatusUnprocessableEntity, msg)
		return
	}
	if _, err := h.Pool.Exec(r.Context(),
		`UPDATE public.news_comments SET content = $1, updated_at = now() WHERE id = $2`, content, c.ID); err != nil {
		newsCommentsFailDB(w, "update", err)
		return
	}
	h.respondOne(w, r, cur, c.ID, "Комментарий изменён")
}

// delete — комментарий с ответами удаляется мягко («Комментарий удалён», ветка
// остаётся), без ответов и ответы — физически. Удалённый ответ был последним в
// ветке мягко удалённого комментария — уходит и он.
func (h *NewsComments) delete(w http.ResponseWriter, r *http.Request, cur *auth.User, body map[string]any) {
	ctx := r.Context()
	c, ok := h.loadComment(w, r, int64Of(body["id"]))
	if !ok {
		return
	}
	if c.Deleted {
		httpx.Fail(w, http.StatusNotFound, "Комментарий уже удалён")
		return
	}
	if c.AuthorID != cur.ID && !auth.CanEditSection(ctx, h.Pool, cur, "news") {
		httpx.Fail(w, http.StatusForbidden, "Удалить комментарий может автор или редактор новостей")
		return
	}
	var err error
	switch {
	case c.RootID == nil && c.Replies > 0:
		_, err = h.Pool.Exec(ctx, `UPDATE public.news_comments SET deleted_at = now(), content = '' WHERE id = $1`, c.ID)
	default:
		_, err = h.Pool.Exec(ctx, `DELETE FROM public.news_comments WHERE id = $1`, c.ID)
		if err == nil && c.RootID != nil {
			_, err = h.Pool.Exec(ctx, sqlDeleteEmptyDeletedRoot, *c.RootID)
		}
	}
	if err != nil {
		newsCommentsFailDB(w, "delete", err)
		return
	}
	httpx.OK(w, map[string]any{"id": c.ID, "rootId": c.RootID, "soft": c.RootID == nil && c.Replies > 0}, "Комментарий удалён")
}

// ── Реакции ────────────────────────────────────────────────────────────────────

// reactionSummaries — для каждого комментария {key, count, mine} в порядке
// NewsReactionKeys (ключи общие с новостями); нулевые не попадают.
func (h *NewsComments) reactionSummaries(ctx context.Context, ids []int64, viewer int64) map[int64][]map[string]any {
	type agg struct {
		count int64
		mine  bool
	}
	by := map[int64]map[string]*agg{}
	for _, id := range ids {
		by[id] = map[string]*agg{}
	}
	if len(ids) > 0 {
		rows, err := h.Pool.Query(ctx, `
			SELECT comment_id, reaction, COUNT(*)::bigint, BOOL_OR(user_id = $2)
			FROM public.news_comment_reactions
			WHERE comment_id = ANY($1)
			GROUP BY comment_id, reaction`, ids, viewer)
		if err != nil {
			log.Printf("news comments: reactions summary: %v", err)
		} else {
			for rows.Next() {
				var id, count int64
				var key string
				var mine bool
				if rows.Scan(&id, &key, &count, &mine) != nil {
					continue
				}
				if m, ok := by[id]; ok && isNewsReaction(key) {
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
			if a := by[id][key]; a != nil && a.count > 0 {
				list = append(list, map[string]any{"key": key, "count": a.count, "mine": a.mine})
			}
		}
		out[id] = list
	}
	return out
}

func (h *NewsComments) react(w http.ResponseWriter, r *http.Request, cur *auth.User, body map[string]any) {
	c, ok := h.loadComment(w, r, int64Of(body["id"]))
	if !ok {
		return
	}
	if c.Deleted {
		httpx.Fail(w, http.StatusUnprocessableEntity, "Комментарий удалён")
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
			INSERT INTO public.news_comment_reactions (comment_id, user_id, reaction)
			VALUES ($1, $2, $3)
			ON CONFLICT (comment_id, user_id, reaction) DO NOTHING`, c.ID, cur.ID, reaction)
	} else {
		_, err = h.Pool.Exec(r.Context(), `
			DELETE FROM public.news_comment_reactions
			WHERE comment_id = $1 AND user_id = $2 AND reaction = $3`, c.ID, cur.ID, reaction)
	}
	if err != nil {
		newsCommentsFailDB(w, "react", err)
		return
	}
	httpx.OK(w, map[string]any{
		"id":        c.ID,
		"reactions": h.reactionSummaries(r.Context(), []int64{c.ID}, cur.ID)[c.ID],
	}, "Реакция сохранена")
}

func (h *NewsComments) reactors(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	id, _ := strconv.ParseInt(q.Get("id"), 10, 64)
	reaction := strings.ToLower(strings.TrimSpace(q.Get("reaction")))
	if id <= 0 || !isNewsReaction(reaction) {
		httpx.Fail(w, http.StatusBadRequest, "Некорректный запрос")
		return
	}
	rows, err := h.Pool.Query(r.Context(), `
		SELECT cr.user_id,
			COALESCE(NULLIF(TRIM(CONCAT_WS(' ', u.surname, u.firstname, u.lastname)), ''), 'Сотрудник'),
			COALESCE(u.avatar_url, '')
		FROM public.news_comment_reactions cr
		LEFT JOIN public.user_info u ON u.id = cr.user_id
		WHERE cr.comment_id = $1 AND cr.reaction = $2
		ORDER BY cr.created_at DESC
		LIMIT 50`, id, reaction)
	if err != nil {
		newsCommentsFailDB(w, "reactors", err)
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
