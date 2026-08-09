# 创建文章

## 接口

```
POST /api/v1/posts
```

需要 `admin` 权限。

## 请求参数

| 参数 | 位置 | 类型 | 必填 | 默认值 | 约束 | 示例 |
|---|---|---|---|---|---|---|
| `slug` | Body | `string` | 是 | — | 去除首尾空格后非空；服务端统一转为小写；全局唯一 | `doc-wiki-guide` |
| `title` | Body | `string` | 是 | — | 去除首尾空格后非空 | `接口文档指南` |
| `content_md` | Body | `string` | 是 | — | Markdown 原文，非空 | `# 接口文档指南` |

## 请求示例

```json
{
  "slug": "doc-wiki-guide",
  "title": "接口文档指南",
  "content_md": "# 接口文档指南\n\n本文用于接口文档联调。"
}
```

## 成功响应

HTTP `201 Created`

```json
{
  "code": 1,
  "data": {
    "id": 5,
    "slug": "doc-wiki-guide",
    "title": "接口文档指南",
    "published_at": "2026-08-09T07:23:54.974423844Z",
    "content_html": "<h1 id=\"heading\">接口文档指南</h1>\n<p>本文用于接口文档联调。</p>\n"
  }
}
```

`content_html` 由服务端将 `content_md` 渲染为 HTML（GFM + 代码高亮）并经过安全过滤，前端直接展示。

## 失败响应

未登录：

HTTP `401 Unauthorized`

```json
{
  "code": 401,
  "message": "login required"
}
```

已登录但角色不是 `admin`：

HTTP `403 Forbidden`

```json
{
  "code": 403,
  "message": "insufficient permissions"
}
```
