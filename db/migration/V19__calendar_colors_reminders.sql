-- Цвет личных событий и напоминания накануне (колокольчик).
--   calendar_entries.color       — цвет события '#rrggbb', NULL — цвет по типу (встреча / личное).
--   notifications (kind = 'event_reminder') — «завтра событие»: личное (calendar_entry_id) или
--                                  мероприятие, куда записались (event_id). event_date — дата события,
--                                  на которую напомнили: перенесли событие — напомнят заново.
--                                  actor_id у напоминания нет — колонка становится необязательной.
-- Напоминания создаёт Go при запросе уведомлений (notifications.go, generateReminders), отдельного
-- планировщика нет. Нужна V17 (notifications) и V16 (calendar_entries, event_rsvps).
-- Повторно применима: deploy.sh прогоняет все V*.sql при каждом деплое.

ALTER TABLE public.calendar_entries ADD COLUMN IF NOT EXISTS color text;

ALTER TABLE public.notifications ALTER COLUMN actor_id DROP NOT NULL;

ALTER TABLE public.notifications
  ADD COLUMN IF NOT EXISTS calendar_entry_id bigint REFERENCES public.calendar_entries (id) ON DELETE CASCADE;

ALTER TABLE public.notifications
  ADD COLUMN IF NOT EXISTS event_id bigint REFERENCES public.events (id) ON DELETE CASCADE;

ALTER TABLE public.notifications ADD COLUMN IF NOT EXISTS event_date date;

-- Одно напоминание на событие и дату.
CREATE UNIQUE INDEX IF NOT EXISTS notifications_event_reminder_uidx
  ON public.notifications (user_id, COALESCE(calendar_entry_id, 0), COALESCE(event_id, 0), event_date)
  WHERE kind = 'event_reminder';
