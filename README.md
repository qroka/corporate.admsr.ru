# Корпоративный портал ADMSR

> Внутренний портал для сотрудников Администрации: новости, мероприятия, тесты, кадры и сервисы самообслуживания в одном веб-приложении.

**Продакшен:** [corporate.admsr.ru](https://corporate.admsr.ru)

| Слой | Стек |
|------|------|
| Frontend | Vue 3 · Vite · Nuxt UI · Vue Router · Pinia · Tailwind CSS 4 |
| API | PHP 8.3 (PHP-FPM) |
| БД | PostgreSQL 14+ |
| AI-чат | Python · FastAPI · Ollama (`python-ai/`) |

---

## Зачем это нужно

Сотрудникам нужен единый вход в корпоративные сервисы: узнать новости и события, отметить отсутствие, пройти тест или опрос, посмотреть дни рождения коллег, открыть материалы отделов. Портал закрывает эти сценарии без разрозненных таблиц и внешних форм.

Киоск-режим (`/kiosk`) даёт тот же контент на общих экранах — без админ-функций и с упрощённой навигацией.

---

## Quick Start

**Требования:** Node.js 20 LTS, npm 10+.

```bash
npm install
npm run dev
```

Откройте [http://localhost:5173](http://localhost:5173).

В dev запросы `/api` и `/img` проксируются на тестовый сервер (см. `vite.config.js`). Локальный PHP и PostgreSQL для фронтенд-разработки не обязательны.

```bash
npm run build    # → dist/
npm run preview  # проверка production-сборки
```

> В отличие от [grafic.admsr.ru](https://github.com/qroka/grafic.admsr.ru), здесь **нет** Node.js/Fastify и скрипта `build:server`. API — PHP через PHP-FPM.

---

## Возможности

### Для сотрудников

| Раздел | Маршрут | Описание |
|--------|---------|----------|
| Главная | `/` | Лента новостей, актуальные мероприятия, дни рождения |
| Новости | `/news`, `/news/:id` | Лента и карточка новости, реакции |
| Мероприятия | `/events`, `/events/:id` | Афиша и детальная страница |
| Фотогалерея | `/gallery`, `/gallery/:albumId` | Альбомы и фото |
| Дни рождения | `/birthdays` | Календарь дней рождения коллег |
| Журнал отсутствия | `/absence-journal` | Отметки об отсутствии |
| Профиль | `/profile` | Карточка сотрудника, стена постов |
| Тесты и формы | `/tests` | Конструктор, прохождение, статистика |
| Публичная ссылка | `/t/:token` | Прохождение теста/формы без входа |
| Учебные курсы | `/courses`, `/courses/history` | Назначенные курсы, прохождение, история |
| AI-ассистент | `/chatbot` | Чат с локальной LLM |
| Онбординг | `/welcome` | Первый вход нового пользователя |

### Отделы и справочники

- **Новичкам** (`/newcomers`), **корпоративная культура** (`/culture`)
- **База знаний**, **заявки**, **кадровый резерв**
- Страницы отделов: кадров, муниципальной службы, развития и мотивации
- **ОФО** — организационно-функциональная структура (дерево подразделений и должностей)

### Администрирование

- Дэшборд администратора (`/admin`, роль `admin`)
- Управление пользователями, ОФО, контентом новостей/событий
- Публикация, архивация и отчёты по формам/тестам
- Учебные курсы (`/admin/courses`): конструктор, публикация, назначение, результаты

### Киоск

Префикс `/kiosk` — те же разделы в layout `AppKiosk`. Роль принудительно `user`; админ-режим недоступен.

---

## Архитектура

```
┌─────────────────┐     /api/*.php      ┌──────────────────┐
│  Vue 3 SPA      │ ──────────────────► │  PHP-FPM         │
│  (dist/)        │                     │  api/            │
└────────┬────────┘                     └────────┬─────────┘
         │ /img/*                                │
         ▼                                       ▼
┌─────────────────┐                     ┌──────────────────┐
│  Uploads        │                     │  PostgreSQL      │
│  /var/lib/...   │                     │  corporate_portal│
└─────────────────┘                     └──────────────────┘

         /chatbot ──► python-ai (FastAPI + Ollama)
```

### Структура репозитория

```
├── src/                 # Vue-приложение
│   ├── pages/           # Страницы (маршруты)
│   ├── components/      # UI-компоненты
│   ├── composables/     # Данные и логика (news, events, tests…)
│   ├── stores/          # Pinia / локальные сторы (role, absence)
│   ├── router/          # Vue Router + auth guards
│   └── tests/           # Модуль форм/тестов (типы, API, схемы Zod)
├── api/                 # PHP endpoints
│   └── Upload/          # Загрузка изображений
├── db/migration/        # SQL-миграции (Flyway-стиль)
├── deploy/              # nginx, deploy.sh, env-пример
├── scripts/             # Утилиты (webp, gallery JSON, Excel…)
├── python-ai/           # FastAPI-сервер чат-бота
├── public/              # Статика и symlink на uploads
└── dist/                # Результат `npm run build`
```

### Аутентификация

1. Пользователь вводит логин/пароль на `/login`.
2. `POST /api/auth.php` проверяет учётные данные (в т.ч. через синхронизацию с ASU при первом входе) и возвращает профиль.
3. Профиль хранится в `localStorage` (`auth-user`).
4. Вместе с профилем API отдаёт `sessionToken` — серверная сессия в `user_sessions` (см. миграцию V4). SPA сохраняет токен в `localStorage` (`auth-session`) и шлёт его как `Authorization: Bearer` / `X-Session-Token` на API **курсов** и `tests_attempt_*`.
5. Router guard каждые 15 минут вызывает `POST /api/check-auth.php`.
6. Публичный маршрут `/t/:token` авторизации не требует (для обычных тестов с `access_by_link`, не для тестов курса).

Конфиг БД для health-check: `api/config.local.php` (из `api/config.local.php.example`). Не коммитьте секреты.

> Модуль курсов **не** доверяет `userId` из тела запроса. Часть легаси API тестов/форм по-прежнему может принимать `userId` в JSON — см. `docs/tests-users-ofo.md` и `docs/courses-permissions.md`.

---

## API (обзор)

Все endpoints — JSON over HTTP, каталог `api/`.

| Группа | Файлы | Назначение |
|--------|-------|------------|
| Auth | `auth.php`, `check-auth.php`, `logout.php`, `heartbeat.php` | Вход, проверка сессии, выход |
| Контент | `news.php`, `events.php`, `gallery.php`, `gallery_base.php` | Новости, события, галерея |
| Люди | `users.php`, `profile.php`, `birthdays.php` | Пользователи, профиль, ДР |
| ОФО | `ofo.php`, `ofo_tree.php`, `ofo_positions.php`, `ofo_seats.php` | Структура организации |
| Отсутствия | `absence_journal.php` | Журнал отсутствия |
| Тесты/формы | `forms*.php`, `tests_*.php` | CRUD, публикация, прохождение, статистика |
| Курсы (LMS) | `courses_*.php`, `course_*.php`, `tests_attempt_*.php` | Конструктор, назначение, прохождение, попытки тестов курса |
| Прочее | `chat.php`, `sync.php`, `Upload/upload.php`, `health.php` | Чат, синхронизация, загрузки, мониторинг |

Проверка живости:

```bash
curl -fsS -H "Host: corporate.admsr.ru" http://127.0.0.1/api/health.php
```

Ожидаемый ответ: `{ "ok": true, "service": "corporate-portal", "database": "ok", ... }`.

---

## База данных

Миграции в `db/migration/`:

| Файл | Содержание |
|------|------------|
| `V1__init_schema.sql` | Формы/тесты: `users`, `forms`, `questions`, `question_options`, `form_responses`, ответы |
| `V2__tests_module.sql` | Расширения модуля тестов |
| `V3__tests_link.sql` | Публичные ссылки по токену |
| `V4__courses_module.sql` | LMS: `user_sessions`, `course_*` (версии, темы, материалы, enrollment, completions) |

Основная БД: `corporate_portal` (PostgreSQL). Расширение `pgcrypto` для UUID.

---

## Скрипты npm

| Команда | Описание |
|---------|----------|
| `npm run dev` | Dev-сервер Vite (`:5173`) |
| `npm run build` | Production-сборка в `dist/` |
| `npm run preview` | Локальный просмотр сборки |
| `npm run images:webp` | Генерация WebP-обложек мероприятий |
| `npm run gallery:json` | Сборка JSON галереи |
| `npm run formdata:excel` | Экспорт formdata в Excel |
| `npm run test:courses` | Smoke-тест схемы курсов (`php scripts/test_courses.php`) |

Дополнительно в `scripts/`: конвертация журнала отсутствия (Python), экспорт SQL галереи, smoke LMS.

---

## AI-ассистент (`python-ai/`)

Отдельный сервис на FastAPI + Ollama для страницы `/chatbot`.

```bash
cd python-ai
pip install -r requirements.txt   # Python ≥ 3.11
./start.sh                        # или python server.py
```

Требуется запущенный демон Ollama на машине сервера.

---

## Развёртывание (Ubuntu 20.04 / 22.04)

### 1. Требования

| Компонент | Версия |
|-----------|--------|
| Node.js | 20 LTS |
| npm | 10+ |
| nginx | 1.18+ |
| PHP-FPM | 8.2+ (на сервере: 8.3) |
| PostgreSQL | 14+ |
| pg_dump | для бэкапов в `deploy.sh` |

```bash
# Node 20 (NodeSource)
curl -fsSL https://deb.nodesource.com/setup_20.x | sudo -E bash -
sudo apt-get install -y nodejs build-essential

# PHP, nginx, PostgreSQL
sudo apt-get install -y nginx php8.3-fpm php8.3-pgsql php8.3-mbstring php8.3-xml postgresql-client
```

### 2. Клонирование

```bash
sudo mkdir -p /var/www/corporate.admsr.ru
sudo chown -R "$USER:www-data" /var/www/corporate.admsr.ru
git clone https://github.com/qroka/corporate.admsr.ru.git /var/www/corporate.admsr.ru
cd /var/www/corporate.admsr.ru
git checkout main
```

### 3. Конфигурация API

```bash
cp api/config.local.php.example api/config.local.php
# отредактируйте учётные данные PostgreSQL
chmod 640 api/config.local.php
```

Файл `api/config.local.php` использует `api/health.php`. Остальные endpoint'ы могут содержать константы в своих `.php` — при переносе на прод сверьте пароли во всех файлах `api/` или вынесите в общий `config.local.php`.

### 4. Каталог данных

```bash
sudo mkdir -p /var/lib/corporate-app/uploads/img \
  /var/lib/corporate-app/uploads/courses \
  /var/lib/corporate-app/backups
sudo chown -R www-data:www-data /var/lib/corporate-app
cd /var/www/corporate.admsr.ru
ln -sfn /var/lib/corporate-app/uploads/img public/img
```

Материалы курсов пишутся в `/var/lib/corporate-app/uploads/courses/{courseId}/` и отдаются через `GET /api/course_file.php` (с сессией), не через публичный `/img/`.

### 5. Переменные деплоя

```bash
cp deploy/deploy.env.example deploy/deploy.env
# задайте PGPASSWORD для бэкапов
chmod 600 deploy/deploy.env
```

### 6. Первая сборка

```bash
npm ci
npm run build
```

Проверка health (после настройки nginx и PHP):

```bash
curl -fsS -H "Host: corporate.admsr.ru" http://127.0.0.1/api/health.php
```

### 7. PHP-FPM

API не запускается отдельным Node-процессом — используется **php8.3-fpm**:

```bash
sudo systemctl enable php8.3-fpm
sudo systemctl start php8.3-fpm
sudo systemctl status php8.3-fpm
```

Сокет по умолчанию: `unix:/run/php/php8.3-fpm.sock` (см. nginx-конфиг).

### 8. nginx

```bash
sudo cp deploy/nginx-corporate.admsr.ru.conf /etc/nginx/sites-available/corporate.admsr.ru
sudo ln -sf /etc/nginx/sites-available/corporate.admsr.ru /etc/nginx/sites-enabled/
sudo nginx -t
sudo systemctl reload nginx
```

Сертификаты: `/etc/nginx/ssl/admsr.ru/admsr.crt` и `admsr.key`.

| Параметр | Значение |
|----------|----------|
| Домен | `corporate.admsr.ru` |
| Корень SPA | `/var/www/corporate.admsr.ru/dist` |
| API | `/var/www/corporate.admsr.ru/api/*.php` |
| Загрузки | `/var/lib/corporate-app/uploads/img/` → `/img/` |

### 9. Обновление на сервере

```bash
cd /var/www/corporate.admsr.ru
chmod +x deploy/deploy.sh
./deploy/deploy.sh
```

Опции:

```bash
./deploy/deploy.sh -b main              # ветка
./deploy/deploy.sh -n                   # dry-run
./deploy/deploy.sh --skip-backup        # без pg_dump
./deploy/deploy.sh --skip-pull          # только сборка
```

Скрипт выполняет:

1. бэкап PostgreSQL (если задан `PGPASSWORD` в `deploy/deploy.env`);
2. сброс локальных изменений в `auto-imports.d.ts`, `components.d.ts`, `src/route-map.d.ts`;
3. `git pull --ff-only`;
4. `npm ci` (или `pnpm install --frozen-lockfile`);
5. `npm run build`;
6. `systemctl reload php8.3-fpm`;
7. health-check `GET /api/health.php` (до 10 попыток).

### 10. Права git на сервере

Деплой и `git pull` выполняйте **одним пользователем** (например `www-data` или свой user с группой `www-data`):

```bash
sudo chown -R deploy-user:www-data /var/www/corporate.admsr.ru
```

Не смешивайте `git pull` от `root` и сборку от обычного пользователя — иначе ошибки прав на `.git/objects`.

---

## Частые проблемы

| Симптом | Причина | Решение |
|---------|---------|---------|
| **502** на `/api/*` | PHP-FPM не запущен или неверный сокет | `systemctl status php8.3-fpm`, сверить `fastcgi_pass` в nginx |
| **404** на `/api/*.php` | Неверный `SCRIPT_FILENAME` | Использовать `deploy/nginx-corporate.admsr.ru.conf` |
| **Белая страница** | Нет `dist/index.html` | `npm run build`, nginx `root` → `.../dist` |
| **git pull** конфликтует с `package-lock.json` | Локальный `npm install` на сервере | `git checkout -- package-lock.json && git pull` |
| Health-check падает | Нет nginx / неверный Host | `curl -H "Host: corporate.admsr.ru" http://127.0.0.1/api/health.php` |
| Нет картинок `/img/` | Не создан symlink | `ln -sfn /var/lib/corporate-app/uploads/img public/img` |
| `insufficient permission` в `.git` | pull от другого пользователя | `sudo chown -R user:user .git` |
| Редирект на `/login` | Нет `auth-user` в localStorage или сессия протухла | Войти заново; проверить `check-auth.php` |

---

## Структура deploy

```
deploy/
  deploy.sh                      # скрипт обновления
  deploy.env.example             # пример переменных (скопировать в deploy.env)
  nginx-corporate.admsr.ru.conf  # виртуальный хост nginx
```

Эталон по форме: [grafic.admsr.ru](https://github.com/qroka/grafic.admsr.ru) (`deploy/deploy.sh`, nginx, README) — с учётом стека **PHP + PostgreSQL** вместо Node + SQLite.

---

## Разработка: соглашения

- **UI:** Nuxt UI + Tailwind; primary — `emerald`, neutral — `slate` (`vite.config.js`).
- **Данные:** composables (`useNewsData`, `useEventsData`, …) ходят в `/api/*.php`.
- **Роли:** `src/stores/role.js` — `user` | `admin`; админ-маршруты с `meta.requiresAdmin`.
- **Формы/тесты:** типы и Zod-схемы в `src/tests/`; API-обёртки в `src/tests/api.ts`.
- **Курсы:** страницы в `src/pages/Courses/`; стор `useCoursesStore.ts`; session fetch — `useAuthSession.ts`.
- **Секреты:** только в `api/config.local.php` и `deploy/deploy.env` (в `.gitignore`).

---

## Модуль учебных курсов

LMS поверх активного модуля тестов (`test_forms`): курс → версия → темы → материалы / тесты; назначение создаёт enrollment; прохождение с sequential unlock и снимком в `course_completions`.

| | |
|--|--|
| Админ UI | `/admin/courses` … create, workspace, topics, materials, tests, publish, assign, results |
| Сотрудник | `/courses`, `/courses/:enrollmentId`, темы, тесты, `/result`, `/history` |
| Миграция | `db/migration/V4__courses_module.sql` (+ таблица `user_sessions`) |
| Uploads | `/var/lib/corporate-app/uploads/courses/` |
| Auth | `sessionToken` из `auth.php`; admin = `user_group=admin`; body `userId` не используется |
| Smoke | `npm run test:courses` |

Документация в `docs/`:

| Файл | Содержание |
|------|------------|
| [courses-architecture.md](docs/courses-architecture.md) | Иерархия, версионирование, enrollment |
| [courses-database.md](docs/courses-database.md) | Таблицы V4, индексы, FK |
| [courses-api.md](docs/courses-api.md) | Эндпоинты `courses_*` / `course_*` / `tests_attempt_*` |
| [courses-test-integration.md](docs/courses-test-integration.md) | Связь с `test_forms`, attempt lifecycle |
| [courses-admin-guide.md](docs/courses-admin-guide.md) | UX администратора |
| [courses-employee-guide.md](docs/courses-employee-guide.md) | UX сотрудника |
| [courses-permissions.md](docs/courses-permissions.md) | Сессии и ограничения легаси |
| [courses-deployment.md](docs/courses-deployment.md) | Миграция, nginx, php-fpm, uploads |
| [courses-testing.md](docs/courses-testing.md) | Smoke, E2E и security checklist |

---

## Лицензия

ISC © ADMSR
