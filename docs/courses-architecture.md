# Архитектура модуля учебных курсов (LMS)

Практический обзор: сущности, версионирование, enrollment и связь с активным модулем тестов (`test_forms`).

Связанные документы: [БД](courses-database.md) · [API](courses-api.md) · [Интеграция с тестами](courses-test-integration.md) · [Права](courses-permissions.md)

---

## 1. Иерархия сущностей

```
course_courses                    ← «курс» как контейнер
  └── course_versions             ← draft | published | archived
        ├── course_topics
        │     ├── course_materials   (rich_text | file | pdf | image | video | link)
        │     └── course_test_links  (type=topic → test_forms)
        └── course_test_links        (type=final → test_forms)

Назначение:
  course_assignments  →  course_enrollments (на конкретную version)
       └── progress: topic_progress, material_progress, learning_sessions
       └── course_test_attempt_links → test_attempts
       └── course_completions (неизменяемый снимок)
```

| Уровень | Таблица | Смысл |
|---------|--------|--------|
| Курс | `course_courses` | Владелец, название, категория, указатель на текущую версию |
| Версия | `course_versions` | Контент и правила прохождения; редактируется только `draft` |
| Тема | `course_topics` | Блок обучения, порядок, обязательность, мин. активное время |
| Материал | `course_materials` | Текст / файл / ссылка внутри темы |
| Тест | `course_test_links` + `test_forms` | Промежуточный (тема) или итоговый; данные теста живут в модуле тестов |

Общая логика API — `api/courses_common.php` (сборка дерева версии, readiness, publish, progress, completion).

---

## 2. Версионирование

1. При создании курса сразу создаётся версия №1 со статусом `draft`.
2. `course_courses.current_version_id` указывает на актуальную версию (после публикации — на опубликованную).
3. Контент (темы, материалы, тесты, описания) меняется у версии в статусе `draft` **и `published`** — `VersionEditable` в `backend/internal/courses/common.go`, `cs_version_editable` в `api/courses_common.php` («позволяем редактировать не только черновик, но и опубликованную версию»). Новая версия при правке **не создаётся**: изменения опубликованного курса сразу видят сотрудники, которые его проходят. Неизменяема только `archived`. (До 2026-09-25 здесь было написано обратное — см. DOC-07, ADR-029.)
4. Публикация (`courses_publish.php` → `cs_publish_version`):
   - проверка readiness (`cs_readiness`);
   - связанные `test_forms` переводятся в `published` (получают `list_no`);
   - версия → `published`, `published_at`;
   - `current_version_id` обновляется.
5. Архивация (`courses_archive.php`) — статус `archived`.
6. Дублирование (`courses_duplicate.php`) — новый курс + draft v1 с копией дерева и клонами `test_forms`.

Enrollment всегда привязан к **конкретной** `course_version_id`. Сотрудник, назначенный на v1, не «переезжает» автоматически на новую версию.

---

## 3. Интеграция с `test_forms`

Тесты курса **не** дублируют схему вопросов: используется активный модуль тестов (миграции V2/V3).

- Создание: `cs_create_topic_test` / `cs_create_final_test` → `tf_persistForm` с `kind=test`, `visibility=private`, без публичной ссылки (`access_by_link=false`).
- Связь: `course_test_links` (один topic-тест на тему, один final на версию).
- Прохождение: `tests_attempt_*` + `course_test_attempt_links`, не публичный `/t/:token`.
- В `tests_list.php` формы из `course_test_links` **исключены** из всех списков.

Подробнее: [courses-test-integration.md](courses-test-integration.md).

---

## 4. Поток enrollment (сотрудник)

```
Админ: publish → assign (user / ofo)
         ↓
course_assignments + course_enrollments (status=not_started)
         ↓
Сотрудник: courses_for_me → course_start → in_progress
         ↓
Темы (sequential или свободный порядок):
  material_open → heartbeat → material_complete
  → topic test (tests_attempt_*) при необходимости
  → тема completed → разблокировка следующей
         ↓
Итоговый тест (если require_final_test)
         ↓
cs_try_complete_enrollment → course_completions + status=completed
```

Ключевые сервисы:

| Функция | Назначение |
|---------|------------|
| `cs_recalculate_locks` | `locked` / `available` при `sequential_progress` |
| `cs_check_topic_complete` | Материалы + мин. время + обязательный тест темы |
| `cs_next_action` | Что делать дальше (material / topic_test / final_test / …) |
| `cs_try_complete_enrollment` | Снимок результата в `course_completions` |
| `cs_mark_overdue` | `deadline_at` истёк → `overdue` |

Статусы enrollment: `not_started` → `in_progress` → `completed` | `failed` | `overdue` | `cancelled`.

---

## 5. Назначение (assign)

- Цели: `target_type` = `user` | `ofo` | `all` (все активные учётные записи, в том числе без ОФО; V12).
- ОФО: опционально `include_children` — рекурсия по `ofo_unit.parent_id`.
- На **опубликованную** версию создаются assignment (по одному на цель) + enrollment на каждого активного пользователя (одна запись, даже если он попал в несколько целей).
- `ofo` и `all` — **постоянные**: `courses.Service.SyncStandingAssignments` (`backend/internal/courses/assign.go`) досоздаёт записи при входе, сохранении профиля и в `courses_for_me` (ADR-036). Срок: общий `deadline_at` назначения или `deadline_days` от момента выдачи.
- Один активный enrollment на пару `(user_id, course_version_id)` (уникальный индекс без `cancelled`).

---

## 6. Аутентификация в модуле

Модуль курсов опирается на **серверную сессию**:

1. `POST auth.php` → `sessionToken` (+ cookie `corp_session`).
2. SPA хранит токен в `localStorage` (`auth-session`) и шлёт `Authorization: Bearer` / `X-Session-Token`.
3. `auth_context.php` резолвит пользователя по `user_sessions.token_hash` — **не** по `userId` из body.

Админ-эндпоинты: `user_group = 'admin'`. См. [courses-permissions.md](courses-permissions.md).

---

## 7. Файлы материалов

- Загрузка: `course_materials_upload.php` → каталог `/var/lib/corporate-app/uploads/courses/{courseId}/` (fallback: `uploads/courses/` в корне проекта).
- Выдача: `GET course_file.php` — Go (`course_file.go`), сессия по cookie + доступ: редактор раздела «Обучение» или неотменённый enrollment на версию материала; `Range` поддерживается (видео).
- Прямой nginx alias для `/courses/` **не** обязателен — доступ через PHP.

---

## 8. Frontend-маршруты

| Роль | Префикс | Примеры |
|------|---------|---------|
| Админ | `/admin/courses` | список, workspace, темы, материалы, тесты, publish, assign, results |
| Сотрудник | `/courses` | мои курсы, enrollment, тема (оглавление), материал (`…/topics/:topicId/materials/:materialId`), тест, результат |

См. `src/router/index.js`, страницы в `src/pages/Courses/`.

---

## 9. Аудит

Действия пишутся в `course_audit_logs` через `cs_audit` (создание, publish, assign, completion и т.д.).
