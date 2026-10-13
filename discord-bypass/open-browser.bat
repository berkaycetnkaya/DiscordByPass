@echo off
title Discord Web Tarayici
cd /d "%~dp0"

echo ========================================================
echo       DISCORD WEB TARAYICI BASLATICI
echo ========================================================
echo.
echo [*] Tarayici aciliyor...

REM Google Chrome kontrolü
if exist "%ProgramFiles%\Google\Chrome\Application\chrome.exe" (
    start "" "%ProgramFiles%\Google\Chrome\Application\chrome.exe" --proxy-server="http://127.0.0.1:8085" "https://discord.com"
    exit /b
)
if exist "%ProgramFiles(x86)%\Google\Chrome\Application\chrome.exe" (
    start "" "%ProgramFiles(x86)%\Google\Chrome\Application\chrome.exe" --proxy-server="http://127.0.0.1:8085" "https://discord.com"
    exit /b
)

REM Varsayılan Microsoft Edge
start msedge --proxy-server="http://127.0.0.1:8085" "https://discord.com"
exit /b
