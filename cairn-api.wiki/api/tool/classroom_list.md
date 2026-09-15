# 教室占用列表

## 接口

```
GET /api/v1/tool/classroom/list
```

需要登录 Cairn。接口优先读取每日同步到本地数据库的快照；如果上海时区当天还没有快照，会使用当前登录用户已保存的 BJTU 凭据同步当天数据，然后返回查询结果。历史或未来日期缺失时不会触发同步，数据库只保留当天快照。

## 请求参数

| 参数 | 位置 | 类型 | 必填 | 默认值 | 约束 |
|---|---|---|---|---|---|
| `date` | Query | `string` | 否 | 上海时区当天 | `YYYY-MM-DD` |
| `period` | Query | `integer` | 否 | 全部节次 | `1` 到 `7` |
| `building_id` | Query | `string` | 否 | 全部教学楼 | 最多 16 字符 |
| `min_capacity` | Query | `integer` | 否 | `0` | `0` 到 `10000` |

## 成功响应

HTTP `200 OK`

```json
{
  "code": 0,
  "data": {
    "date": "2026-09-15",
        "period": 3,
    "synced_at": "2026-09-15T01:00:12+08:00",
    "rooms": [
      {
		"period": 3,
        "building_id": "1",
        "building_name": "思源楼",
        "room_name": "SY103",
        "capacity": 32,
        "occupation_type": "free"
      }
    ]
  }
}
```

`occupation_type` 的取值为 `free`（空闲）、`scheduled`（排课）、`rescheduled`（调课）、`exam`（考试）、`laboratory`（实验）和 `other`（其他安排）。不传 `period` 时，每间教室会按七个节次分别返回七条记录。

非当天日期尚未同步时，`rooms` 为空且不返回 `synced_at`。客户端应把这种情况显示为“暂无快照”，不能解释为“没有空闲教室”。当天首次同步失败时，接口会返回对应的 BJTU 登录、验证码或上游服务错误。

## 同步配置

服务每天按 `Asia/Shanghai` 时区在 `01:00` 同步。设置环境变量 `CAIRN_CLASSROOM_SYNC_USER_ID` 为一个已经通过教务登录接口保存凭据的 Cairn 用户 ID；值为 `0` 时关闭同步。服务在当天 01:00 后启动且当天还没有快照时会自动补跑。
