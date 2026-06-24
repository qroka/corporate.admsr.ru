# Корпоративный портал ADMSR

Vue 3 + Vite + Nuxt UI + Vue Router. API — PHP (каталог `api/`), БД — PostgreSQL.

## Разработка

```bash
npm install
npm run dev
```

Фронтенд: `http://localhost:5173`. В dev запросы `/api` и `/img` проксируются на тестовый сервер (см. `vite.config.js`).

## Сборка

```bash
npm ci
npm run build
```

Результат — каталог `dist/`. Отдельной сборки API нет: PHP выполняется через PHP-FPM.

> В отличие от [grafic.admsr.ru](https://github.com/qroka/grafic.admsr.ru), здесь **нет** Node.js/Fastify и скрипта `build:server`.

---

## Развёртывание (Ubuntu 20.04 / 22.04)

### 1. Требования

| Компонент | Версия |
|-----------|--------|
| Node.js | 20 LTS |
| npm | 10+ (или pnpm 9+, если перейдёте на `pnpm-lock.yaml`) |
| nginx | 1.18+ |
| PHP-FPM | 8.2+ (на сервере: 8.3) |
| PostgreSQL | 14+ |
| pg_dump | для бэкапов в `deploy.sh` |

```bash
# Node 20 (NodeSource)
curl -fsSL https://deb.nodesource.com/setup_20.x | sudo -E bash -
sudo apt-get install -y nodejs build-essential

# PHP, nginx, PostgreSQL
sudo apt-get install -y nginx php8.3-fpm php8.3-pgsql php8.3-mbstring php8.3-xml postgresql-client
```

### 2. Клонирование

```bash
sudo mkdir -p /var/www/corporate.admsr.ru
sudo chown -R "$USER:www-data" /var/www/corporate.admsr.ru
git clone https://github.com/qroka/corporate.admsr.ru.git /var/www/corporate.admsr.ru
cd /var/www/corporate.admsr.ru
git checkout main
```

### 3. Конфигурация API

```bash
cp api/config.local.php.example api/config.local.php
# отредактируйте учётные данные PostgreSQL
chmod 640 api/config.local.php
```

Файл `api/config.local.php` используется `api/health.php`. Остальные endpoint'ы пока содержат константы в своих `.php` — при переносе на прод сверьте пароли во всех файлах `api/` или вынесите в общий `config.local.php`.

### 4. Каталог данных

```bash
sudo mkdir -p /var/lib/corporate-app/uploads/img /var/lib/corporate-app/backups
sudo chown -R www-data:www-data /var/lib/corporate-app
cd /var/www/corporate.admsr.ru
ln -sfn /var/lib/corporate-app/uploads/img public/img
```

### 5. Переменные деплоя

```bash
cp deploy/deploy.env.example deploy/deploy.env
# задайте PGPASSWORD для бэкапов
chmod 600 deploy/deploy.env
```

### 6. Первая сборка

```bash
npm ci
npm run build
```

Проверка health (после настройки nginx и PHP):

```bash
curl -fsS -H "Host: corporate.admsr.ru" http://127.0.0.1/api/health.php
```

### 7. PHP-FPM

API не запускается отдельным Node-процессом — используется **php8.3-fpm**:

```bash
sudo systemctl enable php8.3-fpm
sudo systemctl start php8.3-fpm
sudo systemctl status php8.3-fpm
```

Сокет по умолчанию: `unix:/run/php/php8.3-fpm.sock` (см. nginx-конфиг).

### 8. nginx

```bash
sudo cp deploy/nginx-corporate.admsr.ru.conf /etc/nginx/sites-available/corporate.admsr.ru
sudo ln -sf /etc/nginx/sites-available/corporate.admsr.ru /etc/nginx/sites-enabled/
sudo nginx -t
sudo systemctl reload nginx
```

Сертификаты: `/etc/nginx/ssl/admsr.ru/admsr.crt` и `admsr.key`.

| Параметр | Значение |
|----------|----------|
| Домен | `corporate.admsr.ru` |
| Корень SPA | `/var/www/corporate.admsr.ru/dist` |
| API | `/var/www/corporate.admsr.ru/api/*.php` |
| Загрузки | `/var/lib/corporate-app/uploads/img/` → `/img/` |

### 9. Обновление на сервере

```bash
cd /var/www/corporate.admsr.ru
chmod +x deploy/deploy.sh
./deploy/deploy.sh
```

Опции:

```bash
./deploy/deploy.sh -b main              # ветка
./deploy/deploy.sh -n                   # dry-run
./deploy/deploy.sh --skip-backup        # без pg_dump
./deploy/deploy.sh --skip-pull          # только сборка
```

Скрипт выполняет:

1. бэкап PostgreSQL (если задан `PGPASSWORD` в `deploy/deploy.env`);
2. сброс локальных изменений в `auto-imports.d.ts`, `components.d.ts`, `src/route-map.d.ts`;
3. `git pull --ff-only`;
4. `npm ci` (или `pnpm install --frozen-lockfile`);
5. `npm run build`;
6. `systemctl reload php8.3-fpm`;
7. health-check `GET /api/health.php` (до 10 попыток).

### 10. Права git на сервере

Деплой и `git pull` выполняйте **одним пользователем** (например `www-data` или свой user с группой `www-data`):

```bash
sudo chown -R deploy-user:www-data /var/www/corporate.admsr.ru
```

Не смешивайте `git pull` от `root` и сборку от обычного пользователя — иначе ошибки прав на `.git/objects`.

---

## Частые проблемы

| Симптом | Причина | Решение |
|---------|---------|---------|
| **502** на `/api/*` | PHP-FPM не запущен или неверный сокет | `systemctl status php8.3-fpm`, сверить `fastcgi_pass` в nginx |
| **404** на `/api/*.php` | Неверный `SCRIPT_FILENAME` | Использовать `deploy/nginx-corporate.admsr.ru.conf` |
| **Белая страница** | Нет `dist/index.html` | `npm run build`, nginx `root` → `.../dist` |
| **git pull** конфликтует с `package-lock.json` | Локальный `npm install` на сервере | `git checkout -- package-lock.json && git pull` |
| Health-check падает | Нет nginx / неверный Host | `curl -H "Host: corporate.admsr.ru" http://127.0.0.1/api/health.php` |
| Нет картинок `/img/` | Не создан symlink | `ln -sfn /var/lib/corporate-app/uploads/img public/img` |
| `insufficient permission` в `.git` | pull от другого пользователя | `sudo chown -R user:user .git` |

---

## Структура deploy

```
deploy/
  deploy.sh                      # скрипт обновления
  deploy.env.example             # пример переменных (скопировать в deploy.env)
  nginx-corporate.admsr.ru.conf  # виртуальный хост nginx
```

Эталон по форме: [grafic.admsr.ru](https://github.com/qroka/grafic.admsr.ru) (`deploy/deploy.sh`, nginx, README) — с учётом стека **PHP + PostgreSQL** вместо Node + SQLite.
