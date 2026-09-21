#!/usr/bin/env bash
# deploy.sh — обновление corporate.admsr.ru (Vue SPA + Go API + PHP-FPM fallback)
#
# Переменные: deploy/deploy.env или окружение.

set -euo pipefail

# --- Параметры по умолчанию ------------------------------------------------
APP_DIR="${APP_DIR:-/var/www/corporate.admsr.ru}"
DATA_DIR="${DATA_DIR:-/var/lib/corporate-app}"
BACKUP_DIR="${BACKUP_DIR:-${DATA_DIR}/backups}"
BACKUP_KEEP="${BACKUP_KEEP:-7}"
DOMAIN="${DOMAIN:-corporate.admsr.ru}"
SERVICE_NAME="${SERVICE_NAME:-php8.3-fpm}"
GO_SERVICE_NAME="${GO_SERVICE_NAME:-corporate-go-api}"
HEALTH_URL="${HEALTH_URL:-https://127.0.0.1/api/health.php}"
HEALTH_HOST="${HEALTH_HOST:-${DOMAIN}}"
PACKAGE_MANAGER="${PACKAGE_MANAGER:-npm}"
GIT_BRANCH="${GIT_BRANCH:-main}"
HEALTH_RETRIES="${HEALTH_RETRIES:-10}"
HEALTH_SLEEP="${HEALTH_SLEEP:-2}"
GO_VERSION="${GO_VERSION:-1.22.10}"
GO_AUTO_INSTALL="${GO_AUTO_INSTALL:-1}"
GO_SYNC_NGINX="${GO_SYNC_NGINX:-0}"
SKIP_GO="${SKIP_GO:-0}"

DRY_RUN=0
SKIP_BACKUP=0
SKIP_PULL=0

# --- Цвета -------------------------------------------------------------------
if [[ -t 1 ]] && command -v tput >/dev/null 2>&1; then
  RED=$(tput setaf 1)
  GREEN=$(tput setaf 2)
  YELLOW=$(tput setaf 3)
  BLUE=$(tput setaf 4)
  BOLD=$(tput bold)
  RESET=$(tput sgr0)
else
  RED="" GREEN="" YELLOW="" BLUE="" BOLD="" RESET=""
fi

log()  { echo -e "${BLUE}[deploy]${RESET} $*"; }
ok()   { echo -e "${GREEN}[deploy]${RESET} $*"; }
warn() { echo -e "${YELLOW}[deploy]${RESET} $*"; }
err()  { echo -e "${RED}[deploy]${RESET} $*" >&2; }

run() {
  if [[ "$DRY_RUN" -eq 1 ]]; then
    log "${BOLD}(dry-run)${RESET} $*"
  else
    log "$*"
    eval "$@"
  fi
}

usage() {
  cat <<'EOF'
Использование: ./deploy/deploy.sh [опции]

Опции:
  -b, --branch <ветка>   Ветка для git pull --ff-only (по умолчанию: main)
  -n, --dry-run          Показать команды без выполнения
      --skip-backup      Пропустить бэкап PostgreSQL
      --skip-pull        Пропустить git pull
      --skip-go          Не собирать / не перезапускать Go API
  -h, --help             Справка

Переменные (deploy/deploy.env):
  APP_DIR, DATA_DIR, DOMAIN, SERVICE_NAME, GO_SERVICE_NAME,
  HEALTH_URL, HEALTH_HOST, PACKAGE_MANAGER, GIT_BRANCH,
  GO_VERSION, GO_AUTO_INSTALL (1/0), GO_SYNC_NGINX (1/0), SKIP_GO (1/0),
  PGPASSWORD, PGHOST, PGUSER, PGDATABASE

Пример:
  ./deploy/deploy.sh
  GO_SYNC_NGINX=1 ./deploy/deploy.sh -b main
EOF
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    -b|--branch) GIT_BRANCH="$2"; shift 2 ;;
    -n|--dry-run) DRY_RUN=1; shift ;;
    --skip-backup) SKIP_BACKUP=1; shift ;;
    --skip-pull) SKIP_PULL=1; shift ;;
    --skip-go) SKIP_GO=1; shift ;;
    -h|--help) usage; exit 0 ;;
    *) err "Неизвестный аргумент: $1"; usage; exit 1 ;;
  esac
done

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ENV_FILE="${SCRIPT_DIR}/deploy.env"
if [[ -f "$ENV_FILE" ]]; then
  # shellcheck disable=SC1090
  source "$ENV_FILE"
fi

if [[ ! -d "$APP_DIR" ]]; then
  err "Каталог приложения не найден: $APP_DIR"
  exit 1
fi

cd "$APP_DIR"

if [[ ! -f "api/config.local.php" ]] && [[ -f "api/config.local.php.example" ]]; then
  warn "Нет api/config.local.php — создайте из api/config.local.php.example (health-check БД будет без проверки PG)."
fi

# --- 1. Бэкап PostgreSQL -----------------------------------------------------
backup_database() {
  if [[ "$SKIP_BACKUP" -eq 1 ]]; then
    warn "Бэкап пропущен (--skip-backup)."
    return 0
  fi

  if [[ -z "${PGPASSWORD:-}" ]]; then
    warn "PGPASSWORD не задан — бэкап PostgreSQL пропущен."
    return 0
  fi

  if ! command -v pg_dump >/dev/null 2>&1; then
    warn "pg_dump не найден — бэкап пропущен."
    return 0
  fi

  run "mkdir -p '${BACKUP_DIR}'"
  local ts file
  ts="$(date +%Y%m%d-%H%M%S)"
  file="${BACKUP_DIR}/corporate_portal-${ts}.sql.gz"

  log "Бэкап БД → ${file}"
  if [[ "$DRY_RUN" -eq 0 ]]; then
    pg_dump -h "${PGHOST:-localhost}" -p "${PGPORT:-5432}" -U "${PGUSER:-myuser}" \
      "${PGDATABASE:-corporate_portal}" | gzip -9 > "$file"
    ok "Бэкап создан: ${file}"

    mapfile -t old_backups < <(ls -1t "${BACKUP_DIR}"/corporate_portal-*.sql.gz 2>/dev/null || true)
    if ((${#old_backups[@]} > BACKUP_KEEP)); then
      for ((i = BACKUP_KEEP; i < ${#old_backups[@]}; i++)); do
        rm -f "${old_backups[$i]}"
        log "Удалён старый бэкап: ${old_backups[$i]}"
      done
    fi
  fi
}

# --- 2. Сброс автогенерируемых файлов перед pull -----------------------------
reset_generated_files() {
  local files=(
    "auto-imports.d.ts"
    "components.d.ts"
    "src/route-map.d.ts"
  )
  for f in "${files[@]}"; do
    if [[ -f "$f" ]] && git status --porcelain -- "$f" 2>/dev/null | grep -q .; then
      warn "Сброс локальных изменений: $f"
      run "git checkout -- '$f' 2>/dev/null || rm -f '$f'"
    fi
  done
}

# --- 3. git pull -------------------------------------------------------------
git_update() {
  if [[ "$SKIP_PULL" -eq 1 ]]; then
    warn "git pull пропущен (--skip-pull)."
    return 0
  fi

  run "git fetch origin '${GIT_BRANCH}'"
  run "git pull --ff-only origin '${GIT_BRANCH}'"
}

# --- 4. Зависимости ----------------------------------------------------------
install_deps() {
  case "$PACKAGE_MANAGER" in
    pnpm)
      if [[ "$DRY_RUN" -eq 0 ]]; then
        command -v pnpm >/dev/null 2>&1 || { err "pnpm не установлен"; exit 1; }
      fi
      if [[ -f pnpm-lock.yaml ]]; then
        run "pnpm install --frozen-lockfile"
      else
        warn "pnpm-lock.yaml не найден — выполняется pnpm install"
        run "pnpm install"
      fi
      ;;
    npm)
      if [[ "$DRY_RUN" -eq 0 ]]; then
        command -v npm >/dev/null 2>&1 || { err "npm не установлен"; exit 1; }
      fi
      if [[ -f package-lock.json ]]; then
        run "npm ci"
      else
        warn "package-lock.json не найден — выполняется npm install"
        run "npm install"
      fi
      ;;
    *)
      err "Неизвестный PACKAGE_MANAGER: $PACKAGE_MANAGER (допустимо: npm, pnpm)"
      exit 1
      ;;
  esac
}

# --- 5. Сборка фронтенда -----------------------------------------------------
build_frontend() {
  if [[ "$PACKAGE_MANAGER" == "pnpm" ]]; then
    run "pnpm run build"
  else
    run "npm run build"
  fi

  if [[ "$DRY_RUN" -eq 0 ]] && [[ ! -f dist/index.html ]]; then
    err "Сборка не создала dist/index.html"
    exit 1
  fi
}

# --- 6. Каталоги данных ------------------------------------------------------
ensure_data_dirs() {
  run "sudo mkdir -p '${DATA_DIR}/uploads/img' '${DATA_DIR}/uploads/courses' '${BACKUP_DIR}'"
  run "sudo chown -R www-data:www-data '${DATA_DIR}'"
  if [[ "$DRY_RUN" -eq 0 ]] && [[ ! -e public/img ]]; then
    run "ln -sfn '${DATA_DIR}/uploads/img' public/img"
  fi
}

# --- 6b. Миграции PostgreSQL -------------------------------------------------
apply_migrations() {
  if [[ -z "${PGPASSWORD:-}" ]]; then
    warn "PGPASSWORD не задан — миграции БД пропущены. Примените вручную: db/migration/V4__courses_module.sql"
    return 0
  fi
  if ! command -v psql >/dev/null 2>&1; then
    warn "psql не найден — миграции пропущены."
    return 0
  fi

  local mig_dir="${APP_DIR}/db/migration"
  if [[ ! -d "$mig_dir" ]]; then
    warn "Каталог миграций не найден: ${mig_dir}"
    return 0
  fi

  log "Применение SQL-миграций из ${mig_dir}"
  local f
  for f in "$mig_dir"/V*.sql; do
    [[ -f "$f" ]] || continue
    log "Миграция: $(basename "$f")"
    if [[ "$DRY_RUN" -eq 0 ]]; then
      PGPASSWORD="$PGPASSWORD" psql -h "${PGHOST:-localhost}" -p "${PGPORT:-5432}" \
        -U "${PGUSER:-myuser}" -d "${PGDATABASE:-corporate_portal}" \
        -v ON_ERROR_STOP=1 -f "$f"
    else
      log "(dry-run) psql -f $f"
    fi
  done
  ok "Миграции применены (идемпотентно)."
}

# --- 6c. Go toolchain --------------------------------------------------------
ensure_go() {
  if command -v go >/dev/null 2>&1; then
    return 0
  fi
  if [[ -x /usr/local/go/bin/go ]]; then
    export PATH="/usr/local/go/bin:${PATH}"
    return 0
  fi
  if [[ "${GO_AUTO_INSTALL}" != "1" ]]; then
    return 1
  fi

  local arch goarch tarball url tmp
  arch="$(uname -m)"
  case "$arch" in
    x86_64|amd64) goarch=amd64 ;;
    aarch64|arm64) goarch=arm64 ;;
    *) warn "Неизвестная архитектура ${arch} — автоустановка Go пропущена"; return 1 ;;
  esac

  tarball="go${GO_VERSION}.linux-${goarch}.tar.gz"
  url="https://go.dev/dl/${tarball}"
  tmp="$(mktemp -d)"
  log "Установка Go ${GO_VERSION} → /usr/local/go (${url})"
  if [[ "$DRY_RUN" -eq 1 ]]; then
    log "(dry-run) download+install Go"
    return 0
  fi
  if ! curl -fsSL "$url" -o "${tmp}/${tarball}"; then
    warn "Не удалось скачать Go — сборка API пропущена"
    rm -rf "$tmp"
    return 1
  fi
  sudo rm -rf /usr/local/go
  sudo tar -C /usr/local -xzf "${tmp}/${tarball}"
  rm -rf "$tmp"
  export PATH="/usr/local/go/bin:${PATH}"
  go version
  ok "Go установлен."
}

# --- 6d. backend/.env --------------------------------------------------------
ensure_go_env() {
  local envf="${APP_DIR}/backend/.env"
  local example="${APP_DIR}/backend/.env.example"
  if [[ -f "$envf" ]]; then
    return 0
  fi
  if [[ ! -f "$example" ]]; then
    warn "Нет backend/.env и .env.example — создайте вручную."
    return 1
  fi
  log "Создаю backend/.env из .env.example (проверьте DB_PASS / UPLOAD_DIR)"
  if [[ "$DRY_RUN" -eq 0 ]]; then
    cp "$example" "$envf"
    chmod 600 "$envf"
    # testing defaults that we already know work
    if grep -q '^UPLOAD_DIR=' "$envf"; then
      sed -i 's|^UPLOAD_DIR=.*|UPLOAD_DIR=/var/www/corporate.admsr.ru/public|' "$envf"
    fi
    if grep -q '^HTTP_ADDR=' "$envf"; then
      sed -i 's|^HTTP_ADDR=.*|HTTP_ADDR=127.0.0.1:8081|' "$envf"
    else
      echo 'HTTP_ADDR=127.0.0.1:8081' >> "$envf"
    fi
    warn "Отредактируйте DB_PASS в ${envf} при первом деплое!"
  fi
}

# --- 6e. Сборка + systemd Go API ---------------------------------------------
deploy_go_api() {
  if [[ "$SKIP_GO" -eq 1 ]]; then
    warn "Go API пропущен (--skip-go / SKIP_GO=1)."
    return 0
  fi

  local backend="${APP_DIR}/backend"
  if [[ ! -d "$backend" ]]; then
    warn "Нет каталога backend/ — Go API пропущен."
    return 0
  fi

  ensure_go_env || true

  if ensure_go; then
    log "Сборка Go API…"
    if [[ "$DRY_RUN" -eq 0 ]]; then
      mkdir -p "${backend}/bin"
      (cd "$backend" && CGO_ENABLED=0 go build -o bin/api ./cmd/api)
      chmod +x "${backend}/bin/api"
      ok "Собран ${backend}/bin/api"
    else
      log "(dry-run) go build -o bin/api ./cmd/api"
    fi
  else
    if [[ -x "${backend}/bin/api" ]]; then
      warn "Go не найден — перезапуск существующего bin/api без пересборки."
    else
      err "Go не установлен и нет ${backend}/bin/api — установите Go или положите бинарник."
      exit 1
    fi
  fi

  # systemd unit
  local unit_src="${APP_DIR}/deploy/corporate-go-api.service"
  local unit_dst="/etc/systemd/system/${GO_SERVICE_NAME}.service"
  if [[ -f "$unit_src" ]]; then
    run "sudo cp '${unit_src}' '${unit_dst}'"
    run "sudo systemctl daemon-reload"
    run "sudo systemctl enable '${GO_SERVICE_NAME}'"
    # убить старый nohup, если слушал порт
    if [[ "$DRY_RUN" -eq 0 ]]; then
      fuser -k 8081/tcp 2>/dev/null || true
      # на случай старого :8080 (если вдруг)
      if ! ss -lntp 2>/dev/null | grep -q ':8080.*nginx'; then
        fuser -k 8080/tcp 2>/dev/null || true
      fi
    fi
    run "sudo systemctl restart '${GO_SERVICE_NAME}'"
    if [[ "$DRY_RUN" -eq 0 ]]; then
      sleep 1
      if systemctl is-active --quiet "$GO_SERVICE_NAME"; then
        ok "Сервис ${GO_SERVICE_NAME} активен."
      else
        err "Сервис ${GO_SERVICE_NAME} не запустился:"
        sudo journalctl -u "$GO_SERVICE_NAME" -n 30 --no-pager || true
        exit 1
      fi
    fi
  else
    warn "Нет ${unit_src} — fallback nohup"
    if [[ "$DRY_RUN" -eq 0 ]]; then
      fuser -k 8081/tcp 2>/dev/null || true
      (cd "$backend" && nohup ./bin/api >> /var/log/corporate-go-api.log 2>&1 &)
      sleep 1
    fi
  fi
}

# --- 6f. nginx (опционально) -------------------------------------------------
sync_nginx() {
  if [[ "${GO_SYNC_NGINX}" != "1" ]]; then
    return 0
  fi
  local src="${APP_DIR}/deploy/nginx-corporate.admsr.ru.conf"
  local dst="/etc/nginx/sites-available/corporate.admsr.ru.conf"
  if [[ ! -f "$src" ]]; then
    warn "Нет ${src} — sync nginx пропущен."
    return 0
  fi
  log "Обновление nginx site из репозитория (GO_SYNC_NGINX=1)"
  run "sudo cp '${src}' '${dst}'"
  if [[ "$DRY_RUN" -eq 0 ]]; then
    if sudo nginx -t; then
      sudo systemctl reload nginx
      ok "nginx перезагружен."
    else
      err "nginx -t failed — конфиг не применён"
      exit 1
    fi
  fi
}

# --- 7. Перезапуск PHP-FPM ---------------------------------------------------
restart_php() {
  if [[ "$DRY_RUN" -eq 1 ]]; then
    run "sudo systemctl reload '${SERVICE_NAME}'"
    return 0
  fi
  if systemctl is-active --quiet "$SERVICE_NAME" 2>/dev/null; then
    run "sudo systemctl reload '${SERVICE_NAME}'"
    ok "Сервис ${SERVICE_NAME} перезагружен."
  else
    warn "Сервис ${SERVICE_NAME} не активен — пропуск reload (проверьте php-fpm)."
  fi
}

# --- 8. Health-check ---------------------------------------------------------
health_check() {
  local i curl_args=(-fsS)
  if [[ "$HEALTH_URL" == http://127.0.0.1/* ]] || [[ "$HEALTH_URL" == http://localhost/* ]]; then
    HEALTH_URL="${HEALTH_URL/http:/https:}"
  fi
  if [[ "$HEALTH_URL" == https://127.0.0.1/* ]] || [[ "$HEALTH_URL" == https://localhost/* ]] \
     || [[ "${HEALTH_INSECURE:-0}" == "1" ]]; then
    curl_args+=(-k)
  fi
  if [[ "$HEALTH_URL" == *127.0.0.1* ]] || [[ "$HEALTH_URL" == *localhost* ]]; then
    curl_args+=(-H "Host: ${HEALTH_HOST}")
  fi

  log "Health-check: ${HEALTH_URL} (Host: ${HEALTH_HOST})"
  for ((i = 1; i <= HEALTH_RETRIES; i++)); do
    if [[ "$DRY_RUN" -eq 1 ]]; then
      ok "(dry-run) health-check пропущен"
      return 0
    fi
    local body
    body="$(curl "${curl_args[@]}" "$HEALTH_URL" 2>/dev/null || true)"
    if printf '%s' "$body" | grep -q '"ok"[[:space:]]*:[[:space:]]*true'; then
      if printf '%s' "$body" | grep -q '"backend"[[:space:]]*:[[:space:]]*"go"'; then
        ok "Health-check OK (Go API, попытка ${i}/${HEALTH_RETRIES})"
      else
        warn "Health OK, но backend не go — проверьте nginx allowlist / ${GO_SERVICE_NAME}"
        ok "Health-check OK (попытка ${i}/${HEALTH_RETRIES})"
      fi
      return 0
    fi
    if [[ -n "$body" ]]; then
      warn "Ответ health: ${body:0:200}"
    fi
    warn "Health-check не прошёл (попытка ${i}/${HEALTH_RETRIES}), ждём ${HEALTH_SLEEP}s…"
    sleep "$HEALTH_SLEEP"
  done

  err "Health-check не прошёл после ${HEALTH_RETRIES} попыток: ${HEALTH_URL}"
  warn "Проверьте: systemctl status ${GO_SERVICE_NAME}; journalctl -u ${GO_SERVICE_NAME} -n 50"
  warn "Вручную: curl -sk -H \"Host: ${HEALTH_HOST}\" https://127.0.0.1/api/health.php"
  exit 1
}

# --- main --------------------------------------------------------------------
log "Деплой corporate.admsr.ru → ${APP_DIR} (ветка: ${GIT_BRANCH})"
backup_database
reset_generated_files
git_update
install_deps
build_frontend
ensure_data_dirs
apply_migrations
deploy_go_api
sync_nginx
restart_php
health_check
ok "Деплой завершён успешно."
