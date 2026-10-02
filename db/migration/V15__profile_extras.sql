-- V15__profile_extras.sql
-- Страница профиля: «Желания» сотрудника и «Награды» (грамоты, достижения).
--   profile_wishes — пишет и удаляет сам сотрудник (до 20 штук, проверяет Go);
--   profile_awards — выдаёт и удаляет администратор портала.
-- user_id / issued_by — soft refs на user_info (таблица вне миграций, IMP-14).
-- Повторно применима: deploy.sh прогоняет все V*.sql при каждом деплое.

CREATE TABLE IF NOT EXISTS public.profile_wishes (
  id         bigserial   PRIMARY KEY,
  user_id    bigint      NOT NULL,
  text       text        NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS profile_wishes_user_idx
  ON public.profile_wishes (user_id, id);

CREATE TABLE IF NOT EXISTS public.profile_awards (
  id          bigserial   PRIMARY KEY,
  user_id     bigint      NOT NULL,   -- кому выдано
  title       text        NOT NULL,
  description text        NOT NULL DEFAULT '',
  awarded_on  date        NOT NULL,
  issued_by   bigint,                 -- кто выдал
  created_at  timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS profile_awards_user_idx
  ON public.profile_awards (user_id, awarded_on DESC, id DESC);
