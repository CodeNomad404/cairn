# 注册

## 接口

```
POST /api/v1/auth/register
```

## 请求参数

| 参数 | 位置 | 类型 | 必填 | 默认值 | 约束 | 示例 |
|---|---|---|---|---|---|---|
| `username` | Body | `string` | 是 | — | 去除首尾空格后长度为 2–64 个字符 | `alice` |
| `password` | Body | `string` | 是 | — | 长度至少为 6 个字符 | `secret123` |

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

用户名已被注册：HTTP `422 Unprocessable Entity`

```json
{
  "code": 422,
  "message": "用户名已被注册"
}
```
