@echo off
setlocal EnableExtensions
cd /d "%~dp0"
call "%~dp0config.cmd"

echo ADMSR Kiosk watchdog
echo URL: %KIOSK_URL%
echo Browser: %KIOSK_BROWSER%
echo.

if defined KIOSK_START_DELAY (
  echo Ожидание сети %KIOSK_START_DELAY% сек...
  timeout /t %KIOSK_START_DELAY% /nobreak >nul
)

:loop
call "%~dp0start-kiosk.cmd"

REM Ждём, пока процесс браузера жив; если закрыли — перезапускаем
:wait
timeout /t %KIOSK_WATCH_INTERVAL% /nobreak >nul

if /I "%KIOSK_BROWSER%"=="chrome" (
  tasklist /FI "IMAGENAME eq chrome.exe" 2>nul | find /I "chrome.exe" >nul
) else (
  tasklist /FI "IMAGENAME eq msedge.exe" 2>nul | find /I "msedge.exe" >nul
)

if errorlevel 1 (
  echo [%date% %time%] Браузер закрыт — перезапуск...
  goto loop
)
goto wait
