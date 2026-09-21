package handlers

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"corporate.admsr.ru/backend/internal/courses"
	"corporate.admsr.ru/backend/internal/tests"
)

// buildAttemptReview returns attempt meta + per-question answers (forms + courses admin drill-down).
func buildAttemptReview(ctx context.Context, pool *pgxpool.Pool, formID, attemptID int64) (map[string]any, []map[string]any, error) {
	var score *float64
	var passed *bool
	var finishedAt, durationSec any
	err := pool.QueryRow(ctx, `
		SELECT score, passed, finished_at, duration_sec FROM public.test_attempts
		WHERE id = $1 AND form_id = $2`, attemptID, formID).Scan(&score, &passed, &finishedAt, &durationSec)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil, courses.Err(http.StatusNotFound, "Попытка не найдена")
	}
	if err != nil {
		return nil, nil, err
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

	qRows, err := pool.Query(ctx, `
		SELECT id, type, title, correct_value FROM public.test_questions
		WHERE form_id = $1 ORDER BY position, id`, formID)
	if err != nil {
		return nil, nil, err
	}
	defer qRows.Close()

	var answers []map[string]any
	for qRows.Next() {
		var qid int64
		var qtype, title string
		var correctValue *string
		if err := qRows.Scan(&qid, &qtype, &title, &correctValue); err != nil {
			continue
		}
		optText := map[int64]string{}
		var correctOptIDs []int64
		if qtype == "single" || qtype == "multiple" || qtype == "dropdown" {
			optRows, _ := pool.Query(ctx, `
				SELECT id, text, is_correct FROM public.test_options
				WHERE question_id = $1 ORDER BY position, id`, qid)
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
		err := pool.QueryRow(ctx, `
			SELECT id, text_value, number_value, is_correct, answered
			FROM public.test_answers WHERE attempt_id = $1 AND question_id = $2`, attemptID, qid).Scan(
			&ansID, &textValue, &numberValue, &isCorrect, &answered)
		hasAns := err == nil

		userAnswer := "— нет ответа"
		if hasAns && answered {
			switch qtype {
			case "single", "multiple", "dropdown":
				selRows, _ := pool.Query(ctx, `SELECT option_id FROM public.test_answer_options WHERE answer_id = $1`, ansID)
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

	attempt := map[string]any{
		"id": attemptID, "score": score, "passed": passedOut,
		"finishedAt": finishedAt, "durationSec": durOut,
	}
	return attempt, answers, nil
}
