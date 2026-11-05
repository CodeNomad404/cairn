# 公共 MIS 账号绑定

`POST /api/v1/tool/mis-account/bind`，需要登录 Cairn。

```json
{"student_id":"24280000","password":"<mis-password>"}
```

请求只允许上述字段，学号最长 64 字节、密码最长 256 字节。用户身份从认证上下文取得，拒绝 `user_id` 等未知字段。

验证 SSO 登录成功后才保存；失败不覆盖旧账号。复用 `course_registration_credentials`，更新时保留图鉴配置。不会记录抢课次数或恢复抢课任务。

成功响应：

```json
{"code":0,"data":{"bound":true,"student_id":"24****00"}}
```

重新绑定递增 `binding_version`，作业快照与课程平台会话必须匹配当前版本。旧账号快照立即不可见，旧同步写入被拒绝。没有新增独立的 MIS 模块或绑定表。

错误：参数无效 400；未登录 Cairn 或 MIS 验证失败 401；上游不可用或保存失败 500。响应不包含密码、Cookie 或课程平台令牌。
