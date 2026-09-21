package handlers

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"

	"corporate.admsr.ru/backend/internal/auth"

	"corporate.admsr.ru/backend/internal/httpx"
)

type Users struct {
	Pool *pgxpool.Pool
	Auth *auth.Service
}

func (h *Users) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Справочник пользователей и его редактирование — только для администратора.
	// До исправления (SEC-001) эндпоинт был полностью открыт: GET отдавал колонку
	// password всех сотрудников, PUT позволял менять любому пользователю пароль
	// и user_group.
	if _, ok := requireAdmin(w, r, h.Auth); !ok {
		return
	}
	switch r.Method {
	case http.MethodGet:
		h.list(w, r)
	case http.MethodPut:
		h.put(w, r)
	default:
		httpx.MethodNotAllowed(w)
	}
}

func (h *Users) list(w http.ResponseWriter, r *http.Request) {
	// Колонка password намеренно не выбирается и не отдаётся клиенту (SEC-001/SEC-005):
	// админке она не нужна, а любая утечка ответа означала бы утечку учётных данных.
	const qWithGroups = `
		SELECT u.id, u.status, u.login, u.firstname, u.surname, u.lastname, u.ofo, u.user_group, u.phone, u.email,
		       (u.auth = true AND u.last_activity IS NOT NULL AND u.last_activity > now() - interval '24 hours') AS auth,
		       to_char(u.last_activity AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"') AS last_activity,
		       u.avatar_url, u.role,
		       (SELECT string_agg(g.name, ', ' ORDER BY g.name)
		        FROM public.portal_group_members m
		        JOIN public.portal_access_groups g ON g.id = m.group_id
		        WHERE m.user_id = u.id) AS access_groups
		FROM public.user_info u
		ORDER BY u.id ASC`
	const qFallback = `
		SELECT id, status, login, firstname, surname, lastname, ofo, user_group, phone, email,
		       (auth = true AND last_activity IS NOT NULL AND last_activity > now() - interval '24 hours') AS auth,
		       to_char(last_activity AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"') AS last_activity,
		       avatar_url, role, ''::text AS access_groups
		FROM public.user_info
		ORDER BY id ASC`

	rows, err := h.Pool.Query(r.Context(), qWithGroups)
	if err != nil {
		rows, err = h.Pool.Query(r.Context(), qFallback)
		if err != nil {
			httpx.Fail(w, http.StatusInternalServerError, "Ошибка подключения к БД")
			return
		}
	}
	defer rows.Close()

	items := []map[string]any{}
	for rows.Next() {
		var (
			id                                  int64
			status, login                       any
			firstname, surname, lastname        any
			ofo, userGroup, phone, email        any
			authActive                          bool
			lastActivity, avatarURL, role, aggs any
		)
		if err := rows.Scan(
			&id, &status, &login, &firstname, &surname, &lastname, &ofo, &userGroup,
			&phone, &email, &authActive, &lastActivity, &avatarURL, &role, &aggs,
		); err != nil {
			httpx.Fail(w, http.StatusInternalServerError, "Ошибка подключения к БД: "+err.Error())
			return
		}
		authStr := ""
		if authActive {
			authStr = "1" // PHP (string)$bool often "1"/"" depending on driver; frontend checks truthy
		}
		items = append(items, map[string]any{
			"id":            id,
			"status":        anyToString(status),
			"login":         anyToString(login),
			"firstname":     anyToString(firstname),
			"surname":       anyToString(surname),
			"lastname":      anyToString(lastname),
			"ofo":           anyToString(ofo),
			"user_group":    anyToString(userGroup),
			"access_groups": anyToString(aggs),
			"phone":         anyToString(phone),
			"email":         anyToString(email),
			"auth":          authStr,
			"last_activity": anyToString(lastActivity),
			"avatar_url":    anyToString(avatarURL),
			"role":          anyToString(role),
		})
	}
	if err := rows.Err(); err != nil {
		httpx.Fail(w, http.StatusInternalServerError, "Ошибка подключения к БД")
		return
	}
	httpx.OK(w, items, "OK")
}

func anyToString(v any) string {
	if v == nil {
		return ""
	}
	switch t := v.(type) {
	case string:
		return t
	case []byte:
		return string(t)
	case bool:
		if t {
			return "1"
		}
		return ""
	default:
		return fmt.Sprint(t)
	}
}

func (h *Users) put(w http.ResponseWriter, r *http.Request) {
	id, ok := queryID(r)
	if !ok {
		httpx.Fail(w, http.StatusBadRequest, "Укажите ?id=...")
		return
	}

	var d map[string]any
	if err := httpx.DecodeJSON(r, &d); err != nil {
		httpx.Fail(w, http.StatusBadRequest, "Некорректный JSON")
		return
	}

	var exists int64
	err := h.Pool.QueryRow(r.Context(), `SELECT id FROM public.user_info WHERE id = $1`, id).Scan(&exists)
	if err != nil {
		httpx.Fail(w, http.StatusNotFound, "Пользователь не найден")
		return
	}

	updatable := []string{"status", "login", "password", "firstname", "surname", "lastname", "ofo", "user_group", "phone", "email", "auth", "avatar_url", "role"}
	setParts := []string{}
	args := []any{id}
	argN := 2
	for _, field := range updatable {
		v, has := d[field]
		if !has {
			continue
		}
		if field == "password" {
			// Пустая строка = «не менять пароль»: в списке пользователей пароль
			// больше не отдаётся, поэтому форма редактирования присылает пустое поле,
			// когда администратор пароль не трогал. Раньше это затирало пароль.
			plain, _ := v.(string)
			if strings.TrimSpace(plain) == "" {
				continue
			}
			hashed, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.DefaultCost)
			if err != nil {
				httpx.Fail(w, http.StatusInternalServerError, "Не удалось сохранить пароль")
				return
			}
			v = string(hashed)
		}
		setParts = append(setParts, fmt.Sprintf("%s = $%d", field, argN))
		args = append(args, v)
		argN++
	}
	if len(setParts) > 0 {
		sql := "UPDATE public.user_info SET " + strings.Join(setParts, ", ") + " WHERE id = $1"
		if _, err := h.Pool.Exec(r.Context(), sql, args...); err != nil {
			httpx.Fail(w, http.StatusInternalServerError, "Ошибка подключения к БД")
			return
		}
	}

	var (
		outID                        int64
		status, login                any
		firstname, surname, lastname any
		ofo, userGroup, phone, email any
		authVal                      bool
		avatarURL, role              any
	)
	err = h.Pool.QueryRow(r.Context(), `
		SELECT id, status, login, firstname, surname, lastname, ofo, user_group, phone, email, auth, avatar_url, role
		FROM public.user_info WHERE id = $1`, id).Scan(
		&outID, &status, &login, &firstname, &surname, &lastname, &ofo, &userGroup,
		&phone, &email, &authVal, &avatarURL, &role,
	)
	if err != nil {
		httpx.Fail(w, http.StatusInternalServerError, "Ошибка подключения к БД")
		return
	}

	httpx.OK(w, map[string]any{
		"id":         outID,
		"status":     status,
		"login":      login,
		"firstname":  firstname,
		"surname":    surname,
		"lastname":   lastname,
		"ofo":        ofo,
		"user_group": userGroup,
		"phone":      phone,
		"email":      email,
		"auth":       authVal,
		"avatar_url": anyToString(avatarURL),
		"role":       anyToString(role),
	}, "Пользователь обновлен")
}
