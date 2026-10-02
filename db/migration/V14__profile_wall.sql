-- V14__profile_wall.sql
-- Профиль сотрудника «как стена ВКонтакте»:
--   profile_about       — «О себе» и «Интересы», заполняет сам сотрудник;
--   wall_posts          — записи на стене: писать может любой вошедший коллега;
--   wall_post_reactions — реакции на записи, те же ключи, что у новостей
--                         (backend/internal/handlers/news_reactions.go и
--                         src/composables/useNewsReactions.ts — менять парами).
-- owner_id / author_id / user_id — soft refs на user_info (таблица вне миграций, IMP-14).
-- Повторно применима: deploy.sh прогоняет все V*.sql при каждом деплое.

CREATE TABLE IF NOT EXISTS public.profile_about (
  user_id    bigint      PRIMARY KEY,
  about      text        NOT NULL DEFAULT '',
  interests  text        NOT NULL DEFAULT '',
  updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS public.wall_posts (
  id         bigserial   PRIMARY KEY,
  owner_id   bigint      NOT NULL,   -- чья стена
  author_id  bigint      NOT NULL,   -- кто написал
  content    text        NOT NULL,   -- простой текст, без HTML
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz
);

CREATE INDEX IF NOT EXISTS wall_posts_owner_created_idx
  ON public.wall_posts (owner_id, created_at DESC, id DESC);

CREATE TABLE IF NOT EXISTS public.wall_post_reactions (
  post_id    bigint      NOT NULL REFERENCES public.wall_posts (id) ON DELETE CASCADE,
  user_id    bigint      NOT NULL,
  reaction   text        NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (post_id, user_id, reaction)
);

CREATE INDEX IF NOT EXISTS wall_post_reactions_post_reaction_idx
  ON public.wall_post_reactions (post_id, reaction, created_at DESC);
