@echo off
title Discord DPI Bypass - Durdurucu
echo [*] Discord ve DPI Bypass islemleri kapatiliyor...
taskkill /F /IM Discord.exe >nul 2>&1
taskkill /F /IM discord-bypass.exe >nul 2>&1

echo [*] Sistem vekil ayarlari normale donduruluyor...
reg add "HKCU\Software\Microsoft\Windows\CurrentVersion\Internet Settings" /v ProxyEnable /t REG_DWORD /d 0 /f >nul 2>&1

echo [OK] Tum servisler durduruldu ve ayarlar normale donduruldu.
timeout /t 2
