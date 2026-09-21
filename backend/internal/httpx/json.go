package httpx

import (
	"encoding/json"
	"net/http"
)

type Envelope struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
	Data    any    `json:"data"`
}

func WriteJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(v)
}

func OK(w http.ResponseWriter, data any, message string) {
	if message == "" {
		message = "OK"
	}
	WriteJSON(w, http.StatusOK, Envelope{Success: true, Message: message, Data: data})
}

func Created(w http.ResponseWriter, data any, message string) {
	if message == "" {
		message = "OK"
	}
	WriteJSON(w, http.StatusCreated, Envelope{Success: true, Message: message, Data: data})
}

func Fail(w http.ResponseWriter, status int, message string) {
	WriteJSON(w, status, Envelope{Success: false, Message: message, Data: nil})
}

func DecodeJSON(r *http.Request, dst any) error {
	defer r.Body.Close()
	dec := json.NewDecoder(r.Body)
	return dec.Decode(dst)
}

func MethodNotAllowed(w http.ResponseWriter) {
	Fail(w, http.StatusMethodNotAllowed, "Метод не поддерживается")
}
