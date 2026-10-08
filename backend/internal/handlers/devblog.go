package handlers

import (
	"context"
	"errors"
	"log"
	"net/http"
	"regexp"
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

// Devblog — заметки об обновлениях портала (/api/devblog.php, V20).
//
// Администраторы пишут один общий черновик в Markdown. Сохранение — с номером
// версии: если черновик за это время сохранил другой, ответ 409 с его версией
// (без молчаливой перезаписи). Публикация создаёт обычную новость категории
// «Девблог» (лента, реакции, комментарии — как у новостей), уведомление
// kind = 'devblog' всем активным сотрудникам и окошко при первом заходе:
// последний опубликованный девблог, который сотрудник ещё не закрыл.
//
// HTML новости строит фронт из Markdown (редактор Nuxt UI): разбирает его в свою
// схему документа, так что произвольный HTML из текста в новость не попадает.
type Devblog struct {
	Pool *pgxpool.Pool
	Auth *auth.Service
}

const (
	devblogCategory     = "Девблог"
	devblogDefaultCover = "/devblog-cover.svg" // public/devblog-cover.svg; public/img/ — загрузки, вне git
	devblogTitleMax     = 200
	devblogBodyMax      = 50000
	devblogHTMLMax      = 400000
)

func devblogFailDB(w http.ResponseWriter, op string, err error) {
	log.Printf("devblog: %s: %v", op, err)
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && (pgErr.Code == "42P01" || pgErr.Code == "42703") {
		httpx.Fail(w, http.StatusServiceUnavailable, "Девблог ещё не включён на сервере: нужна миграция V20__devblog.sql")
		return
	}
	httpx.Fail(w, http.StatusInternalServerError, "Ошибка подключения к БД")
}

func (h *Devblog) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		if r.URL.Query().Get("action") == "draft" {
			cur, ok := requireAdmin(w, r, h.Auth)
			if !ok {
				return
			}
			h.getDraft(w, r, cur)
			return
		}
		cur, ok := requireUser(w, r, h.Auth)
		if !ok {
			return
		}
		h.pending(w, r, cur)
	case http.MethodPost:
		var body map[string]any
		if err := httpx.DecodeJSON(r, &body); err != nil {
			httpx.Fail(w, http.StatusBadRequest, "Некорректный JSON")
			return
		}
		switch strVal(body["action"]) {
		case "save":
			cur, ok := requireAdmin(w, r, h.Auth)
			if !ok {
				return
			}
			h.save(w, r, cur, body)
		case "publish":
			cur, ok := requireAdmin(w, r, h.Auth)
			if !ok {
				return
			}
			h.publish(w, r, cur, body)
		case "dismiss":
			cur, ok := requireUser(w, r, h.Auth)
			if !ok {
				return
			}
			h.dismiss(w, r, cur, body)
		default:
			httpx.Fail(w, http.StatusBadRequest, "Неизвестное действие")
		}
	default:
		httpx.MethodNotAllowed(w)
	}
}

// ── Черновик ───────────────────────────────────────────────────────────────────

// devblogVersionRe — версия выпуска «X.Y.Z» (V21).
var devblogVersionRe = regexp.MustCompile(`^(\d{1,4})\.(\d{1,4})\.(\d{1,4})$`)

// nextDevblogVersion — предложение для нового выпуска: последняя опубликованная + 0.0.1;
// выпусков ещё не было (или версия не в формате X.Y.Z) — 1.0.0.
func nextDevblogVersion(last string) string {
	m := devblogVersionRe.FindStringSubmatch(strings.TrimSpace(last))
	if m == nil {
		return "1.0.0"
	}
	patch, _ := strconv.Atoi(m[3])
	return m[1] + "." + m[2] + "." + strconv.Itoa(patch+1)
}

// sqlDevblogLastVersion — версия последнего опубликованного девблога.
const sqlDevblogLastVersion = `
	SELECT release_version FROM public.devblog_posts ORDER BY published_at DESC, news_id DESC LIMIT 1`

const sqlDevblogDraft = `
	SELECT d.title, d.body_md, COALESCE(d.image_path, ''), d.release_version, d.version, d.updated_at, d.updated_by,
		COALESCE(NULLIF(TRIM(CONCAT_WS(' ', u.firstname, u.surname)), ''), 'Администратор')
	FROM public.devblog_draft d
	LEFT JOIN public.user_info u ON u.id = d.updated_by
	WHERE d.id = 1`

// SQL записи — константами, чтобы их можно было проверить на БД в транзакции с откатом.

// sqlDevblogSave: $1 title, $2 body, $3 image, $4 автор, $5 версия черновика, с которой начали правку,
// $6 force, $7 версия выпуска.
const sqlDevblogSave = `
	UPDATE public.devblog_draft
	SET title = $1, body_md = $2, image_path = NULLIF($3, ''), release_version = $7, version = version + 1,
	    updated_by = $4, updated_at = now()
	WHERE id = 1 AND (version = $5 OR $6)`

// sqlDevblogInsertNews: $1 title, $2 категория, $3 html, $4 дата, $5 обложка.
const sqlDevblogInsertNews = `
	INSERT INTO public.news (title, category, description, date, image_path)
	VALUES ($1, $2, $3, $4, $5) RETURNING id`

// sqlDevblogInsertPost: $1 новость, $2 Markdown, $3 автор, $4 версия выпуска.
const sqlDevblogInsertPost = `
	INSERT INTO public.devblog_posts (news_id, body_md, published_by, release_version) VALUES ($1, $2, $3, $4)`

// sqlDevblogDismiss: $1 новость, $2 сотрудник.
const sqlDevblogDismiss = `
	INSERT INTO public.devblog_dismissed (news_id, user_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`

// sqlDevblogNotifyAll: $1 автор (ему не шлём), $2 вид, $3 новость. Всем активным сотрудникам.
const sqlDevblogNotifyAll = `
	INSERT INTO public.notifications (user_id, actor_id, kind, news_id)
	SELECT u.id, $1::bigint, $2, $3::bigint FROM public.user_info u
	WHERE u.status = true AND u.id <> $1::bigint`

// sqlDevblogResetDraft: $1 автор. После публикации черновик пустой.
const sqlDevblogResetDraft = `
	UPDATE public.devblog_draft
	SET title = '', body_md = '', image_path = NULL, release_version = '', version = version + 1,
	    updated_by = $1, updated_at = now()
	WHERE id = 1`

// sqlDevblogMarkRead: $1 сотрудник, $2 вид, $3 новость.
const sqlDevblogMarkRead = `
	UPDATE public.notifications SET read_at = now()
	WHERE user_id = $1 AND kind = $2 AND news_id = $3 AND read_at IS NULL`

func (h *Devblog) loadDraft(ctx context.Context) (map[string]any, error) {
	var last string
	if err := h.Pool.QueryRow(ctx, sqlDevblogLastVersion).Scan(&last); err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}
	suggested := nextDevblogVersion(last)
	var (
		title, body, image, release, byName string
		version                             int64
		updatedAt                           *time.Time
		updatedBy                           *int64
	)
	err := h.Pool.QueryRow(ctx, sqlDevblogDraft).Scan(&title, &body, &image, &release, &version, &updatedAt, &updatedBy, &byName)
	if errors.Is(err, pgx.ErrNoRows) {
		// Строку создаёт V20; если её удалили руками — начинаем с пустого.
		if _, err := h.Pool.Exec(ctx, `INSERT INTO public.devblog_draft (id) VALUES (1) ON CONFLICT (id) DO NOTHING`); err != nil {
			return nil, err
		}
		return map[string]any{"title": "", "body": "", "imagePath": "", "releaseVersion": "", "suggestedVersion": suggested,
			"version": 0, "updatedAt": nil, "updatedBy": nil}, nil
	}
	if err != nil {
		return nil, err
	}
	out := map[string]any{"title": title, "body": body, "imagePath": image, "releaseVersion": release,
		"suggestedVersion": suggested, "version": version, "updatedAt": nil, "updatedBy": nil}
	if updatedAt != nil {
		out["updatedAt"] = updatedAt.Format(time.RFC3339)
	}
	if updatedBy != nil {
		out["updatedBy"] = map[string]any{"id": *updatedBy, "name": byName}
	}
	return out, nil
}

func (h *Devblog) getDraft(w http.ResponseWriter, r *http.Request, _ *auth.User) {
	d, err := h.loadDraft(r.Context())
	if err != nil {
		devblogFailDB(w, "draft", err)
		return
	}
	httpx.OK(w, d, "OK")
}

// devblogFields — поля черновика из тела запроса.
type devblogFields struct {
	Title, Body, Image, Release string
}

// devblogInput — поля черновика из тела; problem != "" — что не так. Версия выпуска
// в черновике может быть пустой (подставится предложенная), но если есть — «X.Y.Z».
func devblogInput(body map[string]any) (f devblogFields, problem string) {
	f.Title = strings.TrimSpace(strVal(body["title"]))
	f.Body = strings.ReplaceAll(strVal(body["body"]), "\r\n", "\n")
	f.Image = strings.TrimSpace(strVal(body["imagePath"]))
	f.Release = strings.TrimSpace(strVal(body["releaseVersion"]))
	switch {
	case utf8.RuneCountInString(f.Title) > devblogTitleMax:
		problem = "Заголовок длиннее " + strconv.Itoa(devblogTitleMax) + " символов"
	case utf8.RuneCountInString(f.Body) > devblogBodyMax:
		problem = "Текст длиннее " + strconv.Itoa(devblogBodyMax) + " символов"
	case f.Image != "" && !strings.HasPrefix(f.Image, "/img/"):
		problem = "Обложка — только загруженная на портал картинка"
	case f.Release != "" && !devblogVersionRe.MatchString(f.Release):
		problem = "Версия — в формате 1.0.0"
	}
	return
}

// save — автосохранение. version — та, с которой фронт начал правку; force —
// «перезаписать своей», когда пользователь видел конфликт и выбрал свою версию.
func (h *Devblog) save(w http.ResponseWriter, r *http.Request, cur *auth.User, body map[string]any) {
	f, problem := devblogInput(body)
	if problem != "" {
		httpx.Fail(w, http.StatusUnprocessableEntity, problem)
		return
	}
	force, _ := body["force"].(bool)
	tag, err := h.Pool.Exec(r.Context(), sqlDevblogSave, f.Title, f.Body, f.Image, cur.ID, int64Of(body["version"]), force, f.Release)
	if err != nil {
		devblogFailDB(w, "save", err)
		return
	}
	d, err := h.loadDraft(r.Context())
	if err != nil {
		devblogFailDB(w, "save reload", err)
		return
	}
	if tag.RowsAffected() == 0 {
		httpx.WriteJSON(w, http.StatusConflict, map[string]any{
			"success": false,
			"message": "Черновик изменил другой администратор",
			"data":    d,
		})
		return
	}
	httpx.OK(w, d, "Черновик сохранён")
}

// ── Публикация ─────────────────────────────────────────────────────────────────

func (h *Devblog) publish(w http.ResponseWriter, r *http.Request, cur *auth.User, body map[string]any) {
	f, problem := devblogInput(body)
	title, md, image := f.Title, f.Body, f.Image
	html := strings.TrimSpace(strVal(body["html"]))
	switch {
	case problem != "":
	case title == "":
		problem = "Укажите заголовок"
	case f.Release == "":
		problem = "Укажите версию"
	case strings.TrimSpace(md) == "" || html == "":
		problem = "Текст девблога пустой"
	case len(html) > devblogHTMLMax:
		problem = "Текст девблога слишком большой"
	}
	if problem != "" {
		httpx.Fail(w, http.StatusUnprocessableEntity, problem)
		return
	}
	if image == "" {
		image = devblogDefaultCover
	}
	force, _ := body["force"].(bool)

	ctx := r.Context()
	tx, err := h.Pool.Begin(ctx)
	if err != nil {
		devblogFailDB(w, "publish begin", err)
		return
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Публикуем то, что видел администратор: если черновик успели изменить — 409.
	var version int64
	if err := tx.QueryRow(ctx, `SELECT version FROM public.devblog_draft WHERE id = 1 FOR UPDATE`).Scan(&version); err != nil {
		devblogFailDB(w, "publish lock", err)
		return
	}
	if !force && version != int64Of(body["version"]) {
		_ = tx.Rollback(ctx)
		d, err := h.loadDraft(ctx)
		if err != nil {
			devblogFailDB(w, "publish reload", err)
			return
		}
		httpx.WriteJSON(w, http.StatusConflict, map[string]any{
			"success": false,
			"message": "Черновик изменил другой администратор — проверьте текст перед публикацией",
			"data":    d,
		})
		return
	}

	var newsID int64
	if err := tx.QueryRow(ctx, sqlDevblogInsertNews,
		title, devblogCategory, html, time.Now().Format("2006-01-02"), image).Scan(&newsID); err != nil {
		devblogFailDB(w, "publish news", err)
		return
	}
	if _, err := tx.Exec(ctx, sqlDevblogInsertPost, newsID, md, cur.ID, f.Release); err != nil {
		devblogFailDB(w, "publish post", err)
		return
	}
	// Автору окошко не нужно — он только что всё прочитал.
	if _, err := tx.Exec(ctx, sqlDevblogDismiss, newsID, cur.ID); err != nil {
		devblogFailDB(w, "publish dismiss author", err)
		return
	}
	tag, err := tx.Exec(ctx, sqlDevblogNotifyAll, cur.ID, notifyKindDevblog, newsID)
	if err != nil {
		devblogFailDB(w, "publish notify", err)
		return
	}
	if _, err := tx.Exec(ctx, sqlDevblogResetDraft, cur.ID); err != nil {
		devblogFailDB(w, "publish reset draft", err)
		return
	}
	if err := tx.Commit(ctx); err != nil {
		devblogFailDB(w, "publish commit", err)
		return
	}
	d, err := h.loadDraft(ctx)
	if err != nil {
		devblogFailDB(w, "publish draft", err)
		return
	}
	httpx.OK(w, map[string]any{"newsId": newsID, "notified": tag.RowsAffected(), "draft": d}, "Девблог опубликован")
}

// ── Окошко при входе ───────────────────────────────────────────────────────────

// sqlDevblogPending: $1 — сотрудник. Только последний опубликованный девблог и
// только если сотрудник его не закрыл: старые выпуски окошком не догоняют.
const sqlDevblogPending = `
	SELECT p.news_id, n.title, COALESCE(n.description, ''), COALESCE(n.image_path, ''),
		COALESCE(to_char(n.date, 'YYYY-MM-DD'), ''), p.published_at, p.release_version
	FROM (SELECT news_id, published_at, release_version FROM public.devblog_posts
		ORDER BY published_at DESC, news_id DESC LIMIT 1) p
	JOIN public.news n ON n.id = p.news_id
	WHERE NOT EXISTS (
		SELECT 1 FROM public.devblog_dismissed d WHERE d.news_id = p.news_id AND d.user_id = $1)`

func (h *Devblog) pending(w http.ResponseWriter, r *http.Request, cur *auth.User) {
	var (
		id                       int64
		title, html, image, date string
		release                  string
		publishedAt              time.Time
	)
	err := h.Pool.QueryRow(r.Context(), sqlDevblogPending, cur.ID).Scan(&id, &title, &html, &image, &date, &publishedAt, &release)
	if errors.Is(err, pgx.ErrNoRows) {
		httpx.OK(w, nil, "OK")
		return
	}
	if err != nil {
		// Окошко — не главное: без V20 портал работает, просто без него.
		log.Printf("devblog: pending: %v", err)
		httpx.OK(w, nil, "OK")
		return
	}
	news := &News{Pool: h.Pool, Auth: h.Auth}
	httpx.OK(w, map[string]any{
		"newsId":      id,
		"title":       title,
		"html":        html,
		"imagePath":   image,
		"date":        date,
		"publishedAt": publishedAt.Format(time.RFC3339),
		"version":     release,
		"reactions":   news.reactionSummaries(r.Context(), []int64{id}, cur.ID, nil)[id],
	}, "OK")
}

func (h *Devblog) dismiss(w http.ResponseWriter, r *http.Request, cur *auth.User, body map[string]any) {
	newsID := int64Of(body["newsId"])
	if newsID <= 0 {
		httpx.Fail(w, http.StatusBadRequest, "Некорректный newsId")
		return
	}
	if _, err := h.Pool.Exec(r.Context(), sqlDevblogDismiss, newsID, cur.ID); err != nil {
		devblogFailDB(w, "dismiss", err)
		return
	}
	// Прочитал в окошке — уведомление о том же девблоге тоже прочитано.
	if _, err := h.Pool.Exec(r.Context(), sqlDevblogMarkRead, cur.ID, notifyKindDevblog, newsID); err != nil {
		log.Printf("devblog: mark notification read: %v", err)
	}
	httpx.OK(w, map[string]any{"newsId": newsID}, "OK")
}
