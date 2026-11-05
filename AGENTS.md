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
  app/                    应用组装 — 创建数据库、Repository、Logic、Handler 和 chi Router
  auth/                   认证模块 — handler/logic/repository/middleware
  blog/                   博客模块 — handler/logic/repository
  tool/                   工具模块 — handler/logic
  config/                 viper 读取 config.yaml + 环境变量
  httpx/                  JSON Body、Query、Path 解析和统一 JSON API 响应
  middleware/             HTTP 通用基础设施：安全头、限流和代理 IP
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
  → Handler
  → httpx 读取并校验 JSON Body、Query 或 Path
  → Logic 方法(ctx, ...)
  → Repository
  → PostgreSQL
  → httpx.Reply
```

各层职责：

- **Router**：声明路径、HTTP 方法和中间件顺序。需登录或角色的接口在路由层使用 `RequireUser` / `RequireRole`，使鉴权先于请求体解析执行。
- **httpx/request.go**：负责 HTTP 输入的通用机械处理。`ParseBody` 限制 Body 大小、拒绝未知字段和尾随数据，并调用请求 DTO 的 `Validate()`；`ParsePagination`、`ParsePath` 和 `ParsePositiveInt64Path` 负责通用 Query 和 Path 参数解析。这些函数只返回解析结果或错误，不写入 HTTP 响应，不定义具体业务 DTO，也不包含业务规则。
- **Handler**：定义请求 DTO（含 `Validate()`）和 HTTP 接口方法。Handler 只负责调用 `httpx` 解析请求、取得认证上下文、调用 Logic 和调用 `httpx.Reply`；解析失败时使用 `httpx.CodeBadRequest` 立即响应。Handler 不得定义 JSON、Query、Path、分页或类型转换等通用解析辅助函数，不直接访问数据库，也不接收数据库模型后再组装响应。请求 DTO 及其 `Validate()` 必须与对应接口方法放在同一个 Handler 文件内，并紧挨该接口方法。
- **Logic**：负责业务规则、业务编排以及最终输出数据的组装。Logic 在应用组装时创建，构造函数接收所需的 Repository 或其他长期依赖；每个业务方法通过第一个参数接收 `context.Context`。Repository 返回的数据库模型只能在 Logic 内部使用，Logic 转换成可直接交给 `httpx.Reply` 的 `XxxResult`。面向 JSON API Handler 的查询或创建方法统一返回 `(data any, code httpx.Code, err error)`；无响应数据的修改方法可返回 `(code httpx.Code, err error)`。
- **Repository**：封装 PostgreSQL 读写，接收 `context.Context`，不处理 HTTP 响应。
- **app**：作为组合根，在应用启动时统一创建数据库连接、Repository、Logic 和 Handler，再注册 chi Router；不在业务代码中重复创建长期依赖。
- **httpx/response.go**：`httpx.Reply(ctx, w, data, code, err)` 负责业务码到 HTTP 状态的映射和 JSON 信封格式。非 `AppError` 的内部错误不会原样暴露给客户端。
- **middleware**：仅保留认证、授权、安全头、限流和代理 IP 等 HTTP 通用基础设施，不放置 JSON 请求解析、参数校验或 API 响应函数。

### JSON API 开发约定

1. JSON Body 的 Request DTO 放在对应的 Handler 文件中，实现 `Validate() error`；不要放入 `logic` 包，也不要集中到全局 `dto`、`common` 或 `util` 包。
2. 新增接口时，按照“Request DTO → `Validate()` → Handler 方法”的顺序紧挨编写，参数定义与使用它的接口不得分散到不同文件。
3. JSON Body、Query 和 Path 参数统一由 `internal/httpx/request.go` 中的通用类型化解析函数处理。解析或校验失败时 Handler 立即调用 `httpx.Reply` 响应，不进入 Logic。
4. 有 JSON Body 的 Handler 使用 `httpx.ParseBody[T](w, r, maxBytes)`。Query、Path 参数分别使用 `httpx.ParsePagination`、`httpx.ParsePath`、`httpx.ParsePositiveInt64Path` 等通用函数，不重复编写 Decode、默认值、类型转换和大小限制。
5. Handler 文件中不得出现 `ParseXxx`、`strconv`、直接读取 `r.URL.Query()` 或 `chi.URLParam()` 等请求解析代码。类似 `ParseListPosts` 的函数属于 request 层职责，不得放在 Handler。
6. Handler DTO 的 `Validate()` 只校验 HTTP 输入契约，例如必填、长度、格式和取值范围；权限、数据是否存在、唯一性和状态转换等业务规则仍由 Logic 校验。
7. 鉴权放在 chi 路由中间件中，不在每个 Handler 里重复检查 `user == nil` 或角色。
8. API 输出结构在 Logic 中命名为 `XxxResult`。Logic 负责从数据库模型挑选字段并组装完整 Result；禁止把 Repository 返回的模型直接交给 Handler，禁止在 Handler 中使用 `data.(*Model)` 后再次加工。
9. Handler 调用 Logic 时使用 `data, code, err := l.Method(r.Context(), ...)`，并立即调用 `httpx.Reply(r.Context(), w, data, code, err)`；无响应数据的方法使用 `code, err := l.Method(r.Context(), ...)`。Cookie 等必须由 HTTP 层处理的副作用除外。
10. `healthz`、`robots.txt`、`sitemap.xml` 不是 JSON 业务 API，可按各自协议直接输出原始 JSON、文本或 XML，不强制使用 `httpx.Reply`。
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

## API 接口

- 新增 JSON 业务接口统一使用 `/api/v1/{业务模块}/{资源}/{动作}` 路径结构，例如 `POST /api/v1/blog/comment/create`。
- 模块名、资源名和动作名使用小写英文；多个单词使用短横线连接；资源名使用单数形式。
- 优先使用 `create`、`get`、`list`、`update`、`delete`、`enable`、`disable`、`pause`、`resume`、`preview`、`test` 等统一动作名。
- 纯查询接口使用 `GET`；创建、修改、删除数据或触发外部行为的接口统一使用 `POST`。新接口不使用 `PUT`、`PATCH` 或 `DELETE`。
- `GET` 参数放在 Query 中；`POST` 参数放在 JSON Body 中；新接口不把资源 ID 嵌入路径。
- 新业务不得新增 REST 风格路径。已有接口保持兼容，只在明确的接口迁移任务中按业务模块逐步修改，不得在无关改动中顺手迁移。
- 接口变更遵循契约先行：先更新 `api/openapi.yaml`，并在同一变更中同步后端路由、前端调用和接口文档。
- 完整规则和迁移示例见 `docs/API接口命名规范.md`。
