@echo off
title Discord DPI Bypass
cd /d "%~dp0"

echo ========================================================
echo         DISCORD DPI BYPASS (TURKEY UNBLOCK)
echo ========================================================
echo.

REM 1. Takili kalmis eski Discord islemlerini temizle
taskkill /F /IM Discord.exe >nul 2>&1

REM 2. Port cakismasini onlemek icin eski bypass calisiyorsa kapat
taskkill /F /IM discord-bypass.exe >nul 2>&1
timeout /t 1 /nobreak >nul

REM 3. Proxy ortam degiskenlerini hazirla
set HTTP_PROXY=http://127.0.0.1:8085
set HTTPS_PROXY=http://127.0.0.1:8085
set http_proxy=http://127.0.0.1:8085
set https_proxy=http://127.0.0.1:8085
set ALL_PROXY=http://127.0.0.1:8085

REM 4. Tüneli calistir ve Discord'u baslat
echo [*] DPI Bypass tüneli ve Discord baslatiliyor...
"%~dp0discord-bypass.exe" -port 8085 -launch=true -sysproxy=true

REM 5. Kapatildiginda proxy'yi guvenle devre disi birak
reg add "HKCU\Software\Microsoft\Windows\CurrentVersion\Internet Settings" /v ProxyEnable /t REG_DWORD /d 0 /f >nul 2>&1
echo [*] Servis guvenle kapatildi.
