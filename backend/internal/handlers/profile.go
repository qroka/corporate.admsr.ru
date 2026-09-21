package handlers

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"corporate.admsr.ru/backend/internal/auth"
	"corporate.admsr.ru/backend/internal/httpx"
)

type Profile struct {
	Pool *pgxpool.Pool
}

func (h *Profile) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.get(w, r)
	case http.MethodPost:
		h.post(w, r)
	default:
		httpx.Fail(w, http.StatusMethodNotAllowed, "Метод не поддерживается")
	}
}

func (h *Profile) get(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		httpx.Fail(w, http.StatusBadRequest, "Некорректный id")
		return
	}

	var (
		firstname, surname, lastname string
		ofo, userGroup               string
		phone, email, role           *string
		avatarURL                    *string
	)
	err = h.Pool.QueryRow(r.Context(), `
		SELECT id, firstname, surname, lastname, ofo, user_group, phone, email, role, avatar_url
		FROM public.user_info WHERE id = $1 LIMIT 1`, id).Scan(
		&id, &firstname, &surname, &lastname, &ofo, &userGroup, &phone, &email, &role, &avatarURL,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			httpx.Fail(w, http.StatusNotFound, "Пользователь не найден")
			return
		}
		httpx.Fail(w, http.StatusInternalServerError, "Ошибка подключения к БД")
		return
	}

	isAdmin := userGroup == "admin"
	u := &auth.User{ID: id, UserGroup: userGroup}
	sections := auth.UserSections(r.Context(), h.Pool, u)
	courseCategories := auth.UserCourseCategories(r.Context(), h.Pool, u)
	if len(sections) == 0 && isAdmin {
		sections = append([]string{}, auth.PortalSections...)
	}
	if len(courseCategories) == 0 && isAdmin {
		courseCategories = append([]string{}, auth.CourseCategories...)
	}

	ph := ""
	if phone != nil {
		ph = *phone
	}
	em := ""
	if email != nil {
		em = *email
	}
	rl := ""
	if role != nil {
		rl = *role
	}
	av := ""
	if avatarURL != nil {
		av = *avatarURL
	}

	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"success": true,
		"data": map[string]any{
			"id":               id,
			"firstname":        nullStrDef(firstname),
			"surname":          nullStrDef(surname),
			"lastname":         nullStrDef(lastname),
			"ofo":              nullStrDef(ofo),
			"user_group":       nullStrDef(userGroup),
			"phone":            ph,
			"email":            em,
			"role":             rl,
			"avatar_url":       av,
			"isAdmin":          isAdmin,
			"sections":         sections,
			"courseCategories": courseCategories,
		},
	})
}

func (h *Profile) post(w http.ResponseWriter, r *http.Request) {
	var body map[string]any
	if err := httpx.DecodeJSON(r, &body); err != nil {
		httpx.Fail(w, http.StatusBadRequest, "Некорректный JSON")
		return
	}

	id, err := strconv.ParseInt(strVal(body["id"]), 10, 64)
	if err != nil || id <= 0 {
		httpx.Fail(w, http.StatusBadRequest, "Некорректный id")
		return
	}

	firstname := trimPtr(body["firstname"])
	surname := trimPtr(body["surname"])
	lastname := trimPtr(body["lastname"])
	phone := trimPtr(body["phone"])
	email := trimPtr(body["email"])
	ofo := trimPtr(body["ofo"])
	role := trimPtr(body["role"])
	avatarURL := trimPtr(body["avatar_url"])

	_, err = h.Pool.Exec(r.Context(), `
		UPDATE public.user_info
		SET firstname = $1, surname = $2, lastname = $3,
		    phone = $4, email = $5, ofo = $6, role = $7, avatar_url = $8
		WHERE id = $9`,
		firstname, surname, lastname, phone, email, ofo, role, avatarURL, id,
	)
	if err != nil {
		httpx.Fail(w, http.StatusInternalServerError, "Ошибка подключения к БД")
		return
	}

	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"success": true,
		"data":    nil,
	})
}

func trimPtr(v any) *string {
	if v == nil {
		return nil
	}
	s := strings.TrimSpace(strVal(v))
	return &s
}
