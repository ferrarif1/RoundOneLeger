@echo off
setlocal

echo ============================================
echo  离线部署 RoundOneLeger（Offline Install）
echo ============================================

set OFFLINE_DIR=%cd%\offline

echo ---- 解压安装 Go ----
powershell Expand-Archive -Path "%OFFLINE_DIR%\go\go.zip" -DestinationPath "C:\"
setx PATH "%PATH%;C:\go\bin"

echo ---- 解压安装 Node.js ----
powershell Expand-Archive -Path "%OFFLINE_DIR%\node\node.zip" -DestinationPath "C:\node"
setx PATH "%PATH%;C:\node"

echo ---- 安装 PostgreSQL ----
"%OFFLINE_DIR%\postgres\pgsql.exe" --mode unattended --superpassword "123456" --prefix "C:\pgsql"

echo ---- 初始化数据库 ----
"C:\pgsql\bin\createdb.exe" -U postgres roundoneleger

echo ---- 安装前端离线依赖 ----
cd roundoneleger\web
npm install --offline --cache "%OFFLINE_DIR%\npm-packages"
cd ..\..

echo ---- 完成离线安装 ----
pause
