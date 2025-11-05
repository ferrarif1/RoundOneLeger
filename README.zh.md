# RoundOneLeger

RoundOneLeger 是一套轻量的资产台账方案，包含 Go 后端与 React（Vite）控制台，可统一管理 IP、人员、系统，并提供协同工作区与审计追踪。

## 提供能力
- 覆盖认证、台账、历史、审计与协作工作区的 REST 接口
- 内置 XLSX 工具，支持表格导入导出
- 默认管理员账号 `hzdsz_admin` / `Hzdsz@2025#`，并结合会话与 IP 白名单机制

## 仓库结构
```
cmd/server            # HTTP 服务入口
internal/             # 处理器、认证、存储、中间件及 XLSX 工具
migrations/           # PostgreSQL 示例表结构
openapi.yaml          # API 说明
web/                  # React 管理端
```

## 快速上手
### 后端
```bash
make run          # 在 http://localhost:8080 启动接口
make test         # 运行 Go 测试
```
如需连接 PostgreSQL，可设置 `DB_HOST`、`DB_PORT`、`DB_USER`、`DB_PASS` 等环境变量。

### 前端
```bash
cd web
npm install
npm run dev       # 在 http://localhost:5173 启动 Vite 开发服务器
```
前端默认走同源接口，若单独部署可通过 `VITE_API_BASE_URL` 指定后端地址。

## Docker Compose
使用一条命令启动后端与 PostgreSQL：
```bash
docker-compose up --build
```

## 登录流程
1. 调用 `POST /auth/password-login` 使用默认管理员获取 Token。
2. 之后的 `/api/v1/**` 请求在 Header 中携带 `Authorization: Bearer <token>`。
3. 登录后可在“用户中心”页面管理其他操作员。

更多接口示例请参考 `openapi.yaml` 或直接查看前端实现。
