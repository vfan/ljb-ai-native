# 大学生信息管理系统

基于 [Eino](https://github.com/cloudwego/eino) 构建的大学生信息管理系统，集成 AI 智能助手，并同时支持**浏览器登录**、**个人访问令牌（PAT）**和 **OAuth2 授权码 + PKCE** 三种访问方式，方便外部 AI Agent 安全接入。

## 技术栈

- **后端**: Go + Gin + GORM + SQLite
- **前端**: React 19 + TypeScript + Ant Design + Vite
- **AI**: Eino ReAct Agent + DeepSeek API
- **对外能力**: REST API + Skill 技能文档 + `eino-oa-cli` 授权工具

## 功能特性

- 学生 / 课程 / 成绩管理（增删改查、搜索筛选、分页）
- 成绩统计、学分加权 GPA、学业预警
- AI 智能助手（自然语言操作系统，11 个工具）
- 会话历史持久化，按登录用户隔离
- 登录鉴权、登录限流，账号密码由 `backend/.env` 管理
- 个人访问令牌（PAT）：只读 / 读写、有效期、随时吊销
- OAuth2 授权码 + PKCE：CLI 打开浏览器授权，短期令牌自动续期
- Skill 技能文档，可被外部 Agent 直接下载

## 快速开始

> 环境要求：Go 1.25+、Node.js 20.19+ / 22.12+、npm 10+。
> SQLite 驱动需要 CGO：macOS 执行 `xcode-select --install`，Linux 安装 `gcc` / `build-essential`。

### 1. 准备配置

所有配置只读 **`backend/.env`**（与 `backend/go.mod` 同级）。仓库根目录的 `.env` 不生效。

```bash
cd backend
cp .env.example .env
```

编辑 `backend/.env`：

```dotenv
ADMIN_USERNAME=admin
# 修改后重启生效
ADMIN_PASSWORD=change-me-123
# 留空则关闭 AI 助手，其它功能不受影响
DEEPSEEK_API_KEY=sk-...
```

- `ADMIN_USERNAME` 默认 `admin`，账号不存在时会自动创建；
- `ADMIN_PASSWORD` **必填**；未设置时后端会直接启动失败并提示；
- 每次启动都会把这两个值同步到数据库，**改密码就是改 `backend/.env` 后重启**；
- 修改密码后，旧的浏览器会话和访问令牌会自动失效。

### 2. 启动后端

```bash
cd backend
go mod tidy
go run ./cmd/server
```

启动成功会看到：

```text
已加载配置文件: .env
管理员账号 "admin" 已就绪
服务启动在 :8080
```

### 3. 启动前端

```bash
cd frontend
npm install
npm run dev
```

访问 <http://localhost:3000>，使用 `backend/.env` 里的管理员账号登录。

Vite 会自动读取 `backend/.env` 的 `SERVER_PORT` 作为代理目标（默认 8080）。
需要临时指向其他后端时：

```bash
VITE_BACKEND_URL=http://localhost:8081 npm run dev
```

### 4. 生产构建

```bash
cd frontend
npm run build
```

构建产物在 `frontend/dist`。后端检测到该目录会直接托管静态文件，从而保证登录 Cookie、OAuth 授权页和 API 同源。

更详细的启动、配置和排错说明：

- [后端说明](backend/README.md)
- [前端说明](frontend/README.md)

## 登录与鉴权

系统有三类凭证，都通过 `Authorization: Bearer <token>` 或浏览器 Cookie 使用：

| 凭证 | 获取方式 | 有效期 | 适用场景 |
| --- | --- | --- | --- |
| 登录会话 | 网页用户名 / 密码登录 | 默认 7 天 | 浏览器操作 |
| PAT | 登录后「访问令牌」页面创建 | 7 天 / 30 天 / 90 天 / 1 年 / 永久 | 外部 Agent、脚本 |
| OAuth Access Token | `eino-oa-cli login` 授权后签发 | 1 小时，可自动刷新 | CLI / MCP / 桌面 Agent |

### 方式一：给 Agent 一段话 + PAT

登录网页 →「访问令牌」→「新建令牌」→ 选择只读或读写 → 复制**接入指令**，直接发给 Agent。
指令里包含技能文档地址、`EINO_OA_BASE_URL` 和 `EINO_OA_API_TOKEN`。

也可以用 curl 手动创建：

```bash
curl -X POST http://localhost:8080/api/auth/tokens \
  -H "Cookie: eo_session=..." \
  -H "Content-Type: application/json" \
  -d '{"name":"my-agent","scopes":"read","expires_in_days":90}'
```

返回的 `token` 只显示一次。

### 方式二：Agent 用 CLI 打开授权页（推荐）

```bash
cd backend
go build -o eino-oa-cli ./cmd/eino-oa-cli

./eino-oa-cli login --server http://localhost:8080
# 浏览器打开授权页 → 点击「同意授权」
# 凭证保存到 ~/.eino-oa/credentials.json

export EINO_OA_API_TOKEN="$(./eino-oa-cli token)"
```

授权页会显示请求方 `client_id` 和申请的权限范围。CLI 只监听
`127.0.0.1` 的临时端口接收回调，授权码 5 分钟过期且只能使用一次。

> 如果浏览器和 CLI 不在同一台机器（例如远程 SSH），需要把授权链接复制到本机打开；
> 更彻底的方案是 Device Authorization Grant（设备码轮询），当前版本未实现。

### 令牌权限

- `read`：查询学生、课程、成绩、统计、GPA、预警。
- `read,write`：额外允许新增 / 修改 / 删除。
- 只读令牌调用写接口会返回 `403`；令牌管理只允许浏览器登录态。
- 「访问令牌」页面的「吊销 Agent 授权」会立即吊销所有 CLI / OAuth 的 access + refresh token。

### OAuth2 端点

| 端点 | 说明 |
| --- | --- |
| `GET /oauth/authorize` | 授权页，支持 `authorization_code` + `S256` PKCE |
| `POST /oauth/authorize` | 用户同意 / 拒绝授权 |
| `POST /oauth/token` | 授权码换令牌、刷新令牌（刷新时轮换） |
| `GET /.well-known/oauth-authorization-server` | 授权服务器元数据 |

公开客户端：`eino-oa-cli`、`eino-oa-mcp`、`eino-oa-agent`。回调地址只允许
`http://127.0.0.1:<port>/callback` 或 `http://localhost:<port>/callback`。

### Skill 文档

后端会把 `skills/` 目录挂到 `/skills`，Agent 可直接下载：

```bash
curl http://localhost:8080/skills/student-manage/SKILL.md
curl http://localhost:8080/skills/grade-analysis/SKILL.md
```

文档本身不含任何密钥，访问 API 时仍需携带令牌。

## 环境变量

配置文件固定为 **`backend/.env`**，后端启动时自动加载；从仓库根目录启动会读取 `backend/.env`。
根目录的 `.env` 会被忽略。环境变量本身优先于 `.env` 文件。

| 变量名 | 说明 | 默认值 |
| --- | --- | --- |
| `SERVER_PORT` | HTTP 服务端口 | 8080 |
| `DB_PATH` | SQLite 数据库路径；相对路径按 `backend/` 解析 | data/eino-oa.db |
| `DEEPSEEK_API_KEY` | DeepSeek API 密钥（留空则关闭 AI） | 空 |
| `DEEPSEEK_MODEL` | 模型名称 | deepseek-chat |
| `DEEPSEEK_BASE_URL` | API 基础地址 | https://api.deepseek.com/v1 |
| `ADMIN_USERNAME` | 管理员用户名，每次启动同步 | admin |
| `ADMIN_PASSWORD` | 管理员密码，**必填**，每次启动同步 | 无 |
| `SESSION_TTL_HOURS` | 浏览器登录有效期（小时） | 168 |

CLI 相关环境变量：`EINO_OA_SERVER`（默认服务地址）、`EINO_OA_CREDENTIALS`（凭证文件路径）。

## 项目结构

```
eino-oa/
  backend/
    .env              配置文件（不提交）
    .env.example      配置模板
    cmd/
      server/          主服务入口（HTTP API + AI Agent + OAuth2）
      eino-oa-cli/     CLI 授权登录工具
    internal/
      auth/            密码、会话、令牌、PKCE、中间件
      model/           数据模型（含 User / AuthSession / APIToken / OAuthCode）
      repository/      数据库操作层
      service/         业务逻辑层
      handler/         HTTP Handler（auth / oauth / 业务）
      agent/           Eino Agent + Tools
      report/          ChatTemplate + Chain 学业报告
    config/            配置管理
  frontend/
    src/
      auth/            登录态 Context
      pages/           业务页面（含登录、访问令牌）
      services/        API 封装
      components/      通用组件
  skills/
    student-manage/    学生管理技能（REST API）
    grade-analysis/    成绩分析技能（REST API）
```

## 生产部署要点

- 启用 HTTPS；反向代理需传递 `X-Forwarded-Proto`，后端会据此设置 `Secure` Cookie。
- 前端、`/api`、`/oauth`、`/skills` 必须同源，否则登录 Cookie 和 OAuth 回调无法工作。
- 不要信任公网来的 `X-Forwarded-For`；后端已关闭 Gin 的代理信任，登录限流按直连 IP 计数。
- 定期在「访问令牌」页面清理不再使用的 PAT；修改 `ADMIN_PASSWORD` 并重启后，旧会话和旧令牌会自动失效。

## AI 助手用法示例

- "查一下张三的成绩"
- "分析一下数据结构的成绩分布"
- "哪些学生的 GPA 低于 2.0"
- "帮我录入一个新学生，学号 2024010，姓名刘明"
- "生成王五的综合成绩报告"
