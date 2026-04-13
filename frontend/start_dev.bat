@echo off
echo Inicializando Servidor Vite com Bypass de Path...
set PATH=%PATH%;C:\Program Files\nodejs\
call npm.cmd run dev
pause
