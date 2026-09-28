-- Базовые («легаси») таблицы портала для ЛОКАЛЬНОЙ БД разработки.
--
-- В проде эти таблицы созданы вне репозитория, а db/migration/ их не создаёт
-- (IMP-14). Схема восстановлена по SQL-запросам Go (backend/internal/**) —
-- это НЕ копия продовой схемы: типы и ограничения подобраны так, чтобы
-- запросы Go работали. На сервере этот файл не применять.

CREATE TABLE IF NOT EXISTS public.user_info (
  id            bigint PRIMARY KEY,
  status        boolean NOT NULL DEFAULT true,
  login         text NOT NULL,
  password      text,
  firstname     text,
  surname       text,
  lastname      text,
  phone         text,
  email         text,
  ofo           text,              -- id ofo_unit строкой; '-1' = не задано
  user_group    text NOT NULL DEFAULT 'user',  -- 'admin' | 'user'
  role          text,              -- должность, не роль доступа
  auth          boolean NOT NULL DEFAULT false,
  last_activity timestamptz,
  avatar_url    text
);

CREATE TABLE IF NOT EXISTS public.ofo_category (
  id         bigint PRIMARY KEY,
  name       text NOT NULL,
  sort_order int NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS public.ofo_unit (
  id            bigint PRIMARY KEY,
  name          text NOT NULL,
  category_id   bigint NOT NULL,
  parent_id     bigint,
  level         int NOT NULL DEFAULT 0,
  unit_number   bigint NOT NULL,
  family_number bigint,
  sort_order    int NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS public.ofo_position (
  id         bigint PRIMARY KEY,
  name       text NOT NULL,
  is_head    boolean NOT NULL DEFAULT false,
  sort_order int NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS public.ofo_unit_position (
  unit_number bigint NOT NULL,
  position_id bigint NOT NULL
);

CREATE TABLE IF NOT EXISTS public.ofo (
  id         bigint PRIMARY KEY,
  title      text NOT NULL,
  parent     bigint NOT NULL DEFAULT 0,
  type       text NOT NULL DEFAULT '',
  sort_order int NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS public.ofo_seats (
  id        bigint PRIMARY KEY,
  title     text NOT NULL DEFAULT '',
  ofo       text NOT NULL DEFAULT '',
  insurance text NOT NULL DEFAULT '',
  rating    text NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS public.news (
  id          bigserial PRIMARY KEY,
  title       text NOT NULL,
  category    text NOT NULL DEFAULT '',
  description text NOT NULL DEFAULT '',
  date        date NOT NULL,
  image_path  text,
  created_at  timestamptz DEFAULT now(),
  updated_at  timestamptz DEFAULT now(),
  likes       bigint DEFAULT 0,
  views       bigint DEFAULT 0
);

CREATE TABLE IF NOT EXISTS public.gallery (
  id          bigserial PRIMARY KEY,
  name        text NOT NULL,
  description text,
  date        date
);

CREATE TABLE IF NOT EXISTS public.gallery_base (
  id              bigserial PRIMARY KEY,
  album_id        bigint NOT NULL,
  image_full_url  text NOT NULL,
  image_small_url text NOT NULL
);

CREATE TABLE IF NOT EXISTS public.events (
  id          bigserial PRIMARY KEY,
  title       text NOT NULL,
  description text,
  badge       text,
  date        timestamptz,
  image       text NOT NULL DEFAULT '/favicon.svg',
  image_full  text NOT NULL DEFAULT '/favicon.svg',
  created_at  timestamptz DEFAULT now(),
  updated_at  timestamptz DEFAULT now()
);

CREATE TABLE IF NOT EXISTS public.absence_journal (
  id             bigserial PRIMARY KEY,
  user_id        bigint NOT NULL,
  fio            text NOT NULL DEFAULT '',
  ofo            bigint NOT NULL DEFAULT 0,
  role           text NOT NULL DEFAULT '',
  start_datetime timestamptz NOT NULL,
  end_datetime   timestamptz,
  reason         text,
  created_at     timestamptz NOT NULL DEFAULT now()
);

-- portal_feedback создаёт сам Go (handlers/feedback.go), здесь не нужна.
