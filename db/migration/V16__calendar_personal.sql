-- V16__calendar_personal.sql
-- Серверное хранение того, что раньше жило только в localStorage браузера (Q-03):
--   event_rsvps      — «Записаться» на мероприятие (раньше events-rsvp:v1);
--   calendar_entries — личные записи и встречи в календаре (раньше portal-calendar-local:v1).
-- user_id — soft ref на user_info (таблица вне миграций, IMP-14). Личность — только из сессии.
-- Повторно применима: deploy.sh прогоняет все V*.sql при каждом деплое.

CREATE TABLE IF NOT EXISTS public.event_rsvps (
  event_id   bigint      NOT NULL REFERENCES public.events (id) ON DELETE CASCADE,
  user_id    bigint      NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (event_id, user_id)
);

CREATE INDEX IF NOT EXISTS event_rsvps_user_idx ON public.event_rsvps (user_id);

CREATE TABLE IF NOT EXISTS public.calendar_entries (
  id         bigserial   PRIMARY KEY,
  user_id    bigint      NOT NULL,
  source     text        NOT NULL,   -- 'meeting' | 'personal'
  date_key   date        NOT NULL,
  title      text        NOT NULL,
  time_start text,                   -- 'HH:MM'
  time_end   text,                   -- 'HH:MM'
  location   text,
  created_at timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT calendar_entries_source_chk CHECK (source IN ('meeting', 'personal'))
);

CREATE INDEX IF NOT EXISTS calendar_entries_user_date_idx
  ON public.calendar_entries (user_id, date_key);
