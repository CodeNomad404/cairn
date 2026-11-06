# 教务系统登录

## 接口

```
POST /api/v1/tools/course-registration/login
```

需要先登录 Cairn。首次调用时，后端进入北交大 SSO OAuth 登录页，使用页面动态下发的 SM2 公钥加密统一身份认证密码，再经 MIS 回调建立 Cookie 会话。登录成功后保存凭据，并在内存中保存该 Cairn 用户对应的 MIS Cookie 会话。后续调用可以提交空对象，使用已保存的凭据重新登录。

## 请求参数

| 参数 | 位置 | 类型 | 必填 | 默认值 | 约束 | 示例 |
|---|---|---|---|---|---|---|
| `student_id` | Body | `string` | 否 | 已保存值 | 尚未保存凭据时必填；北交大统一身份认证账号 | `24280000` |
| `password` | Body | `string` | 否 | 已保存值 | 尚未保存凭据时必填；北交大统一身份认证密码 | `<bjtu-password>` |
| `tujian_username` | Body | `string` | 否 | 已保存值 | 尚未保存凭据时必填；选课验证码使用的图鉴账号 | `example-user` |
| `tujian_password` | Body | `string` | 否 | 已保存值 | 尚未保存凭据时必填；选课验证码使用的图鉴密码 | `<tujian-password>` |

## 请求示例

```json
{
  "student_id": "24280000",
  "password": "<bjtu-password>",
  "tujian_username": "example-user",
  "tujian_password": "<tujian-password>"
}
```

四项参数可同时提交，也可提交空对象使用已保存凭据。已通过[公共 MIS 绑定接口](mis_account_bind.md)绑定的用户，还可只提交 `tujian_username` 和 `tujian_password` 更新图鉴配置并登录。图鉴凭据不参与 SSO 登录，仅供后续提交选课验证码使用。首次登录成功后，当前开发版本会将凭据明文保存到数据库；该临时方案必须在功能稳定后迁移为加密存储。SSO 登录失败时不会保存或覆盖凭据。

绑定 MIS 本身不记录抢课登录次数、不启动或恢复抢课任务；本旧接口仍保留原有的抢课任务恢复行为。

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

北交大账号或密码被 SSO 拒绝，或 SSO 要求当前实现尚未支持的额外验证：

HTTP `401 Unauthorized`

```json
{
  "code": 401,
  "message": "BJTU login failed"
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
