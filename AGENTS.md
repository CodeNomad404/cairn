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
cmd/server/main.go        进程入口 — 启动 HTTP Server、优雅关闭
cmd/seed/main.go          开发数据初始化
internal/
  app/                    应用组装（Composition Root）— 创建数据库、Repository、Logic、Handler 和 chi Router
  auth/                   认证模块 — handler/logic/repository/middleware
  blog/                   博客模块 — handler/logic/repository
  tool/                   工具模块 — handler/logic
  config/                 viper 读取 config.yaml + 环境变量
  util/                   通用请求适配器、参数校验调用和统一 JSON API 响应
  web/                    HTTP 通用基础设施：安全头、限流、代理 IP、站点 URL
  seo/                    robots.txt 和 sitemap.xml
web/src/                  Vue 3 + TypeScript 前端
migrations/               golang-migrate SQL 迁移脚本
docker-compose.yml        本地开发环境
docs/                     项目文档
```

### 后端分层与调用链

后端是按业务模块组织的模块化单体，不是微服务。JSON API 的标准调用链为：

```text
HTTP 请求
  → chi Router
  → 认证/授权 Middleware
  → util 请求适配器（JSON Body、Query、Path）
  → Handler
  → Logic（启动时组装一次，方法显式接收 ctx）
  → Repository
  → PostgreSQL
  → util.Reply
```

各层职责：

- **Router**：声明路径、HTTP 方法和中间件顺序。需登录或角色的接口在路由层使用 `RequireUser` / `RequireRole`，使鉴权先于请求体解析执行。
- **util/request.go**：负责所有 HTTP 输入的机械处理，包括 Body、Query 和 Path 参数的读取、默认值、类型转换、大小限制、未知字段检查、调用 DTO 的 `Validate()`，以及解析失败时写入统一错误响应。成功后通过类型安全的回调进入 Handler。该文件只提供通用解析能力，不定义具体业务接口的 DTO，也不包含业务规则。
- **Handler**：只允许定义请求 DTO（含 `Validate()`）和 Handler 接口方法。Handler 方法获取 util 层已经解析并校验的类型化参数，只负责取得认证上下文、调用 Logic 和调用 `util.Reply`；不得定义 JSON、Query、Path、分页或类型转换等解析辅助函数，不直接访问数据库，也不接收数据库模型后再组装响应。请求 DTO 及其 `Validate()` 必须与对应接口方法放在同一个 Handler 文件内，并紧挨该接口方法，方便阅读接口时立即看到参数契约。
- **Logic**：负责业务规则、业务编排以及最终输出数据的组装。Repository 返回的数据库模型只能在 Logic 内部使用；Logic 必须转换成可直接交给 `util.Reply` 的 `XxxResult`。Logic 在应用启动时组装一次，构造函数只接收其实际使用的 Repository 接口；不保存 `context.Context`，需要 Context 的方法将 `ctx context.Context` 作为第一个参数并传给 Repository。面向 JSON API Handler 的业务方法统一返回 `(data any, code util.Code, msg string)`。
- **Repository**：封装 PostgreSQL 读写，接收 `context.Context`，不处理 HTTP 响应。
- **app（Composition Root）**：应用启动时创建 Repository、Logic、Handler 并完成组装；Handler 与 Middleware 接收已经创建好的 Logic，请求过程中不创建 Logic。初始化后只读的组件（Goldmark、Bluemonday Policy）只在启动时初始化一次，可安全并发复用。
- **util/response.go**：`util.Reply(ctx, w, data, code, msg)` 负责业务码到 HTTP 状态的映射和 JSON 信封格式。Handler 不再自行拼装 JSON 错误响应。
- **web**：仅保留与 HTTP/站点相关的通用基础设施，不放置 JSON 请求解析、参数校验或 API 响应函数。

### JSON API 开发约定

1. JSON Body 的 Request DTO 放在对应的 Handler 文件中，实现 `Validate() error`；不要放入 `logic` 包，也不要集中到全局 `dto`、`common` 或 `util` 包。
2. 新增接口时，按照“Request DTO → `Validate()` → Handler 方法”的顺序紧挨编写，参数定义与使用它的接口不得分散到不同文件。
3. JSON Body、Query 和 Path 参数统一由 `internal/util/request.go` 中的通用类型化适配器解析。路由把解析成功后的 Handler 作为回调传入；解析或校验失败时适配器直接响应，不进入 Handler。
4. 有 JSON Body 的路由使用 `util.ParseBody[T](maxBytes, h.callback)`，不在 Handler 方法中重复编写 Decode、Validate、Reply 和 `return`。Query、Path 参数分别使用 `util.ParsePagination`、`util.ParsePath`、`util.ParsePositiveInt64Path` 等同类适配器。
5. Handler 文件中不得出现 `ParseXxx`、`strconv`、直接读取 `r.URL.Query()` 或 `chi.URLParam()` 等请求解析代码。类似 `ParseListPosts` 的函数属于 request 层职责，不得放在 Handler。
6. Handler DTO 的 `Validate()` 只校验 HTTP 输入契约，例如必填、长度、格式和取值范围；权限、数据是否存在、唯一性和状态转换等业务规则仍由 Logic 校验。
7. 鉴权放在 chi 路由中间件中，不在每个 Handler 里重复检查 `user == nil` 或角色。
8. API 输出结构在 Logic 中命名为 `XxxResult`。Logic 负责从数据库模型挑选字段并组装完整 Result；禁止把 Repository 返回的模型直接交给 Handler，禁止在 Handler 中使用 `data.(*Model)` 后再次加工。
9. Handler 调用 Logic 时使用 `data, code, msg := l.Method(...)`，并立即调用 `util.Reply(r.Context(), w, data, code, msg)`；Cookie 等必须由 HTTP 层处理的副作用除外。
10. `healthz`、`robots.txt`、`sitemap.xml` 不是 JSON 业务 API，可按各自协议直接输出原始 JSON、文本或 XML，不强制使用 `util.Reply`。
11. 新模块遵循相同的 `handler/logic/repository` 结构；只在真正需要持久化时增加 Repository。

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
