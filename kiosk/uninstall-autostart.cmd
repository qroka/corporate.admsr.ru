@echo off
setlocal EnableExtensions
cd /d "%~dp0"
call "%~dp0config.cmd"

set "STARTUP=%APPDATA%\Microsoft\Windows\Start Menu\Programs\Startup"
set "LNK=%STARTUP%\%KIOSK_SHORTCUT_NAME%.lnk"

if exist "%LNK%" (
  del /f /q "%LNK%"
  echo OK: автозапуск удалён
) else (
  echo Ярлык автозапуска не найден: %LNK%
)

pause
