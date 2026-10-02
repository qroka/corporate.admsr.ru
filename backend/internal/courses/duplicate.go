package courses

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/jackc/pgx/v5"

	"corporate.admsr.ru/backend/internal/auth"
)

// DuplicateCourse — глубокая копия курса в новый черновик (версия 1): темы, материалы и
// клоны тестов (Q-02, перенос courses_duplicate.php). Копируется текущая версия курса;
// «Полное описание» не копируется (Q-09). Файлы материалов не дублируются — копия ссылается
// на те же файлы, как и в PHP.
//
// В отличие от PHP-версии клон теста переносит и роли вариантов (`role`, `target_option_id` —
// вопросы «соответствие» и «классификация», V13); PHP копировал их как обычные варианты.
func (s *Service) DuplicateCourse(ctx context.Context, courseID int64, user *auth.User, r *http.Request) (map[string]any, error) {
	var title string
	var category *string
	var srcVersionID *int64
	err := s.Pool.QueryRow(ctx, `
		SELECT title, category, current_version_id FROM public.course_courses
		WHERE id = $1 AND deleted_at IS NULL`, courseID).Scan(&title, &category, &srcVersionID)
	if errors.Is(err, pgx.ErrNoRows) || (err == nil && srcVersionID == nil) {
		return nil, Err(http.StatusNotFound, "Курс или текущая версия не найдены")
	}
	if err != nil {
		return nil, err
	}

	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var newCourseID int64
	if err := tx.QueryRow(ctx, `
		INSERT INTO public.course_courses (owner_id, title, category)
		VALUES ($1, $2, $3) RETURNING id`, user.ID, "Копия: "+title, category).Scan(&newCourseID); err != nil {
		return nil, err
	}

	var newVersionID int64
	if err := tx.QueryRow(ctx, `
		INSERT INTO public.course_versions (
			course_id, version_number, status, short_description, cover_url, sequential_progress,
			completion_rule, default_deadline_days, final_passing_score, require_final_test,
			generate_certificate, created_by
		)
		SELECT $1, 1, 'draft', short_description, cover_url, sequential_progress,
		       completion_rule, default_deadline_days, final_passing_score, require_final_test,
		       generate_certificate, $2
		FROM public.course_versions WHERE id = $3
		RETURNING id`, newCourseID, user.ID, *srcVersionID).Scan(&newVersionID); err != nil {
		return nil, err
	}
	if _, err := tx.Exec(ctx,
		`UPDATE public.course_courses SET current_version_id = $1, updated_at = now() WHERE id = $2`,
		newVersionID, newCourseID); err != nil {
		return nil, err
	}

	// Темы: сначала читаем список целиком, потом вставляем — на одной транзакции нельзя
	// держать открытый курсор и выполнять другие запросы.
	topicRows, err := tx.Query(ctx, `
		SELECT id FROM public.course_topics
		WHERE course_version_id = $1 AND deleted_at IS NULL ORDER BY sort_order, id`, *srcVersionID)
	if err != nil {
		return nil, err
	}
	var oldTopicIDs []int64
	for topicRows.Next() {
		var id int64
		if err := topicRows.Scan(&id); err != nil {
			topicRows.Close()
			return nil, err
		}
		oldTopicIDs = append(oldTopicIDs, id)
	}
	topicRows.Close()
	if err := topicRows.Err(); err != nil {
		return nil, err
	}

	topicMap := make(map[int64]int64, len(oldTopicIDs))
	for _, oldID := range oldTopicIDs {
		var newID int64
		if err := tx.QueryRow(ctx, `
			INSERT INTO public.course_topics (
				course_version_id, title, description, sort_order, is_required,
				minimum_active_seconds, completion_rule
			)
			SELECT $1, title, description, sort_order, is_required, minimum_active_seconds, completion_rule
			FROM public.course_topics WHERE id = $2
			RETURNING id`, newVersionID, oldID).Scan(&newID); err != nil {
			return nil, err
		}
		topicMap[oldID] = newID
		if _, err := tx.Exec(ctx, `
			INSERT INTO public.course_materials (
				topic_id, type, title, description, content_html, file_url, external_url,
				mime_type, file_size, original_filename, sort_order, is_required, minimum_active_seconds
			)
			SELECT $1, type, title, description, content_html, file_url, external_url,
			       mime_type, file_size, original_filename, sort_order, is_required, minimum_active_seconds
			FROM public.course_materials
			WHERE topic_id = $2 AND deleted_at IS NULL
			ORDER BY sort_order, id`, newID, oldID); err != nil {
			return nil, err
		}
	}

	type testLink struct {
		topicID  *int64
		formID   int64
		kind     string
		required bool
		sort     int
	}
	linkRows, err := tx.Query(ctx, `
		SELECT topic_id, test_form_id, type, is_required, sort_order
		FROM public.course_test_links WHERE course_version_id = $1 ORDER BY sort_order, id`, *srcVersionID)
	if err != nil {
		return nil, err
	}
	var links []testLink
	for linkRows.Next() {
		var l testLink
		if err := linkRows.Scan(&l.topicID, &l.formID, &l.kind, &l.required, &l.sort); err != nil {
			linkRows.Close()
			return nil, err
		}
		links = append(links, l)
	}
	linkRows.Close()
	if err := linkRows.Err(); err != nil {
		return nil, err
	}

	for _, l := range links {
		var newTopic *int64
		switch l.kind {
		case "topic":
			// тест удалённой темы не копируем
			if l.topicID == nil {
				continue
			}
			nt, ok := topicMap[*l.topicID]
			if !ok {
				continue
			}
			newTopic = &nt
		case "final":
			// topic_id у итогового теста пустой
		default:
			continue
		}
		newFormID, err := cloneTestForm(ctx, tx, l.formID, user.ID)
		if err != nil {
			return nil, err
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO public.course_test_links (course_version_id, topic_id, test_form_id, type, is_required, sort_order)
			VALUES ($1, $2, $3, $4, $5, $6)`, newVersionID, newTopic, newFormID, l.kind, l.required, l.sort); err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	uid := user.ID
	Audit(ctx, s.Pool, &uid, "course.duplicate", "course", &newCourseID, map[string]any{
		"sourceCourseId": courseID, "sourceVersionId": *srcVersionID, "versionId": newVersionID,
	}, r)

	course, err := s.GetCourse(ctx, newCourseID, false)
	if err != nil {
		return nil, err
	}
	version, err := s.AssembleVersion(ctx, newVersionID, true)
	if err != nil {
		return nil, err
	}
	return map[string]any{"course": course, "version": version}, nil
}

// cloneTestForm копирует форму теста с вопросами и вариантами внутри текущей транзакции.
// Новая форма — черновик, приватная, без доступа по ссылке.
func cloneTestForm(ctx context.Context, tx pgx.Tx, formID, ownerID int64) (int64, error) {
	var newID int64
	err := tx.QueryRow(ctx, `
		INSERT INTO public.test_forms (
			status, owner_id, kind, visibility, title, description, completion_message,
			shuffle, shuffle_options, show_progress, free_navigation, anonymous,
			allow_change_answer, live_results, allow_revote, notify_creator,
			use_passing_score, passing_score, show_correct_answers, restrict_by_ofo,
			use_time_limit, time_limit_sec, limit_attempts, attempts,
			use_start, starts_at, use_end, ends_at, show_result,
			access_by_link, link_access
		)
		SELECT 'draft', $2, COALESCE(NULLIF(kind, ''), 'test'), 'private', title, description, completion_message,
		       shuffle, shuffle_options, show_progress, free_navigation, anonymous,
		       allow_change_answer, live_results, allow_revote, notify_creator,
		       use_passing_score, passing_score, show_correct_answers, restrict_by_ofo,
		       use_time_limit, time_limit_sec, limit_attempts, attempts,
		       use_start, starts_at, use_end, ends_at, show_result,
		       false, COALESCE(link_access, 'any')
		FROM public.test_forms WHERE id = $1
		RETURNING id`, formID, ownerID).Scan(&newID)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, fmt.Errorf("тест для копирования не найден (id %d)", formID)
	}
	if err != nil {
		return 0, err
	}

	qRows, err := tx.Query(ctx, `SELECT id FROM public.test_questions WHERE form_id = $1 ORDER BY position, id`, formID)
	if err != nil {
		return 0, err
	}
	var questionIDs []int64
	for qRows.Next() {
		var id int64
		if err := qRows.Scan(&id); err != nil {
			qRows.Close()
			return 0, err
		}
		questionIDs = append(questionIDs, id)
	}
	qRows.Close()
	if err := qRows.Err(); err != nil {
		return 0, err
	}

	type optLink struct{ oldID, oldTarget int64 }
	optionMap := map[int64]int64{}
	var targets []optLink
	for _, oldQ := range questionIDs {
		var newQ int64
		if err := tx.QueryRow(ctx, `
			INSERT INTO public.test_questions (
				form_id, position, type, title, hint, required,
				scale_min, scale_max, scale_min_label, scale_max_label, correct_value
			)
			SELECT $1, position, type, title, hint, required,
			       scale_min, scale_max, scale_min_label, scale_max_label, correct_value
			FROM public.test_questions WHERE id = $2
			RETURNING id`, newID, oldQ).Scan(&newQ); err != nil {
			return 0, err
		}
		oRows, err := tx.Query(ctx, `SELECT id, target_option_id FROM public.test_options WHERE question_id = $1 ORDER BY position, id`, oldQ)
		if err != nil {
			return 0, err
		}
		type opt struct {
			id     int64
			target *int64
		}
		var opts []opt
		for oRows.Next() {
			var o opt
			if err := oRows.Scan(&o.id, &o.target); err != nil {
				oRows.Close()
				return 0, err
			}
			opts = append(opts, o)
		}
		oRows.Close()
		if err := oRows.Err(); err != nil {
			return 0, err
		}
		for _, o := range opts {
			var newO int64
			if err := tx.QueryRow(ctx, `
				INSERT INTO public.test_options (question_id, position, text, is_correct, role)
				SELECT $1, position, text, is_correct, role FROM public.test_options WHERE id = $2
				RETURNING id`, newQ, o.id).Scan(&newO); err != nil {
				return 0, err
			}
			optionMap[o.id] = newO
			if o.target != nil {
				targets = append(targets, optLink{oldID: o.id, oldTarget: *o.target})
			}
		}
	}
	// Связи «элемент → цель» (соответствие / классификация) — после того, как все варианты скопированы.
	for _, t := range targets {
		newOpt, okO := optionMap[t.oldID]
		newTarget, okT := optionMap[t.oldTarget]
		if !okO || !okT {
			continue
		}
		if _, err := tx.Exec(ctx, `UPDATE public.test_options SET target_option_id = $1 WHERE id = $2`, newTarget, newOpt); err != nil {
			return 0, err
		}
	}
	return newID, nil
}
