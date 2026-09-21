# Go API — параллельная миграция с PHP

Vue SPA без изменений. Go-сервис (`backend/`) обслуживает выбранные
эндпоинты с теми же путями `/api/*.php`. Остальное остаётся на PHP-FPM.
nginx переключает маршруты точным allowlist (`location =`).

## Запуск локально

Требования: Go 1.22+, доступ к PostgreSQL `corporate_portal`.

```bash
cd backend
cp .env.example .env   # заполните DB_* ; для upload: UPLOAD_DIR=.../public
make run               # слушает :8080
```

Проверка:

```bash
curl -s http://127.0.0.1:8080/api/health.php
# { "ok": true, "service": "corporate-portal", "database": "ok", "backend": "go", ... }
```

## Перенесённые эндпоинты

### Ядро и контент
| Path | Go |
|------|-----|
| `health`, `auth`, `logout`, `check-auth`, `heartbeat`, `session_bootstrap` | ✓ |
| `news`, `events`, `gallery`, `gallery_base`, `Upload/upload` | ✓ |
| `users`, `profile`, `feedback` | ✓ |
| `ofo`, `ofo_seats`, `ofo_tree`, `ofo_positions` | ✓ |
| `absence_journal`, `portal_groups`, `portal_my_permissions` | ✓ |

### Волна 2
| Path | Go |
|------|-----|
| `sync.php` | ✓ |
| `birthdays.php` | ✓ |
| `tests_list/save/publish/unpublish/delete/direct/submit/stats/participant/by_token` | ✓ |
| `tests_attempt_start/save/get/finish` | ✓ |
| `forms.php`, `forms_list/publish/submit/report/archive/delete` | ✓ |
| `courses_*`, `course_*` (LMS, ~33 эндпоинта) | ✓ |

Ещё на PHP: прочие редкие скрипты (`course_file.php` и т.п., если не используются SPA).

## Прод: nginx allowlist

- Базовые маршруты: [`deploy/nginx-corporate.admsr.ru.conf`](../deploy/nginx-corporate.admsr.ru.conf)
- Волна 2 (tests/forms/courses/birthdays/sync): [`deploy/nginx-go-api-wave2.conf`](../deploy/nginx-go-api-wave2.conf)

## Dev: Vite proxy

[`vite.config.js`](../vite.config.js) — мигрированные пути → `127.0.0.1:8080`, остальное → PHP-сервер.

## Upload / birthdays

`UPLOAD_DIR` — каталог public (на проде `/var/lib/corporate-app/uploads` или `.../public`).
Внутри: `img/FullPic`, `img/SmallPic`, `birthdays_xlsx/`.

## Известные отличия от PHP

- Изображения сохраняются как JPEG (без CGO/libwebp).
- `tests_attempt_finish`: полная цепочка `nextAction` / auto-complete enrollment интегрирована с `internal/courses`.

Правило: handler → smoke → nginx allowlist → наблюдение → удаление PHP только после стабилизации.
