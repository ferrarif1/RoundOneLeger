@echo off
setlocal enabledelayedexpansion

echo ============================================
echo  准备离线部署依赖（Online）
echo ============================================

REM 指定离线包目录
set OFFLINE_DIR=%cd%\offline
mkdir %OFFLINE_DIR%

echo ---- 下载 Go 离线包 ----
mkdir %OFFLINE_DIR%\go
curl -L https://go.dev/dl/go1.23.0.windows-amd64.zip -o %OFFLINE_DIR%\go\go.zip

echo ---- 下载 Node.js 离线包 ----
mkdir %OFFLINE_DIR%\node
curl -L https://nodejs.org/dist/v20.11.0/node-v20.11.0-win-x64.zip -o %OFFLINE_DIR%\node\node.zip

echo ---- 下载 PostgreSQL Windows 安装包 ----
mkdir %OFFLINE_DIR%\postgres
curl -L https://get.enterprisedb.com/postgresql/postgresql-15.5-1-windows-x64.exe -o %OFFLINE_DIR%\postgres\pgsql.exe

echo ---- 下载前端依赖离线包（npm pack 全部依赖）----
cd web
mkdir ..\offline\npm-packages
npm install
npm pack --pack-destination ../offline/npm-packages
cd ..

echo ---- 拷贝项目代码 ----
xcopy /E /I . %OFFLINE_DIR%\roundoneleger

echo 完成！离线包目录：
echo %OFFLINE_DIR%
pause
