package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"corporate.admsr.ru/backend/internal/auth"
)

// Реакции привязаны к сотруднику: без сессии — 401 до обращения к БД
// (Pool = nil, пропущенная проверка упадёт паникой). Это касается и старого
// action=like, который раньше менял анонимный счётчик без входа.
func TestNewsReactionsRequireSession(t *testing.T) {
	h := &News{Pool: nil, Auth: &auth.Service{Pool: nil}}
	cases := []struct {
		name, method, target, body string
	}{
		{"react", http.MethodPost, "/api/news.php?id=1&action=react", `{"reaction":"love","active":true}`},
		{"legacy like", http.MethodPost, "/api/news.php?id=1&action=like", `{"liked":true}`},
		{"reactors", http.MethodGet, "/api/news.php?id=1&action=reactors&reaction=like", ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			req := httptest.NewRequest(c.method, c.target, strings.NewReader(c.body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)
			if rec.Code != http.StatusUnauthorized {
				t.Fatalf("want 401, got %d: %s", rec.Code, rec.Body.String())
			}
		})
	}
}

func TestNewsReactionKeys(t *testing.T) {
	for _, k := range []string{"like", "love", "party"} {
		if !isNewsReaction(k) {
			t.Errorf("%q must be allowed", k)
		}
	}
	for _, k := range []string{"", "LIKE", "dislike", "like;drop"} {
		if isNewsReaction(k) {
			t.Errorf("%q must be rejected", k)
		}
	}
}
