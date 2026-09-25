package courses

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"corporate.admsr.ru/backend/internal/auth"
	"corporate.admsr.ru/backend/internal/tests"
)

type HTTPError struct {
	Status  int
	Message string
}

func (e HTTPError) Error() string { return e.Message }

func Err(status int, msg string) HTTPError { return HTTPError{Status: status, Message: msg} }

type Service struct {
	Pool        *pgxpool.Pool
	UploadRoot  string
}

func Bool(v any) bool { return tests.Bool(v) }

func Audit(ctx context.Context, pool *pgxpool.Pool, userID *int64, action, entityType string, entityID *int64, payload map[string]any, r *http.Request) {
	var ip, ua *string
	if r != nil {
		if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
			parts := strings.Split(xff, ",")
			s := strings.TrimSpace(parts[0])
			if s != "" {
				ip = &s
			}
		} else if r.RemoteAddr != "" {
			s := r.RemoteAddr
			ip = &s
		}
		if u := r.UserAgent(); u != "" {
			if len([]rune(u)) > 500 {
				u = string([]rune(u)[:500])
			}
			ua = &u
		}
	}
	b, _ := json.Marshal(payload)
	_, _ = pool.Exec(ctx, `
		INSERT INTO public.course_audit_logs (user_id, action, entity_type, entity_id, payload, ip_address, user_agent)
		VALUES ($1,$2,$3,$4,$5::jsonb,$6,$7)`,
		userID, action, entityType, entityID, string(b), ip, ua)
}

var (
	onAttrRe    = regexp.MustCompile(`(?i)\son\w+\s*=\s*("[^"]*"|'[^']*'|[^\s>]+)`)
	jsURLRe     = regexp.MustCompile(`(?i)\s(href|src)\s*=\s*("|')?\s*javascript:[^"'\s>]*`)
	iframeRe    = regexp.MustCompile(`(?i)<\/?iframe\b[^>]*>`)
	allowedTags = "<p><br><strong><b><em><i><u><s><ul><ol><li><a><h1><h2><h3><h4><img><table><thead><tbody><tr><td><th><blockquote><code><pre><span><div><hr>"
)

func SanitizeHTML(html *string) string {
	if html == nil || strings.TrimSpace(*html) == "" {
		return ""
	}
	s := *html
	// strip disallowed tags roughly
	out := s
	for _, tag := range []string{"script", "iframe", "object", "embed"} {
		re := regexp.MustCompile(`(?i)<\/?` + tag + `\b[^>]*>`)
		out = re.ReplaceAllString(out, "")
	}
	out = onAttrRe.ReplaceAllString(out, "")
	out = jsURLRe.ReplaceAllString(out, "")
	out = iframeRe.ReplaceAllString(out, "")
	_ = allowedTags
	return out
}

func MapVersionRow(row map[string]any) map[string]any {
	return map[string]any{
		"id":                  tests.ToInt64Must(row["id"]),
		"courseId":            tests.ToInt64Must(row["course_id"]),
		"versionNumber":       toInt(row["version_number"]),
		"status":              fmt.Sprint(row["status"]),
		"shortDescription":    strOr(row["short_description"], ""),
		"fullDescription":     strOr(row["full_description"], ""),
		"coverUrl":            row["cover_url"],
		"sequentialProgress":  Bool(row["sequential_progress"]),
		"completionRule":      strOr(row["completion_rule"], "all_required"),
		"defaultDeadlineDays": intPtr(row["default_deadline_days"]),
		"finalPassingScore":   floatPtr(row["final_passing_score"]),
		"requireFinalTest":    Bool(row["require_final_test"]),
		"generateCertificate": Bool(row["generate_certificate"]),
		"createdBy":           intPtr(row["created_by"]),
		"createdAt":           row["created_at"],
		"updatedAt":           row["updated_at"],
		"publishedAt":         row["published_at"],
		"archivedAt":          row["archived_at"],
	}
}

func MapCourseRow(row map[string]any) map[string]any {
	return map[string]any{
		"id":               tests.ToInt64Must(row["id"]),
		"ownerId":          tests.ToInt64Must(row["owner_id"]),
		"title":            fmt.Sprint(row["title"]),
		"category":         row["category"],
		"currentVersionId": intPtr(row["current_version_id"]),
		"createdAt":        row["created_at"],
		"updatedAt":        row["updated_at"],
		"deletedAt":        row["deleted_at"],
	}
}

func (s *Service) GetCourse(ctx context.Context, courseID int64, includeDeleted bool) (map[string]any, error) {
	q := `SELECT * FROM public.course_courses WHERE id = $1`
	if !includeDeleted {
		q += ` AND deleted_at IS NULL`
	}
	row, err := scanOne(ctx, s.Pool, q, courseID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	course := MapCourseRow(row)
	course["currentVersion"] = nil
	if cv := intPtr(row["current_version_id"]); cv != nil {
		if v, err := s.GetVersion(ctx, *cv); err != nil {
			return nil, err
		} else if v != nil {
			course["currentVersion"] = map[string]any{
				"id": v["id"], "versionNumber": v["versionNumber"], "status": v["status"],
				"shortDescription": v["shortDescription"], "coverUrl": v["coverUrl"],
				"publishedAt": v["publishedAt"], "requireFinalTest": v["requireFinalTest"],
				"generateCertificate": v["generateCertificate"],
				"sequentialProgress": v["sequentialProgress"],
			}
		}
	}
	return course, nil
}

func (s *Service) GetVersion(ctx context.Context, versionID int64) (map[string]any, error) {
	row, err := scanOne(ctx, s.Pool, `SELECT * FROM public.course_versions WHERE id = $1`, versionID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return MapVersionRow(row), nil
}

func RequireCourseAdmin(ctx context.Context, pool *pgxpool.Pool, authSvc *auth.Service, r *http.Request, courseID int64) (*auth.User, map[string]any, error) {
	user, err := authSvc.RequireSection(ctx, r, "courses")
	if err != nil {
		return nil, nil, err
	}
	svc := &Service{Pool: pool}
	course, err := svc.GetCourse(ctx, courseID, false)
	if err != nil {
		return nil, nil, err
	}
	if course == nil {
		return nil, nil, Err(http.StatusNotFound, "Курс не найден")
	}
	if catPtr := course["category"]; catPtr != nil {
		c := fmt.Sprint(catPtr)
		if !auth.CanEditCourseCategory(ctx, pool, user, &c) {
			return nil, nil, Err(http.StatusForbidden, "Нет доступа к категории этого курса")
		}
	}
	return user, course, nil
}

func VersionEditable(version map[string]any) bool {
	st := fmt.Sprint(version["status"])
	return st == "draft" || st == "published"
}

func AssertVersionEditable(version map[string]any) error {
	if !VersionEditable(version) {
		return Err(http.StatusConflict, "Версия недоступна для редактирования (архивирована)")
	}
	return nil
}

func (s *Service) TestFormSummary(ctx context.Context, formID int64) (map[string]any, error) {
	row, err := scanOne(ctx, s.Pool, `
		SELECT f.id, f.title, f.status, f.list_no, f.use_passing_score, f.passing_score,
		       (SELECT COUNT(*) FROM public.test_questions q WHERE q.form_id = f.id) AS question_count
		FROM public.test_forms f WHERE f.id = $1`, formID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"id": tests.ToInt64Must(row["id"]), "title": fmt.Sprint(row["title"]),
		"status": fmt.Sprint(row["status"]), "listNo": intPtr(row["list_no"]),
		"usePassingScore": Bool(row["use_passing_score"]), "passingScore": toInt(row["passing_score"]),
		"questionCount": toInt(row["question_count"]),
	}, nil
}

func MapTestLink(row map[string]any, formSummary map[string]any) map[string]any {
	qc := 0
	if formSummary != nil {
		qc = toInt(formSummary["questionCount"])
	}
	return map[string]any{
		"id": tests.ToInt64Must(row["id"]), "courseVersionId": tests.ToInt64Must(row["course_version_id"]),
		"topicId": intPtr(row["topic_id"]), "testFormId": tests.ToInt64Must(row["test_form_id"]),
		"type": fmt.Sprint(row["type"]), "isRequired": Bool(row["is_required"]),
		"sortOrder": toInt(row["sort_order"]), "createdAt": row["created_at"], "updatedAt": row["updated_at"],
		"form": formSummary, "questionCount": qc,
	}
}

func (s *Service) AssembleVersion(ctx context.Context, versionID int64, withContent bool) (map[string]any, error) {
	version, err := s.GetVersion(ctx, versionID)
	if err != nil || version == nil {
		return version, err
	}

	topicRows, err := s.Pool.Query(ctx, `
		SELECT * FROM public.course_topics
		WHERE course_version_id = $1 AND deleted_at IS NULL
		ORDER BY sort_order, id`, versionID)
	if err != nil {
		return nil, err
	}
	defer topicRows.Close()

	linkRows, err := s.Pool.Query(ctx, `
		SELECT l.*, f.title AS form_title, f.status AS form_status, f.list_no,
		       f.use_passing_score, f.passing_score,
		       (SELECT COUNT(*) FROM public.test_questions q WHERE q.form_id = l.test_form_id) AS question_count
		FROM public.course_test_links l
		JOIN public.test_forms f ON f.id = l.test_form_id
		WHERE l.course_version_id = $1`, versionID)
	if err != nil {
		return nil, err
	}
	topicLinks := map[int64]map[string]any{}
	var finalLink map[string]any
	for linkRows.Next() {
		l, err := scanRow(linkRows)
		if err != nil {
			linkRows.Close()
			return nil, err
		}
		summary := map[string]any{
			"id": tests.ToInt64Must(l["test_form_id"]), "title": fmt.Sprint(l["form_title"]),
			"status": fmt.Sprint(l["form_status"]), "listNo": intPtr(l["list_no"]),
			"usePassingScore": Bool(l["use_passing_score"]), "passingScore": toInt(l["passing_score"]),
			"questionCount": toInt(l["question_count"]),
		}
		mapped := MapTestLink(l, summary)
		if fmt.Sprint(l["type"]) == "final" {
			finalLink = mapped
		} else if l["topic_id"] != nil {
			topicLinks[tests.ToInt64Must(l["topic_id"])] = mapped
		}
	}
	linkRows.Close()

	var topics []map[string]any
	for topicRows.Next() {
		t, err := scanRow(topicRows)
		if err != nil {
			return nil, err
		}
		tid := tests.ToInt64Must(t["id"])
		matRows, err := s.Pool.Query(ctx, `
			SELECT * FROM public.course_materials
			WHERE topic_id = $1 AND deleted_at IS NULL ORDER BY sort_order, id`, tid)
		if err != nil {
			return nil, err
		}
		var materials []map[string]any
		for matRows.Next() {
			m, err := scanRow(matRows)
			if err != nil {
				matRows.Close()
				return nil, err
			}
			mat := mapMaterial(m, withContent)
			materials = append(materials, mat)
		}
		matRows.Close()
		topics = append(topics, map[string]any{
			"id": tid, "courseVersionId": versionID,
			"title": fmt.Sprint(t["title"]), "description": strOr(t["description"], ""),
			"sortOrder": toInt(t["sort_order"]), "isRequired": Bool(t["is_required"]),
			"minimumActiveSeconds": toInt(t["minimum_active_seconds"]),
			"completionRule": strOr(t["completion_rule"], "all_required_materials"),
			"createdAt": t["created_at"], "updatedAt": t["updated_at"],
			"materials": materials, "topicTest": topicLinks[tid],
		})
	}
	version["topics"] = topics
	version["finalTest"] = finalLink
	return version, nil
}

func mapMaterial(m map[string]any, withContent bool) map[string]any {
	var fileURL any
	if fu := m["file_url"]; fu != nil && fmt.Sprint(fu) != "" {
		fileURL = "/api/course_file.php?path=" + strings.ReplaceAll(urlQueryEscape(fmt.Sprint(fu)), "+", "%20")
	}
	mat := map[string]any{
		"id": tests.ToInt64Must(m["id"]), "topicId": tests.ToInt64Must(m["topic_id"]),
		"type": fmt.Sprint(m["type"]), "title": fmt.Sprint(m["title"]),
		"description": strOr(m["description"], ""), "fileUrl": fileURL,
		"externalUrl": m["external_url"], "mimeType": m["mime_type"],
		"fileSize": intPtr(m["file_size"]), "originalFilename": m["original_filename"],
		"sortOrder": toInt(m["sort_order"]), "isRequired": Bool(m["is_required"]),
		"minimumActiveSeconds": toInt(m["minimum_active_seconds"]),
		"createdAt": m["created_at"], "updatedAt": m["updated_at"],
	}
	if withContent {
		mat["contentHtml"] = strOr(m["content_html"], "")
	}
	return mat
}

func urlQueryEscape(s string) string {
	var b strings.Builder
	for _, r := range s {
		if (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_' || r == '.' || r == '/' {
			b.WriteRune(r)
		} else {
			fmt.Fprintf(&b, "%%%02X", r)
		}
	}
	return b.String()
}

func (s *Service) CreateCourse(ctx context.Context, user *auth.User, data map[string]any, r *http.Request) (map[string]any, error) {
	title := strings.TrimSpace(fmt.Sprint(data["title"]))
	if title == "" {
		return nil, Err(http.StatusBadRequest, "Не указано название курса")
	}
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var courseID int64
	err = tx.QueryRow(ctx, `
		INSERT INTO public.course_courses (owner_id, title, category)
		VALUES ($1,$2,$3) RETURNING id`, user.ID, title, nilStr(data["category"])).Scan(&courseID)
	if err != nil {
		return nil, err
	}

	sd, fd := "", ""
	if sp := strPtr(data["shortDescription"], data["short_description"]); sp != nil {
		sd = SanitizeHTML(sp)
	}
	if fp := strPtr(data["fullDescription"], data["full_description"]); fp != nil {
		fd = SanitizeHTML(fp)
	}
	var versionID int64
	err = tx.QueryRow(ctx, `
		INSERT INTO public.course_versions (
			course_id, version_number, status, short_description, full_description,
			cover_url, sequential_progress, completion_rule, default_deadline_days,
			final_passing_score, require_final_test, generate_certificate, created_by
		) VALUES ($1,1,'draft',$2,$3,$4,$5,$6,$7,$8,$9,$10,$11) RETURNING id`,
		courseID, sd, fd, dataField(data, "coverUrl", "cover_url"),
		Bool(dataField(data, "sequentialProgress", "sequential_progress")),
		strOr(dataField(data, "completionRule", "completion_rule"), "all_required"),
		intField(data, "defaultDeadlineDays", "default_deadline_days"),
		dataField(data, "finalPassingScore", "final_passing_score"),
		Bool(dataField(data, "requireFinalTest", "require_final_test")),
		Bool(dataField(data, "generateCertificate", "generate_certificate")), user.ID,
	).Scan(&versionID)
	if err != nil {
		return nil, err
	}
	_, _ = tx.Exec(ctx, `UPDATE public.course_courses SET current_version_id = $1, updated_at = now() WHERE id = $2`, versionID, courseID)
	uid := user.ID
	Audit(ctx, s.Pool, &uid, "course.create", "course", &courseID, map[string]any{"versionId": versionID, "title": title}, r)
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	version, err := s.AssembleVersion(ctx, versionID, true)
	if err != nil {
		return nil, err
	}
	course, err := s.GetCourse(ctx, courseID, false)
	if err != nil {
		return nil, err
	}
	return map[string]any{"course": course, "version": version}, nil
}

func (s *Service) PublishVersion(ctx context.Context, versionID int64, user *auth.User, r *http.Request) (map[string]any, error) {
	version, err := s.GetVersion(ctx, versionID)
	if err != nil || version == nil {
		return nil, Err(http.StatusNotFound, "Версия не найдена")
	}
	st := fmt.Sprint(version["status"])
	if st == "published" {
		return nil, Err(http.StatusConflict, "Версия уже опубликована")
	}
	if st == "archived" {
		return nil, Err(http.StatusConflict, "Архивная версия не может быть опубликована")
	}
	assembledCheck, err := s.AssembleVersion(ctx, versionID, false)
	if err != nil {
		return nil, err
	}
	if ready, problems, _ := VersionReadiness(assembledCheck); !ready {
		return nil, Err(http.StatusBadRequest, "Курс не готов к публикации: "+strings.Join(problems, "; "))
	}
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	rows, _ := tx.Query(ctx, `SELECT DISTINCT test_form_id FROM public.course_test_links WHERE course_version_id = $1`, versionID)
	var formIDs []int64
	for rows != nil && rows.Next() {
		var fid int64
		if rows.Scan(&fid) == nil {
			formIDs = append(formIDs, fid)
		}
	}
	if rows != nil {
		rows.Close()
	}
	for _, fid := range formIDs {
		_, _ = tx.Exec(ctx, `
			UPDATE public.test_forms SET status = 'published',
				published_at = COALESCE(published_at, now()),
				list_no = COALESCE(list_no, nextval('public.test_forms_list_no_seq')),
				updated_at = now() WHERE id = $1`, fid)
	}
	_, _ = tx.Exec(ctx, `
		UPDATE public.course_versions SET status = 'published', published_at = COALESCE(published_at, now()), updated_at = now()
		WHERE id = $1`, versionID)
	courseID := tests.ToInt64Must(version["courseId"])
	_, _ = tx.Exec(ctx, `UPDATE public.course_courses SET current_version_id = $1, updated_at = now() WHERE id = $2`, versionID, courseID)
	uid := user.ID
	Audit(ctx, s.Pool, &uid, "course.version.publish", "course_version", &versionID, map[string]any{"courseId": courseID, "formIds": formIDs}, r)
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return s.AssembleVersion(ctx, versionID, true)
}

// VersionReadiness — блокеры публикации (как cs_readiness в PHP, без разбора каждого ответа).
func VersionReadiness(version map[string]any) (ready bool, errors []string, warnings []string) {
	if version == nil {
		return false, []string{"Версия не найдена"}, nil
	}
	topics, _ := version["topics"].([]map[string]any)
	if len(topics) == 0 {
		errors = append(errors, "Нет тем в версии курса")
	}
	for _, topic := range topics {
		title := strings.TrimSpace(fmt.Sprint(topic["title"]))
		if title == "" || title == "<nil>" {
			title = "Тема"
		}
		mats, _ := topic["materials"].([]map[string]any)
		tt, _ := topic["topicTest"].(map[string]any)
		if !Bool(topic["isRequired"]) {
			if len(mats) == 0 && tt == nil {
				warnings = append(warnings, fmt.Sprintf("Тема «%s» необязательна и пуста", title))
			}
			continue
		}
		if len(mats) == 0 {
			errors = append(errors, fmt.Sprintf("Обязательная тема «%s» без материалов", title))
		}
		if tt != nil && Bool(tt["isRequired"]) && toInt(tt["questionCount"]) < 1 {
			errors = append(errors, fmt.Sprintf("Тест темы «%s» без вопросов", title))
		}
	}
	if Bool(version["requireFinalTest"]) {
		ft, _ := version["finalTest"].(map[string]any)
		if ft == nil {
			errors = append(errors, "Требуется итоговый тест, но он не создан")
		} else if toInt(ft["questionCount"]) < 1 {
			errors = append(errors, "Итоговый тест без вопросов")
		}
	} else if ft, ok := version["finalTest"].(map[string]any); ok && ft != nil {
		warnings = append(warnings, "Итоговый тест есть, но требование итогового теста выключено")
	}
	return len(errors) == 0, errors, warnings
}

func (s *Service) ResolveVersionID(ctx context.Context, body map[string]any) (int64, error) {
	if v, err := tests.ToInt64Public(body["versionId"]); err == nil && v > 0 {
		return v, nil
	}
	courseID, err := tests.ToInt64Public(body["courseId"])
	if err != nil || courseID <= 0 {
		return 0, Err(http.StatusBadRequest, "Укажите courseId или versionId")
	}
	course, err := s.GetCourse(ctx, courseID, false)
	if err != nil {
		return 0, err
	}
	if course == nil || course["currentVersionId"] == nil {
		return 0, Err(http.StatusNotFound, "Курс или версия не найдены")
	}
	return tests.ToInt64Must(course["currentVersionId"]), nil
}

func MapEnrollment(r map[string]any, progress, course map[string]any) map[string]any {
	out := map[string]any{
		"id": tests.ToInt64Must(r["id"]), "assignmentId": intPtr(r["assignment_id"]),
		"courseVersionId": tests.ToInt64Must(r["course_version_id"]), "userId": tests.ToInt64Must(r["user_id"]),
		"status": fmt.Sprint(r["status"]), "assignedAt": r["assigned_at"], "startsAt": r["starts_at"],
		"deadlineAt": r["deadline_at"], "startedAt": r["started_at"], "lastActivityAt": r["last_activity_at"],
		"completedAt": r["completed_at"], "finalScore": floatPtr(r["final_score"]),
		"createdAt": r["created_at"], "updatedAt": r["updated_at"],
	}
	if progress != nil {
		out["progress"] = progress
	}
	if course != nil {
		out["course"] = course
	}
	return out
}

func (s *Service) RequireEnrollmentAccess(ctx context.Context, enrollmentID int64, user *auth.User, adminOK bool) (map[string]any, error) {
	enr, err := scanOne(ctx, s.Pool, `SELECT * FROM public.course_enrollments WHERE id = $1`, enrollmentID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, Err(http.StatusNotFound, "Запись на курс не найдена")
	}
	if err != nil {
		return nil, err
	}
	if tests.ToInt64Must(enr["user_id"]) == user.ID {
		return enr, nil
	}
	if adminOK && auth.IsAdmin(user) {
		return enr, nil
	}
	return nil, Err(http.StatusForbidden, "Нет доступа к этой записи")
}

func (s *Service) MarkOverdue(ctx context.Context) {
	_, _ = s.Pool.Exec(ctx, `
		UPDATE public.course_enrollments SET status = 'overdue', updated_at = now()
		WHERE deadline_at IS NOT NULL AND deadline_at < now()
		  AND status IN ('not_started', 'in_progress')`)
}

func (s *Service) EnsureTopicProgressRows(ctx context.Context, enrollmentID, versionID int64) error {
	enr, err := scanOne(ctx, s.Pool, `SELECT * FROM public.course_enrollments WHERE id = $1`, enrollmentID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	version, err := s.GetVersion(ctx, versionID)
	if err != nil || version == nil {
		return err
	}
	rows, err := s.Pool.Query(ctx, `
		SELECT id FROM public.course_topics WHERE course_version_id = $1 AND deleted_at IS NULL ORDER BY sort_order, id`, versionID)
	if err != nil {
		return err
	}
	var ids []int64
	for rows.Next() {
		var id int64
		if rows.Scan(&id) == nil {
			ids = append(ids, id)
		}
	}
	rows.Close()
	if len(ids) == 0 {
		return nil
	}
	sequential := Bool(version["sequentialProgress"])
	for i, tid := range ids {
		status := "available"
		if sequential && i > 0 {
			status = "locked"
		}
		_, _ = s.Pool.Exec(ctx, `
			INSERT INTO public.course_topic_progress (enrollment_id, topic_id, status)
			VALUES ($1,$2,$3) ON CONFLICT (enrollment_id, topic_id) DO NOTHING`,
			enrollmentID, tid, status)
	}
	_ = enr
	return nil
}

func (s *Service) RecalculateLocks(ctx context.Context, enrollmentID int64) error {
	enr, err := scanOne(ctx, s.Pool, `SELECT * FROM public.course_enrollments WHERE id = $1`, enrollmentID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	versionID := tests.ToInt64Must(enr["course_version_id"])
	version, err := s.GetVersion(ctx, versionID)
	if err != nil || version == nil {
		return err
	}
	_ = s.EnsureTopicProgressRows(ctx, enrollmentID, versionID)
	rows, err := s.Pool.Query(ctx, `
		SELECT t.id, COALESCE(p.status, 'locked') AS status
		FROM public.course_topics t
		LEFT JOIN public.course_topic_progress p ON p.topic_id = t.id AND p.enrollment_id = $1
		WHERE t.course_version_id = $2 AND t.deleted_at IS NULL
		ORDER BY t.sort_order, t.id`, enrollmentID, versionID)
	if err != nil {
		return err
	}
	type item struct{ id int64; status string }
	var items []item
	for rows.Next() {
		var it item
		if rows.Scan(&it.id, &it.status) == nil {
			items = append(items, it)
		}
	}
	rows.Close()
	sequential := Bool(version["sequentialProgress"])
	prevCompleted := true
	for _, it := range items {
		if it.status == "completed" {
			prevCompleted = true
			continue
		}
		if !sequential || prevCompleted {
			if it.status == "locked" {
				_, _ = s.Pool.Exec(ctx, `UPDATE public.course_topic_progress SET status = 'available', updated_at = now() WHERE enrollment_id = $1 AND topic_id = $2`, enrollmentID, it.id)
			}
			prevCompleted = false
		} else if it.status != "locked" {
			_, _ = s.Pool.Exec(ctx, `UPDATE public.course_topic_progress SET status = 'locked', updated_at = now() WHERE enrollment_id = $1 AND topic_id = $2`, enrollmentID, it.id)
			prevCompleted = false
		} else {
			prevCompleted = false
		}
	}
	return nil
}

func (s *Service) TestLinkPassed(ctx context.Context, enrollmentID, linkID int64) bool {
	var passed *bool
	var usePassing bool
	err := s.Pool.QueryRow(ctx, `
		SELECT a.passed, f.use_passing_score
		FROM public.course_test_attempt_links tal
		JOIN public.test_attempts a ON a.id = tal.test_attempt_id
		JOIN public.course_test_links l ON l.id = tal.course_test_link_id
		JOIN public.test_forms f ON f.id = l.test_form_id
		WHERE tal.enrollment_id = $1 AND tal.course_test_link_id = $2 AND a.status = 'completed'
		ORDER BY a.finished_at DESC NULLS LAST, a.id DESC LIMIT 1`, enrollmentID, linkID).Scan(&passed, &usePassing)
	if errors.Is(err, pgx.ErrNoRows) {
		return false
	}
	if usePassing {
		return passed != nil && *passed
	}
	return true
}

func (s *Service) CheckTopicComplete(ctx context.Context, enrollmentID, topicID int64) bool {
	topic, err := scanOne(ctx, s.Pool, `
		SELECT t.*, COALESCE(p.active_seconds, 0) AS progress_active
		FROM public.course_topics t
		LEFT JOIN public.course_topic_progress p ON p.topic_id = t.id AND p.enrollment_id = $1
		WHERE t.id = $2 AND t.deleted_at IS NULL`, enrollmentID, topicID)
	if err != nil || topic == nil {
		return false
	}
	matRows, _ := s.Pool.Query(ctx, `
		SELECT m.is_required, m.minimum_active_seconds,
		       COALESCE(mp.status, 'not_started') AS mp_status,
		       COALESCE(mp.active_seconds, 0) AS mp_active
		FROM public.course_materials m
		LEFT JOIN public.course_material_progress mp ON mp.material_id = m.id AND mp.enrollment_id = $1
		WHERE m.topic_id = $2 AND m.deleted_at IS NULL`, enrollmentID, topicID)
	for matRows != nil && matRows.Next() {
		var required bool
		var minSec, mpActive int
		var mpStatus string
		if matRows.Scan(&required, &minSec, &mpStatus, &mpActive) == nil && required {
			if mpStatus != "completed" || (minSec > 0 && mpActive < minSec) {
				matRows.Close()
				return false
			}
		}
	}
	if matRows != nil {
		matRows.Close()
	}
	minTopic := toInt(topic["minimum_active_seconds"])
	if minTopic > 0 && toInt(topic["progress_active"]) < minTopic {
		return false
	}
	link, _ := scanOne(ctx, s.Pool, `SELECT id, is_required FROM public.course_test_links WHERE topic_id = $1 AND type = 'topic' LIMIT 1`, topicID)
	if link != nil && Bool(link["is_required"]) {
		if !s.TestLinkPassed(ctx, enrollmentID, tests.ToInt64Must(link["id"])) {
			return false
		}
	}
	return true
}

func (s *Service) NextAction(ctx context.Context, enrollmentID int64) map[string]any {
	enr, err := scanOne(ctx, s.Pool, `SELECT * FROM public.course_enrollments WHERE id = $1`, enrollmentID)
	if err != nil || enr == nil {
		return map[string]any{"type": "unknown", "label": "Запись не найдена"}
	}
	st := fmt.Sprint(enr["status"])
	switch st {
	case "completed", "cancelled":
		return map[string]any{"type": "done", "label": "Курс завершён"}
	case "failed":
		return map[string]any{"type": "failed", "label": "Курс не сдан"}
	}
	// "overdue" намеренно не обрывает расчёт: просроченный курс по-прежнему проходим
	// (TryCompleteEnrollment статус не проверяет, AttemptStart пускает overdue),
	// поэтому сотруднику нужен реальный следующий шаг, а не тупик.
	versionID := tests.ToInt64Must(enr["course_version_id"])
	_ = s.RecalculateLocks(ctx, enrollmentID)
	assembled, _ := s.AssembleVersion(ctx, versionID, false)
	if assembled == nil {
		return map[string]any{"type": "unknown", "label": "Версия не найдена"}
	}
	topicProg := map[int64]map[string]any{}
	tpRows, _ := s.Pool.Query(ctx, `SELECT topic_id, status, last_material_id FROM public.course_topic_progress WHERE enrollment_id = $1`, enrollmentID)
	for tpRows != nil && tpRows.Next() {
		var tid int64
		var status string
		var lastMat *int64
		if tpRows.Scan(&tid, &status, &lastMat) == nil {
			topicProg[tid] = map[string]any{"status": status, "lastMaterialId": lastMat}
		}
	}
	if tpRows != nil {
		tpRows.Close()
	}
	matProg := map[int64]string{}
	mpRows, _ := s.Pool.Query(ctx, `SELECT material_id, status FROM public.course_material_progress WHERE enrollment_id = $1`, enrollmentID)
	for mpRows != nil && mpRows.Next() {
		var mid int64
		var status string
		if mpRows.Scan(&mid, &status) == nil {
			matProg[mid] = status
		}
	}
	if mpRows != nil {
		mpRows.Close()
	}
	topics, _ := assembled["topics"].([]map[string]any)
	for _, topic := range topics {
		tid := tests.ToInt64Must(topic["id"])
		ps := "locked"
		if p, ok := topicProg[tid]; ok {
			ps = fmt.Sprint(p["status"])
		}
		if ps == "completed" {
			continue
		}
		if ps == "locked" {
			return map[string]any{"type": "locked", "topicId": tid, "label": "Тема «" + fmt.Sprint(topic["title"]) + "» ещё недоступна"}
		}
		mats, _ := topic["materials"].([]map[string]any)
		for _, m := range mats {
			if !Bool(m["isRequired"]) {
				continue
			}
			mid := tests.ToInt64Must(m["id"])
			if matProg[mid] != "completed" {
				return map[string]any{"type": "material", "topicId": tid, "materialId": mid, "label": "Изучить: " + fmt.Sprint(m["title"])}
			}
		}
		if tt, ok := topic["topicTest"].(map[string]any); ok && tt != nil && Bool(tt["isRequired"]) {
			lid := tests.ToInt64Must(tt["id"])
			if !s.TestLinkPassed(ctx, enrollmentID, lid) {
				return map[string]any{"type": "topic_test", "topicId": tid, "courseTestLinkId": lid, "label": "Пройти тест темы «" + fmt.Sprint(topic["title"]) + "»"}
			}
		}
		if !s.CheckTopicComplete(ctx, enrollmentID, tid) {
			return map[string]any{"type": "topic", "topicId": tid, "label": "Завершите тему «" + fmt.Sprint(topic["title"]) + "»"}
		}
		// Условия темы выполнены, но статус не обновлён: так бывает, когда минимум
		// времени темы набран уже после отметки последнего материала. Эндпоинта для
		// ручного завершения темы нет, поэтому завершаем здесь и открываем следующую.
		// Рекурсия ограничена числом тем: каждый проход переводит одну тему в completed.
		if s.markTopicCompleted(ctx, enrollmentID, tid) {
			return s.NextAction(ctx, enrollmentID)
		}
		return map[string]any{"type": "complete_topic", "topicId": tid, "label": "Отметить тему «" + fmt.Sprint(topic["title"]) + "» завершённой"}
	}
	if Bool(assembled["requireFinalTest"]) {
		if ft, ok := assembled["finalTest"].(map[string]any); ok && ft != nil {
			lid := tests.ToInt64Must(ft["id"])
			if !s.TestLinkPassed(ctx, enrollmentID, lid) {
				return map[string]any{"type": "final_test", "courseTestLinkId": lid, "label": "Пройти итоговый тест"}
			}
		}
	}
	return map[string]any{"type": "complete_course", "label": "Завершить курс"}
}

// markTopicCompleted переводит тему в completed и пересчитывает блокировки.
// Возвращает true, только если статус действительно изменился — это и есть
// гарантия, что рекурсия в NextAction не зациклится.
func (s *Service) markTopicCompleted(ctx context.Context, enrollmentID, topicID int64) bool {
	tag, err := s.Pool.Exec(ctx, `
		UPDATE public.course_topic_progress
		SET status = 'completed', completed_at = COALESCE(completed_at, now()), updated_at = now()
		WHERE enrollment_id = $1 AND topic_id = $2 AND status <> 'completed'`, enrollmentID, topicID)
	if err != nil || tag.RowsAffected() == 0 {
		return false
	}
	_ = s.RecalculateLocks(ctx, enrollmentID)
	return true
}

// CompleteTopicIfReady — то же, но с проверкой условий. Нужна там, где время
// темы растёт (heartbeat), а статус иначе никто бы не пересмотрел.
func (s *Service) CompleteTopicIfReady(ctx context.Context, enrollmentID, topicID int64) bool {
	if !s.CheckTopicComplete(ctx, enrollmentID, topicID) {
		return false
	}
	return s.markTopicCompleted(ctx, enrollmentID, topicID)
}

func (s *Service) EnrollmentProgress(ctx context.Context, enrollmentID int64) map[string]any {
	enr, err := scanOne(ctx, s.Pool, `SELECT * FROM public.course_enrollments WHERE id = $1`, enrollmentID)
	if err != nil || enr == nil {
		return map[string]any{
			"percent": 0, "topicsCompleted": 0, "topicsTotal": 0,
			"nextAction": map[string]any{"type": "unknown", "label": "Запись не найдена"},
		}
	}
	// NextAction может завершить «застрявшую» тему — считаем его до подсчёта,
	// чтобы проценты и счётчик тем в этом же ответе уже учли изменение.
	nextAction := s.NextAction(ctx, enrollmentID)
	versionID := tests.ToInt64Must(enr["course_version_id"])
	_ = s.EnsureTopicProgressRows(ctx, enrollmentID, versionID)
	var totalReq, doneReq int
	_ = s.Pool.QueryRow(ctx, `
		SELECT
		 COUNT(*) FILTER (WHERE t.is_required IS TRUE),
		 COUNT(*) FILTER (WHERE t.is_required IS TRUE AND p.status = 'completed')
		FROM public.course_topics t
		LEFT JOIN public.course_topic_progress p ON p.topic_id = t.id AND p.enrollment_id = $1
		WHERE t.course_version_id = $2 AND t.deleted_at IS NULL`, enrollmentID, versionID).Scan(&totalReq, &doneReq)
	version, _ := s.GetVersion(ctx, versionID)
	stepsTotal, stepsDone := totalReq, doneReq
	if version != nil && Bool(version["requireFinalTest"]) {
		stepsTotal++
		var flID int64
		if s.Pool.QueryRow(ctx, `SELECT id FROM public.course_test_links WHERE course_version_id = $1 AND type = 'final' LIMIT 1`, versionID).Scan(&flID) == nil {
			if s.TestLinkPassed(ctx, enrollmentID, flID) {
				stepsDone++
			}
		}
	}
	percent := 0
	if stepsTotal > 0 {
		percent = int(math.Round(float64(stepsDone) / float64(stepsTotal) * 100))
	}
	if fmt.Sprint(enr["status"]) == "completed" {
		percent = 100
	}
	return map[string]any{
		"percent": percent, "topicsCompleted": doneReq, "topicsTotal": totalReq,
		"nextAction": nextAction,
	}
}

func (s *Service) TryCompleteEnrollment(ctx context.Context, enrollmentID int64, r *http.Request) (map[string]any, error) {
	enr, err := scanOne(ctx, s.Pool, `SELECT * FROM public.course_enrollments WHERE id = $1`, enrollmentID)
	if err != nil || enr == nil {
		return nil, nil
	}
	st := fmt.Sprint(enr["status"])
	if st == "completed" || st == "cancelled" {
		row, _ := scanOne(ctx, s.Pool, `SELECT * FROM public.course_completions WHERE enrollment_id = $1 ORDER BY id DESC LIMIT 1`, enrollmentID)
		return row, nil
	}
	versionID := tests.ToInt64Must(enr["course_version_id"])
	version, err := s.GetVersion(ctx, versionID)
	if err != nil || version == nil {
		return nil, err
	}
	tRows, _ := s.Pool.Query(ctx, `
		SELECT id FROM public.course_topics
		WHERE course_version_id = $1 AND deleted_at IS NULL AND is_required IS TRUE ORDER BY sort_order, id`, versionID)
	for tRows != nil && tRows.Next() {
		var tid int64
		if tRows.Scan(&tid) == nil {
			if !s.CheckTopicComplete(ctx, enrollmentID, tid) {
				tRows.Close()
				return nil, nil
			}
			_, _ = s.Pool.Exec(ctx, `
				UPDATE public.course_topic_progress SET status = 'completed',
				 completed_at = COALESCE(completed_at, now()), updated_at = now()
				WHERE enrollment_id = $1 AND topic_id = $2 AND status <> 'completed'`, enrollmentID, tid)
		}
	}
	if tRows != nil {
		tRows.Close()
	}
	var finalScore *float64
	finalPassed := true
	if Bool(version["requireFinalTest"]) {
		fl, _ := scanOne(ctx, s.Pool, `SELECT id FROM public.course_test_links WHERE course_version_id = $1 AND type = 'final' LIMIT 1`, versionID)
		if fl == nil || !s.TestLinkPassed(ctx, enrollmentID, tests.ToInt64Must(fl["id"])) {
			return nil, nil
		}
		var score *float64
		var passed *bool
		_ = s.Pool.QueryRow(ctx, `
			SELECT a.score, a.passed FROM public.course_test_attempt_links tal
			JOIN public.test_attempts a ON a.id = tal.test_attempt_id
			WHERE tal.enrollment_id = $1 AND tal.course_test_link_id = $2 AND a.status = 'completed'
			ORDER BY a.finished_at DESC NULLS LAST, a.id DESC LIMIT 1`,
			enrollmentID, tests.ToInt64Must(fl["id"])).Scan(&score, &passed)
		finalScore = score
		if passed != nil {
			finalPassed = *passed
		}
	}
	snapshot := s.buildCompletionSnapshot(ctx, enrollmentID, enr, version, finalScore)
	totalActive := toInt(snapshot["totalActiveSeconds"])
	userID := tests.ToInt64Must(enr["user_id"])
	courseID := tests.ToInt64Must(version["courseId"])
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	var completionNumber int
	_ = tx.QueryRow(ctx, `
		SELECT COALESCE(MAX(completion_number), 0) + 1 FROM public.course_completions WHERE user_id = $1 AND course_id = $2`,
		userID, courseID).Scan(&completionNumber)
	snapB, _ := json.Marshal(snapshot)
	var completionID int64
	err = tx.QueryRow(ctx, `
		INSERT INTO public.course_completions (
		 enrollment_id, user_id, course_id, course_version_id, completion_number,
		 assigned_at, started_at, completed_at, total_active_seconds, final_score, passed, result_snapshot
		) VALUES ($1,$2,$3,$4,$5,$6,$7,now(),$8,$9,$10,$11::jsonb) RETURNING id`,
		enrollmentID, userID, courseID, versionID, completionNumber,
		enr["assigned_at"], enr["started_at"], totalActive, finalScore, finalPassed, string(snapB)).Scan(&completionID)
	if err != nil {
		return nil, err
	}
	_, _ = tx.Exec(ctx, `
		UPDATE public.course_enrollments SET status = 'completed', completed_at = now(), final_score = $1, updated_at = now()
		WHERE id = $2`, finalScore, enrollmentID)
	uid := userID
	eid := enrollmentID
	Audit(ctx, s.Pool, &uid, "course.enrollment.complete", "course_enrollment", &eid, map[string]any{
		"completionId": completionID, "finalScore": finalScore,
	}, r)
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return scanOne(ctx, s.Pool, `SELECT * FROM public.course_completions WHERE id = $1`, completionID)
}

func (s *Service) buildCompletionSnapshot(ctx context.Context, enrollmentID int64, enr, version map[string]any, finalScore *float64) map[string]any {
	course, _ := s.GetCourse(ctx, tests.ToInt64Must(version["courseId"]), true)
	userID := tests.ToInt64Must(enr["user_id"])
	u, _ := scanOne(ctx, s.Pool, `SELECT id, firstname, surname, lastname, role, ofo FROM public.user_info WHERE id = $1`, userID)
	fio := UserFio(u)
	var ofoID *int64
	var ofoName *string
	if u != nil {
		raw := fmt.Sprint(u["ofo"])
		if raw != "" && raw != "-1" {
			if id, err := tests.ToInt64Public(raw); err == nil && id > 0 {
				ofoID = &id
				var name string
				if s.Pool.QueryRow(ctx, `SELECT name FROM public.ofo_unit WHERE id = $1`, id).Scan(&name) == nil {
					ofoName = &name
				}
			}
		}
	}
	var topicsOut []map[string]any
	totalActive := 0
	tRows, _ := s.Pool.Query(ctx, `
		SELECT t.id, t.title, COALESCE(p.active_seconds, 0) AS active_seconds
		FROM public.course_topics t
		LEFT JOIN public.course_topic_progress p ON p.topic_id = t.id AND p.enrollment_id = $1
		WHERE t.course_version_id = $2 AND t.deleted_at IS NULL ORDER BY t.sort_order, t.id`, enrollmentID, tests.ToInt64Must(version["id"]))
	for tRows != nil && tRows.Next() {
		var tid int64
		var title string
		var active int
		if tRows.Scan(&tid, &title, &active) == nil {
			totalActive += active
			topicsOut = append(topicsOut, map[string]any{"topicId": tid, "title": title, "activeSeconds": active})
		}
	}
	if tRows != nil {
		tRows.Close()
	}
	out := map[string]any{
		"courseTitle": "", "versionNumber": version["versionNumber"], "userFio": fio,
		"role": "", "ofoId": ofoID, "ofoName": ofoName, "topics": topicsOut,
		"finalScore": finalScore, "finalAttempts": 0, "totalActiveSeconds": totalActive,
		"generateCertificate": Bool(version["generateCertificate"]),
		"requireFinalTest":    Bool(version["requireFinalTest"]),
	}
	if course != nil {
		out["courseTitle"] = course["title"]
	}
	if u != nil {
		out["role"] = u["role"]
	}
	return out
}

func UserFio(u map[string]any) string {
	if u == nil {
		return ""
	}
	parts := []string{fmt.Sprint(u["surname"]), fmt.Sprint(u["firstname"]), fmt.Sprint(u["lastname"])}
	var out []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" && p != "<nil>" {
			out = append(out, p)
		}
	}
	return strings.Join(out, " ")
}

func UploadsRoot(base string) string {
	preferred := filepath.Join(base, "courses")
	if st, err := os.Stat(preferred); err == nil && st.IsDir() {
		if writable(preferred) {
			return preferred
		}
	}
	fallback := filepath.Join(base, "courses")
	_ = os.MkdirAll(fallback, 0o755)
	return fallback
}

func CourseUploadDir(uploadRoot string, courseID int64) (absDir, relPrefix string) {
	rel := fmt.Sprintf("courses/%d", courseID)
	abs := filepath.Join(uploadRoot, fmt.Sprint(courseID))
	_ = os.MkdirAll(abs, 0o755)
	return abs, rel
}

func writable(path string) bool {
	f, err := os.OpenFile(filepath.Join(path, ".write_test"), os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return false
	}
	_ = f.Close()
	_ = os.Remove(filepath.Join(path, ".write_test"))
	return true
}

func (s *Service) TopicVersionRow(ctx context.Context, topicID int64) (map[string]any, error) {
	return scanOne(ctx, s.Pool, `
		SELECT t.*, v.status AS version_status, v.id AS version_id, v.course_id
		FROM public.course_topics t
		JOIN public.course_versions v ON v.id = t.course_version_id
		WHERE t.id = $1 AND t.deleted_at IS NULL`, topicID)
}

func (s *Service) MaterialVersionRow(ctx context.Context, materialID int64) (map[string]any, error) {
	return scanOne(ctx, s.Pool, `
		SELECT m.*, t.course_version_id, v.status AS version_status, v.course_id, t.id AS topic_id_chk
		FROM public.course_materials m
		JOIN public.course_topics t ON t.id = m.topic_id
		JOIN public.course_versions v ON v.id = t.course_version_id
		WHERE m.id = $1 AND m.deleted_at IS NULL AND t.deleted_at IS NULL`, materialID)
}

func (s *Service) OfoDescendants(ctx context.Context, ofoIDs []int64) ([]int64, error) {
	ids := uniquePositive(ofoIDs)
	if len(ids) == 0 {
		return nil, nil
	}
	var hasParent bool
	_ = s.Pool.QueryRow(ctx, `
		SELECT EXISTS (
		 SELECT 1 FROM information_schema.columns
		 WHERE table_schema = 'public' AND table_name = 'ofo_unit' AND column_name = 'parent_id')`).Scan(&hasParent)
	if !hasParent {
		return ids, nil
	}
	ph := make([]string, len(ids))
	args := make([]any, len(ids))
	for i, id := range ids {
		ph[i] = fmt.Sprintf("$%d", i+1)
		args[i] = id
	}
	q := fmt.Sprintf(`
		WITH RECURSIVE tree AS (
		 SELECT id FROM public.ofo_unit WHERE id IN (%s)
		 UNION ALL
		 SELECT u.id FROM public.ofo_unit u INNER JOIN tree t ON u.parent_id = t.id
		) SELECT DISTINCT id FROM tree`, strings.Join(ph, ","))
	rows, err := s.Pool.Query(ctx, q, args...)
	if err != nil {
		return ids, nil
	}
	defer rows.Close()
	var out []int64
	for rows.Next() {
		var id int64
		if rows.Scan(&id) == nil {
			out = append(out, id)
		}
	}
	if len(out) == 0 {
		return ids, nil
	}
	return out, nil
}

func (s *Service) ResolveOfoUsers(ctx context.Context, ofoIDs []int64, includeChildren bool) ([]int64, error) {
	ids := uniquePositive(ofoIDs)
	if len(ids) == 0 {
		return nil, nil
	}
	if includeChildren {
		var err error
		ids, err = s.OfoDescendants(ctx, ids)
		if err != nil {
			return nil, err
		}
	}
	ph := make([]string, len(ids))
	args := make([]any, len(ids))
	for i, id := range ids {
		ph[i] = fmt.Sprintf("$%d", i+1)
		args[i] = fmt.Sprint(id)
	}
	q := fmt.Sprintf(`
		SELECT id FROM public.user_info
		WHERE status IS TRUE AND ofo IS NOT NULL AND ofo <> '' AND ofo <> '-1' AND ofo IN (%s)`, strings.Join(ph, ","))
	rows, err := s.Pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []int64
	for rows.Next() {
		var id int64
		if rows.Scan(&id) == nil {
			out = append(out, id)
		}
	}
	return out, nil
}

func (s *Service) CreateTopicTest(ctx context.Context, topicID int64, user *auth.User, testData map[string]any, r *http.Request) (map[string]any, error) {
	topic, err := s.TopicVersionRow(ctx, topicID)
	if errors.Is(err, pgx.ErrNoRows) || topic == nil {
		return nil, Err(http.StatusNotFound, "Тема не найдена")
	}
	if err != nil {
		return nil, err
	}
	if fmt.Sprint(topic["version_status"]) != "draft" && fmt.Sprint(topic["version_status"]) != "published" {
		return nil, Err(http.StatusConflict, "Версия недоступна для редактирования (архивирована)")
	}
	var exist int64
	if s.Pool.QueryRow(ctx, `SELECT id FROM public.course_test_links WHERE topic_id = $1 AND type = 'topic' LIMIT 1`, topicID).Scan(&exist) == nil {
		return nil, Err(http.StatusConflict, "У темы уже есть промежуточный тест")
	}
	payload := map[string]any{"kind": "test", "visibility": "private", "title": "Тест: " + fmt.Sprint(topic["title"]), "questions": []any{}}
	for k, v := range testData {
		payload[k] = v
	}
	payload["kind"] = "test"
	payload["visibility"] = "private"
	delete(payload, "id")
	formID, err := tests.PersistForm(ctx, s.Pool, payload, user.ID)
	if err != nil {
		if pe, ok := err.(*tests.PersistError); ok {
			return nil, Err(http.StatusForbidden, pe.Message)
		}
		return nil, err
	}
	isRequired := true
	if v, ok := testData["isRequired"]; ok {
		isRequired = Bool(v)
	} else if v, ok := testData["is_required"]; ok {
		isRequired = Bool(v)
	}
	sortOrder := toInt(testData["sortOrder"])
	if sortOrder == 0 {
		sortOrder = toInt(testData["sort_order"])
	}
	link, err := scanOne(ctx, s.Pool, `
		INSERT INTO public.course_test_links (course_version_id, topic_id, test_form_id, type, is_required, sort_order)
		VALUES ($1,$2,$3,'topic',$4,$5) RETURNING *`,
		tests.ToInt64Must(topic["version_id"]), topicID, formID, isRequired, sortOrder)
	if err != nil {
		return nil, err
	}
	uid := user.ID
	lid := tests.ToInt64Must(link["id"])
	Audit(ctx, s.Pool, &uid, "course.topic_test.create", "course_test_link", &lid, map[string]any{"topicId": topicID, "testFormId": formID}, r)
	summary, _ := s.TestFormSummary(ctx, formID)
	return MapTestLink(link, summary), nil
}

func (s *Service) CreateFinalTest(ctx context.Context, versionID int64, user *auth.User, testData map[string]any, r *http.Request) (map[string]any, error) {
	version, err := s.GetVersion(ctx, versionID)
	if err != nil || version == nil {
		return nil, Err(http.StatusNotFound, "Версия не найдена")
	}
	if err := AssertVersionEditable(version); err != nil {
		return nil, err
	}
	var exist int64
	if s.Pool.QueryRow(ctx, `SELECT id FROM public.course_test_links WHERE course_version_id = $1 AND type = 'final' LIMIT 1`, versionID).Scan(&exist) == nil {
		return nil, Err(http.StatusConflict, "Итоговый тест уже создан")
	}
	payload := map[string]any{"kind": "test", "visibility": "private", "title": "Итоговый тест", "questions": []any{}}
	for k, v := range testData {
		payload[k] = v
	}
	payload["kind"] = "test"
	payload["visibility"] = "private"
	delete(payload, "id")
	formID, err := tests.PersistForm(ctx, s.Pool, payload, user.ID)
	if err != nil {
		if pe, ok := err.(*tests.PersistError); ok {
			return nil, Err(http.StatusForbidden, pe.Message)
		}
		return nil, err
	}
	isRequired := true
	if v, ok := testData["isRequired"]; ok {
		isRequired = Bool(v)
	}
	sortOrder := toInt(testData["sortOrder"])
	link, err := scanOne(ctx, s.Pool, `
		INSERT INTO public.course_test_links (course_version_id, topic_id, test_form_id, type, is_required, sort_order)
		VALUES ($1,NULL,$2,'final',$3,$4) RETURNING *`, versionID, formID, isRequired, sortOrder)
	if err != nil {
		return nil, err
	}
	if !Bool(version["requireFinalTest"]) {
		_, _ = s.Pool.Exec(ctx, `UPDATE public.course_versions SET require_final_test = true, updated_at = now() WHERE id = $1`, versionID)
	}
	uid := user.ID
	lid := tests.ToInt64Must(link["id"])
	Audit(ctx, s.Pool, &uid, "course.final_test.create", "course_test_link", &lid, map[string]any{"versionId": versionID, "testFormId": formID}, r)
	summary, _ := s.TestFormSummary(ctx, formID)
	return MapTestLink(link, summary), nil
}

// scan helpers
func scanOne(ctx context.Context, pool *pgxpool.Pool, query string, args ...any) (map[string]any, error) {
	rows, err := pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	if !rows.Next() {
		return nil, pgx.ErrNoRows
	}
	return scanRow(rows)
}

func scanRow(rows pgx.Rows) (map[string]any, error) {
	desc := rows.FieldDescriptions()
	vals := make([]any, len(desc))
	ptrs := make([]any, len(desc))
	for i := range vals {
		ptrs[i] = &vals[i]
	}
	if err := rows.Scan(ptrs...); err != nil {
		return nil, err
	}
	out := map[string]any{}
	for i, fd := range desc {
		out[string(fd.Name)] = vals[i]
	}
	return out, nil
}

func strOr(v any, def string) string {
	if v == nil {
		return def
	}
	s := fmt.Sprint(v)
	if s == "" || s == "<nil>" {
		return def
	}
	return s
}

func strPtr(keys ...any) *string {
	for _, k := range keys {
		if k == nil {
			continue
		}
		s := strings.TrimSpace(fmt.Sprint(k))
		if s != "" && s != "<nil>" {
			return &s
		}
	}
	return nil
}

func nilStr(v any) any {
	if v == nil {
		return nil
	}
	s := strings.TrimSpace(fmt.Sprint(v))
	if s == "" {
		return nil
	}
	return s
}

func dataField(data map[string]any, keys ...string) any {
	for _, k := range keys {
		if v, ok := data[k]; ok {
			return v
		}
	}
	return nil
}

func intField(data map[string]any, keys ...string) any {
	v := dataField(data, keys...)
	if v == nil || fmt.Sprint(v) == "" {
		return nil
	}
	return v
}

func toInt(v any) int {
	n, _ := tests.ToInt64Public(v)
	return int(n)
}

func intPtr(v any) *int64 {
	if v == nil {
		return nil
	}
	n, err := tests.ToInt64Public(v)
	if err != nil {
		return nil
	}
	return &n
}

func floatPtr(v any) *float64 {
	if v == nil {
		return nil
	}
	f, err := tests.ToFloatPublic(v)
	if err != nil {
		return nil
	}
	return &f
}

func uniquePositive(ids []int64) []int64 {
	seen := map[int64]struct{}{}
	var out []int64
	for _, id := range ids {
		if id <= 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}
