# 评论列表

## 接口

```
GET /api/v1/posts/{slug}/comments
```

## 请求参数

| 参数 | 位置 | 类型 | 必填 | 默认值 | 约束 | 示例 |
|---|---|---|---|---|---|---|
| `slug` | Path | `string` | 是 | — | URL 编码后的文章标识 | `day-one-server-ready` |

## 成功响应

HTTP `200 OK`

```json
{
  "code": 0,
  "data": {
    "items": [
      {
        "id": 1,
        "post_id": 5,
        "parent_id": null,
        "user_id": 4,
        "content": "第一条评论，用于接口文档。",
        "status": "approved",
        "created_at": "2026-08-09T07:23:55.008378Z",
        "username": "admin"
      }
    ]
  }
}
```

没有评论时 `items` 为空数组：

```json
{
  "code": 0,
  "data": {
    "items": []
  }
}
```

## 失败响应

文章不存在或已被删除：

HTTP `404 Not Found`

```json
{
  "code": 404,
  "message": "post not found"
}
```
