-- V3__tests_link.sql — доступ по ссылке (внешние / авторизованные / все) + гости.
-- Идемпотентно.

-- Режим доступа по ссылке на форме (access_token уже есть в V2)
ALTER TABLE public.test_forms ADD COLUMN IF NOT EXISTS link_access text NOT NULL DEFAULT 'any';
DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'test_forms_link_access_chk') THEN
    ALTER TABLE public.test_forms
      ADD CONSTRAINT test_forms_link_access_chk CHECK (link_access IN ('authorized', 'guest', 'any'));
  END IF;
END $$;

-- Метаданные прохождения по ссылке / гостя
ALTER TABLE public.test_attempts ADD COLUMN IF NOT EXISTS via_link boolean NOT NULL DEFAULT false;
ALTER TABLE public.test_attempts ADD COLUMN IF NOT EXISTS respondent_token text; -- дедуп гостя (браузерный токен)
ALTER TABLE public.test_attempts ADD COLUMN IF NOT EXISTS guest_name text;       -- ФИО гостя
ALTER TABLE public.test_attempts ADD COLUMN IF NOT EXISTS guest_ofo_id bigint;   -- выбранное гостем ОФО (ofo_unit.id)

CREATE INDEX IF NOT EXISTS test_attempts_respondent_idx ON public.test_attempts(form_id, respondent_token);
