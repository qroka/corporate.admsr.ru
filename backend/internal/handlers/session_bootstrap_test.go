package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"corporate.admsr.ru/backend/internal/auth"
)

// Регрессия SEC-006.
//
// До исправления POST /api/session_bootstrap.php с телом {"id": N} выдавал
// валидный токен сессии любого активного пользователя без каких-либо
// доказательств личности. Теперь личность берётся только из действующей
// сессии, поэтому запрос без токена/cookie обязан завершаться 401 и
// НЕ должен доходить до БД (Pool здесь намеренно nil — обращение к нему
// в этом сценарии означало бы, что проверка сессии пропущена).
func TestSessionBootstrapRejectsBodyIDWithoutSession(t *testing.T) {
	h := &AuthHandlers{Pool: nil, Auth: &auth.Service{Pool: nil}}

	for _, body := range []string{`{"id":1}`, `{"id":999999}`, `{}`, ``} {
		req := httptest.NewRequest(http.MethodPost, "/api/session_bootstrap.php", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		h.SessionBootstrap(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("body %q: ожидался 401, получен %d (тело: %s)", body, rec.Code, rec.Body.String())
		}

		var resp map[string]any
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("body %q: ответ не JSON: %v", body, err)
		}
		if ok, _ := resp["success"].(bool); ok {
			t.Fatalf("body %q: success=true при отсутствии сессии", body)
		}
		if strings.Contains(rec.Body.String(), "sessionToken") {
			t.Fatalf("body %q: в ответе без сессии присутствует sessionToken", body)
		}
	}
}

// Метод, отличный от POST, не должен обрабатываться.
func TestSessionBootstrapRejectsNonPost(t *testing.T) {
	h := &AuthHandlers{Pool: nil, Auth: &auth.Service{Pool: nil}}
	req := httptest.NewRequest(http.MethodGet, "/api/session_bootstrap.php", nil)
	rec := httptest.NewRecorder()

	h.SessionBootstrap(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("ожидался 405, получен %d", rec.Code)
	}
}
