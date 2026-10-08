-- Версия выпуска девблога («1.0.1»): поле в черновике и у опубликованного девблога.
-- По умолчанию фронт подставляет последнюю опубликованную + 0.0.1 (первый выпуск — 1.0.0),
-- администратор может изменить вручную. Показывается под заголовком новости девблога.
-- Нужна V20. Повторно применима: deploy.sh прогоняет все V*.sql при каждом деплое.

ALTER TABLE public.devblog_draft ADD COLUMN IF NOT EXISTS release_version text NOT NULL DEFAULT '';

ALTER TABLE public.devblog_posts ADD COLUMN IF NOT EXISTS release_version text NOT NULL DEFAULT '';
