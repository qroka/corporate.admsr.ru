-- V12: назначение курса «всем сотрудникам» и относительный срок у назначения.
--
-- target_type = 'all' (target_id = 0) — курс получают все активные учётные
-- записи, в том числе те, кто ещё не входил; кто появится позже, получает
-- курс при входе (courses.Service.SyncStandingAssignments, ADR-036).
-- deadline_days — срок «N дней», считается от момента, когда сотрудник
-- получил курс, а не от даты назначения.
--
-- Идемпотентно: deploy.sh прогоняет все V*.sql при каждом деплое — и в
-- лексическом порядке (V12 раньше V4), поэтому без таблицы ничего не делаем:
-- на чистой БД миграция применится при следующем прогоне.

DO $$
BEGIN
  IF to_regclass('public.course_assignments') IS NULL THEN
    RETURN;
  END IF;

  ALTER TABLE public.course_assignments DROP CONSTRAINT IF EXISTS course_assignments_target_chk;
  ALTER TABLE public.course_assignments
    ADD CONSTRAINT course_assignments_target_chk CHECK (target_type IN ('user', 'ofo', 'all'));

  ALTER TABLE public.course_assignments ADD COLUMN IF NOT EXISTS deadline_days int;

  CREATE INDEX IF NOT EXISTS course_assignments_active_idx
    ON public.course_assignments(target_type, target_id)
    WHERE cancelled_at IS NULL;
END
$$;
