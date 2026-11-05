# 前端

React 19 + TypeScript + Ant Design 6 + Vite。包含学生 / 课程 / 成绩管理、AI 对话、登录页和访问令牌管理页。

## 环境要求

- Node.js 20.19+ 或 22.12+
- npm 10+
- 后端已启动（端口读取 `backend/.env` 的 `SERVER_PORT`）

## 启动开发服务器

```bash
cd frontend
npm install
npm run dev
```

访问 <http://localhost:3000>，使用 `backend/.env` 里的 `ADMIN_USERNAME` / `ADMIN_PASSWORD` 登录。

Vite 会把以下路径代理到后端：

| 前端路径 | 用途 |
| --- | --- |
| `/api` | 业务 API、登录、令牌管理 |
| `/oauth` | OAuth2 授权页与 token 端点 |
| `/.well-known` | OAuth2 元数据 |
| `/skills` | 技能文档下载 |

Vite 启动时会读取 `backend/.env` 的 `SERVER_PORT` 作为代理目标，默认 `8080`。需要临时指向别的后端时：

```bash
VITE_BACKEND_URL=http://localhost:8081 npm run dev
```

代理必须存在，否则浏览器无法携带登录 Cookie，OAuth 授权页也无法同源回调。

## 构建

```bash
cd frontend
npm run build
```

产物在 `frontend/dist`。后端启动时如果检测到该目录，会直接托管静态文件，从而保证页面、`/api`、`/oauth` 同源。生产环境推荐这种方式，或在 Nginx 中做同样的同源转发。

`npm run preview` 只用于查看静态构建，不会自动代理 `/api`，联调请用 `npm run dev` 或后端托管 `dist`。

## 常用脚本

| 命令 | 说明 |
| --- | --- |
| `npm run dev` | 启动 Vite 开发服务器（3000） |
| `npm run build` | TypeScript 检查并构建到 `dist/` |
| `npm run lint` | ESLint 检查 |
| `npm run preview` | 本地预览构建产物（不含 API 代理） |

## 目录结构

```text
src/
  auth/            登录态 Context、会话恢复
  components/      布局、导航等通用组件
  pages/
    login/         登录页
    students/      学生管理
    courses/       课程管理
    grades/        成绩管理、统计、预警
    chat/          AI 助手与历史会话
    tokens/        PAT / OAuth 令牌管理
  services/        axios 封装与 API 类型
  App.tsx          路由与登录守卫
  main.tsx         入口
```

## 登录与令牌

- 登录成功后后端设置 HttpOnly Cookie `eo_session`，前端不保存密码。
- 未登录访问业务页面会跳转到 `/login`。
- 「访问令牌」页可以创建只读 / 读写 PAT，或吊销所有 CLI / OAuth 令牌。
- 修改 `backend/.env` 的管理员密码并重启后，旧会话会失效，需要重新登录。

## 常见问题

**页面能打开，但接口 401**

先确认后端已启动、`backend/.env` 的密码与登录输入一致。浏览器开发者工具中查看 `/api/auth/me` 是否返回 401。

**接口 404 或跨域**

确认是通过 `http://localhost:3000` 访问，而不是直接打开 `dist/index.html`；Vite 代理只在开发服务器下生效。

**修改后端端口后前端仍请求旧端口**

Vite 会读取 `backend/.env` 的 `SERVER_PORT`；如果后端是用环境变量临时换的端口，用 `VITE_BACKEND_URL=http://localhost:新端口 npm run dev` 启动。
