package handlers

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"corporate.admsr.ru/backend/internal/auth"
	"corporate.admsr.ru/backend/internal/httpx"
)

type Portal struct {
	Pool *pgxpool.Pool
	Auth *auth.Service
}

func (h *Portal) Groups(w http.ResponseWriter, r *http.Request) {
	if _, ok := requireAdmin(w, r, h.Auth); !ok {
		return
	}

	switch r.Method {
	case http.MethodGet:
		h.groupsGet(w, r)
	case http.MethodPost:
		h.groupsPost(w, r)
	case http.MethodPut:
		h.groupsPut(w, r)
	case http.MethodDelete:
		h.groupsDelete(w, r)
	default:
		httpx.Fail(w, http.StatusMethodNotAllowed, "Метод не поддерживается")
	}
}

func (h *Portal) MyPermissions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httpx.Fail(w, http.StatusMethodNotAllowed, "Метод не поддерживается")
		return
	}
	u, ok := requireUser(w, r, h.Auth)
	if !ok {
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"success": true,
		"data": map[string]any{
			"isAdmin":          auth.IsAdmin(u),
			"sections":         auth.UserSections(r.Context(), h.Pool, u),
			"courseCategories": auth.UserCourseCategories(r.Context(), h.Pool, u),
		},
	})
}

func (h *Portal) groupsGet(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if idStr := r.URL.Query().Get("id"); idStr != "" {
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil || id <= 0 {
			portalFail(w, http.StatusBadRequest, "Не передан id")
			return
		}
		g, err := h.fetchGroup(ctx, id)
		if err != nil {
			if portalMigrationErr(w, err) {
				return
			}
			portalFail(w, http.StatusInternalServerError, "Ошибка сервера")
			return
		}
		if g == nil {
			portalFail(w, http.StatusNotFound, "Группа не найдена")
			return
		}
		portalOK(w, g)
		return
	}

	rows, err := h.Pool.Query(ctx, `
		SELECT g.id, g.name, g.description, g.created_at, g.updated_at,
		       (SELECT COUNT(*) FROM public.portal_group_members m WHERE m.group_id = g.id) AS member_count,
		       (SELECT string_agg(p.section_key, ',' ORDER BY p.section_key)
		        FROM public.portal_group_permissions p WHERE p.group_id = g.id) AS permissions,
		       (SELECT string_agg(c.category_key, ',' ORDER BY c.category_key)
		        FROM public.portal_group_course_categories c WHERE c.group_id = g.id) AS course_categories
		FROM public.portal_access_groups g
		ORDER BY g.name`)
	if err != nil {
		if portalMigrationErr(w, err) {
			return
		}
		portalFail(w, http.StatusInternalServerError, "Ошибка сервера")
		return
	}
	defer rows.Close()

	list := []map[string]any{}
	for rows.Next() {
		var id, memberCount int64
		var name string
		var description, permissions, courseCategories *string
		var createdAt, updatedAt any
		if err := rows.Scan(&id, &name, &description, &createdAt, &updatedAt, &memberCount, &permissions, &courseCategories); err != nil {
			portalFail(w, http.StatusInternalServerError, "Ошибка сервера")
			return
		}
		desc := ""
		if description != nil {
			desc = *description
		}
		list = append(list, mapGroupListRow(id, name, desc, memberCount, permissions, courseCategories, createdAt, updatedAt))
	}
	portalOK(w, list)
}

func (h *Portal) groupsPost(w http.ResponseWriter, r *http.Request) {
	body, err := portalBody(r)
	if err != nil {
		portalFail(w, http.StatusBadRequest, "Некорректный JSON")
		return
	}
	name := strings.TrimSpace(strVal(body["name"]))
	if name == "" {
		portalFail(w, http.StatusBadRequest, "Укажите название группы")
		return
	}
	description := strings.TrimSpace(strVal(body["description"]))
	permissions := normalizePermissions(body["permissions"])
	courseCategories := []string{}
	if containsStr(permissions, "courses") {
		courseCategories = normalizeCourseCategories(body["courseCategories"])
		if len(courseCategories) == 0 {
			portalFail(w, http.StatusBadRequest, "Для права «Курсы» выберите хотя бы одну категорию")
			return
		}
	}
	memberIDs := normalizeMemberIDs(body["memberIds"])

	ctx := r.Context()
	tx, err := h.Pool.Begin(ctx)
	if err != nil {
		portalFail(w, http.StatusInternalServerError, "Ошибка сервера")
		return
	}
	defer tx.Rollback(ctx)

	var newID int64
	err = tx.QueryRow(ctx, `
		INSERT INTO public.portal_access_groups (name, description)
		VALUES ($1, $2) RETURNING id`, name, description).Scan(&newID)
	if err != nil {
		if isUniqueViolation(err) {
			portalFail(w, http.StatusConflict, "Группа с таким названием уже есть")
			return
		}
		if portalMigrationErr(w, err) {
			return
		}
		portalFail(w, http.StatusInternalServerError, "Ошибка сервера")
		return
	}

	if err := replacePermissions(ctx, tx, newID, permissions); err != nil {
		portalFail(w, http.StatusInternalServerError, "Ошибка сервера")
		return
	}
	if err := replaceCourseCategories(ctx, tx, newID, courseCategories); err != nil {
		if portalMigrationErr(w, err) {
			return
		}
		portalFail(w, http.StatusInternalServerError, "Ошибка сервера")
		return
	}
	if err := replaceMembers(ctx, tx, newID, memberIDs); err != nil {
		portalFail(w, http.StatusInternalServerError, "Ошибка сервера")
		return
	}
	if err := tx.Commit(ctx); err != nil {
		portalFail(w, http.StatusInternalServerError, "Ошибка сервера")
		return
	}

	g, err := h.fetchGroup(ctx, newID)
	if err != nil {
		portalFail(w, http.StatusInternalServerError, "Ошибка сервера")
		return
	}
	portalOK(w, g)
}

func (h *Portal) groupsPut(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.URL.Query().Get("id"), 10, 64)
	if err != nil || id <= 0 {
		portalFail(w, http.StatusBadRequest, "Не передан id")
		return
	}

	ctx := r.Context()
	existing, err := h.fetchGroup(ctx, id)
	if err != nil {
		if portalMigrationErr(w, err) {
			return
		}
		portalFail(w, http.StatusInternalServerError, "Ошибка сервера")
		return
	}
	if existing == nil {
		portalFail(w, http.StatusNotFound, "Группа не найдена")
		return
	}

	body, err := portalBody(r)
	if err != nil {
		portalFail(w, http.StatusBadRequest, "Некорректный JSON")
		return
	}

	name := strVal(existing["name"])
	if _, has := body["name"]; has {
		name = strings.TrimSpace(strVal(body["name"]))
	}
	if name == "" {
		portalFail(w, http.StatusBadRequest, "Укажите название группы")
		return
	}

	description := strVal(existing["description"])
	if _, has := body["description"]; has {
		description = strings.TrimSpace(strVal(body["description"]))
	}

	permissions, _ := existing["permissions"].([]string)
	if _, has := body["permissions"]; has {
		permissions = normalizePermissions(body["permissions"])
	}

	courseCategories, _ := existing["courseCategories"].([]string)
	if _, has := body["courseCategories"]; has {
		courseCategories = normalizeCourseCategories(body["courseCategories"])
	}
	if !containsStr(permissions, "courses") {
		courseCategories = []string{}
	} else if len(courseCategories) == 0 {
		portalFail(w, http.StatusBadRequest, "Для права «Курсы» выберите хотя бы одну категорию")
		return
	}

	memberIDs, _ := existing["memberIds"].([]int64)
	if _, has := body["memberIds"]; has {
		memberIDs = normalizeMemberIDs(body["memberIds"])
	}

	tx, err := h.Pool.Begin(ctx)
	if err != nil {
		portalFail(w, http.StatusInternalServerError, "Ошибка сервера")
		return
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `
		UPDATE public.portal_access_groups
		SET name = $1, description = $2, updated_at = now()
		WHERE id = $3`, name, description, id)
	if err != nil {
		if isUniqueViolation(err) {
			portalFail(w, http.StatusConflict, "Группа с таким названием уже есть")
			return
		}
		portalFail(w, http.StatusInternalServerError, "Ошибка сервера")
		return
	}

	if err := replacePermissions(ctx, tx, id, permissions); err != nil {
		portalFail(w, http.StatusInternalServerError, "Ошибка сервера")
		return
	}
	if err := replaceCourseCategories(ctx, tx, id, courseCategories); err != nil {
		if portalMigrationErr(w, err) {
			return
		}
		portalFail(w, http.StatusInternalServerError, "Ошибка сервера")
		return
	}
	if err := replaceMembers(ctx, tx, id, memberIDs); err != nil {
		portalFail(w, http.StatusInternalServerError, "Ошибка сервера")
		return
	}
	if err := tx.Commit(ctx); err != nil {
		portalFail(w, http.StatusInternalServerError, "Ошибка сервера")
		return
	}

	g, err := h.fetchGroup(ctx, id)
	if err != nil {
		portalFail(w, http.StatusInternalServerError, "Ошибка сервера")
		return
	}
	portalOK(w, g)
}

func (h *Portal) groupsDelete(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.URL.Query().Get("id"), 10, 64)
	if err != nil || id <= 0 {
		portalFail(w, http.StatusBadRequest, "Не передан id")
		return
	}

	var deleted int64
	err = h.Pool.QueryRow(r.Context(), `
		DELETE FROM public.portal_access_groups WHERE id = $1 RETURNING id`, id).Scan(&deleted)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			portalFail(w, http.StatusNotFound, "Группа не найдена")
			return
		}
		if portalMigrationErr(w, err) {
			return
		}
		portalFail(w, http.StatusInternalServerError, "Ошибка сервера")
		return
	}
	portalOK(w, map[string]any{"id": id})
}

func (h *Portal) fetchGroup(ctx context.Context, id int64) (map[string]any, error) {
	var name string
	var description *string
	var createdAt, updatedAt any
	err := h.Pool.QueryRow(ctx, `
		SELECT id, name, description, created_at, updated_at
		FROM public.portal_access_groups WHERE id = $1`, id).
		Scan(&id, &name, &description, &createdAt, &updatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	desc := ""
	if description != nil {
		desc = *description
	}

	permRows, err := h.Pool.Query(ctx, `
		SELECT section_key FROM public.portal_group_permissions
		WHERE group_id = $1 ORDER BY section_key`, id)
	if err != nil {
		return nil, err
	}
	permissions := scanStringRows(permRows)

	courseCategories := []string{}
	catRows, err := h.Pool.Query(ctx, `
		SELECT category_key FROM public.portal_group_course_categories
		WHERE group_id = $1 ORDER BY category_key`, id)
	if err == nil {
		courseCategories = scanStringRows(catRows)
	}

	memRows, err := h.Pool.Query(ctx, `
		SELECT m.user_id, u.surname, u.firstname, u.lastname, u.login
		FROM public.portal_group_members m
		LEFT JOIN public.user_info u ON u.id = m.user_id
		WHERE m.group_id = $1
		ORDER BY u.surname NULLS LAST, u.firstname NULLS LAST, m.user_id`, id)
	if err != nil {
		return nil, err
	}
	defer memRows.Close()

	memberIDs := []int64{}
	members := []map[string]any{}
	for memRows.Next() {
		var uid int64
		var surname, firstname, lastname, login *string
		if err := memRows.Scan(&uid, &surname, &firstname, &lastname, &login); err != nil {
			return nil, err
		}
		memberIDs = append(memberIDs, uid)
		parts := filterNonEmpty([]string{strPtr(surname), strPtr(firstname), strPtr(lastname)})
		fio := strings.Join(parts, " ")
		if fio == "" {
			fio = "ID " + strconv.FormatInt(uid, 10)
		}
		lg := ""
		if login != nil {
			lg = *login
		}
		members = append(members, map[string]any{
			"id":    uid,
			"fio":   fio,
			"login": lg,
		})
	}

	return map[string]any{
		"id":               id,
		"name":             name,
		"description":      desc,
		"permissions":      permissions,
		"courseCategories": courseCategories,
		"memberIds":        memberIDs,
		"members":          members,
		"memberCount":      len(memberIDs),
		"createdAt":        createdAt,
		"updatedAt":        updatedAt,
	}, nil
}

func mapGroupListRow(id int64, name, description string, memberCount int64, permissions, courseCategories *string, createdAt, updatedAt any) map[string]any {
	perms := splitCSV(permissions)
	cats := splitCSV(courseCategories)
	return map[string]any{
		"id":               id,
		"name":             name,
		"description":      description,
		"memberCount":      memberCount,
		"permissions":      perms,
		"courseCategories": cats,
		"createdAt":        createdAt,
		"updatedAt":        updatedAt,
	}
}

func portalBody(r *http.Request) (map[string]any, error) {
	var body map[string]any
	if err := httpx.DecodeJSON(r, &body); err != nil {
		return nil, err
	}
	if body == nil {
		return map[string]any{}, nil
	}
	return body, nil
}

func normalizePermissions(raw any) []string {
	arr, ok := raw.([]any)
	if !ok {
		return []string{}
	}
	allowed := map[string]struct{}{}
	for _, s := range auth.PortalSections {
		allowed[s] = struct{}{}
	}
	seen := map[string]struct{}{}
	out := []string{}
	for _, item := range arr {
		key := strings.TrimSpace(strVal(item))
		if key == "" {
			continue
		}
		if _, ok := allowed[key]; !ok {
			continue
		}
		if _, dup := seen[key]; dup {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, key)
	}
	return out
}

func normalizeCourseCategories(raw any) []string {
	arr, ok := raw.([]any)
	if !ok {
		return []string{}
	}
	allowed := map[string]struct{}{}
	for _, c := range auth.CourseCategories {
		allowed[c] = struct{}{}
	}
	seen := map[string]struct{}{}
	out := []string{}
	for _, item := range arr {
		key := ""
		if m, ok := item.(map[string]any); ok {
			key = strings.TrimSpace(strVal(m["value"]))
			if key == "" {
				key = strings.TrimSpace(strVal(m["label"]))
			}
		} else {
			key = strings.TrimSpace(strVal(item))
		}
		if key == "" {
			continue
		}
		if _, ok := allowed[key]; !ok {
			continue
		}
		if _, dup := seen[key]; dup {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, key)
	}
	return out
}

func normalizeMemberIDs(raw any) []int64 {
	arr, ok := raw.([]any)
	if !ok {
		return []int64{}
	}
	seen := map[int64]struct{}{}
	out := []int64{}
	for _, item := range arr {
		id, err := strconv.ParseInt(strVal(item), 10, 64)
		if err != nil || id <= 0 {
			continue
		}
		if _, dup := seen[id]; dup {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}

type querier interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

func replacePermissions(ctx context.Context, q querier, groupID int64, permissions []string) error {
	if _, err := q.Exec(ctx, `DELETE FROM public.portal_group_permissions WHERE group_id = $1`, groupID); err != nil {
		return err
	}
	for _, s := range permissions {
		if _, err := q.Exec(ctx, `
			INSERT INTO public.portal_group_permissions (group_id, section_key) VALUES ($1, $2)`, groupID, s); err != nil {
			return err
		}
	}
	return nil
}

func replaceCourseCategories(ctx context.Context, q querier, groupID int64, categories []string) error {
	if _, err := q.Exec(ctx, `DELETE FROM public.portal_group_course_categories WHERE group_id = $1`, groupID); err != nil {
		return err
	}
	for _, c := range categories {
		if _, err := q.Exec(ctx, `
			INSERT INTO public.portal_group_course_categories (group_id, category_key) VALUES ($1, $2)`, groupID, c); err != nil {
			return err
		}
	}
	return nil
}

func replaceMembers(ctx context.Context, q querier, groupID int64, memberIDs []int64) error {
	if _, err := q.Exec(ctx, `DELETE FROM public.portal_group_members WHERE group_id = $1`, groupID); err != nil {
		return err
	}
	for _, uid := range memberIDs {
		if _, err := q.Exec(ctx, `
			INSERT INTO public.portal_group_members (group_id, user_id) VALUES ($1, $2)`, groupID, uid); err != nil {
			return err
		}
	}
	return nil
}

func scanStringRows(rows pgx.Rows) []string {
	defer rows.Close()
	out := []string{}
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			continue
		}
		out = append(out, s)
	}
	return out
}

func splitCSV(v *string) []string {
	if v == nil || *v == "" {
		return []string{}
	}
	parts := strings.Split(*v, ",")
	out := []string{}
	for _, p := range parts {
		if s := strings.TrimSpace(p); s != "" {
			out = append(out, s)
		}
	}
	return out
}

func containsStr(list []string, target string) bool {
	for _, s := range list {
		if s == target {
			return true
		}
	}
	return false
}

func strPtr(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

func portalOK(w http.ResponseWriter, data any) {
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"success": true, "data": data})
}

func portalFail(w http.ResponseWriter, status int, msg string) {
	httpx.WriteJSON(w, status, map[string]any{"success": false, "message": msg, "data": nil})
}

func portalMigrationErr(w http.ResponseWriter, err error) bool {
	msg := strings.ToLower(err.Error())
	if strings.Contains(msg, "portal_services") {
		portalFail(w, http.StatusServiceUnavailable, "Нужна миграция V10__portal_services.sql")
		return true
	}
	if strings.Contains(msg, "portal_access_groups") ||
		strings.Contains(msg, "portal_group_course_categories") ||
		strings.Contains(msg, "does not exist") {
		portalFail(w, http.StatusServiceUnavailable, "Нужны миграции V5__portal_access_groups.sql и V6__portal_group_course_categories.sql")
		return true
	}
	return false
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505" || strings.Contains(pgErr.Message, "portal_access_groups_name")
	}
	return strings.Contains(err.Error(), "portal_access_groups_name")
}
