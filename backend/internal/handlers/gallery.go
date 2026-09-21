package handlers

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"corporate.admsr.ru/backend/internal/auth"
	"corporate.admsr.ru/backend/internal/httpx"
	"corporate.admsr.ru/backend/internal/media"
)

type Gallery struct {
	Pool      *pgxpool.Pool
	Auth      *auth.Service
	UploadDir string
}

func (h *Gallery) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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

func (h *Gallery) get(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	if idStr := q.Get("id"); idStr != "" {
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil || id <= 0 {
			httpx.Fail(w, http.StatusBadRequest, "Некорректный id")
			return
		}
		var (
			name, desc string
			date       *time.Time
			photoCount int
		)
		err = h.Pool.QueryRow(r.Context(), `
			SELECT g.id, g.name, COALESCE(g.description,''), g.date,
			       (SELECT COUNT(*)::int FROM public.gallery_base WHERE album_id = g.id)
			FROM public.gallery g WHERE g.id = $1`, id).Scan(&id, &name, &desc, &date, &photoCount)
		if err != nil {
			httpx.Fail(w, http.StatusNotFound, "Альбом не найден")
			return
		}
		httpx.OK(w, fmtAlbum(id, name, desc, date, nil, photoCount), "OK")
		return
	}
	if q.Has("limit") || q.Has("cursor") {
		page, err := h.cursorPage(r.Context(), q)
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
	rows, err := h.Pool.Query(r.Context(), `
		SELECT g.id, g.name, COALESCE(g.description,''), g.date,
		       (SELECT image_small_url FROM public.gallery_base WHERE album_id = g.id ORDER BY id ASC LIMIT 1) AS cover,
		       (SELECT COUNT(*)::int FROM public.gallery_base WHERE album_id = g.id) AS photo_count
		FROM public.gallery g
		ORDER BY g.date DESC NULLS LAST, g.id DESC`)
	if err != nil {
		httpx.Fail(w, http.StatusInternalServerError, "Ошибка подключения к БД")
		return
	}
	defer rows.Close()
	var items []map[string]any
	for rows.Next() {
		var id int64
		var name, desc string
		var date *time.Time
		var cover *string
		var count int
		if err := rows.Scan(&id, &name, &desc, &date, &cover, &count); err != nil {
			httpx.Fail(w, http.StatusInternalServerError, "Ошибка подключения к БД")
			return
		}
		items = append(items, fmtAlbum(id, name, desc, date, cover, count))
	}
	if items == nil {
		items = []map[string]any{}
	}
	httpx.OK(w, items, "OK")
}

func (h *Gallery) post(w http.ResponseWriter, r *http.Request) {
	if _, ok := requireSection(w, r, h.Pool, h.Auth, "gallery"); !ok {
		return
	}
	var d map[string]any
	if err := httpx.DecodeJSON(r, &d); err != nil {
		httpx.Fail(w, http.StatusBadRequest, "Некорректный JSON")
		return
	}
	name := strings.TrimSpace(fmt.Sprint(d["name"]))
	if name == "" || name == "<nil>" {
		httpx.Fail(w, http.StatusUnprocessableEntity, "Поле «name» обязательно")
		return
	}
	var desc *string
	if v := strings.TrimSpace(fmt.Sprint(d["description"])); v != "" && v != "<nil>" {
		desc = &v
	}
	var date any
	if v := strings.TrimSpace(fmt.Sprint(d["date"])); v != "" && v != "<nil>" {
		date = v
	}
	var newID int64
	err := h.Pool.QueryRow(r.Context(), `
		INSERT INTO public.gallery (name, description, date) VALUES ($1,$2,$3) RETURNING id`,
		name, desc, date).Scan(&newID)
	if err != nil {
		httpx.Fail(w, http.StatusInternalServerError, "Ошибка подключения к БД")
		return
	}
	var outName, outDesc string
	var outDate *time.Time
	_ = h.Pool.QueryRow(r.Context(), `SELECT name, COALESCE(description,''), date FROM public.gallery WHERE id=$1`, newID).
		Scan(&outName, &outDesc, &outDate)
	httpx.Created(w, fmtAlbum(newID, outName, outDesc, outDate, nil, 0), "Альбом создан")
}

func (h *Gallery) put(w http.ResponseWriter, r *http.Request) {
	if _, ok := requireSection(w, r, h.Pool, h.Auth, "gallery"); !ok {
		return
	}
	id, ok := queryID(r)
	if !ok {
		httpx.Fail(w, http.StatusBadRequest, "Укажите ?id=...")
		return
	}
	var exists int64
	if err := h.Pool.QueryRow(r.Context(), `SELECT id FROM public.gallery WHERE id=$1`, id).Scan(&exists); err != nil {
		httpx.Fail(w, http.StatusNotFound, "Альбом не найден")
		return
	}
	var d map[string]any
	if err := httpx.DecodeJSON(r, &d); err != nil {
		httpx.Fail(w, http.StatusBadRequest, "Некорректный JSON")
		return
	}
	sets := []string{}
	args := []any{}
	n := 1
	if v, has := d["name"]; has {
		name := strings.TrimSpace(fmt.Sprint(v))
		if name != "" {
			sets = append(sets, fmt.Sprintf("name=$%d", n))
			args = append(args, name)
			n++
		}
	}
	if _, has := d["description"]; has {
		sets = append(sets, fmt.Sprintf("description=$%d", n))
		args = append(args, d["description"])
		n++
	}
	if v, has := d["date"]; has {
		date := strings.TrimSpace(fmt.Sprint(v))
		if date != "" {
			sets = append(sets, fmt.Sprintf("date=$%d", n))
			args = append(args, date)
			n++
		}
	}
	if len(sets) > 0 {
		args = append(args, id)
		_, err := h.Pool.Exec(r.Context(), `UPDATE public.gallery SET `+strings.Join(sets, ", ")+fmt.Sprintf(` WHERE id=$%d`, n), args...)
		if err != nil {
			httpx.Fail(w, http.StatusInternalServerError, "Ошибка подключения к БД")
			return
		}
	}
	var name, desc string
	var date *time.Time
	_ = h.Pool.QueryRow(r.Context(), `SELECT name, COALESCE(description,''), date FROM public.gallery WHERE id=$1`, id).
		Scan(&name, &desc, &date)
	httpx.OK(w, fmtAlbum(id, name, desc, date, nil, 0), "Альбом обновлён")
}

func (h *Gallery) del(w http.ResponseWriter, r *http.Request) {
	if _, ok := requireSection(w, r, h.Pool, h.Auth, "gallery"); !ok {
		return
	}
	id, ok := queryID(r)
	if !ok {
		httpx.Fail(w, http.StatusBadRequest, "Укажите ?id=...")
		return
	}
	var exists int64
	if err := h.Pool.QueryRow(r.Context(), `SELECT id FROM public.gallery WHERE id=$1`, id).Scan(&exists); err != nil {
		httpx.Fail(w, http.StatusNotFound, "Альбом не найден")
		return
	}
	_, _ = h.Pool.Exec(r.Context(), `DELETE FROM public.gallery_base WHERE album_id=$1`, id)
	_, _ = h.Pool.Exec(r.Context(), `DELETE FROM public.gallery WHERE id=$1`, id)
	httpx.OK(w, nil, "Альбом удалён")
}

func (h *Gallery) cursorPage(ctx context.Context, q interface{ Get(string) string }) (map[string]any, error) {
	limit := 12
	if v := q.Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			limit = n
		}
	}
	if limit < 1 {
		limit = 12
	}
	if limit > 48 {
		limit = 48
	}
	cond := []string{}
	args := []any{}
	argN := 1
	if s := strings.TrimSpace(q.Get("search")); s != "" {
		cond = append(cond, fmt.Sprintf("(g.name ILIKE $%d OR g.description ILIKE $%d)", argN, argN))
		args = append(args, "%"+s+"%")
		argN++
	}
	if c := strings.TrimSpace(q.Get("cursor")); c != "" {
		cur, ok := decodeDateIDCursor(c)
		if !ok {
			return nil, errors.New("bad_cursor")
		}
		cond = append(cond, fmt.Sprintf(`(
			COALESCE(g.date, DATE '0001-01-01') < $%d
			OR (COALESCE(g.date, DATE '0001-01-01') = $%d AND g.id < $%d)
		)`, argN, argN, argN+1))
		args = append(args, cur.Date, cur.ID)
		argN += 2
	}
	sql := `SELECT g.id, g.name, COALESCE(g.description,''), g.date,
		(SELECT image_small_url FROM public.gallery_base WHERE album_id = g.id ORDER BY id ASC LIMIT 1),
		(SELECT COUNT(*)::int FROM public.gallery_base WHERE album_id = g.id)
		FROM public.gallery g`
	if len(cond) > 0 {
		sql += " WHERE " + strings.Join(cond, " AND ")
	}
	sql += fmt.Sprintf(" ORDER BY g.date DESC NULLS LAST, g.id DESC LIMIT $%d", argN)
	args = append(args, limit+1)

	rows, err := h.Pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []map[string]any
	var lastDate string
	var lastID int64
	for rows.Next() {
		var id int64
		var name, desc string
		var date *time.Time
		var cover *string
		var count int
		if err := rows.Scan(&id, &name, &desc, &date, &cover, &count); err != nil {
			return nil, err
		}
		items = append(items, fmtAlbum(id, name, desc, date, cover, count))
		lastID = id
		if date != nil {
			lastDate = date.Format("2006-01-02")
		} else {
			lastDate = "0001-01-01"
		}
	}
	hasMore := len(items) > limit
	if hasMore {
		items = items[:limit]
	}
	var next any
	if hasMore && len(items) > 0 {
		next = encodeDateIDCursor(lastDate, lastID)
		// fix: last should be last of truncated — recompute from items
		last := items[len(items)-1]
		d, _ := last["date"].(string)
		if d == "" {
			d = "0001-01-01"
		}
		id, _ := last["id"].(int64)
		next = encodeDateIDCursor(d, id)
	}
	if items == nil {
		items = []map[string]any{}
	}
	return map[string]any{"items": items, "nextCursor": next, "hasMore": hasMore}, nil
}

func fmtAlbum(id int64, name, desc string, date *time.Time, cover *string, count int) map[string]any {
	dateStr := ""
	if date != nil {
		dateStr = date.Format("2006-01-02")
	}
	return map[string]any{
		"id": id, "name": name, "description": desc, "date": dateStr,
		"cover": cover, "photo_count": count,
	}
}

// GalleryBase handles /api/gallery_base.php
type GalleryBase struct {
	Pool      *pgxpool.Pool
	Auth      *auth.Service
	UploadDir string
}

func (h *GalleryBase) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.get(w, r)
	case http.MethodPost:
		h.post(w, r)
	case http.MethodDelete:
		h.del(w, r)
	default:
		httpx.MethodNotAllowed(w)
	}
}

func (h *GalleryBase) get(w http.ResponseWriter, r *http.Request) {
	albumID, _ := strconv.ParseInt(r.URL.Query().Get("album_id"), 10, 64)
	if albumID <= 0 {
		httpx.Fail(w, http.StatusBadRequest, "Укажите album_id")
		return
	}
	q := r.URL.Query()
	if q.Has("limit") || q.Has("cursor") {
		limit := 36
		if v := q.Get("limit"); v != "" {
			if n, err := strconv.Atoi(v); err == nil {
				limit = n
			}
		}
		if limit < 1 {
			limit = 36
		}
		if limit > 72 {
			limit = 72
		}
		cond := "album_id = $1"
		args := []any{albumID}
		argN := 2
		if c := strings.TrimSpace(q.Get("cursor")); c != "" {
			cid, err := strconv.ParseInt(c, 10, 64)
			if err != nil || cid < 1 {
				httpx.Fail(w, http.StatusBadRequest, "Некорректный cursor")
				return
			}
			cond += fmt.Sprintf(" AND id > $%d", argN)
			args = append(args, cid)
			argN++
		}
		args = append(args, limit+1)
		rows, err := h.Pool.Query(r.Context(), `
			SELECT id, album_id, image_full_url, image_small_url
			FROM public.gallery_base WHERE `+cond+`
			ORDER BY id ASC LIMIT $`+strconv.Itoa(argN), args...)
		if err != nil {
			httpx.Fail(w, http.StatusInternalServerError, "Ошибка подключения к БД")
			return
		}
		defer rows.Close()
		var items []map[string]any
		for rows.Next() {
			var id, aid int64
			var full, small string
			if err := rows.Scan(&id, &aid, &full, &small); err != nil {
				httpx.Fail(w, http.StatusInternalServerError, "Ошибка подключения к БД")
				return
			}
			items = append(items, map[string]any{
				"id": id, "album_id": aid, "image_full_url": full, "image_small_url": small,
			})
		}
		hasMore := len(items) > limit
		if hasMore {
			items = items[:limit]
		}
		var next any
		if hasMore && len(items) > 0 {
			next = fmt.Sprint(items[len(items)-1]["id"])
		}
		if items == nil {
			items = []map[string]any{}
		}
		httpx.OK(w, map[string]any{"items": items, "nextCursor": next, "hasMore": hasMore}, "OK")
		return
	}
	rows, err := h.Pool.Query(r.Context(), `
		SELECT id, album_id, image_full_url, image_small_url
		FROM public.gallery_base WHERE album_id=$1 ORDER BY id ASC`, albumID)
	if err != nil {
		httpx.Fail(w, http.StatusInternalServerError, "Ошибка подключения к БД")
		return
	}
	defer rows.Close()
	var items []map[string]any
	for rows.Next() {
		var id, aid int64
		var full, small string
		_ = rows.Scan(&id, &aid, &full, &small)
		items = append(items, map[string]any{
			"id": id, "album_id": aid, "image_full_url": full, "image_small_url": small,
		})
	}
	if items == nil {
		items = []map[string]any{}
	}
	httpx.OK(w, items, "OK")
}

func (h *GalleryBase) post(w http.ResponseWriter, r *http.Request) {
	if _, ok := requireSection(w, r, h.Pool, h.Auth, "gallery"); !ok {
		return
	}
	if err := r.ParseMultipartForm(210 << 20); err != nil {
		httpx.Fail(w, http.StatusBadRequest, "Некорректный multipart")
		return
	}
	albumID, _ := strconv.ParseInt(r.FormValue("album_id"), 10, 64)
	if albumID <= 0 {
		httpx.Fail(w, http.StatusBadRequest, "Укажите album_id")
		return
	}
	file, _, err := r.FormFile("image")
	if err != nil {
		httpx.Fail(w, http.StatusBadRequest, "Файл не передан")
		return
	}
	defer file.Close()
	raw, err := io.ReadAll(file)
	if err != nil {
		httpx.Fail(w, http.StatusBadRequest, "Ошибка загрузки файла")
		return
	}
	mime := media.DetectMIME(raw)
	isVideo := mime == "video/mp4"
	if mime != "image/jpeg" && mime != "image/png" && mime != "image/webp" && !isVideo {
		httpx.Fail(w, http.StatusBadRequest, "Допустимы только изображения (JPEG, PNG, WebP) или видео MP4")
		return
	}
	maxSize := 20 << 20
	if isVideo {
		maxSize = 200 << 20
	}
	if len(raw) > maxSize {
		if isVideo {
			httpx.Fail(w, http.StatusBadRequest, "Видео превышает 200 МБ")
		} else {
			httpx.Fail(w, http.StatusBadRequest, "Файл превышает 20 МБ")
		}
		return
	}
	root := media.ImgRoot(h.UploadDir)
	var small, full string
	if isVideo {
		small, full, err = media.SaveMP4(root, raw)
	} else {
		small, full, err = media.SaveWebPPair(root, raw, mime)
	}
	if err != nil {
		httpx.Fail(w, http.StatusInternalServerError, "Ошибка обработки изображения: "+err.Error())
		return
	}
	var newID int64
	err = h.Pool.QueryRow(r.Context(), `
		INSERT INTO public.gallery_base (album_id, image_full_url, image_small_url)
		VALUES ($1,$2,$3) RETURNING id`, albumID, full, small).Scan(&newID)
	if err != nil {
		httpx.Fail(w, http.StatusInternalServerError, "Ошибка подключения к БД")
		return
	}
	msg := "Фото загружено"
	if isVideo {
		msg = "Видео загружено"
	}
	httpx.OK(w, map[string]any{
		"id": newID, "album_id": albumID, "image_full_url": full, "image_small_url": small,
	}, msg)
}

func (h *GalleryBase) del(w http.ResponseWriter, r *http.Request) {
	if _, ok := requireSection(w, r, h.Pool, h.Auth, "gallery"); !ok {
		return
	}
	id, ok := queryID(r)
	if !ok {
		httpx.Fail(w, http.StatusBadRequest, "Укажите ?id=...")
		return
	}
	_, _ = h.Pool.Exec(r.Context(), `DELETE FROM public.gallery_base WHERE id=$1`, id)
	httpx.OK(w, nil, "Фото удалено")
}

// Upload handles /api/Upload/upload.php
type Upload struct {
	UploadDir string
}

func (h *Upload) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httpx.MethodNotAllowed(w)
		return
	}
	if err := r.ParseMultipartForm(25 << 20); err != nil {
		httpx.Fail(w, http.StatusBadRequest, "Некорректный multipart")
		return
	}
	file, _, err := r.FormFile("image")
	if err != nil {
		httpx.Fail(w, http.StatusBadRequest, "Файл не передан")
		return
	}
	defer file.Close()
	raw, err := io.ReadAll(file)
	if err != nil {
		httpx.Fail(w, http.StatusBadRequest, "Ошибка загрузки файла")
		return
	}
	if len(raw) > 20<<20 {
		httpx.Fail(w, http.StatusBadRequest, "Файл превышает 20 МБ")
		return
	}
	mime := media.DetectMIME(raw)
	if mime != "image/jpeg" && mime != "image/png" && mime != "image/webp" {
		httpx.Fail(w, http.StatusBadRequest, "Допустимы только JPEG, PNG или WebP")
		return
	}
	small, full, err := media.SaveWebPPair(media.ImgRoot(h.UploadDir), raw, mime)
	if err != nil {
		httpx.Fail(w, http.StatusInternalServerError, "Ошибка обработки изображения: "+err.Error())
		return
	}
	httpx.OK(w, map[string]any{"image": small, "image_full": full}, "Изображение загружено")
}
