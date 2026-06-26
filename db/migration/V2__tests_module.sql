-- V2__tests_module.sql
-- Модуль «Тесты» (test / survey / poll). Отдельная схема, не пересекается с легаси public.forms/*.
-- DB: PostgreSQL. Schema: public. Идемпотентно (IF NOT EXISTS / guards).
--
-- Ссылки на внешние сущности (owner_id, user_id, ofo_unit_id) — обычные bigint без жёстких FK,
-- т.к. public.user_info и public.ofo_unit синхронизируются (sync.php). Целостность — на стороне API.

CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- Последовательность для публичного номера в «Списке» (list_no).
-- Выдаётся только при первой публикации, не переиспользуется.
CREATE SEQUENCE IF NOT EXISTS public.test_forms_list_no_seq;

-- ─────────────────────────────────────────────────────────────────────────────
-- 1. test_forms — форма (черновик / опубликованная)
-- ─────────────────────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS public.test_forms (
  id                   bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  list_no              bigint UNIQUE,                -- NULL до первой публикации; затем закрепляется навсегда
  status               text NOT NULL DEFAULT 'draft',
  owner_id             bigint,                       -- user_info.id создателя
  kind                 text NOT NULL DEFAULT 'test',
  visibility           text NOT NULL DEFAULT 'public',
  title                text NOT NULL DEFAULT '',
  description          text NOT NULL DEFAULT '',
  completion_message   text NOT NULL DEFAULT '',
  -- Параметры
  shuffle              boolean NOT NULL DEFAULT false,
  shuffle_options      boolean NOT NULL DEFAULT false,
  show_progress        boolean NOT NULL DEFAULT false,
  free_navigation      boolean NOT NULL DEFAULT false,
  anonymous            boolean NOT NULL DEFAULT false,
  allow_change_answer  boolean NOT NULL DEFAULT false,
  live_results         boolean NOT NULL DEFAULT false,
  allow_revote         boolean NOT NULL DEFAULT false,
  notify_creator       boolean NOT NULL DEFAULT false,
  use_passing_score    boolean NOT NULL DEFAULT false,
  passing_score        int NOT NULL DEFAULT 70,
  show_correct_answers boolean NOT NULL DEFAULT false,
  restrict_by_ofo      boolean NOT NULL DEFAULT false,
  -- Ограничения
  use_time_limit       boolean NOT NULL DEFAULT false,
  time_limit_sec       int,
  limit_attempts       boolean NOT NULL DEFAULT false,
  attempts             int NOT NULL DEFAULT 1,
  use_start            boolean NOT NULL DEFAULT true,
  starts_at            date,
  use_end              boolean NOT NULL DEFAULT false,
  ends_at              date,
  show_result          text NOT NULL DEFAULT 'after',
  access_by_link       boolean NOT NULL DEFAULT false,
  access_token         text UNIQUE,
  -- Мета
  created_at           timestamptz NOT NULL DEFAULT now(),
  updated_at           timestamptz NOT NULL DEFAULT now(),
  published_at         timestamptz,
  CONSTRAINT test_forms_status_chk      CHECK (status IN ('draft', 'published')),
  CONSTRAINT test_forms_kind_chk        CHECK (kind IN ('test', 'survey', 'poll')),
  CONSTRAINT test_forms_visibility_chk  CHECK (visibility IN ('public', 'private')),
  CONSTRAINT test_forms_show_result_chk CHECK (show_result IN ('immediate', 'after', 'never')),
  CONSTRAINT test_forms_passing_chk     CHECK (passing_score BETWEEN 0 AND 100)
);

CREATE INDEX IF NOT EXISTS test_forms_status_idx     ON public.test_forms(status);
CREATE INDEX IF NOT EXISTS test_forms_owner_idx      ON public.test_forms(owner_id);
CREATE INDEX IF NOT EXISTS test_forms_list_no_idx    ON public.test_forms(list_no);
CREATE INDEX IF NOT EXISTS test_forms_created_at_idx ON public.test_forms(created_at);

-- ─────────────────────────────────────────────────────────────────────────────
-- 2. test_questions — вопросы формы (упорядочены position)
-- ─────────────────────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS public.test_questions (
  id               bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  form_id          bigint NOT NULL REFERENCES public.test_forms(id) ON DELETE CASCADE,
  position         int NOT NULL DEFAULT 0,
  type             text NOT NULL,
  title            text NOT NULL DEFAULT '',
  hint             text NOT NULL DEFAULT '',
  required         boolean NOT NULL DEFAULT false,
  scale_min        int,
  scale_max        int,
  scale_min_label  text NOT NULL DEFAULT '',
  scale_max_label  text NOT NULL DEFAULT '',
  correct_value    text,  -- yesno: yes/no; scale/number/date/text/textarea: литерал (сравнение в API)
  created_at       timestamptz NOT NULL DEFAULT now(),
  updated_at       timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT test_questions_type_chk CHECK (type IN (
    'single', 'multiple', 'dropdown', 'text', 'textarea', 'scale', 'yesno', 'number', 'date'
  ))
);

CREATE INDEX IF NOT EXISTS test_questions_form_idx     ON public.test_questions(form_id);
CREATE INDEX IF NOT EXISTS test_questions_form_pos_idx ON public.test_questions(form_id, position);

-- ─────────────────────────────────────────────────────────────────────────────
-- 3. test_options — варианты / кандидаты (single / multiple / dropdown)
-- ─────────────────────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS public.test_options (
  id           bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  question_id  bigint NOT NULL REFERENCES public.test_questions(id) ON DELETE CASCADE,
  position     int NOT NULL DEFAULT 0,
  text         text NOT NULL DEFAULT '',
  is_correct   boolean NOT NULL DEFAULT false,
  created_at   timestamptz NOT NULL DEFAULT now(),
  updated_at   timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS test_options_question_idx     ON public.test_options(question_id);
CREATE INDEX IF NOT EXISTS test_options_question_pos_idx ON public.test_options(question_id, position);

-- ─────────────────────────────────────────────────────────────────────────────
-- 4. test_audience_ofo — кому направлена форма (ОФО)
-- ─────────────────────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS public.test_audience_ofo (
  id           bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  form_id      bigint NOT NULL REFERENCES public.test_forms(id) ON DELETE CASCADE,
  ofo_unit_id  bigint NOT NULL,  -- ofo_unit.id
  source       text NOT NULL DEFAULT 'directed',
  created_at   timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT test_audience_ofo_source_chk CHECK (source IN ('initial', 'directed')),
  CONSTRAINT test_audience_ofo_uniq UNIQUE (form_id, ofo_unit_id)
);

CREATE INDEX IF NOT EXISTS test_audience_ofo_form_idx ON public.test_audience_ofo(form_id);

-- ─────────────────────────────────────────────────────────────────────────────
-- 5. test_audience_users — кому направлена форма (лично)
-- ─────────────────────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS public.test_audience_users (
  id          bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  form_id     bigint NOT NULL REFERENCES public.test_forms(id) ON DELETE CASCADE,
  user_id     bigint NOT NULL,  -- user_info.id
  source      text NOT NULL DEFAULT 'directed',
  created_at  timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT test_audience_users_source_chk CHECK (source IN ('initial', 'directed')),
  CONSTRAINT test_audience_users_uniq UNIQUE (form_id, user_id)
);

CREATE INDEX IF NOT EXISTS test_audience_users_form_idx ON public.test_audience_users(form_id);

-- ─────────────────────────────────────────────────────────────────────────────
-- 6. test_attempts — прохождения и незавершённые сессии («Продолжить»)
-- ─────────────────────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS public.test_attempts (
  id             bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  form_id        bigint NOT NULL REFERENCES public.test_forms(id) ON DELETE CASCADE,
  user_id        bigint,                              -- NULL = аноним
  session_token  uuid NOT NULL DEFAULT gen_random_uuid(),
  status         text NOT NULL DEFAULT 'in_progress',
  current_page   int NOT NULL DEFAULT 0,
  started_at     timestamptz NOT NULL DEFAULT now(),
  finished_at    timestamptz,
  duration_sec   int,
  score          numeric(6,2),
  max_score      numeric(6,2),
  passed         boolean,
  ip             inet,
  user_agent     text,
  created_at     timestamptz NOT NULL DEFAULT now(),
  updated_at     timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT test_attempts_status_chk CHECK (status IN ('in_progress', 'completed', 'expired')),
  CONSTRAINT test_attempts_session_uniq UNIQUE (form_id, session_token)
);

CREATE INDEX IF NOT EXISTS test_attempts_form_status_idx ON public.test_attempts(form_id, status);
CREATE INDEX IF NOT EXISTS test_attempts_user_idx        ON public.test_attempts(user_id);
CREATE INDEX IF NOT EXISTS test_attempts_finished_idx    ON public.test_attempts(finished_at);

-- ─────────────────────────────────────────────────────────────────────────────
-- 7. test_answers — ответы в попытке
-- ─────────────────────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS public.test_answers (
  id            bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  attempt_id    bigint NOT NULL REFERENCES public.test_attempts(id) ON DELETE CASCADE,
  question_id   bigint NOT NULL REFERENCES public.test_questions(id) ON DELETE CASCADE,
  text_value    text,
  number_value  numeric(14,4),
  is_correct    boolean,
  answered      boolean NOT NULL DEFAULT true,
  created_at    timestamptz NOT NULL DEFAULT now(),
  updated_at    timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT test_answers_uniq UNIQUE (attempt_id, question_id)
);

CREATE INDEX IF NOT EXISTS test_answers_attempt_idx  ON public.test_answers(attempt_id);
CREATE INDEX IF NOT EXISTS test_answers_question_idx ON public.test_answers(question_id);

-- ─────────────────────────────────────────────────────────────────────────────
-- 8. test_answer_options — выбранные варианты (single / multiple / dropdown)
-- ─────────────────────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS public.test_answer_options (
  answer_id   bigint NOT NULL REFERENCES public.test_answers(id) ON DELETE CASCADE,
  option_id   bigint NOT NULL REFERENCES public.test_options(id) ON DELETE CASCADE,
  PRIMARY KEY (answer_id, option_id)
);

CREATE INDEX IF NOT EXISTS test_answer_options_option_idx ON public.test_answer_options(option_id);

-- ─────────────────────────────────────────────────────────────────────────────
-- Триггеры updated_at (переиспользуем public.set_updated_at() из V1)
-- ─────────────────────────────────────────────────────────────────────────────
DO $$
DECLARE
  t text;
  tables text[] := ARRAY[
    'test_forms', 'test_questions', 'test_options', 'test_attempts', 'test_answers'
  ];
BEGIN
  FOREACH t IN ARRAY tables LOOP
    IF NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'trg_' || t || '_updated_at') THEN
      EXECUTE format(
        'CREATE TRIGGER trg_%1$s_updated_at BEFORE UPDATE ON public.%1$s
         FOR EACH ROW EXECUTE FUNCTION public.set_updated_at();', t
      );
    END IF;
  END LOOP;
END $$;
