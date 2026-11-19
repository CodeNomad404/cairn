# 注册

## 接口

```
POST /api/v1/auth/register
```

## 请求参数

| 参数 | 位置 | 类型 | 必填 | 默认值 | 约束 | 示例 |
|---|---|---|---|---|---|---|
| `username` | Body | `string` | 是 | — | 去除首尾空白后长度为 2–64 个 Unicode 字符，不含控制字符，区分大小写 | `alice` |
| `password` | Body | `string` | 是 | — | 至少 8 个 Unicode 字符，UTF-8 编码最多 72 字节 | `secret123` |

## 请求示例

```json
{
  "username": "alice",
  "password": "secret123"
}
```

## 成功响应

HTTP `201 Created`

```json
{
  "code": 1,
  "data": {
    "id": 9,
    "username": "alice",
    "role": "user",
    "created_at": "2026-08-09T06:23:57.835025Z"
  }
}
```

注册成功后自动登录并创建 Session Cookie，详见[鉴权方式](../schema.md#鉴权方式)。

## 失败响应

- HTTP `400`：注册输入不符合上面的约束。
- HTTP `429`：同一 IP 每分钟最多提交 5 次注册请求。响应头 `Retry-After` 给出等待秒数。
- HTTP `500`：注册未能完成，请稍后重试。

用户名已被注册：HTTP `422 Unprocessable Entity`

```json
{
  "code": 422,
  "message": "用户名已被注册"
}
```
