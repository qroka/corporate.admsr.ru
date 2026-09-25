# API

## Формат

Все эндпоинты — JSON over HTTP по путям вида `/api/<имя>.php`. Расширение `.php`
сохранено как URL-контракт: за ним может стоять как PHP-скрипт, так и Go-хендлер.
**[ПОДТВЕРЖДЕНО]**

Конверт ответа Go (`backend/internal/httpx/json.go`):

```json
{ "success": true, "message": "OK", "data": { … } }
```

При ошибке — `success: false`, `message` на русском, `data: null`, HTTP-код
соответствует ситуации.

Стандартные коды из `backend/internal/handlers/section.go`:

| Код | Сообщение | Когда |
|-----|-----------|-------|
| 401 | `Требуется авторизация` | нет/невалидна сессия |
| 403 | `Недостаточно прав` / `Недостаточно прав для этого раздела` | сессия есть, прав нет |
| 405 | `Метод не поддерживается` | неверный HTTP-метод |
| 500 | `Ошибка подключения к БД` | сбой БД |

> Часть PHP-скриптов отвечает полями `error` вместо `message` — клиент
> `src/tests/api.ts:27` учитывает оба варианта. **[ПОДТВЕРЖДЕНО]**

## Маршрутизация Go ↔ PHP

Три списка путей должны совпадать. На 2026-09-22 они **совпадают полностью:
79 эндпоинтов** в каждом. **[ПОДТВЕРЖДЕНО — сверено скриптом]**

| Список | Файл |
|--------|------|
| Go-маршруты | `backend/cmd/api/main.go` |
| Прод-allowlist nginx | `deploy/nginx-corporate.admsr.ru.conf` + `deploy/nginx-go-api-wave2.conf` |
| Dev-прокси Vite | `vite.config.js` |

**При добавлении Go-эндпоинта обновляйте все три.** Иначе запрос молча уйдёт в
PHP-реализацию (или в 404, если PHP-файла нет).

## Каталог эндпоинтов

### Обслуживаются Go (79)

| Группа | Эндпоинты | Хендлер |
|--------|-----------|---------|
| Здоровье | `health.php` | `health.go` |
| Аутентификация | `auth.php`, `logout.php`, `check-auth.php`, `heartbeat.php`, `session_bootstrap.php` | `auth.go` |
| Контент | `news.php`, `events.php`, `gallery.php`, `gallery_base.php`, `Upload/upload.php` | `news.go`, `events.go`, `gallery.go` |
| Люди | `users.php`, `profile.php`, `feedback.php`, `birthdays.php` | `users.go`, `profile.go`, `feedback.go`, `birthdays.go` |
| ОФО | `ofo.php`, `ofo_seats.php`, `ofo_tree.php`, `ofo_positions.php` | `ofo.go` |
| Отсутствия | `absence_journal.php` | `absence.go` |
| Портал | `portal_groups.php`, `portal_my_permissions.php`, `portal_services.php` | `portal.go`, `portal_services.go` |
| Синхронизация | `sync.php` | `sync.go` |
| Тесты (легаси-модуль) | `tests_list/save/publish/unpublish/delete/direct/submit/stats/participant/by_token.php` | `tests_routes.go` |
| Попытки тестов | `tests_attempt_start/save/get/finish.php` | `tests_routes.go`, `attempt_review.go` |
| Формы (второй модуль) | `forms.php`, `forms_list/publish/submit/report/archive/delete.php` | `forms.go` |
| LMS | `courses_list/get/create/update/delete/publish/unpublish/for_me.php`, `course_topics_*`, `course_materials_*`, `course_tests_*`, `course_assign*`, `course_admin_*`, `course_enrollment_*`, `course_start/topic_get/material_open/material_heartbeat/material_complete/next_action/result.php` | `courses.go` |

### Только PHP (Go-реализации нет)

`auth_context.php`, `course_file.php`, `course_history.php`,
`course_assignment_cancel.php`, `course_assignments_list.php`,
`course_materials_order.php`, `courses_archive.php`, `courses_duplicate.php`,
`courses_readiness.php`. **[ПОДТВЕРЖДЕНО]**

Из них фронтенд не вызывает ни один (проверено grep'ом по `src/`).
`course_file.php` упомянут в `README.md` как способ выдачи материалов курса.
**[ПОДТВЕРЖДЕНО, что вызова из SPA нет]**

### Только Go (PHP-файла нет)

`portal_services.php`. Раздел «Сервисы» без запущенного Go API отработает на
`FALLBACK_PORTAL_SERVICES`. **[ПОДТВЕРЖДЕНО]**

## Авторизация по эндпоинтам

Сведено из вызовов `requireUser` / `requireAdmin` / `requireSection` /
`CanEditSection` в `backend/internal/handlers/`. **[ПОДТВЕРЖДЕНО]**

| Эндпоинт | GET | POST / PUT / DELETE |
|----------|-----|---------------------|
| `health.php` | публично | — |
| `auth.php`, `logout.php`, `check-auth.php`, `heartbeat.php` | — | публично (сам механизм входа) |
| `session_bootstrap.php` | — | только для текущей сессии; чужой `id` в теле → 403 |
| `news.php` | публично | секция `news` |
| `events.php` | публично | секция `events` |
| `gallery.php`, `gallery_base.php` | публично | секция `gallery` |
| `Upload/upload.php` | — | см. `gallery.go:519` (`Upload`) |
| `birthdays.php` | авторизованный | секция `birthdays` |
| `absence_journal.php` | `requireUser` | секция `absence_journal` |
| `users.php` | `requireAdmin` | `requireAdmin` |
| `profile.php` | `requireUser` | владелец записи либо админ |
| `feedback.php` | — | `requireUser` |
| `ofo*.php` | `requireUser` | `requireUser` |
| `portal_groups.php` | `requireUser` / `requireAdmin` | `requireAdmin` |
| `portal_my_permissions.php` | `requireUser` | — |
| `portal_services.php` | `requireUser` | `requireAdmin` |
| `sync.php` | — | общий секрет в заголовке `X-Sync-Secret` |
| `tests_*` | личность только из сессии; публичные потоки (список опубликованных, отправка по токену) доступны гостю с `viewer=0` | операции владельца требуют сессии |
| `forms*.php` | `formsRequireUser` / `formsRequireEditor` (право секции `tests`) | то же |
| `courses_*`, `course_*` | `requireUser` + проверка enrollment; админ-операции — секция `courses` + категория курса | то же |

Детали и история закрытия дыр — [auth-and-permissions.md](auth-and-permissions.md)
и [PROJECT_AUDIT.md](PROJECT_AUDIT.md).

## CORS

`backend/internal/httpx/cors.go` разрешает с `Allow-Credentials: true` только
`http://localhost:5173`, `:5174` и их `127.0.0.1`-варианты (для Vite).
Для прочих origin'ов ставится `Access-Control-Allow-Origin: http://localhost:5173`
без `Allow-Credentials`. Разрешённые заголовки: `Content-Type`, `Authorization`,
`X-Session-Token`. **[ПОДТВЕРЖДЕНО]**

> Заголовок `X-User-Id`, который шлёт `src/tests/api.ts:22`, в
> `Access-Control-Allow-Headers` **не входит**. В проде это не мешает
> (same-origin), в dev — запрос идёт через прокси Vite, тоже same-origin.
> На реальный кросс-доменный сценарий не рассчитано. **[ПОДТВЕРЖДЕНО]**

## Курсорная пагинация

`backend/internal/handlers/cursor.go` + `src/composables/useCursorFeed.ts`.

Ответ страницы:

```json
{ "items": [...], "nextCursor": "…" | null, "hasMore": true | false }
```

Клиент дедуплицирует по `getId`, отменяет предыдущий запрос через
`AbortController`, хранит курсор в сторе (для новостей — Pinia `newsFeed`).

## Проверка живости

```bash
curl -fsS -H "Host: corporate.admsr.ru" http://127.0.0.1/api/health.php
```

Ожидается `{"ok":true,"service":"corporate-portal","database":"ok","backend":"go",…}`.
Поле `backend: "go"` — признак того, что запрос дошёл до Go, а не до PHP.
**[ПОДТВЕРЖДЕНО — `backend/README.md`, `deploy/deploy.sh`]**

## Добавление нового эндпоинта: чек-лист

1. Хендлер в `backend/internal/handlers/<домен>.go`.
2. **Guard в самом начале метода** (`requireUser` / `requireAdmin` / `requireSection`)
   — middleware-цепочки нет, забыть легко.
3. Регистрация в `backend/cmd/api/main.go`.
4. `location = /api/<имя>.php { proxy_pass http://go_api; … }` в
   `deploy/nginx-go-api-wave2.conf` (копировать блок целиком, включая
   `proxy_set_header Authorization $http_authorization`).
5. Строка в `server.proxy` в `vite.config.js` → `http://127.0.0.1:8080`.
6. Клиентская обёртка: `apiSessionFetch` для авторизованного вызова.
7. `go build ./... && go vet ./... && go test ./...`.
8. Обновить этот документ.
