# API модуля учебных курсов

Все эндпоинты — JSON over HTTP в `api/`, кроме `course_file.php` (бинарный ответ).  
Метод почти везде **POST**. Тело — JSON (`cs_body()`), если не указано иное.

**Envelope:**

```json
{ "success": true, "message": "OK", "data": { ... } }
```

**Auth:** заголовок `Authorization: Bearer <sessionToken>` и/или `X-Session-Token` (или cookie `corp_session`).  
Токен выдаёт `POST /api/auth.php` → `user.sessionToken`.

| Метка | Функция | Условие |
|-------|---------|---------|
| **admin** | `auth_require_admin` | сессия + `user_group = 'admin'` |
| **user** | `auth_require_user` | валидная сессия |
| **user†** | user + доп. проверка | владелец enrollment / админ / enrollment на версию |

Общая логика: `courses_common.php`, `auth_context.php`.

Многие admin-методы принимают `courseId` **или** `versionId` (`cs_resolve_version_id` → current version).

---

## courses_* (админ / список для себя)

| Endpoint | Auth | Body (основные поля) | `data` (кратко) |
|----------|------|----------------------|-----------------|
| `courses_list.php` | admin | — | `{ items: Course[] }` |
| `courses_create.php` | admin | **title**; опц. category, short/fullDescription, coverUrl, sequentialProgress, completionRule, defaultDeadlineDays, finalPassingScore, requireFinalTest | `{ course, version }` |
| `courses_get.php` | admin | **courseId**; опц. versionId | `{ course, version }` (дерево с контентом) |
| `courses_update.php` | admin | **courseId**; опц. versionId, title, category; контент только draft | `{ course, version }` |
| `courses_delete.php` | admin | **courseId** | `{ courseId }` (soft-delete) |
| `courses_duplicate.php` | admin | **courseId** | `{ course, version }` (новая копия) |
| `courses_publish.php` | admin | courseId \| versionId | `{ version }` |
| `courses_archive.php` | admin | courseId \| versionId (должен быть published) | `{ version }` |
| `courses_readiness.php` | admin | courseId \| versionId | `{ ready, errors[], warnings[] }` |
| `courses_for_me.php` | user | — | `{ active, completed, overdue, failed }` — элементы `{ enrollment, course }` |

---

## course_* — темы и материалы (admin)

| Endpoint | Auth | Body | `data` |
|----------|------|------|--------|
| `course_topics_create.php` | admin | courseId\|versionId, **title**; опц. description, isRequired, minimumActiveSeconds | `{ topic }` |
| `course_topics_update.php` | admin | **topicId**; поля темы (draft) | `{ topic }` |
| `course_topics_delete.php` | admin | **topicId** | `{ topicId }` |
| `course_topics_order.php` | admin | **versionId**, **topicIds[]** | `{ versionId, topicIds }` |
| `course_materials_create.php` | admin | **topicId**, **title**; type, description, contentHtml, fileUrl, externalUrl, … | `{ material }` |
| `course_materials_update.php` | admin | **materialId**; поля материала | `{ material }` |
| `course_materials_delete.php` | admin | **materialId** | `{ materialId }` |
| `course_materials_order.php` | admin | **topicId**, **materialIds[]** | `{ topicId, materialIds }` |
| `course_materials_upload.php` | admin | **multipart**: file + topicId\|materialId; опц. title, type… (до ~50 MB) | `{ materialId, fileUrl, storageKey, mimeType, fileSize, originalFilename, type }` |

Типы материала: `rich_text` \| `file` \| `pdf` \| `image` \| `video` \| `link`.

---

## course_* — тесты (admin / чтение)

| Endpoint | Auth | Body | `data` |
|----------|------|------|--------|
| `course_tests_create.php` | admin | topic-тест: **topicId** + опц. form/title/questions/isRequired; final: `type: "final"` + courseId\|versionId | `{ link }` |
| `course_tests_update.php` | admin | courseTestLinkId\|testFormId, **form**; опц. isRequired | `{ link, form }` |
| `course_tests_delete.php` | admin | **courseTestLinkId** | `{ courseTestLinkId, testFormId }` |
| `course_tests_get.php` | user† | courseTestLinkId \| testFormId \| topicId \| {versionId\|courseId, type:"final"} | `{ link, form }` (correct answers скрыты не-админу) |

`link`: `{ id, courseVersionId, topicId, testFormId, type, isRequired, sortOrder, form, questionCount, … }`.

---

## course_* — назначение и результаты (admin)

| Endpoint | Auth | Body | `data` |
|----------|------|------|--------|
| `course_assign_preview.php` | admin | опц. userIds[], ofoIds[], includeChildren | `{ count, recipients[], fromUsers, fromOfo }` |
| `course_assign.php` | admin | courseId\|versionId; **userIds** и/или **ofoIds**; опц. includeChildren, startsAt, deadlineAt, deadlineDays, comment (только published) | `{ assignmentIds[], enrollmentsCreated, skipped }` |
| `course_assignment_cancel.php` | admin | **assignmentId** | `{ assignmentId, cancelledEnrollments }` |
| `course_assignments_list.php` | admin | опц. courseId, versionId, activeOnly | `{ items[] }` |
| `course_admin_results.php` | admin | опц. courseId, versionId, status, ofoId, q, limit (≤200), offset | `{ aggregates, items[], limit, offset }` |
| `course_admin_participant.php` | admin | **enrollmentId** | `{ enrollment, version, user, topics[], materials[], attempts[], completion }` |

---

## course_* — прохождение (сотрудник)

| Endpoint | Auth | Body | `data` |
|----------|------|------|--------|
| `course_enrollment_get.php` | user† | **enrollmentId** | `{ enrollment, version (+progress), nextAction }` |
| `course_start.php` | user† | **enrollmentId** | `{ enrollment, nextAction }` |
| `course_next_action.php` | user† | **enrollmentId** | `{ nextAction, progress }` |
| `course_topic_get.php` | user† | **enrollmentId**, **topicId** | `{ topic, nextAction }` |
| `course_material_open.php` | user† | **enrollmentId**, **materialId** | `{ material, sessionId }` |
| `course_material_heartbeat.php` | user† | enrollmentId, materialId; опц. sessionId, clientGapSec | `{ addedSeconds, sessionId, activeSeconds, … }` или ignored |
| `course_material_complete.php` | user† | enrollmentId, materialId | `{ materialId, nextAction, progress }` |
| `course_result.php` | user† | enrollmentId \| completionId | `{ completion }` |
| `course_history.php` | user | — | `{ items[] }` завершений |

`nextAction.type`: `material` \| `topic_test` \| `final_test` \| `locked` \| `complete_topic` \| `complete_course` \| `done` \| `failed` \| `overdue` \| …

---

## Файлы

| Endpoint | Method | Auth | Параметры | Ответ |
|----------|--------|------|-----------|-------|
| `course_file.php` | **GET** | user† | query: `materialId` **или** `path` (`courses/{courseId}/…`) | binary stream |

---

## tests_attempt_* (попытки тестов курса и обычных)

Требуют сессию (**user**). Для тестов курса обязателен `enrollmentId`.

| Endpoint | Body | `data` |
|----------|------|--------|
| `tests_attempt_start.php` | formId \| courseTestLinkId; для курса — **enrollmentId**; опц. preview (admin) | `{ attemptId, form, courseTestLinkId?, enrollmentId? }` |
| `tests_attempt_get.php` | attemptId \| formId | `{ attemptId, status, formId, answers, form, course? }` |
| `tests_attempt_save.php` | **attemptId**, **answers** | `{ attemptId, saved }` |
| `tests_attempt_finish.php` | **attemptId**; опц. answers, durationSec | `{ attemptId, score, passed, correctCount, scorable, enrollmentId?, nextAction? }` |

Оценка: `tf_evaluate_answers` в `tests_common.php`. После finish для курса обновляется прогресс темы / возможен `cs_try_complete_enrollment`.

---

## Примеры вызова

```bash
# Список курсов (админ)
curl -sS -X POST https://corporate.admsr.ru/api/courses_list.php \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{}'

# Мои курсы
curl -sS -X POST https://corporate.admsr.ru/api/courses_for_me.php \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{}'
```

См. также: [интеграция с тестами](courses-test-integration.md), [права](courses-permissions.md).
