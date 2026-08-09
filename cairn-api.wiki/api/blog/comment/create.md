# 发表评论

## 接口

```
POST /api/v1/posts/{slug}/comments
```

需要登录。

## 请求参数

| 参数 | 位置 | 类型 | 必填 | 默认值 | 约束 | 示例 |
|---|---|---|---|---|---|---|
| `slug` | Path | `string` | 是 | — | URL 编码后的文章标识 | `day-one-server-ready` |
| `content` | Body | `string` | 是 | — | 去除首尾空格后非空，最长 4000 字符 | `写得很棒！` |
| `parent_id` | Body | `int64` / `null` | 否 | — | 父评论 id；仅支持一级回复，回复的回复会被拒绝 | `2` |

## 请求示例

```json
{
  "content": "写得很棒！",
  "parent_id": null
}
```

## 成功响应

HTTP `201 Created`

```json
{
  "code": 1,
  "data": {
    "id": 2,
    "post_id": 1,
    "parent_id": null,
    "user_id": 4,
    "content": "接口文档联调评论。",
    "status": "approved",
    "created_at": "2026-08-09T07:28:31.54834Z",
    "username": "admin"
  }
}
```

回复时 `parent_id` 指向父评论，`username` 为评论作者（当前登录用户）。

## 失败响应

未登录：

HTTP `401 Unauthorized`

```json
{
  "code": 401,
  "message": "login required"
}
```

文章不存在或已被删除：

HTTP `404 Not Found`

```json
{
  "code": 404,
  "message": "post not found"
}
```

`parent_id` 指向的评论不存在：

HTTP `422 Unprocessable Entity`

```json
{
  "code": 422,
  "message": "parent comment not found"
}
```

回复已有回复的评论（仅支持一级回复）：

HTTP `422 Unprocessable Entity`

```json
{
  "code": 422,
  "message": "only one level of replies is supported"
}
```
