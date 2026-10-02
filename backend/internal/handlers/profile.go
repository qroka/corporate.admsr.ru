package handlers

import (
	"context"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"corporate.admsr.ru/backend/internal/auth"
	"corporate.admsr.ru/backend/internal/courses"
	"corporate.admsr.ru/backend/internal/httpx"
	"corporate.admsr.ru/backend/internal/media"
)

type Profile struct {
	Pool *pgxpool.Pool
	Auth *auth.Service
	// Birthdays — источник дня рождения для страницы профиля (?view=page).
	Birthdays *Birthdays
	// UploadDir — корень загрузок: отсюда удаляется прежний загруженный аватар.
	UploadDir string
}

func (h *Profile) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Карточка сотрудника — внутренние данные портала: читать может
	// авторизованный пользователь, изменять — только владелец или админ (SEC-001).
	cur, ok := requireUser(w, r, h.Auth)
	if !ok {
		return
	}
	switch r.Method {
	case http.MethodGet:
		h.get(w, r, cur.ID)
	case http.MethodPost:
		h.post(w, r, cur)
	default:
		httpx.Fail(w, http.StatusMethodNotAllowed, "Метод не поддерживается")
	}
}

func (h *Profile) get(w http.ResponseWriter, r *http.Request, viewerID int64) {
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

	data := map[string]any{
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
	}
	// Страница профиля «как стена ВКонтакте» — подразделение, день рождения,
	// «О себе», коллеги. Остальным вызовам (шапка, онбординг) это не нужно.
	if r.URL.Query().Get("view") == "page" {
		h.addPageExtras(r.Context(), data, id, ofo, surname, firstname, lastname)
		h.addPageSections(r.Context(), data, id, viewerID)
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"success": true, "data": data})
}

func (h *Profile) addPageExtras(ctx context.Context, data map[string]any, id int64, ofo, surname, firstname, lastname string) {
	data["ofoName"] = ""
	if ofoID, err := strconv.ParseInt(strings.TrimSpace(ofo), 10, 64); err == nil && ofoID > 0 {
		var name string
		if err := h.Pool.QueryRow(ctx, `SELECT name FROM public.ofo_unit WHERE id = $1`, ofoID).Scan(&name); err == nil {
			data["ofoName"] = name
		}
	}

	data["birthday"] = nil
	fio := strings.Join(strings.Fields(surname+" "+firstname+" "+lastname), " ")
	if m, d, ok := h.Birthdays.BirthdayOf(fio); ok {
		data["birthday"] = map[string]any{"month": m, "day": d}
	}

	about, interests := "", ""
	if err := h.Pool.QueryRow(ctx,
		`SELECT about, interests FROM public.profile_about WHERE user_id = $1`, id).Scan(&about, &interests); err != nil &&
		!errors.Is(err, pgx.ErrNoRows) {
		log.Printf("profile: about for user %d: %v", id, err) // без V14 страница всё равно открывается
	}
	data["about"] = about
	data["interests"] = interests

	// Коллеги — активные сотрудники того же подразделения (как «Друзья» у ВКонтакте).
	colleagues := []map[string]any{}
	var colleaguesTotal int64
	if strings.TrimSpace(ofo) != "" && strings.TrimSpace(ofo) != "-1" {
		_ = h.Pool.QueryRow(ctx,
			`SELECT COUNT(*) FROM public.user_info WHERE ofo = $1 AND id <> $2 AND status`, ofo, id).Scan(&colleaguesTotal)
		rows, err := h.Pool.Query(ctx, `
			SELECT id, COALESCE(firstname, ''), COALESCE(surname, ''), COALESCE(avatar_url, '')
			FROM public.user_info
			WHERE ofo = $1 AND id <> $2 AND status
			ORDER BY surname, firstname
			LIMIT 6`, ofo, id)
		if err == nil {
			for rows.Next() {
				var cid int64
				var fn, sn, av string
				if rows.Scan(&cid, &fn, &sn, &av) == nil {
					colleagues = append(colleagues, map[string]any{"id": cid, "firstname": fn, "surname": sn, "avatar_url": av})
				}
			}
			rows.Close()
		} else {
			log.Printf("profile: colleagues for user %d: %v", id, err)
		}
	}
	data["colleagues"] = map[string]any{"total": colleaguesTotal, "items": colleagues}
}

func (h *Profile) post(w http.ResponseWriter, r *http.Request, cur *auth.User) {
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

	// id из тела запроса — не доказательство личности: редактировать можно
	// только свою карточку, чужую — только администратору (SEC-001, IDOR).
	if id != cur.ID && !auth.IsAdmin(cur) {
		httpx.Fail(w, http.StatusForbidden, "Недостаточно прав")
		return
	}

	// ФИО, телефон и почта на портале не редактируются (ADR-041): тело запроса
	// их игнорирует. ОФО и должность — только пока не заданы (первый вход, онбординг).
	var curOFO, curRole, curAvatar string
	if err := h.Pool.QueryRow(r.Context(),
		`SELECT COALESCE(ofo, ''), COALESCE(role, ''), COALESCE(avatar_url, '') FROM public.user_info WHERE id = $1`, id).Scan(&curOFO, &curRole, &curAvatar); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			httpx.Fail(w, http.StatusNotFound, "Пользователь не найден")
			return
		}
		httpx.Fail(w, http.StatusInternalServerError, "Ошибка подключения к БД")
		return
	}
	newOFO, newRole := planWorkFields(curOFO, curRole, body)

	sets := []string{}
	args := []any{}
	add := func(col string, v any) {
		args = append(args, v)
		sets = append(sets, col+" = $"+strconv.Itoa(len(args)))
	}
	newAvatar := curAvatar
	if _, has := body["avatar_url"]; has {
		newAvatar = strings.TrimSpace(strVal(body["avatar_url"]))
		add("avatar_url", newAvatar)
	}
	if newOFO != nil {
		add("ofo", *newOFO)
	}
	if newRole != nil {
		add("role", *newRole)
	}
	if len(sets) > 0 {
		args = append(args, id)
		if _, err := h.Pool.Exec(r.Context(),
			`UPDATE public.user_info SET `+strings.Join(sets, ", ")+` WHERE id = $`+strconv.Itoa(len(args)), args...); err != nil {
			httpx.Fail(w, http.StatusInternalServerError, "Ошибка подключения к БД")
			return
		}
	}
	// Сменил загруженное фото на другой аватар — файл прежнего больше не нужен.
	if newAvatar != curAvatar {
		media.RemoveUploadedAvatar(media.ImgRoot(h.UploadDir), id, curAvatar)
	}

	// «О себе» и «Интересы» — только пришедшие поля (V14, profile_about).
	_, hasAbout := body["about"]
	_, hasInterests := body["interests"]
	if hasAbout || hasInterests {
		var about, interests *string
		if hasAbout {
			v := limitRunes(strings.TrimSpace(strVal(body["about"])), 2000)
			about = &v
		}
		if hasInterests {
			v := limitRunes(strings.TrimSpace(strVal(body["interests"])), 1000)
			interests = &v
		}
		if _, err := h.Pool.Exec(r.Context(), `
			INSERT INTO public.profile_about (user_id, about, interests, updated_at)
			VALUES ($1, COALESCE($2::text, ''), COALESCE($3::text, ''), now())
			ON CONFLICT (user_id) DO UPDATE
			SET about = COALESCE($2::text, public.profile_about.about),
			    interests = COALESCE($3::text, public.profile_about.interests),
			    updated_at = now()`,
			id, about, interests); err != nil {
			log.Printf("profile: save about for user %d: %v", id, err)
			if wallTableMissing(err) {
				httpx.Fail(w, http.StatusServiceUnavailable, "«О себе» ещё не включено на сервере: нужна миграция V14__profile_wall.sql")
				return
			}
			httpx.Fail(w, http.StatusInternalServerError, "Ошибка подключения к БД")
			return
		}
	}

	// Выбрал ОФО — сразу выдать курсы, назначенные на подразделение.
	if newOFO != nil {
		if _, err := (&courses.Service{Pool: h.Pool}).SyncStandingAssignments(r.Context(), id); err != nil {
			log.Printf("profile: sync standing assignments for user %d: %v", id, err)
		}
	}

	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"success": true,
		"data":    nil,
	})
}

func limitRunes(s string, n int) string {
	r := []rune(s)
	if len(r) > n {
		return string(r[:n])
	}
	return s
}

// ofoUnset — ОФО не задано: пусто, «-1» (так создаёт учётку синхронизация) или не положительное число.
func ofoUnset(ofo string) bool {
	n, err := strconv.ParseInt(strings.TrimSpace(ofo), 10, 64)
	return err != nil || n <= 0
}

// planWorkFields решает, что из присланных ОФО и должности можно записать.
// ОФО — только пока оно не задано и пришло положительным числом. Должность — пока
// она пуста или пока ОФО ещё не задано (тогда её выбирают вместе с ОФО).
// Остальное сервер молча не трогает: на портале это не редактируется (ADR-041).
func planWorkFields(curOFO, curRole string, body map[string]any) (ofo, role *string) {
	unset := ofoUnset(curOFO)
	if v, has := body["ofo"]; has && unset {
		if s := strings.TrimSpace(strVal(v)); !ofoUnset(s) {
			ofo = &s
		}
	}
	if v, has := body["role"]; has && (unset || strings.TrimSpace(curRole) == "") {
		s := strings.TrimSpace(strVal(v))
		role = &s
	}
	return ofo, role
}

// addPageSections — блоки страницы профиля: пройденные курсы, отсутствия,
// желания, награды. Любой блок, который не удалось прочитать (например, не
// применена V15), отдаётся пустым — страница всё равно открывается.
func (h *Profile) addPageSections(ctx context.Context, data map[string]any, id, viewerID int64) {
	// Пройденные курсы: завершённые назначения неудалённых курсов. Ссылка на
	// назначение (enrollmentId) — только владельцу: чужое назначение не открыть.
	courseItems := []map[string]any{}
	var coursesTotal int64
	if err := h.Pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM public.course_enrollments e
		JOIN public.course_versions v ON v.id = e.course_version_id
		JOIN public.course_courses c ON c.id = v.course_id
		WHERE e.user_id = $1 AND e.status = 'completed' AND c.deleted_at IS NULL`, id).Scan(&coursesTotal); err != nil {
		log.Printf("profile: completed courses count for user %d: %v", id, err)
	} else if rows, err := h.Pool.Query(ctx, `
		SELECT e.id, c.id, c.title, e.completed_at, e.final_score::float8
		FROM public.course_enrollments e
		JOIN public.course_versions v ON v.id = e.course_version_id
		JOIN public.course_courses c ON c.id = v.course_id
		WHERE e.user_id = $1 AND e.status = 'completed' AND c.deleted_at IS NULL
		ORDER BY e.completed_at DESC NULLS LAST, e.id DESC
		LIMIT 20`, id); err != nil {
		log.Printf("profile: completed courses for user %d: %v", id, err)
	} else {
		for rows.Next() {
			var enrollmentID, courseID int64
			var title string
			var completedAt *time.Time
			var score *float64
			if rows.Scan(&enrollmentID, &courseID, &title, &completedAt, &score) != nil {
				continue
			}
			item := map[string]any{"courseId": courseID, "title": title, "completedAt": nil, "score": score}
			if completedAt != nil {
				item["completedAt"] = completedAt.Format(time.RFC3339)
			}
			if id == viewerID {
				item["enrollmentId"] = enrollmentID
			}
			courseItems = append(courseItems, item)
		}
		rows.Close()
	}
	data["courses"] = map[string]any{"total": coursesTotal, "items": courseItems}

	// Журнал отсутствия — те же записи, что читает любой вошедший в /absence_journal.php?user_id=.
	absences := []map[string]any{}
	if rows, err := h.Pool.Query(ctx, `
		SELECT id, start_datetime, end_datetime, COALESCE(reason, '')
		FROM public.absence_journal WHERE user_id = $1
		ORDER BY start_datetime DESC, id DESC LIMIT 5`, id); err != nil {
		log.Printf("profile: absences for user %d: %v", id, err)
	} else {
		for rows.Next() {
			var aid int64
			var start time.Time
			var end *time.Time
			var reason string
			if rows.Scan(&aid, &start, &end, &reason) != nil {
				continue
			}
			item := map[string]any{"id": aid, "start": start.Format(time.RFC3339), "end": nil, "reason": reason, "active": end == nil}
			if end != nil {
				item["end"] = end.Format(time.RFC3339)
			}
			absences = append(absences, item)
		}
		rows.Close()
	}
	data["absences"] = absences

	data["wishes"] = loadWishes(ctx, h.Pool, id)
	data["awards"] = loadAwards(ctx, h.Pool, id)
}
