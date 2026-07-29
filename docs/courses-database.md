# База данных модуля курсов (V4)

Миграция: `db/migration/V4__courses_module.sql`  
Схема: `public`. Идемпотентна (`IF NOT EXISTS`). Требует `pgcrypto` и уже существующие `test_forms` / `test_attempts` (V2).

Soft refs на `user_info` / `ofo_unit` — **без жёстких FK** (как в V2): внешние таблицы синхронизируются отдельно.

---

## Обзор таблиц

| # | Таблица | Назначение |
|---|--------|------------|
| 0 | `user_sessions` | Серверные сессии (auth для курсов) |
| 1 | `course_courses` | Курс-контейнер |
| 2 | `course_versions` | Версии контента |
| 3 | `course_topics` | Темы версии |
| 4 | `course_materials` | Материалы темы |
| 5 | `course_test_links` | Связь с `test_forms` |
| 6 | `course_assignments` | Назначения (user / ofo) |
| 7 | `course_enrollments` | Записи сотрудников |
| 8 | `course_topic_progress` | Прогресс по темам |
| 9 | `course_material_progress` | Прогресс по материалам |
| 10 | `course_learning_sessions` | Heartbeat / активное время |
| 11 | `course_test_attempt_links` | Связь попыток с enrollment |
| 12 | `course_completions` | Неизменяемый снимок результата |
| 13 | `course_audit_logs` | Аудит действий |

---

## 0. `user_sessions`

| Колонка | Тип | Примечание |
|---------|-----|------------|
| `id` | bigint identity PK | |
| `user_id` | bigint NOT NULL | soft → `user_info.id` |
| `token_hash` | text UNIQUE | SHA-256 от plaintext token |
| `created_at`, `last_seen_at` | timestamptz | |
| `expires_at` | timestamptz | TTL (в коде 24 ч) |
| `revoked_at` | timestamptz | мягкий logout |
| `ip_address`, `user_agent` | text | |

**Индексы:** `user_sessions_user_idx` (активные), `user_sessions_expires_idx` (активные).

---

## 1. `course_courses`

| Колонка | Тип | Примечание |
|---------|-----|------------|
| `id` | bigint identity PK | |
| `owner_id` | bigint NOT NULL | soft → создатель |
| `title` | text NOT NULL | |
| `category` | text | |
| `current_version_id` | bigint | FK → `course_versions` (DEFERRABLE) |
| `created_at`, `updated_at` | timestamptz | |
| `deleted_at` | timestamptz | soft-delete |

**Индексы:** `course_courses_owner_idx`, `course_courses_deleted_idx`.

**FK:** `course_courses_current_version_fk` — `current_version_id` → `course_versions(id)`, `DEFERRABLE INITIALLY DEFERRED` (чтобы создать курс и версию в одной транзакции).

---

## 2. `course_versions`

| Колонка | Тип | Примечание |
|---------|-----|------------|
| `course_id` | bigint NOT NULL | FK → `course_courses` |
| `version_number` | integer NOT NULL | UNIQUE с course_id |
| `status` | text | `draft` \| `published` \| `archived` |
| `short_description`, `full_description` | text | |
| `cover_url` | text | |
| `sequential_progress` | boolean DEFAULT true | |
| `completion_rule` | text DEFAULT `all_required` | |
| `default_deadline_days` | integer | |
| `final_passing_score` | numeric(5,2) | |
| `require_final_test` | boolean DEFAULT false | |
| `created_by` | bigint | soft |
| `published_at`, `archived_at` | timestamptz | |

**Constraints:**

- `course_versions_status_chk`
- `course_versions_uniq` (`course_id`, `version_number`)

**Индекс:** `course_versions_course_status_idx`.

---

## 3. `course_topics`

FK: `course_version_id` → `course_versions` **ON DELETE CASCADE**. Soft-delete через `deleted_at`.

Поля: `title`, `description`, `sort_order`, `is_required`, `minimum_active_seconds`, `completion_rule` (по умолчанию `all_required_materials`).

**Индекс:** `course_topics_version_order_idx` (`course_version_id`, `sort_order`).

---

## 4. `course_materials`

FK: `topic_id` → `course_topics` **ON DELETE CASCADE**. Soft-delete: `deleted_at`.

| `type` (CHECK) | Смысл |
|----------------|--------|
| `rich_text` | HTML (`content_html`, санитизация в API) |
| `file`, `pdf`, `image`, `video` | файл (`file_url`, mime, size, filename) |
| `link` | `external_url` |

Также: `is_required`, `minimum_active_seconds`, `sort_order`.

**Индекс:** `course_materials_topic_order_idx`.

---

## 5. `course_test_links`

| Поле | Смысл |
|------|--------|
| `course_version_id` | FK CASCADE |
| `topic_id` | FK CASCADE, NULL для final |
| `test_form_id` | FK → `test_forms(id)` |
| `type` | `topic` \| `final` |

**Constraints:**

- `course_test_links_type_chk`
- `course_test_links_topic_chk`: topic ⇒ `topic_id IS NOT NULL`; final ⇒ `topic_id IS NULL`

**Уникальные частичные индексы:**

- один topic-тест на тему: `course_test_links_topic_uniq`
- один final на версию: `course_test_links_final_uniq`

Доп. индексы: version, topic, form.

---

## 6. `course_assignments`

| Поле | Смысл |
|------|--------|
| `course_version_id` | FK → versions |
| `target_type` | `user` \| `ofo` |
| `target_id` | soft id пользователя или ОФО |
| `starts_at`, `deadline_at` | |
| `assigned_by` | soft |
| `include_children` | для ofo |
| `cancelled_at` | отмена назначения |

**Индексы:** version, (`target_type`, `target_id`).

---

## 7. `course_enrollments`

| `status` (CHECK) |
|------------------|
| `not_started`, `in_progress`, `completed`, `failed`, `overdue`, `cancelled` |

`assignment_id` — nullable FK. Soft `user_id`.

**Уникальность:** `course_enrollments_active_uniq` на `(user_id, course_version_id)` WHERE `status NOT IN ('cancelled')`.

**Индексы:** user+status, version+status, deadline.

---

## 8–9. Progress

### `course_topic_progress`

Статусы: `locked` \| `available` \| `in_progress` \| `completed`.  
UNIQUE `(enrollment_id, topic_id)`. FK enrollment CASCADE.

### `course_material_progress`

Статусы: `not_started` \| `in_progress` \| `completed`.  
UNIQUE `(enrollment_id, material_id)`.

---

## 10. `course_learning_sessions`

Телеметрия: `enrollment_id`, `topic_id`, `material_id`, `user_id`, `started_at`, `last_heartbeat_at`, `finished_at`, `active_seconds`.

**Индекс:** `(enrollment_id, last_heartbeat_at)`.

---

## 11. `course_test_attempt_links`

Связывает enrollment + `course_test_link` + `test_attempt`.

- UNIQUE `test_attempt_id`
- FK: enrollments CASCADE, `course_test_links`, `test_attempts`

---

## 12. `course_completions`

Неизменяемый снимок: ФИО/ОФО/темы/баллы в `result_snapshot` (jsonb), `final_score`, `passed`, `completion_number`, `total_active_seconds`.

**Индекс:** `(user_id, completed_at)`.

Жёстких UNIQUE на enrollment нет — API обычно создаёт одну запись при завершении.

---

## 13. `course_audit_logs`

`user_id`, `action`, `entity_type`, `entity_id`, `payload` jsonb, IP/UA, `created_at`.

Индексы: entity, created_at.

---

## Жёсткие vs soft FK

| Есть FK | Soft (без FK) |
|---------|----------------|
| courses ↔ versions (взаимно, deferred) | `owner_id`, `created_by`, `user_id`, `assigned_by` → user_info |
| topics → versions CASCADE | `target_id` (user/ofo) |
| materials → topics CASCADE | `ofo` в snapshot через user_info |
| test_links → versions/topics/forms | |
| enrollments → versions / assignments | |
| progress / sessions → enrollments CASCADE | |
| attempt_links → attempts / links | |

---

## Применение миграции

```bash
psql -h localhost -U myuser -d corporate_portal -f db/migration/V4__courses_module.sql
```

Проверка: `npm run test:courses` (см. [courses-testing.md](courses-testing.md)).
