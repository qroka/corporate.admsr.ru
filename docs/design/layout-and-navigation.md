# Каркас и навигация

## Каркас приложения

`src/App.vue` выбирает один из двух режимов отрисовки. **[ПОДТВЕРЖДЕНО]**

```
route.meta.kiosk | layout==='auth' | public   →  голый <RouterView />
иначе                                          →  dashboard-каркас
```

### Dashboard-каркас

```
<div class="app-root h-dvh overflow-hidden">
  <UApp :locale="ru">                                  ← русская локаль Nuxt UI
    <UDashboardGroup unit="px" storage-key="portal-dashboard" class="h-dvh w-full">
      <AppAside />                                     ← сайдбар
      <UDashboardSearch … />                           ← глобальный поиск (оверлей)
      <UDashboardPanel id="portal-main" class="bg-default">
        <template #header><AppHeader /></template>
        <template #body><RouterView /></template>
      </UDashboardPanel>
    </UDashboardGroup>
  </UApp>
</div>
```

Ключевое: `h-dvh overflow-hidden` на корне. **Страница не скроллит документ —
скроллит собственный внутренний контейнер.** Отсюда повсеместные
`h-full min-h-0 overflow-y-auto`. **[ПОДТВЕРЖДЕНО]**

`body` основной панели (`App.vue:32-38`):
`px-4 pt-4 pb-0 sm:px-4 sm:pt-4 sm:pb-0 bg-default overflow-hidden flex flex-col min-h-0`.

Размер и состояние сайдбара сохраняются в `localStorage['portal-dashboard']`.

---

## Сайдбар

`src/components/AppAside.vue`, на базе `UDashboardSidebar`.

| Параметр | Значение |
|----------|----------|
| Ширина по умолчанию | 268px |
| Мин. / макс. | 220px / 400px |
| Свёрнутый | 0px |
| Возможности | `collapsible`, `resizable`, свой `resize-handle` |

Переопределения `:ui` (`AppAside.vue:27-33`):

```
root:   border-0 border-e-0 bg-elevated rounded-panel my-4 ms-4 min-h-0 h-[calc(100dvh-2rem)]
header: h-[60px] shrink-0 px-4
body:   px-4 py-2 flex flex-col gap-4
footer: px-4 pb-4 pt-2 gap-2
```

Визуально это **плавающая панель**: приподнятый фон, скруглённый угол
`rounded-panel`, отступ `my-4 ms-4`, без внешних рамок.

### Шапка сайдбара

Логотип (inline SVG, `viewBox="0 0 48 48"`, `fill-current`, `text-primary`,
`size-7`, `aria-hidden="true"`) + подпись «Корпоративный портал»
(`text-sm font-semibold text-highlighted truncate`). В свёрнутом виде остаётся
только знак. Тот же контур перекрашивает favicon
(`useUiTheme.ts:FAVICON_PATH`). **[ПОДТВЕРЖДЕНО]**

> Кнопка `i-lucide-chevrons-up-down` справа в шапке сайдбара
> (`AppAside.vue:62-72`) обработчика не имеет. **[ПОДТВЕРЖДЕНО]**

### Пункты меню

`src/composables/usePortalNavigation.ts:useSidebarNavItems` — два
`UNavigationMenu` с `orientation="vertical"`.

**Основные (`mainItems`):**

| Пункт | Иконка | Путь |
|-------|--------|------|
| Рабочий стол | `i-lucide-grip` | `/` |
| Новости | `i-lucide-newspaper` | `/news` |
| Календарь | `i-lucide-calendar-days` | `/calendar` |
| **Сервисы** | `i-lucide-layout-grid` | `/services`, `defaultOpen: true`, дети — включённые внутренние сервисы из БД |
| Мероприятия | `i-lucide-calendar` | `/events` |
| Фотогалерея | `i-lucide-images` | `/gallery` |
| Обучение | `i-lucide-graduation-cap` | `/courses` |

**Нижние (`footerItems`)**, прижаты `mt-auto`, с `trailingIcon: 'i-lucide-arrow-up-right'`:
Обратная связь (`/feedback`), Документация (`/documentation`).

Активность считается двумя способами: по имени маршрута (`isRouteMatch`,
учитывает префиксы `news-`, `admin-news`) и по пути (`pathActive`, для
динамических сервисов). **[ПОДТВЕРЖДЕНО]**

---

## Шапка

`src/components/AppHeader.vue`, на базе `UDashboardNavbar`.

```
root:  @container h-[60px] shrink-0 border-0 mx-2 sm:mx-4 mt-4 rounded-panel bg-elevated px-2 sm:px-4
left:  min-w-0 flex-1
right: gap-2 sm:gap-4 shrink-0 min-w-0
```

Та же «плавающая панель», что и сайдбар: `bg-elevated` + `rounded-panel`,
без рамок.

`@container` на корне — шапка является контейнером для container queries
Tailwind 4: ширина шапки зависит от сайдбара (220–400px, сворачивается), и
брейкпоинты окна (`lg:`) этого не учитывают. См. [decisions.md](../decisions.md),
ADR-027.

**Слева** — хлебные крошки (в слоте `#title`).
**Справа**, слева направо:

| Элемент | Ширина шапки | Поведение |
|---------|--------------|-----------|
| `UDashboardSearchButton` компактный | до 56rem (`@4xl:hidden`) | только иконка; подсказка «Поиск…» + `⌘K` снизу |
| `UDashboardSearchButton` средний | 56–72rem | `w-44`, подпись «Поиск…», `⌘K` |
| `UDashboardSearchButton` полный | от 72rem (`@6xl:inline-flex`) | `w-[340px]`, подпись «Искать сотрудника, памятку, документ...» |
| Кнопка уведомлений | всегда | `i-lucide-bell` в `UTooltip` «Уведомления» — **обработчика нет** (IMP-05) |
| `UDropdownMenu` профиля | всегда | аватар + имя (скрыто до `sm`) + шеврон, поворот на 180° при открытии |

Все контролы шапки — `color="neutral" variant="outline" size="md" class="h-8"`.
**[ПОДТВЕРЖДЕНО — единый паттерн]**

**Иконочная кнопка без видимой подписи** получает и `aria-label`, и
`UTooltip` (`:content="{ side: 'bottom' }"` в шапке, чтобы не уходило за край).
Если кнопка — триггер `UDropdownMenu`, подсказку вешают на `span`-обёртку:
у двух компонентов Nuxt UI не может быть общего `as-child`-триггера.

### Меню профиля

Группы (`AppHeader.vue:114-263`):
1. Метка с аватаром и именем (`type: 'label'`);
2. «Профиль» → `/profile`;
3. «Роль» — `UTabs` Пользователь / Администратор (только при `canToggleAdminRole`);
4. Тема — `UTabs` со слотом `#theme-trailing`: Светлая / Тёмная / Система,
   подписи `sr-only`, видны только иконки;
5. Палитра — выбор primary/neutral, шрифта, радиуса, случайная тема, сброс;
6. «Выйти» (`color: 'error'`).

Приёмы:
- цветные чипы палитры — слот `#chip-leading` с CSS-переменными
  `--chip-light` / `--chip-dark`, которые ссылаются на `--color-<палитра>-500/400`;
- `keepMenuOpen` (`e.preventDefault()`) не даёт меню закрыться при переключении
  темы и роли;
- вкладки внутри меню — `@click.stop`;
- во время загрузки профиля вместо аватара и имени — два `USkeleton`.

**[ПОДТВЕРЖДЕНО]**

---

## Хлебные крошки

`usePortalBreadcrumbs` (`src/composables/usePortalNavigation.ts:237-289`).

Правила:
1. Первый элемент — всегда **«Рабочий стол»** с `i-lucide-grip`.
2. На самом рабочем столе (и на главной киоска) крошка одна, без ссылки, с
   `ui: { linkLeadingIcon: 'text-primary' }`.
3. Промежуточные сегменты берутся из таблицы `BREADCRUMB_PARENTS` по имени
   маршрута; параметры пробрасываются функцией `params(route.params)`.
4. Последний сегмент — текущая страница, **без `to`**, с подсвеченной иконкой
   (`text-primary`).
5. Иконка — из `BREADCRUMB_ICONS`; фоллбэки: `admin-course*`/`course-*` →
   `i-lucide-graduation-cap`, `kiosk-*` → иконка базового маршрута, иначе
   `i-lucide-file`.

Динамические подписи (название альбома, курса) подставляются через
`useBreadcrumbCurrentLabel()` и `useBreadcrumbLabelsByRoute()` — страница
записывает в них значение после загрузки данных. **[ПОДТВЕРЖДЕНО]**

Оформление (`AppHeader.vue`): `linkLabel: text-sm font-medium truncate`,
иконки `size-5 text-muted`, список `flex-nowrap overflow-hidden` — крошки не
переносятся, длинные обрезаются.

**Сворачивание длинного пути** (`visibleBreadcrumbs` в `AppHeader.vue`): если
элементов больше трёх, показываются первый, «…» и два последних. «…» — кнопка
с меню скрытых уровней (переход по ним сохраняется) и подсказкой «Показать
весь путь». Сам `usePortalBreadcrumbs` возвращает полный путь — сворачивание
только визуальное.

**Подсказка у обрезанной подписи**: в слоте `#item-label` подпись обёрнута в
`UTooltip`, который включается, только если текст реально обрезан
(`scrollWidth > clientWidth` при наведении, `markCrumbTruncation`).

**При добавлении вложенного маршрута нужно обновить `BREADCRUMB_PARENTS` и
`BREADCRUMB_ICONS`** — иначе крошка будет называться именем маршрута и получит
иконку-файл.

---

## Каркас страницы

Доминирующий паттерн (`NewsPage`, `EventsPage`, `GalleryPage`, `MyCoursesPage`,
`ApplicationsPage` и др.):

```html
<UMain class="relative w-full h-full min-h-0">
  <div class="flex flex-col gap-6 w-full h-full min-h-0 max-w-[1600px] mx-auto
              overflow-y-auto scrollbar-hide p-px pb-8">
    <UPageHeader headline="…" title="…" description="…" />
    <!-- содержимое -->
  </div>
</UMain>
```

Разбор классов:

| Класс | Зачем |
|-------|-------|
| `h-full min-h-0` | обязательное условие вложенного скролла во flex-контейнере |
| `overflow-y-auto scrollbar-hide` | скроллит страница, а не документ; полоса скрыта |
| `max-w-[1600px] mx-auto` | предельная ширина контента |
| `gap-6` | вертикальный ритм секций |
| `p-px` | 1px, чтобы не обрезались кольца фокуса и `ring-*` у крайних элементов |
| `pb-8` | воздух внизу (основная панель задаёт `pb-0`) |

`<UMain>` использован примерно в 32 файлах. **[ПОДТВЕРЖДЕНО]**

### Заголовок страницы

`UPageHeader` (~21 файл), глобально стилизован в `vite.config.js:20-24`:
`relative border-b border-default py-4`.

Используемые пропсы: `title` (7), `description` (7), `headline` (5) —
надзаголовок вроде «Сервисы».

Варианты:
- `HomePage.vue:375-384` — `:ui` с обнулением рамки и `title: text-2xl font-bold`;
- `SectionInDevelopment.vue:11-16` — `title=""` + слот `#title` с
  `text-4xl font-normal font-unbounded`.

### Секции внутри страницы

```html
<section class="flex flex-col gap-4 w-full" aria-labelledby="home-services-title">
  <div class="flex items-center gap-1 min-w-0">
    <h2 id="home-services-title" class="text-lg font-bold leading-7 text-highlighted">Сервисы</h2>
    <UTooltip text="…"><UButton icon="i-lucide-info" variant="ghost" size="xs" square … /></UTooltip>
  </div>
  …
</section>
```
(`HomePage.vue:395-426`) — заголовок `<h2>` + `aria-labelledby` на секции.

### Сетки

| Контекст | Классы |
|----------|--------|
| Плитки сервисов | `grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-5 gap-3` |
| Рабочий стол с правой колонкой | `grid grid-cols-1 xl:grid-cols-[minmax(0,1fr)_420px] gap-4 items-start` |
| Парная навигация «пред./след.» | `grid grid-cols-1 sm:grid-cols-2 gap-6` |

---

## Экраны без каркаса

### Вход (`layout: 'auth'`)

`src/pages/login.vue`: `min-h-screen flex items-center justify-center bg-(--ui-bg) px-4 py-11`,
заголовок с флюидным `clamp()`, форма в `UCard` шириной `max-w-[384px]`.
Сайдбара и шапки нет. **[ПОДТВЕРЖДЕНО]**

### Публичная страница формы (`meta.public`)

`src/pages/TestLinkPage.vue` — `/t/:token`, без авторизации и без каркаса.

---

## Киоск

`src/AppKiosk.vue`:

```html
<div class="min-h-screen grid place-items-center bg-default">
  <div class="w-[min(1080px,100vw)] h-[min(1920px,100vh)] aspect-9/16 flex flex-col gap-6 p-6">
    <KioskHeader … />
    <main class="flex-1 min-h-0 flex flex-col"><RouterView /></main>
    <aside class="flex-none"><KioskAside … /></aside>
  </div>
</div>
```

Фиксированный **вертикальный** холст 1080×1920 (9:16) по центру экрана.
Навигация — крупные плитки в нижнем `KioskAside` (`text-3xl font-unbounded`,
`p-6`, `rounded-3xl`, иконки `size-12`). Тема переключается по расписанию.

> Киоск — отдельная визуальная система: собственные размеры, радиусы и
> цветовые классы. Правила основного портала на него не распространяются.
> По решению владельца киоск не изменяется (см. [decisions.md](../decisions.md), ADR-002).

---

## Глобальный поиск

`UDashboardSearch` в `App.vue:19-28`:

```
placeholder:          «Поиск по порталу: сотрудники, новости, сервисы…»
shortcut:             ctrl_shift_alt_f12   ← намеренно недостижим, см. ADR-014
:color-mode="false"   ← блок переключения темы в палитре отключён
:preserve-group-order="true"
:fuse="{ resultLimit: 48 }"
```

Открытие — собственный обработчик Ctrl/⌘+K по `e.code === 'KeyK'`
(работает на русской раскладке, перехватывает поиск браузера).

Группы: Разделы, Сотрудники, Новости, Мероприятия, Фотогалерея, Формы, Сервисы,
Обучение, Документация.

---

## Чек-лист новой страницы

1. Маршрут в `src/router/index.js` с `meta.title`.
2. Нужен ли киоск-аналог → `kioskRouteNameByName`.
3. `BREADCRUMB_PARENTS` (если вложенная) и `BREADCRUMB_ICONS`.
4. Каркас: `UMain` + скролл-контейнер по образцу выше.
5. `UPageHeader` с `headline` / `title` / `description`.
6. Состояния загрузки / пустоты / ошибки —
   [states-and-feedback.md](states-and-feedback.md).
7. Нужен ли пункт в сайдбаре (`useSidebarNavItems`), в `PORTAL_SERVICE_CATALOG`,
   в `PAGE_ITEMS` глобального поиска.
8. Проверить на ~375 / 768 / 1280 / 1600 px и в обеих темах.
