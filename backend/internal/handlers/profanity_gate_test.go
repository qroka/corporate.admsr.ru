package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestProfanityGateCleanTextPasses(t *testing.T) {
	rec := httptest.NewRecorder()
	got, ok := profanityGate(rec, 1, map[string]any{}, "Отличная новость, спасибо!")
	if !ok || got != "Отличная новость, спасибо!" {
		t.Fatalf("got %q ok=%v", got, ok)
	}
}

func TestProfanityGateAsksChallenge(t *testing.T) {
	rec := httptest.NewRecorder()
	if _, ok := profanityGate(rec, 1, map[string]any{}, "ну и хуйня"); ok {
		t.Fatal("текст с матом прошёл без задачи")
	}
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status %d", rec.Code)
	}
	var env struct {
		Success bool
		Data    struct {
			Profanity profanityChallenge `json:"profanity"`
		}
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
		t.Fatal(err)
	}
	ch := env.Data.Profanity
	if env.Success || ch.Token == "" || ch.Question == "" || ch.Masked != "ну и х***я" {
		t.Fatalf("challenge %+v", ch)
	}
}

func TestProfanityGateAcceptsRightAnswer(t *testing.T) {
	const user, text = int64(7), "ну и хуйня"
	token := signProfanityToken(user, text, 42, time.Now().Add(time.Minute))

	rec := httptest.NewRecorder()
	got, ok := profanityGate(rec, user, map[string]any{"challengeToken": token, "challengeAnswer": "42"}, text)
	if !ok || got != "ну и х***я" {
		t.Fatalf("got %q ok=%v body=%s", got, ok, rec.Body.String())
	}

	// неверный ответ, чужой пользователь, другой текст — новая задача
	for _, c := range []struct {
		user   int64
		text   string
		answer string
	}{
		{user, text, "41"},
		{user + 1, text, "42"},
		{user, "ну и хуйня какая-то", "42"},
		{user, text, "сорок два"},
	} {
		rec := httptest.NewRecorder()
		if _, ok := profanityGate(rec, c.user, map[string]any{"challengeToken": token, "challengeAnswer": c.answer}, c.text); ok {
			t.Errorf("прошло: %+v", c)
		}
		if rec.Code != http.StatusUnprocessableEntity || !strings.Contains(rec.Body.String(), "Неверный ответ") {
			t.Errorf("%+v: %d %s", c, rec.Code, rec.Body.String())
		}
	}
	profanityWrong.reset(user)
	profanityWrong.reset(user + 1)
}

func TestProfanityTokenExpires(t *testing.T) {
	token := signProfanityToken(1, "x", 5, time.Now().Add(-time.Second))
	if r := verifyProfanityToken(token, 1, "x", 5, true, time.Now()); r != challengeExpired {
		t.Fatalf("got %v", r)
	}
}

func TestProfanityGateBlocksBruteForce(t *testing.T) {
	const user, text = int64(99), "сука"
	token := signProfanityToken(user, text, 3, time.Now().Add(time.Minute))
	for i := 0; i < profanityMaxWrong; i++ {
		profanityGate(httptest.NewRecorder(), user, map[string]any{"challengeToken": token, "challengeAnswer": "0"}, text)
	}
	rec := httptest.NewRecorder()
	if _, ok := profanityGate(rec, user, map[string]any{"challengeToken": token, "challengeAnswer": "3"}, text); ok || rec.Code != http.StatusTooManyRequests {
		t.Fatalf("после %d неверных ответов: ok=%v code=%d", profanityMaxWrong, ok, rec.Code)
	}
	profanityWrong.reset(user)
}

func TestParseChallengeAnswer(t *testing.T) {
	for in, want := range map[string]int{"42": 42, " -7 ": -7, "−7": -7, "–12": -12} {
		if got, ok := parseChallengeAnswer(in); !ok || got != want {
			t.Errorf("%q → %d %v", in, got, ok)
		}
	}
	if _, ok := parseChallengeAnswer("7.5"); ok {
		t.Error("7.5 принят")
	}
}

func TestFormatPoly(t *testing.T) {
	cases := map[[3]int]string{
		{2, -3, 5}:  "2x² − 3x + 5",
		{1, 0, -4}:  "x² − 4",
		{0, 7, -12}: "7x − 12",
		{0, -1, 0}:  "−x",
		{0, 0, 0}:   "0",
	}
	for in, want := range cases {
		if got := formatPoly(in[0], in[1], in[2]); got != want {
			t.Errorf("formatPoly%v = %q, want %q", in, got, want)
		}
	}
}

func TestNewMathTask(t *testing.T) {
	for i := 0; i < 200; i++ {
		q, _ := newMathTask()
		if q == "" || strings.Contains(q, "+ −") || strings.Contains(q, "1x") {
			t.Fatalf("плохая задача: %q", q)
		}
	}
}
