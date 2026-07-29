# Интеграция тестов курса с активным модулем test_forms

Курсы **не** используют легаси `forms` / `api/forms*.php`. Вопросы, варианты, попытки и оценка — из активного модуля (`test_*`, миграции V2/V3).

---

## 1. Модель связи

```
course_versions
  └── course_test_links
        ├── type=topic  + topic_id  →  test_forms
        └── type=final  (topic_id NULL) → test_forms

course_enrollments
  └── course_test_attempt_links
        └── test_attempts → test_answers …
```

Ограничения БД: не больше одного topic-теста на тему и одного final на версию.

При создании/обновлении API принудительно ставит:

- `kind = 'test'`
- `visibility = 'private'`
- `access_by_link = false` (через `tf_persistForm` / явный insert при clone)

---

## 2. Скрытие из `tests_list`

В `api/tests_list.php` ко всем выборкам добавлено:

```sql
AND NOT EXISTS (
  SELECT 1 FROM public.course_test_links ctl WHERE ctl.test_form_id = f.id
)
```

Итог: тесты курса не появляются в «Черновиках», «Все формы», «Мои формы», «Для меня» портала `/tests`. Редактирование — только через UI курсов (`/admin/courses/.../test`).

При публикации курса формы получают `list_no` (как обычные тесты), но из списков всё равно исключены.

---

## 3. Нет публичного `/t/:token` по умолчанию

| Механизм | Поведение для курса |
|----------|---------------------|
| `access_by_link` | `false` при create / clone / update |
| `tests_by_token.php` / публичный submit по токену | требуют `access_by_link = true` + валидный token → **не сработают** |
| Маршрут SPA `/t/:token` | для course-тестов не предназначен |
| Прохождение | только `tests_attempt_*` при валидной сессии и `enrollmentId` |

Попытки курса создаются с `via_link = false`.

---

## 4. Жизненный цикл попытки

```
1. tests_attempt_start
     — formId или courseTestLinkId
     — enrollmentId (обязателен, если форма в course_test_links)
     — проверка: enrollment принадлежит пользователю, version совпадает, status ∈ not_started|in_progress|overdue
     — reuse in_progress attempt или INSERT test_attempts
     — INSERT course_test_attempt_links
     — при not_started → enrollment in_progress

2. tests_attempt_save
     — черновые answers

3. tests_attempt_finish
     — tf_evaluate_answers → score, passed, correctCount
     — attempt status=completed
     — если topic-тест сдан и тема готова → topic_progress completed, cs_recalculate_locks
     — cs_try_complete_enrollment (все обязательные темы + final)
     — nextAction в ответе
```

Лимит попыток — настройки `test_forms` (`limit_attempts` / `attempts`).

Админский `preview=true` в start может вернуть форму с правильными ответами (только admin).

---

## 5. `tf_evaluate_answers`

Общая функция модуля тестов (`tests_common.php`):

- сравнивает ответы с `test_options.is_correct` / `correct_value`;
- считает процент и `passed` при `use_passing_score`;
- для course finish результат пишется в `test_attempts` и влияет на `cs_test_link_passed`.

Критерий «тест ссылки пройден» (`cs_test_link_passed`):

- есть completed attempt;
- если у формы `use_passing_score` — нужен `passed=true`;
- иначе достаточно completed.

---

## 6. Readiness и публикация

`cs_readiness` / `courses_readiness.php` проверяет:

- наличие тем и обязательных материалов;
- у обязательных тестов — вопросы с корректными ответами;
- при `require_final_test` — наличие final и валидный порог.

`cs_publish_version` публикует **все** `test_form_id` из links версии вместе с версией курса.

---

## 7. UI сотрудника

Маршрут: `/courses/:enrollmentId/tests/:courseTestLinkId` → `CourseTestPage.vue`  
→ `tests_attempt_start` / save / finish с session token.

Админ: `TopicTestPage.vue`, `FinalTestPage.vue` — CRUD через `course_tests_*`.

---

## 8. Чего не делать

- Не открывать course-тесты через `/tests` или `/t/:token`.
- Не полагаться на `userId` в body для attempt API — личность только из сессии.
- Не включать `access_by_link` у course-форм без отдельного продуктового решения (сейчас API это подавляет).
