package httpx

import (
	"net/http"
	"strings"
)

var allowedOrigins = map[string]struct{}{
	"http://localhost:5173":     {},
	"http://localhost:5174":     {},
	"http://127.0.0.1:5173":     {},
	"http://127.0.0.1:5174":     {},
}

// CORS wraps handlers with PHP-compatible CORS headers for Vite dev.
func CORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if _, ok := allowedOrigins[origin]; ok {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Credentials", "true")
		} else if origin == "" {
			// same-origin / server-side — no CORS header needed
		} else {
			w.Header().Set("Access-Control-Allow-Origin", "http://localhost:5173")
		}

		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Session-Token")
		w.Header().Set("Access-Control-Max-Age", "86400")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// AllowMethods returns 405 unless method is in the list (after OPTIONS handled by CORS).
func AllowMethods(methods ...string) func(http.Handler) http.Handler {
	allowed := make(map[string]struct{}, len(methods))
	for _, m := range methods {
		allowed[strings.ToUpper(m)] = struct{}{}
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if _, ok := allowed[r.Method]; !ok {
				MethodNotAllowed(w)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
