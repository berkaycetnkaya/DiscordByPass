@echo off
title Derleme - Discord Bypass
cd /d "%~dp0"

echo [*] Go ile derleniyor...
"C:\Program Files\Go\bin\go.exe" build -ldflags="-s -w" -o discord-bypass.exe ./cmd
if %ERRORLEVEL% EQU 0 (
    echo [OK] discord-bypass.exe basariyla olusturuldu!
) else (
    echo [HATA] Derleme basarisiz oldu.
)
pause
