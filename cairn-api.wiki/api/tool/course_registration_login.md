# 教务系统登录

## 接口

```
POST /api/v1/tools/course-registration/login
```

需要先登录 Cairn。首次调用时，后端使用用户提交的教务系统凭据和图鉴凭据完成一次 CAS 登录；登录成功后保存凭据，并在内存中保存该 Cairn 用户对应的 MIS Cookie 会话。后续调用可以提交空对象，使用已保存的凭据重新登录。

## 请求参数

| 参数 | 位置 | 类型 | 必填 | 默认值 | 约束 | 示例 |
|---|---|---|---|---|---|---|
| `student_id` | Body | `string` | 否 | 已保存值 | 尚未保存凭据时必填；北交大统一身份认证账号 | `24280000` |
| `password` | Body | `string` | 否 | 已保存值 | 尚未保存凭据时必填；北交大统一身份认证密码 | `<bjtu-password>` |
| `tujian_username` | Body | `string` | 否 | 已保存值 | 尚未保存凭据时必填；图鉴账号 | `example-user` |
| `tujian_password` | Body | `string` | 否 | 已保存值 | 尚未保存凭据时必填；图鉴密码 | `<tujian-password>` |

## 请求示例

```json
{
  "student_id": "24280000",
  "password": "<bjtu-password>",
  "tujian_username": "example-user",
  "tujian_password": "<tujian-password>"
}
```

四项参数必须同时提交或同时省略。首次登录成功后，当前开发版本会将凭据明文保存到数据库；该临时方案必须在功能稳定后迁移为加密存储。CAS 登录失败时不会保存或覆盖凭据。

已有保存凭据时的请求示例：

```json
{}
```

后端只在内存中保留登录成功后的 Cookie 会话；服务重启后可以使用空对象重新登录。

## 成功响应

HTTP `200 OK`

```json
{
  "code": 0,
  "data": {
    "logged_in": true
  }
}
```

## 失败响应

Cairn 尚未登录：

HTTP `401 Unauthorized`

```json
{
  "code": 401,
  "message": "login required"
}
```

北交大账号、密码、验证码被 CAS 拒绝：

HTTP `401 Unauthorized`

```json
{
  "code": 401,
  "message": "BJTU login failed"
}
```

图鉴账号不可用或验证码识别失败：

HTTP `422 Unprocessable Entity`

```json
{
  "code": 422,
  "message": "captcha recognition failed"
}
```

提交空对象但当前 Cairn 用户尚未保存凭据：

HTTP `404 Not Found`

```json
{
  "code": 404,
  "message": "saved course registration credentials not found"
}
```
