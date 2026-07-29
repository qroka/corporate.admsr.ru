# Тестирование модуля учебных курсов

Автоматических vitest/jest-сьютов для фронта курсов в репозитории нет (`package.json` → `"test"` — заглушка). Основная проверка: CLI smoke + ручной E2E + security checklist.

---

## 1. Smoke-тест БД / API

Скрипт: `scripts/test_courses.php`  
npm: `npm run test:courses`

### Что делает

1. Подключается к PostgreSQL теми же `DB_*`, что API (`auth_context` / `tests_common` константы; при наличии читает `api/config.local.php`).
2. Проверяет в `information_schema`, что существуют таблицы модуля (`course_*`, `user_sessions`).
3. Опционально: если заданы env `COURSE_TEST_TOKEN` и `COURSE_TEST_BASE`, делает `POST …/courses_list.php` с Bearer и ожидает `success: true`.
4. Печатает строки `PASS` / `FAIL`, код выхода **0** / **1**.

### Запуск

```bash
# Только проверка схемы (нужен доступ к БД с машины)
php scripts/test_courses.php
# или
npm run test:courses

# Со HTTP-проверкой списка курсов (нужен admin sessionToken)
set COURSE_TEST_BASE=https://corporate.admsr.ru
set COURSE_TEST_TOKEN=ваш_session_token
php scripts/test_courses.php
```

На Linux/macOS:

```bash
export COURSE_TEST_BASE=https://corporate.admsr.ru
export COURSE_TEST_TOKEN=...
php scripts/test_courses.php
```

Токен: войти в портал админом → DevTools → `localStorage['auth-session']`, либо из ответа `auth.php`.

---

## 2. Ручной E2E checklist (админ → сотрудник)

### Админ

- [ ] Логин → в `localStorage` есть `auth-session`
- [ ] `/admin/courses` открывается (не 403)
- [ ] Создать курс с названием
- [ ] Добавить 2 темы, упорядочить
- [ ] В теме 1: rich_text материал + файл (upload)
- [ ] В теме 1: промежуточный тест с ≥1 вопросом и правильным ответом
- [ ] Итоговый тест (если нужен) + `requireFinalTest`
- [ ] Readiness / Review без ошибок
- [ ] Publish успешен; повторное редактирование контента блокируется
- [ ] Assign на тестового пользователя (или своё ОФО)
- [ ] Results: enrollment виден как `not_started`

### Сотрудник (вторая учётка или тот же user без admin UI)

- [ ] `/courses` — курс в активных
- [ ] Start → прогресс, nextAction на первый материал
- [ ] Открыть материал, heartbeat тикает, complete
- [ ] При sequential вторая тема locked до завершения первой
- [ ] Пройти topic test через `/courses/.../tests/...` (не `/t/...`)
- [ ] После всех тем — final test
- [ ] Result page + запись в `/courses/history`
- [ ] Админ: results → completed, snapshot заполнен

### Негатив

- [ ] Без токена `courses_list` → 401
- [ ] User без admin → `courses_create` → 403
- [ ] Чужой `enrollmentId` → 403
- [ ] Course-форма **нет** в `/tests` списках
- [ ] `/t/<token>` для course-теста не работает (нет access_by_link)

---

## 3. Security checklist

- [ ] Identity только из session / cookie; подмена `userId` в body на course API не даёт чужих данных
- [ ] Admin API требует `user_group=admin`
- [ ] Файлы материалов не отдаются статикой nginx без auth; только `course_file.php`
- [ ] HTML материалов проходит `cs_sanitize_html` (нет script/on*)
- [ ] Course tests: `visibility=private`, `access_by_link=false`
- [ ] `tests_list` исключает `course_test_links`
- [ ] Attempt finish чужой attemptId → отказ
- [ ] Upload: тип/размер ограничены; путь не выходит за uploads/courses
- [ ] Помнить: легаси `tests_*` / `forms*` могут всё ещё доверять body `userId` — не использовать их для LMS auth

---

## 4. Что не покрыто автотестами

- Vue-страницы Courses (ручной UI)
- Полный цикл heartbeat / overdue job
- Нагрузочное тестирование assign на большие ОФО

При доработках API добавляйте проверки в `scripts/test_courses.php` или отдельные CLI-скрипты — vitest в проект пока не подключён.
