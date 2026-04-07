-- V1__init_schema.sql
-- Flyway migration: init schema for forms/tests (Yandex Forms-like)
-- DB: PostgreSQL

-- Extensions
CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- ─────────────────────────────────────────────────────────────────────────────
-- Table: public.users
-- Purpose: SaaS-level users (optional mapping to existing user_info).
-- Note: In this project there is already public.user_info (legacy).
--       This table is introduced for the tests/forms module only.
-- ─────────────────────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS public.users (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  -- Link to existing numeric user id (if present in your system)
  external_user_id bigint UNIQUE,
  fio text NOT NULL DEFAULT '',
  email text,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);

-- ─────────────────────────────────────────────────────────────────────────────
-- Table: public.forms
-- Purpose: Form/test metadata & settings.
-- ─────────────────────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS public.forms (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  owner_user_id uuid REFERENCES public.users(id) ON DELETE SET NULL,
  title text NOT NULL,
  description text NOT NULL DEFAULT '',
  cover_url text,
  status text NOT NULL DEFAULT 'draft',
  mode text NOT NULL DEFAULT 'survey', -- survey | test
  settings jsonb NOT NULL DEFAULT '{}'::jsonb,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT forms_status_chk CHECK (status IN ('draft', 'published', 'archived')),
  CONSTRAINT forms_mode_chk CHECK (mode IN ('survey', 'test'))
);

CREATE INDEX IF NOT EXISTS forms_status_idx ON public.forms(status);
CREATE INDEX IF NOT EXISTS forms_owner_user_id_idx ON public.forms(owner_user_id);
CREATE INDEX IF NOT EXISTS forms_created_at_idx ON public.forms(created_at);

-- ─────────────────────────────────────────────────────────────────────────────
-- Table: public.questions
-- Purpose: Questions of a form (ordered).
-- ─────────────────────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS public.questions (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  form_id uuid NOT NULL REFERENCES public.forms(id) ON DELETE CASCADE,
  type text NOT NULL,
  "order" int NOT NULL DEFAULT 0,
  title text NOT NULL,
  hint text NOT NULL DEFAULT '',
  required boolean NOT NULL DEFAULT false,
  -- correct answer & extra config per question (for tests/scoring)
  config jsonb NOT NULL DEFAULT '{}'::jsonb,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT questions_type_chk CHECK (type IN (
    'single_choice',
    'multiple_choice',
    'short_text',
    'long_text',
    'rating_1_10',
    'select',
    'file'
  ))
);

CREATE INDEX IF NOT EXISTS questions_form_id_idx ON public.questions(form_id);
CREATE INDEX IF NOT EXISTS questions_form_order_idx ON public.questions(form_id, "order");

-- ─────────────────────────────────────────────────────────────────────────────
-- Table: public.question_options
-- Purpose: Options for choice/select questions (ordered).
-- ─────────────────────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS public.question_options (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  question_id uuid NOT NULL REFERENCES public.questions(id) ON DELETE CASCADE,
  label text NOT NULL,
  "order" int NOT NULL DEFAULT 0,
  is_correct boolean NOT NULL DEFAULT false, -- used when form.mode='test'
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS question_options_question_id_idx ON public.question_options(question_id);
CREATE INDEX IF NOT EXISTS question_options_question_order_idx ON public.question_options(question_id, "order");

-- ─────────────────────────────────────────────────────────────────────────────
-- Table: public.form_responses
-- Purpose: Response session ("attempt") per form. Supports anonymous sessions.
-- Security:
--  - unique (form_id, session_id): protects from re-submit
--  - ip stored for rate limiting
-- ─────────────────────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS public.form_responses (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  form_id uuid NOT NULL REFERENCES public.forms(id) ON DELETE CASCADE,
  user_id uuid REFERENCES public.users(id) ON DELETE SET NULL,
  session_id uuid NOT NULL,
  ip inet,
  user_agent text,
  started_at timestamptz NOT NULL DEFAULT now(),
  completed_at timestamptz,
  status text NOT NULL DEFAULT 'incomplete',
  score numeric(10,2),
  max_score numeric(10,2),
  correct_percent numeric(6,2),
  meta jsonb NOT NULL DEFAULT '{}'::jsonb,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT form_responses_status_chk CHECK (status IN ('incomplete', 'completed')),
  CONSTRAINT form_responses_unique_session UNIQUE (form_id, session_id)
);

CREATE INDEX IF NOT EXISTS form_responses_form_id_idx ON public.form_responses(form_id);
CREATE INDEX IF NOT EXISTS form_responses_completed_at_idx ON public.form_responses(completed_at);
CREATE INDEX IF NOT EXISTS form_responses_status_idx ON public.form_responses(status);
CREATE INDEX IF NOT EXISTS form_responses_ip_created_at_idx ON public.form_responses(ip, created_at);

-- ─────────────────────────────────────────────────────────────────────────────
-- Table: public.response_answers
-- Purpose: Concrete answers per question.
-- Storage strategy:
--  - answer_json keeps the raw payload (incl. files metadata/base64 if allowed)
--  - extracted columns are for analytics queries
-- ─────────────────────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS public.response_answers (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  response_id uuid NOT NULL REFERENCES public.form_responses(id) ON DELETE CASCADE,
  question_id uuid NOT NULL REFERENCES public.questions(id) ON DELETE CASCADE,
  answer_json jsonb NOT NULL DEFAULT '{}'::jsonb,
  text_value text,
  number_value numeric(12,4),
  option_ids uuid[],
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT response_answers_unique_question UNIQUE (response_id, question_id)
);

CREATE INDEX IF NOT EXISTS response_answers_response_id_idx ON public.response_answers(response_id);
CREATE INDEX IF NOT EXISTS response_answers_question_id_idx ON public.response_answers(question_id);

-- ─────────────────────────────────────────────────────────────────────────────
-- updated_at maintenance trigger
-- ─────────────────────────────────────────────────────────────────────────────
CREATE OR REPLACE FUNCTION public.set_updated_at()
RETURNS trigger AS $$
BEGIN
  NEW.updated_at = now();
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'trg_users_updated_at') THEN
    CREATE TRIGGER trg_users_updated_at BEFORE UPDATE ON public.users
    FOR EACH ROW EXECUTE FUNCTION public.set_updated_at();
  END IF;
  IF NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'trg_forms_updated_at') THEN
    CREATE TRIGGER trg_forms_updated_at BEFORE UPDATE ON public.forms
    FOR EACH ROW EXECUTE FUNCTION public.set_updated_at();
  END IF;
  IF NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'trg_questions_updated_at') THEN
    CREATE TRIGGER trg_questions_updated_at BEFORE UPDATE ON public.questions
    FOR EACH ROW EXECUTE FUNCTION public.set_updated_at();
  END IF;
  IF NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'trg_question_options_updated_at') THEN
    CREATE TRIGGER trg_question_options_updated_at BEFORE UPDATE ON public.question_options
    FOR EACH ROW EXECUTE FUNCTION public.set_updated_at();
  END IF;
  IF NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'trg_form_responses_updated_at') THEN
    CREATE TRIGGER trg_form_responses_updated_at BEFORE UPDATE ON public.form_responses
    FOR EACH ROW EXECUTE FUNCTION public.set_updated_at();
  END IF;
  IF NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'trg_response_answers_updated_at') THEN
    CREATE TRIGGER trg_response_answers_updated_at BEFORE UPDATE ON public.response_answers
    FOR EACH ROW EXECUTE FUNCTION public.set_updated_at();
  END IF;
END $$;

