@echo off
setlocal

echo ============================================
echo  启动所有服务（Postgres → Go 后端 → Web 前端）
echo ============================================

echo ---- 启动 PostgreSQL ----
net start postgresql-x64-15

echo ---- 启动 Go 后端 ----
cd roundoneleger
start cmd /k "go run ./cmd/server"
cd ..

echo ---- 启动前端 ----
cd roundoneleger\web
start cmd /k "npm run dev"
cd ..\..

echo ---- 全部启动完成 ----
echo 后端      http://localhost:8080
echo 前端      http://localhost:5173
pause
