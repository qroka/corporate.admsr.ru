package handlers

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"corporate.admsr.ru/backend/internal/auth"
	"corporate.admsr.ru/backend/internal/courses"
	"corporate.admsr.ru/backend/internal/httpx"
	"corporate.admsr.ru/backend/internal/tests"
)

type TestsHandler struct {
	Pool *pgxpool.Pool
	Auth *auth.Service
}

func (h *TestsHandler) decodeBody(r *http.Request) (map[string]any, error) {
	var body map[string]any
	if err := httpx.DecodeJSON(r, &body); err != nil {
		return nil, err
	}
	if body == nil {
		body = map[string]any{}
	}
	return body, nil
}

// viewer возвращает личность вызывающего, полученную ИСКЛЮЧИТЕЛЬНО из серверной
// сессии (cookie corp_session / Bearer).
//
// SEC-008: раньше личность бралась из userId в теле/query, который присылает сам
// клиент. Проверки владельца вида `owner_id != viewer` тем самым обходились —
// злоумышленник подставлял userId владельца и мог редактировать, удалять,
// снимать с публикации чужие формы и читать ответы участников. Параметр body
// сохранён в сигнатуре для совместимости мест вызова, но больше не используется.
func (h *TestsHandler) viewer(_ map[string]any, r *http.Request) int64 {
	if h.Auth == nil {
		return 0
	}
	u, err := h.Auth.CurrentUser(r.Context(), r)
	if err != nil || u == nil {
		return 0
	}
	return u.ID
}

// requireViewer требует авторизованную сессию для операций над формами
// (создание/публикация/снятие/удаление/просмотр ответов). Возвращает 401 иначе.
func (h *TestsHandler) requireViewer(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id := h.viewer(nil, r)
	if id <= 0 {
		httpx.Fail(w, http.StatusUnauthorized, "Требуется авторизация")
		return 0, false
	}
	return id, true
}

func (h *TestsHandler) List(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httpx.MethodNotAllowed(w)
		return
	}
	body, err := h.decodeBody(r)
	if err != nil {
		httpx.Fail(w, http.StatusBadRequest, "Некорректный JSON")
		return
	}
	viewer := h.viewer(body, r)
	ctx := r.Context()

	notInCourse := ""
	if tests.HasCourseTestLinksTable(ctx, h.Pool) {
		notInCourse = `AND NOT EXISTS (
			SELECT 1 FROM public.course_test_links ctl WHERE ctl.test_form_id = f.id
		)`
	}

	var myOfo *int64
	if viewer > 0 {
		var raw *string
		_ = h.Pool.QueryRow(ctx, `SELECT ofo::text FROM public.user_info WHERE id = $1`, viewer).Scan(&raw)
		if raw != nil {
			if matched, _ := regexpMatchDigits(*raw); matched {
				if n, err := strconv.ParseInt(*raw, 10, 64); err == nil {
					myOfo = &n
				}
			}
		}
	}

	drafts, err := h.fetchFormList(ctx, viewer, fmt.Sprintf(`
		SELECT %s FROM public.test_forms f
		WHERE f.status = 'draft' AND f.owner_id = $1 %s
		ORDER BY f.updated_at DESC`, tests.FormSelectCols(), notInCourse), viewer)
	if err != nil {
		httpx.Fail(w, http.StatusInternalServerError, "Ошибка подключения к БД")
		return
	}

	published, err := h.fetchFormList(ctx, viewer, fmt.Sprintf(`
		SELECT %s FROM public.test_forms f
		WHERE f.status = 'published' AND f.visibility = 'public' %s
		ORDER BY f.list_no`, tests.FormSelectCols(), notInCourse))
	if err != nil {
		httpx.Fail(w, http.StatusInternalServerError, "Ошибка подключения к БД")
		return
	}

	mine, err := h.fetchFormList(ctx, viewer, fmt.Sprintf(`
		SELECT %s FROM public.test_forms f
		WHERE f.status = 'published' AND f.owner_id = $1 %s
		ORDER BY f.list_no`, tests.FormSelectCols(), notInCourse), viewer)
	if err != nil {
		httpx.Fail(w, http.StatusInternalServerError, "Ошибка подключения к БД")
		return
	}

	ofoArg := int64(0)
	if myOfo != nil {
		ofoArg = *myOfo
	}
	forMe, err := h.fetchFormList(ctx, viewer, fmt.Sprintf(`
		SELECT %s FROM public.test_forms f
		WHERE f.status = 'published' %s
		  AND (
		    EXISTS (SELECT 1 FROM public.test_audience_users au WHERE au.form_id = f.id AND au.user_id = $1)
		    OR EXISTS (SELECT 1 FROM public.test_audience_ofo ao WHERE ao.form_id = f.id AND ao.ofo_unit_id = $2)
		  )
		ORDER BY f.list_no`, tests.FormSelectCols(), notInCourse), viewer, ofoArg)
	if err != nil {
		httpx.Fail(w, http.StatusInternalServerError, "Ошибка подключения к БД")
		return
	}

	httpx.OK(w, map[string]any{
		"drafts": drafts, "published": published, "mine": mine, "forMe": forMe,
	}, "OK")
}

func (h *TestsHandler) fetchFormList(ctx context.Context, viewer int64, query string, args ...any) ([]map[string]any, error) {
	rows, err := h.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []map[string]any
	for rows.Next() {
		f, err := tests.ScanFormRow(rows)
		if err != nil {
			return nil, err
		}
		form, err := tests.AssembleForm(ctx, h.Pool, f, viewer)
		if err != nil {
			return nil, err
		}
		out = append(out, form)
	}
	if out == nil {
		out = []map[string]any{}
	}
	return out, nil
}

func (h *TestsHandler) Save(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httpx.MethodNotAllowed(w)
		return
	}
	body, err := h.decodeBody(r)
	if err != nil {
		httpx.Fail(w, http.StatusBadRequest, "Некорректный JSON")
		return
	}
	viewer, ok := h.requireViewer(w, r)
	if !ok {
		return
	}
	form, ok := body["form"].(map[string]any)
	if !ok {
		httpx.Fail(w, http.StatusBadRequest, "Не передана форма")
		return
	}
	id, err := tests.PersistForm(r.Context(), h.Pool, form, viewer)
	if err != nil {
		var pe *tests.PersistError
		if errors.As(err, &pe) {
			httpx.Fail(w, http.StatusForbidden, pe.Message)
			return
		}
		httpx.Fail(w, http.StatusInternalServerError, "Ошибка сохранения")
		return
	}
	out, err := tests.LoadForm(r.Context(), h.Pool, id, viewer)
	if err != nil || out == nil {
		httpx.Fail(w, http.StatusInternalServerError, "Ошибка сохранения")
		return
	}
	httpx.OK(w, out, "OK")
}

func (h *TestsHandler) Publish(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httpx.MethodNotAllowed(w)
		return
	}
	body, err := h.decodeBody(r)
	if err != nil {
		httpx.Fail(w, http.StatusBadRequest, "Некорректный JSON")
		return
	}
	viewer, ok := h.requireViewer(w, r)
	if !ok {
		return
	}
	form, ok := body["form"].(map[string]any)
	if !ok {
		httpx.Fail(w, http.StatusBadRequest, "Не передана форма")
		return
	}
	id, err := tests.PersistForm(r.Context(), h.Pool, form, viewer)
	if err != nil {
		var pe *tests.PersistError
		if errors.As(err, &pe) {
			httpx.Fail(w, http.StatusForbidden, pe.Message)
			return
		}
		httpx.Fail(w, http.StatusInternalServerError, "Ошибка публикации")
		return
	}
	tok := make([]byte, 16)
	_, _ = rand.Read(tok)
	token := hex.EncodeToString(tok)
	_, err = h.Pool.Exec(r.Context(), `
		UPDATE public.test_forms
		SET status = 'published',
		    published_at = COALESCE(published_at, now()),
		    list_no = COALESCE(list_no, nextval('public.test_forms_list_no_seq')),
		    access_token = CASE WHEN access_by_link AND access_token IS NULL THEN $2 ELSE access_token END,
		    updated_at = now()
		WHERE id = $1`, id, token)
	if err != nil {
		httpx.Fail(w, http.StatusInternalServerError, "Ошибка публикации")
		return
	}
	out, err := tests.LoadForm(r.Context(), h.Pool, id, viewer)
	if err != nil || out == nil {
		httpx.Fail(w, http.StatusInternalServerError, "Ошибка публикации")
		return
	}
	httpx.OK(w, out, "OK")
}

func (h *TestsHandler) Unpublish(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httpx.MethodNotAllowed(w)
		return
	}
	body, err := h.decodeBody(r)
	if err != nil {
		httpx.Fail(w, http.StatusBadRequest, "Некорректный JSON")
		return
	}
	viewer, ok := h.requireViewer(w, r)
	if !ok {
		return
	}
	formID, _ := tests.ToInt64Public(body["formId"])
	if formID <= 0 {
		httpx.Fail(w, http.StatusBadRequest, "Не передан formId")
		return
	}
	var ownerID *int64
	err = h.Pool.QueryRow(r.Context(), `SELECT owner_id FROM public.test_forms WHERE id = $1`, formID).Scan(&ownerID)
	if errors.Is(err, pgx.ErrNoRows) {
		httpx.Fail(w, http.StatusNotFound, "Форма не найдена")
		return
	}
	if err != nil {
		httpx.Fail(w, http.StatusInternalServerError, "Ошибка подключения к БД")
		return
	}
	// Форму без владельца (nil) может трогать только админ: иначе любой
	// авторизованный пользователь мог бы снять с публикации легаси-форму.
	if ownerID == nil || *ownerID != viewer {
		httpx.Fail(w, http.StatusForbidden, "Снять с публикации может только создатель")
		return
	}
	_, _ = h.Pool.Exec(r.Context(), `UPDATE public.test_forms SET status = 'draft', updated_at = now() WHERE id = $1`, formID)
	out, _ := tests.LoadForm(r.Context(), h.Pool, formID, viewer)
	httpx.OK(w, out, "OK")
}

func (h *TestsHandler) Delete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httpx.MethodNotAllowed(w)
		return
	}
	body, err := h.decodeBody(r)
	if err != nil {
		httpx.Fail(w, http.StatusBadRequest, "Некорректный JSON")
		return
	}
	viewer, ok := h.requireViewer(w, r)
	if !ok {
		return
	}
	formID, _ := tests.ToInt64Public(body["formId"])
	if formID <= 0 {
		httpx.Fail(w, http.StatusBadRequest, "Не передан formId")
		return
	}
	var ownerID *int64
	var status string
	err = h.Pool.QueryRow(r.Context(), `SELECT owner_id, status FROM public.test_forms WHERE id = $1`, formID).Scan(&ownerID, &status)
	if errors.Is(err, pgx.ErrNoRows) {
		httpx.Fail(w, http.StatusNotFound, "Форма не найдена")
		return
	}
	if err != nil {
		httpx.Fail(w, http.StatusInternalServerError, "Ошибка подключения к БД")
		return
	}
	if ownerID == nil || *ownerID != viewer {
		httpx.Fail(w, http.StatusForbidden, "Удалить может только создатель")
		return
	}
	if status != "draft" {
		httpx.Fail(w, http.StatusConflict, "Сначала уберите форму из публикации")
		return
	}
	_, _ = h.Pool.Exec(r.Context(), `DELETE FROM public.test_forms WHERE id = $1`, formID)
	httpx.OK(w, map[string]any{"deleted": true}, "OK")
}

func (h *TestsHandler) Direct(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httpx.MethodNotAllowed(w)
		return
	}
	body, err := h.decodeBody(r)
	if err != nil {
		httpx.Fail(w, http.StatusBadRequest, "Некорректный JSON")
		return
	}
	viewer, ok := h.requireViewer(w, r)
	if !ok {
		return
	}
	formID, _ := tests.ToInt64Public(body["formId"])
	mode := fmt.Sprint(body["mode"])
	force := tests.Bool(body["force"])
	ids := uniqueInt64FromAny(body["ids"])
	if formID <= 0 {
		httpx.Fail(w, http.StatusBadRequest, "Не передан formId")
		return
	}
	if mode != "ofo" && mode != "users" {
		httpx.Fail(w, http.StatusBadRequest, "mode должен быть ofo или users")
		return
	}
	if len(ids) == 0 {
		httpx.Fail(w, http.StatusBadRequest, "Не выбраны получатели")
		return
	}
	var ownerID *int64
	err = h.Pool.QueryRow(r.Context(), `SELECT owner_id FROM public.test_forms WHERE id = $1`, formID).Scan(&ownerID)
	if errors.Is(err, pgx.ErrNoRows) {
		httpx.Fail(w, http.StatusNotFound, "Форма не найдена")
		return
	}
	if err != nil {
		httpx.Fail(w, http.StatusInternalServerError, "Ошибка подключения к БД")
		return
	}
	if ownerID == nil || *ownerID != viewer {
		httpx.Fail(w, http.StatusForbidden, "Направлять может только создатель")
		return
	}

	table, col := "public.test_audience_ofo", "ofo_unit_id"
	if mode == "users" {
		table, col = "public.test_audience_users", "user_id"
	}
	rows, err := h.Pool.Query(r.Context(), fmt.Sprintf(`SELECT %s FROM %s WHERE form_id = $1`, col, table), formID)
	if err != nil {
		httpx.Fail(w, http.StatusInternalServerError, "Ошибка подключения к БД")
		return
	}
	var existing []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err == nil {
			existing = append(existing, id)
		}
	}
	rows.Close()

	var already []int64
	for _, id := range ids {
		for _, ex := range existing {
			if ex == id {
				already = append(already, id)
				break
			}
		}
	}
	if len(already) > 0 && !force {
		httpx.OK(w, map[string]any{"needConfirm": true, "already": already}, "OK")
		return
	}
	for _, t := range ids {
		_, _ = h.Pool.Exec(r.Context(), fmt.Sprintf(`
			INSERT INTO %s (form_id, %s, source) VALUES ($1,$2,'directed')
			ON CONFLICT (form_id, %s) DO NOTHING`, table, col, col), formID, t)
	}
	out, _ := tests.LoadForm(r.Context(), h.Pool, formID, viewer)
	httpx.OK(w, out, "OK")
}

func (h *TestsHandler) Submit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httpx.MethodNotAllowed(w)
		return
	}
	body, err := h.decodeBody(r)
	if err != nil {
		httpx.Fail(w, http.StatusBadRequest, "Некорректный JSON")
		return
	}
	viewer := h.viewer(body, r)
	ctx := r.Context()

	token := strings.TrimSpace(fmt.Sprint(body["token"]))
	answers, _ := body["answers"].(map[string]any)
	if answers == nil {
		answers = map[string]any{}
	}
	var durationSec *int
	if body["durationSec"] != nil {
		if n, err := tests.ToInt64Public(body["durationSec"]); err == nil {
			v := int(n)
			durationSec = &v
		}
	}

	viaLink := false
	var respondentToken, guestName *string
	var guestOfo *int64
	var userID *int64
	if viewer > 0 {
		userID = &viewer
	}

	var formRow tests.DBFormRow
	var formID int64

	if token != "" && token != "<nil>" {
		row := h.Pool.QueryRow(ctx, `
			SELECT `+tests.FormSelectCols()+`
			FROM public.test_forms
			WHERE access_token = $1 AND status = 'published' AND access_by_link = true`, token)
		formRow, err = tests.ScanFormRow(row)
		if errors.Is(err, pgx.ErrNoRows) {
			httpx.Fail(w, http.StatusNotFound, "Ссылка недействительна")
			return
		}
		if err != nil {
			httpx.Fail(w, http.StatusInternalServerError, "Ошибка подключения к БД")
			return
		}
		formID = formRow.ID
		viaLink = true
		mode := "any"
		if formRow.LinkAccess != nil {
			mode = *formRow.LinkAccess
		}
		if viewer > 0 {
			userID = &viewer
		} else {
			if mode == "authorized" {
				httpx.Fail(w, http.StatusForbidden, "Форма доступна только авторизованным — войдите в портал")
				return
			}
			userID = nil
			if rt, ok := body["respondentToken"].(string); ok {
				if len(rt) > 100 {
					rt = rt[:100]
				}
				respondentToken = &rt
			}
			gn := strings.TrimSpace(fmt.Sprint(body["guestName"]))
			if gn == "" || gn == "<nil>" {
				httpx.Fail(w, http.StatusBadRequest, "Укажите ФИО")
				return
			}
			guestName = &gn
			if gofo, err := tests.ToInt64Public(body["guestOfoId"]); err == nil && gofo > 0 {
				guestOfo = &gofo
			}
		}
	} else {
		formID, _ = tests.ToInt64Public(body["formId"])
		if formID <= 0 {
			httpx.Fail(w, http.StatusBadRequest, "Не передан formId")
			return
		}
		row := h.Pool.QueryRow(ctx, `SELECT `+tests.FormSelectCols()+` FROM public.test_forms WHERE id = $1`, formID)
		formRow, err = tests.ScanFormRow(row)
		if errors.Is(err, pgx.ErrNoRows) {
			httpx.Fail(w, http.StatusNotFound, "Форма не найдена")
			return
		}
		if err != nil {
			httpx.Fail(w, http.StatusInternalServerError, "Ошибка подключения к БД")
			return
		}
		if formRow.Status != "published" {
			httpx.Fail(w, http.StatusConflict, "Форма не опубликована")
			return
		}
		if formRow.Visibility == "private" {
			allowed := viewer > 0 && formRow.OwnerID != nil && *formRow.OwnerID == viewer
			if !allowed && viewer > 0 {
				var n int
				_ = h.Pool.QueryRow(ctx, `SELECT 1 FROM public.test_audience_users WHERE form_id = $1 AND user_id = $2 LIMIT 1`, formID, viewer).Scan(&n)
				if n == 1 {
					allowed = true
				}
			}
			if !allowed && viewer > 0 {
				var raw *string
				_ = h.Pool.QueryRow(ctx, `SELECT ofo::text FROM public.user_info WHERE id = $1`, viewer).Scan(&raw)
				if raw != nil {
					if ok, _ := regexpMatchDigits(*raw); ok {
						if ofo, err := strconv.ParseInt(*raw, 10, 64); err == nil {
							var n int
							_ = h.Pool.QueryRow(ctx, `SELECT 1 FROM public.test_audience_ofo WHERE form_id = $1 AND ofo_unit_id = $2 LIMIT 1`, formID, ofo).Scan(&n)
							if n == 1 {
								allowed = true
							}
						}
					}
				}
			}
			if !allowed {
				httpx.Fail(w, http.StatusForbidden, "Нет доступа к этой форме")
				return
			}
		}
	}

	if !(formRow.Kind == "poll" && formRow.AllowRevote) {
		allowedAttempts := 1
		if formRow.LimitAttempts {
			allowedAttempts = int(formRow.Attempts)
			if allowedAttempts < 1 {
				allowedAttempts = 1
			}
		}
		if userID != nil {
			var cnt int
			_ = h.Pool.QueryRow(ctx, `
				SELECT COUNT(*) FROM public.test_attempts
				WHERE form_id = $1 AND user_id = $2 AND status = 'completed'`, formID, *userID).Scan(&cnt)
			if cnt >= allowedAttempts {
				httpx.Fail(w, http.StatusConflict, "Вы уже прошли эту форму допустимое число раз")
				return
			}
		} else if respondentToken != nil {
			var cnt int
			_ = h.Pool.QueryRow(ctx, `
				SELECT COUNT(*) FROM public.test_attempts
				WHERE form_id = $1 AND respondent_token = $2 AND status = 'completed'`, formID, *respondentToken).Scan(&cnt)
			if cnt >= allowedAttempts {
				httpx.Fail(w, http.StatusConflict, "С этого устройства форма уже пройдена")
				return
			}
		}
	}

	qRows, err := h.Pool.Query(ctx, `SELECT id, type, correct_value FROM public.test_questions WHERE form_id = $1 ORDER BY position, id`, formID)
	if err != nil {
		httpx.Fail(w, http.StatusInternalServerError, "Ошибка записи прохождения")
		return
	}
	var questions []tests.QuestionEval
	for qRows.Next() {
		var q tests.QuestionEval
		if err := qRows.Scan(&q.ID, &q.Type, &q.CorrectValue); err != nil {
			qRows.Close()
			httpx.Fail(w, http.StatusInternalServerError, "Ошибка записи прохождения")
			return
		}
		questions = append(questions, q)
	}
	qRows.Close()

	ip := clientIP(r)
	ua := r.UserAgent()
	dur := 0
	if durationSec != nil {
		dur = *durationSec
	}

	tx, err := h.Pool.Begin(ctx)
	if err != nil {
		httpx.Fail(w, http.StatusInternalServerError, "Ошибка записи прохождения")
		return
	}
	defer tx.Rollback(ctx)

	var attemptID int64
	err = tx.QueryRow(ctx, `
		INSERT INTO public.test_attempts
		  (form_id, user_id, status, current_page, started_at, finished_at, duration_sec, ip, user_agent, via_link, respondent_token, guest_name, guest_ofo_id)
		VALUES
		  ($1,$2,'completed',0, now() - make_interval(secs => $3::int), now(), $4,$5,$6,$7,$8,$9,$10)
		RETURNING id`,
		formID, userID, dur, durationSec, ip, ua, viaLink, respondentToken, guestName, guestOfo,
	).Scan(&attemptID)
	if err != nil {
		httpx.Fail(w, http.StatusInternalServerError, "Ошибка записи прохождения")
		return
	}

	eval, err := tests.EvaluateAnswers(ctx, h.Pool, formRow, questions, answers)
	if err != nil {
		httpx.Fail(w, http.StatusInternalServerError, "Ошибка записи прохождения")
		return
	}

	for _, d := range eval.Details {
		var ic *bool
		if d.IsCorrect != nil {
			ic = d.IsCorrect
		}
		var ansID int64
		err = tx.QueryRow(ctx, `
			INSERT INTO public.test_answers (attempt_id, question_id, text_value, number_value, is_correct, answered)
			VALUES ($1,$2,$3,$4,$5,$6) RETURNING id`,
			attemptID, d.QuestionID, d.TextValue, d.NumberValue, ic, d.Answered,
		).Scan(&ansID)
		if err != nil {
			httpx.Fail(w, http.StatusInternalServerError, "Ошибка записи прохождения")
			return
		}
		for _, optID := range d.Selected {
			_, _ = tx.Exec(ctx, `INSERT INTO public.test_answer_options (answer_id, option_id) VALUES ($1,$2) ON CONFLICT DO NOTHING`, ansID, optID)
		}
	}

	if eval.Score != nil {
		_, _ = tx.Exec(ctx, `
			UPDATE public.test_attempts SET score = $1, max_score = 100, passed = $2 WHERE id = $3`,
			*eval.Score, eval.Passed, attemptID)
	}

	if err := tx.Commit(ctx); err != nil {
		httpx.Fail(w, http.StatusInternalServerError, "Ошибка записи прохождения")
		return
	}

	httpx.OK(w, map[string]any{
		"attemptId": attemptID, "score": eval.Score, "passed": eval.Passed,
		"correctCount": eval.CorrectCount, "scorable": eval.Scorable,
	}, "OK")
}

func (h *TestsHandler) Stats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httpx.MethodNotAllowed(w)
		return
	}
	body, err := h.decodeBody(r)
	if err != nil {
		httpx.Fail(w, http.StatusBadRequest, "Некорректный JSON")
		return
	}
	requester, ok := h.requireViewer(w, r)
	if !ok {
		return
	}
	formID, _ := tests.ToInt64Public(body["formId"])
	if formID <= 0 {
		httpx.Fail(w, http.StatusBadRequest, "Не передан formId")
		return
	}
	ctx := r.Context()

	row := h.Pool.QueryRow(ctx, `SELECT `+tests.FormSelectCols()+` FROM public.test_forms WHERE id = $1`, formID)
	formRow, err := tests.ScanFormRow(row)
	if errors.Is(err, pgx.ErrNoRows) {
		httpx.Fail(w, http.StatusNotFound, "Форма не найдена")
		return
	}
	if err != nil {
		httpx.Fail(w, http.StatusInternalServerError, "Ошибка подключения к БД")
		return
	}
	// SEC-008: статистика формы (агрегаты по попыткам, баллы, прохождения)
	// раньше отдавалась любому по formId без проверки. Теперь — только создателю.
	if formRow.OwnerID == nil || *formRow.OwnerID != requester {
		httpx.Fail(w, http.StatusForbidden, "Доступно только создателю формы")
		return
	}

	isTest := formRow.Kind == "test"
	anonymous := formRow.Anonymous
	pct := func(p, w int) int {
		if w <= 0 {
			return 0
		}
		return int(math.Round(float64(p) / float64(w) * 100))
	}

	var completions, started, avgTime int
	var lastAt any
	var avgScore *int
	var passedCnt, scoredCnt int
	_ = h.Pool.QueryRow(ctx, `
		SELECT
		  COUNT(*) FILTER (WHERE status = 'completed'),
		  COUNT(*),
		  COALESCE(ROUND(AVG(duration_sec) FILTER (WHERE status = 'completed'))::int, 0),
		  MAX(finished_at),
		  ROUND(AVG(score) FILTER (WHERE score IS NOT NULL))::int,
		  COUNT(*) FILTER (WHERE passed IS TRUE),
		  COUNT(*) FILTER (WHERE passed IS NOT NULL)
		FROM public.test_attempts WHERE form_id = $1`, formID).Scan(
		&completions, &started, &avgTime, &lastAt, &avgScore, &passedCnt, &scoredCnt)

	ofoNames := map[int64]string{}
	oRows, _ := h.Pool.Query(ctx, `SELECT id, name FROM public.ofo_unit`)
	for oRows != nil && oRows.Next() {
		var id int64
		var name string
		if err := oRows.Scan(&id, &name); err == nil {
			ofoNames[id] = name
		}
	}
	if oRows != nil {
		oRows.Close()
	}

	var byOfo []map[string]any
	oqRows, err := h.Pool.Query(ctx, `
		SELECT ofo, COUNT(*) AS c FROM (
		  SELECT u.ofo::int AS ofo
		  FROM public.test_attempts a JOIN public.user_info u ON u.id = a.user_id
		  WHERE a.form_id = $1 AND a.status = 'completed' AND a.user_id IS NOT NULL AND u.ofo ~ '^[0-9]+$'
		  UNION ALL
		  SELECT guest_ofo_id AS ofo
		  FROM public.test_attempts
		  WHERE form_id = $2 AND status = 'completed' AND user_id IS NULL AND guest_ofo_id IS NOT NULL
		) t GROUP BY ofo ORDER BY c DESC`, formID, formID)
	if err == nil {
		for oqRows.Next() {
			var ofo int64
			var c int
			if err := oqRows.Scan(&ofo, &c); err == nil {
				name := ofoNames[ofo]
				if name == "" {
					name = fmt.Sprintf("ОФО #%d", ofo)
				}
				byOfo = append(byOfo, map[string]any{"name": name, "count": c, "percent": pct(c, completions)})
			}
		}
		oqRows.Close()
	}

	var participants any
	if !anonymous {
		var plist []map[string]any
		pqRows, _ := h.Pool.Query(ctx, `
			SELECT DISTINCT u.id, TRIM(CONCAT_WS(' ', u.surname, u.firstname, u.lastname)) AS name
			FROM public.test_attempts a JOIN public.user_info u ON u.id = a.user_id
			WHERE a.form_id = $1 AND a.status = 'completed' AND a.user_id IS NOT NULL
			ORDER BY name`, formID)
		for pqRows != nil && pqRows.Next() {
			var id int64
			var name string
			if err := pqRows.Scan(&id, &name); err == nil {
				if name == "" {
					name = fmt.Sprintf("ID %d", id)
				}
				plist = append(plist, map[string]any{"id": id, "name": name, "guest": false})
			}
		}
		if pqRows != nil {
			pqRows.Close()
		}
		gi := 0
		gqRows, _ := h.Pool.Query(ctx, `
			SELECT guest_name FROM public.test_attempts
			WHERE form_id = $1 AND status = 'completed' AND user_id IS NULL AND guest_name IS NOT NULL
			ORDER BY guest_name`, formID)
		for gqRows != nil && gqRows.Next() {
			var gn string
			if err := gqRows.Scan(&gn); err == nil {
				gi++
				plist = append(plist, map[string]any{"id": -gi, "name": gn + " (гость)", "guest": true})
			}
		}
		if gqRows != nil {
			gqRows.Close()
		}
		participants = plist
	}

	var guestCompletions int
	_ = h.Pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM public.test_attempts
		WHERE form_id = $1 AND status = 'completed' AND user_id IS NULL AND via_link = true`, formID).Scan(&guestCompletions)

	qStmt, err := h.Pool.Query(ctx, `SELECT * FROM public.test_questions WHERE form_id = $1 ORDER BY position, id`, formID)
	if err != nil {
		httpx.Fail(w, http.StatusInternalServerError, "Ошибка подключения к БД")
		return
	}
	defer qStmt.Close()

	var questions []map[string]any
	var hardest map[string]any
	for qStmt.Next() {
		qmap, err := scanRowToMap(qStmt)
		if err != nil {
			continue
		}
		qid, _ := tests.ToInt64Public(qmap["id"])
		qtype := fmt.Sprint(qmap["type"])
		title := fmt.Sprint(qmap["title"])

		var answered int
		_ = h.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM public.test_answers WHERE question_id = $1 AND answered = true`, qid).Scan(&answered)

		var correctRate *int
		if isTest {
			var c, t int
			_ = h.Pool.QueryRow(ctx, `
				SELECT COUNT(*) FILTER (WHERE is_correct), COUNT(*) FILTER (WHERE is_correct IS NOT NULL)
				FROM public.test_answers WHERE question_id = $1`, qid).Scan(&c, &t)
			if t > 0 {
				v := pct(c, t)
				correctRate = &v
			}
		}

		var options any
		isText := false
		var avgNumber any

		switch qtype {
		case "single", "multiple", "dropdown":
			optRows, _ := h.Pool.Query(ctx, `
				SELECT o.id, o.text, o.is_correct, COUNT(ao.answer_id) c
				FROM public.test_options o
				LEFT JOIN public.test_answer_options ao ON ao.option_id = o.id
				WHERE o.question_id = $1
				GROUP BY o.id, o.text, o.is_correct, o.position
				ORDER BY o.position, o.id`, qid)
			var opts []map[string]any
			for optRows != nil && optRows.Next() {
				var oid int64
				var text string
				var ic bool
				var c int
				if err := optRows.Scan(&oid, &text, &ic, &c); err == nil {
					opts = append(opts, map[string]any{
						"id": strconv.FormatInt(oid, 10), "label": text, "count": c,
						"percent": pct(c, completions), "correct": ic,
					})
				}
			}
			if optRows != nil {
				optRows.Close()
			}
			options = opts
		case "yesno":
			cnt := map[string]int{"yes": 0, "no": 0}
			ynRows, _ := h.Pool.Query(ctx, `
				SELECT text_value, COUNT(*) FROM public.test_answers
				WHERE question_id = $1 AND answered = true GROUP BY text_value`, qid)
			for ynRows != nil && ynRows.Next() {
				var v string
				var c int
				if err := ynRows.Scan(&v, &c); err == nil {
					cnt[v] = c
				}
			}
			if ynRows != nil {
				ynRows.Close()
			}
			cv := fmt.Sprint(qmap["correct_value"])
			options = []map[string]any{
				{"label": "Да", "count": cnt["yes"], "percent": pct(cnt["yes"], completions), "correct": cv == "yes"},
				{"label": "Нет", "count": cnt["no"], "percent": pct(cnt["no"], completions), "correct": cv == "no"},
			}
		case "scale":
			scaleMap := map[int]int{}
			sRows, _ := h.Pool.Query(ctx, `
				SELECT number_value, COUNT(*) FROM public.test_answers
				WHERE question_id = $1 AND answered = true AND number_value IS NOT NULL GROUP BY number_value`, qid)
			for sRows != nil && sRows.Next() {
				var v float64
				var c int
				if err := sRows.Scan(&v, &c); err == nil {
					scaleMap[int(math.Round(v))] = c
				}
			}
			if sRows != nil {
				sRows.Close()
			}
			min := toIntDefaultMap(qmap["scale_min"], 1)
			max := toIntDefaultMap(qmap["scale_max"], 5)
			if max < min {
				min, max = max, min
			}
			var opts []map[string]any
			for n := min; n <= max && len(opts) < 50; n++ {
				c := scaleMap[n]
				opts = append(opts, map[string]any{"label": strconv.Itoa(n), "count": c, "percent": pct(c, completions), "correct": false})
			}
			options = opts
		case "number":
			var avg *float64
			_ = h.Pool.QueryRow(ctx, `SELECT ROUND(AVG(number_value), 2) FROM public.test_answers WHERE question_id = $1 AND answered = true`, qid).Scan(&avg)
			if avg != nil {
				avgNumber = *avg
			}
			isText = true
		default:
			isText = true
		}

		qOut := map[string]any{
			"id": strconv.FormatInt(qid, 10), "title": title, "type": qtype,
			"answered": answered, "skipped": maxInt(0, completions-answered),
			"options": options, "correctRate": correctRate, "isText": isText, "avgNumber": avgNumber,
		}
		questions = append(questions, qOut)
		if correctRate != nil && (hardest == nil || *correctRate < hardest["correctRate"].(int)) {
			t := title
			if t == "" {
				t = "Без названия"
			}
			hardest = map[string]any{"title": t, "correctRate": *correctRate}
		}
	}

	var avgScoreOut any
	if isTest && avgScore != nil {
		avgScoreOut = *avgScore
	}
	var passRate any
	if isTest && scoredCnt > 0 {
		passRate = pct(passedCnt, scoredCnt)
	}

	httpx.OK(w, map[string]any{
		"completions": completions, "started": started, "completionRate": pct(completions, started),
		"avgTimeSec": avgTime, "lastAt": lastAt, "avgScore": avgScoreOut, "passRate": passRate,
		"hardest": hardest, "byOfo": byOfo, "participants": participants,
		"guestCompletions": guestCompletions, "questions": questions,
	}, "OK")
}

func (h *TestsHandler) Participant(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httpx.MethodNotAllowed(w)
		return
	}
	body, err := h.decodeBody(r)
	if err != nil {
		httpx.Fail(w, http.StatusBadRequest, "Некорректный JSON")
		return
	}
	requester, ok := h.requireViewer(w, r)
	if !ok {
		return
	}
	formID, _ := tests.ToInt64Public(body["formId"])
	participantID, _ := tests.ToInt64Public(body["participantId"])
	if formID <= 0 || participantID <= 0 {
		httpx.Fail(w, http.StatusBadRequest, "Нужны formId и participantId")
		return
	}
	ctx := r.Context()

	row := h.Pool.QueryRow(ctx, `SELECT `+tests.FormSelectCols()+` FROM public.test_forms WHERE id = $1`, formID)
	formRow, err := tests.ScanFormRow(row)
	if errors.Is(err, pgx.ErrNoRows) {
		httpx.Fail(w, http.StatusNotFound, "Форма не найдена")
		return
	}
	if err != nil {
		httpx.Fail(w, http.StatusInternalServerError, "Ошибка подключения к БД")
		return
	}
	if formRow.OwnerID == nil || *formRow.OwnerID != requester {
		httpx.Fail(w, http.StatusForbidden, "Доступно только создателю формы")
		return
	}
	if formRow.Anonymous {
		httpx.Fail(w, http.StatusForbidden, "Форма анонимна — ответы участников скрыты")
		return
	}

	var attemptID int64
	var score *float64
	var passed *bool
	var finishedAt, durationSec any
	err = h.Pool.QueryRow(ctx, `
		SELECT id, score, passed, finished_at, duration_sec FROM public.test_attempts
		WHERE form_id = $1 AND user_id = $2 AND status = 'completed'
		ORDER BY finished_at DESC NULLS LAST, id DESC LIMIT 1`, formID, participantID).Scan(
		&attemptID, &score, &passed, &finishedAt, &durationSec)
	if errors.Is(err, pgx.ErrNoRows) {
		httpx.Fail(w, http.StatusNotFound, "У участника нет завершённых прохождений")
		return
	}
	if err != nil {
		httpx.Fail(w, http.StatusInternalServerError, "Ошибка подключения к БД")
		return
	}
	yn := func(v string) string {
		switch v {
		case "yes":
			return "Да"
		case "no":
			return "Нет"
		default:
			return v
		}
	}

	qRows, _ := h.Pool.Query(ctx, `SELECT id, type, title, correct_value FROM public.test_questions WHERE form_id = $1 ORDER BY position, id`, formID)
	var answers []map[string]any
	for qRows != nil && qRows.Next() {
		var qid int64
		var qtype, title string
		var correctValue *string
		if err := qRows.Scan(&qid, &qtype, &title, &correctValue); err != nil {
			continue
		}
		optText := map[int64]string{}
		var correctOptIDs []int64
		if qtype == "single" || qtype == "multiple" || qtype == "dropdown" {
			optRows, _ := h.Pool.Query(ctx, `SELECT id, text, is_correct FROM public.test_options WHERE question_id = $1 ORDER BY position, id`, qid)
			for optRows != nil && optRows.Next() {
				var oid int64
				var text string
				var ic bool
				if err := optRows.Scan(&oid, &text, &ic); err == nil {
					optText[oid] = text
					if ic {
						correctOptIDs = append(correctOptIDs, oid)
					}
				}
			}
			if optRows != nil {
				optRows.Close()
			}
		}

		var ansID int64
		var textValue *string
		var numberValue *float64
		var isCorrect *bool
		var answered bool
		err := h.Pool.QueryRow(ctx, `
			SELECT id, text_value, number_value, is_correct, answered
			FROM public.test_answers WHERE attempt_id = $1 AND question_id = $2`, attemptID, qid).Scan(
			&ansID, &textValue, &numberValue, &isCorrect, &answered)
		hasAns := err == nil

		userAnswer := "— нет ответа"
		if hasAns && answered {
			switch qtype {
			case "single", "multiple", "dropdown":
				selRows, _ := h.Pool.Query(ctx, `SELECT option_id FROM public.test_answer_options WHERE answer_id = $1`, ansID)
				var labels []string
				for selRows != nil && selRows.Next() {
					var oid int64
					if err := selRows.Scan(&oid); err == nil {
						lbl := optText[oid]
						if lbl == "" {
							lbl = "?"
						}
						labels = append(labels, lbl)
					}
				}
				if selRows != nil {
					selRows.Close()
				}
				if len(labels) > 0 {
					userAnswer = strings.Join(labels, ", ")
				}
			case "yesno":
				if textValue != nil {
					userAnswer = yn(*textValue)
				}
			case "scale", "number":
				if numberValue != nil {
					userAnswer = fmt.Sprint(*numberValue)
				} else if textValue != nil {
					userAnswer = *textValue
				}
			default:
				if textValue != nil {
					userAnswer = *textValue
				}
			}
		}

		correctAnswer := ""
		switch qtype {
		case "single", "multiple", "dropdown":
			var labels []string
			for _, id := range correctOptIDs {
				lbl := optText[id]
				if lbl == "" {
					lbl = "?"
				}
				labels = append(labels, lbl)
			}
			correctAnswer = strings.Join(labels, ", ")
		case "yesno":
			if correctValue != nil {
				correctAnswer = yn(*correctValue)
			}
		default:
			if correctValue != nil {
				correctAnswer = *correctValue
			}
		}

		var passedVal any
		if isCorrect != nil {
			passedVal = *isCorrect
		}
		answers = append(answers, map[string]any{
			"title": title, "type": qtype, "userAnswer": userAnswer, "correctAnswer": correctAnswer,
			"isCorrect": passedVal, "answered": hasAns && answered,
		})
	}
	if qRows != nil {
		qRows.Close()
	}

	var passedOut any
	if passed != nil {
		passedOut = *passed
	}
	var durOut any
	if durationSec != nil {
		if n, err := tests.ToInt64Public(durationSec); err == nil {
			durOut = int(n)
		}
	}

	httpx.OK(w, map[string]any{
		"attempt": map[string]any{
			"score": score, "passed": passedOut, "finishedAt": finishedAt, "durationSec": durOut,
		},
		"answers": answers,
	}, "OK")
}

func (h *TestsHandler) ByToken(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httpx.MethodNotAllowed(w)
		return
	}
	body, err := h.decodeBody(r)
	if err != nil {
		httpx.Fail(w, http.StatusBadRequest, "Некорректный JSON")
		return
	}
	viewer := h.viewer(body, r)
	token := strings.TrimSpace(fmt.Sprint(body["token"]))
	if token == "" || token == "<nil>" {
		if t := r.URL.Query().Get("token"); t != "" {
			token = t
		}
	}
	if token == "" {
		httpx.Fail(w, http.StatusBadRequest, "Не передан token")
		return
	}
	respondent := strings.TrimSpace(fmt.Sprint(body["respondentToken"]))
	if len(respondent) > 100 {
		respondent = respondent[:100]
	}

	ctx := r.Context()
	row := h.Pool.QueryRow(ctx, `
		SELECT `+tests.FormSelectCols()+`
		FROM public.test_forms
		WHERE access_token = $1 AND status = 'published' AND access_by_link = true`, token)
	formRow, err := tests.ScanFormRow(row)
	if errors.Is(err, pgx.ErrNoRows) {
		httpx.Fail(w, http.StatusNotFound, "Ссылка недействительна")
		return
	}
	if err != nil {
		httpx.Fail(w, http.StatusInternalServerError, "Ошибка подключения к БД")
		return
	}

	today := time.Now().Format("2006-01-02")
	if formRow.UseStart && formRow.StartsAt != nil && *formRow.StartsAt > today {
		httpx.Fail(w, http.StatusConflict, fmt.Sprintf("Форма ещё не началась (с %s)", *formRow.StartsAt))
		return
	}
	if formRow.UseEnd && formRow.EndsAt != nil && *formRow.EndsAt < today {
		httpx.Fail(w, http.StatusConflict, fmt.Sprintf("Приём ответов завершён (%s)", *formRow.EndsAt))
		return
	}

	out, err := tests.AssembleForm(ctx, h.Pool, formRow, viewer)
	if err != nil {
		httpx.Fail(w, http.StatusInternalServerError, "Ошибка подключения к БД")
		return
	}

	used := 0
	if viewer > 0 {
		_ = h.Pool.QueryRow(ctx, `
			SELECT COUNT(*) FROM public.test_attempts
			WHERE form_id = $1 AND user_id = $2 AND status = 'completed'`, formRow.ID, viewer).Scan(&used)
	} else if respondent != "" && respondent != "<nil>" {
		_ = h.Pool.QueryRow(ctx, `
			SELECT COUNT(*) FROM public.test_attempts
			WHERE form_id = $1 AND respondent_token = $2 AND status = 'completed'`, formRow.ID, respondent).Scan(&used)
	}
	out["attemptsUsed"] = used
	httpx.OK(w, out, "OK")
}

// AttemptStart requires session auth (like PHP courses_common auth_require_user).
func (h *TestsHandler) AttemptStart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
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
	body, err := h.decodeBody(r)
	if err != nil {
		httpx.Fail(w, http.StatusBadRequest, "Некорректный JSON")
		return
	}
	ctx := r.Context()
	formID, _ := tests.ToInt64Public(body["formId"])
	enrollmentID, _ := tests.ToInt64Public(body["enrollmentId"])
	linkID, _ := tests.ToInt64Public(body["courseTestLinkId"])
	preview := tests.Bool(body["preview"])

	if formID <= 0 && linkID <= 0 {
		httpx.Fail(w, http.StatusBadRequest, "Укажите formId или courseTestLinkId")
		return
	}

	var link map[string]any
	if linkID > 0 {
		link, err = scanOneMap(ctx, h.Pool, `SELECT * FROM public.course_test_links WHERE id = $1`, linkID)
		if errors.Is(err, pgx.ErrNoRows) || link == nil {
			httpx.Fail(w, http.StatusNotFound, "Связь теста не найдена")
			return
		}
		formID, _ = tests.ToInt64Public(link["test_form_id"])
	} else if formID > 0 {
		link, _ = scanOneMap(ctx, h.Pool, `SELECT * FROM public.course_test_links WHERE test_form_id = $1 LIMIT 1`, formID)
	}

	formRow, err := loadFormRow(ctx, h.Pool, formID)
	if errors.Is(err, pgx.ErrNoRows) {
		httpx.Fail(w, http.StatusNotFound, "Форма не найдена")
		return
	}
	if err != nil {
		httpx.Fail(w, http.StatusInternalServerError, "Ошибка подключения к БД")
		return
	}

	var enr map[string]any
	if link != nil {
		if enrollmentID <= 0 {
			httpx.Fail(w, http.StatusBadRequest, "Для теста курса нужен enrollmentId")
			return
		}
		enr, err = h.requireEnrollmentAccess(ctx, enrollmentID, user)
		if err != nil {
			writeTestsErr(w, err)
			return
		}
		if cv, _ := tests.ToInt64Public(enr["course_version_id"]); cv != tests.ToInt64Must(link["course_version_id"]) {
			httpx.Fail(w, http.StatusForbidden, "Enrollment не соответствует тесту")
			return
		}
		st := fmt.Sprint(enr["status"])
		if st != "in_progress" && st != "not_started" && st != "overdue" {
			httpx.Fail(w, http.StatusConflict, "Курс недоступен для прохождения теста")
			return
		}
	} else if formRow.Status != "published" && !auth.IsAdmin(user) && (formRow.OwnerID == nil || *formRow.OwnerID != user.ID) {
		httpx.Fail(w, http.StatusForbidden, "Нет доступа")
		return
	}

	var existingID *int64
	var tmp int64
	err = h.Pool.QueryRow(ctx, `
		SELECT id FROM public.test_attempts
		WHERE form_id = $1 AND user_id = $2 AND status = 'in_progress'
		ORDER BY id DESC LIMIT 1`, formID, user.ID).Scan(&tmp)
	if err == nil {
		existingID = &tmp
	}

	if existingID == nil && !(formRow.Kind == "poll" && formRow.AllowRevote) {
		allowedAttempts := 999
		if formRow.LimitAttempts {
			allowedAttempts = int(formRow.Attempts)
			if allowedAttempts < 1 {
				allowedAttempts = 1
			}
		}
		var cnt int
		_ = h.Pool.QueryRow(ctx, `
			SELECT COUNT(*) FROM public.test_attempts
			WHERE form_id = $1 AND user_id = $2 AND status = 'completed'`, formID, user.ID).Scan(&cnt)
		if cnt >= allowedAttempts {
			httpx.Fail(w, http.StatusConflict, "Исчерпан лимит попыток")
			return
		}
	}

	tx, err := h.Pool.Begin(ctx)
	if err != nil {
		httpx.Fail(w, http.StatusInternalServerError, "Ошибка создания попытки")
		return
	}
	defer tx.Rollback(ctx)

	var attemptID int64
	if existingID != nil {
		attemptID = *existingID
	} else {
		ip := clientIP(r)
		ua := r.UserAgent()
		err = tx.QueryRow(ctx, `
			INSERT INTO public.test_attempts
			  (form_id, user_id, status, current_page, started_at, ip, user_agent, via_link)
			VALUES ($1,$2,'in_progress',0, now(),$3,$4,false) RETURNING id`,
			formID, user.ID, ip, ua).Scan(&attemptID)
		if err != nil {
			httpx.Fail(w, http.StatusInternalServerError, "Ошибка создания попытки")
			return
		}
	}

	if link != nil && enr != nil {
		_, _ = tx.Exec(ctx, `
			INSERT INTO public.course_test_attempt_links (enrollment_id, course_test_link_id, test_attempt_id)
			VALUES ($1,$2,$3) ON CONFLICT (test_attempt_id) DO NOTHING`,
			enrollmentID, tests.ToInt64Must(link["id"]), attemptID)
		if fmt.Sprint(enr["status"]) == "not_started" {
			_, _ = tx.Exec(ctx, `
				UPDATE public.course_enrollments
				SET status = 'in_progress', started_at = COALESCE(started_at, now()),
				    last_activity_at = now(), updated_at = now()
				WHERE id = $1`, enrollmentID)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		httpx.Fail(w, http.StatusInternalServerError, "Ошибка создания попытки")
		return
	}

	form, err := tests.LoadForm(ctx, h.Pool, formID, user.ID)
	if err != nil || form == nil {
		httpx.Fail(w, http.StatusNotFound, "Форма не найдена")
		return
	}
	showCorrect := preview && auth.IsAdmin(user)
	if !showCorrect {
		form = tests.StripCorrect(form)
	}

	var linkIDOut, enrIDOut any
	if link != nil {
		linkIDOut = tests.ToInt64Must(link["id"])
	}
	if enrollmentID > 0 {
		enrIDOut = enrollmentID
	}
	httpx.OK(w, map[string]any{
		"attemptId": attemptID, "form": form,
		"courseTestLinkId": linkIDOut, "enrollmentId": enrIDOut,
	}, "OK")
}

func (h *TestsHandler) AttemptSave(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
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
	body, err := h.decodeBody(r)
	if err != nil {
		httpx.Fail(w, http.StatusBadRequest, "Некорректный JSON")
		return
	}
	attemptID, _ := tests.ToInt64Public(body["attemptId"])
	if attemptID <= 0 {
		httpx.Fail(w, http.StatusBadRequest, "Не передан attemptId")
		return
	}
	answers, _ := body["answers"].(map[string]any)
	if answers == nil {
		answers = map[string]any{}
	}

	att, err := scanOneMap(r.Context(), h.Pool, `SELECT * FROM public.test_attempts WHERE id = $1`, attemptID)
	if errors.Is(err, pgx.ErrNoRows) || att == nil {
		httpx.Fail(w, http.StatusNotFound, "Попытка не найдена")
		return
	}
	if tests.ToInt64Must(att["user_id"]) != user.ID && !auth.IsAdmin(user) {
		httpx.Fail(w, http.StatusForbidden, "Нет доступа")
		return
	}
	if fmt.Sprint(att["status"]) != "in_progress" {
		httpx.Fail(w, http.StatusConflict, "Попытка уже завершена")
		return
	}

	if err := h.saveAttemptAnswers(r.Context(), attemptID, tests.ToInt64Must(att["form_id"]), answers); err != nil {
		httpx.Fail(w, http.StatusInternalServerError, "Ошибка сохранения ответов")
		return
	}
	httpx.OK(w, map[string]any{"attemptId": attemptID, "saved": len(answers)}, "OK")
}

func (h *TestsHandler) AttemptGet(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
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
	body, err := h.decodeBody(r)
	if err != nil {
		httpx.Fail(w, http.StatusBadRequest, "Некорректный JSON")
		return
	}
	ctx := r.Context()
	attemptID, _ := tests.ToInt64Public(body["attemptId"])
	formID, _ := tests.ToInt64Public(body["formId"])

	var att map[string]any
	if attemptID > 0 {
		att, err = scanOneMap(ctx, h.Pool, `SELECT * FROM public.test_attempts WHERE id = $1`, attemptID)
	} else if formID > 0 {
		att, err = scanOneMap(ctx, h.Pool, `
			SELECT * FROM public.test_attempts
			WHERE form_id = $1 AND user_id = $2 AND status = 'in_progress'
			ORDER BY id DESC LIMIT 1`, formID, user.ID)
	} else {
		httpx.Fail(w, http.StatusBadRequest, "Укажите attemptId или formId")
		return
	}
	if errors.Is(err, pgx.ErrNoRows) || att == nil {
		httpx.Fail(w, http.StatusNotFound, "Попытка не найдена")
		return
	}
	if tests.ToInt64Must(att["user_id"]) != user.ID && !auth.IsAdmin(user) {
		httpx.Fail(w, http.StatusForbidden, "Нет доступа")
		return
	}

	attemptID = tests.ToInt64Must(att["id"])
	formID = tests.ToInt64Must(att["form_id"])
	answers, _ := h.loadAttemptAnswers(ctx, attemptID)

	form, _ := tests.LoadForm(ctx, h.Pool, formID, user.ID)
	if form != nil && !auth.IsAdmin(user) {
		form = tests.StripCorrect(form)
	}

	var courseMeta any
	lr, _ := scanOneMap(ctx, h.Pool, `
		SELECT tal.enrollment_id, tal.course_test_link_id, l.type
		FROM public.course_test_attempt_links tal
		JOIN public.course_test_links l ON l.id = tal.course_test_link_id
		WHERE tal.test_attempt_id = $1 LIMIT 1`, attemptID)
	if lr != nil {
		courseMeta = map[string]any{
			"enrollmentId":     tests.ToInt64Must(lr["enrollment_id"]),
			"courseTestLinkId": tests.ToInt64Must(lr["course_test_link_id"]),
			"type":             fmt.Sprint(lr["type"]),
		}
	}

	httpx.OK(w, map[string]any{
		"attemptId": attemptID, "status": fmt.Sprint(att["status"]), "formId": formID,
		"answers": answers, "form": form, "course": courseMeta,
	}, "OK")
}

func (h *TestsHandler) AttemptFinish(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
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
	body, err := h.decodeBody(r)
	if err != nil {
		httpx.Fail(w, http.StatusBadRequest, "Некорректный JSON")
		return
	}
	ctx := r.Context()
	attemptID, _ := tests.ToInt64Public(body["attemptId"])
	if attemptID <= 0 {
		httpx.Fail(w, http.StatusBadRequest, "Не передан attemptId")
		return
	}

	att, err := scanOneMap(ctx, h.Pool, `SELECT * FROM public.test_attempts WHERE id = $1`, attemptID)
	if errors.Is(err, pgx.ErrNoRows) || att == nil {
		httpx.Fail(w, http.StatusNotFound, "Попытка не найдена")
		return
	}
	if tests.ToInt64Must(att["user_id"]) != user.ID && !auth.IsAdmin(user) {
		httpx.Fail(w, http.StatusForbidden, "Нет доступа")
		return
	}
	if fmt.Sprint(att["status"]) != "in_progress" {
		httpx.Fail(w, http.StatusConflict, "Попытка уже завершена")
		return
	}

	formID := tests.ToInt64Must(att["form_id"])
	formRow, err := loadFormRow(ctx, h.Pool, formID)
	if errors.Is(err, pgx.ErrNoRows) {
		httpx.Fail(w, http.StatusNotFound, "Форма не найдена")
		return
	}
	if err != nil {
		httpx.Fail(w, http.StatusInternalServerError, "Ошибка подключения к БД")
		return
	}

	if incoming, ok := body["answers"].(map[string]any); ok && len(incoming) > 0 {
		if err := h.saveAttemptAnswers(ctx, attemptID, formID, incoming); err != nil {
			httpx.Fail(w, http.StatusInternalServerError, "Ошибка сохранения ответов")
			return
		}
	}

	answersInt, _ := h.loadAttemptAnswersInt(ctx, attemptID)
	qRows, err := h.Pool.Query(ctx, `SELECT id, type, correct_value FROM public.test_questions WHERE form_id = $1 ORDER BY position, id`, formID)
	if err != nil {
		httpx.Fail(w, http.StatusInternalServerError, "Ошибка завершения попытки")
		return
	}
	var questions []tests.QuestionEval
	for qRows.Next() {
		var q tests.QuestionEval
		if err := qRows.Scan(&q.ID, &q.Type, &q.CorrectValue); err != nil {
			qRows.Close()
			httpx.Fail(w, http.StatusInternalServerError, "Ошибка завершения попытки")
			return
		}
		questions = append(questions, q)
	}
	qRows.Close()

	answersAny := map[string]any{}
	for k, v := range answersInt {
		answersAny[strconv.FormatInt(k, 10)] = v
	}
	eval, err := tests.EvaluateAnswers(ctx, h.Pool, formRow, questions, answersAny)
	if err != nil {
		httpx.Fail(w, http.StatusInternalServerError, "Ошибка завершения попытки")
		return
	}

	var durationSec *int
	if body["durationSec"] != nil {
		if n, err := tests.ToInt64Public(body["durationSec"]); err == nil {
			v := int(n)
			durationSec = &v
		}
	}

	tx, err := h.Pool.Begin(ctx)
	if err != nil {
		httpx.Fail(w, http.StatusInternalServerError, "Ошибка завершения попытки")
		return
	}
	defer tx.Rollback(ctx)

	for _, d := range eval.Details {
		var ic *bool
		if d.IsCorrect != nil {
			ic = d.IsCorrect
		}
		_, _ = tx.Exec(ctx, `
			UPDATE public.test_answers SET is_correct = $1
			WHERE attempt_id = $2 AND question_id = $3`, ic, attemptID, d.QuestionID)
	}

	_, err = tx.Exec(ctx, `
		UPDATE public.test_attempts
		SET status = 'completed', finished_at = now(),
		    duration_sec = COALESCE($1, duration_sec),
		    score = $2, max_score = 100, passed = $3
		WHERE id = $4`, durationSec, eval.Score, eval.Passed, attemptID)
	if err != nil {
		httpx.Fail(w, http.StatusInternalServerError, "Ошибка завершения попытки")
		return
	}

	var enrollmentID *int64
	cl, _ := scanOneMap(ctx, h.Pool, `
		SELECT enrollment_id, course_test_link_id FROM public.course_test_attempt_links WHERE test_attempt_id = $1`, attemptID)
	if cl != nil {
		eid := tests.ToInt64Must(cl["enrollment_id"])
		enrollmentID = &eid
		linkID := tests.ToInt64Must(cl["course_test_link_id"])
		link, _ := scanOneMap(ctx, h.Pool, `SELECT * FROM public.course_test_links WHERE id = $1`, linkID)
		if link != nil && fmt.Sprint(link["type"]) == "topic" && link["topic_id"] != nil {
			tid := tests.ToInt64Must(link["topic_id"])
			if h.checkTopicComplete(ctx, eid, tid) {
				_, _ = tx.Exec(ctx, `
					UPDATE public.course_topic_progress
					SET status = 'completed', completed_at = COALESCE(completed_at, now()), updated_at = now()
					WHERE enrollment_id = $1 AND topic_id = $2`, eid, tid)
			}
		}
		_, _ = tx.Exec(ctx, `
			UPDATE public.course_enrollments SET last_activity_at = now(), updated_at = now() WHERE id = $1`, eid)
	}

	if err := tx.Commit(ctx); err != nil {
		httpx.Fail(w, http.StatusInternalServerError, "Ошибка завершения попытки")
		return
	}

	var nextAction any
	if enrollmentID != nil {
		coursesSvc := &courses.Service{Pool: h.Pool}
		_, _ = coursesSvc.TryCompleteEnrollment(ctx, *enrollmentID, r)
		nextAction = h.nextActionStub(ctx, *enrollmentID)
	}

	var enrOut any
	if enrollmentID != nil {
		enrOut = *enrollmentID
	}

	httpx.OK(w, map[string]any{
		"attemptId": attemptID, "score": eval.Score, "passed": eval.Passed,
		"correctCount": eval.CorrectCount, "scorable": eval.Scorable,
		"enrollmentId": enrOut, "nextAction": nextAction,
	}, "OK")
}

func (h *TestsHandler) saveAttemptAnswers(ctx context.Context, attemptID, formID int64, answers map[string]any) error {
	qTypes := map[int64]string{}
	rows, err := h.Pool.Query(ctx, `SELECT id, type FROM public.test_questions WHERE form_id = $1`, formID)
	if err != nil {
		return err
	}
	for rows.Next() {
		var id int64
		var t string
		if err := rows.Scan(&id, &t); err != nil {
			rows.Close()
			return err
		}
		qTypes[id] = t
	}
	rows.Close()

	tx, err := h.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	for qidRaw, val := range answers {
		qid, err := strconv.ParseInt(qidRaw, 10, 64)
		if err != nil {
			continue
		}
		qtype, ok := qTypes[qid]
		if !ok {
			continue
		}
		textVal, numVal, selected, answered := parseAnswerValue(qtype, val)
		_, _ = tx.Exec(ctx, `
			DELETE FROM public.test_answer_options WHERE answer_id IN
			(SELECT id FROM public.test_answers WHERE attempt_id = $1 AND question_id = $2)`, attemptID, qid)
		_, _ = tx.Exec(ctx, `DELETE FROM public.test_answers WHERE attempt_id = $1 AND question_id = $2`, attemptID, qid)
		var ansID int64
		err = tx.QueryRow(ctx, `
			INSERT INTO public.test_answers (attempt_id, question_id, text_value, number_value, is_correct, answered)
			VALUES ($1,$2,$3,$4,NULL,$5) RETURNING id`,
			attemptID, qid, textVal, numVal, answered).Scan(&ansID)
		if err != nil {
			return err
		}
		for _, optID := range selected {
			_, _ = tx.Exec(ctx, `INSERT INTO public.test_answer_options (answer_id, option_id) VALUES ($1,$2) ON CONFLICT DO NOTHING`, ansID, optID)
		}
	}
	_, _ = tx.Exec(ctx, `UPDATE public.test_attempts SET updated_at = now() WHERE id = $1`, attemptID)
	return tx.Commit(ctx)
}

func (h *TestsHandler) loadAttemptAnswers(ctx context.Context, attemptID int64) (map[string]any, error) {
	intMap, err := h.loadAttemptAnswersInt(ctx, attemptID)
	if err != nil {
		return nil, err
	}
	out := map[string]any{}
	for k, v := range intMap {
		out[strconv.FormatInt(k, 10)] = v
	}
	return out, nil
}

func (h *TestsHandler) loadAttemptAnswersInt(ctx context.Context, attemptID int64) (map[int64]any, error) {
	rows, err := h.Pool.Query(ctx, `SELECT * FROM public.test_answers WHERE attempt_id = $1`, attemptID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[int64]any{}
	for rows.Next() {
		a, err := scanRowToMap(rows)
		if err != nil {
			continue
		}
		qid := tests.ToInt64Must(a["question_id"])
		optRows, _ := h.Pool.Query(ctx, `SELECT option_id FROM public.test_answer_options WHERE answer_id = $1`, tests.ToInt64Must(a["id"]))
		var opts []int64
		for optRows != nil && optRows.Next() {
			var oid int64
			if err := optRows.Scan(&oid); err == nil {
				opts = append(opts, oid)
			}
		}
		if optRows != nil {
			optRows.Close()
		}
		if len(opts) > 0 {
			if len(opts) == 1 {
				out[qid] = opts[0]
			} else {
				out[qid] = opts
			}
		} else if a["number_value"] != nil {
			if f, err := tests.ToFloatPublic(a["number_value"]); err == nil {
				out[qid] = f
			}
		} else {
			out[qid] = a["text_value"]
		}
	}
	return out, nil
}

func (h *TestsHandler) requireEnrollmentAccess(ctx context.Context, enrollmentID int64, user *auth.User) (map[string]any, error) {
	enr, err := scanOneMap(ctx, h.Pool, `SELECT * FROM public.course_enrollments WHERE id = $1`, enrollmentID)
	if errors.Is(err, pgx.ErrNoRows) || enr == nil {
		return nil, testsHTTPError{code: http.StatusNotFound, msg: "Запись на курс не найдена"}
	}
	if err != nil {
		return nil, err
	}
	if tests.ToInt64Must(enr["user_id"]) == user.ID {
		return enr, nil
	}
	if auth.IsAdmin(user) {
		return enr, nil
	}
	return nil, testsHTTPError{code: http.StatusForbidden, msg: "Нет доступа к этой записи"}
}

func (h *TestsHandler) checkTopicComplete(ctx context.Context, enrollmentID, topicID int64) bool {
	topic, err := scanOneMap(ctx, h.Pool, `
		SELECT t.*, p.active_seconds AS progress_active
		FROM public.course_topics t
		LEFT JOIN public.course_topic_progress p ON p.topic_id = t.id AND p.enrollment_id = $1
		WHERE t.id = $2 AND t.deleted_at IS NULL`, enrollmentID, topicID)
	if err != nil || topic == nil {
		return false
	}
	matRows, _ := h.Pool.Query(ctx, `
		SELECT m.id, m.is_required, m.minimum_active_seconds,
		       COALESCE(mp.status, 'not_started') AS mp_status,
		       COALESCE(mp.active_seconds, 0) AS mp_active
		FROM public.course_materials m
		LEFT JOIN public.course_material_progress mp ON mp.material_id = m.id AND mp.enrollment_id = $1
		WHERE m.topic_id = $2 AND m.deleted_at IS NULL`, enrollmentID, topicID)
	for matRows != nil && matRows.Next() {
		var id int64
		var required bool
		var minSec int
		var mpStatus string
		var mpActive int
		if err := matRows.Scan(&id, &required, &minSec, &mpStatus, &mpActive); err != nil {
			continue
		}
		if !required {
			continue
		}
		if mpStatus != "completed" {
			matRows.Close()
			return false
		}
		if minSec > 0 && mpActive < minSec {
			matRows.Close()
			return false
		}
	}
	if matRows != nil {
		matRows.Close()
	}
	minTopic := toIntDefaultMap(topic["minimum_active_seconds"], 0)
	progressActive := toIntDefaultMap(topic["progress_active"], 0)
	if minTopic > 0 && progressActive < minTopic {
		return false
	}
	link, _ := scanOneMap(ctx, h.Pool, `
		SELECT id, is_required FROM public.course_test_links
		WHERE topic_id = $1 AND type = 'topic' LIMIT 1`, topicID)
	if link != nil && tests.Bool(link["is_required"]) {
		if !h.testLinkPassed(ctx, enrollmentID, tests.ToInt64Must(link["id"])) {
			return false
		}
	}
	return true
}

func (h *TestsHandler) testLinkPassed(ctx context.Context, enrollmentID, linkID int64) bool {
	var passed *bool
	err := h.Pool.QueryRow(ctx, `
		SELECT a.passed FROM public.course_test_attempt_links tal
		JOIN public.test_attempts a ON a.id = tal.test_attempt_id
		WHERE tal.enrollment_id = $1 AND tal.course_test_link_id = $2 AND a.status = 'completed'
		ORDER BY a.finished_at DESC NULLS LAST, a.id DESC LIMIT 1`, enrollmentID, linkID).Scan(&passed)
	if errors.Is(err, pgx.ErrNoRows) {
		return false
	}
	if passed == nil {
		return true
	}
	return *passed
}

func (h *TestsHandler) nextActionStub(ctx context.Context, enrollmentID int64) any {
	svc := &courses.Service{Pool: h.Pool}
	return svc.NextAction(ctx, enrollmentID)
}

type testsHTTPError struct {
	code int
	msg  string
}

func (e testsHTTPError) Error() string { return e.msg }

func writeTestsErr(w http.ResponseWriter, err error) {
	var he testsHTTPError
	if errors.As(err, &he) {
		httpx.Fail(w, he.code, he.msg)
		return
	}
	httpx.Fail(w, http.StatusInternalServerError, "Ошибка подключения к БД")
}

func loadFormRow(ctx context.Context, pool *pgxpool.Pool, id int64) (tests.DBFormRow, error) {
	row := pool.QueryRow(ctx, `SELECT `+tests.FormSelectCols()+` FROM public.test_forms WHERE id = $1`, id)
	return tests.ScanFormRow(row)
}

func parseAnswerValue(qtype string, val any) (textVal *string, numVal *float64, selected []int64, answered bool) {
	switch qtype {
	case "single", "dropdown":
		if val != nil && fmt.Sprint(val) != "" {
			if id, err := tests.ToInt64Public(val); err == nil {
				selected = []int64{id}
				answered = true
			}
		}
	case "multiple":
		if arr, ok := val.([]any); ok && len(arr) > 0 {
			for _, item := range arr {
				if id, err := tests.ToInt64Public(item); err == nil {
					selected = append(selected, id)
				}
			}
			answered = len(selected) > 0
		}
	case "scale", "number":
		if val != nil && fmt.Sprint(val) != "" {
			if f, err := tests.ToFloatPublic(val); err == nil {
				numVal = &f
				s := fmt.Sprint(val)
				textVal = &s
				answered = true
			}
		}
	default:
		if val != nil && fmt.Sprint(val) != "" {
			s := fmt.Sprint(val)
			textVal = &s
			answered = true
		}
	}
	return
}

func uniqueInt64FromAny(v any) []int64 {
	arr, ok := v.([]any)
	if !ok {
		return nil
	}
	seen := map[int64]struct{}{}
	var out []int64
	for _, item := range arr {
		if id, err := tests.ToInt64Public(item); err == nil {
			if _, ok := seen[id]; ok {
				continue
			}
			seen[id] = struct{}{}
			out = append(out, id)
		}
	}
	return out
}

func clientIP(r *http.Request) *string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		ip := strings.TrimSpace(parts[0])
		if ip != "" {
			return &ip
		}
	}
	if r.RemoteAddr != "" {
		host, _, err := netSplitHostPort(r.RemoteAddr)
		if err == nil {
			return &host
		}
		ip := r.RemoteAddr
		return &ip
	}
	return nil
}

func netSplitHostPort(s string) (string, string, error) {
	if i := strings.LastIndex(s, ":"); i >= 0 {
		return s[:i], s[i+1:], nil
	}
	return s, "", fmt.Errorf("no port")
}

func regexpMatchDigits(s string) (bool, error) {
	for _, c := range s {
		if c < '0' || c > '9' {
			return false, nil
		}
	}
	return len(s) > 0, nil
}

func scanRowToMap(rows pgx.Rows) (map[string]any, error) {
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

func scanOneMap(ctx context.Context, pool *pgxpool.Pool, query string, args ...any) (map[string]any, error) {
	rows, err := pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	if !rows.Next() {
		return nil, pgx.ErrNoRows
	}
	return scanRowToMap(rows)
}

func toIntDefaultMap(v any, def int) int {
	if n, err := tests.ToInt64Public(v); err == nil {
		return int(n)
	}
	return def
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
