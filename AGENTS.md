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

## 项目架构

```
cmd/server/main.go        进程入口 — 启动 HTTP Server、优雅关闭
cmd/seed/main.go          开发数据初始化
internal/
  app/                    应用组装（Composition Root）— 创建数据库、Repository、Logic、Handler 和 chi Router
  auth/                   认证模块 — handler/logic/repository/middleware；model/ 按表拆文件（user.go、session.go）
  blog/                   博客模块 — handler/logic/repository；model/ 按表拆文件（post.go、comment.go）
  tool/                   工具模块 — handler/logic
  config/                 viper 读取 config.yaml + 环境变量
  httpx/                  JSON HTTP 请求解析、业务码、AppError 和统一 JSON API 响应
  middleware/             业务无关的全局 HTTP Middleware：安全头、限流、代理 IP
  seo/                    robots.txt 和 sitemap.xml（canonical 站点 URL 为包内私有常量）
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
  → Handler（方法内显式调用 httpx 请求解析函数）
  → Logic（启动时组装一次，方法显式接收 ctx）
  → Repository
  → PostgreSQL
  → httpx.Reply
```

各层职责：

- **Router**：声明路径、HTTP 方法和中间件顺序。需登录或角色的接口在路由层使用 `RequireUser` / `RequireRole`，使鉴权先于 Handler 内的参数解析执行。
- **httpx/request.go**：负责所有 HTTP 输入的机械处理，包括 Body、Query 和 Path 参数的读取、默认值、类型转换、大小限制、未知字段检查、调用 DTO 的 `Validate()`。提供少量稳定的直接解析函数（`ParseBody[T]`、`ParsePath`、`ParsePositiveInt64Path`、`ParsePagination`），由 Handler 方法显式调用；解析失败返回 `AppError`（存在底层错误时通过 Cause 保留），不写响应。该文件只提供通用解析能力，不定义具体业务接口的 DTO，也不包含业务规则。
- **Handler**：只允许定义请求 DTO（含 `Validate()`）和 Handler 接口方法。Handler 方法按接口所需顺序显式调用 `httpx.ParseBody`、`httpx.ParsePath`、`httpx.ParsePositiveInt64Path`、`httpx.ParsePagination` 完成参数解析，解析失败立即 `httpx.Reply(r.Context(), w, nil, httpx.CodeBadRequest, err)` 返回；成功后只负责取得认证上下文、调用 Logic 和调用 `httpx.Reply`。Handler 中不得直接使用 `chi.URLParam`、`r.URL.Query()`、`strconv`、`json.Decoder`，不得定义解析辅助函数，不直接访问数据库，也不接收数据库模型后再组装响应。请求 DTO 及其 `Validate()` 必须与对应接口方法放在同一个 Handler 文件内，并紧挨该接口方法，方便阅读接口时立即看到参数契约。
- **Logic**：负责业务规则、业务编排以及最终输出数据的组装。Repository 返回的数据库模型只能在 Logic 内部使用；Logic 必须转换成可直接交给 `httpx.Reply` 的 `XxxResult`。Logic 在应用启动时组装一次，构造函数只接收其实际使用的 Repository 接口；不保存 `context.Context`，需要 Context 的方法将 `ctx context.Context` 作为第一个参数并传给 Repository。面向 JSON API Handler 的业务方法返回强类型 Result、`httpx.Code` 和 `error`；无数据操作返回 `httpx.Code` 和 `error`。
- **Repository**：封装 PostgreSQL 读写，接收 `context.Context`，不处理 HTTP 响应。
- **app（Composition Root）**：应用启动时创建 Repository、Logic、Handler 并完成组装；Handler 与 Middleware 接收已经创建好的 Logic，请求过程中不创建 Logic。初始化后只读的组件（Goldmark、Bluemonday Policy）只在启动时初始化一次，可安全并发复用。
- **httpx/response.go**：集中定义业务码、`AppError` 和 `httpx.Reply(ctx, w, data, code, err)`。`AppError` 保存可安全公开的消息和原始 Cause；`Reply` 负责业务码到 HTTP 状态的映射、JSON 信封以及未知错误的 500 兜底。Handler 不自行拼装 JSON 错误响应。
- **middleware**：只保存与具体业务模块无关的全局 HTTP Middleware（`SecurityHeaders`、`SameOrigin`、`TrustedProxyRealIP`、`RateLimit`）。认证 Middleware（`LoadUser`、`GetPrincipal`、`WithPrincipal`、`RequireUser`、`RequireRole`）依赖认证身份模型 `auth.Principal`、会话模型（`auth/model.Session`）和 `AuthLogic`，继续属于 `auth/middleware`。SEO 使用的固定 canonical 站点 URL（`https://cairn.cn`）是 `seo` 包内私有实现，不属于公共包。

### JSON API 开发约定

1. JSON Body 的 Request DTO 放在对应的 Handler 文件中，实现 `Validate() error`；不要放入 `logic` 包，也不要集中到全局 `dto`、`common` 或 `httpx` 包。
2. 新增接口时，按照“Request DTO → `Validate()` → Handler 方法”的顺序紧挨编写，参数定义与使用它的接口不得分散到不同文件。
3. JSON Body、Query 和 Path 参数统一由 `internal/httpx/request.go` 中的 `ParseBody[T]`、`ParsePath`、`ParsePositiveInt64Path`、`ParsePagination` 解析。Handler 方法按接口所需顺序显式调用；解析失败立即 `httpx.Reply(r.Context(), w, nil, httpx.CodeBadRequest, err)` 返回，不进入 Logic。
4. 路由只声明路径、HTTP 方法和鉴权中间件，直接注册标准 Handler 方法（`func (h *XxxHandler) method(w http.ResponseWriter, r *http.Request)`）。不使用返回 `http.HandlerFunc` 的解析适配器；Body 大小限制、未知字段检查、尾随数据检查与 `Validate()` 仍由 `ParseBody` 统一处理。
5. Handler 文件中不得出现 `chi.URLParam`、`r.URL.Query()`、`strconv`、`json.Decoder`、Body 大小限制或分页默认值/范围转换等机械解析代码，这些统一由 `Parse*` 负责。不得新增组合解析函数（如 `ParsePathAndBody`）；类似 `ParseListPosts` 的函数属于 request 层职责，不得放在 Handler。
6. Handler DTO 的 `Validate()` 只校验 HTTP 输入契约，例如必填、长度、格式和取值范围；权限、数据是否存在、唯一性和状态转换等业务规则仍由 Logic 校验。
7. 鉴权放在 chi 路由中间件中，不在每个 Handler 里重复检查 `principal == nil` 或角色。
8. API 输出结构在 Logic 中命名为 `XxxResult`。Logic 负责从数据库模型挑选字段并组装完整 Result；禁止把 Repository 返回的模型直接交给 Handler，禁止在 Handler 中使用 `data.(*Model)` 后再次加工。
9. Handler 调用 Logic 时使用 `data, code, err := l.Method(...)`，并立即调用 `httpx.Reply(r.Context(), w, data, code, err)`；无数据操作只接收 `code, err` 并传入 `nil` data，Cookie 等必须由 HTTP 层处理的副作用除外。
10. `healthz`、`robots.txt`、`sitemap.xml` 不是 JSON 业务 API，可按各自协议直接输出原始 JSON、文本或 XML，不强制使用 `httpx.Reply`。
11. 新模块遵循相同的 `handler/logic/repository` 结构；只在真正需要持久化时增加 Repository。

### 领域模型、认证身份与 API Result 边界

- 持久化模型位于各自模块的 `model` 子包，一张数据库表对应一个生产 model 文件：

```text
internal/auth/model/
  user.go       users 表记录模型
  session.go    sessions 表记录模型
internal/blog/model/
  post.go       posts 表记录模型
  comment.go    comments 表记录模型
```

  model 文件只保存该表对应的数据结构，只依赖标准库，不声明 JSON 标签，不承担 JSON API 契约。该按表拆文件规则只针对数据库 Model；A11 驳回的是"按接口拆分 Handler 文件"，两者不冲突。
- Repository 接口（`UserRepo`、`SessionRepo`、`PostRepo`、`CommentRepo`）留在模块根包，方法参数与返回值使用对应 model 包的类型；model 包不定义接口。
- `auth.Principal`（`ID`、`Username`、`Role`）是认证身份，不属于持久化模型，保留在 `auth` 根包。Request Context 只保存 `Principal`，不保存完整 `User`；`Principal` 只包含认证授权所需的最小字段，不含密码哈希、Session Token 等内部状态，也不承担 API 响应职责。
- JSON 标签只属于 Handler 中的 Request DTO 和 Logic 中定义的 `XxxResult`。Repository 返回的内部模型只能在 Logic 内部使用，Logic 必须转换成可直接交给 `httpx.Reply` 的 `XxxResult`；Handler 不直接响应内部模型或 `Principal`。
- 正常认证调用链通过强类型 `UserResult` 等 Result 收紧，Handler 遵守架构约定，并通过精确响应契约测试（data 恰好是约定的字段集合）防止字段意外增加；最终仍需静态检查和 Code Review 把关。
- Cookie 等 HTTP 副作用仍由 Handler 负责。

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
