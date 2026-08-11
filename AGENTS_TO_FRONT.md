# AGENTS_TO_FRONT.md

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

## 项目与技术栈

Cairn（`cairn.cn`）是个人博客、作品集和工具站。前端位于 `web/`，使用 Vue 3、TypeScript、Vue Router、Pinia、Vite 和原生 `fetch`，通过相对地址 `/api/v1/...` 调用 Go 后端。

除非明确要求，不引入 Axios、UI 组件库、服务端数据缓存库、运行时类型库、微前端或自动生成 API Client。

## 目录结构

```text
web/src/
├── main.ts                    # 启动入口
├── assets/                    # Vite 构建资源
├── app/                       # 应用装配
│   ├── App.vue
│   ├── router/                # 路由、Layout、守卫
│   ├── layouts/               # 应用级骨架
│   └── components/            # 应用级组件
├── modules/                   # 业务模块
│   └── <module>/
│       ├── index.ts           # 模块对外入口
│       ├── api/               # 后端接口
│       ├── views/             # 路由页面
│       ├── components/        # 业务组件
│       ├── composables/       # 可复用状态逻辑
│       ├── store.ts           # 可选：跨页面状态
│       └── types.ts           # Request、Result、共享类型
└── shared/                    # 无业务含义的底层能力
    ├── api/
    ├── styles/
    └── <capability>/          # 按明确能力命名
```

目录按需创建，不机械补齐。`web/public/` 保存固定公开 URL 资源，`src/assets/` 保存经 Vite 导入构建的资源。

## 依赖与职责

依赖方向：

```text
main → app → modules → shared
```

- `app`：Router、Layout、全局初始化和应用级组件，负责组装业务模块。
- `modules`：业务内聚代码；不得导入 `app`。
- `shared`：不理解 Cairn 业务；不得导入 `app` 或 `modules`。
- 业务模块之间可单向依赖，但不得形成循环依赖。
- 模块外部代码（包括 `app` 和其他模块）只能从目标模块根 `index.ts` 导入。
- 禁止外部直接依赖模块的 `api`、`views`、`components`、`composables`、`store.ts`、`types.ts` 或其他内部文件。
- 模块内部直接导入自身文件，不得从自己的根 `index.ts` 导入。
- `index.ts` 必须显式导出，禁止 `export *`。
- `app/router` 直接动态导入模块 View 是唯一例外，用于保留路由懒加载。
- Layout 由 `app/router` 通过嵌套路由装配，模块 View 不得包裹或导入 Layout。
- 归属看语义，不看引用次数。业务代码优先留在模块；禁止用 `common/helpers/misc` 堆放杂项。
- 除模块根 `index.ts` 外，不创建汇总式 barrel 文件。

页面数据流：

```text
Router → View → Component
           ├→ Module API
           └→ Composable → Module API

Router / View → Store → Module API
Module API → Shared HTTP → Go API
```

- View：路由输入、页面装配、页面状态和导航。
- 简单 View 可直接调用同模块 Module API；不为保持 View 空壳而机械创建 Page Composable。
- Component：界面与局部交互；需要独立异步状态时使用同模块 Composable，不直接调用 Module API 或 Shared HTTP。
- Composable：异步 `data/loading/error`、生命周期、请求竞态、共享响应式状态或复用逻辑。
- Store：仅保存跨页面或跨生命周期共享状态，不作为默认页面缓存。
- Utility：无 Vue 生命周期依赖的纯函数。
- `types.ts`：类型契约，不含运行时逻辑。
- Module API：路径、方法、参数、返回类型；不操作 Router、Store 或 UI。
- Shared HTTP：通用传输和响应解析，不含业务行为。

View 可保留少量模板专用 `ref`、`computed`、导航和事件转发。API、类型、Store 和纯函数不得统一塞入 `composables`。

页面专用 `computed` 可以留在 View；不要为一行派生状态创建 Composable。SFC 建议顺序：imports → props/router/store → 源状态 → computed → composable → handlers → watch/lifecycle → template → style。

## 组件与状态

- 统一使用 Composition API 和 `<script setup lang="ts">`。
- View 是路由页面；Layout 是应用骨架；`app/components` 可理解整个应用；业务组件属于所在模块。
- Shared 组件不导入 Router、业务 Store 或业务类型，只通过 Props、Events、Slots 和 `v-model` 协作。
- 状态必须只有一个可信来源；禁止修改普通 Prop。
- `v-model` 子组件只因直接用户输入更新模型；预填、重置和外部状态切换由状态持有者负责。
- 禁止用模板 Ref/`defineExpose` 操作普通表单或业务状态；仅命令式 UI 能力可例外。
- `provide/inject` 仅用于深层组件树的稳定上下文，使用强类型 `InjectionKey`，不替代普通父子通信。
- 提交成功、取消、切换目标时必须明确清空或保留输入。
- 页面输入、按钮 submitting、局部错误不得放全局 Store。
- 只按独立职责、稳定接口、递归、实际复用或测试边界拆组件；不按行数或假设的未来复用拆分。
- Composable 使用 `useXxx` 命名，返回明确状态和操作，并在失效或卸载时清理副作用。
- 不依赖 Vue 的逻辑保持为纯函数；只在复杂、需测试或真实复用时提取文件。

## API 与类型

- 以真实 Go Handler、Logic Result 和 `httpx.Reply` 为准，并同步核对 OpenAPI、Wiki、Module API、TypeScript 类型。
- API Body 使用 `XxxRequest`，成功 `data` 使用 `XxxResult`；模块公共契约放 `types.ts`，Props 和内部辅助类型就近定义。
- 禁止显式 `any`、`@ts-ignore`、双重断言和无检查的宽泛 `as`；外部值用 `unknown` 并收窄。
- 优先类型守卫和 `satisfies`；断言只留在已检查边界、`as const` 或 Shared HTTP 单一传输边界。
- `request<T>` 只有编译期约束，不代表运行时校验；Shared HTTP 必须先验证通用响应信封。
- 区分 `T | null`、可选字段和未加载状态；空集合用 `[]`。
- API 日期保持 ISO 8601 `string`，JSON ID 用 `number`，固定枚举值用字符串联合，不用 TypeScript `enum`。
- 不建立内容相同的 DTO/Entity/Model/ViewModel；泛型只表达真实类型关系。
- 纯类型导入必须使用 `import type`。
- Path 使用 `encodeURIComponent`，Query 使用 `URLSearchParams`，JSON 由 Shared HTTP 序列化。
- 本项目评论列表是顶层数组加可选嵌套 `replies`，禁止再次按扁平列表组树并覆盖回复。

## 异步与错误

- 每个资源独立维护 data、loading、error；文章、评论、列表和写操作不得共用模糊 loading。
- loading 时不显示“暂无数据”；已有数据刷新或加载更多失败也必须显示错误。
- 写操作成功但刷新失败属于部分成功，需明确提示。
- Composable 若吞掉错误，必须返回调用方可判断的状态；不能让调用方误判成功。

路由参数驱动的请求必须：

1. 隔离或清理旧数据。
2. 用 AbortController 取消同资源旧请求。
3. controller 捕获为本次调用的局部变量。
4. 写入 data/error/loading 前校验 controller 身份和当前参数。
5. 卸载时取消请求。
6. 写操作完成后仅在参数仍为当前值时刷新，禁止把旧页面数据写入新页面。

核心模式：

```ts
const controller = new AbortController()
currentController = controller
const result = await fetchData(key, controller.signal)
if (currentController !== controller || key !== currentKey()) return
data.value = result
// catch/finally 同样先校验 currentController === controller
```

## 认证与路由

认证区分：未初始化、已认证、确认未登录、服务不可用。

- 只有 `/auth/me` 返回 200 或 401 才标记初始化完成；网络/500 保持未初始化并可重试。
- 并发初始化复用同一个 Promise。
- 公开路由后台初始化，不得等待并阻塞导航。
- `guestOnly/requiresAuth/requiresAdmin` 才等待身份；`requiresAdmin` 隐含 `requiresAuth`。
- 确认未登录才跳登录页；非管理员拒绝管理员页面。
- 服务不可用时 guest-only 可放行，受保护路由必须取消或展示无法验证身份，不能直接放行。
- 前端守卫只改善体验，Go 后端始终最终鉴权。
- 登录 redirect 只接受合法站内路径。

## 样式

- `shared/styles`：Reset、无障碍和通用工具。
- `app/styles`：Cairn Token、全局排版、Layout。
- `modules/*`：业务样式，优先组件 scoped style。

## 验收

完成前至少执行：

```text
npm run type-check
npm run build
git diff --check
```

有相关测试时必须运行。架构变更还要检查 `modules` 不导入 `app`，`shared` 不导入上层，并删除本次产生的孤儿代码。

类型检查和构建不能证明运行时正确；按改动验证参数快速切换、请求取消/乱序、认证 200/401/网络/500、各类路由、空数据与刷新失败、表单完整周期、评论嵌套回复。
