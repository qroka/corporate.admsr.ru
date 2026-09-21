package tests

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("not found")

type PersistError struct {
	Message string
}

func (e *PersistError) Error() string { return e.Message }

func Bool(v any) bool {
	switch x := v.(type) {
	case bool:
		return x
	case string:
		return x == "t" || x == "1" || x == "true"
	case int:
		return x == 1
	case int64:
		return x == 1
	case float64:
		return x == 1
	default:
		return false
	}
}

func SecToHMS(sec any) string {
	if sec == nil {
		return ""
	}
	var s int
	switch x := sec.(type) {
	case int:
		s = x
	case int32:
		s = int(x)
	case int64:
		s = int(x)
	case float64:
		s = int(x)
	default:
		return ""
	}
	if s < 0 {
		s = 0
	}
	return fmt.Sprintf("%02d:%02d:%02d", s/3600, (s%3600)/60, s%60)
}

func HMSToSec(v string) *int {
	if v == "" {
		return nil
	}
	parts := strings.Split(v, ":")
	if len(parts) != 3 {
		return nil
	}
	h, _ := strconv.Atoi(parts[0])
	m, _ := strconv.Atoi(parts[1])
	s, _ := strconv.Atoi(parts[2])
	total := h*3600 + m*60 + s
	return &total
}

type DBFormRow struct {
	ID                 int64
	ListNo             *int32
	Status             string
	Kind               string
	Visibility         string
	Title              string
	Description        *string
	CompletionMessage  *string
	Shuffle            bool
	ShuffleOptions     bool
	ShowProgress       bool
	FreeNavigation     bool
	Anonymous          bool
	AllowChangeAnswer  bool
	LiveResults        bool
	AllowRevote        bool
	NotifyCreator      bool
	UsePassingScore    bool
	PassingScore       int32
	ShowCorrectAnswers bool
	RestrictByOfo      bool
	UseTimeLimit       bool
	TimeLimitSec       *int32
	LimitAttempts      bool
	Attempts           int32
	UseStart           bool
	StartsAt           *string
	UseEnd             bool
	EndsAt             *string
	ShowResult         string
	AccessByLink       bool
	LinkAccess         *string
	AccessToken        *string
	OwnerID            *int64
	CreatedAt          any
	UpdatedAt          any
}

func strVal(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

func AssembleForm(ctx context.Context, pool *pgxpool.Pool, row DBFormRow, viewerID int64) (map[string]any, error) {
	fid := row.ID

	qRows, err := pool.Query(ctx, `
		SELECT id, title, hint, type, required, scale_min, scale_max, scale_min_label, scale_max_label, correct_value
		FROM public.test_questions WHERE form_id = $1 ORDER BY position, id`, fid)
	if err != nil {
		return nil, err
	}
	defer qRows.Close()

	var questions []map[string]any
	for qRows.Next() {
		var (
			qid                                           int64
			title, hint, qtype                            string
			required                                      bool
			scaleMin, scaleMax                            int32
			scaleMinLabel, scaleMaxLabel, correctValue    *string
		)
		if err := qRows.Scan(&qid, &title, &hint, &qtype, &required, &scaleMin, &scaleMax, &scaleMinLabel, &scaleMaxLabel, &correctValue); err != nil {
			return nil, err
		}

		optRows, err := pool.Query(ctx, `
			SELECT id, text, is_correct FROM public.test_options
			WHERE question_id = $1 ORDER BY position, id`, qid)
		if err != nil {
			return nil, err
		}
		var options []map[string]any
		var correctSingle *string
		var correctMulti []string
		for optRows.Next() {
			var oid int64
			var text string
			var isCorrect bool
			if err := optRows.Scan(&oid, &text, &isCorrect); err != nil {
				optRows.Close()
				return nil, err
			}
			sid := strconv.FormatInt(oid, 10)
			options = append(options, map[string]any{"id": sid, "text": text})
			if isCorrect {
				correctSingle = &sid
				correctMulti = append(correctMulti, sid)
			}
		}
		optRows.Close()

		var correct any
		switch qtype {
		case "single", "dropdown":
			correct = correctSingle
		case "multiple":
			if len(correctMulti) == 0 {
				correct = []string{}
			} else {
				correct = correctMulti
			}
		case "scale", "number":
			if correctValue != nil && *correctValue != "" {
				if f, err := strconv.ParseFloat(*correctValue, 64); err == nil {
					correct = f
				} else {
					correct = nil
				}
			} else {
				correct = nil
			}
		default:
			correct = correctValue
		}

		questions = append(questions, map[string]any{
			"id":              strconv.FormatInt(qid, 10),
			"title":           title,
			"hint":            hint,
			"type":            qtype,
			"required":        required,
			"options":         options,
			"scaleMin":        scaleMin,
			"scaleMax":        scaleMax,
			"scaleMinLabel":   strVal(scaleMinLabel),
			"scaleMaxLabel":   strVal(scaleMaxLabel),
			"correct":         correct,
		})
	}
	if questions == nil {
		questions = []map[string]any{}
	}

	var initialOfo, directedOfo []int64
	aRows, err := pool.Query(ctx, `SELECT ofo_unit_id, source FROM public.test_audience_ofo WHERE form_id = $1`, fid)
	if err != nil {
		return nil, err
	}
	for aRows.Next() {
		var id int64
		var source string
		if err := aRows.Scan(&id, &source); err != nil {
			aRows.Close()
			return nil, err
		}
		if source == "initial" {
			initialOfo = append(initialOfo, id)
		} else {
			directedOfo = append(directedOfo, id)
		}
	}
	aRows.Close()

	var initialUsers, directedUsers []int64
	uRows, err := pool.Query(ctx, `SELECT user_id, source FROM public.test_audience_users WHERE form_id = $1`, fid)
	if err != nil {
		return nil, err
	}
	for uRows.Next() {
		var id int64
		var source string
		if err := uRows.Scan(&id, &source); err != nil {
			uRows.Close()
			return nil, err
		}
		if source == "initial" {
			initialUsers = append(initialUsers, id)
		} else {
			directedUsers = append(directedUsers, id)
		}
	}
	uRows.Close()

	attemptsUsed := 0
	if viewerID > 0 {
		_ = pool.QueryRow(ctx, `
			SELECT COUNT(*) FROM public.test_attempts
			WHERE form_id = $1 AND user_id = $2 AND status = 'completed'`, fid, viewerID).Scan(&attemptsUsed)
	}

	var listID any
	if row.ListNo != nil {
		listID = int(*row.ListNo)
	}

	var accessToken any
	if viewerID > 0 && row.OwnerID != nil && *row.OwnerID == viewerID {
		accessToken = row.AccessToken
	} else {
		accessToken = nil
	}

	mine := viewerID > 0 && row.OwnerID != nil && *row.OwnerID == viewerID

	return map[string]any{
		"id":                  fid,
		"listId":              listID,
		"status":              row.Status,
		"kind":                row.Kind,
		"visibility":          row.Visibility,
		"title":               row.Title,
		"description":         strVal(row.Description),
		"completionMessage":   strVal(row.CompletionMessage),
		"shuffle":             row.Shuffle,
		"shuffleOptions":      row.ShuffleOptions,
		"showProgress":        row.ShowProgress,
		"freeNavigation":      row.FreeNavigation,
		"anonymous":           row.Anonymous,
		"allowChangeAnswer":   row.AllowChangeAnswer,
		"liveResults":         row.LiveResults,
		"allowRevote":         row.AllowRevote,
		"notifyAdmin":         row.NotifyCreator,
		"usePassingScore":     row.UsePassingScore,
		"passingScore":        row.PassingScore,
		"showCorrectAnswers":  row.ShowCorrectAnswers,
		"restrictByOfo":       row.RestrictByOfo,
		"useTimeLimit":        row.UseTimeLimit,
		"timeLimit":           SecToHMS(row.TimeLimitSec),
		"limitAttempts":       row.LimitAttempts,
		"attempts":            row.Attempts,
		"useStart":            row.UseStart,
		"startsAt":            strVal(row.StartsAt),
		"useEnd":              row.UseEnd,
		"endsAt":              strVal(row.EndsAt),
		"showResult":          row.ShowResult,
		"accessByLink":        row.AccessByLink,
		"linkAccess":          strVal(row.LinkAccess),
		"accessToken":         accessToken,
		"ownerId":             row.OwnerID,
		"mine":                mine,
		"ofoIds":              initialOfo,
		"recipients":          initialUsers,
		"directedOfo":         directedOfo,
		"directedUsers":       directedUsers,
		"attemptsUsed":        attemptsUsed,
		"questions":           questions,
		"createdAt":           row.CreatedAt,
		"updatedAt":           row.UpdatedAt,
	}, nil
}

func ScanFormRow(row pgx.Row) (DBFormRow, error) {
	var f DBFormRow
	err := row.Scan(
		&f.ID, &f.ListNo, &f.Status, &f.Kind, &f.Visibility, &f.Title, &f.Description, &f.CompletionMessage,
		&f.Shuffle, &f.ShuffleOptions, &f.ShowProgress, &f.FreeNavigation, &f.Anonymous, &f.AllowChangeAnswer,
		&f.LiveResults, &f.AllowRevote, &f.NotifyCreator, &f.UsePassingScore, &f.PassingScore, &f.ShowCorrectAnswers,
		&f.RestrictByOfo, &f.UseTimeLimit, &f.TimeLimitSec, &f.LimitAttempts, &f.Attempts,
		&f.UseStart, &f.StartsAt, &f.UseEnd, &f.EndsAt, &f.ShowResult, &f.AccessByLink, &f.LinkAccess,
		&f.AccessToken, &f.OwnerID, &f.CreatedAt, &f.UpdatedAt,
	)
	return f, err
}

// FormSelectCols is the SELECT column list for test_forms rows.
func FormSelectCols() string { return formSelectCols }

const formSelectCols = `
	id, list_no, status, kind, visibility, title, description, completion_message,
	shuffle, shuffle_options, show_progress, free_navigation, anonymous, allow_change_answer,
	live_results, allow_revote, notify_creator, use_passing_score, passing_score, show_correct_answers,
	restrict_by_ofo, use_time_limit, time_limit_sec, limit_attempts, attempts,
	use_start, starts_at, use_end, ends_at, show_result, access_by_link, link_access,
	access_token, owner_id, created_at, updated_at`

func LoadForm(ctx context.Context, pool *pgxpool.Pool, id, viewerID int64) (map[string]any, error) {
	row := pool.QueryRow(ctx, `SELECT `+formSelectCols+` FROM public.test_forms WHERE id = $1`, id)
	f, err := ScanFormRow(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return AssembleForm(ctx, pool, f, viewerID)
}

func StripCorrect(form map[string]any) map[string]any {
	qs, _ := form["questions"].([]map[string]any)
	if qs == nil {
		if raw, ok := form["questions"].([]any); ok {
			for _, item := range raw {
				if q, ok := item.(map[string]any); ok {
					delete(q, "correct")
				}
			}
		}
		return form
	}
	for _, q := range qs {
		delete(q, "correct")
	}
	return form
}

type QuestionEval struct {
	ID           int64
	Type         string
	CorrectValue *string
}

type AnswerDetail struct {
	QuestionID  int64
	Type        string
	TextValue   *string
	NumberValue *float64
	Selected    []int64
	Answered    bool
	IsCorrect   *bool
}

type EvalResult struct {
	Score        *float64
	Passed       *bool
	CorrectCount int
	Scorable     int
	Details      []AnswerDetail
}

func EvaluateAnswers(ctx context.Context, pool *pgxpool.Pool, formRow DBFormRow, questions []QuestionEval, answers map[string]any) (EvalResult, error) {
	isTest := formRow.Kind == "test"
	norm := func(s string) string {
		return strings.ToLower(strings.TrimSpace(s))
	}

	var result EvalResult
	for _, q := range questions {
		qid := q.ID
		qtype := q.Type
		val := answers[strconv.FormatInt(qid, 10)]
		if val == nil {
			val = answers[fmt.Sprint(qid)]
		}

		var textVal *string
		var numVal *float64
		var selected []int64
		answered := false

		switch qtype {
		case "single", "dropdown":
			if val != nil && fmt.Sprint(val) != "" {
				if id, err := toInt64(val); err == nil {
					selected = []int64{id}
					answered = true
				}
			}
		case "multiple":
			if arr, ok := val.([]any); ok && len(arr) > 0 {
				for _, item := range arr {
					if id, err := toInt64(item); err == nil {
						selected = append(selected, id)
					}
				}
				answered = len(selected) > 0
			}
		case "scale", "number":
			if val != nil && fmt.Sprint(val) != "" {
				if f, err := toFloat(val); err == nil {
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

		var isCorrect *bool
		if isTest {
			hasCorrect := false
			var correctOptIDs []int64
			cv := q.CorrectValue
			if qtype == "single" || qtype == "multiple" || qtype == "dropdown" {
				optRows, err := pool.Query(ctx, `SELECT id, is_correct FROM public.test_options WHERE question_id = $1`, qid)
				if err != nil {
					return result, err
				}
				for optRows.Next() {
					var oid int64
					var ic bool
					if err := optRows.Scan(&oid, &ic); err != nil {
						optRows.Close()
						return result, err
					}
					if ic {
						correctOptIDs = append(correctOptIDs, oid)
					}
				}
				optRows.Close()
				hasCorrect = len(correctOptIDs) > 0
				if hasCorrect {
					sortInt64s(correctOptIDs)
					sel := append([]int64(nil), selected...)
					sortInt64s(sel)
					var ok bool
					if qtype == "multiple" {
						ok = int64SlicesEqual(sel, correctOptIDs)
					} else {
						ok = len(sel) == 1 && containsInt64(correctOptIDs, sel[0])
					}
					isCorrect = &ok
				}
			} else if cv != nil && *cv != "" {
				hasCorrect = true
				var ok bool
				if qtype == "scale" || qtype == "number" {
					if numVal != nil {
						if cvf, err := strconv.ParseFloat(*cv, 64); err == nil {
							ok = math.Abs(cvf-*numVal) < 1e-9
						}
					}
				} else if textVal != nil {
					ok = norm(*textVal) == norm(*cv)
				}
				isCorrect = &ok
			}
			if hasCorrect {
				result.Scorable++
				if isCorrect != nil && *isCorrect {
					result.CorrectCount++
				}
			} else {
				isCorrect = nil
			}
		}

		result.Details = append(result.Details, AnswerDetail{
			QuestionID: qid, Type: qtype, TextValue: textVal, NumberValue: numVal,
			Selected: selected, Answered: answered, IsCorrect: isCorrect,
		})
	}

	if isTest && result.Scorable > 0 {
		score := math.Round(float64(result.CorrectCount)/float64(result.Scorable)*100*100) / 100
		result.Score = &score
		if formRow.UsePassingScore {
			passed := score >= float64(formRow.PassingScore)
			result.Passed = &passed
		}
	}
	return result, nil
}

func PersistForm(ctx context.Context, pool *pgxpool.Pool, data map[string]any, viewerID int64) (int64, error) {
	var id *int64
	if raw, ok := data["id"]; ok && raw != nil && fmt.Sprint(raw) != "" && fmt.Sprint(raw) != "0" {
		if v, err := toInt64(raw); err == nil {
			id = &v
		}
	}

	b := func(k string) bool {
		v, ok := data[k]
		return ok && Bool(v)
	}
	enum := func(v any, allowed []string, def string) string {
		s := fmt.Sprint(v)
		for _, a := range allowed {
			if s == a {
				return a
			}
		}
		return def
	}

	kind := enum(data["kind"], []string{"test", "survey", "poll"}, "test")
	visibility := enum(data["visibility"], []string{"public", "private"}, "public")
	showResult := enum(data["showResult"], []string{"immediate", "after", "never"}, "after")
	linkAccess := enum(data["linkAccess"], []string{"authorized", "guest", "any"}, "any")

	passingScore := clampInt(toIntDefault(data["passingScore"], 70), 0, 100)
	attempts := clampInt(toIntDefault(data["attempts"], 1), 1, math.MaxInt32)

	var timeLimitSec *int
	if tl, ok := data["timeLimit"].(string); ok {
		timeLimitSec = HMSToSec(tl)
	} else if data["timeLimit"] != nil {
		timeLimitSec = HMSToSec(fmt.Sprint(data["timeLimit"]))
	}

	var startsAt, endsAt any
	if s, ok := data["startsAt"].(string); ok && s != "" {
		startsAt = s
	}
	if s, ok := data["endsAt"].(string); ok && s != "" {
		endsAt = s
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx)

	var formID int64
	if id != nil {
		var ownerID *int64
		err := tx.QueryRow(ctx, `SELECT owner_id FROM public.test_forms WHERE id = $1`, *id).Scan(&ownerID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return 0, &PersistError{Message: "Форма не найдена"}
			}
			return 0, err
		}
		if ownerID != nil && *ownerID != viewerID {
			return 0, &PersistError{Message: "Нет прав на изменение"}
		}
		_, err = tx.Exec(ctx, `
			UPDATE public.test_forms SET
				kind=$2, visibility=$3, title=$4, description=$5, completion_message=$6,
				shuffle=$7, shuffle_options=$8, show_progress=$9, free_navigation=$10, anonymous=$11,
				allow_change_answer=$12, live_results=$13, allow_revote=$14, notify_creator=$15,
				use_passing_score=$16, passing_score=$17, show_correct_answers=$18, restrict_by_ofo=$19,
				use_time_limit=$20, time_limit_sec=$21, limit_attempts=$22, attempts=$23,
				use_start=$24, starts_at=$25, use_end=$26, ends_at=$27, show_result=$28,
				access_by_link=$29, link_access=$30, updated_at=now()
			WHERE id=$1`,
			*id, kind, visibility, strField(data, "title"), strField(data, "description"), strField(data, "completionMessage"),
			b("shuffle"), b("shuffleOptions"), b("showProgress"), b("freeNavigation"), b("anonymous"),
			b("allowChangeAnswer"), b("liveResults"), b("allowRevote"), b("notifyAdmin"),
			b("usePassingScore"), passingScore, b("showCorrectAnswers"), b("restrictByOfo"),
			b("useTimeLimit"), timeLimitSec, b("limitAttempts"), attempts,
			b("useStart"), startsAt, b("useEnd"), endsAt, showResult,
			b("accessByLink"), linkAccess,
		)
		if err != nil {
			return 0, err
		}
		formID = *id
	} else {
		var owner any
		if viewerID > 0 {
			owner = viewerID
		}
		err = tx.QueryRow(ctx, `
			INSERT INTO public.test_forms (
				owner_id, kind, visibility, title, description, completion_message,
				shuffle, shuffle_options, show_progress, free_navigation, anonymous,
				allow_change_answer, live_results, allow_revote, notify_creator,
				use_passing_score, passing_score, show_correct_answers, restrict_by_ofo,
				use_time_limit, time_limit_sec, limit_attempts, attempts,
				use_start, starts_at, use_end, ends_at, show_result,
				access_by_link, link_access
			) VALUES (
				$1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24,$25,$26,$27,$28,$29,$30
			) RETURNING id`,
			owner, kind, visibility, strField(data, "title"), strField(data, "description"), strField(data, "completionMessage"),
			b("shuffle"), b("shuffleOptions"), b("showProgress"), b("freeNavigation"), b("anonymous"),
			b("allowChangeAnswer"), b("liveResults"), b("allowRevote"), b("notifyAdmin"),
			b("usePassingScore"), passingScore, b("showCorrectAnswers"), b("restrictByOfo"),
			b("useTimeLimit"), timeLimitSec, b("limitAttempts"), attempts,
			b("useStart"), startsAt, b("useEnd"), endsAt, showResult,
			b("accessByLink"), linkAccess,
		).Scan(&formID)
		if err != nil {
			return 0, err
		}
	}

	if _, err := tx.Exec(ctx, `DELETE FROM public.test_questions WHERE form_id = $1`, formID); err != nil {
		return 0, err
	}

	questions, _ := data["questions"].([]any)
	for qi, raw := range questions {
		q, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		qtype := fmt.Sprint(q["type"])
		if qtype == "" {
			qtype = "single"
		}
		correct := q["correct"]
		var correctValue *string
		if qtype == "yesno" || qtype == "text" || qtype == "textarea" || qtype == "date" ||
			qtype == "scale" || qtype == "number" {
			if correct != nil && fmt.Sprint(correct) != "" {
				s := fmt.Sprint(correct)
				correctValue = &s
			}
		}

		scaleMin := toIntDefault(q["scaleMin"], 1)
		scaleMax := toIntDefault(q["scaleMax"], 5)

		var qid int64
		err = tx.QueryRow(ctx, `
			INSERT INTO public.test_questions
				(form_id, position, type, title, hint, required, scale_min, scale_max, scale_min_label, scale_max_label, correct_value)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11) RETURNING id`,
			formID, qi, qtype, strField(q, "title"), strField(q, "hint"), Bool(q["required"]),
			scaleMin, scaleMax, strField(q, "scaleMinLabel"), strField(q, "scaleMaxLabel"), correctValue,
		).Scan(&qid)
		if err != nil {
			return 0, err
		}

		if qtype == "single" || qtype == "multiple" || qtype == "dropdown" {
			var correctIDs []string
			if qtype == "multiple" {
				if arr, ok := correct.([]any); ok {
					for _, item := range arr {
						correctIDs = append(correctIDs, fmt.Sprint(item))
					}
				}
			} else if correct != nil {
				correctIDs = []string{fmt.Sprint(correct)}
			}
			opts, _ := q["options"].([]any)
			for oi, oraw := range opts {
				opt, ok := oraw.(map[string]any)
				if !ok {
					continue
				}
				clientID := fmt.Sprint(opt["id"])
				isCorrect := false
				for _, cid := range correctIDs {
					if cid == clientID {
						isCorrect = true
						break
					}
				}
				if _, err := tx.Exec(ctx, `
					INSERT INTO public.test_options (question_id, position, text, is_correct)
					VALUES ($1,$2,$3,$4)`, qid, oi, strField(opt, "text"), isCorrect); err != nil {
					return 0, err
				}
			}
		}
	}

	if _, err := tx.Exec(ctx, `DELETE FROM public.test_audience_ofo WHERE form_id = $1 AND source = 'initial'`, formID); err != nil {
		return 0, err
	}
	if _, err := tx.Exec(ctx, `DELETE FROM public.test_audience_users WHERE form_id = $1 AND source = 'initial'`, formID); err != nil {
		return 0, err
	}

	for _, t := range uniqueInt64s(toInt64Slice(data["ofoIds"])) {
		if _, err := tx.Exec(ctx, `
			INSERT INTO public.test_audience_ofo (form_id, ofo_unit_id, source) VALUES ($1,$2,'initial')
			ON CONFLICT (form_id, ofo_unit_id) DO NOTHING`, formID, t); err != nil {
			return 0, err
		}
	}
	for _, t := range uniqueInt64s(toInt64Slice(data["recipients"])) {
		if _, err := tx.Exec(ctx, `
			INSERT INTO public.test_audience_users (form_id, user_id, source) VALUES ($1,$2,'initial')
			ON CONFLICT (form_id, user_id) DO NOTHING`, formID, t); err != nil {
			return 0, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, err
	}
	return formID, nil
}

func HasCourseTestLinksTable(ctx context.Context, pool *pgxpool.Pool) bool {
	var n int
	err := pool.QueryRow(ctx, `SELECT 1 FROM public.course_test_links LIMIT 0`).Scan(&n)
	return err == nil
}

func strField(m map[string]any, k string) string {
	if v, ok := m[k]; ok && v != nil {
		return fmt.Sprint(v)
	}
	return ""
}

func ToInt64Public(v any) (int64, error) { return toInt64(v) }

func ToFloatPublic(v any) (float64, error) { return toFloat(v) }

func ToInt64Must(v any) int64 {
	n, _ := toInt64(v)
	return n
}

func toInt64(v any) (int64, error) {
	switch x := v.(type) {
	case nil:
		return 0, fmt.Errorf("nil")
	case float64:
		return int64(x), nil
	case float32:
		return int64(x), nil
	case int:
		return int64(x), nil
	case int8:
		return int64(x), nil
	case int16:
		return int64(x), nil
	case int32:
		return int64(x), nil
	case int64:
		return x, nil
	case uint:
		return int64(x), nil
	case uint32:
		return int64(x), nil
	case uint64:
		if x > math.MaxInt64 {
			return 0, fmt.Errorf("uint64 overflow")
		}
		return int64(x), nil
	case *int64:
		if x == nil {
			return 0, fmt.Errorf("nil")
		}
		return *x, nil
	case *int:
		if x == nil {
			return 0, fmt.Errorf("nil")
		}
		return int64(*x), nil
	case *int32:
		if x == nil {
			return 0, fmt.Errorf("nil")
		}
		return int64(*x), nil
	case *float64:
		if x == nil {
			return 0, fmt.Errorf("nil")
		}
		return int64(*x), nil
	case string:
		return strconv.ParseInt(strings.TrimSpace(x), 10, 64)
	case []byte:
		return strconv.ParseInt(strings.TrimSpace(string(x)), 10, 64)
	default:
		s := strings.TrimSpace(fmt.Sprint(v))
		if s == "" || s == "<nil>" {
			return 0, fmt.Errorf("empty")
		}
		return strconv.ParseInt(s, 10, 64)
	}
}

func toFloat(v any) (float64, error) {
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

func toIntDefault(v any, def int) int {
	if v == nil {
		return def
	}
	n, err := toInt64(v)
	if err != nil {
		return def
	}
	return int(n)
}

func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func toInt64Slice(v any) []int64 {
	arr, ok := v.([]any)
	if !ok {
		return nil
	}
	var out []int64
	for _, item := range arr {
		if n, err := toInt64(item); err == nil {
			out = append(out, n)
		}
	}
	return out
}

func uniqueInt64s(in []int64) []int64 {
	seen := map[int64]struct{}{}
	var out []int64
	for _, v := range in {
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	return out
}

func sortInt64s(a []int64) {
	for i := 0; i < len(a); i++ {
		for j := i + 1; j < len(a); j++ {
			if a[j] < a[i] {
				a[i], a[j] = a[j], a[i]
			}
		}
	}
}

func int64SlicesEqual(a, b []int64) bool {
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

func containsInt64(a []int64, v int64) bool {
	for _, x := range a {
		if x == v {
			return true
		}
	}
	return false
}
