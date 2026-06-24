#!/usr/bin/env bash
# deploy.sh — обновление corporate.admsr.ru на Ubuntu (Vue SPA + PHP API + PostgreSQL)
#
# Стек отличается от grafic.admsr.ru: здесь нет Node/Fastify — API на PHP-FPM.
# Переменные можно переопределить через deploy/deploy.env или окружение.

set -euo pipefail

# --- Параметры по умолчанию (corporate.admsr.ru) ---------------------------
APP_DIR="${APP_DIR:-/var/www/corporate.admsr.ru}"
DATA_DIR="${DATA_DIR:-/var/lib/corporate-app}"
BACKUP_DIR="${BACKUP_DIR:-${DATA_DIR}/backups}"
BACKUP_KEEP="${BACKUP_KEEP:-7}"
DOMAIN="${DOMAIN:-corporate.admsr.ru}"
SERVICE_NAME="${SERVICE_NAME:-php8.3-fpm}"
HEALTH_URL="${HEALTH_URL:-http://127.0.0.1/api/health.php}"
HEALTH_HOST="${HEALTH_HOST:-${DOMAIN}}"
PACKAGE_MANAGER="${PACKAGE_MANAGER:-npm}"
GIT_BRANCH="${GIT_BRANCH:-main}"
HEALTH_RETRIES="${HEALTH_RETRIES:-10}"
HEALTH_SLEEP="${HEALTH_SLEEP:-2}"

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
  -h, --help             Справка

Переменные окружения (или deploy/deploy.env):
  APP_DIR, DATA_DIR, BACKUP_KEEP, SERVICE_NAME, HEALTH_URL, HEALTH_HOST,
  PACKAGE_MANAGER (npm|pnpm), DOMAIN, GIT_BRANCH

Пример:
  ./deploy/deploy.sh
  APP_DIR=/var/www/corporate.admsr.ru ./deploy/deploy.sh -b main
EOF
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    -b|--branch) GIT_BRANCH="$2"; shift 2 ;;
    -n|--dry-run) DRY_RUN=1; shift ;;
    --skip-backup) SKIP_BACKUP=1; shift ;;
    --skip-pull) SKIP_PULL=1; shift ;;
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

    # ротация
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
  run "sudo mkdir -p '${DATA_DIR}/uploads/img' '${BACKUP_DIR}'"
  run "sudo chown -R www-data:www-data '${DATA_DIR}'"
  # Загрузки галереи/аватаров — вне git (public/img в .gitignore)
  if [[ "$DRY_RUN" -eq 0 ]] && [[ ! -e public/img ]]; then
    run "ln -sfn '${DATA_DIR}/uploads/img' public/img"
  fi
}

# --- 7. Перезапуск PHP-FPM ---------------------------------------------------
restart_service() {
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
    curl_args+=(-H "Host: ${HEALTH_HOST}")
  fi

  log "Health-check: ${HEALTH_URL} (Host: ${HEALTH_HOST})"
  for ((i = 1; i <= HEALTH_RETRIES; i++)); do
    if [[ "$DRY_RUN" -eq 1 ]]; then
      ok "(dry-run) health-check пропущен"
      return 0
    fi
    if curl "${curl_args[@]}" "$HEALTH_URL" | grep -q '"ok"[[:space:]]*:[[:space:]]*true'; then
      ok "Health-check OK (попытка ${i}/${HEALTH_RETRIES})"
      return 0
    fi
    warn "Health-check не прошёл (попытка ${i}/${HEALTH_RETRIES}), ждём ${HEALTH_SLEEP}s…"
    sleep "$HEALTH_SLEEP"
  done

  err "Health-check не прошёл после ${HEALTH_RETRIES} попыток: ${HEALTH_URL}"
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
restart_service
health_check
ok "Деплой завершён успешно."
