package handlers

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"

	"corporate.admsr.ru/backend/internal/auth"
	"corporate.admsr.ru/backend/internal/config"
	"corporate.admsr.ru/backend/internal/httpx"
)

type AuthHandlers struct {
	Pool   *pgxpool.Pool
	Auth   *auth.Service
	Config config.Config
}

type loginBody struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

type idBody struct {
	ID int64 `json:"id"`
}

func (h *AuthHandlers) Login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httpx.MethodNotAllowed(w)
		return
	}

	var body loginBody
	if err := httpx.DecodeJSON(r, &body); err != nil {
		httpx.Fail(w, http.StatusBadRequest, "Некорректный JSON")
		return
	}
	login := strings.TrimSpace(body.Login)
	password := body.Password
	if login == "" || password == "" {
		httpx.Fail(w, http.StatusBadRequest, "Введите логин и пароль")
		return
	}

	ctx := r.Context()
	user, err := h.findLocalUser(ctx, login)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		httpx.Fail(w, http.StatusInternalServerError, "Ошибка подключения к БД")
		return
	}
	if errors.Is(err, pgx.ErrNoRows) {
		user, err = h.asuLookupAndCreate(ctx, login, password)
		if err != nil || user == nil {
			httpx.WriteJSON(w, http.StatusUnauthorized, map[string]any{
				"success": false,
				"message": "Неверный логин или пароль",
			})
			return
		}
	}

	if !passwordOK(password, user.Password) {
		httpx.WriteJSON(w, http.StatusUnauthorized, map[string]any{
			"success": false,
			"message": "Неверный логин или пароль",
		})
		return
	}
	if !user.Status {
		httpx.WriteJSON(w, http.StatusForbidden, map[string]any{
			"success": false,
			"message": "Учётная запись отключена. Обратитесь к администратору",
		})
		return
	}

	_, _ = h.Pool.Exec(ctx, `UPDATE public.user_info SET auth = true, last_activity = now() WHERE id = $1`, user.ID)

	isAdmin := user.UserGroup == "admin"
	var sessionToken *string
	sections := []string{}
	courseCategories := []string{}

	tok, err := h.Auth.CreateSession(ctx, w, r, user.ID)
	if err == nil {
		sessionToken = &tok
		au := &auth.User{ID: user.ID, UserGroup: user.UserGroup}
		sections = auth.UserSections(ctx, h.Pool, au)
		courseCategories = auth.UserCourseCategories(ctx, h.Pool, au)
	} else {
		if isAdmin {
			sections = append([]string{}, auth.PortalSections...)
			courseCategories = append([]string{}, auth.CourseCategories...)
		}
	}

	fioParts := []string{}
	for _, p := range []string{user.Surname, user.Firstname, user.Lastname} {
		if strings.TrimSpace(p) != "" {
			fioParts = append(fioParts, p)
		}
	}

	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"success": true,
		"user": map[string]any{
			"id":               user.ID,
			"fio":              strings.Join(fioParts, " "),
			"ofo":              user.OFO,
			"user_group":       user.UserGroup,
			"role":             user.Role,
			"sessionToken":     sessionToken,
			"isAdmin":          isAdmin,
			"sections":         sections,
			"courseCategories": courseCategories,
		},
	})
}

func (h *AuthHandlers) Logout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httpx.MethodNotAllowed(w)
		return
	}
	var body idBody
	_ = httpx.DecodeJSON(r, &body)

	ctx := r.Context()
	token := auth.ExtractToken(r)
	uidFromSession, ok := h.Auth.SessionUserID(ctx, token)
	h.Auth.RevokeSession(ctx, w, token)

	uid := body.ID
	if uid <= 0 && ok {
		uid = uidFromSession
	}
	if uid > 0 {
		_, _ = h.Pool.Exec(ctx, `UPDATE public.user_info SET auth = false WHERE id = $1`, uid)
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"success": true})
}

func (h *AuthHandlers) CheckAuth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httpx.MethodNotAllowed(w)
		return
	}
	var body idBody
	if err := httpx.DecodeJSON(r, &body); err != nil || body.ID <= 0 {
		httpx.Fail(w, http.StatusBadRequest, "Некорректный id")
		return
	}

	var ok bool
	err := h.Pool.QueryRow(r.Context(), `
		SELECT (auth = true AND last_activity IS NOT NULL AND last_activity > now() - interval '24 hours')
		FROM public.user_info WHERE id = $1 AND status = true LIMIT 1`, body.ID).Scan(&ok)
	if err != nil {
		httpx.WriteJSON(w, http.StatusOK, map[string]any{"success": true, "auth": false})
		return
	}
	if !ok {
		_, _ = h.Pool.Exec(r.Context(), `UPDATE public.user_info SET auth = false WHERE id = $1 AND auth = true`, body.ID)
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"success": true, "auth": ok})
}

func (h *AuthHandlers) Heartbeat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httpx.MethodNotAllowed(w)
		return
	}
	var body idBody
	if err := httpx.DecodeJSON(r, &body); err != nil || body.ID <= 0 {
		httpx.Fail(w, http.StatusBadRequest, "Некорректный id")
		return
	}
	tag, err := h.Pool.Exec(r.Context(),
		`UPDATE public.user_info SET last_activity = now() WHERE id = $1 AND auth = true AND status = true`,
		body.ID)
	if err != nil {
		httpx.Fail(w, http.StatusInternalServerError, "Ошибка подключения к БД")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"success": true, "updated": tag.RowsAffected() > 0})
}

func (h *AuthHandlers) SessionBootstrap(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httpx.MethodNotAllowed(w)
		return
	}
	var body idBody
	_ = httpx.DecodeJSON(r, &body)

	// Личность берётся ТОЛЬКО из действующей сессии (cookie corp_session / Bearer).
	// id из тела запроса не является доказательством личности: раньше он позволял
	// выпустить токен для любого активного пользователя без единого секрета.
	cur, err := h.Auth.CurrentUser(r.Context(), r)
	if err != nil {
		httpx.Fail(w, http.StatusUnauthorized, "Сессия портала недействительна. Войдите снова.")
		return
	}
	if body.ID > 0 && body.ID != cur.ID {
		httpx.Fail(w, http.StatusForbidden, "Недостаточно прав")
		return
	}

	id := cur.ID
	userGroup := cur.UserGroup

	token, err := h.Auth.CreateSession(r.Context(), w, r, id)
	if err != nil {
		httpx.Fail(w, http.StatusInternalServerError, "Не удалось создать сессию (проверьте миграцию V4)")
		return
	}
	httpx.OK(w, map[string]any{
		"sessionToken": token,
		"userGroup":    userGroup,
	}, "OK")
}

type localUser struct {
	ID        int64
	Status    bool
	Password  string
	Firstname string
	Surname   string
	Lastname  string
	OFO       any
	UserGroup string
	Role      string
}

func (h *AuthHandlers) findLocalUser(ctx context.Context, login string) (*localUser, error) {
	var u localUser
	err := h.Pool.QueryRow(ctx, `
		SELECT id, status, password, firstname, surname, lastname, ofo, user_group, COALESCE(role, '')
		FROM public.user_info WHERE LOWER(login) = LOWER($1) LIMIT 1`, login).Scan(
		&u.ID, &u.Status, &u.Password, &u.Firstname, &u.Surname, &u.Lastname, &u.OFO, &u.UserGroup, &u.Role,
	)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func passwordOK(plain, stored string) bool {
	if stored == plain {
		return true
	}
	hash := stored
	if strings.HasPrefix(hash, "$2y$") {
		hash = "$2a$" + hash[4:]
	}
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain)) == nil
}

func (h *AuthHandlers) asuLookupAndCreate(ctx context.Context, login, password string) (*localUser, error) {
	payload, _ := json.Marshal(map[string]string{"login": login, "password": password})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, h.Config.ASUURL, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Host = h.Config.ASUHost
	req.Header.Set("X-Sync-Secret", h.Config.ASUSecret)

	client := &http.Client{
		Timeout: 5 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, //nolint:gosec // mirrors PHP CURLOPT_SSL_VERIFYPEER=false
		},
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)

	var data struct {
		Success bool `json:"success"`
		User    *struct {
			ID       any    `json:"id"`
			Login    string `json:"login"`
			Password string `json:"password"`
			Name     string `json:"name"`
			Phone    string `json:"phone"`
			Email    string `json:"email"`
		} `json:"user"`
	}
	if err := json.Unmarshal(raw, &data); err != nil || !data.Success || data.User == nil {
		return nil, errors.New("asu lookup failed")
	}
	u := data.User
	parts := strings.SplitN(strings.TrimSpace(u.Name), " ", 3)
	surname, firstname, lastname := "", "", ""
	if len(parts) > 0 {
		surname = parts[0]
	}
	if len(parts) > 1 {
		firstname = parts[1]
	}
	if len(parts) > 2 {
		lastname = parts[2]
	}

	id := asInt64(u.ID)
	_, err = h.Pool.Exec(ctx, `
		INSERT INTO public.user_info
			(id, status, login, password, firstname, surname, lastname, phone, email, ofo, user_group, role, auth)
		VALUES ($1, true, $2, $3, $4, $5, $6, $7, $8, -1, 'user', '', false)
		ON CONFLICT (id) DO NOTHING`,
		id, nullStr(u.Login), nullStr(u.Password), firstname, surname, lastname, u.Phone, u.Email,
	)
	if err != nil {
		return nil, err
	}

	var ofo any
	var userGroup, role string
	var status bool
	err = h.Pool.QueryRow(ctx, `SELECT ofo, user_group, role, status FROM public.user_info WHERE id = $1`, id).
		Scan(&ofo, &userGroup, &role, &status)
	if err != nil {
		ofo = -1
		userGroup = "user"
		role = ""
		status = true
	}

	return &localUser{
		ID:        id,
		Status:    status,
		Password:  u.Password,
		Firstname: firstname,
		Surname:   surname,
		Lastname:  lastname,
		OFO:       ofo,
		UserGroup: userGroup,
		Role:      role,
	}, nil
}

func nullStr(s string) string { return s }

func asInt64(v any) int64 {
	switch t := v.(type) {
	case float64:
		return int64(t)
	case int64:
		return t
	case int:
		return int64(t)
	case json.Number:
		n, _ := t.Int64()
		return n
	case string:
		n, _ := strconv.ParseInt(strings.TrimSpace(t), 10, 64)
		return n
	default:
		b, _ := json.Marshal(v)
		var n int64
		_ = json.Unmarshal(b, &n)
		return n
	}
}
