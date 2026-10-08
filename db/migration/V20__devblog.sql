-- Девблог: администраторы пишут заметку об обновлении портала (Markdown), публикуют её
-- новостью; все сотрудники видят окошко при первом заходе после публикации и уведомление.
--   devblog_draft      — один общий черновик на всех администраторов (id = 1). version растёт
--                        при каждом сохранении: сохранение с устаревшей версией отвергается
--                        («черновик изменил другой») — без молчаливой перезаписи.
--   devblog_posts      — опубликованные девблоги: новость (news_id) + исходный Markdown.
--   devblog_dismissed  — кто закрыл окошко девблога.
--   notifications.news_id — уведомление kind = 'devblog' ведёт на новость.
-- news_id / *_by / user_id — soft refs (news и user_info вне миграций, IMP-14).
-- Нужна V17 (notifications). Повторно применима: deploy.sh прогоняет все V*.sql при каждом деплое.

CREATE TABLE IF NOT EXISTS public.devblog_draft (
  id         smallint    PRIMARY KEY DEFAULT 1,
  title      text        NOT NULL DEFAULT '',
  body_md    text        NOT NULL DEFAULT '',
  image_path text,
  version    bigint      NOT NULL DEFAULT 0,
  updated_by bigint,
  updated_at timestamptz,
  CONSTRAINT devblog_draft_single_chk CHECK (id = 1)
);

INSERT INTO public.devblog_draft (id) VALUES (1) ON CONFLICT (id) DO NOTHING;

CREATE TABLE IF NOT EXISTS public.devblog_posts (
  news_id      bigint      PRIMARY KEY,
  body_md      text        NOT NULL,
  published_by bigint      NOT NULL,
  published_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS devblog_posts_published_idx
  ON public.devblog_posts (published_at DESC);

CREATE TABLE IF NOT EXISTS public.devblog_dismissed (
  news_id      bigint      NOT NULL,
  user_id      bigint      NOT NULL,
  dismissed_at timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (news_id, user_id)
);

ALTER TABLE public.notifications ADD COLUMN IF NOT EXISTS news_id bigint;
