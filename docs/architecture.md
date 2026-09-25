# Архитектура

## Общая схема

```
Браузер
  │
  ├─ /                 → nginx → dist/index.html (SPA fallback)
  ├─ /assets/*         → nginx → dist/assets (immutable, 1y)
  ├─ /img/*            → nginx alias → public/img (см. прим. ниже)
  │
  └─ /api/*.php        → nginx
                          ├─ location = /api/<точный файл>  → Go API  127.0.0.1:8081
                          └─ location ~ ^/api/(.+\.php)$    → PHP-FPM unix:/run/php/php8.3-fpm.sock
                                                │
                                                └──────────→ PostgreSQL corporate_portal
```

Источники: `deploy/nginx-corporate.admsr.ru.conf`,
`deploy/nginx-go-api-wave2.conf` (подключается через `include` в строке 281 основного конфига).
**[ПОДТВЕРЖДЕНО]**

> **Примечание про `/img/`.** В прод-конфиге alias указывает на
> `/var/www/corporate.admsr.ru/public/img/` (`nginx-corporate.admsr.ru.conf:297-301`),
> а README описывает `/var/lib/corporate-app/uploads/img/`. Комментарий в конфиге
> объясняет это состоянием тестового стенда. Расхождение зафиксировано в
> [known-issues.md](known-issues.md). **[ПОДТВЕРЖДЕНО]**

## Слои и границы

### 1. Оболочка приложения

| Файл | Ответственность |
|------|-----------------|
| `src/main.js` | Создание приложения, подключение Pinia / Router / Nuxt UI, применение темы, масштаба и цветовой схемы **до** монтирования |
| `src/App.vue` | Основной layout: `UApp` → `UDashboardGroup` → `AppAside` + `UDashboardPanel`(`AppHeader` + `RouterView`); глобальный поиск; анимация переключения темы через View Transitions |
| `src/AppKiosk.vue` | Отдельный layout киоска (фиксированный холст 9:16), своя шапка/подвал, авто-тема по расписанию |

`App.vue` выбирает между «голым» `RouterView` и dashboard-каркасом по метаданным
маршрута: `kiosk`, `layout === 'auth'`, `public` — `src/App.vue:96-98`. **[ПОДТВЕРЖДЕНО]**

### 2. Маршрутизация и гварды

`src/router/index.js` — единственный файл роутинга. Все компоненты страниц
импортируются статически (без lazy-loading). **[ПОДТВЕРЖДЕНО]**

Метаданные маршрута:

| meta | Смысл | Где обрабатывается |
|------|-------|--------------------|
| `title` | Заголовок вкладки и крошки | `router.afterEach`, `usePortalBreadcrumbs` |
| `public` | Без авторизации (`/t/:token`) | `beforeEach:212` |
| `layout: 'auth'` | Экраны входа/онбординга | `beforeEach:218` |
| `kiosk` | Ветка киоска | `beforeEach:215` |
| `requiresAdmin` | Только суперадмин **и** включённый UI-тоггл роли | `beforeEach:259-265` |
| `requiresSection` | Право на раздел портала | `beforeEach:268-274` |

Особенность киоска: находясь внутри `/kiosk`, переход на «обычный» маршрут
перенаправляется на киоск-эквивалент через таблицу `kioskRouteNameByName`
**до** проверки авторизации — `src/router/index.js:222-230`. **[ПОДТВЕРЖДЕНО]**

### 3. Слой данных (composables)

Основной паттерн: **composable-синглтон с модульным состоянием**, а не Pinia.
`ref`-ы объявлены на уровне модуля вне функции, поэтому состояние общее для всех
вызовов. Примеры: `useNewsData`, `usePortalServices`, `useSectionAccess`,
`useCoursesStore`, `useTestsStore`. **[ПОДТВЕРЖДЕНО]**

Типовая форма:

```ts
const rows = ref<T[]>([]);
const loading = ref(false);
const loaded = ref(false);
let loadPromise: Promise<void> | null = null;

export function useXData() {
  async function ensureLoaded() { /* однократная загрузка, дедупликация промиса */ }
  async function reload() { /* принудительно */ }
  return { rows, loading, ensureLoaded, reload /* + мутации */ };
}
```

Pinia используется точечно:

| Стор | Файл | Зачем |
|------|------|-------|
| `newsFeed` | `src/stores/newsFeed.ts` | Кэш страниц ленты + позиция скролла между переходами |
| `tests` (второй модуль форм) | `src/tests/store.ts` | Состояние конструктора форм киоск-ветки |

Плюс два не-Pinia стора на `ref`: `src/stores/role.js` (роль в localStorage) и
`src/stores/absenceJournal.js`.

#### Пагинация лент

`src/composables/useCursorFeed.ts` — универсальный курсорный бесконечный список
(`buildUrl(cursor)` + `mapItem` + `getId`, дедупликация, отмена через
`AbortController`). Поверх него — `useNewsFeed`, ленты мероприятий и галереи.
Триггер догрузки — `useFeedSentinel` (IntersectionObserver). **[ПОДТВЕРЖДЕНО]**

### 4. Транспорт до API

Три разных способа обращения к API сосуществуют. **[ПОДТВЕРЖДЕНО]**

| Способ | Где | Авторизация |
|--------|-----|-------------|
| `apiSessionFetch` / `apiSessionUpload` — `src/composables/useAuthSession.ts` | Курсы, портальные сервисы, группы, обратная связь, журнал отсутствия (мутации) | `Authorization: Bearer <token>` + `X-Session-Token` |
| Голый `fetch('/api/…')` | Новости, мероприятия, галерея, профиль, пользователи, `useTestsStore` | httpOnly-cookie `corp_session` (same-origin) |
| `jsonFetch` — `src/tests/api.ts` | Второй модуль форм (`/kiosk/tests`) | cookie + заголовок `X-User-Id` (сервером **игнорируется**, см. SEC-007) |

Единый конверт ответа со стороны Go — `backend/internal/httpx/json.go`:

```json
{ "success": true|false, "message": "…", "data": … }
```

### 5. Backend (Go)

```
backend/
  cmd/api/main.go          — сборка зависимостей и таблица маршрутов (ServeMux)
  internal/
    config/                — Config из env + LoadDotEnv
    db/                    — пул pgx
    auth/
      session.go           — сессии: создание, извлечение токена, CurrentUser, Require*
      permissions.go       — PortalSections, CanEditSection, категории курсов
    handlers/              — по одному файлу на предметную область
    httpx/                 — конверт JSON, CORS
    courses/, tests/       — общая доменная логика, переиспользуемая хендлерами
    media/                 — обработка изображений
```

Маршрутизация плоская: `mux.Handle("/api/<файл>.php", h)` — без middleware-цепочек.
Проверка прав вызывается **внутри** каждого метода хендлера через
`requireUser` / `requireAdmin` / `requireSection` (`backend/internal/handlers/section.go`).
**[ПОДТВЕРЖДЕНО]**

Единственный глобальный middleware — `httpx.CORS` (`main.go:135`), он разрешает
origin'ы Vite (`localhost:5173/5174`). **[ПОДТВЕРЖДЕНО]**

### 6. Backend (PHP, легаси)

`api/` — 87 файлов, каждый самодостаточный скрипт. Используется как fallback для
эндпоинтов, не покрытых allowlist'ом nginx.

Эндпоинты, существующие **только** в PHP (в Go-mux их нет):
`auth_context.php`, `course_file.php`, `course_history.php`,
`course_assignment_cancel.php`, `course_assignments_list.php`,
`course_materials_order.php`, `courses_archive.php`, `courses_duplicate.php`,
`courses_readiness.php`. **[ПОДТВЕРЖДЕНО]**

Эндпоинт, существующий **только** в Go: `portal_services.php` (PHP-файла нет) —
поэтому раздел «Сервисы» работает лишь при запущенном Go API. **[ПОДТВЕРЖДЕНО]**

## Границы модулей: что где менять

| Хочу поменять | Иду в |
|---------------|-------|
| Пункт бокового меню | `src/composables/usePortalNavigation.ts:useSidebarNavItems` |
| Хлебные крошки вложенной страницы | `src/composables/usePortalNavigation.ts:BREADCRUMB_PARENTS` + `BREADCRUMB_ICONS` |
| Новый маршрут | `src/router/index.js` (+ `kioskRouteNameByName`, если нужен киоск-аналог) |
| Токен темы (цвет/шрифт/радиус) | `src/composables/useUiTheme.ts` + `src/assets/tailwind.css` |
| Глобальные настройки Nuxt UI | `vite.config.js` → `ui({ ui: { … } })` |
| Права на раздел | `src/pages/Admin/portalSections.ts` **и** `backend/internal/auth/permissions.go:PortalSections` (списки нужно держать синхронными вручную) |
| Категорию курса | `src/pages/Courses/courseCategories.ts` **и** `backend/internal/auth/permissions.go:CourseCategories` |
| Каталог сервисов (fallback) | `src/composables/usePortalServices.ts:FALLBACK_PORTAL_SERVICES` + сид в `db/migration/V10__portal_services.sql` |
| Новый Go-эндпоинт | хендлер в `backend/internal/handlers/`, регистрация в `cmd/api/main.go`, прокси в `vite.config.js`, `location =` в `deploy/nginx-go-api-wave2.conf` |

> **Дублирование списков — реальный источник ошибок.** `PORTAL_SECTIONS` (фронт) и
> `PortalSections` (Go) — два независимых литерала; на 2026-09-22 они совпадают
> (7 ключей: `news`, `events`, `gallery`, `courses`, `tests`, `absence_journal`,
> `birthdays`). То же с категориями курсов (2 значения). **[ПОДТВЕРЖДЕНО]**

## Что проверять после изменений

| Изменение | Проверить |
|-----------|-----------|
| Новый/переименованный маршрут | крошки (`BREADCRUMB_PARENTS`, `BREADCRUMB_ICONS`), сайдбар, `PORTAL_SERVICE_CATALOG`, `PAGE_ITEMS` в глобальном поиске, киоск-таблицу |
| Новый Go-эндпоинт | `vite.config.js` (dev-прокси), nginx allowlist, наличие guard'а `require*` |
| Изменение прав | оба списка секций (фронт + Go), гейтинг контролов в UI, реальный ответ API |
| Изменение темы | обе цветовые схемы, все 4 шрифта, 5 радиусов, 5 уровней масштаба |
| Изменение схемы БД | новая миграция `V<N>__*.sql` (идемпотентная), Go-хендлеры, PHP-аналог (если он в бою) |
