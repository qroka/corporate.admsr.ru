-- Тестовые данные для локальной БД. Пароли — только для локального стенда.
-- Вход идёт по user_info без обращения к ASU (auth.go: findLocalUser → passwordOK).
--
--   логин dev.admin    пароль dev-admin-pass     суперадминистратор
--   логин dev.employee пароль dev-employee-pass  сотрудник «Отдела кадров»
--   логин dev.second   пароль dev-second-pass    сотрудник «Юридического отдела»

INSERT INTO public.ofo_category (id, name, sort_order) VALUES
  (1, 'Структурные подразделения', 1)
ON CONFLICT (id) DO NOTHING;

INSERT INTO public.ofo_unit (id, name, category_id, parent_id, level, unit_number, sort_order) VALUES
  (100, 'Аппарат администрации', 1, NULL, 0, 100, 1),
  (101, 'Отдел кадров',          1, 100,  1, 101, 1),
  (102, 'Юридический отдел',     1, 100,  1, 102, 2)
ON CONFLICT (id) DO NOTHING;

INSERT INTO public.ofo_position (id, name, is_head, sort_order) VALUES
  (1, 'Начальник отдела', true, 1),
  (2, 'Главный специалист', false, 2)
ON CONFLICT (id) DO NOTHING;

INSERT INTO public.ofo_unit_position (unit_number, position_id)
SELECT v.u, v.p FROM (VALUES (101, 1), (101, 2), (102, 1), (102, 2)) AS v(u, p)
WHERE NOT EXISTS (SELECT 1 FROM public.ofo_unit_position);

INSERT INTO public.user_info (id, status, login, password, firstname, surname, lastname, email, ofo, user_group, role, avatar_url) VALUES
  -- avatar_url задан, чтобы вход не уводил на онбординг (/welcome)
  (900001, true, 'dev.admin',    'dev-admin-pass',    'Анна',   'Админова',    'Сергеевна',  'admin@example.test',    '100', 'admin', 'Администратор портала', '/favicon.svg'),
  (900002, true, 'dev.employee', 'dev-employee-pass', 'Иван',   'Сотрудников', 'Петрович',   'employee@example.test', '101', 'user',  'Главный специалист', '/favicon.svg'),
  (900003, true, 'dev.second',   'dev-second-pass',   'Мария',  'Юристова',    'Андреевна',  'second@example.test',   '102', 'user',  'Начальник отдела', '/favicon.svg')
ON CONFLICT (id) DO NOTHING;

INSERT INTO public.news (title, category, description, date)
SELECT 'Локальный стенд запущен', 'Новости', '<p>Тестовая новость локальной БД.</p>', CURRENT_DATE
WHERE NOT EXISTS (SELECT 1 FROM public.news);
