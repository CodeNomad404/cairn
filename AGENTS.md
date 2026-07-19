# AGENTS.md

Behavioral guidelines to reduce common LLM coding mistakes. Merge with project-specific instructions as needed.

**Tradeoff:** These guidelines bias toward caution over speed. For trivial tasks, use judgment.

## 1. Think Before Coding

**Don't assume. Don't hide confusion. Surface tradeoffs.**

Before implementing:
- State your assumptions explicitly. If uncertain, ask.
- If multiple interpretations exist, present them - don't pick silently.
- If a simpler approach exists, say so. Push back when warranted.
- If something is unclear, stop. Name what's confusing. Ask.

## 2. Simplicity First

**Minimum code that solves the problem. Nothing speculative.**

- No features beyond what was asked.
- No abstractions for single-use code.
- No "flexibility" or "configurability" that wasn't requested.
- No error handling for impossible scenarios.
- If you write 200 lines and it could be 50, rewrite it.

Ask yourself: "Would a senior engineer say this is overcomplicated?" If yes, simplify.

## 3. Surgical Changes

**Touch only what you must. Clean up only your own mess.**

When editing existing code:
- Don't "improve" adjacent code, comments, or formatting.
- Don't refactor things that aren't broken.
- Match existing style, even if you'd do it differently.
- If you notice unrelated dead code, mention it - don't delete it.

When your changes create orphans:
- Remove imports/variables/functions that YOUR changes made unused.
- Don't remove pre-existing dead code unless asked.

The test: Every changed line should trace directly to the user's request.

## 4. Goal-Driven Execution

**Define success criteria. Loop until verified.**

Transform tasks into verifiable goals:
- "Add validation" → "Write tests for invalid inputs, then make them pass"
- "Fix the bug" → "Write a test that reproduces it, then make it pass"
- "Refactor X" → "Ensure tests pass before and after"

For multi-step tasks, state a brief plan:
```
1. [Step] → verify: [check]
2. [Step] → verify: [check]
3. [Step] → verify: [check]
```

Strong success criteria let you loop independently. Weak criteria ("make it work") require constant clarification.

---

**These guidelines are working if:** fewer unnecessary changes in diffs, fewer rewrites due to overcomplication, and clarifying questions come before implementation rather than after mistakes.
## 项目概况

**Cairn**（`cairn.cn`）— 个人博客 + 作品集 + 工具站。名字来自山野里用石头堆的路标（trail cairns）。

- **语言**：Go 1.26
- **服务器**：腾讯云轻量 2C2G，上海，Ubuntu 22.04，Docker CE
- **当前状态**：后端已改为 `/api/v1` JSON API，前端采用 Vue 3；本地开发环境统一使用 Docker Compose，生产部署等待前端适配完成后再设计。

## 常用命令

```bash
# 启动完整开发环境（前后端热重载）
docker compose up --build

# 停止或清空本地环境
docker compose down
docker compose down -v       # 同时清空本地数据库

# 构建与检查
docker compose build
golangci-lint run                    # 代码检查
go mod tidy                          # 整理依赖

# 验证
curl http://localhost:8080/healthz   # 检查服务是否正常
```

Taskfile 快捷命令：`task dev`、`task build`、`task lint`、`task db:up`、`task db:down`、`task db:migrate`、`task tidy`。

## 项目架构

```
cmd/server/main.go        入口 — 组装配置、数据库、路由、优雅关闭
internal/
  auth/                   认证模块 — handler/service/repository
  blog/                   博客模块 — handler/service/repository
  tool/                   工具模块 — handler/service
  config/                 viper 读 config.yaml + 环境变量
web/src/                  Vue 3 + TypeScript 前端
migrations/               golang-migrate SQL 迁移脚本
docker-compose.yml        本地开发环境
docs/                     项目文档
```

**已确定的技术栈**：Go 1.26、`chi`、`pgxpool`、PostgreSQL、`slog`、`viper`、Vue 3、TypeScript、Vue Router、Pinia、Vite、原生 `fetch`、Docker Compose、Caddy。

**明确不用**：GORM、Gin/Beego/Echo、React/Next.js、Kubernetes、消息队列、Elasticsearch、Axios、Vuex。

## 已做的关键决策

| 决策 | 结论 | 记录位置 |
|---|---|---|
| Redis | 延后，需要时再加 | diff-05-31 第三节 |
| COS 对象存储 | 延后，先用本地目录存文件 | diff-05-31 第四节 |
| 本地容器化 | 前端、后端、数据库和迁移均使用 Docker Compose | `docker-compose.yml` |
| CI/CD（GitHub Actions + TCR）| 延后，目前直接 ssh → go build → 重启 | diff-05-31 第六节 |
| 本地运行方式 | Docker Compose；Vue 使用 Vite 热更新，Go 使用 Air 热更新 | `docker-compose.yml` |

## 技术总结
当我每学过一个新知识新技术，需要总结成博客文档，放到网站上

## 技术实践
写代码之前，要查最佳实践，确保拥抱新趋势新技术
