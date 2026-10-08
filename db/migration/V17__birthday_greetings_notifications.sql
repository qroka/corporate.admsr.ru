-- Поздравления с днём рождения и уведомления (колокольчик в шапке).
--   wall_posts.birthday_year — запись стены является поздравлением с днём рождения
--                              этого года (год самого дня рождения, а не публикации:
--                              поздравление 2 января с 31 декабря — прошлый год).
--                              Одно поздравление от автора владельцу в год — уникальный индекс.
--   notifications            — уведомления сотруднику: «написал на вашей стене»,
--                              «поздравил с днём рождения». Удалили запись — ушло и уведомление.
-- user_id / actor_id — soft refs на user_info (таблица вне миграций, IMP-14).
-- Повторно применима: deploy.sh прогоняет все V*.sql при каждом деплое.

ALTER TABLE public.wall_posts ADD COLUMN IF NOT EXISTS birthday_year integer;

CREATE UNIQUE INDEX IF NOT EXISTS wall_posts_birthday_greeting_uidx
  ON public.wall_posts (owner_id, author_id, birthday_year)
  WHERE birthday_year IS NOT NULL;

CREATE TABLE IF NOT EXISTS public.notifications (
  id         bigserial   PRIMARY KEY,
  user_id    bigint      NOT NULL,   -- кому
  actor_id   bigint      NOT NULL,   -- кто вызвал
  kind       text        NOT NULL,   -- 'wall_post' | 'birthday_greeting'
  post_id    bigint      REFERENCES public.wall_posts (id) ON DELETE CASCADE,
  created_at timestamptz NOT NULL DEFAULT now(),
  read_at    timestamptz
);

CREATE INDEX IF NOT EXISTS notifications_user_created_idx
  ON public.notifications (user_id, created_at DESC, id DESC);

CREATE INDEX IF NOT EXISTS notifications_user_unread_idx
  ON public.notifications (user_id)
  WHERE read_at IS NULL;
