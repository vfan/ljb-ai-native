# Eino OA Skills

面向外部 AI Agent 的技能文档。文档本身不含密钥；调用 API 时需要单独提供访问令牌。

## 技能列表

| 技能 | 能力 | 建议权限 |
| --- | --- | --- |
| [student-manage](student-manage/SKILL.md) | 学生、课程的查询与增删改 | `read,write` |
| [grade-analysis](grade-analysis/SKILL.md) | 成绩统计、GPA、学业预警、综合报告 | `read` |

## 安装方式

Agent 可以直接通过 HTTP 下载（后端已把 `skills/` 挂到 `/skills`）：

```bash
curl -sS http://localhost:8080/skills/student-manage/SKILL.md
curl -sS http://localhost:8080/skills/grade-analysis/SKILL.md
```

也可以克隆仓库后读取 `skills/` 目录。

## 配置令牌

任选一种方式：

### A. 个人访问令牌（PAT）

登录网页 →「访问令牌」→「新建令牌」→ 复制令牌：

```bash
export EINO_OA_BASE_URL="http://localhost:8080/api"
export EINO_OA_API_TOKEN="eo_pat_xxx"
```

### B. CLI 授权（OAuth2 + PKCE）

```bash
eino-oa-cli login --server http://localhost:8080
export EINO_OA_API_TOKEN="$(eino-oa-cli token)"
```

## 直接发给 Agent 的接入指令

```text
你是一个 AI Agent。请先下载并完整阅读以下技能文档：
- http://localhost:8080/skills/student-manage/SKILL.md
- http://localhost:8080/skills/grade-analysis/SKILL.md

调用 API 时使用：
export EINO_OA_BASE_URL="http://localhost:8080/api"
export EINO_OA_API_TOKEN="<在这里粘贴令牌>"

所有请求都需要携带：
Authorization: Bearer $EINO_OA_API_TOKEN

令牌等同于我的账号权限，请勿写入公开仓库或日志。
```

> 把 `localhost:8080` 换成实际可访问的地址。若 Agent 运行在另一台机器，
> 需要确保后端监听地址和防火墙允许访问，并优先使用 HTTPS。
