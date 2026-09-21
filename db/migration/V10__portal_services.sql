-- Portal services (Services page / home tiles / sidebar children)
CREATE TABLE IF NOT EXISTS public.portal_services (
  id            bigserial PRIMARY KEY,
  kind          text NOT NULL CHECK (kind IN ('internal', 'external')),
  label         text NOT NULL,
  description   text NOT NULL DEFAULT '',
  icon          text NOT NULL DEFAULT 'i-lucide-layout-grid',
  internal_key  text NULL,
  path          text NULL,
  external_url  text NULL,
  sort_order    integer NOT NULL DEFAULT 0,
  is_enabled    boolean NOT NULL DEFAULT true,
  created_at    timestamptz NOT NULL DEFAULT now(),
  updated_at    timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT portal_services_internal_chk CHECK (
    (kind = 'internal' AND path IS NOT NULL AND btrim(path) <> '')
    OR (kind = 'external' AND external_url IS NOT NULL AND btrim(external_url) <> '')
  )
);

CREATE INDEX IF NOT EXISTS portal_services_enabled_order_idx
  ON public.portal_services (is_enabled, sort_order, id);

-- Seed current hardcoded services (skip if already present)
INSERT INTO public.portal_services (kind, label, description, icon, internal_key, path, sort_order, is_enabled)
SELECT v.kind, v.label, v.description, v.icon, v.internal_key, v.path, v.sort_order, true
FROM (VALUES
  ('internal', 'Журнал отсутствия', 'Отметки об отсутствии сотрудника', 'i-lucide-calendar-off', 'absence-journal', '/absence-journal', 10),
  ('internal', 'Формы', 'Опросы, анкеты и тесты', 'i-lucide-clipboard-list', 'tests', '/tests', 20),
  ('internal', 'Заявки', 'Подача и отслеживание сервисных заявок', 'i-lucide-file-text', 'applications', '/applications', 30)
) AS v(kind, label, description, icon, internal_key, path, sort_order)
WHERE NOT EXISTS (
  SELECT 1 FROM public.portal_services s WHERE s.internal_key = v.internal_key
);
