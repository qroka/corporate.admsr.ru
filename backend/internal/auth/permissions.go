package auth

import (
	"context"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

var PortalSections = []string{
	"news", "events", "gallery", "courses", "tests", "absence_journal", "birthdays",
}

var CourseCategories = []string{
	"Кадровая деятельность", "Безопасность",
}

func UserSections(ctx context.Context, pool *pgxpool.Pool, u *User) []string {
	if IsAdmin(u) {
		out := make([]string, len(PortalSections))
		copy(out, PortalSections)
		return out
	}
	if u == nil || u.ID <= 0 {
		return []string{}
	}
	rows, err := pool.Query(ctx, `
		SELECT DISTINCT p.section_key
		FROM public.portal_group_members m
		JOIN public.portal_group_permissions p ON p.group_id = m.group_id
		WHERE m.user_id = $1`, u.ID)
	if err != nil {
		return []string{}
	}
	defer rows.Close()

	allowed := map[string]struct{}{}
	for _, s := range PortalSections {
		allowed[s] = struct{}{}
	}
	var out []string
	seen := map[string]struct{}{}
	for rows.Next() {
		var key string
		if err := rows.Scan(&key); err != nil {
			continue
		}
		if _, ok := allowed[key]; ok {
			if _, dup := seen[key]; !dup {
				seen[key] = struct{}{}
				out = append(out, key)
			}
		}
	}
	if out == nil {
		out = []string{}
	}
	return out
}

func UserCourseCategories(ctx context.Context, pool *pgxpool.Pool, u *User) []string {
	if IsAdmin(u) {
		out := make([]string, len(CourseCategories))
		copy(out, CourseCategories)
		return out
	}
	if !CanEditSection(ctx, pool, u, "courses") {
		return []string{}
	}
	if u == nil || u.ID <= 0 {
		return []string{}
	}
	rows, err := pool.Query(ctx, `
		SELECT DISTINCT c.category_key
		FROM public.portal_group_members m
		JOIN public.portal_group_permissions p
		  ON p.group_id = m.group_id AND p.section_key = 'courses'
		JOIN public.portal_group_course_categories c ON c.group_id = m.group_id
		WHERE m.user_id = $1`, u.ID)
	if err != nil {
		out := make([]string, len(CourseCategories))
		copy(out, CourseCategories)
		return out
	}
	defer rows.Close()

	allowed := map[string]struct{}{}
	for _, c := range CourseCategories {
		allowed[c] = struct{}{}
	}
	var out []string
	seen := map[string]struct{}{}
	for rows.Next() {
		var key string
		if err := rows.Scan(&key); err != nil {
			continue
		}
		if _, ok := allowed[key]; ok {
			if _, dup := seen[key]; !dup {
				seen[key] = struct{}{}
				out = append(out, key)
			}
		}
	}
	if out == nil {
		out = []string{}
	}
	return out
}

func CanEditCourseCategory(ctx context.Context, pool *pgxpool.Pool, u *User, category *string) bool {
	if IsAdmin(u) {
		return true
	}
	if !CanEditSection(ctx, pool, u, "courses") {
		return false
	}
	if category == nil || strings.TrimSpace(*category) == "" {
		return false
	}
	cat := strings.TrimSpace(*category)
	for _, c := range UserCourseCategories(ctx, pool, u) {
		if c == cat {
			return true
		}
	}
	return false
}

func CanEditSection(ctx context.Context, pool *pgxpool.Pool, u *User, section string) bool {
	ok := false
	for _, s := range PortalSections {
		if s == section {
			ok = true
			break
		}
	}
	if !ok {
		return false
	}
	if IsAdmin(u) {
		return true
	}
	for _, s := range UserSections(ctx, pool, u) {
		if s == section {
			return true
		}
	}
	return false
}
