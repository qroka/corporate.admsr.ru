#!/usr/bin/env bash
# ─── Ручной запуск сервера (без systemd) ─────────────────────────────────────
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

# Активируем venv если есть
if [[ -f ".venv/bin/activate" ]]; then
    # shellcheck disable=SC1091
    source .venv/bin/activate
fi

echo "Запускаю ADMSR AI Chat Server на порту 8000..."
echo "Остановить: Ctrl+C"
echo ""

python server.py
