# Развёртывание модуля учебных курсов

Базовое развёртывание портала — в корневом [README.md](../README.md). Ниже — дополнения именно для LMS.

---

## 1. Миграция V4

На сервере с доступом к PostgreSQL `corporate_portal`:

```bash
cd /var/www/corporate.admsr.ru
psql -h localhost -U myuser -d corporate_portal -f db/migration/V4__courses_module.sql
```

Миграция идемпотентна. Создаёт `user_sessions`, все `course_*` таблицы и индексы.  
Требуются уже применённые V2 (как минимум `test_forms`, `test_attempts`).

Проверка:

```bash
npm run test:courses
# или: php scripts/test_courses.php
```

---

## 2. Каталог uploads для курсов

Предпочтительный путь (как в `cs_uploads_root()`):

```bash
sudo mkdir -p /var/lib/corporate-app/uploads/courses
sudo chown -R www-data:www-data /var/lib/corporate-app
sudo chmod -R u+rwX,g+rwX /var/lib/corporate-app/uploads
```

Структура файлов: `/var/lib/corporate-app/uploads/courses/{courseId}/…`

Если каталог недоступен на запись PHP-FPM, API использует fallback:

```text
<repo>/uploads/courses/
```

Рекомендуется на проде держать данные **вне** git в `/var/lib/corporate-app/`.

Существующий symlink для картинок (`public/img` → uploads/img) **не** обслуживает файлы курсов. Выдача — через `GET /api/course_file.php` (auth).

---

## 3. nginx

Текущий `deploy/nginx-corporate.admsr.ru.conf` уже проксирует `/api/*.php` в PHP-FPM — этого достаточно для upload/file/API курсов.

Рекомендации:

| Параметр | Зачем |
|----------|--------|
| `client_max_body_size` | Загрузка материалов; в конфиге сейчас `25m`. Если нужны файлы ближе к лимиту API (~50 MB) — поднять, например до `55m`, и согласовать с `upload_max_filesize` / `post_max_size` в PHP |
| Отдельный `location` на `/var/lib/.../courses` | **Не обязателен** и обычно **нежелателен** без ACL — файлы приватные |

SPA-маршруты `/admin/courses` и `/courses` закрывает fallback `try_files … /index.html`.

После правок:

```bash
sudo nginx -t && sudo systemctl reload nginx
```

---

## 4. PHP-FPM

```bash
sudo systemctl enable php8.3-fpm
sudo systemctl restart php8.3-fpm
```

Убедитесь, что пользователь пула (`www-data`) может писать в uploads/courses.

Полезные php.ini (или pool):

```ini
upload_max_filesize = 50M
post_max_size = 55M
```

Константы БД: как у остальных API (`DB_*` в `tests_common` / `auth_context` или вынесенный `config.local.php` — сверьте с продом).

---

## 5. Frontend build

```bash
cd /var/www/corporate.admsr.ru
git pull --ff-only   # или ./deploy/deploy.sh
npm ci
npm run build        # → dist/
sudo systemctl reload php8.3-fpm
```

`deploy/deploy.sh` уже делает `npm run build` и health-check. После выката LMS дополнительно прогоните `npm run test:courses` и ручной чеклист из [courses-testing.md](courses-testing.md).

---

## 6. Auth на проде

1. Пользователи должны логиниться через актуальный `auth.php`, чтобы получить `sessionToken`.
2. Старые вкладки без токена увидят 401 на API курсов — нужен повторный вход.
3. Cookie `corp_session` работает на same-origin; SPA дублирует Bearer из localStorage.

---

## 7. Чеклист выката LMS

- [ ] V4 применена
- [ ] `user_sessions` и `course_courses` существуют (`test_courses.php`)
- [ ] `/var/lib/corporate-app/uploads/courses` writable для php-fpm
- [ ] `client_max_body_size` / PHP upload limits согласованы
- [ ] `npm run build` выполнен, nginx отдаёт новый `dist`
- [ ] Вход админом → `/admin/courses` открывается
- [ ] Создание черновика + upload материала работает
- [ ] Сотрудник видит назначенный курс на `/courses`
