package handlers

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/jackc/pgx/v5"

	"corporate.admsr.ru/backend/internal/auth"
	"corporate.admsr.ru/backend/internal/courses"
	"corporate.admsr.ru/backend/internal/httpx"
	"corporate.admsr.ru/backend/internal/tests"
)

// coursePathKey — ключ файла в хранилище: courses/{courseId}/{имя}.
var coursePathKey = regexp.MustCompile(`^courses/(\d+)/([a-zA-Z0-9._-]+)$`)

// File — GET /api/course_file.php?materialId=N | ?path=courses/{id}/{name}.
//
// Перенос api/course_file.php. Доступ: редактор раздела «Обучение» или
// сотрудник с неотменённой записью на версию курса этого материала.
// Отдаётся через http.ServeContent — он понимает Range, поэтому видео можно
// перематывать во встроенном плеере, а PDF открывается постранично.
func (h *CoursesHandler) File(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		httpx.MethodNotAllowed(w)
		return
	}
	user, err := h.Auth.RequireUser(r.Context(), r)
	if err != nil {
		if errors.Is(err, auth.ErrUnauthorized) {
			httpx.Fail(w, http.StatusUnauthorized, "Требуется авторизация")
			return
		}
		httpx.Fail(w, http.StatusInternalServerError, "Ошибка подключения к БД")
		return
	}
	ctx := r.Context()
	svc := h.svc()

	var m map[string]any
	if id, e := tests.ToInt64Public(r.URL.Query().Get("materialId")); e == nil && id > 0 {
		m, err = svc.MaterialVersionRow(ctx, id)
	} else if key := strings.TrimSpace(r.URL.Query().Get("path")); key != "" {
		if !coursePathKey.MatchString(key) {
			httpx.Fail(w, http.StatusBadRequest, "Некорректный path")
			return
		}
		m, err = scanOneMap(ctx, h.Pool, `
			SELECT m.*, t.course_version_id, v.course_id
			FROM public.course_materials m
			JOIN public.course_topics t ON t.id = m.topic_id
			JOIN public.course_versions v ON v.id = t.course_version_id
			WHERE m.file_url = $1 AND m.deleted_at IS NULL AND t.deleted_at IS NULL
			LIMIT 1`, key)
	} else {
		httpx.Fail(w, http.StatusBadRequest, "Укажите materialId или path")
		return
	}
	if errors.Is(err, pgx.ErrNoRows) || m == nil || fmt.Sprint(m["file_url"]) == "" || m["file_url"] == nil {
		httpx.Fail(w, http.StatusNotFound, "Файл не найден")
		return
	}
	if err != nil {
		httpx.Fail(w, http.StatusInternalServerError, "Ошибка подключения к БД")
		return
	}

	if !h.canReadCourseFile(r, user, m) {
		httpx.Fail(w, http.StatusForbidden, "Нет доступа к файлу")
		return
	}

	match := coursePathKey.FindStringSubmatch(fmt.Sprint(m["file_url"]))
	if match == nil {
		httpx.Fail(w, http.StatusNotFound, "Файл отсутствует на диске")
		return
	}
	abs := filepath.Join(svc.UploadRoot, match[1], match[2])
	f, err := os.Open(abs)
	if err != nil {
		httpx.Fail(w, http.StatusNotFound, "Файл отсутствует на диске")
		return
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil || st.IsDir() {
		httpx.Fail(w, http.StatusNotFound, "Файл отсутствует на диске")
		return
	}

	mimeType := strOrEmpty(m["mime_type"])
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}
	name := strOrEmpty(m["original_filename"])
	if name == "" {
		name = "file"
	}
	w.Header().Set("Content-Type", mimeType)
	w.Header().Set("Content-Disposition", "inline; filename*=UTF-8''"+url.PathEscape(name))
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Cache-Control", "private, max-age=300")
	http.ServeContent(w, r, "", st.ModTime(), f)
}

func (h *CoursesHandler) canReadCourseFile(r *http.Request, user *auth.User, m map[string]any) bool {
	ctx := r.Context()
	courseID := tests.ToInt64Must(m["course_id"])
	if _, _, err := courses.RequireCourseAdmin(ctx, h.Pool, h.Auth, r, courseID); err == nil {
		return true
	}
	var ok bool
	_ = h.Pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM public.course_enrollments
			WHERE user_id = $1 AND course_version_id = $2 AND status <> 'cancelled')`,
		user.ID, m["course_version_id"]).Scan(&ok)
	return ok
}

func strOrEmpty(v any) string {
	if v == nil {
		return ""
	}
	return strings.TrimSpace(fmt.Sprint(v))
}
