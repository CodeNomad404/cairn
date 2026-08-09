# 删除评论

## 接口

```
DELETE /api/v1/comments/{id}
```

需要登录。作者本人或 `admin` 可以删除。

## 请求参数

| 参数 | 位置 | 类型 | 必填 | 默认值 | 约束 | 示例 |
|---|---|---|---|---|---|---|
| `id` | Path | `int64` | 是 | — | 大于 0 的评论 id | `2` |

## 成功响应

HTTP `204 No Content`

无响应体。

## 失败响应

未登录：

HTTP `401 Unauthorized`

```json
{
  "code": 401,
  "message": "login required"
}
```

评论不存在或已被删除：

HTTP `404 Not Found`

```json
{
  "code": 404,
  "message": "comment not found"
}
```

已登录但不是评论作者、也没有 `admin` 权限：

HTTP `403 Forbidden`

```json
{
  "code": 403,
  "message": "cannot delete this comment"
}
```
