package handlers

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"

	"corporate.admsr.ru/backend/internal/auth"
	"corporate.admsr.ru/backend/internal/courses"
	"corporate.admsr.ru/backend/internal/tests"
)

// Эндпоинты курсов, перенесённые из PHP последними (Q-02, ADR-047). SPA их не вызывает —
// это возможности API без экрана в интерфейсе. Права — как у остальных админских эндпоинтов
// курсов: секция `courses` + доступ к категории конкретного курса (в PHP категория здесь
// не проверялась).

// courseOfVersion возвращает courseId версии или 404.
func (h *CoursesHandler) courseOfVersion(ctx context.Context, versionID int64) (map[string]any, error) {
	version, err := h.svc().GetVersion(ctx, versionID)
	if err != nil || version == nil {
		return nil, courses.Err(http.StatusNotFound, "Версия не найдена")
	}
	return version, nil
}

// Archive — POST /api/courses_archive.php {courseId|versionId}: опубликованная версия → archived.
func (h *CoursesHandler) Archive(w http.ResponseWriter, r *http.Request) {
	h.postSection(w, r, func(ctx context.Context, user *auth.User, body map[string]any, req *http.Request) (any, error) {
		svc := h.svc()
		versionID, err := svc.ResolveVersionID(ctx, body)
		if err != nil {
			return nil, err
		}
		version, err := h.courseOfVersion(ctx, versionID)
		if err != nil {
			return nil, err
		}
		courseID := tests.ToInt64Must(version["courseId"])
		if _, _, err := courses.RequireCourseAdmin(ctx, h.Pool, h.Auth, req, courseID); err != nil {
			return nil, err
		}
		if fmt.Sprint(version["status"]) != "published" {
			return nil, courses.Err(http.StatusConflict, "Архивировать можно только опубликованную версию")
		}
		if _, err := h.Pool.Exec(ctx, `
			UPDATE public.course_versions SET status = 'archived', archived_at = now(), updated_at = now()
			WHERE id = $1`, versionID); err != nil {
			return nil, err
		}
		uid := user.ID
		courses.Audit(ctx, h.Pool, &uid, "course.version.archive", "course_version", &versionID,
			map[string]any{"courseId": courseID}, req)
		assembled, err := svc.AssembleVersion(ctx, versionID, true)
		if err != nil {
			return nil, err
		}
		return map[string]any{"version": assembled}, nil
	})
}

// Duplicate — POST /api/courses_duplicate.php {courseId}: копия курса в новый черновик.
func (h *CoursesHandler) Duplicate(w http.ResponseWriter, r *http.Request) {
	h.postSection(w, r, func(ctx context.Context, user *auth.User, body map[string]any, req *http.Request) (any, error) {
		courseID, err := tests.ToInt64Public(body["courseId"])
		if err != nil || courseID <= 0 {
			return nil, courses.Err(http.StatusBadRequest, "Не передан courseId")
		}
		if _, _, err := courses.RequireCourseAdmin(ctx, h.Pool, h.Auth, req, courseID); err != nil {
			return nil, err
		}
		out, err := h.svc().DuplicateCourse(ctx, courseID, user, req)
		if err != nil {
			var he courses.HTTPError
			if !errors.As(err, &he) {
				log.Printf("courses: duplicate %d: %v", courseID, err)
				return nil, courses.Err(http.StatusInternalServerError, "Ошибка дублирования")
			}
			return nil, err
		}
		return out, nil
	})
}

// Readiness — POST /api/courses_readiness.php {courseId|versionId}: {ready, errors, warnings}.
// Та же проверка, что блокирует публикацию (`courses.VersionReadiness`), поэтому ответ
// «готов» гарантирует, что публикация не откажет по этой причине. PHP дополнительно разбирал
// ответы каждого вопроса (cs_form_answer_errors) — в Go этого нет ни здесь, ни при публикации.
func (h *CoursesHandler) Readiness(w http.ResponseWriter, r *http.Request) {
	h.postSection(w, r, func(ctx context.Context, _ *auth.User, body map[string]any, req *http.Request) (any, error) {
		svc := h.svc()
		versionID, err := svc.ResolveVersionID(ctx, body)
		if err != nil {
			return nil, err
		}
		version, err := h.courseOfVersion(ctx, versionID)
		if err != nil {
			return nil, err
		}
		if _, _, err := courses.RequireCourseAdmin(ctx, h.Pool, h.Auth, req, tests.ToInt64Must(version["courseId"])); err != nil {
			return nil, err
		}
		assembled, err := svc.AssembleVersion(ctx, versionID, false)
		if err != nil {
			return nil, err
		}
		ready, problems, warnings := courses.VersionReadiness(assembled)
		if problems == nil {
			problems = []string{}
		}
		if warnings == nil {
			warnings = []string{}
		}
		return map[string]any{"ready": ready, "errors": problems, "warnings": warnings}, nil
	})
}

// MaterialsOrder — POST /api/course_materials_order.php {topicId, materialIds[]}.
func (h *CoursesHandler) MaterialsOrder(w http.ResponseWriter, r *http.Request) {
	h.postSection(w, r, func(ctx context.Context, user *auth.User, body map[string]any, req *http.Request) (any, error) {
		topicID, err := tests.ToInt64Public(body["topicId"])
		if err != nil || topicID <= 0 {
			return nil, courses.Err(http.StatusBadRequest, "Нужны topicId и materialIds")
		}
		if _, ok := body["materialIds"].([]any); !ok {
			return nil, courses.Err(http.StatusBadRequest, "Нужны topicId и materialIds")
		}
		svc := h.svc()
		topic, err := svc.TopicVersionRow(ctx, topicID)
		if errors.Is(err, pgx.ErrNoRows) || (err == nil && topic == nil) {
			return nil, courses.Err(http.StatusNotFound, "Тема не найдена")
		}
		if err != nil {
			return nil, err
		}
		if _, _, err := courses.RequireCourseAdmin(ctx, h.Pool, h.Auth, req, tests.ToInt64Must(topic["course_id"])); err != nil {
			return nil, err
		}
		if st := fmt.Sprint(topic["version_status"]); st != "draft" && st != "published" {
			return nil, courses.Err(http.StatusConflict, "Версия недоступна для редактирования (только черновик/опубликовано)")
		}
		ids := uniqueInt64FromAny(body["materialIds"])
		tx, err := h.Pool.Begin(ctx)
		if err != nil {
			return nil, err
		}
		defer tx.Rollback(ctx)
		for i, id := range ids {
			if _, err := tx.Exec(ctx, `
				UPDATE public.course_materials SET sort_order = $1, updated_at = now()
				WHERE id = $2 AND topic_id = $3 AND deleted_at IS NULL`, i, id, topicID); err != nil {
				return nil, courses.Err(http.StatusInternalServerError, "Ошибка сортировки")
			}
		}
		if err := tx.Commit(ctx); err != nil {
			return nil, courses.Err(http.StatusInternalServerError, "Ошибка сортировки")
		}
		uid := user.ID
		courses.Audit(ctx, h.Pool, &uid, "course.material.reorder", "course_topic", &topicID,
			map[string]any{"materialIds": ids}, req)
		return map[string]any{"topicId": topicID, "materialIds": ids}, nil
	})
}

// AssignmentCancel — POST /api/course_assignment_cancel.php {assignmentId}: отменяет назначение
// целиком и записи в нём, которые ещё не начаты. Начатые и пройденные записи не трогает; точечно
// снять курс с одного сотрудника — course_enrollment_cancel.php. Отменённое постоянное назначение
// («Все сотрудники», ОФО) перестаёт выдавать курс тем, кто придёт позже.
func (h *CoursesHandler) AssignmentCancel(w http.ResponseWriter, r *http.Request) {
	h.postSection(w, r, func(ctx context.Context, user *auth.User, body map[string]any, req *http.Request) (any, error) {
		assignmentID, err := tests.ToInt64Public(body["assignmentId"])
		if err != nil || assignmentID <= 0 {
			return nil, courses.Err(http.StatusBadRequest, "Не передан assignmentId")
		}
		var versionID int64
		var cancelledAt *time.Time
		err = h.Pool.QueryRow(ctx,
			`SELECT course_version_id, cancelled_at FROM public.course_assignments WHERE id = $1`,
			assignmentID).Scan(&versionID, &cancelledAt)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, courses.Err(http.StatusNotFound, "Назначение не найдено")
		}
		if err != nil {
			return nil, err
		}
		version, err := h.courseOfVersion(ctx, versionID)
		if err != nil {
			return nil, err
		}
		if _, _, err := courses.RequireCourseAdmin(ctx, h.Pool, h.Auth, req, tests.ToInt64Must(version["courseId"])); err != nil {
			return nil, err
		}
		if cancelledAt != nil {
			return nil, courses.Err(http.StatusConflict, "Уже отменено")
		}
		tx, err := h.Pool.Begin(ctx)
		if err != nil {
			return nil, err
		}
		defer tx.Rollback(ctx)
		if _, err := tx.Exec(ctx,
			`UPDATE public.course_assignments SET cancelled_at = now() WHERE id = $1 AND cancelled_at IS NULL`,
			assignmentID); err != nil {
			return nil, courses.Err(http.StatusInternalServerError, "Ошибка отмены")
		}
		tag, err := tx.Exec(ctx, `
			UPDATE public.course_enrollments SET status = 'cancelled', updated_at = now()
			WHERE assignment_id = $1 AND status = 'not_started'`, assignmentID)
		if err != nil {
			return nil, courses.Err(http.StatusInternalServerError, "Ошибка отмены")
		}
		if err := tx.Commit(ctx); err != nil {
			return nil, courses.Err(http.StatusInternalServerError, "Ошибка отмены")
		}
		cancelled := tag.RowsAffected()
		uid := user.ID
		courses.Audit(ctx, h.Pool, &uid, "course.assignment.cancel", "course_assignment", &assignmentID,
			map[string]any{"cancelledEnrollments": cancelled}, req)
		return map[string]any{"assignmentId": assignmentID, "cancelledEnrollments": cancelled}, nil
	})
}

// AssignmentsList — POST /api/course_assignments_list.php {courseId|versionId?, activeOnly?}.
// С courseId/versionId проверяется доступ к курсу; без них возвращаются назначения только тех
// категорий, к которым у пользователя есть доступ (в PHP фильтра по категориям не было).
func (h *CoursesHandler) AssignmentsList(w http.ResponseWriter, r *http.Request) {
	h.postSection(w, r, func(ctx context.Context, user *auth.User, body map[string]any, req *http.Request) (any, error) {
		q := `SELECT a.id, a.course_version_id, v.course_id, c.title, c.category, a.target_type, a.target_id,
		             a.starts_at, a.deadline_at, a.assigned_by, a.comment, a.include_children,
		             a.created_at, a.cancelled_at,
		             (SELECT COUNT(*) FROM public.course_enrollments e
		              WHERE e.assignment_id = a.id AND e.status <> 'cancelled') AS enrollment_count
		      FROM public.course_assignments a
		      JOIN public.course_versions v ON v.id = a.course_version_id
		      JOIN public.course_courses c ON c.id = v.course_id
		      WHERE 1=1`
		args := []any{}
		if vid, err := tests.ToInt64Public(body["versionId"]); err == nil && vid > 0 {
			version, err := h.courseOfVersion(ctx, vid)
			if err != nil {
				return nil, err
			}
			if _, _, err := courses.RequireCourseAdmin(ctx, h.Pool, h.Auth, req, tests.ToInt64Must(version["courseId"])); err != nil {
				return nil, err
			}
			args = append(args, vid)
			q += fmt.Sprintf(" AND a.course_version_id = $%d", len(args))
		} else if cid, err := tests.ToInt64Public(body["courseId"]); err == nil && cid > 0 {
			if _, _, err := courses.RequireCourseAdmin(ctx, h.Pool, h.Auth, req, cid); err != nil {
				return nil, err
			}
			args = append(args, cid)
			q += fmt.Sprintf(" AND v.course_id = $%d", len(args))
		}
		if courses.Bool(body["activeOnly"]) {
			q += " AND a.cancelled_at IS NULL"
		}
		q += " ORDER BY a.created_at DESC LIMIT 500"

		rows, err := h.Pool.Query(ctx, q, args...)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		items := []map[string]any{}
		for rows.Next() {
			var id, versionID, courseID, targetID, assignedBy, enrollmentCount int64
			var title, targetType string
			var category, comment *string
			var includeChildren bool
			var startsAt, deadlineAt, createdAt, cancelledAt *time.Time
			if err := rows.Scan(&id, &versionID, &courseID, &title, &category, &targetType, &targetID,
				&startsAt, &deadlineAt, &assignedBy, &comment, &includeChildren,
				&createdAt, &cancelledAt, &enrollmentCount); err != nil {
				return nil, err
			}
			if category != nil && !auth.CanEditCourseCategory(ctx, h.Pool, user, category) {
				continue
			}
			items = append(items, map[string]any{
				"id": id, "courseVersionId": versionID, "courseId": courseID, "courseTitle": title,
				"targetType": targetType, "targetId": targetID, "startsAt": startsAt, "deadlineAt": deadlineAt,
				"assignedBy": assignedBy, "comment": comment, "includeChildren": includeChildren,
				"createdAt": createdAt, "cancelledAt": cancelledAt, "enrollmentCount": enrollmentCount,
			})
		}
		if err := rows.Err(); err != nil {
			return nil, err
		}
		return map[string]any{"items": items}, nil
	})
}

// History — POST /api/course_history.php: завершённые курсы текущего пользователя.
func (h *CoursesHandler) History(w http.ResponseWriter, r *http.Request) {
	h.post(w, r, func(ctx context.Context, user *auth.User, _ map[string]any, _ *http.Request) (any, error) {
		rows, err := h.Pool.Query(ctx, `
			SELECT cc.id, cc.enrollment_id, cc.course_id, c.title, c.category, cc.course_version_id,
			       cc.completion_number, cc.assigned_at, cc.started_at, cc.completed_at,
			       cc.total_active_seconds, cc.final_score::float8, cc.passed
			FROM public.course_completions cc
			JOIN public.course_courses c ON c.id = cc.course_id
			WHERE cc.user_id = $1
			ORDER BY cc.completed_at DESC`, user.ID)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		items := []map[string]any{}
		for rows.Next() {
			var id, enrollmentID, courseID, versionID int64
			var number, activeSeconds int
			var title string
			var category *string
			var assignedAt, startedAt, completedAt *time.Time
			var finalScore *float64
			var passed bool
			if err := rows.Scan(&id, &enrollmentID, &courseID, &title, &category, &versionID, &number,
				&assignedAt, &startedAt, &completedAt, &activeSeconds, &finalScore, &passed); err != nil {
				return nil, err
			}
			items = append(items, map[string]any{
				"id": id, "enrollmentId": enrollmentID, "courseId": courseID, "courseTitle": title,
				"category": category, "courseVersionId": versionID, "completionNumber": number,
				"assignedAt": assignedAt, "startedAt": startedAt, "completedAt": completedAt,
				"totalActiveSeconds": activeSeconds, "finalScore": finalScore, "passed": passed,
			})
		}
		if err := rows.Err(); err != nil {
			return nil, err
		}
		return map[string]any{"items": items}, nil
	})
}
