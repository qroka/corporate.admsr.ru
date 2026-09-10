@echo off
REM ============================================================
REM  Настройки киоска — отредактируйте перед установкой
REM ============================================================

REM URL киоска (продакшен)
set "KIOSK_URL=https://newcorp.admsr.ru/kiosk"

REM Браузер: edge | chrome  (рекомендуется edge)
set "KIOSK_BROWSER=edge"

REM Имя ярлыка в автозагрузке
set "KIOSK_SHORTCUT_NAME=ADMSR Kiosk"

REM Пауза перед стартом после входа в Windows (секунды) — дать сети подняться
set "KIOSK_START_DELAY=8"

REM Интервал проверки, что браузер жив (секунды) для watchdog
set "KIOSK_WATCH_INTERVAL=15"
