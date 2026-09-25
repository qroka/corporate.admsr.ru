# Документация corporate.admsr.ru

Карта документации проекта. Все утверждения здесь и в дочерних документах
привязаны к файлам репозитория — если код и документ разошлись, **источник
истины — код**, а расхождение фиксируется в [known-issues.md](known-issues.md).

**Дата последней сверки с кодом: 2026-09-22** (ветка `main`, HEAD `bcd9d95`).
О незавершённой работе в рабочей копии — см. «Состояние рабочей копии» ниже.

---

## Легенда статусов

В документах используются пометки достоверности:

| Пометка | Значение |
|---------|----------|
| **[ПОДТВЕРЖДЕНО]** | Проверено чтением кода; указан путь к файлу |
| **[ЧАСТИЧНО]** | Реализовано не полностью / работает только на части сценариев |
| **[ЗАГЛУШКА]** | Экран/функция существует, но реальной логики нет |
| **[ПРЕДЛОЖЕНИЕ]** | Идея документатора, в коде и в требованиях отсутствует |
| **[НЕИЗВЕСТНО]** | Из кода не выводится; требуется решение владельца |

Отдельно: причины архитектурных решений указываются только если они
подтверждены комментарием в коде, README, аудитом или коммитом. Если причина
неизвестна — так и написано.

---

## Что читать перед задачей

Не нужно читать всё. Выберите маршрут по типу задачи.

### Любая задача (обязательный минимум)
1. [`/CLAUDE.md`](../CLAUDE.md) — правила работы и команды
2. Этот файл — чтобы понять, где лежит нужное
3. [overview.md](overview.md) — что это за продукт и что в нём есть

### Задача по интерфейсу (страница, компонент, стиль, тема)
1. [design/README.md](design/README.md) — вход в дизайн-документацию
2. [design/tokens.md](design/tokens.md) — цвета, типографика, радиусы, темы
3. [design/layout-and-navigation.md](design/layout-and-navigation.md) — каркас страницы
4. [design/components.md](design/components.md) — что уже есть, прежде чем создавать новое
5. [design/states-and-feedback.md](design/states-and-feedback.md) — загрузка / пусто / ошибка / успех
6. [design/nuxt-ui.md](design/nuxt-ui.md) — конфигурация Nuxt UI и статус MCP

### Задача по данным / API / бизнес-логике фронтенда
1. [architecture.md](architecture.md) — границы слоёв
2. [modules.md](modules.md) — где лежит нужный модуль
3. [api.md](api.md) — контракты `/api/*.php`, разделение Go / PHP
4. [auth-and-permissions.md](auth-and-permissions.md) — если трогаете гейтинг

### Задача по backend (Go)
1. [architecture.md](architecture.md)
2. [api.md](api.md)
3. [auth-and-permissions.md](auth-and-permissions.md)
4. [data-model.md](data-model.md)
5. [`backend/README.md`](../backend/README.md) — список перенесённых эндпоинтов

### Задача по схеме БД
1. [data-model.md](data-model.md)
2. [quality-and-deploy.md](quality-and-deploy.md) — как миграции применяются на сервере

### Задача по запуску / сборке / деплою
1. [local-setup.md](local-setup.md)
2. [quality-and-deploy.md](quality-and-deploy.md)

### Перед тем как «починить странность»
1. [known-issues.md](known-issues.md) — возможно, она уже описана и намеренна
2. [decisions.md](decisions.md) — возможно, это осознанное решение
3. [PROJECT_AUDIT.md](PROJECT_AUDIT.md) — живой аудит безопасности и качества

---

## Состав документации

### Основная

| Документ | О чём |
|----------|-------|
| [overview.md](overview.md) | Назначение портала, аудитория, текущие возможности, стек и версии |
| [architecture.md](architecture.md) | Слои, границы модулей, путь запроса, двойной backend (Go/PHP) |
| [local-setup.md](local-setup.md) | Запуск, конфигурация, переменные окружения, команды |
| [modules.md](modules.md) | Модули и пользовательские сценарии с путями к файлам |
| [api.md](api.md) | Каталог эндпоинтов, формат ответа, какие в Go, какие в PHP |
| [data-model.md](data-model.md) | Таблицы, миграции, что есть в миграциях, а что — легаси |
| [auth-and-permissions.md](auth-and-permissions.md) | Сессии, роли, группы доступа, гейтинг UI |
| [quality-and-deploy.md](quality-and-deploy.md) | Проверки, тесты, сборка, nginx, systemd, деплой |
| [known-issues.md](known-issues.md) | Расхождения код↔документация, дефекты, открытые вопросы |
| [decisions.md](decisions.md) | Журнал архитектурных решений (ADR) |

### Дизайн

| Документ | О чём |
|----------|-------|
| [design/README.md](design/README.md) | Вход в дизайн-документацию, правила применения |
| [design/tokens.md](design/tokens.md) | Цвета, типографика, отступы, радиусы, тени, иконки, темы, брейкпоинты |
| [design/layout-and-navigation.md](design/layout-and-navigation.md) | Каркас приложения, сайдбар, шапка, крошки, каркас страницы, киоск |
| [design/components.md](design/components.md) | Компоненты проекта и паттерны использования Nuxt UI |
| [design/states-and-feedback.md](design/states-and-feedback.md) | Состояния, валидация, уведомления, доступность |
| [design/nuxt-ui.md](design/nuxt-ui.md) | Версия, конфигурация, способы расширения, инвентарь компонентов, статус MCP |
| [design/inconsistencies.md](design/inconsistencies.md) | Обнаруженные несоответствия и предложения по унификации |

### Ранее существовавшая документация (сохранена, не переписана)

| Документ | О чём | Примечание |
|----------|-------|------------|
| [`/README.md`](../README.md) | Основной README проекта: онбординг, деплой, возможности | Содержит устаревшие места — см. [known-issues.md](known-issues.md) |
| [`/backend/README.md`](../backend/README.md) | Go API: запуск, перенесённые эндпоинты | Актуален |
| [PROJECT_AUDIT.md](PROJECT_AUDIT.md) | Живой аудит (безопасность, качество, реестр проблем) от 2026-09-21 | Основной источник по SEC-*/BUG-* |
| [courses-architecture.md](courses-architecture.md) | LMS: иерархия, версии, enrollment | |
| [courses-database.md](courses-database.md) | LMS: таблицы V4, индексы, FK | |
| [courses-api.md](courses-api.md) | LMS: эндпоинты | Описывает PHP-реализацию; сейчас большинство обслуживает Go |
| [courses-test-integration.md](courses-test-integration.md) | LMS: связь с модулем тестов, lifecycle попытки | |
| [courses-permissions.md](courses-permissions.md) | LMS: сессии и ограничения легаси | |
| [courses-admin-guide.md](courses-admin-guide.md) | LMS: UX администратора | |
| [courses-employee-guide.md](courses-employee-guide.md) | LMS: UX сотрудника | |
| [courses-deployment.md](courses-deployment.md) | LMS: миграция, nginx, php-fpm, uploads | |
| [courses-testing.md](courses-testing.md) | LMS: smoke, E2E, security checklist | |
| [tests-users-ofo.md](tests-users-ofo.md) | Модуль тестов: пользователи и ОФО | |

### Инструкции для инструментов

| Файл | Назначение |
|------|------------|
| [`/CLAUDE.md`](../CLAUDE.md) | Правила работы над проектом (обязательны) |
| `/.cursor/rules/*.mdc` | Личности агентов Cursor (generic, не про этот проект); не удалялись |

---

## Состояние рабочей копии на момент сверки

Коммит `bcd9d95` «fix(auth): stop repeated "authorization required" errors in
courses (UX-001)» закрыл часть UX-001: 401 после неудачной попытки восстановить
токен теперь приводит к единому выходу на экран входа с сообщением «Сессия
истекла. Войдите снова.» — см. [auth-and-permissions.md](auth-and-permissions.md).

Параллельно в рабочей копии шла **незавершённая работа по модулю обучения**
(`src/pages/Courses/**`, `src/composables/useCoursesStore.ts`,
`src/pages/Admin/AdminDashboardPage.vue`; новые файлы `MyCourseCard.vue`,
`courseDeadline.ts`, `courseDuration.ts`). Эти изменения **не отражены** в
[modules.md](modules.md) и [design/components.md](design/components.md).

**Перед задачей по курсам сверьтесь с текущим состоянием файлов, а не только с
этой документацией.**
