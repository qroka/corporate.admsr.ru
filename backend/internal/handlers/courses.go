package handlers

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"corporate.admsr.ru/backend/internal/auth"
	"corporate.admsr.ru/backend/internal/courses"
	"corporate.admsr.ru/backend/internal/httpx"
	"corporate.admsr.ru/backend/internal/tests"
)

type CoursesHandler struct {
	Pool      *pgxpool.Pool
	Auth      *auth.Service
	UploadDir string
}

func (h *CoursesHandler) svc() *courses.Service {
	return &courses.Service{Pool: h.Pool, UploadRoot: courses.UploadsRoot(h.UploadDir)}
}

func (h *CoursesHandler) decodeBody(r *http.Request) (map[string]any, error) {
	var body map[string]any
	if err := httpx.DecodeJSON(r, &body); err != nil {
		return nil, err
	}
	if body == nil {
		body = map[string]any{}
	}
	return body, nil
}

func (h *CoursesHandler) post(w http.ResponseWriter, r *http.Request, fn func(context.Context, *auth.User, map[string]any, *http.Request) (any, error)) {
	if r.Method != http.MethodPost {
		httpx.MethodNotAllowed(w)
		return
	}
	body, err := h.decodeBody(r)
	if err != nil {
		httpx.Fail(w, http.StatusBadRequest, "Некорректный JSON")
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
	data, err := fn(r.Context(), user, body, r)
	if writeCourseErr(w, err) {
		return
	}
	httpx.OK(w, data, "OK")
}

func (h *CoursesHandler) postSection(w http.ResponseWriter, r *http.Request, fn func(context.Context, *auth.User, map[string]any, *http.Request) (any, error)) {
	if r.Method != http.MethodPost {
		httpx.MethodNotAllowed(w)
		return
	}
	body, err := h.decodeBody(r)
	if err != nil {
		httpx.Fail(w, http.StatusBadRequest, "Некорректный JSON")
		return
	}
	user, err := h.Auth.RequireSection(r.Context(), r, "courses")
	if err != nil {
		if errors.Is(err, auth.ErrUnauthorized) {
			httpx.Fail(w, http.StatusUnauthorized, "Требуется авторизация")
			return
		}
		if errors.Is(err, auth.ErrForbidden) {
			httpx.Fail(w, http.StatusForbidden, "Недостаточно прав для этого раздела")
			return
		}
		httpx.Fail(w, http.StatusInternalServerError, "Ошибка подключения к БД")
		return
	}
	data, err := fn(r.Context(), user, body, r)
	if writeCourseErr(w, err) {
		return
	}
	httpx.OK(w, data, "OK")
}

func writeCourseErr(w http.ResponseWriter, err error) bool {
	if err == nil {
		return false
	}
	var he courses.HTTPError
	if errors.As(err, &he) {
		httpx.Fail(w, he.Status, he.Message)
		return true
	}
	if errors.Is(err, auth.ErrForbidden) {
		httpx.Fail(w, http.StatusForbidden, "Недостаточно прав")
		return true
	}
	if errors.Is(err, auth.ErrUnauthorized) {
		httpx.Fail(w, http.StatusUnauthorized, "Требуется авторизация")
		return true
	}
	if pe, ok := err.(*tests.PersistError); ok {
		httpx.Fail(w, http.StatusForbidden, pe.Message)
		return true
	}
	httpx.Fail(w, http.StatusInternalServerError, "Ошибка подключения к БД")
	return true
}

func (h *CoursesHandler) List(w http.ResponseWriter, r *http.Request) {
	h.postSection(w, r, func(ctx context.Context, user *auth.User, _ map[string]any, _ *http.Request) (any, error) {
		cats := auth.UserCourseCategories(ctx, h.Pool, user)
		if len(cats) == 0 {
			return map[string]any{"items": []any{}}, nil
		}
		ph := make([]string, len(cats))
		args := make([]any, len(cats))
		for i, c := range cats {
			ph[i] = fmt.Sprintf("$%d", i+1)
			args[i] = c
		}
		q := fmt.Sprintf(`
			SELECT c.*, v.status AS ver_status, v.short_description, v.published_at, v.version_number
			FROM public.course_courses c
			LEFT JOIN public.course_versions v ON v.id = c.current_version_id
			WHERE c.deleted_at IS NULL AND c.category IN (%s)
			ORDER BY c.updated_at DESC`, strings.Join(ph, ","))
		rows, err := h.Pool.Query(ctx, q, args...)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		var items []map[string]any
		for rows.Next() {
			row, err := scanRowToMap(rows)
			if err != nil {
				return nil, err
			}
			course := courses.MapCourseRow(row)
			if row["current_version_id"] != nil {
				course["currentVersion"] = map[string]any{
					"id": row["current_version_id"], "versionNumber": coursesToInt(row["version_number"]),
					"status": fmt.Sprint(row["ver_status"]), "shortDescription": fmt.Sprint(row["short_description"]),
					"publishedAt": row["published_at"],
				}
			}
			items = append(items, course)
		}
		return map[string]any{"items": items}, nil
	})
}

func (h *CoursesHandler) Get(w http.ResponseWriter, r *http.Request) {
	h.postSection(w, r, func(ctx context.Context, user *auth.User, body map[string]any, _ *http.Request) (any, error) {
		courseID, err := tests.ToInt64Public(body["courseId"])
		if err != nil || courseID <= 0 {
			return nil, courses.Err(http.StatusBadRequest, "Не передан courseId")
		}
		_, course, err := courses.RequireCourseAdmin(ctx, h.Pool, h.Auth, r, courseID)
		if err != nil {
			return nil, err
		}
		svc := h.svc()
		versionID, _ := tests.ToInt64Public(body["versionId"])
		if versionID <= 0 {
			versionID = tests.ToInt64Must(course["currentVersionId"])
		}
		version, err := svc.GetVersion(ctx, versionID)
		if err != nil || version == nil || tests.ToInt64Must(version["courseId"]) != courseID {
			return nil, courses.Err(http.StatusNotFound, "Версия не найдена")
		}
		assembled, err := svc.AssembleVersion(ctx, versionID, true)
		if err != nil {
			return nil, err
		}
		_ = user
		return map[string]any{"course": course, "version": assembled}, nil
	})
}

func (h *CoursesHandler) Create(w http.ResponseWriter, r *http.Request) {
	h.postSection(w, r, func(ctx context.Context, user *auth.User, body map[string]any, req *http.Request) (any, error) {
		category := strings.TrimSpace(fmt.Sprint(body["category"]))
		if category == "" || category == "<nil>" {
			return nil, courses.Err(http.StatusBadRequest, "Укажите категорию курса")
		}
		if !auth.CanEditCourseCategory(ctx, h.Pool, user, &category) {
			return nil, courses.Err(http.StatusForbidden, "Нет доступа к этой категории курсов")
		}
		return h.svc().CreateCourse(ctx, user, body, req)
	})
}

func (h *CoursesHandler) Update(w http.ResponseWriter, r *http.Request) {
	h.postSection(w, r, func(ctx context.Context, user *auth.User, body map[string]any, req *http.Request) (any, error) {
		courseID, err := tests.ToInt64Public(body["courseId"])
		if err != nil || courseID <= 0 {
			return nil, courses.Err(http.StatusBadRequest, "Не передан courseId")
		}
		_, course, err := courses.RequireCourseAdmin(ctx, h.Pool, h.Auth, r, courseID)
		if err != nil {
			return nil, err
		}
		versionID, _ := tests.ToInt64Public(body["versionId"])
		if versionID <= 0 {
			versionID = tests.ToInt64Must(course["currentVersionId"])
		}
		svc := h.svc()
		version, err := svc.GetVersion(ctx, versionID)
		if err != nil || version == nil || tests.ToInt64Must(version["courseId"]) != courseID {
			return nil, courses.Err(http.StatusNotFound, "Версия не найдена")
		}
		if _, ok := body["category"]; ok {
			newCat := strings.TrimSpace(fmt.Sprint(body["category"]))
			if newCat == "" || !auth.CanEditCourseCategory(ctx, h.Pool, user, &newCat) {
				return nil, courses.Err(http.StatusForbidden, "Нет доступа к выбранной категории курсов")
			}
		}
		contentKeys := []string{"shortDescription", "fullDescription", "coverUrl", "sequentialProgress", "completionRule", "defaultDeadlineDays", "finalPassingScore", "requireFinalTest", "generateCertificate"}
		wantsContent := false
		for _, k := range contentKeys {
			if _, ok := body[k]; ok {
				wantsContent = true
				break
			}
		}
		if wantsContent {
			if err := courses.AssertVersionEditable(version); err != nil {
				return nil, err
			}
		}
		tx, err := h.Pool.Begin(ctx)
		if err != nil {
			return nil, err
		}
		defer tx.Rollback(ctx)
		if _, ok := body["title"]; ok || body["category"] != nil {
			var title any
			if t, ok := body["title"]; ok {
				title = strings.TrimSpace(fmt.Sprint(t))
			}
			_, err = tx.Exec(ctx, `
				UPDATE public.course_courses SET
				 title = COALESCE($2, title),
				 category = CASE WHEN $3 THEN $4 ELSE category END,
				 updated_at = now() WHERE id = $1`,
				courseID, title, body["category"] != nil, body["category"])
			if err != nil {
				return nil, err
			}
		}
		if wantsContent {
			if v, ok := body["shortDescription"]; ok {
				s := courses.SanitizeHTML(strPtrVal(v))
				_, _ = tx.Exec(ctx, `UPDATE public.course_versions SET short_description = $2, updated_at = now() WHERE id = $1`, versionID, s)
			}
			if v, ok := body["fullDescription"]; ok {
				s := courses.SanitizeHTML(strPtrVal(v))
				_, _ = tx.Exec(ctx, `UPDATE public.course_versions SET full_description = $2, updated_at = now() WHERE id = $1`, versionID, s)
			}
			if v, ok := body["coverUrl"]; ok {
				_, _ = tx.Exec(ctx, `UPDATE public.course_versions SET cover_url = $2, updated_at = now() WHERE id = $1`, versionID, v)
			}
			if v, ok := body["sequentialProgress"]; ok {
				_, _ = tx.Exec(ctx, `UPDATE public.course_versions SET sequential_progress = $2, updated_at = now() WHERE id = $1`, versionID, courses.Bool(v))
			}
			if v, ok := body["completionRule"]; ok {
				_, _ = tx.Exec(ctx, `UPDATE public.course_versions SET completion_rule = $2, updated_at = now() WHERE id = $1`, versionID, v)
			}
			if v, ok := body["defaultDeadlineDays"]; ok {
				_, _ = tx.Exec(ctx, `UPDATE public.course_versions SET default_deadline_days = $2, updated_at = now() WHERE id = $1`, versionID, v)
			}
			if v, ok := body["finalPassingScore"]; ok {
				_, _ = tx.Exec(ctx, `UPDATE public.course_versions SET final_passing_score = $2, updated_at = now() WHERE id = $1`, versionID, v)
			}
			if v, ok := body["requireFinalTest"]; ok {
				_, _ = tx.Exec(ctx, `UPDATE public.course_versions SET require_final_test = $2, updated_at = now() WHERE id = $1`, versionID, courses.Bool(v))
			}
			if v, ok := body["generateCertificate"]; ok {
				_, _ = tx.Exec(ctx, `UPDATE public.course_versions SET generate_certificate = $2, updated_at = now() WHERE id = $1`, versionID, courses.Bool(v))
			}
		}
		uid := user.ID
		courses.Audit(ctx, h.Pool, &uid, "course.update", "course", &courseID, map[string]any{"versionId": versionID}, req)
		if err := tx.Commit(ctx); err != nil {
			return nil, courses.Err(http.StatusInternalServerError, "Ошибка обновления")
		}
		course, _ = svc.GetCourse(ctx, courseID, false)
		assembled, err := svc.AssembleVersion(ctx, versionID, true)
		if err != nil {
			return nil, err
		}
		return map[string]any{"course": course, "version": assembled}, nil
	})
}

func strPtrVal(v any) *string {
	s := strings.TrimSpace(fmt.Sprint(v))
	if s == "" || s == "<nil>" {
		empty := ""
		return &empty
	}
	return &s
}

func coursesToInt(v any) int {
	n, _ := tests.ToInt64Public(v)
	return int(n)
}

func (h *CoursesHandler) Delete(w http.ResponseWriter, r *http.Request) {
	h.postSection(w, r, func(ctx context.Context, user *auth.User, body map[string]any, req *http.Request) (any, error) {
		courseID, err := tests.ToInt64Public(body["courseId"])
		if err != nil || courseID <= 0 {
			return nil, courses.Err(http.StatusBadRequest, "Не передан courseId")
		}
		if _, _, err := courses.RequireCourseAdmin(ctx, h.Pool, h.Auth, r, courseID); err != nil {
			return nil, err
		}
		_, err = h.Pool.Exec(ctx, `UPDATE public.course_courses SET deleted_at = now(), updated_at = now() WHERE id = $1`, courseID)
		if err != nil {
			return nil, err
		}
		uid := user.ID
		courses.Audit(ctx, h.Pool, &uid, "course.delete", "course", &courseID, map[string]any{}, req)
		return map[string]any{"courseId": courseID}, nil
	})
}

func (h *CoursesHandler) Publish(w http.ResponseWriter, r *http.Request) {
	h.postSection(w, r, func(ctx context.Context, user *auth.User, body map[string]any, req *http.Request) (any, error) {
		svc := h.svc()
		versionID, err := svc.ResolveVersionID(ctx, body)
		if err != nil {
			return nil, err
		}
		version, err := svc.GetVersion(ctx, versionID)
		if err != nil || version == nil {
			return nil, courses.Err(http.StatusNotFound, "Версия не найдена")
		}
		if _, _, err := courses.RequireCourseAdmin(ctx, h.Pool, h.Auth, r, tests.ToInt64Must(version["courseId"])); err != nil {
			return nil, err
		}
		assembled, err := svc.PublishVersion(ctx, versionID, user, req)
		if err != nil {
			return nil, err
		}
		return map[string]any{"version": assembled}, nil
	})
}

func (h *CoursesHandler) Unpublish(w http.ResponseWriter, r *http.Request) {
	h.postSection(w, r, func(ctx context.Context, user *auth.User, body map[string]any, req *http.Request) (any, error) {
		svc := h.svc()
		versionID, err := svc.ResolveVersionID(ctx, body)
		if err != nil {
			return nil, err
		}
		version, err := svc.GetVersion(ctx, versionID)
		if err != nil || version == nil {
			return nil, courses.Err(http.StatusNotFound, "Версия не найдена")
		}
		if _, _, err := courses.RequireCourseAdmin(ctx, h.Pool, h.Auth, r, tests.ToInt64Must(version["courseId"])); err != nil {
			return nil, err
		}
		if fmt.Sprint(version["status"]) != "published" {
			return nil, courses.Err(http.StatusConflict, "Версия не опубликована")
		}
		_, err = h.Pool.Exec(ctx, `UPDATE public.course_versions SET status = 'draft', published_at = NULL, updated_at = now() WHERE id = $1`, versionID)
		if err != nil {
			return nil, err
		}
		uid := user.ID
		courses.Audit(ctx, h.Pool, &uid, "course.version.unpublish", "course_version", &versionID, map[string]any{}, req)
		assembled, err := svc.AssembleVersion(ctx, versionID, true)
		if err != nil {
			return nil, err
		}
		return map[string]any{"version": assembled}, nil
	})
}

func (h *CoursesHandler) TopicsCreate(w http.ResponseWriter, r *http.Request) {
	h.postSection(w, r, h.topicMutate("course.topic.create", func(ctx context.Context, versionID int64, body map[string]any) (map[string]any, error) {
		title := strings.TrimSpace(fmt.Sprint(body["title"]))
		if title == "" {
			return nil, courses.Err(http.StatusBadRequest, "Не указано название темы")
		}
		var ord int
		_ = h.Pool.QueryRow(ctx, `SELECT COALESCE(MAX(sort_order), -1) + 1 FROM public.course_topics WHERE course_version_id = $1 AND deleted_at IS NULL`, versionID).Scan(&ord)
		row, err := scanOneMap(ctx, h.Pool, `
			INSERT INTO public.course_topics (course_version_id, title, description, sort_order, is_required, minimum_active_seconds)
			VALUES ($1,$2,$3,$4,$5,$6) RETURNING *`,
			versionID, title, fmt.Sprint(body["description"]), ord, courses.Bool(body["isRequired"]), maxInt(toIntDefaultMap(body["minimumActiveSeconds"], 0), 0))
		if err != nil {
			return nil, err
		}
		return map[string]any{
			"topic": map[string]any{
				"id": row["id"], "courseVersionId": versionID, "title": row["title"], "description": row["description"],
				"sortOrder": row["sort_order"], "isRequired": courses.Bool(row["is_required"]),
				"minimumActiveSeconds": row["minimum_active_seconds"], "completionRule": row["completion_rule"],
				"materials": []any{}, "topicTest": nil,
			},
		}, nil
	}))
}

func (h *CoursesHandler) TopicsUpdate(w http.ResponseWriter, r *http.Request) {
	h.postSection(w, r, func(ctx context.Context, user *auth.User, body map[string]any, req *http.Request) (any, error) {
		topicID, err := tests.ToInt64Public(body["topicId"])
		if err != nil || topicID <= 0 {
			return nil, courses.Err(http.StatusBadRequest, "Не передан topicId")
		}
		svc := h.svc()
		topic, err := svc.TopicVersionRow(ctx, topicID)
		if errors.Is(err, pgx.ErrNoRows) || topic == nil {
			return nil, courses.Err(http.StatusNotFound, "Тема не найдена")
		}
		if _, _, err := courses.RequireCourseAdmin(ctx, h.Pool, h.Auth, r, tests.ToInt64Must(topic["course_id"])); err != nil {
			return nil, err
		}
		if fmt.Sprint(topic["version_status"]) != "draft" && fmt.Sprint(topic["version_status"]) != "published" {
			return nil, courses.Err(http.StatusConflict, "Версия недоступна для редактирования (архивирована)")
		}
		if t, ok := body["title"]; ok {
			if strings.TrimSpace(fmt.Sprint(t)) == "" {
				return nil, courses.Err(http.StatusBadRequest, "Пустое название темы")
			}
			_, _ = h.Pool.Exec(ctx, `UPDATE public.course_topics SET title = $2, updated_at = now() WHERE id = $1`, topicID, strings.TrimSpace(fmt.Sprint(t)))
		}
		if v, ok := body["description"]; ok {
			_, _ = h.Pool.Exec(ctx, `UPDATE public.course_topics SET description = $2, updated_at = now() WHERE id = $1`, topicID, v)
		}
		if v, ok := body["isRequired"]; ok {
			_, _ = h.Pool.Exec(ctx, `UPDATE public.course_topics SET is_required = $2, updated_at = now() WHERE id = $1`, topicID, courses.Bool(v))
		}
		if v, ok := body["minimumActiveSeconds"]; ok {
			_, _ = h.Pool.Exec(ctx, `UPDATE public.course_topics SET minimum_active_seconds = $2, updated_at = now() WHERE id = $1`, topicID, toIntDefaultMap(v, 0))
		}
		if v, ok := body["completionRule"]; ok {
			_, _ = h.Pool.Exec(ctx, `UPDATE public.course_topics SET completion_rule = $2, updated_at = now() WHERE id = $1`, topicID, v)
		}
		row, _ := scanOneMap(ctx, h.Pool, `SELECT * FROM public.course_topics WHERE id = $1`, topicID)
		uid := user.ID
		courses.Audit(ctx, h.Pool, &uid, "course.topic.update", "course_topic", &topicID, map[string]any{}, req)
		return map[string]any{"topic": row}, nil
	})
}

func (h *CoursesHandler) TopicsDelete(w http.ResponseWriter, r *http.Request) {
	h.postSection(w, r, func(ctx context.Context, user *auth.User, body map[string]any, req *http.Request) (any, error) {
		topicID, err := tests.ToInt64Public(body["topicId"])
		if err != nil || topicID <= 0 {
			return nil, courses.Err(http.StatusBadRequest, "Не передан topicId")
		}
		svc := h.svc()
		topic, err := svc.TopicVersionRow(ctx, topicID)
		if errors.Is(err, pgx.ErrNoRows) || topic == nil {
			return nil, courses.Err(http.StatusNotFound, "Тема не найдена")
		}
		if _, _, err := courses.RequireCourseAdmin(ctx, h.Pool, h.Auth, r, tests.ToInt64Must(topic["course_id"])); err != nil {
			return nil, err
		}
		if fmt.Sprint(topic["version_status"]) != "draft" && fmt.Sprint(topic["version_status"]) != "published" {
			return nil, courses.Err(http.StatusConflict, "Версия недоступна для редактирования (архивирована)")
		}
		_, _ = h.Pool.Exec(ctx, `UPDATE public.course_topics SET deleted_at = now(), updated_at = now() WHERE id = $1`, topicID)
		uid := user.ID
		courses.Audit(ctx, h.Pool, &uid, "course.topic.delete", "course_topic", &topicID, map[string]any{}, req)
		return map[string]any{"topicId": topicID}, nil
	})
}

func (h *CoursesHandler) TopicsOrder(w http.ResponseWriter, r *http.Request) {
	h.postSection(w, r, func(ctx context.Context, user *auth.User, body map[string]any, req *http.Request) (any, error) {
		versionID, err := tests.ToInt64Public(body["versionId"])
		if err != nil || versionID <= 0 {
			return nil, courses.Err(http.StatusBadRequest, "Не передан versionId")
		}
		svc := h.svc()
		version, err := svc.GetVersion(ctx, versionID)
		if err != nil || version == nil {
			return nil, courses.Err(http.StatusNotFound, "Версия не найдена")
		}
		if _, _, err := courses.RequireCourseAdmin(ctx, h.Pool, h.Auth, r, tests.ToInt64Must(version["courseId"])); err != nil {
			return nil, err
		}
		if err := courses.AssertVersionEditable(version); err != nil {
			return nil, err
		}
		ids := uniqueInt64FromAny(body["topicIds"])
		tx, err := h.Pool.Begin(ctx)
		if err != nil {
			return nil, err
		}
		defer tx.Rollback(ctx)
		for i, tid := range ids {
			_, _ = tx.Exec(ctx, `UPDATE public.course_topics SET sort_order = $2, updated_at = now() WHERE id = $1 AND course_version_id = $3`, tid, i, versionID)
		}
		if err := tx.Commit(ctx); err != nil {
			return nil, courses.Err(http.StatusInternalServerError, "Не удалось изменить порядок")
		}
		uid := user.ID
		courses.Audit(ctx, h.Pool, &uid, "course.topics.order", "course_version", &versionID, map[string]any{"topicIds": ids}, req)
		return map[string]any{"versionId": versionID, "topicIds": ids}, nil
	})
}

func (h *CoursesHandler) topicMutate(auditAction string, fn func(context.Context, int64, map[string]any) (map[string]any, error)) func(context.Context, *auth.User, map[string]any, *http.Request) (any, error) {
	return func(ctx context.Context, user *auth.User, body map[string]any, req *http.Request) (any, error) {
		svc := h.svc()
		versionID, err := svc.ResolveVersionID(ctx, body)
		if err != nil {
			return nil, err
		}
		version, err := svc.GetVersion(ctx, versionID)
		if err != nil || version == nil {
			return nil, courses.Err(http.StatusNotFound, "Версия не найдена")
		}
		if _, _, err := courses.RequireCourseAdmin(ctx, h.Pool, h.Auth, req, tests.ToInt64Must(version["courseId"])); err != nil {
			return nil, err
		}
		if err := courses.AssertVersionEditable(version); err != nil {
			return nil, err
		}
		out, err := fn(ctx, versionID, body)
		if err != nil {
			return nil, err
		}
		if topic, ok := out["topic"].(map[string]any); ok {
			if id, err := tests.ToInt64Public(topic["id"]); err == nil {
				uid := user.ID
				courses.Audit(ctx, h.Pool, &uid, auditAction, "course_topic", &id, map[string]any{"versionId": versionID}, req)
			}
		}
		return out, nil
	}
}

func (h *CoursesHandler) MaterialsCreate(w http.ResponseWriter, r *http.Request) {
	h.postSection(w, r, func(ctx context.Context, user *auth.User, body map[string]any, req *http.Request) (any, error) {
		topicID, err := tests.ToInt64Public(body["topicId"])
		if err != nil || topicID <= 0 {
			return nil, courses.Err(http.StatusBadRequest, "Не передан topicId")
		}
		svc := h.svc()
		topic, err := svc.TopicVersionRow(ctx, topicID)
		if errors.Is(err, pgx.ErrNoRows) || topic == nil {
			return nil, courses.Err(http.StatusNotFound, "Тема не найдена")
		}
		if _, _, err := courses.RequireCourseAdmin(ctx, h.Pool, h.Auth, r, tests.ToInt64Must(topic["course_id"])); err != nil {
			return nil, err
		}
		if fmt.Sprint(topic["version_status"]) != "draft" && fmt.Sprint(topic["version_status"]) != "published" {
			return nil, courses.Err(http.StatusConflict, "Версия недоступна для редактирования (архивирована)")
		}
		title := strings.TrimSpace(fmt.Sprint(body["title"]))
		if title == "" {
			return nil, courses.Err(http.StatusBadRequest, "Не указано название материала")
		}
		matType := fmt.Sprint(body["type"])
		if matType == "" {
			matType = "rich_text"
		}
		if matType == "pdf" || matType == "image" || matType == "video" {
			matType = "file"
		}
		var ord int
		_ = h.Pool.QueryRow(ctx, `SELECT COALESCE(MAX(sort_order), -1)+1 FROM public.course_materials WHERE topic_id = $1 AND deleted_at IS NULL`, topicID).Scan(&ord)
		row, err := scanOneMap(ctx, h.Pool, `
			INSERT INTO public.course_materials (topic_id, type, title, description, content_html, file_url, external_url, mime_type, file_size, original_filename, sort_order, is_required, minimum_active_seconds)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13) RETURNING *`,
			topicID, matType, title, body["description"], courses.SanitizeHTML(strPtrVal(body["contentHtml"])),
			body["fileUrl"], body["externalUrl"], body["mimeType"], body["fileSize"], body["originalFilename"],
			ord, courses.Bool(body["isRequired"]), toIntDefaultMap(body["minimumActiveSeconds"], 0))
		if err != nil {
			return nil, err
		}
		mid := tests.ToInt64Must(row["id"])
		uid := user.ID
		courses.Audit(ctx, h.Pool, &uid, "course.material.create", "course_material", &mid, map[string]any{}, req)
		return map[string]any{"material": mapMaterialRow(row, true)}, nil
	})
}

func mapMaterialRow(row map[string]any, withContent bool) map[string]any {
	m := map[string]any{
		"id": row["id"], "topicId": row["topic_id"], "type": row["type"], "title": row["title"],
		"description": row["description"], "fileUrl": row["file_url"], "externalUrl": row["external_url"],
		"mimeType": row["mime_type"], "fileSize": row["file_size"], "originalFilename": row["original_filename"],
		"sortOrder": row["sort_order"], "isRequired": courses.Bool(row["is_required"]),
		"minimumActiveSeconds": row["minimum_active_seconds"],
	}
	if withContent {
		m["contentHtml"] = row["content_html"]
	}
	return m
}

func (h *CoursesHandler) MaterialsUpdate(w http.ResponseWriter, r *http.Request) {
	h.postSection(w, r, func(ctx context.Context, user *auth.User, body map[string]any, req *http.Request) (any, error) {
		materialID, err := tests.ToInt64Public(body["materialId"])
		if err != nil || materialID <= 0 {
			return nil, courses.Err(http.StatusBadRequest, "Не передан materialId")
		}
		svc := h.svc()
		m, err := svc.MaterialVersionRow(ctx, materialID)
		if errors.Is(err, pgx.ErrNoRows) || m == nil {
			return nil, courses.Err(http.StatusNotFound, "Материал не найден")
		}
		if _, _, err := courses.RequireCourseAdmin(ctx, h.Pool, h.Auth, r, tests.ToInt64Must(m["course_id"])); err != nil {
			return nil, err
		}
		if fmt.Sprint(m["version_status"]) != "draft" && fmt.Sprint(m["version_status"]) != "published" {
			return nil, courses.Err(http.StatusConflict, "Версия недоступна для редактирования (архивирована)")
		}
		if v, ok := body["title"]; ok {
			_, _ = h.Pool.Exec(ctx, `UPDATE public.course_materials SET title = $2, updated_at = now() WHERE id = $1`, materialID, v)
		}
		if v, ok := body["description"]; ok {
			_, _ = h.Pool.Exec(ctx, `UPDATE public.course_materials SET description = $2, updated_at = now() WHERE id = $1`, materialID, v)
		}
		if v, ok := body["contentHtml"]; ok {
			s := courses.SanitizeHTML(strPtrVal(v))
			_, _ = h.Pool.Exec(ctx, `UPDATE public.course_materials SET content_html = $2, updated_at = now() WHERE id = $1`, materialID, s)
		}
		for _, pair := range [][2]string{{"fileUrl", "file_url"}, {"externalUrl", "external_url"}, {"mimeType", "mime_type"}, {"fileSize", "file_size"}, {"originalFilename", "original_filename"}, {"type", "type"}} {
			if v, ok := body[pair[0]]; ok {
				_, _ = h.Pool.Exec(ctx, fmt.Sprintf(`UPDATE public.course_materials SET %s = $2, updated_at = now() WHERE id = $1`, pair[1]), materialID, v)
			}
		}
		if v, ok := body["isRequired"]; ok {
			_, _ = h.Pool.Exec(ctx, `UPDATE public.course_materials SET is_required = $2, updated_at = now() WHERE id = $1`, materialID, courses.Bool(v))
		}
		if v, ok := body["minimumActiveSeconds"]; ok {
			_, _ = h.Pool.Exec(ctx, `UPDATE public.course_materials SET minimum_active_seconds = $2, updated_at = now() WHERE id = $1`, materialID, v)
		}
		row, _ := scanOneMap(ctx, h.Pool, `SELECT * FROM public.course_materials WHERE id = $1`, materialID)
		uid := user.ID
		courses.Audit(ctx, h.Pool, &uid, "course.material.update", "course_material", &materialID, map[string]any{}, req)
		return map[string]any{"material": mapMaterialRow(row, true)}, nil
	})
}

func (h *CoursesHandler) MaterialsDelete(w http.ResponseWriter, r *http.Request) {
	h.postSection(w, r, func(ctx context.Context, user *auth.User, body map[string]any, req *http.Request) (any, error) {
		materialID, err := tests.ToInt64Public(body["materialId"])
		if err != nil || materialID <= 0 {
			return nil, courses.Err(http.StatusBadRequest, "Не передан materialId")
		}
		svc := h.svc()
		m, err := svc.MaterialVersionRow(ctx, materialID)
		if errors.Is(err, pgx.ErrNoRows) || m == nil {
			return nil, courses.Err(http.StatusNotFound, "Материал не найден")
		}
		if _, _, err := courses.RequireCourseAdmin(ctx, h.Pool, h.Auth, r, tests.ToInt64Must(m["course_id"])); err != nil {
			return nil, err
		}
		_, _ = h.Pool.Exec(ctx, `UPDATE public.course_materials SET deleted_at = now(), updated_at = now() WHERE id = $1`, materialID)
		uid := user.ID
		courses.Audit(ctx, h.Pool, &uid, "course.material.delete", "course_material", &materialID, map[string]any{}, req)
		return map[string]any{"materialId": materialID}, nil
	})
}

func (h *CoursesHandler) MaterialsUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httpx.MethodNotAllowed(w)
		return
	}
	user, err := h.Auth.RequireSection(r.Context(), r, "courses")
	if err != nil {
		if errors.Is(err, auth.ErrUnauthorized) {
			httpx.Fail(w, http.StatusUnauthorized, "Требуется авторизация")
			return
		}
		if errors.Is(err, auth.ErrForbidden) {
			httpx.Fail(w, http.StatusForbidden, "Недостаточно прав для этого раздела")
			return
		}
		httpx.Fail(w, http.StatusInternalServerError, "Ошибка подключения к БД")
		return
	}
	if err := r.ParseMultipartForm(52 << 20); err != nil {
		httpx.Fail(w, http.StatusBadRequest, "Файл не загружен")
		return
	}
	ctx := r.Context()
	svc := h.svc()
	topicID, _ := tests.ToInt64Public(r.FormValue("topicId"))
	materialID, _ := tests.ToInt64Public(r.FormValue("materialId"))
	var topic map[string]any
	if materialID > 0 {
		mRow, err := svc.MaterialVersionRow(ctx, materialID)
		if errors.Is(err, pgx.ErrNoRows) || mRow == nil {
			httpx.Fail(w, http.StatusNotFound, "Материал не найден")
			return
		}
		topicID = tests.ToInt64Must(mRow["topic_id"])
		topic, _ = svc.TopicVersionRow(ctx, topicID)
	} else {
		topic, _ = svc.TopicVersionRow(ctx, topicID)
	}
	if topicID <= 0 || topic == nil {
		httpx.Fail(w, http.StatusBadRequest, "Нужен topicId или materialId")
		return
	}
	if _, _, err := courses.RequireCourseAdmin(ctx, h.Pool, h.Auth, r, tests.ToInt64Must(topic["course_id"])); err != nil {
		writeCourseErr(w, err)
		return
	}
	file, hdr, err := r.FormFile("file")
	if err != nil {
		httpx.Fail(w, http.StatusBadRequest, "Файл не загружен")
		return
	}
	defer file.Close()
	if hdr.Size > 50<<20 {
		httpx.Fail(w, http.StatusBadRequest, "Файл больше 50 МБ")
		return
	}
	ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(hdr.Filename), "."))
	allowed := map[string]bool{"pdf": true, "png": true, "jpg": true, "jpeg": true, "gif": true, "webp": true, "mp4": true, "webm": true, "doc": true, "docx": true, "xls": true, "xlsx": true, "ppt": true, "pptx": true, "txt": true, "zip": true}
	forbidden := map[string]bool{"php": true, "phar": true, "phtml": true, "sh": true, "exe": true, "bat": true, "cmd": true, "js": true, "html": true, "htm": true, "shtml": true, "cgi": true}
	if ext == "" || forbidden[ext] || !allowed[ext] {
		httpx.Fail(w, http.StatusBadRequest, "Недопустимое расширение файла")
		return
	}
	mimeType := mime.TypeByExtension("." + ext)
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}
	courseID := tests.ToInt64Must(topic["course_id"])
	absDir, relPrefix := courses.CourseUploadDir(svc.UploadRoot, courseID)
	storedName := randomHex(16) + "." + ext
	absPath := filepath.Join(absDir, storedName)
	out, err := os.Create(absPath)
	if err != nil {
		httpx.Fail(w, http.StatusInternalServerError, "Не удалось сохранить файл")
		return
	}
	if _, err := io.Copy(out, file); err != nil {
		out.Close()
		httpx.Fail(w, http.StatusInternalServerError, "Не удалось сохранить файл")
		return
	}
	out.Close()
	relKey := relPrefix + "/" + storedName
	fileURL := "/api/course_file.php?path=" + relKey
	matType := r.FormValue("type")
	if matType == "" || matType == "pdf" || matType == "image" || matType == "video" {
		matType = "file"
	}
	title := strings.TrimSpace(r.FormValue("title"))
	if title == "" {
		title = hdr.Filename
	}
	if materialID > 0 {
		_, _ = h.Pool.Exec(ctx, `UPDATE public.course_materials SET type=$2, title=$3, file_url=$4, mime_type=$5, file_size=$6, original_filename=$7, updated_at=now() WHERE id=$1`,
			materialID, matType, title, relKey, mimeType, hdr.Size, hdr.Filename)
	} else {
		var ord int
		_ = h.Pool.QueryRow(ctx, `SELECT COALESCE(MAX(sort_order), -1)+1 FROM public.course_materials WHERE topic_id=$1 AND deleted_at IS NULL`, topicID).Scan(&ord)
		err = h.Pool.QueryRow(ctx, `
			INSERT INTO public.course_materials (topic_id, type, title, description, file_url, mime_type, file_size, original_filename, sort_order, is_required, minimum_active_seconds)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11) RETURNING id`,
			topicID, matType, title, r.FormValue("description"), relKey, mimeType, hdr.Size, hdr.Filename, ord,
			courses.Bool(r.FormValue("isRequired")), toIntDefaultMap(r.FormValue("minimumActiveSeconds"), 0)).Scan(&materialID)
		if err != nil {
			httpx.Fail(w, http.StatusInternalServerError, "Не удалось сохранить файл")
			return
		}
	}
	uid := user.ID
	courses.Audit(ctx, h.Pool, &uid, "course.material.upload", "course_material", &materialID, map[string]any{"fileUrl": relKey, "size": hdr.Size}, r)
	httpx.OK(w, map[string]any{
		"materialId": materialID, "fileUrl": fileURL, "storageKey": relKey,
		"mimeType": mimeType, "fileSize": hdr.Size, "originalFilename": hdr.Filename, "type": matType,
	}, "OK")
}

func randomHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func (h *CoursesHandler) TestsCreate(w http.ResponseWriter, r *http.Request) {
	h.postSection(w, r, func(ctx context.Context, user *auth.User, body map[string]any, req *http.Request) (any, error) {
		svc := h.svc()
		topicID, _ := tests.ToInt64Public(body["topicId"])
		testType := fmt.Sprint(body["type"])
		if testType == "" {
			if topicID > 0 {
				testType = "topic"
			} else {
				testType = "final"
			}
		}
		testData, _ := body["form"].(map[string]any)
		if testData == nil {
			testData = body
		}
		if testType == "final" || (topicID <= 0 && body["versionId"] != nil) {
			versionID, err := svc.ResolveVersionID(ctx, body)
			if err != nil {
				return nil, err
			}
			version, err := svc.GetVersion(ctx, versionID)
			if err != nil || version == nil {
				return nil, courses.Err(http.StatusNotFound, "Версия не найдена")
			}
			if _, _, err := courses.RequireCourseAdmin(ctx, h.Pool, h.Auth, req, tests.ToInt64Must(version["courseId"])); err != nil {
				return nil, err
			}
			link, err := svc.CreateFinalTest(ctx, versionID, user, testData, req)
			if err != nil {
				return nil, err
			}
			return map[string]any{"link": link}, nil
		}
		if topicID <= 0 {
			return nil, courses.Err(http.StatusBadRequest, "Укажите topicId или type=final")
		}
		topic, err := svc.TopicVersionRow(ctx, topicID)
		if errors.Is(err, pgx.ErrNoRows) || topic == nil {
			return nil, courses.Err(http.StatusNotFound, "Тема не найдена")
		}
		if _, _, err := courses.RequireCourseAdmin(ctx, h.Pool, h.Auth, req, tests.ToInt64Must(topic["course_id"])); err != nil {
			return nil, err
		}
		link, err := svc.CreateTopicTest(ctx, topicID, user, testData, req)
		if err != nil {
			return nil, err
		}
		return map[string]any{"link": link}, nil
	})
}

func (h *CoursesHandler) TestsGet(w http.ResponseWriter, r *http.Request) {
	h.post(w, r, func(ctx context.Context, user *auth.User, body map[string]any, _ *http.Request) (any, error) {
		linkID, _ := tests.ToInt64Public(body["courseTestLinkId"])
		formID, _ := tests.ToInt64Public(body["testFormId"])
		if formID <= 0 {
			formID, _ = tests.ToInt64Public(body["formId"])
		}
		var link map[string]any
		var err error
		if linkID > 0 {
			link, err = scanOneMap(ctx, h.Pool, `SELECT * FROM public.course_test_links WHERE id = $1`, linkID)
		} else if formID > 0 {
			link, err = scanOneMap(ctx, h.Pool, `SELECT * FROM public.course_test_links WHERE test_form_id = $1 LIMIT 1`, formID)
		} else if tid, e := tests.ToInt64Public(body["topicId"]); e == nil && tid > 0 {
			link, err = scanOneMap(ctx, h.Pool, `SELECT * FROM public.course_test_links WHERE topic_id = $1 AND type = 'topic' LIMIT 1`, tid)
		} else {
			svc := h.svc()
			versionID, e := svc.ResolveVersionID(ctx, body)
			if e != nil {
				return nil, e
			}
			link, err = scanOneMap(ctx, h.Pool, `SELECT * FROM public.course_test_links WHERE course_version_id = $1 AND type = 'final' LIMIT 1`, versionID)
		}
		if errors.Is(err, pgx.ErrNoRows) || link == nil {
			return nil, courses.Err(http.StatusNotFound, "Тест курса не найден")
		}
		formID = tests.ToInt64Must(link["test_form_id"])
		versionID := tests.ToInt64Must(link["course_version_id"])
		allowed := auth.IsAdmin(user)
		if !allowed {
			var ok bool
			_ = h.Pool.QueryRow(ctx, `
				SELECT EXISTS(
				 SELECT 1 FROM public.course_enrollments
				 WHERE user_id = $1 AND course_version_id = $2 AND status <> 'cancelled')`, user.ID, versionID).Scan(&ok)
			allowed = ok
		}
		if !allowed {
			return nil, courses.Err(http.StatusForbidden, "Нет доступа")
		}
		svc := h.svc()
		form, err := tests.LoadForm(ctx, h.Pool, formID, user.ID)
		if err != nil || form == nil {
			return nil, courses.Err(http.StatusNotFound, "Форма не найдена")
		}
		if !auth.IsAdmin(user) {
			form = tests.StripCorrect(form)
		}
		summary, _ := svc.TestFormSummary(ctx, formID)
		return map[string]any{"link": courses.MapTestLink(link, summary), "form": form}, nil
	})
}

func (h *CoursesHandler) TestsUpdate(w http.ResponseWriter, r *http.Request) {
	h.postSection(w, r, func(ctx context.Context, user *auth.User, body map[string]any, req *http.Request) (any, error) {
		linkID, _ := tests.ToInt64Public(body["courseTestLinkId"])
		formID, _ := tests.ToInt64Public(body["testFormId"])
		formData, ok := body["form"].(map[string]any)
		if !ok || formData == nil {
			return nil, courses.Err(http.StatusBadRequest, "Не передана form")
		}
		var link map[string]any
		if linkID > 0 {
			link, _ = scanOneMap(ctx, h.Pool, `SELECT * FROM public.course_test_links WHERE id = $1`, linkID)
		} else if formID > 0 {
			link, _ = scanOneMap(ctx, h.Pool, `SELECT * FROM public.course_test_links WHERE test_form_id = $1 LIMIT 1`, formID)
		}
		if link == nil {
			return nil, courses.Err(http.StatusNotFound, "Тест курса не найден")
		}
		svc := h.svc()
		version, _ := svc.GetVersion(ctx, tests.ToInt64Must(link["course_version_id"]))
		if version == nil {
			return nil, courses.Err(http.StatusNotFound, "Версия не найдена")
		}
		if _, _, err := courses.RequireCourseAdmin(ctx, h.Pool, h.Auth, req, tests.ToInt64Must(version["courseId"])); err != nil {
			return nil, err
		}
		if err := courses.AssertVersionEditable(version); err != nil {
			return nil, err
		}
		fid, err := tests.PersistForm(ctx, h.Pool, formData, user.ID)
		if err != nil {
			return nil, err
		}
		if v, ok := body["isRequired"]; ok {
			_, _ = h.Pool.Exec(ctx, `UPDATE public.course_test_links SET is_required = $2, updated_at = now() WHERE id = $1`, link["id"], courses.Bool(v))
		}
		link, _ = scanOneMap(ctx, h.Pool, `SELECT * FROM public.course_test_links WHERE test_form_id = $1 LIMIT 1`, fid)
		form, _ := tests.LoadForm(ctx, h.Pool, fid, user.ID)
		summary, _ := svc.TestFormSummary(ctx, fid)
		return map[string]any{"link": courses.MapTestLink(link, summary), "form": form}, nil
	})
}

func (h *CoursesHandler) TestsDelete(w http.ResponseWriter, r *http.Request) {
	h.postSection(w, r, func(ctx context.Context, user *auth.User, body map[string]any, req *http.Request) (any, error) {
		linkID, err := tests.ToInt64Public(body["courseTestLinkId"])
		if err != nil || linkID <= 0 {
			return nil, courses.Err(http.StatusBadRequest, "Не передан courseTestLinkId")
		}
		link, err := scanOneMap(ctx, h.Pool, `SELECT * FROM public.course_test_links WHERE id = $1`, linkID)
		if errors.Is(err, pgx.ErrNoRows) || link == nil {
			return nil, courses.Err(http.StatusNotFound, "Тест курса не найден")
		}
		svc := h.svc()
		version, _ := svc.GetVersion(ctx, tests.ToInt64Must(link["course_version_id"]))
		if version == nil {
			return nil, courses.Err(http.StatusNotFound, "Версия не найдена")
		}
		if _, _, err := courses.RequireCourseAdmin(ctx, h.Pool, h.Auth, req, tests.ToInt64Must(version["courseId"])); err != nil {
			return nil, err
		}
		if err := courses.AssertVersionEditable(version); err != nil {
			return nil, err
		}
		formID := tests.ToInt64Must(link["test_form_id"])
		_, _ = h.Pool.Exec(ctx, `DELETE FROM public.course_test_links WHERE id = $1`, linkID)
		var status string
		_ = h.Pool.QueryRow(ctx, `SELECT status FROM public.test_forms WHERE id = $1`, formID).Scan(&status)
		if status == "draft" {
			_, _ = h.Pool.Exec(ctx, `DELETE FROM public.test_forms WHERE id = $1`, formID)
		}
		if fmt.Sprint(link["type"]) == "final" {
			_, _ = h.Pool.Exec(ctx, `UPDATE public.course_versions SET require_final_test = false, updated_at = now() WHERE id = $1`, link["course_version_id"])
		}
		return map[string]any{"courseTestLinkId": linkID, "testFormId": formID}, nil
	})
}

func (h *CoursesHandler) AssignPreview(w http.ResponseWriter, r *http.Request) {
	h.postSection(w, r, func(ctx context.Context, _ *auth.User, body map[string]any, _ *http.Request) (any, error) {
		svc := h.svc()
		userIDs := uniqueInt64FromAny(body["userIds"])
		ofoIDs := uniqueInt64FromAny(body["ofoIds"])
		includeChildren := courses.Bool(body["includeChildren"])
		fromOfo, err := svc.ResolveOfoUsers(ctx, ofoIDs, includeChildren)
		if err != nil {
			return nil, err
		}
		seen := map[int64]struct{}{}
		for _, id := range userIDs {
			seen[id] = struct{}{}
		}
		for _, id := range fromOfo {
			seen[id] = struct{}{}
		}
		var recipients []map[string]any
		for id := range seen {
			u, _ := scanOneMap(ctx, h.Pool, `SELECT id, firstname, surname, lastname, role, ofo FROM public.user_info WHERE id = $1 AND status IS TRUE`, id)
			if u == nil {
				continue
			}
			recipients = append(recipients, map[string]any{
				"id": id, "fio": courses.UserFio(u), "role": u["role"], "ofo": u["ofo"],
			})
		}
		return map[string]any{"count": len(recipients), "recipients": recipients, "fromUsers": len(userIDs), "fromOfo": len(fromOfo)}, nil
	})
}

func (h *CoursesHandler) Assign(w http.ResponseWriter, r *http.Request) {
	h.postSection(w, r, func(ctx context.Context, user *auth.User, body map[string]any, req *http.Request) (any, error) {
		svc := h.svc()
		versionID, err := svc.ResolveVersionID(ctx, body)
		if err != nil {
			return nil, err
		}
		version, err := svc.GetVersion(ctx, versionID)
		if err != nil || version == nil {
			return nil, courses.Err(http.StatusNotFound, "Версия не найдена")
		}
		if _, _, err := courses.RequireCourseAdmin(ctx, h.Pool, h.Auth, req, tests.ToInt64Must(version["courseId"])); err != nil {
			return nil, err
		}
		if fmt.Sprint(version["status"]) != "published" {
			return nil, courses.Err(http.StatusConflict, "Назначать можно только опубликованную версию")
		}
		userIDs := uniqueInt64FromAny(body["userIds"])
		ofoIDs := uniqueInt64FromAny(body["ofoIds"])
		if len(userIDs) == 0 && len(ofoIDs) == 0 {
			return nil, courses.Err(http.StatusBadRequest, "Укажите userIds и/или ofoIds")
		}
		includeChildren := courses.Bool(body["includeChildren"])
		startsAt := body["startsAt"]
		deadlineAt := body["deadlineAt"]
		var days *int
		if deadlineAt == nil {
			if v, ok := body["deadlineDays"]; ok {
				d := toIntDefaultMap(v, 0)
				days = &d
			} else if version["defaultDeadlineDays"] != nil {
				d := toIntDefaultMap(version["defaultDeadlineDays"], 0)
				days = &d
			}
		}
		comment := body["comment"]
		var assignmentIDs []int64
		created, skipped := 0, 0
		tx, err := h.Pool.Begin(ctx)
		if err != nil {
			return nil, err
		}
		defer tx.Rollback(ctx)
		resolveDeadline := func(sa any) any {
			if deadlineAt != nil {
				return deadlineAt
			}
			if days == nil || *days <= 0 {
				return nil
			}
			base := time.Now()
			if s, ok := sa.(string); ok && s != "" {
				if t, err := time.Parse(time.RFC3339, s); err == nil {
					base = t
				}
			}
			d := base.Add(time.Duration(*days) * 24 * time.Hour).Format(time.RFC3339)
			return d
		}
		for _, uid := range userIDs {
			da := resolveDeadline(startsAt)
			var aid int64
			_ = tx.QueryRow(ctx, `
				INSERT INTO public.course_assignments (course_version_id, target_type, target_id, starts_at, deadline_at, assigned_by, comment, include_children)
				VALUES ($1,'user',$2,$3,$4,$5,$6,false) RETURNING id`, versionID, uid, startsAt, da, user.ID, comment).Scan(&aid)
			assignmentIDs = append(assignmentIDs, aid)
			var exist int64
			if tx.QueryRow(ctx, `SELECT id FROM public.course_enrollments WHERE user_id=$1 AND course_version_id=$2 AND status <> 'cancelled' LIMIT 1`, uid, versionID).Scan(&exist) == nil {
				skipped++
				continue
			}
			var eid int64
			_ = tx.QueryRow(ctx, `
				INSERT INTO public.course_enrollments (assignment_id, course_version_id, user_id, status, starts_at, deadline_at)
				VALUES ($1,$2,$3,'not_started',$4,$5) RETURNING id`, aid, versionID, uid, startsAt, da).Scan(&eid)
			_ = svc.EnsureTopicProgressRows(ctx, eid, versionID)
			created++
		}
		fromOfo, _ := svc.ResolveOfoUsers(ctx, ofoIDs, includeChildren)
		for _, ofoID := range ofoIDs {
			da := resolveDeadline(startsAt)
			var aid int64
			_ = tx.QueryRow(ctx, `
				INSERT INTO public.course_assignments (course_version_id, target_type, target_id, starts_at, deadline_at, assigned_by, comment, include_children)
				VALUES ($1,'ofo',$2,$3,$4,$5,$6,$7) RETURNING id`, versionID, ofoID, startsAt, da, user.ID, comment, includeChildren).Scan(&aid)
			assignmentIDs = append(assignmentIDs, aid)
			for _, mid := range fromOfo {
				skip := false
				for _, u := range userIDs {
					if u == mid {
						skip = true
						break
					}
				}
				if skip {
					continue
				}
				var exist int64
				if tx.QueryRow(ctx, `SELECT id FROM public.course_enrollments WHERE user_id=$1 AND course_version_id=$2 AND status <> 'cancelled' LIMIT 1`, mid, versionID).Scan(&exist) == nil {
					skipped++
					continue
				}
				var eid int64
				_ = tx.QueryRow(ctx, `
					INSERT INTO public.course_enrollments (assignment_id, course_version_id, user_id, status, starts_at, deadline_at)
					VALUES ($1,$2,$3,'not_started',$4,$5) RETURNING id`, aid, versionID, mid, startsAt, da).Scan(&eid)
				_ = svc.EnsureTopicProgressRows(ctx, eid, versionID)
				created++
			}
		}
		if err := tx.Commit(ctx); err != nil {
			return nil, courses.Err(http.StatusInternalServerError, "Ошибка назначения")
		}
		uid := user.ID
		courses.Audit(ctx, h.Pool, &uid, "course.assign", "course_version", &versionID, map[string]any{"assignments": assignmentIDs, "enrollments": created, "skipped": skipped}, req)
		return map[string]any{"assignmentIds": assignmentIDs, "enrollmentsCreated": created, "skipped": skipped}, nil
	})
}

func (h *CoursesHandler) AdminResults(w http.ResponseWriter, r *http.Request) {
	h.postSection(w, r, func(ctx context.Context, _ *auth.User, body map[string]any, _ *http.Request) (any, error) {
		svc := h.svc()
		svc.MarkOverdue(ctx)
		where := []string{"c.deleted_at IS NULL"}
		args := []any{}
		n := 1
		if v, err := tests.ToInt64Public(body["versionId"]); err == nil && v > 0 {
			where = append(where, fmt.Sprintf("e.course_version_id = $%d", n))
			args = append(args, v)
			n++
		} else if v, err := tests.ToInt64Public(body["courseId"]); err == nil && v > 0 {
			where = append(where, fmt.Sprintf("v.course_id = $%d", n))
			args = append(args, v)
			n++
		}
		if st, ok := body["status"].(string); ok && st != "" {
			where = append(where, fmt.Sprintf("e.status = $%d", n))
			args = append(args, st)
			n++
		}
		if ofoID, err := tests.ToInt64Public(body["ofoId"]); err == nil && ofoID > 0 {
			ids, _ := svc.OfoDescendants(ctx, []int64{ofoID})
			if len(ids) == 0 {
				where = append(where, "1=0")
			} else {
				ph := []string{}
				for _, id := range ids {
					ph = append(ph, fmt.Sprintf("$%d", n))
					args = append(args, fmt.Sprint(id))
					n++
				}
				where = append(where, fmt.Sprintf("u.ofo IN (%s)", strings.Join(ph, ",")))
			}
		}
		if q, ok := body["q"].(string); ok && strings.TrimSpace(q) != "" {
			where = append(where, fmt.Sprintf("(u.surname ILIKE $%d OR u.firstname ILIKE $%d OR u.login ILIKE $%d OR c.title ILIKE $%d)", n, n, n, n))
			args = append(args, "%"+strings.TrimSpace(q)+"%")
			n++
		}
		wsql := strings.Join(where, " AND ")
		limit := toIntDefaultMap(body["limit"], 50)
		if limit > 5000 {
			limit = 5000
		}
		if limit < 1 {
			limit = 50
		}
		offset := toIntDefaultMap(body["offset"], 0)
		if offset < 0 {
			offset = 0
		}
		var agg map[string]any
		row, err := scanOneMap(ctx, h.Pool, fmt.Sprintf(`
			SELECT COUNT(*) AS total,
			       COUNT(*) FILTER (WHERE e.status = 'not_started') AS not_started,
			       COUNT(*) FILTER (WHERE e.status = 'in_progress') AS in_progress,
			       COUNT(*) FILTER (WHERE e.status = 'completed') AS completed,
			       COUNT(*) FILTER (WHERE e.status = 'failed') AS failed,
			       COUNT(*) FILTER (WHERE e.status = 'overdue') AS overdue,
			       COUNT(*) FILTER (WHERE e.status = 'cancelled') AS cancelled,
			       AVG(e.final_score) FILTER (WHERE e.final_score IS NOT NULL) AS avg_score
			FROM public.course_enrollments e
			JOIN public.course_versions v ON v.id = e.course_version_id
			JOIN public.course_courses c ON c.id = v.course_id
			JOIN public.user_info u ON u.id = e.user_id WHERE %s`, wsql), args...)
		if err == nil {
			agg = map[string]any{
				"total": toIntDefaultMap(row["total"], 0), "notStarted": toIntDefaultMap(row["not_started"], 0),
				"inProgress": toIntDefaultMap(row["in_progress"], 0), "completed": toIntDefaultMap(row["completed"], 0),
				"failed": toIntDefaultMap(row["failed"], 0), "overdue": toIntDefaultMap(row["overdue"], 0),
				"cancelled": toIntDefaultMap(row["cancelled"], 0), "avgScore": row["avg_score"],
			}
		}
		rows, err := h.Pool.Query(ctx, fmt.Sprintf(`
			SELECT e.*, c.id AS course_id, c.title AS course_title, v.version_number,
			       u.firstname, u.surname, u.lastname, u.role, u.ofo, u.login, o.name AS ofo_name
			FROM public.course_enrollments e
			JOIN public.course_versions v ON v.id = e.course_version_id
			JOIN public.course_courses c ON c.id = v.course_id
			JOIN public.user_info u ON u.id = e.user_id
			LEFT JOIN public.ofo_unit o ON u.ofo ~ '^[0-9]+$' AND o.id = CAST(u.ofo AS integer)
			WHERE %s ORDER BY e.assigned_at DESC LIMIT %d OFFSET %d`, wsql, limit, offset), args...)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		var items []map[string]any
		for rows.Next() {
			er, err := scanRowToMap(rows)
			if err != nil {
				return nil, err
			}
			eid := tests.ToInt64Must(er["id"])
			prog := svc.EnrollmentProgress(ctx, eid)
			items = append(items, map[string]any{
				"enrollment": courses.MapEnrollment(er, prog, nil),
				"courseId": er["course_id"], "courseTitle": er["course_title"], "versionNumber": er["version_number"],
				"user": map[string]any{
					"id": er["user_id"], "fio": courses.UserFio(er), "login": er["login"],
					"role": er["role"], "ofo": er["ofo"], "ofoName": er["ofo_name"],
				},
			})
		}
		return map[string]any{"aggregates": agg, "items": items, "limit": limit, "offset": offset}, nil
	})
}

func (h *CoursesHandler) AdminParticipant(w http.ResponseWriter, r *http.Request) {
	h.postSection(w, r, func(ctx context.Context, _ *auth.User, body map[string]any, _ *http.Request) (any, error) {
		enrollmentID, err := tests.ToInt64Public(body["enrollmentId"])
		if err != nil || enrollmentID <= 0 {
			return nil, courses.Err(http.StatusBadRequest, "Не передан enrollmentId")
		}
		svc := h.svc()
		enr, err := scanOneMap(ctx, h.Pool, `SELECT * FROM public.course_enrollments WHERE id = $1`, enrollmentID)
		if errors.Is(err, pgx.ErrNoRows) || enr == nil {
			return nil, courses.Err(http.StatusNotFound, "Запись не найдена")
		}
		versionID := tests.ToInt64Must(enr["course_version_id"])
		version, _ := svc.GetVersion(ctx, versionID)
		course, _ := svc.GetCourse(ctx, tests.ToInt64Must(version["courseId"]), true)
		progress := svc.EnrollmentProgress(ctx, enrollmentID)
		assembled, _ := svc.AssembleVersion(ctx, versionID, false)

		u, _ := scanOneMap(ctx, h.Pool, `
			SELECT id, firstname, surname, lastname, login, role, ofo, email, phone
			FROM public.user_info WHERE id = $1`, enr["user_id"])
		var ofoID any
		var ofoName any
		rawOfo := fmt.Sprint(u["ofo"])
		if rawOfo != "" && rawOfo != "<nil>" && rawOfo != "-1" {
			if id, e := tests.ToInt64Public(u["ofo"]); e == nil && id > 0 {
				ofoID = id
				if o, _ := scanOneMap(ctx, h.Pool, `SELECT name FROM public.ofo_unit WHERE id = $1`, id); o != nil {
					ofoName = o["name"]
				}
			}
		}

		topicRows, _ := h.Pool.Query(ctx, `
			SELECT p.*, t.title FROM public.course_topic_progress p
			JOIN public.course_topics t ON t.id = p.topic_id
			WHERE p.enrollment_id = $1 ORDER BY t.sort_order, t.id`, enrollmentID)
		var topics []map[string]any
		for topicRows != nil && topicRows.Next() {
			p, err := scanRowToMap(topicRows)
			if err != nil {
				continue
			}
			topics = append(topics, map[string]any{
				"topicId": tests.ToInt64Must(p["topic_id"]), "title": fmt.Sprint(p["title"]),
				"status": fmt.Sprint(p["status"]), "activeSeconds": toIntDefaultMap(p["active_seconds"], 0),
				"openedAt": p["opened_at"], "completedAt": p["completed_at"],
			})
		}
		if topicRows != nil {
			topicRows.Close()
		}
		if topics == nil {
			topics = []map[string]any{}
		}

		matRows, _ := h.Pool.Query(ctx, `
			SELECT mp.*, m.title, m.topic_id FROM public.course_material_progress mp
			JOIN public.course_materials m ON m.id = mp.material_id
			WHERE mp.enrollment_id = $1`, enrollmentID)
		var materials []map[string]any
		for matRows != nil && matRows.Next() {
			p, err := scanRowToMap(matRows)
			if err != nil {
				continue
			}
			materials = append(materials, map[string]any{
				"materialId": tests.ToInt64Must(p["material_id"]), "topicId": tests.ToInt64Must(p["topic_id"]),
				"title": fmt.Sprint(p["title"]), "status": fmt.Sprint(p["status"]),
				"activeSeconds": toIntDefaultMap(p["active_seconds"], 0),
				"openedAt": p["opened_at"], "completedAt": p["completed_at"],
			})
		}
		if matRows != nil {
			matRows.Close()
		}
		if materials == nil {
			materials = []map[string]any{}
		}

		linkRows, _ := h.Pool.Query(ctx, `
			SELECT l.id, l.type, l.topic_id, l.is_required, l.test_form_id,
			       f.title AS form_title, t.title AS topic_title, t.sort_order
			FROM public.course_test_links l
			JOIN public.test_forms f ON f.id = l.test_form_id
			LEFT JOIN public.course_topics t ON t.id = l.topic_id AND t.deleted_at IS NULL
			WHERE l.course_version_id = $1
			ORDER BY CASE WHEN l.type = 'final' THEN 1 ELSE 0 END, t.sort_order NULLS LAST, t.id, l.id`, versionID)
		var testItems []map[string]any
		for linkRows != nil && linkRows.Next() {
			link, err := scanRowToMap(linkRows)
			if err != nil {
				continue
			}
			linkID := tests.ToInt64Must(link["id"])
			best, _ := scanOneMap(ctx, h.Pool, `
				SELECT a.id AS attempt_id, a.score, a.passed, a.status, a.started_at, a.finished_at
				FROM public.course_test_attempt_links tal
				JOIN public.test_attempts a ON a.id = tal.test_attempt_id
				WHERE tal.enrollment_id = $1 AND tal.course_test_link_id = $2
				ORDER BY
				  CASE WHEN a.passed IS TRUE THEN 0 WHEN a.status IN ('finished','completed') THEN 1 ELSE 2 END,
				  a.score DESC NULLS LAST,
				  a.finished_at DESC NULLS LAST,
				  a.id DESC
				LIMIT 1`, enrollmentID, linkID)
			var attemptsCount int
			_ = h.Pool.QueryRow(ctx, `
				SELECT COUNT(*) FROM public.course_test_attempt_links
				WHERE enrollment_id = $1 AND course_test_link_id = $2`, enrollmentID, linkID).Scan(&attemptsCount)

			typ := fmt.Sprint(link["type"])
			topicTitle := ""
			if link["topic_title"] != nil {
				topicTitle = strings.TrimSpace(fmt.Sprint(link["topic_title"]))
			}
			formTitle := strings.TrimSpace(fmt.Sprint(link["form_title"]))
			if formTitle == "" || formTitle == "<nil>" {
				if typ == "final" {
					formTitle = "Итоговый тест"
				} else if topicTitle != "" {
					formTitle = "Тест: " + topicTitle
				} else {
					formTitle = "Тест"
				}
			}
			item := map[string]any{
				"courseTestLinkId": linkID, "type": typ,
				"topicId": nil, "topicTitle": nil,
				"title": formTitle, "isRequired": courses.Bool(link["is_required"]),
				"attemptsCount": attemptsCount,
				"attemptId": nil, "score": nil, "passed": nil,
				"status": "not_started", "startedAt": nil, "finishedAt": nil,
				"testFormId": tests.ToInt64Must(link["test_form_id"]),
			}
			if tid, e := tests.ToInt64Public(link["topic_id"]); e == nil && tid > 0 {
				item["topicId"] = tid
			}
			if topicTitle != "" && topicTitle != "<nil>" {
				item["topicTitle"] = topicTitle
			}
			if best != nil {
				item["attemptId"] = tests.ToInt64Must(best["attempt_id"])
				item["score"] = best["score"]
				if best["passed"] != nil {
					item["passed"] = courses.Bool(best["passed"])
				}
				item["status"] = fmt.Sprint(best["status"])
				item["startedAt"] = best["started_at"]
				item["finishedAt"] = best["finished_at"]
			}
			testItems = append(testItems, item)
		}
		if linkRows != nil {
			linkRows.Close()
		}
		if testItems == nil {
			testItems = []map[string]any{}
		}

		comp, _ := scanOneMap(ctx, h.Pool, `
			SELECT * FROM public.course_completions WHERE enrollment_id = $1 ORDER BY id DESC LIMIT 1`, enrollmentID)
		var completion any
		if comp != nil {
			completion = map[string]any{
				"id": tests.ToInt64Must(comp["id"]),
				"completionNumber": toIntDefaultMap(comp["completion_number"], 0),
				"completedAt": comp["completed_at"],
				"totalActiveSeconds": toIntDefaultMap(comp["total_active_seconds"], 0),
				"finalScore": comp["final_score"],
				"passed": courses.Bool(comp["passed"]),
			}
		}

		return map[string]any{
			"enrollment": courses.MapEnrollment(enr, progress, course),
			"version":    assembled,
			"user": map[string]any{
				"id": tests.ToInt64Must(u["id"]), "fio": courses.UserFio(u),
				"login": fmt.Sprint(u["login"]), "role": fmt.Sprint(u["role"]),
				"ofo": u["ofo"], "ofoId": ofoID, "ofoName": ofoName,
				"email": u["email"], "phone": u["phone"],
			},
			"topics": topics, "materials": materials, "tests": testItems,
			"completion": completion,
		}, nil
	})
}

func (h *CoursesHandler) AdminAttemptAnswers(w http.ResponseWriter, r *http.Request) {
	h.postSection(w, r, func(ctx context.Context, _ *auth.User, body map[string]any, req *http.Request) (any, error) {
		enrollmentID, err := tests.ToInt64Public(body["enrollmentId"])
		attemptID, err2 := tests.ToInt64Public(body["attemptId"])
		if err != nil || err2 != nil || enrollmentID <= 0 || attemptID <= 0 {
			return nil, courses.Err(http.StatusBadRequest, "Нужны enrollmentId и attemptId")
		}
		link, err := scanOneMap(ctx, h.Pool, `
			SELECT tal.course_test_link_id, l.test_form_id, l.course_version_id, l.type, f.title AS form_title
			FROM public.course_test_attempt_links tal
			JOIN public.course_test_links l ON l.id = tal.course_test_link_id
			JOIN public.test_forms f ON f.id = l.test_form_id
			WHERE tal.enrollment_id = $1 AND tal.test_attempt_id = $2`, enrollmentID, attemptID)
		if errors.Is(err, pgx.ErrNoRows) || link == nil {
			return nil, courses.Err(http.StatusNotFound, "Попытка не связана с этим назначением")
		}
		version, _ := h.svc().GetVersion(ctx, tests.ToInt64Must(link["course_version_id"]))
		if version == nil {
			return nil, courses.Err(http.StatusNotFound, "Версия курса не найдена")
		}
		if _, _, err := courses.RequireCourseAdmin(ctx, h.Pool, h.Auth, req, tests.ToInt64Must(version["courseId"])); err != nil {
			return nil, err
		}
		formID := tests.ToInt64Must(link["test_form_id"])
		attempt, answers, err := buildAttemptReview(ctx, h.Pool, formID, attemptID)
		if err != nil {
			return nil, err
		}
		return map[string]any{
			"attempt": attempt,
			"answers": answers,
			"test": map[string]any{
				"courseTestLinkId": tests.ToInt64Must(link["course_test_link_id"]),
				"type":             fmt.Sprint(link["type"]),
				"title":            fmt.Sprint(link["form_title"]),
				"testFormId":       formID,
			},
		}, nil
	})
}

func (h *CoursesHandler) EnrollmentReset(w http.ResponseWriter, r *http.Request) {
	h.postSection(w, r, func(ctx context.Context, user *auth.User, body map[string]any, req *http.Request) (any, error) {
		enrollmentID, err := tests.ToInt64Public(body["enrollmentId"])
		if err != nil || enrollmentID <= 0 {
			return nil, courses.Err(http.StatusBadRequest, "Не передан enrollmentId")
		}
		enr, err := scanOneMap(ctx, h.Pool, `SELECT * FROM public.course_enrollments WHERE id = $1`, enrollmentID)
		if errors.Is(err, pgx.ErrNoRows) || enr == nil {
			return nil, courses.Err(http.StatusNotFound, "Запись не найдена")
		}
		svc := h.svc()
		version, _ := svc.GetVersion(ctx, tests.ToInt64Must(enr["course_version_id"]))
		if version == nil {
			return nil, courses.Err(http.StatusNotFound, "Версия курса не найдена")
		}
		if _, _, err := courses.RequireCourseAdmin(ctx, h.Pool, h.Auth, req, tests.ToInt64Must(version["courseId"])); err != nil {
			return nil, err
		}
		tx, err := h.Pool.Begin(ctx)
		if err != nil {
			return nil, err
		}
		defer tx.Rollback(ctx)
		attRows, _ := tx.Query(ctx, `SELECT test_attempt_id FROM public.course_test_attempt_links WHERE enrollment_id = $1`, enrollmentID)
		var attemptIDs []int64
		for attRows != nil && attRows.Next() {
			var id int64
			if attRows.Scan(&id) == nil {
				attemptIDs = append(attemptIDs, id)
			}
		}
		if attRows != nil {
			attRows.Close()
		}
		_, _ = tx.Exec(ctx, `DELETE FROM public.course_test_attempt_links WHERE enrollment_id = $1`, enrollmentID)
		for _, id := range attemptIDs {
			_, _ = tx.Exec(ctx, `DELETE FROM public.test_attempts WHERE id = $1`, id)
		}
		_, _ = tx.Exec(ctx, `DELETE FROM public.course_learning_sessions WHERE enrollment_id = $1`, enrollmentID)
		_, _ = tx.Exec(ctx, `DELETE FROM public.course_material_progress WHERE enrollment_id = $1`, enrollmentID)
		_, _ = tx.Exec(ctx, `DELETE FROM public.course_topic_progress WHERE enrollment_id = $1`, enrollmentID)
		_, _ = tx.Exec(ctx, `DELETE FROM public.course_completions WHERE enrollment_id = $1`, enrollmentID)
		_, _ = tx.Exec(ctx, `
			UPDATE public.course_enrollments SET status='not_started', started_at=NULL, completed_at=NULL,
			 last_activity_at=NULL, final_score=NULL, updated_at=now() WHERE id=$1`, enrollmentID)
		if err := tx.Commit(ctx); err != nil {
			return nil, courses.Err(http.StatusInternalServerError, "Не удалось обнулить результат")
		}
		_ = svc.EnsureTopicProgressRows(ctx, enrollmentID, tests.ToInt64Must(enr["course_version_id"]))
		_ = svc.RecalculateLocks(ctx, enrollmentID)
		enr, _ = scanOneMap(ctx, h.Pool, `SELECT * FROM public.course_enrollments WHERE id = $1`, enrollmentID)
		return map[string]any{"enrollment": courses.MapEnrollment(enr, svc.EnrollmentProgress(ctx, enrollmentID), nil)}, nil
	})
}

func (h *CoursesHandler) ForMe(w http.ResponseWriter, r *http.Request) {
	h.post(w, r, func(ctx context.Context, user *auth.User, _ map[string]any, _ *http.Request) (any, error) {
		svc := h.svc()
		svc.MarkOverdue(ctx)
		rows, err := h.Pool.Query(ctx, `
			SELECT e.*, c.id AS course_id, c.title AS course_title, c.category,
			       v.short_description, v.cover_url, v.version_number, v.status AS version_status
			FROM public.course_enrollments e
			JOIN public.course_versions v ON v.id = e.course_version_id
			JOIN public.course_courses c ON c.id = v.course_id
			WHERE e.user_id = $1 AND e.status <> 'cancelled' AND c.deleted_at IS NULL
			ORDER BY e.assigned_at DESC`, user.ID)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		groups := map[string][]map[string]any{"active": {}, "completed": {}, "overdue": {}, "failed": {}}
		for rows.Next() {
			er, err := scanRowToMap(rows)
			if err != nil {
				return nil, err
			}
			eid := tests.ToInt64Must(er["id"])
			prog := svc.EnrollmentProgress(ctx, eid)
			item := map[string]any{
				"enrollment": courses.MapEnrollment(er, prog, nil),
				"course": map[string]any{
					"id": er["course_id"], "title": er["course_title"], "category": er["category"],
					"shortDescription": er["short_description"], "coverUrl": er["cover_url"],
					"versionNumber": er["version_number"], "versionStatus": er["version_status"],
				},
			}
			switch fmt.Sprint(er["status"]) {
			case "completed":
				groups["completed"] = append(groups["completed"], item)
			case "overdue":
				groups["overdue"] = append(groups["overdue"], item)
			case "failed":
				groups["failed"] = append(groups["failed"], item)
			default:
				groups["active"] = append(groups["active"], item)
			}
		}
		return groups, nil
	})
}

func (h *CoursesHandler) EnrollmentGet(w http.ResponseWriter, r *http.Request) {
	h.post(w, r, func(ctx context.Context, user *auth.User, body map[string]any, _ *http.Request) (any, error) {
		enrollmentID, err := tests.ToInt64Public(body["enrollmentId"])
		if err != nil || enrollmentID <= 0 {
			return nil, courses.Err(http.StatusBadRequest, "Не передан enrollmentId")
		}
		svc := h.svc()
		enr, err := svc.RequireEnrollmentAccess(ctx, enrollmentID, user, true)
		if err != nil {
			return nil, err
		}
		_ = svc.RecalculateLocks(ctx, enrollmentID)
		version, _ := svc.GetVersion(ctx, tests.ToInt64Must(enr["course_version_id"]))
		course, _ := svc.GetCourse(ctx, tests.ToInt64Must(version["courseId"]), false)
		assembled, _ := svc.AssembleVersion(ctx, tests.ToInt64Must(enr["course_version_id"]), false)
		progress := svc.EnrollmentProgress(ctx, enrollmentID)
		h.mergeProgress(ctx, enrollmentID, assembled)
		return map[string]any{
			"enrollment": courses.MapEnrollment(enr, progress, course),
			"version": assembled, "nextAction": progress["nextAction"],
		}, nil
	})
}

func (h *CoursesHandler) mergeProgress(ctx context.Context, enrollmentID int64, assembled map[string]any) {
	if assembled == nil {
		return
	}
	tpRows, _ := h.Pool.Query(ctx, `SELECT topic_id, status, active_seconds, last_material_id FROM public.course_topic_progress WHERE enrollment_id = $1`, enrollmentID)
	topicStatus := map[int64]map[string]any{}
	for tpRows != nil && tpRows.Next() {
		var tid int64
		var status string
		var active int
		var lastMat *int64
		if tpRows.Scan(&tid, &status, &active, &lastMat) == nil {
			topicStatus[tid] = map[string]any{"status": status, "activeSeconds": active, "lastMaterialId": lastMat}
		}
	}
	if tpRows != nil {
		tpRows.Close()
	}
	mpRows, _ := h.Pool.Query(ctx, `SELECT material_id, status, active_seconds FROM public.course_material_progress WHERE enrollment_id = $1`, enrollmentID)
	matStatus := map[int64]map[string]any{}
	for mpRows != nil && mpRows.Next() {
		var mid int64
		var status string
		var active int
		if mpRows.Scan(&mid, &status, &active) == nil {
			matStatus[mid] = map[string]any{"status": status, "activeSeconds": active}
		}
	}
	if mpRows != nil {
		mpRows.Close()
	}
	topics, _ := assembled["topics"].([]map[string]any)
	for _, t := range topics {
		tid := tests.ToInt64Must(t["id"])
		if p, ok := topicStatus[tid]; ok {
			t["progress"] = p
		} else {
			t["progress"] = map[string]any{"status": "locked", "activeSeconds": 0, "lastMaterialId": nil}
		}
		mats, _ := t["materials"].([]map[string]any)
		for _, m := range mats {
			mid := tests.ToInt64Must(m["id"])
			if p, ok := matStatus[mid]; ok {
				m["progress"] = p
			} else {
				m["progress"] = map[string]any{"status": "not_started", "activeSeconds": 0}
			}
		}
	}
}

func (h *CoursesHandler) Start(w http.ResponseWriter, r *http.Request) {
	h.post(w, r, func(ctx context.Context, user *auth.User, body map[string]any, req *http.Request) (any, error) {
		enrollmentID, err := tests.ToInt64Public(body["enrollmentId"])
		if err != nil || enrollmentID <= 0 {
			return nil, courses.Err(http.StatusBadRequest, "Не передан enrollmentId")
		}
		svc := h.svc()
		enr, err := svc.RequireEnrollmentAccess(ctx, enrollmentID, user, false)
		if err != nil {
			return nil, err
		}
		st := fmt.Sprint(enr["status"])
		if st == "completed" || st == "cancelled" || st == "failed" {
			return nil, courses.Err(http.StatusConflict, "Курс недоступен")
		}
		if sa := enr["starts_at"]; sa != nil {
			if t, ok := sa.(time.Time); ok && t.After(time.Now()) {
				return nil, courses.Err(http.StatusConflict, "Курс ещё не начался")
			}
		}
		_, _ = h.Pool.Exec(ctx, `
			UPDATE public.course_enrollments SET status='in_progress', started_at=COALESCE(started_at, now()),
			 last_activity_at=now(), updated_at=now() WHERE id=$1`, enrollmentID)
		_ = svc.EnsureTopicProgressRows(ctx, enrollmentID, tests.ToInt64Must(enr["course_version_id"]))
		_ = svc.RecalculateLocks(ctx, enrollmentID)
		enr, _ = scanOneMap(ctx, h.Pool, `SELECT * FROM public.course_enrollments WHERE id = $1`, enrollmentID)
		return map[string]any{"enrollment": courses.MapEnrollment(enr, svc.EnrollmentProgress(ctx, enrollmentID), nil), "nextAction": svc.NextAction(ctx, enrollmentID)}, nil
	})
}

func (h *CoursesHandler) TopicGet(w http.ResponseWriter, r *http.Request) {
	h.post(w, r, func(ctx context.Context, user *auth.User, body map[string]any, _ *http.Request) (any, error) {
		enrollmentID, err := tests.ToInt64Public(body["enrollmentId"])
		topicID, err2 := tests.ToInt64Public(body["topicId"])
		if err != nil || err2 != nil || enrollmentID <= 0 || topicID <= 0 {
			return nil, courses.Err(http.StatusBadRequest, "Нужны enrollmentId и topicId")
		}
		svc := h.svc()
		enr, err := svc.RequireEnrollmentAccess(ctx, enrollmentID, user, true)
		if err != nil {
			return nil, err
		}
		_ = svc.EnsureTopicProgressRows(ctx, enrollmentID, tests.ToInt64Must(enr["course_version_id"]))
		_ = svc.RecalculateLocks(ctx, enrollmentID)
		topic, err := svc.TopicVersionRow(ctx, topicID)
		if errors.Is(err, pgx.ErrNoRows) || topic == nil || tests.ToInt64Must(topic["course_version_id"]) != tests.ToInt64Must(enr["course_version_id"]) {
			return nil, courses.Err(http.StatusNotFound, "Тема не найдена")
		}
		isReview := fmt.Sprint(enr["status"]) == "completed" || fmt.Sprint(enr["status"]) == "failed"
		prog, _ := scanOneMap(ctx, h.Pool, `SELECT * FROM public.course_topic_progress WHERE enrollment_id = $1 AND topic_id = $2`, enrollmentID, topicID)
		if !isReview && (prog == nil || fmt.Sprint(prog["status"]) == "locked") {
			return nil, courses.Err(http.StatusForbidden, "Тема ещё недоступна")
		}
		if !isReview && prog != nil && fmt.Sprint(prog["status"]) == "available" {
			_, _ = h.Pool.Exec(ctx, `UPDATE public.course_topic_progress SET status='in_progress', opened_at=COALESCE(opened_at, now()), updated_at=now() WHERE id=$1`, prog["id"])
		}
		assembled, _ := svc.AssembleVersion(ctx, tests.ToInt64Must(enr["course_version_id"]), true)
		var topicData map[string]any
		if topics, ok := assembled["topics"].([]map[string]any); ok {
			for _, t := range topics {
				if tests.ToInt64Must(t["id"]) == topicID {
					topicData = t
					break
				}
			}
		}
		if topicData == nil {
			return nil, courses.Err(http.StatusNotFound, "Тема не найдена")
		}
		h.mergeProgress(ctx, enrollmentID, map[string]any{"topics": []map[string]any{topicData}})
		prog, _ = scanOneMap(ctx, h.Pool, `SELECT * FROM public.course_topic_progress WHERE enrollment_id = $1 AND topic_id = $2`, enrollmentID, topicID)
		topicData["progress"] = map[string]any{
			"status": fmt.Sprint(prog["status"]), "activeSeconds": prog["active_seconds"],
			"openedAt": prog["opened_at"], "completedAt": prog["completed_at"],
		}
		return map[string]any{
			"topic": topicData, "nextAction": svc.NextAction(ctx, enrollmentID),
			"nextTopic": nil, "reviewMode": isReview, "enrollmentStatus": enr["status"],
		}, nil
	})
}

func (h *CoursesHandler) MaterialOpen(w http.ResponseWriter, r *http.Request) {
	h.post(w, r, func(ctx context.Context, user *auth.User, body map[string]any, _ *http.Request) (any, error) {
		enrollmentID, err := tests.ToInt64Public(body["enrollmentId"])
		materialID, err2 := tests.ToInt64Public(body["materialId"])
		if err != nil || err2 != nil || enrollmentID <= 0 || materialID <= 0 {
			return nil, courses.Err(http.StatusBadRequest, "Нужны enrollmentId и materialId")
		}
		svc := h.svc()
		enr, err := svc.RequireEnrollmentAccess(ctx, enrollmentID, user, false)
		if err != nil {
			return nil, err
		}
		m, err := svc.MaterialVersionRow(ctx, materialID)
		if errors.Is(err, pgx.ErrNoRows) || m == nil || tests.ToInt64Must(m["course_version_id"]) != tests.ToInt64Must(enr["course_version_id"]) {
			return nil, courses.Err(http.StatusNotFound, "Материал не найден")
		}
		topicID := tests.ToInt64Must(m["topic_id"])
		_ = svc.EnsureTopicProgressRows(ctx, enrollmentID, tests.ToInt64Must(enr["course_version_id"]))
		_ = svc.RecalculateLocks(ctx, enrollmentID)
		isReview := fmt.Sprint(enr["status"]) == "completed" || fmt.Sprint(enr["status"]) == "failed"
		var ts string
		_ = h.Pool.QueryRow(ctx, `SELECT status FROM public.course_topic_progress WHERE enrollment_id=$1 AND topic_id=$2`, enrollmentID, topicID).Scan(&ts)
		if !isReview && ts == "locked" {
			return nil, courses.Err(http.StatusForbidden, "Тема ещё недоступна")
		}
		var sessionID int64
		tx, err := h.Pool.Begin(ctx)
		if err != nil {
			return nil, err
		}
		defer tx.Rollback(ctx)
		_, _ = tx.Exec(ctx, `
			INSERT INTO public.course_material_progress (enrollment_id, material_id, status, opened_at)
			VALUES ($1,$2,'in_progress',now())
			ON CONFLICT (enrollment_id, material_id) DO UPDATE SET
			  status = CASE WHEN course_material_progress.status = 'completed' THEN 'completed' ELSE 'in_progress' END,
			  opened_at = COALESCE(course_material_progress.opened_at, now()), updated_at = now()`, enrollmentID, materialID)
		_, _ = tx.Exec(ctx, `
			UPDATE public.course_topic_progress SET
			  status = CASE WHEN status='completed' THEN 'completed' WHEN status='locked' THEN 'locked' ELSE 'in_progress' END,
			  opened_at = COALESCE(opened_at, now()), last_material_id = $3, updated_at = now()
			WHERE enrollment_id = $1 AND topic_id = $2`, enrollmentID, topicID, materialID)
		ip := clientIP(r)
		ua := r.UserAgent()
		_ = tx.QueryRow(ctx, `
			INSERT INTO public.course_learning_sessions (enrollment_id, topic_id, material_id, user_id, ip_address, user_agent)
			VALUES ($1,$2,$3,$4,$5,$6) RETURNING id`, enrollmentID, topicID, materialID, user.ID, ip, ua).Scan(&sessionID)
		_, _ = tx.Exec(ctx, `
			UPDATE public.course_enrollments SET status = CASE WHEN status='not_started' THEN 'in_progress' ELSE status END,
			 started_at = COALESCE(started_at, now()), last_activity_at = now(), updated_at = now() WHERE id = $1`, enrollmentID)
		if err := tx.Commit(ctx); err != nil {
			return nil, courses.Err(http.StatusInternalServerError, "Ошибка открытия материала")
		}
		var fileURL any
		if m["file_url"] != nil && fmt.Sprint(m["file_url"]) != "" {
			fileURL = fmt.Sprintf("/api/course_file.php?materialId=%d", materialID)
		}
		return map[string]any{
			"material": map[string]any{
				"id": materialID, "topicId": topicID, "type": m["type"], "title": m["title"],
				"description": m["description"], "contentHtml": m["content_html"], "fileUrl": fileURL,
				"storageKey": m["file_url"], "externalUrl": m["external_url"], "mimeType": m["mime_type"],
				"minimumActiveSeconds": m["minimum_active_seconds"], "isRequired": courses.Bool(m["is_required"]),
			},
			"sessionId": sessionID,
		}, nil
	})
}

func (h *CoursesHandler) MaterialHeartbeat(w http.ResponseWriter, r *http.Request) {
	h.post(w, r, func(ctx context.Context, user *auth.User, body map[string]any, _ *http.Request) (any, error) {
		enrollmentID, err := tests.ToInt64Public(body["enrollmentId"])
		materialID, err2 := tests.ToInt64Public(body["materialId"])
		if err != nil || err2 != nil || enrollmentID <= 0 || materialID <= 0 {
			return nil, courses.Err(http.StatusBadRequest, "Нужны enrollmentId и materialId")
		}
		svc := h.svc()
		if _, err := svc.RequireEnrollmentAccess(ctx, enrollmentID, user, false); err != nil {
			return nil, err
		}
		m, err := svc.MaterialVersionRow(ctx, materialID)
		if err != nil || m == nil {
			return nil, courses.Err(http.StatusNotFound, "Материал не найден")
		}
		topicID := tests.ToInt64Must(m["topic_id"])
		if gap, ok := body["clientGapSec"]; ok && toIntDefaultMap(gap, 0) > 90 {
			return map[string]any{"ignored": true, "reason": "client_gap", "addedSeconds": 0}, nil
		}
		sessionID, _ := tests.ToInt64Public(body["sessionId"])
		var session map[string]any
		if sessionID > 0 {
			session, _ = scanOneMap(ctx, h.Pool, `SELECT * FROM public.course_learning_sessions WHERE id=$1 AND enrollment_id=$2 AND finished_at IS NULL`, sessionID, enrollmentID)
		} else {
			session, _ = scanOneMap(ctx, h.Pool, `SELECT * FROM public.course_learning_sessions WHERE enrollment_id=$1 AND material_id=$2 AND finished_at IS NULL ORDER BY id DESC LIMIT 1`, enrollmentID, materialID)
		}
		if session == nil {
			return nil, courses.Err(http.StatusNotFound, "Сессия обучения не найдена")
		}
		lastHB, _ := session["last_heartbeat_at"].(time.Time)
		delta := int(time.Since(lastHB).Seconds())
		if delta < 0 {
			delta = 0
		}
		if delta > 90 {
			_, _ = h.Pool.Exec(ctx, `UPDATE public.course_learning_sessions SET last_heartbeat_at = now() WHERE id = $1`, session["id"])
			return map[string]any{"ignored": true, "reason": "server_gap", "addedSeconds": 0, "sessionId": session["id"]}, nil
		}
		add := delta
		if add > 60 {
			add = 60
		}
		if add < 1 {
			return map[string]any{"ignored": false, "addedSeconds": 0, "sessionId": session["id"], "activeSeconds": session["active_seconds"], "status": "in_progress"}, nil
		}
		tx, err := h.Pool.Begin(ctx)
		if err != nil {
			return nil, err
		}
		defer tx.Rollback(ctx)
		_, _ = tx.Exec(ctx, `UPDATE public.course_learning_sessions SET active_seconds = active_seconds + $2, last_heartbeat_at = now() WHERE id = $1`, session["id"], add)
		_, _ = tx.Exec(ctx, `
			INSERT INTO public.course_material_progress (enrollment_id, material_id, status, opened_at, active_seconds)
			VALUES ($1,$2,'in_progress',now(),$3)
			ON CONFLICT (enrollment_id, material_id) DO UPDATE SET active_seconds = course_material_progress.active_seconds + $3, updated_at = now()`,
			enrollmentID, materialID, add)
		_, _ = tx.Exec(ctx, `UPDATE public.course_topic_progress SET active_seconds = active_seconds + $3, updated_at = now() WHERE enrollment_id = $1 AND topic_id = $2`, enrollmentID, topicID, add)
		_, _ = tx.Exec(ctx, `UPDATE public.course_enrollments SET last_activity_at = now(), updated_at = now() WHERE id = $1`, enrollmentID)
		if err := tx.Commit(ctx); err != nil {
			return nil, courses.Err(http.StatusInternalServerError, "Ошибка heartbeat")
		}
		row, _ := scanOneMap(ctx, h.Pool, `SELECT active_seconds, status FROM public.course_material_progress WHERE enrollment_id=$1 AND material_id=$2`, enrollmentID, materialID)
		return map[string]any{
			"ignored": false, "addedSeconds": add, "sessionId": session["id"],
			"activeSeconds": row["active_seconds"], "status": row["status"],
			"minimumActiveSeconds": m["minimum_active_seconds"],
		}, nil
	})
}

func (h *CoursesHandler) MaterialComplete(w http.ResponseWriter, r *http.Request) {
	h.post(w, r, func(ctx context.Context, user *auth.User, body map[string]any, req *http.Request) (any, error) {
		enrollmentID, err := tests.ToInt64Public(body["enrollmentId"])
		materialID, err2 := tests.ToInt64Public(body["materialId"])
		if err != nil || err2 != nil || enrollmentID <= 0 || materialID <= 0 {
			return nil, courses.Err(http.StatusBadRequest, "Нужны enrollmentId и materialId")
		}
		svc := h.svc()
		if _, err := svc.RequireEnrollmentAccess(ctx, enrollmentID, user, false); err != nil {
			return nil, err
		}
		m, err := svc.MaterialVersionRow(ctx, materialID)
		if err != nil || m == nil {
			return nil, courses.Err(http.StatusNotFound, "Материал не найден")
		}
		topicID := tests.ToInt64Must(m["topic_id"])
		prog, _ := scanOneMap(ctx, h.Pool, `SELECT * FROM public.course_material_progress WHERE enrollment_id=$1 AND material_id=$2`, enrollmentID, materialID)
		min := toIntDefaultMap(m["minimum_active_seconds"], 0)
		active := toIntDefaultMap(prog["active_seconds"], 0)
		if min > 0 && active < min {
			return nil, courses.Err(http.StatusConflict, fmt.Sprintf("Недостаточно активного времени (нужно %d сек, есть %d)", min, active))
		}
		tx, err := h.Pool.Begin(ctx)
		if err != nil {
			return nil, err
		}
		defer tx.Rollback(ctx)
		_, _ = tx.Exec(ctx, `
			INSERT INTO public.course_material_progress (enrollment_id, material_id, status, opened_at, completed_at)
			VALUES ($1,$2,'completed',now(),now())
			ON CONFLICT (enrollment_id, material_id) DO UPDATE SET status='completed', completed_at=COALESCE(course_material_progress.completed_at, now()), updated_at=now()`,
			enrollmentID, materialID)
		_, _ = tx.Exec(ctx, `UPDATE public.course_learning_sessions SET finished_at=now() WHERE enrollment_id=$1 AND material_id=$2 AND finished_at IS NULL`, enrollmentID, materialID)
		if svc.CheckTopicComplete(ctx, enrollmentID, topicID) {
			_, _ = tx.Exec(ctx, `UPDATE public.course_topic_progress SET status='completed', completed_at=COALESCE(completed_at, now()), updated_at=now() WHERE enrollment_id=$1 AND topic_id=$2`, enrollmentID, topicID)
			_ = svc.RecalculateLocks(ctx, enrollmentID)
		}
		_, _ = tx.Exec(ctx, `UPDATE public.course_enrollments SET last_activity_at=now(), updated_at=now() WHERE id=$1`, enrollmentID)
		if err := tx.Commit(ctx); err != nil {
			return nil, courses.Err(http.StatusInternalServerError, "Ошибка завершения материала")
		}
		_, _ = svc.TryCompleteEnrollment(ctx, enrollmentID, req)
		return map[string]any{
			"materialId": materialID, "nextAction": svc.NextAction(ctx, enrollmentID),
			"progress": svc.EnrollmentProgress(ctx, enrollmentID),
		}, nil
	})
}

func (h *CoursesHandler) NextAction(w http.ResponseWriter, r *http.Request) {
	h.post(w, r, func(ctx context.Context, user *auth.User, body map[string]any, _ *http.Request) (any, error) {
		enrollmentID, err := tests.ToInt64Public(body["enrollmentId"])
		if err != nil || enrollmentID <= 0 {
			return nil, courses.Err(http.StatusBadRequest, "Не передан enrollmentId")
		}
		svc := h.svc()
		if _, err := svc.RequireEnrollmentAccess(ctx, enrollmentID, user, true); err != nil {
			return nil, err
		}
		progress := svc.EnrollmentProgress(ctx, enrollmentID)
		return map[string]any{"nextAction": progress["nextAction"], "progress": progress}, nil
	})
}

func (h *CoursesHandler) Result(w http.ResponseWriter, r *http.Request) {
	h.post(w, r, func(ctx context.Context, user *auth.User, body map[string]any, req *http.Request) (any, error) {
		enrollmentID, _ := tests.ToInt64Public(body["enrollmentId"])
		completionID, _ := tests.ToInt64Public(body["completionId"])
		svc := h.svc()
		var row map[string]any
		var err error
		if completionID > 0 {
			row, err = scanOneMap(ctx, h.Pool, `SELECT * FROM public.course_completions WHERE id = $1`, completionID)
		} else if enrollmentID > 0 {
			if _, err := svc.RequireEnrollmentAccess(ctx, enrollmentID, user, true); err != nil {
				return nil, err
			}
			// Досоздаём completion, если все условия выполнены (в т.ч. без итогового теста).
			_, _ = svc.TryCompleteEnrollment(ctx, enrollmentID, req)
			row, err = scanOneMap(ctx, h.Pool, `SELECT * FROM public.course_completions WHERE enrollment_id = $1 ORDER BY id DESC LIMIT 1`, enrollmentID)
		} else {
			return nil, courses.Err(http.StatusBadRequest, "Укажите enrollmentId или completionId")
		}
		if errors.Is(err, pgx.ErrNoRows) || row == nil {
			return nil, courses.Err(http.StatusNotFound, "Результат недоступен")
		}
		if !auth.IsAdmin(user) && tests.ToInt64Must(row["user_id"]) != user.ID {
			return nil, courses.Err(http.StatusForbidden, "Нет доступа")
		}
		completion := mapCompletion(row)
		generateCertificate := false
		requireFinalTest := false
		versionID := tests.ToInt64Must(row["course_version_id"])
		if version, verr := svc.GetVersion(ctx, versionID); verr == nil && version != nil {
			generateCertificate = courses.Bool(version["generateCertificate"])
			requireFinalTest = courses.Bool(version["requireFinalTest"])
		}
		if snap, ok := completion["resultSnapshot"].(map[string]any); ok {
			if !generateCertificate {
				generateCertificate = courses.Bool(snap["generateCertificate"])
			}
			if title, ok := snap["courseTitle"]; ok {
				completion["courseTitle"] = title
			}
			if fio, ok := snap["userFio"]; ok {
				completion["userFio"] = fio
			}
			if ofo, ok := snap["ofoName"]; ok {
				completion["ofoName"] = ofo
			}
		}
		courseTitle := fmt.Sprint(completion["courseTitle"])
		if courseTitle == "" || courseTitle == "<nil>" {
			if c, cerr := svc.GetCourse(ctx, tests.ToInt64Must(row["course_id"]), true); cerr == nil && c != nil {
				courseTitle = fmt.Sprint(c["title"])
				completion["courseTitle"] = courseTitle
			}
		}
		return map[string]any{
			"completion":          completion,
			"generateCertificate": generateCertificate,
			"requireFinalTest":    requireFinalTest,
			"course":              map[string]any{"id": row["course_id"], "title": courseTitle},
		}, nil
	})
}

func mapCompletion(row map[string]any) map[string]any {
	return map[string]any{
		"id": row["id"], "enrollmentId": row["enrollment_id"], "courseId": row["course_id"],
		"courseVersionId": row["course_version_id"], "completionNumber": row["completion_number"],
		"assignedAt": row["assigned_at"], "startedAt": row["started_at"], "completedAt": row["completed_at"],
		"totalActiveSeconds": row["total_active_seconds"], "finalScore": row["final_score"],
		"passed": courses.Bool(row["passed"]), "resultSnapshot": normalizeJSONMap(row["result_snapshot"]),
	}
}

func normalizeJSONMap(v any) map[string]any {
	switch t := v.(type) {
	case map[string]any:
		return t
	case []byte:
		var out map[string]any
		if json.Unmarshal(t, &out) == nil {
			return out
		}
	case string:
		var out map[string]any
		if json.Unmarshal([]byte(t), &out) == nil {
			return out
		}
	}
	return map[string]any{}
}
