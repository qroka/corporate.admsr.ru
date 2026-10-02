# Снятие PHP (`api/`) — чек-лист

Решение владельца 2026-10-02 (Q-02, ADR-047): перенести остаток PHP в Go, затем удалить PHP.
**Перенос закончен. Удаление `api/` — НЕ выполнено** и делается отдельно, после проверки на сервере.

## 1. Что подтверждено в репозитории

Проверено сверкой `api/**/*.php` ↔ маршруты `backend/cmd/api/main.go` ↔ `location = /api/…` в nginx ↔
прокси `vite.config.js` (2026-10-02):

- В `api/` 88 файлов: 85 точек входа + 3 общих хелпера (`auth_context.php`, `courses_common.php`,
  `tests_common.php`).
- **Каждая из 85 точек входа имеет Go-маршрут**, точный `location = …` в nginx
  (`deploy/nginx-corporate.admsr.ru.conf` — 22 штуки, `deploy/nginx-go-api-wave2.conf` — остальные) и
  строку в прокси Vite. Без Go-аналога не осталось ничего.
- Последние 7 перенесены 2026-10-02: `course_assignment_cancel`, `course_assignments_list`,
  `course_history`, `course_materials_order`, `courses_archive`, `courses_duplicate`,
  `courses_readiness`. SPA их не вызывала; проверены вживую на локальной БД (ADR-047).
- Только в Go (PHP-файла нет): `portal_services.php`, `event_rsvp.php`, `calendar_entries.php`,
  `profile_wall.php`, `profile_extras.php`, `profile_avatar.php`.

**Чего подтвердить нельзя из репозитория:** что на проде именно эти конфиги и что в nginx не осталось
других путей в PHP. Для этого п. 3.

## 2. Где PHP ещё упоминается (что менять при удалении)

| Место | Что там |
|-------|---------|
| `api/` | сам каталог |
| `deploy/nginx-corporate.admsr.ru.conf` (≈ стр. 283–292) | `location ~ ^/api/(.+\.php)$` → `fastcgi_pass` php-fpm: запасной путь для всего, чего нет в allowlist. После удаления — заменить на `return 404` (или убрать) |
| `deploy/nginx/corporate.admsr.ru.conf`, `deploy/nginx/default` | старые конфиги «всё в PHP-FPM»; по виду не используются — **уточнить**, какой конфиг реально на сервере |
| `deploy/deploy.sh` | `SERVICE_NAME=php8.3-fpm`, `restart_php()` (шаг 7, вызов ≈ стр. 496), предупреждение про `api/config.local.php` (≈ стр. 107) |
| `deploy/deploy.env.example` | `SERVICE_NAME=php8.3-fpm` |
| `vite.config.js` | запасной `'/api'` → `https://172.17.4.21` (там PHP) и комментарий «everything else → PHP»; после удаления все нужные пути уже проксируются в Go |
| `package.json` | `test:courses` → `php scripts/test_courses.php`; `php-serialize` (используется `scripts/formdata-to-excel.mjs` — читает PHP-сериализованные данные, решить отдельно) |
| `scripts/` | `test_courses.php`, `seed_demo_course.php` — PHP-скрипты поверх `api/`; `scripts/course_import/` ходит по HTTP (`/api/courses_*.php`) и от PHP не зависит |
| Документация | `README.md`, `docs/architecture.md` §6, `docs/api.md` («Только PHP»), `docs/courses-api.md`, `CLAUDE.md` (таблица «Границы», п. 1 «Два backend'а»), `docs/local-setup.md` |
| Безопасность | удаление `api/` закрывает SEC-002 (пароль БД в файлах), SEC-003 (лог ASU в `/tmp`), SEC-004 (отключённая проверка TLS в ASU) из `PROJECT_AUDIT.md`. SEC-005 (пароли в БД открытым текстом) — это **данные**, удаление PHP их не меняет |

## 3. Проверка на сервере перед удалением

Нужно убедиться, что **ни один** запрос не доходит до PHP-FPM.

1. В `location ~ ^/api/(.+\.php)$` временно добавить отдельный лог:
   ```nginx
   access_log /var/log/nginx/php-fallback.log;
   ```
   и перезагрузить nginx. Дать поработать рабочий день, включая ночные задачи (синхронизация ASU — `sync.php`).
2. Посмотреть, были ли обращения:
   ```bash
   sudo wc -l /var/log/nginx/php-fallback.log
   sudo awk '{print $7}' /var/log/nginx/php-fallback.log | sort | uniq -c | sort -rn | head
   ```
   Пусто — PHP не использовался. Есть строки — это пути, которых нет в allowlist (внешний вызов или забытый
   клиент); решить по каждому.
3. Проверить внешних вызывающих `sync.php` (планировщик/cron на стороне ASU) — он идёт в Go
   (`location = /api/sync.php` в `nginx-go-api-wave2.conf`), но секрет `ASU_SECRET` должен быть задан у Go
   (см. `backend/.env`).
4. Убедиться, что на проде применены миграции V11–V16 (`deploy.sh` пропускает их молча без `PGPASSWORD`, IMP-49).

## 4. Порядок удаления

1. Сделать п. 3 и дождаться пустого лога.
2. Одним коммитом: `git rm -r api/`, правки из таблицы п. 2 (кроме `php-serialize`), обновление документации и
   CLAUDE.md (снять правило «`api/` не изменять»), закрыть SEC-002/003/004 в `PROJECT_AUDIT.md`.
3. Деплой; на сервере остановить и отключить `php8.3-fpm`, убрать пакеты PHP, когда убедитесь, что всё работает.
4. Откат: `git revert` этого коммита + вернуть блок `fastcgi` в nginx — файлы PHP останутся в истории.

## 5. Известные отличия Go от PHP у перенесённых эндпоинтов

- `courses_readiness` — только проверка, которая блокирует публикацию (`VersionReadiness`); PHP дополнительно
  разбирал ответы каждого вопроса (`cs_form_answer_errors`). Публикация в Go и до этого их не проверяла.
- `courses_duplicate` — переносит `role` и `target_option_id` вариантов (вопросы «соответствие» /
  «классификация», V13) и `generate_certificate`; PHP их не копировал, и копия таких вопросов ломалась.
  `fullDescription` не копируется (Q-09).
- `course_assignment_cancel`, `course_assignments_list`, `courses_readiness`, `courses_archive`,
  `course_materials_order` проверяют доступ к категории курса (PHP — только секцию `courses`).
  `course_assignments_list` без `courseId`/`versionId` отдаёт назначения только доступных категорий.
