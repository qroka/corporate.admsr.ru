# Токены оформления

Источники: `src/assets/tailwind.css`, `src/composables/useUiTheme.ts`,
`src/composables/useUiZoom.ts`, `vite.config.js`,
`node_modules/@nuxt/ui/dist/runtime/index.css`.

---

## 1. Цвет

### Как устроена система

Три уровня. **[ПОДТВЕРЖДЕНО]**

```
1. Палитры Tailwind           --color-emerald-500, --color-zinc-800, …
   (src/assets/tailwind.css, объявлены в :root целиком — комментарий в файле:
    «so they are never tree-shaken by Tailwind»)
        ↓ копируются из JS при применении темы (useUiTheme.applyUiTheme)
2. Палитры Nuxt UI            --ui-color-primary-50…950, --ui-color-neutral-50…950, --ui-primary
        ↓ семантические псевдонимы Nuxt UI
3. Токены интерфейса          --ui-bg, --ui-text-muted, --ui-border, …
   → утилиты bg-default, text-muted, border-default, …
```

**В разметке используется третий уровень.** Прямые цвета Tailwind
(`text-green-600`, `bg-red-500`) — отклонение, см.
[inconsistencies.md](inconsistencies.md).

### Семантические токены (уровень 3)

Определения из `@nuxt/ui/dist/runtime/index.css`. **[ПОДТВЕРЖДЕНО]**

| Утилита | Светлая тема | Тёмная тема | Где применять |
|---------|--------------|-------------|---------------|
| `bg-default` | `#fff` | `neutral-900` | Фон приложения и основной панели |
| `bg-muted` | `neutral-50` | `neutral-800` | Слегка приглушённый фон |
| `bg-elevated` | `neutral-100` | `neutral-800` | Карточки, сайдбар, шапка |
| `bg-accented` | `neutral-200` | `neutral-700` | Выделенный фон, hover |
| `bg-inverted` | `neutral-900` | `#fff` | Контрастные плашки |
| `text-highlighted` | `neutral-900` | `#fff` | Заголовки, ключевые значения |
| `text-default` | `neutral-700` | `neutral-200` | Основной текст |
| `text-toned` | `neutral-600` | `neutral-300` | Вторичный текст |
| `text-muted` | `neutral-500` | `neutral-400` | Подписи, метаданные |
| `text-dimmed` | `neutral-400` | `neutral-500` | Плейсхолдеры, третьестепенное |
| `text-inverted` | `#fff` | `neutral-900` | Текст на инвертированном фоне |
| `border-default` / `ring-default` | `neutral-200` | `neutral-800` | Рамки по умолчанию |
| `border-muted` | `neutral-200` | `neutral-700` | Мягкие разделители |
| `border-accented` | `neutral-300` | `neutral-700` | Активная/hover-рамка |
| `border-inverted` | `neutral-900` | `#fff` | Контрастная рамка |

> **Рамка на приподнятом фоне.** В тёмной теме `bg-elevated` и
> `border-default` / `ring-default` — один и тот же `neutral-800`: рамка или
> разделитель **внутри** карточки с `bg-elevated` не видны. Там используйте
> `border-accented` / `ring-accented` (`neutral-700`), а для hover —
> `bg-accented/40`. На фоне страницы (`bg-default`) `*-default` работает.
> См. [decisions.md](../decisions.md), ADR-022.

Фактическое использование в `src/` (число вхождений):
`text-muted` 238, `text-highlighted` 173, `bg-elevated` 98, `text-dimmed` 74,
`ring-default` 61, `border-default` 47, `text-default` 39, `ring-primary` 32,
`bg-primary` 30, `text-primary` 26, `bg-default` 20. **[ПОДТВЕРЖДЕНО]**

### Статусные цвета

Псевдонимы Nuxt UI и их базовые палитры
(`@nuxt/ui/dist/shared/ui.CoJ8bnb0.mjs:60-70`):

| Псевдоним | Палитра | Использование в проекте |
|-----------|---------|-------------------------|
| `primary` | `emerald` (переопределено в `vite.config.js:10`) | 90 вхождений `color="primary"` |
| `neutral` | `zinc` (переопределено в `vite.config.js:11`) | 279 вхождений — **самый частый** |
| `error` | `red` | 49 |
| `warning` | `yellow` | 29 |
| `info` | `blue` | 5 |
| `success` | `green` | 2 (как `color`), 16 как `text-success` |
| `secondary` | `blue` | не используется |

> **Важно:** переключатель темы меняет **только** `primary` и `neutral`
> (`useUiTheme.applyUiTheme` переопределяет `--ui-color-primary-*` и
> `--ui-color-neutral-*`). `success`/`info`/`warning`/`error` остаются
> зелёным / синим / жёлтым / красным при любой теме. **[ПОДТВЕРЖДЕНО]**

> **Недопустимые значения `color`.** Валидны только `primary`, `secondary`,
> `success`, `info`, `warning`, `error`, `neutral`. В коде встречаются
> `color="red"` (8 раз) и `color="amber"` (3 раза) — такие значения ни с чем не
> сопоставляются. Места перечислены в [inconsistencies.md](inconsistencies.md).
> **[ПОДТВЕРЖДЕНО]**

### Брендовый emerald

`src/assets/tailwind.css`, комментарий «Brand emerald tuned to Figma primary #00DC82»:

| Шаг | Значение |
|-----|----------|
| 50 | `#ecfdf5` |
| 100 | `#d1fae5` |
| 200 | `#a7f3d0` |
| 300 | `#6ee7b7` |
| 400 | `#34d399` |
| **500** | **`#00DC82`** ← бренд |
| 600 | `#00c16a` |
| 700 | `#059669` |
| 800 | `#065f46` |
| 900 | `#064e3b` |
| 950 | `#022c22` |

Остальные палитры заданы в `oklch` по значениям Tailwind.

### Доступные палитры темы

`src/composables/useUiTheme.ts:1-58`. **[ПОДТВЕРЖДЕНО]**

**Primary (18):** `white` (адаптивный), `red`, `orange`, `amber`, `yellow`,
`lime`, `green`, `emerald` ← по умолчанию, `teal`, `cyan`, `sky`, `blue`,
`indigo`, `violet`, `purple`, `fuchsia`, `pink`, `rose`.

**Neutral (9):** `slate`, `gray`, `zinc` ← по умолчанию, `neutral`, `stone`,
`taupe`, `mauve`, `mist`, `olive`.

Последние четыре (`taupe`, `mauve`, `mist`, `olive`) — не стандартные палитры
Tailwind, а собственные шкалы проекта в `oklch` (`tailwind.css`).
**[ПОДТВЕРЖДЕНО]**

### Адаптивный primary «Белый / чёрный»

Псевдопалитра `white` инвертируется по теме: `--color-white-500` = `oklch(14.5% 0 0)`
(почти чёрный) в светлой и `oklch(100% 0 0)` в тёмной (блок `.dark` в
`tailwind.css`). Именно поэтому `applyColorModeToDocument` при каждой смене
схемы повторно вызывает `applyUiTheme` (`useColorMode.ts:10-14`).
**[ПОДТВЕРЖДЕНО]**

### Прочие цветовые константы

| Токен | Значение | Где | Использование |
|-------|----------|-----|---------------|
| `--color-brand` | `#50c891` | `tailwind.css:12` | **0 использований** (`text-brand`/`bg-brand` не встречаются) |
| `--shadow-brand` | многослойное свечение из `--ui-primary` | `tailwind.css:13-17` | 1 файл |
| `EMERALD_500` | `#00DC82` | `useUiTheme.ts:141` | цвет favicon по умолчанию |

---

## 2. Типографика

### Шрифты

Подключены через Google Fonts в `index.html:10-13`: Inter (100–900),
Montserrat (100–900, italic), Unbounded (200–900). **[ПОДТВЕРЖДЕНО]**

Выбор пользователя — `useUiTheme.FONT_OPTIONS`:

| id | Подпись в UI | Стек |
|----|--------------|------|
| `inter` | Inter | `'Inter', system-ui, -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif` ← по умолчанию |
| `unbounded` | Unbounded | `'Unbounded', …` |
| `comic-sans` | Comic Sans | `'Comic Sans MS', 'Comic Sans', 'Chalkboard SE', 'Comic Neue', cursive` |
| `montserrat` | Montserrat | `'Montserrat', …` |

Применяется как `--font-sans` на `<html>` + атрибут `data-ui-font`.

> Comic Sans не подгружается с Google Fonts — берётся системный.
> **[ПОДТВЕРЖДЕНО]**

#### Коррекция весов для Unbounded

`tailwind.css` сдвигает веса вниз, когда выбран Unbounded
(комментарий: «даже Medium (500) выглядит как Bold»):

```css
html[data-ui-font='unbounded'] :where(.font-medium)                       { font-weight: 400; }
html[data-ui-font='unbounded'] :where(.font-semibold)                     { font-weight: 500; }
html[data-ui-font='unbounded'] :where(.font-bold, .font-extrabold, .font-black) { font-weight: 600; }
```

#### Display-гарнитура

`--font-unbounded` (`tailwind.css:7-8`) → утилита `font-unbounded`, 6 файлов.
Применяется к крупным заголовкам страниц:

```html
<h1 class="text-4xl font-normal font-unbounded">{{ title }}</h1>
```
(`src/components/SectionInDevelopment.vue:14`)

> `src/pages/login.vue:79,86` задаёт Unbounded **инлайновым**
> `style="font-family: 'Unbounded', sans-serif;"` вместо утилиты — отклонение.

### Размеры

`body` — `font-family: var(--font-sans)`, `font-weight: 400` (`tailwind.css`).

Фактическая шкала в разметке:

| Класс | Типичное применение |
|-------|---------------------|
| `text-4xl` | Заголовок страницы-заглушки (`font-unbounded`) |
| `text-2xl` / `text-3xl` | Заголовок рабочего стола, крупные значения |
| `text-lg` | Заголовки секций (`<h2>`) |
| `text-base` | Основной текст |
| `text-sm` | Подписи, крошки, пункты меню, кнопки |
| `text-xs` | Метаданные, заголовки колонок таблиц |

На экране входа заголовки заданы через `clamp()`:
`text-[clamp(22px,3.5vw,36px)]` и `text-[clamp(38px,6.5vw,64px)]`
(`login.vue:79,86`) — единственное место с флюидной типографикой.
**[ПОДТВЕРЖДЕНО]**

---

## 3. Отступы

Отдельной шкалы отступов проект не вводит — используется стандартная шкала
Tailwind. Фактическое распределение `gap-*`:

| Класс | Вхождений | Где типично |
|-------|-----------|-------------|
| `gap-3` | 215 | Сетки карточек, строки контролов |
| `gap-2` | 202 | Иконка + текст, плотные группы |
| `gap-1` | 113 | Очень плотные группы (крошки, чипы) |
| `gap-4` | 93 | Блоки внутри секции |
| `gap-6` | 40 | Секции внутри страницы |
| `gap-5` | 9 | редко |

**Правило, выводимое из кода:** вертикальный ритм страницы — `gap-6` между
крупными секциями и `gap-4` внутри секции. **[ПОДТВЕРЖДЕНО — доминирующий паттерн]**

Отступы контента основной панели заданы централизованно в `App.vue:35`:
`px-4 pt-4 pb-0 sm:px-4 sm:pt-4 sm:pb-0`. Страницы добавляют `pb-8` в своём
скролл-контейнере.

Контейнеры Nuxt UI обнулены глобально (`vite.config.js:13-15`):
`container.base: 'p-0 sm:p-0 md:p-0 lg:p-0 xl:p-0 mx-0'` — отступы задаются
только каркасом страницы.

---

## 4. Радиусы

### Пользовательская настройка

`useUiTheme.RADIUS_OPTIONS` → `--ui-radius`:

| id | Подпись | Значение |
|----|---------|----------|
| `none` | Нет | `0` |
| `sm` | Маленький | `0.125rem` |
| `md` | Средний | `0.25rem` ← по умолчанию |
| `lg` | Большой | `0.375rem` |
| `xl` | Очень большой | `0.5rem` |

Компоненты Nuxt UI наследуют `--ui-radius` автоматически.

### `rounded-panel` — радиус панелей проекта

`tailwind.css`:

```css
.rounded-panel { border-radius: calc(var(--ui-radius, 0.25rem) * 4); }
```

Комментарий в файле: «Панели/карточки: масштаб как у прежнего `rounded-2xl` при
`radius=md`». То есть панель пропорционально следует за настройкой радиуса.
Применяется к сайдбару (`AppAside.vue:29`) и шапке (`AppHeader.vue:268`).
56 вхождений. **[ПОДТВЕРЖДЕНО]**

### Фиксированные радиусы в разметке

`rounded-lg` 80, `rounded-xl` 60, `rounded-full` 35, `rounded-md` 10,
`rounded-3xl` 8, `rounded-2xl` 6, `rounded-sm` 2.

> Фиксированные радиусы **не** следуют за настройкой пользователя. Для панелей и
> карточек предпочтителен `rounded-panel`.

---

## 5. Тени

Проект почти не использует тени — глубина передаётся фоном (`bg-elevated`) и
рамками (`ring-default`). **[ПОДТВЕРЖДЕНО]**

Единственный собственный токен — `--shadow-brand` (`tailwind.css:13-17`):
пять слоёв свечения из `--ui-primary` через `color-mix(in oklab, …)`, от 40px до
2px. Используется в одном файле.

---

## 6. Иконки

- **Набор: Lucide.** ~850 вхождений `i-lucide-*`. Других наборов нет.
  **[ПОДТВЕРЖДЕНО]**
- Компонент — `UIcon` (`@nuxt/ui`), внутри `@iconify/vue`.
- Префикс `i-` снимается автоматически (`Icon.vue:32`).
- Типовые размеры: `size-5` (шапка, крошки), `size-4` (внутри текста),
  `size-12` / `size-20` (иллюстративные блоки).
- Декоративные иконки помечаются `aria-hidden="true"` (29 вхождений).

Набор иконок для админ-редактора сервисов ограничен списком
`PORTAL_SERVICE_ICON_OPTIONS` (16 значений) —
`src/pages/Services/portalServiceCatalog.ts`.

> Иконки резолвятся `@iconify/vue` и при отсутствии локальных коллекций
> запрашиваются с `https://api.iconify.design`. Пакета `@iconify-json/lucide` и
> вызова `addCollection` в проекте нет. См. [известные проблемы](../known-issues.md),
> IMP-10. **[ЧАСТИЧНО ПОДТВЕРЖДЕНО — по коду, в прод-сети не проверялось]**

### Favicon

Перекрашивается под выбранный primary: SVG собирается в data-URI и
подставляется в `<link rel="icon">` (`useUiTheme.ts:144-158`). Исходная иконка —
`public/favicon.svg`, тот же контур используется в сайдбаре
(`AppAside.vue:44-57`). **[ПОДТВЕРЖДЕНО]**

---

## 7. Темы

### Основной портал

`src/composables/useColorMode.ts`. Три режима: `light`, `dark`, `system`.
**По умолчанию — `light`** (`readMainColorModePreference:20-25`), не `system`.
Хранится в `localStorage['ui-color-mode']`.

Применение: класс `.dark` на `<html>` + повторный `applyUiTheme` (ради
адаптивного primary «white»).

Синхронизация: слушатель `matchMedia('(prefers-color-scheme: dark)')` для режима
`system` и собственное событие `ui-color-mode-change` для связи между
компонентами (`useColorMode.ts:60-67,85-98`).

#### Анимация переключения

`src/App.vue:114-167` (`startThemeTransition`): круговое раскрытие из точки клика через
`document.startViewTransition` + `element.animate` по `clip-path`,
длительность 600 мс, easing `cubic-bezier(.76,.32,.29,.99)`. При отсутствии
поддержки API тема переключается мгновенно. **[ПОДТВЕРЖДЕНО]**

### Киоск

`src/composables/useColorModeSchedule.ts`. Режимы: `auto` (по умолчанию),
`light`, `dark`; ключ `localStorage['ui-kiosk-color-mode']`.

Расписание `auto`: **00:00–14:59 светлая, 15:00–23:59 тёмная**
(`getScheduledColorMode:8-10`). Переключение на границе — по таймеру.

---

## 8. Масштаб интерфейса

`src/composables/useUiZoom.ts`. Пять уровней: **80 %, 90 %, 100 %, 110 %, 125 %**.
Применяется как `document.documentElement.style.zoom`, ключ
`localStorage['ui-zoom:v1']`, атрибут `data-ui-zoom`.

При вёрстке не полагайтесь на абсолютные пиксельные размеры вьюпорта: `zoom`
меняет эффективную ширину. **[ПОДТВЕРЖДЕНО]**

---

## 9. Брейкпоинты

Стандартные Tailwind плюс один собственный (`tailwind.css:5`):

| Имя | Ширина | Вхождений в `src/` |
|-----|--------|--------------------|
| `sm` | 640px | 159 |
| `md` | 768px | 42 |
| `lg` | 1024px | 53 |
| `xl` | 1280px | 33 |
| `2xl` | 1536px | 0 |
| `3xl` | **1700px** (`--breakpoint-3xl`) | 0 |

**Фактические точки адаптива — `sm`, `md`, `lg`, `xl`.** `2xl` и `3xl` не
используются нигде, хотя `3xl` объявлен специально. **[ПОДТВЕРЖДЕНО]**

Ключевые переключения:
- `HomePage.vue:388-393` — правая колонка появляется на `xl`:
  `grid-cols-1 xl:grid-cols-[minmax(0,1fr)_420px]`;
- `HomePage.vue:412` — сетка плиток сервисов: `grid-cols-2 sm:grid-cols-3 lg:grid-cols-5`;
- `AppHeader.vue:293,304` — компактная кнопка поиска до `lg`, полная от `lg`;
- `AppHeader.vue:349` — имя пользователя скрыто до `sm`.

Единственное чтение брейкпоинтов из JS — `CalendarPage.vue:38-39`
(`useMediaQuery('(min-width: 1024px)')`, `768px`).

### Предельная ширина контента

`max-w-[1600px]` с `mx-auto` — ~14 вхождений; задан и глобально в
`vite.config.js:17` (`main.base`). Это фактическая максимальная ширина рабочей
области. **[ПОДТВЕРЖДЕНО]**

---

## 10. Скроллбары

`tailwind.css` задаёт единый стиль для всего проекта, включая телепортируемые
слои (Slideover, Modal):

- ширина/высота 8px, трек прозрачный, бегунок `border-radius: 9999px`;
- бегунок **невидим в покое** (альфа 0) и появляется на hover/active:
  `rgb(161 161 170)` в светлой, `rgb(82 82 91)` в тёмной;
- Firefox — `scrollbar-width: thin` + `scrollbar-color`.

Утилита `.scrollbar-hide` полностью прячет скроллбар (18 файлов) — применяется к
основным скролл-контейнерам страниц. **[ПОДТВЕРЖДЕНО]**

---

## 11. Переопределения Nuxt UI в `vite.config.js`

Глобальная конфигурация — `vite.config.js:7-53`. **[ПОДТВЕРЖДЕНО]**

| Что | Значение | Комментарий в коде |
|-----|----------|--------------------|
| `colors.primary` | `emerald` | — |
| `colors.neutral` | `zinc` | — |
| `container.base` | `p-0 … mx-0` | отступы обнулены |
| `main.base` | `min-h-[calc(100vh-var(--ui-header-height))] w-full max-w-[1600px] mx-auto` | — |
| `pageHeader.slots.root` | `relative border-b border-default py-4` | — |
| `inputDate.slots.segment` + варианты размеров | расширенная ширина сегментов | «Год в сегментах date picker: дефолтный `w-11` слишком узкий для `text-base` / кастомного шрифта» |

> `--ui-header-height` по умолчанию `4rem` (64px), а фактическая высота шапки и
> заголовка сайдбара — `h-[60px]` (`AppHeader.vue:268`, `AppAside.vue:30`).
> Переменная в проекте не переопределяется. **[ПОДТВЕРЖДЕНО]**
