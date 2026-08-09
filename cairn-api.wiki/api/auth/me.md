# 当前用户信息

## 接口

```
GET /api/v1/auth/me
```

## 请求参数

无。登录后浏览器自动携带 `cairn_session` Cookie，详见[鉴权方式](../schema.md#鉴权方式)。

## 成功响应

HTTP `200 OK`

```json
{
  "code": 0,
  "data": {
    "id": 9,
    "username": "doc_guide_01",
    "role": "user",
    "created_at": "2026-08-09T06:23:57.835025Z"
  }
}
```

`role` 取值为 `user` 或 `admin`，前端据此控制管理功能的显示。

## 失败响应

未登录、会话已过期或 Cookie 无效：

HTTP `401 Unauthorized`

```json
{
  "code": 401,
  "message": "login required"
}
```

前端收到 401 后应清除本地登录状态并跳转登录页。
