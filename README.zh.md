# RoundOneLeger

## 概述
RoundOneLeger 是一套资产台账系统，用于集中管理 IP 地址、设备、人员以及配套的审计数据。后端使用 Go 编写并集成了 Gin 兼容路由，前端则采用 React 与 Tailwind 风格的样式，提供贴近 Eidos 设计语言的管理控制台。

## 核心特性
- **台账管理接口**：覆盖 IP、设备、人员与系统等实体，支持手动新增、修改、删除、排序以及标签元数据。
- **Excel 导入导出流程**：能够自动规范表头、基于正则识别 IP 字段，并生成多表或笛卡尔积式的报表文件。
- **撤销/重做历史**：记录所有变更操作，允许管理员在双向最多十步内撤销或重做。
- **认证与白名单能力**：提供基于 SDID 钱包的一次性随机数登录流程，并支持可选的固定网络白名单控制。
- **可验证的审计日志**：提供导出签名链和校验历史完整性的接口，保障日志不可篡改。

## 架构组成
- **后端（`cmd/server`, `internal/`）**：Go 1.22 服务，通过自带的 Gin 兼容封装暴露 REST 接口，内置台账存储、认证辅助和中间件。
- **Excel 工具（`internal/xlsx`）**：无外部依赖的 XLSX 读写模块，为导入导出流程提供支持。
- **OpenAPI 规范（`openapi.yaml`）**：描述认证、台账、白名单、审计与历史记录等接口，方便客户端集成。
- **前端（`web/`）**：基于 React + Vite 的控制台，包含 Tailwind 配置、会话管理钩子、台账管理页面以及偏向 Ant Design 风格的组件。
- **配套工具**：Makefile 提供构建/运行/测试任务，后端与前端 Dockerfile、docker-compose 编排以及 GitHub Actions CI 模板。

## 快速开始
### 环境依赖
- Go 1.22 及以上
- Node.js 18 及以上（用于前端）
- Docker 与 Docker Compose（可选，用于容器化流程）
- 如需连接真实数据库，请准备 PostgreSQL 并通过环境变量配置

### 本地后端
1. 安装 Go 依赖（项目内已自带 Gin 兼容实现，可选执行 `go mod tidy`）。
2. 如使用 PostgreSQL，请设置 `DB_HOST`、`DB_PORT`、`DB_NAME`、`DB_USER`、`DB_PASS` 等环境变量。
3. 启动接口服务：
   ```bash
   make run
   ```
   健康检查地址 `http://localhost:8080/health` 会返回 `{"status":"ok"}`。
4. 运行测试：
   ```bash
   make test
   ```

### 前端工作区
1. 进入 `web/` 并安装依赖：
   ```bash
   npm install
   ```
2. 启动开发服务器：
   ```bash
   npm run dev
   ```
3. 配置 Vite 代理或环境变量，使前端请求指向 Go 后端（默认 `http://localhost:8080`）。

### Docker Compose
项目内提供组合方案便于本地体验：
```bash
docker-compose up --build
```
该命令会启动 Postgres 与后端容器。运行前可根据需要修改 `docker-compose.yml` 中的环境变量。

### 认证
- 前端点击“使用 SDID 一键登录”后会向 `/auth/request-nonce` 请求随机数，响应包含 `nonce` 与需要签名的文本 `message`。
- [ferrarif1/SDID](https://github.com/ferrarif1/SDID) 浏览器插件会调用 `requestLogin`，返回 DID、包含 ECDSA P-256 公钥的 `publicKeyJwk`、canonical 请求体以及位于 `signature` 或 `proof.signatureValue` 中的签名。
- 将 `{nonce, response}` 提交到 `/auth/login`，其中 `response` 为插件的原始返回值；后端会重建 canonical 请求并基于提供的 JWK 验证 DER 编码的签名，无需本地注册或管理员审批，身份完全由 SDID 管理。
- 若需限制访问来源，可继续在控制台维护 IP 白名单，只有来自允许网段的请求才可访问受保护接口。

## 目录结构
```
.
├── cmd/server            # 接口入口与 HTTP 服务启动逻辑
├── internal/api          # HTTP 处理器与路由注册
├── internal/auth         # 会话令牌管理器
├── internal/db           # 数据库配置与连接工具
├── internal/middleware   # Gin 中间件（如 IP 白名单）
├── internal/models       # 数据模型与内存存储实现
├── internal/xlsx         # Excel 读写工具
├── migrations            # 数据库迁移文件（请自行补充）
├── openapi.yaml          # OpenAPI v3 规范
├── third_party/gin       # 轻量级 Gin 兼容封装
└── web                   # React 管理控制台
```

## 测试与质量
- 通过 `make test` 执行 Go 测试用例。
- `.github/workflows/ci.yml` 会在每次推送时运行 `go test ./...` 与 `go vet ./...`。

## 其他说明
- `migrate` 目标目前为占位符，可根据团队习惯集成如 golang-migrate 等工具。

