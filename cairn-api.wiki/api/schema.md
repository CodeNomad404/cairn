# API Schema

## HOST

接口以 `HTTP` 协议进行通信

- 本地开发

```
http://localhost:8080
```

- 生产

```
https://cairn.cn
```

- 前端开发环境经 Vite 代理 `/api`，页面内一律使用相对路径 `/api/...`

## 请求方法

JSON 业务接口统一使用 `POST`、`GET`、`DELETE` 三种方法：

| 方法 | 用途 |
|---|---|
| `POST` | 创建资源或触发动作：注册、登录、登出、创建文章、发表评论、JSON 格式化 |
| `GET` | 查询：当前用户信息、文章列表/详情、评论列表 |
| `DELETE` | 删除：删除文章、删除评论 |

不使用 `PUT` / `PATCH`。对不存在的路径或方法使用错误的方法调用属于调用方开发错误，路由层返回 `404`，不在各接口页逐项列出。

## 请求数据格式

- JSON 业务接口请求体为 `Content-Type: application/json` 的 UTF-8 文本；无请求体的接口（`GET`、`DELETE`、`POST /auth/logout`）不带 Body。
- 各接口 Body 大小限制：注册/登录 16 KB，创建文章与 JSON 格式化 1 MB，发表评论 32 KB。
- 以下情况均返回 HTTP `400`（调用方开发错误，不在接口页逐项列出，由前端表单校验与请求封装统一兜底）：
  - 缺少必填参数、参数为空白
  - JSON 损坏（非合法 JSON、尾随数据）
  - 字段类型错误、存在未知字段
  - Query / Path 参数解析失败（如 `page=abc`、非法的 `{id}`）
- 分页参数通用约定：`page` 默认 `1`，`page_size` 默认 `20`、取值范围 1–100。

## 请求响应

JSON 业务接口统一返回 `{code, message?, data?}` 信封，`Content-Type: application/json; charset=utf-8`：

| HTTP 状态 | `code` | 含义 |
|---|---|---|
| `200 OK` | `0` | 成功 |
| `201 Created` | `1` | 创建成功 |
| `204 No Content` | `2` | 成功且无响应体 |
| `400 Bad Request` | `400` | 请求参数不合法 |
| `401 Unauthorized` | `401` | 未登录或会话失效 |
| `403 Forbidden` | `403` | 已登录但权限不足 |
| `404 Not Found` | `404` | 资源不存在（文章/评论/接口） |
| `422 Unprocessable Entity` | `422` | 业务校验失败（如重复用户名、非法父评论） |
| `500 Internal Server Error` | `500` | 服务端内部错误 |

- `data` 在成功响应中携带业务数据，`204` 时无响应体；`message` 仅在失败时出现。
- 数据库异常等内部错误统一返回 `500` 和安全的兜底文案，具体原因只进服务端日志，前端展示统一错误提示。
- 全站限流：`/api/v1` 下每 IP 每分钟最多 10 次请求，超出后返回 HTTP `429`，响应体为纯文本：

```
too many requests
```

- `healthz`、`robots.txt`、`sitemap.xml` 不是 JSON 业务接口，按各自协议输出，见 [api/public](../api/public/)。

## 鉴权方式

认证使用名为 `cairn_session` 的 Session Cookie。登录或注册成功后，服务端通过 `Set-Cookie` 响应头创建会话；前端不读取、不保存 Session Token，浏览器会在后续同源请求中自动携带 Cookie。

Cookie 属性：

| 属性 | 作用 |
|---|---|
| `Path=/` | Cookie 对全站路径生效 |
| `Max-Age=2592000` | 会话有效期为 30 天 |
| `HttpOnly` | 禁止前端 JavaScript 读取 Cookie |
| `SameSite=Lax` | 限制跨站请求携带 Cookie，降低 CSRF 风险 |
| `Secure` | 生产环境只通过 HTTPS 发送 Cookie |

前端请求统一使用相对路径 `/api/v1/...`。当前公共请求封装已设置 `credentials: 'include'`，业务模块无需单独处理 Cookie。
