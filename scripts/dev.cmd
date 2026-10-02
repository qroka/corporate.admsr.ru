@echo off
rem Запуск локального стенда: база + Go API + Vite. Подробности — в scripts\dev.ps1
powershell -NoProfile -ExecutionPolicy Bypass -File "%~dp0dev.ps1" %*
