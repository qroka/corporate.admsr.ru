-- V13: типы вопросов «Соответствие» (match) и «Классификация» (classify).
--
-- Оба типа — «каждому элементу слева выбрать один вариант справа»:
--   match    — понятие → определение (определения не повторяются; допустимы лишние);
--   classify — утверждение → категория (категории повторяются).
-- Хранение в test_options:
--   role = 'item'   — элемент слева (понятие / утверждение), target_option_id —
--                     правильный вариант справа (NULL — не задан / опрос);
--   role = 'target' — вариант справа (определение / категория);
--   role = 'option' — обычный вариант single / multiple / dropdown (как раньше).
-- Ответ сотрудника хранится JSON-словарём {itemId: targetId} в test_answers.text_value.
--
-- Идемпотентно: deploy.sh прогоняет все V*.sql при каждом деплое, в лексическом
-- порядке (V13 раньше V2) — на чистой БД без таблиц ничего не делаем.

DO $$
BEGIN
  IF to_regclass('public.test_questions') IS NULL OR to_regclass('public.test_options') IS NULL THEN
    RETURN;
  END IF;

  ALTER TABLE public.test_questions DROP CONSTRAINT IF EXISTS test_questions_type_chk;
  ALTER TABLE public.test_questions
    ADD CONSTRAINT test_questions_type_chk CHECK (type IN (
      'single', 'multiple', 'dropdown', 'text', 'textarea', 'scale', 'yesno', 'number', 'date',
      'match', 'classify'
    ));

  ALTER TABLE public.test_options ADD COLUMN IF NOT EXISTS role text NOT NULL DEFAULT 'option';
  ALTER TABLE public.test_options
    ADD COLUMN IF NOT EXISTS target_option_id bigint REFERENCES public.test_options(id) ON DELETE SET NULL;

  ALTER TABLE public.test_options DROP CONSTRAINT IF EXISTS test_options_role_chk;
  ALTER TABLE public.test_options
    ADD CONSTRAINT test_options_role_chk CHECK (role IN ('option', 'item', 'target'));
END
$$;
