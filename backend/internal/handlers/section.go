package handlers

import (
	"errors"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"

	"corporate.admsr.ru/backend/internal/auth"
	"corporate.admsr.ru/backend/internal/httpx"
)

func requireSection(w http.ResponseWriter, r *http.Request, pool *pgxpool.Pool, svc *auth.Service, section string) (*auth.User, bool) {
	u, err := svc.RequireUser(r.Context(), r)
	if err != nil {
		if errors.Is(err, auth.ErrUnauthorized) {
			httpx.Fail(w, http.StatusUnauthorized, "Требуется авторизация")
			return nil, false
		}
		httpx.Fail(w, http.StatusInternalServerError, "Ошибка подключения к БД")
		return nil, false
	}
	if !auth.CanEditSection(r.Context(), pool, u, section) {
		httpx.Fail(w, http.StatusForbidden, "Недостаточно прав для этого раздела")
		return nil, false
	}
	return u, true
}

func requireAdmin(w http.ResponseWriter, r *http.Request, svc *auth.Service) (*auth.User, bool) {
	u, err := svc.RequireAdmin(r.Context(), r)
	if err != nil {
		if errors.Is(err, auth.ErrUnauthorized) {
			httpx.Fail(w, http.StatusUnauthorized, "Требуется авторизация")
			return nil, false
		}
		if errors.Is(err, auth.ErrForbidden) {
			httpx.Fail(w, http.StatusForbidden, "Недостаточно прав")
			return nil, false
		}
		httpx.Fail(w, http.StatusInternalServerError, "Ошибка подключения к БД")
		return nil, false
	}
	return u, true
}

func requireUser(w http.ResponseWriter, r *http.Request, svc *auth.Service) (*auth.User, bool) {
	u, err := svc.RequireUser(r.Context(), r)
	if err != nil {
		if errors.Is(err, auth.ErrUnauthorized) {
			httpx.Fail(w, http.StatusUnauthorized, "Требуется авторизация")
			return nil, false
		}
		httpx.Fail(w, http.StatusInternalServerError, "Ошибка подключения к БД")
		return nil, false
	}
	return u, true
}
