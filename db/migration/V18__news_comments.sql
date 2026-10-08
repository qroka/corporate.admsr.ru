-- Комментарии к новостям: два уровня, как во ВКонтакте.
--   news_comments           — root_id IS NULL: комментарий к новости; иначе — ответ в ветке root_id.
--                             Ответ на ответ попадает в ту же ветку; reply_to_id / reply_to_user_id —
--                             кому ответили («в ответ Ивану»). Комментарий с ответами удаляется мягко
--                             (deleted_at, текст стирается), без ответов и ответы — физически.
--   news_comment_reactions  — реакции, те же ключи, что у новостей
--                             (backend/internal/handlers/news_reactions.go и
--                             src/composables/useNewsReactions.ts — менять парами).
--   notifications.comment_id — уведомление «ответили на ваш комментарий» (kind = 'comment_reply').
-- news_id / author_id / reply_to_user_id / user_id — soft refs (news и user_info вне миграций, IMP-14).
-- Нужна V17 (таблица notifications). Повторно применима: deploy.sh прогоняет все V*.sql при каждом деплое.

CREATE TABLE IF NOT EXISTS public.news_comments (
  id               bigserial   PRIMARY KEY,
  news_id          bigint      NOT NULL,
  root_id          bigint      REFERENCES public.news_comments (id) ON DELETE CASCADE,
  reply_to_id      bigint      REFERENCES public.news_comments (id) ON DELETE SET NULL,
  reply_to_user_id bigint,
  author_id        bigint      NOT NULL,
  content          text        NOT NULL,   -- простой текст, без HTML
  created_at       timestamptz NOT NULL DEFAULT now(),
  updated_at       timestamptz,
  deleted_at       timestamptz
);

CREATE INDEX IF NOT EXISTS news_comments_news_roots_idx
  ON public.news_comments (news_id, created_at DESC)
  WHERE root_id IS NULL;

CREATE INDEX IF NOT EXISTS news_comments_root_idx
  ON public.news_comments (root_id, created_at)
  WHERE root_id IS NOT NULL;

CREATE TABLE IF NOT EXISTS public.news_comment_reactions (
  comment_id bigint      NOT NULL REFERENCES public.news_comments (id) ON DELETE CASCADE,
  user_id    bigint      NOT NULL,
  reaction   text        NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (comment_id, user_id, reaction)
);

CREATE INDEX IF NOT EXISTS news_comment_reactions_comment_reaction_idx
  ON public.news_comment_reactions (comment_id, reaction, created_at DESC);

ALTER TABLE public.notifications
  ADD COLUMN IF NOT EXISTS comment_id bigint REFERENCES public.news_comments (id) ON DELETE CASCADE;
