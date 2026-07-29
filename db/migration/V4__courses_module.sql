-- V4__courses_module.sql
-- Модуль учебных курсов + серверные сессии авторизации.
-- PostgreSQL. Schema: public. Идемпотентно.
-- Soft refs на user_info / ofo_unit (без жёстких FK) — как в V2.
-- Тесты курса хранятся в test_forms; связь через course_test_links.

CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- ─────────────────────────────────────────────────────────────────────────────
-- 0. user_sessions — минимальный серверный auth context (для модуля курсов)
-- ─────────────────────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS public.user_sessions (
  id            bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  user_id       bigint NOT NULL,
  token_hash    text NOT NULL UNIQUE,
  created_at    timestamptz NOT NULL DEFAULT now(),
  last_seen_at  timestamptz NOT NULL DEFAULT now(),
  expires_at    timestamptz NOT NULL,
  revoked_at    timestamptz,
  ip_address    text,
  user_agent    text
);

CREATE INDEX IF NOT EXISTS user_sessions_user_idx
  ON public.user_sessions(user_id) WHERE revoked_at IS NULL;
CREATE INDEX IF NOT EXISTS user_sessions_expires_idx
  ON public.user_sessions(expires_at) WHERE revoked_at IS NULL;

-- ─────────────────────────────────────────────────────────────────────────────
-- 1. course_courses
-- ─────────────────────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS public.course_courses (
  id                  bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  owner_id            bigint NOT NULL,
  title               text NOT NULL,
  category            text,
  current_version_id  bigint,
  created_at          timestamptz NOT NULL DEFAULT now(),
  updated_at          timestamptz NOT NULL DEFAULT now(),
  deleted_at          timestamptz
);

CREATE INDEX IF NOT EXISTS course_courses_owner_idx ON public.course_courses(owner_id);
CREATE INDEX IF NOT EXISTS course_courses_deleted_idx ON public.course_courses(deleted_at);

-- ─────────────────────────────────────────────────────────────────────────────
-- 2. course_versions
-- ─────────────────────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS public.course_versions (
  id                      bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  course_id               bigint NOT NULL REFERENCES public.course_courses(id),
  version_number          integer NOT NULL,
  status                  text NOT NULL DEFAULT 'draft',
  short_description       text NOT NULL DEFAULT '',
  full_description        text NOT NULL DEFAULT '',
  cover_url               text,
  sequential_progress     boolean NOT NULL DEFAULT true,
  completion_rule         text NOT NULL DEFAULT 'all_required',
  default_deadline_days   integer,
  final_passing_score     numeric(5,2),
  require_final_test      boolean NOT NULL DEFAULT false,
  created_by              bigint,
  created_at              timestamptz NOT NULL DEFAULT now(),
  updated_at              timestamptz NOT NULL DEFAULT now(),
  published_at            timestamptz,
  archived_at             timestamptz,
  CONSTRAINT course_versions_status_chk CHECK (status IN ('draft', 'published', 'archived')),
  CONSTRAINT course_versions_uniq UNIQUE (course_id, version_number)
);

CREATE INDEX IF NOT EXISTS course_versions_course_status_idx
  ON public.course_versions(course_id, status);

-- current_version_id → course_versions (после создания таблицы)
DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM pg_constraint WHERE conname = 'course_courses_current_version_fk'
  ) THEN
    ALTER TABLE public.course_courses
      ADD CONSTRAINT course_courses_current_version_fk
      FOREIGN KEY (current_version_id) REFERENCES public.course_versions(id)
      DEFERRABLE INITIALLY DEFERRED;
  END IF;
END $$;

-- ─────────────────────────────────────────────────────────────────────────────
-- 3. course_topics
-- ─────────────────────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS public.course_topics (
  id                      bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  course_version_id       bigint NOT NULL REFERENCES public.course_versions(id) ON DELETE CASCADE,
  title                   text NOT NULL,
  description             text NOT NULL DEFAULT '',
  sort_order              integer NOT NULL DEFAULT 0,
  is_required             boolean NOT NULL DEFAULT true,
  minimum_active_seconds  integer NOT NULL DEFAULT 0,
  completion_rule         text NOT NULL DEFAULT 'all_required_materials',
  created_at              timestamptz NOT NULL DEFAULT now(),
  updated_at              timestamptz NOT NULL DEFAULT now(),
  deleted_at              timestamptz
);

CREATE INDEX IF NOT EXISTS course_topics_version_order_idx
  ON public.course_topics(course_version_id, sort_order);

-- ─────────────────────────────────────────────────────────────────────────────
-- 4. course_materials
-- ─────────────────────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS public.course_materials (
  id                      bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  topic_id                bigint NOT NULL REFERENCES public.course_topics(id) ON DELETE CASCADE,
  type                    text NOT NULL,
  title                   text NOT NULL,
  description             text NOT NULL DEFAULT '',
  content_html            text,
  file_url                text,
  external_url            text,
  mime_type               text,
  file_size               bigint,
  original_filename       text,
  sort_order              integer NOT NULL DEFAULT 0,
  is_required             boolean NOT NULL DEFAULT true,
  minimum_active_seconds  integer NOT NULL DEFAULT 0,
  created_at              timestamptz NOT NULL DEFAULT now(),
  updated_at              timestamptz NOT NULL DEFAULT now(),
  deleted_at              timestamptz,
  CONSTRAINT course_materials_type_chk CHECK (type IN (
    'rich_text', 'file', 'pdf', 'image', 'video', 'link'
  ))
);

CREATE INDEX IF NOT EXISTS course_materials_topic_order_idx
  ON public.course_materials(topic_id, sort_order);

-- ─────────────────────────────────────────────────────────────────────────────
-- 5. course_test_links — связь с test_forms
-- ─────────────────────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS public.course_test_links (
  id                  bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  course_version_id   bigint NOT NULL REFERENCES public.course_versions(id) ON DELETE CASCADE,
  topic_id            bigint REFERENCES public.course_topics(id) ON DELETE CASCADE,
  test_form_id        bigint NOT NULL REFERENCES public.test_forms(id),
  type                text NOT NULL,
  is_required         boolean NOT NULL DEFAULT true,
  sort_order          integer NOT NULL DEFAULT 0,
  created_at          timestamptz NOT NULL DEFAULT now(),
  updated_at          timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT course_test_links_type_chk CHECK (type IN ('topic', 'final')),
  CONSTRAINT course_test_links_topic_chk CHECK (
    (type = 'topic' AND topic_id IS NOT NULL) OR
    (type = 'final' AND topic_id IS NULL)
  )
);

CREATE UNIQUE INDEX IF NOT EXISTS course_test_links_topic_uniq
  ON public.course_test_links(topic_id) WHERE type = 'topic' AND topic_id IS NOT NULL;
CREATE UNIQUE INDEX IF NOT EXISTS course_test_links_final_uniq
  ON public.course_test_links(course_version_id) WHERE type = 'final';
CREATE INDEX IF NOT EXISTS course_test_links_version_idx ON public.course_test_links(course_version_id);
CREATE INDEX IF NOT EXISTS course_test_links_topic_idx ON public.course_test_links(topic_id);
CREATE INDEX IF NOT EXISTS course_test_links_form_idx ON public.course_test_links(test_form_id);

-- ─────────────────────────────────────────────────────────────────────────────
-- 6. course_assignments
-- ─────────────────────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS public.course_assignments (
  id                  bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  course_version_id   bigint NOT NULL REFERENCES public.course_versions(id),
  target_type         text NOT NULL,
  target_id           bigint NOT NULL,
  starts_at           timestamptz,
  deadline_at         timestamptz,
  assigned_by         bigint NOT NULL,
  comment             text,
  include_children    boolean NOT NULL DEFAULT false,
  created_at          timestamptz NOT NULL DEFAULT now(),
  cancelled_at        timestamptz,
  CONSTRAINT course_assignments_target_chk CHECK (target_type IN ('user', 'ofo'))
);

CREATE INDEX IF NOT EXISTS course_assignments_version_idx ON public.course_assignments(course_version_id);
CREATE INDEX IF NOT EXISTS course_assignments_target_idx ON public.course_assignments(target_type, target_id);

-- ─────────────────────────────────────────────────────────────────────────────
-- 7. course_enrollments
-- ─────────────────────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS public.course_enrollments (
  id                  bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  assignment_id       bigint REFERENCES public.course_assignments(id),
  course_version_id   bigint NOT NULL REFERENCES public.course_versions(id),
  user_id             bigint NOT NULL,
  status              text NOT NULL DEFAULT 'not_started',
  assigned_at         timestamptz NOT NULL DEFAULT now(),
  starts_at           timestamptz,
  deadline_at         timestamptz,
  started_at          timestamptz,
  last_activity_at    timestamptz,
  completed_at        timestamptz,
  final_score         numeric(5,2),
  created_at          timestamptz NOT NULL DEFAULT now(),
  updated_at          timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT course_enrollments_status_chk CHECK (status IN (
    'not_started', 'in_progress', 'completed', 'failed', 'overdue', 'cancelled'
  ))
);

-- Один активный enrollment на пользователя + версию
CREATE UNIQUE INDEX IF NOT EXISTS course_enrollments_active_uniq
  ON public.course_enrollments(user_id, course_version_id)
  WHERE status NOT IN ('cancelled');

CREATE INDEX IF NOT EXISTS course_enrollments_user_status_idx
  ON public.course_enrollments(user_id, status);
CREATE INDEX IF NOT EXISTS course_enrollments_version_status_idx
  ON public.course_enrollments(course_version_id, status);
CREATE INDEX IF NOT EXISTS course_enrollments_deadline_idx
  ON public.course_enrollments(deadline_at);

-- ─────────────────────────────────────────────────────────────────────────────
-- 8. course_topic_progress
-- ─────────────────────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS public.course_topic_progress (
  id                bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  enrollment_id     bigint NOT NULL REFERENCES public.course_enrollments(id) ON DELETE CASCADE,
  topic_id          bigint NOT NULL REFERENCES public.course_topics(id),
  status            text NOT NULL DEFAULT 'locked',
  opened_at         timestamptz,
  completed_at      timestamptz,
  active_seconds    integer NOT NULL DEFAULT 0,
  last_material_id  bigint,
  updated_at        timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT course_topic_progress_status_chk CHECK (status IN (
    'locked', 'available', 'in_progress', 'completed'
  )),
  CONSTRAINT course_topic_progress_uniq UNIQUE (enrollment_id, topic_id)
);

CREATE INDEX IF NOT EXISTS course_topic_progress_enrollment_idx
  ON public.course_topic_progress(enrollment_id);

-- ─────────────────────────────────────────────────────────────────────────────
-- 9. course_material_progress
-- ─────────────────────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS public.course_material_progress (
  id              bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  enrollment_id   bigint NOT NULL REFERENCES public.course_enrollments(id) ON DELETE CASCADE,
  material_id     bigint NOT NULL REFERENCES public.course_materials(id),
  status          text NOT NULL DEFAULT 'not_started',
  opened_at       timestamptz,
  completed_at    timestamptz,
  active_seconds  integer NOT NULL DEFAULT 0,
  updated_at      timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT course_material_progress_status_chk CHECK (status IN (
    'not_started', 'in_progress', 'completed'
  )),
  CONSTRAINT course_material_progress_uniq UNIQUE (enrollment_id, material_id)
);

CREATE INDEX IF NOT EXISTS course_material_progress_enrollment_idx
  ON public.course_material_progress(enrollment_id);

-- ─────────────────────────────────────────────────────────────────────────────
-- 10. course_learning_sessions — телеметрия активного времени
-- ─────────────────────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS public.course_learning_sessions (
  id                  bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  enrollment_id       bigint NOT NULL REFERENCES public.course_enrollments(id) ON DELETE CASCADE,
  topic_id            bigint NOT NULL REFERENCES public.course_topics(id),
  material_id         bigint REFERENCES public.course_materials(id),
  user_id             bigint NOT NULL,
  started_at          timestamptz NOT NULL DEFAULT now(),
  last_heartbeat_at   timestamptz NOT NULL DEFAULT now(),
  finished_at         timestamptz,
  active_seconds      integer NOT NULL DEFAULT 0,
  ip_address          text,
  user_agent          text
);

CREATE INDEX IF NOT EXISTS course_learning_sessions_enrollment_hb_idx
  ON public.course_learning_sessions(enrollment_id, last_heartbeat_at);

-- ─────────────────────────────────────────────────────────────────────────────
-- 11. course_test_attempt_links
-- ─────────────────────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS public.course_test_attempt_links (
  id                    bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  enrollment_id         bigint NOT NULL REFERENCES public.course_enrollments(id) ON DELETE CASCADE,
  course_test_link_id   bigint NOT NULL REFERENCES public.course_test_links(id),
  test_attempt_id       bigint NOT NULL REFERENCES public.test_attempts(id),
  created_at            timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT course_test_attempt_links_attempt_uniq UNIQUE (test_attempt_id)
);

CREATE INDEX IF NOT EXISTS course_test_attempt_links_enrollment_idx
  ON public.course_test_attempt_links(enrollment_id);

-- ─────────────────────────────────────────────────────────────────────────────
-- 12. course_completions — неизменяемый снимок
-- ─────────────────────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS public.course_completions (
  id                    bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  enrollment_id         bigint NOT NULL REFERENCES public.course_enrollments(id),
  user_id               bigint NOT NULL,
  course_id             bigint NOT NULL,
  course_version_id     bigint NOT NULL,
  completion_number     integer NOT NULL DEFAULT 1,
  assigned_at           timestamptz,
  started_at            timestamptz,
  completed_at          timestamptz NOT NULL DEFAULT now(),
  total_active_seconds  integer NOT NULL DEFAULT 0,
  final_score           numeric(5,2),
  passed                boolean NOT NULL DEFAULT false,
  result_snapshot       jsonb NOT NULL DEFAULT '{}'::jsonb,
  created_at            timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS course_completions_user_completed_idx
  ON public.course_completions(user_id, completed_at);

-- ─────────────────────────────────────────────────────────────────────────────
-- 13. course_audit_logs
-- ─────────────────────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS public.course_audit_logs (
  id            bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  user_id       bigint,
  action        text NOT NULL,
  entity_type   text NOT NULL,
  entity_id     bigint,
  payload       jsonb NOT NULL DEFAULT '{}'::jsonb,
  ip_address    text,
  user_agent    text,
  created_at    timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS course_audit_logs_entity_idx
  ON public.course_audit_logs(entity_type, entity_id);
CREATE INDEX IF NOT EXISTS course_audit_logs_created_idx
  ON public.course_audit_logs(created_at);
