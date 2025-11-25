# RoundOne Ledger

轻量 Go 后端 + React 控制台，用于管理 IP / 人员 / 系统台账，数据持久化到 PostgreSQL。

## 环境
- Go 1.23+
- PostgreSQL 15+（默认端口 5433）
- Docker / Docker Compose（可选）

## 快速开始
```bash
docker-compose up --build   # 启动 Postgres:5433 和后端:8080
# 或
make build && make run      # 使用本地 Postgres，按需设置环境变量
```
健康检查：<http://localhost:8080/health>

## 环境变量
| 变量 | 默认值 | 说明 |
| --- | --- | --- |
| DB_HOST | localhost | |
| DB_PORT | 5433 | |
| DB_NAME | ledger | |
| DB_USER | postgres | |
| DB_PASS | postgres | |
| LEDGER_ADMIN_PASSWORD | *(可选)* | 初始化 `hzdsz_admin` 的明文密码 |
| LEDGER_ADMIN_PASSWORD_HASH | *(可选)* | PBKDF2-HMAC-SHA256 哈希 |

默认管理员 `hzdsz_admin` 会自动创建；请在首登后新建个人账号并删除默认账号。

## 导入 / 导出
- 快照存储在 Postgres `snapshots` 表，`LEDGER_DATA_DIR` 用于资产文件。
- `GET /api/v1/export/all`：下载包含 `snapshot.sql` 与 `assets/` 的 ZIP。
- `POST /api/v1/import/all`：上传 ZIP/SQL/JSON，可同时恢复资产。
- Ledger/Workspace 仍支持 XLSX 导入导出。

## 认证
- 登录：`POST /auth/password-login`，返回 token。
- 访问 `/api/v1/**` 需携带 `Authorization: Bearer <token>`。

## 测试
```bash
go test ./...
```

完整接口见 `openapi.yaml`。

## Docker 镜像导出/导入
- 导出：
  ```bash
  docker save -o roundoneleger-frontend.tar roundoneleger-frontend:latest
  docker save -o roundoneleger-app.tar roundoneleger-app:latest
  docker save -o postgres-15-alpine.tar mirror.gcr.io/library/postgres:15-alpine
  ```
  生成的文件应包含：
  ```
  dockerimgs/
  ├── roundoneleger-frontend.tar
  ├── roundoneleger-app.tar
  └── postgres-15-alpine.tar
  ```
- 新机器导入：
  ```bash
  docker load -i roundoneleger-frontend.tar
  docker load -i roundoneleger-app.tar
  docker load -i postgres-15-alpine.tar
  ```
- 启动（推荐 docker compose）：
  ```bash
  docker compose up -d
  ```
  或手动：
  ```bash
  docker run -d --name db -e POSTGRES_USER=ledger -e POSTGRES_PASSWORD=ledger123 -e POSTGRES_DB=ledgerdb -p 5433:5432 mirror.gcr.io/library/postgres:15-alpine
  docker run -d --name app -p 8080:8080 --link db:db roundoneleger-app:latest
  docker run -d --name frontend -p 5173:80 roundoneleger-frontend:latest
  ```
