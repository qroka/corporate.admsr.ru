# Права доступа и сессии (модуль курсов)

---

## 1. Кто считается администратором

Проверка на сервере (`auth_require_admin`):

```text
user_info.user_group === 'admin'
```

Поле `role` (должность) **не** даёт прав на admin API курсов. UI-флаг в SPA должен совпадать с `user_group`, иначе API вернёт **403**.

Сотрудник: любая валидная сессия (`auth_require_user`) + доступ к своему enrollment.

---

## 2. Session token

| Шаг | Где |
|-----|-----|
| Логин | `POST /api/auth.php` → `auth_create_session` |
| Ответ | `user.sessionToken` (plaintext один раз) |
| Хранение в БД | `user_sessions.token_hash` = SHA-256(token) |
| SPA | `localStorage['auth-session']` (`useAuthSession.ts`) |
| Cookie | `corp_session` (HttpOnly, SameSite=Lax) — опционально |
| Запросы курсов | `Authorization: Bearer …` + `X-Session-Token` |
| TTL | 24 часа (`AUTH_SESSION_TTL_HOURS`) |
| Logout | `auth_revoke_session` / `logout.php` |

Без валидного токена endpoints курсов отвечают **401**.

Профиль в `localStorage['auth-user']` — для UI; **не** источник identity на API курсов.

---

## 3. Никогда не доверять `userId` из body

Модуль курсов:

- личность = `auth_current_user()` по токену;
- enrollment чужого пользователя → **403** (кроме админа, где это явно разрешено);
- attempt finish/save — только владелец попытки.

Передача `userId` в JSON **не** подменяет сессию.

---

## 4. Матрица доступа (кратко)

| Действие | Admin | Владелец enrollment | Чужой user |
|----------|-------|---------------------|------------|
| CRUD курса, publish, assign, results | ✅ | ❌ | ❌ |
| `courses_for_me`, history | ✅ (свои) | ✅ | только свои |
| Прохождение материалов/тестов | preview/participant API | ✅ | ❌ |
| `course_file` | ✅ | если enrollment на версию | ❌ |
| Attempt start (course test) | нет* | ✅ | ❌ |

\* Админ проходит тесты курса через enrollment / preview-флаги по сценарию UI, не через подмену userId.

---

## 5. Ограничения легаси API (важно)

Активный модуль **тестов** (`tests_list`, `tests_submit`, CRUD форм и т.д.) и часть старых endpoint'ов портала **по-прежнему** могут принимать `userId` из тела и не проверять `user_group` на сервере. Это описано в [tests-users-ofo.md](tests-users-ofo.md).

| Слой | Identity | Admin check |
|------|----------|-------------|
| Курсы (`course_*`, `courses_*`, `tests_attempt_*`) | session token | `user_group=admin` |
| Классические `tests_*.php` (кроме attempt) | часто `userId` в body | в основном UI |
| Легаси `forms*.php` | отдельно | отдельно |

**Вывод для интеграторов:** новые фичи и внешние клиенты для LMS должны использовать только session auth. Не смешивать с «доверенным» userId легаси.

`check-auth.php` / heartbeat портала могут жить параллельно со старой моделью — для курсов обязателен именно `sessionToken` из `auth.php`.

---

## 6. Практические правила

1. После логина сохраняйте `sessionToken` и шлите его на все `course_*` / `courses_*` / `tests_attempt_*`.
2. Не передавайте и не принимайте `userId` как доказательство личности в LMS.
3. Для админ-скриптов используйте учётку с `user_group=admin`.
4. Ротация: при компрометации — revoke сессий (`auth_revoke_user_sessions`).
