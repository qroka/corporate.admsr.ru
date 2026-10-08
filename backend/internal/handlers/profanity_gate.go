package handlers

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"math/big"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"corporate.admsr.ru/backend/internal/httpx"
	"corporate.admsr.ru/backend/internal/profanity"
)

// Фильтр мата в комментариях и на стене (ADR-053). Текст с матом сразу не
// публикуется: сервер отвечает 422 с задачей (data.profanity). Автор решает её и
// отправляет тот же текст ещё раз с challengeToken + challengeAnswer — тогда текст
// публикуется, но найденные слова скрыты звёздочками (х****й).
//
// Токен не хранится на сервере: это подпись HMAC над (пользователь, хеш текста,
// ответ, срок). Ответа в токене нет — проверка пересчитывает подпись с присланным
// ответом. Ключ создаётся при старте процесса: после перезапуска API открытые
// задачи просто перестают подходить, и автор получает новую.

const (
	profanityChallengeTTL = 15 * time.Minute
	// Неверных ответов подряд до паузы — чтобы ответ не перебирали запросами.
	profanityMaxWrong   = 8
	profanityWrongPause = 10 * time.Minute
)

var profanitySecret = func() []byte {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return b
}()

type profanityChallenge struct {
	Token    string `json:"token"`
	Question string `json:"question"`
	// Masked — как текст будет опубликован после решения.
	Masked string `json:"masked"`
}

// profanityGate — проверка текста перед сохранением. Возвращает текст, который
// нужно сохранить (без мата — как есть; после решённой задачи — со звёздочками).
// false — ответ уже записан (задача или ошибка), хендлер должен выйти.
func profanityGate(w http.ResponseWriter, userID int64, body map[string]any, content string) (string, bool) {
	masked, found := profanity.Check(content)
	if !found {
		return content, true
	}
	token := strings.TrimSpace(strVal(body["challengeToken"]))
	if token == "" {
		writeProfanityChallenge(w, userID, content, masked,
			"В тексте есть нецензурные выражения. Чтобы опубликовать, решите задачу — часть слов всё равно скроем звёздочками.")
		return "", false
	}
	if profanityWrong.blocked(userID) {
		httpx.Fail(w, http.StatusTooManyRequests, "Слишком много неверных ответов. Попробуйте через 10 минут или уберите нецензурные слова.")
		return "", false
	}
	answer, ok := parseChallengeAnswer(strVal(body["challengeAnswer"]))
	switch verifyProfanityToken(token, userID, content, answer, ok, time.Now()) {
	case challengeOK:
		profanityWrong.reset(userID)
		return masked, true
	case challengeExpired:
		writeProfanityChallenge(w, userID, content, masked, "Время на ответ вышло — вот новая задача.")
	default:
		profanityWrong.fail(userID)
		writeProfanityChallenge(w, userID, content, masked, "Неверный ответ — вот новая задача.")
	}
	return "", false
}

func writeProfanityChallenge(w http.ResponseWriter, userID int64, content, masked, message string) {
	question, answer := newMathTask()
	ch := profanityChallenge{
		Token:    signProfanityToken(userID, content, answer, time.Now().Add(profanityChallengeTTL)),
		Question: question,
		Masked:   masked,
	}
	httpx.WriteJSON(w, http.StatusUnprocessableEntity, httpx.Envelope{
		Success: false,
		Message: message,
		Data:    map[string]any{"profanity": ch},
	})
}

// parseChallengeAnswer — целое число; «−» и «–» вместо минуса тоже принимаются.
func parseChallengeAnswer(s string) (int, bool) {
	s = strings.TrimSpace(s)
	s = strings.NewReplacer("−", "-", "–", "-", " ", "").Replace(s)
	n, err := strconv.Atoi(s)
	return n, err == nil
}

type challengeResult int

const (
	challengeWrong challengeResult = iota
	challengeOK
	challengeExpired
)

func profanityMAC(userID int64, content string, answer int, exp int64, nonce string) []byte {
	sum := sha256.Sum256([]byte(content))
	mac := hmac.New(sha256.New, profanitySecret)
	fmt.Fprintf(mac, "%d|%x|%d|%d|%s", userID, sum, answer, exp, nonce)
	return mac.Sum(nil)
}

// Токен: «срок.nonce.подпись».
func signProfanityToken(userID int64, content string, answer int, expires time.Time) string {
	nb := make([]byte, 8)
	_, _ = rand.Read(nb)
	nonce := hex.EncodeToString(nb)
	exp := expires.Unix()
	sig := base64.RawURLEncoding.EncodeToString(profanityMAC(userID, content, answer, exp, nonce))
	return strconv.FormatInt(exp, 10) + "." + nonce + "." + sig
}

func verifyProfanityToken(token string, userID int64, content string, answer int, answerOK bool, now time.Time) challengeResult {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return challengeWrong
	}
	exp, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return challengeWrong
	}
	sig, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return challengeWrong
	}
	if now.Unix() > exp {
		return challengeExpired
	}
	if !answerOK || !hmac.Equal(sig, profanityMAC(userID, content, answer, exp, parts[1])) {
		return challengeWrong
	}
	return challengeOK
}

// --- задачи ------------------------------------------------------------------

func randInt(lo, hi int) int { // [lo, hi]
	n, err := rand.Int(rand.Reader, big.NewInt(int64(hi-lo+1)))
	if err != nil {
		return lo
	}
	return lo + int(n.Int64())
}

func randNonZero(lo, hi int) int {
	for {
		if n := randInt(lo, hi); n != 0 {
			return n
		}
	}
}

// newMathTask — задача с целым ответом: значение функции, уравнение или композиция.
func newMathTask() (question string, answer int) {
	switch randInt(0, 2) {
	case 0:
		a, b, c, k := randInt(1, 3), randNonZero(-7, 7), randInt(-9, 9), randInt(2, 5)
		return fmt.Sprintf("f(x) = %s\nНайдите f(%d)", formatPoly(a, b, c), k), a*k*k + b*k + c
	case 1:
		a, x, b := randInt(2, 9), randNonZero(-9, 12), randNonZero(-20, 20)
		return fmt.Sprintf("Решите уравнение: %s = %s\nx = ?", formatPoly(0, a, b), formatInt(a*x+b)), x
	default:
		a, b, c, k := randInt(2, 4), randNonZero(-9, 9), randInt(1, 9), randInt(2, 4)
		return fmt.Sprintf("f(x) = %s,  g(x) = %s\nНайдите f(g(%d))", formatPoly(0, a, b), formatPoly(1, 0, -c), k),
			a*(k*k-c) + b
	}
}

// formatPoly — «2x² − 3x + 5» для a·x² + b·x + c (нулевые члены пропускаются).
func formatPoly(a, b, c int) string {
	var out strings.Builder
	term := func(coef int, x string) {
		if coef == 0 {
			return
		}
		abs := coef
		if coef < 0 {
			abs = -coef
		}
		switch {
		case out.Len() == 0 && coef < 0:
			out.WriteString("−")
		case out.Len() > 0 && coef < 0:
			out.WriteString(" − ")
		case out.Len() > 0:
			out.WriteString(" + ")
		}
		if abs != 1 || x == "" {
			out.WriteString(strconv.Itoa(abs))
		}
		out.WriteString(x)
	}
	term(a, "x²")
	term(b, "x")
	term(c, "")
	if out.Len() == 0 {
		return "0"
	}
	return out.String()
}

func formatInt(n int) string {
	if n < 0 {
		return "−" + strconv.Itoa(-n)
	}
	return strconv.Itoa(n)
}

// --- неверные ответы -----------------------------------------------------------

type wrongAnswers struct {
	mu sync.Mutex
	m  map[int64]wrongEntry
}

type wrongEntry struct {
	n     int
	until time.Time // до какого времени считать (и блокировать после лимита)
}

var profanityWrong = &wrongAnswers{m: map[int64]wrongEntry{}}

func (wa *wrongAnswers) blocked(userID int64) bool {
	wa.mu.Lock()
	defer wa.mu.Unlock()
	e, ok := wa.m[userID]
	if !ok {
		return false
	}
	if time.Now().After(e.until) {
		delete(wa.m, userID)
		return false
	}
	return e.n >= profanityMaxWrong
}

func (wa *wrongAnswers) fail(userID int64) {
	wa.mu.Lock()
	defer wa.mu.Unlock()
	e := wa.m[userID]
	if time.Now().After(e.until) {
		e = wrongEntry{}
	}
	e.n++
	e.until = time.Now().Add(profanityWrongPause)
	wa.m[userID] = e
}

func (wa *wrongAnswers) reset(userID int64) {
	wa.mu.Lock()
	defer wa.mu.Unlock()
	delete(wa.m, userID)
}
