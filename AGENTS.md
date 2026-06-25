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
- **当前状态**：Phase 1 骨架完成 — `/healthz` 能跑，数据库能连，热重载正常。下一步 Phase 2（博客 MVP）。

## 常用命令

```bash
# 启动开发（热重载）
DB_PASSWORD=cairn_dev air

# 数据库
docker compose up -d        # 启动 PostgreSQL
docker compose down          # 停止
docker compose down -v       # 清空数据

# 构建与检查
go build -o bin/cairn ./cmd/server   # 手动编译
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
  config/                 viper 读 config.yaml + 环境变量 → Config 结构体
  handler/                HTTP 处理函数（空，Phase 2 填）
  service/                业务逻辑（空，Phase 2 填）
  repository/             数据库操作（空，Phase 2 填）
web/static/css/input.css  Tailwind CSS 入口（@tailwind base/components/utilities）
web/templates/            Go html/template 模板（空，Phase 2 填）
migrations/               golang-migrate SQL 迁移脚本（空，Phase 2 填）
docs/                     项目文档 — 决策记录、开发日志、路线图
```

**已确定的技术栈**：`chi` 路由、`pgxpool`（直接连 PostgreSQL，没用 `database/sql` 包装层）、`slog`（JSON 格式输出到 stdout）、`viper`（读 config.yaml）、`html/template`（服务端渲染）、Tailwind CSS、HTMX + Alpine.js（Phase 3 开始用）。

**明确不用**：GORM、Gin/Beego/Echo、React/Vue/Next.js、Kubernetes、消息队列、Elasticsearch。完整技术栈和理由见 `docs/blog-project-discussion-2026-05-19.md` 第三节。

## 已做的关键决策

| 决策 | 结论 | 记录位置 |
|---|---|---|
| Redis | 延后，需要时再加 | diff-05-31 第三节 |
| COS 对象存储 | 延后，先用本地目录存文件 | diff-05-31 第四节 |
| Dockerfile / Caddyfile | 延后到 Phase 7 部署时再写 | diff-05-31 第五节 |
| CI/CD（GitHub Actions + TCR）| 延后，目前直接 ssh → go build → 重启 | diff-05-31 第六节 |
| 本地 Go 运行方式 | 裸机直接跑（不用 Docker），air 才能检测文件变化 | diff-05-31 第二节 |

## 技术总结
当我每学过一个新知识新技术，需要总结成博客文档，放到网站上

## 技术实践
写代码之前，要查最佳实践，确保拥抱新趋势新技术