# 登录

## 接口

```
POST /api/v1/auth/login
```

## 请求参数

| 参数 | 位置 | 类型 | 必填 | 默认值 | 约束 | 示例 |
|---|---|---|---|---|---|---|
| `username` | Body | `string` | 是 | — | 非空 | `alice` |
| `password` | Body | `string` | 是 | — | 非空 | `secret123` |

## 请求示例

```json
{
  "username": "alice",
  "password": "secret123"
}
```

## 成功响应

HTTP `200 OK`

```json
{
  "code": 0,
  "data": {
    "id": 9,
    "username": "alice",
    "role": "user",
    "created_at": "2026-08-09T06:23:57.835025Z"
  }
}
```

登录成功后服务端创建 Session Cookie，浏览器在后续请求中自动携带，详见[鉴权方式](../schema.md#鉴权方式)。

## 失败响应

用户名不存在与密码错误返回相同响应，避免泄露账号是否存在。

HTTP `401 Unauthorized`

```json
{
  "code": 401,
  "message": "用户名或密码错误"
}
```
