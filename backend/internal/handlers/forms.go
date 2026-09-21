package handlers

import (
	"context"
	"crypto/rand"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"corporate.admsr.ru/backend/internal/auth"
	"corporate.admsr.ru/backend/internal/httpx"
)

type FormsHandler struct {
	Pool *pgxpool.Pool
	Auth *auth.Service
}

var uuidRe = regexp.MustCompile(`(?i)^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)

func formsOK(w http.ResponseWriter, data any) {
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"success": true, "data": data})
}

func formsFail(w http.ResponseWriter, code int, msg string) {
	httpx.WriteJSON(w, code, map[string]any{"success": false, "error": msg})
}

func uuidV4() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

func isUUID(s string) bool {
	return uuidRe.MatchString(s)
}

func (h *FormsHandler) Forms(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id != "" && !isUUID(id) {
		formsFail(w, http.StatusBadRequest, "Некорректный id")
		return
	}

	switch r.Method {
	case http.MethodPost:
		if !h.formsRequireEditor(w, r) {
			return
		}
		h.formsCreate(w, r)
	case http.MethodGet:
		// Полная форма содержит в т.ч. правильные ответы теста — анонимам не отдаём.
		if _, ok := h.formsRequireUser(w, r); !ok {
			return
		}
		if id == "" {
			formsFail(w, http.StatusBadRequest, "Укажите ?id=UUID")
			return
		}
		h.formsGet(w, r, id)
	case http.MethodPut:
		if !h.formsRequireEditor(w, r) {
			return
		}
		if id == "" {
			formsFail(w, http.StatusBadRequest, "Укажите ?id=UUID")
			return
		}
		h.formsUpdate(w, r, id)
	default:
		formsFail(w, http.StatusMethodNotAllowed, "Метод не поддерживается")
	}
}

func (h *FormsHandler) List(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodGet {
		formsFail(w, http.StatusMethodNotAllowed, "Метод не поддерживается")
		return
	}

	if _, ok := h.formsRequireUser(w, r); !ok {
		return
	}

	ctx := r.Context()
	admin := h.isLegacyFormsAdmin(ctx, r)

	status := strings.TrimSpace(r.URL.Query().Get("status"))
	q := strings.TrimSpace(r.URL.Query().Get("q"))

	where := []string{}
	args := []any{}
	n := 1

	if admin {
		if status != "" {
			if status != "draft" && status != "published" && status != "archived" {
				formsFail(w, http.StatusBadRequest, "Некорректный status")
				return
			}
			where = append(where, fmt.Sprintf("f.status = $%d", n))
			args = append(args, status)
			n++
		}
	} else {
		where = append(where, "f.status = 'published'")
	}
	if q != "" {
		where = append(where, fmt.Sprintf("f.title ILIKE $%d", n))
		args = append(args, "%"+q+"%")
		n++
	}

	whereSQL := ""
	if len(where) > 0 {
		whereSQL = "WHERE " + strings.Join(where, " AND ")
	}

	rows, err := h.Pool.Query(ctx, fmt.Sprintf(`
		SELECT f.id, f.title, f.description, f.cover_url, f.status, f.mode, f.created_at, f.updated_at,
		       (SELECT COUNT(*)::int FROM public.questions q WHERE q.form_id = f.id) AS questions_count
		FROM public.forms f %s ORDER BY f.updated_at DESC LIMIT 500`, whereSQL), args...)
	if err != nil {
		formsFail(w, http.StatusInternalServerError, "Ошибка подключения к БД")
		return
	}
	defer rows.Close()

	var out []map[string]any
	for rows.Next() {
		var id, title, status, mode string
		var description, coverURL *string
		var createdAt, updatedAt any
		var questionsCount int
		if err := rows.Scan(&id, &title, &description, &coverURL, &status, &mode, &createdAt, &updatedAt, &questionsCount); err != nil {
			continue
		}
		desc := ""
		if description != nil {
			desc = *description
		}
		out = append(out, map[string]any{
			"id": id, "title": title, "description": desc, "coverUrl": coverURL,
			"status": status, "mode": mode, "createdAt": createdAt, "updatedAt": updatedAt,
			"questionsCount": questionsCount,
		})
	}
	if out == nil {
		out = []map[string]any{}
	}
	formsOK(w, out)
}

func (h *FormsHandler) Publish(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodPost {
		formsFail(w, http.StatusMethodNotAllowed, "Метод не поддерживается")
		return
	}
	if !h.formsRequireEditor(w, r) {
		return
	}
	id := r.URL.Query().Get("id")
	if id == "" || !isUUID(id) {
		formsFail(w, http.StatusBadRequest, "Укажите ?id=UUID")
		return
	}
	ctx := r.Context()

	var status, mode string
	err := h.Pool.QueryRow(ctx, `SELECT status, mode FROM public.forms WHERE id = $1`, id).Scan(&status, &mode)
	if errors.Is(err, pgx.ErrNoRows) {
		formsFail(w, http.StatusNotFound, "Форма не найдена")
		return
	}
	if err != nil {
		formsFail(w, http.StatusInternalServerError, "Ошибка подключения к БД")
		return
	}
	if status == "archived" {
		formsFail(w, http.StatusConflict, "Форма в архиве")
		return
	}

	qRows, err := h.Pool.Query(ctx, `SELECT id, type FROM public.questions WHERE form_id = $1 ORDER BY "order" ASC`, id)
	if err != nil {
		formsFail(w, http.StatusInternalServerError, "Ошибка подключения к БД")
		return
	}
	var questions []struct {
		id, qtype string
	}
	for qRows.Next() {
		var qid, qtype string
		if err := qRows.Scan(&qid, &qtype); err == nil {
			questions = append(questions, struct{ id, qtype string }{qid, qtype})
		}
	}
	qRows.Close()
	if len(questions) < 1 {
		formsFail(w, http.StatusBadRequest, "Перед публикацией добавьте хотя бы 1 вопрос")
		return
	}

	for _, q := range questions {
		if q.qtype != "single_choice" && q.qtype != "multiple_choice" && q.qtype != "select" {
			continue
		}
		optRows, err := h.Pool.Query(ctx, `SELECT is_correct FROM public.question_options WHERE question_id = $1`, q.id)
		if err != nil {
			formsFail(w, http.StatusInternalServerError, "Ошибка подключения к БД")
			return
		}
		var opts []bool
		for optRows.Next() {
			var ic bool
			if err := optRows.Scan(&ic); err == nil {
				opts = append(opts, ic)
			}
		}
		optRows.Close()
		if len(opts) < 2 {
			formsFail(w, http.StatusBadRequest, "Вопросы с выбором должны иметь минимум 2 варианта")
			return
		}
		if mode == "test" {
			anyCorrect := false
			for _, ic := range opts {
				if ic {
					anyCorrect = true
					break
				}
			}
			if !anyCorrect {
				formsFail(w, http.StatusBadRequest, "В тесте у вопроса должен быть отмечен хотя бы один правильный вариант")
				return
			}
		}
	}

	_, err = h.Pool.Exec(ctx, `UPDATE public.forms SET status = 'published' WHERE id = $1`, id)
	if err != nil {
		formsFail(w, http.StatusInternalServerError, "Ошибка подключения к БД")
		return
	}
	formsOK(w, nil)
}

func (h *FormsHandler) Submit(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodPost {
		formsFail(w, http.StatusMethodNotAllowed, "Метод не поддерживается")
		return
	}
	formID := r.URL.Query().Get("id")
	if formID == "" || !isUUID(formID) {
		formsFail(w, http.StatusBadRequest, "Укажите ?id=UUID")
		return
	}

	var body map[string]any
	if err := httpx.DecodeJSON(r, &body); err != nil {
		formsFail(w, http.StatusBadRequest, "Некорректный JSON")
		return
	}

	sessionID, _ := body["sessionId"].(string)
	if sessionID == "" || !isUUID(sessionID) {
		formsFail(w, http.StatusBadRequest, "sessionId обязателен (UUID)")
		return
	}
	answers, _ := body["answers"].([]any)
	if answers == nil {
		answers = []any{}
	}

	ip := clientIP(r)
	ua := r.UserAgent()
	ctx := r.Context()

	if ip != nil {
		var cnt int
		_ = h.Pool.QueryRow(ctx, `
			SELECT COUNT(*)::int FROM public.form_responses
			WHERE form_id = $1 AND ip = $2 AND created_at > (now() - interval '1 minute')`, formID, *ip).Scan(&cnt)
		if cnt >= 5 {
			formsFail(w, http.StatusTooManyRequests, "Слишком много запросов. Попробуйте позже.")
			return
		}
	}

	var status, mode string
	var settingsJSON []byte
	err := h.Pool.QueryRow(ctx, `SELECT status, mode, settings FROM public.forms WHERE id = $1`, formID).Scan(&status, &mode, &settingsJSON)
	if errors.Is(err, pgx.ErrNoRows) {
		formsFail(w, http.StatusNotFound, "Форма не найдена")
		return
	}
	if err != nil {
		formsFail(w, http.StatusInternalServerError, "Ошибка подключения к БД")
		return
	}
	if status != "published" {
		formsFail(w, http.StatusConflict, "Форма не опубликована")
		return
	}

	settings := map[string]any{}
	_ = json.Unmarshal(settingsJSON, &settings)
	showMode := "immediate"
	if v, ok := settings["showResultMode"].(string); ok && v != "" {
		showMode = v
	}
	showResult := showMode == "immediate"

	qRows, err := h.Pool.Query(ctx, `SELECT id, type, required FROM public.questions WHERE form_id = $1 ORDER BY "order" ASC`, formID)
	if err != nil {
		formsFail(w, http.StatusInternalServerError, "Ошибка подключения к БД")
		return
	}
	qByID := map[string]map[string]any{}
	var qIDs []string
	for qRows.Next() {
		var id, qtype string
		var required bool
		if err := qRows.Scan(&id, &qtype, &required); err == nil {
			qByID[id] = map[string]any{"id": id, "type": qtype, "required": required}
			qIDs = append(qIDs, id)
		}
	}
	qRows.Close()
	if len(qByID) == 0 {
		formsFail(w, http.StatusConflict, "У формы нет вопросов")
		return
	}

	optionsByQ := map[string][]map[string]any{}
	if len(qIDs) > 0 {
		optRows, err := h.Pool.Query(ctx, `
			SELECT id, question_id, is_correct FROM public.question_options WHERE question_id = ANY($1)`, qIDs)
		if err == nil {
			for optRows.Next() {
				var oid, qid string
				var ic bool
				if err := optRows.Scan(&oid, &qid, &ic); err == nil {
					optionsByQ[qid] = append(optionsByQ[qid], map[string]any{"id": oid, "is_correct": ic})
				}
			}
			optRows.Close()
		}
	}

	answerMap := map[string]map[string]any{}
	for _, a := range answers {
		am, ok := a.(map[string]any)
		if !ok {
			continue
		}
		qid, _ := am["questionId"].(string)
		if qid == "" || !isUUID(qid) || qByID[qid] == nil {
			continue
		}
		value, ok := am["value"].(map[string]any)
		if !ok || value["type"] == nil {
			continue
		}
		answerMap[qid] = value
	}

	for _, q := range qByID {
		if !q["required"].(bool) {
			continue
		}
		qid := q["id"].(string)
		v := answerMap[qid]
		if v == nil {
			formsFail(w, http.StatusBadRequest, "Заполните все обязательные вопросы")
			return
		}
		t := fmt.Sprint(v["type"])
		switch {
		case (t == "single" || t == "select") && fmt.Sprint(v["optionId"]) == "":
			formsFail(w, http.StatusBadRequest, "Заполните все обязательные вопросы")
			return
		case t == "multiple":
			ids, _ := v["optionIds"].([]any)
			if len(ids) < 1 {
				formsFail(w, http.StatusBadRequest, "Заполните все обязательные вопросы")
				return
			}
		case (t == "short_text" || t == "long_text") && strings.TrimSpace(fmt.Sprint(v["text"])) == "":
			formsFail(w, http.StatusBadRequest, "Заполните все обязательные вопросы")
			return
		case t == "rating_1_10":
			if n, _ := toInt64Forms(v["value"]); n < 1 {
				formsFail(w, http.StatusBadRequest, "Заполните все обязательные вопросы")
				return
			}
		case t == "file" && fmt.Sprint(v["base64"]) == "":
			formsFail(w, http.StatusBadRequest, "Заполните все обязательные вопросы")
			return
		}
	}

	var score, maxScore, correctPercent *float64
	var perQuestion []map[string]any
	scorableTotal := 0
	scorableCorrect := 0

	if mode == "test" {
		for _, q := range qByID {
			qid := q["id"].(string)
			qt := q["type"].(string)
			v := answerMap[qid]
			isScorable := qt == "single_choice" || qt == "multiple_choice" || qt == "select"
			if !isScorable {
				perQuestion = append(perQuestion, map[string]any{"questionId": qid, "isCorrect": nil, "earned": nil, "possible": nil})
				continue
			}
			scorableTotal++

			var correctIDs []string
			for _, o := range optionsByQ[qid] {
				if o["is_correct"].(bool) {
					correctIDs = append(correctIDs, o["id"].(string))
				}
			}
			ok := false
			if v != nil {
				if (qt == "single_choice" || qt == "select") && (v["type"] == "single" || v["type"] == "select") {
					ok = stringInSlice(fmt.Sprint(v["optionId"]), correctIDs)
				} else if qt == "multiple_choice" && v["type"] == "multiple" {
					given := setifyForms(v["optionIds"])
					corr := setifyForms(correctIDs)
					ok = stringSlicesEqual(given, corr)
				}
			}
			if ok {
				scorableCorrect++
			}
			earned := 0
			if ok {
				earned = 1
			}
			perQuestion = append(perQuestion, map[string]any{
				"questionId": qid, "isCorrect": ok, "earned": earned, "possible": 1,
			})
		}
		s := float64(scorableCorrect)
		m := float64(scorableTotal)
		score = &s
		maxScore = &m
		if scorableTotal > 0 {
			cp := float64(scorableCorrect) / float64(scorableTotal) * 100
			cp = float64(int(cp*100+0.5)) / 100
			correctPercent = &cp
		}
	}

	responseID := uuidV4()
	meta := map[string]any{"respondent": map[string]any{}}
	if resp, ok := body["respondent"].(map[string]any); ok {
		meta["respondent"] = resp
	}
	metaBytes, _ := json.Marshal(meta)

	tx, err := h.Pool.Begin(ctx)
	if err != nil {
		formsFail(w, http.StatusInternalServerError, "Ошибка сохранения результата")
		return
	}
	defer tx.Rollback(ctx)

	var ipVal, uaVal any
	if ip != nil {
		ipVal = *ip
	}
	if ua != "" {
		uaVal = ua
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO public.form_responses
		  (id, form_id, session_id, ip, user_agent, status, completed_at, score, max_score, correct_percent, meta)
		VALUES ($1,$2,$3,$4,$5,'completed', now(), $6,$7,$8,$9::jsonb)`,
		responseID, formID, sessionID, ipVal, uaVal, score, maxScore, correctPercent, string(metaBytes))
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			formsFail(w, http.StatusConflict, "Эта сессия уже отправляла ответы по этой форме")
			return
		}
		formsFail(w, http.StatusInternalServerError, "Ошибка сохранения результата")
		return
	}

	for qid, v := range answerMap {
		qt := qByID[qid]["type"].(string)
		var isCorrect any
		for _, pq := range perQuestion {
			if pq["questionId"] == qid {
				isCorrect = pq["isCorrect"]
				break
			}
		}
		answerJSON, _ := json.Marshal(map[string]any{"value": v, "isCorrect": isCorrect})

		var textValue, optionIDs any
		var numberValue *int
		t := fmt.Sprint(v["type"])
		switch t {
		case "short_text", "long_text":
			textValue = fmt.Sprint(v["text"])
		case "rating_1_10":
			if n, err := toInt64Forms(v["value"]); err == nil {
				numberValue = new(int)
				*numberValue = int(n)
			}
		case "single", "select":
			optionIDs = "{" + fmt.Sprint(v["optionId"]) + "}"
		case "multiple":
			ids, _ := v["optionIds"].([]any)
			parts := make([]string, 0, len(ids))
			for _, id := range ids {
				parts = append(parts, fmt.Sprint(id))
			}
			optionIDs = "{" + strings.Join(parts, ",") + "}"
		case "file":
			textValue = fmt.Sprint(v["fileName"])
		}

		_, err = tx.Exec(ctx, `
			INSERT INTO public.response_answers
			  (id, response_id, question_id, answer_json, text_value, number_value, option_ids)
			VALUES ($1,$2,$3,$4::jsonb,$5,$6,$7)
			ON CONFLICT (response_id, question_id)
			DO UPDATE SET answer_json = EXCLUDED.answer_json, text_value = EXCLUDED.text_value,
			              number_value = EXCLUDED.number_value, option_ids = EXCLUDED.option_ids`,
			uuidV4(), responseID, qid, string(answerJSON), textValue, numberValue, optionIDs)
		if err != nil {
			formsFail(w, http.StatusInternalServerError, "Ошибка сохранения результата")
			return
		}
		_ = qt
	}

	if err := tx.Commit(ctx); err != nil {
		formsFail(w, http.StatusInternalServerError, "Ошибка сохранения результата")
		return
	}

	formsOK(w, map[string]any{
		"responseId": responseID, "score": score, "maxScore": maxScore,
		"correctPercent": correctPercent, "showResult": showResult, "perQuestion": perQuestion,
	})
}

func (h *FormsHandler) Report(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodGet {
		formsFail(w, http.StatusMethodNotAllowed, "Метод не поддерживается")
		return
	}
	// Отчёт содержит все ответы респондентов и их ФИО — только для редакторов раздела.
	if !h.formsRequireEditor(w, r) {
		return
	}
	formID := r.URL.Query().Get("id")
	if formID == "" || !isUUID(formID) {
		formsFail(w, http.StatusBadRequest, "Укажите ?id=UUID")
		return
	}

	from := strings.TrimSpace(r.URL.Query().Get("from"))
	to := strings.TrimSpace(r.URL.Query().Get("to"))
	status := strings.TrimSpace(r.URL.Query().Get("status"))
	format := strings.TrimSpace(r.URL.Query().Get("format"))

	if status != "" && status != "completed" && status != "incomplete" {
		formsFail(w, http.StatusBadRequest, "status должен быть completed|incomplete")
		return
	}

	ctx := r.Context()
	form, err := h.loadLegacyFormMeta(ctx, formID)
	if errors.Is(err, pgx.ErrNoRows) {
		formsFail(w, http.StatusNotFound, "Форма не найдена")
		return
	}
	if err != nil {
		formsFail(w, http.StatusInternalServerError, "Ошибка подключения к БД")
		return
	}

	where := "form_id = $1"
	args := []any{formID}
	n := 2
	if status != "" {
		where += fmt.Sprintf(" AND status = $%d", n)
		args = append(args, status)
		n++
	}
	if from != "" {
		where += fmt.Sprintf(" AND created_at >= $%d::date", n)
		args = append(args, from)
		n++
	}
	if to != "" {
		where += fmt.Sprintf(" AND created_at < ($%d::date + interval '1 day')", n)
		args = append(args, to)
		n++
	}

	pRows, err := h.Pool.Query(ctx, fmt.Sprintf(`
		SELECT id, session_id, started_at, completed_at, status, score, max_score, meta
		FROM public.form_responses WHERE %s ORDER BY created_at DESC LIMIT 5000`, where), args...)
	if err != nil {
		formsFail(w, http.StatusInternalServerError, "Ошибка подключения к БД")
		return
	}
	defer pRows.Close()

	var participants []map[string]any
	for pRows.Next() {
		var id, sessionID, status string
		var startedAt, completedAt, score, maxScore any
		var metaJSON []byte
		if err := pRows.Scan(&id, &sessionID, &startedAt, &completedAt, &status, &score, &maxScore, &metaJSON); err != nil {
			continue
		}
		meta := map[string]any{}
		_ = json.Unmarshal(metaJSON, &meta)
		resp, _ := meta["respondent"].(map[string]any)
		var userID, fio any
		if resp != nil {
			if v, ok := resp["userId"]; ok {
				userID = fmt.Sprint(v)
			}
			if v, ok := resp["fio"]; ok {
				fio = fmt.Sprint(v)
			}
		}
		var scoreF, maxF any
		if score != nil {
			if f, err := toFloatForms(score); err == nil {
				scoreF = f
			}
		}
		if maxScore != nil {
			if f, err := toFloatForms(maxScore); err == nil {
				maxF = f
			}
		}
		participants = append(participants, map[string]any{
			"responseId": id, "sessionId": sessionID, "userId": userID, "fio": fio,
			"startedAt": startedAt, "completedAt": completedAt, "status": status,
			"score": scoreF, "maxScore": maxF,
		})
	}

	var total, completed int
	var avgScore, avgCorrect, medianScore, stdDevScore *float64
	_ = h.Pool.QueryRow(ctx, fmt.Sprintf(`
		SELECT COUNT(*)::int, COUNT(*) FILTER (WHERE status = 'completed')::int,
		       AVG(score) FILTER (WHERE status='completed' AND score IS NOT NULL),
		       AVG(correct_percent) FILTER (WHERE status='completed' AND correct_percent IS NOT NULL),
		       percentile_cont(0.5) WITHIN GROUP (ORDER BY score) FILTER (WHERE status='completed' AND score IS NOT NULL),
		       stddev_pop(score) FILTER (WHERE status='completed' AND score IS NOT NULL)
		FROM public.form_responses WHERE %s`, where), args...).Scan(
		&total, &completed, &avgScore, &avgCorrect, &medianScore, &stdDevScore)

	summary := map[string]any{
		"totalResponses": total, "completedResponses": completed,
		"avgScore": roundPtr(avgScore, 2), "medianScore": roundPtr(medianScore, 2),
		"stdDevScore": roundPtr(stdDevScore, 2), "correctPercentAvg": roundPtr(avgCorrect, 2),
	}

	qRows, err := h.Pool.Query(ctx, `SELECT id, type, "order", title FROM public.questions WHERE form_id = $1 ORDER BY "order" ASC`, formID)
	if err != nil {
		formsFail(w, http.StatusInternalServerError, "Ошибка подключения к БД")
		return
	}
	defer qRows.Close()

	type qRow struct{ id, qtype, title string }
	var qs []qRow
	var qIDs []string
	for qRows.Next() {
		var id, qtype, title string
		var order int
		if err := qRows.Scan(&id, &qtype, &order, &title); err == nil {
			qs = append(qs, qRow{id, qtype, title})
			qIDs = append(qIDs, id)
		}
	}

	optLabels := map[string]map[string]string{}
	if len(qIDs) > 0 {
		oRows, _ := h.Pool.Query(ctx, `SELECT id, question_id, label, "order" FROM public.question_options WHERE question_id = ANY($1) ORDER BY question_id, "order"`, qIDs)
		for oRows != nil && oRows.Next() {
			var oid, qid, label string
			var order int
			if err := oRows.Scan(&oid, &qid, &label, &order); err == nil {
				if optLabels[qid] == nil {
					optLabels[qid] = map[string]string{}
				}
				optLabels[qid][oid] = label
			}
		}
		if oRows != nil {
			oRows.Close()
		}
	}

	var funnel []map[string]any
	for _, q := range qs {
		var reached int
		_ = h.Pool.QueryRow(ctx, fmt.Sprintf(`
			SELECT COUNT(DISTINCT fr.id)::int
			FROM public.form_responses fr
			JOIN public.response_answers ra ON ra.response_id = fr.id AND ra.question_id = $%d
			WHERE %s`, len(args)+1, where),
			append(append([]any{}, args...), q.id)...).Scan(&reached)
		funnel = append(funnel, map[string]any{"questionId": q.id, "reached": reached})
	}

	var questionsOut []map[string]any
	for _, q := range qs {
		var dist []map[string]any
		switch {
		case q.qtype == "single_choice" || q.qtype == "select" || q.qtype == "multiple_choice":
			rows, _ := h.Pool.Query(ctx, fmt.Sprintf(`
				SELECT ra.option_ids FROM public.response_answers ra
				JOIN public.form_responses fr ON fr.id = ra.response_id
				WHERE ra.question_id = $%d AND %s`, len(args)+1, where),
				append(append([]any{}, args...), q.id)...)
			counts := map[string]int{}
			for rows != nil && rows.Next() {
				var raw *string
				if err := rows.Scan(&raw); err != nil {
					continue
				}
				if raw == nil || strings.Trim(*raw, "{}") == "" {
					counts["__no_answer__"]++
					continue
				}
				trim := strings.Trim(*raw, "{}")
				for _, oid := range strings.Split(trim, ",") {
					oid = strings.Trim(strings.TrimSpace(oid), `"`)
					if oid == "" {
						counts["__no_answer__"]++
					} else {
						counts[oid]++
					}
				}
			}
			if rows != nil {
				rows.Close()
			}
			for oid, label := range optLabels[q.id] {
				dist = append(dist, map[string]any{"key": oid, "label": label, "count": counts[oid]})
			}
			dist = append(dist, map[string]any{"key": "__no_answer__", "label": "Нет ответа", "count": counts["__no_answer__"]})
		case q.qtype == "rating_1_10":
			rows, _ := h.Pool.Query(ctx, fmt.Sprintf(`
				SELECT ra.number_value FROM public.response_answers ra
				JOIN public.form_responses fr ON fr.id = ra.response_id
				WHERE ra.question_id = $%d AND %s`, len(args)+1, where),
				append(append([]any{}, args...), q.id)...)
			counts := make([]int, 11)
			empty := 0
			for rows != nil && rows.Next() {
				var v *int
				if err := rows.Scan(&v); err != nil || v == nil {
					empty++
					continue
				}
				if *v >= 1 && *v <= 10 {
					counts[*v]++
				}
			}
			if rows != nil {
				rows.Close()
			}
			for i := 1; i <= 10; i++ {
				dist = append(dist, map[string]any{"key": fmt.Sprint(i), "label": fmt.Sprint(i), "count": counts[i]})
			}
			dist = append(dist, map[string]any{"key": "__no_answer__", "label": "Нет ответа", "count": empty})
		default:
			rows, _ := h.Pool.Query(ctx, fmt.Sprintf(`
				SELECT ra.text_value FROM public.response_answers ra
				JOIN public.form_responses fr ON fr.id = ra.response_id
				WHERE ra.question_id = $%d AND %s`, len(args)+1, where),
				append(append([]any{}, args...), q.id)...)
			answered, empty := 0, 0
			for rows != nil && rows.Next() {
				var t *string
				if err := rows.Scan(&t); err != nil || t == nil || strings.TrimSpace(*t) == "" {
					empty++
				} else {
					answered++
				}
			}
			if rows != nil {
				rows.Close()
			}
			dist = []map[string]any{
				{"key": "answered", "label": "Ответили", "count": answered},
				{"key": "empty", "label": "Пусто", "count": empty},
			}
		}
		questionsOut = append(questionsOut, map[string]any{
			"questionId": q.id, "type": q.qtype, "title": q.title, "distribution": dist,
		})
	}

	var topMistakes []map[string]any
	if form["mode"] == "test" {
		tmRows, _ := h.Pool.Query(ctx, fmt.Sprintf(`
			SELECT ra.question_id,
			       AVG(CASE WHEN (ra.answer_json->>'isCorrect') = 'true' THEN 1 ELSE 0 END) AS avg_correct
			FROM public.response_answers ra
			JOIN public.form_responses fr ON fr.id = ra.response_id
			JOIN public.questions q ON q.id = ra.question_id
			WHERE q.form_id = $%d AND q.type IN ('single_choice','multiple_choice','select') AND %s
			GROUP BY ra.question_id ORDER BY avg_correct ASC NULLS LAST LIMIT 3`, len(args)+1, where),
			append(append([]any{}, args...), formID)...)
		for tmRows != nil && tmRows.Next() {
			var qid string
			var avg *float64
			if err := tmRows.Scan(&qid, &avg); err == nil {
				wrong := 0.0
				if avg != nil {
					wrong = (1.0 - *avg) * 100.0
				}
				topMistakes = append(topMistakes, map[string]any{
					"questionId": qid, "wrongPercent": mathRound(wrong, 2),
				})
			}
		}
		if tmRows != nil {
			tmRows.Close()
		}
	}

	if format == "csv" {
		w.Header().Set("Content-Type", "text/csv; charset=utf-8")
		w.Header().Set("Content-Disposition", `attachment; filename="report.csv"`)
		cw := csv.NewWriter(w)
		_ = cw.Write([]string{"response_id", "session_id", "fio", "user_id", "status", "started_at", "completed_at", "score", "max_score"})
		for _, p := range participants {
			_ = cw.Write([]string{
				fmt.Sprint(p["responseId"]), fmt.Sprint(p["sessionId"]), fmt.Sprint(p["fio"]),
				fmt.Sprint(p["userId"]), fmt.Sprint(p["status"]), fmt.Sprint(p["startedAt"]),
				fmt.Sprint(p["completedAt"]), fmt.Sprint(p["score"]), fmt.Sprint(p["maxScore"]),
			})
		}
		cw.Flush()
		return
	}

	formsOK(w, map[string]any{
		"form": form, "summary": summary, "funnel": funnel,
		"topMistakes": topMistakes, "questions": questionsOut, "participants": participants,
	})
}

func (h *FormsHandler) Archive(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodPost {
		formsFail(w, http.StatusMethodNotAllowed, "Метод не поддерживается")
		return
	}
	id := r.URL.Query().Get("id")
	if id == "" || !isUUID(id) {
		formsFail(w, http.StatusBadRequest, "Укажите ?id=UUID")
		return
	}
	var body map[string]any
	if err := httpx.DecodeJSON(r, &body); err != nil {
		formsFail(w, http.StatusBadRequest, "Некорректный JSON")
		return
	}
	next := fmt.Sprint(body["status"])
	if next != "archived" && next != "draft" {
		formsFail(w, http.StatusBadRequest, "status должен быть archived|draft")
		return
	}
	if !h.isLegacyFormsAdmin(r.Context(), r) {
		formsFail(w, http.StatusForbidden, "Недостаточно прав")
		return
	}
	_, err := h.Pool.Exec(r.Context(), `UPDATE public.forms SET status = $1 WHERE id = $2`, next, id)
	if err != nil {
		formsFail(w, http.StatusInternalServerError, "Ошибка подключения к БД")
		return
	}
	formsOK(w, nil)
}

func (h *FormsHandler) Delete(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodDelete {
		formsFail(w, http.StatusMethodNotAllowed, "Метод не поддерживается")
		return
	}
	id := r.URL.Query().Get("id")
	if id == "" || !isUUID(id) {
		formsFail(w, http.StatusBadRequest, "Укажите ?id=UUID")
		return
	}
	if !h.isLegacyFormsAdmin(r.Context(), r) {
		formsFail(w, http.StatusForbidden, "Недостаточно прав")
		return
	}
	_, err := h.Pool.Exec(r.Context(), `DELETE FROM public.forms WHERE id = $1`, id)
	if err != nil {
		formsFail(w, http.StatusInternalServerError, "Ошибка подключения к БД")
		return
	}
	formsOK(w, nil)
}

func (h *FormsHandler) formsCreate(w http.ResponseWriter, r *http.Request) {
	var body map[string]any
	if err := httpx.DecodeJSON(r, &body); err != nil {
		formsFail(w, http.StatusBadRequest, "Некорректный JSON")
		return
	}
	p, err := validateLegacyFormPayload(body)
	if err != nil {
		formsFail(w, http.StatusBadRequest, err.Error())
		return
	}
	formID := uuidV4()
	settings, _ := json.Marshal(p["settings"])
	_, err = h.Pool.Exec(r.Context(), `
		INSERT INTO public.forms (id, title, description, cover_url, status, mode, settings)
		VALUES ($1,$2,$3,$4,'draft',$5,$6::jsonb)`,
		formID, p["title"], p["description"], p["cover_url"], p["mode"], string(settings))
	if err != nil {
		formsFail(w, http.StatusInternalServerError, "Ошибка подключения к БД")
		return
	}
	formsOK(w, map[string]any{"id": formID})
}

func (h *FormsHandler) formsGet(w http.ResponseWriter, r *http.Request, id string) {
	form, err := h.loadLegacyFormFull(r.Context(), id)
	if errors.Is(err, pgx.ErrNoRows) {
		formsFail(w, http.StatusNotFound, "Форма не найдена")
		return
	}
	if err != nil {
		formsFail(w, http.StatusInternalServerError, "Ошибка подключения к БД")
		return
	}
	formsOK(w, form)
}

func (h *FormsHandler) formsUpdate(w http.ResponseWriter, r *http.Request, id string) {
	var body map[string]any
	if err := httpx.DecodeJSON(r, &body); err != nil {
		formsFail(w, http.StatusBadRequest, "Некорректный JSON")
		return
	}
	p, err := validateLegacyFormPayload(body)
	if err != nil {
		formsFail(w, http.StatusBadRequest, err.Error())
		return
	}

	ctx := r.Context()
	var existingStatus string
	err = h.Pool.QueryRow(ctx, `SELECT status FROM public.forms WHERE id = $1`, id).Scan(&existingStatus)
	if errors.Is(err, pgx.ErrNoRows) {
		formsFail(w, http.StatusNotFound, "Форма не найдена")
		return
	}
	if err != nil {
		formsFail(w, http.StatusInternalServerError, "Ошибка подключения к БД")
		return
	}
	_ = existingStatus

	settings, _ := json.Marshal(p["settings"])
	tx, err := h.Pool.Begin(ctx)
	if err != nil {
		formsFail(w, http.StatusInternalServerError, "Ошибка сохранения формы")
		return
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `
		UPDATE public.forms SET title=$2, description=$3, cover_url=$4, mode=$5, settings=$6::jsonb WHERE id=$1`,
		id, p["title"], p["description"], p["cover_url"], p["mode"], string(settings))
	if err != nil {
		formsFail(w, http.StatusInternalServerError, "Ошибка сохранения формы")
		return
	}

	_, _ = tx.Exec(ctx, `DELETE FROM public.questions WHERE form_id = $1`, id)
	questions := p["questions"].([]map[string]any)
	for _, q := range questions {
		qid, _ := q["id"].(string)
		if qid == "" || !isUUID(qid) {
			qid = uuidV4()
		}
		_, err = tx.Exec(ctx, `
			INSERT INTO public.questions (id, form_id, type, "order", title, hint, required, config)
			VALUES ($1,$2,$3,$4,$5,$6,$7,'{}'::jsonb)`,
			qid, id, q["type"], q["order"], q["title"], q["hint"], q["required"])
		if err != nil {
			formsFail(w, http.StatusInternalServerError, "Ошибка сохранения формы")
			return
		}
		opts, _ := q["options"].([]map[string]any)
		for _, opt := range opts {
			oid, _ := opt["id"].(string)
			if oid == "" || !isUUID(oid) {
				oid = uuidV4()
			}
			_, err = tx.Exec(ctx, `
				INSERT INTO public.question_options (id, question_id, label, "order", is_correct)
				VALUES ($1,$2,$3,$4,$5)`,
				oid, qid, opt["label"], opt["order"], opt["isCorrect"])
			if err != nil {
				formsFail(w, http.StatusInternalServerError, "Ошибка сохранения формы")
				return
			}
		}
	}

	if err := tx.Commit(ctx); err != nil {
		formsFail(w, http.StatusInternalServerError, "Ошибка сохранения формы")
		return
	}
	formsOK(w, nil)
}

// isLegacyFormsAdmin определяет право редактировать формы.
//
// SEC-007: раньше личность бралась из заголовка X-User-Id, который присылает сам
// клиент, — любой мог объявить себя администратором форм. Теперь источник личности
// только один: серверная сессия. Признак редактора согласован с остальным
// порталом — user_group='admin' либо права на раздел tests через группы
// (раньше проверялась колонка role, которая хранит должность, а не роль доступа).
func (h *FormsHandler) isLegacyFormsAdmin(ctx context.Context, r *http.Request) bool {
	if h.Auth == nil {
		return false
	}
	u, err := h.Auth.CurrentUser(ctx, r)
	if err != nil || u == nil {
		return false
	}
	return auth.CanEditSection(ctx, h.Pool, u, "tests")
}

// formsRequireUser — доступ к модулю форм только для авторизованных (SEC-001).
func (h *FormsHandler) formsRequireUser(w http.ResponseWriter, r *http.Request) (*auth.User, bool) {
	if h.Auth == nil {
		formsFail(w, http.StatusUnauthorized, "Требуется авторизация")
		return nil, false
	}
	u, err := h.Auth.CurrentUser(r.Context(), r)
	if err != nil || u == nil {
		formsFail(w, http.StatusUnauthorized, "Требуется авторизация")
		return nil, false
	}
	return u, true
}

// formsRequireEditor — операции над самими формами (создание, правка, публикация,
// архивация, удаление, отчёты) доступны только редакторам раздела.
func (h *FormsHandler) formsRequireEditor(w http.ResponseWriter, r *http.Request) bool {
	u, ok := h.formsRequireUser(w, r)
	if !ok {
		return false
	}
	if !auth.CanEditSection(r.Context(), h.Pool, u, "tests") {
		formsFail(w, http.StatusForbidden, "Недостаточно прав для этого раздела")
		return false
	}
	return true
}

func (h *FormsHandler) loadLegacyFormMeta(ctx context.Context, id string) (map[string]any, error) {
	var title, status, mode string
	var description, coverURL *string
	var settingsJSON []byte
	var createdAt, updatedAt any
	err := h.Pool.QueryRow(ctx, `
		SELECT title, description, cover_url, status, mode, settings, created_at, updated_at
		FROM public.forms WHERE id = $1`, id).Scan(&title, &description, &coverURL, &status, &mode, &settingsJSON, &createdAt, &updatedAt)
	if err != nil {
		return nil, err
	}
	settings := map[string]any{}
	_ = json.Unmarshal(settingsJSON, &settings)
	desc := ""
	if description != nil {
		desc = *description
	}
	return map[string]any{
		"id": id, "title": title, "description": desc, "coverUrl": coverURL,
		"status": status, "mode": mode, "settings": settings,
		"createdAt": createdAt, "updatedAt": updatedAt,
	}, nil
}

func (h *FormsHandler) loadLegacyFormFull(ctx context.Context, id string) (map[string]any, error) {
	form, err := h.loadLegacyFormMeta(ctx, id)
	if err != nil {
		return nil, err
	}
	qRows, err := h.Pool.Query(ctx, `
		SELECT id, form_id, type, "order", title, hint, required FROM public.questions
		WHERE form_id = $1 ORDER BY "order" ASC`, id)
	if err != nil {
		return nil, err
	}
	defer qRows.Close()

	var qIDs []string
	type qItem struct {
		id, formID, qtype, title, hint string
		order                          int
		required                       bool
	}
	var items []qItem
	for qRows.Next() {
		var q qItem
		if err := qRows.Scan(&q.id, &q.formID, &q.qtype, &q.order, &q.title, &q.hint, &q.required); err == nil {
			items = append(items, q)
			qIDs = append(qIDs, q.id)
		}
	}

	optionsByQ := map[string][]map[string]any{}
	if len(qIDs) > 0 {
		oRows, err := h.Pool.Query(ctx, `
			SELECT id, question_id, label, "order", is_correct FROM public.question_options
			WHERE question_id = ANY($1) ORDER BY question_id ASC, "order" ASC`, qIDs)
		if err == nil {
			for oRows.Next() {
				var oid, qid, label string
				var order int
				var ic bool
				if err := oRows.Scan(&oid, &qid, &label, &order, &ic); err == nil {
					optionsByQ[qid] = append(optionsByQ[qid], map[string]any{
						"id": oid, "label": label, "order": order, "isCorrect": ic,
					})
				}
			}
			oRows.Close()
		}
	}

	var questions []map[string]any
	for _, q := range items {
		opts := optionsByQ[q.id]
		if opts == nil {
			opts = []map[string]any{}
		}
		questions = append(questions, map[string]any{
			"id": q.id, "formId": q.formID, "type": q.qtype, "order": q.order,
			"title": q.title, "hint": q.hint, "required": q.required, "options": opts,
		})
	}
	if questions == nil {
		questions = []map[string]any{}
	}
	form["questions"] = questions
	return form, nil
}

func validateLegacyFormPayload(d map[string]any) (map[string]any, error) {
	title := strings.TrimSpace(fmt.Sprint(d["title"]))
	if title == "" || title == "<nil>" {
		return nil, fmt.Errorf("title обязателен")
	}
	if utf8.RuneCountInString(title) > 200 {
		return nil, fmt.Errorf("title слишком длинный")
	}
	mode := fmt.Sprint(d["mode"])
	if mode == "" || mode == "<nil>" {
		mode = "survey"
	}
	if mode != "survey" && mode != "test" {
		return nil, fmt.Errorf("mode должен быть survey|test")
	}
	status := fmt.Sprint(d["status"])
	if status == "" || status == "<nil>" {
		status = "draft"
	}
	if status != "draft" && status != "published" && status != "archived" {
		return nil, fmt.Errorf("Некорректный status")
	}
	description := fmt.Sprint(d["description"])
	if description == "<nil>" {
		description = ""
	}
	if utf8.RuneCountInString(description) > 2000 {
		return nil, fmt.Errorf("description слишком длинный")
	}

	var coverURL any
	if v, ok := d["coverUrl"]; ok && v != nil {
		s := strings.TrimSpace(fmt.Sprint(v))
		if s != "" && s != "<nil>" {
			if len(s) < 5 {
				return nil, fmt.Errorf("coverUrl должен быть URL или null")
			}
			coverURL = s
		}
	}

	settings, ok := d["settings"].(map[string]any)
	if !ok {
		settings = map[string]any{}
	}
	rawQuestions, ok := d["questions"].([]any)
	if !ok {
		rawQuestions = []any{}
	}

	allowedTypes := map[string]struct{}{
		"single_choice": {}, "multiple_choice": {}, "short_text": {}, "long_text": {},
		"rating_1_10": {}, "select": {}, "file": {},
	}

	var normQuestions []map[string]any
	for idx, raw := range rawQuestions {
		q, ok := raw.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("Некорректный question")
		}
		qid, _ := q["id"].(string)
		if qid != "" && !isUUID(qid) {
			return nil, fmt.Errorf("question.id должен быть UUID")
		}
		qtype := fmt.Sprint(q["type"])
		if _, ok := allowedTypes[qtype]; !ok {
			return nil, fmt.Errorf("Некорректный type у вопроса #%d", idx+1)
		}
		qTitle := strings.TrimSpace(fmt.Sprint(q["title"]))
		if qTitle == "" || qTitle == "<nil>" {
			return nil, fmt.Errorf("Пустой title у вопроса #%d", idx+1)
		}
		hint := fmt.Sprint(q["hint"])
		if hint == "<nil>" {
			hint = ""
		}
		required := false
		if v, ok := q["required"].(bool); ok {
			required = v
		}
		order := idx
		if v, err := toInt64Forms(q["order"]); err == nil {
			order = int(v)
		}

		var normOptions []map[string]any
		needsOptions := qtype == "single_choice" || qtype == "multiple_choice" || qtype == "select"
		if needsOptions {
			opts, ok := q["options"].([]any)
			if !ok {
				return nil, fmt.Errorf("options обязателен у вопроса #%d", idx+1)
			}
			if len(opts) < 2 {
				return nil, fmt.Errorf("Минимум 2 варианта у вопроса #%d", idx+1)
			}
			anyCorrect := false
			for oIdx, oraw := range opts {
				o, ok := oraw.(map[string]any)
				if !ok {
					return nil, fmt.Errorf("Некорректный option")
				}
				oid, _ := o["id"].(string)
				if oid != "" && !isUUID(oid) {
					return nil, fmt.Errorf("option.id должен быть UUID")
				}
				label := strings.TrimSpace(fmt.Sprint(o["label"]))
				if label == "" || label == "<nil>" {
					return nil, fmt.Errorf("Пустой label у варианта #%d", oIdx+1)
				}
				isCorrect := false
				if v, ok := o["isCorrect"].(bool); ok {
					isCorrect = v
				}
				if isCorrect {
					anyCorrect = true
				}
				oOrder := oIdx
				if v, err := toInt64Forms(o["order"]); err == nil {
					oOrder = int(v)
				}
				normOptions = append(normOptions, map[string]any{
					"id": oid, "label": label, "order": oOrder, "isCorrect": isCorrect,
				})
			}
			if mode == "test" && !anyCorrect {
				return nil, fmt.Errorf("В тесте у вопроса #%d должен быть отмечен хотя бы один правильный вариант", idx+1)
			}
		}

		normQuestions = append(normQuestions, map[string]any{
			"id": qid, "type": qtype, "order": order, "title": qTitle,
			"hint": hint, "required": required, "options": normOptions,
		})
	}

	return map[string]any{
		"title": title, "description": description, "cover_url": coverURL,
		"mode": mode, "status": status, "settings": settings, "questions": normQuestions,
	}, nil
}

func setifyForms(v any) []string {
	arr, ok := v.([]any)
	if !ok {
		if sarr, ok := v.([]string); ok {
			arr = make([]any, len(sarr))
			for i, s := range sarr {
				arr[i] = s
			}
		}
	}
	m := map[string]struct{}{}
	for _, item := range arr {
		m[fmt.Sprint(item)] = struct{}{}
	}
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func stringInSlice(s string, list []string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

func stringSlicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func toInt64Forms(v any) (int64, error) {
	switch x := v.(type) {
	case float64:
		return int64(x), nil
	case int:
		return int64(x), nil
	case int64:
		return x, nil
	case string:
		return strconv.ParseInt(x, 10, 64)
	default:
		return strconv.ParseInt(fmt.Sprint(v), 10, 64)
	}
}

func toFloatForms(v any) (float64, error) {
	switch x := v.(type) {
	case float64:
		return x, nil
	case int:
		return float64(x), nil
	case int64:
		return float64(x), nil
	default:
		return strconv.ParseFloat(fmt.Sprint(v), 64)
	}
}

func roundPtr(v *float64, places int) any {
	if v == nil {
		return nil
	}
	return mathRound(*v, places)
}

func mathRound(v float64, places int) float64 {
	p := math.Pow(10, float64(places))
	return math.Round(v*p) / p
}
