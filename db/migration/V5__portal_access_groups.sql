-- V5__portal_access_groups.sql
-- Группы пользователей и права на разделы портала.
-- PostgreSQL. Schema: public. Идемпотентно.

-- ─────────────────────────────────────────────────────────────────────────────
-- 1. portal_access_groups
-- ─────────────────────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS public.portal_access_groups (
  id           bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  name         text NOT NULL,
  description  text NOT NULL DEFAULT '',
  created_at   timestamptz NOT NULL DEFAULT now(),
  updated_at   timestamptz NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS portal_access_groups_name_uidx
  ON public.portal_access_groups (lower(name));

-- ─────────────────────────────────────────────────────────────────────────────
-- 2. portal_group_permissions — section_key из фиксированного списка приложения
-- ─────────────────────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS public.portal_group_permissions (
  group_id     bigint NOT NULL REFERENCES public.portal_access_groups(id) ON DELETE CASCADE,
  section_key  text NOT NULL,
  PRIMARY KEY (group_id, section_key)
);

CREATE INDEX IF NOT EXISTS portal_group_permissions_section_idx
  ON public.portal_group_permissions(section_key);

-- ─────────────────────────────────────────────────────────────────────────────
-- 3. portal_group_members — пользователь может быть в нескольких группах
-- ─────────────────────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS public.portal_group_members (
  group_id  bigint NOT NULL REFERENCES public.portal_access_groups(id) ON DELETE CASCADE,
  user_id   bigint NOT NULL,
  PRIMARY KEY (group_id, user_id)
);

CREATE INDEX IF NOT EXISTS portal_group_members_user_idx
  ON public.portal_group_members(user_id);
