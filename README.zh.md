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
