<#
  Локальный запуск портала одной командой: база (Docker) + Go API (:8080) + Vite (:5173).

    powershell -ExecutionPolicy Bypass -File scripts\dev.ps1
    scripts\dev.cmd                      # то же, двойным щелчком / из cmd

  Что делает:
    1. запускает Docker Desktop, если он не запущен, и ждёт движок;
    2. поднимает локальный Postgres (db/local-dev, порт 55432) и ждёт готовности;
    3. собирает backend\bin\api.exe и запускает его в фоне;
    4. запускает Vite на переднем плане. Ctrl+C останавливает Vite и API (база остаётся работать).

  Ключи:
    -NoBuild   не пересобирать API (взять готовый backend\bin\api.exe)
    -DryRun    только проверить предусловия и показать, что будет запущено

  Логины тестовых пользователей — в комментарии db/local-dev/20_seed.sql.
#>
param(
  [switch]$NoBuild,
  [switch]$DryRun
)

$ErrorActionPreference = 'Stop'
$root = Split-Path -Parent $PSScriptRoot
Set-Location $root

function Step($text) { Write-Host "[dev] $text" -ForegroundColor Cyan }
function Fail($text) { Write-Host "[dev] $text" -ForegroundColor Red; exit 1 }

function Test-PortBusy([int]$port) {
  [bool](Get-NetTCPConnection -State Listen -LocalPort $port -ErrorAction SilentlyContinue)
}

foreach ($tool in 'docker', 'go', 'npm') {
  if (-not (Get-Command $tool -ErrorAction SilentlyContinue)) { Fail "Не найден '$tool' в PATH." }
}
foreach ($port in 8080, 5173) {
  if (Test-PortBusy $port) { Fail "Порт $port уже занят — остановите запущенный сервер (или закройте предыдущий запуск скрипта)." }
}

if ($DryRun) {
  Step 'DryRun: docker/go/npm найдены, порты 8080 и 5173 свободны.'
  Step 'Будет: Docker Desktop → db/local-dev → go build backend\bin\api.exe → api.exe (:8080) → npm run dev (:5173).'
  exit 0
}

# 1. Docker
docker info *> $null
if ($LASTEXITCODE -ne 0) {
  Step 'Запускаю Docker Desktop…'
  $desktop = Join-Path $env:ProgramFiles 'Docker\Docker\Docker Desktop.exe'
  if (-not (Test-Path $desktop)) { Fail "Docker Desktop не найден: $desktop" }
  Start-Process $desktop
  $ready = $false
  for ($i = 0; $i -lt 60; $i++) {
    Start-Sleep -Seconds 5
    docker info *> $null
    if ($LASTEXITCODE -eq 0) { $ready = $true; break }
  }
  if (-not $ready) { Fail 'Docker не запустился за 5 минут.' }
}
Step 'Docker готов.'

# 2. База
Step 'Поднимаю Postgres (db/local-dev)…'
docker compose -f db/local-dev/docker-compose.yml up -d
if ($LASTEXITCODE -ne 0) { Fail 'docker compose up не удался.' }
$dbReady = $false
for ($i = 0; $i -lt 30; $i++) {
  docker exec corporate-portal-db pg_isready -U devuser *> $null
  if ($LASTEXITCODE -eq 0) { $dbReady = $true; break }
  Start-Sleep -Seconds 2
}
if (-not $dbReady) { Fail 'Postgres не ответил за 60 секунд.' }
Step 'Postgres готов (127.0.0.1:55432).'

# 3. Go API
$api = Join-Path $root 'backend\bin\api.exe'
if (-not $NoBuild -or -not (Test-Path $api)) {
  Step 'Собираю Go API…'
  Push-Location (Join-Path $root 'backend')
  go build -o bin\api.exe .\cmd\api
  $buildCode = $LASTEXITCODE
  Pop-Location
  if ($buildCode -ne 0) { Fail 'go build не удался.' }
}

# те же настройки, что в .claude/launch.json
$env:DB_HOST = '127.0.0.1'
$env:DB_PORT = '55432'
$env:DB_USER = 'devuser'
$env:DB_PASS = 'devpass'

Step 'Запускаю Go API на :8080…'
$apiProc = Start-Process -FilePath $api -WorkingDirectory (Join-Path $root 'backend') -PassThru -NoNewWindow
Start-Sleep -Seconds 2
if ($apiProc.HasExited) { Fail 'Go API завершился сразу после старта — смотрите вывод выше.' }

# 4. Vite
try {
  Step 'Запускаю Vite: http://localhost:5173  (Ctrl+C — остановить)'
  npm run dev -- --host 127.0.0.1
}
finally {
  if ($apiProc -and -not $apiProc.HasExited) {
    Step 'Останавливаю Go API…'
    Stop-Process -Id $apiProc.Id -Force
  }
  Step 'Готово. База осталась запущенной (docker compose -f db/local-dev/docker-compose.yml down — остановить).'
}
