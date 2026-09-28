-- V11__news_reactions.sql
-- Реакции сотрудников на новости: у одного сотрудника на одну новость может быть
-- несколько разных реакций, каждая — не больше одного раза.
-- Допустимые ключи реакций задаются в коде (backend/internal/handlers/news_reactions.go
-- и src/composables/useNewsReactions.ts) — CHECK не ставим, чтобы добавление
-- реакции не требовало миграции.
-- news_id / user_id — soft refs, как в остальных модулях (news и user_info вне миграций).
-- Колонка news.likes остаётся: это накопленные анонимные лайки до появления реакций,
-- они прибавляются к счётчику реакции «like».

CREATE TABLE IF NOT EXISTS public.news_reactions (
  news_id    bigint      NOT NULL,
  user_id    bigint      NOT NULL,
  reaction   text        NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (news_id, user_id, reaction)
);

CREATE INDEX IF NOT EXISTS news_reactions_news_reaction_idx
  ON public.news_reactions (news_id, reaction, created_at DESC);
