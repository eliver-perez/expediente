@echo off
title Configurar servidor AIDEX
powershell.exe -NoProfile -NoExit -ExecutionPolicy Bypass -File "%~dp0manage.ps1" -Action Port
