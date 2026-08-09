# 文章列表

## 接口

```
GET /api/v1/posts
```

## 请求参数

| 参数 | 位置 | 类型 | 必填 | 默认值 | 约束 | 示例 |
|---|---|---|---|---|---|---|
| `page` | Query | `int` | 否 | `1` | ≥ 1 | `2` |
| `page_size` | Query | `int` | 否 | `20` | 1–100 | `10` |

## 成功响应

HTTP `200 OK`

```json
{
  "code": 0,
  "data": {
    "items": [
      {
        "id": 4,
        "slug": "blog-project-diff-2026-05-31",
        "title": "开发日志 2026-05-31",
        "published_at": "2026-06-08T15:11:16.23812Z"
      },
      {
        "id": 3,
        "slug": "blog-project-diff-2026-05-27",
        "title": "开发日志 2026-05-27",
        "published_at": "2026-06-08T15:10:51.5658Z"
      },
      {
        "id": 1,
        "slug": "day-one-server-ready",
        "title": "Cairn 的第一天：把服务器搞定了",
        "published_at": "2026-06-06T14:23:33.893575Z"
      }
    ],
    "page": 1,
    "page_size": 20,
    "has_more": false
  }
}
```

按发布时间倒序返回已发布的文章（软删除的文章不出现）。`has_more` 为 `true` 时可用 `page + 1` 继续翻页。

## 失败响应

无预期失败。
