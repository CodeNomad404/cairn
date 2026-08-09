# 删除文章

## 接口

```
DELETE /api/v1/posts/{slug}
```

需要 `admin` 权限。

## 请求参数

| 参数 | 位置 | 类型 | 必填 | 默认值 | 约束 | 示例 |
|---|---|---|---|---|---|---|
| `slug` | Path | `string` | 是 | — | URL 编码后的文章标识 | `doc-wiki-guide` |

## 成功响应

HTTP `204 No Content`

无响应体。

删除为软删除：文章状态标记为 `deleted`，详情与列表不再返回，文章下的评论记录保留，`slug` 不会被释放。

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

文章不存在或已被删除：

HTTP `404 Not Found`

```json
{
  "code": 404,
  "message": "post not found"
}
```
