# 文章详情

## 接口

```
GET /api/v1/posts/{slug}
```

## 请求参数

| 参数 | 位置 | 类型 | 必填 | 默认值 | 约束 | 示例 |
|---|---|---|---|---|---|---|
| `slug` | Path | `string` | 是 | — | URL 编码后的文章标识 | `doc-wiki-guide` |

## 成功响应

HTTP `200 OK`

```json
{
  "code": 0,
  "data": {
    "id": 5,
    "slug": "doc-wiki-guide",
    "title": "接口文档指南",
    "published_at": "2026-08-09T07:23:54.974423Z",
    "content_html": "<h1 id=\"heading\">接口文档指南</h1>\n<p>本文用于接口文档联调。</p>\n"
  }
}
```

## 失败响应

文章不存在或已被删除（含软删除）：

HTTP `404 Not Found`

```json
{
  "code": 404,
  "message": "post not found"
}
```
