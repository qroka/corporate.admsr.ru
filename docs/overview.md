# Обзор проекта

## Что это

Внутренний корпоративный портал Администрации (ADMSR). Единая точка входа для
сотрудников: новости, мероприятия, фотогалерея, календарь, журнал отсутствия,
профиль, формы/тесты, учебные курсы, справочники отделов.

Продакшен-домен (из nginx-конфига): `corporate.admsr.ru`, дополнительно
`newcorp.admsr.ru` — `deploy/nginx-corporate.admsr.ru.conf:25,34`. **[ПОДТВЕРЖДЕНО]**

Отдельный режим — **киоск** (`/kiosk`): тот же контент на общих экранах, без
входа и без админ-функций, вертикальный макет 9:16 — `src/AppKiosk.vue:3`,
`src/router/index.js:51-74`. **[ПОДТВЕРЖДЕНО]**

## Кто пользуется

| Аудитория | Как определяется в коде |
|-----------|-------------------------|
| Сотрудник | любая активная запись `user_info` с валидной сессией |
| Редактор раздела | членство в группе портала с правом на `section_key` — `backend/internal/auth/permissions.go:CanEditSection` |
| Суперадминистратор | `user_info.user_group = 'admin'` — `backend/internal/auth/session.go:IsAdmin` |
| Посетитель киоска | без авторизации, роль принудительно `user` — `src/router/index.js:245-251` |

Подробно — [auth-and-permissions.md](auth-and-permissions.md).

## Стек и фактические версии

Версии сняты из `node_modules` и `backend/go.mod` на 2026-09-22. **[ПОДТВЕРЖДЕНО]**

| Слой | Технология | Версия | Где |
|------|-----------|--------|-----|
| UI-фреймворк | Vue 3 (SFC, `<script setup>`) | 3.5.29 | `src/` |
| Сборка | Vite | 7.3.1 | `vite.config.js` |
| UI-библиотека | Nuxt UI (Vue-режим) | 4.8.0 | `vite.config.js`, `src/main.js` |
| CSS | Tailwind CSS | 4.3.0 | `src/assets/tailwind.css` |
| Роутинг | Vue Router | 4.6.4 | `src/router/index.js` |
| Стор | Pinia | 3.0.4 | `src/stores/`, `src/tests/store.ts` |
| Валидация (FE) | Zod | 4.3.6 | `src/pages/login.vue`, `src/tests/schemas.ts` |
| Утилиты | @vueuse/core | 14.3.0 | `src/pages/CalendarPage.vue` |
| Графики | chart.js | 4.5.1 | `src/components/tests/FormReport.vue` |
| PDF | jspdf | 4.2.1 | `src/pages/Courses/certificatePdf.ts` |
| Excel | xlsx (SheetJS) | 0.18.5 | `src/pages/Courses/admin/CourseResultsPage.vue` |
| Drag&drop | vuedraggable | 4.1.0 | `src/components/tests/FormBuilder.vue` |
| Rich text | @tiptap/* (через Nuxt UI `UEditor`) | 3.22.x | `src/composables/newsEditor*.ts` |
| API (новый) | Go, `net/http` + `ServeMux`, pgx/v5 | go 1.26.5 | `backend/` |
| API (легаси) | PHP 8.3 + PDO (PHP-FPM) | — | `api/` (87 файлов) |
| БД | PostgreSQL 14+, база `corporate_portal` | — | `db/migration/` (V1…V10) |
| Веб-сервер | nginx (SPA + allowlist в Go + fallback в PHP) | — | `deploy/` |

### Заявлено в зависимостях, но не используется

- `@heroicons/vue` 2.2.0 — ни одного импорта в `src/`. Иконки берутся из Lucide
  через `UIcon`. **[ПОДТВЕРЖДЕНО]** (см. [known-issues.md](known-issues.md))

### Упомянуто в README, но отсутствует в репозитории

- `python-ai/` (FastAPI + Ollama, AI-ассистент). Каталога нет; маршрут
  `/chatbot` редиректит на `/feedback` — `src/router/index.js:97`.
  **[ПОДТВЕРЖДЕНО]**

## Что реально работает

### Реализовано и подтверждено

| Раздел | Маршрут | Где код |
|--------|---------|---------|
| Рабочий стол | `/` | `src/pages/HomePage.vue` |
| Новости (лента, карточка, реакции) | `/news`, `/news/:id` | `src/pages/News/` |
| Мероприятия | `/events`, `/events/:id` | `src/pages/Events/` |
| Фотогалерея | `/gallery`, `/gallery/:albumId` | `src/pages/Gallery/` |
| Календарь | `/calendar` | `src/pages/CalendarPage.vue` |
| Журнал отсутствия | `/absence-journal` | `src/pages/AbsenceJournalPage.vue` |
| Профиль + стена постов | `/profile` | `src/pages/ProfilePage.vue` |
| Сервисы (каталог) | `/services` | `src/pages/ServicesPage.vue` |
| Формы / тесты | `/tests` | `src/pages/TestsBlankPage.vue` + `src/pages/Tests/` |
| Прохождение по публичной ссылке | `/t/:token` | `src/pages/TestLinkPage.vue` |
| Обучение (сотрудник) | `/courses`, `/courses/:enrollmentId`, … | `src/pages/Courses/employee/` |
| Обучение (администратор) | `/admin/courses/**` | `src/pages/Courses/admin/` |
| Обратная связь | `/feedback` | `src/pages/FeedbackPage.vue` |
| Документация портала (для пользователя) | `/documentation` | `src/pages/DocumentationPage.vue` |
| Онбординг первого входа | `/welcome` | `src/pages/OnboardingPage.vue` |
| Вход | `/login` | `src/pages/login.vue` |
| Дэшборд администратора | `/admin` | `src/pages/Admin/AdminDashboardPage.vue` |
| Киоск | `/kiosk/**` | `src/AppKiosk.vue`, `src/pages/Kiosk/` |
| Глобальный поиск (Ctrl/⌘+K) | — | `src/composables/usePortalGlobalSearch.ts` |
| Темизация (цвет/шрифт/радиус/масштаб) | — | `src/composables/useUiTheme.ts`, `useUiZoom.ts` |

### Реализовано частично

- **Модуль форм существует в двух реализациях.** Основной портал `/tests`
  использует легаси-модуль (`src/composables/useTestsStore.ts` → `/api/tests_*.php`,
  числовые id). Второй модуль (`src/tests/` + `src/components/tests/`,
  `/api/forms*.php`, UUID, Zod-схемы) подключён **только** к киоск-маршруту
  `/kiosk/tests` через `src/pages/TestsPage.vue`. **[ПОДТВЕРЖДЕНО]**
  Причина такого состояния в коде не зафиксирована. **[НЕИЗВЕСТНО]**
- **Онбординг**: определяется по незаполненным ОФО / должности / аватару —
  `src/composables/useOnboarding.ts:isProfileIncomplete`. **[ПОДТВЕРЖДЕНО]**

### Заглушки (экран есть, логики нет)

| Раздел | Маршрут | Файл |
|--------|---------|------|
| Заявки | `/applications` | `src/pages/ApplicationsPage.vue` — только `UEmpty` «Раздел в разработке» |
| Кадровый резерв | `/personnel-reserve` | `src/pages/PersonnelReservePage.vue` — `SectionInDevelopment` |
| Отдел развития и мотивации | `/development-motivation` | `src/pages/DevelopmentMotivationDepartmentPage.vue` — `SectionInDevelopment` |
| Кнопка «Уведомления» в шапке | — | `src/components/AppHeader.vue:315-324` — нет обработчика |

### Упомянуто в README, но маршрута нет

`/newcomers` (Новичкам) и `/culture` (Корпоративная культура) в
`src/router/index.js` отсутствуют. `/knowledge-base` редиректит на
`/documentation`, `/courses/history` — на `/courses`, `/tests/old` — на `/tests`.
**[ПОДТВЕРЖДЕНО]** См. [known-issues.md](known-issues.md).

## Ключевая особенность, которую нужно знать сразу

Идёт **параллельная миграция PHP → Go** с сохранением URL-контрактов `/api/*.php`.
nginx точечным allowlist (`location = /api/xxx.php`) отправляет конкретные
эндпоинты в Go (`127.0.0.1:8081`), всё остальное — в PHP-FPM. Для многих
эндпоинтов **существуют две реализации**, и они не идентичны.

Решение владельца от 2026-09-21 (зафиксировано в `docs/PROJECT_AUDIT.md`, §8):
**PHP-слой `api/` больше не изменять**, киоск не трогать, все правки backend —
только в Go. **[ПОДТВЕРЖДЕНО]**

Подробности — [architecture.md](architecture.md) и [api.md](api.md).
