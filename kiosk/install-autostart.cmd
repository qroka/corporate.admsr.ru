@echo off
setlocal EnableExtensions
cd /d "%~dp0"
call "%~dp0config.cmd"

REM Автозагрузка текущего пользователя: ярлык на watchdog.cmd
set "STARTUP=%APPDATA%\Microsoft\Windows\Start Menu\Programs\Startup"
set "TARGET=%~dp0watchdog.cmd"
set "LNK=%STARTUP%\%KIOSK_SHORTCUT_NAME%.lnk"

powershell -NoProfile -ExecutionPolicy Bypass -Command ^
  "$ws = New-Object -ComObject WScript.Shell; $s = $ws.CreateShortcut('%LNK%'); $s.TargetPath = '%TARGET%'; $s.WorkingDirectory = '%~dp0'; $s.WindowStyle = 7; $s.Description = 'ADMSR Corporate Portal Kiosk'; $s.Save()"

if exist "%LNK%" (
  echo OK: автозапуск установлен
  echo Ярлык: %LNK%
  echo При входе в Windows запустится киоск: %KIOSK_URL%
) else (
  echo ERROR: не удалось создать ярлык
  exit /b 1
)

echo.
echo Рекомендации для стойки:
echo  1. Отдельная учётка Windows без прав администратора
echo  2. Автологин этой учётки
echo  3. Edge: Параметры -^> Киоск / назначенный доступ ^(если Windows Pro/Enterprise^)
echo  4. Отключить жесты края экрана и Ctrl+Alt+Del по политике, если нужно сильнее закрыть ОС
echo.
pause
