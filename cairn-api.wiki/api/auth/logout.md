# 登出

## 接口

```
POST /api/v1/auth/logout
```

## 请求参数

无

## 成功响应

HTTP `204 No Content`

无响应体。服务端通过 `Set-Cookie` 清除名为 `cairn_session` 的 Session Cookie（值为空，`Max-Age=0`），前端无需处理响应体，跳转到登录页即可。

未携带会话 Cookie 时调用同样返回 204，接口幂等。

## 失败响应

无预期失败。会话已失效或未登录时登出同样返回 204。
