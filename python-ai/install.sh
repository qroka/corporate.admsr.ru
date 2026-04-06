#!/usr/bin/env bash
# ─── Установка ADMSR AI Chat Server на Ubuntu/Debian ─────────────────────────
# Запуск: bash install.sh
set -euo pipefail

echo "======================================================"
echo "  ADMSR AI Chat Server — Установка"
echo "======================================================"

# ── 1. Ollama ──────────────────────────────────────────────────────────────────
echo ""
echo "→ Устанавливаю Ollama..."
if command -v ollama &>/dev/null; then
    echo "  Ollama уже установлена: $(ollama --version)"
else
    curl -fsSL https://ollama.com/install.sh | sh
    echo "  Ollama установлена."
fi

# Запускаем службу Ollama
echo "→ Запускаю службу Ollama..."
sudo systemctl enable --now ollama 2>/dev/null || true
sleep 2

# ── 2. Скачиваем модель Qwen2.5:7b ────────────────────────────────────────────
echo ""
echo "→ Скачиваю модель Qwen2.5:7b (~4.7 ГБ, может занять несколько минут)..."
ollama pull qwen2.5:7b
echo "  Модель готова."

# ── 3. Python venv ────────────────────────────────────────────────────────────
echo ""
echo "→ Создаю виртуальное окружение Python..."
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

if ! command -v python3 &>/dev/null; then
    echo "  Python3 не найден. Устанавливаю..."
    sudo apt-get update -qq
    sudo apt-get install -y python3 python3-pip python3-venv
fi

python3 -m venv .venv
# shellcheck disable=SC1091
source .venv/bin/activate
pip install --upgrade pip -q
pip install -r requirements.txt -q
echo "  Зависимости установлены."

# ── 4. Systemd сервис ─────────────────────────────────────────────────────────
echo ""
echo "→ Создаю systemd-сервис admsr-ai..."

PYTHON_PATH="$SCRIPT_DIR/.venv/bin/python"

sudo tee /etc/systemd/system/admsr-ai.service > /dev/null <<EOF
[Unit]
Description=ADMSR AI Chat Server (Qwen2.5:7b via Ollama)
After=network.target ollama.service
Requires=ollama.service

[Service]
Type=simple
User=$USER
WorkingDirectory=$SCRIPT_DIR
ExecStart=$PYTHON_PATH $SCRIPT_DIR/server.py
Restart=on-failure
RestartSec=5
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
EOF

sudo systemctl daemon-reload
sudo systemctl enable admsr-ai
sudo systemctl start admsr-ai

echo ""
echo "======================================================"
echo "  Установка завершена!"
echo ""
echo "  Сервис:  http://127.0.0.1:8000"
echo "  Health:  curl http://127.0.0.1:8000/health"
echo ""
echo "  Управление:"
echo "    systemctl status admsr-ai"
echo "    systemctl restart admsr-ai"
echo "    journalctl -u admsr-ai -f"
echo "======================================================"
