# Nuxt UI в проекте

## Версия и режим подключения

| Параметр | Значение |
|----------|----------|
| Пакет | `@nuxt/ui` |
| Версия в `package.json` | `^4.8.0` |
| Версия в `node_modules` | **4.8.0** |
| Режим | **Vue (без Nuxt)** — Vite-плагин + Vue-плагин |
| Tailwind | 4.3.0 |
| Префикс компонентов | `U` |

Подключение. **[ПОДТВЕРЖДЕНО]**

```js
// vite.config.js
import ui from '@nuxt/ui/vite';
plugins: [vue(), ui({ ui: { … } })]
```

```js
// src/main.js
import ui from '@nuxt/ui/vue-plugin';
import './assets/tailwind.css';
app.use(ui);
```

```css
/* src/assets/tailwind.css */
@import "tailwindcss";
@import "@nuxt/ui";
```

Локаль: `<UApp :locale="ru">` c `import { ru } from '@nuxt/ui/locale'`
(`src/App.vue:3`, `src/App.vue:69`).

---

## Статус MCP Nuxt UI

**MCP-сервер Nuxt UI в текущей сессии недоступен.**

Проверено: в списке инструментов сессии нет ни одного сервера, относящегося к
Nuxt UI; поиск по реестру отложенных инструментов (`+nuxt`) не дал совпадений;
в репозитории нет `.mcp.json` или иной конфигурации MCP.

**Никакие сведения в этой документации не получены через MCP Nuxt UI.**

### Чем пользоваться вместо MCP

В порядке надёжности:

1. **Типы компонента в `node_modules`** — источник истины для props, slots и
   emits ровно той версии, что стоит в проекте:
   ```
   node_modules/@nuxt/ui/dist/runtime/components/<Component>.vue.d.ts
   node_modules/@nuxt/ui/dist/runtime/components/<Component>.d.vue.ts
   ```
2. **Исходник компонента** — `node_modules/@nuxt/ui/dist/runtime/components/<Component>.vue`:
   видно имена слотов и структуру `:ui`.
3. **Дефолты темы** — `node_modules/@nuxt/ui/dist/shared/ui.*.mjs`
   (`getDefaultConfig`, `resolveColors`), `node_modules/@nuxt/ui/dist/runtime/index.css`
   (семантические CSS-переменные).
4. **Официальная документация** — только для версии 4.x, со сверкой по типам
   выше. API между мажорами менялся.

Если MCP станет доступен — использовать его, но всё равно сверять с версией
4.8.0 из `node_modules`.

---

## Глобальная конфигурация

`vite.config.js:7-53`. **[ПОДТВЕРЖДЕНО]**

```js
ui({
  ui: {
    colors: { primary: 'emerald', neutral: 'zinc' },
    container: { base: 'p-0 sm:p-0 md:p-0 lg:p-0 xl:p-0 mx-0' },
    main:      { base: 'min-h-[calc(100vh-var(--ui-header-height))] w-full max-w-[1600px] mx-auto' },
    pageHeader:{ slots: { root: 'relative border-b border-default py-4' } },
    inputDate: { slots: { segment: [...] }, variants: { size: { xs|sm|md|lg|xl } } },
  },
})
```

Комментарий к `inputDate` в конфиге: «Год в сегментах date picker: дефолтный
`w-11` слишком узкий для `text-base` / кастомного шрифта».

### Как конфиг соотносится с темой пользователя

`colors.primary` / `colors.neutral` из `vite.config.js` задают **стартовые**
палитры. Далее `applyUiTheme()` (`src/composables/useUiTheme.ts:188-226`)
перезаписывает `--ui-color-primary-*`, `--ui-color-neutral-*` и `--ui-primary`
значениями выбранной пользователем палитры — при каждом старте приложения и при
каждой смене темы. Значения по умолчанию совпадают (emerald / zinc).
**[ПОДТВЕРЖДЕНО]**

Семантические `success`, `info`, `warning`, `error` **не переопределяются** и
остаются green / blue / yellow / red.

---

## Автоимпорт

Плагин генерирует два файла в корне (оба — в `.gitignore`, но отслеживаются git;
`deploy.sh` сбрасывает их перед `git pull`):

| Файл | Что содержит |
|------|--------------|
| `components.d.ts` | Компоненты Nuxt UI + компоненты проекта, попавшие в сборку |
| `auto-imports.d.ts` | Композаблы и утилиты (`useToast`, `defineShortcuts`, …) |

Явный импорт компонента не нужен. Композаблы, используемые в коде проекта,
импортируются явно из `@nuxt/ui/composables` (`useAppToast.ts:1`).

---

## Инвентарь компонентов

В `@nuxt/ui` 4.8.0 — **121 компонент**. Проект использует **52**
(по `components.d.ts` последней сборки; 53-я запись с префиксом `U` —
`UContentSurround`, это компонент проекта, а не библиотеки).
**[ПОДТВЕРЖДЕНО — сверено скриптом]**

### Используются (52)

`UAlert`, `UApp`, `UAvatar`, `UBadge`, `UBlogPost`, `UBreadcrumb`, `UButton`,
`UCalendar`, `UCard`, `UCarousel`, `UCheckbox`, `UCheckboxGroup`, `UContainer`,
`UDashboardGroup`, `UDashboardNavbar`, `UDashboardPanel`,
`UDashboardResizeHandle`, `UDashboardSearch`, `UDashboardSearchButton`,
`UDashboardSidebar`, `UDropdownMenu`, `UEditor`, `UEditorEmojiMenu`,
`UEditorToolbar`, `UEmpty`, `UFileUpload`, `UForm`, `UFormField`, `UIcon`,
`UInput`, `UInputDate`, `UInputTime`, `UMain`, `UModal`, `UNavigationMenu`,
`UPageCard`, `UPageHeader`, `UPopover`, `UProgress`, `URadioGroup`,
`UScrollArea`, `USelect`, `USelectMenu`, `USeparator`, `USkeleton`,
`USlideover`, `USwitch`, `UTable`, `UTabs`, `UTextarea`, `UTooltip`, `UUser`.

### Доступны, но не используются (69)

Могут пригодиться вместо самописного:

| Компонент | Возможное применение |
|-----------|----------------------|
| `UAccordion`, `UCollapsible` | Сворачиваемые блоки |
| `UPagination` | Постраничная навигация (сейчас везде бесконечные ленты) |
| `UStepper` | Пошаговые сценарии (онбординг, создание курса) |
| `UTimeline` | История прохождения, аудит |
| `UTree` | Дерево ОФО (сейчас свои компоненты) |
| `UInputNumber`, `UInputTags`, `UPinInput`, `USlider`, `UColorPicker` | Специализированные поля |
| `UChip` | Индикатор-точка на аватаре/кнопке (например, у уведомлений) |
| `UFieldGroup` | Группировка контролов в одну «кнопочную» полосу |
| `UKbd` | Отображение клавиш |
| `UDrawer` | Нижняя панель на мобильных |
| `UContextMenu` | Контекстное меню |
| `UBanner` | Полоса-объявление вверху страницы |
| `UPage*` (`UPageGrid`, `UPageList`, `UPageSection`, `UPageAside`, …) | Раскладки страниц |
| `UDashboardToolbar`, `UDashboardSidebarCollapse`, `UDashboardSidebarToggle` | Элементы dashboard-каркаса |
| `UChat*` (9 шт.) | Чат-интерфейсы |
| `UAuthForm` | Форма входа |
| `UError`, `UFooter`, `UHeader`, `UMarquee`, `UPricing*`, `UChangelog*`, `UBlogPosts` | Прочее |

Полный актуальный список:

```bash
ls node_modules/@nuxt/ui/dist/runtime/components/*.vue | xargs -n1 basename | sed 's/\.vue$//'
```

---

## Иконки

`UIcon` → `@iconify/vue` (`node_modules/@nuxt/ui/dist/runtime/vue/components/Icon.vue`).
Префикс `i-` снимается автоматически: `i-lucide-user` → `lucide:user`.

Используется только Lucide (~826 вхождений).

> Локальных коллекций (`@iconify-json/*`) и вызова `addCollection` в проекте нет,
> поэтому данные иконок запрашиваются `@iconify/vue` c
> `https://api.iconify.design`. См. [known-issues.md](../known-issues.md), IMP-10.
> **[ЧАСТИЧНО ПОДТВЕРЖДЕНО — по коду]**

---

## Расширение Nuxt UI: принятый порядок

**[ПОДТВЕРЖДЕНО — по фактическому использованию]**

| Уровень | Способ | Когда |
|---------|--------|-------|
| 1 | `vite.config.js` → `ui.<component>.slots` / `variants` | Правило действует во всём приложении |
| 2 | Проп `:ui="{ slot: 'классы' }"` | Один экземпляр (самый частый способ) |
| 3 | `class="…"` | Простые правки; объединяется через `tailwind-merge` |
| 4 | `!`-префикс (`!max-w-4xl`) | Перебить базовый класс компонента |
| 5 | Утилита в `tailwind.css` | Повторяющийся приём: `.rounded-panel`, `.scrollbar-hide` |
| 6 | Компонент-обёртка в `src/components/` | Нужна своя логика, а не только стиль |

Чего **не** делать:
- копировать разметку компонента Nuxt UI в свой SFC;
- переопределять внутренние классы через глобальные селекторы в `tailwind.css`;
- вводить новую библиотеку UI-компонентов.

---

## Чек-лист UI-задачи

1. Компонент уже есть в проекте? → [components.md](components.md).
2. Есть в Nuxt UI? → инвентарь выше.
3. Проверить props/slots/emits по `*.vue.d.ts` **установленной версии 4.8.0**,
   не по памяти и не по документации другого мажора.
4. Цвета — только семантические токены; `color` — только из
   `primary|secondary|success|info|warning|error|neutral`.
5. Кастомизация — по уровням выше, начиная с минимального.
6. Проверить в светлой и тёмной теме, на нескольких primary-палитрах и
   брейкпоинтах.
7. Не заявлять, что API проверен через MCP, если обращения не было.
