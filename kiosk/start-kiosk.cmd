@echo off
setlocal EnableExtensions
cd /d "%~dp0"
call "%~dp0config.cmd"

REM --- Поиск браузера ---
set "BROWSER_EXE="

if /I "%KIOSK_BROWSER%"=="chrome" goto find_chrome
goto find_edge

:find_edge
if exist "%ProgramFiles(x86)%\Microsoft\Edge\Application\msedge.exe" set "BROWSER_EXE=%ProgramFiles(x86)%\Microsoft\Edge\Application\msedge.exe"
if exist "%ProgramFiles%\Microsoft\Edge\Application\msedge.exe" set "BROWSER_EXE=%ProgramFiles%\Microsoft\Edge\Application\msedge.exe"
if defined BROWSER_EXE goto launch
echo [ERROR] Microsoft Edge не найден. Поставьте Edge или смените KIOSK_BROWSER=chrome в config.cmd
exit /b 1

:find_chrome
if exist "%ProgramFiles%\Google\Chrome\Application\chrome.exe" set "BROWSER_EXE=%ProgramFiles%\Google\Chrome\Application\chrome.exe"
if exist "%ProgramFiles(x86)%\Google\Chrome\Application\chrome.exe" set "BROWSER_EXE=%ProgramFiles(x86)%\Google\Chrome\Application\chrome.exe"
if exist "%LocalAppData%\Google\Chrome\Application\chrome.exe" set "BROWSER_EXE=%LocalAppData%\Google\Chrome\Application\chrome.exe"
if defined BROWSER_EXE goto launch
echo [ERROR] Google Chrome не найден. Поставьте Chrome или смените KIOSK_BROWSER=edge в config.cmd
exit /b 1

:launch
REM Закрыть уже открытые окна этого браузера (чтобы не было второго экрана)
if /I "%KIOSK_BROWSER%"=="chrome" (
  taskkill /IM chrome.exe /F >nul 2>&1
) else (
  taskkill /IM msedge.exe /F >nul 2>&1
)

REM Небольшая пауза после kill
timeout /t 1 /nobreak >nul

if /I "%KIOSK_BROWSER%"=="chrome" goto launch_chrome

REM Edge: fullscreen kiosk — без вкладок, адресной строки, меню
REM Выход из киоска: Alt+F4 (или Ctrl+Alt+Del → диспетчер задач, если разрешён)
start "" "%BROWSER_EXE%" ^
  --kiosk "%KIOSK_URL%" ^
  --edge-kiosk-type=fullscreen ^
  --no-first-run ^
  --disable-features=TranslateUI ^
  --disable-pinch ^
  --overscroll-history-navigation=0
exit /b 0

:launch_chrome
start "" "%BROWSER_EXE%" ^
  --kiosk "%KIOSK_URL%" ^
  --no-first-run ^
  --disable-infobars ^
  --disable-session-crashed-bubble ^
  --disable-translate ^
  --disable-features=TranslateUI ^
  --disable-pinch ^
  --overscroll-history-navigation=0 ^
  --check-for-update-interval=31536000
exit /b 0
