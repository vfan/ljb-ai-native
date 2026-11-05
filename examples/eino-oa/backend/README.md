# 后端服务

Go + Gin + GORM + SQLite，提供学生 / 课程 / 成绩 API、AI 助手，以及登录会话、PAT、OAuth2 三种鉴权方式。

## 环境要求

- Go 1.25+
- C 编译器（SQLite 驱动 `mattn/go-sqlite3` 需要 CGO）
  - macOS：`xcode-select --install`
  - Linux：安装 `gcc` 或 `build-essential`
- 可选：DeepSeek API Key（不填则关闭 AI 助手，其余功能正常）

## 配置文件

配置固定放在 **`backend/.env`**，与 `backend/go.mod` 同级：

```text
backend/
  .env          ← 实际读取
  .env.example  ← 模板
  go.mod
```

初始化配置：

```bash
cd backend
cp .env.example .env
```

编辑 `backend/.env`，最少需要设置管理员密码：

```dotenv
SERVER_PORT=8080
DB_PATH=data/eino-oa.db

ADMIN_USERNAME=admin
ADMIN_PASSWORD=change-me-123

DEEPSEEK_API_KEY=sk-...
DEEPSEEK_MODEL=deepseek-chat
DEEPSEEK_BASE_URL=https://api.deepseek.com/v1
```

说明：

- 只读取 `backend/.env`；仓库根目录的 `.env` 不会加载。
- 进程环境变量优先于 `.env`，例如 `ADMIN_PASSWORD=xxx go run ./cmd/server`。
- `ADMIN_PASSWORD` 必填，未设置时启动直接失败。
- 每次启动都会把 `ADMIN_USERNAME` / `ADMIN_PASSWORD` 同步到数据库：
  - 账号不存在则创建；
  - 密码变化则更新，并注销该账号的旧会话和旧令牌；
  - 密码没变则保留现有会话。
- 修改密码 = 修改 `backend/.env` 后重启，没有单独的改密接口。

`DB_PATH` 为相对路径时，统一按 `backend/` 目录解析。因此从仓库根目录运行编译好的二进制，也会使用 `backend/data/eino-oa.db`，不会在根目录另建一份数据库。

## 启动

```bash
cd backend
go mod tidy
go run ./cmd/server
```

也可以直接运行入口文件：

```bash
go run cmd/server/main.go
```

启动成功会看到：

```text
已加载配置文件: .env
管理员账号 "admin" 已就绪
服务启动在 :8080
```

## 编译与部署

```bash
cd backend
go build -o eino-oa-server ./cmd/server
```

在 `backend/` 运行：

```bash
./eino-oa-server
```

在仓库根目录运行：

```bash
./backend/eino-oa-server
```

两种方式都会读取 `backend/.env`，并使用 `backend/data/eino-oa.db`。生产环境建议由反向代理统一提供 HTTPS，并保证前端、`/api`、`/oauth`、`/.well-known`、`/skills` 同源。

## CLI 授权工具

```bash
cd backend
go build -o eino-oa-cli ./cmd/eino-oa-cli

./eino-oa-cli login --server http://localhost:8080
./eino-oa-cli token
```

凭证默认保存在 `~/.eino-oa/credentials.json`，可用 `EINO_OA_CREDENTIALS` 覆盖。

## 测试与检查

```bash
cd backend
go test ./...
go vet ./...
```

## 常见问题

**启动报 `ADMIN_PASSWORD 未设置`**

在 `backend/.env` 中设置 `ADMIN_PASSWORD`，重启即可。不需要删库或重建账号。

**改了根目录 `.env` 没生效**

请改 `backend/.env`，启动日志会显示实际加载的文件。

**端口被占用**

改 `backend/.env` 的 `SERVER_PORT`，或临时覆盖：

```bash
SERVER_PORT=8081 go run ./cmd/server
```

**数据库在哪里**

默认 `backend/data/eino-oa.db`。备份时复制该文件即可；删除它会丢失学生、成绩、会话和令牌数据。
