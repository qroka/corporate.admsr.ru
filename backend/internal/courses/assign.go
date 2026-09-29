package courses

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// Назначения курса (ADR-036).
//
// Назначение на пользователя — разовое. Назначения на ОФО и «всем» —
// постоянные: кроме записей, созданных в момент назначения, недостающие
// записи досоздаются для сотрудника, когда он входит на портал, выбирает
// ОФО в профиле или открывает «Моё обучение» (SyncStandingAssignments).
// Так курс получают и те, кто на момент назначения ещё не входил (учётная
// запись создаётся синхронизацией ASU или первым входом, ОФО — пусто).

// Assignment — строка course_assignments, нужная для создания записей.
type Assignment struct {
	ID           int64
	VersionID    int64
	StartsAt     *time.Time
	DeadlineAt   *time.Time
	DeadlineDays *int
}

// DBTX — общее у pgxpool.Pool и pgx.Tx.
type DBTX interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// EnrollmentDeadline — срок записи по назначению. Относительный срок
// («N дней») отсчитывается от момента выдачи курса или от даты начала, если
// она позже; иначе — общий срок назначения (может быть и в прошлом:
// опоздавший получает курс сразу просроченным, как и все по этому назначению).
func EnrollmentDeadline(now time.Time, a Assignment) *time.Time {
	if a.DeadlineDays != nil && *a.DeadlineDays > 0 {
		base := now
		if a.StartsAt != nil && a.StartsAt.After(base) {
			base = *a.StartsAt
		}
		d := base.Add(time.Duration(*a.DeadlineDays) * 24 * time.Hour)
		return &d
	}
	return a.DeadlineAt
}

// Enroll создаёт запись сотрудника по назначению. Возвращает id новой записи
// или 0, если у сотрудника уже есть действующая запись на эту версию либо
// (keepCancelled) администратор отменял ему запись именно по этому назначению —
// отмена не должна «воскресать» при следующем входе.
func Enroll(ctx context.Context, q DBTX, a Assignment, userID int64, now time.Time, keepCancelled bool) (int64, error) {
	if keepCancelled {
		var cancelled bool
		if err := q.QueryRow(ctx, `
			SELECT EXISTS (SELECT 1 FROM public.course_enrollments
			               WHERE user_id = $1 AND assignment_id = $2 AND status = 'cancelled')`,
			userID, a.ID).Scan(&cancelled); err != nil {
			return 0, err
		}
		if cancelled {
			return 0, nil
		}
	}
	var id int64
	err := q.QueryRow(ctx, `
		INSERT INTO public.course_enrollments (assignment_id, course_version_id, user_id, status, starts_at, deadline_at)
		VALUES ($1, $2, $3, 'not_started', $4, $5)
		ON CONFLICT (user_id, course_version_id) WHERE status NOT IN ('cancelled') DO NOTHING
		RETURNING id`,
		a.ID, a.VersionID, userID, a.StartsAt, EnrollmentDeadline(now, a)).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, nil
	}
	return id, err
}

// ResolveAllUsers — все активные учётные записи, в том числе без ОФО
// (не входившие на портал или не прошедшие приветствие).
func (s *Service) ResolveAllUsers(ctx context.Context) ([]int64, error) {
	rows, err := s.Pool.Query(ctx, `SELECT id FROM public.user_info WHERE status IS TRUE ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

// ofoWithAncestors — подразделение и все его родители (для назначений
// «включая вложенные» на вышестоящее ОФО).
func (s *Service) ofoWithAncestors(ctx context.Context, ofoID int64) ([]int64, error) {
	rows, err := s.Pool.Query(ctx, `
		WITH RECURSIVE up AS (
			SELECT id, parent_id, 1 AS depth FROM public.ofo_unit WHERE id = $1
			UNION ALL
			SELECT o.id, o.parent_id, up.depth + 1 FROM public.ofo_unit o
			JOIN up ON o.id = up.parent_id
			WHERE up.depth < 32
		)
		SELECT id FROM up`, ofoID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

// SyncStandingAssignments досоздаёт сотруднику записи по действующим
// назначениям «всем» и на его ОФО (опубликованные версии неудалённых курсов).
// Возвращает число созданных записей.
func (s *Service) SyncStandingAssignments(ctx context.Context, userID int64) (int, error) {
	var active bool
	var ofoRaw *string
	err := s.Pool.QueryRow(ctx, `SELECT status IS TRUE, ofo FROM public.user_info WHERE id = $1`, userID).Scan(&active, &ofoRaw)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	if !active {
		return 0, nil
	}
	var ofoID int64 = -1
	if ofoRaw != nil {
		var n int64
		if _, e := fmt.Sscan(*ofoRaw, &n); e == nil && n > 0 {
			ofoID = n
		}
	}
	chain := []int64{}
	if ofoID > 0 {
		if chain, err = s.ofoWithAncestors(ctx, ofoID); err != nil {
			return 0, err
		}
	}

	rows, err := s.Pool.Query(ctx, `
		SELECT a.id, a.course_version_id, a.starts_at, a.deadline_at, a.deadline_days
		FROM public.course_assignments a
		JOIN public.course_versions v ON v.id = a.course_version_id AND v.status = 'published'
		JOIN public.course_courses c ON c.id = v.course_id AND c.deleted_at IS NULL
		WHERE a.cancelled_at IS NULL
		  AND (a.target_type = 'all'
		       OR (a.target_type = 'ofo' AND (a.target_id = $1 OR (a.include_children AND a.target_id = ANY($2)))))
		  AND NOT EXISTS (SELECT 1 FROM public.course_enrollments e
		                  WHERE e.user_id = $3 AND e.course_version_id = a.course_version_id
		                    AND e.status NOT IN ('cancelled'))
		ORDER BY a.created_at, a.id`, ofoID, chain, userID)
	if err != nil {
		return 0, err
	}
	var todo []Assignment
	for rows.Next() {
		var a Assignment
		if err := rows.Scan(&a.ID, &a.VersionID, &a.StartsAt, &a.DeadlineAt, &a.DeadlineDays); err != nil {
			rows.Close()
			return 0, err
		}
		todo = append(todo, a)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, err
	}

	created := 0
	now := time.Now()
	for _, a := range todo {
		id, err := Enroll(ctx, s.Pool, a, userID, now, true)
		if err != nil {
			return created, err
		}
		if id > 0 {
			created++
			_ = s.EnsureTopicProgressRows(ctx, id, a.VersionID)
			Audit(ctx, s.Pool, &userID, "course.enrollment.auto", "course_enrollment", &id,
				map[string]any{"assignmentId": a.ID}, nil)
		}
	}
	return created, nil
}

// MigrationError переводит ошибку схемы в понятный ответ: без V12 назначение
// «всем» упирается в CHECK, а новые поля — в отсутствующую колонку.
func MigrationError(err error) error {
	var pe *pgconn.PgError
	if errors.As(err, &pe) && (pe.Code == "23514" || pe.Code == "42703") {
		return Err(http.StatusServiceUnavailable, "База не обновлена: нужна миграция V12__course_assign_all.sql")
	}
	return err
}
