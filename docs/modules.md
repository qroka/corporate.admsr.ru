# Модули и пользовательские сценарии

Для каждого модуля: где страница, где логика данных, какие эндпоинты, что
проверить после изменения.

---

## Вход и сессия

**Сценарий.** Логин/пароль → `POST /api/auth.php` → профиль + `sessionToken` в
`localStorage`, httpOnly-cookie `corp_session` ставит API → редирект на `/welcome`
(если профиль неполный) или `/`.

| Что | Где |
|-----|-----|
| Экран | `src/pages/login.vue` (единственная форма с Zod-схемой и `UForm :schema`) |
| Клиентская сессия | `src/composables/useAuthSession.ts` |
| Фоновая активность / авто-логаут | `src/composables/useSessionActivity.ts` |
| Гвард | `src/router/index.js:210-294` |
| Онбординг | `src/composables/useOnboarding.ts`, `src/pages/OnboardingPage.vue` |
| Backend | `backend/internal/handlers/auth.go`, `backend/internal/auth/session.go` |

Эндпоинты: `auth.php`, `logout.php`, `check-auth.php`, `heartbeat.php`,
`session_bootstrap.php`.

**После изменения проверить:** вход, выход, поведение при истёкшей сессии,
`/welcome` для нового пользователя, что киоск по-прежнему открывается без входа.

Детали — [auth-and-permissions.md](auth-and-permissions.md).

---

## Рабочий стол

**Сценарий.** Приветствие → плитки сервисов → лента новостей → правая колонка
(календарь, дни рождения, журнал отсутствия).

| Что | Где |
|-----|-----|
| Страница | `src/pages/HomePage.vue` (693 строки) |
| Виджеты | `src/components/home/HomeNewsCard.vue`, `HomeCalendarWidget.vue`, `HomeAbsenceWidget.vue` |
| Виджет обучения | `src/pages/Courses/components/LearningHomeWidget.vue` |
| Лента | `src/composables/useNewsFeed.ts` + `src/stores/newsFeed.ts` |
| Плитки сервисов | `src/composables/usePortalServices.ts` |

Макет: при наличии контента правой колонки — `xl:grid-cols-[minmax(0,1fr)_420px]`,
иначе одна колонка с `max-w-4xl` (`HomePage.vue:388-393`). **[ПОДТВЕРЖДЕНО]**

---

## Новости

| Что | Где |
|-----|-----|
| Лента | `src/pages/News/NewsPage.vue` |
| Карточка | `src/pages/News/NewsDetailsPage.vue` |
| Данные (полный список) | `src/composables/useNewsData.ts` |
| Данные (курсорная лента) | `src/composables/useNewsFeed.ts`, `useCursorFeed.ts` |
| Реакции | `src/composables/useNewsReactions.ts` |
| Редактор (TipTap через `UEditor`) | `src/composables/newsEditor*.ts` (5 файлов) |

Эндпоинт: `news.php` (GET публичный; POST/PUT/DELETE — право секции `news`,
`backend/internal/handlers/news.go`).

Реакции: счётчики уходят на сервер (`?action=like`, `?action=view`), а «я уже
лайкнул / просмотрел» хранится локально в `news-likes:v1` и `news-viewed:v1` —
`useNewsReactions.ts:4-5,81-87`. **[ЧАСТИЧНО]**

Индекс под курсорную пагинацию: `db/migration/V8__news_feed_index.sql`.

---

## Мероприятия

| Что | Где |
|-----|-----|
| Афиша | `src/pages/Events/EventsPage.vue` |
| Карточка | `src/pages/Events/EventDetailsPage.vue` |
| Данные | `src/composables/useEventsData.ts` |

Эндпоинт: `events.php` (GET публичный, мутации — секция `events`).
Связь с альбомом галереи: `events.album_id` — `db/migration/V7__events_gallery_album.sql`.

Отметка «пойду» (RSVP) хранится **только в localStorage** (`events-rsvp:v1`,
`useCalendarFeed.ts:37,125-132`). На сервер не уходит. **[ЧАСТИЧНО]**

Генерация WebP-обложек: `npm run images:webp` → `scripts/generate-events-webp.mjs`.

---

## Фотогалерея

| Что | Где |
|-----|-----|
| Альбомы | `src/pages/Gallery/GalleryPage.vue` |
| Альбом | `src/pages/Gallery/GalleryAlbumPage.vue` |
| Данные | `src/composables/useGalleryData.ts` |

Эндпоинты: `gallery.php`, `gallery_base.php` (мутации — секция `gallery`,
`backend/internal/handlers/gallery.go:109,146,204,436,506`).

---

## Календарь

| Что | Где |
|-----|-----|
| Страница | `src/pages/CalendarPage.vue` (1141 строка) |
| Агрегатор источников | `src/composables/useCalendarFeed.ts` |

Пять источников (`CalendarSource`): `event`, `meeting`, `birthday`, `learning`,
`personal` — `useCalendarFeed.ts:5`. Личные записи (`personal`) и встречи
пользователя хранятся в localStorage `portal-calendar-local:v1`
(`useCalendarFeed.ts:36,106-123`) — серверного хранилища нет. **[ЧАСТИЧНО]**

Единственное место в проекте, где брейкпоинты читаются из JS:
`useMediaQuery('(min-width: 1024px)')` и `768px` — `CalendarPage.vue:38-39`.

---

## Журнал отсутствия

| Что | Где |
|-----|-----|
| Страница | `src/pages/AbsenceJournalPage.vue` (1992 строки — самый крупный файл фронта) |
| Флаг «есть активное отсутствие» | `src/stores/absenceJournal.js` (localStorage `absence-journal:has-active`, синхронизация между вкладками через событие `storage`) |
| Backend | `backend/internal/handlers/absence.go` |

Эндпоинт: `absence_journal.php`. GET — любой авторизованный (`requireUser`),
POST/PUT/DELETE — секция `absence_journal` (закрыто в рамках SEC-009).

Мутации идут через `apiSessionFetch` (Bearer), а контролы правки гейтятся
`canEditSection('absence_journal')`.

---

## Профиль и стена

| Что | Где |
|-----|-----|
| Страница | `src/pages/ProfilePage.vue` (822 строки) |
| Отображаемое имя/аватар в шапке | `src/composables/useProfileDisplay.ts`, `useHeaderUser.ts` |
| Стена постов | `src/composables/useProfileWall.ts` |
| Компоненты стены | `src/components/profile/ProfileCreatePost.vue`, `ProfileWallPost.vue` |
| Аватары по умолчанию | `src/constants/profileAvatars.ts` |

Сохранение карточки: `POST /api/profile.php` — реальная запись в БД
(`ProfilePage.vue:293-310`).

**Стена постов работает только в localStorage** (`profile-wall-posts:v2`,
`useProfileWall.ts:12,42,67`): ни одного обращения к API. Посты не видны
другим пользователям и теряются при очистке браузера. **[ЧАСТИЧНО]**

---

## Дни рождения

| Что | Где |
|-----|-----|
| Данные | `src/composables/useBirthdayColleagues.ts` |
| Backend | `backend/internal/handlers/birthdays.go` |

Эндпоинт: `birthdays.php`. Источник данных — `*.xlsx` в каталоге `BIRTHDAYS_DIR`
или `${UPLOAD_DIR}/birthdays_xlsx` (`birthdays.go:51`), парсинг через
`xuri/excelize`. Право на редактирование — секция `birthdays`.

---

## Сервисы

| Что | Где |
|-----|-----|
| Страница-каталог | `src/pages/ServicesPage.vue` (544 строки, включая админ-редактор) |
| Данные | `src/composables/usePortalServices.ts` |
| Каталог внутренних страниц | `src/pages/Services/portalServiceCatalog.ts` |
| Backend | `backend/internal/handlers/portal_services.go` |
| Таблица | `db/migration/V10__portal_services.sql` |

Сервис бывает `internal` (путь внутри SPA) или `external` (внешний URL).
Включённые внутренние сервисы становятся детьми пункта «Сервисы» в сайдбаре —
`usePortalNavigation.ts:useSidebarNavItems`.

Эндпоинт `portal_services.php` существует **только в Go**; PHP-реализации нет.
При недоступном API используется `FALLBACK_PORTAL_SERVICES` (журнал отсутствия,
формы, заявки) — `usePortalServices.ts:30-64`. **[ПОДТВЕРЖДЕНО]**

---

## Формы и тесты — две параллельные реализации

Это самое неочевидное место проекта. **[ПОДТВЕРЖДЕНО]**

### A. Легаси-модуль — то, что видит сотрудник на `/tests`

| Что | Где |
|-----|-----|
| Точка входа | `src/pages/TestsBlankPage.vue` (маршрут `/tests`) |
| Конструктор / прохождение / статистика | `src/pages/Tests/TestBuilder.vue`, `TestRunner.vue`, `StatsDetail.vue`, `QuestionsBuilder.vue`, `StatChart.vue` |
| Модель формы | `src/pages/Tests/testForm.ts` (числовой `id`) |
| Типы вопросов | `src/pages/Tests/questionTypes.ts` — 9 типов: `single`, `multiple`, `dropdown`, `text`, `textarea`, `scale`, `yesno`, `number`, `date` |
| Стор | `src/composables/useTestsStore.ts` |
| Публичная ссылка | `src/pages/TestLinkPage.vue` (`/t/:token`) |
| Агрегация статистики | `src/composables/useTestStats.ts` |
| Backend | `backend/internal/handlers/tests_routes.go`, `backend/internal/tests/` |

Эндпоинты: `tests_list/save/publish/unpublish/delete/direct/submit/stats/participant/by_token.php`.

Клиент до сих пор кладёт `userId` в тело запроса (`useTestsStore.ts:37,53,…`),
но сервер эту величину **игнорирует** и берёт личность только из сессии
(исправление SEC-008). Рудимент не удалён. **[ПОДТВЕРЖДЕНО]**

### B. Второй модуль — доступен только на `/kiosk/tests`

| Что | Где |
|-----|-----|
| Точка входа | `src/pages/TestsPage.vue` (маршрут `/kiosk/tests`) |
| Компоненты | `src/components/tests/FormBuilder.vue`, `FormPreview.vue`, `FormTake.vue`, `FormReport.vue`, `FormsList.vue`, `QuestionEditor.vue` |
| Типы | `src/tests/types.ts` (UUID) |
| Zod-схемы | `src/tests/schemas.ts` (используются в `FormBuilder.vue:5`) |
| API-обёртки | `src/tests/api.ts` |
| Pinia-стор | `src/tests/store.ts` |
| Backend | `backend/internal/handlers/forms.go` |

Эндпоинты: `forms.php`, `forms_list/publish/submit/report/archive/delete.php`.
Таблицы `forms`, `questions`, `question_options`, `form_responses`,
`response_answers` (V1), отдельные от `test_forms`/`test_questions` (V2).

> Роль каждого модуля и планы по их объединению в коде не зафиксированы.
> **[НЕИЗВЕСТНО]** — вопрос владельцу, см. [known-issues.md](known-issues.md).

---

## Учебные курсы (LMS)

| Что | Где |
|-----|-----|
| Админ-страницы | `src/pages/Courses/admin/` (12 экранов) |
| Страницы сотрудника | `src/pages/Courses/employee/` (5 экранов) |
| Общие компоненты | `src/pages/Courses/components/` |
| Стор | `src/composables/useCoursesStore.ts` (565 строк) |
| Сертификат PDF | `src/pages/Courses/certificatePdf.ts` (jsPDF) |
| Готовность курса | `src/pages/Courses/courseReadiness.ts` |
| Следующее действие | `src/pages/Courses/followNextAction.ts` |
| Категории | `src/pages/Courses/courseCategories.ts` (2 значения) |
| Крошки | `src/pages/Courses/useAdminCoursePortalBreadcrumbs.ts` |
| Backend | `backend/internal/handlers/courses.go`, `attempt_review.go`, `backend/internal/courses/` |

Иерархия: курс → версия → темы → материалы / тесты; назначение создаёт
enrollment; прохождение с последовательной разблокировкой и снимком в
`course_completions`. Подробно — [courses-architecture.md](courses-architecture.md).

Экспорт результатов в Excel — `CourseResultsPage.vue` (SheetJS).
Флаг генерации сертификата — `db/migration/V9__course_certificate.sql`.

Smoke: `npm run test:courses`.

---

## Администрирование

| Что | Где |
|-----|-----|
| Дэшборд | `src/pages/Admin/AdminDashboardPage.vue` (маршрут `/admin`, `requiresAdmin`) |
| Разделы прав | `src/pages/Admin/portalSections.ts` |
| Панель ОФО | `src/components/AdminOfoPanel.vue` |
| Пользователи | `src/composables/useUsersData.ts` |
| Группы доступа | `src/composables/useGroupsData.ts` |
| ОФО | `src/composables/useOfoData.ts`, `useOfoTree.ts`; компоненты `OfoSelect.vue`, `OfoMultiSelect.vue` |

Эндпоинты: `users.php` (только админ), `portal_groups.php`, `ofo*.php`.

BUG-001/BUG-002 (откат оптимистичных обновлений при ошибке сервера) исправлены —
см. [PROJECT_AUDIT.md](PROJECT_AUDIT.md).

---

## Глобальный поиск

| Что | Где |
|-----|-----|
| Логика | `src/composables/usePortalGlobalSearch.ts` (436 строк) |
| Подключение | `src/App.vue:19-28` (`UDashboardSearch`) |
| Горячая клавиша | `src/App.vue:76-83` — Ctrl/⌘+K по `e.code === 'KeyK'`, работает на русской раскладке |

Группы результатов: «Разделы» (статический список), «Сотрудники», «Новости»,
«Мероприятия», «Фотогалерея», «Формы», «Сервисы», «Обучение», «Документация».
Фильтрация на стороне Fuse.js (`:fuse="{ resultLimit: 48 }"`), группы помечены
`ignoreFilter: true` — фильтрацию делает сам composable.

> `UDashboardSearch` получает `shortcut="ctrl_shift_alt_f12"` (`App.vue:21`) —
> встроенный шорткат намеренно «уведён» в нереальную комбинацию, чтобы работал
> собственный обработчик Ctrl+K. Комментария об этом в коде нет. **[ПОДТВЕРЖДЕНО]**

---

## Киоск

| Что | Где |
|-----|-----|
| Layout | `src/AppKiosk.vue` |
| Шапка / подвал | `src/components/KioskHeader.vue`, `KioskAside.vue` |
| Главная | `src/pages/Kiosk/KioskHomePage.vue` |
| Авто-тема | `src/composables/useColorModeSchedule.ts` |
| Скрипты запуска на Windows | `kiosk/*.cmd` |

Холст: `min(1080px,100vw) × min(1920px,100vh)`, `aspect-9/16` (`AppKiosk.vue:3`).
Тема по расписанию: 00:00–14:59 светлая, 15:00–23:59 тёмная
(`useColorModeSchedule.ts:8-10`).

Ограничения: авторизация не требуется; роль принудительно `user`;
`requiresAdmin` / `requiresSection` внутри киоска → редирект на `/kiosk`
(`router/index.js:266-277`).

> По решению владельца (PROJECT_AUDIT §8) киоск **не трогаем**.

---

## Загрузка файлов

| Что | Где |
|-----|-----|
| Изображения | `POST /api/Upload/upload.php` → `backend/internal/handlers/gallery.go` (`Upload`), `backend/internal/media/webp.go` |
| Материалы курсов | `POST /api/course_materials_upload.php` → `coursesH.MaterialsUpload` |

Каталоги: `${UPLOAD_DIR}/img/FullPic`, `${UPLOAD_DIR}/img/SmallPic`,
`${UPLOAD_DIR}/courses/{courseId}/`.

Известное отличие Go от PHP: изображения сохраняются как JPEG, без libwebp
(`backend/README.md`, раздел «Известные отличия»). **[ПОДТВЕРЖДЕНО]**

Выдача файлов курса через `GET /api/course_file.php` — **только PHP**, в Go-mux
эндпоинта нет (`backend/cmd/api/main.go`). **[ПОДТВЕРЖДЕНО]**
